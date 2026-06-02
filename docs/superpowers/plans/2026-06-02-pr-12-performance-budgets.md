# PR 12: Performance Budgets and Cardinality Docs — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add configurable runtime performance budgets (scan duration, list p95 latency, memory), emit budget-breach signals, introduce a soft CI benchmark gate, and document supported cardinality envelope.

**Architecture:** Extend Go config with `performance.budgets`, evaluate breaches in monitor/metrics paths with low-cardinality signals, and add a non-blocking benchmark workflow/report script. Keep runtime behavior read-only (warnings/metrics only) and update architecture/config docs to publish the initial envelope.

**Tech Stack:** Go 1.24+, Prometheus metrics, GitHub Actions, Bash script tooling, Markdown docs.

**Spec:** [2026-06-02-pr-12-performance-budgets-design.md](../specs/2026-06-02-pr-12-performance-budgets-design.md)

**Worktree:** `.worktrees/pr-12-performance-budgets` on branch `feature/pr-12-performance-budgets`

---

## Task 1: Go config — budget keys and validation (TDD)

**Files:**
- Modify: `go/pkg/config/config.go`, `go/pkg/config/config_test.go`
- Modify: `config.go.example`, `config.yaml.example`

- [x] **Step 1:** Add `BudgetConfig` under `PerformanceConfig` with `scan_duration_seconds`, `list_phase_p95_seconds`, `memory_rss_mb`.
- [x] **Step 2:** Set defaults in `DefaultConfig()` with positive baseline values.
- [x] **Step 3:** Add validation to reject zero/negative budget values with clear field-specific errors.
- [x] **Step 4:** Add/extend table-driven config tests for defaults, explicit values, and invalid values.
- [x] **Step 5:** Run `cd go && go test ./pkg/config/... -v`.

## Task 2: Go metrics — budget breach instruments

**Files:**
- Modify: `go/pkg/metrics/exporter.go`, `go/pkg/metrics/exporter_test.go`

- [x] **Step 1:** Add budget breach counter metric (`truenas_monitor_performance_budget_breaches_total`) with low-cardinality labels (`budget`, `phase`).
- [x] **Step 2:** Add budget status/last breach gauges needed for operator visibility.
- [x] **Step 3:** Add helper methods on exporter to record budget breaches and update gauges.
- [x] **Step 4:** Add unit tests validating registration and label behavior for new metrics.
- [x] **Step 5:** Run `cd go && go test ./pkg/metrics/... -v`.

## Task 3: Go monitor runtime checks — scan/list/memory budgets

**Files:**
- Modify: `go/pkg/monitor/service.go`, `go/pkg/monitor/service_test.go`
- Modify: `go/pkg/orphan/detector.go` (only if list phase data plumbing changes are needed)

- [x] **Step 1:** Evaluate scan duration against `scan_duration_seconds` and emit warning + breach metric on overrun.
- [x] **Step 2:** Compute/evaluate list-phase p95 budget from existing phase histogram inputs and emit warning + breach metric for over-budget phases.
- [x] **Step 3:** Evaluate process memory (RSS MB) against `memory_rss_mb` when available; emit warning + breach metric on overrun.
- [x] **Step 4:** Guard memory evaluation when runtime stats are unavailable (no panic/no false breach).
- [x] **Step 5:** Add/extend service tests for scan breach, list p95 breach, memory breach, and unavailable-memory path.
- [x] **Step 6:** Run `cd go && go test ./pkg/monitor/... -v`.

## Task 4: Benchmark script and soft CI gate

**Files:**
- Create or modify: `scripts/perf-budget-benchmark.sh`
- Modify or create workflow: `.github/workflows/performance-benchmark.yml` (or existing CI workflow if already hosting perf checks)

- [x] **Step 1:** Add deterministic benchmark command/script that emits scan/list/memory budget report with pass/warn statuses.
- [x] **Step 2:** Ensure script exits zero for soft-gate warning scenarios and non-zero only for script/runtime errors.
- [x] **Step 3:** Add CI workflow/job that runs benchmark script, uploads artifact, and is non-required (soft gate).
- [x] **Step 4:** Add workflow test notes/comments so maintainers understand soft-gate semantics.

## Task 5: Documentation and plan tracking

**Files:**
- Modify: `docs/ARCHITECTURE.md`
- Modify: `docs/config-compatibility.md`
- Modify: `docs/superpowers/plans/2026-05-28-repo-health-remediation.md`

- [x] **Step 1:** Document `performance.budgets` keys and behavior in config compatibility docs.
- [x] **Step 2:** Add cardinality envelope section in architecture with explicit initial numbers backed by benchmark evidence.
- [x] **Step 3:** Document assumptions (cache enabled, watch/poll behavior, environment caveats) and tuning guidance.
- [x] **Step 4:** Update remediation plan PR 12 section status to `In Progress`, link spec/plan, and fill PR URL when opened.

## Task 6: Verification gate

- [x] `cd go && go test ./pkg/config/... ./pkg/metrics/... ./pkg/monitor/... -v`
- [x] `cd go && go test ./... -v`
- [x] `make go-test && make go-lint && cd go && go vet ./...`
- [x] `./scripts/perf-budget-benchmark.sh`
- [x] `rg "performance:\\s*$|budgets:|scan_duration_seconds|list_phase_p95_seconds|memory_rss_mb" config.yaml.example config.go.example docs/config-compatibility.md docs/ARCHITECTURE.md`
- [ ] Manual pre-commit review completed and recorded in PR notes
- [ ] Manual pre-push review completed and recorded in PR notes
