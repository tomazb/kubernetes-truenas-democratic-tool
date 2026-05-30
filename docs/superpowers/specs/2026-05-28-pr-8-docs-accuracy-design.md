# PR 8: Documentation Accuracy Refresh — Design Spec

**Date:** 2026-05-28  
**Plan item:** [Remediation plan PR 8](../plans/2026-05-28-repo-health-remediation.md)  
**Milestone:** M3 — Operational Readiness

## Problem statement and scope

Operator and contributor documentation describes components, commands, and paths that are not shipped in the baseline codebase. README and ARCHITECTURE present aspirational features (Helm chart, Go controller, Web UI, full CLI workflows) as if they exist today. Config examples use a single schema while Go and Python runtimes expect different YAML keys. Placeholder GitHub URLs (`yourusername`) create dead ends for new contributors.

**In scope:**

- Align `README.md`, `docs/ARCHITECTURE.md`, and `CONTRIBUTING.md` with shipped baseline (PRs 1–7)
- Add config compatibility documentation and separate Go/Python config examples
- Cross-link existing `docs/api-endpoints.md`; add CLI command maturity summary
- Fix placeholder URLs in README, CONTRIBUTING, and `python/pyproject.toml` `[project.urls]`
- Update remediation plan tracking for PR 8 (and PR 6 status housekeeping)

**Out of scope:**

- `docs/PRD.md` roadmap rewrite
- `.kiro/` specs, `docs/go-vs-python-comparison.md` (design rationale, not operator docs)
- Implementing Helm chart, controller, Web UI, or full CLI logic (Stage 3+ / backlog)
- Docs-lint CI automation (Stage 5 governance)
- Raising Python coverage to 90% (backlog BL-20260528-python-coverage-gate)

## Current vs target behavior

| Area | Current | Target |
|------|---------|--------|
| README quick start | Helm chart path, PyPI install, CLI as production-ready | `make dev-setup`, `kubectl apply`, Go API for orphan detection; CLI labeled scaffold |
| README architecture | Lists Go controller as shipped | Lists only `monitor` + `api-server` |
| ARCHITECTURE.md | Entire doc reads as production system | **Current (shipped)** section first; existing content under **Target (planned)** |
| CONTRIBUTING | `make test-coverage`, 90% coverage, `yourusername` URLs | Valid make targets, 70% interim gate, real repo URLs |
| Config examples | Single Python-oriented `config.yaml.example` | Python example + `config.go.example`; `docs/config-compatibility.md` |
| API maturity | Document exists but under-linked | Prominent README/ARCHITECTURE links + summary counts |
| Go version in docs | 1.21+ | 1.24+ (matches `go/go.mod`) |

## Technical approach

1. Create PR-8 spec (this document) before doc edits.
2. Work in branch `docs/pr-8-docs-accuracy` / worktree `.worktrees/pr-8-docs-accuracy`.
3. Rewrite README quick start, maturity section, and feature claims against shipped components table.
4. Prepend current-state architecture to `docs/ARCHITECTURE.md`; relabel existing sections as planned/target.
5. Add `docs/config-compatibility.md` and `config.go.example`; revise `config.yaml.example` header and trim/mark unimplemented sections.
6. Fix CONTRIBUTING commands, coverage gate, and URLs.
7. Update `python/pyproject.toml` project URLs only (no author identity changes).

## Shipped baseline (source of truth for edits)

| Component | Status |
|-----------|--------|
| Go monitor (`go/cmd/monitor`) | Shipped |
| Go API server (`go/cmd/api-server`) | Shipped (partial routes; see api-endpoints.md) |
| Go orphan detector | Shipped |
| Python library | Shipped |
| Python CLI | Scaffold (demo/TODO output) |
| K8s manifests (`deploy/kubernetes/`) | Shipped |
| Docker images (`deploy/docker/Dockerfile.{monitor,api,cli}`) | Shipped |
| Helm chart, Go controller, Web UI, Redis/gRPC/ML | Not shipped |

## Risk, failure modes, and mitigations

| Risk | Mitigation |
|------|------------|
| Over-scoping into PRD/roadmap rewrite | Limit to operator-facing files listed in plan |
| Removing useful future architecture | Keep under Target/planned section with clear banner |
| Config doc drift from code | Derive mapping from `go/pkg/config/config.go` and `python/truenas_storage_monitor/config.py` |
| Docs-only PR skips verification | Manual contributor walkthrough + `make ci-precheck` |

## Test strategy

```bash
make ci-precheck
make go-build
# Manual walkthrough (document in PR notes):
# - make dev-setup
# - make test-all
# - Verify README paths/commands exist
# - No yourusername / ./charts/ / cmd/controller in touched docs
```

## Rollout and backout

**Rollout:** Merge PR; documentation reflects baseline maturity; M3 exit criteria met.

**Backout:** Revert PR; restores prior (inaccurate) documentation.
