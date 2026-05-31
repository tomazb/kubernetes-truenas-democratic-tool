"""Performance observability helpers for scan operations."""

from __future__ import annotations

import logging
import time
from contextlib import contextmanager
from dataclasses import dataclass, field
from typing import Dict, Generator, Optional, Protocol


class ScanMetricsProtocol(Protocol):
    """Protocol for optional Prometheus scan metrics."""

    def observe_scan(self, duration: float) -> None: ...

    def observe_list_phase(self, phase: str, duration: float) -> None: ...


logger = logging.getLogger(__name__)


@dataclass
class ScanObservability:
    """Collects scan timing and optionally exports Prometheus metrics."""

    metrics_enabled: bool = False
    phase_timings: Dict[str, float] = field(default_factory=dict)
    _scan_start: Optional[float] = field(default=None, init=False, repr=False)
    _metrics: Optional[ScanMetricsProtocol] = field(default=None, init=False, repr=False)

    def begin_scan(self) -> None:
        """Mark the start of a full orphan scan."""
        self.phase_timings.clear()
        self._scan_start = time.perf_counter()
        if self.metrics_enabled:
            self._metrics = _get_metrics_registry()

    def finish_scan(self) -> float:
        """Return total scan duration in seconds."""
        if self._scan_start is None:
            return 0.0
        duration = time.perf_counter() - self._scan_start
        if self._metrics is not None:
            self._metrics.observe_scan(duration)
            for phase, phase_duration in self.phase_timings.items():
                self._metrics.observe_list_phase(phase, phase_duration)
        return duration

    @contextmanager
    def phase(self, name: str) -> Generator[None, None, None]:
        """Time a named list or detection phase."""
        start = time.perf_counter()
        try:
            yield
        finally:
            elapsed = time.perf_counter() - start
            self.phase_timings[name] = elapsed
            logger.info("Scan phase completed", extra={"phase": name, "duration_seconds": elapsed})


def _get_metrics_registry() -> ScanMetricsProtocol:
    """Lazy-load optional Prometheus metrics."""
    from .prometheus_metrics import ScanMetrics

    return ScanMetrics()
