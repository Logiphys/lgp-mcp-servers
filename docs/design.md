# LGP MCP Go-Monorepo — Design Document

> Consolidation of all Logiphys MCP servers into a single Go monorepo.
> Single-binary deployment, cross-compile for macOS (arm64/amd64) + Windows (amd64),
> shared resilience library, unified architecture.

## Table of Contents

1. [Repository Structure](#1-repository-structure)
2. [Shared Libraries (`pkg/`)](#2-shared-libraries)
3. [Server-Specific Architecture (`internal/`)](#3-server-specific-architecture)
4. [Build, Release & Deployment](#4-build-release--deployment)
5. [Implementation Phases](#5-implementation-phases)

---

## 1. Repository Structure

```
lgp-mcp/                              # https://github.com/Logiphys/lgp-mcp
├── go.mod
├── go.sum
├── Makefile
├── CLAUDE.md
├── README.md
├── LICENSE                            # MIT
├── .gitignore
│
├── pkg/                               # Shared libraries (all servers)
│   ├── resilience/
│   │   ├── ratelimiter.go             # Token Bucket (port from rateLimiter.ts)
│   │   ├── ratelimiter_test.go
│   │   ├── circuitbreaker.go          # Circuit Breaker (port from circuitBreaker.ts)
│   │   ├── circuitbreaker_test.go
│   │   ├── compactor.go               # Response Compaction (port from responseCompactor.ts)
│   │   ├── compactor_test.go
│   │   ├── middleware.go              # Combines all three as middleware
│   │   └── middleware_test.go
│   │
│   ├── mcputil/
│   │   ├── result.go                  # TextResult, ErrorResult, JSONResult helpers
│   │   ├── annotations.go            # ReadOnly, Destructive, Idempotent, OpenWorld
│   │   ├── formatter.go              # Entity-aware response formatter
│   │   ├── formatter_test.go
│   │   ├── htmlstrip.go              # HTML-to-plaintext converter
│   │   ├── htmlstrip_test.go
│   │   └── errors.go                 # Standard MCP error messages
│   │
│   ├── apihelper/
│   │   ├── client.go                  # HTTP client with retry, timeout, user-agent
│   │   ├── client_test.go
│   │   ├── jsonapi.go                 # JSON:API parser (for IT Glue)
│   │   ├── jsonapi_test.go
│   │   ├── oauth.go                   # OAuth2 token manager (for Datto RMM)
│   │   ├── oauth_test.go
│   │   ├── mapping.go                 # Generic ID-to-name cache with TTL
│   │   ├── mapping_test.go
│   │   ├── pagination.go             # Cursor + offset pagination iterator
│   │   └── pagination_test.go
│   │
│   └── config/
│       ├── config.go                  # MustEnv, OptEnv, log-level parsing
│       └── config_test.go
│
├── cmd/                               # Binary entry points
│   ├── autotask-mcp/
│   │   └── main.go
│   ├── itglue-mcp/
│   │   └── main.go
│   ├── datto-rmm-mcp/
│   │   └── main.go
│   └── rocketcyber-mcp/
│       └── main.go
│
├── internal/                          # Server-specific logic
│   ├── autotask/
│   │   ├── client.go                  # Autotask REST API client
│   │   ├── client_test.go
│   │   ├── entities.go                # Typed structs for all entity types
│   │   ├── picklist.go                # Picklist cache (queues, statuses, priorities)
│   │   ├── tools.go                   # 92 MCP tool definitions
│   │   ├── tools_test.go
│   │   └── elicitation.go            # Intent routing + elicitation support
│   ├── itglue/
│   │   ├── client.go                  # IT Glue API client (JSON:API)
│   │   ├── client_test.go
│   │   ├── entities.go
│   │   ├── tools.go                   # 20 MCP tool definitions
│   │   └── tools_test.go
│   ├── dattormm/
│   │   ├── client.go                  # Datto RMM API client (OAuth2)
│   │   ├── client_test.go
│   │   ├── entities.go
│   │   ├── tools.go                   # 64 MCP tool definitions
│   │   └── tools_test.go
│   └── rocketcyber/
│       ├── client.go                  # RocketCyber API client
│       ├── client_test.go
│       ├── entities.go
│       ├── tools.go                   # 10 MCP tool definitions
│       └── tools_test.go
│
├── config/
│   ├── claude-code-settings.json.example
│   └── claude-desktop-config.json.example
│
├── scripts/
│   ├── install.sh                     # Cross-platform installer
│   └── release.sh                     # GitHub Release helper
│
├── docs/
│   ├── design.md                      # This document
│   └── api-quirks.md                 # Known API issues and workarounds
│
├── dist/                              # Build output (gitignored)
│
└── .github/
    └── workflows/
        ├── ci.yml
        └── release.yml
```

### Server Overview

| Server | Tools | Auth | API Format | Key Dependency |
|--------|-------|------|-----------|----------------|
| autotask-mcp | 92 | Custom Headers (User/Secret/IntCode) | REST JSON | pkg/resilience |
| itglue-mcp | 20 | API Key (`x-api-key`) | JSON:API | pkg/apihelper/jsonapi |
| datto-rmm-mcp | 64 | OAuth2 (Key/Secret -> Bearer) | REST JSON | pkg/apihelper/oauth |
| rocketcyber-mcp | 10 | API Key | REST JSON | (simplest server) |
| **Total** | **186** | | | |

---

## 2. Shared Libraries

### 2.1 `pkg/resilience/` — Resilience Middleware

Ported from `lgp-autotask-mcp/src/utils/`. Generic, usable by all 4 servers.

#### Rate Limiter (`ratelimiter.go`)

Token Bucket algorithm. Source: `rateLimiter.ts`.

```go
type RateLimiter struct {
    mu            sync.Mutex
    tokens        float64
    maxTokens     float64
    refillRate    float64 // tokens per millisecond
    lastRefill    time.Time
}

func NewRateLimiter(tokensPerHour int) *RateLimiter

func (r *RateLimiter) Allow(n int) bool
func (r *RateLimiter) WaitTime() time.Duration
func (r *RateLimiter) Available() float64
func (r *RateLimiter) Reset()
```

- Default: 5000 tokens/hour (Autotask limit), configurable per server
- `sync.Mutex` for thread-safety
- Non-blocking: `Allow()` returns false immediately if insufficient tokens

#### Circuit Breaker (`circuitbreaker.go`)

State machine. Source: `circuitBreaker.ts`.

```go
type CircuitState int
const (
    StateClosed   CircuitState = iota  // normal operation
    StateOpen                          // rejecting requests
    StateHalfOpen                      // testing recovery
)

type CircuitBreaker struct {
    mu               sync.RWMutex
    state            CircuitState
    failures         int
    successes        int
    lastFailure      time.Time
    failureThreshold int           // default: 5
    cooldown         time.Duration // default: 30s
    successThreshold int           // default: 3
}

func NewCircuitBreaker(opts ...Option) *CircuitBreaker

func (cb *CircuitBreaker) CanExecute() bool
func (cb *CircuitBreaker) RecordSuccess()
func (cb *CircuitBreaker) RecordFailure()
func (cb *CircuitBreaker) State() CircuitState
func (cb *CircuitBreaker) Reset()
```

- State transitions: `CLOSED --(5 failures)--> OPEN --(30s cooldown)--> HALF_OPEN --(3 successes)--> CLOSED`
- `HALF_OPEN --(1 failure)--> OPEN`
- `sync.RWMutex` for concurrent state reads

#### Response Compactor (`compactor.go`)

Recursive null/empty removal. Source: `responseCompactor.ts`.

```go
func Compact(data any) any
func EstimateSavings(original, compacted any) float64
```

- Removes: `nil`, empty slices, empty maps
- Preserves: `0`, `false`, `""` (meaningful values)
- Operates on `map[string]any` — no reflect needed (API responses are JSON-unmarshalled)

#### Middleware (`middleware.go`)

Combines all three patterns.

```go
type Config struct {
    RateLimit        int           // tokens per hour (0 = disabled)
    FailureThreshold int           // circuit breaker failures (0 = disabled)
    Cooldown         time.Duration // circuit breaker cooldown
    SuccessThreshold int           // half-open successes to close
    Compact          bool          // enable response compaction
}

type Middleware struct { ... }

func New(cfg Config) *Middleware

func (m *Middleware) Execute(ctx context.Context, fn func() (any, error)) (any, error)
func (m *Middleware) IsCircuitOpen() bool
func (m *Middleware) RateLimiterStatus() (available float64, waitTime time.Duration)
func (m *Middleware) CircuitBreakerStatus() (state CircuitState, failures int)
```

Execution order:
1. Circuit Breaker check (fast-fail if OPEN)
2. Rate Limiter check (return wait time if exhausted)
3. Execute function
4. Record success/failure for Circuit Breaker
5. Compact response if enabled

Usage example:
```go
mw := resilience.New(resilience.Config{
    RateLimit:        5000,
    FailureThreshold: 5,
    Cooldown:         30 * time.Second,
    Compact:          true,
})
result, err := mw.Execute(ctx, func() (any, error) {
    return a.client.Get("/Tickets", params)
})
```

### 2.2 `pkg/mcputil/` — MCP Helpers

Thin wrappers around `mcp-go`, reducing boilerplate.

#### `result.go` — Response Builders

```go
func TextResult(text string) *mcp.CallToolResult
func ErrorResult(err error) *mcp.CallToolResult
func JSONResult(data any) *mcp.CallToolResult
```

#### `annotations.go` — Tool Annotations

```go
func ReadOnly() mcp.ToolAnnotation
func Destructive() mcp.ToolAnnotation
func Idempotent() mcp.ToolAnnotation
func OpenWorld() mcp.ToolAnnotation
```

Applied during tool registration:
- All List/Get tools: `ReadOnly()` + `Idempotent()`
- All Delete tools: `Destructive()`
- All List tools: `OpenWorld()`

#### `formatter.go` — Entity-Aware Response Formatter

```go
type FieldSet map[string][]string // entity type -> essential fields

func FormatCompact(entityType string, data []map[string]any, fields FieldSet) string
func FormatFull(data map[string]any) string
func WithPagination(text string, current, total, count int) string
func WithNames(data map[string]any, names map[string]string) map[string]any
```

Per-entity essential fields:
- **Tickets:** id, ticketNumber, title, status, priority, companyID, assignedResourceID, createDate, dueDateTime
- **Companies:** id, companyName, isActive, phone, city, state
- **Projects:** id, projectName, status, companyID, projectLeadResourceID, startDate, endDate
- **Time Entries:** id, resourceID, ticketID, dateWorked, hoursWorked, summaryNotes
- **Billing Items:** id, itemName, companyID, ticketID, postedDate, totalAmount, invoiceID

Inlines resolved names (companyName, resourceName) when mapping cache is available.

#### `htmlstrip.go` — HTML to Plaintext

```go
func StripHTML(html string) string
func StripHTMLWithLimit(html string, maxChars int) string
```

- Converts `<br>`, `<p>`, `<li>` to newlines
- Strips all remaining HTML tags
- Default truncation: 25,000 characters
- Used by: IT Glue Document Sections, Autotask Ticket Descriptions

#### `errors.go` — Standard Error Messages

```go
var (
    ErrNotFound      = errors.New("resource not found")
    ErrRateLimited   = errors.New("rate limit exceeded, retry after wait period")
    ErrCircuitOpen   = errors.New("service temporarily unavailable (circuit breaker open)")
    ErrValidation    = errors.New("input validation failed")
    ErrUnauthorized  = errors.New("authentication failed — check credentials")
)
```

### 2.3 `pkg/apihelper/` — HTTP Client Utilities

#### `client.go` — Shared HTTP Client

```go
type ClientConfig struct {
    BaseURL    string
    Timeout    time.Duration     // default: 30s
    MaxRetries int               // default: 3
    UserAgent  string            // default: "lgp-mcp/<version>"
    Headers    map[string]string // static headers (auth, content-type)
}

type Client struct { ... }

func NewClient(cfg ClientConfig) *Client

func (c *Client) Get(ctx context.Context, path string, params url.Values) ([]byte, error)
func (c *Client) Post(ctx context.Context, path string, body any) ([]byte, error)
func (c *Client) Patch(ctx context.Context, path string, body any) ([]byte, error)
func (c *Client) Delete(ctx context.Context, path string) error
```

- Exponential backoff with jitter on retries
- Retries only on 429 (rate limit) and 5xx (server error)
- Context cancellation respected
- Response body always closed

#### `jsonapi.go` — JSON:API Parser (IT Glue)

```go
type JSONAPIResponse[T any] struct {
    Data     []T            `json:"data"`
    Meta     PaginationMeta `json:"meta"`
    Included []any          `json:"included,omitempty"`
}

type JSONAPIResource struct {
    ID         string         `json:"id"`
    Type       string         `json:"type"`
    Attributes map[string]any `json:"attributes"`
}

func ParseResponse[T any](body []byte) (*JSONAPIResponse[T], error)
func BuildFilterParams(filters map[string]string) url.Values
```

#### `oauth.go` — OAuth2 Token Manager (Datto RMM)

```go
type OAuth2Config struct {
    TokenURL     string
    ClientID     string // API Key
    ClientSecret string // API Secret
}

type TokenManager struct { ... }

func NewTokenManager(cfg OAuth2Config) *TokenManager

func (t *TokenManager) Token(ctx context.Context) (string, error)
```

- Token cached with automatic refresh (5min buffer before expiry)
- Deduplicates concurrent refresh requests via `singleflight.Group`
- Basic Auth for token endpoint

#### `mapping.go` — Generic ID-to-Name Cache

```go
type MappingCache[K comparable, V any] struct { ... }

func NewMappingCache[K comparable, V any](ttl time.Duration) *MappingCache[K, V]

func (m *MappingCache[K, V]) Get(ctx context.Context, key K, fetch func(K) (V, error)) (V, error)
func (m *MappingCache[K, V]) Warm(ctx context.Context, fetchAll func() (map[K]V, error)) error
func (m *MappingCache[K, V]) Clear()
func (m *MappingCache[K, V]) Stats() (size int, hitRate float64)
```

- TTL-based expiration (default: 30min)
- Single-flight for concurrent requests to same key
- Bulk preload via `Warm()` at startup
- Used by: Autotask (companies, resources, queues), IT Glue (organizations), Datto (sites)

#### `pagination.go` — Pagination Iterator

```go
type PageFetcher[T any] func(ctx context.Context, page int) (items []T, hasMore bool, err error)

func Paginate[T any](ctx context.Context, fetch PageFetcher[T]) iter.Seq[T]
```

- Supports cursor-based (Autotask: NextPageURL) and offset-based (IT Glue: page[number])
- Go 1.23 iterator pattern (`iter.Seq`)
- Stops on context cancellation

### 2.4 `pkg/config/` — Configuration

```go
func MustEnv(name string) string         // panics if empty
func OptEnv(name, fallback string) string // returns fallback if empty
func LogLevel() slog.Level               // parses LOG_LEVEL env var
```

---

## 3. Server-Specific Architecture

### 3.1 Common Entry Point Pattern

Every `cmd/<server>/main.go` follows the same structure:

```go
func main() {
    cfg := config.MustLoad()
    logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

    server := mcp.NewServer(serverInfo)
    client := autotask.NewClient(cfg, logger)
    autotask.RegisterTools(server, client)

    transport := stdio.NewTransport()
    if err := server.Run(ctx, transport); err != nil {
        logger.Error("server failed", "error", err)
        os.Exit(1)
    }
}
```

Signal handling (SIGINT/SIGTERM) for graceful shutdown in all 4 binaries.

### 3.2 `internal/autotask/` — Autotask PSA (92 Tools)

**References:**
- `tphakala/autotask-mcp` (Go) — API client pattern, entity definitions
- Previous TypeScript implementation — LGP extensions, tool definitions

#### `client.go` — Autotask REST API Client
- Base URL: `https://webservices24.autotask.net/ATServicesRest`
- Auth: Custom Headers (`UserName`, `Secret`, `IntegrationCode`, `ApiIntegrationCode`)
- Generic CRUD: `Get(entity, id)`, `List(entity, filter)`, `Create(entity, body)`, `Update(entity, body)`, `Delete(entity, id)`
- Autotask query-filter builder: `{"op": "eq", "field": "status", "value": 1}`
- All calls go through `pkg/resilience` middleware

#### `entities.go` — Entity Definitions
- Typed structs: Company, Contact, Ticket, Quote, QuoteItem, Resource, TimeEntry, Project, Task, Phase, Note, ServiceCall, BillingItem, ExpenseReport, ConfigurationItem, Contract, Invoice, TicketCharge, Attachment
- All support `map[string]any` for dynamic/unknown fields

#### `picklist.go` — Picklist Cache
- Lazy-load field metadata per entity type via `getFieldInfo(entityType)`
- Cached: Queues, Statuses, Priorities, Ticket-Types, Issue-Types
- Session lifetime (no TTL — picklists rarely change)
- Used by formatter for ID-to-label resolution

#### `tools.go` — 92 MCP Tools in 12 Categories

| Category | Tools | Count |
|----------|-------|-------|
| Utility | test_connection, list_queues, list_ticket_statuses, list_ticket_priorities, get_field_info | 5 |
| Companies | search, create, update | 3 |
| Contacts | search, create | 2 |
| Tickets | search, get_details, create, update, get/search_notes, create_note, get/search_attachments, get/search/create/update/delete_charges | 14 |
| Projects | search, create, search/create_tasks, list/create_phases, get/search/create_project_notes | 9 |
| Time & Billing | create/search_time_entries, search/get_billing_items, search_approval_levels, get/search/create_expense_reports, create_expense_item | 9 |
| Financial | get/search/create_quotes, get/search/create/update/delete_quote_items, get/search/create_opportunities, search_invoices, search_contracts | 13 |
| Products & Services | get/search_products, get/search_services, get/search_service_bundles | 6 |
| Resources | search_resources | 1 |
| Configuration Items | search_configuration_items | 1 |
| Company Notes | get/search/create_company_notes | 3 |
| Service Calls | search/get/create/update/delete_service_calls, search/create/delete_service_call_tickets, search/create/delete_service_call_ticket_resources | 11 |
| Discovery | execute_tool (lazy-loading meta-tool) | 1 |

**Known API Quirks (hardcoded workarounds):**
- `search_resources` → HTTP 500 persistent → fallback to `list` with client-side filter
- `search_invoices` with companyID → inconsistent results → use `search_billing_items` instead
- Pagination: use `postedAfter` date filter for large result sets

#### `elicitation.go` — Intent Routing
- `RouteIntent(query string) (toolName string, params map[string]any)` — natural language to tool mapping
- Elicitation support (MCP spec): date-range picker, company selector, item selector
- Fallback: tool list with categories if no intent matched

### 3.3 `internal/itglue/` — IT Glue (20 Tools)

**References:**
- Previous TypeScript implementation — existing tools
- `Junto-Platforms/itglue-mcp-server` — response optimizations

#### `client.go` — IT Glue API Client
- Uses `pkg/apihelper/jsonapi.go` for JSON:API format
- Auth: `x-api-key` header
- Region-based base URLs: US (`api.itglue.com`), EU (`api.eu.itglue.com`), AU (`api.au.itglue.com`)
- Content-Type: `application/vnd.api+json`

#### `tools.go` — 20 Tools

| Category | Tools | Count |
|----------|-------|-------|
| Organizations | search, get, create, update | 4 |
| Configurations | list, get | 2 |
| Passwords | list, get | 2 |
| Flexible Assets | list_types, search, create, update | 4 |
| Documents | list, get, create, update, publish, delete | 6 |
| Document Sections | list, get, create, update, delete | 5 |
| Utility | health_check | 1 |

**Features from Junto integration:**
- HTML-to-plaintext for Document Sections (via `pkg/mcputil/htmlstrip.go`)
- Dual API-call pattern for list_documents (root + folder + deduplication)
- Content truncation (25k characters)
- MCP annotations (destructive hints on delete tools)
- 4 section types: Text, Heading, Gallery, Step (with type-specific parameters)

### 3.4 `internal/dattormm/` — Datto RMM (64 Tools)

**References:**
- Previous TypeScript implementation — existing code
- OpenAPI spec in upstream repo

#### `client.go` — Datto RMM API Client
- OAuth2 via `pkg/apihelper/oauth.go` (Basic Auth → Bearer Token)
- 6 platform regions: Pinotage, Merlot, Concord, Vidal, Zinfandel, Syrah
- Optional: struct generation from OpenAPI spec via `oapi-codegen`

#### `tools.go` — 64 Tools in 10 Categories

| Category | Tools | Count |
|----------|-------|-------|
| Account | get, list_users, get/list/create/update/delete variables | 8 |
| Sites | list, get, create, update, get_settings, update/delete_proxy, list/create/update/delete variables | 8 |
| Devices | list, get_by_id, get_by_mac, get_audit, get_audit_by_mac, get_software, get_esxi, get_printer, move, set_udf, set_warranty | 9 |
| Alerts | list_open, list_resolved, list_device_open, list_device_resolved, list_site_open, list_site_resolved, get, resolve | 2 |
| Jobs | get, get_components, get_results, get_stdout, get_stderr, create_quick_job | 5 |
| Audit | get_device, get_device_by_mac, get_esxi, get_printer, get_software | 5 |
| Activity | get_activity_logs | 1 |
| Filters | list_default, list_custom, list_site | 2 |
| System | get_rate_limit, get_pagination_config, get_system_status | 3 |
| Components | list_components | 1 |

**Resources:** `datto://account`, `datto://sites`, `datto://alerts/open`

### 3.5 `internal/rocketcyber/` — RocketCyber SOC (10 Tools)

**Reference:** `wyre-technology/rocketcyber-mcp`

#### `client.go` — RocketCyber API Client
- Simplest client: API Key in header, direct REST calls
- No Node SDK wrapper needed — HTTP endpoints directly
- Region support (default: `us`)

#### `tools.go` — 10 Tools

| Tool | Description | Parameters |
|------|-------------|-----------|
| test_connection | Test API connectivity | — |
| get_account | Account information | accountId (opt) |
| list_agents | Monitored endpoints | accountId, status, hostname, platform, page, pageSize, date range |
| list_incidents | Security incidents | status, severity, title, page, pageSize, date range |
| list_events | Security events | eventType, severity, hostname, page, pageSize, date range |
| get_event_summary | Event statistics | accountId, date range |
| list_firewalls | Firewall devices | connectivity, hostname, vendor, page, pageSize |
| list_apps | Managed apps | status, name |
| get_defender | Windows Defender status | accountId |
| get_office | Office 365 status | accountId |

**Resources:** `rocketcyber://account`, `rocketcyber://incidents`, `rocketcyber://agents`

---

## 4. Build, Release & Deployment

### 4.1 Go Module

```
module github.com/Logiphys/lgp-mcp

go 1.23

require (
    github.com/mark3labs/mcp-go v0.x.x    // MCP protocol
    golang.org/x/sync v0.x.x               // errgroup, singleflight
)
```

Minimal dependencies — Go stdlib for HTTP, JSON, logging (`slog`), testing.

### 4.2 Makefile

```makefile
MODULE      := github.com/Logiphys/lgp-mcp
VERSION     := $(shell git describe --tags --always --dirty)
LDFLAGS     := -s -w -X main.version=$(VERSION)
SERVERS     := autotask-mcp itglue-mcp datto-rmm-mcp rocketcyber-mcp
PLATFORMS   := darwin/arm64 darwin/amd64 windows/amd64

.PHONY: build
build:
	@for s in $(SERVERS); do \
		go build -ldflags "$(LDFLAGS)" -o dist/$$s ./cmd/$$s; \
	done

.PHONY: build-all
build-all:
	@for s in $(SERVERS); do \
		for p in $(PLATFORMS); do \
			GOOS=$${p%%/*} GOARCH=$${p##*/} \
			go build -ldflags "$(LDFLAGS)" \
				-o dist/$$s-$${p%%/*}-$${p##*/}$$([ $${p%%/*} = windows ] && echo .exe) \
				./cmd/$$s; \
		done \
	done

.PHONY: build-%
build-%:
	go build -ldflags "$(LDFLAGS)" -o dist/$* ./cmd/$*

.PHONY: test
test:
	go test ./... -v -race -count=1

.PHONY: test-cover
test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: clean
clean:
	rm -rf dist/
```

### 4.3 GitHub Actions CI

```yaml
# .github/workflows/ci.yml
name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: go test ./... -race -count=1
      - run: golangci-lint run ./...

  build:
    needs: test
    strategy:
      matrix:
        goos: [darwin, windows]
        goarch: [amd64, arm64]
        exclude:
          - goos: windows
            goarch: arm64
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: |
          for server in autotask-mcp itglue-mcp datto-rmm-mcp rocketcyber-mcp; do
            GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} \
            go build -ldflags "-s -w" -o dist/${server}-${{ matrix.goos }}-${{ matrix.goarch }} \
            ./cmd/${server}
          done
      - uses: actions/upload-artifact@v4
        with:
          name: binaries-${{ matrix.goos }}-${{ matrix.goarch }}
          path: dist/
```

### 4.4 GitHub Actions Release

```yaml
# .github/workflows/release.yml
name: Release
on:
  push:
    tags: ['v*']

jobs:
  release:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: make build-all
      - uses: softprops/action-gh-release@v2
        with:
          files: dist/*
          generate_release_notes: true
```

### 4.5 Deployment Strategies

#### Option 1: GitHub Releases (Primary)
- Push tag → CI builds binaries → GitHub Release with assets
- Technicians download matching binary manually or via script
- Config template (`settings.json.example`) included in release

#### Option 2: Datto RMM Component (Automatic)
- PowerShell component that:
  1. Detects OS + Arch
  2. Downloads matching binary from GitHub Release
  3. Copies to `C:\Program Files\Logiphys\` (Win) or `/usr/local/bin/` (macOS)
  4. Updates Claude Desktop / Claude Code settings
- Deployed via Datto RMM to all technician machines
- Update mechanism: component checks current version vs. GitHub latest release

#### Option 3: Homebrew Tap (macOS, optional)
```ruby
# logiphys/homebrew-tap/Formula/lgp-mcp.rb
class LgpMcp < Formula
  desc "Logiphys MCP Servers for Claude"
  homepage "https://github.com/Logiphys/lgp-mcp"
  # platform-specific URLs for binaries
end
```

#### Option 4: Installer Script (Cross-Platform)
```bash
# curl -fsSL https://raw.githubusercontent.com/Logiphys/lgp-mcp/main/scripts/install.sh | bash
# Detects OS/Arch, downloads binaries, creates config template
```

**Recommended rollout order:**
1. GitHub Releases — immediate, MVP deployment
2. Datto RMM Component — automatic rollout to technicians
3. Homebrew Tap + Installer Script — nice-to-have

### 4.6 Target MCP Configuration

After migration, `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "autotask-mcp": {
      "command": "/usr/local/bin/lgp-autotask-mcp",
      "env": {
        "AUTOTASK_USERNAME": "...",
        "AUTOTASK_SECRET": "...",
        "AUTOTASK_INTEGRATION_CODE": "..."
      }
    },
    "itglue-mcp": {
      "command": "/usr/local/bin/lgp-itglue-mcp",
      "env": {
        "ITGLUE_API_KEY": "...",
        "ITGLUE_REGION": "eu"
      }
    },
    "datto-rmm-mcp": {
      "command": "/usr/local/bin/lgp-datto-rmm-mcp",
      "env": {
        "DATTO_API_URL": "...",
        "DATTO_API_KEY": "...",
        "DATTO_API_SECRET": "...",
        "DATTO_PLATFORM": "merlot"
      }
    },
    "rocketcyber-mcp": {
      "command": "/usr/local/bin/lgp-rocketcyber-mcp",
      "env": {
        "ROCKETCYBER_API_KEY": "..."
      }
    }
  }
}
```

On Windows: `C:\Program Files\Logiphys\lgp-autotask-mcp.exe` etc.

---

## 5. Implementation Phases

### Phase Overview

```
Phase 1: Foundation          ████░░░░░░░░░░░░░░░░  pkg/* basis
Phase 2: Autotask (92 Tools) ████████░░░░░░░░░░░░  largest server, sets patterns
Phase 3: IT Glue (20 Tools)  ██░░░░░░░░░░░░░░░░░░  JSON:API, HTML strip
Phase 4: Datto RMM (64 Tools)████████░░░░░░░░░░░░  OAuth2, OpenAPI spec
Phase 5: RocketCyber (10)    █░░░░░░░░░░░░░░░░░░░  simplest server
Phase 6: Deployment & Rollout██░░░░░░░░░░░░░░░░░░  CI/CD, Datto Component
```

### Dependencies

```
Phase 1 ──→ Phase 2 ──→ Phase 3 ──→ Phase 6
                    ├──→ Phase 4 ──→ Phase 6
                    └──→ Phase 5 ──→ Phase 6
```

- Phase 1 is prerequisite for everything
- Phase 2 (Autotask) first — largest server, defines patterns for all others
- Phases 3, 4, 5 can run **in parallel** after Phase 2 (independent servers)
- Phase 6 starts as soon as at least one server is done (incremental rollout)

### Phase 1: Foundation (`pkg/*`)

**Goal:** All shared libraries built and tested, ready for server implementations.

**1.1 — Repo Bootstrap**
- `go mod init github.com/Logiphys/lgp-mcp`
- Directory structure, Makefile, CI workflow
- `CLAUDE.md` with Go conventions
- `.gitignore`

**1.2 — `pkg/resilience/`**
- Rate Limiter, Circuit Breaker, Compactor, Middleware
- Full test coverage, race condition tests (`-race`)

**1.3 — `pkg/apihelper/`**
- HTTP Client, Pagination, Mapping Cache
- JSON:API Parser, OAuth2 Token Manager, HTML Strip

**1.4 — `pkg/mcputil/`**
- Result builders, annotations, formatter, errors

**1.5 — `pkg/config/`**
- Env-var loading with validation

**Validation:** `make test` all green, `make lint` clean.

### Phase 2: Autotask MCP (92 Tools)

**Goal:** Full replacement for `lgp-autotask-mcp` (TypeScript).

**2.1 — API Client** (`internal/autotask/client.go`)
**2.2 — Entities & Picklist** (`internal/autotask/entities.go`, `picklist.go`)
**2.3 — Tools** (5 batches):
- Batch 1 (Core, 25): Companies, Contacts, Tickets, Resources, ConfigItems
- Batch 2 (Financial, 13): Quotes, QuoteItems, Opportunities, Invoices, Contracts
- Batch 3 (Operations, 18): Projects, Tasks, Phases, TimeEntries, Expenses
- Batch 4 (Dispatch, 11): Service Calls + Ticket-Linking + Resource-Assignment
- Batch 5 (Notes & Utils, 11): Company/Project/Ticket Notes, Attachments, Field Info, Discovery
**2.4 — Elicitation** (`internal/autotask/elicitation.go`)
**2.5 — Entry Point** (`cmd/autotask-mcp/main.go`)

**Validation:**
- Unit tests for client, entities, each tool category
- Integration test against live Autotask API (manual, not in CI)
- Binary test as MCP server in Claude Code
- Side-by-side comparison: same queries to TypeScript and Go versions

### Phase 3: IT Glue MCP (20 Tools)

**Goal:** Port of `lgp-itglue-mcp` + Junto patterns.

**3.1 — API Client** (`internal/itglue/client.go`) — JSON:API, region support
**3.2 — Entities** (`internal/itglue/entities.go`)
**3.3 — Tools** (`internal/itglue/tools.go`) — 20 tools with annotations
**3.4 — Entry Point** (`cmd/itglue-mcp/main.go`)

**Validation:** Unit tests, live API test, binary test in Claude Code.

### Phase 4: Datto RMM MCP (64 Tools)

**Goal:** Port of `lgp-datto-rmm-mcp`.

**4.1 — API Client** (`internal/dattormm/client.go`) — OAuth2, region support
**4.2 — Entities** (`internal/dattormm/entities.go`) — from OpenAPI spec
**4.3 — Tools** (`internal/dattormm/tools.go`) — 64 tools + resources
**4.4 — Entry Point** (`cmd/datto-rmm-mcp/main.go`)

**Validation:** Unit tests, live API test, binary test in Claude Code.

### Phase 5: RocketCyber MCP (10 Tools)

**Goal:** Port of `wyre-technology/rocketcyber-mcp`. Smallest server.

**5.1 — API Client** (`internal/rocketcyber/client.go`) — API key, direct REST
**5.2 — Entities** (`internal/rocketcyber/entities.go`)
**5.3 — Tools** (`internal/rocketcyber/tools.go`) — 10 tools + 3 resources
**5.4 — Entry Point** (`cmd/rocketcyber-mcp/main.go`)

**Validation:** Unit tests, live API test, binary test in Claude Code.

### Phase 6: Deployment & Rollout

**6.1 — Release Pipeline**
- `.github/workflows/release.yml` — tag → build → GitHub Release
- Semantic versioning: `v1.0.0` for first stable release

**6.2 — Datto RMM Component**
- PowerShell script for automatic deployment
- OS/Arch detection → download → install → config update

**6.3 — Config Templates**
- `config/claude-code-settings.json.example`
- `config/claude-desktop-config.json.example`
- Install script that fills templates with correct paths

**6.4 — Cutover**
- Switch one server at a time (Go binary replaces Node/npx)
- 1 week parallel operation with logging comparison
- Archive TypeScript repos after successful migration

---

## Appendix A: Gateway Mode (Optional)

For cloud deployment scenarios (e.g., central instance for multiple technicians):

```go
// pkg/mcpserver/gateway.go
// POST /mcp   — MCP request with credentials in headers
// GET  /health — health check endpoint

// Headers:
// X-API-Key           → username / api key
// X-API-Secret        → secret / api secret
// X-Integration-Code  → integration code (Autotask only)
// X-API-URL           → custom API URL (optional)
```

Fresh server instance per request with injected credentials.
Not required for MVP — stdio transport is the default.
