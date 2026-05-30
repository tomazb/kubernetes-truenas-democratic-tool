# PR 4: API Honesty and Endpoint Maturity Cleanup — Design Spec

**Date:** 2026-05-28  
**Plan item:** [Remediation plan PR 4](../plans/2026-05-28-repo-health-remediation.md)  
**Milestone:** M2 — Contract Integrity

## Problem statement and scope

The Go API server exposes orphan endpoints that return HTTP 200 with empty orphan arrays and `total_orphans: 0`, implying a successful scan with no findings. That is misleading when no detection logic runs. Fourteen other routes already return HTTP 501, but with an inconsistent JSON envelope.

Additionally, `NewServer` constructs a `monitor.Service` that is never started or used by handlers, while `orphan.Detector` already implements synchronous detection used by the monitor binary.

**In scope:**

- Wire `/api/v1/orphans` and `/api/v1/orphans/pvs` to `orphan.Detector.DetectOrphanedResources`
- Remove unused `monitor.Service` wiring from the API server
- Standardize HTTP 501 JSON envelope for unimplemented routes
- Add `go/pkg/api/server_test.go` handler contract tests
- Add minimal API maturity documentation (`docs/api-endpoints.md`)
- Update remediation plan tracking

**Out of scope:**

- Detector matching fidelity (snapshot hardcoding, RBAC stubs) → PR-5
- Implementing remaining 501 list/analysis/report endpoints → Stage 3 / later PRs
- Full README endpoint overhaul → PR-8
- Background monitor scans / rate-limiter refactor → PR-6

## Current behavior vs target behavior

| Aspect | Current | Target |
|--------|---------|--------|
| `GET /api/v1/orphans` | HTTP 200, empty arrays, no detection | HTTP 200 with real `orphan.DetectionResult` payload |
| `GET /api/v1/orphans/pvs` | Lists PVs but always empty `orphaned_pvs` | HTTP 200 with detector PV subset |
| Unimplemented routes | HTTP 501, `{"message":"not implemented"}` | HTTP 501 with `error`, `message`, `endpoint` fields |
| API server monitor wiring | `monitor.Service` created, unused | Removed; orphan detection via `orphan.Detector` per request |
| Handler tests | None | Contract tests for orphans, 501 routes, error paths |

## Technical approach

1. Remove `monitor.NewService` from `api.NewServer`.
2. Add helper to parse `age_threshold` query (default `24h`) and run detection via per-request `orphan.NewDetector` (read-only `DryRun: true`).
3. Map `DetectionResult` to JSON including `total_orphans` count.
4. Replace 501 one-liners with shared `notImplemented(c, endpoint)` helper.
5. Add httptest-based tests with stub `k8s.Client` / `truenas.Client` implementations.
6. Document endpoint maturity in `docs/api-endpoints.md`.

**Alternatives considered:**

| Alternative | Decision |
|-------------|----------|
| Return 501 for orphan routes until PR-5 | Rejected — user chose wire to detector now |
| Use `monitor.Service.GetLastScanResult()` | Rejected — requires background Start + metrics exporter |
| Mutate shared detector config per request | Rejected — per-request construction avoids shared state |

## Risk, failure modes, and mitigations

| Risk | Mitigation |
|------|------------|
| Detector creates its own logger internally | Accept for PR-4; refactor in PR-5 if needed |
| Synchronous scan latency on large clusters | Document behavior; cache/background scan deferred to PR-6 |
| Clients expecting empty orphan arrays | Document breaking semantic fix in rollout notes |
| Per-request detector overhead | Acceptable for honesty PR; optimize if profiling shows need |

## Test strategy

| Test | Purpose |
|------|---------|
| `TestListOrphansHandler_ReturnsDetectorResults` | Non-empty orphan signal when stubs provide aged unbound PV |
| `TestListOrphansHandler_InvalidAgeThreshold_Returns400` | Bad `age_threshold` rejected |
| `TestListOrphanedPVsHandler_ReturnsPVSubsetOnly` | PV endpoint returns PV orphan subset |
| `TestListOrphansHandler_DetectorError_Returns500` | Upstream list failure surfaces as 500 |
| `TestNotImplementedRoutes_Return501WithStandardEnvelope` | All stub routes return standardized 501 JSON |

**Validation commands:**

```bash
cd go && go test ./pkg/api/... -v -cover
cd go && go test ./... -v
cd go && go vet ./...
make go-lint
```

## Rollout and backout

**Rollout:** Deploy updated API server. Orphan endpoints now perform real synchronous detection; clients may see non-zero orphan counts where previously always zero.

**Backout:** Revert PR; orphan endpoints return to placeholder empty success responses (not recommended).
