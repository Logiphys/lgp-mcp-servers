# Phase 2: Autotask MCP Server Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a complete Autotask MCP server with all 83 tools (79 direct + 4 meta-tools), porting from the TypeScript implementation at `previous-ts-impl/src/`.

**Architecture:** The server uses `internal/autotask/` for all Autotask-specific logic, built on the `pkg/*` foundation from Phase 1. The API client wraps `pkg/apihelper.Client` with Autotask-specific auth headers, query filter builder, and child entity URL routing. Tools are registered via `mcp-go` with a dispatch-table pattern — each tool maps to a handler function that calls the client and formats the response.

**Tech Stack:** Go 1.23+, `github.com/mark3labs/mcp-go` (MCP protocol), `pkg/resilience` (rate limiting, circuit breaker), `pkg/mcputil` (response formatting), `pkg/apihelper` (HTTP client, mapping cache), `pkg/config` (env vars).

**TypeScript References:**
- Tool definitions: `previous-ts-impl/src/handlers/tool.definitions.ts`
- Tool handlers: `previous-ts-impl/src/handlers/tool.handler.ts`
- API service: `previous-ts-impl/src/services/autotask.service.ts`
- Picklist cache: `previous-ts-impl/src/services/picklist.cache.ts`
- Response formatter: `previous-ts-impl/src/utils/response.formatter.ts`

**Autotask REST API:**
- Base URL: `https://webservices24.autotask.net/ATServicesRest` (configurable via `AUTOTASK_API_URL`)
- Auth headers: `UserName`, `Secret`, `ApiIntegrationcode`
- Content-Type: `application/json`
- Query format: POST to `/{Entity}/query` with `{"filter": [{"op":"eq","field":"id","value":1}]}`
- Child entities: `/{Parent}/{parentId}/{Child}` (Notes, Charges, Phases, TimeEntries)
- Response ID extraction: `.itemId` > `.item.id` > `.id`

---

## Task 1: Autotask API Client

**Files:**
- Create: `internal/autotask/client.go`
- Create: `internal/autotask/client_test.go`

This is the core API client. It wraps `pkg/apihelper.Client` with:
- Autotask custom header authentication
- Query filter builder (eq, contains, gte, lte, or)
- Generic CRUD operations (Get, Query, Create, Update, Delete)
- Child entity URL routing (Notes, Charges, Phases, TimeEntries)
- Resilience middleware integration

**Key design decisions:**
- All API responses are `map[string]any` (JSON-unmarshalled) — no typed structs for API responses, since fields are dynamic
- Query uses POST `/{entity}/query` with filter body
- Create uses POST `/{parent}/{parentId}/{child}` for child entities
- Response ID extraction follows `.itemId` > `.item.id` > `.id` pattern
- The client takes `*resilience.Middleware` for rate limiting and circuit breaking

**Implementation:**

The client should implement these methods:
```go
type Client struct {
    http       *apihelper.Client
    middleware *resilience.Middleware
    logger     *slog.Logger
    companies  *MappingCache  // company ID -> name
    resources  *MappingCache  // resource ID -> name
}

type Config struct {
    Username        string
    Secret          string
    IntegrationCode string
    BaseURL         string // default: https://webservices24.autotask.net/ATServicesRest
}

// Core methods
func NewClient(cfg Config, logger *slog.Logger) *Client
func (c *Client) Get(ctx context.Context, entity string, id int) (map[string]any, error)
func (c *Client) Query(ctx context.Context, entity string, filters []Filter, opts QueryOpts) ([]map[string]any, error)
func (c *Client) Create(ctx context.Context, entity string, data map[string]any) (int, error)
func (c *Client) CreateChild(ctx context.Context, parent string, parentID int, child string, data map[string]any) (int, error)
func (c *Client) Update(ctx context.Context, entity string, data map[string]any) error
func (c *Client) Patch(ctx context.Context, entity string, id int, data map[string]any) error
func (c *Client) Delete(ctx context.Context, entity string, id int) error
func (c *Client) DeleteChild(ctx context.Context, parent string, parentID int, child string, childID int) error
func (c *Client) GetFieldInfo(ctx context.Context, entity string) ([]FieldInfo, error)

// Filter types
type Filter struct {
    Op    string   `json:"op"`
    Field string   `json:"field,omitempty"`
    Value any      `json:"value,omitempty"`
    Items []Filter `json:"items,omitempty"` // for "or" filters
}

type QueryOpts struct {
    Page     int
    PageSize int
    MaxSize  int // cap for this entity type
}

// Response parsing
type queryResponse struct {
    Items      []map[string]any `json:"items"`
    PageDetails *pageDetails    `json:"pageDetails"`
}
type createResponse struct {
    ItemID int `json:"itemId"`
    Item   struct{ ID int `json:"id"` } `json:"item"`
    ID     int `json:"id"`
}
```

**Tests should cover:**
- Custom auth headers sent correctly
- GET `/Entity/id` returns parsed entity
- POST `/Entity/query` with filters returns items
- POST `/Parent/parentId/Child` for child entity creation
- DELETE `/Parent/parentId/Child/childId` for child entity deletion
- Response ID extraction from all three patterns
- Retry on 429/5xx via middleware
- Error handling for 404, 400, 500

**Step 1:** Write test file with httptest mocks for all client methods.
**Step 2:** Run tests — verify they FAIL.
**Step 3:** Write `client.go` implementation.
**Step 4:** Run tests — verify they PASS.
**Step 5:** Commit: `feat: add Autotask REST API client in internal/autotask`

---

## Task 2: Entity Definitions & Picklist Cache

**Files:**
- Create: `internal/autotask/entities.go`
- Create: `internal/autotask/picklist.go`
- Create: `internal/autotask/picklist_test.go`
- Delete: `internal/autotask/.gitkeep`

**entities.go** defines entity type constants and the known entity-to-API-path mapping:

```go
// Entity path mapping — maps logical names to Autotask REST API paths
var EntityPaths = map[string]string{
    "Companies":                    "Companies",
    "Contacts":                     "Contacts",
    "Tickets":                      "Tickets",
    "TicketNotes":                  "Tickets/{parentId}/Notes",
    "TicketAttachments":            "TicketAttachments",
    "TicketCharges":                "TicketCharges",
    "Projects":                     "Projects",
    "ProjectNotes":                 "Projects/{parentId}/Notes",
    "ProjectTasks":                 "Tasks",  // Autotask API uses "Tasks" not "ProjectTasks"
    "Phases":                       "Projects/{parentId}/Phases",
    "Resources":                    "Resources",
    "TimeEntries":                  "TimeEntries",
    "BillingItems":                 "BillingItems",
    "BillingItemApprovalLevels":    "BillingItemApprovalLevels",
    "ConfigurationItems":           "ConfigurationItems",
    "Contracts":                    "Contracts",
    "Invoices":                     "Invoices",
    "Quotes":                       "Quotes",
    "QuoteItems":                   "QuoteItems",
    "Opportunities":                "Opportunities",
    "Products":                     "Products",
    "Services":                     "Services",
    "ServiceBundles":               "ServiceBundles",
    "ExpenseReports":               "ExpenseReports",
    "ExpenseItems":                 "ExpenseItems",
    "ServiceCalls":                 "ServiceCalls",
    "ServiceCallTickets":           "ServiceCallTickets",
    "ServiceCallTicketResources":   "ServiceCallTicketResources",
    "CompanyNotes":                 "Companies/{parentId}/Notes",
}

// Entity type aliases (for get_field_info normalization)
var EntityAliases = map[string]string{
    "tasks":           "ProjectTasks",
    "phases":          "Phases",
    "ticket_notes":    "TicketNotes",
    "project_notes":   "ProjectNotes",
    "company_notes":   "CompanyNotes",
    // ... etc
}

// Compact response fields per entity type (for formatCompactResponse)
var CompactFields = map[string][]string{
    "Tickets":      {"id", "ticketNumber", "title", "status", "priority", "companyID", "assignedResourceID", "createDate", "dueDateTime"},
    "Companies":    {"id", "companyName", "isActive", "phone", "city", "state"},
    "Contacts":     {"id", "firstName", "lastName", "emailAddress", "companyID"},
    "Projects":     {"id", "projectName", "status", "companyID", "projectLeadResourceID", "startDate", "endDate"},
    "Tasks":        {"id", "title", "status", "projectID", "assignedResourceID", "percentComplete"},
    "Resources":    {"id", "firstName", "lastName", "email", "isActive"},
    "BillingItems": {"id", "itemName", "companyID", "ticketID", "projectID", "postedDate", "totalAmount", "invoiceID", "billingItemType"},
    "BillingItemApprovalLevels": {"id", "timeEntryID", "approvalLevel", "approvalResourceID", "approvalDateTime"},
    "TimeEntries":  {"id", "resourceID", "ticketID", "projectID", "taskID", "dateWorked", "hoursWorked", "summaryNotes"},
}

// Tools that use compact formatting
var CompactSearchTools = map[string]string{
    "autotask_search_tickets":                       "Tickets",
    "autotask_search_companies":                     "Companies",
    "autotask_search_contacts":                      "Contacts",
    "autotask_search_projects":                      "Projects",
    "autotask_search_tasks":                         "Tasks",
    "autotask_search_resources":                     "Resources",
    "autotask_search_billing_items":                 "BillingItems",
    "autotask_search_billing_item_approval_levels":  "BillingItemApprovalLevels",
    "autotask_search_time_entries":                  "TimeEntries",
    "autotask_search_ticket_charges":                "TicketCharges",
}
```

**picklist.go** implements the lazy-loading field metadata cache:

```go
type FieldInfo struct {
    Name                   string         `json:"name"`
    DataType               string         `json:"dataType"`
    Length                 int            `json:"length,omitempty"`
    IsRequired             bool           `json:"isRequired"`
    IsReadOnly             bool           `json:"isReadOnly"`
    IsQueryable            bool           `json:"isQueryable"`
    IsReference            bool           `json:"isReference"`
    ReferenceEntityType    string         `json:"referenceEntityType,omitempty"`
    IsPickList             bool           `json:"isPickList"`
    PicklistValues         []PicklistValue `json:"picklistValues,omitempty"`
    PicklistParentField    string         `json:"picklistParentValueField,omitempty"`
}

type PicklistValue struct {
    Value          string `json:"value"`
    Label          string `json:"label"`
    IsDefaultValue bool   `json:"isDefaultValue"`
    SortOrder      int    `json:"sortOrder"`
    IsActive       bool   `json:"isActive"`
    IsSystem       bool   `json:"isSystem"`
    ParentValue    string `json:"parentValue,omitempty"`
}

type PicklistCache struct {
    mu     sync.RWMutex
    cache  map[string][]FieldInfo
    client *Client
    logger *slog.Logger
}

func NewPicklistCache(client *Client, logger *slog.Logger) *PicklistCache
func (p *PicklistCache) GetFields(ctx context.Context, entityType string) ([]FieldInfo, error)
func (p *PicklistCache) GetPicklistValues(ctx context.Context, entityType, fieldName string) ([]PicklistValue, error)
func (p *PicklistCache) GetQueues(ctx context.Context) ([]PicklistValue, error)
func (p *PicklistCache) GetTicketStatuses(ctx context.Context) ([]PicklistValue, error)
func (p *PicklistCache) GetTicketPriorities(ctx context.Context) ([]PicklistValue, error)
```

**Tests should cover:**
- PicklistCache lazy loading (first call fetches, second returns cached)
- GetQueues/GetTicketStatuses/GetTicketPriorities convenience methods
- Thread-safe concurrent access
- Empty/error handling

**Step 1:** Write entities.go (no tests needed — static data).
**Step 2:** Write picklist_test.go with mock client.
**Step 3:** Run tests — verify FAIL.
**Step 4:** Write picklist.go.
**Step 5:** Run tests — verify PASS.
**Step 6:** Commit: `feat: add Autotask entity definitions and picklist cache`

---

## Task 3: Tool Registration Infrastructure & Response Formatting

**Files:**
- Create: `internal/autotask/tools.go`
- Create: `internal/autotask/format.go`
- Create: `internal/autotask/format_test.go`

This task creates the tool registration framework and response formatting. The actual tool definitions are added in Tasks 4-8.

**format.go** implements response formatting:
```go
// FormatSearchResult formats a search result with compact fields and pagination
func FormatSearchResult(toolName string, items []map[string]any, page, pageSize int) string

// FormatGetResult formats a single entity result as full JSON
func FormatGetResult(item map[string]any) string

// FormatCreateResult formats a create response
func FormatCreateResult(entityType string, id int) string

// FormatUpdateResult formats an update response
func FormatUpdateResult(entityType string, id int) string

// FormatDeleteResult formats a delete response
func FormatDeleteResult(entityType string, id int) string

// FormatNotFound formats a not-found error with context
func FormatNotFound(entityType string, criteria map[string]any) string

// EnhanceWithNames inlines company and resource names into items
func (c *Client) EnhanceWithNames(ctx context.Context, items []map[string]any) []map[string]any
```

**tools.go** sets up the registration infrastructure:
```go
// RegisterTools registers all 83 Autotask MCP tools on the server
func RegisterTools(srv *server.MCPServer, client *Client, picklist *PicklistCache, logger *slog.Logger)
```

The registration function defines all tools using `mcp.NewTool()` with `mcp.WithString()`, `mcp.WithNumber()`, `mcp.WithBoolean()` etc., and maps each to a handler function.

**Tests for format.go should cover:**
- Compact formatting picks correct fields per entity type
- Pagination metadata correct
- Enhancement inlines names
- Not-found formatting includes criteria

**Step 1:** Write format_test.go.
**Step 2:** Run tests — FAIL.
**Step 3:** Write format.go.
**Step 4:** Run tests — PASS.
**Step 5:** Write tools.go skeleton (RegisterTools function with just test_connection tool as proof).
**Step 6:** Commit: `feat: add Autotask response formatting and tool registration infrastructure`

---

## Task 4: Tools Batch 1 — Core (25 tools)

**Files:**
- Modify: `internal/autotask/tools.go`

Register these tools in `RegisterTools`:

| # | Tool Name | Handler Pattern |
|---|-----------|----------------|
| 1 | `autotask_test_connection` | `client.TestConnection(ctx)` |
| 2 | `autotask_list_queues` | `picklist.GetQueues(ctx)` |
| 3 | `autotask_list_ticket_statuses` | `picklist.GetTicketStatuses(ctx)` |
| 4 | `autotask_list_ticket_priorities` | `picklist.GetTicketPriorities(ctx)` |
| 5 | `autotask_get_field_info` | `picklist.GetFields(ctx, entityType)` |
| 6 | `autotask_search_companies` | Query Companies with searchTerm(contains companyName), isActive(eq), page, pageSize(max:200) |
| 7 | `autotask_create_company` | Create Companies with companyName(req), companyType(req), phone, address1, city, state, postalCode, ownerResourceID, isActive |
| 8 | `autotask_update_company` | Patch Companies/{id} with companyName, phone, address1, city, state, postalCode, isActive |
| 9 | `autotask_search_contacts` | Query Contacts with searchTerm(or: firstName,lastName,emailAddress), companyID(eq), isActive(eq), page, pageSize(max:200) |
| 10 | `autotask_create_contact` | Create Contacts with companyID(req), firstName(req), lastName(req), emailAddress, phone, title |
| 11 | `autotask_search_tickets` | Query Tickets with searchTerm(beginsWith ticketNumber), companyID(eq), status(eq), assignedResourceID(eq), unassigned(eq assignedResourceID=null), createdAfter(gte createDate), createdBefore(lte createDate), lastActivityAfter(gte lastActivityDate), page, pageSize(max:500) |
| 12 | `autotask_get_ticket_details` | Get Tickets/{ticketID} |
| 13 | `autotask_create_ticket` | Create Tickets with companyID(req), title(req), description(req), status, priority, assignedResourceID, assignedResourceRoleID, contactID |
| 14 | `autotask_update_ticket` | Patch Tickets/{ticketId} with title, description, status, priority, assignedResourceID, assignedResourceRoleID, dueDateTime, contactID |
| 15 | `autotask_get_ticket_charge` | Get TicketCharges/{chargeId} |
| 16 | `autotask_search_ticket_charges` | Query TicketCharges with ticketId(eq ticketID), pageSize(max:100, default:25 with filter, 10 without) |
| 17 | `autotask_create_ticket_charge` | CreateChild Tickets/{ticketID}/Charges with ticketID(req), name(req), chargeType(req), description, unitQuantity, unitPrice, unitCost, datePurchased, productID, billingCodeID, billableToAccount(default:true), status |
| 18 | `autotask_update_ticket_charge` | Patch TicketCharges/{chargeId} with name, description, unitQuantity, unitPrice, unitCost, billableToAccount, status |
| 19 | `autotask_delete_ticket_charge` | DeleteChild Tickets/{ticketId}/Charges/{chargeId} |
| 20 | `autotask_get_ticket_note` | Query TicketNotes with ticketId(eq) AND noteId(eq id), return first |
| 21 | `autotask_search_ticket_notes` | Query TicketNotes with ticketId(eq), pageSize(max:100) |
| 22 | `autotask_create_ticket_note` | CreateChild Tickets/{ticketId}/Notes with ticketID, title, description(req), noteType(default:1), publish(default:1) |
| 23 | `autotask_get_ticket_attachment` | Query TicketAttachments with parentId(eq ticketId) AND id(eq attachmentId), return first |
| 24 | `autotask_search_ticket_attachments` | Query TicketAttachments with parentId(eq ticketId), pageSize(max:50, default:10) |
| 25 | `autotask_search_resources` | Query Resources with searchTerm(or: email,firstName,lastName), isActive(eq), page, pageSize(max:500) |

Each tool follows this pattern:
```go
srv.AddTool(
    mcp.NewTool("autotask_search_companies",
        mcp.WithDescription("Search for companies in Autotask (25 results per page default)"),
        mcp.WithString("searchTerm", mcp.Description("Search term for company name")),
        mcp.WithBoolean("isActive", mcp.Description("Filter by active status")),
        mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
        mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(200)),
        mcp.WithReadOnlyHintAnnotation(true),
        mcp.WithOpenWorldHintAnnotation(true),
    ),
    func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        var filters []Filter
        if term := req.GetString("searchTerm", ""); term != "" {
            filters = append(filters, Filter{Op: "contains", Field: "companyName", Value: term})
        }
        if isActive, err := req.RequireBool("isActive"); err == nil {
            filters = append(filters, Filter{Op: "eq", Field: "isActive", Value: isActive})
        }
        items, err := client.Query(ctx, "Companies", filters, QueryOpts{
            Page:     req.GetInt("page", 1),
            PageSize: req.GetInt("pageSize", 25),
            MaxSize:  200,
        })
        if err != nil {
            return mcputil.ErrorResult(err), nil
        }
        if len(items) == 0 {
            return mcputil.TextResult(FormatNotFound("Companies", map[string]any{"searchTerm": req.GetString("searchTerm", "")})), nil
        }
        items = client.EnhanceWithNames(ctx, items)
        return mcputil.TextResult(FormatSearchResult("autotask_search_companies", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
    },
)
```

**Step 1:** Add all 25 tool definitions to tools.go.
**Step 2:** Run: `go build ./internal/autotask/...` — verify compilation.
**Step 3:** Commit: `feat: add Autotask tools batch 1 — core (companies, contacts, tickets, resources)`

---

## Task 5: Tools Batch 2 — Financial (13 tools)

**Files:**
- Modify: `internal/autotask/tools.go`

| # | Tool Name | Entity | Key Parameters |
|---|-----------|--------|---------------|
| 1 | `autotask_get_quote` | Quotes | quoteId(req) |
| 2 | `autotask_search_quotes` | Quotes | companyId, contactId, opportunityId, searchTerm, pageSize(max:100) |
| 3 | `autotask_create_quote` | Quotes | companyId(req), name, description, contactId, opportunityId, effectiveDate, expirationDate |
| 4 | `autotask_get_quote_item` | QuoteItems | quoteItemId(req) |
| 5 | `autotask_search_quote_items` | QuoteItems | quoteId, searchTerm, pageSize(max:100, default:50) |
| 6 | `autotask_create_quote_item` | QuoteItems | quoteId(req), quantity(req), name, description, unitPrice, unitCost, unitDiscount, lineDiscount, percentageDiscount, isOptional, serviceID, productID, serviceBundleID, sortOrderID, quoteItemType |
| 7 | `autotask_update_quote_item` | QuoteItems | quoteItemId(req), quantity, unitPrice, unitDiscount, lineDiscount, percentageDiscount, isOptional, sortOrderID |
| 8 | `autotask_delete_quote_item` | QuoteItems | quoteId(req), quoteItemId(req) — DeleteChild Quotes/{quoteId}/Items/{quoteItemId} |
| 9 | `autotask_get_opportunity` | Opportunities | opportunityId(req) |
| 10 | `autotask_search_opportunities` | Opportunities | companyId, searchTerm, status, pageSize(max:100) |
| 11 | `autotask_create_opportunity` | Opportunities | title(req), companyId(req), ownerResourceId(req), status(req), stage(req), projectedCloseDate(req), startDate(req), probability(default:50), amount(default:0), cost(default:0), useQuoteTotals(default:true), totalAmountMonths, contactId, description, opportunityCategoryID |
| 12 | `autotask_search_invoices` | Invoices | companyID, invoiceNumber, isVoided, pageSize(max:500) |
| 13 | `autotask_search_contracts` | Contracts | searchTerm(contains contractName), companyID, status, pageSize(max:500) |

**Step 1:** Add all 13 tools to tools.go.
**Step 2:** Verify compilation.
**Step 3:** Commit: `feat: add Autotask tools batch 2 — financial (quotes, opportunities, invoices, contracts)`

---

## Task 6: Tools Batch 3 — Operations (18 tools)

**Files:**
- Modify: `internal/autotask/tools.go`

| # | Tool Name | Entity | Key Parameters |
|---|-----------|--------|---------------|
| 1 | `autotask_search_projects` | Projects | searchTerm, companyID, status, projectLeadResourceID, page, pageSize(max:100) |
| 2 | `autotask_create_project` | Projects | companyID(req), projectName(req), status(req), projectType(req), description, startDate→startDateTime, endDate→endDateTime, projectLeadResourceID, estimatedHours |
| 3 | `autotask_search_tasks` | Tasks | searchTerm, projectID, status, assignedResourceID, page, pageSize(max:100) |
| 4 | `autotask_create_task` | Tasks | projectID(req), title(req), status(req), description, assignedResourceID, estimatedHours, taskType(default:1), startDateTime, endDateTime |
| 5 | `autotask_list_phases` | Phases | projectID(req), pageSize(max:100) — uses GET /Projects/{projectID}/Phases |
| 6 | `autotask_create_phase` | Phases | projectID(req), title(req), description, startDate, dueDate, estimatedHours — CreateChild Projects/{projectID}/Phases |
| 7 | `autotask_create_time_entry` | TimeEntries | resourceID(req), dateWorked(req), hoursWorked(req), summaryNotes(req), ticketID, taskID, startDateTime, endDateTime, internalNotes — child entity routing based on ticketID/taskID |
| 8 | `autotask_search_time_entries` | TimeEntries | resourceId, ticketId, projectId, taskId, approvalStatus, billable, dateWorkedAfter, dateWorkedBefore, page, pageSize(max:500) |
| 9 | `autotask_search_billing_items` | BillingItems | companyId, ticketId, projectId, contractId, invoiceId, postedAfter, postedBefore, page, pageSize(max:500) |
| 10 | `autotask_get_billing_item` | BillingItems | billingItemId(req) |
| 11 | `autotask_search_billing_item_approval_levels` | BillingItemApprovalLevels | timeEntryId, approvalResourceId, approvalLevel, approvedAfter, approvedBefore, page, pageSize(max:500) |
| 12 | `autotask_get_expense_report` | ExpenseReports | reportId(req) |
| 13 | `autotask_search_expense_reports` | ExpenseReports | submitterId, status, pageSize(max:100) |
| 14 | `autotask_create_expense_report` | ExpenseReports | name(req), submitterId(req), weekEndingDate(req), description |
| 15 | `autotask_create_expense_item` | ExpenseItems | expenseReportId(req), description(req), expenseDate(req), expenseCategory(req), amount(req), companyId(default:0), haveReceipt(default:false), isBillableToCompany(default:false), isReimbursable(default:true), paymentType |
| 16 | `autotask_search_configuration_items` | ConfigurationItems | searchTerm, companyID, isActive, productID, pageSize(max:500) |
| 17 | `autotask_get_project_note` | ProjectNotes | projectId(req), noteId(req) — Query with projectId AND id filter |
| 18 | `autotask_search_project_notes` | ProjectNotes | projectId(req), pageSize(max:100) |

**Step 1:** Add all 18 tools.
**Step 2:** Verify compilation.
**Step 3:** Commit: `feat: add Autotask tools batch 3 — operations (projects, tasks, time, billing, expenses)`

---

## Task 7: Tools Batch 4 — Dispatch & Notes (15 tools)

**Files:**
- Modify: `internal/autotask/tools.go`

| # | Tool Name | Entity | Key Parameters |
|---|-----------|--------|---------------|
| 1 | `autotask_create_project_note` | ProjectNotes | projectId(req), description(req), title, noteType, publish(default:1), isAnnouncement(default:false) — CreateChild Projects/{projectId}/Notes |
| 2 | `autotask_get_company_note` | CompanyNotes | companyId(req), noteId(req) — Query with accountId AND id filter |
| 3 | `autotask_search_company_notes` | CompanyNotes | companyId(req), pageSize(max:100) |
| 4 | `autotask_create_company_note` | CompanyNotes | companyId(req), description(req), title, actionType — CreateChild Companies/{companyId}/Notes |
| 5 | `autotask_search_service_calls` | ServiceCalls | status, startDate, endDate, page, pageSize(max:200) |
| 6 | `autotask_get_service_call` | ServiceCalls | serviceCallId(req) |
| 7 | `autotask_create_service_call` | ServiceCalls | description(req), startDateTime(req), endDateTime(req), status, duration, companyID, companyLocationID, complete(default:false) |
| 8 | `autotask_update_service_call` | ServiceCalls | serviceCallId(req), description, startDateTime, endDateTime, status, duration, complete |
| 9 | `autotask_delete_service_call` | ServiceCalls | serviceCallId(req) |
| 10 | `autotask_search_service_call_tickets` | ServiceCallTickets | serviceCallId, ticketId, page, pageSize(max:200) |
| 11 | `autotask_create_service_call_ticket` | ServiceCallTickets | serviceCallID(req), ticketID(req) |
| 12 | `autotask_delete_service_call_ticket` | ServiceCallTickets | id(req) |
| 13 | `autotask_search_service_call_ticket_resources` | ServiceCallTicketResources | serviceCallTicketId, resourceId, page, pageSize(max:200) |
| 14 | `autotask_create_service_call_ticket_resource` | ServiceCallTicketResources | serviceCallTicketID(req), resourceID(req), roleID |
| 15 | `autotask_delete_service_call_ticket_resource` | ServiceCallTicketResources | id(req) |

**Step 1:** Add all 15 tools.
**Step 2:** Verify compilation.
**Step 3:** Commit: `feat: add Autotask tools batch 4 — service calls, notes`

---

## Task 8: Tools Batch 5 — Products, Services & Meta-Tools (12 tools)

**Files:**
- Modify: `internal/autotask/tools.go`
- Create: `internal/autotask/metatools.go`

| # | Tool Name | Type | Description |
|---|-----------|------|-------------|
| 1 | `autotask_get_product` | Get | productId(req) |
| 2 | `autotask_search_products` | Query | searchTerm, isActive, pageSize(max:100) |
| 3 | `autotask_get_service` | Get | serviceId(req) |
| 4 | `autotask_search_services` | Query | searchTerm, isActive, pageSize(max:100) |
| 5 | `autotask_get_service_bundle` | Get | serviceBundleId(req) |
| 6 | `autotask_search_service_bundles` | Query | searchTerm, isActive, pageSize(max:100) |
| 7 | `autotask_list_categories` | Meta | Returns category list with tool counts |
| 8 | `autotask_list_category_tools` | Meta | category(req) — returns tools with schemas |
| 9 | `autotask_execute_tool` | Meta | toolName(req), arguments(object) — dynamic dispatch |
| 10 | `autotask_router` | Meta | intent(req) — natural language intent routing |

**metatools.go** implements the meta-tool logic:
- `ToolCategories` — map of category name to tool names
- `ListCategories()` — returns category summary
- `ListCategoryTools(category)` — returns tool definitions for category
- `RouteIntent(intent)` — keyword-based intent matching to suggest tools

**Step 1:** Add 6 product/service tools to tools.go.
**Step 2:** Write metatools.go with meta-tool infrastructure.
**Step 3:** Add 4 meta-tools to tools.go.
**Step 4:** Verify compilation.
**Step 5:** Commit: `feat: add Autotask tools batch 5 — products, services, meta-tools`

---

## Task 9: Entry Point & Integration

**Files:**
- Create: `cmd/autotask-mcp/main.go`
- Delete: `cmd/autotask-mcp/.gitkeep`

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "github.com/Logiphys/lgp-mcp-servers/internal/autotask"
    "github.com/Logiphys/lgp-mcp-servers/pkg/config"
    "github.com/mark3labs/mcp-go/server"
)

var version = "dev"

func main() {
    level := config.LogLevel()
    logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

    cfg := autotask.Config{
        Username:        config.MustEnv("AUTOTASK_USERNAME"),
        Secret:          config.MustEnv("AUTOTASK_SECRET"),
        IntegrationCode: config.MustEnv("AUTOTASK_INTEGRATION_CODE"),
        BaseURL:         config.OptEnv("AUTOTASK_API_URL", "https://webservices24.autotask.net/ATServicesRest"),
    }

    client := autotask.NewClient(cfg, logger)
    picklist := autotask.NewPicklistCache(client, logger)

    srv := server.NewMCPServer(
        "autotask-mcp",
        version,
        server.WithLogging(),
    )

    autotask.RegisterTools(srv, client, picklist, logger)

    logger.Info("starting autotask-mcp", "version", version)

    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    if err := server.ServeStdio(srv); err != nil {
        logger.Error("server failed", "error", err)
        os.Exit(1)
    }
}
```

**Step 1:** Write main.go.
**Step 2:** Run: `go build -o dist/autotask-mcp ./cmd/autotask-mcp` — verify binary builds.
**Step 3:** Commit: `feat: add autotask-mcp entry point with stdio transport`

---

## Task 10: Build Verification & Cleanup

**Step 1:** Run full test suite: `go test ./... -race -count=1`
**Step 2:** Run `go vet ./...`
**Step 3:** Build binary: `make build-autotask-mcp`
**Step 4:** Remove remaining .gitkeep files in internal/autotask/ and cmd/autotask-mcp/
**Step 5:** Commit cleanup if needed: `chore: remove .gitkeep files from autotask directories`

---

## Summary

| Task | What | Tools | Files |
|------|------|-------|-------|
| 1 | API Client | — | client.go, client_test.go |
| 2 | Entities & Picklist | — | entities.go, picklist.go, picklist_test.go |
| 3 | Tool infra & formatting | — | tools.go, format.go, format_test.go |
| 4 | Batch 1: Core | 25 | tools.go (modify) |
| 5 | Batch 2: Financial | 13 | tools.go (modify) |
| 6 | Batch 3: Operations | 18 | tools.go (modify) |
| 7 | Batch 4: Dispatch & Notes | 15 | tools.go (modify) |
| 8 | Batch 5: Products & Meta | 12 | tools.go (modify), metatools.go |
| 9 | Entry point | — | cmd/autotask-mcp/main.go |
| 10 | Verification | — | — |

**Total: 10 tasks, 83 tools, ~10 new files.**
