"""Smoke tests for package import hygiene (PR-3)."""

import os
import subprocess
import sys
from pathlib import Path

_PYTHON_ROOT = Path(__file__).resolve().parents[2]
_SUBPROCESS_TIMEOUT = 30
_VENV_TIMEOUT = 60
_PIP_TIMEOUT = 180
_MINIMAL_IMPORT_SCRIPT = "import truenas_storage_monitor; import truenas_storage_monitor.monitor"


def _venv_scripts_dir(venv: Path) -> Path:
    return venv / ("Scripts" if os.name == "nt" else "bin")


def _run_isolated_import_check(
    script: str,
) -> subprocess.CompletedProcess[str]:
    """Run import check in a subprocess (does not pollute pytest sys.modules)."""
    return subprocess.run(
        [sys.executable, "-c", script],
        cwd=_PYTHON_ROOT,
        capture_output=True,
        text=True,
        timeout=_SUBPROCESS_TIMEOUT,
        check=False,
    )


def test_package_init_does_not_import_cli_module():
    result = _run_isolated_import_check(
        "import truenas_storage_monitor; "
        "import sys; "
        "assert 'truenas_storage_monitor.cli' not in sys.modules"
    )
    assert result.returncode == 0, result.stderr or result.stdout


def test_cli_main_not_in_package_all():
    result = _run_isolated_import_check(
        "import truenas_storage_monitor as pkg; "
        "assert 'cli_main' not in getattr(pkg, '__all__', [])"
    )
    assert result.returncode == 0, result.stderr or result.stdout


def test_core_public_symbols_importable():
    result = _run_isolated_import_check(
        "import truenas_storage_monitor as pkg; "
        "assert pkg.Monitor is not None; "
        "assert pkg.Config is not None; "
        "assert pkg.load_config is not None"
    )
    assert result.returncode == 0, result.stderr or result.stdout


def test_minimal_install_imports_package(tmp_path):
    """Core-only editable install imports library (uses pinned requirements.txt)."""
    venv = tmp_path / "venv"
    scripts = _venv_scripts_dir(venv)
    subprocess.run(
        [sys.executable, "-m", "venv", str(venv)],
        check=True,
        timeout=_VENV_TIMEOUT,
    )
    pip = scripts / "pip"
    py = scripts / "python"
    subprocess.run(
        [str(pip), "install", "-r", str(_PYTHON_ROOT / "requirements.txt")],
        check=True,
        timeout=_PIP_TIMEOUT,
    )
    subprocess.run(
        [str(pip), "install", "--no-deps", "-e", str(_PYTHON_ROOT)],
        check=True,
        timeout=_PIP_TIMEOUT,
    )
    subprocess.run(
        [str(py), "-c", _MINIMAL_IMPORT_SCRIPT],
        check=True,
        timeout=_SUBPROCESS_TIMEOUT,
        env={**os.environ, "PYTHONPATH": ""},
    )
