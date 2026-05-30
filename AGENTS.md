# AGENTS Playbook

This is the single source of truth for agent guidance, execution tracking, and repository standards.

## Project Purpose

Monitor and analyze OpenShift + TrueNAS Scale + democratic-csi integrations to detect orphaned resources, configuration drift, snapshot issues, and storage efficiency risks.

## Repository Mission and Context

This repository exists to close an operational visibility gap between Kubernetes/OpenShift storage objects and TrueNAS-backed reality.

In many environments, teams can see:
- Kubernetes intent (PVs, PVCs, StorageClasses, snapshots), and
- TrueNAS state (datasets, volumes, snapshots),

but they lack a reliable correlation layer across both systems. This project provides that correlation layer so teams can identify mismatches early, reduce storage incidents, and operate with safer defaults.

## Who This Is For

- Platform/SRE teams running OpenShift or Kubernetes with democratic-csi on TrueNAS.
- Storage administrators responsible for TrueNAS capacity, snapshot lifecycle, and reliability.
- DevOps/GitOps operators who need machine-readable checks and API/CLI integration points.

## What Success Looks Like

The repository should enable operators to:
- Detect orphaned resources quickly (both K8s-side and TrueNAS-side).
- Validate storage configuration and snapshot policy consistency.
- Track capacity and thin-provisioning risk before outages.
- Produce actionable outputs (CLI/API/reporting) for remediation workflows.

## Primary Use Cases

1. **Orphan Detection**
   - PV references missing in TrueNAS.
   - TrueNAS datasets/volumes not represented by active PVs.
   - Snapshot object mismatches between Kubernetes and TrueNAS.

2. **Configuration and Health Validation**
   - Storage class and CSI assumptions checked against runtime state.
   - Baseline checks for required permissions and component health.

3. **Capacity and Efficiency Analysis**
   - Allocated vs used storage visibility.
   - Thin provisioning and overcommitment risk tracking.
   - Snapshot growth trend awareness.

## Deliverables Provided by This Repo

- Go runtime services for monitoring and API access.
- Python tooling for CLI-oriented checks and analysis/reporting flows.
- Tests, CI workflows, and deployment assets for running and validating the stack.
- Documentation that should track implemented (not aspirational) behavior.

## Scope and Non-Goals

In scope:
- Cross-system correlation, validation, and analysis for OpenShift/Kubernetes + TrueNAS + democratic-csi.
- Safe, idempotent operational checks with clear failure semantics.

Out of scope (for baseline phases):
- Broad infrastructure orchestration unrelated to storage validation.
- UI-first development before core correctness and API/CLI reliability are mature.
- Shipping placeholder behavior as production-ready features.

## Current Maturity Snapshot

This snapshot is for contributor orientation and should be kept current as PRs merge.

### Implemented Now (baseline)

- Core Go and Python codebases exist for monitoring, API surfaces, and CLI-oriented workflows.
- Basic CI, tests, and deployment scaffolding exist in repository.
- Foundational storage analysis paths are present across K8s/OpenShift and TrueNAS domains.

### Partially Implemented / In Progress

- Some API/validation paths still include placeholder or not-implemented behavior.
- Test coverage quality is uneven due to interface drift and stale assumptions in parts of the suite.
- Documentation has been converging toward implemented reality but still requires ongoing alignment.

### Planned / Later Stage

- Full endpoint maturity with complete end-to-end behavior guarantees.
- Scalability upgrades (incremental/watch-driven reconciliation, smarter caching).
- Higher operational maturity (performance budgets, resilience hardening, stronger SLO-driven operations).

### Contribution Guidance by Maturity

- Prioritize correctness and security gaps before net-new features.
- When touching partial areas, either complete the vertical slice or make maturity status explicit in code/docs.
- Do not market planned capabilities as currently available behavior.

## Canonical Plan Pointers

- Active remediation plan: `docs/superpowers/plans/2026-05-28-repo-health-remediation.md`
- Canonical backlog: `docs/superpowers/backlog.md`

Use these repository-local documents as the durable source of truth (not machine-local session artifacts).

Milestone model in use:
- M1: Baseline Safety
- M2: Contract Integrity
- M3: Operational Readiness
- M4: Scale and Product Depth

## Branch and PR Policy

- Branches:
  - `main`: production-ready, protected
  - `develop`: integration branch
  - `feature/*`, `bugfix/*`, `security/*`
- All changes go through PRs (no direct commits to protected branches).
- PR expectations:
  - Minimum 1 approval
  - All required GitHub Actions CI checks green on the PR head commit (mandatory merge gate)
  - CI checks pass (no failing, pending, or skipped required jobs without documented exception)
  - Coverage does not decrease
  - Security checks pass
  - DCO sign-off when required

## Engineering Principles

### Security-First

- Secure by default.
- Principle of least privilege for all access.
- Never leak secrets in logs/errors.
- Prefer defense in depth.

### Idempotency and Safety

- Read-only by default unless explicitly operating in write mode.
- Deterministic behavior for repeated runs.
- Validate state before action.
- Retries must be safe and not create duplicate side effects.

### TDD Workflow

1. Red: write failing test
2. Green: implement minimum fix
3. Refactor: improve without breaking tests

Testing requirements:
- Unit tests first
- Test success and failure paths
- Mock external dependencies for unit scope
- Maintain strong coverage (target >=90% where practical)

## Architecture Intent (Current Direction)

- Go: monitor service + API server for performance-sensitive runtime paths.
- Python: CLI, analysis, reporting, integrations.
- Shared contracts/schemas should be kept consistent between Go and Python.

Key monitored resources:
- PV/PVC/StorageClass/VolumeAttachment
- CSI driver and node components
- VolumeSnapshot/VolumeSnapshotContent
- TrueNAS volumes/datasets/snapshots
- Capacity and thin-provisioning efficiency

## Execution and Tracking Rules

- Keep work PR-sized and reviewable.
- Update status fields and checkboxes in the active plan before and after each PR.
- Keep each PR scoped to one numbered plan item unless explicitly re-scoped.
- Add PR URL under the relevant section in the active plan.
- Mark acceptance criteria complete only after verification passes.
- If priorities change, update the plan first, then execute.

Execution order (current):
1. Security baseline (PR 1)
2. Python correctness and packaging hygiene (PR 2-3)
3. API honesty and detector fidelity (PR 4-5)
4. Reliability/performance guardrails (PR 6)
5. CI/release trust + docs accuracy (PR 7-8)
6. Later-stage roadmap (Stage 2+)

## Spec and Design Requirement (Global)

Every PR must have a dedicated design/spec document before implementation starts.

- Required location: `docs/superpowers/specs/`
- Required naming: `YYYY-MM-DD-pr-<n>-<topic>-design.md`
- Minimum required sections:
  - problem statement and scope boundaries
  - current behavior vs target behavior
  - technical approach and alternatives considered
  - risk/failure modes and mitigations
  - test strategy and validation commands
  - rollout/backout notes (if operational behavior changes)
- The PR description must link the spec.
- If implementation changes scope, update the spec first.
- Do not merge if implementation materially diverges from spec.

## Definition of Done (Per PR)

A PR is only done when all items below are true:

- Scope is limited to one plan item (or explicitly approved re-scope).
- Tests for changed behavior exist and pass locally/CI.
- All required GitHub Actions CI jobs are green on the PR head commit.
- Existing tests remain green for touched areas.
- Lint/type/security checks for touched stack pass.
- Docs/config/examples are updated for user-visible changes.
- No placeholder success behavior is introduced.
- Risk notes and rollback/mitigation are included for high-risk changes.
- Plan tracking is updated (status, acceptance checklist, PR URL).

## PR Comment and Review Thread Policy (Global)

PR comments and review threads are part of completion criteria and must be actively managed.

- Before merge, fetch and review all open PR comments/threads.
- Validate each comment technically (do not apply suggestions blindly).
- For accepted comments:
  - implement the fix,
  - add/adjust tests where relevant,
  - reply with concise change summary.
- For non-applicable comments:
  - reply with technical rationale and evidence.
- For out-of-scope comments (outside approved spec/design):
  - do not implement in the current PR,
  - reply that the item is out of scope for this PR,
  - add to backlog when sensible (with short rationale and proposed target PR/phase).
- Resolve PR threads only after:
  - code and tests reflect the agreed change, and
  - the response is posted on the PR.
- Final merge gate: no unresolved actionable review threads and all required CI checks green.

Scope rule:
- The approved PR spec/design is the source of truth for scope.
- Review feedback that changes scope requires explicit scope update (spec + plan) before implementation.

## Mandatory Pre-Commit and Pre-Push Code Review

Code review is required before both commit and push.

- Before commit:
  - run an explicit code-review pass on the working diff,
  - address critical/warning findings or document rationale for deferral in PR notes/backlog.
- Before push:
  - run a second code-review pass on the final staged/commit-ready diff.
- Preferred tool:
  - `coderabbit review --prompt-only` (or equivalent configured review tool).
- Minimum enforcement:
  - no commit or push should happen without a recorded pre-commit and pre-push review check.
  - if review tooling is unavailable, perform manual review and document that fallback in PR notes.

## Working Agreement for Agents

- Keep code, tests, and docs aligned in the same change when feasible.
- Avoid shipping placeholder behavior that looks production-ready.
- Keep docs aligned with implemented reality (not aspirational state).
- Do not introduce destructive or irreversible operations without explicit approval.

## Mandatory PR Checklist Template

Copy this checklist into every PR description:

- [ ] **Scope:** Maps to one plan item (or approved exception documented).
- [ ] **Correctness:** Added/updated tests for changed logic.
- [ ] **Regression Safety:** Relevant existing tests pass.
- [ ] **CI:** All required GitHub Actions checks green on the PR head commit.
- [ ] **Security:** No insecure defaults introduced; secrets not exposed.
- [ ] **Reliability:** Failure paths handled; retries/timeouts/backoff are sane.
- [ ] **Performance:** No obvious unbounded loops/maps or avoidable N^2 regressions.
- [ ] **Docs:** Updated README/docs/config examples if behavior changed.
- [ ] **Ops/Runbook:** Added migration/rollout notes when operational behavior changes.
- [ ] **Verification Evidence:** Commands run and results captured in PR notes.
- [ ] **Tracking:** Plan status/checklists and PR URL updated.

## Verification Gate (Before Merge)

Local verification is required, but **green CI is also a mandatory merge gate**. Do not merge until required GitHub Actions jobs pass on the PR head commit (verify with `gh pr checks <number>` or the PR checks UI).

Minimum verification for touched stack:

- Go changes (module root is `go/`):
  - `make go-test` (or `cd go && go test ./... -v -cover`)
  - `make go-lint` (or `cd go && golangci-lint run ./...`)
  - `cd go && go vet ./...`
- Python changes (package root is `python/`):
  - `make python-test` (or `cd python && pytest tests/ -v --cov=. --cov-report=html`)
  - `make python-lint` (or `cd python && black . --check && flake8 . && mypy .`)
  - `cd python && bandit -r .`
- Cross-cutting/integration changes:
  - `make test-all`
  - `make security-scan`

If a command is not applicable or temporarily skipped, document the reason in the PR.

Before merge, confirm required CI is green:
- `gh pr checks <pr-number>` shows no failing required checks
- Re-run or fix CI after new commits; do not merge on stale green results from an older commit

## Common Commands

### Go

- `make go-test`
- `make go-lint`
- `cd go && go build -o ../bin/monitor ./cmd/monitor`
- `cd go && go build -o ../bin/api-server ./cmd/api-server`
- `cd go && go vet ./...`

### Python

- `python -m venv venv && source venv/bin/activate`
- `make python-test`
- `make python-lint`
- `cd python && bandit -r .`

### Combined

- `make test-all`
- `make build-all`
- `make security-scan`

## Update Procedure

When starting work:
- Set target PR section status to `In Progress`.
- Confirm scope and acceptance criteria in the active plan.

When finishing work:
- Add verification notes and PR link in the corresponding section.
- Mark completed checklist items and acceptance criteria.
- Update `Last updated` in the active plan.

## Longer-Term Improvement Themes

- Scalability: watch/incremental detection and caching.
- Reliability: stronger retries, backpressure, graceful degradation.
- Product depth: complete endpoint vertical slices with tests/docs.
- Governance: docs ownership, docs-lint checks, maturity-based docs.
