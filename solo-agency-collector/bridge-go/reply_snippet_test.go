package main

import "testing"

// The reply that exposed the gap: the operator got a notification, opened the
// dashboard, and found nothing but "replied" — the words "I am in Canada", which
// decide whether the lead is workable at all, were parsed and thrown away.
func TestReplySnippetKeepsTheirWordsDropsOurs(t *testing.T) {
	body := "I am in Canada\n\nThank you,\n\nAlex Moshkovich | Broker Of Record\n\n" +
		"On Wed, Aug 19, 2026 at 9:14 PM Binh <binh@example.com> wrote:\n" +
		"> Alex, the quick checks before making an offer\n" +
		"> I put together 15 videos for the next 30 days\n"
	got := replySnippet(body, 2000)
	want := "I am in Canada\n\nThank you,\n\nAlex Moshkovich | Broker Of Record"
	if got != want {
		t.Fatalf("snippet = %q, want %q", got, want)
	}
}

func TestReplySnippetQuoteHeaders(t *testing.T) {
	for _, head := range []string{
		"On Mon, 1 Sep 2026 at 10:00, X wrote:",
		"Vào Th 2, 1 thg 9, 2026 lúc 10:00 X đã viết:",
		"-----Original Message-----",
		"---------- Forwarded message ---------",
		"________________________________",
	} {
		got := replySnippet("yes please\n\n"+head+"\nold thread here", 2000)
		if got != "yes please" {
			t.Errorf("head %q: snippet = %q, want %q", head, got, "yes please")
		}
	}
}

// A reply that is nothing but quoted text must not masquerade as content: an
// empty snippet is what tells the caller to fall back to the plain summary.
func TestReplySnippetAllQuotedIsEmpty(t *testing.T) {
	if got := replySnippet("> only quoted\n> more quoted\n", 2000); got != "" {
		t.Fatalf("snippet = %q, want empty", got)
	}
}

// Truncation counts runes, not bytes: cutting Vietnamese on a byte boundary
// would emit a broken rune straight into the dashboard.
func TestReplySnippetTruncatesOnRunes(t *testing.T) {
	got := replySnippet("Tôi đang ở Việt Nam và rất quan tâm", 10)
	if []rune(got)[len([]rune(got))-1] != '…' {
		t.Fatalf("snippet = %q, want an ellipsis suffix", got)
	}
	if n := len([]rune(got)); n != 11 {
		t.Fatalf("snippet rune length = %d, want 11 (10 + ellipsis)", n)
	}
	for _, r := range got {
		if r == '�' {
			t.Fatalf("snippet %q contains a replacement rune: cut on a byte boundary", got)
		}
	}
}

func TestFirstLine(t *testing.T) {
	if got := firstLine("I am in Canada\nThank you,", 120); got != "I am in Canada" {
		t.Fatalf("firstLine = %q", got)
	}
	if got := firstLine("aaaaaaaaaa", 4); got != "aaaa…" {
		t.Fatalf("firstLine truncated = %q, want %q", got, "aaaa…")
	}
}
