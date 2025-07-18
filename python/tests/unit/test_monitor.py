"""Unit tests for the Monitor class."""

import pytest
from unittest.mock import Mock, patch
from datetime import datetime, timedelta

from truenas_storage_monitor.monitor import Monitor
from truenas_storage_monitor.config import Config
from truenas_storage_monitor.exceptions import TrueNASMonitorError


class TestMonitor:
    """Test cases for the Monitor class."""

    @pytest.fixture
    def mock_config(self):
        """Create a mock configuration."""
        config = Mock(spec=Config)
        config.kubernetes = {'namespace': 'democratic-csi'}
        config.truenas = {'url': 'https://truenas.test'}
        return config

    @pytest.fixture
    def monitor(self, mock_config):
        """Create a Monitor instance with mocked dependencies."""
        with patch('truenas_storage_monitor.monitor.K8sClient'), \
             patch('truenas_storage_monitor.monitor.TrueNASClient'):
            return Monitor(mock_config)

    def test_monitor_initialization(self, mock_config):
        """Test that Monitor initializes correctly."""
        with patch('truenas_storage_monitor.monitor.K8sClient') as mock_k8s, \
             patch('truenas_storage_monitor.monitor.TrueNASClient') as mock_truenas:
            
            monitor = Monitor(mock_config)
            
            assert monitor.config == mock_config
            mock_k8s.assert_called_once_with(mock_config.kubernetes)
            mock_truenas.assert_called_once_with(mock_config.truenas)

    def test_find_orphaned_resources_success(self, monitor):
        """Test successful orphaned resource detection."""
        # Mock data
        mock_pvs = [
            {
                'metadata': {
                    'name': 'pv-test',
                    'creationTimestamp': (datetime.now() - timedelta(hours=25)).isoformat() + 'Z'
                },
                'spec': {
                    'csi': {'driver': 'democratic-csi'},
                    'capacity': {'storage': '10Gi'},
                    'storageClassName': 'truenas-iscsi'
                }
            }
        ]
        
        mock_pvcs = [
            {
                'metadata': {
                    'name': 'pvc-test',
                    'namespace': 'default',
                    'creationTimestamp': (datetime.now() - timedelta(hours=25)).isoformat() + 'Z'
                },
                'status': {'phase': 'Pending'},
                'spec': {
                    'resources': {'requests': {'storage': '5Gi'}},
                    'storageClassName': 'truenas-iscsi'
                }
            }
        ]
        
        mock_snapshots = []
        mock_truenas_volumes = []
        mock_truenas_snapshots = []

        # Setup mocks
        monitor.k8s_client.list_persistent_volumes.return_value = mock_pvs
        monitor.k8s_client.list_persistent_volume_claims.return_value = mock_pvcs
        monitor.k8s_client.list_volume_snapshots.return_value = mock_snapshots
        monitor.truenas_client.list_volumes.return_value = mock_truenas_volumes
        monitor.truenas_client.list_snapshots.return_value = mock_truenas_snapshots

        # Execute
        result = monitor.find_orphaned_resources()

        # Verify
        assert 'timestamp' in result
        assert 'orphaned_pvs' in result
        assert 'orphaned_pvcs' in result
        assert 'orphaned_snapshots' in result
        assert result['total_pvs'] == 1
        assert result['total_pvcs'] == 1
        assert result['total_snapshots'] == 0

        # Should find orphaned PV (no corresponding TrueNAS volume)
        assert len(result['orphaned_pvs']) == 1
        assert result['orphaned_pvs'][0]['name'] == 'pv-test'

        # Should find orphaned PVC (pending for > 24h)
        assert len(result['orphaned_pvcs']) == 1
        assert result['orphaned_pvcs'][0]['name'] == 'pvc-test'

    def test_find_orphaned_resources_error_handling(self, monitor):
        """Test error handling in orphaned resource detection."""
        # Setup mock to raise exception
        monitor.k8s_client.list_persistent_volumes.side_effect = Exception("K8s API error")

        # Execute and verify exception
        with pytest.raises(TrueNASMonitorError) as exc_info:
            monitor.find_orphaned_resources()
        
        assert "Failed to scan for orphaned resources" in str(exc_info.value)

    def test_is_democratic_csi_pv(self, monitor):
        """Test democratic-csi PV detection."""
        # Test positive case
        pv_democratic = {
            'spec': {
                'csi': {'driver': 'org.democratic-csi.iscsi'}
            }
        }
        assert monitor._is_democratic_csi_pv(pv_democratic) is True

        # Test TrueNAS driver
        pv_truenas = {
            'spec': {
                'csi': {'driver': 'truenas.csi.driver'}
            }
        }
        assert monitor._is_democratic_csi_pv(pv_truenas) is True

        # Test negative case
        pv_other = {
            'spec': {
                'csi': {'driver': 'other.csi.driver'}
            }
        }
        assert monitor._is_democratic_csi_pv(pv_other) is False

        # Test missing CSI info
        pv_no_csi = {'spec': {}}
        assert monitor._is_democratic_csi_pv(pv_no_csi) is False

    def test_parse_storage_size(self, monitor):
        """Test storage size parsing."""
        # Test various formats
        assert monitor._parse_storage_size('1Gi') == 1024**3
        assert monitor._parse_storage_size('5G') == 5 * 1024**3
        assert monitor._parse_storage_size('100Mi') == 100 * 1024**2
        assert monitor._parse_storage_size('1Ti') == 1024**4
        assert monitor._parse_storage_size('1024') == 1024
        assert monitor._parse_storage_size('') == 0
        assert monitor._parse_storage_size('invalid') == 0

    def test_analyze_storage_usage(self, monitor):
        """Test storage usage analysis."""
        # Mock data
        mock_pvcs = [
            {
                'spec': {
                    'resources': {'requests': {'storage': '10Gi'}}
                }
            },
            {
                'spec': {
                    'resources': {'requests': {'storage': '5Gi'}}
                }
            }
        ]
        
        mock_pvs = [{'metadata': {'name': 'pv1'}}, {'metadata': {'name': 'pv2'}}]
        mock_truenas_volumes = [
            {'used_bytes': 5 * 1024**3},  # 5GB used
            {'used_bytes': 3 * 1024**3}   # 3GB used
        ]

        # Setup mocks
        monitor.k8s_client.list_persistent_volume_claims.return_value = mock_pvcs
        monitor.k8s_client.list_persistent_volumes.return_value = mock_pvs
        monitor.truenas_client.list_volumes.return_value = mock_truenas_volumes

        # Execute
        result = monitor.analyze_storage_usage()

        # Verify
        assert result['total_allocated_gb'] == 15.0  # 10Gi + 5Gi
        assert result['total_used_gb'] == 8.0        # 5GB + 3GB
        assert result['total_pvcs'] == 2
        assert result['total_pvs'] == 2
        assert 'thin_provisioning_efficiency' in result
        assert 'recommendations' in result

    def test_check_health(self, monitor):
        """Test health check functionality."""
        # Setup mocks for healthy state
        monitor.k8s_client.test_connection.return_value = None
        monitor.truenas_client.test_connection.return_value = None
        monitor.k8s_client.list_pods.return_value = [
            {'status': {'phase': 'Running'}},
            {'status': {'phase': 'Running'}}
        ]

        # Execute
        result = monitor.check_health()

        # Verify
        assert result['healthy'] is True
        assert 'components' in result
        assert result['components']['kubernetes']['healthy'] is True
        assert result['components']['truenas']['healthy'] is True
        assert result['components']['csi_driver']['healthy'] is True

    def test_check_health_with_failures(self, monitor):
        """Test health check with component failures."""
        # Setup mocks for unhealthy state
        monitor.k8s_client.test_connection.side_effect = Exception("Connection failed")
        monitor.truenas_client.test_connection.return_value = None
        monitor.k8s_client.list_pods.return_value = []

        # Execute
        result = monitor.check_health()

        # Verify
        assert result['healthy'] is False
        assert result['components']['kubernetes']['healthy'] is False
        assert result['components']['truenas']['healthy'] is True
        assert result['components']['csi_driver']['healthy'] is False

    def test_generate_recommendations(self, monitor):
        """Test recommendation generation."""
        # Mock data with large PVC
        mock_pvcs = [
            {
                'metadata': {'name': 'large-pvc'},
                'spec': {
                    'resources': {'requests': {'storage': '200Gi'}}
                }
            },
            {
                'metadata': {'name': 'normal-pvc'},
                'spec': {
                    'resources': {'requests': {'storage': '10Gi'}}
                }
            }
        ]
        
        mock_truenas_volumes = [
            {'name': 'vol1'},
            {'name': 'vol2'},
            {'name': 'vol3'}  # More volumes than PVCs
        ]

        # Execute
        recommendations = monitor._generate_recommendations(mock_pvcs, mock_truenas_volumes)

        # Verify
        assert len(recommendations) >= 1
        assert any('large-pvc' in rec for rec in recommendations)
        assert any('unused TrueNAS volumes' in rec for rec in recommendations)