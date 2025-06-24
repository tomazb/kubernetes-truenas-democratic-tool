"""Integration tests for TrueNAS API interactions.

These tests verify actual API behavior, error handling, and data parsing
using real HTTP interactions where possible.
"""

import pytest
import requests
import json
import time
from unittest.mock import Mock, patch
from datetime import datetime, timedelta

from truenas_storage_monitor.truenas_client import (
    TrueNASClient, TrueNASConfig, TrueNASError, AuthenticationError
)


class MockTrueNASServer:
    """Mock TrueNAS server for integration testing."""
    
    def __init__(self):
        self.endpoints = {}
        self.auth_required = True
        self.valid_credentials = {'admin': 'password'}
        
    def add_endpoint(self, path: str, method: str, response_data: dict, status_code: int = 200):
        """Add a mock endpoint."""
        key = f"{method.upper()}:{path}"
        self.endpoints[key] = {
            'data': response_data,
            'status_code': status_code
        }
    
    def setup_realistic_endpoints(self):
        """Setup realistic TrueNAS API endpoints."""
        # Version endpoint
        self.add_endpoint('/api/v2.0/system/version', 'GET', {
            'version': 'TrueNAS-SCALE-22.12.0'
        })
        
        # Pool endpoints
        self.add_endpoint('/api/v2.0/pool', 'GET', [
            {
                'name': 'tank',
                'status': 'ONLINE',
                'size': 1000000000000,
                'allocated': 500000000000,
                'free': 500000000000,
                'fragmentation': '15%',
                'healthy': True,
                'scan': {'state': 'FINISHED'}
            }
        ])
        
        # Dataset endpoints
        self.add_endpoint('/api/v2.0/pool/dataset', 'GET', [
            {
                'id': 'tank/k8s',
                'type': 'FILESYSTEM',
                'used': {'value': 100000000},
                'available': {'value': 900000000},
                'referenced': {'value': 50000000},
                'quota': {'value': None},
                'compressratio': '1.5x'
            }
        ])
        
        # iSCSI extent endpoints
        self.add_endpoint('/api/v2.0/iscsi/extent', 'GET', [
            {
                'name': 'pvc-123',
                'path': '/mnt/tank/k8s/iscsi/pvc-123',
                'filesize': 10737418240,
                'type': 'FILE',
                'enabled': True
            }
        ])
        
        # Snapshot endpoints
        self.add_endpoint('/api/v2.0/zfs/snapshot', 'GET', [
            {
                'snapshot_name': 'daily-2024-01-01',
                'dataset': 'tank/k8s/volumes/pvc-123',
                'id': 'tank/k8s/volumes/pvc-123@daily-2024-01-01',
                'properties': {
                    'creation': {'value': str(int(datetime.now().timestamp()))},
                    'used': {'value': '1073741824'},
                    'referenced': {'value': '2147483648'}
                }
            }
        ])


class TestTrueNASAPIContract:
    """Test TrueNAS API contract and behavior."""
    
    def test_truenas_config_validation(self):
        """Test TrueNAS configuration validation."""
        # Valid config with API key
        config = TrueNASConfig(
            host='truenas.example.com',
            api_key='test-key'
        )
        assert config.api_key == 'test-key'
        assert config.base_url == 'https://truenas.example.com:443/api/v2.0'
        
        # Valid config with username/password
        config = TrueNASConfig(
            host='truenas.example.com',
            username='admin',
            password='secret'
        )
        assert config.username == 'admin'
        assert config.password == 'secret'
        
        # Invalid config
        with pytest.raises(ValueError):
            TrueNASConfig(host='truenas.example.com')  # No credentials
    
    def test_http_session_configuration(self):
        """Test that HTTP session is configured correctly."""
        config = TrueNASConfig(
            host='truenas.example.com',
            username='admin',
            password='test',
            verify_ssl=False,
            timeout=30
        )
        
        client = TrueNASClient(config)
        
        # Check session configuration
        assert client.session.auth == ('admin', 'test')
        assert client.session.verify is False
        assert 'application/json' in client.session.headers['Content-Type']
        assert 'application/json' in client.session.headers['Accept']
    
    def test_api_error_handling(self):
        """Test API error handling with real HTTP responses."""
        config = TrueNASConfig(
            host='httpbin.org',  # Use httpbin for testing HTTP behavior
            username='admin',
            password='test'
        )
        
        client = TrueNASClient(config)
        
        # Test 404 handling
        with patch.object(client.session, 'get') as mock_get:
            mock_response = Mock()
            mock_response.status_code = 404
            mock_response.text = 'Not Found'
            mock_get.return_value = mock_response
            
            with pytest.raises(TrueNASError, match='404'):
                client._make_request('GET', '/nonexistent')
        
        # Test 401 handling
        with patch.object(client.session, 'get') as mock_get:
            mock_response = Mock()
            mock_response.status_code = 401
            mock_response.text = 'Unauthorized'
            mock_get.return_value = mock_response
            
            with pytest.raises(AuthenticationError):
                client._make_request('GET', '/api/v2.0/pool')
        
        # Test network error handling
        with patch.object(client.session, 'get') as mock_get:
            mock_get.side_effect = requests.exceptions.ConnectionError('Connection failed')
            
            with pytest.raises(TrueNASError, match='Failed to connect'):
                client._make_request('GET', '/api/v2.0/pool')
    
    def test_json_response_parsing(self):
        """Test JSON response parsing with various data types."""
        config = TrueNASConfig(
            host='truenas.example.com',
            username='admin',
            password='test'
        )
        
        client = TrueNASClient(config)
        
        # Test valid JSON parsing
        with patch.object(client.session, 'get') as mock_get:
            mock_response = Mock()
            mock_response.status_code = 200
            mock_response.json.return_value = {'test': 'data', 'number': 42}
            mock_get.return_value = mock_response
            
            result = client._make_request('GET', '/api/v2.0/test')
            assert result == {'test': 'data', 'number': 42}
        
        # Test invalid JSON handling
        with patch.object(client.session, 'get') as mock_get:
            mock_response = Mock()
            mock_response.status_code = 200
            mock_response.json.side_effect = json.JSONDecodeError('Invalid JSON', '', 0)
            mock_response.text = 'Invalid JSON response'
            mock_get.return_value = mock_response
            
            with pytest.raises(TrueNASError, match='Invalid JSON'):
                client._make_request('GET', '/api/v2.0/test')


class TestTrueNASDataParsing:
    """Test parsing of TrueNAS API responses."""
    
    def test_pool_data_parsing(self):
        """Test parsing of pool data."""
        config = TrueNASConfig(
            host='truenas.example.com',
            username='admin',
            password='test'
        )
        
        client = TrueNASClient(config)
        
        # Mock pool response
        pool_data = [
            {
                'name': 'tank',
                'status': 'ONLINE',
                'size': 1000000000000,
                'allocated': 500000000000,
                'free': 500000000000,
                'fragmentation': '15%',
                'healthy': True,
                'scan': {'state': 'FINISHED'}
            }
        ]
        
        with patch.object(client, '_make_request', return_value=pool_data):
            pools = client.get_pools()
            
            assert len(pools) == 1
            pool = pools[0]
            assert pool.name == 'tank'
            assert pool.status == 'ONLINE'
            assert pool.size == 1000000000000
            assert pool.healthy is True
    
    def test_snapshot_data_parsing(self):
        """Test parsing of snapshot data with various formats."""
        config = TrueNASConfig(
            host='truenas.example.com',
            username='admin',
            password='test'
        )
        
        client = TrueNASClient(config)
        
        # Mock snapshot response with realistic data
        snapshot_data = [
            {
                'snapshot_name': 'auto-2024-01-01-120000',
                'dataset': 'tank/k8s/volumes/pvc-test',
                'id': 'tank/k8s/volumes/pvc-test@auto-2024-01-01-120000',
                'properties': {
                    'creation': {'value': '1704110400'},  # Unix timestamp
                    'used': {'value': '1073741824'},      # 1GB in bytes
                    'referenced': {'value': '2147483648'} # 2GB in bytes
                }
            }
        ]
        
        with patch.object(client, '_make_request', return_value=snapshot_data):
            snapshots = client.get_snapshots()
            
            assert len(snapshots) == 1
            snapshot = snapshots[0]
            assert snapshot.name == 'auto-2024-01-01-120000'
            assert snapshot.dataset == 'tank/k8s/volumes/pvc-test'
            assert snapshot.used_size == 1073741824
            assert snapshot.referenced_size == 2147483648
            assert isinstance(snapshot.creation_time, datetime)
    
    def test_volume_data_parsing(self):
        """Test parsing of volume data."""
        config = TrueNASConfig(
            host='truenas.example.com',
            username='admin',
            password='test'
        )
        
        client = TrueNASClient(config)
        
        # Mock iSCSI extent response
        extent_data = [
            {
                'name': 'pvc-dynamic-test',
                'path': '/mnt/tank/k8s/iscsi/pvc-dynamic-test',
                'filesize': 21474836480,  # 20GB
                'type': 'FILE',
                'enabled': True,
                'naa': '0x6589cfc000000001'
            }
        ]
        
        with patch.object(client, '_make_request', return_value=extent_data):
            volumes = client.get_volumes()
            
            assert len(volumes) == 1
            volume = volumes[0]
            assert volume.name == 'pvc-dynamic-test'
            assert volume.size == 21474836480
            assert '/k8s/iscsi/' in volume.path


class TestTrueNASConnectionScenarios:
    """Test various connection scenarios."""
    
    def test_connection_timeout_handling(self):
        """Test connection timeout handling."""
        config = TrueNASConfig(
            host='httpbin.org',
            username='admin',
            password='test',
            timeout=1  # Very short timeout
        )
        
        client = TrueNASClient(config)
        
        # Test with a slow endpoint
        with patch.object(client.session, 'get') as mock_get:
            mock_get.side_effect = requests.exceptions.Timeout('Request timed out')
            
            with pytest.raises(TrueNASError, match='timeout'):
                client.test_connection()
    
    def test_ssl_verification_scenarios(self):
        """Test SSL verification scenarios."""
        # Test with SSL verification enabled
        config_ssl = TrueNASConfig(
            host='truenas.example.com',
            username='admin',
            password='test',
            verify_ssl=True
        )
        
        client_ssl = TrueNASClient(config_ssl)
        assert client_ssl.session.verify is True
        
        # Test with SSL verification disabled
        config_no_ssl = TrueNASConfig(
            host='truenas.example.com',
            username='admin',
            password='test',
            verify_ssl=False
        )
        
        client_no_ssl = TrueNASClient(config_no_ssl)
        assert client_no_ssl.session.verify is False
    
    def test_retry_mechanism(self):
        """Test retry mechanism for failed requests."""
        config = TrueNASConfig(
            host='truenas.example.com',
            username='admin',
            password='test',
            max_retries=3
        )
        
        client = TrueNASClient(config)
        
        # Test with intermittent failures
        with patch.object(client.session, 'get') as mock_get:
            # Fail twice, then succeed
            mock_get.side_effect = [
                requests.exceptions.ConnectionError('Temporary failure'),
                requests.exceptions.ConnectionError('Another failure'),
                Mock(status_code=200, json=lambda: {'success': True})
            ]
            
            # Should eventually succeed after retries
            result = client._make_request('GET', '/api/v2.0/test')
            assert result == {'success': True}
            assert mock_get.call_count == 3


class TestTrueNASRealWorldScenarios:
    """Test scenarios that mimic real-world usage."""
    
    def test_orphaned_volume_detection_logic(self):
        """Test the logic for detecting orphaned volumes."""
        config = TrueNASConfig(
            host='truenas.example.com',
            username='admin',
            password='test'
        )
        
        client = TrueNASClient(config)
        
        # Mock volumes that include both active and orphaned
        mock_volumes = [
            Mock(name='pvc-active-1', path='/mnt/tank/k8s/volumes/pvc-active-1'),
            Mock(name='pvc-active-2', path='/mnt/tank/k8s/volumes/pvc-active-2'),
            Mock(name='pvc-orphaned-1', path='/mnt/tank/k8s/volumes/pvc-orphaned-1'),
            Mock(name='old-manual-volume', path='/mnt/tank/manual/old-manual-volume'),
        ]
        
        with patch.object(client, 'get_volumes', return_value=mock_volumes):
            # Active volumes from K8s
            active_k8s_volumes = ['pvc-active-1', 'pvc-active-2']
            
            orphans = client.find_orphaned_volumes(active_k8s_volumes)
            
            # Should find orphaned volumes
            assert len(orphans) >= 1
            orphan_names = [v.name for v in orphans]
            assert 'pvc-orphaned-1' in orphan_names
            assert 'pvc-active-1' not in orphan_names
            assert 'pvc-active-2' not in orphan_names
    
    def test_snapshot_analysis_real_data(self):
        """Test snapshot analysis with realistic data patterns."""
        config = TrueNASConfig(
            host='truenas.example.com',
            username='admin',
            password='test'
        )
        
        client = TrueNASClient(config)
        
        # Create realistic snapshot data
        now = datetime.now()
        mock_snapshots = [
            Mock(
                name='hourly-01',
                dataset='tank/k8s/vol1',
                creation_time=now - timedelta(hours=1),
                used_size=500 * 1024**2,  # 500MB
                referenced_size=1024**3,   # 1GB
                full_name='tank/k8s/vol1@hourly-01'
            ),
            Mock(
                name='daily-01',
                dataset='tank/k8s/vol1',
                creation_time=now - timedelta(days=1),
                used_size=2 * 1024**3,    # 2GB
                referenced_size=4 * 1024**3,  # 4GB
                full_name='tank/k8s/vol1@daily-01'
            ),
            Mock(
                name='old-snapshot',
                dataset='tank/k8s/vol2',
                creation_time=now - timedelta(days=60),
                used_size=10 * 1024**3,   # 10GB - large and old
                referenced_size=20 * 1024**3,
                full_name='tank/k8s/vol2@old-snapshot'
            )
        ]
        
        with patch.object(client, 'get_snapshots', return_value=mock_snapshots):
            analysis = client.analyze_snapshot_usage()
            
            # Verify analysis structure
            assert 'total_snapshots' in analysis
            assert 'total_snapshot_size' in analysis
            assert 'snapshots_by_age' in analysis
            
            # Verify calculations
            assert analysis['total_snapshots'] == 3
            expected_size = (500 * 1024**2) + (2 * 1024**3) + (10 * 1024**3)
            assert analysis['total_snapshot_size'] == expected_size
            
            # Age categorization
            assert analysis['snapshots_by_age']['last_24h'] == 1
            assert analysis['snapshots_by_age']['last_week'] == 1
            assert analysis['snapshots_by_age']['older'] == 1
    
    def test_cross_system_snapshot_matching(self):
        """Test matching snapshots between TrueNAS and K8s."""
        config = TrueNASConfig(
            host='truenas.example.com',
            username='admin',
            password='test'
        )
        
        client = TrueNASClient(config)
        
        # TrueNAS snapshots
        truenas_snapshots = [
            Mock(
                name='k8s-snapshot-1',
                dataset='tank/k8s/volumes/pvc-test',
                full_name='tank/k8s/volumes/pvc-test@k8s-snapshot-1'
            ),
            Mock(
                name='orphaned-snapshot',
                dataset='tank/k8s/volumes/pvc-deleted',
                full_name='tank/k8s/volumes/pvc-deleted@orphaned-snapshot'
            ),
            Mock(
                name='manual-snapshot',
                dataset='tank/k8s/volumes/pvc-test',
                full_name='tank/k8s/volumes/pvc-test@manual-snapshot'
            )
        ]
        
        # K8s snapshots
        k8s_snapshots = [
            Mock(name='k8s-snapshot-1', source_pvc='pvc-test'),
            Mock(name='k8s-snapshot-2', source_pvc='pvc-other')
        ]
        
        with patch.object(client, 'get_snapshots', return_value=truenas_snapshots):
            orphans = client.find_orphaned_truenas_snapshots(k8s_snapshots)
            
            # Should find orphaned and manual snapshots
            orphan_names = [s.name for s in orphans]
            assert 'orphaned-snapshot' in orphan_names
            assert 'manual-snapshot' in orphan_names
            assert 'k8s-snapshot-1' not in orphan_names  # This one matches


class TestTrueNASPerformance:
    """Test performance characteristics of TrueNAS operations."""
    
    def test_large_dataset_handling(self):
        """Test handling of large numbers of volumes/snapshots."""
        config = TrueNASConfig(
            host='truenas.example.com',
            username='admin',
            password='test'
        )
        
        client = TrueNASClient(config)
        
        # Simulate large number of snapshots
        large_snapshot_list = []
        for i in range(1000):
            large_snapshot_list.append({
                'snapshot_name': f'auto-snapshot-{i:04d}',
                'dataset': f'tank/k8s/vol{i % 10}',
                'id': f'tank/k8s/vol{i % 10}@auto-snapshot-{i:04d}',
                'properties': {
                    'creation': {'value': str(int(time.time()) - i * 3600)},
                    'used': {'value': str(1024**3)},
                    'referenced': {'value': str(2 * 1024**3)}
                }
            })
        
        with patch.object(client, '_make_request', return_value=large_snapshot_list):
            start_time = time.time()
            snapshots = client.get_snapshots()
            end_time = time.time()
            
            # Should handle large datasets efficiently
            assert len(snapshots) == 1000
            parse_time = end_time - start_time
            assert parse_time < 5.0, f"Parsing took too long: {parse_time}s"
    
    def test_concurrent_request_handling(self):
        """Test that multiple requests can be handled."""
        config = TrueNASConfig(
            host='truenas.example.com',
            username='admin',
            password='test'
        )
        
        client = TrueNASClient(config)
        
        import threading
        import queue
        
        results = queue.Queue()
        
        def make_request():
            try:
                with patch.object(client, '_make_request', return_value=[]):
                    pools = client.get_pools()
                    results.put(('success', len(pools)))
            except Exception as e:
                results.put(('error', str(e)))
        
        # Start multiple threads
        threads = []
        for i in range(5):
            thread = threading.Thread(target=make_request)
            threads.append(thread)
            thread.start()
        
        # Wait for completion
        for thread in threads:
            thread.join(timeout=5)
        
        # Check results
        success_count = 0
        while not results.empty():
            result_type, result_data = results.get()
            if result_type == 'success':
                success_count += 1
        
        # Most or all should succeed
        assert success_count >= 3, f"Only {success_count}/5 requests succeeded"