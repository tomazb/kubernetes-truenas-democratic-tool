# PR 9: Config Wiring and Performance Observability — Design Spec

**Date:** 2026-05-30  
**Plan item:** [Remediation plan PR 9](../plans/2026-05-28-repo-health-remediation.md)  
**Milestone:** M4 — Scale and Product Depth (Stage 2 foundation)

## Problem statement and scope

Operators configure orphan and snapshot retention thresholds in YAML, but runtime code ignores them:

- Go monitor hardcodes `24h` / `30d` in `monitor.NewService`.
- Go API hardcodes the same in `api.NewServer`; only `?age_threshold=` overrides age, not snapshot retention.
- Python `Monitor.find_orphaned_resources()` defaults to `age_threshold_hours=24` and never reads `monitoring.orphan_threshold` or `monitoring.snapshot.max_age`.
- Python returns `scan_duration: 0` (TODO); Go exports only a last-scan gauge with no distribution.

Without wired config and observability, Stage 2 cache (PR 10) and watch (PR 11) work cannot be measured or validated.

**In scope:**

- Wire Go monitor and API to `monitor.orphan_threshold` and `monitor.snapshot_retention` from YAML.
- Wire Python monitor to `monitoring.orphan_threshold` and `monitoring.snapshot.max_age` when caller omits explicit threshold.
- Add Go Prometheus histogram for scan duration (keep existing gauge for backward compatibility).
- Add Go phase-duration metrics for list operations (`k8s_pvs`, `k8s_pvcs`, `k8s_snapshots`, `truenas_datasets`, `truenas_snapshots`).
- Implement Python scan timing (structured logs + `scan_duration` in response).
- Optional Python Prometheus metrics when `metrics.enabled` is true in config defaults.
- Docs: mark threshold keys as wired in `config-compatibility.md`; update ARCHITECTURE metrics list.

**Out of scope:**

- TTL inventory cache (PR 10).
- Watch/incremental reconcile (PR 11).
- Performance budget enforcement (PR 12).
- Aligning Python TrueNAS inventory API with Go datasets API.
- OpenTelemetry tracing.

## Current vs target behavior

| Surface | Current | Target |
|---------|---------|--------|
| Go monitor detector | Hardcoded 24h / 30d | Uses `cfg.Monitor.OrphanThreshold` / `SnapshotRetention` |
| Go API default age | Hardcoded 24h | Uses config; `?age_threshold=` still overrides |
| Go API snapshot retention | Hardcoded 30d in detector | Uses config; echoed in `/api/v1/orphans` response |
| Python monitor thresholds | Default 24h param only | Reads config when `age_threshold_hours` is None |
| Go metrics | Gauge `scan_duration_seconds` only | Gauge retained + histogram `scan_duration_seconds` + phase histogram |
| Python scan timing | `scan_duration: 0` | Real duration in seconds; INFO logs per list phase |

## Technical approach

### Config wiring (Go)

1. Extend `monitor.Config` with `OrphanThreshold` and `SnapshotRetention`.
2. Pass values from `go/cmd/monitor/main.go` after `config.Load`.
3. Extend `api.Config` with same fields; pass from `go/cmd/api-server/main.go`.
4. Replace `defaultOrphanAgeThreshold` usage in `parseAgeThreshold` with server-configured default.
5. Initialize detector with config values in both monitor and API.

### Config wiring (Python)

1. Add `parse_duration(value) -> timedelta` in `config.py` supporting `24h`, `720h`, `30d`.
2. Add `Config.orphan_threshold` and `Config.snapshot_retention` properties returning `timedelta`.
3. Change `find_orphaned_resources(..., age_threshold_hours: Optional[int] = None)` — when None, use config.
4. Use snapshot retention for TrueNAS-side orphan detection (mirror Go retention threshold for TN snapshots without K8s match).

### Observability (Go)

1. Add `truenas_monitor_scan_duration_seconds` Histogram with buckets `[0.5, 1, 2, 5, 10, 30, 60, 120]`.
2. Keep existing Gauge for last scan duration (dashboards may depend on it).
3. Add `truenas_monitor_list_duration_seconds` Histogram with label `phase` (limited set: `k8s_pvs`, `k8s_pvcs`, `k8s_snapshots`, `truenas_datasets`, `truenas_snapshots`).
4. Extend `DetectionResult` with optional `PhaseStats []PhaseStat` or record timings in detector and expose via `ScanStats`.
5. Monitor service calls new exporter methods after each scan.

### Observability (Python)

1. New `observability.py` with `ScanTimer` context manager logging phase name + duration.
2. Wrap each list call in `find_orphaned_resources`.
3. When `metrics.enabled` is true, register optional Prometheus histogram (lazy import).

### Alternatives considered

| Approach | Verdict |
|----------|---------|
| Gauge-only extension | Rejected — no p95/regression detection |
| OpenTelemetry now | Deferred to Stage 4 |
| Replace gauge with histogram | Rejected — keep gauge for compat; add histogram |

## Risk, failure modes, and mitigations

| Risk | Mitigation |
|------|------------|
| Go/Python duration parsing drift (`30d` vs `720h`) | Canonical internal representation as `timedelta`/hours; document mapping |
| Histogram label cardinality | Fixed phase label set only; no per-object labels |
| API response breaking change | Add `snapshot_retention` field; do not remove existing fields |
| Python snapshot retention semantics differ from Go | Spec: Python uses `max_age` for TN-side retention check in snapshot pass |

## Test strategy

```bash
cd go && go test ./pkg/monitor/... ./pkg/api/... ./pkg/metrics/... ./pkg/orphan/... -v
cd python && pytest tests/unit/test_monitor.py tests/unit/test_config.py tests/unit/test_observability.py -v
make go-test && make python-test
make go-lint && make python-lint
```

**Go tests:**

- Monitor service uses configured thresholds (not hardcoded).
- API default age threshold from config; query override still works.
- Exporter histogram observes scan duration.
- Config duration parsing for Python `24h`, `720h`, `30d`.

**Python tests:**

- Monitor uses config threshold when param omitted.
- Scan duration > 0 in mocked scan.
- `parse_duration` edge cases.

## Rollout and backout

**Rollout:** Deploy updated monitor/API binaries. Existing YAML configs take effect immediately. New histogram metrics appear on `/metrics`; existing gauge unchanged.

**Backout:** Revert PR; restores hardcoded thresholds and gauge-only metrics.
