---
name: outreachcrm-report-design
description: >-
  Use before generating, reviewing, fixing, or packaging any OutreachCRM
  report HTML/PDF (weekly client report, Approval Report, Today View, daily ops).
  Adapts landing-page design discipline from leonxlnx/taste-skill into a
  report-specific, client-blind (for the weekly client report), print-safe standard.
---

# OutreachCRM Report Design Skill

This is the report-specific design module. Load it before Stage 15 client-report
generation, Approval Report / Today View rendering, or any report repair, after the
factual source record is ready and before writing HTML or PDF.

Source note: this module adapts the design-read, dial, anti-default, typography,
layout, and preflight discipline from `leonxlnx/taste-skill` for OutreachCRM
reports. Do not treat the original landing-page skill as a runtime dependency.
This local module is the canonical report design rule.

## Report types this skill covers

- **Weekly client report** — the ONLY client-facing deliverable. Must pass the
  Client-Blind Scrub Gate. Pipeline snapshot, movements, win/loss, forecast, next
  actions — reads like a premium agency deliverable.
- **Approval Report** — operator-only, NOT scrubbed. One card per lead: dossier +
  clickable evidence URLs + editable draft (contenteditable) + Copy button + warning
  flags, split into High confidence / Review carefully groups.
- **Today View** — operator-only. Due tasks, hot replies awaiting response, deals
  past SLA, drafts awaiting approval.
- **Daily ops report** + **INTERNAL_REPORT** — operator-only, full detail.

## Design Read

```text
Reading this as: outreach/CRM operations report for a busy operator, editorial
landing-page language, standalone HTML, strong hierarchy, print-safe CSS, no
remote dependencies.
```

Report dials:

- `DESIGN_VARIANCE: 7` - asymmetry, editorial spacing, visual contrast, memorable rhythm.
- `MOTION_INTENSITY: 1` - static and print-safe. Hover polish allowed in HTML; no required animation.
- `VISUAL_DENSITY: 6` - more information than a landing page, never a raw data dump.

## Non-Negotiables

- The report must feel like a premium deliverable, not exported Markdown.
- The first viewport must work like a landing-page hero: report title, client/date
  context, the decisive takeaway, and 3-5 scan-friendly highlights.
- **The weekly client report and its PDF companion remain client-blind** (see the
  Client-Blind Scrub Gate term list in `tools/report_renderer.py` / DESIGN.md §19):
  do not mention OutreachCRM, WideCast, PDNA/provider tooling, OpenAPI, MCP, API keys,
  Telegram, automation/scheduled tasks, sendboxes, crm_store, sent_log, suppression,
  warmup, quota, guessed, tracker domains, agent/debug details, or `INTERNAL_REPORT`.
  (Operator-only reports — Approval Report, Today View, daily ops, INTERNAL_REPORT —
  are NOT scrubbed and may name any internal component.)
- No remote CSS, JavaScript, fonts, icons, images, tracking pixels, or CDNs.
- No fake static action buttons. Use links only when they go somewhere real. (The
  Approval Report's Copy button and contenteditable blocks are real, local, and
  provided by the renderer.)
- No raw Markdown dumps, plain default browser tables, or endless equal cards.
- No document-level horizontal scroll on 390px mobile. Only dedicated table wrappers may scroll.
- The PDF source must be print-friendly without hover, copy buttons, collapses, or scripts.

## Required Report Shapes

**Weekly client report:** the minimal, data-driven version is generated in one step by
`python3 tools/crm_store.py --client-dir DIR weekly-report --client-name "…"`, which assembles a
scrubbed Markdown source from CRM state and renders it client-facing (see the Renderer Contract
below). Use this skill to review/upgrade its visual polish; the required content shape is:
1. Hero: reporting period, headline outcome (e.g. "3 new opportunities, 1 won"), next step.
2. Executive snapshot: what moved this week, why it matters, what to do next.
3. Pipeline snapshot: deals by stage, values, forecast (Σ value×probability).
4. Movements: stage changes, new opportunities, wins/losses with reasons.
5. Activity summary: outreach sent, reply rate, meetings booked (client-safe framing).
6. Next actions and asks.

**Approval Report** (operator-only): header with totals by campaign/step split into
High confidence / Review carefully; one card per lead with id, name/company/email +
verify status, hooks with clickable evidence URLs, subject + editable body, warning flags.

**Today View** (operator-only): due tasks, hot replies, SLA-breach deals, drafts awaiting approval.

## Visual System

- Typography: modern sans stack by default. Avoid browser-default / terminal-dump look.
- Color: one strong accent plus neutral ink/paper surfaces. Avoid generic AI purple/blue gradients.
- Layout: asymmetric hero, metric strip, editorial section headers, cards, deliberate whitespace.
- Cards: for repeated items only. Do not put cards inside cards.
- Tables: only for real comparison/ledger data. Wrap in `.table-scroll`, readable in print.
- Numbers: real collected counts only. Do not invent fake-precise metrics. Label estimated
  metrics (opens) as estimated.
- References: every claim/opportunity/draft carries a visible source URL or a clear unavailable note.

## Renderer Contract

Use the shared renderer instead of writing ad hoc Python/shell/browser/PDF scripts:

```sh
python3 tools/report_renderer.py render --input REPORT.md --output-html REPORT.html \
  --title "Weekly Report" --client-name "Client Name" --report-kind "Weekly Report" \
  --client-facing --fail-on-scrub
```

The `--client-facing --fail-on-scrub` flags run the Client-Blind Scrub Gate and refuse
to write the real output path on a hit (writing a `.blocked.html` sidecar, exit code 3).
Use them ONLY for the weekly client report. Operator-only reports (Approval Report,
Today View, daily ops, INTERNAL_REPORT) are rendered WITHOUT `--client-facing`.

To combine scrubbed staging fragments into the client-facing HTML + PDF companion:

```sh
python3 tools/report_renderer.py package --inputs PART1.html PART2.html \
  --output-html CLIENT-weekly-client-report.html --output-pdf CLIENT-weekly-client-report.pdf \
  --title "Weekly Report" --client-name "Client Name" --client-facing --fail-on-scrub
```

Allowed deviations:

- If the renderer is missing or fails, fix `tools/report_renderer.py` or log the exact
  blocker. Do not replace it with a one-off report script.
- A client's custom approved template may be layered into the renderer or a named
  reusable template file. Do not improvise a new unnamed renderer during the run.

## Preflight

- [ ] Loaded this module and the relevant reporting stage in the current turn/run.
- [ ] Used `tools/report_renderer.py` or logged why the reusable renderer was unavailable.
- [ ] HTML is standalone, mobile-friendly, and visually polished.
- [ ] The first viewport has a useful hero, not a file title plus wall of text.
- [ ] For the weekly client report: rendered with `--client-facing --fail-on-scrub` and the
      scrub gate passed (exit 0, no `.blocked` sidecar).
- [ ] Operator-only reports were NOT scrubbed (they must retain internal detail).
- [ ] The PDF companion was generated from the same client-facing HTML, or the exact PDF blocker recorded.
- [ ] No fake buttons, remote dependencies, raw Markdown dump, or mobile body overflow.
