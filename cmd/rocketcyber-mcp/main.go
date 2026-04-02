package main

import (
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp-servers/pkg/rocketcyber"
	"github.com/Logiphys/lgp-mcp-servers/pkg/config"
	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

var version = "dev"
var buildDate = ""

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: config.LogLevel(),
	}))

	cfg := rocketcyber.Config{
		APIKey:  config.MustEnv("ROCKETCYBER_API_KEY"),
		Region:  config.OptEnv("ROCKETCYBER_REGION", "us"),
		BaseURL: config.OptEnv("ROCKETCYBER_BASE_URL", ""),
	}

	client := rocketcyber.NewClient(cfg, logger)
	tier := config.AccessTier("ROCKETCYBER_ACCESS_TIER")

	srv := server.NewMCPServer("rocketcyber-mcp", version)

	rocketcyber.RegisterTools(srv, client, logger, tier)
	mcputil.RegisterServerInfoTool(srv, mcputil.ServerInfo{Name: "rocketcyber-mcp", Version: version, BuildDate: buildDate, Prefix: "rocketcyber", AccessTier: tier})

	if err := server.ServeStdio(srv); err != nil {
		logger.Error("serve error", "err", err)
		os.Exit(1)
	}
}
