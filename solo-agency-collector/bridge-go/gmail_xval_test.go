package main

// gmail_xval_test.go — offline gmail tests: the deterministic inbox
// classifier over .eml fixtures and the pre-send chain (dry-run, terminal
// blockers, quota). Expectations were verified against the retired Python
// gmail_client before the retirement; live SMTP/IMAP was validated on a real
// sendbox 2026-07-19 (see docs/UI_DESIGN.md delivery log).

import (
	"encoding/json"
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
