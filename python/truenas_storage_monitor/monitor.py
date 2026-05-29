"""Core monitoring functionality."""

import logging
from datetime import timedelta
from typing import Dict, List, Optional, Any

from .k8s_client import (
    K8sClient,
    PersistentVolumeClaimInfo,
    PersistentVolumeInfo,
    VolumeSnapshotInfo,
)
from .truenas_client import TrueNASClient, VolumeInfo, SnapshotInfo
from .config import Config
from .exceptions import TrueNASMonitorError
from .time_utils import ensure_utc, resource_age, utc_now

logger = logging.getLogger(__name__)


class Monitor:
    """Main monitoring class that orchestrates storage monitoring."""

    def __init__(self, config: Config):
        """Initialize the monitor with configuration."""
        self.config = config
        self.k8s_client = K8sClient(config.k8s_config())
        self.truenas_client = TrueNASClient(config.truenas_config())

    def find_orphaned_resources(
        self, namespace: Optional[str] = None, age_threshold_hours: int = 24
    ) -> Dict[str, Any]:
        """Find orphaned storage resources."""
        logger.info(f"Scanning for orphaned resources (threshold: {age_threshold_hours}h)")

        try:
            k8s_pvs = self.k8s_client.get_persistent_volumes()
            k8s_pvcs = self.k8s_client.get_persistent_volume_claims(namespace)
            k8s_snapshots = self.k8s_client.get_volume_snapshots(namespace)

            truenas_volumes = self.truenas_client.get_volumes()
            truenas_snapshots = self.truenas_client.get_snapshots()

            orphaned_pvs = self._find_orphaned_pvs(k8s_pvs, truenas_volumes, age_threshold_hours)
            orphaned_pvcs = self._find_orphaned_pvcs(k8s_pvcs, age_threshold_hours)
            orphaned_snapshots = self._find_orphaned_snapshots(
                k8s_snapshots, truenas_snapshots, age_threshold_hours
            )

            return {
                "timestamp": utc_now().isoformat(),
                "orphaned_pvs": orphaned_pvs,
                "orphaned_pvcs": orphaned_pvcs,
                "orphaned_snapshots": orphaned_snapshots,
                "total_pvs": len(k8s_pvs),
                "total_pvcs": len(k8s_pvcs),
                "total_snapshots": len(k8s_snapshots),
                "scan_duration": 0,  # TODO: Implement timing
            }

        except Exception as e:
            logger.error(f"Error finding orphaned resources: {e}")
            raise TrueNASMonitorError(f"Failed to scan for orphaned resources: {e}")

    def _find_orphaned_pvs(
        self,
        k8s_pvs: List[PersistentVolumeInfo],
        truenas_volumes: List[VolumeInfo],
        age_threshold_hours: int,
    ) -> List[Dict]:
        """Find PVs without corresponding TrueNAS volumes."""
        orphaned = []
        threshold = utc_now() - timedelta(hours=age_threshold_hours)

        for pv in k8s_pvs:
            if not self._is_democratic_csi_pv(pv):
                continue

            if pv.creation_time is None:
                continue

            created = ensure_utc(pv.creation_time)
            if created > threshold:
                continue

            if not self._has_corresponding_truenas_volume(pv, truenas_volumes):
                age = resource_age(created)
                orphaned.append(
                    {
                        "name": pv.name,
                        "age": age,
                        "reason": "No corresponding TrueNAS volume found",
                        "size": pv.capacity or "Unknown",
                        "storage_class": pv.storage_class or "Unknown",
                    }
                )

        return orphaned

    def _find_orphaned_pvcs(
        self, k8s_pvcs: List[PersistentVolumeClaimInfo], age_threshold_hours: int
    ) -> List[Dict]:
        """Find unbound PVCs older than threshold."""
        orphaned = []
        threshold = utc_now() - timedelta(hours=age_threshold_hours)

        for pvc in k8s_pvcs:
            if pvc.phase != "Pending" or pvc.creation_time is None:
                continue

            created = ensure_utc(pvc.creation_time)
            if created <= threshold:
                age = resource_age(created)
                orphaned.append(
                    {
                        "name": pvc.name,
                        "namespace": pvc.namespace,
                        "age": age,
                        "reason": f"Unbound for {age}",
                        "size": pvc.capacity or "Unknown",
                        "storage_class": pvc.storage_class or "Unknown",
                    }
                )

        return orphaned

    def _find_orphaned_snapshots(
        self,
        k8s_snapshots: List[VolumeSnapshotInfo],
        truenas_snapshots: List[SnapshotInfo],
        age_threshold_hours: int,
    ) -> List[Dict]:
        """Find snapshots without corresponding TrueNAS snapshots."""
        orphaned = []
        threshold = utc_now() - timedelta(hours=age_threshold_hours)

        for snapshot in k8s_snapshots:
            if snapshot.creation_time is None:
                continue

            created = ensure_utc(snapshot.creation_time)
            if created > threshold:
                continue

            if not self._has_corresponding_truenas_snapshot(snapshot, truenas_snapshots):
                orphaned.append(
                    {
                        "name": snapshot.name,
                        "namespace": snapshot.namespace,
                        "age": resource_age(created),
                        "reason": "No corresponding TrueNAS snapshot found",
                        "source_pvc": snapshot.source_pvc or "Unknown",
                    }
                )

        return orphaned

    def _is_democratic_csi_pv(self, pv: PersistentVolumeInfo) -> bool:
        """Check if PV is managed by democratic-csi."""
        driver = pv.driver or ""
        return "democratic-csi" in driver or "truenas" in driver.lower()

    def _has_corresponding_truenas_volume(
        self, pv: PersistentVolumeInfo, truenas_volumes: List[VolumeInfo]
    ) -> bool:
        """Check if PV has corresponding TrueNAS volume."""
        volume_handle = pv.volume_handle
        if not volume_handle:
            return False

        for volume in truenas_volumes:
            if volume.name in volume_handle or volume_handle in volume.name:
                return True

        return False

    def _has_corresponding_truenas_snapshot(
        self, snapshot: VolumeSnapshotInfo, truenas_snapshots: List[SnapshotInfo]
    ) -> bool:
        """Check if K8s snapshot has corresponding TrueNAS snapshot."""
        snapshot_name = snapshot.name

        for truenas_snapshot in truenas_snapshots:
            names = {truenas_snapshot.name, truenas_snapshot.full_name}
            if any(name and (snapshot_name in name or name in snapshot_name) for name in names):
                return True

        return False

    def analyze_storage_usage(
        self, days: int = 7, namespace: Optional[str] = None
    ) -> Dict[str, Any]:
        """Analyze storage usage and trends."""
        logger.info(f"Analyzing storage usage for {days} days")

        try:
            pvcs = self.k8s_client.get_persistent_volume_claims(namespace)
            pvs = self.k8s_client.get_persistent_volumes()
            truenas_volumes = self.truenas_client.get_volumes()

            total_allocated = sum(self._parse_storage_size(pvc.capacity or "0") for pvc in pvcs)
            total_used = sum(self._get_volume_used_space(vol) for vol in truenas_volumes)

            efficiency = (
                (total_allocated - total_used) / total_allocated * 100 if total_allocated > 0 else 0
            )

            return {
                "total_allocated_gb": total_allocated / (1024**3),
                "total_used_gb": total_used / (1024**3),
                "thin_provisioning_efficiency": f"{efficiency:.1f}%",
                "total_pvcs": len(pvcs),
                "total_pvs": len(pvs),
                "growth_trend": "Stable",  # TODO: Implement trend analysis
                "recommendations": self._generate_recommendations(pvcs, truenas_volumes),
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
            "K": 1024,
            "KI": 1024,
            "M": 1024**2,
            "MI": 1024**2,
            "G": 1024**3,
            "GI": 1024**3,
            "T": 1024**4,
            "TI": 1024**4,
        }

        for suffix, multiplier in multipliers.items():
            if size_str.endswith(suffix):
                return int(float(size_str[: -len(suffix)]) * multiplier)

        return int(size_str) if size_str.isdigit() else 0

    def _get_volume_used_space(self, volume: VolumeInfo) -> int:
        """Get used space for a TrueNAS volume."""
        return volume.size

    def _generate_recommendations(
        self, pvcs: List[PersistentVolumeClaimInfo], truenas_volumes: List[VolumeInfo]
    ) -> List[str]:
        """Generate storage optimization recommendations."""
        recommendations = []

        for pvc in pvcs:
            requested = self._parse_storage_size(pvc.capacity or "0")
            if requested > 100 * 1024**3:
                size_gb = requested / 1024**3
                recommendations.append(
                    f"Consider reviewing large PVC: {pvc.name} ({size_gb:.1f}GB)"
                )

        if len(truenas_volumes) > len(pvcs):
            recommendations.append(
                f"Found {len(truenas_volumes) - len(pvcs)} potentially unused TrueNAS volumes"
            )

        return recommendations

    def generate_report(self) -> Dict[str, Any]:
        """Generate comprehensive storage report."""
        logger.info("Generating comprehensive storage report")

        try:
            orphans = self.find_orphaned_resources()
            analysis = self.analyze_storage_usage()
            health = self.check_health()

            return {
                "timestamp": utc_now().isoformat(),
                "summary": {
                    "total_orphaned_resources": len(orphans["orphaned_pvs"])
                    + len(orphans["orphaned_pvcs"])
                    + len(orphans["orphaned_snapshots"]),
                    "storage_efficiency": analysis.get("thin_provisioning_efficiency", "Unknown"),
                    "health_status": "Healthy" if health["healthy"] else "Issues Found",
                },
                "orphaned_resources": orphans,
                "storage_analysis": analysis,
                "health_check": health,
                "recommendations": analysis.get("recommendations", []),
            }

        except Exception as e:
            logger.error(f"Error generating report: {e}")
            raise TrueNASMonitorError(f"Failed to generate report: {e}")

    def validate_configuration(self) -> Dict[str, Dict[str, Any]]:
        """Validate system configuration."""
        logger.info("Validating configuration")

        results = {}

        try:
            self.k8s_client.test_connection()
            results["kubernetes"] = {"valid": True, "message": "Connection successful"}
        except Exception as e:
            results["kubernetes"] = {"valid": False, "message": str(e)}

        try:
            self.truenas_client.test_connection()
            results["truenas"] = {"valid": True, "message": "Connection successful"}
        except Exception as e:
            results["truenas"] = {"valid": False, "message": str(e)}

        try:
            namespaces = self.k8s_client.list_namespaces()
            csi_namespace = self.config.openshift.get("namespace", "democratic-csi")
            if csi_namespace in namespaces:
                results["democratic_csi"] = {
                    "valid": True,
                    "message": f"Namespace {csi_namespace} found",
                }
            else:
                results["democratic_csi"] = {
                    "valid": False,
                    "message": f"Namespace {csi_namespace} not found",
                }
        except Exception as e:
            results["democratic_csi"] = {"valid": False, "message": str(e)}

        return results

    def check_health(self) -> Dict[str, Any]:
        """Check overall system health."""
        logger.info("Checking system health")

        components = {}

        try:
            self.k8s_client.test_connection()
            components["kubernetes"] = {"healthy": True, "message": "API server accessible"}
        except Exception as e:
            components["kubernetes"] = {"healthy": False, "message": str(e)}

        try:
            self.truenas_client.test_connection()
            components["truenas"] = {"healthy": True, "message": "API accessible"}
        except Exception as e:
            components["truenas"] = {"healthy": False, "message": str(e)}

        try:
            csi_health = self.k8s_client.check_csi_driver_health()
            components["csi_driver"] = {
                "healthy": csi_health["healthy"],
                "message": csi_health.get("reason", "CSI driver health unknown"),
            }
        except Exception as e:
            components["csi_driver"] = {"healthy": False, "message": str(e)}

        overall_healthy = all(comp["healthy"] for comp in components.values())

        return {
            "healthy": overall_healthy,
            "components": components,
            "timestamp": utc_now().isoformat(),
        }
