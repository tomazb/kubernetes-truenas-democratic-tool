"""TrueNAS Storage Monitor - Comprehensive monitoring for OpenShift/Kubernetes with TrueNAS."""

__version__ = "0.1.0"
__author__ = "Your Name"
__email__ = "your.email@example.com"

# Public API
from .cli import main as cli_main
from .monitor import Monitor
from .analyzer import StorageAnalyzer
from .exceptions import (
    TrueNASMonitorError,
    ConfigurationError,
    ConnectionError,
    ValidationError,
)
from .k8s_client import K8sClient, K8sConfig, OrphanedResource, ResourceType
from .truenas_client import TrueNASClient, TrueNASConfig, VolumeInfo, SnapshotInfo, PoolInfo
from .config import Config, load_config

__all__ = [
    "cli_main",
    "Monitor",
    "StorageAnalyzer",
    "TrueNASMonitorError",
    "ConfigurationError",
    "ConnectionError",
    "ValidationError",
    "K8sClient",
    "K8sConfig",
    "OrphanedResource",
    "ResourceType",
    "TrueNASClient",
    "TrueNASConfig",
    "VolumeInfo",
    "SnapshotInfo",
    "PoolInfo",
    "Config",
    "load_config",
]