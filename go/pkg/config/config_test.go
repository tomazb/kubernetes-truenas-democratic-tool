package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name       string
		configYAML string
		envVars    map[string]string
		wantErr    bool
		validate   func(*testing.T, *Config)
	}{
		{
			name: "default config",
			configYAML: `
kubernetes:
  namespace: democratic-csi
truenas:
  url: https://truenas.example.com
  username: admin
  password: secret123
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "democratic-csi", cfg.Kubernetes.Namespace)
				assert.True(t, cfg.Kubernetes.InCluster)
				assert.Equal(t, "https://truenas.example.com", cfg.TrueNAS.URL)
				assert.Equal(t, "admin", cfg.TrueNAS.Username)
				assert.Equal(t, "secret123", cfg.TrueNAS.Password)
				assert.Equal(t, "30s", cfg.TrueNAS.Timeout)
				assert.Equal(t, 5*time.Minute, cfg.Monitor.ScanInterval)
				assert.Equal(t, 24*time.Hour, cfg.Monitor.OrphanThreshold)
				assert.True(t, cfg.Metrics.Enabled)
				assert.Equal(t, 8080, cfg.Metrics.Port)
				assert.Equal(t, "/metrics", cfg.Metrics.Path)
			},
		},
		{
			name: "custom config",
			configYAML: `
kubernetes:
  kubeconfig: /path/to/kubeconfig
  in_cluster: false
  namespace: custom-csi
truenas:
  url: https://custom.truenas.com
  username: customuser
  password: custompass
  timeout: 60s
monitor:
  scan_interval: 10m
  orphan_threshold: 48h
  snapshot_retention: 168h
metrics:
  enabled: false
  port: 9090
  path: /custom-metrics
alerts:
  slack:
    webhook: https://hooks.slack.com/test
    channel: "#alerts"
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "/path/to/kubeconfig", cfg.Kubernetes.Kubeconfig)
				assert.False(t, cfg.Kubernetes.InCluster)
				assert.Equal(t, "custom-csi", cfg.Kubernetes.Namespace)
				assert.Equal(t, "https://custom.truenas.com", cfg.TrueNAS.URL)
				assert.Equal(t, "customuser", cfg.TrueNAS.Username)
				assert.Equal(t, "custompass", cfg.TrueNAS.Password)
				assert.Equal(t, "60s", cfg.TrueNAS.Timeout)
				assert.Equal(t, 10*time.Minute, cfg.Monitor.ScanInterval)
				assert.Equal(t, 48*time.Hour, cfg.Monitor.OrphanThreshold)
				assert.Equal(t, 168*time.Hour, cfg.Monitor.SnapshotRetention)
				assert.False(t, cfg.Metrics.Enabled)
				assert.Equal(t, 9090, cfg.Metrics.Port)
				assert.Equal(t, "/custom-metrics", cfg.Metrics.Path)
				assert.Equal(t, "https://hooks.slack.com/test", cfg.Alerts.Slack.Webhook)
				assert.Equal(t, "#alerts", cfg.Alerts.Slack.Channel)
			},
		},
		{
			name: "environment variable substitution",
			configYAML: `
truenas:
  url: ${TRUENAS_URL}
  username: ${TRUENAS_USERNAME}
  password: ${TRUENAS_PASSWORD}
alerts:
  slack:
    webhook: ${SLACK_WEBHOOK}
`,
			envVars: map[string]string{
				"TRUENAS_URL":      "https://env.truenas.com",
				"TRUENAS_USERNAME": "envuser",
				"TRUENAS_PASSWORD": "envpass",
				"SLACK_WEBHOOK":    "https://hooks.slack.com/env",
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "https://env.truenas.com", cfg.TrueNAS.URL)
				assert.Equal(t, "envuser", cfg.TrueNAS.Username)
				assert.Equal(t, "envpass", cfg.TrueNAS.Password)
				assert.Equal(t, "https://hooks.slack.com/env", cfg.Alerts.Slack.Webhook)
			},
		},
		{
			name: "missing required fields",
			configYAML: `
kubernetes:
  namespace: test
# Missing truenas section
`,
			wantErr: true,
		},
		{
			name: "invalid scan interval",
			configYAML: `
truenas:
  url: https://truenas.example.com
  username: admin
  password: secret123
monitor:
  scan_interval: 30s  # Less than 1 minute
`,
			wantErr: true,
		},
		{
			name: "invalid timeout format",
			configYAML: `
truenas:
  url: https://truenas.example.com
  username: admin
  password: secret123
  timeout: invalid-duration
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Create temporary config file
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "config.yaml")
			err := os.WriteFile(configFile, []byte(tt.configYAML), 0644)
			require.NoError(t, err)

			// Load configuration
			cfg, err := Load(configFile)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}
		})
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	// Test loading non-existent file should use defaults
	cfg, err := Load("/non/existent/file.yaml")
	
	// Should not error, should use defaults
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	
	// Check defaults are applied
	assert.Equal(t, "democratic-csi", cfg.Kubernetes.Namespace)
	assert.True(t, cfg.Kubernetes.InCluster)
	assert.Equal(t, "30s", cfg.TrueNAS.Timeout)
	assert.Equal(t, 5*time.Minute, cfg.Monitor.ScanInterval)
	assert.Equal(t, 24*time.Hour, cfg.Monitor.OrphanThreshold)
	assert.True(t, cfg.Metrics.Enabled)
	assert.Equal(t, 8080, cfg.Metrics.Port)
	assert.Equal(t, "/metrics", cfg.Metrics.Path)
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				TrueNAS: TrueNASConfig{
					URL:      "https://truenas.example.com",
					Username: "admin",
					Password: "secret123",
					Timeout:  "30s",
				},
				Monitor: MonitorConfig{
					ScanInterval:    5 * time.Minute,
					OrphanThreshold: 24 * time.Hour,
				},
				Metrics: MetricsConfig{
					Port: 8080,
					Path: "/metrics",
				},
				Logging: LoggingConfig{
					Level:    "info",
					Encoding: "json",
				},
				Security: SecurityConfig{
					TLSMinVersion:  "1.3",
					RateLimitRPS:   100,
					AllowedOrigins: []string{"*"},
					SessionTimeout: 24 * time.Hour,
				},
			},
			wantErr: false,
		},
		{
			name: "missing TrueNAS URL",
			config: &Config{
				TrueNAS: TrueNASConfig{
					Username: "admin",
					Password: "secret123",
				},
			},
			wantErr: true,
			errMsg:  "truenas.url is required",
		},
		{
			name: "missing TrueNAS username",
			config: &Config{
				TrueNAS: TrueNASConfig{
					URL:      "https://truenas.example.com",
					Password: "secret123",
				},
			},
			wantErr: true,
			errMsg:  "truenas.username is required",
		},
		{
			name: "missing TrueNAS password",
			config: &Config{
				TrueNAS: TrueNASConfig{
					URL:      "https://truenas.example.com",
					Username: "admin",
				},
			},
			wantErr: true,
			errMsg:  "truenas.password is required",
		},
		{
			name: "scan interval too short",
			config: &Config{
				TrueNAS: TrueNASConfig{
					URL:      "https://truenas.example.com",
					Username: "admin",
					Password: "secret123",
					Timeout:  "30s",
				},
				Monitor: MonitorConfig{
					ScanInterval:    30 * time.Second,
					OrphanThreshold: 24 * time.Hour,
				},
				Metrics: MetricsConfig{
					Port: 8080,
					Path: "/metrics",
				},
			},
			wantErr: true,
			errMsg:  "monitor.scan_interval must be at least 1 minute",
		},
		{
			name: "scan interval too long",
			config: &Config{
				TrueNAS: TrueNASConfig{
					URL:      "https://truenas.example.com",
					Username: "admin",
					Password: "secret123",
					Timeout:  "30s",
				},
				Monitor: MonitorConfig{
					ScanInterval:    25 * time.Hour,
					OrphanThreshold: 24 * time.Hour,
				},
				Metrics: MetricsConfig{
					Port: 8080,
					Path: "/metrics",
				},
			},
			wantErr: true,
			errMsg:  "monitor.scan_interval must not exceed 24 hours",
		},
		{
			name: "orphan threshold too short",
			config: &Config{
				TrueNAS: TrueNASConfig{
					URL:      "https://truenas.example.com",
					Username: "admin",
					Password: "secret123",
					Timeout:  "30s",
				},
				Monitor: MonitorConfig{
					ScanInterval:    5 * time.Minute,
					OrphanThreshold: 30 * time.Minute,
				},
				Metrics: MetricsConfig{
					Port: 8080,
					Path: "/metrics",
				},
			},
			wantErr: true,
			errMsg:  "monitor.orphan_threshold must be at least 1 hour",
		},
		{
			name: "invalid metrics port",
			config: &Config{
				TrueNAS: TrueNASConfig{
					URL:      "https://truenas.example.com",
					Username: "admin",
					Password: "secret123",
					Timeout:  "30s",
				},
				Monitor: MonitorConfig{
					ScanInterval:    5 * time.Minute,
					OrphanThreshold: 24 * time.Hour,
				},
				Metrics: MetricsConfig{
					Port: 70000,
					Path: "/metrics",
				},
			},
			wantErr: true,
			errMsg:  "metrics.port must be between 1 and 65535",
		},
		{
			name: "empty metrics path",
			config: &Config{
				TrueNAS: TrueNASConfig{
					URL:      "https://truenas.example.com",
					Username: "admin",
					Password: "secret123",
					Timeout:  "30s",
				},
				Monitor: MonitorConfig{
					ScanInterval:    5 * time.Minute,
					OrphanThreshold: 24 * time.Hour,
				},
				Metrics: MetricsConfig{
					Port: 8080,
					Path: "",
				},
			},
			wantErr: true,
			errMsg:  "metrics.path cannot be empty",
		},
		{
			name: "invalid timeout format",
			config: &Config{
				TrueNAS: TrueNASConfig{
					URL:      "https://truenas.example.com",
					Username: "admin",
					Password: "secret123",
					Timeout:  "invalid",
				},
				Monitor: MonitorConfig{
					ScanInterval:    5 * time.Minute,
					OrphanThreshold: 24 * time.Hour,
				},
				Metrics: MetricsConfig{
					Port: 8080,
					Path: "/metrics",
				},
			},
			wantErr: true,
			errMsg:  "invalid truenas.timeout format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	// Test that defaults are properly set
	cfg, err := Load("/non/existent/file.yaml")
	require.NoError(t, err)

	// Kubernetes defaults
	assert.Equal(t, "democratic-csi", cfg.Kubernetes.Namespace)
	assert.True(t, cfg.Kubernetes.InCluster)
	assert.Empty(t, cfg.Kubernetes.Kubeconfig)

	// TrueNAS defaults
	assert.Equal(t, "30s", cfg.TrueNAS.Timeout)

	// Monitor defaults
	assert.Equal(t, 5*time.Minute, cfg.Monitor.ScanInterval)
	assert.Equal(t, 24*time.Hour, cfg.Monitor.OrphanThreshold)
	assert.Equal(t, 30*24*time.Hour, cfg.Monitor.SnapshotRetention)

	// Metrics defaults
	assert.True(t, cfg.Metrics.Enabled)
	assert.Equal(t, 8080, cfg.Metrics.Port)
	assert.Equal(t, "/metrics", cfg.Metrics.Path)
}

func TestEnvironmentVariableExpansion(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_URL", "https://test.example.com")
	os.Setenv("TEST_USER", "testuser")
	os.Setenv("TEST_PASS", "testpass")
	defer func() {
		os.Unsetenv("TEST_URL")
		os.Unsetenv("TEST_USER")
		os.Unsetenv("TEST_PASS")
	}()

	configYAML := `
truenas:
  url: ${TEST_URL}
  username: ${TEST_USER}
  password: ${TEST_PASS}
`

	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configFile, []byte(configYAML), 0644)
	require.NoError(t, err)

	// Load configuration
	cfg, err := Load(configFile)
	require.NoError(t, err)

	// Verify environment variables were expanded
	assert.Equal(t, "https://test.example.com", cfg.TrueNAS.URL)
	assert.Equal(t, "testuser", cfg.TrueNAS.Username)
	assert.Equal(t, "testpass", cfg.TrueNAS.Password)
}

func TestInvalidYAML(t *testing.T) {
	invalidYAML := `
truenas:
  url: https://truenas.example.com
  username: admin
  password: secret123
invalid yaml structure
  missing colon
`

	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configFile, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	// Load configuration should fail
	cfg, err := Load(configFile)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "failed to parse config file")
}