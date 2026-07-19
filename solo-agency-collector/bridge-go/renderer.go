package main

// renderer.go — unified Go port of tools/solo_report_renderer.py and
// outreach/tools/report_renderer.py (the two ~51-line-divergent forks).
// Markdown -> standalone HTML must stay BYTE-IDENTICAL to the Python output
// (modulo the generated-at timestamp), verified by the renderer golden test.
//
// Unification decisions (per docs/UI_DESIGN.md §8 "adopt the stricter scrub
// logic"): the client-blind term list is the UNION of both forks; natural-
// English terms use word-boundary matching; the rendered-HTML scrub also scans
// a tag-stripped copy. PDF export keeps Chrome-headless + wkhtmltopdf; the
// weasyprint/reportlab Python-library fallbacks are dropped (they cannot exist
// in a Go binary and were optional).

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

// Union of both forks' lists; order follows the outreach fork with the
// solo-agency-only terms appended (order only affects the found-list order).
var clientBlindTerms = []string{
	"OutreachCRM",
	"WideCast",
	"INTERNAL_REPORT",
	"MCP",
	"OpenAPI",
	"API key",
	"Telegram",
	"PDNA",
	"provider_config",
	"Client tools",
	"global MCP",
	"automation",
	"scheduled task",
	"agent debug",
	"config file",
	"debug",
	"sendbox",
	"gmail_client",
	"crm_store",
	"storage_config",
	"trk.",
	"HMAC",
	"token.json",
	"sent_log",
	"suppression",
	"warmup",
	"quota",
	"guessed",
	// solo-agency fork additions
	"Solo Agency",
	"Local Collector",
	"Chrome extension",
	"PDNA provider",
	"collector bridge",
}

var naturalWordTerms = map[string]bool{"quota": true, "automation": true, "debug": true,
	"guessed": true, "warmup": true, "suppression": true}

// --- tiny Python-compat helpers -----------------------------------------------

// pyHTMLEscape mirrors Python html.escape (which emits &#x27; for ' — Go's
// html.EscapeString emits &#39; and would break byte parity).
func pyHTMLEscape(s string, quote bool) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	if quote {
		s = strings.ReplaceAll(s, `"`, "&quot;")
		s = strings.ReplaceAll(s, "'", "&#x27;")
	}
	return s
}

func rendererSlugify(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(v, "-")
	v = strings.Trim(v, "-")
	if v == "" {
		return "report"
	}
	return v
}

// rendererNowISO = Python dt.datetime.now(utc).astimezone().isoformat(timespec="seconds"):
// local time with numeric offset.
func rendererNowISO() string {
	return time.Now().Format("2006-01-02T15:04:05-07:00")
}

func parseFrontmatter(text string) (map[string]string, string) {
	meta := map[string]string{}
	if !strings.HasPrefix(text, "---\n") {
		return meta, text
	}
	end := strings.Index(text[4:], "\n---")
	if end == -1 {
		return meta, text
	}
	end += 4
	raw := strings.TrimSpace(text[4:end])
	body := strings.TrimLeft(text[end+4:], "\n")
	for _, line := range strings.Split(raw, "\n") {
		if !strings.Contains(line, ":") {
			continue
		}
		k, v, _ := strings.Cut(line, ":")
		v = strings.TrimSpace(v)
		v = strings.Trim(v, `"`)
		v = strings.Trim(v, `'`)
		meta[strings.TrimSpace(k)] = v
	}
	return meta, body
}

var (
	mdCodeRe     = regexp.MustCompile("`([^`]+)`")
	mdImageRe    = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	mdLinkRe     = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	mdStrongRe   = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	mdMarksRe    = regexp.MustCompile(`[*_#>]+`)
	mdHeadingRe  = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	mdBulletRe   = regexp.MustCompile(`^[-*]\s+`)
	mdNumberRe   = regexp.MustCompile(`^\d+[.)]\s+`)
	tableDelimRe = regexp.MustCompile(`^:?-{3,}:?$`)
	versionRe    = regexp.MustCompile(`(?i)^Version\s+\d+\b`)
	firstH1Re    = regexp.MustCompile(`(?m)^#\s+(.+)$`)
)

func stripMD(v string) string {
	v = mdCodeRe.ReplaceAllString(v, "$1")
	v = mdImageRe.ReplaceAllString(v, "$1")
	v = mdLinkRe.ReplaceAllString(v, "$1")
	v = mdMarksRe.ReplaceAllString(v, "")
	return strings.TrimSpace(v)
}

// emReplace mirrors Python (?<!\*)\*([^*]+)\*(?!\*) — Go regexp has no
// lookaround, so boundary chars are checked by hand.
func emReplace(v string) string {
	re := regexp.MustCompile(`\*([^*]+)\*`)
	var sb strings.Builder
	last := 0
	for _, m := range re.FindAllStringSubmatchIndex(v, -1) {
		start, end := m[0], m[1]
		if (start > 0 && v[start-1] == '*') || (end < len(v) && v[end] == '*') {
			continue
		}
		sb.WriteString(v[last:start])
		sb.WriteString("<em>" + v[m[2]:m[3]] + "</em>")
		last = end
	}
	sb.WriteString(v[last:])
	return sb.String()
}

func inlineMD(v string) string {
	var codeSpans []string
	v = mdCodeRe.ReplaceAllStringFunc(v, func(m string) string {
		inner := mdCodeRe.FindStringSubmatch(m)[1]
		codeSpans = append(codeSpans, "<code>"+pyHTMLEscape(inner, true)+"</code>")
		return fmt.Sprintf("\x00CODE%d\x00", len(codeSpans)-1)
	})
	v = pyHTMLEscape(v, false)
	v = mdImageRe.ReplaceAllStringFunc(v, func(m string) string {
		g := mdImageRe.FindStringSubmatch(m)
		return fmt.Sprintf(`<img src="%s" alt="%s" loading="lazy">`,
			pyHTMLEscape(g[2], true), pyHTMLEscape(g[1], true))
	})
	v = mdLinkRe.ReplaceAllStringFunc(v, func(m string) string {
		g := mdLinkRe.FindStringSubmatch(m)
		return fmt.Sprintf(`<a href="%s">%s</a>`, pyHTMLEscape(g[2], true), g[1])
	})
	v = mdStrongRe.ReplaceAllString(v, "<strong>$1</strong>")
	v = emReplace(v)
	for i, span := range codeSpans {
		v = strings.Replace(v, fmt.Sprintf("\x00CODE%d\x00", i), span, 1)
	}
	return v
}

func isTableDelimiter(line string) bool {
	s := strings.TrimSpace(line)
	if !strings.Contains(s, "|") {
		return false
	}
	cells := strings.Split(strings.Trim(s, "|"), "|")
	if len(cells) == 0 {
		return false
	}
	for _, c := range cells {
		if !tableDelimRe.MatchString(strings.TrimSpace(c)) {
			return false
		}
	}
	return true
}

func splitTableRow(line string) []string {
	cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

func renderTable(lines []string) string {
	var rows [][]string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			rows = append(rows, splitTableRow(l))
		}
	}
	if len(rows) < 2 {
		return ""
	}
	header := rows[0]
	body := rows[2:]
	var parts []string
	parts = append(parts, `<div class="table-scroll"><table><thead><tr>`)
	for _, cell := range header {
		parts = append(parts, "<th>"+inlineMD(cell)+"</th>")
	}
	parts = append(parts, "</tr></thead><tbody>")
	for _, row := range body {
		parts = append(parts, "<tr>")
		for i := range header {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			parts = append(parts, "<td>"+inlineMD(cell)+"</td>")
		}
		parts = append(parts, "</tr>")
	}
	parts = append(parts, "</tbody></table></div>")
	return strings.Join(parts, "")
}

func isBlockStart(line string, next *string) bool {
	s := strings.TrimSpace(line)
	if s == "" {
		return true
	}
	if strings.HasPrefix(s, "```") {
		return true
	}
	if mdHeadingRe.MatchString(s) {
		return true
	}
	if mdBulletRe.MatchString(s) {
		return true
	}
	if mdNumberRe.MatchString(s) {
		return true
	}
	if strings.HasPrefix(s, ">") {
		return true
	}
	if strings.Contains(s, "|") && next != nil && isTableDelimiter(*next) {
		return true
	}
	return false
}

func headingID(text string, used map[string]bool) string {
	base := rendererSlugify(stripMD(text))
	cand := base
	counter := 2
	for used[cand] {
		cand = fmt.Sprintf("%s-%d", base, counter)
		counter++
	}
	used[cand] = true
	return cand
}

type navEntry struct{ ID, Label string }

func renderMarkdownBody(markdown string) (string, []navEntry) {
	lines := strings.Split(markdown, "\n")
	var out []string
	var nav []navEntry
	used := map[string]bool{}
	inSection := false
	inVersionSection := false

	closeCurrent := func() {
		if inSection {
			if inVersionSection {
				out = append(out, "</div>")
				inVersionSection = false
			}
			out = append(out, "</section>")
			inSection = false
		}
	}

	i := 0
	for i < len(lines) {
		line := lines[i]
		stripped := strings.TrimSpace(line)
		if stripped == "" {
			i++
			continue
		}
		if strings.HasPrefix(stripped, "```") {
			lang := strings.TrimSpace(strings.Trim(stripped, "`"))
			i++
			var codeLines []string
			for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
				codeLines = append(codeLines, lines[i])
				i++
			}
			if i < len(lines) {
				i++
			}
			classAttr := ""
			if lang != "" {
				classAttr = fmt.Sprintf(` class="language-%s"`, pyHTMLEscape(lang, true))
			}
			out = append(out, fmt.Sprintf("<pre><code%s>%s</code></pre>", classAttr,
				pyHTMLEscape(strings.Join(codeLines, "\n"), true)))
			continue
		}
		if h := mdHeadingRe.FindStringSubmatch(stripped); h != nil {
			level := len(h[1])
			text := strings.TrimSpace(h[2])
			if level == 2 {
				closeCurrent()
				sid := headingID(text, used)
				nav = append(nav, navEntry{sid, stripMD(text)})
				if versionRe.MatchString(stripMD(text)) {
					bodyID := sid + "-draft"
					out = append(out, fmt.Sprintf(
						`<section class="report-section" id="%s"><h2>%s</h2>`+
							`<div class="draft-actions"><button type="button" class="copy-btn" `+
							`data-copy-target="%s">Copy draft</button></div>`+
							`<div class="draft-editable" id="%s" contenteditable="true" spellcheck="true">`,
						sid, inlineMD(text), bodyID, bodyID))
					inSection = true
					inVersionSection = true
				} else {
					out = append(out, fmt.Sprintf(`<section class="report-section" id="%s"><h2>%s</h2>`, sid, inlineMD(text)))
					inSection = true
				}
			} else if level == 1 {
				out = append(out, "<h2>"+inlineMD(text)+"</h2>")
			} else {
				lv := level
				if lv > 4 {
					lv = 4
				}
				out = append(out, fmt.Sprintf("<h%d>%s</h%d>", lv, inlineMD(text), lv))
			}
			i++
			continue
		}
		if strings.Contains(stripped, "|") && i+1 < len(lines) && isTableDelimiter(lines[i+1]) {
			tableLines := []string{lines[i], lines[i+1]}
			i += 2
			for i < len(lines) && strings.Contains(lines[i], "|") && strings.TrimSpace(lines[i]) != "" {
				tableLines = append(tableLines, lines[i])
				i++
			}
			if !inSection {
				out = append(out, `<section class="report-section">`)
				inSection = true
			}
			out = append(out, renderTable(tableLines))
			continue
		}
		if mdBulletRe.MatchString(stripped) {
			if !inSection {
				out = append(out, `<section class="report-section">`)
				inSection = true
			}
			out = append(out, "<ul>")
			for i < len(lines) && mdBulletRe.MatchString(strings.TrimSpace(lines[i])) {
				item := mdBulletRe.ReplaceAllString(strings.TrimSpace(lines[i]), "")
				out = append(out, "<li>"+inlineMD(item)+"</li>")
				i++
			}
			out = append(out, "</ul>")
			continue
		}
		if mdNumberRe.MatchString(stripped) {
			if !inSection {
				out = append(out, `<section class="report-section">`)
				inSection = true
			}
			out = append(out, "<ol>")
			for i < len(lines) && mdNumberRe.MatchString(strings.TrimSpace(lines[i])) {
				item := mdNumberRe.ReplaceAllString(strings.TrimSpace(lines[i]), "")
				out = append(out, "<li>"+inlineMD(item)+"</li>")
				i++
			}
			out = append(out, "</ol>")
			continue
		}
		if strings.HasPrefix(stripped, ">") {
			if !inSection {
				out = append(out, `<section class="report-section">`)
				inSection = true
			}
			var quoteLines []string
			for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), ">") {
				q := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(lines[i]), ">"))
				quoteLines = append(quoteLines, q)
				i++
			}
			out = append(out, "<blockquote>"+inlineMD(strings.Join(quoteLines, " "))+"</blockquote>")
			continue
		}
		paragraphLines := []string{stripped}
		i++
		for i < len(lines) {
			nextLine := lines[i]
			var following *string
			if i+1 < len(lines) {
				following = &lines[i+1]
			}
			if isBlockStart(nextLine, following) {
				break
			}
			paragraphLines = append(paragraphLines, strings.TrimSpace(nextLine))
			i++
		}
		if !inSection {
			out = append(out, `<section class="report-section">`)
			inSection = true
		}
		klass := ""
		if len(out) < 3 {
			klass = ` class="lead-paragraph"`
		}
		out = append(out, fmt.Sprintf("<p%s>%s</p>", klass, inlineMD(strings.Join(paragraphLines, " "))))
	}
	closeCurrent()
	return strings.Join(out, "\n"), nav
}

func inferSubtitle(markdown string) string {
	for _, line := range strings.Split(markdown, "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") || strings.HasPrefix(s, "|") {
			continue
		}
		if strings.HasPrefix(s, "-") || strings.HasPrefix(s, "*") || strings.HasPrefix(s, "```") || strings.HasPrefix(s, ">") {
			continue
		}
		v := stripMD(s)
		if len(v) > 220 {
			v = v[:220]
		}
		return v
	}
	return "A concise intelligence report with evidence, recommendations, and next actions."
}

// --- scrub gate -----------------------------------------------------------------

func scrubCheck(text string) []string {
	var found []string
	for _, term := range clientBlindTerms {
		pattern := regexp.QuoteMeta(term)
		if naturalWordTerms[strings.ToLower(term)] {
			pattern = `\b` + pattern + `\b`
		}
		if regexp.MustCompile(`(?i)` + pattern).MatchString(text) {
			found = append(found, term)
		}
	}
	return found
}

var tagRe = regexp.MustCompile(`<[^>]+>`)
var wsRe = regexp.MustCompile(`\s+`)

func scrubCheckRendered(htmlText string) []string {
	stripped := wsRe.ReplaceAllString(tagRe.ReplaceAllString(htmlText, ""), " ")
	set := map[string]bool{}
	for _, t := range scrubCheck(htmlText) {
		set[t] = true
	}
	for _, t := range scrubCheck(stripped) {
		set[t] = true
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// --- full page ------------------------------------------------------------------

func renderFullHTML(bodyHTML string, nav []navEntry, title, subtitle, clientName, reportKind,
	reportDate, generatedAt, statusNote string) string {
	navItems := nav
	if len(navItems) > 14 {
		navItems = navItems[:14]
	}
	var navLines []string
	for _, n := range navItems {
		navLines = append(navLines, fmt.Sprintf(`<a href="#%s">%s</a>`, n.ID, pyHTMLEscape(n.Label, true)))
	}
	navHTML := strings.Join(navLines, "\n")
	if navHTML == "" {
		navHTML = `<a href="#report">Report</a>`
	}
	status := ""
	if statusNote != "" {
		status = `<div class="status-note">` + inlineMD(statusNote) + `</div>`
	}
	if clientName == "" {
		clientName = "Client"
	}
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<style>%s</style>
</head>
<body>
<div class="report-page">
  <header class="report-hero">
    <div class="hero-copy">
      <p class="eyebrow">%s</p>
      <h1>%s</h1>
      <p class="dek">%s</p>
      %s
    </div>
    <aside class="hero-panel" aria-label="Report metadata">
      <div class="hero-panel-inner">
        <div class="meta-grid">
          <div class="meta-item"><span class="meta-label">Client</span><span class="meta-value">%s</span></div>
          <div class="meta-item"><span class="meta-label">Report date</span><span class="meta-value">%s</span></div>
          <div class="meta-item"><span class="meta-label">Format</span><span class="meta-value">HTML report plus PDF companion</span></div>
          <div class="meta-item"><span class="meta-label">Generated</span><span class="meta-value">%s</span></div>
        </div>
      </div>
    </aside>
  </header>
  <div class="report-shell" id="report">
    <nav class="toc" aria-label="Report sections">
      <p class="toc-title">Report Sections</p>
      %s
    </nav>
    <main class="report-body">
      %s
    </main>
  </div>
</div>
<script>%s</script>
</body>
</html>
`, pyHTMLEscape(title, true), rendererCSS, pyHTMLEscape(reportKind, true), pyHTMLEscape(title, true),
		pyHTMLEscape(subtitle, true), status, pyHTMLEscape(clientName, true), pyHTMLEscape(reportDate, true),
		pyHTMLEscape(generatedAt, true), navHTML, bodyHTML, rendererCopyScript)
}

func findFirstH1(markdown string) string {
	if m := firstH1Re.FindStringSubmatch(markdown); m != nil {
		return stripMD(m[1])
	}
	return ""
}

func removeFirstH1(markdown string) string {
	loc := regexp.MustCompile(`(?m)^#\s+.+\n?`).FindStringIndex(markdown)
	if loc == nil {
		return markdown
	}
	return markdown[:loc[0]] + markdown[loc[1]:]
}

// --- render + package commands ----------------------------------------------------

func writeRenderText(path, text string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(text), 0o644)
}

func writeRenderStatus(outputHTML string, status map[string]any) {
	statusPath := outputHTML + ".render_status.json"
	_ = writeRenderText(statusPath, marshalIndentJSON(status)+"\n")
}

func blockedName(outputHTML string) string {
	ext := filepath.Ext(outputHTML)
	stem := strings.TrimSuffix(filepath.Base(outputHTML), ext)
	return filepath.Join(filepath.Dir(outputHTML), stem+".blocked"+ext)
}

type renderOpts struct {
	Input        string
	OutputHTML   string
	OutputPDF    string
	Title        string
	Subtitle     string
	ClientName   string
	ReportKind   string
	ReportDate   string
	StatusNote   string
	ClientFacing bool
	FailOnScrub  bool
}

func renderCommand(o renderOpts) (int, map[string]any, error) {
	raw, err := os.ReadFile(o.Input)
	if err != nil {
		return 1, nil, err
	}
	frontmatter, markdown := parseFrontmatter(string(raw))
	title := o.Title
	if title == "" {
		title = frontmatter["title"]
	}
	if title == "" {
		title = findFirstH1(markdown)
	}
	if title == "" {
		title = "Intelligence Report"
	}
	markdown = removeFirstH1(markdown)
	subtitle := o.Subtitle
	if subtitle == "" {
		subtitle = frontmatter["subtitle"]
	}
	if subtitle == "" {
		subtitle = inferSubtitle(markdown)
	}
	clientName := o.ClientName
	if clientName == "" {
		clientName = frontmatter["client"]
	}
	reportKind := o.ReportKind
	if reportKind == "" {
		reportKind = frontmatter["kind"]
	}
	if reportKind == "" {
		reportKind = "Agency Intelligence Report"
	}
	reportDate := o.ReportDate
	if reportDate == "" {
		reportDate = frontmatter["date"]
	}
	if reportDate == "" {
		reportDate = time.Now().Format("2006-01-02")
	}
	generatedAt := rendererNowISO()

	bodyHTML, nav := renderMarkdownBody(markdown)
	htmlText := renderFullHTML(bodyHTML, nav, title, subtitle, clientName, reportKind,
		reportDate, generatedAt, o.StatusNote)

	var scrubFound []string
	if o.ClientFacing {
		scrubFound = scrubCheckRendered(htmlText)
	}
	if o.ClientFacing && len(scrubFound) > 0 && o.FailOnScrub {
		blocked := blockedName(o.OutputHTML)
		if err := writeRenderText(blocked, htmlText); err != nil {
			return 1, nil, err
		}
		status := map[string]any{
			"command": "render", "input": o.Input, "output_html": o.OutputHTML,
			"blocked_output_html": blocked, "generated_at": generatedAt,
			"client_blind_terms_found": strsToAny(scrubFound), "scrub_status": "blocked",
			"pdf_status": "not_generated_scrub_blocked", "pdf_path": "", "pdf_blocker": "",
		}
		writeRenderStatus(blocked, status)
		return 3, status, nil
	}
	if err := writeRenderText(o.OutputHTML, htmlText); err != nil {
		return 1, nil, err
	}
	scrubStatus := "not_checked"
	if o.ClientFacing {
		scrubStatus = "clean"
	}
	status := map[string]any{
		"command": "render", "input": o.Input, "output_html": o.OutputHTML,
		"generated_at": generatedAt, "client_blind_terms_found": strsToAny(scrubFound),
		"scrub_status": scrubStatus, "pdf_status": "not_requested", "pdf_path": "", "pdf_blocker": "",
	}
	if o.OutputPDF != "" {
		pdfStatus, blocker := exportPDF(o.OutputHTML, o.OutputPDF)
		status["pdf_status"] = pdfStatus
		status["pdf_path"] = o.OutputPDF
		status["pdf_blocker"] = blocker
	}
	writeRenderStatus(o.OutputHTML, status)
	if status["pdf_status"] == "blocked" {
		return 2, status, nil
	}
	return 0, status, nil
}

func strsToAny(in []string) []any {
	out := []any{}
	for _, s := range in {
		out = append(out, s)
	}
	return out
}

// --- package command --------------------------------------------------------------

var scriptTagRe = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`)
var mainTagRe = regexp.MustCompile(`(?is)<main\b[^>]*>(.*?)</main>`)
var bodyTagRe = regexp.MustCompile(`(?is)<body\b[^>]*>(.*?)</body>`)
var hrefAttrRe = regexp.MustCompile(`(?i)\bhref=(["'])([^"']+)["']`)
var idAttrRe = regexp.MustCompile(`(?i)\bid=(["'])([^"']+)["']`)
var hrefHashRe = regexp.MustCompile(`(?i)\bhref=(["'])#([^"']+)["']`)

func extractBodyFragment(htmlText string) string {
	htmlText = scriptTagRe.ReplaceAllString(htmlText, "")
	if m := mainTagRe.FindStringSubmatch(htmlText); m != nil {
		return strings.TrimSpace(m[1])
	}
	if m := bodyTagRe.FindStringSubmatch(htmlText); m != nil {
		return strings.TrimSpace(m[1])
	}
	return htmlText
}

func packageSectionLabel(path string) string {
	lower := strings.ToLower(filepath.Base(path))
	if strings.Contains(lower, "daily-report") {
		return "Daily Cover"
	}
	if strings.Contains(lower, "public-data-sources-report") {
		return "Public Data Sources"
	}
	if strings.Contains(lower, "private-data-sources-report") {
		return "Private Data Sources"
	}
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return pyTitle(strings.ReplaceAll(stem, "-", " "))
}

func localHrefBasename(value string) string {
	unescaped := strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#x27;", "'").Replace(value)
	u, err := url.Parse(unescaped)
	if err != nil || u.Scheme != "" || u.Host != "" {
		return ""
	}
	p, err := url.PathUnescape(u.Path)
	if err != nil {
		p = u.Path
	}
	return filepath.Base(p)
}

func rewriteSiblingReportLinks(fragment string, linkTargets map[string][2]string) string {
	fragment = hrefAttrRe.ReplaceAllStringFunc(fragment, func(m string) string {
		g := hrefAttrRe.FindStringSubmatch(m)
		quote, value := g[1], g[2]
		basename := localHrefBasename(value)
		if t, ok := linkTargets[basename]; ok {
			return "href=" + quote + pyHTMLEscape(t[0], true) + quote
		}
		return m
	})
	names := make([]string, 0, len(linkTargets))
	for n := range linkTargets {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, basename := range names {
		t := linkTargets[basename]
		escapedName := pyHTMLEscape(basename, true)
		replacement := fmt.Sprintf(`<a href="%s">%s</a>`, pyHTMLEscape(t[0], true), pyHTMLEscape(t[1], true))
		cellRe := regexp.MustCompile(`(?i)(<t[dh][^>]*>\s*)` + regexp.QuoteMeta(escapedName) + `(\s*</t[dh]>)`)
		fragment = cellRe.ReplaceAllString(fragment, "${1}"+strings.ReplaceAll(replacement, "$", "$$")+"${2}")
	}
	return fragment
}

func namespaceFragmentIDs(fragment, prefix string) string {
	idMap := map[string]string{}
	for _, m := range idAttrRe.FindAllStringSubmatch(fragment, -1) {
		old := m[2]
		if !strings.HasPrefix(old, prefix+"-") {
			idMap[old] = prefix + "-" + old
		}
	}
	if len(idMap) == 0 {
		return fragment
	}
	fragment = idAttrRe.ReplaceAllStringFunc(fragment, func(m string) string {
		g := idAttrRe.FindStringSubmatch(m)
		quote, old := g[1], g[2]
		v, ok := idMap[old]
		if !ok {
			v = old
		}
		return "id=" + quote + pyHTMLEscape(v, true) + quote
	})
	fragment = hrefHashRe.ReplaceAllStringFunc(fragment, func(m string) string {
		g := hrefHashRe.FindStringSubmatch(m)
		quote, old := g[1], g[2]
		v, ok := idMap[old]
		if !ok {
			v = old
		}
		return "href=" + quote + "#" + pyHTMLEscape(v, true) + quote
	})
	return fragment
}

type packageOpts struct {
	Inputs       []string
	OutputHTML   string
	OutputPDF    string
	Title        string
	Subtitle     string
	ClientName   string
	ReportKind   string
	ReportDate   string
	StatusNote   string
	ClientFacing bool
	FailOnScrub  bool
}

func packageCommand(o packageOpts) (int, map[string]any, error) {
	generatedAt := rendererNowISO()
	var missing []any
	linkTargets := map[string][2]string{}
	for idx, p := range o.Inputs {
		linkTargets[filepath.Base(p)] = [2]string{fmt.Sprintf("#part-%d", idx+1), packageSectionLabel(p)}
	}
	var sections []string
	for idx, p := range o.Inputs {
		sectionID := fmt.Sprintf("part-%d", idx+1)
		label := packageSectionLabel(p)
		data, err := os.ReadFile(p)
		if err != nil {
			missing = append(missing, p)
			sections = append(sections, fmt.Sprintf(
				`<section class="report-section" id="%s"><h2>%s</h2>`+
					"<p>This section was not available when this report was prepared.</p></section>",
				sectionID, pyHTMLEscape(label, true)))
			continue
		}
		fragment := extractBodyFragment(string(data))
		fragment = rewriteSiblingReportLinks(fragment, linkTargets)
		fragment = namespaceFragmentIDs(fragment, sectionID)
		sections = append(sections, fmt.Sprintf(
			`<section class="report-section print-source" id="%s"><div class="print-source-title">`+
				"<strong>%s</strong></div>%s</section>",
			sectionID, pyHTMLEscape(label, true), fragment))
	}
	bodyHTML := strings.Join(sections, "\n")
	var nav []navEntry
	for idx, p := range o.Inputs {
		nav = append(nav, navEntry{fmt.Sprintf("part-%d", idx+1), packageSectionLabel(p)})
	}
	title := o.Title
	if title == "" {
		title = "Client Report"
	}
	subtitle := o.Subtitle
	if subtitle == "" {
		subtitle = "A complete client-facing report assembled into one standalone HTML file with a matching PDF companion."
	}
	reportKind := o.ReportKind
	if reportKind == "" {
		reportKind = "Client Report Package"
	}
	reportDate := o.ReportDate
	if reportDate == "" {
		reportDate = time.Now().Format("2006-01-02")
	}
	htmlText := renderFullHTML(bodyHTML, nav, title, subtitle, o.ClientName, reportKind,
		reportDate, generatedAt, o.StatusNote)

	inputsAny := strsToAny(o.Inputs)
	if missing == nil {
		missing = []any{}
	}
	var scrubFound []string
	if o.ClientFacing {
		scrubFound = scrubCheckRendered(htmlText)
	}
	if o.ClientFacing && len(scrubFound) > 0 && o.FailOnScrub {
		blocked := blockedName(o.OutputHTML)
		if err := writeRenderText(blocked, htmlText); err != nil {
			return 1, nil, err
		}
		status := map[string]any{
			"command": "package", "inputs": inputsAny, "missing_inputs": missing,
			"output_html": o.OutputHTML, "blocked_output_html": blocked,
			"generated_at": generatedAt, "client_blind_terms_found": strsToAny(scrubFound),
			"scrub_status": "blocked", "pdf_status": "not_generated_scrub_blocked",
			"pdf_path": "", "pdf_blocker": "",
		}
		writeRenderStatus(blocked, status)
		return 3, status, nil
	}
	if err := writeRenderText(o.OutputHTML, htmlText); err != nil {
		return 1, nil, err
	}
	scrubStatus := "not_checked"
	if o.ClientFacing {
		scrubStatus = "clean"
	}
	status := map[string]any{
		"command": "package", "inputs": inputsAny, "missing_inputs": missing,
		"output_html": o.OutputHTML, "generated_at": generatedAt,
		"client_blind_terms_found": strsToAny(scrubFound), "scrub_status": scrubStatus,
		"pdf_status": "not_requested", "pdf_path": "", "pdf_blocker": "",
	}
	if o.OutputPDF != "" {
		pdfStatus, blocker := exportPDF(o.OutputHTML, o.OutputPDF)
		status["pdf_status"] = pdfStatus
		status["pdf_path"] = o.OutputPDF
		status["pdf_blocker"] = blocker
	}
	writeRenderStatus(o.OutputHTML, status)
	if status["pdf_status"] == "blocked" {
		return 2, status, nil
	}
	return 0, status, nil
}

// --- PDF export (chrome headless, wkhtmltopdf fallback) ---------------------------

func chromeCandidates() []string {
	var out []string
	if env := os.Getenv("CHROME_PATH"); env != "" {
		out = append(out, env)
	}
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "msedge"} {
		if p, err := exec.LookPath(name); err == nil {
			out = append(out, p)
		}
	}
	mac := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	if _, err := os.Stat(mac); err == nil {
		out = append(out, mac)
	}
	if runtime.GOOS == "windows" {
		for _, p := range []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		} {
			if _, err := os.Stat(p); err == nil {
				out = append(out, p)
			}
		}
	}
	return out
}

func fileURI(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = p
	}
	abs = filepath.ToSlash(abs)
	if !strings.HasPrefix(abs, "/") {
		abs = "/" + abs // windows drive letter
	}
	return "file://" + (&url.URL{Path: abs}).EscapedPath()
}

func exportPDFWithChrome(htmlPath, pdfPath string) (bool, string) {
	var errors []string
	for _, chrome := range chromeCandidates() {
		for _, headlessFlag := range []string{"--headless=new", "--headless"} {
			if err := os.MkdirAll(filepath.Dir(pdfPath), 0o755); err != nil {
				return false, err.Error()
			}
			tmp, err := os.MkdirTemp("", "solo-report-chrome-")
			if err != nil {
				return false, err.Error()
			}
			cmd := exec.Command(chrome, headlessFlag, "--disable-gpu", "--disable-dev-shm-usage",
				"--no-sandbox", "--no-first-run", "--no-default-browser-check",
				"--user-data-dir="+tmp, "--print-to-pdf="+pdfPath, fileURI(htmlPath))
			var outBuf, errBuf strings.Builder
			cmd.Stdout, cmd.Stderr = &outBuf, &errBuf
			err = runWithTimeout(cmd, 90*time.Second)
			os.RemoveAll(tmp)
			if err == errTimeout {
				errors = append(errors, filepath.Base(chrome)+" "+headlessFlag+": timeout")
				continue
			}
			if st, statErr := os.Stat(pdfPath); err == nil && statErr == nil && st.Size() > 0 {
				return true, ""
			}
			last := strings.TrimSpace(errBuf.String())
			if last == "" {
				last = strings.TrimSpace(outBuf.String())
			}
			if last == "" {
				last = fmt.Sprintf("exit err %v", err)
			}
			if len(last) > 500 {
				last = last[len(last)-500:]
			}
			errors = append(errors, filepath.Base(chrome)+" "+headlessFlag+": "+last)
		}
	}
	if len(errors) > 0 {
		if len(errors) > 6 {
			errors = errors[:6]
		}
		return false, "chrome_print_failed: " + strings.Join(errors, " ; ")
	}
	return false, "chrome_not_found"
}

var errTimeout = fmt.Errorf("timeout")

func runWithTimeout(cmd *exec.Cmd, d time.Duration) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(d):
		_ = cmd.Process.Kill()
		<-done
		return errTimeout
	}
}

func exportPDFWithWkhtmltopdf(htmlPath, pdfPath string) (bool, string) {
	binary, err := exec.LookPath("wkhtmltopdf")
	if err != nil {
		return false, "wkhtmltopdf_not_found"
	}
	cmd := exec.Command(binary, htmlPath, pdfPath)
	var outBuf, errBuf strings.Builder
	cmd.Stdout, cmd.Stderr = &outBuf, &errBuf
	runErr := runWithTimeout(cmd, 90*time.Second)
	if st, statErr := os.Stat(pdfPath); runErr == nil && statErr == nil && st.Size() > 0 {
		return true, ""
	}
	msg := strings.TrimSpace(errBuf.String())
	if msg == "" {
		msg = strings.TrimSpace(outBuf.String())
	}
	if len(msg) > 500 {
		msg = msg[len(msg)-500:]
	}
	return false, "wkhtmltopdf_failed: " + msg
}

func exportPDF(htmlPath, pdfPath string) (string, string) {
	var blockers []string
	for _, attempt := range []func(string, string) (bool, string){exportPDFWithChrome, exportPDFWithWkhtmltopdf} {
		ok, blocker := attempt(htmlPath, pdfPath)
		if ok {
			return "generated", ""
		}
		blockers = append(blockers, blocker)
	}
	return "blocked", strings.Join(blockers, " | ")
}

// --- CLI + in-binary hook ---------------------------------------------------------

func runRenderReportCLI(args []string) int {
	valueFlags := map[string]bool{"--input": true, "--output-html": true, "--output-pdf": true,
		"--title": true, "--subtitle": true, "--client-name": true, "--report-kind": true,
		"--report-date": true, "--status-note": true, "--inputs": true}
	boolFlags := map[string]bool{"--client-facing": true, "--fail-on-scrub": true}
	if len(args) == 0 {
		return crmUsageErr("render-report needs a subcommand (render | package)")
	}
	cmd := args[0]
	// argparse nargs="+" for --inputs: collect values following it until the next flag
	var rest []string
	var inputs []string
	i := 1
	for i < len(args) {
		if args[i] == "--inputs" {
			i++
			for i < len(args) && !strings.HasPrefix(args[i], "--") {
				inputs = append(inputs, args[i])
				i++
			}
			continue
		}
		rest = append(rest, args[i])
		i++
	}
	a, err := parseCLIArgs(rest, valueFlags, boolFlags)
	if err != nil {
		return crmUsageErr(err.Error())
	}
	printStatus := func(rc int, status map[string]any) int {
		if rc == 3 {
			fmt.Fprintln(os.Stderr, marshalIndentJSON(status))
		} else {
			fmt.Println(marshalIndentJSON(status))
		}
		return rc
	}
	switch cmd {
	case "render":
		if a.get("--input") == "" || a.get("--output-html") == "" {
			return crmUsageErr("render needs --input and --output-html")
		}
		rc, status, err := renderCommand(renderOpts{
			Input: a.get("--input"), OutputHTML: a.get("--output-html"), OutputPDF: a.get("--output-pdf"),
			Title: a.get("--title"), Subtitle: a.get("--subtitle"), ClientName: a.get("--client-name"),
			ReportKind: a.get("--report-kind"), ReportDate: a.get("--report-date"),
			StatusNote: a.get("--status-note"), ClientFacing: a.bools["--client-facing"],
			FailOnScrub: a.bools["--fail-on-scrub"]})
		if err != nil {
			return crmFail(err)
		}
		return printStatus(rc, status)
	case "package":
		if len(inputs) == 0 || a.get("--output-html") == "" {
			return crmUsageErr("package needs --inputs and --output-html")
		}
		rc, status, err := packageCommand(packageOpts{
			Inputs: inputs, OutputHTML: a.get("--output-html"), OutputPDF: a.get("--output-pdf"),
			Title: a.get("--title"), Subtitle: a.get("--subtitle"), ClientName: a.get("--client-name"),
			ReportKind: a.get("--report-kind"), ReportDate: a.get("--report-date"),
			StatusNote: a.get("--status-note"), ClientFacing: a.bools["--client-facing"],
			FailOnScrub: a.bools["--fail-on-scrub"]})
		if err != nil {
			return crmFail(err)
		}
		return printStatus(rc, status)
	}
	return crmUsageErr("unknown render-report subcommand " + cmd)
}

// Native renderer replaces the render.go stub: crm_store's three subprocess
// sites become in-process calls (json parse of stderr no longer needed —
// blind terms come back structured).
func init() {
	renderReportFile = func(req rendererRequest) rendererResult {
		rc, status, err := renderCommand(renderOpts{
			Input: req.Input, OutputHTML: req.OutputHTML, Title: req.Title,
			ClientName: req.ClientName, ReportKind: req.ReportKind, ReportDate: req.ReportDate,
			ClientFacing: req.ClientFacing, FailOnScrub: req.FailOnScrub})
		if err != nil {
			return rendererResult{RC: 1}
		}
		res := rendererResult{RC: rc}
		if status != nil {
			if terms, ok := status["client_blind_terms_found"].([]any); ok {
				res.BlindTerms = terms
			}
		}
		return res
	}
}

var _ = json.Marshal // keep encoding/json imported for future structured status use
