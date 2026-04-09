# Attribution

This project was built from scratch by [Logiphys Datensysteme GmbH](https://logiphys.de). The following open-source projects served as references during development for API client patterns, entity definitions, and tool structures. No code was copied verbatim, but architectural patterns and API knowledge were derived from studying these implementations.

## Referenced Projects

### tphakala/autotask-mcp

- **Repository**: https://github.com/tphakala/autotask-mcp
- **License**: Apache License 2.0
- **Author**: tphakala
- **Used as reference for**: Autotask PSA API client patterns, entity type definitions, query filter structure
- **Affected packages**: `pkg/autotask/`

### wyre-technology/rocketcyber-mcp

- **Repository**: https://github.com/wyre-technology/rocketcyber-mcp
- **License**: Apache License 2.0
- **Copyright**: 2026 Wyre Technology
- **Used as reference for**: RocketCyber API endpoint structure and authentication flow
- **Affected packages**: `pkg/rocketcyber/`

## Dependencies

Direct dependencies are listed in `go.mod`. Key libraries:

| Module | License |
|--------|---------|
| [github.com/mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) | MIT |
| [golang.org/x/sync](https://pkg.go.dev/golang.org/x/sync) | BSD-3-Clause |
