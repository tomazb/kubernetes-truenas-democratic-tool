package truenas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tomazb/kubernetes-truenas-democratic-tool/internal/types"
)

// Client is the TrueNAS API client
type Client struct {
	config     *Config
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new TrueNAS client
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.APIKey == "" && (config.Username == "" || config.Password == "") {
		return nil, fmt.Errorf("either APIKey or Username/Password must be provided")
	}

	// Set defaults
	if config.Port == 0 {
		config.Port = 443
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	// Build base URL
	scheme := "https"
	if config.Port == 80 {
		scheme = "http"
	}
	
	// Handle case where Host might already include scheme
	host := config.Host
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		// Extract just the host part
		u, err := url.Parse(host)
		if err != nil {
			return nil, fmt.Errorf("invalid host URL: %w", err)
		}
		host = u.Host
		if u.Scheme == "http" {
			scheme = "http"
		}
	}
	
	baseURL := fmt.Sprintf("%s://%s:%d/api/v2.0", scheme, host, config.Port)

	client := &Client{
		config:     config,
		baseURL:    baseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}

	return client, nil
}

// TestConnection tests the connection to TrueNAS
func (c *Client) TestConnection(ctx context.Context) error {
	req, err := c.newRequest(ctx, "GET", "/auth/me", nil)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := c.do(req, &result); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	return nil
}

// GetPools returns all storage pools
func (c *Client) GetPools(ctx context.Context) ([]*Pool, error) {
	req, err := c.newRequest(ctx, "GET", "/pool", nil)
	if err != nil {
		return nil, err
	}

	var pools []*Pool
	if err := c.do(req, &pools); err != nil {
		return nil, fmt.Errorf("failed to get pools: %w", err)
	}

	return pools, nil
}

// GetDatasets returns all datasets
func (c *Client) GetDatasets(ctx context.Context, pool string) ([]*Dataset, error) {
	endpoint := "/pool/dataset"
	if pool != "" {
		endpoint = fmt.Sprintf("%s?pool=%s", endpoint, url.QueryEscape(pool))
	}

	req, err := c.newRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var datasets []*Dataset
	if err := c.do(req, &datasets); err != nil {
		return nil, fmt.Errorf("failed to get datasets: %w", err)
	}

	return datasets, nil
}

// GetVolumes returns all iSCSI volumes (extents)
func (c *Client) GetVolumes(ctx context.Context) ([]*Volume, error) {
	req, err := c.newRequest(ctx, "GET", "/iscsi/extent", nil)
	if err != nil {
		return nil, err
	}

	var volumes []*Volume
	if err := c.do(req, &volumes); err != nil {
		return nil, fmt.Errorf("failed to get volumes: %w", err)
	}

	return volumes, nil
}

// GetNFSShares returns all NFS shares
func (c *Client) GetNFSShares(ctx context.Context) ([]*NFSShare, error) {
	req, err := c.newRequest(ctx, "GET", "/sharing/nfs", nil)
	if err != nil {
		return nil, err
	}

	var shares []*NFSShare
	if err := c.do(req, &shares); err != nil {
		return nil, fmt.Errorf("failed to get NFS shares: %w", err)
	}

	return shares, nil
}

// GetSnapshots returns all snapshots, optionally filtered by dataset
func (c *Client) GetSnapshots(ctx context.Context, dataset string) ([]*Snapshot, error) {
	endpoint := "/zfs/snapshot"
	if dataset != "" {
		endpoint = fmt.Sprintf("%s?dataset=%s", endpoint, url.QueryEscape(dataset))
	}

	req, err := c.newRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var rawSnapshots []map[string]interface{}
	if err := c.do(req, &rawSnapshots); err != nil {
		return nil, fmt.Errorf("failed to get snapshots: %w", err)
	}

	// Convert raw snapshots to typed structures
	snapshots := make([]*Snapshot, 0, len(rawSnapshots))
	for _, raw := range rawSnapshots {
		snapshot := &Snapshot{
			ID:           getString(raw, "id"),
			Dataset:      getString(raw, "dataset"),
			SnapshotName: getString(raw, "snapshot_name"),
		}

		// Parse properties
		if props, ok := raw["properties"].(map[string]interface{}); ok {
			snapshot.Properties = parseSnapshotProperties(props)
		}

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// CreateSnapshot creates a new ZFS snapshot
func (c *Client) CreateSnapshot(ctx context.Context, dataset, name string, recursive bool) (*Snapshot, error) {
	payload := map[string]interface{}{
		"dataset":   dataset,
		"name":      name,
		"recursive": recursive,
	}

	req, err := c.newRequest(ctx, "POST", "/zfs/snapshot", payload)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := c.do(req, &result); err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	snapshot := &Snapshot{
		ID:           getString(result, "id"),
		Dataset:      getString(result, "dataset"),
		SnapshotName: getString(result, "snapshot_name"),
	}

	return snapshot, nil
}

// DeleteSnapshot deletes a ZFS snapshot
func (c *Client) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	// URL encode the snapshot ID
	encodedID := url.QueryEscape(snapshotID)
	endpoint := fmt.Sprintf("/zfs/snapshot/id/%s", encodedID)

	req, err := c.newRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return err
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	return nil
}

// FindOrphanedVolumes finds volumes that exist in TrueNAS but not in Kubernetes
func (c *Client) FindOrphanedVolumes(ctx context.Context, k8sVolumeNames []string) ([]*OrphanedVolume, error) {
	k8sNamesSet := make(map[string]bool)
	for _, name := range k8sVolumeNames {
		k8sNamesSet[name] = true
	}

	var orphans []*OrphanedVolume

	// Check iSCSI volumes
	volumes, err := c.GetVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get iSCSI volumes: %w", err)
	}

	for _, volume := range volumes {
		if !k8sNamesSet[volume.Name] {
			orphan := &OrphanedVolume{
				Name: volume.Name,
				Path: volume.Path,
				Type: "iscsi",
				Size: volume.FileSize,
			}
			orphans = append(orphans, orphan)
		}
	}

	// Check NFS shares
	shares, err := c.GetNFSShares(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get NFS shares: %w", err)
	}

	for _, share := range shares {
		// Extract volume name from path (e.g., /mnt/tank/k8s/nfs/pvc-xxx)
		if strings.Contains(share.Path, "/k8s/nfs/") {
			parts := strings.Split(share.Path, "/k8s/nfs/")
			if len(parts) == 2 {
				volumeName := parts[1]
				if volumeName != "" && !k8sNamesSet[volumeName] {
					orphan := &OrphanedVolume{
						Name: volumeName,
						Path: share.Path,
						Type: "nfs",
					}
					orphans = append(orphans, orphan)
				}
			}
		}
	}

	return orphans, nil
}

// Helper methods

func (c *Client) newRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Request, error) {
	url := c.baseURL + endpoint
	
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set authentication
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	} else {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	return req, nil
}

func (c *Client) do(req *http.Request, result interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Message != "" {
			return fmt.Errorf("API error (status %d): %s", resp.StatusCode, errResp.Message)
		}
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response if result is provided
	if result != nil && len(body) > 0 {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// Helper functions

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func getInt64(m map[string]interface{}, key string) int64 {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return int64(v)
		case int64:
			return v
		case string:
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				return i
			}
		}
	}
	return 0
}

func parseSnapshotProperties(props map[string]interface{}) SnapshotProperties {
	sp := SnapshotProperties{}

	if used, ok := props["used"].(map[string]interface{}); ok {
		if val, ok := used["value"].(string); ok {
			sp.Used, _ = strconv.ParseInt(val, 10, 64)
		}
	}

	if referenced, ok := props["referenced"].(map[string]interface{}); ok {
		if val, ok := referenced["value"].(string); ok {
			sp.Referenced, _ = strconv.ParseInt(val, 10, 64)
		}
	}

	if creation, ok := props["creation"].(map[string]interface{}); ok {
		if val, ok := creation["value"].(string); ok {
			sp.Creation, _ = strconv.ParseInt(val, 10, 64)
		}
	}

	return sp
}

// ValidateConfiguration validates the TrueNAS client configuration and connectivity
func (c *Client) ValidateConfiguration(ctx context.Context) (*types.ValidationResult, error) {
	result := &types.ValidationResult{
		Valid:     true,
		Timestamp: time.Now(),
		Checks:    []types.HealthCheck{},
		Errors:    []string{},
		Warnings:  []string{},
	}

	// Test basic API connectivity by getting system info
	req, err := c.newRequest(ctx, "GET", "/api/v2.0/system/info", nil)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to create request: %v", err))
		result.Checks = append(result.Checks, types.HealthCheck{
			Name:      "truenas_api_connectivity",
			Healthy:   false,
			Message:   fmt.Sprintf("Cannot create API request: %v", err),
			Timestamp: time.Now(),
		})
		return result, nil
	}

	var systemInfo map[string]interface{}
	err = c.do(req, &systemInfo)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to connect to TrueNAS API: %v", err))
		result.Checks = append(result.Checks, types.HealthCheck{
			Name:      "truenas_api_connectivity",
			Healthy:   false,
			Message:   fmt.Sprintf("Cannot reach TrueNAS API: %v", err),
			Timestamp: time.Now(),
		})
		return result, nil
	}

	result.Checks = append(result.Checks, types.HealthCheck{
		Name:      "truenas_api_connectivity",
		Healthy:   true,
		Message:   "Successfully connected to TrueNAS API",
		Timestamp: time.Now(),
	})

	// Test authentication by trying to access pools
	_, err = c.GetPools(ctx)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Authentication failed: %v", err))
		result.Checks = append(result.Checks, types.HealthCheck{
			Name:      "truenas_authentication",
			Healthy:   false,
			Message:   fmt.Sprintf("Authentication failed: %v", err),
			Timestamp: time.Now(),
		})
		return result, nil
	}

	result.Checks = append(result.Checks, types.HealthCheck{
		Name:      "truenas_authentication",
		Healthy:   true,
		Message:   "Successfully authenticated with TrueNAS",
		Timestamp: time.Now(),
	})

	// Test pool access (required for storage operations)
	pools, err := c.GetPools(ctx)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Cannot access storage pools: %v", err))
		result.Checks = append(result.Checks, types.HealthCheck{
			Name:      "storage_pools_access",
			Healthy:   false,
			Message:   fmt.Sprintf("Limited storage pool access: %v", err),
			Timestamp: time.Now(),
		})
	} else if len(pools) == 0 {
		result.Warnings = append(result.Warnings, "No storage pools found")
		result.Checks = append(result.Checks, types.HealthCheck{
			Name:      "storage_pools_access",
			Healthy:   false,
			Message:   "No storage pools available",
			Timestamp: time.Now(),
		})
	} else {
		result.Checks = append(result.Checks, types.HealthCheck{
			Name:      "storage_pools_access",
			Healthy:   true,
			Message:   fmt.Sprintf("Successfully accessed %d storage pool(s)", len(pools)),
			Timestamp: time.Now(),
		})
	}

	// Test snapshot functionality
	_, err = c.GetSnapshots(ctx, "")
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Cannot access snapshots: %v", err))
		result.Checks = append(result.Checks, types.HealthCheck{
			Name:      "snapshots_access",
			Healthy:   false,
			Message:   fmt.Sprintf("Snapshot functionality may be limited: %v", err),
			Timestamp: time.Now(),
		})
	} else {
		result.Checks = append(result.Checks, types.HealthCheck{
			Name:      "snapshots_access",
			Healthy:   true,
			Message:   "Successfully accessed snapshot functionality",
			Timestamp: time.Now(),
		})
	}

	return result, nil
}