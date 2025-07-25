package k8s

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/tomazb/kubernetes-truenas-democratic-tool/internal/types"
)

// Client implements the Kubernetes client interface
type Client struct {
	clientset     kubernetes.Interface
	dynamicClient dynamic.Interface
	config        *Config
}

// NewClient creates a new Kubernetes client
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Set defaults
	if config.ResyncPeriod == 0 {
		config.ResyncPeriod = 30 * time.Minute
	}

	var restConfig *rest.Config
	var err error

	if config.InCluster {
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
		}
	} else {
		kubeconfig := config.Kubeconfig
		if kubeconfig == "" {
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			}
		}

		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create config from kubeconfig: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &Client{
		clientset:     clientset,
		dynamicClient: dynamicClient,
		config:        config,
	}, nil
}

// GetPersistentVolumes returns all PersistentVolumes filtered by CSI driver
func (c *Client) GetPersistentVolumes(ctx context.Context) ([]*v1.PersistentVolume, error) {
	pvList, err := c.clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list PVs: %w", err)
	}

	var filteredPVs []*v1.PersistentVolume
	for i := range pvList.Items {
		pv := &pvList.Items[i]
		if c.isPVManaged(pv) {
			filteredPVs = append(filteredPVs, pv)
		}
	}

	return filteredPVs, nil
}

// GetPersistentVolume returns a specific PersistentVolume by name
func (c *Client) GetPersistentVolume(ctx context.Context, name string) (*v1.PersistentVolume, error) {
	pv, err := c.clientset.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get PV %s: %w", name, err)
	}

	if !c.isPVManaged(pv) {
		return nil, fmt.Errorf("PV %s is not managed by %s", name, c.config.CSIDriver)
	}

	return pv, nil
}

// WatchPersistentVolumes watches for PersistentVolume changes
func (c *Client) WatchPersistentVolumes(ctx context.Context, eventCh chan<- PVEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			watcher, err := c.clientset.CoreV1().PersistentVolumes().Watch(ctx, metav1.ListOptions{})
			if err != nil {
				eventCh <- PVEvent{Type: EventError, PV: nil}
				time.Sleep(5 * time.Second)
				continue
			}

			c.processPVEvents(ctx, watcher, eventCh)
		}
	}
}

// GetPersistentVolumeClaims returns all PVCs filtered by namespace and storage class
func (c *Client) GetPersistentVolumeClaims(ctx context.Context) ([]*v1.PersistentVolumeClaim, error) {
	namespace := c.config.Namespace
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	pvcList, err := c.clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list PVCs: %w", err)
	}

	var filteredPVCs []*v1.PersistentVolumeClaim
	for i := range pvcList.Items {
		pvc := &pvcList.Items[i]
		if c.isPVCManaged(pvc) {
			filteredPVCs = append(filteredPVCs, pvc)
		}
	}

	return filteredPVCs, nil
}

// GetPersistentVolumeClaim returns a specific PVC
func (c *Client) GetPersistentVolumeClaim(ctx context.Context, namespace, name string) (*v1.PersistentVolumeClaim, error) {
	pvc, err := c.clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get PVC %s/%s: %w", namespace, name, err)
	}

	if !c.isPVCManaged(pvc) {
		return nil, fmt.Errorf("PVC %s/%s is not managed by storage class %s", namespace, name, c.config.StorageClassName)
	}

	return pvc, nil
}

// WatchPersistentVolumeClaims watches for PVC changes
func (c *Client) WatchPersistentVolumeClaims(ctx context.Context, eventCh chan<- PVCEvent) {
	namespace := c.config.Namespace
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			watcher, err := c.clientset.CoreV1().PersistentVolumeClaims(namespace).Watch(ctx, metav1.ListOptions{})
			if err != nil {
				eventCh <- PVCEvent{Type: EventError, PVC: nil}
				time.Sleep(5 * time.Second)
				continue
			}

			c.processPVCEvents(ctx, watcher, eventCh)
		}
	}
}

// GetVolumeSnapshots returns all VolumeSnapshots in the configured namespace
func (c *Client) GetVolumeSnapshots(ctx context.Context) ([]*snapshotv1.VolumeSnapshot, error) {
	if c.dynamicClient == nil {
		return nil, fmt.Errorf("dynamic client not initialized")
	}
	
	namespace := c.config.Namespace
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	gvr := schema.GroupVersionResource{
		Group:    "snapshot.storage.k8s.io",
		Version:  "v1beta1",
		Resource: "volumesnapshots",
	}

	var snapshots []*snapshotv1.VolumeSnapshot

	if namespace == metav1.NamespaceAll {
		unstructuredList, err := c.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list snapshots: %w", err)
		}

		for _, item := range unstructuredList.Items {
			snapshot := &snapshotv1.VolumeSnapshot{}
			if err := convertUnstructured(&item, snapshot); err != nil {
				continue
			}
			snapshots = append(snapshots, snapshot)
		}
	} else {
		unstructuredList, err := c.dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list snapshots in namespace %s: %w", namespace, err)
		}

		for _, item := range unstructuredList.Items {
			snapshot := &snapshotv1.VolumeSnapshot{}
			if err := convertUnstructured(&item, snapshot); err != nil {
				continue
			}
			snapshots = append(snapshots, snapshot)
		}
	}

	return snapshots, nil
}

// ValidateConfiguration validates the Kubernetes client configuration and connectivity
func (c *Client) ValidateConfiguration(ctx context.Context) (*types.ValidationResult, error) {
	result := &types.ValidationResult{
		Valid:     true,
		Timestamp: time.Now(),
		Checks:    []types.HealthCheck{},
		Errors:    []string{},
		Warnings:  []string{},
	}

	// Test basic API connectivity
	_, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to connect to Kubernetes API: %v", err))
		result.Checks = append(result.Checks, types.HealthCheck{
			Name:      "kubernetes_api_connectivity",
			Healthy:   false,
			Message:   fmt.Sprintf("Cannot reach Kubernetes API server: %v", err),
			Timestamp: time.Now(),
		})
		return result, nil
	}

	result.Checks = append(result.Checks, types.HealthCheck{
		Name:      "kubernetes_api_connectivity", 
		Healthy:   true,
		Message:   "Successfully connected to Kubernetes API server",
		Timestamp: time.Now(),
	})

	// Test namespace access
	namespace := c.config.Namespace
	if namespace == "" {
		namespace = "default"
	}

	_, err = c.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to access namespace '%s': %v", namespace, err))
		result.Checks = append(result.Checks, types.HealthCheck{
			Name:      "namespace_access",
			Healthy:   false, 
			Message:   fmt.Sprintf("Cannot access namespace '%s': %v", namespace, err),
			Timestamp: time.Now(),
		})
		return result, nil
	}

	result.Checks = append(result.Checks, types.HealthCheck{
		Name:      "namespace_access",
		Healthy:   true,
		Message:   fmt.Sprintf("Successfully accessed namespace '%s'", namespace),
		Timestamp: time.Now(),
	})

	// Test storage classes access (for CSI functionality)
	_, err = c.clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Cannot list storage classes: %v", err))
		result.Checks = append(result.Checks, types.HealthCheck{
			Name:      "storage_classes_access",
			Healthy:   false,
			Message:   fmt.Sprintf("Limited storage class access: %v", err),
			Timestamp: time.Now(),
		})
	} else {
		result.Checks = append(result.Checks, types.HealthCheck{
			Name:      "storage_classes_access",
			Healthy:   true,
			Message:   "Successfully accessed storage classes",
			Timestamp: time.Now(),
		})
	}

	// Test volume snapshot access (for snapshot functionality)
	if c.dynamicClient != nil {
		gvr := schema.GroupVersionResource{
			Group:    "snapshot.storage.k8s.io",
			Version:  "v1beta1",
			Resource: "volumesnapshots",
		}
		
		_, err = c.dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{Limit: 1})
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Cannot access volume snapshots: %v", err))
			result.Checks = append(result.Checks, types.HealthCheck{
				Name:      "volume_snapshots_access",
				Healthy:   false,
				Message:   fmt.Sprintf("Volume snapshot functionality may be limited: %v", err),
				Timestamp: time.Now(),
			})
		} else {
			result.Checks = append(result.Checks, types.HealthCheck{
				Name:      "volume_snapshots_access",
				Healthy:   true,
				Message:   "Successfully accessed volume snapshots",
				Timestamp: time.Now(),
			})
		}
	}

	return result, nil
}

// GetVolumeSnapshot returns a specific VolumeSnapshot
func (c *Client) GetVolumeSnapshot(ctx context.Context, namespace, name string) (*snapshotv1.VolumeSnapshot, error) {
	gvr := schema.GroupVersionResource{
		Group:    "snapshot.storage.k8s.io",
		Version:  "v1beta1",
		Resource: "volumesnapshots",
	}

	unstructured, err := c.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot %s/%s: %w", namespace, name, err)
	}

	snapshot := &snapshotv1.VolumeSnapshot{}
	if err := convertUnstructured(unstructured, snapshot); err != nil {
		return nil, fmt.Errorf("failed to convert snapshot: %w", err)
	}

	return snapshot, nil
}

// WatchVolumeSnapshots watches for VolumeSnapshot changes
func (c *Client) WatchVolumeSnapshots(ctx context.Context, eventCh chan<- SnapshotEvent) {
	namespace := c.config.Namespace
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	gvr := schema.GroupVersionResource{
		Group:    "snapshot.storage.k8s.io",
		Version:  "v1beta1",
		Resource: "volumesnapshots",
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			var watcher watch.Interface
			var err error

			if namespace == metav1.NamespaceAll {
				watcher, err = c.dynamicClient.Resource(gvr).Watch(ctx, metav1.ListOptions{})
			} else {
				watcher, err = c.dynamicClient.Resource(gvr).Namespace(namespace).Watch(ctx, metav1.ListOptions{})
			}

			if err != nil {
				eventCh <- SnapshotEvent{Type: EventError, Snapshot: nil}
				time.Sleep(5 * time.Second)
				continue
			}

			c.processSnapshotEvents(ctx, watcher, eventCh)
		}
	}
}

// GetStorageClasses returns all StorageClasses filtered by provisioner
func (c *Client) GetStorageClasses(ctx context.Context) ([]*storagev1.StorageClass, error) {
	scList, err := c.clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list storage classes: %w", err)
	}

	var filteredSCs []*storagev1.StorageClass
	for i := range scList.Items {
		sc := &scList.Items[i]
		if c.config.CSIDriver == "" || sc.Provisioner == c.config.CSIDriver {
			filteredSCs = append(filteredSCs, sc)
		}
	}

	return filteredSCs, nil
}

// GetStorageClass returns a specific StorageClass
func (c *Client) GetStorageClass(ctx context.Context, name string) (*storagev1.StorageClass, error) {
	sc, err := c.clientset.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get storage class %s: %w", name, err)
	}

	return sc, nil
}

// GetCSINodes returns all CSI nodes
func (c *Client) GetCSINodes(ctx context.Context) ([]*storagev1.CSINode, error) {
	nodeList, err := c.clientset.StorageV1().CSINodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list CSI nodes: %w", err)
	}

	var nodes []*storagev1.CSINode
	for i := range nodeList.Items {
		nodes = append(nodes, &nodeList.Items[i])
	}

	return nodes, nil
}

// GetVolumeAttachments returns all VolumeAttachments
func (c *Client) GetVolumeAttachments(ctx context.Context) ([]*storagev1.VolumeAttachment, error) {
	vaList, err := c.clientset.StorageV1().VolumeAttachments().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list volume attachments: %w", err)
	}

	var attachments []*storagev1.VolumeAttachment
	for i := range vaList.Items {
		va := &vaList.Items[i]
		if c.config.CSIDriver == "" || va.Spec.Attacher == c.config.CSIDriver {
			attachments = append(attachments, va)
		}
	}

	return attachments, nil
}

// CheckCSIDriverHealth checks if the CSI driver is healthy
func (c *Client) CheckCSIDriverHealth(ctx context.Context) error {
	pods, err := c.GetCSIDriverPods(ctx)
	if err != nil {
		return fmt.Errorf("failed to get CSI driver pods: %w", err)
	}

	if len(pods) == 0 {
		return fmt.Errorf("no CSI driver pods found")
	}

	for _, pod := range pods {
		if pod.Status.Phase != v1.PodRunning {
			return fmt.Errorf("CSI driver pod %s/%s is not running: %s", pod.Namespace, pod.Name, pod.Status.Phase)
		}

		for _, container := range pod.Status.ContainerStatuses {
			if !container.Ready {
				return fmt.Errorf("container %s in pod %s/%s is not ready", container.Name, pod.Namespace, pod.Name)
			}
		}
	}

	return nil
}

// GetCSIDriverPods returns all pods related to the CSI driver
func (c *Client) GetCSIDriverPods(ctx context.Context) ([]*v1.Pod, error) {
	if c.config.CSIDriver == "" {
		return nil, fmt.Errorf("CSI driver not configured")
	}

	// Common labels used by CSI drivers
	labelSelectors := []string{
		fmt.Sprintf("app=%s", c.config.CSIDriver),
		fmt.Sprintf("app.kubernetes.io/name=%s", c.config.CSIDriver),
		"app=democratic-csi",
		"app.kubernetes.io/name=democratic-csi",
	}

	var allPods []*v1.Pod
	seen := make(map[string]bool)

	for _, selector := range labelSelectors {
		podList, err := c.clientset.CoreV1().Pods(metav1.NamespaceAll).List(ctx, metav1.ListOptions{
			LabelSelector: selector,
		})
		if err != nil {
			continue
		}

		for i := range podList.Items {
			pod := &podList.Items[i]
			key := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
			if !seen[key] {
				seen[key] = true
				allPods = append(allPods, pod)
			}
		}
	}

	return allPods, nil
}

// Helper methods

func (c *Client) isPVManaged(pv *v1.PersistentVolume) bool {
	if c.config.CSIDriver == "" {
		return true
	}

	if pv.Spec.CSI != nil && pv.Spec.CSI.Driver == c.config.CSIDriver {
		return true
	}

	// Check labels
	if pv.Labels != nil {
		if provisioner, ok := pv.Labels["provisioner"]; ok && provisioner == c.config.CSIDriver {
			return true
		}
	}

	return false
}

func (c *Client) isPVCManaged(pvc *v1.PersistentVolumeClaim) bool {
	if c.config.StorageClassName == "" {
		return true
	}

	if pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName == c.config.StorageClassName {
		return true
	}

	return false
}

func (c *Client) processPVEvents(ctx context.Context, watcher watch.Interface, eventCh chan<- PVEvent) {
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return
			}

			pv, ok := event.Object.(*v1.PersistentVolume)
			if !ok {
				continue
			}

			if !c.isPVManaged(pv) {
				continue
			}

			eventType := EventType(event.Type)
			eventCh <- PVEvent{
				Type: eventType,
				PV:   pv,
			}
		}
	}
}

func (c *Client) processPVCEvents(ctx context.Context, watcher watch.Interface, eventCh chan<- PVCEvent) {
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return
			}

			pvc, ok := event.Object.(*v1.PersistentVolumeClaim)
			if !ok {
				continue
			}

			if !c.isPVCManaged(pvc) {
				continue
			}

			eventType := EventType(event.Type)
			eventCh <- PVCEvent{
				Type: eventType,
				PVC:  pvc,
			}
		}
	}
}

func (c *Client) processSnapshotEvents(ctx context.Context, watcher watch.Interface, eventCh chan<- SnapshotEvent) {
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return
			}

			unstructured, ok := event.Object.(*unstructured.Unstructured)
			if !ok {
				continue
			}

			snapshot := &snapshotv1.VolumeSnapshot{}
			if err := convertUnstructured(unstructured, snapshot); err != nil {
				continue
			}

			eventType := EventType(event.Type)
			eventCh <- SnapshotEvent{
				Type:     eventType,
				Snapshot: snapshot,
			}
		}
	}
}

func convertUnstructured(u *unstructured.Unstructured, obj interface{}) error {
	return runtime.DefaultUnstructuredConverter.FromUnstructured(u.UnstructuredContent(), obj)
}