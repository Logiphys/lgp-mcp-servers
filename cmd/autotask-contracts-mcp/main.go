package main

import (
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp-servers/pkg/autotask"
	"github.com/Logiphys/lgp-mcp-servers/pkg/autotaskcontracts"
	"github.com/Logiphys/lgp-mcp-servers/pkg/config"
	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

var version = "dev"
var buildDate = ""

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: config.LogLevel()}))
	client := autotask.NewClient(autotask.Config{
		Username:        config.MustEnv("AUTOTASK_USERNAME"),
		Secret:          config.MustEnv("AUTOTASK_SECRET"),
		IntegrationCode: config.MustEnv("AUTOTASK_INTEGRATION_CODE"),
		BaseURL:         config.OptEnv("AUTOTASK_BASE_URL", ""),
	}, logger)
	const name = "autotask-contracts-mcp"
	srv := server.NewMCPServer(name, version)
	autotaskcontracts.RegisterTools(srv, client)
	mcputil.RegisterServerInfoTool(srv, mcputil.ServerInfo{Name: name, Version: version, BuildDate: buildDate})
	if err := server.ServeStdio(srv); err != nil {
		logger.Error("serve error", "err", err)
		os.Exit(1)
	}
}
