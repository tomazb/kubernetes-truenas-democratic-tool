# PR 9: Config Wiring and Performance Observability — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Wire YAML orphan/snapshot thresholds into Go and Python runtimes and add performance observability (histograms, scan timing).

**Architecture:** Extend existing config structs and detector/monitor wiring; add Prometheus histograms alongside existing gauge; Python uses `observability.py` for phase timing.

**Tech Stack:** Go 1.24+, Python 3.10+, Prometheus client libraries.

**Spec:** [2026-05-30-pr-9-config-perf-observability-design.md](../specs/2026-05-30-pr-9-config-perf-observability-design.md)

---

### Task 1: Go config wiring (monitor + API)

**Files:** `go/pkg/monitor/service.go`, `go/cmd/monitor/main.go`, `go/pkg/api/server.go`, `go/cmd/api-server/main.go`

- [ ] Extend `monitor.Config` with threshold fields; pass to `orphan.NewDetector`
- [ ] Extend `api.Config`; store defaults on `Server`; use in `parseAgeThreshold`
- [ ] Wire from `config.Load` in both mains
- [ ] Add `snapshot_retention` to orphan API JSON response

### Task 2: Go metrics histograms

**Files:** `go/pkg/metrics/exporter.go`, `go/pkg/monitor/service.go`, `go/pkg/orphan/detector.go`

- [ ] Add scan duration histogram + list phase histogram
- [ ] Add `PhaseTimings` to `DetectionResult`; time list calls in detector
- [ ] Monitor `updateMetrics` observes histograms

### Task 3: Go tests

**Files:** `go/pkg/monitor/service_test.go`, `go/pkg/api/server_test.go`, `go/pkg/metrics/exporter_test.go`

- [ ] Test configured thresholds used by monitor/API
- [ ] Test histogram observation

### Task 4: Python config + observability

**Files:** `python/truenas_storage_monitor/config.py`, `observability.py`, `monitor.py`

- [ ] `parse_duration()` for `24h`, `720h`, `30d`
- [ ] Config properties for thresholds
- [ ] `ScanTimer` context manager; wire monitor lists

### Task 5: Python tests + docs

**Files:** `python/tests/unit/test_config.py`, `test_monitor.py`, `test_observability.py`, docs

- [ ] Duration parsing tests; monitor config default tests; scan duration > 0
- [ ] Update `docs/config-compatibility.md`, `docs/ARCHITECTURE.md`

### Task 6: Verification

```bash
cd go && go test ./pkg/monitor/... ./pkg/api/... ./pkg/metrics/... ./pkg/orphan/... -v
cd python && pytest tests/unit/test_monitor.py tests/unit/test_config.py tests/unit/test_observability.py -v
make go-test && make python-test
```
