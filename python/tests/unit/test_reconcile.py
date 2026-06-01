"""Tests for watch-mode reconcile helpers."""

import threading
import time
from unittest.mock import MagicMock, patch

import pytest

from truenas_storage_monitor.reconcile import Debouncer, WatchReconciler


def test_debouncer_coalesces_events():
    calls = []

    debouncer = Debouncer(0.05, lambda: calls.append(time.time()))
    debouncer.trigger()
    debouncer.trigger()
    debouncer.trigger()
    time.sleep(0.15)
    assert len(calls) == 1


def test_debouncer_cancel_prevents_fire():
    calls = []

    debouncer = Debouncer(0.05, lambda: calls.append(1))
    debouncer.trigger()
    debouncer.cancel()
    time.sleep(0.1)
    assert calls == []


def test_watch_reconciler_debounced_scan():
    from truenas_storage_monitor.config import Config

    config = Config.__new__(Config)
    config.data = {
        "monitoring": {
            "reconcile_mode": "watch",
            "debounce": "1s",
            "truenas_poll_interval": "1h",
            "orphan_threshold": "24h",
            "snapshot": {"max_age": "30d"},
        },
        "openshift": {"namespace": "default"},
        "truenas": {
            "url": "https://tn.example.com",
            "username": "u",
            "password": "p",
        },
        "performance": {"cache": {"enabled": False}},
    }

    monitor = MagicMock()
    monitor.find_orphaned_resources.return_value = {
        "summary": {"orphaned_pvs": 0, "orphaned_pvcs": 0, "orphaned_snapshots": 0},
        "scan_duration": 0.1,
    }
    monitor.k8s_client = MagicMock()
    monitor.k8s_client._inventory_cache = None
    monitor.truenas_client = MagicMock()
    monitor.truenas_client.get_volumes.return_value = []
    monitor.truenas_client.get_snapshots.return_value = []

    reconciler = WatchReconciler(monitor, config)

    with patch.object(reconciler, "_start_k8s_watches"):
        stop_timer = threading.Timer(0.2, reconciler._stop.set)
        stop_timer.start()
        try:
            reconciler.run_until_signal()
        finally:
            stop_timer.cancel()

    assert monitor.find_orphaned_resources.call_count >= 1


def test_config_reconcile_mode_validation():
    from truenas_storage_monitor.config import Config
    from truenas_storage_monitor.exceptions import ConfigurationError

    config = Config.__new__(Config)
    config.data = {"monitoring": {"reconcile_mode": "invalid"}}
    with pytest.raises(ConfigurationError, match="reconcile_mode"):
        _ = config.reconcile_mode
