# Product Requirements Document (PRD)
# Kubernetes TrueNAS Democratic Tool

> **How to read this document:** Sections through **§2 Current baseline** describe what is **shipped or partially shipped today**. **§3–§6** are **target roadmap** requirements for later releases. Maintainer detail lives in [ARCHITECTURE.md](ARCHITECTURE.md) and [api-endpoints.md](api-endpoints.md).

## 0. Documentation maturity model

| Label | Meaning |
|-------|---------|
| **Implemented** | Available in the baseline repository with real behavior (not demo/501). |
| **Partial** | Some behavior exists; gaps remain (scaffold CLI, subset of API routes, soft-gate metrics only). |
| **Planned** | Roadmap intent; not production-ready in the baseline repo. |

Do not treat **Planned** or HTTP **501** endpoints as production capabilities. See [§8 Documentation honesty guardrails](#8-documentation-honesty-guardrails).

---

## 1. Executive summary

### 1.1 Product vision

A monitoring and management tool that bridges OpenShift/Kubernetes, TrueNAS Scale, and democratic-csi—giving operators visibility, issue detection, and actionable insights for persistent storage.

### 1.2 Problem statement

Organizations using OpenShift/Kubernetes with TrueNAS Scale via democratic-csi face operational challenges:

- No unified view across Kubernetes, TrueNAS, and CSI layers
- Manual orphan detection wastes capacity and time
- Configuration drift surfaces late, often at failure time
- Limited proactive monitoring and compliance signal for storage practices

### 1.3 Solution overview

An open-source, security-first tool that (at full maturity) will:

- Monitor the storage stack continuously
- Detect and report configuration and orphan issues
- Correlate Kubernetes intent with TrueNAS reality
- Provide actionable recommendations
- Support safe, approval-gated remediation workflows (**Planned**)
- Meet enterprise security and compliance expectations (**Partial** today; see baseline)

**Today (baseline):** orphan detection via Go monitor and API, connectivity validation, Prometheus metrics, and incremental watch/cache/performance guardrails. See [§2 Current baseline](#2-current-baseline-shipped-now).

---

## 2. Current baseline (shipped now)

Canonical operator architecture: [ARCHITECTURE.md](ARCHITECTURE.md) (current section). API route truth: [api-endpoints.md](api-endpoints.md).

### 2.1 Shipped components

| Component | Path | Maturity | Role |
|-----------|------|----------|------|
| Monitor service | `go/cmd/monitor` | **Implemented** | Periodic K8s/TrueNAS scans, orphan metrics, optional watch mode, TTL cache, soft performance budgets |
| API server | `go/cmd/api-server` | **Partial** | REST API; **7 routes implemented**, **15 return HTTP 501** |
| Orphan detector | `go/pkg/orphan/` | **Implemented** | Correlates PV handles with TrueNAS volumes/snapshots |
| Python library | `python/truenas_storage_monitor/` | **Implemented** | Config, clients, monitor module |
| Python CLI | `python/truenas_storage_monitor/cli.py` | **Partial** | Commands exist; **scaffold** output (demo/TODO)—does not call Go API |
| K8s manifests | `deploy/kubernetes/` | **Implemented** | Deployments, services, RBAC, ConfigMap |
| Container images | `deploy/docker/Dockerfile.{monitor,api,cli}` | **Implemented** | Built by CI/release workflow |

**Not in baseline repo:** Web UI, Helm chart, Go controller, Redis cache, gRPC layer, notification integrations, auto-remediation APIs.

### 2.2 What operators can use today

| Capability | How | Maturity |
|------------|-----|----------|
| Orphan detection (aggregate + PV subset) | `GET /api/v1/orphans`, `GET /api/v1/orphans/pvs` | **Partial** |
| List PVs / TrueNAS volumes | `GET /api/v1/resources/pvs`, `GET /api/v1/truenas/volumes` | **Implemented** (list only) |
| Connectivity validation | `GET /api/v1/validate`, `GET /ready` | **Partial** (connectivity; not full config audit) |
| Background monitoring + Prometheus | Go monitor service | **Implemented** |
| TLS to TrueNAS (secure-by-default) | Config `truenas.insecure: false`, optional `ca_file` | **Implemented** |
| Production orphan checks via Python CLI | Not recommended | **Partial**—use Go API |

### 2.3 Baseline performance envelope

Aligned with [ARCHITECTURE.md](ARCHITECTURE.md#supported-cardinality-envelope-current-baseline):

- Initial operating envelope: roughly **1,000** PV/PVC/snapshot objects
- Target scan duration: **under 5 minutes** in representative CI/dev environments
- Performance budgets (`performance.budgets.*`): **warnings and metrics on breach** (soft gate), not hard failure

Roadmap targets (10,000+ volumes, sub-second API at scale) are in [§5.2](#52-performance-metrics).

### 2.4 Baseline compatibility

- OpenShift 4.10+ / Kubernetes 1.24+
- TrueNAS Scale 22.12+
- Democratic-CSI 1.7+
- Python 3.10+ / **Go 1.25.0+** (matches `go/go.mod`; use `GOTOOLCHAIN=auto` or install Go 1.25+ locally)

Go and Python use **different config schemas** (`kubernetes:` vs `openshift:`). See [config-compatibility.md](config-compatibility.md).

### 2.5 Testing and verification (baseline)

- CI gate: `make test-ci-gate`, `make ci-precheck`
- Staging lane: `make test-staging` (requires `TEST_STAGING=true`) — [staging-test-runbook.md](testing/staging-test-runbook.md)
- Release matrix: `make test-release-matrix` — [test-inventory.md](testing/test-inventory.md)

---

## 3. Target roadmap — key features and requirements

The subsections below describe **product goals for future releases**. Each lists **roadmap maturity** (target state) and **target acceptance criteria** unless noted as already partially met in §2.

### 3.1 Core monitoring features

#### 3.1.1 Orphaned resource detection

**Roadmap priority:** P0 | **Baseline maturity:** **Partial**

**Target capabilities:**

- Detect PVs without corresponding TrueNAS volumes
- Identify TrueNAS volumes without Kubernetes PVs
- Find unbound PVCs older than configurable threshold
- Locate stale volume attachments
- Track orphaned snapshots across systems (API routes for PVC/snapshot orphans still **Planned**)

**Target acceptance criteria (GA):**

- Detection accuracy > 99.9%
- Scan completion < 5 minutes for 1,000 volumes (baseline envelope today)
- Near-zero false positives for active resources

#### 3.1.2 Snapshot management

**Roadmap priority:** P0 | **Baseline maturity:** **Planned**

**Target capabilities:**

- Track snapshot count and growth per volume
- Map TrueNAS snapshots to VolumeSnapshot objects
- Monitor snapshot age and retention compliance
- Detect failed or stuck snapshot operations
- Calculate snapshot storage consumption

**Target acceptance criteria (GA):**

- Snapshot tracking lag < 1 minute
- Support for 10,000+ snapshots per cluster
- Automated retention policy enforcement

#### 3.1.3 Configuration validation

**Roadmap priority:** P0 | **Baseline maturity:** **Partial** (`GET /api/v1/validate` connectivity only)

**Target capabilities:**

- Validate StorageClass against TrueNAS capabilities
- Verify CSI driver health across nodes
- Check RBAC completeness
- Validate snapshot class configurations
- Ensure retention policy alignment

**Target acceptance criteria (GA):**

- Coverage of critical configuration items
- Validation runs < 30 seconds
- Clear remediation steps per finding

### 3.2 Analysis and intelligence

#### 3.2.1 Storage usage analytics

**Roadmap priority:** P1 | **Baseline maturity:** **Planned** (all `/api/v1/analysis/*` return 501)

**Target capabilities:** Allocated vs used comparison, thin-provisioning efficiency, pool overcommitment, capacity warnings, savings metrics.

**Target acceptance criteria (GA):** Accuracy within 1% of actual usage; 90+ day trending; predictive alerts 7 days ahead.

#### 3.2.2 Performance monitoring

**Roadmap priority:** P1 | **Baseline maturity:** **Partial** (Prometheus + soft budgets)

**Target capabilities:** Provisioning/mount/snapshot latency, pool metrics, end-to-end latency, anomaly detection.

**Target acceptance criteria (GA):** Metrics every 30s; P95 tracking; automated degradation detection.

### 3.3 Security and compliance

#### 3.3.1 Zero-trust architecture

**Roadmap priority:** P0 | **Baseline maturity:** **Partial** (TLS secure-by-default to TrueNAS; no full mTLS/audit stack)

**Target capabilities:** mTLS everywhere, least-privilege RBAC, no hardcoded credentials, audit logging, secret rotation.

**Target acceptance criteria (GA):** Security audit with zero critical findings; external secret managers; complete audit trail.

#### 3.3.2 Compliance reporting

**Roadmap priority:** P1 | **Baseline maturity:** **Planned**

**Target capabilities:** Encryption status, access control reports, backup verification, retention adherence, custom frameworks.

**Target acceptance criteria (GA):** Reports in < 5 minutes; PDF/HTML/JSON; scheduled delivery.

### 3.4 Integration and automation

#### 3.4.1 Notification systems

**Roadmap priority:** P1 | **Baseline maturity:** **Planned**

**Target capabilities:** Slack/Teams, email, PagerDuty, webhooks, severity routing.

**Target acceptance criteria (GA):** Delivery < 30 seconds; templates; escalation policies.

#### 3.4.2 Auto-remediation

**Roadmap priority:** P2 | **Baseline maturity:** **Planned**

**Target capabilities:** Approved orphan cleanup, snapshot pruning, drift correction, capacity rebalance, safe retries.

**Target acceptance criteria (GA):** Approval workflows; dry-run; rollback; detailed logs.

### 3.5 User interfaces

#### 3.5.1 Command line interface

**Roadmap priority:** P0 | **Baseline maturity:** **Partial** (scaffold)

**Target capabilities:** Rich output formats, interactive troubleshooting, shell completion, scriptable operations with live data.

**Target acceptance criteria (GA):** Commands < 10 seconds; consistent syntax; comprehensive help.

#### 3.5.2 REST API

**Roadmap priority:** P0 | **Baseline maturity:** **Partial** (7/22 routes; per-client rate limiting shipped)

**Target capabilities:** OpenAPI 3.0, token auth, webhooks, WebSocket streaming.

**Target acceptance criteria (GA):** p95 < 500ms for implemented routes at baseline scale; high availability; backward compatibility.

#### 3.5.3 Web dashboard

**Roadmap priority:** P2 | **Baseline maturity:** **Planned**

**Target capabilities:** Live metrics, trends, alerts UI, reports, mobile-responsive WCAG 2.1 AA.

**Target acceptance criteria (GA):** Page load < 2s; 100+ concurrent users.

---

## 4. Target roadmap — technical requirements

### 4.1 Architecture (target)

- Hybrid Go/Python (**Implemented** layout; full microservices/HA **Planned**)
- Cloud-native design principles
- Horizontal scaling and HA deployment options (**Planned**—see [ARCHITECTURE.md](ARCHITECTURE.md) target sections)

### 4.2 Performance (target)

| Scope | Target |
|-------|--------|
| **Baseline (today)** | ~1,000 objects; scan < 5 min; soft performance budgets |
| **Roadmap (GA)** | 10,000+ volumes; 1M+ metrics/min; sub-second API at scale; < 500MB RAM base |

### 4.3 Compatibility (target)

Same as [§2.4](#24-baseline-compatibility); maintain version matrix as integrations evolve.

### 4.4 Deployment options

| Option | Baseline maturity |
|--------|-------------------|
| Standalone binary | **Implemented** |
| Container images | **Implemented** |
| Kubernetes manifests (`deploy/kubernetes/`) | **Implemented** |
| Kubernetes Operator | **Planned** |
| Helm chart | **Planned** (backlog) |

---

## 5. Success metrics

### 5.1 Adoption metrics (roadmap)

Community and adoption goals for post-baseline releases—not current release gates:

- 100+ production deployments in 6 months
- 1,000+ GitHub stars in 12 months
- 50+ active contributors
- 10+ enterprise adoptions

### 5.2 Performance metrics

| Metric | Baseline (today) | Roadmap (GA) |
|--------|------------------|--------------|
| Detection accuracy | Improving; not yet 99.9% gate | > 99.9% |
| Scan time (1k volumes) | Target < 5 min (soft budgets) | Same + scale to 10k+ |
| False positive rate | Operator validation | < 1% |
| Automated resolution | N/A | 90%+ where auto-remediation ships |

### 5.3 Business impact (roadmap)

Targets for mature product adoption: fewer storage incidents, better utilization, faster MTTR, measurable cost savings per deployment.

---

## 6. Release plan (roadmap milestones)

> **Note:** Milestones below are **sequencing intent**, not a claim about the current repository version. Baseline behavior is described in [§2](#2-current-baseline-shipped-now).

### 6.1 MVP (v0.1.0) — baseline themes

- Core monitoring and orphan detection (Go monitor + partial API) — **Partial / Implemented**
- CLI interface — **Partial** (scaffold)
- Basic notifications — **Planned**

### 6.2 Beta (v0.5.0)

- Snapshot management APIs and workflows — **Planned**
- REST API breadth (beyond 7 routes) — **Partial** today
- Integration and staging test depth — **In progress** ([test-inventory.md](testing/test-inventory.md))

### 6.3 GA (v1.0.0)

- Auto-remediation — **Planned**
- Web dashboard — **Planned**
- Enterprise hardening (SLOs, full validation, compliance reports)

### 6.4 Future releases

- v1.1.0 — ML-based predictions (**Planned**)
- v1.2.0 — Multi-cluster support (**Planned**)
- v2.0.0 — Plugin architecture (**Planned**)

---

## 7. Target users

### 7.1 Primary users

1. **Platform engineers** — unified monitoring; today: Go API/monitor orphan paths
2. **Storage administrators** — capacity and orphan visibility; today: metrics + API lists
3. **DevOps teams** — reliable storage operations; today: validation and detection signal

### 7.2 Secondary users

Security, finance, and application teams gain full value as compliance, cost, and performance features move from **Planned** to **Implemented**.

---

## 8. Documentation honesty guardrails

For contributors and technical writers:

1. Label every user-visible capability **Implemented**, **Partial**, or **Planned** in PRD and operator docs.
2. Do not describe HTTP **501** routes or scaffold CLI output as production-ready.
3. Keep [api-endpoints.md](api-endpoints.md) synchronized with `go/pkg/api/server.go`.
4. Prefer linking to [ARCHITECTURE.md](ARCHITECTURE.md) for design detail rather than duplicating target diagrams in PRD.
5. When roadmap scope changes, update this PRD and the relevant spec under `docs/superpowers/specs/` before implementation.

Related docs: [AGENTS.md](../AGENTS.md), [config-compatibility.md](config-compatibility.md), [remediation plan](superpowers/plans/2026-05-28-repo-health-remediation.md).

---

## 9. Risks and mitigations

### 9.1 Technical risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| API breaking changes | High | Version detection, compatibility matrix |
| Performance at scale | Medium | Caching, watch reconcile, performance budgets (baseline); horizontal scaling (roadmap) |
| Security vulnerabilities | High | Audits, dependency scanning, TLS defaults |

### 9.2 Adoption risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Complex deployment | Medium | Clear docs; Helm/operator later |
| Learning curve | Low | Examples, runbooks |
| Enterprise requirements | Medium | Modular architecture; optional support |

---

## 10. Open source strategy

### 10.1 License

- Apache 2.0 License
- CLA for contributors when required
- No proprietary dependencies in baseline

### 10.2 Community building

- Public roadmap (this PRD + remediation plan)
- Regular releases and contribution guidelines
- Code of conduct

### 10.3 Governance

Maintainer, security, and documentation responsibilities per [AGENTS.md](../AGENTS.md) and project norms.

---

## 11. Competitive analysis

### 11.1 Existing solutions

| Solution | Strengths | Weaknesses | Our advantage |
|----------|-----------|------------|---------------|
| Manual scripts | Free, customizable | Not scalable | Purpose-built correlation |
| Generic monitoring | Mature | Not storage-specific | K8s + TrueNAS + CSI focus |
| Vendor tools | Integrated | Lock-in, cost | Open source, flexible |

### 11.2 Unique value proposition (full vision)

1. Purpose-built for OpenShift/Kubernetes + TrueNAS + democratic-csi
2. Security-first design (baseline: TLS defaults; roadmap: full zero-trust)
3. Idempotent, safe automation (**Planned** remediation)
4. Hybrid Go/Python architecture (**Implemented** split)
5. Open source with a path to enterprise depth

---

## 12. Resource requirements (planning)

### 12.1 Development team (illustrative)

- Senior engineers (Go/Python), DevOps, technical writing, part-time product ownership

### 12.2 Timeline (roadmap)

- Phased delivery via remediation plan and Stage 2–5 items—not fixed calendar commitments in baseline repo

### 12.3 Budget

Open-source contribution model; infrastructure primarily GitHub Actions; optional support tiers **Planned**.
