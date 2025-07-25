"""Kubernetes client wrapper for TrueNAS storage monitoring."""

import logging
from dataclasses import dataclass, field
from datetime import datetime, timedelta
from enum import Enum
from typing import List, Dict, Optional, Any, Generator

from kubernetes import client as k8s_client, config, watch
from kubernetes.client.rest import ApiException


logger = logging.getLogger(__name__)


class ResourceType(Enum):
    """Types of Kubernetes resources."""
    PERSISTENT_VOLUME = "PersistentVolume"
    PERSISTENT_VOLUME_CLAIM = "PersistentVolumeClaim"
    VOLUME_SNAPSHOT = "VolumeSnapshot"
    VOLUME_ATTACHMENT = "VolumeAttachment"
    STORAGE_CLASS = "StorageClass"
    CSI_NODE = "CSINode"
    POD = "Pod"


@dataclass
class K8sConfig:
    """Configuration for Kubernetes client."""
    kubeconfig: Optional[str] = None
    namespace: Optional[str] = None
    csi_driver: str = "org.democratic-csi.nfs"
    storage_class: Optional[str] = None
    in_cluster: bool = False


@dataclass
class PersistentVolumeInfo:
    """Information about a PersistentVolume."""
    name: str
    volume_handle: str
    driver: str
    capacity: str
    access_modes: List[str]
    phase: str
    claim_ref: Optional[Dict[str, str]] = None
    creation_time: Optional[datetime] = None
    labels: Dict[str, str] = field(default_factory=dict)
    annotations: Dict[str, str] = field(default_factory=dict)
    # Additional field for test compatibility
    status: Optional[str] = None
    storage_class: Optional[str] = None
    
    def __post_init__(self):
        # Map phase to status for backward compatibility
        if self.status is None:
            self.status = self.phase


@dataclass
class PersistentVolumeClaimInfo:
    """Information about a PersistentVolumeClaim."""
    name: str
    namespace: str
    storage_class: Optional[str]
    volume_name: Optional[str]
    capacity: str
    phase: str
    creation_time: Optional[datetime] = None
    labels: Dict[str, str] = field(default_factory=dict)
    annotations: Dict[str, str] = field(default_factory=dict)
    # Additional field for test compatibility
    status: Optional[str] = None
    
    def __post_init__(self):
        # Map phase to status for backward compatibility
        if self.status is None:
            self.status = self.phase


@dataclass
class VolumeSnapshotInfo:
    """Information about a VolumeSnapshot."""
    name: str
    namespace: str
    source_pvc: Optional[str]
    snapshot_class: Optional[str]
    ready_to_use: bool
    creation_time: Optional[datetime] = None
    labels: Dict[str, str] = field(default_factory=dict)
    annotations: Dict[str, str] = field(default_factory=dict)
    # Additional field for test compatibility
    size: Optional[str] = None


@dataclass
class OrphanedResource:
    """Information about an orphaned resource."""
    resource_type: ResourceType
    name: str
    namespace: Optional[str]
    volume_handle: Optional[str]
    creation_time: datetime
    size: Optional[str]
    location: str
    reason: str
    details: Dict[str, Any] = field(default_factory=dict)


class K8sClient:
    """Kubernetes client wrapper for storage operations."""

    def __init__(self, config_obj: K8sConfig):
        """Initialize Kubernetes client.
        
        Args:
            config_obj: Kubernetes client configuration
        """
        self.config = config_obj
        
        # Load Kubernetes configuration
        if config_obj.in_cluster:
            config.load_incluster_config()
        else:
            config.load_kube_config(config_file=config_obj.kubeconfig)
        
        # Initialize API clients
        self.core_v1 = k8s_client.CoreV1Api()
        self.storage_v1 = k8s_client.StorageV1Api()
        self.custom_objects = k8s_client.CustomObjectsApi()
    
    def test_connection(self) -> bool:
        """Test connection to Kubernetes cluster.
        
        Returns:
            True if connection successful, False otherwise
        """
        try:
            # Quick connectivity test - just get version info
            version = self.core_v1.get_api_resources()
            return True
        except Exception:
            return False
        
    def get_persistent_volumes(self) -> List[PersistentVolumeInfo]:
        """Get all PersistentVolumes managed by the CSI driver.
        
        Returns:
            List of PersistentVolumeInfo objects
        """
        try:
            pvs = self.core_v1.list_persistent_volume()
            result = []
            
            for pv in pvs.items:
                # Filter by CSI driver if specified
                if pv.spec.csi and pv.spec.csi.driver == self.config.csi_driver:
                    claim_ref = None
                    if pv.spec.claim_ref:
                        claim_ref = {
                            "name": pv.spec.claim_ref.name,
                            "namespace": pv.spec.claim_ref.namespace,
                        }
                    
                    pv_info = PersistentVolumeInfo(
                        name=pv.metadata.name,
                        volume_handle=pv.spec.csi.volume_handle,
                        driver=pv.spec.csi.driver,
                        capacity=pv.spec.capacity.get("storage", ""),
                        access_modes=pv.spec.access_modes,
                        phase=pv.status.phase,
                        claim_ref=claim_ref,
                        creation_time=pv.metadata.creation_timestamp,
                        labels=pv.metadata.labels or {},
                        annotations=pv.metadata.annotations or {},
                    )
                    result.append(pv_info)
            
            logger.info(f"Found {len(result)} PersistentVolumes for driver {self.config.csi_driver}")
            return result
            
        except ApiException as e:
            logger.error(f"Failed to list PersistentVolumes: {e}")
            raise

    def get_persistent_volume_claims(self, namespace: Optional[str] = None) -> List[PersistentVolumeClaimInfo]:
        """Get all PersistentVolumeClaims.
        
        Args:
            namespace: Namespace to filter by (None for all namespaces)
            
        Returns:
            List of PersistentVolumeClaimInfo objects
        """
        try:
            namespace = namespace or self.config.namespace
            
            if namespace:
                pvcs = self.core_v1.list_namespaced_persistent_volume_claim(namespace)
            else:
                pvcs = self.core_v1.list_persistent_volume_claim_for_all_namespaces()
            
            result = []
            for pvc in pvcs.items:
                # Filter by storage class if specified
                if self.config.storage_class and pvc.spec.storage_class_name != self.config.storage_class:
                    continue
                
                pvc_info = PersistentVolumeClaimInfo(
                    name=pvc.metadata.name,
                    namespace=pvc.metadata.namespace,
                    storage_class=pvc.spec.storage_class_name,
                    volume_name=pvc.spec.volume_name,
                    capacity=pvc.spec.resources.requests.get("storage", ""),
                    phase=pvc.status.phase,
                    creation_time=pvc.metadata.creation_timestamp,
                    labels=pvc.metadata.labels or {},
                    annotations=pvc.metadata.annotations or {},
                )
                result.append(pvc_info)
            
            logger.info(f"Found {len(result)} PersistentVolumeClaims")
            return result
            
        except ApiException as e:
            logger.error(f"Failed to list PersistentVolumeClaims: {e}")
            raise

    def get_volume_snapshots(self, namespace: Optional[str] = None) -> List[VolumeSnapshotInfo]:
        """Get all VolumeSnapshots.
        
        Args:
            namespace: Namespace to filter by (None for all namespaces)
            
        Returns:
            List of VolumeSnapshotInfo objects
        """
        try:
            namespace = namespace or self.config.namespace
            group = "snapshot.storage.k8s.io"
            version = "v1beta1"
            plural = "volumesnapshots"
            
            if namespace:
                snapshots = self.custom_objects.list_namespaced_custom_object(
                    group=group,
                    version=version,
                    namespace=namespace,
                    plural=plural,
                )
            else:
                snapshots = self.custom_objects.list_cluster_custom_object(
                    group=group,
                    version=version,
                    plural=plural,
                )
            
            result = []
            for snapshot in snapshots.get("items", []):
                metadata = snapshot.get("metadata", {})
                spec = snapshot.get("spec", {})
                status = snapshot.get("status", {})
                
                snapshot_info = VolumeSnapshotInfo(
                    name=metadata.get("name"),
                    namespace=metadata.get("namespace"),
                    source_pvc=spec.get("source", {}).get("persistentVolumeClaimName"),
                    snapshot_class=spec.get("volumeSnapshotClassName"),
                    ready_to_use=status.get("readyToUse", False),
                    creation_time=datetime.fromisoformat(
                        metadata.get("creationTimestamp", "").replace("Z", "+00:00")
                    ) if metadata.get("creationTimestamp") else None,
                    labels=metadata.get("labels", {}),
                    annotations=metadata.get("annotations", {}),
                )
                result.append(snapshot_info)
            
            logger.info(f"Found {len(result)} VolumeSnapshots")
            return result
            
        except ApiException as e:
            logger.error(f"Failed to list VolumeSnapshots: {e}")
            raise

    def get_storage_classes(self) -> List[Dict[str, Any]]:
        """Get all StorageClasses for the CSI driver.
        
        Returns:
            List of StorageClass information
        """
        try:
            scs = self.storage_v1.list_storage_class()
            result = []
            
            for sc in scs.items:
                # Filter by provisioner if CSI driver is specified
                if self.config.csi_driver and sc.provisioner != self.config.csi_driver:
                    continue
                
                sc_info = {
                    "name": sc.metadata.name,
                    "provisioner": sc.provisioner,
                    "parameters": sc.parameters or {},
                    "reclaim_policy": sc.reclaim_policy,
                    "volume_binding_mode": sc.volume_binding_mode,
                    "allow_volume_expansion": sc.allow_volume_expansion or False,
                }
                result.append(sc_info)
            
            logger.info(f"Found {len(result)} StorageClasses")
            return result
            
        except ApiException as e:
            logger.error(f"Failed to list StorageClasses: {e}")
            raise

    def get_csi_nodes(self) -> List[Dict[str, Any]]:
        """Get all CSINode objects.
        
        Returns:
            List of CSINode information
        """
        try:
            nodes = self.storage_v1.list_csi_node()
            result = []
            
            for node in nodes.items:
                # Check if node has our CSI driver
                has_driver = any(
                    driver.name == self.config.csi_driver 
                    for driver in node.spec.drivers
                )
                
                if has_driver:
                    node_info = {
                        "name": node.metadata.name,
                        "drivers": [
                            {
                                "name": driver.name,
                                "node_id": driver.node_id,
                                "allocatable": driver.allocatable.as_dict() if driver.allocatable else None,
                            }
                            for driver in node.spec.drivers
                        ],
                    }
                    result.append(node_info)
            
            logger.info(f"Found {len(result)} CSINodes with driver {self.config.csi_driver}")
            return result
            
        except ApiException as e:
            logger.error(f"Failed to list CSINodes: {e}")
            raise

    def get_csi_driver_pods(self) -> List[Dict[str, Any]]:
        """Get all pods related to the CSI driver.
        
        Returns:
            List of pod information
        """
        try:
            # Common label selectors for CSI drivers
            label_selectors = [
                f"app={self.config.csi_driver}",
                f"app.kubernetes.io/name={self.config.csi_driver}",
                "app=democratic-csi",
                "app.kubernetes.io/name=democratic-csi",
            ]
            
            all_pods = []
            seen = set()
            
            for selector in label_selectors:
                try:
                    pods = self.core_v1.list_pod_for_all_namespaces(label_selector=selector)
                    for pod in pods.items:
                        key = f"{pod.metadata.namespace}/{pod.metadata.name}"
                        if key not in seen:
                            seen.add(key)
                            all_pods.append(pod)
                except ApiException:
                    continue
            
            result = []
            for pod in all_pods:
                pod_info = {
                    "name": pod.metadata.name,
                    "namespace": pod.metadata.namespace,
                    "status": pod.status.phase,
                    "ready": all(
                        container.ready 
                        for container in pod.status.container_statuses or []
                    ),
                    "containers": [
                        {
                            "name": container.name,
                            "ready": container.ready,
                            "restart_count": container.restart_count,
                        }
                        for container in pod.status.container_statuses or []
                    ],
                }
                result.append(pod_info)
            
            logger.info(f"Found {len(result)} CSI driver pods")
            return result
            
        except ApiException as e:
            logger.error(f"Failed to list CSI driver pods: {e}")
            raise

    def check_csi_driver_health(self) -> Dict[str, Any]:
        """Check the health of the CSI driver.
        
        Returns:
            Health status information
        """
        pods = self.get_csi_driver_pods()
        
        if not pods:
            return {
                "healthy": False,
                "reason": "No CSI driver pods found",
                "total_pods": 0,
                "running_pods": 0,
                "ready_pods": 0,
            }
        
        running_pods = sum(1 for pod in pods if pod["status"] == "Running")
        ready_pods = sum(1 for pod in pods if pod["ready"])
        
        healthy = running_pods > 0 and running_pods == ready_pods
        
        return {
            "healthy": healthy,
            "reason": "All pods running and ready" if healthy else "Some pods not ready",
            "total_pods": len(pods),
            "running_pods": running_pods,
            "ready_pods": ready_pods,
            "pods": pods,
        }

    def find_orphaned_pvs(self) -> List[OrphanedResource]:
        """Find PersistentVolumes that are not bound to any PVC.
        
        Returns:
            List of orphaned PV resources
        """
        orphans = []
        pvs = self.get_persistent_volumes()
        
        for pv in pvs:
            if pv.phase == "Available" or (pv.phase == "Released" and not pv.claim_ref):
                orphan = OrphanedResource(
                    resource_type=ResourceType.PERSISTENT_VOLUME,
                    name=pv.name,
                    namespace=None,
                    volume_handle=pv.volume_handle,
                    creation_time=pv.creation_time or datetime.now(),
                    size=pv.capacity,
                    location="Kubernetes",
                    reason="No PVC bound" if pv.phase == "Available" else "PVC deleted",
                    details={
                        "phase": pv.phase,
                        "driver": pv.driver,
                        "access_modes": pv.access_modes,
                    },
                )
                orphans.append(orphan)
        
        logger.info(f"Found {len(orphans)} orphaned PersistentVolumes")
        return orphans

    def find_orphaned_pvcs(self, pending_threshold_minutes: int = 60) -> List[OrphanedResource]:
        """Find PersistentVolumeClaims that are stuck in pending state.
        
        Args:
            pending_threshold_minutes: Minutes before considering a pending PVC as orphaned
            
        Returns:
            List of orphaned PVC resources
        """
        orphans = []
        pvcs = self.get_persistent_volume_claims()
        threshold = datetime.now(tz=pvcs[0].creation_time.tzinfo if pvcs and pvcs[0].creation_time else None) - timedelta(minutes=pending_threshold_minutes)
        
        for pvc in pvcs:
            if pvc.phase == "Pending" and pvc.creation_time and pvc.creation_time < threshold:
                orphan = OrphanedResource(
                    resource_type=ResourceType.PERSISTENT_VOLUME_CLAIM,
                    name=pvc.name,
                    namespace=pvc.namespace,
                    volume_handle=None,
                    creation_time=pvc.creation_time,
                    size=pvc.capacity,
                    location="Kubernetes",
                    reason=f"Pending for over {pending_threshold_minutes} minutes",
                    details={
                        "phase": pvc.phase,
                        "storage_class": pvc.storage_class,
                    },
                )
                orphans.append(orphan)
        
        logger.info(f"Found {len(orphans)} orphaned PersistentVolumeClaims")
        return orphans

    def watch_persistent_volumes(self, timeout_seconds: Optional[int] = None) -> Generator[Dict[str, Any], None, None]:
        """Watch for PersistentVolume events.
        
        Args:
            timeout_seconds: Timeout for the watch operation
            
        Yields:
            Dictionary containing event information
        """
        w = watch.Watch()
        
        try:
            for event in w.stream(
                self.core_v1.list_persistent_volume,
                timeout_seconds=timeout_seconds,
            ):
                pv = event["object"]
                
                # Filter by CSI driver
                if pv.spec.csi and pv.spec.csi.driver == self.config.csi_driver:
                    yield {
                        "type": event["type"],
                        "name": pv.metadata.name,
                        "volume_handle": pv.spec.csi.volume_handle,
                        "phase": pv.status.phase,
                        "timestamp": datetime.now(),
                    }
        finally:
            w.stop()

    def watch_persistent_volume_claims(
        self, 
        namespace: Optional[str] = None,
        timeout_seconds: Optional[int] = None
    ) -> Generator[Dict[str, Any], None, None]:
        """Watch for PersistentVolumeClaim events.
        
        Args:
            namespace: Namespace to watch (None for all)
            timeout_seconds: Timeout for the watch operation
            
        Yields:
            Dictionary containing event information
        """
        w = watch.Watch()
        namespace = namespace or self.config.namespace
        
        try:
            if namespace:
                stream = w.stream(
                    self.core_v1.list_namespaced_persistent_volume_claim,
                    namespace=namespace,
                    timeout_seconds=timeout_seconds,
                )
            else:
                stream = w.stream(
                    self.core_v1.list_persistent_volume_claim_for_all_namespaces,
                    timeout_seconds=timeout_seconds,
                )
            
            for event in stream:
                pvc = event["object"]
                
                # Filter by storage class
                if self.config.storage_class and pvc.spec.storage_class_name != self.config.storage_class:
                    continue
                
                yield {
                    "type": event["type"],
                    "name": pvc.metadata.name,
                    "namespace": pvc.metadata.namespace,
                    "phase": pvc.status.phase,
                    "volume_name": pvc.spec.volume_name,
                    "timestamp": datetime.now(),
                }
        finally:
            w.stop()

    def find_orphaned_snapshots(self, truenas_snapshots: List = None, age_threshold_minutes: int = 1440) -> List[OrphanedResource]:
        """Find snapshots that are orphaned.
        
        Args:
            truenas_snapshots: List of TrueNAS snapshots to compare against
            age_threshold_minutes: Age threshold in minutes for considering snapshots orphaned
            
        Returns:
            List of OrphanedResource objects for orphaned snapshots
        """
        orphaned = []
        age_threshold = datetime.now() - timedelta(minutes=age_threshold_minutes)
        
        try:
            # Get all VolumeSnapshots from Kubernetes
            k8s_snapshots = self.get_volume_snapshots()
            
            # Find VolumeSnapshots without corresponding TrueNAS snapshots
            if truenas_snapshots is not None:
                truenas_snapshot_names = {snap.full_name for snap in truenas_snapshots}
                
                for k8s_snap in k8s_snapshots:
                    # Try to match K8s snapshot to TrueNAS snapshot
                    # This may need adjustment based on naming conventions
                    potential_names = [
                        f"{k8s_snap.source_pvc}@{k8s_snap.name}",
                        f"tank/k8s/volumes/{k8s_snap.source_pvc}@{k8s_snap.name}",
                        f"pool0/k8s/volumes/{k8s_snap.source_pvc}@{k8s_snap.name}",
                    ]
                    
                    if not any(name in truenas_snapshot_names for name in potential_names):
                        orphaned.append(OrphanedResource(
                            resource_type=ResourceType.VOLUME_SNAPSHOT,
                            name=k8s_snap.name,
                            namespace=k8s_snap.namespace,
                            volume_handle=None,
                            creation_time=k8s_snap.creation_time or datetime.now(),
                            size=None,
                            location="Kubernetes",
                            reason="No corresponding TrueNAS snapshot found",
                            details={
                                "source_pvc": k8s_snap.source_pvc,
                                "snapshot_class": k8s_snap.snapshot_class,
                                "ready": k8s_snap.ready_to_use,
                                "potential_truenas_names": potential_names
                            }
                        ))
            
            # Find old VolumeSnapshots that might be stuck
            for k8s_snap in k8s_snapshots:
                if k8s_snap.creation_time and k8s_snap.creation_time < age_threshold:
                    if not k8s_snap.ready_to_use:
                        orphaned.append(OrphanedResource(
                            resource_type=ResourceType.VOLUME_SNAPSHOT,
                            name=k8s_snap.name,
                            namespace=k8s_snap.namespace,
                            volume_handle=None,
                            creation_time=k8s_snap.creation_time or datetime.now(),
                            size=None,
                            location="Kubernetes",
                            reason=f"VolumeSnapshot not ready after {age_threshold_minutes} minutes",
                            details={
                                "source_pvc": k8s_snap.source_pvc,
                                "snapshot_class": k8s_snap.snapshot_class,
                                "ready": k8s_snap.ready_to_use,
                                "age_hours": (datetime.now() - k8s_snap.creation_time).total_seconds() / 3600 if k8s_snap.creation_time else 0
                            }
                        ))
            
            logger.info(f"Found {len(orphaned)} orphaned snapshots")
            return orphaned
            
        except ApiException as e:
            logger.error(f"Failed to find orphaned snapshots: {e}")
            return []
    def get_volume_names(self) -> List[str]:
        """Get names of all volumes from PVCs.
        
        Returns:
            List of volume names
        """
        try:
            pvcs = self.get_persistent_volume_claims()
            volume_names = []
            
            for pvc in pvcs:
                if pvc.volume_name:
                    volume_names.append(pvc.volume_name)
                else:
                    # Use PVC name if no volume is bound yet
                    volume_names.append(pvc.name)
            
            logger.info(f"Found {len(volume_names)} volume names")
            return volume_names
            
        except ApiException as e:
            logger.error(f"Failed to get volume names: {e}")
            return []

    def find_orphaned_k8s_snapshots(self, active_pvcs: List, age_days: int = 30) -> List[OrphanedResource]:
        """Find Kubernetes snapshots that are orphaned.
        
        Args:
            active_pvcs: List of active PVC objects
            age_days: Age threshold in days
            
        Returns:
            List of orphaned snapshot resources
        """
        return self.find_orphaned_snapshots(age_threshold_minutes=age_days * 24 * 60)

    @property
    def custom_objects_api(self):
        """Access to custom objects API for testing."""
        return self.custom_objects

    def find_stale_snapshots(self, age_threshold_days: int = 30) -> List[VolumeSnapshotInfo]:
        """Find VolumeSnapshots older than the specified threshold.
        
        Args:
            age_threshold_days: Age threshold in days
            
        Returns:
            List of stale VolumeSnapshotInfo objects
        """
        stale_snapshots = []
        age_threshold = datetime.now() - timedelta(days=age_threshold_days)
        
        try:
            snapshots = self.get_volume_snapshots()
            
            for snapshot in snapshots:
                if snapshot.creation_time < age_threshold:
                    stale_snapshots.append(snapshot)
            
            logger.info(f"Found {len(stale_snapshots)} stale snapshots older than {age_threshold_days} days")
            return stale_snapshots
            
        except ApiException as e:
            logger.error(f"Failed to find stale snapshots: {e}")
            return []