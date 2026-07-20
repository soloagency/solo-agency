package main

// gmail.go — Go port of outreach/tools/gmail_client.py (G3): auth/health/
// quota/send/sync for @gmail.com sendboxes via App Password. SMTP is implicit
// TLS 465 (tls.Dial + net/smtp.Client); IMAP is the minimal client in imap.go.
// The ordered pre-send re-check chain, blocker persistence semantics
// (terminal vs transient), sticky sender, sent_log/sync_log rows and the
// deterministic inbox classifier mirror the Python line by line. Offline
// paths (presend, dry-run, quota, classify) are cross-validated against
// Python; the live SMTP/IMAP path needs a real sendbox (UI_DESIGN §8).

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/smtp"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	gmailSMTPHost = "smtp.gmail.com"
	gmailSMTPPort = 465
	gmailIMAPHost = "imap.gmail.com"
	gmailIMAPPort = 993
)

// --- sendbox config ---------------------------------------------------------------

func sendboxesPath(clientDir string) string {
	return filepath.Join(clientDir, "sendboxes", "sendboxes.json")
}

func credPath(clientDir, slug string) string {
	return filepath.Join(clientDir, "sendboxes", slug, "credentials.json")
}

func loadSendboxesDoc(clientDir string) map[string]any {
	if m, err := readJSONFile(sendboxesPath(clientDir)); err == nil {
		return m
	}
	return map[string]any{"sendboxes": []any{}}
}

func getSendbox(clientDir, slug string) map[string]any {
	for _, sb := range mapsOf(mList(loadSendboxesDoc(clientDir), "sendboxes")) {
		if mStr(sb, "slug") == slug {
			return sb
		}
	}
	return nil
}

func saveSendbox(clientDir string, sb map[string]any) error {
	data := loadSendboxesDoc(clientDir)
	var boxes []any
	for _, b := range mList(data, "sendboxes") {
		if bm, ok := b.(map[string]any); ok && mStr(bm, "slug") == mStr(sb, "slug") {
			continue
		}
		boxes = append(boxes, b)
	}
	boxes = append(boxes, sb)
	data["sendboxes"] = boxes
	return atomicWriteFile(sendboxesPath(clientDir), marshalIndentJSON(data))
}

func loadCredentials(clientDir, slug string) (map[string]any, error) {
	p := credPath(clientDir, slug)
	m, err := readJSONFile(p)
	if err != nil {
		return nil, fmt.Errorf("no credentials for sendbox %s; run `gmail auth` first", pyRepr(slug))
	}
	return m, nil
}

// --- SMTP/IMAP login --------------------------------------------------------------

func smtpLogin(emailAddr, appPassword string) (*smtp.Client, error) {
	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", gmailSMTPHost, gmailSMTPPort),
		&tls.Config{ServerName: gmailSMTPHost})
	if err != nil {
		return nil, err
	}
	conn.SetDeadline(time.Now().Add(30 * time.Second))
	c, err := smtp.NewClient(conn, gmailSMTPHost)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := c.Auth(smtp.PlainAuth("", emailAddr, appPassword, gmailSMTPHost)); err != nil {
		c.Close()
		return nil, fmt.Errorf("SMTPAuthenticationError: %w", err)
	}
	conn.SetDeadline(time.Time{})
	return c, nil
}

func imapLogin(emailAddr, appPassword string) (*imapClient, error) {
	c, err := imapDial(gmailIMAPHost, gmailIMAPPort, 30*time.Second)
	if err != nil {
		return nil, err
	}
	if err := c.login(emailAddr, appPassword); err != nil {
		c.conn.Close()
		return nil, fmt.Errorf("IMAPAuthenticationError: %w", err)
	}
	return c, nil
}

// --- auth / health ----------------------------------------------------------------

// gmailVerifyLogin proves SMTP+IMAP access and returns the mailbox's current
// top UID (the first-sync baseline). Injectable so UI/CLI tests can stub the
// live network round-trip.
var gmailVerifyLogin = func(emailAddr, appPassword string) (int, error) {
	s, err := smtpLogin(emailAddr, appPassword)
	if err != nil {
		return 0, err
	}
	s.Quit()
	m, err := imapLogin(emailAddr, appPassword)
	if err != nil {
		return 0, err
	}
	baseline := 0
	if err := m.selectInbox(); err == nil {
		if uids, err := m.uidSearch("ALL"); err == nil {
			for _, u := range uids {
				if u > baseline {
					baseline = u
				}
			}
		}
	}
	m.logout()
	return baseline, nil
}

func gmailCmdAuth(clientDir, slug, emailAddr string) (map[string]any, error) {
	appPassword := os.Getenv("OUTREACHCRM_APP_PASSWORD")
	if appPassword == "" {
		return nil, fmt.Errorf("set OUTREACHCRM_APP_PASSWORD to the 16-char Gmail App Password (never pass it as a CLI arg)")
	}
	return gmailAuthWithPassword(clientDir, slug, emailAddr, appPassword)
}

// gmailAuthWithPassword is the shared auth core: verify both channels live,
// persist credentials (0600) + the sendbox entry. Also the UI's paste-the-App-
// Password endpoint — the secret flows browser → bridge → Gmail and is never
// echoed into chat, ui_inbox, or any agent-readable queue.
func gmailAuthWithPassword(clientDir, slug, emailAddr, appPassword string) (map[string]any, error) {
	if err := safeID(slug); err != nil {
		return nil, err
	}
	appPassword = strings.ReplaceAll(appPassword, " ", "")
	baseline, err := gmailVerifyLogin(emailAddr, appPassword)
	if err != nil {
		return nil, err
	}
	credDir := filepath.Join(clientDir, "sendboxes", slug)
	if err := os.MkdirAll(credDir, 0o755); err != nil {
		return nil, err
	}
	cpath := credPath(clientDir, slug)
	cred := map[string]any{"email": emailAddr, "app_password": appPassword,
		"smtp_host": gmailSMTPHost, "imap_host": gmailIMAPHost}
	if err := os.WriteFile(cpath, []byte(marshalLineJSON(cred)), 0o600); err != nil {
		return nil, err
	}
	_ = os.Chmod(cpath, 0o600)
	parts := strings.Split(emailAddr, "@")
	domain := strings.ToLower(parts[len(parts)-1])
	existing := getSendbox(clientDir, slug)
	quota := 20
	warmup := "week_1"
	lastSync := ""
	var cursor any = baseline
	if existing != nil {
		quota = mInt(existing, "quota_today", 20)
		if w := mStr(existing, "warmup_stage"); w != "" {
			warmup = w
		}
		lastSync = mStr(existing, "last_successful_sync_ts")
		if c, ok := existing["imap_uid_cursor"]; ok && c != nil {
			cursor = c
		}
	}
	sb := map[string]any{"slug": slug, "auth_mode": "app_password", "email": emailAddr,
		"domain": domain, "quota_today": quota, "warmup_stage": warmup,
		"status": "healthy", "historyId": nil,
		"imap_uid_cursor": cursor, "last_successful_sync_ts": lastSync}
	if err := saveSendbox(clientDir, sb); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "sendbox": slug, "email": emailAddr, "smtp": "ok", "imap": "ok",
		"quota_today": quota, "warmup_stage": warmup}, nil
}

func gmailCmdHealth(clientDir, slug string) (map[string]any, error) {
	cred, err := loadCredentials(clientDir, slug)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"sendbox": slug, "email": cred["email"], "smtp": "?", "imap": "?", "status": "healthy"}
	if s, err := smtpLogin(mStr(cred, "email"), mStr(cred, "app_password")); err != nil {
		out["smtp"] = "fail: " + errClassName(err)
		out["status"] = "needs_reauth"
	} else {
		s.Quit()
		out["smtp"] = "ok"
	}
	if m, err := imapLogin(mStr(cred, "email"), mStr(cred, "app_password")); err != nil {
		out["imap"] = "fail: " + errClassName(err)
		out["status"] = "needs_reauth"
	} else {
		m.logout()
		out["imap"] = "ok"
	}
	if sb := getSendbox(clientDir, slug); sb != nil && mStr(sb, "status") != out["status"] {
		sb["status"] = out["status"]
		if err := saveSendbox(clientDir, sb); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func errClassName(err error) string {
	msg := err.Error()
	if i := strings.Index(msg, ":"); i > 0 && !strings.ContainsAny(msg[:i], " ") {
		return msg[:i]
	}
	return "Error"
}

// --- quota ------------------------------------------------------------------------

func gmailSentLogFiles(clientDir, campaignSlug string) []string {
	base := filepath.Join(clientDir, "campaigns", campaignSlug, "sent")
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	var out []string
	for _, month := range names {
		p := filepath.Join(base, month, "sent_log.jsonl")
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			out = append(out, p)
		}
	}
	return out
}

func gmailSentCountToday(clientDir, slug, day string) int {
	n := 0
	campRoot := filepath.Join(clientDir, "campaigns")
	entries, err := os.ReadDir(campRoot)
	if err != nil {
		return 0
	}
	for _, e := range entries {
		for _, p := range gmailSentLogFiles(clientDir, e.Name()) {
			for _, r := range readJSONLines(p) {
				sa := mStr(r, "sent_at")
				if len(sa) >= 10 {
					sa = sa[:10]
				}
				if mStr(r, "sendbox") == slug && sa == day && mStr(r, "rfc_message_id") != "" {
					n++
				}
			}
		}
	}
	return n
}

func gmailCmdQuota(clientDir, slug, day string) map[string]any {
	sb := getSendbox(clientDir, slug)
	cap := 0
	if sb != nil {
		cap = mInt(sb, "quota_today", 0)
	}
	sent := gmailSentCountToday(clientDir, slug, day)
	store := newCrmStore(clientDir)
	reserved := store.a.reservationCount(slug, day)
	used := sent
	if reserved > used {
		used = reserved
	}
	remaining := cap - used
	if remaining < 0 {
		remaining = 0
	}
	return map[string]any{"sendbox": slug, "day": day, "cap": cap, "sent": sent,
		"reserved": reserved, "remaining": remaining}
}

// --- send -------------------------------------------------------------------------

func headerSafe(v string) bool {
	return !strings.ContainsAny(v, "\n\r")
}

func identityPath(clientDir string) string {
	return filepath.Join(clientDir, "config", "sending_identity.json")
}

func loadSendingIdentity(clientDir string) map[string]any {
	if m, err := readJSONFile(identityPath(clientDir)); err == nil {
		return m
	}
	return map[string]any{}
}

func complianceFooter(identity map[string]any) string {
	addr := strings.TrimSpace(mStr(identity, "physical_mailing_address"))
	if addr == "" {
		return ""
	}
	name := strings.TrimSpace(mStr(identity, "from_name"))
	optout := strings.TrimSpace(mStr(identity, "unsubscribe_text"))
	if optout == "" {
		optout = `Don't want these emails? Reply "unsubscribe" and we'll stop.`
	}
	lines := []string{"-- "}
	if name != "" {
		lines = append(lines, name)
	}
	lines = append(lines, addr, optout)
	return strings.Join(lines, "\n")
}

var subjectReplyRe = regexp.MustCompile(`(?i)^\s*re\s*:`)
var subjectStep1Re = regexp.MustCompile(`(?i)^\s*(re|fwd)\s*:`)

// gmailPresendCheck is the ordered in-code pre-send re-check (DESIGN §10/§16).
func gmailPresendCheck(store *crmStore, clientDir string, sb, draft map[string]any, day string,
	reserve bool) (bool, string, string, error) {
	slug := mStr(sb, "slug")
	leadID := store.resolve(mStr(draft, "lead_id"))
	contact := store.getContact(leadID)
	if contact == nil {
		return false, "contact_not_found", "", nil
	}
	ids := mMap(contact, "identities")
	var emails, phones, socials []string
	for _, e := range mapsOf(mList(ids, "emails")) {
		if a := mStr(e, "address"); a != "" {
			emails = append(emails, a)
		}
	}
	for _, p := range mapsOf(mList(ids, "phones")) {
		if n := mStr(p, "number"); n != "" {
			phones = append(phones, n)
		}
	}
	for _, v := range mMap(ids, "socials") {
		if s, ok := v.(string); ok && s != "" {
			socials = append(socials, s)
		}
	}
	toAddr := strings.TrimSpace(mStr(draft, "to"))
	normSet := map[string]bool{}
	for _, e := range emails {
		normSet[normalizeEmail(e)] = true
	}
	if toAddr == "" || !normSet[normalizeEmail(toAddr)] {
		return false, "recipient_not_a_contact_identity", "", nil
	}
	if !headerSafe(mStr(draft, "subject")) || !headerSafe(toAddr) {
		return false, "invalid_draft_headers", "", nil
	}
	checkAddrs := map[string]bool{toAddr: true}
	for _, e := range emails {
		checkAddrs[e] = true
	}
	for a := range checkAddrs {
		if store.isSuppressed(a, "", nil) != nil {
			return false, "suppressed", "", nil
		}
	}
	for _, ph := range phones {
		if store.isSuppressed("", ph, nil) != nil {
			return false, "suppressed", "", nil
		}
	}
	if len(socials) > 0 && store.isSuppressed("", "", socials) != nil {
		return false, "suppressed", "", nil
	}
	emailStatus := mStr(mMap(mMap(contact, "channels"), "email"), "status")
	if emailStatus == "opted_out" || emailStatus == "bounced" {
		return false, "email_channel_not_usable", "", nil
	}
	if complianceFooter(loadSendingIdentity(clientDir)) == "" {
		return false, "missing_physical_address", "", nil
	}
	var prim map[string]any
	for _, e := range mapsOf(mList(ids, "emails")) {
		if mBool(e, "is_primary") {
			prim = e
			break
		}
	}
	if prim != nil && mStr(prim, "status") == "guessed_only" && !mBool(draft, "guessed_approved") {
		return false, "guessed_email_needs_approval", "", nil
	}
	if mStr(contact, "sequence_state") == "frozen" {
		return false, "sequence_frozen", "", nil
	}
	step := mInt(draft, "step", 1)
	if step == 1 && subjectStep1Re.MatchString(mStr(draft, "subject")) {
		return false, "step1_subject_looks_like_reply", "", nil
	}
	if gmailAlreadySent(clientDir, mStr(draft, "campaign_slug"), leadID, step) {
		return false, "already_sent", "", nil
	}
	assigned := mStr(contact, "assigned_sendbox")
	if step > 1 && assigned != "" && assigned != slug {
		return false, "wrong_sendbox_for_sticky_sender", "", nil
	}
	cap := mInt(sb, "quota_today", 0)
	if gmailSentCountToday(clientDir, slug, day) >= cap {
		return false, "quota_exhausted", "", nil
	}
	if !reserve {
		return true, "ok", "", nil
	}
	token, err := store.a.reserve(slug, day, cap)
	if err != nil {
		return false, "", "", err
	}
	if token == "" {
		return false, "quota_exhausted", "", nil
	}
	return true, "ok", token, nil
}

func gmailAlreadySent(clientDir, campaignSlug, leadID string, step int) bool {
	for _, p := range gmailSentLogFiles(clientDir, campaignSlug) {
		for _, r := range readJSONLines(p) {
			if mStr(r, "lead_id") == leadID && mInt(r, "step", 0) == step && mStr(r, "rfc_message_id") != "" {
				return true
			}
		}
	}
	return false
}

func gmailMkToken() string {
	var b [6]byte
	rand.Read(b[:])
	return fmt.Sprintf("%x", b)
}

type mimeMessage struct {
	Headers    [][2]string
	Body       string
	rawSubject string
}

func encodeHeaderWord(s string) string {
	if isASCII(s) {
		return s
	}
	return mime.QEncoding.Encode("utf-8", s)
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 0x7E || s[i] < 0x20 {
			return false
		}
	}
	return true
}

func formatAddr(name, addr string) string {
	if name == "" {
		return addr
	}
	a := mail.Address{Name: name, Address: addr}
	return a.String()
}

// gmailBuildMIME mirrors build_mime: headers + compliance footer + optional
// html alternative; mutates draft["token"] like the Python does.
func gmailBuildMIME(sb, draft map[string]any, rfcMessageID, threadRefs, footer string) (*mimeMessage, error) {
	fromName := mStr(draft, "from_name")
	if fromName == "" {
		fromName = mStr(sb, "from_name")
	}
	sbEmail := mStr(sb, "email")
	subject := mStr(draft, "subject")
	step := mInt(draft, "step", 1)
	if step > 1 && threadRefs != "" && !subjectReplyRe.MatchString(subject) {
		subject = "Re: " + subject
	}
	token := mStr(draft, "token")
	if token == "" {
		token = gmailMkToken()
	}
	local, dom, ok := strings.Cut(sbEmail, "@")
	if !ok {
		return nil, fmt.Errorf("bad sendbox email %q", sbEmail)
	}
	if !headerSafe(subject) || !headerSafe(mStr(draft, "to")) {
		return nil, fmt.Errorf("header injection")
	}
	msg := &mimeMessage{}
	push := func(k, v string) { msg.Headers = append(msg.Headers, [2]string{k, v}) }
	push("From", formatAddr(fromName, sbEmail))
	push("To", mStr(draft, "to"))
	push("Subject", encodeHeaderWord(subject))
	push("Message-ID", rfcMessageID)
	push("Date", time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700"))
	push("List-Unsubscribe", fmt.Sprintf("<mailto:%s+unsub-%s@%s?subject=unsubscribe>", local, token, dom))
	if threadRefs != "" {
		push("In-Reply-To", threadRefs)
		push("References", threadRefs)
	}
	body := mStr(draft, "body_text")
	if footer != "" {
		body = strings.TrimRight(body, " \t\n") + "\n\n" + footer
	}
	if mStr(draft, "tracking") == "pixel_and_links" && mStr(draft, "body_html") != "" {
		htmlBody := mStr(draft, "body_html")
		if footer != "" {
			htmlBody = htmlBody + "<br><br>" + strings.ReplaceAll(pyHTMLEscape(footer, false), "\n", "<br>")
		}
		boundary := "==bnd-" + gmailMkToken()
		push("MIME-Version", "1.0")
		push("Content-Type", fmt.Sprintf(`multipart/alternative; boundary="%s"`, boundary))
		var sb2 strings.Builder
		writePart := func(ctype, content string) {
			sb2.WriteString("--" + boundary + "\r\n")
			sb2.WriteString("Content-Type: " + ctype + "; charset=\"utf-8\"\r\n")
			sb2.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
			w := quotedprintable.NewWriter(&sb2)
			w.Write([]byte(content))
			w.Close()
			sb2.WriteString("\r\n")
		}
		writePart("text/plain", body)
		writePart("text/html", htmlBody)
		sb2.WriteString("--" + boundary + "--\r\n")
		msg.Body = sb2.String()
	} else {
		push("MIME-Version", "1.0")
		push("Content-Type", `text/plain; charset="utf-8"`)
		push("Content-Transfer-Encoding", "quoted-printable")
		var sb2 strings.Builder
		w := quotedprintable.NewWriter(&sb2)
		w.Write([]byte(body))
		w.Close()
		msg.Body = sb2.String()
	}
	draft["token"] = token
	msg.subjectRaw(subject)
	return msg, nil
}

// subjectRaw keeps the human-readable subject available for dry-run output.
func (m *mimeMessage) subjectRaw(s string) { m.rawSubject = s }

func (m *mimeMessage) header(name string) string {
	for _, h := range m.Headers {
		if h[0] == name {
			return h[1]
		}
	}
	return ""
}

func (m *mimeMessage) bytes() []byte {
	var sb strings.Builder
	for _, h := range m.Headers {
		sb.WriteString(h[0] + ": " + h[1] + "\r\n")
	}
	sb.WriteString("\r\n")
	sb.WriteString(m.Body)
	return []byte(sb.String())
}

var terminalBlockers = map[string]bool{"suppressed": true, "email_channel_not_usable": true,
	"sequence_frozen": true, "recipient_not_a_contact_identity": true, "contact_not_found": true,
	"already_sent": true, "invalid_draft_headers": true, "step1_subject_looks_like_reply": true}

func persistSendBlocker(draftPath string, draft map[string]any, reason string) {
	if mStr(draft, "status") != "approved" {
		return
	}
	draft["blocker"] = reason
	draft["blocked_at"] = nowISO()
	if terminalBlockers[reason] {
		draft["status"] = "blocked"
	}
	_ = os.WriteFile(draftPath, []byte(marshalIndentJSON(draft)), 0o644)
}

func gmailCmdSend(clientDir, draftPath string, dryRun bool) (map[string]any, error) {
	draft, err := readJSONFile(draftPath)
	if err != nil {
		return nil, err
	}
	store := newCrmStore(clientDir)
	slug := mStr(draft, "sendbox")
	sb := getSendbox(clientDir, slug)
	if sb == nil {
		if !dryRun {
			persistSendBlocker(draftPath, draft, "sendbox_not_configured")
		}
		return map[string]any{"ok": false, "blocker": "sendbox_not_configured", "sendbox": slug}, nil
	}
	if mStr(sb, "status") != "healthy" {
		blocker := "sendbox_" + mStr(sb, "status")
		if !dryRun {
			persistSendBlocker(draftPath, draft, blocker)
		}
		return map[string]any{"ok": false, "blocker": blocker, "sendbox": slug}, nil
	}
	if mStr(draft, "status") != "approved" {
		return map[string]any{"ok": false, "blocker": "draft_not_approved",
			"draft_status": draft["status"]}, nil
	}
	day := todayStr("")
	leadID := store.resolve(mStr(draft, "lead_id"))
	step := mInt(draft, "step", 1)
	ok, reason, token, err := gmailPresendCheck(store, clientDir, sb, draft, day, !dryRun)
	if err != nil {
		return nil, err
	}
	if !ok {
		if !dryRun {
			persistSendBlocker(draftPath, draft, reason)
		}
		return map[string]any{"ok": false, "blocker": reason, "lead_id": draft["lead_id"]}, nil
	}
	threadRefs := ""
	if step > 1 {
		threadRefs = gmailPriorMessageID(clientDir, mStr(draft, "campaign_slug"), leadID)
	}
	var rb [16]byte
	rand.Read(rb[:])
	parts := strings.SplitN(mStr(sb, "email"), "@", 2)
	rfcMessageID := fmt.Sprintf("<%x@%s>", rb, parts[len(parts)-1])
	footer := complianceFooter(loadSendingIdentity(clientDir))
	msg, err := gmailBuildMIME(sb, draft, rfcMessageID, threadRefs, footer)
	if err != nil {
		if token != "" {
			store.a.release(slug, day, token)
		}
		if !dryRun {
			persistSendBlocker(draftPath, draft, "invalid_draft_headers")
		}
		return map[string]any{"ok": false, "blocker": "invalid_draft_headers", "error": err.Error()}, nil
	}
	if dryRun {
		return map[string]any{"ok": true, "dry_run": true, "would_send_to": draft["to"],
			"sendbox": slug, "subject": msg.rawSubject, "rfc_message_id": rfcMessageID,
			"list_unsubscribe": msg.header("List-Unsubscribe"),
			"note":             "dry-run: no quota reserved, nothing sent"}, nil
	}
	cred, err := loadCredentials(clientDir, slug)
	if err != nil {
		return nil, err
	}
	sendErr := gmailSMTPSend(mStr(cred, "email"), mStr(cred, "app_password"),
		mStr(sb, "email"), mStr(draft, "to"), msg.bytes())
	if sendErr != nil {
		if token != "" {
			store.a.release(slug, day, token)
		}
		needsReauth := strings.Contains(strings.ToLower(errClassName(sendErr)), "auth")
		blocker := "smtp_send_failed"
		if needsReauth {
			blocker = "needs_reauth"
			sb["status"] = "needs_reauth"
			_ = saveSendbox(clientDir, sb)
		}
		persistSendBlocker(draftPath, draft, blocker)
		return map[string]any{"ok": false, "blocker": blocker, "error": errClassName(sendErr)}, nil
	}
	sentAt := nowISO()
	links := draft["links"]
	if links == nil {
		links = map[string]any{}
	}
	if err := gmailAppendSentLog(clientDir, mStr(draft, "campaign_slug"), map[string]any{
		"lead_id": leadID, "campaign": draft["campaign_slug"], "step": step,
		"sendbox": slug, "provider_id": "", "thread_id": threadRefs,
		"rfc_message_id": rfcMessageID, "token": mStr(draft, "token"),
		"links": links, "sent_at": sentAt,
	}); err != nil {
		return nil, err
	}
	act, err := store.logActivity("email_sent", leadID, fmt.Sprintf("sent step %d via %s", step, slug),
		"agent", nil, map[string]any{"message_id": rfcMessageID})
	if err != nil {
		return nil, err
	}
	if step == 1 {
		ct := store.getContact(leadID)
		if mStr(ct, "assigned_sendbox") == "" {
			if _, err := store.setContact(leadID, map[string]any{"assigned_sendbox": slug}); err != nil {
				return nil, err
			}
		}
	}
	draft["status"] = "sent"
	draft["decided_at"] = sentAt
	if err := os.WriteFile(draftPath, []byte(marshalIndentJSON(draft)), 0o644); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "sent_to": draft["to"], "sendbox": slug,
		"rfc_message_id": rfcMessageID, "activity_seq": act["seq"], "sent_at": sentAt}, nil
}

func gmailSMTPSend(authEmail, appPassword, from, to string, raw []byte) error {
	c, err := smtpLogin(authEmail, appPassword)
	if err != nil {
		return err
	}
	defer c.Close()
	if err := c.Mail(from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(raw); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

func gmailAppendSentLog(clientDir, campaignSlug string, record map[string]any) error {
	d := filepath.Join(clientDir, "campaigns", campaignSlug, "sent", monthStr(""))
	if err := os.MkdirAll(d, 0o755); err != nil {
		return err
	}
	p := filepath.Join(d, "sent_log.jsonl")
	rec := map[string]any{"seq": appendJSONLSeq(p), "ts": nowISO()}
	for k, v := range record {
		rec[k] = v
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(marshalLineJSON(rec) + "\n")
	return err
}

func gmailPriorMessageID(clientDir, campaignSlug, leadID string) string {
	last := ""
	for _, p := range gmailSentLogFiles(clientDir, campaignSlug) {
		for _, r := range readJSONLines(p) {
			if mStr(r, "lead_id") == leadID && mStr(r, "rfc_message_id") != "" {
				last = mStr(r, "rfc_message_id")
			}
		}
	}
	return last
}

// --- parsed-message model + classifier ---------------------------------------------

type parsedPart struct {
	ContentType string
	FullType    string // raw Content-Type header value
	Header      mail.Header
	Body        []byte
	Parts       []*parsedPart
}

func parseEmailMessage(raw []byte) (*parsedPart, error) {
	m, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		return nil, err
	}
	return parsePart(m.Header, m.Body)
}

func parsePart(h mail.Header, body io.Reader) (*parsedPart, error) {
	ctRaw := h.Get("Content-Type")
	ctype := "text/plain"
	var params map[string]string
	if ctRaw != "" {
		if mt, ps, err := mime.ParseMediaType(ctRaw); err == nil {
			ctype = mt
			params = ps
		}
	}
	p := &parsedPart{ContentType: strings.ToLower(ctype), FullType: ctRaw, Header: h}
	if strings.HasPrefix(p.ContentType, "multipart/") && params["boundary"] != "" {
		mr := multipart.NewReader(body, params["boundary"])
		for {
			sp, err := mr.NextPart()
			if err != nil {
				break
			}
			data, _ := io.ReadAll(sp)
			child, perr := parsePart(mail.Header(sp.Header), strings.NewReader(string(data)))
			if perr == nil {
				p.Parts = append(p.Parts, child)
			}
		}
		return p, nil
	}
	data, _ := io.ReadAll(body)
	p.Body = decodeCTE(h, data)
	return p, nil
}

func decodeCTE(h mail.Header, data []byte) []byte {
	switch strings.ToLower(strings.TrimSpace(h.Get("Content-Transfer-Encoding"))) {
	case "base64":
		if out, err := io.ReadAll(newB64Reader(string(data))); err == nil {
			return out
		}
	case "quoted-printable":
		if out, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(string(data)))); err == nil {
			return out
		}
	}
	return data
}

func newB64Reader(s string) io.Reader {
	clean := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, s)
	return base64.NewDecoder(base64.StdEncoding, strings.NewReader(clean))
}

func (p *parsedPart) walk(fn func(*parsedPart)) {
	fn(p)
	for _, c := range p.Parts {
		c.walk(fn)
	}
}

func (p *parsedPart) get(name string) string {
	v := p.Header.Get(name)
	if v == "" {
		return ""
	}
	dec := new(mime.WordDecoder)
	if d, err := dec.DecodeHeader(v); err == nil {
		return d
	}
	return v
}

func (p *parsedPart) plainBody() string {
	if len(p.Parts) > 0 {
		var found string
		p.walk(func(x *parsedPart) {
			if found == "" && x.ContentType == "text/plain" && len(x.Parts) == 0 {
				found = string(x.Body)
			}
		})
		return found
	}
	return string(p.Body)
}

var midRe = regexp.MustCompile(`<[^>]+>`)
var oooRe = regexp.MustCompile(`(?i)out of (the )?office|auto[- ]?reply|automatic reply`)
var dsnStatusRe = regexp.MustCompile(`([245]\.\d+\.\d+)`)
var dsnFinalRcptRe = regexp.MustCompile(`(?im)^\s*(?:Final-Recipient|Original-Recipient)\s*:\s*(?:rfc822;)?\s*([^\s;]+@[^\s;]+)`)
var dsnStatusLineRe = regexp.MustCompile(`(?im)^\s*Status\s*:\s*([245]\.\d+\.\d+)`)
var bodyMidRe = regexp.MustCompile(`(?im)^\s*(?:X-Google-Original-Message-ID|Message-ID)\s*:\s*(<[^>]+>)`)
var hard5xxRe = regexp.MustCompile(`\b5\d\d[\s-]`)
var hardEnhancedRe = regexp.MustCompile(`\b5\.\d+\.\d+\b`)
var emailInTextRe = regexp.MustCompile(`([^\s;]+@[^\s;]+)`)

func gmailClassifyMessage(msg *parsedPart, sendboxEmail string, knownMessageIDs map[string]map[string]any,
	fromResolver func(string) map[string]any) map[string]any {
	fromHdr := strings.ToLower(msg.get("From"))
	ctype := msg.ContentType
	toAll := strings.Join([]string{msg.get("To"), msg.get("Delivered-To"), msg.get("X-Original-To")}, " ")
	subject := msg.get("Subject")

	isDSN := strings.Contains(fromHdr, "mailer-daemon") || strings.Contains(fromHdr, "postmaster") ||
		ctype == "multipart/report" || strings.Contains(strings.ToLower(msg.Header.Get("Content-Type")), "report-type=delivery-status")
	if isDSN {
		d := gmailDSNDetails(msg)
		return map[string]any{"kind": "bounce", "hard": d["hard"],
			"bounced_message_id": d["original_mid"], "final_recipient": d["final_recipient"]}
	}
	if strings.HasPrefix(strings.ToLower(msg.get("Auto-Submitted")), "auto") || oooRe.MatchString(subject) {
		return map[string]any{"kind": "auto_reply_ooo"}
	}
	local, dom, _ := strings.Cut(sendboxEmail, "@")
	unsubRe := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(local) + `\+unsub-([A-Za-z0-9]+)@` + regexp.QuoteMeta(dom))
	if m := unsubRe.FindStringSubmatch(toAll); m != nil {
		return map[string]any{"kind": "unsubscribe", "token": m[1]}
	}
	refs := msg.get("In-Reply-To") + " " + msg.get("References")
	for _, mid := range midRe.FindAllString(refs, -1) {
		if info, ok := knownMessageIDs[mid]; ok {
			return map[string]any{"kind": "campaign_reply", "lead_id": info["lead_id"],
				"campaign": info["campaign"], "in_reply_to": mid}
		}
	}
	addr := ""
	if a, err := mail.ParseAddress(msg.get("From")); err == nil {
		addr = strings.ToLower(a.Address)
	} else {
		addr = strings.ToLower(strings.TrimSpace(msg.get("From")))
	}
	if fromResolver != nil {
		if info := fromResolver(addr); info != nil && mStr(info, "lead_id") != "" {
			return map[string]any{"kind": "campaign_reply", "lead_id": info["lead_id"],
				"campaign": info["campaign"], "matched_by": "from_address"}
		}
	}
	return map[string]any{"kind": "contact_or_personal", "from": addr}
}

func gmailDSNDetails(msg *parsedPart) map[string]any {
	originalMid, finalRecipient, status := "", "", ""
	msg.walk(func(part *parsedPart) {
		switch part.ContentType {
		case "message/delivery-status":
			txt := string(part.Body)
			if finalRecipient == "" {
				if m := dsnFinalRcptRe.FindStringSubmatch(txt); m != nil {
					finalRecipient = strings.ToLower(strings.Trim(strings.TrimSpace(m[1]), "<>"))
				}
			}
			if status == "" {
				if m := dsnStatusLineRe.FindStringSubmatch(txt); m != nil {
					status = m[1]
				}
			}
		case "message/rfc822", "text/rfc822-headers":
			if originalMid == "" {
				sub, err := parseEmailMessage(part.Body)
				if err == nil {
					if mid := strings.TrimSpace(sub.Header.Get("Message-ID")); mid != "" {
						originalMid = mid
					}
				} else if m := bodyMidRe.FindStringSubmatch(string(part.Body)); m != nil {
					originalMid = m[1]
				}
			}
		}
	})
	body := msg.plainBody()
	if originalMid == "" {
		if m := bodyMidRe.FindStringSubmatch(body); m != nil {
			originalMid = m[1]
		}
	}
	hard := false
	if status != "" {
		hard = strings.HasPrefix(status, "5")
	} else {
		hard = hard5xxRe.MatchString(body) || hardEnhancedRe.MatchString(body)
	}
	return map[string]any{"original_mid": originalMid, "final_recipient": finalRecipient,
		"hard": hard, "status": status}
}

func gmailLoadKnownMessageIDs(clientDir string) map[string]map[string]any {
	out := map[string]map[string]any{}
	campRoot := filepath.Join(clientDir, "campaigns")
	entries, err := os.ReadDir(campRoot)
	if err != nil {
		return out
	}
	for _, e := range entries {
		for _, p := range gmailSentLogFiles(clientDir, e.Name()) {
			for _, r := range readJSONLines(p) {
				if mid := mStr(r, "rfc_message_id"); mid != "" {
					out[mid] = map[string]any{"lead_id": r["lead_id"], "campaign": r["campaign"]}
				}
			}
		}
	}
	return out
}

// --- sync -------------------------------------------------------------------------

func gmailCmdSync(clientDir, slug string, maxMsgs int) (map[string]any, error) {
	cred, err := loadCredentials(clientDir, slug)
	if err != nil {
		return nil, err
	}
	sb := getSendbox(clientDir, slug)
	if sb == nil {
		return nil, fmt.Errorf("sendbox %s not configured", pyRepr(slug))
	}
	store := newCrmStore(clientDir)
	known := gmailLoadKnownMessageIDs(clientDir)
	fromResolver := func(addr string) map[string]any {
		if lead := store.a.findByIdentity("email", normalizeEmail(addr)); lead != "" {
			return map[string]any{"lead_id": lead}
		}
		return nil
	}
	m, err := imapLogin(mStr(cred, "email"), mStr(cred, "app_password"))
	if err != nil {
		return nil, err
	}
	if err := m.selectInbox(); err != nil {
		m.logout()
		return nil, err
	}
	rawCursor, hasCursor := sb["imap_uid_cursor"]
	if !hasCursor || rawCursor == nil {
		uids, _ := m.uidSearch("ALL")
		baseline := 0
		for _, u := range uids {
			if u > baseline {
				baseline = u
			}
		}
		m.logout()
		sb["imap_uid_cursor"] = baseline
		sb["last_successful_sync_ts"] = nowISO()
		if err := saveSendbox(clientDir, sb); err != nil {
			return nil, err
		}
		return map[string]any{"sendbox": slug, "checked": 0, "cursor": baseline, "baseline_set": true,
			"counts": map[string]any{"bounce": 0, "auto_reply_ooo": 0, "unsubscribe": 0,
				"campaign_reply": 0, "personal": 0},
			"replies_untriaged": []any{},
			"note":              "first sync: baselined to current mailbox top; only new mail is processed from here"}, nil
	}
	cursor := int(asFloat(rawCursor, 0))
	found, err := m.uidSearch(fmt.Sprintf("UID %d:*", cursor+1))
	if err != nil {
		m.logout()
		return nil, err
	}
	var uids []int
	for _, u := range found {
		if u > cursor {
			uids = append(uids, u)
		}
	}
	sort.Ints(uids)
	batch := uids
	if len(batch) > maxMsgs {
		batch = batch[:maxMsgs]
	}
	results := map[string]int{"bounce": 0, "auto_reply_ooo": 0, "unsubscribe": 0,
		"campaign_reply": 0, "personal": 0}
	replies := []any{}
	newCursor := cursor
	for _, uid := range batch {
		raw, err := m.uidFetchRFC822(uid)
		if err != nil || raw == nil {
			break
		}
		msg, err := parseEmailMessage(raw)
		if err != nil {
			break
		}
		cls := gmailClassifyMessage(msg, mStr(cred, "email"), known, fromResolver)
		switch mStr(cls, "kind") {
		case "bounce":
			lead := ""
			if info, ok := known[mStr(cls, "bounced_message_id")]; ok {
				lead = mStr(info, "lead_id")
			}
			if lead == "" && mStr(cls, "final_recipient") != "" {
				lead = store.a.findByIdentity("email", normalizeEmail(mStr(cls, "final_recipient")))
			}
			if lead != "" {
				if truthy(cls["hard"]) {
					if _, err := store.suppressContact(lead, "hard_bounce", "", "rule"); err != nil {
						m.logout()
						return nil, err
					}
					if _, err := store.setContact(lead, map[string]any{"sequence_state": "frozen"}); err != nil {
						m.logout()
						return nil, err
					}
				}
				kind := "soft"
				if truthy(cls["hard"]) {
					kind = "hard"
				}
				if _, err := store.logActivity("email_bounce", lead, kind+" bounce", "rule", nil, nil); err != nil {
					m.logout()
					return nil, err
				}
			}
			results["bounce"]++
		case "auto_reply_ooo":
			results["auto_reply_ooo"]++
		case "unsubscribe":
			info := gmailLookupToken(clientDir, mStr(cls, "token"))
			if info != nil && mStr(info, "lead_id") != "" {
				lead := mStr(info, "lead_id")
				if _, err := store.suppressContact(lead, "unsubscribe", "", "rule"); err != nil {
					m.logout()
					return nil, err
				}
				if _, err := store.setContact(lead, map[string]any{"sequence_state": "frozen"}); err != nil {
					m.logout()
					return nil, err
				}
				if _, err := store.logActivity("unsubscribe", lead, "unsubscribed via mailto alias", "rule", nil, nil); err != nil {
					m.logout()
					return nil, err
				}
			}
			results["unsubscribe"]++
		case "campaign_reply":
			lead := mStr(cls, "lead_id")
			if lead != "" {
				if _, err := store.setContact(lead, map[string]any{"sequence_state": "frozen"}); err != nil {
					m.logout()
					return nil, err
				}
				act, err := store.logActivity("email_reply", lead, "campaign reply (untriaged)", "rule",
					nil, map[string]any{"message_id": msg.Header.Get("Message-ID")})
				if err != nil {
					m.logout()
					return nil, err
				}
				fromAddr := ""
				if a, err := mail.ParseAddress(msg.get("From")); err == nil {
					fromAddr = a.Address
				}
				matchedBy := mStr(cls, "matched_by")
				if matchedBy == "" {
					matchedBy = "thread"
				}
				replies = append(replies, map[string]any{"lead_id": lead, "campaign": cls["campaign"],
					"activity_seq": act["seq"], "subject": msg.get("Subject"),
					"from": fromAddr, "matched_by": matchedBy})
			}
			results["campaign_reply"]++
		default:
			results["personal"]++
		}
		newCursor = uid
	}
	m.logout()
	sb["imap_uid_cursor"] = newCursor
	sb["last_successful_sync_ts"] = nowISO()
	if err := saveSendbox(clientDir, sb); err != nil {
		return nil, err
	}
	processed := 0
	countsAny := map[string]any{}
	for k, v := range results {
		processed += v
		countsAny[k] = v
	}
	backlog := len(uids) - processed
	if backlog < 0 {
		backlog = 0
	}
	syncRec := map[string]any{"checked": processed, "backlog_remaining": backlog}
	for k, v := range results {
		syncRec[k] = v
	}
	if err := gmailAppendSyncLog(clientDir, slug, syncRec); err != nil {
		return nil, err
	}
	return map[string]any{"sendbox": slug, "checked": processed, "cursor": newCursor,
		"backlog_remaining": backlog, "counts": countsAny, "replies_untriaged": replies}, nil
}

func gmailLookupToken(clientDir, token string) map[string]any {
	campRoot := filepath.Join(clientDir, "campaigns")
	entries, err := os.ReadDir(campRoot)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		for _, p := range gmailSentLogFiles(clientDir, e.Name()) {
			for _, r := range readJSONLines(p) {
				if mStr(r, "token") == token {
					return map[string]any{"lead_id": r["lead_id"], "campaign": r["campaign"]}
				}
			}
		}
	}
	return nil
}

func gmailAppendSyncLog(clientDir, slug string, record map[string]any) error {
	d := filepath.Join(clientDir, "inbox_sync", monthStr(""))
	if err := os.MkdirAll(d, 0o755); err != nil {
		return err
	}
	p := filepath.Join(d, "sync_log.jsonl")
	rec := map[string]any{"seq": appendJSONLSeq(p), "ts": nowISO(), "sendbox": slug}
	for k, v := range record {
		rec[k] = v
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(marshalLineJSON(rec) + "\n")
	return err
}

// --- CLI: tool gmail --------------------------------------------------------------

func runGmailCLI(args []string) int {
	valueFlags := map[string]bool{"--client-dir": true, "--sendbox": true, "--email": true,
		"--day": true, "--draft": true, "--max": true}
	boolFlags := map[string]bool{"--dry-run": true}
	a, err := parseCLIArgs(args, valueFlags, boolFlags)
	if err != nil {
		return crmUsageErr(err.Error())
	}
	if len(a.pos) == 0 {
		return crmUsageErr("a subcommand is required (auth | health | quota | send | sync)")
	}
	cd := a.get("--client-dir")
	if cd == "" {
		return crmUsageErr("--client-dir is required")
	}
	var out map[string]any
	var cmdErr error
	switch a.pos[0] {
	case "auth":
		if a.get("--sendbox") == "" || a.get("--email") == "" {
			return crmUsageErr("auth needs --sendbox and --email")
		}
		out, cmdErr = gmailCmdAuth(cd, a.get("--sendbox"), a.get("--email"))
	case "health":
		if a.get("--sendbox") == "" {
			return crmUsageErr("health needs --sendbox")
		}
		out, cmdErr = gmailCmdHealth(cd, a.get("--sendbox"))
	case "quota":
		if a.get("--sendbox") == "" {
			return crmUsageErr("quota needs --sendbox")
		}
		out = gmailCmdQuota(cd, a.get("--sendbox"), strOr(a.get("--day"), todayStr("")))
	case "send":
		if a.get("--draft") == "" {
			return crmUsageErr("send needs --draft")
		}
		out, cmdErr = gmailCmdSend(cd, a.get("--draft"), a.bools["--dry-run"])
	case "sync":
		if a.get("--sendbox") == "" {
			return crmUsageErr("sync needs --sendbox")
		}
		out, cmdErr = gmailCmdSync(cd, a.get("--sendbox"), a.getInt("--max", 100))
	default:
		return crmUsageErr("unknown gmail subcommand " + a.pos[0])
	}
	if cmdErr != nil {
		fmt.Fprintln(os.Stderr, cmdErr.Error())
		return 1
	}
	fmt.Println(marshalIndentJSON(out))
	if ok, exists := out["ok"]; exists && ok == false {
		return 1
	}
	return 0
}

var _ = json.Marshal
