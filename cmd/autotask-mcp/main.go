package main

import (
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp-servers/internal/autotask"
	"github.com/Logiphys/lgp-mcp-servers/pkg/config"
	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

var version = "dev"
var buildDate = ""

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: config.LogLevel(),
	}))

	cfg := autotask.Config{
		Username:        config.MustEnv("AUTOTASK_USERNAME"),
		Secret:          config.MustEnv("AUTOTASK_SECRET"),
		IntegrationCode: config.MustEnv("AUTOTASK_INTEGRATION_CODE"),
		BaseURL:         config.OptEnv("AUTOTASK_BASE_URL", ""),
	}

	client := autotask.NewClient(cfg, logger)

	picklist := autotask.NewPicklistCache(client, logger)

	srv := server.NewMCPServer("autotask-mcp", version)

	tier := config.AccessTier("AUTOTASK_ACCESS_TIER")
	autotask.RegisterTools(srv, client, picklist, logger, tier)
	mcputil.RegisterServerInfoTool(srv, mcputil.ServerInfo{Name: "autotask-mcp", Version: version, BuildDate: buildDate, Prefix: "autotask", AccessTier: tier})

	if err := server.ServeStdio(srv); err != nil {
		logger.Error("serve error", "err", err)
		os.Exit(1)
	}
}
