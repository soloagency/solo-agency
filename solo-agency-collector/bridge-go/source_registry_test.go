package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSourceUIDCanonicalization(t *testing.T) {
	cases := []struct {
		in, wantUID, wantDomain string
	}{
		// Facebook groups collapse to groups/<id> whatever the decoration.
		{"https://www.facebook.com/groups/123456", "facebook.com/groups/123456", "facebook.com"},
		{"https://facebook.com/groups/123456/", "facebook.com/groups/123456", "facebook.com"},
		{"https://m.facebook.com/groups/123456?ref=share&mibextid=abc", "facebook.com/groups/123456", "facebook.com"},
		{"https://www.facebook.com/groups/123456/buy_sell_discussion", "facebook.com/groups/123456", "facebook.com"},
		{"https://www.facebook.com/groups/RealEstateOC/permalink/999", "facebook.com/groups/realestateoc", "facebook.com"},
		// Profiles keep identity params, drop tracking.
		{"https://www.facebook.com/profile.php?id=100001&fbclid=XYZ", "facebook.com/profile.php?id=100001", "facebook.com"},
		{"https://www.facebook.com/somepage?utm_source=x", "facebook.com/somepage", "facebook.com"},
		// Websites: path-preserving, scheme/www/params/fragment/trailing slash stripped.
		{"https://www.ocregister.com/real-estate/?utm_campaign=daily#top", "ocregister.com/real-estate", "ocregister.com"},
		{"http://ocregister.com/real-estate", "ocregister.com/real-estate", "ocregister.com"},
		{"https://example.com/", "example.com", "example.com"},
		// Mirror-prefix strip applies ONLY to known social platforms — real
		// hosts that merely start with m./web. keep their identity.
		{"https://web.dev/articles/vitals", "web.dev/articles/vitals", "web.dev"},
		{"https://m.me/somebusiness", "m.me/somebusiness", "m.me"},
		// l.facebook.com is a link REDIRECTOR: identity is the u= destination.
		{"https://l.facebook.com/l.php?u=https%3A%2F%2Fexample.com%2Fa&h=AT123", "example.com/a", "example.com"},
		{"https://l.instagram.com/?u=https%3A%2F%2Fother.org%2Fb", "other.org/b", "other.org"},
		// A redirector without a destination carries no identity at all.
		{"https://l.facebook.com/l.php?h=AT999", "", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		uid, domain := sourceUID(c.in)
		if uid != c.wantUID || domain != c.wantDomain {
			t.Errorf("sourceUID(%q) = (%q, %q), want (%q, %q)", c.in, uid, domain, c.wantUID, c.wantDomain)
		}
	}
	// Same source registered ten different ways = one uid.
	u1, _ := sourceUID("https://www.facebook.com/groups/123456?ref=share")
	u2, _ := sourceUID("m.facebook.com/groups/123456/about/")
	if u1 != u2 {
		t.Errorf("group variants did not collapse: %q vs %q", u1, u2)
	}
	// Two different redirector destinations must NOT share a uid.
	r1, _ := sourceUID("https://l.facebook.com/l.php?u=https%3A%2F%2Fsite-a.com%2Fx")
	r2, _ := sourceUID("https://l.facebook.com/l.php?u=https%3A%2F%2Fsite-b.com%2Fy")
	if r1 == r2 || r1 == "" {
		t.Errorf("redirector destinations collapsed or vanished: %q vs %q", r1, r2)
	}
	if len(uidHash(u1)) != 12 {
		t.Errorf("uidHash length = %d, want 12", len(uidHash(u1)))
	}
}

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what
// it printed — the CLIs under test write their JSON result to stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string)
	go func() {
		buf := make([]byte, 0, 4096)
		tmp := make([]byte, 4096)
		for {
			n, rerr := r.Read(tmp)
			buf = append(buf, tmp[:n]...)
			if rerr != nil {
				break
			}
		}
		done <- string(buf)
	}()
	defer func() {
		os.Stdout = old
	}()
	fn()
	w.Close()
	out := <-done
	os.Stdout = old
	return out
}

func registryCLICode(t *testing.T, args ...string) (int, string) {
	t.Helper()
	code := 0
	out := captureStdout(t, func() { code = runSourceRegistryCLI(args) })
	return code, out
}

func registryCLI(t *testing.T, args ...string) map[string]any {
	t.Helper()
	code, out := registryCLICode(t, args...)
	if code != 0 {
		t.Fatalf("source-registry %v exited %d", args, code)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("bad JSON from source-registry %v: %v\n%s", args, err, out)
	}
	return m
}

func TestSourceRegistryDueRecordRoundTrip(t *testing.T) {
	pipeline := t.TempDir()
	groupURL := "https://www.facebook.com/groups/oc-realtors?ref=share"

	// First client asks on an UNREGISTERED url: auto-created as "unclassified",
	// gets a scan (and the claim).
	res := registryCLI(t, "--pipeline", pipeline, "due", "--client", "angela", "--kind", "private", "--url", groupURL)
	if len(mList(res, "scan")) != 1 || len(mList(res, "reuse")) != 0 {
		t.Fatalf("fresh registry should demand a scan: %v", res)
	}
	if mList(res, "scan")[0].(map[string]any)["scope"] != "unclassified" {
		t.Fatalf("auto-created entry must be unclassified: %v", res)
	}

	// Unclassified sources never share: minh scans too (both-scan is the safe
	// default until an explicit register decides shared vs exclusive), so no
	// wait disposition applies here.
	res = registryCLI(t, "--pipeline", pipeline, "due", "--client", "minh", "--kind", "private",
		"--url", "https://m.facebook.com/groups/oc-realtors/")
	if len(mList(res, "scan")) != 1 || len(mList(res, "wait")) != 0 || len(mList(res, "reuse")) != 0 {
		t.Fatalf("unclassified source: both clients scan, no wait/reuse: %v", res)
	}

	// Record angela's completed scan (data dir must exist).
	dataDir := filepath.Join(pipeline, "collector", "inbox", "2026-08", "angela", "run1")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	registryCLI(t, "--pipeline", pipeline, "record", "--client", "angela", "--run", "run1",
		"--kind", "private", "--data-dir", dataDir, "--url", groupURL)

	// Unclassified is NEVER served as reuse — minh still scans.
	res = registryCLI(t, "--pipeline", pipeline, "due", "--client", "minh", "--kind", "private", "--url", groupURL)
	if len(mList(res, "reuse")) != 0 {
		t.Fatalf("unclassified entry must not be reused: %v", res)
	}

	// Register (default scope: shared). Now the fresh scan serves everyone.
	registryCLI(t, "--pipeline", pipeline, "register", "--client", "minh", "--kind", "private",
		"--url", groupURL)
	res = registryCLI(t, "--pipeline", pipeline, "due", "--client", "minh", "--kind", "private",
		"--url", "https://m.facebook.com/groups/oc-realtors/")
	reuse := mList(res, "reuse")
	if len(reuse) != 1 || len(mList(res, "scan")) != 0 {
		t.Fatalf("registered shared source with fresh scan should reuse: %v", res)
	}
	got := reuse[0].(map[string]any)
	if got["scanned_by"] != "angela" || got["uid_hash"] == "" {
		t.Fatalf("reuse pointer wrong: %v", got)
	}
	if got["data_dir"] != dataDir {
		t.Fatalf("reuse data_dir = %v, want %v", got["data_dir"], dataDir)
	}

	// Both clients are subscribers of ONE entry.
	res = registryCLI(t, "--pipeline", pipeline, "list")
	if int(res["count"].(float64)) != 1 {
		t.Fatalf("expected 1 registry entry, got %v", res["count"])
	}
	entry := mList(res, "sources")[0].(map[string]any)
	if len(mList(entry, "subscribers")) != 2 {
		t.Fatalf("expected 2 subscribers: %v", entry)
	}

	// Expire the scan by rewriting completed_at beyond the TTL -> scan again.
	regPath := sourceRegistryPath(pipeline)
	raw, _ := os.ReadFile(regPath)
	var reg sourceRegistry
	if err := json.Unmarshal(raw, &reg); err != nil {
		t.Fatal(err)
	}
	reg.Sources[0].LastScan.CompletedAt = time.Now().UTC().Add(-40 * time.Hour).Format(time.RFC3339)
	out, _ := json.Marshal(reg)
	if err := os.WriteFile(regPath, out, 0o600); err != nil {
		t.Fatal(err)
	}
	res = registryCLI(t, "--pipeline", pipeline, "due", "--client", "minh", "--kind", "private", "--url", groupURL)
	if len(mList(res, "scan")) != 1 {
		t.Fatalf("expired scan should demand a rescan: %v", res)
	}

	// minh now holds the claim: angela's due must WAIT with the stale pointer.
	res = registryCLI(t, "--pipeline", pipeline, "due", "--client", "angela", "--kind", "private", "--url", groupURL)
	wait := mList(res, "wait")
	if len(wait) != 1 || len(mList(res, "scan")) != 0 {
		t.Fatalf("claimed source should return wait for other clients: %v", res)
	}
	w := wait[0].(map[string]any)
	if w["claimed_by"] != "minh" || w["stale_data_dir"] != dataDir {
		t.Fatalf("wait entry wrong: %v", w)
	}

	// Failed record must NOT overwrite last_scan, and clears minh's claim.
	registryCLI(t, "--pipeline", pipeline, "record", "--client", "minh", "--run", "run2",
		"--kind", "private", "--status", "failed", "--url", groupURL)
	res = registryCLI(t, "--pipeline", pipeline, "list")
	entry = mList(res, "sources")[0].(map[string]any)
	if mStr(mMap(entry, "last_scan"), "run_id") != "run1" {
		t.Fatalf("failed record overwrote last_scan: %v", entry)
	}
	if mStr(mMap(entry, "last_failed"), "run_id") != "run2" {
		t.Fatalf("failed record not tracked: %v", entry)
	}
	if _, hasClaim := entry["scan_claim"]; hasClaim {
		t.Fatalf("record should clear the recording client's claim: %v", entry)
	}
}

func TestSourceRegistryExclusiveAlwaysScans(t *testing.T) {
	pipeline := t.TempDir()
	own := "https://www.facebook.com/angelasownpage"
	registryCLI(t, "--pipeline", pipeline, "register", "--client", "angela", "--url", own,
		"--scope", "exclusive", "--kind", "private")
	dataDir := filepath.Join(pipeline, "collector", "inbox", "2026-08", "angela", "runx")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	registryCLI(t, "--pipeline", pipeline, "record", "--client", "angela", "--run", "runx",
		"--kind", "private", "--data-dir", dataDir, "--url", own)
	// Fresh scan exists, but exclusive -> still scan (never served as reuse).
	res := registryCLI(t, "--pipeline", pipeline, "due", "--client", "angela", "--kind", "private", "--url", own)
	if len(mList(res, "scan")) != 1 || len(mList(res, "reuse")) != 0 {
		t.Fatalf("exclusive source must always scan: %v", res)
	}
}

func TestSourceRegistryValidationAndDowngrade(t *testing.T) {
	pipeline := t.TempDir()
	u := "https://www.facebook.com/groups/g1"

	if code, _ := registryCLICode(t, "--pipeline", pipeline, "register", "--client", "a",
		"--url", u, "--scope", "everyone"); code != 2 {
		t.Fatalf("bogus --scope must exit 2, got %d", code)
	}
	if code, _ := registryCLICode(t, "--pipeline", pipeline, "record", "--client", "a",
		"--run", "r1", "--url", u, "--status", "completed"); code != 2 {
		t.Fatalf("bogus --status must exit 2, got %d", code)
	}
	if code, _ := registryCLICode(t, "--pipeline", pipeline, "due", "--client", "a",
		"--kind", "social", "--url", u); code != 2 {
		t.Fatalf("bogus --kind must exit 2, got %d", code)
	}

	// exclusive -> shared downgrade via register is refused; set-scope is the override.
	registryCLI(t, "--pipeline", pipeline, "register", "--client", "a", "--url", u, "--scope", "exclusive")
	res := registryCLI(t, "--pipeline", pipeline, "register", "--client", "b", "--url", u, "--scope", "shared")
	made := mList(res, "registered")[0].(map[string]any)
	if made["ok"] != false || mStr(made, "reason") != "downgrade_refused_use_set_scope" {
		t.Fatalf("exclusive->shared downgrade must be refused: %v", made)
	}
	res = registryCLI(t, "--pipeline", pipeline, "list")
	if mList(res, "sources")[0].(map[string]any)["scope"] != "exclusive" {
		t.Fatalf("scope must remain exclusive after refused downgrade")
	}
	registryCLI(t, "--pipeline", pipeline, "set-scope", "--url", u, "--scope", "shared")
	res = registryCLI(t, "--pipeline", pipeline, "list")
	if mList(res, "sources")[0].(map[string]any)["scope"] != "shared" {
		t.Fatalf("set-scope must apply the explicit override")
	}
}

func TestSourceRegistryKindLaneSeparation(t *testing.T) {
	pipeline := t.TempDir()
	u := "https://www.ocregister.com/real-estate"
	registryCLI(t, "--pipeline", pipeline, "register", "--client", "a", "--url", u, "--kind", "public")
	pubDir := filepath.Join(pipeline, "collector", "public_pool", "abc")
	if err := os.MkdirAll(pubDir, 0o700); err != nil {
		t.Fatal(err)
	}
	registryCLI(t, "--pipeline", pipeline, "record", "--client", "a", "--run", "rp",
		"--kind", "public", "--data-dir", pubDir, "--url", u)

	// Same URL asked for the PRIVATE lane: the public scan must not satisfy it.
	res := registryCLI(t, "--pipeline", pipeline, "due", "--client", "b", "--kind", "private", "--url", u)
	if len(mList(res, "reuse")) != 0 {
		t.Fatalf("public scan must not serve a private-lane request: %v", res)
	}
	// Public lane requester gets the reuse.
	res = registryCLI(t, "--pipeline", pipeline, "due", "--client", "c", "--kind", "public", "--url", u)
	if len(mList(res, "reuse")) != 1 {
		t.Fatalf("public scan should serve a public-lane request: %v", res)
	}
}

func TestSearchPoolRoundTrip(t *testing.T) {
	pipeline := t.TempDir()

	out := captureStdout(t, func() {
		if code := runSearchPoolCLI([]string{"--pipeline", pipeline, "check",
			"--industry", "Real  Estate", "--keyword", "FAIR plan rate hike"}); code != 0 {
			t.Fatalf("check exited nonzero")
		}
	})
	var res map[string]any
	_ = json.Unmarshal([]byte(out), &res)
	if res["fresh"] != false {
		t.Fatalf("empty pool should not be fresh: %v", res)
	}

	resultsFile := filepath.Join(pipeline, "results.json")
	if err := os.WriteFile(resultsFile, []byte(`[{"url":"https://a.com","title":"A"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	out = captureStdout(t, func() {
		if code := runSearchPoolCLI([]string{"--pipeline", pipeline, "record", "--client", "angela",
			"--industry", "real estate", "--keyword", "FAIR  plan rate hike", "--results-file", resultsFile}); code != 0 {
			t.Fatalf("record exited nonzero")
		}
	})

	// Different client, differently-spaced/cased keyword and industry -> fresh hit.
	out = captureStdout(t, func() {
		if code := runSearchPoolCLI([]string{"--pipeline", pipeline, "check",
			"--industry", "REAL ESTATE", "--keyword", "fair plan RATE hike"}); code != 0 {
			t.Fatalf("check exited nonzero")
		}
	})
	_ = json.Unmarshal([]byte(out), &res)
	if res["fresh"] != true {
		t.Fatalf("normalized keyword should hit the pool: %v", res)
	}
	entry := mMap(res, "entry")
	if mStr(entry, "client_slug") != "angela" || len(mList(entry, "results")) != 1 {
		t.Fatalf("pool entry wrong: %v", entry)
	}
}

func TestPointUIDStability(t *testing.T) {
	uid1, _ := sourceUID("https://www.facebook.com/groups/123")
	uid2, _ := sourceUID("m.facebook.com/groups/123/")
	p1 := uidHash(uid1 + "|" + normalizeSocial("https://www.facebook.com/groups/123/posts/777?comment_id=5"))
	p2 := uidHash(uid2 + "|" + normalizeSocial("facebook.com/groups/123/posts/777"))
	if p1 != p2 {
		t.Errorf("same post through different url shapes should share point_uid: %q vs %q", p1, p2)
	}
	other := uidHash(uid1 + "|" + normalizeSocial("facebook.com/groups/123/posts/778"))
	if p1 == other {
		t.Errorf("different posts must not share point_uid")
	}
}

func TestCanonicalStoreURL(t *testing.T) {
	cases := []struct {
		in, want string
		ok       bool
	}{
		// The operator's paste variants all collapse to the canonical store form.
		{"https://www.facebook.com/groups/nhacuamy", "https://www.facebook.com/groups/nhacuamy", true},
		{"https://www.facebook.com/groups/nhacuamy/", "https://www.facebook.com/groups/nhacuamy", true},
		{"https://www.facebook.com/groups/nhacuamy?abd=xqj&nsfn=ksdj", "https://www.facebook.com/groups/nhacuamy", true},
		{"https://m.facebook.com/groups/nhacuamy/about?ref=share", "https://www.facebook.com/groups/nhacuamy", true},
		{"facebook.com/groups/NhaCuaMy", "https://www.facebook.com/groups/nhacuamy", true},
		// Identity params survive on social hosts.
		{"https://www.facebook.com/profile.php?id=100001&fbclid=XYZ", "https://www.facebook.com/profile.php?id=100001", true},
		// Arbitrary websites: conservative — path case kept, unknown params kept,
		// tracking params/fragment/trailing slash dropped.
		{"https://Example.com/Contact-Us/?utm_source=x&page=2#top", "https://example.com/Contact-Us?page=2", true},
		// Opaque redirector: refuse to store.
		{"https://l.facebook.com/l.php?h=AT999", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := canonicalStoreURL(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("canonicalStoreURL(%q) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
	// Redirector WITH destination stores the destination's clean form.
	got, ok := canonicalStoreURL("https://l.facebook.com/l.php?u=https%3A%2F%2Fexample.com%2Fa%2F&h=AT1")
	if !ok || got != "https://example.com/a" {
		t.Errorf("redirector destination store form = (%q, %v)", got, ok)
	}
}

func TestNormalizeCLI(t *testing.T) {
	code := 0
	out := captureStdout(t, func() {
		code = runSourceRegistryCLI([]string{"normalize",
			"--url", "https://www.facebook.com/groups/nhacuamy?abd=xqj&nsfn=ksdj",
			"--url", "https://l.facebook.com/l.php?h=opaque"})
	})
	if code != 0 {
		t.Fatalf("normalize exited %d", code)
	}
	var res map[string]any
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("bad JSON: %v\n%s", err, out)
	}
	rows := mList(res, "normalized")
	first := rows[0].(map[string]any)
	if first["clean_url"] != "https://www.facebook.com/groups/nhacuamy" || first["changed"] != true {
		t.Fatalf("dirty group url not cleaned: %v", first)
	}
	second := rows[1].(map[string]any)
	if second["ok"] != false {
		t.Fatalf("opaque redirector must be refused for storage: %v", second)
	}
}
