# Contributing to LGP MCP Servers

Thank you for your interest in contributing! This project provides MCP servers for IT service management platforms commonly used by MSPs (Managed Service Providers).

## Getting Started

1. **Fork** the repository
2. **Clone** your fork: `git clone https://github.com/your-username/lgp-mcp-servers.git`
3. **Build** all servers: `make build`
4. **Run tests**: `make test`

## Prerequisites

- Go 1.23+
- [golangci-lint](https://golangci-lint.run/) for linting

## Development Workflow

1. Create a feature branch: `git checkout -b feature/your-feature`
2. Make your changes
3. Run tests: `make test`
4. Run linter: `make lint`
5. Commit with a descriptive message
6. Push and open a Pull Request

## Code Conventions

- **Structured logging** with `slog` (JSON to stderr)
- **Error wrapping** with `fmt.Errorf("context: %w", err)`
- **Context propagation** on all I/O operations
- **No global state** — pass dependencies via constructors
- **Table-driven tests** with the `-race` flag
- **No `any` type** unless deserializing JSON

## Project Structure

- `pkg/` — shared libraries importable by all servers
- `internal/` — server-specific logic (one package per server, not importable externally)
- `cmd/` — binary entry points (one `main.go` per server)

Each server follows the same pattern:
1. `client.go` — API client with authentication and HTTP methods
2. `tools.go` (+ optional `tools_*.go`) — MCP tool registrations
3. `cmd/<server>/main.go` — entry point wiring config, client, and tools

## Adding a New Tool

1. Add the tool definition in the appropriate `internal/<server>/tools*.go` file
2. Register it in the `RegisterTools()` function
3. Follow existing patterns for pagination, error handling, and response formatting
4. Use `mcputil.JSONResult()` for structured responses and `mcputil.ErrorResult()` for errors

## Adding a New Server

1. Create `internal/<servername>/client.go` and `tools.go`
2. Create `cmd/<server-name>-mcp/main.go`
3. Add the server to the `SERVERS` list in `Makefile`
4. Add configuration examples to `config/`
5. Document the server in `README.md`

## Reporting Issues

Please include:
- Which MCP server is affected
- The tool name and parameters used
- The error message or unexpected behavior
- Your Go version (`go version`)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
