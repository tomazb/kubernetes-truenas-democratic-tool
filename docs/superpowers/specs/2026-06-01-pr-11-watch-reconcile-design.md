# PR 11: Watch-Based Incremental Reconcile — Design Spec

**Date:** 2026-06-01  
**Plan item:** [Remediation plan PR 11](../plans/2026-05-28-repo-health-remediation.md)  
**Milestone:** M4 — Scale and Product Depth (Stage 2 foundation)  
**Depends on:** PR 10 (TTL inventory cache, merged)

## Problem statement and scope

PR 10 reduced repeated **list** traffic via TTL cache, but the Go monitor and Python library still drive orphan detection on a fixed **poll** loop (ticker / one-shot CLI). On active clusters, storage objects change between scan intervals; operators want fresher K8s-side detection without listing the full inventory on every watch notification.

**In scope:**

- Opt-in `watch` reconcile mode (`poll` remains default) in Go monitor and Python long-running CLI
- Go: `client-go` SharedInformers for PV, PVC, and VolumeSnapshot (democratic-csi filtered)
- Debounced reconcile: coalesce watch events; run orphan detector once per debounce window
- TrueNAS: separate periodic poll in watch mode; debounced K8s reconcile uses last TN snapshot between polls
- Watch failure: immediate full poll reconcile, then restart watch
- Invalidate or bypass PR 10 inventory cache for K8s keys on debounced reconcile (TrueNAS lists follow TN poll / full-scan rules)
- Prometheus metrics for mode, watch events, reconcile triggers, TN poll age
- Python parity: equivalent config keys, debounced reconcile, `truenas-monitor watch` command
- Docs: `config-compatibility.md`, `config.go.example`, `config.yaml.example`

**Out of scope:**

- API server watch mode (stays on-demand list + cache; no long-running informers in API process)
- TrueNAS streaming watch (not available)
- Redis / shared informer cache across replicas (ARCHITECTURE future state)
- Performance budget enforcement (PR 12)
- Horizontal scaling / leader election for multiple monitor replicas

## Decisions (brainstorming)

| Topic | Choice |
|-------|--------|
| Vertical slice | Full parity: Go monitor + Python watch CLI with aligned config |
| TrueNAS in watch mode | Separate `truenas_poll_interval`; debounced K8s reconcile uses cached TN snapshot between polls |
| Watch disconnect / resync failure | Immediate full poll reconcile, then restart watch |

## Current vs target behavior

| Surface | Current | Target |
|---------|---------|--------|
| Go monitor loop | Ticker → full `performScan` (lists + detector) | `poll`: unchanged; `watch`: informers + debounced reconcile |
| Go K8s reads on change | N/A (poll only) | Informer listers; optional cache invalidation before detector |
| Go TrueNAS reads | Every scan | Watch mode: periodic background poll + snapshot for reconcile |
| Go config | `monitor.scan_interval` only | Add `reconcile_mode`, `debounce`, `truenas_poll_interval` |
| Python CLI | One-shot `orphans` / `analyze` | Add long-running `watch` subcommand when `reconcile_mode=watch` |
| Python K8s | `watch_*` generators exist but unused by monitor | Wired into debounced reconcile loop |
| Metrics | Scan / list phase histograms (PR 9) | Add watch/reconcile counters and mode gauge |

## Technical approach

### Config (aligned cross-stack)

**Go** (`monitor` section):

```yaml
monitor:
  reconcile_mode: poll          # poll | watch
  scan_interval: 5m             # poll mode only
  debounce: 30s                 # watch mode: min gap between reconciles
  truenas_poll_interval: 5m     # watch mode: TrueNAS background refresh
  orphan_threshold: 24h
  snapshot_retention: 30d
```

**Python** (`monitoring` section):

```yaml
monitoring:
  reconcile_mode: poll
  debounce: 30s
  truenas_poll_interval: 5m
  orphan_threshold: 24h
```

Validation: reject unknown `reconcile_mode`; require `debounce` / `truenas_poll_interval` > 0 when mode is `watch`.

### Go architecture

New package `go/pkg/reconcile/`:

1. **`Controller`** — owns mode, debouncer, TN snapshot store (mutex-protected), and `Run(ctx)`.
2. **`WatchRunner`** — builds `SharedInformerFactory` with resync period 0 (rely on API resync + our failure handler). Registers PV, PVC, VolumeSnapshot informers with democratic-csi / namespace filters matching `k8s.Client` list behavior.
3. **`Debouncer`** — `Trigger()` resets timer; on fire calls `reconcileFn` once.
4. **`reconcileFn`** — invalidates K8s inventory cache keys (if cache enabled), runs existing `orphan.Detector.Detect` (lists still work via informer-backed client or explicit list — prefer lister adapter to avoid duplicate API calls where practical).
5. **`TrueNASPoller`** — goroutine ticker at `truenas_poll_interval`; updates in-memory TN volumes/snapshots snapshot used by detector wrapper or pre-list injection.
6. **Failure handler** — on informer watch error channel: log, run one full `performScan` equivalent (poll path), clear debouncer, restart informers.

Wire in `go/cmd/monitor/main.go`: if `reconcile_mode == watch`, use `reconcile.Controller` instead of ticker-only `monitorLoop`; poll mode keeps current `Service.monitorLoop`.

**Cache interaction (PR 10):** On debounced reconcile, invalidate `k8s_pvs`, `k8s_pvcs:*`, `k8s_snapshots:*` keys. TN data served from poller snapshot until next TN poll (no per-event TN list).

### Python architecture

New module `python/truenas_storage_monitor/reconcile.py`:

1. **`WatchReconciler`** — threads: TN poller, K8s watch multiplex (PV + PVC + VolumeSnapshot using existing client patterns), debounce timer (`threading.Timer`).
2. **`truenas-monitor watch`** — Click command: load config, run reconciler until SIGINT, log orphan counts / scan duration like Go.
3. Config properties: `reconcile_mode`, `debounce`, `truenas_poll_interval`.
4. One-shot CLI commands remain poll-based (unchanged behavior).

### Metrics

| Metric | Type | Labels |
|--------|------|--------|
| `truenas_monitor_reconcile_mode` | Gauge | `mode=poll\|watch` |
| `truenas_monitor_watch_events_total` | Counter | `resource=pv\|pvc\|snapshot` |
| `truenas_monitor_reconcile_triggers_total` | Counter | `trigger=debounce\|full_scan\|poll_tick` |
| `truenas_monitor_truenas_snapshot_age_seconds` | Gauge | — |

Python: same names when `metrics.enabled`.

### Alternatives considered

| Approach | Verdict |
|----------|---------|
| Watch only in Go; Python docs-only | Rejected — user chose full parity |
| Refresh TrueNAS on every debounced K8s event | Rejected — too much TN API load |
| Fall back to poll for process lifetime on watch error | Rejected — prefer immediate full scan + restart watch |
| Informers inside `k8s.Client` | Rejected — keep client list-based; reconcile package owns informers |

## Risk, failure modes, and mitigations

| Risk | Mitigation |
|------|------------|
| Informer memory for large clusters | Document supported cardinality; PR 12 budgets |
| Missed events during reconnect | Full scan on watch failure before restart |
| Stale TN snapshot between polls | Configurable `truenas_poll_interval`; metric for snapshot age |
| Debounce delays urgent orphan detection | Tunable `debounce`; document tradeoff |
| Python GIL / thread safety | Mutex around TN snapshot and reconcile entry |
| Duplicate work with TTL cache | Invalidate K8s keys on debounced reconcile |

## Test strategy

```bash
cd go && go test ./pkg/reconcile/... ./pkg/monitor/... -v
cd python && pytest tests/unit/test_reconcile.py tests/unit/test_monitor.py -v
make go-test && make python-test
make go-lint && make python-lint
```

**Go:** debouncer unit tests (fake clock); controller triggers detector once per burst; watch failure invokes full scan mock; TN poller updates snapshot.

**Python:** debounce coalescing; watch command exits on signal; config validation for modes.

**Integration (optional manual):** `reconcile_mode: watch` against kind cluster; verify metrics increment on PVC create/delete.

## Rollout and backout

**Rollout:** Deploy with `reconcile_mode: poll` (default). Enable watch per environment after validating metrics and TN poll interval.

**Backout:** Set `reconcile_mode: poll` or revert PR; no schema migration.
