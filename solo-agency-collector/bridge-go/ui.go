package main

// ui.go — the local operator UI (U1, read-only) per docs/UI_DESIGN.md.
//
// Principles enforced here:
//   - Read-only: this file never writes into the data root except the single
//     bridge/ui_token file. All mutating surfaces arrive in U2 via ui_inbox/.
//   - UI failure must never break the collector role: initUI errors are logged
//     and the extension endpoints keep working.
//   - Agents never fetch these URLs; the human's browser does. Auth is a local
//     token cookie so other local pages cannot read operator data.

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const uiCookieName = "sa_ui"

// ---------- data root ----------

// deriveDataRoot finds the daily-content-pipeline root from the bridge config:
// prefer the collector_config.json location (…/daily-content-pipeline/collector/x.json),
// else walk up from the output dir until a plausible pipeline root is found.
func deriveDataRoot(cfg config) string {
	if cfg.configFile != "" {
		if abs, err := filepath.Abs(filepath.Dir(filepath.Dir(cfg.configFile))); err == nil {
			return abs
		}
	}
	dir, err := filepath.Abs(cfg.outputDir)
	if err != nil {
		return ""
	}
	for i := 0; i < 6; i++ {
		if filepath.Base(dir) == "daily-content-pipeline" {
			return dir
		}
		if st, err := os.Stat(filepath.Join(dir, "clients")); err == nil && st.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// ---------- init + auth ----------

func (b *bridge) initUI() error {
	root := deriveDataRoot(b.cfg)
	if root == "" {
		return fmt.Errorf("ui: could not derive data root from config/output paths")
	}
	b.uiDataRoot = root
	tokenPath := filepath.Join(root, "bridge", "ui_token")
	if data, err := os.ReadFile(tokenPath); err == nil {
		tok := strings.TrimSpace(string(data))
		if len(tok) >= 16 {
			b.uiToken = tok
		}
	}
	if b.uiToken == "" {
		raw := make([]byte, 16)
		if _, err := rand.Read(raw); err != nil {
			return err
		}
		b.uiToken = hex.EncodeToString(raw)
		if err := os.MkdirAll(filepath.Dir(tokenPath), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(tokenPath, []byte(b.uiToken+"\n"), 0o600); err != nil {
			return err
		}
	}
	log.Printf("ui: enabled — entry http://%s:%d/ui/enter/%s (token file %s)",
		b.cfg.host, b.cfg.port, b.uiToken, tokenPath)
	return nil
}

func (b *bridge) registerUIRoutes(mux *http.ServeMux) {
	if err := b.initUI(); err != nil {
		log.Printf("ui: disabled — %v (collector endpoints unaffected)", err)
		return
	}
	mux.HandleFunc("/ui/enter/", b.handleUIEnter)
	// stylesheet is served unauthenticated so the locked page renders styled
	mux.HandleFunc("/ui/assets/pico.min.css", handleUIPicoCSS)
	mux.HandleFunc("/ui", b.uiAuth(b.handleUIHome))
	mux.HandleFunc("/ui/", b.uiAuth(b.handleUIRouter))
	mux.HandleFunc("/files/", b.uiAuth(b.handleUIFiles))
	mux.HandleFunc("/events", b.uiAuth(b.handleUIEvents))
	mux.HandleFunc("/api/ui/", b.uiAuth(b.handleUIAPI))
}

func (b *bridge) handleUIEnter(w http.ResponseWriter, r *http.Request) {
	tok := strings.TrimPrefix(r.URL.Path, "/ui/enter/")
	if tok == "" || tok != b.uiToken {
		http.Error(w, "invalid entry token", http.StatusForbidden)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: uiCookieName, Value: tok, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/ui", http.StatusFound)
}

func (b *bridge) uiAuthorized(r *http.Request) bool {
	c, err := r.Cookie(uiCookieName)
	return err == nil && c.Value == b.uiToken
}

func (b *bridge) uiAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Same-origin only: browser same-origin requests carry no Origin on GET;
		// refuse any cross-origin caller outright.
		if o := r.Header.Get("Origin"); o != "" && o != fmt.Sprintf("http://%s:%d", b.cfg.host, b.cfg.port) {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}
		// U2: mutations exist only under /api/ui/ and land exclusively in ui_inbox/.
		if r.Method != http.MethodGet &&
			!(r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/ui/")) {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !b.uiAuthorized(r) {
			w.WriteHeader(http.StatusForbidden)
			_ = uiTpl.ExecuteTemplate(w, "locked", map[string]any{"Title": "Locked"})
			return
		}
		next(w, r)
	}
}

// ---------- /files/ (read-only static serving with guardrails) ----------

// uiFilesDenied blocks secret-bearing paths from ever being served.
func uiFilesDenied(rel string) bool {
	rel = strings.ToLower(filepath.ToSlash(rel))
	base := path.Base(rel)
	if base == "credentials.json" || base == "token.json" || base == "ui_token" {
		return true
	}
	if strings.HasPrefix(base, "provider_config.local") {
		return true
	}
	for _, seg := range strings.Split(rel, "/") {
		if seg == "secrets" {
			return true
		}
	}
	return false
}

// uiResolveFile maps /files/<rel> to an absolute path inside root, rejecting
// traversal and denied names. Returns "" when the request must be refused.
func uiResolveFile(root, urlPath string) string {
	rel := strings.TrimPrefix(urlPath, "/files/")
	rel = path.Clean("/" + rel)[1:] // collapse ../ tricks against the virtual root
	if rel == "" || rel == "." || strings.Contains(rel, "\x00") {
		return ""
	}
	if uiFilesDenied(rel) {
		return ""
	}
	full := filepath.Join(root, filepath.FromSlash(rel))
	rootSep := strings.TrimSuffix(root, string(filepath.Separator)) + string(filepath.Separator)
	if !strings.HasPrefix(full, rootSep) {
		return ""
	}
	st, err := os.Stat(full)
	if err != nil || st.IsDir() {
		return ""
	}
	return full
}

func (b *bridge) handleUIFiles(w http.ResponseWriter, r *http.Request) {
	full := uiResolveFile(b.uiDataRoot, r.URL.Path)
	if full == "" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, full)
}

// ---------- SSE change feed ----------

// handleUIEvents emits a "change" event whenever a watched directory's
// fingerprint moves. Cheap mtime polling only — no fsnotify dependency.
func (b *bridge) handleUIEvents(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	last := b.uiFingerprint()
	fmt.Fprintf(w, "event: hello\ndata: %q\n\n", last)
	fl.Flush()
	tick := time.NewTicker(2 * time.Second)
	heartbeat := time.NewTicker(25 * time.Second)
	defer tick.Stop()
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			fmt.Fprint(w, ": ping\n\n")
			fl.Flush()
		case <-tick.C:
			cur := b.uiFingerprint()
			if cur != last {
				last = cur
				fmt.Fprintf(w, "event: change\ndata: %q\n\n", cur)
				fl.Flush()
			}
		}
	}
}

func (b *bridge) uiFingerprint() string {
	var sb strings.Builder
	stamp := func(p string) {
		if st, err := os.Stat(p); err == nil {
			fmt.Fprintf(&sb, "%s=%d;", p, st.ModTime().UnixNano())
		}
	}
	root := b.uiDataRoot
	for _, d := range []string{"pending", "claimed", "completed"} {
		stamp(filepath.Join(root, "collector", "jobs", d))
	}
	stamp(filepath.Join(root, "collector", "inbox"))
	for _, ws := range b.uiClients() {
		stamp(filepath.Join(ws.Path, "outputs"))
		stamp(filepath.Join(ws.Path, "outreach", "outputs"))
		stamp(filepath.Join(ws.Path, "outreach", "ui_inbox"))
		stamp(filepath.Join(ws.Path, "ui_inbox"))
		stamp(filepath.Join(ws.Path, "history", "discovery_shortlist.json"))
		// pending_approval holds YYYY-MM-DD subdirs; stamp those so new drafts refresh the page
		campaigns := filepath.Join(ws.Path, "outreach", "campaigns")
		if camps, err := os.ReadDir(campaigns); err == nil {
			for _, camp := range camps {
				if !camp.IsDir() {
					continue
				}
				pa := filepath.Join(campaigns, camp.Name(), "outbox", "pending_approval")
				stamp(pa)
				if days, err := os.ReadDir(pa); err == nil {
					for _, day := range days {
						stamp(filepath.Join(pa, day.Name()))
					}
				}
			}
		}
	}
	return sb.String()
}

// ---------- data readers (read-only) ----------

type uiClient struct {
	Slug      string
	Workspace string
	Path      string
}

func (b *bridge) uiClients() []uiClient {
	var out []uiClient
	base := filepath.Join(b.uiDataRoot, "clients")
	slugs, err := os.ReadDir(base)
	if err != nil {
		return out
	}
	for _, s := range slugs {
		if !s.IsDir() {
			continue
		}
		subs, err := os.ReadDir(filepath.Join(base, s.Name()))
		if err != nil {
			continue
		}
		for _, ws := range subs {
			if !ws.IsDir() {
				continue
			}
			out = append(out, uiClient{
				Slug:      s.Name(),
				Workspace: ws.Name(),
				Path:      filepath.Join(base, s.Name(), ws.Name()),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out
}

func (b *bridge) uiFindClient(slug string) (uiClient, bool) {
	for _, c := range b.uiClients() {
		if c.Slug == slug {
			return c, true
		}
	}
	return uiClient{}, false
}

type uiFile struct {
	Name    string
	Rel     string // data-root-relative, for /files/ links
	ModTime time.Time
	Size    int64
}

// uiListFiles walks base (bounded depth) collecting files with the given
// extensions, newest first, capped.
func (b *bridge) uiListFiles(base string, exts []string, cap int) []uiFile {
	var out []uiFile
	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		if depth > 4 || len(out) > cap*3 {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			p := filepath.Join(dir, e.Name())
			if e.IsDir() {
				walk(p, depth+1)
				continue
			}
			keep := false
			for _, x := range exts {
				if strings.HasSuffix(strings.ToLower(e.Name()), x) {
					keep = true
					break
				}
			}
			if !keep {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			rel, err := filepath.Rel(b.uiDataRoot, p)
			if err != nil || uiFilesDenied(rel) {
				continue
			}
			out = append(out, uiFile{Name: e.Name(), Rel: filepath.ToSlash(rel), ModTime: info.ModTime(), Size: info.Size()})
		}
	}
	walk(base, 0)
	sort.Slice(out, func(i, j int) bool { return out[i].ModTime.After(out[j].ModTime) })
	if len(out) > cap {
		out = out[:cap]
	}
	return out
}

func uiReadJSON(path string, into any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, into)
}

type uiJob struct {
	State   string
	Name    string
	ModTime time.Time
	RunID   string
	Client  string
	Kind    string
}

func (b *bridge) uiJobs() []uiJob {
	var out []uiJob
	for _, state := range []string{"pending", "claimed", "completed"} {
		dir := filepath.Join(b.uiDataRoot, "collector", "jobs", state)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		var batch []uiJob
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			j := uiJob{State: state, Name: e.Name(), ModTime: info.ModTime()}
			var doc map[string]any
			if uiReadJSON(filepath.Join(dir, e.Name()), &doc) == nil {
				j.RunID, _ = doc["run_id"].(string)
				j.Client, _ = doc["client_slug"].(string)
				if v, ok := doc["job_type"].(string); ok {
					j.Kind = v
				} else if v, ok := doc["purpose"].(string); ok {
					j.Kind = v
				}
			}
			batch = append(batch, j)
		}
		sort.Slice(batch, func(i, j int) bool { return batch[i].ModTime.After(batch[j].ModTime) })
		if state == "completed" && len(batch) > 30 {
			batch = batch[:30]
		}
		out = append(out, batch...)
	}
	return out
}

type uiSendbox struct {
	Client string
	Slug   string
	Email  string
	Status string
	Quota  string
	Warmup string
}

func (b *bridge) uiSendboxes() []uiSendbox {
	var out []uiSendbox
	for _, c := range b.uiClients() {
		var doc struct {
			Sendboxes []map[string]any `json:"sendboxes"`
		}
		p := filepath.Join(c.Path, "outreach", "sendboxes", "sendboxes.json")
		if uiReadJSON(p, &doc) != nil {
			continue
		}
		for _, sb := range doc.Sendboxes {
			row := uiSendbox{Client: c.Slug}
			row.Slug, _ = sb["slug"].(string)
			row.Email, _ = sb["email"].(string)
			row.Status, _ = sb["status"].(string)
			row.Warmup, _ = sb["warmup_stage"].(string)
			for _, k := range []string{"quota_today", "daily_quota"} {
				if v, ok := sb[k]; ok {
					row.Quota = fmt.Sprintf("%v", v)
					break
				}
			}
			out = append(out, row)
		}
	}
	return out
}

type uiContact struct {
	ID       string
	Name     string
	Email    string
	Vertical string
	Stage    string
}

func (b *bridge) uiContacts(c uiClient, cap int) []uiContact {
	dir := filepath.Join(c.Path, "outreach", "crm", "contacts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []uiContact
	for _, e := range entries {
		if len(out) >= cap || e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		var doc map[string]any
		if uiReadJSON(filepath.Join(dir, e.Name()), &doc) != nil {
			continue
		}
		ct := uiContact{ID: strings.TrimSuffix(e.Name(), ".json")}
		for _, k := range []string{"display_name", "full_name", "name"} {
			if v, ok := doc[k].(string); ok && v != "" {
				ct.Name = v
				break
			}
		}
		if ids, ok := doc["identities"].(map[string]any); ok {
			if emails, ok := ids["emails"].([]any); ok && len(emails) > 0 {
				if em, ok := emails[0].(map[string]any); ok {
					ct.Email, _ = em["address"].(string)
				}
			}
		}
		if cf, ok := doc["custom_fields"].(map[string]any); ok {
			if v, ok := cf["professional_vertical"].(string); ok {
				ct.Vertical = v
			}
		}
		ct.Stage, _ = doc["lifecycle_stage"].(string)
		out = append(out, ct)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

type uiDeal struct {
	ID      string
	Stage   string
	Contact string
	Title   string
}

func (b *bridge) uiDeals(c uiClient) []uiDeal {
	dir := filepath.Join(c.Path, "outreach", "crm", "deals")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []uiDeal
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		var doc map[string]any
		if uiReadJSON(filepath.Join(dir, e.Name()), &doc) != nil {
			continue
		}
		d := uiDeal{ID: strings.TrimSuffix(e.Name(), ".json")}
		d.Stage, _ = doc["stage"].(string)
		d.Contact, _ = doc["contact_id"].(string)
		d.Title, _ = doc["title"].(string)
		out = append(out, d)
	}
	return out
}

// ---------- handlers ----------

func (b *bridge) handleUIHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/ui" {
		http.NotFound(w, r)
		return
	}
	b.uiRender(w, "home", map[string]any{
		"Title":   "Solo Agency",
		"Clients": b.uiClients(),
		"Jobs":    b.uiJobs(),
	})
}

// handleUIRouter dispatches /ui/... subpaths.
func (b *bridge) handleUIRouter(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/ui/"), "/"), "/")
	switch {
	case len(parts) == 1 && parts[0] == "jobs":
		b.uiRender(w, "jobs", map[string]any{"Title": "Jobs", "Jobs": b.uiJobs(), "Active": b.uiActiveRuns()})
	case len(parts) == 1 && parts[0] == "status":
		b.uiRenderStatus(w)
	case len(parts) == 1 && parts[0] != "":
		b.uiRenderClient(w, parts[0])
	case len(parts) == 2 && parts[1] == "reports":
		b.uiRenderReports(w, parts[0])
	case len(parts) == 2 && parts[1] == "crm":
		b.uiRenderCRM(w, parts[0])
	case len(parts) == 2 && parts[1] == "approvals":
		b.uiRenderApprovals(w, parts[0])
	case len(parts) == 2 && parts[1] == "shortlist":
		b.uiRenderShortlist(w, parts[0])
	case len(parts) == 2 && parts[1] == "sendboxes":
		b.uiRenderSendboxes(w, parts[0])
	default:
		http.NotFound(w, r)
	}
}

func handleUIPicoCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write([]byte(picoCSS))
}

// uiClientSendboxes reads {ws}/outreach/sendboxes/sendboxes.json for one client.
func (b *bridge) uiClientSendboxes(c uiClient) []map[string]any {
	p := filepath.Join(c.Path, "outreach", "sendboxes", "sendboxes.json")
	if m, err := readJSONFile(p); err == nil {
		return mapsOf(mList(m, "sendboxes"))
	}
	return nil
}

func (b *bridge) uiRenderSendboxes(w http.ResponseWriter, slug string) {
	c, ok := b.uiFindClient(slug)
	if !ok {
		http.Error(w, "unknown client", http.StatusNotFound)
		return
	}
	b.uiRender(w, "sendboxes", map[string]any{
		"Title": c.Slug + " sendboxes", "Client": c, "Sendboxes": b.uiClientSendboxes(c),
	})
}

// ---------- U2: interactive approvals + shortlist (writes go to ui_inbox only) ----------

type uiDraft struct {
	ID        string
	Campaign  string
	Step      any
	To        string
	Subject   string
	Body      string
	Band      string
	Warnings  []string
	Companion string
	Hooks     []map[string]any
}

func (b *bridge) uiPendingDrafts(c uiClient) []uiDraft {
	var out []uiDraft
	campaignsDir := filepath.Join(c.Path, "outreach", "campaigns")
	camps, err := os.ReadDir(campaignsDir)
	if err != nil {
		return out
	}
	for _, camp := range camps {
		if !camp.IsDir() {
			continue
		}
		base := filepath.Join(campaignsDir, camp.Name(), "outbox", "pending_approval")
		_ = filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
				return nil
			}
			var doc map[string]any
			if uiReadJSON(p, &doc) != nil {
				return nil
			}
			if s, _ := doc["status"].(string); s != "pending_approval" {
				return nil
			}
			dr := uiDraft{Campaign: camp.Name()}
			dr.ID, _ = doc["id"].(string)
			dr.Step = doc["step"]
			dr.To, _ = doc["to"].(string)
			dr.Subject, _ = doc["subject"].(string)
			dr.Body, _ = doc["body_text"].(string)
			dr.Band, _ = doc["confidence_band"].(string)
			dr.Companion, _ = doc["companion_url"].(string)
			if ws, ok := doc["warnings"].([]any); ok {
				for _, wv := range ws {
					if s, ok := wv.(string); ok {
						dr.Warnings = append(dr.Warnings, s)
					}
				}
			}
			if hs, ok := doc["hooks_used"].([]any); ok {
				for _, hv := range hs {
					if hm, ok := hv.(map[string]any); ok {
						dr.Hooks = append(dr.Hooks, hm)
					}
				}
			}
			out = append(out, dr)
			return nil
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Band != out[j].Band {
			return out[i].Band < out[j].Band // "high" before "review_carefully"
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (b *bridge) uiRenderApprovals(w http.ResponseWriter, slug string) {
	c, ok := b.uiFindClient(slug)
	if !ok {
		http.Error(w, "unknown client", http.StatusNotFound)
		return
	}
	b.uiRender(w, "approvals", map[string]any{
		"Title": c.Slug + " approvals", "Client": c, "Drafts": b.uiPendingDrafts(c),
	})
}

type uiShortlistCandidate struct {
	N          any            `json:"n"`
	SourceName string         `json:"source_name"`
	SourceURL  string         `json:"source_url"`
	Platform   string         `json:"platform"`
	Cadence    string         `json:"cadence_suggested"`
	Why        string         `json:"why"`
	Class      string         `json:"classification"`
	Extra      map[string]any `json:"-"`
}

func (b *bridge) uiShortlist(c uiClient) (string, []uiShortlistCandidate) {
	p := filepath.Join(c.Path, "history", "discovery_shortlist.json")
	var doc struct {
		GeneratedAt string                 `json:"generated_at"`
		Candidates  []uiShortlistCandidate `json:"candidates"`
	}
	if uiReadJSON(p, &doc) != nil {
		return "", nil
	}
	return doc.GeneratedAt, doc.Candidates
}

func (b *bridge) uiRenderShortlist(w http.ResponseWriter, slug string) {
	c, ok := b.uiFindClient(slug)
	if !ok {
		http.Error(w, "unknown client", http.StatusNotFound)
		return
	}
	gen, cands := b.uiShortlist(c)
	b.uiRender(w, "shortlist", map[string]any{
		"Title": c.Slug + " shortlist", "Client": c, "GeneratedAt": gen, "Candidates": cands,
	})
}

// appendUIInbox appends one JSON line to a ui_inbox file. The bridge is the
// sole writer of these files (docs/UI_DESIGN.md §6.3), so O_APPEND + fsync of
// a single line is safe and keeps the file valid JSONL at all times.
func appendUIInbox(path string, obj map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	line, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

// handleUIAPI accepts POST /api/ui/{client}/approval and /api/ui/{client}/shortlist.
// Every write lands ONLY in ui_inbox/ (never a canonical ledger/CRM file); the
// Python/Go tools ingest them at the next run (crm_store ingest-ui).
func (b *bridge) handleUIAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/ui/"), "/"), "/")
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	c, ok := b.uiFindClient(parts[0])
	if !ok {
		http.Error(w, "unknown client", http.StatusNotFound)
		return
	}
	var body map[string]any
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxJSONBodyBytes))
	if err := dec.Decode(&body); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	session := b.uiToken[:8]
	writeJSON := func(v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
	switch parts[1] {
	case "approval":
		decision, _ := body["decision"].(string)
		draftID, _ := body["draft_id"].(string)
		if draftID == "" || !uiValidDecision(decision, "approve", "reject", "hold", "edit") {
			http.Error(w, "draft_id + decision(approve|reject|hold|edit) required", http.StatusBadRequest)
			return
		}
		rec := map[string]any{"ts": now, "draft_id": draftID, "decision": decision, "ui_session": session}
		for _, k := range []string{"campaign", "edited_subject", "edited_body", "note"} {
			if v, ok := body[k].(string); ok && v != "" {
				rec[k] = v
			}
		}
		p := filepath.Join(c.Path, "outreach", "ui_inbox", "approval_decisions.jsonl")
		if err := appendUIInbox(p, rec); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(map[string]any{"ok": true, "queued": draftID,
			"note": "recorded in ui_inbox; the next run (or 'ingest UI approvals' in chat) applies it"})
	case "shortlist":
		raw, ok := body["decisions"].([]any)
		if !ok || len(raw) == 0 {
			http.Error(w, "decisions[] required", http.StatusBadRequest)
			return
		}
		p := filepath.Join(c.Path, "ui_inbox", "shortlist_decisions.jsonl")
		n := 0
		for _, rv := range raw {
			rm, ok := rv.(map[string]any)
			if !ok {
				continue
			}
			decision, _ := rm["decision"].(string)
			srcURL, _ := rm["source_url"].(string)
			if srcURL == "" || !uiValidDecision(decision, "approve", "skip") {
				continue
			}
			rec := map[string]any{"ts": now, "source_url": srcURL, "decision": decision, "ui_session": session}
			for _, k := range []string{"source_name", "cadence"} {
				if v, ok := rm[k].(string); ok && v != "" {
					rec[k] = v
				}
			}
			if err := appendUIInbox(p, rec); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			n++
		}
		writeJSON(map[string]any{"ok": true, "queued": n,
			"note": "recorded in ui_inbox; tell your agent to apply the shortlist decisions"})
	case "sendbox-auth":
		// The ONE canonical write the UI performs outside ui_inbox (spec §6.2 v1.3):
		// the App Password must never transit chat or any agent-readable queue, so
		// the bridge itself verifies SMTP+IMAP live and persists credentials (0600).
		slug := strings.TrimSpace(mStr(body, "slug"))
		emailAddr := strings.TrimSpace(mStr(body, "email"))
		appPassword, _ := body["app_password"].(string)
		if slug == "" || emailAddr == "" || strings.TrimSpace(appPassword) == "" {
			http.Error(w, "slug + email + app_password required", http.StatusBadRequest)
			return
		}
		res, err := gmailAuthWithPassword(filepath.Join(c.Path, "outreach"), slug, emailAddr, appPassword)
		if err != nil {
			// sanitized: class-level reason only; never echo the password or raw
			// server chatter into the response
			writeJSON(map[string]any{"ok": false, "error": "auth_failed", "detail": errClassName(err)})
			return
		}
		writeJSON(res)
	default:
		http.NotFound(w, r)
	}
}

func uiValidDecision(v string, allowed ...string) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}

func (b *bridge) uiActiveRuns() []any {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.activeRunSummariesLocked()
}

func (b *bridge) uiRenderStatus(w http.ResponseWriter) {
	b.mu.Lock()
	exts := make([]map[string]string, 0, len(b.extensions))
	for _, t := range b.extensions {
		exts = append(exts, map[string]string{
			"Instance": t.instanceID, "Client": t.clientSlug, "Name": t.displayName,
			"Last": t.lastCheckAt.Format("2006-01-02 15:04:05"),
		})
	}
	b.mu.Unlock()
	b.uiRender(w, "status", map[string]any{
		"Title": "Status", "StartedAt": b.startedAt.Format(time.RFC3339),
		"DataRoot": b.uiDataRoot, "Persistent": b.cfg.persistent,
		"Extensions": exts, "Sendboxes": b.uiSendboxes(),
	})
}

func (b *bridge) uiRenderClient(w http.ResponseWriter, slug string) {
	c, ok := b.uiFindClient(slug)
	if !ok {
		http.Error(w, "unknown client", http.StatusNotFound)
		return
	}
	latest := b.uiListFiles(filepath.Join(c.Path, "outputs", "latest"), []string{".html", ".pdf"}, 20)
	latest = append(latest, b.uiListFiles(filepath.Join(c.Path, "outreach", "outputs", "latest"), []string{".html", ".pdf"}, 20)...)
	b.uiRender(w, "client", map[string]any{
		"Title": c.Slug, "Client": c, "Latest": latest,
		"Pending": len(b.uiPendingDrafts(c)),
	})
}

func (b *bridge) uiRenderReports(w http.ResponseWriter, slug string) {
	c, ok := b.uiFindClient(slug)
	if !ok {
		http.Error(w, "unknown client", http.StatusNotFound)
		return
	}
	files := b.uiListFiles(filepath.Join(c.Path, "outputs"), []string{".html", ".pdf"}, 120)
	files = append(files, b.uiListFiles(filepath.Join(c.Path, "outreach", "outputs"), []string{".html", ".pdf"}, 120)...)
	sort.Slice(files, func(i, j int) bool { return files[i].ModTime.After(files[j].ModTime) })
	b.uiRender(w, "reports", map[string]any{"Title": c.Slug + " reports", "Client": c, "Files": files})
}

func (b *bridge) uiRenderCRM(w http.ResponseWriter, slug string) {
	c, ok := b.uiFindClient(slug)
	if !ok {
		http.Error(w, "unknown client", http.StatusNotFound)
		return
	}
	deals := b.uiDeals(c)
	stages := map[string][]uiDeal{}
	var order []string
	var pipe struct {
		Pipelines []struct {
			Stages []struct {
				ID string `json:"id"`
			} `json:"stages"`
		} `json:"pipelines"`
	}
	if uiReadJSON(filepath.Join(c.Path, "outreach", "crm", "pipelines.json"), &pipe) == nil && len(pipe.Pipelines) > 0 {
		for _, s := range pipe.Pipelines[0].Stages {
			order = append(order, s.ID)
			stages[s.ID] = nil
		}
	}
	for _, d := range deals {
		stages[d.Stage] = append(stages[d.Stage], d)
	}
	b.uiRender(w, "crm", map[string]any{
		"Title": c.Slug + " CRM", "Client": c,
		"Contacts": b.uiContacts(c, 500), "StageOrder": order, "Stages": stages,
	})
}

func (b *bridge) uiRender(w http.ResponseWriter, page string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := uiTpl.ExecuteTemplate(w, page, data); err != nil {
		log.Printf("ui: template %s: %v", page, err)
	}
}

// ---------- templates (embedded, no build chain) ----------

var uiTpl = template.Must(template.New("ui").Parse(`
{{define "head"}}<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{.Title}} · Solo Agency</title>
<link rel="stylesheet" href="/ui/assets/pico.min.css">
<style>
/* thin overlay on Pico: map the app's structural classes onto Pico tokens */
:root{--pico-font-size:97%}
body>nav.sa{display:flex;gap:1.1rem;align-items:center;justify-content:flex-start;flex-wrap:wrap;padding:.6rem 1.1rem;border-bottom:1px solid var(--pico-muted-border-color)}
nav.sa a{text-decoration:none;font-weight:600}nav.sa .brand{color:var(--pico-primary)}
main.container{padding-top:1.1rem}
h1{font-size:1.45rem;margin-bottom:1rem}h2{font-size:1.05rem;margin:1.4rem 0 .55rem}
.mut{color:var(--pico-muted-color)}
.card{background:var(--pico-card-background-color);border:1px solid var(--pico-muted-border-color);border-radius:var(--pico-border-radius);box-shadow:var(--pico-card-box-shadow);padding:.85rem 1rem;margin:.5rem 0}
.grid-cards{display:grid;grid-template-columns:repeat(auto-fill,minmax(240px,1fr));gap:.7rem}
.pill{display:inline-block;border:1px solid var(--pico-muted-border-color);border-radius:999px;padding:.05rem .6rem;font-size:.75rem;color:var(--pico-muted-color);vertical-align:middle}
.wrap{overflow-x:auto}
table{font-size:.85rem}th,td{vertical-align:top}
textarea{min-height:150px;white-space:pre-wrap}select{width:auto}
button{width:auto}button.ok{--pico-background-color:var(--pico-primary);--pico-border-color:var(--pico-primary)}
.draft.done{opacity:.55}.acts{display:flex;gap:.5rem;margin-top:.6rem;flex-wrap:wrap;align-items:center}
.acts button{margin-bottom:0;padding:.35rem .9rem;font-size:.85rem}
.band-high{color:#16a34a;border-color:#16a34a}.band-review_carefully{color:#d97706;border-color:#d97706}
input,select,textarea{margin-bottom:.6rem}
</style></head><body>
<nav class="sa"><a class="brand" href="/ui"><strong>Solo Agency</strong></a><a href="/ui/jobs">Jobs</a><a href="/ui/status">Status</a></nav>
<main class="container"><h1>{{.Title}}</h1>{{end}}

{{define "foot"}}</main><script>
try{var es=new EventSource('/events');es.addEventListener('change',function(){location.reload()})}catch(e){}
</script></body></html>{{end}}

{{define "locked"}}{{template "head" .}}
<div class="card"><p><strong>UI locked.</strong> Open the tokenized entry link once to unlock this browser.</p>
<p class="mut">Ask your AI agent for the entry link, or read <code>daily-content-pipeline/bridge/ui_token</code> and open <code>/ui/enter/&lt;token&gt;</code>.</p></div>
{{template "foot" .}}{{end}}

{{define "home"}}{{template "head" .}}
<h2>Clients</h2><div class="grid-cards">
{{range .Clients}}<div class="card"><strong><a href="/ui/{{.Slug}}">{{.Slug}}</a></strong><br>
<span class="mut">{{.Workspace}}</span><br>
<a href="/ui/{{.Slug}}/reports">reports</a> · <a href="/ui/{{.Slug}}/crm">crm</a> · <a href="/ui/{{.Slug}}/approvals">approvals</a></div>
{{else}}<p class="mut">No clients yet.</p>{{end}}</div>
<h2>Recent jobs</h2><div class="wrap"><table><tr><th>state</th><th>client</th><th>kind</th><th>file</th><th>when</th></tr>
{{range .Jobs}}<tr><td><span class="pill">{{.State}}</span></td><td>{{.Client}}</td><td>{{.Kind}}</td><td class="mut">{{.Name}}</td><td class="mut">{{.ModTime.Format "01-02 15:04"}}</td></tr>{{else}}<tr><td colspan="5" class="mut">none</td></tr>{{end}}</table></div>
{{template "foot" .}}{{end}}

{{define "jobs"}}{{template "head" .}}
<h2>Active runs</h2><div class="card"><pre class="mut" style="margin:0;white-space:pre-wrap">{{range .Active}}{{printf "%v" .}}
{{else}}none{{end}}</pre></div>
<h2>Queue</h2><div class="wrap"><table><tr><th>state</th><th>client</th><th>kind</th><th>run id</th><th>file</th><th>when</th></tr>
{{range .Jobs}}<tr><td><span class="pill">{{.State}}</span></td><td>{{.Client}}</td><td>{{.Kind}}</td><td class="mut">{{.RunID}}</td><td class="mut">{{.Name}}</td><td class="mut">{{.ModTime.Format "01-02 15:04"}}</td></tr>{{else}}<tr><td colspan="6" class="mut">empty</td></tr>{{end}}</table></div>
{{template "foot" .}}{{end}}

{{define "status"}}{{template "head" .}}
<div class="card">Bridge started <strong>{{.StartedAt}}</strong> · persistent: {{.Persistent}}<br>
<span class="mut">data root: {{.DataRoot}}</span></div>
<h2>Extensions</h2><div class="wrap"><table><tr><th>client</th><th>instance</th><th>name</th><th>last check-in</th></tr>
{{range .Extensions}}<tr><td>{{.Client}}</td><td class="mut">{{.Instance}}</td><td>{{.Name}}</td><td class="mut">{{.Last}}</td></tr>{{else}}<tr><td colspan="4" class="mut">no extension check-ins yet</td></tr>{{end}}</table></div>
<h2>Sendboxes</h2><div class="wrap"><table><tr><th>client</th><th>slug</th><th>email</th><th>status</th><th>quota</th><th>warmup</th></tr>
{{range .Sendboxes}}<tr><td>{{.Client}}</td><td>{{.Slug}}</td><td>{{.Email}}</td><td><span class="pill">{{.Status}}</span></td><td>{{.Quota}}</td><td class="mut">{{.Warmup}}</td></tr>{{else}}<tr><td colspan="6" class="mut">none configured</td></tr>{{end}}</table></div>
{{template "foot" .}}{{end}}

{{define "client"}}{{template "head" .}}
<p><a href="/ui/{{.Client.Slug}}/reports">All reports</a> · <a href="/ui/{{.Client.Slug}}/crm">CRM</a> ·
<a href="/ui/{{.Client.Slug}}/approvals">Approvals{{if .Pending}} <strong>({{.Pending}})</strong>{{end}}</a> ·
<a href="/ui/{{.Client.Slug}}/shortlist">Shortlist</a> ·
<a href="/ui/{{.Client.Slug}}/sendboxes">Sendboxes</a></p>
<h2>Latest</h2><div class="wrap"><table><tr><th>file</th><th>when</th></tr>
{{range .Latest}}<tr><td><a href="/files/{{.Rel}}">{{.Name}}</a></td><td class="mut">{{.ModTime.Format "2006-01-02 15:04"}}</td></tr>{{else}}<tr><td colspan="2" class="mut">no outputs yet — run the client's daily task</td></tr>{{end}}</table></div>
{{template "foot" .}}{{end}}

{{define "reports"}}{{template "head" .}}
<div class="wrap"><table><tr><th>file</th><th>when</th><th>size</th></tr>
{{range .Files}}<tr><td><a href="/files/{{.Rel}}">{{.Rel}}</a></td><td class="mut">{{.ModTime.Format "2006-01-02 15:04"}}</td><td class="mut">{{.Size}}</td></tr>{{else}}<tr><td colspan="3" class="mut">no reports yet</td></tr>{{end}}</table></div>
{{template "foot" .}}{{end}}

{{define "crm"}}{{template "head" .}}
<h2>Pipeline</h2><div class="grid-cards">
{{$st := .Stages}}{{range .StageOrder}}<div class="card"><strong>{{.}}</strong>
{{range index $st .}}<div class="mut">{{if .Title}}{{.Title}}{{else}}{{.ID}}{{end}}</div>{{else}}<div class="mut">—</div>{{end}}</div>{{end}}</div>
<h2>Contacts</h2><div class="wrap"><table><tr><th>name</th><th>email</th><th>vertical</th><th>stage</th></tr>
{{range .Contacts}}<tr><td>{{if .Name}}{{.Name}}{{else}}<span class="mut">{{.ID}}</span>{{end}}</td><td class="mut">{{.Email}}</td><td>{{.Vertical}}</td><td class="mut">{{.Stage}}</td></tr>{{else}}<tr><td colspan="4" class="mut">no contacts yet</td></tr>{{end}}</table></div>
{{template "foot" .}}{{end}}

{{define "footform"}}
<div class="card mut" style="margin-top:16px">Decisions are queued in <code>ui_inbox/</code> — the agent applies them automatically at the start of the next campaign run, or tell it: <em>"apply my UI decisions"</em>.</div>
</main></body></html>{{end}}

{{define "approvals"}}{{template "head" .}}
<p><a href="/ui/{{.Client.Slug}}">← {{.Client.Slug}}</a> · <span id="left">{{len .Drafts}}</span> pending
<button id="allhigh" style="margin-left:10px">Approve all high-confidence</button></p>
{{range .Drafts}}
<div class="card draft" data-id="{{.ID}}" data-campaign="{{.Campaign}}" data-band="{{.Band}}">
<div><strong>{{.To}}</strong> <span class="pill band-{{.Band}}">{{.Band}}</span>
<span class="pill">{{.Campaign}}</span> <span class="pill">step {{.Step}}</span>
{{if .Companion}}<a class="pill" href="{{.Companion}}" target="_blank" rel="noopener">companion ↗</a>{{end}}</div>
{{if .Warnings}}<div style="margin-top:6px">{{range .Warnings}}<span class="pill band-review_carefully">⚠ {{.}}</span> {{end}}</div>{{end}}
{{if .Hooks}}<div class="mut" style="margin-top:6px;font-size:13px">hooks: {{range .Hooks}}{{if index . "evidence_url"}}<a href="{{index . "evidence_url"}}" target="_blank" rel="noopener">{{index . "type"}}</a> {{else}}{{index . "type"}} {{end}}{{end}}</div>{{end}}
<div style="margin-top:8px"><input class="subj" type="text" value="{{.Subject}}"></div>
<div style="margin-top:8px"><textarea class="body">{{.Body}}</textarea></div>
<div class="acts">
<button class="ok" data-act="approve">Approve</button>
<button data-act="edit">Save edit (keep pending)</button>
<button data-act="hold">Hold</button>
<button data-act="reject">Reject…</button>
</div></div>
{{else}}<p class="mut">No drafts waiting for approval. New drafts appear here after the campaign's daily run.</p>{{end}}
<script>
var CLIENT="{{.Client.Slug}}";
function payload(card){var p={draft_id:card.dataset.id,campaign:card.dataset.campaign};
 var s=card.querySelector('.subj'),b=card.querySelector('.body');
 if(s.value!==s.defaultValue)p.edited_subject=s.value;
 if(b.value!==b.defaultValue)p.edited_body=b.value;return p}
function send(card,act,note){var p=payload(card);p.decision=act;if(note)p.note=note;
 return fetch('/api/ui/'+CLIENT+'/approval',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(p)})
 .then(function(r){if(!r.ok)throw new Error('HTTP '+r.status);return r.json()})
 .then(function(){if(act!=='edit'){card.classList.add('done');card.querySelector('.acts').innerHTML='<span class="pill">'+act+' ✓ queued</span>'}
  else{card.querySelector('.subj').defaultValue=card.querySelector('.subj').value;card.querySelector('.body').defaultValue=card.querySelector('.body').value}
  var n=document.querySelectorAll('.draft:not(.done)').length;document.getElementById('left').textContent=n})
 .catch(function(e){alert('Failed: '+e.message)})}
document.addEventListener('click',function(e){var b=e.target.closest('button[data-act]');if(!b)return;
 var card=b.closest('.draft');var act=b.getAttribute('data-act');
 if(act==='reject'){var note=prompt('Reject reason (feeds the learning log):','');if(note===null)return;send(card,act,note)}
 else send(card,act)});
document.getElementById('allhigh').addEventListener('click',function(){
 var cards=document.querySelectorAll('.draft[data-band="high"]:not(.done)');
 if(!cards.length){alert('No untouched high-confidence drafts.');return}
 if(!confirm('Approve '+cards.length+' high-confidence draft(s)?'))return;
 var q=Promise.resolve();cards.forEach(function(c){q=q.then(function(){return send(c,'approve')})})});
</script>
{{template "footform" .}}{{end}}

{{define "shortlist"}}{{template "head" .}}
<p><a href="/ui/{{.Client.Slug}}">← {{.Client.Slug}}</a>{{if .GeneratedAt}} · <span class="mut">generated {{.GeneratedAt}}</span>{{end}}</p>
{{if .Candidates}}
<div class="wrap"><table><tr><th>keep</th><th>#</th><th>source</th><th>platform</th><th>why</th><th>cadence</th></tr>
{{range .Candidates}}<tr data-url="{{.SourceURL}}" data-name="{{.SourceName}}">
<td><input class="pick" type="checkbox" checked></td><td class="mut">{{.N}}</td>
<td><strong>{{.SourceName}}</strong>{{if .Class}} <span class="pill">{{.Class}}</span>{{end}}<br><a href="{{.SourceURL}}" target="_blank" rel="noopener" class="mut" style="font-size:12px">{{.SourceURL}}</a></td>
<td>{{.Platform}}</td><td class="mut" style="font-size:13px">{{.Why}}</td>
<td><select class="cad"><option{{if eq .Cadence "daily"}} selected{{end}}>daily</option><option{{if eq .Cadence "weekly"}} selected{{end}}>weekly</option><option{{if eq .Cadence "optional"}} selected{{end}}>optional</option></select></td>
</tr>{{end}}</table></div>
<div class="acts"><button class="ok" id="submit">Submit decisions</button><span class="mut" id="msg"></span></div>
<script>
var CLIENT="{{.Client.Slug}}";
document.getElementById('submit').addEventListener('click',function(){
 var ds=[];document.querySelectorAll('tr[data-url]').forEach(function(r){
  ds.push({source_url:r.dataset.url,source_name:r.dataset.name,
   decision:r.querySelector('.pick').checked?'approve':'skip',
   cadence:r.querySelector('.cad').value})});
 var btn=this;btn.disabled=true;
 fetch('/api/ui/'+CLIENT+'/shortlist',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({decisions:ds})})
 .then(function(r){if(!r.ok)throw new Error('HTTP '+r.status);return r.json()})
 .then(function(j){document.getElementById('msg').textContent='✓ '+j.queued+' decision(s) queued'})
 .catch(function(e){btn.disabled=false;alert('Failed: '+e.message)})});
</script>
{{else}}<p class="mut">No shortlist published. The agent writes <code>history/discovery_shortlist.json</code> when a private-source discovery finishes.</p>{{end}}
{{template "footform" .}}{{end}}

{{define "sendboxes"}}{{template "head" .}}
<p><a href="/ui/{{.Client.Slug}}">← {{.Client.Slug}}</a></p>
{{if .Sendboxes}}
<div class="wrap"><table><tr><th>slug</th><th>email</th><th>status</th><th>quota/day</th><th>warmup</th><th>last sync</th><th></th></tr>
{{range .Sendboxes}}<tr>
<td><code>{{.slug}}</code></td><td>{{.email}}</td>
<td><span class="pill{{if eq .status "healthy"}} band-high{{else}} band-review_carefully{{end}}">{{.status}}</span></td>
<td>{{.quota_today}}</td><td class="mut">{{.warmup_stage}}</td>
<td class="mut">{{.last_successful_sync_ts}}</td>
<td><a href="#connect" class="pick-box" data-slug="{{.slug}}" data-email="{{.email}}">connect / re-auth</a></td>
</tr>{{end}}</table></div>
{{else}}<p class="mut">No sendboxes yet — connect the first one below.</p>{{end}}

<h2 id="connect">Connect a sendbox (Gmail App Password)</h2>
<div class="card" style="max-width:560px">
<form id="authform">
<label>Sendbox slug
<input id="f-slug" type="text" placeholder="sb-a" required></label>
<label>Gmail address
<input id="f-email" type="email" placeholder="you@gmail.com" required></label>
<label>App Password <span class="mut">(16 characters — Google Account → Security → App passwords)</span>
<input id="f-pass" type="password" autocomplete="off" placeholder="xxxx xxxx xxxx xxxx" required></label>
<button class="ok" type="submit">Connect &amp; verify</button>
<span id="authmsg" class="mut"></span>
</form>
<p class="mut" style="font-size:.8rem;margin-bottom:0">The password goes from this page straight to Gmail over TLS and is stored only on this machine
(<code>sendboxes/&lt;slug&gt;/credentials.json</code>, permissions 0600). Never paste an App Password
into the agent chat — this page is the one intended place for it.</p>
</div>
<script>
var CLIENT="{{.Client.Slug}}";
document.querySelectorAll('.pick-box').forEach(function(a){a.addEventListener('click',function(){
 document.getElementById('f-slug').value=this.dataset.slug;
 document.getElementById('f-email').value=this.dataset.email;
 document.getElementById('f-pass').focus()})});
document.getElementById('authform').addEventListener('submit',function(e){
 e.preventDefault();
 var btn=this.querySelector('button');btn.disabled=true;btn.setAttribute('aria-busy','true');
 var msg=document.getElementById('authmsg');msg.textContent='Verifying SMTP + IMAP with Gmail…';
 fetch('/api/ui/'+CLIENT+'/sendbox-auth',{method:'POST',headers:{'Content-Type':'application/json'},
  body:JSON.stringify({slug:document.getElementById('f-slug').value.trim(),
   email:document.getElementById('f-email').value.trim(),
   app_password:document.getElementById('f-pass').value})})
 .then(function(r){return r.json()})
 .then(function(j){
  if(j.ok){msg.textContent='✓ connected ('+j.email+', quota '+j.quota_today+'/day)';
   document.getElementById('f-pass').value='';setTimeout(function(){location.reload()},900)}
  else{btn.disabled=false;btn.removeAttribute('aria-busy');
   msg.textContent='✗ '+(j.error||'failed')+(j.detail?' — '+j.detail:'')+' (check the address and the App Password)'}})
 .catch(function(err){btn.disabled=false;btn.removeAttribute('aria-busy');msg.textContent='✗ '+err.message})});
</script>
</main></body></html>{{end}}
`))
