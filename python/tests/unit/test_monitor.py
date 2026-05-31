"""Unit tests for the Monitor class."""

import pytest
from unittest.mock import Mock, patch
from datetime import datetime, timedelta, timezone

from truenas_storage_monitor.monitor import Monitor
from truenas_storage_monitor.config import Config
from truenas_storage_monitor.exceptions import TrueNASMonitorError
from truenas_storage_monitor.k8s_client import (
    K8sConfig,
    PersistentVolumeClaimInfo,
    PersistentVolumeInfo,
)
from truenas_storage_monitor.truenas_client import TrueNASConfig, VolumeInfo


class TestMonitor:
    """Test cases for the Monitor class."""

    @pytest.fixture
    def mock_config(self):
        """Create a mock configuration with factory methods."""
        config = Mock(spec=Config)
        config.openshift = {"namespace": "democratic-csi"}
        config.k8s_config.return_value = K8sConfig(namespace="democratic-csi")
        config.truenas_config.return_value = TrueNASConfig(
            host="truenas.test",
            api_key="test-key",
        )
        config.orphan_threshold = timedelta(hours=24)
        config.snapshot_retention = timedelta(days=30)
        config.metrics_enabled = False
        config.cache_enabled = False
        return config

    @pytest.fixture
    def monitor(self, mock_config):
        """Create a Monitor instance with mocked dependencies."""
        with (
            patch("truenas_storage_monitor.monitor.K8sClient") as mock_k8s_cls,
            patch("truenas_storage_monitor.monitor.TrueNASClient") as mock_truenas_cls,
        ):
            mock_k8s_cls.return_value = Mock()
            mock_truenas_cls.return_value = Mock()
            monitor = Monitor(mock_config)
            return monitor

    def test_monitor_initialization(self, mock_config):
        """Test that Monitor initializes with typed client configs."""
        with (
            patch("truenas_storage_monitor.monitor.K8sClient") as mock_k8s,
            patch("truenas_storage_monitor.monitor.TrueNASClient") as mock_truenas,
        ):
            monitor = Monitor(mock_config)

            assert monitor.config == mock_config
            mock_config.k8s_config.assert_called_once()
            mock_config.truenas_config.assert_called_once()
            mock_k8s.assert_called_once_with(
                mock_config.k8s_config.return_value, inventory_cache=None
            )
            mock_truenas.assert_called_once_with(
                mock_config.truenas_config.return_value, inventory_cache=None
            )

    def test_monitor_initialization_with_cache_enabled(self, mock_config):
        """Monitor shares one InventoryCache between clients when enabled."""
        mock_config.cache_enabled = True
        mock_config.cache_ttl = timedelta(minutes=5)
        mock_config.cache_max_size = 1000

        with (
            patch("truenas_storage_monitor.monitor.K8sClient") as mock_k8s_cls,
            patch("truenas_storage_monitor.monitor.TrueNASClient") as mock_truenas_cls,
            patch("truenas_storage_monitor.monitor.InventoryCache") as mock_cache_cls,
        ):
            cache_instance = Mock()
            mock_cache_cls.return_value = cache_instance
            monitor = Monitor(mock_config)

            mock_cache_cls.assert_called_once_with(
                ttl=timedelta(minutes=5),
                max_size=1000,
                enabled=True,
            )
            mock_k8s_cls.assert_called_once_with(
                mock_config.k8s_config.return_value, inventory_cache=cache_instance
            )
            mock_truenas_cls.assert_called_once_with(
                mock_config.truenas_config.return_value, inventory_cache=cache_instance
            )
            assert monitor.config == mock_config

    def test_find_orphaned_resources_success(self, monitor):
        """Test successful orphaned resource detection."""
        old_created = utc_now() - timedelta(hours=25)

        mock_pvs = [
            PersistentVolumeInfo(
                name="pv-test",
                volume_handle="vol-test",
                driver="org.democratic-csi.nfs",
                capacity="10Gi",
                access_modes=["ReadWriteOnce"],
                phase="Bound",
                creation_time=old_created,
            )
        ]

        mock_pvcs = [
            PersistentVolumeClaimInfo(
                name="pvc-test",
                namespace="default",
                storage_class="truenas-iscsi",
                volume_name=None,
                capacity="5Gi",
                phase="Pending",
                creation_time=old_created,
            )
        ]

        monitor.k8s_client.get_persistent_volumes.return_value = mock_pvs
        monitor.k8s_client.get_persistent_volume_claims.return_value = mock_pvcs
        monitor.k8s_client.get_volume_snapshots.return_value = []
        monitor.truenas_client.get_volumes.return_value = []
        monitor.truenas_client.get_snapshots.return_value = []

        result = monitor.find_orphaned_resources()

        assert "timestamp" in result
        assert result["total_pvs"] == 1
        assert result["total_pvcs"] == 1
        assert len(result["orphaned_pvs"]) == 1
        assert result["orphaned_pvs"][0]["name"] == "pv-test"
        assert len(result["orphaned_pvcs"]) == 1
        assert result["orphaned_pvcs"][0]["name"] == "pvc-test"
        assert result["scan_duration"] >= 0
        assert "phase_timings" in result
        assert "k8s_pvs" in result["phase_timings"]

    def test_find_orphaned_resources_naive_creation_time(self, monitor):
        """Naive creation timestamps do not raise TypeError."""
        naive_created = datetime(2020, 1, 1, 0, 0, 0)

        mock_pvs = [
            PersistentVolumeInfo(
                name="pv-naive",
                volume_handle="vol-naive",
                driver="org.democratic-csi.nfs",
                capacity="10Gi",
                access_modes=["ReadWriteOnce"],
                phase="Bound",
                creation_time=naive_created,
            )
        ]

        monitor.k8s_client.get_persistent_volumes.return_value = mock_pvs
        monitor.k8s_client.get_persistent_volume_claims.return_value = []
        monitor.k8s_client.get_volume_snapshots.return_value = []
        monitor.truenas_client.get_volumes.return_value = []
        monitor.truenas_client.get_snapshots.return_value = []

        result = monitor.find_orphaned_resources()

        assert len(result["orphaned_pvs"]) == 1
        assert result["orphaned_pvs"][0]["name"] == "pv-naive"

    def test_find_orphaned_resources_error_handling(self, monitor):
        """Test error handling in orphaned resource detection."""
        monitor.k8s_client.get_persistent_volumes.side_effect = Exception("K8s API error")

        with pytest.raises(TrueNASMonitorError) as exc_info:
            monitor.find_orphaned_resources()

        assert "Failed to scan for orphaned resources" in str(exc_info.value)

    def test_is_democratic_csi_pv(self, monitor):
        """Test democratic-csi PV detection."""
        assert (
            monitor._is_democratic_csi_pv(
                PersistentVolumeInfo(
                    name="pv1",
                    volume_handle="v1",
                    driver="org.democratic-csi.iscsi",
                    capacity="1Gi",
                    access_modes=[],
                    phase="Bound",
                )
            )
            is True
        )
        assert (
            monitor._is_democratic_csi_pv(
                PersistentVolumeInfo(
                    name="pv2",
                    volume_handle="v2",
                    driver="other.csi.driver",
                    capacity="1Gi",
                    access_modes=[],
                    phase="Bound",
                )
            )
            is False
        )

    def test_parse_storage_size(self, monitor):
        """Test storage size parsing."""
        assert monitor._parse_storage_size("1Gi") == 1024**3
        assert monitor._parse_storage_size("5G") == 5 * 1024**3
        assert monitor._parse_storage_size("100Mi") == 100 * 1024**2
        assert monitor._parse_storage_size("1Ti") == 1024**4
        assert monitor._parse_storage_size("1024") == 1024
        assert monitor._parse_storage_size("") == 0
        assert monitor._parse_storage_size("invalid") == 0

    def test_analyze_storage_usage(self, monitor):
        """Test storage usage analysis."""
        mock_pvcs = [
            PersistentVolumeClaimInfo(
                name="pvc1",
                namespace="default",
                storage_class="sc1",
                volume_name="pv1",
                capacity="10Gi",
                phase="Bound",
            ),
            PersistentVolumeClaimInfo(
                name="pvc2",
                namespace="default",
                storage_class="sc1",
                volume_name="pv2",
                capacity="5Gi",
                phase="Bound",
            ),
        ]

        mock_pvs = [
            PersistentVolumeInfo(
                name="pv1",
                volume_handle="v1",
                driver="org.democratic-csi.nfs",
                capacity="10Gi",
                access_modes=[],
                phase="Bound",
            )
        ]
        mock_truenas_volumes = [
            VolumeInfo(name="vol1", path="/mnt/1", size=5 * 1024**3, type="FILE", enabled=True),
            VolumeInfo(name="vol2", path="/mnt/2", size=3 * 1024**3, type="FILE", enabled=True),
        ]

        monitor.k8s_client.get_persistent_volume_claims.return_value = mock_pvcs
        monitor.k8s_client.get_persistent_volumes.return_value = mock_pvs
        monitor.truenas_client.get_volumes.return_value = mock_truenas_volumes

        result = monitor.analyze_storage_usage()

        assert result["total_allocated_gb"] == 15.0
        assert result["total_used_gb"] == 8.0
        assert result["total_pvcs"] == 2
        assert result["total_pvs"] == 1
        assert "thin_provisioning_efficiency" in result
        assert "recommendations" in result

    def test_check_health(self, monitor):
        """Test health check functionality."""
        monitor.k8s_client.test_connection.return_value = True
        monitor.truenas_client.test_connection.return_value = True
        monitor.k8s_client.check_csi_driver_health.return_value = {
            "healthy": True,
            "reason": "All pods running and ready",
            "total_pods": 2,
            "running_pods": 2,
            "ready_pods": 2,
        }

        result = monitor.check_health()

        assert result["healthy"] is True
        assert result["components"]["kubernetes"]["healthy"] is True
        assert result["components"]["truenas"]["healthy"] is True
        assert result["components"]["csi_driver"]["healthy"] is True

    def test_check_health_with_failures(self, monitor):
        """Test health check with component failures."""
        monitor.k8s_client.test_connection.side_effect = Exception("Connection failed")
        monitor.truenas_client.test_connection.return_value = True
        monitor.k8s_client.check_csi_driver_health.return_value = {
            "healthy": False,
            "reason": "No CSI driver pods found",
        }

        result = monitor.check_health()

        assert result["healthy"] is False
        assert result["components"]["kubernetes"]["healthy"] is False
        assert result["components"]["truenas"]["healthy"] is True
        assert result["components"]["csi_driver"]["healthy"] is False

    def test_generate_recommendations(self, monitor):
        """Test recommendation generation."""
        mock_pvcs = [
            PersistentVolumeClaimInfo(
                name="large-pvc",
                namespace="default",
                storage_class="sc1",
                volume_name="pv1",
                capacity="200Gi",
                phase="Bound",
            ),
            PersistentVolumeClaimInfo(
                name="normal-pvc",
                namespace="default",
                storage_class="sc1",
                volume_name="pv2",
                capacity="10Gi",
                phase="Bound",
            ),
        ]

        mock_truenas_volumes = [
            VolumeInfo(name="vol1", path="/1", size=1, type="FILE", enabled=True),
            VolumeInfo(name="vol2", path="/2", size=1, type="FILE", enabled=True),
            VolumeInfo(name="vol3", path="/3", size=1, type="FILE", enabled=True),
        ]

        recommendations = monitor._generate_recommendations(mock_pvcs, mock_truenas_volumes)

        assert len(recommendations) >= 1
        assert any("large-pvc" in rec for rec in recommendations)
        assert any("unused TrueNAS volumes" in rec for rec in recommendations)


def utc_now():
    """Local helper mirroring production UTC helper."""
    return datetime.now(timezone.utc)
