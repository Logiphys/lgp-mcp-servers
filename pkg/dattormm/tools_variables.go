package dattormm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerVariableTools(srv *server.MCPServer, c *Client, _ *slog.Logger) {
	// Account variables

	srv.AddTool(
		mcp.NewTool("create_account_variable",
			mcp.WithDescription("Create a new account-level variable in Datto RMM."),
			mcp.WithString("name",
				mcp.Description("The name of the variable."),
				mcp.Required(),
			),
			mcp.WithString("value",
				mcp.Description("The value of the variable."),
				mcp.Required(),
			),
			mcp.WithBoolean("masked",
				mcp.Description("Whether the variable value should be masked (hidden) in the UI."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := req.GetString("name", "")
			value := req.GetString("value", "")
			if name == "" {
				return mcputil.ErrorResult(fmt.Errorf("name is required")), nil
			}
			if value == "" {
				return mcputil.ErrorResult(fmt.Errorf("value is required")), nil
			}
			args := req.GetArguments()
			masked, _ := args["masked"].(bool)
			body := map[string]any{
				"name":   name,
				"value":  value,
				"masked": masked,
			}
			if _, err := c.Put(ctx, "/account/variable", body); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult("Account variable created."), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("update_account_variable",
			mcp.WithDescription("Update an existing account-level variable in Datto RMM."),
			mcp.WithString("variableId",
				mcp.Description("The ID of the variable to update."),
				mcp.Required(),
			),
			mcp.WithString("name",
				mcp.Description("The new name of the variable."),
			),
			mcp.WithString("value",
				mcp.Description("The new value of the variable."),
			),
			mcp.WithBoolean("masked",
				mcp.Description("Whether the variable value should be masked (hidden) in the UI."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			variableId := req.GetString("variableId", "")
			if variableId == "" {
				return mcputil.ErrorResult(fmt.Errorf("variableId is required")), nil
			}
			args := req.GetArguments()
			body := map[string]any{}
			if v, ok := args["name"].(string); ok && v != "" {
				body["name"] = v
			}
			if v, ok := args["value"].(string); ok && v != "" {
				body["value"] = v
			}
			if v, ok := args["masked"].(bool); ok {
				body["masked"] = v
			}
			result, err := c.Post(ctx, fmt.Sprintf("/account/variable/%s", variableId), body)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("delete_account_variable",
			mcp.WithDescription("Delete an account-level variable in Datto RMM."),
			mcp.WithString("variableId",
				mcp.Description("The ID of the variable to delete."),
				mcp.Required(),
			),
			mcp.WithDestructiveHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			variableId := req.GetString("variableId", "")
			if variableId == "" {
				return mcputil.ErrorResult(fmt.Errorf("variableId is required")), nil
			}
			if err := c.Delete(ctx, fmt.Sprintf("/account/variable/%s", variableId)); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult("Account variable deleted."), nil
		},
	)

	// Site variables

	srv.AddTool(
		mcp.NewTool("create_site_variable",
			mcp.WithDescription("Create a new site-level variable in Datto RMM."),
			mcp.WithString("siteUid",
				mcp.Description("The UID of the site."),
				mcp.Required(),
			),
			mcp.WithString("name",
				mcp.Description("The name of the variable."),
				mcp.Required(),
			),
			mcp.WithString("value",
				mcp.Description("The value of the variable."),
				mcp.Required(),
			),
			mcp.WithBoolean("masked",
				mcp.Description("Whether the variable value should be masked (hidden) in the UI."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			siteUid := req.GetString("siteUid", "")
			name := req.GetString("name", "")
			value := req.GetString("value", "")
			if siteUid == "" {
				return mcputil.ErrorResult(fmt.Errorf("siteUid is required")), nil
			}
			if name == "" {
				return mcputil.ErrorResult(fmt.Errorf("name is required")), nil
			}
			if value == "" {
				return mcputil.ErrorResult(fmt.Errorf("value is required")), nil
			}
			args := req.GetArguments()
			masked, _ := args["masked"].(bool)
			body := map[string]any{
				"name":   name,
				"value":  value,
				"masked": masked,
			}
			if _, err := c.Put(ctx, fmt.Sprintf("/site/%s/variable", siteUid), body); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult("Site variable created."), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("update_site_variable",
			mcp.WithDescription("Update an existing site-level variable in Datto RMM."),
			mcp.WithString("siteUid",
				mcp.Description("The UID of the site."),
				mcp.Required(),
			),
			mcp.WithString("variableId",
				mcp.Description("The ID of the variable to update."),
				mcp.Required(),
			),
			mcp.WithString("name",
				mcp.Description("The new name of the variable."),
			),
			mcp.WithString("value",
				mcp.Description("The new value of the variable."),
			),
			mcp.WithBoolean("masked",
				mcp.Description("Whether the variable value should be masked (hidden) in the UI."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			siteUid := req.GetString("siteUid", "")
			variableId := req.GetString("variableId", "")
			if siteUid == "" {
				return mcputil.ErrorResult(fmt.Errorf("siteUid is required")), nil
			}
			if variableId == "" {
				return mcputil.ErrorResult(fmt.Errorf("variableId is required")), nil
			}
			args := req.GetArguments()
			body := map[string]any{}
			if v, ok := args["name"].(string); ok && v != "" {
				body["name"] = v
			}
			if v, ok := args["value"].(string); ok && v != "" {
				body["value"] = v
			}
			if v, ok := args["masked"].(bool); ok {
				body["masked"] = v
			}
			result, err := c.Post(ctx, fmt.Sprintf("/site/%s/variable/%s", siteUid, variableId), body)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("delete_site_variable",
			mcp.WithDescription("Delete a site-level variable in Datto RMM."),
			mcp.WithString("siteUid",
				mcp.Description("The UID of the site."),
				mcp.Required(),
			),
			mcp.WithString("variableId",
				mcp.Description("The ID of the variable to delete."),
				mcp.Required(),
			),
			mcp.WithDestructiveHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			siteUid := req.GetString("siteUid", "")
			variableId := req.GetString("variableId", "")
			if siteUid == "" {
				return mcputil.ErrorResult(fmt.Errorf("siteUid is required")), nil
			}
			if variableId == "" {
				return mcputil.ErrorResult(fmt.Errorf("variableId is required")), nil
			}
			if err := c.Delete(ctx, fmt.Sprintf("/site/%s/variable/%s", siteUid, variableId)); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult("Site variable deleted."), nil
		},
	)

	// Site proxy

	srv.AddTool(
		mcp.NewTool("update_site_proxy",
			mcp.WithDescription("Set or update the proxy configuration for a Datto RMM site."),
			mcp.WithString("siteUid",
				mcp.Description("The UID of the site."),
				mcp.Required(),
			),
			mcp.WithString("type",
				mcp.Description("The proxy type: http, socks4, or socks5."),
				mcp.Required(),
			),
			mcp.WithString("host",
				mcp.Description("The proxy host address."),
				mcp.Required(),
			),
			mcp.WithNumber("port",
				mcp.Description("The proxy port number."),
				mcp.Required(),
			),
			mcp.WithString("username",
				mcp.Description("The proxy authentication username (optional)."),
			),
			mcp.WithString("password",
				mcp.Description("The proxy authentication password (optional)."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			siteUid := req.GetString("siteUid", "")
			proxyType := req.GetString("type", "")
			host := req.GetString("host", "")
			if siteUid == "" {
				return mcputil.ErrorResult(fmt.Errorf("siteUid is required")), nil
			}
			if proxyType == "" {
				return mcputil.ErrorResult(fmt.Errorf("type is required")), nil
			}
			if host == "" {
				return mcputil.ErrorResult(fmt.Errorf("host is required")), nil
			}
			port := req.GetInt("port", 0)
			if port == 0 {
				return mcputil.ErrorResult(fmt.Errorf("port is required")), nil
			}
			args := req.GetArguments()
			body := map[string]any{
				"type": proxyType,
				"host": host,
				"port": port,
			}
			if v, ok := args["username"].(string); ok && v != "" {
				body["username"] = v
			}
			if v, ok := args["password"].(string); ok && v != "" {
				body["password"] = v
			}
			if _, err := c.Put(ctx, fmt.Sprintf("/site/%s/settings/proxy", siteUid), body); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult("Site proxy updated."), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("delete_site_proxy",
			mcp.WithDescription("Remove the proxy configuration for a Datto RMM site."),
			mcp.WithString("siteUid",
				mcp.Description("The UID of the site."),
				mcp.Required(),
			),
			mcp.WithDestructiveHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			siteUid := req.GetString("siteUid", "")
			if siteUid == "" {
				return mcputil.ErrorResult(fmt.Errorf("siteUid is required")), nil
			}
			if err := c.Delete(ctx, fmt.Sprintf("/site/%s/settings/proxy", siteUid)); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult("Site proxy deleted."), nil
		},
	)
}
