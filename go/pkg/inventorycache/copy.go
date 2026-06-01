package inventorycache

import (
	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/truenas"
	corev1 "k8s.io/api/core/v1"
)

// cloneSlice returns a shallow copy of a slice so callers cannot mutate cached data.
func cloneSlice[T any](items []T) []T {
	if items == nil {
		return nil
	}
	if len(items) == 0 {
		return []T{}
	}
	return append([]T{}, items...)
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

func clonePersistentVolumes(in []corev1.PersistentVolume) []corev1.PersistentVolume {
	if in == nil {
		return nil
	}
	out := make([]corev1.PersistentVolume, len(in))
	for i := range in {
		out[i] = *in[i].DeepCopy()
	}
	return out
}

func clonePersistentVolumeClaims(in []corev1.PersistentVolumeClaim) []corev1.PersistentVolumeClaim {
	if in == nil {
		return nil
	}
	out := make([]corev1.PersistentVolumeClaim, len(in))
	for i := range in {
		out[i] = *in[i].DeepCopy()
	}
	return out
}

func cloneVolumeSnapshots(in []snapshotv1.VolumeSnapshot) []snapshotv1.VolumeSnapshot {
	if in == nil {
		return nil
	}
	out := make([]snapshotv1.VolumeSnapshot, len(in))
	for i := range in {
		out[i] = *in[i].DeepCopy()
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
