# Kubernetes TrueNAS Democratic Tool

[![CI Pipeline](https://github.com/tomazb/kubernetes-truenas-democratic-tool/actions/workflows/ci.yml/badge.svg)](https://github.com/tomazb/kubernetes-truenas-democratic-tool/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/tomazb/kubernetes-truenas-democratic-tool)](https://goreportcard.com/report/github.com/tomazb/kubernetes-truenas-democratic-tool)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A comprehensive monitoring and management tool for OpenShift/Kubernetes clusters using TrueNAS Scale storage via democratic-csi.

## Overview

This tool analyzes and monitors the integration between OpenShift, TrueNAS Scale, and democratic-csi to identify configuration issues, orphaned resources, and ensure best practices.

### Key Features

- **Orphaned Resource Detection** - Identify PVs, volumes, and snapshots without corresponding resources
- **Snapshot Management** - Track snapshot growth, retention, and storage consumption
- **Configuration Validation** - Verify StorageClass, CSI driver, and RBAC configurations
- **Storage Analytics** - Monitor thin provisioning efficiency and capacity trends
- **Security-First Design** - Zero-trust architecture with comprehensive audit logging
- **Idempotent Operations** - All operations are safe to retry

## Architecture

The tool uses a hybrid Go/Python architecture:

- **Go Components** - Performance-critical monitoring, API server, and controller
- **Python Components** - CLI tool, analysis engine, and integrations

## Quick Start

### Prerequisites

- Kubernetes/OpenShift cluster with democratic-csi
- TrueNAS Scale with API access
- Go 1.21+ (for development)
- Python 3.10+ (for CLI)

### Installation

#### CLI Tool (Python)

```bash
pip install truenas-storage-monitor
```

#### Container Deployment

```bash
# Deploy monitoring stack
helm install truenas-monitor ./charts/truenas-monitor \
  --namespace storage-monitoring \
  --create-namespace
```

### Basic Usage

```bash
# Check for orphaned resources
truenas-monitor orphans

# Analyze storage usage
truenas-monitor analyze --trend 30d

# Generate HTML report
truenas-monitor report --output report.html

# Validate configuration
truenas-monitor validate
```

## Configuration

Create a `config.yaml` file:

```yaml
openshift:
  kubeconfig: ~/.kube/config
  namespace: democratic-csi

truenas:
  url: https://truenas.example.com
  username: admin
  password: ${TRUENAS_PASSWORD}  # Use environment variable

monitoring:
  orphan_threshold: 24h
  snapshot_retention: 30d
  
alerts:
  slack:
    webhook: ${SLACK_WEBHOOK}
```

## Development

### Test-Driven Development

This project follows strict TDD practices:

```bash
# Run unit tests
make test-unit

# Run all tests with coverage
make test-all

# Run in watch mode
make test-watch
```

### Building

```bash
# Build all components
make build-all

# Build containers (using Podman)
make container-build-all
```

## Documentation

- [Architecture](docs/ARCHITECTURE.md) - System design and components
- [PRD](docs/PRD.md) - Product requirements and roadmap
- [CLAUDE.md](CLAUDE.md) - Development guidelines
- [API Reference](https://yourusername.github.io/kubernetes-truenas-democratic-tool/)

## Security

This tool follows security best practices:

- Zero-trust architecture
- Minimal RBAC permissions
- No credentials in logs
- TLS 1.3+ for all connections
- Regular security scans via GitHub Actions

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests first (TDD)
4. Commit your changes (`git commit -s -m 'Add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/yourusername/kubernetes-truenas-democratic-tool/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/kubernetes-truenas-democratic-tool/discussions)
- **Security**: See [SECURITY.md](.github/SECURITY.md)

## Roadmap

See our [Product Requirements Document](docs/PRD.md) for the complete roadmap. Key upcoming features:

- Grafana integration
- Auto-remediation framework
- ML-based storage predictions
- Multi-cluster support

## Acknowledgments

- OpenShift/Kubernetes community
- TrueNAS Scale team
- democratic-csi project