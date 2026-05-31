# PR 10: TTL Inventory Cache — Design Spec

**Date:** 2026-05-31  
**Plan item:** [Remediation plan PR 10](../plans/2026-05-28-repo-health-remediation.md)  
**Milestone:** M4 — Scale and Product Depth (Stage 2 foundation)

## Problem statement and scope

Every orphan scan re-lists all K8s PVs/PVCs/snapshots and TrueNAS datasets/snapshots. On large clusters this is O(n) API traffic per scan interval with no reuse between consecutive monitor scans or rapid API orphan requests.

**Known duplicate today (Go):** `detectOrphanedPVCs` calls both `ListUnboundPersistentVolumeClaims` and `ListPersistentVolumeClaims`, but unbound already re-lists all PVCs internally in `go/pkg/k8s/client.go`.

Python already lists PVCs once in `find_orphaned_resources()` — no dedupe needed there.

PR 9 delivered phase histograms and scan timing needed to validate cache effectiveness.

**In scope:**

- In-process, thread-safe TTL cache for list inventory operations (Go + Python)
- Config keys wired: `enabled`, `ttl`, `max_size`
- Cache hit/miss metrics (Go; optional Python when `metrics.enabled`)
- Go PVC dedupe: single list + in-memory unbound filter in detector
- Tests: hit, miss, expiry, disabled cache passthrough
- Docs: `config-compatibility.md`, `config.go.example`

**Out of scope:**

- Redis/shared cache (ARCHITECTURE target-state only)
- Watch/incremental reconcile (PR 11)
- Performance budget enforcement (PR 12)
- Caching mutating operations or per-object reads

## Current vs target behavior

| Surface | Current | Target |
|---------|---------|--------|
| Go monitor/API list calls | Every scan hits K8s + TrueNAS APIs | Cached within TTL; zero upstream calls on hit |
| Go PVC orphan detection | Two PVC list calls per scan | One PVC list; unbound filtered in memory |
| Go config cache keys | Not present | `performance.cache.enabled/ttl/max_size` wired |
| Python list calls | Every scan hits APIs | Cached when `performance.cache.enabled` |
| Python cache config | Defaults exist but unused | Wired from config |
| Metrics | List phase histograms only | Add cache hit/miss counters by operation |

## Technical approach

### Cache keys

| Operation | Cache key | Notes |
|-----------|-----------|-------|
| K8s democratic PVs | `k8s_pvs` | cluster-scoped |
| K8s PVCs | `k8s_pvcs:<namespace\|*>` | empty namespace = all namespaces |
| K8s VolumeSnapshots | `k8s_snapshots:<namespace\|*>` | |
| TrueNAS volumes | `truenas_datasets` | |
| TrueNAS snapshots | `truenas_snapshots` | |

Default TTL: **5m**. Default **enabled: true**.

### Go architecture

1. New package `go/pkg/inventorycache/` — generic TTL store with max size and injectable clock for tests.
2. `CachedK8sClient` / `CachedTrueNASClient` decorators implementing existing `k8s.Client` and `truenas.Client` interfaces; cache only list inventory methods.
3. Wire in `go/cmd/monitor/main.go` and `go/cmd/api-server/main.go` after `config.Load`.
4. Refactor `detectOrphanedPVCs` to list once and filter Pending locally.
5. Add Prometheus counters `truenas_monitor_inventory_cache_hits_total` and `_misses_total` with fixed `operation` label.

### Python architecture

1. New `python/truenas_storage_monitor/inventory_cache.py` — TTL dict with max size.
2. Wrap list methods on k8s/truenas clients or inject cache at client construction.
3. Wire `performance.cache.*` from config (properties on `Config`).

### Alternatives considered

| Approach | Verdict |
|----------|---------|
| Cache only inside detector | Rejected — API direct list handlers would miss cache |
| Cache inside raw k8s client | Rejected — couples client to cache lifecycle |
| External Redis | Out of scope for PR 10 |

## Risk, failure modes, and mitigations

| Risk | Mitigation |
|------|------------|
| Stale inventory within TTL | Document TTL tradeoff; PR 11 watch mode addresses freshness |
| Memory growth from large lists | `max_size` cap with LRU-style eviction |
| Cache key collision across namespaces | Namespace included in PVC/snapshot keys |
| Disabled cache misconfiguration | Passthrough to underlying client when `enabled: false` |
| Thread safety under concurrent API requests | Mutex-protected cache store |

## Test strategy

```bash
cd go && go test ./pkg/inventorycache/... ./pkg/orphan/... ./pkg/k8s/... ./pkg/metrics/... -v
cd python && pytest tests/unit/test_inventory_cache.py tests/unit/test_monitor.py -v
make go-test && make python-test
make go-lint && make python-lint
```

**Go tests:**

- TTL cache hit/miss/expiry/max_size with fake clock
- Caching client wrapper reduces upstream call count (2 scans within TTL → 1 call)
- Detector PVC test proves single `ListPersistentVolumeClaims` invocation
- Exporter observes hit/miss counters

**Python tests:**

- Cache hit/miss/expiry/disabled unit tests
- Monitor scan uses cache when enabled

## Rollout and backout

**Rollout:** Deploy with cache enabled by default (5m TTL). Metrics show hit ratio on `/metrics`.

**Backout:** Set `performance.cache.enabled: false` or revert PR; restores direct list behavior with no schema migration.
