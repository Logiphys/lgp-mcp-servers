package dattormm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerDeviceTools(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	// datto_get_device
	srv.AddTool(
		mcp.NewTool("datto_get_device",
			mcp.WithDescription("Get details for a specific Datto RMM device by UID."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("deviceUid",
				mcp.Description("The UID of the device."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			deviceUid := req.GetString("deviceUid", "")
			result, err := client.Get(ctx, fmt.Sprintf("/device/%s", deviceUid), nil)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	// datto_get_device_by_id
	srv.AddTool(
		mcp.NewTool("datto_get_device_by_id",
			mcp.WithDescription("Get details for a specific Datto RMM device by numeric ID."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithNumber("deviceId",
				mcp.Description("The numeric ID of the device."),
				mcp.Required(),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			deviceId := req.GetInt("deviceId", 0)
			result, err := client.Get(ctx, fmt.Sprintf("/device/%d", deviceId), nil)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	// datto_get_device_by_mac
	srv.AddTool(
		mcp.NewTool("datto_get_device_by_mac",
			mcp.WithDescription("Get details for a Datto RMM device by MAC address."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("macAddress",
				mcp.Description("The MAC address of the device (e.g. 00:11:22:33:44:55)."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			macAddress := req.GetString("macAddress", "")
			result, err := client.Get(ctx, fmt.Sprintf("/device/mac/%s", macAddress), nil)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	// datto_list_device_open_alerts
	srv.AddTool(
		mcp.NewTool("datto_list_device_open_alerts",
			mcp.WithDescription("List open alerts for a specific Datto RMM device."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("deviceUid",
				mcp.Description("The UID of the device."),
				mcp.Required(),
			),
			mcp.WithBoolean("muted",
				mcp.Description("Filter alerts by muted status."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination."),
				mcp.Min(1),
			),
			mcp.WithNumber("max",
				mcp.Description("Maximum number of results per page."),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			deviceUid := req.GetString("deviceUid", "")
			params := paginationParams(req)
			args := req.GetArguments()
			if v, ok := args["muted"]; ok {
				if b, ok := v.(bool); ok {
					if b {
						params["muted"] = "true"
					} else {
						params["muted"] = "false"
					}
				}
			}
			items, pageInfo, err := client.GetList(ctx, fmt.Sprintf("/device/%s/alerts/open", deviceUid), params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return listResult(items, pageInfo), nil
		},
	)

	// datto_list_device_resolved_alerts
	srv.AddTool(
		mcp.NewTool("datto_list_device_resolved_alerts",
			mcp.WithDescription("List resolved alerts for a specific Datto RMM device."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("deviceUid",
				mcp.Description("The UID of the device."),
				mcp.Required(),
			),
			mcp.WithBoolean("muted",
				mcp.Description("Filter alerts by muted status."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination."),
				mcp.Min(1),
			),
			mcp.WithNumber("max",
				mcp.Description("Maximum number of results per page."),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			deviceUid := req.GetString("deviceUid", "")
			params := paginationParams(req)
			args := req.GetArguments()
			if v, ok := args["muted"]; ok {
				if b, ok := v.(bool); ok {
					if b {
						params["muted"] = "true"
					} else {
						params["muted"] = "false"
					}
				}
			}
			items, pageInfo, err := client.GetList(ctx, fmt.Sprintf("/device/%s/alerts/resolved", deviceUid), params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return listResult(items, pageInfo), nil
		},
	)

	// datto_move_device
	srv.AddTool(
		mcp.NewTool("datto_move_device",
			mcp.WithDescription("Move a Datto RMM device to a different site."),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithString("deviceUid",
				mcp.Description("The UID of the device to move."),
				mcp.Required(),
			),
			mcp.WithString("siteUid",
				mcp.Description("The UID of the destination site."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			deviceUid := req.GetString("deviceUid", "")
			siteUid := req.GetString("siteUid", "")
			if err := client.Put(ctx, fmt.Sprintf("/device/%s/site/%s", deviceUid, siteUid), nil); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult("Device moved successfully."), nil
		},
	)

	// datto_create_quick_job
	srv.AddTool(
		mcp.NewTool("datto_create_quick_job",
			mcp.WithDescription("Create a quick job on a Datto RMM device to run a component."),
			mcp.WithString("deviceUid",
				mcp.Description("The UID of the target device."),
				mcp.Required(),
			),
			mcp.WithString("jobName",
				mcp.Description("Name of the quick job."),
				mcp.Required(),
			),
			mcp.WithString("componentUid",
				mcp.Description("The UID of the component to run."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			deviceUid := req.GetString("deviceUid", "")
			body := map[string]any{
				"jobName":      req.GetString("jobName", ""),
				"componentUid": req.GetString("componentUid", ""),
			}
			args := req.GetArguments()
			if v, ok := args["variables"]; ok {
				body["variables"] = v
			}
			result, err := client.Post(ctx, fmt.Sprintf("/device/%s/quickjob", deviceUid), body)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	// datto_set_device_udf
	srv.AddTool(
		mcp.NewTool("datto_set_device_udf",
			mcp.WithDescription("Set user-defined fields (UDFs) on a Datto RMM device. Provide any of udf1 through udf30."),
			mcp.WithString("deviceUid",
				mcp.Description("The UID of the device."),
				mcp.Required(),
			),
			mcp.WithString("udf1", mcp.Description("User-defined field 1.")),
			mcp.WithString("udf2", mcp.Description("User-defined field 2.")),
			mcp.WithString("udf3", mcp.Description("User-defined field 3.")),
			mcp.WithString("udf4", mcp.Description("User-defined field 4.")),
			mcp.WithString("udf5", mcp.Description("User-defined field 5.")),
			mcp.WithString("udf6", mcp.Description("User-defined field 6.")),
			mcp.WithString("udf7", mcp.Description("User-defined field 7.")),
			mcp.WithString("udf8", mcp.Description("User-defined field 8.")),
			mcp.WithString("udf9", mcp.Description("User-defined field 9.")),
			mcp.WithString("udf10", mcp.Description("User-defined field 10.")),
			mcp.WithString("udf11", mcp.Description("User-defined field 11.")),
			mcp.WithString("udf12", mcp.Description("User-defined field 12.")),
			mcp.WithString("udf13", mcp.Description("User-defined field 13.")),
			mcp.WithString("udf14", mcp.Description("User-defined field 14.")),
			mcp.WithString("udf15", mcp.Description("User-defined field 15.")),
			mcp.WithString("udf16", mcp.Description("User-defined field 16.")),
			mcp.WithString("udf17", mcp.Description("User-defined field 17.")),
			mcp.WithString("udf18", mcp.Description("User-defined field 18.")),
			mcp.WithString("udf19", mcp.Description("User-defined field 19.")),
			mcp.WithString("udf20", mcp.Description("User-defined field 20.")),
			mcp.WithString("udf21", mcp.Description("User-defined field 21.")),
			mcp.WithString("udf22", mcp.Description("User-defined field 22.")),
			mcp.WithString("udf23", mcp.Description("User-defined field 23.")),
			mcp.WithString("udf24", mcp.Description("User-defined field 24.")),
			mcp.WithString("udf25", mcp.Description("User-defined field 25.")),
			mcp.WithString("udf26", mcp.Description("User-defined field 26.")),
			mcp.WithString("udf27", mcp.Description("User-defined field 27.")),
			mcp.WithString("udf28", mcp.Description("User-defined field 28.")),
			mcp.WithString("udf29", mcp.Description("User-defined field 29.")),
			mcp.WithString("udf30", mcp.Description("User-defined field 30.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			deviceUid := req.GetString("deviceUid", "")
			body := map[string]any{}
			udfFields := []string{
				"udf1", "udf2", "udf3", "udf4", "udf5",
				"udf6", "udf7", "udf8", "udf9", "udf10",
				"udf11", "udf12", "udf13", "udf14", "udf15",
				"udf16", "udf17", "udf18", "udf19", "udf20",
				"udf21", "udf22", "udf23", "udf24", "udf25",
				"udf26", "udf27", "udf28", "udf29", "udf30",
			}
			args := req.GetArguments()
			for _, field := range udfFields {
				if v, ok := args[field]; ok {
					body[field] = v
				}
			}
			result, err := client.Post(ctx, fmt.Sprintf("/device/%s/udf", deviceUid), body)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	// datto_set_device_warranty
	srv.AddTool(
		mcp.NewTool("datto_set_device_warranty",
			mcp.WithDescription("Set or clear the warranty date for a Datto RMM device."),
			mcp.WithString("deviceUid",
				mcp.Description("The UID of the device."),
				mcp.Required(),
			),
			mcp.WithString("warrantyDate",
				mcp.Description("Warranty expiry date in YYYY-MM-DD format. Leave empty to clear the warranty date."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			deviceUid := req.GetString("deviceUid", "")
			body := map[string]any{
				"warrantyDate": req.GetString("warrantyDate", ""),
			}
			result, err := client.Post(ctx, fmt.Sprintf("/device/%s/warranty", deviceUid), body)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	// datto_get_device_audit
	srv.AddTool(
		mcp.NewTool("datto_get_device_audit",
			mcp.WithDescription("Get audit information for a specific Datto RMM device."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("deviceUid",
				mcp.Description("The UID of the device."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			deviceUid := req.GetString("deviceUid", "")
			result, err := client.Get(ctx, fmt.Sprintf("/device/%s/audit", deviceUid), nil)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	_ = logger
}
