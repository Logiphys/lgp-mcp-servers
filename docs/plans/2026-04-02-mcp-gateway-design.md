# LGP MCP Gateway — Design

**Date:** 2026-04-02
**Status:** Approved
**Repo:** github.com/Logiphys/lgp-mcp-gateway (separate repo)

## Problem

MCP servers currently run as local stdio binaries. External platforms (Langdock) and multi-user scenarios require HTTP-based access with centralized authentication. API keys are currently per-user — they should be centrally managed.

## Solution

A standalone MCP Gateway that:
- Authenticates users via Microsoft Entra ID (OAuth2/JWT)
- Maps Entra groups to MCP server access + tier levels
- Stores all API keys centrally in Azure Key Vault
- Proxies MCP requests to backend servers
- Runs on Azure Container Apps

## Architecture

```
Client (Langdock, Claude, Agent)
  |  HTTPS + OAuth2 Bearer Token
  v
+--------------------------------------+
|         LGP MCP Gateway              |
|         (Azure Container Apps)       |
|                                      |
|  1. Validate JWT (Entra ID)          |
|  2. Group claims -> role             |
|  3. Role -> allowed servers + tiers  |
|  4. Route MCP request to backend     |
|  5. Write audit log                  |
+--------------------------------------+
  |  stdio (backends as subprocesses)
  v
+------------------------------------------+
|  MCP Server Backends (all at Tier 3)     |
|  autotask-mcp  itglue-mcp  datto-rmm-mcp|
|  hornetsecurity-mcp  ingram-mcp  ...     |
+------------------------------------------+
  |
  v
External APIs (Autotask, IT Glue, Datto, ...)
```

## Components

### 1. Auth Layer — Entra ID OAuth2/JWT
- Client authenticates via MSAL, sends Bearer token
- Gateway validates JWT against Entra ID JWKS endpoint
- Extracts `groups` claim for role resolution

### 2. Role Resolver — YAML Config
- Maps Entra group IDs to server access + tier
- Multiple groups: highest tier per server wins
- No entry = no access

### 3. Router — MCP Proxy
- Receives MCP requests via Streamable HTTP
- Filters tool list based on role/tier
- Blocks disallowed tool calls before reaching backend
- Forwards allowed calls to correct backend

### 4. Secret Store — Azure Key Vault
- All API keys/secrets stored centrally
- Gateway injects credentials into backend connections
- Users never see/know the keys

### 5. Audit Log
- Who called which tool when
- Azure Log Analytics

## Roles

```yaml
roles:
  MCP-Technik:
    autotask: 3
    itglue: 3
    datto-rmm: 3
    datto-edr: 3
    datto-uc: 2
    rocketcyber: 2
    datto-network: 2
    datto-backup: 2
    myitprocess: 2

  MCP-Vertrieb:
    autotask: 3
    itglue: 1
    datto-rmm: 1
    myitprocess: 2

  MCP-Beratung:
    autotask: 1
    itglue: 1
    myitprocess: 2

  MCP-Automation:
    autotask: 3
    itglue: 2
    datto-rmm: 3
    datto-edr: 2
    datto-uc: 2
    datto-backup: 2
    rocketcyber: 2
    datto-network: 2
    myitprocess: 2

  MCP-GL:
    autotask: 2
    itglue: 2
    datto-rmm: 2
    datto-edr: 2
    datto-uc: 2
    rocketcyber: 2
    datto-network: 2
    datto-backup: 2
    myitprocess: 2
```

## Tiering Interaction

- **Standalone mode** (local Claude Code): MCP servers use `*_ACCESS_TIER` env var
- **Behind gateway**: MCP servers run at Tier 3 (all tools), gateway filters per role
- No conflict, no double-tiering

## Tech Stack

| Component | Technology |
|---|---|
| Gateway | Go (separate repo `lgp-mcp-gateway`) |
| Client transport | Streamable HTTP (MCP spec 2025) |
| Backend transport | stdio (subprocesses) |
| Auth | Entra ID OAuth2 / JWT validation |
| Secrets | Azure Key Vault |
| Hosting | Azure Container Apps |
| Config | YAML roles file |
| Container | Docker (multi-stage build) |
| Audit | Azure Log Analytics |

## Extensibility

Adding new MCP servers (Hornetsecurity, Ingram, Pax8):
1. Add binary to container image
2. Register backend in gateway config
3. Add to roles YAML
4. No gateway code change needed
