package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	if got := getString(runNow, "run_id", ""); got != "" {
		t.Fatalf("run_now active immediately = %q, want queued only", got)
	}
	if _, err := os.Stat(requestPath); !os.IsNotExist(err) {
		t.Fatalf("request file still exists or unexpected stat error: %v", err)
	}
	pendingMatches, err := filepath.Glob(filepath.Join(root, "jobs", "pending", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(pendingMatches) != 1 {
		t.Fatalf("pending jobs = %d, want 1", len(pendingMatches))
	}
	job, available, runID := b.currentPersistentJob(time.Now(), extensionIdentity{instanceID: "ext_any"})
	if !available {
		t.Fatal("queued legacy run-now file was not available to extension")
	}
	if runID != "2026-06-20_client_manual_120000" {
		t.Fatalf("runID = %q, want request run id", runID)
	}
	if got := getString(job, "run_id", ""); got != "2026-06-20_client_manual_120000" {
		t.Fatalf("job run_id = %q, want request run id", got)
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
	if got := getInt(status, "queued_count", 0); got != 1 {
		t.Fatalf("status queued_count = %d, want 1", got)
	}
	if got := getString(status, "request", ""); got != "POST /jobs/run_now" {
		t.Fatalf("status request = %q, want POST /jobs/run_now", got)
	}
	if got := getBool(status, "queued", false); !got {
		t.Fatalf("status queued = %v, want true", got)
	}
	if got := getBool(status, "loaded", true); got {
		t.Fatalf("status loaded = %v, want false before extension claim", got)
	}

	angela := extensionIdentity{clientSlug: "angela-do", instanceID: "ext_angelado_default", displayName: "Angela Do - Solo Agency Collector"}
	job, available, runID := b.currentPersistentJob(time.Now(), angela)
	if !available {
		t.Fatal("queued API run-now job was not available to matching extension")
	}
	if runID != "2026-06-23_angela-do_smoke" {
		t.Fatalf("runID = %q, want api run id", runID)
	}
	if got := getString(job, "client_slug", ""); got != "angela-do" {
		t.Fatalf("job client_slug = %q, want angela-do", got)
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

func TestRunNowBatchQueuesMultipleClientsInParallel(t *testing.T) {
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
  "batch_id": "manual_two_clients",
  "jobs": [
    {
      "run_id": "2026-06-24_angela-do_manual",
      "client_slug": "angela-do",
      "allowed_extension_instance_ids": ["ext_angelado_default"],
      "sources": [{"name":"Angela Group","url":"https://www.facebook.com/groups/angela"}]
    },
    {
      "run_id": "2026-06-24_aven-ngo_manual",
      "client_slug": "aven-ngo",
      "allowed_extension_instance_ids": ["ext_avenngo_default"],
      "sources": [{"name":"Aven Group","url":"https://www.facebook.com/groups/aven"}]
    }
  ]
}`))
	runRec := httptest.NewRecorder()
	b.handleRunNowJob(runRec, runReq)
	if runRec.Code != http.StatusOK {
		t.Fatalf("run-now batch status = %d, want %d; body=%s", runRec.Code, http.StatusOK, runRec.Body.String())
	}
	status, err := readMapFile(filepath.Join(root, "run_now_request_status.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := getInt(status, "queued_count", 0); got != 2 {
		t.Fatalf("queued_count = %d, want 2", got)
	}
	pending, err := filepath.Glob(filepath.Join(root, "jobs", "pending", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 2 {
		t.Fatalf("pending jobs = %d, want 2", len(pending))
	}

	now := time.Now()
	angela := extensionIdentity{clientSlug: "angela-do", instanceID: "ext_angelado_default"}
	aven := extensionIdentity{clientSlug: "aven-ngo", instanceID: "ext_avenngo_default"}

	jobA, availableA, runIDA := b.currentPersistentJob(now, angela)
	if !availableA {
		t.Fatal("angela queued job was not available")
	}
	if runIDA != "2026-06-24_angela-do_manual" {
		t.Fatalf("runIDA = %q, want angela run id", runIDA)
	}
	if got := getString(jobA, "client_slug", ""); got != "angela-do" {
		t.Fatalf("jobA client_slug = %q, want angela-do", got)
	}
	outputA := getString(jobA, "output_dir", "")
	jobB, availableB, runIDB := b.currentPersistentJob(now, aven)
	if !availableB {
		t.Fatal("aven queued job was not available while angela run was active")
	}
	if runIDB != "2026-06-24_aven-ngo_manual" {
		t.Fatalf("runIDB = %q, want aven run id", runIDB)
	}
	if got := getString(jobB, "client_slug", ""); got != "aven-ngo" {
		t.Fatalf("jobB client_slug = %q, want aven-ngo", got)
	}
	outputB := getString(jobB, "output_dir", "")
	if outputA == "" || outputB == "" || outputA == outputB {
		t.Fatalf("output dirs should be non-empty and distinct: A=%q B=%q", outputA, outputB)
	}

	writeA := httptest.NewRequest(http.MethodPost, "/collect/data_point", bytes.NewBufferString(`{"run_id":"`+runIDA+`","client_slug":"angela-do","source_name":"Angela Group"}`))
	writeA.Header.Set("X-Collector-Client-Slug", "angela-do")
	writeA.Header.Set("X-Collector-Extension-Instance", "ext_angelado_default")
	writeRecA := httptest.NewRecorder()
	b.handleDataPoint(writeRecA, writeA)
	if writeRecA.Code != http.StatusOK {
		t.Fatalf("angela write status = %d, want %d; body=%s", writeRecA.Code, http.StatusOK, writeRecA.Body.String())
	}
	writeB := httptest.NewRequest(http.MethodPost, "/collect/data_point", bytes.NewBufferString(`{"run_id":"`+runIDB+`","client_slug":"aven-ngo","source_name":"Aven Group"}`))
	writeB.Header.Set("X-Collector-Client-Slug", "aven-ngo")
	writeB.Header.Set("X-Collector-Extension-Instance", "ext_avenngo_default")
	writeRecB := httptest.NewRecorder()
	b.handleDataPoint(writeRecB, writeB)
	if writeRecB.Code != http.StatusOK {
		t.Fatalf("aven write status = %d, want %d; body=%s", writeRecB.Code, http.StatusOK, writeRecB.Body.String())
	}
	if data, err := os.ReadFile(filepath.Join(outputA, "private_data_points.jsonl")); err != nil || !strings.Contains(string(data), "Angela Group") {
		t.Fatalf("angela output missing data; err=%v data=%s", err, string(data))
	}
	if data, err := os.ReadFile(filepath.Join(outputB, "private_data_points.jsonl")); err != nil || !strings.Contains(string(data), "Aven Group") {
		t.Fatalf("aven output missing data; err=%v data=%s", err, string(data))
	}

	completeReq := httptest.NewRequest(http.MethodPost, "/complete", bytes.NewBufferString(`{"run_id":"`+runIDA+`","client_slug":"angela-do"}`))
	completeReq.Header.Set("X-Collector-Client-Slug", "angela-do")
	completeReq.Header.Set("X-Collector-Extension-Instance", "ext_angelado_default")
	completeRec := httptest.NewRecorder()
	b.handleComplete(completeRec, completeReq)
	if completeRec.Code != http.StatusOK {
		t.Fatalf("complete angela status = %d, want %d; body=%s", completeRec.Code, http.StatusOK, completeRec.Body.String())
	}

	stillB, stillAvailableB, stillRunIDB := b.currentPersistentJob(now, aven)
	if !stillAvailableB {
		t.Fatal("aven active run should remain available after angela completes")
	}
	if stillRunIDB != runIDB {
		t.Fatalf("stillRunIDB = %q, want %q", stillRunIDB, runIDB)
	}
	if got := getString(stillB, "client_slug", ""); got != "aven-ngo" {
		t.Fatalf("stillB client_slug = %q, want aven-ngo", got)
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
	// The output month is derived from the run id's date (2026-06-23 -> 2026-06),
	// so this is deterministic regardless of the wall clock when the test runs.
	output := getString(job, "output_dir", "")
	if !strings.Contains(output, filepath.Join("inbox", "2026-06", "avenngo", "2026-06-23_avenngo_001")) {
		t.Fatalf("output_dir = %q, want inbox/2026-06/avenngo/2026-06-23_avenngo_001 (month from run id)", output)
	}
}

func TestMonthForRun(t *testing.T) {
	now := time.Date(2030, time.November, 9, 0, 0, 0, 0, time.UTC)
	cases := []struct{ runID, want string }{
		{"2026-06-23_avenngo_001", "2026-06"},        // full date prefix
		{"2026-07_client", "2026-07"},                // year-month prefix
		{"2026-12-31_x", "2026-12"},                  // december
		{"graphql_test_aven_20260715_08", "2030-11"}, // no dashed date -> current month
		{"manual_deep_scan", "2030-11"},              // no date -> current month
		{"2026-13-01_bad", "2030-11"},                // invalid month -> current month
	}
	for _, c := range cases {
		if got := monthForRun(c.runID, now); got != c.want {
			t.Errorf("monthForRun(%q) = %q, want %q", c.runID, got, c.want)
		}
	}
}

func TestScheduledJobsRunInParallelPerClient(t *testing.T) {
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

	jobB, availableB, runIDB := b.currentPersistentJob(now, other)
	if !availableB {
		t.Fatal("other client scheduled job was not available while avenngo active")
	}
	if runIDB == runIDA {
		t.Fatalf("other client reused avenngo runID %q", runIDB)
	}
	if got := getString(jobB, "client_slug", ""); got != "other-client" {
		t.Fatalf("jobB client_slug = %q, want other-client", got)
	}

	completeReq := httptest.NewRequest(http.MethodPost, "/complete", bytes.NewBufferString(`{"run_id":"`+runIDA+`","client_slug":"avenngo"}`))
	completeReq.Header.Set("X-Collector-Client-Slug", "avenngo")
	completeReq.Header.Set("X-Collector-Extension-Instance", "ext_avenngo_default")
	completeRec := httptest.NewRecorder()
	b.handleComplete(completeRec, completeReq)
	if completeRec.Code != http.StatusOK {
		t.Fatalf("complete status = %d, want %d; body=%s", completeRec.Code, http.StatusOK, completeRec.Body.String())
	}

	stillB, stillAvailableB, stillRunIDB := b.currentPersistentJob(now, other)
	if !stillAvailableB {
		t.Fatal("other client job should remain active after avenngo completed")
	}
	if stillRunIDB != runIDB {
		t.Fatalf("stillRunIDB = %q, want %q", stillRunIDB, runIDB)
	}
	if got := getString(stillB, "client_slug", ""); got != "other-client" {
		t.Fatalf("stillB client_slug = %q, want other-client", got)
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

// TestDataPointContactsSurviveToRecord locks in the additive contract for the
// collector's new structured email/phone extraction: the bridge is a pass-through
// of arbitrary data-point JSON, so the optional `contacts` object (plus flat
// emails/phones, if ever sent) posted by the extension must survive sanitize and
// land verbatim in the written private_data_points.jsonl record.
func TestDataPointContactsSurviveToRecord(t *testing.T) {
	root := t.TempDir()
	outputDir := filepath.Join(root, "inbox")
	b, err := newBridge(config{
		host:      defaultHost,
		port:      defaultPort,
		outputDir: outputDir,
		ttl:       defaultTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := b.activateRunNowJob(map[string]any{
		"run_id": "2026-07-16_contacts_test",
		"sources": []any{
			map[string]any{"name": "Example", "url": "https://example.com/contact"},
		},
	}, time.Now(), "test")
	if err != nil {
		t.Fatal(err)
	}
	runOutputDir := getString(resp, "output_dir", "")
	if runOutputDir == "" {
		t.Fatal("activateRunNowJob returned empty output_dir")
	}

	body := `{"run_id":"2026-07-16_contacts_test","source_name":"Example",` +
		`"contacts":{"emails":["jane@acme.com","sales@acme.com"],"phones":["+14155550100"]},` +
		`"emails":["jane@acme.com"],"phones":["+14155550100"]}`
	req := httptest.NewRequest(http.MethodPost, "/collect/data_point", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	b.handleDataPoint(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("data_point status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	data, err := os.ReadFile(filepath.Join(runOutputDir, "private_data_points.jsonl"))
	if err != nil {
		t.Fatalf("read data points: %v", err)
	}
	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(data), &record); err != nil {
		t.Fatalf("decode record: %v; data=%s", err, string(data))
	}

	contacts, ok := record["contacts"].(map[string]any)
	if !ok {
		t.Fatalf("record has no structured contacts object; record=%s", string(data))
	}
	emails, _ := contacts["emails"].([]any)
	if len(emails) != 2 || fmt.Sprint(emails[0]) != "jane@acme.com" || fmt.Sprint(emails[1]) != "sales@acme.com" {
		t.Fatalf("contacts.emails = %v, want [jane@acme.com sales@acme.com]", contacts["emails"])
	}
	phones, _ := contacts["phones"].([]any)
	if len(phones) != 1 || fmt.Sprint(phones[0]) != "+14155550100" {
		t.Fatalf("contacts.phones = %v, want [+14155550100]", contacts["phones"])
	}

	// Flat top-level arrays must also pass through untouched (arbitrary additive keys).
	flatEmails, _ := record["emails"].([]any)
	if len(flatEmails) != 1 || fmt.Sprint(flatEmails[0]) != "jane@acme.com" {
		t.Fatalf("record.emails = %v, want [jane@acme.com]", record["emails"])
	}
	flatPhones, _ := record["phones"].([]any)
	if len(flatPhones) != 1 || fmt.Sprint(flatPhones[0]) != "+14155550100" {
		t.Fatalf("record.phones = %v, want [+14155550100]", record["phones"])
	}
}
