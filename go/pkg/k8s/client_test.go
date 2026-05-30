package k8s

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	authorizationv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	snapshotfake "github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned/fake"

	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/logging"
)

func testLogger(t *testing.T) *logging.Logger {
	t.Helper()
	logger, err := logging.NewLogger(logging.Config{
		Level:    "error",
		Encoding: "json",
	})
	if err != nil {
		t.Fatalf("failed to create test logger: %v", err)
	}
	return logger
}

func TestNewClient(t *testing.T) {
	t.Run("success with minimal kubeconfig", func(t *testing.T) {
		dir := t.TempDir()
		kubeconfig := filepath.Join(dir, "config")
		content := []byte(`apiVersion: v1
kind: Config
clusters:
- name: local
  cluster:
    server: https://127.0.0.1:6443
contexts:
- name: local
  context:
    cluster: local
    user: test
current-context: local
users:
- name: test
  user:
    token: fake-token
`)
		if err := os.WriteFile(kubeconfig, content, 0o600); err != nil {
			t.Fatalf("failed to write kubeconfig: %v", err)
		}

		c, err := NewClient(Config{Kubeconfig: kubeconfig})
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if c == nil {
			t.Fatal("expected non-nil client")
		}
	})

	t.Run("missing kubeconfig file", func(t *testing.T) {
		_, err := NewClient(Config{Kubeconfig: "testdata/does-not-exist"})
		if err == nil {
			t.Fatal("expected error for missing kubeconfig file")
		}
	})
}

func TestClient_ListPersistentVolumes(t *testing.T) {
	ctx := context.Background()

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
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{
					Driver:       "org.democratic-csi.nfs",
					VolumeHandle: "nfs-volume-1",
				},
			},
		},
	}

	pv2 := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "pv-test-2"},
		Spec: v1.PersistentVolumeSpec{
			Capacity: v1.ResourceList{
				v1.ResourceStorage: resource.MustParse("20Gi"),
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				HostPath: &v1.HostPathVolumeSource{Path: "/tmp/data"},
			},
		},
	}

	fakeClient := fake.NewSimpleClientset(pv1, pv2)
	c := &client{
		clientset: fakeClient,
		config:    Config{Namespace: "default"},
		logger:    testLogger(t),
	}

	pvs, err := c.ListPersistentVolumes(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pvs) != 2 {
		t.Fatalf("expected 2 PVs, got %d", len(pvs))
	}
}

func TestClient_ListPersistentVolumeClaims(t *testing.T) {
	ctx := context.Background()
	namespace := "test-namespace"

	pvc1 := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pvc-test-1",
			Namespace: namespace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			StorageClassName: stringPtr("democratic-csi-nfs"),
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
	}

	fakeClient := fake.NewSimpleClientset(pvc1, pvc2)
	c := &client{
		clientset: fakeClient,
		config:    Config{Namespace: namespace},
		logger:    testLogger(t),
	}

	pvcs, err := c.ListPersistentVolumeClaims(ctx, namespace)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pvcs) != 2 {
		t.Fatalf("expected 2 PVCs, got %d", len(pvcs))
	}
}

func TestClient_ListStorageClasses(t *testing.T) {
	ctx := context.Background()

	sc1 := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "democratic-csi-nfs"},
		Provisioner: "org.democratic-csi.nfs",
	}
	sc2 := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "local-storage"},
		Provisioner: "kubernetes.io/no-provisioner",
	}

	fakeClient := fake.NewSimpleClientset(sc1, sc2)
	c := &client{
		clientset: fakeClient,
		config:    Config{Namespace: "default"},
		logger:    testLogger(t),
	}

	scs, err := c.ListStorageClasses(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scs) != 2 {
		t.Fatalf("expected 2 StorageClasses, got %d", len(scs))
	}
}

func stringPtr(s string) *string {
	return &s
}

func TestClient_ValidateRBACPermissions_Allowed(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	fakeClient.PrependReactor(
		"create",
		"selfsubjectaccessreviews",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			review := action.(k8stesting.CreateAction).GetObject().(*authorizationv1.SelfSubjectAccessReview)
			review.Status = authorizationv1.SubjectAccessReviewStatus{Allowed: true}
			return true, review, nil
		},
	)

	c := &client{
		clientset:      fakeClient,
		snapshotClient: snapshotfake.NewSimpleClientset(),
		config:         Config{Namespace: "monitoring"},
		logger:         testLogger(t),
	}

	result, err := c.ValidateRBACPermissions(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasRequiredPermissions {
		t.Fatalf("expected HasRequiredPermissions true, missing=%v", result.MissingPermissions)
	}
	if !result.PermissionChecks["persistentvolumes/list"] {
		t.Fatal("expected persistentvolumes/list allowed")
	}
	if result.Namespace != "monitoring" {
		t.Fatalf("namespace = %q, want monitoring", result.Namespace)
	}
}

func TestClient_ValidateRBACPermissions_Denied(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	fakeClient.PrependReactor(
		"create",
		"selfsubjectaccessreviews",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			review := action.(k8stesting.CreateAction).GetObject().(*authorizationv1.SelfSubjectAccessReview)
			allowed := review.Spec.ResourceAttributes.Resource != "persistentvolumes"
			review.Status = authorizationv1.SubjectAccessReviewStatus{Allowed: allowed}
			return true, review, nil
		},
	)

	c := &client{
		clientset:      fakeClient,
		snapshotClient: nil,
		config:         Config{Namespace: "default"},
		logger:         testLogger(t),
	}

	result, err := c.ValidateRBACPermissions(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasRequiredPermissions {
		t.Fatal("expected HasRequiredPermissions false when PV list denied")
	}
	if len(result.MissingPermissions) == 0 {
		t.Fatal("expected missing permissions listed")
	}
	foundSkip := false
	for _, m := range result.MissingPermissions {
		if m == "skipped: volumesnapshots (snapshot client unavailable)" {
			foundSkip = true
		}
	}
	if !foundSkip {
		t.Fatalf("expected snapshot skip note, got %v", result.MissingPermissions)
	}
}
