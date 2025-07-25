package k8s

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"

	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	snapshotclient "github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned"

	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/logging"
	"go.uber.org/zap"
)

// RBACValidationResult holds RBAC validation results
type RBACValidationResult struct {
	HasRequiredPermissions bool                    `json:"has_required_permissions"`
	MissingPermissions     []string                `json:"missing_permissions"`
	PermissionChecks       map[string]bool         `json:"permission_checks"`
	ServiceAccount         string                  `json:"service_account"`
	Namespace              string                  `json:"namespace"`
}

// ClusterInfo holds cluster information
type ClusterInfo struct {
	Version           string            `json:"version"`
	Platform          string            `json:"platform"`
	NodeCount         int               `json:"node_count"`
	NamespaceCount    int               `json:"namespace_count"`
	StorageClasses    []string          `json:"storage_classes"`
	CSIDrivers        []string          `json:"csi_drivers"`
	DemocraticCSI     bool              `json:"democratic_csi_present"`
	Capabilities      map[string]bool   `json:"capabilities"`
}

// Client represents a Kubernetes client
type Client interface {
	// Core resource listing
	ListPersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error)
	ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]corev1.PersistentVolumeClaim, error)
	ListVolumeSnapshots(ctx context.Context, namespace string) ([]snapshotv1.VolumeSnapshot, error)
	ListStorageClasses(ctx context.Context) ([]storagev1.StorageClass, error)
	ListPods(ctx context.Context, namespace string) ([]corev1.Pod, error)
	ListNamespaces(ctx context.Context) ([]corev1.Namespace, error)
	GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error)
	
	// Resource filtering
	ListPersistentVolumesByStorageClass(ctx context.Context, storageClass string) ([]corev1.PersistentVolume, error)
	ListPersistentVolumeClaimsByStorageClass(ctx context.Context, namespace, storageClass string) ([]corev1.PersistentVolumeClaim, error)
	ListDemocraticCSIPersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error)
	ListUnboundPersistentVolumeClaims(ctx context.Context, namespace string) ([]corev1.PersistentVolumeClaim, error)
	
	// Health and validation
	TestConnection(ctx context.Context) error
	ValidateRBACPermissions(ctx context.Context) (*RBACValidationResult, error)
	GetClusterInfo(ctx context.Context) (*ClusterInfo, error)
	
	// CSI specific
	ListCSINodes(ctx context.Context) ([]storagev1.CSINode, error)
	ListCSIDrivers(ctx context.Context) ([]storagev1.CSIDriver, error)
	ListVolumeAttachments(ctx context.Context) ([]storagev1.VolumeAttachment, error)
	GetCSIDriverPods(ctx context.Context, namespace string) ([]corev1.Pod, error)
}

// client implements the Client interface
type client struct {
	clientset       kubernetes.Interface
	snapshotClient  snapshotclient.Interface
	logger          *logging.Logger
	config          Config
}

// Config holds Kubernetes client configuration
type Config struct {
	Kubeconfig    string
	InCluster     bool
	Namespace     string
	Timeout       time.Duration
	RetryAttempts int
	QPS           float32
	Burst         int
}

// NewClient creates a new Kubernetes client
func NewClient(config Config) (Client, error) {
	// Set defaults
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}
	if config.QPS == 0 {
		config.QPS = 50.0
	}
	if config.Burst == 0 {
		config.Burst = 100
	}

	var restConfig *rest.Config
	var err error

	if config.InCluster {
		// Use in-cluster configuration
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
		}
	} else {
		// Use kubeconfig file
		kubeconfigPath := config.Kubeconfig
		if kubeconfigPath == "" {
			if home := homedir.HomeDir(); home != "" {
				kubeconfigPath = filepath.Join(home, ".kube", "config")
			}
		}

		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create config from kubeconfig: %w", err)
		}
	}

	// Configure connection settings
	restConfig.Timeout = config.Timeout
	restConfig.QPS = config.QPS
	restConfig.Burst = config.Burst

	// Create clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	// Create snapshot client
	snapshotClient, err := snapshotclient.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot client: %w", err)
	}

	// Initialize logger
	logger, err := logging.NewLogger(logging.Config{
		Level:       "info",
		Encoding:    "json",
		Development: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return &client{
		clientset:      clientset,
		snapshotClient: snapshotClient,
		logger:         logger,
		config:         config,
	}, nil
}

// ListPersistentVolumes lists all persistent volumes with retry logic
func (c *client) ListPersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error) {
	var pvList *corev1.PersistentVolumeList
	
	err := retry.OnError(
		retry.DefaultRetry,
		func(err error) bool {
			// Retry on temporary network errors
			return err != nil
		},
		func() error {
			var err error
			pvList, err = c.clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
			return err
		},
	)
	
	if err != nil {
		c.logger.Error("Failed to list persistent volumes after retries", zap.Error(err))
		return nil, fmt.Errorf("failed to list persistent volumes: %w", err)
	}

	c.logger.LogK8sOperation("list", "persistentvolumes", "", "", nil)
	c.logger.Debug("Kubernetes operation completed",
		zap.String("operation", "list"),
		zap.String("resource", "persistentvolumes"),
		zap.Int("count", len(pvList.Items)))
	
	return pvList.Items, nil
}

// ListPersistentVolumeClaims lists persistent volume claims in a namespace with retry logic
func (c *client) ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]corev1.PersistentVolumeClaim, error) {
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	var pvcList *corev1.PersistentVolumeClaimList
	
	err := retry.OnError(
		retry.DefaultRetry,
		func(err error) bool {
			return err != nil
		},
		func() error {
			var err error
			pvcList, err = c.clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
			return err
		},
	)
	
	if err != nil {
		c.logger.Error("Failed to list persistent volume claims after retries",
			zap.Error(err),
			zap.String("namespace", namespace))
		return nil, fmt.Errorf("failed to list persistent volume claims: %w", err)
	}

	c.logger.LogK8sOperation("list", "persistentvolumeclaims", namespace, "", nil)
	
	return pvcList.Items, nil
}

// ListVolumeSnapshots lists volume snapshots in a namespace with retry logic
func (c *client) ListVolumeSnapshots(ctx context.Context, namespace string) ([]snapshotv1.VolumeSnapshot, error) {
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	var snapshotList *snapshotv1.VolumeSnapshotList
	
	err := retry.OnError(
		retry.DefaultRetry,
		func(err error) bool {
			return err != nil
		},
		func() error {
			var err error
			snapshotList, err = c.snapshotClient.SnapshotV1().VolumeSnapshots(namespace).List(ctx, metav1.ListOptions{})
			return err
		},
	)
	
	if err != nil {
		c.logger.Error("Failed to list volume snapshots after retries",
			zap.Error(err),
			zap.String("namespace", namespace))
		return nil, fmt.Errorf("failed to list volume snapshots: %w", err)
	}

	c.logger.LogK8sOperation("list", "volumesnapshots", namespace, "", nil)
	
	return snapshotList.Items, nil
}

// ListStorageClasses lists all storage classes with retry logic
func (c *client) ListStorageClasses(ctx context.Context) ([]storagev1.StorageClass, error) {
	var scList *storagev1.StorageClassList
	
	err := retry.OnError(
		retry.DefaultRetry,
		func(err error) bool {
			return err != nil
		},
		func() error {
			var err error
			scList, err = c.clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
			return err
		},
	)
	
	if err != nil {
		c.logger.Error("Failed to list storage classes after retries", zap.Error(err))
		return nil, fmt.Errorf("failed to list storage classes: %w", err)
	}

	c.logger.LogK8sOperation("list", "storageclasses", "", "", nil)
	
	return scList.Items, nil
}

// ListPods lists pods in a namespace with retry logic
func (c *client) ListPods(ctx context.Context, namespace string) ([]corev1.Pod, error) {
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	var podList *corev1.PodList
	
	err := retry.OnError(
		retry.DefaultRetry,
		func(err error) bool {
			return err != nil
		},
		func() error {
			var err error
			podList, err = c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
			return err
		},
	)
	
	if err != nil {
		c.logger.Error("Failed to list pods after retries",
			zap.Error(err),
			zap.String("namespace", namespace))
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	c.logger.LogK8sOperation("list", "pods", namespace, "", nil)
	
	return podList.Items, nil
}

// GetNamespace gets a specific namespace with retry logic
func (c *client) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	var namespace *corev1.Namespace
	
	err := retry.OnError(
		retry.DefaultRetry,
		func(err error) bool {
			return err != nil
		},
		func() error {
			var err error
			namespace, err = c.clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
			return err
		},
	)
	
	if err != nil {
		c.logger.Error("Failed to get namespace after retries",
			zap.Error(err),
			zap.String("namespace", name))
		return nil, fmt.Errorf("failed to get namespace %s: %w", name, err)
	}

	c.logger.LogK8sOperation("get", "namespace", "", name, nil)
	
	return namespace, nil
}

// TestConnection tests the Kubernetes connection with retry logic
func (c *client) TestConnection(ctx context.Context) error {
	err := retry.OnError(
		retry.DefaultRetry,
		func(err error) bool {
			return err != nil
		},
		func() error {
			_, err := c.clientset.Discovery().ServerVersion()
			return err
		},
	)
	
	if err != nil {
		c.logger.Error("Failed to connect to Kubernetes API after retries", zap.Error(err))
		return fmt.Errorf("failed to connect to Kubernetes API: %w", err)
	}

	c.logger.Info("Kubernetes connection test successful")
	
	return nil
}

// ListDemocraticCSIPersistentVolumes lists PVs managed by democratic-csi
func (c *client) ListDemocraticCSIPersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error) {
	pvs, err := c.ListPersistentVolumes(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []corev1.PersistentVolume
	for _, pv := range pvs {
		if pv.Spec.CSI != nil && isDemocraticCSIDriver(pv.Spec.CSI.Driver) {
			filtered = append(filtered, pv)
		}
	}

	c.logger.Info("Filtered PVs by democratic-csi driver",
		zap.Int("total_pvs", len(pvs)),
		zap.Int("democratic_csi_pvs", len(filtered)))

	return filtered, nil
}

// ListPersistentVolumesByStorageClass lists PVs filtered by storage class
func (c *client) ListPersistentVolumesByStorageClass(ctx context.Context, storageClass string) ([]corev1.PersistentVolume, error) {
	pvs, err := c.ListPersistentVolumes(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []corev1.PersistentVolume
	for _, pv := range pvs {
		if pv.Spec.StorageClassName == storageClass {
			filtered = append(filtered, pv)
		}
	}

	c.logger.Info("Filtered PVs by storage class",
		zap.String("storage_class", storageClass),
		zap.Int("total_pvs", len(pvs)),
		zap.Int("filtered_pvs", len(filtered)))

	return filtered, nil
}

// ListUnboundPersistentVolumeClaims lists PVCs that are in Pending state
func (c *client) ListUnboundPersistentVolumeClaims(ctx context.Context, namespace string) ([]corev1.PersistentVolumeClaim, error) {
	pvcs, err := c.ListPersistentVolumeClaims(ctx, namespace)
	if err != nil {
		return nil, err
	}

	var unbound []corev1.PersistentVolumeClaim
	for _, pvc := range pvcs {
		if pvc.Status.Phase == corev1.ClaimPending {
			unbound = append(unbound, pvc)
		}
	}

	c.logger.Info("Found unbound PVCs",
		zap.String("namespace", namespace),
		zap.Int("total_pvcs", len(pvcs)),
		zap.Int("unbound_pvcs", len(unbound)))

	return unbound, nil
}

// ListNamespaces lists all namespaces
func (c *client) ListNamespaces(ctx context.Context) ([]corev1.Namespace, error) {
	var nsList *corev1.NamespaceList
	
	err := retry.OnError(
		retry.DefaultRetry,
		func(err error) bool {
			return err != nil
		},
		func() error {
			var err error
			nsList, err = c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
			return err
		},
	)
	
	if err != nil {
		c.logger.Error("Failed to list namespaces after retries", zap.Error(err))
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	c.logger.LogK8sOperation("list", "namespaces", "", "", nil)
	
	return nsList.Items, nil
}

// GetCSIDriverPods lists pods for CSI drivers in the specified namespace
func (c *client) GetCSIDriverPods(ctx context.Context, namespace string) ([]corev1.Pod, error) {
	pods, err := c.ListPods(ctx, namespace)
	if err != nil {
		return nil, err
	}

	var csiPods []corev1.Pod
	for _, pod := range pods {
		// Look for CSI-related pods based on labels or names
		if isCSIDriverPod(pod) {
			csiPods = append(csiPods, pod)
		}
	}

	c.logger.Info("Found CSI driver pods",
		zap.String("namespace", namespace),
		zap.Int("total_pods", len(pods)),
		zap.Int("csi_pods", len(csiPods)))

	return csiPods, nil
}

// Stub implementations for missing methods
func (c *client) ValidateRBACPermissions(ctx context.Context) (*RBACValidationResult, error) {
	// TODO: Implement RBAC validation
	return &RBACValidationResult{
		HasRequiredPermissions: true,
		MissingPermissions:     []string{},
		PermissionChecks:       map[string]bool{},
		ServiceAccount:         "default",
		Namespace:              "default",
	}, nil
}

func (c *client) GetClusterInfo(ctx context.Context) (*ClusterInfo, error) {
	// TODO: Implement cluster info gathering
	return &ClusterInfo{
		Version:        "unknown",
		Platform:       "unknown",
		NodeCount:      0,
		NamespaceCount: 0,
		StorageClasses: []string{},
		CSIDrivers:     []string{},
		DemocraticCSI:  false,
		Capabilities:   map[string]bool{},
	}, nil
}

func (c *client) ListCSINodes(ctx context.Context) ([]storagev1.CSINode, error) {
	// TODO: Implement CSI node listing
	return []storagev1.CSINode{}, nil
}

func (c *client) ListCSIDrivers(ctx context.Context) ([]storagev1.CSIDriver, error) {
	// TODO: Implement CSI driver listing
	return []storagev1.CSIDriver{}, nil
}

func (c *client) ListVolumeAttachments(ctx context.Context) ([]storagev1.VolumeAttachment, error) {
	// TODO: Implement volume attachment listing
	return []storagev1.VolumeAttachment{}, nil
}

func (c *client) ListPersistentVolumeClaimsByStorageClass(ctx context.Context, namespace, storageClass string) ([]corev1.PersistentVolumeClaim, error) {
	// TODO: Implement PVC filtering by storage class
	return c.ListPersistentVolumeClaims(ctx, namespace)
}

// Helper functions

// isDemocraticCSIDriver checks if the driver name indicates democratic-csi
func isDemocraticCSIDriver(driverName string) bool {
	democraticCSIDrivers := []string{
		"org.democratic-csi.iscsi",
		"org.democratic-csi.nfs",
		"org.democratic-csi.smb",
		"democratic-csi",
	}
	
	for _, driver := range democraticCSIDrivers {
		if driverName == driver {
			return true
		}
	}
	return false
}

// isCSIDriverPod checks if a pod is a CSI driver pod
func isCSIDriverPod(pod corev1.Pod) bool {
	// Check labels for CSI-related components
	labels := pod.Labels
	if labels == nil {
		return false
	}
	
	for k, v := range labels {
		if k == "app" && v == "csi-driver" ||
		   k == "component" && v == "csi-driver" ||
		   k == "app.kubernetes.io/component" && v == "csi-driver" ||
		   v == "democratic-csi" {
			return true
		}
	}
	
	// Check pod name patterns
	csiNamePatterns := []string{
		"csi-",
		"democratic-csi",
	}
	
	for _, pattern := range csiNamePatterns {
		if len(pod.Name) >= len(pattern) && pod.Name[:len(pattern)] == pattern {
			return true
		}
	}
	
	return false
}