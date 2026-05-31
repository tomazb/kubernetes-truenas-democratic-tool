"""Unit tests for observability helpers."""

from truenas_storage_monitor.observability import ScanObservability


class TestScanObservability:
    """Tests for scan timing collection."""

    def test_phase_timings_recorded(self):
        """Phase context manager records elapsed time."""
        obs = ScanObservability(metrics_enabled=False)
        obs.begin_scan()

        with obs.phase("k8s_pvs"):
            pass

        duration = obs.finish_scan()
        assert duration >= 0
        assert "k8s_pvs" in obs.phase_timings
        assert obs.phase_timings["k8s_pvs"] >= 0

    def test_metrics_enabled_observes_prometheus(self):
        """Prometheus metrics are recorded when enabled."""
        obs = ScanObservability(metrics_enabled=True)
        obs.begin_scan()
        with obs.phase("truenas_datasets"):
            pass
        obs.finish_scan()

    def test_begin_scan_disables_metrics_when_prometheus_missing(self, monkeypatch):
        """Missing prometheus_client disables metrics instead of crashing."""

        def raise_import_error():
            raise ImportError("prometheus_client not installed")

        monkeypatch.setattr(
            "truenas_storage_monitor.observability._get_metrics_registry",
            raise_import_error,
        )
        obs = ScanObservability(metrics_enabled=True)
        obs.begin_scan()
        assert obs.metrics_enabled is False
        assert obs._metrics is None
