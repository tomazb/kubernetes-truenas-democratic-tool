package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	"github.com/stretchr/testify/require"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/k8s"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/truenas"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"go.uber.org/zap"
)

type stubK8sClient struct {
	democraticPVs      []corev1.PersistentVolume
	democraticPVsErr   error
	unboundPVCs        []corev1.PersistentVolumeClaim
	allPVCs            []corev1.PersistentVolumeClaim
	volumeSnapshots    []snapshotv1.VolumeSnapshot
	listPersistentPVs  []corev1.PersistentVolume
	testConnectionErr  error
}

func (s *stubK8sClient) ListPersistentVolumes(context.Context) ([]corev1.PersistentVolume, error) {
	return s.listPersistentPVs, nil
}

func (s *stubK8sClient) ListPersistentVolumeClaims(context.Context, string) ([]corev1.PersistentVolumeClaim, error) {
	if s.allPVCs == nil {
		return []corev1.PersistentVolumeClaim{}, nil
	}
	return s.allPVCs, nil
}

func (s *stubK8sClient) ListVolumeSnapshots(context.Context, string) ([]snapshotv1.VolumeSnapshot, error) {
	if s.volumeSnapshots == nil {
		return []snapshotv1.VolumeSnapshot{}, nil
	}
	return s.volumeSnapshots, nil
}

func (s *stubK8sClient) ListStorageClasses(context.Context) ([]storagev1.StorageClass, error) {
	return nil, nil
}

func (s *stubK8sClient) ListPods(context.Context, string) ([]corev1.Pod, error) {
	return nil, nil
}

func (s *stubK8sClient) ListNamespaces(context.Context) ([]corev1.Namespace, error) {
	return nil, nil
}

func (s *stubK8sClient) GetNamespace(context.Context, string) (*corev1.Namespace, error) {
	return nil, nil
}

func (s *stubK8sClient) ListPersistentVolumesByStorageClass(context.Context, string) ([]corev1.PersistentVolume, error) {
	return nil, nil
}

func (s *stubK8sClient) ListPersistentVolumeClaimsByStorageClass(context.Context, string, string) ([]corev1.PersistentVolumeClaim, error) {
	return nil, nil
}

func (s *stubK8sClient) ListDemocraticCSIPersistentVolumes(context.Context) ([]corev1.PersistentVolume, error) {
	if s.democraticPVsErr != nil {
		return nil, s.democraticPVsErr
	}
	if s.democraticPVs == nil {
		return []corev1.PersistentVolume{}, nil
	}
	return s.democraticPVs, nil
}

func (s *stubK8sClient) ListUnboundPersistentVolumeClaims(context.Context, string) ([]corev1.PersistentVolumeClaim, error) {
	if s.unboundPVCs == nil {
		return []corev1.PersistentVolumeClaim{}, nil
	}
	return s.unboundPVCs, nil
}

func (s *stubK8sClient) TestConnection(context.Context) error {
	return s.testConnectionErr
}

func (s *stubK8sClient) ValidateRBACPermissions(context.Context) (*k8s.RBACValidationResult, error) {
	return nil, nil
}

func (s *stubK8sClient) GetClusterInfo(context.Context) (*k8s.ClusterInfo, error) {
	return nil, nil
}

func (s *stubK8sClient) ListCSINodes(context.Context) ([]storagev1.CSINode, error) {
	return nil, nil
}

func (s *stubK8sClient) ListCSIDrivers(context.Context) ([]storagev1.CSIDriver, error) {
	return nil, nil
}

func (s *stubK8sClient) ListVolumeAttachments(context.Context) ([]storagev1.VolumeAttachment, error) {
	return nil, nil
}

func (s *stubK8sClient) GetCSIDriverPods(context.Context, string) ([]corev1.Pod, error) {
	return nil, nil
}

type stubTruenasClient struct {
	volumes           []truenas.Volume
	snapshots         []truenas.Snapshot
	testConnectionErr error
	listVolumesErr    error
}

func (s *stubTruenasClient) ListVolumes(context.Context) ([]truenas.Volume, error) {
	if s.listVolumesErr != nil {
		return nil, s.listVolumesErr
	}
	if s.volumes == nil {
		return []truenas.Volume{}, nil
	}
	return s.volumes, nil
}

func (s *stubTruenasClient) ListSnapshots(context.Context) ([]truenas.Snapshot, error) {
	if s.snapshots == nil {
		return []truenas.Snapshot{}, nil
	}
	return s.snapshots, nil
}

func (s *stubTruenasClient) ListPools(context.Context) ([]truenas.Pool, error) {
	return nil, nil
}

func (s *stubTruenasClient) GetSystemInfo(context.Context) (*truenas.SystemInfo, error) {
	return nil, nil
}

func (s *stubTruenasClient) TestConnection(context.Context) error {
	return s.testConnectionErr
}

func newTestServer(t *testing.T, k8sClient k8s.Client, truenasClient truenas.Client) *Server {
	t.Helper()

	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	server, err := NewServer(Config{
		Port:          0,
		K8sClient:     k8sClient,
		TruenasClient: truenasClient,
		Logger:        logger,
	})
	require.NoError(t, err)

	return server
}

func performRequest(server *Server, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(rec, req)
	return rec
}

func orphanedDemocraticPV(name string) corev1.PersistentVolume {
	return corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			CreationTimestamp: metav1.NewTime(time.Now().Add(-48 * time.Hour)),
		},
		Spec: corev1.PersistentVolumeSpec{
			StorageClassName: "democratic-csi-nfs",
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{
					Driver:       "org.democratic-csi.nfs",
					VolumeHandle: "tank/k8s/" + name,
				},
			},
		},
	}
}

func TestListOrphansHandler_ReturnsDetectorResults(t *testing.T) {
	k8sStub := &stubK8sClient{
		democraticPVs: []corev1.PersistentVolume{orphanedDemocraticPV("orphan-pv")},
	}
	truenasStub := &stubTruenasClient{volumes: []truenas.Volume{}}
	server := newTestServer(t, k8sStub, truenasStub)

	rec := performRequest(server, http.MethodGet, "/api/v1/orphans?age_threshold=24h")
	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	orphanedPVs, ok := body["orphaned_pvs"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, orphanedPVs)
	require.EqualValues(t, 1, body["total_orphans"])
}

func TestListOrphansHandler_InvalidAgeThreshold_Returns400(t *testing.T) {
	server := newTestServer(t, &stubK8sClient{}, &stubTruenasClient{})

	rec := performRequest(server, http.MethodGet, "/api/v1/orphans?age_threshold=not-a-duration")
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "invalid age_threshold format", body["error"])
}

func TestListOrphanedPVsHandler_ReturnsPVSubsetOnly(t *testing.T) {
	k8sStub := &stubK8sClient{
		democraticPVs: []corev1.PersistentVolume{orphanedDemocraticPV("orphan-pv")},
	}
	truenasStub := &stubTruenasClient{volumes: []truenas.Volume{}}
	server := newTestServer(t, k8sStub, truenasStub)

	rec := performRequest(server, http.MethodGet, "/api/v1/orphans/pvs?age_threshold=24h")
	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	orphanedPVs, ok := body["orphaned_pvs"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, orphanedPVs)
	require.EqualValues(t, 1, body["total_orphans"])
	require.NotContains(t, body, "orphaned_pvcs")
	require.NotContains(t, body, "orphaned_snapshots")
}

func TestListOrphansHandler_DetectorError_Returns500(t *testing.T) {
	k8sStub := &stubK8sClient{
		democraticPVsErr: errors.New("kubernetes unavailable"),
	}
	server := newTestServer(t, k8sStub, &stubTruenasClient{})

	rec := performRequest(server, http.MethodGet, "/api/v1/orphans")
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "orphan detection failed", body["error"])
}

func TestNotImplementedRoutes_Return501WithStandardEnvelope(t *testing.T) {
	server := newTestServer(t, &stubK8sClient{}, &stubTruenasClient{})

	routes := []struct {
		path     string
		endpoint string
	}{
		{"/api/v1/orphans/pvcs", "/api/v1/orphans/pvcs"},
		{"/api/v1/orphans/snapshots", "/api/v1/orphans/snapshots"},
		{"/api/v1/analysis", "/api/v1/analysis"},
		{"/api/v1/analysis/usage", "/api/v1/analysis/usage"},
		{"/api/v1/analysis/trends", "/api/v1/analysis/trends"},
		{"/api/v1/resources/pvcs", "/api/v1/resources/pvcs"},
		{"/api/v1/resources/snapshots", "/api/v1/resources/snapshots"},
		{"/api/v1/resources/storageclasses", "/api/v1/resources/storageclasses"},
		{"/api/v1/truenas/snapshots", "/api/v1/truenas/snapshots"},
		{"/api/v1/truenas/pools", "/api/v1/truenas/pools"},
		{"/api/v1/truenas/info", "/api/v1/truenas/info"},
		{"/api/v1/validate/config", "/api/v1/validate/config"},
		{"/api/v1/validate/connectivity", "/api/v1/validate/connectivity"},
		{"/api/v1/reports/summary", "/api/v1/reports/summary"},
		{"/api/v1/reports/detailed", "/api/v1/reports/detailed"},
	}

	for _, route := range routes {
		t.Run(route.path, func(t *testing.T) {
			rec := performRequest(server, http.MethodGet, route.path)
			require.Equal(t, http.StatusNotImplemented, rec.Code)

			var body map[string]interface{}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, "not_implemented", body["error"])
			require.Equal(t, "endpoint not implemented", body["message"])
			require.Equal(t, route.endpoint, body["endpoint"])
		})
	}
}
