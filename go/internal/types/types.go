package types

import (
	"time"
)

// ResourceType represents the type of Kubernetes resource
type ResourceType string

const (
	ResourceTypePV       ResourceType = "PersistentVolume"
	ResourceTypePVC      ResourceType = "PersistentVolumeClaim"
	ResourceTypeSnapshot ResourceType = "VolumeSnapshot"
)

// OrphanedResource represents a resource that appears to be orphaned
type OrphanedResource struct {
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace,omitempty"`
	Type         ResourceType      `json:"type"`
	CreationTime time.Time         `json:"creation_time"`
	Reason       string            `json:"reason"`
	Details      map[string]string `json:"details,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// PersistentVolumeInfo represents information about a PV
type PersistentVolumeInfo struct {
	Name          string            `json:"name"`
	Capacity      string            `json:"capacity"`
	Phase         string            `json:"phase"`
	StorageClass  string            `json:"storage_class,omitempty"`
	ClaimName     string            `json:"claim_name,omitempty"`
	ClaimNamespace string           `json:"claim_namespace,omitempty"`
	CreationTime  time.Time         `json:"creation_time"`
	Labels        map[string]string `json:"labels,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`
}

// PersistentVolumeClaimInfo represents information about a PVC
type PersistentVolumeClaimInfo struct {
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace"`
	Capacity     string            `json:"capacity,omitempty"`
	Phase        string            `json:"phase"`
	StorageClass string            `json:"storage_class,omitempty"`
	VolumeName   string            `json:"volume_name,omitempty"`
	CreationTime time.Time         `json:"creation_time"`
	Labels       map[string]string `json:"labels,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
}

// VolumeSnapshotInfo represents information about a VolumeSnapshot
type VolumeSnapshotInfo struct {
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace"`
	ReadyToUse   bool              `json:"ready_to_use"`
	CreationTime time.Time         `json:"creation_time"`
	SourcePVC    string            `json:"source_pvc,omitempty"`
	Size         string            `json:"size,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
}

// StorageClassInfo represents information about a StorageClass
type StorageClassInfo struct {
	Name        string            `json:"name"`
	Provisioner string            `json:"provisioner"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// TrueNASPool represents a TrueNAS storage pool
type TrueNASPool struct {
	Name          string  `json:"name"`
	TotalSize     int64   `json:"total_size"`
	UsedSize      int64   `json:"used_size"`
	FreeSize      int64   `json:"free_size"`
	Status        string  `json:"status"`
	Healthy       bool    `json:"healthy"`
	Fragmentation string  `json:"fragmentation,omitempty"`
	Topology      string  `json:"topology,omitempty"`
}

// TrueNASDataset represents a TrueNAS dataset
type TrueNASDataset struct {
	Name         string            `json:"name"`
	Pool         string            `json:"pool"`
	Type         string            `json:"type"`
	Used         int64             `json:"used"`
	Available    int64             `json:"available"`
	Referenced   int64             `json:"referenced"`
	Compression  string            `json:"compression,omitempty"`
	Properties   map[string]string `json:"properties,omitempty"`
	CreationTime time.Time         `json:"creation_time"`
}

// TrueNASSnapshot represents a TrueNAS snapshot
type TrueNASSnapshot struct {
	Name         string    `json:"name"`
	Dataset      string    `json:"dataset"`
	FullName     string    `json:"full_name"`
	UsedSize     int64     `json:"used_size"`
	Referenced   int64     `json:"referenced"`
	CreationTime time.Time `json:"creation_time"`
}

// TrueNASVolume represents a TrueNAS volume (iSCSI or NFS)
type TrueNASVolume struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"` // iscsi, nfs
	Size         int64             `json:"size"`
	Path         string            `json:"path"`
	Dataset      string            `json:"dataset"`
	Status       string            `json:"status"`
	Properties   map[string]string `json:"properties,omitempty"`
	CreationTime time.Time         `json:"creation_time"`
}

// CSIDriverInfo represents information about a CSI driver
type CSIDriverInfo struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Pods    []PodInfo `json:"pods"`
}

// PodInfo represents information about a pod
type PodInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Node      string `json:"node,omitempty"`
	Phase     string `json:"phase"`
	Ready     bool   `json:"ready"`
	Restarts  int32  `json:"restarts"`
}

// MonitoringResult represents the result of a monitoring check
type MonitoringResult struct {
	Timestamp       time.Time          `json:"timestamp"`
	OrphanedPVs     []OrphanedResource `json:"orphaned_pvs"`
	OrphanedPVCs    []OrphanedResource `json:"orphaned_pvcs"`
	OrphanedSnapshots []OrphanedResource `json:"orphaned_snapshots"`
	StorageUsage    []TrueNASPool      `json:"storage_usage"`
	CSIDriverHealth CSIDriverInfo      `json:"csi_driver_health"`
	Alerts          []Alert            `json:"alerts"`
}

// Alert represents a monitoring alert
type Alert struct {
	Level     string            `json:"level"`     // info, warning, error, critical
	Category  string            `json:"category"`  // cleanup, storage, health, system
	Message   string            `json:"message"`
	Details   map[string]string `json:"details,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// HealthCheck represents a health check result
type HealthCheck struct {
	Name      string            `json:"name"`
	Healthy   bool              `json:"healthy"`
	Message   string            `json:"message"`
	Details   map[string]string `json:"details,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// ValidationResult represents configuration validation results
type ValidationResult struct {
	Valid       bool          `json:"valid"`
	Checks      []HealthCheck `json:"checks"`
	Errors      []string      `json:"errors,omitempty"`
	Warnings    []string      `json:"warnings,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
}

// StorageAnalysis represents storage usage analysis
type StorageAnalysis struct {
	TotalAllocated         int64   `json:"total_allocated"`
	TotalUsed              int64   `json:"total_used"`
	ThinProvisioningRatio  float64 `json:"thin_provisioning_ratio"`
	CompressionRatio       float64 `json:"compression_ratio"`
	SnapshotOverhead       int64   `json:"snapshot_overhead"`
	SnapshotOverheadPercent float64 `json:"snapshot_overhead_percent"`
	Pools                  []TrueNASPool `json:"pools"`
	Recommendations        []string `json:"recommendations"`
	Timestamp              time.Time `json:"timestamp"`
}

// SnapshotAnalysis represents snapshot analysis results
type SnapshotAnalysis struct {
	TotalSnapshots      int                    `json:"total_snapshots"`
	TotalSize           int64                  `json:"total_size"`
	AverageAge          time.Duration          `json:"average_age"`
	SnapshotsByAge      map[string]int         `json:"snapshots_by_age"`
	LargeSnapshots      []TrueNASSnapshot      `json:"large_snapshots"`
	OrphanedSnapshots   []OrphanedResource     `json:"orphaned_snapshots"`
	Recommendations     []string               `json:"recommendations"`
	Timestamp           time.Time              `json:"timestamp"`
}