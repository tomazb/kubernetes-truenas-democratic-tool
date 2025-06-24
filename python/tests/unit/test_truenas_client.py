"""Unit tests for TrueNAS client."""

import pytest
from unittest.mock import Mock, patch, MagicMock
from datetime import datetime, timedelta
import requests

from truenas_storage_monitor.truenas_client import (
    TrueNASClient, TrueNASConfig, TrueNASError, AuthenticationError,
    PoolInfo, DatasetInfo, VolumeInfo, SnapshotInfo
)


@pytest.fixture
def truenas_config():
    """Create a test TrueNAS configuration."""
    return TrueNASConfig(
        host="truenas.example.com",
        port=443,
        username="admin",
        password="test-password",
        verify_ssl=False
    )


@pytest.fixture
def mock_session():
    """Create a mocked requests session."""
    return Mock(spec=requests.Session)


@pytest.fixture
def truenas_client(truenas_config, mock_session):
    """Create a TrueNAS client with mocked session."""
    with patch('truenas_storage_monitor.truenas_client.requests.Session', return_value=mock_session):
        client = TrueNASClient(truenas_config)
        client.session = mock_session
        return client


def test_truenas_config_validation():
    """Test TrueNAS configuration validation."""
    # Valid config with API key
    config = TrueNASConfig(host="test.com", api_key="test-key")
    assert config.api_key == "test-key"
    
    # Valid config with username/password
    config = TrueNASConfig(host="test.com", username="admin", password="secret")
    assert config.username == "admin"
    
    # Invalid config - missing credentials
    with pytest.raises(ValueError, match="Either api_key or username/password"):
        TrueNASConfig(host="test.com")


def test_truenas_config_base_url():
    """Test TrueNAS base URL generation."""
    config = TrueNASConfig(host="truenas.example.com", port=443, api_key="test")
    assert config.base_url == "https://truenas.example.com:443/api/v2.0"
    
    config = TrueNASConfig(host="truenas.example.com", port=80, api_key="test")
    assert config.base_url == "http://truenas.example.com:80/api/v2.0"


def test_truenas_client_initialization(truenas_config):
    """Test TrueNAS client initialization."""
    with patch('truenas_storage_monitor.truenas_client.requests.Session'):
        client = TrueNASClient(truenas_config)
        assert client.config == truenas_config


def test_test_connection_success(truenas_client, mock_session):
    """Test successful connection test."""
    mock_response = Mock()
    mock_response.status_code = 200
    mock_session.get.return_value = mock_response
    
    result = truenas_client.test_connection()
    assert result is True


def test_test_connection_auth_failure(truenas_client, mock_session):
    """Test connection test with authentication failure."""
    mock_response = Mock()
    mock_response.status_code = 401
    mock_session.get.return_value = mock_response
    
    with pytest.raises(AuthenticationError):
        truenas_client.test_connection()


def test_test_connection_network_error(truenas_client, mock_session):
    """Test connection test with network error."""
    mock_session.get.side_effect = requests.exceptions.ConnectionError("Connection failed")
    
    with pytest.raises(TrueNASError, match="Failed to connect"):
        truenas_client.test_connection()


def test_get_pools(truenas_client, mock_session):
    """Test getting storage pools."""
    mock_response = Mock()
    mock_response.json.return_value = [
        {
            "name": "tank",
            "status": "ONLINE",
            "size": 1000000000000,  # 1TB
            "allocated": 500000000000,  # 500GB
            "free": 500000000000,  # 500GB
            "fragmentation": "15%",
            "healthy": True,
            "scan": {"state": "FINISHED"}
        }
    ]
    mock_session.get.return_value = mock_response
    
    pools = truenas_client.get_pools()
    
    assert len(pools) == 1
    assert pools[0].name == "tank"
    assert pools[0].status == "ONLINE"
    assert pools[0].healthy is True


def test_get_datasets(truenas_client, mock_session):
    """Test getting datasets."""
    mock_response = Mock()
    mock_response.json.return_value = [
        {
            "id": "tank/k8s",
            "type": "FILESYSTEM",
            "used": {"value": 100000000},  # 100MB
            "available": {"value": 900000000},  # 900MB
            "referenced": {"value": 50000000},  # 50MB
            "quota": {"value": None},
            "compressratio": "1.5x"
        }
    ]
    mock_session.get.return_value = mock_response
    
    datasets = truenas_client.get_datasets()
    
    assert len(datasets) == 1
    assert datasets[0].name == "tank/k8s"
    assert datasets[0].type == "FILESYSTEM"


def test_get_volumes(truenas_client, mock_session):
    """Test getting iSCSI volumes."""
    mock_response = Mock()
    mock_response.json.return_value = [
        {
            "name": "pvc-123",
            "path": "/mnt/tank/k8s/iscsi/pvc-123",
            "filesize": 10737418240,  # 10GB
            "type": "FILE",
            "enabled": True,
            "naa": "0x6589cfc000000123",
            "serial": "pvc-123"
        }
    ]
    mock_session.get.return_value = mock_response
    
    volumes = truenas_client.get_volumes()
    
    assert len(volumes) == 1
    assert volumes[0].name == "pvc-123"
    assert volumes[0].size == 10737418240


def test_get_snapshots(truenas_client, mock_session):
    """Test getting snapshots."""
    creation_time = int(datetime.now().timestamp())
    
    mock_response = Mock()
    mock_response.json.return_value = [
        {
            "snapshot_name": "daily-2024-01-01",
            "dataset": "tank/k8s/volumes/pvc-123",
            "id": "tank/k8s/volumes/pvc-123@daily-2024-01-01",
            "properties": {
                "creation": {"value": str(creation_time)},
                "used": {"value": "1073741824"},  # 1GB
                "referenced": {"value": "2147483648"}  # 2GB
            }
        }
    ]
    mock_session.get.return_value = mock_response
    
    snapshots = truenas_client.get_snapshots()
    
    assert len(snapshots) == 1
    assert snapshots[0].name == "daily-2024-01-01"
    assert snapshots[0].dataset == "tank/k8s/volumes/pvc-123"
    assert snapshots[0].used_size == 1073741824


def test_create_snapshot(truenas_client, mock_session):
    """Test creating a snapshot."""
    mock_response = Mock()
    mock_response.json.return_value = {"id": "tank/dataset@snapshot"}
    mock_session.post.return_value = mock_response
    
    result = truenas_client.create_snapshot("tank/dataset", "snapshot")
    
    assert result["id"] == "tank/dataset@snapshot"
    mock_session.post.assert_called_once()


def test_delete_snapshot(truenas_client, mock_session):
    """Test deleting a snapshot."""
    mock_response = Mock()
    mock_session.delete.return_value = mock_response
    
    result = truenas_client.delete_snapshot("tank/dataset@snapshot")
    
    assert result is True
    mock_session.delete.assert_called_once()


def test_find_orphaned_volumes(truenas_client, mock_session):
    """Test finding orphaned volumes."""
    # Mock iSCSI volumes
    mock_iscsi_response = Mock()
    mock_iscsi_response.json.return_value = [
        {
            "name": "orphaned-volume",
            "path": "/mnt/tank/k8s/iscsi/orphaned-volume",
            "filesize": 10737418240,
            "type": "FILE",
            "enabled": True
        }
    ]
    
    # Mock NFS shares
    mock_nfs_response = Mock()
    mock_nfs_response.json.return_value = [
        {
            "path": "/mnt/tank/k8s/nfs/orphaned-nfs-volume"
        }
    ]
    
    # Mock session to return different responses for different endpoints
    def mock_get(url, **kwargs):
        if "iscsi/extent" in url:
            return mock_iscsi_response
        elif "sharing/nfs" in url:
            return mock_nfs_response
        return Mock()
    
    mock_session.get.side_effect = mock_get
    
    # K8s volume names (none match our orphaned volumes)
    k8s_volumes = ["active-volume-1", "active-volume-2"]
    
    orphans = truenas_client.find_orphaned_volumes(k8s_volumes)
    
    assert len(orphans) == 2  # One iSCSI orphan + one NFS orphan
    assert any(o.name == "orphaned-volume" for o in orphans)
    assert any(o.name == "orphaned-nfs-volume" for o in orphans)


def test_analyze_snapshot_usage(truenas_client, mock_session):
    """Test snapshot usage analysis."""
    # Create mock snapshots with various ages and sizes
    now = datetime.now()
    snapshots = [
        SnapshotInfo(
            name="recent",
            dataset="tank/k8s/vol1",
            creation_time=now - timedelta(hours=12),
            used_size=1024**3,  # 1GB
            referenced_size=2*1024**3,
            full_name="tank/k8s/vol1@recent"
        ),
        SnapshotInfo(
            name="old",
            dataset="tank/k8s/vol2",
            creation_time=now - timedelta(days=45),
            used_size=5*1024**3,  # 5GB - large
            referenced_size=10*1024**3,
            full_name="tank/k8s/vol2@old"
        )
    ]
    
    truenas_client.get_snapshots = Mock(return_value=snapshots)
    
    analysis = truenas_client.analyze_snapshot_usage()
    
    assert analysis["total_snapshots"] == 2
    assert analysis["total_snapshot_size"] == 6*1024**3  # 6GB total
    assert analysis["snapshots_by_age"]["last_24h"] == 1
    assert analysis["snapshots_by_age"]["older"] == 1
    assert len(analysis["large_snapshots"]) == 1  # One >1GB
    assert len(analysis["recommendations"]) > 0


def test_find_orphaned_truenas_snapshots(truenas_client, mock_session):
    """Test finding orphaned TrueNAS snapshots."""
    # Mock TrueNAS snapshots
    truenas_snapshots = [
        SnapshotInfo(
            name="orphaned-snap",
            dataset="tank/k8s/volumes/pvc-orphaned",
            creation_time=datetime.now() - timedelta(days=10),
            used_size=1024**3,
            referenced_size=2*1024**3,
            full_name="tank/k8s/volumes/pvc-orphaned@orphaned-snap"
        ),
        SnapshotInfo(
            name="matched-snap",
            dataset="tank/k8s/volumes/pvc-active",
            creation_time=datetime.now() - timedelta(days=5),
            used_size=1024**3,
            referenced_size=2*1024**3,
            full_name="tank/k8s/volumes/pvc-active@matched-snap"
        )
    ]
    
    truenas_client.get_snapshots = Mock(return_value=truenas_snapshots)
    
    # Mock K8s snapshots (only one matches)
    k8s_snapshots = [
        Mock(name="matched-snap", source_pvc="pvc-active")
    ]
    
    orphans = truenas_client.find_orphaned_truenas_snapshots(k8s_snapshots)
    
    # Should find the orphaned snapshot
    assert len(orphans) == 1
    assert orphans[0].name == "orphaned-snap"