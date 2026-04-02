# Phase 7: New MCP Servers — Datto Networking, BCDR, Backup, MyITProcess, EDR

**Date:** 2026-04-02
**Status:** Planned

## Overview

Add 5 new MCP servers to the monorepo, following the same patterns as existing servers (RocketCyber, IT Glue, Datto RMM, Autotask).

## Servers

### 1. Datto Networking (DNA) — `datto-network-mcp`
- **Base URL:** `https://api.dna.datto.com/dna-api/v1/`
- **Auth:** API Key (Public Key + Secret Key from Datto Partner Portal)
- **Env vars:** `DATTO_NETWORK_PUBLIC_KEY`, `DATTO_NETWORK_SECRET_KEY`
- **Tools (~10):**
  - `test_connection` / `server_info`
  - `get_whoami` — current user
  - `list_devices` — all device MACs
  - `get_devices_overview` — fleet overview
  - `get_device` — single device by MAC
  - `get_router` — router-specific info
  - `get_device_clients_overview` — client overview (window param)
  - `get_device_clients_usage` — per-client bandwidth
  - `get_device_wan_usage` — WAN bandwidth
  - `get_device_applications` — top apps by bandwidth

### 2. Datto BCDR — `datto-bcdr-mcp`
- **Base URL:** `https://api.datto.com/v1`
- **Auth:** Basic Auth (same keys as networking? or separate Partner Portal keys)
- **Env vars:** `DATTO_BCDR_PUBLIC_KEY`, `DATTO_BCDR_SECRET_KEY`
- **Tools (~10):**
  - `test_connection` / `server_info`
  - `list_devices` — all BCDR devices (SIRIS, ALTO, NAS)
  - `get_device` — single device by serial
  - `list_device_assets` — agents + shares for a device
  - `list_device_agents` — agents only
  - `list_device_shares` — shares only
  - `list_device_alerts` — alerts for a device
  - `list_device_vm_restores` — VM restores
  - `list_agents` — all EB4PC agents
  - `get_activity_log` — activity/audit log

### 3. Datto Backup (Unitrends) — `datto-backup-mcp`
- **Base URL:** `https://public-api.backup.net`
- **Auth:** OAuth2 client_credentials via `https://login.backup.net/connect/token`
- **Env vars:** `DATTO_BACKUP_CLIENT_ID`, `DATTO_BACKUP_CLIENT_SECRET`
- **Client:** Needs OAuth2 token management (fetch + refresh), similar to Datto RMM
- **Tools (~12):**
  - `test_connection` / `server_info`
  - `list_customers` — tenants
  - `list_appliances` — backup appliances
  - `list_assets` — protected assets
  - `list_backups` — backup jobs
  - `list_alerts` — BackupIQ alerts (type required)
  - `get_agent_version` — latest agent version
  - `list_endpoint_assets` — endpoint backup assets
  - `list_spanning_domains` — M365/Google Spanning domains
  - `list_spanning_domain_users` — users in a domain
  - `list_entra_domains` — Entra ID domains

### 4. MyITProcess — `myitprocess-mcp`
- **Base URL:** `https://reporting.live.myitprocess.com/public-api/v1`
- **Auth:** Header `mitp-api-key: <key>`
- **Rate limit:** 50 req/min
- **Env vars:** `MYITPROCESS_API_KEY`
- **Tools (~10):**
  - `test_connection` / `server_info`
  - `list_clients` — all clients
  - `list_users` — all users
  - `list_reviews` — all reviews
  - `list_overdue_reviews` — overdue review categories
  - `list_findings` — assessment findings
  - `list_recommendations` — recommendations
  - `get_recommendation_configurations` — configs for a recommendation
  - `list_initiatives` — strategic initiatives
  - `list_meetings` — meetings

### 5. Datto EDR (Infocyte) — `datto-edr-mcp`
- **Base URL:** `https://yourorg.infocyte.com/api`
- **Auth:** Bearer token
- **Env vars:** `DATTO_EDR_API_KEY`, `DATTO_EDR_BASE_URL` (instance-specific)
- **Filtering:** LoopBack where-filter syntax
- **Tools (~20, curated from 103 models):**
  - `test_connection` / `server_info`
  - **Agents:** `list_agents`, `get_agent`
  - **Alerts:** `list_alerts`, `get_alert`, `list_alerts_archive`
  - **Scans:** `list_scan_history`, `trigger_scan`
  - **Response:** `isolate_host`, `restore_host`
  - **Organizations:** `list_organizations`
  - **Locations:** `list_locations`
  - **Device Groups:** `list_device_groups`
  - **Policies:** `list_policies`
  - **Rules:** `list_rules`, `list_suppression_rules`
  - **Extensions:** `list_extensions`
  - **Quarantine:** `list_quarantined_files`
  - **Dashboard:** `get_dashboard`

## Implementation Pattern (per server)

Each server follows the established pattern:

```
internal/<name>/
  client.go      — API client (auth, HTTP, pagination)
  tools.go       — MCP tool registrations
cmd/<name>-mcp/
  main.go        — entry point with config, logger, server
```

Shared libraries from `pkg/` (apihelper, resilience, mcputil, config) are reused.

## Implementation Order

1. **MyITProcess** — simplest (API-key header, 9 read-only endpoints, well-documented)
2. **Datto Networking** — simple auth, small API
3. **Datto BCDR** — Basic Auth, small API
4. **Datto Backup (Unitrends)** — OAuth2 token flow needed
5. **Datto EDR** — largest, LoopBack filter syntax

## Build Integration

- Add to `Makefile` targets
- Add to `cmd/` entry points
- Config entries for both `~/.claude/settings.json` and Claude Desktop
