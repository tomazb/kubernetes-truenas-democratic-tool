"""Core monitoring functionality."""

import asyncio
import logging
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any

from .k8s_client import K8sClient
from .truenas_client import TrueNASClient
from .config import Config
from .exceptions import TrueNASMonitorError


logger = logging.getLogger(__name__)


class Monitor:
    """Main monitoring class that orchestrates storage monitoring."""
    
    def __init__(self, config: Config):
        """Initialize the monitor with configuration."""
        self.config = config
        self.k8s_client = K8sClient(config.kubernetes)
        self.truenas_client = TrueNASClient(config.truenas)
        
    def find_orphaned_resources(
        self, 
        namespace: Optional[str] = None,
        age_threshold_hours: int = 24
    ) -> Dict[str, Any]:
        """Find orphaned storage resources."""
        logger.info(f"Scanning for orphaned resources (threshold: {age_threshold_hours}h)")
        
        try:
            # Get resources from both systems
            k8s_pvs = self.k8s_client.list_persistent_volumes()
            k8s_pvcs = self.k8s_client.list_persistent_volume_claims(namespace)
            k8s_snapshots = self.k8s_client.list_volume_snapshots(namespace)
            
            truenas_volumes = self.truenas_client.list_volumes()
            truenas_snapshots = self.truenas_client.list_snapshots()
            
            # Find orphaned resources
            orphaned_pvs = self._find_orphaned_pvs(k8s_pvs, truenas_volumes, age_threshold_hours)
            orphaned_pvcs = self._find_orphaned_pvcs(k8s_pvcs, age_threshold_hours)
            orphaned_snapshots = self._find_orphaned_snapshots(k8s_snapshots, truenas_snapshots, age_threshold_hours)
            
            return {
                'timestamp': datetime.now().isoformat(),
                'orphaned_pvs': orphaned_pvs,
                'orphaned_pvcs': orphaned_pvcs,
                'orphaned_snapshots': orphaned_snapshots,
                'total_pvs': len(k8s_pvs),
                'total_pvcs': len(k8s_pvcs),
                'total_snapshots': len(k8s_snapshots),
                'scan_duration': 0  # TODO: Implement timing
            }
            
        except Exception as e:
            logger.error(f"Error finding orphaned resources: {e}")
            raise TrueNASMonitorError(f"Failed to scan for orphaned resources: {e}")
    
    def _find_orphaned_pvs(self, k8s_pvs: List[Dict], truenas_volumes: List[Dict], age_threshold_hours: int) -> List[Dict]:
        """Find PVs without corresponding TrueNAS volumes."""
        orphaned = []
        threshold = datetime.now() - timedelta(hours=age_threshold_hours)
        
        for pv in k8s_pvs:
            # Check if PV is from democratic-csi
            if not self._is_democratic_csi_pv(pv):
                continue
                
            # Check age
            created_str = pv.get('metadata', {}).get('creationTimestamp', '')
            if created_str:
                created = datetime.fromisoformat(created_str.replace('Z', '+00:00'))
                if created > threshold:
                    continue
                    
                # Check if corresponding TrueNAS volume exists
                if not self._has_corresponding_truenas_volume(pv, truenas_volumes):
                    orphaned.append({
                        'name': pv['metadata']['name'],
                        'age': str(datetime.now() - created),
                        'reason': 'No corresponding TrueNAS volume found',
                        'size': pv.get('spec', {}).get('capacity', {}).get('storage', 'Unknown'),
                        'storage_class': pv.get('spec', {}).get('storageClassName', 'Unknown')
                    })
        
        return orphaned
    
    def _find_orphaned_pvcs(self, k8s_pvcs: List[Dict], age_threshold_hours: int) -> List[Dict]:
        """Find unbound PVCs older than threshold."""
        orphaned = []
        threshold = datetime.now() - timedelta(hours=age_threshold_hours)
        
        for pvc in k8s_pvcs:
            # Check if PVC is pending/unbound
            if pvc.get('status', {}).get('phase') != 'Pending':
                continue
                
            # Check age
            created_str = pvc.get('metadata', {}).get('creationTimestamp', '')
            if created_str:
                created = datetime.fromisoformat(created_str.replace('Z', '+00:00'))
                if created <= threshold:
                    orphaned.append({
                        'name': pvc['metadata']['name'],
                        'namespace': pvc['metadata']['namespace'],
                        'age': str(datetime.now() - created),
                        'reason': f'Unbound for {datetime.now() - created}',
                        'size': pvc.get('spec', {}).get('resources', {}).get('requests', {}).get('storage', 'Unknown'),
                        'storage_class': pvc.get('spec', {}).get('storageClassName', 'Unknown')
                    })
        
        return orphaned
    
    def _find_orphaned_snapshots(self, k8s_snapshots: List[Dict], truenas_snapshots: List[Dict], age_threshold_hours: int) -> List[Dict]:
        """Find snapshots without corresponding TrueNAS snapshots."""
        orphaned = []
        threshold = datetime.now() - timedelta(hours=age_threshold_hours)
        
        for snapshot in k8s_snapshots:
            # Check age
            created_str = snapshot.get('metadata', {}).get('creationTimestamp', '')
            if created_str:
                created = datetime.fromisoformat(created_str.replace('Z', '+00:00'))
                if created > threshold:
                    continue
                    
                # Check if corresponding TrueNAS snapshot exists
                if not self._has_corresponding_truenas_snapshot(snapshot, truenas_snapshots):
                    orphaned.append({
                        'name': snapshot['metadata']['name'],
                        'namespace': snapshot['metadata']['namespace'],
                        'age': str(datetime.now() - created),
                        'reason': 'No corresponding TrueNAS snapshot found',
                        'source_pvc': snapshot.get('spec', {}).get('source', {}).get('persistentVolumeClaimName', 'Unknown')
                    })
        
        return orphaned
    
    def _is_democratic_csi_pv(self, pv: Dict) -> bool:
        """Check if PV is managed by democratic-csi."""
        csi_driver = pv.get('spec', {}).get('csi', {}).get('driver', '')
        return 'democratic-csi' in csi_driver or 'truenas' in csi_driver.lower()
    
    def _has_corresponding_truenas_volume(self, pv: Dict, truenas_volumes: List[Dict]) -> bool:
        """Check if PV has corresponding TrueNAS volume."""
        # Extract volume handle from PV
        volume_handle = pv.get('spec', {}).get('csi', {}).get('volumeHandle', '')
        if not volume_handle:
            return False
            
        # Look for matching TrueNAS volume
        for volume in truenas_volumes:
            if volume.get('name') in volume_handle or volume_handle in volume.get('name', ''):
                return True
        
        return False
    
    def _has_corresponding_truenas_snapshot(self, snapshot: Dict, truenas_snapshots: List[Dict]) -> bool:
        """Check if K8s snapshot has corresponding TrueNAS snapshot."""
        snapshot_name = snapshot['metadata']['name']
        
        # Look for matching TrueNAS snapshot
        for truenas_snapshot in truenas_snapshots:
            if snapshot_name in truenas_snapshot.get('name', '') or truenas_snapshot.get('name', '') in snapshot_name:
                return True
        
        return False
    
    def analyze_storage_usage(self, days: int = 7, namespace: Optional[str] = None) -> Dict[str, Any]:
        """Analyze storage usage and trends."""
        logger.info(f"Analyzing storage usage for {days} days")
        
        try:
            # Get current storage data
            pvcs = self.k8s_client.list_persistent_volume_claims(namespace)
            pvs = self.k8s_client.list_persistent_volumes()
            truenas_volumes = self.truenas_client.list_volumes()
            
            # Calculate metrics
            total_allocated = sum(self._parse_storage_size(pvc.get('spec', {}).get('resources', {}).get('requests', {}).get('storage', '0')) for pvc in pvcs)
            total_used = sum(self._get_volume_used_space(vol) for vol in truenas_volumes)
            
            efficiency = (total_allocated - total_used) / total_allocated * 100 if total_allocated > 0 else 0
            
            return {
                'total_allocated_gb': total_allocated / (1024**3),
                'total_used_gb': total_used / (1024**3),
                'thin_provisioning_efficiency': f"{efficiency:.1f}%",
                'total_pvcs': len(pvcs),
                'total_pvs': len(pvs),
                'growth_trend': 'Stable',  # TODO: Implement trend analysis
                'recommendations': self._generate_recommendations(pvcs, truenas_volumes)
            }
            
        except Exception as e:
            logger.error(f"Error analyzing storage usage: {e}")
            raise TrueNASMonitorError(f"Failed to analyze storage usage: {e}")
    
    def _parse_storage_size(self, size_str: str) -> int:
        """Parse Kubernetes storage size string to bytes."""
        if not size_str:
            return 0
            
        size_str = size_str.upper()
        multipliers = {
            'K': 1024, 'KI': 1024,
            'M': 1024**2, 'MI': 1024**2,
            'G': 1024**3, 'GI': 1024**3,
            'T': 1024**4, 'TI': 1024**4
        }
        
        for suffix, multiplier in multipliers.items():
            if size_str.endswith(suffix):
                return int(float(size_str[:-len(suffix)]) * multiplier)
        
        return int(size_str) if size_str.isdigit() else 0
    
    def _get_volume_used_space(self, volume: Dict) -> int:
        """Get used space for a TrueNAS volume."""
        # This would need to be implemented based on TrueNAS API response format
        return volume.get('used_bytes', 0)
    
    def _generate_recommendations(self, pvcs: List[Dict], truenas_volumes: List[Dict]) -> List[str]:
        """Generate storage optimization recommendations."""
        recommendations = []
        
        # Check for oversized PVCs
        for pvc in pvcs:
            requested = self._parse_storage_size(pvc.get('spec', {}).get('resources', {}).get('requests', {}).get('storage', '0'))
            if requested > 100 * 1024**3:  # > 100GB
                recommendations.append(f"Consider reviewing large PVC: {pvc['metadata']['name']} ({requested / 1024**3:.1f}GB)")
        
        # Check for unused volumes
        if len(truenas_volumes) > len(pvcs):
            recommendations.append(f"Found {len(truenas_volumes) - len(pvcs)} potentially unused TrueNAS volumes")
        
        return recommendations
    
    def generate_report(self) -> Dict[str, Any]:
        """Generate comprehensive storage report."""
        logger.info("Generating comprehensive storage report")
        
        try:
            orphans = self.find_orphaned_resources()
            analysis = self.analyze_storage_usage()
            health = self.check_health()
            
            return {
                'timestamp': datetime.now().isoformat(),
                'summary': {
                    'total_orphaned_resources': len(orphans['orphaned_pvs']) + len(orphans['orphaned_pvcs']) + len(orphans['orphaned_snapshots']),
                    'storage_efficiency': analysis.get('thin_provisioning_efficiency', 'Unknown'),
                    'health_status': 'Healthy' if health['healthy'] else 'Issues Found'
                },
                'orphaned_resources': orphans,
                'storage_analysis': analysis,
                'health_check': health,
                'recommendations': analysis.get('recommendations', [])
            }
            
        except Exception as e:
            logger.error(f"Error generating report: {e}")
            raise TrueNASMonitorError(f"Failed to generate report: {e}")
    
    def validate_configuration(self) -> Dict[str, Dict[str, Any]]:
        """Validate system configuration."""
        logger.info("Validating configuration")
        
        results = {}
        
        # Test Kubernetes connection
        try:
            self.k8s_client.test_connection()
            results['kubernetes'] = {'valid': True, 'message': 'Connection successful'}
        except Exception as e:
            results['kubernetes'] = {'valid': False, 'message': str(e)}
        
        # Test TrueNAS connection
        try:
            self.truenas_client.test_connection()
            results['truenas'] = {'valid': True, 'message': 'Connection successful'}
        except Exception as e:
            results['truenas'] = {'valid': False, 'message': str(e)}
        
        # Check democratic-csi namespace
        try:
            namespaces = self.k8s_client.list_namespaces()
            csi_namespace = self.config.kubernetes.get('namespace', 'democratic-csi')
            if any(ns['metadata']['name'] == csi_namespace for ns in namespaces):
                results['democratic_csi'] = {'valid': True, 'message': f'Namespace {csi_namespace} found'}
            else:
                results['democratic_csi'] = {'valid': False, 'message': f'Namespace {csi_namespace} not found'}
        except Exception as e:
            results['democratic_csi'] = {'valid': False, 'message': str(e)}
        
        return results
    
    def check_health(self) -> Dict[str, Any]:
        """Check overall system health."""
        logger.info("Checking system health")
        
        components = {}
        
        # Check Kubernetes health
        try:
            self.k8s_client.test_connection()
            components['kubernetes'] = {'healthy': True, 'message': 'API server accessible'}
        except Exception as e:
            components['kubernetes'] = {'healthy': False, 'message': str(e)}
        
        # Check TrueNAS health
        try:
            self.truenas_client.test_connection()
            components['truenas'] = {'healthy': True, 'message': 'API accessible'}
        except Exception as e:
            components['truenas'] = {'healthy': False, 'message': str(e)}
        
        # Check CSI driver health
        try:
            csi_pods = self.k8s_client.list_pods(namespace=self.config.kubernetes.get('namespace', 'democratic-csi'))
            healthy_pods = [pod for pod in csi_pods if pod.get('status', {}).get('phase') == 'Running']
            if healthy_pods:
                components['csi_driver'] = {'healthy': True, 'message': f'{len(healthy_pods)} CSI pods running'}
            else:
                components['csi_driver'] = {'healthy': False, 'message': 'No healthy CSI pods found'}
        except Exception as e:
            components['csi_driver'] = {'healthy': False, 'message': str(e)}
        
        # Overall health
        overall_healthy = all(comp['healthy'] for comp in components.values())
        
        return {
            'healthy': overall_healthy,
            'components': components,
            'timestamp': datetime.now().isoformat()
        }