package autotask

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerProductTools(srv *server.MCPServer, client *Client, _ *slog.Logger) {
	// === PRODUCTS ===

	// autotask_get_product
	addTool(srv,
		mcp.NewTool("autotask_get_product",
			mcp.WithDescription("Get a specific product by ID"),
			mcp.WithNumber("id", mcp.Description("Product ID"), mcp.Required()),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("id", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
			}
			item, err := client.Get(ctx, "Products", id)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if item == nil {
				return mcputil.TextResult(FormatNotFound("product", map[string]any{"id": id})), nil
			}
			return mcputil.TextResult(FormatGetResult(item)), nil
		},
	)

	// autotask_search_products
	addTool(srv,
		mcp.NewTool("autotask_search_products",
			mcp.WithDescription("Search for products in Autotask"),
			mcp.WithString("searchTerm", mcp.Description("Search by product name")),
			mcp.WithNumber("productCategory", mcp.Description("Filter by product category ID")),
			mcp.WithBoolean("isActive", mcp.Description("Filter by active status")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(500)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			if term := req.GetString("searchTerm", ""); term != "" {
				filters = append(filters, Filter{Op: "contains", Field: "name", Value: term})
			}
			if v, ok := args["productCategory"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "productCategory", Value: v})
			}
			if v, ok := args["isActive"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "isActive", Value: v})
			}
			items, err := client.Query(ctx, "Products", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 25),
				MaxSize:  500,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("products", map[string]any{"searchTerm": req.GetString("searchTerm", "")})), nil
			}
			return mcputil.TextResult(FormatSearchResult("autotask_search_products", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
		},
	)

	// === SERVICES ===

	// autotask_get_service
	addTool(srv,
		mcp.NewTool("autotask_get_service",
			mcp.WithDescription("Get a specific service by ID"),
			mcp.WithNumber("id", mcp.Description("Service ID"), mcp.Required()),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("id", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
			}
			item, err := client.Get(ctx, "Services", id)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if item == nil {
				return mcputil.TextResult(FormatNotFound("service", map[string]any{"id": id})), nil
			}
			return mcputil.TextResult(FormatGetResult(item)), nil
		},
	)

	// autotask_search_services
	addTool(srv,
		mcp.NewTool("autotask_search_services",
			mcp.WithDescription("Search for services in Autotask"),
			mcp.WithString("searchTerm", mcp.Description("Search by service name")),
			mcp.WithBoolean("isActive", mcp.Description("Filter by active status")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(500)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			if term := req.GetString("searchTerm", ""); term != "" {
				filters = append(filters, Filter{Op: "contains", Field: "name", Value: term})
			}
			if v, ok := args["isActive"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "isActive", Value: v})
			}
			items, err := client.Query(ctx, "Services", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 25),
				MaxSize:  500,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("services", map[string]any{"searchTerm": req.GetString("searchTerm", "")})), nil
			}
			return mcputil.TextResult(FormatSearchResult("autotask_search_services", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
		},
	)

	// === SERVICE BUNDLES ===

	// autotask_get_service_bundle
	addTool(srv,
		mcp.NewTool("autotask_get_service_bundle",
			mcp.WithDescription("Get a specific service bundle by ID"),
			mcp.WithNumber("id", mcp.Description("Service bundle ID"), mcp.Required()),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("id", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
			}
			item, err := client.Get(ctx, "ServiceBundles", id)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if item == nil {
				return mcputil.TextResult(FormatNotFound("service bundle", map[string]any{"id": id})), nil
			}
			return mcputil.TextResult(FormatGetResult(item)), nil
		},
	)

	// autotask_search_service_bundles
	addTool(srv,
		mcp.NewTool("autotask_search_service_bundles",
			mcp.WithDescription("Search for service bundles in Autotask"),
			mcp.WithString("searchTerm", mcp.Description("Search by service bundle name")),
			mcp.WithBoolean("isActive", mcp.Description("Filter by active status")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(500)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			if term := req.GetString("searchTerm", ""); term != "" {
				filters = append(filters, Filter{Op: "contains", Field: "name", Value: term})
			}
			if v, ok := args["isActive"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "isActive", Value: v})
			}
			items, err := client.Query(ctx, "ServiceBundles", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 25),
				MaxSize:  500,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("service bundles", map[string]any{"searchTerm": req.GetString("searchTerm", "")})), nil
			}
			return mcputil.TextResult(FormatSearchResult("autotask_search_service_bundles", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
		},
	)

	_ = server.ToolHandlerFunc(nil)
}
