package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Kubernetes  KubernetesConfig  `yaml:"kubernetes"`
	TrueNAS     TrueNASConfig     `yaml:"truenas"`
	Monitor     MonitorConfig     `yaml:"monitor"`
	Metrics     MetricsConfig     `yaml:"metrics"`
	Performance PerformanceConfig `yaml:"performance"`
	Alerts      AlertsConfig      `yaml:"alerts"`
	Logging     LoggingConfig     `yaml:"logging"`
	Security    SecurityConfig    `yaml:"security"`
}

// KubernetesConfig holds Kubernetes connection settings
type KubernetesConfig struct {
	Kubeconfig string `yaml:"kubeconfig"`
	Namespace  string `yaml:"namespace"`
	InCluster  bool   `yaml:"in_cluster"`
}

// TrueNASConfig holds TrueNAS connection settings
type TrueNASConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Timeout  string `yaml:"timeout"`
	Insecure bool   `yaml:"insecure"`
	CAFile   string `yaml:"ca_file"`
}

const (
	ReconcileModePoll  = "poll"
	ReconcileModeWatch = "watch"
)

// MonitorConfig holds monitoring settings
type MonitorConfig struct {
	ReconcileMode       string        `yaml:"reconcile_mode"`
	ScanInterval        time.Duration `yaml:"scan_interval"`
	Debounce            time.Duration `yaml:"debounce"`
	TruenasPollInterval time.Duration `yaml:"truenas_poll_interval"`
	OrphanThreshold     time.Duration `yaml:"orphan_threshold"`
	SnapshotRetention   time.Duration `yaml:"snapshot_retention"`
}

// MetricsConfig holds metrics export settings
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
}

// PerformanceConfig holds performance tuning settings.
type PerformanceConfig struct {
	Cache   CacheConfig  `yaml:"cache"`
	Budgets BudgetConfig `yaml:"budgets"`
}

// CacheConfig holds in-process inventory cache settings.
type CacheConfig struct {
	Enabled bool          `yaml:"enabled"`
	TTL     time.Duration `yaml:"ttl"`
	MaxSize int           `yaml:"max_size"`
}

// BudgetConfig holds runtime performance budget settings.
type BudgetConfig struct {
	ScanDurationSeconds float64 `yaml:"scan_duration_seconds"`
	ListPhaseP95Seconds float64 `yaml:"list_phase_p95_seconds"`
	MemoryRSSMB         int     `yaml:"memory_rss_mb"`
}

// AlertsConfig holds alerting settings
type AlertsConfig struct {
	Slack SlackConfig `yaml:"slack"`
}

// SlackConfig holds Slack webhook settings
type SlackConfig struct {
	Webhook string `yaml:"webhook"`
	Channel string `yaml:"channel"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level       string `yaml:"level"`
	Development bool   `yaml:"development"`
	Encoding    string `yaml:"encoding"`
}

// SecurityConfig holds security settings
type SecurityConfig struct {
	TLSMinVersion  string        `yaml:"tls_min_version"`
	RequireAuth    bool          `yaml:"require_auth"`
	AllowedOrigins []string      `yaml:"allowed_origins"`
	RateLimitRPS   int           `yaml:"rate_limit_rps"`
	SessionTimeout time.Duration `yaml:"session_timeout"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	// Set defaults
	config := &Config{
		Kubernetes: KubernetesConfig{
			Namespace: "democratic-csi",
			InCluster: true,
		},
		TrueNAS: TrueNASConfig{
			Timeout: "30s",
		},
		Monitor: MonitorConfig{
			ReconcileMode:       ReconcileModePoll,
			ScanInterval:        5 * time.Minute,
			Debounce:            30 * time.Second,
			TruenasPollInterval: 5 * time.Minute,
			OrphanThreshold:     24 * time.Hour,
			SnapshotRetention:   30 * 24 * time.Hour,
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Port:    8080,
			Path:    "/metrics",
		},
		Performance: PerformanceConfig{
			Cache: CacheConfig{
				Enabled: true,
				TTL:     5 * time.Minute,
				MaxSize: 1000,
			},
			Budgets: BudgetConfig{
				ScanDurationSeconds: 300,
				ListPhaseP95Seconds: 2,
				MemoryRSSMB:         2048,
			},
		},
		Logging: LoggingConfig{
			Level:       "info",
			Development: false,
			Encoding:    "json",
		},
		Security: SecurityConfig{
			TLSMinVersion:  "1.3",
			RequireAuth:    true,
			AllowedOrigins: []string{"*"},
			RateLimitRPS:   100,
			SessionTimeout: 24 * time.Hour,
		},
	}

	fileExists := false
	// Read file if it exists
	if _, err := os.Stat(path); err == nil {
		fileExists = true
		// #nosec G304 -- config path is an explicit operator-provided input.
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		// Expand environment variables with enhanced substitution
		expanded := expandEnvVars(string(data))

		if err := yaml.Unmarshal([]byte(expanded), config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}

		if err := config.validatePerformanceCache(); err != nil {
			return nil, fmt.Errorf("invalid configuration: %w", err)
		}
	}

	config.applyPerformanceDefaults()
	config.applyMonitorDefaults()

	// Validate configuration only if file exists
	if fileExists {
		if err := config.validate(); err != nil {
			return nil, fmt.Errorf("invalid configuration: %w", err)
		}
	}

	return config, nil
}

func (c *Config) validatePerformanceCache() error {
	if c.Performance.Cache.TTL != 0 && c.Performance.Cache.TTL < time.Second {
		return fmt.Errorf("performance.cache.ttl must be at least 1 second")
	}
	if c.Performance.Cache.MaxSize < 0 {
		return fmt.Errorf("performance.cache.max_size must be at least 1")
	}
	return nil
}

func (c *Config) applyPerformanceDefaults() {
	if c.Performance.Cache.TTL == 0 {
		c.Performance.Cache.TTL = 5 * time.Minute
	}
	if c.Performance.Cache.MaxSize == 0 {
		c.Performance.Cache.MaxSize = 1000
	}
	if c.Performance.Budgets.ScanDurationSeconds == 0 {
		c.Performance.Budgets.ScanDurationSeconds = 300
	}
	if c.Performance.Budgets.ListPhaseP95Seconds == 0 {
		c.Performance.Budgets.ListPhaseP95Seconds = 2
	}
	if c.Performance.Budgets.MemoryRSSMB == 0 {
		c.Performance.Budgets.MemoryRSSMB = 2048
	}
}

func (c *Config) applyMonitorDefaults() {
	if c.Monitor.ReconcileMode == "" {
		c.Monitor.ReconcileMode = ReconcileModePoll
	}
	if c.Monitor.Debounce == 0 {
		c.Monitor.Debounce = 30 * time.Second
	}
	if c.Monitor.TruenasPollInterval == 0 {
		c.Monitor.TruenasPollInterval = 5 * time.Minute
	}
}

// expandEnvVars expands environment variables in the format ${VAR_NAME} or ${VAR_NAME:default}
func expandEnvVars(input string) string {
	// Regex to match ${VAR_NAME} or ${VAR_NAME:default_value}
	re := regexp.MustCompile(`\$\{([^}:]+)(?::([^}]*))?\}`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		// Extract variable name and default value
		parts := re.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		varName := parts[1]
		defaultValue := ""
		if len(parts) > 2 {
			defaultValue = parts[2]
		}

		// Get environment variable value
		if value := os.Getenv(varName); value != "" {
			return value
		}

		return defaultValue
	})
}

// validate checks if the configuration is valid
func (c *Config) validate() error {
	// TrueNAS validation
	if c.TrueNAS.URL == "" {
		return fmt.Errorf("truenas.url is required")
	}

	if c.TrueNAS.Username == "" {
		return fmt.Errorf("truenas.username is required")
	}

	if c.TrueNAS.Password == "" {
		return fmt.Errorf("truenas.password is required")
	}

	// Validate TrueNAS timeout
	if _, err := time.ParseDuration(c.TrueNAS.Timeout); err != nil {
		return fmt.Errorf("invalid truenas.timeout format: %w", err)
	}

	if c.TrueNAS.CAFile != "" {
		info, err := os.Stat(c.TrueNAS.CAFile)
		if err != nil {
			return fmt.Errorf("truenas.ca_file %q: %w", c.TrueNAS.CAFile, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("truenas.ca_file %q must be a regular file", c.TrueNAS.CAFile)
		}
	}

	// Monitor validation
	mode := strings.ToLower(strings.TrimSpace(c.Monitor.ReconcileMode))
	if mode != ReconcileModePoll && mode != ReconcileModeWatch {
		return fmt.Errorf("monitor.reconcile_mode must be %q or %q", ReconcileModePoll, ReconcileModeWatch)
	}
	c.Monitor.ReconcileMode = mode

	if mode == ReconcileModePoll {
		if c.Monitor.ScanInterval < time.Minute {
			return fmt.Errorf("monitor.scan_interval must be at least 1 minute")
		}
		if c.Monitor.ScanInterval > 24*time.Hour {
			return fmt.Errorf("monitor.scan_interval must not exceed 24 hours")
		}
	}

	if mode == ReconcileModeWatch {
		if c.Monitor.Debounce < time.Second {
			return fmt.Errorf("monitor.debounce must be at least 1 second")
		}
		if c.Monitor.TruenasPollInterval < time.Minute {
			return fmt.Errorf("monitor.truenas_poll_interval must be at least 1 minute")
		}
	}

	if c.Monitor.OrphanThreshold < time.Hour {
		return fmt.Errorf("monitor.orphan_threshold must be at least 1 hour")
	}

	// Metrics validation
	if c.Metrics.Port < 1 || c.Metrics.Port > 65535 {
		return fmt.Errorf("metrics.port must be between 1 and 65535")
	}

	if c.Metrics.Path == "" {
		return fmt.Errorf("metrics.path cannot be empty")
	}

	if c.Performance.Cache.TTL < time.Second {
		return fmt.Errorf("performance.cache.ttl must be at least 1 second")
	}

	if c.Performance.Cache.Enabled && c.Performance.Cache.MaxSize < 1 {
		return fmt.Errorf("performance.cache.max_size must be at least 1")
	}
	if c.Performance.Budgets.ScanDurationSeconds <= 0 {
		return fmt.Errorf("performance.budgets.scan_duration_seconds must be greater than 0")
	}
	if c.Performance.Budgets.ListPhaseP95Seconds <= 0 {
		return fmt.Errorf("performance.budgets.list_phase_p95_seconds must be greater than 0")
	}
	if c.Performance.Budgets.MemoryRSSMB <= 0 {
		return fmt.Errorf("performance.budgets.memory_rss_mb must be greater than 0")
	}

	// Logging validation
	validLogLevels := []string{"debug", "info", "warn", "error", "fatal"}
	if !contains(validLogLevels, strings.ToLower(c.Logging.Level)) {
		return fmt.Errorf("logging.level must be one of: %s", strings.Join(validLogLevels, ", "))
	}

	validEncodings := []string{"json", "console"}
	if !contains(validEncodings, c.Logging.Encoding) {
		return fmt.Errorf("logging.encoding must be one of: %s", strings.Join(validEncodings, ", "))
	}

	// Security validation
	validTLSVersions := []string{"1.2", "1.3"}
	if !contains(validTLSVersions, c.Security.TLSMinVersion) {
		return fmt.Errorf("security.tls_min_version must be one of: %s", strings.Join(validTLSVersions, ", "))
	}

	if c.Security.RateLimitRPS < 1 || c.Security.RateLimitRPS > 10000 {
		return fmt.Errorf("security.rate_limit_rps must be between 1 and 10000")
	}

	if c.Security.SessionTimeout < time.Minute {
		return fmt.Errorf("security.session_timeout must be at least 1 minute")
	}

	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
