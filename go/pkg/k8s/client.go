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
	logger          *zap.Logger
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
	logger, _ := zap.NewProduction()

	return &client{
		clientset:      clientset,
		snapshotClient: snapshotClient,
		logger:         logger.With(zap.String("component", "k8s-client")),
		config:         config,
	}, nil
}

// ListPersistentVolumes lists all persistent volumes with retry logic
func (c *client) ListPersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error) {
	start := time.Now()
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
		c.logger.WithError(err).Error("Failed to list persistent volumes after retries")
		return nil, fmt.Errorf("failed to list persistent volumes: %w", err)
	}

	duration := time.Since(start)
	c.logger.Info("Kubernetes operation completed",
		zap.String("operation", "list"),
		zap.String("resource", "persistentvolumes"),
		zap.Int("count", len(pvList.Items)),
		zap.Int64("duration_ms", duration.Milliseconds()))
	
	return pvList.Items, nil
}

// ListPersistentVolumeClaims lists persistent volume claims in a namespace with retry logic
func (c *client) ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]corev1.PersistentVolumeClaim, error) {
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	start := time.Now()
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
		c.logger.WithError(err).WithFields(map[string]interface{}{
			"namespace": namespace,
		}).Error("Failed to list persistent volume claims after retries")
		return nil, fmt.Errorf("failed to list persistent volume claims: %w", err)
	}

	duration := time.Since(start)
	c.logger.LogK8sOperation("list", "persistentvolumeclaims", namespace, len(pvcList.Items), duration.Milliseconds())
	
	return pvcList.Items, nil
}

// ListVolumeSnapshots lists volume snapshots in a namespace with retry logic
func (c *client) ListVolumeSnapshots(ctx context.Context, namespace string) ([]snapshotv1.VolumeSnapshot, error) {
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	start := time.Now()
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
		c.logger.WithError(err).WithFields(map[string]interface{}{
			"namespace": namespace,
		}).Error("Failed to list volume snapshots after retries")
		return nil, fmt.Errorf("failed to list volume snapshots: %w", err)
	}

	duration := time.Since(start)
	c.logger.LogK8sOperation("list", "volumesnapshots", namespace, len(snapshotList.Items), duration.Milliseconds())
	
	return snapshotList.Items, nil
}

// ListStorageClasses lists all storage classes with retry logic
func (c *client) ListStorageClasses(ctx context.Context) ([]storagev1.StorageClass, error) {
	start := time.Now()
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
		c.logger.WithError(err).Error("Failed to list storage classes after retries")
		return nil, fmt.Errorf("failed to list storage classes: %w", err)
	}

	duration := time.Since(start)
	c.logger.LogK8sOperation("list", "storageclasses", "", len(scList.Items), duration.Milliseconds())
	
	return scList.Items, nil
}

// ListPods lists pods in a namespace with retry logic
func (c *client) ListPods(ctx context.Context, namespace string) ([]corev1.Pod, error) {
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	start := time.Now()
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
		c.logger.WithError(err).WithFields(map[string]interface{}{
			"namespace": namespace,
		}).Error("Failed to list pods after retries")
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	duration := time.Since(start)
	c.logger.LogK8sOperation("list", "pods", namespace, len(podList.Items), duration.Milliseconds())
	
	return podList.Items, nil
}

// GetNamespace gets a specific namespace with retry logic
func (c *client) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	start := time.Now()
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
		c.logger.WithError(err).WithFields(map[string]interface{}{
			"namespace": name,
		}).Error("Failed to get namespace after retries")
		return nil, fmt.Errorf("failed to get namespace %s: %w", name, err)
	}

	duration := time.Since(start)
	c.logger.LogK8sOperation("get", "namespace", name, 1, duration.Milliseconds())
	
	return namespace, nil
}

// TestConnection tests the Kubernetes connection with retry logic
func (c *client) TestConnection(ctx context.Context) error {
	start := time.Now()
	
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
		c.logger.WithError(err).Error("Failed to connect to Kubernetes API after retries")
		return fmt.Errorf("failed to connect to Kubernetes API: %w", err)
	}

	duration := time.Since(start)
	c.logger.WithFields(map[string]interface{}{
		"duration_ms": duration.Milliseconds(),
	}).Info("Kubernetes connection test successful")
	
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

	c.logger.WithFields(map[string]interface{}{
		"total_pvs":         len(pvs),
		"democratic_csi_pvs": len(filtered),
	}).Info("Filtered PVs by democratic-csi driver")

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
		if pv.Spec.StorageClassName != nil && *pv.Spec.StorageClassName == storageClass {
			filtered = append(filtered, pv)
		}
	}

	c.logger.WithFields(map[string]interface{}{
		"storage_class": storageClass,
		"total_pvs":     len(pvs),
		"filtered_pvs":  len(filtered),
	}).Info("Filtered PVs by storage class")

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

	c.logger.WithFields(map[string]interface{}{
		"namespace":    namespace,
		"total_pvcs":   len(pvcs),
		"unbound_pvcs": len(unbound),
	}).Info("Found unbound PVCs")

	return unbound, nil
}

// ListNamespaces lists all namespaces
func (c *client) ListNamespaces(ctx context.Context) ([]corev1.Namespace, error) {
	start := time.Now()
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
		c.logger.WithError(err).Error("Failed to list namespaces after retries")
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	duration := time.Since(start)
	c.logger.LogK8sOperation("list", "namespaces", "", len(nsList.Items), duration.Milliseconds())
	
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

	c.logger.WithFields(map[string]interface{}{
		"namespace":  namespace,
		"total_pods": len(pods),
		"csi_pods":   len(csiPods),
	}).Info("Found CSI driver pods")

	return csiPods, nil
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

	// Also check for TrueNAS-related drivers
	return containsIgnoreCase(driverName, "truenas") || 
		   containsIgnoreCase(driverName, "democratic")
}

// isCSIDriverPod checks if a pod is related to CSI drivers
func isCSIDriverPod(pod corev1.Pod) bool {
	// Check pod name patterns
	csiPodPatterns := []string{
		"csi-",
		"democratic-csi",
		"truenas-csi",
	}

	podName := pod.Name
	for _, pattern := range csiPodPatterns {
		if containsIgnoreCase(podName, pattern) {
			return true
		}
	}

	// Check labels
	if pod.Labels != nil {
		for key, value := range pod.Labels {
			if containsIgnoreCase(key, "csi") || containsIgnoreCase(value, "csi") ||
			   containsIgnoreCase(key, "democratic") || containsIgnoreCase(value, "democratic") {
				return true
			}
		}
	}

	return false
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr ||
		      containsSubstring(s, substr))))
}

// containsSubstring is a simple substring search
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ValidateRBACPermissions validates that the client has required RBAC permissions
func (c *client) ValidateRBACPermissions(ctx context.Context) (*RBACValidationResult, error) {
	result := &RBACValidationResult{
		PermissionChecks: make(map[string]bool),
	}

	// Get current service account info
	if c.config.InCluster {
		result.ServiceAccount = "system:serviceaccount:truenas-monitor:truenas-monitor"
		result.Namespace = "truenas-monitor"
	} else {
		result.ServiceAccount = "current-user"
		result.Namespace = "default"
	}

	// Define required permissions for monitoring
	requiredPermissions := map[string]struct {
		resource string
		verbs    []string
	}{
		"persistentvolumes":       {"persistentvolumes", []string{"get", "list", "watch"}},
		"persistentvolumeclaims":  {"persistentvolumeclaims", []string{"get", "list", "watch"}},
		"volumesnapshots":         {"volumesnapshots", []string{"get", "list", "watch"}},
		"storageclasses":          {"storageclasses", []string{"get", "list"}},
		"csinodes":                {"csinodes", []string{"get", "list"}},
		"csidrivers":              {"csidrivers", []string{"get", "list"}},
		"pods":                    {"pods", []string{"get", "list"}},
		"namespaces":              {"namespaces", []string{"get", "list"}},
	}

	allPermissionsValid := true
	var missingPermissions []string

	// Test each required permission
	for permName, perm := range requiredPermissions {
		hasPermission := true
		
		// Try to list the resource to test permissions
		switch perm.resource {
		case "persistentvolumes":
			_, err := c.ListPersistentVolumes(ctx)
			hasPermission = err == nil
		case "persistentvolumeclaims":
			_, err := c.ListPersistentVolumeClaims(ctx, "")
			hasPermission = err == nil
		case "volumesnapshots":
			_, err := c.ListVolumeSnapshots(ctx, "")
			hasPermission = err == nil
		case "storageclasses":
			_, err := c.ListStorageClasses(ctx)
			hasPermission = err == nil
		case "pods":
			_, err := c.ListPods(ctx, "kube-system")
			hasPermission = err == nil
		case "namespaces":
			_, err := c.ListNamespaces(ctx)
			hasPermission = err == nil
		}

		result.PermissionChecks[permName] = hasPermission
		if !hasPermission {
			allPermissionsValid = false
			missingPermissions = append(missingPermissions, permName)
		}
	}

	result.HasRequiredPermissions = allPermissionsValid
	result.MissingPermissions = missingPermissions

	c.logger.WithFields(map[string]interface{}{
		"has_permissions":     allPermissionsValid,
		"missing_permissions": len(missingPermissions),
		"service_account":     result.ServiceAccount,
	}).Info("RBAC permissions validation completed")

	return result, nil
}

// GetClusterInfo retrieves comprehensive cluster information
func (c *client) GetClusterInfo(ctx context.Context) (*ClusterInfo, error) {
	info := &ClusterInfo{
		Capabilities: make(map[string]bool),
	}

	// Get cluster version
	version, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		c.logger.WithError(err).Warn("Failed to get cluster version")
		info.Version = "unknown"
	} else {
		info.Version = version.String()
	}

	// Detect platform
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
	if err == nil && len(nodes.Items) > 0 {
		node := nodes.Items[0]
		if node.Status.NodeInfo.OSImage != "" {
			if containsIgnoreCase(node.Status.NodeInfo.OSImage, "openshift") {
				info.Platform = "OpenShift"
			} else if containsIgnoreCase(node.Status.NodeInfo.OSImage, "kubernetes") {
				info.Platform = "Kubernetes"
			} else {
				info.Platform = "Unknown"
			}
		}
	}

	// Count nodes
	allNodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err == nil {
		info.NodeCount = len(allNodes.Items)
	}

	// Count namespaces
	namespaces, err := c.ListNamespaces(ctx)
	if err == nil {
		info.NamespaceCount = len(namespaces)
	}

	// Get storage classes
	storageClasses, err := c.ListStorageClasses(ctx)
	if err == nil {
		for _, sc := range storageClasses {
			info.StorageClasses = append(info.StorageClasses, sc.Name)
		}
	}

	// Check for democratic-csi
	democraticCSIFound := false
	for _, sc := range storageClasses {
		if isDemocraticCSIDriver(sc.Provisioner) {
			democraticCSIFound = true
			break
		}
	}
	info.DemocraticCSI = democraticCSIFound

	// Check capabilities
	info.Capabilities["volume_snapshots"] = c.hasVolumeSnapshotSupport(ctx)
	info.Capabilities["csi_drivers"] = c.hasCSISupport(ctx)
	info.Capabilities["storage_classes"] = len(info.StorageClasses) > 0

	c.logger.WithFields(map[string]interface{}{
		"version":         info.Version,
		"platform":        info.Platform,
		"nodes":           info.NodeCount,
		"namespaces":      info.NamespaceCount,
		"storage_classes": len(info.StorageClasses),
		"democratic_csi":  info.DemocraticCSI,
	}).Info("Cluster information gathered")

	return info, nil
}

// ListCSINodes lists all CSI nodes
func (c *client) ListCSINodes(ctx context.Context) ([]storagev1.CSINode, error) {
	start := time.Now()
	var csiNodeList *storagev1.CSINodeList
	
	err := retry.OnError(
		retry.DefaultRetry,
		func(err error) bool {
			return err != nil
		},
		func() error {
			var err error
			csiNodeList, err = c.clientset.StorageV1().CSINodes().List(ctx, metav1.ListOptions{})
			return err
		},
	)
	
	if err != nil {
		c.logger.WithError(err).Error("Failed to list CSI nodes after retries")
		return nil, fmt.Errorf("failed to list CSI nodes: %w", err)
	}

	duration := time.Since(start)
	c.logger.LogK8sOperation("list", "csinodes", "", len(csiNodeList.Items), duration.Milliseconds())
	
	return csiNodeList.Items, nil
}

// ListCSIDrivers lists all CSI drivers
func (c *client) ListCSIDrivers(ctx context.Context) ([]storagev1.CSIDriver, error) {
	start := time.Now()
	var csiDriverList *storagev1.CSIDriverList
	
	err := retry.OnError(
		retry.DefaultRetry,
		func(err error) bool {
			return err != nil
		},
		func() error {
			var err error
			csiDriverList, err = c.clientset.StorageV1().CSIDrivers().List(ctx, metav1.ListOptions{})
			return err
		},
	)
	
	if err != nil {
		c.logger.WithError(err).Error("Failed to list CSI drivers after retries")
		return nil, fmt.Errorf("failed to list CSI drivers: %w", err)
	}

	duration := time.Since(start)
	c.logger.LogK8sOperation("list", "csidrivers", "", len(csiDriverList.Items), duration.Milliseconds())
	
	return csiDriverList.Items, nil
}

// ListVolumeAttachments lists all volume attachments
func (c *client) ListVolumeAttachments(ctx context.Context) ([]storagev1.VolumeAttachment, error) {
	start := time.Now()
	var vaList *storagev1.VolumeAttachmentList
	
	err := retry.OnError(
		retry.DefaultRetry,
		func(err error) bool {
			return err != nil
		},
		func() error {
			var err error
			vaList, err = c.clientset.StorageV1().VolumeAttachments().List(ctx, metav1.ListOptions{})
			return err
		},
	)
	
	if err != nil {
		c.logger.WithError(err).Error("Failed to list volume attachments after retries")
		return nil, fmt.Errorf("failed to list volume attachments: %w", err)
	}

	duration := time.Since(start)
	c.logger.LogK8sOperation("list", "volumeattachments", "", len(vaList.Items), duration.Milliseconds())
	
	return vaList.Items, nil
}

// ListPersistentVolumeClaimsByStorageClass lists PVCs filtered by storage class
func (c *client) ListPersistentVolumeClaimsByStorageClass(ctx context.Context, namespace, storageClass string) ([]corev1.PersistentVolumeClaim, error) {
	pvcs, err := c.ListPersistentVolumeClaims(ctx, namespace)
	if err != nil {
		return nil, err
	}

	var filtered []corev1.PersistentVolumeClaim
	for _, pvc := range pvcs {
		if pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName == storageClass {
			filtered = append(filtered, pvc)
		}
	}

	c.logger.WithFields(map[string]interface{}{
		"namespace":     namespace,
		"storage_class": storageClass,
		"total_pvcs":    len(pvcs),
		"filtered_pvcs": len(filtered),
	}).Info("Filtered PVCs by storage class")

	return filtered, nil
}

// Helper methods for capability detection

func (c *client) hasVolumeSnapshotSupport(ctx context.Context) bool {
	_, err := c.ListVolumeSnapshots(ctx, "")
	return err == nil
}

func (c *client) hasCSISupport(ctx context.Context) bool {
	_, err := c.ListCSIDrivers(ctx)
	return err == nil
}