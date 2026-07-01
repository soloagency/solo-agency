# Solo Agency Playbook

Version: modular-router-1.0

This root playbook is the thin router for a daily AI marketing agency workflow. It tells the agent what to load next, what gates must never be skipped, and how to avoid jumping ahead.

Detailed protocols live in `playbooks/`. The root must stay small. Do not paste the full protocols back into this file.

## First Instruction To The Agent

Before asking any setup question, load:

1. `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`
2. `playbooks/01_BASIC_PROFILE_PUBLIC_REPORT.md`

Only after those two files are loaded may the agent ask the first setup question.

## First Human Question

Ask only:

```text
What product/service, profession, expertise, business description, or public website/profile URL should this pipeline focus on? If location matters, include the target location.
```

Do not ask for industry, sub-industry, target audience, pain points, content pillars, idea categories, public data sources, or private data sources in the first question. A public website/profile URL is acceptable as first setup input; the agent may read it for setup context when web access is available, but this is not an operational public data source scan or report run. Infer what can be inferred first. Step 5 may ask only a lightweight preference question about whether to include private data sources later. Step 7A is where the agent asks for actual private data source URLs/lists, offers discovery, gets approval, and handles Local Collector activation if needed.

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

Do not bury required questions or actions inside long paragraphs, progress roadmaps, report links, Markdown files, or status summaries. If a required question appears in body text, repeat the final required ask in the action block. If no human action is needed, say:

```text
No action required right now.
```

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
- Scheduled runs and notifications should use the block only when the human must act. Otherwise they should include `No action required right now.`

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

The report set must use `outputs/YYYY-MM/YYYY-MM-DD/{client-name}-report_state.json` so later automation passes can update only the intended lane. The `latest` human-facing link must point to `{client-name}-daily-report.html`, not a lane-specific report unless explicitly requested.

After a private data source scan completes, the agent must reconcile the whole report set before handoff: `{client-name}-private-data-sources-report.html`, `{client-name}-daily-report.html`, `{client-name}-daily-report.md` or source record, `{client-name}-report_state.json`, and `outputs/latest/` convenience copies must agree on private scan status, completed timestamps, sources attempted/completed, data point counts, lead counts, competitor counts, recommended-source counts, blocker counts, and notification/delivery status. Do not leave stale phrases such as `scan in progress`, `partial`, `pending`, or old recommended-source totals in one artifact after another artifact says the private scan is complete.

Every run must also create/update `{client-name}-INTERNAL_REPORT.html` and `outputs/latest/{client-name}-INTERNAL_REPORT.html`. Client-facing report files and the PDF companion must stay free of Solo Agency, WideCast, provider, Local Collector, automation, API-key/config, Telegram, and debug/system details; put those details in `INTERNAL_REPORT` instead.

Two notifications are acceptable: one when the public report is ready and one when the private report is ready or blocked. Notifications to the user/operator should normally point to `{client-name}-daily-report.html` or its uploaded operator-delivery URL, include the mandatory PDF companion path/status, include `{client-name}-INTERNAL_REPORT.html` path/status, and include lane-specific report links as secondary links when useful.

After creating or updating the three client-facing HTML files, run the Client-Blind Scrub Gate, then create or update the mandatory PDF companion package from those scrubbed HTML files: `{client-name}-client-report.html`, `{client-name}-client-report.pdf`, and `outputs/latest/{client-name}-client-report.pdf`. The PDF must be offered alongside the HTML report so the recipient can choose the format. It must not replace the three canonical HTML files. Private data source details must be safe summarized; raw private content, private source inventory, and internal system details belong in `INTERNAL_REPORT`. If PDF generation is blocked by tooling or redaction uncertainty, create the print-friendly `{client-name}-client-report.html`, record `client_pdf_status: blocked` with the exact blocker, and still hand off the HTML report plus the PDF blocker plus the INTERNAL_REPORT path/status.

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

Load only the stage needed for the current action, plus any dependency named by that stage.

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
| 7 | `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md` | Load whenever creating files, updating profile/history/logs, adding clients, or reading prior context. |
| 8 | `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md` | Load when installing, running, checking, scheduling, or troubleshooting the Local Collector. |
| 9 | `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md` | Load before claiming setup, daily run, private scan, production, measurement, or schedule completion. |
| 10 | `playbooks/10_LEAD_COMPETITOR_DETECTION.md` | Load whenever detecting, scoring, reporting, storing, or improving lead and competitor opportunities, including first runs and scheduled runs. |
| 11 | `playbooks/11_UPDATE_AND_VERSION_WATCH.md` | Load when the human asks to update/upgrade/sync latest Solo Agency, during stale-version/blocker recovery, and for the daily GitHub update-watch task. |
| Setup Entrypoint | `playbooks/SETUP_FLOW_ENTRYPOINT.md` | Use for setup/configuration sessions. Setup Flow configures clients, extensions, collector, schedules, automation prompts, and resync logs, but does not run reports. |
| Scheduled Entrypoint | `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` | Use as the scheduler prompt for unattended daily runs. |
| TODO | `playbooks/TODO.md` | Backlog for future improvements. Do not treat TODO items as daily questions to the human. |

## Mandatory Setup Flow

The setup flow is fixed and must stay aligned with the 10-item `Solo Agency one-time setup process` roadmap. Do not introduce hidden setup steps 11+ in human-facing setup messages.

1. Load Stage 0 and Stage 1, then ask only the first human question.
2. Infer and show industry, sub-industry, related industries, target audience, offer, location dependency, and language assumptions before asking the next question.
3. Infer and show pain points, content pillars, and the content mix rule. Ask target location only if location materially changes the plan and cannot be inferred.
4. Select public data sources and build a public search keyword bank. The keyword bank must include broad industry keywords, but it must be driven primarily by the target audience's pain points, problems, objections, questions, needs, buying triggers, and local context. The public data source list is not fixed: after each run, useful recurring public data sources discovered through search or reading must be saved/promoted so future scheduled runs can visit them automatically.
5. Ask one lightweight private data source preference question: whether the human wants Solo Agency to include private data sources later. Do not ask for URLs, group lists, account lists, discovery details, or Local Collector setup at this point. Valid outcomes are `private_sources_requested`, `private_sources_declined`, `private_sources_postponed`, or `private_sources_unsure`.
6. Configure the recurring schedule/routine once the basic public source plan and private data source preference are known. If private data sources were requested but are not active, configure the schedule as public data sources only for now and keep private data sources as `pending_private_activation`.
7A. Resolve or record the private data source checkpoint before claiming the client automation task is ready. If private data sources were requested, the human is unsure, private data sources already exist, or private data source discovery is needed, load `playbooks/PRIVATE_SOURCE_GATE.md`, Stage 2, Stage 8, and Stage 9. Ask for manual private data sources or offer one optional discovery pass from approved joined/followed/member spaces or Facebook keyword group search, get human approval before adding sources, and guide Local Collector setup if the human wants the automation task to include those sources. If the human wants to move faster, configure the first automation run to use public data sources only until private data sources are activated.
7B. Create or verify the client-specific automation task that will run the first report. The task name must begin with the client name, for example `AvenNgo - Solo Agency First Run` or `AvenNgo - Solo Agency Daily Run`. Do not run the first report inside Setup Flow. After schedule/automation exists, offer the maintenance task `Solo Agency - GitHub Update Watch` as a separate update-watch automation, not as a new human-facing setup step.
8. After the automation task is configured, ask whether the human wants PDNA setup - Production, Distribution, Notification, and Analytics - as client-scoped provider configuration. Do not treat a global MCP/native provider account as this client's PDNA connection. Do not create video/blog/social assets, render, publish, or spend credits inside Setup Flow.
9. If published URL history exists, record that future Automation Flow should load Stage 5 and scan analytics/signals; if no published URL history exists, mark analytics as not available yet. Do not scan analytics inside Setup Flow.
10. End Setup Flow only after setup/configuration state is current and the human has the exact client-specific automation task name to run for the first report. Do not update reports, idea matrices, best ideas, leads, competitors, drafts, or the learning loop inside Setup Flow; those belong to Automation Flow.

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
○ 5. You choose whether to include private data sources later (logged-in social groups, profiles, pages, channels, or communities). No links or install are needed at this step.
○ 6. I configure the automatic schedule/routine
○ 7A. If you chose private data sources, I help you provide/approve sources or discover candidates from approved joined/followed/member spaces or Facebook keyword group search, then guide Local Collector setup so the first run can include those sources; if you want to move faster, I keep private data sources pending
○ 7B. I create or verify the client-specific automation task that will run the first report in Automation Flow; I do not run the report inside this setup chat
○ 8. I help set up PDNA provider configuration only: Production (create real video/blog/social assets), Distribution (publish approved content), Notification (send reports/blockers), and Analytics (measure results)
○ 9. In Automation Flow, from the second run onward, if PDNA is set up, the task scans analytics for published URLs from the last 7 days
○ 10. In Automation Flow, the task updates the report, idea matrix, best idea, Lead & Competitor Opportunities, drafts, analytics/statistics, and learning loop
```

Progress roadmap integrity rule:

- Every setup progress block must show all 10 numbered items in order, including both substeps 7A and 7B.
- Never hide steps 5-10 because they are pending, declined, blocked, or not applicable yet.
- Use `○` for pending items, `→` for the current active item, `✓` for completed items, `!` for blocked or human-action-needed items, and `–` only after the human has explicitly declined or the item has been logged as not applicable with a reason.
- Step 5 is only a lightweight preference gate. Do not ask the human for private data source URLs, group lists, account lists, discovery details, or Local Collector setup in step 5.
- Do not ask private data source discovery as a separate roadmap item or gate. If the human wants private data sources or is unsure, offer one optional private data source discovery pass in plain language during step 7A, not during step 5.
- A declined or postponed discovery pass is valid, but the agent must record the status and explain that public-only runs may miss many lead/competitor/community signals.
- Step 6 is the one-time schedule/routine setup. It should happen before the first full agency run so future automation is already defined.
- Step 7A is the private data source intake, discovery, approval, and activation checkpoint. If private data sources were requested, the human is unsure, sources were provided/approved, or Local Collector is not installed/running/healthy, 7A becomes the next required question after step 6. The agent must either collect/triage/approve sources and guide Local Collector setup, or mark private data sources as pending so the client-specific automation task can run public data sources only until activation is complete.
- Step 7A may be marked `–` only when no private data sources exist, the human declines/postpones Local Collector, or the human explicitly chooses a public data sources only first run. The reason must be shown in plain language.
- Step 7B is the automation handoff, not the report itself. It must verify or create the client-specific automation task, state whether that task will use public data sources only or public plus activated private data sources, and give the exact task name the human should run for the first report. The agent must not generate the report, idea matrix, drafts, or video inside Setup Flow. After step 7B, the next setup question is step 8.
- Step 8 is client-scoped provider/capability setup only: choose the provider path, connect or document the production/distribution/notification/analytics provider for the current client, verify the account through that client's provider config/OpenAPI credential, check notification/publishing/analytics availability, and save the setup status. Notification setup must stay inside this step. It must not expand into open-ended trial video creation, scene editing, rendering, or publishing while the one-time setup process is still incomplete unless the human explicitly overrides after being told that setup will resume immediately after a short checkpoint.
- Step 9 applies only after PDNA - Production, Distribution, Notification, and Analytics - has been set up and published URL history exists. It must not be marked complete on the first setup run unless PDNA is set up, published URLs exist, and measurable signals already exist. If PDNA is not set up yet or there is no published URL history yet, mark step 9 as `–` with the honest reason such as `PDNA not set up yet` or `no published URLs yet`.
- Step 10 is shown in the setup roadmap only to explain what Automation Flow will do later. It is not executed in Setup Flow. On the first automation run it uses report/draft content and data from activated private data sources; from the second automation run onward it can also include analytics/statistics from step 9.

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
- do not start open-ended trial video creation or editing while steps 9-10 are still pending;
- after provider setup, gently return to the next setup step;
- defer trial video creation/editing until after the one-time setup process unless the human explicitly insists.

Good transition after provider setup:

```text
Production provider setup is connected. To keep the agency setup complete, I will finish the main setup path first: analytics history if there is published data, then the learning loop. After setup is complete, I can come back to a trial video or edits.
```

If the human explicitly asks to create or edit a video before setup is complete, treat it as a short controlled branch:

- save the parent setup checkpoint before entering the branch;
- state that this is a temporary branch and the agent will resume setup at the next checkpoint;
- show a compact parent checkpoint, not the full 16-item setup list, while the branch is active;
- after one natural checkpoint, gently resume the parent setup unless the human explicitly asks to continue the production branch.

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
Do you want to create the video from Version 1 now?
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

The agent may omit the next-step question only when the entire requested workflow is complete and no human decision is required.

## Non-Negotiable Summary

- Preserve every requirement in the loaded playbooks.
- Ask only for information that cannot be inferred, researched, discovered, or read from local files.
- Ask the first setup question only for product/service, profession, expertise, business description, or a public website/profile URL.
- Do not ask the human to define industry or sub-industry.
- Show inference before asking the next question.
- Configure schedule/routine and the client-specific automation task before the first report; if private data sources exist and Local Collector is not active, handle step 7A or mark private data sources as pending before declaring automation ready.
- If no private data sources are provided, offer optional private data source discovery from approved joined groups, subreddits, communities, followed profiles/pages/KOLs, subscribed channels, and feeds before treating the private data source step as resolved.
- Canonical client-facing reports are HTML and client-blind. Markdown is internal. A PDF companion is mandatory after the HTML report set is created or updated; it must be derived from the three scrubbed HTML files, offered alongside the HTML handoff, and recorded as generated or blocked with the exact blocker. The operator-only `INTERNAL_REPORT` path/status must be handed off alongside the client-ready files.
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
- Do not invent metrics. Mark unavailable metrics clearly.
- Communicate with the human in the human's language.
- Keyword language must follow the target audience's likely search/comment language, not automatically the human's chat language. If the human chats in one language but the target audience searches and comments in another, the keyword bank should prioritize the audience language.
- If a workflow is not complete and the agent is handing control back to the human, show progress and end with exactly one next-step question.

## Completion Gates

Setup is not complete until:

- Stage 0 and Stage 1 were loaded.
- The first question followed the minimal-input rule.
- Inference was shown to the human.
- Public data sources and keyword strategy were selected.
- The public keyword bank includes pain-point/problem/need keywords, not only generic industry keywords, uses the target audience's search language, and the full bank was saved for rotation.
- Useful recurring public data sources discovered during runs were saved/promoted into `public_data_sources` with cadence so later scheduled runs can revisit them.
- Step 5 private data source preference was resolved before schedule setup, and step 7A private data source intake/discovery/approval plus the Local Collector checkpoint were resolved or honestly marked pending before the client-specific automation task was declared ready.
- Schedule/routine and the client-specific automation task were configured before the first report.
- The automation task contract requires the first automation run to load Stage 10, generate the three-file client-facing HTML report set (`{client-name}-public-data-sources-report.html`, `{client-name}-private-data-sources-report.html`, `{client-name}-daily-report.html`), generate `{client-name}-INTERNAL_REPORT.html`, pass the Client-Blind Scrub Gate, include lane-specific Lead & Competitor Opportunities with post/current URLs and copy-ready value-first comments when opportunities exist, reject direct-promo ideas as `promotional_not_value_first`, and create at least one useful audience-value-first draft script/blog/caption.
- The setup handoff showed the exact task name the human should run for the first report.
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
- If private data sources were requested or the human was unsure, step 7A offered manual source intake and optional private data source discovery from approved joined groups, subreddits, communities, followed profiles/pages/KOLs, subscribed channels, and feeds, or recorded that the human declined/postponed it.
- Any approved discovery scan was filtered before activation.
- The Local Collector status was checked or the blocker was documented.
- Collected data was analyzed for data points, leads, competitors, new sources, idea matrix, best idea, and drafts.
- Stage 10 was loaded before presenting lead and competitor opportunities.
- The HTML report was regenerated.

Production/distribution is not complete until:

- Stage 3 was loaded.
- Drafts were shown to the human.
- Explicit approval was received for any create/render/export/publish/credit-spending/clone action.
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

- If the agent is about to ask setup questions but Stage 0 or Stage 1 is not loaded, load them first.
- If the agent is about to discuss private data sources but the private data source gate and Stage 2 are not loaded, load `playbooks/PRIVATE_SOURCE_GATE.md` and Stage 2 first.
- If the agent is about to scan, open, monitor, or collect from a private data source, stop and reload `playbooks/PRIVATE_SOURCE_GATE.md`, Stage 2, Stage 8, and Stage 9 before opening any browser or URL.
- If the agent is about to install or run collector tooling but Stage 8 is not loaded, load it first.
- If the agent is about to detect, score, report, store, or improve leads or competitors, load Stage 10 first.
- If the agent is about to answer an update/upgrade/sync-latest request, or is about to resolve a blocker by checking the latest GitHub version, load Stage 11 first.
- If the agent is about to create, render, publish, or notify through a production provider but Stage 3 is not loaded, load it first.
- If the setup agent is about to run the first agency run/report directly, stop and prepare or resync the client-specific automation task instead.
- If an automation agent is about to run the first report before private data source status, the 7A Local Collector checkpoint, and schedule/routine are resolved or honestly marked pending, stop and load the needed stage.
- If the agent is running from a schedule, it must still load the needed stage playbooks again at run time; schedule execution is the same workflow with saved context, not a memory-only shortcut.
- If the agent is about to claim completion, load Stage 9 and run the relevant checklist.

## Self-Audit Summary

Before every reply, the agent must check:

- Did I answer in the human's language?
- Did I avoid asking for things I can infer or research?
- Did I load the required stage files for the action I am taking?
- Did I avoid jumping past private data source status, the 7A Local Collector checkpoint, schedule/routine setup, client-specific automation readiness, approval gates, or measurement gates?
- Did I give the human a short approval-ready decision instead of a long questionnaire?
- Did I avoid presenting Markdown as the human-facing report?
- Did I preserve safety, credentials, private-data, and approval rules?

If any required stage was not loaded, load it before proceeding.
