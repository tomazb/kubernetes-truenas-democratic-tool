# Prometheus Metrics Reference

The TrueNAS Storage Monitor exports comprehensive metrics for monitoring storage health, efficiency, and performance.

## Overview

Metrics are exposed on port 9090 by default at `/metrics` endpoint when monitoring is started:

```bash
truenas-monitor monitor --metrics-port 9090
```

## Metric Categories

### Snapshot Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `truenas_snapshots_total` | Gauge | system, pool, dataset | Total number of snapshots |
| `truenas_snapshots_size_bytes` | Gauge | system, pool, dataset | Total size of snapshots in bytes |
| `truenas_snapshots_age_days` | Gauge | system, pool, dataset, snapshot_name | Age of snapshots in days |
| `truenas_orphaned_snapshots_total` | Gauge | system, type | Number of orphaned snapshots |

**Types for orphaned snapshots:**
- `k8s_without_truenas` - K8s VolumeSnapshots without TrueNAS snapshots
- `truenas_without_k8s` - TrueNAS snapshots without K8s VolumeSnapshots

### Storage Pool Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `truenas_storage_pool_size_bytes` | Gauge | pool_name, metric_type | Storage pool size metrics |
| `truenas_storage_pool_utilization_percent` | Gauge | pool_name | Pool utilization percentage |
| `truenas_storage_pool_health` | Gauge | pool_name, status | Pool health (1=healthy, 0=unhealthy) |

**Metric types for pool size:**
- `total` - Total pool capacity
- `used` - Used space
- `free` - Available space

### Volume Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `truenas_persistent_volumes_total` | Gauge | namespace, storage_class, status | Total PVs |
| `truenas_persistent_volume_claims_total` | Gauge | namespace, storage_class, status | Total PVCs |
| `truenas_orphaned_pvs_total` | Gauge | namespace | Orphaned PVs count |
| `truenas_orphaned_pvcs_total` | Gauge | namespace | Orphaned PVCs count |

### Efficiency Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `truenas_thin_provisioning_ratio` | Gauge | pool_name | Ratio of allocated to used storage |
| `truenas_compression_ratio` | Gauge | pool_name | Data compression ratio |
| `truenas_snapshot_overhead_percent` | Gauge | pool_name | Snapshot storage overhead % |

### System Health Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `truenas_system_connectivity` | Gauge | system | Connection status (1=connected, 0=disconnected) |
| `truenas_csi_driver_pods_total` | Gauge | driver_name, status | CSI driver pod counts |
| `truenas_active_alerts_total` | Gauge | level, category | Active alert counts |

**System labels:**
- `kubernetes` - K8s cluster connectivity
- `truenas` - TrueNAS API connectivity

**Alert levels:**
- `warning` - Non-critical issues
- `error` - Critical issues requiring attention
- `critical` - Severe issues affecting service

**Alert categories:**
- `cleanup` - Resource cleanup needed
- `storage` - Storage capacity/efficiency issues
- `health` - System health problems

### Operation Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `truenas_monitoring_runs_total` | Counter | status | Total monitoring runs |
| `truenas_monitoring_duration_seconds` | Histogram | operation | Duration of monitoring operations |

**Operation types:**
- `full_check` - Complete monitoring cycle
- `snapshot_check` - Snapshot-specific checks
- `orphan_check` - Orphaned resource detection

### System Info

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `truenas_monitor_info` | Info | version, component | Monitor version and component info |

## Example Prometheus Queries

### Storage Health

```promql
# Pool utilization above 80%
truenas_storage_pool_utilization_percent > 80

# Unhealthy storage pools
truenas_storage_pool_health == 0

# Total storage used across all pools
sum(truenas_storage_pool_size_bytes{metric_type="used"})
```

### Snapshot Management

```promql
# Total snapshot size by pool
sum by (pool_name) (truenas_snapshots_size_bytes)

# Snapshots older than 30 days
truenas_snapshots_age_days > 30

# Orphaned snapshot trend
rate(truenas_orphaned_snapshots_total[5m])
```

### Resource Efficiency

```promql
# Thin provisioning efficiency
truenas_thin_provisioning_ratio > 2.0

# Snapshot overhead above 20%
truenas_snapshot_overhead_percent > 20

# Wasted storage from orphaned resources
sum(truenas_orphaned_pvs_total) * avg(truenas_persistent_volumes_total)
```

### System Monitoring

```promql
# System connectivity issues
truenas_system_connectivity == 0

# CSI driver unhealthy pods
truenas_csi_driver_pods_total{status="unhealthy"} > 0

# Alert rate by category
rate(truenas_active_alerts_total[5m]) by (category)
```

## Grafana Dashboard

Example dashboard JSON for importing into Grafana:

```json
{
  "dashboard": {
    "title": "TrueNAS Storage Monitor",
    "panels": [
      {
        "title": "Storage Pool Utilization",
        "targets": [{
          "expr": "truenas_storage_pool_utilization_percent"
        }]
      },
      {
        "title": "Snapshot Growth",
        "targets": [{
          "expr": "rate(truenas_snapshots_size_bytes[1h])"
        }]
      },
      {
        "title": "Orphaned Resources",
        "targets": [{
          "expr": "sum(truenas_orphaned_pvs_total) + sum(truenas_orphaned_snapshots_total)"
        }]
      }
    ]
  }
}
```

## Alerting Rules

Example Prometheus alerting rules:

```yaml
groups:
  - name: truenas_storage
    rules:
      - alert: StoragePoolHighUtilization
        expr: truenas_storage_pool_utilization_percent > 85
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Storage pool {{ $labels.pool_name }} is {{ $value }}% full"
          
      - alert: OrphanedResourcesDetected
        expr: sum(truenas_orphaned_pvs_total + truenas_orphaned_snapshots_total) > 10
        for: 30m
        labels:
          severity: warning
        annotations:
          summary: "{{ $value }} orphaned resources detected"
          
      - alert: SnapshotOverheadHigh
        expr: truenas_snapshot_overhead_percent > 25
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "Snapshot overhead is {{ $value }}% on {{ $labels.pool_name }}"
          
      - alert: SystemConnectivityLost
        expr: truenas_system_connectivity == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Lost connectivity to {{ $labels.system }}"
```

## Integration with Monitoring Stack

### Prometheus Configuration

Add to `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'truenas-monitor'
    static_configs:
      - targets: ['truenas-monitor:9090']
    scrape_interval: 30s
```

### Kubernetes Service Monitor

For Prometheus Operator:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: truenas-monitor
  namespace: storage-monitoring
spec:
  selector:
    matchLabels:
      app: truenas-monitor
  endpoints:
    - port: metrics
      interval: 30s
      path: /metrics
```

## Performance Considerations

- Metrics are updated during each monitoring cycle
- Default interval is 5 minutes (configurable)
- Each metric update is wrapped in error handling
- Separate Prometheus registry per Monitor instance prevents conflicts
- Histogram buckets are optimized for storage operations (seconds)

## Troubleshooting

### No metrics exposed

Check if metrics are enabled in config:
```yaml
metrics:
  enabled: true
  port: 9090
```

### Port already in use

Change the metrics port:
```bash
truenas-monitor monitor --metrics-port 9091
```

### Missing metrics

Ensure the monitoring service is running and has connectivity to both Kubernetes and TrueNAS.

### High cardinality

Monitor label combinations to avoid metrics explosion. The tool limits labels to essential dimensions.