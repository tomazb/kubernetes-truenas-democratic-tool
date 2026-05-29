# Canonical Backlog

This is the single source of truth for deferred work that is not in active PR scope.

## Usage Rules

- Add items here when review comments or discoveries are valuable but out of scope for the current PR/spec.
- Do not implement backlog items in the current PR unless scope is explicitly updated (spec + plan).
- Each item must include rationale and a proposed target phase/PR.
- Link related PRs, issues, and specs for traceability.

## Intake Template

Copy this template for each new item:

```md
### BL-YYYYMMDD-<short-slug>
- Status: Proposed | Planned | In Progress | Done | Rejected
- Priority: P0 | P1 | P2 | P3
- Milestone: M1 | M2 | M3 | M4 | Unassigned
- Source: PR comment | Audit | Incident | Other
- Opened by: <name>
- Date: YYYY-MM-DD
- In scope of current PR: No
- Rationale:
- Proposed target: PR <n> | Stage <n> | Separate epic
- Related spec(s):
- Related PR(s)/issue(s):
- Description:
- Acceptance criteria:
  - [ ] ...
  - [ ] ...
```

## Status Definitions

- **Proposed:** captured and awaiting triage.
- **Planned:** accepted and assigned to a future target.
- **In Progress:** actively being implemented.
- **Done:** implemented and verified.
- **Rejected:** intentionally not pursued, with reason recorded.

## Milestone Map

- **M1 (Baseline Safety):** Security and correctness baseline work.
- **M2 (Contract Integrity):** API/detector/validation contract fidelity.
- **M3 (Operational Readiness):** Reliability, CI/release trust, docs accuracy.
- **M4 (Scale and Product Depth):** Scalability, advanced product depth, and governance maturity.

## Backlog Items

### BL-20260528-python-coverage-gate
- Status: Planned
- Priority: P1
- Milestone: M3
- Source: PR comment
- Opened by: agent
- Date: 2026-05-28
- In scope of current PR: No (documented interim gate only)
- Rationale: Python unit coverage is ~71% today; enforcing 90% in CI would block merges until test debt from PR 2–3 work is cleared.
- Proposed target: PR 7 (CI/release trust + docs accuracy)
- Related spec(s): `docs/superpowers/plans/2026-05-28-repo-health-remediation.md` (PR 7)
- Related PR(s)/issue(s): https://github.com/tomazb/kubernetes-truenas-democratic-tool/pull/43
- Description: Raise CI `--cov-fail-under` from interim 70% to 90% once remediation PRs improve suite coverage and contract tests.
- Acceptance criteria:
  - [ ] `pytest` coverage for `truenas_storage_monitor` is >=90% locally.
  - [ ] `.github/workflows/ci.yml` uses `--cov-fail-under=90`.
  - [ ] `python/pyproject.toml` `fail_under` aligned with CI.

### BL-20260528-backlog-bootstrap
- Status: Done
- Priority: P2
- Milestone: M3
- Source: Process improvement
- Opened by: agent
- Date: 2026-05-28
- In scope of current PR: No
- Rationale: Establish canonical backlog process to support strict PR scope discipline.
- Proposed target: Immediate process baseline
- Related spec(s): `docs/superpowers/plans/2026-05-28-repo-health-remediation.md`
- Related PR(s)/issue(s): https://github.com/tomazb/kubernetes-truenas-democratic-tool/pull/43
- Description: Create and adopt a single backlog document for out-of-scope but valuable feedback and findings.
- Acceptance criteria:
  - [x] `AGENTS.md` points to this backlog.
  - [x] Active plan points to this backlog.
  - [x] Out-of-scope PR comments are recorded here with target phase/PR.

### BL-20260529-dependency-pr-remediation
- Status: Done
- Priority: P1
- Milestone: M3
- Source: Maintenance request (@tomazb): review, validate and address all dependency PRs
- Opened by: agent
- Date: 2026-05-29
- In scope of current PR: No (tracking record)
- Rationale: 15 open Dependabot PRs were all blocked by a red required `CI Pipeline` check caused by the Security Scan SARIF upload lacking `security-events: write` permission (not test failures). Several Go PRs were stale/superseded. Consolidated by ecosystem and addressed.
- Proposed target: Completed
- Related spec(s):
  - `docs/superpowers/specs/2026-05-29-pr-45-ci-security-scan-permissions-design.md`
  - `docs/superpowers/specs/2026-05-29-pr-46-go-module-deps-design.md`
  - `docs/superpowers/specs/2026-05-29-pr-47-github-actions-deps-design.md`
  - `docs/superpowers/specs/2026-05-29-pr-48-docker-image-deps-design.md`
  - `docs/superpowers/specs/2026-05-29-pr-50-quic-go-dep-design.md`
- Related PR(s)/issue(s):
  - Merged: #45 (CI fix), #46 (Go modules + Go 1.24 toolchain), #47 (GitHub Actions), #48 (Docker images), #50 (quic-go)
  - Superseded/closed: #42, #38, #37, #36, #21, #14 (folded into #46); #39, #34, #41, #15 (folded into #47); #31, #40 (folded into #48); #49 (folded into #50)
  - Obsolete/closed (no longer applicable): #12 controller-runtime, #13 viper (removed from go.mod); #9 codecov (already at v5)
- Description: Fixed the CI permissions blocker, then consolidated and merged all still-relevant dependency bumps; bumped CI/CodeQL/release Go to 1.24 to match go.mod and raised golangci-lint timeout for the larger dependency graph.
- Acceptance criteria:
  - [x] Required `CI Pipeline` check green on `main` and on each dependency PR.
  - [x] Every open Dependabot dependency PR merged or closed with rationale.
  - [x] Each merged change has a design spec under `docs/superpowers/specs/`.
