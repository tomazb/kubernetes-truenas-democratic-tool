# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- GitHub repository location in CLAUDE.md for reference

### Changed
- Migrated from Docker to Podman for all containerization
- Renamed Dockerfiles to Containerfiles following Podman conventions
- Updated all build commands and documentation to use Podman
- Modified GitHub Actions workflows to use Podman and buildah

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