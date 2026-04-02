package main

import (
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp/internal/dattonetwork"
	"github.com/Logiphys/lgp-mcp/pkg/config"
	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
)

var version = "dev"
var buildDate = ""

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: config.LogLevel(),
	}))

	cfg := dattonetwork.Config{
		PublicKey: config.MustEnv("DATTO_NETWORK_PUBLIC_KEY"),
		SecretKey: config.MustEnv("DATTO_NETWORK_SECRET_KEY"),
		BaseURL:   config.OptEnv("DATTO_NETWORK_BASE_URL", ""),
	}

	client := dattonetwork.NewClient(cfg, logger)
	tier := config.AccessTier("DATTO_NETWORK_ACCESS_TIER")

	srv := server.NewMCPServer("datto-network-mcp", version)

	dattonetwork.RegisterTools(srv, client, logger, tier)
	mcputil.RegisterServerInfoTool(srv, mcputil.ServerInfo{Name: "datto-network-mcp", Version: version, BuildDate: buildDate, Prefix: "datto_network", AccessTier: tier})

	if err := server.ServeStdio(srv); err != nil {
		logger.Error("serve error", "err", err)
		os.Exit(1)
	}
}
