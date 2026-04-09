# Changelog

All notable changes to this project will be documented in this file.

## [v1.2.1] - 2026-04-09

### Fixed

- Upgrade golangci-lint action v6 → v7 (Go 1.26 compatibility)
- Add all 9 servers to CI build matrix (was only 4)
- Resolve all errcheck lint violations across production and test code
- Remove unused `addSortingRule` function

## [v1.2.0] - 2026-04-09

### Changed

- **Remove access tier logic from standalone servers** — Access control (GDPR tiers, role-based tool filtering) is now handled entirely by the [MCP Gateway](https://github.com/Logiphys/lgp-mcp-gateway). All `*_ACCESS_TIER` environment variables and `config.AccessTier()` have been removed.
- Simplified `RegisterTools()` signatures across all 9 servers (no longer accept `tier` parameter)
- Removed `AccessTier` from `ServerInfo`
- Removed `autotask_` prefix from internal tool name mappings

## [v1.1.0] - 2026-04-02

### Changed

- Move backend packages from `internal/` to `pkg/` for external import by the gateway

## [v1.0.0] - 2026-04-02

### Changed

- Rename module `github.com/Logiphys/lgp-mcp` → `github.com/Logiphys/lgp-mcp-servers`

## [v0.8.0] - 2026-04-02

### Added

- Comprehensive README with all 9 servers and environment variable documentation
- CHANGELOG.md
- Updated config examples for Claude Code and Claude Desktop with all 9 servers
- Updated API quirks documentation covering all 9 servers

### Fixed

- Remove non-existent EDR endpoints (Scans, Jobs, ActivityTraces, ScanHosts, RunExtension)
- Fix IT Glue configuration_interfaces to use nested API path

## [v0.7.2] - 2026-04-02

### Added
- **29 new tools** across 7 servers from full API audit
- Datto BCDR renamed to **Datto Unified Continuity** (`datto-uc-mcp`) reflecting broader API scope
- 5 Direct-to-Cloud (DTC) tools in datto-uc-mcp: list_dtc_assets, get_dtc_asset, list_dtc_rmm_templates, get_dtc_storage_pool, list_dtc_client_assets
- 2 new Datto EDR tools: get_alert_count, get_agent_count
- 10 new IT Glue tools: search_contacts, get_contact, search_locations, get_location, list_configuration_types, list_configuration_statuses, list_password_categories, search_domains, list_expirations, search_configuration_interfaces
- 2 new Datto RMM tools: get_dnet_site_mappings, get_site_network_interfaces
- 2 new Datto Networking tools: get_user_devices, get_reseller_overview
- Enhanced parameters across RocketCyber (connectivity, os, sort, verdict), Datto Backup (order_by, order_direction, version, helix_status), and other servers

### Fixed
- EDR `get_dashboard` now handles array response correctly (was failing with "cannot unmarshal array into map")
- EDR Rules, SuppressionRules, and Extensions use JSON string filter format instead of bracket notation
- Datto UC pagination: `list_agents` and `get_activity_log` use `_perPage` (underscore prefix) as required by API

### Removed
- Old `datto-bcdr-mcp` server (replaced by `datto-uc-mcp`)

## [v0.7.1] - 2026-04-02

### Added
- 5 new MCP servers: Datto EDR, Datto BCDR, Datto Backup, Datto Networking, MyITProcess

## [v0.7.0] - 2026-04-02

### Fixed
- Autotask filter wrapping for nested field queries
- Datto RMM OAuth2 authentication flow
- IT Glue double-wrapped filter parameters
- Server info tool registration

## [v0.6.0] - 2026-04-01

### Added
- CI/CD pipeline, install scripts, configuration infrastructure
- IT Glue, Datto RMM, and RocketCyber MCP servers
- 81 Autotask MCP tools with full PSA coverage

### Infrastructure
- Makefile with cross-compilation support
- Shared resilience middleware (rate limiter, circuit breaker, compactor)
- Common HTTP client, JSON:API parser, pagination utilities
