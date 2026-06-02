# PR 12: Performance Budgets and Cardinality Docs — Design Spec

**Date:** 2026-06-02  
**Plan item:** [Remediation plan PR 12](../plans/2026-05-28-repo-health-remediation.md)  
**Milestone:** M4 — Scale and Product Depth (Stage 2 foundation)

## Problem statement and scope boundaries

PRs 9-11 added observability, TTL inventory cache, and watch-driven reconcile. The repository still lacks explicit, operator-configurable performance budgets and a documented supported object-count envelope. Operators can observe metrics, but cannot consistently tell when the system is outside accepted operating bounds.

PR 12 establishes explicit budget contracts and documentation so performance regressions are visible early without introducing hard-fail operational behavior.

**In scope:**

- Add performance budget config keys for:
  - full scan duration
  - list phase p95 latency
  - process memory usage
- Validate config values at load time with safe defaults.
- Evaluate budget breaches at runtime in Go monitor paths.
- Emit breach signals as structured warnings and exporter metrics.
- Add a soft CI benchmark gate that reports budget pressure but does not fail required checks.
- Document supported cardinality envelope and benchmark assumptions in architecture docs.
- Update remediation plan PR 12 tracking fields.

**Out of scope:**

- Hard CI fail gate on budget breach.
- Automatic runtime throttling, adaptive backpressure, or circuit breaking.
- Full load-test framework for Stage 4.
- Major Python runtime budget enforcement (Python remains observability-focused in this PR).

## Current behavior vs target behavior

| Surface | Current | Target |
|---------|---------|--------|
| Performance budgets | No explicit runtime budget contract | Configurable budgets with defaults and validation |
| Runtime signal on regressions | Raw metrics only | Budget-breach warnings + budget-breach metrics |
| CI performance guardrail | No benchmark gate for PR 12 scope | Soft benchmark job reports over-budget findings without failing build |
| Cardinality support docs | Mentioned indirectly across docs | Explicit supported object-count envelope in `docs/ARCHITECTURE.md` |

## Technical approach and alternatives considered

### Runtime budget contract

Add new config under `performance.budgets`:

- `scan_duration_seconds` (float > 0)
- `list_phase_p95_seconds` (float > 0)
- `memory_rss_mb` (int > 0)

Defaults are conservative and aligned to current baseline behavior measured in existing CI/dev runs. Values are overridable via `config.yaml`.

### Runtime evaluation path

Use existing monitor/exporter integration points:

- Evaluate scan duration budget on every completed scan.
- Evaluate list phase duration histograms through a rolling-window approximation for p95 using existing phase histogram streams (without high-cardinality labels).
- Evaluate memory budget from process metrics (RSS) when available from exporter/runtime collector.

On breach:

- Log structured warning with budget name, observed value, threshold, phase (if applicable).
- Increment breach counter metric with low-cardinality labels.
- Set/refresh last-breach timestamp gauge for operational visibility.

No hard error is returned to API clients for budget breach in this PR.

### Soft CI benchmark gate

Add a non-required CI workflow/job that runs a deterministic benchmark command (or script) and uploads a budget report artifact.

- Job outcome:
  - Success with warnings in logs/artifact when over budget.
  - Does not block merge (soft gate).
- The report includes benchmark thresholds and observations (`ns_per_op`, `bytes_per_op`) plus an overall pass/warn status for the soft gate.

### Documentation update

Add a cardinality envelope section in `docs/ARCHITECTURE.md` including:

- Supported baseline envelope with explicit initial numbers grounded in PR 12 benchmark evidence (anchored to existing PRD expectation of handling 1,000 volumes with scan time under 5 minutes).
- Assumptions (cache enabled, watch mode behavior, polling fallback).
- How to interpret budget warnings and tune thresholds.

Also update config docs/examples (`config.yaml.example`, `config.go.example`, `docs/config-compatibility.md`) for new budget keys.

### Alternatives considered

| Approach | Verdict |
|----------|---------|
| Metrics-only (no runtime evaluator) | Rejected: too indirect for operators and noisy for triage |
| Hard CI fail gate immediately | Rejected for PR 12: likely flaky while baseline stabilizes |
| Enforce budgets in both Go and Python now | Rejected for scope control; Go-first contract is sufficient for this PR |

## Risks, failure modes, and mitigations

| Risk | Mitigation |
|------|------------|
| False positives due to noisy environments | Keep CI gate soft; document benchmark environment assumptions |
| Alert fatigue from repeated breach logs | Throttle/reduce duplicate warnings per budget window |
| Label-cardinality explosion in new metrics | Restrict labels to fixed budget and phase enums only |
| Misconfigured thresholds (too low) | Validate positive values and ship sane defaults |
| Memory metric unavailability on some runtime targets | Guard with capability checks and emit "not available" debug signal |

## Test strategy and validation commands

Go (required for changed behavior):

```bash
cd go && go test ./pkg/config/... ./pkg/monitor/... ./pkg/metrics/... -v
cd go && go test ./... -v
```

Docs/config verification:

```bash
rg "performance:\\s*$|budgets:|scan_duration_seconds|list_phase_p95_seconds|memory_rss_mb" config.yaml.example config.go.example docs/config-compatibility.md docs/ARCHITECTURE.md
```

CI benchmark gate validation:

```bash
# run locally if script is added
./scripts/perf-budget-benchmark.sh
```

Quality gates for touched stack:

```bash
make go-test
make go-lint
```

## Rollout and backout notes

**Rollout:**

- Merge with default budgets enabled.
- Observe breach counters/warnings for one release cycle.
- Tune defaults only after measured evidence from CI and production-like environments.

**Backout:**

- Set budget thresholds high to suppress warnings temporarily, or remove `performance.budgets` from config to use defaults.
- Revert PR 12 if budget evaluator introduces instability.

No schema/data migration is required for rollout or rollback.
