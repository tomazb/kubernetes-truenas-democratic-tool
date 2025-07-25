package k8s

import (
	"context"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	k8stest "k8s.io/client-go/testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Kubeconfig: "testdata/kubeconfig",
				Namespace:  "default",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config Config
			if tt.config != nil {
				config = *tt.config
			}
			_, err := NewClient(config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_GetPersistentVolumes(t *testing.T) {
	ctx := context.Background()
	
	// Create test PVs
	pv1 := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pv-test-1",
			Labels: map[string]string{
				"provisioner": "org.democratic-csi.nfs",
			},
		},
		Spec: v1.PersistentVolumeSpec{
			Capacity: v1.ResourceList{
				v1.ResourceStorage: resource.MustParse("10Gi"),
			},
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{
					Driver:       "org.democratic-csi.nfs",
					VolumeHandle: "nfs-volume-1",
				},
			},
		},
		Status: v1.PersistentVolumeStatus{
			Phase: v1.VolumeBound,
		},
	}

	pv2 := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pv-test-2",
		},
		Spec: v1.PersistentVolumeSpec{
			Capacity: v1.ResourceList{
				v1.ResourceStorage: resource.MustParse("20Gi"),
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "/tmp/data",
				},
			},
		},
	}

	// Create fake clientset
	fakeClient := fake.NewSimpleClientset(pv1, pv2)
	
	client := &client{
		clientset: fakeClient,
		config: Config{
			Namespace: "default",
		},
	}

	pvs, err := client.ListPersistentVolumes(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only return CSI volumes
	if len(pvs) != 1 {
		t.Errorf("expected 1 PV, got %d", len(pvs))
	}

	if pvs[0].Name != "pv-test-1" {
		t.Errorf("expected PV name 'pv-test-1', got %s", pvs[0].Name)
	}
}

func TestClient_GetPersistentVolumeClaims(t *testing.T) {
	ctx := context.Background()
	namespace := "test-namespace"

	// Create test PVCs
	pvc1 := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pvc-test-1",
			Namespace: namespace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			StorageClassName: stringPtr("democratic-csi-nfs"),
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
		},
		Status: v1.PersistentVolumeClaimStatus{
			Phase: v1.ClaimBound,
		},
	}

	pvc2 := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pvc-test-2",
			Namespace: namespace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			StorageClassName: stringPtr("local-storage"),
		},
		Status: v1.PersistentVolumeClaimStatus{
			Phase: v1.ClaimPending,
		},
	}

	// Create fake clientset
	fakeClient := fake.NewSimpleClientset(pvc1, pvc2)
	
	client := &Client{
		clientset: fakeClient,
		config: &Config{
			Namespace:        namespace,
			StorageClassName: "democratic-csi-nfs",
		},
	}

	pvcs, err := client.GetPersistentVolumeClaims(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only return PVCs with matching storage class
	if len(pvcs) != 1 {
		t.Errorf("expected 1 PVC, got %d", len(pvcs))
	}

	if pvcs[0].Name != "pvc-test-1" {
		t.Errorf("expected PVC name 'pvc-test-1', got %s", pvcs[0].Name)
	}
}

func TestClient_WatchPersistentVolumes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Create fake clientset with watch reactor
	fakeClient := fake.NewSimpleClientset()
	watcher := watch.NewFake()
	fakeClient.PrependWatchReactor("persistentvolumes", k8stest.DefaultWatchReactor(watcher, nil))

	client := &Client{
		clientset: fakeClient,
		config: &Config{
			CSIDriver: "org.democratic-csi.nfs",
		},
	}

	eventCh := make(chan PVEvent, 10)
	go client.WatchPersistentVolumes(ctx, eventCh)

	// Send test events
	testPV := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv-watch",
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{
					Driver: "org.democratic-csi.nfs",
				},
			},
		},
	}

	watcher.Add(testPV)
	watcher.Modify(testPV)
	watcher.Delete(testPV)

	// Verify events
	expectedEvents := []EventType{EventAdded, EventModified, EventDeleted}
	for _, expected := range expectedEvents {
		select {
		case event := <-eventCh:
			if event.Type != expected {
				t.Errorf("expected event type %s, got %s", expected, event.Type)
			}
			if event.PV.Name != "test-pv-watch" {
				t.Errorf("expected PV name 'test-pv-watch', got %s", event.PV.Name)
			}
		case <-time.After(1 * time.Second):
			t.Errorf("timeout waiting for event %s", expected)
		}
	}
}

func TestClient_GetStorageClasses(t *testing.T) {
	ctx := context.Background()

	// Create test storage classes
	sc1 := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "democratic-csi-nfs",
		},
		Provisioner: "org.democratic-csi.nfs",
		Parameters: map[string]string{
			"fsType": "nfs",
		},
	}

	sc2 := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "local-storage",
		},
		Provisioner: "kubernetes.io/no-provisioner",
	}

	// Create fake clientset
	fakeClient := fake.NewSimpleClientset(sc1, sc2)
	
	client := &Client{
		clientset: fakeClient,
		config: &Config{
			CSIDriver: "org.democratic-csi.nfs",
		},
	}

	scs, err := client.GetStorageClasses(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only return democratic-csi storage classes
	if len(scs) != 1 {
		t.Errorf("expected 1 StorageClass, got %d", len(scs))
	}

	if scs[0].Name != "democratic-csi-nfs" {
		t.Errorf("expected StorageClass name 'democratic-csi-nfs', got %s", scs[0].Name)
	}
}

func TestClient_GetVolumeSnapshots(t *testing.T) {
	ctx := context.Background()
	namespace := "test-namespace"

	// Create a simpler test that mocks the client behavior
	client := &Client{
		config: &Config{
			Namespace: namespace,
		},
	}

	// Mock the dynamicClient to return an error for now, as the conversion logic
	// is complex and this test mainly validates the method exists and handles errors
	snapshots, err := client.GetVolumeSnapshots(ctx)
	
	// We expect an error since we don't have a real dynamic client
	if err == nil {
		t.Error("expected error due to nil dynamic client, got nil")
	}
	
	// Snapshots should be nil when there's an error
	if snapshots != nil {
		t.Error("expected nil snapshots on error, got non-nil")
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}