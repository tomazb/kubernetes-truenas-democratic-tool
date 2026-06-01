# PR 11: Watch-Based Incremental Reconcile — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add opt-in watch mode with debounced orphan reconcile (Go monitor + Python `watch` CLI), periodic TrueNAS polling in watch mode, and full-scan recovery on watch failure.

**Architecture:** New `go/pkg/reconcile/` controller with SharedInformers + debouncer + TN snapshot poller; Python `reconcile.py` + `truenas-monitor watch`; poll mode unchanged.

**Tech Stack:** Go 1.24+, client-go informers, Python 3.10+, kubernetes watch client.

**Spec:** [2026-06-01-pr-11-watch-reconcile-design.md](../specs/2026-06-01-pr-11-watch-reconcile-design.md)

**Worktree:** `.worktrees/pr-11-watch-reconcile` on branch `feature/pr-11-watch-reconcile`

---

## Task 1: Go config — reconcile mode keys (TDD)

**Files:**
- Modify: `go/pkg/config/config.go`, `config_test.go`
- Modify: `config.go.example`

- [x] **Step 1:** Add `ReconcileMode`, `Debounce`, `TruenasPollInterval` to `MonitorConfig`; defaults `poll`, `30s`, `5m`
- [x] **Step 2:** Validate `reconcile_mode` ∈ {poll, watch}; positive durations for watch-only fields
- [x] **Step 3:** `go test ./pkg/config/... -v`

## Task 2: Go debouncer package (TDD)

**Files:**
- Create: `go/pkg/reconcile/debounce.go`, `debounce_test.go`

- [x] **Step 1:** Failing tests — single fire after quiet period; reset on retrigger; no fire after cancel
- [x] **Step 2:** Implement with injectable clock
- [x] **Step 3:** `go test ./pkg/reconcile/... -run Debounce -v`

## Task 3: Go TrueNAS snapshot poller

**Files:**
- Create: `go/pkg/reconcile/truenas_poller.go`, `truenas_poller_test.go`

- [x] **Step 1:** Poller updates mutex-protected volumes/snapshots on ticker
- [x] **Step 2:** Test fake client called at interval; readers see latest snapshot
- [x] **Step 3:** `go test ./pkg/reconcile/... -run Truenas -v`

## Task 4: Go watch runner + informers

**Files:**
- Create: `go/pkg/reconcile/watch.go`, `watch_test.go`
- Modify: `go/pkg/k8s/client.go` (only if lister helpers needed)

- [x] **Step 1:** Build SharedInformerFactory for PV/PVC/VolumeSnapshot with existing label/driver filters
- [x] **Step 2:** Event handlers call `debouncer.Trigger()`; increment watch event metric
- [x] **Step 3:** Fake clientset / envtest-style unit tests for handler registration (minimal)

## Task 5: Go reconcile controller

**Files:**
- Create: `go/pkg/reconcile/controller.go`, `controller_test.go`
- Modify: `go/pkg/monitor/service.go`

- [x] **Step 1:** `Controller.Run` — poll branch delegates to existing ticker loop; watch branch starts poller + informers
- [x] **Step 2:** Debounced reconcile: invalidate inventory cache K8s keys, run detector, update metrics/last scan
- [x] **Step 3:** Watch error → full scan once → restart watch (test with injected error channel)
- [x] **Step 4:** Wire `go/cmd/monitor/main.go` from config

## Task 6: Go metrics

**Files:**
- Modify: `go/pkg/metrics/exporter.go`, tests

- [x] **Step 1:** Add gauge/counters per spec
- [x] **Step 2:** Controller sets mode gauge on start

## Task 7: Python config + validation

**Files:**
- Modify: `python/truenas_storage_monitor/config.py`, `tests/unit/test_config.py`
- Modify: `config.yaml.example`

- [x] **Step 1:** Properties `reconcile_mode`, `debounce`, `truenas_poll_interval`
- [x] **Step 2:** Validation tests mirror Go rules

## Task 8: Python WatchReconciler (TDD)

**Files:**
- Create: `python/truenas_storage_monitor/reconcile.py`, `tests/unit/test_reconcile.py`

- [x] **Step 1:** Debounce tests with mock clock/timer
- [x] **Step 2:** TN poller thread + snapshot
- [x] **Step 3:** K8s watch loops (mock streams) trigger single reconcile per burst
- [x] **Step 4:** Watch failure runs full `find_orphaned_resources` then restarts watches

## Task 9: Python CLI `watch` command

**Files:**
- Modify: `python/truenas_storage_monitor/cli.py`, `tests/unit/test_cli.py`

- [x] **Step 1:** `truenas-monitor watch` runs reconciler until SIGINT
- [x] **Step 2:** Rejects if `reconcile_mode=poll` with clear message (or honors CLI `--mode` override documented in help)

## Task 10: Docs and plan tracking

**Files:**
- Modify: `docs/config-compatibility.md`, `docs/superpowers/plans/2026-05-28-repo-health-remediation.md`

- [x] **Step 1:** Document new keys and poll vs watch semantics
- [x] **Step 2:** Link spec + PR URL when opened — https://github.com/tomazb/kubernetes-truenas-democratic-tool/pull/61

## Task 11: Verification gate

- [x] `make go-test && make go-lint && cd go && go vet ./...`
- [x] `make python-test` (106 passed, coverage 77.9%)
- [x] Python lint on touched files; project-wide flake8 has pre-existing E501 in `truenas_client.py`
- [ ] Manual note in PR: watch mode smoke on kind (optional)
