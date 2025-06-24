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
    
    mock_k8s_client.storage_v1.list_storage_class.return_value.items = [mock_sc]
    
    scs = mock_k8s_client.get_storage_classes()
    
    assert len(scs) == 1
    assert scs[0]["name"] == "democratic-csi-nfs"
    assert scs[0]["provisioner"] == "org.democratic-csi.nfs"


def test_get_persistent_volumes(mock_k8s_client):
    """Test getting persistent volumes."""
    # Mock PV object
    mock_pv = Mock()
    mock_pv.metadata.name = "pv-test-123"
    mock_pv.spec.csi.driver = "org.democratic-csi.nfs"
    mock_pv.spec.csi.volume_handle = "test-volume-handle"
    mock_pv.spec.capacity = {"storage": "10Gi"}
    mock_pv.spec.access_modes = ["ReadWriteOnce"]
    mock_pv.status.phase = "Bound"
    mock_pv.spec.claim_ref = Mock()
    mock_pv.spec.claim_ref.name = "test-pvc"
    mock_pv.spec.claim_ref.namespace = "test-ns"
    mock_pv.metadata.creation_timestamp = datetime.now()
    mock_pv.metadata.labels = {"app": "test"}
    mock_pv.metadata.annotations = {"volume.beta.kubernetes.io/storage-provisioner": "org.democratic-csi.nfs"}
    
    mock_k8s_client.core_v1.list_persistent_volume.return_value.items = [mock_pv]
    
    pvs = mock_k8s_client.get_persistent_volumes()
    
    assert len(pvs) == 1
    assert pvs[0].name == "pv-test-123"
    assert pvs[0].driver == "org.democratic-csi.nfs"


def test_get_persistent_volume_claims(mock_k8s_client):
    """Test getting persistent volume claims."""
    # Mock PVC object
    mock_pvc = Mock()
    mock_pvc.metadata.name = "test-pvc"
    mock_pvc.metadata.namespace = "test-namespace"
    mock_pvc.spec.storage_class_name = "democratic-csi-nfs"
    mock_pvc.spec.volume_name = "pv-test-123"
    mock_pvc.spec.resources.requests = {"storage": "10Gi"}
    mock_pvc.status.phase = "Bound"
    mock_pvc.metadata.creation_timestamp = datetime.now()
    mock_pvc.metadata.labels = {}
    mock_pvc.metadata.annotations = {}
    
    mock_k8s_client.core_v1.list_namespaced_persistent_volume_claim.return_value.items = [mock_pvc]
    
    pvcs = mock_k8s_client.get_persistent_volume_claims()
    
    assert len(pvcs) == 1
    assert pvcs[0].name == "test-pvc"
    assert pvcs[0].namespace == "test-namespace"


def test_get_volume_snapshots(mock_k8s_client):
    """Test getting volume snapshots."""
    # Mock VolumeSnapshot object
    mock_snapshot = {
        "metadata": {
            "name": "test-snapshot",
            "namespace": "test-namespace",
            "creationTimestamp": "2024-01-01T10:00:00Z",
            "labels": {},
            "annotations": {}
        },
        "spec": {
            "source": {"persistentVolumeClaimName": "test-pvc"},
            "volumeSnapshotClassName": "democratic-csi-nfs"
        },
        "status": {
            "readyToUse": True
        }
    }
    
    mock_k8s_client.custom_objects.list_namespaced_custom_object.return_value = {
        "items": [mock_snapshot]
    }
    
    snapshots = mock_k8s_client.get_volume_snapshots()
    
    assert len(snapshots) == 1
    assert snapshots[0].name == "test-snapshot"
    assert snapshots[0].source_pvc == "test-pvc"
    assert snapshots[0].ready_to_use is True


def test_find_orphaned_pvs(mock_k8s_client):
    """Test finding orphaned PVs."""
    # Mock orphaned PV
    mock_pv = PersistentVolumeInfo(
        name="orphaned-pv",
        volume_handle="orphaned-handle",
        driver="org.democratic-csi.nfs",
        capacity="10Gi",
        access_modes=["ReadWriteOnce"],
        phase="Available",  # Available = orphaned
        creation_time=datetime.now() - timedelta(hours=25)
    )
    
    mock_k8s_client.get_persistent_volumes = Mock(return_value=[mock_pv])
    
    orphans = mock_k8s_client.find_orphaned_pvs()
    
    assert len(orphans) == 1
    assert orphans[0].name == "orphaned-pv"
    assert orphans[0].resource_type == ResourceType.PERSISTENT_VOLUME


def test_find_orphaned_pvcs(mock_k8s_client):
    """Test finding orphaned PVCs."""
    # Mock orphaned PVC (stuck in Pending)
    mock_pvc = PersistentVolumeClaimInfo(
        name="orphaned-pvc",
        namespace="test",
        storage_class="democratic-csi-nfs",
        volume_name=None,
        capacity="10Gi",
        phase="Pending",  # Pending for too long = orphaned
        creation_time=datetime.now() - timedelta(hours=2)  # > 60 minutes
    )
    
    mock_k8s_client.get_persistent_volume_claims = Mock(return_value=[mock_pvc])
    
    orphans = mock_k8s_client.find_orphaned_pvcs()
    
    assert len(orphans) == 1
    assert orphans[0].name == "orphaned-pvc"
    assert orphans[0].resource_type == ResourceType.PERSISTENT_VOLUME_CLAIM


def test_check_csi_driver_health(mock_k8s_client):
    """Test CSI driver health check."""
    # Mock healthy pods
    mock_pod = {
        "name": "democratic-csi-controller",
        "namespace": "democratic-csi",
        "status": "Running",
        "ready": True,
        "containers": [
            {"name": "csi-controller", "ready": True, "restart_count": 0}
        ]
    }
    
    mock_k8s_client.get_csi_driver_pods = Mock(return_value=[mock_pod])
    
    health = mock_k8s_client.check_csi_driver_health()
    
    assert health["healthy"] is True
    assert health["total_pods"] == 1
    assert health["running_pods"] == 1
    assert health["ready_pods"] == 1


def test_find_orphaned_snapshots(mock_k8s_client):
    """Test finding orphaned snapshots."""
    # Mock K8s snapshot
    k8s_snapshot = VolumeSnapshotInfo(
        name="test-snapshot",
        namespace="test",
        source_pvc="test-pvc",
        snapshot_class="democratic-csi-nfs",
        ready_to_use=True,
        creation_time=datetime.now() - timedelta(days=2)
    )
    
    mock_k8s_client.get_volume_snapshots = Mock(return_value=[k8s_snapshot])
    
    # Mock TrueNAS snapshots (empty - so K8s snapshot is orphaned)
    truenas_snapshots = []
    
    orphans = mock_k8s_client.find_orphaned_snapshots(truenas_snapshots)
    
    assert len(orphans) == 1
    assert orphans[0].name == "test-snapshot"


def test_find_stale_snapshots(mock_k8s_client):
    """Test finding stale snapshots."""
    # Mock old snapshot
    old_snapshot = VolumeSnapshotInfo(
        name="old-snapshot",
        namespace="test",
        source_pvc="test-pvc",
        snapshot_class="democratic-csi-nfs",
        ready_to_use=True,
        creation_time=datetime.now() - timedelta(days=45)  # 45 days old
    )
    
    # Mock recent snapshot
    recent_snapshot = VolumeSnapshotInfo(
        name="recent-snapshot",
        namespace="test",
        source_pvc="test-pvc",
        snapshot_class="democratic-csi-nfs",
        ready_to_use=True,
        creation_time=datetime.now() - timedelta(days=5)  # 5 days old
    )
    
    mock_k8s_client.get_volume_snapshots = Mock(return_value=[old_snapshot, recent_snapshot])
    
    stale = mock_k8s_client.find_stale_snapshots(age_threshold_days=30)
    
    assert len(stale) == 1
    assert stale[0].name == "old-snapshot"