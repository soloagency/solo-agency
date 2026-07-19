package main

// renderer_xval_test.go — byte-parity golden test: the Go renderer must emit
// EXACTLY the HTML the Python renderer emits (modulo the generated-at
// timestamp) for a fixture that exercises every markdown feature the two
// forks support. Scrub-gate behavior (exit 3, .blocked sidecar, found terms)
// must match on terms present in BOTH forks' lists. Skips without python3.

import (
	"os"
	"os/exec"
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

func TestRendererByteParityVsPython(t *testing.T) {
	pyRepo := findCrmStorePy(t) // ensures repo + python3
	pyRenderer := filepath.Join(filepath.Dir(pyRepo), "report_renderer.py")
	if _, err := os.Stat(pyRenderer); err != nil {
		t.Skip("report_renderer.py not present")
	}
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "report.md")
	if err := os.WriteFile(mdPath, []byte(rendererFixtureMD), 0o644); err != nil {
		t.Fatal(err)
	}
	pyHTML := filepath.Join(dir, "py.html")
	goHTML := filepath.Join(dir, "go.html")

	cmd := exec.Command("python3", pyRenderer, "render", "--input", mdPath,
		"--output-html", pyHTML, "--status-note", "Everything on *track*")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("python render failed: %v\n%s", err, out)
	}
	rc := runRenderReportCLI([]string{"render", "--input", mdPath,
		"--output-html", goHTML, "--status-note", "Everything on *track*"})
	if rc != 0 {
		t.Fatalf("go render rc=%d", rc)
	}
	pyB, _ := os.ReadFile(pyHTML)
	goB, _ := os.ReadFile(goHTML)
	pyN, goN := normalizeRendered(string(pyB)), normalizeRendered(string(goB))
	if pyN != goN {
		i := 0
		for i < len(pyN) && i < len(goN) && pyN[i] == goN[i] {
			i++
		}
		lo := i - 120
		if lo < 0 {
			lo = 0
		}
		t.Fatalf("HTML differs at byte %d\npy: …%s…\ngo: …%s…", i,
			pyN[lo:min(i+120, len(pyN))], goN[lo:min(i+120, len(goN))])
	}
}

func TestRendererScrubGateParity(t *testing.T) {
	pyRepo := findCrmStorePy(t)
	pyRenderer := filepath.Join(filepath.Dir(pyRepo), "report_renderer.py")
	if _, err := os.Stat(pyRenderer); err != nil {
		t.Skip("report_renderer.py not present")
	}
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "leaky.md")
	// terms chosen from the intersection of both forks' lists; "API **key**" is
	// split across markup so only the stripped-copy scan catches it
	leaky := "# T\n\nOur API **key** rotates; the sendbox warmup and quota look fine.\n"
	if err := os.WriteFile(mdPath, []byte(leaky), 0o644); err != nil {
		t.Fatal(err)
	}
	pyHTML := filepath.Join(dir, "py.html")
	goHTML := filepath.Join(dir, "go.html")

	cmd := exec.Command("python3", pyRenderer, "render", "--input", mdPath,
		"--output-html", pyHTML, "--client-facing", "--fail-on-scrub")
	out, err := cmd.CombinedOutput()
	ee, ok := err.(*exec.ExitError)
	if !ok || ee.ExitCode() != 3 {
		t.Fatalf("python expected exit 3, got %v\n%s", err, out)
	}
	rc := runRenderReportCLI([]string{"render", "--input", mdPath,
		"--output-html", goHTML, "--client-facing", "--fail-on-scrub"})
	if rc != 3 {
		t.Fatalf("go expected exit 3, got %d", rc)
	}
	// real outputs must not exist; .blocked sidecars must
	for _, p := range []string{pyHTML, goHTML} {
		if _, err := os.Stat(p); err == nil {
			t.Fatalf("%s must not exist on scrub block", p)
		}
	}
	pyB, err := os.ReadFile(filepath.Join(dir, "py.blocked.html"))
	if err != nil {
		t.Fatal(err)
	}
	goB, err := os.ReadFile(filepath.Join(dir, "go.blocked.html"))
	if err != nil {
		t.Fatal(err)
	}
	if normalizeRendered(string(pyB)) != normalizeRendered(string(goB)) {
		t.Fatal("blocked HTML differs")
	}
	// found terms match on the shared vocabulary
	for _, term := range []string{"API key", "sendbox", "warmup", "quota"} {
		if !strings.Contains(string(out), `"`+term+`"`) {
			t.Errorf("python did not report %q:\n%s", term, out)
		}
	}
	goStatus, err := os.ReadFile(filepath.Join(dir, "go.blocked.html.render_status.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, term := range []string{"API key", "sendbox", "warmup", "quota"} {
		if !strings.Contains(string(goStatus), `"`+term+`"`) {
			t.Errorf("go did not report %q:\n%s", term, goStatus)
		}
	}
}

func TestRendererPackageParity(t *testing.T) {
	pyRepo := findCrmStorePy(t)
	pyRenderer := filepath.Join(filepath.Dir(pyRepo), "report_renderer.py")
	if _, err := os.Stat(pyRenderer); err != nil {
		t.Skip("report_renderer.py not present")
	}
	dir := t.TempDir()
	// two rendered inputs that cross-link each other by file name
	mdA := "# A\n\nSee [the other](leadup-private-data-sources-report.html) report.\n\n## Section One\n\nBody A.\n"
	mdB := "# B\n\n## Section Two\n\nBody B references leadup-daily-report.html in text.\n\n| File |\n|---|\n| leadup-daily-report.html |\n"
	aMD, bMD := filepath.Join(dir, "a.md"), filepath.Join(dir, "b.md")
	os.WriteFile(aMD, []byte(mdA), 0o644)
	os.WriteFile(bMD, []byte(mdB), 0o644)
	aHTML := filepath.Join(dir, "leadup-daily-report.html")
	bHTML := filepath.Join(dir, "leadup-private-data-sources-report.html")
	for _, pair := range [][2]string{{aMD, aHTML}, {bMD, bHTML}} {
		if rc := runRenderReportCLI([]string{"render", "--input", pair[0], "--output-html", pair[1]}); rc != 0 {
			t.Fatalf("go render %s rc=%d", pair[0], rc)
		}
	}
	pyOut := filepath.Join(dir, "py-package.html")
	goOut := filepath.Join(dir, "go-package.html")
	cmd := exec.Command("python3", pyRenderer, "package", "--inputs", aHTML, bHTML,
		"--output-html", pyOut, "--title", "Client Package", "--client-name", "Leadup",
		"--report-date", "2026-07-19")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("python package failed: %v\n%s", err, out)
	}
	rc := runRenderReportCLI([]string{"package", "--inputs", aHTML, bHTML,
		"--output-html", goOut, "--title", "Client Package", "--client-name", "Leadup",
		"--report-date", "2026-07-19"})
	if rc != 0 {
		t.Fatalf("go package rc=%d", rc)
	}
	pyB, _ := os.ReadFile(pyOut)
	goB, _ := os.ReadFile(goOut)
	if normalizeRendered(string(pyB)) != normalizeRendered(string(goB)) {
		pyN, goN := normalizeRendered(string(pyB)), normalizeRendered(string(goB))
		i := 0
		for i < len(pyN) && i < len(goN) && pyN[i] == goN[i] {
			i++
		}
		lo := i - 120
		if lo < 0 {
			lo = 0
		}
		t.Fatalf("package HTML differs at byte %d\npy: …%s…\ngo: …%s…", i,
			pyN[lo:min(i+120, len(pyN))], goN[lo:min(i+120, len(goN))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
