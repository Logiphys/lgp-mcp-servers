package main

import (
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp/internal/myitprocess"
	"github.com/Logiphys/lgp-mcp/pkg/config"
	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
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

	client := myitprocess.NewClient(cfg, logger)

	srv := server.NewMCPServer("myitprocess-mcp", version)

	myitprocess.RegisterTools(srv, client, logger)
	mcputil.RegisterServerInfoTool(srv, mcputil.ServerInfo{Name: "myitprocess-mcp", Version: version, BuildDate: buildDate, Prefix: "myitprocess"})

	if err := server.ServeStdio(srv); err != nil {
		logger.Error("serve error", "err", err)
		os.Exit(1)
	}
}
