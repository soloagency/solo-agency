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
