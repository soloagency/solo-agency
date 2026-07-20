package main

// provider_xval_test.go — provider adapter tests against a local OpenAPI stub.
// Originally cross-validated against the retired Python adapter; the
// expectations below are the Python-verified outcomes asserted directly.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	record := func(op string, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		calls = append(calls, map[string]any{"op": op, "auth": r.Header.Get("Authorization"), "body": body})
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

func runProviderGo(t *testing.T, args []string) xresult {
	t.Helper()
	return runCLIStep(t, xstep{"2026-07-19T12:00:00Z", args}, runProviderCLI)
}

func TestProviderStubFlow(t *testing.T) {
	srv, calls := providerStub(t)
	tmp := t.TempDir()
	report := filepath.Join(tmp, "report.html")
	os.WriteFile(report, []byte("<html>weekly</html>"), 0o644)
	cfg := filepath.Join(tmp, "provider_config.local.json")
	os.WriteFile(cfg, []byte(fmt.Sprintf(`{"active_provider": "widecast", "providers": {"widecast": {
	  "discovery_url": "%s/openapi.yaml", "server_url": "%s", "api_key_local": "k-123",
	  "notification": {"enabled": true}}}}`, srv.URL, srv.URL)), 0o644)
	logPath := filepath.Join(tmp, "notification_log.md")

	// discover: aliases resolved; union capability groups present
	disc := parseOut(t, runProviderGo(t, []string{"--config", cfg, "discover", "--out-dir", tmp}))
	aliases := mMap(disc, "operation_aliases")
	if aliases["send_notification"] != "sendNotification" || aliases["upload_asset"] != "uploadAsset" {
		t.Fatalf("aliases: %v", aliases)
	}
	caps := mMap(disc, "capability_status")
	if caps["notification"] != "available" || caps["production"] != "unavailable" {
		t.Fatalf("capability union: %v", caps)
	}
	if _, err := os.Stat(filepath.Join(tmp, "provider_capabilities.json")); err != nil {
		t.Fatalf("capabilities file: %v", err)
	}

	// account call
	acct := parseOut(t, runProviderGo(t, []string{"--config", cfg, "account"}))
	if mInt(acct, "status", 0) != 200 || mStr(mMap(acct, "body"), "account") != "leadup" {
		t.Fatalf("account: %v", acct)
	}

	// notify dry-run: no network side effects on the ops, logs the plan
	dry := parseOut(t, runProviderGo(t, []string{"--config", cfg, "notify",
		"--message", "5 drafts", "--dry-run", "--log", logPath}))
	if mStr(dry, "status") != "dry_run" {
		t.Fatalf("dry-run: %v", dry)
	}

	// real notify with report upload: uploads, links, sends subject+message
	preCalls := len(*calls)
	real := parseOut(t, runProviderGo(t, []string{"--config", cfg, "notify",
		"--message", "Daily run xong: 3 sent", "--subject", "LeadUp daily",
		"--report-file", report, "--log", logPath}))
	if mStr(real, "status") != "sent" || mStr(real, "uploaded_report_url") != "https://cdn.example/r/abc.html" {
		t.Fatalf("notify: %v", real)
	}
	var sendBody map[string]any
	for _, c := range (*calls)[preCalls:] {
		if c["op"] == "sendNotification" {
			sendBody, _ = c["body"].(map[string]any)
		}
		if auth, _ := c["auth"].(string); auth != "Bearer k-123" {
			t.Fatalf("missing bearer auth on %v", c["op"])
		}
	}
	if mStr(sendBody, "subject") != "LeadUp daily" ||
		!strings.Contains(mStr(sendBody, "message"), "Report: https://cdn.example/r/abc.html") {
		t.Fatalf("sendNotification body: %v", sendBody)
	}

	// notification log: header + dry_run row + sent row
	logB, _ := os.ReadFile(logPath)
	logS := string(logB)
	if !strings.Contains(logS, "| Date | Agent |") || !strings.Contains(logS, "dry_run") ||
		!strings.Contains(logS, "| sent |") {
		t.Fatalf("notification log:\n%s", logS)
	}
	// provider_calls.jsonl written next to the config
	if rows := readJSONLines(filepath.Join(tmp, "provider_calls.jsonl")); len(rows) < 2 {
		t.Fatalf("provider_calls.jsonl rows: %d", len(rows))
	}
}

func TestProviderNotifyDegrades(t *testing.T) {
	tmp := t.TempDir()
	// no config at all -> local_path_only, exit 0 (a degraded state is not a failure)
	r := runProviderGo(t, []string{"--config", filepath.Join(tmp, "missing.json"),
		"notify", "--message", "x"})
	out := parseOut(t, r)
	if r.Code != 0 || mStr(out, "status") != "local_path_only" || mStr(out, "blocker") != "provider_config_missing" {
		t.Fatalf("degrade: %d %v", r.Code, out)
	}
}
