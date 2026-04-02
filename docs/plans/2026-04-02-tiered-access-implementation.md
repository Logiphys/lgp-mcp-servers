# Tiered Access Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add three-tier GDPR-aware access control to all 9 MCP servers so tools are conditionally registered based on an `*_ACCESS_TIER` environment variable (default: 1 = safe read-only).

**Architecture:** Each server's `RegisterTools` function gains a `tier int` parameter. A new `config.AccessTier()` helper reads/validates the env var. Tool registration is gated by `if tier >= N` checks. `server_info` is enhanced to show the active tier. Six servers that only have read-only APIs still benefit from tier 1/2 separation (hiding sensitive data endpoints).

**Tech Stack:** Go 1.23, `github.com/mark3labs/mcp-go`, `slog`, table-driven tests with `-race`

**Design doc:** `docs/plans/2026-04-02-tiered-access-design.md`

---

### Task 1: Add `config.AccessTier()` helper

**Files:**
- Modify: `pkg/config/config.go`
- Modify: `pkg/config/config_test.go`

**Step 1: Write failing tests**

Add to `pkg/config/config_test.go`:

```go
func TestAccessTier(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want int
	}{
		{"default when unset", "", 1},
		{"explicit tier 1", "1", 1},
		{"explicit tier 2", "2", 2},
		{"explicit tier 3", "3", 3},
		{"clamp below minimum", "0", 1},
		{"clamp above maximum", "5", 3},
		{"invalid string", "abc", 1},
		{"negative", "-1", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env == "" {
				os.Unsetenv("TEST_ACCESS_TIER")
			} else {
				t.Setenv("TEST_ACCESS_TIER", tt.env)
			}
			if got := AccessTier("TEST_ACCESS_TIER"); got != tt.want {
				t.Errorf("AccessTier(%q) = %d, want %d", tt.env, got, tt.want)
			}
		})
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd /Users/zeisler/lgp-mcp-servers && go test ./pkg/config/ -run TestAccessTier -v`
Expected: FAIL — `AccessTier` not defined

**Step 3: Implement `AccessTier`**

Add to `pkg/config/config.go` (add `"strconv"` to imports):

```go
// AccessTier reads an access-tier env var, returning 1–3 (default 1).
func AccessTier(envKey string) int {
	v := os.Getenv(envKey)
	if v == "" {
		return 1
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return 1
	}
	if n > 3 {
		return 3
	}
	return n
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/zeisler/lgp-mcp-servers && go test ./pkg/config/ -run TestAccessTier -v -race`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/config/config.go pkg/config/config_test.go
git commit -m "feat: add config.AccessTier() helper for tiered access control"
```

---

### Task 2: Enhance `server_info` to show tier

**Files:**
- Modify: `pkg/mcputil/serverinfo.go`

**Step 1: Add `AccessTier` field to `ServerInfo` struct and include it in output**

In `pkg/mcputil/serverinfo.go`, add `AccessTier int` to the `ServerInfo` struct. Update the handler to include `access_tier` and `tier_description` in the JSON output.

Tier descriptions:
- 1: `"Safe Read-Only"`
- 2: `"Read + Sensitive Data"`
- 3: `"Full Access"`

```go
type ServerInfo struct {
	Name       string
	Version    string
	BuildDate  string
	Prefix     string
	AccessTier int
}
```

Update the handler's result map to use `map[string]any` instead of `map[string]string`:

```go
func tierDescription(tier int) string {
	switch tier {
	case 2:
		return "Read + Sensitive Data"
	case 3:
		return "Full Access"
	default:
		return "Safe Read-Only"
	}
}
```

And in the handler:

```go
return JSONResult(map[string]any{
	"server":           info.Name,
	"version":          info.Version,
	"build_date":       buildDate,
	"developer":        "Logiphys Datensysteme GmbH",
	"website":          "https://logiphys.de",
	"runtime":          runtime.Version(),
	"os":               runtime.GOOS,
	"arch":             runtime.GOARCH,
	"access_tier":      info.AccessTier,
	"tier_description": tierDescription(info.AccessTier),
}), nil
```

**Step 2: Run full test suite**

Run: `cd /Users/zeisler/lgp-mcp-servers && go test ./pkg/... -v -race`
Expected: PASS (no existing tests break — ServerInfo is constructed by callers)

**Step 3: Commit**

```bash
git add pkg/mcputil/serverinfo.go
git commit -m "feat: show access_tier and tier_description in server_info"
```

---

### Task 3: IT Glue — Add tiered access

**Files:**
- Modify: `internal/itglue/tools.go`
- Modify: `cmd/itglue-mcp/main.go`

**Step 1: Update `RegisterTools` signature and add tier gating**

In `internal/itglue/tools.go`:

```go
func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger, tier int) {
	// Tier 1 — Safe Read-Only (always registered)
	registerOrganizationTools(srv, client, logger)
	registerConfigurationTools(srv, client, logger)
	registerDocumentReadTools(srv, client, logger)     // search + get only
	registerFlexibleAssetTools(srv, client, logger)
	registerHealthTools(srv, client, logger)
	registerLocationReadTools(srv, client, logger)      // search + get only
	registerMetadataTools(srv, client, logger)
	registerDomainTools(srv, client, logger)
	registerExpirationTools(srv, client, logger)
	registerConfigurationInterfaceTools(srv, client, logger)

	// Tier 2 — Sensitive Data
	if tier >= 2 {
		registerContactTools(srv, client, logger)
		registerPasswordTools(srv, client, logger)
	}

	// Tier 3 — Write Operations
	if tier >= 3 {
		registerDocumentWriteTools(srv, client, logger)   // create, update, delete, publish
	}
}
```

**Important:** IT Glue document tools currently combine read+write in `registerDocumentTools`. This function must be split into two:
- `registerDocumentReadTools` — search_documents, get_document, list_document_sections, get_document_section
- `registerDocumentWriteTools` — create_document, update_document, delete_document, create_document_section, update_document_section, delete_document_section, publish_document

Similarly for locations, if `registerLocationTools` contains any write ops. Check the file — if it only has search+get, just keep using `registerLocationTools` (no rename needed).

**Step 2: Update `cmd/itglue-mcp/main.go`**

```go
tier := config.AccessTier("ITGLUE_ACCESS_TIER")

// ... existing code ...

itglue.RegisterTools(srv, client, logger, tier)
mcputil.RegisterServerInfoTool(srv, mcputil.ServerInfo{
	Name: "itglue-mcp", Version: version, BuildDate: buildDate,
	Prefix: "itglue", AccessTier: tier,
})
```

**Step 3: Build and verify**

Run: `cd /Users/zeisler/lgp-mcp-servers && PATH="/opt/homebrew/bin:$PATH" make build-itglue-mcp`
Expected: BUILD SUCCESS

**Step 4: Run tests**

Run: `cd /Users/zeisler/lgp-mcp-servers && go test ./internal/itglue/... -v -race`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/itglue/tools.go internal/itglue/tools_documents.go cmd/itglue-mcp/main.go
git commit -m "feat(itglue): add tiered access — contacts/passwords require tier 2, writes require tier 3"
```

---

### Task 4: Autotask — Add tiered access

**Files:**
- Modify: `internal/autotask/tools.go`
- Modify: `internal/autotask/tools_contacts.go`
- Modify: `internal/autotask/tools_companies.go`
- Modify: `internal/autotask/tools_tickets.go`
- Modify: `internal/autotask/tools_projects.go`
- Modify: `internal/autotask/tools_financial.go`
- Modify: `internal/autotask/tools_time_billing.go`
- Modify: `internal/autotask/tools_service_calls.go`
- Modify: `internal/autotask/tools_products.go`
- Modify: `cmd/autotask-mcp/main.go`

**Approach:** Autotask groups contain mixed read+write tools (e.g., `registerCompanyTools` has both `search_companies` and `create_company`). Two options:

**Option A (recommended):** Split each `registerXxxTools` into `registerXxxReadTools` and `registerXxxWriteTools`.

**Option B:** Pass `tier` into each sub-registration function and gate individual tool registrations.

Use **Option A** — cleaner, no tier logic scattered through 10 files. Each existing `registerXxxTools` function gets renamed/split:

| Current function | Tier 1 (read) | Tier 2 (sensitive) | Tier 3 (write) |
|---|---|---|---|
| `registerUtilityTools` | Keep as-is (test_connection, router, get_field_info) | — | — |
| `registerCompanyTools` | `registerCompanyReadTools` (search_companies, search_company_notes, get_company_note) | — | `registerCompanyWriteTools` (create_company, update_company, create_company_note) |
| `registerContactTools` | — | `registerContactTools` (search_contacts) | `registerContactWriteTools` (create_contact) |
| `registerTicketTools` | `registerTicketReadTools` (search_tickets, get_ticket_details, list_statuses, list_priorities, list_queues, search_ticket_notes, get_ticket_note, search_ticket_attachments, get_ticket_attachment, search_ticket_charges, get_ticket_charge) | — | `registerTicketWriteTools` (create_ticket, update_ticket, create_ticket_note, create_ticket_charge, update_ticket_charge, delete_ticket_charge) |
| `registerProjectTools` | `registerProjectReadTools` (search_projects, search_project_notes, get_project_note, list_phases) | — | `registerProjectWriteTools` (create_project, create_project_note, create_phase, create_task) |
| `registerFinancialTools` | `registerFinancialReadTools` (search_billing_items, get_billing_item, search_billing_item_approval_levels, search_invoices, search_quotes, get_quote, search_quote_items, get_quote_item, search_opportunities, get_opportunity, search_contracts) | search_expense_reports, get_expense_report | `registerFinancialWriteTools` (create_quote, create_quote_item, update_quote_item, delete_quote_item, create_opportunity, create_expense_item, create_expense_report) |
| `registerTimeBillingTools` | — | `registerTimeBillingReadTools` (search_time_entries, search_resources, search_configuration_items) | `registerTimeBillingWriteTools` (create_time_entry) |
| `registerServiceCallTools` | `registerServiceCallReadTools` (search_service_calls, get_service_call, search_service_call_tickets, search_service_call_ticket_resources) | — | `registerServiceCallWriteTools` (create/update/delete service_call, create/delete service_call_ticket, create/delete service_call_ticket_resource) |
| `registerProductTools` | Keep as-is (search_products, get_product, search_services, get_service, search_service_bundles, get_service_bundle) — all read-only | — | — |
| `registerMetaTools` | Keep as-is (list_categories, list_category_tools, execute_tool) | — | — |

**Step 1: Update `internal/autotask/tools.go`**

```go
func RegisterTools(srv *server.MCPServer, client *Client, picklist *PicklistCache, logger *slog.Logger, tier int) {
	// Tier 1 — Safe Read-Only
	registerUtilityTools(srv, client, picklist, logger)
	registerCompanyReadTools(srv, client, logger)
	registerTicketReadTools(srv, client, picklist, logger)
	registerProjectReadTools(srv, client, logger)
	registerFinancialReadTools(srv, client, logger)
	registerServiceCallReadTools(srv, client, logger)
	registerProductTools(srv, client, logger)
	registerMetaTools(srv, client, logger)

	// Tier 2 — Sensitive (personal data, time entries)
	if tier >= 2 {
		registerContactReadTools(srv, client, logger)
		registerTimeBillingReadTools(srv, client, logger)
		registerFinancialSensitiveTools(srv, client, logger)  // expense reports
	}

	// Tier 3 — Write Operations
	if tier >= 3 {
		registerCompanyWriteTools(srv, client, logger)
		registerContactWriteTools(srv, client, logger)
		registerTicketWriteTools(srv, client, logger)
		registerProjectWriteTools(srv, client, logger)
		registerFinancialWriteTools(srv, client, logger)
		registerTimeBillingWriteTools(srv, client, logger)
		registerServiceCallWriteTools(srv, client, logger)
	}
}
```

**Step 2: Split each tools_*.go file** into read/write functions. The existing functions are renamed — the tool registration code inside stays identical, just grouped differently.

**Step 3: Update `cmd/autotask-mcp/main.go`**

```go
tier := config.AccessTier("AUTOTASK_ACCESS_TIER")
autotask.RegisterTools(srv, client, picklist, logger, tier)
mcputil.RegisterServerInfoTool(srv, mcputil.ServerInfo{
	Name: "autotask-mcp", Version: version, BuildDate: buildDate,
	Prefix: "autotask", AccessTier: tier,
})
```

**Step 4: Build and test**

Run: `cd /Users/zeisler/lgp-mcp-servers && PATH="/opt/homebrew/bin:$PATH" make build-autotask-mcp && go test ./internal/autotask/... -v -race`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/autotask/ cmd/autotask-mcp/main.go
git commit -m "feat(autotask): add tiered access — contacts/time entries tier 2, all writes tier 3"
```

---

### Task 5: Datto RMM — Add tiered access

**Files:**
- Modify: `internal/dattormm/tools.go`
- Modify: `internal/dattormm/tools_sites.go` (split read/write)
- Modify: `internal/dattormm/tools_devices.go` (split read/write)
- Modify: `internal/dattormm/tools_alerts.go` (split read/write)
- Modify: `internal/dattormm/tools_variables.go` (split read/write)
- Modify: `internal/dattormm/tools_jobs.go` (move to tier 2)
- Modify: `internal/dattormm/tools_audit.go` (split tier 1/2)
- Modify: `internal/dattormm/tools_activity.go` (move to tier 2)
- Modify: `internal/dattormm/tools_account.go` (split for list_users)
- Modify: `cmd/datto-rmm-mcp/main.go`

**Same split approach as Autotask.** Refer to design doc for exact tier assignments.

**Step 1: Update `internal/dattormm/tools.go`**

```go
func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger, tier int) {
	// Tier 1 — Safe Read-Only
	registerAccountReadTools(srv, client, logger)    // get_account
	registerSiteReadTools(srv, client, logger)
	registerDeviceReadTools(srv, client, logger)
	registerAlertReadTools(srv, client, logger)
	registerAuditSafeTools(srv, client, logger)      // esxi, printer
	registerFilterTools(srv, client, logger)
	registerSystemTools(srv, client, logger)

	// Tier 2 — Sensitive
	if tier >= 2 {
		registerAccountSensitiveTools(srv, client, logger) // list_users
		registerAuditSensitiveTools(srv, client, logger)   // device_audit, device_software
		registerActivityTools(srv, client, logger)
		registerJobTools(srv, client, logger)
		registerVariableReadTools(srv, client, logger)
	}

	// Tier 3 — Write
	if tier >= 3 {
		registerSiteWriteTools(srv, client, logger)
		registerDeviceWriteTools(srv, client, logger)
		registerAlertWriteTools(srv, client, logger)     // resolve_alert
		registerJobWriteTools(srv, client, logger)       // create_quick_job
		registerVariableWriteTools(srv, client, logger)
	}
}
```

**Step 2: Split each tools_*.go, update main.go** (same pattern as Tasks 3-4)

**Step 3: Build and test**

Run: `cd /Users/zeisler/lgp-mcp-servers && PATH="/opt/homebrew/bin:$PATH" make build-datto-rmm-mcp && go test ./internal/dattormm/... -v -race`

**Step 4: Commit**

```bash
git add internal/dattormm/ cmd/datto-rmm-mcp/main.go
git commit -m "feat(datto-rmm): add tiered access — audits/logs tier 2, writes tier 3"
```

---

### Task 6: Datto EDR — Add tiered access

**Files:**
- Modify: `internal/dattoedr/tools.go`
- Modify: `cmd/datto-edr-mcp/main.go`

**This server is simple — all tools are registered individually in `RegisterTools`, no sub-functions to split.**

**Step 1: Update `internal/dattoedr/tools.go`**

```go
func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger, tier int) {
	// Tier 1 — Safe Read-Only
	registerTestConnection(srv, client, logger)
	registerGetDashboard(srv, client, logger)
	registerListAgents(srv, client, logger)
	registerGetAgent(srv, client, logger)
	registerGetAgentCount(srv, client, logger)
	registerListAlerts(srv, client, logger)
	registerGetAlert(srv, client, logger)
	registerGetAlertCount(srv, client, logger)
	registerListOrganizations(srv, client, logger)
	registerListLocations(srv, client, logger)
	registerListDeviceGroups(srv, client, logger)
	registerListPolicies(srv, client, logger)
	registerListRules(srv, client, logger)
	registerListSuppressionRules(srv, client, logger)
	registerListExtensions(srv, client, logger)

	// Tier 2 — Sensitive
	if tier >= 2 {
		registerListAlertsArchive(srv, client, logger)
		registerListQuarantinedFiles(srv, client, logger)
	}

	// Tier 3 — Actions
	if tier >= 3 {
		registerScanAgent(srv, client, logger)
		registerIsolateHost(srv, client, logger)
		registerRestoreHost(srv, client, logger)
	}
}
```

**Step 2: Update `cmd/datto-edr-mcp/main.go`** (same pattern)

**Step 3: Build and test**

**Step 4: Commit**

```bash
git add internal/dattoedr/tools.go cmd/datto-edr-mcp/main.go
git commit -m "feat(datto-edr): add tiered access — archive/quarantine tier 2, scan/isolate tier 3"
```

---

### Task 7: Datto UC — Add tiered access

**Files:**
- Modify: `internal/dattouc/tools.go`
- Modify: `cmd/datto-uc-mcp/main.go`

**Simple — individual registrations, no sub-functions to split. No tier 3 (read-only API).**

**Step 1: Update `internal/dattouc/tools.go`**

```go
func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger, tier int) {
	// Tier 1 — Safe Read-Only
	registerTestConnection(srv, client, logger)
	registerListDevices(srv, client, logger)
	registerGetDevice(srv, client, logger)
	registerListDeviceAssets(srv, client, logger)
	registerListDeviceAgents(srv, client, logger)
	registerListDeviceShares(srv, client, logger)
	registerListDeviceAlerts(srv, client, logger)
	registerListDeviceVMRestores(srv, client, logger)
	registerListAgents(srv, client, logger)
	registerGetDeviceVolumeAssets(srv, client, logger)
	registerListDTCAssets(srv, client, logger)
	registerListDTCRMMTemplates(srv, client, logger)
	registerGetDTCStoragePool(srv, client, logger)
	registerListDTCClientAssets(srv, client, logger)
	registerGetDTCAsset(srv, client, logger)
	registerListSaaSDomains(srv, client, logger)

	// Tier 2 — Sensitive
	if tier >= 2 {
		registerGetActivityLog(srv, client, logger)
		registerGetSaaSApplications(srv, client, logger)
		registerGetSaaSSeats(srv, client, logger)
	}
}
```

**Step 2: Update main.go, build, test, commit**

```bash
git commit -m "feat(datto-uc): add tiered access — activity log/SaaS data tier 2"
```

---

### Task 8: RocketCyber — Add tiered access

**Files:**
- Modify: `internal/rocketcyber/tools.go`
- Modify: `cmd/rocketcyber-mcp/main.go`

**Simple — individual registrations. No tier 3.**

**Step 1: Update `internal/rocketcyber/tools.go`**

```go
func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger, tier int) {
	// Tier 1 — Safe Read-Only
	registerTestConnection(srv, client, logger)
	registerListAgents(srv, client, logger)
	registerListEvents(srv, client, logger)
	registerGetEventSummary(srv, client, logger)
	registerListIncidents(srv, client, logger)
	registerListApps(srv, client, logger)
	registerListFirewalls(srv, client, logger)
	registerListSuppressionRules(srv, client, logger)
	registerGetSuppressionRule(srv, client, logger)

	// Tier 2 — Sensitive
	if tier >= 2 {
		registerGetAccount(srv, client, logger)
		registerGetDefender(srv, client, logger)
		registerGetOffice(srv, client, logger)
	}
}
```

**Step 2: Update main.go, build, test, commit**

```bash
git commit -m "feat(rocketcyber): add tiered access — account/defender/office details tier 2"
```

---

### Task 9: Datto Networking — Add tiered access

**Files:**
- Modify: `internal/dattonetwork/tools.go`
- Modify: `cmd/datto-network-mcp/main.go`

**No tier 3.**

**Step 1: Update `internal/dattonetwork/tools.go`**

```go
func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger, tier int) {
	// Tier 1 — Safe Read-Only
	registerTestConnection(srv, client, logger)
	registerListDevices(srv, client, logger)
	registerGetDevice(srv, client, logger)
	registerGetDevicesOverview(srv, client, logger)
	registerGetResellerOverview(srv, client, logger)
	registerGetRouter(srv, client, logger)
	registerGetWhoami(srv, client, logger)
	registerGetUserDevices(srv, client, logger)

	// Tier 2 — Sensitive (client usage data)
	if tier >= 2 {
		registerGetDeviceClientsOverview(srv, client, logger)
		registerGetDeviceClientsUsage(srv, client, logger)
		registerGetDeviceWanUsage(srv, client, logger)
		registerGetDeviceApplications(srv, client, logger)
	}
}
```

**Step 2: Update main.go, build, test, commit**

```bash
git commit -m "feat(datto-network): add tiered access — client usage data tier 2"
```

---

### Task 10: Datto Backup — Add tiered access

**Files:**
- Modify: `internal/dattobackup/tools.go`
- Modify: `cmd/datto-backup-mcp/main.go`

**No tier 3.**

**Step 1: Update `internal/dattobackup/tools.go`**

```go
func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger, tier int) {
	// Tier 1 — Safe Read-Only
	registerTestConnection(srv, client, logger)
	registerListAppliances(srv, client, logger)
	registerListAssets(srv, client, logger)
	registerListBackups(srv, client, logger)
	registerListAlerts(srv, client, logger)
	registerGetAgentVersion(srv, client, logger)
	registerListSpanningDomains(srv, client, logger)
	registerListEntraDomains(srv, client, logger)

	// Tier 2 — Sensitive (customer/user data)
	if tier >= 2 {
		registerListCustomers(srv, client, logger)
		registerListEndpointAssets(srv, client, logger)
		registerListSpanningDomainUsers(srv, client, logger)
	}
}
```

**Step 2: Update main.go, build, test, commit**

```bash
git commit -m "feat(datto-backup): add tiered access — customer/user data tier 2"
```

---

### Task 11: MyITProcess — Add tiered access

**Files:**
- Modify: `internal/myitprocess/tools.go`
- Modify: `cmd/myitprocess-mcp/main.go`

**No tier 3.**

**Step 1: Update `internal/myitprocess/tools.go`**

```go
func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger, tier int) {
	// Tier 1 — Safe Read-Only
	registerTestConnection(srv, client, logger)
	registerListReviews(srv, client, logger)
	registerListOverdueReviews(srv, client, logger)
	registerListFindings(srv, client, logger)
	registerListRecommendations(srv, client, logger)
	registerGetRecommendationConfigurations(srv, client, logger)
	registerListInitiatives(srv, client, logger)

	// Tier 2 — Sensitive (client/user data)
	if tier >= 2 {
		registerListClients(srv, client, logger)
		registerListUsers(srv, client, logger)
		registerListMeetings(srv, client, logger)
	}
}
```

**Step 2: Update main.go, build, test, commit**

```bash
git commit -m "feat(myitprocess): add tiered access — client/user data tier 2"
```

---

### Task 12: Full build + test all servers

**Step 1: Build all servers**

Run: `cd /Users/zeisler/lgp-mcp-servers && PATH="/opt/homebrew/bin:$PATH" make build`
Expected: All 9 binaries build successfully

**Step 2: Run all tests**

Run: `cd /Users/zeisler/lgp-mcp-servers && go test ./... -race`
Expected: All PASS

**Step 3: Run linter**

Run: `cd /Users/zeisler/lgp-mcp-servers && PATH="/opt/homebrew/bin:$PATH" make lint`
Expected: No issues

**Step 4: Commit any fixes if needed**

---

### Task 13: Update documentation

**Files:**
- Modify: `README.md` — add Tiered Access section with env var table, tier descriptions, migration guide
- Modify: `CHANGELOG.md` — add v0.8.0 entry (breaking change)
- Modify: `config/claude-code-settings.json.example` — add `*_ACCESS_TIER` comments
- Modify: `config/claude-desktop-config.json.example` — add `*_ACCESS_TIER` comments
- Modify: `docs/api-quirks.md` — mention tier system if relevant

**Step 1: Update README.md**

Add a new section "## Access Tiers (GDPR/Privacy)" after the Quick Start section explaining:
- The three tiers and what each exposes
- Default is Tier 1 (breaking change from previous versions)
- Table of env vars per server
- Example config showing how to set tiers
- Migration guide for existing users

**Step 2: Update CHANGELOG.md**

Add v0.8.0 entry noting breaking change.

**Step 3: Update config examples**

Add `ACCESS_TIER` env vars (commented out, default 1) to both config files.

**Step 4: Commit**

```bash
git add README.md CHANGELOG.md config/
git commit -m "docs: add tiered access documentation and migration guide"
```

---

### Task 14: Install, tag, and push

**Step 1: Install binaries**

```bash
cd /Users/zeisler/lgp-mcp-servers
for bin in dist/darwin-arm64/*; do cp "$bin" /usr/local/bin/; done
```

**Step 2: Tag and push**

```bash
git tag v0.8.0
git push origin main --tags
```
