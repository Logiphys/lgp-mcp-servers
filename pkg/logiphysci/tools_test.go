package logiphysci

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/server"
)

// fakeBuildRunner records the RunBuilder call instead of invoking Python.
type fakeBuildRunner struct {
	called  bool
	script  string
	ext     string
	payload any
}

func (f *fakeBuildRunner) RunBuilder(_ context.Context, scriptName string, payload any, ext string) (map[string]any, error) {
	f.called = true
	f.script = scriptName
	f.payload = payload
	f.ext = ext
	return map[string]any{
		"filename":       "out." + ext,
		"mime_type":      "application/octet-stream",
		"size_bytes":     1,
		"content_base64": "QQ==",
	}, nil
}

// callTool drives a registered tool through the server's JSON-RPC layer and
// reports whether the result is an error.
func callTool(t *testing.T, srv *server.MCPServer, name string, args map[string]any) (isError bool, raw string) {
	t.Helper()
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  map[string]any{"name": name, "arguments": args},
	}
	msg, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	resp := srv.HandleMessage(context.Background(), msg)
	out, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Result struct {
			IsError bool `json:"isError"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("unmarshal response: %v (%s)", err, out)
	}
	if parsed.Error != nil {
		t.Fatalf("JSON-RPC error: %s", parsed.Error.Message)
	}
	return parsed.Result.IsError, string(out)
}

func newToolServer(t *testing.T) (*server.MCPServer, *fakeBuildRunner) {
	t.Helper()
	srv := server.NewMCPServer("test", "0.0.0")
	runner := &fakeBuildRunner{}
	RegisterTools(srv, runner, newTestLogger())
	return srv, runner
}

func schemaProperties(t *testing.T, schema string) (props map[string]any, required []string) {
	t.Helper()
	var parsed struct {
		Properties map[string]any `json:"properties"`
		Required   []string       `json:"required"`
	}
	if err := json.Unmarshal([]byte(schema), &parsed); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
	return parsed.Properties, parsed.Required
}

func TestToolsList_ContainsAllBuilders(t *testing.T) {
	srv, _ := newToolServer(t)
	msg := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	resp := srv.HandleMessage(context.Background(), msg)
	out, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"build_brief", "build_angebot", "build_bericht",
		"build_lieferschein", "build_mahnung", "build_konzept", "version",
	} {
		if !strings.Contains(string(out), fmt.Sprintf("%q", name)) {
			t.Errorf("tools/list missing %q", name)
		}
	}
}

func TestKonzeptSchema_RequiresOnlyTitel(t *testing.T) {
	props, required := schemaProperties(t, konzeptSchema)

	if len(required) != 1 || required[0] != "titel" {
		t.Errorf("expected required [titel], got %v", required)
	}
	for _, want := range []string{
		"titel", "untertitel", "auftraggeber", "bezug", "datum", "version",
		"dokumenttitel_kopfzeile", "ersteller_name", "ersteller_rolle",
		"ersteller_email", "ersteller_telefon", "vorspann_blocks",
	} {
		if _, ok := props[want]; !ok {
			t.Errorf("konzeptSchema missing property %q", want)
		}
	}
}

func TestBuildKonzept_RunsBuilder(t *testing.T) {
	srv, runner := newToolServer(t)

	isError, raw := callTool(t, srv, "build_konzept", map[string]any{
		"titel":           "Diagnose- und Lösungsbericht",
		"vorspann_blocks": []any{"Einleitungstext"},
	})
	if isError {
		t.Fatalf("expected success, got error result: %s", raw)
	}
	if !runner.called {
		t.Fatal("runner was not invoked")
	}
	if runner.script != "build_konzept.py" {
		t.Errorf("expected build_konzept.py, got %s", runner.script)
	}
	if runner.ext != "pdf" {
		t.Errorf("expected pdf, got %s", runner.ext)
	}
}

func TestBuildKonzept_RequiresTitel(t *testing.T) {
	srv, runner := newToolServer(t)

	isError, _ := callTool(t, srv, "build_konzept", map[string]any{
		"untertitel": "ohne Titel",
	})
	if !isError {
		t.Fatal("expected error result for missing titel")
	}
	if runner.called {
		t.Error("runner must not be invoked on validation failure")
	}
}

func TestAngebotSchema_GenericDocumentFields(t *testing.T) {
	props, required := schemaProperties(t, angebotSchema)

	// build_angebot.py (lgp-ci >= 2.x) verlangt nur noch empfaenger_zeilen;
	// alles andere hat Defaults.
	if len(required) != 1 || required[0] != "empfaenger_zeilen" {
		t.Errorf("expected required [empfaenger_zeilen], got %v", required)
	}
	for _, want := range []string{
		"anrede", "betreff", "meta_zeilen", "vorspann_blocks", "nachspann_blocks",
		"unterschrift", "absender_grussformel", "absender_name",
		"sig_width_mm", "sig_height_mm", "auftraggeber_signatur",
	} {
		if _, ok := props[want]; !ok {
			t.Errorf("angebotSchema missing property %q", want)
		}
	}
}

func TestBuildAngebot_MinimalGenericPayload(t *testing.T) {
	srv, runner := newToolServer(t)

	isError, raw := callTool(t, srv, "build_angebot", map[string]any{
		"empfaenger_zeilen": []any{"Test GmbH", "Musterstr. 1", "12345 Stadt"},
		"betreff":           "Diagnosebericht",
		"vorspann_blocks":   []any{"Text"},
	})
	if isError {
		t.Fatalf("expected success for minimal generic payload, got error: %s", raw)
	}
	if !runner.called {
		t.Fatal("runner was not invoked")
	}
	if runner.script != "build_angebot.py" {
		t.Errorf("expected build_angebot.py, got %s", runner.script)
	}
}
