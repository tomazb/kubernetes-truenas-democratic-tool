package orphan

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/k8s"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/logging"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/truenas"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExtractDatasetFromVolumeHandle(t *testing.T) {
	tests := []struct {
		name   string
		handle string
		want   string
	}{
		{"iscsi handle", "iqn.2005-10.org.freenas.ctl:my-volume", "my-volume"},
		{"zfs path handle", "tank/k8s/vol-1", "vol-1"},
		{"plain handle", "standalone-id", "standalone-id"},
		{"zfs snapshot suffix stripped", "tank/k8s/vol-1@daily", "vol-1"},
		{"malformed iscsi trailing colon yields empty token", "iqn.2005-10.org.freenas.ctl:", ""},
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
			name:         "path suffix match",
			volume:       truenas.Volume{Name: "other", Path: "/mnt/tank/k8s/vol-1"},
			volumeHandle: "tank/k8s/vol-1",
			datasetName:  "vol-1",
			want:         true,
		},
		{
			name: "property exact match restored",
			volume: truenas.Volume{
				Name:       "unrelated",
				Properties: map[string]string{"zfs:dataset": "tank/k8s/vol-1"},
			},
			volumeHandle: "tank/k8s/vol-1",
			datasetName:  "vol-1",
			want:         true,
		},
		{
			name: "property substring collision avoided",
			volume: truenas.Volume{
				Name:       "unrelated",
				Properties: map[string]string{"note": "prefix-vol-1-suffix"},
			},
			volumeHandle: "tank/k8s/vol-1",
			datasetName:  "vol-1",
			want:         false,
		},
		{
			name:         "vol-1 does not match vol-10 path",
			volume:       truenas.Volume{Path: "/mnt/tank/k8s/vol-10"},
			volumeHandle: "tank/k8s/vol-1",
			datasetName:  "vol-1",
			want:         false,
		},
		{
			name:         "empty dataset token never matches",
			volume:       truenas.Volume{ID: "anything"},
			volumeHandle: "tank/k8s/",
			datasetName:  "",
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
			Name:              "daily",
			Namespace:         "apps",
			CreationTimestamp: metav1.NewTime(old),
			Annotations: map[string]string{
				"zfs.dataset": "tank/k8s/vol-1",
			},
		},
	}

	tests := []struct {
		name    string
		truenas []truenas.Snapshot
		want    bool
	}{
		{
			name: "matching name and dataset",
			truenas: []truenas.Snapshot{
				{Name: "daily", Dataset: "tank/k8s/vol-1"},
			},
			want: true,
		},
		{
			name: "full zfs snapshot id with dataset hint",
			truenas: []truenas.Snapshot{
				{Name: "tank/k8s/vol-1@daily", Dataset: "tank/k8s/vol-1"},
			},
			want: true,
		},
		{
			name: "same snapshot name on different dataset is not a peer",
			truenas: []truenas.Snapshot{
				{Name: "daily", Dataset: "tank/k8s/vol-2"},
			},
			want: false,
		},
		{
			name: "no peer",
			truenas: []truenas.Snapshot{
				{Name: "other-snap", Dataset: "tank/k8s/vol-9"},
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
				Annotations: map[string]string{
					"zfs.dataset": "tank/k8s/vol-1",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "tank/k8s/vol-1@snap-a",
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
			name:    "name and dataset hint match",
			truenas: truenas.Snapshot{Name: "snap-a", Dataset: "tank/k8s/vol-1"},
			want:    true,
		},
		{
			name:    "full zfs id matches k8s object named with full id",
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

type pvcListSpyK8sClient struct {
	pvcListCalls     int32
	unboundListCalls int32
}

func (s *pvcListSpyK8sClient) ListPersistentVolumeClaims(context.Context, string) ([]corev1.PersistentVolumeClaim, error) {
	atomic.AddInt32(&s.pvcListCalls, 1)
	old := time.Now().Add(-48 * time.Hour)
	return []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pending-old",
				Namespace:         "apps",
				CreationTimestamp: metav1.NewTime(old),
			},
			Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "bound-old",
				Namespace:         "apps",
				CreationTimestamp: metav1.NewTime(old),
			},
			Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
		},
	}, nil
}

func (s *pvcListSpyK8sClient) ListPersistentVolumes(context.Context) ([]corev1.PersistentVolume, error) {
	return nil, nil
}

func (s *pvcListSpyK8sClient) ListVolumeSnapshots(context.Context, string) ([]snapshotv1.VolumeSnapshot, error) {
	return nil, nil
}

func (s *pvcListSpyK8sClient) ListStorageClasses(context.Context) ([]storagev1.StorageClass, error) {
	return nil, nil
}

func (s *pvcListSpyK8sClient) ListPods(context.Context, string) ([]corev1.Pod, error) {
	return nil, nil
}

func (s *pvcListSpyK8sClient) ListNamespaces(context.Context) ([]corev1.Namespace, error) {
	return nil, nil
}

func (s *pvcListSpyK8sClient) GetNamespace(context.Context, string) (*corev1.Namespace, error) {
	return nil, nil
}

func (s *pvcListSpyK8sClient) ListPersistentVolumesByStorageClass(context.Context, string) ([]corev1.PersistentVolume, error) {
	return nil, nil
}

func (s *pvcListSpyK8sClient) ListPersistentVolumeClaimsByStorageClass(context.Context, string, string) ([]corev1.PersistentVolumeClaim, error) {
	return nil, nil
}

func (s *pvcListSpyK8sClient) ListDemocraticCSIPersistentVolumes(context.Context) ([]corev1.PersistentVolume, error) {
	return nil, nil
}

func (s *pvcListSpyK8sClient) ListUnboundPersistentVolumeClaims(context.Context, string) ([]corev1.PersistentVolumeClaim, error) {
	atomic.AddInt32(&s.unboundListCalls, 1)
	return nil, nil
}

func (s *pvcListSpyK8sClient) TestConnection(context.Context) error { return nil }

func (s *pvcListSpyK8sClient) ValidateRBACPermissions(context.Context) (*k8s.RBACValidationResult, error) {
	return nil, nil
}

func (s *pvcListSpyK8sClient) GetClusterInfo(context.Context) (*k8s.ClusterInfo, error) {
	return nil, nil
}

func (s *pvcListSpyK8sClient) ListCSINodes(context.Context) ([]storagev1.CSINode, error) {
	return nil, nil
}

func (s *pvcListSpyK8sClient) ListCSIDrivers(context.Context) ([]storagev1.CSIDriver, error) {
	return nil, nil
}

func (s *pvcListSpyK8sClient) ListVolumeAttachments(context.Context) ([]storagev1.VolumeAttachment, error) {
	return nil, nil
}

func (s *pvcListSpyK8sClient) GetCSIDriverPods(context.Context, string) ([]corev1.Pod, error) {
	return nil, nil
}

func TestDetectOrphanedPVCs_SingleListCall(t *testing.T) {
	spy := &pvcListSpyK8sClient{}
	logger, err := logging.NewLogger(logging.Config{Level: "error", Encoding: "json"})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	d := &Detector{
		k8sClient: spy,
		logger:    logger,
		config: Config{
			AgeThreshold: 24 * time.Hour,
		},
	}

	orphaned, total, err := d.detectOrphanedPVCs(context.Background(), "apps", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spy.pvcListCalls != 1 {
		t.Fatalf("ListPersistentVolumeClaims calls = %d, want 1", spy.pvcListCalls)
	}
	if spy.unboundListCalls != 0 {
		t.Fatalf("ListUnboundPersistentVolumeClaims calls = %d, want 0", spy.unboundListCalls)
	}
	if total != 2 {
		t.Fatalf("total PVCs = %d, want 2", total)
	}
	if len(orphaned) != 1 {
		t.Fatalf("orphaned PVCs = %d, want 1", len(orphaned))
	}
}
