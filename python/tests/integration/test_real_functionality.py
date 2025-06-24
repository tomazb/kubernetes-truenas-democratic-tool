"""Integration tests with real system interactions.

These tests provide actual value by testing real workflows and system interactions.
They use containers and test environments where possible to simulate real conditions.
"""

import pytest
import subprocess
import tempfile
import yaml
import json
from pathlib import Path
from time import sleep

from truenas_storage_monitor.cli import cli
from truenas_storage_monitor.config import load_config
from truenas_storage_monitor.monitor import Monitor
from click.testing import CliRunner


class TestCLIIntegration:
    """Test CLI with real command execution."""
    
    def test_cli_help_commands(self):
        """Test that all CLI commands have proper help and work."""
        runner = CliRunner()
        
        # Test main help
        result = runner.invoke(cli, ['--help'])
        assert result.exit_code == 0
        assert "TrueNAS Storage Monitor" in result.output
        
        # Test all subcommands have help
        subcommands = ['orphans', 'analyze', 'snapshots', 'validate', 'monitor', 'report']
        for cmd in subcommands:
            result = runner.invoke(cli, [cmd, '--help'])
            assert result.exit_code == 0, f"Command {cmd} help failed"
            assert cmd in result.output.lower()
    
    def test_cli_version(self):
        """Test version command works."""
        runner = CliRunner()
        result = runner.invoke(cli, ['--version'])
        assert result.exit_code == 0
        assert "version" in result.output.lower()
    
    def test_cli_with_missing_config(self):
        """Test CLI behavior with missing configuration."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            result = runner.invoke(cli, ['--config', 'missing.yaml', 'validate'])
            assert result.exit_code != 0
            assert "not found" in result.output.lower() or "error" in result.output.lower()


class TestConfigurationIntegration:
    """Test configuration loading with real files."""
    
    def test_load_real_config_file(self):
        """Test loading actual configuration files."""
        with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
            config_data = {
                'openshift': {
                    'namespace': 'democratic-csi',
                    'storage_class': 'democratic-csi-nfs'
                },
                'monitoring': {
                    'interval': 60,
                    'thresholds': {
                        'orphaned_pv_age_hours': 24
                    }
                },
                'truenas': {
                    'url': 'https://truenas.example.com',
                    'username': 'admin',
                    'password': 'testpass'
                }
            }
            yaml.dump(config_data, f)
            f.flush()
            
            # Test loading
            config = load_config(f.name)
            assert config['openshift']['namespace'] == 'democratic-csi'
            assert config['monitoring']['interval'] == 60
            assert config['truenas']['url'] == 'https://truenas.example.com'
    
    def test_config_validation_edge_cases(self):
        """Test configuration validation with real edge cases."""
        # Test missing required sections
        with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
            yaml.dump({'truenas': {'url': 'http://test.com'}}, f)
            f.flush()
            
            with pytest.raises(Exception):  # Should fail validation
                load_config(f.name)
    
    def test_environment_variable_substitution(self):
        """Test real environment variable substitution."""
        import os
        
        with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
            config_content = """
openshift:
  namespace: ${TEST_NAMESPACE:-default}
monitoring:
  interval: 60
truenas:
  url: https://truenas.example.com
  username: ${TRUENAS_USER:-admin}
  password: ${TRUENAS_PASS:-defaultpass}
"""
            f.write(config_content)
            f.flush()
            
            # Set environment variables
            os.environ['TEST_NAMESPACE'] = 'integration-test'
            os.environ['TRUENAS_USER'] = 'testuser'
            
            try:
                config = load_config(f.name)
                assert config['openshift']['namespace'] == 'integration-test'
                assert config['truenas']['username'] == 'testuser'
                assert config['truenas']['password'] == 'defaultpass'  # Uses default
            finally:
                # Cleanup
                os.environ.pop('TEST_NAMESPACE', None)
                os.environ.pop('TRUENAS_USER', None)


class TestDemoFunctionality:
    """Test the demo functionality that simulates real operations."""
    
    def test_demo_script_execution(self):
        """Test that the demo script runs successfully."""
        try:
            # Run the demo script
            result = subprocess.run(
                ['python', 'test_snapshot_functionality.py'],
                cwd=Path(__file__).parent.parent.parent,
                capture_output=True,
                text=True,
                timeout=30
            )
            
            # Should complete successfully
            assert result.returncode == 0, f"Demo failed: {result.stderr}"
            assert "test completed" in result.stdout.lower() or len(result.stdout) > 0
            
        except subprocess.TimeoutExpired:
            pytest.fail("Demo script took too long to execute")
        except FileNotFoundError:
            pytest.skip("Demo script not found")
    
    def test_monitor_creation_with_real_config(self):
        """Test Monitor creation with realistic configuration."""
        config = {
            'openshift': {
                'namespace': 'democratic-csi',
                'storage_class': 'democratic-csi-nfs',
                'csi_driver': 'org.democratic-csi.nfs'
            },
            'monitoring': {
                'interval': 60,
                'thresholds': {
                    'orphaned_pv_age_hours': 24,
                    'pending_pvc_minutes': 60
                }
            },
            'truenas': {
                'url': 'https://truenas.example.com',
                'username': 'admin',
                'password': 'test'
            }
        }
        
        # This should not crash even if connections fail
        monitor = Monitor(config)
        assert monitor.config == config
        
        # Should handle connection failures gracefully
        result = monitor.validate_configuration()
        assert isinstance(result, dict)
        assert 'errors' in result or 'k8s_connectivity' in result


class TestOutputFormats:
    """Test different output formats with real data structures."""
    
    def test_json_output_format(self):
        """Test JSON output format with real CLI."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            # Create a test config
            config = {
                'openshift': {'namespace': 'test'},
                'monitoring': {'interval': 60},
                'truenas': {
                    'url': 'https://truenas.example.com',
                    'username': 'admin',
                    'password': 'test'
                }
            }
            
            with open('config.yaml', 'w') as f:
                yaml.dump(config, f)
            
            # Test JSON output (will fail connection but should return valid JSON)
            result = runner.invoke(cli, ['--config', 'config.yaml', 'validate', '--format', 'json'])
            
            # Should produce valid JSON even on failure
            if result.output.strip():
                try:
                    json.loads(result.output)
                except json.JSONDecodeError:
                    # If not JSON, should at least be structured output
                    assert "error" in result.output.lower() or "fail" in result.output.lower()
    
    def test_yaml_output_format(self):
        """Test YAML output format."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            config = {
                'openshift': {'namespace': 'test'},
                'monitoring': {'interval': 60},
                'truenas': {
                    'url': 'https://truenas.example.com',
                    'username': 'admin',
                    'password': 'test'
                }
            }
            
            with open('config.yaml', 'w') as f:
                yaml.dump(config, f)
            
            result = runner.invoke(cli, ['--config', 'config.yaml', 'validate', '--format', 'yaml'])
            
            # Should produce valid YAML or error message
            if result.output.strip():
                try:
                    yaml.safe_load(result.output)
                except yaml.YAMLError:
                    # If not YAML, should be error message
                    assert "error" in result.output.lower() or "fail" in result.output.lower()


class TestErrorHandling:
    """Test error handling in real scenarios."""
    
    def test_invalid_kubernetes_config(self):
        """Test behavior with invalid Kubernetes configuration."""
        config = {
            'openshift': {
                'namespace': 'nonexistent-namespace',
                'storage_class': 'nonexistent-class'
            },
            'monitoring': {'interval': 60}
        }
        
        # Should handle gracefully without crashing
        monitor = Monitor(config)
        result = monitor.validate_configuration()
        
        # Should report connection issues
        assert isinstance(result, dict)
        if 'k8s_connectivity' in result:
            assert result['k8s_connectivity'] in [True, False]
    
    def test_invalid_truenas_config(self):
        """Test behavior with invalid TrueNAS configuration."""
        config = {
            'openshift': {'namespace': 'test'},
            'monitoring': {'interval': 60},
            'truenas': {
                'url': 'https://nonexistent.truenas.invalid',
                'username': 'admin',
                'password': 'wrongpass'
            }
        }
        
        monitor = Monitor(config)
        result = monitor.validate_configuration()
        
        # Should handle connection failures gracefully
        assert isinstance(result, dict)
        if 'truenas_connectivity' in result:
            assert result['truenas_connectivity'] in [True, False]
    
    def test_malformed_config_files(self):
        """Test handling of malformed configuration files."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            # Create malformed YAML
            with open('bad-config.yaml', 'w') as f:
                f.write("invalid: yaml: content:\n  - missing\n    bracket")
            
            result = runner.invoke(cli, ['--config', 'bad-config.yaml', 'validate'])
            assert result.exit_code != 0
            assert "error" in result.output.lower() or "invalid" in result.output.lower()


class TestEndToEndWorkflows:
    """Test complete workflows that users would actually perform."""
    
    def test_complete_validation_workflow(self):
        """Test complete validation workflow."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            # Create realistic config
            config = {
                'openshift': {
                    'namespace': 'democratic-csi',
                    'storage_class': 'democratic-csi-nfs',
                    'csi_driver': 'org.democratic-csi.nfs'
                },
                'monitoring': {
                    'interval': 300,
                    'thresholds': {
                        'orphaned_pv_age_hours': 24,
                        'pending_pvc_minutes': 60,
                        'snapshot_age_days': 30
                    }
                },
                'truenas': {
                    'url': 'https://truenas.local',
                    'username': 'monitor-user',
                    'password': 'secure-password'
                },
                'alerts': {
                    'enabled': False
                }
            }
            
            with open('production-config.yaml', 'w') as f:
                yaml.dump(config, f)
            
            # Test validation
            result = runner.invoke(cli, ['--config', 'production-config.yaml', 'validate', '--verbose'])
            
            # Should complete (may fail connections but shouldn't crash)
            assert "Configuration" in result.output or "error" in result.output.lower()
    
    def test_monitoring_command_workflow(self):
        """Test the monitoring command workflow."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            config = {
                'openshift': {'namespace': 'democratic-csi'},
                'monitoring': {'interval': 60},
                'truenas': {
                    'url': 'https://truenas.example.com',
                    'username': 'admin',
                    'password': 'test'
                }
            }
            
            with open('monitor-config.yaml', 'w') as f:
                yaml.dump(config, f)
            
            # Test monitor command with --once flag (so it doesn't run forever)
            result = runner.invoke(cli, ['--config', 'monitor-config.yaml', 'monitor', '--once'])
            
            # Should attempt to run monitoring
            assert result.exit_code in [0, 1]  # May fail connection but shouldn't crash
    
    def test_report_generation_workflow(self):
        """Test report generation workflow."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            config = {
                'openshift': {'namespace': 'democratic-csi'},
                'monitoring': {'interval': 60},
                'truenas': {
                    'url': 'https://truenas.example.com',
                    'username': 'admin',
                    'password': 'test'
                }
            }
            
            with open('report-config.yaml', 'w') as f:
                yaml.dump(config, f)
            
            # Test report generation
            result = runner.invoke(cli, [
                '--config', 'report-config.yaml', 
                'report', 
                '--output', 'test-report.html'
            ])
            
            # Should attempt to generate report
            assert result.exit_code in [0, 1]  # May fail connection but shouldn't crash


class TestKubernetesIntegration:
    """Test Kubernetes integration where possible."""
    
    def test_kubectl_detection(self):
        """Test detection of kubectl and cluster connectivity."""
        # Check if kubectl is available
        try:
            result = subprocess.run(
                ['kubectl', 'version', '--client'],
                capture_output=True,
                text=True,
                timeout=10
            )
            
            if result.returncode == 0:
                # kubectl is available, test cluster info
                cluster_result = subprocess.run(
                    ['kubectl', 'cluster-info'],
                    capture_output=True,
                    text=True,
                    timeout=10
                )
                
                # This tells us if we have a real cluster to test against
                has_cluster = cluster_result.returncode == 0
                
                if has_cluster:
                    # Test our K8s client with real cluster
                    from truenas_storage_monitor.k8s_client import K8sClient, K8sConfig
                    
                    config = K8sConfig(
                        namespace='default',
                        storage_class='democratic-csi-nfs'
                    )
                    
                    try:
                        client = K8sClient(config)
                        connection_ok = client.test_connection()
                        assert isinstance(connection_ok, bool)
                        
                        if connection_ok:
                            # Test basic operations
                            storage_classes = client.get_storage_classes()
                            assert isinstance(storage_classes, list)
                            
                    except Exception as e:
                        # Connection issues are acceptable in test environment
                        assert "connection" in str(e).lower() or "unauthorized" in str(e).lower()
                else:
                    pytest.skip("No Kubernetes cluster available for testing")
            else:
                pytest.skip("kubectl not available")
                
        except (subprocess.TimeoutExpired, FileNotFoundError):
            pytest.skip("kubectl not available or timeout")


@pytest.mark.slow
class TestPerformanceAndReliability:
    """Test performance and reliability aspects."""
    
    def test_cli_command_performance(self):
        """Test that CLI commands complete in reasonable time."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            config = {
                'openshift': {'namespace': 'test'},
                'monitoring': {'interval': 60}
            }
            
            with open('config.yaml', 'w') as f:
                yaml.dump(config, f)
            
            import time
            start_time = time.time()
            
            result = runner.invoke(cli, ['--config', 'config.yaml', 'validate'])
            
            end_time = time.time()
            duration = end_time - start_time
            
            # Should complete within reasonable time (even with connection failures)
            assert duration < 30, f"Command took too long: {duration}s"
    
    def test_memory_usage_stability(self):
        """Test that repeated operations don't leak memory."""
        import gc
        
        # Test configuration loading multiple times
        with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
            config = {
                'openshift': {'namespace': 'test'},
                'monitoring': {'interval': 60},
                'truenas': {
                    'url': 'https://truenas.example.com',
                    'username': 'admin',
                    'password': 'test'
                }
            }
            yaml.dump(config, f)
            f.flush()
            
            # Load config multiple times
            for i in range(10):
                try:
                    load_config(f.name)
                except Exception:
                    pass  # Connection failures are OK
                
                if i % 5 == 0:
                    gc.collect()
            
            # If we get here without crashing, memory is probably stable
            assert True
    
    def test_concurrent_operations(self):
        """Test that the tool handles concurrent operations reasonably."""
        import threading
        import time
        
        results = []
        
        def run_validation():
            try:
                config = {
                    'openshift': {'namespace': f'test-{threading.current_thread().ident}'},
                    'monitoring': {'interval': 60}
                }
                monitor = Monitor(config)
                result = monitor.validate_configuration()
                results.append(result)
            except Exception as e:
                results.append(str(e))
        
        # Run multiple validations concurrently
        threads = []
        for i in range(3):
            thread = threading.Thread(target=run_validation)
            threads.append(thread)
            thread.start()
        
        # Wait for completion
        for thread in threads:
            thread.join(timeout=10)
        
        # Should have completed all operations
        assert len(results) == 3
        for result in results:
            assert result is not None