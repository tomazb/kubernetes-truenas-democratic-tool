package schemas

import (
	"time"
)

// OrphanedResourcesReport represents the orphaned resources scan report
type OrphanedResourcesReport struct {
	Timestamp           time.Time            `json:"timestamp"`
	ScanDurationSeconds float64              `json:"scan_duration_seconds"`
	ClusterInfo         *ClusterInfo         `json:"cluster_info,omitempty"`
	TrueNASInfo         *TrueNASInfo         `json:"truenas_info,omitempty"`
	Summary             OrphanedSummary      `json:"summary"`
	OrphanedResources   []OrphanedResource   `json:"orphaned_resources"`
}

// ClusterInfo contains Kubernetes cluster information
type ClusterInfo struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Platform string `json:"platform,omitempty"`
}

// TrueNASInfo contains TrueNAS system information
type TrueNASInfo struct {
	Host    string   `json:"host"`
	Version string   `json:"version,omitempty"`
	Pools   []string `json:"pools,omitempty"`
}

// OrphanedSummary contains summary statistics
type OrphanedSummary struct {
	TotalOrphans             int   `json:"total_orphans"`
	OrphanedPVs              int   `json:"orphaned_pvs"`
	OrphanedPVCs             int   `json:"orphaned_pvcs"`
	OrphanedSnapshots        int   `json:"orphaned_snapshots"`
	OrphanedTrueNASVolumes   int   `json:"orphaned_truenas_volumes"`
	TotalWastedSpaceBytes    int64 `json:"total_wasted_space_bytes"`
}

// OrphanedResource represents a single orphaned resource
type OrphanedResource struct {
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	Namespace    *string                `json:"namespace,omitempty"`
	VolumeHandle *string                `json:"volume_handle,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	SizeBytes    *int64                 `json:"size_bytes,omitempty"`
	Location     string                 `json:"location"`
	Reason       string                 `json:"reason"`
	Remediation  Remediation            `json:"remediation"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// Remediation contains remediation information
type Remediation struct {
	Action  string  `json:"action"`
	Safe    bool    `json:"safe"`
	Command *string `json:"command,omitempty"`
	Notes   *string `json:"notes,omitempty"`
}

// StorageAnalysisReport represents storage analysis results
type StorageAnalysisReport struct {
	Timestamp       time.Time          `json:"timestamp"`
	AnalysisPeriod  *AnalysisPeriod    `json:"analysis_period,omitempty"`
	StorageSummary  StorageSummary     `json:"storage_summary"`
	PoolStatistics  []PoolStats        `json:"pool_statistics,omitempty"`
	VolumeStatistics []VolumeStats     `json:"volume_statistics,omitempty"`
	GrowthTrends    *GrowthTrends      `json:"growth_trends,omitempty"`
	Recommendations []Recommendation   `json:"recommendations,omitempty"`
	Alerts          []Alert            `json:"alerts,omitempty"`
}

// AnalysisPeriod defines the time period analyzed
type AnalysisPeriod struct {
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	DurationDays int       `json:"duration_days"`
}

// StorageSummary contains overall storage statistics
type StorageSummary struct {
	TotalCapacityBytes      int64   `json:"total_capacity_bytes"`
	TotalAllocatedBytes     int64   `json:"total_allocated_bytes"`
	TotalUsedBytes          int64   `json:"total_used_bytes"`
	ThinProvisioningRatio   float64 `json:"thin_provisioning_ratio,omitempty"`
	StorageEfficiencyPercent float64 `json:"storage_efficiency_percent,omitempty"`
	SnapshotOverheadBytes   int64   `json:"snapshot_overhead_bytes,omitempty"`
}

// PoolStats contains pool-level statistics
type PoolStats struct {
	Name                 string  `json:"name"`
	CapacityBytes        int64   `json:"capacity_bytes"`
	UsedBytes            int64   `json:"used_bytes"`
	FreeBytes            int64   `json:"free_bytes"`
	FragmentationPercent float64 `json:"fragmentation_percent,omitempty"`
	HealthStatus         string  `json:"health_status"`
	DatasetCount         int     `json:"dataset_count,omitempty"`
	SnapshotCount        int     `json:"snapshot_count,omitempty"`
}

// VolumeStats contains volume-level statistics
type VolumeStats struct {
	PVName                string     `json:"pv_name"`
	PVCName               *string    `json:"pvc_name,omitempty"`
	Namespace             *string    `json:"namespace,omitempty"`
	StorageClass          string     `json:"storage_class"`
	AllocatedBytes        int64      `json:"allocated_bytes"`
	UsedBytes             int64      `json:"used_bytes"`
	SnapshotCount         int        `json:"snapshot_count,omitempty"`
	SnapshotUsedBytes     int64      `json:"snapshot_used_bytes,omitempty"`
	EfficiencyRatio       float64    `json:"efficiency_ratio,omitempty"`
	LastAccessed          *time.Time `json:"last_accessed,omitempty"`
	GrowthRateDailyBytes  int64      `json:"growth_rate_daily_bytes,omitempty"`
	ProjectedFullDate     *time.Time `json:"projected_full_date,omitempty"`
}

// GrowthTrends contains storage growth trend information
type GrowthTrends struct {
	DailyGrowthRateBytes   int64      `json:"daily_growth_rate_bytes"`
	WeeklyGrowthRateBytes  int64      `json:"weekly_growth_rate_bytes"`
	MonthlyGrowthRateBytes int64      `json:"monthly_growth_rate_bytes"`
	ProjectedFullDate      *time.Time `json:"projected_full_date,omitempty"`
	DaysUntilFull          *int       `json:"days_until_full,omitempty"`
}

// Recommendation represents a storage optimization recommendation
type Recommendation struct {
	Type                   string  `json:"type"`
	Severity               string  `json:"severity"`
	Resource               string  `json:"resource,omitempty"`
	Description            string  `json:"description"`
	PotentialSavingsBytes  *int64  `json:"potential_savings_bytes,omitempty"`
	Action                 string  `json:"action"`
	Impact                 string  `json:"impact,omitempty"`
}

// Alert represents a storage alert
type Alert struct {
	Level        string                 `json:"level"`
	Category     string                 `json:"category"`
	Message      string                 `json:"message"`
	Resource     *string                `json:"resource,omitempty"`
	Threshold    *float64               `json:"threshold,omitempty"`
	CurrentValue *float64               `json:"current_value,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// ConfigValidationReport represents configuration validation results
type ConfigValidationReport struct {
	Timestamp                   time.Time                    `json:"timestamp"`
	ValidationSummary           ValidationSummary            `json:"validation_summary"`
	StorageClassValidations     []StorageClassValidation     `json:"storage_class_validations,omitempty"`
	CSIDriverValidation         *CSIDriverValidation         `json:"csi_driver_validation,omitempty"`
	SnapshotClassValidations    []SnapshotClassValidation    `json:"snapshot_class_validations,omitempty"`
	TrueNASConnectionValidation *TrueNASValidation           `json:"truenas_connection_validation,omitempty"`
	BestPracticeChecks          []BestPracticeCheck          `json:"best_practice_checks,omitempty"`
}

// ValidationSummary contains validation summary statistics
type ValidationSummary struct {
	TotalChecks    int    `json:"total_checks"`
	Passed         int    `json:"passed"`
	Failed         int    `json:"failed"`
	Warnings       int    `json:"warnings"`
	OverallStatus  string `json:"overall_status"`
}

// StorageClassValidation contains StorageClass validation results
type StorageClassValidation struct {
	Name        string              `json:"name"`
	Provisioner string              `json:"provisioner"`
	Status      string              `json:"status"`
	Checks      []ValidationCheck   `json:"checks"`
	Parameters  map[string]string   `json:"parameters,omitempty"`
	Issues      []string            `json:"issues,omitempty"`
}

// CSIDriverValidation contains CSI driver validation results
type CSIDriverValidation struct {
	DriverName        string            `json:"driver_name"`
	Status            string            `json:"status"`
	ControllerPods    []PodStatus       `json:"controller_pods,omitempty"`
	NodePods          []PodStatus       `json:"node_pods,omitempty"`
	CSINodes          []CSINodeStatus   `json:"csi_nodes,omitempty"`
	RBACPermissions   *RBACPermissions  `json:"rbac_permissions,omitempty"`
}

// SnapshotClassValidation contains VolumeSnapshotClass validation results
type SnapshotClassValidation struct {
	Name           string   `json:"name"`
	Driver         string   `json:"driver"`
	DeletionPolicy string   `json:"deletion_policy"`
	Status         string   `json:"status"`
	Issues         []string `json:"issues,omitempty"`
}

// TrueNASValidation contains TrueNAS connection validation results
type TrueNASValidation struct {
	Host               string       `json:"host"`
	ConnectionStatus   string       `json:"connection_status"`
	APIVersion         *string      `json:"api_version,omitempty"`
	Pools              []PoolStatus `json:"pools,omitempty"`
	DatasetsConfigured bool         `json:"datasets_configured"`
	ISCSIConfigured    bool         `json:"iscsi_configured"`
	NFSConfigured      bool         `json:"nfs_configured"`
	Issues             []string     `json:"issues,omitempty"`
}

// BestPracticeCheck represents a best practice validation
type BestPracticeCheck struct {
	Category         string  `json:"category"`
	Check            string  `json:"check"`
	Status           string  `json:"status"`
	Severity         string  `json:"severity"`
	Description      string  `json:"description"`
	Recommendation   *string `json:"recommendation,omitempty"`
	DocumentationURL *string `json:"documentation_url,omitempty"`
}

// ValidationCheck represents a single validation check result
type ValidationCheck struct {
	Name    string                 `json:"name"`
	Passed  bool                   `json:"passed"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// PodStatus represents the status of a pod
type PodStatus struct {
	Name       string             `json:"name"`
	Namespace  string             `json:"namespace"`
	Node       string             `json:"node,omitempty"`
	Status     string             `json:"status"`
	Ready      bool               `json:"ready"`
	Restarts   int                `json:"restarts,omitempty"`
	Containers []ContainerStatus  `json:"containers,omitempty"`
}

// ContainerStatus represents the status of a container
type ContainerStatus struct {
	Name  string `json:"name"`
	Ready bool   `json:"ready"`
	Image string `json:"image,omitempty"`
}

// CSINodeStatus represents CSI node status
type CSINodeStatus struct {
	NodeName        string  `json:"node_name"`
	DriverInstalled bool    `json:"driver_installed"`
	DriverVersion   *string `json:"driver_version,omitempty"`
}

// RBACPermissions represents RBAC permission information
type RBACPermissions struct {
	ServiceAccount     string   `json:"service_account"`
	ClusterRoles       []string `json:"cluster_roles,omitempty"`
	MissingPermissions []string `json:"missing_permissions,omitempty"`
}

// PoolStatus represents pool availability status
type PoolStatus struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Available bool   `json:"available"`
}