package main

import (
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp/internal/dattoedr"
	"github.com/Logiphys/lgp-mcp/pkg/config"
	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
)

var version = "dev"
var buildDate = ""

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: config.LogLevel(),
	}))

	cfg := dattoedr.Config{
		APIKey:  config.MustEnv("DATTO_EDR_API_KEY"),
		BaseURL: config.MustEnv("DATTO_EDR_BASE_URL"),
	}

	client := dattoedr.NewClient(cfg, logger)
	tier := config.AccessTier("DATTO_EDR_ACCESS_TIER")

	srv := server.NewMCPServer("datto-edr-mcp", version)

	dattoedr.RegisterTools(srv, client, logger, tier)
	mcputil.RegisterServerInfoTool(srv, mcputil.ServerInfo{Name: "datto-edr-mcp", Version: version, BuildDate: buildDate, Prefix: "datto_edr", AccessTier: tier})

	logger.Info("starting datto-edr-mcp", "version", version, "access_tier", tier)

	if err := server.ServeStdio(srv); err != nil {
		logger.Error("serve error", "err", err)
		os.Exit(1)
	}
}
