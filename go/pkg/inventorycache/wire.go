package inventorycache

import (
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/config"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/metrics"
)

// NewFromConfig builds an inventory cache from application config and optional metrics.
func NewFromConfig(cfg config.PerformanceConfig, exporter *metrics.Exporter) *Cache {
	var stats StatsRecorder
	if exporter != nil {
		stats = exporter.RecordInventoryCacheAccess
	}

	return NewCache(Config{
		Enabled: cfg.Cache.Enabled,
		TTL:     cfg.Cache.TTL,
		MaxSize: cfg.Cache.MaxSize,
		Stats:   stats,
	})
}
