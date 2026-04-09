package main

import (
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp-servers/pkg/config"
	"github.com/Logiphys/lgp-mcp-servers/pkg/dattobackup"
	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

var version = "dev"
var buildDate = ""

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: config.LogLevel(),
	}))

	cfg := dattobackup.Config{
		ClientID:     config.MustEnv("DATTO_BACKUP_CLIENT_ID"),
		ClientSecret: config.MustEnv("DATTO_BACKUP_CLIENT_SECRET"),
		BaseURL:      config.OptEnv("DATTO_BACKUP_BASE_URL", ""),
	}

	client := dattobackup.NewClient(cfg, logger)

	srv := server.NewMCPServer("datto-backup-mcp", version)

	dattobackup.RegisterTools(srv, client, logger)
	mcputil.RegisterServerInfoTool(srv, mcputil.ServerInfo{Name: "datto-backup-mcp", Version: version, BuildDate: buildDate})

	if err := server.ServeStdio(srv); err != nil {
		logger.Error("serve error", "err", err)
		os.Exit(1)
	}
}
