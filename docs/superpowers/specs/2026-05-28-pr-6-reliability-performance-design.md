# PR 6: Reliability and Performance Guardrails — Design Spec

**Date:** 2026-05-28  
**Plan item:** [Remediation plan PR 6](../plans/2026-05-28-repo-health-remediation.md)  
**Milestone:** M3 — Operational Readiness

## Problem statement and scope

Runtime paths can fail or degrade under load:

- API rate limiting uses one global `rate.Limiter`, so a single client can exhaust quota for all callers.
- Per-client limiter map has no eviction and could grow without bound if many unique client keys appear.
- `monitor.Service` dereferences `metricsExporter` without nil checks (panic if unset).
- Kubernetes client retries retry on **any** error (`return err != nil`), including permanent API failures (404/403), wasting time and log noise.

**In scope:**

- Per-client API rate limiting with idle eviction
- Nil-safe metrics updates in monitor service
- Transient-only retry predicate for k8s list/get operations
- Unit tests for limiter fairness, eviction, retry classification, nil exporter

**Out of scope:**

- Configurable rate limits from YAML (existing `security.rate_limit_rps` validation only)
- Full monitor horizontal scaling / watch-based detection
- TrueNAS client retry policy (separate backlog if needed)

## Current vs target behavior

| Area | Current | Target |
|------|---------|--------|
| API rate limit | Global limiter | Per client key with TTL eviction (RemoteAddr unless trusted proxies configured) |
| Limiter map | N/A (single limiter) | Evict idle clients; cap map growth |
| Monitor metrics | Panic if exporter nil | Skip metric updates when nil |
| K8s retries | Retry all errors | Retry timeouts, 429, 5xx, transient net errors only |

## Technical approach

1. Add `go/pkg/api/ratelimit.go` with `perClientRateLimiter` and wire middleware in `server.go`.
2. Add `go/pkg/k8s/retry.go` with `isTransientK8sError` used by all `retry.OnError` call sites in `client.go`.
3. Guard `metricsExporter` in `monitor/service.go` Start/Stop/updateMetrics.
4. Table-driven tests in `*_test.go` files.

## Risk, failure modes, and mitigations

| Risk | Mitigation |
|------|------------|
| Limiter map growth under many unique client keys | Idle TTL eviction + max entry cap with oldest-first trim |
| Hot-path eviction overhead | Evict only when inserting a new client key |
| Spoofed `X-Forwarded-For` bypass | `SetTrustedProxies(nil)` by default; optional `TrustedProxies` CIDR list for ingress |
| Wrong `retry_after` for custom limiter rates | Derive from limiter `rate.Limit` + set HTTP `Retry-After` header |
| Nil metrics exporter panic | Guard Start/Stop/updateMetrics when exporter unset |
| Missing retries on transient K8s errors | `isTransientK8sError` table-tested; permanent errors fail fast |
| Over-aggressive retry suppression | Monitor 429/5xx retry metrics in ops backlog; tune predicate if needed |

## Test strategy

```bash
cd go && go test ./pkg/api/... ./pkg/k8s/... ./pkg/monitor/... -v
cd go && go vet ./...
make go-test && make go-lint
```

## Rollout and backout

**Rollout:** Deploy updated API/monitor binaries. Rate limits apply per source IP; noisy clients no longer starve others.

**Backout:** Revert PR; restores global limiter and permissive retries.
