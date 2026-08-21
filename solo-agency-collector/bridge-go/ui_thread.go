package main

// ui_thread.go — the conversation behind one contact.
//
// The activities ledger is the spine: every email_sent and email_reply row is a
// message that happened, and that is true whether or not its text survived. The
// body is joined on from crm/messages/ when it is there. Anything that cannot be
// proven is labelled, never guessed — a card must say WHY it has no body, because
// "blank" and "nothing was sent" look identical to a reader and only one of them
// is ever true.

import (
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// uiMsg is one card in the conversation.
type uiMsg struct {
	Dir       string // "out" | "in"
	At        string
	From      string
	To        string
	Cc        string
	Subject   string // the real Subject header; empty when it was never recorded
	Summary   string // the activity's own words, shown when there is no subject
	Body      string
	Chip      string // as sent | as received | excerpt | excerpt unavailable | not retained
	Note      string // why the body is partial or absent; empty when whole
	MessageID string
	By        string
	Seq       int
}

// uiContactThread reads one contact's conversation, oldest first.
func (b *bridge) uiContactThread(c uiClient, id string) []uiMsg {
	clientDir := filepath.Join(c.Path, "outreach")
	store := newCrmStore(clientDir)

	// Bodies, newest row per Message-ID: a correction is a NEW row, never a
	// rewrite, so the highest seq is the current truth.
	bodies := map[string]map[string]any{}
	base := filepath.Join(clientDir, "crm", "messages")
	months, _ := os.ReadDir(base)
	for _, m := range months {
		for _, r := range readJSONLines(filepath.Join(base, m.Name(), "messages.jsonl")) {
			mid := mStr(r, "rfc_message_id")
			if mid == "" {
				continue
			}
			if prev, ok := bodies[mid]; !ok || mInt(r, "seq", 0) >= mInt(prev, "seq", 0) {
				bodies[mid] = r
			}
		}
	}

	memo := map[string]string{}
	resolved := func(x string) string {
		if x == "" {
			return x
		}
		if v, ok := memo[x]; ok {
			return v
		}
		v := store.resolve(x)
		memo[x] = v
		return v
	}

	var out []uiMsg
	abase := filepath.Join(clientDir, "crm", "activities")
	amonths, _ := os.ReadDir(abase)
	names := make([]string, 0, len(amonths))
	for _, m := range amonths {
		names = append(names, m.Name())
	}
	sort.Strings(names)
	for _, m := range names {
		for _, a := range readJSONLines(filepath.Join(abase, m, "activities.jsonl")) {
			t := mStr(a, "type")
			if t != "email_sent" && t != "email_reply" {
				continue
			}
			if cid := mStr(a, "contact_id"); cid != id && resolved(cid) != id {
				continue
			}
			ref := mMap(a, "ref")
			mid := mStr(ref, "message_id")
			msg := uiMsg{At: mStr(a, "ts"), MessageID: mid, By: mStr(a, "by"), Seq: mInt(a, "seq", 0)}
			if t == "email_sent" {
				msg.Dir = "out"
			} else {
				msg.Dir = "in"
			}
			row := bodies[mid]
			switch {
			case row != nil && mStr(row, "body_text") != "":
				msg.Body = mStr(row, "body_text")
				msg.From, msg.To, msg.Cc = mStr(row, "from"), mStr(row, "to"), mStr(row, "cc")
				msg.Subject = mStr(row, "subject")
				if msg.Dir == "out" {
					msg.Chip = "as sent"
				} else {
					msg.Chip = "as received"
				}
			case msg.Dir == "in" && mStr(ref, "snippet") != "":
				msg.Body = mStr(ref, "snippet")
				msg.From, msg.Subject = mStr(ref, "from"), mStr(ref, "subject")
				msg.Chip = "excerpt"
				msg.Note = "stored excerpt: quoted thread removed, capped. The whole message is in the sendbox."
			case msg.Dir == "in" && hasKey(ref, "snippet"):
				msg.From, msg.Subject = mStr(ref, "from"), mStr(ref, "subject")
				msg.Chip = "excerpt unavailable"
				msg.Note = "this reply was written below the quoted thread, so the excerpt writer stored nothing. Read it in the sendbox."
			default:
				msg.Chip = "not retained"
				msg.Note = "body not retained: this message predates message-body retention. The original is in the sendbox."
			}
			// Never dress the activity summary up as a Subject header. "sent step 1
			// via sb-e" is what the ledger said happened, not what the recipient saw
			// in their inbox, and a card that prints it on the subject line invents a
			// fact about the email.
			msg.Summary = mStr(a, "summary")
			out = append(out, msg)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Seq < out[j].Seq })
	return out
}

// hasKey distinguishes "the writer ran and stored an empty string" from "the
// writer never ran". Those are different facts and the card says different things.
func hasKey(m map[string]any, k string) bool {
	if m == nil {
		return false
	}
	_, ok := m[k]
	return ok
}

// uiLinkify escapes first, then turns bare http(s) URLs into links. Order is the
// whole point: there is no CSP header anywhere in this binary, so anything that
// reaches the page as markup runs. javascript: and data: never become hrefs
// because only an http/https prefix is matched at all.
func uiLinkify(s string) string {
	esc := html.EscapeString(s)
	var out strings.Builder
	for {
		i := strings.Index(esc, "http")
		if i < 0 || (!strings.HasPrefix(esc[i:], "http://") && !strings.HasPrefix(esc[i:], "https://")) {
			if i < 0 {
				out.WriteString(esc)
				break
			}
			out.WriteString(esc[:i+4])
			esc = esc[i+4:]
			continue
		}
		out.WriteString(esc[:i])
		esc = esc[i:]
		j := strings.IndexAny(esc, " \t\r\n<\"')")
		if j < 0 {
			j = len(esc)
		}
		u := strings.TrimRight(esc[:j], ".,;:")
		out.WriteString(`<a href="` + u + `" target="_blank" rel="noopener noreferrer">` + u + `</a>`)
		esc = esc[len(u):]
	}
	return out.String()
}
