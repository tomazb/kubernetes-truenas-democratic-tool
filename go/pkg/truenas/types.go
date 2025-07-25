package truenas

import (
	"time"
)

// Config holds the configuration for TrueNAS client
type Config struct {
	Host       string
	Port       int
	APIKey     string
	Username   string
	Password   string
	VerifySSL  bool
	Timeout    time.Duration
	MaxRetries int
}

// Pool represents a TrueNAS storage pool
type Pool struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Status       string   `json:"status"`
	Size         int64    `json:"size"`
	Allocated    int64    `json:"allocated"`
	Free         int64    `json:"free"`
	Fragmentation string  `json:"fragmentation"`
	Healthy      bool     `json:"healthy"`
	Scan         PoolScan `json:"scan"`
}

// PoolScan represents pool scrub/resilver status
type PoolScan struct {
	State    string `json:"state"`
	Function string `json:"function"`
	Errors   int    `json:"errors"`
}

// Dataset represents a ZFS dataset
type Dataset struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Used       DatasetProperty `json:"used"`
	Available  DatasetProperty `json:"available"`
	Referenced DatasetProperty `json:"referenced"`
	Quota      DatasetProperty `json:"quota"`
	Refquota   DatasetProperty `json:"refquota"`
	Compression string        `json:"compression"`
	Compressratio string      `json:"compressratio"`
}

// DatasetProperty represents a dataset property value
type DatasetProperty struct {
	Value  interface{} `json:"value"`
	Parsed int64      `json:"parsed,omitempty"`
}

// Volume represents an iSCSI extent
type Volume struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Path     string `json:"path"`
	FileSize int64  `json:"filesize"`
	NAA      string `json:"naa"`
	Serial   string `json:"serial"`
	Enabled  bool   `json:"enabled"`
	RO       bool   `json:"ro"`
}

// NFSShare represents an NFS share
type NFSShare struct {
	ID          int      `json:"id"`
	Path        string   `json:"path"`
	Comment     string   `json:"comment"`
	Enabled     bool     `json:"enabled"`
	Hosts       []string `json:"hosts"`
	MapallUser  string   `json:"mapall_user"`
	MapallGroup string   `json:"mapall_group"`
}

// Snapshot represents a ZFS snapshot
type Snapshot struct {
	ID           string             `json:"id"`
	Dataset      string             `json:"dataset"`
	SnapshotName string             `json:"snapshot_name"`
	Properties   SnapshotProperties `json:"properties"`
}

// SnapshotProperties contains snapshot properties
type SnapshotProperties struct {
	Used       int64 `json:"used"`
	Referenced int64 `json:"referenced"`
	Creation   int64 `json:"creation"`
}

// OrphanedVolume represents a volume that exists in TrueNAS but not in Kubernetes
type OrphanedVolume struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Type         string    `json:"type"` // "iscsi" or "nfs"
	Size         int64     `json:"size,omitempty"`
	CreationTime time.Time `json:"creation_time,omitempty"`
}

// TargetGroup represents an iSCSI target group
type TargetGroup struct {
	ID          int                  `json:"id"`
	Name        string               `json:"name"`
	Mode        string               `json:"mode"`
	Groups      []InitiatorGroup     `json:"groups"`
	Auth        *TargetAuth          `json:"auth"`
}

// InitiatorGroup represents an iSCSI initiator group
type InitiatorGroup struct {
	ID         int      `json:"id"`
	Initiators []string `json:"initiators"`
	Comment    string   `json:"comment"`
}

// TargetAuth represents iSCSI target authentication
type TargetAuth struct {
	AuthMethod string `json:"authmethod"`
	Tag        int    `json:"tag"`
}

// TargetExtent represents an iSCSI target to extent mapping
type TargetExtent struct {
	ID       int    `json:"id"`
	Target   int    `json:"target"`
	Extent   int    `json:"extent"`
	LunID    int    `json:"lunid"`
}

// ErrorResponse represents an error response from TrueNAS API
type ErrorResponse struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}