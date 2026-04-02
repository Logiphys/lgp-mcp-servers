package autotask

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	toolHandlersMu sync.RWMutex
	toolHandlers   = make(map[string]server.ToolHandlerFunc)
)

// RegisterHandler stores a tool handler for later dispatch by autotask_execute_tool.
func RegisterHandler(name string, handler server.ToolHandlerFunc) {
	toolHandlersMu.Lock()
	defer toolHandlersMu.Unlock()
	toolHandlers[name] = handler
}

// addTool registers a tool on the MCP server and also stores its handler for meta-dispatch.
func addTool(srv *server.MCPServer, tool mcp.Tool, handler server.ToolHandlerFunc) {
	RegisterHandler(tool.Name, handler)
	srv.AddTool(tool, handler)
}

// ExecuteTool dispatches to a registered tool handler by name.
func ExecuteTool(ctx context.Context, toolName string, arguments map[string]any) (*mcp.CallToolResult, error) {
	toolHandlersMu.RLock()
	handler, ok := toolHandlers[toolName]
	toolHandlersMu.RUnlock()
	if !ok {
		return mcputil.ErrorResult(fmt.Errorf("unknown tool: %s", toolName)), nil
	}
	req := mcp.CallToolRequest{}
	req.Params.Name = toolName
	req.Params.Arguments = arguments
	return handler(ctx, req)
}

// ToolCategories maps category names to tool names.
var ToolCategories = map[string][]string{
	"utility":            {"autotask_test_connection", "autotask_list_queues", "autotask_list_ticket_statuses", "autotask_list_ticket_priorities", "autotask_get_field_info"},
	"companies":          {"autotask_search_companies", "autotask_create_company", "autotask_update_company"},
	"contacts":           {"autotask_search_contacts", "autotask_create_contact"},
	"tickets":            {"autotask_search_tickets", "autotask_get_ticket_details", "autotask_create_ticket", "autotask_update_ticket"},
	"ticket_charges":     {"autotask_get_ticket_charge", "autotask_search_ticket_charges", "autotask_create_ticket_charge", "autotask_update_ticket_charge", "autotask_delete_ticket_charge"},
	"ticket_notes":       {"autotask_get_ticket_note", "autotask_search_ticket_notes", "autotask_create_ticket_note"},
	"ticket_attachments": {"autotask_get_ticket_attachment", "autotask_search_ticket_attachments"},
	"projects":           {"autotask_search_projects", "autotask_create_project", "autotask_search_tasks", "autotask_create_task", "autotask_list_phases", "autotask_create_phase"},
	"project_notes":      {"autotask_get_project_note", "autotask_search_project_notes", "autotask_create_project_note"},
	"company_notes":      {"autotask_get_company_note", "autotask_search_company_notes", "autotask_create_company_note"},
	"time_billing":       {"autotask_create_time_entry", "autotask_search_time_entries", "autotask_search_billing_items", "autotask_get_billing_item", "autotask_search_billing_item_approval_levels"},
	"expenses":           {"autotask_get_expense_report", "autotask_search_expense_reports", "autotask_create_expense_report", "autotask_create_expense_item"},
	"financial":          {"autotask_get_quote", "autotask_search_quotes", "autotask_create_quote", "autotask_get_quote_item", "autotask_search_quote_items", "autotask_create_quote_item", "autotask_update_quote_item", "autotask_delete_quote_item", "autotask_get_opportunity", "autotask_search_opportunities", "autotask_create_opportunity", "autotask_search_invoices", "autotask_search_contracts"},
	"resources":          {"autotask_search_resources"},
	"configuration":      {"autotask_search_configuration_items"},
	"service_calls":      {"autotask_search_service_calls", "autotask_get_service_call", "autotask_create_service_call", "autotask_update_service_call", "autotask_delete_service_call", "autotask_search_service_call_tickets", "autotask_create_service_call_ticket", "autotask_delete_service_call_ticket", "autotask_search_service_call_ticket_resources", "autotask_create_service_call_ticket_resource", "autotask_delete_service_call_ticket_resource"},
	"products":           {"autotask_get_product", "autotask_search_products", "autotask_get_service", "autotask_search_services", "autotask_get_service_bundle", "autotask_search_service_bundles"},
}

// RouteIntent suggests a tool based on natural language intent.
func RouteIntent(intent string) map[string]any {
	lower := strings.ToLower(intent)

	type route struct {
		keywords []string
		action   string
		tool     string
		desc     string
	}

	routes := []route{
		{[]string{"ticket", "create"}, "create", "autotask_create_ticket", "Create a new ticket"},
		{[]string{"ticket", "new"}, "create", "autotask_create_ticket", "Create a new ticket"},
		{[]string{"ticket", "update"}, "update", "autotask_update_ticket", "Update an existing ticket"},
		{[]string{"ticket", "note"}, "note", "autotask_create_ticket_note", "Create a ticket note"},
		{[]string{"ticket", "charge"}, "charge", "autotask_search_ticket_charges", "Search ticket charges"},
		{[]string{"ticket", "time"}, "time", "autotask_create_time_entry", "Create a time entry for a ticket"},
		{[]string{"ticket"}, "search", "autotask_search_tickets", "Search tickets"},

		{[]string{"company", "create"}, "create", "autotask_create_company", "Create a new company"},
		{[]string{"company", "new"}, "create", "autotask_create_company", "Create a new company"},
		{[]string{"company", "note"}, "note", "autotask_create_company_note", "Create a company note"},
		{[]string{"company"}, "search", "autotask_search_companies", "Search companies"},

		{[]string{"contact", "create"}, "create", "autotask_create_contact", "Create a new contact"},
		{[]string{"contact", "new"}, "create", "autotask_create_contact", "Create a new contact"},
		{[]string{"contact"}, "search", "autotask_search_contacts", "Search contacts"},

		{[]string{"project", "create"}, "create", "autotask_create_project", "Create a new project"},
		{[]string{"project", "new"}, "create", "autotask_create_project", "Create a new project"},
		{[]string{"project", "note"}, "note", "autotask_create_project_note", "Create a project note"},
		{[]string{"project"}, "search", "autotask_search_projects", "Search projects"},

		{[]string{"task", "create"}, "create", "autotask_create_task", "Create a new task"},
		{[]string{"task", "new"}, "create", "autotask_create_task", "Create a new task"},
		{[]string{"task"}, "search", "autotask_search_tasks", "Search tasks"},

		{[]string{"time", "entry"}, "time", "autotask_create_time_entry", "Create a time entry"},
		{[]string{"time"}, "search", "autotask_search_time_entries", "Search time entries"},

		{[]string{"billing"}, "search", "autotask_search_billing_items", "Search billing items"},

		{[]string{"quote", "create"}, "create", "autotask_create_quote", "Create a new quote"},
		{[]string{"quote", "item"}, "item", "autotask_search_quote_items", "Search quote items"},
		{[]string{"quote"}, "search", "autotask_search_quotes", "Search quotes"},

		{[]string{"opportunity", "create"}, "create", "autotask_create_opportunity", "Create a new opportunity"},
		{[]string{"opportunity"}, "search", "autotask_search_opportunities", "Search opportunities"},

		{[]string{"invoice"}, "search", "autotask_search_invoices", "Search invoices"},
		{[]string{"contract"}, "search", "autotask_search_contracts", "Search contracts"},

		{[]string{"resource"}, "search", "autotask_search_resources", "Search resources"},

		{[]string{"expense"}, "search", "autotask_search_expense_reports", "Search expense reports"},

		{[]string{"service call", "create"}, "create", "autotask_create_service_call", "Create a service call"},
		{[]string{"service call"}, "search", "autotask_search_service_calls", "Search service calls"},

		{[]string{"configuration", "item"}, "search", "autotask_search_configuration_items", "Search configuration items"},
		{[]string{"config"}, "search", "autotask_search_configuration_items", "Search configuration items"},

		{[]string{"product"}, "search", "autotask_search_products", "Search products"},
		{[]string{"service bundle"}, "search", "autotask_search_service_bundles", "Search service bundles"},
		{[]string{"service"}, "search", "autotask_search_services", "Search services"},

		{[]string{"phase"}, "list", "autotask_list_phases", "List project phases"},
		{[]string{"queue"}, "list", "autotask_list_queues", "List queues"},
		{[]string{"status"}, "list", "autotask_list_ticket_statuses", "List ticket statuses"},
		{[]string{"priority"}, "list", "autotask_list_ticket_priorities", "List ticket priorities"},
		{[]string{"field"}, "info", "autotask_get_field_info", "Get field info for an entity"},
	}

	for _, r := range routes {
		match := true
		for _, kw := range r.keywords {
			if !strings.Contains(lower, kw) {
				match = false
				break
			}
		}
		if match {
			return map[string]any{
				"suggested_tool": r.tool,
				"description":    r.desc,
			}
		}
	}

	return map[string]any{
		"suggested_tool": "autotask_list_categories",
		"description":    "List available tool categories to explore",
	}
}

// registerMetaTools registers the meta-tools on the MCP server.
func registerMetaTools(srv *server.MCPServer, _ *Client, _ *slog.Logger) {
	// autotask_list_categories
	addTool(srv,
		mcp.NewTool("autotask_list_categories",
			mcp.WithDescription("List all available Autotask tool categories with tool counts"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			categories := make(map[string]any)
			for name, tools := range ToolCategories {
				categories[name] = map[string]any{
					"count": len(tools),
				}
			}
			return mcputil.JSONResult(categories), nil
		},
	)

	// autotask_list_category_tools
	addTool(srv,
		mcp.NewTool("autotask_list_category_tools",
			mcp.WithDescription("List all tools in a specific category"),
			mcp.WithString("category", mcp.Description("Category name (e.g. tickets, companies, financial)"), mcp.Required()),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			category := req.GetString("category", "")
			if category == "" {
				return mcputil.ErrorResult(fmt.Errorf("category is required")), nil
			}
			tools, ok := ToolCategories[category]
			if !ok {
				available := make([]string, 0, len(ToolCategories))
				for k := range ToolCategories {
					available = append(available, k)
				}
				return mcputil.ErrorResult(fmt.Errorf("unknown category %q, available: %s", category, strings.Join(available, ", "))), nil
			}
			return mcputil.JSONResult(map[string]any{
				"category": category,
				"tools":    tools,
			}), nil
		},
	)

	// autotask_execute_tool
	addTool(srv,
		mcp.NewTool("autotask_execute_tool",
			mcp.WithDescription("Execute another Autotask tool by name with given arguments"),
			mcp.WithString("toolName", mcp.Description("Name of the tool to execute"), mcp.Required()),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			toolName := req.GetString("toolName", "")
			if toolName == "" {
				return mcputil.ErrorResult(fmt.Errorf("toolName is required")), nil
			}
			args := req.GetArguments()
			// Remove our own params, pass the rest
			arguments := make(map[string]any)
			for k, v := range args {
				if k != "toolName" {
					arguments[k] = v
				}
			}
			return ExecuteTool(ctx, toolName, arguments)
		},
	)

	// autotask_router
	addTool(srv,
		mcp.NewTool("autotask_router",
			mcp.WithDescription("Route a natural language intent to the most appropriate Autotask tool"),
			mcp.WithString("intent", mcp.Description("Natural language description of what you want to do"), mcp.Required()),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			intent := req.GetString("intent", "")
			if intent == "" {
				return mcputil.ErrorResult(fmt.Errorf("intent is required")), nil
			}
			result := RouteIntent(intent)
			b, _ := json.MarshalIndent(result, "", "  ")
			return mcputil.TextResult(string(b)), nil
		},
	)
}
