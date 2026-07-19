package main

// gmail_xval_test.go — cross-validates gmail_client's OFFLINE surfaces against
// Python: the deterministic inbox classifier (DSN/OOO/unsub/thread/from-
// fallback/personal over .eml fixtures), the ordered pre-send chain incl.
// terminal-blocker persistence, dry-run send, and quota. The live SMTP/IMAP
// paths need a real sendbox and stay out of scope here (UI_DESIGN §8 G3).

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const pyClassifyHarness = `
import sys, json, email
tools = sys.argv[1]
sys.path.insert(0, tools)
import gmail_client as g
raw = open(sys.argv[2], "rb").read()
known = json.loads(sys.argv[3])
resolver_map = json.loads(sys.argv[4])
msg = email.message_from_bytes(raw)
res = g.classify_message("", msg, sys.argv[5], known,
                         from_resolver=lambda a: resolver_map.get(a))
print(json.dumps(res, sort_keys=True))
`

func TestGmailClassifyParity(t *testing.T) {
	pyRepo := findCrmStorePy(t)
	tools := filepath.Dir(pyRepo)
	if _, err := os.Stat(filepath.Join(tools, "gmail_client.py")); err != nil {
		t.Skip("gmail_client.py not present")
	}
	dir := t.TempDir()
	sendbox := "me@gmail.com"
	known := map[string]map[string]any{
		"<mid-1@gmail.com>": {"lead_id": "c_lead1", "campaign": "demo"},
	}
	resolver := map[string]map[string]any{
		"susan@kw.com": {"lead_id": "c_lead1"},
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
	ooo := "From: pto@corp.com\r\nTo: me@gmail.com\r\nSubject: Automatic reply: Idea\r\nAuto-Submitted: auto-replied\r\n\r\nI am away.\r\n"
	unsub := "From: susan@kw.com\r\nTo: me+unsub-abc123@gmail.com\r\nSubject: unsubscribe\r\n\r\nstop\r\n"
	threadReply := "From: other@kw.com\r\nTo: me@gmail.com\r\nSubject: Re: Idea\r\nIn-Reply-To: <mid-1@gmail.com>\r\n\r\nTell me more\r\n"
	fromReply := "From: Susan Vo <susan@kw.com>\r\nTo: me@gmail.com\r\nSubject: hello again\r\n\r\nSounds good\r\n"
	personal := "From: mom@family.com\r\nTo: me@gmail.com\r\nSubject: dinner\r\n\r\nSunday?\r\n"

	cases := map[string]string{"dsn.eml": dsn, "ooo.eml": ooo, "unsub.eml": unsub,
		"thread.eml": threadReply, "from.eml": fromReply, "personal.eml": personal}

	knownJSON, _ := json.Marshal(known)
	resolverJSON, _ := json.Marshal(resolver)
	for name, body := range cases {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("python3", "-c", pyClassifyHarness, tools, p,
			string(knownJSON), string(resolverJSON), sendbox)
		pyOut, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: python classify failed: %v\n%s", name, err, pyOut)
		}
		msg, err := parseEmailMessage([]byte(body))
		if err != nil {
			t.Fatalf("%s: go parse failed: %v", name, err)
		}
		goRes := gmailClassifyMessage(msg, sendbox, known, func(addr string) map[string]any {
			if info, ok := resolver[addr]; ok {
				return info
			}
			return nil
		})
		gb, _ := json.Marshal(goRes)
		var pv, gv any
		if err := json.Unmarshal(pyOut, &pv); err != nil {
			t.Fatalf("%s: python output not json: %s", name, pyOut)
		}
		json.Unmarshal(gb, &gv)
		pn, _ := json.Marshal(pv)
		gn, _ := json.Marshal(gv)
		if string(pn) != string(gn) {
			t.Errorf("%s: classify mismatch\npy: %s\ngo: %s", name, pn, gn)
		}
	}
}

var msgIDRe = regexp.MustCompile(`<[0-9a-f]{32}@[^>]+>`)
var unsubTokRe = regexp.MustCompile(`\+unsub-[A-Za-z0-9]+@`)

func canonGmail(s string) string {
	s = msgIDRe.ReplaceAllString(s, "<MID>")
	return unsubTokRe.ReplaceAllString(s, "+unsub-TOK@")
}

func TestGmailSendOfflineParity(t *testing.T) {
	pyRepo := findCrmStorePy(t)
	tools := filepath.Dir(pyRepo)
	gmailPy := filepath.Join(tools, "gmail_client.py")
	if _, err := os.Stat(gmailPy); err != nil {
		t.Skip("gmail_client.py not present")
	}

	setup := func(root string) (string, string) {
		ws := filepath.Join(root, "clients", "leadup", "video_us", "outreach")
		steps := [][]string{
			{"--pipeline", root, "--client", "leadup", "--business", "video", "--location", "us", "init-client"},
			{"--client-dir", ws, "contact", "add", "--json", `{"id": "c_lead1", "name": {"full": "Susan Vo"}, "identities": {"emails": [{"address": "susan@kw.com", "is_primary": true}]}}`},
			{"--client-dir", ws, "campaign", "create", "--slug", "demo", "--json", `{"audience": {"segment": "all"}, "sendboxes": ["sb-a"]}`},
		}
		for _, argv := range steps {
			if rc := runGoStep(t, xstep{FakeNow: "2026-07-19T12:00:00Z", Argv: argv}); rc.Code != 0 {
				t.Fatalf("setup %v rc=%d %s", argv, rc.Code, rc.Stderr)
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
		return ws, draftPath
	}

	pyWS, pyDraft := setup(filepath.Join(t.TempDir(), "dcp"))
	goWS, goDraft := setup(filepath.Join(t.TempDir(), "dcp"))

	runPyGmail := func(args []string) xresult {
		cmd := exec.Command("python3", append([]string{gmailPy}, args...)...)
		cmd.Env = append(os.Environ(), "OUTREACHCRM_TEST_MODE=1", "OUTREACHCRM_FAKE_NOW=2026-07-19T12:05:00Z")
		var out, errb strings.Builder
		cmd.Stdout, cmd.Stderr = &out, &errb
		err := cmd.Run()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
		return xresult{code, out.String(), errb.String()}
	}
	runGoGmail := func(args []string) xresult {
		t.Setenv("OUTREACHCRM_TEST_MODE", "1")
		t.Setenv("OUTREACHCRM_FAKE_NOW", "2026-07-19T12:05:00Z")
		oldOut, oldErr := os.Stdout, os.Stderr
		rOut, wOut, _ := os.Pipe()
		rErr, wErr, _ := os.Pipe()
		os.Stdout, os.Stderr = wOut, wErr
		code := runGmailCLI(args)
		os.Stdout, os.Stderr = oldOut, oldErr
		wOut.Close()
		wErr.Close()
		return xresult{code, readAllString(rOut), readAllString(rErr)}
	}

	compare := func(name string, pyArgs, goArgs []string) {
		pr := runPyGmail(pyArgs)
		gr := runGoGmail(goArgs)
		if pr.Code != gr.Code {
			t.Fatalf("%s: exit py=%d go=%d\npy: %s%s\ngo: %s%s", name, pr.Code, gr.Code,
				pr.Stdout, pr.Stderr, gr.Stdout, gr.Stderr)
		}
		pc := canonGmail(strings.ReplaceAll(pr.Stdout, pyWS, "<WS>"))
		gc := canonGmail(strings.ReplaceAll(gr.Stdout, goWS, "<WS>"))
		var pv, gv any
		if json.Unmarshal([]byte(pc), &pv) != nil || json.Unmarshal([]byte(gc), &gv) != nil {
			t.Fatalf("%s: non-json output\npy: %s\ngo: %s", name, pc, gc)
		}
		pn, _ := json.Marshal(pv)
		gn, _ := json.Marshal(gv)
		if string(pn) != string(gn) {
			t.Fatalf("%s: mismatch\npy: %s\ngo: %s", name, pn, gn)
		}
	}

	// 1. dry-run send passes the whole presend chain and reports the plan
	compare("dry-run",
		[]string{"--client-dir", pyWS, "send", "--draft", pyDraft, "--dry-run"},
		[]string{"--client-dir", goWS, "send", "--draft", goDraft, "--dry-run"})

	// 2. quota (no sends, no reservations yet)
	compare("quota",
		[]string{"--client-dir", pyWS, "quota", "--sendbox", "sb-a", "--day", "2026-07-19"},
		[]string{"--client-dir", goWS, "quota", "--sendbox", "sb-a", "--day", "2026-07-19"})

	// 3. terminal blocker: suppress the recipient, then send (not dry-run) —
	// presend fails BEFORE any SMTP, the draft flips to status=blocked
	for _, side := range []struct {
		ws string
		fn func([]string) xresult
	}{{pyWS, runPyGmail}, {goWS, runGoGmail}} {
		res := runGoStep(t, xstep{FakeNow: "2026-07-19T12:06:00Z",
			Argv: []string{"--client-dir", side.ws, "suppress", "add", "--kind", "email",
				"--value", "susan@kw.com", "--reason", "unsubscribe"}})
		if res.Code != 0 {
			t.Fatalf("suppress setup failed: %s", res.Stderr)
		}
	}
	compare("send-suppressed",
		[]string{"--client-dir", pyWS, "send", "--draft", pyDraft},
		[]string{"--client-dir", goWS, "send", "--draft", goDraft})
	for name, p := range map[string]string{"py": pyDraft, "go": goDraft} {
		d, err := readJSONFile(p)
		if err != nil {
			t.Fatalf("%s draft unreadable: %v", name, err)
		}
		if mStr(d, "status") != "blocked" || mStr(d, "blocker") != "suppressed" {
			t.Fatalf("%s draft not terminal-blocked: status=%s blocker=%s",
				name, mStr(d, "status"), mStr(d, "blocker"))
		}
	}

	// 4. draft_not_approved short-circuit
	for _, p := range []string{pyDraft, goDraft} {
		d, _ := readJSONFile(p)
		d["status"] = "hold"
		os.WriteFile(p, []byte(marshalIndentJSON(d)), 0o644)
	}
	compare("send-not-approved",
		[]string{"--client-dir", pyWS, "send", "--draft", pyDraft},
		[]string{"--client-dir", goWS, "send", "--draft", goDraft})

	_ = fmt.Sprint()
}
