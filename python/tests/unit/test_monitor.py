"""Unit tests for Monitor class."""

import pytest
from unittest.mock import Mock, patch, MagicMock
from datetime import datetime, timedelta

from truenas_storage_monitor.monitor import Monitor
from truenas_storage_monitor.k8s_client import (
    K8sClient, PersistentVolumeInfo, VolumeSnapshotInfo
)
from truenas_storage_monitor.truenas_client import (
    TrueNASClient, PoolInfo, DatasetInfo, VolumeInfo, SnapshotInfo
)
@pytest.fixture
def mock_k8s_client():
    """Create a mocked K8s client."""
    client = Mock(spec=K8sClient)
    return client


@pytest.fixture
def mock_truenas_client():
    """Create a mocked TrueNAS client."""
    client = Mock(spec=TrueNASClient)
    return client


@pytest.fixture
def monitoring_config():
    """Create a test monitoring configuration."""
    return {
        "interval": 60,
        "thresholds": {
            "orphaned_pv_age_hours": 24,
            "pending_pvc_minutes": 60,
            "snapshot_age_days": 30,
            "pool_usage_percent": 80,
            "snapshot_size_gb": 100
        }
    }


@pytest.fixture
def monitor(monitoring_config):
    """Create a Monitor instance with mocked clients."""
    full_config = {
        "openshift": {
            "namespace": "democratic-csi",
            "storage_class": "democratic-csi-nfs",
            "csi_driver": "org.democratic-csi.nfs"
        },
        "truenas": {
            "url": "https://truenas.example.com",
            "username": "admin",
            "password": "test"
        },
        **monitoring_config
    }
    
    with patch('truenas_storage_monitor.monitor.K8sClient') as mock_k8s_cls, \
         patch('truenas_storage_monitor.monitor.TrueNASClient') as mock_truenas_cls:
        
        monitor_instance = Monitor(full_config)
        # Replace the clients with mocks for testing
        monitor_instance.k8s_client = Mock(spec=K8sClient)
        monitor_instance.truenas_client = Mock(spec=TrueNASClient)
        return monitor_instance


def test_monitor_initialization(monitor, mock_k8s_client, mock_truenas_client, monitoring_config):
    """Test Monitor initialization."""
    assert monitor.k8s_client == mock_k8s_client
    assert monitor.truenas_client == mock_truenas_client
    assert monitor.config == monitoring_config


def test_check_orphaned_pvs(monitor, mock_k8s_client, mock_truenas_client):
    """Test orphaned PV detection."""
    # Mock PVs from K8s
    mock_k8s_pvs = [
        PersistentVolumeInfo(
            name="pv-orphaned",
            capacity="10Gi",
            status="Available",
            claim_ref=None,
            storage_class="democratic-csi-nfs",
            creation_time=datetime.now() - timedelta(hours=48),
            volume_handle="tank/k8s/volumes/pv-orphaned"
        ),
        PersistentVolumeInfo(
            name="pv-bound",
            capacity="20Gi", 
            status="Bound",
            claim_ref="default/test-pvc",
            storage_class="democratic-csi-nfs",
            creation_time=datetime.now() - timedelta(hours=12),
            volume_handle="tank/k8s/volumes/pv-bound"
        )
    ]
    
    # Mock TrueNAS volumes  
    mock_truenas_volumes = [
        VolumeInfo(
            name="pv-orphaned",
            path="/mnt/tank/k8s/volumes/pv-orphaned",
            size=10737418240,
            type="FILE"
        ),
        VolumeInfo(
            name="pv-bound", 
            path="/mnt/tank/k8s/volumes/pv-bound",
            size=21474836480,
            type="FILE"
        )
    ]
    
    mock_k8s_client.get_persistent_volumes.return_value = mock_k8s_pvs
    mock_truenas_client.get_volumes.return_value = mock_truenas_volumes
    
    result = monitor.check_orphaned_pvs()
    
    assert len(result["orphaned_pvs"]) == 1
    assert result["orphaned_pvs"][0].name == "pv-orphaned"
    assert result["total_orphaned"] == 1
    assert len(result["recommendations"]) > 0


def test_check_orphaned_volumes(monitor, mock_k8s_client, mock_truenas_client):
    """Test orphaned volume detection."""
    # Mock active K8s volume names
    mock_k8s_client.get_volume_names.return_value = ["active-volume-1", "active-volume-2"]
    
    # Mock orphaned volumes from TrueNAS
    mock_orphaned_volumes = [
        VolumeInfo(
            name="orphaned-volume",
            path="/mnt/tank/k8s/volumes/orphaned-volume", 
            size=10737418240,
            type="FILE"
        )
    ]
    
    mock_truenas_client.find_orphaned_volumes.return_value = mock_orphaned_volumes
    
    result = monitor.check_orphaned_volumes()
    
    assert len(result["orphaned_volumes"]) == 1
    assert result["orphaned_volumes"][0].name == "orphaned-volume"
    assert result["total_orphaned"] == 1
    assert len(result["recommendations"]) > 0


def test_check_snapshot_health(monitor, mock_k8s_client, mock_truenas_client):
    """Test snapshot health checking."""
    # Mock K8s snapshots
    mock_k8s_snapshots = [
        VolumeSnapshotInfo(
            name="k8s-snapshot-1",
            namespace="default",
            source_pvc="test-pvc",
            ready_to_use=True,
            creation_time=datetime.now() - timedelta(days=5),
            size="1Gi"
        ),
        VolumeSnapshotInfo(
            name="k8s-snapshot-orphaned",
            namespace="default", 
            source_pvc="missing-pvc",
            ready_to_use=True,
            creation_time=datetime.now() - timedelta(days=40),
            size="2Gi"
        )
    ]
    
    # Mock TrueNAS snapshots
    mock_truenas_snapshots = [
        SnapshotInfo(
            name="k8s-snapshot-1",
            dataset="tank/k8s/volumes/test-pvc",
            creation_time=datetime.now() - timedelta(days=5),
            used_size=1073741824,
            referenced_size=2147483648,
            full_name="tank/k8s/volumes/test-pvc@k8s-snapshot-1"
        ),
        SnapshotInfo(
            name="truenas-orphaned",
            dataset="tank/k8s/volumes/old-volume",
            creation_time=datetime.now() - timedelta(days=60),
            used_size=5368709120,
            referenced_size=10737418240,
            full_name="tank/k8s/volumes/old-volume@truenas-orphaned"
        )
    ]
    
    mock_k8s_client.get_volume_snapshots.return_value = mock_k8s_snapshots
    mock_truenas_client.get_snapshots.return_value = mock_truenas_snapshots
    mock_truenas_client.find_orphaned_truenas_snapshots.return_value = [mock_truenas_snapshots[1]]
    
    result = monitor.check_snapshot_health()
    
    assert result["k8s_snapshots"]["total"] == 2
    assert result["truenas_snapshots"]["total"] == 2
    assert len(result["orphaned_resources"]["k8s_orphaned"]) == 1
    assert len(result["orphaned_resources"]["truenas_orphaned"]) == 1
    assert len(result["recommendations"]) > 0


def test_analyze_storage_efficiency(monitor, mock_k8s_client, mock_truenas_client):
    """Test storage efficiency analysis."""
    # Mock storage pools
    mock_pools = [
        PoolInfo(
            name="tank",
            status="ONLINE",
            size=1099511627776,  # 1TB
            allocated=549755813888,  # 512GB
            free=549755813888,  # 512GB
            fragmentation="15%",
            healthy=True
        )
    ]
    
    # Mock datasets
    mock_datasets = [
        DatasetInfo(
            name="tank/k8s",
            type="FILESYSTEM",
            used=107374182400,  # 100GB
            available=992137445376,  # 924GB
            referenced=53687091200,  # 50GB
            quota=None,
            compressratio="1.5x"
        )
    ]
    
    # Mock snapshots for analysis
    mock_snapshots = [
        SnapshotInfo(
            name="snap-1",
            dataset="tank/k8s/vol1",
            creation_time=datetime.now() - timedelta(days=10),
            used_size=10737418240,  # 10GB
            referenced_size=21474836480,  # 20GB
            full_name="tank/k8s/vol1@snap-1"
        )
    ]
    
    mock_truenas_client.get_pools.return_value = mock_pools
    mock_truenas_client.get_datasets.return_value = mock_datasets
    mock_truenas_client.get_snapshots.return_value = mock_snapshots
    
    result = monitor.analyze_storage_efficiency()
    
    assert "pools" in result
    assert "datasets" in result
    assert "snapshots" in result
    assert "efficiency_metrics" in result
    assert "recommendations" in result
    
    # Check pool metrics
    pool_metrics = result["pools"][0]
    assert pool_metrics["name"] == "tank"
    assert pool_metrics["usage_percent"] == 50.0  # 512GB / 1TB
    
    # Check efficiency metrics
    assert "total_pool_size" in result["efficiency_metrics"]
    assert "total_used_space" in result["efficiency_metrics"]
    assert "snapshot_overhead_gb" in result["efficiency_metrics"]


def test_validate_configuration(monitor, mock_k8s_client, mock_truenas_client):
    """Test configuration validation."""
    # Mock successful connections
    mock_k8s_client.test_connection.return_value = True
    mock_truenas_client.test_connection.return_value = True
    
    # Mock some basic resources
    mock_k8s_client.get_storage_classes.return_value = [
        Mock(name="democratic-csi-nfs", provisioner="org.democratic-csi.nfs")
    ]
    mock_truenas_client.get_pools.return_value = [
        Mock(name="tank", healthy=True)
    ]
    
    result = monitor.validate_configuration()
    
    assert result["k8s_connectivity"] is True
    assert result["truenas_connectivity"] is True
    assert len(result["storage_classes"]) == 1
    assert len(result["truenas_pools"]) == 1
    assert result["overall_status"] == "healthy"


def test_validate_configuration_failures(monitor, mock_k8s_client, mock_truenas_client):
    """Test configuration validation with failures."""
    # Mock connection failures
    mock_k8s_client.test_connection.side_effect = Exception("K8s connection failed")
    mock_truenas_client.test_connection.side_effect = Exception("TrueNAS connection failed")
    
    result = monitor.validate_configuration()
    
    assert result["k8s_connectivity"] is False
    assert result["truenas_connectivity"] is False
    assert result["overall_status"] == "unhealthy"
    assert len(result["errors"]) == 2


def test_get_monitoring_summary(monitor, mock_k8s_client, mock_truenas_client):
    """Test monitoring summary generation."""
    # Mock various metrics
    mock_k8s_client.get_persistent_volumes.return_value = [Mock(), Mock(), Mock()]
    mock_k8s_client.get_volume_snapshots.return_value = [Mock(), Mock()]
    mock_truenas_client.get_volumes.return_value = [Mock(), Mock(), Mock(), Mock()]
    mock_truenas_client.get_snapshots.return_value = [Mock(), Mock(), Mock()]
    
    # Mock health check methods
    monitor.check_orphaned_pvs = Mock(return_value={"total_orphaned": 1})
    monitor.check_orphaned_volumes = Mock(return_value={"total_orphaned": 2})
    monitor.check_snapshot_health = Mock(return_value={
        "orphaned_resources": {"k8s_orphaned": [Mock()], "truenas_orphaned": [Mock()]}
    })
    
    result = monitor.get_monitoring_summary()
    
    assert result["resources"]["k8s_pvs"] == 3
    assert result["resources"]["k8s_snapshots"] == 2
    assert result["resources"]["truenas_volumes"] == 4
    assert result["resources"]["truenas_snapshots"] == 3
    assert result["health"]["orphaned_pvs"] == 1
    assert result["health"]["orphaned_volumes"] == 2
    assert result["health"]["orphaned_snapshots"] == 2  # k8s + truenas


def test_run_health_check(monitor):
    """Test comprehensive health check."""
    # Mock all the health check methods
    monitor.validate_configuration = Mock(return_value={
        "overall_status": "healthy",
        "errors": []
    })
    monitor.check_orphaned_pvs = Mock(return_value={
        "total_orphaned": 0,
        "orphaned_pvs": []
    })
    monitor.check_orphaned_volumes = Mock(return_value={
        "total_orphaned": 1,
        "orphaned_volumes": [Mock()]
    })
    monitor.check_snapshot_health = Mock(return_value={
        "orphaned_resources": {
            "k8s_orphaned": [],
            "truenas_orphaned": [Mock()]
        }
    })
    
    result = monitor.run_health_check()
    
    assert "configuration" in result
    assert "orphaned_pvs" in result
    assert "orphaned_volumes" in result
    assert "snapshot_health" in result
    assert "summary" in result
    
    # Check that all methods were called
    monitor.validate_configuration.assert_called_once()
    monitor.check_orphaned_pvs.assert_called_once()
    monitor.check_orphaned_volumes.assert_called_once()
    monitor.check_snapshot_health.assert_called_once()


def test_analyze_trends(monitor, mock_truenas_client):
    """Test trend analysis."""
    # Mock historical snapshots with different ages
    now = datetime.now()
    mock_snapshots = [
        SnapshotInfo(
            name="recent",
            dataset="tank/k8s/vol1",
            creation_time=now - timedelta(hours=6),
            used_size=1073741824,  # 1GB
            referenced_size=2147483648,
            full_name="tank/k8s/vol1@recent"
        ),
        SnapshotInfo(
            name="daily",
            dataset="tank/k8s/vol1", 
            creation_time=now - timedelta(days=1),
            used_size=2147483648,  # 2GB
            referenced_size=4294967296,
            full_name="tank/k8s/vol1@daily"
        ),
        SnapshotInfo(
            name="weekly",
            dataset="tank/k8s/vol1",
            creation_time=now - timedelta(days=7),
            used_size=5368709120,  # 5GB
            referenced_size=10737418240,
            full_name="tank/k8s/vol1@weekly"
        )
    ]
    
    mock_truenas_client.analyze_snapshot_usage.return_value = {
        "total_snapshots": 3,
        "total_snapshot_size": 8589934592,  # 8GB
        "snapshots_by_age": {
            "last_24h": 2,
            "last_week": 1,
            "older": 0
        },
        "large_snapshots": [mock_snapshots[2]],
        "growth_trend": "increasing"
    }
    
    result = monitor.analyze_trends(days=7)
    
    assert result["period_days"] == 7
    assert result["snapshot_analysis"]["total_snapshots"] == 3
    assert result["snapshot_analysis"]["growth_trend"] == "increasing"
    assert len(result["recommendations"]) > 0