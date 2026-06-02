"""CLI tests for watch command."""

from datetime import timedelta
from unittest.mock import MagicMock, patch

from click.testing import CliRunner

from truenas_storage_monitor.cli import cli


@patch("truenas_storage_monitor.cli.load_config")
def test_watch_rejects_poll_mode_without_override(mock_load_config):
    runner = CliRunner()
    config = MagicMock()
    config.reconcile_mode = "poll"
    mock_load_config.return_value = config

    result = runner.invoke(cli, ["watch"])

    assert result.exit_code == 1
    assert "reconcile_mode=watch" in result.output


@patch("truenas_storage_monitor.cli.Monitor")
@patch("truenas_storage_monitor.cli.WatchReconciler")
@patch("truenas_storage_monitor.cli.load_config")
def test_watch_accepts_mode_override(mock_load_config, mock_reconciler, _mock_monitor):
    runner = CliRunner()
    config = MagicMock()
    config.reconcile_mode = "poll"
    config.debounce = timedelta(seconds=30)
    config.truenas_poll_interval = timedelta(minutes=5)
    mock_load_config.return_value = config
    mock_reconciler.return_value.run_until_signal.return_value = None

    result = runner.invoke(cli, ["watch", "--mode", "watch"])

    assert result.exit_code == 0
    mock_reconciler.assert_called_once()
