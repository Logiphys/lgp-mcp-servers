# Tiered Access System — Design

**Date:** 2026-04-02
**Status:** Approved

## Problem

MCP servers expose tools that read/write sensitive data (passwords, personal contacts, device credentials) through AI agents. Under GDPR/DSGVO, organizations need control over what data flows through AI models. Currently all tools are always registered — no way to restrict access.

## Solution

Three-tier access system per server, controlled via environment variables. Default is Tier 1 (safe read-only).

## Tiers

| Tier | Name | Description |
|------|------|-------------|
| 1 | Safe Read-Only | No personal data, no credentials, no write ops |
| 2 | Read + Sensitive | Adds passwords, contacts, user data, audit logs |
| 3 | Full Access | Adds create, update, delete operations |

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Default tier | 1 | Secure by default; breaking change but correct for GDPR |
| Env var naming | Per server (`ITGLUE_ACCESS_TIER`) | Granular control — IT Glue Tier 2 doesn't imply Autotask Tier 2 |
| server_info | Shows active tier | User sees immediately which mode is active |
| Scope | Existing 249 tools only | No new tools in this change |

## Architecture

### Config Helper

New function in `pkg/config/config.go`:

```go
func AccessTier(envKey string) int
```

- Reads env var, parses as int, defaults to 1
- Clamps to range 1–3

### RegisterTools Signature Change

All 9 servers get a `tier int` parameter:

```go
// Before:
func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger)

// After:
func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger, tier int)
```

Autotask (has extra picklist param):
```go
func RegisterTools(srv *server.MCPServer, client *Client, picklist *PicklistCache, logger *slog.Logger, tier int)
```

### Conditional Registration

```go
func RegisterTools(srv *server.MCPServer, c *Client, logger *slog.Logger, tier int) {
    // Tier 1 — always registered
    registerSearchOrganizations(srv, c)

    if tier >= 2 {
        registerSearchContacts(srv, c)
        registerGetPassword(srv, c)
    }

    if tier >= 3 {
        registerCreateOrganization(srv, c)
    }
}
```

### main.go Pattern

```go
tier := config.AccessTier("ITGLUE_ACCESS_TIER")
logger.Info("starting", "tier", tier)
itglue.RegisterTools(srv, client, logger, tier)
```

### server_info Enhancement

```json
{
  "server": "itglue-mcp",
  "version": "0.8.0",
  "access_tier": 2,
  "tier_description": "Read + Sensitive Data",
  "tools_registered": 25
}
```

## Tier Assignment Per Server

### IT Glue (30 tools)

**Tier 1** — Safe Read:
- server_info, health_check
- search_organizations, get_organization
- search_configurations, get_configuration
- search_configuration_interfaces
- list_configuration_types, list_configuration_statuses
- search_documents, get_document
- list_document_sections, get_document_section
- search_flexible_assets, list_flexible_asset_types
- search_locations, get_location
- search_domains, list_expirations
- list_password_categories

**Tier 2** — +Sensitive:
- search_contacts, get_contact
- search_passwords, get_password

**Tier 3** — +Write:
- create_document, update_document, delete_document
- create_document_section, update_document_section, delete_document_section
- publish_document

### Autotask (81 tools)

**Tier 1** — Safe Read:
- server_info, test_connection, router
- search_tickets, get_ticket_details, list_ticket_statuses, list_ticket_priorities, list_queues
- search_companies, get_field_info
- search_projects, search_project_notes, get_project_note, list_phases
- search_quotes, get_quote, search_quote_items, get_quote_item
- search_services, get_service, search_service_bundles, get_service_bundle
- search_service_calls, get_service_call
- search_service_call_tickets, search_service_call_ticket_resources
- search_billing_items, get_billing_item, search_billing_item_approval_levels
- search_invoices
- search_products, get_product
- search_opportunities, get_opportunity
- search_contracts
- search_ticket_attachments, get_ticket_attachment
- search_ticket_notes, get_ticket_note
- search_ticket_charges, get_ticket_charge
- search_company_notes, get_company_note
- list_categories, list_category_tools, execute_tool

**Tier 2** — +Sensitive:
- search_contacts, search_resources
- search_expense_reports, get_expense_report
- search_time_entries
- search_configuration_items

**Tier 3** — +Write:
- create_ticket, update_ticket
- create_ticket_note, create_ticket_charge, update_ticket_charge, delete_ticket_charge
- create_company, update_company, create_company_note
- create_contact
- create_project, create_project_note
- create_phase, create_task
- create_opportunity
- create_quote, create_quote_item, update_quote_item, delete_quote_item
- create_service_call, update_service_call, delete_service_call
- create_service_call_ticket, delete_service_call_ticket
- create_service_call_ticket_resource, delete_service_call_ticket_resource
- create_expense_item, create_expense_report
- create_time_entry

### Datto RMM (54 tools)

**Tier 1** — Safe Read:
- server_info
- list_sites, get_site, get_site_settings, get_site_network_interfaces
- list_devices, get_device, get_device_by_id, get_device_by_mac
- list_open_alerts, list_resolved_alerts, get_alert
- list_site_devices, list_site_open_alerts, list_site_resolved_alerts
- list_device_open_alerts, list_device_resolved_alerts
- list_default_filters, list_custom_filters, list_site_filters
- list_components
- get_account, get_system_status
- get_pagination_config, get_rate_limit
- get_dnet_site_mappings
- get_esxi_audit, get_printer_audit

**Tier 2** — +Sensitive:
- get_device_audit, get_device_audit_by_mac
- get_device_software
- get_activity_logs
- list_users
- list_account_variables, list_site_variables
- get_job, get_job_components, get_job_results, get_job_stdout, get_job_stderr

**Tier 3** — +Write:
- create_site, update_site
- create_quick_job
- move_device
- set_device_udf, set_device_warranty
- resolve_alert
- create_account_variable, update_account_variable, delete_account_variable
- create_site_variable, update_site_variable, delete_site_variable
- update_site_proxy, delete_site_proxy

### Datto EDR (20 tools)

**Tier 1** — Safe Read:
- test_connection
- get_dashboard
- list_agents, get_agent, get_agent_count
- list_alerts, get_alert, get_alert_count
- list_organizations, list_locations
- list_device_groups
- list_policies
- list_rules, list_suppression_rules
- list_extensions

**Tier 2** — +Sensitive:
- list_alerts_archive
- list_quarantined_files

**Tier 3** — +Write/Action:
- scan_agent
- isolate_host
- restore_host

### Datto Unified Continuity (19 tools)

**Tier 1** — Safe Read:
- test_connection
- list_devices, get_device
- list_device_agents, list_device_assets
- list_device_shares, list_device_vm_restores
- list_device_alerts
- get_device_volume_assets
- list_agents
- list_dtc_assets, list_dtc_client_assets, get_dtc_asset
- get_dtc_storage_pool
- list_dtc_rmm_templates
- list_saas_domains

**Tier 2** — +Sensitive:
- get_activity_log
- get_saas_applications, get_saas_seats

**Tier 3:** (no write operations in this API)

### RocketCyber (12 tools)

**Tier 1** — Safe Read:
- test_connection
- list_agents
- list_events, get_event_summary
- list_incidents
- list_apps
- list_firewalls
- list_suppression_rules, get_suppression_rule

**Tier 2** — +Sensitive:
- get_account
- get_defender
- get_office

**Tier 3:** (no write operations in this API)

### Datto Networking (12 tools)

**Tier 1** — Safe Read:
- test_connection
- list_devices, get_device
- get_devices_overview, get_reseller_overview
- get_router
- get_whoami, get_user_devices

**Tier 2** — +Sensitive:
- get_device_clients_overview, get_device_clients_usage
- get_device_wan_usage
- get_device_applications

**Tier 3:** (no write operations in this API)

### Datto Backup (11 tools)

**Tier 1** — Safe Read:
- test_connection
- list_appliances
- list_assets
- list_backups
- list_alerts
- get_agent_version
- list_spanning_domains
- list_entra_domains

**Tier 2** — +Sensitive:
- list_customers
- list_endpoint_assets
- list_spanning_domain_users

**Tier 3:** (no write operations in this API)

### MyITProcess (10 tools)

**Tier 1** — Safe Read:
- test_connection
- list_reviews, list_overdue_reviews
- list_findings
- list_recommendations, get_recommendation_configurations
- list_initiatives

**Tier 2** — +Sensitive:
- list_clients
- list_users
- list_meetings

**Tier 3:** (no write operations in this API)

## Environment Variables

| Server | Env Var |
|--------|---------|
| IT Glue | `ITGLUE_ACCESS_TIER` |
| Autotask | `AUTOTASK_ACCESS_TIER` |
| Datto RMM | `DATTO_RMM_ACCESS_TIER` |
| Datto EDR | `DATTO_EDR_ACCESS_TIER` |
| Datto UC | `DATTO_UC_ACCESS_TIER` |
| RocketCyber | `ROCKETCYBER_ACCESS_TIER` |
| Datto Networking | `DATTO_NETWORK_ACCESS_TIER` |
| Datto Backup | `DATTO_BACKUP_ACCESS_TIER` |
| MyITProcess | `MYITPROCESS_ACCESS_TIER` |

## Breaking Change

This is a breaking change. Users who rely on contacts, passwords, or write operations must add `*_ACCESS_TIER=2` or `*_ACCESS_TIER=3` to their environment config.

Migration: Add the appropriate `ACCESS_TIER` env var to each server's config in Claude Code settings or Claude Desktop config.
