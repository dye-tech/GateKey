# Changelog

All notable changes to GateKey are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.8.0] - 2026-03-16

### Added
- **Just-In-Time (JIT) Access**: Time-limited, approval-based access request system
  - Users request temporary access to resources (access rules, proxy apps, mesh hubs)
  - Admin approval queue with approve/deny and optional notes
  - Automatic revocation on expiry via 60-second background job
  - JIT grants transparently enforced by existing gateway firewall
  - Self-service portal with resource browser, countdown timers
  - Configurable policies: per-resource auto-approve, max duration limits, request expiry
  - Analytics dashboard: approval rates, average duration, requests by type, top requesters
  - Slack/Teams webhook notifications on request, approval, and denial
- **Session Recording**: Record and replay remote shell sessions for compliance
  - Asciicast v2 format with gzip compression for web replay
  - Recording hooks in WebSocket session manager (start on connect, tee output, stop on disconnect)
  - Admin UI with recordings list, metadata, stream endpoint for replay
  - Configurable retention with automatic cleanup (6-hour cycle)
- **Proxy Access Logging**: Enhanced proxy app request logging
  - Optional request/response header capture (sanitized — Cookie, Authorization redacted)
  - Per-app logging toggle (`access_logging_enabled`) and header capture toggle (`log_headers`)
  - Cross-app admin log viewer with filtering by app, user, method, status range
- **VPN Network Flow Logging**: Per-connection network flow data from gateway agents
  - Tracks destination IPs, ports, protocols, bytes sent/received, duration
  - Gateway agent flow report endpoint (`POST /api/v1/gateway/flow-report`)
  - Admin dashboard: flow list with filters, top destinations, per-user activity, aggregate stats
- **Privacy Controls**: Sensitive data masking for session recordings
  - Regex-based output masking (passwords, API keys, Bearer tokens)
  - Configurable mask patterns via admin settings
  - Per-session-type recording toggles (terminal, proxy, flow)
  - Default patterns applied automatically on startup
- **SSH Bastion Proxy**: SSH jump host with session recording
  - SSH server on configurable port (default 2222)
  - API key authentication for SSH access
  - Session mode: terminate SSH, proxy to target with recording
  - Jump host mode (`ssh -J`): direct-tcpip forwarding with flow logging
  - Access rule enforcement against target host/port
  - ED25519 host key generation and persistence
  - Configurable target host key validation via environment variable

### Database
- Migration 049: `jit_access_requests`, `jit_access_grants`
- Migration 050: `jit_access_policies`
- Migration 051: `session_recordings`
- Migration 052: Enhanced `proxy_access_logs` with headers, per-app logging toggles
- Migration 053: `network_flow_logs`

## [1.7.0] - 2026-01-27

### Added
- **Prometheus Metrics**: Comprehensive metrics collection for monitoring and alerting
  - HTTP metrics: `gatekey_http_requests_total`, `gatekey_http_request_duration_seconds`, `gatekey_http_requests_in_flight`
  - Authentication metrics: `gatekey_auth_sessions_active`, `gatekey_auth_logins_total`
  - PKI metrics: `gatekey_certificates_issued_total`, `gatekey_certificates_revoked_total`, `gatekey_ca_expiry_seconds`
  - Gateway metrics: `gatekey_gateway_connections_active`, `gatekey_gateway_heartbeats_total`
  - Database metrics: `gatekey_db_connections_open`, `gatekey_db_connections_in_use`, `gatekey_db_connections_idle`
  - Policy metrics: `gatekey_policy_evaluations_total`, `gatekey_policy_evaluation_duration_seconds`
  - Server info: `gatekey_server_info` with version labels
- **OpenTelemetry Tracing**: Distributed tracing support for end-to-end request visibility
  - Automatic span creation for all HTTP requests
  - W3C Trace Context propagation
  - OTLP export via gRPC or HTTP protocols
  - Configurable sampling rate for production environments
  - Rich span attributes including method, path, status, user agent, and client IP
- **Custom Error Types**: Structured error responses for consistent API error handling
  - `NotFoundError`, `ValidationError`, `AuthenticationError`, `AuthorizationError`
  - Consistent JSON error format across all endpoints
  - Improved error messages for debugging

### Changed
- Gin middleware now automatically records HTTP metrics for all requests
- Server gracefully shuts down OpenTelemetry provider on termination

### Fixed
- Removed dead code handlers and unused stub functions
- Fixed potential security issues identified in code review

### Testing
- Added comprehensive test coverage for security-critical paths
- Added unit tests for metrics package and middleware
- Added unit tests for telemetry package and middleware

## [1.6.1] - 2026-01-27

### Fixed
- Unchecked `rand.Read()` error during admin password generation (`internal/db/users.go`)
- Goroutine leak in rate limiter - cleanup routine now properly stops on server shutdown
- Request handlers using `context.Background()` instead of request context for OIDC and async operations

### Changed
- Updated Alpine base images to 3.23 for all Docker images
- Updated Node.js to 25-alpine for web build image
- Removed unused scaffolding code and AI-generated placeholder implementations

### Documentation
- Added complete configuration examples for all 9 binaries
- Renamed config files for clarity: `gatex.yaml` → `server.yaml`, `gateway.yaml` → `openvpn-gateway.yaml`, `hub.yaml` → `openvpn-hub.yaml`
- Fixed config key discrepancies (`stats_report_interval` → `stats_sync_interval` in wireguard-gateway.yaml)

### Dependencies
- Bump axios from 1.13.2 to 1.13.3
- Bump react-router-dom from 7.12.0 to 7.13.0
- Bump @types/react from 19.2.8 to 19.2.9

## [1.6.0] - 2026-01-26

### Added
- **FIPS Mode Enforcement**: New "Enforce FIPS Mode" option for gateways, mesh hubs, and mesh spokes
  - Enables FIPS 140-3 cryptographic compliance at the OS level
  - Configurable per-gateway and per-spoke for granular control
  - Requires FIPS-enabled operating system on the gateway/spoke host
- Database migration (000048) for `enforce_fips_mode` column across gateway tables
- Ingress configuration documentation covering Istio, NGINX, Traefik, and cloud load balancers

### Changed
- Mesh spokes can now have independent FIPS mode settings instead of only inheriting from the parent hub
- CI/CD pipeline now automatically cascades releases to Helm chart and Homebrew repositories

### Documentation
- Added comprehensive Kubernetes ingress guide (`docs/ingress.md`)
- Covers TLS termination, UDP ingress for VPN traffic, certificate management, and troubleshooting

## [1.5.4] - 2026-01-25

### Fixed
- Gateway client-connect hook timing issue where VPN connections were being skipped because OpenVPN updates the status file on a 10s timer, not immediately on connect
- Added missing database columns for gateway and mesh configurations

### Changed
- Gateway, mesh hub, and mesh spoke names are now immutable after creation to prevent authentication failures when agents reconnect with mismatched credentials

### Security
- Prevents accidental authentication breakage by blocking name changes on deployed agents

## [1.5.3] - 2026-01-25

### Fixed
- OpenVPN mesh and gateway creation issues with certificate generation
- Crypto profile validation during gateway creation

## [1.5.2] - 2026-01-25

### Changed
- Consolidated all Dockerfiles into a single parameterized Dockerfile for easier maintenance
- Removed unused Homebrew formula templates

### Fixed
- Added missing nginx proxy routes for backend API endpoints in web container
- Server Docker image now includes install scripts for gateway provisioning

## [1.5.1] - 2026-01-25

### Fixed
- WireGuard mesh hub configs were incorrectly generated in OpenVPN format
- Frontend now calls the correct API endpoint (`/api/v1/mesh/wireguard/generate-config`) for WireGuard mesh hubs
- Download content-type headers now correctly use `application/x-wireguard-profile` for WireGuard configs
- Mesh config download modal displays correct file extension (`.conf` for WireGuard, `.ovpn` for OpenVPN)
- Manual configuration instructions now show WireGuard-specific setup steps when connecting to WireGuard mesh hubs

## [1.5.0] - 2026-01-22

### Added
- Standalone web UI Docker image (`gatekey-web`) with nginx for flexible deployment architectures
- Mesh hub Docker image (`gatekey-hub`) for containerized OpenVPN hub deployments
- WireGuard gateway Docker image (`gatekey-wireguard-gateway`) for containerized WireGuard gateway deployments
- WireGuard mesh hub Docker image (`gatekey-wireguard-hub`) for containerized WireGuard hub deployments
- GitHub Actions build caching for faster CI/CD pipelines

### Changed
- **Breaking**: Renamed config file from `gatex.yaml` to `gatekey.yaml` across all deployment methods
- **Breaking**: Changed environment variable prefix from `GATEX_` to `GATEKEY_` for consistency
- CI/CD pipeline now builds all 6 Docker images in parallel using matrix strategy
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
