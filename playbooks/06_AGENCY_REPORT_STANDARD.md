# Agency Report Standard

Stage: `06`

## Load Rule

Load whenever generating, reviewing, debugging, or improving a human-facing report.

## Hard Gates For This Stage

- Human-facing reports are HTML only.
- Markdown is internal.
- The report must be standalone, mobile-friendly, agency-grade, and factually aligned with the Markdown source.
- Include reference URLs beside claims, ideas, leads, competitors, and drafts.
- Do not create fake action buttons in static HTML.

## Source Preservation Rule

This file is detailed source material moved from the original monolithic `SOLO_AGENCY_PLAYBOOK.md`.

Do not summarize away requirements, examples, checklists, schemas, protocols, URLs, edge cases, warnings, approval gates, or completion gates. If a downstream agent needs to shorten its response to the human, it may summarize the response, but it must still obey the full requirements in this file.

---

## 11. Daily Output File Format

Each active client must have one daily output file:

```text
outputs/YYYY-MM/YYYY-MM-DD.md
```

Markdown is the canonical internal output record. HTML is the only human-facing rendered report.

The agent must keep the Markdown file even when an HTML report is created, unless the current environment truly cannot write Markdown. The Markdown file is required for:

- agent-readable history;
- future learning and optimization;
- duplicate idea detection;
- diffing changes across days;
- regenerating HTML reports;
- preserving references, reasoning, leads, competitors, and WideCast scripts without parsing HTML.

The HTML report must be created from the same facts, references, ideas, analysis, and draft content as the Markdown report, but it may use a custom structure and design. It must not become a factually divergent report. If the agent can only preserve one long-term artifact, preserve the Markdown file first and regenerate HTML later. If the agent can only deliver one artifact to the human, deliver the HTML report because all user-facing reports must be HTML.

### Human-Facing Report Rule: HTML Only

The agent must show report results to the human as HTML only.

Do not show, send, link, or ask the human to open the Markdown report as the user-facing report.

Allowed:

- Save `.md` internally for agent memory, history, learning, diffing, and regeneration.
- Mention that a Markdown source file exists only when explaining internal storage or troubleshooting.
- Deliver the `.html` report path/link to the human.
- Send Telegram/WideCast notifications with the `.html` path/link.

Not allowed:

- "Open the Markdown report."
- "See `outputs/YYYY-MM/YYYY-MM-DD.md` for details."
- Using the `.md` file as the primary human-facing report.
- Sending both `.md` and `.html` and making the human decide which one to open.

Every user-facing report path, notification, or review instruction must point to the `.html` file.

### Internal Markdown, Beautiful HTML

The agent must create two artifacts with different purposes:

1. Markdown is the internal canonical record for agent memory, history, learning, diffing, and regeneration.
2. HTML is the human-facing report and must be designed for a polished mobile review experience.

The HTML report does not need to be a direct Markdown render. If a direct Markdown-to-HTML renderer produces an ugly or hard-to-use report, the agent should create a better designed standalone HTML report instead.

Correct behavior:

1. Author and save the complete internal report as `outputs/YYYY-MM/YYYY-MM-DD.md`.
2. Create `outputs/YYYY-MM/YYYY-MM-DD.html` as a polished human-facing report using the same facts, references, ideas, lead/competitor data, and draft content.
3. The HTML may be custom structured and styled for readability, mobile scanning, editable draft review, and copy workflow.
4. The HTML must not omit required report sections that exist in the Markdown.
5. If the Markdown report changes, update/regenerate the HTML so both artifacts stay factually aligned.

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
outputs/latest.md
outputs/latest.html
```

`latest.md` is internal. `latest.html` is the human-facing convenience file.

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

## Private Collector Health

- Bridge status:
- Bridge persistent mode:
- Extension status: recent | stale | no_extension_check_yet | unavailable
- Last extension check:
- Seconds since last extension check:
- Chrome/session status:
- Private collection impact:
- Required human action:

## Private Sources Pending Activation

Use this section when private sources were provided but the Solo Agency Local Collector extension and Local Collector app are not activated yet.

- Status: pending_private_activation | activated | not_provided | unavailable
- Why private sources were not scanned today:
- What is needed to activate them:
- Suggested next question:
  - `Private sources are not activated yet because they require the Solo Agency Local Collector extension and Local Collector app. Do you want me to set that up now?`
- Sources waiting for activation:
  - Source:
    - URL:
    - Platform:
    - Why it matters:

## Private Source Discovery

Use this section when the human approved, declined, postponed, or has not yet been asked about optional source discovery from joined groups, followed profiles/pages/KOLs, subscribed channels, or platform recommendation feeds.

- Status: not_asked | declined | approved_pending_activation | active | blocked | completed
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

### Public Sources

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

### Private Sources

- Source:
  - URL captured:
  - Login/session status:
  - Notes:

## Sources Skipped

| Source | URL | Reason | Next Action |
|---|---|---|---|
|  |  |  |  |

## New Private Sources Detected

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

## Leads Detected

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
  - Outreach/compliance note:

## Competitors Detected

- Competitor:
  - Competitor type: direct | adjacent | audience
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

## Idea Matrix

### 1. Hot / Trend / News

#### Global

- Idea:
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

- Idea:
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

- Idea:
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

- Idea:
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

- Idea:
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

- Idea:
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

Include the reference URLs that support the script's key claims. For private sources, include the captured private URL and note that the human may need to be logged in to verify it.

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
  - Internal source: `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outputs/YYYY-MM/YYYY-MM-DD.md`
  - Human-facing report: `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outputs/YYYY-MM/YYYY-MM-DD.html`
  - Internal latest pointer: `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outputs/latest.md`
  - Human-facing latest report: `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outputs/latest.html`
- Master report:
  - Internal source: `daily-content-pipeline/outputs/YYYY-MM/YYYY-MM-DD_master_digest.md`
  - Human-facing report: `daily-content-pipeline/outputs/YYYY-MM/YYYY-MM-DD_master_digest.html`
  - Internal latest pointer: `daily-content-pipeline/outputs/latest_master_digest.md`
  - Human-facing latest report: `daily-content-pipeline/outputs/latest_master_digest.html`

The human should only be shown the `Human-facing report` and `Human-facing latest report` paths. Internal `.md` paths are for the agent only.

Do not present this older ambiguous shape to the human:

```text
Open either latest.md or latest.html
```

Only present:

```text
Open latest.html
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
- Short section summaries before long details.
- Collapsible sections when the report is long, if the environment can generate them.
- Clear color/status labels for hot leads, warm leads, high-threat competitors, approval-needed items, and session-expired blockers.

The HTML report must include:

- Run date.
- Agent identity.
- Clients processed.
- Client status.
- Private collector health: bridge status, extension last check time, extension status, and private-source blockers.
- Private Source Discovery status when asked, approved, pending, blocked, or completed.
- Top ideas.
- Best idea.
- Mapped content pillar.
- Reference URLs.
- Leads detected.
- Lead profile URLs and post/current URLs.
- Competitors detected.
- Competitor profile URLs and post/current URLs.
- Data source issues.
- Private sessions needing login.
- WideCast-writing-skill draft: video script, blog/article, social caption, or configured combination.
- Production/provider status: draft only, approval required, provider setup required, video/blog/social asset created, ready to publish, published, or blocked.
- `Unlock Production & Distribution & Measure-Learning Loop With WideCast` section when WideCast account tools, Telegram notification, publishing, or video creation are not connected yet.
- Approval options.
- Next actions.

### Agency-Grade HTML Report Standard

The HTML report is not a data dump. It is a professional agency decision report. A busy client or agency owner should be able to open it on a phone, understand the opportunity, verify the evidence, approve a draft, and know the next action within 60 seconds.

The report must feel like it came from a capable media strategist, not from a crawler. It should combine research, judgment, prioritization, production readiness, and clear client communication.

Required report hierarchy:

1. `Executive Snapshot`
   - Client name.
   - Run date and agent identity.
   - Source coverage status: public-only, public + private, private pending, private failed, or mixed.
   - Best idea of the day in one sentence.
   - Why it matters today.
   - Content asset status: draft ready, approval required, provider setup required, video/blog/social asset created, ready to publish, published, needs human detail, needs visual assets, or blocked.
   - Lead count: hot, warm, and not scanned/pending if applicable.
   - Competitor signal count.
   - One recommended next action.

2. `Today's Recommendation`
   - The single best idea.
   - Target audience segment.
   - Pain point or desire it hits.
   - Content pillar.
   - Primary industry or related industry label.
   - If related industry, explain the business logic bridge back to the client's offer.
   - Why this won over the other ideas.
   - Confidence level: high, medium, or low.
   - Main reference URLs.

3. `Evidence Ledger`
   - Every important factual claim, number, date, law/regulation point, price, platform policy, market signal, or news claim used in the report must appear in a claim-level evidence table.
   - Required columns: `Claim`, `Source URL`, `Source type`, `Captured at`, `Confidence`, `Used in`.
   - If a claim is inferred rather than directly stated by a source, label it `inference` and explain the logic.
   - Do not use unsupported numeric claims in the final recommendation or script.
   - If a useful claim cannot be verified, either remove it or mark it as low confidence and keep it out of the main hook.

4. `Source Coverage And Data Quality`
   - Public search keywords used today.
   - Public sources scanned.
   - Private sources scanned, pending, skipped, failed, or session-expired.
   - New private sources detected.
   - Known blind spots for this run.
   - Data confidence summary.
   - If private sources were provided but not activated yet, state that clearly and do not imply lead coverage is complete.

5. `Private Source Discovery`
   - Discovery categories approved, declined, pending, or not requested.
   - Platforms and discovery URLs used.
   - Whether the Solo Agency Local Collector is active, pending, or blocked.
   - Candidate groups, profiles, pages, KOLs, channels, communities, and feed-surfaced sources found.
   - Which candidates are recommended for daily, weekly, optional, or watch-once monitoring.
   - Which candidates were skipped as irrelevant, too broad, too noisy, sensitive/risky, or unavailable.
   - Feed signals detected, with current URL and source when visible.
   - Sources requiring human approval before being activated.
   - Reassurance summary: professional one-time setup, local-only data safety, and daily scanning to avoid missed signals.

6. `Idea Portfolio`
   - Keep the three-section idea structure:
     - Hot / Trend / News
     - Evergreen / Foundation
     - Lead-Gen / Conversion
   - Include both global and local angles when they matter.
   - For every idea, show:
     - Title.
     - Global or local label.
     - Primary industry or related industry label.
     - Pain point or content pillar.
     - Reference URL(s).
     - Short rationale.
   - Score or qualitative rating for heat, relevance, lead potential, novelty, and confidence.
   - Empty weak slots are allowed. Do not fill a matrix slot with filler just to make the report look complete.

7. `Decision Scorecard`
   - Compare the top candidate ideas before choosing the best one.
   - Score at least: trend heat, audience pain intensity, business relevance, lead potential, novelty/history risk, evidence strength, and production effort.
   - Briefly explain why the selected idea won and why the other strong candidates did not win today.

8. `Leads Detected`
   - Separate `Hot leads`, `Warm leads`, and `Not scanned / pending`.
   - Each lead must include profile URL and post/current URL when available.
   - Include source, captured_at, need signal, why hot/warm, suggested next step, and safety note.
   - If no leads were found, say whether that means `no leads found after scanning` or `lead sources not activated yet`.

9. `Competitor Intelligence`
   - Separate direct competitors, adjacent competitors, and audience competitors.
   - Each competitor item must include profile URL and post/current URL when available.
   - Include positioning, repeated content theme, engagement signal, threat level, and opportunity.
   - If competitor data is inferred without a captured URL, label it as market hypothesis, not detected competitor evidence.

10. `Production-Ready Drafts`
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

11. `Compliance And Brand Safety`
   - Include short risk notes for legal, financial, insurance, medical, regulated, or sensitive industries.
   - Avoid guarantees, misleading claims, unsafe outreach, or pretending the client has performed services they have not performed.
   - If a CTA needs license number, phone number, disclaimer, location, or offer approval, list it under `Needs human detail`.

12. `Next Action`
   - End with exactly one primary next action.
   - Secondary actions may be listed below, but they must not compete with the primary next action.
   - Before the first agency run, if schedule/routine is not configured yet, the primary next action should be schedule/routine setup.
   - After schedule/routine is configured but the first agency run has not happened, the primary next action should be asking whether to run the first agency run now.
   - After the first agency run small win exists and Production & Distribution & Notification & Analytics setup has not been completed/declined/blocked, the primary next action should usually be that setup gate.
   - For reports where production setup is completed/declined/blocked and private sources are pending, the primary next action should usually be activating the Solo Agency Local Collector or marking private sources pending, not starting a video branch.
   - Do not ask "make a video now?" as the primary next action immediately after the small win; production/provider setup comes first.

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
- If WideCast notification/Telegram is available, the agent must try to deliver the report through WideCast notification.
- If a WideCast HTML-capable report/file/asset upload tool is available, upload the `.html` report first and send the uploaded WideCast report URL.
- If WideCast report upload is unavailable or fails, the agent must log the blocker and still include the best available local/hosted `.html` report path/link in the notification.
- If the agent accidentally sends a notification without a report URL/path, it must immediately send a correction notification containing the HTML report URL/path and log the correction.

Do not end a report handoff with:

- only the report link;
- only a summary;
- "let me know";
- "see next steps in the report";
- multiple competing questions.

Examples of correct final questions:

```text
Do you want me to activate private sources now?
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
- Do not pretend public-only research has private lead coverage.

Recommended HTML section order:

```text
1. Executive Snapshot
2. Today's Recommendation
3. Evidence Ledger
4. Source Coverage And Data Quality
5. Private Source Discovery
6. Idea Portfolio
7. Decision Scorecard
8. Leads Detected
9. Competitor Intelligence
10. Production-Ready Drafts
11. Compliance And Brand Safety
12. Unlock Production & Distribution & Measure-Learning Loop With WideCast, when applicable
13. Next Action
14. Appendix / Raw References, optional
```

Static HTML reports are not application UIs. The agent must not create buttons that imply an action will happen when the human taps them unless the button is backed by a real working URL or app action. For approval, revision, choosing another idea, creating a WideCast video, or replying to leads, the report should say what to tell the AI agent or where to open WideCast.

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

- WideCast MCP notification/Telegram tool with the uploaded WideCast HTML report URL when report upload is available. WideCast's notification API may automatically fall back to email when the human has not connected Telegram yet.
- If the WideCast notification tool itself is unavailable, use a connected Gmail/email MCP, connector, or tool to email the HTML report or HTML report link to the human if available and authorized.
- Agent chat file attachment if supported.
- Local file path in the automation/thread output.
- Slack, Discord, Google Drive, Notion, or other connector if available and authorized.
- If none of those are available, provide the local path and clearly say where to open it.

Notification fallback rule:

- WideCast MCP notification/Telegram is the preferred scheduled-run notification channel.
- If WideCast notification tools are available, call the WideCast notification tool even if the human has not connected Telegram yet. WideCast should handle fallback email delivery when Telegram is not connected.
- If WideCast notification tools are available and WideCast exposes an HTML-capable report/file/asset upload API, upload the `.html` report to WideCast first and send the uploaded URL through WideCast Telegram/email fallback.
- Do not send only a local file path when an uploaded WideCast report URL is available.
- If the current WideCast wrapper cannot upload `.html` files, log `widecast_report_upload_unavailable`, send the best available HTML path/link, and state the upload blocker clearly.
- Never send a report-ready notification that contains only a status summary. The notification must include an `HTML report` URL/path field.
- The agent should not switch to Gmail/email merely because Telegram is not connected in WideCast.
- Use Gmail/email only when WideCast notification tools are unavailable or blocked.
- If Gmail/email is available and WideCast notification tools are unavailable, send the HTML report or a link/path to the HTML report by email to the human, using the same language the human uses.
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

---
