# Agency Report Standard

Stage: `06`

## Load Rule

Load whenever generating, reviewing, debugging, or improving a human-facing report.

## Hard Gates For This Stage

- Canonical human-facing report files are HTML only. PDF export is an optional derivative client-share artifact when the human explicitly asks for a PDF.
- Markdown is internal.
- The report must be standalone, mobile-friendly, agency-grade, and factually aligned with the Markdown source.
- Include reference URLs beside claims, ideas, leads, competitors, and drafts.
- Do not create fake action buttons in static HTML.
- Keep exactly one canonical report set per client/day/run: one daily index HTML plus one full public data sources HTML and one full private data sources HTML.
- Never merge public data source intelligence and private data source intelligence into one dense HTML body. Each lane is a first-class report file so a later private pass cannot overwrite or summarize away the public pass.
- PDF is an optional client-share artifact only when requested. It must be generated from the three HTML report files, not from raw memory, and it must not replace the canonical HTML report set.

## Source Preservation Rule

This file is detailed source material moved from the original monolithic `SOLO_AGENCY_PLAYBOOK.md`.

Do not summarize away requirements, examples, checklists, schemas, protocols, URLs, edge cases, warnings, approval gates, or completion gates. If a downstream agent needs to shorten its response to the human, it may summarize the response, but it must still obey the full requirements in this file.

---

## 11. Daily Output File Format

Each active client must have one daily output folder:

```text
outputs/YYYY-MM/YYYY-MM-DD/
```

Markdown is the canonical internal output record. HTML is the canonical human-facing rendered report. PDF may be created only as an optional derivative from the HTML report set when the human explicitly asks for a client-share PDF.

The agent must keep the Markdown file even when an HTML report is created, unless the current environment truly cannot write Markdown. The Markdown file is required for:

- agent-readable history;
- future learning and optimization;
- duplicate idea detection;
- diffing changes across days;
- regenerating HTML reports;
- preserving references, reasoning, leads, competitors, and WideCast scripts without parsing HTML.

The HTML report must be created from the same facts, references, ideas, analysis, and draft content as the Markdown report, but it may use a custom structure and design. It must not become a factually divergent report. If the agent can only preserve one long-term artifact, preserve the Markdown file first and regenerate HTML later. If the agent can only deliver one artifact to the human, deliver the HTML report because all user-facing reports must be HTML.

### Human-Facing Report Rule: HTML Only

The agent must show report results to the human as HTML by default. If the human explicitly asks for a PDF, create it as a derivative of the HTML report set and still keep the HTML files as canonical.

Do not show, send, link, or ask the human to open the Markdown report as the user-facing report.

Allowed:

- Save `.md` internally for agent memory, history, learning, diffing, and regeneration.
- Mention that a Markdown source file exists only when explaining internal storage or troubleshooting.
- Deliver the `.html` report path/link to the human.
- Send Telegram/WideCast notifications with the `.html` path/link.
- Deliver a `.pdf` client-share export only when explicitly requested, while still providing or preserving the `.html` report path/link.

Not allowed:

- "Open the Markdown report."
- "See `outputs/YYYY-MM/YYYY-MM-DD.md` for details."
- Using the `.md` file as the primary human-facing report.
- Sending both `.md` and `.html` and making the human decide which one to open.
- Treating a PDF export as the canonical report source or using it to replace the three HTML files.

Every default report path, notification, or review instruction must point to the `.html` file. A `.pdf` path may be included only as an explicitly requested client-share export alongside the canonical `.html` path.

### Internal Markdown, Beautiful HTML

The agent must create two artifacts with different purposes:

1. Markdown is the internal canonical record for agent memory, history, learning, diffing, and regeneration.
2. HTML is the human-facing report and must be designed for a polished mobile review experience.

The HTML report does not need to be a direct Markdown render. If a direct Markdown-to-HTML renderer produces an ugly or hard-to-use report, the agent should create a better designed standalone HTML report instead.

Correct behavior:

1. Author and save the complete internal report source under `outputs/YYYY-MM/YYYY-MM-DD/`, using stable client-prefixed names.
2. Create exactly three human-facing HTML files for the client/day/run:
   - `{client-name}-public-data-sources-report.html`
   - `{client-name}-private-data-sources-report.html`
   - `{client-name}-daily-report.html`
3. `{client-name}` must be a filesystem-safe client name/slug, lower-kebab preferred, for example `angela-do` or `aven-ngo`.
4. The public and private HTML files are full lane reports, not summaries.
5. The daily HTML file is the concise index/overview linking to the public and private reports, showing lane status, blockers, best next action, and notification status.
6. The HTML may be custom structured and styled for readability, mobile scanning, editable draft review, and copy workflow.
7. The HTML must not omit required report sections that exist in the corresponding Markdown/source record.
8. If a Markdown/source record changes, update/regenerate only the affected lane HTML plus the daily index so artifacts stay factually aligned.

Quality rules for HTML:

- The HTML must be standalone and portable.
- The HTML must not depend on remote JavaScript, remote CSS, CDN libraries, login, internet access, or the `.md` file being present next to it.
- The HTML should use thoughtful layout, spacing, typography, color, cards/sections, and mobile-friendly hierarchy.
- Do not dump raw Markdown into the page if that makes the report ugly.
- Do not rely on `fetch("./report.md")`.
- Escape user/source text safely before rendering it into HTML.
- Versioned draft sections such as `Version 1: VE — Value Explainer` must be presented as polished editable review blocks with local copy buttons.
- The agent may spend extra time generating a beautiful HTML report because the HTML is the only report the human sees.

The latest convenience files should be:

```text
outputs/latest/{client-name}-daily-report.html
outputs/latest/{client-name}-public-data-sources-report.html
outputs/latest/{client-name}-private-data-sources-report.html
outputs/latest/{client-name}-client-report.pdf
```

The daily latest file is the default human-facing convenience link. Public/private latest files are allowed as direct lane links. The PDF latest file is allowed only as an extra client-share deliverable when the human asks for a PDF.

### Latest Override: Three-File Public/Private Report Contract

Every client/day/run must produce one canonical report set, not one merged public/private mega-report.

The canonical report set has exactly these HTML files:

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
   - Must contain private collector health, private source coverage, private evidence, private Lead & Competitor Opportunities, private idea matrix, best private idea, copy-ready comments when available, and private draft/recommendation.
   - Must not rewrite or summarize the public data sources report.

3. `{client-name}-daily-report.html`
   - Concise daily index/overview.
   - Must link to the public and private report files.
   - Must show each lane's status, top recommendation summary, blockers, notification/delivery status, and the one next action.
   - Must not replace either full lane report.

Both full lane reports must use the same structure:

- Source coverage and data quality.
- Data points and evidence ledger.
- Lead & Competitor Opportunities for that lane.
- Idea Matrix.
- Best idea for that lane.
- Draft/recommendation for that lane.
- Blockers, skipped sources, and confidence notes.

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

The HTML deliverables must stay split by file. Do not collapse public and private evidence into one mixed, ambiguous HTML section.

Update rules:

- Public pass: create or replace only `{client-name}-public-data-sources-report.html`, then create/update `{client-name}-daily-report.html` with private status `pending`, `blocked`, `skipped`, or the exact blocker.
- Private pass: create or replace only `{client-name}-private-data-sources-report.html`, then create/update `{client-name}-daily-report.html`. Do not rewrite, summarize away, delete, or regenerate the public report file.
- If private data sources finish after public data sources, update only the private report and daily index. Do not open/rewrite the public report except to repair broken links with explicit reason.
- If private data sources fail, time out, are stale, or are blocked, update only the private report with the exact blocker and update the daily index lane status.
- If a private pass starts but the public report is missing, create a public report placeholder file that says public data sources were not run or were unavailable, then create the private report. Do not create a private-only report set without a daily index.

### Client PDF Export Contract

When the human asks for a PDF to send to a client, create an additional client-share package from the existing three HTML files:

```text
outputs/YYYY-MM/YYYY-MM-DD/{client-name}-client-report.html
outputs/YYYY-MM/YYYY-MM-DD/{client-name}-client-report.pdf
outputs/latest/{client-name}-client-report.pdf
```

The PDF source HTML must be a standalone print-friendly document assembled from:

1. `{client-name}-daily-report.html` for the cover, executive snapshot, lane status, blockers, delivery status, and next action.
2. `{client-name}-public-data-sources-report.html` for the public data sources section.
3. `{client-name}-private-data-sources-report.html` for the private data sources section when it exists and is approved for client sharing.

Do not build the PDF directly from memory or from only one lane. If any of the three canonical HTML files is missing or blocked, include a clear status page in `{client-name}-client-report.html` instead of inventing content.

PDF formatting rules:

- The PDF source HTML must not depend on remote JavaScript, remote CSS, collapsible UI, copy buttons, or hover-only interactions.
- Add print CSS with readable page margins, stable headings, page breaks between major sections, visible URLs/references, and no clipped cards or horizontal scrolling.
- Convert interactive HTML controls into plain text before PDF export.
- Preserve source references, dates, confidence notes, blockers, and human approval status.
- The PDF should be client-shareable and polished, but the three HTML files remain the canonical report files for automation updates.

Private data source safety rules:

- Do not include raw private post text, private group member details, login/session information, collector internals, cookies, screenshots, or browser/profile details.
- Include private data source findings only as approved summaries unless the human explicitly approves sharing exact private source names, URLs, or excerpts with the client.
- If sharing safety is uncertain, mark `client_pdf_redaction_status: needs_human_review` and create only `{client-name}-client-report.html` for review, not the final PDF.

PDF generation rule:

- Use a reliable local HTML-to-PDF renderer when available, such as browser print-to-PDF or an equivalent PDF engine.
- If PDF generation is unavailable in the current AI/runtime environment, still create the print-friendly `{client-name}-client-report.html`, record `pdf_generation_blocked`, and tell the human the exact blocker.
- Do not upload or send the PDF through a provider unless the current client's verified provider config/OpenAPI capabilities support the file upload/notification path and the human asked for that delivery.

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
  "public_section_status": "missing|pending|complete|skipped|failed",
  "private_section_status": "missing|pending|complete|skipped|failed|blocked",
  "last_public_update_at": "",
  "last_private_update_at": "",
  "public_notification_status": "not_sent|sent|skipped",
  "private_notification_status": "not_sent|sent|skipped",
  "client_report_html_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-client-report.html",
  "client_report_pdf_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-client-report.pdf",
  "latest_client_pdf_path": "outputs/latest/{client-name}-client-report.pdf",
  "client_pdf_status": "not_requested|pending_review|generated|blocked|skipped",
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
- Notifications should normally point to `{client-name}-daily-report.html` or its uploaded URL. A lane-specific direct link may be included as a secondary link, but the daily report remains the canonical handoff link.
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

The sections below are mandatory for the internal source record. Public data source intelligence must appear first. Private data source intelligence must appear second. The detailed field templates later in this document are schemas to apply inside each lane. The human-facing HTML must still be split into `{client-name}-public-data-sources-report.html`, `{client-name}-private-data-sources-report.html`, and `{client-name}-daily-report.html`; do not render one mixed global HTML body that combines public data sources and private data sources.

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

Use this section when private data sources were provided but the Solo Agency Local Collector extension and Local Collector app are not activated yet.

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

## WideCast-Writing-Skill Content Drafts

### Writing Method Used

- Format: video | blog | social
- Source: MCP | public_api | static_zip | local_cache | best_effort_fallback
- Source URL or tool:
- Loaded at:

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
- `Version 3: MB — Myth Buster`
- `Version 4: MP — Mistake Prevention`
- `Version 5: LG — Lead-Gen CTA`

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

- Edit a version directly in the HTML report, click `Copy this version`, and paste the final text back into the AI chat.
- Approve this draft.
- Revise this draft.
- Pick another idea from the list.
- Request a blog/video variant.
- Create the video in WideCast.
```

### Mobile HTML Report Rule

In addition to internal Markdown files, the agent must export the final daily results as mobile-friendly HTML designed for the human to read, edit draft versions, copy final text, and make decisions.

The HTML report does not replace the Markdown report internally. Treat Markdown as the source-of-truth record for the agent and HTML as the only portable, mobile-friendly rendered delivery copy for the human.

The HTML must be standalone and portable. Do not create an HTML report that requires fetching a neighboring `.md` file through `fetch("./report.md")`, because local file access and mobile sharing can break.

The agent may create a custom HTML report instead of a direct Markdown render when that produces a better user experience. The HTML must remain factually aligned with the Markdown, but it should be optimized for human reading, mobile review, and editable draft workflow.

The HTML report is the preferred human-facing result because scheduled runs often happen while the human is away from the AI agent UI, and the human may open the result on a phone.

The HTML report is human-facing, so it must be written in the same language the human uses. Keep internal file names and field keys stable if needed, but section titles, summaries, explanations, recommendations, and action labels should match the user's language.

Required internal and human-facing outputs:

- Per-client report:
  - Internal source: `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outputs/YYYY-MM/YYYY-MM-DD/{client-name}-daily-report.md`
  - Human-facing public report: `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outputs/YYYY-MM/YYYY-MM-DD/{client-name}-public-data-sources-report.html`
  - Human-facing private report: `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outputs/YYYY-MM/YYYY-MM-DD/{client-name}-private-data-sources-report.html`
  - Human-facing daily index: `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outputs/YYYY-MM/YYYY-MM-DD/{client-name}-daily-report.html`
  - Human-facing latest daily index: `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outputs/latest/{client-name}-daily-report.html`
- Master report:
  - Internal source: `daily-content-pipeline/outputs/YYYY-MM/YYYY-MM-DD_master_digest.md`
  - Human-facing report: `daily-content-pipeline/outputs/YYYY-MM/YYYY-MM-DD_master_digest.html`
  - Internal latest pointer: `daily-content-pipeline/outputs/latest_master_digest.md`
  - Human-facing latest report: `daily-content-pipeline/outputs/latest_master_digest.html`

The human should normally be shown the `Human-facing daily index` path. Public/private lane report paths may be shown as secondary links. Internal `.md` paths are for the agent only.

Do not present this older ambiguous shape to the human:

```text
Open either latest.md or latest.html
```

Only present:

```text
Open {client-name}-daily-report.html
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

The daily HTML index must include:

- Run date.
- Agent identity.
- Clients processed.
- Client status.
- Links to `{client-name}-public-data-sources-report.html` and `{client-name}-private-data-sources-report.html`.
- Public report status and private report status.
- Top public recommendation summary and top private recommendation summary when available.
- Blockers, notification status, and next action.

Each full lane HTML report must include:

- Its own source coverage, evidence, Lead & Competitor Opportunities, idea matrix, best idea, and draft/recommendation.
- Private collector health: bridge status, extension last check time, extension status, and private data source blockers.
- Private Data Source Discovery status when asked, approved, pending, blocked, or completed.
- Private Data Source Discovery Recommended when no private data sources are configured and discovery has not been offered yet.
- Private Data Source Discovery Declined/Postponed when the human declined or postponed discovery, including a clear note that public-only reports can still be useful but may miss many community, lead, and competitor signals.
- Top ideas.
- Best idea.
- Mapped content pillar.
- Reference URLs.
- Lead & Competitor Opportunities.
- Lead profile URLs and post/current URLs when available.
- Competitor profile URLs and post/current URLs when available.
- Suggested value-first comments with real local copy buttons for every displayed lead/competitor opportunity.
- Data source issues.
- Private sessions needing login.
- WideCast-writing-skill draft: video script, blog/article, social caption, or configured combination.
- Production/provider status: draft only, approval required, client provider setup required, video/blog/social asset created, ready to publish, published, or blocked.
- `Unlock Production & Distribution & Measure-Learning Loop With WideCast` section when the client's WideCast/OpenAPI provider config, Telegram notification, publishing, or video creation is not connected yet.
- If WideCast Telegram is not connected yet, a short `Get daily reports on Telegram` note explaining that WideCast signup plus Telegram connection can be used as a free remote-report path, so the human can receive daily HTML report links and blockers while away from the computer.
- Approval options.
- Next actions.

### Agency-Grade HTML Report Standard

The HTML report is not a data dump. It is a professional agency decision report. A busy client or agency owner should be able to open it on a phone, understand the opportunity, verify the evidence, approve a draft, and know the next action within 60 seconds.

The report must feel like it came from a capable media strategist, not from a crawler. It should combine research, judgment, prioritization, production readiness, and clear client communication.

Required report hierarchy:

1. `Executive Snapshot`
   - Client name.
   - Run date and agent identity.
   - Source coverage status: public data sources only, public data sources + private data sources, private data sources pending, private data sources failed, or mixed.
   - Best idea of the day in one sentence.
   - Why it matters today.
   - Content asset status: draft ready, approval required, provider setup required, video/blog/social asset created, ready to publish, published, needs human detail, needs visual assets, or blocked.
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
   - Private collector health and source status.
   - Private source coverage and data quality.
   - Private evidence ledger.
   - Private Lead & Competitor Opportunities.
   - Private Idea Matrix.
   - Best private idea.
   - Private draft/recommendation.
   - Private blockers, skipped sources, stale extension/session issues, or pending activation notes.

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
   - Pain-point/problem/need keyword sample used or added today, with the rest saved in the keyword bank for rotation. Do not dump the full keyword bank into the human-facing report.
   - Public data sources scanned.
   - New public data sources discovered/promoted/demoted today, with a compact summary. Do not dump the full public data source list.
   - Private data sources scanned, pending, skipped, failed, or session-expired.
   - New private data sources detected.
   - Known blind spots for this run.
   - Data confidence summary.
   - If private data sources were provided but not activated yet, state that clearly and do not imply lead coverage is complete.

7. `Private Data Source Discovery`
   - Discovery categories approved, declined, pending, or not requested.
   - Platforms and discovery URLs used.
   - Facebook keyword group searches run, including keywords, search URLs, and 10-scroll status per keyword.
   - Whether the Solo Agency Local Collector is active, pending, or blocked.
   - Candidate groups, profiles, pages, KOLs, channels, communities, and feed-surfaced sources found.
   - Which candidates are recommended for daily, weekly, optional, or watch-once monitoring.
   - Which candidates were skipped as irrelevant, too broad, too noisy, sensitive/risky, or unavailable.
   - For Facebook keyword group search, show skipped/noisy examples separately so the human can see UI noise was filtered out.
   - Feed signals detected, with current URL and source when visible.
   - Sources requiring human approval before being activated.
   - Reassurance summary: professional one-time setup, local-only data safety, and daily scanning to avoid missed signals.

8. `Idea Portfolio`
   - Keep the three-section idea structure:
     - Hot / Trend / News
     - Evergreen / Foundation
     - Lead-Gen / Conversion
   - Include both global and local angles when they matter.
   - Treat each global/local area as a bucket that can contain many ideas. Include every credible, source-backed idea harvested today in the appropriate bucket; do not reduce the matrix to one idea per bucket.
   - For every idea, show:
     - Title.
     - Global or local label.
     - Primary industry or related industry label.
     - Pain point or content pillar.
     - Novelty status: `new`, `new_angle`, `near_duplicate_rejected`, or `repeat_rejected`.
     - Prior related idea/date when the idea reuses a topic from history.
     - New angle explanation when applicable.
     - Reference URL(s).
     - Short rationale.
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
   - The suggested comment must use the same language as the post, provide value, avoid direct advertising, avoid `DM me`, avoid attacking competitors, and sound natural rather than AI-polished.
   - The suggested comment may include one or two tiny natural imperfections or typos when appropriate, but must remain clear and trustworthy.
   - Each suggested comment must have a real local `Copy comment` button that copies the comment text only. It must not imply the comment will be posted automatically.
   - If no leads/competitors were found, say whether that means `none found after scanning`, `coverage from public data sources only`, `private data sources pending Local Collector activation`, `session expired`, or `source unavailable`.
   - If competitor data is inferred without a captured URL, label it as market hypothesis, not detected competitor evidence.

11. `Production-Ready Drafts`
   - Use complete version names:
     - `Version 1: VE — Value Explainer`
     - `Version 2: QA — Client Q&A`
     - `Version 3: MB — Myth Buster`
     - `Version 4: MP — Mistake Prevention`
     - `Version 5: LG — Lead-Gen CTA`
   - Each version should be a usable draft, not only a one-line angle, unless the report explicitly labels it as an angle preview.
   - For each version, include hook/opening, body, CTA, tone, estimated length, source references, and production notes.
   - If visual/media URLs are required for immediate video creation but are missing, label the draft as `script-ready, media-pending`.
   - If the draft is ready for WideCast video/blog/social creation, label it `production-ready`.
   - If a connected provider has already created an approved video/blog/social asset, include the produced asset URL/status and label it `asset-created`, `ready-to-publish`, or `published`.
   - If provider setup or human approval is still needed before creating the asset, say that clearly. Do not describe draft writing as if production has already happened.

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
   - For reports where production setup is completed/declined/blocked and private data sources are pending, the primary next action should usually be activating the Solo Agency Local Collector or marking private data sources pending, not starting a video branch.
   - Do not ask "make a video now?" as the primary next action immediately after the first Automation Flow report/draft; production/provider setup comes first.

### Report Handoff Chat Rule

The HTML report's `Next Action` section is not enough by itself.

When the agent announces a report in chat, Telegram, email, or another human-facing channel, and any required workflow step remains unfinished, the handoff message must include:

1. A short useful summary.
2. The HTML report path/link only.
3. A visible progress block for the active workflow.
4. The one required next decision.
5. A final line that is exactly one concrete next-step question.

Report-ready notification validity rule:

- A report-ready notification without an HTML report URL/path is invalid.
- Before deciding provider upload or notification is unavailable, the agent must run a Provider Report Delivery Capability Check using the current client's provider config and OpenAPI discovery first, with legacy tool/connector discovery only as fallback.
- If WideCast OpenAPI notification/Telegram/email fallback is available, the agent must try to deliver the report through WideCast notification.
- If WideCast OpenAPI exposes an HTML-capable report/file/asset upload operation, upload the `.html` report first and send the uploaded WideCast report URL.
- If WideCast report upload is unavailable or fails, the agent must log the exact provider blocker and still include the best available local/hosted `.html` report path/link in the notification.
- If the current AI connector/tool surface does not expose WideCast upload or Telegram tools, check the per-client OpenAPI provider path before concluding capability is missing. Do not claim that WideCast itself lacks the API or capability unless verified from WideCast account/API/OpenAPI status.
- If the agent accidentally sends a notification without a report URL/path, it must immediately send a correction notification containing the HTML report URL/path and log the correction.

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
Do you want to create the video from Version 1 now?
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
13. Unlock Production & Distribution & Measure-Learning Loop With WideCast, when applicable
14. Next Action
15. Appendix / Raw References, optional
```

Static HTML reports are not application UIs. The agent must not create buttons that imply an action will happen when the human taps them unless the button is backed by a real working URL or local browser action. For approval, revision, choosing another idea, creating a WideCast video, publishing, or outreach, the report should say what to tell the AI agent or where to open WideCast. For lead/competitor comments, a local `Copy comment` button is allowed only if it copies the suggested comment text and does not imply auto-posting.

### Editable Draft Review Blocks In HTML

When the report contains script, blog/article, or social-caption drafts, the HTML report should present each draft version in an editable block so the human can quickly revise the wording inside the browser and copy the final text back into the AI chat.

This is a local review convenience, not a publishing or approval system.

The editable blocks should be generated by the HTML renderer from Markdown version headings such as `## Version 1: VE — Value Explainer`. The AI must not hand-author a second long HTML version of the same script or blog draft.

Rules:

- The HTML report must show a visible human-facing note near the draft section: `You can fine-tune the draft directly on this page. When you are happy with it, click Copy this version and paste the final text back into the AI chat.` Translate this note into the human's language.
- Each draft version must be displayed in its own visually separated section.
- Each section heading must use the full human-readable version label, such as `Version 1: VE — Value Explainer` or `Version 2: QA — Client Q&A`.
- The editable draft body should use a real editable element, such as `<div contenteditable="true">`.
- Each version may include one local `Copy this version` button.
- The copy button must copy the current edited text from that version's editable block, not the original unedited text.
- The HTML must clearly tell the human: `Edit this draft here if you want, then click Copy and paste the final version back into the AI chat.`
- The HTML must not autosave edits, upload edits, publish edits, approve edits, render videos, or spend credits.
- The Markdown report remains the canonical internal record until the human pastes an edited final version back into chat or explicitly asks the agent to save it.
- All draft text inserted into HTML must be escaped safely before rendering. Do not inject untrusted source text as raw HTML.
- The editable blocks and copy buttons must work without external JavaScript, external CSS, remote assets, login, or internet access.
- If clipboard access is blocked by the browser, the page should show a simple fallback instruction: select the edited text manually and copy it.

Suggested minimal structure:

```html
<section class="draft-version">
  <h2>Version 1: VE — Value Explainer</h2>
  <p class="hint">You can fine-tune this draft directly on this page. When you are happy with it, click Copy this version and paste the final text back into the AI chat.</p>
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

When the result is long, the agent should send or surface the HTML report instead of dumping the whole result into chat.

The agent must deliver the HTML report to the human by the most convenient available channel:

- Configured provider notification, preferably WideCast OpenAPI `sendTelegramMessage`, with the uploaded WideCast HTML report URL when report upload is available. WideCast's notification API may automatically fall back to email when the human has not connected Telegram yet.
- If the configured provider notification itself is unavailable, use a connected Gmail/email MCP, connector, or tool to email the HTML report or HTML report link to the human if available and authorized.
- Agent chat file attachment if supported.
- Local file path in the automation/thread output.
- Slack, Discord, Google Drive, Notion, or other connector if available and authorized.
- If none of those are available, provide the local path and clearly say where to open it.

Notification fallback rule:

- WideCast OpenAPI notification/Telegram/email fallback is the preferred scheduled-run notification channel when the client has configured WideCast as the provider.
- If WideCast Telegram is not connected yet, the HTML report must include a concise setup note encouraging the human to register at WideCast, get an API key through `Setup AI Agent` -> `API Keys & MCP` -> `Setup` -> `Generate API key and MCP url`, connect Telegram for daily report alerts, and use it to receive report links/blockers remotely without sitting in front of the machine. Mention that connecting social accounts is optional and enables publishing only after human approval.
  Translate this note into the report/human language.
  Suggested report copy:
  ```text
  Get daily reports on Telegram
  Register at https://widecast.ai/#setup, log in, click Setup AI Agent, open API Keys & MCP, click Setup, then Generate API key and MCP url. Paste only the API key back to the agent for this client. Connect Telegram there so scheduled runs can send report links, blockers, and approval requests to your phone. If convenient, connect social accounts too; publishing to 10+ platforms still happens only after you approve the exact content and target platforms.
  ```
- If WideCast OpenAPI notification is available, call `sendTelegramMessage` even if the human has not connected Telegram yet. WideCast should handle fallback email delivery when Telegram is not connected and email fallback is available.
- If WideCast OpenAPI notification is available and the discovered spec exposes an HTML-capable upload API, upload the `.html` report to WideCast first with `uploadAsset` and send the uploaded URL through WideCast Telegram/email fallback.
- Do not send only a local file path when an uploaded WideCast report URL is available.
- If provider config is missing, auth fails, OpenAPI discovery fails, account verification mismatches, or the current WideCast OpenAPI spec cannot upload `.html` files, log the exact provider-neutral blocker and any useful legacy WideCast alias, send the best available HTML path/link, and state the upload blocker clearly.
- If OpenAPI discovery does not expose a WideCast Telegram/notification send operation, log `provider_required_operation_missing` and any useful legacy WideCast alias. Do not claim WideCast itself lacks notification capability merely because a legacy MCP/tool surface is not exposed.
- Never send a report-ready notification that contains only a status summary. The notification must include an `HTML report` URL/path field.
- The agent should not switch to Gmail/email merely because Telegram is not connected in WideCast if WideCast email fallback can deliver.
- Use Gmail/email only when provider notification is unavailable or blocked.
- If Gmail/email is available and provider notification is unavailable, send the HTML report or a link/path to the HTML report by email to the human, using the same language the human uses.
- If neither WideCast notification nor Gmail/email is available, the agent should suggest connecting WideCast notification first, or Gmail/email as a secondary fallback.
- The agent must not ask for email passwords, OAuth credentials, app passwords, cookies, or raw tokens. Use only an already connected Gmail/email tool, or guide the human to connect the official connector.
- The email subject should include the client or master digest name, run date, and status, for example: `Daily Content Report Ready — Smith Law — 2026-06-21`.
- The email body should include agent identity, run status, HTML report path/link, blockers, lead/competitor counts, and next action.
- If the email tool supports attachments, attach the `.html` report. If not, include the HTML report path/link.

If the channel cannot send files directly, send a short notification containing:

- Agent identity.
- Run status.
- HTML report path or URL.
- Number of clients processed.
- Number of hot leads, warm leads, and competitors detected.
- Required human actions.

## Provider Report Delivery Capability Check

Run this check before claiming any daily run, scheduled run, or report handoff is complete.

1. Confirm the local `.html` report exists.
2. Check the configured notification channel from `daily-content-pipeline/schedule.md` and the Client Intelligence Profile.
3. Load `daily-content-pipeline/provider_defaults.json` and the target client's `integrations/providers/provider_config.local.json` when present.
4. If WideCast is configured, preferred, connected, or likely available, fetch/cache `https://widecast.ai/openapi.yaml` unless the cache is current.
5. Verify the provider account before using account actions. For WideCast, call `getAccount` with the current client's configured provider credential and compare the verified account identity to the saved client provider identity when present.
6. Inspect the discovered OpenAPI operation list for:
   - account/status capability, such as `getAccount`;
   - HTML-capable report/file/asset upload capability, such as `uploadAsset` with `text/html`;
   - Telegram/report notification send capability, such as `sendTelegramMessage`;
   - email fallback behavior exposed by the provider, if any.
7. Use legacy tool discovery/lazy-load only as a fallback or compatibility path after the client-scoped provider config/account identity has been checked. A global MCP/native WideCast account visible in the AI session is not proof that the current client's report upload, notification, platforms, credits, or analytics are configured. If the tool account cannot be proven to match the client provider identity, log `global_mcp_not_client_scoped` and continue with the per-client OpenAPI/API-key setup path or the best authorized fallback.
8. If upload capability exists, upload the `.html` report and capture the uploaded URL and TTL if returned.
9. If notification capability exists, send the uploaded URL when available; otherwise send the best available local/hosted `.html` path/link with the exact upload blocker.
10. If the provider notification operation itself is unavailable, use an authorized fallback channel such as Gmail/email only when available and authorized; otherwise surface the local HTML path in chat and log the notification blocker.
11. Save a report-delivery record in `daily-content-pipeline/notifications/notification_log.md`.

The report-delivery record must include:

```yaml
html_report_path:
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
provider_uploaded_report_url_ttl:
provider_notification_attempted: true | false
provider_notification_status: sent | failed | unavailable | skipped
fallback_notification_channel:
final_report_link_sent_to_human:
blocker:
```

Completion is invalid if the agent says only `report ready` or `config updated` without this delivery outcome when a scheduled run or daily report was requested.

Correct blocker wording:

```text
Report delivery: HTML report generated. Provider capability check completed. WideCast OpenAPI discovery or account verification is blocked by `{exact blocker}`, so I logged `{provider blocker}` and am giving you the local HTML report path here.
```

Incorrect blocker wording:

```text
WideCast cannot upload reports.
```

That is too broad unless the agent verified WideCast account/API status and the current OpenAPI spec directly.

---
