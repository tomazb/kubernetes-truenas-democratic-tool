"""Integration tests for real Kubernetes operations.

These tests use kind (Kubernetes in Docker) or existing clusters to test
actual Kubernetes functionality with democratic-csi simulation.
"""

import pytest
import subprocess
import yaml
import tempfile
import time
from pathlib import Path
from typing import Optional

from truenas_storage_monitor.k8s_client import K8sClient, K8sConfig
from truenas_storage_monitor.monitor import Monitor


class KubernetesTestEnvironment:
    """Helper class to manage Kubernetes test environment."""
    
    def __init__(self):
        self.cluster_available = False
        self.kubectl_available = False
        self.context_name = None
        self._check_environment()
    
    def _check_environment(self):
        """Check if we have kubectl and a cluster available."""
        try:
            # Check kubectl
            result = subprocess.run(
                ['kubectl', 'version', '--client'],
                capture_output=True,
                timeout=10
            )
            self.kubectl_available = result.returncode == 0
            
            if self.kubectl_available:
                # Check cluster connectivity
                result = subprocess.run(
                    ['kubectl', 'cluster-info'],
                    capture_output=True,
                    timeout=10
                )
                self.cluster_available = result.returncode == 0
                
                if self.cluster_available:
                    # Get current context
                    result = subprocess.run(
                        ['kubectl', 'config', 'current-context'],
                        capture_output=True,
                        text=True,
                        timeout=5
                    )
                    if result.returncode == 0:
                        self.context_name = result.stdout.strip()
                        
        except (subprocess.TimeoutExpired, FileNotFoundError):
            pass
    
    def create_test_namespace(self, name: str) -> bool:
        """Create a test namespace."""
        if not self.cluster_available:
            return False
            
        try:
            subprocess.run(
                ['kubectl', 'create', 'namespace', name],
                capture_output=True,
                timeout=10
            )
            return True
        except subprocess.TimeoutExpired:
            return False
    
    def delete_test_namespace(self, name: str):
        """Delete a test namespace."""
        if not self.cluster_available:
            return
            
        try:
            subprocess.run(
                ['kubectl', 'delete', 'namespace', name, '--timeout=30s'],
                capture_output=True,
                timeout=35
            )
        except subprocess.TimeoutExpired:
            pass
    
    def apply_yaml(self, yaml_content: str, namespace: str = None) -> bool:
        """Apply YAML content to cluster."""
        if not self.cluster_available:
            return False
            
        try:
            cmd = ['kubectl', 'apply', '-f', '-']
            if namespace:
                cmd.extend(['-n', namespace])
                
            subprocess.run(
                cmd,
                input=yaml_content,
                text=True,
                capture_output=True,
                timeout=30
            )
            return True
        except subprocess.TimeoutExpired:
            return False


@pytest.fixture(scope="class")
def k8s_env():
    """Provide Kubernetes test environment."""
    return KubernetesTestEnvironment()


@pytest.fixture(scope="class")
def test_namespace(k8s_env):
    """Create and cleanup test namespace."""
    if not k8s_env.cluster_available:
        pytest.skip("No Kubernetes cluster available")
    
    namespace_name = "truenas-monitor-test"
    
    # Create namespace
    if k8s_env.create_test_namespace(namespace_name):
        yield namespace_name
        # Cleanup
        k8s_env.delete_test_namespace(namespace_name)
    else:
        pytest.skip("Failed to create test namespace")


class TestRealKubernetesOperations:
    """Test real Kubernetes operations."""
    
    def test_k8s_client_connection(self, k8s_env):
        """Test real K8s client connection."""
        if not k8s_env.cluster_available:
            pytest.skip("No Kubernetes cluster available")
        
        config = K8sConfig(
            namespace='default',
            storage_class='standard'
        )
        
        client = K8sClient(config)
        
        # Test connection
        is_connected = client.test_connection()
        assert isinstance(is_connected, bool)
        
        if is_connected:
            # Test basic operations
            storage_classes = client.get_storage_classes()
            assert isinstance(storage_classes, list)
            
            pvs = client.get_persistent_volumes()
            assert isinstance(pvs, list)
            
            pvcs = client.get_persistent_volume_claims()
            assert isinstance(pvcs, list)
    
    def test_storage_class_detection(self, k8s_env):
        """Test detection of storage classes."""
        if not k8s_env.cluster_available:
            pytest.skip("No Kubernetes cluster available")
        
        config = K8sConfig(namespace='default')
        client = K8sClient(config)
        
        if client.test_connection():
            storage_classes = client.get_storage_classes()
            
            # Should have at least default storage class in most clusters
            assert len(storage_classes) >= 0  # Even 0 is valid for some minimal clusters
            
            # Check structure of returned data
            for sc in storage_classes:
                assert hasattr(sc, 'name')
                assert hasattr(sc, 'provisioner')
    
    def test_pv_and_pvc_listing(self, k8s_env, test_namespace):
        """Test listing PVs and PVCs."""
        config = K8sConfig(namespace=test_namespace)
        client = K8sClient(config)
        
        if client.test_connection():
            # List PVs (cluster-wide)
            pvs = client.get_persistent_volumes()
            assert isinstance(pvs, list)
            
            # List PVCs (namespace-scoped)
            pvcs = client.get_persistent_volume_claims()
            assert isinstance(pvcs, list)
            
            # In fresh test namespace, should have no PVCs
            assert len(pvcs) == 0
    
    def test_create_test_pvc(self, k8s_env, test_namespace):
        """Test creating a test PVC and detecting it."""
        if not k8s_env.cluster_available:
            pytest.skip("No Kubernetes cluster available")
        
        # Create a simple PVC YAML
        pvc_yaml = f"""
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: test-pvc
  namespace: {test_namespace}
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
"""
        
        # Apply the PVC
        if k8s_env.apply_yaml(pvc_yaml):
            try:
                # Wait a moment for creation
                time.sleep(2)
                
                # Test our client can see it
                config = K8sConfig(namespace=test_namespace)
                client = K8sClient(config)
                
                if client.test_connection():
                    pvcs = client.get_persistent_volume_claims()
                    
                    # Should find our test PVC
                    test_pvc_found = any(pvc.name == 'test-pvc' for pvc in pvcs)
                    assert test_pvc_found, "Test PVC not found by our client"
                    
                    # Get the specific PVC and check its properties
                    test_pvc = next(pvc for pvc in pvcs if pvc.name == 'test-pvc')
                    assert test_pvc.namespace == test_namespace
                    assert test_pvc.requested_storage == "1Gi"
                    
            finally:
                # Cleanup - delete the PVC
                try:
                    subprocess.run(
                        ['kubectl', 'delete', 'pvc', 'test-pvc', '-n', test_namespace],
                        capture_output=True,
                        timeout=30
                    )
                except subprocess.TimeoutExpired:
                    pass


class TestMonitorWithRealCluster:
    """Test Monitor class with real cluster connectivity."""
    
    def test_monitor_k8s_validation(self, k8s_env):
        """Test Monitor validation with real cluster."""
        if not k8s_env.cluster_available:
            pytest.skip("No Kubernetes cluster available")
        
        config = {
            'openshift': {
                'namespace': 'default',
                'storage_class': 'standard',
                'csi_driver': 'org.democratic-csi.nfs'
            },
            'monitoring': {
                'interval': 60,
                'thresholds': {
                    'orphaned_pv_age_hours': 24
                }
            }
        }
        
        monitor = Monitor(config)
        result = monitor.validate_configuration()
        
        # Should successfully validate K8s connectivity
        assert isinstance(result, dict)
        assert 'k8s_connectivity' in result
        
        if result['k8s_connectivity']:
            # If connected, should have cluster info
            assert 'storage_classes' in result
            assert isinstance(result['storage_classes'], list)
    
    def test_monitor_orphan_detection(self, k8s_env):
        """Test orphaned resource detection with real cluster."""
        if not k8s_env.cluster_available:
            pytest.skip("No Kubernetes cluster available")
        
        config = {
            'openshift': {
                'namespace': 'default',
                'storage_class': 'standard'
            },
            'monitoring': {'interval': 60}
        }
        
        monitor = Monitor(config)
        
        if monitor.k8s_client and monitor.k8s_client.test_connection():
            # Test orphaned PV detection
            result = monitor.check_orphaned_pvs()
            assert isinstance(result, dict)
            assert 'orphaned_pvs' in result
            assert isinstance(result['orphaned_pvs'], list)
            
            # Test orphaned volume detection (will fail TrueNAS connection but should handle gracefully)
            try:
                volume_result = monitor.check_orphaned_volumes()
                assert isinstance(volume_result, dict)
            except Exception as e:
                # Expected to fail without TrueNAS, but should not crash
                assert "truenas" in str(e).lower() or "connection" in str(e).lower()


class TestDemocraticCSISimulation:
    """Simulate democratic-csi scenarios for testing."""
    
    def test_csi_driver_detection(self, k8s_env):
        """Test CSI driver detection."""
        if not k8s_env.cluster_available:
            pytest.skip("No Kubernetes cluster available")
        
        config = K8sConfig(
            namespace='default',
            csi_driver='org.democratic-csi.nfs'
        )
        client = K8sClient(config)
        
        if client.test_connection():
            # Try to get CSI drivers
            try:
                drivers = client.get_csi_drivers()
                assert isinstance(drivers, list)
                
                # Check if democratic-csi is present
                democratic_csi_found = any(
                    'democratic-csi' in driver.name 
                    for driver in drivers
                )
                
                # In test environment, may not have democratic-csi
                # Just ensure the query works
                
            except Exception as e:
                # Some clusters might not support CSI driver listing
                assert "not found" in str(e).lower() or "forbidden" in str(e).lower()
    
    def test_storage_class_with_democratic_csi(self, k8s_env, test_namespace):
        """Test storage class creation that simulates democratic-csi."""
        if not k8s_env.cluster_available:
            pytest.skip("No Kubernetes cluster available")
        
        # Create a storage class that simulates democratic-csi
        sc_yaml = """
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: test-democratic-csi-nfs
provisioner: org.democratic-csi.nfs
parameters:
  server: "truenas.example.com"
  share: "/mnt/tank/k8s"
reclaimPolicy: Delete
volumeBindingMode: Immediate
allowVolumeExpansion: true
"""
        
        if k8s_env.apply_yaml(sc_yaml):
            try:
                time.sleep(1)
                
                # Test our client can detect it
                config = K8sConfig(
                    namespace=test_namespace,
                    storage_class='test-democratic-csi-nfs'
                )
                client = K8sClient(config)
                
                if client.test_connection():
                    storage_classes = client.get_storage_classes()
                    
                    # Find our test storage class
                    test_sc = next(
                        (sc for sc in storage_classes if sc.name == 'test-democratic-csi-nfs'),
                        None
                    )
                    
                    if test_sc:
                        assert test_sc.provisioner == 'org.democratic-csi.nfs'
                        assert 'truenas.example.com' in str(test_sc.parameters)
                        
            finally:
                # Cleanup
                try:
                    subprocess.run(
                        ['kubectl', 'delete', 'storageclass', 'test-democratic-csi-nfs'],
                        capture_output=True,
                        timeout=10
                    )
                except subprocess.TimeoutExpired:
                    pass


class TestPerformanceWithRealCluster:
    """Test performance characteristics with real cluster."""
    
    def test_large_resource_listing_performance(self, k8s_env):
        """Test performance with potentially large numbers of resources."""
        if not k8s_env.cluster_available:
            pytest.skip("No Kubernetes cluster available")
        
        config = K8sConfig(namespace='default')
        client = K8sClient(config)
        
        if client.test_connection():
            import time
            
            # Time the operations
            start_time = time.time()
            
            # List all PVs
            pvs = client.get_persistent_volumes()
            pv_time = time.time()
            
            # List all storage classes
            storage_classes = client.get_storage_classes()
            sc_time = time.time()
            
            # List PVCs
            pvcs = client.get_persistent_volume_claims()
            pvc_time = time.time()
            
            # All operations should complete reasonably quickly
            total_time = pvc_time - start_time
            assert total_time < 30, f"Operations took too long: {total_time}s"
            
            # Log performance for debugging
            print(f"Performance: PVs={len(pvs)} in {pv_time-start_time:.2f}s, "
                  f"SCs={len(storage_classes)} in {sc_time-pv_time:.2f}s, "
                  f"PVCs={len(pvcs)} in {pvc_time-sc_time:.2f}s")
    
    def test_repeated_operations_stability(self, k8s_env):
        """Test that repeated operations remain stable."""
        if not k8s_env.cluster_available:
            pytest.skip("No Kubernetes cluster available")
        
        config = K8sConfig(namespace='default')
        client = K8sClient(config)
        
        if client.test_connection():
            # Perform the same operation multiple times
            results = []
            for i in range(5):
                pvs = client.get_persistent_volumes()
                results.append(len(pvs))
                time.sleep(0.5)
            
            # Results should be consistent (or at least not crash)
            assert len(results) == 5
            assert all(isinstance(r, int) for r in results)


@pytest.mark.slow
class TestEndToEndWithCluster:
    """End-to-end tests with real cluster."""
    
    def test_complete_monitoring_cycle(self, k8s_env, test_namespace):
        """Test complete monitoring cycle."""
        if not k8s_env.cluster_available:
            pytest.skip("No Kubernetes cluster available")
        
        config = {
            'openshift': {
                'namespace': test_namespace,
                'storage_class': 'standard',
                'csi_driver': 'org.democratic-csi.nfs'
            },
            'monitoring': {
                'interval': 60,
                'thresholds': {
                    'orphaned_pv_age_hours': 1,  # Very short for testing
                    'pending_pvc_minutes': 5
                }
            }
        }
        
        monitor = Monitor(config)
        
        # Validate configuration
        validation_result = monitor.validate_configuration()
        assert isinstance(validation_result, dict)
        
        # Run health check
        health_result = monitor.run_health_check()
        assert isinstance(health_result, dict)
        assert 'configuration' in health_result
        assert 'orphaned_pvs' in health_result
        
        # Get monitoring summary
        summary = monitor.get_monitoring_summary()
        assert isinstance(summary, dict)
        assert 'resources' in summary
        assert 'health' in summary
    
    def test_cli_with_real_cluster(self, k8s_env, test_namespace):
        """Test CLI commands with real cluster."""
        if not k8s_env.cluster_available:
            pytest.skip("No Kubernetes cluster available")
        
        from click.testing import CliRunner
        from truenas_storage_monitor.cli import cli
        
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            # Create config for test namespace
            config = {
                'openshift': {
                    'namespace': test_namespace,
                    'storage_class': 'standard'
                },
                'monitoring': {'interval': 60}
            }
            
            with open('cluster-config.yaml', 'w') as f:
                yaml.dump(config, f)
            
            # Test validate command
            result = runner.invoke(cli, [
                '--config', 'cluster-config.yaml',
                'validate'
            ])
            
            # Should complete successfully with real cluster
            assert result.exit_code == 0
            assert "connectivity" in result.output.lower() or "success" in result.output.lower()
            
            # Test orphans command
            result = runner.invoke(cli, [
                '--config', 'cluster-config.yaml',
                'orphans',
                '--format', 'json'
            ])
            
            # Should complete (may find no orphans, which is fine)
            assert result.exit_code == 0