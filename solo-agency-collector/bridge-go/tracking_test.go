package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Byte-for-byte parity with the Python packer in dashboard2.py.
func TestTrackTokenParity(t *testing.T) {
	tc := trackCfg{Enabled: true, Company: "acme", Secret: "test_secret_xyz",
		BaseURL: "https://widecast.ai/app/dashboard"}
	cases := []struct{ got, want string }{
		{tc.trackToken("o", "01ABC", ""), "eyJrIjoibyIsInQiOiJhY21lIiwibSI6IjAxQUJDIn0.4c134cf89faf298e090283f0c7fb2519"},
		{tc.trackToken("c", "01ABC", "https://leadupteam.com/Proposals/EN.html"), "eyJrIjoiYyIsInQiOiJhY21lIiwibSI6IjAxQUJDIiwidSI6Imh0dHBzOi8vbGVhZHVwdGVhbS5jb20vUHJvcG9zYWxzL0VOLmh0bWwifQ.659ece706b5f77e7c2027d68ed7bc4c4"},
		{tc.trackToken("u", "01ABC", ""), "eyJrIjoidSIsInQiOiJhY21lIiwibSI6IjAxQUJDIn0.7cbe3c4de049265f6fd08c40554f2485"},
		{tc.trackToken("c", "01ABC", "https://x.io/a?b=1&c=2"), "eyJrIjoiYyIsInQiOiJhY21lIiwibSI6IjAxQUJDIiwidSI6Imh0dHBzOi8veC5pby9hP2I9MSZjPTIifQ.60d154848bc76ac62fc4e35dfed311c6"},
	}
	for i, c := range cases {
		if c.got != c.want {
			t.Fatalf("vector %d mismatch:\n got=%s\nwant=%s", i, c.got, c.want)
		}
	}
}

func TestTrackHTMLInjectionAndHeaders(t *testing.T) {
	tc := trackCfg{Enabled: true, Company: "acme", Secret: "s", BaseURL: "https://wc.ai/app/dashboard"}
	html := `<p>Hi, see <a href="https://leadupteam.com/demo">the demo</a>.</p>`
	out := tc.trackHTMLBody(html, "<m1@x>", "https://leadupteam.com/demo")
	if !strings.Contains(out, `href="https://wc.ai/app/dashboard/t/c/`) {
		t.Fatal("companion link not click-wrapped")
	}
	if strings.Contains(out, `href="https://leadupteam.com/demo"`) {
		t.Fatal("original companion href must be replaced")
	}
	if !strings.Contains(out, `/t/o/`) || !strings.Contains(out, `width="1"`) {
		t.Fatal("open pixel not appended")
	}
	// header merge: https one-click first, mailto kept, Post header present
	lu, post := tc.unsubHeader("<m1@x>", "<mailto:a+unsub-t@x.com?subject=unsubscribe>")
	if !strings.HasPrefix(lu, "<https://wc.ai/app/dashboard/t/u/") || !strings.Contains(lu, "mailto:a+unsub-t") {
		t.Fatalf("bad List-Unsubscribe: %s", lu)
	}
	if post != "List-Unsubscribe=One-Click" {
		t.Fatalf("bad post header: %s", post)
	}
	// disabled tracking is a pure no-op
	off := trackCfg{}
	if off.trackHTMLBody(html, "m", "u") != html {
		t.Fatal("disabled tracking must not touch the body")
	}
	if l, p := off.unsubHeader("m", "<mailto:x>"); l != "<mailto:x>" || p != "" {
		t.Fatal("disabled tracking must leave the mailto header alone")
	}
}

func TestTrackPollWritesAndSuppresses(t *testing.T) {
	root := t.TempDir()
	clientDir := filepath.Join(root, "clients", "leadup", "main", "outreach")
	mk := func(rel, body string) {
		p := filepath.Join(root, "clients", "leadup", "main", filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
	}
	mk("outreach/crm/contacts/c_9.json", `{"id":"c_9","identities":{"emails":[{"address":"z@x.com"}]}}`)
	mk("outreach/campaigns/camp/sent/2026-07/sent_log.jsonl",
		`{"lead_id":"c_9","campaign":"camp","step":1,"sendbox":"sb-a","sent_at":"2026-07-20T09:00:00Z","rfc_message_id":"<mZ@x>"}`+"\n")

	// fake WideCast: one open, one unsub for <mZ@x>, plus a scanner fan-out
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer wc_live_") {
			w.WriteHeader(401)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"boot_id": "boot1", "restarted": false, "cursor": 4,
			"events": []map[string]any{
				{"seq": 1, "kind": "open", "m": "<mZ@x>", "ts": 1, "ip_hash": "h1"},
				{"seq": 2, "kind": "unsub", "m": "<mZ@x>", "ts": 2, "ip_hash": "h1"},
				{"seq": 3, "kind": "click", "m": "<mA@x>", "ts": 3, "ip_hash": "bot"},
				{"seq": 4, "kind": "click", "m": "<mB@x>", "ts": 4, "ip_hash": "bot"},
			},
		})
	}))
	defer srv.Close()

	cfg := trackCfg{Enabled: true, BaseURL: srv.URL, Company: "acme", Secret: "s",
		apiKey: "wc_live_test", clientDir: clientDir}
	cfg.pollTrackingEvents()

	// events.jsonl written
	evs := readJSONLines(filepath.Join(clientDir, "tracking", "events.jsonl"))
	if len(evs) != 4 {
		t.Fatalf("want 4 events written, got %d", len(evs))
	}
	// the one-click unsub suppressed + froze c_9
	store := newCrmStore(clientDir)
	ct := store.getContact("c_9")
	if mStr(ct, "sequence_state") != "frozen" {
		t.Fatal("unsub must freeze the sequence")
	}
	// cursor advanced so a second poll (fake server would repeat) is deduped by cursor
	cur, _ := readJSONFile(filepath.Join(clientDir, "tracking", ".pull_cursor.json"))
	if mInt(cur, "cursor", 0) != 4 || mStr(cur, "boot_id") != "boot1" {
		t.Fatalf("cursor not persisted: %v", cur)
	}
	// bot fan-out (5+ msgs one ip) would tag; here 'bot' hit only 2 msgs so NOT flagged
	// (assert the heuristic didn't over-flag the 2-message case)
	for _, e := range evs {
		if mStr(e, "ip_hash") == "bot" && mBool(e, "bot") {
			t.Fatal("2-message ip must not be flagged as bot (threshold is 5)")
		}
	}
}

func TestFetchTenantSecretAndCanSign(t *testing.T) {
	root := t.TempDir()
	clientDir := filepath.Join(root, "clients", "leadup", "main", "outreach")
	os.MkdirAll(clientDir, 0o755)

	// fake WideCast /v1/track/secret: returns a per-tenant secret for the key
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/track/secret" && strings.HasPrefix(r.Header.Get("Authorization"), "Bearer wc_live_") {
			json.NewEncoder(w).Encode(map[string]any{
				"object": "track.secret", "company_id": "acme", "secret": "derived_secret_abc"})
			return
		}
		w.WriteHeader(401)
	}))
	defer srv.Close()

	cfg := trackCfg{Enabled: true, BaseURL: srv.URL, apiKey: "wc_live_test", clientDir: clientDir}
	// before fetch: cannot sign, so signing is a pure no-op
	if cfg.canSign() {
		t.Fatal("must not sign before the secret is fetched")
	}
	html := `<a href="https://x.io/demo">d</a>`
	if cfg.trackHTMLBody(html, "<m@x>", "https://x.io/demo") != html {
		t.Fatal("un-fetched tracking must not touch the body")
	}

	// fetch caches the secret 0600
	cfg = cfg.fetchTenantSecret()
	if cfg.Company != "acme" || cfg.Secret != "derived_secret_abc" || !cfg.canSign() {
		t.Fatalf("fetch did not populate signing material: %+v", cfg)
	}
	info, err := os.Stat(filepath.Join(clientDir, "tracking", ".tenant_secret.json"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("secret cache must be 0600: %v %v", err, info.Mode())
	}

	// a fresh load (as the send path does) now reads the cache and can sign
	// without any network — company_id + secret come from the cache file
	os.MkdirAll(filepath.Join(root, "clients", "leadup", "main", "integrations", "providers"), 0o755)
	os.WriteFile(filepath.Join(root, "clients", "leadup", "main", "integrations", "providers", "provider_config.local.json"),
		[]byte(`{"active_provider":"widecast","providers":{"widecast":{"api_key_local":"wc_live_test"}},
		         "tracking":{"enabled":true,"base_url":"`+srv.URL+`"}}`), 0o644)
	fresh := loadTrackCfg(clientDir)
	if !fresh.canSign() || fresh.Company != "acme" || fresh.pinned {
		t.Fatalf("send-path load must sign from cache (not pinned): %+v", fresh)
	}
	out := fresh.trackHTMLBody(html, "<m@x>", "https://x.io/demo")
	if !strings.Contains(out, "/t/c/") || !strings.Contains(out, "/t/o/") {
		t.Fatal("cached secret must enable click-wrap + pixel")
	}

	// a manually pinned secret is honored and never auto-fetched
	os.WriteFile(filepath.Join(root, "clients", "leadup", "main", "integrations", "providers", "provider_config.local.json"),
		[]byte(`{"active_provider":"widecast","providers":{"widecast":{"api_key_local":"wc_live_test"}},
		         "tracking":{"enabled":true,"base_url":"`+srv.URL+`","company_id":"pinnedco","secret":"pinnedsec"}}`), 0o644)
	pin := loadTrackCfg(clientDir)
	if !pin.pinned || pin.Company != "pinnedco" || pin.Secret != "pinnedsec" {
		t.Fatalf("manual pin must win: %+v", pin)
	}
}
