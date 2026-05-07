package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp-servers/pkg/config"
	"github.com/Logiphys/lgp-mcp-servers/pkg/logiphysci"
	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

var (
	version   = "dev"
	buildDate = ""
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: config.LogLevel(),
	}))

	skillDir := config.OptEnv("LOGIPHYS_CI_SKILL_DIR", defaultSkillDir())
	pythonBin := config.OptEnv("LOGIPHYS_CI_PYTHON_BIN", "python3")
	outputDir := config.OptEnv("LOGIPHYS_CI_OUTPUT_DIR", os.TempDir())

	runner := logiphysci.NewRunner(logiphysci.RunnerConfig{
		SkillDir:  skillDir,
		PythonBin: pythonBin,
		OutputDir: outputDir,
		Logger:    logger,
	})

	srv := server.NewMCPServer("logiphys-ci-mcp", version)
	logiphysci.RegisterTools(srv, runner, logger)
	mcputil.RegisterServerInfoTool(srv, mcputil.ServerInfo{
		Name:      "logiphys-ci-mcp",
		Version:   version,
		BuildDate: buildDate,
	})

	if err := server.ServeStdio(srv); err != nil {
		logger.Error("serve error", "err", err)
		os.Exit(1)
	}
}

// defaultSkillDir resolves the skill directory.
//
// Order:
//  1. <bin>/../skills/logiphys-ci  (production deploy: /opt/lgp-mcp-gateway/skills/...)
//  2. external/logiphys-marketplace/plugins/lgp-docs/skills/logiphys-ci  (local dev via submodule)
func defaultSkillDir() string {
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "..", "skills", "logiphys-ci")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "external/logiphys-marketplace/plugins/lgp-docs/skills/logiphys-ci"
}
