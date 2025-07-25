package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	OpenShift OpenShiftConfig `mapstructure:"openshift"`
	TrueNAS   TrueNASConfig   `mapstructure:"truenas"`
	Monitor   MonitorConfig   `mapstructure:"monitoring"`
	API       APIConfig       `mapstructure:"api"`
	Metrics   MetricsConfig   `mapstructure:"metrics"`
	Logging   LoggingConfig   `mapstructure:"logging"`
}

// OpenShiftConfig represents Kubernetes/OpenShift configuration
type OpenShiftConfig struct {
	Kubeconfig       string   `mapstructure:"kubeconfig"`
	Context          string   `mapstructure:"context"`
	Namespace        string   `mapstructure:"namespace"`
	MonitorNamespace []string `mapstructure:"monitor_namespaces"`
	StorageClass     string   `mapstructure:"storage_class"`
	CSIDriver        string   `mapstructure:"csi_driver"`
}

// TrueNASConfig represents TrueNAS configuration
type TrueNASConfig struct {
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	APIKey   string `mapstructure:"api_key"`
	Insecure bool   `mapstructure:"insecure"`
	Timeout  int    `mapstructure:"timeout"`
	Pool     string `mapstructure:"pool"`
}

// MonitorConfig represents monitoring configuration
type MonitorConfig struct {
	OrphanCheckInterval time.Duration     `mapstructure:"orphan_check_interval"`
	OrphanThreshold     time.Duration     `mapstructure:"orphan_threshold"`
	Snapshot            SnapshotConfig    `mapstructure:"snapshot"`
	Storage             StorageConfig     `mapstructure:"storage"`
	Workers             int               `mapstructure:"workers"`
	BatchSize           int               `mapstructure:"batch_size"`
}

// SnapshotConfig represents snapshot monitoring configuration
type SnapshotConfig struct {
	MaxAge        time.Duration `mapstructure:"max_age"`
	MaxCount      int           `mapstructure:"max_count"`
	CheckInterval time.Duration `mapstructure:"check_interval"`
}

// StorageConfig represents storage monitoring configuration
type StorageConfig struct {
	PoolWarningThreshold    int     `mapstructure:"pool_warning_threshold"`
	PoolCriticalThreshold   int     `mapstructure:"pool_critical_threshold"`
	VolumeWarningThreshold  int     `mapstructure:"volume_warning_threshold"`
	VolumeCriticalThreshold int     `mapstructure:"volume_critical_threshold"`
	MaxOvercommitRatio      float64 `mapstructure:"max_overcommit_ratio"`
}

// APIConfig represents API server configuration
type APIConfig struct {
	Listen string    `mapstructure:"listen"`
	TLS    TLSConfig `mapstructure:"tls"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"`
	Path    string `mapstructure:"path"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig() (*Config, error) {
	// Set defaults
	viper.SetDefault("openshift.namespace", "democratic-csi")
	viper.SetDefault("openshift.csi_driver", "org.democratic-csi.nfs")
	viper.SetDefault("truenas.timeout", 30)
	viper.SetDefault("monitoring.orphan_check_interval", "1h")
	viper.SetDefault("monitoring.orphan_threshold", "24h")
	viper.SetDefault("monitoring.snapshot.max_age", "30d")
	viper.SetDefault("monitoring.snapshot.max_count", 50)
	viper.SetDefault("monitoring.snapshot.check_interval", "6h")
	viper.SetDefault("monitoring.storage.pool_warning_threshold", 80)
	viper.SetDefault("monitoring.storage.pool_critical_threshold", 90)
	viper.SetDefault("monitoring.storage.volume_warning_threshold", 85)
	viper.SetDefault("monitoring.storage.volume_critical_threshold", 95)
	viper.SetDefault("monitoring.storage.max_overcommit_ratio", 2.0)
	viper.SetDefault("monitoring.workers", 10)
	viper.SetDefault("monitoring.batch_size", 100)
	viper.SetDefault("api.listen", ":8080")
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.port", 9090)
	viper.SetDefault("metrics.path", "/metrics")
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")

	// Environment variable substitution
	viper.AutomaticEnv()
	viper.SetEnvPrefix("TRUENAS_MONITOR")

	// Expand environment variables in config values
	for _, key := range []string{
		"truenas.username", "truenas.password", "truenas.api_key",
		"openshift.kubeconfig",
	} {
		if value := viper.GetString(key); value != "" {
			viper.Set(key, os.ExpandEnv(value))
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	if config.TrueNAS.URL == "" {
		return fmt.Errorf("truenas.url is required")
	}

	if config.TrueNAS.Username == "" && config.TrueNAS.APIKey == "" {
		return fmt.Errorf("either truenas.username or truenas.api_key is required")
	}

	if config.TrueNAS.Username != "" && config.TrueNAS.Password == "" {
		return fmt.Errorf("truenas.password is required when using username authentication")
	}

	if config.Monitor.OrphanCheckInterval <= 0 {
		return fmt.Errorf("monitoring.orphan_check_interval must be positive")
	}

	if config.Monitor.OrphanThreshold <= 0 {
		return fmt.Errorf("monitoring.orphan_threshold must be positive")
	}

	if config.Monitor.Workers <= 0 {
		return fmt.Errorf("monitoring.workers must be positive")
	}

	if config.Monitor.BatchSize <= 0 {
		return fmt.Errorf("monitoring.batch_size must be positive")
	}

	return nil
}