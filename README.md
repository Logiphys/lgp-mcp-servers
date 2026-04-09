# LGP MCP Servers

[![CI](https://github.com/Logiphys/lgp-mcp-servers/actions/workflows/ci.yml/badge.svg)](https://github.com/Logiphys/lgp-mcp-servers/actions/workflows/ci.yml)
[![Release](https://github.com/Logiphys/lgp-mcp-servers/actions/workflows/release.yml/badge.svg)](https://github.com/Logiphys/lgp-mcp-servers/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Logiphys/lgp-mcp-servers)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Go monorepo for Logiphys MCP (Model Context Protocol) servers. Provides Claude with structured access to IT service management, security, backup, and documentation platforms used at [Logiphys Datensysteme GmbH](https://logiphys.de).

**258 tools** across **9 MCP servers**, built as single-binary deployments.

## Servers

| Server | Platform | Tools | Description |
|--------|----------|------:|-------------|
| `autotask-mcp` | [Autotask PSA](https://www.datto.com/products/autotask-psa) | 82 | Tickets, companies, contacts, projects, billing, time entries, service calls, quotes |
| `datto-rmm-mcp` | [Datto RMM](https://www.datto.com/products/rmm) | 55 | Remote monitoring & management — devices, sites, alerts, jobs, audit, variables |
| `itglue-mcp` | [IT Glue](https://www.itglue.com) | 31 | IT documentation — organizations, configurations, passwords, contacts, domains, expirations |
| `datto-edr-mcp` | [Datto EDR](https://www.datto.com/products/edr) | 21 | Endpoint detection & response — agents, alerts, rules, extensions, quarantine, isolation |
| `datto-uc-mcp` | [Datto Unified Continuity](https://www.datto.com/products/unified-continuity) | 20 | BCDR appliances, SaaS Protection, Direct-to-Cloud backup, activity logs |
| `rocketcyber-mcp` | [RocketCyber](https://www.rocketcyber.com) | 13 | Managed SOC — agents, events, incidents, firewalls, suppression rules |
| `datto-network-mcp` | [Datto Networking](https://www.datto.com/products/networking) | 13 | Network devices, clients, WAN usage, application visibility |
| `datto-backup-mcp` | [Datto Backup](https://www.datto.com/products/backup) | 12 | Backup appliances, assets, alerts, customers, endpoint & SaaS backup |
| `myitprocess-mcp` | [MyITProcess](https://www.myitprocess.com) | 11 | vCIO — clients, reviews, findings, initiatives, recommendations |

## Quick Start

### Prerequisites

- Go 1.26+
- API credentials for the platforms you want to connect

### Build

```bash
make build                # Build all servers for current platform
make build-all            # Cross-compile for macOS (arm64/amd64) + Windows
make build-autotask-mcp   # Build a single server
```

Binaries are output to `dist/`.

### Install

Copy the built binaries to a location on your PATH:

```bash
cp dist/*-mcp /usr/local/bin/
```

### Configure

Each server is configured via environment variables. Add them to your Claude Code or Claude Desktop configuration.

**Claude Code** (`~/.claude/settings.json`):

```json
{
  "mcpServers": {
    "autotask-mcp": {
      "command": "autotask-mcp",
      "env": {
        "AUTOTASK_USERNAME": "api-user@company.com",
        "AUTOTASK_SECRET": "your-api-secret",
        "AUTOTASK_INTEGRATION_CODE": "your-integration-code"
      }
    }
  }
}
```

See `config/` for full configuration examples with all 9 servers. See [CONTRIBUTING](CONTRIBUTING.md) for development guidelines.

## Access Control

Access control (role-based tool filtering, GDPR/privacy tiers) is handled by the [LGP MCP Gateway](https://github.com/Logiphys/lgp-mcp-gateway). The standalone servers expose all tools — the gateway enforces which tools are available per user based on Entra ID roles.

## Server Details

### autotask-mcp

Connects to the [Autotask PSA REST API](https://autotask.net/help/DeveloperHelp/Content/APIs/REST/REST_API_Home.htm). Covers the full PSA workflow: tickets, companies, contacts, projects, billing items, time entries, quotes, service calls, expenses, and configuration items.

| Environment Variable | Required | Description |
|---------------------|----------|-------------|
| `AUTOTASK_USERNAME` | Yes | API user email |
| `AUTOTASK_SECRET` | Yes | API secret |
| `AUTOTASK_INTEGRATION_CODE` | Yes | Integration tracking code |
| `AUTOTASK_BASE_URL` | No | Override API base URL |

### itglue-mcp

Connects to the [IT Glue API](https://api.itglue.com/developer/). Manages IT documentation: organizations, configurations, passwords, documents, flexible assets, contacts, locations, domains, and expirations.

| Environment Variable | Required | Default | Description |
|---------------------|----------|---------|-------------|
| `ITGLUE_API_KEY` | Yes | | API key (starts with `ITG.`) |
| `ITGLUE_REGION` | No | `us` | API region (`us` or `eu`) |
| `ITGLUE_BASE_URL` | No | | Override API base URL |

### datto-rmm-mcp

Connects to the [Datto RMM API](https://rmm-api-d.datto.com/). Full remote monitoring coverage: devices, sites, alerts, jobs, audit data, variables, components, and user management.

| Environment Variable | Required | Default | Description |
|---------------------|----------|---------|-------------|
| `DATTO_API_KEY` | Yes | | API key |
| `DATTO_API_SECRET` | Yes | | API secret |
| `DATTO_PLATFORM` | No | `merlot` | Platform region (merlot, pinotage, concord, vidal, zinfandel, syrah) |
| `DATTO_BASE_URL` | No | | Override API base URL |

### rocketcyber-mcp

Connects to the [RocketCyber API](https://api-eu.rocketcyber.com/v3/docs). Managed SOC platform: agents, security events, incidents, firewalls, Defender status, and suppression rules.

| Environment Variable | Required | Default | Description |
|---------------------|----------|---------|-------------|
| `ROCKETCYBER_API_KEY` | Yes | | API Bearer token |
| `ROCKETCYBER_REGION` | No | `us` | API region (`us` or `eu`) |
| `ROCKETCYBER_BASE_URL` | No | | Override API base URL |

### datto-uc-mcp

Connects to the [Datto Unified Continuity API](https://api.datto.com/v1/). Covers BCDR appliances, SaaS Protection (M365/Google Workspace), Direct-to-Cloud backup, and activity reporting.

| Environment Variable | Required | Description |
|---------------------|----------|-------------|
| `DATTO_UC_PUBLIC_KEY` | Yes | API public key |
| `DATTO_UC_SECRET_KEY` | Yes | API secret key |
| `DATTO_UC_BASE_URL` | No | Override API base URL |

### datto-edr-mcp

Connects to the [Datto EDR (Infocyte) LoopBack API](https://docs.infocyte.com). Endpoint detection & response: agents, alerts, detection rules, suppression rules, extensions, quarantined files, and host isolation.

| Environment Variable | Required | Description |
|---------------------|----------|-------------|
| `DATTO_EDR_API_KEY` | Yes | API access token |
| `DATTO_EDR_BASE_URL` | Yes | Instance URL (e.g. `https://yourorg.infocyte.com`) |

### datto-backup-mcp

Connects to the [Datto Backup API](https://backup-api.datto.com/). Manages backup appliances, protected assets, alerts, customers, endpoint backup, and SaaS backup domains.

| Environment Variable | Required | Description |
|---------------------|----------|-------------|
| `DATTO_BACKUP_CLIENT_ID` | Yes | OAuth2 client ID |
| `DATTO_BACKUP_CLIENT_SECRET` | Yes | OAuth2 client secret |
| `DATTO_BACKUP_BASE_URL` | No | Override API base URL |

### datto-network-mcp

Connects to the [Datto Networking (DNA) API](https://api.dna.datto.com). Network device management: access points, switches, routers, client overview, WAN usage, and application visibility.

| Environment Variable | Required | Description |
|---------------------|----------|-------------|
| `DATTO_NETWORK_PUBLIC_KEY` | Yes | API public key |
| `DATTO_NETWORK_SECRET_KEY` | Yes | API secret key |
| `DATTO_NETWORK_BASE_URL` | No | Override API base URL |

### myitprocess-mcp

Connects to the [MyITProcess API](https://api.myitprocess.com). Virtual CIO platform: clients, reviews, meetings, findings, initiatives, and recommendations.

| Environment Variable | Required | Description |
|---------------------|----------|-------------|
| `MYITPROCESS_API_KEY` | Yes | API key |

## Architecture

```
pkg/resilience/        Rate limiter, circuit breaker, response compactor
pkg/apihelper/         HTTP client, JSON:API parser, OAuth2, pagination
pkg/mcputil/           MCP response helpers, annotations, formatters
pkg/config/            Environment variable loading

pkg/autotask/          Autotask PSA logic (tickets, billing, projects)
pkg/itglue/            IT Glue logic (documentation, passwords, configs)
pkg/dattormm/          Datto RMM logic (devices, sites, alerts, jobs)
pkg/rocketcyber/       RocketCyber logic (SOC events, incidents, agents)
pkg/dattouc/           Datto Unified Continuity logic (BCDR, SaaS, DTC)
pkg/dattoedr/          Datto EDR logic (detection, response, quarantine)
pkg/dattobackup/       Datto Backup logic (appliances, assets, alerts)
pkg/dattonetwork/      Datto Networking logic (devices, clients, WAN)
pkg/myitprocess/       MyITProcess logic (reviews, findings, initiatives)

cmd/autotask-mcp/      Binary entry points (one per server)
cmd/itglue-mcp/
cmd/datto-rmm-mcp/
cmd/rocketcyber-mcp/
cmd/datto-uc-mcp/
cmd/datto-edr-mcp/
cmd/datto-backup-mcp/
cmd/datto-network-mcp/
cmd/myitprocess-mcp/
```

All servers share a resilience middleware stack (rate limiting, circuit breaker, response compaction) and common HTTP client utilities. Each server is a standalone binary communicating over stdio using the [MCP protocol](https://modelcontextprotocol.io).

## Development

```bash
make test          # Run all tests with race detection
make lint          # Run golangci-lint
make test-cover    # Generate coverage report
```

Requires Go 1.26+ and [golangci-lint](https://golangci-lint.run/).

## License

MIT — see [LICENSE](LICENSE).
