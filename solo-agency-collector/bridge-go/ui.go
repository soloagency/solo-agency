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
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
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
	stamp(filepath.Join(root, "collector", "logs", "extension_health.jsonl"))
	for _, ws := range b.uiClients() {
		stamp(filepath.Join(ws.Path, "outputs"))
		stamp(filepath.Join(ws.Path, "outreach", "outputs"))
		stamp(filepath.Join(ws.Path, "outreach", "ui_inbox"))
		stamp(filepath.Join(ws.Path, "ui_inbox"))
		stamp(filepath.Join(ws.Path, "history", "discovery_shortlist.json"))
		stamp(filepath.Join(ws.Path, "outreach", "crm", "contacts"))
		stamp(filepath.Join(ws.Path, "outreach", "crm", "deals"))
		// pending_approval holds YYYY-MM-DD subdirs; stamp those so new drafts refresh the page
		campaigns := filepath.Join(ws.Path, "outreach", "campaigns")
		if camps, err := os.ReadDir(campaigns); err == nil {
			for _, camp := range camps {
				if !camp.IsDir() {
					continue
				}
				stamp(filepath.Join(campaigns, camp.Name(), "campaign_config.json"))
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
	ID              string
	ShortID         string
	Name            string
	Email           string
	Phone           string
	Social          string // one representative social/profile URL
	Vertical        string
	Stage           string
	Band            string // enrichment.confidence_band ("" = not enriched)
	Seeds           int
	SeedsUnresolved int
}

// shortID trims a ULID to a short, still-unique display code: the type prefix
// plus the last 6 chars (the ULID's random tail — where collisions can't hide),
// e.g. "c_01KXY7Q17X7MYGMTRSPPFNNR92" -> "c_…FNNR92".
func shortID(id string) string {
	prefix := ""
	body := id
	if i := strings.Index(id, "_"); i >= 0 && i < 5 {
		prefix = id[:i+1]
		body = id[i+1:]
	}
	if len(body) <= 8 {
		return id
	}
	return prefix + "…" + body[len(body)-6:]
}

// contactName pulls the best display name from a contact doc.
func contactName(doc map[string]any) string {
	for _, k := range []string{"display_name", "full_name"} {
		if v, ok := doc[k].(string); ok && v != "" {
			return v
		}
	}
	if n, ok := doc["name"].(map[string]any); ok {
		if v, ok := n["full"].(string); ok && v != "" {
			return v
		}
	}
	if v, ok := doc["name"].(string); ok && v != "" {
		return v
	}
	return ""
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
		doc, err := readJSONFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		// hide merge tombstones — they resolve to the winner
		if mStr(mMap(doc, "merge"), "status") == "merged" {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		ct := uiContact{ID: id, ShortID: shortID(id), Name: contactName(doc)}
		ids := mMap(doc, "identities")
		if emails := mapsOf(mList(ids, "emails")); len(emails) > 0 {
			ct.Email = mStr(emails[0], "address")
		}
		if phones := mapsOf(mList(ids, "phones")); len(phones) > 0 {
			ct.Phone = mStr(phones[0], "number")
		}
		for _, k := range sortedKeys(mMap(ids, "socials")) {
			if v, ok := mMap(ids, "socials")[k].(string); ok && v != "" {
				ct.Social = v
				break
			}
		}
		if ct.Social == "" {
			if w := mStr(ids, "website"); w != "" {
				ct.Social = w
			}
		}
		ct.Vertical = mStr(mMap(doc, "custom_fields"), "professional_vertical")
		ct.Stage = mStr(doc, "lifecycle_stage")
		ct.Band = mStr(mMap(doc, "enrichment"), "confidence_band")
		for _, sd := range mapsOf(mList(ids, "seeds")) {
			ct.Seeds++
			if mStr(sd, "status") != "resolved" {
				ct.SeedsUnresolved++
			}
		}
		out = append(out, ct)
	}
	// enriched + named first, then by name, then by short id
	sort.Slice(out, func(i, j int) bool {
		if (out[i].Name != "") != (out[j].Name != "") {
			return out[i].Name != ""
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// uiContactDetail returns the full record + a personalization view (hooks =
// the "latest activities" used to personalize email) + the activity timeline.
func (b *bridge) uiContactDetail(c uiClient, id string) map[string]any {
	if safeID(id) != nil {
		return nil
	}
	doc, err := readJSONFile(filepath.Join(c.Path, "outreach", "crm", "contacts", id+".json"))
	if err != nil {
		return nil
	}
	ids := mMap(doc, "identities")
	var emails, phones []map[string]any
	emails = mapsOf(mList(ids, "emails"))
	phones = mapsOf(mList(ids, "phones"))
	var socials [][2]string
	for _, k := range sortedKeys(mMap(ids, "socials")) {
		if v, ok := mMap(ids, "socials")[k].(string); ok && v != "" {
			socials = append(socials, [2]string{k, v})
		}
	}
	en := mMap(doc, "enrichment")
	var hooks []map[string]any
	for _, h := range mapsOf(mList(en, "hooks")) {
		hooks = append(hooks, map[string]any{
			"Type": mStr(h, "type"), "Summary": mStr(h, "summary"),
			"URL": mStr(h, "evidence_url"), "Observed": mStr(h, "observed_date"),
			"UsedIn": mList(h, "used_in"),
		})
	}
	brief := mMap(en, "writing_brief")
	ident := mMap(en, "identity")
	return map[string]any{
		"ID": id, "Name": contactName(doc), "Stage": mStr(doc, "lifecycle_stage"),
		"Emails": emails, "Phones": phones, "Socials": socials,
		"Website": mStr(ids, "website"), "Seeds": mapsOf(mList(ids, "seeds")),
		"Band": mStr(en, "confidence_band"), "Enriched": len(en) > 0,
		"StillActive": mStr(ident, "still_active"), "Company": mStr(ident, "current_company"),
		"Role": mStr(ident, "role"), "OneLiner": mStr(brief, "one_liner"),
		"Angles": mList(brief, "ranked_angles"), "DoNotMention": mList(brief, "do_not_mention"),
		"Hooks": hooks, "HooksRefreshed": mStr(en, "hooks_refreshed_at"),
		"Vertical":          mStr(mMap(doc, "custom_fields"), "professional_vertical"),
		"SequenceState":     mStr(doc, "sequence_state"),
		"DuplicateSuspects": mapsOf(mList(doc, "duplicate_suspects")),
		"Activities":        b.uiContactActivities(c, id, 40),
	}
}

// uiContactActivities scans the monthly activity logs for one contact, newest
// first. Rows logged against ids that were later MERGED into this contact are
// included too (memoized resolve, so consolidated fragments keep their history).
func (b *bridge) uiContactActivities(c uiClient, id string, cap int) []map[string]any {
	base := filepath.Join(c.Path, "outreach", "crm", "activities")
	months, _ := os.ReadDir(base)
	names := make([]string, 0, len(months))
	for _, m := range months {
		names = append(names, m.Name())
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	store := newCrmStore(filepath.Join(c.Path, "outreach"))
	memo := map[string]string{}
	resolved := func(x string) string {
		if x == "" {
			return x
		}
		if v, ok := memo[x]; ok {
			return v
		}
		v := store.resolve(x)
		memo[x] = v
		return v
	}
	var out []map[string]any
	for _, m := range names {
		rows := readJSONLines(filepath.Join(base, m, "activities.jsonl"))
		for i := len(rows) - 1; i >= 0; i-- {
			r := rows[i]
			if cid := mStr(r, "contact_id"); cid != id && resolved(cid) != id {
				continue
			}
			out = append(out, map[string]any{
				"Type": mStr(r, "type"), "Summary": mStr(r, "summary"),
				"By": mStr(r, "by"), "At": mStr(r, "ts"),
			})
			if len(out) >= cap {
				return out
			}
		}
	}
	return out
}

func (b *bridge) uiRenderContact(w http.ResponseWriter, slug, id string) {
	c, ok := b.uiFindClient(slug)
	if !ok {
		http.Error(w, "unknown client", http.StatusNotFound)
		return
	}
	// a consolidated fragment's old link redirects to its survivor
	if safeID(id) == nil {
		if rid := newCrmStore(filepath.Join(c.Path, "outreach")).resolve(id); rid != id {
			w.Header().Set("Location", "/ui/"+slug+"/contact/"+rid)
			w.WriteHeader(http.StatusFound)
			return
		}
	}
	d := b.uiContactDetail(c, id)
	if d == nil {
		http.Error(w, "unknown contact", http.StatusNotFound)
		return
	}
	name := mStr(d, "Name")
	if name == "" {
		name = shortID(id)
	}
	b.uiRender(w, "contact", map[string]any{"Title": name, "Client": c, "C": d})
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
	case len(parts) == 3 && parts[1] == "contact":
		b.uiRenderContact(w, parts[0], parts[2])
	case len(parts) == 2 && parts[1] == "campaigns":
		b.uiRenderCampaigns(w, parts[0])
	case len(parts) == 3 && parts[1] == "campaign":
		b.uiRenderCampaign(w, parts[0], parts[2])
	case len(parts) == 2 && parts[1] == "approvals":
		b.uiRenderApprovals(w, parts[0])
	case len(parts) == 2 && parts[1] == "shortlist":
		b.uiRenderShortlist(w, parts[0])
	case len(parts) == 2 && parts[1] == "sendboxes":
		b.uiRenderSendboxes(w, parts[0])
	case len(parts) == 2 && parts[1] == "extension":
		b.uiRenderExtension(w, parts[0])
	default:
		http.NotFound(w, r)
	}
}

func handleUIPicoCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write([]byte(picoCSS))
}

// ---------- feature catalog (mirrors playbooks/FEATURE_CATALOG.md headline rows) ----------

// uiFeature is one action card. Kind "ui" runs right here (Href is the
// per-client subpage); kind "agent" is started by pasting Phrase into the
// right chat session (Session says which one). Keep phrases IDENTICAL to the
// trigger phrases in playbooks/FEATURE_CATALOG.md — that file is the honesty
// guardrail; never list a capability it does not have.
type uiFeature struct {
	Group   string
	Title   string
	Value   string
	Kind    string
	Href    string
	Phrase  string
	Session string
}

var uiFeatures = []uiFeature{
	{"Content", "Run today's report", "Fresh ideas, leads and drafts for this client — right now, no need to wait for the schedule", "agent", "", "run today's content for {client}", "a NEW chat session (automation)"},
	{"Content", "Make a video", "Turn an approved idea into a real short video (WideCast)", "agent", "", "make a video from today's best idea", "a NEW chat session (automation)"},
	{"Content", "Blog + social posts", "Turn one idea into a blog and platform-ready posts", "agent", "", "write the blog and social posts", "a NEW chat session (automation)"},
	{"Content", "Private-source discovery", "Find the groups/communities your audience gathers in, from places you already joined", "agent", "", "run discovery", "the shared SETUP session"},
	{"Content", "Latest reports", "Daily HTML reports: ideas, drafts, leads, opportunities", "ui", "reports", "", ""},
	{"Content", "Install / check the Chrome extension", "Drag-and-drop install for this client's collector extension, with a live connected check", "ui", "extension", "", ""},
	{"Outreach", "Create a cold-email campaign", "Personalized, evidence-backed cold email — 3 questions and it runs; nothing sends without your approval", "agent", "", "set up a cold-email campaign", "the shared SETUP session"},
	{"Outreach", "Manage campaigns", "Edit each campaign's goal and companion link, change the daily budget, pause/resume — right here", "ui", "campaigns", "", ""},
	{"Outreach", "Import a lead list", "Bring in a CSV of prospects, deduped and suppression-checked", "agent", "", "import a list: <path to your CSV>", "the shared SETUP session"},
	{"Outreach", "Review & approve drafts", "Approve, edit, hold or reject every drafted email — right here", "ui", "approvals", "", ""},
	{"Outreach", "Approve discovered sources", "Tick the monitoring shortlist the agent proposed", "ui", "shortlist", "", ""},
	{"Outreach", "Connect a sendbox", "Paste the Gmail App Password here — never into chat", "ui", "sendboxes", "", ""},
	{"Outreach", "CRM pipeline", "Replies become deals moving through stages", "ui", "crm", "", ""},
}

func uiFeaturesFor(slug string) []map[string]any {
	out := make([]map[string]any, 0, len(uiFeatures))
	for _, f := range uiFeatures {
		out = append(out, map[string]any{
			"Group": f.Group, "Title": f.Title, "Value": f.Value, "Kind": f.Kind,
			"Href": f.Href, "Session": f.Session,
			"Phrase": strings.ReplaceAll(f.Phrase, "{client}", slug),
		})
	}
	return out
}

// resolveSendboxSlug maps an email to its existing box (re-auth) or mints the
// next free conventional slug so the UI never has to ask a human for one.
func resolveSendboxSlug(clientDir, emailAddr string) string {
	boxes := mapsOf(mList(loadSendboxesDoc(clientDir), "sendboxes"))
	taken := map[string]bool{}
	for _, b := range boxes {
		if normalizeEmail(mStr(b, "email")) == normalizeEmail(emailAddr) {
			return mStr(b, "slug")
		}
		taken[mStr(b, "slug")] = true
	}
	for ch := 'a'; ch <= 'z'; ch++ {
		cand := "sb-" + string(ch)
		if !taken[cand] {
			return cand
		}
	}
	return "sb-" + gmailMkToken()[:4]
}

// ---------- campaigns ----------

// uiCampaignRow summarizes one campaign for the list page.
func (b *bridge) uiCampaignRows(c uiClient) []map[string]any {
	store := newCrmStore(filepath.Join(c.Path, "outreach"))
	var out []map[string]any
	pendingByCamp := map[string]int{}
	for _, d := range b.uiPendingDrafts(c) {
		pendingByCamp[d.Campaign]++
	}
	for _, cfg := range store.listCampaigns() {
		slug := mStr(cfg, "campaign_slug")
		row := map[string]any{
			"Slug": slug, "Status": strOr(mStr(cfg, "status"), "active"),
			"GoalType":  mStr(mMap(cfg, "goal"), "goal_type"),
			"Objective": mStr(mMap(cfg, "goal"), "objective"),
			"Quota":     mInt(cfg, "daily_quota", 40),
			"Pending":   pendingByCamp[slug],
			"Sent":      0, "LastSent": "",
		}
		if budget, err := store.draftBudget(slug, ""); err == nil {
			row["UsedToday"] = budget["used_today"]
		}
		sent, last := 0, ""
		for _, p := range store.allSentLogs(slug) {
			for _, r := range readJSONLines(p) {
				if mStr(r, "rfc_message_id") != "" {
					sent++
					if sa := mStr(r, "sent_at"); sa > last {
						last = sa
					}
				}
			}
		}
		row["Sent"] = sent
		if len(last) >= 10 {
			row["LastSent"] = last[:10]
		}
		out = append(out, row)
	}
	return out
}

func (b *bridge) uiRenderCampaigns(w http.ResponseWriter, slug string) {
	c, ok := b.uiFindClient(slug)
	if !ok {
		http.Error(w, "unknown client", http.StatusNotFound)
		return
	}
	b.uiRender(w, "campaigns", map[string]any{
		"Title": c.Slug + " campaigns", "Client": c, "Rows": b.uiCampaignRows(c),
	})
}

func (b *bridge) uiRenderCampaign(w http.ResponseWriter, slug, camp string) {
	c, ok := b.uiFindClient(slug)
	if !ok {
		http.Error(w, "unknown client", http.StatusNotFound)
		return
	}
	store := newCrmStore(filepath.Join(c.Path, "outreach"))
	cfg := store.getCampaign(camp)
	if cfg == nil {
		http.Error(w, "unknown campaign", http.StatusNotFound)
		return
	}
	goal := mMap(cfg, "goal")
	cd := mMap(goal, "companion_doc")
	var proofLines []string
	for _, p := range mList(goal, "proof_points") {
		proofLines = append(proofLines, fmt.Sprint(p))
	}
	pending := 0
	for _, d := range b.uiPendingDrafts(c) {
		if d.Campaign == camp {
			pending++
		}
	}
	data := map[string]any{
		"Title": camp, "Client": c, "Slug": camp,
		"Status":    strOr(mStr(cfg, "status"), "active"),
		"Quota":     mInt(cfg, "daily_quota", 40),
		"Segment":   mStr(mMap(cfg, "audience"), "segment"),
		"Sendboxes": mList(cfg, "sendboxes"),
		"GoalType":  mStr(goal, "goal_type"), "GoalTypes": sortedGoalTypes(),
		"Objective": mStr(goal, "objective"), "Offer": mStr(goal, "offer"),
		"ValueProp":             mStr(goal, "value_proposition"),
		"Proof":                 strings.Join(proofLines, "\n"),
		"CTAText":               mStr(mMap(goal, "cta"), "text"),
		"CompanionInstructions": mStr(cd, "instructions"),
		"CompanionOnFail":       strOr(mStr(cd, "on_fail"), "skip"),
		"CompanionDefault":      mStr(cd, "default_link"),
		"Pending":               pending,
	}
	if budget, err := store.draftBudget(camp, ""); err == nil {
		data["UsedToday"] = budget["used_today"]
	}
	b.uiRender(w, "campaign", data)
}

// ---------- extension install helper ----------

// uiOpenInFileManager reveals a folder in the OS file manager so the human can
// DRAG it onto chrome://extensions instead of memorizing a path. Injectable
// for tests.
var uiOpenInFileManager = func(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("explorer", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start() // fire and forget; explorer's exit codes are meaningless
}

// uiExtensionInfo resolves the client's extension folder (registry entry wins,
// conventional {setup-root}/extensions/{slug} otherwise) and its live check-in
// state from the bridge's in-memory tracker.
func (b *bridge) uiExtensionInfo(c uiClient) map[string]any {
	setupRoot := filepath.Dir(b.uiDataRoot)
	folder := filepath.Join(setupRoot, "extensions", c.Slug)
	info := map[string]any{"Folder": folder, "Exists": false, "Instance": "",
		"LastCheck": "", "CheckedIn": false}
	if reg, err := readJSONFile(filepath.Join(b.uiDataRoot, "collector", "extension_registry.json")); err == nil {
		for _, e := range mapsOf(mList(reg, "clients")) {
			if mStr(e, "client_slug") == c.Slug {
				if f := mStr(e, "extension_folder"); f != "" {
					info["Folder"] = f
					folder = f
				}
				info["Instance"] = mStr(e, "extension_instance_id")
			}
		}
	}
	if st, err := os.Stat(folder); err == nil && st.IsDir() {
		info["Exists"] = true
	}
	b.mu.Lock()
	for _, t := range b.extensions {
		if t.clientSlug == c.Slug {
			info["CheckedIn"] = true
			info["LastCheck"] = t.lastCheckAt.Format("2006-01-02 15:04:05")
			if mStr(info, "Instance") == "" {
				info["Instance"] = t.instanceID
			}
		}
	}
	b.mu.Unlock()
	return info
}

func (b *bridge) uiRenderExtension(w http.ResponseWriter, slug string) {
	c, ok := b.uiFindClient(slug)
	if !ok {
		http.Error(w, "unknown client", http.StatusNotFound)
		return
	}
	b.uiRender(w, "extension", map[string]any{
		"Title": c.Slug + " extension", "Client": c, "Ext": b.uiExtensionInfo(c),
	})
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
	case "campaign-update":
		// Operator-owned campaign config: applied through the SAME whitelist
		// as `tool crm-store campaign update` (instant effect — the daily run
		// reads the file fresh), plus an informational ui_inbox event so the
		// agent knows the operator changed it.
		campSlug := strings.TrimSpace(mStr(body, "slug"))
		patch, _ := body["patch"].(map[string]any)
		if campSlug == "" || len(patch) == 0 {
			http.Error(w, "slug + patch required", http.StatusBadRequest)
			return
		}
		store := newCrmStore(filepath.Join(c.Path, "outreach"))
		res, err := store.campaignUpdate(campSlug, patch)
		if err != nil {
			writeJSON(map[string]any{"ok": false, "error": err.Error()})
			return
		}
		if changed := mList(res, "changed"); len(changed) > 0 {
			_ = appendUIInbox(filepath.Join(c.Path, "outreach", "ui_inbox", "campaign_edits.jsonl"),
				map[string]any{"ts": now, "campaign": campSlug, "changed": changed, "ui_session": session})
		}
		writeJSON(map[string]any{"ok": true, "campaign": campSlug, "changed": res["changed"],
			"note": "saved — takes effect from the next run; the agent is notified via ui_inbox"})
	case "reveal-extension":
		info := b.uiExtensionInfo(c)
		folder := mStr(info, "Folder")
		setupRoot := filepath.Dir(b.uiDataRoot)
		cleanFolder := filepath.Clean(folder)
		if !strings.HasPrefix(cleanFolder, filepath.Clean(setupRoot)+string(filepath.Separator)) {
			http.Error(w, "extension folder outside the install root", http.StatusBadRequest)
			return
		}
		if info["Exists"] != true {
			writeJSON(map[string]any{"ok": false, "error": "folder_missing", "folder": folder,
				"note": "the per-client extension folder was not prepared yet — ask the agent to prepare it"})
			return
		}
		if err := uiOpenInFileManager(cleanFolder); err != nil {
			writeJSON(map[string]any{"ok": false, "error": "open_failed", "folder": folder})
			return
		}
		writeJSON(map[string]any{"ok": true, "folder": folder})
	case "sendbox-auth":
		// The ONE canonical write the UI performs outside ui_inbox (spec §6.2 v1.3):
		// the App Password must never transit chat or any agent-readable queue, so
		// the bridge itself verifies SMTP+IMAP live and persists credentials (0600).
		slug := strings.TrimSpace(mStr(body, "slug"))
		emailAddr := strings.TrimSpace(mStr(body, "email"))
		appPassword, _ := body["app_password"].(string)
		if emailAddr == "" || strings.TrimSpace(appPassword) == "" {
			http.Error(w, "email + app_password required", http.StatusBadRequest)
			return
		}
		if slug == "" {
			// Non-tech users never see a "slug": same email -> re-auth the same
			// box; new email -> next free conventional name (sb-a, sb-b, ...).
			slug = resolveSendboxSlug(filepath.Join(c.Path, "outreach"), emailAddr)
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
		"Pending":  len(b.uiPendingDrafts(c)),
		"Features": uiFeaturesFor(c.Slug),
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

var uiTplFuncs = template.FuncMap{
	"shortid": shortID,
	// groups: distinct Group values in first-appearance order
	"groups": func(feats []map[string]any) []string {
		var out []string
		seen := map[string]bool{}
		for _, f := range feats {
			if g, _ := f["Group"].(string); !seen[g] {
				seen[g] = true
				out = append(out, g)
			}
		}
		return out
	},
	"featIn": func(feats []map[string]any, group string) []map[string]any {
		var out []map[string]any
		for _, f := range feats {
			if f["Group"] == group {
				out = append(out, f)
			}
		}
		return out
	},
}

var uiTpl = template.Must(template.New("ui").Funcs(uiTplFuncs).Parse(`
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
<a href="/ui/{{.Slug}}/reports">reports</a> · <a href="/ui/{{.Slug}}/campaigns">campaigns</a> · <a href="/ui/{{.Slug}}/crm">crm</a> · <a href="/ui/{{.Slug}}/approvals">approvals</a> · <a href="/ui/{{.Slug}}/sendboxes">sendboxes</a></div>
{{else}}<p class="mut">No clients yet.</p>{{end}}</div>
<h2>What this system can do</h2>
<div class="grid-cards">
<div class="card"><strong>Content pipeline</strong><br><span class="mut" style="font-size:.85rem">daily ideas · short videos (WideCast) · blog + social posts · private-source monitoring · daily/weekly HTML reports · analytics loop</span></div>
<div class="card"><strong>Outreach + CRM</strong><br><span class="mut" style="font-size:.85rem">cold-email campaigns · lead-list import · evidence-backed personalization · draft approvals · follow-up engine · CRM pipeline</span></div>
</div>
<p class="mut" style="font-size:.85rem">Open a client above for the full action cards — web-UI actions run right here; agent actions give you the exact phrase to paste into chat.</p>
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
<p><a href="/ui/{{.Client.Slug}}/reports">All reports</a> · <a href="/ui/{{.Client.Slug}}/campaigns">Campaigns</a> · <a href="/ui/{{.Client.Slug}}/crm">CRM</a> ·
<a href="/ui/{{.Client.Slug}}/approvals">Approvals{{if .Pending}} <strong>({{.Pending}})</strong>{{end}}</a> ·
<a href="/ui/{{.Client.Slug}}/shortlist">Shortlist</a> ·
<a href="/ui/{{.Client.Slug}}/sendboxes">Sendboxes</a></p>

<h2>Actions</h2>
<p class="mut" style="margin-top:-.3rem">Cards marked <span class="pill band-high">web UI</span> run right here. Cards marked <span class="pill">agent chat</span> start by pasting the phrase into the named chat session.</p>
{{$slug := .Client.Slug}}
{{range $grp := groups .Features}}
<h3 style="font-size:.95rem;margin:.9rem 0 .3rem">{{$grp}}</h3>
<div class="grid-cards">
{{range featIn $.Features $grp}}
<div class="card">
<strong>{{.Title}}</strong>
{{if eq .Kind "ui"}}<span class="pill band-high">web UI</span>{{else}}<span class="pill">agent chat</span>{{end}}<br>
<span class="mut" style="font-size:.82rem">{{.Value}}</span><br>
{{if eq .Kind "ui"}}
<a role="button" class="ok" style="display:inline-block;margin-top:.5rem;padding:.3rem .8rem;font-size:.82rem" href="/ui/{{$slug}}/{{.Href}}">Open</a>
{{else}}
<code style="font-size:.78rem;display:inline-block;margin-top:.5rem">{{.Phrase}}</code>
<button class="copy-phrase" data-phrase="{{.Phrase}}" style="padding:.15rem .6rem;font-size:.75rem;margin:.3rem 0 0">Copy</button>
<br><span class="mut" style="font-size:.72rem">→ paste into {{.Session}}</span>
{{end}}
</div>
{{end}}
</div>
{{end}}
<script>
document.addEventListener('click',function(e){var b=e.target.closest('.copy-phrase');if(!b)return;
 navigator.clipboard.writeText(b.dataset.phrase).then(function(){var t=b.textContent;b.textContent='Copied';setTimeout(function(){b.textContent=t},1200)})});
</script>

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
<h2>Contacts <span class="mut" style="font-size:.8rem">({{len .Contacts}}) — click a row for the full profile + latest activities</span></h2>
<div class="wrap"><table><tr><th>name</th><th>email</th><th>phone</th><th>social</th><th>vertical</th><th>state</th></tr>
{{$slug := .Client.Slug}}{{range .Contacts}}<tr style="cursor:pointer" onclick="location.href='/ui/{{$slug}}/contact/{{.ID}}'">
<td>{{if .Name}}<strong>{{.Name}}</strong>{{else}}<span class="mut" title="{{.ID}}">{{.ShortID}}</span>{{end}}</td>
<td class="mut">{{if .Email}}{{.Email}}{{else}}—{{end}}</td>
<td class="mut">{{if .Phone}}{{.Phone}}{{else}}—{{end}}</td>
<td class="mut">{{if .Social}}<a href="{{.Social}}" target="_blank" rel="noopener" onclick="event.stopPropagation()">link ↗</a>{{else}}—{{end}}</td>
<td>{{.Vertical}}</td>
<td>{{if .Band}}<span class="pill band-high">enriched</span>{{else if .SeedsUnresolved}}<span class="pill band-review_carefully">seed: trace origin</span>{{else}}<span class="pill">{{if .Stage}}{{.Stage}}{{else}}new{{end}}</span>{{end}}</td>
</tr>{{else}}<tr><td colspan="6" class="mut">no contacts yet — import a list or run discovery</td></tr>{{end}}</table></div>
{{template "foot" .}}{{end}}

{{define "contact"}}{{template "head" .}}
{{$c := .C}}
<p><a href="/ui/{{.Client.Slug}}/crm">← CRM</a></p>
{{if $c.DuplicateSuspects}}
<div class="card" style="border-color:#b8860b">
<strong>⚠ Possible duplicate</strong> <span class="mut">— shares an identity with:</span>
{{range $c.DuplicateSuspects}}<div style="margin-top:.25rem"><a href="/ui/{{$.Client.Slug}}/contact/{{.id}}"><code style="font-size:.8rem">{{.id}}</code></a> <span class="mut" style="font-size:.82rem">(shared {{.via}}: {{.value}})</span></div>{{end}}
<p class="mut" style="font-size:.78rem;margin-bottom:0">Same person → tell the agent: <code>merge these contacts</code>. Different people → <code>clear the duplicate flag</code>. Suspected duplicates are held out of campaign queues until resolved.</p>
</div>
{{end}}
<div class="card">
<div style="display:flex;justify-content:space-between;flex-wrap:wrap;gap:.5rem;align-items:baseline">
<div><strong style="font-size:1.15rem">{{if $c.Name}}{{$c.Name}}{{else}}<span class="mut">{{shortid $c.ID}}</span>{{end}}</strong>
{{if $c.Company}} <span class="mut">· {{$c.Company}}{{if $c.Role}}, {{$c.Role}}{{end}}</span>{{end}}</div>
<div>
{{if $c.Band}}<span class="pill band-high">enriched · {{$c.Band}}</span>{{else}}<span class="pill band-review_carefully">not enriched yet</span>{{end}}
{{if $c.StillActive}}<span class="pill">{{$c.StillActive}}</span>{{end}}
{{if eq $c.SequenceState "frozen"}}<span class="pill">sequence frozen (replied)</span>{{end}}
</div></div>
{{if $c.OneLiner}}<p class="mut" style="margin:.4rem 0 0">{{$c.OneLiner}}</p>{{end}}
</div>

<div class="grid-cards">
<div class="card"><strong>Identities</strong>
<table style="margin-top:.4rem"><tbody>
{{range $c.Emails}}<tr><td class="mut">email</td><td>{{.address}} <span class="mut">({{.status}})</span></td></tr>{{end}}
{{range $c.Phones}}<tr><td class="mut">phone</td><td>{{.number}}</td></tr>{{end}}
{{range $c.Socials}}<tr><td class="mut">{{index . 0}}</td><td><a href="{{index . 1}}" target="_blank" rel="noopener">{{index . 1}}</a></td></tr>{{end}}
{{if $c.Website}}<tr><td class="mut">website</td><td><a href="{{$c.Website}}" target="_blank" rel="noopener">{{$c.Website}}</a></td></tr>{{end}}
{{if and (not $c.Emails) (not $c.Phones) (not $c.Socials) (not $c.Website)}}<tr><td colspan="2" class="mut">no reachable identity yet — enrichment must resolve one</td></tr>{{end}}
</tbody></table></div>

{{if $c.Seeds}}<div class="card"><strong>Content clues (seeds)</strong>
<span class="mut" style="font-size:.8rem"> — traced back to a profile during enrichment</span>
<table style="margin-top:.4rem"><tbody>
{{range $c.Seeds}}<tr><td class="mut">{{.kind}}{{if .platform}} · {{.platform}}{{end}}</td>
<td><a href="{{.url}}" target="_blank" rel="noopener">{{.url}}</a>
{{if eq (printf "%v" .status) "resolved"}}<span class="pill band-high">resolved</span>{{else}}<span class="pill band-review_carefully">unresolved</span>{{end}}</td></tr>{{end}}
</tbody></table></div>{{end}}
</div>

<h2>Latest activities <span class="mut" style="font-size:.8rem">— the proof-of-life hooks used to personalize email</span></h2>
{{if $c.Hooks}}
<div class="wrap"><table><tr><th>signal</th><th>what</th><th>observed</th><th>evidence</th><th>used</th></tr>
{{range $c.Hooks}}<tr>
<td><span class="pill">{{.Type}}</span></td>
<td>{{.Summary}}</td>
<td class="mut">{{if .Observed}}{{.Observed}}{{else}}<span title="recency unverified">?</span>{{end}}</td>
<td>{{if .URL}}<a href="{{.URL}}" target="_blank" rel="noopener">source ↗</a>{{else}}<span class="mut">—</span>{{end}}</td>
<td class="mut">{{range .UsedIn}}{{.}} {{else}}—{{end}}</td>
</tr>{{end}}</table></div>
{{if $c.HooksRefreshed}}<p class="mut" style="font-size:.8rem">hooks refreshed {{$c.HooksRefreshed}}</p>{{end}}
{{else}}
<p class="mut">No hooks yet. These are the recent, evidenced signals (a new listing, a post, a review, an award) that make each email genuinely personal — enrichment fills them in. Run the client's daily task, or tell the agent "enrich my leads".</p>
{{end}}

{{if $c.DoNotMention}}<p class="mut" style="font-size:.8rem">Do not mention: {{range $c.DoNotMention}}{{.}}; {{end}}</p>{{end}}

<h2>Activity timeline</h2>
{{if $c.Activities}}
<div class="wrap"><table><tr><th>when</th><th>event</th><th>detail</th><th>by</th></tr>
{{range $c.Activities}}<tr><td class="mut">{{.At}}</td><td><span class="pill">{{.Type}}</span></td><td>{{.Summary}}</td><td class="mut">{{.By}}</td></tr>{{end}}</table></div>
{{else}}<p class="mut">No activity recorded yet (sends, replies, stage changes appear here).</p>{{end}}
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

{{define "campaigns"}}{{template "head" .}}
<p><a href="/ui/{{.Client.Slug}}">← {{.Client.Slug}}</a></p>
{{$slug := .Client.Slug}}
{{if .Rows}}
<div class="grid-cards">
{{range .Rows}}
<div class="card" style="cursor:pointer" onclick="location.href='/ui/{{$slug}}/campaign/{{.Slug}}'">
<div style="display:flex;justify-content:space-between;align-items:baseline;gap:.5rem">
<strong>{{.Slug}}</strong>
{{if eq .Status "paused"}}<span class="pill band-review_carefully">⏸ paused</span>{{else}}<span class="pill band-high">active</span>{{end}}
</div>
<span class="mut" style="font-size:.82rem">{{.GoalType}}{{if .Objective}} · {{.Objective}}{{end}}</span>
<div class="mut" style="font-size:.8rem;margin-top:.5rem">
today {{.UsedToday}}/{{.Quota}} drafts
{{if .Pending}} · <a href="/ui/{{$slug}}/approvals" onclick="event.stopPropagation()"><strong>{{.Pending}} awaiting approval</strong></a>{{end}}
· {{.Sent}} sent{{if .LastSent}} (last {{.LastSent}}){{end}}
</div>
</div>
{{end}}
</div>
{{else}}<p class="mut">No campaigns yet — tell the agent: <code>set up a cold-email campaign</code> (3 questions and it runs).</p>{{end}}
{{template "foot" .}}{{end}}

{{define "campaign"}}{{template "head" .}}
<p><a href="/ui/{{.Client.Slug}}/campaigns">← campaigns</a></p>
<div class="card">
<div style="display:flex;justify-content:space-between;flex-wrap:wrap;gap:.6rem;align-items:center">
<div>
{{if eq .Status "paused"}}<span class="pill band-review_carefully">⏸ paused — nothing drafts or sends</span>{{else}}<span class="pill band-high">active</span>{{end}}
<span class="mut" style="font-size:.85rem"> · today {{.UsedToday}}/{{.Quota}} drafts{{if .Pending}} · <a href="/ui/{{.Client.Slug}}/approvals">{{.Pending}} awaiting approval</a>{{end}} · audience: {{.Segment}}{{if .Sendboxes}} · boxes: {{range .Sendboxes}}<code style="font-size:.75rem">{{.}}</code> {{end}}{{end}}</span>
</div>
{{if eq .Status "paused"}}
<button class="ok" id="toggle" data-to="active">▶ Resume campaign</button>
{{else}}
<button id="toggle" data-to="paused">⏸ Pause campaign</button>
{{end}}
</div>
</div>

<form id="campform">
<h2>Goal <span class="mut" style="font-size:.8rem">— what every email in this campaign is trying to achieve</span></h2>
<div class="card">
<label>Goal type
<select id="f-goaltype">{{$gt := .GoalType}}{{range .GoalTypes}}<option value="{{.}}"{{if eq . $gt}} selected{{end}}>{{.}}</option>{{end}}</select></label>
<label>Objective <span class="mut">(one line — what success looks like)</span>
<input id="f-objective" type="text" value="{{.Objective}}" placeholder="e.g. book 5 intro calls with realtors this month"></label>
<label>Offer <span class="mut">(what you're actually proposing to them)</span>
<textarea id="f-offer" style="min-height:70px">{{.Offer}}</textarea></label>
<label>Value proposition <span class="mut">(why it's worth their time)</span>
<textarea id="f-valueprop" style="min-height:70px">{{.ValueProp}}</textarea></label>
<label>Proof points <span class="mut">(one per line — real, verifiable)</span>
<textarea id="f-proof" style="min-height:70px">{{.Proof}}</textarea></label>
<label>Call-to-action text <span class="mut">(the one ask at the end of the email)</span>
<input id="f-cta" type="text" value="{{.CTAText}}" placeholder="e.g. Worth a quick look?"></label>
</div>

<h2>Companion link <span class="mut" style="font-size:.8rem">— the support link each email carries (demo page, sample video...)</span></h2>
<div class="card">
<label>How to get the link for each lead <span class="mut">(write it like instructions to an assistant: a fixed link, a per-language rule, or a step-by-step recipe — the agent follows exactly this)</span>
<textarea id="f-comp-instructions" style="min-height:90px" placeholder="e.g. use https://leadup.example/demo for every lead&#10;or: US lead → https://…/en, Vietnamese lead → https://…/vi&#10;or: personalize template X from the dossier, upload via API Y, use the returned URL">{{.CompanionInstructions}}</textarea></label>
<label>If getting the link fails
<select id="f-comp-onfail">
<option value="skip"{{if eq .CompanionOnFail "skip"}} selected{{end}}>skip that lead (no email without the link)</option>
<option value="default_link"{{if eq .CompanionOnFail "default_link"}} selected{{end}}>fall back to the default link below</option>
</select></label>
<label>Default link <span class="mut">(required when falling back)</span>
<input id="f-comp-default" type="text" value="{{.CompanionDefault}}" placeholder="https://…"></label>
<p class="mut" style="font-size:.78rem;margin-bottom:0">Leave the instructions empty to send emails without a companion link.</p>
</div>

<h2>Sending</h2>
<div class="card">
<label>Daily draft budget <span class="mut">(max new drafts per day for this campaign)</span>
<input id="f-quota" type="number" min="1" max="500" value="{{.Quota}}" style="width:8rem"></label>
</div>

<div class="acts">
<button class="ok" type="submit">Save changes</button>
<span id="savemsg" class="mut"></span>
</div>
</form>
<script>
var CLIENT="{{.Client.Slug}}", CAMP="{{.Slug}}";
function postUpdate(patch, done){
 fetch('/api/ui/'+CLIENT+'/campaign-update',{method:'POST',headers:{'Content-Type':'application/json'},
  body:JSON.stringify({slug:CAMP,patch:patch})})
 .then(function(r){return r.json()})
 .then(function(j){done(j)})
 .catch(function(e){done({ok:false,error:e.message})});
}
document.getElementById('toggle').addEventListener('click',function(e){
 e.preventDefault();var to=this.dataset.to;var self=this;self.disabled=true;self.setAttribute('aria-busy','true');
 postUpdate({status:to},function(j){ if(j.ok){location.reload()} else {self.disabled=false;self.removeAttribute('aria-busy');alert(j.error)} })});
document.getElementById('campform').addEventListener('submit',function(e){
 e.preventDefault();
 var btn=this.querySelector('button[type=submit]');btn.disabled=true;btn.setAttribute('aria-busy','true');
 var msg=document.getElementById('savemsg');msg.textContent='Saving…';
 var proof=document.getElementById('f-proof').value.split('\n').map(function(s){return s.trim()}).filter(Boolean);
 var instructions=document.getElementById('f-comp-instructions').value.trim();
 var goal={goal_type:document.getElementById('f-goaltype').value,
  objective:document.getElementById('f-objective').value.trim(),
  offer:document.getElementById('f-offer').value.trim(),
  value_proposition:document.getElementById('f-valueprop').value.trim(),
  proof_points:proof,
  cta:{text:document.getElementById('f-cta').value.trim()},
  companion_doc: instructions ? {instructions:instructions,
    on_fail:document.getElementById('f-comp-onfail').value,
    default_link:document.getElementById('f-comp-default').value.trim()} : null};
 postUpdate({goal:goal, daily_quota:parseInt(document.getElementById('f-quota').value,10)},
  function(j){btn.disabled=false;btn.removeAttribute('aria-busy');
   if(j.ok){msg.textContent = (j.changed&&j.changed.length) ? '✓ saved ('+j.changed.join(', ')+') — takes effect from the next run; the agent is notified' : '✓ nothing changed';}
   else{msg.textContent='✗ '+j.error}})});
</script>
</main></body></html>{{end}}

{{define "sendboxes"}}{{template "head" .}}
<p><a href="/ui/{{.Client.Slug}}">← {{.Client.Slug}}</a></p>
{{if .Sendboxes}}
<div class="wrap"><table><tr><th>name</th><th>email</th><th>status</th><th>quota/day</th><th>warmup</th><th>last sync</th><th></th></tr>
{{range .Sendboxes}}<tr>
<td><code>{{.slug}}</code></td><td>{{.email}}</td>
<td><span class="pill{{if eq .status "healthy"}} band-high{{else}} band-review_carefully{{end}}">{{.status}}</span></td>
<td>{{.quota_today}}</td><td class="mut">{{.warmup_stage}}</td>
<td class="mut">{{.last_successful_sync_ts}}</td>
<td><a href="#connect" class="pick-box" data-email="{{.email}}">connect / re-auth</a></td>
</tr>{{end}}</table></div>
{{else}}<p class="mut">No sendboxes yet — connect the first one below.</p>{{end}}

<h2 id="connect">Connect a sending mailbox (Gmail App Password)</h2>
<div class="card" style="max-width:560px">
<form id="authform">
<label>Gmail address
<input id="f-email" type="email" placeholder="you@gmail.com" required></label>
<label>App Password <span class="mut">(16 characters — Google Account → Security → App passwords)</span>
<input id="f-pass" type="password" autocomplete="off" placeholder="xxxx xxxx xxxx xxxx" required></label>
<button class="ok" type="submit">Connect &amp; verify</button>
<span id="authmsg" class="mut"></span>
</form>
<p class="mut" style="font-size:.8rem">Reconnecting an address in the list updates that same mailbox; a new address is added
automatically under the next free internal name — nothing else to fill in.</p>
<p class="mut" style="font-size:.8rem;margin-bottom:0">The password goes from this page straight to Gmail over TLS and is stored only on this machine
(<code>sendboxes/&lt;slug&gt;/credentials.json</code>, permissions 0600). Never paste an App Password
into the agent chat — this page is the one intended place for it.</p>
</div>
<script>
var CLIENT="{{.Client.Slug}}";
document.querySelectorAll('.pick-box').forEach(function(a){a.addEventListener('click',function(){
 document.getElementById('f-email').value=this.dataset.email;
 document.getElementById('f-pass').focus()})});
document.getElementById('authform').addEventListener('submit',function(e){
 e.preventDefault();
 var btn=this.querySelector('button');btn.disabled=true;btn.setAttribute('aria-busy','true');
 var msg=document.getElementById('authmsg');msg.textContent='Verifying SMTP + IMAP with Gmail…';
 fetch('/api/ui/'+CLIENT+'/sendbox-auth',{method:'POST',headers:{'Content-Type':'application/json'},
  body:JSON.stringify({email:document.getElementById('f-email').value.trim(),
   app_password:document.getElementById('f-pass').value})})
 .then(function(r){return r.json()})
 .then(function(j){
  if(j.ok){msg.textContent='✓ connected ('+j.email+' as '+j.sendbox+', quota '+j.quota_today+'/day)';
   document.getElementById('f-pass').value='';setTimeout(function(){location.reload()},900)}
  else{btn.disabled=false;btn.removeAttribute('aria-busy');
   msg.textContent='✗ '+(j.error||'failed')+(j.detail?' — '+j.detail:'')+' (check the address and the App Password)'}})
 .catch(function(err){btn.disabled=false;btn.removeAttribute('aria-busy');msg.textContent='✗ '+err.message})});
</script>
</main></body></html>{{end}}

{{define "extension"}}{{template "head" .}}
<p><a href="/ui/{{.Client.Slug}}">← {{.Client.Slug}}</a></p>
<div class="card" style="max-width:640px">
{{if .Ext.CheckedIn}}
<p><span class="pill band-high">✓ extension connected</span> <span class="mut">last check-in {{.Ext.LastCheck}}{{if .Ext.Instance}} · {{.Ext.Instance}}{{end}}</span></p>
<p class="mut">The Chrome extension for this client is talking to the collector. Nothing to do here.</p>
{{else}}
<p><span class="pill band-review_carefully">not connected yet</span> <span class="mut">no check-in from this client's extension since the bridge started</span></p>
<h2 style="margin-top:.6rem">Install in 3 steps — no path typing</h2>
<ol>
<li><button class="ok" id="reveal" style="padding:.3rem .9rem">Open the extension folder</button>
<span id="revealmsg" class="mut"></span><br>
<span class="mut" style="font-size:.8rem">Finder/Explorer opens the exact folder. Keep that window visible.</span></li>
<li>In the Chrome profile for <strong>{{.Client.Slug}}</strong>, open <code>chrome://extensions</code> and switch on <strong>Developer mode</strong> (top right).</li>
<li><strong>Drag the opened folder</strong> from Finder/Explorer and drop it anywhere on the <code>chrome://extensions</code> page — that installs it (same as "Load unpacked", minus the file picker). This page flips to <span class="pill band-high">✓ connected</span> on its own once the extension checks in.</li>
</ol>
<p class="mut" style="font-size:.8rem">Manual fallback: click "Load unpacked", press <kbd>Cmd</kbd>+<kbd>Shift</kbd>+<kbd>G</kbd> (Mac) or paste into the address bar (Windows), then paste this path:<br>
<code id="extpath" style="font-size:.75rem">{{.Ext.Folder}}</code>
<button class="copy-phrase" data-phrase="{{.Ext.Folder}}" style="padding:.1rem .5rem;font-size:.7rem">Copy</button></p>
{{end}}
</div>
<script>
var CLIENT="{{.Client.Slug}}";
var rv=document.getElementById('reveal');
if(rv){rv.addEventListener('click',function(){
 fetch('/api/ui/'+CLIENT+'/reveal-extension',{method:'POST',headers:{'Content-Type':'application/json'},body:'{}'})
 .then(function(r){return r.json()})
 .then(function(j){document.getElementById('revealmsg').textContent=j.ok?'✓ folder opened':'✗ '+(j.note||j.error)})
 .catch(function(e){document.getElementById('revealmsg').textContent='✗ '+e.message})})}
document.addEventListener('click',function(e){var b=e.target.closest('.copy-phrase');if(!b)return;
 navigator.clipboard.writeText(b.dataset.phrase).then(function(){var t=b.textContent;b.textContent='Copied';setTimeout(function(){b.textContent=t},1200)})});
</script>
{{template "foot" .}}{{end}}
`))
