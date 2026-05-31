package inventorycache

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	"github.com/stretchr/testify/require"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/k8s"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/truenas"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type countingK8sClient struct {
	pvcCalls int32
	pvCalls  int32
}

func (c *countingK8sClient) ListPersistentVolumes(context.Context) ([]corev1.PersistentVolume, error) {
	atomic.AddInt32(&c.pvCalls, 1)
	return []corev1.PersistentVolume{{ObjectMeta: metav1.ObjectMeta{Name: "pv-1"}}}, nil
}

func (c *countingK8sClient) ListPersistentVolumeClaims(context.Context, string) ([]corev1.PersistentVolumeClaim, error) {
	atomic.AddInt32(&c.pvcCalls, 1)
	return []corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: "pvc-1"}}}, nil
}

func (c *countingK8sClient) ListVolumeSnapshots(context.Context, string) ([]snapshotv1.VolumeSnapshot, error) {
	return nil, nil
}

func (c *countingK8sClient) ListStorageClasses(context.Context) ([]storagev1.StorageClass, error) {
	return nil, nil
}

func (c *countingK8sClient) ListPods(context.Context, string) ([]corev1.Pod, error) {
	return nil, nil
}

func (c *countingK8sClient) ListNamespaces(context.Context) ([]corev1.Namespace, error) {
	return nil, nil
}

func (c *countingK8sClient) GetNamespace(context.Context, string) (*corev1.Namespace, error) {
	return nil, nil
}

func (c *countingK8sClient) ListPersistentVolumesByStorageClass(context.Context, string) ([]corev1.PersistentVolume, error) {
	return nil, nil
}

func (c *countingK8sClient) ListPersistentVolumeClaimsByStorageClass(context.Context, string, string) ([]corev1.PersistentVolumeClaim, error) {
	return nil, nil
}

func (c *countingK8sClient) ListDemocraticCSIPersistentVolumes(context.Context) ([]corev1.PersistentVolume, error) {
	return nil, nil
}

func (c *countingK8sClient) ListUnboundPersistentVolumeClaims(context.Context, string) ([]corev1.PersistentVolumeClaim, error) {
	return nil, nil
}

func (c *countingK8sClient) TestConnection(context.Context) error { return nil }

func (c *countingK8sClient) ValidateRBACPermissions(context.Context) (*k8s.RBACValidationResult, error) {
	return nil, nil
}

func (c *countingK8sClient) GetClusterInfo(context.Context) (*k8s.ClusterInfo, error) {
	return nil, nil
}

func (c *countingK8sClient) ListCSINodes(context.Context) ([]storagev1.CSINode, error) {
	return nil, nil
}

func (c *countingK8sClient) ListCSIDrivers(context.Context) ([]storagev1.CSIDriver, error) {
	return nil, nil
}

func (c *countingK8sClient) ListVolumeAttachments(context.Context) ([]storagev1.VolumeAttachment, error) {
	return nil, nil
}

func (c *countingK8sClient) GetCSIDriverPods(context.Context, string) ([]corev1.Pod, error) {
	return nil, nil
}

type countingTrueNASClient struct {
	volumeCalls   int32
	snapshotCalls int32
}

func (c *countingTrueNASClient) ListVolumes(context.Context) ([]truenas.Volume, error) {
	atomic.AddInt32(&c.volumeCalls, 1)
	return []truenas.Volume{{Name: "vol-1"}}, nil
}

func (c *countingTrueNASClient) ListSnapshots(context.Context) ([]truenas.Snapshot, error) {
	atomic.AddInt32(&c.snapshotCalls, 1)
	return []truenas.Snapshot{{Name: "snap-1"}}, nil
}

func (c *countingTrueNASClient) ListPools(context.Context) ([]truenas.Pool, error) {
	return nil, nil
}

func (c *countingTrueNASClient) GetSystemInfo(context.Context) (*truenas.SystemInfo, error) {
	return nil, nil
}

func (c *countingTrueNASClient) TestConnection(context.Context) error { return nil }

func TestWrapK8sClient_CachesListCallsWithinTTL(t *testing.T) {
	base := &countingK8sClient{}
	cache := NewCache(Config{Enabled: true, TTL: time.Minute, MaxSize: 10})
	wrapped := WrapK8sClient(base, cache)

	ctx := context.Background()
	_, err := wrapped.ListPersistentVolumeClaims(ctx, "apps")
	require.NoError(t, err)
	_, err = wrapped.ListPersistentVolumeClaims(ctx, "apps")
	require.NoError(t, err)

	require.Equal(t, int32(1), atomic.LoadInt32(&base.pvcCalls))
}

func TestWrapTrueNASClient_CachesListCallsWithinTTL(t *testing.T) {
	base := &countingTrueNASClient{}
	cache := NewCache(Config{Enabled: true, TTL: time.Minute, MaxSize: 10})
	wrapped := WrapTrueNASClient(base, cache)

	ctx := context.Background()
	_, err := wrapped.ListVolumes(ctx)
	require.NoError(t, err)
	_, err = wrapped.ListVolumes(ctx)
	require.NoError(t, err)

	require.Equal(t, int32(1), atomic.LoadInt32(&base.volumeCalls))
}

func TestWrapK8sClient_DisabledReturnsBase(t *testing.T) {
	base := &countingK8sClient{}
	cache := NewCache(Config{Enabled: false})
	wrapped := WrapK8sClient(base, cache)
	require.Equal(t, base, wrapped)
}
