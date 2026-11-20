//go:build pilot

package pilot_test

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

// startLocalFixture runs the actual gateway and both stdio binaries against
// loopback product APIs. All keys, gateway state and binaries are temporary.
func startLocalFixture(t *testing.T, gateway string) {
	t.Helper()
	gateway, err := filepath.Abs(gateway)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	servers := filepath.Join(dir, "config", "servers")
	if err := os.MkdirAll(servers, 0o700); err != nil {
		t.Fatal(err)
	}
	write := func(path string, v any) {
		t.Helper()
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, path), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	build := exec.CommandContext(t.Context(), "go", "build", "-o", dir, "./cmd/autotask-mcp", "./cmd/datto-rmm-mcp")
	build.Dir = filepath.Join("..", "..")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build product binaries: %v\n%s", err, out)
	}
	const linePath = "/ATServicesRest/V1.0/ContractServices/query"
	const unitsPath = "/ATServicesRest/V1.0/ContractServiceUnits/query"
	calls := map[string]int{}
	var mu sync.Mutex
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		mu.Lock()
		calls[path]++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + path {
		case "POST /auth/oauth/token":
			fmt.Fprint(w, `{"access_token":"local-fixture","expires_in":3600}`)
		case "GET /api/v2/device/device-1/patches":
			fmt.Fprint(w, `{"patches":[{"patchId":"p1","installStatus":"APPROVED_PENDING"}],"pageDetails":{"count":1,"totalCount":1,"nextPageUrl":null,"prevPageUrl":null}}`)
		case "GET /api/v2/device/restricted/patches":
			w.WriteHeader(http.StatusForbidden)
		case "POST " + linePath:
			fmt.Fprint(w, `{"items":[{"id":11,"contractID":7,"serviceID":8,"unitPrice":9.99}],"pageDetails":{"nextPageUrl":null}}`)
		case "POST " + unitsPath:
			fmt.Fprint(w, `{"items":[{"id":22,"contractServiceID":11,"contractID":7,"serviceID":8,"units":4,"price":4.04,"startDate":"2026-04-15","endDate":"2026-04-30"}],"pageDetails":{"nextPageUrl":null}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(func() {
		api.Close()
		want := map[string]int{"/auth/oauth/token": 1, "/api/v2/device/device-1/patches": 1, "/api/v2/device/restricted/patches": 1, linePath: 2, unitsPath: 2}
		if !t.Failed() && !maps.Equal(calls, want) {
			t.Error("unexpected access to the fixture APIs")
		}
	})

	var keys []map[string]any
	for role, level := range map[string]int{"ALLOWED": 2, "BASIC": 1} {
		key := "lgp_" + rand.Text()
		t.Setenv("LGP_PILOT_"+role+"_KEY", key)
		keys = append(keys, map[string]any{"name": role, "hash": fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(key))), "servers": map[string]int{"autotask": level, "datto-rmm": level}})
	}
	write("config/apikeys.yaml", map[string]any{"keys": keys})
	write("config/admins.yaml", map[string]any{"admins": []any{}})
	binaries := []struct {
		name, prefix, tool string
		env                map[string]string
	}{
		{"autotask", "autotask", "search_contract_services", map[string]string{"AUTOTASK_ENABLE_CONTRACT_SERVICES": "true", "AUTOTASK_USERNAME": "local", "AUTOTASK_SECRET": "fixture", "AUTOTASK_INTEGRATION_CODE": "local", "AUTOTASK_BASE_URL": api.URL + "/ATServicesRest/V1.0"}},
		{"datto-rmm", "datto_rmm", "list_patches", map[string]string{"DATTO_ENABLE_PATCH_READS": "true", "DATTO_API_KEY": "local", "DATTO_API_SECRET": "fixture", "DATTO_BASE_URL": api.URL}},
	}
	for _, b := range binaries {
		bin := filepath.Join(dir, b.name+"-mcp")
		if runtime.GOOS == "windows" {
			bin += ".exe"
		}
		config := map[string]any{"name": b.name, "type": "binary", "enabled": true, "command": bin, "env": b.env, "prefix": b.prefix, "health_tool": "none", "tiering": map[string]any{"tier2": []string{b.prefix + "_" + b.tool}}}
		if b.name == "autotask" {
			config["dispatchers"] = map[string]string{"execute_tool": "toolName"}
		}
		write("config/servers/"+b.name+".yaml", config)
	}
	write("fixture.json", map[string]any{"deviceUID": "device-1", "deniedDeviceUID": "restricted", "patchID": "p1", "installStatus": "APPROVED_PENDING", "asOf": "2026-04-15", "contractID": 7, "lineID": 11, "quantity": 4, "unitPrice": 9.99, "periodPrice": 4.04})
	t.Setenv("LGP_PILOT_FIXTURE", filepath.Join(dir, "fixture.json"))

	port, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := port.Addr().String()
	if err := port.Close(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LGP_PILOT_GATEWAY_URL", "http://"+addr+"/mcp")
	write("config/gateway.yaml", map[string]any{"tls": map[string]string{"mode": "off"}, "listen": map[string]string{"http": addr}})
	process := exec.Command(gateway, "--env", "", "--addr", addr, "--force-http", "--config", filepath.Join(dir, "config"))
	process.Dir = dir
	process.Env = []string{"HOME=" + dir, "PATH=" + os.Getenv("PATH")}
	if runtime.GOOS == "windows" {
		process.Env = append(process.Env, "SystemRoot="+os.Getenv("SystemRoot"))
	}
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() {
		_ = process.Wait()
		close(exited)
	}()
	t.Cleanup(func() {
		if err := process.Process.Signal(os.Interrupt); err != nil {
			_ = process.Process.Kill()
		}
		select {
		case <-exited:
		case <-time.After(5 * time.Second):
			_ = process.Process.Kill()
			<-exited
		}
	})
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+addr+"/health", nil)
	if err != nil {
		t.Fatal(err)
	}
	hc := http.Client{Timeout: time.Second}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if resp, err := hc.Do(req); err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		select {
		case <-exited:
			t.Fatal("local gateway exited during startup")
		case <-time.After(100 * time.Millisecond):
		}
	}
	t.Fatal("local gateway did not become healthy")
}
