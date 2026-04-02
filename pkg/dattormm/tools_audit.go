package dattormm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerAuditTools(srv *server.MCPServer, c *Client, _ *slog.Logger, tier int) {
	// Tier 1
	srv.AddTool(
		mcp.NewTool("datto_get_esxi_audit",
			mcp.WithDescription("Retrieve the ESXi audit data for a Datto RMM device."),
			mcp.WithString("deviceUid",
				mcp.Description("The UID of the device."),
				mcp.Required(),
			),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			deviceUid := req.GetString("deviceUid", "")
			if deviceUid == "" {
				return mcputil.ErrorResult(fmt.Errorf("deviceUid is required")), nil
			}
			result, err := c.Get(ctx, fmt.Sprintf("/device/%s/audit/esxi", deviceUid), nil)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("datto_get_printer_audit",
			mcp.WithDescription("Retrieve the printer audit data for a Datto RMM device."),
			mcp.WithString("deviceUid",
				mcp.Description("The UID of the device."),
				mcp.Required(),
			),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			deviceUid := req.GetString("deviceUid", "")
			if deviceUid == "" {
				return mcputil.ErrorResult(fmt.Errorf("deviceUid is required")), nil
			}
			result, err := c.Get(ctx, fmt.Sprintf("/device/%s/audit/printer", deviceUid), nil)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	// Tier 2
	if tier >= 2 {

	srv.AddTool(
		mcp.NewTool("datto_get_device_software",
			mcp.WithDescription("List the installed software on a Datto RMM device, with pagination."),
			mcp.WithString("deviceUid",
				mcp.Description("The UID of the device."),
				mcp.Required(),
			),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			deviceUid := req.GetString("deviceUid", "")
			if deviceUid == "" {
				return mcputil.ErrorResult(fmt.Errorf("deviceUid is required")), nil
			}
			params := paginationParams(req)
			items, pageInfo, err := c.GetList(ctx, fmt.Sprintf("/device/%s/software", deviceUid), params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return listResult(items, pageInfo), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("datto_get_device_audit_by_mac",
			mcp.WithDescription("Retrieve the audit data for a Datto RMM device identified by its MAC address."),
			mcp.WithString("macAddress",
				mcp.Description("The MAC address of the device (e.g. 00:11:22:33:44:55)."),
				mcp.Required(),
			),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			macAddress := req.GetString("macAddress", "")
			if macAddress == "" {
				return mcputil.ErrorResult(fmt.Errorf("macAddress is required")), nil
			}
			result, err := c.Get(ctx, fmt.Sprintf("/device/mac/%s/audit", macAddress), nil)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	} // end tier 2
}
