package truenas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config with API key",
			config: &Config{
				Host:   "truenas.example.com",
				Port:   443,
				APIKey: "test-api-key",
			},
			wantErr: false,
		},
		{
			name: "valid config with username/password",
			config: &Config{
				Host:     "truenas.example.com",
				Port:     80,
				Username: "admin",
				Password: "password",
			},
			wantErr: false,
		},
		{
			name: "invalid config - no auth",
			config: &Config{
				Host: "truenas.example.com",
			},
			wantErr: true,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_TestConnection(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		wantErr    bool
	}{
		{
			name:       "successful connection",
			statusCode: http.StatusOK,
			response:   map[string]interface{}{"username": "root"},
			wantErr:    false,
		},
		{
			name:       "authentication failure",
			statusCode: http.StatusUnauthorized,
			response:   map[string]interface{}{"message": "Invalid credentials"},
			wantErr:    true,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			response:   map[string]interface{}{"message": "Internal error"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v2.0/auth/me", r.URL.Path)
				assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := &Client{
				config: &Config{
					Host:   server.URL,
					APIKey: "test-key",
				},
				httpClient: &http.Client{Timeout: 5 * time.Second},
			}
			client.baseURL = server.URL + "/api/v2.0"

			err := client.TestConnection(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_GetPools(t *testing.T) {
	ctx := context.Background()
	
	mockPools := []map[string]interface{}{
		{
			"id":           1,
			"name":         "tank",
			"status":       "ONLINE",
			"size":         1099511627776,
			"allocated":    549755813888,
			"free":         549755813888,
			"fragmentation": "5%",
			"healthy":      true,
			"scan": map[string]interface{}{
				"state": "FINISHED",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2.0/pool", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockPools)
	}))
	defer server.Close()

	client := createTestClient(server.URL)

	pools, err := client.GetPools(ctx)
	require.NoError(t, err)
	require.Len(t, pools, 1)

	pool := pools[0]
	assert.Equal(t, "tank", pool.Name)
	assert.Equal(t, "ONLINE", pool.Status)
	assert.Equal(t, int64(1099511627776), pool.Size)
	assert.Equal(t, int64(549755813888), pool.Allocated)
	assert.True(t, pool.Healthy)
}

func TestClient_GetVolumes(t *testing.T) {
	ctx := context.Background()
	
	mockExtents := []map[string]interface{}{
		{
			"id":       1,
			"name":     "pvc-abc123",
			"type":     "FILE",
			"path":     "/mnt/tank/k8s/volumes/pvc-abc123",
			"filesize": 10737418240,
			"naa":      "naa.6589cfc0000000b4c7f2f0e8a91b6f3d",
			"enabled":  true,
			"ro":       false,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2.0/iscsi/extent", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockExtents)
	}))
	defer server.Close()

	client := createTestClient(server.URL)

	volumes, err := client.GetVolumes(ctx)
	require.NoError(t, err)
	require.Len(t, volumes, 1)

	volume := volumes[0]
	assert.Equal(t, "pvc-abc123", volume.Name)
	assert.Equal(t, "/mnt/tank/k8s/volumes/pvc-abc123", volume.Path)
	assert.Equal(t, int64(10737418240), volume.FileSize)
	assert.True(t, volume.Enabled)
}

func TestClient_GetSnapshots(t *testing.T) {
	ctx := context.Background()
	
	mockSnapshots := []map[string]interface{}{
		{
			"id":            "tank/k8s/volumes/pvc-abc123@snapshot-1",
			"dataset":       "tank/k8s/volumes/pvc-abc123",
			"snapshot_name": "snapshot-1",
			"properties": map[string]interface{}{
				"used": map[string]interface{}{
					"value": "1073741824",
				},
				"referenced": map[string]interface{}{
					"value": "10737418240",
				},
				"creation": map[string]interface{}{
					"value": "1704067200",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2.0/zfs/snapshot", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockSnapshots)
	}))
	defer server.Close()

	client := createTestClient(server.URL)

	snapshots, err := client.GetSnapshots(ctx, "")
	require.NoError(t, err)
	require.Len(t, snapshots, 1)

	snapshot := snapshots[0]
	assert.Equal(t, "snapshot-1", snapshot.SnapshotName)
	assert.Equal(t, "tank/k8s/volumes/pvc-abc123", snapshot.Dataset)
	assert.Equal(t, int64(1073741824), snapshot.Properties.Used)
}

func TestClient_CreateSnapshot(t *testing.T) {
	ctx := context.Background()
	dataset := "tank/k8s/volumes/pvc-test"
	snapshotName := "test-snapshot"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2.0/zfs/snapshot", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)

		assert.Equal(t, dataset, body["dataset"])
		assert.Equal(t, snapshotName, body["name"])
		assert.Equal(t, false, body["recursive"])

		response := map[string]interface{}{
			"id":            dataset + "@" + snapshotName,
			"dataset":       dataset,
			"snapshot_name": snapshotName,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := createTestClient(server.URL)

	snapshot, err := client.CreateSnapshot(ctx, dataset, snapshotName, false)
	require.NoError(t, err)
	assert.Equal(t, dataset+"@"+snapshotName, snapshot.ID)
}

func TestClient_DeleteSnapshot(t *testing.T) {
	ctx := context.Background()
	snapshotID := "tank/k8s/volumes/pvc-test@snapshot-1"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/v2.0/zfs/snapshot/id/")
		assert.Equal(t, http.MethodDelete, r.Method)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(true)
	}))
	defer server.Close()

	client := createTestClient(server.URL)

	err := client.DeleteSnapshot(ctx, snapshotID)
	assert.NoError(t, err)
}

func TestClient_FindOrphanedVolumes(t *testing.T) {
	ctx := context.Background()
	
	mockExtents := []map[string]interface{}{
		{
			"name": "pvc-orphaned",
			"path": "/mnt/tank/k8s/volumes/pvc-orphaned",
			"filesize": 5368709120,
		},
		{
			"name": "pvc-active",
			"path": "/mnt/tank/k8s/volumes/pvc-active",
		},
	}

	mockShares := []map[string]interface{}{
		{
			"path": "/mnt/tank/k8s/nfs/pvc-nfs-orphaned",
			"enabled": true,
		},
		{
			"path": "/mnt/tank/k8s/nfs/pvc-nfs-active",
			"enabled": true,
		},
	}

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch callCount {
		case 0:
			assert.Equal(t, "/api/v2.0/iscsi/extent", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockExtents)
		case 1:
			assert.Equal(t, "/api/v2.0/sharing/nfs", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockShares)
		}
		callCount++
	}))
	defer server.Close()

	client := createTestClient(server.URL)

	k8sVolumes := []string{"pvc-active", "pvc-nfs-active"}
	orphans, err := client.FindOrphanedVolumes(ctx, k8sVolumes)
	
	require.NoError(t, err)
	assert.Len(t, orphans, 2)
	
	// Check that orphaned volumes were found
	orphanNames := make(map[string]bool)
	for _, orphan := range orphans {
		orphanNames[orphan.Name] = true
	}
	
	assert.True(t, orphanNames["pvc-orphaned"])
	assert.True(t, orphanNames["pvc-nfs-orphaned"])
}

func TestClient_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Internal server error",
		})
	}))
	defer server.Close()

	client := createTestClient(server.URL)

	_, err := client.GetPools(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Internal server error")
}

// Helper function to create a test client
func createTestClient(serverURL string) *Client {
	return &Client{
		config: &Config{
			Host:   serverURL,
			APIKey: "test-key",
		},
		baseURL:    serverURL + "/api/v2.0",
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}