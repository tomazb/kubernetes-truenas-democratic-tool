# CLAUDE.md

This file provides guidance to AI assistants when working with code in this repository.

## Project Overview

This tool analyzes and monitors the integration between OpenShift, TrueNAS Scale, and democratic-csi to identify configuration issues, orphaned resources, and ensure best practices.

### What Makes This Project Exceptional

1. **Fills a Critical Gap** - No existing tool comprehensively monitors the OpenShift + TrueNAS + democratic-csi stack
2. **Production-Ready Security** - Security-first design with zero-trust architecture
3. **True Idempotency** - Every operation is safe to retry, critical for automation
4. **Test-Driven** - 90%+ test coverage ensures reliability
5. **Cloud-Native** - First-class Kubernetes/OpenShift support
6. **Actionable Insights** - Not just monitoring, but recommendations and auto-remediation
7. **Open Source** - Community-driven development with enterprise features

### Key Differentiators

- **Holistic View** - Correlates data across all three systems
- **Storage Intelligence** - Understands thin provisioning, snapshots, and storage efficiency
- **Preventive Care** - Predicts issues before they occur
- **Automation-Friendly** - API-first design for GitOps integration
- **Cost-Conscious** - Helps optimize storage costs

## GitHub Repository Structure

### Branch Strategy
- `main` - Production-ready code, protected branch
- `develop` - Integration branch for features
- `feature/*` - Feature development branches
- `bugfix/*` - Bug fix branches
- `security/*` - Security-related fixes

### Pull Request Requirements
1. All changes via PR - no direct commits to main/develop
2. Minimum 1 approval required
3. All CI checks must pass
4. Code coverage must not decrease
5. Security scan must pass
6. DCO sign-off required

### GitHub Free Plan Features Utilized

#### Security
- **Dependabot** - Automated dependency updates
- **Secret scanning** - Prevent credential leaks
- **Code scanning** - CodeQL analysis for vulnerabilities
- **Security advisories** - Private vulnerability reporting
- **Branch protection** - Enforce PR reviews and status checks

#### Automation (GitHub Actions - 2000 minutes/month)
```yaml
# .github/workflows/ci.yml
name: CI Pipeline
on:
  pull_request:
  push:
    branches: [main, develop]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        python-version: [3.10, 3.11]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v4
      - run: |
          pip install -r requirements-dev.txt
          pytest tests/unit/ --cov --cov-fail-under=90
      - uses: codecov/codecov-action@v3

  security-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: aquasecurity/trivy-action@master
      - uses: pyupio/safety-action@v1
      - run: pip install bandit && bandit -r src/

  build-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/build-push-action@v5
        with:
          context: .
          push: false
          tags: test:latest
```

#### Testing & Quality
- **GitHub Pages** - Host test coverage reports
- **Issue templates** - Standardize bug reports
- **PR templates** - Ensure quality checklist
- **CODEOWNERS** - Automatic review assignments
- **Status badges** - Show build/coverage status

### Repository Files
```
.github/
├── workflows/
│   ├── ci.yml              # Main CI pipeline
│   ├── codeql.yml          # Security analysis
│   └── release.yml         # Automated releases
├── dependabot.yml          # Dependency updates
├── ISSUE_TEMPLATE/
│   ├── bug_report.md
│   └── feature_request.md
├── pull_request_template.md
├── CODEOWNERS
└── SECURITY.md

# Branch protection rules (via GitHub UI)
# - Require PR reviews
# - Dismiss stale reviews
# - Require status checks
# - Include administrators
```

### Cost-Effective CI Strategy
1. Use matrix builds sparingly (2 Python versions max)
2. Cache dependencies between runs
3. Run expensive tests only on main/develop
4. Use workflow conditions to skip unnecessary runs
5. Leverage free CodeQL scans for security

## Development Philosophy

### Test-Driven Development (TDD)
1. **Red** - Write a failing test that defines desired behavior
2. **Green** - Write minimal code to make the test pass
3. **Refactor** - Improve code while keeping tests green

### Testing Principles
- **Unit tests come first** - No code without a test
- **Fast feedback** - Unit tests must run in milliseconds
- **Isolation** - Mock all external dependencies
- **Documentation** - Tests serve as living documentation
- **Coverage** - Maintain minimum 90% code coverage

## Architecture

### Security-First Design Principles
1. **Zero Trust Architecture** - Never assume trust, always verify
2. **Principle of Least Privilege** - Minimal permissions required for each operation
3. **Defense in Depth** - Multiple security layers
4. **Secure by Default** - All configurations secure out of the box
5. **Audit Everything** - Comprehensive logging of all actions

### Idempotency Guarantees
1. **Read-Only by Default** - All operations are non-destructive unless explicitly requested
2. **Deterministic Operations** - Same input always produces same output
3. **State Validation** - Check current state before any operation
4. **Transaction Safety** - All-or-nothing operations with rollback capability
5. **Retry Safety** - Operations can be safely retried without side effects

### Hybrid Architecture (Go + Python)

#### Go Components (Performance-Critical Core)

1. **Monitoring Service** (95% coverage required)
   - Real-time resource watching (PVs, PVCs, Snapshots)
   - Prometheus metrics exporter
   - Event stream processing
   - WebSocket API for real-time updates
   - Language: Go with client-go

2. **API Server** (95% coverage required)
   - RESTful API for all operations
   - Authentication/authorization middleware
   - Rate limiting and caching
   - OpenAPI/Swagger documentation
   - Language: Go with Gin/Echo framework

3. **Resource Controller** (100% coverage required)
   - Kubernetes controller pattern
   - Reconciliation loops
   - State management
   - Webhook handlers
   - Language: Go with controller-runtime

#### Python Components (Flexibility & Analysis)

4. **CLI Tool** (90% coverage required)
   - User-friendly command interface
   - Report generation (HTML/PDF)
   - Interactive troubleshooting
   - Data visualization
   - Language: Python with Click

5. **Analysis Engine** (95% coverage required)
   - Storage trend analysis
   - Capacity planning algorithms
   - Anomaly detection
   - Cost optimization recommendations
   - Language: Python with Pandas/NumPy

6. **Integration Scripts** (85% coverage required)
   - Slack/Teams notifications
   - JIRA/ServiceNow ticket creation
   - Custom webhooks
   - Data export utilities
   - Language: Python

#### Shared Components

7. **TrueNAS Client Library**
   - Go package for monitoring service
   - Python package for CLI/scripts
   - Shared API definitions
   - Common test fixtures

8. **Configuration Management**
   - YAML/JSON schemas
   - Environment variable support
   - Secret management integration
   - Validation libraries for both languages

### Key Resources to Monitor
- PersistentVolumes (PV) and PersistentVolumeClaims (PVC)
- StorageClasses with democratic-csi provisioner
- CSI nodes, drivers, and volume attachments
- TrueNAS volumes, datasets, and iSCSI/NFS shares
- VolumeSnapshots and VolumeSnapshotContents in OpenShift
- TrueNAS ZFS snapshots and their retention
- Snapshot policies and schedules
- Storage capacity tracking (allocated vs used)
- Thin provisioning metrics

## Deployment Options

### 1. Command Line Usage
```bash
# Python CLI tool
pip install truenas-storage-monitor
truenas-monitor check
truenas-monitor orphans --format json
truenas-monitor analyze --trend 30d
truenas-monitor report --output report.html

# Direct API access via curl
curl -H "Authorization: Bearer $TOKEN" \
     http://monitor-api:8080/api/v1/orphans
```

### 2. Container Usage
```bash
# Run monitoring service (Go)
docker run -d --name monitor \
           -v ~/.kube:/home/app/.kube:ro \
           -v ./config.yaml:/app/config.yaml:ro \
           ghcr.io/yourusername/truenas-monitor:latest

# Run CLI tool (Python)
docker run --rm -it \
           -v ~/.kube:/home/app/.kube:Z \
           -v ./config.yaml:/app/config.yaml:Z \
           ghcr.io/yourusername/truenas-cli:latest analyze
```

### 3. Kubernetes/OpenShift Deployment
```bash
# Deploy complete stack
helm install truenas-monitor ./charts/truenas-monitor \
  --namespace storage-monitoring \
  --create-namespace

# Components deployed:
# - Monitor Service (Go) - Deployment
# - API Server (Go) - Deployment  
# - Analysis CronJob (Python) - CronJob
# - CLI Pod (Python) - Pod for ad-hoc queries

# Access the API
oc port-forward svc/monitor-api 8080:8080
curl http://localhost:8080/api/v1/health
```

## Test-Driven Development

### Testing Hierarchy (Bottom-Up)
1. **Unit Tests** - Test individual functions and classes in isolation
2. **Integration Tests** - Test component interactions
3. **End-to-End Tests** - Test complete workflows
4. **Security Tests** - Validate security controls
5. **Idempotency Tests** - Ensure operation safety

### Development Workflow
```bash
# 1. Write failing unit test first
pytest tests/unit/test_new_feature.py -v

# 2. Implement minimal code to pass
python -m pytest tests/unit/test_new_feature.py::test_specific_case -v

# 3. Refactor with confidence
pytest tests/unit/ --cov=src --cov-fail-under=90

# 4. Run integration tests
pytest tests/integration/ -v

# 5. Validate security
pytest tests/security/ -v
bandit -r src/

# 6. Check idempotency
pytest tests/idempotency/ --run-twice

# Full test suite with coverage
pytest --cov=src --cov-report=html --cov-report=term-missing

# Watch mode for TDD
ptw tests/unit/ -- -v
```

### Test Structure
```
tests/
├── unit/                    # Pure unit tests (mocked dependencies)
│   ├── test_openshift_client.py
│   ├── test_truenas_client.py
│   ├── test_reconciler.py
│   ├── test_storage_analyzer.py
│   └── test_snapshot_manager.py
├── integration/             # Component integration tests
│   ├── test_api_integration.py
│   └── test_csi_integration.py
├── e2e/                    # End-to-end scenarios
│   └── test_workflows.py
├── security/               # Security-specific tests
│   ├── test_authentication.py
│   ├── test_authorization.py
│   └── test_encryption.py
├── idempotency/           # Idempotency verification
│   └── test_operations.py
└── fixtures/              # Shared test data
    └── mock_data.py
```

### Unit Test Requirements
- Minimum 90% code coverage
- All public methods must have tests
- Mock all external dependencies
- Test both success and failure paths
- Validate error messages don't leak secrets

## Development Commands

### Go Components
```bash
# Run Go tests
go test ./... -v -cover

# Build Go binaries
go build -o bin/monitor ./cmd/monitor
go build -o bin/api-server ./cmd/api-server

# Go linting
golangci-lint run
go vet ./...

# Go security scanning
gosec ./...
```

### Python Components
```bash
# Python virtual environment
python -m venv venv
source venv/bin/activate

# Run Python tests
pytest tests/python/ --cov=python --cov-report=html

# Python linting and formatting
black python/ --check
flake8 python/
mypy python/
bandit -r python/

# Build Python package
python -m build
```

### Combined Operations
```bash
# Run all tests
make test-all  # Runs both Go and Python tests

# Build everything
make build-all  # Builds Go binaries and Python packages

# Security scan all components
make security-scan

# Container builds
make docker-build-monitor  # Go monitoring service
make docker-build-cli      # Python CLI tools
```

### Container Architecture
```dockerfile
# Go service container (15-40MB)
FROM golang:1.21 AS go-builder
# ... build steps ...
FROM gcr.io/distroless/static:nonroot
COPY --from=go-builder /app/monitor /monitor

# Python CLI container (150MB)
FROM python:3.11-slim
# ... install dependencies ...
```

## Configuration

The tool expects configuration in `config.yaml`:
- OpenShift cluster connection details
- TrueNAS Scale API endpoint and credentials
- Democratic-CSI namespace and configuration
- Thresholds and alert settings

### Security Configuration

#### Authentication & Authorization
```yaml
# RBAC permissions (minimal required)
- apiGroups: [""]
  resources: ["persistentvolumes", "persistentvolumeclaims"]
  verbs: ["get", "list"]
- apiGroups: ["storage.k8s.io"]
  resources: ["storageclasses", "volumeattachments"]
  verbs: ["get", "list"]
- apiGroups: ["snapshot.storage.k8s.io"]
  resources: ["volumesnapshots", "volumesnapshotcontents"]
  verbs: ["get", "list"]
```

#### Secret Management
- Kubernetes Secrets for credentials (never in ConfigMaps)
- Support for external secret managers (Vault, Sealed Secrets)
- Automatic credential rotation
- No credentials in logs or error messages

#### Network Security
- TLS 1.3 minimum for all connections
- Certificate pinning for TrueNAS API
- Network policies for pod-to-pod communication
- Egress restrictions to known endpoints only

### OpenShift-Specific Features
- Automatic ServiceAccount authentication when running in-cluster
- SecurityContextConstraints (SCC) - restricted by default
- Route exposure with TLS edge termination
- Integration with OpenShift monitoring stack (Prometheus)
- Compliance with OpenShift security policies

### Directory Structure
```
.
├── go/                     # Go components
│   ├── cmd/
│   │   ├── monitor/       # Monitoring service
│   │   ├── api-server/    # REST API server
│   │   └── controller/    # K8s controller
│   ├── pkg/
│   │   ├── k8s/          # Kubernetes client
│   │   ├── truenas/      # TrueNAS client
│   │   ├── metrics/      # Prometheus metrics
│   │   └── api/          # API definitions
│   └── tests/
├── python/                 # Python components
│   ├── cli/              # CLI tool
│   ├── analysis/         # Analysis engine
│   ├── integrations/     # Slack, JIRA, etc.
│   ├── reports/          # Report generation
│   └── tests/
├── shared/                 # Shared between Go/Python
│   ├── schemas/          # JSON/YAML schemas
│   ├── fixtures/         # Test data
│   └── docs/            # API documentation
├── deploy/
│   ├── kubernetes/       # Standard K8s manifests
│   ├── openshift/       # OpenShift-specific
│   └── helm/            # Helm charts
└── scripts/               # Build and deploy scripts
```

### Communication Between Components
```yaml
# API Communication Flow
Python CLI -> Go API Server -> Go Monitor Service
     ↓              ↓                    ↓
   Reports    Prometheus          Kubernetes API
              Metrics             TrueNAS API
```

## Key Analysis Areas

1. **Orphaned Resources Detection**
   - PVs without corresponding TrueNAS volumes
   - TrueNAS volumes without OpenShift PVs
   - Unbound PVCs older than threshold
   - Stale volume attachments
   - Orphaned snapshots in TrueNAS without VolumeSnapshot objects
   - VolumeSnapshots without corresponding ZFS snapshots

2. **Snapshot Management**
   - Track snapshot count per volume on TrueNAS
   - Map TrueNAS snapshots to OpenShift VolumeSnapshot objects
   - Identify snapshot growth trends and storage consumption
   - Detect failed or stuck snapshot operations
   - Monitor snapshot age and retention policy compliance
   - Alert on excessive snapshot accumulation
   - Verify snapshot restoration capabilities

3. **Configuration Validation**
   - StorageClass parameters match TrueNAS capabilities
   - CSI driver pods are healthy
   - Node plugins are running on all nodes
   - Proper RBAC permissions
   - VolumeSnapshotClass configuration
   - Snapshot retention policies alignment

4. **Performance Monitoring**
   - Volume provisioning times
   - Mount/unmount operation durations
   - TrueNAS pool usage and performance
   - Snapshot creation/deletion times
   - Snapshot space consumption trends

5. **Storage Usage Analysis**
   - Compare allocated (virtual) vs actual (used) storage per volume
   - Track thin provisioning efficiency ratios
   - Monitor overcommitment levels on TrueNAS pools
   - Identify volumes with high actual usage vs allocated
   - Detect volumes approaching capacity limits
   - Calculate total storage savings from thin provisioning
   - Alert on dangerous overcommitment ratios

6. **Best Practices Checks**
   - Snapshot policies configured
   - Resource quotas and limits
   - High availability setup
   - Backup verification
   - Snapshot scheduling optimization
   - Storage efficiency recommendations
   - Thin provisioning thresholds
   - Pool capacity planning

## Idempotent Operation Patterns

### Command Execution
- All commands can be run multiple times safely
- No side effects from repeated execution
- State checking before any operation
- Clear distinction between read and write operations

### Error Handling
- Graceful degradation on partial failures
- No partial state modifications
- Comprehensive rollback mechanisms
- Detailed error reporting without exposing secrets

### Caching Strategy
- Immutable cache keys based on resource versions
- TTL-based cache expiration
- Cache invalidation on configuration changes
- Separate caches for different security contexts

## Future Enhancements

### 1. Intelligent Automation
- **Auto-remediation** - Fix common issues automatically (with approval workflow)
- **Predictive Analytics** - ML-based prediction of storage exhaustion
- **Smart Scheduling** - Optimize snapshot schedules based on usage patterns
- **Anomaly Detection** - Identify unusual storage consumption patterns

### 2. Enhanced Observability
- **Real-time Dashboard** - Web-based monitoring dashboard
- **Grafana Integration** - Export metrics to Prometheus/Grafana
- **Historical Trending** - Track resource usage over time
- **SLO/SLI Tracking** - Monitor storage availability objectives

### 3. Advanced Integrations
- **Notification Channels** - Slack, Teams, PagerDuty, Email
- **Ticketing Systems** - Auto-create JIRA/ServiceNow tickets
- **ChatOps** - Slack/Teams bot for queries and actions
- **Webhook Support** - Generic webhook for custom integrations

### 4. Operational Features
- **Backup Validation** - Verify backup integrity regularly
- **DR Testing** - Automated disaster recovery validation
- **Cost Analysis** - Calculate storage costs and optimization opportunities
- **Compliance Reports** - Generate audit-ready compliance reports

### 5. Developer Experience
- **Plugin System** - Extensible architecture for custom checks
- **REST API** - Full-featured API for integration
- **Terraform Provider** - Manage monitoring via IaC
- **Operator Pattern** - Kubernetes operator for automated management

### 6. Performance & Scale
- **Distributed Mode** - Scale across multiple clusters
- **Event Streaming** - Kafka integration for real-time events
- **Time-series DB** - InfluxDB/TimescaleDB for metrics
- **Caching Layer** - Redis for improved performance

### 7. User Interface Options
- **TUI (Terminal UI)** - Rich terminal interface with charts
- **Mobile App** - View alerts and metrics on mobile
- **Voice Alerts** - Critical alerts via phone call
- **AR/VR Visualization** - 3D storage visualization (future)

### Implementation Priority
1. **Phase 1** - Grafana integration, Slack notifications
2. **Phase 2** - Web dashboard, auto-remediation framework
3. **Phase 3** - ML-based predictions, plugin system
4. **Phase 4** - Distributed mode, advanced visualizations

### 8. Community & Ecosystem
- **Runbook Automation** - Pre-built remediation playbooks
- **Community Checks** - Marketplace for custom health checks
- **Benchmarking** - Compare metrics with anonymized community data
- **Multi-tenancy** - Support multiple teams with RBAC
- **Internationalization** - Multi-language support

### 9. Sustainability & Efficiency
- **Carbon Tracking** - Monitor power usage effectiveness
- **Resource Optimization** - Suggest consolidation opportunities
- **Idle Resource Detection** - Find and reclaim unused storage
- **Green Scheduling** - Run intensive tasks during renewable energy availability

### 10. Enterprise Features
- **Audit Logging** - Immutable audit trail with blockchain option
- **Compliance Frameworks** - HIPAA, PCI-DSS, SOC2 templates
- **SLA Management** - Track and report on storage SLAs
- **Change Management** - Integration with change approval systems
- **Multi-cluster Federation** - Manage multiple OpenShift clusters

### Quick Wins (Implement First)
1. **Prometheus Metrics Export** - Easy Grafana integration
2. **Slack Webhook** - Simple alerting
3. **JSON Output Mode** - Machine-readable reports
4. **Dry-run Everything** - Safe exploration mode
5. **Shell Completion** - Better CLI experience

## Important Guidelines

### No Promotional Content
- **NEVER** include promotional text or branding for any AI assistant tools
- **NEVER** add "Generated with [Tool Name]" messages in code, commits, or documentation
- **NEVER** include AI assistant attribution in commit messages
- Keep all code and documentation focused purely on the project functionality
- Maintain professional, tool-agnostic content throughout the codebase