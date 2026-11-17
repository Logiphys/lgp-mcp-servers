# Known API Quirks & Workarounds

## Autotask PSA

### `search_resources` -> HTTP 500
- **Issue:** `search_resources` consistently returns HTTP 500 from the Autotask API.
- **Workaround:** Use `list` endpoint with client-side filtering instead.
- **Status:** Persistent, not retryable.

### `search_invoices` with companyID -> Inconsistent Results
- **Issue:** Filtering invoices by `companyID` returns incomplete or inconsistent data.
- **Workaround:** Use `search_billing_items` instead, which provides reliable company-level billing data.

### Pagination on Large Result Sets
- **Issue:** Default pagination can time out or return errors on very large collections.
- **Workaround:** Use `postedAfter` date filter to narrow the result set before paginating.

### Company Search Pagination

`search_companies` returns one real upstream page. Start with, for example:

```json
{"isActive": true, "pageSize": 200, "page": 1}
```

The JSON text response retains `items` and `summary.returned`, `page`,
`pageSize`, and `hasMore`, and adds `summary.nextPageUrl` (URL or `null`).
For each subsequent call, pass that URL as `nextPageUrl`, keeping `isActive`,
`searchTerm` and `pageSize` unchanged. Increment `page` if desired: with a cursor
it is a display label, not an instruction to replay previous pages.

```json
{"items": [], "summary": {"returned": 0, "page": 3, "pageSize": 200, "hasMore": false, "nextPageUrl": null}, "error": "No companies found"}
```

Stop only when `nextPageUrl` is `null`. A full page may be final; an empty
page may still have a continuation. Empty results retain the legacy `error`
text field for compatibility, but are not MCP errors. Consumers must use the
pagination metadata rather than that text or the item count to detect the end.
HTTP failures, invalid responses and missing pagination metadata are MCP errors;
do not treat a partially collected catalogue as complete. Consumers should reject
repeated cursors across calls and restart an expired query explicitly, not silently
append a restarted first page. Pagination is not a snapshot of concurrently changing data.

Legacy `page > 1` calls without a cursor replay the upstream chain from page 1,
returning only the requested page. Use the cursor for linear catalogue traversal.
Defaults remain page 1 and page size 25; sizes must be integers from 1 to 200.
The existing generic `Client.Query` and other search tools are unchanged.

Continuations use POST with the original filter body and `MaxRecords`. Only URLs
on the configured API origin and the Companies next-page endpoint are accepted;
Autotask HTTP redirects are rejected to protect credentials. Configure the final
zone API URL directly. No new credentials or configuration settings are required.

Source: [Autotask Advanced query features — Paginating results](https://www.autotask.net/help/developerhelp/content/apis/REST/API_Calls/REST_Advanced_Query_Features.htm),
verified 2026-11-17. Forward traversal is not limited to the 50-page history used
for backward navigation.

### Impersonation (`ImpersonationResourceId`)
- **Behavior:** autotask-mcp sends `ImpersonationResourceId: <resource id>` on create/update/delete when the gateway actor carries `autotask_resource_id` (Autotask docs: REST security and authentication).
- **Prerequisites (Autotask admin):** The API user's security level must allow impersonation for the entity type, the impersonated resource's security level must allow impersonation, and the resource itself needs permission for the action. Time entries are additionally subject to proxy time-entry security.
- **Open point:** The docs do not state whether a disallowed impersonation fails the request or falls back to "API User". Verify on a test object before relying on it; the server does not retry without the header (no silent loss of attribution).

## IT Glue

### Document Listing -- Root vs. Folder Documents
- **Issue:** The documents list endpoint only returns root-level documents by default. Documents nested in folders are not included.
- **Workaround:** Make two API calls (root + folders) and deduplicate results.

### HTML Content in Document Sections
- **Issue:** Section content is stored as HTML, which wastes tokens when sent to LLMs.
- **Workaround:** Strip HTML to plaintext before returning via MCP tools.

### Configuration Interfaces -- Nested Endpoint
- **Issue:** `/configuration_interfaces` is not a top-level endpoint. It requires a configuration ID in the path.
- **Path:** `/configurations/{id}/relationships/configuration_interfaces`
- **Status:** Fixed in tool implementation. The `configuration_id` parameter is required.

## Datto RMM

### OAuth2 Token Refresh
- **Issue:** Tokens expire and concurrent requests can trigger multiple refresh calls.
- **Workaround:** Use singleflight pattern to deduplicate concurrent token refresh requests. Refresh 5 minutes before expiry.

### Platform Regions
- **Issue:** 6 different API base URLs depending on platform region.
- **Config:** `DATTO_PLATFORM` env var selects the correct base URL (pinotage, merlot, concord, vidal, zinfandel, syrah).

## RocketCyber

### EU vs US Region
- **Issue:** Separate API endpoints for EU and US regions.
- **Config:** `ROCKETCYBER_REGION` env var (`us` or `eu`) selects the correct base URL.
- **EU:** `https://api-eu.rocketcyber.com/v3`
- **US:** `https://api-us.rocketcyber.com/v3`

## Datto Unified Continuity (BCDR + SaaS + DTC)

### Inconsistent Pagination Parameter Names
- **Issue:** Most endpoints use `page`/`perPage`, but `/bcdr/agent` and `/report/activity-log` use `_page`/`_perPage` (with underscore prefix).
- **Workaround:** Separate pagination helpers (`addPaginationParams` vs `addUnderscorePaginationParams`) are used depending on the endpoint.

### API Scope
- **Note:** Despite being commonly called the "BCDR API", the Datto Unified Continuity API at `api.datto.com/v1` covers four product categories:
  - BCDR appliances (devices, agents, assets, shares, alerts)
  - SaaS Protection (M365/Google Workspace domains, seats, applications)
  - Direct-to-Cloud (DTC assets, RMM templates, storage pools)
  - Reporting (activity log)

## Datto EDR (Infocyte)

### LoopBack 3 API with Mixed Filter Formats
- **Issue:** Most endpoints accept bracket-notation filters (`filter[limit]=5`), but Rules, SuppressionRules, and Extensions require a single JSON string parameter (`filter={"limit":5}`).
- **Workaround:** `BuildJSONFilter()` function generates the JSON string format for those endpoints. Other endpoints use standard bracket notation via `addLoopBackPagination()`.

### Authentication via Query Parameter
- **Issue:** Authentication uses `access_token` as a query parameter, not a Bearer header.
- **Config:** `DATTO_EDR_API_KEY` is appended as `?access_token=...` to every request.

### Instance-Specific Base URL
- **Issue:** Each Datto EDR tenant has its own subdomain (e.g. `yourorg.infocyte.com`).
- **Config:** `DATTO_EDR_BASE_URL` must be set to the full instance URL.

### Dashboard Returns Array
- **Issue:** `GET /api/dashboard` returns a JSON array, not an object. Most other endpoints return objects.
- **Workaround:** Use `GetList()` instead of `Get()` for the dashboard endpoint.

### Non-Existent Endpoints in Documentation
- **Issue:** Some endpoints mentioned in the LoopBack explorer (`/api/Scans`, `/api/Jobs`, `/api/ActivityTraces`, `/api/ScanHosts`) return 404 and are not actually available.
- **Status:** These tools have been removed from the server.

## Datto Backup

### OAuth2 Client Credentials
- **Issue:** Uses OAuth2 client credentials flow, not API key auth like other Datto products.
- **Config:** Requires `DATTO_BACKUP_CLIENT_ID` and `DATTO_BACKUP_CLIENT_SECRET`.

## Datto Networking (DNA)

### Separate API from Unified Continuity
- **Issue:** Datto Networking has its own API at `api.dna.datto.com`, separate from the Unified Continuity API at `api.datto.com`. Different credentials are needed.
- **Note:** The DNA API covers both Datto Networking hardware and the legacy CloudTrax platform (rebranded to Datto Networking after acquisition).

## MyITProcess

### GraphQL-Based API
- **Issue:** MyITProcess uses a GraphQL API, unlike all other servers which use REST.
- **Workaround:** The MCP server abstracts GraphQL queries behind standard tool interfaces with pagination support.
