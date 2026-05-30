package orphan

import (
	"strings"

	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/truenas"
)

func truenasSnapshotFullName(s truenas.Snapshot) string {
	if strings.Contains(s.Name, "@") {
		return s.Name
	}
	if s.Dataset != "" && s.Name != "" {
		return s.Dataset + "@" + s.Name
	}
	return s.Name
}

func truenasSnapshotComponentName(s truenas.Snapshot) string {
	full := truenasSnapshotFullName(s)
	if idx := strings.LastIndex(full, "@"); idx >= 0 {
		return full[idx+1:]
	}
	return s.Name
}

func snapshotNameMatches(k8sName string, tn truenas.Snapshot) bool {
	full := truenasSnapshotFullName(tn)
	component := truenasSnapshotComponentName(tn)
	return tn.Name == k8sName || full == k8sName || strings.HasSuffix(full, "@"+k8sName) || component == k8sName
}

func k8sSnapshotDatasetHints(k8s snapshotv1.VolumeSnapshot) []string {
	var hints []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" {
			hints = append(hints, s)
		}
	}

	for _, m := range []map[string]string{k8s.Labels, k8s.Annotations} {
		for _, v := range m {
			if strings.Contains(v, "/") {
				add(v)
			}
		}
	}

	if k8s.Spec.Source.PersistentVolumeClaimName != nil {
		pvc := *k8s.Spec.Source.PersistentVolumeClaimName
		if k8s.Namespace != "" {
			add(k8s.Namespace + "/" + pvc)
		}
	}
	if k8s.Spec.Source.VolumeSnapshotContentName != nil {
		add(*k8s.Spec.Source.VolumeSnapshotContentName)
	}
	if k8s.Status != nil && k8s.Status.BoundVolumeSnapshotContentName != nil {
		add(*k8s.Status.BoundVolumeSnapshotContentName)
	}

	return hints
}

func truenasDatasetMatchesHints(dataset string, hints []string) bool {
	dataset = strings.Trim(dataset, "/")
	if dataset == "" {
		return false
	}
	for _, hint := range hints {
		hint = strings.Trim(hint, "/")
		if hint == dataset {
			return true
		}
		if strings.HasSuffix(hint, "/"+dataset) || strings.HasSuffix(dataset, "/"+hint) {
			return true
		}
	}
	return false
}

func snapshotCorrelatesPair(k8s snapshotv1.VolumeSnapshot, tn truenas.Snapshot) bool {
	if !snapshotNameMatches(k8s.Name, tn) {
		return false
	}

	hints := k8sSnapshotDatasetHints(k8s)
	if len(hints) == 0 {
		return truenasSnapshotFullName(tn) == k8s.Name
	}

	return truenasDatasetMatchesHints(tn.Dataset, hints) ||
		truenasDatasetMatchesHints(truenasSnapshotFullName(tn), hints)
}

func snapshotCorrelatesWithTrueNAS(k8s snapshotv1.VolumeSnapshot, truenasSnapshots []truenas.Snapshot) bool {
	for _, tn := range truenasSnapshots {
		if snapshotCorrelatesPair(k8s, tn) {
			return true
		}
	}
	return false
}

func truenasSnapshotCorrelatesWithK8s(tn truenas.Snapshot, k8sSnapshots []snapshotv1.VolumeSnapshot) bool {
	for _, ks := range k8sSnapshots {
		if snapshotCorrelatesPair(ks, tn) {
			return true
		}
	}
	return false
}

func extractDatasetFromVolumeHandle(volumeHandle string) string {
	handle := strings.TrimSpace(volumeHandle)
	if strings.Contains(handle, "iqn.") {
		handle = strings.TrimRight(handle, ":")
		if idx := strings.LastIndex(handle, ":"); idx >= 0 && idx+1 < len(handle) {
			handle = handle[idx+1:]
		} else {
			return ""
		}
	} else {
		handle = strings.TrimRight(handle, "/")
		if idx := strings.LastIndex(handle, "/"); idx >= 0 && idx+1 < len(handle) {
			handle = handle[idx+1:]
		}
	}

	if idx := strings.LastIndex(handle, "@"); idx >= 0 {
		handle = handle[:idx]
	}
	return strings.TrimSpace(handle)
}

func volumeMatches(volume truenas.Volume, volumeHandle, datasetName string) bool {
	if datasetName == "" {
		return false
	}
	if volume.Name == datasetName || volume.Name == volumeHandle {
		return true
	}
	if volume.ID == datasetName ||
		strings.HasSuffix(volume.ID, "/"+datasetName) ||
		strings.HasSuffix(volume.ID, ":"+datasetName) {
		return true
	}
	path := strings.TrimRight(volume.Path, "/")
	if path == datasetName || strings.HasSuffix(path, "/"+datasetName) {
		return true
	}
	if volume.Properties != nil {
		for _, value := range volume.Properties {
			if value == datasetName ||
				strings.HasSuffix(value, "/"+datasetName) ||
				strings.HasSuffix(value, ":"+datasetName) {
				return true
			}
		}
	}
	return false
}
