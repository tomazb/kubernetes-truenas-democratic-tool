# PR 7: Test and CI/Release Trustworthiness — Design Spec

**Date:** 2026-05-28  
**Plan item:** [Remediation plan PR 7](../plans/2026-05-28-repo-health-remediation.md)  
**Milestone:** M3 — Operational Readiness

## Problem statement and scope

Automation and local developer targets reference paths and artifacts that do not exist in the current repository layout. This causes false confidence (release/helm targets that cannot succeed) and friction (Makefile `go-build` fails on missing `cmd/controller`, pytest coverage gates disagree between `pyproject.toml` and CI).

**In scope:**

- Align Makefile targets with existing Go/Python/cmd and test layout
- Fix release workflow Dockerfile mapping (`api-server` → `Dockerfile.api`) and broken job outputs
- Remove or gate release steps that reference missing Helm chart assets
- Add a CI/Makefile precheck that fails when referenced build/test paths are missing
- Align Python coverage gate configuration (`pyproject.toml`, `pytest.ini`, CI, Makefile) to the interim 70% threshold documented in backlog BL-20260528-python-coverage-gate

**Out of scope:**

- Raising Python coverage to 90% (tracked in backlog; requires additional test debt work)
- Implementing `cmd/controller` or Helm chart assets (Stage 3+ product depth)
- Documentation accuracy refresh (PR 8)
- Net-new product features

## Current vs target behavior

| Area | Current | Target |
|------|---------|--------|
| `make go-build` | Builds missing `./cmd/controller` | Builds only shipped binaries (`monitor`, `api-server`) |
| `make test-e2e` etc. | Invokes repo-root `tests/` dirs that do not exist | Uses `python/tests/` with pytest markers or skips when no tests match |
| `make docs` / `helm-*` | Reference missing `python/docs/` and `deploy/helm/` | Removed or fail fast with clear message via precheck |
| Release containers | Matrix uses `Dockerfile.api-server` (missing) | Maps components to existing Dockerfiles |
| Release Helm publish | References missing chart + undefined `upload_url` output | Job removed until chart exists; release job uses maintained action |
| Coverage gates | `pyproject.toml` requires 90%; CI uses 70% | All gates aligned at 70% until backlog item completes |
| CI trust | No path validation before expensive jobs | `ci-precheck` job validates referenced paths early |

## Technical approach

1. Add `scripts/ci-precheck.sh` validating Go cmd paths, Dockerfiles, Python test root, and Makefile build targets.
2. Wire `ci-precheck` Makefile target and a fast CI job that runs before build/test jobs.
3. Trim Makefile to match shipped components; route optional test tiers through pytest markers under `python/tests/`.
4. Fix `.github/workflows/release.yml` component→Dockerfile mapping; replace deprecated release action; drop Helm publish until chart lands.
5. Remove stale `deploy/docker/Dockerfile.controller` (references non-existent `cmd/controller`; not built in CI).
6. Align `fail_under` / `--cov-fail-under` to 70 across config surfaces.

## Risk, failure modes, and mitigations

| Risk | Mitigation |
|------|------------|
| Removing Helm publish blocks chart releases | Backlog entry; chart is not present today so job was already broken |
| Lowering pyproject `fail_under` masks coverage debt | Document linkage to BL-20260528; CI still enforces threshold explicitly |
| Precheck too strict on aspirational paths | Only validate paths referenced by CI, Makefile, and release workflow |
| Release action migration changes permissions | Keep `contents: write` and `packages: write`; test workflow syntax locally |

## Test strategy

```bash
./scripts/ci-precheck.sh
make go-build go-test
make python-test
make ci-precheck
# YAML sanity (optional): actionlint if available
```

## Rollout and backout

**Rollout:** Merge PR; CI precheck runs on all PRs; release workflow fixed on next tag push.

**Backout:** Revert PR; restores prior (broken) Makefile/release targets.
