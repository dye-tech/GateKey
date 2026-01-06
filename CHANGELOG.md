# Changelog

All notable changes to GateKey are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.0] - 2026-01-06

### Added
- WireGuard VPN protocol support for Android client
- WireGuard gateway integration with control plane
- WireGuard peer synchronization between control plane and gateways

### Changed
- Cleaned up deprecated code and TODO comments
- Removed bundled binaries from repository

### Fixed
- WireGuard gateway peer synchronization issues
- Android VPN service lifecycle management

## [1.2.1] - 2026-01-05

### Fixed
- Bug fixes for gateway connections
- Login session handling improvements
- Connection stability improvements

## [1.2.0] - 2026-01-04

### Added
- Android client documentation
- Gateway connection management
- Geofencing support for access control
- Mesh connection enhancements
- Real-time monitoring dashboard
- Local groups for user management
- API key rotation feature

### Changed
- Schema updates for new features
- UI improvements across dashboard

## [1.1.5] - 2026-01-02

### Changed
- Updated Docker configurations
- README documentation updates

### Fixed
- OpenVPN disconnect bug - process not being properly terminated
- Lint issues across codebase

## [1.1.4] - 2026-01-02

### Added
- Helm chart support for admin user variable configuration

## [1.1.3] - 2026-01-02

### Changed
- Updated Homebrew formula

## [1.1.2] - 2026-01-02

### Fixed
- Access routes in CLI client

## [1.1.1] - 2026-01-02

### Fixed
- Access routes configuration
- Install scripts - removed easy-rsa dependency
- Amazon Linux 3 support

## [1.1.0] - 2026-01-01

### Added
- Remote sessions feature for gateway management
- Network topology visualization
- Troubleshooting tools and diagnostics
- Config management for administrators
- CLI with API capabilities
- Mesh networking routing rules
- Mesh networking hub-and-spoke VPN feature
- CLI mesh hub support

### Changed
- Updated admin binaries
- Schema improvements
- Web UI improvements

### Fixed
- Mesh client TLS verification
- Mesh routing rule verbosity

## [1.0.1] - 2025-12-30

### Changed
- Updated UI components
- Updated Go to v1.25
- Updated packages: vite, tailwindcss, @vitejs/plugin-react
- Bumped Alpine from 3.19 to 3.23 in Docker images

### Fixed
- Lint issues

### Security
- Updated dependencies for security patches
- Bumped actions/setup-go from 5 to 6
- Bumped actions/setup-node from 4 to 6
- Bumped actions/upload-artifact from 4 to 6
- Bumped github/codeql-action from 3 to 4
- Bumped actions/checkout from 4 to 6

## [1.0.0] - 2025-12-30

### Added
- Initial GA release
- Zero-trust VPN control plane
- OpenVPN gateway management
- User and device authentication
- Web-based administration UI
- CLI client for gateway management
- OIDC/SSO integration support
- PostgreSQL database backend
- Docker and Kubernetes deployment support
- Helm chart for Kubernetes deployments
