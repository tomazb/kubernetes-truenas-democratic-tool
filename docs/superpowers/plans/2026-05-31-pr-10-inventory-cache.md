# PR 10: TTL Inventory Cache — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add in-process TTL cache for expensive K8s/TrueNAS list operations and eliminate duplicate Go PVC listing.

**Architecture:** Generic TTL store in `go/pkg/inventorycache/` with caching client decorators; Python parity via `inventory_cache.py`; config wired in both stacks.

**Tech Stack:** Go 1.24+, Python 3.10+, Prometheus client libraries.

**Spec:** [2026-05-31-pr-10-inventory-cache-design.md](../specs/2026-05-31-pr-10-inventory-cache-design.md)

---

## Task 1: Go TTL cache package (TDD)

**Files:** `go/pkg/inventorycache/cache.go`, `cache_test.go`

- [ ] Write failing tests: hit, miss, expiry, max_size eviction, disabled passthrough
- [ ] Implement thread-safe TTL cache with injectable `now func() time.Time`
- [ ] Run `go test ./pkg/inventorycache/... -v`

## Task 2: Go caching client wrappers

**Files:** `go/pkg/inventorycache/cached_k8s.go`, `cached_truenas.go`, `*_test.go`

- [ ] Wrap list methods only; passthrough for health/RBAC/other ops
- [ ] Cache keys per spec (`k8s_pvs`, `k8s_pvcs:<ns>`, etc.)
- [ ] Test: two calls within TTL → one upstream invocation

## Task 3: Go config wiring

**Files:** `go/pkg/config/config.go`, `config.go.example`, defaults

- [ ] Add `PerformanceConfig` with `CacheConfig` (`enabled`, `ttl`, `max_size`)
- [ ] Defaults: enabled=true, ttl=5m, max_size=1000
- [ ] Config tests for parsing

## Task 4: Go PVC dedupe

**Files:** `go/pkg/orphan/detector.go`, `detector_test.go`

- [ ] Refactor `detectOrphanedPVCs` to single `ListPersistentVolumeClaims` + local Pending filter
- [ ] Test: mock client call count = 1 for PVC detection path

## Task 5: Go metrics

**Files:** `go/pkg/metrics/exporter.go`, `exporter_test.go`

- [ ] Add `truenas_monitor_inventory_cache_hits_total` and `_misses_total` with `operation` label
- [ ] Wire cache stats callback from inventorycache wrappers

## Task 6: Go integration

**Files:** `go/cmd/monitor/main.go`, `go/cmd/api-server/main.go`

- [ ] Wrap k8s/truenas clients with caching decorators when cache enabled
- [ ] Pass metrics exporter for hit/miss recording

## Task 7: Python inventory cache

**Files:** `python/truenas_storage_monitor/inventory_cache.py`, `config.py`, `k8s_client.py`, `truenas_client.py`

- [ ] TTL cache module with max_size
- [ ] Config properties: `cache_enabled`, `cache_ttl`, `cache_max_size`
- [ ] Wrap list methods on clients

## Task 8: Python tests

**Files:** `python/tests/unit/test_inventory_cache.py`

- [ ] hit/miss/expiry/disabled tests
- [ ] Monitor integration smoke test

## Task 9: Docs

**Files:** `docs/config-compatibility.md`, `config.go.example`, `config.yaml.example`

- [ ] Document wired cache keys for Go and Python
- [ ] Note in-process L1 only; Redis remains planned

## Task 10: Verification

```bash
cd go && go test ./pkg/inventorycache/... ./pkg/orphan/... ./pkg/config/... ./pkg/metrics/... -v
cd python && pytest tests/unit/test_inventory_cache.py tests/unit/test_monitor.py -v
make go-test && make python-test
make go-lint && make python-lint
```

- [ ] Code review before commit and push
- [ ] Update remediation plan with PR URL
