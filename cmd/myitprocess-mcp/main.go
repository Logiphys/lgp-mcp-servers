package main

import (
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp-servers/internal/myitprocess"
	"github.com/Logiphys/lgp-mcp-servers/pkg/config"
	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

var version = "dev"
var buildDate = ""

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: config.LogLevel(),
	}))

	cfg := myitprocess.Config{
		APIKey: config.MustEnv("MYITPROCESS_API_KEY"),
	}

	tier := config.AccessTier("MYITPROCESS_ACCESS_TIER")
	client := myitprocess.NewClient(cfg, logger)

	srv := server.NewMCPServer("myitprocess-mcp", version)

	myitprocess.RegisterTools(srv, client, logger, tier)
	mcputil.RegisterServerInfoTool(srv, mcputil.ServerInfo{Name: "myitprocess-mcp", Version: version, BuildDate: buildDate, Prefix: "myitprocess", AccessTier: tier})

	if err := server.ServeStdio(srv); err != nil {
		logger.Error("serve error", "err", err)
		os.Exit(1)
	}
}
