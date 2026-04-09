package main

import (
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp-servers/pkg/dattoedr"
	"github.com/Logiphys/lgp-mcp-servers/pkg/config"
	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
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

	srv := server.NewMCPServer("datto-edr-mcp", version)

	dattoedr.RegisterTools(srv, client, logger)
	mcputil.RegisterServerInfoTool(srv, mcputil.ServerInfo{Name: "datto-edr-mcp", Version: version, BuildDate: buildDate})

	logger.Info("starting datto-edr-mcp", "version", version)

	if err := server.ServeStdio(srv); err != nil {
		logger.Error("serve error", "err", err)
		os.Exit(1)
	}
}
