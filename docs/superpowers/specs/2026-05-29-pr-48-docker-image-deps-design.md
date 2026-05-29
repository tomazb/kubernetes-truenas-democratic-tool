# Design (PR #48): Docker base image dependency bumps

- Date: 2026-05-29
- Status: Implemented
- Supersedes Dependabot PRs: #31 (golang base), #40 (python base).

## Problem statement and scope

Dependabot proposes newer base images for the container builds. Consolidate into a single change.

Scope: `deploy/docker/Dockerfile.api`, `Dockerfile.controller`, `Dockerfile.monitor`,
`Dockerfile.cli`.

## Current vs target behavior

| Image | From | To | Source PR | Files |
|-------|------|----|-----------|-------|
| golang (builder) | 1.24-alpine | 1.25-alpine | #31 | Dockerfile.api, .controller, .monitor |
| python (runtime) | 3.13-slim | 3.14-slim | #40 | Dockerfile.cli |

`go.mod` requires `go 1.24.0` (set in #46); `golang:1.25-alpine` satisfies this.

## Technical approach and alternatives

Update the `FROM` lines. Alternative (rejected): merge each Dependabot PR separately.

## Risk / failure modes and mitigations

- **python:3.14-slim** could break the CLI image if a dependency lacks 3.14 wheels. Mitigation:
  the test suite already runs on Python 3.14 locally (pytest green), indicating the package and its
  dependencies install/run on 3.14. The `Container Build Test` CI job is the gate; if it fails,
  hold #40 (revert the python bump) and record in backlog rather than force-merge.
- golang 1.25-alpine builder is a minor bump; Go is backward compatible. Validated by the Go
  builder stages in `Container Build Test`.

## Test strategy and validation

- `Container Build Test` CI job builds all three container images (monitor, api, cli).
- No local Docker available in the working environment; CI is the authoritative gate.

## Rollout / backout

- Rollout: merge to `main`.
- Backout: revert the `FROM` line(s). The python and golang bumps are independent and can be
  reverted separately if only one base image regresses.
