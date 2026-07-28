package main

// crm_xval_test.go — behavioral scenario tests for the crm-store and
// import-leads CLIs. History: these began as golden cross-validation against
// the retired Python implementation (every expectation below was verified
// byte/semantics-equal against outreach/tools/crm_store.py before the
// retirement); they now assert those verified outcomes directly.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type xstep struct {
	FakeNow string
	Argv    []string
}

type xresult struct {
	Code   int
	Stdout string
	Stderr string
}

// runGoStep executes one crm-store CLI invocation in-process under the
// injected test clock, capturing stdout/stderr.
func runGoStep(t *testing.T, s xstep) xresult {
	t.Helper()
	return runCLIStep(t, s, runCrmStoreCLI)
}

func runCLIStep(t *testing.T, s xstep, fn func([]string) int) xresult {
	t.Helper()
	t.Setenv("OUTREACHCRM_TEST_MODE", "1")
	t.Setenv("OUTREACHCRM_FAKE_NOW", s.FakeNow)
	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr
	code := fn(s.Argv)
	os.Stdout, os.Stderr = oldOut, oldErr
	wOut.Close()
	wErr.Close()
	outB, _ := io.ReadAll(rOut)
	errB, _ := io.ReadAll(rErr)
	return xresult{code, string(outB), string(errB)}
}

func parseOut(t *testing.T, r xresult) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(r.Stdout), &m); err != nil {
		t.Fatalf("stdout not a JSON object: %v\n%s", err, r.Stdout)
	}
	return m
}

func parseOutList(t *testing.T, r xresult) []map[string]any {
	t.Helper()
	var l []any
	if err := json.Unmarshal([]byte(r.Stdout), &l); err != nil {
		t.Fatalf("stdout not a JSON list: %v\n%s", err, r.Stdout)
	}
	return mapsOf(l)
}

func writeFixture(t *testing.T, ws string) {
	t.Helper()
	mustWrite := func(rel, body string) {
		p := filepath.Join(ws, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("sendboxes/sendboxes.json", `{"sendboxes": [
	  {"slug": "sb-a", "email": "a@gmail.com", "domain": "gmail.com", "quota_today": 40, "status": "healthy", "imap_uid_cursor": 0},
	  {"slug": "sb-b", "email": "b@gmail.com", "domain": "gmail.com", "quota_today": 40, "status": "needs_reauth", "imap_uid_cursor": 0}
	]}`)
	// a prior send at step 1 makes c_lead2 bump-due later
	mustWrite("campaigns/demo/sent/2026-07/sent_log.jsonl",
		`{"lead_id": "c_lead2", "campaign": "demo", "step": 1, "sent_at": "2026-07-10T09:00:00Z", "sendbox": "sb-a", "rfc_message_id": "<m1@x>"}`+"\n")
}

func TestCrmStoreScenario(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dcp")
	ws := filepath.Join(root, "clients", "leadup", "video_us", "outreach")

	dossier := `{"identity": {"still_active": "confirmed", "current_company": "KW", "channels_found": {"emails": ["extra@kw.com"]}},
	 "hooks": [
	   {"type": "new_listing", "summary": "listed 123 Main St", "evidence_url": "https://z/1", "observed_date": "2026-07-14", "confidence": 0.9, "analysis": {"sensitivity": "public_business"}},
	   {"type": "award", "summary": "personal award", "evidence_url": "https://z/2", "confidence": 0.8, "analysis": {"sensitivity": "personal"}},
	   {"type": "bad", "summary": "no url", "evidence_url": "N/A", "confidence": 0.5}
	 ],
	 "writing_brief": {"one_liner": "top KW agent", "personalization_confidence": 0.85, "do_not_mention": ["kids"]}}`

	run := func(now string, argv ...string) xresult {
		return runGoStep(t, xstep{FakeNow: now, Argv: argv})
	}
	mustOK := func(r xresult) map[string]any {
		t.Helper()
		if r.Code != 0 {
			t.Fatalf("exit %d: %s%s", r.Code, r.Stdout, r.Stderr)
		}
		return parseOut(t, r)
	}

	mustOK(run("2026-07-19T10:00:00Z", "--pipeline", root, "--client", "leadup",
		"--business", "video", "--location", "us", "init-client"))
	writeFixture(t, ws)

	out := mustOK(run("2026-07-19T10:01:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"id": "c_lead1", "name": {"full": "Susan Vo"}, "identities": {"emails": [{"address": "Susan@KW.com", "is_primary": true}], "phones": [{"number": "(415) 555-0101"}]}}`))
	if out["lead_id"] != "c_lead1" || out["outcome"] != "created" {
		t.Fatalf("lead1 add: %v", out)
	}
	mustOK(run("2026-07-19T10:02:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"id": "c_lead2", "name": {"full": "Binh Tran"}, "identities": {"emails": [{"address": "binh@remax.com", "is_primary": true}]}}`))
	out = mustOK(run("2026-07-19T10:03:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"name": {"full": "Dup Susan"}, "identities": {"emails": [{"address": "susan@kw.com"}]}}`))
	if out["lead_id"] != "c_lead1" || out["outcome"] != "matched" {
		t.Fatalf("email dedupe failed: %v", out)
	}

	list := parseOutList(t, run("2026-07-19T10:04:00Z", "--client-dir", ws, "contact", "list",
		"--where", "lifecycle_stage,=,lead"))
	if len(list) != 2 {
		t.Fatalf("contact list = %d, want 2", len(list))
	}

	mustOK(run("2026-07-19T10:05:00Z", "--client-dir", ws, "segment", "set", "--json",
		`{"id": "all", "name": "all", "where": [["lifecycle_stage", "=", "lead"]]}`))
	cfg := mustOK(run("2026-07-19T10:06:00Z", "--client-dir", ws, "campaign", "create",
		"--slug", "demo", "--json", `{"audience": {"segment": "all"}, "sendboxes": ["sb-a", "sb-b"], "daily_quota": 10}`))
	if mInt(cfg, "daily_quota", 0) != 10 || mStr(cfg, "approval_mode") != "manual_all" {
		t.Fatalf("campaign cfg: %v", cfg)
	}

	q := mustOK(run("2026-07-19T10:07:00Z", "--client-dir", ws, "campaign", "queue", "--slug", "demo"))
	if mInt(q, "queued", -1) != 1 || mInt(mMap(q, "skipped"), "already_in_campaign", -1) != 1 {
		t.Fatalf("queue guards: %v", q) // c_lead2 already sent in demo -> skipped
	}

	st := mustOK(run("2026-07-19T10:08:00Z", "--client-dir", ws, "enrich", "status", "--contact", "c_lead1"))
	if st["needs"] != "enrich" || st["reason"] != "identity_stale_or_missing" {
		t.Fatalf("enrich status: %v", st)
	}
	ew := mustOK(run("2026-07-19T10:10:00Z", "--client-dir", ws, "enrich", "write",
		"--contact", "c_lead1", "--campaign", "demo", "--json", dossier))
	// personal hook -> do_not_mention; bad evidence_url dropped with problem noted
	if mInt(ew, "usable_hooks", -1) != 1 || mStr(ew, "confidence_band") != "high" ||
		mInt(ew, "do_not_mention", -1) != 2 || len(mList(ew, "problems")) != 1 {
		t.Fatalf("enrich write: %v", ew)
	}

	dw := mustOK(run("2026-07-19T10:12:00Z", "--client-dir", ws, "draft", "write",
		"--contact", "c_lead1", "--campaign", "demo", "--json",
		`{"step": 1, "subject": "Idea for 123 Main St", "body_text": "Hi Susan...", "hooks_used": [{"type": "new_listing", "evidence_url": "https://z/1"}]}`))
	if mStr(dw, "sendbox") != "sb-a" || mStr(dw, "confidence_band") != "high" || len(mList(dw, "warnings")) != 0 {
		t.Fatalf("draft write: %v", dw) // sb-b is needs_reauth -> rotation picks sb-a
	}

	bad := run("2026-07-19T10:13:00Z", "--client-dir", ws, "draft", "write",
		"--contact", "c_lead1", "--campaign", "demo", "--json",
		`{"step": 1, "subject": "Re: hello", "body_text": "x", "hooks_used": [{"type": "new_listing", "evidence_url": "https://z/1"}]}`)
	if bad.Code == 0 || !strings.Contains(bad.Stderr, "step-1 subject must not begin with Re:/Fwd:") {
		t.Fatalf("subject gate: %d %s", bad.Code, bad.Stderr)
	}

	budget := mustOK(run("2026-07-19T10:14:00Z", "--client-dir", ws, "draft", "budget", "--campaign", "demo"))
	if mInt(budget, "used_today", -1) != 1 || mInt(budget, "remaining", -1) != 9 {
		t.Fatalf("budget: %v", budget)
	}

	rep := mustOK(run("2026-07-19T10:16:00Z", "--client-dir", ws, "approval-report"))
	if mInt(rep, "drafts", -1) != 1 || rep["html_rendered"] != true {
		t.Fatalf("approval report: %v", rep)
	}
	ap := mustOK(run("2026-07-19T10:17:00Z", "--client-dir", ws, "approve", "--json",
		`{"edit": [{"n": 1, "subject": "Better idea for 123 Main St"}], "approve": "1"}`))
	if len(mList(ap, "edited")) != 1 || len(mList(ap, "approved")) != 1 {
		t.Fatalf("approve: %v", ap)
	}
	appr := mapsOf(mList(ap, "approved"))[0]
	d, err := readJSONFile(mStr(appr, "path"))
	if err != nil || mStr(d, "subject") != "Better idea for 123 Main St" || mStr(d, "status") != "approved" {
		t.Fatalf("approved draft content: %v %v", err, d)
	}

	due := parseOutList(t, run("2026-07-19T10:18:00Z", "--client-dir", ws, "followups", "due", "--campaign", "demo"))
	if len(due) != 1 || mStr(due[0], "lead_id") != "c_lead2" || mInt(due[0], "next_step", 0) != 2 {
		t.Fatalf("followups due: %v", due)
	}

	ar := mustOK(run("2026-07-19T10:19:00Z", "--client-dir", ws, "apply-rules",
		"--event", "reply_positive", "--contact", "c_lead1", "--activity", "act_fixture01"))
	if len(mList(ar, "applied")) != 3 { // deal + task + freeze
		t.Fatalf("apply-rules r1: %v", ar)
	}
	ar2 := mustOK(run("2026-07-19T10:20:00Z", "--client-dir", ws, "apply-rules",
		"--event", "reply_positive", "--contact", "c_lead1", "--activity", "act_fixture01"))
	if len(mList(ar2, "applied")) != 0 {
		t.Fatalf("apply-rules must be idempotent: %v", ar2)
	}

	mustOK(run("2026-07-19T10:21:00Z", "--client-dir", ws, "suppress", "add",
		"--kind", "email", "--value", "Spam@X.com", "--reason", "unsubscribe", "--tag", "test_fixture"))
	sc := mustOK(run("2026-07-19T10:22:00Z", "--client-dir", ws, "suppress", "check", "--email", "spam@x.com"))
	if sc["suppressed"] != true {
		t.Fatalf("suppress check: %v", sc)
	}

	for i, wantGranted := range []bool{true, true, false} {
		r := mustOK(run(fmt.Sprintf("2026-07-19T10:2%d:00Z", 4+i), "--client-dir", ws,
			"reserve", "--sendbox", "sb-a", "--day", "2026-07-19", "--cap", "2"))
		if r["granted"] != wantGranted {
			t.Fatalf("reserve %d: %v", i, r)
		}
	}

	mg := mustOK(run("2026-07-19T10:27:00Z", "--client-dir", ws, "contact", "merge",
		"--loser", "c_lead2", "--winner", "c_lead1"))
	if mStr(mg, "id") != "c_lead1" {
		t.Fatalf("merge: %v", mg)
	}
	va := mustOK(run("2026-07-19T10:28:00Z", "--client-dir", ws, "validate", "--rebuild-index"))
	if mInt(va, "contacts", -1) != 2 || len(mList(va, "problems")) != 0 || va["index_rebuilt"] != true {
		t.Fatalf("validate: %v", va)
	}

	for _, cmd := range [][]string{{"today-view"}, {"kanban"}, {"weekly-report"}, {"monthly-report", "--month", "2026-07"}} {
		r := mustOK(run("2026-07-19T10:30:00Z", append([]string{"--client-dir", ws}, cmd...)...))
		if r["html_rendered"] != true {
			t.Fatalf("%v: %v", cmd, r)
		}
	}
}

func TestImportLeadsFlow(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dcp")
	ws := filepath.Join(root, "clients", "leadup", "video_us", "outreach")

	csvBody := "Full Name,Email,Cell Phone,Office Name,Website\n" +
		"Susan Vo,susan@kw.com,(415) 555-0101,KW Bay Area,https://susanvo.com\n" +
		"Binh Tran,BINH@remax.com,,RE/MAX,\n" +
		"No Identity,,,,\n" +
		"Susan Dup,susan@kw.com,,KW,\n" +
		"Spam Guy,spam@x.com,,X,\n"
	csvPath := filepath.Join(t.TempDir(), "leads.csv")
	if err := os.WriteFile(csvPath, []byte(csvBody), 0o644); err != nil {
		t.Fatal(err)
	}

	if r := runGoStep(t, xstep{"2026-07-19T11:00:00Z", []string{"--pipeline", root, "--client", "leadup",
		"--business", "video", "--location", "us", "init-client"}}); r.Code != 0 {
		t.Fatal(r.Stderr)
	}
	if r := runGoStep(t, xstep{"2026-07-19T11:01:00Z", []string{"--client-dir", ws, "suppress", "add",
		"--kind", "email", "--value", "spam@x.com", "--reason", "unsubscribe"}}); r.Code != 0 {
		t.Fatal(r.Stderr)
	}

	leads := func(now string, argv ...string) xresult {
		return runCLIStep(t, xstep{now, argv}, runImportLeadsCLI)
	}
	insp := parseOut(t, leads("2026-07-19T11:02:00Z", "inspect", "--file", csvPath))
	if mInt(insp, "total_rows", -1) != 5 || mStr(mMap(insp, "proposed_mapping"), "email") != "Email" {
		t.Fatalf("inspect: %v", insp)
	}

	imp := parseOut(t, leads("2026-07-19T11:03:00Z", "import", "--client-dir", ws,
		"--file", csvPath, "--list-slug", "realtors", "--no-mx-check"))
	m := mMap(imp, "manifest")
	// Verified against the Python implementation before its retirement:
	// name-only row imports as a fragment; the dup matches; spam suppressed.
	if imp["skipped"] != false || mInt(m, "contacts_created", -1) != 3 ||
		mInt(m, "contacts_matched_existing", -1) != 1 || mInt(m, "suppressed_at_import", -1) != 1 {
		t.Fatalf("import: %v", imp)
	}

	imp2 := parseOut(t, leads("2026-07-19T11:04:00Z", "import", "--client-dir", ws,
		"--file", csvPath, "--list-slug", "realtors", "--no-mx-check"))
	if imp2["skipped"] != true {
		t.Fatalf("idempotency: %v", imp2)
	}
}

// A rich operator list carries research clues in columns the mapping does not
// claim (NPN, license, years in practice...). Import must KEEP them on the
// contact (custom_fields) and in the list audit, so enrichment can use them as
// starting points instead of re-researching from scratch.
func TestImportKeepsUnmappedColumns(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dcp")
	ws := filepath.Join(root, "clients", "leadup", "video_us", "outreach")
	// "City" is a recognized column (maps to the canonical slot); the rest are
	// real-world names the auto-mapper does NOT know and used to be dropped.
	csvBody := "Full Name,Email,Agency,NPN,License Type,Years In Practice,City,Bus State,Notes\n" +
		"Pat Nguyen,pat@ins.com,Statewide Ins,8123456,P&C,12,Garden Grove,CA,referred by AL\n"
	csvPath := filepath.Join(t.TempDir(), "rich.csv")
	if err := os.WriteFile(csvPath, []byte(csvBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := runGoStep(t, xstep{"2026-07-21T09:00:00Z", []string{"--pipeline", root, "--client", "leadup",
		"--business", "video", "--location", "us", "init-client"}}); r.Code != 0 {
		t.Fatal(r.Stderr)
	}
	leads := func(now string, argv ...string) xresult {
		return runCLIStep(t, xstep{now, argv}, runImportLeadsCLI)
	}
	imp := parseOut(t, leads("2026-07-21T09:01:00Z", "import", "--client-dir", ws,
		"--file", csvPath, "--list-slug", "insurance", "--no-mx-check"))
	if mInt(mMap(imp, "manifest"), "contacts_created", -1) != 1 {
		t.Fatalf("import: %v", imp)
	}

	// the contact keeps every unmapped column as a clue
	store := newCrmStore(ws)
	lead := store.a.findByIdentity("email", "pat@ins.com")
	if lead == "" {
		t.Fatal("contact not created")
	}
	cf := mMap(store.getContact(lead), "custom_fields")
	for k, want := range map[string]string{
		"NPN": "8123456", "License Type": "P&C", "Years In Practice": "12",
		"Notes": "referred by AL", "Agency": "Statewide Ins",
	} {
		if mStr(cf, k) != want {
			t.Errorf("custom_fields[%q] = %q, want %q (unmapped clue dropped)", k, mStr(cf, k), want)
		}
	}
	// mapped fields still win their canonical slots
	if mStr(cf, "city") != "Garden Grove" {
		t.Errorf("mapped city lost: %v", cf)
	}
	// an unrecognized column name (the shape real operator lists use) survives
	// verbatim instead of being silently dropped
	if mStr(cf, "Bus State") != "CA" {
		t.Errorf("unrecognized column dropped: %v", cf)
	}

	// the list audit row carries them too (same `norm`), so nothing is lost
	rows := readJSONLines(filepath.Join(ws, "lists", "insurance", "leads.jsonl"))
	if len(rows) != 1 || mStr(mMap(mMap(rows[0], "normalized"), "extra"), "NPN") != "8123456" {
		t.Fatalf("audit row missing extra clues: %v", rows)
	}
}

func TestSeedLeadFlow(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dcp")
	ws := filepath.Join(root, "clients", "leadup", "video_us", "outreach")
	if r := runGoStep(t, xstep{"2026-07-19T13:00:00Z", []string{"--pipeline", root, "--client", "leadup",
		"--business", "video", "--location", "us", "init-client"}}); r.Code != 0 {
		t.Fatal(r.Stderr)
	}
	leads := func(now string, argv ...string) xresult {
		return runCLIStep(t, xstep{now, argv}, runImportLeadsCLI)
	}

	// CSV: reel-only, youtube-watch-only, profile-only (in the generic Link col),
	// a reel pasted under the Facebook column, and a name-only fragment
	csvBody := "Name,Email,Facebook,Reel,Link\n" +
		",,https://www.facebook.com/reel/123456789,,\n" +
		",,,https://youtube.com/watch?v=abc123xyz,\n" +
		",,,,https://www.facebook.com/susan.vo.realtor\n" +
		",,https://facebook.com/reel/999888777,,\n" +
		"Nguyen Van Hiem Realty,,,,\n"
	csvPath := filepath.Join(t.TempDir(), "clues.csv")
	os.WriteFile(csvPath, []byte(csvBody), 0o644)

	imp := parseOut(t, leads("2026-07-19T13:01:00Z", "import", "--client-dir", ws,
		"--file", csvPath, "--list-slug", "clues", "--no-mx-check"))
	m := mMap(imp, "manifest")
	if mInt(m, "contacts_created", -1) != 5 || mInt(m, "rows_skipped", -1) != 0 {
		t.Fatalf("seed import: %v", m)
	}

	// same reel again -> dedupe hit via the seed identity index
	csv2 := filepath.Join(t.TempDir(), "again.csv")
	os.WriteFile(csv2, []byte("Reel\nhttps://facebook.com/reel/123456789\n"), 0o644)
	imp2 := parseOut(t, leads("2026-07-19T13:02:00Z", "import", "--client-dir", ws,
		"--file", csv2, "--list-slug", "clues2", "--no-mx-check"))
	m2 := mMap(imp2, "manifest")
	if mInt(m2, "contacts_created", -1) != 0 || mInt(m2, "contacts_matched_existing", -1) != 1 {
		t.Fatalf("seed dedupe: %v", m2)
	}

	// find the reel-only contact and walk the resolution ladder
	rows := parseOutList(t, runGoStep(t, xstep{"2026-07-19T13:03:00Z",
		[]string{"--client-dir", ws, "contact", "list"}}))
	var reelLead, nameLead string
	for _, ct := range rows {
		seeds := mapsOf(mList(mMap(ct, "identities"), "seeds"))
		for _, sd := range seeds {
			if strings.Contains(mStr(sd, "url"), "reel/123456789") {
				reelLead = mStr(ct, "id")
				if mStr(sd, "kind") != "reel" || mStr(sd, "platform") != "facebook" ||
					mStr(sd, "status") != "unresolved" || mStr(sd, "source") != "import" {
					t.Fatalf("seed entry shape: %v", sd)
				}
			}
		}
		if mStr(mMap(ct, "name"), "full") == "Nguyen Van Hiem Realty" {
			nameLead = mStr(ct, "id")
		}
	}
	if reelLead == "" || nameLead == "" {
		t.Fatalf("leads not found: reel=%q name=%q", reelLead, nameLead)
	}

	st := parseOut(t, runGoStep(t, xstep{"2026-07-19T13:04:00Z",
		[]string{"--client-dir", ws, "enrich", "status", "--contact", reelLead}}))
	if st["needs"] != "enrich" || st["reason"] != "seed_unresolved" {
		t.Fatalf("reel lead status: %v", st)
	}
	st = parseOut(t, runGoStep(t, xstep{"2026-07-19T13:04:30Z",
		[]string{"--client-dir", ws, "enrich", "status", "--contact", nameLead}}))
	if st["reason"] != "name_only_fragment" {
		t.Fatalf("name-only status: %v", st)
	}

	// enrichment resolves the origin: profile + email found -> canonical write-back
	ew := parseOut(t, runGoStep(t, xstep{"2026-07-19T13:05:00Z",
		[]string{"--client-dir", ws, "enrich", "write", "--contact", reelLead, "--json",
			`{"identity": {"still_active": "confirmed",
			   "channels_found": {"emails": ["susan@kw.com"],
			                      "profiles": {"facebook": "https://facebook.com/susan.vo.99"}}},
			  "hooks": [], "writing_brief": {"personalization_confidence": 0.5}}`}}))
	if mInt(ew, "usable_hooks", -1) != 0 {
		t.Fatalf("enrich write: %v", ew)
	}
	ct := parseOut(t, runGoStep(t, xstep{"2026-07-19T13:06:00Z",
		[]string{"--client-dir", ws, "contact", "get", "--id", reelLead}}))
	ids := mMap(ct, "identities")
	if mStr(mMap(ids, "socials"), "facebook") != "https://facebook.com/susan.vo.99" {
		t.Fatalf("found profile not canonical: %v", ids["socials"])
	}
	gotEmail := false
	for _, e := range mapsOf(mList(ids, "emails")) {
		if mStr(e, "address") == "susan@kw.com" {
			gotEmail = true
		}
	}
	if !gotEmail {
		t.Fatalf("found email not written: %v", ids["emails"])
	}
	seeds := mapsOf(mList(ids, "seeds"))
	if len(seeds) != 1 || mStr(seeds[0], "status") != "resolved" ||
		mStr(seeds[0], "resolved_profile") != "https://facebook.com/susan.vo.99" {
		t.Fatalf("seed not resolved: %v", seeds)
	}
	// resolved profile is now a dedupe identity: importing that profile matches
	csv3 := filepath.Join(t.TempDir(), "profile.csv")
	os.WriteFile(csv3, []byte("Profile\nhttps://facebook.com/susan.vo.99\n"), 0o644)
	imp3 := parseOut(t, leads("2026-07-19T13:07:00Z", "import", "--client-dir", ws,
		"--file", csv3, "--list-slug", "clues3", "--no-mx-check"))
	if mInt(mMap(imp3, "manifest"), "contacts_matched_existing", -1) != 1 {
		t.Fatalf("resolved profile must dedupe: %v", imp3)
	}
}

func TestSeedTxtClassification(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dcp")
	ws := filepath.Join(root, "clients", "leadup", "video_us", "outreach")
	if r := runGoStep(t, xstep{"2026-07-19T13:10:00Z", []string{"--pipeline", root, "--client", "leadup",
		"--business", "video", "--location", "us", "init-client"}}); r.Code != 0 {
		t.Fatal(r.Stderr)
	}
	txt := "susan@kw.com\nhttps://facebook.com/reel/5551112223\n(415) 555-0101\nHiem Nguyen Realty\n"
	p := filepath.Join(t.TempDir(), "mixed.txt")
	os.WriteFile(p, []byte(txt), 0o644)
	imp := parseOut(t, runCLIStep(t, xstep{"2026-07-19T13:11:00Z",
		[]string{"import", "--client-dir", ws, "--file", p, "--list-slug", "mixed", "--no-mx-check"}}, runImportLeadsCLI))
	m := mMap(imp, "manifest")
	if mInt(m, "contacts_created", -1) != 4 || mInt(m, "rows_skipped", -1) != 0 {
		t.Fatalf("txt classify import: %v", m)
	}
	rows := parseOutList(t, runGoStep(t, xstep{"2026-07-19T13:12:00Z",
		[]string{"--client-dir", ws, "contact", "list"}}))
	var haveEmail, haveSeed, havePhone, haveName bool
	for _, ct := range rows {
		ids := mMap(ct, "identities")
		for _, e := range mapsOf(mList(ids, "emails")) {
			if mStr(e, "address") == "susan@kw.com" {
				haveEmail = true
			}
		}
		for _, ph := range mapsOf(mList(ids, "phones")) {
			if mStr(ph, "number") == "+14155550101" {
				havePhone = true
			}
		}
		for _, sd := range mapsOf(mList(ids, "seeds")) {
			if strings.Contains(mStr(sd, "url"), "reel/5551112223") && mStr(sd, "kind") == "reel" {
				haveSeed = true
			}
		}
		if mStr(mMap(ct, "name"), "full") == "Hiem Nguyen Realty" {
			haveName = true
		}
	}
	if !haveEmail || !haveSeed || !havePhone || !haveName {
		t.Fatalf("txt classification: email=%v seed=%v phone=%v name=%v", haveEmail, haveSeed, havePhone, haveName)
	}
}

func TestEnrichBandGates(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dcp")
	ws := filepath.Join(root, "clients", "leadup", "video_us", "outreach")
	if r := runGoStep(t, xstep{"2026-07-20T08:00:00Z", []string{"--pipeline", root, "--client", "leadup",
		"--business", "video", "--location", "us", "init-client"}}); r.Code != 0 {
		t.Fatal(r.Stderr)
	}
	add := func(id, extra string) {
		body := `{"id": "` + id + `", "name": {"full": "Test ` + id + `"}, "identities": {"emails": [{"address": "` + id + `@law.com", "is_primary": true}]` + extra + `}}`
		if r := runGoStep(t, xstep{"2026-07-20T08:01:00Z", []string{"--client-dir", ws, "contact", "add", "--json", body}}); r.Code != 0 {
			t.Fatal(r.Stderr)
		}
	}
	add("c_web", "")
	add("c_fb", `, "socials": {"facebook": "https://facebook.com/attorney.jen"}`)
	add("c_fbread", `, "socials": {"facebook": "https://facebook.com/attorney.tom"}`)
	add("c_months", "")

	enrich := func(id, dossier string) map[string]any {
		r := runGoStep(t, xstep{"2026-07-20T08:05:00Z", []string{"--client-dir", ws,
			"enrich", "write", "--contact", id, "--json", dossier}})
		if r.Code != 0 {
			t.Fatal(r.Stderr)
		}
		return parseOut(t, r)
	}
	joined := func(res map[string]any) string {
		var sb strings.Builder
		for _, p := range mList(res, "problems") {
			sb.WriteString(fmt.Sprint(p) + " | ")
		}
		return sb.String()
	}

	// 1. website-only signal, self-claimed 0.9 -> capped, and the read-date stamp is called out
	res := enrich("c_web", `{"identity": {"still_active": "confirmed"},
	  "hooks": [{"type": "website_update", "summary": "site touts bilingual practice", "evidence_url": "https://dolaw.example.com/about", "observed_date": "2026-07-20", "confidence": 0.9}],
	  "writing_brief": {"personalization_confidence": 0.9}}`)
	if mStr(res, "confidence_band") != "review_carefully" {
		t.Fatalf("website-only must be capped: %v", res)
	}
	if !strings.Contains(joined(res), "band capped") || !strings.Contains(joined(res), "when the page was READ") {
		t.Fatalf("cap reasons missing: %s", joined(res))
	}

	// 2. dated non-fb hook but the facebook profile on file was never read -> capped
	res = enrich("c_fb", `{"identity": {"still_active": "confirmed"},
	  "hooks": [{"type": "new_listing", "summary": "press mention", "evidence_url": "https://news.example.com/a", "observed_date": "2026-07-15", "confidence": 0.9}],
	  "writing_brief": {"personalization_confidence": 0.9}}`)
	if mStr(res, "confidence_band") != "review_carefully" || !strings.Contains(joined(res), "facebook-sourced hook") {
		t.Fatalf("fb-unread must be capped: %v %s", res, joined(res))
	}

	// 3. facebook actually read (dated fb hook) -> high stands
	res = enrich("c_fbread", `{"identity": {"still_active": "confirmed"},
	  "hooks": [{"type": "social_post", "summary": "posted client Q&A reel", "evidence_url": "https://facebook.com/attorney.tom/videos/123", "observed_date": "2026-07-18", "confidence": 0.9}],
	  "writing_brief": {"personalization_confidence": 0.9}}`)
	if mStr(res, "confidence_band") != "high" {
		t.Fatalf("fb-read high must stand: %v %s", res, joined(res))
	}

	// 4. a hook older than the recency window (360d) alone cannot hold high
	res = enrich("c_web", `{"identity": {"still_active": "confirmed"},
	  "hooks": [{"type": "award", "summary": "old award", "evidence_url": "https://bar.example.com/award", "observed_date": "2024-11-05", "confidence": 0.9}],
	  "writing_brief": {"personalization_confidence": 0.9}}`)
	if mStr(res, "confidence_band") != "review_carefully" {
		t.Fatalf("stale-only must be capped: %v", res)
	}

	// 5. a hook INSIDE the year-long window holds high (this is what the old
	// 60-day rule was killing: a real signal a few months back)
	res = enrich("c_months", `{"identity": {"still_active": "confirmed"},
	  "hooks": [{"type": "social_post", "summary": "opened a second office", "evidence_url": "https://facebook.com/agent.x/posts/9", "observed_date": "2026-03-02", "confidence": 0.9}],
	  "writing_brief": {"personalization_confidence": 0.9}}`)
	if mStr(res, "confidence_band") != "high" {
		t.Fatalf("a signal inside the recency window must hold high: %v %s", res, joined(res))
	}
}

// TestOperatorSeedExemptsRecency: a reel/post the OPERATOR saved is pre-qualified
// by that human act — its age is irrelevant. A hook the SYSTEM found is not.
func TestOperatorSeedExemptsRecency(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dcp")
	ws := filepath.Join(root, "clients", "leadup", "video_us", "outreach")
	run := func(now string, argv ...string) xresult {
		return runGoStep(t, xstep{FakeNow: now, Argv: argv})
	}
	mustOK := func(r xresult) map[string]any {
		t.Helper()
		if r.Code != 0 {
			t.Fatalf("exit %d: %s%s", r.Code, r.Stdout, r.Stderr)
		}
		return parseOut(t, r)
	}
	mustOK(run("2026-07-27T10:00:00Z", "--pipeline", root, "--client", "leadup",
		"--business", "video", "--location", "us", "init-client"))

	// the operator pasted this reel link; the store stamps source: import
	mustOK(run("2026-07-27T10:01:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"id":"c_saved","identities":{"emails":[{"address":"a@x.com","is_primary":true}],
		  "seeds":[{"url":"https://www.facebook.com/reel/777","kind":"reel","platform":"facebook"}]}}`))
	doc, _ := readJSONFile(filepath.Join(ws, "crm", "contacts", "c_saved.json"))
	sd := mapsOf(mList(mMap(doc, "identities"), "seeds"))
	if len(sd) != 1 || mStr(sd[0], "source") != "import" || mStr(sd[0], "status") != "unresolved" {
		t.Fatalf("store must stamp seed provenance itself: %v", sd)
	}

	// a hook citing THAT saved reel, two years old, still supports high
	res := mustOK(run("2026-07-27T10:02:00Z", "--client-dir", ws, "enrich", "write", "--contact", "c_saved", "--json",
		`{"identity":{"still_active":"confirmed"},
		  "hooks":[{"type":"social_post","summary":"walkthrough of a renovated Whittier home","evidence_url":"https://www.facebook.com/reel/777","observed_date":"2024-06-01","confidence":0.9}],
		  "writing_brief":{"personalization_confidence":0.9}}`))
	if mStr(res, "confidence_band") != "high" {
		t.Fatalf("a hook on an operator-saved seed is exempt from recency: %v", res)
	}

	// but an equally old hook the system found elsewhere is NOT exempt
	mustOK(run("2026-07-27T10:03:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"id":"c_found","identities":{"emails":[{"address":"b@x.com","is_primary":true}]}}`))
	res = mustOK(run("2026-07-27T10:04:00Z", "--client-dir", ws, "enrich", "write", "--contact", "c_found", "--json",
		`{"identity":{"still_active":"confirmed"},
		  "hooks":[{"type":"social_post","summary":"old post we dug up","evidence_url":"https://www.facebook.com/reel/999","observed_date":"2024-06-01","confidence":0.9}],
		  "writing_brief":{"personalization_confidence":0.9}}`))
	if mStr(res, "confidence_band") != "review_carefully" {
		t.Fatalf("a system-found old hook must still be capped: %v", res)
	}
}

// TestCampaignUpdateAndPause covers the operator edit whitelist and the paused
// gates: queue no-ops, draft write refuses, resume restores everything.
func TestCampaignUpdateAndPause(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dcp")
	ws := filepath.Join(root, "clients", "leadup", "video_us", "outreach")
	run := func(now string, argv ...string) xresult {
		return runGoStep(t, xstep{FakeNow: now, Argv: argv})
	}
	mustOK := func(r xresult) map[string]any {
		t.Helper()
		if r.Code != 0 {
			t.Fatalf("exit %d: %s%s", r.Code, r.Stdout, r.Stderr)
		}
		return parseOut(t, r)
	}
	mustOK(run("2026-07-20T10:00:00Z", "--pipeline", root, "--client", "leadup",
		"--business", "video", "--location", "us", "init-client"))
	writeFixture(t, ws)
	mustOK(run("2026-07-20T10:01:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"id": "c_p1", "name": {"full": "Pat Doe"}, "identities": {"emails": [{"address": "pat@x.com", "is_primary": true}]}}`))
	mustOK(run("2026-07-20T10:02:00Z", "--client-dir", ws, "segment", "set", "--json",
		`{"id": "all", "name": "all", "where": [["lifecycle_stage", "=", "lead"]]}`))
	mustOK(run("2026-07-20T10:03:00Z", "--client-dir", ws, "campaign", "create",
		"--slug", "demo", "--json", `{"audience": {"segment": "all"}, "sendboxes": ["sb-a"], "daily_quota": 10}`))

	// 1. whitelisted operator edit: goal fields + companion_doc + quota
	up := mustOK(run("2026-07-20T10:04:00Z", "--client-dir", ws, "campaign", "update",
		"--slug", "demo", "--json",
		`{"daily_quota": 25, "goal": {"goal_type": "book_meeting", "objective": "book 5 calls",
		  "cta": {"text": "Worth a look?"},
		  "companion_doc": {"instructions": "use https://leadup.example/demo for every lead",
		    "on_fail": "default_link", "default_link": "https://leadup.example/demo"}}}`))
	changed := fmt.Sprint(mList(up, "changed"))
	for _, want := range []string{"daily_quota", "goal.objective", "goal.companion_doc"} {
		if !strings.Contains(changed, want) {
			t.Fatalf("changed missing %s: %v", want, changed)
		}
	}
	got := mustOK(run("2026-07-20T10:05:00Z", "--client-dir", ws, "campaign", "get", "--slug", "demo"))
	goal := mMap(got, "goal")
	if mInt(got, "daily_quota", 0) != 25 || mStr(goal, "objective") != "book 5 calls" ||
		mStr(mMap(goal, "companion_doc"), "on_fail") != "default_link" {
		t.Fatalf("update not persisted: %v", got)
	}

	// 2. non-whitelisted key rejected loudly
	bad := run("2026-07-20T10:06:00Z", "--client-dir", ws, "campaign", "update",
		"--slug", "demo", "--json", `{"sendboxes": ["sb-evil"]}`)
	if bad.Code == 0 || !strings.Contains(bad.Stderr, "not operator-editable") {
		t.Fatalf("unknown key must be rejected: %d %s", bad.Code, bad.Stderr)
	}
	// 3. invalid goal_type rejected
	bad = run("2026-07-20T10:07:00Z", "--client-dir", ws, "campaign", "update",
		"--slug", "demo", "--json", `{"goal": {"goal_type": "world_peace"}}`)
	if bad.Code == 0 || !strings.Contains(bad.Stderr, "goal_type") {
		t.Fatalf("bad goal_type must be rejected: %d %s", bad.Code, bad.Stderr)
	}
	// 4. default_link fallback needs a valid URL
	bad = run("2026-07-20T10:08:00Z", "--client-dir", ws, "campaign", "update",
		"--slug", "demo", "--json",
		`{"goal": {"companion_doc": {"instructions": "x", "on_fail": "default_link", "default_link": "not-a-url"}}}`)
	if bad.Code == 0 {
		t.Fatalf("bad default_link must be rejected: %s", bad.Stdout)
	}

	// 5. pause: queue no-ops, draft write refuses with campaign_paused
	mustOK(run("2026-07-20T10:09:00Z", "--client-dir", ws, "campaign", "update",
		"--slug", "demo", "--json", `{"status": "paused"}`))
	q := mustOK(run("2026-07-20T10:10:00Z", "--client-dir", ws, "campaign", "queue", "--slug", "demo"))
	if mInt(q, "queued", -1) != 0 || mInt(mMap(q, "skipped"), "campaign_paused", -1) != 1 {
		t.Fatalf("paused queue must no-op: %v", q)
	}
	dw := run("2026-07-20T10:11:00Z", "--client-dir", ws, "draft", "write",
		"--contact", "c_p1", "--campaign", "demo", "--json",
		`{"step": 1, "subject": "Hi", "body_text": "x", "hooks_used": [{"type": "new_listing", "evidence_url": "https://z/1"}]}`)
	if dw.Code == 0 || !strings.Contains(dw.Stderr, "campaign_paused") {
		t.Fatalf("paused draft write must refuse: %d %s", dw.Code, dw.Stderr)
	}

	// 6. resume restores queueing
	mustOK(run("2026-07-20T10:12:00Z", "--client-dir", ws, "campaign", "update",
		"--slug", "demo", "--json", `{"status": "active"}`))
	q = mustOK(run("2026-07-20T10:13:00Z", "--client-dir", ws, "campaign", "queue", "--slug", "demo"))
	if mInt(q, "queued", -1) != 1 {
		t.Fatalf("resume must queue again: %v", q)
	}
}

// TestConsolidationFlow covers the "many reels → one person" path: matched
// import rows keep their new data, enrich-time discovery auto-merges safe
// pairs with a FULL union, conflicting pairs are flagged (never index-stolen)
// and held out of queues until resolved.
func TestConsolidationFlow(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dcp")
	ws := filepath.Join(root, "clients", "leadup", "video_us", "outreach")
	run := func(now string, argv ...string) xresult {
		return runGoStep(t, xstep{FakeNow: now, Argv: argv})
	}
	mustOK := func(r xresult) map[string]any {
		t.Helper()
		if r.Code != 0 {
			t.Fatalf("exit %d: %s%s", r.Code, r.Stdout, r.Stderr)
		}
		return parseOut(t, r)
	}
	readDoc := func(id string) map[string]any {
		t.Helper()
		m, err := readJSONFile(filepath.Join(ws, "crm", "contacts", id+".json"))
		if err != nil {
			t.Fatalf("read %s: %v", id, err)
		}
		return m
	}
	seedURLs := func(doc map[string]any) map[string]string {
		out := map[string]string{}
		for _, sd := range mapsOf(mList(mMap(doc, "identities"), "seeds")) {
			out[mStr(sd, "url")] = mStr(sd, "status")
		}
		return out
	}
	mustOK(run("2026-07-20T11:00:00Z", "--pipeline", root, "--client", "leadup",
		"--business", "video", "--location", "us", "init-client"))
	writeFixture(t, ws)

	// 1. matched import row keeps its NEW seed (no data loss on dedupe)
	a := mustOK(run("2026-07-20T11:01:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"id": "c_a", "name": {"full": "Susan Vo"}, "identities": {"emails": [{"address": "susan@x.com", "is_primary": true}]}}`))
	if a["outcome"] != "created" {
		t.Fatalf("A add: %v", a)
	}
	m := mustOK(run("2026-07-20T11:02:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"identities": {"emails": [{"address": "susan@x.com"}],
		  "seeds": [{"url": "https://www.facebook.com/reel/999", "kind": "reel", "platform": "facebook", "source": "import", "status": "unresolved"}]}}`))
	if m["outcome"] != "matched" || m["lead_id"] != "c_a" {
		t.Fatalf("matched row: %v", m)
	}
	if s := seedURLs(readDoc("c_a")); s["https://www.facebook.com/reel/999"] == "" {
		t.Fatalf("matched row's NEW seed lost: %v", s)
	}

	// 2. two reel fragments resolve to ONE profile -> auto-merge, full union
	mustOK(run("2026-07-20T11:03:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"id": "c_b", "identities": {"seeds": [{"url": "https://www.facebook.com/reel/111", "kind": "reel", "platform": "facebook", "source": "import", "status": "unresolved"}]}}`))
	mustOK(run("2026-07-20T11:04:00Z", "--client-dir", ws, "enrich", "write",
		"--contact", "c_b", "--campaign", "", "--json",
		`{"identity": {"still_active": "confirmed", "channels_found": {"profiles": {"facebook": "https://www.facebook.com/pro.susan"}}},
		  "hooks": [{"type": "new_listing", "summary": "reel one", "evidence_url": "https://www.facebook.com/reel/111", "observed_date": "2026-07-14", "confidence": 0.9}],
		  "writing_brief": {"personalization_confidence": 0.8}}`))
	mustOK(run("2026-07-20T11:05:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"id": "c_c", "identities": {"seeds": [{"url": "https://www.facebook.com/reel/222", "kind": "reel", "platform": "facebook", "source": "import", "status": "unresolved"}]}}`))
	ew := mustOK(run("2026-07-20T11:06:00Z", "--client-dir", ws, "enrich", "write",
		"--contact", "c_c", "--campaign", "", "--json",
		`{"identity": {"still_active": "confirmed", "channels_found": {"profiles": {"facebook": "https://www.facebook.com/pro.susan"}}},
		  "hooks": [{"type": "award", "summary": "reel two", "evidence_url": "https://www.facebook.com/reel/222", "observed_date": "2026-07-15", "confidence": 0.9}],
		  "writing_brief": {"personalization_confidence": 0.8}}`))
	if ew["lead_id"] != "c_b" {
		t.Fatalf("dossier must land on survivor: %v", ew)
	}
	cons := mapsOf(mList(ew, "consolidated"))
	if len(cons) != 1 || mStr(cons[0], "survivor") != "c_b" || mStr(cons[0], "merged") != "c_c" {
		t.Fatalf("consolidated event wrong: %v", ew["consolidated"])
	}
	b := readDoc("c_b")
	seeds := seedURLs(b)
	if seeds["https://www.facebook.com/reel/111"] != "resolved" || seeds["https://www.facebook.com/reel/222"] != "resolved" {
		t.Fatalf("union must keep BOTH reels resolved: %v", seeds)
	}
	hookURLs := map[string]bool{}
	for _, h := range mapsOf(mList(mMap(b, "enrichment"), "hooks")) {
		hookURLs[mStr(h, "evidence_url")] = true
	}
	if !hookURLs["https://www.facebook.com/reel/111"] || !hookURLs["https://www.facebook.com/reel/222"] {
		t.Fatalf("union must keep BOTH hooks: %v", hookURLs)
	}
	if mStr(mMap(readDoc("c_c"), "merge"), "merged_into") != "c_b" {
		t.Fatal("loser must be tombstoned into survivor")
	}
	if m := mustOK(run("2026-07-20T11:07:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"identities": {"seeds": [{"url": "https://www.facebook.com/reel/222", "kind": "reel", "platform": "facebook", "source": "import", "status": "unresolved"}]}}`)); m["lead_id"] != "c_b" {
		t.Fatalf("loser's seed must now resolve to survivor: %v", m)
	}

	// 3. conflict (two NAMED people share one page URL) -> flag, no merge, no theft
	mustOK(run("2026-07-20T11:08:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"id": "c_d", "name": {"full": "David Do"}, "identities": {"emails": [{"address": "david@x.com", "is_primary": true}]}}`))
	mustOK(run("2026-07-20T11:09:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"id": "c_e", "name": {"full": "Emma Vo"}, "identities": {"emails": [{"address": "emma@y.com", "is_primary": true}]}}`))
	mustOK(run("2026-07-20T11:10:00Z", "--client-dir", ws, "enrich", "write",
		"--contact", "c_e", "--campaign", "", "--json",
		`{"identity": {"still_active": "confirmed", "channels_found": {"profiles": {"facebook": "https://www.facebook.com/brokerage.page"}}},
		  "writing_brief": {"personalization_confidence": 0.5}}`))
	dw := mustOK(run("2026-07-20T11:11:00Z", "--client-dir", ws, "enrich", "write",
		"--contact", "c_d", "--campaign", "", "--json",
		`{"identity": {"still_active": "confirmed", "channels_found": {"profiles": {"facebook": "https://www.facebook.com/brokerage.page"}}},
		  "writing_brief": {"personalization_confidence": 0.5}}`))
	sus := mapsOf(mList(dw, "duplicate_suspected"))
	if len(sus) != 1 || mStr(sus[0], "other_id") != "c_e" {
		t.Fatalf("conflict must be flagged, not merged: %v", dw)
	}
	d := readDoc("c_d")
	if truthy(mMap(mMap(d, "identities"), "socials")["facebook"]) {
		t.Fatal("conflicting URL must be withheld from the losing write")
	}
	if len(mList(d, "duplicate_suspects")) != 1 || len(mList(readDoc("c_e"), "duplicate_suspects")) != 1 {
		t.Fatal("both records must carry the mutual flag")
	}
	if m := mustOK(run("2026-07-20T11:12:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"identities": {"socials": {"facebook": "https://www.facebook.com/brokerage.page"}}}`)); m["lead_id"] != "c_e" {
		t.Fatalf("index must still point at the original owner: %v", m)
	}

	// 4. suspects are held out of campaign queues; unsuspect releases them
	mustOK(run("2026-07-20T11:13:00Z", "--client-dir", ws, "segment", "set", "--json",
		`{"id": "all", "name": "all", "where": [["lifecycle_stage", "=", "lead"]]}`))
	mustOK(run("2026-07-20T11:14:00Z", "--client-dir", ws, "campaign", "create",
		"--slug", "demo", "--json", `{"audience": {"segment": "all"}, "sendboxes": ["sb-a"], "daily_quota": 10}`))
	q := mustOK(run("2026-07-20T11:15:00Z", "--client-dir", ws, "campaign", "queue", "--slug", "demo"))
	if mInt(mMap(q, "skipped"), "duplicate_suspected", -1) != 2 {
		t.Fatalf("both suspects must be held out: %v", q)
	}
	mustOK(run("2026-07-20T11:16:00Z", "--client-dir", ws, "contact", "unsuspect", "--id", "c_d", "--other", "c_e"))
	if len(mList(readDoc("c_d"), "duplicate_suspects")) != 0 || len(mList(readDoc("c_e"), "duplicate_suspects")) != 0 {
		t.Fatal("unsuspect must clear both sides")
	}
	q = mustOK(run("2026-07-20T11:17:00Z", "--client-dir", ws, "campaign", "queue", "--slug", "demo"))
	if mInt(q, "queued", -1) != 2 || mInt(mMap(q, "skipped"), "duplicate_suspected", -1) != 0 {
		t.Fatalf("cleared suspects must queue: %v", q)
	}
}

// TestPersonalizationGates locks the fixes for the 108-draft audit: enrichment
// data on file must actually reach the email, a declared hook must be woven,
// and an undated website_update never counts as proof-of-life.
func TestPersonalizationGates(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dcp")
	ws := filepath.Join(root, "clients", "leadup", "video_us", "outreach")
	run := func(now string, argv ...string) xresult {
		return runGoStep(t, xstep{FakeNow: now, Argv: argv})
	}
	mustOK := func(r xresult) map[string]any {
		t.Helper()
		if r.Code != 0 {
			t.Fatalf("exit %d: %s%s", r.Code, r.Stdout, r.Stderr)
		}
		return parseOut(t, r)
	}
	mustOK(run("2026-07-21T10:00:00Z", "--pipeline", root, "--client", "leadup",
		"--business", "video", "--location", "us", "init-client"))
	writeFixture(t, ws)
	mustOK(run("2026-07-21T10:01:00Z", "--client-dir", ws, "segment", "set", "--json",
		`{"id":"all","name":"all","where":[["lifecycle_stage","=","lead"]]}`))
	mustOK(run("2026-07-21T10:02:00Z", "--client-dir", ws, "campaign", "create", "--slug", "demo",
		"--json", `{"audience":{"segment":"all","personalization":{"no_hook_fallback":"generic_honest_opener"}},"sendboxes":["sb-a"],"daily_quota":50}`))

	// A: contact WITH a dated, real hook
	mustOK(run("2026-07-21T10:03:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"id":"c_hot","name":{"full":"Maria Mendez"},"identities":{"emails":[{"address":"maria@re.com","is_primary":true}]}}`))
	mustOK(run("2026-07-21T10:04:00Z", "--client-dir", ws, "enrich", "write", "--contact", "c_hot", "--json",
		`{"identity":{"still_active":"confirmed"},
		  "hooks":[{"type":"social_post","summary":"July 15 video tour of a remodeled Cerritos townhome","evidence_url":"https://fb/1","observed_date":"2026-07-15","confidence":0.9}],
		  "writing_brief":{"personalization_confidence":0.8}}`))

	// 1. hooks on file + generic email (no hooks_used) => REFUSED, even though
	//    the campaign allows a generic opener when there is nothing to say.
	r := run("2026-07-21T10:05:00Z", "--client-dir", ws, "draft", "write", "--contact", "c_hot",
		"--campaign", "demo", "--json",
		`{"step":1,"subject":"Maria: a 30-day plan","body_text":"Hi Maria, I prepared a personalized plan based on your work as a real estate professional."}`)
	if r.Code == 0 || !strings.Contains(r.Stderr, "hooks_available_but_unused") {
		t.Fatalf("generic email to an enriched contact must be refused: %d %s", r.Code, r.Stderr)
	}
	// held, not dropped: an activity records it for the operator
	held := false
	for _, p := range mustGlob(t, filepath.Join(ws, "crm", "activities", "*", "activities.jsonl")) {
		for _, a := range readJSONLines(p) {
			if mStr(a, "type") == "draft_held_no_evidence" && mStr(a, "contact_id") == "c_hot" {
				held = true
			}
		}
	}
	if !held {
		t.Fatal("a refused draft must be recorded as held_no_evidence, not silently dropped")
	}

	// 2. declares the hook but writes a template => REFUSED (decorative hook)
	r = run("2026-07-21T10:07:00Z", "--client-dir", ws, "draft", "write", "--contact", "c_hot",
		"--campaign", "demo", "--json",
		`{"step":1,"subject":"Maria: content angles","body_text":"Hi Maria, I prepared a refreshed 30-day content plan for your business.","hooks_used":[{"type":"social_post","evidence_url":"https://fb/1"}]}`)
	if r.Code == 0 || !strings.Contains(r.Stderr, "hook_not_woven") {
		t.Fatalf("decorative hook must be refused: %d %s", r.Code, r.Stderr)
	}

	// 3. genuinely woven (mentions Cerritos) => ACCEPTED
	ok := mustOK(run("2026-07-21T10:08:00Z", "--client-dir", ws, "draft", "write", "--contact", "c_hot",
		"--campaign", "demo", "--json",
		`{"step":1,"subject":"That Cerritos townhome tour","body_text":"Hi Maria, your Cerritos tour answered the question buyers keep asking, which is why it deserves a series.","hooks_used":[{"type":"social_post","evidence_url":"https://fb/1"}]}`))
	if mStr(ok, "draft_id") == "" && mStr(ok, "id") == "" {
		t.Fatalf("woven draft should be written: %v", ok)
	}

	// B: contact whose ONLY "hook" is an undated website_update => not proof of
	// life, so it falls to the no-hook path (generic opener allowed here).
	mustOK(run("2026-07-21T10:09:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"id":"c_web","name":{"full":"David G"},"identities":{"emails":[{"address":"d@hub.com","is_primary":true}]}}`))
	mustOK(run("2026-07-21T10:10:00Z", "--client-dir", ws, "enrich", "write", "--contact", "c_web", "--json",
		`{"identity":{"still_active":"confirmed"},
		  "hooks":[{"type":"website_update","summary":"Insurance Brokers | HUB International","evidence_url":"https://hub.com/","confidence":0.5}],
		  "writing_brief":{"personalization_confidence":0.5}}`))
	gen := mustOK(run("2026-07-21T10:11:00Z", "--client-dir", ws, "draft", "write", "--contact", "c_web",
		"--campaign", "demo", "--json",
		`{"step":1,"subject":"David: content angles","body_text":"Hi David, I put together a plan for your agency."}`))
	if !containsStr(mList(gen, "warnings"), "generic_opener") {
		t.Fatalf("undated website_update must not count as proof-of-life: %v", gen)
	}

	// 4. with no_hook_fallback=skip the same lead is HELD, not drafted
	mustOK(run("2026-07-21T10:12:00Z", "--client-dir", ws, "campaign", "update", "--slug", "demo",
		"--json", `{"audience":{"personalization":{"no_hook_fallback":"skip"}}}`))
	mustOK(run("2026-07-21T10:13:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"id":"c_bare","name":{"full":"No Data"},"identities":{"emails":[{"address":"n@x.com","is_primary":true}]}}`))
	r = run("2026-07-21T10:14:00Z", "--client-dir", ws, "draft", "write", "--contact", "c_bare",
		"--campaign", "demo", "--json", `{"step":1,"subject":"Hello","body_text":"Hi there, a quick note."}`)
	if r.Code == 0 || !strings.Contains(r.Stderr, "no_evidenced_hook") {
		t.Fatalf("skip policy must refuse a hookless draft: %d %s", r.Code, r.Stderr)
	}
}

func containsStr(l []any, want string) bool {
	for _, x := range l {
		if fmt.Sprint(x) == want {
			return true
		}
	}
	return false
}

func mustGlob(t *testing.T, pat string) []string {
	t.Helper()
	m, err := filepath.Glob(pat)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// TestNoVerifiableHookCache locks the negative cache that was written but never
// read: a lead whose springboard was already exhausted must not be re-hunted
// every hookTTL (10d) forever, and the mark must clear the moment a hook IS found.
func TestNoVerifiableHookCache(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dcp")
	ws := filepath.Join(root, "clients", "leadup", "video_us", "outreach")
	run := func(now string, argv ...string) xresult {
		return runGoStep(t, xstep{FakeNow: now, Argv: argv})
	}
	mustOK := func(r xresult) map[string]any {
		t.Helper()
		if r.Code != 0 {
			t.Fatalf("exit %d: %s%s", r.Code, r.Stdout, r.Stderr)
		}
		return parseOut(t, r)
	}
	mustOK(run("2026-07-01T10:00:00Z", "--pipeline", root, "--client", "leadup",
		"--business", "video", "--location", "us", "init-client"))
	mustOK(run("2026-07-01T10:01:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"id":"c_dry","name":{"full":"Dry Lead"},"identities":{"emails":[{"address":"dry@x.com","is_primary":true}]}}`))

	// springboard exhausted, nothing evidenced -> mark_no_hook
	mustOK(run("2026-07-01T10:02:00Z", "--client-dir", ws, "enrich", "write", "--contact", "c_dry", "--json",
		`{"identity":{"still_active":"confirmed"},"hooks":[],"mark_no_hook":true,
		  "writing_brief":{"personalization_confidence":0.2}}`))

	// 12 days later: hooks are stale (TTL 10) but the negative cache must hold —
	// this is exactly the case that used to re-burn collector calls every 10 days
	st := mustOK(run("2026-07-13T10:00:00Z", "--client-dir", ws, "enrich", "status", "--contact", "c_dry"))
	if st["needs"] != "skip" || st["reason"] != "no_verifiable_hook_recent" {
		t.Fatalf("inside the 30d window the lead must be skipped, got: %v", st)
	}

	// after the 30-day window it becomes workable again (not abandoned forever)
	st = mustOK(run("2026-08-05T10:00:00Z", "--client-dir", ws, "enrich", "status", "--contact", "c_dry"))
	if st["needs"] == "skip" {
		t.Fatalf("after 30d the lead must be retryable, got: %v", st)
	}

	// and a pass that DOES find a hook clears the mark immediately
	mustOK(run("2026-08-05T10:01:00Z", "--client-dir", ws, "enrich", "write", "--contact", "c_dry", "--json",
		`{"identity":{"still_active":"confirmed"},
		  "hooks":[{"type":"social_post","summary":"opened a second office in Tustin","evidence_url":"https://z/9","observed_date":"2026-08-04","confidence":0.8}],
		  "writing_brief":{"personalization_confidence":0.8}}`))
	doc, err := readJSONFile(filepath.Join(ws, "crm", "contacts", "c_dry.json"))
	if err != nil {
		t.Fatal(err)
	}
	if mStr(mMap(doc, "enrichment"), "no_verifiable_hook_at") != "" {
		t.Fatal("finding a hook must clear no_verifiable_hook_at, else the lead stays skipped with good data")
	}
	st = mustOK(run("2026-08-05T10:02:00Z", "--client-dir", ws, "enrich", "status", "--contact", "c_dry"))
	if st["reason"] != "dossier_fresh" {
		t.Fatalf("with a fresh hook the lead must be workable: %v", st)
	}
}

// TestFacebookURLShapeClassification locks the story.php misclassification: a
// post URL that lands in identities.socials is treated as a reachable profile
// anchor, so the seed -> profile -> email ladder never runs for that lead and
// the post URL becomes the person's dedupe key.
func TestFacebookURLShapeClassification(t *testing.T) {
	cases := []struct{ url, wantKind, wantSub string }{
		// the bug: one path segment + "story_fbid=" (which contains "id=")
		{"https://m.facebook.com/story.php?story_fbid=pfbid0abc&id=100001", "seed", "post"},
		{"https://www.facebook.com/story.php?story_fbid=pfbid0abc", "seed", "post"},
		{"https://www.facebook.com/permalink.php?story_fbid=123&id=456", "seed", "post"},
		{"https://www.facebook.com/share.php?u=x", "seed", "post"},
		{"https://www.facebook.com/photo.php?fbid=99", "seed", "post"},
		{"https://www.facebook.com/susan.vo/posts/12345", "seed", "post"},
		{"https://www.facebook.com/reel/1004302002391157/", "seed", "reel"},
		{"https://www.facebook.com/groups/realtors/permalink/9/", "seed", "group"},
		// real profiles must still classify as profiles
		{"https://www.facebook.com/susan.vo", "profile", ""},
		{"https://www.facebook.com/profile.php?id=100001", "profile", ""},
	}
	for _, c := range cases {
		kind, platform, sub := classifyLeadURLFull(c.url)
		if kind != c.wantKind || sub != c.wantSub {
			t.Errorf("%s => (%s,%s,%s), want (%s,facebook,%s)", c.url, kind, platform, sub, c.wantKind, c.wantSub)
		}
		if platform != "facebook" {
			t.Errorf("%s => platform %q, want facebook", c.url, platform)
		}
	}
}

// TestEmailDiscoveryNotExhausted reproduces the live failure: 100 Facebook seeds
// produced 99 hooks and 0 emails, and the run reported "skipped 100" although the
// email hunt had never left the Facebook pass.
func TestEmailDiscoveryNotExhausted(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dcp")
	ws := filepath.Join(root, "clients", "leadup", "video_us", "outreach")
	run := func(now string, argv ...string) xresult {
		return runGoStep(t, xstep{FakeNow: now, Argv: argv})
	}
	mustOK := func(r xresult) map[string]any {
		t.Helper()
		if r.Code != 0 {
			t.Fatalf("exit %d: %s%s", r.Code, r.Stdout, r.Stderr)
		}
		return parseOut(t, r)
	}
	mustOK(run("2026-07-28T09:00:00Z", "--pipeline", root, "--client", "leadup",
		"--business", "video", "--location", "us", "init-client"))
	// a hand-saved reel, resolved to its owner, hook found, still no address
	mustOK(run("2026-07-28T09:01:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"id":"c_reel","identities":{"socials":{"facebook":"https://www.facebook.com/agent.x"},
		  "seeds":[{"url":"https://www.facebook.com/reel/555","kind":"reel","platform":"facebook"}]}}`))

	// the Facebook-only pass: real hook, but mark_email_not_found must be REFUSED
	res := mustOK(run("2026-07-28T09:02:00Z", "--client-dir", ws, "enrich", "write", "--contact", "c_reel", "--json",
		`{"identity":{"still_active":"confirmed","channels_found":{"profiles":{"facebook":"https://www.facebook.com/agent.x"}}},
		  "hooks":[{"type":"social_post","summary":"reel about a Whittier listing","evidence_url":"https://www.facebook.com/reel/555","observed_date":"2026-07-20","confidence":0.8}],
		  "mark_email_not_found":true,
		  "writing_brief":{"personalization_confidence":0.7}}`))
	if !strings.Contains(fmt.Sprint(mList(res, "problems")), "mark_email_not_found REFUSED") {
		t.Fatalf("a facebook-only dossier must not be allowed to declare the email unfindable: %v", res)
	}
	if mInt(res, "usable_hooks", -1) != 1 {
		t.Fatalf("the hooks it DID gather must still be kept: %v", res)
	}
	// and the lead stays queued as unfinished research, not skipped
	st := mustOK(run("2026-07-28T09:03:00Z", "--client-dir", ws, "enrich", "status", "--contact", "c_reel"))
	if st["needs"] != "enrich" || st["reason"] != "email_discovery" {
		t.Fatalf("lead must remain queued for email discovery, got: %v", st)
	}

	// leaving Facebook is not enough on its own: with a profile on file, the
	// About / Contact-info tab is where the address usually lives, and a run that
	// skipped it has not exhausted the hunt
	res2 := mustOK(run("2026-07-28T09:04:00Z", "--client-dir", ws, "enrich", "write", "--contact", "c_reel", "--json",
		`{"identity":{"still_active":"confirmed","evidence":[{"fact":"no address on the site","url":"https://agentx-realty.com/contact","retrieved_at":"2026-07-28"}]},
		  "hooks":[],"mark_email_not_found":true,
		  "writing_brief":{"personalization_confidence":0.7}}`))
	if !strings.Contains(fmt.Sprint(mList(res2, "problems")), "directory_contact_info") {
		t.Fatalf("skipping the About tab must be refused when a profile is on file: %v", res2["problems"])
	}

	// the real contact surfaces are sub-pages, not "/about": a run that visited
	// directory_contact_info has done the high-yield step and is accepted
	mustOK(run("2026-07-28T09:04:30Z", "--client-dir", ws, "enrich", "write", "--contact", "c_reel", "--json",
		`{"identity":{"still_active":"confirmed","evidence":[
		    {"fact":"Contact info page shows no address","url":"https://www.facebook.com/agent.x/directory_contact_info","retrieved_at":"2026-07-28"},
		    {"fact":"no address on the site","url":"https://agentx-realty.com/contact","retrieved_at":"2026-07-28"}]},
		  "hooks":[],"mark_email_not_found":true,
		  "writing_brief":{"personalization_confidence":0.7}}`))
	st = mustOK(run("2026-07-28T09:05:00Z", "--client-dir", ws, "enrich", "status", "--contact", "c_reel"))
	if st["needs"] != "skip" || st["reason"] != "email_not_found_recent" {
		t.Fatalf("after a genuine off-Facebook hunt the negative cache must apply: %v", st)
	}

	// a contact WITH an address is never held for email discovery
	mustOK(run("2026-07-28T09:06:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"id":"c_has","identities":{"emails":[{"address":"has@x.com","is_primary":true}]}}`))
	mustOK(run("2026-07-28T09:07:00Z", "--client-dir", ws, "enrich", "write", "--contact", "c_has", "--json",
		`{"identity":{"still_active":"confirmed"},
		  "hooks":[{"type":"social_post","summary":"posted a market update","evidence_url":"https://www.facebook.com/has/posts/1","observed_date":"2026-07-25","confidence":0.8}],
		  "writing_brief":{"personalization_confidence":0.8}}`))
	st = mustOK(run("2026-07-28T09:08:00Z", "--client-dir", ws, "enrich", "status", "--contact", "c_has"))
	if st["reason"] != "dossier_fresh" {
		t.Fatalf("a contact with an address must be workable: %v", st)
	}
}

// TestEmailChannelBecomesUsable: the drafting bar is a REAL address that parses,
// not a mailbox probe. `channels.email.status` used to read "needs_data" forever,
// which a live run mistook for a blocker and refused to draft 28 valid contacts.
func TestEmailChannelBecomesUsable(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dcp")
	ws := filepath.Join(root, "clients", "leadup", "video_us", "outreach")
	run := func(now string, argv ...string) xresult {
		return runGoStep(t, xstep{FakeNow: now, Argv: argv})
	}
	mustOK := func(r xresult) map[string]any {
		t.Helper()
		if r.Code != 0 {
			t.Fatalf("exit %d: %s%s", r.Code, r.Stdout, r.Stderr)
		}
		return parseOut(t, r)
	}
	readC := func(id string) map[string]any {
		d, err := readJSONFile(filepath.Join(ws, "crm", "contacts", id+".json"))
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	mustOK(run("2026-07-28T10:00:00Z", "--pipeline", root, "--client", "leadup",
		"--business", "video", "--location", "us", "init-client"))
	writeFixture(t, ws)

	// a seed-only contact starts with no address: needs_data is correct here
	mustOK(run("2026-07-28T10:01:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"id":"c_seed","identities":{"seeds":[{"url":"https://www.facebook.com/reel/31","kind":"reel","platform":"facebook"}]}}`))
	if st := mStr(mMap(mMap(readC("c_seed"), "channels"), "email"), "status"); st != "needs_data" {
		t.Fatalf("no address yet => needs_data, got %q", st)
	}

	// enrichment finds a real address off the website: the channel must flip
	mustOK(run("2026-07-28T10:02:00Z", "--client-dir", ws, "enrich", "write", "--contact", "c_seed", "--json",
		`{"identity":{"still_active":"confirmed","channels_found":{"emails":["agent@realty.com"]},
		   "evidence":[{"fact":"address on the contact page","url":"https://realty.com/contact","retrieved_at":"2026-07-28"}]},
		  "hooks":[{"type":"social_post","summary":"reel about a Whittier listing","evidence_url":"https://www.facebook.com/reel/31","observed_date":"2026-07-20","confidence":0.8}],
		  "writing_brief":{"personalization_confidence":0.8}}`))
	if st := mStr(mMap(mMap(readC("c_seed"), "channels"), "email"), "status"); st != "usable" {
		t.Fatalf("a real found address must make the email channel usable, got %q", st)
	}
	// enrichment addresses are stored `unverified` (no mailbox probe exists);
	// that must NOT stop drafting
	em := mapsOf(mList(mMap(readC("c_seed"), "identities"), "emails"))
	if len(em) != 1 || mStr(em[0], "status") != "unverified" {
		t.Fatalf("enrich addresses stay unverified by design: %v", em)
	}
	st := mustOK(run("2026-07-28T10:03:00Z", "--client-dir", ws, "enrich", "status", "--contact", "c_seed"))
	if st["reason"] != "dossier_fresh" {
		t.Fatalf("with hooks + an address the lead is workable: %v", st)
	}

	// suppression stays terminal: a bounced channel is never flipped back
	mustOK(run("2026-07-28T10:04:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"id":"c_bad","identities":{"emails":[{"address":"bad@x.com","is_primary":true}]},
		  "channels":{"email":{"status":"bounced"}}}`))
	mustOK(run("2026-07-28T10:05:00Z", "--client-dir", ws, "contact", "add", "--json",
		`{"id":"c_bad","identities":{"emails":[{"address":"bad@x.com"}]}}`))
	if st := mStr(mMap(mMap(readC("c_bad"), "channels"), "email"), "status"); st != "bounced" {
		t.Fatalf("suppression must stay terminal, got %q", st)
	}
}

// TestSearchURLNeverBecomesIdentity reproduces the corruption: a live run stored
// Bing SEARCH URLs as "found profiles". normalizeSocial drops the query string,
// so every search collapsed to the key "bing.com/search", unrelated leads
// appeared to share an identity, and consolidation chain-merged 54 reels from
// different people into one contact.
func TestSearchURLNeverBecomesIdentity(t *testing.T) {
	// 1. the classifier refuses search/aggregator hosts outright
	for _, u := range []string{
		"https://www.bing.com/search?q=alex+lehr+realtor",
		"https://www.google.com/search?q=huong+ly",
		"https://duckduckgo.com/?q=realtor+magazine",
		"https://l.facebook.com/l.php?u=https%3A%2F%2Fx.com",
	} {
		if k, p, _ := classifyLeadURLFull(u); k != "" || p != "" {
			t.Errorf("%s must not classify as anything usable, got kind=%q platform=%q", u, k, p)
		}
	}

	root := filepath.Join(t.TempDir(), "dcp")
	ws := filepath.Join(root, "clients", "leadup", "video_us", "outreach")
	run := func(now string, argv ...string) xresult {
		return runGoStep(t, xstep{FakeNow: now, Argv: argv})
	}
	mustOK := func(r xresult) map[string]any {
		t.Helper()
		if r.Code != 0 {
			t.Fatalf("exit %d: %s%s", r.Code, r.Stdout, r.Stderr)
		}
		return parseOut(t, r)
	}
	mustOK(run("2026-07-28T11:00:00Z", "--pipeline", root, "--client", "leadup",
		"--business", "video", "--location", "us", "init-client"))

	// 2. two unrelated reels, each "resolved" to a Bing search URL. Before the
	//    fix this made them share an identity and merged them together.
	for _, id := range []string{"c_r1", "c_r2"} {
		mustOK(run("2026-07-28T11:01:00Z", "--client-dir", ws, "contact", "add", "--json",
			`{"id":"`+id+`","identities":{"seeds":[{"url":"https://www.facebook.com/reel/`+id+`","kind":"reel","platform":"facebook"}]}}`))
	}
	for i, id := range []string{"c_r1", "c_r2"} {
		res := mustOK(run("2026-07-28T11:0"+string(rune('2'+i))+":00Z", "--client-dir", ws, "enrich", "write", "--contact", id, "--json",
			`{"identity":{"still_active":"confirmed","channels_found":{"profiles":{"facebook":"https://www.bing.com/search?q=some+name"}}},
			  "writing_brief":{"personalization_confidence":0.5}}`))
		if !strings.Contains(fmt.Sprint(mList(res, "problems")), "not a facebook PROFILE url") {
			t.Fatalf("a search URL must be rejected as identity: %v", res["problems"])
		}
		if len(mapsOf(mList(res, "consolidated"))) != 0 {
			t.Fatalf("a rejected identity must never trigger a merge: %v", res["consolidated"])
		}
	}
	// both contacts must still be independent and un-merged
	for _, id := range []string{"c_r1", "c_r2"} {
		d, err := readJSONFile(filepath.Join(ws, "crm", "contacts", id+".json"))
		if err != nil {
			t.Fatal(err)
		}
		if mStr(mMap(d, "merge"), "status") == "merged" {
			t.Fatalf("%s was wrongly merged on a search URL", id)
		}
		if truthy(mMap(mMap(d, "identities"), "socials")["facebook"]) {
			t.Fatalf("%s stored a search URL as its facebook identity", id)
		}
	}

	// 3. a REAL profile URL still resolves and consolidates normally
	mustOK(run("2026-07-28T11:05:00Z", "--client-dir", ws, "enrich", "write", "--contact", "c_r1", "--json",
		`{"identity":{"still_active":"confirmed","channels_found":{"profiles":{"facebook":"https://www.facebook.com/alex.lehr"}}},
		  "writing_brief":{"personalization_confidence":0.5}}`))
	res := mustOK(run("2026-07-28T11:06:00Z", "--client-dir", ws, "enrich", "write", "--contact", "c_r2", "--json",
		`{"identity":{"still_active":"confirmed","channels_found":{"profiles":{"facebook":"https://www.facebook.com/alex.lehr"}}},
		  "writing_brief":{"personalization_confidence":0.5}}`))
	if len(mapsOf(mList(res, "consolidated"))) != 1 {
		t.Fatalf("a genuine shared profile must still consolidate: %v", res)
	}
}

// TestFacebookContactSurfaces: About is a chooser, the address lives in a
// sub-page. Each of these must satisfy the "did you look where it actually is"
// gate, or a run that did the right thing would be refused.
func TestFacebookContactSurfaces(t *testing.T) {
	for _, u := range []string{
		"https://www.facebook.com/bgvinvest/directory_contact_info",
		"https://www.facebook.com/bgvinvest/directory_intro",
		"https://www.facebook.com/bgvinvest/directory_basic_info",
		"https://www.facebook.com/bgvinvest/directory_links",
		"https://www.facebook.com/Khanhngo.us/about",
		"https://www.facebook.com/profile.php?id=123&sk=about_contact_and_basic_info",
	} {
		d := map[string]any{"hooks": []any{}}
		id := map[string]any{"evidence": []any{map[string]any{"url": u}}}
		if !dossierReadAboutTab(d, id) {
			t.Errorf("%s is a real contact surface and must satisfy the gate", u)
		}
	}
	// the reel page itself never counts: it is where the run wrongly stopped
	d := map[string]any{"hooks": []any{}}
	id := map[string]any{"evidence": []any{map[string]any{"url": "https://www.facebook.com/reel/12345"}}}
	if dossierReadAboutTab(d, id) {
		t.Error("a reel page must not count as having looked for contact details")
	}
}
