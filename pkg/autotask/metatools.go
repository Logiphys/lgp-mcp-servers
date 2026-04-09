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
	"utility":            {"test_connection", "list_queues", "list_ticket_statuses", "list_ticket_priorities", "get_field_info"},
	"companies":          {"search_companies", "create_company", "update_company"},
	"contacts":           {"search_contacts", "create_contact"},
	"tickets":            {"search_tickets", "get_ticket_details", "create_ticket", "update_ticket"},
	"ticket_charges":     {"get_ticket_charge", "search_ticket_charges", "create_ticket_charge", "update_ticket_charge", "delete_ticket_charge"},
	"ticket_notes":       {"get_ticket_note", "search_ticket_notes", "create_ticket_note"},
	"ticket_attachments": {"get_ticket_attachment", "search_ticket_attachments"},
	"projects":           {"search_projects", "create_project", "search_tasks", "create_task", "list_phases", "create_phase"},
	"project_notes":      {"get_project_note", "search_project_notes", "create_project_note"},
	"company_notes":      {"get_company_note", "search_company_notes", "create_company_note"},
	"time_billing":       {"create_time_entry", "search_time_entries", "search_billing_items", "get_billing_item", "search_billing_item_approval_levels"},
	"expenses":           {"get_expense_report", "search_expense_reports", "create_expense_report", "create_expense_item"},
	"financial":          {"get_quote", "search_quotes", "create_quote", "get_quote_item", "search_quote_items", "create_quote_item", "update_quote_item", "delete_quote_item", "get_opportunity", "search_opportunities", "create_opportunity", "search_invoices", "search_contracts"},
	"resources":          {"search_resources"},
	"configuration":      {"search_configuration_items"},
	"service_calls":      {"search_service_calls", "get_service_call", "create_service_call", "update_service_call", "delete_service_call", "search_service_call_tickets", "create_service_call_ticket", "delete_service_call_ticket", "search_service_call_ticket_resources", "create_service_call_ticket_resource", "delete_service_call_ticket_resource"},
	"products":           {"get_product", "search_products", "get_service", "search_services", "get_service_bundle", "search_service_bundles"},
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
		{[]string{"ticket", "create"}, "create", "create_ticket", "Create a new ticket"},
		{[]string{"ticket", "new"}, "create", "create_ticket", "Create a new ticket"},
		{[]string{"ticket", "update"}, "update", "update_ticket", "Update an existing ticket"},
		{[]string{"ticket", "note"}, "note", "create_ticket_note", "Create a ticket note"},
		{[]string{"ticket", "charge"}, "charge", "search_ticket_charges", "Search ticket charges"},
		{[]string{"ticket", "time"}, "time", "create_time_entry", "Create a time entry for a ticket"},
		{[]string{"ticket"}, "search", "search_tickets", "Search tickets"},

		{[]string{"company", "create"}, "create", "create_company", "Create a new company"},
		{[]string{"company", "new"}, "create", "create_company", "Create a new company"},
		{[]string{"company", "note"}, "note", "create_company_note", "Create a company note"},
		{[]string{"company"}, "search", "search_companies", "Search companies"},

		{[]string{"contact", "create"}, "create", "create_contact", "Create a new contact"},
		{[]string{"contact", "new"}, "create", "create_contact", "Create a new contact"},
		{[]string{"contact"}, "search", "search_contacts", "Search contacts"},

		{[]string{"project", "create"}, "create", "create_project", "Create a new project"},
		{[]string{"project", "new"}, "create", "create_project", "Create a new project"},
		{[]string{"project", "note"}, "note", "create_project_note", "Create a project note"},
		{[]string{"project"}, "search", "search_projects", "Search projects"},

		{[]string{"task", "create"}, "create", "create_task", "Create a new task"},
		{[]string{"task", "new"}, "create", "create_task", "Create a new task"},
		{[]string{"task"}, "search", "search_tasks", "Search tasks"},

		{[]string{"time", "entry"}, "time", "create_time_entry", "Create a time entry"},
		{[]string{"time"}, "search", "search_time_entries", "Search time entries"},

		{[]string{"billing"}, "search", "search_billing_items", "Search billing items"},

		{[]string{"quote", "create"}, "create", "create_quote", "Create a new quote"},
		{[]string{"quote", "item"}, "item", "search_quote_items", "Search quote items"},
		{[]string{"quote"}, "search", "search_quotes", "Search quotes"},

		{[]string{"opportunity", "create"}, "create", "create_opportunity", "Create a new opportunity"},
		{[]string{"opportunity"}, "search", "search_opportunities", "Search opportunities"},

		{[]string{"invoice"}, "search", "search_invoices", "Search invoices"},
		{[]string{"contract"}, "search", "search_contracts", "Search contracts"},

		{[]string{"resource"}, "search", "search_resources", "Search resources"},

		{[]string{"expense"}, "search", "search_expense_reports", "Search expense reports"},

		{[]string{"service call", "create"}, "create", "create_service_call", "Create a service call"},
		{[]string{"service call"}, "search", "search_service_calls", "Search service calls"},

		{[]string{"configuration", "item"}, "search", "search_configuration_items", "Search configuration items"},
		{[]string{"config"}, "search", "search_configuration_items", "Search configuration items"},

		{[]string{"product"}, "search", "search_products", "Search products"},
		{[]string{"service bundle"}, "search", "search_service_bundles", "Search service bundles"},
		{[]string{"service"}, "search", "search_services", "Search services"},

		{[]string{"phase"}, "list", "list_phases", "List project phases"},
		{[]string{"queue"}, "list", "list_queues", "List queues"},
		{[]string{"status"}, "list", "list_ticket_statuses", "List ticket statuses"},
		{[]string{"priority"}, "list", "list_ticket_priorities", "List ticket priorities"},
		{[]string{"field"}, "info", "get_field_info", "Get field info for an entity"},
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
		"suggested_tool": "list_categories",
		"description":    "List available tool categories to explore",
	}
}

// registerMetaTools registers the meta-tools on the MCP server.
func registerMetaTools(srv *server.MCPServer, _ *Client, _ *slog.Logger) {
	// autotask_list_categories
	addTool(srv,
		mcp.NewTool("list_categories",
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
		mcp.NewTool("list_category_tools",
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
		mcp.NewTool("execute_tool",
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
		mcp.NewTool("router",
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
