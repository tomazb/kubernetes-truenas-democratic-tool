# Configuration compatibility — Go vs Python

Go services and the Python library/CLI use **different YAML schemas**. A single config file is not portable without translation.

## Which file to use

| Runtime | Example | Required top-level keys |
|---------|---------|-------------------------|
| Go monitor (`bin/monitor`) | [config.go.example](../config.go.example) | `kubernetes`, `truenas` |
| Go API server (`bin/api-server`) | [config.go.example](../config.go.example) | `kubernetes`, `truenas` |
| Python CLI / library | [config.yaml.example](../config.yaml.example) | `openshift`, `monitoring` |
| In-cluster Go deploy | [deploy/kubernetes/configmap.yaml](../deploy/kubernetes/configmap.yaml) | Embedded Go schema |

## Key mapping

| Concern | Go (`go/pkg/config`) | Python (`truenas_storage_monitor.config`) |
|---------|----------------------|-------------------------------------------|
| Cluster section | `kubernetes:` | `openshift:` (accepts deprecated `kubernetes:` alias via normalization) |
| Kubeconfig | `kubernetes.kubeconfig` | `openshift.kubeconfig` |
| In-cluster mode | `kubernetes.in_cluster` | `openshift.in_cluster` (boolean, defaults to false) |
| CSI namespace | `kubernetes.namespace` | `openshift.namespace` |
| Monitor tuning | `monitor.scan_interval`, `monitor.orphan_threshold`, `monitor.snapshot_retention` | `monitoring.orphan_threshold`, `monitoring.orphan_check_interval`, nested `monitoring.snapshot` / `monitoring.storage` |
| TrueNAS URL | `truenas.url` | `truenas.url` |
| TrueNAS auth | `truenas.username`, `truenas.password` | `truenas.username`/`password` or `truenas.api_key` |
| TLS insecure | `truenas.insecure` (default false) | `truenas.insecure` (default false) |
| Custom CA | `truenas.ca_file` | `truenas.ca_file` |
| TrueNAS timeout | `truenas.timeout` as duration string (`30s`) | `truenas.timeout` as integer seconds or string with `s` suffix (e.g. `30`, `30s`) |
| Slack alerts | `alerts.slack.webhook` | `alerts.slack.webhook_url` |
| Metrics | `metrics.enabled`, `metrics.port`, `metrics.path` | `metrics` section in example only; not fully wired in Python baseline |
| Logging | `logging.level`, `logging.encoding` | `logging.level`, `logging.format` in example only |
| API server listen/TLS | Not in Go config file (CLI flags) | `api:` block in Python example is **planned**, not read today |
| API auth / security block | `security:` keys parsed in Go config but **not enforced** by shipped API server | Not applicable |

## Minimal examples

**Go** (orphan API / monitor):

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

**Python** (CLI scaffold):

```yaml
openshift:
  kubeconfig: ~/.kube/config
  namespace: democratic-csi

monitoring:
  orphan_threshold: 24h

truenas:
  url: https://truenas.example.com
  username: admin
  password: ${TRUENAS_PASSWORD}
```

## Environment variable expansion

- **Go** expands `${VAR}` and `${VAR:default}` (default used when `VAR` is unset or empty).
- **Python** uses `os.path.expandvars`, which supports `$VAR` and `${VAR}` only. The `${VAR:default}` form is **not** expanded; the literal string remains in the config value.

## Validation differences

- **Go** requires `truenas.url`, `truenas.username`, and `truenas.password` when a config file is present.
- **Python** requires `openshift` and `monitoring` sections; TrueNAS must have URL plus username/password or `api_key`.

## Planned sections (Python example only)

Blocks in `config.yaml.example` under `reporting`, `api`, `performance`, and `features` are **roadmap placeholders**. The baseline Python library does not load them. Do not assume they affect runtime behavior.
