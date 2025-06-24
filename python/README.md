# TrueNAS Storage Monitor - Python CLI

A comprehensive monitoring tool for OpenShift/Kubernetes storage with TrueNAS and democratic-csi.

[![Tests](https://github.com/tomazb/kubernetes-truenas-democratic-tool/actions/workflows/ci.yml/badge.svg)](https://github.com/tomazb/kubernetes-truenas-democratic-tool/actions/workflows/ci.yml)
[![Coverage](https://codecov.io/gh/tomazb/kubernetes-truenas-democratic-tool/branch/main/graph/badge.svg)](https://codecov.io/gh/tomazb/kubernetes-truenas-democratic-tool)
[![Python Version](https://img.shields.io/badge/python-3.10%2B-blue)](https://www.python.org/downloads/)

## Features

- **Comprehensive Monitoring** - Monitor PVs, PVCs, snapshots, and storage pools
- **Orphaned Resource Detection** - Find resources without corresponding pairs
- **Advanced Snapshot Management** - Cross-system health monitoring and analysis
- **Storage Analytics** - Thin provisioning efficiency and capacity planning
- **Prometheus Metrics** - Export 15+ metrics for monitoring and alerting
- **Multiple Output Formats** - Table, JSON, and YAML outputs
- **Real-time Monitoring** - Continuous monitoring with configurable intervals

## Installation

### From Source

```bash
git clone https://github.com/tomazb/kubernetes-truenas-democratic-tool.git
cd kubernetes-truenas-democratic-tool/python
pip install -r requirements.txt
pip install -e .
```

### From PyPI (when available)

```bash
pip install truenas-storage-monitor
```

### Using Docker/Podman

```bash
podman run -it --rm \
  -v ~/.kube:/home/app/.kube:ro \
  -v ./config.yaml:/app/config.yaml:ro \
  ghcr.io/tomazb/truenas-monitor:latest --help
```

## Quick Start

### 1. Create Configuration

Create a `config.yaml` file:

```yaml
openshift:
  namespace: democratic-csi
  storage_class: democratic-csi-nfs

truenas:
  url: https://truenas.example.com
  username: admin
  password: ${TRUENAS_PASSWORD}  # Use environment variable

monitoring:
  interval: 300  # 5 minutes
  thresholds:
    orphaned_pv_age_hours: 24
    snapshot_age_days: 30
    pool_usage_percent: 80

metrics:
  enabled: true
  port: 9090
```

### 2. Validate Configuration

```bash
truenas-monitor validate
```

### 3. Check for Issues

```bash
# Find orphaned resources
truenas-monitor orphans

# Check snapshot health
truenas-monitor snapshots --health

# Analyze storage efficiency
truenas-monitor analyze
```

## Command Reference

### Global Options

```bash
truenas-monitor [OPTIONS] COMMAND [ARGS]...

Options:
  --version                Show version and exit
  -c, --config PATH       Configuration file path
  -l, --log-level LEVEL   Set logging level (debug/info/warning/error)
  --help                  Show help and exit
```

### Commands

#### `validate` - Validate Configuration

```bash
truenas-monitor validate [OPTIONS]

Options:
  -v, --verbose           Show detailed validation results
  --format FORMAT         Output format (table/json/yaml)
```

Validates:
- Configuration file syntax
- Kubernetes connectivity
- TrueNAS API access
- CSI driver health
- RBAC permissions

#### `orphans` - Find Orphaned Resources

```bash
truenas-monitor orphans [OPTIONS]

Options:
  --format FORMAT         Output format (table/json/yaml)
  --fix                   Fix orphaned resources (interactive)
```

Detects:
- PVs without TrueNAS volumes
- TrueNAS volumes without PVs
- Unbound PVCs
- Orphaned snapshots

#### `analyze` - Storage Analysis

```bash
truenas-monitor analyze [OPTIONS]

Options:
  --format FORMAT         Output format (table/json/yaml)
  --pools                 Show pool-level analysis
  --volumes               Show volume-level analysis
```

Analyzes:
- Thin provisioning efficiency
- Storage utilization trends
- Compression ratios
- Snapshot overhead
- Capacity planning recommendations

#### `snapshots` - Snapshot Management

```bash
truenas-monitor snapshots [OPTIONS]

Options:
  --health                Show snapshot health status
  --orphaned              Find orphaned snapshots
  --analysis              Show detailed usage analysis
  --age-days N            Filter snapshots older than N days
  --volume VOLUME         Filter by volume name
  --format FORMAT         Output format (table/json/yaml)
```

Features:
- Cross-system snapshot tracking
- Size and age analysis
- Growth trend monitoring
- Cleanup recommendations

#### `monitor` - Start Monitoring Service

```bash
truenas-monitor monitor [OPTIONS]

Options:
  --once                  Run single check and exit
  --interval SECONDS      Check interval (default: from config)
  --metrics-port PORT     Prometheus metrics port (default: 9090)
  --namespace NS          Override namespace from config
```

Provides:
- Continuous monitoring
- Prometheus metrics export
- Alert generation
- Health status tracking

#### `report` - Generate Reports

```bash
truenas-monitor report [OPTIONS]

Options:
  -o, --output PATH       Output file path
  -f, --format FORMAT     Report format (html/pdf/json)
  --email EMAIL           Email report to address
```

## Output Formats

### Table Format (Default)

```
┏━━━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━━━┓
┃ Resource             ┃ Status ┃ Size       ┃ Age         ┃
┡━━━━━━━━━━━━━━━━━━━━━━╇━━━━━━━━╇━━━━━━━━━━━━╇━━━━━━━━━━━━━┩
│ pvc-12345           │ ⚠️      │ 10.5 GiB   │ 45 days     │
│ pvc-67890           │ ❌      │ 5.2 GiB    │ 92 days     │
└──────────────────────┴────────┴────────────┴─────────────┘
```

### JSON Format

```bash
truenas-monitor orphans --format json | jq .
```

```json
{
  "orphaned_pvs": [
    {
      "name": "pvc-12345",
      "size_gb": 10.5,
      "age_days": 45,
      "status": "Released"
    }
  ],
  "total": 1
}
```

### YAML Format

```bash
truenas-monitor analyze --format yaml
```

```yaml
storage_efficiency:
  thin_provisioning_ratio: 2.5
  compression_ratio: 1.8
  snapshot_overhead_percent: 15.2
pools:
  - name: tank
    utilization_percent: 72.5
    health: healthy
```

## Prometheus Metrics

When monitoring is started, metrics are exposed at `http://localhost:9090/metrics`:

- `truenas_snapshots_total` - Total snapshot count
- `truenas_snapshots_size_bytes` - Snapshot storage usage
- `truenas_orphaned_snapshots_total` - Orphaned snapshots
- `truenas_storage_pool_utilization_percent` - Pool usage
- `truenas_thin_provisioning_ratio` - Overprovisioning ratio
- `truenas_system_connectivity` - System health status

See [Prometheus Metrics Documentation](../docs/PROMETHEUS_METRICS.md) for complete reference.

## Environment Variables

- `KUBECONFIG` - Path to kubeconfig file
- `TRUENAS_URL` - TrueNAS API URL
- `TRUENAS_USER` - TrueNAS username
- `TRUENAS_PASSWORD` - TrueNAS password
- `TRUENAS_MONITOR_CONFIG` - Config file path
- `TRUENAS_MONITOR_LOG_LEVEL` - Logging level

## Development

### Running Tests

```bash
# Unit tests with coverage
python run_unit_tests.py

# Integration tests
python run_integration_tests.py

# Quick smoke tests
python run_integration_tests.py --smoke

# Specific test category
python run_integration_tests.py --category cli
```

### Test Categories

- **Unit Tests** - Fast, isolated component tests (90% coverage target)
- **Integration Tests** - Real system interaction tests
- **CLI Tests** - Command-line interface testing
- **Kubernetes Tests** - K8s API interaction tests
- **TrueNAS Tests** - TrueNAS API contract tests

### Code Quality

```bash
# Format code
black truenas_storage_monitor/

# Lint code
flake8 truenas_storage_monitor/

# Type checking
mypy truenas_storage_monitor/

# Security scan
bandit -r truenas_storage_monitor/
```

## Troubleshooting

### Connection Issues

```bash
# Test with verbose logging
truenas-monitor validate -v --log-level debug

# Check specific connectivity
truenas-monitor validate | grep -E "(Kubernetes|TrueNAS)"
```

### Performance Issues

```bash
# Run single check with timing
time truenas-monitor orphans --format json

# Profile memory usage
mprof run truenas-monitor monitor --once
mprof plot
```

### Metric Collection Issues

```bash
# Test metrics endpoint
curl http://localhost:9090/metrics

# Check metrics in verbose mode
truenas-monitor monitor --log-level debug --once
```

## Support

- **Issues**: [GitHub Issues](https://github.com/tomazb/kubernetes-truenas-democratic-tool/issues)
- **Documentation**: [Full Documentation](https://github.com/tomazb/kubernetes-truenas-democratic-tool)
- **Examples**: See [DEMO.md](DEMO.md) for usage examples

## License

Apache License 2.0 - See [LICENSE](../LICENSE) for details.