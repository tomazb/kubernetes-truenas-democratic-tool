package truenas

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/yourusername/kubernetes-truenas-democratic-tool/pkg/logging"
)

// Client represents a TrueNAS client
type Client interface {
	ListVolumes(ctx context.Context) ([]Volume, error)
	ListSnapshots(ctx context.Context) ([]Snapshot, error)
	ListPools(ctx context.Context) ([]Pool, error)
	GetSystemInfo(ctx context.Context) (*SystemInfo, error)
	TestConnection(ctx context.Context) error
}

// client implements the Client interface
type client struct {
	httpClient *resty.Client
	baseURL    string
	logger     *logging.Logger
}

// Config holds TrueNAS client configuration
type Config struct {
	URL      string
	Username string
	Password string
	Timeout  time.Duration
}

// Volume represents a TrueNAS volume
type Volume struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Path        string            `json:"path"`
	Type        string            `json:"type"`
	Used        int64             `json:"used"`
	Available   int64             `json:"available"`
	Properties  map[string]string `json:"properties"`
	CreatedAt   time.Time         `json:"created_at"`
}

// Snapshot represents a TrueNAS snapshot
type Snapshot struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Dataset   string            `json:"dataset"`
	Used      int64             `json:"used"`
	CreatedAt time.Time         `json:"created_at"`
	Properties map[string]string `json:"properties"`
}

// Pool represents a TrueNAS storage pool
type Pool struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	Size      int64   `json:"size"`
	Used      int64   `json:"used"`
	Available int64   `json:"available"`
	Health    string  `json:"health"`
}

// SystemInfo represents TrueNAS system information
type SystemInfo struct {
	Version   string `json:"version"`
	Hostname  string `json:"hostname"`
	Uptime    string `json:"uptime"`
	LoadAvg   string `json:"loadavg"`
	Memory    Memory `json:"memory"`
}

// Memory represents system memory information
type Memory struct {
	Total     int64 `json:"total"`
	Available int64 `json:"available"`
	Used      int64 `json:"used"`
	Percent   float64 `json:"percent"`
}

// NewClient creates a new TrueNAS client
func NewClient(config Config) (Client, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("TrueNAS URL is required")
	}

	if config.Username == "" {
		return nil, fmt.Errorf("TrueNAS username is required")
	}

	if config.Password == "" {
		return nil, fmt.Errorf("TrueNAS password is required")
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	httpClient := resty.New().
		SetBaseURL(config.URL).
		SetBasicAuth(config.Username, config.Password).
		SetTimeout(timeout).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")

	// Disable SSL verification for self-signed certificates
	// In production, you should properly configure SSL certificates
	httpClient.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	// Initialize logger
	logger, err := logging.NewLogger(logging.Config{
		Level:     "info",
		Format:    "json",
		Component: "truenas-client",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return &client{
		httpClient: httpClient,
		baseURL:    config.URL,
		logger:     logger,
	}, nil
}

// ListVolumes lists all volumes/datasets with enhanced metadata
func (c *client) ListVolumes(ctx context.Context) ([]Volume, error) {
	start := time.Now()
	
	// TrueNAS API response structure
	var datasets []struct {
		ID         string            `json:"id"`
		Name       string            `json:"name"`
		Pool       string            `json:"pool"`
		Type       string            `json:"type"`
		Used       struct {
			Parsed int64 `json:"parsed"`
		} `json:"used"`
		Available struct {
			Parsed int64 `json:"parsed"`
		} `json:"available"`
		Mountpoint  string            `json:"mountpoint"`
		Properties  map[string]interface{} `json:"properties"`
		Children    []interface{}     `json:"children"`
	}

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetResult(&datasets).
		Get("/api/v2.0/pool/dataset")

	if err != nil {
		c.logger.WithError(err).Error("Failed to list TrueNAS datasets")
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		c.logger.WithFields(map[string]interface{}{
			"status_code": resp.StatusCode(),
			"response":    resp.String(),
		}).Error("TrueNAS API returned error status")
		return nil, fmt.Errorf("TrueNAS API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	// Transform TrueNAS dataset response to our Volume format
	var result []Volume
	for _, dataset := range datasets {
		// Convert properties map to string map
		props := make(map[string]string)
		for k, v := range dataset.Properties {
			if str, ok := v.(string); ok {
				props[k] = str
			} else {
				props[k] = fmt.Sprintf("%v", v)
			}
		}

		volume := Volume{
			ID:         dataset.ID,
			Name:       dataset.Name,
			Path:       dataset.Mountpoint,
			Type:       dataset.Type,
			Used:       dataset.Used.Parsed,
			Available:  dataset.Available.Parsed,
			Properties: props,
			CreatedAt:  time.Now(), // TrueNAS doesn't provide creation time in this API
		}

		// Add pool information if available
		if dataset.Pool != "" {
			volume.Properties["pool"] = dataset.Pool
		}

		result = append(result, volume)
	}

	duration := time.Since(start)
	c.logger.LogTrueNASOperation("list", "datasets", len(result), duration.Milliseconds())

	return result, nil
}

// ListSnapshots lists all snapshots with enhanced metadata
func (c *client) ListSnapshots(ctx context.Context) ([]Snapshot, error) {
	start := time.Now()
	
	// TrueNAS API response structure for snapshots
	var snapshotData []struct {
		ID         string            `json:"id"`
		Name       string            `json:"name"`
		Dataset    string            `json:"dataset"`
		Used       struct {
			Parsed int64 `json:"parsed"`
		} `json:"used"`
		Created    struct {
			Parsed int64 `json:"parsed"`
		} `json:"created"`
		Properties map[string]interface{} `json:"properties"`
	}

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetResult(&snapshotData).
		Get("/api/v2.0/zfs/snapshot")

	if err != nil {
		c.logger.WithError(err).Error("Failed to list TrueNAS snapshots")
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		c.logger.WithFields(map[string]interface{}{
			"status_code": resp.StatusCode(),
			"response":    resp.String(),
		}).Error("TrueNAS API returned error status for snapshots")
		return nil, fmt.Errorf("TrueNAS API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	// Transform TrueNAS snapshot response to our Snapshot format
	var result []Snapshot
	for _, snap := range snapshotData {
		// Convert properties map to string map
		props := make(map[string]string)
		for k, v := range snap.Properties {
			if str, ok := v.(string); ok {
				props[k] = str
			} else {
				props[k] = fmt.Sprintf("%v", v)
			}
		}

		snapshot := Snapshot{
			ID:         snap.ID,
			Name:       snap.Name,
			Dataset:    snap.Dataset,
			Used:       snap.Used.Parsed,
			CreatedAt:  time.Unix(snap.Created.Parsed, 0),
			Properties: props,
		}

		result = append(result, snapshot)
	}

	duration := time.Since(start)
	c.logger.LogTrueNASOperation("list", "snapshots", len(result), duration.Milliseconds())

	return result, nil
}

// ListPools lists all storage pools
func (c *client) ListPools(ctx context.Context) ([]Pool, error) {
	var pools []Pool

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetResult(&pools).
		Get("/api/v2.0/pool")

	if err != nil {
		return nil, fmt.Errorf("failed to list pools: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("TrueNAS API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	return pools, nil
}

// GetSystemInfo gets system information
func (c *client) GetSystemInfo(ctx context.Context) (*SystemInfo, error) {
	var sysInfo SystemInfo

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetResult(&sysInfo).
		Get("/api/v2.0/system/info")

	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("TrueNAS API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	return &sysInfo, nil
}

// TestConnection tests the connection to TrueNAS
func (c *client) TestConnection(ctx context.Context) error {
	resp, err := c.httpClient.R().
		SetContext(ctx).
		Get("/api/v2.0/system/info")

	if err != nil {
		return fmt.Errorf("failed to connect to TrueNAS: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("TrueNAS API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	return nil
}