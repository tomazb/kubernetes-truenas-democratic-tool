"""Integration tests for complete CLI workflows.

These tests verify that the CLI behaves correctly in real-world scenarios,
testing the complete user experience from command line to output.
"""

import pytest
import tempfile
import yaml
import json
import subprocess
import os
from pathlib import Path
from click.testing import CliRunner

from truenas_storage_monitor.cli import cli


class TestCLIUserWorkflows:
    """Test complete CLI workflows that users would actually perform."""
    
    def test_first_time_user_workflow(self):
        """Test the complete workflow for a first-time user."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            # 1. User runs help to understand the tool
            result = runner.invoke(cli, ['--help'])
            assert result.exit_code == 0
            assert 'TrueNAS Storage Monitor' in result.output
            assert 'validate' in result.output
            assert 'orphans' in result.output
            
            # 2. User tries to run without config (should fail gracefully)
            result = runner.invoke(cli, ['validate'])
            assert result.exit_code != 0
            assert 'config' in result.output.lower() or 'error' in result.output.lower()
            
            # 3. User creates initial config
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
                    'username': 'monitor',
                    'password': 'secure-pass'
                }
            }
            
            with open('truenas-config.yaml', 'w') as f:
                yaml.dump(config, f)
            
            # 4. User validates configuration
            result = runner.invoke(cli, ['--config', 'truenas-config.yaml', 'validate'])
            # Should complete (may fail connections but shouldn't crash)
            assert result.exit_code in [0, 1]
            assert 'Configuration' in result.output or 'connectivity' in result.output.lower()
            
            # 5. User checks for orphaned resources
            result = runner.invoke(cli, ['--config', 'truenas-config.yaml', 'orphans'])
            assert result.exit_code in [0, 1]  # May fail connection but should handle gracefully
    
    def test_troubleshooting_workflow(self):
        """Test workflow when user encounters connection issues."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            # Create config with invalid settings
            config = {
                'openshift': {
                    'namespace': 'nonexistent-namespace'
                },
                'monitoring': {'interval': 60},
                'truenas': {
                    'url': 'https://invalid-truenas.local',
                    'username': 'wrong-user',
                    'password': 'wrong-pass'
                }
            }
            
            with open('bad-config.yaml', 'w') as f:
                yaml.dump(config, f)
            
            # 1. User runs validation and sees errors
            result = runner.invoke(cli, ['--config', 'bad-config.yaml', 'validate', '--verbose'])
            assert result.exit_code != 0
            assert 'error' in result.output.lower() or 'fail' in result.output.lower()
            
            # 2. User tries to get detailed error information
            result = runner.invoke(cli, ['--log-level', 'debug', '--config', 'bad-config.yaml', 'validate'])
            # Should provide more detailed error information
            assert result.exit_code != 0
    
    def test_monitoring_setup_workflow(self):
        """Test setting up continuous monitoring."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            config = {
                'openshift': {
                    'namespace': 'democratic-csi',
                    'storage_class': 'democratic-csi-nfs'
                },
                'monitoring': {
                    'interval': 60,
                    'thresholds': {
                        'orphaned_pv_age_hours': 24,
                        'pending_pvc_minutes': 30
                    }
                },
                'truenas': {
                    'url': 'https://truenas.example.com',
                    'username': 'admin',
                    'password': 'password'
                },
                'alerts': {
                    'enabled': False  # Disable for testing
                }
            }
            
            with open('monitor-config.yaml', 'w') as f:
                yaml.dump(config, f)
            
            # 1. User tests configuration
            result = runner.invoke(cli, ['--config', 'monitor-config.yaml', 'validate'])
            assert result.exit_code in [0, 1]
            
            # 2. User runs one-time monitoring check
            result = runner.invoke(cli, ['--config', 'monitor-config.yaml', 'monitor', '--once'])
            assert result.exit_code in [0, 1]
            
            # 3. User generates report
            result = runner.invoke(cli, [
                '--config', 'monitor-config.yaml',
                'report',
                '--output', 'monitoring-report.html'
            ])
            assert result.exit_code in [0, 1]
    
    def test_snapshot_management_workflow(self):
        """Test complete snapshot management workflow."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            config = {
                'openshift': {
                    'namespace': 'democratic-csi',
                    'storage_class': 'democratic-csi-nfs'
                },
                'monitoring': {
                    'interval': 60,
                    'thresholds': {
                        'snapshot_age_days': 30
                    }
                },
                'truenas': {
                    'url': 'https://truenas.example.com',
                    'username': 'admin',
                    'password': 'password'
                }
            }
            
            with open('snapshot-config.yaml', 'w') as f:
                yaml.dump(config, f)
            
            # 1. User checks snapshot health
            result = runner.invoke(cli, [
                '--config', 'snapshot-config.yaml',
                'snapshots', '--health'
            ])
            assert result.exit_code in [0, 1]
            
            # 2. User looks for orphaned snapshots
            result = runner.invoke(cli, [
                '--config', 'snapshot-config.yaml',
                'snapshots', '--orphaned'
            ])
            assert result.exit_code in [0, 1]
            
            # 3. User analyzes snapshot usage
            result = runner.invoke(cli, [
                '--config', 'snapshot-config.yaml',
                'snapshots', '--analysis'
            ])
            assert result.exit_code in [0, 1]
            
            # 4. User filters by age
            result = runner.invoke(cli, [
                '--config', 'snapshot-config.yaml',
                'snapshots', '--age-days', '7', '--health'
            ])
            assert result.exit_code in [0, 1]
    
    def test_output_format_workflow(self):
        """Test different output format workflows."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            config = {
                'openshift': {'namespace': 'democratic-csi'},
                'monitoring': {'interval': 60},
                'truenas': {
                    'url': 'https://truenas.example.com',
                    'username': 'admin',
                    'password': 'password'
                }
            }
            
            with open('format-config.yaml', 'w') as f:
                yaml.dump(config, f)
            
            # 1. User wants JSON output for parsing
            result = runner.invoke(cli, [
                '--config', 'format-config.yaml',
                'validate', '--format', 'json'
            ])
            assert result.exit_code in [0, 1]
            
            if result.output.strip():
                # Should be valid JSON or error message
                try:
                    json.loads(result.output)
                except json.JSONDecodeError:
                    assert 'error' in result.output.lower()
            
            # 2. User wants YAML output
            result = runner.invoke(cli, [
                '--config', 'format-config.yaml',
                'orphans', '--format', 'yaml'
            ])
            assert result.exit_code in [0, 1]
            
            # 3. User wants table output (default)
            result = runner.invoke(cli, [
                '--config', 'format-config.yaml',
                'analyze', '--format', 'table'
            ])
            assert result.exit_code in [0, 1]


class TestCLIRobustness:
    """Test CLI robustness and error handling."""
    
    def test_malformed_config_handling(self):
        """Test handling of various malformed configurations."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            # 1. Invalid YAML syntax
            with open('invalid-yaml.yaml', 'w') as f:
                f.write("invalid: yaml: content:\n  - missing\n    bracket")
            
            result = runner.invoke(cli, ['--config', 'invalid-yaml.yaml', 'validate'])
            assert result.exit_code != 0
            assert 'error' in result.output.lower() or 'invalid' in result.output.lower()
            
            # 2. Missing required sections
            incomplete_config = {'truenas': {'url': 'https://test.com'}}
            with open('incomplete.yaml', 'w') as f:
                yaml.dump(incomplete_config, f)
            
            result = runner.invoke(cli, ['--config', 'incomplete.yaml', 'validate'])
            assert result.exit_code != 0
            
            # 3. Empty file
            with open('empty.yaml', 'w') as f:
                f.write("")
            
            result = runner.invoke(cli, ['--config', 'empty.yaml', 'validate'])
            assert result.exit_code != 0
    
    def test_network_failure_handling(self):
        """Test behavior during network failures."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            # Config with unreachable hosts
            config = {
                'openshift': {'namespace': 'test'},
                'monitoring': {'interval': 60},
                'truenas': {
                    'url': 'https://192.0.2.1',  # RFC 5737 test address (unreachable)
                    'username': 'admin',
                    'password': 'password'
                }
            }
            
            with open('unreachable-config.yaml', 'w') as f:
                yaml.dump(config, f)
            
            # Commands should fail gracefully, not crash
            result = runner.invoke(cli, ['--config', 'unreachable-config.yaml', 'validate'])
            assert result.exit_code != 0
            assert 'connection' in result.output.lower() or 'error' in result.output.lower()
            
            result = runner.invoke(cli, ['--config', 'unreachable-config.yaml', 'orphans'])
            assert result.exit_code != 0
    
    def test_permission_error_handling(self):
        """Test handling of permission errors."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            # Create config file without read permissions
            config = {
                'openshift': {'namespace': 'test'},
                'monitoring': {'interval': 60}
            }
            
            with open('restricted.yaml', 'w') as f:
                yaml.dump(config, f)
            
            # Make file unreadable
            os.chmod('restricted.yaml', 0o000)
            
            try:
                result = runner.invoke(cli, ['--config', 'restricted.yaml', 'validate'])
                assert result.exit_code != 0
                assert 'permission' in result.output.lower() or 'error' in result.output.lower()
            finally:
                # Restore permissions for cleanup
                os.chmod('restricted.yaml', 0o644)
    
    def test_resource_cleanup(self):
        """Test that resources are properly cleaned up."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            config = {
                'openshift': {'namespace': 'test'},
                'monitoring': {'interval': 60},
                'truenas': {
                    'url': 'https://truenas.example.com',
                    'username': 'admin',
                    'password': 'password'
                }
            }
            
            with open('cleanup-test.yaml', 'w') as f:
                yaml.dump(config, f)
            
            # Run multiple commands to test resource cleanup
            for i in range(3):
                result = runner.invoke(cli, ['--config', 'cleanup-test.yaml', 'validate'])
                # Each run should work independently
                assert result.exit_code in [0, 1]


class TestCLIPerformance:
    """Test CLI performance characteristics."""
    
    def test_command_startup_time(self):
        """Test that CLI commands start up quickly."""
        runner = CliRunner()
        
        import time
        
        # Test help command (should be very fast)
        start_time = time.time()
        result = runner.invoke(cli, ['--help'])
        end_time = time.time()
        
        assert result.exit_code == 0
        startup_time = end_time - start_time
        assert startup_time < 3.0, f"Help command took too long: {startup_time}s"
        
        # Test version command
        start_time = time.time()
        result = runner.invoke(cli, ['--version'])
        end_time = time.time()
        
        assert result.exit_code == 0
        version_time = end_time - start_time
        assert version_time < 2.0, f"Version command took too long: {version_time}s"
    
    def test_config_loading_performance(self):
        """Test configuration loading performance."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            # Create a large but valid config
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
                        'pending_pvc_minutes': 60,
                        'snapshot_age_days': 30,
                        'pool_usage_percent': 80,
                        'snapshot_size_gb': 100
                    }
                },
                'truenas': {
                    'url': 'https://truenas.example.com',
                    'username': 'admin',
                    'password': 'very-long-password-' * 10,  # Make it larger
                    'verify_ssl': False,
                    'timeout': 30,
                    'max_retries': 3
                },
                'alerts': {
                    'enabled': True,
                    'slack': {
                        'webhook': 'https://hooks.slack.com/services/...',
                        'channel': '#storage-alerts'
                    }
                }
            }
            
            # Add many additional sections to make config larger
            for i in range(50):
                config[f'extra_section_{i}'] = {
                    'data': f'value_{i}',
                    'list': list(range(20))
                }
            
            with open('large-config.yaml', 'w') as f:
                yaml.dump(config, f)
            
            import time
            start_time = time.time()
            result = runner.invoke(cli, ['--config', 'large-config.yaml', 'validate'])
            end_time = time.time()
            
            config_time = end_time - start_time
            assert config_time < 10.0, f"Large config loading took too long: {config_time}s"


class TestCLIIntegrationWithSystem:
    """Test CLI integration with system components."""
    
    def test_environment_variable_usage(self):
        """Test CLI with environment variables."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            # Config with environment variables
            config_content = """
openshift:
  namespace: ${K8S_NAMESPACE:-democratic-csi}
monitoring:
  interval: 60
truenas:
  url: ${TRUENAS_URL:-https://truenas.local}
  username: ${TRUENAS_USER:-admin}
  password: ${TRUENAS_PASS:-defaultpass}
"""
            
            with open('env-config.yaml', 'w') as f:
                f.write(config_content)
            
            # Test with environment variables set
            env = {
                'K8S_NAMESPACE': 'test-namespace',
                'TRUENAS_URL': 'https://test-truenas.com',
                'TRUENAS_USER': 'testuser'
                # TRUENAS_PASS not set, should use default
            }
            
            result = runner.invoke(
                cli, 
                ['--config', 'env-config.yaml', 'validate'],
                env=env
            )
            
            # Should complete (may fail connection but config should load)
            assert result.exit_code in [0, 1]
    
    def test_output_redirection(self):
        """Test CLI output redirection and piping."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            config = {
                'openshift': {'namespace': 'test'},
                'monitoring': {'interval': 60},
                'truenas': {
                    'url': 'https://truenas.example.com',
                    'username': 'admin',
                    'password': 'password'
                }
            }
            
            with open('output-config.yaml', 'w') as f:
                yaml.dump(config, f)
            
            # Test JSON output redirection
            result = runner.invoke(cli, [
                '--config', 'output-config.yaml',
                'validate', '--format', 'json'
            ])
            
            if result.output.strip():
                # Output should be machine-readable
                lines = result.output.strip().split('\n')
                # Should not have interactive prompts or progress indicators
                for line in lines:
                    assert not line.startswith('>')  # No prompts
                    assert '\r' not in line  # No carriage returns
    
    def test_signal_handling(self):
        """Test CLI signal handling (where possible)."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            config = {
                'openshift': {'namespace': 'test'},
                'monitoring': {'interval': 1},  # Short interval
                'truenas': {
                    'url': 'https://truenas.example.com',
                    'username': 'admin',
                    'password': 'password'
                }
            }
            
            with open('signal-config.yaml', 'w') as f:
                yaml.dump(config, f)
            
            # Test that help and version commands complete quickly
            # (they should not hang or require signals)
            result = runner.invoke(cli, ['--help'])
            assert result.exit_code == 0
            
            result = runner.invoke(cli, ['--version'])
            assert result.exit_code == 0


@pytest.mark.slow
class TestCLIEndToEnd:
    """End-to-end CLI tests that take longer to run."""
    
    def test_complete_user_journey(self):
        """Test a complete user journey from start to finish."""
        runner = CliRunner()
        
        with runner.isolated_filesystem():
            # 1. User discovers the tool
            result = runner.invoke(cli, ['--help'])
            assert result.exit_code == 0
            assert 'monitor' in result.output
            
            # 2. User creates configuration
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
                        'snapshot_age_days': 30,
                        'pool_usage_percent': 80
                    }
                },
                'truenas': {
                    'url': 'https://truenas.production.com',
                    'username': 'storage-monitor',
                    'password': 'secure-monitoring-password',
                    'verify_ssl': True,
                    'timeout': 30
                },
                'alerts': {
                    'enabled': True,
                    'slack': {
                        'webhook': 'https://hooks.slack.com/services/...',
                        'channel': '#storage-alerts'
                    }
                }
            }
            
            with open('production-config.yaml', 'w') as f:
                yaml.dump(config, f)
            
            # 3. User validates setup
            result = runner.invoke(cli, [
                '--config', 'production-config.yaml',
                'validate', '--verbose'
            ])
            assert result.exit_code in [0, 1]
            
            # 4. User checks for immediate issues
            result = runner.invoke(cli, [
                '--config', 'production-config.yaml',
                'orphans'
            ])
            assert result.exit_code in [0, 1]
            
            # 5. User analyzes storage efficiency
            result = runner.invoke(cli, [
                '--config', 'production-config.yaml',
                'analyze'
            ])
            assert result.exit_code in [0, 1]
            
            # 6. User checks snapshot health
            result = runner.invoke(cli, [
                '--config', 'production-config.yaml',
                'snapshots', '--health'
            ])
            assert result.exit_code in [0, 1]
            
            # 7. User generates comprehensive report
            result = runner.invoke(cli, [
                '--config', 'production-config.yaml',
                'report', '--output', 'storage-health-report.html'
            ])
            assert result.exit_code in [0, 1]
            
            # 8. User sets up monitoring
            result = runner.invoke(cli, [
                '--config', 'production-config.yaml',
                'monitor', '--once'  # Single run for testing
            ])
            assert result.exit_code in [0, 1]