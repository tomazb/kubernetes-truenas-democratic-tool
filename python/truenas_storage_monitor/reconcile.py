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

WATCH_RESTART_BACKOFF_SECONDS = 5


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

            timer_holder: List[Optional[threading.Timer]] = [None]

            def fire_fn() -> None:
                self._fire(timer_holder[0])

            timer = threading.Timer(self._delay, fire_fn)
            timer.daemon = True
            timer_holder[0] = timer
            self._timer = timer
            timer.start()

    def cancel(self) -> None:
        with self._lock:
            self._stopped = True
            if self._timer is not None:
                self._timer.cancel()
                self._timer = None

    def _fire(self, timer: Optional[threading.Timer]) -> None:
        with self._lock:
            if self._stopped or timer is None or self._timer is not timer:
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
        self._reconcile_lock = threading.Lock()
        self._force_live_tn = False
        self._watch_threads: List[threading.Thread] = []

        client = monitor.truenas_client
        self._orig_get_volumes = client.get_volumes
        self._orig_get_snapshots = client.get_snapshots
        client.get_volumes = self._cached_get_volumes  # type: ignore[method-assign]
        client.get_snapshots = self._cached_get_snapshots  # type: ignore[method-assign]

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

    def _cached_get_volumes(self) -> List[Any]:
        if self._force_live_tn:
            return self._orig_get_volumes()
        with self._tn_lock:
            if self._tn_snapshot.updated_at is not None:
                return list(self._tn_snapshot.volumes)
        return self._orig_get_volumes()

    def _cached_get_snapshots(self, dataset: Optional[str] = None) -> List[Any]:
        if dataset is not None:
            return self._orig_get_snapshots(dataset)
        if self._force_live_tn:
            return self._orig_get_snapshots()
        with self._tn_lock:
            if self._tn_snapshot.updated_at is not None:
                return list(self._tn_snapshot.snapshots)
        return self._orig_get_snapshots()

    def _truenas_poll_loop(self) -> None:
        interval = self._config.truenas_poll_interval.total_seconds()
        while not self._stop.is_set():
            try:
                volumes = self._orig_get_volumes()
                snapshots = self._orig_get_snapshots()
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

        def watch_loop(
            name: str,
            stream_factory: Callable[[], Any],
        ) -> None:
            while not self._stop.is_set():
                try:
                    for _ in stream_factory():
                        if self._stop.is_set():
                            break
                        debouncer.trigger()
                except Exception as exc:  # noqa: BLE001
                    logger.warning("%s watch ended: %s", name, exc)
                    self._on_watch_failure(debouncer)
                if self._stop.wait(WATCH_RESTART_BACKOFF_SECONDS):
                    break

        threads = [
            (
                "pv",
                lambda: k8s.watch_persistent_volumes(timeout_seconds=60),
            ),
            (
                "pvc",
                lambda: k8s.watch_persistent_volume_claims(namespace=namespace, timeout_seconds=60),
            ),
            (
                "snapshot",
                lambda: k8s.watch_volume_snapshots(namespace=namespace, timeout_seconds=60),
            ),
        ]

        for name, stream_factory in threads:
            thread = threading.Thread(
                target=watch_loop,
                args=(name, stream_factory),
                name=f"watch-{name}",
                daemon=True,
            )
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
        self._force_live_tn = True
        try:
            self._invalidate_all_cache()
            self._run_reconcile(trigger=trigger)
        finally:
            self._force_live_tn = False

    def _run_reconcile(self, trigger: str) -> None:
        with self._reconcile_lock:
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

    @staticmethod
    def _client_cache(client: Any) -> Any:
        return getattr(client, "_cache", None)

    def _invalidate_k8s_cache(self) -> None:
        cache = self._client_cache(self._monitor.k8s_client)
        if cache is not None:
            cache.invalidate_prefix("k8s_")

    def _invalidate_all_cache(self) -> None:
        for client in (self._monitor.k8s_client, self._monitor.truenas_client):
            cache = self._client_cache(client)
            if cache is not None:
                cache.clear()
