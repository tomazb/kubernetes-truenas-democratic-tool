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

### BL-20260528-backlog-bootstrap
- Status: Planned
- Priority: P2
- Milestone: M3
- Source: Process improvement
- Opened by: agent
- Date: 2026-05-28
- In scope of current PR: No
- Rationale: Establish canonical backlog process to support strict PR scope discipline.
- Proposed target: Immediate process baseline
- Related spec(s): `docs/superpowers/plans/2026-05-28-repo-health-remediation.md`
- Related PR(s)/issue(s): N/A
- Description: Create and adopt a single backlog document for out-of-scope but valuable feedback and findings.
- Acceptance criteria:
  - [ ] `AGENTS.md` points to this backlog.
  - [ ] Active plan points to this backlog.
  - [ ] Out-of-scope PR comments are recorded here with target phase/PR.
