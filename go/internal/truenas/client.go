package truenas

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tomazb/kubernetes-truenas-democratic-tool/internal/config"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/internal/types"
)

// Client represents a TrueNAS API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	username   string
	password   string
}

// NewClient creates a new TrueNAS API client
func NewClient(cfg *config.TrueNASConfig) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("TrueNAS URL is required")
	}

	// Parse URL to validate it
	parsedURL, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid TrueNAS URL: %w", err)
	}

	// Create HTTP client with timeout
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	httpClient := &http.Client{
		Timeout: timeout,
	}

	// Configure TLS if needed
	if cfg.Insecure {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	client := &Client{
		baseURL:    strings.TrimSuffix(parsedURL.String(), "/"),
		httpClient: httpClient,
		apiKey:     cfg.APIKey,
		username:   cfg.Username,
		password:   cfg.Password,
	}

	return client, nil
}

// TestConnection tests the connection to TrueNAS
func (c *Client) TestConnection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v2.0/system/info", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to TrueNAS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("TrueNAS API returned status %d", resp.StatusCode)
	}

	return nil
}

// setAuth sets authentication headers
func (c *Client) setAuth(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	} else if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
}

// GetPools returns all storage pools
func (c *Client) GetPools(ctx context.Context) ([]types.TrueNASPool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v2.0/pool", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get pools: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TrueNAS API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rawPools []map[string]interface{}
	if err := json.Unmarshal(body, &rawPools); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var pools []types.TrueNASPool
	for _, rawPool := range rawPools {
		pool := types.TrueNASPool{
			Name:   getString(rawPool, "name"),
			Status: getString(rawPool, "status"),
		}

		// Parse pool statistics
		if stats, ok := rawPool["stats"].(map[string]interface{}); ok {
			pool.TotalSize = int64(getFloat64(stats, "size"))
			pool.UsedSize = int64(getFloat64(stats, "allocated"))
			pool.FreeSize = pool.TotalSize - pool.UsedSize
		}

		// Determine health status
		pool.Healthy = pool.Status == "ONLINE"

		// Get fragmentation if available
		if frag, ok := rawPool["fragmentation"].(string); ok {
			pool.Fragmentation = frag
		}

		pools = append(pools, pool)
	}

	return pools, nil
}

// GetDatasets returns all datasets
func (c *Client) GetDatasets(ctx context.Context) ([]types.TrueNASDataset, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v2.0/pool/dataset", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get datasets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TrueNAS API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rawDatasets []map[string]interface{}
	if err := json.Unmarshal(body, &rawDatasets); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var datasets []types.TrueNASDataset
	for _, rawDataset := range rawDatasets {
		dataset := types.TrueNASDataset{
			Name: getString(rawDataset, "name"),
			Type: getString(rawDataset, "type"),
		}

		// Parse pool name from dataset name
		parts := strings.Split(dataset.Name, "/")
		if len(parts) > 0 {
			dataset.Pool = parts[0]
		}

		// Parse usage statistics
		dataset.Used = int64(getFloat64(rawDataset, "used"))
		dataset.Available = int64(getFloat64(rawDataset, "available"))
		dataset.Referenced = int64(getFloat64(rawDataset, "referenced"))

		// Parse properties
		if props, ok := rawDataset["properties"].(map[string]interface{}); ok {
			dataset.Properties = make(map[string]string)
			for key, value := range props {
				if strValue, ok := value.(string); ok {
					dataset.Properties[key] = strValue
				}
			}
			dataset.Compression = dataset.Properties["compression"]
		}

		datasets = append(datasets, dataset)
	}

	return datasets, nil
}

// GetSnapshots returns all snapshots
func (c *Client) GetSnapshots(ctx context.Context) ([]types.TrueNASSnapshot, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v2.0/zfs/snapshot", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TrueNAS API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rawSnapshots []map[string]interface{}
	if err := json.Unmarshal(body, &rawSnapshots); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var snapshots []types.TrueNASSnapshot
	for _, rawSnapshot := range rawSnapshots {
		snapshot := types.TrueNASSnapshot{
			FullName:   getString(rawSnapshot, "name"),
			UsedSize:   int64(getFloat64(rawSnapshot, "used")),
			Referenced: int64(getFloat64(rawSnapshot, "referenced")),
		}

		// Parse dataset and snapshot name
		parts := strings.Split(snapshot.FullName, "@")
		if len(parts) == 2 {
			snapshot.Dataset = parts[0]
			snapshot.Name = parts[1]
		}

		// Parse creation time
		if createTime, ok := rawSnapshot["createtxg"].(float64); ok {
			// Convert TXG to approximate timestamp (this is a simplification)
			snapshot.CreationTime = time.Unix(int64(createTime), 0)
		}

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// GetVolumes returns all volumes (iSCSI and NFS)
func (c *Client) GetVolumes(ctx context.Context) ([]types.TrueNASVolume, error) {
	// Get iSCSI volumes
	iscsiVolumes, err := c.getISCSIVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get iSCSI volumes: %w", err)
	}

	// Get NFS shares (treated as volumes)
	nfsVolumes, err := c.getNFSVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get NFS volumes: %w", err)
	}

	// Combine all volumes
	volumes := append(iscsiVolumes, nfsVolumes...)
	return volumes, nil
}

// getISCSIVolumes gets iSCSI volumes
func (c *Client) getISCSIVolumes(ctx context.Context) ([]types.TrueNASVolume, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v2.0/iscsi/extent", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get iSCSI extents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TrueNAS API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rawExtents []map[string]interface{}
	if err := json.Unmarshal(body, &rawExtents); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var volumes []types.TrueNASVolume
	for _, rawExtent := range rawExtents {
		volume := types.TrueNASVolume{
			Name:   getString(rawExtent, "name"),
			Type:   "iscsi",
			Path:   getString(rawExtent, "path"),
			Size:   int64(getFloat64(rawExtent, "filesize")),
			Status: "active", // Simplification
		}

		volumes = append(volumes, volume)
	}

	return volumes, nil
}

// getNFSVolumes gets NFS shares
func (c *Client) getNFSVolumes(ctx context.Context) ([]types.TrueNASVolume, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v2.0/sharing/nfs", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get NFS shares: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TrueNAS API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rawShares []map[string]interface{}
	if err := json.Unmarshal(body, &rawShares); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var volumes []types.TrueNASVolume
	for _, rawShare := range rawShares {
		volume := types.TrueNASVolume{
			Name:   getString(rawShare, "comment"),
			Type:   "nfs",
			Path:   getString(rawShare, "path"),
			Status: "active", // Simplification
		}

		if volume.Name == "" {
			volume.Name = getString(rawShare, "path")
		}

		volumes = append(volumes, volume)
	}

	return volumes, nil
}

// AnalyzeSnapshots analyzes snapshot usage
func (c *Client) AnalyzeSnapshots(ctx context.Context) (*types.SnapshotAnalysis, error) {
	snapshots, err := c.GetSnapshots(ctx)
	if err != nil {
		return nil, err
	}

	analysis := &types.SnapshotAnalysis{
		TotalSnapshots: len(snapshots),
		SnapshotsByAge: make(map[string]int),
		Timestamp:      time.Now(),
	}

	var totalSize int64
	var totalAge time.Duration
	now := time.Now()

	for _, snapshot := range snapshots {
		totalSize += snapshot.UsedSize
		age := now.Sub(snapshot.CreationTime)
		totalAge += age

		// Categorize by age
		days := int(age.Hours() / 24)
		switch {
		case days <= 1:
			analysis.SnapshotsByAge["last_24h"]++
		case days <= 7:
			analysis.SnapshotsByAge["last_week"]++
		case days <= 30:
			analysis.SnapshotsByAge["last_month"]++
		default:
			analysis.SnapshotsByAge["older"]++
		}

		// Track large snapshots (>1GB)
		if snapshot.UsedSize > 1024*1024*1024 {
			analysis.LargeSnapshots = append(analysis.LargeSnapshots, snapshot)
		}
	}

	analysis.TotalSize = totalSize
	if len(snapshots) > 0 {
		analysis.AverageAge = totalAge / time.Duration(len(snapshots))
	}

	// Generate recommendations
	if analysis.TotalSnapshots > 100 {
		analysis.Recommendations = append(analysis.Recommendations, "Consider implementing automated snapshot cleanup")
	}

	if totalSize > 100*1024*1024*1024 { // 100GB
		analysis.Recommendations = append(analysis.Recommendations, "Snapshot storage usage is high, review retention policies")
	}

	if analysis.SnapshotsByAge["older"] > 50 {
		analysis.Recommendations = append(analysis.Recommendations, "Many old snapshots detected, consider cleanup")
	}

	return analysis, nil
}

// ValidateConfiguration validates the TrueNAS configuration
func (c *Client) ValidateConfiguration(ctx context.Context) (*types.ValidationResult, error) {
	result := &types.ValidationResult{
		Valid:     true,
		Timestamp: time.Now(),
	}

	// Test connection
	connectionCheck := types.HealthCheck{
		Name:      "TrueNAS Connection",
		Timestamp: time.Now(),
	}

	if err := c.TestConnection(ctx); err != nil {
		connectionCheck.Healthy = false
		connectionCheck.Message = fmt.Sprintf("Failed to connect: %v", err)
		result.Valid = false
		result.Errors = append(result.Errors, connectionCheck.Message)
	} else {
		connectionCheck.Healthy = true
		connectionCheck.Message = "Successfully connected to TrueNAS API"
	}

	result.Checks = append(result.Checks, connectionCheck)

	// Check pools
	poolCheck := types.HealthCheck{
		Name:      "Storage Pools",
		Timestamp: time.Now(),
	}

	pools, err := c.GetPools(ctx)
	if err != nil {
		poolCheck.Healthy = false
		poolCheck.Message = fmt.Sprintf("Failed to get pools: %v", err)
		result.Warnings = append(result.Warnings, poolCheck.Message)
	} else {
		poolCheck.Healthy = true
		poolCheck.Message = fmt.Sprintf("Found %d storage pools", len(pools))
		
		// Check pool health
		for _, pool := range pools {
			if !pool.Healthy {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Pool %s is not healthy: %s", pool.Name, pool.Status))
			}
		}
	}

	result.Checks = append(result.Checks, poolCheck)

	return result, nil
}

// Helper functions

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	if val, ok := m[key].(int); ok {
		return float64(val)
	}
	if val, ok := m[key].(string); ok {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return 0
}