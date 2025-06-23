"""Unit tests for Kubernetes client wrapper."""

import pytest
from unittest.mock import Mock, patch, MagicMock
from datetime import datetime
from kubernetes import client as k8s_client
from kubernetes.client.rest import ApiException

from truenas_storage_monitor.k8s_client import (
    K8sClient,
    K8sConfig,
    PersistentVolumeInfo,
    PersistentVolumeClaimInfo,
    VolumeSnapshotInfo,
    OrphanedResource,
    ResourceType,
)


class TestK8sConfig:
    """Test K8sConfig validation."""

    def test_valid_config(self):
        """Test creating valid configuration."""
        config = K8sConfig(
            kubeconfig="/path/to/kubeconfig",
            namespace="default",
            csi_driver="org.democratic-csi.nfs",
            storage_class="democratic-csi-nfs",
        )
        assert config.kubeconfig == "/path/to/kubeconfig"
        assert config.namespace == "default"
        assert config.csi_driver == "org.democratic-csi.nfs"
        assert config.storage_class == "democratic-csi-nfs"

    def test_in_cluster_config(self):
        """Test in-cluster configuration."""
        config = K8sConfig(in_cluster=True)
        assert config.in_cluster is True
        assert config.kubeconfig is None


class TestK8sClient:
    """Test K8sClient functionality."""

    @pytest.fixture
    def mock_k8s_config(self):
        """Create mock Kubernetes configuration."""
        return K8sConfig(
            namespace="test-namespace",
            csi_driver="org.democratic-csi.nfs",
            storage_class="democratic-csi-nfs",
        )

    @pytest.fixture
    def mock_client(self, mock_k8s_config):
        """Create K8sClient with mocked Kubernetes API."""
        with patch('truenas_storage_monitor.k8s_client.config') as mock_config:
            with patch('truenas_storage_monitor.k8s_client.k8s_client') as mock_api:
                client = K8sClient(mock_k8s_config)
                client.core_v1 = Mock()
                client.storage_v1 = Mock()
                client.custom_objects = Mock()
                return client

    def test_client_initialization(self, mock_k8s_config):
        """Test client initialization."""
        with patch('truenas_storage_monitor.k8s_client.config') as mock_config:
            with patch('truenas_storage_monitor.k8s_client.k8s_client'):
                client = K8sClient(mock_k8s_config)
                assert client.config == mock_k8s_config
                mock_config.load_kube_config.assert_called_once()

    def test_client_initialization_in_cluster(self):
        """Test in-cluster client initialization."""
        config = K8sConfig(in_cluster=True)
        with patch('truenas_storage_monitor.k8s_client.config') as mock_config:
            with patch('truenas_storage_monitor.k8s_client.k8s_client'):
                client = K8sClient(config)
                mock_config.load_incluster_config.assert_called_once()

    def test_get_persistent_volumes(self, mock_client):
        """Test getting persistent volumes."""
        # Mock PV objects
        pv1 = Mock()
        pv1.metadata.name = "pv-test-1"
        pv1.metadata.creation_timestamp = datetime.now()
        pv1.spec.csi = Mock(driver="org.democratic-csi.nfs", volume_handle="vol-1")
        pv1.spec.capacity = {"storage": "10Gi"}
        pv1.spec.access_modes = ["ReadWriteOnce"]
        pv1.spec.claim_ref = Mock(namespace="default", name="pvc-1")
        pv1.status.phase = "Bound"
        
        pv2 = Mock()
        pv2.metadata.name = "pv-test-2"
        pv2.spec.csi = None  # Non-CSI PV
        
        mock_client.core_v1.list_persistent_volume.return_value = Mock(items=[pv1, pv2])
        
        pvs = mock_client.get_persistent_volumes()
        
        assert len(pvs) == 1
        assert pvs[0].name == "pv-test-1"
        assert pvs[0].volume_handle == "vol-1"
        assert pvs[0].driver == "org.democratic-csi.nfs"

    def test_get_persistent_volume_claims(self, mock_client):
        """Test getting persistent volume claims."""
        # Mock PVC objects
        pvc1 = Mock()
        pvc1.metadata.name = "pvc-test-1"
        pvc1.metadata.namespace = "test-namespace"
        pvc1.metadata.creation_timestamp = datetime.now()
        pvc1.spec.storage_class_name = "democratic-csi-nfs"
        pvc1.spec.volume_name = "pv-test-1"
        pvc1.spec.resources.requests = {"storage": "10Gi"}
        pvc1.status.phase = "Bound"
        
        pvc2 = Mock()
        pvc2.metadata.name = "pvc-test-2"
        pvc2.spec.storage_class_name = "other-storage"
        
        mock_client.core_v1.list_namespaced_persistent_volume_claim.return_value = Mock(items=[pvc1, pvc2])
        
        pvcs = mock_client.get_persistent_volume_claims()
        
        assert len(pvcs) == 1
        assert pvcs[0].name == "pvc-test-1"
        assert pvcs[0].storage_class == "democratic-csi-nfs"

    def test_get_volume_snapshots(self, mock_client):
        """Test getting volume snapshots."""
        # Mock snapshot response
        snapshot1 = {
            "metadata": {
                "name": "snapshot-1",
                "namespace": "test-namespace",
                "creationTimestamp": "2024-01-01T00:00:00Z",
            },
            "spec": {
                "source": {"persistentVolumeClaimName": "pvc-1"},
                "volumeSnapshotClassName": "democratic-csi-snapshot",
            },
            "status": {
                "readyToUse": True,
                "creationTime": "2024-01-01T00:00:00Z",
            },
        }
        
        mock_client.custom_objects.list_namespaced_custom_object.return_value = {
            "items": [snapshot1]
        }
        
        snapshots = mock_client.get_volume_snapshots()
        
        assert len(snapshots) == 1
        assert snapshots[0].name == "snapshot-1"
        assert snapshots[0].source_pvc == "pvc-1"
        assert snapshots[0].ready_to_use is True

    def test_get_storage_classes(self, mock_client):
        """Test getting storage classes."""
        # Mock StorageClass objects
        sc1 = Mock()
        sc1.metadata.name = "democratic-csi-nfs"
        sc1.provisioner = "org.democratic-csi.nfs"
        sc1.parameters = {"fsType": "nfs"}
        sc1.reclaim_policy = "Delete"
        sc1.volume_binding_mode = "Immediate"
        sc1.allow_volume_expansion = True
        
        sc2 = Mock()
        sc2.metadata.name = "other-storage"
        sc2.provisioner = "other.csi.driver"
        
        mock_client.storage_v1.list_storage_class.return_value = Mock(items=[sc1, sc2])
        
        scs = mock_client.get_storage_classes()
        
        assert len(scs) == 1
        assert scs[0].name == "democratic-csi-nfs"
        assert scs[0].provisioner == "org.democratic-csi.nfs"

    def test_get_csi_nodes(self, mock_client):
        """Test getting CSI nodes."""
        # Mock CSINode objects
        node1 = Mock()
        node1.metadata.name = "node-1"
        driver1 = Mock()
        driver1.name = "org.democratic-csi.nfs"
        driver1.node_id = "node-1"
        node1.spec.drivers = [driver1]
        
        mock_client.storage_v1.list_csi_node.return_value = Mock(items=[node1])
        
        nodes = mock_client.get_csi_nodes()
        
        assert len(nodes) == 1
        assert nodes[0].name == "node-1"
        assert any(d.name == "org.democratic-csi.nfs" for d in nodes[0].drivers)

    def test_get_csi_driver_pods(self, mock_client):
        """Test getting CSI driver pods."""
        # Mock Pod objects
        pod1 = Mock()
        pod1.metadata.name = "democratic-csi-controller-0"
        pod1.metadata.namespace = "democratic-csi"
        pod1.metadata.labels = {"app": "democratic-csi"}
        pod1.status.phase = "Running"
        
        container1 = Mock()
        container1.name = "csi-driver"
        container1.ready = True
        pod1.status.container_statuses = [container1]
        
        mock_client.core_v1.list_pod_for_all_namespaces.return_value = Mock(items=[pod1])
        
        pods = mock_client.get_csi_driver_pods()
        
        assert len(pods) == 1
        assert pods[0].name == "democratic-csi-controller-0"
        assert pods[0].status == "Running"

    def test_check_csi_driver_health(self, mock_client):
        """Test checking CSI driver health."""
        # Mock healthy pods
        pod1 = Mock()
        pod1.metadata.name = "democratic-csi-controller-0"
        pod1.status.phase = "Running"
        container1 = Mock()
        container1.ready = True
        pod1.status.container_statuses = [container1]
        
        mock_client.core_v1.list_pod_for_all_namespaces.return_value = Mock(items=[pod1])
        
        health = mock_client.check_csi_driver_health()
        
        assert health["healthy"] is True
        assert health["total_pods"] == 1
        assert health["running_pods"] == 1

    def test_check_csi_driver_health_unhealthy(self, mock_client):
        """Test checking CSI driver health with unhealthy pods."""
        # Mock unhealthy pod
        pod1 = Mock()
        pod1.metadata.name = "democratic-csi-controller-0"
        pod1.status.phase = "CrashLoopBackOff"
        
        mock_client.core_v1.list_pod_for_all_namespaces.return_value = Mock(items=[pod1])
        
        health = mock_client.check_csi_driver_health()
        
        assert health["healthy"] is False
        assert health["total_pods"] == 1
        assert health["running_pods"] == 0

    def test_find_orphaned_pvs(self, mock_client):
        """Test finding orphaned PVs."""
        # Mock PV without PVC
        pv1 = Mock()
        pv1.metadata.name = "pv-orphaned"
        pv1.metadata.creation_timestamp = datetime.now()
        pv1.spec.csi = Mock(driver="org.democratic-csi.nfs", volume_handle="vol-orphaned")
        pv1.spec.claim_ref = None
        pv1.status.phase = "Available"
        
        mock_client.core_v1.list_persistent_volume.return_value = Mock(items=[pv1])
        
        orphans = mock_client.find_orphaned_pvs()
        
        assert len(orphans) == 1
        assert orphans[0].resource_type == ResourceType.PERSISTENT_VOLUME
        assert orphans[0].name == "pv-orphaned"
        assert orphans[0].reason == "No PVC bound"

    def test_find_orphaned_pvcs(self, mock_client):
        """Test finding orphaned PVCs."""
        # Mock PVC in pending state
        pvc1 = Mock()
        pvc1.metadata.name = "pvc-orphaned"
        pvc1.metadata.namespace = "test-namespace"
        pvc1.metadata.creation_timestamp = datetime(2024, 1, 1)
        pvc1.spec.storage_class_name = "democratic-csi-nfs"
        pvc1.status.phase = "Pending"
        
        mock_client.core_v1.list_namespaced_persistent_volume_claim.return_value = Mock(items=[pvc1])
        
        orphans = mock_client.find_orphaned_pvcs(pending_threshold_minutes=60)
        
        assert len(orphans) == 1
        assert orphans[0].resource_type == ResourceType.PERSISTENT_VOLUME_CLAIM
        assert orphans[0].name == "pvc-orphaned"

    def test_error_handling_api_exception(self, mock_client):
        """Test error handling for Kubernetes API exceptions."""
        mock_client.core_v1.list_persistent_volume.side_effect = ApiException(
            status=403, reason="Forbidden"
        )
        
        with pytest.raises(ApiException):
            mock_client.get_persistent_volumes()

    def test_watch_persistent_volumes(self, mock_client):
        """Test watching PV events."""
        # Mock watch stream
        mock_event = Mock()
        mock_event.type = "ADDED"
        mock_event.object = Mock()
        mock_event.object.metadata.name = "pv-new"
        mock_event.object.spec.csi = Mock(driver="org.democratic-csi.nfs")
        
        with patch('truenas_storage_monitor.k8s_client.watch.Watch') as mock_watch:
            mock_watch.return_value.stream.return_value = [mock_event]
            
            events = []
            for event in mock_client.watch_persistent_volumes(timeout_seconds=1):
                events.append(event)
                break  # Process one event
            
            assert len(events) == 1
            assert events[0]["type"] == "ADDED"
            assert events[0]["name"] == "pv-new"