# PR 13 Design: Full Test Matrix and Agent-Executable Validation

## Problem Statement and Scope Boundaries

The repository needs a unified, executable test strategy that covers deterministic CI gates and deeper staged-system validation. Current test commands exist but are not fully organized into explicit lanes, and autonomous agents lack a single runbook for execution order, stop conditions, and artifacts.

In scope:

- Define test lanes and coverage mapping for Go and Python behavior.
- Add test orchestration targets (`test-ci-gate`, `test-staging`, `test-release-matrix`, `test-matrix`).
- Add baseline staging and regression matrix tests.
- Align CI commands with deterministic lane-1 execution.
- Add agent-facing documentation and staging runbook.

Out of scope:

- Implementing all future end-to-end staging scenarios against live infra.
- Replacing existing product behavior with new runtime functionality.
- Pinning all dependency versions for unpinned safety warnings.

## Current Behavior vs Target Behavior

Current behavior:

- Go and Python tests run, but lane semantics are implicit.
- Security tooling behavior differs across local and CI contexts.
- No single agent runbook defines strict order, retries, and evidence artifacts.
- No dedicated staging workflow for deep matrix execution.

Target behavior:

- Three explicit lanes:
  - Lane 1 CI gate (deterministic and merge-blocking)
  - Lane 2 staging validation (real environment)
  - Lane 3 release matrix (resilience/security/perf checks)
- A single orchestration target (`make test-matrix`) runs lanes in order and writes summary evidence.
- CI workflow executes lane-1 commands consistently.
- Staging workflow exists for scheduled/manual deep runs.
- Docs describe capability coverage, staging preflight, and agent execution contract.

## Technical Approach and Alternatives Considered

Chosen approach:

- Add docs under `docs/testing/` for inventory, staging runbook, and agent execution.
- Introduce lane make targets and orchestration in `Makefile`.
- Add lightweight but explicit staging/regression tests in `python/tests/staging/` and `python/tests/regression/`.
- Update CI workflow to invoke canonical make targets.
- Add separate staging matrix workflow for scheduled/manual execution.

Alternatives considered:

1. Keep existing commands and only add docs.
   - Rejected: does not enforce deterministic execution or agent-ready orchestration.
2. Move all logic into GitHub workflows only.
   - Rejected: weak local reproducibility; harder for agents and developers to run identical gates.
3. Build a custom Python test orchestrator script.
   - Rejected: redundant with Make targets and increases maintenance burden.

## Risk/Failure Modes and Mitigations

- Security scanner false positives/failing due to environment package inventory:
  - Mitigation: scope Safety scan to repository requirements files and Bandit to project package paths.
- Staging test accidental execution without environment:
  - Mitigation: enforce `TEST_STAGING=true` gate and required environment variable checks.
- Flaky or nondeterministic local runs:
  - Mitigation: deterministic flags (`-count=1`, `PYTHONHASHSEED=0`) and explicit lane boundaries.
- Tooling availability drift (e.g., `gosec` missing):
  - Mitigation: install fallback in `go-security` target.

## Test Strategy and Validation Commands

Primary validation commands:

- `make test-ci-gate`
- `make test-staging`
- `make test-release-matrix`
- `make test-matrix`
- `make security-scan`

Expected outcomes:

- Lane 1 commands pass on local/CI deterministic path.
- Lane 2 staging preflight test skips without staging env and executes when enabled.
- Lane 3 regression and benchmark checks pass with artifacts.
- `artifacts/summary.json` is produced by matrix orchestration.

## Rollout/Backout Notes

Rollout:

- Merge branch and enable staging workflow secrets in repository settings.
- Run manual `workflow_dispatch` for staging matrix first, then keep nightly schedule.

Backout:

- Revert workflow and make-target additions if operationally disruptive.
- Keep docs and inventory mapping as non-breaking references if partial rollback needed.
