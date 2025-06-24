"""Unit tests for Prometheus metrics."""

import pytest
from unittest.mock import Mock, patch
from prometheus_client import CollectorRegistry

from truenas_storage_monitor.metrics import TrueNASMetrics


class TestTrueNASMetrics:
    """Test TrueNAS metrics collection."""
    
    @pytest.fixture
    def metrics(self):
        """Create TrueNAS metrics with test registry."""
        registry = CollectorRegistry()
        return TrueNASMetrics(registry=registry)
    
    def test_metrics_initialization(self, metrics):
        """Test metrics initialization."""
        assert metrics.registry is not None
        assert metrics.snapshots_total is not None
        assert metrics.storage_pool_size_bytes is not None
        assert metrics.orphaned_pvs_total is not None
    
    def test_update_snapshot_metrics(self, metrics):
        """Test updating snapshot metrics."""
        snapshot_health = {
            'k8s_snapshots': {
                'total': 5,
                'ready': 4,
                'pending': 1
            },
            'truenas_snapshots': {
                'total': 10,
                'total_size_gb': 50
            },
            'orphaned_resources': {
                'k8s_without_truenas': 2,
                'truenas_without_k8s': 3,
                'total_orphaned': 5
            },
            'analysis': {
                'snapshots_by_age': {
                    'last_24h': 3,
                    'last_week': 4,
                    'older': 3
                }
            }
        }
        
        # Should not raise exception
        metrics.update_snapshot_metrics(snapshot_health)
        
        # Verify some metrics were set
        k8s_total = metrics.snapshots_total.labels(system='kubernetes', pool='', dataset='')
        assert k8s_total._value._value == 5
        
        truenas_total = metrics.snapshots_total.labels(system='truenas', pool='', dataset='')
        assert truenas_total._value._value == 10
    
    def test_update_storage_metrics(self, metrics):
        """Test updating storage metrics."""
        storage_usage = {
            'pools': {
                'tank': {
                    'total': 1000000000000,  # 1TB
                    'used': 500000000000,    # 500GB
                    'free': 500000000000,    # 500GB
                    'healthy': True,
                    'status': 'ONLINE'
                },
                'backup': {
                    'total': 2000000000000,  # 2TB
                    'used': 1000000000000,   # 1TB
                    'free': 1000000000000,   # 1TB
                    'healthy': True,
                    'status': 'ONLINE'
                }
            }
        }
        
        # Should not raise exception
        metrics.update_storage_metrics(storage_usage)
        
        # Verify pool metrics were set
        tank_total = metrics.storage_pool_size_bytes.labels(pool_name='tank', metric_type='total')
        assert tank_total._value._value == 1000000000000
        
        tank_utilization = metrics.storage_pool_utilization_percent.labels(pool_name='tank')
        assert tank_utilization._value._value == 50.0  # 50% utilization
    
    def test_update_volume_metrics(self, metrics):
        """Test updating volume metrics."""
        orphaned_resources = {
            'orphaned_pvs': 3,
            'orphaned_pvcs': 2,
            'orphaned_snapshots': 5
        }
        
        # Should not raise exception
        metrics.update_volume_metrics(orphaned_resources)
        
        # Verify volume metrics were set
        orphaned_pvs = metrics.orphaned_pvs_total.labels(namespace='')
        assert orphaned_pvs._value._value == 3
        
        orphaned_pvcs = metrics.orphaned_pvcs_total.labels(namespace='')
        assert orphaned_pvcs._value._value == 2
    
    def test_update_system_metrics(self, metrics):
        """Test updating system connectivity metrics."""
        validation_result = {
            'k8s_connectivity': True,
            'truenas_connectivity': False,
            'configuration': 'valid'
        }
        
        # Should not raise exception
        metrics.update_system_metrics(validation_result)
        
        # Verify connectivity metrics were set
        k8s_conn = metrics.system_connectivity.labels(system='kubernetes')
        assert k8s_conn._value._value == 1
        
        truenas_conn = metrics.system_connectivity.labels(system='truenas')
        assert truenas_conn._value._value == 0
    
    def test_update_alert_metrics(self, metrics):
        """Test updating alert metrics."""
        alerts = [
            {'level': 'warning', 'category': 'cleanup'},
            {'level': 'warning', 'category': 'cleanup'},
            {'level': 'error', 'category': 'health'},
            {'level': 'warning', 'category': 'storage'}
        ]
        
        # Should not raise exception
        metrics.update_alert_metrics(alerts)
        
        # Verify alert counts
        warning_cleanup = metrics.active_alerts_total.labels(level='warning', category='cleanup')
        assert warning_cleanup._value._value == 2
        
        error_health = metrics.active_alerts_total.labels(level='error', category='health')
        assert error_health._value._value == 1
    
    def test_record_monitoring_run(self, metrics):
        """Test recording monitoring runs."""
        # Should not raise exception
        metrics.record_monitoring_run('success', 2.5, 'full_check')
        metrics.record_monitoring_run('error', 1.0, 'snapshot_check')
        
        # Verify counters were incremented (we can't easily check exact values with prometheus_client)
        # Just ensure no exceptions were raised
    
    def test_metrics_server_start_failure(self, metrics):
        """Test metrics server start failure handling."""
        with patch('truenas_storage_monitor.metrics.start_http_server') as mock_start:
            mock_start.side_effect = Exception("Port already in use")
            
            with pytest.raises(Exception):
                metrics.start_metrics_server(9090)
    
    def test_error_handling_in_updates(self, metrics):
        """Test error handling in metric updates."""
        # Test with invalid data that might cause errors
        with patch.object(metrics.snapshots_total, 'labels') as mock_labels:
            mock_labels.side_effect = Exception("Metric error")
            
            # Should not raise exception, just log error
            metrics.update_snapshot_metrics({})
    
    def test_efficiency_metrics_update(self, metrics):
        """Test updating efficiency metrics."""
        efficiency_analysis = {
            'overall_efficiency': {
                'thin_provisioning_ratio': 2.5,
                'compression_ratio': 1.8,
                'snapshot_overhead_percent': 15.0
            },
            'pool_efficiency': [
                {
                    'name': 'tank',
                    'utilization_percent': 75.0
                }
            ]
        }
        
        # Should not raise exception
        metrics.update_efficiency_metrics(efficiency_analysis)
        
        # Verify efficiency metrics were set
        thin_ratio = metrics.thin_provisioning_ratio.labels(pool_name='overall')
        assert thin_ratio._value._value == 2.5
        
        compression = metrics.compression_ratio.labels(pool_name='overall')
        assert compression._value._value == 1.8
    
    def test_csi_metrics_update(self, metrics):
        """Test updating CSI driver metrics."""
        csi_health = {
            'total_pods': 6,
            'running_pods': 5,
            'unhealthy_pods': 1
        }
        
        # Should not raise exception
        metrics.update_csi_metrics(csi_health)
        
        # Verify CSI metrics were set
        running_pods = metrics.csi_driver_pods_total.labels(
            driver_name='democratic-csi', status='running'
        )
        assert running_pods._value._value == 5
        
        unhealthy_pods = metrics.csi_driver_pods_total.labels(
            driver_name='democratic-csi', status='unhealthy'
        )
        assert unhealthy_pods._value._value == 1


class TestMetricsIntegration:
    """Test metrics integration with Monitor."""
    
    def test_monitor_with_metrics_disabled(self):
        """Test Monitor with metrics disabled."""
        from truenas_storage_monitor.monitor import Monitor
        
        config = {
            'openshift': {'namespace': 'test'},
            'monitoring': {'interval': 60},
            'metrics': {'enabled': False}
        }
        
        with patch('truenas_storage_monitor.monitor.TrueNASMetrics') as mock_metrics:
            monitor = Monitor(config)
            
            # Metrics should be initialized but server not started
            mock_metrics.assert_called_once()
            assert not mock_metrics.return_value.start_metrics_server.called
    
    def test_monitor_with_custom_metrics_port(self):
        """Test Monitor with custom metrics port."""
        from truenas_storage_monitor.monitor import Monitor
        
        config = {
            'openshift': {'namespace': 'test'},
            'monitoring': {'interval': 60},
            'metrics': {'enabled': True, 'port': 9091}
        }
        
        with patch('truenas_storage_monitor.monitor.TrueNASMetrics') as mock_metrics:
            monitor = Monitor(config)
            
            # Metrics server should be started with custom port
            mock_metrics.return_value.start_metrics_server.assert_called_with(9091)
    
    def test_monitor_metrics_collection(self):
        """Test that Monitor methods update metrics."""
        from truenas_storage_monitor.monitor import Monitor
        
        config = {
            'openshift': {'namespace': 'test'},
            'monitoring': {'interval': 60},
            'metrics': {'enabled': False}  # Disable server for testing
        }
        
        monitor = Monitor(config)
        
        # Create a real mock metrics object
        mock_metrics = Mock()
        monitor.metrics = mock_metrics
        
        # Mock the monitor to avoid actual connections
        monitor.k8s_client = None
        monitor.truenas_client = None
        
        # Call method that should update metrics
        result = monitor.check_orphaned_resources()
        
        # Verify metrics were updated
        mock_metrics.update_volume_metrics.assert_called_once_with(result)