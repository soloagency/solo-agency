package main

import (
	"os"
	"path/filepath"
	"testing"
)

// fixtureClient builds the smallest client tree uiReplyState reads.
func fixtureClient(t *testing.T, acts []map[string]any, contact map[string]any) (uiClient, string) {
	t.Helper()
	root := t.TempDir()
	outreach := filepath.Join(root, "outreach")
	store := newCrmStore(outreach)
	id, _, err := store.addContact(contact)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range acts {
		a["contact_id"] = id
		if _, err := store.a.appendLog("activities", a); err != nil {
			t.Fatal(err)
		}
	}
	// one sent row so a campaign can be derived
	sent := filepath.Join(outreach, "campaigns", "camp-a", "sent", "2026-08")
	if err := os.MkdirAll(sent, 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"lead_id":"` + id + `","campaign":"camp-a","step":1,"sendbox":"sb-e","rfc_message_id":"<ours@x>","sent_at":"2026-08-20T10:00:00Z"}` + "\n"
	if err := os.WriteFile(filepath.Join(sent, "sent_log.jsonl"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	return uiClient{Slug: "c", Path: root}, id
}

func inbound(seq int, snippet string) map[string]any {
	return map[string]any{"type": "email_reply", "ts": "2026-08-20T11:00:00Z", "seq": seq,
		"ref": map[string]any{"message_id": "<theirs@x>", "snippet": snippet, "from": "lead@example.com", "subject": "Hi"}}
}
func outbound(seq int) map[string]any {
	return map[string]any{"type": "email_sent", "ts": "2026-08-20T10:00:00Z", "seq": seq,
		"ref": map[string]any{"message_id": "<ours@x>"}}
}

// The lane may only ANSWER. With no inbound message there is nothing to answer,
// and allowing it would turn the contact page into a way to originate mail
// outside the campaign engine.
func TestReplyRefusedWithNoInbound(t *testing.T) {
	b := &bridge{}
	c, id := fixtureClient(t, []map[string]any{outbound(1)},
		map[string]any{"name": map[string]any{"full": "A"}, "identities": map[string]any{"emails": []any{map[string]any{"address": "lead@example.com", "is_primary": true}}}})
	st := b.uiReplyState(c, id)
	if st.CanReply || st.Blocker != "nothing_to_answer" {
		t.Fatalf("blocker = %q, canReply = %v", st.Blocker, st.CanReply)
	}
}

// Someone who asked to be removed must not be answerable from a one-click box.
// The freeze does not protect them: gmailPresendCheck exempts is_reply drafts.
func TestReplyRefusedOnRemoveRequest(t *testing.T) {
	b := &bridge{}
	c, id := fixtureClient(t, []map[string]any{outbound(1), inbound(2, "Please kindly remove me from your email list.")},
		map[string]any{"name": map[string]any{"full": "A"}, "identities": map[string]any{"emails": []any{map[string]any{"address": "lead@example.com", "is_primary": true}}}})
	st := b.uiReplyState(c, id)
	if st.CanReply || st.Blocker != "remove_request" {
		t.Fatalf("blocker = %q, canReply = %v", st.Blocker, st.CanReply)
	}
}

// An operator cannot answer what the page could not show them.
func TestReplyRefusedWhenTextNotStored(t *testing.T) {
	b := &bridge{}
	c, id := fixtureClient(t, []map[string]any{outbound(1), inbound(2, "")},
		map[string]any{"name": map[string]any{"full": "A"}, "identities": map[string]any{"emails": []any{map[string]any{"address": "lead@example.com", "is_primary": true}}}})
	st := b.uiReplyState(c, id)
	if st.CanReply || st.Blocker != "reply_text_not_stored" {
		t.Fatalf("blocker = %q, canReply = %v", st.Blocker, st.CanReply)
	}
}

// The happy path: an unanswered inbound reply opens the composer, on the sticky
// sendbox, at a step that threads.
func TestReplyAllowedAfterInbound(t *testing.T) {
	b := &bridge{}
	c, id := fixtureClient(t, []map[string]any{outbound(1), inbound(2, "Send me a sample please")},
		map[string]any{"name": map[string]any{"full": "A"}, "assigned_sendbox": "sb-e",
			"identities": map[string]any{"emails": []any{map[string]any{"address": "lead@example.com", "is_primary": true}}}})
	st := b.uiReplyState(c, id)
	if !st.CanReply {
		t.Fatalf("blocked: %s — %s", st.Blocker, st.Detail)
	}
	if st.Campaign != "camp-a" {
		t.Errorf("campaign = %q, want camp-a", st.Campaign)
	}
	if st.NextStep < 2 {
		t.Errorf("step = %d, must be >= 2 or the reply does not thread", st.NextStep)
	}
	if st.Sendbox != "sb-e" {
		t.Errorf("sendbox = %q, want the sticky sb-e", st.Sendbox)
	}
	if st.InboundMID != "<theirs@x>" {
		t.Errorf("inbound id = %q", st.InboundMID)
	}
}

// Once we answer, the composer closes until they write again — otherwise the
// lane becomes a bump channel with no gap rule.
func TestReplyClosedAfterWeAnswered(t *testing.T) {
	b := &bridge{}
	c, id := fixtureClient(t, []map[string]any{outbound(1), inbound(2, "hello"), outbound(3)},
		map[string]any{"name": map[string]any{"full": "A"}, "identities": map[string]any{"emails": []any{map[string]any{"address": "lead@example.com", "is_primary": true}}}})
	st := b.uiReplyState(c, id)
	if st.CanReply || st.Blocker != "already_answered" {
		t.Fatalf("blocker = %q, canReply = %v", st.Blocker, st.CanReply)
	}
}

func TestRemoveIntentPhrases(t *testing.T) {
	hits := []string{"Please kindly remove me from your email list.", "unsubscribe", "Take me off this list",
		"please opt out", "STOP EMAILING ME", "đừng gửi mail nữa", "ngừng gửi cho tôi", "hủy đăng ký"}
	for _, s := range hits {
		if !removeIntentRe.MatchString(s) {
			t.Errorf("missed a removal request: %q", s)
		}
	}
	for _, s := range []string{"Send me a sample video please", "remove the inspection contingency", "I will opt for the corner unit"} {
		if removeIntentRe.MatchString(s) {
			t.Errorf("false positive on: %q", s)
		}
	}
}
