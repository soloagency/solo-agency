package main

import (
	"os"
	"path/filepath"
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
