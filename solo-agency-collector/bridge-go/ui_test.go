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
