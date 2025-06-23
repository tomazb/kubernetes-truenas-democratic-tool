"""Configuration management for TrueNAS Storage Monitor."""

import os
from pathlib import Path
from typing import Any, Dict, Optional

import yaml

from .exceptions import ConfigurationError


def load_config(config_path: Optional[str] = None) -> Dict[str, Any]:
    """Load configuration from file.
    
    Args:
        config_path: Path to configuration file. If not provided, will search
                    in default locations.
    
    Returns:
        Configuration dictionary
        
    Raises:
        ConfigurationError: If configuration cannot be loaded or is invalid
    """
    if config_path:
        path = Path(config_path)
        if not path.exists():
            raise ConfigurationError(f"Configuration file not found: {config_path}")
    else:
        # Search in default locations
        search_paths = [
            Path("config.yaml"),
            Path.home() / ".config" / "truenas-monitor" / "config.yaml",
            Path("/etc/truenas-monitor/config.yaml"),
        ]
        
        for path in search_paths:
            if path.exists():
                config_path = str(path)
                break
        else:
            # No config file found, use defaults
            return get_default_config()
    
    try:
        with open(config_path, "r") as f:
            config = yaml.safe_load(f)
    except Exception as e:
        raise ConfigurationError(f"Failed to load configuration: {e}")
    
    # Expand environment variables
    config = expand_env_vars(config)
    
    # Validate configuration
    validate_config(config)
    
    return config


def get_default_config() -> Dict[str, Any]:
    """Get default configuration."""
    return {
        "openshift": {
            "namespace": "democratic-csi",
        },
        "monitoring": {
            "orphan_check_interval": "1h",
            "orphan_threshold": "24h",
            "snapshot": {
                "max_age": "30d",
                "max_count": 50,
                "check_interval": "6h",
            },
            "storage": {
                "pool_warning_threshold": 80,
                "pool_critical_threshold": 90,
                "volume_warning_threshold": 85,
                "volume_critical_threshold": 95,
                "max_overcommit_ratio": 2.0,
            },
        },
        "alerts": {
            "enabled": False,
        },
        "reporting": {
            "output_dir": "/tmp/truenas-monitor/reports",
            "formats": ["html"],
        },
        "api": {
            "listen": "0.0.0.0:8080",
        },
        "metrics": {
            "enabled": True,
            "port": 9090,
            "path": "/metrics",
        },
        "logging": {
            "level": "info",
            "format": "json",
            "output": "stdout",
        },
        "performance": {
            "workers": 10,
            "batch_size": 100,
            "cache": {
                "enabled": True,
                "ttl": "5m",
                "max_size": 1000,
            },
        },
    }


def expand_env_vars(config: Any) -> Any:
    """Recursively expand environment variables in configuration."""
    if isinstance(config, dict):
        return {k: expand_env_vars(v) for k, v in config.items()}
    elif isinstance(config, list):
        return [expand_env_vars(item) for item in config]
    elif isinstance(config, str):
        # Expand ${VAR} or $VAR format
        return os.path.expandvars(config)
    else:
        return config


def validate_config(config: Dict[str, Any]) -> None:
    """Validate configuration.
    
    Args:
        config: Configuration dictionary
        
    Raises:
        ConfigurationError: If configuration is invalid
    """
    # Check required sections
    required_sections = ["openshift", "monitoring"]
    for section in required_sections:
        if section not in config:
            raise ConfigurationError(f"Missing required configuration section: {section}")
    
    # Validate TrueNAS configuration if present
    if "truenas" in config:
        truenas = config["truenas"]
        if "url" not in truenas:
            raise ConfigurationError("TrueNAS URL is required")
        
        # Check for authentication
        has_password = "username" in truenas and "password" in truenas
        has_api_key = "api_key" in truenas
        
        if not has_password and not has_api_key:
            raise ConfigurationError(
                "TrueNAS authentication required: provide username/password or api_key"
            )
    
    # Validate thresholds
    monitoring = config.get("monitoring", {})
    storage = monitoring.get("storage", {})
    
    warning = storage.get("pool_warning_threshold", 80)
    critical = storage.get("pool_critical_threshold", 90)
    
    if warning >= critical:
        raise ConfigurationError(
            "Pool warning threshold must be less than critical threshold"
        )
    
    if not 0 < warning <= 100 or not 0 < critical <= 100:
        raise ConfigurationError("Thresholds must be between 0 and 100")


def merge_configs(base: Dict[str, Any], override: Dict[str, Any]) -> Dict[str, Any]:
    """Deep merge two configuration dictionaries.
    
    Args:
        base: Base configuration
        override: Configuration to override with
        
    Returns:
        Merged configuration
    """
    result = base.copy()
    
    for key, value in override.items():
        if key in result and isinstance(result[key], dict) and isinstance(value, dict):
            result[key] = merge_configs(result[key], value)
        else:
            result[key] = value
    
    return result