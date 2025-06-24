"""Prometheus metrics for TrueNAS Storage Monitor."""

import logging
from typing import Dict, Any, Optional
from prometheus_client import Counter, Gauge, Histogram, Info, start_http_server, CollectorRegistry, REGISTRY
import time

logger = logging.getLogger(__name__)


class TrueNASMetrics:
    """Prometheus metrics collector for TrueNAS Storage Monitor."""
    
    def __init__(self, registry: Optional[CollectorRegistry] = None):
        """Initialize metrics collector.
        
        Args:
            registry: Prometheus registry to use. If None, uses default registry.
        """
        self.registry = registry or REGISTRY
        
        # Snapshot metrics
        self.snapshots_total = Gauge(
            'truenas_snapshots_total',
            'Total number of snapshots',
            ['system', 'pool', 'dataset'],
            registry=self.registry
        )
        
        self.snapshots_size_bytes = Gauge(
            'truenas_snapshots_size_bytes',
            'Total size of snapshots in bytes',
            ['system', 'pool', 'dataset'],
            registry=self.registry
        )
        
        self.snapshots_age_days = Gauge(
            'truenas_snapshots_age_days',
            'Age of snapshots in days',
            ['system', 'pool', 'dataset', 'snapshot_name'],
            registry=self.registry
        )
        
        self.orphaned_snapshots_total = Gauge(
            'truenas_orphaned_snapshots_total',
            'Total number of orphaned snapshots',
            ['system', 'type'],  # type: k8s_without_truenas, truenas_without_k8s
            registry=self.registry
        )
        
        # Storage metrics
        self.storage_pool_size_bytes = Gauge(
            'truenas_storage_pool_size_bytes',
            'Total storage pool size in bytes',
            ['pool_name', 'metric_type'],  # metric_type: total, used, free
            registry=self.registry
        )
        
        self.storage_pool_utilization_percent = Gauge(
            'truenas_storage_pool_utilization_percent',
            'Storage pool utilization percentage',
            ['pool_name'],
            registry=self.registry
        )
        
        self.storage_pool_health = Gauge(
            'truenas_storage_pool_health',
            'Storage pool health status (1=healthy, 0=unhealthy)',
            ['pool_name', 'status'],
            registry=self.registry
        )
        
        # Volume metrics
        self.persistent_volumes_total = Gauge(
            'truenas_persistent_volumes_total',
            'Total number of persistent volumes',
            ['namespace', 'storage_class', 'status'],
            registry=self.registry
        )
        
        self.persistent_volume_claims_total = Gauge(
            'truenas_persistent_volume_claims_total',
            'Total number of persistent volume claims',
            ['namespace', 'storage_class', 'status'],
            registry=self.registry
        )
        
        self.orphaned_pvs_total = Gauge(
            'truenas_orphaned_pvs_total',
            'Total number of orphaned persistent volumes',
            ['namespace'],
            registry=self.registry
        )
        
        self.orphaned_pvcs_total = Gauge(
            'truenas_orphaned_pvcs_total',
            'Total number of orphaned persistent volume claims',
            ['namespace'],
            registry=self.registry
        )
        
        # Efficiency metrics
        self.thin_provisioning_ratio = Gauge(
            'truenas_thin_provisioning_ratio',
            'Thin provisioning ratio (allocated/used)',
            ['pool_name'],
            registry=self.registry
        )
        
        self.compression_ratio = Gauge(
            'truenas_compression_ratio',
            'Compression ratio',
            ['pool_name'],
            registry=self.registry
        )
        
        self.snapshot_overhead_percent = Gauge(
            'truenas_snapshot_overhead_percent',
            'Snapshot overhead as percentage of total storage',
            ['pool_name'],
            registry=self.registry
        )
        
        # System health metrics
        self.system_connectivity = Gauge(
            'truenas_system_connectivity',
            'System connectivity status (1=connected, 0=disconnected)',
            ['system'],  # system: kubernetes, truenas
            registry=self.registry
        )
        
        self.csi_driver_pods_total = Gauge(
            'truenas_csi_driver_pods_total',
            'Total number of CSI driver pods',
            ['driver_name', 'status'],  # status: running, pending, failed
            registry=self.registry
        )
        
        # Operation metrics
        self.monitoring_runs_total = Counter(
            'truenas_monitoring_runs_total',
            'Total number of monitoring runs',
            ['status'],  # status: success, error
            registry=self.registry
        )
        
        self.monitoring_duration_seconds = Histogram(
            'truenas_monitoring_duration_seconds',
            'Duration of monitoring runs in seconds',
            ['operation'],  # operation: full_check, snapshot_check, orphan_check
            registry=self.registry
        )
        
        # Alert metrics
        self.active_alerts_total = Gauge(
            'truenas_active_alerts_total',
            'Total number of active alerts',
            ['level', 'category'],  # level: warning, error, critical; category: cleanup, storage, health
            registry=self.registry
        )
        
        # System info
        self.info = Info(
            'truenas_monitor_info',
            'Information about TrueNAS Storage Monitor',
            registry=self.registry
        )
        
        # Set initial info
        self.info.info({
            'version': '1.0.0',
            'component': 'truenas-storage-monitor'
        })
    
    def update_snapshot_metrics(self, snapshot_health: Dict[str, Any]) -> None:
        """Update snapshot-related metrics.
        
        Args:
            snapshot_health: Snapshot health data from monitor.check_snapshot_health()
        """
        try:
            # K8s snapshot metrics
            k8s_snapshots = snapshot_health.get('k8s_snapshots', {})
            self.snapshots_total.labels(system='kubernetes', pool='', dataset='').set(
                k8s_snapshots.get('total', 0)
            )
            
            # TrueNAS snapshot metrics
            truenas_snapshots = snapshot_health.get('truenas_snapshots', {})
            self.snapshots_total.labels(system='truenas', pool='', dataset='').set(
                truenas_snapshots.get('total', 0)
            )
            
            self.snapshots_size_bytes.labels(system='truenas', pool='', dataset='').set(
                truenas_snapshots.get('total_size_gb', 0) * (1024**3)
            )
            
            # Orphaned snapshot metrics
            orphaned = snapshot_health.get('orphaned_resources', {})
            self.orphaned_snapshots_total.labels(
                system='kubernetes', 
                type='k8s_without_truenas'
            ).set(orphaned.get('k8s_without_truenas', 0))
            
            self.orphaned_snapshots_total.labels(
                system='truenas', 
                type='truenas_without_k8s'
            ).set(orphaned.get('truenas_without_k8s', 0))
            
            # Snapshot age metrics (from analysis)
            analysis = snapshot_health.get('analysis', {})
            snapshots_by_age = analysis.get('snapshots_by_age', {})
            
            for age_category, count in snapshots_by_age.items():
                # Convert age category to approximate days
                days = 0
                if age_category == 'last_24h':
                    days = 1
                elif age_category == 'last_week':
                    days = 7
                elif age_category == 'older':
                    days = 30  # Approximate for older snapshots
                
                self.snapshots_age_days.labels(
                    system='truenas',
                    pool='',
                    dataset='',
                    snapshot_name=age_category
                ).set(days)
            
            logger.debug(f"Updated snapshot metrics: K8s={k8s_snapshots.get('total', 0)}, "
                        f"TrueNAS={truenas_snapshots.get('total', 0)}, "
                        f"Orphaned={orphaned.get('total_orphaned', 0)}")
                        
        except Exception as e:
            logger.error(f"Error updating snapshot metrics: {e}")
    
    def update_storage_metrics(self, storage_usage: Dict[str, Any]) -> None:
        """Update storage-related metrics.
        
        Args:
            storage_usage: Storage usage data from monitor.check_storage_usage()
        """
        try:
            pools = storage_usage.get('pools', {})
            
            for pool_name, pool_data in pools.items():
                # Pool size metrics
                self.storage_pool_size_bytes.labels(
                    pool_name=pool_name, metric_type='total'
                ).set(pool_data.get('total', 0))
                
                self.storage_pool_size_bytes.labels(
                    pool_name=pool_name, metric_type='used'
                ).set(pool_data.get('used', 0))
                
                self.storage_pool_size_bytes.labels(
                    pool_name=pool_name, metric_type='free'
                ).set(pool_data.get('free', 0))
                
                # Pool utilization
                total = pool_data.get('total', 1)  # Avoid division by zero
                used = pool_data.get('used', 0)
                utilization = (used / total) * 100 if total > 0 else 0
                
                self.storage_pool_utilization_percent.labels(
                    pool_name=pool_name
                ).set(utilization)
                
                # Pool health
                healthy = 1 if pool_data.get('healthy', False) else 0
                status = pool_data.get('status', 'unknown')
                
                self.storage_pool_health.labels(
                    pool_name=pool_name, status=status
                ).set(healthy)
            
            logger.debug(f"Updated storage metrics for {len(pools)} pools")
            
        except Exception as e:
            logger.error(f"Error updating storage metrics: {e}")
    
    def update_volume_metrics(self, orphaned_resources: Dict[str, int]) -> None:
        """Update volume-related metrics.
        
        Args:
            orphaned_resources: Orphaned resource counts from monitor.check_orphaned_resources()
        """
        try:
            # Orphaned volumes
            self.orphaned_pvs_total.labels(namespace='').set(
                orphaned_resources.get('orphaned_pvs', 0)
            )
            
            self.orphaned_pvcs_total.labels(namespace='').set(
                orphaned_resources.get('orphaned_pvcs', 0)
            )
            
            logger.debug(f"Updated volume metrics: PVs={orphaned_resources.get('orphaned_pvs', 0)}, "
                        f"PVCs={orphaned_resources.get('orphaned_pvcs', 0)}")
            
        except Exception as e:
            logger.error(f"Error updating volume metrics: {e}")
    
    def update_efficiency_metrics(self, efficiency_analysis: Dict[str, Any]) -> None:
        """Update storage efficiency metrics.
        
        Args:
            efficiency_analysis: Efficiency data from monitor.analyze_storage_efficiency()
        """
        try:
            overall = efficiency_analysis.get('overall_efficiency', {})
            
            # Overall efficiency metrics
            self.thin_provisioning_ratio.labels(pool_name='overall').set(
                overall.get('thin_provisioning_ratio', 0.0)
            )
            
            self.compression_ratio.labels(pool_name='overall').set(
                overall.get('compression_ratio', 1.0)
            )
            
            self.snapshot_overhead_percent.labels(pool_name='overall').set(
                overall.get('snapshot_overhead_percent', 0.0)
            )
            
            # Per-pool efficiency metrics
            for pool_data in efficiency_analysis.get('pool_efficiency', []):
                pool_name = pool_data.get('name', 'unknown')
                
                # Note: Individual pool ratios would need to be calculated
                # if we have per-pool data available
            
            logger.debug("Updated efficiency metrics")
            
        except Exception as e:
            logger.error(f"Error updating efficiency metrics: {e}")
    
    def update_system_metrics(self, validation_result: Dict[str, Any]) -> None:
        """Update system health metrics.
        
        Args:
            validation_result: Validation data from monitor.validate_configuration()
        """
        try:
            # Connectivity metrics
            k8s_connected = 1 if validation_result.get('k8s_connectivity', False) else 0
            truenas_connected = 1 if validation_result.get('truenas_connectivity', False) else 0
            
            self.system_connectivity.labels(system='kubernetes').set(k8s_connected)
            self.system_connectivity.labels(system='truenas').set(truenas_connected)
            
            logger.debug(f"Updated system metrics: K8s={k8s_connected}, TrueNAS={truenas_connected}")
            
        except Exception as e:
            logger.error(f"Error updating system metrics: {e}")
    
    def update_csi_metrics(self, csi_health: Dict[str, Any]) -> None:
        """Update CSI driver metrics.
        
        Args:
            csi_health: CSI health data from monitor.check_csi_health()
        """
        try:
            total_pods = csi_health.get('total_pods', 0)
            running_pods = csi_health.get('running_pods', 0)
            unhealthy_pods = total_pods - running_pods
            
            self.csi_driver_pods_total.labels(
                driver_name='democratic-csi', status='running'
            ).set(running_pods)
            
            self.csi_driver_pods_total.labels(
                driver_name='democratic-csi', status='unhealthy'
            ).set(unhealthy_pods)
            
            logger.debug(f"Updated CSI metrics: {running_pods}/{total_pods} pods running")
            
        except Exception as e:
            logger.error(f"Error updating CSI metrics: {e}")
    
    def update_alert_metrics(self, alerts: list) -> None:
        """Update alert metrics.
        
        Args:
            alerts: List of alert dictionaries with level and category
        """
        try:
            # Reset all alert counters
            self.active_alerts_total.clear()
            
            # Count alerts by level and category
            alert_counts = {}
            for alert in alerts:
                level = alert.get('level', 'unknown')
                category = alert.get('category', 'unknown')
                key = (level, category)
                alert_counts[key] = alert_counts.get(key, 0) + 1
            
            # Set the counts
            for (level, category), count in alert_counts.items():
                self.active_alerts_total.labels(level=level, category=category).set(count)
            
            logger.debug(f"Updated alert metrics: {len(alerts)} total alerts")
            
        except Exception as e:
            logger.error(f"Error updating alert metrics: {e}")
    
    def record_monitoring_run(self, status: str, duration: float, operation: str = 'full_check') -> None:
        """Record a monitoring run.
        
        Args:
            status: 'success' or 'error'
            duration: Duration in seconds
            operation: Type of operation performed
        """
        try:
            self.monitoring_runs_total.labels(status=status).inc()
            self.monitoring_duration_seconds.labels(operation=operation).observe(duration)
            
            logger.debug(f"Recorded monitoring run: {operation} {status} in {duration:.2f}s")
            
        except Exception as e:
            logger.error(f"Error recording monitoring run: {e}")
    
    def start_metrics_server(self, port: int = 9090) -> None:
        """Start the Prometheus metrics HTTP server.
        
        Args:
            port: Port to listen on
        """
        try:
            start_http_server(port, registry=self.registry)
            logger.info(f"Prometheus metrics server started on port {port}")
        except Exception as e:
            logger.error(f"Failed to start metrics server: {e}")
            raise


def timed_operation(metrics: TrueNASMetrics, operation: str):
    """Decorator to time operations and record metrics.
    
    Args:
        metrics: TrueNASMetrics instance
        operation: Operation name for metrics
    """
    def decorator(func):
        def wrapper(*args, **kwargs):
            start_time = time.time()
            try:
                result = func(*args, **kwargs)
                duration = time.time() - start_time
                metrics.record_monitoring_run('success', duration, operation)
                return result
            except Exception as e:
                duration = time.time() - start_time
                metrics.record_monitoring_run('error', duration, operation)
                raise
        return wrapper
    return decorator