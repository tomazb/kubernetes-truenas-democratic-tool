package orphan

import (
	"testing"
	"time"

	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/truenas"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExtractDatasetFromVolumeHandle(t *testing.T) {
	tests := []struct {
		name    string
		handle  string
		want    string
	}{
		{"iscsi handle", "iqn.2005-10.org.freenas.ctl:my-volume", "my-volume"},
		{"zfs path handle", "tank/k8s/vol-1", "vol-1"},
		{"plain handle", "standalone-id", "standalone-id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDatasetFromVolumeHandle(tt.handle)
			if got != tt.want {
				t.Fatalf("extractDatasetFromVolumeHandle(%q) = %q, want %q", tt.handle, got, tt.want)
			}
		})
	}
}

func TestVolumeMatches(t *testing.T) {
	tests := []struct {
		name         string
		volume       truenas.Volume
		volumeHandle string
		datasetName  string
		want         bool
	}{
		{
			name:         "direct name match",
			volume:       truenas.Volume{Name: "vol-1"},
			volumeHandle: "tank/k8s/vol-1",
			datasetName:  "vol-1",
			want:         true,
		},
		{
			name:         "path contains dataset",
			volume:       truenas.Volume{Name: "other", Path: "/mnt/tank/k8s/vol-1"},
			volumeHandle: "tank/k8s/vol-1",
			datasetName:  "vol-1",
			want:         true,
		},
		{
			name: "unrelated property substring does not match",
			volume: truenas.Volume{
				Name:       "unrelated",
				Properties: map[string]string{"note": "prefix-vol-1-suffix"},
			},
			volumeHandle: "tank/k8s/vol-1",
			datasetName:  "vol-1",
			want:         false,
		},
		{
			name:         "no match",
			volume:       truenas.Volume{Name: "other-vol"},
			volumeHandle: "tank/k8s/vol-1",
			datasetName:  "vol-1",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := volumeMatches(tt.volume, tt.volumeHandle, tt.datasetName)
			if got != tt.want {
				t.Fatalf("volumeMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnapshotCorrelatesWithTrueNAS(t *testing.T) {
	old := time.Now().Add(-48 * time.Hour)
	k8sSnap := snapshotv1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "snap-a",
			Namespace:         "apps",
			CreationTimestamp: metav1.NewTime(old),
		},
	}

	tests := []struct {
		name     string
		truenas  []truenas.Snapshot
		want     bool
	}{
		{
			name: "matching snapshot name",
			truenas: []truenas.Snapshot{
				{Name: "snap-a", Dataset: "tank/k8s/vol-1"},
			},
			want: true,
		},
		{
			name: "matching dataset@name",
			truenas: []truenas.Snapshot{
				{Name: "snap-a", Dataset: "tank/k8s/vol-1"},
			},
			want: true,
		},
		{
			name: "full zfs snapshot id",
			truenas: []truenas.Snapshot{
				{Name: "tank/k8s/vol-1@snap-a", Dataset: "tank/k8s/vol-1"},
			},
			want: true,
		},
		{
			name: "no peer",
			truenas: []truenas.Snapshot{
				{Name: "other-snap", Dataset: "tank/k8s/vol-2"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snapshotCorrelatesWithTrueNAS(k8sSnap, tt.truenas)
			if got != tt.want {
				t.Fatalf("snapshotCorrelatesWithTrueNAS() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTruenasSnapshotCorrelatesWithK8s(t *testing.T) {
	old := metav1.NewTime(time.Now().Add(-48 * time.Hour))
	k8sSnaps := []snapshotv1.VolumeSnapshot{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "snap-a",
				Namespace:         "apps",
				CreationTimestamp: old,
			},
		},
	}

	tests := []struct {
		name    string
		truenas truenas.Snapshot
		want    bool
	}{
		{
			name:    "name match",
			truenas: truenas.Snapshot{Name: "snap-a", Dataset: "tank/k8s/vol-1"},
			want:    true,
		},
		{
			name:    "full id match",
			truenas: truenas.Snapshot{Name: "tank/k8s/vol-1@snap-a", Dataset: "tank/k8s/vol-1"},
			want:    true,
		},
		{
			name:    "orphan on truenas side",
			truenas: truenas.Snapshot{Name: "orphan-snap", Dataset: "tank/k8s/vol-9"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truenasSnapshotCorrelatesWithK8s(tt.truenas, k8sSnaps)
			if got != tt.want {
				t.Fatalf("truenasSnapshotCorrelatesWithK8s() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectOrphanedSnapshots_FlagsMissingPeers(t *testing.T) {
	threshold := 24 * time.Hour
	old := metav1.NewTime(time.Now().Add(-48 * time.Hour))

	d := &Detector{
		config: Config{
			AgeThreshold:      threshold,
			SnapshotRetention: 30 * 24 * time.Hour,
		},
	}

	k8sSnaps := []snapshotv1.VolumeSnapshot{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "k8s-only",
				CreationTimestamp: old,
			},
		},
	}
	truenasSnaps := []truenas.Snapshot{
		{
			Name:      "truenas-only",
			Dataset:   "tank/k8s/vol-1",
			CreatedAt: time.Now().Add(-60 * 24 * time.Hour),
		},
	}

	orphaned, total, err := d.detectOrphanedSnapshotsFromLists(k8sSnaps, truenasSnaps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Fatalf("total snapshots = %d, want 1", total)
	}
	if len(orphaned) != 2 {
		t.Fatalf("orphaned count = %d, want 2 (one k8s-only, one truenas-only)", len(orphaned))
	}
}

func TestHasCorrespondingTrueNASVolume_EmptyCSI(t *testing.T) {
	d := &Detector{}
	pv := corev1.PersistentVolume{
		Spec: corev1.PersistentVolumeSpec{},
	}
	if d.hasCorrespondingTrueNASVolume(pv, nil) {
		t.Fatal("expected false when PV has no CSI source")
	}
}
