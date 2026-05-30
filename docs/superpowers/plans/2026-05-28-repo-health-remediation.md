# Repository Health Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate the highest-risk security/correctness gaps, align tests and docs with reality, and establish a sustainable path for performance and maintainability improvements.

**Architecture:** Execute in small PRs that each close one risk theme end-to-end (code + tests + docs). Prioritize security and runtime correctness first, then API contract integrity, then reliability/performance, then CI/docs governance.

**Tech Stack:** Go (services/APIs), Python (CLI/analysis), GitHub Actions, Markdown docs.

---

## How Tracking Works

- [ ] Before starting each PR, set its status to `In Progress`.
- [ ] Keep acceptance criteria in this file updated during implementation.
- [ ] Link the PR URL under the corresponding section once opened.
- [ ] Mark a PR `Done` only after tests, lint, and docs for that scope pass.
- [ ] If scope changes, update this plan first, then execute.
- [ ] Log out-of-scope but valuable items in `docs/superpowers/backlog.md`.
- [ ] Run code review before commit and again before push; record outcomes in PR notes.
- [ ] Ensure PR review-thread handling is complete (or backlogged with rationale) before merge.

## Code Review Gate (Required Before Commit and Push)

Every PR in this plan must pass two review checkpoints:

- [ ] Pre-commit review of working diff completed.
- [ ] Pre-push review of final staged/commit-ready diff completed.

Preferred command:
- `coderabbit review --prompt-only`

Fallback rule:
- If automated review tooling is unavailable, perform manual review and document fallback rationale in PR notes.

## PR Comment and Review Thread Policy (Required)

Review comments are part of completion criteria and must be actively managed.

- [ ] Fetch and review all open PR comments/threads before merge.
- [ ] Validate each comment technically (do not apply suggestions blindly).
- [ ] For accepted comments: implement fix, add/adjust tests where relevant, and reply with concise summary.
- [ ] For non-applicable comments: reply with technical rationale and evidence.
- [ ] For out-of-scope comments (outside approved spec/design): do not implement in current PR; reply as out-of-scope and add to `docs/superpowers/backlog.md` when sensible with proposed target PR/milestone.
- [ ] Resolve threads only after code/tests are updated and response is posted on the PR.
- [ ] Final merge gate: no unresolved actionable review threads.

Scope rule:
- The approved PR spec/design is the source of truth for scope.
- Scope-changing feedback requires explicit spec + plan update before implementation.

## Spec and Design Policy (Required Per PR)

Every PR must have a dedicated spec/design document that defines the intended change and validation approach before coding starts.

- [ ] Create one PR-specific design/spec doc before implementation.
- [ ] Keep the spec in `docs/superpowers/specs/`.
- [ ] Name specs with date + PR identifier + topic:
  - `docs/superpowers/specs/YYYY-MM-DD-pr-<n>-<topic>-design.md`
- [ ] Include at minimum:
  - problem statement and scope boundaries
  - current behavior vs target behavior
  - technical approach and alternatives considered
  - risk/failure modes and mitigations
  - test strategy and concrete validation commands
  - rollout/backout notes when operational behavior changes
- [ ] Link the spec in the PR description and under the PR section in this plan.
- [ ] Update the spec if implementation scope changes.
- [ ] Do not merge PRs whose implementation materially diverges from spec without spec update.

Backlog linkage:
- [ ] For review comments outside spec scope, reply on PR and add an entry to `docs/superpowers/backlog.md` with proposed target PR/phase.

## Branch and Worktree Policy (Required)

Every PR in this plan must be developed in its own branch and dedicated worktree.

- [ ] Create a branch per PR using a clear naming pattern (`feature/pr-<n>-<topic>` or `bugfix/pr-<n>-<topic>`).
- [ ] Create a dedicated worktree for that branch before making changes.
- [ ] Keep only one plan PR worth of changes in each worktree.
- [ ] Do not mix files for multiple plan PRs in the same branch/worktree.
- [ ] Rebase/sync that branch regularly against its target base branch as needed.
- [ ] After PR merge/close, remove the worktree and local branch if no longer needed.

Suggested worktree layout:
- `.worktrees/pr-1-security-baseline`
- `.worktrees/pr-2-python-correctness`
- `.worktrees/pr-3-python-import-hygiene`
- `.worktrees/pr-4-api-honesty`
- `.worktrees/pr-5-detector-validation`
- `.worktrees/pr-6-reliability-performance`
- `.worktrees/pr-7-test-ci-alignment`
- `.worktrees/pr-8-docs-accuracy`

## Overall Status

- **Current phase:** Phase 3 - Operational readiness (PR 8 in progress)
- **Primary owner:** TBD
- **Last updated:** 2026-05-30
- **Last merged:** PR 7 test/CI alignment [#57](https://github.com/tomazb/kubernetes-truenas-democratic-tool/pull/57) (2026-05-30)
- **Next up:** PR 8 - Documentation accuracy refresh

## Milestones

### M1: Baseline Safety

- **Goal:** Eliminate high-risk security and correctness defects in baseline runtime and imports.
- **Includes:** PR 1, PR 2, PR 3
- **Exit criteria:**
  - TLS secure-by-default implemented and tested.
  - Python monitor/config/timezone correctness validated.
  - Python import/packaging side effects removed and tested.

### M2: Contract Integrity

- **Goal:** Ensure API and detector behavior is honest, deterministic, and test-backed.
- **Includes:** PR 4, PR 5
- **Exit criteria:**
  - No misleading success placeholders for unimplemented behavior.
  - Snapshot and validation paths produce real signal (not hardcoded pass).
  - Handler/detector tests cover core contracts.

### M3: Operational Readiness

- **Goal:** Improve runtime resilience and release confidence for day-2 operations.
- **Includes:** PR 6, PR 7, PR 8
- **Exit criteria:**
  - Reliability/performance guardrails merged.
  - CI/release and tests align with actual repo structure.
  - Documentation accurately reflects current behavior and maturity.

### M4: Scale and Product Depth

- **Goal:** Deliver roadmap-stage scalability, resilience, and product depth capabilities.
- **Includes:** Stage 2+, Stage 3, Stage 4, Stage 5 items
- **Exit criteria:**
  - Incremental/scalable detection approach adopted.
  - Product-depth endpoints/features implemented with tests/docs.
  - Governance and docs lifecycle controls operationalized.

## PR Backlog and Execution Tracking

### Foundation: Agent Governance and Tracking (GitHub PR #43)

- **Status:** Done
- **Risk:** Low (process/docs; bundled CI/Python test alignment)
- **Focus:** Establish `AGENTS.md`, remediation plan, canonical backlog, specs layout, and contributor pointers.
- **Branch:** `docs/governance-playbook-and-tracking` (merged; remove local branch after sync)
- **Merged:** 2026-05-29
- **Merge commit:** `6a635d3`
- **PR URL:** https://github.com/tomazb/kubernetes-truenas-democratic-tool/pull/43
- **Acceptance criteria:**
  - [x] `AGENTS.md` is the single source of truth for agent/repo guidance.
  - [x] Active remediation plan and backlog documents exist under `docs/superpowers/`.
  - [x] Spec directory and naming convention documented (`docs/superpowers/specs/README.md`).
  - [x] CI/Python checks aligned enough to merge governance work.
- **Notes:** Numbered remediation PRs (1-8) remain **Planned**; do not count this foundation PR toward M1-M3 exit criteria.

### PR 1: Security Baseline Hardening

- **Status:** Done
- **Risk:** High
- **Focus:** Secure TLS defaults for TrueNAS communication.
- **Branch/worktree:**
  - Branch: `feature/pr-1-security-baseline`
  - Worktree: `.worktrees/pr-1-security-baseline`
- **Spec/design:** `docs/superpowers/specs/2026-05-28-pr-1-security-baseline-design.md`
- **Files (expected):**
  - `go/pkg/truenas/client.go`
  - `go/pkg/config/config.go` (if config wiring is needed)
  - `go/pkg/truenas/*_test.go`
- **Implementation steps:**
  - [x] Make TLS verification secure-by-default.
  - [x] Add explicit opt-in insecure mode for development use only.
  - [x] Add custom CA bundle support (or clear error if unavailable).
  - [x] Add tests for secure default and explicit insecure behavior.
  - [x] Update docs/config example for new TLS options.
- **Acceptance criteria:**
  - [x] No unconditional `InsecureSkipVerify: true`.
  - [x] Default runtime verifies certificates.
  - [x] Tests cover secure and insecure modes.
- **PR URL:** https://github.com/tomazb/kubernetes-truenas-democratic-tool/pull/44

### PR 2: Python Correctness and Contract Alignment

- **Status:** Done
- **Risk:** High
- **Focus:** Remove runtime mismatches and datetime bugs in monitor path.
- **Branch/worktree:**
  - Branch: `bugfix/pr-2-python-correctness` (merged; removed)
  - Worktree: `.worktrees/pr-2-python-correctness` (removed)
- **Merged:** 2026-05-29
- **Merge commit:** `d4a3631`
- **Spec/design:** `docs/superpowers/specs/2026-05-28-pr-2-python-correctness-design.md`
- **Files (expected):**
  - `python/truenas_storage_monitor/monitor.py`
  - `python/truenas_storage_monitor/k8s_client.py`
  - `python/truenas_storage_monitor/config.py`
  - `python/tests/unit/test_monitor*.py`
  - `python/tests/unit/test_k8s_client*.py`
- **Implementation steps:**
  - [x] Fix monitor-to-client method contract mismatches.
  - [x] Resolve config shape mismatch (`kubernetes` vs `openshift` semantics).
  - [x] Standardize datetime comparisons to timezone-aware UTC.
  - [x] Add tests for mixed timezone inputs and boundary behavior.
  - [x] Validate monitor initialization with real public APIs.
- **Acceptance criteria:**
  - [x] No naive/aware datetime comparison failures.
  - [x] Monitor initialization and checks use valid client methods.
  - [x] Unit tests cover contract and timezone edge cases.
- **PR URL:** https://github.com/tomazb/kubernetes-truenas-democratic-tool/pull/52

### PR 3: Python Import and Packaging Hygiene

- **Status:** Done
- **Risk:** Medium
- **Focus:** Avoid import side effects and CLI dependency leakage.
- **Branch/worktree:**
  - Branch: `bugfix/pr-3-python-import-hygiene` (merged; removed)
  - Worktree: `.worktrees/pr-3-python-import-hygiene` (removed)
- **Merged:** 2026-05-29
- **Merge commit:** `4a855d7`
- **Spec/design:** `docs/superpowers/specs/2026-05-28-pr-3-python-import-hygiene-design.md`
- **Files (expected):**
  - `python/truenas_storage_monitor/__init__.py`
  - `python/truenas_storage_monitor/cli.py`
  - Python package metadata files
  - New import smoke tests
- **Implementation steps:**
  - [x] Remove CLI-heavy imports from package `__init__`.
  - [x] Keep CLI entrypoint separate from library import path.
  - [x] Add smoke tests proving core imports work without optional CLI deps.
- **Acceptance criteria:**
  - [x] `import truenas_storage_monitor` succeeds without CLI extras.
  - [x] Test collection/import does not fail due to optional packages.
- **PR URL:** https://github.com/tomazb/kubernetes-truenas-democratic-tool/pull/53

### PR 4: API Honesty and Endpoint Maturity Cleanup

- **Status:** Done
- **Risk:** High
- **Focus:** Replace misleading placeholder success payloads.
- **Branch/worktree:**
  - Branch: `feature/pr-4-api-honesty` (merged; removed)
  - Worktree: `.worktrees/pr-4-api-honesty` (removed)
- **Merged:** 2026-05-30
- **Merge commit:** `1d619a7`
- **Spec/design:** `docs/superpowers/specs/2026-05-28-pr-4-api-honesty-design.md`
- **Files (expected):**
  - `go/pkg/api/server.go`
  - `go/pkg/api/*_test.go`
  - API docs section in `README.md` or `docs/`
- **Implementation steps:**
  - [x] Identify placeholder endpoints returning pseudo-success.
  - [x] Standardize unfinished endpoints to explicit `501` responses.
  - [x] Optionally implement one complete vertical slice endpoint (`/api/v1/orphans`).
  - [x] Add handler contract tests for implemented and unimplemented endpoints.
- **Acceptance criteria:**
  - [x] No placeholder endpoint returns misleading success data.
  - [x] Endpoint maturity is explicit and test-verified.
- **PR URL:** https://github.com/tomazb/kubernetes-truenas-democratic-tool/pull/54

### PR 5: Orphan Detection and Validation Debt

- **Status:** Done
- **Risk:** High
- **Focus:** Replace hardcoded detector logic and optimistic validation stubs.
- **Branch/worktree:**
  - Branch: `feature/pr-5-detector-validation`
  - Worktree: `.worktrees/pr-5-detector-validation`
- **Spec/design:** `docs/superpowers/specs/2026-05-28-pr-5-detector-validation-design.md`
- **Files (expected):**
  - `go/pkg/orphan/detector.go`
  - `go/pkg/k8s/client.go`
  - `go/pkg/orphan/*_test.go`
  - `go/pkg/k8s/*_test.go`
- **Implementation steps:**
  - [x] Implement real snapshot matching in detector.
  - [x] Replace hardcoded `return true` paths with deterministic matching.
  - [x] Replace always-pass RBAC validation placeholders with real checks or explicit not-implemented errors.
  - [x] Add table-driven tests for volume handles and snapshot correlation.
- **Acceptance criteria:**
  - [x] Snapshot orphan detection produces real signal.
  - [x] Validation behavior is not silently optimistic.
  - [x] Tests cover core matching edge cases.
- **PR URL:** https://github.com/tomazb/kubernetes-truenas-democratic-tool/pull/55

### PR 6: Reliability and Performance Guardrails

- **Status:** Done
- **Risk:** Medium
- **Focus:** Improve resilience and fairness under load/failures.
- **Branch/worktree:**
  - Branch: `feature/pr-6-reliability-performance`
  - Worktree: `.worktrees/pr-6-reliability-performance`
- **Spec/design:** `docs/superpowers/specs/2026-05-28-pr-6-reliability-performance-design.md`
- **Files (expected):**
  - `go/pkg/api/server.go`
  - `go/pkg/monitor/service.go`
  - `go/pkg/k8s/client.go`
  - Corresponding tests
- **Implementation steps:**
  - [x] Replace global rate limiter with per-client limiter strategy.
  - [x] Add limiter eviction/cleanup to prevent unbounded growth.
  - [x] Guard nil metrics exporter paths to prevent panics.
  - [x] Restrict retries to transient error classes.
  - [x] Add tests for limiter fairness and retry predicates.
- **Acceptance criteria:**
  - [x] No nil exporter panic path.
  - [x] No global limiter starvation by a single noisy client.
  - [x] Retry behavior avoids repeated non-transient failures.
- **PR URL:** https://github.com/tomazb/kubernetes-truenas-democratic-tool/pull/56

### PR 7: Test and CI/Release Trustworthiness

- **Status:** Done
- **Risk:** High
- **Focus:** Align tests/workflows with actual repo structure.
- **Branch/worktree:**
  - Branch: `bugfix/pr-7-test-ci-alignment` (merged; removed)
  - Worktree: `.worktrees/pr-7-test-ci-alignment` (removed)
- **Spec/design:** `docs/superpowers/specs/2026-05-28-pr-7-test-ci-alignment-design.md`
- **Files (expected):**
  - `scripts/ci-precheck.sh`
  - `go/pkg/monitor/service_test.go`
  - `go/pkg/k8s/client_test.go`
  - `.github/workflows/ci.yml`
  - `.github/workflows/release.yml`
  - `Makefile`
  - `python/pyproject.toml`
- **Implementation steps:**
  - [x] Fix stale test assumptions and signature drift.
  - [x] Remove or implement missing artifact references in workflows/make targets.
  - [x] Fix release workflow output wiring issues.
  - [x] Add precheck validating referenced paths/targets exist.
- **Acceptance criteria:**
  - [x] CI is green against current code layout.
  - [x] Release workflow references valid outputs/artifacts.
  - [x] Tests fail on regressions, not stale wiring.
- **PR URL:** https://github.com/tomazb/kubernetes-truenas-democratic-tool/pull/57

### PR 8: Documentation Accuracy Refresh

- **Status:** In Progress
- **Risk:** Medium
- **Focus:** Bring docs in sync with implemented reality.
- **Branch/worktree:**
  - Branch: `docs/pr-8-docs-accuracy`
  - Worktree: `.worktrees/pr-8-docs-accuracy`
- **Spec/design:** `docs/superpowers/specs/2026-05-28-pr-8-docs-accuracy-design.md`
- **Files (expected):**
  - `README.md`
  - `docs/ARCHITECTURE.md`
  - `CONTRIBUTING.md`
  - `config.yaml.example`
  - `config.go.example`
  - `docs/config-compatibility.md`
  - `python/pyproject.toml`
- **Implementation steps:**
  - [x] Remove stale/missing command/path references.
  - [x] Split architecture into current vs target state.
  - [x] Add config compatibility section (Go vs Python).
  - [x] Add endpoint maturity table/list.
  - [x] Remove placeholder external links.
- **Acceptance criteria:**
  - [x] New contributor can run documented paths without dead ends.
  - [x] Architecture clearly distinguishes shipped vs planned components.
- **PR URL:** https://github.com/tomazb/kubernetes-truenas-democratic-tool/pull/58

## Phase 2+ Roadmap (Later Stages)

### Stage 2: Scalability Foundation

- [ ] Introduce incremental/watch-based detection where practical.
- [ ] Add caching/TTL for expensive list operations.
- [ ] Define and track performance budgets (scan time, API p95, memory).

### Stage 3: Product Depth

- [ ] Implement additional API routes end-to-end with tests/docs in same PR.
- [ ] Add persistent storage for trend/history analysis.
- [ ] Reduce duplicated Go/Python business logic via clearer service boundaries.

### Stage 4: Operational Maturity

- [ ] Add resilience patterns (graceful degradation, backpressure).
- [ ] Add load/perf tests for realistic object cardinality.
- [ ] Define SLOs and alerts for scan freshness, API availability, and correctness.

### Stage 5: Documentation Governance

- [ ] Add docs ownership and docs-impact checklist to PR template.
- [ ] Add docs-lint checks for path/command validity in CI.
- [ ] Version documentation by maturity/release channel.

## Sequencing

- [ ] Complete PR 1 first (security baseline).
- [ ] Complete PRs 2 and 3 (Python correctness/hygiene).
- [ ] Complete PRs 4 and 5 (API honesty + detector fidelity).
- [x] Complete PR 6 (reliability/performance guardrails).
- [x] Complete PR 7 (CI trust).
- [x] Complete PR 8 (docs accuracy).
- [ ] Start Stage 2+ initiatives after baseline PRs are merged.
