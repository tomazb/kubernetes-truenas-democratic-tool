# Go Components Documentation Update

## Installation & Build Instructions

### Prerequisites
- Go 1.21 or higher
- Kubernetes cluster access
- TrueNAS Scale with API access

### Building Go Components

```bash
# Clone repository
git clone https://github.com/tomazb/kubernetes-truenas-democratic-tool.git
cd kubernetes-truenas-democratic-tool/go

# Install dependencies
go mod download

# Build API Server
go build -o bin/api-server ./cmd/api-server

# Build Monitor Service  
go build -o bin/monitor ./cmd/monitor

# Verify builds
./bin/api-server --help
./bin/monitor --help
```

## Go Services Overview

### 1. API Server (`bin/api-server`)
RESTful API server providing access to TrueNAS storage monitoring data and operations.

**Available Endpoints:**
- `GET /health` - Basic health check
- `GET /ready` - Readiness check with real Kubernetes and TrueNAS connectivity validation
- `GET /version` - Version information with build metadata
- `GET /api/v1/status` - System status
- `GET /api/v1/validate` - Configuration validation with connectivity tests
- `GET /api/v1/orphans` - List orphaned resources
- `POST /api/v1/orphans/cleanup` - Cleanup orphaned resources with detailed workflow
- `GET /api/v1/storage` - Storage usage analysis
- `GET /api/v1/csi/health` - CSI driver health
- `GET /api/v1/snapshots` - Snapshot management with analysis
- `POST /api/v1/reports` - Generate comprehensive reports

**Enhanced Health Checks:**
The API server includes comprehensive health validation:
- `/health` - Basic service health
- `/ready` - Full connectivity validation to Kubernetes API and TrueNAS
- `/api/v1/validate` - Detailed configuration validation with authentication tests

**Usage:**
```bash
# Start API server
./bin/api-server --config config.yaml --listen :8080

# Test endpoints
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/status
```

### 2. Monitor Service (`bin/monitor`)
Continuous monitoring daemon that watches for orphaned resources and exports Prometheus metrics.

**Features:**
- Real-time Kubernetes resource monitoring
- TrueNAS integration for storage data
- Prometheus metrics export
- Configurable monitoring intervals

**Usage:**
```bash
# Start monitor service
./bin/monitor --config config.yaml --log-level info

# Start with JSON logging
./bin/monitor --config config.yaml --json-logs

# Run once (no continuous monitoring)
./bin/monitor --config config.yaml --once
```

## Configuration

### Complete Configuration Example
```yaml
# config.yaml - Full configuration for Go services
openshift:
  kubeconfig: ~/.kube/config
  namespace: democratic-csi
  storage_class: democratic-csi-nfs
  csi_driver: org.democratic-csi.nfs

truenas:
  url: https://truenas.example.com
  username: admin
  password: ${TRUENAS_PASSWORD}
  verify_ssl: true
  timeout: 30

monitoring:
  orphan_check_interval: 1h
  orphan_threshold: 24h
  workers: 10
  batch_size: 100

api:
  listen: ":8080"
  tls:
    enabled: false

metrics:
  enabled: true
  port: 9090
  path: /metrics

logging:
  level: info
  format: json
  output: stdout
```

## Kubernetes Deployment

### API Server Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: truenas-api-server
  namespace: monitoring
spec:
  replicas: 2
  selector:
    matchLabels:
      app: truenas-api-server
  template:
    metadata:
      labels:
        app: truenas-api-server
    spec:
      containers:
      - name: api-server
        image: truenas-monitor:latest
        command: ["/app/bin/api-server"]
        args: ["--config", "/etc/config/config.yaml"]
        ports:
        - containerPort: 8080
          name: http
        volumeMounts:
        - name: config
          mountPath: /etc/config
        env:
        - name: TRUENAS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: truenas-credentials
              key: password
      volumes:
      - name: config
        configMap:
          name: truenas-config
---
apiVersion: v1
kind: Service
metadata:
  name: truenas-api-server
  namespace: monitoring
spec:
  selector:
    app: truenas-api-server
  ports:
  - port: 8080
    targetPort: 8080
    name: http
```

### Monitor Service Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: truenas-monitor
  namespace: monitoring
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
        image: truenas-monitor:latest
        command: ["/app/bin/monitor"]
        args: ["--config", "/etc/config/config.yaml"]
        ports:
        - containerPort: 9090
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
      volumes:
      - name: config
        configMap:
          name: truenas-config
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
  - port: 9090
    targetPort: 9090
    name: metrics
```

## Integration with Python CLI

The Go services are designed to work alongside the existing Python CLI:

```bash
# Use Python CLI with Go API server
truenas-monitor --api-url http://localhost:8080 orphans

# Monitor with Go services, analyze with Python
./bin/monitor --config config.yaml &
truenas-monitor analyze --format json
```

## Prometheus Metrics

Both Go services export Prometheus metrics:

```bash
# API Server metrics
curl http://localhost:8080/metrics

# Monitor service metrics  
curl http://localhost:9090/metrics
```

**Key Metrics:**
- `truenas_pvs_total` - Total persistent volumes
- `truenas_pvcs_total` - Total persistent volume claims
- `truenas_orphaned_pvs` - Orphaned persistent volumes
- `truenas_snapshots_total` - Total snapshots
- `truenas_storage_pool_usage` - Storage pool utilization