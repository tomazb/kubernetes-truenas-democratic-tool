# Design (PR #45): Fix CI Security Scan SARIF upload permissions

- Date: 2026-05-29
- Status: Implemented
- Related: unblocks all open Dependabot dependency PRs (#9, #12, #13, #14, #15, #21, #31, #34, #36, #37, #38, #39, #40, #41, #42)

## Problem statement and scope

The required status check `CI Pipeline` is failing on `main` and therefore on every open PR. The
failure is **not** a test/lint/build failure — the `Security Scan` job's
`github/codeql-action/upload-sarif` step fails with
`Resource not accessible by integration` because the job has no `permissions:` block granting
`security-events: write`. Dependabot/fork PRs additionally run with a read-only token and cannot
upload SARIF at all.

Scope: `.github/workflows/ci.yml`, `security-scan` job only. No application code changes.

## Current vs target behavior

- Current: `security-scan` job inherits default token permissions (read-only for the SARIF upload
  context) → upload-sarif fails → whole job fails → `CI Pipeline` red everywhere.
- Target: `security-scan` job declares `security-events: write` so SARIF uploads succeed on pushes
  to protected branches; on read-only (Dependabot/fork) PR contexts the upload step is tolerated
  via `continue-on-error` so the job stays green while Trivy/Bandit/Safety still run.

## Technical approach

1. Add a job-scoped `permissions` block:
   ```yaml
   permissions:
     contents: read
     security-events: write
   ```
2. Add `continue-on-error: true` to the upload-sarif step so Dependabot/fork PRs (which cannot
   write security events) do not fail the required check.

Alternatives considered:
- Workflow-level `permissions`: broader than needed; job-scoped is least-privilege.
- Dropping the SARIF upload entirely: loses GitHub Security tab integration on main. Rejected.
- Gating the upload on `github.event_name == 'push'`: viable, but `continue-on-error` keeps Trivy
  output visible in PR logs without branching logic. Chosen for simplicity.

## Risk / failure modes and mitigations

- Risk: over-broad permissions. Mitigation: job-scoped, only `security-events: write` + `contents: read`.
- Risk: upload silently failing on main. Mitigation: `continue-on-error` only tolerates the upload
  step; the proper permissions mean it succeeds on push. Visible in job logs if it regresses.

## Test strategy and validation

- Open the PR and confirm the `Security Scan` job (and overall `CI Pipeline`) is green.
- `actionlint .github/workflows/ci.yml` (or YAML lint) to confirm syntax.

## Rollout / backout

- Rollout: merge to `main`; required check turns green, unblocking dependency PRs.
- Backout: revert the single-file change; behavior returns to prior (failing) state.
