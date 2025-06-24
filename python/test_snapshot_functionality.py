#!/usr/bin/env python3
"""Test script to demonstrate snapshot functionality."""

import json
import yaml
from datetime import datetime, timedelta
from unittest.mock import Mock, MagicMock, patch

# Import our modules
from truenas_storage_monitor.monitor import Monitor
from truenas_storage_monitor.k8s_client import VolumeSnapshotInfo, OrphanedResource, ResourceType
from truenas_storage_monitor.truenas_client import SnapshotInfo


def create_mock_k8s_snapshots():
    """Create mock K8s snapshots for testing."""
    return [
        VolumeSnapshotInfo(
            name="snapshot-1",
            namespace="default",
            source_pvc="pvc-12345",
            snapshot_class="democratic-csi-nfs",
            ready_to_use=True,
            creation_time=datetime.now() - timedelta(days=5)
        ),
        VolumeSnapshotInfo(
            name="snapshot-2",
            namespace="default",
            source_pvc="pvc-67890",
            snapshot_class="democratic-csi-nfs",
            ready_to_use=False,  # Stuck in pending
            creation_time=datetime.now() - timedelta(days=2)
        ),
        VolumeSnapshotInfo(
            name="old-snapshot",
            namespace="default",
            source_pvc="pvc-old",
            snapshot_class="democratic-csi-nfs",
            ready_to_use=True,
            creation_time=datetime.now() - timedelta(days=45)  # Old snapshot
        )
    ]


def create_mock_truenas_snapshots():
    """Create mock TrueNAS snapshots for testing."""
    return [
        SnapshotInfo(
            name="snapshot-1",
            dataset="tank/k8s/volumes/pvc-12345",
            creation_time=datetime.now() - timedelta(days=5),
            used_size=1024**3,  # 1GB
            referenced_size=2*1024**3,
            full_name="tank/k8s/volumes/pvc-12345@snapshot-1"
        ),
        # This one has no corresponding K8s snapshot (orphaned)
        SnapshotInfo(
            name="orphaned-snap",
            dataset="tank/k8s/volumes/pvc-99999",
            creation_time=datetime.now() - timedelta(days=10),
            used_size=5*1024**3,  # 5GB - large snapshot
            referenced_size=10*1024**3,
            full_name="tank/k8s/volumes/pvc-99999@orphaned-snap"
        ),
        # Very old snapshot
        SnapshotInfo(
            name="ancient-snap",
            dataset="tank/k8s/volumes/pvc-ancient",
            creation_time=datetime.now() - timedelta(days=90),
            used_size=10*1024**3,  # 10GB
            referenced_size=20*1024**3,
            full_name="tank/k8s/volumes/pvc-ancient@ancient-snap"
        )
    ]


def test_snapshot_health():
    """Test snapshot health checking functionality."""
    print("=== Testing Snapshot Health Check ===\n")
    
    # Create mock config
    config = {
        "openshift": {"namespace": "default"},
        "truenas": {"url": "https://truenas.example.com", "username": "admin", "password": "test"}
    }
    
    # Create monitor with mocked clients
    with patch('truenas_storage_monitor.monitor.K8sClient') as mock_k8s, \
         patch('truenas_storage_monitor.monitor.TrueNASClient') as mock_truenas:
        
        monitor = Monitor(config)
        
        # Setup mocks
        monitor.k8s_client = mock_k8s.return_value
        monitor.truenas_client = mock_truenas.return_value
        
        # Mock K8s methods
        monitor.k8s_client.get_volume_snapshots.return_value = create_mock_k8s_snapshots()
        monitor.k8s_client.find_stale_snapshots.return_value = [
            snap for snap in create_mock_k8s_snapshots() 
            if snap.creation_time < datetime.now() - timedelta(days=30)
        ]
        monitor.k8s_client.find_orphaned_snapshots.return_value = []
        
        # Mock TrueNAS methods
        truenas_snapshots = create_mock_truenas_snapshots()
        monitor.truenas_client.analyze_snapshot_usage.return_value = {
            "total_snapshots": len(truenas_snapshots),
            "total_snapshot_size": sum(s.used_size for s in truenas_snapshots),
            "snapshots_by_age": {
                "last_24h": 0,
                "last_week": 2,
                "last_month": 1,
                "older": 2
            },
            "average_snapshot_age_days": 35,
            "large_snapshots": [
                {
                    "name": "orphaned-snap",
                    "dataset": "tank/k8s/volumes/pvc-99999",
                    "size_gb": 5.0,
                    "age_days": 10
                },
                {
                    "name": "ancient-snap",
                    "dataset": "tank/k8s/volumes/pvc-ancient",
                    "size_gb": 10.0,
                    "age_days": 90
                }
            ],
            "recommendations": [
                "Consider cleaning up 2 snapshots older than 30 days",
                "Total snapshot size is 16.0GB - consider retention policy"
            ]
        }
        
        monitor.truenas_client.find_orphaned_truenas_snapshots.return_value = [
            s for s in truenas_snapshots if "orphaned" in s.name or "ancient" in s.name
        ]
        
        # Run health check
        health_status = monitor.check_snapshot_health()
        
        # Print results
        print("Snapshot Health Status:")
        print(json.dumps(health_status, indent=2, default=str))
        
        print("\n✅ Health check completed successfully!")
        print(f"- K8s snapshots: {health_status['k8s_snapshots']['total']} total")
        print(f"- TrueNAS snapshots: {health_status['truenas_snapshots']['total']} total")
        print(f"- Orphaned snapshots: {health_status['orphaned_resources']['total_orphaned']} found")
        print(f"- Alerts generated: {len(health_status['alerts'])}")


def test_snapshot_analysis():
    """Test snapshot analysis functionality."""
    print("\n\n=== Testing Snapshot Analysis ===\n")
    
    config = {"truenas": {"url": "https://truenas.example.com", "username": "admin", "password": "test"}}
    
    with patch('truenas_storage_monitor.truenas_client.requests.Session'):
        from truenas_storage_monitor.truenas_client import TrueNASClient, TrueNASConfig
        
        truenas_config = TrueNASConfig(host="truenas.example.com", username="admin", password="test")
        client = TrueNASClient(truenas_config)
        
        # Mock get_snapshots
        client.get_snapshots = Mock(return_value=create_mock_truenas_snapshots())
        
        # Run analysis
        analysis = client.analyze_snapshot_usage()
        
        print("Snapshot Usage Analysis:")
        print(f"- Total snapshots: {analysis['total_snapshots']}")
        print(f"- Total size: {analysis['total_snapshot_size'] / (1024**3):.2f} GB")
        print(f"- Average age: {analysis['average_snapshot_age_days']:.1f} days")
        print("\nAge distribution:")
        for age_range, count in analysis['snapshots_by_age'].items():
            print(f"  - {age_range}: {count}")
        
        if analysis['large_snapshots']:
            print("\nLarge snapshots (>1GB):")
            for snap in analysis['large_snapshots']:
                print(f"  - {snap['name']}: {snap['size_gb']:.2f} GB, {snap['age_days']} days old")
        
        if analysis['recommendations']:
            print("\nRecommendations:")
            for rec in analysis['recommendations']:
                print(f"  • {rec}")


def test_orphaned_detection():
    """Test orphaned snapshot detection."""
    print("\n\n=== Testing Orphaned Snapshot Detection ===\n")
    
    from truenas_storage_monitor.k8s_client import K8sClient, K8sConfig
    
    # Create K8s client mock
    k8s_config = K8sConfig(namespace="default")
    with patch('truenas_storage_monitor.k8s_client.config'), \
         patch('truenas_storage_monitor.k8s_client.k8s_client'):
        
        client = K8sClient(k8s_config)
        
        # Mock get_volume_snapshots
        client.get_volume_snapshots = Mock(return_value=create_mock_k8s_snapshots())
        
        # Find orphaned snapshots (no matching TrueNAS snapshots)
        mock_truenas_snapshots = [s for s in create_mock_truenas_snapshots() if "snapshot-1" in s.name]
        orphaned = client.find_orphaned_snapshots(mock_truenas_snapshots)
        
        print(f"Found {len(orphaned)} orphaned K8s snapshots:")
        for orphan in orphaned:
            print(f"  - {orphan.name} in {orphan.namespace}: {orphan.reason}")


def test_storage_efficiency():
    """Test storage efficiency analysis."""
    print("\n\n=== Testing Storage Efficiency Analysis ===\n")
    
    config = {
        "openshift": {"namespace": "default"},
        "truenas": {"url": "https://truenas.example.com", "username": "admin", "password": "test"}
    }
    
    with patch('truenas_storage_monitor.monitor.K8sClient') as mock_k8s, \
         patch('truenas_storage_monitor.monitor.TrueNASClient') as mock_truenas:
        
        monitor = Monitor(config)
        monitor.k8s_client = mock_k8s.return_value
        monitor.truenas_client = mock_truenas.return_value
        
        # Mock pool data
        from truenas_storage_monitor.truenas_client import PoolInfo
        monitor.truenas_client.get_pools.return_value = [
            PoolInfo(
                name="tank",
                status="ONLINE",
                total_size=100*1024**3,  # 100GB
                used_size=75*1024**3,    # 75GB used (75%)
                free_size=25*1024**3,
                fragmentation="15%",
                healthy=True
            )
        ]
        
        # Mock PV data
        from truenas_storage_monitor.k8s_client import PersistentVolumeInfo
        monitor.k8s_client.get_persistent_volumes.return_value = [
            PersistentVolumeInfo(
                name=f"pv-{i}",
                volume_handle=f"volume-{i}",
                driver="org.democratic-csi.nfs",
                capacity="10Gi",
                access_modes=["ReadWriteOnce"],
                phase="Bound"
            ) for i in range(10)  # 10 PVs x 10GB = 100GB allocated
        ]
        
        # Mock snapshot analysis
        monitor.truenas_client.analyze_snapshot_usage.return_value = {
            "total_snapshot_size": 15*1024**3  # 15GB in snapshots
        }
        
        # Run efficiency analysis
        efficiency = monitor.analyze_storage_efficiency()
        
        print("Storage Efficiency Analysis:")
        print(f"- Thin provisioning ratio: {efficiency['overall_efficiency']['thin_provisioning_ratio']:.2f}")
        print(f"- Snapshot overhead: {efficiency['overall_efficiency']['snapshot_overhead_percent']:.1f}%")
        
        print("\nPool Efficiency:")
        for pool in efficiency['pool_efficiency']:
            print(f"  - {pool['name']}: {pool['utilization_percent']:.1f}% utilized")
            print(f"    Capacity: {pool['capacity_gb']:.1f} GB")
            print(f"    Used: {pool['used_gb']:.1f} GB")
            print(f"    Fragmentation: {pool['fragmentation']}")
        
        if efficiency['recommendations']:
            print("\nRecommendations:")
            for rec in efficiency['recommendations']:
                print(f"  • {rec}")


if __name__ == "__main__":
    print("🚀 Testing TrueNAS Storage Monitor Snapshot Functionality\n")
    print("This script demonstrates the snapshot management capabilities\n")
    
    test_snapshot_health()
    test_snapshot_analysis()
    test_orphaned_detection()
    test_storage_efficiency()
    
    print("\n\n✅ All functionality tests completed!")
    print("\nYou can use these same commands with real K8s and TrueNAS clusters:")
    print("  truenas-monitor snapshots --health")
    print("  truenas-monitor snapshots --analysis")
    print("  truenas-monitor snapshots --orphaned")
    print("  truenas-monitor analyze")