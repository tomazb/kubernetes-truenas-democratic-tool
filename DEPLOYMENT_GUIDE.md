# TrueNAS Storage Monitor - Test & Deployment Guide

This guide provides step-by-step instructions for testing and deploying the TrueNAS Storage Monitor in various environments.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Test Setup](#quick-test-setup)
3. [Environment Preparation](#environment-preparation)
4. [Installation Methods](#installation-methods)
5. [Configuration](#configuration)
6. [Testing & Validation](#testing--validation)
7. [Production Deployment](#production-deployment)
8. [Troubleshooting](#troubleshooting)
9. [Monitoring & Maintenance](#monitoring--maintenance)

## Prerequisites

### Required Infrastructure

- **Kubernetes/OpenShift Cluster** (v1.24+)
  - democratic-csi installed and configured
  - RBAC permissions for monitoring
  - Access to cluster API
  
- **TrueNAS Scale** (22.12+)
  - REST API enabled
  - Administrative access or dedicated API user
  - Network connectivity from monitoring location

- **Development Environment**
  - Python 3.10+ or Docker/Podman
  - Git for source code access
  - Network access to both K8s and TrueNAS

### Network Requirements

```bash
# Test connectivity to your systems
curl -k https://your-truenas.example.com/api/v2.0/auth/me
kubectl cluster-info
```

## Quick Test Setup

### 1. Clone the Repository

```bash
git clone https://github.com/tomazb/kubernetes-truenas-democratic-tool.git
cd kubernetes-truenas-democratic-tool/python
```

### 2. Install Dependencies

```bash
# Using virtual environment (recommended)
python -m venv venv
source venv/bin/activate  # Linux/Mac
# or: venv\Scripts\activate  # Windows

# Install requirements
pip install -r requirements.txt
pip install -e .

# Verify installation
truenas-monitor --help
```

### 3. Create Test Configuration

```bash
# Create config.yaml
cat > config.yaml << 'EOF'
openshift:
  # Use current kubeconfig context
  namespace: democratic-csi
  storage_class: democratic-csi-nfs
  csi_driver: org.democratic-csi.nfs

truenas:
  url: https://your-truenas.example.com
  username: admin
  password: your-password
  verify_ssl: false  # Set to true in production

monitoring:
  interval: 60
  thresholds:
    orphaned_pv_age_hours: 24
    pending_pvc_minutes: 60
    snapshot_age_days: 30
    pool_usage_percent: 80
    snapshot_size_gb: 100

alerts:
  enabled: false  # Enable for production
EOF
```

### 4. Initial Validation

```bash
# Test configuration and connectivity
truenas-monitor --config config.yaml validate

# If validation passes, run a quick check
truenas-monitor --config config.yaml orphans
```

## Environment Preparation

### Kubernetes RBAC Setup

Create the necessary permissions for the monitoring tool:

```yaml
# rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: truenas-monitor
  namespace: monitoring
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: truenas-monitor
rules:
# Core resources
- apiGroups: [""]
  resources: ["persistentvolumes", "persistentvolumeclaims", "pods", "namespaces"]
  verbs: ["get", "list", "watch"]
# Storage resources
- apiGroups: ["storage.k8s.io"]
  resources: ["storageclasses", "volumeattachments", "csinodes"]
  verbs: ["get", "list", "watch"]
# Snapshot resources
- apiGroups: ["snapshot.storage.k8s.io"]
  resources: ["volumesnapshots", "volumesnapshotcontents", "volumesnapshotclasses"]
  verbs: ["get", "list", "watch"]
# Events for monitoring
- apiGroups: [""]
  resources: ["events"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: truenas-monitor
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: truenas-monitor
subjects:
- kind: ServiceAccount
  name: truenas-monitor
  namespace: monitoring
```

Apply the RBAC configuration:

```bash
kubectl apply -f rbac.yaml
```

### TrueNAS API User Setup

1. **Create dedicated API user (recommended for production):**

```bash
# SSH to TrueNAS or use web UI
midclt call user.create '{
  "username": "truenas-monitor",
  "full_name": "TrueNAS Monitor",
  "group": 1000,
  "password": "your-secure-password",
  "ssh_password_enabled": false,
  "shell": "/usr/bin/nologin"
}'

# Grant necessary permissions
midclt call user.update "truenas-monitor" '{
  "groups": [
    {"id": "storage_admin"},
    {"id": "readonly"}
  ]
}'
```

2. **Generate API Key (alternative to password):**

```bash
# In TrueNAS web UI: Account > API Keys > Add
# Save the generated key securely
```

## Installation Methods

### Method 1: Python Package Installation

```bash
# Install from source
cd python/
pip install -e .

# Or install from PyPI (when available)
pip install truenas-storage-monitor
```

### Method 2: Container Deployment

```bash
# Build container
podman build -t truenas-monitor -f container/Containerfile.cli .

# Run with config mounted
podman run -it --rm \
  -v ~/.kube:/root/.kube:ro \
  -v ./config.yaml:/app/config.yaml:ro \
  truenas-monitor:latest validate
```

### Method 3: Kubernetes Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: truenas-monitor
  namespace: monitoring
  labels:
    app: truenas-monitor
spec:
  replicas: 1
  selector:
    matchLabels:
      app: truenas-monitor
  template:
    metadata:
      labels:
        app: truenas-monitor
    spec:
      serviceAccountName: truenas-monitor
      containers:
      - name: monitor
        image: ghcr.io/yourusername/truenas-monitor:v0.2.0-beta
        args: ["monitor", "--config", "/etc/config/config.yaml", "--metrics-port", "8080"]
        ports:
        - containerPort: 8080
          name: metrics
        volumeMounts:
        - name: config
          mountPath: /etc/config
        env:
        - name: TRUENAS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: truenas-credentials
              key: password
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"
      volumes:
      - name: config
        configMap:
          name: truenas-monitor-config
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: truenas-monitor-config
  namespace: monitoring
data:
  config.yaml: |
    openshift:
      namespace: democratic-csi
      storage_class: democratic-csi-nfs
    truenas:
      url: https://truenas.example.com
      username: truenas-monitor
      password: ${TRUENAS_PASSWORD}
      verify_ssl: true
    monitoring:
      interval: 300
      thresholds:
        orphaned_pv_age_hours: 24
        snapshot_age_days: 30
---
apiVersion: v1
kind: Secret
metadata:
  name: truenas-credentials
  namespace: monitoring
type: Opaque
stringData:
  password: "your-truenas-password"
---
apiVersion: v1
kind: Service
metadata:
  name: truenas-monitor-metrics
  namespace: monitoring
spec:
  selector:
    app: truenas-monitor
  ports:
  - port: 8080
    targetPort: 8080
    name: metrics
```

Deploy to Kubernetes:

```bash
kubectl apply -f deployment.yaml
```

## Configuration

### Complete Configuration Reference

```yaml
# config.yaml - Full configuration example
openshift:
  # Kubernetes connection settings
  kubeconfig: ~/.kube/config  # Optional, uses default if not specified
  namespace: democratic-csi   # Main CSI namespace
  storage_class: democratic-csi-nfs  # Storage class to monitor
  csi_driver: org.democratic-csi.nfs  # CSI driver name
  
truenas:
  # TrueNAS connection settings
  url: https://truenas.example.com
  username: admin  # Or API user
  password: ${TRUENAS_PASSWORD}  # Use env var for security
  api_key: ${TRUENAS_API_KEY}    # Alternative to username/password
  verify_ssl: true  # Always true in production
  timeout: 30       # Request timeout in seconds
  max_retries: 3    # Number of retry attempts

monitoring:
  # Monitoring behavior
  interval: 300     # Check interval in seconds
  thresholds:
    orphaned_pv_age_hours: 24      # PV orphan threshold
    pending_pvc_minutes: 60        # PVC pending threshold
    snapshot_age_days: 30          # Snapshot age threshold
    pool_usage_percent: 80         # Pool usage alert threshold
    snapshot_size_gb: 100          # Large snapshot threshold
    thin_provisioning_ratio: 3.0   # Overcommit ratio threshold
  
  # Metrics and export
  metrics:
    enabled: true
    port: 8080
    path: /metrics
  
alerts:
  # Alert configuration
  enabled: true
  channels:
    slack:
      webhook_url: ${SLACK_WEBHOOK}
      channel: "#storage-alerts"
    email:
      smtp_server: smtp.example.com
      smtp_port: 587
      username: ${SMTP_USER}
      password: ${SMTP_PASS}
      from: alerts@example.com
      to: ["admin@example.com"]

logging:
  level: info  # debug, info, warning, error
  format: json # json or text
  file: /var/log/truenas-monitor.log  # Optional log file
```

### Environment Variables

```bash
# Security-sensitive values
export TRUENAS_PASSWORD="your-secure-password"
export TRUENAS_API_KEY="your-api-key"
export SLACK_WEBHOOK="https://hooks.slack.com/..."
export SMTP_USER="alerts@example.com"
export SMTP_PASS="smtp-password"

# Optional overrides
export TRUENAS_MONITOR_CONFIG="/path/to/config.yaml"
export TRUENAS_MONITOR_LOG_LEVEL="debug"
```

## Testing & Validation

### 1. Basic Functionality Tests

```bash
# Test configuration loading
truenas-monitor --config config.yaml validate

# Check connectivity to both systems
truenas-monitor --config config.yaml validate --verbose

# Test basic monitoring functions
truenas-monitor --config config.yaml orphans --format json
truenas-monitor --config config.yaml analyze
```

### 2. Snapshot Management Tests

```bash
# Test snapshot health monitoring
truenas-monitor --config config.yaml snapshots --health

# Test orphaned snapshot detection
truenas-monitor --config config.yaml snapshots --orphaned

# Test snapshot analysis
truenas-monitor --config config.yaml snapshots --analysis

# Test with specific volume
truenas-monitor --config config.yaml snapshots --volume pvc-12345 --analysis
```

### 3. Run Demo Test Script

```bash
# Download and run the comprehensive test script
python test_snapshot_functionality.py

# Expected output should show:
# ✅ Health check completed successfully!
# ✅ Snapshot analysis working
# ✅ Orphan detection functional
# ✅ Storage efficiency analysis working
```

### 4. Integration Testing

```bash
# Test with real infrastructure (requires active systems)
truenas-monitor --config config.yaml monitor --once --verbose

# Check Prometheus metrics (if enabled)
curl http://localhost:8080/metrics | grep truenas

# Test alerting (if configured)
truenas-monitor --config config.yaml snapshots --health --format json | jq '.alerts'
```

### 5. Performance Testing

```bash
# Time the operations
time truenas-monitor --config config.yaml snapshots --health
time truenas-monitor --config config.yaml analyze

# Monitor resource usage
top -p $(pgrep -f truenas-monitor)

# Test with large datasets
truenas-monitor --config config.yaml snapshots --format json | jq 'length'
```

## Production Deployment

### 1. Security Hardening

```bash
# Use strong credentials
openssl rand -base64 32  # Generate secure password

# Store secrets securely
kubectl create secret generic truenas-credentials \
  --from-literal=password="$(openssl rand -base64 32)" \
  --from-literal=api-key="your-api-key"

# Enable SSL verification
sed -i 's/verify_ssl: false/verify_ssl: true/' config.yaml
```

### 2. High Availability Setup

```yaml
# ha-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: truenas-monitor
spec:
  replicas: 2  # Multiple replicas for HA
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  template:
    spec:
      containers:
      - name: monitor
        image: truenas-monitor:v0.2.0-beta
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        resources:
          requests:
            memory: "256Mi"
            cpu: "200m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

### 3. Monitoring Integration

```yaml
# servicemonitor.yaml (for Prometheus Operator)
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: truenas-monitor
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: truenas-monitor
  endpoints:
  - port: metrics
    interval: 60s
    path: /metrics
```

### 4. Alerting Rules

```yaml
# prometheus-rules.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: truenas-storage-alerts
spec:
  groups:
  - name: storage.rules
    rules:
    - alert: HighOrphanedResources
      expr: truenas_orphaned_resources_total > 10
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High number of orphaned storage resources"
        description: "{{ $value }} orphaned resources detected"
    
    - alert: SnapshotStorageHigh
      expr: truenas_snapshot_storage_bytes / truenas_total_storage_bytes > 0.3
      for: 10m
      labels:
        severity: warning
      annotations:
        summary: "Snapshot storage usage high"
        description: "Snapshots using {{ $value | humanizePercentage }} of total storage"
    
    - alert: TrueNASConnectionFailed
      expr: up{job="truenas-monitor"} == 0
      for: 2m
      labels:
        severity: critical
      annotations:
        summary: "TrueNAS monitoring connection failed"
```

### 5. Backup and Recovery

```bash
# Backup configuration
kubectl get configmap truenas-monitor-config -o yaml > backup-config.yaml
kubectl get secret truenas-credentials -o yaml > backup-secrets.yaml

# Create recovery procedure
cat > recovery.md << 'EOF'
# Recovery Procedure

1. Restore configuration:
   kubectl apply -f backup-config.yaml
   kubectl apply -f backup-secrets.yaml

2. Restart monitoring:
   kubectl rollout restart deployment/truenas-monitor

3. Verify functionality:
   kubectl logs -f deployment/truenas-monitor
EOF
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Connection Issues

**Problem:** Cannot connect to TrueNAS
```bash
# Debug steps
curl -k https://truenas.example.com/api/v2.0/auth/me
truenas-monitor --log-level debug validate
```

**Solutions:**
- Check network connectivity
- Verify SSL certificates (`verify_ssl: false` for testing)
- Confirm API credentials
- Check TrueNAS API service status

#### 2. Kubernetes Permission Issues

**Problem:** RBAC errors
```bash
# Check permissions
kubectl auth can-i list persistentvolumes --as=system:serviceaccount:monitoring:truenas-monitor
kubectl auth can-i list volumesnapshots --as=system:serviceaccount:monitoring:truenas-monitor
```

**Solutions:**
- Apply correct RBAC configuration
- Verify ServiceAccount exists
- Check ClusterRoleBinding

#### 3. High Memory Usage

**Problem:** Memory consumption too high
```bash
# Monitor memory usage
kubectl top pod -l app=truenas-monitor
```

**Solutions:**
- Increase resource limits
- Reduce monitoring frequency
- Filter by namespace/storage class

#### 4. Missing Snapshots

**Problem:** Snapshots not detected
```bash
# Debug snapshot detection
truenas-monitor snapshots --volume pvc-name --format json
kubectl get volumesnapshots -A
```

**Solutions:**
- Check VolumeSnapshotClass configuration
- Verify CSI driver supports snapshots
- Check dataset naming conventions

### Debug Commands

```bash
# Enable debug logging
truenas-monitor --log-level debug snapshots --health

# Test specific components
truenas-monitor validate --component kubernetes
truenas-monitor validate --component truenas

# Export detailed information
truenas-monitor analyze --format yaml > analysis.yaml
truenas-monitor orphans --format json > orphans.json

# Check configuration
truenas-monitor --config config.yaml validate --dry-run
```

### Log Analysis

```bash
# Container logs
kubectl logs -f deployment/truenas-monitor

# Find errors
kubectl logs deployment/truenas-monitor | grep ERROR

# Monitor metrics
curl -s http://localhost:8080/metrics | grep -E "(error|failed)"
```

## Monitoring & Maintenance

### 1. Regular Health Checks

```bash
# Daily health check script
#!/bin/bash
# daily-check.sh

echo "$(date): Running daily storage health check"

# Basic health
if truenas-monitor --config /etc/config.yaml validate; then
    echo "✅ Configuration valid"
else
    echo "❌ Configuration issues detected"
    exit 1
fi

# Check for orphans
ORPHANS=$(truenas-monitor --config /etc/config.yaml orphans --format json | jq 'length')
if [ "$ORPHANS" -gt 10 ]; then
    echo "⚠️  High orphan count: $ORPHANS"
    # Send alert
fi

# Snapshot health
truenas-monitor --config /etc/config.yaml snapshots --health --format json > /tmp/snapshot-health.json
ALERTS=$(jq '.alerts | length' /tmp/snapshot-health.json)
if [ "$ALERTS" -gt 0 ]; then
    echo "⚠️  Snapshot alerts: $ALERTS"
    jq '.alerts' /tmp/snapshot-health.json
fi

echo "✅ Daily check completed"
```

### 2. Performance Monitoring

```bash
# Weekly performance report
#!/bin/bash
# weekly-report.sh

echo "Weekly Storage Performance Report - $(date)"
echo "============================================"

# Storage efficiency
truenas-monitor analyze --format json | jq '
{
  "storage_efficiency": .storage_efficiency,
  "recommendations": .recommendations,
  "total_capacity_gb": (.total_capacity_bytes / 1073741824 | round),
  "used_capacity_gb": (.total_used_bytes / 1073741824 | round),
  "utilization_percent": ((.total_used_bytes / .total_capacity_bytes) * 100 | round)
}'

# Snapshot statistics
truenas-monitor snapshots --analysis --format json | jq '
{
  "total_snapshots": .total_snapshots,
  "total_size_gb": (.total_snapshot_size / 1073741824 | round),
  "average_age_days": .average_snapshot_age_days,
  "large_snapshots": (.large_snapshots | length),
  "recommendations": .recommendations
}'
```

### 3. Automated Cleanup

```bash
# Cleanup script (use with caution)
#!/bin/bash
# cleanup-orphans.sh

# Find orphaned resources older than 7 days
truenas-monitor orphans --format json | jq -r '
  .[] | select(.age_days > 7) | 
  "kubectl delete pv " + .name
' > cleanup-commands.sh

# Review before executing
echo "Review cleanup commands:"
cat cleanup-commands.sh

# Uncomment to execute (BE CAREFUL!)
# bash cleanup-commands.sh
```

### 4. Upgrade Procedures

```bash
# Upgrade to new version
#!/bin/bash
# upgrade.sh

VERSION="v0.3.0"  # Update as needed

# Backup current configuration
kubectl get configmap truenas-monitor-config -o yaml > backup-$(date +%Y%m%d).yaml

# Update image
kubectl set image deployment/truenas-monitor \
  monitor=ghcr.io/yourusername/truenas-monitor:$VERSION

# Wait for rollout
kubectl rollout status deployment/truenas-monitor

# Verify new version
kubectl exec deployment/truenas-monitor -- truenas-monitor --version

# Test functionality
kubectl exec deployment/truenas-monitor -- truenas-monitor validate
```

---

## Quick Reference

### Essential Commands

```bash
# Basic operations
truenas-monitor validate                    # Test connectivity
truenas-monitor orphans                     # Find orphaned resources
truenas-monitor analyze                     # Storage analysis
truenas-monitor snapshots --health          # Snapshot health check
truenas-monitor monitor --once              # Single monitoring cycle

# Output formats
truenas-monitor orphans --format json       # JSON output
truenas-monitor snapshots --format yaml     # YAML output

# Filtering
truenas-monitor snapshots --age-days 30     # Old snapshots
truenas-monitor snapshots --volume pvc-name # Specific volume

# Monitoring
truenas-monitor monitor --interval 300      # Continuous monitoring
truenas-monitor monitor --metrics-port 8080 # With Prometheus metrics
```

### Configuration Files

- **config.yaml** - Main configuration
- **rbac.yaml** - Kubernetes permissions
- **deployment.yaml** - Kubernetes deployment
- **backup-*.yaml** - Configuration backups

This guide provides comprehensive instructions for testing and deploying the TrueNAS Storage Monitor. Start with the Quick Test Setup and progress through the sections based on your deployment needs.