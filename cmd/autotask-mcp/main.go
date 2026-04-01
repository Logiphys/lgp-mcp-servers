package main

import (
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp/internal/autotask"
	"github.com/Logiphys/lgp-mcp/pkg/config"
)

var version = "dev"

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

	autotask.RegisterTools(srv, client, picklist, logger)

	if err := server.ServeStdio(srv); err != nil {
		logger.Error("serve error", "err", err)
		os.Exit(1)
	}
}
