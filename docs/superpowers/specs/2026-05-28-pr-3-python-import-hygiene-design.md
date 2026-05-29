# PR 3: Python Import and Packaging Hygiene — Design Spec

**Date:** 2026-05-28  
**Plan item:** [Remediation plan PR 3](../plans/2026-05-28-repo-health-remediation.md)  
**Milestone:** M1 — Baseline Safety

## Problem statement and scope

The package `__init__.py` re-exports `cli_main` from `cli.py`, which eagerly imports Click and Rich. Any `import truenas_storage_monitor` (library use, test collection, or embedding) pulls in the full CLI dependency graph even when only `Monitor`, `Config`, or clients are needed.

Additionally, all runtime and CLI-only dependencies are listed in `[project] dependencies`, so `pip install .` installs reporting/plotting stacks that the library path does not require.

**In scope:**

- Remove CLI import from `truenas_storage_monitor/__init__.py`; keep library public API only
- Keep console script `truenas-monitor` → `truenas_storage_monitor.cli:main`
- Split `pyproject.toml`: core `dependencies` vs `[project.optional-dependencies] cli`
- Align CI/Makefile dev install with `pip install -e ".[dev,cli]"`
- Import hygiene unit tests (in-process + minimal venv install)
- Update remediation plan tracking

**Out of scope:**

- Analyzer/report implementation (pandas/plotly feature work)
- README overhaul (PR-8)
- Raising coverage gate to 90% (backlog BL-20260528-python-coverage-gate)
- Go API/detector placeholders (PR-4/5)

## Current behavior vs target behavior

| Aspect | Current | Target |
|--------|---------|--------|
| `import truenas_storage_monitor` | Loads `cli` → click/rich | Does not load `truenas_storage_monitor.cli` |
| `cli_main` in `__all__` | Exported at package root | Removed; use `from truenas_storage_monitor.cli import main` |
| `pip install .` | Installs full dependency set | Installs **core** (incl. `click`/`rich` for console script) |
| `pip install ".[cli]"` | N/A | Adds reporting CLI extras; `truenas-monitor` works |
| CI/dev install | `requirements.txt` monolith | `pip install -e ".[dev,cli]"` (requirements.txt documents extras) |

## Technical approach

1. Remove `from .cli import main as cli_main` and `cli_main` from `__all__` in `__init__.py`.
2. Move Click, Rich, and reporting-related packages to `[project.optional-dependencies] cli`.
3. Keep core deps: `kubernetes`, `requests`, `pyyaml`, `python-dateutil`.
4. Remove bogus `asyncio>=3.4.3` PyPI package (stdlib on 3.10+).
5. Update `requirements.txt` to reference pyproject extras; keep full list for convenience.
6. Add `tests/unit/test_import_hygiene.py` for import and minimal-install smoke checks.
7. Update `.github/workflows/ci.yml` and `Makefile` `python-deps` target.

**Alternatives considered:**

| Alternative | Decision |
|-------------|----------|
| Lazy `cli_main` re-export in `__init__` | Rejected — no in-repo callers; keeps API muddy |
| Keep all deps in core, only fix `__init__` | Rejected — does not meet “without CLI extras” acceptance |
| Generated requirements from pip-compile | Deferred — hand-maintained lists sufficient for now |

## Risk, failure modes, and mitigations

| Risk | Mitigation |
|------|------------|
| External code `from truenas_storage_monitor import cli_main` | Document breaking change; grep shows no in-repo usage |
| `truenas-monitor` missing after `pip install .` | Document `pip install ".[cli]"`; verify in PR notes |
| CI breaks on install path change | Update ci.yml and Makefile together |
| Minimal-install subprocess test slow/flaky | Keep test; use tmp venv with `-e` core-only install |

## Test strategy

| Test | Purpose |
|------|---------|
| `test_package_init_does_not_import_cli_module` | `cli` not in `sys.modules` after package import |
| `test_cli_main_not_in_package_all` | `cli_main` removed from `__all__` |
| `test_core_public_symbols_importable` | `Monitor`, `Config`, `load_config` at package root |
| `test_minimal_install_imports_package` | Editable core install imports library in clean venv |

**Validation commands:**

```bash
cd python && pytest tests/unit/test_import_hygiene.py -v
cd python && pytest tests/ -v --cov=truenas_storage_monitor --cov-fail-under=70
make python-lint
cd python && bandit -r truenas_storage_monitor/
pip install -e ".[cli]" && truenas-monitor --help
pip install -e . && python -c "import truenas_storage_monitor"
```

## Rollout and backout

**Rollout:** Library consumers install core package; operators/CLI users install with `[cli]` extra or full dev requirements.

**Backout:** Revert PR; restore monolithic dependencies and `cli_main` in `__init__` if needed.
