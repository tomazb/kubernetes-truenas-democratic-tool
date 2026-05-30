# PR 5: Orphan Detection and Validation Debt — Design Spec

**Date:** 2026-05-28  
**Plan item:** [Remediation plan PR 5](../plans/2026-05-28-repo-health-remediation.md)  
**Milestone:** M2 — Contract Integrity

## Problem statement and scope

PR 4 wired orphan API routes to `orphan.Detector`, but several detector and k8s client paths still return optimistic success:

- Snapshot correlation helpers (`hasCorrespondingTrueNASSnapshot`, `hasCorrespondingK8sSnapshot`) always return `true`, so snapshot orphan detection never reports orphans.
- `ValidateRBACPermissions` returns `HasRequiredPermissions: true` without checking the API server.
- Volume-handle matching uses broad substring checks that can false-positive; tests do not cover edge cases.

Operators relying on `/api/v1/orphans` after PR 4 get real PV/PVC signal but misleading snapshot and validation semantics.

**In scope:**

- Implement deterministic K8s ↔ TrueNAS snapshot matching using `snapshotv1.VolumeSnapshot` and `truenas.Snapshot`
- Tighten volume-handle matching where low-risk (document remaining heuristics)
- Replace optimistic `ValidateRBACPermissions` with SelfSubjectAccessReview-based checks for required list/get verbs, or return an explicit error (never silent `true`)
- Add table-driven unit tests in `go/pkg/orphan/*_test.go` and `go/pkg/k8s/*_test.go`
- Update remediation plan tracking

**Out of scope:**

- API route wiring / 501 envelope changes (PR 4, done)
- Background monitor caching, rate limiter, retries (PR 6)
- Full `GetClusterInfo` / CSI listing stubs (backlog unless required for RBAC UX)
- Python CLI parity (later milestone)

## Current behavior vs target behavior

| Aspect | Current | Target |
|--------|---------|--------|
| K8s snapshot without TrueNAS peer | Never flagged (matcher returns `true`) | Flagged when age > threshold and no match |
| TrueNAS snapshot without K8s peer | Never flagged (matcher returns `true`) | Flagged when older than retention and no match |
| Snapshot matching | Stub `interface{}` params | Typed `VolumeSnapshot` / `[]truenas.Snapshot` |
| `ValidateRBACPermissions` | Always `HasRequiredPermissions: true` | SSAR checks per required resource/verb; missing perms listed |
| Volume handle matching | Broad `strings.Contains` on properties | Deterministic rules with table-driven tests |
| Orphan package tests | None (`0%` coverage in CI) | Table-driven tests for matchers and snapshot paths |

## Technical approach

### 1. Snapshot correlation

Match using democratic-csi conventions:

- **K8s → TrueNAS:** Compare `VolumeSnapshot` name/namespace labels and `status.boundVolumeSnapshotContentName` / content source PVC to TrueNAS snapshot `Name`, `Dataset`, and `@` snapshot suffix patterns.
- **TrueNAS → K8s:** Match `truenas.Snapshot.Dataset` + snapshot name to K8s snapshot metadata and volume snapshot content references.

Extract small pure functions (e.g. `snapshotMatchesTrueNAS`, `snapshotMatchesK8s`) for testability.

### 2. RBAC validation

Use `authorizationv1.SelfSubjectAccessReview` for a minimal required set:

- `persistentvolumes`: `list`, `get`
- `persistentvolumeclaims`: `list`, `get`
- `volumesnapshots.snapshot.storage.k8s.io`: `list`, `get` (when snapshot CRD client is configured)

If snapshot client is nil/unavailable, record check as skipped with explicit message rather than `true`.

### 3. Volume-handle matching

Keep existing strategies but add tests for:

- iSCSI `iqn.*:volume` handles
- `pool/dataset/volume` paths
- Non-match when dataset name is substring of unrelated volume
- Empty handle / missing CSI

Refine property `strings.Contains` only if tests prove false positives; otherwise document heuristic in code comment.

### 4. Tests

| Test file | Focus |
|-----------|--------|
| `go/pkg/orphan/detector_test.go` | Snapshot matchers, volume handle extraction, PV orphan age threshold |
| `go/pkg/k8s/client_test.go` | `ValidateRBACPermissions` with fake clientset / SSAR responses |

**Alternatives considered:**

| Alternative | Decision |
|-------------|----------|
| Return 501 from monitor until snapshots fixed | Rejected — PR 5 scope is detector fidelity, not API removal |
| Skip RBAC and document manual check | Rejected — violates API honesty; use real SSAR or explicit error |
| Full VolumeSnapshotContent correlation | Deferred unless needed for match accuracy in tests |

## Risk, failure modes, and mitigations

| Risk | Mitigation |
|------|------------|
| False-positive snapshot orphans due to naming drift | Table tests from documented handle formats; conservative matching |
| SSAR requires in-cluster config | Unit tests with fake authorization interface; integration note in rollout |
| Snapshot CRD not installed | Skip snapshot RBAC check with clear `PermissionChecks` entry |
| Breaking change: non-zero snapshot orphans | Document in rollout; expected honesty fix |

## Test strategy

```bash
cd go && go test ./pkg/orphan/... ./pkg/k8s/... -v -cover
cd go && go test ./... -v
cd go && go vet ./...
make go-lint
```

## Rollout and backout

**Rollout:** Deploy monitor/API builds. Snapshot orphan counts may increase from zero; RBAC validation may report missing permissions where previously always green.

**Backout:** Revert PR; snapshot detection returns to no-op matchers and RBAC returns optimistic pass (not recommended).
