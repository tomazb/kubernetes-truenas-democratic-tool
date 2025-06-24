# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Comprehensive snapshot management functionality
  - Snapshot health monitoring across K8s and TrueNAS
  - Orphaned snapshot detection in both systems
  - Snapshot usage analysis with size and age metrics
  - Smart recommendations for snapshot optimization
  - Multi-format output support (table, JSON, YAML)
- Enhanced CLI commands for snapshot operations
  - `snapshots --health` for comprehensive health status
  - `snapshots --analysis` for detailed usage analysis
  - `snapshots --orphaned` for orphan detection
  - `snapshots --age-days` for filtering by age
- Storage efficiency analysis
  - Thin provisioning ratio calculations
  - Snapshot overhead percentage tracking
  - Pool utilization and fragmentation monitoring
  - Automated recommendations for optimization
- Monitor class enhancements
  - `check_snapshot_health()` method with alert generation
  - `analyze_storage_efficiency()` method for capacity planning
  - Integrated orphaned resource detection for snapshots
- Python implementation of all core CLI commands
  - `orphans` command with table/JSON/YAML output
  - `analyze` command for storage usage analysis
  - `validate` command for configuration validation
  - `monitor` command with Prometheus metrics support
  - Enhanced `snapshots` command with multiple options
- Integration test suite structure
  - K8s integration tests with service availability checks
  - TrueNAS integration tests with connection validation
  - End-to-end workflow tests
  - Proper pytest markers and skip conditions
- Enhanced error handling
  - Timezone-aware datetime comparisons
  - Proper attribute validation in data classes
  - Comprehensive exception handling in all clients
- Demo guide and test scripts
  - DEMO.md with comprehensive usage examples
  - test_snapshot_functionality.py demonstrating all features
- Subagent usage guidelines in CLAUDE.md

### Changed
- Migrated from Docker to Podman for all containerization
- Renamed Dockerfiles to Containerfiles following Podman conventions
- Updated all build commands and documentation to use Podman
- Modified GitHub Actions workflows to use Podman and buildah
- Enhanced K8s client with snapshot-specific methods
  - `find_orphaned_snapshots()` for K8s orphan detection
  - `find_stale_snapshots()` for age-based filtering
- Enhanced TrueNAS client with snapshot analysis
  - `find_orphaned_truenas_snapshots()` for TrueNAS orphan detection
  - `analyze_snapshot_usage()` for comprehensive metrics
  - Improved snapshot data models with size tracking
- Updated shared schemas to include snapshot analysis structures

### Fixed
- Import errors in Python package initialization
- YAML parsing errors in test configurations
- URL vs host parameter mismatch in TrueNAS configuration
- Attribute name inconsistencies in snapshot data classes
- Missing datetime imports for timedelta operations
- Test coverage reporting configuration

## [0.1.0-alpha] - 2025-06-23

### Added
- Initial project structure with hybrid Go/Python architecture
- Kubernetes client library (Go) with comprehensive test coverage
  - PersistentVolume operations and watching
  - PersistentVolumeClaim operations and watching
  - VolumeSnapshot operations and watching
  - StorageClass and CSI driver monitoring
  - Health check functionality
- Kubernetes client wrapper (Python) with test coverage
  - Integration with official Python client
  - Orphaned resource detection
  - Real-time event watching capabilities
- TrueNAS REST API client (Go) with full functionality
  - Pool and dataset management
  - iSCSI volume operations
  - Snapshot management
  - Orphaned volume detection
- TrueNAS REST API client (Python) with comprehensive features
  - Complete REST API coverage
  - Pagination support
  - Error handling and retry logic
- Shared JSON schemas for data exchange
  - Orphaned resources report schema
  - Storage analysis report schema
  - Configuration validation report schema
- Schema validation utilities for both Go and Python
- GitHub Actions CI/CD pipeline
  - Multi-stage testing (unit, integration, security)
  - Container building with Podman
  - Automated releases
  - Security scanning with CodeQL
- Project documentation
  - Comprehensive README with quick start guide
  - Architecture document with 20+ Mermaid diagrams
  - Product Requirements Document (PRD)
  - Go vs Python comparison analysis
  - Development guidelines (CLAUDE.md)
- Security features
  - Branch protection on main branch
  - Dependabot enabled for security updates
  - Secret scanning enabled
  - No promotional content policy
- Development infrastructure
  - Makefile with all common operations
  - Python virtual environment setup
  - Go module configuration
  - Multi-stage container builds
  - Kubernetes deployment manifests

### Security
- Configured minimal RBAC permissions
- Implemented secure credential handling
- Added comprehensive .gitignore for sensitive files
- Enabled GitHub security features

[Unreleased]: https://github.com/tomazb/kubernetes-truenas-democratic-tool/compare/v0.1.0-alpha...HEAD
[0.1.0-alpha]: https://github.com/tomazb/kubernetes-truenas-democratic-tool/releases/tag/v0.1.0-alpha