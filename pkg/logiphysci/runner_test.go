package logiphysci

import (
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakePythonScript creates an executable shell wrapper that mimics the Python
// builders' --json/--output CLI: it writes a small payload to the output path
// and exits 0. On Windows, tests are skipped (no /bin/sh).
func fakePythonScript(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake python script needs /bin/sh, not available on Windows")
	}
	dir := t.TempDir()
	scriptDir := filepath.Join(dir, "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}

	wrapperPath := filepath.Join(scriptDir, "fake_builder.py")
	wrapper := `#!/bin/sh
out=""
while [ $# -gt 0 ]; do
  case "$1" in
    --output) out="$2"; shift 2 ;;
    *) shift ;;
  esac
done
printf '%s' "MOCKBYTES" > "$out"
exit 0
`
	if err := os.WriteFile(wrapperPath, []byte(wrapper), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestRunnerRunBuilder_Success(t *testing.T) {
	skillDir := fakePythonScript(t)

	r := NewRunner(RunnerConfig{
		SkillDir:  skillDir,
		PythonBin: "/bin/sh",
		OutputDir: t.TempDir(),
		Logger:    newTestLogger(),
	})

	result, err := r.RunBuilder(context.Background(), "fake_builder.py",
		map[string]any{"empfaenger_zeilen": []string{"x"}}, "docx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["mime_type"] != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
		t.Errorf("wrong mime_type: %v", result["mime_type"])
	}
	if result["size_bytes"].(int) != len("MOCKBYTES") {
		t.Errorf("wrong size_bytes: %v", result["size_bytes"])
	}
	decoded, err := base64.StdEncoding.DecodeString(result["content_base64"].(string))
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	if string(decoded) != "MOCKBYTES" {
		t.Errorf("wrong content: %q", string(decoded))
	}
	if !strings.HasSuffix(result["filename"].(string), ".docx") {
		t.Errorf("filename does not end in .docx: %v", result["filename"])
	}
}

func TestRunnerRunBuilder_PythonFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs /bin/sh")
	}
	dir := t.TempDir()
	scriptDir := filepath.Join(dir, "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	failScript := filepath.Join(scriptDir, "fail.py")
	if err := os.WriteFile(failScript, []byte("#!/bin/sh\necho oops >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	r := NewRunner(RunnerConfig{
		SkillDir:  dir,
		PythonBin: "/bin/sh",
		OutputDir: t.TempDir(),
		Logger:    newTestLogger(),
	})

	_, err := r.RunBuilder(context.Background(), "fail.py", map[string]any{}, "pdf")
	if err == nil {
		t.Fatal("expected error from failing script, got nil")
	}
	if !strings.Contains(err.Error(), "fail.py failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunnerRunBuilder_RejectsEmptyConfig(t *testing.T) {
	cases := []struct {
		name string
		cfg  RunnerConfig
		want string
	}{
		{"empty SkillDir", RunnerConfig{PythonBin: "x", OutputDir: t.TempDir()}, "SkillDir is empty"},
		{"empty PythonBin", RunnerConfig{SkillDir: "x", OutputDir: t.TempDir()}, "PythonBin is empty"},
		{"empty OutputDir", RunnerConfig{SkillDir: "x", PythonBin: "x"}, "OutputDir is empty"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			tt.cfg.Logger = newTestLogger()
			r := NewRunner(tt.cfg)
			_, err := r.RunBuilder(context.Background(), "x.py", map[string]any{}, "pdf")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("got %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestNewRunner_DefaultsLogger(t *testing.T) {
	r := NewRunner(RunnerConfig{})
	if r.cfg.Logger == nil {
		t.Error("expected default logger, got nil")
	}
}
