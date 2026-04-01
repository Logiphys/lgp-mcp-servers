# LGP MCP — Go Monorepo

## Project Context

Go monorepo consolidating all Logiphys MCP servers (Autotask, IT Glue, Datto RMM, RocketCyber) into single-binary deployments.

- **Design Document**: `docs/design.md` — full architecture, API details, implementation phases
- **MCP Library**: `github.com/mark3labs/mcp-go`
- **Go Version**: 1.23+

## Repository Layout

- `pkg/` — shared libraries (resilience, apihelper, mcputil, config) — importable by all servers
- `internal/` — server-specific logic (autotask, itglue, dattormm, rocketcyber) — not importable externally
- `cmd/` — binary entry points (one `main.go` per server)
- `config/` — example configuration files
- `scripts/` — build and deployment scripts

## Conventions

- Go 1.23, type hints everywhere (no `any` unless deserializing JSON)
- `slog` for structured logging (JSON to stderr)
- Table-driven tests with `-race` flag
- `golangci-lint` for linting
- Error wrapping with `fmt.Errorf("context: %w", err)`
- Context propagation on all I/O operations
- No global state — pass dependencies via constructors

## Build

```bash
make build       # all servers, current platform
make build-all   # all servers, all platforms (darwin/arm64, darwin/amd64, windows/amd64)
make build-NAME  # single server (e.g., make build-autotask-mcp)
make test        # all tests with -race
make lint        # golangci-lint
```

## TypeScript Sources (Migration Reference)

- Autotask: `/Users/zeisler/lgp-autotask-mcp/src/`
- IT Glue: `/Users/zeisler/lgp-itglue-mcp/src/`
- Datto RMM: `/Users/zeisler/datto-rmm/`
- RocketCyber: `github.com/wyre-technology/rocketcyber-mcp`
- Autotask Go Reference: `github.com/tphakala/autotask-mcp`

## Important

- pip install always with --break-system-packages (if Python needed for tooling)
- No interactive editors (nano etc.)
- Absolute paths
- Ask before destructive actions (delete, overwrite)
