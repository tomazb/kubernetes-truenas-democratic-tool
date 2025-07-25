package k8s

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
)

// EventType represents the type of Kubernetes watch event
type EventType string

const (
	EventAdded    EventType = "ADDED"
	EventModified EventType = "MODIFIED"
	EventDeleted  EventType = "DELETED"
	EventError    EventType = "ERROR"
)

// Config holds the configuration for the Kubernetes client
type Config struct {
	// Kubeconfig is the path to the kubeconfig file
	Kubeconfig string
	// Namespace to watch (empty for all namespaces)
	Namespace string
	// CSIDriver name to filter resources
	CSIDriver string
	// StorageClassName to filter PVCs
	StorageClassName string
	// InCluster indicates if running inside a cluster
	InCluster bool
	// ResyncPeriod for informers
	ResyncPeriod time.Duration
}

// PVEvent represents a PersistentVolume watch event
type PVEvent struct {
	Type EventType
	PV   *v1.PersistentVolume
}

// PVCEvent represents a PersistentVolumeClaim watch event
type PVCEvent struct {
	Type EventType
	PVC  *v1.PersistentVolumeClaim
}

// SnapshotEvent represents a VolumeSnapshot watch event
type SnapshotEvent struct {
	Type     EventType
	Snapshot *snapshotv1.VolumeSnapshot
}

// OrphanedResource represents a resource that exists in one system but not the other
type OrphanedResource struct {
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	Namespace    string                 `json:"namespace,omitempty"`
	VolumeHandle string                 `json:"volumeHandle,omitempty"`
	CreatedAt    time.Time              `json:"createdAt"`
	Size         string                 `json:"size,omitempty"`
	Location     string                 `json:"location"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// VolumeInfo contains information about a volume from both K8s and TrueNAS
type VolumeInfo struct {
	// Kubernetes information
	PVName       string                     `json:"pvName"`
	PVCName      string                     `json:"pvcName,omitempty"`
	Namespace    string                     `json:"namespace,omitempty"`
	VolumeHandle string                     `json:"volumeHandle"`
	Size         string                     `json:"size"`
	Status       v1.PersistentVolumePhase   `json:"status"`
	AccessModes  []v1.PersistentVolumeAccessMode `json:"accessModes"`
	
	// TrueNAS information
	TrueNASVolume string                 `json:"truenasVolume,omitempty"`
	ActualSize    int64                  `json:"actualSize,omitempty"`
	UsedSize      int64                  `json:"usedSize,omitempty"`
	SnapshotCount int                    `json:"snapshotCount,omitempty"`
	
	// Metadata
	CreatedAt     time.Time              `json:"createdAt"`
	Labels        map[string]string      `json:"labels,omitempty"`
	Annotations   map[string]string      `json:"annotations,omitempty"`
}

// SnapshotInfo contains information about a snapshot from both K8s and TrueNAS
type SnapshotInfo struct {
	// Kubernetes information
	Name               string    `json:"name"`
	Namespace          string    `json:"namespace"`
	SourcePVCName      string    `json:"sourcePvcName"`
	SnapshotClassName  string    `json:"snapshotClassName"`
	ReadyToUse         bool      `json:"readyToUse"`
	CreationTime       time.Time `json:"creationTime"`
	
	// TrueNAS information
	TrueNASSnapshot    string    `json:"truenasSnapshot,omitempty"`
	ActualSize         int64     `json:"actualSize,omitempty"`
	ReferencedSize     int64     `json:"referencedSize,omitempty"`
	
	// Metadata
	Labels             map[string]string `json:"labels,omitempty"`
	Annotations        map[string]string `json:"annotations,omitempty"`
}

// StorageClassInfo contains StorageClass configuration details
type StorageClassInfo struct {
	Name              string            `json:"name"`
	Provisioner       string            `json:"provisioner"`
	Parameters        map[string]string `json:"parameters"`
	ReclaimPolicy     string            `json:"reclaimPolicy"`
	VolumeBindingMode string            `json:"volumeBindingMode"`
	AllowVolumeExpansion bool           `json:"allowVolumeExpansion"`
}

// ClientInterface defines the interface for Kubernetes operations
type ClientInterface interface {
	// PersistentVolume operations
	GetPersistentVolumes(ctx context.Context) ([]*v1.PersistentVolume, error)
	GetPersistentVolume(ctx context.Context, name string) (*v1.PersistentVolume, error)
	WatchPersistentVolumes(ctx context.Context, eventCh chan<- PVEvent)
	
	// PersistentVolumeClaim operations
	GetPersistentVolumeClaims(ctx context.Context) ([]*v1.PersistentVolumeClaim, error)
	GetPersistentVolumeClaim(ctx context.Context, namespace, name string) (*v1.PersistentVolumeClaim, error)
	WatchPersistentVolumeClaims(ctx context.Context, eventCh chan<- PVCEvent)
	
	// VolumeSnapshot operations
	GetVolumeSnapshots(ctx context.Context) ([]*snapshotv1.VolumeSnapshot, error)
	GetVolumeSnapshot(ctx context.Context, namespace, name string) (*snapshotv1.VolumeSnapshot, error)
	WatchVolumeSnapshots(ctx context.Context, eventCh chan<- SnapshotEvent)
	
	// StorageClass operations
	GetStorageClasses(ctx context.Context) ([]*storagev1.StorageClass, error)
	GetStorageClass(ctx context.Context, name string) (*storagev1.StorageClass, error)
	
	// CSI operations
	GetCSINodes(ctx context.Context) ([]*storagev1.CSINode, error)
	GetVolumeAttachments(ctx context.Context) ([]*storagev1.VolumeAttachment, error)
	
	// Health checks
	CheckCSIDriverHealth(ctx context.Context) error
	GetCSIDriverPods(ctx context.Context) ([]*v1.Pod, error)
}