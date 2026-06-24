package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestRunNowAPIWritesStatusAndCompletion(t *testing.T) {
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

	runReq := httptest.NewRequest(http.MethodPost, "/jobs/run_now", bytes.NewBufferString(`{
  "run_id": "2026-06-23_angela-do_smoke",
  "client_slug": "angela-do",
  "allowed_extension_instance_ids": ["ext_angelado_default"],
  "sources": [
    {"name":"Example Group","url":"https://www.facebook.com/groups/example","platform":"facebook"}
  ]
}`))
	runRec := httptest.NewRecorder()
	b.handleRunNowJob(runRec, runReq)
	if runRec.Code != http.StatusOK {
		t.Fatalf("run-now status = %d, want %d; body=%s", runRec.Code, http.StatusOK, runRec.Body.String())
	}

	statusPath := filepath.Join(root, "run_now_request_status.json")
	status, err := readMapFile(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := getString(status, "run_id", ""); got != "2026-06-23_angela-do_smoke" {
		t.Fatalf("status run_id = %q, want api run id", got)
	}
	if got := getString(status, "request", ""); got != "POST /jobs/run_now" {
		t.Fatalf("status request = %q, want POST /jobs/run_now", got)
	}
	if got := getBool(status, "loaded", false); !got {
		t.Fatalf("status loaded = %v, want true", got)
	}

	completeReq := httptest.NewRequest(http.MethodPost, "/complete", bytes.NewBufferString(`{"run_id":"2026-06-23_angela-do_smoke","client_slug":"angela-do"}`))
	completeReq.Header.Set("X-Collector-Client-Slug", "angela-do")
	completeReq.Header.Set("X-Collector-Extension-Instance", "ext_angelado_default")
	completeRec := httptest.NewRecorder()
	b.handleComplete(completeRec, completeReq)
	if completeRec.Code != http.StatusOK {
		t.Fatalf("complete status = %d, want %d; body=%s", completeRec.Code, http.StatusOK, completeRec.Body.String())
	}

	status, err = readMapFile(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := getString(status, "status", ""); got != "completed" {
		t.Fatalf("status after complete = %q, want completed", got)
	}
	if got := getBool(status, "completed", false); !got {
		t.Fatalf("completed = %v, want true", got)
	}
	if got := getString(status, "extension_instance_id", ""); got != "ext_angelado_default" {
		t.Fatalf("extension_instance_id = %q, want ext_angelado_default", got)
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

func TestQueuedJobRoutesOnlyToMatchingExtension(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "collector_config.json")
	outputDir := filepath.Join(root, "inbox")
	if err := os.WriteFile(configPath, []byte(`{"version":"0.1.0","scheduled_windows":[],"clients":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	pendingDir := filepath.Join(root, "jobs", "pending")
	if err := os.MkdirAll(pendingDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pendingDir, "avenngo.json"), []byte(`{
  "run_id": "2026-06-23_avenngo_001",
  "client_slug": "avenngo",
  "allowed_extension_instance_ids": ["ext_avenngo_default"],
  "sources": [
    {"name":"AvenNgo Group","url":"https://www.facebook.com/groups/avenngo","platform":"facebook"}
  ]
}`), 0o600); err != nil {
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

	other := extensionIdentity{clientSlug: "other-client", instanceID: "ext_other_default"}
	if _, available, _ := b.currentPersistentJob(time.Now(), other); available {
		t.Fatal("queued avenngo job was exposed to another client extension")
	}

	avenngo := extensionIdentity{clientSlug: "avenngo", instanceID: "ext_avenngo_default", displayName: "AvenNgo - Solo Agency Collector"}
	job, available, runID := b.currentPersistentJob(time.Now(), avenngo)
	if !available {
		t.Fatal("queued avenngo job was not available to matching extension")
	}
	if runID != "2026-06-23_avenngo_001" {
		t.Fatalf("runID = %q, want queued run id", runID)
	}
	if got := getString(job, "client_slug", ""); got != "avenngo" {
		t.Fatalf("job client_slug = %q, want avenngo", got)
	}
	output := getString(job, "output_dir", "")
	if !strings.Contains(output, filepath.Join("inbox", "2026-06", "avenngo", "2026-06-23_avenngo_001")) {
		t.Fatalf("output_dir = %q, want per-client output dir", output)
	}
}

func TestScheduledJobDoesNotSwitchClientsWhileActive(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "collector_config.json")
	outputDir := filepath.Join(root, "inbox")
	if err := os.WriteFile(configPath, []byte(`{
  "version": "0.1.0",
  "scheduled_job_ttl_minutes": 60,
  "scheduled_windows": [
    {"name":"morning","enabled":true,"local_time_start":"09:00","local_time_end":"10:00","days":["tue"]}
  ],
  "clients": [
    {
      "client_slug": "avenngo",
      "sources": [{"name":"AvenNgo Group","url":"https://www.facebook.com/groups/avenngo"}]
    },
    {
      "client_slug": "other-client",
      "sources": [{"name":"Other Group","url":"https://www.facebook.com/groups/other"}]
    }
  ]
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
	now := time.Date(2026, 6, 23, 9, 5, 0, 0, time.UTC)
	avenngo := extensionIdentity{clientSlug: "avenngo", instanceID: "ext_avenngo_default"}
	other := extensionIdentity{clientSlug: "other-client", instanceID: "ext_other_default"}

	jobA, availableA, runIDA := b.currentPersistentJob(now, avenngo)
	if !availableA {
		t.Fatal("scheduled avenngo job was not available")
	}
	if got := getString(jobA, "client_slug", ""); got != "avenngo" {
		t.Fatalf("jobA client_slug = %q, want avenngo", got)
	}

	if jobB, availableB, runIDB := b.currentPersistentJob(now, other); availableB {
		t.Fatalf("other client received job while avenngo active: runID=%s job=%v", runIDB, jobB)
	} else if runIDB != runIDA {
		t.Fatalf("blocked runID = %q, want active runID %q", runIDB, runIDA)
	}

	completeReq := httptest.NewRequest(http.MethodPost, "/complete", bytes.NewBufferString(`{"run_id":"`+runIDA+`","client_slug":"avenngo"}`))
	completeReq.Header.Set("X-Collector-Client-Slug", "avenngo")
	completeReq.Header.Set("X-Collector-Extension-Instance", "ext_avenngo_default")
	completeRec := httptest.NewRecorder()
	b.handleComplete(completeRec, completeReq)
	if completeRec.Code != http.StatusOK {
		t.Fatalf("complete status = %d, want %d; body=%s", completeRec.Code, http.StatusOK, completeRec.Body.String())
	}

	jobB, availableB, runIDB := b.currentPersistentJob(now, other)
	if !availableB {
		t.Fatal("other client job was not available after avenngo completed")
	}
	if runIDB == runIDA {
		t.Fatalf("other client reused avenngo runID %q", runIDB)
	}
	if got := getString(jobB, "client_slug", ""); got != "other-client" {
		t.Fatalf("jobB client_slug = %q, want other-client", got)
	}
}

func TestRejectsWritesFromWrongExtensionClient(t *testing.T) {
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
		"run_id":                         "2026-06-23_avenngo_001",
		"client_slug":                    "avenngo",
		"allowed_extension_instance_ids": []any{"ext_avenngo_default"},
		"sources": []any{
			map[string]any{"name": "AvenNgo Group", "url": "https://www.facebook.com/groups/avenngo"},
		},
	}, time.Now(), "test"); err != nil {
		t.Fatal(err)
	}

	wrongReq := httptest.NewRequest(http.MethodPost, "/collect/data_point", bytes.NewBufferString(`{"run_id":"2026-06-23_avenngo_001","client_slug":"other-client","source_name":"Wrong"}`))
	wrongReq.Header.Set("X-Collector-Client-Slug", "other-client")
	wrongReq.Header.Set("X-Collector-Extension-Instance", "ext_other_default")
	wrongRec := httptest.NewRecorder()
	b.handleDataPoint(wrongRec, wrongReq)
	if wrongRec.Code != http.StatusConflict {
		t.Fatalf("wrong extension write status = %d, want %d; body=%s", wrongRec.Code, http.StatusConflict, wrongRec.Body.String())
	}

	rightReq := httptest.NewRequest(http.MethodPost, "/collect/data_point", bytes.NewBufferString(`{"run_id":"2026-06-23_avenngo_001","client_slug":"avenngo","source_name":"Right"}`))
	rightReq.Header.Set("X-Collector-Client-Slug", "avenngo")
	rightReq.Header.Set("X-Collector-Extension-Instance", "ext_avenngo_default")
	rightRec := httptest.NewRecorder()
	b.handleDataPoint(rightRec, rightReq)
	if rightRec.Code != http.StatusOK {
		t.Fatalf("right extension write status = %d, want %d; body=%s", rightRec.Code, http.StatusOK, rightRec.Body.String())
	}
}
