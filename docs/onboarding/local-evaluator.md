# Local evaluator onboarding (appendix)

Use this path to evaluate behavior locally before or alongside in-cluster deployment.

This guide focuses on low-risk checks and implemented surfaces only.

## Scope and boundaries

- Primary local target is the **Go API server** and optional **Go monitor**.
- Python CLI commands exist, but several remain scaffold/demo outputs.
- Do not treat local output as an authorization to perform destructive storage changes.

## Choose the right config file first

Go and Python use different schemas:

- Go API/monitor: `config.go.example` (`kubernetes:` + `truenas:`)
- Python library/CLI: `config.yaml.example` (`openshift:` + `monitoring:`)

If your goal is production-grade orphan checks today, use the Go API path.
See [config compatibility](../config-compatibility.md).

## Prerequisites

- Go 1.25.0+
- Access to a non-production cluster (or controlled test context)
- Reachable TrueNAS endpoint with valid credentials

## Build binaries

```bash
make go-build
```

Expected binaries:

- `bin/api-server`
- `bin/monitor`

## Configure environment

Use environment variables for secrets:

```bash
export TRUENAS_USERNAME='service-account-name'
export TRUENAS_PASSWORD='replace-me'
```

Avoid shell history leakage in shared environments (use secure secret injection when possible).

## Run local API server

```bash
./bin/api-server -config config.go.example -port 8080
```

In another terminal:

```bash
curl -sS http://127.0.0.1:8080/health
curl -sS http://127.0.0.1:8080/ready
curl -sS http://127.0.0.1:8080/api/v1/orphans
curl -sS http://127.0.0.1:8080/api/v1/orphans/pvs
curl -sS http://127.0.0.1:8080/api/v1/resources/pvs
curl -sS http://127.0.0.1:8080/api/v1/truenas/volumes
curl -sS http://127.0.0.1:8080/api/v1/validate
```

## Optional: run local monitor

```bash
./bin/monitor -config config.go.example
```

Observe logs and metrics exposure based on your config (`metrics.enabled`, `metrics.port`, `metrics.path`).

## Expected route behavior during local evaluation

- Implemented endpoints return normal payloads.
- Many roadmap endpoints return HTTP 501.
- 501 is expected for unimplemented routes and indicates maturity boundaries, not runtime instability.

Full route matrix: [api-endpoints.md](../api-endpoints.md).

## Python CLI caveat (important)

`truenas-monitor` command group is present, but several commands still return scaffold/demo output in baseline.

For real orphan signal in current baseline, prefer:

- Go API: `GET /api/v1/orphans`
- Go API: `GET /api/v1/orphans/pvs`

## Common local failure signatures

- `/ready` fails: check Kubernetes context and TrueNAS reachability/credentials.
- TLS failures to TrueNAS: verify CA trust; keep `insecure` disabled outside lab-only tests.
- Empty or unexpected payloads: verify namespace and age-threshold assumptions in config.

## Next step

When local checks are stable, follow [operator-first.md](operator-first.md) for controlled in-cluster onboarding and rollout guardrails.
