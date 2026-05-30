"""Optional Prometheus metrics for Python monitor scans."""

from prometheus_client import Histogram

_SCAN_DURATION = Histogram(
    "truenas_python_scan_duration_seconds",
    "Distribution of Python orphan scan durations in seconds",
    buckets=(0.5, 1, 2, 5, 10, 30, 60, 120),
)

_LIST_DURATION = Histogram(
    "truenas_python_list_duration_seconds",
    "Duration of inventory list operations during Python orphan detection",
    labelnames=("phase",),
    buckets=(0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30),
)


class ScanMetrics:
    """Records scan and list phase durations."""

    def observe_scan(self, duration: float) -> None:
        _SCAN_DURATION.observe(duration)

    def observe_list_phase(self, phase: str, duration: float) -> None:
        _LIST_DURATION.labels(phase=phase).observe(duration)
