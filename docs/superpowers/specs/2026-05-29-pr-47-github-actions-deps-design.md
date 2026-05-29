# Design (PR #47): Consolidated GitHub Actions dependency bumps

- Date: 2026-05-29
- Status: Implemented
- Supersedes Dependabot PRs: #39 (codeql-action), #34 (setup-python), #41 (upload-artifact),
  #15 (setup-helm). Also closes obsolete #9 (codecov-action, already at v5).

## Problem statement and scope

Several Dependabot PRs bump GitHub Actions used in the workflows. Consolidate the still-relevant
ones into a single change and close the obsolete one.

Scope: `.github/workflows/ci.yml`, `.github/workflows/codeql.yml`, `.github/workflows/release.yml`.

## Current vs target behavior

| Action | From | To | Source PR | Files |
|--------|------|----|-----------|-------|
| github/codeql-action/* | v3 | v4 | #39 | codeql.yml (init, analyze), ci.yml (upload-sarif) |
| actions/setup-python | v4/v5 | v6 | #34 | ci.yml, release.yml |
| actions/upload-artifact | v3/v4 | v5 | #41 | ci.yml, release.yml |
| azure/setup-helm | v3 | v4 | #15 | release.yml |

### Obsolete PR
- #9 `codecov/codecov-action` 3→5 — `ci.yml` already pins `@v5`, so no change is needed.

## Technical approach and alternatives

Pin each action to the new major tag across all workflow files. Alternative (rejected): merge each
Dependabot PR separately — more CI runs and review overhead for trivial tag bumps.

## Risk / failure modes and mitigations

- `codeql-action@v4` and `setup-python@v6` are major bumps; behavior is backward compatible for our
  usage (default inputs). Mitigation: full `CI Pipeline` run validates on the PR.
- `upload-artifact@v4+` changed artifact immutability semantics (no same-name re-upload). Our
  workflows upload a single `build-artifacts` and SARIF once per run, so unaffected.

## Test strategy and validation

- YAML lint (`python -c "import yaml; ..."`).
- Full `CI Pipeline` green on the PR (exercises setup-python, upload-artifact, codeql upload-sarif).
- `release.yml` actions are validated by syntax + the next release run; low risk (tag bumps only).

## Rollout / backout

- Rollout: merge to `main`.
- Backout: revert the workflow edits.
