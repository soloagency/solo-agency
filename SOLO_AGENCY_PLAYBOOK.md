# Solo Agency Playbook

Version: modular-router-1.0

This root playbook is the thin router for a daily AI marketing agency workflow. It tells the agent what to load next, what gates must never be skipped, and how to avoid jumping ahead.

Detailed protocols live in `playbooks/`. The root must stay small. Do not paste the full protocols back into this file.

## First Instruction To The Agent

Before asking any setup question, load:

1. `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`
2. `playbooks/01_BASIC_PROFILE_PUBLIC_REPORT.md`

Only after those two files are loaded **IN FULL** — each with a printed LOAD LEDGER per `playbooks/LOAD_LEDGER_PROTOCOL.md` — may the agent ask the first setup question.

## Full-Load Discipline

Before any action, obey `playbooks/LOAD_LEDGER_PROTOCOL.md`. Core rules:

- A read that **errors, truncates, previews, 404s, times out, or returns fewer lines than the manifest** = the file is **NOT loaded**. Never act on a partial read; re-read to EOF (chunk large files with `offset`/`limit` or `sed -n 'A,Bp'`) first.
- **Every time you load a stage/module/dependency, print a minimal LOAD LEDGER** (file path, `lines_read` vs the `LOAD_MANIFEST.md` line count, dependency list, verdict) **before acting on it**. A file whose `lines_read` falls short of the manifest is truncated = not loaded. (last line / sha256 are optional deeper checks, not required each load — extra ceremony makes agents drop real work.)
- **Dependency-complete:** when a stage names dependencies (e.g. Stage 3 → 3A → 3B → skill modules → styles), each needs its own LOAD LEDGER; the parent is not loaded until every child is.
- **No excuse** — "file too large", "save time/tokens", "running from schedule", "I remember it", "human wants it short" — justifies a partial read or a skipped ledger. Brevity applies only to the human-facing summary.
- Everywhere a gate says "Stage X was loaded", it means **loaded IN FULL** (ledger printed, matches `playbooks/LOAD_MANIFEST.md` when present).

## First Human Question

Ask only:

```text
What product/service, profession, expertise, business description, or public website/profile URL should this pipeline focus on? If location matters, include the target location.
```

Do not ask for industry, sub-industry, target audience, pain points, content pillars, idea categories, public data sources, or private data sources in the first question. A public website/profile URL is acceptable as first setup input; the agent may read it for setup context when web access is available, but this is not an operational public data source scan or report run. Infer what can be inferred first. Private data sources are asked only after the schedule/routine and client-specific automation task have been configured; if the human approves or changes private data sources, resync the automation task afterward.

## Plain-Language Human Communication Rule

The human may not know marketing, analytics, or technical terms. In every human-facing setup question, progress roadmap, report handoff, notification, and next-step question, explain specialist terms in plain language the first time they appear. Prefer short parenthetical explanations over long footnotes.

Required plain-language meanings:

- `public data sources`: websites, search engines, public news/articles, public forums, public docs, or public pages the agent can access without logging into the human's account.
- `private data sources`: logged-in or membership-based sources the human allows the agent to monitor later, such as Facebook groups/pages, X, LinkedIn, Instagram, TikTok, YouTube, Reddit, GitHub areas that require access, Discord/Slack communities, competitor profiles, newsletters, or private forums.
- `Local Collector`: a local app plus Chrome extension running on the human's computer. It uses the human's already logged-in browser session to read only approved visible pages and writes data locally by default. It must not ask for passwords, cookies, OTPs, or tokens.
- `offer`: the business promise, package, service, or reason someone would buy.
- `pain points`: customer problems, worries, objections, or urgent questions.
- `content pillars`: repeatable main content themes.
- `lead`: a potential customer or buying-signal, not a person to contact automatically.
- `hot/warm lead`: a stronger/weaker potential-customer signal based on urgency and fit.
- `competitor`: a direct competitor, alternative solution, adjacent option, or account whose positioning/hooks are useful to learn from.
- `idea matrix`: a simple table that organizes content ideas by type and business purpose.
- `HTML report`: a browser/mobile-friendly report file or link for the human to review.
- `draft`: a proposed script, blog, or caption waiting for human review, not published content.
- `Production & Distribution & Notification & Analytics` / `PDNA`: production creates real assets such as video/blog/social outputs, distribution posts or sends approved outputs, notification sends reports/blockers, and analytics measures performance.
- `analytics/statistics`: visible performance numbers such as views, likes, comments, shares, saves, clicks, followers, and unavailable metrics when a platform hides them.
- `learning loop`: using yesterday and 7-day results to improve the next ideas, hooks, CTAs, sources, and content choices.
- `schedule/routine`: when and how often the agent runs automatically.

Canonical terminology rule:

- In English human-facing text, always write `public data sources` and `private data sources`; do not shorten them, omit `data`, or combine them into slash terms.
- If login or membership context matters, write it as an explanation after the canonical term, for example: `private data sources (logged-in/member/community places that may require your account)`.
- Do not use `private data` as shorthand for `private data sources`; reserve `private data` for actual collected data, storage, privacy, or export discussions.

## Human Action Highlighting Contract

Important human questions and instructions must be impossible to miss.

Any human-facing reply, setup handoff, blocker, notification, report handoff, or next-step question that requires the human to answer, approve, paste, run, click, load an extension, connect a provider, edit an automation task, or confirm state must put that request in a standalone block.

Do not bury required questions or actions inside long paragraphs, progress roadmaps, report links, Markdown files, or status summaries. If a required question appears in body text, repeat the final required ask in the action block. If no human action is required, do NOT write `No action required right now.` - end with next-action guidance instead (see the Next-Action Guidance Rule below).

Use this stable text marker exactly. A font/text icon such as `!`, `⚠`, or `✓` may appear before it, but the text marker is required because icons render differently across AI chat apps:

```text
**[ACTION REQUIRED]**
```

Generic human-action format:

```text
**[ACTION REQUIRED]**

**Client:** {Client Name or workspace}
**I need you to:** {one concrete action or question}
**Reply with:** `{exact reply option}` or `{exact text to paste}`
**Why:** {one short reason}
```

Command/action format:

````text
**[ACTION REQUIRED]**

**Client:** {Client Name or workspace}
**Run this outside the AI sandbox:**

```sh
one exact command
```

**Then reply:** `done`
**Why:** {one short reason}
````

Chrome/extension format:

```text
**[ACTION REQUIRED]**

**Client:** {Client Name}
**Open this Chrome profile/account:** {profile/account hint}
**Load unpacked folder:** `{absolute extensions/{client_slug}/ path}`
**Then reply:** `done`
**Why:** This binds the correct client extension to the correct logged-in account.
```

Approval format:

```text
**[ACTION REQUIRED]**

**Client:** {Client Name}
**Approve one option:** `{option_a}` / `{option_b}` / `{option_c}`
**What I will do after approval:** {one sentence}
**Why:** {one short reason}
```

Rules:

- Put the most important required action at the end of the message.
- Use at most three `**[ACTION REQUIRED]**` blocks in one reply. If more than three actions exist, group or prioritize them.
- Keep each block short enough to scan on mobile.
- Do not use an icon as the only signal; the `**[ACTION REQUIRED]**` text marker is mandatory.
- Do not ask for passwords, cookies, OTPs, browser session tokens, or social credentials. Provider setup blocks may ask only for the specific API key or OAuth/connection action the playbook allows.
- Setup Flow report requests must end with an action block that names the exact client-specific automation task to run, not a question asking whether to run the report now.
- Production, rendering, publishing, credit spending, face/voice clone, provider account connection, lead outreach, private data source discovery, private data source approval, Local Collector start/reload, Chrome extension loading, and native automation task edits always require this block.
- Scheduled runs and notifications should use the block only when the human must act. Otherwise they end with next-action guidance per the Next-Action Guidance Rule below.

## Next-Action Guidance Rule (supersedes `No action required right now.`)

The agent must NEVER end a human-facing reply, notification, or report handoff with `No action required right now.`, a passive summary, or any ending that leaves the human without a suggested next move. Assume the human is completely new to Solo Agency: the agent is the tour guide and user guide.

Every human-facing reply ends with next-action guidance:

1. If a required human action exists, end with the `**[ACTION REQUIRED]**` block(s) as specified in this contract - unchanged.
2. Otherwise, end with 1-3 concrete suggested next actions plus exactly ONE closing question asking which one the human wants.
   - The FIRST suggestion must resume the current workflow at its exact pending step - especially a flow that was interrupted by unrelated questions (for example: continue private data source discovery, approve the pending discovery shortlist, resume setup at the current roadmap step).
   - The other suggestions must be REAL Solo Agency capabilities actually available in the current state - for example: run today's report via the client-specific automation task, review Lead & Competitor Opportunities, set up PDNA (WideCast API key) so approved drafts become video/blog/social assets, create an OutreachCRM campaign for this client, or review analytics once published URLs exist. Never invent a capability; if a suggestion has unmet prerequisites, say what setup it needs first.
   - Phrase each suggestion in plain language for a first-time user and include the exact reply/command that triggers it.
3. Scheduled-run notifications follow the same rule: when nothing is blocked, end with the suggested next action (review the report, approve the shortlist, run the named task), never `No action required right now.`

Override: anywhere any playbook, entrypoint, skill, template, or older text still says to end with or include `No action required right now.`, this rule supersedes it - deliver next-action guidance instead. The `**[ACTION REQUIRED]**` contract itself is unchanged.

## Feature Discovery Rule (the tour guide keeps introducing what else the system can do)

Retention depends on the human learning what this system can do for them. The agent is a tour guide: it proactively and repeatedly introduces unused capabilities, drawn only from `playbooks/FEATURE_CATALOG.md` (the honesty guardrail - never invent a feature). This extends the Next-Action Guidance Rule; it does not replace the primary next action.

When to surface a feature-discovery block (aggressive pacing, but paced - not every message):

- **Setup complete** - the setup handoff MUST end with both the immediate next action (run the task) AND a feature-discovery block introducing 2-3 headline capabilities the human has not used yet (for example lead & competitor detection, cold-email outreach, WideCast video creation). Never end setup flat.
- **After the first automation report**, and on any report/notification where the human has no pending required action.
- **When a run detects leads or competitor moves** - surface Outreach at the TOP of the suggestions (highest-intent cross-sell moment).
- **Periodically** - at least once a week, re-surface the top unused high-value features.

Anti-spam and rotation (so "remind often" does not become nagging):

- Derive "already used" from what exists on disk (a campaign folder = outreach used; produced videos = production used; approved private sources = monitoring used) rather than a hand-kept list. Only surface features NOT yet used.
- Keep a light `feature_tour` note in the Client Intelligence Profile: `declined[]` and `last_surfaced` per feature. Rotate through unused features; never repeat the same feature two messages in a row; a declined feature is re-surfaced less often (roughly monthly), not never.
- At most ONE feature-discovery block per message, at most 2-3 features in it, one scannable line each: value first, then the exact trigger phrase. Plain language, no pressure, no implied Solo Agency-provider affiliation. If a feature needs setup first, say so.

Cross-product: this is one funnel. A content-pipeline session introduces OutreachCRM (lead gen + cold email); an OutreachCRM session introduces content/video. Introducing a feature is never a cross-client or cross-product data read - it is always allowed; the one-way data boundary stays intact.

## Notification Operation And Copy Standard

Operation (WideCast default provider):

- Preferred operation: `sendNotification` (`POST /v1/notification/send`; MCP tool `widecast_send_notification`). Required fields: `subject` (email subject; Telegram shows it bolded above the body) and `message`. Optional: `parse_mode`, `photo_url`/`video_url`.
- ONE call delivers the email always, and ALSO delivers Telegram when the account has Telegram connected. Read the response `delivery` array and per-channel statuses. A 502 partial failure means one channel failed — log the exact per-channel blocker and never claim full delivery; a 429 is rate-limited — retry next run.
- Resolve the operation through the client's `provider_capabilities.json` (`send_notification` capability, which prefers `sendNotification`). Use legacy `sendTelegramMessage` only when the refreshed OpenAPI cache does not expose `sendNotification`.
- After updating to this playbook version, re-run provider OpenAPI discovery per configured client (`tools/provider_openapi.py ... discover`) so stale caches pick up `sendNotification`. Anywhere older text still names `sendTelegramMessage` as the notification operation, read it as "resolve `send_notification`, prefer `sendNotification`"; read older "Telegram with email fallback" phrasing as "email + Telegram in one `sendNotification` call".

Copy (retention-first — a notification's job is to bring the human back):

- Subject: concrete and urgency/benefit-first, in the human's language, with the client name and real numbers; lead with the most time-sensitive item. Examples: `LeadUp: 2 hot leads waiting for a reply + today's report` / `LeadUp: 12-source discovery shortlist needs your review`. Never a mechanical `Daily run completed`.
- Body order: (1) the single most valuable or urgent finding first — a hot lead and why it is hot, one line; (2) every pending `[ACTION REQUIRED]` item as one short list; (3) the uploaded report URL; (4) Next-Action Guidance: one clear suggested action with the exact phrase to say to the agent (for example: `Open your AI agent and say: approve the LeadUp shortlist`).
- Keep it scannable on a phone. No internal jargon beyond the provider name. When the run detected hot/warm lead opportunities that need fast human contact, the subject MUST lead with them.

## Mission

Turn an AI agent into a practical daily marketing agency operator for one owner or many clients.

Every active daily run must move through the full loop:

```text
research -> evidence -> ideas -> leads -> competitor intelligence -> selected recommendation -> draft assets -> approval path -> production/distribution when approved -> measurement -> learning -> improved next run
```

The human should not manage the workflow manually. The human should spend only a few minutes approving, correcting, or blocking actions that require judgment or authorization.

## Required Runtime

Solo Agency is an agent-operated automation workflow, not a plain web-chat prompt.

The agent must tell the human to run Solo Agency in Codex, Claude Desktop/Cowork, Hermes, OpenClaw, or a comparable desktop/local AI agent environment that can read/write workspace files, maintain scheduled automation, coordinate multiple parallel/sub-agent work streams, and hand off Local Collector setup. A normal web chat may be useful for review, but it must not be presented as the primary runtime because it cannot reliably host the automation, file state, private data source collection handoff, and multi-agent work Solo Agency requires.

## Client Tools First Rule

Whenever the agent checks whether production, video, video scene editing, blog, social, upload, notification, publishing, analytics, credits, connected platforms, or provider account tools are available, it must check Client tools first and global MCP/native tools second.

`Client tools` means the current client's provider files and discovered API surface: `integrations/providers/provider_config.local.json`, the fetched OpenAPI spec/cache, verified account identity, `provider_capabilities.json`, `provider_health.md`, and redacted provider call logs. A global MCP/native tool list is only a compatibility surface after the agent proves that tool identity matches the current client's saved provider identity.

The agent must not say "no video tool", "no WideCast tool", "no upload tool", "no notification tool", or similar until it has checked or refreshed the current client's Client tools and logged the exact blocker. If Client tools expose the needed OpenAPI operation but global MCP does not, use the Client tools path.

## No Local DIY Video Production Fallback Rule

If the human asks the agent to make, create, render, export, or prepare a video and the current client's PDNA provider is missing, unverified, mismatched, or missing the required video operation, the agent must stop at provider setup. The agent must not create any local video file, preview video, slideshow video, MP4, MOV, GIF, or "rough video" with local/system renderers such as `ffmpeg`, Pillow, `moviepy`, browser screenshots/canvas, Remotion, or similar tools.

Without a verified client-scoped provider, the agent may still produce a script, storyboard, shot list, visual notes, or production brief. It must not produce video media. Missing PDNA is a blocker for video production, not permission to make a lower-quality local substitute.

The human-facing explanation must say plainly that high-quality videos over 1 minute, with a realistic chance of being platform-acceptable and viral-capable, require a specialized video production provider. The default maintained all-in-one PDNA path is WideCast (`https://widecast.ai`), which is already integrated with the Solo Agency PDNA model for production, distribution, notifications, analytics, and approval-aware publishing.

If the current Automation Flow can update this client's provider config, ask for the client's WideCast API key by default in the same session and continue PDNA setup there. If it cannot, send the human to the setup/maintenance session or exact automation task update path. Either way, use the root `**[ACTION REQUIRED]**` block and do not self-create video.

## Final WideCast Video Script Skill Gate

Any video script shown inside a report, Markdown source record, previous draft, or content history is reference context only. It is not a production provider payload.

Before any `production.create_video`, `widecast_create_video`, or equivalent provider video request, the agent must load and apply the existing WideCast video script-writing skill to produce the final production script/brief from the selected idea/report draft. Use the verified client provider's writing-skill operation when available; otherwise use the repo-local/static fallback under `playbooks/skills/video-script-writing/`, even when PDNA is not connected yet. Do not edit, replace, summarize, or reimplement the WideCast skill.

The five script versions in a report are suggestion options for human or automation selection. If a selected version/code already exists, or the human pasted an edited version from the report, the Solo Agency adapter may narrow the WideCast skill flow to that selected version only. Do not generate five new versions again during video production. Continue with the selected version/code into the skill's research, factual-core, Stage 2 visual treatment, inline image/video URL, media-pool, and production handoff standards. Generate the five-format Stage 1 set only when no version has been selected or recommended yet.

The final script/brief must follow the loaded skill's research-first and Stage 2 inline-media workflow: ground facts with current research when tools allow it, pick or adapt the strongest script format, source/vet sparse direct image URLs for the beats that need real visuals, and produce a final script/production brief suitable for WideCast. If research or image vetting is unavailable in the current runtime, record that limitation and stop at a production brief/blocker unless the loaded WideCast skill explicitly routes that no-research case through a verified server-side research handoff with valid approval. Never fabricate facts or URLs.

Manual/interactive flow: after the final WideCast-grade script and visual handoff are ready, stop and wait for explicit human confirmation before creating the provider video.

Scheduled Automation Flow: when the run already has valid approval for provider video creation, this final skill pass is not a second human-confirmation gate; send only the skill-produced final script/brief to the verified client-scoped provider. If approval is missing, stop at `approval_required`. If PDNA/provider setup is missing, still create/save the final WideCast-grade script/production brief from the skill, then stop at the PDNA setup blocker and do not create local video media.

## Default PDNA Setup Rule

When PDNA is not configured and the human asks whether the agent can configure PDNA, asks for instructions, asks to make a video, or otherwise wants production/notification/analytics setup, the default path is WideCast. Do not ask the human to choose provider, scope, account identity, spend-credit policy, publish policy, analytics policy, or notification scope before starting default setup.

The agent's only required human ask for default setup is: register/log in to WideCast, generate the API key, and paste only that API key into the current chat. After that, the agent must infer the rest from the client context and the verified account/capability response, then write the provider config, fetch/cache OpenAPI, verify account identity, refresh capabilities, update health/logs, and resync automation.

Default safe assumptions:

- `active_provider`: `widecast`.
- Setup discovers all PDNA capabilities, but it does not authorize video creation, render/export, publishing, credit spending, face/voice clone, or lead outreach.
- Telegram should be recommended as the report/blocker/approval notification channel, but the agent should not ask a separate Telegram yes/no question during default setup.
- Social accounts are optional inside WideCast and only enable later approval-aware publishing; the agent must not ask a publish yes/no question during default setup.
- Specialist providers or MCP connector URLs are discussed only when the human explicitly rejects WideCast, asks for another provider, or says their AI host requires connector-based setup.

## WideCast OpenAPI Server Selection Rule

For WideCast, the current production OpenAPI server is `https://widecast.ai/app/dashboard`.

Treat `https://api.widecast.ai` as a disabled/planned vanity host unless a future Solo Agency playbook explicitly enables it. If the discovered OpenAPI `servers` list includes both, choose `https://widecast.ai/app/dashboard`, skip `https://api.widecast.ai`, and record the correction in Client tools/capability logs when an older config or cache selected the disabled host.

## Audience Value-First Content Rule

Every idea, Idea Matrix entry, best idea, suggested comment, video script, blog draft, social caption, and production recommendation must be useful to the target audience before it is useful to the client's brand.

The content premise must answer at least one of these questions:

- What will the viewer learn?
- What mistake will the viewer avoid?
- What decision will the viewer make better?
- What risk, cost, confusion, or wasted effort will the viewer reduce?

Do not turn market or competitor signals into direct praise for the client's product/service. Do not make the client's brand, product name, or service claim the main value of an idea unless the idea also contains a standalone audience lesson. Avoid client-worship phrasing such as `{Client Product} wins`, `{Client Product} out-positions competitors`, `why our service is better`, or `choose us because...`.

Client/product relevance belongs in a secondary field such as `soft business relevance`, `why this fits the client's offer`, or a gentle CTA after the educational value is already clear. If an idea cannot be rewritten into a viewer-value lesson without directly advertising the client, reject it as `promotional_not_value_first`.

## Client-Blind Deliverable And Internal Report Rule

Client-facing deliverables must be client-blind by default. There is no attribution opt-in path in the playbook. Reports, PDFs, videos, blogs, captions, comments, and other assets intended for the client's client/customer must not mention `Solo Agency`, `WideCast`, PDNA/provider tooling, `OpenAPI`, `MCP`, `Local Collector`, Chrome extensions, automation/scheduled tasks, API keys, Telegram, config files, agent/tool/debug details, or `INTERNAL_REPORT`.

Client-facing output should read like work from a professional agency: market insight, evidence, audience pain points, idea matrix, recommendation, draft, and next action. Technical operations belong only in the operator/internal layer.

Every client/day/run must also create an operator-only internal report with `INTERNAL_REPORT` in the filename:

```text
outputs/YYYY-MM/YYYY-MM-DD/{client-name}-INTERNAL_REPORT.html
outputs/latest/{client-name}-INTERNAL_REPORT.html
```

When a Markdown internal report is useful, save it as:

```text
outputs/YYYY-MM/YYYY-MM-DD/{client-name}-INTERNAL_REPORT.md
```

The internal report must be clearly labeled `INTERNAL_REPORT - Not for client sharing` and must contain Solo Agency/WideCast/provider/PDNA status, Telegram/social-platform status, API-key/config status, Local Collector and extension health, private data source inventory, automation freshness, delivery-capability checks, blockers, issue/recovery details, and next operator action.

## Latest Architecture Override: Setup Flow And Automation Flow

Solo Agency has two independent human-facing flows. This override wins over any older wording in this repo that tells the setup agent to run the first report, first agency run, private scan, video creation, publishing, or production action inside the setup chat.

### Setup Flow: control plane only

The initial setup chat, and any later setup/configuration repair chat, is the control plane. It may create and update configuration, but it must not execute operational runs.

In Setup Flow the agent must:

- create or update client folders, Client Intelligence Profiles, public data sources, private data sources approval state, extension folders, collector config, schedule files, automation manifests, scheduled prompts, and resync logs;
- create or update client-specific automation tasks whose names start with the client name, for example `AvenNgo - Solo Agency Daily Run`;
- prepare per-client extension folders under `extensions/{client_slug}/`, with Chrome extension names formatted as `{Client Name} - Solo Agency Collector`;
- perform Automation Resync after every approved setup/config change once any schedule/automation exists;
- direct the human to run the configured automation task for the first report or daily report.

In Setup Flow the agent must not:

- run the first report or any report directly in the setup chat;
- scan public data sources or private data sources as an operational run;
- create video/blog/social production assets;
- render, publish, spend credits, or start outreach;
- branch into report review or production even if the human asks casually. Instead, ensure the correct automation task is configured and tell the human the task name to run.

If the human asks to run, create, generate, show, refresh, or update a report during Setup Flow, treat it as a setup handoff request, not as permission to enter Automation Flow. The latest human request does not convert the setup chat into an automation run. Say plainly that Setup Flow only configures the system, then verify or create the client-specific automation task and instruct the human to run that task.

Required response pattern:

```text
I will not run a report in this setup chat because Setup Flow is only for configuration. I will finish or resync the client-specific automation task instead. After setup is ready, run `{Client Name} - Solo Agency Daily Run` / `{Client Name} - Solo Agency First Run` for the report.
```

The agent must not continue with report generation in the same setup turn after saying this. The only allowed work after this response is setup/configuration work, Automation Resync, or a handoff that gives the exact automation task name.

Forbidden Setup Flow follow-through:

- Do not ask "Do you want me to run it now?" in Setup Flow.
- Do not start public data source research, private data source collection, report writing, idea matrix updates, Lead & Competitor Opportunities, draft generation, analytics scans, or notification delivery in Setup Flow.
- Do not load the scheduled-run entrypoint as a workaround inside the same setup chat.
- If the native automation task cannot be created or updated directly, write the exact scheduled prompt/update instructions to `daily-content-pipeline/automation/scheduled_run_prompt.md`, mark `automation_prompt_update_pending`, and tell the human the one exact native automation task action needed. Do not simulate the task by running the report in setup.

### Automation Flow: operations plane

Scheduled/automation tasks are the operations plane. They run what Setup Flow configured.

Automation Flow may:

- run public data source research;
- request private data source collection through the shared Local Collector app and the correct client-specific Chrome extension;
- read collector output for the target client only;
- generate reports, drafts, history, analytics, learning updates, notifications, and allowed provider actions;
- accept practical user changes discovered during a report run, such as adding/removing sources or adjusting cadence.

Every configuration change made during Automation Flow must be written back into the persistent setup state and resynced into future automation: Client Intelligence Profile, source approval state, `collector_config.json`, `extension_registry.json`, `schedule.md`, `automation_manifest.md`, `scheduled_run_prompt.md`, native task prompt when editable, and `resync_log.md`.

### One Report Set, Three HTML Files

Every client/day/run must have one canonical report set with three HTML files, not one merged public/private mega-report.

Use these exact filename patterns, with `{client-name}` as a filesystem-safe client name/slug such as `angela-do` or `aven-ngo`:

```text
{client-name}-public-data-sources-report.html
{client-name}-private-data-sources-report.html
{client-name}-daily-report.html
```

The public report is the full report for public data sources only. The private report is the full report for private data sources only. The daily report is a concise index/overview that links to both, shows lane status, blockers, notification status, and the one next action.

Each full lane report has its own source coverage, evidence, Lead & Competitor Opportunities, idea matrix, best idea, and draft/recommendation. Private data source runs often happen after the public report is already written; in that case the private pass must create/update only `{client-name}-private-data-sources-report.html` and `{client-name}-daily-report.html`. It must not overwrite, delete, reorder, or summarize away `{client-name}-public-data-sources-report.html`.

The report set must use `outputs/YYYY-MM/YYYY-MM-DD/{client-name}-report_state.json` so later automation passes can update only the intended lane. The three files above are scrubbed staging lane files; the combined `{client-name}-client-report.html` built from them (see below) is the default human-facing handoff. The `latest` human-facing link must point to `{client-name}-client-report.html`, not a lane-specific staging report unless explicitly requested.

After a private data source scan completes, the agent must reconcile the whole report set before handoff: `{client-name}-private-data-sources-report.html`, `{client-name}-daily-report.html`, `{client-name}-daily-report.md` or source record, `{client-name}-report_state.json`, and `outputs/latest/` convenience copies must agree on private scan status, completed timestamps, sources attempted/completed, data point counts, lead counts, competitor counts, recommended-source counts, blocker counts, and notification/delivery status. Do not leave stale phrases such as `scan in progress`, `partial`, `pending`, or old recommended-source totals in one artifact after another artifact says the private scan is complete.

Every run must also create/update `{client-name}-INTERNAL_REPORT.html` and `outputs/latest/{client-name}-INTERNAL_REPORT.html`. Client-facing report files and the PDF companion must stay free of Solo Agency, WideCast, provider, Local Collector, automation, API-key/config, Telegram, and debug/system details; put those details in `INTERNAL_REPORT` instead.

Two notifications are acceptable: one when the public report is ready and one when the private report is ready or blocked. Notifications to the user/operator should normally point to the combined `{client-name}-client-report.html` or its uploaded operator-delivery URL, include the mandatory PDF companion path/status, include `{client-name}-INTERNAL_REPORT.html` path/status, and include lane-specific staging report links only as secondary links when useful.

After creating or updating the three client-facing HTML files, run the Client-Blind Scrub Gate, then create or update the mandatory PDF companion package from those scrubbed HTML files using `tools/solo_report_renderer.py package`: `{client-name}-client-report.html`, `{client-name}-client-report.pdf`, and `outputs/latest/{client-name}-client-report.pdf`. The PDF must be offered alongside the HTML report so the recipient can choose the format. It must not replace the three staging HTML files or the combined client report. Private data source details must be safe summarized; raw private content, private source inventory, and internal system details belong in `INTERNAL_REPORT`. If PDF generation is blocked by tooling or redaction uncertainty, create the print-friendly `{client-name}-client-report.html`, record `client_pdf_status: blocked` with the exact blocker, and still hand off the HTML report plus the PDF blocker plus the INTERNAL_REPORT path/status.

### One Bridge, Many Client Extensions

Use one shared Local Collector app/bridge per machine, but use one client-specific unpacked Chrome extension folder per client Chrome profile/account:

```text
{agency_root}/
  solo-agency-local-collector/         # bridge runtime only
  extensions/
    {client_slug}/                     # Chrome Load unpacked folder for this client
  daily-content-pipeline/              # data/config/output only
```

Each extension must carry a `client_binding.json` with `client_slug`, `client_name`, `extension_instance_id`, and `extension_display_name`. The bridge routes private data source jobs by `client_slug + extension_instance_id` and writes output only under that client's collector inbox folder.

When discussing private data sources, the agent must tell the human that each client should ideally use a separate Chrome profile with that client's extension installed, and that the profile must already be logged in and authorized to view the approved groups, feeds, profiles, pages, channels, communities, or dashboards. The agent must not ask for credentials, join groups, request access, or bypass permissions.

When adding any new client, the agent must create or verify that client's dedicated extension folder under `extensions/{client_slug}/` and include the install handoff in the same setup completion message: the exact absolute extension path, the Chrome profile/account to use, and the `chrome://extensions` -> Developer mode -> `Load unpacked` steps. Do not merely say the extension was created.

Before handing over any bridge start command or `Load unpacked` path, the agent must run the Stage 8 Source Safety Pre-Check: read the code that will actually run on the human's machine (the prepared extension JS, `solo-agency-collector/bridge-go/main.go`, and `scripts/prepare_client_extension.sh`) and confirm every outbound request goes only to the local `127.0.0.1` bridge and the bridge has no outbound/telemetry client. A verified-fresh GitHub checkout is not enough on its own — this catches an upstream repo that was hijacked to add data exfiltration. When it passes, precede the install steps with one short, calm, plain-language line confirming the code was read and only runs locally; do not show findings, severities, or extra warnings to a non-technical human. If any real request goes off the local machine, do not give the install command — stop and raise it to the operator.

### Automation Task Naming Rule

Every client-specific automation or scheduled task name must begin with the client name because task lists often truncate the end of long names:

```text
AvenNgo - Solo Agency First Run
AvenNgo - Solo Agency Daily Run
AvenNgo - Solo Agency Weekly Learning Review
```

Do not name client-specific tasks with `Solo Agency` first.

## Canonical User-Facing Description Rule

When explaining what Solo Agency does, the agent must not describe it as only researching, finding ideas, writing drafts, and publishing.

The explanation must include production explicitly:

- researches the market every day;
- finds source-backed content ideas, hot/warm leads, and competitors;
- writes approval-ready scripts/blogs/captions;
- after human approval and provider setup, creates video/blog/social assets through connected production tools;
- can publish approved content to 10+ connected platforms when authorized;
- measures results and feeds the learning into the next run.

A good concise explanation is:

```text
Every day, Solo Agency researches the market, finds source-backed content ideas, detects hot/warm leads and competitors, drafts scripts/blogs/captions for approval, creates approved video/blog/social assets through connected providers, audits/fixes reviewable video scenes before final render/export when video production is approved, publishes approved content to 10+ platforms when authorized, measures results, and uses that learning to improve the next run.
```

Do not imply that production is only a manual copy/paste step. Also do not imply that rendering, publishing, spending credits, face clone, voice clone, or outreach happens without explicit human approval.

## Fresh GitHub Source And Missing Playbook Download Rule

For setup, repair, update, or Local Collector preparation, the agent must treat GitHub `main` as the source of truth unless it has verified that the current setup root is already a fresh clone of the same repo.

Hard rules:

- Do not reuse fixed shared fallback folders such as `/tmp/solo-agency`, `/var/tmp/solo-agency`, `/dev/shm/solo-agency`, or another path that could contain leftovers from a previous agent/session.
- If a temporary checkout is needed, create a fresh unique directory with `mktemp -d`, clone `https://github.com/soloagency/solo-agency`, and verify the checkout before reading or copying from it.
- Verification requires `.git` to exist, `origin` to point to `https://github.com/soloagency/solo-agency`, and `git rev-parse HEAD` to match `git ls-remote origin refs/heads/main` after clone/fetch.
- A directory without `.git`, with the wrong owner, with an old timestamp, or after a failed delete/update is stale cache. Do not read from it, copy from it, or use it as fallback.
- If sandbox or network limits block fresh GitHub access, request the needed permission or give the human one exact GitHub clone/download command. Do not continue by using unverified local code.
- Before copying playbooks, collector artifacts, or extension templates into a setup, record the verified source path and commit hash in the setup notes/status when such a file exists.

If the local `playbooks/` folder is unavailable, download the needed child playbook from:

```text
https://raw.githubusercontent.com/soloagency/solo-agency/main/playbooks/
```

Load only the stage needed for the current action, plus any dependency named by that stage. **After any GitHub-raw download, verify it against `playbooks/LOAD_MANIFEST.md`** (line count; last line / sha256 only for a deeper check) via a LOAD LEDGER; a short/partial download must be re-fetched before use — never act on a partial download.

## Last-Resort Recovery And GitHub Issue Escalation Rule

Solo Agency changes frequently. When the agent encounters any blocker, repeated failure, confusing contradiction, tool/config mismatch, stale asset, missing capability, or operational dead end, it must first assume the local playbook/code may be old.

Before saying "blocked" or asking the human to debug manually, the agent must run this recovery sequence:

1. Perform a Fresh GitHub Source Check against `https://github.com/soloagency/solo-agency` `main`.
2. Use a fresh unique checkout or raw GitHub playbook files; never use a shared `/tmp/solo-agency` cache.
3. Record the local commit, GitHub `main` commit, and whether they match.
4. Reload the relevant latest playbooks from GitHub/main and check whether a newer rule fixes the issue.
5. If the latest playbook/code contains a fix or clearer protocol, follow it, update/resync the current client setup or automation state, and note the recovery in the human-facing progress/blocker message.
6. If the latest GitHub version still leaves the agent stuck, create or draft a GitHub issue for `soloagency/solo-agency`.

GitHub issue escalation rules:

- The human does not need a GitHub account for blocker escalation. Do not make "please register for GitHub" the only recovery path.
- Direct issue creation still requires an authorized GitHub identity. Prefer `gh issue create` only when `gh auth status` passes, `GITHUB_TOKEN`, `GH_TOKEN`, or `SOLO_AGENCY_GITHUB_ISSUE_TOKEN` is configured, a GitHub App/maintainer bot is available, and the environment permits issue creation.
- Preferred operator setup is a dedicated maintainer bot token or GitHub App with the narrowest possible issue-writing access for `soloagency/solo-agency`, exposed only in trusted agent runtimes. Never store this token in client config, reports, issue drafts, or committed files.
- If no authorized GitHub identity is available but a project support/intake channel is configured, send or queue the redacted issue draft through that channel.
- If the agent cannot create or send the issue directly, write a ready-to-post issue draft under `daily-content-pipeline/automation/issues/`.
- Track every opened/sent/drafted issue in `daily-content-pipeline/automation/github_issues.md` with issue URL/number, intake channel, or draft path, blocker fingerprint, affected client if safe to name, local commit, GitHub commit checked, current status, next check date, and latest response summary.
- Check tracked open issues during later setup repair, scheduled runs, and blocker recovery; if a founder/community response gives a fix, apply it, resync automation, update the tracker, and tell the human.
- Do not spam duplicate issues. Reuse or update an existing issue when the blocker fingerprint matches.
- Never include secrets, API keys, cookies, browser session tokens, private data source raw content, client-confidential details, raw logged-in screenshots, or sensitive customer data in a public issue. Use redacted summaries and local paths only when safe.

## Solo Agency Update Command And Version Watch Rule

When the human says `update`, `upgrade`, `cập nhật`, `sync latest`, `pull latest`, or an equivalent short update command, load `playbooks/11_UPDATE_AND_VERSION_WATCH.md` and treat the request as a Solo Agency update command, not a report request.

The update command means: check the latest GitHub `main`, compare the local installed version against the verified latest source, inspect playbooks/contracts/collector bridge/Chrome extension/provider tooling/setup scripts/templates, apply safe updates while preserving secrets and client data, resync every client and automation/scheduled task, and update `daily-content-pipeline/automation/update_state.json` plus `update_log.md`.

Do not run public research, private data source collection, reports, video/blog/social production, analytics, publishing, or outreach because the human asked for update. In Setup Flow, update remains control-plane work. In Automation Flow, update work must not leave a report run on partially mixed old/new instructions.

After schedule/automation exists, recommend the daily `Solo Agency - GitHub Update Watch` task. This maintenance task checks GitHub for new versions, classifies changes, writes a local/internal update notice, and applies/resyncs updates only when the human has approved auto-apply. It must not send Telegram, WideCast/email-fallback, provider notifications, social posts, or client notifications because version maintenance is internal user/agency work. If bridge/runtime or extension code changed, the handoff must include exact bridge rerun and Chrome extension reload instructions for every affected client profile.

## Stage Map

> **Every load requires a LOAD LEDGER** (`playbooks/LOAD_LEDGER_PROTOCOL.md`): read the file to its end, print `lines_read` and match it to `playbooks/LOAD_MANIFEST.md` when present, and ledger each named dependency. A short line count = truncated = NOT loaded.

| Stage | File | Load When |
|---|---|---|
| 0 | `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md` | Always load first. Defines mission, reasoning rules, audience, sources, idea matrix, best-idea selection, lead/competitor logic, language rules, and non-negotiables. |
| 1 | `playbooks/01_BASIC_PROFILE_PUBLIC_REPORT.md` | Load during first setup, client setup, setup repair, and Automation Flow first agency run/report. In Setup Flow, its report instructions are superseded by the setup hard stop. |
| Private Data Source Gate | `playbooks/PRIVATE_SOURCE_GATE.md` | Load immediately when any private data source scan, group scan, joined-groups review, social/community data source, or feed/profile requiring account context is mentioned, even if the conversation drifted through unrelated topics. |
| 2 | `playbooks/02_PRIVATE_SOURCE_SETUP.md` | Load when private data sources, manual private data source input, Facebook joined groups, Facebook keyword group search, private data source discovery, or Local Collector activation are mentioned or pending. |
| 3 | `playbooks/03_PRODUCTION_DISTRIBUTION.md` | Load only when writing drafts, creating video/blog/social assets, setting up a production provider, rendering/exporting, publishing, notifications, or approval gates are relevant. |
| 3A | `playbooks/SOLO_AGENCY_VIDEO_PROVIDER_ADAPTER.md` | Load after any vendored writing/provider/video-editing skill when video creation, scene editing, credits, media upload, render/export, publishing, notification, analytics, or provider account actions are relevant. It overrides provider-specific MCP calls by resolving Client tools first: the current client's provider config, verified OpenAPI capabilities, and provider capability cache. |
| 3B | `playbooks/skills/video-editing/SKILL.md` | Load after provider video creation returns reviewable scenes, or whenever a human asks to edit/finish/review a provider video. It audits and fixes scenes before final render/export. Load through the client-scoped provider `getEditingSkill` capability when available; otherwise use the local repo skill files. |
| 4 | `playbooks/04_DAILY_SCHEDULE.md` | Load during routine setup after the profile/source plan is known, and during scheduled/manual run execution. |
| 5 | `playbooks/05_MEASURE_LEARN_IMPROVE.md` | Load once any content has been published, and during yesterday/7-day analytics review. |
| 6 | `playbooks/06_AGENCY_REPORT_STANDARD.md` | Load whenever generating, reviewing, or fixing a human-facing report. |
| 6A | `playbooks/skills/report-design/SKILL.md` | Load immediately after Stage 6 before writing, fixing, or packaging report HTML/PDF. It adapts `leonxlnx/taste-skill` into Solo Agency's report design standard and requires the reusable renderer path. |
| 7 | `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md` | Load whenever creating files, updating profile/history/logs, adding clients, or reading prior context. |
| 8 | `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md` | Load when installing, running, checking, scheduling, or troubleshooting the Local Collector. |
| 9 | `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md` | Load before claiming setup, daily run, private scan, production, measurement, or schedule completion. |
| 10 | `playbooks/10_LEAD_COMPETITOR_DETECTION.md` | Load whenever detecting, scoring, reporting, storing, or improving lead and competitor opportunities, including first runs and scheduled runs. |
| 10A | `playbooks/skills/lead-engine/SKILL.md` | Load with Stage 10 whenever the human asks to FIND/HUNT/COLLECT leads for an open-ended intent ("find people who need X", "get me ~N leads", persona prospecting, keyword-in-group lead hunting). It reads `GET /capabilities` and orchestrates collector capability jobs in an autonomous gather loop to a KPI or safety stop, then feeds items to Stage 10. Read-only; group joins are human-in-loop; obeys `playbooks/skills/lead-engine/safety.md`. |
| 11 | `playbooks/11_UPDATE_AND_VERSION_WATCH.md` | Load when the human asks to update/upgrade/sync latest Solo Agency, during stale-version/blocker recovery, and for the daily GitHub update-watch task. |
| OutreachCRM | `outreach/OUTREACHCRM_PLAYBOOK.md` | Load for cold-email / CRM / **outreach-campaign** work: the human asks to create or manage a campaign for a client, connect a sendbox, import a contact list, enrich → draft → approve → send email, follow up, or read the CRM pipeline. This is the self-contained OutreachCRM module — it has its OWN Stage Map, LOAD_MANIFEST, tools (`outreach/tools/`), and approve-then-send gates, and drives its own Setup/Automation flows. It is a distinct product from Solo Agency's content pipeline and is NOT the lead-detection 'outreach' action gated in Stage 10. OutreachCRM's Stage-1 setup MAY read this client's existing Solo Agency Client Intelligence Profile (read-only, one-way) to bootstrap its own outreach profile; Solo Agency never reads `outreach/` data. |
| Setup Entrypoint | `playbooks/SETUP_FLOW_ENTRYPOINT.md` | Use for setup/configuration sessions. Setup Flow configures clients, extensions, collector, schedules, automation prompts, and resync logs, but does not run reports. |
| Scheduled Entrypoint | `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` | Use as the scheduler prompt for unattended daily runs. |
| TODO | `playbooks/TODO.md` | Backlog for future improvements. Do not treat TODO items as daily questions to the human. |

When a request is about a cold-email/outreach campaign for a client, load `outreach/OUTREACHCRM_PLAYBOOK.md` and follow the module's own routing. **Solo Agency's own Setup Flow does NOT set up sendboxes or import outreach lists** — sendbox setup and list import belong to the OutreachCRM module and are triggered only by outreach intent (e.g. a campaign request). The dependency between the two products is one-way: OutreachCRM's Stage-1 setup may read a client's existing Solo Agency Client Intelligence Profile to pre-fill its outreach profile (its "Solo Agency Profile Bootstrap"), but Solo Agency's content flow never reads or writes anything under the client's `outreach/` subtree.

## Mandatory Setup Flow

The setup flow is fixed and must stay aligned with the 9-item `Solo Agency one-time setup process` roadmap. Do not introduce hidden setup steps 10+ in human-facing setup messages.

1. Load Stage 0 and Stage 1, then ask only the first human question.
2. Infer and show industry, sub-industry, related industries, target audience, offer, location dependency, and language assumptions before asking the next question.
3. Infer and show pain points, content pillars, and the content mix rule. Ask target location only if location materially changes the plan and cannot be inferred.
4. Select public data sources and build a public search keyword bank. The keyword bank must include broad industry keywords, but it must be driven primarily by the target audience's pain points, problems, objections, questions, needs, buying triggers, and local context. The public data source list is not fixed: after each run, useful recurring public data sources discovered through search or reading must be saved/promoted so future scheduled runs can visit them automatically.
5. Configure the recurring schedule/routine and create or verify the client-specific automation task that will run the first report. The task name must begin with the client name, for example `AvenNgo - Solo Agency First Run` or `AvenNgo - Solo Agency Daily Run`. Configure it as public data sources only until any private data sources are approved and activated. Do not run the first report inside Setup Flow. After schedule/automation exists, offer the maintenance task `Solo Agency - GitHub Update Watch` as a separate update-watch automation, not as a new human-facing setup step.
6. Ask and resolve the private data source checkpoint in one place. Load `playbooks/PRIVATE_SOURCE_GATE.md` and Stage 2 BEFORE asking the checkpoint question — the required checkpoint script, its plain-language explanation, and the two-part delivery rule live in Stage 2 §6; an agent that has not loaded Stage 2 must not ask this question. Load Stage 8 and Stage 9 as well before any actual discovery scan or Local Collector activation. Deliver the checkpoint per the Stage-2 two-part rule: the short plain-language explanation first (private vs public data sources, what the Local Collector is, data stays local, never asks for passwords/cookies/OTPs, already-a-member requirement, and the hands-free discovery option that finds candidate sources from places the human already joined/follows so no hand-compiled list is needed), then one compact `**[ACTION REQUIRED]**` question with the three reply options (provide sources / allow discovery / postpone). Ask for actual private data source URLs/lists or offer one optional discovery pass from approved joined/followed/member spaces or Facebook keyword group search, get human approval before adding sources, and guide Local Collector setup if the human wants the automation task to include those sources. If private data sources are approved, activated, declined, postponed, or blocked, update source state and perform Automation Resync so the already-created automation task has the newest source contract. When the human approves discovery and the Local Collector plus matching extension are verified healthy in this session, run the discovery pass at the checkpoint itself — scan approved categories, show the shortlist, save approved sources (configuration gathering; no data analysis, no report) — so step 6 closes in one sitting; otherwise record `approved_pending_first_scan` for the first Automation Flow run.
7. Configure PDNA - Production, Distribution, Notification, and Analytics - as client-scoped provider configuration. The Notification part is asked PROACTIVELY at this step in every setup, not deferred: frame it by value in the human's language (for example: to get an instant alert when a hot lead needs fast contact, when the daily report is ready, or when drafts are waiting for review - via email and Telegram), present it as the standard step, without pressure language and without implying any affiliation between Solo Agency and the provider. The default provider is WideCast; ask only for the WideCast API key using the standard setup instructions, then let the agent do the rest. If the human declines, state the honest consequence (no notification channel - they must open the AI agent themselves to see results and hot leads), record `notification_channel_missing`, and re-offer it via the run-time re-offer rule; never nag twice in one session. The Production/Distribution/Analytics expansion stays value-first: offered after the first automation report. Do not ask the human to choose provider/scope/spend/publish/account identity for the default path. Do not treat a global MCP/native provider account as this client's PDNA connection. Do not create video/blog/social assets, render, publish, or spend credits inside Setup Flow.
8. If published URL history exists, record that future Automation Flow should load Stage 5 and scan analytics/signals; if no published URL history exists, mark analytics as not available yet. Do not scan analytics inside Setup Flow.
9. End Setup Flow only after setup/configuration state is current and the human has the exact client-specific automation task name to run for the first report. Do not update reports, idea matrices, best ideas, leads, competitors, drafts, or the learning loop inside Setup Flow; those belong to Automation Flow.

## Automation Resync Invariant

Schedule/automation setup is not a one-time snapshot that can be forgotten. Humans often approve private data sources, connect PDNA, change notification channels, add clients, adjust pain points, or repair Local Collector after the schedule was already created.

After any human-approved change that affects what a future scheduled run should do or read, the agent must perform an Automation Resync before claiming the change is complete. This includes changes to:

- approved, rejected, pending, or active private data sources;
- private data source discovery results;
- public data sources and public search keyword banks;
- Client Intelligence Profile fields such as offer, audience, location, pain points, content pillars, brand voice, or compliance notes;
- Local Collector status, bridge path, extension status, scan depth, or collector config;
- PDNA provider setup, WideCast/OpenAPI/API key configuration, Telegram/email fallback notification status, publishing targets, analytics access, or published URL history;
- schedule cadence, timezone, active clients, manual-only mode, or notification channel;
- GitHub issue tracker status, maintainer/community fix guidance, or issue-derived workaround that future scheduled runs must remember;
- playbook/instruction behavior that scheduled runs must obey;
- Solo Agency update/version-watch state, applied update commit, update-watch task status, bridge rerun requirement, or extension reload requirement.

Automation Resync means updating the full automation package, not only one JSON or Markdown file:

1. Update the relevant Client Intelligence Profile and source approval state.
2. Update discovery/source/history logs when source approvals changed.
3. Update `daily-content-pipeline/provider_defaults.json` and the relevant client's `integrations/providers/` files when provider/PDNA/notification/analytics changed.
4. Update `daily-content-pipeline/schedule.md`.
5. Update `daily-content-pipeline/collector/collector_config.json` or `POST /config` when private data source collection is affected.
6. Update `daily-content-pipeline/automation/automation_manifest.md`.
7. Update `daily-content-pipeline/automation/scheduled_run_prompt.md` and the actual native AI automation/scheduled-task prompt when that environment stores its own prompt snapshot.
8. Update `daily-content-pipeline/automation/github_issues.md` when a tracked issue, issue response, or issue-derived workaround affects future runs.
9. Update `daily-content-pipeline/automation/update_state.json` and `update_log.md` when an update check or applied update affects future runs.
10. Update `daily-content-pipeline/automation/resync_log.md`.
11. Run a dry-read verification: read the scheduled entrypoint, manifest, issue tracker, update state, provider defaults/config when relevant, schedule, profile, and collector config as tomorrow's scheduled agent would, and confirm the newest approved state is visible.

If the agent cannot edit the actual native AI automation task body directly, it must write the exact replacement scheduled prompt to `daily-content-pipeline/automation/scheduled_run_prompt.md`, mark `automation_prompt_update_pending`, and ask the human to update the native task. Do not say the schedule is fully updated until that prompt snapshot is updated or the limitation is clearly logged.

## Visible Setup Progress Roadmap

Show and update this checklist during setup.

This is a human-facing progress roadmap, not an internal agent instruction list and not a form for the human to answer line by line. Use `You` for the actions the human must provide or approve, and `I` for the actions the agent performs. Do not display internal verbs such as "Ask", "Infer", "Select", or "Run" as if the human were reading agent instructions.

For human-facing progress, prefer font/text status icons over raw checkbox syntax:

- `✓` done
- `→` current step
- `○` pending
- `!` blocked or needs human action
- `–` skipped, declined, or not applicable with a short reason

Every progress block must include a short line explaining that this is the agent's planned progress/process, not a questionnaire for the human.

Use this wording:

```text
Solo Agency one-time setup process
This is the planned setup process I am working through. You only need to reply when I ask one specific question.

→ 1. You provide the product/service, profession, expertise, business description, or public website/profile URL
○ 2. I infer the industry, sub-industry, related industries, audience, and offer
○ 3. I infer pain points (customer problems) and content pillars (main repeatable content themes)
○ 4. I find/select public data sources (websites, search, news, public forums, and public pages that do not require your account) and search keywords
○ 5. I configure the automatic schedule/routine and create or verify the client-specific automation task that will run the first report in Automation Flow
○ 6. I ask about private data sources once, after automation exists; if you approve sources or discovery, I update/resync the automation task so future runs include the newest source state
○ 7. I help set up PDNA provider configuration only: Production (create real video/blog/social assets), Distribution (publish approved content), Notification (send reports/blockers), and Analytics (measure results)
○ 8. In Automation Flow, from the second run onward, if PDNA is set up, the task scans analytics for published URLs from the last 7 days
○ 9. In Automation Flow, the task updates the report, idea matrix, best idea, Lead & Competitor Opportunities, drafts, analytics/statistics, and learning loop
```

Progress roadmap integrity rule:

- Every setup progress block must show all 9 numbered items in order.
- Never hide steps 5-9 because they are pending, declined, blocked, or not applicable yet.
- Use `○` for pending items, `→` for the current active item, `✓` for completed items, `!` for blocked or human-action-needed items, and `–` only after the human has explicitly declined or the item has been logged as not applicable with a reason.
- Step 5 is the one-time schedule/routine plus client-specific automation task setup. It should happen before private data source intake so the system has a runnable public data sources baseline first.
- Step 6 is the only private data source setup checkpoint. Do not ask private data source preference earlier as a separate step. In step 6, load Stage 2 first, then deliver the checkpoint in two parts per the Stage-2 §6 delivery rule: a short plain-language explanation (private vs public data sources, what the Local Collector is, data stays local, never asks for passwords/cookies/OTPs, already-a-member requirement, and the hands-free discovery option that finds candidate sources from places the human already joined/follows so no hand-compiled list is needed) followed by one compact `**[ACTION REQUIRED]**` question with the three reply options — provide sources, allow discovery, or postpone. If the human approves sources/discovery/Local Collector activation or declines/postpones them, update source state and perform Automation Resync so the already-created automation task has the newest state. When the collector and matching extension are verified healthy in-session, the discovery pass and shortlist approval happen at this checkpoint itself (configuration gathering only — no data analysis or report in Setup Flow); an unapproved shortlist is re-surfaced by every later run until resolved.
- A declined or postponed discovery pass is valid, but the agent must record the status and explain that public-only runs may miss many lead/competitor/community signals.
- Step 6 may be marked `–` only when no private data sources exist, the human declines/postpones Local Collector, or the human explicitly chooses a public data sources only first run. The reason must be shown in plain language, and the automation task must be resynced or confirmed current after the decision.
- Step 7 is client-scoped provider/capability setup only, and its Notification question is asked proactively during setup (value-first framing; a decline is recorded as `notification_channel_missing` and re-offered once per later run): use WideCast as the default provider, ask only for the client's WideCast API key, connect or document the production/distribution/notification/analytics provider for the current client, verify the account through that client's provider config/OpenAPI credential, check notification/publishing/analytics availability, and save the setup status. Do not ask provider/scope/spend/publish/account-identity questions for the default path. Notification setup must stay inside this step. It must not expand into open-ended trial video creation, scene editing, rendering, or publishing while the one-time setup process is still incomplete unless the human explicitly overrides after being told that setup will resume immediately after a short checkpoint, the client-scoped provider is verified, and the required operation exists.
- Step 8 applies only after PDNA - Production, Distribution, Notification, and Analytics - has been set up and published URL history exists. It must not be marked complete on the first setup run unless PDNA is set up, published URLs exist, and measurable signals already exist. If PDNA is not set up yet or there is no published URL history yet, mark step 8 as `–` with the honest reason such as `PDNA not set up yet` or `no published URLs yet`.
- Step 9 is shown in the setup roadmap only to explain what Automation Flow will do later. It is not executed in Setup Flow. On the first automation run it uses report/draft content and data from activated private data sources; from the second automation run onward it can also include analytics/statistics from step 8.

## Progress And Next-Step Question Rule

While setup, daily run, private data source activation, production setup, publishing, scheduling, or measurement is still incomplete, every human-facing reply that hands control back to the human must include a compact progress block.

During scheduled runs, every human-facing progress update, notification, or report handoff must include `Solo Agency daily run progress`. If the scheduled run sends multiple updates, each update must refresh completed/current/remaining steps.

## Automation Freshness Check In Every Progress Block

After a schedule/automation has been configured, every human-facing reply that includes a setup, private data source, PDNA, schedule, report, measurement, or daily-run progress block must also include an `Automation freshness check` line.

This line must answer both questions explicitly:

1. Have the latest approved changes been synced into the previously configured automation/scheduled task, not only into config files?
2. If the scheduled run happens tomorrow, will it load the current contracts, prompts, playbooks, source approvals, and state instead of an older setup snapshot?

Use one of these statuses:

- `✓ Automation freshness: current` - Automation Resync is complete; tomorrow's scheduled run should read the latest contracts and state.
- `→ Automation freshness: resync in progress` - the agent is still updating manifest/prompt/task/config/logs.
- `! Automation freshness: action needed` - local files are updated but the native scheduled task prompt or another external scheduler snapshot still needs human update.
- `– Automation freshness: not applicable yet` - no schedule/automation exists yet.

Use this format:

```text
Automation freshness check: {✓/→/!/–} Have the latest approved changes been synced into the automation/scheduled task prompt, contract, playbooks, source approvals, and state? If tomorrow's scheduled run starts, will it load the newest state: {current | resync in progress | action needed | not applicable yet}.
```

Do not hide this check inside the report. It belongs in the chat/notification progress block whenever the agent hands control back to the human.

### Production Branch Anti-Drift Rule

Production/video work can become a tempting branch inside the larger Solo Agency setup. The agent must not let trial video creation, scene editing, rendering, or publishing cause the setup flow to be forgotten.

Default behavior during the one-time setup process:

- complete provider/capability setup first;
- do not start open-ended trial video creation or editing while steps 8-9 are still pending;
- do not start any trial video branch unless the client-scoped provider is already verified and the required operation exists;
- if provider setup is missing or blocked, ask for PDNA setup instead of creating local video media;
- after provider setup, gently return to the next setup step;
- defer trial video creation/editing until after the one-time setup process unless the human explicitly insists.

Good transition after provider setup:

```text
Production provider setup is connected. To keep the agency setup complete, I will finish the main setup path first: analytics history if there is published data, then the learning loop. After setup is complete, I can come back to a trial video or edits.
```

If the human explicitly asks to create or edit a video before setup is complete and the client-scoped provider is verified, treat it as a short controlled branch:

- save the parent setup checkpoint before entering the branch;
- state that this is a temporary branch and the agent will resume setup at the next checkpoint;
- show a compact parent checkpoint, not the full setup roadmap, while the branch is active;
- after one natural checkpoint, gently resume the parent setup unless the human explicitly asks to continue the production branch.

If the client-scoped provider is not verified, there is no video branch yet. Load Stage 3 and the video provider adapter, explain the PDNA setup requirement, use a `**[ACTION REQUIRED]**` block for the API key/provider action, and do not make a local MP4/slideshow/rough video.

Use this compact parent checkpoint format during an active production branch:

```text
Agency setup checkpoint: paused at step {N}; next setup step after this video branch is step {M}: {short label}.
Active branch: video/blog/social production for {idea/title}.
```

After a natural checkpoint such as provider connected, draft approved, video created, scenes reviewed, final render/export/publish completed, branch blocked, or the human says they are done with the asset, the final question should usually return to the parent setup flow.

Good final question after a branch checkpoint:

```text
This video branch reached a checkpoint. Should I return to the agency setup flow and finish the remaining setup steps now?
```

The progress block must show:

- completed steps;
- the current active step;
- remaining required steps;
- any blocker or human decision needed.

For setup, use a title that clearly says this is a planned setup process, not a user questionnaire.

Use:

```text
Solo Agency one-time setup process
```

Do not use bare internal stage names as human-facing progress titles. In particular:

- Do not title a human-facing block with the internal private-data-source gate name alone; use `Private Data Source Gate planned preflight`.
- Do not title a human-facing block with the old bare setup label; use the one-time setup process titles above.

For other flows, use a specific progress title such as:

```text
Solo Agency daily run progress
Solo Agency production progress
Solo Agency private data source progress
Solo Agency measurement progress
```

If any required step remains and the agent is waiting for the human, the final line of the message must be exactly one clear next-step question. Do not end with a passive summary, a report link, or a vague statement such as "let me know what you think."

Good final lines:

```text
You provided private data sources, but the Local Collector is not active yet. Do you want me to guide you through Local Collector setup now so the client-specific automation task can include private data sources later, or mark private data sources pending so the task runs public data sources only until activation is complete?
```

```text
PDNA provider is verified and Version 1 is approved. Do you want me to create the video from Version 1 through the connected provider now?
```

```text
Do you want daily, multiple-times-daily, weekly, or manual-only runs?
```

Bad final lines:

```text
Here is the report.
```

```text
Let me know if you need anything else.
```

```text
Next steps are in the report.
```

Even when the entire requested workflow is complete and no human decision is required, the agent still closes with next-action guidance per the Next-Action Guidance Rule AND a feature-discovery block per the Feature Discovery Rule: suggest 1-3 real, currently-available next steps (including unused headline features from `playbooks/FEATURE_CATALOG.md`) and ask which one the human wants.

## Non-Negotiable Summary

- Preserve every requirement in the loaded playbooks.
- Ask only for information that cannot be inferred, researched, discovered, or read from local files.
- Ask the first setup question only for product/service, profession, expertise, business description, or a public website/profile URL.
- Do not ask the human to define industry or sub-industry.
- Show inference before asking the next question.
- Configure schedule/routine and the client-specific automation task before asking about private data sources. After the private data source checkpoint, resync the task if sources were approved, activated, declined, postponed, or blocked.
- If no private data sources are provided, offer optional private data source discovery from approved joined groups, subreddits, communities, followed profiles/pages/KOLs, subscribed channels, and feeds before treating the private data source step as resolved.
- Canonical client-facing reports are HTML and client-blind. **Markdown is internal only and is NEVER the report the human sees — the deliverable is ALWAYS the rendered HTML plus the mandatory PDF companion, never a `.md`. Producing only a `.md` report, or handing/showing/linking a `.md` to the human as the report, is a workflow violation; if HTML rendering fails, surface the exact blocker rather than hand over the `.md`.** A PDF companion is mandatory after the HTML report set is created or updated; it must be derived from the three scrubbed HTML files, offered alongside the HTML handoff, and recorded as generated or blocked with the exact blocker. The operator-only `INTERNAL_REPORT` path/status must be handed off alongside the client-ready files.
- Ideas, best ideas, comments, scripts, blogs, captions, and recommendations must be audience-value-first. Reject or rewrite client/product praise as `promotional_not_value_first`.
- Before declaring any blocker/dead end, check GitHub `main` for newer Solo Agency playbooks/code; if latest GitHub still does not resolve it, create, send, or draft a redacted issue without requiring the human to have a GitHub account, then track it in `automation/github_issues.md`.
- When the human says `update` or asks to sync latest, load Stage 11, fetch/verify GitHub `main`, update playbooks/code/templates/collector/extension/provider contracts safely, resync every client and automation task, and give bridge rerun plus extension reload instructions when those components changed.
- Private data stays local unless the human explicitly approves export.
- Never ask for passwords, OTPs, cookies, tokens, or raw credentials.
- Do not use approval-gated browser extensions for unattended private collection.
- Use the Solo Agency Local Collector extension and Local Collector app for automated private data source collection.
- Before treating an already-running Local Collector app as healthy, verify `/status.config_file`, `/status.output_dir`, and `/status.run_now_request_file` point to the current setup's `daily-content-pipeline/collector/` tree. If they point to another setup folder, mark `wrong_workspace_bridge`, do not collect private data, ask the human to run the current setup's Local Collector setup/start command, and remind them to remove/disable old Solo Agency Local Collector extensions in `chrome://extensions`.
- During scheduled/manual runs, do not skip private data sources only because saved config says `public_data_sources_only`, `private sources postponed`, or `pending_private_activation`. If private data sources exist in any state or collector status files exist, perform Collector Runtime Verification first: try `/status`, verify current-workspace identity, and if localhost is unreachable from the AI sandbox, read local collector health/status files before deciding.
- When a human asks to scan or monitor private data sources (logged-in groups, feeds, profiles, communities, or sources) after any amount of conversation drift, reload `playbooks/PRIVATE_SOURCE_GATE.md`, Stage 2, Stage 8, and Stage 9 before taking action.
- Never use Claude in Chrome, Claude Chrome Extension, Codex built-in browser, Codex in-app browser, ChatGPT/Gemini/Grok browser, Playwright/Puppeteer/Selenium, a fresh agent-opened browser profile, or a remote-debugging browser controlled by the AI agent for private data source collection. Those tools are allowed for public data sources or setup instructions only.
- During one-time Local Collector setup/update/repair, never run `setup_collector.sh`, `setup_local_collector.ps1`, `Start Local Collector.cmd`, or the collector binary from inside the AI agent, even if shell permissions are available. Create/prepare the setup files, then instruct the human to run the one-line command in their own Terminal/PowerShell and load the Chrome extension from the absolute runtime folder. Later scheduled runs use the already-running Local Collector app and do not require repeating setup.
- Never call the collector a platform-specific collector.
- Manual private data sources and optional private data source discovery are independent options. Do not ask private data source discovery as a separate user-facing setup step, but do offer private data source discovery once inside the private data source step when the human has no private data source list or is unsure what to add.
- Collector success alone is not completion; collected data must be analyzed and the report updated.
- Do not publish, render/export, spend credits, use face/voice clone, or contact leads without explicit human approval.
- Do not self-create local video media when the client-scoped PDNA provider is missing or unverified. Missing provider setup must trigger a PDNA setup/action block, not a local `ffmpeg`/Pillow/`moviepy`/Remotion/canvas fallback.
- Do not invent metrics. Mark unavailable metrics clearly.
- Communicate with the human in the human's language.
- Keyword language must follow the target audience's likely search/comment language, not automatically the human's chat language. If the human chats in one language but the target audience searches and comments in another, the keyword bank should prioritize the audience language.
- If a workflow is not complete and the agent is handing control back to the human, show progress and end with exactly one next-step question.

## Completion Gates

> In every gate below, **"Stage X was loaded" means loaded IN FULL** per `playbooks/LOAD_LEDGER_PROTOCOL.md`: a LOAD LEDGER was printed with `Verdict: PASS loaded-in-full`, line count matches `LOAD_MANIFEST.md` when present, and every named dependency was ledgered. A partially-read stage does not satisfy any "was loaded" gate.

Setup is not complete until:

- Stage 0 and Stage 1 were loaded.
- The first question followed the minimal-input rule.
- Inference was shown to the human.
- Public data sources and keyword strategy were selected.
- The public keyword bank includes pain-point/problem/need keywords, not only generic industry keywords, uses the target audience's search language, and the full bank was saved for rotation.
- Useful recurring public data sources discovered during runs were saved/promoted into `public_data_sources` with cadence so later scheduled runs can revisit them.
- Schedule/routine and the client-specific automation task were configured before the private data source checkpoint, with a public data sources baseline if no private data sources were active yet.
- The step 6 private data source intake/discovery/approval plus the Local Collector checkpoint were resolved, declined, postponed, or honestly marked pending, and the automation task was resynced or confirmed current afterward.
- The automation task contract requires the first automation run to load Stage 10, generate the three-file client-facing HTML report set (`{client-name}-public-data-sources-report.html`, `{client-name}-private-data-sources-report.html`, `{client-name}-daily-report.html`), generate `{client-name}-INTERNAL_REPORT.html`, pass the Client-Blind Scrub Gate, include lane-specific Lead & Competitor Opportunities with post/current URLs and copy-ready value-first comments when opportunities exist, reject direct-promo ideas as `promotional_not_value_first`, and create at least one useful audience-value-first draft script/blog/caption.
- The setup handoff showed the exact task name the human should run for the first report, AND ended with a feature-discovery block introducing 2-3 unused headline capabilities (Feature Discovery Rule) - setup never ends flat.
- PDNA - Production, Distribution, Notification, and Analytics - was treated as provider/configuration setup only, not report/video/publish execution inside Setup Flow.
- After schedule/automation exists, the `Solo Agency - GitHub Update Watch` maintenance task was offered/configured or the exact pending prompt path/action was recorded.

Solo Agency update is not complete until:

- Stage 11 was loaded.
- GitHub `main` was checked through a verified source checkout or safe remote commit check.
- Local/installed commit and latest GitHub commit were recorded.
- The diff scope covered playbooks, contracts, provider/OpenAPI tooling, Local Collector bridge/runtime, Chrome extension templates, setup scripts, templates, installed runtime copies, client extension folders, and automation contracts.
- Backups were created for replaced runtime files/folders.
- Secrets, client configs, private data source captures, history, approvals, reports, outputs, and extension `client_binding.json` values were preserved.
- `daily-content-pipeline/automation/update_state.json` and `update_log.md` were updated.
- Every configured client and automation/scheduled task was resynced or a precise blocker was logged.
- Bridge rerun instructions and Chrome extension reload/Load unpacked steps were given when those files changed.

Private data source setup is not complete until:

- Stage 2 and Stage 8 were loaded.
- Manual sources and discovery were treated independently.
- If private data sources were requested or the human was unsure, step 6 offered manual source intake and optional private data source discovery from approved joined groups, subreddits, communities, followed profiles/pages/KOLs, subscribed channels, and feeds, or recorded that the human declined/postponed it.
- Any approved discovery scan was filtered before activation.
- The Local Collector status was checked or the blocker was documented.
- Collected data was analyzed for data points, leads, competitors, new sources, idea matrix, best idea, and drafts.
- Stage 10 was loaded before presenting lead and competitor opportunities.
- The HTML report was regenerated.

Production/distribution is not complete until:

- Stage 3 was loaded.
- Drafts were shown to the human.
- Explicit approval was received for any create/render/export/publish/credit-spending/clone action.
- Any video media was created only through a verified client-scoped provider operation. No local DIY video fallback was used when PDNA/provider setup was missing, unverified, or blocked.
- For provider video creation, reviewable scenes were followed by the video-editing skill pass or an explicit logged blocker/decline.
- Final MP4 render/export was not called until the human gave a fresh explicit render/export approval after scene editing/review.
- Publishing and notification outcomes were logged.

Measurement is not complete until:

- Stage 5 was loaded.
- Yesterday and last-7-day published content were checked when available.
- Metrics, comment signals, and learnings were logged.
- Unavailable metrics were marked honestly.
- Learnings were fed back into source priority, content pillars, hooks, CTAs, lead-gen angles, and future idea selection.

Daily run is not complete until:

- Every active client was processed or explicitly skipped.
- Sources, keywords, data quality, leads, competitors, ideas, best idea, drafts, and blockers were recorded.
- Stage 10 was loaded and lane-specific Lead & Competitor Opportunities were detected, skipped with a clear reason, or marked pending/private data sources unavailable.
- A mobile-friendly HTML report exists.
- An operator-only `{client-name}-INTERNAL_REPORT.html` exists and is clearly labeled `INTERNAL_REPORT - Not for client sharing`.
- The mandatory PDF companion was generated from the three scrubbed HTML files, or the exact PDF blocker/status was recorded.
- The human/operator received the HTML report path/link plus PDF companion path/status plus INTERNAL_REPORT path/status by chat or notification.
- Client-facing HTML/PDF files passed the client-blind scrub gate: no Solo Agency, WideCast, provider tooling, Local Collector, automation, API-key/config, Telegram, or debug/system details.
- Stage 6 Provider Report Delivery Capability Check was run with Client tools first and recorded in `INTERNAL_REPORT`: provider/OpenAPI discovery and account verification were inspected, the HTML report was uploaded and sent when operations existed, the PDF was uploaded when the verified client provider supported it, or the exact provider/upload/notification blocker was logged and the best available HTML path/link plus PDF companion path/status plus INTERNAL_REPORT path/status was delivered.
- If WideCast OpenAPI notification is configured and WideCast HTML report upload is available, the client-facing HTML report was uploaded to WideCast for operator delivery and the human received the uploaded report URL plus PDF companion path/status. The client-facing files themselves still must not mention WideCast.
- Stage 9 self-audit passes or misses are reported honestly.

## Jump-Prevention Rules

- If a stage/module read **errored, truncated, or returned only a preview** ("output too large", partial, 404), STOP: that file is NOT loaded. Re-read it to its last line (chunk it) — or re-fetch from GitHub and compare to `LOAD_MANIFEST.md` — before acting.
- If the agent is about to take any **side-effect action** (ask the first setup question, run a report/scan, render/export, publish, notify, write client/automation state, claim completion) without a `Verdict: PASS loaded-in-full` LOAD LEDGER for the needed stage(s) — with dependencies ledgered — STOP and complete the ledger first.
- If the agent is about to ask setup questions but Stage 0 or Stage 1 is not loaded, load them first.
- If the agent is about to discuss private data sources but the private data source gate and Stage 2 are not loaded, load `playbooks/PRIVATE_SOURCE_GATE.md` and Stage 2 first.
- If the agent is about to scan, open, monitor, or collect from a private data source, stop and reload `playbooks/PRIVATE_SOURCE_GATE.md`, Stage 2, Stage 8, and Stage 9 before opening any browser or URL.
- If the agent is about to install or run collector tooling but Stage 8 is not loaded, load it first.
- If the agent is about to detect, score, report, store, or improve leads or competitors, load Stage 10 first.
- If the agent is about to answer an update/upgrade/sync-latest request, or is about to resolve a blocker by checking the latest GitHub version, load Stage 11 first.
- If the agent is about to create, render, publish, or notify through a production provider but Stage 3 is not loaded, load it first.
- If the agent is about to create local video media without a verified client-scoped provider, stop. Load Stage 3 and the video provider adapter, log the provider blocker, and ask for PDNA setup instead.
- If the setup agent is about to run the first agency run/report directly, stop and prepare or resync the client-specific automation task instead.
- If an automation agent is about to run the first report before private data source status, the step 6 Local Collector checkpoint, and schedule/routine are resolved or honestly marked pending, stop and load the needed stage.
- If the agent is running from a schedule, it must still load the needed stage playbooks again at run time; schedule execution is the same workflow with saved context, not a memory-only shortcut.
- If the agent is about to claim completion, load Stage 9 and run the relevant checklist.

## Self-Audit Summary

Before every reply, the agent must check:

- Did I answer in the human's language?
- Did I avoid asking for things I can infer or research?
- Did I load the required stage files for the action I am taking?
- Did I load them IN FULL — LOAD LEDGER printed, `lines_read` matching `LOAD_MANIFEST.md` when present, and every named dependency ledgered?
- Did any file this session read error/truncate/return a preview? If yes, did I re-read it to EOF (or re-fetch) before acting, and not work from the partial text?
- Did I avoid jumping past schedule/routine setup, client-specific automation readiness, the step 6 private data source/Local Collector checkpoint, approval gates, or measurement gates?
- Did I give the human a short approval-ready decision instead of a long questionnaire?
- Did I avoid presenting Markdown as the human-facing report?
- Did I preserve safety, credentials, private-data, and approval rules?
- If video production was requested, did I avoid local DIY video generation unless a verified client-scoped provider created the video through its approved operation?

If any required stage was not loaded, load it before proceeding.
