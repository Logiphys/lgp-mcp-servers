package dattonetwork

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

// RegisterTools registers all Datto Networking MCP tools on the given server.
func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	registerTestConnection(srv, client, logger)
	registerListDevices(srv, client, logger)
	registerGetDevice(srv, client, logger)
	registerGetDevicesOverview(srv, client, logger)
	registerGetResellerOverview(srv, client, logger)
	registerGetRouter(srv, client, logger)
	registerGetWhoami(srv, client, logger)
	registerGetUserDevices(srv, client, logger)
	registerGetDeviceClientsOverview(srv, client, logger)
	registerGetDeviceClientsUsage(srv, client, logger)
	registerGetDeviceWanUsage(srv, client, logger)
	registerGetDeviceApplications(srv, client, logger)
}

// --- helpers ----------------------------------------------------------------

func buildListResult(items []any) *mcp.CallToolResult {
	result := map[string]any{"data": items, "count": len(items)}
	return mcputil.JSONResult(result)
}

// --- tool registrations -----------------------------------------------------

func registerTestConnection(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("test_connection",
		mcp.WithDescription("Test connectivity to the Datto Networking (DNA) API. Returns success or an error message."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := client.TestConnection(ctx); err != nil {
			logger.ErrorContext(ctx, "test connection failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.TextResult("Datto Networking API connection successful"), nil
	})
}

func registerGetWhoami(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("get_whoami",
		mcp.WithDescription("Get the current authenticated user information from the Datto Networking API."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := client.Get(ctx, "/whoami", nil)
		if err != nil {
			logger.ErrorContext(ctx, "get whoami failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

func registerListDevices(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("list_devices",
		mcp.WithDescription("List all Datto Networking devices. Returns an array of MAC addresses."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		items, err := client.GetList(ctx, "/devices/list", nil)
		if err != nil {
			logger.ErrorContext(ctx, "list devices failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerGetDevicesOverview(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("get_devices_overview",
		mcp.WithDescription("Get an overview of all Datto Networking devices in the fleet."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		items, err := client.GetList(ctx, "/devices/overview", nil)
		if err != nil {
			logger.ErrorContext(ctx, "get devices overview failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerGetDevice(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("get_device",
		mcp.WithDescription("Get comprehensive information about a specific Datto Networking device by MAC address. MAC format: 12 uppercase hex chars, no delimiters (e.g., AABBCCDDEEFF)."),
		mcp.WithString("mac", mcp.Description("Device MAC address (12 uppercase hex chars, e.g., AABBCCDDEEFF)"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mac := req.GetString("mac", "")
		if mac == "" {
			return mcputil.ErrorResult(fmt.Errorf("mac is required")), nil
		}

		path := fmt.Sprintf("/devices/%s", mac)
		result, err := client.Get(ctx, path, nil)
		if err != nil {
			logger.ErrorContext(ctx, "get device failed", "mac", mac, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

func registerGetRouter(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("get_router",
		mcp.WithDescription("Get router information for a specific Datto Networking device by MAC address. MAC format: 12 uppercase hex chars, no delimiters (e.g., AABBCCDDEEFF)."),
		mcp.WithString("mac", mcp.Description("Router MAC address (12 uppercase hex chars, e.g., AABBCCDDEEFF)"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mac := req.GetString("mac", "")
		if mac == "" {
			return mcputil.ErrorResult(fmt.Errorf("mac is required")), nil
		}

		path := fmt.Sprintf("/routers/%s", mac)
		result, err := client.Get(ctx, path, nil)
		if err != nil {
			logger.ErrorContext(ctx, "get router failed", "mac", mac, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

func registerGetDeviceClientsOverview(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("get_device_clients_overview",
		mcp.WithDescription("Get an overview of clients connected to a specific Datto Networking device. MAC format: 12 uppercase hex chars, no delimiters (e.g., AABBCCDDEEFF). Optional window parameter for time range (e.g., '1d', '3h', '1w')."),
		mcp.WithString("mac", mcp.Description("Device MAC address (12 uppercase hex chars, e.g., AABBCCDDEEFF)"), mcp.Required()),
		mcp.WithString("window", mcp.Description("Time window for data (e.g., '1d', '3h', '1w')")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mac := req.GetString("mac", "")
		if mac == "" {
			return mcputil.ErrorResult(fmt.Errorf("mac is required")), nil
		}

		path := fmt.Sprintf("/devices/%s/clients/overview", mac)
		params := make(map[string]string)
		if v := req.GetString("window", ""); v != "" {
			params["window"] = v
		}

		result, err := client.Get(ctx, path, params)
		if err != nil {
			logger.ErrorContext(ctx, "get device clients overview failed", "mac", mac, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

func registerGetDeviceClientsUsage(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("get_device_clients_usage",
		mcp.WithDescription("Get usage data for clients connected to a specific Datto Networking device. MAC format: 12 uppercase hex chars, no delimiters (e.g., AABBCCDDEEFF). Optional window, order, and limit parameters."),
		mcp.WithString("mac", mcp.Description("Device MAC address (12 uppercase hex chars, e.g., AABBCCDDEEFF)"), mcp.Required()),
		mcp.WithString("window", mcp.Description("Time window for data (e.g., '1d', '3h', '1w')")),
		mcp.WithString("order", mcp.Description("Sort order: 'asc' or 'desc'")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results to return")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mac := req.GetString("mac", "")
		if mac == "" {
			return mcputil.ErrorResult(fmt.Errorf("mac is required")), nil
		}

		path := fmt.Sprintf("/devices/%s/clients/usage", mac)
		params := make(map[string]string)
		if v := req.GetString("window", ""); v != "" {
			params["window"] = v
		}
		if v := req.GetString("order", ""); v != "" {
			params["order"] = v
		}
		if v := req.GetInt("limit", 0); v > 0 {
			params["limit"] = strconv.Itoa(v)
		}

		items, err := client.GetList(ctx, path, params)
		if err != nil {
			logger.ErrorContext(ctx, "get device clients usage failed", "mac", mac, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerGetDeviceWanUsage(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("get_device_wan_usage",
		mcp.WithDescription("Get WAN usage data for a specific Datto Networking device. MAC format: 12 uppercase hex chars, no delimiters (e.g., AABBCCDDEEFF). Optional window parameter for time range."),
		mcp.WithString("mac", mcp.Description("Device MAC address (12 uppercase hex chars, e.g., AABBCCDDEEFF)"), mcp.Required()),
		mcp.WithString("window", mcp.Description("Time window for data (e.g., '1d', '3h', '1w')")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mac := req.GetString("mac", "")
		if mac == "" {
			return mcputil.ErrorResult(fmt.Errorf("mac is required")), nil
		}

		path := fmt.Sprintf("/devices/%s/wans/usage", mac)
		params := make(map[string]string)
		if v := req.GetString("window", ""); v != "" {
			params["window"] = v
		}

		items, err := client.GetList(ctx, path, params)
		if err != nil {
			logger.ErrorContext(ctx, "get device wan usage failed", "mac", mac, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerGetDeviceApplications(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("get_device_applications",
		mcp.WithDescription("Get application usage data for a specific Datto Networking device. MAC format: 12 uppercase hex chars, no delimiters (e.g., AABBCCDDEEFF). Optional window and limit parameters."),
		mcp.WithString("mac", mcp.Description("Device MAC address (12 uppercase hex chars, e.g., AABBCCDDEEFF)"), mcp.Required()),
		mcp.WithString("window", mcp.Description("Time window for data (e.g., '1d', '3h', '1w')")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results to return")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mac := req.GetString("mac", "")
		if mac == "" {
			return mcputil.ErrorResult(fmt.Errorf("mac is required")), nil
		}

		path := fmt.Sprintf("/devices/%s/applications", mac)
		params := make(map[string]string)
		if v := req.GetString("window", ""); v != "" {
			params["window"] = v
		}
		if v := req.GetInt("limit", 0); v > 0 {
			params["limit"] = strconv.Itoa(v)
		}

		items, err := client.GetList(ctx, path, params)
		if err != nil {
			logger.ErrorContext(ctx, "get device applications failed", "mac", mac, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerGetUserDevices(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("get_user_devices",
		mcp.WithDescription("List devices accessible to a specific SSO user in Datto Networking."),
		mcp.WithString("username", mcp.Description("The SSO username to look up devices for"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		username := req.GetString("username", "")
		if username == "" {
			return mcputil.ErrorResult(fmt.Errorf("username is required")), nil
		}

		path := fmt.Sprintf("/users/%s/devices", url.PathEscape(username))
		items, err := client.GetList(ctx, path, nil)
		if err != nil {
			logger.ErrorContext(ctx, "get user devices failed", "username", username, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerGetResellerOverview(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("get_reseller_overview",
		mcp.WithDescription("Get a network overview for all devices belonging to a reseller in Datto Networking."),
		mcp.WithString("resellerId", mcp.Description("The reseller ID to get the network overview for"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resellerId := req.GetString("resellerId", "")
		if resellerId == "" {
			return mcputil.ErrorResult(fmt.Errorf("resellerId is required")), nil
		}

		path := fmt.Sprintf("/users/%s/overview", url.PathEscape(resellerId))
		result, err := client.Get(ctx, path, nil)
		if err != nil {
			logger.ErrorContext(ctx, "get reseller overview failed", "resellerId", resellerId, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}
