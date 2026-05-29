# Design (PR #46): Consolidated Go module dependency bumps

- Date: 2026-05-29
- Status: Implemented
- Supersedes Dependabot PRs: #42 (x/crypto), #38 (x/time), #37 (gin), #36 (testify),
  #21 (zap), #14 (logr). Also closes obsolete #12 (controller-runtime) and #13 (viper).

## Problem statement and scope

Multiple individual Dependabot PRs bump Go modules. They conflict with each other
(separate go.mod/go.sum edits) and several are stale. Per the remediation decision, consolidate the
still-relevant Go bumps into a single PR with one `go mod tidy`, and close the obsolete/superseded
individual PRs.

Scope: `go/go.mod`, `go/go.sum`, and the Go toolchain version pinned in CI workflows
(`.github/workflows/ci.yml`, `codeql.yml`, `release.yml`) — see note below.

## Current vs target behavior

Direct dependency versions:

| Module | From | To | Source PR |
|--------|------|----|-----------|
| github.com/gin-gonic/gin | v1.9.1 | v1.11.0 | #37 |
| github.com/stretchr/testify | v1.9.0 | v1.11.1 | #36 |
| go.uber.org/zap | v1.26.0 | v1.27.0 | #21 |
| golang.org/x/time | v0.6.0 | v0.14.0 | #38 |
| golang.org/x/crypto (indirect) | v0.36.0 | v0.45.0 | #42 |
| github.com/go-logr/logr (indirect) | v1.4.2 | v1.4.3 | #14 |
| github.com/prometheus/client_golang (direct, via tidy) | v1.17.0 | v1.19.1 | (consolidated) |

Transitive upgrades pulled by `go mod tidy` (gin 1.11 chain, x/net, protobuf, etc.) are reflected
in go.sum.

### Go toolchain bump (required)
`golang.org/x/time` v0.14.0, `golang.org/x/crypto` v0.45.0, and the transitive `golang.org/x/net`
v0.47.0 all declare `go 1.24.0`. `go mod tidy` therefore raises the module's `go` directive from
`1.23.0` to `1.24.0` (and drops the now-redundant `toolchain` line). To keep CI building, the Go
version pinned in workflows is raised accordingly:
- `.github/workflows/ci.yml` `GO_VERSION`: `1.23` → `1.24`
- `.github/workflows/codeql.yml` `go-version`: `1.23` → `1.24`
- `.github/workflows/release.yml` `go-version`: `1.21` → `1.24`

The container images already build on `golang:1.24-alpine`, so no Dockerfile change is needed here.

The `golangci-lint` step gains `args: --timeout=5m`: the larger module graph from gin 1.11
(adds quic-go and others) pushed analysis past the action's 1-minute default, causing a timeout
(exit code 4) unrelated to any real lint issue.

### Obsolete PRs (no longer applicable)
- #12 `sigs.k8s.io/controller-runtime` 0.16→0.21 — package is no longer a dependency in go.mod.
- #13 `github.com/spf13/viper` 1.17→1.20 — package is no longer a dependency in go.mod.

## Technical approach and alternatives

`go get <module>@<version>` for each target, then `go mod tidy`. Validate with build, vet, race
tests, and golangci-lint.

Alternative (rejected): rebase each Dependabot PR individually — produces 6+ conflicting PRs and
redundant CI runs for the same go.sum.

## Risk / failure modes and mitigations

- gin 1.9→1.11 is a two-minor jump with a larger transitive graph (adds quic-go). Mitigation:
  full build + race test suite pass; no source changes required (verified).
- Indirect upgrades could surface incompatibilities. Mitigation: `go build ./...` and
  `go test ./... -race` both green.

## Test strategy and validation commands

- `cd go && go build ./...`
- `cd go && go vet ./...`
- `cd go && go test ./... -race`
- `cd go && golangci-lint run ./...`

## Rollout / backout

- Rollout: merge to `main`; CI revalidates.
- Backout: revert the go.mod/go.sum commit.
