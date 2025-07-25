package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/tomazb/kubernetes-truenas-democratic-tool/internal/config"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/internal/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

// Client represents a Kubernetes client
type Client struct {
	clientset kubernetes.Interface
	config    *config.OpenShiftConfig
}

// NewClient creates a new Kubernetes client
func NewClient(cfg *config.OpenShiftConfig) (*Client, error) {
	var kubeconfig *rest.Config
	var err error

	if cfg.Kubeconfig != "" {
		// Use provided kubeconfig
		kubeconfig, err = clientcmd.BuildConfigFromFlags("", cfg.Kubeconfig)
	} else {
		// Try in-cluster config first
		kubeconfig, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to default kubeconfig location
			if home := homedir.HomeDir(); home != "" {
				kubeconfigPath := filepath.Join(home, ".kube", "config")
				kubeconfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Client{
		clientset: clientset,
		config:    cfg,
	}, nil
}

// TestConnection tests the connection to the Kubernetes cluster
func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		return fmt.Errorf("failed to connect to kubernetes cluster: %w", err)
	}
	return nil
}

// GetPersistentVolumes returns all persistent volumes
func (c *Client) GetPersistentVolumes(ctx context.Context) ([]types.PersistentVolumeInfo, error) {
	pvList, err := c.clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list persistent volumes: %w", err)
	}

	var pvs []types.PersistentVolumeInfo
	for _, pv := range pvList.Items {
		pvInfo := types.PersistentVolumeInfo{
			Name:         pv.Name,
			Phase:        string(pv.Status.Phase),
			CreationTime: pv.CreationTimestamp.Time,
			Labels:       pv.Labels,
			Annotations:  pv.Annotations,
		}

		if pv.Spec.Capacity != nil {
			if storage, ok := pv.Spec.Capacity[corev1.ResourceStorage]; ok {
				pvInfo.Capacity = storage.String()
			}
		}

		if pv.Spec.StorageClassName != "" {
			pvInfo.StorageClass = pv.Spec.StorageClassName
		}

		if pv.Spec.ClaimRef != nil {
			pvInfo.ClaimName = pv.Spec.ClaimRef.Name
			pvInfo.ClaimNamespace = pv.Spec.ClaimRef.Namespace
		}

		pvs = append(pvs, pvInfo)
	}

	return pvs, nil
}

// GetPersistentVolumeClaims returns all persistent volume claims
func (c *Client) GetPersistentVolumeClaims(ctx context.Context, namespace string) ([]types.PersistentVolumeClaimInfo, error) {
	var pvcList *corev1.PersistentVolumeClaimList
	var err error

	if namespace != "" {
		pvcList, err = c.clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	} else {
		pvcList, err = c.clientset.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list persistent volume claims: %w", err)
	}

	var pvcs []types.PersistentVolumeClaimInfo
	for _, pvc := range pvcList.Items {
		pvcInfo := types.PersistentVolumeClaimInfo{
			Name:         pvc.Name,
			Namespace:    pvc.Namespace,
			Phase:        string(pvc.Status.Phase),
			CreationTime: pvc.CreationTimestamp.Time,
			Labels:       pvc.Labels,
			Annotations:  pvc.Annotations,
		}

		if pvc.Spec.Resources.Requests != nil {
			if storage, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
				pvcInfo.Capacity = storage.String()
			}
		}

		if pvc.Spec.StorageClassName != nil {
			pvcInfo.StorageClass = *pvc.Spec.StorageClassName
		}

		if pvc.Spec.VolumeName != "" {
			pvcInfo.VolumeName = pvc.Spec.VolumeName
		}

		pvcs = append(pvcs, pvcInfo)
	}

	return pvcs, nil
}

// GetStorageClasses returns all storage classes (commented out - not used in monitor)
/*
func (c *Client) GetStorageClasses(ctx context.Context) ([]types.StorageClassInfo, error) {
	scList, err := c.clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list storage classes: %w", err)
	}

	var storageClasses []types.StorageClassInfo
	for _, sc := range scList.Items {
		scInfo := types.StorageClassInfo{
			Name:        sc.Name,
			Provisioner: sc.Provisioner,
			Parameters:  sc.Parameters,
			Labels:      sc.Labels,
			Annotations: sc.Annotations,
		}
		storageClasses = append(storageClasses, scInfo)
	}

	return storageClasses, nil
}
*/

// FindOrphanedPVs finds persistent volumes that appear to be orphaned
func (c *Client) FindOrphanedPVs(ctx context.Context, thresholdAge time.Duration) ([]types.OrphanedResource, error) {
	pvs, err := c.GetPersistentVolumes(ctx)
	if err != nil {
		return nil, err
	}

	var orphaned []types.OrphanedResource
	threshold := time.Now().Add(-thresholdAge)

	for _, pv := range pvs {
		// Check if PV is in Released state and older than threshold
		if pv.Phase == "Released" && pv.CreationTime.Before(threshold) {
			orphaned = append(orphaned, types.OrphanedResource{
				Name:         pv.Name,
				Type:         types.ResourceTypePV,
				CreationTime: pv.CreationTime,
				Reason:       "PV is in Released state and older than threshold",
				Details: map[string]string{
					"phase":         pv.Phase,
					"storage_class": pv.StorageClass,
					"capacity":      pv.Capacity,
				},
				Metadata: pv.Labels,
			})
		}

		// Check if PV is Available for too long
		if pv.Phase == "Available" && pv.CreationTime.Before(threshold) {
			orphaned = append(orphaned, types.OrphanedResource{
				Name:         pv.Name,
				Type:         types.ResourceTypePV,
				CreationTime: pv.CreationTime,
				Reason:       "PV is Available but not bound for extended period",
				Details: map[string]string{
					"phase":         pv.Phase,
					"storage_class": pv.StorageClass,
					"capacity":      pv.Capacity,
				},
				Metadata: pv.Labels,
			})
		}
	}

	return orphaned, nil
}

// FindOrphanedPVCs finds persistent volume claims that appear to be orphaned
func (c *Client) FindOrphanedPVCs(ctx context.Context, thresholdAge time.Duration) ([]types.OrphanedResource, error) {
	pvcs, err := c.GetPersistentVolumeClaims(ctx, "")
	if err != nil {
		return nil, err
	}

	var orphaned []types.OrphanedResource
	threshold := time.Now().Add(-thresholdAge)

	for _, pvc := range pvcs {
		// Check if PVC is in Pending state for too long
		if pvc.Phase == "Pending" && pvc.CreationTime.Before(threshold) {
			orphaned = append(orphaned, types.OrphanedResource{
				Name:         pvc.Name,
				Namespace:    pvc.Namespace,
				Type:         types.ResourceTypePVC,
				CreationTime: pvc.CreationTime,
				Reason:       "PVC is in Pending state for extended period",
				Details: map[string]string{
					"phase":         pvc.Phase,
					"storage_class": pvc.StorageClass,
					"capacity":      pvc.Capacity,
				},
				Metadata: pvc.Labels,
			})
		}

		// Check if PVC is Lost
		if pvc.Phase == "Lost" {
			orphaned = append(orphaned, types.OrphanedResource{
				Name:         pvc.Name,
				Namespace:    pvc.Namespace,
				Type:         types.ResourceTypePVC,
				CreationTime: pvc.CreationTime,
				Reason:       "PVC is in Lost state",
				Details: map[string]string{
					"phase":         pvc.Phase,
					"storage_class": pvc.StorageClass,
					"capacity":      pvc.Capacity,
				},
				Metadata: pvc.Labels,
			})
		}
	}

	return orphaned, nil
}

// CheckCSIDriverHealth checks the health of the CSI driver
func (c *Client) CheckCSIDriverHealth(ctx context.Context) (*types.CSIDriverInfo, error) {
	namespace := c.config.Namespace
	if namespace == "" {
		namespace = "democratic-csi"
	}

	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
	}

	var csiPods []types.PodInfo
	healthyPods := 0

	for _, pod := range pods.Items {
		podInfo := types.PodInfo{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Node:      pod.Spec.NodeName,
			Phase:     string(pod.Status.Phase),
			Ready:     isPodReady(&pod),
			Restarts:  getPodRestarts(&pod),
		}

		if podInfo.Ready && podInfo.Phase == "Running" {
			healthyPods++
		}

		csiPods = append(csiPods, podInfo)
	}

	driverInfo := &types.CSIDriverInfo{
		Name:    c.config.CSIDriver,
		Healthy: healthyPods > 0 && healthyPods == len(csiPods),
		Pods:    csiPods,
	}

	return driverInfo, nil
}

// isPodReady checks if a pod is ready
func isPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// getPodRestarts calculates the total number of restarts for a pod
func getPodRestarts(pod *corev1.Pod) int32 {
	var restarts int32
	for _, containerStatus := range pod.Status.ContainerStatuses {
		restarts += containerStatus.RestartCount
	}
	return restarts
}

// ValidateConfiguration validates the Kubernetes configuration
func (c *Client) ValidateConfiguration(ctx context.Context) (*types.ValidationResult, error) {
	result := &types.ValidationResult{
		Valid:     true,
		Timestamp: time.Now(),
	}

	// Test connection
	connectionCheck := types.HealthCheck{
		Name:      "Kubernetes Connection",
		Timestamp: time.Now(),
	}

	if err := c.TestConnection(ctx); err != nil {
		connectionCheck.Healthy = false
		connectionCheck.Message = fmt.Sprintf("Failed to connect: %v", err)
		result.Valid = false
		result.Errors = append(result.Errors, connectionCheck.Message)
	} else {
		connectionCheck.Healthy = true
		connectionCheck.Message = "Successfully connected to Kubernetes cluster"
	}

	result.Checks = append(result.Checks, connectionCheck)

	// Check namespace
	namespaceCheck := types.HealthCheck{
		Name:      "CSI Namespace",
		Timestamp: time.Now(),
	}

	namespace := c.config.Namespace
	if namespace == "" {
		namespace = "democratic-csi"
	}

	_, err := c.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		namespaceCheck.Healthy = false
		namespaceCheck.Message = fmt.Sprintf("Namespace %s not found", namespace)
		result.Valid = false
		result.Errors = append(result.Errors, namespaceCheck.Message)
	} else {
		namespaceCheck.Healthy = true
		namespaceCheck.Message = fmt.Sprintf("Namespace %s exists", namespace)
	}

	result.Checks = append(result.Checks, namespaceCheck)

	// Check RBAC permissions
	rbacCheck := types.HealthCheck{
		Name:      "RBAC Permissions",
		Timestamp: time.Now(),
	}

	// Test key permissions
	_, err = c.clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		rbacCheck.Healthy = false
		rbacCheck.Message = "Missing permissions to list PersistentVolumes"
		result.Valid = false
		result.Errors = append(result.Errors, rbacCheck.Message)
	} else {
		rbacCheck.Healthy = true
		rbacCheck.Message = "Required RBAC permissions granted"
	}

	result.Checks = append(result.Checks, rbacCheck)

	return result, nil
}