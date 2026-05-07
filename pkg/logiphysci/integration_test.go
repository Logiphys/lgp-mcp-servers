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

// TestIntegration_BuildBrief invokes the real Python builder and checks the
// generated DOCX has the expected ZIP magic bytes. Requires:
//   - python3 in PATH
//   - python-docx, reportlab, pypdf, pillow installed
//   - external/logiphys-marketplace submodule initialized
func TestIntegration_BuildBrief(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("integration tests not supported on Windows")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not in PATH")
	}

	repoRoot := findRepoRoot(t)
	skillDir := filepath.Join(repoRoot, "external", "logiphys-marketplace",
		"plugins", "lgp-docs", "skills", "logiphys-ci")
	if _, err := os.Stat(filepath.Join(skillDir, "scripts", "build_brief.py")); err != nil {
		t.Skipf("submodule not initialized: %v", err)
	}

	r := NewRunner(RunnerConfig{
		SkillDir:  skillDir,
		PythonBin: "python3",
		OutputDir: t.TempDir(),
		Logger:    newTestLogger(),
	})

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
