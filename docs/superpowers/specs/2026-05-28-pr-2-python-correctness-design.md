# PR 2: Python Correctness and Contract Alignment — Design Spec

**Date:** 2026-05-28  
**Plan item:** [Remediation plan PR 2](../plans/2026-05-28-repo-health-remediation.md)  
**Milestone:** M1 — Baseline Safety

## Problem statement and scope

The Python `Monitor` class was written against a stale API: dict-based Kubernetes objects, `list_*` client methods, and a `config.kubernetes` section. The implemented stack uses typed dataclass clients (`K8sClient`, `TrueNASClient`), `get_*` methods, and an `openshift` config section. At runtime, monitor initialization and scans fail with `AttributeError` unless clients are fully mocked.

Additionally, the K8s client mixes naive and timezone-aware datetimes in orphan detection and watch handlers, which can raise `TypeError` when comparing timestamps from the Kubernetes API.

**In scope:**

- Config normalization (`openshift` / `kubernetes` alias) and factory methods (`k8s_config()`, `truenas_config()`)
- Monitor wiring to typed clients and real `get_*` APIs
- Shared UTC time helpers and datetime fixes in K8s client
- Thin `test_connection()` and `list_namespaces()` on `K8sClient`
- Unit test updates to catch contract drift

**Out of scope:**

- Package `__init__.py` import side effects (PR-3)
- Deploy manifest renames or README overhaul (PR-8)
- Go API/detector placeholders (PR-4/5)
- Full cross-system orphan correlation logic (future product depth)

## Current behavior vs target behavior

| Aspect | Current | Target |
|--------|---------|--------|
| Monitor init | `K8sClient(config.kubernetes)` dict | `K8sClient(config.k8s_config())` |
| TrueNAS init | `TrueNASClient(config.truenas)` dict | `TrueNASClient(config.truenas_config())` |
| K8s resource fetch | `list_persistent_volumes()` etc. | `get_persistent_volumes()` etc. |
| Config section | `kubernetes` (missing on `Config`) | `openshift` canonical; `kubernetes` alias at load |
| Datetime comparisons | Mixed naive/aware | Always UTC-aware via helpers |
| CSI health | `list_pods()` + nested status dict | `check_csi_driver_health()` |
| Unit tests | Mock non-existent APIs | Test real method names and dataclass shapes |

## Technical approach

1. Add `normalize_cluster_config()` in `load_config()` to merge `kubernetes` → `openshift`.
2. Add `Config.k8s_config()` and `Config.truenas_config()` factory methods.
3. Add `time_utils.py` with `utc_now()`, `ensure_utc()`, `parse_rfc3339()`.
4. Add `test_connection()` and `list_namespaces()` to `K8sClient`; fix datetime usage.
5. Refactor `monitor.py` to use dataclass fields from client return types.
6. Update unit tests for config factories, monitor contract, and timezone edge cases.

**Alternatives considered:**

| Alternative | Decision |
|-------------|----------|
| Add `list_*` aliases on clients only | Rejected — does not fix dict vs dataclass mismatch |
| Refactor monitor to dataclass fields | Accepted — aligns with client design |
| Rename deploy ConfigMap to `openshift` | Deferred to PR-8; load-time alias covers runtime |

## Risk, failure modes, and mitigations

| Risk | Mitigation |
|------|------------|
| Monitor output shape changes | Keep top-level result dict keys stable |
| URL parsing edge cases for TrueNAS | Table-driven tests for host/port variants |
| Backward compat for `kubernetes` key | Load-time normalization + deprecated property alias |
| Tests masked by full client patching | Partial mocks with real method names |

## Test strategy

| Test | Purpose |
|------|---------|
| `test_k8s_config_from_openshift` | Factory maps openshift → K8sConfig |
| `test_kubernetes_alias_normalized` | `kubernetes` section accepted at load |
| `test_truenas_config_url_parsing` | URL → host/port, insecure → verify_ssl |
| `test_parse_rfc3339_*` | Aware UTC parsing |
| `test_find_orphaned_pvcs_timezone` | Naive/aware creation_time handling |
| `test_monitor_initialization` | Factory wiring to typed clients |
| `test_find_orphaned_resources_success` | Real method names + dataclass fixtures |

**Validation commands:**

```bash
cd python && pytest tests/unit/test_config.py tests/unit/test_monitor.py tests/unit/test_k8s_client.py tests/unit/test_time_utils.py -v
make python-test
make python-lint
cd python && bandit -r .
```

## Rollout and backout

**Rollout:** Deploy updated Python monitor/CLI; existing configs with `openshift` or `kubernetes` sections continue to work via normalization.

**Backout:** Revert to previous release; no schema migration required.
