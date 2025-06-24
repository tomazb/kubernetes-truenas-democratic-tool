"""Unit tests for CLI module."""

import pytest
from unittest.mock import Mock, patch, MagicMock
from click.testing import CliRunner
import json
import yaml

from truenas_storage_monitor.cli import cli, orphans, analyze, snapshots, validate, monitor, report


@pytest.fixture
def runner():
    """Create a Click test runner."""
    return CliRunner()


@pytest.fixture
def mock_config():
    """Create a mock configuration."""
    return {
        "openshift": {"namespace": "democratic-csi"},
        "monitoring": {"interval": 60},
        "truenas": {
            "url": "https://truenas.example.com",
            "username": "admin",
            "password": "test"
        }
    }


@pytest.fixture
def mock_monitor():
    """Create a mock Monitor instance."""
    monitor = Mock()
    monitor.check_orphaned_pvs.return_value = {
        "orphaned_pvs": [],
        "total_orphaned": 0,
        "recommendations": []
    }
    monitor.check_orphaned_volumes.return_value = {
        "orphaned_volumes": [],
        "total_orphaned": 0, 
        "recommendations": []
    }
    monitor.check_snapshot_health.return_value = {
        "k8s_snapshots": {"total": 5},
        "truenas_snapshots": {"total": 8},
        "orphaned_resources": {
            "k8s_orphaned": [],
            "truenas_orphaned": []
        },
        "recommendations": []
    }
    monitor.analyze_storage_efficiency.return_value = {
        "pools": [],
        "datasets": [],
        "efficiency_metrics": {},
        "recommendations": []
    }
    monitor.validate_configuration.return_value = {
        "overall_status": "healthy",
        "k8s_connectivity": True,
        "truenas_connectivity": True,
        "errors": []
    }
    return monitor


def test_cli_help(runner):
    """Test CLI help command."""
    result = runner.invoke(cli, ['--help'])
    assert result.exit_code == 0
    assert "TrueNAS Storage Monitor" in result.output
    assert "orphans" in result.output
    assert "snapshots" in result.output


def test_cli_version(runner):
    """Test CLI version command."""
    result = runner.invoke(cli, ['--version'])
    assert result.exit_code == 0
    assert "version" in result.output.lower()


@patch('truenas_storage_monitor.cli.load_config')
@patch('truenas_storage_monitor.cli.create_monitor')
def test_orphans_command_table_format(mock_create_monitor, mock_load_config, runner, mock_config, mock_monitor):
    """Test orphans command with table format."""
    mock_load_config.return_value = mock_config
    mock_create_monitor.return_value = mock_monitor
    
    result = runner.invoke(orphans, ['--format', 'table'])
    
    assert result.exit_code == 0
    mock_monitor.check_orphaned_pvs.assert_called_once()
    mock_monitor.check_orphaned_volumes.assert_called_once()


@patch('truenas_storage_monitor.cli.load_config')
@patch('truenas_storage_monitor.cli.create_monitor') 
def test_orphans_command_json_format(mock_create_monitor, mock_load_config, runner, mock_config, mock_monitor):
    """Test orphans command with JSON format."""
    mock_load_config.return_value = mock_config
    mock_create_monitor.return_value = mock_monitor
    
    result = runner.invoke(orphans, ['--format', 'json'])
    
    assert result.exit_code == 0
    # Should be valid JSON
    try:
        json.loads(result.output)
    except json.JSONDecodeError:
        pytest.fail("Output is not valid JSON")


@patch('truenas_storage_monitor.cli.load_config')
@patch('truenas_storage_monitor.cli.create_monitor')
def test_orphans_command_yaml_format(mock_create_monitor, mock_load_config, runner, mock_config, mock_monitor):
    """Test orphans command with YAML format."""
    mock_load_config.return_value = mock_config
    mock_create_monitor.return_value = mock_monitor
    
    result = runner.invoke(orphans, ['--format', 'yaml'])
    
    assert result.exit_code == 0
    # Should be valid YAML
    try:
        yaml.safe_load(result.output)
    except yaml.YAMLError:
        pytest.fail("Output is not valid YAML")


@patch('truenas_storage_monitor.cli.load_config')
@patch('truenas_storage_monitor.cli.create_monitor')
def test_analyze_command(mock_create_monitor, mock_load_config, runner, mock_config, mock_monitor):
    """Test analyze command."""
    mock_load_config.return_value = mock_config
    mock_create_monitor.return_value = mock_monitor
    
    result = runner.invoke(analyze, [])
    
    assert result.exit_code == 0
    mock_monitor.analyze_storage_efficiency.assert_called_once()


@patch('truenas_storage_monitor.cli.load_config')
@patch('truenas_storage_monitor.cli.create_monitor')
def test_snapshots_command_health(mock_create_monitor, mock_load_config, runner, mock_config, mock_monitor):
    """Test snapshots command with health flag."""
    mock_load_config.return_value = mock_config
    mock_create_monitor.return_value = mock_monitor
    
    result = runner.invoke(snapshots, ['--health'])
    
    assert result.exit_code == 0
    mock_monitor.check_snapshot_health.assert_called_once()


@patch('truenas_storage_monitor.cli.load_config')
@patch('truenas_storage_monitor.cli.create_monitor')
def test_snapshots_command_analysis(mock_create_monitor, mock_load_config, runner, mock_config, mock_monitor):
    """Test snapshots command with analysis flag."""
    mock_load_config.return_value = mock_config
    mock_create_monitor.return_value = mock_monitor
    
    # Mock the analyze_trends method
    mock_monitor.analyze_trends.return_value = {
        "period_days": 30,
        "snapshot_analysis": {
            "total_snapshots": 10,
            "growth_trend": "stable"
        },
        "recommendations": []
    }
    
    result = runner.invoke(snapshots, ['--analysis'])
    
    assert result.exit_code == 0
    mock_monitor.analyze_trends.assert_called_once()


@patch('truenas_storage_monitor.cli.load_config')
@patch('truenas_storage_monitor.cli.create_monitor')
def test_snapshots_command_orphaned(mock_create_monitor, mock_load_config, runner, mock_config, mock_monitor):
    """Test snapshots command with orphaned flag."""
    mock_load_config.return_value = mock_config
    mock_create_monitor.return_value = mock_monitor
    
    result = runner.invoke(snapshots, ['--orphaned'])
    
    assert result.exit_code == 0
    mock_monitor.check_snapshot_health.assert_called_once()


@patch('truenas_storage_monitor.cli.load_config')
@patch('truenas_storage_monitor.cli.create_monitor')
def test_snapshots_command_with_volume_filter(mock_create_monitor, mock_load_config, runner, mock_config, mock_monitor):
    """Test snapshots command with volume filter."""
    mock_load_config.return_value = mock_config
    mock_create_monitor.return_value = mock_monitor
    
    result = runner.invoke(snapshots, ['--volume', 'test-volume', '--health'])
    
    assert result.exit_code == 0
    mock_monitor.check_snapshot_health.assert_called_once()


@patch('truenas_storage_monitor.cli.load_config')
@patch('truenas_storage_monitor.cli.create_monitor')
def test_validate_command(mock_create_monitor, mock_load_config, runner, mock_config, mock_monitor):
    """Test validate command."""
    mock_load_config.return_value = mock_config
    mock_create_monitor.return_value = mock_monitor
    
    result = runner.invoke(validate, [])
    
    assert result.exit_code == 0
    mock_monitor.validate_configuration.assert_called_once()


@patch('truenas_storage_monitor.cli.load_config')
@patch('truenas_storage_monitor.cli.create_monitor')
def test_validate_command_verbose(mock_create_monitor, mock_load_config, runner, mock_config, mock_monitor):
    """Test validate command with verbose flag."""
    mock_load_config.return_value = mock_config
    mock_create_monitor.return_value = mock_monitor
    
    result = runner.invoke(validate, ['--verbose'])
    
    assert result.exit_code == 0
    mock_monitor.validate_configuration.assert_called_once()


@patch('truenas_storage_monitor.cli.load_config')
@patch('truenas_storage_monitor.cli.create_monitor')
def test_monitor_command(mock_create_monitor, mock_load_config, runner, mock_config, mock_monitor):
    """Test monitor command."""
    mock_load_config.return_value = mock_config
    mock_create_monitor.return_value = mock_monitor
    
    # Mock monitoring summary
    mock_monitor.get_monitoring_summary.return_value = {
        "resources": {"k8s_pvs": 5, "truenas_volumes": 8},
        "health": {"orphaned_pvs": 0, "orphaned_volumes": 1},
        "timestamp": "2024-01-01T12:00:00Z"
    }
    
    # Use --once flag to avoid infinite loop
    result = runner.invoke(monitor, ['--once'])
    
    assert result.exit_code == 0
    mock_monitor.get_monitoring_summary.assert_called()


@patch('truenas_storage_monitor.cli.load_config')
@patch('truenas_storage_monitor.cli.create_monitor')
def test_report_command(mock_create_monitor, mock_load_config, runner, mock_config, mock_monitor):
    """Test report command."""
    mock_load_config.return_value = mock_config
    mock_create_monitor.return_value = mock_monitor
    
    # Mock comprehensive health check
    mock_monitor.run_health_check.return_value = {
        "configuration": {"overall_status": "healthy"},
        "orphaned_pvs": {"total_orphaned": 0},
        "orphaned_volumes": {"total_orphaned": 1},
        "snapshot_health": {"orphaned_resources": {"k8s_orphaned": [], "truenas_orphaned": []}},
        "summary": {"total_issues": 1}
    }
    
    with runner.isolated_filesystem():
        result = runner.invoke(report, ['--output', 'test-report.html'])
        
        assert result.exit_code == 0
        mock_monitor.run_health_check.assert_called_once()


def test_cli_with_custom_config(runner):
    """Test CLI with custom config file."""
    with runner.isolated_filesystem():
        # Create a test config file
        config_content = """
openshift:
  namespace: test-namespace
monitoring:
  interval: 30
truenas:
  url: https://test-truenas.example.com
  username: testuser
  password: testpass
"""
        with open('test-config.yaml', 'w') as f:
            f.write(config_content)
        
        result = runner.invoke(cli, ['--config', 'test-config.yaml', '--help'])
        assert result.exit_code == 0


def test_cli_with_invalid_config(runner):
    """Test CLI with invalid config file."""
    result = runner.invoke(orphans, ['--config', 'nonexistent.yaml'])
    assert result.exit_code != 0


def test_cli_with_log_level(runner):
    """Test CLI with different log levels."""
    result = runner.invoke(cli, ['--log-level', 'debug', '--help'])
    assert result.exit_code == 0
    
    result = runner.invoke(cli, ['--log-level', 'info', '--help'])
    assert result.exit_code == 0


@patch('truenas_storage_monitor.cli.load_config')
def test_cli_config_validation_error(mock_load_config, runner):
    """Test CLI with configuration validation error."""
    from truenas_storage_monitor.exceptions import ConfigurationError
    mock_load_config.side_effect = ConfigurationError("Invalid config")
    
    result = runner.invoke(orphans, [])
    assert result.exit_code != 0
    assert "Invalid config" in result.output


@patch('truenas_storage_monitor.cli.load_config') 
@patch('truenas_storage_monitor.cli.create_monitor')
def test_cli_connection_error(mock_create_monitor, mock_load_config, runner, mock_config):
    """Test CLI with connection error."""
    mock_load_config.return_value = mock_config
    from truenas_storage_monitor.exceptions import TrueNASError
    mock_create_monitor.side_effect = TrueNASError("Connection failed")
    
    result = runner.invoke(orphans, [])
    assert result.exit_code != 0
    assert "Connection failed" in result.output


def test_format_table_output():
    """Test table formatting utility."""
    from truenas_storage_monitor.cli import format_table_output
    
    data = [
        {"name": "item1", "value": "test1"},
        {"name": "item2", "value": "test2"}
    ]
    
    result = format_table_output(data, ["name", "value"])
    assert "item1" in result
    assert "test1" in result
    assert "item2" in result
    assert "test2" in result


def test_format_json_output():
    """Test JSON formatting utility."""
    from truenas_storage_monitor.cli import format_json_output
    
    data = {"test": "value", "number": 42}
    result = format_json_output(data)
    
    # Should be valid JSON
    parsed = json.loads(result)
    assert parsed["test"] == "value"
    assert parsed["number"] == 42


def test_format_yaml_output():
    """Test YAML formatting utility."""
    from truenas_storage_monitor.cli import format_yaml_output
    
    data = {"test": "value", "number": 42}
    result = format_yaml_output(data)
    
    # Should be valid YAML
    parsed = yaml.safe_load(result)
    assert parsed["test"] == "value"
    assert parsed["number"] == 42


@patch('truenas_storage_monitor.cli.load_config')
@patch('truenas_storage_monitor.cli.create_monitor')
def test_snapshots_age_filter(mock_create_monitor, mock_load_config, runner, mock_config, mock_monitor):
    """Test snapshots command with age filter."""
    mock_load_config.return_value = mock_config
    mock_create_monitor.return_value = mock_monitor
    
    result = runner.invoke(snapshots, ['--age-days', '14', '--health'])
    
    assert result.exit_code == 0
    mock_monitor.check_snapshot_health.assert_called_once()


@patch('truenas_storage_monitor.cli.load_config')
@patch('truenas_storage_monitor.cli.create_monitor')
def test_monitor_with_metrics_port(mock_create_monitor, mock_load_config, runner, mock_config, mock_monitor):
    """Test monitor command with metrics port.""" 
    mock_load_config.return_value = mock_config
    mock_create_monitor.return_value = mock_monitor
    
    mock_monitor.get_monitoring_summary.return_value = {
        "resources": {"k8s_pvs": 5},
        "health": {"orphaned_pvs": 0},
        "timestamp": "2024-01-01T12:00:00Z"
    }
    
    result = runner.invoke(monitor, ['--metrics-port', '8080', '--once'])
    
    assert result.exit_code == 0


def test_cli_context_settings():
    """Test CLI context settings."""
    # Test that CLI has proper context settings
    assert cli.context_settings is not None
    

def test_snapshots_command_all_flags(runner):
    """Test snapshots command help shows all flags."""
    result = runner.invoke(snapshots, ['--help'])
    
    assert result.exit_code == 0
    assert "--health" in result.output
    assert "--analysis" in result.output
    assert "--orphaned" in result.output
    assert "--volume" in result.output
    assert "--age-days" in result.output
    assert "--format" in result.output