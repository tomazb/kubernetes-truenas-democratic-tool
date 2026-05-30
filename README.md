# Kubernetes TrueNAS Democratic Tool

[![CI Pipeline](https://github.com/tomazb/kubernetes-truenas-democratic-tool/actions/workflows/ci.yml/badge.svg)](https://github.com/tomazb/kubernetes-truenas-democratic-tool/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/tomazb/kubernetes-truenas-democratic-tool)](https://goreportcard.com/report/github.com/tomazb/kubernetes-truenas-democratic-tool)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A monitoring and analysis tool for OpenShift/Kubernetes clusters using TrueNAS Scale storage via democratic-csi.

## Overview

This tool correlates Kubernetes storage objects (PVs, PVCs, snapshots) with TrueNAS datasets and volumes to detect orphaned resources, configuration drift, and storage efficiency risks.

### Maturity snapshot

| Area | Status | Notes |
|------|--------|-------|
| Go monitor service | **Implemented** | Periodic scans, Prometheus metrics, orphan detection |
| Go API server | **Partial** | 7 routes implemented; 15 return HTTP 501 — see [API endpoint maturity](docs/api-endpoints.md) |
| Go orphan detector | **Implemented** | Real K8s/TrueNAS correlation (PR 5) |
| Python library | **Implemented** | Config, K8s/TrueNAS clients, monitor module |
| Python CLI (`truenas-monitor`) | **Scaffold** | Commands exist; output is demo/TODO — use Go API for real orphan data |
| Kubernetes manifests | **Implemented** | `deploy/kubernetes/` |
| Helm chart | **Planned** | Tracked in [backlog](docs/superpowers/backlog.md) (BL-20260530-helm-chart-release) |

### Key capabilities

**Implemented today:**

- Orphaned resource detection via Go API (`GET /api/v1/orphans`, `/api/v1/orphans/pvs`)
- Kubernetes and TrueNAS connectivity validation
- TLS secure-by-default for TrueNAS (with explicit dev-only insecure opt-in)
- Per-client API rate limiting and transient-error retries (PR 6)

**Planned (not yet shipped):**

- Full Python CLI implementations (analyze, report, validate with live data)
- Snapshot orphan API routes, analysis/trends endpoints
- Auto-remediation, Web UI, Helm packaging

## Architecture

Hybrid Go/Python layout for performance-sensitive runtime paths and flexible CLI/analysis:

- **Go** — Monitor service (`go/cmd/monitor`) and REST API server (`go/cmd/api-server`)
- **Python** — Library and CLI scaffold (`python/truenas_storage_monitor/`)

See [Architecture](docs/ARCHITECTURE.md) for current vs planned system design.

## Quick Start

### Prerequisites

- Kubernetes or OpenShift cluster with democratic-csi
- TrueNAS Scale with API access
- Go 1.24+ (for Go services)
- Python 3.10+ (for library/CLI)

### Development setup

```bash
git clone https://github.com/tomazb/kubernetes-truenas-democratic-tool.git
cd kubernetes-truenas-democratic-tool
make dev-setup
source venv/bin/activate
make test-all
make go-build
```

Binaries are written to `bin/monitor` and `bin/api-server`.

### Run the Go API server

Use the Go config schema (`kubernetes:` key). See [config compatibility](docs/config-compatibility.md) and [config.go.example](config.go.example).

```bash
export TRUENAS_USERNAME=admin
export TRUENAS_PASSWORD='change-me'
./bin/api-server -config config.go.example -port 8080
curl -s http://localhost:8080/health
curl -s http://localhost:8080/api/v1/orphans
```

### Deploy to Kubernetes

Helm is not shipped yet. Apply the bundled manifests:

```bash
kubectl apply -f deploy/kubernetes/
```

### Python CLI (scaffold)

Install from the repo (PyPI publish is not part of the baseline):

```bash
cd python && pip install -e ".[dev,cli]"
truenas-monitor --help
```

CLI commands (`orphans`, `analyze`, `report`, `validate`) print demo or placeholder output. For production orphan checks, use the Go API.

| CLI command | Status |
|-------------|--------|
| `orphans` | Scaffold (demo table) |
| `analyze` | Scaffold (hardcoded summary) |
| `report` | Scaffold (no file written) |
| `validate` | Scaffold (hardcoded pass/fail) |
| `monitor` | Scaffold (sleep loop) |

## Configuration

Go services and Python use **different YAML schemas**. Do not assume one file works for both.

| Runtime | Example file | Cluster key |
|---------|--------------|-------------|
| Go monitor / API | [config.go.example](config.go.example) | `kubernetes:` |
| Python CLI / library | [config.yaml.example](config.yaml.example) | `openshift:` |

Full key mapping: [docs/config-compatibility.md](docs/config-compatibility.md).

Minimal Go example:

```yaml
kubernetes:
  kubeconfig: ~/.kube/config
  namespace: democratic-csi

truenas:
  url: https://truenas.example.com
  username: admin
  password: ${TRUENAS_PASSWORD}
  insecure: false
```

## Development

This project follows TDD practices where tests exist for changed behavior:

```bash
make test-unit      # Go + Python unit tests
make test-all       # Full test suites
make go-test-coverage
make lint-all
make ci-precheck    # Validate CI/Makefile path references
```

Build containers:

```bash
make docker-build-all   # monitor, api, cli images
```

## Contributor governance

[AGENTS.md](AGENTS.md) is the canonical playbook for contributors and automation agents.

- Standards, verification gates, and PR checklists live in `AGENTS.md`.
- Each implementation PR needs a design spec under `docs/superpowers/specs/` before coding starts.
- Active remediation work: [remediation plan](docs/superpowers/plans/2026-05-28-repo-health-remediation.md); out-of-scope items: [backlog](docs/superpowers/backlog.md).

## Documentation

- [AGENTS.md](AGENTS.md) — Contributor playbook, PR policy, verification gates
- [Architecture](docs/ARCHITECTURE.md) — Current (shipped) vs target (planned) design
- [Config compatibility](docs/config-compatibility.md) — Go vs Python YAML schemas
- [API endpoint maturity](docs/api-endpoints.md) — Implemented vs 501 routes (7 implemented, 15 not implemented)
- [PRD](docs/PRD.md) — Product requirements and roadmap
- [Remediation plan](docs/superpowers/plans/2026-05-28-repo-health-remediation.md) — Phased repo health work
- [Backlog](docs/superpowers/backlog.md) — Canonical out-of-scope intake

## Security

- TLS certificate verification enabled by default for TrueNAS (`truenas.insecure: false`)
- Optional custom CA bundle (`truenas.ca_file`)
- No credentials in logs
- Security scans via GitHub Actions

## Contributing

Start with [AGENTS.md](AGENTS.md) for PR requirements, then [CONTRIBUTING.md](CONTRIBUTING.md) for setup and workflow.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/your-feature`)
3. Write tests for changed behavior
4. Commit with sign-off (`git commit -s -m '...'`)
5. Open a Pull Request

## License

Apache License 2.0 — see [LICENSE](LICENSE).

## Support

- **Issues**: [GitHub Issues](https://github.com/tomazb/kubernetes-truenas-democratic-tool/issues)
- **Discussions**: [GitHub Discussions](https://github.com/tomazb/kubernetes-truenas-democratic-tool/discussions)
- **Security**: [SECURITY.md](.github/SECURITY.md)

## Roadmap

See [docs/PRD.md](docs/PRD.md) for the full roadmap. Upcoming themes include Grafana integration, auto-remediation, and multi-cluster support — none are shipped in the baseline.

## Acknowledgments

- OpenShift/Kubernetes community
- TrueNAS Scale team
- democratic-csi project
