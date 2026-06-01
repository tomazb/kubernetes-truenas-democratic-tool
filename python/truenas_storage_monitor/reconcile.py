"""Watch-mode incremental reconcile for the Python monitor."""

from __future__ import annotations

import logging
import signal
import threading
from dataclasses import dataclass, field
from datetime import datetime
from typing import Any, Callable, List, Optional

from .config import Config
from .exceptions import TrueNASMonitorError
from .monitor import Monitor
from .time_utils import utc_now

logger = logging.getLogger(__name__)


@dataclass
class TruenasSnapshot:
    """In-memory TrueNAS inventory from background polling."""

    volumes: List[Any] = field(default_factory=list)
    snapshots: List[Any] = field(default_factory=list)
    updated_at: Optional[datetime] = None


class Debouncer:
    """Coalesce rapid events into a single delayed callback."""

    def __init__(self, delay_seconds: float, callback: Callable[[], None]) -> None:
        self._delay = delay_seconds
        self._callback = callback
        self._timer: Optional[threading.Timer] = None
        self._lock = threading.Lock()
        self._stopped = False

    def trigger(self) -> None:
        with self._lock:
            if self._stopped:
                return
            if self._timer is not None:
                self._timer.cancel()
            self._timer = threading.Timer(self._delay, self._fire)
            self._timer.daemon = True
            self._timer.start()

    def cancel(self) -> None:
        with self._lock:
            self._stopped = True
            if self._timer is not None:
                self._timer.cancel()
                self._timer = None

    def _fire(self) -> None:
        with self._lock:
            if self._stopped:
                return
        self._callback()


class WatchReconciler:
    """Runs debounced orphan detection driven by Kubernetes watches."""

    def __init__(self, monitor: Monitor, config: Config) -> None:
        self._monitor = monitor
        self._config = config
        self._stop = threading.Event()
        self._tn_snapshot = TruenasSnapshot()
        self._tn_lock = threading.Lock()
        self._watch_threads: List[threading.Thread] = []

    def run_until_signal(self) -> None:
        """Block until SIGINT/SIGTERM, running watch reconcile."""
        debouncer = Debouncer(
            self._config.debounce.total_seconds(),
            self._debounced_reconcile,
        )

        self._run_full_scan("startup")

        tn_thread = threading.Thread(
            target=self._truenas_poll_loop,
            name="truenas-poller",
            daemon=True,
        )
        tn_thread.start()

        self._start_k8s_watches(debouncer)

        previous = signal.signal(signal.SIGINT, lambda *_: self._stop.set())
        previous_term = signal.signal(signal.SIGTERM, lambda *_: self._stop.set())
        try:
            while not self._stop.wait(timeout=1):
                pass
        finally:
            signal.signal(signal.SIGINT, previous)
            signal.signal(signal.SIGTERM, previous_term)
            debouncer.cancel()
            self._stop.set()

    def _truenas_poll_loop(self) -> None:
        interval = self._config.truenas_poll_interval.total_seconds()
        while not self._stop.is_set():
            try:
                volumes = self._monitor.truenas_client.get_volumes()
                snapshots = self._monitor.truenas_client.get_snapshots()
                with self._tn_lock:
                    self._tn_snapshot = TruenasSnapshot(
                        volumes=list(volumes),
                        snapshots=list(snapshots),
                        updated_at=utc_now(),
                    )
            except Exception as exc:  # noqa: BLE001 - log and continue polling
                logger.warning("TrueNAS poll failed: %s", exc)
            if self._stop.wait(interval):
                break

    def _start_k8s_watches(self, debouncer: Debouncer) -> None:
        k8s = self._monitor.k8s_client
        namespace = k8s.config.namespace

        def watch_pv() -> None:
            try:
                for _ in k8s.watch_persistent_volumes(timeout_seconds=60):
                    if self._stop.is_set():
                        break
                    debouncer.trigger()
            except Exception as exc:  # noqa: BLE001
                logger.warning("PV watch ended: %s", exc)
                self._on_watch_failure(debouncer)

        def watch_pvc() -> None:
            try:
                for _ in k8s.watch_persistent_volume_claims(
                    namespace=namespace, timeout_seconds=60
                ):
                    if self._stop.is_set():
                        break
                    debouncer.trigger()
            except Exception as exc:  # noqa: BLE001
                logger.warning("PVC watch ended: %s", exc)
                self._on_watch_failure(debouncer)

        for target, fn in (("pv", watch_pv), ("pvc", watch_pvc)):
            thread = threading.Thread(target=fn, name=f"watch-{target}", daemon=True)
            thread.start()
            self._watch_threads.append(thread)

    def _on_watch_failure(self, debouncer: Debouncer) -> None:
        if self._stop.is_set():
            return
        self._run_full_scan("watch_failure")
        debouncer.trigger()

    def _debounced_reconcile(self) -> None:
        if self._stop.is_set():
            return
        self._invalidate_k8s_cache()
        self._run_reconcile(trigger="debounce")

    def _run_full_scan(self, trigger: str) -> None:
        self._invalidate_all_cache()
        self._run_reconcile(trigger=trigger)

    def _run_reconcile(self, trigger: str) -> None:
        logger.info("Running orphan reconcile (trigger=%s)", trigger)
        try:
            result = self._monitor.find_orphaned_resources()
            summary = result.get("summary", {})
            logger.info(
                "Reconcile complete: pvs=%s pvcs=%s snapshots=%s duration=%ss",
                summary.get("orphaned_pvs", 0),
                summary.get("orphaned_pvcs", 0),
                summary.get("orphaned_snapshots", 0),
                result.get("scan_duration", 0),
            )
        except TrueNASMonitorError as exc:
            logger.error("Reconcile failed: %s", exc)

    def _invalidate_k8s_cache(self) -> None:
        cache = getattr(self._monitor.k8s_client, "_inventory_cache", None)
        if cache is not None:
            cache.invalidate_prefix("k8s_")

    def _invalidate_all_cache(self) -> None:
        for client in (self._monitor.k8s_client, self._monitor.truenas_client):
            cache = getattr(client, "_inventory_cache", None)
            if cache is not None:
                cache.clear()
