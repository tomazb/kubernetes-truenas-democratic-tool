package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

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
	totalPVs               prometheus.Gauge
	totalPVCs              prometheus.Gauge
	totalSnapshots         prometheus.Gauge
	storageEfficiency      prometheus.Gauge
	lastScanTimestamp      prometheus.Gauge
}

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
		w.Write([]byte("OK"))
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