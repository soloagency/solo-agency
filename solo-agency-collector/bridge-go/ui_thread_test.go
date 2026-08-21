package main

import (
	"strings"
	"testing"
)

// There is no CSP header anywhere in this binary, so anything that reaches the
// page as markup runs. Escaping must happen before linkifying, and only http(s)
// may ever become an href.
func TestLinkifyEscapesBeforeLinking(t *testing.T) {
	got := uiLinkify(`<script>alert(1)</script> see https://redfin.com/x?a=1 and javascript:evil() and <b>`)
	for _, bad := range []string{"<script>", "<b>", `href="javascript:`} {
		if strings.Contains(got, bad) {
			t.Fatalf("output contains %q: %s", bad, got)
		}
	}
	if !strings.Contains(got, `<a href="https://redfin.com/x?a=1"`) {
		t.Fatalf("http(s) url was not linked: %s", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Fatalf("script tag was not escaped: %s", got)
	}
}

// A trailing sentence period must not be swallowed into the href.
func TestLinkifyDoesNotEatPunctuation(t *testing.T) {
	got := uiLinkify("look at https://example.com/a. thanks")
	if !strings.Contains(got, `href="https://example.com/a"`) {
		t.Fatalf("href wrong: %s", got)
	}
	if !strings.Contains(got, ". thanks") {
		t.Fatalf("punctuation lost: %s", got)
	}
}
