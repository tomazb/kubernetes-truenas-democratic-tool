"""Optional Prometheus metrics for Python monitor scans."""

from typing import Any, Tuple

_scan_duration: Any = None
_list_duration: Any = None


def _histograms() -> Tuple[Any, Any]:
    global _scan_duration, _list_duration
    if _scan_duration is None:
        from prometheus_client import Histogram

        _scan_duration = Histogram(
            "truenas_python_scan_duration_seconds",
            "Distribution of Python orphan scan durations in seconds",
            buckets=(0.5, 1, 2, 5, 10, 30, 60, 120),
        )
        _list_duration = Histogram(
            "truenas_python_list_duration_seconds",
            "Duration of inventory list operations during Python orphan detection",
            labelnames=("phase",),
            buckets=(0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30),
        )
    return _scan_duration, _list_duration


class ScanMetrics:
    """Records scan and list phase durations."""

    def observe_scan(self, duration: float) -> None:
        scan, _ = _histograms()
        scan.observe(duration)

    def observe_list_phase(self, phase: str, duration: float) -> None:
        _, list_duration = _histograms()
        list_duration.labels(phase=phase).observe(duration)
