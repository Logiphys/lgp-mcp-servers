//go:build integration

package logiphysci

import (
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// findRepoRoot walks up from the test file location until it finds the repo root
// (identified by go.mod). Returns "" if not found.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found walking up from test cwd")
		}
		dir = parent
	}
}

// newIntegrationRunner skips the test unless python3 and the initialized
// logiphys-marketplace submodule are available, then returns a real Runner.
// Requires python-docx, reportlab, pypdf, pillow installed.
func newIntegrationRunner(t *testing.T) *Runner {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("integration tests not supported on Windows")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not in PATH")
	}

	repoRoot := findRepoRoot(t)
	skillDir := filepath.Join(repoRoot, "external", "logiphys-marketplace",
		"plugins", "lgp-docs", "skills", "lgp-ci")
	if _, err := os.Stat(filepath.Join(skillDir, "scripts", "build_brief.py")); err != nil {
		t.Skipf("submodule not initialized: %v", err)
	}

	return NewRunner(RunnerConfig{
		SkillDir:  skillDir,
		PythonBin: "python3",
		OutputDir: t.TempDir(),
		Logger:    newTestLogger(),
	})
}

// TestIntegration_BuildBrief invokes the real Python builder and checks the
// generated DOCX has the expected ZIP magic bytes.
func TestIntegration_BuildBrief(t *testing.T) {
	r := newIntegrationRunner(t)

	payload := map[string]any{
		"empfaenger_zeilen": []string{"Test GmbH", "Musterstr. 1", "12345 Stadt"},
		"betreff":           "Integration-Test",
		"body_paragraphs":   []string{"Hallo", "Mit freundlichen Grüßen"},
	}

	result, err := r.RunBuilder(context.Background(), "build_brief.py", payload, "docx")
	if err != nil {
		t.Fatalf("build_brief failed: %v", err)
	}

	data, err := base64.StdEncoding.DecodeString(result["content_base64"].(string))
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	if len(data) < 4 {
		t.Fatalf("DOCX too small: %d bytes", len(data))
	}
	// DOCX is a ZIP archive — magic bytes "PK\x03\x04"
	if string(data[:4]) != "PK\x03\x04" {
		t.Errorf("DOCX magic bytes missing, got %x...", data[:4])
	}
	if result["mime_type"] != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
		t.Errorf("wrong mime_type: %v", result["mime_type"])
	}
}

// TestIntegration_BuildKonzept exercises the konzept builder end-to-end,
// including heading, table (col_widths) and plain-string blocks.
func TestIntegration_BuildKonzept(t *testing.T) {
	r := newIntegrationRunner(t)

	payload := map[string]any{
		"titel":      "Integrations-Konzept",
		"untertitel": "Testlauf",
		"vorspann_blocks": []any{
			map[string]any{"type": "heading", "text": "Abschnitt 1", "level": 1},
			"Ein einfacher Absatz.",
			map[string]any{
				"type":       "table",
				"rows":       []any{[]any{"Spalte A", "Spalte B"}, []any{"1", "2"}},
				"col_widths": []any{0.3, 0.7},
			},
		},
	}

	result, err := r.RunBuilder(context.Background(), "build_konzept.py", payload, "pdf")
	if err != nil {
		t.Fatalf("build_konzept failed: %v", err)
	}

	data, err := base64.StdEncoding.DecodeString(result["content_base64"].(string))
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	if len(data) < 5 || string(data[:5]) != "%PDF-" {
		t.Errorf("PDF magic bytes missing, got %x...", data[:min(5, len(data))])
	}
	if result["mime_type"] != "application/pdf" {
		t.Errorf("wrong mime_type: %v", result["mime_type"])
	}
}
