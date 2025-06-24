# TrueNAS Storage Monitor - Demo & Usage Guide

This comprehensive guide demonstrates all features of the TrueNAS Storage Monitor, including snapshot management, orphaned resource detection, storage analytics, and Prometheus metrics integration.

## Prerequisites

1. Install the tool:
```bash
pip install -e .
```

2. Create a configuration file (`config.yaml`):
```yaml
openshift:
  namespace: democratic-csi
  storage_class: democratic-csi-nfs
  csi_driver: org.democratic-csi.nfs

truenas:
  url: https://your-truenas.example.com
  username: admin
  password: your-password
  verify_ssl: false

monitoring:
  interval: 300  # 5 minutes
  thresholds:
    orphaned_pv_age_hours: 24
    pending_pvc_minutes: 60
    snapshot_age_days: 30
    pool_usage_percent: 80
    snapshot_size_gb: 100

metrics:
  enabled: true
  port: 9090
  path: /metrics
```

## Snapshot Management Commands

### 1. Check Snapshot Health

Get a comprehensive overview of snapshot health across both systems:

```bash
# Table format (default)
truenas-monitor snapshots --health

# JSON format for automation
truenas-monitor snapshots --health --format json

# YAML format
truenas-monitor snapshots --health --format yaml
```

**Output includes:**
- K8s snapshot counts (total, ready, pending, stale)
- TrueNAS snapshot metrics (size, age distribution)
- Orphaned resource detection
- Alerts for issues requiring attention
- Recommendations for optimization

### 2. Analyze Snapshot Usage

Get detailed analysis of snapshot storage consumption:

```bash
# Analyze all snapshots
truenas-monitor snapshots --analysis

# Analyze specific volume
truenas-monitor snapshots --volume pvc-12345 --analysis

# Get analysis in JSON
truenas-monitor snapshots --analysis --format json
```

**Analysis includes:**
- Total snapshot count and size
- Age distribution (last 24h, week, month, older)
- Large snapshots (>1GB) with details
- Average snapshot age
- Storage recommendations

### 3. Find Orphaned Snapshots

Identify snapshots that exist in one system but not the other:

```bash
# Find orphaned snapshots
truenas-monitor snapshots --orphaned

# Output in YAML for review
truenas-monitor snapshots --orphaned --format yaml
```

**Detects:**
- K8s VolumeSnapshots without TrueNAS backing
- TrueNAS snapshots without K8s objects
- Stuck snapshots in pending state

### 4. List Snapshots with Filters

View snapshots with various filtering options:

```bash
# List all snapshots
truenas-monitor snapshots

# List snapshots for specific volume
truenas-monitor snapshots --volume pvc-abc123

# Show only old snapshots (>30 days)
truenas-monitor snapshots --age-days 30

# Combine filters
truenas-monitor snapshots --volume pvc-xyz --age-days 7 --format json
```

## Storage Efficiency Analysis

Analyze overall storage efficiency including snapshot overhead:

```bash
# Run comprehensive storage analysis
truenas-monitor analyze

# Get thin provisioning ratios and recommendations
truenas-monitor analyze --format json
```

**Provides:**
- Thin provisioning ratio
- Snapshot overhead percentage
- Pool utilization and health
- Fragmentation warnings
- Capacity planning recommendations

## Monitoring Mode

Run continuous monitoring with Prometheus metrics:

```bash
# Run once
truenas-monitor monitor --once

# Continuous monitoring (default 60s interval)
truenas-monitor monitor

# Custom interval
truenas-monitor monitor --interval 300

# Expose Prometheus metrics on custom port
truenas-monitor monitor --metrics-port 8080

# Access metrics endpoint
curl http://localhost:9090/metrics | grep truenas_
```

## Example Outputs

### Health Check Output
```
Snapshot Health Status

Kubernetes Snapshots
┏━━━━━━━━━━━━━━━━━┳━━━━━━━┓
┃ Metric          ┃ Count ┃
┡━━━━━━━━━━━━━━━━━╇━━━━━━━┩
│ Total           │ 25    │
│ Ready           │ 23    │
│ Pending         │ 2     │
│ Stale           │ 5     │
└─────────────────┴───────┘

TrueNAS Snapshots
┏━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━┓
┃ Metric             ┃ Value    ┃
┡━━━━━━━━━━━━━━━━━━━━╇━━━━━━━━━━┩
│ Total              │ 30       │
│ Total Size GB      │ 125.50   │
│ Recent (<24h)      │ 3        │
│ Old (>30d)         │ 8        │
│ Large (>1GB)       │ 12       │
└────────────────────┴──────────┘

Alerts:
WARNING: Found 7 orphaned snapshots that may need cleanup
ERROR: 2 snapshots are stuck in pending state

Recommendations:
• Consider cleaning up 8 snapshots older than 30 days
• Snapshot overhead is 22.5% - consider snapshot cleanup
```

### Analysis Output
```
Snapshot Analysis

Summary Statistics
┏━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━┓
┃ Metric            ┃ Value    ┃
┡━━━━━━━━━━━━━━━━━━━╇━━━━━━━━━━┩
│ Total Snapshots   │ 45       │
│ Total Size        │ 234.75GB │
│ Average Age       │ 18.5 days│
└───────────────────┴──────────┘

Large Snapshots (>1GB)
┏━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━┳━━━━━━━━━━┓
┃ Name           ┃ Dataset               ┃ Size(GB) ┃ Age(days)┃
┡━━━━━━━━━━━━━━━━╇━━━━━━━━━━━━━━━━━━━━━━━╇━━━━━━━━━━╇━━━━━━━━━━┩
│ daily-2024-01  │ tank/k8s/volumes/pvc-1│ 15.2     │ 30       │
│ weekly-2024-w2 │ tank/k8s/volumes/pvc-2│ 22.5     │ 14       │
└────────────────┴───────────────────────┴──────────┴──────────┘
```

## Integration with Automation

### Using with CI/CD

```bash
# Check for orphaned resources in CI pipeline
if truenas-monitor snapshots --orphaned --format json | jq '.[] | select(.type=="orphaned")' | grep -q .; then
  echo "Found orphaned snapshots!"
  exit 1
fi
```

### Prometheus Alerts

```yaml
groups:
  - name: snapshot_alerts
    rules:
      - alert: HighSnapshotCount
        expr: truenas_snapshots_total > 100
        for: 5m
        annotations:
          summary: "High number of snapshots detected"
          
      - alert: OrphanedSnapshots
        expr: truenas_orphaned_snapshots_total > 5
        for: 10m
        annotations:
          summary: "Multiple orphaned snapshots found"
```

## Troubleshooting

### Common Issues

1. **"No snapshots found"**
   - Verify TrueNAS credentials
   - Check dataset paths in configuration
   - Ensure democratic-csi is creating snapshots

2. **"Cannot connect to Kubernetes"**
   - Verify kubeconfig is available
   - Check cluster connectivity
   - Ensure proper RBAC permissions

3. **"High orphaned snapshot count"**
   - Review snapshot retention policies
   - Check for failed snapshot operations
   - Verify CSI driver health

### Debug Mode

Run commands with debug logging:

```bash
truenas-monitor --log-level debug snapshots --health
```

## Prometheus Metrics

The tool exports comprehensive metrics for monitoring:

### Available Metrics

```bash
# View all metrics
curl http://localhost:9090/metrics

# Key metrics to monitor:
truenas_snapshots_total                    # Total snapshots by system
truenas_snapshots_size_bytes               # Snapshot storage usage
truenas_orphaned_snapshots_total           # Orphaned snapshots count
truenas_storage_pool_utilization_percent   # Pool usage percentage
truenas_thin_provisioning_ratio            # Overprovisioning ratio
truenas_system_connectivity                # System health status
truenas_monitoring_duration_seconds        # Operation performance
```

### Grafana Dashboard Example

```json
{
  "panels": [
    {
      "title": "Storage Pool Usage",
      "expr": "truenas_storage_pool_utilization_percent"
    },
    {
      "title": "Snapshot Growth Rate",
      "expr": "rate(truenas_snapshots_size_bytes[1h])"
    },
    {
      "title": "Orphaned Resources",
      "expr": "truenas_orphaned_pvs_total + truenas_orphaned_snapshots_total"
    }
  ]
}
```

## Configuration Validation

Always validate your configuration before starting monitoring:

```bash
# Basic validation
truenas-monitor validate

# Verbose validation with all checks
truenas-monitor validate --verbose

# Output validation results as JSON
truenas-monitor validate --format json
```

**Validation checks:**
- Configuration file syntax
- Kubernetes cluster connectivity
- TrueNAS API accessibility
- CSI driver health
- RBAC permissions
- Namespace existence

## Performance Optimization

### CLI Performance

All commands are optimized for speed:
- `validate`: < 10 seconds
- `orphans`: < 5 seconds  
- `snapshots`: < 5 seconds
- `analyze`: < 10 seconds

### Monitoring Performance

```bash
# Test performance with timing
time truenas-monitor orphans --format json

# Run with performance metrics
truenas-monitor monitor --once --log-level debug
```

## Testing

### Unit Tests
```bash
# Run unit tests with coverage
cd python/
python run_unit_tests.py

# Quick unit tests
python run_unit_tests.py --quick

# Test specific module
python run_unit_tests.py --module config
```

### Integration Tests
```bash
# Run all integration tests
python run_integration_tests.py

# Run smoke tests only
python run_integration_tests.py --smoke

# Test specific category
python run_integration_tests.py --category cli
```

## Best Practices

1. **Regular Health Checks**: Run `snapshots --health` daily
2. **Orphan Cleanup**: Check for orphans weekly
3. **Retention Policy**: Keep snapshots under 30 days
4. **Size Monitoring**: Alert when total size exceeds threshold
5. **Automation**: Integrate checks into CI/CD pipelines
6. **Metrics Monitoring**: Set up Prometheus alerts for key metrics
7. **Configuration Validation**: Always validate config changes
8. **Performance Testing**: Monitor command execution times