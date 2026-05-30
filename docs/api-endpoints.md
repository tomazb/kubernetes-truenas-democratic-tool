# API Endpoint Maturity

This document describes the current maturity of HTTP routes exposed by the Go API server (`cmd/api-server`).

Orphan detection runs synchronously on each request for implemented orphan routes. Detection quality continues to improve in PR-5 (detector fidelity).

## Infrastructure

| Route | Status | Notes |
|-------|--------|-------|
| `GET /health` | Implemented | Process liveness |
| `GET /ready` | Implemented | Kubernetes + TrueNAS connectivity |

## Orphan detection

| Route | Status | Notes |
|-------|--------|-------|
| `GET /api/v1/orphans` | Implemented | Sync `orphan.Detector`; query: `namespace`, `age_threshold` (default from config); response includes `snapshot_retention` |
| `GET /api/v1/orphans/pvs` | Implemented | PV orphan subset; query: `age_threshold` |
| `GET /api/v1/orphans/pvcs` | Not implemented (501) | |
| `GET /api/v1/orphans/snapshots` | Not implemented (501) | |

## Resources

| Route | Status | Notes |
|-------|--------|-------|
| `GET /api/v1/resources/pvs` | Implemented | Lists Kubernetes PVs |
| `GET /api/v1/resources/pvcs` | Not implemented (501) | |
| `GET /api/v1/resources/snapshots` | Not implemented (501) | |
| `GET /api/v1/resources/storageclasses` | Not implemented (501) | |

## TrueNAS

| Route | Status | Notes |
|-------|--------|-------|
| `GET /api/v1/truenas/volumes` | Implemented | Lists TrueNAS volumes |
| `GET /api/v1/truenas/snapshots` | Not implemented (501) | |
| `GET /api/v1/truenas/pools` | Not implemented (501) | |
| `GET /api/v1/truenas/info` | Not implemented (501) | |

## Analysis

| Route | Status |
|-------|--------|
| `GET /api/v1/analysis` | Not implemented (501) |
| `GET /api/v1/analysis/usage` | Not implemented (501) |
| `GET /api/v1/analysis/trends` | Not implemented (501) |

## Validation

| Route | Status | Notes |
|-------|--------|-------|
| `GET /api/v1/validate` | Implemented | Connectivity checks |
| `GET /api/v1/validate/config` | Not implemented (501) | |
| `GET /api/v1/validate/connectivity` | Not implemented (501) | |

## Reports

| Route | Status |
|-------|--------|
| `GET /api/v1/reports/summary` | Not implemented (501) |
| `GET /api/v1/reports/detailed` | Not implemented (501) |

## Unimplemented response contract

Routes marked **Not implemented** return HTTP 501 with:

```json
{
  "error": "not_implemented",
  "message": "endpoint not implemented",
  "endpoint": "/api/v1/..."
}
```
