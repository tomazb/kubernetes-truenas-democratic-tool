package inventorycache

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/truenas"
)

// cloneSlice returns a shallow copy of a slice so callers cannot mutate cached data.
func cloneSlice[T any](items []T) []T {
	if len(items) == 0 {
		return nil
	}
	return append([]T(nil), items...)
}

func cloneStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func cloneObjectMeta(meta metav1.ObjectMeta) metav1.ObjectMeta {
	meta.Labels = cloneStringMap(meta.Labels)
	meta.Annotations = cloneStringMap(meta.Annotations)
	return meta
}

func clonePersistentVolumes(in []corev1.PersistentVolume) []corev1.PersistentVolume {
	out := cloneSlice(in)
	for i := range out {
		out[i].ObjectMeta = cloneObjectMeta(out[i].ObjectMeta)
	}
	return out
}

func clonePersistentVolumeClaims(in []corev1.PersistentVolumeClaim) []corev1.PersistentVolumeClaim {
	out := cloneSlice(in)
	for i := range out {
		out[i].ObjectMeta = cloneObjectMeta(out[i].ObjectMeta)
	}
	return out
}

func cloneVolumeSnapshots(in []snapshotv1.VolumeSnapshot) []snapshotv1.VolumeSnapshot {
	out := cloneSlice(in)
	for i := range out {
		out[i].ObjectMeta = cloneObjectMeta(out[i].ObjectMeta)
	}
	return out
}

func cloneTrueNASVolumes(in []truenas.Volume) []truenas.Volume {
	out := cloneSlice(in)
	for i := range out {
		out[i].Properties = cloneStringMap(out[i].Properties)
	}
	return out
}

func cloneTrueNASSnapshots(in []truenas.Snapshot) []truenas.Snapshot {
	out := cloneSlice(in)
	for i := range out {
		out[i].Properties = cloneStringMap(out[i].Properties)
	}
	return out
}
