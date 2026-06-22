package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	defaultHost      = "127.0.0.1"
	defaultPort      = 17321
	defaultTTL       = 30 * time.Minute
	maxJSONBodyBytes = 512 * 1024
	maxHTMLBodyBytes = 3 * 1024 * 1024
)

type config struct {
	host        string
	port        int
	runID       string
	jobFile     string
	configFile  string
	outputDir   string
	ttl         time.Duration
	extensionID string
	token       string
	persistent  bool
	verbose     bool
}

type bridge struct {
	cfg        config
	token      string
	startedAt  time.Time
	job        map[string]any
	runNowJob  map[string]any
	configDoc  map[string]any
	outputRoot string
	server     *http.Server

	configFileModTime    time.Time
	configFileSize       int64
	runNowRequestPath    string
	lastRunNowRequestSig string

	mu          sync.Mutex
	counts      map[string]int
	completed   bool
	completions map[string]string
	extension   extensionTelemetry
	stopping    bool
}

type extensionTelemetry struct {
	lastCheckAt time.Time
	lastOrigin  string
	lastVersion string
	checkCount  int
}

func main() {
	cfg := parseFlags()
	if cfg.persistent {
		if cfg.outputDir == "" {
			cfg.outputDir = filepath.Join("daily-content-pipeline", "collector", "inbox")
		}
	} else if cfg.runID == "" {
		cfg.runID = time.Now().UTC().Format("2006-01-02_150405")
	}
	if cfg.outputDir == "" {
		cfg.outputDir = filepath.Join("daily-content-pipeline", "collector", "inbox", cfg.runID)
	}

	b, err := newBridge(cfg)
	if err != nil {
		log.Fatalf("collector bridge init failed: %v", err)
	}
	if err := b.run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("collector bridge failed: %v", err)
	}
}

func parseFlags() config {
	var ttlMinutes int
	cfg := config{}
	flag.StringVar(&cfg.host, "host", defaultHost, "host to bind; use 127.0.0.1 for local-only operation")
	flag.IntVar(&cfg.port, "port", defaultPort, "localhost port")
	flag.StringVar(&cfg.runID, "run-id", "", "run id, usually YYYY-MM-DD_client_slug")
	flag.StringVar(&cfg.jobFile, "job-file", "", "path to collector job JSON")
	flag.StringVar(&cfg.configFile, "config-file", "", "path to shared collector_config.json")
	flag.StringVar(&cfg.outputDir, "output-dir", "", "directory for collector output files")
	flag.IntVar(&ttlMinutes, "ttl-minutes", int(defaultTTL/time.Minute), "auto-shutdown timeout in minutes")
	flag.StringVar(&cfg.extensionID, "extension-id", "", "optional Chrome extension id allowed to use this bridge")
	flag.StringVar(&cfg.token, "token", "", "optional write token; generated if omitted")
	flag.BoolVar(&cfg.persistent, "persistent", false, "run continuously and expose scheduled jobs from collector_config.json")
	flag.BoolVar(&cfg.verbose, "verbose", false, "enable verbose request logging")
	flag.Parse()

	if ttlMinutes <= 0 {
		ttlMinutes = int(defaultTTL / time.Minute)
	}
	cfg.ttl = time.Duration(ttlMinutes) * time.Minute
	return cfg
}

func newBridge(cfg config) (*bridge, error) {
	if cfg.host != "127.0.0.1" && cfg.host != "localhost" {
		return nil, fmt.Errorf("refusing to bind non-local host %q", cfg.host)
	}
	if cfg.token == "" {
		tok, err := randomToken()
		if err != nil {
			return nil, err
		}
		cfg.token = tok
	}
	job, err := loadJob(cfg)
	if err != nil {
		return nil, err
	}
	configDoc, err := loadCollectorConfig(cfg)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(cfg.outputDir, "snapshots"), 0o700); err != nil {
		return nil, err
	}
	completions := loadCompletions(cfg.outputDir)
	configModTime, configSize := fileStamp(cfg.configFile)

	b := &bridge{
		cfg:               cfg,
		token:             cfg.token,
		startedAt:         time.Now().UTC(),
		job:               job,
		configDoc:         configDoc,
		outputRoot:        cfg.outputDir,
		configFileModTime: configModTime,
		configFileSize:    configSize,
		runNowRequestPath: defaultRunNowRequestPath(cfg),
		completions:       completions,
		counts:            emptyCounts(),
	}
	if err := b.writeStatus("ready", "bridge started"); err != nil {
		return nil, err
	}
	if err := b.writeHealthFile(); err != nil {
		return nil, err
	}
	return b, nil
}

func loadCollectorConfig(cfg config) (map[string]any, error) {
	doc := defaultCollectorConfig()
	if cfg.configFile == "" {
		return doc, nil
	}
	f, err := os.Open(cfg.configFile)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(cfg.configFile), 0o700); err != nil {
				return nil, err
			}
			if err := writeConfigFile(cfg.configFile, doc); err != nil {
				return nil, err
			}
			return doc, nil
		}
		return nil, err
	}
	defer f.Close()
	dec := json.NewDecoder(io.LimitReader(f, maxJSONBodyBytes))
	dec.UseNumber()
	if err := dec.Decode(&doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func fileStamp(path string) (time.Time, int64) {
	if path == "" {
		return time.Time{}, 0
	}
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, 0
	}
	return info.ModTime(), info.Size()
}

func defaultRunNowRequestPath(cfg config) string {
	if cfg.configFile != "" {
		return filepath.Join(filepath.Dir(cfg.configFile), "run_now_request.json")
	}
	if cfg.outputDir != "" {
		return filepath.Join(cfg.outputDir, "run_now_request.json")
	}
	return ""
}

func defaultCollectorConfig() map[string]any {
	return map[string]any{
		"version":                "0.1.0",
		"timezone":               "local",
		"run_mode":               "persistent_bridge_scheduler",
		"default_runs_per_day":   1,
		"poll_interval_seconds":  5,
		"max_sources_per_run":    20,
		"max_scrolls_per_source": 5,
		"max_scrolls_allowed":    10,
		"scroll_delay_seconds":   5,
		"duplicate_filter": map[string]any{
			"compare_against_previous_day": true,
			"method":                       "visible_text_matching",
			"parse_html":                   false,
		},
		"scheduled_windows": []any{
			map[string]any{
				"name":             "daily_default",
				"enabled":          true,
				"local_time_start": "09:00",
				"local_time_end":   "09:30",
				"days":             []any{"mon", "tue", "wed", "thu", "fri", "sat", "sun"},
			},
		},
		"clients": []any{},
	}
}

func writeConfigFile(path string, doc map[string]any) error {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func loadJob(cfg config) (map[string]any, error) {
	job := map[string]any{
		"run_id":       cfg.runID,
		"sources":      []any{},
		"created_at":   time.Now().UTC().Format(time.RFC3339),
		"job_file":     cfg.jobFile,
		"collector":    "media-agency-local-collector",
		"job_fallback": true,
	}
	if cfg.jobFile == "" {
		return job, nil
	}
	f, err := os.Open(cfg.jobFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := json.NewDecoder(io.LimitReader(f, maxJSONBodyBytes))
	dec.UseNumber()
	if err := dec.Decode(&job); err != nil {
		return nil, err
	}
	if cfg.runID != "" {
		job["run_id"] = cfg.runID
	}
	return job, nil
}

func loadCompletions(outputDir string) map[string]string {
	path := filepath.Join(outputDir, "completed_runs.json")
	f, err := os.Open(path)
	if err != nil {
		return map[string]string{}
	}
	defer f.Close()
	var out map[string]string
	if err := json.NewDecoder(io.LimitReader(f, maxJSONBodyBytes)).Decode(&out); err != nil {
		return map[string]string{}
	}
	if out == nil {
		return map[string]string{}
	}
	return out
}

func (b *bridge) saveCompletionsLocked() {
	if !b.cfg.persistent {
		return
	}
	data, err := json.MarshalIndent(b.completions, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(b.outputRoot, "completed_runs.json"), append(data, '\n'), 0o600)
}

func (b *bridge) refreshLocalControlFiles() {
	b.reloadConfigIfChanged()
	b.loadRunNowRequestIfPresent()
}

func (b *bridge) reloadConfigIfChanged() {
	if b.cfg.configFile == "" {
		return
	}
	info, err := os.Stat(b.cfg.configFile)
	if err != nil {
		return
	}
	b.mu.Lock()
	unchanged := info.ModTime().Equal(b.configFileModTime) && info.Size() == b.configFileSize
	b.mu.Unlock()
	if unchanged {
		return
	}
	doc, err := loadCollectorConfig(b.cfg)
	if err != nil {
		log.Printf("collector config reload skipped: %v", err)
		return
	}
	b.mu.Lock()
	b.configDoc = doc
	b.configFileModTime = info.ModTime()
	b.configFileSize = info.Size()
	b.mu.Unlock()
	_ = b.writeStatus("ready", "collector config reloaded from file")
	log.Printf("collector config reloaded from file=%s", b.cfg.configFile)
}

func (b *bridge) loadRunNowRequestIfPresent() {
	path := b.runNowRequestPath
	if path == "" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	sig := fmt.Sprintf("%d:%d", info.ModTime().UnixNano(), info.Size())
	b.mu.Lock()
	if b.lastRunNowRequestSig == sig {
		b.mu.Unlock()
		return
	}
	b.mu.Unlock()
	job, err := readMapFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("run-now request file not loaded: %v", err)
		}
		return
	}
	now := time.Now()
	if getString(job, "run_id", "") == "" {
		if requestID := getString(job, "request_id", ""); requestID != "" {
			job["run_id"] = requestID
		}
	}
	resp, err := b.activateRunNowJob(job, now, "file")
	status := map[string]any{
		"object":     "collector_run_now_request_status",
		"request_at": now.UTC().Format(time.RFC3339),
		"request":    filepath.Base(path),
	}
	if err != nil {
		status["ok"] = false
		status["error"] = err.Error()
		_ = writeMapFile(filepath.Join(filepath.Dir(path), "run_now_request_status.json"), status)
		log.Printf("run-now request file rejected: %v", err)
		return
	}
	for key, value := range resp {
		status[key] = value
	}
	consumedPath, consumeErr := consumeRunNowRequestFile(path, getString(resp, "run_id", now.Format("2006-01-02_150405")), now)
	status["consumed"] = consumeErr == nil
	if consumedPath != "" {
		status["consumed_path"] = consumedPath
	}
	if consumeErr != nil {
		status["consume_warning"] = consumeErr.Error()
		log.Printf("run-now request loaded but consume had warning: %v", consumeErr)
	}
	_ = writeMapFile(filepath.Join(filepath.Dir(path), "run_now_request_status.json"), status)
	b.mu.Lock()
	b.lastRunNowRequestSig = sig
	b.mu.Unlock()
}

func consumeRunNowRequestFile(path, runID string, now time.Time) (string, error) {
	consumedPath := filepath.Join(filepath.Dir(path), fmt.Sprintf(
		"run_now_request.%s.%d.consumed.json",
		safeFilename(runID),
		now.UTC().UnixNano(),
	))
	if err := os.Rename(path, consumedPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		if removeErr := os.Remove(path); removeErr == nil {
			return "", fmt.Errorf("move failed (%v); original removed to prevent replay", err)
		} else {
			return "", fmt.Errorf("move failed (%v); remove failed (%v)", err, removeErr)
		}
	}
	return consumedPath, nil
}

func (b *bridge) run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", b.withCORS(b.handleStatus))
	mux.HandleFunc("/config", b.withCORS(b.handleConfig))
	mux.HandleFunc("/jobs/current", b.withCORS(b.handleCurrentJob))
	mux.HandleFunc("/jobs/run_now", b.withCORS(b.handleRunNowJob))
	mux.HandleFunc("/collect/data_point", b.withCORS(b.requireToken(b.handleDataPoint)))
	mux.HandleFunc("/collect/lead", b.withCORS(b.requireToken(b.handleLead)))
	mux.HandleFunc("/collect/competitor", b.withCORS(b.requireToken(b.handleCompetitor)))
	mux.HandleFunc("/collect/new_private_source", b.withCORS(b.requireToken(b.handleNewPrivateSource)))
	mux.HandleFunc("/collect/source_status", b.withCORS(b.requireToken(b.handleSourceStatus)))
	mux.HandleFunc("/collect/snapshot", b.withCORS(b.requireToken(b.handleSnapshot)))
	mux.HandleFunc("/complete", b.withCORS(b.requireToken(b.handleComplete)))
	mux.HandleFunc("/shutdown", b.withCORS(b.requireToken(b.handleShutdown)))

	addr := fmt.Sprintf("%s:%d", b.cfg.host, b.cfg.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()

	b.server = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		if b.cfg.persistent {
			<-ctx.Done()
			b.shutdown("signal")
			return
		}
		select {
		case <-ctx.Done():
			b.shutdown("signal")
		case <-time.After(b.cfg.ttl):
			b.shutdown("ttl_expired")
		}
	}()

	log.Printf("media agency collector bridge listening on http://%s", addr)
	log.Printf("run_id=%s output_dir=%s ttl=%s", b.cfg.runID, b.cfg.outputDir, b.cfg.ttl)
	if b.cfg.persistent {
		log.Printf("persistent scheduler mode enabled config_file=%s", b.cfg.configFile)
	}
	if b.cfg.extensionID != "" {
		log.Printf("allowed extension origin=chrome-extension://%s", b.cfg.extensionID)
	} else {
		log.Printf("allowed extension origin=any chrome-extension:// origin")
	}
	return b.server.Serve(ln)
}

func (b *bridge) withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if b.cfg.verbose {
			log.Printf("%s %s origin=%s", r.Method, r.URL.Path, r.Header.Get("Origin"))
		}
		if !b.allowOrigin(w, r) {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodOptions && r.URL.Path == "/status" {
			b.recordExtensionCheck(r)
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func (b *bridge) allowOrigin(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	allowed := false
	if b.cfg.extensionID != "" {
		allowed = origin == "chrome-extension://"+b.cfg.extensionID
	} else {
		allowed = strings.HasPrefix(origin, "chrome-extension://")
	}
	if !allowed {
		return false
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Collector-Token, X-Collector-Extension, X-Collector-Extension-Version, Authorization")
	w.Header().Set("Access-Control-Max-Age", "300")
	return true
}

func (b *bridge) recordExtensionCheck(r *http.Request) {
	origin := r.Header.Get("Origin")
	header := r.Header.Get("X-Collector-Extension")
	version := r.Header.Get("X-Collector-Extension-Version")
	if !strings.HasPrefix(origin, "chrome-extension://") && header != "media-agency-local-collector" {
		return
	}
	if origin == "" {
		origin = "chrome-extension://detected-by-header"
	}
	now := time.Now().UTC()
	b.mu.Lock()
	b.extension.lastCheckAt = now
	b.extension.lastOrigin = origin
	b.extension.lastVersion = version
	b.extension.checkCount++
	b.mu.Unlock()
	_ = b.writeHealthFile()
}

func (b *bridge) requireToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		got := r.Header.Get("X-Collector-Token")
		if got == "" {
			got = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		}
		if got == "" || got != b.token {
			http.Error(w, "invalid collector token", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (b *bridge) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	b.refreshLocalControlFiles()
	jobAvailable := true
	activeRunID := b.cfg.runID
	status := "ready"
	currentJobType := "on_demand"
	if b.cfg.persistent {
		job, available, runID := b.currentPersistentJob(time.Now())
		jobAvailable = available
		activeRunID = runID
		currentJobType = "none"
		if available {
			currentJobType = "scheduled"
			if getBool(job, "run_now", false) {
				currentJobType = "run_now"
			}
		}
		if !available {
			status = "idle"
		}
	}
	b.mu.Lock()
	counts := cloneIntMap(b.counts)
	completed := b.completed
	if b.cfg.persistent {
		_, completed = b.completions[activeRunID]
	}
	outputDir := b.cfg.outputDir
	startedAt := b.startedAt.Format(time.RFC3339)
	ttlSeconds := int(b.cfg.ttl.Seconds())
	extension := b.extensionStatusLocked(time.Now().UTC())
	configFile := b.cfg.configFile
	configFileUpdatedAt := ""
	if !b.configFileModTime.IsZero() {
		configFileUpdatedAt = b.configFileModTime.Format(time.RFC3339)
	}
	runNowRequestFile := b.runNowRequestPath
	b.mu.Unlock()
	writeJSON(w, map[string]any{
		"object":                 "collector_bridge_status",
		"status":                 status,
		"run_id":                 activeRunID,
		"started_at":             startedAt,
		"ttl_seconds":            ttlSeconds,
		"persistent":             b.cfg.persistent,
		"current_job_type":       currentJobType,
		"output_dir":             outputDir,
		"job_available":          jobAvailable && !completed,
		"completed":              completed,
		"counts":                 counts,
		"extension_health":       extension,
		"config_file":            configFile,
		"config_file_updated_at": configFileUpdatedAt,
		"run_now_request_file":   runNowRequestFile,
	})
}

func (b *bridge) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		b.reloadConfigIfChanged()
		b.mu.Lock()
		doc := cloneMap(b.configDoc)
		b.mu.Unlock()
		writeJSON(w, doc)
	case http.MethodPost:
		next, err := readMap(r, maxJSONBodyBytes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		next = sanitizeMap(next)
		if _, ok := next["version"]; !ok {
			next["version"] = "0.1.0"
		}
		b.mu.Lock()
		b.configDoc = next
		configFile := b.cfg.configFile
		b.mu.Unlock()
		if configFile != "" {
			if err := writeConfigFile(configFile, next); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			modTime, size := fileStamp(configFile)
			b.mu.Lock()
			b.configFileModTime = modTime
			b.configFileSize = size
			b.mu.Unlock()
		}
		writeJSON(w, map[string]any{"ok": true, "object": "collector_config_saved"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (b *bridge) handleCurrentJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	b.refreshLocalControlFiles()
	job := cloneMap(b.job)
	if b.cfg.persistent {
		persistentJob, available, _ := b.currentPersistentJob(time.Now())
		if !available {
			http.Error(w, "no active scheduled job", http.StatusNoContent)
			return
		}
		job = persistentJob
	}
	job["collector_bridge"] = map[string]any{
		"object":      "collector_bridge_session",
		"base_url":    fmt.Sprintf("http://%s:%d", b.cfg.host, b.cfg.port),
		"run_id":      job["run_id"],
		"write_token": b.token,
		"ttl_seconds": int(b.cfg.ttl.Seconds()),
		"persistent":  b.cfg.persistent,
	}
	writeJSON(w, job)
}

func (b *bridge) handleRunNowJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	job, err := readMap(r, maxJSONBodyBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := b.activateRunNowJob(job, time.Now(), "api")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, resp)
}

func (b *bridge) activateRunNowJob(job map[string]any, now time.Time, source string) (map[string]any, error) {
	job = sanitizeMap(job)
	runID := getString(job, "run_id", fmt.Sprintf("%s_first_trial", now.Format("2006-01-02_150405")))
	job["run_id"] = runID
	job["collector"] = "media-agency-local-collector"
	job["created_at"] = now.UTC().Format(time.RFC3339)
	job["scheduled"] = false
	job["run_now"] = true
	ttlMinutes := clampInt(getInt(job, "run_now_ttl_minutes", 30), 1, 120)
	expiresAt := now.Add(time.Duration(ttlMinutes) * time.Minute).UTC().Format(time.RFC3339)
	job["run_now_ttl_minutes"] = ttlMinutes
	job["run_now_expires_at"] = expiresAt
	if _, ok := job["force"]; !ok {
		job["force"] = false
	}
	if _, ok := job["sources"]; !ok {
		job["sources"] = []any{}
	}
	if _, ok := job["pacing"]; !ok {
		b.mu.Lock()
		configDoc := cloneMap(b.configDoc)
		b.mu.Unlock()
		job["pacing"] = defaultPacing(configDoc)
	}
	if _, ok := job["collector_policy"]; !ok {
		job["collector_policy"] = defaultCollectorPolicy()
	}
	month := now.Format("2006-01")
	outputDir := filepath.Join(b.outputRoot, month, safeFilename(runID))
	job["output_dir"] = outputDir
	if err := os.MkdirAll(filepath.Join(outputDir, "snapshots"), 0o700); err != nil {
		return nil, err
	}

	b.mu.Lock()
	b.runNowJob = job
	b.cfg.runID = runID
	b.cfg.outputDir = outputDir
	b.completed = false
	b.counts = emptyCounts()
	delete(b.completions, runID)
	b.saveCompletionsLocked()
	b.mu.Unlock()
	_ = b.writeStatus("ready", fmt.Sprintf("run-now collector job loaded from %s", source))
	return map[string]any{
		"ok":                 true,
		"object":             "collector_run_now_loaded",
		"run_id":             runID,
		"output_dir":         outputDir,
		"run_now_expires_at": expiresAt,
		"source":             source,
	}, nil
}

func (b *bridge) currentPersistentJob(now time.Time) (map[string]any, bool, string) {
	b.mu.Lock()
	if b.runNowJob != nil {
		runNow := cloneMap(b.runNowJob)
		runID := getString(runNow, "run_id", b.cfg.runID)
		completed := b.completed
		expired := false
		if rawExpires := getString(runNow, "run_now_expires_at", ""); rawExpires != "" {
			if expiresAt, err := time.Parse(time.RFC3339, rawExpires); err == nil && now.UTC().After(expiresAt) {
				expired = true
				b.runNowJob = nil
				b.completed = false
			}
		}
		b.mu.Unlock()
		if expired {
			return nil, false, runID
		}
		if completed {
			return nil, false, runID
		}
		return runNow, true, runID
	}
	doc := cloneMap(b.configDoc)
	root := b.outputRoot
	completions := make(map[string]string, len(b.completions))
	for k, v := range b.completions {
		completions[k] = v
	}
	fallbackJob := cloneMap(b.job)
	b.mu.Unlock()

	window, ok := activeWindow(doc, now)
	if !ok {
		return nil, false, ""
	}
	windowName := getString(window, "name", "daily_default")
	runID := fmt.Sprintf("%s_%s", now.Format("2006-01-02"), safeFilename(windowName))
	if completions[runID] != "" {
		return nil, false, runID
	}

	sources := configuredSources(doc)
	if len(sources) == 0 {
		if fallbackSources, ok := fallbackJob["sources"].([]any); ok {
			sources = fallbackSources
		}
	}
	if len(sources) == 0 {
		return nil, false, runID
	}

	maxAllowed := clampInt(getInt(doc, "max_scrolls_allowed", 10), 1, 10)
	scrolls := clampInt(getInt(doc, "max_scrolls_per_source", 5), 0, maxAllowed)
	delay := clampInt(getInt(doc, "scroll_delay_seconds", 5), 5, 60)
	maxSources := clampInt(getInt(doc, "max_sources_per_run", 20), 1, 20)
	month := now.Format("2006-01")
	outputDir := filepath.Join(root, month, runID)
	_ = os.MkdirAll(filepath.Join(outputDir, "snapshots"), 0o700)

	job := map[string]any{
		"run_id":        runID,
		"collector":     "media-agency-local-collector",
		"created_at":    now.UTC().Format(time.RFC3339),
		"scheduled":     true,
		"schedule_name": windowName,
		"output_dir":    outputDir,
		"sources":       sources,
		"pacing": map[string]any{
			"min_delay_seconds": delay,
			"max_delay_seconds": delay,
			"max_sources":       maxSources,
			"scroll_steps":      scrolls,
			"max_text_chars":    12000,
		},
		"collector_policy": map[string]any{
			"read_only":                     true,
			"do_not_message":                true,
			"do_not_comment":                true,
			"do_not_react":                  true,
			"do_not_scrape_contact_details": true,
		},
	}

	b.mu.Lock()
	if b.cfg.runID != runID {
		b.counts = emptyCounts()
	}
	b.cfg.runID = runID
	b.cfg.outputDir = outputDir
	b.completed = false
	b.mu.Unlock()
	return job, true, runID
}

func activeWindow(doc map[string]any, now time.Time) (map[string]any, bool) {
	for _, item := range asSlice(doc["scheduled_windows"]) {
		win, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if enabled, ok := win["enabled"].(bool); ok && !enabled {
			continue
		}
		if !dayMatches(win["days"], now.Weekday()) {
			continue
		}
		start, okStart := minutesOfDay(getString(win, "local_time_start", "09:00"))
		end, okEnd := minutesOfDay(getString(win, "local_time_end", "09:30"))
		if !okStart || !okEnd {
			continue
		}
		current := now.Hour()*60 + now.Minute()
		if start <= end {
			if current >= start && current < end {
				return win, true
			}
			continue
		}
		if current >= start || current < end {
			return win, true
		}
	}
	return nil, false
}

func defaultPacing(doc map[string]any) map[string]any {
	maxAllowed := clampInt(getInt(doc, "max_scrolls_allowed", 10), 1, 10)
	scrolls := clampInt(getInt(doc, "max_scrolls_per_source", 5), 0, maxAllowed)
	delay := clampInt(getInt(doc, "scroll_delay_seconds", 5), 5, 60)
	maxSources := clampInt(getInt(doc, "max_sources_per_run", 20), 1, 20)
	return map[string]any{
		"min_delay_seconds": delay,
		"max_delay_seconds": delay,
		"max_sources":       maxSources,
		"scroll_steps":      scrolls,
		"max_text_chars":    12000,
	}
}

func defaultCollectorPolicy() map[string]any {
	return map[string]any{
		"read_only":                     true,
		"do_not_message":                true,
		"do_not_comment":                true,
		"do_not_react":                  true,
		"do_not_scrape_contact_details": true,
	}
}

func configuredSources(doc map[string]any) []any {
	var out []any
	appendSources := func(sourceItems []any, client map[string]any) {
		for _, item := range sourceItems {
			src, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if getBool(src, "enabled", true) == false {
				continue
			}
			next := cloneMap(src)
			if client != nil {
				copyDefault(next, "client_slug", getString(client, "client_slug", ""))
				copyDefault(next, "business_slug", getString(client, "business_slug", ""))
				copyDefault(next, "location_slug", getString(client, "location_slug", ""))
				copyDefault(next, "language", getString(client, "language", ""))
			}
			if getString(next, "source_type", "") == "" {
				next["source_type"] = "private"
			}
			out = append(out, next)
		}
	}
	appendSources(asSlice(doc["private_sources"]), nil)
	appendSources(asSlice(doc["sources"]), nil)
	for _, item := range asSlice(doc["clients"]) {
		client, ok := item.(map[string]any)
		if !ok || !getBool(client, "enabled", true) {
			continue
		}
		appendSources(asSlice(client["private_sources"]), client)
		appendSources(asSlice(client["sources"]), client)
	}
	return out
}

func copyDefault(dst map[string]any, key, value string) {
	if value == "" {
		return
	}
	if getString(dst, key, "") == "" {
		dst[key] = value
	}
}

func dayMatches(raw any, day time.Weekday) bool {
	items := asSlice(raw)
	if len(items) == 0 {
		return true
	}
	current := strings.ToLower(day.String()[:3])
	for _, item := range items {
		if strings.ToLower(fmt.Sprint(item)) == current {
			return true
		}
	}
	return false
}

func minutesOfDay(raw string) (int, bool) {
	parts := strings.Split(raw, ":")
	if len(parts) != 2 {
		return 0, false
	}
	hour, errHour := parseSmallInt(parts[0])
	minute, errMinute := parseSmallInt(parts[1])
	if errHour != nil || errMinute != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}

func parseSmallInt(raw string) (int, error) {
	var out int
	_, err := fmt.Sscanf(strings.TrimSpace(raw), "%d", &out)
	return out, err
}

func (b *bridge) handleDataPoint(w http.ResponseWriter, r *http.Request) {
	b.appendRecord(w, r, "data_point", "private_data_points.jsonl", "data_points")
}

func (b *bridge) handleLead(w http.ResponseWriter, r *http.Request) {
	b.appendRecord(w, r, "lead", "leads.jsonl", "leads")
}

func (b *bridge) handleCompetitor(w http.ResponseWriter, r *http.Request) {
	b.appendRecord(w, r, "competitor", "competitors.jsonl", "competitors")
}

func (b *bridge) handleNewPrivateSource(w http.ResponseWriter, r *http.Request) {
	b.appendRecord(w, r, "new_private_source", "new_private_sources.jsonl", "new_private_sources")
}

func (b *bridge) handleSourceStatus(w http.ResponseWriter, r *http.Request) {
	b.appendRecord(w, r, "source_status", "source_status.jsonl", "source_statuses")
}

func (b *bridge) appendRecord(w http.ResponseWriter, r *http.Request, kind, filename, countKey string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	record, err := readMap(r, maxJSONBodyBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	record = sanitizeMap(record)
	activeRunID, err := b.activeWriteRunID(record)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	record["object"] = "collector_" + kind
	record["record_type"] = kind
	record["run_id"] = activeRunID
	record["received_at"] = time.Now().UTC().Format(time.RFC3339)
	record["bridge_runtime"] = runtime.GOOS + "/" + runtime.GOARCH

	if err := b.writeJSONL(filename, record); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	b.increment(countKey)
	writeJSON(w, map[string]any{"ok": true, "record_type": kind})
}

func (b *bridge) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	record, err := readMap(r, maxHTMLBodyBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	name, _ := record["filename"].(string)
	if name == "" {
		name = "snapshot.html"
	}
	name = safeFilename(name)
	if !strings.HasSuffix(name, ".html") {
		name += ".html"
	}
	html, _ := record["html"].(string)
	if html == "" {
		http.Error(w, "missing html", http.StatusBadRequest)
		return
	}
	meta := sanitizeMap(record)
	activeRunID, err := b.activeWriteRunID(meta)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	path := filepath.Join(b.cfg.outputDir, "snapshots", name)
	if err := os.WriteFile(path, []byte(html), 0o600); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	delete(meta, "html")
	meta["snapshot_path"] = path
	meta["record_type"] = "snapshot"
	meta["run_id"] = activeRunID
	meta["received_at"] = time.Now().UTC().Format(time.RFC3339)
	if err := b.writeJSONL("source_status.jsonl", meta); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	b.increment("snapshots")
	writeJSON(w, map[string]any{"ok": true, "snapshot_path": path})
}

func (b *bridge) activeWriteRunID(record map[string]any) (string, error) {
	incomingRunID := getString(record, "run_id", "")
	b.mu.Lock()
	persistent := b.cfg.persistent
	cfgRunID := b.cfg.runID
	completed := b.completed
	b.mu.Unlock()

	if !persistent {
		if completed {
			return cfgRunID, errors.New("collector run is already completed")
		}
		if incomingRunID != "" && incomingRunID != cfgRunID {
			return cfgRunID, fmt.Errorf("stale collector run %q; active run is %q", incomingRunID, cfgRunID)
		}
		return cfgRunID, nil
	}

	_, available, activeRunID := b.currentPersistentJob(time.Now())
	if !available || activeRunID == "" {
		return activeRunID, errors.New("no active collector job")
	}
	if incomingRunID != "" && incomingRunID != activeRunID {
		return activeRunID, fmt.Errorf("stale collector run %q; active run is %q", incomingRunID, activeRunID)
	}
	return activeRunID, nil
}

func (b *bridge) handleComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	b.mu.Lock()
	b.completed = true
	runID := b.cfg.runID
	if b.cfg.persistent && runID != "" {
		b.completions[runID] = time.Now().UTC().Format(time.RFC3339)
		if b.runNowJob != nil && getString(b.runNowJob, "run_id", "") == runID {
			b.runNowJob = nil
		}
		b.saveCompletionsLocked()
	}
	persistent := b.cfg.persistent
	b.mu.Unlock()
	_ = b.writeStatus("completed", "collector marked run complete")
	writeJSON(w, map[string]any{"ok": true, "status": "completed"})
	if persistent {
		return
	}
	go func() {
		time.Sleep(500 * time.Millisecond)
		b.shutdown("completed")
	}()
}

func (b *bridge) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "status": "shutting_down"})
	go func() {
		time.Sleep(500 * time.Millisecond)
		b.shutdown("requested")
	}()
}

func (b *bridge) writeJSONL(filename string, record map[string]any) error {
	path := filepath.Join(b.cfg.outputDir, filename)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	return enc.Encode(record)
}

func (b *bridge) increment(key string) {
	b.mu.Lock()
	b.counts[key]++
	b.mu.Unlock()
	_ = b.writeStatus("ready", "record received")
}

func (b *bridge) writeStatus(status, message string) error {
	b.mu.Lock()
	counts := cloneIntMap(b.counts)
	completed := b.completed
	extension := b.extensionStatusLocked(time.Now().UTC())
	b.mu.Unlock()
	payload := map[string]any{
		"object":           "collector_status",
		"status":           status,
		"message":          message,
		"run_id":           b.cfg.runID,
		"started_at":       b.startedAt.Format(time.RFC3339),
		"updated_at":       time.Now().UTC().Format(time.RFC3339),
		"output_dir":       b.cfg.outputDir,
		"job_file":         b.cfg.jobFile,
		"completed":        completed,
		"counts":           counts,
		"bridge_host":      b.cfg.host,
		"bridge_port":      b.cfg.port,
		"persistent":       b.cfg.persistent,
		"extension_health": extension,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(b.cfg.outputDir, "collector_status.json"), append(data, '\n'), 0o600)
}

func (b *bridge) writeHealthFile() error {
	b.mu.Lock()
	payload := map[string]any{
		"object":           "collector_bridge_health",
		"status":           "running",
		"updated_at":       time.Now().UTC().Format(time.RFC3339),
		"started_at":       b.startedAt.Format(time.RFC3339),
		"bridge_host":      b.cfg.host,
		"bridge_port":      b.cfg.port,
		"persistent":       b.cfg.persistent,
		"config_file":      b.cfg.configFile,
		"output_root":      b.outputRoot,
		"extension_health": b.extensionStatusLocked(time.Now().UTC()),
	}
	b.mu.Unlock()
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(b.outputRoot, "bridge_health.json"), append(data, '\n'), 0o600)
}

func (b *bridge) extensionStatusLocked(now time.Time) map[string]any {
	pollSeconds := getInt(b.configDoc, "poll_interval_seconds", 5)
	staleAfter := pollSeconds*3 + 10
	if staleAfter < 25 {
		staleAfter = 25
	}
	out := map[string]any{
		"status":                         "no_extension_check_yet",
		"last_extension_check_at":        "",
		"seconds_since_last_check":       nil,
		"extension_check_count":          b.extension.checkCount,
		"last_extension_origin":          b.extension.lastOrigin,
		"extension_version":              b.extension.lastVersion,
		"expected_poll_seconds":          pollSeconds,
		"stale_after_seconds":            staleAfter,
		"possible_missing_reasons":       []string{"Chrome is closed", "extension is not installed", "extension is disabled or removed", "bridge URL/port mismatch", "Chrome service worker is sleeping"},
		"can_collect_when_chrome_closed": false,
	}
	if b.extension.lastCheckAt.IsZero() {
		return out
	}
	seconds := int(now.Sub(b.extension.lastCheckAt).Seconds())
	out["last_extension_check_at"] = b.extension.lastCheckAt.Format(time.RFC3339)
	out["seconds_since_last_check"] = seconds
	if seconds <= staleAfter {
		out["status"] = "recent"
		out["possible_missing_reasons"] = []string{}
	} else {
		out["status"] = "stale"
	}
	return out
}

func (b *bridge) shutdown(reason string) {
	b.mu.Lock()
	if b.stopping {
		b.mu.Unlock()
		return
	}
	b.stopping = true
	b.mu.Unlock()
	_ = b.writeStatus("stopping", reason)
	if b.server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = b.server.Shutdown(ctx)
}

func readMap(r *http.Request, limit int64) (map[string]any, error) {
	defer r.Body.Close()
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, limit))
	dec.UseNumber()
	var record map[string]any
	if err := dec.Decode(&record); err != nil {
		return nil, err
	}
	return record, nil
}

func readMapFile(path string) (map[string]any, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := json.NewDecoder(io.LimitReader(f, maxJSONBodyBytes))
	dec.UseNumber()
	var record map[string]any
	if err := dec.Decode(&record); err != nil {
		return nil, err
	}
	return record, nil
}

func writeMapFile(path string, payload map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(payload)
}

func randomToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneIntMap(in map[string]int) map[string]int {
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func emptyCounts() map[string]int {
	return map[string]int{
		"data_points":         0,
		"leads":               0,
		"competitors":         0,
		"new_private_sources": 0,
		"source_statuses":     0,
		"snapshots":           0,
	}
}

func asSlice(v any) []any {
	switch typed := v.(type) {
	case []any:
		return typed
	default:
		return nil
	}
}

func getString(m map[string]any, key, fallback string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return fallback
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" {
		return fallback
	}
	return s
}

func getBool(m map[string]any, key string, fallback bool) bool {
	v, ok := m[key]
	if !ok {
		return fallback
	}
	switch typed := v.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "y":
			return true
		case "false", "0", "no", "n":
			return false
		}
	}
	return fallback
}

func getInt(m map[string]any, key string, fallback int) int {
	v, ok := m[key]
	if !ok {
		return fallback
	}
	switch typed := v.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		n, err := typed.Int64()
		if err == nil {
			return int(n)
		}
	case string:
		n, err := parseSmallInt(typed)
		if err == nil {
			return n
		}
	}
	return fallback
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func sanitizeMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		if isSensitiveKey(k) {
			out[k] = "[redacted]"
			continue
		}
		out[k] = sanitizeValue(k, v)
	}
	return out
}

func sanitizeValue(key string, v any) any {
	switch typed := v.(type) {
	case map[string]any:
		return sanitizeMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, sanitizeValue(key, item))
		}
		return out
	case string:
		if isSensitiveKey(key) {
			return "[redacted]"
		}
		if looksLikeURL(typed) {
			return sanitizeURL(typed)
		}
		return redactSecretsInString(typed)
	default:
		return typed
	}
}

func isSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	needles := []string{"cookie", "token", "secret", "password", "passwd", "pwd", "otp", "authorization", "auth", "session", "bearer", "csrf", "xsrf"}
	for _, n := range needles {
		if strings.Contains(k, n) {
			return true
		}
	}
	return false
}

func looksLikeURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func sanitizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return raw
	}
	q := u.Query()
	for key := range q {
		lk := strings.ToLower(key)
		if strings.HasPrefix(lk, "utm_") || lk == "fbclid" || lk == "gclid" || lk == "msclkid" || isSensitiveKey(lk) || strings.Contains(lk, "access") {
			q.Del(key)
		}
	}
	u.RawQuery = q.Encode()
	u.Fragment = ""
	return u.String()
}

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)bearer\s+[a-z0-9._~+/=-]+`),
	regexp.MustCompile(`(?i)(access_token|id_token|refresh_token|sessionid|csrftoken)=([^&\s]+)`),
}

func redactSecretsInString(s string) string {
	out := s
	for _, re := range secretPatterns {
		out = re.ReplaceAllString(out, "[redacted]")
	}
	return out
}

func safeFilename(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, string(filepath.Separator), "_")
	name = regexp.MustCompile(`[^a-zA-Z0-9._-]+`).ReplaceAllString(name, "_")
	name = strings.Trim(name, "._-")
	if name == "" {
		return "snapshot.html"
	}
	if len(name) > 120 {
		name = name[:120]
	}
	return name
}
