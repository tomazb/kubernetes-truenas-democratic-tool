"""Unit tests for TrueNAS client."""

import pytest
from unittest.mock import Mock, patch, MagicMock
from datetime import datetime
import json

from truenas_storage_monitor.truenas_client import (
    TrueNASClient,
    TrueNASConfig,
    VolumeInfo,
    SnapshotInfo,
    PoolInfo,
    DatasetInfo,
    TrueNASError,
    AuthenticationError,
)


class TestTrueNASConfig:
    """Test TrueNASConfig validation."""

    def test_valid_config(self):
        """Test creating valid configuration."""
        config = TrueNASConfig(
            host="truenas.example.com",
            port=443,
            api_key="test-api-key",
            verify_ssl=True,
        )
        assert config.host == "truenas.example.com"
        assert config.port == 443
        assert config.api_key == "test-api-key"
        assert config.verify_ssl is True
        assert config.base_url == "https://truenas.example.com:443/api/v2.0"

    def test_config_with_username_password(self):
        """Test configuration with username/password."""
        config = TrueNASConfig(
            host="truenas.example.com",
            username="admin",
            password="secret",
        )
        assert config.username == "admin"
        assert config.password == "secret"
        assert config.api_key is None

    def test_config_validation_no_auth(self):
        """Test configuration validation with no authentication."""
        with pytest.raises(ValueError, match="Either api_key or username/password"):
            TrueNASConfig(host="truenas.example.com")


class TestTrueNASClient:
    """Test TrueNASClient functionality."""

    @pytest.fixture
    def mock_config(self):
        """Create mock TrueNAS configuration."""
        return TrueNASConfig(
            host="truenas.example.com",
            port=443,
            api_key="test-api-key",
            verify_ssl=False,
        )

    @pytest.fixture
    def mock_client(self, mock_config):
        """Create TrueNASClient with mocked session."""
        with patch('truenas_storage_monitor.truenas_client.requests.Session'):
            client = TrueNASClient(mock_config)
            client.session = Mock()
            return client

    def test_client_initialization(self, mock_config):
        """Test client initialization."""
        with patch('truenas_storage_monitor.truenas_client.requests.Session'):
            client = TrueNASClient(mock_config)
            assert client.config == mock_config
            assert client.base_url == "https://truenas.example.com:443/api/v2.0"

    def test_authentication_with_api_key(self, mock_client):
        """Test authentication with API key."""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {"username": "root"}
        mock_client.session.get.return_value = mock_response
        
        result = mock_client.test_connection()
        
        assert result is True
        mock_client.session.get.assert_called_once_with(
            f"{mock_client.base_url}/auth/me",
            timeout=30
        )

    def test_authentication_failure(self, mock_client):
        """Test authentication failure."""
        mock_response = Mock()
        mock_response.status_code = 401
        mock_response.text = "Unauthorized"
        mock_client.session.get.return_value = mock_response
        
        with pytest.raises(AuthenticationError):
            mock_client.test_connection()

    def test_get_pools(self, mock_client):
        """Test getting storage pools."""
        mock_pools = [
            {
                "id": 1,
                "name": "tank",
                "status": "ONLINE",
                "size": 1099511627776,  # 1TB
                "allocated": 549755813888,  # 512GB
                "free": 549755813888,
                "fragmentation": "5%",
                "healthy": True,
                "scan": {"state": "FINISHED"},
            }
        ]
        
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = mock_pools
        mock_client.session.get.return_value = mock_response
        
        pools = mock_client.get_pools()
        
        assert len(pools) == 1
        assert pools[0].name == "tank"
        assert pools[0].status == "ONLINE"
        assert pools[0].total_size == 1099511627776
        assert pools[0].used_size == 549755813888

    def test_get_datasets(self, mock_client):
        """Test getting datasets."""
        mock_datasets = [
            {
                "id": "tank/k8s",
                "name": "tank/k8s",
                "type": "FILESYSTEM",
                "used": {"value": 107374182400},  # 100GB
                "available": {"value": 442381127680},  # 412GB
                "quota": {"value": 0},
                "refquota": {"value": 0},
                "compression": "lz4",
                "compressratio": "1.5x",
            }
        ]
        
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = mock_datasets
        mock_client.session.get.return_value = mock_response
        
        datasets = mock_client.get_datasets()
        
        assert len(datasets) == 1
        assert datasets[0].name == "tank/k8s"
        assert datasets[0].used_size == 107374182400
        assert datasets[0].available_size == 442381127680

    def test_get_volumes(self, mock_client):
        """Test getting iSCSI volumes."""
        mock_extents = [
            {
                "id": 1,
                "name": "pvc-abc123",
                "type": "FILE",
                "path": "/mnt/tank/k8s/volumes/pvc-abc123",
                "filesize": 10737418240,  # 10GB
                "naa": "naa.6589cfc0000000b4c7f2f0e8a91b6f3d",
                "enabled": True,
                "ro": False,
            }
        ]
        
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = mock_extents
        mock_client.session.get.return_value = mock_response
        
        volumes = mock_client.get_volumes()
        
        assert len(volumes) == 1
        assert volumes[0].name == "pvc-abc123"
        assert volumes[0].size == 10737418240
        assert volumes[0].path == "/mnt/tank/k8s/volumes/pvc-abc123"

    def test_get_nfs_shares(self, mock_client):
        """Test getting NFS shares."""
        mock_shares = [
            {
                "id": 1,
                "path": "/mnt/tank/k8s/nfs/pvc-def456",
                "comment": "Democratic CSI NFS share",
                "enabled": True,
                "hosts": ["10.0.0.0/24"],
                "mapall_user": "root",
                "mapall_group": "wheel",
            }
        ]
        
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = mock_shares
        mock_client.session.get.return_value = mock_response
        
        shares = mock_client.get_nfs_shares()
        
        assert len(shares) == 1
        assert shares[0]["path"] == "/mnt/tank/k8s/nfs/pvc-def456"
        assert shares[0]["enabled"] is True

    def test_get_snapshots(self, mock_client):
        """Test getting ZFS snapshots."""
        mock_snapshots = [
            {
                "id": "tank/k8s/volumes/pvc-abc123@snapshot-1",
                "name": "tank/k8s/volumes/pvc-abc123@snapshot-1",
                "dataset": "tank/k8s/volumes/pvc-abc123",
                "snapshot_name": "snapshot-1",
                "properties": {
                    "used": {"value": "1073741824"},  # 1GB
                    "referenced": {"value": "10737418240"},  # 10GB
                    "creation": {"value": "1704067200"},  # Unix timestamp
                },
            }
        ]
        
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = mock_snapshots
        mock_client.session.get.return_value = mock_response
        
        snapshots = mock_client.get_snapshots()
        
        assert len(snapshots) == 1
        assert snapshots[0].name == "snapshot-1"
        assert snapshots[0].dataset == "tank/k8s/volumes/pvc-abc123"
        assert snapshots[0].used_size == 1073741824

    def test_get_volume_snapshots(self, mock_client):
        """Test getting snapshots for a specific volume."""
        volume_name = "pvc-abc123"
        mock_snapshots = [
            {
                "id": f"tank/k8s/volumes/{volume_name}@snapshot-1",
                "dataset": f"tank/k8s/volumes/{volume_name}",
                "snapshot_name": "snapshot-1",
                "properties": {
                    "used": {"value": "1073741824"},
                    "creation": {"value": "1704067200"},
                },
            }
        ]
        
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = mock_snapshots
        mock_client.session.get.return_value = mock_response
        
        snapshots = mock_client.get_volume_snapshots(volume_name)
        
        assert len(snapshots) == 1
        assert snapshots[0].name == "snapshot-1"
        mock_client.session.get.assert_called_with(
            f"{mock_client.base_url}/zfs/snapshot",
            params={"dataset__startswith": f"tank/k8s/volumes/{volume_name}"},
            timeout=30
        )

    def test_create_snapshot(self, mock_client):
        """Test creating a snapshot."""
        dataset = "tank/k8s/volumes/pvc-abc123"
        snapshot_name = "manual-snapshot-1"
        
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "id": f"{dataset}@{snapshot_name}",
            "dataset": dataset,
            "snapshot_name": snapshot_name,
        }
        mock_client.session.post.return_value = mock_response
        
        result = mock_client.create_snapshot(dataset, snapshot_name)
        
        assert result["id"] == f"{dataset}@{snapshot_name}"
        mock_client.session.post.assert_called_with(
            f"{mock_client.base_url}/zfs/snapshot",
            json={
                "dataset": dataset,
                "name": snapshot_name,
                "recursive": False,
            },
            timeout=30
        )

    def test_delete_snapshot(self, mock_client):
        """Test deleting a snapshot."""
        snapshot_id = "tank/k8s/volumes/pvc-abc123@snapshot-1"
        
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = True
        mock_client.session.delete.return_value = mock_response
        
        result = mock_client.delete_snapshot(snapshot_id)
        
        assert result is True
        mock_client.session.delete.assert_called_with(
            f"{mock_client.base_url}/zfs/snapshot/id/{snapshot_id}",
            timeout=30
        )

    def test_get_dataset_usage(self, mock_client):
        """Test getting dataset usage statistics."""
        dataset = "tank/k8s"
        
        mock_dataset_info = {
            "id": dataset,
            "name": dataset,
            "used": {"value": 107374182400},
            "available": {"value": 442381127680},
            "referenced": {"value": 53687091200},
            "quota": {"value": 0},
            "children": [
                {
                    "name": "tank/k8s/volumes",
                    "used": {"value": 53687091200},
                }
            ],
        }
        
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = [mock_dataset_info]
        mock_client.session.get.return_value = mock_response
        
        usage = mock_client.get_dataset_usage(dataset)
        
        assert usage["used"] == 107374182400
        assert usage["available"] == 442381127680
        assert len(usage["children"]) == 1

    def test_find_orphaned_volumes(self, mock_client):
        """Test finding orphaned TrueNAS volumes."""
        # Mock iSCSI extents
        mock_extents = [
            {"name": "pvc-orphaned", "path": "/mnt/tank/k8s/volumes/pvc-orphaned"},
            {"name": "pvc-active", "path": "/mnt/tank/k8s/volumes/pvc-active"},
        ]
        
        # Mock NFS shares
        mock_shares = [
            {"path": "/mnt/tank/k8s/nfs/pvc-nfs-orphaned"},
            {"path": "/mnt/tank/k8s/nfs/pvc-nfs-active"},
        ]
        
        mock_response1 = Mock()
        mock_response1.status_code = 200
        mock_response1.json.return_value = mock_extents
        
        mock_response2 = Mock()
        mock_response2.status_code = 200
        mock_response2.json.return_value = mock_shares
        
        mock_client.session.get.side_effect = [mock_response1, mock_response2]
        
        # K8s volumes to check against
        k8s_volumes = ["pvc-active", "pvc-nfs-active"]
        
        orphans = mock_client.find_orphaned_volumes(k8s_volumes)
        
        assert len(orphans) == 2
        assert any(o.name == "pvc-orphaned" for o in orphans)
        assert any(o.name == "pvc-nfs-orphaned" for o in orphans)

    def test_error_handling(self, mock_client):
        """Test error handling for API failures."""
        mock_response = Mock()
        mock_response.status_code = 500
        mock_response.text = "Internal Server Error"
        mock_response.raise_for_status.side_effect = Exception("Server Error")
        mock_client.session.get.return_value = mock_response
        
        with pytest.raises(TrueNASError):
            mock_client.get_pools()

    def test_connection_timeout(self, mock_client):
        """Test handling connection timeouts."""
        mock_client.session.get.side_effect = TimeoutError("Connection timeout")
        
        with pytest.raises(TrueNASError, match="timeout"):
            mock_client.get_pools()

    def test_pagination(self, mock_client):
        """Test handling paginated responses."""
        # First page
        mock_response1 = Mock()
        mock_response1.status_code = 200
        mock_response1.json.return_value = [{"id": 1, "name": "vol1"}]
        mock_response1.headers = {"X-Total-Count": "2"}
        
        # Second page
        mock_response2 = Mock()
        mock_response2.status_code = 200
        mock_response2.json.return_value = [{"id": 2, "name": "vol2"}]
        mock_response2.headers = {"X-Total-Count": "2"}
        
        mock_client.session.get.side_effect = [mock_response1, mock_response2]
        
        # Assuming get_all_pages method exists
        mock_client._get_all_pages = Mock(return_value=[
            {"id": 1, "name": "vol1"},
            {"id": 2, "name": "vol2"},
        ])
        
        result = mock_client._get_all_pages("/some/endpoint")
        assert len(result) == 2