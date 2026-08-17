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
	"math"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
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
	mux.HandleFunc("/ui/assets/app.css", handleUIAppCSS)
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
	// NOT extension_health.jsonl. It is a HEARTBEAT, not content: every installed extension
	// appends to it every ~5s just to say it is alive, so including it made the fingerprint
	// change perpetually and every open page reload itself every few seconds — forever, on
	// every screen, wiping whatever the operator was typing. Measured: that file moved three
	// times in eight seconds while every other stamped path sat still. Liveness belongs to the
	// "live" dot and /status, not to a page reload.
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
	RampStep int
	RampMax  int
	Client   string
	Slug     string
	Email    string
	Status   string
	Quota    string
	Warmup   string
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
			row.Quota = fmt.Sprintf("%d", effectiveQuota(sb, ""))
			if r := mMap(sb, "warmup_ramp"); len(r) > 0 {
				row.Warmup = fmt.Sprintf("+%d/day → %d", mInt(r, "step_per_day", 0), mInt(r, "max_quota", 0))
				row.RampStep = mInt(r, "step_per_day", 0)
				row.RampMax = mInt(r, "max_quota", 0)
			}
			out = append(out, row)
		}
	}
	return out
}

type uiContact struct {
	N               int // 1-based row number, continuous across pages
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
	CreatedAt       string // RFC3339; drives the lead-growth panel
}

// uiLeadGrowth summarises how the lead list is GROWING, which is a different question from
// how big it is. Bucketed by week rather than by day on purpose: leads arrive in bursts (an
// import, a scan harvest), so a daily chart is mostly empty columns and reads as broken —
// measured on the live list, all 488 contacts landed on two days.
type uiLeadGrowth struct {
	Total         int
	ThisWeek      int
	LastWeek      int
	Delta         int    // ThisWeek - LastWeek
	DaysSinceLast int    // -1 when nothing has ever been added
	LastAdded     string // YYYY-MM-DD of the most recent contact
	Weeks         []uiLeadWeek
	Max           int
}

type uiLeadWeek struct {
	Label string // week starting, e.g. "Jul 28"
	Count int
	Pct   int // bar height as a percentage of Max
}

// leadGrowth buckets contacts into the last 12 ISO weeks (Monday-based) by created_at.
func leadGrowth(all []uiContact, now time.Time) uiLeadGrowth {
	const weeks = 12
	g := uiLeadGrowth{Total: len(all), DaysSinceLast: -1}
	now = now.Local()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	// Monday of the current week.
	offset := (int(today.Weekday()) + 6) % 7
	thisMonday := today.AddDate(0, 0, -offset)

	counts := make([]int, weeks) // counts[weeks-1] == current week
	var newest time.Time
	for _, ct := range all {
		if ct.CreatedAt == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, ct.CreatedAt)
		if err != nil {
			continue
		}
		t = t.Local()
		if t.After(newest) {
			newest = t
		}
		day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
		// Bucket by the Monday of the contact's own week, then count whole weeks back.
		// Rounding the day difference keeps a DST week (167h or 169h) from landing in the
		// neighbouring bucket.
		cMonday := day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
		back := int(math.Round(thisMonday.Sub(cMonday).Hours()/24)) / 7
		idx := weeks - 1 - back
		if idx >= 0 && idx < weeks {
			counts[idx]++
		}
	}
	for i, n := range counts {
		start := thisMonday.AddDate(0, 0, -7*(weeks-1-i))
		if n > g.Max {
			g.Max = n
		}
		g.Weeks = append(g.Weeks, uiLeadWeek{Label: start.Format("Jan 2"), Count: n})
	}
	for i := range g.Weeks {
		if g.Max > 0 {
			g.Weeks[i].Pct = g.Weeks[i].Count * 100 / g.Max
		}
	}
	g.ThisWeek = counts[weeks-1]
	if weeks >= 2 {
		g.LastWeek = counts[weeks-2]
	}
	g.Delta = g.ThisWeek - g.LastWeek
	if !newest.IsZero() {
		g.LastAdded = newest.Format("2006-01-02")
		d := time.Date(newest.Year(), newest.Month(), newest.Day(), 0, 0, 0, 0, newest.Location())
		g.DaysSinceLast = int(today.Sub(d).Hours() / 24)
	}
	return g
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
		ct := uiContact{ID: id, ShortID: shortID(id), Name: contactName(doc), CreatedAt: mStr(doc, "created_at")}
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
	clients := b.uiClients()
	b.uiRender(w, "home", map[string]any{
		"Title":    "Solo Agency",
		"Clients":  clients,
		"Jobs":     b.uiJobs(),
		"Stats":    b.uiHomeStats(clients),
		"Features": uiFeaturesFor("{client}"),
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
	case len(parts) == 1 && parts[0] == "settings":
		b.uiRenderSettings(w)
	case len(parts) == 1 && parts[0] == "fleet":
		b.uiRenderFleet(w)
	case len(parts) == 1 && parts[0] != "":
		b.uiRenderClient(w, parts[0])
	case len(parts) == 2 && parts[1] == "reports":
		b.uiRenderReports(w, parts[0])
	case len(parts) == 2 && parts[1] == "crm":
		b.uiRenderCRM(w, parts[0], r)
	case len(parts) == 3 && parts[1] == "contact":
		b.uiRenderContact(w, parts[0], parts[2])
	case len(parts) == 2 && parts[1] == "campaigns":
		b.uiRenderCampaigns(w, parts[0])
	case len(parts) == 2 && parts[1] == "sent":
		b.uiRenderSent(w, parts[0])
	case len(parts) == 4 && parts[1] == "campaign" && parts[3] == "harvest":
		b.uiRenderHarvest(w, parts[0], parts[2], r)
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

func handleUIAppCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	// no-cache: after a bridge upgrade the restyled pages must not render with
	// the previous binary's stylesheet for an hour (localhost, ~15KB, cheap)
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write([]byte(appCSS))
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
	Icon    string
}

var uiFeatures = []uiFeature{
	{"Content pipeline", "Run today's report", "Fresh ideas, leads and ready-to-post drafts for this client, right now, no schedule needed", "agent", "", "run today's content for {client}", "a NEW chat session (automation)", "calendar"},
	{"Content pipeline", "Make a video", "Turn an approved idea into a finished short video through WideCast", "agent", "", "make a video from today's best idea", "a NEW chat session (automation)", "video"},
	{"Content pipeline", "Blog + social posts", "One idea becomes a blog post and platform-ready social captions", "agent", "", "write the blog and social posts", "a NEW chat session (automation)", "article"},
	{"Content pipeline", "Private-source discovery", "Find the groups and communities your audience gathers in, from places you already joined", "agent", "", "run discovery", "the shared SETUP session", "radar"},
	{"Content pipeline", "Latest reports", "Daily and weekly HTML reports: ideas, drafts, leads, opportunities, analytics", "ui", "reports", "", "", "file"},
	{"Content pipeline", "Chrome extension", "Drag-and-drop install for this client's collector extension, with a live connected check", "ui", "extension", "", "", "puzzle"},
	{"Outreach + CRM", "Create a cold-email campaign", "Personalized, evidence-backed cold email: 3 questions and it runs; nothing sends without your approval", "agent", "", "set up a cold-email campaign", "the shared SETUP session", "send"},
	{"Outreach + CRM", "Manage campaigns", "Edit each campaign's goal and companion link, change the daily budget, pause and resume", "ui", "campaigns", "", "", "adjust"},
	{"Outreach + CRM", "Import any lead list", "Emails, phones, reels, posts, profiles: every unique fragment becomes a record, deduped and suppression-checked", "agent", "", "import a list: <path to your CSV>", "the shared SETUP session", "upload"},
	{"Outreach + CRM", "Approve drafts in batch", "Tick the drafts you want, approve the whole batch in one click; edit, hold or reject the rest", "ui", "approvals", "", "", "checks"},
	{"Outreach + CRM", "Approve discovered sources", "Tick the monitoring shortlist the agent proposed after discovery", "ui", "shortlist", "", "", "list"},
	{"Outreach + CRM", "Connect a sendbox", "Paste the Gmail App Password here, never into chat; verified live over SMTP and IMAP", "ui", "sendboxes", "", "", "mail"},
	{"Outreach + CRM", "CRM pipeline", "Replies become deals moving through stages; every contact keeps its proof-of-life hooks", "ui", "crm", "", "", "kanban"},
	{"Outreach + CRM", "Distribution", "Everything this client has published: emails sent, comments and group posts, with their reply state", "ui", "sent", "", "", "send"},
}

func uiFeaturesFor(slug string) []map[string]any {
	out := make([]map[string]any, 0, len(uiFeatures))
	for _, f := range uiFeatures {
		out = append(out, map[string]any{
			"Group": f.Group, "Title": f.Title, "Value": f.Value, "Kind": f.Kind,
			"Href": f.Href, "Session": f.Session, "Icon": f.Icon,
			"Phrase": strings.ReplaceAll(f.Phrase, "{client}", slug),
		})
	}
	return out
}

// uiHomeStats aggregates read-only counts across all clients for the home
// hero: pending approvals, active campaigns, emails sent today.
func (b *bridge) uiHomeStats(clients []uiClient) map[string]any {
	pending, activeCamps, sentToday := 0, 0, 0
	today := todayStr("")
	for _, c := range clients {
		pending += len(b.uiPendingDrafts(c))
		store := newCrmStore(filepath.Join(c.Path, "outreach"))
		for _, cfg := range store.listCampaigns() {
			if strOr(mStr(cfg, "status"), "active") == "active" {
				activeCamps++
			}
		}
		for _, p := range store.allSentLogs("") {
			for _, r := range readJSONLines(p) {
				if mStr(r, "rfc_message_id") != "" && strings.HasPrefix(mStr(r, "sent_at"), today) {
					sentToday++
				}
			}
		}
	}
	return map[string]any{"Clients": len(clients), "Pending": pending,
		"Campaigns": activeCamps, "SentToday": sentToday}
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
		ch := strOr(mStr(cfg, "channel_strategy"), "email_first")
		row := map[string]any{
			"Slug": slug, "Status": strOr(mStr(cfg, "status"), "active"),
			"Channel":   ch,
			"GoalType":  mStr(mMap(cfg, "goal"), "goal_type"),
			"Objective": mStr(mMap(cfg, "goal"), "objective"),
			"Quota":     mInt(cfg, "daily_quota", 40),
			"UsedToday": 0,
			"Pending":   pendingByCamp[slug],
			"Sent":      0, "LastSent": "",
		}
		// A harvest campaign drafts nothing, approves nothing and sends nothing, so the
		// drafts/awaiting-approval/sent card reports every one of them as an idle campaign
		// forever. Show what it actually does: today's enrich count against the harvest
		// budget (NOT daily_quota, which no harvest code reads), what it kept, and what is
		// waiting for the agent's judgement.
		if ch == harvestChannel || ch == zillowChannel {
			row["IsHarvest"], row["Kept"], row["Await"], row["DaemonState"] = true, 0, 0, ""
			outreachDir := filepath.Join(c.Path, "outreach")
			if p, perr := withProgress(outreachDir, slug, func(*harvestProgress) error { return nil }); perr == nil {
				hc := harvestConfigFrom(cfg, loadSystemSettings(b.uiDataRoot))
				hc.Channel = ch
				if ch == zillowChannel {
					hc.Zillow = zillowConfigFrom(cfg)
				}
				st := b.harvestCommonStatus(p, hc, outreachDir)
				row["Quota"], row["UsedToday"] = st["day_budget"], st["day_enriched"]
				row["Kept"], row["Await"], row["DaemonState"] = st["kept"], st["await"], st["state"]
			}
			out = append(out, row)
			continue
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
		"Title": "Campaigns", "Client": c, "Rows": b.uiCampaignRows(c),
	})
}

// uiSlugify turns operator-typed text into a campaign slug. The slug becomes a DIRECTORY
// name, so anything that could escape the campaigns folder or collide with path syntax is
// dropped rather than escaped — a campaign called "../x" must be impossible, not merely ugly.
func uiSlugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		case r == '-' || r == '_' || r == ' ' || r == '.' || r == '/':
			if b.Len() > 0 && !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 60 {
		out = strings.Trim(out[:60], "-")
	}
	return out
}

// uiScannedGroups lists the Facebook groups this client's collector already scans daily.
// A comment campaign's group list is PREFILLED from here and then narrowed by the operator —
// it is not a live pointer at this list, because scanning a group for leads and commenting in
// it are different decisions with different risk.
func (b *bridge) uiScannedGroups(clientSlug string) []string {
	doc, err := loadCollectorConfig(b.cfg)
	if err != nil {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, c := range mapsOf(mList(doc, "clients")) {
		if mStr(c, "client_slug") != clientSlug {
			continue
		}
		for _, s := range mapsOf(mList(c, "sources")) {
			u := strings.TrimRight(strings.TrimSpace(mStr(s, "url")), "/")
			if u == "" || !strings.Contains(strings.ToLower(u), "facebook.com/groups/") {
				continue
			}
			if seen[strings.ToLower(u)] {
				continue
			}
			seen[strings.ToLower(u)] = true
			out = append(out, u)
		}
	}
	sort.Strings(out)
	return out
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
	// The message bank is what the operator actually wants to fill — the key
	// messages every email must land — and it was not on this page at all, while
	// proof_points (rarely applicable, and broken) was. Operator-authored entries
	// are the point: an agent-written bank is the vendor paraphrasing itself.
	var bankLines []string
	operatorMsgs := 0
	for _, m := range mapsOf(mList(goal, "message_bank")) {
		msg := mStr(m, "msg")
		if msg == "" {
			continue
		}
		if mStr(m, "source") == "operator" {
			operatorMsgs++
		} else {
			msg += "  (agent)"
		}
		bankLines = append(bankLines, msg)
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
		"AllSendboxes": func() []map[string]any {
			picked := map[string]bool{}
			for _, r := range mList(cfg, "sendboxes") {
				if sl, ok := r.(string); ok {
					picked[sl] = true
				}
			}
			boxes := b.uiClientSendboxes(c)
			for _, sb := range boxes {
				sb["picked"] = picked[mStr(sb, "slug")]
			}
			return boxes
		}(),
		"Channel": strOr(mStr(cfg, "channel_strategy"), "email_first"), "Channels": sortedChannelStrategies(),
		"Seeds": func() []string {
			var out []string
			for _, s := range mList(cfg, "seed_profiles") {
				out = append(out, sprint(s))
			}
			return out
		}(),
		"HarvestKeywords": func() string {
			var kw []string
			for _, k := range mList(mMap(cfg, "harvest"), "goal_keywords") {
				kw = append(kw, sprint(k))
			}
			return strings.Join(kw, ", ")
		}(),
		"HarvestDaily": func() string {
			if v := mInt(mMap(cfg, "harvest"), "daily_budget", 0); v > 0 {
				return fmt.Sprint(v)
			}
			return ""
		}(),
		"HarvestDefaults": map[string]any{"daily": loadSystemSettings(b.uiDataRoot).HarvestDailyBudget},
		"ZillowLocations": func() []string {
			var out []string
			for _, l := range mList(cfg, "zillow_locations") {
				out = append(out, sprint(l))
			}
			return out
		}(),
		"ZillowKeywords": func() string {
			var kw []string
			for _, k := range mList(cfg, "zillow_keywords") {
				kw = append(kw, sprint(k))
			}
			return strings.Join(kw, ", ")
		}(),
		"ZillowStatus": func() map[string]any {
			if mStr(cfg, "channel_strategy") != zillowChannel {
				return nil
			}
			outreachDir := filepath.Join(c.Path, "outreach")
			p, err := withProgress(outreachDir, camp, func(*harvestProgress) error { return nil })
			if err != nil {
				return nil
			}
			hc := harvestConfigFrom(cfg, loadSystemSettings(b.uiDataRoot))
			hc.Channel, hc.Zillow = zillowChannel, zillowConfigFrom(cfg)
			st := b.harvestCommonStatus(p, hc, outreachDir)
			zc := p.zillow()
			loc, kw := "", ""
			if zc.LocIdx < len(hc.Zillow.Locations) {
				loc = hc.Zillow.Locations[zc.LocIdx]
			}
			if zc.KwIdx < len(hc.Zillow.Keywords) {
				kw = hc.Zillow.Keywords[zc.KwIdx]
			}
			walk := "walking"
			switch {
			case zc.Exhausted:
				walk = "done — every location × keyword walked"
			case len(hc.Zillow.Locations) == 0 || len(hc.Zillow.Keywords) == 0:
				walk = "waiting for locations + keywords"
			}
			var blocked []string
			for box := range zc.Blocked {
				blocked = append(blocked, box)
			}
			sort.Strings(blocked)
			st["walk_state"] = walk
			st["location"] = loc
			st["keyword"] = kw
			st["page"] = zc.Page
			st["queries_done"] = len(zc.QueriesDone)
			st["queries_total"] = len(hc.Zillow.Locations) * len(hc.Zillow.Keywords)
			st["cards_seen"] = zc.CardsSeen
			st["blocked"] = strings.Join(blocked, ", ")
			return st
		}(),
		"HarvestStatus": func() map[string]any {
			if mStr(cfg, "channel_strategy") != harvestChannel {
				return nil
			}
			outreachDir := filepath.Join(c.Path, "outreach")
			p, err := withProgress(outreachDir, camp, func(*harvestProgress) error { return nil })
			if err != nil || len(p.Seeds) == 0 {
				return nil
			}
			pos := 0
			for i, s := range p.Seeds {
				if s.URL == p.CurrentSeed {
					pos = i + 1
				}
			}
			hc := harvestConfigFrom(cfg, loadSystemSettings(b.uiDataRoot))
			// Per-seed kept counts come from the client-wide seen registry.
			keptBySeed := map[string]int{}
			rejBySeed := map[string]int{}
			if reg, rerr := withSeen(outreachDir, func(*seenRegistry) error { return nil }); rerr == nil {
				for _, sp := range reg.Profiles {
					switch sp.Status {
					case "kept":
						keptBySeed[sp.Seed]++
					case "rejected":
						rejBySeed[sp.Seed]++
					}
				}
			}
			var seedRows []map[string]any
			for _, s := range p.Seeds {
				state := "waiting"
				switch {
				case s.Error != "":
					state = "error: " + s.Error
				case s.Exhausted:
					state = "done"
				case s.URL == p.CurrentSeed:
					state = "walking"
				}
				last := ""
				if s.LastLegAt != "" {
					if t, perr := time.Parse(time.RFC3339, s.LastLegAt); perr == nil {
						last = fmt.Sprintf("%.0fh ago", time.Since(t).Hours())
					}
				}
				seedRows = append(seedRows, map[string]any{"url": s.URL, "friends_seen": s.FriendsSeen,
					"legs": s.LegsDone, "state": state, "kept": keptBySeed[s.URL], "rejected": rejBySeed[s.URL],
					"last_leg": last, "box": s.LastLegBox})
			}
			st := b.harvestCommonStatus(p, hc, outreachDir)
			keptTotal, rejTotal := st["kept"].(int), st["rejected"].(int)
			live, quarantined, eligible := st["live_collectors"].(int), st["quarantined"].(int), st["eligible"].(int)
			state, ceiling, reason := st["state"].(string), st["ceiling"].(int), st["ceiling_reason"].(string)
			return map[string]any{"seed_pos": pos, "seed_total": len(p.Seeds), "friends_seen": p.Totals["friends_seen"],
				"already_known": p.Totals["already_known"], "retried": p.Totals["requeued"],
				"queue": len(p.Queue), "in_flight": len(p.InFlight), "await": len(p.AwaitDecision),
				"kept": keptTotal, "rejected": rejTotal,
				"day_enriched": p.DayEnriched, "day_budget": hc.DailyBudget, "live_collectors": live,
				"quarantined": quarantined, "eligible": eligible, "state": state,
				"ceiling": ceiling, "ceiling_reason": reason,
				"last_enrich": p.LastEnrichAt, "seeds": seedRows}
		}(),
		// A comment campaign watches its OWN group list. Offer the client's already-scanned
		// groups as the pick-list so the operator narrows an existing set instead of retyping
		// urls, but keep the two independent once saved.
		"Groups": func() []map[string]any {
			picked := map[string]bool{}
			for _, g := range mList(mMap(cfg, "audience"), "groups") {
				picked[strings.ToLower(strings.TrimRight(fmt.Sprint(g), "/"))] = true
			}
			seen := map[string]bool{}
			var out []map[string]any
			add := func(u string, inScan bool) {
				k := strings.ToLower(strings.TrimRight(u, "/"))
				if k == "" || seen[k] {
					return
				}
				seen[k] = true
				out = append(out, map[string]any{"url": u, "picked": picked[k], "scanned": inScan})
			}
			for _, u := range b.uiScannedGroups(slug) {
				add(u, true)
			}
			// A group the operator added by hand that is no longer in the scan list must not
			// silently vanish from the page — it is still being commented in.
			for _, g := range mList(mMap(cfg, "audience"), "groups") {
				add(strings.TrimRight(fmt.Sprint(g), "/"), false)
			}
			return out
		}(),
		"GoalType": mStr(goal, "goal_type"), "GoalTypes": sortedGoalTypes(),
		"GoalDesc":              mStr(goal, "description"),
		"Bank":                  strings.Join(bankLines, "\n"),
		"BankOperatorCount":     operatorMsgs,
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

// ---------- sent history (T1b: the centralized send registry) ----------

// uiSentRows joins the append-only sent_log with reply activities (and, once
// tracking lands, pulled open/click events) into one operator view: every email
// ever sent, which campaign/step/sendbox, and whether it got opened/clicked/
// replied. Read-only; caps at `limit` newest rows.
func (b *bridge) uiSentRows(c uiClient, limit int) (rows []map[string]any, total, replied int) {
	clientDir := filepath.Join(c.Path, "outreach")
	store := newCrmStore(clientDir)

	// lead -> newest reply ts (email_reply activities are the reply source of truth)
	replyTS := map[string]string{}
	if acts, err := store.a.readLog("activities", -1, nil); err == nil {
		for _, a := range acts {
			if mStr(a, "type") != "email_reply" {
				continue
			}
			lead := store.resolve(mStr(a, "contact_id"))
			if ts := mStr(a, "ts"); ts > replyTS[lead] {
				replyTS[lead] = ts
			}
		}
	}
	// tracking events pulled from WideCast land here (T2); keyed by msg ref,
	// which the sender stamps as the row's rfc_message_id
	opened, clicked := map[string]bool{}, map[string]bool{}
	for _, ev := range readJSONLines(filepath.Join(clientDir, "tracking", "events.jsonl")) {
		m := mStr(ev, "m")
		switch mStr(ev, "kind") {
		case "open":
			opened[m] = true
		case "click":
			clicked[m] = true
		}
	}

	nameCache := map[string]map[string]any{}
	for _, p := range store.allSentLogs("") {
		for _, r := range readJSONLines(p) {
			// A collector-published row (comment / group post) carries a channel and no
			// rfc_message_id; only reservation rows carry neither.
			chn := strOr(mStr(r, "channel"), "email")
			if mStr(r, "rfc_message_id") == "" && chn == "email" {
				continue // reservation rows etc.
			}
			total++
			lead := store.resolve(mStr(r, "lead_id"))
			ct, ok := nameCache[lead]
			if !ok {
				ct = store.getContact(lead)
				nameCache[lead] = ct
			}
			to, name := "", ""
			if ct != nil {
				name = contactName(ct)
				for _, e := range mapsOf(mList(mMap(ct, "identities"), "emails")) {
					if to == "" || mBool(e, "is_primary") {
						to = mStr(e, "address")
					}
				}
			}
			rid := mStr(r, "rfc_message_id")
			hasReply := replyTS[lead] != "" && replyTS[lead] >= mStr(r, "sent_at")
			if hasReply {
				replied++
			}
			if chn != "email" {
				// Nothing was addressed to a person: the target is the post or the group,
				// and the "sendbox" is the collector account that published it.
				to, name, lead = mStr(r, "target_url"), "", ""
			}
			rows = append(rows, map[string]any{
				"Lead": lead, "Name": name, "To": to, "Channel": chn,
				"Campaign": mStr(r, "campaign"), "Step": mInt(r, "step", 0),
				"Sendbox": strOr(mStr(r, "sendbox"), mStr(r, "collector")), "SentAt": mStr(r, "sent_at"),
				"Opened": opened[rid], "Clicked": clicked[rid], "Replied": hasReply,
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		return mStr(rows[i], "SentAt") > mStr(rows[j], "SentAt")
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, total, replied
}

func (b *bridge) uiRenderSent(w http.ResponseWriter, slug string) {
	c, ok := b.uiFindClient(slug)
	if !ok {
		http.Error(w, "unknown client", http.StatusNotFound)
		return
	}
	rows, total, replied := b.uiSentRows(c, 200)
	rate := ""
	if total > 0 {
		rate = fmt.Sprintf("%.1f%%", float64(replied)/float64(total)*100)
	}
	b.uiRender(w, "sent", map[string]any{
		"Title": "Sent", "Client": c, "Rows": rows,
		"Total": total, "Replied": replied, "Rate": rate,
		"Truncated": total > len(rows),
	})
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
		"Title": "Extension", "Client": c, "Ext": b.uiExtensionInfo(c),
	})
}

// uiClientSendboxes reads {ws}/outreach/sendboxes/sendboxes.json for one client,
// annotating each box with today's COMPUTED quota and the ramp description so
// the page shows the number the send engine will actually honor.
func (b *bridge) uiClientSendboxes(c uiClient) []map[string]any {
	p := filepath.Join(c.Path, "outreach", "sendboxes", "sendboxes.json")
	m, err := readJSONFile(p)
	if err != nil {
		return nil
	}
	boxes := mapsOf(mList(m, "sendboxes"))
	for _, sb := range boxes {
		sb["quota_effective"] = effectiveQuota(sb, "")
		sb["ramp_step"], sb["ramp_max"] = 0, 0
		sb["warmup_desc"] = mStr(sb, "warmup_stage")
		if r := mMap(sb, "warmup_ramp"); len(r) > 0 {
			sb["ramp_step"] = mInt(r, "step_per_day", 0)
			sb["ramp_max"] = mInt(r, "max_quota", 0)
			sb["warmup_desc"] = fmt.Sprintf("+%d/day → %d", mInt(r, "step_per_day", 0), mInt(r, "max_quota", 0))
		}
	}
	return boxes
}

func (b *bridge) uiRenderSendboxes(w http.ResponseWriter, slug string) {
	c, ok := b.uiFindClient(slug)
	if !ok {
		http.Error(w, "unknown client", http.StatusNotFound)
		return
	}
	store := newCrmStore(filepath.Join(c.Path, "outreach"))
	// every key present, else a missing one renders as a literal "<no value>"
	sender := map[string]any{"from_name": "", "from_title": "", "signature_block": ""}
	for k, v := range store.senderSignature() {
		sender[k] = v
	}
	b.uiRender(w, "sendboxes", map[string]any{
		"Title": "Sendboxes", "Client": c, "Sendboxes": b.uiClientSendboxes(c),
		"Sender": sender,
	})
}

// ---------- U2: interactive approvals + shortlist (writes go to ui_inbox only) ----------

type uiDraft struct {
	ID        string
	Channel   string // email | comment | post — the badge on the approvals card
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
			// The queue mixes channels now, and an email and a public group comment are
			// not the same decision to make at a glance. Older email drafts carry no
			// `channel` key at all, so absence means email.
			if ch, _ := doc["channel"].(string); ch != "" {
				dr.Channel = ch
			} else {
				dr.Channel = "email"
			}
			dr.Step = doc["step"]
			dr.To, _ = doc["to"].(string)
			dr.Subject, _ = doc["subject"].(string)
			dr.Body, _ = doc["body_text"].(string)
			dr.Band, _ = doc["confidence_band"].(string)
			dr.Companion, _ = doc["companion_url"].(string)
			// A publish that failed sends the draft back here for another decision; the
			// reason has to travel with it or the operator re-approves into the same wall.
			if bl, _ := doc["blocker"].(string); bl != "" {
				dr.Warnings = append(dr.Warnings, "last attempt failed: "+bl)
			}
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
		"Title": "Approvals", "Client": c, "Drafts": b.uiPendingDrafts(c),
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
		"Title": "Shortlist", "Client": c, "GeneratedAt": gen, "Candidates": cands,
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
	// Global (non-client) endpoint: the operator settings page saves here.
	if parts[0] == "system" && parts[1] == "settings" {
		var body map[string]any
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxJSONBodyBytes))
		if err := dec.Decode(&body); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		saved, err := saveSystemSettings(b.uiDataRoot, func(s *systemSettings) error {
			if v, ok := body["operator_email"].(string); ok {
				v = strings.TrimSpace(v)
				if v != "" && !strings.Contains(v, "@") {
					return fmt.Errorf("operator_email is not an email address")
				}
				s.OperatorEmail = v
			}
			intField := func(key string, dst *int, min, max int) error {
				v, ok := body[key].(float64)
				if !ok {
					return nil
				}
				n := int(v)
				if n < min || n > max {
					return fmt.Errorf("%s must be between %d and %d", key, min, max)
				}
				*dst = n
				return nil
			}
			if err := intField("max_concurrent_tasks", &s.MaxConcurrentTasks, 1, 500); err != nil {
				return err
			}
			if err := intField("slot_step_minutes", &s.SlotStepMinutes, 1, 240); err != nil {
				return err
			}
			if err := intField("slot_horizon_days", &s.SlotHorizonDays, 7, 366); err != nil {
				return err
			}
			if err := intField("default_task_duration_min", &s.DefaultTaskDurationMin, 5, 480); err != nil {
				return err
			}
			if err := intField("accountability_max_gap_hours", &s.AccountabilityMaxGapHours, 1, 720); err != nil {
				return err
			}
			// The upper bounds here are a safety rail, not a formality: these caps
			// govern how hard a real Facebook account is pushed, and a mistyped 500
			// would spend that account's standing in a single run. Refuse the value
			// rather than accept it and hope the campaign asks for less.
			if err := intField("dm_per_account_per_day", &s.DMPerAccountPerDay, 1, 50); err != nil {
				return err
			}
			if err := intField("comment_groups_per_account_per_day", &s.CommentGroupsPerAccountPerDay, 1, 20); err != nil {
				return err
			}
			if err := intField("comments_per_group_per_day", &s.CommentsPerGroupPerDay, 1, 10); err != nil {
				return err
			}
			if err := intField("posts_per_account_per_day", &s.PostsPerAccountPerDay, 1, 10); err != nil {
				return err
			}
			if err := intField("publish_gap_minutes", &s.PublishGapMinutes, 1, 240); err != nil {
				return err
			}
			if err := intField("harvest_daily_budget", &s.HarvestDailyBudget, 1, 5000); err != nil {
				return err
			}
			if err := intField("harvest_per_collector_budget", &s.HarvestPerCollectorBudget, 1, 2000); err != nil {
				return err
			}
			for key, dst := range map[string]*string{"harvest_quiet_from": &s.HarvestQuietFrom, "harvest_quiet_to": &s.HarvestQuietTo} {
				if v, ok := body[key].(string); ok {
					v = strings.TrimSpace(v)
					if v != "" && !runTimeRe.MatchString(v) {
						return fmt.Errorf("%s must be HH:MM", key)
					}
					if v != "" {
						*dst = v
					}
				}
			}
			return nil
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "settings": saved})
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
		// A COMMENT is published by the bridge the moment it is approved
		// (operator ruling 2026-08-17): the draft already carries the permalink and
		// the exact text, so nothing here needs a language model, and making the
		// operator wait for the next agent run would mean commenting on a thread
		// that has moved on. Email and DM keep going through the run's ingest,
		// untouched. The ui_inbox line above is still written first — it stays the
		// durable journal, and ingestUIDecisions skips a draft that is no longer
		// pending_approval, so it can never be applied twice.
		if decision == "approve" {
			if out, handled, aerr := b.approveCommentDraft(filepath.Join(c.Path, "outreach"), draftID, time.Now()); handled {
				if aerr != nil {
					http.Error(w, aerr.Error(), http.StatusInternalServerError)
					return
				}
				writeJSON(out)
				return
			}
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
	case "campaign-create":
		// Creating a campaign was CLI-only, so the operator had to go through the agent for
		// the one action that starts everything. Same store call, same validation — the agent
		// route keeps working unchanged, this just adds a second door to it.
		campSlug := uiSlugify(mStr(body, "slug"))
		if campSlug == "" {
			http.Error(w, "slug required (letters, numbers and dashes)", http.StatusBadRequest)
			return
		}
		channel := strings.TrimSpace(mStr(body, "channel_strategy"))
		if channel == "" {
			channel = "email_first"
		}
		cfg := map[string]any{"channel_strategy": channel}
		if d := strings.TrimSpace(mStr(body, "goal_description")); d != "" {
			cfg["goal"] = map[string]any{"description": d}
		}
		if q := int(asFloat(body["daily_quota"], 0)); q > 0 {
			cfg["daily_quota"] = q
		}
		store := newCrmStore(filepath.Join(c.Path, "outreach"))
		created, err := store.createCampaign(campSlug, cfg)
		if err != nil {
			writeJSON(map[string]any{"ok": false, "error": err.Error()})
			return
		}
		_ = appendUIInbox(filepath.Join(c.Path, "outreach", "ui_inbox", "campaign_edits.jsonl"),
			map[string]any{"ts": now, "campaign": campSlug, "created": true,
				"channel_strategy": mStr(created, "channel_strategy"), "ui_session": session})
		writeJSON(map[string]any{"ok": true, "campaign": campSlug,
			"url": "/ui/" + c.Slug + "/campaign/" + campSlug})
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
	case "sender-update":
		// Operator-owned sender identity: edits from_name / from_title /
		// signature_block in the client profile's sending_identity section —
		// the fields the brief hands the writer and the missing_signature gate
		// verifies. Same contract as campaign-update: instant effect (the file
		// is read fresh per brief/draft), agent notified via ui_inbox.
		store := newCrmStore(filepath.Join(c.Path, "outreach"))
		changed, err := store.senderIdentityUpdate(map[string]string{
			"from_name":       mStr(body, "from_name"),
			"from_title":      mStr(body, "from_title"),
			"signature_block": mStr(body, "signature_block"),
		})
		if err != nil {
			writeJSON(map[string]any{"ok": false, "error": err.Error()})
			return
		}
		if len(changed) > 0 {
			_ = appendUIInbox(filepath.Join(c.Path, "outreach", "ui_inbox", "profile_edits.jsonl"),
				map[string]any{"ts": now, "section": "sending_identity", "changed": changed, "ui_session": session})
		}
		writeJSON(map[string]any{"ok": true, "changed": changed,
			"note": "saved — every new draft signs as this from the next brief on; the agent is notified"})
	case "sendbox-quota":
		// Operator-owned daily cap + warm-up ramp. Playbook 02's ramp was
		// "documented policy the operator sets by hand" — prose, so no box was
		// ever advanced. The plan is config now: effectiveQuota() computes
		// today's cap everywhere (send cap, rotation, this page), nothing
		// mutates daily, restarts lose nothing.
		slug := strings.TrimSpace(mStr(body, "slug"))
		quota := mInt(body, "quota", 0)
		step := mInt(body, "step_per_day", 0)
		maxQ := mInt(body, "max_quota", 0)
		if slug == "" || quota < 1 || quota > 500 {
			http.Error(w, "slug + quota (1..500) required", http.StatusBadRequest)
			return
		}
		clientDir := filepath.Join(c.Path, "outreach")
		sb := getSendbox(clientDir, slug)
		if sb == nil {
			writeJSON(map[string]any{"ok": false, "error": "unknown sendbox " + slug})
			return
		}
		sb["quota_today"] = quota
		if step > 0 {
			if maxQ < quota {
				maxQ = quota
			}
			if maxQ > 500 {
				maxQ = 500
			}
			sb["warmup_ramp"] = map[string]any{"start_date": now[:10], "start_quota": quota,
				"step_per_day": step, "max_quota": maxQ}
		} else {
			delete(sb, "warmup_ramp")
		}
		if err := saveSendbox(clientDir, sb); err != nil {
			writeJSON(map[string]any{"ok": false, "error": err.Error()})
			return
		}
		_ = appendUIInbox(filepath.Join(clientDir, "ui_inbox", "sendbox_edits.jsonl"),
			map[string]any{"ts": now, "sendbox": slug, "quota": quota,
				"step_per_day": step, "max_quota": maxQ, "ui_session": session})
		note := "saved — takes effect immediately (send cap, rotation and this page all compute from it)"
		if strings.HasSuffix(strings.ToLower(mStr(sb, "domain")), "gmail.com") && (quota > 50 || maxQ > 50) {
			note += ". Caution: consumer @gmail.com boxes above ~50 cold emails/day get flagged (playbook 02 §8)"
		}
		writeJSON(map[string]any{"ok": true, "sendbox": slug, "effective_today": effectiveQuota(sb, ""), "note": note})
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

// uiRenderSettings — the operator's global system config page (/ui/settings):
// system parameters editable without chatting with an agent. Settings persist
// in {data root}/system_settings.json; agents and `tool schedule-slots` read
// the same file.
func (b *bridge) uiRenderSettings(w http.ResponseWriter) {
	settings := loadSystemSettings(b.uiDataRoot)
	var tasks []taskSlot
	if reg, err := withTaskSlots(b.uiDataRoot, func(*taskSlotRegistry) error { return nil }); err == nil {
		tasks = reg.Tasks
	}
	sortedTasks := append([]taskSlot{}, tasks...)
	sort.Slice(sortedTasks, func(i, j int) bool { return sortedTasks[i].TaskName < sortedTasks[j].TaskName })
	rows := make([]map[string]string, 0, len(sortedTasks))
	for _, t := range sortedTasks {
		cad := "monthly"
		if !t.Monthly {
			switch t.CadenceHours {
			case 24:
				cad = "daily"
			case 168:
				cad = "weekly"
			default:
				cad = fmt.Sprintf("%gh", t.CadenceHours)
			}
		}
		rows = append(rows, map[string]string{
			"Task": t.TaskName, "Client": t.ClientSlug, "Cadence": cad,
			"Time": t.RunTime, "Anchor": t.AnchorDate,
			"Duration": fmt.Sprintf("%d min", t.DurationMin), "Status": t.Status,
		})
	}
	b.uiRender(w, "settings", map[string]any{
		"Title": "Settings", "Settings": settings, "Tasks": rows,
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
	pending := len(b.uiPendingDrafts(c))
	activeCamps := 0
	store := newCrmStore(filepath.Join(c.Path, "outreach"))
	for _, cfg := range store.listCampaigns() {
		if strOr(mStr(cfg, "status"), "active") == "active" {
			activeCamps++
		}
	}
	b.uiRender(w, "client", map[string]any{
		"Title": "Overview", "Client": c, "Latest": latest,
		"Pending":  pending,
		"Features": uiFeaturesFor(c.Slug),
		"CStats": map[string]any{"Pending": pending, "Campaigns": activeCamps,
			"Contacts": len(b.uiContacts(c, 100000))},
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
	b.uiRender(w, "reports", map[string]any{"Title": "Reports", "Client": c, "Files": files})
}

const uiCRMPageSize = 100

// uiContactStats answers the questions the operator actually asks of this page:
// how many leads are there, and how many of them are ACTUALLY workable — an
// address to send to and a dossier to write from. "Both" is the number that
// matters, because either alone cannot produce an email.
type uiContactStats struct {
	Total, WithEmail, Enriched, Both, SeedUnresolved int
}

func uiComputeContactStats(all []uiContact) uiContactStats {
	var st uiContactStats
	st.Total = len(all)
	for _, ct := range all {
		hasEmail := ct.Email != ""
		enriched := ct.Band != ""
		if hasEmail {
			st.WithEmail++
		}
		if enriched {
			st.Enriched++
		}
		if hasEmail && enriched {
			st.Both++
		}
		if !enriched && ct.SeedsUnresolved > 0 {
			st.SeedUnresolved++
		}
	}
	return st
}

func (b *bridge) uiRenderCRM(w http.ResponseWriter, slug string, r *http.Request) {
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
	// count over everything, display one page: the stats must describe the whole
	// list, not the slice being looked at.
	all := b.uiContacts(c, 100000)
	stats := uiComputeContactStats(all)
	pages := (len(all) + uiCRMPageSize - 1) / uiCRMPageSize
	if pages < 1 {
		pages = 1
	}
	page := 1
	if n, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && n > 0 {
		page = n
	}
	if page > pages {
		page = pages
	}
	start := (page - 1) * uiCRMPageSize
	end := start + uiCRMPageSize
	if end > len(all) {
		end = len(all)
	}
	rows := all[start:end]
	for i := range rows {
		rows[i].N = start + i + 1 // continuous numbering, not per-page
	}
	var pageNums []int
	for i := 1; i <= pages; i++ {
		pageNums = append(pageNums, i)
	}
	b.uiRender(w, "crm", map[string]any{
		"Title": "CRM", "Client": c,
		"Contacts": rows, "StageOrder": order, "Stages": stages,
		"Growth": leadGrowth(all, time.Now()),
		"Stats":  stats, "Page": page, "Pages": pages, "PageNums": pageNums,
		"From": start + 1, "To": end,
		"PrevPage": page - 1, "NextPage": page + 1,
	})
}

func (b *bridge) uiRender(w http.ResponseWriter, page string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// presentation-only enrichment for the shared shell: which sidebar item is
	// active, and the pending-approvals badge when in a client context
	nav := page
	switch page {
	case "contact":
		nav = "crm"
	case "campaign", "harvest":
		nav = "campaigns"
	}
	data["NavPage"] = nav
	if c, ok := data["Client"].(uiClient); ok {
		if _, has := data["NavPending"]; !has {
			data["NavPending"] = len(b.uiPendingDrafts(c))
		}
	}
	if err := uiTpl.ExecuteTemplate(w, page, data); err != nil {
		log.Printf("ui: template %s: %v", page, err)
	}
}

// ---------- templates (embedded, no build chain) ----------

// uiIcons — inline SVG path data vendored from Tabler Icons (MIT,
// https://tabler.io/icons), one family, 24px grid, stroked.
var uiIcons = map[string]string{
	"bolt":        `<path d="M13 3l0 7l6 0l-8 11l0 -7l-6 0l8 -11"/>`,
	"adjustments": `<path d="M4 10a2 2 0 1 0 4 0a2 2 0 0 0 -4 0"/><path d="M6 4v4"/><path d="M6 12v8"/><path d="M10 16a2 2 0 1 0 4 0a2 2 0 0 0 -4 0"/><path d="M12 4v10"/><path d="M12 18v2"/><path d="M16 7a2 2 0 1 0 4 0a2 2 0 0 0 -4 0"/><path d="M18 4v1"/><path d="M18 9v11"/>`,
	"home":        `<path d="M5 12l-2 0l9 -9l9 9l-2 0"/><path d="M5 12v7a2 2 0 0 0 2 2h10a2 2 0 0 0 2 -2v-7"/><path d="M9 21v-6a2 2 0 0 1 2 -2h2a2 2 0 0 1 2 2v6"/>`,
	"activity":    `<path d="M3 12h4l3 8l4 -16l3 8h4"/>`,
	"heart":       `<path d="M19.5 12.572l-7.5 7.428l-7.5 -7.428a5 5 0 1 1 7.5 -6.566a5 5 0 1 1 7.5 6.572"/>`,
	"layout":      `<path d="M4 4m0 1a1 1 0 0 1 1 -1h14a1 1 0 0 1 1 1v2a1 1 0 0 1 -1 1h-14a1 1 0 0 1 -1 -1z"/><path d="M4 12m0 1a1 1 0 0 1 1 -1h6a1 1 0 0 1 1 1v6a1 1 0 0 1 -1 1h-6a1 1 0 0 1 -1 -1z"/><path d="M16 12l4 0"/><path d="M16 16l4 0"/><path d="M16 20l4 0"/>`,
	"send":        `<path d="M10 14l11 -11"/><path d="M21 3l-6.5 18a.55 .55 0 0 1 -1 0l-3.5 -7l-7 -3.5a.55 .55 0 0 1 0 -1l18 -6.5"/>`,
	"checks":      `<path d="M7 12l5 5l10 -10"/><path d="M2 12l5 5m5 -5l5 -5"/>`,
	"users":       `<path d="M9 7m-4 0a4 4 0 1 0 8 0a4 4 0 1 0 -8 0"/><path d="M3 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/><path d="M21 21v-2a4 4 0 0 0 -3 -3.85"/>`,
	"file":        `<path d="M14 3v4a1 1 0 0 0 1 1h4"/><path d="M17 21h-10a2 2 0 0 1 -2 -2v-14a2 2 0 0 1 2 -2h7l5 5v11a2 2 0 0 1 -2 2z"/><path d="M9 9l1 0"/><path d="M9 13l6 0"/><path d="M9 17l6 0"/>`,
	"list":        `<path d="M3.5 5.5l1.5 1.5l2.5 -2.5"/><path d="M3.5 11.5l1.5 1.5l2.5 -2.5"/><path d="M3.5 17.5l1.5 1.5l2.5 -2.5"/><path d="M11 6l9 0"/><path d="M11 12l9 0"/><path d="M11 18l9 0"/>`,
	"mail":        `<path d="M3 7a2 2 0 0 1 2 -2h14a2 2 0 0 1 2 2v10a2 2 0 0 1 -2 2h-14a2 2 0 0 1 -2 -2v-10z"/><path d="M3 7l9 6l9 -6"/>`,
	"puzzle":      `<path d="M4 7h3a1 1 0 0 0 1 -1v-1a2 2 0 0 1 4 0v1a1 1 0 0 0 1 1h3a1 1 0 0 1 1 1v3a1 1 0 0 0 1 1h1a2 2 0 0 1 0 4h-1a1 1 0 0 0 -1 1v3a1 1 0 0 1 -1 1h-3a1 1 0 0 1 -1 -1v-1a2 2 0 0 0 -4 0v1a1 1 0 0 1 -1 1h-3a1 1 0 0 1 -1 -1v-3a1 1 0 0 1 1 -1h1a2 2 0 0 0 0 -4h-1a1 1 0 0 1 -1 -1v-3a1 1 0 0 1 1 -1"/>`,
	"video":       `<path d="M15 10l4.553 -2.276a1 1 0 0 1 1.447 .894v6.764a1 1 0 0 1 -1.447 .894l-4.553 -2.276v-4z"/><path d="M3 6m0 2a2 2 0 0 1 2 -2h8a2 2 0 0 1 2 2v8a2 2 0 0 1 -2 2h-8a2 2 0 0 1 -2 -2z"/>`,
	"article":     `<path d="M3 4m0 2a2 2 0 0 1 2 -2h14a2 2 0 0 1 2 2v12a2 2 0 0 1 -2 2h-14a2 2 0 0 1 -2 -2z"/><path d="M7 8h10"/><path d="M7 12h10"/><path d="M7 16h10"/>`,
	"radar":       `<path d="M21 12h-8a1 1 0 1 0 -1 1v8a9 9 0 0 0 9 -9"/><path d="M16 9a5 5 0 1 0 -7 7"/><path d="M20.486 9a9 9 0 1 0 -11.482 11.495"/>`,
	"upload":      `<path d="M4 17v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2 -2v-2"/><path d="M7 9l5 -5l5 5"/><path d="M12 4l0 12"/>`,
	"calendar":    `<path d="M4 5m0 2a2 2 0 0 1 2 -2h12a2 2 0 0 1 2 2v12a2 2 0 0 1 -2 2h-12a2 2 0 0 1 -2 -2z"/><path d="M16 3l0 4"/><path d="M8 3l0 4"/><path d="M4 11l16 0"/><path d="M8 15h2v2h-2z"/>`,
	"adjust":      `<path d="M4 6l8 0"/><path d="M16 6l4 0"/><path d="M8 12l12 0"/><path d="M4 12l0 0"/><path d="M4 18l12 0"/><path d="M20 18l0 0"/><path d="M14 4m0 1a1 1 0 0 1 1 -1a1 1 0 0 1 1 1v2a1 1 0 0 1 -1 1a1 1 0 0 1 -1 -1z"/><path d="M6 10m0 1a1 1 0 0 1 1 -1a1 1 0 0 1 1 1v2a1 1 0 0 1 -1 1a1 1 0 0 1 -1 -1z"/><path d="M16 16m0 1a1 1 0 0 1 1 -1a1 1 0 0 1 1 1v2a1 1 0 0 1 -1 1a1 1 0 0 1 -1 -1z"/>`,
	"refresh":     `<path d="M20 11a8.1 8.1 0 0 0 -15.5 -2m-.5 -4v4h4"/><path d="M4 13a8.1 8.1 0 0 0 15.5 2m.5 4v-4h-4"/>`,
	"kanban":      `<path d="M4 4h6v8h-6z"/><path d="M4 16h6v4h-6z"/><path d="M14 4h6v4h-6z"/><path d="M14 12h6v8h-6z"/>`,
	"plug":        `<path d="M9.785 6l8.215 8.215l-2.054 2.054a5.81 5.81 0 1 1 -8.215 -8.215l2.054 -2.054z"/><path d="M4 20l3.5 -3.5"/><path d="M15 4l-3.5 3.5"/><path d="M20 9l-3.5 3.5"/>`,
	"shield":      `<path d="M11.46 20.846a12 12 0 0 1 -7.96 -14.846a12 12 0 0 0 8.5 -3a12 12 0 0 0 8.5 3a12 12 0 0 1 -.09 7.06"/><path d="M15 19l2 2l4 -4"/>`,
}

var uiTplFuncs = template.FuncMap{
	"shortid": shortID,
	// rawhtml renders an INTERNAL constant (feature group names) unescaped so
	// "+" survives; never call it with user/agent data.
	"rawhtml": func(s string) template.HTML { return template.HTML(s) },
	// pct: integer percent for progress bars, clamped 0..100
	"pct": func(a, b any) int {
		x, y := asFloat(a, 0), asFloat(b, 0)
		if y <= 0 {
			return 0
		}
		p := int(x / y * 100)
		if p < 0 {
			p = 0
		}
		if p > 100 {
			p = 100
		}
		return p
	},
	"icon": func(name string) template.HTML {
		p, ok := uiIcons[name]
		if !ok {
			return ""
		}
		return template.HTML(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">` + p + `</svg>`)
	},
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
{{define "head"}}<!doctype html><html lang="en" data-theme="dark"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{.Title}} · Solo Agency</title>
<link rel="stylesheet" href="/ui/assets/pico.min.css">
<link rel="stylesheet" href="/ui/assets/app.css">
<style>
/* Channel badge: the approvals queue and the distribution list both mix email,
   comment and post now, and those are not the same thing to read at a glance. */
.chanbadge{display:inline-block;padding:.16rem .6rem;border-radius:.4rem;font-size:.95rem;
 font-weight:700;letter-spacing:.02em;line-height:1.3;color:#fff;text-transform:uppercase;white-space:nowrap}
.chan-email{background:#2563eb}
.chan-comment{background:#0f9d58}
.chan-post{background:#c2410c}
.chan-messenger{background:#7c3aed}
</style>
</head><body>
<div class="shell">
<aside class="side">
<a class="brand" href="/ui">{{icon "bolt"}}<span>Solo Agency</span></a>
<nav class="snav">
<div class="ngroup">Agency</div>
<a href="/ui"{{if eq .NavPage "home"}} class="on"{{end}}>{{icon "home"}}Home</a>
<a href="/ui/jobs"{{if eq .NavPage "jobs"}} class="on"{{end}}>{{icon "activity"}}Jobs</a>
<a href="/ui/status"{{if eq .NavPage "status"}} class="on"{{end}}>{{icon "heart"}}Status</a>
<a href="/ui/fleet"{{if eq .NavPage "fleet"}} class="on"{{end}}>{{icon "radar"}}Fleet</a>
<a href="/ui/settings"{{if eq .NavPage "settings"}} class="on"{{end}}>{{icon "adjustments"}}Settings</a>
{{with .Client}}
<div class="ngroup">{{.Slug}}</div>
<a href="/ui/{{.Slug}}"{{if eq $.NavPage "client"}} class="on"{{end}}>{{icon "layout"}}Overview</a>
<a href="/ui/{{.Slug}}/campaigns"{{if eq $.NavPage "campaigns"}} class="on"{{end}}>{{icon "send"}}Campaigns</a>
<a href="/ui/{{.Slug}}/approvals"{{if eq $.NavPage "approvals"}} class="on"{{end}}>{{icon "checks"}}Approvals{{if $.NavPending}}<span class="nbadge">{{$.NavPending}}</span>{{end}}</a>
<a href="/ui/{{.Slug}}/sent"{{if eq $.NavPage "sent"}} class="on"{{end}}>{{icon "send"}}Distribution</a>
<a href="/ui/{{.Slug}}/crm"{{if eq $.NavPage "crm"}} class="on"{{end}}>{{icon "users"}}CRM</a>
<a href="/ui/{{.Slug}}/reports"{{if eq $.NavPage "reports"}} class="on"{{end}}>{{icon "file"}}Reports</a>
<a href="/ui/{{.Slug}}/shortlist"{{if eq $.NavPage "shortlist"}} class="on"{{end}}>{{icon "list"}}Shortlist</a>
<a href="/ui/{{.Slug}}/sendboxes"{{if eq $.NavPage "sendboxes"}} class="on"{{end}}>{{icon "mail"}}Sendboxes</a>
<a href="/ui/{{.Slug}}/extension"{{if eq $.NavPage "extension"}} class="on"{{end}}>{{icon "puzzle"}}Extension</a>
{{end}}
</nav>
</aside>
<div class="maincol">
<header class="topbar">
<div class="crumb">{{with .Client}}<a href="/ui/{{.Slug}}">{{.Slug}}</a><span class="sep">/</span>{{end}}{{if ne .NavPage "home"}}<span>{{.Title}}</span>{{end}}</div>
<span class="live" id="livedot"><i></i>live</span>
</header>
<main class="content"><h1>{{.Title}}</h1>{{end}}

{{define "foot"}}</main></div></div><script>
// Live pages reload themselves when the data behind them changes. That is fine until the
// operator is halfway through typing a goal or ticking groups — an auto-reload then throws
// their work away, silently. So: track whether anything on the page has unsaved input, and
// when it does, offer the refresh instead of performing it.
try{
 var dirty=false;
 document.addEventListener('input',function(e){
  var t=e.target;
  if(t&&(t.tagName==='INPUT'||t.tagName==='TEXTAREA'||t.tagName==='SELECT'))dirty=true;
 },true);
 document.addEventListener('change',function(e){
  var t=e.target;
  if(t&&(t.tagName==='INPUT'||t.tagName==='SELECT'||t.tagName==='TEXTAREA'))dirty=true;
 },true);
 document.addEventListener('submit',function(){dirty=false},true);
 function offerRefresh(){
  if(document.getElementById('stalebar'))return;
  var bar=document.createElement('button');
  bar.id='stalebar';bar.type='button';
  bar.textContent='New data arrived — refresh';
  bar.title='Refreshing now would discard what you have typed on this page.';
  bar.style.cssText='position:fixed;right:16px;bottom:16px;z-index:9999;width:auto;'+
   'padding:.5rem .9rem;border-radius:999px;box-shadow:0 2px 10px rgba(0,0,0,.25);cursor:pointer';
  bar.addEventListener('click',function(){dirty=false;location.reload()});
  document.body.appendChild(bar);
 }
 var es=new EventSource('/events');
 es.addEventListener('change',function(){ if(dirty){offerRefresh()} else {location.reload()} });
 es.onopen=function(){var l=document.getElementById('livedot');if(l)l.classList.add('on')};
}catch(e){}
</script></body></html>{{end}}

{{define "locked"}}{{template "head" .}}
<div class="lockwrap">
<span class="fico">{{icon "shield"}}</span>
<div class="card" style="text-align:left"><p><strong>UI Locked.</strong> Open the tokenized entry link once to unlock this browser.</p>
<p class="mut" style="margin-bottom:0">Ask your AI agent for the entry link, or read <code>daily-content-pipeline/bridge/ui_token</code> and open <code>/ui/enter/&lt;token&gt;</code>.</p></div>
</div>
{{template "foot" .}}{{end}}

{{define "home"}}{{template "head" .}}
<p class="sub">Your content and cold-outreach agency in one local binary: the agent does the work, you approve what leaves.</p>
{{with .Stats}}
<div class="stats">
<div class="stat"><b>{{.Clients}}</b><span>clients</span></div>
<div class="stat"><b>{{.Campaigns}}</b><span>active campaigns</span></div>
<div class="stat hot"><b>{{.Pending}}</b><span>drafts awaiting approval</span></div>
<div class="stat"><b>{{.SentToday}}</b><span>emails sent today</span></div>
</div>
{{end}}
<h2>Clients</h2><div class="grid-cards">
{{range .Clients}}<div class="card clientcard">
<div class="cname"><a href="/ui/{{.Slug}}">{{.Slug}}</a></div>
<div class="cpath">{{.Workspace}}</div>
<div class="clinks">
<a href="/ui/{{.Slug}}/campaigns">campaigns</a>
<a href="/ui/{{.Slug}}/approvals">approvals</a>
<a href="/ui/{{.Slug}}/crm">crm</a>
<a href="/ui/{{.Slug}}/reports">reports</a>
<a href="/ui/{{.Slug}}/sendboxes">sendboxes</a>
</div></div>
{{else}}<div class="empty"><b>No clients yet.</b><br>Tell the agent: <code>set up a new client</code> in the shared SETUP session.</div>{{end}}</div>

<h2>What this system can do</h2>
{{$feats := .Features}}
{{range $grp := groups $feats}}
<h3 style="font-size:.8rem;color:var(--tx2);text-transform:uppercase;letter-spacing:.08em;margin:18px 0 8px">{{rawhtml $grp}}</h3>
<div class="grid-cards">
{{range featIn $feats $grp}}
<div class="card feat">
<div class="fhead"><span class="fico">{{icon .Icon}}</span><strong>{{.Title}}</strong>
<span class="fkind">{{if eq .Kind "ui"}}<span class="pill band-high">web UI</span>{{else}}<span class="pill info">agent chat</span>{{end}}</span></div>
<p>{{.Value}}</p>
</div>
{{end}}
</div>
{{end}}
<h3 style="font-size:.8rem;color:var(--tx2);text-transform:uppercase;letter-spacing:.08em;margin:18px 0 8px">Runs itself</h3>
<div class="grid-cards">
<div class="card feat"><div class="fhead"><span class="fico">{{icon "refresh"}}</span><strong>Follow-up engine</strong></div>
<p>Blind-safe bump sequences with a shrinking ask; a reply freezes the thread instantly, everywhere.</p></div>
<div class="card feat"><div class="fhead"><span class="fico">{{icon "shield"}}</span><strong>Compliance built in</strong></div>
<p>Suppression on every send path, CAN-SPAM footer in code, no guessed addresses, approval required to send.</p></div>
<div class="card feat"><div class="fhead"><span class="fico">{{icon "plug"}}</span><strong>Survives reboots</strong></div>
<p>The bridge registers itself with the OS at setup and restarts after crashes on macOS, Linux and Windows.</p></div>
<div class="card feat"><div class="fhead"><span class="fico">{{icon "activity"}}</span><strong>Jobs and health</strong></div>
<p>Every collector run and automation is visible under <a href="/ui/jobs">Jobs</a> and <a href="/ui/status">Status</a>.</p></div>
</div>
<p class="mut" style="font-size:.82rem;margin-top:10px">Open a client above for the action cards: web-UI actions run right here, agent actions give you the exact phrase to paste into chat.</p>

<h2>Recent jobs</h2><div class="wrap"><table><tr><th>state</th><th>client</th><th>kind</th><th>file</th><th>when</th></tr>
{{range .Jobs}}<tr><td><span class="pill">{{.State}}</span></td><td>{{.Client}}</td><td>{{.Kind}}</td><td class="mut">{{.Name}}</td><td class="mut">{{.ModTime.Format "01-02 15:04"}}</td></tr>{{else}}<tr><td colspan="5" class="mut">none</td></tr>{{end}}</table></div>
{{template "foot" .}}{{end}}

{{define "jobs"}}{{template "head" .}}
<p class="sub">Collector jobs and automation runs, live from the file bus.</p>
<h2>Active runs</h2><div class="card"><pre class="mut" style="margin:0;white-space:pre-wrap">{{range .Active}}{{printf "%v" .}}
{{else}}none{{end}}</pre></div>
<h2>Queue</h2><div class="wrap"><table><tr><th>state</th><th>client</th><th>kind</th><th>run id</th><th>file</th><th>when</th></tr>
{{range .Jobs}}<tr><td><span class="pill">{{.State}}</span></td><td>{{.Client}}</td><td>{{.Kind}}</td><td class="mut">{{.RunID}}</td><td class="mut">{{.Name}}</td><td class="mut">{{.ModTime.Format "01-02 15:04"}}</td></tr>{{else}}<tr><td colspan="6" class="mut">empty</td></tr>{{end}}</table></div>
{{template "foot" .}}{{end}}

{{define "status"}}{{template "head" .}}
<p class="sub">Bridge, extension, sendbox and provider health at a glance.</p>
<div class="card"><span class="dot ok"></span> Bridge started <strong>{{.StartedAt}}</strong> (persistent: {{.Persistent}})<br>
<span class="mut" style="font-family:var(--mono);font-size:.75rem">data root: {{.DataRoot}}</span></div>
<h2>Extensions</h2><div class="wrap"><table><tr><th>client</th><th>instance</th><th>name</th><th>last check-in</th></tr>
{{range .Extensions}}<tr><td>{{.Client}}</td><td class="mut">{{.Instance}}</td><td>{{.Name}}</td><td class="mut">{{.Last}}</td></tr>{{else}}<tr><td colspan="4" class="mut">no extension check-ins yet</td></tr>{{end}}</table></div>
<h2>Sendboxes</h2><div class="wrap"><table><tr><th>client</th><th>slug</th><th>email</th><th>status</th><th>quota</th><th>warmup</th></tr>
{{range .Sendboxes}}<tr><td>{{.Client}}</td><td>{{.Slug}}</td><td>{{.Email}}</td><td><span class="pill">{{.Status}}</span></td><td>{{.Quota}}</td><td class="mut">{{.Warmup}}</td></tr>{{else}}<tr><td colspan="6" class="mut">none configured</td></tr>{{end}}</table></div>
{{template "foot" .}}{{end}}

{{define "fleet"}}{{template "head" .}}
<p class="sub">Every client on one screen: last post, performance, and who needs attention — worst first. Data is written by each run as it finishes; this page never queries anything.</p>
<div class="grid" style="grid-template-columns:repeat(auto-fit,minmax(130px,1fr));gap:10px;margin-bottom:14px">
<div class="card" style="text-align:center"><div style="font-size:1.6rem;font-weight:700">{{.Tiles.total}}</div><div class="mut">clients</div></div>
<div class="card" style="text-align:center"><div style="font-size:1.6rem;font-weight:700;color:#4ade80">{{.Tiles.green}}</div><div class="mut">posting on time</div></div>
<div class="card" style="text-align:center"><div style="font-size:1.6rem;font-weight:700;color:#eab308">{{.Tiles.yellow}}</div><div class="mut">approaching {{.GapHours}}h</div></div>
<div class="card" style="text-align:center"><div style="font-size:1.6rem;font-weight:700;color:#ef4444">{{.Tiles.red}}</div><div class="mut">red alerts</div></div>
<div class="card" style="text-align:center"><div style="font-size:1.6rem;font-weight:700">{{.SumPosts}}</div><div class="mut">posts, 7 days</div></div>
<div class="card" style="text-align:center"><div style="font-size:1.6rem;font-weight:700">{{.SumViews}}</div><div class="mut">views, 7 days</div></div>
<div class="card" style="text-align:center"><div style="font-size:1.6rem;font-weight:700">{{.SumDrafts}}</div><div class="mut">drafts pending</div></div>
</div>
<div class="wrap"><table>
<tr><th></th><th>client</th><th>last post</th><th>posts 7d/30d</th><th>views 7d</th><th>likes 7d</th><th>leads hot/warm</th><th>ideas queued</th><th>drafts pending</th><th>report</th><th>updated</th></tr>
{{range .Rows}}<tr>
<td><span class="dot{{if ne .Color "none"}} {{.Color}}{{end}}"></span></td>
<td><a href="/ui/{{.Client}}">{{.Name}}</a></td>
<td>{{.LastPost}}{{if .Reminders}}<br><span class="mut" style="font-size:.75rem">{{.Reminders}}</span>{{end}}</td>
<td>{{.Posts}}</td><td>{{.Views7}}</td><td>{{.Likes7}}</td><td>{{.Leads}}</td><td>{{.IdeasQ}}</td><td>{{.DraftsPend}}</td>
<td>{{if .ReportHref}}<a href="{{.ReportHref}}" target="_blank" rel="noopener">open ↗</a>{{else}}<span class="mut">–</span>{{end}}</td>
<td class="mut">{{.Updated}}{{if .Stale}} <span class="pill band-review_carefully">stale</span>{{end}}</td>
</tr>{{else}}<tr><td colspan="11" class="mut">no snapshots yet — each client appears here after its next report run (fleet/{client}.json)</td></tr>{{end}}
</table></div>
{{template "foot" .}}{{end}}

{{define "harvest"}}{{template "head" .}}
<p class="sub"><a href="/ui/{{.Client.Slug}}/campaign/{{.Slug}}">← {{.Slug}}</a> · <strong>{{.StageLabel}}</strong> — {{.StageHelp}}. {{.Count}} profile(s).</p>
<div class="toolbar" style="flex-wrap:wrap;gap:.4rem">
{{$cur := .Stage}}{{$cl := .Client.Slug}}{{$sl := .Slug}}{{range .Stages}}<a class="pill{{if eq .Key $cur}} band-high{{end}}" href="/ui/{{$cl}}/campaign/{{$sl}}/harvest?stage={{.Key}}">{{.Label}}</a> {{end}}
</div>
{{if .Goal}}<div class="card mut" style="font-size:.85rem">Goal the agent judges against: <em>{{.Goal}}</em></div>{{end}}
{{range .Rows}}
<div class="card">
<div style="display:flex;align-items:baseline;gap:.5rem;flex-wrap:wrap">
<strong>{{if .Name}}{{.Name}}{{else}}<span class="mut">(name not read yet)</span>{{end}}</strong>
<span class="pill">{{.Status}}</span>
{{if .OK}}<span class="pill{{if eq .OK "failed"}} band-review_carefully{{end}}">{{if eq .OK "ok"}}enriched ok{{else}}enrich failed{{end}}</span>{{end}}
{{if .Priority}}<span class="pill">{{.Priority}}</span>{{end}}
{{if .Attempts}}<span class="pill">attempt {{.Attempts}}</span>{{end}}
{{if .LeadID}}<a class="pill band-high" href="/ui/{{$.Client.Slug}}/contact/{{.LeadID}}">CRM {{.LeadID}} ↗</a>{{end}}
<span class="mut" style="font-size:.8rem;margin-left:auto">{{.When}}{{if .Box}} · {{.Box}}{{end}}</span>
</div>
<div class="mut" style="font-size:.85rem;margin-top:4px"><a href="{{.URL}}" target="_blank" rel="noopener">{{.URL}}</a>{{if .Subtitle}} · {{.Subtitle}}{{end}}{{if .Category}} · {{.Category}}{{end}}</div>
{{if .Seed}}<div class="mut" style="font-size:.78rem">via seed {{.Seed}}</div>{{end}}
{{if .Reason}}<div style="margin-top:6px"><span class="mut">Verdict reason:</span> {{.Reason}}</div>{{end}}
{{if .Error}}<div style="margin-top:6px" class="mut">Error: {{.Error}}</div>{{end}}
{{if .AvoidBox}}<div class="mut" style="font-size:.78rem">retry avoids {{.AvoidBox}}</div>{{end}}
{{if or .About .Work .Emails .Websites .Posts}}
<details style="margin-top:8px"><summary class="mut" style="cursor:pointer;font-size:.85rem">Collected data</summary>
{{if .Work}}<div style="margin-top:6px"><span class="mut">Work:</span> {{range .Work}}<div>· {{.}}</div>{{end}}</div>{{end}}
{{if .About}}<div style="margin-top:6px"><span class="mut">About:</span> {{range .About}}<div>· {{.}}</div>{{end}}</div>{{end}}
{{if .Emails}}<div style="margin-top:6px"><span class="mut">Emails:</span> {{range .Emails}}<code>{{.}}</code> {{end}}</div>{{end}}
{{if .Websites}}<div style="margin-top:6px"><span class="mut">Websites:</span> {{range .Websites}}<a href="{{.}}" target="_blank" rel="noopener">{{.}}</a> {{end}}</div>{{end}}
{{if .Posts}}<div style="margin-top:6px"><span class="mut">Recent posts / reels:</span>{{range .Posts}}<div style="margin:4px 0 0 8px">{{if .date}}<span class="mut">{{.date}}</span> {{end}}{{.caption}}{{if .url}} <a href="{{.url}}" target="_blank" rel="noopener">↗</a>{{end}}</div>{{end}}</div>{{end}}
</details>
{{end}}
</div>
{{else}}<div class="card mut">Nothing at this stage right now.</div>{{end}}
{{template "foot" .}}{{end}}

{{define "settings"}}{{template "head" .}}
<p class="sub">Global system configuration for this Solo Agency install. Changes apply from the next run — no agent chat needed.</p>
<div class="card">
<h2 style="margin-top:0">Operator</h2>
<label>Operator email
<input type="email" id="s-email" value="{{.Settings.OperatorEmail}}" placeholder="you@example.com">
<small class="mut">Where agents send operator notifications (blockers, action-required) when an operator channel is used. Client notifications are separate and go to each client's own channel.</small></label>
<h2>Task scheduling</h2>
<label>Max tasks running in the same time slot
<input type="number" id="s-max" value="{{.Settings.MaxConcurrentTasks}}" min="1" max="500">
<small class="mut">When a new automation task would land in a slot already holding this many tasks (projected across cadences: daily/48h/72h/weekly/monthly), its start time is pushed later in steps until a free slot is found.</small></label>
<div class="grid">
<label>Push step (minutes)
<input type="number" id="s-step" value="{{.Settings.SlotStepMinutes}}" min="1" max="240"></label>
<label>Projection horizon (days)
<input type="number" id="s-horizon" value="{{.Settings.SlotHorizonDays}}" min="7" max="366"></label>
<label>Default task duration (minutes)
<input type="number" id="s-duration" value="{{.Settings.DefaultTaskDurationMin}}" min="5" max="480"></label>
</div>
<h2>Client accountability</h2>
<label>Posting-gap alert threshold (hours)
<input type="number" id="s-gap" value="{{.Settings.AccountabilityMaxGapHours}}" min="1" max="720">
<small class="mut">A client who has not posted a video for longer than this gets a reminder (their own channel, separate from reports). Default 72h — a weekend gap is normal. After 3 unheeded reminders the operator gets an email at the address above.</small></label>
<h2>Distribution caps (per Facebook account, per day)</h2>
<p class="mut" style="font-size:.83rem;margin-top:0">Agency-wide ceilings for Messenger and comment outreach. The scarce resource is <strong>account health</strong>: a bounced email costs one lead, but an account that looks like a bulk sender gets restricted and takes its warmup history with it. Start low and raise only on evidence. A campaign may ask for less than these; never more.</p>
<div class="grid">
<label>Direct messages / account / day
<input type="number" id="s-dm" value="{{.Settings.DMPerAccountPerDay}}" min="1" max="50"></label>
<label>Groups commented in / account / day
<input type="number" id="s-cgroups" value="{{.Settings.CommentGroupsPerAccountPerDay}}" min="1" max="20"></label>
<label>Comments / group / day
<input type="number" id="s-cper" value="{{.Settings.CommentsPerGroupPerDay}}" min="1" max="10"></label>
<label>New posts into groups / account / day
<input type="number" id="s-posts" value="{{.Settings.PostsPerAccountPerDay}}" min="1" max="10"></label>
<label>Minutes between two published actions
<input type="number" id="s-pubgap" value="{{.Settings.PublishGapMinutes}}" min="1" max="240"></label>
</div>
<small class="mut">Groups × comments-per-group is the daily comment ceiling for one account. The group count is a <em>diversity</em> cap — spreading across a few groups reads as human, hammering one does not. Approving several drafts at once does not publish them at once: they are spaced by the gap below. <strong>Posting</strong> is the most exposed of the three: a standalone piece of content in front of the whole group, the thing members report as spam, and often held for admin approval — keep it lowest, and never push the same post into many groups on the same day.</small>
<h2>Leads From Friends (harvest pacing)</h2>
<div class="grid">
<label>Profiles enriched per day, per campaign
<input type="number" id="s-hd" value="{{.Settings.HarvestDailyBudget}}" min="1" max="5000"></label>
<label>Per collector account, per day
<input type="number" id="s-hb" value="{{.Settings.HarvestPerCollectorBudget}}" min="1" max="2000"></label>
<label>Quiet hours from (HH:MM)
<input type="text" id="s-hqf" value="{{.Settings.HarvestQuietFrom}}" placeholder="01:00"></label>
<label>Quiet hours to (HH:MM)
<input type="text" id="s-hqt" value="{{.Settings.HarvestQuietTo}}" placeholder="06:00"></label>
</div>
<small class="mut">The harvest daemon spreads enrichment across the day (20–40s between profiles), rotates collector accounts every leg, and never exceeds these caps. With N collectors the effective daily ceiling is min(daily, N × per-collector) — add extensions to raise it. Nothing runs inside quiet hours.</small>
<button class="ok" id="s-save" style="margin-top:12px">Save settings</button>
<span class="mut" id="s-msg" style="font-size:.83rem;margin-left:8px"></span>
</div>
<h2>Registered automation tasks</h2>
<div class="wrap"><table><tr><th>task</th><th>client</th><th>cadence</th><th>time</th><th>anchor date</th><th>duration</th><th>status</th></tr>
{{range .Tasks}}<tr><td>{{.Task}}</td><td>{{.Client}}</td><td><span class="pill">{{.Cadence}}</span></td><td>{{.Time}}</td><td class="mut">{{.Anchor}}</td><td class="mut">{{.Duration}}</td><td><span class="pill">{{.Status}}</span></td></tr>{{else}}<tr><td colspan="7" class="mut">no tasks registered yet — tasks appear here as agents create them (tool schedule-slots register)</td></tr>{{end}}</table></div>
<script>
document.getElementById('s-save').onclick=function(){
 var m=document.getElementById('s-msg');m.textContent='saving...';
 fetch('/api/ui/system/settings',{method:'POST',headers:{'Content-Type':'application/json'},
  body:JSON.stringify({operator_email:document.getElementById('s-email').value,
   max_concurrent_tasks:+document.getElementById('s-max').value,
   slot_step_minutes:+document.getElementById('s-step').value,
   slot_horizon_days:+document.getElementById('s-horizon').value,
   default_task_duration_min:+document.getElementById('s-duration').value,
   accountability_max_gap_hours:+document.getElementById('s-gap').value,
   dm_per_account_per_day:+document.getElementById('s-dm').value,
   comment_groups_per_account_per_day:+document.getElementById('s-cgroups').value,
   comments_per_group_per_day:+document.getElementById('s-cper').value,
   posts_per_account_per_day:+document.getElementById('s-posts').value,
   publish_gap_minutes:+document.getElementById('s-pubgap').value,
   harvest_daily_budget:+document.getElementById('s-hd').value,
   harvest_per_collector_budget:+document.getElementById('s-hb').value,
   harvest_quiet_from:document.getElementById('s-hqf').value,
   harvest_quiet_to:document.getElementById('s-hqt').value})})
 .then(function(r){return r.ok?r.json():r.text().then(function(t){throw new Error(t)})})
 .then(function(){m.textContent='saved ✓'})
 .catch(function(e){m.textContent='error: '+e.message});
};
</script>
{{template "foot" .}}{{end}}

{{define "client"}}{{template "head" .}}
<p class="sub">Everything for this client in one place. Web-UI cards run right here; agent cards give you the exact phrase to paste into chat.</p>
{{with .CStats}}<div class="statrow">
<a class="stat hot" style="text-decoration:none" href="/ui/{{$.Client.Slug}}/approvals"><b>{{.Pending}}</b><span>drafts awaiting approval</span></a>
<a class="stat" style="text-decoration:none" href="/ui/{{$.Client.Slug}}/campaigns"><b>{{.Campaigns}}</b><span>active campaigns</span></a>
<a class="stat" style="text-decoration:none" href="/ui/{{$.Client.Slug}}/crm"><b>{{.Contacts}}</b><span>contacts in CRM</span></a>
</div>{{end}}
<h2>Actions</h2>
<p class="mut" style="margin-top:-.3rem;font-size:.84rem">Cards marked <span class="pill band-high">web UI</span> run right here. Cards marked <span class="pill info">agent chat</span> start by pasting the phrase into the named chat session.</p>
{{$slug := .Client.Slug}}
{{range $grp := groups .Features}}
<h3 style="font-size:.95rem;margin:.9rem 0 .3rem">{{rawhtml $grp}}</h3>
<div class="grid-cards">
{{range featIn $.Features $grp}}
<div class="card feat">
<div class="fhead"><span class="fico">{{icon .Icon}}</span><strong>{{.Title}}</strong>
<span class="fkind">{{if eq .Kind "ui"}}<span class="pill band-high">web UI</span>{{else}}<span class="pill info">agent chat</span>{{end}}</span></div>
<p>{{.Value}}</p>
{{if eq .Kind "ui"}}
<div><a role="button" class="ok" style="display:inline-block;padding:.3rem .9rem;font-size:.8rem" href="/ui/{{$slug}}/{{.Href}}">Open</a></div>
{{else}}
<div><code style="font-size:.76rem">{{.Phrase}}</code>
<button class="copy-phrase" data-phrase="{{.Phrase}}" style="padding:.15rem .6rem;font-size:.72rem;margin:0">Copy</button></div>
<span class="mut" style="font-size:.72rem">paste into {{.Session}}</span>
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
{{range .Latest}}<tr><td><a href="/files/{{.Rel}}">{{.Name}}</a></td><td class="mut">{{.ModTime.Format "2006-01-02 15:04"}}</td></tr>{{else}}<tr><td colspan="2" class="mut">no outputs yet: run the client's daily task</td></tr>{{end}}</table></div>
{{template "foot" .}}{{end}}

{{define "reports"}}{{template "head" .}}
<p class="sub">Daily and weekly HTML reports; the newest run sits on top.</p>
<div class="wrap"><table><tr><th>file</th><th>when</th><th>size</th></tr>
{{range .Files}}<tr><td><a href="/files/{{.Rel}}">{{.Rel}}</a></td><td class="mut">{{.ModTime.Format "2006-01-02 15:04"}}</td><td class="mut">{{.Size}}</td></tr>{{else}}<tr><td colspan="3" class="mut">no reports yet</td></tr>{{end}}</table></div>
{{template "foot" .}}{{end}}

{{define "crm"}}{{template "head" .}}
{{if .StageOrder}}<h2>Pipeline</h2><div class="grid-cards">
{{$st := .Stages}}{{range .StageOrder}}<div class="card"><strong>{{.}}</strong>
{{range index $st .}}<div class="mut">{{if .Title}}{{.Title}}{{else}}{{.ID}}{{end}}</div>{{else}}<div class="mut" style="opacity:.5">empty</div>{{end}}</div>{{end}}</div>
{{else}}<div class="empty" style="margin-top:0"><b>No deals yet.</b><br>Replies become deals here automatically; approve some drafts and let the campaign run.</div>{{end}}
<h2>Lead growth <span class="mut" style="font-size:.8rem">how the list is growing, not just how big it is</span></h2>
<div class="card">
<div class="statrow" style="margin-bottom:.9rem">
<div class="stat"><b>{{.Growth.Total}}</b><span>leads total</span></div>
<div class="stat{{if .Growth.ThisWeek}} hot{{end}}"><b>{{.Growth.ThisWeek}}</b><span>added this week</span></div>
<div class="stat"><b>{{.Growth.LastWeek}}</b><span>last week</span></div>
<div class="stat"><b>{{if gt .Growth.Delta 0}}+{{end}}{{.Growth.Delta}}</b><span>week over week</span></div>
{{if ge .Growth.DaysSinceLast 0}}<div class="stat"><b>{{.Growth.DaysSinceLast}}d</b><span>since the last new lead</span></div>{{end}}
</div>
{{if .Growth.Max}}
<div style="display:flex;align-items:flex-end;gap:4px;height:90px;padding-top:4px">
{{range .Growth.Weeks}}<div style="flex:1;display:flex;flex-direction:column;justify-content:flex-end;align-items:center;height:100%" title="week of {{.Label}}: {{.Count}} new">
<span class="mut" style="font-size:.7rem;line-height:1">{{if .Count}}{{.Count}}{{end}}</span>
<div style="width:100%;background:var(--accent,#2a6);opacity:{{if .Count}}.85{{else}}.12{{end}};height:{{if .Count}}{{.Pct}}%{{else}}2px{{end}};border-radius:2px 2px 0 0;min-height:2px"></div>
</div>{{end}}
</div>
<div style="display:flex;gap:4px;margin-top:3px">
{{range .Growth.Weeks}}<span class="mut" style="flex:1;text-align:center;font-size:.62rem;white-space:nowrap;overflow:hidden">{{.Label}}</span>{{end}}
</div>
<p class="mut" style="font-size:.78rem;margin:.7rem 0 0">New contacts per week, last 12 weeks. Leads arrive in bursts — an import, or a harvest from a comment run — so an empty week is normal; a long empty stretch is the signal.{{if ge .Growth.DaysSinceLast 7}} <strong>Nothing new since {{.Growth.LastAdded}}</strong> — the pipeline is not feeding itself right now.{{end}}</p>
{{else}}
<p class="mut" style="margin:0">No contacts carry a creation date yet, so growth cannot be charted. Totals above still hold.</p>
{{end}}
</div>

<h2>Contacts <span class="mut" style="font-size:.8rem">click a row for the full profile and its latest activities</span></h2>
<div class="statrow">
<div class="stat"><b>{{.Stats.Total}}</b><span>leads total</span></div>
<div class="stat"><b>{{.Stats.WithEmail}}</b><span>have an email</span></div>
<div class="stat"><b>{{.Stats.Enriched}}</b><span>enriched</span></div>
<div class="stat"><b>{{.Stats.Both}}</b><span>email + enriched <span class="mut">(writable)</span></span></div>
{{if .Stats.SeedUnresolved}}<div class="stat"><b>{{.Stats.SeedUnresolved}}</b><span>origin unresolved</span></div>{{end}}
</div>
<div class="wrap"><table><tr><th>#</th><th>name</th><th>email</th><th>phone</th><th>social</th><th>vertical</th><th>state</th></tr>
{{$slug := .Client.Slug}}{{range .Contacts}}<tr style="cursor:pointer" onclick="location.href='/ui/{{$slug}}/contact/{{.ID}}'">
<td class="mut" style="text-align:right;font-variant-numeric:tabular-nums">{{.N}}</td>
<td>{{if .Name}}<strong>{{.Name}}</strong>{{else}}<span class="mut" title="{{.ID}}">{{.ShortID}}</span>{{end}}</td>
<td class="mut">{{if .Email}}{{.Email}}{{else}}·{{end}}</td>
<td class="mut">{{if .Phone}}{{.Phone}}{{else}}·{{end}}</td>
<td class="mut">{{if .Social}}<a href="{{.Social}}" target="_blank" rel="noopener" onclick="event.stopPropagation()">link ↗</a>{{else}}·{{end}}</td>
<td>{{.Vertical}}</td>
<td>{{if .Band}}<span class="pill band-high">enriched</span>{{else if .SeedsUnresolved}}<span class="pill band-review_carefully">seed: trace origin</span>{{else}}<span class="pill">{{if .Stage}}{{.Stage}}{{else}}new{{end}}</span>{{end}}</td>
</tr>{{else}}<tr><td colspan="7" class="mut">no contacts yet: import a list or run discovery</td></tr>{{end}}</table></div>
{{if gt .Pages 1}}<nav class="pager">
<span class="mut">{{.From}}&ndash;{{.To}} of {{.Stats.Total}}</span>
{{if gt .Page 1}}<a href="?page={{.PrevPage}}">&larr; prev</a>{{else}}<span class="mut" style="opacity:.4">&larr; prev</span>{{end}}
{{$cur := .Page}}{{$slug2 := .Client.Slug}}{{range .PageNums}}{{if eq . $cur}}<strong class="here">{{.}}</strong>{{else}}<a href="?page={{.}}">{{.}}</a>{{end}}{{end}}
{{if lt .Page .Pages}}<a href="?page={{.NextPage}}">next &rarr;</a>{{else}}<span class="mut" style="opacity:.4">next &rarr;</span>{{end}}
</nav>{{end}}
{{template "foot" .}}{{end}}

{{define "contact"}}{{template "head" .}}
{{$c := .C}}
{{if $c.DuplicateSuspects}}
<div class="card warnb">
<strong>Possible duplicate</strong> <span class="mut">shares an identity with:</span>
{{range $c.DuplicateSuspects}}<div style="margin-top:.25rem"><a href="/ui/{{$.Client.Slug}}/contact/{{.id}}"><code style="font-size:.8rem">{{.id}}</code></a> <span class="mut" style="font-size:.82rem">(shared {{.via}}: {{.value}})</span></div>{{end}}
<p class="mut" style="font-size:.78rem;margin-bottom:0">Same person: tell the agent <code>merge these contacts</code>. Different people: <code>clear the duplicate flag</code>. Suspected duplicates are held out of campaign queues until resolved.</p>
</div>
{{end}}
<div class="card">
<div style="display:flex;justify-content:space-between;flex-wrap:wrap;gap:.5rem;align-items:baseline">
<div><strong style="font-size:1.15rem">{{if $c.Name}}{{$c.Name}}{{else}}<span class="mut">{{shortid $c.ID}}</span>{{end}}</strong>
{{if $c.Company}} <span class="mut">· {{$c.Company}}{{if $c.Role}}, {{$c.Role}}{{end}}</span>{{end}}</div>
<div>
{{if $c.Band}}<span class="pill band-high">enriched: {{$c.Band}}</span>{{else}}<span class="pill band-review_carefully">not enriched yet</span>{{end}}
{{if $c.StillActive}}<span class="pill">{{$c.StillActive}}</span>{{end}}
{{if eq $c.SequenceState "frozen"}}<span class="pill">sequence frozen (replied)</span>{{end}}
</div></div>
{{if $c.OneLiner}}<p class="mut" style="margin:.4rem 0 0">{{$c.OneLiner}}</p>{{end}}
</div>

<div class="grid-cards">
<div class="card"><strong>Identities</strong>
<table style="margin-top:.4rem"><tbody>
{{range $c.Emails}}<tr><td class="mut">email</td><td>{{.address}}{{if .status}} <span class="mut">({{.status}})</span>{{end}}</td></tr>{{end}}
{{range $c.Phones}}<tr><td class="mut">phone</td><td>{{.number}}</td></tr>{{end}}
{{range $c.Socials}}<tr><td class="mut">{{index . 0}}</td><td><a href="{{index . 1}}" target="_blank" rel="noopener">{{index . 1}}</a></td></tr>{{end}}
{{if $c.Website}}<tr><td class="mut">website</td><td><a href="{{$c.Website}}" target="_blank" rel="noopener">{{$c.Website}}</a></td></tr>{{end}}
{{if and (not $c.Emails) (not $c.Phones) (not $c.Socials) (not $c.Website)}}<tr><td colspan="2" class="mut">no reachable identity yet: enrichment must resolve one</td></tr>{{end}}
</tbody></table></div>

{{if $c.Seeds}}<div class="card"><strong>Content clues (seeds)</strong>
<span class="mut" style="font-size:.8rem">traced back to a profile during enrichment</span>
<table style="margin-top:.4rem"><tbody>
{{range $c.Seeds}}<tr><td class="mut">{{.kind}}{{if .platform}} · {{.platform}}{{end}}</td>
<td><a href="{{.url}}" target="_blank" rel="noopener">{{.url}}</a>
{{if eq (printf "%v" .status) "resolved"}}<span class="pill band-high">resolved</span>{{else}}<span class="pill band-review_carefully">unresolved</span>{{end}}</td></tr>{{end}}
</tbody></table></div>{{end}}
</div>

<h2>Latest activities <span class="mut" style="font-size:.8rem">the proof-of-life hooks used to personalize email</span></h2>
{{if $c.Hooks}}
<div class="wrap"><table><tr><th>signal</th><th>what</th><th>observed</th><th>evidence</th><th>used</th></tr>
{{range $c.Hooks}}<tr>
<td><span class="pill">{{.Type}}</span></td>
<td>{{.Summary}}</td>
<td class="mut">{{if .Observed}}{{.Observed}}{{else}}<span title="recency unverified">?</span>{{end}}</td>
<td>{{if .URL}}<a href="{{.URL}}" target="_blank" rel="noopener">source ↗</a>{{else}}<span class="mut">·</span>{{end}}</td>
<td class="mut">{{range .UsedIn}}{{.}} {{else}}·{{end}}</td>
</tr>{{end}}</table></div>
{{if $c.HooksRefreshed}}<p class="mut" style="font-size:.8rem">hooks refreshed {{$c.HooksRefreshed}}</p>{{end}}
{{else}}
<p class="mut">No hooks yet. These are the recent, evidenced signals (a new listing, a post, a review, an award) that make each email genuinely personal; enrichment fills them in. Run the client's daily task, or tell the agent "enrich my leads".</p>
{{end}}

{{if $c.DoNotMention}}<p class="mut" style="font-size:.8rem">Do not mention: {{range $c.DoNotMention}}{{.}}; {{end}}</p>{{end}}

<h2>Activity timeline</h2>
{{if $c.Activities}}
<div class="wrap"><table><tr><th>when</th><th>event</th><th>detail</th><th>by</th></tr>
{{range $c.Activities}}<tr><td class="mut">{{.At}}</td><td><span class="pill">{{.Type}}</span></td><td>{{.Summary}}</td><td class="mut">{{.By}}</td></tr>{{end}}</table></div>
{{else}}<p class="mut">No activity recorded yet (sends, replies, stage changes appear here).</p>{{end}}
{{template "foot" .}}{{end}}

{{define "footform"}}
<div class="card mut" style="margin-top:16px">Decisions are queued in <code>ui_inbox/</code>; the agent applies them automatically at the start of the next campaign run, or tell it: <em>"apply my UI decisions"</em>.</div>
</main></div></div></body></html>{{end}}

{{define "approvals"}}{{template "head" .}}
<p class="sub"><span id="left">{{len .Drafts}}</span> drafts waiting. Nothing sends without your approval; edits made here are kept.</p>
{{if .Drafts}}
<div class="toolbar">
<label><input type="checkbox" id="checkall" checked> All</label>
<button class="ok" id="approvechecked">Approve checked (<span id="ckcount">0</span>)</button>
<a href="#" id="onlyhigh" class="mut" style="font-size:.83rem">select high-confidence only</a>
<label id="campwrap" class="mut" style="font-size:.83rem;margin:0">campaign <select id="campfilter" style="width:auto;display:inline-block;margin:0 0 0 .3rem;padding:.1rem .3rem;font-size:.83rem"><option value="">all</option></select></label>
<span class="mut" id="batchmsg" style="font-size:.83rem"></span>
</div>
{{end}}
{{range .Drafts}}
<div class="card draft" data-id="{{.ID}}" data-campaign="{{.Campaign}}" data-band="{{.Band}}">
<div style="display:flex;align-items:baseline;gap:.5rem;flex-wrap:wrap">
<label style="margin:0;cursor:pointer"><input class="pick" type="checkbox" checked style="margin:0"></label>
<span class="chanbadge chan-{{.Channel}}">{{if eq .Channel "comment"}}Comment{{else if eq .Channel "post"}}Post{{else if eq .Channel "messenger"}}DM{{else}}Email{{end}}</span>
<strong>{{.To}}</strong> <span class="pill band-{{.Band}}">{{.Band}}</span>
<span class="pill">{{.Campaign}}</span>{{if eq .Channel "email"}} <span class="pill">step {{.Step}}</span>{{end}}
{{if .Companion}}<a class="pill" href="{{.Companion}}" target="_blank" rel="noopener">companion ↗</a>{{end}}
</div>
{{if .Warnings}}<div style="margin-top:6px">{{range .Warnings}}<span class="pill band-review_carefully">⚠ {{.}}</span> {{end}}</div>{{end}}
{{if .Hooks}}<div class="mut" style="margin-top:6px;font-size:13px">hooks: {{range .Hooks}}{{if index . "evidence_url"}}<a href="{{index . "evidence_url"}}" target="_blank" rel="noopener">{{index . "type"}}</a> {{else}}{{index . "type"}} {{end}}{{end}}</div>{{end}}
<div style="margin-top:8px"{{if ne .Channel "email"}} hidden{{end}}><input class="subj" type="text" value="{{.Subject}}"{{if ne .Channel "email"}} disabled{{end}}></div>
<div style="margin-top:8px"><textarea class="body">{{.Body}}</textarea></div>
<div class="acts">
<button class="ok" data-act="approve">Approve</button>
<button data-act="edit">Save edit (keep pending)</button>
<button data-act="hold">Hold</button>
<button data-act="reject">Reject…</button>
</div></div>
{{else}}<p class="mut">No drafts waiting for approval. New drafts appear here after the campaign's daily run. Approving a <b>Comment</b> or <b>Post</b> publishes it straight away; an <b>Email</b> goes out on the next run.</p>{{end}}
<script>
var CLIENT="{{.Client.Slug}}";
function payload(card){var p={draft_id:card.dataset.id,campaign:card.dataset.campaign};
 var s=card.querySelector('.subj'),b=card.querySelector('.body');
 if(s.value!==s.defaultValue)p.edited_subject=s.value;
 if(b.value!==b.defaultValue)p.edited_body=b.value;return p}
function allCards(){return Array.prototype.slice.call(document.querySelectorAll('.draft'))}
// Hidden cards are NOT pickable. Everything that acts in bulk — the All checkbox,
// "select high-confidence only", "Approve checked" — runs off this list, so a filtered
// view can never approve a draft belonging to a campaign the operator cannot see.
function pickable(){return allCards().filter(function(c){return !c.classList.contains('done')&&c.style.display!=='none'})}
function checkedCards(){return pickable().filter(function(c){var p=c.querySelector('.pick');return p&&p.checked})}
function updateCount(){var all=document.getElementById('checkall');if(!all)return;
 var cards=pickable(),n=checkedCards().length;
 document.getElementById('ckcount').textContent=n;
 all.checked=n>0&&n===cards.length;all.indeterminate=n>0&&n<cards.length;
 document.getElementById('approvechecked').disabled=(n===0);
 var left=document.getElementById('left');
 if(left){var total=allCards().filter(function(c){return !c.classList.contains('done')}).length;
  // Say "3 of 50" while filtered — never let the headline imply the hidden ones are gone.
  left.textContent=(cards.length===total)?String(total):(cards.length+' of '+total)}}
function send(card,act,note){var p=payload(card);p.decision=act;if(note)p.note=note;
 return fetch('/api/ui/'+CLIENT+'/approval',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(p)})
 .then(function(r){if(!r.ok)throw new Error('HTTP '+r.status);return r.json()})
 .then(function(){if(act!=='edit'){card.classList.add('done');card.querySelector('.acts').innerHTML='<span class="pill">'+act+' ✓ queued</span>'}
  else{card.querySelector('.subj').defaultValue=card.querySelector('.subj').value;card.querySelector('.body').defaultValue=card.querySelector('.body').value}
  updateCount()})
 .catch(function(e){alert('Failed: '+e.message)})}
document.addEventListener('click',function(e){var b=e.target.closest('button[data-act]');if(!b)return;
 var card=b.closest('.draft');var act=b.getAttribute('data-act');
 if(act==='reject'){var note=prompt('Reject reason (feeds the learning log):','');if(note===null)return;send(card,act,note)}
 else send(card,act)});
document.addEventListener('change',function(e){
 if(e.target.id==='checkall'){var on=e.target.checked;
  pickable().forEach(function(c){c.querySelector('.pick').checked=on});updateCount()}
 else if(e.target.classList&&e.target.classList.contains('pick')){updateCount()}});
var onlyHigh=document.getElementById('onlyhigh');
if(onlyHigh)onlyHigh.addEventListener('click',function(e){e.preventDefault();
 pickable().forEach(function(c){c.querySelector('.pick').checked=(c.dataset.band==='high')});updateCount()});
// Campaign filter. One client now runs several campaigns at once — email, comment and DM —
// each at its own pace, and they all land in this one queue. The options are derived from
// the cards themselves rather than passed from Go, so the list always matches what is
// actually pending. Hidden when there is only one campaign to choose from.
var campSel=document.getElementById('campfilter');
if(campSel){var seen={},order=[];
 allCards().forEach(function(c){var k=c.dataset.campaign||'';if(!k||seen[k])return;seen[k]=1;order.push(k)});
 order.sort().forEach(function(k){var o=document.createElement('option');o.value=k;o.textContent=k;campSel.appendChild(o)});
 if(order.length<2){var w=document.getElementById('campwrap');if(w)w.style.display='none'}
 campSel.addEventListener('change',function(){var want=campSel.value;
  allCards().forEach(function(c){c.style.display=(!want||c.dataset.campaign===want)?'':'none'});
  updateCount()})}
var batchBtn=document.getElementById('approvechecked');
if(batchBtn)batchBtn.addEventListener('click',function(){
 var cards=checkedCards();
 if(!cards.length)return;
 if(!confirm('Approve '+cards.length+' checked draft(s)? Any inline edits you made are kept.'))return;
 var msg=document.getElementById('batchmsg');batchBtn.disabled=true;batchBtn.setAttribute('aria-busy','true');
 var i=0,q=Promise.resolve();
 cards.forEach(function(c){q=q.then(function(){i++;msg.textContent='Approving '+i+'/'+cards.length+'…';return send(c,'approve')})});
 q.then(function(){batchBtn.removeAttribute('aria-busy');batchBtn.disabled=false;
  msg.textContent='✓ '+cards.length+' approved and queued: applied by the next run';updateCount()})});
updateCount();
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
<p class="sub">Each campaign owns its goal, companion link and daily budget. Click a card to edit or pause it.</p>
{{$slug := .Client.Slug}}
{{if .Rows}}
<div class="grid-cards">
{{range .Rows}}
<div class="card" style="cursor:pointer" onclick="location.href='/ui/{{$slug}}/campaign/{{.Slug}}'">
<div style="display:flex;justify-content:space-between;align-items:baseline;gap:.5rem">
<strong>{{.Slug}}</strong>
{{if eq .Status "paused"}}<span class="pill band-review_carefully">paused</span>{{else}}<span class="pill band-high">active</span>{{end}}
</div>
<span class="mut" style="font-size:.82rem">{{if .IsHarvest}}{{if eq .Channel "zillow_harvest"}}Leads From Zillow{{else}}Leads From Friends{{end}} &middot; collected by the bridge{{else}}{{.GoalType}}{{if .Objective}}: {{.Objective}}{{end}}{{end}}</span>
<div class="bar{{if ge (pct .UsedToday .Quota) 100}} full{{end}}"><i style="width:{{pct .UsedToday .Quota}}%"></i></div>
<div class="mut" style="font-size:.78rem">today {{.UsedToday}}/{{.Quota}} {{if .IsHarvest}}enriched{{else}}drafts{{end}}</div>
<div class="chips" style="margin-top:.5rem;font-size:.78rem">
{{if .IsHarvest}}
<span class="pill">{{.Kept}} kept</span>
{{if .Await}}<span class="pill band-review_carefully">{{.Await}} awaiting decision</span>{{end}}
{{else}}
{{if .Pending}}<a class="pill band-high" style="text-decoration:none" href="/ui/{{$slug}}/approvals" onclick="event.stopPropagation()">{{.Pending}} awaiting approval</a>{{end}}
<span class="pill">{{.Sent}} sent{{if .LastSent}}, last {{.LastSent}}{{end}}</span>
{{end}}
</div>
{{if .DaemonState}}<div class="mut" style="font-size:.75rem;margin-top:.4rem">{{.DaemonState}}</div>{{end}}
</div>
{{end}}
</div>
{{else}}<div class="empty"><b>No campaigns yet.</b><br>Create one below, or tell the agent: <code>set up a cold-email campaign</code> (3 questions and it runs).</div>{{end}}

<h2>New campaign</h2>
<div class="card">
<p class="mut" style="font-size:.85rem;margin-top:0">One campaign per channel, each with its own goal — the same offer needs a different pen for a cold email, a direct message, a public comment and a standalone group post. You can also just tell the agent <code>set up a comment campaign</code>; both routes create the same thing.</p>
<div class="grid">
<label>Name <span class="mut">(becomes the campaign id)</span>
<input id="nc-slug" type="text" placeholder="e.g. leadup comments" autocomplete="off"></label>
<label>Channel
<select id="nc-channel">
<option value="email_first">Email</option>
<option value="messenger">Messenger DM</option>
<option value="comment">Facebook comments</option>
<option value="post">Posts into groups</option>
<option value="friend_harvest">Leads From Friends</option>
<option value="zillow_harvest">Leads From Zillow</option>
</select></label>
<label id="nc-quota-wrap">Daily budget <span class="mut">(max new drafts/day)</span>
<input id="nc-quota" type="number" min="1" max="500" value="40" style="width:8rem"></label>
</div>
<label id="nc-goal-wrap"><span id="nc-goal-label">Goal</span> <span class="mut" id="nc-goal-help">(your words — what should this campaign achieve, and what are you offering? Everything else is derived from this plus your client profile. You can refine it after creating.)</span>
<textarea id="nc-goal" style="min-height:80px" placeholder="e.g. Draw attention to the profile with genuinely useful answers on other people's posts, so readers click through and discover LeadUp"></textarea></label>
<p class="mut" id="nc-hint" style="font-size:.8rem;margin:-.3rem 0 .8rem"></p>
<button class="ok" id="nc-create">Create campaign</button>
<span class="mut" id="nc-msg" style="font-size:.85rem;margin-left:8px"></span>
</div>
<script>
(function(){
 var slug=document.getElementById('nc-slug'), ch=document.getElementById('nc-channel'),
     hint=document.getElementById('nc-hint'), btn=document.getElementById('nc-create'),
     msg=document.getElementById('nc-msg');
 function slugify(s){return s.toLowerCase().replace(/[^a-z0-9]+/g,'-').replace(/^-+|-+$/g,'').slice(0,60)}
 function show(id,on){var el=document.getElementById(id);if(el)el.style.display=on?'':'none'}
 function sync(){
  var v=ch.value, id=slugify(slug.value), harvest=(v==='friend_harvest'||v==='zillow_harvest');
  var what = v==='comment' ? 'After creating, pick which groups to comment in — prefilled from the groups this client already scans.'
    : v==='post' ? 'After creating, pick which groups to post into. Posting is the most exposed channel, so its daily cap is the lowest.'
    : v==='messenger' ? 'Targets contacts that have a Facebook profile. Nothing sends without your approval.'
    : v==='friend_harvest' ? 'Created paused. After creating, paste the seed profiles whose friend lists to walk; the daily pace comes from agency settings and can be overridden there.'
    : v==='zillow_harvest' ? 'Created paused. After creating, paste the Zillow directory urls and the name keywords; the daily pace comes from agency settings and can be overridden there.'
    : 'Targets a segment of contacts that have an email address.';
  hint.innerHTML = (id? 'id: <code>'+id+'</code> &middot; ' : '') + what;
  // Only ask for what this channel reads. A harvest campaign has no drafts, so the draft
  // budget would be written and never read; Zillow has no judgement step, so it has no goal.
  show('nc-quota-wrap', !harvest);
  show('nc-goal-wrap', v!=='zillow_harvest');
  var gl=document.getElementById('nc-goal-label'), gh=document.getElementById('nc-goal-help');
  if(gl&&gh){
   if(v==='friend_harvest'){
    gl.textContent='Who to keep';
    gh.textContent='(your words — the agent judges every enriched friend against this and keeps or rejects them. Nothing is written or sent.)';
   } else {
    gl.textContent='Goal';
    gh.textContent='(your words — what should this campaign achieve, and what are you offering? Everything else is derived from this plus your client profile. You can refine it after creating.)';
   }
  }
 }
 slug.addEventListener('input',sync); ch.addEventListener('change',sync); sync();
 btn.addEventListener('click',function(){
  var id=slugify(slug.value);
  if(!id){msg.textContent='Give it a name first.';return}
  btn.disabled=true;btn.setAttribute('aria-busy','true');msg.textContent='Creating…';
  var v=ch.value, harvest=(v==='friend_harvest'||v==='zillow_harvest');
  var body={slug:id,channel_strategy:v};
  // Never submit a field the operator could not see (same rule as the edit form).
  if(v!=='zillow_harvest'){body.goal_description=document.getElementById('nc-goal').value.trim()}
  if(!harvest){body.daily_quota=parseInt(document.getElementById('nc-quota').value,10)}
  fetch('/api/ui/{{$slug}}/campaign-create',{method:'POST',headers:{'Content-Type':'application/json'},
   body:JSON.stringify(body)})
  .then(function(r){return r.ok?r.json():r.text().then(function(t){throw new Error(t)})})
  .then(function(j){
   if(j.ok){msg.textContent='✓ created — opening…';location.href=j.url}
   else{btn.disabled=false;btn.removeAttribute('aria-busy');msg.textContent='✗ '+j.error}})
  .catch(function(e){btn.disabled=false;btn.removeAttribute('aria-busy');msg.textContent='✗ '+e.message});
 });
})();
</script>
{{template "foot" .}}{{end}}

{{define "campaign"}}{{template "head" .}}
<div class="card">
<div style="display:flex;justify-content:space-between;flex-wrap:wrap;gap:.6rem;align-items:center">
<div>
<div class="chips">
{{if eq .Status "paused"}}<span class="pill band-review_carefully">paused: drafting and sending are stopped</span>{{else}}<span class="pill band-high">active</span>{{end}}
<span class="pill">today {{.UsedToday}}/{{.Quota}} drafts</span>
{{if .Pending}}<a class="pill band-high" style="text-decoration:none" href="/ui/{{.Client.Slug}}/approvals">{{.Pending}} awaiting approval</a>{{end}}
<span class="pill">audience: {{.Segment}}</span>
{{if .Sendboxes}}<span class="pill">boxes: {{range .Sendboxes}}{{.}} {{end}}</span>{{end}}
</div>
</div>
{{if eq .Status "paused"}}
<button class="ok" id="toggle" data-to="active">{{if or (eq .Channel "friend_harvest") (eq .Channel "zillow_harvest")}}Start harvest{{else}}Resume campaign{{end}}</button>
{{if or (eq .Channel "friend_harvest") (eq .Channel "zillow_harvest")}}<span class="mut" style="font-size:.83rem;margin-left:8px">Created paused on purpose: finish the inputs, save, then start. The daemon begins within a minute of starting.</span>{{end}}
{{else}}
<button id="toggle" data-to="paused">{{if or (eq .Channel "friend_harvest") (eq .Channel "zillow_harvest")}}Pause harvest{{else}}Pause campaign{{end}}</button>
{{end}}
</div>
</div>

<form id="campform">
<h2>Channel <span class="mut" style="font-size:.8rem">how this campaign reaches people — it decides how the goal is written</span></h2>
<div class="card">
<label>Channel
<select id="f-channel">
{{$ch := .Channel}}{{range .Channels}}<option value="{{.}}"{{if eq . $ch}} selected{{end}}>{{if eq . "email_first"}}Email{{else if eq . "messenger"}}Messenger DM{{else if eq . "comment"}}Facebook comments{{else if eq . "post"}}Posts into groups{{else if eq . "friend_harvest"}}Leads From Friends{{else if eq . "zillow_harvest"}}Leads From Zillow{{else}}{{.}}{{end}}</option>{{end}}
</select>
<small class="mut">One campaign per channel, each with its own goal: the same offer needs a different pen for a cold email, a direct message, a public comment, and a standalone group post. Email and Messenger target a segment of people; comments and posts target a list of groups.</small></label>
</div>

<div class="card" id="groupcard">
<h2 style="margin-top:0" id="grouphead">Groups</h2>
<p class="mut" style="font-size:.85rem;margin-top:0">Prefilled with the groups this client's collector already scans daily &mdash; untick the ones you don't want to act in. Scanning a group for leads and acting in it are different decisions, so this list is the campaign's own: changing it here does not change what the collector monitors.</p>
<div style="display:flex;flex-direction:column;gap:.4rem;margin-bottom:.8rem">
{{range .Groups}}<label style="display:flex;align-items:center;gap:.4rem;margin:0;font-weight:400">
<input type="checkbox" class="f-group" value="{{.url}}"{{if .picked}} checked{{end}} style="margin:0">
<span><code>{{.url}}</code>{{if not .scanned}} <span class="pill">not in the daily scan</span>{{end}}</span>
</label>{{else}}<span class="mut">This client has no Facebook groups in its collector sources yet — add one below, or add it as a private data source first so it gets scanned too.</span>{{end}}
</div>
<label>Add a group <span class="mut">(a full facebook.com/groups/… url)</span>
<div style="display:flex;gap:.5rem">
<input id="f-groupadd" type="text" placeholder="https://www.facebook.com/groups/…" style="flex:1">
<button type="button" id="f-groupaddbtn" style="width:auto">Add</button>
</div></label>
<p class="mut" style="font-size:.8rem;margin-bottom:0">The acting account must be a <strong>member</strong> of the group. Groups no account has joined are skipped and reported — joining is never automated.</p>
</div>

<div class="card" id="zillowcard">
<h2 style="margin-top:0">Zillow directory</h2>
<p class="mut" style="font-size:.85rem;margin-top:0">Leads From Zillow walks the agent directory: each location url × each keyword, page by page (Zillow serves at most 25 pages per query — coverage comes from more keywords and locations, not depth). Every agent profile is read once; a profile with an email or phone goes straight into the CRM (the data is published by the agent — no review step); one with neither is skipped. Zillow's "Press &amp; Hold" bot check pauses the collector and chimes until you pass it — then it continues.</p>
<label>Location urls <span class="mut">(one per line — a zillow.com/professionals/… page; keyword and page params are added automatically)</span>
<textarea id="f-zlocs" rows="4" placeholder="https://www.zillow.com/professionals/real-estate-agent-reviews/example-city-ca/?name=" spellcheck="false">{{range .ZillowLocations}}{{.}}
{{end}}</textarea></label>
<label>Keywords <span class="mut">(comma-separated; each is tried against every location — surnames work well)</span>
<input id="f-zkws" type="text" value="{{.ZillowKeywords}}" placeholder="kim, nguyen, tran"></label>
<label>Daily budget override <span class="mut">(blank = system setting {{.HarvestDefaults.daily}})</span>
<input id="f-zdaily" type="number" min="1" max="5000" value="{{.HarvestDaily}}"></label>
{{if .ZillowStatus}}
<h3 style="margin:.9rem 0 .3rem">Progress</h3>
<div class="statrow">
<a class="stat" href="/ui/{{.Client.Slug}}/campaign/{{.Slug}}/harvest?stage=listed" style="text-decoration:none"><b>{{.ZillowStatus.day_enriched}}/{{.ZillowStatus.day_budget}}</b><span>profiles read today</span></a>
<a class="stat" href="/ui/{{.Client.Slug}}/campaign/{{.Slug}}/harvest?stage=listed" style="text-decoration:none"><b>{{.ZillowStatus.cards_seen}}</b><span>agent cards listed</span></a>
<a class="stat" href="/ui/{{.Client.Slug}}/campaign/{{.Slug}}/harvest?stage=known" style="text-decoration:none"><b>{{.ZillowStatus.already_known}}</b><span>skipped (already known)</span></a>
<a class="stat" href="/ui/{{.Client.Slug}}/campaign/{{.Slug}}/harvest?stage=queued" style="text-decoration:none"><b>{{.ZillowStatus.queue}}</b><span>queued to read</span></a>
<a class="stat" href="/ui/{{.Client.Slug}}/campaign/{{.Slug}}/harvest?stage=in_flight" style="text-decoration:none"><b>{{.ZillowStatus.in_flight}}</b><span>reading now</span></a>
<a class="stat" href="/ui/{{.Client.Slug}}/campaign/{{.Slug}}/harvest?stage=kept" style="text-decoration:none"><b>{{.ZillowStatus.kept}}</b><span>added to CRM</span></a>
<a class="stat" href="/ui/{{.Client.Slug}}/campaign/{{.Slug}}/harvest?stage=rejected" style="text-decoration:none"><b>{{.ZillowStatus.rejected}}</b><span>skipped (no email/phone)</span></a>
<a class="stat" href="/ui/{{.Client.Slug}}/campaign/{{.Slug}}/harvest?stage=retried" style="text-decoration:none"><b>{{.ZillowStatus.retried}}</b><span>retried</span></a>
<div class="stat"><b>{{.ZillowStatus.eligible}}/{{.ZillowStatus.live_collectors}}</b><span>collectors usable / live{{if .ZillowStatus.quarantined}} · {{.ZillowStatus.quarantined}} quarantined{{end}}</span></div>
</div>
<p style="font-size:.9rem;margin:-4px 0 6px"><strong>Daemon:</strong> {{.ZillowStatus.state}}</p>
<div class="wrap"><table>
<tr><th>walking now</th><th>location</th><th>keyword</th><th>page</th><th>queries done</th><th>blocked on</th></tr>
<tr><td><span class="pill">{{.ZillowStatus.walk_state}}</span></td><td><code>{{.ZillowStatus.location}}</code></td><td>{{.ZillowStatus.keyword}}</td><td>{{.ZillowStatus.page}}</td><td>{{.ZillowStatus.queries_done}} of {{.ZillowStatus.queries_total}}</td><td class="mut">{{if .ZillowStatus.blocked}}{{.ZillowStatus.blocked}}{{else}}–{{end}}</td></tr>
</table></div>
<p class="mut" style="font-size:.8rem">Ask the agent for the same view any time: <code>tool crm-store harvest status --campaign {{.Slug}}</code>.</p>
{{end}}
</div>

<div class="card" id="harvestcard">
<h2 style="margin-top:0">Seed profiles</h2>
<p class="mut" style="font-size:.85rem;margin-top:0">Leads From Friends walks each seed's friend list in legs, enriches every friend once (20–40s apart, rotating collector accounts, spread across the day), and the agent keeps the ones that match the goal straight into the CRM — nothing is sent, so nothing needs approval. Paste any profile url form; each is cleaned and de-duplicated on save.</p>
<label>Profile urls <span class="mut">(one per line)</span>
<textarea id="f-seeds" rows="6" placeholder="https://www.facebook.com/example.seed.profile&#10;https://www.facebook.com/profile.php?id=100000000000000" spellcheck="false">{{range .Seeds}}{{.}}
{{end}}</textarea></label>
<div class="grid">
<label>Goal keywords <span class="mut">(comma-separated; friends whose list subtitle mentions one are enriched first)</span>
<input id="f-hkw" type="text" value="{{.HarvestKeywords}}" placeholder="realtor, loan officer, insurance"></label>
<label>Daily budget override <span class="mut">(blank = system setting {{.HarvestDefaults.daily}})</span>
<input id="f-hdaily" type="number" min="1" max="5000" value="{{.HarvestDaily}}"></label>
</div>
{{if .HarvestStatus}}
<h3 style="margin:.9rem 0 .3rem">Progress</h3>
<div class="statrow">
<a class="stat" href="/ui/{{.Client.Slug}}/campaign/{{.Slug}}/harvest?stage=listed" style="text-decoration:none"><b>{{.HarvestStatus.day_enriched}}/{{.HarvestStatus.day_budget}}</b><span>enriched today</span></a>
<a class="stat" href="/ui/{{.Client.Slug}}/campaign/{{.Slug}}/harvest?stage=listed" style="text-decoration:none"><b>{{.HarvestStatus.friends_seen}}</b><span>friends listed</span></a>
<a class="stat" href="/ui/{{.Client.Slug}}/campaign/{{.Slug}}/harvest?stage=known" style="text-decoration:none"><b>{{.HarvestStatus.already_known}}</b><span>skipped (already known)</span></a>
<a class="stat" href="/ui/{{.Client.Slug}}/campaign/{{.Slug}}/harvest?stage=queued" style="text-decoration:none"><b>{{.HarvestStatus.queue}}</b><span>queued to enrich</span></a>
<a class="stat" href="/ui/{{.Client.Slug}}/campaign/{{.Slug}}/harvest?stage=in_flight" style="text-decoration:none"><b>{{.HarvestStatus.in_flight}}</b><span>enriching now</span></a>
<a class="stat" href="/ui/{{.Client.Slug}}/campaign/{{.Slug}}/harvest?stage=await" style="text-decoration:none"><b>{{.HarvestStatus.await}}</b><span>awaiting decision</span></a>
<a class="stat" href="/ui/{{.Client.Slug}}/campaign/{{.Slug}}/harvest?stage=kept" style="text-decoration:none"><b>{{.HarvestStatus.kept}}</b><span>kept → CRM</span></a>
<a class="stat" href="/ui/{{.Client.Slug}}/campaign/{{.Slug}}/harvest?stage=rejected" style="text-decoration:none"><b>{{.HarvestStatus.rejected}}</b><span>rejected</span></a>
<a class="stat" href="/ui/{{.Client.Slug}}/campaign/{{.Slug}}/harvest?stage=retried" style="text-decoration:none"><b>{{.HarvestStatus.retried}}</b><span>retried</span></a>
<div class="stat"><b>{{.HarvestStatus.eligible}}/{{.HarvestStatus.live_collectors}}</b><span>collectors usable / live{{if .HarvestStatus.quarantined}} · {{.HarvestStatus.quarantined}} quarantined{{end}}</span></div>
</div>
<p style="font-size:.9rem;margin:-4px 0 6px"><strong>Daemon:</strong> {{.HarvestStatus.state}}</p>
<p class="mut" style="font-size:.83rem;margin:0 0 10px">Today's real ceiling: <strong>{{.HarvestStatus.ceiling}}</strong> ({{.HarvestStatus.ceiling_reason}}). The budget is a cap, not a quota — the 20–40 s spacing between profiles is fixed, and raising the budget late in the day never compresses it. "Friends listed" counts names read from friend lists (cheap, one leg reads ~80); only "enriched" opens a profile.</p>
<div class="wrap"><table>
<tr><th>seed</th><th>state</th><th>friends seen</th><th>legs</th><th>kept</th><th>rejected</th><th>last leg</th></tr>
{{range .HarvestStatus.seeds}}<tr><td><code>{{.url}}</code></td><td><span class="pill">{{.state}}</span></td><td>{{.friends_seen}}</td><td>{{.legs}}</td><td>{{.kept}}</td><td>{{.rejected}}</td><td class="mut">{{.last_leg}}{{if .box}} · {{.box}}{{end}}</td></tr>{{end}}
</table></div>
<p class="mut" style="font-size:.8rem">Ask the agent for the same view any time: <code>tool crm-store harvest status --campaign {{.Slug}}</code>. The daemon walks one seed at a time, top to bottom; "awaiting decision" clears on the next scheduled run.</p>
{{end}}
</div>

<section id="goalsec">
<h2><span id="goalhead">Goal</span> <span class="mut" style="font-size:.8rem" id="goalheadhelp">what every message in this campaign is trying to achieve</span></h2>
<div class="card">
<label><span id="goaldesclabel">Goal</span> <span class="mut" id="goaldeschelp">(your words. What should these emails achieve, and what are you offering? The agent derives everything below from this + your client profile.)</span>
<textarea id="f-goaldesc" style="min-height:90px" placeholder="e.g. Gioi thieu dich vu video cho cac professional, muc tieu la ho mo xem proposal ca nhan hoa va reply">{{.GoalDesc}}</textarea></label>
<p class="mut" id="goaldescnote" style="margin:-.4rem 0 .9rem;font-size:.85rem">That one answer is all the setup asks for &mdash; the agent derives the rest from it and your client profile (offer, audience, pain points are already there from onboarding).</p>
<div id="bankblock">
<label>Key messages <span class="mut">(one per line — what you want every email to TEACH the reader: the things you know from doing this work that they don't. Not slogans.)</span>
<textarea id="f-bank" style="min-height:110px" placeholder="e.g. Platforms reward posting that is correct, regular and complete&#10;e.g. Every piece has to be genuinely useful to the viewer — give away knowledge">{{.Bank}}</textarea></label>
<p class="mut" style="margin:-.4rem 0 .9rem;font-size:.85rem">Each email weaves 1&ndash;2 of these and rotates across the follow-ups, so the reader learns something instead of being pitched. Lines you write here are yours; lines the agent suggested are marked <em>(agent)</em> and you can edit or delete them. {{if eq .BankOperatorCount 0}}<strong>None of these are yours yet</strong> &mdash; an all-agent bank is the product describing itself.{{else}}{{.BankOperatorCount}} of them are yours.{{end}}</p>
</div>
<div id="ctablock">
<label>Call-to-action text <span class="mut">(the one ask at the end of the email)</span>
<input id="f-cta" type="text" value="{{.CTAText}}" placeholder="e.g. Worth a quick look?"></label>
</div>
</div>
</section>

<section id="companionsec">
<h2>Companion link <span class="mut" style="font-size:.8rem">the support link each message carries (demo page, sample video...)</span></h2>
<div class="card">
<label>How to get the link for each lead <span class="mut">(write it like instructions to an assistant: a fixed link, a per-language rule, or a step-by-step recipe; the agent follows it exactly)</span>
<textarea id="f-comp-instructions" style="min-height:90px" placeholder="e.g. use https://leadup.example/demo for every lead&#10;or: US lead → https://…/en, Vietnamese lead → https://…/vi&#10;or: personalize template X from the dossier, upload via API Y, use the returned URL">{{.CompanionInstructions}}</textarea></label>
<label>If getting the link fails
<select id="f-comp-onfail">
<option value="skip"{{if eq .CompanionOnFail "skip"}} selected{{end}}>skip that lead (no email without the link)</option>
<option value="default_link"{{if eq .CompanionOnFail "default_link"}} selected{{end}}>fall back to the default link below</option>
</select></label>
<label>Default link <span class="mut">(required when falling back)</span>
<input id="f-comp-default" type="text" value="{{.CompanionDefault}}" placeholder="https://…"></label>
<p class="mut" style="font-size:.78rem;margin-bottom:0">Leave the instructions empty to send messages without a companion link.</p>
</div>
</section>

<section id="sendingsec">
<h2 id="sendinghead">Sending</h2>
<div class="card">
<label id="quotalabel">Daily draft budget <span class="mut">(max new drafts per day for this campaign)</span>
<input id="f-quota" type="number" min="1" max="500" value="{{.Quota}}" style="width:8rem"></label>
<div id="sboxblock">
<label style="margin-bottom:.3rem">Sendboxes <span class="mut">(which mailboxes this campaign rotates onto. Tick none = every healthy box. Sending capacity is the sum of the ticked boxes' daily quotas.)</span></label>
<div style="display:flex;flex-wrap:wrap;gap:.5rem 1.1rem;margin-bottom:.9rem">
{{range .AllSendboxes}}<label style="display:flex;align-items:center;gap:.35rem;margin:0;font-weight:400">
<input type="checkbox" class="f-sbox" value="{{.slug}}"{{if .picked}} checked{{end}}{{if ne .status "healthy"}} disabled{{end}} style="margin:0">
<span{{if ne .status "healthy"}} class="mut" title="{{.status}}"{{end}}><code>{{.slug}}</code> {{.email}} <span class="mut">{{.quota_effective}}/day</span></span>
</label>{{else}}<span class="mut">No sendboxes connected yet.</span>{{end}}
</div>
<p class="mut" style="font-size:.82rem;margin:-.5rem 0 0">Ticked: <b id="sbox-sum">–</b></p>
</div>
<p class="mut" id="fbcapnote" style="font-size:.82rem;margin:-.2rem 0 0;display:none">Which Facebook account acts is decided by the account pool, not here — least-loaded eligible account, and for a group it must be a member. Per-account daily ceilings live in <a href="/ui/settings">agency settings</a>.</p>
</div>
</section>

<div class="acts">
<button class="ok" type="submit">Save changes</button>
<span id="savemsg" class="mut"></span>
</div>
</form>
<script>
(function(){
 var boxes=document.querySelectorAll('.f-sbox');
 if(!boxes.length)return;
 var caps={};{{range .AllSendboxes}}caps["{{.slug}}"]={{.quota_effective}};{{end}}
 function sum(){
  var n=0,c=0;
  boxes.forEach(function(b){if(b.checked){n++;c+=(caps[b.value]||0)}});
  var el=document.getElementById('sbox-sum');
  if(el)el.textContent = n? (n+' box(es), '+c+' emails/day of sending capacity') : 'none ticked — every healthy box will be used';
 }
 boxes.forEach(function(b){b.addEventListener('change',sum)});
 sum();
})();
// The group list belongs to comment campaigns only — show it when that channel is chosen,
// including immediately after the operator switches to it, so the list is filled in the same
// pass rather than after a save-and-reload.
(function(){
 var sel=document.getElementById('f-channel'), card=document.getElementById('groupcard'),
     head=document.getElementById('grouphead');
 if(!sel||!card)return;
 function usesGroups(v){return v==='comment'||v==='post'}
 function show(id,on){var el=document.getElementById(id);if(el)el.style.display=on?'':'none'}
 // Only show what the chosen channel actually READS. A field on screen is a promise that
 // setting it does something; leaving Key messages, a call-to-action or a draft budget on a
 // harvest campaign invites the operator to configure something no code path will ever read.
 // The matrix below is the code that reads each field, not a guess:
 //   goal.description  — every drafting channel; friend_harvest (agent's keep/reject judgement)
 //   message_bank/cta  — drafting channels only (draftBrief / draftWrite gates)
 //   daily_quota       — drafting channels only (draftBudget); harvest paces on harvest.daily_budget
 //   sendboxes         — email only;  groups — comment/post only;  companion link — email/DM only
 function sync(){
  var v=sel.value, groups=usesGroups(v), harvest=(v==='friend_harvest'||v==='zillow_harvest');
  card.style.display=groups?'':'none';
  show('harvestcard', v==='friend_harvest');
  show('zillowcard', v==='zillow_harvest');
  if(head)head.textContent=(v==='post')?'Groups to post in':'Groups to comment in';
  show('sboxblock', v==='email_first');
  show('fbcapnote', v!=='email_first');
  show('companionsec', v==='email_first'||v==='messenger');
  // Zillow has no judgement step at all — the daemon writes a contact whenever an email or a
  // phone is published — so the whole Goal card is dead there. Friends keeps the description
  // (the agent judges against it) but not the copywriting fields.
  show('goalsec', v!=='zillow_harvest');
  show('bankblock', !harvest);
  show('ctablock', !harvest);
  show('sendingsec', !harvest);
  var gh=document.getElementById('goalhead'), ghh=document.getElementById('goalheadhelp'),
      gl=document.getElementById('goaldesclabel'), glh=document.getElementById('goaldeschelp'),
      gn=document.getElementById('goaldescnote');
  if(gh&&ghh&&gl&&glh){
   if(v==='friend_harvest'){
    gh.textContent='Who to keep';
    ghh.textContent='the test every enriched friend is judged against';
    gl.textContent='Who to keep';
    glh.textContent='(your words. The agent reads this on each run and keeps or rejects every profile the daemon enriched. Nothing is drafted or sent.)';
    if(gn)gn.style.display='none';
   } else {
    gh.textContent='Goal';
    ghh.textContent='what every message in this campaign is trying to achieve';
    gl.textContent='Goal';
    glh.textContent='(your words. What should these emails achieve, and what are you offering? The agent derives everything below from this + your client profile.)';
    if(gn)gn.style.display='';
   }
  }
  var sh=document.getElementById('sendinghead');
  if(sh)sh.textContent=(v==='email_first')?'Sending':'Volume';
  var ql=document.getElementById('quotalabel');
  if(ql){
   var what = v==='comment' ? 'comment drafts' : v==='post' ? 'post drafts' : v==='messenger' ? 'message drafts' : 'drafts';
   ql.childNodes[0].nodeValue='Daily draft budget ';
   var hint=ql.querySelector('span');
   if(hint)hint.textContent='(max new '+what+' per day for this campaign)';
  }
 }
 sel.addEventListener('change',sync);sync();
 var add=document.getElementById('f-groupaddbtn'), input=document.getElementById('f-groupadd');
 if(!add||!input)return;
 add.addEventListener('click',function(){
  var u=input.value.trim().replace(/\/+$/,'');
  if(!u)return;
  if(u.toLowerCase().indexOf('facebook.com/groups/')<0){alert('That is not a Facebook group url.');return}
  var dup=Array.prototype.slice.call(document.querySelectorAll('.f-group'))
    .some(function(x){return x.value.toLowerCase()===u.toLowerCase()});
  if(dup){alert('That group is already listed.');input.value='';return}
  var row=document.createElement('label');
  row.style.cssText='display:flex;align-items:center;gap:.4rem;margin:0;font-weight:400';
  var cb=document.createElement('input');cb.type='checkbox';cb.className='f-group';cb.value=u;cb.checked=true;cb.style.margin='0';
  var sp=document.createElement('span');
  var code=document.createElement('code');code.textContent=u;sp.appendChild(code);
  var pill=document.createElement('span');pill.className='pill';pill.textContent='not in the daily scan';
  sp.appendChild(document.createTextNode(' '));sp.appendChild(pill);
  row.appendChild(cb);row.appendChild(sp);
  input.parentNode.parentNode.parentNode.querySelector('div[style*="flex-direction:column"]').appendChild(row);
  input.value='';
 });
})();
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
 var bank=document.getElementById('f-bank').value.split('\n').map(function(s){return s.trim()}).filter(Boolean)
   .map(function(s){var a=/\s*\(agent\)\s*$/.test(s);return {msg:s.replace(/\s*\(agent\)\s*$/,''),source:a?'agent':'operator',approved:true}});
 var instructions=document.getElementById('f-comp-instructions').value.trim();
 var chv=document.getElementById('f-channel').value;
 var isHarvest=(chv==='friend_harvest'||chv==='zillow_harvest');
 var patch={channel_strategy:chv};
 // Never submit a field the operator could not see. Sendboxes are hidden on non-email
 // channels, so saving must leave whatever is stored alone rather than writing the state of
 // controls nobody looked at. The same holds for every field sync() hides: the goal patch is
 // merged key by key on the server, so an omitted key keeps its stored value.
 if(chv!=='zillow_harvest'){
  var goal={description:document.getElementById('f-goaldesc').value.trim()};
  if(!isHarvest){
   goal.message_bank=bank;
   goal.cta={text:document.getElementById('f-cta').value.trim()};
  }
  // The companion link is only offered on email and DM; on a comment or post campaign the
  // controls are hidden, and an empty hidden textarea must not erase a stored link.
  if(chv==='email_first'||chv==='messenger'){
   goal.companion_doc = instructions ? {instructions:instructions,
     on_fail:document.getElementById('f-comp-onfail').value,
     default_link:document.getElementById('f-comp-default').value.trim()} : null;
  }
  patch.goal=goal;
 }
 // daily_quota bounds DRAFTS. Harvest campaigns draft nothing and pace on harvest.daily_budget.
 if(!isHarvest){patch.daily_quota=parseInt(document.getElementById('f-quota').value,10)}
 if(patch.channel_strategy==='email_first'){
  patch.sendboxes=Array.prototype.slice.call(document.querySelectorAll('.f-sbox:checked')).map(function(x){return x.value});
 }
 // Only a group-targeting campaign owns a group list. Sending it on an email or DM campaign
 // would let a stray tick quietly arm a channel the operator did not choose.
 if(patch.channel_strategy==='comment'||patch.channel_strategy==='post'){
  patch['audience.groups']=Array.prototype.slice.call(document.querySelectorAll('.f-group:checked')).map(function(x){return x.value});
 }
 // Leads From Friends owns its seed list + harvest overrides; only submit them on that channel.
 if(patch.channel_strategy==='zillow_harvest'){
  patch.zillow_locations=document.getElementById('f-zlocs').value.split('\n').map(function(s){return s.trim()}).filter(Boolean);
  patch.zillow_keywords=document.getElementById('f-zkws').value;
  var zd=parseInt(document.getElementById('f-zdaily').value,10);
  patch.harvest={daily_budget: isNaN(zd)?0:zd};
 }
 if(patch.channel_strategy==='friend_harvest'){
  patch.seed_profiles=document.getElementById('f-seeds').value.split('\n').map(function(s){return s.trim()}).filter(Boolean);
  var kw=document.getElementById('f-hkw').value.split(',').map(function(s){return s.trim().toLowerCase()}).filter(Boolean);
  var hd=parseInt(document.getElementById('f-hdaily').value,10);
  patch.harvest={goal_keywords:kw, daily_budget: isNaN(hd)?0:hd};
 }
 postUpdate(patch,
  function(j){btn.disabled=false;btn.removeAttribute('aria-busy');
   if(j.ok){msg.textContent = (j.changed&&j.changed.length) ? '✓ saved ('+j.changed.join(', ')+'): takes effect from the next run; the agent is notified' : '✓ nothing changed';}
   else{msg.textContent='✗ '+j.error}})});
</script>
</main></div></div></body></html>{{end}}

{{define "sent"}}{{template "head" .}}
<p class="sub">Everything published for this client, newest first — emails sent, comments answered and posts put into groups. A reply freezes that lead's sequence automatically; only replies drive action, opens and clicks are directional.</p>
<div class="statrow">
<div class="stat"><b>{{.Total}}</b><span>published</span></div>
<div class="stat hot"><b>{{.Replied}}</b><span>got a reply</span></div>
{{if .Rate}}<div class="stat"><b>{{.Rate}}</b><span>reply rate</span></div>{{end}}
</div>
{{if .Rows}}
<input id="sentfilter" type="search" placeholder="Filter by name, email, campaign, sendbox..." style="max-width:380px;margin-bottom:10px">
<div class="wrap"><table id="senttable"><tr><th>channel</th><th>to</th><th>campaign</th><th>step</th><th>account</th><th>published</th><th>status</th></tr>
{{$slug := .Client.Slug}}{{range .Rows}}<tr{{if .Lead}} style="cursor:pointer" onclick="location.href='/ui/{{$slug}}/contact/{{.Lead}}'"{{end}}>
<td><span class="chanbadge chan-{{.Channel}}">{{if eq .Channel "comment"}}Comment{{else if eq .Channel "post"}}Post{{else if eq .Channel "messenger"}}DM{{else}}Email{{end}}</span></td>
<td>{{if .Name}}<strong>{{.Name}}</strong> <span class="mut" style="font-size:.78rem">{{.To}}</span>{{else if .Lead}}{{.To}}{{else}}<a href="{{.To}}" target="_blank" rel="noopener" class="mut" style="font-size:.8rem">{{.To}}</a>{{end}}</td>
<td class="mut">{{.Campaign}}</td>
<td>{{if ne .Channel "email"}}<span class="mut">—</span>{{else if eq .Step 1}}<span class="pill">cold</span>{{else}}<span class="pill">bump {{.Step}}</span>{{end}}</td>
<td class="mut">{{.Sendbox}}</td>
<td class="mut" style="font-variant-numeric:tabular-nums">{{if ge (len .SentAt) 16}}{{slice .SentAt 0 16}}{{else}}{{.SentAt}}{{end}}</td>
<td>
{{if .Replied}}<span class="pill band-high">replied</span>
{{else if .Clicked}}<span class="pill info">clicked</span>
{{else if .Opened}}<span class="pill">opened</span>
{{else}}<span class="pill" style="opacity:.55">sent</span>{{end}}
</td>
</tr>{{end}}</table></div>
{{if .Truncated}}<p class="mut" style="font-size:.8rem">Showing the newest {{len .Rows}} of {{.Total}}.</p>{{end}}
<script>
document.getElementById('sentfilter').addEventListener('input',function(){
 var q=this.value.toLowerCase();
 document.querySelectorAll('#senttable tr[onclick]').forEach(function(tr){
  tr.style.display=tr.textContent.toLowerCase().indexOf(q)>=0?'':'none'})});
</script>
{{else}}<div class="empty"><b>Nothing sent yet.</b><br>Approved drafts go out on the campaign's daily run and appear here.</div>{{end}}
{{template "foot" .}}{{end}}

{{define "sendboxes"}}{{template "head" .}}
<p class="sub">Sending mailboxes for this client. The App Password is entered here and only here, never in chat.</p>
{{if .Sendboxes}}
<div class="wrap"><table><tr><th>name</th><th>email</th><th>status</th><th>quota/day</th><th>warmup</th><th>last sync</th><th></th></tr>
{{range .Sendboxes}}<tr>
<td><code>{{.slug}}</code></td><td>{{.email}}</td>
<td><span class="pill{{if eq .status "healthy"}} band-high{{else}} band-review_carefully{{end}}"><span class="dot{{if eq .status "healthy"}} ok{{else}} warn{{end}}"></span>{{.status}}</span></td>
<td>{{.quota_effective}}</td><td class="mut">{{.warmup_desc}}</td>
<td class="mut">{{.last_successful_sync_ts}}</td>
<td><a href="#quota" class="pick-quota" data-slug="{{.slug}}" data-quota="{{.quota_effective}}" data-step="{{.ramp_step}}" data-max="{{.ramp_max}}">quota</a> · <a href="#connect" class="pick-box" data-email="{{.email}}">connect / re-auth</a></td>
</tr>{{end}}</table></div>
{{else}}<div class="empty"><b>No sendboxes yet.</b><br>Connect the first one below.</div>{{end}}

<h2>Sender identity <span class="mut" style="font-size:.8rem">who every email is from and how it signs off</span></h2>
<div class="card" style="max-width:560px">
<form id="senderform">
<label>From name <span class="mut">(the person who signs; drafts must end with this given name or they are refused)</span>
<input id="f-fromname" type="text" value="{{.Sender.from_name}}" placeholder="e.g. Binh Nguyen"></label>
<label>Title <span class="mut">(optional)</span>
<input id="f-fromtitle" type="text" value="{{.Sender.from_title}}" placeholder="e.g. Founder, LeadUp"></label>
<label>Signature block <span class="mut">(the footer line; keep it one line)</span>
<input id="f-sigblock" type="text" value="{{.Sender.signature_block}}" placeholder="e.g. Binh Nguyen | Founder, LeadUp | https://leadupteam.com"></label>
<button class="ok" type="submit">Save sender identity</button>
<span id="sendermsg" class="mut"></span>
</form>
</div>

<h2 id="quota">Daily quota &amp; warm-up <span class="mut" style="font-size:.8rem">today's cap, and how it grows by itself</span></h2>
<div class="card" style="max-width:560px">
<form id="quotaform">
<label>Sendbox
<select id="q-slug">{{range .Sendboxes}}<option value="{{.slug}}">{{.slug}} — {{.email}}</option>{{end}}</select></label>
<label>Quota today <span class="mut">(emails/day from today)</span>
<input id="q-quota" type="number" min="1" max="500" value="20"></label>
<label>Auto-increase per day <span class="mut">(0 = fixed; e.g. 5 ramps a new box up automatically)</span>
<input id="q-step" type="number" min="0" max="50" value="0"></label>
<label>Cap <span class="mut">(the ramp stops here; consumer @gmail.com should stay ≤ ~50 cold/day)</span>
<input id="q-max" type="number" min="1" max="500" value="50"></label>
<button class="ok" type="submit">Save quota</button>
<span id="quotamsg" class="mut"></span>
</form>
<p class="mut" style="font-size:.8rem;margin-bottom:0">The ramp is computed, not scheduled: every send, rotation decision and this page derive today's
cap from <code>start + step × days</code>, so restarts change nothing and no one has to remember to raise it.</p>
</div>

<h2 id="connect">Connect a sending mailbox (Gmail App Password)</h2>
<div class="card" style="max-width:560px">
<form id="authform">
<label>Gmail address
<input id="f-email" type="email" placeholder="you@gmail.com" required></label>
<label>App Password <span class="mut">(16 characters: Google Account → Security → App passwords)</span>
<input id="f-pass" type="password" autocomplete="off" placeholder="xxxx xxxx xxxx xxxx" required></label>
<button class="ok" type="submit">Connect &amp; verify</button>
<span id="authmsg" class="mut"></span>
</form>
<p class="mut" style="font-size:.8rem">Reconnecting an address in the list updates that same mailbox; a new address is added
automatically under the next free internal name, nothing else to fill in.</p>
<p class="mut" style="font-size:.8rem;margin-bottom:0">The password goes from this page straight to Gmail over TLS and is stored only on this machine
(<code>sendboxes/&lt;slug&gt;/credentials.json</code>, permissions 0600). Never paste an App Password
into the agent chat; this page is the one intended place for it.</p>
</div>
<script>
var CLIENT="{{.Client.Slug}}";
document.querySelectorAll('.pick-box').forEach(function(a){a.addEventListener('click',function(){
 document.getElementById('f-email').value=this.dataset.email;
 document.getElementById('f-pass').focus()})});
document.querySelectorAll('.pick-quota').forEach(function(a){a.addEventListener('click',function(){
 document.getElementById('q-slug').value=this.dataset.slug;
 document.getElementById('q-quota').value=this.dataset.quota;
 document.getElementById('q-step').value=this.dataset.step||0;
 if((this.dataset.max||"0")!=="0")document.getElementById('q-max').value=this.dataset.max;
 document.getElementById('q-quota').focus()})});
document.getElementById('quotaform').addEventListener('submit',function(e){
 e.preventDefault();
 var btn=this.querySelector('button');btn.disabled=true;btn.setAttribute('aria-busy','true');
 var msg=document.getElementById('quotamsg');msg.textContent='Saving…';
 fetch('/api/ui/'+CLIENT+'/sendbox-quota',{method:'POST',headers:{'Content-Type':'application/json'},
  body:JSON.stringify({slug:document.getElementById('q-slug').value,
   quota:parseInt(document.getElementById('q-quota').value,10),
   step_per_day:parseInt(document.getElementById('q-step').value,10)||0,
   max_quota:parseInt(document.getElementById('q-max').value,10)||0})})
 .then(function(r){return r.json()})
 .then(function(j){btn.disabled=false;btn.removeAttribute('aria-busy');
  if(j.ok){msg.textContent='✓ '+(j.note||'saved')+' — today: '+j.effective_today+'/day';setTimeout(function(){location.reload()},1200);}
  else{msg.textContent='✗ '+(j.error||'failed');}})
 .catch(function(){btn.disabled=false;btn.removeAttribute('aria-busy');msg.textContent='✗ network error';});
});
document.getElementById('senderform').addEventListener('submit',function(e){
 e.preventDefault();
 var btn=this.querySelector('button');btn.disabled=true;btn.setAttribute('aria-busy','true');
 var msg=document.getElementById('sendermsg');msg.textContent='Saving…';
 fetch('/api/ui/'+CLIENT+'/sender-update',{method:'POST',headers:{'Content-Type':'application/json'},
  body:JSON.stringify({from_name:document.getElementById('f-fromname').value,
   from_title:document.getElementById('f-fromtitle').value,
   signature_block:document.getElementById('f-sigblock').value})})
 .then(function(r){return r.json()})
 .then(function(j){btn.disabled=false;btn.removeAttribute('aria-busy');
  msg.textContent=j.ok?((j.changed&&j.changed.length)?'✓ saved ('+j.changed.join(', ')+')':'✓ nothing changed'):'✗ '+(j.error||'failed');})
 .catch(function(){btn.disabled=false;btn.removeAttribute('aria-busy');msg.textContent='✗ network error';});
});
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
   msg.textContent='✗ '+(j.error||'failed')+(j.detail?': '+j.detail:'')+' (check the address and the App Password)'}})
 .catch(function(err){btn.disabled=false;btn.removeAttribute('aria-busy');msg.textContent='✗ '+err.message})});
</script>
</main></div></div></body></html>{{end}}

{{define "extension"}}{{template "head" .}}
<div class="card" style="max-width:660px">
{{if .Ext.CheckedIn}}
<p><span class="pill band-high">✓ extension connected</span> <span class="mut">last check-in {{.Ext.LastCheck}}{{if .Ext.Instance}} · {{.Ext.Instance}}{{end}}</span></p>
<p class="mut">The Chrome extension for this client is talking to the collector. Nothing to do here.</p>
{{else}}
<p><span class="pill band-review_carefully">not connected yet</span> <span class="mut">no check-in from this client's extension since the bridge started</span></p>
<h2 style="margin-top:.6rem">Install in 3 steps, no path typing</h2>
<ol class="steps">
<li><button class="ok" id="reveal" style="padding:.3rem .9rem">Open the extension folder</button>
<span id="revealmsg" class="mut"></span><br>
<span class="mut" style="font-size:.8rem">Finder/Explorer opens the exact folder. Keep that window visible.</span></li>
<li>In the Chrome profile for <strong>{{.Client.Slug}}</strong>, open <code>chrome://extensions</code> and switch on <strong>Developer mode</strong> (top right).</li>
<li><strong>Drag the opened folder</strong> from Finder/Explorer and drop it anywhere on the <code>chrome://extensions</code> page: that installs it (same as "Load unpacked", minus the file picker). This page flips to <span class="pill band-high">✓ connected</span> on its own once the extension checks in.</li>
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

// harvestCommonStatus computes the channel-agnostic half of a harvest campaign's
// Progress panel: kept/rejected from the client-wide registry (the source of
// truth), collector availability from the operator-wide ledger, the daemon's
// current state in words, and today's real ceiling. Shared by the Friends and
// Zillow cards.
func (b *bridge) harvestCommonStatus(p *harvestProgress, hc harvestConfig, outreachDir string) map[string]any {
	campaign := p.Campaign
	seedOf := map[string]bool{}
	for _, sd := range p.Seeds {
		seedOf[sd.URL] = true
	}
	keptTotal, rejTotal := 0, 0
	if reg, rerr := withSeen(outreachDir, func(*seenRegistry) error { return nil }); rerr == nil {
		for _, sp := range reg.Profiles {
			// This campaign's decisions: recorded under it, or (Friends) reached via one of its seeds.
			if sp.Campaign != campaign && !seedOf[sp.Seed] {
				continue
			}
			switch sp.Status {
			case "kept":
				keptTotal++
			case "rejected":
				rejTotal++
			}
		}
	}
	nowT := time.Now()
	liveBoxes := b.liveCollectors(nowT)
	live := len(liveBoxes)
	quarantined, quarantineUntil := 0, ""
	eligible, zillowBlocked := 0, 0
	if l, lerr := withLedger(b.uiDataRoot, nowT, func(*harvestLedger) error { return nil }); lerr == nil {
		for _, bx := range liveBoxes {
			lb := l.Boxes[bx.InstanceID]
			if lb != nil && lb.quarantined(nowT) {
				quarantined++
				if quarantineUntil == "" || lb.QuarantinedUntil < quarantineUntil {
					quarantineUntil = lb.QuarantinedUntil
				}
			} else if lb == nil || lb.DayJobs < hc.PerBoxBudget {
				// Zillow: a box that hit the final bot-check block today is not used for
				// Zillow even though the ledger considers it eligible.
				if hc.Channel == zillowChannel && p.Zillow != nil && p.Zillow.zillowBoxBlocked(bx.InstanceID, nowT) {
					zillowBlocked++
					continue
				}
				eligible++
			}
		}
	}
	state := "walking"
	switch {
	case len(p.InFlight) > 0:
		state = "a collector job is running"
	case inQuietHours(nowT, hc.QuietFrom, hc.QuietTo):
		state = "quiet hours (" + hc.QuietFrom + "–" + hc.QuietTo + ") — nothing runs"
	case p.DayEnriched >= hc.DailyBudget:
		state = "daily budget reached — resumes tomorrow"
	case live == 0:
		state = "no collector is checking in — open the client Chrome profiles with the extension"
	case eligible == 0 && zillowBlocked > 0:
		state = fmt.Sprintf("paused: every usable collector (%d) hit Zillow's bot check today — open zillow.com in that Chrome profile and pass the check once; the walk resumes on its own", zillowBlocked)
	case eligible == 0 && quarantined > 0:
		until := quarantineUntil
		if t, perr := time.Parse(time.RFC3339, quarantineUntil); perr == nil {
			until = t.Local().Format("15:04")
		}
		state = fmt.Sprintf("paused: all %d live collector(s) quarantined after repeated failures — first one back at %s", quarantined, until)
	case eligible == 0:
		state = "paused: every live collector is at its per-collector daily cap"
	case len(p.Queue) == 0 && hc.Channel == zillowChannel && p.Zillow != nil && p.Zillow.Exhausted:
		state = "walk finished — every location × keyword page was read; pause the campaign or add locations/keywords"
	case len(p.Queue) == 0 && hc.Channel == zillowChannel:
		state = "reading the next directory page"
	case len(p.Queue) == 0 && hc.Channel != zillowChannel && harvestAllSeedsDone(p):
		state = "walk finished — every seed's friend list was read; pause the campaign or add seed profiles"
	case len(p.Queue) == 0:
		state = "reading the next friend-list leg"
	default:
		state = "pacing 20–40 s between profiles"
	}
	// Effective ceiling for the rest of today: the budget is a CAP, never a
	// quota — the real bound is min(daily, live boxes × per-box, minutes left ÷ ~0.5).
	minsLeft := 24*60 - (nowT.Hour()*60 + nowT.Minute())
	ceiling := hc.DailyBudget
	byBoxes := live * hc.PerBoxBudget
	byTime := p.DayEnriched + minsLeft*2 // ~30s per profile
	reason := "daily budget"
	if byBoxes < ceiling {
		ceiling, reason = byBoxes, fmt.Sprintf("%d collectors × %d per collector", live, hc.PerBoxBudget)
	}
	if byTime < ceiling {
		ceiling, reason = byTime, fmt.Sprintf("~%d min left at 20–40s per profile", minsLeft)
	}
	return map[string]any{
		"kept": keptTotal, "rejected": rejTotal, "live_collectors": live,
		"quarantined": quarantined, "eligible": eligible, "zillow_blocked": zillowBlocked, "state": state,
		"ceiling": ceiling, "ceiling_reason": reason,
		"day_enriched": p.DayEnriched, "day_budget": hc.DailyBudget,
		"queue": len(p.Queue), "in_flight": len(p.InFlight), "await": len(p.AwaitDecision),
		"already_known": p.Totals["already_known"], "retried": p.Totals["requeued"],
	}
}

// harvestAllSeedsDone: no seed left to walk (each is exhausted, removed, or errored).
func harvestAllSeedsDone(p *harvestProgress) bool {
	if len(p.Seeds) == 0 {
		return false
	}
	for _, sd := range p.Seeds {
		if !sd.Exhausted && !sd.Removed && sd.Error == "" {
			return false
		}
	}
	return true
}
