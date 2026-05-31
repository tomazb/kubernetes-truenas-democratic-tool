"""Configuration management for TrueNAS Storage Monitor."""

import os
import re
from datetime import timedelta
from pathlib import Path
from typing import Any, Dict, Optional, Tuple
from urllib.parse import urlparse

import yaml

from .exceptions import ConfigurationError
from .k8s_client import K8sConfig
from .truenas_client import TrueNASConfig


class Config:
    """Configuration class for TrueNAS Storage Monitor."""

    def __init__(self, config_path: Optional[str] = None):
        """Initialize configuration.

        Args:
            config_path: Path to configuration file
        """
        self.data = load_config(config_path)

    def get(self, key: str, default: Any = None) -> Any:
        """Get configuration value by key.

        Args:
            key: Configuration key (supports dot notation)
            default: Default value if key not found

        Returns:
            Configuration value
        """
        keys = key.split(".")
        value = self.data

        for k in keys:
            if isinstance(value, dict) and k in value:
                value = value[k]
            else:
                return default

        return value

    def set(self, key: str, value: Any) -> None:
        """Set configuration value by key.

        Args:
            key: Configuration key (supports dot notation)
            value: Value to set
        """
        keys = key.split(".")
        config = self.data

        for k in keys[:-1]:
            if k not in config:
                config[k] = {}
            config = config[k]

        config[keys[-1]] = value

    @property
    def truenas(self) -> Dict[str, Any]:
        """Get TrueNAS configuration."""
        return self.get("truenas", {})

    @property
    def openshift(self) -> Dict[str, Any]:
        """Get OpenShift/Kubernetes cluster configuration."""
        return self.get("openshift", {})

    @property
    def kubernetes(self) -> Dict[str, Any]:
        """Deprecated alias for :attr:`openshift`."""
        return self.openshift

    @property
    def monitoring(self) -> Dict[str, Any]:
        """Get monitoring configuration."""
        return self.get("monitoring", {})

    @property
    def logging(self) -> Dict[str, Any]:
        """Get logging configuration."""
        return self.get("logging", {})

    def k8s_config(self) -> K8sConfig:
        """Build typed Kubernetes client configuration."""
        cluster = self.openshift
        return K8sConfig(
            kubeconfig=cluster.get("kubeconfig"),
            namespace=cluster.get("namespace"),
            csi_driver=cluster.get("csi_driver", "org.democratic-csi.nfs"),
            storage_class=cluster.get("storage_class"),
            in_cluster=cluster.get("in_cluster", False),
        )

    def truenas_config(self) -> TrueNASConfig:
        """Build typed TrueNAS client configuration."""
        truenas = self.truenas
        if "url" not in truenas:
            raise ConfigurationError("TrueNAS URL is required")
        host, port, use_https = parse_truenas_url(truenas["url"])
        insecure = truenas.get("insecure", False)
        return TrueNASConfig(
            host=host,
            port=port,
            api_key=truenas.get("api_key"),
            username=truenas.get("username"),
            password=truenas.get("password"),
            verify_ssl=not insecure,
            use_https=use_https,
            timeout=parse_timeout_seconds(truenas.get("timeout", 30)),
            max_retries=truenas.get("max_retries", 3),
        )

    @property
    def orphan_threshold(self) -> timedelta:
        """Orphan age threshold from monitoring.orphan_threshold."""
        raw = self.monitoring.get("orphan_threshold", "24h")
        return parse_duration(raw)

    @property
    def snapshot_retention(self) -> timedelta:
        """Snapshot retention from monitoring.snapshot.max_age."""
        snapshot = self.monitoring.get("snapshot", {})
        raw = snapshot.get("max_age", "30d")
        return parse_duration(raw)

    @property
    def metrics_enabled(self) -> bool:
        """Whether Prometheus metrics export is enabled."""
        return bool(self.get("metrics.enabled", False))

    @property
    def performance(self) -> Dict[str, Any]:
        """Performance tuning configuration."""
        return self.get("performance", {})

    @property
    def cache_enabled(self) -> bool:
        """Whether in-process inventory cache is enabled."""
        raw = self.performance.get("cache", {}).get("enabled", True)
        if isinstance(raw, bool):
            return raw
        if isinstance(raw, str):
            normalized = raw.strip().lower()
            if normalized in {"true", "1", "yes", "on"}:
                return True
            if normalized in {"false", "0", "no", "off"}:
                return False
        raise ConfigurationError(f"Invalid cache enabled flag: {raw!r}")

    @property
    def cache_ttl(self) -> timedelta:
        """Inventory cache TTL from performance.cache.ttl."""
        raw = self.performance.get("cache", {}).get("ttl", "5m")
        return parse_duration(raw)

    @property
    def cache_max_size(self) -> int:
        """Inventory cache max entries from performance.cache.max_size."""
        raw = self.performance.get("cache", {}).get("max_size", 1000)
        if isinstance(raw, bool):
            raise ConfigurationError(f"Invalid cache max_size value: {raw!r}")
        try:
            size = int(raw)
        except (TypeError, ValueError) as exc:
            raise ConfigurationError(f"Invalid cache max_size value: {raw!r}") from exc
        if size <= 0:
            raise ConfigurationError(f"cache max_size must be > 0: {raw!r}")
        return size


def parse_truenas_url(url: str) -> Tuple[str, int, bool]:
    """Parse a TrueNAS URL into host, port, and TLS scheme flag."""
    normalized = url if "://" in url else f"https://{url}"
    try:
        parsed = urlparse(normalized)
        host = parsed.hostname
        if not host:
            raise ConfigurationError(f"Invalid TrueNAS URL: {url!r}")
        if parsed.port is not None:
            port = parsed.port
        elif parsed.scheme == "http":
            port = 80
        else:
            port = 443
        use_https = parsed.scheme != "http"
        return host, port, use_https
    except ValueError as exc:
        raise ConfigurationError(f"Invalid TrueNAS URL: {url!r}") from exc


def parse_timeout_seconds(value: Any) -> int:
    """Parse timeout values such as 30 or '30s' into seconds."""
    if isinstance(value, bool):
        raise ConfigurationError(f"Invalid timeout value: {value!r}")
    try:
        if isinstance(value, int):
            seconds = value
        elif isinstance(value, str):
            stripped = value.strip()
            seconds = int(stripped[:-1]) if stripped.endswith("s") else int(stripped)
        else:
            raise TypeError
    except (TypeError, ValueError) as exc:
        raise ConfigurationError(f"Invalid timeout value: {value!r}") from exc

    if seconds <= 0:
        raise ConfigurationError(f"Timeout must be > 0 seconds: {value!r}")
    return seconds


_DURATION_PATTERN = re.compile(r"^(\d+(?:\.\d+)?)([smhdw])?$", re.IGNORECASE)


def parse_duration(value: Any) -> timedelta:
    """Parse duration strings such as 24h, 720h, 30d, or 30m into timedelta."""
    if isinstance(value, bool):
        raise ConfigurationError(f"Invalid duration value: {value!r}")
    if isinstance(value, timedelta):
        return value
    if isinstance(value, (int, float)):
        raise ConfigurationError(
            f"Duration must include an explicit unit (e.g. 24h, 30d): {value!r}"
        )
    if not isinstance(value, str):
        raise ConfigurationError(f"Invalid duration value: {value!r}")

    stripped = value.strip().lower()
    match = _DURATION_PATTERN.match(stripped)
    if not match:
        raise ConfigurationError(f"Invalid duration format: {value!r}")

    amount = float(match.group(1))
    unit = match.group(2)
    if not unit:
        raise ConfigurationError(
            f"Duration must include an explicit unit (e.g. 24h, 30d): {value!r}"
        )
    if amount <= 0:
        raise ConfigurationError(f"Duration must be > 0: {value!r}")

    if unit == "s":
        return timedelta(seconds=amount)
    if unit == "m":
        return timedelta(minutes=amount)
    if unit == "h":
        return timedelta(hours=amount)
    if unit == "d":
        return timedelta(days=amount)
    if unit == "w":
        return timedelta(weeks=amount)

    raise ConfigurationError(f"Unsupported duration unit in: {value!r}")


def normalize_cluster_config(config: Dict[str, Any]) -> Dict[str, Any]:
    """Normalize legacy ``kubernetes`` section to canonical ``openshift``."""
    if "kubernetes" not in config:
        return config

    kubernetes = config.pop("kubernetes") or {}
    if not isinstance(kubernetes, dict):
        raise ConfigurationError("The 'kubernetes' section must be a mapping")

    if "openshift" not in config:
        config["openshift"] = kubernetes
    else:
        openshift = config["openshift"] or {}
        if not isinstance(openshift, dict):
            raise ConfigurationError("The 'openshift' section must be a mapping")
        config["openshift"] = merge_configs(kubernetes, openshift)
    return config


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
            config = yaml.safe_load(f) or {}
    except (OSError, yaml.YAMLError) as e:
        raise ConfigurationError(f"Failed to load configuration: {e}") from e

    if not isinstance(config, dict):
        raise ConfigurationError("Configuration root must be a YAML mapping/object")

    # Expand environment variables
    config = expand_env_vars(config)
    config = normalize_cluster_config(config)

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
            "enabled": False,
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
    config = normalize_cluster_config(config.copy())

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
        raise ConfigurationError("Pool warning threshold must be less than critical threshold")

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
