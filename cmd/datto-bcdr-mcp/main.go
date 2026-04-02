package main

import (
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp/internal/dattobcdr"
	"github.com/Logiphys/lgp-mcp/pkg/config"
	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
)

var version = "dev"
var buildDate = ""

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: config.LogLevel(),
	}))

	cfg := dattobcdr.Config{
		PublicKey: config.MustEnv("DATTO_BCDR_PUBLIC_KEY"),
		SecretKey: config.MustEnv("DATTO_BCDR_SECRET_KEY"),
		BaseURL:   config.OptEnv("DATTO_BCDR_BASE_URL", ""),
	}

	client := dattobcdr.NewClient(cfg, logger)

	srv := server.NewMCPServer("datto-bcdr-mcp", version)

	dattobcdr.RegisterTools(srv, client, logger)
	mcputil.RegisterServerInfoTool(srv, mcputil.ServerInfo{Name: "datto-bcdr-mcp", Version: version, BuildDate: buildDate, Prefix: "datto_bcdr"})

	if err := server.ServeStdio(srv); err != nil {
		logger.Error("serve error", "err", err)
		os.Exit(1)
	}
}
