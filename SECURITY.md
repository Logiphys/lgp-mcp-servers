# Security Policy

## Reporting Vulnerabilities

Please do **not** open public issues for security vulnerabilities.

Email **security@logiphys.de** with:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if available)

We will acknowledge your report within 48 hours and work on a fix.

## Supported Versions

| Version | Status |
|---------|--------|
| 1.2.x | Supported |
| < 1.2 | Not supported |

## Scope

This project provides MCP server binaries that connect to third-party APIs (Autotask, Datto, IT Glue, etc.). Security issues in those upstream APIs should be reported to the respective vendors.

## Dependencies

We monitor dependencies for known vulnerabilities. Run `go get -u ./...` to pull the latest patches.
