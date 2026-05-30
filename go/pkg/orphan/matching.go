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

func snapshotCorrelatesWithTrueNAS(k8s snapshotv1.VolumeSnapshot, truenasSnapshots []truenas.Snapshot) bool {
	k8sName := k8s.Name
	for _, tn := range truenasSnapshots {
		full := truenasSnapshotFullName(tn)
		if tn.Name == k8sName || full == k8sName || strings.HasSuffix(full, "@"+k8sName) {
			return true
		}
		if component := truenasSnapshotComponentName(tn); component == k8sName {
			return true
		}
	}
	return false
}

func truenasSnapshotCorrelatesWithK8s(tn truenas.Snapshot, k8sSnapshots []snapshotv1.VolumeSnapshot) bool {
	component := truenasSnapshotComponentName(tn)
	full := truenasSnapshotFullName(tn)
	for _, ks := range k8sSnapshots {
		if ks.Name == component || ks.Name == tn.Name || ks.Name == full {
			return true
		}
	}
	return false
}

func extractDatasetFromVolumeHandle(volumeHandle string) string {
	if strings.Contains(volumeHandle, "iqn.") {
		parts := strings.Split(volumeHandle, ":")
		if len(parts) > 1 {
			return parts[len(parts)-1]
		}
	} else if strings.Contains(volumeHandle, "/") {
		parts := strings.Split(volumeHandle, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return volumeHandle
}

func volumeMatches(volume truenas.Volume, volumeHandle, datasetName string) bool {
	if volume.Name == datasetName || volume.Name == volumeHandle {
		return true
	}
	if strings.Contains(volume.ID, datasetName) {
		return true
	}
	if strings.Contains(volume.Path, datasetName) {
		return true
	}
	return false
}
