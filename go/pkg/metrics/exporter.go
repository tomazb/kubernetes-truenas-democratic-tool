package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Exporter handles Prometheus metrics export
type Exporter struct {
	server   *http.Server
	registry *prometheus.Registry
	logger   *zap.Logger

	// Metrics
	orphanedPVsCount       prometheus.Gauge
	orphanedPVCsCount      prometheus.Gauge
	orphanedSnapshotsCount prometheus.Gauge
	scanDuration           prometheus.Gauge
	scanDurationHist       prometheus.Histogram
	listDurationHist       *prometheus.HistogramVec
	cacheHits              *prometheus.CounterVec
	cacheMisses            *prometheus.CounterVec
	totalPVs               prometheus.Gauge
	totalPVCs              prometheus.Gauge
	totalSnapshots         prometheus.Gauge
	storageEfficiency      prometheus.Gauge
	lastScanTimestamp      prometheus.Gauge
}

var scanDurationBuckets = []float64{0.5, 1, 2, 5, 10, 30, 60, 120}

var listDurationBuckets = []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30}

// Config holds metrics exporter configuration
type Config struct {
	Enabled bool
	Port    int
	Path    string
}

// NewExporter creates a new metrics exporter
func NewExporter(config Config) *Exporter {
	registry := prometheus.NewRegistry()
	
	// Create metrics
	orphanedPVsCount := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "truenas_monitor_orphaned_pvs_total",
		Help: "Total number of orphaned persistent volumes",
	})

	orphanedPVCsCount := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "truenas_monitor_orphaned_pvcs_total",
		Help: "Total number of orphaned persistent volume claims",
	})

	orphanedSnapshotsCount := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "truenas_monitor_orphaned_snapshots_total",
		Help: "Total number of orphaned volume snapshots",
	})

	scanDuration := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "truenas_monitor_scan_duration_seconds",
		Help: "Duration of the last monitoring scan in seconds",
	})

	scanDurationHist := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "truenas_monitor_scan_duration_histogram_seconds",
		Help:    "Distribution of monitoring scan durations in seconds",
		Buckets: scanDurationBuckets,
	})

	listDurationHist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "truenas_monitor_list_duration_seconds",
		Help:    "Duration of inventory list operations during orphan detection",
		Buckets: listDurationBuckets,
	}, []string{"phase"})

	cacheHits := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "truenas_monitor_inventory_cache_hits_total",
		Help: "Inventory cache hits by operation",
	}, []string{"operation"})

	cacheMisses := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "truenas_monitor_inventory_cache_misses_total",
		Help: "Inventory cache misses by operation",
	}, []string{"operation"})

	totalPVs := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "truenas_monitor_pvs_total",
		Help: "Total number of persistent volumes",
	})

	totalPVCs := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "truenas_monitor_pvcs_total",
		Help: "Total number of persistent volume claims",
	})

	totalSnapshots := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "truenas_monitor_snapshots_total",
		Help: "Total number of volume snapshots",
	})

	storageEfficiency := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "truenas_monitor_storage_efficiency_percent",
		Help: "Storage efficiency percentage from thin provisioning",
	})

	lastScanTimestamp := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "truenas_monitor_last_scan_timestamp",
		Help: "Timestamp of the last successful scan",
	})

	// Register metrics
	registry.MustRegister(
		orphanedPVsCount,
		orphanedPVCsCount,
		orphanedSnapshotsCount,
		scanDuration,
		scanDurationHist,
		listDurationHist,
		cacheHits,
		cacheMisses,
		totalPVs,
		totalPVCs,
		totalSnapshots,
		storageEfficiency,
		lastScanTimestamp,
	)

	// Create HTTP server
	mux := http.NewServeMux()
	mux.Handle(config.Path, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	logger, _ := zap.NewProduction()

	return &Exporter{
		server:                 server,
		registry:               registry,
		logger:                 logger,
		orphanedPVsCount:       orphanedPVsCount,
		orphanedPVCsCount:      orphanedPVCsCount,
		orphanedSnapshotsCount: orphanedSnapshotsCount,
		scanDuration:           scanDuration,
		scanDurationHist:       scanDurationHist,
		listDurationHist:       listDurationHist,
		cacheHits:              cacheHits,
		cacheMisses:            cacheMisses,
		totalPVs:               totalPVs,
		totalPVCs:              totalPVCs,
		totalSnapshots:         totalSnapshots,
		storageEfficiency:      storageEfficiency,
		lastScanTimestamp:      lastScanTimestamp,
	}
}

// Start starts the metrics HTTP server
func (e *Exporter) Start() error {
	e.logger.Info("Starting metrics server", zap.String("addr", e.server.Addr))

	go func() {
		if err := e.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			e.logger.Error("Metrics server error", zap.Error(err))
		}
	}()

	return nil
}

// Stop gracefully stops the metrics server
func (e *Exporter) Stop() error {
	e.logger.Info("Stopping metrics server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return e.server.Shutdown(ctx)
}

// SetOrphanedPVsCount sets the orphaned PVs count metric
func (e *Exporter) SetOrphanedPVsCount(count float64) {
	e.orphanedPVsCount.Set(count)
}

// SetOrphanedPVCsCount sets the orphaned PVCs count metric
func (e *Exporter) SetOrphanedPVCsCount(count float64) {
	e.orphanedPVCsCount.Set(count)
}

// SetOrphanedSnapshotsCount sets the orphaned snapshots count metric
func (e *Exporter) SetOrphanedSnapshotsCount(count float64) {
	e.orphanedSnapshotsCount.Set(count)
}

// SetScanDuration sets the scan duration metric
func (e *Exporter) SetScanDuration(duration float64) {
	e.scanDuration.Set(duration)
}

// ObserveScanDuration records a scan duration in the histogram
func (e *Exporter) ObserveScanDuration(duration float64) {
	e.scanDurationHist.Observe(duration)
}

// ObserveListPhaseDuration records a list operation duration for a detection phase
func (e *Exporter) ObserveListPhaseDuration(phase string, duration float64) {
	e.listDurationHist.WithLabelValues(phase).Observe(duration)
}

// RecordInventoryCacheAccess increments cache hit or miss counters.
func (e *Exporter) RecordInventoryCacheAccess(operation string, hit bool) {
	if hit {
		e.cacheHits.WithLabelValues(operation).Inc()
		return
	}
	e.cacheMisses.WithLabelValues(operation).Inc()
}

// SetTotalPVs sets the total PVs metric
func (e *Exporter) SetTotalPVs(count float64) {
	e.totalPVs.Set(count)
}

// SetTotalPVCs sets the total PVCs metric
func (e *Exporter) SetTotalPVCs(count float64) {
	e.totalPVCs.Set(count)
}

// SetTotalSnapshots sets the total snapshots metric
func (e *Exporter) SetTotalSnapshots(count float64) {
	e.totalSnapshots.Set(count)
}

// SetStorageEfficiency sets the storage efficiency metric
func (e *Exporter) SetStorageEfficiency(efficiency float64) {
	e.storageEfficiency.Set(efficiency)
}

// SetLastScanTimestamp sets the last scan timestamp metric
func (e *Exporter) SetLastScanTimestamp(timestamp time.Time) {
	e.lastScanTimestamp.Set(float64(timestamp.Unix()))
}

// GatherForTest exposes registered metrics for unit tests.
func (e *Exporter) GatherForTest() ([]*dto.MetricFamily, error) {
	return e.registry.Gather()
}