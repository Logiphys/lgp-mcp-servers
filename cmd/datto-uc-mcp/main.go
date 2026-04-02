package main

import (
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp-servers/pkg/dattouc"
	"github.com/Logiphys/lgp-mcp-servers/pkg/config"
	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

var version = "dev"
var buildDate = ""

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: config.LogLevel(),
	}))

	cfg := dattouc.Config{
		PublicKey: config.MustEnv("DATTO_UC_PUBLIC_KEY"),
		SecretKey: config.MustEnv("DATTO_UC_SECRET_KEY"),
		BaseURL:   config.OptEnv("DATTO_UC_BASE_URL", ""),
	}

	client := dattouc.NewClient(cfg, logger)

	srv := server.NewMCPServer("datto-uc-mcp", version)

	tier := config.AccessTier("DATTO_UC_ACCESS_TIER")
	dattouc.RegisterTools(srv, client, logger, tier)
	mcputil.RegisterServerInfoTool(srv, mcputil.ServerInfo{Name: "datto-uc-mcp", Version: version, BuildDate: buildDate, Prefix: "datto_uc", AccessTier: tier})

	if err := server.ServeStdio(srv); err != nil {
		logger.Error("serve error", "err", err)
		os.Exit(1)
	}
}
