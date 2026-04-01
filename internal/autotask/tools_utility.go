package autotask

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerUtilityTools(srv *server.MCPServer, client *Client, picklist *PicklistCache, _ *slog.Logger) {
	// autotask_test_connection
	addTool(srv,
		mcp.NewTool("autotask_test_connection",
			mcp.WithDescription("Test the connection to Autotask API"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if err := client.TestConnection(ctx); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(`{"message": "Connection to Autotask API successful"}`), nil
		},
	)

	// autotask_list_queues
	addTool(srv,
		mcp.NewTool("autotask_list_queues",
			mcp.WithDescription("List all available ticket queues"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			values, err := picklist.GetQueues(ctx)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatPicklistValues("Queues", values)), nil
		},
	)

	// autotask_list_ticket_statuses
	addTool(srv,
		mcp.NewTool("autotask_list_ticket_statuses",
			mcp.WithDescription("List all available ticket statuses"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			values, err := picklist.GetTicketStatuses(ctx)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatPicklistValues("Ticket Statuses", values)), nil
		},
	)

	// autotask_list_ticket_priorities
	addTool(srv,
		mcp.NewTool("autotask_list_ticket_priorities",
			mcp.WithDescription("List all available ticket priorities"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			values, err := picklist.GetTicketPriorities(ctx)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatPicklistValues("Ticket Priorities", values)), nil
		},
	)

	// autotask_get_field_info
	addTool(srv,
		mcp.NewTool("autotask_get_field_info",
			mcp.WithDescription("Get field definitions for an Autotask entity type"),
			mcp.WithString("entityType", mcp.Description("Entity type (e.g. Tickets, Companies, Contacts)"), mcp.Required()),
			mcp.WithString("fieldName", mcp.Description("Optional: filter to a specific field name")),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			entityType := req.GetString("entityType", "")
			if entityType == "" {
				return mcputil.ErrorResult(fmt.Errorf("entityType is required")), nil
			}
			normalized := NormalizeEntityType(entityType)
			fields, err := picklist.GetFields(ctx, normalized)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}

			fieldName := req.GetString("fieldName", "")
			if fieldName != "" {
				for _, f := range fields {
					if strings.EqualFold(f.Name, fieldName) {
						b, _ := json.MarshalIndent(f, "", "  ")
						return mcputil.TextResult(string(b)), nil
					}
				}
				return mcputil.TextResult(FormatNotFound("field", map[string]any{"entityType": normalized, "fieldName": fieldName})), nil
			}

			b, _ := json.MarshalIndent(fields, "", "  ")
			return mcputil.TextResult(string(b)), nil
		},
	)

	// Suppress unused import warnings
	_ = server.ToolHandlerFunc(nil)
}
