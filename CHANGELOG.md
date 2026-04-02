# Changelog

All notable changes to this project will be documented in this file.

## [v0.8.0] - 2026-04-02

### Breaking Changes

- **Tiered access control** — All servers now default to Tier 1 (safe read-only). Tools accessing sensitive data (passwords, contacts, audit logs) require `*_ACCESS_TIER=2`. Write operations (create, update, delete) require `*_ACCESS_TIER=3`. See [Access Tiers](#access-tiers-gdprprivacy) in README.

### Added

- `config.AccessTier()` helper for reading tier environment variables
- `server_info` now shows `access_tier` and `tier_description`
- Per-server `*_ACCESS_TIER` environment variables (9 servers)

### Changed

- `RegisterTools()` signature on all 9 servers now accepts `tier int` parameter
- IT Glue document tools split into read/write registration functions

## [Unreleased]

### Added
- Remove non-existent EDR endpoints (Scans, Jobs, ActivityTraces, ScanHosts, RunExtension)
- Fix IT Glue configuration_interfaces to use nested API path
- Comprehensive README with all 9 servers and environment variable documentation
- CHANGELOG.md
- Updated config examples for Claude Code and Claude Desktop with all 9 servers
- Updated API quirks documentation covering all 9 servers

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
