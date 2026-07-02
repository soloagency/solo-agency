# Agency Report Standard

Stage: `06`

## Load Rule

Load whenever generating, reviewing, debugging, or improving a human-facing report.

Immediately after loading this file, load `playbooks/skills/report-design/SKILL.md`
before writing, regenerating, reviewing, packaging, or fixing report HTML/PDF.
The report-design skill is the report-specific adaptation of the landing-page
design discipline from `leonxlnx/taste-skill`; Stage 6 remains the operational
contract, and the report-design skill controls visual quality.

## Hard Gates For This Stage

- Canonical human-facing report output is one standalone HTML file per client/day/run: `{client-name}-client-report.html`. A PDF companion generated from that same HTML is mandatory when export is available and safe.
- Markdown is internal.
- The report must be standalone, mobile-friendly, agency-grade, and factually aligned with the Markdown source.
- The report must use the reusable report-design module and reusable renderer path by default. Do not write one-off Python/HTML/PDF scripts for ordinary report generation.
- Include reference URLs beside claims, ideas, leads, competitors, and drafts.
- Every idea, best idea, comment, and draft must be audience-value-first: useful to the viewer before useful to the client's brand. Reject or rewrite direct product/service praise as `promotional_not_value_first`.
- Do not create fake action buttons in static HTML.
- Keep exactly one canonical client-facing report file per client/day/run. Daily/public/private HTML files may be generated as scrubbed staging inputs for lane isolation, but the file handed to the human/client or uploaded through a provider must be the combined `{client-name}-client-report.html`.
- Client-facing report files and the PDF companion must be client-blind: no Solo Agency, WideCast, PDNA/provider tooling, OpenAPI, MCP, Local Collector, Chrome extension, automation/scheduled-task, API-key/config, Telegram, agent/tool/debug, or `INTERNAL_REPORT` details.
- Every run must also create an operator-only `{client-name}-INTERNAL_REPORT.html` clearly labeled `INTERNAL_REPORT - Not for client sharing`.
- Never let a later public/private pass overwrite or summarize away the other lane. Keep lane staging files separate for generation/state safety, then combine them into the one client-facing HTML report.
- The PDF companion must be generated from `{client-name}-client-report.html`, not from raw memory, `INTERNAL_REPORT`, or a short daily index. If PDF export or safe redaction is blocked, record the exact blocker and still provide the combined HTML report path/link.

## Source Preservation Rule

This file is detailed source material moved from the original monolithic `SOLO_AGENCY_PLAYBOOK.md`.

Do not summarize away requirements, examples, checklists, schemas, protocols, URLs, edge cases, warnings, approval gates, or completion gates. If a downstream agent needs to shorten its response to the human, it may summarize the response, but it must still obey the full requirements in this file.

---

## 11. Daily Output File Format

Each active client must have one daily output folder:

```text
outputs/YYYY-MM/YYYY-MM-DD/
```

Markdown is the canonical internal output record. Client-facing HTML is the canonical rendered report. PDF is a mandatory companion derived from the client-facing HTML report set so the recipient can choose HTML or PDF. The operator-only `INTERNAL_REPORT` is a separate diagnostic/operations report and must not be sent to the client.

The agent must keep the Markdown file even when an HTML report is created, unless the current environment truly cannot write Markdown. The Markdown file is required for:

- agent-readable history;
- future learning and optimization;
- duplicate idea detection;
- diffing changes across days;
- regenerating HTML reports;
- preserving references, reasoning, leads, competitors, provider-backed drafts, and operational notes without parsing HTML.

The HTML report must be created from the same facts, references, ideas, analysis, and draft content as the Markdown report, but it may use a custom structure and design. It must not become a factually divergent report. If the agent can only preserve one long-term artifact, preserve the Markdown file first and regenerate HTML later. If the agent can only deliver one artifact to the human, deliver the HTML report because all user-facing reports must be HTML.

### Human-Facing Report Rule: HTML Plus PDF Companion

The agent must show report results to the human as one combined HTML report and provide the PDF companion status/path in the same handoff. The PDF is a derivative of that same combined HTML; the daily/public/private HTML files are staging artifacts for automation updates, not separate files for the human to open.

Do not show, send, link, or ask the human to open the Markdown report as the user-facing report.

Allowed:

- Save `.md` internally for agent memory, history, learning, diffing, and regeneration.
- Mention that a Markdown source file exists only when explaining internal storage or troubleshooting.
- Deliver the combined `{client-name}-client-report.html` path/link to the human.
- Send Telegram/WideCast notifications with the `.html` path/link.
- Deliver a `.pdf` companion alongside the `.html` path/link, or state the exact PDF blocker/status if generation is unavailable or unsafe.

Not allowed:

- "Open the Markdown report."
- "See `outputs/YYYY-MM/YYYY-MM-DD.md` for details."
- Using the `.md` file as the primary human-facing report.
- Sending both `.md` and `.html` and making the human decide which one to open.
- Treating a PDF export as the canonical report source or using it to replace the three HTML files.

Every default report path, notification, or review instruction must point to `{client-name}-client-report.html` as the primary review link and include the `.pdf` companion path/status as the secondary share-ready artifact.

### Internal Markdown, Beautiful HTML

The agent must create two artifacts with different purposes:

1. Markdown is the internal canonical record for agent memory, history, learning, diffing, and regeneration.
2. Client-facing HTML is the combined polished agency report for review/sharing.
3. `INTERNAL_REPORT` is the operator-only system report for automation, PDNA/provider, delivery, and collector details.

The HTML report does not need to be a direct Markdown render. If a direct Markdown-to-HTML renderer produces an ugly or hard-to-use report, the agent should create a better designed standalone HTML report instead.

Correct behavior:

1. Author and save the complete internal report source under `outputs/YYYY-MM/YYYY-MM-DD/`, using stable client-prefixed names.
2. Create exactly three scrubbed staging HTML files for the client/day/run:
   - `{client-name}-public-data-sources-report.html`
   - `{client-name}-private-data-sources-report.html`
   - `{client-name}-daily-report.html`
3. Package those staging files into the single client-facing report:
   - `{client-name}-client-report.html`
4. Create one operator-only internal report:
   - `{client-name}-INTERNAL_REPORT.html`
5. `{client-name}` must be a filesystem-safe client name/slug, lower-kebab preferred, for example `angela-do` or `aven-ngo`.
6. The public and private staging HTML files are full lane reports, not summaries.
7. The daily staging HTML file is the concise cover/index for the combined report, showing lane status, client-relevant blockers/limits, best next action, and delivery status without internal tooling details.
8. The combined client-facing HTML must include the daily cover, full public lane, and full private lane/status in one standalone file. It must not require the reader to open or click into sibling HTML files.
9. The HTML may be custom structured and styled for readability, mobile scanning, editable draft review, and copy workflow.
10. The client-facing HTML must not omit required client-relevant report sections that exist in the corresponding Markdown/source record, but it must scrub internal system/provider/collector details into `INTERNAL_REPORT`.
11. If a Markdown/source record changes, update/regenerate only the affected lane staging HTML plus the daily index, then rebuild `{client-name}-client-report.html` and its PDF companion so the delivered HTML/PDF stay identical in content.

Quality rules for HTML:

- The HTML must be standalone and portable.
- The HTML must not depend on remote JavaScript, remote CSS, CDN libraries, login, internet access, or the `.md` file being present next to it.
- The HTML should use thoughtful layout, spacing, typography, color, cards/sections, and mobile-friendly hierarchy.
- Do not dump raw Markdown into the page if that makes the report ugly.
- Do not rely on `fetch("./report.md")`.
- Escape user/source text safely before rendering it into HTML.
- Versioned draft sections such as `Version 1: VE — Value Explainer` must be presented as polished editable review blocks with local copy buttons.
- The agent may spend extra time generating a beautiful HTML report because the HTML is the only report the human sees.
- The page must not create document-level horizontal scrolling on a 390px-wide mobile viewport. Wide tables must be wrapped in a dedicated `.table-scroll` or equivalent container with `overflow-x: auto`, or transformed into stacked mobile cards. The body, main containers, cards, buttons, code blocks, URLs, and long source names must use responsive width constraints plus `overflow-wrap: anywhere` or equivalent so only the table wrapper scrolls, never the entire page.

### Reusable Report Design And Renderer Contract

The agent must not spend every daily run inventing a fresh report rendering
script. Solo Agency includes a reusable report design skill and renderer:

```text
playbooks/skills/report-design/SKILL.md
tools/solo_report_renderer.py
```

Required order for every client-facing report:

1. Author or update the internal Markdown/source record with complete facts, references, lane markers, drafts, blockers, and operational notes.
2. Load `playbooks/skills/report-design/SKILL.md`.
3. Render each client-facing report from the approved source content with `tools/solo_report_renderer.py render`, or a named reusable template layered into that renderer.
4. Run the Client-Blind Scrub Gate on each client-facing HTML file.
5. Build `{client-name}-client-report.html` and `{client-name}-client-report.pdf` with `tools/solo_report_renderer.py package` from the scrubbed daily/public/private HTML files. This package HTML is the only default human/client-facing report link.
6. If PDF export is unavailable, keep the package HTML, write the renderer status JSON, update `report_state.json`, and record the exact blocker in `INTERNAL_REPORT`.

Default render command pattern:

```sh
python3 tools/solo_report_renderer.py render \
  --input outputs/YYYY-MM/YYYY-MM-DD/{client-name}-public-data-sources-report.md \
  --output-html outputs/YYYY-MM/YYYY-MM-DD/{client-name}-public-data-sources-report.html \
  --title "Public Data Sources Report" \
  --client-name "{Client Name}" \
  --report-kind "Public Data Sources Report" \
  --client-facing \
  --fail-on-scrub
```

Default package command pattern:

```sh
python3 tools/solo_report_renderer.py package \
  --inputs \
    outputs/YYYY-MM/YYYY-MM-DD/{client-name}-daily-report.html \
    outputs/YYYY-MM/YYYY-MM-DD/{client-name}-public-data-sources-report.html \
    outputs/YYYY-MM/YYYY-MM-DD/{client-name}-private-data-sources-report.html \
  --output-html outputs/YYYY-MM/YYYY-MM-DD/{client-name}-client-report.html \
  --output-pdf outputs/YYYY-MM/YYYY-MM-DD/{client-name}-client-report.pdf \
  --title "{Client Name} Client Report" \
  --client-name "{Client Name}" \
  --client-facing \
  --fail-on-scrub
```

Renderer behavior:

- `render` creates standalone responsive HTML with a landing-page-like hero, navigation, polished sections, mobile-safe tables, and print CSS.
- `package` combines scrubbed staging HTML files into the single client-facing HTML report and attempts PDF export from that same HTML through local browser print-to-PDF, WeasyPrint, or `wkhtmltopdf` when available.
- `package` must not leave links that send the reader to sibling daily/public/private HTML files. Any such references must become internal anchors within `{client-name}-client-report.html` or plain section labels.
- Every renderer run writes `{output_html}.render_status.json` so `INTERNAL_REPORT` and `report_state.json` can record generated/blocked PDF status without parsing terminal output.
- The renderer uses no remote CSS, JavaScript, fonts, or external services.

Fallback rule:

- If the renderer is missing or fails because of a bug, fix the reusable renderer or log `report_renderer_blocked`.
- Do not replace it with a one-off script unless the human explicitly requests a custom report template for that client. If a custom template is approved, save it as a reusable named template and update this Stage 6 contract or the client profile so future runs reuse it.

The latest convenience files should be:

```text
outputs/latest/{client-name}-daily-report.html
outputs/latest/{client-name}-public-data-sources-report.html
outputs/latest/{client-name}-private-data-sources-report.html
outputs/latest/{client-name}-INTERNAL_REPORT.html
outputs/latest/{client-name}-client-report.html
outputs/latest/{client-name}-client-report.pdf
```

The latest client report file is the default client-ready convenience link. Daily/public/private latest files are staging/debug convenience copies and must not be the primary handoff unless the human explicitly asks for a lane-only diagnostic. The internal latest file is for the user/operator only. The PDF latest file is the mandatory client-ready companion deliverable when PDF generation is available and safe.

### Latest Override: Single Client Report Contract With Lane Staging

Every client/day/run must produce one canonical client-facing report file:

```text
outputs/YYYY-MM/YYYY-MM-DD/{client-name}-client-report.html
```

The staging set used to build it has exactly these HTML files:

```text
outputs/YYYY-MM/YYYY-MM-DD/{client-name}-public-data-sources-report.html
outputs/YYYY-MM/YYYY-MM-DD/{client-name}-private-data-sources-report.html
outputs/YYYY-MM/YYYY-MM-DD/{client-name}-daily-report.html
```

File responsibilities:

1. `{client-name}-public-data-sources-report.html`
   - Full public data sources report only.
   - Must contain public source coverage, public evidence, public Lead & Competitor Opportunities, public idea matrix, best public idea, and public draft/recommendation.
   - Must not include private data source findings except a status pointer such as `private data sources pending`, `private data sources blocked`, or a link to the private report.

2. `{client-name}-private-data-sources-report.html`
   - Full private data sources report only.
   - Must contain private source coverage, safe summarized private evidence, private Lead & Competitor Opportunities, private idea matrix, best private idea, copy-ready comments when available, and private draft/recommendation.
   - Must not contain Local Collector, Chrome extension, login/session, API, raw private post/member, or private source inventory details.
   - Must not rewrite or summarize the public data sources report.

3. `{client-name}-daily-report.html`
   - Concise cover/index/overview for the combined report.
   - May reference the public/private sections, but the final packaged report must use internal section anchors, not sibling-file links.
   - Must show each lane's status, top recommendation summary, client-relevant blockers/limits, delivery status, and the one next action.
   - Must not show provider, notification-channel, automation, API-key/config, Local Collector, or debug details.
   - Must not replace either full lane report.

4. `{client-name}-client-report.html`
   - The only default client-facing HTML handoff/upload file.
   - Must include the daily cover plus the full public and private lane content/status in one standalone HTML file.
   - Must not require the reader to open `daily-report.html`, `public-data-sources-report.html`, or `private-data-sources-report.html`.
   - Must not contain sibling-file links to those staging files; rewrite them to internal anchors or section labels during packaging.

Both full lane reports must use the same structure:

- Source coverage and data quality.
- Data points and evidence ledger.
- Lead & Competitor Opportunities for that lane.
- Idea Matrix.
- Best idea for that lane.
- Draft/recommendation for that lane.
- Client-relevant blockers/limits, skipped public sources or safely summarized private coverage limits, and confidence notes.

The private lane usually has richer post/current URLs and copy-ready comments. Public data source opportunities should also include copy-ready comments when there is a concrete public post/context where a comment is safe and useful. If the public source does not support a safe comment action, keep the field and state `not available from this public data source` or the same meaning in the report language.

The Markdown/source record may keep explicit section markers for internal continuity:

```md
<!-- SOLO_AGENCY_SECTION:PUBLIC_START -->
## Public Data Source Intelligence
...
<!-- SOLO_AGENCY_SECTION:PUBLIC_END -->

<!-- SOLO_AGENCY_SECTION:PRIVATE_START -->
## Private Data Source Intelligence
...
<!-- SOLO_AGENCY_SECTION:PRIVATE_END -->
```

The staging files must keep public and private evidence split while generating and updating. The delivered `{client-name}-client-report.html` must combine them into one readable report with clearly separated public data sources and private data sources sections.

Update rules:

- Public pass: create or replace only `{client-name}-public-data-sources-report.html`, then create/update `{client-name}-daily-report.html` with private status `pending`, `blocked`, `skipped`, or the exact blocker, then rebuild `{client-name}-client-report.html` and PDF from the staging files.
- Private pass: create or replace only `{client-name}-private-data-sources-report.html`, then create/update `{client-name}-daily-report.html`, then rebuild `{client-name}-client-report.html` and PDF from the staging files. Do not rewrite, summarize away, delete, or regenerate the public report file.
- If private data sources finish after public data sources, update only the private report and daily index. Do not open/rewrite the public report except to repair broken links with explicit reason.
- After a private data source pass reaches a terminal state (`complete`, `complete_live_scan`, `blocked`, `failed`, `skipped`, or equivalent), reconcile the lane status and counts across the private lane report, daily index, internal Markdown/source record, report state JSON, notification log entry, and `outputs/latest/` copies before handoff. The same run must not say `scan in progress` or `partial` in one artifact while another artifact says `complete`.
- Reconciled counts must include sources configured, sources attempted, sources completed, sources blocked/skipped, data points kept, hot/warm/watch leads, competitors, new private data sources recommended, noisy/skipped discovery candidates, and notifications attempted/sent/blocked when those concepts exist in the run.
- If private data sources fail, time out, are stale, or are blocked, update only the private report with the exact blocker and update the daily index lane status.
- If a private pass starts but the public report is missing, create a public report placeholder file that says public data sources were not run or were unavailable, then create the private report and rebuild the combined client report. Do not create a private-only report without the daily cover and combined package.

### Operator Internal Report Contract

Every client/day/run must create or update an operator-only report:

```text
outputs/YYYY-MM/YYYY-MM-DD/{client-name}-INTERNAL_REPORT.html
outputs/latest/{client-name}-INTERNAL_REPORT.html
```

When a Markdown source is useful, save:

```text
outputs/YYYY-MM/YYYY-MM-DD/{client-name}-INTERNAL_REPORT.md
```

The internal report must begin with:

```text
INTERNAL_REPORT - Not for client sharing
```

The internal report is where the agent must put details that are useful to the user/operator but inappropriate for a client-facing deliverable:

- Solo Agency run identity, playbook version/freshness, automation freshness check, scheduled task/manifest status, resync status, and issue/recovery tracking.
- PDNA provider status: current provider such as WideCast, provider config path, API-key/config status, OpenAPI discovery status, account verification identity, provider capabilities, provider health, and redacted provider call logs.
- WideCast-specific operator details when configured: Telegram/email fallback status, connected social platforms, credits/plan when available through the verified client account, upload/notification/render/publish/analytics operation status, and exact blockers.
- Private data source inventory: approved, pending, blocked, daily/weekly/optional sources, discovery candidates, skipped/noisy sources, and access/membership notes.
- Local Collector and extension status: bridge status, config/output/run-now paths, extension instance, Chrome profile guidance, last check time, source/job status, and collector blockers.
- Report delivery log: client-facing HTML path, client PDF path/status, uploaded URLs/TTL when available, notification attempts, blockers, and correction notifications.
- Count/status reconciliation: public/private source counts, data point counts, lead/competitor counts, recommended-source counts, timestamps, and stale-artifact checks.
- Next operator action.

The internal report may mention Solo Agency, WideCast, Local Collector, providers, OpenAPI, MCP, API keys, Telegram, Chrome extension, automation, scheduled tasks, config files, and debug details. Those terms must stay out of client-facing files.

### Client-Blind Scrub Gate

Before handing off or exporting any client-facing HTML/PDF/video/blog/social/caption/comment, scan it for internal system references. Client-facing output must not mention:

```text
Solo Agency
WideCast
INTERNAL_REPORT
Local Collector
Chrome extension
MCP
OpenAPI
API key
Telegram
PDNA provider
provider_config
Client tools
global MCP
automation
scheduled task
collector bridge
agent debug
```

Do not include a footer such as `Powered by Solo Agency`, `Created with Solo Agency`, `Generated by WideCast`, or equivalent. If a footer is needed, keep it neutral or agency-owned, for example:

```text
Prepared for {Client} by {Agency/User}
```

If the configured agency/user name is unknown, omit the footer instead of inserting Solo Agency or WideCast. If a word such as `provider` appears as a normal industry term, for example `insurance provider`, it is allowed only when it is clearly unrelated to tooling/PDNA/provider config.

### Client HTML And PDF Companion Contract

After creating or updating the daily/public/private staging HTML files, create or update the single client-facing report package:

```text
outputs/YYYY-MM/YYYY-MM-DD/{client-name}-client-report.html
outputs/latest/{client-name}-client-report.html
outputs/YYYY-MM/YYYY-MM-DD/{client-name}-client-report.pdf
outputs/latest/{client-name}-client-report.pdf
```

The package HTML must be a standalone, mobile-friendly, print-friendly, client-blind document assembled from scrubbed staging files:

1. `{client-name}-daily-report.html` for the cover, executive snapshot, lane status, blockers, delivery status, and next action.
2. `{client-name}-public-data-sources-report.html` for the public data sources section.
3. `{client-name}-private-data-sources-report.html` for the private data sources section when it exists and is approved for client sharing.

Do not build the package or PDF directly from memory, from only one lane, or from `INTERNAL_REPORT`. If any staging HTML file is missing or blocked, include a clear client-safe status page in `{client-name}-client-report.html` instead of inventing content. Then export the PDF from `{client-name}-client-report.html` when safe, or record the exact blocker in `INTERNAL_REPORT` and `report_state.json`.

The combined `{client-name}-client-report.html` must be content-equivalent to the PDF companion. The reader must not need to open separate daily/public/private HTML files to see Idea Matrix, Lead & Competitor Opportunities, Best Idea, drafts, or lane status.

PDF formatting rules:

- The client report HTML/PDF source must not depend on remote JavaScript, remote CSS, collapsible UI, copy buttons, or hover-only interactions for core readability.
- Add print CSS with readable page margins, stable headings, page breaks between major sections, visible URLs/references, and no clipped cards or horizontal scrolling.
- Convert interactive HTML controls into plain text before PDF export.
- Preserve source references, dates, confidence notes, blockers, and human approval status.
- Run the Client-Blind Scrub Gate before PDF export. The PDF must not contain Solo Agency, WideCast, provider tooling, OpenAPI, MCP, Local Collector, Chrome extension, automation, scheduled task, API-key/config, Telegram, or debug details.
- The PDF should be client-shareable and polished, and it must be exported from the same combined client report HTML. The daily/public/private staging files remain automation update inputs, not separate client handoff files.

Private data source safety rules:

- Do not include raw private post text, private group member details, login/session information, collector internals, cookies, screenshots, browser/profile details, private source inventory, or unapproved private source URLs/excerpts.
- Include private data source findings only as safe agency-style summaries, for example `Community signal: several local homeowners discussed rate increases and coverage confusion.`
- If sharing safety is uncertain, mark `client_pdf_redaction_status: needs_human_review` and create only `{client-name}-client-report.html` for review, not the final PDF.

PDF generation rule:

- Use a reliable local HTML-to-PDF renderer when available, such as browser print-to-PDF or an equivalent PDF engine.
- If PDF generation is unavailable in the current AI/runtime environment, still create the print-friendly `{client-name}-client-report.html`, record `pdf_generation_blocked`, and tell the human the exact blocker.
- Do not upload the PDF through a provider unless the current client's verified provider config/OpenAPI capabilities support the file upload/notification path. If provider PDF upload is unavailable, still include the local PDF path/status beside the HTML link.

State file:

Each report should have a sibling state file:

```text
outputs/YYYY-MM/YYYY-MM-DD/{client-name}-report_state.json
```

The state file must track at least:

```json
{
  "client_slug": "",
  "run_id": "",
  "report_dir": "outputs/YYYY-MM/YYYY-MM-DD/",
  "report_md_path": "",
  "public_report_html_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-public-data-sources-report.html",
  "private_report_html_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-private-data-sources-report.html",
  "daily_report_html_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-daily-report.html",
  "client_report_html_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-client-report.html",
  "latest_client_html_path": "outputs/latest/{client-name}-client-report.html",
  "internal_report_md_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-INTERNAL_REPORT.md",
  "internal_report_html_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-INTERNAL_REPORT.html",
  "latest_internal_report_html_path": "outputs/latest/{client-name}-INTERNAL_REPORT.html",
  "public_section_status": "missing|pending|complete|skipped|failed",
  "private_section_status": "missing|pending|complete|complete_live_scan|skipped|failed|blocked",
  "internal_report_status": "pending|generated|blocked",
  "client_facing_scrub_status": "pending|pass|failed|blocked",
  "client_facing_scrub_blocker": "",
  "public_data_sources_count": 0,
  "private_data_sources_count": 0,
  "public_sources_attempted": 0,
  "public_sources_completed": 0,
  "public_sources_blocked_or_skipped": 0,
  "private_sources_attempted": 0,
  "private_sources_completed": 0,
  "private_sources_blocked_or_skipped": 0,
  "public_data_points_kept": 0,
  "private_data_points_kept": 0,
  "public_lead_count": 0,
  "private_lead_count": 0,
  "public_watch_lead_count": 0,
  "private_watch_lead_count": 0,
  "public_competitor_count": 0,
  "private_competitor_count": 0,
  "public_new_sources_recommended_count": 0,
  "private_new_sources_recommended_count": 0,
  "private_noisy_or_skipped_discovery_candidates_count": 0,
  "counts_reconciled_at": "",
  "last_public_update_at": "",
  "last_private_update_at": "",
  "public_notification_status": "not_sent|sent|skipped",
  "private_notification_status": "not_sent|sent|skipped",
  "client_report_pdf_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-client-report.pdf",
  "latest_client_pdf_path": "outputs/latest/{client-name}-client-report.pdf",
  "client_pdf_status": "pending|pending_review|generated|blocked",
  "client_pdf_redaction_status": "not_needed|redacted|approved_exact_sources|needs_human_review",
  "client_pdf_generated_at": "",
  "client_pdf_blocker": "",
  "last_notification_report_path": "",
  "last_notification_lane": "daily|public|private"
}
```

Before writing a report, the agent must read the existing source/state file when present. If the state file says a lane is complete, a later pass may update only that lane's HTML file and the daily index. It must not regenerate the other lane from memory in a way that drops detail.

Notification rule:

- Two notifications are acceptable: one after the public report is ready and one after the private report is ready/blocked.
- Notifications to the user/operator must normally point to `{client-name}-client-report.html` or its uploaded URL. Daily/public/private staging links must not be the primary report link and should be omitted unless the human explicitly asks for diagnostic lane files.
- Notifications to the user/operator must include the PDF companion path/status beside the HTML link and should include `{client-name}-INTERNAL_REPORT.html` as an operator-only secondary link/path.
- The notification text must say whether the report set is `public_report_ready`, `private_report_ready`, `private_report_blocked`, or `daily_report_ready`.
- Do not send repeated notifications for the same lane in the same run unless correcting a missing/broken report link.
- Log each notification with lane, report path/URL, and report state.

Template:

```md
# Daily Content Pipeline Output: YYYY-MM-DD

## Client Context

- Client:
- Business:
- Industry:
- Sub-industry:
- Related industries:
- Content mix rule: approximately 80% primary industry / 20% related industries
- Recent content mix status:
- Target audience:
- Target location:
- Key pain points:
- Content pillars:
- Business offer:
- Platforms:

## Internal Source Lane Order

The sections below are mandatory for the internal source record. Public data source intelligence must appear first. Private data source intelligence must appear second. Internal operational details must appear only inside the `INTERNAL_REPORT` section. The detailed field templates later in this document are schemas to apply inside each lane. The client-facing HTML must still be split into `{client-name}-public-data-sources-report.html`, `{client-name}-private-data-sources-report.html`, and `{client-name}-daily-report.html`; do not render one mixed global HTML body that combines public data sources and private data sources.

<!-- SOLO_AGENCY_SECTION:PUBLIC_START -->
## Public Data Source Intelligence

### Public Source Coverage And Data Quality

### Public Data Points And Evidence Ledger

### Public Lead & Competitor Opportunities

### Public Idea Matrix

### Best Public Idea

### Public Draft / Recommendation

### Public Blockers And Limits
<!-- SOLO_AGENCY_SECTION:PUBLIC_END -->

<!-- SOLO_AGENCY_SECTION:PRIVATE_START -->
## Private Data Source Intelligence

### Private Collector Health

### Private Data Source Discovery

### Private Source Coverage And Data Quality

### Private Data Points And Evidence Ledger

### Private Lead & Competitor Opportunities

### Private Idea Matrix

### Best Private Idea

### Private Draft / Recommendation

### Private Blockers And Limits
<!-- SOLO_AGENCY_SECTION:PRIVATE_END -->

<!-- SOLO_AGENCY_SECTION:INTERNAL_REPORT_START -->
## INTERNAL_REPORT - Not for client sharing

### Run Status And Automation Freshness

### PDNA Provider Status

### WideCast / Provider Account And Capability Status

### Telegram And Social Platform Connections

### Private Data Sources Inventory

### Local Collector And Extension Health

### Report Delivery And Notification Log

### Count And Status Reconciliation

### Issues, Blockers, Recovery Actions

### Next Operator Action
<!-- SOLO_AGENCY_SECTION:INTERNAL_REPORT_END -->

## Private Collector Health

- Bridge status:
- Bridge persistent mode:
- Extension status: recent | stale | no_extension_check_yet | unavailable
- Last extension check:
- Seconds since last extension check:
- Chrome/session status:
- Private collection impact:
- Required human action:

## Private Data Sources Pending Activation

Use this section inside the internal source record and `INTERNAL_REPORT` when private data sources were provided but the Solo Agency Local Collector extension and Local Collector app are not activated yet. In client-facing reports, translate this into a client-safe coverage note such as `private community sources pending` without naming Solo Agency, Local Collector, Chrome extensions, or internal setup mechanics.

- Status: pending_private_activation | activated | not_provided | unavailable
- Why private data sources were not scanned today:
- What is needed to activate them:
- Suggested next question:
  - `Private data sources (logged-in/social/community places such as groups, profiles, pages, channels, forums, or communities) are not activated yet because they require the Local Collector app and Chrome extension on your computer. Do you want me to prepare the setup files and then give you the two required local steps: run one Terminal/PowerShell command yourself and load the Chrome extension from the folder I show you?`
- Sources waiting for activation:
  - Source:
    - URL:
    - Platform:
    - Why it matters:

## Private Data Source Discovery

Use this section when the human approved, declined, postponed, or has not yet been asked about optional private data source discovery from joined groups/subreddits/communities, followed profiles/pages/KOLs, subscribed channels, or platform recommendation feeds.

- Status: not_asked | recommended | declined | postponed | approved_pending_activation | pending_human_approval | active | blocked | completed | discovery_declined_or_postponed
- Display title, when useful: `Private Data Source Discovery Recommended`, `Private Data Source Discovery Pending Approval`, `Private Data Source Discovery Declined/Postponed`, or the same meaning translated into the report language.
- Why this matters:
  - If no private data sources are active, the report may miss many community discussions, lead signals, competitor posts, objections, and niche content ideas from logged-in/member spaces.
- Recommended next action:
  - If status is `not_asked` or `recommended`, ask whether the human wants a one-time discovery pass through approved joined groups, subreddits, communities, followed pages/KOLs, subscribed channels, and feeds.
  - If status is `pending_human_approval`, ask the human to approve, remove, or add candidate sources before monitoring begins.
  - If status is `declined`, `postponed`, or `discovery_declined_or_postponed`, do not nag, but keep the coverage limitation visible.
- Reassurance shown:
  - Professional agency-scale setup, normally one-time:
  - Local-only data safety:
  - Daily scanning reduces missed signals:
- Approved discovery categories:
  - membership_sources:
  - following_sources:
  - recommendation_feed_sources:
- Discovery URLs used or pending:
  - Platform:
  - Discovery type:
  - URL:
  - Status: pending_private_activation | scanned | login_required | platform_url_changed | failed
- Candidate sources found:
  - Source name:
  - Source URL:
  - Platform:
  - Discovery category:
  - Why relevant:
  - Matched pain points:
  - Matched content pillars:
  - Industry scope: primary_industry | related_industry
  - Recommended cadence: daily | weekly | optional | watch_once
  - Classification: recommended_daily | recommended_weekly | optional | watch_once | skip_not_relevant | skip_too_broad | skip_too_noisy | skip_sensitive_or_risky | skip_platform_unavailable
  - Approval status: pending_human_approval | approved | rejected
- Feed signals detected:
  - Topic/signal:
  - Source/current URL:
  - Why it matters:
  - Suggested action:
- Human approval needed:
  - Which candidate sources should be approved before activation:

## Sources Checked

### Public Data Sources

- Source:
  - URL:
  - Notes:

### Public Search Keywords Used Today

- Keyword:
  - Keyword type: primary_industry | related_industry | local | pain_point | audience | news | buying_intent | other
  - Search tool used: Google Search | web search | other
  - Status: used | useful | weak | retry_later
  - Result quality: strong | medium | weak | irrelevant
  - Useful URLs found:
    - URL:
      - Why useful:
  - Follow-up keyword, if any:
  - Notes:

### Private Data Sources

- Source:
  - URL captured:
  - Login/session status:
  - Notes:

## Sources Skipped

| Source | URL | Reason | Next Action |
|---|---|---|---|
|  |  |  |  |

## New Private Data Sources Detected

- Source:
  - Platform:
  - Source type: group | page | profile | community | creator | forum | other
  - Profile/group URL:
  - Current recommendation URL:
  - Detected while scanning:
  - Why it may be relevant:
  - Related content pillar:
  - Estimated priority: high | medium | low
  - Suggested scan cadence: daily | weekly | optional
  - Status: needs_human_review | added | skipped
  - Notes:

## Data Points Collected

- Data point:
  - Source:
  - Reference URL:
  - Source type: public | private
  - Captured at:
  - Inference type: direct | adjacent
  - Adjacent logic chain, if any:
  - Confidence: high | medium | low
  - Relevance:

## Lead & Competitor Opportunities

Use this section title exactly for English reports, or translate it naturally into the human/report language.

- Opportunity:
  - Opportunity type: lead | competitor | both
  - Classification: hot_lead | warm_lead | watch_lead | direct_competitor | indirect_competitor | adjacent_solution | attention_competitor | authority_or_kol_competing_for_trust
  - Source:
  - Platform:
  - Source type: public | private
  - Profile URL:
  - Post/current URL:
  - Captured at:
  - Safe context summary:
  - Why this matters:
  - Related offer:
  - Related pain point:
  - Confidence: high | medium | low
  - Suggested human action:
  - Copy-ready comment:
  - Comment language:
  - Comment style note:
  - Outreach/compliance note:

## Leads Detected

This may be kept as a detailed subsection, but the human-facing HTML should prioritize the lane-specific `Public Lead & Competitor Opportunities` and `Private Lead & Competitor Opportunities` sections.

### Hot Leads

- Lead:
  - Source:
  - Profile URL:
  - Post/current URL:
  - Source type: public | private
  - Captured at:
  - Safe summary:
  - Why this is hot:
  - Related offer:
  - Related pain point:
  - Suggested next action:
  - Copy-ready suggested comment:
  - Outreach/compliance note:

### Warm Leads

- Lead:
  - Source:
  - Profile URL:
  - Post/current URL:
  - Source type: public | private
  - Captured at:
  - Safe summary:
  - Why this is warm:
  - Related offer:
  - Related pain point:
  - Suggested next action:
  - Copy-ready suggested comment:
  - Outreach/compliance note:

## Competitors Detected

This may be kept as a detailed subsection, but the human-facing HTML should prioritize the lane-specific `Public Lead & Competitor Opportunities` and `Private Lead & Competitor Opportunities` sections.

- Competitor:
  - Competitor type: direct | indirect | adjacent | attention | authority_or_kol
  - Platform:
  - Profile URL:
  - Post/current URL:
  - Location relevance:
  - Audience overlap:
  - Offer/positioning:
  - Content themes:
  - Engagement signal:
  - Threat level: high | medium | low
  - Opportunity:
  - Recommended monitoring action:
  - Copy-ready suggested comment:

## Idea Matrix

The 3x2 matrix is six idea buckets, not six total ideas. Under each Global/Local bucket below, repeat the idea block for every credible, source-backed idea harvested from today's data for that layer and scope. Do not discard useful ideas because the bucket already contains one idea.

### 1. Hot / Trend / News

#### Global

- Idea (repeat this block for every credible idea in this bucket):
  - Visible note, if related industry:
  - Mapped content pillar:
  - Industry scope: primary_industry | related_industry
  - Related industry, if any:
  - Bridge back to primary offer, if related:
  - Why this still fits, if related:
  - Evidence:
  - Reference URLs:
  - Inference type: direct | adjacent
  - Logic chain, if adjacent:
  - Audience pain point:
  - Why it matters:

#### Local

- Idea (repeat this block for every credible idea in this bucket):
  - Visible note, if related industry:
  - Mapped content pillar:
  - Industry scope: primary_industry | related_industry
  - Related industry, if any:
  - Bridge back to primary offer, if related:
  - Why this still fits, if related:
  - Evidence:
  - Reference URLs:
  - Inference type: direct | adjacent
  - Logic chain, if adjacent:
  - Audience pain point:
  - Why it matters:

### 2. Evergreen / Foundation

#### Global

- Idea (repeat this block for every credible idea in this bucket):
  - Visible note, if related industry:
  - Mapped content pillar:
  - Industry scope: primary_industry | related_industry
  - Related industry, if any:
  - Bridge back to primary offer, if related:
  - Why this still fits, if related:
  - Evidence:
  - Reference URLs:
  - Inference type: direct | adjacent
  - Logic chain, if adjacent:
  - Audience pain point:
  - Why it matters:

#### Local

- Idea (repeat this block for every credible idea in this bucket):
  - Visible note, if related industry:
  - Mapped content pillar:
  - Industry scope: primary_industry | related_industry
  - Related industry, if any:
  - Bridge back to primary offer, if related:
  - Why this still fits, if related:
  - Evidence:
  - Reference URLs:
  - Inference type: direct | adjacent
  - Logic chain, if adjacent:
  - Audience pain point:
  - Why it matters:

### 3. Lead-Gen / Conversion

#### Global

- Idea (repeat this block for every credible idea in this bucket):
  - Visible note, if related industry:
  - Mapped content pillar:
  - Industry scope: primary_industry | related_industry
  - Related industry, if any:
  - Bridge back to primary offer, if related:
  - Why this still fits, if related:
  - Evidence:
  - Reference URLs:
  - Inference type: direct | adjacent
  - Logic chain, if adjacent:
  - Audience pain point:
  - Why it matters:

#### Local

- Idea (repeat this block for every credible idea in this bucket):
  - Visible note, if related industry:
  - Mapped content pillar:
  - Industry scope: primary_industry | related_industry
  - Related industry, if any:
  - Bridge back to primary offer, if related:
  - Why this still fits, if related:
  - Evidence:
  - Reference URLs:
  - Inference type: direct | adjacent
  - Logic chain, if adjacent:
  - Audience pain point:
  - Why it matters:

## Best Idea Today

- Selected idea:
- Audience pain point:
- Viewer value / lesson:
- Non-promotional angle:
- Why this helps the audience:
- Soft business relevance:
- Promotional risk check: pass | rewritten | promotional_not_value_first
- Visible note, if related industry:
- Mapped content pillar:
- Industry scope: primary_industry | related_industry
- Related industry, if any:
- Bridge back to primary offer, if related:
- Why this still fits, if related:
- Category:
- Scope:
- Why this won:
- Content mix reason:
- Data points supporting it:
- Reference URLs:
- Inference type: direct | adjacent
- Logic chain, if adjacent:
- Confidence:
- Pain points matched:
- Repetition check:
- Lead potential:

## Production-Ready Content Drafts

### Writing Method Used

- Format: video | blog | social
- Source: MCP | public_api | static_zip | local_cache | best_effort_fallback
- Source URL or tool:
- Loaded at:

Keep writing-method/source/tool details in the internal Markdown/source record and `INTERNAL_REPORT`. In client-facing HTML/PDF, show only the polished draft versions, source-backed rationale, and production readiness status.

### Version Label Rule

Every script, blog/article draft, or social caption variant must have a human-readable version label.

Do not show only an internal abbreviation such as `VE`, `QA`, `POV`, `Myth`, or `Checklist`. The human may not know what the abbreviation means.

Required format:

```text
Version {number}: {short_code} — {plain_English_meaning}
```

Default video-script versions:

- `Version 1: VE — Value Explainer`
- `Version 2: QA — Client Q&A`
- `Version 3: POV — POV`
- `Version 4: CS — Case Study`
- `Version 5: MB — Myth-Buster`

In client-facing reports, these five video-script versions are candidate options for choosing a direction. They should not include inline image/video URLs by default and must not be treated as the final provider video payload. After a version/code is selected, the production flow loads the WideCast video script-writing skill again and processes only that selected version through research, factual-core checks, Stage 2 visual treatment, inline media URLs, media pool, and production handoff.

If a non-video format or a human override produces only one draft, still label it as `Version 1`.

## Version 1: VE — Value Explainer

Use the appropriate version label for the actual draft. `VE — Value Explainer` is only an example.

### Video Script

### Title


### Hook


### Script


### Visual Notes


### On-Screen Text


### CTA


### Source-Backed Rationale

Include the reference URLs that support the script's key claims. For private data sources, include the captured private URL and note that the human may need to be logged in to verify it.

## Version 1: Blog — Educational Article

Include this section only when `output_formats` includes `blog_article` or when the human requested a blog/article variant.

Use the appropriate version label for the actual blog/article draft. `Blog — Educational Article` is only an example.

### Working Title


### Search / Reader Intent


### Draft


### CTA


### Source-Backed Rationale

Include the reference URLs that support the article's key claims.


## Human Approval Options

- Edit a version directly in the HTML report, click `Copy this version`, and use the final text for review or production.
- Approve this draft.
- Revise this draft.
- Pick another idea from the list.
- Request a blog/video variant.
- Create the approved production asset.
```

### Mobile HTML Report Rule

In addition to internal Markdown files, the agent must export the final daily results as mobile-friendly client-facing HTML designed to read like a professional agency report. The user/operator can read, edit draft versions, copy final text, and make decisions, but the client-facing HTML must not expose internal system/tooling details.

The HTML report does not replace the Markdown report internally. Treat Markdown as the source-of-truth record for the agent, client-facing HTML as the portable rendered delivery copy, and `INTERNAL_REPORT` as the operator-only system report.

The HTML must be standalone and portable. Do not create an HTML report that requires fetching a neighboring `.md` file through `fetch("./report.md")`, because local file access and mobile sharing can break.

The agent may create a custom HTML report instead of a direct Markdown render when that produces a better user experience. The client-facing HTML must remain factually aligned with the Markdown, but it should be optimized for client-safe reading, mobile review, and editable draft workflow.

The client-facing HTML report is the preferred review/share result because scheduled runs often happen while the human is away from the AI agent UI, and the human may open the result on a phone.

The client-facing HTML report must be written in the same language the human wants to use with the client. Keep internal file names and field keys stable if needed, but section titles, summaries, explanations, recommendations, and action labels should match the report language.

Required internal, client-facing, and operator-facing outputs:

- Per-client report:
  - Internal source: `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outputs/YYYY-MM/YYYY-MM-DD/{client-name}-daily-report.md`
  - Staging public report: `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outputs/YYYY-MM/YYYY-MM-DD/{client-name}-public-data-sources-report.html`
  - Staging private report: `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outputs/YYYY-MM/YYYY-MM-DD/{client-name}-private-data-sources-report.html`
  - Staging daily cover/index: `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outputs/YYYY-MM/YYYY-MM-DD/{client-name}-daily-report.html`
  - Client-facing combined report: `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outputs/YYYY-MM/YYYY-MM-DD/{client-name}-client-report.html`
  - Client-facing latest combined report: `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outputs/latest/{client-name}-client-report.html`
  - Operator-only internal report: `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outputs/YYYY-MM/YYYY-MM-DD/{client-name}-INTERNAL_REPORT.html`
  - Operator-only latest internal report: `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outputs/latest/{client-name}-INTERNAL_REPORT.html`
- Master report:
  - Internal source: `daily-content-pipeline/outputs/YYYY-MM/YYYY-MM-DD_master_digest.md`
  - Operator-facing report: `daily-content-pipeline/outputs/YYYY-MM/YYYY-MM-DD_master_digest.html`
  - Internal latest pointer: `daily-content-pipeline/outputs/latest_master_digest.md`
  - Operator-facing latest report: `daily-content-pipeline/outputs/latest_master_digest.html`

The human/operator should normally be shown only the client-ready combined report path, PDF companion path/status, and operator-only `INTERNAL_REPORT` path. Public/private lane staging paths should not be shown unless requested for diagnostics. Internal `.md` paths are for the agent only.

Do not present this older ambiguous shape to the human:

```text
Open either latest.md or latest.html
```

Only present:

```text
Open {client-name}-client-report.html
```

The HTML report must be mobile-first:

- Responsive layout.
- Single-column by default.
- Large readable text.
- Clear cards or sections.
- Sticky or top summary when useful.
- Tap-friendly links.
- Do not render fake interactive buttons in static HTML reports. If the report is a static file, approval options and next actions must be plain text instructions or links only.
- Exception: script/blog/social draft review blocks may include a real local `Copy` button that copies the edited draft text to the clipboard. This button must not imply approval, publishing, rendering, messaging, or any server-side action.
- Exception: `Lead & Competitor Opportunities` cards may include a real local `Copy comment` button that copies only the suggested comment text. This button must not imply auto-commenting, messaging, outreach, approval, publishing, rendering, or any server-side action.
- Short section summaries before long details.
- Collapsible sections when the report is long, if the environment can generate them.
- Clear color/status labels for hot leads, warm leads, high-threat competitors, approval-needed items, and session-expired blockers.
- No document-level horizontal overflow at common mobile widths such as 390px. Tables, evidence ledgers, and scorecards must either become stacked cards or be placed inside a clearly styled horizontal-scroll wrapper. Long URLs, group names, post titles, and source names must wrap inside their cells/cards.

The daily staging HTML cover/index must include:

- Run date.
- Clients processed.
- Client status.
- References to the public data sources and private data sources sections. When packaged into `{client-name}-client-report.html`, these must be internal anchors or plain labels, not links to sibling HTML files.
- Public report status and private report status.
- Top public recommendation summary and top private recommendation summary when available.
- Client-relevant blockers/limits, delivery status, and next action.

Each full lane HTML report must include:

- Its own source coverage, evidence, Lead & Competitor Opportunities, idea matrix, best idea, and draft/recommendation.
- Safe private data source coverage status when relevant, without Local Collector, extension, login/session, or internal source inventory details.
- Private Data Source Discovery status when asked, approved, pending, blocked, or completed, stated as client-safe coverage information.
- Private Data Source Discovery Recommended when no private data sources are configured and discovery has not been offered yet, stated without internal setup mechanics.
- Private Data Source Discovery Declined/Postponed when the human declined or postponed discovery, including a clear client-safe note that public-only reports can still be useful but may miss community, lead, and competitor signals.
- Top ideas.
- Best idea.
- Mapped content pillar.
- Reference URLs.
- Lead & Competitor Opportunities.
- Lead profile URLs and post/current URLs when available.
- Competitor profile URLs and post/current URLs when available.
- Suggested value-first comments with real local copy buttons for every displayed lead/competitor opportunity.
- Data source issues.
- Private source access limitations stated without naming login/session mechanics.
- Production-ready draft: video script, blog/article, social caption, or configured combination.
- Production readiness status in client-safe language: draft ready, approval required, ready for production, published, or blocked by missing client detail.
- Approval options.
- Next actions.

### Agency-Grade HTML Report Standard

The HTML report is not a data dump. It is a professional agency decision report. A busy client or agency owner should be able to open it on a phone, understand the opportunity, verify the evidence, approve a draft, and know the next action within 60 seconds.

The report must feel like it came from a capable media strategist, not from a crawler. It should combine research, judgment, prioritization, production readiness, and clear client communication.

Required report hierarchy:

1. `Executive Snapshot`
   - Client name.
   - Run date.
   - Source coverage status: public data sources only, public data sources + private data sources, private data sources pending, private data sources failed, or mixed.
   - Best idea of the day in one sentence.
   - Why it matters today.
   - Content asset status: draft ready, approval required, ready for production, published, needs human detail, needs visual assets, or blocked.
   - Lead count: hot, warm, and not scanned/pending if applicable.
   - Competitor signal count.
   - One recommended next action.

2. `Public Data Source Intelligence`
   - Public source coverage and data quality.
   - Public evidence ledger.
   - Public Lead & Competitor Opportunities.
   - Public Idea Matrix.
   - Best public idea.
   - Public draft/recommendation.
   - Public blockers or limitations.

3. `Private Data Source Intelligence`
   - Private source coverage and status, stated without internal collector/extension/login details.
   - Private source coverage and data quality.
   - Private evidence ledger.
   - Private Lead & Competitor Opportunities.
   - Private Idea Matrix.
   - Best private idea.
   - Private draft/recommendation.
   - Private blockers, skipped sources, or pending coverage notes stated in client-safe language.

4. `Today's Recommendation`
   - The single best idea.
   - Target audience segment.
   - Pain point or desire it hits.
   - Content pillar.
   - Primary industry or related industry label.
   - If related industry, explain the business logic bridge back to the client's offer.
   - Why this won over the other ideas.
   - Confidence level: high, medium, or low.
   - Main reference URLs.

5. `Evidence Ledger`
   - Every important factual claim, number, date, law/regulation point, price, platform policy, market signal, or news claim used in the report must appear in a claim-level evidence table.
   - Required columns: `Claim`, `Source URL`, `Source type`, `Captured at`, `Confidence`, `Used in`.
   - If a claim is inferred rather than directly stated by a source, label it `inference` and explain the logic.
   - Do not use unsupported numeric claims in the final recommendation or script.
   - If a useful claim cannot be verified, either remove it or mark it as low confidence and keep it out of the main hook.
   - Keep source type visible as `public` or `private`; do not merge source evidence so the reader cannot tell where a claim came from.

6. `Source Coverage And Data Quality`
   - Public search keywords used today.
   - Public keyword count and diversity: show whether at least 10 distinct public search keywords were used, or name the blocker if not.
   - Public candidate idea sufficiency: show whether at least 3 source-backed ideas were new or newly angled after history review, or name the blocker if not.
   - Pain-point/problem/need keyword sample used or added today, with the rest saved in the keyword bank for rotation. Do not dump the full keyword bank into the client-facing report.
   - Public data sources scanned.
   - New public data sources discovered/promoted/demoted today, with a compact summary. Do not dump the full public data source list.
   - Private data source coverage summarized safely: scanned, pending, skipped, failed, or unavailable. Do not mention login/session mechanics, Local Collector, Chrome extension, or private source inventory.
   - New private data source signal categories detected, summarized safely.
   - Known blind spots for this run.
   - Data confidence summary.
   - If private data sources were provided but not activated yet, state that clearly and do not imply lead coverage is complete.

7. `Private Data Source Discovery`
   - In client-facing reports, include only a safe summary such as `additional community/source discovery is pending`, `new community signal categories were found`, or `private source coverage was unavailable today`.
   - Do not include discovery URLs, exact private source inventory, Facebook search URLs, scroll counts, Local Collector state, extension state, login/session state, or source-approval mechanics.
   - Put discovery categories, URLs, keywords, candidate groups/profiles/pages/KOLs/channels/communities, skipped/noisy examples, feed signals, source approval needs, and Local Collector status in `INTERNAL_REPORT`.

8. `Idea Portfolio`
   - Keep the three-section idea structure:
     - Hot / Trend / News
     - Evergreen / Foundation
     - Lead-Gen / Conversion
   - Include both global and local angles when they matter.
   - Treat each global/local area as a bucket that can contain many ideas. Include every credible, source-backed idea harvested today in the appropriate bucket; do not reduce the matrix to one idea per bucket.
   - Every idea must be audience-value-first, not client-praise-first. The idea must be useful even if the viewer never buys from the client.
   - For every idea, show:
     - Title.
     - Global or local label.
     - Primary industry or related industry label.
     - Audience pain point.
     - Viewer value / lesson.
     - Source signal.
     - Non-promotional angle.
     - Why this helps the audience.
     - Soft business relevance.
     - Content pillar.
     - Novelty status: `new`, `new_angle`, `near_duplicate_rejected`, or `repeat_rejected`.
     - Value-first status: `pass`, `rewritten`, or `promotional_not_value_first`.
     - Prior related idea/date when the idea reuses a topic from history.
     - New angle explanation when applicable.
     - Reference URL(s).
     - Short rationale.
   - Do not use the client's product/service name as the main value of the idea. The product/service may appear only in `soft business relevance`, a case-study note, or a gentle CTA after the educational value is clear.
   - Reject or rewrite ideas that merely praise, position, or advertise the client/product/service. Use `promotional_not_value_first` for ideas that cannot be made educational.
   - Bad idea: `{Client Product} out-positions competitors with research + multi-platform execution.`
   - Better idea: `Why small brands should test one message across three channels before spending heavily on content production.`
   - Better soft business relevance: `Fits {Client Product} because the offer supports research and multi-platform execution, but the content teaches the audience how to make a smarter decision first.`
   - Score or qualitative rating for heat, relevance, lead potential, novelty, and confidence.
   - Empty weak buckets are allowed. Do not fill a bucket with filler just to make the report look complete, and do not drop credible harvested ideas merely because the bucket already has another idea.

9. `Decision Scorecard`
   - Compare the top candidate ideas before choosing the best one.
   - Score at least: trend heat, audience pain intensity, business relevance, lead potential, novelty/history risk, evidence strength, and production effort.
   - Include the Idea Novelty Check result for the winner and the strongest rejected candidate: whether the idea is new, newly angled, or rejected as too close to prior history.
   - Briefly explain why the selected idea won and why the other strong candidates did not win today.

10. `Public Lead & Competitor Opportunities` and `Private Lead & Competitor Opportunities`
   - Use these lane-specific titles for English reports, or natural same-language titles for the human/report language.
   - Load Stage 10 before generating this section.
   - A report-level `Lead & Competitor Opportunities` rollup is allowed, but it must not replace the lane-specific sections and must keep source type visible.
   - Separate hot leads, warm leads, watch leads, direct competitors, indirect competitors, adjacent solutions, attention competitors, and authority/KOL competitors when relevant.
   - Every displayed opportunity must include post/current URL when available. Include profile URL only when visible and safe.
   - Include source, captured_at, context, need or audience-overlap signal, why it matters, confidence, suggested human action, and safety note.
   - Every displayed opportunity must include one copy-ready suggested comment based on the specific post context.
   - The suggested comment must use the same language as the post, provide value, avoid direct advertising, avoid `DM me`, `message me`, `inbox me`, `book a call`, `reach out to start`, or similar sales CTAs, avoid attacking competitors, and sound natural rather than AI-polished.
   - The suggested comment may include one or two tiny natural imperfections or typos when appropriate, but must remain clear and trustworthy.
   - Each suggested comment must have a real local `Copy comment` button that copies the comment text only. It must not imply the comment will be posted automatically.
   - If no leads/competitors were found, say whether that means `none found after scanning`, `coverage from public data sources only`, `additional private/community coverage pending`, or `source unavailable`. Keep Local Collector/login/session mechanics in `INTERNAL_REPORT`.
   - If competitor data is inferred without a captured URL, label it as market hypothesis, not detected competitor evidence.

11. `Production-Ready Drafts`
   - For video, this section is a selection surface: the five versions are candidate script options for human/automation choice, not final provider-ready payloads.
   - Use complete version names:
     - `Version 1: VE — Value Explainer`
     - `Version 2: QA — Client Q&A`
     - `Version 3: POV — POV`
     - `Version 4: CS — Case Study`
     - `Version 5: MB — Myth-Buster`
   - Each version should be a usable draft, not only a one-line angle, unless the report explicitly labels it as an angle preview.
   - For each version, include hook/opening, body, CTA, tone, estimated length, source references, and production notes.
   - Do not spend report time sourcing or vetting inline image/video URLs for all five video options. Label video options as `script option, visual treatment pending` unless a selected-version WideCast skill pass has already produced the final inline media treatment.
   - If visual/media URLs are required for immediate video creation but are missing, label the draft as `script-ready, media-pending`.
   - If a selected version has already passed the WideCast video script-writing skill's Stage 2 visual treatment, label that one selected version `final script ready for provider`, not all five options.
   - If an approved video/blog/social asset has already been created, include the produced asset URL/status and label it `asset-created`, `ready-to-publish`, or `published`.
   - If human approval or missing client detail is still needed before creating the asset, say that clearly in client-safe language. Put provider/setup blockers in `INTERNAL_REPORT`.

12. `Compliance And Brand Safety`
   - Include short risk notes for legal, financial, insurance, medical, regulated, or sensitive industries.
   - Avoid guarantees, misleading claims, unsafe outreach, or pretending the client has performed services they have not performed.
   - If a CTA needs license number, phone number, disclaimer, location, or offer approval, list it under `Needs human detail`.

11. `Next Action`
   - End with exactly one primary next action.
   - Secondary actions may be listed below, but they must not compete with the primary next action.
   - Before the first agency run, if schedule/routine is not configured yet, the primary next action should be schedule/routine setup.
   - After schedule/routine is configured but the first agency run has not happened, the primary next action should be handoff to the exact client-specific automation task. In Setup Flow, do not ask whether to run the first agency run now.
   - After the first Automation Flow report/draft exists and PDNA setup - Production, Distribution, Notification, and Analytics - has not been completed/declined/blocked, the primary next action should usually be that setup gate.
   - For client-facing reports where private data source coverage is pending, the primary next action should usually be reviewing/approving additional source coverage or continuing with public-data-source-only insights. Put Local Collector activation mechanics in `INTERNAL_REPORT`.
   - Do not ask "make a video now?" as the primary next action immediately after the first Automation Flow report/draft; production/provider setup comes first.

### Report Handoff Chat Rule

The HTML report's `Next Action` section is not enough by itself.

When the agent announces a report in chat, Telegram, email, or another human-facing channel, and any required workflow step remains unfinished, the handoff message must include:

1. A short useful summary.
2. The HTML report path/link as the primary review link, plus the PDF companion path/status.
3. A visible progress block for the active workflow.
4. The one required next decision.
5. A final line that is exactly one concrete next-step question.

Report-ready notification validity rule:

- A report-ready notification without an HTML report URL/path and PDF companion status is invalid.
- Before deciding provider upload or notification is unavailable, the agent must run a Provider Report Delivery Capability Check using Client tools first and record the details in `INTERNAL_REPORT`: the current client's provider config, OpenAPI discovery, verified identity, and capability cache, with legacy/global MCP/native tool discovery only as fallback after identity match.
- If WideCast OpenAPI notification/Telegram/email fallback is available, the agent must try to deliver the report to the user/operator through WideCast notification.
- If WideCast OpenAPI exposes an HTML-capable report/file/asset upload operation, upload the client-facing `.html` report for operator delivery and send the uploaded URL to the user/operator. Treat provider-hosted URLs as operator handoff links, not client-share links, because the URL/domain may reveal the provider.
- If the verified client provider exposes PDF upload, upload the PDF companion too; otherwise include the local PDF path/status.
- If WideCast report upload is unavailable or fails, the agent must log the exact provider blocker in `INTERNAL_REPORT` and still include the best available local/hosted `.html` report path/link plus PDF companion path/status in the operator notification.
- If the current AI connector/tool surface does not expose WideCast upload or Telegram tools, check Client tools first before concluding capability is missing. Do not claim that WideCast itself lacks the API or capability unless verified from the current client's provider config, account/API, and OpenAPI status.
- If the agent accidentally sends a notification without a report URL/path or PDF companion status, it must immediately send a correction notification containing the HTML report URL/path plus PDF status and log the correction.

Do not end a report handoff with:

- only the report link;
- only a summary;
- "let me know";
- "see next steps in the report";
- multiple competing questions.

Examples of correct final questions:

```text
You provided private data sources, but the Local Collector is not active yet. Do you want me to guide you through Local Collector setup now so this client-specific automation task can include private data sources later, or keep private data sources pending so the task runs public data sources only until activation is complete?
```

```text
PDNA provider is verified and Version 1 is approved. Do you want me to create the video from Version 1 through the connected provider now?
```

```text
Do you want daily, multiple-times-daily, weekly, or manual-only runs?
```

If the report is the final output of a fully completed requested workflow and no human decision remains, the agent may end with a completion statement instead of a question. Otherwise, progress plus one final question is mandatory.

Professional presentation rules:

- Keep the top of the report concise. Details belong below the executive snapshot.
- Use plain business language in the human's language.
- Avoid unexplained abbreviations.
- Avoid emoji-heavy, gimmicky, or dashboard-toy styling. A small number of status symbols is acceptable, but the report should feel client-ready.
- Use tables only when they make comparison or verification easier.
- When using tables, make the mobile behavior explicit in the HTML/CSS: wrap wide tables in a scroll container or switch to cards. Never let a table widen the whole page on mobile.
- Put reference links beside the claim, idea, lead, competitor, or draft they support. Do not hide all references in one generic source list.
- Label missing data honestly: `not scanned`, `pending activation`, `session expired`, `not detected`, or `low confidence`.
- Do not pretend research from public data sources only has private lead coverage.

Recommended HTML section order:

```text
1. Executive Snapshot
2. Public Data Source Intelligence
3. Private Data Source Intelligence
4. Today's Recommendation
5. Evidence Ledger
6. Source Coverage And Data Quality
7. Private Data Source Discovery
8. Idea Portfolio
9. Decision Scorecard
10. Public Lead & Competitor Opportunities / Private Lead & Competitor Opportunities
11. Production-Ready Drafts
12. Compliance And Brand Safety
13. Next Action
14. Appendix / Raw References, optional
```

Static HTML reports are not application UIs. The agent must not create buttons that imply an action will happen when the human taps them unless the button is backed by a real working URL or local browser action. For approval, revision, choosing another idea, production, publishing, or outreach, the client-facing report should say what decision to make or what wording to approve without naming Solo Agency, WideCast, providers, or tools. Operator-only instructions about where to open WideCast or another provider belong in `INTERNAL_REPORT`. For lead/competitor comments, a local `Copy comment` button is allowed only if it copies the suggested comment text and does not imply auto-posting.

### Editable Draft Review Blocks In HTML

When the client-facing report contains script, blog/article, or social-caption drafts, the HTML report should present each draft version in an editable block so the reviewer can quickly revise the wording inside the browser and copy the final text for review or production.

This is a local review convenience, not a publishing or approval system.

The editable blocks should be generated by the HTML renderer from Markdown version headings such as `## Version 1: VE — Value Explainer`. The AI must not hand-author a second long HTML version of the same script or blog draft.

Rules:

- The client-facing HTML report may show a neutral note near the draft section: `You can fine-tune the draft directly on this page. When you are happy with it, click Copy this version for review or production.` Translate this note into the report language. Do not mention AI chat, agents, Solo Agency, WideCast, providers, or internal workflow mechanics.
- Each draft version must be displayed in its own visually separated section.
- Each section heading must use the full human-readable version label, such as `Version 1: VE — Value Explainer` or `Version 2: QA — Client Q&A`.
- The editable draft body should use a real editable element, such as `<div contenteditable="true">`.
- Each version may include one local `Copy this version` button.
- The copy button must copy the current edited text from that version's editable block, not the original unedited text.
- The HTML must clearly tell the reviewer: `Edit this draft here if you want, then click Copy when the final version is ready.`
- The HTML must not autosave edits, upload edits, publish edits, approve edits, render videos, or spend credits.
- The Markdown report remains the canonical internal record until the human pastes an edited final version back into chat or explicitly asks the agent to save it.
- All draft text inserted into HTML must be escaped safely before rendering. Do not inject untrusted source text as raw HTML.
- The editable blocks and copy buttons must work without external JavaScript, external CSS, remote assets, login, or internet access.
- If clipboard access is blocked by the browser, the page should show a simple fallback instruction: select the edited text manually and copy it.

Suggested minimal structure:

```html
<section class="draft-version">
  <h2>Version 1: VE — Value Explainer</h2>
  <p class="hint">You can fine-tune this draft directly on this page. When you are happy with it, click Copy this version for review or production.</p>
  <div id="draft-v1" class="editable-draft" contenteditable="true" spellcheck="true">
    <!-- Escaped draft text goes here. -->
  </div>
  <button type="button" data-copy-target="draft-v1">Copy this version</button>
</section>
```

Suggested local JavaScript behavior:

```html
<script>
document.addEventListener("click", async function (event) {
  var button = event.target.closest("[data-copy-target]");
  if (!button) return;
  var target = document.getElementById(button.getAttribute("data-copy-target"));
  var text = target ? target.innerText.trim() : "";
  try {
    await navigator.clipboard.writeText(text);
    button.textContent = "Copied";
    setTimeout(function () { button.textContent = "Copy this version"; }, 1500);
  } catch (error) {
    button.textContent = "Select text manually";
  }
});
</script>
```

When the result is long, the agent should send or surface the HTML report plus PDF companion path/status instead of dumping the whole result into chat.

The agent must deliver the client-facing HTML report, PDF companion path/status, and operator-only `INTERNAL_REPORT` path/status to the user/operator by the most convenient available channel:

- Configured provider notification, preferably WideCast OpenAPI `sendTelegramMessage`, with the uploaded operator-delivery HTML report URL when report upload is available, the PDF companion URL/path/status, and the `INTERNAL_REPORT` path/status. WideCast's notification API may automatically fall back to email when the human has not connected Telegram yet.
- If the configured provider notification itself is unavailable, use a connected Gmail/email MCP, connector, or tool to email the HTML/PDF reports or links to the human if available and authorized.
- Agent chat file attachment if supported.
- Local file path in the automation/thread output.
- Slack, Discord, Google Drive, Notion, or other connector if available and authorized.
- If none of those are available, provide the local path and clearly say where to open it.

Notification fallback rule:

- WideCast OpenAPI notification/Telegram/email fallback is the preferred scheduled-run notification channel when the client has configured WideCast as the provider.
- If WideCast Telegram is not connected yet, `INTERNAL_REPORT` and the operator handoff may include a concise setup note encouraging the human to register at WideCast, get an API key through `Setup AI Agent` -> `API Keys & MCP` -> `Setup` -> `Generate API key and MCP url`, connect Telegram for daily report alerts, and use it to receive report links/blockers remotely without sitting in front of the machine. Mention that connecting social accounts is optional and enables publishing only after human approval. Do not put this note in client-facing reports or PDFs.
  Translate this note into the human/operator language.
  Suggested internal/operator copy:
  ```text
  Get daily reports on Telegram
  Register at https://widecast.ai/#setup, log in, click Setup AI Agent, open API Keys & MCP, click Setup, then Generate API key and MCP url. Paste only the API key back to the agent for this client. Connect Telegram there so scheduled runs can send report links, blockers, and approval requests to your phone. If convenient, connect social accounts too; publishing to 10+ platforms still happens only after you approve the exact content and target platforms.
  ```
- If WideCast OpenAPI notification is available, call `sendTelegramMessage` even if the human has not connected Telegram yet. WideCast should handle fallback email delivery when Telegram is not connected and email fallback is available.
- If WideCast OpenAPI notification is available and Client tools expose an HTML-capable upload API, upload the `.html` report to WideCast first with `uploadAsset` and send the uploaded URL through WideCast Telegram/email fallback. Upload the PDF companion too only when Client tools expose a compatible PDF upload operation; otherwise include the local PDF path/status.
- Do not send only a local file path when an uploaded WideCast report URL is available.
- If provider config is missing, auth fails, OpenAPI discovery fails, account verification mismatches, or the current WideCast OpenAPI spec cannot upload `.html` files, log the exact provider-neutral blocker and any useful legacy WideCast alias, send the best available HTML path/link plus PDF companion path/status, and state the upload blocker clearly.
- If OpenAPI discovery does not expose a WideCast Telegram/notification send operation, log `provider_required_operation_missing` and any useful legacy WideCast alias. Do not claim WideCast itself lacks notification capability merely because a legacy/global MCP/native tool surface is not exposed.
- Never send a report-ready notification that contains only a status summary. The notification must include an `HTML report` URL/path field and a `PDF companion` URL/path/status field.
- The agent should not switch to Gmail/email merely because Telegram is not connected in WideCast if WideCast email fallback can deliver.
- Use Gmail/email only when provider notification is unavailable or blocked.
- If Gmail/email is available and provider notification is unavailable, send the HTML/PDF reports or a link/path to the HTML report plus PDF companion path/status by email to the human, using the same language the human uses.
- If neither WideCast notification nor Gmail/email is available, the agent should suggest connecting WideCast notification first, or Gmail/email as a secondary fallback.
- The agent must not ask for email passwords, OAuth credentials, app passwords, cookies, or raw tokens. Use only an already connected Gmail/email tool, or guide the human to connect the official connector.
- The email subject should include the client or master digest name, run date, and status, for example: `Daily Content Report Ready — Smith Law — 2026-06-21`.
- The email body should include run status, client-facing HTML report path/link, PDF companion path/status, `INTERNAL_REPORT` path/status, blockers, lead/competitor counts, and next action.
- If the email tool supports attachments, attach the `.html` report and `.pdf` companion when available. If not, include the HTML report path/link plus PDF companion path/status.

If the channel cannot send files directly, send a short notification containing:

- Agent identity.
- Run status.
- HTML report path or URL.
- INTERNAL_REPORT path or status.
- Number of clients processed.
- Number of hot leads, warm leads, and competitors detected.
- Required human actions.

## Provider Report Delivery Capability Check

Run this check before claiming any daily run, scheduled run, or report handoff is complete.

This is an operator/internal check. Its detailed output must be recorded in `INTERNAL_REPORT` and `notification_log.md`, not in client-facing reports or PDFs.

1. Confirm the local client-facing `.html` report exists.
2. Confirm the mandatory PDF companion exists or that `client_pdf_status` and `client_pdf_blocker` record the exact blocker.
3. Confirm `{client-name}-INTERNAL_REPORT.html` exists or that `internal_report_status` records the exact blocker.
4. Confirm `client_facing_scrub_status: pass` before calling any report client-ready.
5. Check the configured notification channel from `daily-content-pipeline/schedule.md` and the Client Intelligence Profile.
6. Load `daily-content-pipeline/provider_defaults.json` and the target client's `integrations/providers/provider_config.local.json` when present.
7. If WideCast is configured, preferred, connected, or likely available, fetch/cache `https://widecast.ai/openapi.yaml` unless the cache is current.
8. Verify the provider account before using account actions. For WideCast, call `getAccount` with the current client's configured provider credential and compare the verified account identity to the saved client provider identity when present.
9. Inspect the discovered OpenAPI operation list for:
   - account/status capability, such as `getAccount`;
   - HTML-capable report/file/asset upload capability, such as `uploadAsset` with `text/html`;
   - PDF/file upload capability when exposed by the verified client provider;
   - Telegram/report notification send capability, such as `sendTelegramMessage`;
   - email fallback behavior exposed by the provider, if any.
10. Use legacy/global MCP/native tool discovery/lazy-load only as a fallback or compatibility path after Client tools and the client-scoped provider config/account identity have been checked. A global MCP/native WideCast account visible in the AI session is not proof that the current client's report upload, notification, platforms, credits, or analytics are configured. If the tool account cannot be proven to match the client provider identity, log `global_mcp_not_client_scoped` and continue with the per-client OpenAPI/API-key setup path or the best authorized fallback.
11. If upload capability exists, upload the client-facing `.html` report for user/operator delivery and capture the uploaded URL and TTL if returned. Treat provider-hosted URLs as operator handoff links, not client-share links.
12. If PDF upload capability exists, upload the PDF companion and capture the uploaded URL and TTL if returned.
13. If notification capability exists, send the uploaded URL when available; otherwise send the best available local/hosted `.html` path/link with the exact upload blocker. Always include the PDF companion URL/path/status and the internal report path/status.
14. If the provider notification operation itself is unavailable, use an authorized fallback channel such as Gmail/email only when available and authorized; otherwise surface the local HTML path plus PDF companion path/status plus internal report path/status in chat and log the notification blocker.
15. Save a report-delivery record in `daily-content-pipeline/notifications/notification_log.md`.

The report-delivery record must include:

```yaml
html_report_path:
internal_report_path:
internal_report_status:
client_facing_scrub_status:
client_facing_scrub_blocker:
pdf_report_path:
pdf_status:
pdf_blocker:
provider:
provider_discovery_url:
provider_openapi_checked: true | false
provider_account_verified: true | false | unknown
provider_account_identity:
provider_identity_source: per_client_openapi | global_mcp_compat | unknown
mcp_compatibility_status: not_used | identity_matched | identity_mismatch | not_client_scoped
upload_operation_id:
notification_operation_id:
provider_upload_available: true | false | unknown
provider_notification_available: true | false | unknown
provider_notification_destination_status: connected | fallback_email | not_configured | unknown
provider_upload_attempted: true | false
provider_uploaded_report_url:
provider_uploaded_pdf_url:
provider_uploaded_report_url_ttl:
provider_notification_attempted: true | false
provider_notification_status: sent | failed | unavailable | skipped
fallback_notification_channel:
final_report_link_sent_to_human:
final_pdf_link_or_status_sent_to_human:
final_internal_report_link_or_status_sent_to_human:
blocker:
```

Completion is invalid if the agent says only `report ready` or `config updated` without this delivery outcome when a scheduled run or daily report was requested.

Correct blocker wording:

```text
Report delivery: HTML report generated and PDF companion status recorded. Client tools/provider capability check completed. WideCast OpenAPI discovery or account verification is blocked by `{exact blocker}`, so I logged `{provider blocker}` and am giving you the local HTML report path plus PDF companion path/status here.
```

Incorrect blocker wording:

```text
WideCast cannot upload reports.
```

That is too broad unless the agent verified WideCast account/API status and the current OpenAPI spec directly.

---
