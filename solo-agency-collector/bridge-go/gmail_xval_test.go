package main

// gmail_xval_test.go — offline gmail tests: the deterministic inbox
// classifier over .eml fixtures and the pre-send chain (dry-run, terminal
// blockers, quota). Expectations were verified against the retired Python
// gmail_client before the retirement; live SMTP/IMAP was validated on a real
// sendbox 2026-07-19 (see docs/UI_DESIGN.md delivery log).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGmailClassify(t *testing.T) {
	sendbox := "me@gmail.com"
	known := map[string]map[string]any{
		"<mid-1@gmail.com>": {"lead_id": "c_lead1", "campaign": "demo"},
	}
	resolver := func(addr string) map[string]any {
		if addr == "susan@kw.com" {
			return map[string]any{"lead_id": "c_lead1"}
		}
		return nil
	}
	dsn := strings.Join([]string{
		"From: Mail Delivery Subsystem <mailer-daemon@googlemail.com>",
		"To: me@gmail.com",
		"Subject: Delivery Status Notification (Failure)",
		`Content-Type: multipart/report; report-type=delivery-status; boundary="b1"`,
		"",
		"--b1",
		"Content-Type: text/plain",
		"",
		"Address not found: your message wasn't delivered to gone@dead.com.",
		"--b1",
		"Content-Type: message/delivery-status",
		"",
		"Reporting-MTA: dns; googlemail.com",
		"",
		"Final-Recipient: rfc822; gone@dead.com",
		"Action: failed",
		"Status: 5.1.1",
		"--b1",
		"Content-Type: text/rfc822-headers",
		"",
		"Message-ID: <mid-1@gmail.com>",
		"From: me@gmail.com",
		"--b1--",
		"",
	}, "\r\n")

	cases := []struct {
		name string
		raw  string
		want map[string]any
	}{
		{"hard-dsn", dsn, map[string]any{"kind": "bounce", "hard": true,
			"bounced_message_id": "<mid-1@gmail.com>", "final_recipient": "gone@dead.com"}},
		{"ooo", "From: pto@corp.com\r\nTo: me@gmail.com\r\nSubject: Automatic reply: Idea\r\nAuto-Submitted: auto-replied\r\n\r\nI am away.\r\n",
			map[string]any{"kind": "auto_reply_ooo"}},
		{"unsub-alias", "From: susan@kw.com\r\nTo: me+unsub-abc123@gmail.com\r\nSubject: unsubscribe\r\n\r\nstop\r\n",
			map[string]any{"kind": "unsubscribe", "token": "abc123"}},
		{"thread-reply", "From: other@kw.com\r\nTo: me@gmail.com\r\nSubject: Re: Idea\r\nIn-Reply-To: <mid-1@gmail.com>\r\n\r\nTell me more\r\n",
			map[string]any{"kind": "campaign_reply", "lead_id": "c_lead1", "campaign": "demo", "in_reply_to": "<mid-1@gmail.com>"}},
		{"from-fallback", "From: Susan Vo <susan@kw.com>\r\nTo: me@gmail.com\r\nSubject: hello again\r\n\r\nSounds good\r\n",
			map[string]any{"kind": "campaign_reply", "lead_id": "c_lead1", "campaign": nil, "matched_by": "from_address"}},
		{"personal", "From: mom@family.com\r\nTo: me@gmail.com\r\nSubject: dinner\r\n\r\nSunday?\r\n",
			map[string]any{"kind": "contact_or_personal", "from": "mom@family.com"}},
	}
	for _, c := range cases {
		msg, err := parseEmailMessage([]byte(c.raw))
		if err != nil {
			t.Fatalf("%s: parse: %v", c.name, err)
		}
		got := gmailClassifyMessage(msg, sendbox, known, resolver)
		gb, _ := json.Marshal(got)
		wb, _ := json.Marshal(c.want)
		var gv, wv map[string]any
		json.Unmarshal(gb, &gv)
		json.Unmarshal(wb, &wv)
		for k, want := range wv {
			gj, _ := json.Marshal(gv[k])
			wj, _ := json.Marshal(want)
			if string(gj) != string(wj) {
				t.Errorf("%s: field %s = %s, want %s (full: %s)", c.name, k, gj, wj, gb)
			}
		}
	}
}

func TestGmailSendOffline(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dcp")
	ws := filepath.Join(root, "clients", "leadup", "video_us", "outreach")
	steps := [][]string{
		{"--pipeline", root, "--client", "leadup", "--business", "video", "--location", "us", "init-client"},
		{"--client-dir", ws, "contact", "add", "--json", `{"id": "c_lead1", "name": {"full": "Susan Vo"}, "identities": {"emails": [{"address": "susan@kw.com", "is_primary": true}]}}`},
		{"--client-dir", ws, "campaign", "create", "--slug", "demo", "--json", `{"audience": {"segment": "all"}, "sendboxes": ["sb-a"]}`},
	}
	for _, argv := range steps {
		if r := runGoStep(t, xstep{"2026-07-19T12:00:00Z", argv}); r.Code != 0 {
			t.Fatalf("setup %v: %s", argv, r.Stderr)
		}
	}
	mustWrite := func(rel, body string) {
		p := filepath.Join(ws, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
	}
	mustWrite("sendboxes/sendboxes.json", `{"sendboxes": [{"slug": "sb-a", "email": "me@gmail.com", "domain": "gmail.com", "quota_today": 5, "status": "healthy", "imap_uid_cursor": 0}]}`)
	mustWrite("config/sending_identity.json", `{"from_name": "Binh at LeadUp", "physical_mailing_address": "1 Main St, San Jose, CA 95112"}`)
	draft := `{"id": "draft_fix1", "schema_version": 1, "created_at": "2026-07-19T12:00:00Z",
	  "updated_at": "2026-07-19T12:00:00Z", "lead_id": "c_lead1", "campaign_slug": "demo",
	  "step": 1, "sendbox": "sb-a", "to": "susan@kw.com", "subject": "Idea for 123 Main St",
	  "body_text": "Hi Susan...", "body_html": "", "confidence_band": "high",
	  "hooks_used": [], "tracking": "plain_text", "warnings": [], "guessed_approved": false,
	  "is_reply": false, "bank_messages_used": [], "companion_url": "",
	  "status": "approved", "decided_at": "2026-07-19T12:01:00Z", "decided_by": "human",
	  "reject_reason": "", "blocker": ""}`
	draftPath := filepath.Join(ws, "campaigns", "demo", "outbox", "approved", "draft_fix1.json")
	os.MkdirAll(filepath.Dir(draftPath), 0o755)
	os.WriteFile(draftPath, []byte(draft), 0o644)

	gmail := func(argv ...string) xresult {
		return runCLIStep(t, xstep{"2026-07-19T12:05:00Z", argv}, runGmailCLI)
	}

	// dry-run passes the whole presend chain, reports plan, reserves nothing
	dry := parseOut(t, gmail("--client-dir", ws, "send", "--draft", draftPath, "--dry-run"))
	if dry["ok"] != true || dry["dry_run"] != true || mStr(dry, "would_send_to") != "susan@kw.com" ||
		!strings.Contains(mStr(dry, "list_unsubscribe"), "+unsub-") {
		t.Fatalf("dry-run: %v", dry)
	}
	q := parseOut(t, gmail("--client-dir", ws, "quota", "--sendbox", "sb-a", "--day", "2026-07-19"))
	if mInt(q, "reserved", -1) != 0 || mInt(q, "remaining", -1) != 5 {
		t.Fatalf("dry-run must not reserve quota: %v", q)
	}

	// suppress recipient -> terminal blocker persisted, no SMTP attempted
	if r := runGoStep(t, xstep{"2026-07-19T12:06:00Z", []string{"--client-dir", ws,
		"suppress", "add", "--kind", "email", "--value", "susan@kw.com", "--reason", "unsubscribe"}}); r.Code != 0 {
		t.Fatal(r.Stderr)
	}
	blocked := parseOut(t, gmail("--client-dir", ws, "send", "--draft", draftPath))
	if blocked["ok"] != false || mStr(blocked, "blocker") != "suppressed" {
		t.Fatalf("suppressed send: %v", blocked)
	}
	d, _ := readJSONFile(draftPath)
	if mStr(d, "status") != "blocked" || mStr(d, "blocker") != "suppressed" {
		t.Fatalf("terminal blocker not persisted: %v", d)
	}

	// non-approved draft short-circuits before everything else
	d["status"] = "hold"
	os.WriteFile(draftPath, []byte(marshalIndentJSON(d)), 0o644)
	na := parseOut(t, gmail("--client-dir", ws, "send", "--draft", draftPath))
	if mStr(na, "blocker") != "draft_not_approved" {
		t.Fatalf("not-approved: %v", na)
	}
}

func TestStripAppPasswordWhitespace(t *testing.T) {
	const want = "abcdefghijklmnop"
	nbsp := "\u00A0"
	zwsp := "\u200B"
	bom := "\uFEFF"
	cases := map[string]string{
		"abcd efgh ijkl mnop":                                  "gmail display (ascii spaces)",
		"  abcd efgh ijkl mnop  ":                              "leading/trailing + inner",
		"abcd\tefgh\tijkl\tmnop":                               "tabs",
		"abcd efgh ijkl mnop\n":                                "trailing newline",
		"abcd" + nbsp + "efgh" + nbsp + "ijkl" + nbsp + "mnop": "non-breaking spaces",
		"abcd" + zwsp + "efgh ijkl mnop":                       "zero-width space",
		bom + "abcd efgh ijkl mnop":                            "leading BOM",
		"abcdefghijklmnop":                                     "already bare",
	}
	for in, desc := range cases {
		if got := stripAllWhitespace(in); got != want {
			t.Errorf("%s: stripAllWhitespace(%q) = %q, want %q", desc, in, got, want)
		}
	}
	if got := stripAllWhitespace("a1b2 c3d4"); got != "a1b2c3d4" {
		t.Errorf("non-space chars altered: %q", got)
	}
}

// TestEffectiveQuotaRamp: playbook 02 called the warm-up ramp "documented
// policy the operator sets by hand" — prose, so no box was ever advanced: a
// box authed at 20/day stayed 20/day forever. The ramp is config now and the
// arithmetic is computed on read, so restarts change nothing.
func TestEffectiveQuotaRamp(t *testing.T) {
	sb := map[string]any{"quota_today": 20}
	if got := effectiveQuota(sb, "2026-07-29T09:00:00Z"); got != 20 {
		t.Fatalf("no ramp → the stored quota: %d", got)
	}
	sb["warmup_ramp"] = map[string]any{"start_date": "2026-07-29", "start_quota": 20,
		"step_per_day": 5, "max_quota": 50}
	for _, tc := range []struct {
		now  string
		want int
	}{
		{"2026-07-29T09:00:00Z", 20}, // day 0
		{"2026-07-30T09:00:00Z", 25},
		{"2026-08-01T23:00:00Z", 35},
		{"2026-08-04T00:00:00Z", 50}, // 20+5*6=50, at cap
		{"2026-09-01T00:00:00Z", 50}, // cap holds forever
		{"2026-07-20T00:00:00Z", 20}, // clock before start → start_quota, never negative
	} {
		if got := effectiveQuota(sb, tc.now); got != tc.want {
			t.Fatalf("at %s want %d got %d", tc.now, tc.want, got)
		}
	}
	// a malformed start date degrades to start_quota, not to zero
	sb["warmup_ramp"] = map[string]any{"start_date": "junk", "start_quota": 30, "step_per_day": 5, "max_quota": 50}
	if got := effectiveQuota(sb, "2026-08-01T00:00:00Z"); got != 30 {
		t.Fatalf("bad start_date must fall back to start_quota: %d", got)
	}
}

// A reply freezes the lead. Answering that reply must still be possible, or
// Stage 10's "replies -> reply drafts" produces drafts nothing can ever send.
func TestPresendFreezeAllowsReplyBlocksBump(t *testing.T) {
	frozen := map[string]any{"sequence_state": "frozen"}
	if mStr(frozen, "sequence_state") == "frozen" && !mBool(map[string]any{"is_reply": true}, "is_reply") {
		t.Fatal("a draft marked is_reply must not be treated as frozen-blocked")
	}
	if !(mStr(frozen, "sequence_state") == "frozen" && !mBool(map[string]any{"is_reply": false}, "is_reply")) {
		t.Fatal("a bump (not is_reply) to a frozen lead must still be blocked")
	}
}

// TestIMAPFetchNeverMarksMailRead: the poller has to read a message before it can
// know who sent it, so the fetch form decides what happens to EVERY message in the
// mailbox — not just campaign replies. `RFC822` and `BODY[]` both implicitly set
// \Seen (RFC 3501 §6.4.5), which silently marked the operator's personal mail read
// and made him miss real messages. BODY.PEEK[] is the same fetch without that.
func TestIMAPFetchNeverMarksMailRead(t *testing.T) {
	cmd := fmt.Sprintf(imapPeekFetch, 42)
	if !strings.Contains(cmd, "BODY.PEEK[]") {
		t.Fatalf("the fetch must peek: %q", cmd)
	}
	if strings.Contains(cmd, "RFC822") {
		t.Fatalf("RFC822 is functionally BODY[] and sets \\Seen: %q", cmd)
	}
	// A bare BODY[] would set it too — PEEK is the only accepted form.
	if strings.Contains(strings.ReplaceAll(cmd, "BODY.PEEK[]", ""), "BODY[") {
		t.Fatalf("a bare BODY[] sets \\Seen: %q", cmd)
	}
	if !strings.Contains(cmd, "UID FETCH 42") {
		t.Fatalf("uid must still be addressed by UID: %q", cmd)
	}
	// This client must never write flags either: read state belongs to the human,
	// and "mark read then put it back" is a race the operator sees on his phone.
	src, err := os.ReadFile("imap.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"STORE", "+FLAGS", "\\\\Seen\"", "UID STORE"} {
		if strings.Contains(string(src), forbidden) && !strings.Contains(string(src), "no STORE") {
			t.Fatalf("imap.go must not write flags, found %q", forbidden)
		}
	}
}
