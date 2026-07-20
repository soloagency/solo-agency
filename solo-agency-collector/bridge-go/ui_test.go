package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEmailSyntaxOK(t *testing.T) {
	cases := []struct {
		email string
		want  bool
	}{
		{"a@b.com", true},
		{"first.last@sub.domain.co", true},
		{"a..b@c.com", false},
		{"a@b", false},
		{"", false},
		{"two@@x.com", false},
		{strings.Repeat("a", 250) + "@b.com", false},
	}
	for _, c := range cases {
		if got := emailSyntaxOK(c.email); got != c.want {
			t.Errorf("emailSyntaxOK(%q) = %v, want %v", c.email, got, c.want)
		}
	}
}

func TestEmailCheckContract(t *testing.T) {
	origMX, origHost := lookupMX, lookupHost
	defer func() { lookupMX, lookupHost = origMX, origHost }()

	lookupMX = func(domain string) ([]*net.MX, error) {
		return []*net.MX{{Host: "mx1.example.com.", Pref: 10}}, nil
	}
	res := emailCheck("a@b.com")
	if res.Status != "mx_ok" || !res.MXOK || len(res.MXHosts) != 1 || res.MXHosts[0] != "mx1.example.com" {
		t.Fatalf("mx_ok path wrong: %+v", res)
	}

	lookupMX = func(domain string) ([]*net.MX, error) { return nil, net.UnknownNetworkError("no") }
	lookupHost = func(domain string) ([]string, error) { return []string{"1.2.3.4"}, nil }
	res = emailCheck("a@b.com")
	if res.Status != "mx_ok" || res.MXHosts[0] != "b.com" {
		t.Fatalf("implicit-MX path wrong: %+v", res)
	}

	lookupHost = func(domain string) ([]string, error) { return nil, net.UnknownNetworkError("no") }
	res = emailCheck("a@b.com")
	if res.Status != "mx_fail" || res.MXOK {
		t.Fatalf("mx_fail path wrong: %+v", res)
	}

	res = emailCheck("bad..email@x.com")
	if res.Status != "syntax_invalid" || res.SyntaxOK {
		t.Fatalf("syntax path wrong: %+v", res)
	}
}

func TestUIFilesGuard(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("clients/x/y/outputs/latest/report.html", "<html>ok</html>")
	mustWrite("clients/x/y/outreach/sendboxes/sb-a/credentials.json", "{secret}")
	mustWrite("secrets/key.txt", "nope")
	mustWrite("clients/x/y/outreach/integrations/providers/provider_config.local.json", "{key}")

	if got := uiResolveFile(root, "/files/clients/x/y/outputs/latest/report.html"); got == "" {
		t.Fatal("expected report.html to resolve")
	}
	denied := []string{
		"/files/../../etc/passwd",
		"/files/clients/x/y/outreach/sendboxes/sb-a/credentials.json",
		"/files/secrets/key.txt",
		"/files/clients/x/y/outreach/integrations/providers/provider_config.local.json",
		"/files/bridge/ui_token",
	}
	for _, p := range denied {
		if got := uiResolveFile(root, p); got != "" {
			t.Errorf("expected %s to be denied, got %s", p, got)
		}
	}
}

func TestUIAuthFlow(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "clients"), 0o755); err != nil {
		t.Fatal(err)
	}
	b := &bridge{cfg: config{host: "127.0.0.1", port: 17321, configFile: filepath.Join(root, "collector", "collector_config.json")}}
	if err := b.initUI(); err != nil {
		t.Fatalf("initUI: %v", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/ui/enter/", b.handleUIEnter)
	mux.HandleFunc("/ui", b.uiAuth(b.handleUIHome))

	// no cookie -> locked
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/ui", nil))
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "Locked") {
		t.Fatalf("expected locked page, got %d %q", rec.Code, rec.Body.String()[:80])
	}

	// wrong token -> forbidden
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/ui/enter/wrong", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for wrong token, got %d", rec.Code)
	}

	// correct token -> cookie + redirect
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/ui/enter/"+b.uiToken, nil))
	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	cookie := rec.Result().Cookies()
	if len(cookie) == 0 || cookie[0].Value != b.uiToken {
		t.Fatal("expected session cookie")
	}

	// with cookie -> home renders
	req := httptest.NewRequest("GET", "/ui", nil)
	req.AddCookie(cookie[0])
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Solo Agency") {
		t.Fatalf("expected home page, got %d", rec.Code)
	}

	// token persisted for reuse
	if _, err := os.Stat(filepath.Join(root, "bridge", "ui_token")); err != nil {
		t.Fatalf("ui_token not persisted: %v", err)
	}
}

func TestDeriveDataRoot(t *testing.T) {
	root := t.TempDir()
	cfgFile := filepath.Join(root, "daily-content-pipeline", "collector", "collector_config.json")
	got := deriveDataRoot(config{configFile: cfgFile})
	want, _ := filepath.Abs(filepath.Join(root, "daily-content-pipeline"))
	if got != want {
		t.Fatalf("deriveDataRoot from configFile = %q, want %q", got, want)
	}
	out := filepath.Join(root, "daily-content-pipeline", "collector", "inbox")
	if err := os.MkdirAll(filepath.Join(root, "daily-content-pipeline", "clients"), 0o755); err != nil {
		t.Fatal(err)
	}
	got = deriveDataRoot(config{outputDir: out})
	if got != want {
		t.Fatalf("deriveDataRoot from outputDir = %q, want %q", got, want)
	}
}

func TestUIApprovalAndShortlistAPI(t *testing.T) {
	root := t.TempDir()
	ws := filepath.Join(root, "clients", "leadup", "main")
	mustJSON := func(rel string, body string) {
		p := filepath.Join(ws, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustJSON("outreach/campaigns/camp-a/outbox/pending_approval/2026-07-19/d1.json",
		`{"id":"d1","status":"pending_approval","to":"jane@x.com","subject":"Hello Jane",
		  "body_text":"Hi Jane","confidence_band":"high","step":1,
		  "warnings":["no phone"],"hooks_used":[{"type":"listing","evidence_url":"https://ex.com/1"}]}`)
	mustJSON("outreach/campaigns/camp-a/outbox/pending_approval/2026-07-19/d2.json",
		`{"id":"d2","status":"approved","to":"z@x.com","subject":"s","body_text":"b"}`)
	mustJSON("history/discovery_shortlist.json",
		`{"generated_at":"2026-07-19T01:00:00Z","candidates":[
		  {"n":1,"source_name":"FB Group A","source_url":"https://fb.com/g/a","platform":"facebook",
		   "cadence_suggested":"daily","why":"active","classification":"recommended_daily"}]}`)

	b := &bridge{cfg: config{host: "127.0.0.1", port: 17321,
		configFile: filepath.Join(root, "collector", "collector_config.json")}}
	mux := http.NewServeMux()
	b.registerUIRoutes(mux)

	authed := func(method, url, body string) *httptest.ResponseRecorder {
		var req *http.Request
		if body == "" {
			req = httptest.NewRequest(method, url, nil)
		} else {
			req = httptest.NewRequest(method, url, strings.NewReader(body))
		}
		req.AddCookie(&http.Cookie{Name: uiCookieName, Value: b.uiToken})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	// approvals page lists only pending drafts
	rec := authed("GET", "/ui/leadup/approvals", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "jane@x.com") {
		t.Fatalf("approvals page: %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "z@x.com") {
		t.Fatal("approved draft must not appear on approvals page")
	}
	// batch-approve controls render: check-all box, per-item checkboxes, batch button
	for _, want := range []string{`id="checkall"`, `class="pick"`, `id="approvechecked"`, `id="onlyhigh"`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("approvals page missing batch control %q", want)
		}
	}

	// unauthenticated POST is refused
	recNoAuth := httptest.NewRecorder()
	mux.ServeHTTP(recNoAuth, httptest.NewRequest("POST", "/api/ui/leadup/approval",
		strings.NewReader(`{"draft_id":"d1","decision":"approve"}`)))
	if recNoAuth.Code != http.StatusForbidden {
		t.Fatalf("unauthenticated POST = %d, want 403", recNoAuth.Code)
	}

	// POST outside /api/ui/ stays read-only
	if rec := authed("POST", "/ui/leadup/approvals", "{}"); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST to page = %d, want 405", rec.Code)
	}

	// invalid decision rejected
	if rec := authed("POST", "/api/ui/leadup/approval", `{"draft_id":"d1","decision":"yolo"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad decision = %d, want 400", rec.Code)
	}

	// valid approval with edit lands in ui_inbox as one JSONL line
	rec = authed("POST", "/api/ui/leadup/approval",
		`{"draft_id":"d1","campaign":"camp-a","decision":"approve","edited_subject":"Better subject"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("approval POST = %d body=%s", rec.Code, rec.Body.String())
	}
	inbox, err := os.ReadFile(filepath.Join(ws, "outreach", "ui_inbox", "approval_decisions.jsonl"))
	if err != nil {
		t.Fatalf("approval_decisions.jsonl: %v", err)
	}
	line := strings.TrimSpace(string(inbox))
	if strings.Count(line, "\n") != 0 {
		t.Fatalf("expected exactly 1 line, got %q", line)
	}
	for _, want := range []string{`"draft_id":"d1"`, `"decision":"approve"`, `"edited_subject":"Better subject"`, `"ts":`, `"ui_session":`} {
		if !strings.Contains(line, want) {
			t.Errorf("inbox line missing %s: %s", want, line)
		}
	}

	// shortlist page renders the candidate
	rec = authed("GET", "/ui/leadup/shortlist", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "FB Group A") {
		t.Fatalf("shortlist page: %d", rec.Code)
	}

	// shortlist POST: invalid entries skipped, valid appended
	rec = authed("POST", "/api/ui/leadup/shortlist",
		`{"decisions":[{"source_url":"https://fb.com/g/a","source_name":"FB Group A","decision":"approve","cadence":"weekly"},
		               {"source_url":"","decision":"approve"},
		               {"source_url":"https://fb.com/g/b","decision":"nope"}]}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"queued":1`) {
		t.Fatalf("shortlist POST = %d body=%s", rec.Code, rec.Body.String())
	}
	sl, err := os.ReadFile(filepath.Join(ws, "ui_inbox", "shortlist_decisions.jsonl"))
	if err != nil {
		t.Fatalf("shortlist_decisions.jsonl: %v", err)
	}
	if !strings.Contains(string(sl), `"cadence":"weekly"`) || strings.Contains(string(sl), "g/b") {
		t.Fatalf("shortlist inbox wrong: %s", sl)
	}
}

func TestUISendboxAuth(t *testing.T) {
	root := t.TempDir()
	ws := filepath.Join(root, "clients", "leadup", "main")
	sbPath := filepath.Join(ws, "outreach", "sendboxes", "sendboxes.json")
	if err := os.MkdirAll(filepath.Dir(sbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sbPath, []byte(`{"sendboxes": [{"slug": "sb-a", "email": "old@gmail.com", "domain": "gmail.com", "quota_today": 20, "status": "needs_reauth", "warmup_stage": "week_1", "imap_uid_cursor": 42, "last_successful_sync_ts": ""}]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	origVerify := gmailVerifyLogin
	defer func() { gmailVerifyLogin = origVerify }()
	var gotPass string
	gmailVerifyLogin = func(email, pass string) (int, error) {
		gotPass = pass
		if pass != "goodpass12345678" {
			return 0, fmt.Errorf("SMTPAuthenticationError: 535 bad credentials")
		}
		return 99, nil
	}

	b := &bridge{cfg: config{host: "127.0.0.1", port: 17321,
		configFile: filepath.Join(root, "collector", "collector_config.json")}}
	mux := http.NewServeMux()
	b.registerUIRoutes(mux)
	authed := func(method, url, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, url, strings.NewReader(body))
		req.AddCookie(&http.Cookie{Name: uiCookieName, Value: b.uiToken})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	// page renders with the existing box and the connect form
	rec := authed("GET", "/ui/leadup/sendboxes", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "old@gmail.com") ||
		!strings.Contains(rec.Body.String(), "App Password") {
		t.Fatalf("sendboxes page: %d", rec.Code)
	}
	// stylesheet serves without auth
	recCSS := httptest.NewRecorder()
	mux.ServeHTTP(recCSS, httptest.NewRequest("GET", "/ui/assets/pico.min.css", nil))
	if recCSS.Code != http.StatusOK || !strings.Contains(recCSS.Body.String(), "Pico CSS") {
		t.Fatalf("pico.css: %d", recCSS.Code)
	}

	// wrong password -> sanitized auth_failed, no password echo
	rec = authed("POST", "/api/ui/leadup/sendbox-auth",
		`{"slug": "sb-a", "email": "new@gmail.com", "app_password": "bad pass"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"auth_failed"`) {
		t.Fatalf("bad auth: %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "badpass") || strings.Contains(rec.Body.String(), "535") {
		t.Fatalf("response leaked credential/server detail: %s", rec.Body.String())
	}
	if gotPass != "badpass" {
		t.Fatalf("spaces not stripped before verify: %q", gotPass)
	}

	// good password -> credentials written 0600, sendbox healthy, cursor preserved
	rec = authed("POST", "/api/ui/leadup/sendbox-auth",
		`{"slug": "sb-a", "email": "new@gmail.com", "app_password": "good pass 1234 5678"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("good auth: %d %s", rec.Code, rec.Body.String())
	}
	credPathOut := filepath.Join(ws, "outreach", "sendboxes", "sb-a", "credentials.json")
	st, err := os.Stat(credPathOut)
	if err != nil {
		t.Fatalf("credentials not written: %v", err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("credentials perm = %o, want 600", st.Mode().Perm())
	}
	doc, _ := readJSONFile(sbPath)
	sb := mapsOf(mList(doc, "sendboxes"))[0]
	if mStr(sb, "status") != "healthy" || mStr(sb, "email") != "new@gmail.com" {
		t.Fatalf("sendbox not updated: %v", sb)
	}
	if mInt(sb, "imap_uid_cursor", -1) != 42 {
		t.Fatalf("re-auth must preserve the existing cursor, got %v", sb["imap_uid_cursor"])
	}

	// missing fields -> 400
	if rec := authed("POST", "/api/ui/leadup/sendbox-auth", `{"slug": "sb-a"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("missing fields = %d, want 400", rec.Code)
	}
	// no slug + known email -> re-auths the SAME box (non-tech flow)
	rec = authed("POST", "/api/ui/leadup/sendbox-auth",
		`{"email": "NEW@gmail.com", "app_password": "goodpass12345678"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"sendbox":"sb-a"`) {
		t.Fatalf("email-matched re-auth: %d %s", rec.Code, rec.Body.String())
	}
	// no slug + brand-new email -> next free conventional name
	rec = authed("POST", "/api/ui/leadup/sendbox-auth",
		`{"email": "second@gmail.com", "app_password": "goodpass12345678"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"sendbox":"sb-b"`) {
		t.Fatalf("auto-slug mint: %d %s", rec.Code, rec.Body.String())
	}
	// unsafe slug -> auth_failed via storage error, not a panic/traversal
	if rec := authed("POST", "/api/ui/leadup/sendbox-auth", `{"slug": "../evil", "email": "x@gmail.com", "app_password": "goodpass12345678"}`); !strings.Contains(rec.Body.String(), "auth_failed") {
		t.Fatalf("unsafe slug not rejected: %d %s", rec.Code, rec.Body.String())
	}
}

func TestUIFeaturePanels(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "clients", "leadup", "main"), 0o755); err != nil {
		t.Fatal(err)
	}
	b := &bridge{cfg: config{host: "127.0.0.1", port: 17321,
		configFile: filepath.Join(root, "collector", "collector_config.json")}}
	mux := http.NewServeMux()
	b.registerUIRoutes(mux)
	authed := func(url string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", url, nil)
		req.AddCookie(&http.Cookie{Name: uiCookieName, Value: b.uiToken})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}
	// client page: action cards with agent phrases (client-substituted) + UI links
	rec := authed("/ui/leadup")
	body := rec.Body.String()
	for _, want := range []string{"run today&#39;s content for leadup", "set up a cold-email campaign",
		"/ui/leadup/sendboxes", "agent chat", "web UI", "paste into"} {
		if !strings.Contains(body, want) {
			t.Errorf("client page missing %q", want)
		}
	}
	// home page: capability overview
	rec = authed("/ui")
	body = rec.Body.String()
	for _, want := range []string{"What this system can do", "Content pipeline", "Outreach + CRM"} {
		if !strings.Contains(body, want) {
			t.Errorf("home page missing %q", want)
		}
	}
}

func TestUIExtensionPage(t *testing.T) {
	root := t.TempDir()
	setupRoot := root // dataRoot = root; setup root = parent — build accordingly
	dataRoot := filepath.Join(setupRoot, "daily-content-pipeline")
	ws := filepath.Join(dataRoot, "clients", "leadup", "main")
	if err := os.MkdirAll(ws, 0o755); err != nil {
		t.Fatal(err)
	}
	extDir := filepath.Join(setupRoot, "extensions", "leadup")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		t.Fatal(err)
	}
	regPath := filepath.Join(dataRoot, "collector", "extension_registry.json")
	os.MkdirAll(filepath.Dir(regPath), 0o755)
	os.WriteFile(regPath, []byte(fmt.Sprintf(
		`{"clients": [{"client_slug": "leadup", "extension_folder": %q, "extension_instance_id": "leadup-local-collector"}]}`,
		extDir)), 0o644)

	b := &bridge{cfg: config{host: "127.0.0.1", port: 17321,
		configFile: filepath.Join(dataRoot, "collector", "collector_config.json")}}
	mux := http.NewServeMux()
	b.registerUIRoutes(mux)
	authed := func(method, url, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, url, strings.NewReader(body))
		req.AddCookie(&http.Cookie{Name: uiCookieName, Value: b.uiToken})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	rec := authed("GET", "/ui/leadup/extension", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "not connected yet") ||
		!strings.Contains(rec.Body.String(), "Drag the opened folder") {
		t.Fatalf("extension page: %d", rec.Code)
	}

	origOpen := uiOpenInFileManager
	defer func() { uiOpenInFileManager = origOpen }()
	var opened string
	uiOpenInFileManager = func(p string) error { opened = p; return nil }
	rec = authed("POST", "/api/ui/leadup/reveal-extension", "{}")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("reveal: %d %s", rec.Code, rec.Body.String())
	}
	if opened != extDir {
		t.Fatalf("opened %q, want %q", opened, extDir)
	}

	// missing folder -> folder_missing, opener NOT called
	os.RemoveAll(extDir)
	opened = ""
	rec = authed("POST", "/api/ui/leadup/reveal-extension", "{}")
	if !strings.Contains(rec.Body.String(), "folder_missing") || opened != "" {
		t.Fatalf("missing-folder guard: %s opened=%q", rec.Body.String(), opened)
	}

	// live check-in flips the page
	b.mu.Lock()
	if b.extensions == nil {
		b.extensions = map[string]extensionTelemetry{}
	}
	b.extensions["leadup-local-collector"] = extensionTelemetry{instanceID: "leadup-local-collector",
		clientSlug: "leadup", displayName: "LeadUp", lastCheckAt: time.Now()}
	b.mu.Unlock()
	rec = authed("GET", "/ui/leadup/extension", "")
	if !strings.Contains(rec.Body.String(), "extension connected") {
		t.Fatalf("connected state not shown")
	}
}

func TestShortID(t *testing.T) {
	cases := map[string]string{
		"c_01KXY7Q17X7MYGMTRSPPFNNR92": "c_…FNNR92",
		"d_01KABCDEF":                  "d_…ABCDEF",
		"short":                        "short",
		"":                             "",
	}
	for in, want := range cases {
		if got := shortID(in); got != want {
			t.Errorf("shortID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUIContactDetail(t *testing.T) {
	root := t.TempDir()
	ws := filepath.Join(root, "clients", "leadup", "main")
	crm := filepath.Join(ws, "outreach", "crm", "contacts")
	os.MkdirAll(crm, 0o755)
	// a fully-enriched contact: identities + hook (latest activity) + seed
	os.WriteFile(filepath.Join(crm, "c_01KXY7Q17X7MYGMTRSPPFNNR92.json"), []byte(`{
	  "id": "c_01KXY7Q17X7MYGMTRSPPFNNR92",
	  "name": {"full": "Susan Vo"},
	  "identities": {"emails": [{"address": "susan@kw.com", "status": "mx_ok"}],
	    "phones": [{"number": "+14155550101"}],
	    "socials": {"facebook": "https://facebook.com/susan.vo"},
	    "seeds": [{"url": "https://facebook.com/reel/123", "kind": "reel", "platform": "facebook", "status": "resolved"}]},
	  "lifecycle_stage": "lead",
	  "custom_fields": {"professional_vertical": "real_estate"},
	  "enrichment": {"confidence_band": "high",
	    "identity": {"still_active": "confirmed", "current_company": "KW"},
	    "writing_brief": {"one_liner": "top KW agent"},
	    "hooks": [{"type": "new_listing", "summary": "listed 123 Main St", "evidence_url": "https://z/1", "observed_date": "2026-07-14"}]}
	}`), 0o644)
	// a nameless email-only contact -> list shows short id
	os.WriteFile(filepath.Join(crm, "c_01BARE000000000000000000X.json"), []byte(
		`{"id": "c_01BARE000000000000000000X", "identities": {"emails": [{"address": "bare@x.com"}]}, "lifecycle_stage": "lead"}`), 0o644)
	// an activity
	act := filepath.Join(ws, "outreach", "crm", "activities", "2026-07")
	os.MkdirAll(act, 0o755)
	os.WriteFile(filepath.Join(act, "activities.jsonl"),
		[]byte(`{"contact_id": "c_01KXY7Q17X7MYGMTRSPPFNNR92", "type": "email_sent", "summary": "sent step 1", "by": "agent", "ts": "2026-07-19T10:00:00Z"}`+"\n"), 0o644)

	b := &bridge{cfg: config{host: "127.0.0.1", port: 17321,
		configFile: filepath.Join(root, "collector", "collector_config.json")}}
	mux := http.NewServeMux()
	b.registerUIRoutes(mux)
	get := func(url string) string {
		req := httptest.NewRequest("GET", url, nil)
		req.AddCookie(&http.Cookie{Name: uiCookieName, Value: b.uiToken})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s -> %d", url, rec.Code)
		}
		return rec.Body.String()
	}

	// list: clickable rows, short id for the nameless one, phone/social columns
	crmPage := get("/ui/leadup/crm")
	for _, want := range []string{"contact/c_01KXY7Q17X7MYGMTRSPPFNNR92", "Susan Vo",
		"c_…00000X", "14155550101", "enriched"} {
		if !strings.Contains(crmPage, want) {
			t.Errorf("crm list missing %q", want)
		}
	}
	if strings.Contains(crmPage, "c_01KXY7Q17X7MYGMTRSPPFNNR92</span>") {
		t.Error("full ULID leaked into the visible name cell")
	}

	// detail: identities, the hook (latest activity for personalization), evidence link, timeline
	d := get("/ui/leadup/contact/c_01KXY7Q17X7MYGMTRSPPFNNR92")
	for _, want := range []string{"Susan Vo", "susan@kw.com", "14155550101",
		"facebook.com/susan.vo", "Content clues", "reel", "resolved",
		"listed 123 Main St", "https://z/1", "2026-07-14", "email_sent", "top KW agent"} {
		if !strings.Contains(d, want) {
			t.Errorf("contact detail missing %q", want)
		}
	}

	// unknown contact -> 404
	req := httptest.NewRequest("GET", "/ui/leadup/contact/c_nope", nil)
	req.AddCookie(&http.Cookie{Name: uiCookieName, Value: b.uiToken})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown contact -> %d, want 404", rec.Code)
	}
}

func TestUICampaignPages(t *testing.T) {
	root := t.TempDir()
	ws := filepath.Join(root, "clients", "leadup", "main")
	mustJSON := func(rel, body string) {
		p := filepath.Join(ws, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustJSON("outreach/campaigns/camp-a/campaign_config.json",
		`{"campaign_slug":"camp-a","status":"active","daily_quota":30,
		  "audience":{"segment":"realtors"},"sendboxes":["sb-a"],
		  "goal":{"goal_type":"book_meeting","objective":"book 5 calls",
		    "offer":"free sample video","value_proposition":"we do the editing",
		    "proof_points":["did 40 videos for KW"],"cta":{"text":"Worth a look?"},
		    "companion_doc":{"instructions":"use https://x/demo","on_fail":"skip","default_link":""}}}`)
	mustJSON("outreach/campaigns/camp-b/campaign_config.json",
		`{"campaign_slug":"camp-b","status":"paused","daily_quota":10,
		  "audience":{"segment":"mortgage"},"goal":{"goal_type":"get_reply"}}`)
	mustJSON("outreach/campaigns/camp-a/outbox/pending_approval/2026-07-20/d1.json",
		`{"id":"d1","status":"pending_approval","to":"a@x.com","subject":"s","body_text":"b"}`)
	mustJSON("outreach/campaigns/camp-a/sent/2026-07/sent_log.jsonl",
		`{"lead_id":"c_1","campaign":"camp-a","step":1,"sent_at":"2026-07-18T09:00:00Z","sendbox":"sb-a","rfc_message_id":"<m1@x>"}`+"\n")

	b := &bridge{cfg: config{host: "127.0.0.1", port: 17321,
		configFile: filepath.Join(root, "collector", "collector_config.json")}}
	mux := http.NewServeMux()
	b.registerUIRoutes(mux)
	authed := func(method, url, body string) *httptest.ResponseRecorder {
		var req *http.Request
		if body == "" {
			req = httptest.NewRequest(method, url, nil)
		} else {
			req = httptest.NewRequest(method, url, strings.NewReader(body))
		}
		req.AddCookie(&http.Cookie{Name: uiCookieName, Value: b.uiToken})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	// list page: both campaigns, status pills, pending count, last sent
	rec := authed("GET", "/ui/leadup/campaigns", "")
	page := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(page, "camp-a") || !strings.Contains(page, "camp-b") {
		t.Fatalf("campaigns page: %d", rec.Code)
	}
	for _, want := range []string{"paused", "book_meeting", "1 awaiting approval", "1 sent", "2026-07-18"} {
		if !strings.Contains(page, want) {
			t.Fatalf("campaigns page missing %q", want)
		}
	}

	// detail page: form prefilled from config
	rec = authed("GET", "/ui/leadup/campaign/camp-a", "")
	page = rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("campaign detail: %d", rec.Code)
	}
	for _, want := range []string{"book 5 calls", "free sample video", "use https://x/demo",
		"Pause campaign", "did 40 videos for KW", "Worth a look?"} {
		if !strings.Contains(page, want) {
			t.Fatalf("campaign detail missing %q", want)
		}
	}
	if rec = authed("GET", "/ui/leadup/campaign/nope", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown campaign -> %d, want 404", rec.Code)
	}
	// paused campaign shows Resume
	if page = authed("GET", "/ui/leadup/campaign/camp-b", "").Body.String(); !strings.Contains(page, "Resume campaign") {
		t.Fatal("paused detail must offer Resume")
	}

	// API: pause camp-a through the whitelist + ui_inbox event
	rec = authed("POST", "/api/ui/leadup/campaign-update",
		`{"slug":"camp-a","patch":{"status":"paused","daily_quota":15}}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("campaign-update: %d %s", rec.Code, rec.Body.String())
	}
	cfgRaw, err := os.ReadFile(filepath.Join(ws, "outreach", "campaigns", "camp-a", "campaign_config.json"))
	if err != nil || !strings.Contains(string(cfgRaw), `"paused"`) || !strings.Contains(string(cfgRaw), "15") {
		t.Fatalf("config not updated: %v %s", err, cfgRaw)
	}
	inbox, err := os.ReadFile(filepath.Join(ws, "outreach", "ui_inbox", "campaign_edits.jsonl"))
	if err != nil || strings.Count(strings.TrimSpace(string(inbox)), "\n") != 0 {
		t.Fatalf("campaign_edits.jsonl: %v %q", err, inbox)
	}
	for _, want := range []string{`"campaign":"camp-a"`, "status", "daily_quota", "ui_session"} {
		if !strings.Contains(string(inbox), want) {
			t.Fatalf("edit event missing %q: %s", want, inbox)
		}
	}

	// API: non-whitelisted key refused, nothing appended
	rec = authed("POST", "/api/ui/leadup/campaign-update",
		`{"slug":"camp-a","patch":{"sendboxes":["evil"]}}`)
	if !strings.Contains(rec.Body.String(), "not operator-editable") {
		t.Fatalf("whitelist must hold on the API too: %s", rec.Body.String())
	}
	inbox2, _ := os.ReadFile(filepath.Join(ws, "outreach", "ui_inbox", "campaign_edits.jsonl"))
	if string(inbox2) != string(inbox) {
		t.Fatal("refused edit must not append an event")
	}
}

func TestUIContactMergeRedirectAndSuspects(t *testing.T) {
	root := t.TempDir()
	ws := filepath.Join(root, "clients", "leadup", "main")
	mustJSON := func(rel, body string) {
		p := filepath.Join(ws, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustJSON("outreach/crm/contacts/c_x.json",
		`{"id":"c_x","merge":{"status":"merged","merged_into":"c_y"}}`)
	mustJSON("outreach/crm/contacts/c_y.json",
		`{"id":"c_y","name":{"full":"Susan Vo"},"lifecycle_stage":"lead",
		  "identities":{"emails":[{"address":"susan@x.com"}]},
		  "duplicate_suspects":[{"id":"c_z","via":"social","value":"https://fb.com/page","at":"2026-07-20T11:00:00Z"}]}`)
	mustJSON("outreach/crm/activities/2026-07/activities.jsonl",
		`{"id":"a1","type":"merged","contact_id":"c_x","summary":"fragment history","by":"agent","ts":"2026-07-19T10:00:00Z"}`+"\n")

	b := &bridge{cfg: config{host: "127.0.0.1", port: 17321,
		configFile: filepath.Join(root, "collector", "collector_config.json")}}
	mux := http.NewServeMux()
	b.registerUIRoutes(mux)
	authed := func(url string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", url, nil)
		req.AddCookie(&http.Cookie{Name: uiCookieName, Value: b.uiToken})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	// a consolidated fragment's old link redirects to the survivor
	rec := authed("/ui/leadup/contact/c_x")
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/ui/leadup/contact/c_y" {
		t.Fatalf("tombstone must redirect: %d %q", rec.Code, rec.Header().Get("Location"))
	}
	// survivor page shows the suspect banner and the fragment's activity
	rec = authed("/ui/leadup/contact/c_y")
	page := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("survivor page: %d", rec.Code)
	}
	for _, want := range []string{"Possible duplicate", "c_z", "fragment history"} {
		if !strings.Contains(page, want) {
			t.Fatalf("survivor page missing %q", want)
		}
	}
}

func TestUISentPage(t *testing.T) {
	root := t.TempDir()
	ws := filepath.Join(root, "clients", "leadup", "main")
	mustJSON := func(rel, body string) {
		p := filepath.Join(ws, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustJSON("outreach/crm/contacts/c_1.json",
		`{"id":"c_1","name":{"full":"Susan Vo"},"identities":{"emails":[{"address":"susan@kw.com","is_primary":true}]}}`)
	mustJSON("outreach/crm/contacts/c_2.json",
		`{"id":"c_2","identities":{"emails":[{"address":"mike@rx.com"}]}}`)
	mustJSON("outreach/campaigns/camp-a/sent/2026-07/sent_log.jsonl",
		`{"lead_id":"c_1","campaign":"camp-a","step":1,"sendbox":"sb-a","sent_at":"2026-07-18T09:00:00Z","rfc_message_id":"<m1@x>"}
{"lead_id":"c_2","campaign":"camp-a","step":2,"sendbox":"sb-b","sent_at":"2026-07-19T10:00:00Z","rfc_message_id":"<m2@x>"}
`)
	// c_1 replied AFTER the send; c_2 has an open event pulled from tracking
	mustJSON("outreach/crm/activities/2026-07/activities.jsonl",
		`{"id":"a1","type":"email_reply","contact_id":"c_1","summary":"campaign reply (untriaged)","by":"rule","ts":"2026-07-18T12:00:00Z"}
`)
	mustJSON("outreach/tracking/events.jsonl",
		`{"kind":"open","m":"<m2@x>","ts":1721400000}
`)

	b := &bridge{cfg: config{host: "127.0.0.1", port: 17321,
		configFile: filepath.Join(root, "collector", "collector_config.json")}}
	mux := http.NewServeMux()
	b.registerUIRoutes(mux)
	req := httptest.NewRequest("GET", "/ui/leadup/sent", nil)
	req.AddCookie(&http.Cookie{Name: uiCookieName, Value: b.uiToken})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	page := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("sent page: %d", rec.Code)
	}
	for _, want := range []string{"Susan Vo", "susan@kw.com", "mike@rx.com", "camp-a",
		"sb-a", "sb-b", ">replied<", ">opened<", "bump 2", ">cold<",
		`id="sentfilter"`, "emails sent", "reply rate"} {
		if !strings.Contains(page, want) {
			t.Fatalf("sent page missing %q", want)
		}
	}
	// newest first: c_2's row (07-19) appears before c_1's (07-18)
	if strings.Index(page, "mike@rx.com") > strings.Index(page, "susan@kw.com") {
		t.Fatal("rows must be newest-first")
	}
}

func TestReplyPollerDedup(t *testing.T) {
	dir := t.TempDir()
	b := &bridge{}
	r1 := map[string]any{"lead_id": "c_1", "activity_seq": 7, "from": "a@x.com"}
	r2 := map[string]any{"lead_id": "c_2", "activity_seq": 9, "from": "b@x.com"}
	out := b.filterUnnotifiedReplies(dir, []map[string]any{r1, r2})
	if len(out) != 2 {
		t.Fatalf("first pass: want 2 fresh, got %d", len(out))
	}
	// same replies again (e.g. daily run re-synced the same UID) -> zero
	out = b.filterUnnotifiedReplies(dir, []map[string]any{r1, r2})
	if len(out) != 0 {
		t.Fatalf("second pass: want 0, got %d", len(out))
	}
	// a genuinely new reply still passes
	r3 := map[string]any{"lead_id": "c_1", "activity_seq": 12, "from": "a@x.com"}
	out = b.filterUnnotifiedReplies(dir, []map[string]any{r1, r3})
	if len(out) != 1 || mStr(out[0], "lead_id") != "c_1" {
		t.Fatalf("third pass: want just the new one, got %v", out)
	}
}

func TestReplyNotifyDegradesNoPanic(t *testing.T) {
	// notifyReplies runs unattended every 5 min; with no provider configured it
	// must degrade (local_path_only), log, and never panic or fail the poll.
	root := t.TempDir()
	c := uiClient{Slug: "leadup", Workspace: "main", Path: filepath.Join(root, "clients", "leadup", "main")}
	clientDir := filepath.Join(c.Path, "outreach")
	if err := os.MkdirAll(filepath.Join(clientDir, "crm", "contacts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(clientDir, "crm", "contacts", "c_1.json"),
		[]byte(`{"id":"c_1","name":{"full":"Jane Doe Realtor"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	b := &bridge{}
	b.uiDataRoot = root
	replies := []map[string]any{
		{"lead_id": "c_1", "activity_seq": 1, "from": "x@y.com", "campaign": "camp-a", "subject": "Re: idea"},
		{"lead_id": "c_2", "activity_seq": 2, "from": "z@y.com", "campaign": "camp-a", "subject": "interested"},
	}
	// must not panic even though no provider_config / defaults / notifications dir exist
	b.notifyReplies(c, clientDir, replies)
	// the notification was attempted and logged (degraded), dir auto-created
	logPath := filepath.Join(root, "notifications", "notification_log.md")
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("notification log not written: %v", err)
	}
	data, _ := os.ReadFile(logPath)
	for _, want := range []string{"local_path_only", "Reply Poller", "campaign_reply_detected"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("degraded log row missing %q: %s", want, data)
		}
	}
}

func TestAddrInUseFlapGuard(t *testing.T) {
	// isAddrInUse: real bind conflict, nil, unrelated error
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	_, err = net.Listen("tcp", ln.Addr().String())
	if !isAddrInUse(err) {
		t.Fatalf("real bind conflict not detected: %v", err)
	}
	if isAddrInUse(nil) || isAddrInUse(errors.New("connection refused")) {
		t.Fatal("false positives")
	}

	// probeHealthyBridge: true only for an answering bridge-shaped /status
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/status" {
			w.Write([]byte(`{"config_file":"/x/collector_config.json","active_jobs":[]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	if !probeHealthyBridge(strings.TrimPrefix(srv.URL, "http://")) {
		t.Fatal("healthy bridge not recognized")
	}
	dead, _ := net.Listen("tcp", "127.0.0.1:0")
	deadAddr := dead.Addr().String()
	dead.Close()
	if probeHealthyBridge(deadAddr) {
		t.Fatal("dead port must not probe healthy")
	}
	other := httptest.NewServer(http.NotFoundHandler())
	defer other.Close()
	if probeHealthyBridge(strings.TrimPrefix(other.URL, "http://")) {
		t.Fatal("foreign HTTP server must not probe healthy")
	}
}
