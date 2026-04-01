package main

import (
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp/internal/itglue"
	"github.com/Logiphys/lgp-mcp/pkg/config"
)

var version = "dev"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: config.LogLevel(),
	}))

	cfg := itglue.Config{
		APIKey:  config.MustEnv("ITGLUE_API_KEY"),
		Region:  config.OptEnv("ITGLUE_REGION", "us"),
		BaseURL: config.OptEnv("ITGLUE_BASE_URL", ""),
	}

	client := itglue.NewClient(cfg, logger)

	srv := server.NewMCPServer("itglue-mcp", version)

	itglue.RegisterTools(srv, client, logger)

	if err := server.ServeStdio(srv); err != nil {
		logger.Error("serve error", "err", err)
		os.Exit(1)
	}
}
