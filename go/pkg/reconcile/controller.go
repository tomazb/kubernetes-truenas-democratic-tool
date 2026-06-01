package reconcile

import (
	"context"
	"sync"
	"time"

	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/config"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/inventorycache"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/k8s"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/logging"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/metrics"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/truenas"
	"go.uber.org/zap"
)

// ScanFunc runs a full orphan detection reconcile.
type ScanFunc func(ctx context.Context)

// Controller runs watch-mode reconciliation with debouncing and TN polling.
type Controller struct {
	k8sConfig           k8s.Config
	namespace           string
	debounce            time.Duration
	truenasPollInterval time.Duration
	scan                ScanFunc
	fullScan            ScanFunc
	inventoryCache      *inventorycache.Cache
	metrics             *metrics.Exporter
	logger              *logging.Logger
	truenasPollClient truenas.Client
	truenasPoller     *TruenasPoller
}

// ControllerConfig configures watch-mode reconciliation.
type ControllerConfig struct {
	K8sConfig           k8s.Config
	Namespace           string
	Debounce            time.Duration
	TruenasPollInterval time.Duration
	Scan                ScanFunc
	FullScan            ScanFunc
	InventoryCache      *inventorycache.Cache
	Metrics             *metrics.Exporter
	Logger              *logging.Logger
	TruenasPollClient truenas.Client
	TruenasPoller     *TruenasPoller
}

// NewController creates a watch reconcile controller.
func NewController(cfg ControllerConfig) *Controller {
	fullScan := cfg.FullScan
	if fullScan == nil {
		fullScan = cfg.Scan
	}
	return &Controller{
		k8sConfig:           cfg.K8sConfig,
		namespace:           cfg.Namespace,
		debounce:            cfg.Debounce,
		truenasPollInterval: cfg.TruenasPollInterval,
		scan:                cfg.Scan,
		fullScan:            fullScan,
		inventoryCache:      cfg.InventoryCache,
		metrics:             cfg.Metrics,
		logger:              cfg.Logger,
		truenasPollClient: cfg.TruenasPollClient,
		truenasPoller:     cfg.TruenasPoller,
	}
}

// Run executes watch-mode reconciliation until ctx is cancelled.
func (c *Controller) Run(ctx context.Context) error {
	if c.metrics != nil {
		c.metrics.SetReconcileMode(config.ReconcileModeWatch)
	}

	clientset, snapshotClient, err := k8s.NewKubernetesClients(c.k8sConfig)
	if err != nil {
		return err
	}

	tnPoller := c.truenasPoller
	if tnPoller == nil {
		tnPoller = NewTruenasPoller(c.truenasPollClient, c.truenasPollInterval)
	}
	if c.logger != nil {
		tnPoller.SetLogger(c.logger)
	}
	pollCtx, pollCancel := context.WithCancel(ctx)
	defer pollCancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		tnPoller.Run(pollCtx)
	}()

	debouncer := NewDebouncer(c.debounce, func() {
		c.onDebouncedReconcile(ctx, tnPoller)
	})

	runWatch := func() error {
		watchRunner := NewWatchRunner(WatchConfig{
			Clientset:      clientset,
			SnapshotClient: snapshotClient,
			Namespace:      c.namespace,
			Debouncer:      debouncer,
			Metrics:        c.metrics,
			OnWatchError: func(err error) {
				c.logger.WithComponent("watch-reconcile").Warn("watch failure; running full scan",
					zap.Error(err))
				c.runFullScan(ctx, "full_scan")
				debouncer.Trigger()
			},
		})
		return watchRunner.Run(ctx)
	}

	c.logger.WithComponent("watch-reconcile").Info("starting watch reconcile",
		zap.Duration("debounce", c.debounce),
		zap.Duration("truenas_poll_interval", c.truenasPollInterval),
	)

	c.runFullScan(ctx, "full_scan")

	for ctx.Err() == nil {
		err := runWatch()
		if ctx.Err() != nil {
			break
		}
		if err == nil {
			continue
		}
		c.logger.WithComponent("watch-reconcile").Warn(
			"watch runner stopped; running full scan before restart",
			zap.Error(err),
		)
		c.runFullScan(ctx, "full_scan")

		select {
		case <-ctx.Done():
		case <-time.After(5 * time.Second):
		}
	}

	debouncer.Cancel()
	pollCancel()
	wg.Wait()
	return ctx.Err()
}

func (c *Controller) onDebouncedReconcile(ctx context.Context, poller *TruenasPoller) {
	if c.inventoryCache != nil {
		c.inventoryCache.InvalidateK8sInventory()
	}
	if c.metrics != nil {
		c.metrics.IncReconcileTrigger("debounce")
		age := poller.SnapshotAge(time.Now())
		if age > 0 {
			c.metrics.SetTruenasSnapshotAge(age.Seconds())
		}
	}
	c.scan(ctx)
}

func (c *Controller) runFullScan(ctx context.Context, trigger string) {
	if c.metrics != nil {
		c.metrics.IncReconcileTrigger(trigger)
	}
	c.fullScan(ctx)
}
