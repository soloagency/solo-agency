package main

// renderer_xval_test.go — renderer regression tests. The HTML shape was
// byte-verified against the retired Python renderer; the committed golden file
// (testdata/renderer_fixture_golden.html) freezes that verified output.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var generatedRe = regexp.MustCompile(`(<span class="meta-label">Generated</span><span class="meta-value">)[^<]+`)

func normalizeRendered(html string) string {
	return generatedRe.ReplaceAllString(html, "${1}TS")
}

const rendererFixtureMD = `---
title: "Quarterly Momentum"
subtitle: 'Signals & next steps'
client: Leadup
kind: Weekly Client Report
date: 2026-07-19
---
# This H1 is removed

Lead paragraph with **bold**, *emphasis*, ` + "`code <&> \"quoted\"`" + `, a [link](https://ex.com/a?b=1&c=2) and an ![img](https://ex.com/i.png).

Second paragraph line one
continues on line two.

## Momentum & Wins

- First bullet with *em*
- Second bullet with [ref](https://ex.com/r)

1. Ordered one
2) Ordered two

> A quote spanning
> two lines.

| Col A | Col B & C |
|---|---:|
| **bold cell** | [t](https://ex.com/t) |
| ragged |

## Momentum & Wins

Same heading again (id dedupe).

### Sub head

#### Deep head

##### Deeper head clamps to h4

## Version 1: VE — Value Explainer

Draft body to copy.

` + "```" + `python
print("hi <>&")
` + "```" + `
`

func TestRendererGoldenSnapshot(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "report.md")
	if err := os.WriteFile(mdPath, []byte(rendererFixtureMD), 0o644); err != nil {
		t.Fatal(err)
	}
	outHTML := filepath.Join(dir, "out.html")
	rc, _, err := renderCommand(renderOpts{Input: mdPath, OutputHTML: outHTML,
		StatusNote: "Everything on *track*"})
	if err != nil || rc != 0 {
		t.Fatalf("render rc=%d err=%v", rc, err)
	}
	got, _ := os.ReadFile(outHTML)
	goldenPath := filepath.Join("testdata", "renderer_fixture_golden.html")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		os.MkdirAll("testdata", 0o755)
		if err := os.WriteFile(goldenPath, []byte(normalizeRendered(string(got))), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Log("golden updated")
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("missing golden (run with UPDATE_GOLDEN=1 to regenerate): %v", err)
	}
	g := normalizeRendered(string(got))
	if g != string(want) {
		i := 0
		w := string(want)
		for i < len(g) && i < len(w) && g[i] == w[i] {
			i++
		}
		lo := i - 120
		if lo < 0 {
			lo = 0
		}
		t.Fatalf("HTML drifted from golden at byte %d\ngot:  …%s…\nwant: …%s…", i,
			g[lo:min(i+120, len(g))], w[lo:min(i+120, len(w))])
	}
}

func TestRendererScrubGate(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "leaky.md")
	// "API **key**" is split across markup so only the tag-stripped scan catches it
	leaky := "# T\n\nOur API **key** rotates; the sendbox warmup and quota look fine.\n"
	if err := os.WriteFile(mdPath, []byte(leaky), 0o644); err != nil {
		t.Fatal(err)
	}
	outHTML := filepath.Join(dir, "client.html")
	rc, status, err := renderCommand(renderOpts{Input: mdPath, OutputHTML: outHTML,
		ClientFacing: true, FailOnScrub: true})
	if err != nil || rc != 3 {
		t.Fatalf("expected scrub block rc=3, got %d err=%v", rc, err)
	}
	if _, err := os.Stat(outHTML); err == nil {
		t.Fatal("real output must not exist on scrub block")
	}
	if _, err := os.Stat(filepath.Join(dir, "client.blocked.html")); err != nil {
		t.Fatalf(".blocked sidecar missing: %v", err)
	}
	found := map[string]bool{}
	for _, v := range mList(status, "client_blind_terms_found") {
		found[v.(string)] = true
	}
	for _, term := range []string{"API key", "sendbox", "warmup", "quota"} {
		if !found[term] {
			t.Errorf("blind term %q not reported: %v", term, status["client_blind_terms_found"])
		}
	}
}

func TestRendererPackage(t *testing.T) {
	dir := t.TempDir()
	mdA := "# A\n\nSee [the other](leadup-private-data-sources-report.html) report.\n\n## Section One\n\nBody A.\n"
	mdB := "# B\n\n## Section Two\n\nBody B.\n\n| File |\n|---|\n| leadup-daily-report.html |\n"
	aMD, bMD := filepath.Join(dir, "a.md"), filepath.Join(dir, "b.md")
	os.WriteFile(aMD, []byte(mdA), 0o644)
	os.WriteFile(bMD, []byte(mdB), 0o644)
	aHTML := filepath.Join(dir, "leadup-daily-report.html")
	bHTML := filepath.Join(dir, "leadup-private-data-sources-report.html")
	for _, pair := range [][2]string{{aMD, aHTML}, {bMD, bHTML}} {
		if rc, _, err := renderCommand(renderOpts{Input: pair[0], OutputHTML: pair[1]}); rc != 0 || err != nil {
			t.Fatalf("render %s rc=%d err=%v", pair[0], rc, err)
		}
	}
	out := filepath.Join(dir, "package.html")
	rc, _, err := packageCommand(packageOpts{Inputs: []string{aHTML, bHTML}, OutputHTML: out,
		Title: "Client Package", ClientName: "Leadup", ReportDate: "2026-07-19"})
	if rc != 0 || err != nil {
		t.Fatalf("package rc=%d err=%v", rc, err)
	}
	body, _ := os.ReadFile(out)
	s := string(body)
	// sibling file link rewritten to an in-package anchor; ids namespaced per part
	if !strings.Contains(s, `href="#part-2"`) {
		t.Error("sibling href not rewritten to #part-2")
	}
	if !strings.Contains(s, `<a href="#part-1">Daily Cover</a>`) {
		t.Error("table-cell filename not linkified to Daily Cover")
	}
	if !strings.Contains(s, `id="part-1-section-one"`) {
		t.Error("part ids not namespaced")
	}
	if !strings.Contains(s, "Private Data Sources") || !strings.Contains(s, "Daily Cover") {
		t.Error("section labels missing")
	}
}
