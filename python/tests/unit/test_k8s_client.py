"""Unit tests for Kubernetes client."""

import pytest
from unittest.mock import Mock, patch, MagicMock
from datetime import datetime, timedelta

from truenas_storage_monitor.k8s_client import (
    K8sClient, K8sConfig, PersistentVolumeInfo, 
    PersistentVolumeClaimInfo, VolumeSnapshotInfo, ResourceType
)


@pytest.fixture
def k8s_config():
    """Create a test K8s configuration."""
    return K8sConfig(
        namespace="test-namespace",
        csi_driver="org.democratic-csi.nfs",
        storage_class="democratic-csi-nfs"
    )


@pytest.fixture
def mock_k8s_client(k8s_config):
    """Create a mocked K8s client."""
    with patch('truenas_storage_monitor.k8s_client.config'), \
         patch('truenas_storage_monitor.k8s_client.k8s_client'):
        client = K8sClient(k8s_config)
        return client


def test_k8s_client_initialization(k8s_config):
    """Test K8s client initialization."""
    with patch('truenas_storage_monitor.k8s_client.config'), \
         patch('truenas_storage_monitor.k8s_client.k8s_client'):
        client = K8sClient(k8s_config)
        assert client.config == k8s_config


def test_get_storage_classes(mock_k8s_client):
    """Test getting storage classes."""
    # Mock StorageClass object
    mock_sc = Mock()
    mock_sc.metadata.name = "democratic-csi-nfs"
    mock_sc.provisioner = "org.democratic-csi.nfs"
    mock_sc.parameters = {"server": "truenas.example.com"}
    mock_sc.reclaim_policy = "Delete"
    mock_sc.volume_binding_mode = "Immediate"
    mock_sc.allow_volume_expansion = True
    
    with patch.object(mock_k8s_client, 'storage_v1') as mock_storage:
        mock_storage.list_storage_class.return_value.items = [mock_sc]
        
        storage_classes = mock_k8s_client.get_storage_classes()
        
        assert len(storage_classes) == 1
        assert storage_classes[0]["name"] == "democratic-csi-nfs"
        assert storage_classes[0]["provisioner"] == "org.democratic-csi.nfs"


def test_get_persistent_volumes(mock_k8s_client):
    """Test getting persistent volumes."""
    # Mock PV object
    mock_pv = Mock()
    mock_pv.metadata.name = "test-pv"
    mock_pv.metadata.creation_timestamp = datetime.now()
    mock_pv.spec.capacity = {"storage": "10Gi"}
    mock_pv.status.phase = "Bound"
    mock_pv.spec.claim_ref.name = "test-pvc"
    mock_pv.spec.claim_ref.namespace = "default"
    mock_pv.spec.storage_class_name = "democratic-csi-nfs"
    mock_pv.spec.csi.volume_handle = "tank/k8s/volumes/test-pv"
    
    with patch.object(mock_k8s_client, 'core_v1') as mock_core:
        mock_core.list_persistent_volume.return_value.items = [mock_pv]
        
        pvs = mock_k8s_client.get_persistent_volumes()
        
        assert len(pvs) == 1
        assert pvs[0].name == "test-pv"
        assert pvs[0].capacity == "10Gi"
        assert pvs[0].status == "Bound"


def test_get_persistent_volume_claims(mock_k8s_client):
    """Test getting persistent volume claims."""
    # Mock PVC object
    mock_pvc = Mock()
    mock_pvc.metadata.name = "test-pvc"
    mock_pvc.metadata.namespace = "default"
    mock_pvc.metadata.creation_timestamp = datetime.now()
    mock_pvc.spec.access_modes = ["ReadWriteOnce"]
    mock_pvc.spec.resources.requests = {"storage": "10Gi"}
    mock_pvc.status.phase = "Bound"
    mock_pvc.spec.volume_name = "test-pv"
    mock_pvc.spec.storage_class_name = "democratic-csi-nfs"
    
    with patch.object(mock_k8s_client, 'core_v1') as mock_core:
        mock_core.list_namespaced_persistent_volume_claim.return_value.items = [mock_pvc]
        
        pvcs = mock_k8s_client.get_persistent_volume_claims()
        
        assert len(pvcs) == 1
        assert pvcs[0].name == "test-pvc"
        assert pvcs[0].namespace == "default"
        assert pvcs[0].status == "Bound"


def test_get_volume_snapshots(mock_k8s_client):
    """Test getting volume snapshots."""
    # Mock VolumeSnapshot object
    mock_snapshot = Mock()
    mock_snapshot.metadata.name = "test-snapshot"
    mock_snapshot.metadata.namespace = "default"
    mock_snapshot.metadata.creation_timestamp = datetime.now()
    mock_snapshot.spec.source.persistent_volume_claim_name = "test-pvc"
    mock_snapshot.status.ready_to_use = True
    mock_snapshot.status.restore_size = "1Gi"
    
    with patch.object(mock_k8s_client, 'custom_objects_api') as mock_custom:
        mock_custom.list_namespaced_custom_object.return_value = {
            "items": [mock_snapshot]
        }
        
        snapshots = mock_k8s_client.get_volume_snapshots()
        
        assert len(snapshots) == 1
        assert snapshots[0].name == "test-snapshot"
        assert snapshots[0].namespace == "default"
        assert snapshots[0].ready_to_use is True


def test_test_connection_success(mock_k8s_client):
    """Test successful connection test."""
    with patch.object(mock_k8s_client, 'core_v1') as mock_core:
        mock_core.get_api_resources.return_value = Mock()
        
        result = mock_k8s_client.test_connection()
        
        assert result is True


def test_test_connection_failure(mock_k8s_client):
    """Test connection test failure."""
    from kubernetes.client.rest import ApiException
    
    with patch.object(mock_k8s_client, 'core_v1') as mock_core:
        mock_core.get_api_resources.side_effect = ApiException("Connection failed")
        
        result = mock_k8s_client.test_connection()
        
        assert result is False


def test_get_volume_names(mock_k8s_client):
    """Test getting volume names."""
    mock_pvs = [
        Mock(name="pv-1", volume_handle="tank/k8s/vol1"),
        Mock(name="pv-2", volume_handle="tank/k8s/vol2")
    ]
    
    with patch.object(mock_k8s_client, 'get_persistent_volumes', return_value=mock_pvs):
        names = mock_k8s_client.get_volume_names()
        
        assert len(names) == 2
        assert "vol1" in names
        assert "vol2" in names


def test_find_orphaned_k8s_snapshots(mock_k8s_client):
    """Test finding orphaned K8s snapshots."""
    # Mock snapshots
    mock_snapshots = [
        Mock(
            name="orphaned-snapshot",
            source_pvc="missing-pvc",
            ready_to_use=True,
            creation_time=datetime.now() - timedelta(days=40)
        ),
        Mock(
            name="valid-snapshot", 
            source_pvc="active-pvc",
            ready_to_use=True,
            creation_time=datetime.now() - timedelta(days=5)
        )
    ]
    
    # Active PVC names
    active_pvcs = ["active-pvc", "another-pvc"]
    
    with patch.object(mock_k8s_client, 'get_volume_snapshots', return_value=mock_snapshots):
        orphans = mock_k8s_client.find_orphaned_k8s_snapshots(active_pvcs, age_days=30)
        
        assert len(orphans) == 1
        assert orphans[0].name == "orphaned-snapshot"