"""Unit tests for configuration module."""

import tempfile
import pytest
from pathlib import Path
from unittest.mock import patch, mock_open

from truenas_storage_monitor.config import load_config, ConfigurationError


def test_load_config_missing_file():
    """Test loading config when file doesn't exist."""
    with pytest.raises(FileNotFoundError):
        load_config("nonexistent.yaml")


@patch("truenas_storage_monitor.config.Path.exists")
@patch("builtins.open", new_callable=mock_open, read_data="openshift:\n  namespace: test\nmonitoring: {}")
def test_load_config_from_file(mock_file, mock_exists):
    """Test loading config from file."""
    mock_exists.return_value = True
    
    config = load_config("test.yaml")
    
    assert config is not None
    assert "openshift" in config
    assert config["openshift"]["namespace"] == "test"


def test_load_config_invalid_yaml():
    """Test loading config with invalid YAML."""
    with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
        f.write("invalid: yaml: content:\n  - missing\n    bracket")
        f.flush()
        
        with pytest.raises(ConfigurationError):
            load_config(f.name)


def test_load_config_empty_file():
    """Test loading empty config file."""
    with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
        f.write("")
        f.flush()
        
        config = load_config(f.name)
        assert config == {}


def test_load_config_with_env_vars():
    """Test loading config with environment variable substitution."""
    config_content = """
openshift:
  namespace: ${TEST_NAMESPACE:-default}
truenas:
  password: ${TRUENAS_PASSWORD}
"""
    
    with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
        f.write(config_content)
        f.flush()
        
        with patch.dict('os.environ', {'TEST_NAMESPACE': 'test-ns', 'TRUENAS_PASSWORD': 'secret'}):
            config = load_config(f.name)
            
            assert config["openshift"]["namespace"] == "test-ns"
            assert config["truenas"]["password"] == "secret"


def test_validate_config_missing_sections():
    """Test config validation with missing required sections."""
    config = {"monitoring": {}}
    
    with pytest.raises(ConfigurationError, match="Missing required"):
        from truenas_storage_monitor.config import validate_config
        validate_config(config)


def test_validate_config_valid():
    """Test config validation with valid config."""
    config = {
        "openshift": {"namespace": "test"},
        "truenas": {"url": "https://truenas.example.com", "username": "admin", "password": "secret"}
    }
    
    from truenas_storage_monitor.config import validate_config
    # Should not raise
    validate_config(config)