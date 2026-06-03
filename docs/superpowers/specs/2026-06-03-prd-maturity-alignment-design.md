# PRD Maturity Alignment — Design Spec

**Date:** 2026-06-03  
**Plan item:** PRD alignment (deferred from [PR 8](2026-05-28-pr-8-docs-accuracy-design.md))  
**Milestone:** M3 — Operational Readiness (product documentation honesty)

## Problem statement and scope

`docs/PRD.md` describes a full enterprise product (P0 CLI/API, real-time snapshots, zero-trust everywhere, auto-remediation) without distinguishing what is **shipped today** from **roadmap intent**. Operator docs ([ARCHITECTURE.md](../../ARCHITECTURE.md), [README.md](../../../README.md), [api-endpoints.md](../../api-endpoints.md)) were aligned in PR 8; PRD was explicitly out of scope.

**In scope:**

- Restructure `docs/PRD.md` with maturity model, current baseline, and target roadmap sections
- Tag roadmap features with **Implemented**, **Partial**, or **Planned**
- Cross-link maintainer/operator docs; light README and AGENTS.md pointer updates
- Fix stale PR reference in `docs/api-endpoints.md`
- Add execution note to remediation plan

**Out of scope:**

- ARCHITECTURE.md rewrite (already current)
- Code, API implementation, CLI logic
- Docs-lint CI (Stage 5 backlog)

## Current vs target behavior

| Area | Before | After |
|------|--------|-------|
| PRD structure | Single vision doc | §0 maturity + §2 baseline + §3–6 roadmap |
| Feature claims | Implied shipped | Explicit Partial/Planned per feature |
| Performance | 10k volumes / sub-second API as norm | Baseline envelope (~1k objects, &lt;5 min scan) + GA targets separated |
| Release plan | Reads as current schedule | Labeled roadmap milestones |
| Go version | 1.21+ | 1.24+ |

## Shipped baseline (source of truth)

| Component | Maturity |
|-----------|----------|
| Go monitor | Implemented |
| Go API (7 routes) | Partial |
| Go orphan detector | Implemented |
| Python library | Implemented |
| Python CLI | Partial (scaffold) |
| K8s manifests, Docker images | Implemented |
| Helm, controller, Web UI, Redis/gRPC | Planned |

See [api-endpoints.md](../../api-endpoints.md) for route-level truth.

## Technical approach

1. Write this spec.
2. Rewrite `docs/PRD.md` per plan structure.
3. Update README, AGENTS.md, remediation plan, api-endpoints intro line.

## Test strategy

```bash
make ci-precheck
```

Manual: PRD baseline table vs ARCHITECTURE shipped section; API counts vs api-endpoints.md.

## Rollout and backout

**Rollout:** Merge docs PR; PRD becomes product-facing maturity source linked from README.

**Backout:** Revert PR; restores pre-alignment PRD.
