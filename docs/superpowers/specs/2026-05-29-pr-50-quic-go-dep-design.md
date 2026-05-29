# Design (PR #50): quic-go indirect dependency bump

- Date: 2026-05-29
- Status: Implemented
- Supersedes Dependabot PR: #49 (quic-go 0.54.0 → 0.57.0).

## Problem statement and scope

After the consolidated Go bump (#46) raised gin to v1.11.0, `github.com/quic-go/quic-go` entered the
module graph (indirect). Dependabot then opened #49 to bump it 0.54.0 → 0.57.0. Apply the bump.

Scope: `go/go.mod`, `go/go.sum`.

## Current vs target behavior

| Module | From | To | Source PR |
|--------|------|----|-----------|
| github.com/quic-go/quic-go (indirect) | v0.54.0 | v0.57.0 | #49 |
| github.com/quic-go/qpack (indirect, via tidy) | v0.5.1 | v0.6.0 | — |
| go.uber.org/mock (indirect, via tidy) | v0.5.0 | v0.5.2 | — |

## Technical approach

`go get github.com/quic-go/quic-go@v0.57.0 && go mod tidy`. quic-go is only an indirect dependency
(pulled by gin's HTTP/3 support); the project does not import it directly.

## Risk / failure modes and mitigations

- Indirect-only; no direct API surface used. Mitigation: `go build ./...` and `go test ./...` pass.

## Test strategy and validation

- `cd go && go build ./...`
- `cd go && go test ./...`
- Full `CI Pipeline` on the PR.

## Rollout / backout

- Rollout: merge to `main`.
- Backout: revert the go.mod/go.sum commit.
