// Package logiphysci implements the logiphys-ci MCP server: a thin wrapper
// around the Python helper scripts in the logiphys-marketplace skill that
// generates CI-conform DOCX and PDF documents.
package logiphysci

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// RunnerConfig configures a Runner.
type RunnerConfig struct {
	SkillDir  string // path to the skill root (contains scripts/, assets/)
	PythonBin string // python3 binary
	OutputDir string // where to place tempfiles + generated documents
	Logger    *slog.Logger
}

// BuildRunner executes a Python builder script and returns the result.
//
// Implementations must be safe for concurrent use across goroutines.
type BuildRunner interface {
	RunBuilder(ctx context.Context, scriptName string, payload any, ext string) (map[string]any, error)
}

// Runner is the production BuildRunner. It writes the payload to a tempfile,
// invokes the Python helper, reads the generated artifact, and returns it as
// base64-encoded content.
type Runner struct {
	cfg RunnerConfig
}

// NewRunner constructs a Runner. If Logger is nil, slog.Default() is used.
func NewRunner(cfg RunnerConfig) *Runner {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &Runner{cfg: cfg}
}

// RunBuilder writes payload as JSON to a tempfile, invokes
// `python3 <SkillDir>/scripts/<scriptName> --json <tempfile> --output <outfile>`,
// and returns {filename, mime_type, content_base64, size_bytes}.
//
// Both tempfile and outfile are removed before return.
func (r *Runner) RunBuilder(ctx context.Context, scriptName string, payload any, ext string) (map[string]any, error) {
	if err := r.ensureDirs(); err != nil {
		return nil, err
	}

	jsonFile, err := os.CreateTemp(r.cfg.OutputDir, "logiphys-ci-*.json")
	if err != nil {
		return nil, fmt.Errorf("create tempfile: %w", err)
	}
	jsonPath := jsonFile.Name()
	defer func() {
		if rmErr := os.Remove(jsonPath); rmErr != nil && !os.IsNotExist(rmErr) {
			r.cfg.Logger.Warn("tempfile cleanup failed", "path", jsonPath, "err", rmErr)
		}
	}()

	if err := json.NewEncoder(jsonFile).Encode(payload); err != nil {
		_ = jsonFile.Close()
		return nil, fmt.Errorf("encode payload: %w", err)
	}
	if err := jsonFile.Close(); err != nil {
		return nil, fmt.Errorf("close tempfile: %w", err)
	}

	outFile := filepath.Join(r.cfg.OutputDir, fmt.Sprintf("logiphys-ci-%d.%s", time.Now().UnixNano(), ext))
	defer func() {
		if rmErr := os.Remove(outFile); rmErr != nil && !os.IsNotExist(rmErr) {
			r.cfg.Logger.Warn("output cleanup failed", "path", outFile, "err", rmErr)
		}
	}()

	scriptPath := filepath.Join(r.cfg.SkillDir, "scripts", scriptName)
	cmd := exec.CommandContext(ctx, r.cfg.PythonBin, scriptPath,
		"--json", jsonPath,
		"--output", outFile,
	)
	combined, err := cmd.CombinedOutput()
	if err != nil {
		r.cfg.Logger.Error("python builder failed",
			"script", scriptName,
			"err", err,
			"output", string(combined),
		)
		return nil, fmt.Errorf("python %s failed: %w (output: %s)", scriptName, err, string(combined))
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		return nil, fmt.Errorf("read output %s: %w", outFile, err)
	}

	r.cfg.Logger.Info("builder completed",
		"script", scriptName,
		"output_size", len(data),
		"ext", ext,
	)

	return map[string]any{
		"filename":       filepath.Base(outFile),
		"mime_type":      MimeForExt(ext),
		"content_base64": base64.StdEncoding.EncodeToString(data),
		"size_bytes":     len(data),
	}, nil
}

func (r *Runner) ensureDirs() error {
	if r.cfg.SkillDir == "" {
		return fmt.Errorf("SkillDir is empty")
	}
	if r.cfg.PythonBin == "" {
		return fmt.Errorf("PythonBin is empty")
	}
	if r.cfg.OutputDir == "" {
		return fmt.Errorf("OutputDir is empty")
	}
	if err := os.MkdirAll(r.cfg.OutputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir %s: %w", r.cfg.OutputDir, err)
	}
	return nil
}

// MimeForExt maps a file extension (without dot) to its IANA media type.
func MimeForExt(ext string) string {
	switch ext {
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}
