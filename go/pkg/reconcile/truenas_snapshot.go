package reconcile

import (
	"context"
	"sync"
	"time"

	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/logging"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/truenas"
	"go.uber.org/zap"
)

// TruenasSnapshot holds the latest TrueNAS inventory from background polling.
type TruenasSnapshot struct {
	Volumes   []truenas.Volume
	Snapshots []truenas.Snapshot
	UpdatedAt time.Time
}

// TruenasPoller periodically refreshes TrueNAS inventory into a thread-safe snapshot.
type TruenasPoller struct {
	client   truenas.Client
	interval time.Duration
	now      func() time.Time
	logger   *logging.Logger

	mu   sync.RWMutex
	snap TruenasSnapshot
}

// NewTruenasPoller creates a poller for the given client and interval.
func NewTruenasPoller(client truenas.Client, interval time.Duration) *TruenasPoller {
	return &TruenasPoller{
		client:   client,
		interval: interval,
		now:      time.Now,
	}
}

// SetLogger attaches a logger for background poll failures.
func (p *TruenasPoller) SetLogger(logger *logging.Logger) {
	p.logger = logger
}

// Run polls TrueNAS until ctx is cancelled.
func (p *TruenasPoller) Run(ctx context.Context) {
	p.pollOnce(ctx)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.pollOnce(ctx)
		}
	}
}

func (p *TruenasPoller) pollOnce(ctx context.Context) {
	volumes, err := p.client.ListVolumes(ctx)
	if err != nil {
		p.logPollError("list volumes", err)
		return
	}
	snapshots, err := p.client.ListSnapshots(ctx)
	if err != nil {
		p.logPollError("list snapshots", err)
		return
	}

	p.mu.Lock()
	p.snap = TruenasSnapshot{
		Volumes:   append([]truenas.Volume(nil), volumes...),
		Snapshots: append([]truenas.Snapshot(nil), snapshots...),
		UpdatedAt: p.now(),
	}
	p.mu.Unlock()
}

// Snapshot returns a copy of the latest polled inventory.
func (p *TruenasPoller) Snapshot() TruenasSnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return TruenasSnapshot{
		Volumes:   append([]truenas.Volume(nil), p.snap.Volumes...),
		Snapshots: append([]truenas.Snapshot(nil), p.snap.Snapshots...),
		UpdatedAt: p.snap.UpdatedAt,
	}
}

// SnapshotAge returns time since the last successful poll, or zero if never polled.
func (p *TruenasPoller) SnapshotAge(now time.Time) time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.snap.UpdatedAt.IsZero() {
		return 0
	}
	return now.Sub(p.snap.UpdatedAt)
}

// SnapshotTruenasClient serves ListVolumes/ListSnapshots from a poller snapshot.
type SnapshotTruenasClient struct {
	inner  truenas.Client
	source *TruenasPoller
}

// WrapTruenasWithSnapshot returns a client that reads TN lists from the poller snapshot.
func WrapTruenasWithSnapshot(inner truenas.Client, source *TruenasPoller) truenas.Client {
	return &SnapshotTruenasClient{inner: inner, source: source}
}

func (c *SnapshotTruenasClient) ListVolumes(ctx context.Context) ([]truenas.Volume, error) {
	snap := c.source.Snapshot()
	if !snap.UpdatedAt.IsZero() {
		return append([]truenas.Volume(nil), snap.Volumes...), nil
	}
	return c.inner.ListVolumes(ctx)
}

func (c *SnapshotTruenasClient) ListSnapshots(ctx context.Context) ([]truenas.Snapshot, error) {
	snap := c.source.Snapshot()
	if !snap.UpdatedAt.IsZero() {
		return append([]truenas.Snapshot(nil), snap.Snapshots...), nil
	}
	return c.inner.ListSnapshots(ctx)
}

func (c *SnapshotTruenasClient) ListPools(ctx context.Context) ([]truenas.Pool, error) {
	return c.inner.ListPools(ctx)
}

func (c *SnapshotTruenasClient) GetSystemInfo(ctx context.Context) (*truenas.SystemInfo, error) {
	return c.inner.GetSystemInfo(ctx)
}

func (c *SnapshotTruenasClient) TestConnection(ctx context.Context) error {
	return c.inner.TestConnection(ctx)
}

func (p *TruenasPoller) logPollError(operation string, err error) {
	if p.logger == nil {
		return
	}
	p.logger.WithComponent("truenas-poller").Warn(
		"TrueNAS background poll failed",
		zap.String("operation", operation),
		zap.Error(err),
	)
}
