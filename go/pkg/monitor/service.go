package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/k8s"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/logging"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/metrics"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/orphan"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/truenas"
)

// Service represents the monitoring service
type Service struct {
	k8sClient       k8s.Client
	truenasClient   truenas.Client
	metricsExporter *metrics.Exporter
	logger          *logging.Logger
	scanInterval    time.Duration
	orphanDetector  *orphan.Detector
	
	// Internal state
	mu             sync.RWMutex
	running        bool
	stopChan       chan struct{}
	wg             sync.WaitGroup
	lastScanResult *ScanResult
}

// Config holds the service configuration
type Config struct {
	K8sClient       k8s.Client
	TruenasClient   truenas.Client
	MetricsExporter *metrics.Exporter
	Logger          *logging.Logger
	ScanInterval    time.Duration
}

// OrphanedResource represents an orphaned resource
type OrphanedResource struct {
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Age         time.Duration     `json:"age"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Reason      string            `json:"reason"`
}

// ScanResult represents the result of a monitoring scan
type ScanResult struct {
	Timestamp        time.Time           `json:"timestamp"`
	OrphanedPVs      []OrphanedResource  `json:"orphaned_pvs"`
	OrphanedPVCs     []OrphanedResource  `json:"orphaned_pvcs"`
	OrphanedSnapshots []OrphanedResource `json:"orphaned_snapshots"`
	TotalPVs         int                 `json:"total_pvs"`
	TotalPVCs        int                 `json:"total_pvcs"`
	TotalSnapshots   int                 `json:"total_snapshots"`
	ScanDuration     time.Duration       `json:"scan_duration"`
}

// NewService creates a new monitoring service
func NewService(config Config) (*Service, error) {
	// Initialize orphan detector
	orphanDetector, err := orphan.NewDetector(
		config.K8sClient,
		config.TruenasClient,
		orphan.Config{
			AgeThreshold:      24 * time.Hour,
			SnapshotRetention: 30 * 24 * time.Hour,
			DryRun:            false,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create orphan detector: %w", err)
	}

	return &Service{
		k8sClient:       config.K8sClient,
		truenasClient:   config.TruenasClient,
		metricsExporter: config.MetricsExporter,
		logger:          config.Logger,
		scanInterval:    config.ScanInterval,
		orphanDetector:  orphanDetector,
		stopChan:        make(chan struct{}),
	}, nil
}

// Start begins the monitoring service
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("service is already running")
	}

	s.logger.WithComponent("monitor-service").Info("Starting monitoring service")

	// Start metrics exporter
	if err := s.metricsExporter.Start(); err != nil {
		return fmt.Errorf("failed to start metrics exporter: %w", err)
	}

	s.running = true

	// Start monitoring goroutine
	s.wg.Add(1)
	go s.monitorLoop(ctx)

	return nil
}

// Stop gracefully stops the monitoring service
func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.logger.Info("Stopping monitoring service")

	close(s.stopChan)
	s.running = false

	// Wait for goroutines to finish
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("Monitoring service stopped gracefully")
	case <-ctx.Done():
		s.logger.Warn("Monitoring service stop timed out")
		return ctx.Err()
	}

	// Stop metrics exporter
	return s.metricsExporter.Stop()
}

// GetLastScanResult returns the most recent scan result
func (s *Service) GetLastScanResult() *ScanResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastScanResult
}

// monitorLoop runs the main monitoring loop
func (s *Service) monitorLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.scanInterval)
	defer ticker.Stop()

	// Run initial scan
	s.performScan(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Monitor loop stopped due to context cancellation")
			return
		case <-s.stopChan:
			s.logger.Info("Monitor loop stopped")
			return
		case <-ticker.C:
			s.performScan(ctx)
		}
	}
}

// performScan executes a complete monitoring scan using the orphan detector
func (s *Service) performScan(ctx context.Context) {
	start := time.Now()
	s.logger.Debug("Starting monitoring scan")

	// Use the comprehensive orphan detector
	detectionResult, err := s.orphanDetector.DetectOrphanedResources(ctx, "")
	if err != nil {
		s.logger.WithError(err).Error("Failed to detect orphaned resources")
		return
	}

	// Convert detection result to scan result format
	result := &ScanResult{
		Timestamp:         detectionResult.Timestamp,
		OrphanedPVs:       s.convertOrphanedResources(detectionResult.OrphanedPVs),
		OrphanedPVCs:      s.convertOrphanedResources(detectionResult.OrphanedPVCs),
		OrphanedSnapshots: s.convertOrphanedResources(detectionResult.OrphanedSnapshots),
		TotalPVs:          detectionResult.TotalPVs,
		TotalPVCs:         detectionResult.TotalPVCs,
		TotalSnapshots:    detectionResult.TotalSnapshots,
		ScanDuration:      detectionResult.ScanDuration,
	}

	// Store the latest scan result
	s.mu.Lock()
	s.lastScanResult = result
	s.mu.Unlock()

	// Update metrics
	s.updateMetrics(result)

	// Log scan results using structured logging
	s.logger.LogScanResult(
		len(result.OrphanedPVs),
		len(result.OrphanedPVCs),
		len(result.OrphanedSnapshots),
		result.ScanDuration,
	)
}

// Note: The old placeholder scanning methods have been removed since we now use
// the comprehensive orphan detector which provides much more sophisticated
// detection algorithms with proper correlation between K8s and TrueNAS resources.

// convertOrphanedResources converts orphan detector results to monitor service format
func (s *Service) convertOrphanedResources(orphanResources []orphan.OrphanedResource) []OrphanedResource {
	var result []OrphanedResource
	for _, orphan := range orphanResources {
		result = append(result, OrphanedResource{
			Type:        orphan.Type,
			Name:        orphan.Name,
			Namespace:   orphan.Namespace,
			Age:         orphan.Age,
			Labels:      orphan.Labels,
			Annotations: orphan.Annotations,
			Reason:      orphan.Reason,
		})
	}
	return result
}

// updateMetrics updates Prometheus metrics with scan results
func (s *Service) updateMetrics(result *ScanResult) {
	s.metricsExporter.SetOrphanedPVsCount(float64(len(result.OrphanedPVs)))
	s.metricsExporter.SetOrphanedPVCsCount(float64(len(result.OrphanedPVCs)))
	s.metricsExporter.SetOrphanedSnapshotsCount(float64(len(result.OrphanedSnapshots)))
	s.metricsExporter.SetScanDuration(result.ScanDuration.Seconds())
	s.metricsExporter.SetTotalPVs(float64(result.TotalPVs))
	s.metricsExporter.SetTotalPVCs(float64(result.TotalPVCs))
	s.metricsExporter.SetTotalSnapshots(float64(result.TotalSnapshots))
	s.metricsExporter.SetLastScanTimestamp(result.Timestamp)
}