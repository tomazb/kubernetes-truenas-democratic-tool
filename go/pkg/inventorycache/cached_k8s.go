package inventorycache

import (
	"context"

	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
)

const (
	opK8sAllPVs  = "k8s_all_pvs"
	opK8sPVs     = "k8s_pvs"
	opK8sPVCs    = "k8s_pvcs"
	opK8sSnaps   = "k8s_snapshots"
)

type cachedK8sClient struct {
	base  k8s.Client
	cache *Cache
}

// WrapK8sClient returns a caching decorator when enabled, otherwise the base client.
func WrapK8sClient(base k8s.Client, cache *Cache) k8s.Client {
	if cache == nil || !cache.Enabled() {
		return base
	}
	return &cachedK8sClient{base: base, cache: cache}
}

func (c *cachedK8sClient) ListPersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error) {
	res, err := GetOrLoad(c.cache, opK8sAllPVs, opK8sAllPVs, func() ([]corev1.PersistentVolume, error) {
		return c.base.ListPersistentVolumes(ctx)
	})
	if err != nil {
		return nil, err
	}
	return clonePersistentVolumes(res), nil
}

func (c *cachedK8sClient) ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]corev1.PersistentVolumeClaim, error) {
	key := NamespaceKey(opK8sPVCs, namespace)
	res, err := GetOrLoad(c.cache, opK8sPVCs, key, func() ([]corev1.PersistentVolumeClaim, error) {
		return c.base.ListPersistentVolumeClaims(ctx, namespace)
	})
	if err != nil {
		return nil, err
	}
	return clonePersistentVolumeClaims(res), nil
}

func (c *cachedK8sClient) ListVolumeSnapshots(ctx context.Context, namespace string) ([]snapshotv1.VolumeSnapshot, error) {
	key := NamespaceKey(opK8sSnaps, namespace)
	res, err := GetOrLoad(c.cache, opK8sSnaps, key, func() ([]snapshotv1.VolumeSnapshot, error) {
		return c.base.ListVolumeSnapshots(ctx, namespace)
	})
	if err != nil {
		return nil, err
	}
	return cloneVolumeSnapshots(res), nil
}

func (c *cachedK8sClient) ListDemocraticCSIPersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error) {
	res, err := GetOrLoad(c.cache, opK8sPVs, opK8sPVs, func() ([]corev1.PersistentVolume, error) {
		return c.base.ListDemocraticCSIPersistentVolumes(ctx)
	})
	if err != nil {
		return nil, err
	}
	return clonePersistentVolumes(res), nil
}

func (c *cachedK8sClient) ListUnboundPersistentVolumeClaims(ctx context.Context, namespace string) ([]corev1.PersistentVolumeClaim, error) {
	pvcs, err := c.ListPersistentVolumeClaims(ctx, namespace)
	if err != nil {
		return nil, err
	}

	var unbound []corev1.PersistentVolumeClaim
	for _, pvc := range pvcs {
		if pvc.Status.Phase == corev1.ClaimPending {
			unbound = append(unbound, pvc)
		}
	}
	return unbound, nil
}

func (c *cachedK8sClient) ListStorageClasses(ctx context.Context) ([]storagev1.StorageClass, error) {
	return c.base.ListStorageClasses(ctx)
}

func (c *cachedK8sClient) ListPods(ctx context.Context, namespace string) ([]corev1.Pod, error) {
	return c.base.ListPods(ctx, namespace)
}

func (c *cachedK8sClient) ListNamespaces(ctx context.Context) ([]corev1.Namespace, error) {
	return c.base.ListNamespaces(ctx)
}

func (c *cachedK8sClient) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	return c.base.GetNamespace(ctx, name)
}

func (c *cachedK8sClient) ListPersistentVolumesByStorageClass(ctx context.Context, storageClass string) ([]corev1.PersistentVolume, error) {
	return c.base.ListPersistentVolumesByStorageClass(ctx, storageClass)
}

func (c *cachedK8sClient) ListPersistentVolumeClaimsByStorageClass(ctx context.Context, namespace, storageClass string) ([]corev1.PersistentVolumeClaim, error) {
	return c.base.ListPersistentVolumeClaimsByStorageClass(ctx, namespace, storageClass)
}

func (c *cachedK8sClient) TestConnection(ctx context.Context) error {
	return c.base.TestConnection(ctx)
}

func (c *cachedK8sClient) ValidateRBACPermissions(ctx context.Context) (*k8s.RBACValidationResult, error) {
	return c.base.ValidateRBACPermissions(ctx)
}

func (c *cachedK8sClient) GetClusterInfo(ctx context.Context) (*k8s.ClusterInfo, error) {
	return c.base.GetClusterInfo(ctx)
}

func (c *cachedK8sClient) ListCSINodes(ctx context.Context) ([]storagev1.CSINode, error) {
	return c.base.ListCSINodes(ctx)
}

func (c *cachedK8sClient) ListCSIDrivers(ctx context.Context) ([]storagev1.CSIDriver, error) {
	return c.base.ListCSIDrivers(ctx)
}

func (c *cachedK8sClient) ListVolumeAttachments(ctx context.Context) ([]storagev1.VolumeAttachment, error) {
	return c.base.ListVolumeAttachments(ctx)
}

func (c *cachedK8sClient) GetCSIDriverPods(ctx context.Context, namespace string) ([]corev1.Pod, error) {
	return c.base.GetCSIDriverPods(ctx, namespace)
}
