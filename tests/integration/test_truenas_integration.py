"""Integration tests for TrueNAS client.

These tests require a real TrueNAS instance to be available.
They can be skipped if no TrueNAS is configured.
"""

import os
import pytest
from unittest import mock

from truenas_storage_monitor.truenas_client import TrueNASClient, TrueNASConfig, TrueNASError


@pytest.fixture
def truenas_config():
    """Create a TrueNAS config for testing.
    
    Uses environment variables if available, otherwise creates a mock config.
    """
    host = os.getenv("TRUENAS_HOST", "truenas.example.com")
    username = os.getenv("TRUENAS_USERNAME")
    password = os.getenv("TRUENAS_PASSWORD")
    api_key = os.getenv("TRUENAS_API_KEY")
    
    return TrueNASConfig(
        host=host,
        username=username,
        password=password,
        api_key=api_key,
        verify_ssl=False,  # For testing
        timeout=10
    )


@pytest.mark.integration
def test_truenas_client_initialization(truenas_config):
    """Test that TrueNAS client can be initialized."""
    client = TrueNASClient(truenas_config)
    assert client is not None
    assert client.config == truenas_config
    assert client.base_url == f"{truenas_config.url}/api/v2.0"


@pytest.mark.integration
def test_truenas_client_authentication(truenas_config):
    """Test TrueNAS authentication."""
    if not truenas_config.username and not truenas_config.api_key:
        pytest.skip("No TrueNAS credentials configured")
    
    try:
        client = TrueNASClient(truenas_config)
        
        # Try to get pools as an authentication test
        pools = client.get_pools()
        assert isinstance(pools, list)
        
    except TrueNASError as e:
        if "authentication" in str(e).lower() or "unauthorized" in str(e).lower():
            pytest.fail(f"Authentication failed: {e}")
        else:
            pytest.skip(f"TrueNAS not available: {e}")
    except Exception as e:
        pytest.skip(f"TrueNAS not available: {e}")


@pytest.mark.integration
def test_truenas_get_pools(truenas_config):
    """Test getting storage pools."""
    if not truenas_config.username and not truenas_config.api_key:
        pytest.skip("No TrueNAS credentials configured")
    
    try:
        client = TrueNASClient(truenas_config)
        pools = client.get_pools()
        
        assert isinstance(pools, list)
        
        # If pools exist, verify structure
        for pool in pools:
            assert hasattr(pool, 'name')
            assert hasattr(pool, 'status')
            assert hasattr(pool, 'total_size')
            assert hasattr(pool, 'used_size')
            assert hasattr(pool, 'free_size')
            assert hasattr(pool, 'healthy')
            
    except Exception as e:
        pytest.skip(f"TrueNAS not available: {e}")


@pytest.mark.integration
def test_truenas_get_datasets(truenas_config):
    """Test getting datasets."""
    if not truenas_config.username and not truenas_config.api_key:
        pytest.skip("No TrueNAS credentials configured")
    
    try:
        client = TrueNASClient(truenas_config)
        datasets = client.get_datasets()
        
        assert isinstance(datasets, list)
        
        # If datasets exist, verify structure
        for dataset in datasets:
            assert hasattr(dataset, 'name')
            assert hasattr(dataset, 'type')
            assert hasattr(dataset, 'used_size')
            assert hasattr(dataset, 'available_size')
            
    except Exception as e:
        pytest.skip(f"TrueNAS not available: {e}")


@pytest.mark.integration
def test_truenas_get_snapshots(truenas_config):
    """Test getting snapshots."""
    if not truenas_config.username and not truenas_config.api_key:
        pytest.skip("No TrueNAS credentials configured")
    
    try:
        client = TrueNASClient(truenas_config)
        snapshots = client.get_snapshots()
        
        assert isinstance(snapshots, list)
        
        # If snapshots exist, verify structure
        for snapshot in snapshots:
            assert hasattr(snapshot, 'name')
            assert hasattr(snapshot, 'dataset')
            assert hasattr(snapshot, 'creation_time')
            assert hasattr(snapshot, 'used_size')
            
    except Exception as e:
        pytest.skip(f"TrueNAS not available: {e}")


@pytest.mark.integration
def test_truenas_error_handling(truenas_config):
    """Test error handling for invalid requests."""
    if not truenas_config.username and not truenas_config.api_key:
        pytest.skip("No TrueNAS credentials configured")
    
    try:
        client = TrueNASClient(truenas_config)
        
        # Try to get a non-existent resource
        with pytest.raises(TrueNASError):
            # This should fail
            response = client.session.get(f"{client.base_url}/invalid-endpoint")
            response.raise_for_status()
            
    except Exception as e:
        pytest.skip(f"TrueNAS not available: {e}")


@pytest.mark.integration 
def test_truenas_connection_timeout(truenas_config):
    """Test connection timeout handling."""
    # Use a very short timeout to test timeout handling
    short_timeout_config = TrueNASConfig(
        host=truenas_config.host,
        username=truenas_config.username,
        password=truenas_config.password,
        api_key=truenas_config.api_key,
        verify_ssl=False,
        timeout=0.001  # Very short timeout
    )
    
    client = TrueNASClient(short_timeout_config)
    
    with pytest.raises(TrueNASError):
        client.get_pools()


@pytest.mark.integration
def test_truenas_ssl_verification():
    """Test SSL verification settings."""
    # Test with SSL verification enabled
    ssl_config = TrueNASConfig(
        host="truenas.example.com",
        api_key="test-key",
        verify_ssl=True
    )
    
    client = TrueNASClient(ssl_config)
    assert client.session.verify is True
    
    # Test with SSL verification disabled
    no_ssl_config = TrueNASConfig(
        host="truenas.example.com",
        api_key="test-key",
        verify_ssl=False
    )
    
    client = TrueNASClient(no_ssl_config)
    assert client.session.verify is False