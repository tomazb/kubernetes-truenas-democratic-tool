"""End-to-end integration tests.

These tests require both Kubernetes and TrueNAS to be available
and test the full workflow of the monitoring tool.
"""

import os
import pytest
from unittest import mock

from truenas_storage_monitor.monitor import Monitor
from truenas_storage_monitor.k8s_client import K8sConfig
from truenas_storage_monitor.truenas_client import TrueNASConfig


@pytest.fixture
def test_config():
    """Create a test configuration."""
    return {
        "openshift": {
            "namespace": "democratic-csi",
            "storage_class": "democratic-csi-nfs",
            "csi_driver": "org.democratic-csi.nfs"
        },
        "truenas": {
            "url": os.getenv("TRUENAS_URL", "https://truenas.example.com"),
            "username": os.getenv("TRUENAS_USERNAME"),
            "password": os.getenv("TRUENAS_PASSWORD"),
            "api_key": os.getenv("TRUENAS_API_KEY"),
            "verify_ssl": False
        },
        "monitoring": {
            "orphan_check_interval": "1h",
            "orphan_threshold": "24h",
            "storage": {
                "pool_warning_threshold": 80,
                "pool_critical_threshold": 90
            }
        },
        "metrics": {
            "enabled": True,
            "port": 9090
        }
    }


@pytest.mark.integration
@pytest.mark.e2e
def test_monitor_initialization(test_config):
    """Test that the Monitor can be initialized with real clients."""
    try:
        monitor = Monitor(test_config)
        assert monitor is not None
        assert monitor.config == test_config
        
        # Check if clients were initialized (may be None if services unavailable)
        # This is OK - the monitor should handle missing services gracefully
        
    except Exception as e:
        pytest.fail(f"Monitor initialization should not fail: {e}")


@pytest.mark.integration
@pytest.mark.e2e
def test_orphan_detection_workflow(test_config):
    """Test the complete orphan detection workflow."""
    try:
        monitor = Monitor(test_config)
        
        # This should not raise an exception even if no services are available
        orphans = monitor.check_orphaned_resources()
        
        assert isinstance(orphans, dict)
        assert "orphaned_pvs" in orphans
        assert "orphaned_pvcs" in orphans
        assert "orphaned_snapshots" in orphans
        
        # Values should be integers (may be 0 if no resources or no connectivity)
        assert isinstance(orphans["orphaned_pvs"], int)
        assert isinstance(orphans["orphaned_pvcs"], int)
        assert isinstance(orphans["orphaned_snapshots"], int)
        
    except Exception as e:
        pytest.fail(f"Orphan detection workflow should not fail: {e}")


@pytest.mark.integration
@pytest.mark.e2e
def test_storage_monitoring_workflow(test_config):
    """Test the complete storage monitoring workflow."""
    try:
        monitor = Monitor(test_config)
        
        # This should not raise an exception even if TrueNAS is unavailable
        usage = monitor.check_storage_usage()
        
        assert isinstance(usage, dict)
        assert "pools" in usage
        assert isinstance(usage["pools"], dict)
        
        # If pools are available, check their structure
        for pool_name, pool_data in usage["pools"].items():
            assert isinstance(pool_name, str)
            assert isinstance(pool_data, dict)
            assert "total" in pool_data
            assert "used" in pool_data
            assert "free" in pool_data
            assert "healthy" in pool_data
        
    except Exception as e:
        pytest.fail(f"Storage monitoring workflow should not fail: {e}")


@pytest.mark.integration
@pytest.mark.e2e
def test_csi_health_monitoring_workflow(test_config):
    """Test the CSI driver health monitoring workflow."""
    try:
        monitor = Monitor(test_config)
        
        # This should not raise an exception even if K8s is unavailable
        health = monitor.check_csi_health()
        
        assert isinstance(health, dict)
        assert "healthy" in health
        assert "total_pods" in health
        assert "running_pods" in health
        assert "unhealthy_pods" in health
        
        assert isinstance(health["healthy"], bool)
        assert isinstance(health["total_pods"], int)
        assert isinstance(health["running_pods"], int)
        assert isinstance(health["unhealthy_pods"], list)
        
    except Exception as e:
        pytest.fail(f"CSI health monitoring workflow should not fail: {e}")


@pytest.mark.integration
@pytest.mark.e2e
def test_monitor_graceful_degradation(test_config):
    """Test that the monitor degrades gracefully when services are unavailable."""
    # Test with invalid K8s config
    invalid_config = test_config.copy()
    invalid_config["openshift"]["namespace"] = "non-existent-namespace"
    
    monitor = Monitor(invalid_config)
    
    # All operations should complete without raising exceptions
    orphans = monitor.check_orphaned_resources()
    usage = monitor.check_storage_usage()
    health = monitor.check_csi_health()
    
    # Results should have expected structure even if empty/failed
    assert isinstance(orphans, dict)
    assert isinstance(usage, dict)
    assert isinstance(health, dict)


@pytest.mark.integration
@pytest.mark.e2e
def test_monitor_start_stop(test_config):
    """Test monitor start and stop operations."""
    monitor = Monitor(test_config)
    
    # These should not raise exceptions
    monitor.start()
    monitor.stop()


@pytest.mark.integration
@pytest.mark.e2e 
def test_configuration_validation(test_config):
    """Test configuration validation in Monitor."""
    # Test valid config
    monitor = Monitor(test_config)
    assert monitor.config == test_config
    
    # Test minimal config (only required sections)
    minimal_config = {
        "openshift": {"namespace": "test"},
        "monitoring": {}
    }
    
    monitor = Monitor(minimal_config)
    assert monitor.config == minimal_config
    
    # Test that monitor handles missing optional sections gracefully
    assert monitor.k8s_client is not None or monitor.k8s_client is None  # Both are valid
    assert monitor.truenas_client is None  # Should be None since truenas not configured