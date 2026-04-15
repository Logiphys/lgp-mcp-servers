package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp-servers/pkg/autotask"
	"github.com/Logiphys/lgp-mcp-servers/pkg/config"
	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

var version = "dev"
var buildDate = ""

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: config.LogLevel(),
	}))

	username := config.MustEnv("AUTOTASK_USERNAME")
	var baseURL, webURL string

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	logger.Info("discovering Autotask zone", "user", username)
	zi, err := autotask.DiscoverZone(ctx, username)
	if err != nil {
		logger.Error("zone discovery failed", "err", err)
		os.Exit(1)
	}
	baseURL = zi.BaseURL
	webURL = zi.WebURL
	logger.Info("discovered Autotask zone", "baseURL", baseURL, "webURL", webURL)

	cfg := autotask.Config{
		Username:        username,
		Secret:          config.MustEnv("AUTOTASK_SECRET"),
		IntegrationCode: config.MustEnv("AUTOTASK_INTEGRATION_CODE"),
		BaseURL:         baseURL,
		WebURL:          webURL,
	}

	client := autotask.NewClient(cfg, logger)

	picklist := autotask.NewPicklistCache(client, logger)

	srv := server.NewMCPServer("autotask-mcp", version)

	autotask.RegisterTools(srv, client, picklist, logger)
	mcputil.RegisterServerInfoTool(srv, mcputil.ServerInfo{Name: "autotask-mcp", Version: version, BuildDate: buildDate})

	if err := server.ServeStdio(srv); err != nil {
		logger.Error("serve error", "err", err)
		os.Exit(1)
	}
}
