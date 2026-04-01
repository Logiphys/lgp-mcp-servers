# Known API Quirks & Workarounds

## Autotask PSA

### `search_resources` → HTTP 500
- **Issue:** `search_resources` consistently returns HTTP 500 from the Autotask API.
- **Workaround:** Use `list` endpoint with client-side filtering instead.
- **Status:** Persistent, not retryable.

### `search_invoices` with companyID → Inconsistent Results
- **Issue:** Filtering invoices by `companyID` returns incomplete or inconsistent data.
- **Workaround:** Use `search_billing_items` instead, which provides reliable company-level billing data.

### Pagination on Large Result Sets
- **Issue:** Default pagination can time out or return errors on very large collections.
- **Workaround:** Use `postedAfter` date filter to narrow the result set before paginating.

## IT Glue

### Document Listing — Root vs. Folder Documents
- **Issue:** The documents list endpoint only returns root-level documents by default. Documents nested in folders are not included.
- **Workaround:** Make two API calls (root + folders) and deduplicate results.

### HTML Content in Document Sections
- **Issue:** Section content is stored as HTML, which wastes tokens when sent to LLMs.
- **Workaround:** Strip HTML to plaintext before returning via MCP tools.

## Datto RMM

### OAuth2 Token Refresh
- **Issue:** Tokens expire and concurrent requests can trigger multiple refresh calls.
- **Workaround:** Use singleflight pattern to deduplicate concurrent token refresh requests. Refresh 5 minutes before expiry.

### Platform Regions
- **Issue:** 6 different API base URLs depending on platform region.
- **Config:** `DATTO_PLATFORM` env var selects the correct base URL (pinotage, merlot, concord, vidal, zinfandel, syrah).

## RocketCyber

No known quirks at this time. Simplest API of the four.
