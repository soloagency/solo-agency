package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReloadConfigIfChanged(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "collector_config.json")
	outputDir := filepath.Join(root, "inbox")
	if err := os.WriteFile(configPath, []byte(`{
  "version": "0.1.0",
  "poll_interval_seconds": 5,
  "max_scrolls_per_source": 2,
  "scheduled_windows": [],
  "clients": []
}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := newBridge(config{
		host:       defaultHost,
		port:       defaultPort,
		configFile: configPath,
		outputDir:  outputDir,
		persistent: true,
		ttl:        defaultTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := getInt(b.configDoc, "max_scrolls_per_source", 0); got != 2 {
		t.Fatalf("initial max_scrolls_per_source = %d, want 2", got)
	}

	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(configPath, []byte(`{
  "version": "0.1.0",
  "poll_interval_seconds": 5,
  "max_scrolls_per_source": 7,
  "config_revision": "changed",
  "scheduled_windows": [],
  "clients": []
}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	b.reloadConfigIfChanged()
	if got := getInt(b.configDoc, "max_scrolls_per_source", 0); got != 7 {
		t.Fatalf("reloaded max_scrolls_per_source = %d, want 7", got)
	}
}

func TestRunNowRequestFileLoadsAndMovesAside(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "collector_config.json")
	outputDir := filepath.Join(root, "inbox")
	if err := os.WriteFile(configPath, []byte(`{"version":"0.1.0","scheduled_windows":[],"clients":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := newBridge(config{
		host:       defaultHost,
		port:       defaultPort,
		configFile: configPath,
		outputDir:  outputDir,
		persistent: true,
		ttl:        defaultTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	requestPath := filepath.Join(root, "run_now_request.json")
	if err := os.WriteFile(requestPath, []byte(`{
  "run_id": "2026-06-20_client_manual_120000",
  "run_now": true,
  "run_now_ttl_minutes": 30,
  "sources": [
    {"name":"Example Group","url":"https://www.facebook.com/groups/example","platform":"facebook"}
  ]
}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	b.loadRunNowRequestIfPresent()

	b.mu.Lock()
	runNow := cloneMap(b.runNowJob)
	b.mu.Unlock()
	if got := getString(runNow, "run_id", ""); got != "2026-06-20_client_manual_120000" {
		t.Fatalf("run_now run_id = %q, want request run_id", got)
	}
	if _, err := os.Stat(requestPath); !os.IsNotExist(err) {
		t.Fatalf("request file still exists or unexpected stat error: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(root, "run_now_request.*.consumed.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("consumed request files = %d, want 1", len(matches))
	}
	statusPath := filepath.Join(root, "run_now_request_status.json")
	if _, err := os.Stat(statusPath); err != nil {
		t.Fatalf("status file not written: %v", err)
	}
	status, err := readMapFile(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := getBool(status, "consumed", false); !got {
		t.Fatalf("status consumed = %v, want true", got)
	}
}

func TestRejectsStaleWritesAfterRunNowComplete(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "collector_config.json")
	outputDir := filepath.Join(root, "inbox")
	if err := os.WriteFile(configPath, []byte(`{"version":"0.1.0","scheduled_windows":[],"clients":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := newBridge(config{
		host:       defaultHost,
		port:       defaultPort,
		configFile: configPath,
		outputDir:  outputDir,
		persistent: true,
		ttl:        defaultTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.activateRunNowJob(map[string]any{
		"run_id": "manual_deep_scan",
		"sources": []any{
			map[string]any{"name": "Example", "url": "https://www.facebook.com/groups/example"},
		},
	}, time.Now(), "test"); err != nil {
		t.Fatal(err)
	}

	activeReq := httptest.NewRequest(http.MethodPost, "/collect/data_point", bytes.NewBufferString(`{"run_id":"manual_deep_scan","source_name":"Example"}`))
	activeRec := httptest.NewRecorder()
	b.handleDataPoint(activeRec, activeReq)
	if activeRec.Code != http.StatusOK {
		t.Fatalf("active write status = %d, want %d; body=%s", activeRec.Code, http.StatusOK, activeRec.Body.String())
	}

	completeReq := httptest.NewRequest(http.MethodPost, "/complete", nil)
	completeRec := httptest.NewRecorder()
	b.handleComplete(completeRec, completeReq)
	if completeRec.Code != http.StatusOK {
		t.Fatalf("complete status = %d, want %d; body=%s", completeRec.Code, http.StatusOK, completeRec.Body.String())
	}

	staleReq := httptest.NewRequest(http.MethodPost, "/collect/data_point", bytes.NewBufferString(`{"run_id":"manual_deep_scan","source_name":"Example"}`))
	staleRec := httptest.NewRecorder()
	b.handleDataPoint(staleRec, staleReq)
	if staleRec.Code != http.StatusConflict {
		t.Fatalf("stale write status = %d, want %d; body=%s", staleRec.Code, http.StatusConflict, staleRec.Body.String())
	}
}
