package main

// operator_mail.go — the accountability escalation channel. After N unheeded
// posting reminders to a CLIENT (their WideCast channel), the OPERATOR gets a
// real email: "client X has been silent for D days, reminded N times — time
// for a personal call". The machine nags the client; the human saves the
// account. Sent through one of the operator's own outreach sendboxes (Gmail +
// app password, already configured for cold outreach) to the operator_email
// recorded in system_settings.json (/ui/settings) — no new credentials, no
// new channel setup.
//
// `tool gmail send-operator --pipeline DIR --subject S (--body B | --body-file F)`

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// operatorSMTPSend is gmailSMTPSend, injectable so tests capture the send
// instead of talking to smtp.gmail.com.
var operatorSMTPSend = gmailSMTPSend

// findOperatorSendbox walks every client's outreach sendboxes and returns the
// first healthy one with usable credentials. Any of the operator's own
// app-password accounts will do — this is the operator mailing themselves.
func findOperatorSendbox(pipeline string) (outreachDir, slug, email, appPassword string, found bool) {
	docs, _ := filepath.Glob(filepath.Join(pipeline, "clients", "*", "*", "outreach", "sendboxes", "sendboxes.json"))
	sort.Strings(docs)
	for _, docPath := range docs {
		oDir := filepath.Dir(filepath.Dir(docPath)) // .../outreach
		doc, err := readJSONFile(docPath)
		if err != nil {
			continue
		}
		for _, sb := range mapsOf(mList(doc, "sendboxes")) {
			if mStr(sb, "status") != "healthy" {
				continue
			}
			s := mStr(sb, "slug")
			cred, cerr := readJSONFile(credPath(oDir, s))
			if cerr != nil {
				continue
			}
			em, pw := mStr(cred, "email"), mStr(cred, "app_password")
			if em == "" || pw == "" {
				continue
			}
			return oDir, s, em, pw, true
		}
	}
	return "", "", "", "", false
}

func buildOperatorMIME(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	b.WriteString("\r\n")
	return []byte(b.String())
}

func runGmailSendOperator(a *cliArgs) int {
	pipeline := a.get("--pipeline")
	subject := strings.TrimSpace(a.get("--subject"))
	body := a.get("--body")
	if f := a.get("--body-file"); body == "" && f != "" {
		raw, err := os.ReadFile(f)
		if err != nil {
			return crmFail(err)
		}
		body = string(raw)
	}
	if pipeline == "" || subject == "" || strings.TrimSpace(body) == "" {
		return crmUsageErr("send-operator needs --pipeline, --subject and --body (or --body-file)")
	}
	if abs, aerr := filepath.Abs(pipeline); aerr == nil {
		pipeline = abs
	}
	operatorEmail := strings.TrimSpace(loadSystemSettings(pipeline).OperatorEmail)
	if operatorEmail == "" {
		return crmFail(fmt.Errorf("no operator_email in system_settings.json — the operator sets it in the web UI at /ui/settings"))
	}
	oDir, slug, boxEmail, appPassword, found := findOperatorSendbox(pipeline)
	if !found {
		return crmFail(fmt.Errorf("no healthy sendbox with credentials found under %s/clients/*/*/outreach/sendboxes", pipeline))
	}
	raw := buildOperatorMIME(boxEmail, operatorEmail, subject, body)
	if err := operatorSMTPSend(boxEmail, appPassword, boxEmail, operatorEmail, raw); err != nil {
		return crmFail(fmt.Errorf("send via %s (%s): %w", slug, boxEmail, err))
	}
	logDir := filepath.Join(pipeline, "automation")
	_ = os.MkdirAll(logDir, 0o755)
	logPath := filepath.Join(logDir, "operator_mail_log.jsonl")
	rec := map[string]any{"ts": nowISO(), "to": operatorEmail, "from": boxEmail,
		"sendbox": slug, "sendbox_dir": oDir, "subject": subject}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err == nil {
		_, _ = f.WriteString(marshalLineJSON(rec) + "\n")
		f.Close()
	}
	return crmOut(map[string]any{"ok": true, "to": operatorEmail, "from": boxEmail,
		"sendbox": slug, "subject": subject, "log": logPath}, 0)
}
