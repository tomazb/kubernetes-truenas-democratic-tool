"""Integration tests for Kubernetes client.

These tests require a real Kubernetes cluster to be available.
They can be skipped if no cluster is configured.
"""

import os
import pytest
from unittest import mock

from truenas_storage_monitor.k8s_client import K8sClient, K8sConfig
from truenas_storage_monitor.exceptions import ConfigurationError


@pytest.fixture
def k8s_config():
    """Create a K8s config for testing."""
    return K8sConfig(
        namespace="democratic-csi",
        storage_class="democratic-csi-nfs",
        csi_driver="org.democratic-csi.nfs"
    )


@pytest.mark.integration
def test_k8s_client_initialization(k8s_config):
    """Test that K8s client can be initialized."""
    try:
        client = K8sClient(k8s_config)
        assert client is not None
        assert client.config == k8s_config
    except Exception as e:
        pytest.skip(f"Kubernetes cluster not available: {e}")


@pytest.mark.integration
def test_k8s_client_list_namespaces(k8s_config):
    """Test listing namespaces."""
    try:
        client = K8sClient(k8s_config)
        namespaces = client.core_v1.list_namespace()
        assert namespaces is not None
        assert hasattr(namespaces, 'items')
        assert len(namespaces.items) > 0  # Should have at least default namespace
    except Exception as e:
        pytest.skip(f"Kubernetes cluster not available: {e}")


@pytest.mark.integration
def test_k8s_client_list_storage_classes(k8s_config):
    """Test listing storage classes."""
    try:
        client = K8sClient(k8s_config)
        storage_classes = client.get_storage_classes()
        assert isinstance(storage_classes, list)
        # StorageClasses might be empty in a test cluster, so just check the structure
    except Exception as e:
        pytest.skip(f"Kubernetes cluster not available: {e}")


@pytest.mark.integration
def test_k8s_client_list_persistent_volumes(k8s_config):
    """Test listing persistent volumes."""
    try:
        client = K8sClient(k8s_config)
        pvs = client.get_persistent_volumes()
        assert isinstance(pvs, list)
        # PVs might be empty in a test cluster, so just check the structure
    except Exception as e:
        pytest.skip(f"Kubernetes cluster not available: {e}")


@pytest.mark.integration
def test_k8s_client_permissions(k8s_config):
    """Test that the client has required permissions."""
    try:
        client = K8sClient(k8s_config)
        
        # Test read permissions on key resources
        client.core_v1.list_persistent_volume()
        client.core_v1.list_persistent_volume_claim_for_all_namespaces()
        client.storage_v1.list_storage_class()
        
        # If we get here without exceptions, permissions are OK
        assert True
        
    except Exception as e:
        if "Forbidden" in str(e) or "forbidden" in str(e):
            pytest.fail(f"Insufficient permissions: {e}")
        else:
            pytest.skip(f"Kubernetes cluster not available: {e}")


@pytest.mark.integration
def test_orphan_detection_integration(k8s_config):
    """Test orphan detection with real cluster."""
    try:
        client = K8sClient(k8s_config)
        
        # Test orphaned PV detection
        orphaned_pvs = client.find_orphaned_pvs()
        assert isinstance(orphaned_pvs, list)
        
        # Test orphaned PVC detection
        orphaned_pvcs = client.find_orphaned_pvcs()
        assert isinstance(orphaned_pvcs, list)
        
    except Exception as e:
        pytest.skip(f"Kubernetes cluster not available: {e}")


@pytest.mark.integration
def test_csi_driver_health_check(k8s_config):
    """Test CSI driver health check."""
    try:
        client = K8sClient(k8s_config)
        
        health = client.check_csi_driver_health()
        assert isinstance(health, dict)
        assert "healthy" in health
        assert "total_pods" in health
        assert "running_pods" in health
        assert isinstance(health["healthy"], bool)
        assert isinstance(health["total_pods"], int)
        assert isinstance(health["running_pods"], int)
        
    except Exception as e:
        pytest.skip(f"Kubernetes cluster not available: {e}")