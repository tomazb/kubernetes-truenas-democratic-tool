# PR 1: Security Baseline Hardening — Design Spec

**Date:** 2026-05-28  
**Plan item:** [Remediation plan PR 1](../plans/2026-05-28-repo-health-remediation.md)  
**Milestone:** M1 — Baseline Safety

## Problem statement and scope

The Go TrueNAS HTTP client unconditionally disables TLS certificate verification (`InsecureSkipVerify: true`), which is unsafe for production. Operators cannot trust that connections to TrueNAS are authenticated unless they patch code.

**In scope:**

- Secure-by-default TLS for outbound TrueNAS API calls in `go/pkg/truenas`
- Explicit opt-in `truenas.insecure` for development/lab only
- Optional `truenas.ca_file` for custom CA bundles (e.g. internal PKI, self-signed TrueNAS)
- Config loading, validation, and wiring through `go/pkg/config` and `go/cmd/monitor`, `go/cmd/api-server`
- Unit tests and config/deploy examples

**Out of scope:**

- Python client (`verify_ssl` already defaults to `true`)
- API server inbound TLS (`security.tls` / `tls_min_version`)
- Renaming `insecure` vs `tls_verify` across Python and Go (deferred to docs/PR 8)

## Current behavior vs target behavior

| Aspect | Current | Target |
|--------|---------|--------|
| Default TLS | Always skip verify | Verify using system trust store |
| Dev/lab self-signed | Works silently | Requires `truenas.insecure: true` |
| Custom CA | Not supported | `truenas.ca_file` PEM bundle |
| Config example `insecure: false` | Not applied by Go | Honored by Go services |

Current code (`go/pkg/truenas/client.go`):

```go
httpClient.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
```

Target: `buildTLSConfig` with `InsecureSkipVerify: false` by default; set true only when `truenas.insecure: true`.

## Config contract

```yaml
truenas:
  url: https://truenas.example.com
  username: ${TRUENAS_USERNAME}
  password: ${TRUENAS_PASSWORD}
  timeout: 30s
  insecure: false   # dev/lab only; disables certificate verification
  ca_file: ""       # optional path to PEM CA bundle
```

Go mapping:

- `TrueNASConfig.Insecure` → `truenas.Config.Insecure` → `TLSOptions.InsecureSkipVerify`
- `TrueNASConfig.CAFile` → `truenas.Config.CAFile` → `TLSOptions.CAFile`

Validation: if `ca_file` is non-empty, the file must exist and contain parseable PEM certificates at config load time. Failed CA load must not enable insecure mode.

## Technical approach

1. Add `go/pkg/truenas/tls.go` with `buildTLSConfig(TLSOptions) (*tls.Config, error)`.
2. Extend `truenas.Config` and call `buildTLSConfig` from `NewClient`.
3. Extend `config.TrueNASConfig`; validate `ca_file`; pass fields from cmd entrypoints.
4. Tests: table-driven TLS builder tests; `httptest` TLS server for CA and insecure paths; config load tests.

**Alternatives considered:**

| Alternative | Decision |
|-------------|----------|
| Environment-only `TRUENAS_INSECURE` | Rejected — config file is source of truth for long-running services |
| Always use system + custom CA merge only when `ca_file` set | Accepted |
| Default insecure with opt-in verify | Rejected — violates secure-by-default |

## Risk, failure modes, and mitigations

| Risk | Mitigation |
|------|------------|
| Breaking change for deployments relying on silent skip-verify | Document in PR; operators set `insecure: true` temporarily or provide `ca_file` |
| Invalid `ca_file` at runtime | Fail at config load with clear error |
| CA file missing after deploy | Startup failure (fail fast) |
| `insecure: true` in production | Document as dev/lab only; optional backlog for runtime warning |

## Test strategy

| Test | Purpose |
|------|---------|
| `TestBuildTLSConfig_default` | Secure default |
| `TestBuildTLSConfig_insecure` | Explicit skip verify |
| `TestBuildTLSConfig_caFile` | Custom CA trust |
| `TestBuildTLSConfig_caFile_missing` | Missing file error |
| `TestBuildTLSConfig_caFile_invalidPEM` | Invalid PEM error |
| `TestNewClient_*` | Client wiring |
| `TestLoad` (config) | YAML `insecure` / `ca_file` |

**Validation commands:**

```bash
cd go && go test ./pkg/truenas/... ./pkg/config/... -v -cover
cd go && go vet ./...
make go-lint
make go-test
```

## Rollout and backout

**Rollout:** Deploy updated monitor/API images with config updated for environments using self-signed TrueNAS certs:

1. Mount CA and set `truenas.ca_file`, or
2. Set `truenas.insecure: true` (lab only), or
3. Add CA to node trust store (no `ca_file` needed if using system pool only).

**Backout:** Revert to previous image release; restore prior config if downgraded binary still skipped verify.
