package main

import (
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp-servers/pkg/config"
	"github.com/Logiphys/lgp-mcp-servers/pkg/dattormm"
	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

var version = "dev"
var buildDate = ""

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: config.LogLevel(),
	}))

	cfg := dattormm.Config{
		APIKey:    config.MustEnv("DATTO_API_KEY"),
		APISecret: config.MustEnv("DATTO_API_SECRET"),
		Platform:  config.OptEnv("DATTO_PLATFORM", "merlot"),
		BaseURL:   config.OptEnv("DATTO_BASE_URL", ""),
	}

	client := dattormm.NewClient(cfg, logger)

	srv := server.NewMCPServer("datto-rmm-mcp", version)

	dattormm.RegisterTools(srv, client, logger)
	mcputil.RegisterServerInfoTool(srv, mcputil.ServerInfo{Name: "datto-rmm-mcp", Version: version, BuildDate: buildDate})

	if err := server.ServeStdio(srv); err != nil {
		logger.Error("serve error", "err", err)
		os.Exit(1)
	}
}
