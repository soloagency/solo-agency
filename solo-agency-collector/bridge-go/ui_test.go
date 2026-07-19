package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
