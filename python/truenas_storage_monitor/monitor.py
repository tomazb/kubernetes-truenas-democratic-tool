"""Main monitoring module for TrueNAS Storage Monitor."""

import logging
from typing import Dict, Any, Optional

from .k8s_client import K8sClient, K8sConfig
from .truenas_client import TrueNASClient, TrueNASConfig

logger = logging.getLogger(__name__)


class Monitor:
    """Main monitoring class that coordinates all monitoring activities."""
    
    def __init__(self, config: Dict[str, Any]) -> None:
        """Initialize the monitor with configuration.
        
        Args:
            config: Configuration dictionary
        """
        self.config = config
        self.k8s_client: Optional[K8sClient] = None
        self.truenas_client: Optional[TrueNASClient] = None
        
        # Initialize K8s client
        try:
            k8s_config = K8sConfig(
                namespace=config.get("openshift", {}).get("namespace"),
                storage_class=config.get("openshift", {}).get("storage_class"),
                csi_driver=config.get("openshift", {}).get("csi_driver", "org.democratic-csi.nfs")
            )
            self.k8s_client = K8sClient(k8s_config)
            logger.info("Kubernetes client initialized")
        except Exception as e:
            logger.error(f"Failed to initialize Kubernetes client: {e}")
        
        # Initialize TrueNAS client if configured
        if "truenas" in config:
            try:
                import urllib.parse
                url = config["truenas"]["url"]
                parsed = urllib.parse.urlparse(url)
                host = parsed.hostname
                port = parsed.port or (443 if parsed.scheme == "https" else 80)
                
                truenas_config = TrueNASConfig(
                    host=host,
                    port=port,
                    username=config["truenas"].get("username"),
                    password=config["truenas"].get("password"),
                    api_key=config["truenas"].get("api_key"),
                    verify_ssl=config["truenas"].get("verify_ssl", True)
                )
                self.truenas_client = TrueNASClient(truenas_config)
                logger.info("TrueNAS client initialized")
            except Exception as e:
                logger.error(f"Failed to initialize TrueNAS client: {e}")
    
    def check_orphaned_resources(self) -> Dict[str, int]:
        """Check for orphaned resources.
        
        Returns:
            Dictionary with counts of orphaned resources
        """
        result = {
            "orphaned_pvs": 0,
            "orphaned_pvcs": 0,
            "orphaned_snapshots": 0,
        }
        
        if not self.k8s_client:
            logger.warning("Kubernetes client not available for orphan check")
            return result
        
        try:
            # Check for orphaned PVs
            orphaned_pvs = self.k8s_client.find_orphaned_pvs()
            result["orphaned_pvs"] = len(orphaned_pvs)
            
            # Check for orphaned PVCs
            orphaned_pvcs = self.k8s_client.find_orphaned_pvcs()
            result["orphaned_pvcs"] = len(orphaned_pvcs)
            
            # Check for orphaned snapshots
            if self.truenas_client:
                # Get snapshots from both systems for comparison
                k8s_snapshots = self.k8s_client.get_volume_snapshots()
                truenas_snapshots = self.truenas_client.get_snapshots()
                
                # Find orphaned snapshots in both directions
                orphaned_k8s_snapshots = self.k8s_client.find_orphaned_snapshots(truenas_snapshots)
                orphaned_truenas_snapshots = self.truenas_client.find_orphaned_truenas_snapshots(k8s_snapshots)
                
                result["orphaned_snapshots"] = len(orphaned_k8s_snapshots) + len(orphaned_truenas_snapshots)
                
                logger.info(f"Found {len(orphaned_k8s_snapshots)} orphaned K8s snapshots and {len(orphaned_truenas_snapshots)} orphaned TrueNAS snapshots")
            
            logger.info(f"Found {result['orphaned_pvs']} orphaned PVs, {result['orphaned_pvcs']} orphaned PVCs, and {result['orphaned_snapshots']} orphaned snapshots")
            
        except Exception as e:
            logger.error(f"Error checking orphaned resources: {e}")
            
        return result
    
    def check_storage_usage(self) -> Dict[str, Any]:
        """Check storage usage across pools.
        
        Returns:
            Dictionary with storage usage information
        """
        result = {"pools": {}}
        
        if not self.truenas_client:
            logger.warning("TrueNAS client not available for storage usage check")
            return result
        
        try:
            pools = self.truenas_client.get_pools()
            
            for pool in pools:
                result["pools"][pool.name] = {
                    "total": pool.total_size,
                    "used": pool.used_size,
                    "free": pool.free_size,
                    "healthy": pool.healthy,
                    "status": pool.status,
                    "fragmentation": pool.fragmentation,
                }
                
            logger.info(f"Retrieved usage for {len(pools)} storage pools")
            
        except Exception as e:
            logger.error(f"Error checking storage usage: {e}")
            
        return result
    
    def check_csi_health(self) -> Dict[str, Any]:
        """Check CSI driver health.
        
        Returns:
            Dictionary with CSI driver health information
        """
        result = {
            "healthy": False,
            "total_pods": 0,
            "running_pods": 0,
            "unhealthy_pods": [],
        }
        
        if not self.k8s_client:
            logger.warning("Kubernetes client not available for CSI health check")
            return result
        
        try:
            health = self.k8s_client.check_csi_driver_health()
            result.update(health)
            
            logger.info(f"CSI driver health: {health['running_pods']}/{health['total_pods']} pods running")
            
        except Exception as e:
            logger.error(f"Error checking CSI driver health: {e}")
            
        return result
    
    def check_snapshot_health(self) -> Dict[str, Any]:
        """Check snapshot health and status across both systems.
        
        Returns:
            Dictionary with comprehensive snapshot health information
        """
        result = {
            "k8s_snapshots": {
                "total": 0,
                "ready": 0,
                "pending": 0,
                "error": 0,
                "stale": 0
            },
            "truenas_snapshots": {
                "total": 0,
                "recent": 0,  # Last 24 hours
                "old": 0,     # Older than 30 days
                "large": 0,   # Larger than 1GB
                "total_size_gb": 0
            },
            "orphaned_resources": {
                "k8s_without_truenas": 0,
                "truenas_without_k8s": 0,
                "total_orphaned": 0
            },
            "analysis": {},
            "recommendations": [],
            "alerts": []
        }
        
        if not self.k8s_client:
            logger.warning("Kubernetes client not available for snapshot health check")
            return result
        
        try:
            # Get K8s snapshot status
            k8s_snapshots = self.k8s_client.get_volume_snapshots()
            result["k8s_snapshots"]["total"] = len(k8s_snapshots)
            
            for snap in k8s_snapshots:
                if snap.ready_to_use:
                    result["k8s_snapshots"]["ready"] += 1
                else:
                    result["k8s_snapshots"]["pending"] += 1
            
            # Find stale K8s snapshots
            stale_snapshots = self.k8s_client.find_stale_snapshots(age_threshold_days=30)
            result["k8s_snapshots"]["stale"] = len(stale_snapshots)
            
            # Get TrueNAS snapshot analysis if available
            if self.truenas_client:
                analysis = self.truenas_client.analyze_snapshot_usage()
                result["truenas_snapshots"]["total"] = analysis.get("total_snapshots", 0)
                result["truenas_snapshots"]["total_size_gb"] = analysis.get("total_snapshot_size", 0) / (1024**3)
                result["truenas_snapshots"]["recent"] = analysis.get("snapshots_by_age", {}).get("last_24h", 0)
                result["truenas_snapshots"]["old"] = analysis.get("snapshots_by_age", {}).get("older", 0)
                result["truenas_snapshots"]["large"] = len(analysis.get("large_snapshots", []))
                
                result["analysis"] = analysis
                result["recommendations"].extend(analysis.get("recommendations", []))
                
                # Find orphaned snapshots
                orphaned_k8s = self.k8s_client.find_orphaned_snapshots()
                orphaned_truenas = self.truenas_client.find_orphaned_truenas_snapshots(k8s_snapshots)
                
                result["orphaned_resources"]["k8s_without_truenas"] = len(orphaned_k8s)
                result["orphaned_resources"]["truenas_without_k8s"] = len(orphaned_truenas)
                result["orphaned_resources"]["total_orphaned"] = len(orphaned_k8s) + len(orphaned_truenas)
                
                # Generate alerts based on thresholds
                if result["orphaned_resources"]["total_orphaned"] > 5:
                    result["alerts"].append({
                        "level": "warning",
                        "category": "cleanup",
                        "message": f"Found {result['orphaned_resources']['total_orphaned']} orphaned snapshots that may need cleanup"
                    })
                
                if result["truenas_snapshots"]["total_size_gb"] > 100:
                    result["alerts"].append({
                        "level": "warning", 
                        "category": "storage",
                        "message": f"Snapshot storage usage is {result['truenas_snapshots']['total_size_gb']:.1f}GB - consider retention policy review"
                    })
                
                if result["k8s_snapshots"]["pending"] > result["k8s_snapshots"]["ready"] * 0.1:  # More than 10% pending
                    result["alerts"].append({
                        "level": "error",
                        "category": "health",
                        "message": f"{result['k8s_snapshots']['pending']} snapshots are stuck in pending state"
                    })
            
            logger.info(f"Snapshot health check: K8s={result['k8s_snapshots']['total']}, TrueNAS={result['truenas_snapshots']['total']}, Orphaned={result['orphaned_resources']['total_orphaned']}")
            
        except Exception as e:
            logger.error(f"Error checking snapshot health: {e}")
            result["alerts"].append({
                "level": "error",
                "category": "system",
                "message": f"Failed to check snapshot health: {str(e)}"
            })
        
        return result
    
    def analyze_storage_efficiency(self) -> Dict[str, Any]:
        """Analyze storage efficiency across the entire system.
        
        Returns:
            Dictionary with storage efficiency analysis
        """
        result = {
            "overall_efficiency": {
                "thin_provisioning_ratio": 0.0,
                "compression_ratio": 1.0,
                "deduplication_ratio": 1.0,
                "snapshot_overhead_percent": 0.0
            },
            "volume_efficiency": [],
            "pool_efficiency": [],
            "recommendations": [],
            "potential_savings_gb": 0.0
        }
        
        if not self.truenas_client:
            logger.warning("TrueNAS client not available for efficiency analysis")
            return result
        
        try:
            # Get pool information for efficiency calculations
            pools = self.truenas_client.get_pools()
            total_allocated = 0
            total_used = 0
            
            for pool in pools:
                pool_efficiency = {
                    "name": pool.name,
                    "capacity_gb": pool.total_size / (1024**3),
                    "used_gb": pool.used_size / (1024**3),
                    "utilization_percent": (pool.used_size / pool.total_size) * 100 if pool.total_size > 0 else 0,
                    "fragmentation": pool.fragmentation,
                    "health": pool.healthy
                }
                result["pool_efficiency"].append(pool_efficiency)
                
                total_allocated += pool.total_size
                total_used += pool.used_size
            
            # Calculate overall thin provisioning ratio
            if self.k8s_client:
                pvs = self.k8s_client.get_persistent_volumes()
                k8s_allocated = sum(
                    int(pv.capacity.rstrip('Gi')) * (1024**3) 
                    for pv in pvs 
                    if pv.capacity.endswith('Gi')
                )
                
                if total_used > 0:
                    result["overall_efficiency"]["thin_provisioning_ratio"] = k8s_allocated / total_used
            
            # Analyze snapshot overhead
            snapshot_analysis = self.truenas_client.analyze_snapshot_usage()
            snapshot_size = snapshot_analysis.get("total_snapshot_size", 0)
            
            if total_used > 0:
                result["overall_efficiency"]["snapshot_overhead_percent"] = (snapshot_size / total_used) * 100
            
            # Generate efficiency recommendations
            if result["overall_efficiency"]["thin_provisioning_ratio"] > 2.0:
                result["recommendations"].append(
                    "High thin provisioning ratio detected - monitor for potential overcommitment"
                )
            
            if result["overall_efficiency"]["snapshot_overhead_percent"] > 20:
                result["recommendations"].append(
                    f"Snapshot overhead is {result['overall_efficiency']['snapshot_overhead_percent']:.1f}% - consider snapshot cleanup"
                )
            
            for pool in result["pool_efficiency"]:
                if pool["utilization_percent"] > 80:
                    result["recommendations"].append(
                        f"Pool '{pool['name']}' is {pool['utilization_percent']:.1f}% full - consider expansion"
                    )
                
                if pool["fragmentation"] and float(pool["fragmentation"].rstrip('%')) > 25:
                    result["recommendations"].append(
                        f"Pool '{pool['name']}' has {pool['fragmentation']} fragmentation - consider defragmentation"
                    )
            
            logger.info("Storage efficiency analysis completed")
            
        except Exception as e:
            logger.error(f"Error analyzing storage efficiency: {e}")
        
        return result
    
    def start(self) -> None:
        """Start the monitoring service."""
        logger.info("Monitor started")
    
    def stop(self) -> None:
        """Stop the monitoring service."""
        logger.info("Monitor stopped")