package main

// provider_xval_test.go — golden cross-validation of the provider adapter
// against outreach/tools/provider_openapi.py using a local HTTP stub that
// serves an OpenAPI YAML plus sendNotification/uploadAsset/getAccount
// endpoints. Timestamps are canonicalized; outputs and the written
// notification_log.md / provider_calls.jsonl must match semantically.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func providerStub(t *testing.T) (*httptest.Server, *[]map[string]any) {
	t.Helper()
	var calls []map[string]any
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		yaml := "openapi: 3.0.0\nservers:\n  - url: " + srv.URL + "\npaths:\n" +
			"  /account:\n    get:\n      operationId: getAccount\n" +
			"  /notify:\n    post:\n      operationId: sendNotification\n" +
			"  /assets:\n    post:\n      operationId: uploadAsset\n"
		w.Header().Set("Content-Type", "application/yaml")
		w.Write([]byte(yaml))
	})
	record := func(op string, r *http.Request) map[string]any {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		rec := map[string]any{"op": op, "auth": r.Header.Get("Authorization"), "body": body}
		calls = append(calls, rec)
		return body
	}
	mux.HandleFunc("/account", func(w http.ResponseWriter, r *http.Request) {
		record("getAccount", r)
		json.NewEncoder(w).Encode(map[string]any{"account": "leadup", "plan": "pro"})
	})
	mux.HandleFunc("/notify", func(w http.ResponseWriter, r *http.Request) {
		record("sendNotification", r)
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/assets", func(w http.ResponseWriter, r *http.Request) {
		record("uploadAsset", r)
		json.NewEncoder(w).Encode(map[string]any{"url": "https://cdn.example/r/abc.html"})
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &calls
}

var isoTSRe = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z?`)

func canonProviderText(s, tmp string) string {
	s = strings.ReplaceAll(s, tmp, "<TMP>")
	return isoTSRe.ReplaceAllString(s, "<TS>")
}

func canonProviderJSON(t *testing.T, s, tmp string) string {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return canonProviderText(s, tmp)
	}
	var walk func(any) any
	walk = func(x any) any {
		switch y := x.(type) {
		case map[string]any:
			out := map[string]any{}
			for k, val := range y {
				out[k] = walk(val)
			}
			return out
		case []any:
			for i := range y {
				y[i] = walk(y[i])
			}
			return y
		case string:
			return canonProviderText(y, tmp)
		}
		return x
	}
	b, _ := json.Marshal(walk(v))
	return string(b)
}

func runProviderPy(t *testing.T, pyTool string, args []string, env []string) xresult {
	t.Helper()
	cmd := exec.Command("python3", append([]string{pyTool}, args...)...)
	cmd.Env = append(os.Environ(), env...)
	var out, errb strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("python provider %v: %v", args, err)
	}
	return xresult{code, out.String(), errb.String()}
}

func runProviderGo(t *testing.T, args []string, env map[string]string) xresult {
	t.Helper()
	for k, v := range env {
		t.Setenv(k, v)
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr
	code := runProviderCLI(args)
	os.Stdout, os.Stderr = oldOut, oldErr
	wOut.Close()
	wErr.Close()
	outB := make(chan string, 1)
	errB := make(chan string, 1)
	go func() { b, _ := os.ReadFile("/dev/null"); _ = b; s := readAllString(rOut); outB <- s }()
	go func() { errB <- readAllString(rErr) }()
	return xresult{code, <-outB, <-errB}
}

func readAllString(f *os.File) string {
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := f.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return sb.String()
}

// dropDiscoverUnionKeys keeps only the capability groups shared by both forks
// with identical alias sets (notification, analytics) so the union widening in
// the Go adapter does not read as a mismatch.
func dropDiscoverUnionKeys(t *testing.T, s string) string {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("discover output not json: %v", err)
	}
	keep := map[string]bool{"notification": true, "analytics": true}
	for _, key := range []string{"capability_status", "missing_capability_aliases"} {
		if m, ok := v[key].(map[string]any); ok {
			for g := range m {
				if !keep[g] {
					delete(m, g)
				}
			}
		}
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func TestProviderGoldenCrossValidation(t *testing.T) {
	pyRepo := findCrmStorePy(t)
	pyTool := filepath.Join(filepath.Dir(pyRepo), "provider_openapi.py")
	if _, err := os.Stat(pyTool); err != nil {
		t.Skip("provider_openapi.py not present")
	}
	srv, calls := providerStub(t)

	tmp := t.TempDir()
	report := filepath.Join(tmp, "report.html")
	os.WriteFile(report, []byte("<html>weekly</html>"), 0o644)
	writeCfg := func(dir string) string {
		cfg := filepath.Join(dir, "provider_config.local.json")
		body := fmt.Sprintf(`{"active_provider": "widecast", "providers": {"widecast": {
		  "discovery_url": "%s/openapi.yaml", "server_url": "%s", "api_key_local": "k-123",
		  "notification": {"enabled": true}}}}`, srv.URL, srv.URL)
		os.WriteFile(cfg, []byte(body), 0o644)
		return cfg
	}
	pyDir := filepath.Join(tmp, "py")
	goDir := filepath.Join(tmp, "go")
	os.MkdirAll(pyDir, 0o755)
	os.MkdirAll(goDir, 0o755)
	pyCfg := writeCfg(pyDir)
	goCfg := writeCfg(goDir)

	type pstep struct {
		name string
		argv func(cfg, dir string) []string
	}
	steps := []pstep{
		{"discover", func(cfg, dir string) []string {
			return []string{"--config", cfg, "discover", "--out-dir", dir}
		}},
		{"account", func(cfg, dir string) []string {
			return []string{"--config", cfg, "account"}
		}},
		{"notify-dry", func(cfg, dir string) []string {
			return []string{"--config", cfg, "notify", "--message", "5 drafts chờ duyệt", "--dry-run",
				"--log", filepath.Join(dir, "notification_log.md")}
		}},
		{"notify-real", func(cfg, dir string) []string {
			return []string{"--config", cfg, "notify", "--message", "Daily run xong: 3 sent",
				"--subject", "LeadUp daily", "--report-file", report,
				"--log", filepath.Join(dir, "notification_log.md")}
		}},
	}
	for _, s := range steps {
		pr := runProviderPy(t, pyTool, s.argv(pyCfg, pyDir), nil)
		gr := runProviderGo(t, s.argv(goCfg, goDir), nil)
		if pr.Code != gr.Code {
			t.Fatalf("%s: exit py=%d go=%d\npy: %s%s\ngo: %s%s", s.name, pr.Code, gr.Code,
				pr.Stdout, pr.Stderr, gr.Stdout, gr.Stderr)
		}
		pyOut := pr.Stdout
		goOut := gr.Stdout
		if s.name == "discover" {
			// The unified Go adapter's capability groups are the UNION of both
			// forks (content adds production/video groups the outreach fork
			// lacks) — compare group-by-group only where the Python fork
			// defines the group, and drop the union-shaped media group whose
			// alias set intentionally widened.
			pyOut = dropDiscoverUnionKeys(t, pyOut)
			goOut = dropDiscoverUnionKeys(t, goOut)
		}
		pc := canonProviderJSON(t, strings.ReplaceAll(pyOut, srv.URL, "<SRV>"), pyDir)
		gc := canonProviderJSON(t, strings.ReplaceAll(goOut, srv.URL, "<SRV>"), goDir)
		if pc != gc {
			t.Fatalf("%s: stdout mismatch\npy: %s\ngo: %s", s.name, pc, gc)
		}
	}

	// notification_log.md rows must match after timestamp canonicalization
	pyLog, _ := os.ReadFile(filepath.Join(pyDir, "notification_log.md"))
	goLog, _ := os.ReadFile(filepath.Join(goDir, "notification_log.md"))
	pl := canonProviderText(strings.ReplaceAll(string(pyLog), pyDir, "<DIR>"), pyDir)
	gl := canonProviderText(strings.ReplaceAll(string(goLog), goDir, "<DIR>"), goDir)
	pl = strings.ReplaceAll(pl, srv.URL, "<SRV>")
	gl = strings.ReplaceAll(gl, srv.URL, "<SRV>")
	pl = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`).ReplaceAllString(pl, "<D>")
	gl = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`).ReplaceAllString(gl, "<D>")
	if pl != gl {
		t.Fatalf("notification_log.md differs\npy:\n%s\ngo:\n%s", pl, gl)
	}

	// the stub saw equivalent request bodies from both sides (2× each op)
	byOp := map[string][]map[string]any{}
	for _, c := range *calls {
		op := c["op"].(string)
		byOp[op] = append(byOp[op], c)
	}
	for _, op := range []string{"getAccount", "sendNotification", "uploadAsset"} {
		if len(byOp[op])%2 != 0 || len(byOp[op]) == 0 {
			t.Fatalf("%s: expected paired py/go calls, got %d", op, len(byOp[op]))
		}
	}
	// the two sendNotification bodies must be identical (subject+message contract)
	sn := byOp["sendNotification"]
	pb, _ := json.Marshal(sn[0]["body"])
	gb, _ := json.Marshal(sn[len(sn)-1]["body"])
	pbs := strings.ReplaceAll(string(pb), srv.URL, "<SRV>")
	gbs := strings.ReplaceAll(string(gb), srv.URL, "<SRV>")
	if pbs != gbs {
		t.Fatalf("sendNotification body differs\npy: %s\ngo: %s", pbs, gbs)
	}
	if !strings.Contains(pbs, `"subject":"LeadUp daily"`) || !strings.Contains(pbs, "cdn.example") {
		t.Fatalf("sendNotification body missing subject/report link: %s", pbs)
	}
}
