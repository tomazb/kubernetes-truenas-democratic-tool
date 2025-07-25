package orphan

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"go.uber.org/zap"

	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/k8s"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/logging"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/truenas"
)

// Detector handles orphaned resource detection
type Detector struct {
	k8sClient     k8s.Client
	truenasClient truenas.Client
	logger        *logging.Logger
	config        Config
}

// Config holds detector configuration
type Config struct {
	AgeThreshold      time.Duration
	SnapshotRetention time.Duration
	DryRun            bool
}

// OrphanedResource represents an orphaned resource
type OrphanedResource struct {
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Age         time.Duration     `json:"age"`
	Size        string            `json:"size,omitempty"`
	Reason      string            `json:"reason"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	VolumeHandle string           `json:"volume_handle,omitempty"`
	StorageClass string           `json:"storage_class,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// DetectionResult holds the results of orphan detection
type DetectionResult struct {
	Timestamp         time.Time           `json:"timestamp"`
	OrphanedPVs       []OrphanedResource  `json:"orphaned_pvs"`
	OrphanedPVCs      []OrphanedResource  `json:"orphaned_pvcs"`
	OrphanedSnapshots []OrphanedResource  `json:"orphaned_snapshots"`
	TotalPVs          int                 `json:"total_pvs"`
	TotalPVCs         int                 `json:"total_pvcs"`
	TotalSnapshots    int                 `json:"total_snapshots"`
	ScanDuration      time.Duration       `json:"scan_duration"`
}

// NewDetector creates a new orphan detector
func NewDetector(k8sClient k8s.Client, truenasClient truenas.Client, config Config) (*Detector, error) {
	logger, err := logging.NewLogger(logging.Config{
		Level:    "info",
		Encoding: "json",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Set default values
	if config.AgeThreshold == 0 {
		config.AgeThreshold = 24 * time.Hour
	}
	if config.SnapshotRetention == 0 {
		config.SnapshotRetention = 30 * 24 * time.Hour
	}

	return &Detector{
		k8sClient:     k8sClient,
		truenasClient: truenasClient,
		logger:        logger,
		config:        config,
	}, nil
}

// DetectOrphanedResources performs comprehensive orphan detection
func (d *Detector) DetectOrphanedResources(ctx context.Context, namespace string) (*DetectionResult, error) {
	start := time.Now()
	d.logger.Info("Starting orphaned resource detection",
		zap.String("namespace", namespace),
		zap.String("age_threshold", d.config.AgeThreshold.String()),
		zap.Bool("dry_run", d.config.DryRun),
	)

	result := &DetectionResult{
		Timestamp: start,
	}

	// Detect orphaned PVs
	orphanedPVs, totalPVs, err := d.detectOrphanedPVs(ctx)
	if err != nil {
		d.logger.WithError(err).Error("Failed to detect orphaned PVs")
		return nil, fmt.Errorf("failed to detect orphaned PVs: %w", err)
	}
	result.OrphanedPVs = orphanedPVs
	result.TotalPVs = totalPVs

	// Detect orphaned PVCs
	orphanedPVCs, totalPVCs, err := d.detectOrphanedPVCs(ctx, namespace)
	if err != nil {
		d.logger.WithError(err).Error("Failed to detect orphaned PVCs")
		return nil, fmt.Errorf("failed to detect orphaned PVCs: %w", err)
	}
	result.OrphanedPVCs = orphanedPVCs
	result.TotalPVCs = totalPVCs

	// Detect orphaned snapshots
	orphanedSnapshots, totalSnapshots, err := d.detectOrphanedSnapshots(ctx, namespace)
	if err != nil {
		d.logger.WithError(err).Error("Failed to detect orphaned snapshots")
		return nil, fmt.Errorf("failed to detect orphaned snapshots: %w", err)
	}
	result.OrphanedSnapshots = orphanedSnapshots
	result.TotalSnapshots = totalSnapshots

	result.ScanDuration = time.Since(start)

	d.logger.Info("Orphaned resource detection completed",
		zap.Int("orphaned_pvs", len(result.OrphanedPVs)),
		zap.Int("orphaned_pvcs", len(result.OrphanedPVCs)),
		zap.Int("orphaned_snapshots", len(result.OrphanedSnapshots)),
		zap.Int("total_pvs", result.TotalPVs),
		zap.Int("total_pvcs", result.TotalPVCs),
		zap.Int("total_snapshots", result.TotalSnapshots),
		zap.Int64("scan_duration_ms", result.ScanDuration.Milliseconds()),
	)

	return result, nil
}

// detectOrphanedPVs identifies PVs without corresponding TrueNAS volumes
func (d *Detector) detectOrphanedPVs(ctx context.Context) ([]OrphanedResource, int, error) {
	// Get all democratic-csi PVs from Kubernetes
	pvs, err := d.k8sClient.ListDemocraticCSIPersistentVolumes(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list democratic-csi PVs: %w", err)
	}

	// Get all volumes from TrueNAS
	truenasVolumes, err := d.truenasClient.ListVolumes(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list TrueNAS volumes: %w", err)
	}

	var orphaned []OrphanedResource
	threshold := time.Now().Add(-d.config.AgeThreshold)

	for _, pv := range pvs {
		// Check if PV is old enough to be considered for orphan detection
		if pv.CreationTimestamp.Time.After(threshold) {
			continue
		}

		// Check if PV has corresponding TrueNAS volume
		if !d.hasCorrespondingTrueNASVolume(pv, truenasVolumes) {
			orphan := OrphanedResource{
				Type:         "PersistentVolume",
				Name:         pv.Name,
				Age:          time.Since(pv.CreationTimestamp.Time),
				Reason:       "No corresponding TrueNAS volume found",
				Labels:       pv.Labels,
				Annotations:  pv.Annotations,
				CreatedAt:    pv.CreationTimestamp.Time,
			}

			// Extract additional information
			if pv.Spec.Capacity != nil {
				if storage, ok := pv.Spec.Capacity[corev1.ResourceStorage]; ok {
					orphan.Size = storage.String()
				}
			}

			if pv.Spec.StorageClassName != "" {
				orphan.StorageClass = pv.Spec.StorageClassName
			}

			if pv.Spec.CSI != nil {
				orphan.VolumeHandle = pv.Spec.CSI.VolumeHandle
			}

			orphaned = append(orphaned, orphan)
		}
	}

	d.logger.Info("PV orphan detection completed",
		zap.Int("total_democratic_csi_pvs", len(pvs)),
		zap.Int("orphaned_pvs", len(orphaned)),
		zap.String("age_threshold", d.config.AgeThreshold.String()),
	)

	return orphaned, len(pvs), nil
}

// detectOrphanedPVCs identifies unbound PVCs older than threshold
func (d *Detector) detectOrphanedPVCs(ctx context.Context, namespace string) ([]OrphanedResource, int, error) {
	// Get unbound PVCs
	unboundPVCs, err := d.k8sClient.ListUnboundPersistentVolumeClaims(ctx, namespace)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list unbound PVCs: %w", err)
	}

	// Get total PVCs for reporting
	allPVCs, err := d.k8sClient.ListPersistentVolumeClaims(ctx, namespace)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list all PVCs: %w", err)
	}

	var orphaned []OrphanedResource
	threshold := time.Now().Add(-d.config.AgeThreshold)

	for _, pvc := range unboundPVCs {
		// Check if PVC is old enough to be considered orphaned
		if pvc.CreationTimestamp.Time.Before(threshold) {
			orphan := OrphanedResource{
				Type:        "PersistentVolumeClaim",
				Name:        pvc.Name,
				Namespace:   pvc.Namespace,
				Age:         time.Since(pvc.CreationTimestamp.Time),
				Reason:      fmt.Sprintf("Unbound for %v", time.Since(pvc.CreationTimestamp.Time)),
				Labels:      pvc.Labels,
				Annotations: pvc.Annotations,
				CreatedAt:   pvc.CreationTimestamp.Time,
			}

			// Extract additional information
			if pvc.Spec.Resources.Requests != nil {
				if storage, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
					orphan.Size = storage.String()
				}
			}

			if pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName != "" {
				orphan.StorageClass = *pvc.Spec.StorageClassName
			}

			orphaned = append(orphaned, orphan)
		}
	}

	d.logger.Info("PVC orphan detection completed",
		zap.String("namespace", namespace),
		zap.Int("total_pvcs", len(allPVCs)),
		zap.Int("unbound_pvcs", len(unboundPVCs)),
		zap.Int("orphaned_pvcs", len(orphaned)),
		zap.String("age_threshold", d.config.AgeThreshold.String()),
	)

	return orphaned, len(allPVCs), nil
}

// detectOrphanedSnapshots identifies snapshots without corresponding resources
func (d *Detector) detectOrphanedSnapshots(ctx context.Context, namespace string) ([]OrphanedResource, int, error) {
	// Get all volume snapshots from Kubernetes
	k8sSnapshots, err := d.k8sClient.ListVolumeSnapshots(ctx, namespace)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list Kubernetes snapshots: %w", err)
	}

	// Get all snapshots from TrueNAS
	truenasSnapshots, err := d.truenasClient.ListSnapshots(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list TrueNAS snapshots: %w", err)
	}

	var orphaned []OrphanedResource
	threshold := time.Now().Add(-d.config.AgeThreshold)

	// Check for K8s snapshots without corresponding TrueNAS snapshots
	for _, snapshot := range k8sSnapshots {
		if snapshot.CreationTimestamp.Time.Before(threshold) {
			if !d.hasCorrespondingTrueNASSnapshot(snapshot, truenasSnapshots) {
				orphan := OrphanedResource{
					Type:        "VolumeSnapshot",
					Name:        snapshot.Name,
					Namespace:   snapshot.Namespace,
					Age:         time.Since(snapshot.CreationTimestamp.Time),
					Reason:      "No corresponding TrueNAS snapshot found",
					Labels:      snapshot.Labels,
					Annotations: snapshot.Annotations,
					CreatedAt:   snapshot.CreationTimestamp.Time,
				}

				orphaned = append(orphaned, orphan)
			}
		}
	}

	// Check for old TrueNAS snapshots that might be orphaned
	retentionThreshold := time.Now().Add(-d.config.SnapshotRetention)
	for _, truenasSnapshot := range truenasSnapshots {
		if truenasSnapshot.CreatedAt.Before(retentionThreshold) {
			if !d.hasCorrespondingK8sSnapshot(truenasSnapshot, k8sSnapshots) {
				orphan := OrphanedResource{
					Type:      "TrueNASSnapshot",
					Name:      truenasSnapshot.Name,
					Age:       time.Since(truenasSnapshot.CreatedAt),
					Reason:    "Old TrueNAS snapshot without corresponding VolumeSnapshot",
					Size:      fmt.Sprintf("%d bytes", truenasSnapshot.Used),
					CreatedAt: truenasSnapshot.CreatedAt,
				}

				orphaned = append(orphaned, orphan)
			}
		}
	}

	d.logger.Info("Snapshot orphan detection completed",
		zap.String("namespace", namespace),
		zap.Int("k8s_snapshots", len(k8sSnapshots)),
		zap.Int("truenas_snapshots", len(truenasSnapshots)),
		zap.Int("orphaned_snapshots", len(orphaned)),
		zap.String("age_threshold", d.config.AgeThreshold.String()),
		zap.String("retention_threshold", d.config.SnapshotRetention.String()),
	)

	return orphaned, len(k8sSnapshots), nil
}

// hasCorrespondingTrueNASVolume checks if a PV has a corresponding TrueNAS volume
func (d *Detector) hasCorrespondingTrueNASVolume(pv corev1.PersistentVolume, truenasVolumes []truenas.Volume) bool {
	if pv.Spec.CSI == nil {
		return false
	}

	volumeHandle := pv.Spec.CSI.VolumeHandle
	if volumeHandle == "" {
		return false
	}

	// Extract dataset name from volume handle
	// Democratic-CSI typically uses formats like:
	// - pool/dataset/volume-name
	// - pool/dataset@snapshot-name
	// - iqn.2005-10.org.freenas.ctl:volume-name
	datasetName := d.extractDatasetFromVolumeHandle(volumeHandle)

	for _, volume := range truenasVolumes {
		// Check various matching strategies
		if d.volumeMatches(volume, volumeHandle, datasetName) {
			d.logger.Debug("Found matching TrueNAS volume for PV",
				zap.String("pv_name", pv.Name),
				zap.String("volume_handle", volumeHandle),
				zap.String("dataset_name", datasetName),
				zap.String("truenas_volume", volume.Name),
			)
			return true
		}
	}

	return false
}

// hasCorrespondingTrueNASSnapshot checks if a K8s snapshot has a corresponding TrueNAS snapshot
func (d *Detector) hasCorrespondingTrueNASSnapshot(k8sSnapshot interface{}, truenasSnapshots []truenas.Snapshot) bool {
	// This would need to be implemented based on the actual VolumeSnapshot type
	// For now, return true to avoid false positives
	return true
}

// hasCorrespondingK8sSnapshot checks if a TrueNAS snapshot has a corresponding K8s snapshot
func (d *Detector) hasCorrespondingK8sSnapshot(truenasSnapshot truenas.Snapshot, k8sSnapshots interface{}) bool {
	// This would need to be implemented based on the actual VolumeSnapshot type
	// For now, return true to avoid false positives
	return true
}

// extractDatasetFromVolumeHandle extracts the dataset name from a CSI volume handle
func (d *Detector) extractDatasetFromVolumeHandle(volumeHandle string) string {
	// Handle different volume handle formats
	if strings.Contains(volumeHandle, "iqn.") {
		// iSCSI format: iqn.2005-10.org.freenas.ctl:volume-name
		parts := strings.Split(volumeHandle, ":")
		if len(parts) > 1 {
			return parts[len(parts)-1]
		}
	} else if strings.Contains(volumeHandle, "/") {
		// Dataset format: pool/dataset/volume-name
		parts := strings.Split(volumeHandle, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	return volumeHandle
}

// volumeMatches checks if a TrueNAS volume matches the given identifiers
func (d *Detector) volumeMatches(volume truenas.Volume, volumeHandle, datasetName string) bool {
	// Direct name match
	if volume.Name == datasetName || volume.Name == volumeHandle {
		return true
	}

	// Check if volume ID contains the dataset name
	if strings.Contains(volume.ID, datasetName) {
		return true
	}

	// Check if volume path contains the dataset name
	if strings.Contains(volume.Path, datasetName) {
		return true
	}

	// Check properties for additional matching
	if volume.Properties != nil {
		for _, value := range volume.Properties {
			if strings.Contains(value, datasetName) {
				return true
			}
		}
	}

	return false
}