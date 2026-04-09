package autotask

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerFinancialTools(srv *server.MCPServer, client *Client, _ *slog.Logger) {
	// === QUOTES ===

	// autotask_get_quote
	addTool(srv,
		mcp.NewTool("get_quote",
			mcp.WithDescription("Get a specific quote by ID"),
			mcp.WithNumber("quoteId", mcp.Description("Quote ID"), mcp.Required()),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("quoteId", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("quoteId is required")), nil
			}
			item, err := client.Get(ctx, "Quotes", id)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if item == nil {
				return mcputil.TextResult(FormatNotFound("quote", map[string]any{"id": id})), nil
			}
			return mcputil.TextResult(FormatGetResult(item)), nil
		},
	)

	// autotask_search_quotes
	addTool(srv,
		mcp.NewTool("search_quotes",
			mcp.WithDescription("Search for quotes in Autotask"),
			mcp.WithNumber("companyId", mcp.Description("Filter by company ID")),
			mcp.WithNumber("contactId", mcp.Description("Filter by contact ID")),
			mcp.WithNumber("opportunityId", mcp.Description("Filter by opportunity ID")),
			mcp.WithString("searchTerm", mcp.Description("Search by quote name")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(100)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			if v, ok := args["companyId"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "companyID", Value: v})
			}
			if v, ok := args["contactId"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "contactID", Value: v})
			}
			if v, ok := args["opportunityId"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "opportunityID", Value: v})
			}
			if term := req.GetString("searchTerm", ""); term != "" {
				filters = append(filters, Filter{Op: "contains", Field: "name", Value: term})
			}
			items, err := client.Query(ctx, "Quotes", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 25),
				MaxSize:  100,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("quotes", map[string]any{"searchTerm": req.GetString("searchTerm", "")})), nil
			}
			return mcputil.TextResult(FormatSearchResult("search_quotes", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
		},
	)

	// autotask_create_quote
	addTool(srv,
		mcp.NewTool("create_quote",
			mcp.WithDescription("Create a new quote in Autotask"),
			mcp.WithNumber("companyId", mcp.Description("Company ID"), mcp.Required()),
			mcp.WithString("name", mcp.Description("Quote name")),
			mcp.WithString("description", mcp.Description("Quote description")),
			mcp.WithNumber("contactId", mcp.Description("Contact ID")),
			mcp.WithNumber("opportunityId", mcp.Description("Opportunity ID")),
			mcp.WithString("effectiveDate", mcp.Description("Effective date (ISO 8601)")),
			mcp.WithString("expirationDate", mcp.Description("Expiration date (ISO 8601)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			data := map[string]any{
				"companyID": req.GetInt("companyId", 0),
			}
			args := req.GetArguments()
			if v, ok := args["name"]; ok {
				data["name"] = v
			}
			if v, ok := args["description"]; ok {
				data["description"] = v
			}
			if v, ok := args["contactId"]; ok {
				data["contactID"] = v
			}
			if v, ok := args["opportunityId"]; ok {
				data["opportunityID"] = v
			}
			if v, ok := args["effectiveDate"]; ok {
				data["effectiveDate"] = v
			}
			if v, ok := args["expirationDate"]; ok {
				data["expirationDate"] = v
			}
			id, err := client.Create(ctx, "Quotes", data)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatCreateResult("Quote", id)), nil
		},
	)

	// === QUOTE ITEMS ===

	// autotask_get_quote_item
	addTool(srv,
		mcp.NewTool("get_quote_item",
			mcp.WithDescription("Get a specific quote item by ID"),
			mcp.WithNumber("quoteItemId", mcp.Description("Quote item ID"), mcp.Required()),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("quoteItemId", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("quoteItemId is required")), nil
			}
			item, err := client.Get(ctx, "QuoteItems", id)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if item == nil {
				return mcputil.TextResult(FormatNotFound("quote item", map[string]any{"id": id})), nil
			}
			return mcputil.TextResult(FormatGetResult(item)), nil
		},
	)

	// autotask_search_quote_items
	addTool(srv,
		mcp.NewTool("search_quote_items",
			mcp.WithDescription("Search for quote items"),
			mcp.WithNumber("quoteId", mcp.Description("Filter by quote ID")),
			mcp.WithString("searchTerm", mcp.Description("Search by item name")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(100)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			if v, ok := args["quoteId"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "quoteID", Value: v})
			}
			if term := req.GetString("searchTerm", ""); term != "" {
				filters = append(filters, Filter{Op: "contains", Field: "name", Value: term})
			}
			items, err := client.Query(ctx, "QuoteItems", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 50),
				MaxSize:  100,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("quote items", map[string]any{"quoteId": req.GetInt("quoteId", 0)})), nil
			}
			return mcputil.TextResult(FormatSearchResult("search_quote_items", items, req.GetInt("page", 1), req.GetInt("pageSize", 50))), nil
		},
	)

	// autotask_create_quote_item
	addTool(srv,
		mcp.NewTool("create_quote_item",
			mcp.WithDescription("Create a new quote item"),
			mcp.WithNumber("quoteId", mcp.Description("Quote ID"), mcp.Required()),
			mcp.WithNumber("quantity", mcp.Description("Quantity"), mcp.Required()),
			mcp.WithString("name", mcp.Description("Item name")),
			mcp.WithString("description", mcp.Description("Item description")),
			mcp.WithNumber("unitPrice", mcp.Description("Unit price")),
			mcp.WithNumber("unitCost", mcp.Description("Unit cost")),
			mcp.WithNumber("unitDiscount", mcp.Description("Unit discount (default: 0)")),
			mcp.WithNumber("lineDiscount", mcp.Description("Line discount (default: 0)")),
			mcp.WithNumber("percentageDiscount", mcp.Description("Percentage discount (default: 0)")),
			mcp.WithBoolean("isOptional", mcp.Description("Is optional (default: false)")),
			mcp.WithNumber("serviceID", mcp.Description("Service ID")),
			mcp.WithNumber("productID", mcp.Description("Product ID")),
			mcp.WithNumber("serviceBundleID", mcp.Description("Service bundle ID")),
			mcp.WithNumber("sortOrderID", mcp.Description("Sort order ID")),
			mcp.WithNumber("quoteItemType", mcp.Description("Quote item type")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			data := map[string]any{
				"quoteID":            req.GetInt("quoteId", 0),
				"quantity":           req.GetFloat("quantity", 0),
				"unitDiscount":       0,
				"lineDiscount":       0,
				"percentageDiscount": 0,
				"isOptional":         false,
			}
			args := req.GetArguments()
			if v, ok := args["name"]; ok {
				data["name"] = v
			}
			if v, ok := args["description"]; ok {
				data["description"] = v
			}
			if v, ok := args["unitPrice"]; ok {
				data["unitPrice"] = v
			}
			if v, ok := args["unitCost"]; ok {
				data["unitCost"] = v
			}
			if v, ok := args["unitDiscount"]; ok {
				data["unitDiscount"] = v
			}
			if v, ok := args["lineDiscount"]; ok {
				data["lineDiscount"] = v
			}
			if v, ok := args["percentageDiscount"]; ok {
				data["percentageDiscount"] = v
			}
			if v, ok := args["isOptional"]; ok {
				data["isOptional"] = v
			}
			if v, ok := args["serviceID"]; ok {
				data["serviceID"] = v
			}
			if v, ok := args["productID"]; ok {
				data["productID"] = v
			}
			if v, ok := args["serviceBundleID"]; ok {
				data["serviceBundleID"] = v
			}
			if v, ok := args["sortOrderID"]; ok {
				data["sortOrderID"] = v
			}
			if v, ok := args["quoteItemType"]; ok {
				data["quoteItemType"] = v
			}
			id, err := client.Create(ctx, "QuoteItems", data)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatCreateResult("QuoteItem", id)), nil
		},
	)

	// autotask_update_quote_item
	addTool(srv,
		mcp.NewTool("update_quote_item",
			mcp.WithDescription("Update a quote item"),
			mcp.WithNumber("quoteItemId", mcp.Description("Quote item ID"), mcp.Required()),
			mcp.WithNumber("quantity", mcp.Description("Quantity")),
			mcp.WithNumber("unitPrice", mcp.Description("Unit price")),
			mcp.WithNumber("unitDiscount", mcp.Description("Unit discount")),
			mcp.WithNumber("lineDiscount", mcp.Description("Line discount")),
			mcp.WithNumber("percentageDiscount", mcp.Description("Percentage discount")),
			mcp.WithBoolean("isOptional", mcp.Description("Is optional")),
			mcp.WithNumber("sortOrderID", mcp.Description("Sort order ID")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("quoteItemId", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("quoteItemId is required")), nil
			}
			data := map[string]any{}
			args := req.GetArguments()
			if v, ok := args["quantity"]; ok {
				data["quantity"] = v
			}
			if v, ok := args["unitPrice"]; ok {
				data["unitPrice"] = v
			}
			if v, ok := args["unitDiscount"]; ok {
				data["unitDiscount"] = v
			}
			if v, ok := args["lineDiscount"]; ok {
				data["lineDiscount"] = v
			}
			if v, ok := args["percentageDiscount"]; ok {
				data["percentageDiscount"] = v
			}
			if v, ok := args["isOptional"]; ok {
				data["isOptional"] = v
			}
			if v, ok := args["sortOrderID"]; ok {
				data["sortOrderID"] = v
			}
			if err := client.Update(ctx, "QuoteItems", id, data); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatUpdateResult("QuoteItem", id)), nil
		},
	)

	// autotask_delete_quote_item
	addTool(srv,
		mcp.NewTool("delete_quote_item",
			mcp.WithDescription("Delete a quote item"),
			mcp.WithNumber("quoteId", mcp.Description("Parent quote ID"), mcp.Required()),
			mcp.WithNumber("quoteItemId", mcp.Description("Quote item ID to delete"), mcp.Required()),
			mcp.WithDestructiveHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			quoteID := req.GetInt("quoteId", 0)
			itemID := req.GetInt("quoteItemId", 0)
			if quoteID == 0 || itemID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("quoteId and quoteItemId are required")), nil
			}
			if err := client.DeleteChild(ctx, "Quotes", quoteID, "Items", itemID); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatDeleteResult("QuoteItem", itemID)), nil
		},
	)

	// === OPPORTUNITIES ===

	// autotask_get_opportunity
	addTool(srv,
		mcp.NewTool("get_opportunity",
			mcp.WithDescription("Get a specific opportunity by ID"),
			mcp.WithNumber("opportunityId", mcp.Description("Opportunity ID"), mcp.Required()),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("opportunityId", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("opportunityId is required")), nil
			}
			item, err := client.Get(ctx, "Opportunities", id)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if item == nil {
				return mcputil.TextResult(FormatNotFound("opportunity", map[string]any{"id": id})), nil
			}
			enhanced := client.EnhanceWithNames(ctx, []map[string]any{item})
			return mcputil.TextResult(FormatGetResult(enhanced[0])), nil
		},
	)

	// autotask_search_opportunities
	addTool(srv,
		mcp.NewTool("search_opportunities",
			mcp.WithDescription("Search for opportunities in Autotask"),
			mcp.WithNumber("companyId", mcp.Description("Filter by company ID")),
			mcp.WithString("searchTerm", mcp.Description("Search by opportunity title")),
			mcp.WithNumber("status", mcp.Description("Filter by status ID")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(100)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			if v, ok := args["companyId"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "companyID", Value: v})
			}
			if term := req.GetString("searchTerm", ""); term != "" {
				filters = append(filters, Filter{Op: "contains", Field: "title", Value: term})
			}
			if v, ok := args["status"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "status", Value: v})
			}
			items, err := client.Query(ctx, "Opportunities", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 25),
				MaxSize:  100,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("opportunities", map[string]any{"searchTerm": req.GetString("searchTerm", "")})), nil
			}
			items = client.EnhanceWithNames(ctx, items)
			return mcputil.TextResult(FormatSearchResult("search_opportunities", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
		},
	)

	// autotask_create_opportunity
	addTool(srv,
		mcp.NewTool("create_opportunity",
			mcp.WithDescription("Create a new opportunity in Autotask"),
			mcp.WithString("title", mcp.Description("Opportunity title"), mcp.Required()),
			mcp.WithNumber("companyId", mcp.Description("Company ID"), mcp.Required()),
			mcp.WithNumber("ownerResourceId", mcp.Description("Owner resource ID"), mcp.Required()),
			mcp.WithNumber("status", mcp.Description("Status ID"), mcp.Required()),
			mcp.WithNumber("stage", mcp.Description("Stage ID"), mcp.Required()),
			mcp.WithString("projectedCloseDate", mcp.Description("Projected close date (ISO 8601)"), mcp.Required()),
			mcp.WithString("startDate", mcp.Description("Start date (ISO 8601)"), mcp.Required()),
			mcp.WithNumber("probability", mcp.Description("Probability (default: 50)")),
			mcp.WithNumber("amount", mcp.Description("Amount (default: 0)")),
			mcp.WithNumber("cost", mcp.Description("Cost (default: 0)")),
			mcp.WithBoolean("useQuoteTotals", mcp.Description("Use quote totals (default: true)")),
			mcp.WithNumber("totalAmountMonths", mcp.Description("Total amount months")),
			mcp.WithNumber("contactId", mcp.Description("Contact ID")),
			mcp.WithString("description", mcp.Description("Description")),
			mcp.WithNumber("opportunityCategoryID", mcp.Description("Opportunity category ID")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			data := map[string]any{
				"title":              req.GetString("title", ""),
				"companyID":          req.GetInt("companyId", 0),
				"ownerResourceID":    req.GetInt("ownerResourceId", 0),
				"status":             req.GetInt("status", 0),
				"stage":              req.GetInt("stage", 0),
				"projectedCloseDate": req.GetString("projectedCloseDate", ""),
				"startDate":          req.GetString("startDate", ""),
				"probability":        50,
				"amount":             0,
				"cost":               0,
				"useQuoteTotals":     true,
			}
			args := req.GetArguments()
			if v, ok := args["probability"]; ok {
				data["probability"] = v
			}
			if v, ok := args["amount"]; ok {
				data["amount"] = v
			}
			if v, ok := args["cost"]; ok {
				data["cost"] = v
			}
			if v, ok := args["useQuoteTotals"]; ok {
				data["useQuoteTotals"] = v
			}
			if v, ok := args["totalAmountMonths"]; ok {
				data["totalAmountMonths"] = v
			}
			if v, ok := args["contactId"]; ok {
				data["contactID"] = v
			}
			if v, ok := args["description"]; ok {
				data["description"] = v
			}
			if v, ok := args["opportunityCategoryID"]; ok {
				data["opportunityCategoryID"] = v
			}
			id, err := client.Create(ctx, "Opportunities", data)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatCreateResult("Opportunity", id)), nil
		},
	)

	// === INVOICES & CONTRACTS ===

	// autotask_search_invoices
	addTool(srv,
		mcp.NewTool("search_invoices",
			mcp.WithDescription("Search for invoices in Autotask"),
			mcp.WithNumber("companyID", mcp.Description("Filter by company ID")),
			mcp.WithString("invoiceNumber", mcp.Description("Filter by exact invoice number")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(500)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			if v, ok := args["companyID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "companyID", Value: v})
			}
			if v, ok := args["invoiceNumber"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "invoiceNumber", Value: v})
			}
			items, err := client.Query(ctx, "Invoices", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 25),
				MaxSize:  500,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("invoices", map[string]any{})), nil
			}
			items = client.EnhanceWithNames(ctx, items)
			return mcputil.TextResult(FormatSearchResult("search_invoices", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
		},
	)

	// autotask_search_contracts
	addTool(srv,
		mcp.NewTool("search_contracts",
			mcp.WithDescription("Search for contracts in Autotask"),
			mcp.WithString("searchTerm", mcp.Description("Search by contract name")),
			mcp.WithNumber("companyID", mcp.Description("Filter by company ID")),
			mcp.WithNumber("status", mcp.Description("Filter by status ID")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(500)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			if term := req.GetString("searchTerm", ""); term != "" {
				filters = append(filters, Filter{Op: "contains", Field: "contractName", Value: term})
			}
			if v, ok := args["companyID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "companyID", Value: v})
			}
			if v, ok := args["status"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "status", Value: v})
			}
			items, err := client.Query(ctx, "Contracts", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 25),
				MaxSize:  500,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("contracts", map[string]any{"searchTerm": req.GetString("searchTerm", "")})), nil
			}
			items = client.EnhanceWithNames(ctx, items)
			return mcputil.TextResult(FormatSearchResult("search_contracts", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
		},
	)

	_ = server.ToolHandlerFunc(nil)
}
