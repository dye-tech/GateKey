# Changelog

All notable changes to GateKey are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.5.0] - 2026-01-22

### Added
- Standalone web UI Docker image (`gatekey-web`) with nginx for flexible deployment architectures
- Mesh hub Docker image (`gatekey-hub`) for containerized hub deployments
- GitHub Actions build caching for faster CI/CD pipelines

### Changed
- **Breaking**: Renamed config file from `gatex.yaml` to `gatekey.yaml` across all deployment methods
- **Breaking**: Changed environment variable prefix from `GATEX_` to `GATEKEY_` for consistency
- CI/CD pipeline now builds all 4 Docker images in parallel using matrix strategy (~4x faster)
- Server Docker image (`gatekey-server`) is now API-only; web UI served separately via `gatekey-web`
- Config loader now auto-discovers `gatekey.yaml` in `/app/configs`, `./configs`, or current directory
- Updated Helm chart templates to use `gatekey.yaml` config filename
- Updated Kustomize base with proper config file mounting and secret structure

### Fixed
- Docker build times reduced from ~18 minutes to ~4 minutes with parallelization
- Config loading now properly uses `GATEKEY_` prefixed environment variables
- Kubernetes deployments now correctly mount config files to `/app/configs`

### Removed
- Frontend build stage from main server Dockerfile (moved to dedicated web image)
- Legacy `gatex` naming convention throughout codebase

### Migration Guide
If upgrading from v1.4.x:
1. Rename `gatex.yaml` to `gatekey.yaml` in your ConfigMaps
2. Update any `GATEX_*` environment variables to `GATEKEY_*`
3. If using embedded web UI, switch to the separate `gatekey-web` container

## [1.4.5] - 2026-01-11

### Added
- Per-app TLS verification skip option for proxy applications
- Database support for storing per-app TLS verification settings
- Admin settings for TLS verification configuration

### Changed
- SSRF protection disabled by default for better compatibility
- Added mesh network health check endpoint

### Security
- Enhanced TLS verification controls for enterprise environments

## [1.4.4] - 2026-01-11

### Added
- Database schema dump and setup documentation

### Security
- Security feature improvements and fixes

## [1.4.3] - 2026-01-11

### Fixed
- Resolved golangci-lint errors across codebase

### Documentation
- Added detailed changelog for v1.4.2 release

## [1.4.2] - 2026-01-11

### Added
- IPv6 support for geo-fencing rules with country-level IP ranges
- IPv4/IPv6 tab selector for country-based geo-fence rule creation
- CLI now displays mesh network hubs alongside traditional gateways
- Comprehensive unit tests for geo-fencing functionality

### Changed
- Geo-fence rule dropdown now shows CIDR count badges instead of full lists for better readability
- Gateway admin modals now support scrolling for smaller viewports
- Database schema updated to make IPv4 ranges optional when IPv6-only rules are needed

### Fixed
- CLI mesh hub visibility - `gatekey list` now shows both gateways and mesh networks
- CLI connection to mesh hubs was failing with 403 CSRF error
- Web UI geo-fence API calls were failing with 403 CSRF errors
- Gateway admin "Add Gate" modal was cut off on smaller screens

### Security
- CSRF middleware now correctly exempts all Bearer token authentication (JWT and API keys)
- Added CSRF token handling to web frontend for state-changing API requests

## [1.4.1] - 2026-01-10

### Added
- Unit tests for core functionality
- Database TLS documentation

### Changed
- UI improvements and fixes
- Database migration updates

### Security
- Enforce TLS 1.2 minimum for all connections
- Security hardening across API endpoints
- Fixed route parameter conflict (`:userId` to `:id`)

## [1.4.0] - 2026-01-09

### Added
- IPv6 support for VPN tunnels and gateways
- Dual-stack networking capability

## [1.3.2] - 2026-01-07

### Changed
- Database schema updates for new features

## [1.3.1] - 2026-01-07

### Added
- SAML authentication compatibility
- Comprehensive CHANGELOG.md
- Latest Android APK distribution via web UI

### Changed
- WireGuard mesh compatibility improvements
- Updated CSS styling
- Updated README with enhanced intro and WireGuard documentation
- Updated golangci-lint to v2 config format

### Security
- Bumped golangci/golangci-lint-action from 6 to 9
- Bumped codecov/codecov-action from 4 to 5

### Fixed
- golangci-lint configuration errors

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
