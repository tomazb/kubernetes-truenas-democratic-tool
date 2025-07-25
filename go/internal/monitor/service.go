package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"

	"github.com/tomazb/kubernetes-truenas-democratic-tool/internal/config"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/internal/k8s"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/internal/truenas"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/internal/types"
)

// Service represents the monitoring service
type Service struct {
	config       *config.Config
	k8sClient    *k8s.Client
	truenasClient *truenas.Client
	logger       *zap.Logger
	metrics      *Metrics
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// Metrics holds all Prometheus metrics
type Metrics struct {
	// Snapshot metrics
	SnapshotsTotal        *prometheus.GaugeVec
	SnapshotsSizeBytes    *prometheus.GaugeVec
	OrphanedSnapshotsTotal *prometheus.GaugeVec

	// Storage pool metrics
	StoragePoolSizeBytes         *prometheus.GaugeVec
	StoragePoolUtilizationPercent *prometheus.GaugeVec
	StoragePoolHealth            *prometheus.GaugeVec

	// Volume metrics
	PersistentVolumesTotal      *prometheus.GaugeVec
	PersistentVolumeClaimsTotal *prometheus.GaugeVec
	OrphanedPVsTotal           *prometheus.GaugeVec
	OrphanedPVCsTotal          *prometheus.GaugeVec

	// Efficiency metrics
	ThinProvisioningRatio   *prometheus.GaugeVec
	CompressionRatio        *prometheus.GaugeVec
	SnapshotOverheadPercent *prometheus.GaugeVec

	// System health metrics
	SystemConnectivity *prometheus.GaugeVec
	CSIDriverPodsTotal *prometheus.GaugeVec
	ActiveAlertsTotal  *prometheus.GaugeVec

	// Operation metrics
	MonitoringRunsTotal     *prometheus.CounterVec
	MonitoringDurationSeconds *prometheus.HistogramVec
}

// NewService creates a new monitoring service
func NewService(cfg *config.Config, logger *zap.Logger) (*Service, error) {
	ctx, cancel := context.WithCancel(context.Background())

	service := &Service{
		config: cfg,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize Kubernetes client
	k8sClient, err := k8s.NewClient(&cfg.OpenShift)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}
	service.k8sClient = k8sClient

	// Initialize TrueNAS client
	truenasClient, err := truenas.NewClient(&cfg.TrueNAS)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create TrueNAS client: %w", err)
	}
	service.truenasClient = truenasClient

	// Initialize metrics
	service.metrics = service.initMetrics()

	return service, nil
}

// initMetrics initializes all Prometheus metrics
func (s *Service) initMetrics() *Metrics {
	return &Metrics{
		SnapshotsTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "truenas_snapshots_total",
				Help: "Total number of snapshots",
			},
			[]string{"system", "pool", "dataset"},
		),
		SnapshotsSizeBytes: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "truenas_snapshots_size_bytes",
				Help: "Total size of snapshots in bytes",
			},
			[]string{"system", "pool", "dataset"},
		),
		OrphanedSnapshotsTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "truenas_orphaned_snapshots_total",
				Help: "Number of orphaned snapshots",
			},
			[]string{"system", "type"},
		),
		StoragePoolSizeBytes: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "truenas_storage_pool_size_bytes",
				Help: "Storage pool size metrics",
			},
			[]string{"pool_name", "metric_type"},
		),
		StoragePoolUtilizationPercent: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "truenas_storage_pool_utilization_percent",
				Help: "Pool utilization percentage",
			},
			[]string{"pool_name"},
		),
		StoragePoolHealth: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "truenas_storage_pool_health",
				Help: "Pool health (1=healthy, 0=unhealthy)",
			},
			[]string{"pool_name", "status"},
		),
		PersistentVolumesTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "truenas_persistent_volumes_total",
				Help: "Total PVs",
			},
			[]string{"namespace", "storage_class", "status"},
		),
		PersistentVolumeClaimsTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "truenas_persistent_volume_claims_total",
				Help: "Total PVCs",
			},
			[]string{"namespace", "storage_class", "status"},
		),
		OrphanedPVsTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "truenas_orphaned_pvs_total",
				Help: "Orphaned PVs count",
			},
			[]string{"namespace"},
		),
		OrphanedPVCsTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "truenas_orphaned_pvcs_total",
				Help: "Orphaned PVCs count",
			},
			[]string{"namespace"},
		),
		ThinProvisioningRatio: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "truenas_thin_provisioning_ratio",
				Help: "Ratio of allocated to used storage",
			},
			[]string{"pool_name"},
		),
		CompressionRatio: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "truenas_compression_ratio",
				Help: "Data compression ratio",
			},
			[]string{"pool_name"},
		),
		SnapshotOverheadPercent: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "truenas_snapshot_overhead_percent",
				Help: "Snapshot storage overhead %",
			},
			[]string{"pool_name"},
		),
		SystemConnectivity: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "truenas_system_connectivity",
				Help: "Connection status (1=connected, 0=disconnected)",
			},
			[]string{"system"},
		),
		CSIDriverPodsTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "truenas_csi_driver_pods_total",
				Help: "CSI driver pod counts",
			},
			[]string{"driver_name", "status"},
		),
		ActiveAlertsTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "truenas_active_alerts_total",
				Help: "Active alert counts",
			},
			[]string{"level", "category"},
		),
		MonitoringRunsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "truenas_monitoring_runs_total",
				Help: "Total monitoring runs",
			},
			[]string{"status"},
		),
		MonitoringDurationSeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "truenas_monitoring_duration_seconds",
				Help: "Duration of monitoring operations",
				Buckets: []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0},
			},
			[]string{"operation"},
		),
	}
}

// Start starts the monitoring service
func (s *Service) Start() error {
	s.logger.Info("Starting monitoring service")

	// Test connections
	if err := s.testConnections(); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	// Start monitoring loop
	s.wg.Add(1)
	go s.monitoringLoop()

	s.logger.Info("Monitoring service started")
	return nil
}

// Stop stops the monitoring service
func (s *Service) Stop() error {
	s.logger.Info("Stopping monitoring service")
	s.cancel()
	s.wg.Wait()
	s.logger.Info("Monitoring service stopped")
	return nil
}

// testConnections tests connections to Kubernetes and TrueNAS
func (s *Service) testConnections() error {
	// Test Kubernetes connection
	if err := s.k8sClient.TestConnection(s.ctx); err != nil {
		s.metrics.SystemConnectivity.WithLabelValues("kubernetes").Set(0)
		return fmt.Errorf("kubernetes connection failed: %w", err)
	}
	s.metrics.SystemConnectivity.WithLabelValues("kubernetes").Set(1)

	// Test TrueNAS connection
	if err := s.truenasClient.TestConnection(s.ctx); err != nil {
		s.metrics.SystemConnectivity.WithLabelValues("truenas").Set(0)
		return fmt.Errorf("truenas connection failed: %w", err)
	}
	s.metrics.SystemConnectivity.WithLabelValues("truenas").Set(1)

	return nil
}

// monitoringLoop runs the main monitoring loop
func (s *Service) monitoringLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.Monitor.OrphanCheckInterval)
	defer ticker.Stop()

	// Run initial check
	s.runMonitoringCheck()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.runMonitoringCheck()
		}
	}
}

// runMonitoringCheck runs a complete monitoring check
func (s *Service) runMonitoringCheck() {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		s.metrics.MonitoringDurationSeconds.WithLabelValues("full_check").Observe(duration.Seconds())
	}()

	s.logger.Info("Running monitoring check")

	// Check orphaned resources
	if err := s.checkOrphanedResources(); err != nil {
		s.logger.Error("Failed to check orphaned resources", zap.Error(err))
		s.metrics.MonitoringRunsTotal.WithLabelValues("error").Inc()
		return
	}

	// Check storage usage
	if err := s.checkStorageUsage(); err != nil {
		s.logger.Error("Failed to check storage usage", zap.Error(err))
		s.metrics.MonitoringRunsTotal.WithLabelValues("error").Inc()
		return
	}

	// Check CSI driver health
	if err := s.checkCSIDriverHealth(); err != nil {
		s.logger.Error("Failed to check CSI driver health", zap.Error(err))
		s.metrics.MonitoringRunsTotal.WithLabelValues("error").Inc()
		return
	}

	// Check snapshot health
	if err := s.checkSnapshotHealth(); err != nil {
		s.logger.Error("Failed to check snapshot health", zap.Error(err))
		s.metrics.MonitoringRunsTotal.WithLabelValues("error").Inc()
		return
	}

	s.metrics.MonitoringRunsTotal.WithLabelValues("success").Inc()
	s.logger.Info("Monitoring check completed successfully")
}

// checkOrphanedResources checks for orphaned resources
func (s *Service) checkOrphanedResources() error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		s.metrics.MonitoringDurationSeconds.WithLabelValues("orphan_check").Observe(duration.Seconds())
	}()

	// Check orphaned PVs
	orphanedPVs, err := s.k8sClient.FindOrphanedPVs(s.ctx, s.config.Monitor.OrphanThreshold)
	if err != nil {
		return fmt.Errorf("failed to find orphaned PVs: %w", err)
	}

	// Check orphaned PVCs
	orphanedPVCs, err := s.k8sClient.FindOrphanedPVCs(s.ctx, s.config.Monitor.OrphanThreshold)
	if err != nil {
		return fmt.Errorf("failed to find orphaned PVCs: %w", err)
	}

	// Update metrics
	s.metrics.OrphanedPVsTotal.WithLabelValues("").Set(float64(len(orphanedPVs)))
	s.metrics.OrphanedPVCsTotal.WithLabelValues("").Set(float64(len(orphanedPVCs)))

	s.logger.Info("Orphaned resources check completed",
		zap.Int("orphaned_pvs", len(orphanedPVs)),
		zap.Int("orphaned_pvcs", len(orphanedPVCs)),
	)

	return nil
}

// checkStorageUsage checks storage usage
func (s *Service) checkStorageUsage() error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		s.metrics.MonitoringDurationSeconds.WithLabelValues("storage_check").Observe(duration.Seconds())
	}()

	// Get storage pools
	pools, err := s.truenasClient.GetPools(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to get storage pools: %w", err)
	}

	// Update pool metrics
	for _, pool := range pools {
		s.metrics.StoragePoolSizeBytes.WithLabelValues(pool.Name, "total").Set(float64(pool.TotalSize))
		s.metrics.StoragePoolSizeBytes.WithLabelValues(pool.Name, "used").Set(float64(pool.UsedSize))
		s.metrics.StoragePoolSizeBytes.WithLabelValues(pool.Name, "free").Set(float64(pool.FreeSize))

		utilization := float64(pool.UsedSize) / float64(pool.TotalSize) * 100
		s.metrics.StoragePoolUtilizationPercent.WithLabelValues(pool.Name).Set(utilization)

		healthValue := 0.0
		if pool.Healthy {
			healthValue = 1.0
		}
		s.metrics.StoragePoolHealth.WithLabelValues(pool.Name, pool.Status).Set(healthValue)
	}

	s.logger.Info("Storage usage check completed", zap.Int("pools", len(pools)))
	return nil
}

// checkCSIDriverHealth checks CSI driver health
func (s *Service) checkCSIDriverHealth() error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		s.metrics.MonitoringDurationSeconds.WithLabelValues("csi_health_check").Observe(duration.Seconds())
	}()

	driverInfo, err := s.k8sClient.CheckCSIDriverHealth(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to check CSI driver health: %w", err)
	}

	// Count healthy and unhealthy pods
	healthyPods := 0
	unhealthyPods := 0

	for _, pod := range driverInfo.Pods {
		if pod.Ready && pod.Phase == "Running" {
			healthyPods++
		} else {
			unhealthyPods++
		}
	}

	s.metrics.CSIDriverPodsTotal.WithLabelValues(driverInfo.Name, "healthy").Set(float64(healthyPods))
	s.metrics.CSIDriverPodsTotal.WithLabelValues(driverInfo.Name, "unhealthy").Set(float64(unhealthyPods))

	s.logger.Info("CSI driver health check completed",
		zap.String("driver", driverInfo.Name),
		zap.Int("healthy_pods", healthyPods),
		zap.Int("unhealthy_pods", unhealthyPods),
	)

	return nil
}

// checkSnapshotHealth checks snapshot health
func (s *Service) checkSnapshotHealth() error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		s.metrics.MonitoringDurationSeconds.WithLabelValues("snapshot_check").Observe(duration.Seconds())
	}()

	// Analyze snapshots
	analysis, err := s.truenasClient.AnalyzeSnapshots(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to analyze snapshots: %w", err)
	}

	// Update snapshot metrics
	s.metrics.SnapshotsTotal.WithLabelValues("truenas", "", "").Set(float64(analysis.TotalSnapshots))
	s.metrics.SnapshotsSizeBytes.WithLabelValues("truenas", "", "").Set(float64(analysis.TotalSize))

	s.logger.Info("Snapshot health check completed",
		zap.Int("total_snapshots", analysis.TotalSnapshots),
		zap.Int64("total_size", analysis.TotalSize),
	)

	return nil
}

// GetStatus returns the current status of the monitoring service
func (s *Service) GetStatus() (*types.MonitoringResult, error) {
	result := &types.MonitoringResult{
		Timestamp: time.Now(),
	}

	// Get orphaned resources
	orphanedPVs, err := s.k8sClient.FindOrphanedPVs(s.ctx, s.config.Monitor.OrphanThreshold)
	if err != nil {
		return nil, fmt.Errorf("failed to get orphaned PVs: %w", err)
	}
	result.OrphanedPVs = orphanedPVs

	orphanedPVCs, err := s.k8sClient.FindOrphanedPVCs(s.ctx, s.config.Monitor.OrphanThreshold)
	if err != nil {
		return nil, fmt.Errorf("failed to get orphaned PVCs: %w", err)
	}
	result.OrphanedPVCs = orphanedPVCs

	// Get storage usage
	pools, err := s.truenasClient.GetPools(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage pools: %w", err)
	}
	result.StorageUsage = pools

	// Get CSI driver health
	driverInfo, err := s.k8sClient.CheckCSIDriverHealth(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSI driver health: %w", err)
	}
	result.CSIDriverHealth = *driverInfo

	return result, nil
}

// ValidateConfiguration validates the configuration
func (s *Service) ValidateConfiguration() (*types.ValidationResult, error) {
	// Validate Kubernetes configuration
	k8sResult, err := s.k8sClient.ValidateConfiguration(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to validate Kubernetes configuration: %w", err)
	}

	// Validate TrueNAS configuration
	truenasResult, err := s.truenasClient.ValidateConfiguration(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to validate TrueNAS configuration: %w", err)
	}

	// Combine results
	result := &types.ValidationResult{
		Valid:     k8sResult.Valid && truenasResult.Valid,
		Timestamp: time.Now(),
	}

	result.Checks = append(result.Checks, k8sResult.Checks...)
	result.Checks = append(result.Checks, truenasResult.Checks...)
	result.Errors = append(result.Errors, k8sResult.Errors...)
	result.Errors = append(result.Errors, truenasResult.Errors...)
	result.Warnings = append(result.Warnings, k8sResult.Warnings...)
	result.Warnings = append(result.Warnings, truenasResult.Warnings...)

	return result, nil
}