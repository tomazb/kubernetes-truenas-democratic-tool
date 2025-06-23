"""Unit tests for configuration module."""

import pytest
from unittest.mock import patch, mock_open

from truenas_storage_monitor.config import (
    load_config,
    get_default_config,
    expand_env_vars,
    validate_config,
    merge_configs,
)
from truenas_storage_monitor.exceptions import ConfigurationError


class TestConfigModule:
    """Test cases for configuration module."""
    
    def test_get_default_config(self):
        """Test that default config contains required sections."""
        config = get_default_config()
        
        assert "openshift" in config
        assert "monitoring" in config
        assert "logging" in config
        assert config["monitoring"]["orphan_threshold"] == "24h"
        assert config["logging"]["level"] == "info"
    
    def test_expand_env_vars_simple(self):
        """Test environment variable expansion."""
        with patch.dict("os.environ", {"TEST_VAR": "test_value"}):
            config = {
                "key": "${TEST_VAR}",
                "nested": {
                    "key2": "$TEST_VAR/path"
                }
            }
            
            result = expand_env_vars(config)
            
            assert result["key"] == "test_value"
            assert result["nested"]["key2"] == "test_value/path"
    
    def test_expand_env_vars_list(self):
        """Test environment variable expansion in lists."""
        with patch.dict("os.environ", {"TEST_VAR": "test_value"}):
            config = {
                "items": ["${TEST_VAR}", "static", "$TEST_VAR"]
            }
            
            result = expand_env_vars(config)
            
            assert result["items"] == ["test_value", "static", "test_value"]
    
    def test_validate_config_missing_section(self):
        """Test validation fails for missing required section."""
        config = {"monitoring": {}}  # Missing 'openshift'
        
        with pytest.raises(ConfigurationError, match="Missing required configuration section: openshift"):
            validate_config(config)
    
    def test_validate_config_truenas_missing_url(self):
        """Test validation fails when TrueNAS URL is missing."""
        config = {
            "openshift": {},
            "monitoring": {},
            "truenas": {
                "username": "admin",
                "password": "pass"
            }
        }
        
        with pytest.raises(ConfigurationError, match="TrueNAS URL is required"):
            validate_config(config)
    
    def test_validate_config_truenas_missing_auth(self):
        """Test validation fails when TrueNAS auth is missing."""
        config = {
            "openshift": {},
            "monitoring": {},
            "truenas": {
                "url": "https://truenas.example.com"
            }
        }
        
        with pytest.raises(ConfigurationError, match="TrueNAS authentication required"):
            validate_config(config)
    
    def test_validate_config_invalid_thresholds(self):
        """Test validation fails for invalid threshold values."""
        config = {
            "openshift": {},
            "monitoring": {
                "storage": {
                    "pool_warning_threshold": 90,
                    "pool_critical_threshold": 80  # Warning > Critical
                }
            }
        }
        
        with pytest.raises(ConfigurationError, match="warning threshold must be less than critical"):
            validate_config(config)
    
    def test_merge_configs_simple(self):
        """Test simple config merging."""
        base = {"a": 1, "b": 2}
        override = {"b": 3, "c": 4}
        
        result = merge_configs(base, override)
        
        assert result == {"a": 1, "b": 3, "c": 4}
    
    def test_merge_configs_nested(self):
        """Test nested config merging."""
        base = {
            "section1": {
                "key1": "value1",
                "key2": "value2"
            }
        }
        override = {
            "section1": {
                "key2": "new_value2",
                "key3": "value3"
            }
        }
        
        result = merge_configs(base, override)
        
        assert result == {
            "section1": {
                "key1": "value1",
                "key2": "new_value2",
                "key3": "value3"
            }
        }
    
    @patch("builtins.open", new_callable=mock_open, read_data="openshift:\\n  namespace: test\\nmonitoring: {}")
    def test_load_config_from_file(self, mock_file):
        """Test loading config from file."""
        config = load_config("test.yaml")
        
        assert config["openshift"]["namespace"] == "test"
        assert "monitoring" in config
        mock_file.assert_called_once_with("test.yaml", "r")
    
    def test_load_config_file_not_found(self):
        """Test error when config file not found."""
        with pytest.raises(ConfigurationError, match="Configuration file not found"):
            load_config("/nonexistent/config.yaml")