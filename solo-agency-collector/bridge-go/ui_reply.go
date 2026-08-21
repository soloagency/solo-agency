package main

// ui_reply.go — the operator answers an inbound reply from the contact page.
//
// The bridge does not reimplement sending. It writes the operator's words as a
// draft and calls gmailCmdSend, the same function `tool gmail send` calls, so the
// ordered chain in gmailPresendCheck runs unshortened. This lane ADDS gates; it
// removes none.
//
// The draft goes to a lane of its own, campaigns/{slug}/outbox/operator_reply/,
// which nothing else reads or sweeps. That is what keeps it out of the daily
// run's outbox/approved/ pass and out of every draft counter, so an operator's
// half-finished reply can never be mailed by another process.
//
// Refusals are answers, not errors: the operator gets a reason and the page stays
// usable. The lane may only ANSWER someone who wrote to us — it can never
// originate, bump, or batch.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// removeIntentRe: high-precision phrases only, matched against the part of the
// inbound message the person actually typed. A false positive costs a refusal
// the operator can override in chat; a false negative mails someone who asked to
// be left alone. The asymmetry decides the tuning.
// Go's RE2 reads \b as an ASCII word boundary, so a Vietnamese phrase starting
// with đ can never match one. The ASCII half keeps its boundaries; the
// Vietnamese half does without, which is why it must be spelled precisely.
var removeIntentRe = regexp.MustCompile(`(?i)(\b(unsubscribe|remove me|take me off|opt[- ]?out|stop (?:emailing|contacting) me|do not (?:contact|email) me)\b|đừng (?:gửi|liên hệ|email)|ngừng gửi|bỏ tôi ra khỏi|hủy đăng ký|huỷ đăng ký)`)

type uiReplyResult struct {
	OK        bool   `json:"ok"`
	Blocker   string `json:"blocker,omitempty"`
	Detail    string `json:"detail,omitempty"`
	DraftID   string `json:"draft_id,omitempty"`
	Sendbox   string `json:"sendbox,omitempty"`
	SentTo    string `json:"sent_to,omitempty"`
	CopySent  bool   `json:"copy_sent,omitempty"`
	CopyError string `json:"copy_error,omitempty"`
}

// uiThreadState is what the composer needs to know before it may open at all.
type uiThreadState struct {
	CanReply     bool
	Blocker      string
	Detail       string
	InboundMID   string
	InboundFrom  string
	InboundText  string
	Subject      string
	Campaign     string
	NextStep     int
	Sendbox      string
	RemoveIntent bool
}

// uiReplyState decides whether this contact may be answered, and with what.
// The page and the endpoint both call it, so what the operator is shown and what
// the server enforces cannot drift apart.
func (b *bridge) uiReplyState(c uiClient, leadID string) uiThreadState {
	var st uiThreadState
	clientDir := filepath.Join(c.Path, "outreach")
	store := newCrmStore(clientDir)
	ct := store.getContact(leadID)
	if ct == nil {
		st.Blocker = "unknown_contact"
		return st
	}

	msgs := b.uiContactThread(c, leadID)
	var lastIn, lastOut *uiMsg
	for i := range msgs {
		if msgs[i].Dir == "in" {
			lastIn = &msgs[i]
		} else {
			lastOut = &msgs[i]
		}
	}
	if lastIn == nil {
		st.Blocker = "nothing_to_answer"
		st.Detail = "This lane answers an inbound reply. Nobody has written to us on this thread yet."
		return st
	}
	if lastOut != nil && lastOut.Seq > lastIn.Seq {
		st.Blocker = "already_answered"
		st.Detail = "The newest message on this thread is ours. Wait for their reply, or use chat."
		return st
	}
	st.InboundMID, st.InboundFrom, st.InboundText = lastIn.MessageID, lastIn.From, lastIn.Body
	// Subject: theirs if we recorded it, else the one we sent. Replies before the
	// classifier started keeping the header have neither on the inbound row, and a
	// composer that opens with an empty box invites "Re: " with nothing after it.
	st.Subject = lastIn.Subject
	if st.Subject == "" && lastOut != nil {
		st.Subject = lastOut.Subject
	}
	if st.Subject == "" {
		st.Subject = b.uiLastSentSubject(clientDir, leadID)
	}

	if strings.TrimSpace(st.InboundText) == "" {
		st.Blocker = "reply_text_not_stored"
		st.Detail = "Their message was not stored, so the page cannot show it. Read it in the sendbox before answering; answering blind is how the wrong thing gets said."
		return st
	}
	if removeIntentRe.MatchString(st.InboundText) {
		st.RemoveIntent = true
		st.Blocker = "remove_request"
		st.Detail = "This reads as a request to be removed. Suppress them instead of answering; if that is a misread, say so in chat."
		return st
	}

	// Campaign and step come from the ledger, never from the browser.
	st.Campaign = b.uiLastCampaignFor(clientDir, leadID)
	if st.Campaign == "" {
		st.Blocker = "no_campaign"
		st.Detail = "No sent row for this contact, so there is no thread to reply on."
		return st
	}
	st.NextStep = b.uiNextStepFor(clientDir, st.Campaign, leadID)
	st.Sendbox = mStr(ct, "assigned_sendbox")
	st.CanReply = true
	return st
}

// uiLastSentSubject digs the subject out of the draft we sent, for threads whose
// inbound row predates subject capture. Reads only the live lanes: superseded/
// and the rotated .backup outboxes hold stale duplicates of the same draft ids.
func (b *bridge) uiLastSentSubject(clientDir, leadID string) string {
	best, subj := "", ""
	root := filepath.Join(clientDir, "campaigns")
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if !e.IsDir() || portExcluded(e.Name()) {
			continue
		}
		for _, lane := range []string{"approved", "pending_approval", "operator_reply"} {
			dir := filepath.Join(root, e.Name(), "outbox", lane)
			_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() || !strings.HasSuffix(p, ".json") {
					return nil
				}
				dd, e2 := readJSONFile(p)
				if e2 != nil || mStr(dd, "lead_id") != leadID || mStr(dd, "status") != "sent" {
					return nil
				}
				if at := mStr(dd, "decided_at"); at >= best {
					best, subj = at, mStr(dd, "subject")
				}
				return nil
			})
		}
	}
	return subj
}

// uiLastCampaignFor: the campaign of this contact's newest real send.
func (b *bridge) uiLastCampaignFor(clientDir, leadID string) string {
	best, camp := "", ""
	root := filepath.Join(clientDir, "campaigns")
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if !e.IsDir() || portExcluded(e.Name()) {
			continue
		}
		for _, p := range gmailSentLogFiles(clientDir, e.Name()) {
			for _, r := range readJSONLines(p) {
				if mStr(r, "lead_id") != leadID || mStr(r, "rfc_message_id") == "" {
					continue
				}
				if at := mStr(r, "sent_at"); at > best {
					best, camp = at, e.Name()
				}
			}
		}
	}
	return camp
}

// uiNextStepFor: one past the highest step already sent or drafted. Floor 2,
// because threadRefs is only computed for step > 1 and a reply that does not
// thread is not a reply.
func (b *bridge) uiNextStepFor(clientDir, campaign, leadID string) int {
	hi := 1
	for _, p := range gmailSentLogFiles(clientDir, campaign) {
		for _, r := range readJSONLines(p) {
			if mStr(r, "lead_id") == leadID && mInt(r, "step", 0) > hi {
				hi = mInt(r, "step", 0)
			}
		}
	}
	lane := filepath.Join(clientDir, "campaigns", campaign, "outbox", "operator_reply")
	_ = filepath.WalkDir(lane, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".json") {
			return nil
		}
		if dd, e := readJSONFile(p); e == nil && mStr(dd, "lead_id") == leadID {
			if s := mInt(dd, "step", 0); s > hi {
				hi = s
			}
		}
		return nil
	})
	return hi + 1
}

func uiReplyJSON(w http.ResponseWriter, code int, res uiReplyResult) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(res)
}

// uiReplySend is the endpoint body. The browser supplies contact_id, body_text,
// an optional subject, an optional copy_to, and the Message-ID it was looking at.
// Everything else — campaign, step, recipient, sendbox — the bridge derives, so a
// crafted request cannot redirect a send.
func (b *bridge) uiReplySend(w http.ResponseWriter, c uiClient, body map[string]any, session, now string) {
	leadID, _ := body["contact_id"].(string)
	if safeID(leadID) != nil {
		uiReplyJSON(w, http.StatusBadRequest, uiReplyResult{Blocker: "bad_contact_id"})
		return
	}
	clientDir := filepath.Join(c.Path, "outreach")
	store := newCrmStore(clientDir)
	leadID = store.resolve(leadID)

	text := strings.TrimSpace(strOr(asString(body["body_text"]), ""))
	if text == "" {
		uiReplyJSON(w, http.StatusBadRequest, uiReplyResult{Blocker: "empty_body"})
		return
	}
	copyTo := strings.TrimSpace(asString(body["copy_to"]))

	st := b.uiReplyState(c, leadID)
	if !st.CanReply {
		uiReplyJSON(w, http.StatusConflict, uiReplyResult{Blocker: st.Blocker, Detail: st.Detail})
		return
	}
	// The operator must have been looking at the message they are answering.
	if seen, _ := body["reply_to_message_id"].(string); seen != "" && seen != st.InboundMID {
		uiReplyJSON(w, http.StatusConflict, uiReplyResult{Blocker: "newer_reply_arrived",
			Detail: "They wrote again while this was open. Reload and read the newer message first."})
		return
	}

	subject := strings.TrimSpace(asString(body["subject"]))
	if subject == "" {
		subject = st.Subject
	}
	if strings.TrimSpace(strings.TrimPrefix(strings.ToLower(subject), "re:")) == "" {
		uiReplyJSON(w, http.StatusBadRequest, uiReplyResult{Blocker: "empty_subject",
			Detail: "This thread has no subject on record, so type one. \"Re:\" on its own tells the reader nothing."})
		return
	}
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}
	if !headerSafe(subject) || (copyTo != "" && !headerSafe(copyTo)) {
		uiReplyJSON(w, http.StatusBadRequest, uiReplyResult{Blocker: "invalid_header",
			Detail: "A newline in a subject or address rewrites the envelope; refused."})
		return
	}
	// A copy is an internal copy. If the address belongs to someone in the CRM it
	// is outreach, and outreach goes through the campaign engine with suppression,
	// quota and an opt-out — not through a side channel.
	if copyTo != "" {
		if other := store.a.findByIdentity("email", normalizeEmail(copyTo)); other != "" {
			uiReplyJSON(w, http.StatusConflict, uiReplyResult{Blocker: "copy_to_is_a_contact",
				Detail: "That address belongs to a contact in the CRM. Mail to a contact must go through a campaign, where suppression and opt-out apply."})
			return
		}
	}

	release, err := store.a.tryLock("ui_reply_" + leadID)
	if err == errLocked {
		uiReplyJSON(w, http.StatusConflict, uiReplyResult{Blocker: "reply_in_flight",
			Detail: "Another reply to this contact is being sent right now."})
		return
	} else if err != nil {
		uiReplyJSON(w, http.StatusInternalServerError, uiReplyResult{Blocker: "lock_failed", Detail: err.Error()})
		return
	}
	defer release()

	day := todayStr("")
	draftID := "draft_" + newULID("r")
	laneDir := filepath.Join(clientDir, "campaigns", st.Campaign, "outbox", "operator_reply", day)
	if err := os.MkdirAll(laneDir, 0o755); err != nil {
		uiReplyJSON(w, http.StatusInternalServerError, uiReplyResult{Blocker: "lane_unwritable", Detail: err.Error()})
		return
	}
	draftPath := filepath.Join(laneDir, draftID+".json")
	draft := map[string]any{
		"id": draftID, "lead_id": leadID, "campaign_slug": st.Campaign,
		"step": st.NextStep, "is_reply": true, "authored_by": "operator",
		"subject": subject, "body_text": text, "tracking": "plain_text",
		"to": st.InboundFrom, "sendbox": "", "status": "approved",
		"in_reply_to": st.InboundMID, "hooks_used": []any{},
		"created_at": now, "decided_at": now, "decided_by": "ui",
		"copy_to": copyTo, "guessed_approved": false,
	}
	f, err := os.OpenFile(draftPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		uiReplyJSON(w, http.StatusConflict, uiReplyResult{Blocker: "draft_exists", Detail: err.Error()})
		return
	}
	_, _ = f.Write([]byte(marshalIndentJSON(draft)))
	f.Close()

	// Journal BEFORE the irreversible act, so an interrupted send still left a trace.
	_ = appendUIInbox(filepath.Join(clientDir, "ui_inbox", "reply_sends.jsonl"), map[string]any{
		"ts": now, "contact_id": leadID, "campaign": st.Campaign, "draft_id": draftID,
		"step": st.NextStep, "to": st.InboundFrom, "copy_to": copyTo,
		"in_reply_to": st.InboundMID, "body_chars": len(text), "ui_session": session,
	})
	if err := store.approvalLog(draft, "approve", "ui", "operator reply from contact page"); err != nil {
		fmt.Fprintf(os.Stderr, "ui reply: approval log: %v\n", err)
	}

	res, err := gmailCmdSend(clientDir, draftPath, false)
	outcome := map[string]any{"ts": nowISO(), "draft_id": draftID}
	if err != nil {
		outcome["outcome"] = "unconfirmed"
		_ = appendUIInbox(filepath.Join(clientDir, "ui_inbox", "reply_sends.jsonl"), outcome)
		uiReplyJSON(w, http.StatusInternalServerError, uiReplyResult{Blocker: "send_error", Detail: err.Error(), DraftID: draftID})
		return
	}
	if ok, _ := res["ok"].(bool); !ok {
		outcome["outcome"] = "blocked:" + mStr(res, "blocker")
		_ = appendUIInbox(filepath.Join(clientDir, "ui_inbox", "reply_sends.jsonl"), outcome)
		uiReplyJSON(w, http.StatusConflict, uiReplyResult{Blocker: mStr(res, "blocker"),
			Detail: mStr(res, "note"), DraftID: draftID})
		return
	}
	out := uiReplyResult{OK: true, DraftID: draftID, Sendbox: mStr(res, "sendbox"), SentTo: mStr(res, "sent_to")}
	outcome["outcome"] = "sent"
	outcome["rfc_message_id"] = mStr(res, "rfc_message_id")
	_ = appendUIInbox(filepath.Join(clientDir, "ui_inbox", "reply_sends.jsonl"), outcome)

	// The copy is a SEPARATE message on a separate path: no footer, no
	// unsubscribe, no tracking, and above all no threading headers — a copy that
	// carried In-Reply-To would come back through the reply poller, match our own
	// Message-ID, and be filed as an inbound reply from the lead. The recipient of
	// the original is never told a copy exists.
	if copyTo != "" {
		if err := b.uiSendCopy(clientDir, mStr(res, "sendbox"), copyTo, subject, text, st, mStr(res, "sent_to")); err != nil {
			out.CopyError = errClassName(err)
			_ = appendUIInbox(filepath.Join(clientDir, "ui_inbox", "reply_sends.jsonl"),
				map[string]any{"ts": nowISO(), "draft_id": draftID, "outcome": "copy_failed", "copy_to": copyTo})
		} else {
			out.CopySent = true
			_ = appendUIInbox(filepath.Join(clientDir, "ui_inbox", "reply_sends.jsonl"),
				map[string]any{"ts": nowISO(), "draft_id": draftID, "outcome": "copy_sent", "copy_to": copyTo})
		}
	}
	uiReplyJSON(w, http.StatusOK, out)
}

// uiSendCopy mails one plain copy of what just went out. Never blocks or reverses
// the real send: by the time this runs the lead already has the mail.
func (b *bridge) uiSendCopy(clientDir, slug, to, subject, text string, st uiThreadState, sentTo string) error {
	sb := getSendbox(clientDir, slug)
	if sb == nil {
		return fmt.Errorf("sendbox %s not found", slug)
	}
	cred, err := loadCredentials(clientDir, slug)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Copy of a reply sent to %s at %s.\nThis copy was not sent to them and they were not told it exists.\n\n----\n\n",
		sentTo, nowISO())
	mime := buildOperatorMIME(mStr(sb, "email"), to, "[copy] "+subject, header+text)
	return operatorSMTPSend(mStr(cred, "email"), mStr(cred, "app_password"), mStr(sb, "email"), to, mime)
}

// asString accepts what encoding/json produced without panicking on a number.
func asString(v any) string {
	s, _ := v.(string)
	return s
}

var _ = sort.Strings
