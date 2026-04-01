# LGP MCP Servers

Go monorepo for Logiphys MCP (Model Context Protocol) servers. Provides Claude with access to IT service management platforms used at Logiphys Datensysteme GmbH.

## Servers

| Server | Platform | Tools | Description |
|--------|----------|-------|-------------|
| `autotask-mcp` | Autotask PSA | 92 | Tickets, companies, contacts, projects, billing, service calls |
| `itglue-mcp` | IT Glue | 20 | Documentation, organizations, passwords, flexible assets |
| `datto-rmm-mcp` | Datto RMM | 64 | Remote monitoring, devices, sites, alerts, jobs |
| `rocketcyber-mcp` | RocketCyber | 10 | SOC/security monitoring, incidents, agents, firewalls |

## Quick Start

### Build

```bash
make build          # Build all servers for current platform
make build-all      # Cross-compile for macOS + Windows
make build-autotask-mcp  # Build single server
```

### Configure

Add to `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "autotask-mcp": {
      "command": "/usr/local/bin/lgp-autotask-mcp",
      "env": {
        "AUTOTASK_USERNAME": "your-username",
        "AUTOTASK_SECRET": "your-secret",
        "AUTOTASK_INTEGRATION_CODE": "your-code"
      }
    }
  }
}
```

See `config/` for full configuration examples.

## Architecture

```
pkg/resilience/   — Rate limiter, circuit breaker, response compactor
pkg/apihelper/    — HTTP client, JSON:API parser, OAuth2, pagination
pkg/mcputil/      — MCP response helpers, annotations, formatters
pkg/config/       — Environment variable loading

internal/autotask/     — Autotask PSA server logic
internal/itglue/       — IT Glue server logic
internal/dattormm/     — Datto RMM server logic
internal/rocketcyber/  — RocketCyber server logic

cmd/autotask-mcp/      — Binary entry point
cmd/itglue-mcp/        — Binary entry point
cmd/datto-rmm-mcp/     — Binary entry point
cmd/rocketcyber-mcp/   — Binary entry point
```

All servers share a resilience middleware (rate limiting, circuit breaker, response compaction) and common HTTP client utilities.

## Development

```bash
make test        # Run all tests with race detection
make lint        # Run golangci-lint
make test-cover  # Generate coverage report
```

Requires Go 1.23+ and [golangci-lint](https://golangci-lint.run/).

## License

MIT — see [LICENSE](LICENSE).
