"""Custom exceptions for TrueNAS Storage Monitor."""


class TrueNASMonitorError(Exception):
    """Base exception for all TrueNAS Monitor errors."""

    pass


class ConfigurationError(TrueNASMonitorError):
    """Raised when there's a configuration problem."""

    pass


class ConnectionError(TrueNASMonitorError):
    """Raised when connection to Kubernetes or TrueNAS fails."""

    pass


class ValidationError(TrueNASMonitorError):
    """Raised when validation fails."""

    pass


class AuthenticationError(TrueNASMonitorError):
    """Raised when authentication fails."""

    pass


class ResourceNotFoundError(TrueNASMonitorError):
    """Raised when a requested resource is not found."""

    pass


class OperationTimeoutError(TrueNASMonitorError):
    """Raised when an operation times out."""

    pass