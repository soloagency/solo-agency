---
name: solo-agency-report-design
description: >-
  Use before generating, reviewing, fixing, or packaging any Solo Agency
  client-facing report HTML/PDF. Adapts landing-page design discipline from
  leonxlnx/taste-skill into a report-specific, client-blind, print-safe standard.
---

# Solo Agency Report Design Skill

This is the report-specific design module. Load it before Stage 6 report
generation or report repair, after the factual source record is ready and before
writing HTML or PDF.

Source note: this module adapts the design-read, dial, anti-default, typography,
layout, and preflight discipline from `leonxlnx/taste-skill` for Solo Agency
reports. Do not treat the original landing-page skill as a runtime dependency.
This local module is the canonical report design rule.

## Design Read

Default reading:

```text
Reading this as: agency intelligence report for a busy owner/operator, with a polished editorial landing-page language, leaning toward standalone HTML, strong hierarchy, print-safe CSS, and no remote dependencies.
```

Report dials:

- `DESIGN_VARIANCE: 7` - use asymmetry, editorial spacing, visual contrast, and memorable section rhythm.
- `MOTION_INTENSITY: 1` - reports must be static and print-safe. Hover polish is allowed in HTML, but no required animation.
- `VISUAL_DENSITY: 6` - more information than a landing page, but never a raw data dump.

## Non-Negotiables

- The report must feel like a premium agency deliverable, not exported Markdown.
- The first viewport must work like a landing-page hero: report title, client/date context, decisive recommendation, lane status, and 3-5 scan-friendly highlights.
- Client-facing reports and the PDF companion remain client-blind. Do not mention Solo Agency, WideCast, PDNA/provider tooling, OpenAPI, MCP, Local Collector, Chrome extensions, automation/scheduled tasks, API keys, Telegram, agent/debug details, or `INTERNAL_REPORT`.
- No remote CSS, JavaScript, fonts, icons, images, tracking pixels, or CDNs.
- No fake static action buttons. Use links only when they go somewhere real.
- No raw Markdown dumps, plain default browser tables, or endless equal cards.
- No document-level horizontal scroll on 390px mobile. Only dedicated table wrappers may scroll.
- No long private data source excerpts or private source inventory in client-facing files.
- The PDF source must be print-friendly without hover, copy buttons, collapses, or scripts.

## Required Report Shape

Use this order unless a specific client/report language requires a natural local
translation:

1. Hero: report promise, date, lane status, top recommendation, confidence.
2. Executive snapshot: what changed, why it matters, what to do next.
3. Source coverage and data quality: public data sources or private data sources lane state.
4. Evidence ledger: compact, reference-linked, confidence-labeled.
5. Lead & Competitor Opportunities: clear opportunity cards or a mobile-safe table.
6. Idea Matrix: grouped by audience value, pain point, source signal, and business fit.
7. Best idea: one recommended action with rationale and approval state.
8. Draft/recommendation: polished review blocks, not raw prompt output.
9. Limits and blockers: client-safe coverage limits and next action.

The daily report is a staging cover/index. It should be shorter, but the
delivered client-facing HTML must be the combined `{client-name}-client-report.html`
that includes the daily cover plus the full public data sources and private data
sources sections in one standalone file. Do not make the human/client open
separate lane HTML files.

## Visual System

Use a report-native visual language:

- Typography: modern sans stack by default. Avoid making the report look like a browser default or terminal dump.
- Color: one strong accent plus neutral ink/paper surfaces. Avoid generic AI purple/blue gradients and one-note beige/slate palettes.
- Layout: asymmetric hero, metric strip, editorial section headers, evidence cards, and deliberate whitespace.
- Cards: use cards for repeated items only. Do not put cards inside cards.
- Tables: use tables only for real comparison/ledger data. Wrap in `.table-scroll` and make them readable in print.
- Numbers: use real collected counts only. Do not invent fake-precise metrics for visual effect.
- References: every claim/lead/competitor/draft idea should carry a visible source URL or a clear unavailable note.
- Language: crisp, useful, no filler. The reader should know the next action within 60 seconds.

## Renderer Contract

Use the shared renderer instead of writing ad hoc Python, shell, browser, or PDF
scripts during each run:

```sh
python3 tools/solo_report_renderer.py render --input REPORT.md --output-html REPORT.html --title "Daily Intelligence Report" --client-name "Client Name" --report-kind "Daily Report"
```

To create the single client-facing HTML report and mandatory PDF companion from
the three scrubbed staging HTML files:

```sh
python3 tools/solo_report_renderer.py package --inputs CLIENT-daily-report.html CLIENT-public-data-sources-report.html CLIENT-private-data-sources-report.html --output-html CLIENT-client-report.html --output-pdf CLIENT-client-report.pdf --title "Client Report" --client-name "Client Name"
```

Allowed deviations:

- If the renderer is missing or fails, fix `tools/solo_report_renderer.py` or log the exact blocker. Do not replace it with a one-off report script.
- If a client has a custom approved report template, it may be layered into the renderer or a named reusable template file. Do not improvise a new unnamed renderer during the run.

## Preflight

Before handing off any report:

- [ ] Loaded this module and Stage 6 in the current turn/run.
- [ ] Used `tools/solo_report_renderer.py` or logged why the reusable renderer was unavailable.
- [ ] HTML is standalone, mobile-friendly, and visually polished.
- [ ] The first viewport has a useful hero, not a file title plus wall of text.
- [ ] Public data sources and private data sources remain separate sections inside the combined client report.
- [ ] The delivered HTML is `{client-name}-client-report.html`, not the short daily staging index.
- [ ] The combined HTML does not link out to sibling daily/public/private HTML files; any such references are internal section anchors or plain labels.
- [ ] The PDF companion was generated from the same combined client-facing HTML, or the exact PDF blocker was recorded.
- [ ] Client-Blind Scrub Gate passed for client-facing HTML/PDF.
- [ ] `INTERNAL_REPORT` contains all operations/provider/debug details excluded from client-facing files.
- [ ] No fake buttons, remote dependencies, raw Markdown dump, or mobile body overflow.
