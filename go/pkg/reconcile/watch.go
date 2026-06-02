package reconcile

import (
	"context"
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	snapshotclient "github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned"
	snapshotinformers "github.com/kubernetes-csi/external-snapshotter/client/v6/informers/externalversions"
)

// WatchMetrics records watch-driven reconcile signals.
type WatchMetrics interface {
	RecordWatchEvent(resource string)
}

// WatchRunner wires Kubernetes informers to a debouncer.
type WatchRunner struct {
	clientset      kubernetes.Interface
	snapshotClient snapshotclient.Interface
	namespace      string
	debouncer      *Debouncer
	metrics        WatchMetrics
	onWatchError   func(error)

	stopCh   chan struct{}
	stopOnce sync.Once
}

// WatchConfig holds informer watch settings.
type WatchConfig struct {
	Clientset      kubernetes.Interface
	SnapshotClient snapshotclient.Interface
	Namespace      string
	Debouncer      *Debouncer
	Metrics        WatchMetrics
	OnWatchError   func(error)
}

// NewWatchRunner creates a watch runner for PV/PVC/VolumeSnapshot resources.
func NewWatchRunner(cfg WatchConfig) *WatchRunner {
	return &WatchRunner{
		clientset:      cfg.Clientset,
		snapshotClient: cfg.SnapshotClient,
		namespace:      cfg.Namespace,
		debouncer:      cfg.Debouncer,
		metrics:        cfg.Metrics,
		onWatchError:   cfg.OnWatchError,
		stopCh:         make(chan struct{}),
	}
}

// Run starts informers until ctx is done or Stop is called.
func (w *WatchRunner) Run(ctx context.Context) error {
	if w.clientset == nil || w.snapshotClient == nil {
		return fmt.Errorf("kubernetes clients are required for watch mode")
	}

	defer w.Stop()

	clusterFactory := informers.NewSharedInformerFactory(w.clientset, 0)
	pvInformer := clusterFactory.Core().V1().PersistentVolumes().Informer()

	var (
		pvcInformer      cache.SharedIndexInformer
		vsInformer       cache.SharedIndexInformer
		namespaceFactory informers.SharedInformerFactory
		snapshotFactory  snapshotinformers.SharedInformerFactory
	)

	if w.namespace != "" {
		namespaceFactory = informers.NewSharedInformerFactoryWithOptions(
			w.clientset,
			0,
			informers.WithNamespace(w.namespace),
		)
		pvcInformer = namespaceFactory.Core().V1().PersistentVolumeClaims().Informer()

		snapshotFactory = snapshotinformers.NewSharedInformerFactoryWithOptions(
			w.snapshotClient,
			0,
			snapshotinformers.WithNamespace(w.namespace),
		)
		vsInformer = snapshotFactory.Snapshot().V1().VolumeSnapshots().Informer()
	} else {
		pvcInformer = clusterFactory.Core().V1().PersistentVolumeClaims().Informer()
		snapshotFactory = snapshotinformers.NewSharedInformerFactory(w.snapshotClient, 0)
		vsInformer = snapshotFactory.Snapshot().V1().VolumeSnapshots().Informer()
	}

	pvHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    func(_ any) { w.trigger("pv") },
		UpdateFunc: func(_, _ any) { w.trigger("pv") },
		DeleteFunc: func(_ any) { w.trigger("pv") },
	}
	pvcHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { w.onPVCEvent(obj) },
		UpdateFunc: func(_, obj any) { w.onPVCEvent(obj) },
		DeleteFunc: func(obj any) { w.onPVCEvent(obj) },
	}
	vsHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { w.onSnapshotEvent(obj) },
		UpdateFunc: func(_, obj any) { w.onSnapshotEvent(obj) },
		DeleteFunc: func(obj any) { w.onSnapshotEvent(obj) },
	}

	if _, err := pvInformer.AddEventHandler(pvHandler); err != nil {
		return err
	}
	if _, err := pvcInformer.AddEventHandler(pvcHandler); err != nil {
		return err
	}
	if _, err := vsInformer.AddEventHandler(vsHandler); err != nil {
		return err
	}

	clusterFactory.Start(w.stopCh)
	if namespaceFactory != nil {
		namespaceFactory.Start(w.stopCh)
	}
	snapshotFactory.Start(w.stopCh)

	if !cache.WaitForCacheSync(ctx.Done(), pvInformer.HasSynced, pvcInformer.HasSynced, vsInformer.HasSynced) {
		err := ctx.Err()
		if err == nil {
			err = fmt.Errorf("failed to sync informer caches")
		}
		if w.onWatchError != nil {
			w.onWatchError(err)
		}
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

// Stop shuts down informers.
func (w *WatchRunner) Stop() {
	w.stopOnce.Do(func() {
		close(w.stopCh)
	})
}

func (w *WatchRunner) trigger(resource string) {
	if w.metrics != nil {
		w.metrics.RecordWatchEvent(resource)
	}
	if w.debouncer != nil {
		w.debouncer.Trigger()
	}
}

func (w *WatchRunner) onPVCEvent(obj any) {
	pvc, ok := toPVC(obj)
	if !ok {
		return
	}
	if w.namespace != "" && pvc.Namespace != w.namespace {
		return
	}
	w.trigger("pvc")
}

func (w *WatchRunner) onSnapshotEvent(obj any) {
	vs, ok := toVolumeSnapshot(obj)
	if !ok {
		return
	}
	if w.namespace != "" && vs.Namespace != w.namespace {
		return
	}
	w.trigger("snapshot")
}

func toPVC(obj any) (*corev1.PersistentVolumeClaim, bool) {
	switch t := obj.(type) {
	case *corev1.PersistentVolumeClaim:
		return t, true
	case cache.DeletedFinalStateUnknown:
		return toPVC(t.Obj)
	default:
		return nil, false
	}
}

func toVolumeSnapshot(obj any) (*snapshotv1.VolumeSnapshot, bool) {
	switch t := obj.(type) {
	case *snapshotv1.VolumeSnapshot:
		return t, true
	case cache.DeletedFinalStateUnknown:
		return toVolumeSnapshot(t.Obj)
	default:
		return nil, false
	}
}
