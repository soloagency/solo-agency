package main

import (
	"io"
	"mime/quotedprintable"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The monthly-log refactor must not move a single existing file. activities has
// years of history on the live install; a changed path silently orphans it.
func TestMonthlyLogPathsUnchanged(t *testing.T) {
	s := newJSONStore("/tmp/x")
	got, err := s.logPath("activities", "2026-08-20T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if want := "/tmp/x/crm/activities/2026-08/activities.jsonl"; got != want {
		t.Fatalf("activities path CHANGED: %q, want %q", got, want)
	}
	got, err = s.logPath("messages", "2026-08-20T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if want := "/tmp/x/crm/messages/2026-08/messages.jsonl"; got != want {
		t.Fatalf("messages path = %q, want %q", got, want)
	}
}

// messages holds bodies, not records: it must be reachable only as an
// append-only log, never as a mutable collection through crm-store.
func TestMessagesIsAReservedLog(t *testing.T) {
	s := newJSONStore("/tmp/x")
	if _, err := s.collectionDir("messages"); err == nil {
		t.Fatal("collectionDir(\"messages\") must be refused")
	}
}

// An HTTP handler cannot block on a lock: the browser waits and Go will not kill
// the goroutine. tryLock must answer immediately when the lock is held.
func TestTryLockDoesNotBlock(t *testing.T) {
	dir := t.TempDir()
	s := newJSONStore(dir)
	if err := os.MkdirAll(filepath.Join(dir, "crm"), 0o755); err != nil {
		t.Fatal(err)
	}
	release, err := s.tryLock("ui_reply_c_1")
	if err != nil {
		t.Fatalf("first tryLock: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := s.tryLock("ui_reply_c_1")
		done <- err
	}()
	if err := <-done; err != errLocked {
		t.Fatalf("second tryLock = %v, want errLocked", err)
	}
	release()
	release2, err := s.tryLock("ui_reply_c_1")
	if err != nil {
		t.Fatalf("tryLock after release: %v", err)
	}
	release2()
}

// The IMAP cursor is read before the lock is taken and every in-loop error
// returns before the cursor is saved, so the same UID is re-read routinely.
// A replayed message must not append a second body row.
func TestMessageRetainedIsIdempotencyGuard(t *testing.T) {
	dir := t.TempDir()
	store := newCrmStore(dir)
	when := "2026-08-20T00:00:00Z"
	mid := "<abc@mail.gmail.com>"
	if messageRetained(store, mid, when) {
		t.Fatal("empty ledger must not report the message as retained")
	}
	if _, err := store.a.appendLog("messages", map[string]any{
		"direction": "in", "rfc_message_id": mid, "body_text": "hello",
	}); err != nil {
		t.Fatal(err)
	}
	if !messageRetained(store, mid, when) {
		t.Fatal("a written message must be reported as retained")
	}
	if messageRetained(store, "<other@mail.gmail.com>", when) {
		t.Fatal("a different Message-ID must not match")
	}
}

// The wire body is quoted-printable; the stored body must not be. A recipient's
// mail client undoes the encoding and never sees it, so a message that arrived
// perfectly readable was being read back as "=C4=90=E1=BA=A7u".
func TestStoredBodyIsReadableNotWireEncoded(t *testing.T) {
	draft := map[string]any{
		"to": "a@b.com", "subject": "Chào em", "body_text": "Đầu tiên chúc mừng em có listing mới",
		"tracking": "plain_text", "campaign_slug": "c", "step": 2,
	}
	sb := map[string]any{"slug": "sb-e", "email": "me@x.com"}
	msg, err := gmailBuildMIME(sb, draft, "<mid@x>", "", "", trackCfg{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(msg.Body, "=C4=90") {
		t.Fatalf("the wire body should be quoted-printable, got: %.80q", msg.Body)
	}
	if strings.Contains(msg.plain, "=C4=90") {
		t.Fatalf("the stored body must NOT be encoded, got: %.80q", msg.plain)
	}
	if !strings.Contains(msg.plain, "Đầu tiên chúc mừng em có listing mới") {
		t.Fatalf("stored body lost the original text: %.120q", msg.plain)
	}
	dec, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(msg.Body)))
	if err != nil {
		t.Fatal(err)
	}
	if string(dec) != msg.plain {
		t.Fatalf("stored body is not what was sent:\n stored=%q\n onwire=%q", msg.plain, string(dec))
	}
}
