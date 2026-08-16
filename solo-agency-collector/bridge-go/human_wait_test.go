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

// TestHumanWaitHeartbeatKeepsRunAlive locks the contract behind the Zillow human gate: while
// the extension waits for the operator to pass a bot check it heartbeats source_status
// "waiting_for_human_captcha"; each heartbeat must push the run's expiry out, so a wait longer
// than run_now_ttl_minutes (max 120) does not expire the job and lose the data point that is
// posted once the operator finally passes the check. Other statuses must not extend anything.
func TestHumanWaitHeartbeatKeepsRunAlive(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "collector_config.json")
	if err := os.WriteFile(configPath, []byte(`{"version":"0.1.0","scheduled_windows":[],"clients":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := newBridge(config{
		host: defaultHost, port: defaultPort, configFile: configPath,
		outputDir: filepath.Join(root, "inbox"), persistent: true, ttl: defaultTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	runReq := httptest.NewRequest(http.MethodPost, "/jobs/run_now", bytes.NewBufferString(`{
	  "run_id": "zillow_wait_1", "client_slug": "aven-ngo", "run_now_ttl_minutes": 1,
	  "allowed_extension_instance_ids": ["ext_avenngo_default"],
	  "sources": [{"name":"zillow profile","url":"https://www.zillow.com/profile/tykunkle","capability":"zillow.profile.enrich"}]
	}`))
	runRec := httptest.NewRecorder()
	b.handleRunNowJob(runRec, runReq)
	if runRec.Code != http.StatusOK {
		t.Fatalf("run_now status = %d; body=%s", runRec.Code, runRec.Body.String())
	}
	now := time.Now()
	aven := extensionIdentity{clientSlug: "aven-ngo", instanceID: "ext_avenngo_default"}
	job, available, runID := b.currentPersistentJob(now, aven)
	if !available || runID != "zillow_wait_1" {
		t.Fatalf("job not activated: available=%v runID=%q", available, runID)
	}
	before, err := time.Parse(time.RFC3339, getString(job, "run_now_expires_at", ""))
	if err != nil {
		t.Fatalf("run_now_expires_at unparsable: %v", err)
	}
	if before.After(now.Add(2 * time.Minute)) {
		t.Fatalf("test premise: ttl 1 min, got expiry %v", before)
	}

	post := func(status string) int {
		req := httptest.NewRequest(http.MethodPost, "/collect/source_status", bytes.NewBufferString(`{"run_id":"zillow_wait_1","client_slug":"aven-ngo","source_name":"zillow profile","status":"`+status+`"}`))
		req.Header.Set("X-Collector-Client-Slug", "aven-ngo")
		req.Header.Set("X-Collector-Extension-Instance", "ext_avenngo_default")
		rec := httptest.NewRecorder()
		b.handleSourceStatus(rec, req)
		return rec.Code
	}
	expiry := func() time.Time {
		b.mu.Lock()
		defer b.mu.Unlock()
		for _, run := range b.activeRuns {
			if getString(run.job, "run_id", "") == "zillow_wait_1" {
				ts, err := time.Parse(time.RFC3339, getString(run.job, "run_now_expires_at", ""))
				if err != nil {
					t.Fatalf("expiry unparsable: %v", err)
				}
				return ts
			}
		}
		t.Fatal("run zillow_wait_1 not active")
		return time.Time{}
	}

	// an ordinary status ("started") does not touch the expiry
	if code := post("started"); code != http.StatusOK {
		t.Fatalf("started status = %d", code)
	}
	if got := expiry(); !got.Equal(before) {
		t.Fatalf("'started' must not extend expiry: before=%v after=%v", before, got)
	}
	// the human-wait heartbeat pushes it to at least now+30min
	if code := post(humanWaitSourceStatus); code != http.StatusOK {
		t.Fatalf("heartbeat status = %d", code)
	}
	after := expiry()
	if !after.After(now.Add(humanWaitExpiryFloor - time.Minute)) {
		t.Fatalf("heartbeat must extend expiry to ~now+%v: before=%v after=%v", humanWaitExpiryFloor, before, after)
	}
	// the run is still available to the same extension well past the original 1-minute TTL
	if _, still, _ := b.currentPersistentJob(now.Add(20*time.Minute), aven); !still {
		t.Fatal("run expired despite the human-wait heartbeat")
	}
	// a later heartbeat keeps pushing (never shortens)
	if !b.extendActiveRunExpiry(activeRunKey(job, aven), now.Add(25*time.Minute)) {
		t.Fatal("second heartbeat should extend again")
	}
	if got := expiry(); !got.After(after) {
		t.Fatalf("second heartbeat must move expiry forward: %v -> %v", after, got)
	}
	// unknown key is a harmless no-op
	if b.extendActiveRunExpiry("client:nobody", now) {
		t.Fatal("unknown active key must not report an extension")
	}
	// the data point posted after the operator passed the check is accepted
	dp := httptest.NewRequest(http.MethodPost, "/collect/data_point", bytes.NewBufferString(`{"run_id":"zillow_wait_1","client_slug":"aven-ngo","source_name":"zillow profile","capability":"zillow.profile.enrich","records":{"available":true,"count":1,"items":[{"name":"Tyler Kunkle"}]}}`))
	dp.Header.Set("X-Collector-Client-Slug", "aven-ngo")
	dp.Header.Set("X-Collector-Extension-Instance", "ext_avenngo_default")
	dpRec := httptest.NewRecorder()
	b.handleDataPoint(dpRec, dp)
	if dpRec.Code != http.StatusOK {
		t.Fatalf("data point after the wait = %d; body=%s", dpRec.Code, dpRec.Body.String())
	}
}
