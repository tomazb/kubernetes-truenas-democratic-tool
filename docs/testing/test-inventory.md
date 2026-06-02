# Test Inventory and Coverage Map

This document maps shipped repository capabilities to test lanes, commands, and ownership.

## Lane Definitions

- `lane1-ci-gate`: fast deterministic tests for PR merge blocking
- `lane2-staging`: real Kubernetes + TrueNAS validation in staging
- `lane3-release-matrix`: deep performance/resilience/security validation

## Capability Coverage Matrix

| Surface | Capability | Status | Current Tests | Primary Command(s) | Lane | Owner |
|---|---|---|---|---|---|---|
| Go API | Health/readiness endpoints | Covered | `go/pkg/api/server_test.go` | `make go-test` | lane1-ci-gate | Go maintainers |
| Go API | Orphan detection (`/api/v1/orphans*`) | Covered | `go/pkg/api/server_test.go`, `go/pkg/orphan/detector_test.go` | `make go-test` | lane1-ci-gate | Go maintainers |
| Go API | Unimplemented route honesty (`501`) | Covered | `go/pkg/api/server_test.go` | `make go-test` | lane1-ci-gate | Go maintainers |
| Go runtime | Monitor service loop behavior | Covered | `go/pkg/monitor/service_test.go`, `go/pkg/reconcile/*_test.go` | `make go-test` | lane1-ci-gate | Go maintainers |
| Go runtime | Inventory cache consistency | Covered | `go/pkg/inventorycache/*_test.go` | `make go-test` | lane1-ci-gate | Go maintainers |
| Go runtime | K8s retry/failure behavior | Covered | `go/pkg/k8s/retry_test.go`, `go/pkg/k8s/client_test.go` | `make go-test` | lane1-ci-gate | Go maintainers |
| Go runtime | TLS defaults and TrueNAS client behavior | Covered | `go/pkg/truenas/tls_test.go`, `go/pkg/truenas/client_test.go` | `make go-test` | lane1-ci-gate | Go maintainers |
| Metrics/obs | Exporter and logging baseline behavior | Covered | `go/pkg/metrics/exporter_test.go`, `go/pkg/logging/logger_test.go` | `make go-test` | lane1-ci-gate | Go maintainers |
| Python lib | Config loading/validation | Covered | `python/tests/unit/test_config.py` | `make python-test` | lane1-ci-gate | Python maintainers |
| Python lib | K8s/TrueNAS client wrappers | Covered | `python/tests/unit/test_k8s_client.py`, `python/tests/unit/test_truenas_client.py` | `make python-test` | lane1-ci-gate | Python maintainers |
| Python lib | Monitor, reconcile, cache, observability modules | Covered | `python/tests/unit/test_monitor.py`, `test_reconcile.py`, `test_inventory_cache.py`, `test_observability.py` | `make python-test` | lane1-ci-gate | Python maintainers |
| Python CLI scaffold | Watch command contract | Covered (scaffold semantics) | `python/tests/unit/test_cli_watch.py` | `make python-test` | lane1-ci-gate | Python maintainers |
| Python package hygiene | Import side-effect baseline | Covered | `python/tests/unit/test_import_hygiene.py` | `make python-test` | lane1-ci-gate | Python maintainers |
| Cross-stack | Lint/static correctness | Covered | CI lint jobs | `make lint-all` | lane1-ci-gate | Repo maintainers |
| Cross-stack | Security scanner baseline | Covered | Trivy/Bandit/Safety jobs | `make security-scan` | lane1-ci-gate | Repo maintainers |
| Kubernetes + TrueNAS integration | Real-system preflight and environment contract | Added in this plan | `python/tests/staging/test_staging_preflight.py` | `make test-staging` | lane2-staging | Platform/SRE |
| End-to-end correlation | Real orphan/snapshot reconciliation in staging | Planned harness | staging test suite expansion | `make test-staging` | lane2-staging | Platform/SRE |
| Performance budgets | API latency/reconcile/cardinality budget checks | Partial (soft-gate benchmark present) | `scripts/perf-budget-benchmark.sh` and release matrix tests | `make test-release-matrix` | lane3-release-matrix | Go maintainers |
| Resilience matrix | Timeout/throttle/failure semantics | Added baseline checks | `python/tests/regression/test_resilience_matrix.py` | `make test-release-matrix` | lane3-release-matrix | Go + Python maintainers |
| Security regression matrix | TLS defaults, auth failure contract, redaction | Added baseline checks + existing TLS tests | `python/tests/regression/test_security_matrix.py`, `go/pkg/truenas/tls_test.go` | `make test-release-matrix` | lane3-release-matrix | Security/repo maintainers |

## Gap Register (Current)

- Full staging end-to-end data lifecycle tests still require environment secrets and fixtures to be provisioned in staging.
- Performance budget automation currently mixes benchmark script output and release matrix checks; lane3 thresholds should be tuned with observed staging baselines.
- Python CLI non-watch commands are still scaffolded; tests intentionally validate scaffold semantics instead of production data behavior.

## Canonical Execution Commands

- `make test-unit`
- `make test-integration`
- `make test-security`
- `make test-staging`
- `make test-release-matrix`
- `make test-matrix`
