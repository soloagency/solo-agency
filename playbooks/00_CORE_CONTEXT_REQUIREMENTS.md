# Core Context Requirements

Stage: `00`

## Load Rule

Load first for every setup or run. This stage contains the core reasoning model, non-negotiables, A-H workflow, source logic, lead/competitor logic, adjacent reasoning, idea matrix, best-idea selection, and language/report rules.

## Hard Gates For This Stage

- The agent must infer before asking.
- The agent must preserve related-industry 80/20 logic.
- The agent must detect leads and competitors while researching.
- The agent must load Stage 10 before presenting lead/competitor opportunities, report comments, or lead/competitor logs.
- The agent must keep user-facing language aligned with the human's language.
- The agent must never treat research from public data sources only as private data source coverage.
- The agent must explain marketing, analytics, and technical terms in plain language when speaking to a non-technical/non-marketing human.
- The agent must use canonical source terminology in human-facing text: `public data sources` and `private data sources`. Do not shorten these terms, omit `data`, use slash terms, or use mixed-language shorthand labels.
- The agent must not mention private data sources in the first setup or first add-client question. Private data sources are asked only after the schedule/routine and client-specific automation task have been configured; if the human approves or changes private data sources, resync the automation task afterward.
- The agent must not create local video media as a fallback when a verified client-scoped PDNA provider is missing or blocked. Missing provider setup must trigger a PDNA setup/action block, not a local `ffmpeg`/Pillow/`moviepy`/Remotion/canvas/slideshow video.
- Any video script inside a report, Markdown source record, previous draft, or history is reference context only. Before any provider video creation request, the agent must load and apply the existing WideCast video script-writing skill from the verified provider or `playbooks/skills/video-script-writing/SKILL.md`, produce the final production script/brief with research and inline-media/direct-image-URL workflow where verifiable, and use only that skill-produced final script/brief as the provider payload. Do not edit, replace, summarize, or reimplement the WideCast skill.
- Report video-script versions are selection options only. If a version/code is already selected by the human, pasted back with edits, or saved as the automation recommendation, do not generate five new versions during video production. Continue only with that selected version/code through the WideCast skill's Stage 2 visual treatment and final handoff standards.
- Default PDNA setup must be one-action WideCast setup. When PDNA is missing and the human asks for setup/instructions/video/production, ask only for the client's WideCast API key; do not ask provider, scope, spend, publish, notification, analytics, or account-identity questions before starting the default path.

## Source Preservation Rule

This file is detailed source material moved from the original monolithic `SOLO_AGENCY_PLAYBOOK.md`.

Do not summarize away requirements, examples, checklists, schemas, protocols, URLs, edge cases, warnings, approval gates, or completion gates. If a downstream agent needs to shorten its response to the human, it may summarize the response, but it must still obey the full requirements in this file.

---

## Latest Override: Control Plane Versus Operations Plane

Solo Agency now separates setup/configuration from operational runs:

- Setup Flow is the control plane. It creates and updates client setup, source state, collector/extension config, schedules, automation tasks, and resync logs. It must not run reports, scans, production, publishing, rendering, or outreach.
- Automation Flow is the operations plane. It runs reports/scans/drafts from saved config and may also update config based on real findings, but must resync those changes for future runs.
- The first report is not a setup-session deliverable. It must run through a client-specific automation task whose name begins with the client name.
- Client-specific tasks must process only their pinned `target_client_slug`.
- Use one shared Local Collector app/bridge, and one client-specific Chrome extension folder per client Chrome profile/account under `extensions/{client_slug}/`.

This override wins over older references in this file to first agency runs, report drafts, or report delivery during setup.

Report request hard stop in Setup Flow:

- If the human asks to run, create, generate, show, refresh, or update a report while the current session is Setup Flow, do not comply by running the report.
- A report request does not switch the current setup chat into Automation Flow.
- The agent must finish or resync the client-specific automation task and tell the human the exact task name to run.
- Do not ask whether the human wants the agent to run the report now.
- Do not load the scheduled-run entrypoint, start public research, collect private data sources, generate an idea matrix, create Lead & Competitor Opportunities, write drafts, scan analytics, or send report notifications inside Setup Flow.
- If the native automation task cannot be updated by the agent, mark `automation_prompt_update_pending`, write the exact prompt/update instructions to `daily-content-pipeline/automation/scheduled_run_prompt.md`, and ask the human to update/run the native task.

---

## Required Runtime

Solo Agency must run in an AI agent runtime that supports:

- local workspace file reads/writes;
- scheduled automation or native tasks;
- multiple parallel/sub-agent work streams for research, private collection coordination, reporting, provider checks, and resync verification;
- Local Collector setup handoff and later collector file/status inspection.

Good runtime examples include Codex, Claude Desktop/Cowork, Hermes, OpenClaw, or comparable desktop/local agent environments. A plain web chat is not enough for the full Solo Agency workflow. The agent may use web chat only as a review/conversation surface, not as the primary runtime for setup, scheduled runs, Local Collector coordination, or multi-agent automation.

When asked how to install or run Solo Agency, the agent must say this plainly before setup proceeds. Do not imply that pasting the playbook into a browser-only chat will create the automation system.

## 0. Latest Delta Requirements And Modularization Plan

This section records the latest requirements that must be treated as part of the source of truth before any future split into child playbooks.

The original monolithic playbook remains the base manuscript, but the agent must also preserve and implement the requirements in this section. If the playbook is later split into modular files, do not summarize away any requirement from this section.

### Thin Root And Child Playbooks

Future modular versions must use a thin root `SOLO_AGENCY_PLAYBOOK.md` as an index/router, with detailed instructions moved into child playbooks under `playbooks/`.

The root file should contain only:

- mission;
- first agent instruction;
- stage map;
- missing-playbook download rule;
- mandatory setup flow;
- visible checklist;
- non-negotiable summary;
- completion gates;
- self-audit summary;
- routing instructions.

Detailed protocols must live in child playbooks.

Recommended child playbooks:

- `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`: full A-H workflow, examples, reasoning rules, lead/competitor rules, related-industry rule, language/report rules.
- `playbooks/01_BASIC_PROFILE_PUBLIC_REPORT.md`: setup interview, inference-first profile, public research, keyword rotation, first agency run and HTML report.
- `playbooks/PRIVATE_SOURCE_GATE.md`: short anti-drift gate for any private data source scan request involving logged-in groups, feeds, profiles, or communities.
- `playbooks/02_PRIVATE_SOURCE_SETUP.md`: manual private data sources, optional private data source discovery, Local Collector activation, source discovery, first private scan, private-enhanced report update.
- `playbooks/03_PRODUCTION_DISTRIBUTION.md`: writing drafts, production provider setup, video/blog/social creation, video scene editing, publishing, notifications. Do not name this file after any specific vendor.
- `playbooks/04_DAILY_SCHEDULE.md`: manual/daily/weekly routine and schedule.
- `playbooks/05_MEASURE_LEARN_IMPROVE.md`: published content analytics, comments, 7-day measurement, learning loop.
- `playbooks/06_AGENCY_REPORT_STANDARD.md`: agency-grade mobile HTML report standard.
- `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`: folder structure, client profile schema, logs, ledgers, history.
- `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md`: Solo Agency Local Collector app, Chrome extension, localhost API, health, run-now, source discovery, schedule config.
- `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md`: multi-client ops, approvals, safety, regulated industries, self-audit, completion gates.
- `playbooks/10_LEAD_COMPETITOR_DETECTION.md`: deep lead and competitor opportunity detection, scan depth rules, post-link preservation, value-first copy-ready comments, and report/storage requirements.

If the `playbooks/` folder is not available, the agent must download the needed child playbook from:

```text
https://raw.githubusercontent.com/soloagency/solo-agency/main/playbooks/
```

### Fresh GitHub Source Rule

For setup, setup repair, repo update, Local Collector preparation, and any action that copies playbooks or collector files into a human's setup, GitHub `main` is the source of truth unless the current setup root is verified as a fresh clone of the same repo.

The agent must:

- clone or download from `https://github.com/soloagency/solo-agency` into the current setup root or a fresh unique `mktemp -d` directory;
- never reuse fixed shared fallback folders such as `/tmp/solo-agency`, `/var/tmp/solo-agency`, `/dev/shm/solo-agency`, or any previously used cache path;
- verify `.git` exists, `origin` is the Solo Agency GitHub repo, and local `HEAD` matches `git ls-remote origin refs/heads/main` before reading or copying repo contents;
- treat folders without `.git`, folders owned by another user, old timestamp folders, and failed delete/update targets as stale cache;
- stop, request permission, or hand the human one exact GitHub command if fresh GitHub access is blocked by sandbox/network limits;
- never fall back to unverified local code, even if it appears to contain the expected files.

### Last-Resort Recovery And GitHub Issue Escalation Rule

Because this repo changes frequently, the agent must treat any blocker, repeated failure, confusing contradiction, tool/config mismatch, stale asset, missing capability, or operational dead end as a possible old-playbook/code problem before giving up.

Recovery sequence:

1. Fetch or clone the latest `https://github.com/soloagency/solo-agency` `main` into a fresh unique verified checkout, or read the relevant raw GitHub playbook files.
2. Do not use shared stale paths such as `/tmp/solo-agency`.
3. Compare local `HEAD` with GitHub `main` and record both commit hashes.
4. Reload the relevant latest playbook sections and check whether the newer instructions fix the issue.
5. If the latest version fixes or clarifies the issue, follow it, update/resync the client setup or automation state when relevant, and report the recovery.
6. If the agent is still blocked after the latest GitHub check, open or draft a GitHub issue for `soloagency/solo-agency`.

Issue escalation and tracking:

- The human does not need a GitHub account for blocker escalation. Do not make GitHub registration the required next action.
- Direct GitHub issue creation requires an authorized agent/runtime identity. Use `gh issue create` only when `gh auth status` passes, `GITHUB_TOKEN`, `GH_TOKEN`, or `SOLO_AGENCY_GITHUB_ISSUE_TOKEN` is configured, a GitHub App/maintainer bot is available, and the environment permits issue creation.
- Preferred operator setup is a dedicated maintainer bot token or GitHub App with narrow issue-writing access for `soloagency/solo-agency`, exposed only in trusted agent runtimes. Never store this token in client config, reports, issue drafts, or committed files.
- If no authorized GitHub identity is available but a project support/intake channel is configured, send or queue the redacted issue draft through that channel.
- If direct issue creation/sending is unavailable, write a ready-to-post issue draft under `daily-content-pipeline/automation/issues/`.
- Track issue URL/number, intake channel, or draft path in `daily-content-pipeline/automation/github_issues.md`.
- Include a redacted blocker fingerprint, safe reproduction steps, expected/actual behavior, local commit, GitHub `main` commit checked, environment/runtime, relevant blocker names, and redacted logs.
- Never include API keys, tokens, cookies, passwords, private data source raw content, client-confidential details, raw logged-in screenshots, or sensitive customer data.
- Check tracked issues during later setup repair, blocker recovery, and scheduled runs. If founder/community replies with a fix, apply it, update/resync automation, update the tracker, and notify the human.
- Reuse an existing issue when the blocker fingerprint matches; do not create duplicates.

### Canonical User-Facing Description Rule

When explaining what Solo Agency does, the agent must not describe it as only researching, finding ideas, writing drafts, and publishing.

The explanation must include production explicitly:

- researches the market every day;
- finds source-backed content ideas, hot/warm leads, and competitors;
- writes approval-ready scripts/blogs/captions;
- after human approval and provider setup, creates video/blog/social assets through connected production tools, including scene-editing review before final video render/export;
- can publish approved content to 10+ connected platforms when authorized;
- measures results and feeds the learning into the next run.

A good concise explanation is:

```text
Every day, Solo Agency researches the market, finds source-backed content ideas, detects hot/warm leads and competitors, drafts scripts/blogs/captions for approval, creates approved video/blog/social assets through connected providers, audits/fixes reviewable video scenes before final render/export when video production is approved, publishes approved content to 10+ platforms when authorized, measures results, and uses that learning to improve the next run.
```

Do not imply that production is only a manual copy/paste step. Also do not imply that rendering, publishing, spending credits, face clone, voice clone, or outreach happens without explicit human approval.

### Required Visible Setup Progress Roadmap

The agent must show and update this progress roadmap during setup so the human can catch missed steps.

This is a human-facing progress roadmap, not an internal agent instruction list and not a form for the human to answer line by line. Use `You` for the actions the human must provide or approve, and `I` for the actions the agent performs. Do not display internal verbs such as "Ask", "Infer", "Select", or "Run" as if the human were reading agent instructions.

For human-facing progress, prefer font/text status icons over raw checkbox syntax:

- `✓` done
- `→` current step
- `○` pending
- `!` blocked or needs human action
- `–` skipped, declined, or not applicable with a short reason

Every progress block must include a short line explaining that this is the agent's planned progress/process, not a questionnaire for the human.

The checklist must not assume the human understands marketing or technical terms. Explain terms directly in the checklist or immediately below it. Required meanings:

- `public data sources`: websites, search, news, public forums, and public pages the agent can access without the human's login.
- `private data sources`: logged-in or membership-based sources such as Facebook groups/pages, X, LinkedIn, Instagram, TikTok, YouTube, Reddit, GitHub areas that require access, Discord/Slack communities, competitor profiles, newsletters, or private forums.
- `Local Collector`: local app plus Chrome extension on the human's computer; it uses the already logged-in Chrome session, reads approved visible pages only, and keeps private data local by default.
- `offer`: business promise/package/value proposition.
- `pain points`: customer problems, worries, objections, or urgent questions.
- `content pillars`: repeatable main content themes.
- `lead`: potential-customer or buying-signal.
- `Lead & Competitor Opportunities`: report section where lead/competitor signals become reviewable opportunities with source links, context, and a suggested value-first comment the human can copy.
- `PDNA`: Production creates real assets, Distribution posts/sends approved outputs, Notification sends reports/blockers, Analytics measures performance.
- `learning loop`: using results to improve the next run.
- `hot/warm lead`: a stronger/weaker potential-customer signal based on urgency and fit.
- `competitor`: a direct competitor, alternative solution, adjacent option, or account whose positioning/hooks are useful to learn from.
- `idea matrix`: a simple table that organizes content ideas by type and business purpose.
- `HTML report`: a browser/mobile-friendly report file or link for the human to review.
- `draft`: a proposed script, blog, or caption waiting for human review, not published content.
- `analytics/statistics`: visible performance numbers such as views, likes, comments, shares, saves, clicks, followers, and unavailable metrics when a platform hides them.
- `schedule/routine`: when and how often the agent runs automatically.

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
- Step 7 is provider/capability setup only: use WideCast as the default provider, ask only for the client's WideCast API key, connect or document the production/distribution/notification/analytics provider, check notification/publishing/analytics availability, and save the setup status. Do not ask provider/scope/spend/publish/account-identity questions for the default path. Notification setup must stay inside this step. It must not expand into open-ended trial video creation, scene editing, rendering, or publishing while the one-time setup process is still incomplete unless the human explicitly overrides after being told that setup will resume immediately after a short checkpoint, the client-scoped provider is verified, and the required operation exists.
- After a provider creates reviewable video scenes from an approved script, the normal production branch is not complete until the video-editing skill pass has audited/fixed the scenes or logged an explicit blocker/decline. Final MP4 render/export still requires a fresh explicit approval after that pass.
- Step 8 applies only after PDNA - Production, Distribution, Notification, and Analytics - has been set up and published URL history exists. It must not be marked complete on the first setup run unless PDNA is set up, published URLs exist, and measurable signals already exist. If PDNA is not set up yet or there is no published URL history yet, mark step 8 as `–` with the honest reason such as `PDNA not set up yet` or `no published URLs yet`.
- Step 9 is shown in the setup roadmap only to explain what Automation Flow will do later. It is not executed in Setup Flow. On the first automation run it uses report/draft content and data from activated private data sources; from the second automation run onward it can also include analytics/statistics from step 8.

### Progress And Next-Step Question Rule

While setup, daily run, private data source activation, production setup, publishing, scheduling, or measurement is still incomplete, every human-facing reply that hands control back to the human must include a compact progress block.

During scheduled runs, every human-facing progress update, notification, or report handoff must include `Solo Agency daily run progress`. If the scheduled run sends multiple updates, each update must refresh completed/current/remaining steps.

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

If any required step remains and the agent is waiting for the human, the final line of the message must be exactly one clear next-step question.

Do not end with a passive summary, a report link, or a vague statement such as "let me know what you think."

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

Even when the entire requested workflow is complete and no human decision is required, the agent still closes with next-action guidance per the root Next-Action Guidance Rule: suggest 1-3 real, currently-available next steps and ask which one the human wants.

### Manual Private Data Sources And Discovery Are Independent

Private data source setup must support both paths independently:

1. The human manually provides private data sources:
   - competitor profiles;
   - fanpages;
   - Facebook groups;
   - social profiles;
   - KOL accounts;
   - YouTube channels;
   - TikTok accounts;
   - X accounts, lists, or communities;
   - LinkedIn profiles, pages, or groups;
   - Reddit communities;
   - any private or community source they want monitored.

2. The human allows the agent to discover private data sources from sources and feeds the user already follows or belongs to:
   - groups the user has joined;
   - profiles/pages/KOLs the user follows;
   - channels the user subscribes to;
   - recommendation feeds;
   - news feed / home feed signals.

The human can choose only manual sources, only discovery, both, neither, or postpone either option.

The agent must not assume discovery replaces manual source input.

If the human provides no private data sources, says they are not sure, or skips the question, the agent must treat that as a discovery opportunity, not as proof that private data sources are unnecessary. Offer one concise option to discover candidate private data sources from approved joined groups, subreddits, communities, followed profiles/pages/KOLs, subscribed channels, and feeds. If the human declines or postpones, record that status and continue public data source work with a clear note that lead/competitor/community coverage is limited.

### Source Discovery Deep Scroll Rule

There are two separate private collection modes:

1. Source Discovery Mode:
   - Used to discover groups, profiles, pages, channels, KOLs, and communities.
   - Must scroll deeply until no new source names/URLs appear for 3 consecutive scrolls.
   - Use a hard safety cap of 10 scrolls.
   - Do not stop at the daily default of 5 scrolls because this can miss many joined groups or followed sources.
   - Filter candidate sources by relevance before asking the human to approve them.

2. Daily Content Monitoring Mode:
   - Used after sources are approved.
   - Default: 5 scrolls per source.
   - Maximum: 10 scrolls per source.
   - Delay: 5 seconds between scrolls.
   - Recommend about 20 daily private data sources or fewer per client.

### Private Scan Completion Rule

After private collection runs, the agent must not stop at `collector succeeded`.

Private data source setup or enrichment is complete only when the agent:

1. reads the collected data;
2. extracts relevant data points;
3. detects hot/warm leads;
4. detects competitors;
5. detects new private data sources;
6. updates the idea matrix;
7. re-scores or updates the best idea;
8. updates drafts/scripts/blogs if private data changes the recommendation;
9. regenerates/updates only the private lane report (`{client-name}-private-data-sources-report.html`) and the daily staging index, then rebuilds the combined `{client-name}-client-report.html` + PDF companion + `{client-name}-INTERNAL_REPORT.html`, never overwriting `{client-name}-public-data-sources-report.html`, and reconciles `{client-name}-report_state.json` and `outputs/latest/` copies;
10. shows the updated report to the human.

Collector success alone is not completion.

### Stage Flow After Setup Readiness

Stage 1 must take the client from basic profile to `ready_for_automation_first_run`.

After the first automation report is delivered, the Automation Flow agent may ask:

- Do you want me to set up PDNA - Production (create real video/blog/social assets), Distribution (publish approved content), Notification (send reports/blockers), and Analytics (measure results) - so approved drafts can become real assets and the system can learn from performance later?

After the first automation report is delivered, if the human wants production, video/blog/social, publishing, notifications, analytics, or fully automatic operation, load the production/provider setup playbook and complete checklist step 7. In Setup Flow, treat this as provider/configuration only unless the human has explicitly moved into Automation Flow.

If the human wants private data sources before the first automation run, load the private data source playbook, complete Local Collector setup if needed, then update the client-specific automation task. The step-6 discovery pass is configuration gathering (its output is the approved source list), with two sanctioned paths. Interactive path (preferred): when the human approved discovery in this session AND the Local Collector plus the matching client extension are verified healthy in this session, run the discovery pass at the step-6 checkpoint itself — Local Collector only, approved categories only, Source Discovery Mode pacing — show the filtered shortlist, get approval, save the approved sources, and run Automation Resync, so step 6 closes in one sitting; Setup Flow still must not analyze the collected data, generate a report/idea/draft from it, or start daily monitoring. Deferred path: when the collector is not yet healthy, the human is not present to approve, or the human postpones, record `approved_pending_first_scan` and resync the automation task; the first Automation Flow run MUST then execute the approved discovery/scan and present the candidate shortlist for human approval when the Local Collector and matching extension are healthy, or report the exact collector blocker — it must not silently defer approved discovery while the collector is healthy. Once a scan has produced a shortlist the human has not yet approved (`discovery_completed_pending_approval`), later runs re-surface the shortlist instead of re-running discovery (see the private data source playbook).

### Published Content Measurement Requirement

The measure-learn-improve phase is mandatory once content has been published.

For each published content item, the agent must:

1. Use connected provider analytics when available.
2. If provider OpenAPI/tools are connected and verified for this client, call the relevant operations to retrieve:
   - videos/posts published yesterday;
   - videos/posts published in the last 7 days;
   - published URLs;
   - title;
   - description;
   - caption;
   - hashtags;
   - platform;
   - publish date;
   - topic/video/content IDs;
   - account/platform analytics when available.
3. Measure each published URL daily for up to 7 days after publishing.
4. Reuse the Solo Agency Local Collector to open each published URL when useful and authorized, because some metrics/comments require a logged-in browser.
5. Capture visible:
   - views;
   - likes/reactions;
   - comments;
   - shares;
   - saves;
   - reposts;
   - follower/subscriber count when relevant;
   - audience questions;
   - objections;
   - lead signals in comments.
6. Store metrics in `analytics/metrics_log.md`.
7. Store comments/questions in `analytics/comment_signal_log.md`.
8. Store learnings in `analytics/learning_log.md`.
9. Use learnings to improve:
   - source priority;
   - content pillars;
   - hook selection;
   - CTA selection;
   - idea scoring;
   - lead-gen angles;
   - future scripts/blogs.

Do not invent metrics. Mark unavailable metrics clearly.

### README Feature Requirement

The human README must include a polished feature line explaining that Solo Agency can, with consent, help discover and collect useful private data sources from groups they joined, profiles/pages/KOLs they follow, subscriptions, and feeds, so social signals are not lost.

Phrase it as a marketing benefit, not a technical implementation detail.

## 1. Non-Negotiable Operating Principles

The agent must follow these principles at all times:

- Preserve every requirement in this playbook.
- Think and infer as much as possible before asking the human anything.
- Ask only for information that is truly required and cannot be inferred, researched, or discovered.
- During setup, ask questions step by step; after every human answer, immediately infer what can be inferred from that answer and show the inference before asking the next question.
- Do not ask the human to define `industry` or `sub_industry`.
- Ask the human first only for the product/service, profession, expertise, business description, or a public website/profile URL the agent can inspect for setup context.
- Treat a public website/profile URL as valid setup input. Reading that public page to infer industry, offer, audience, location, and content pillars is allowed during Setup Flow; it is not the same as running an operational public data source scan, report, or daily research pass.
- Infer `related_industries` after inferring the primary industry and sub-industry. Show those related industries to the human during setup and use them to broaden research and content angles.
- Keep the content strategy anchored around the primary industry: approximately 80% of ideas/scripts should revolve around the primary industry and primary offer, and approximately 20% may use related industries when there is a clear logical bridge back to the client's offer, audience, pain points, or lead-generation goals.
- Ask for `target_location` only if the business is location-dependent and the location cannot be inferred.
- Ask the human to provide private data sources they want monitored, such as competitor profiles, fanpages, groups, communities, or social accounts.
- Ask the human whether they want to include Facebook groups where they are already a member as monitored private data sources; explain that the agent will filter those groups based on whether they contain discussions relevant to the client's primary industry, related industries, audience, location, and pain points.
- If the human wants help finding more private data sources, offer optional private data source discovery in plain language: groups/subreddits/communities they joined, pages/profiles/KOLs they follow, channels they subscribe to, and platform feeds that recommend relevant content. Explain that this discovery is optional, requires consent, uses the Solo Agency Local Collector, and must be filtered before anything becomes an active private data source.
- During private data source setup, repeatedly reassure the human in simple language:
  - They are setting up a professional agency-scale system, so the first setup takes patience but normally happens only once.
  - Private data is saved locally on their own computer and must not be sent outside their computer unless they explicitly approve an export.
  - Once activated, the system can scan daily so important market signals, leads, competitor moves, and content ideas are less likely to be missed.
- When researching public data sources, use Google Search or an available equivalent search tool to try primary-industry, related-industry, sub-industry, audience-pain, local, and news-related keywords. Rotate keywords daily or per attempt until the results produce useful data points.
- When scanning private or logged-in sources, use conservative pacing: do not scan aggressively, do not run many private data source browser checks in parallel, and leave a 5 second delay between private data source scroll/read actions so platform feeds have time to load.
- Warn the human not to add too many private data sources for one client. As a practical default, keep the daily private data source monitoring list around 20 sources or fewer per client. If the human provides more, prioritize the most relevant sources and rotate lower-priority sources across different days.
- Do not use Claude Chrome Extension for automated private data source collection. It can require repeated human permission clicks and can trap the human in an approval-gated flow. For Claude, use the Solo Agency Local Collector extension plus the Local Collector app, a user-started Local Collector command, or an OS startup service.
- If the conversation drifts and later returns to private data source work, the agent must treat that as a fresh private data source turn. Before scanning, opening, monitoring, or collecting any private data source, including logged-in groups, feeds, profiles, pages, communities, or sources, reload `playbooks/PRIVATE_SOURCE_GATE.md`, Stage 2, Stage 8, and Stage 9 (print a LOAD LEDGER per `playbooks/LOAD_LEDGER_PROTOCOL.md` for each file loaded).
- Never use Claude in Chrome, Claude Chrome Extension, Codex built-in/in-app browser, ChatGPT/Gemini/Grok browser, Playwright/Puppeteer/Selenium, a fresh agent-opened browser profile, remote-debugging browser, or any agent-controlled browser for private data source collection. Use only the Solo Agency Local Collector extension plus Local Collector app.
- If an AI environment cannot browse private data sources reliably, cannot show a headed browser UI, cannot run downloaded executables, or requires per-run browser approvals, use the Solo Agency Local Collector extension plus the Local Collector app as the preferred private data collection layer instead of trying to bypass permission prompts.
- During one-time Local Collector setup/update/repair, the AI agent must not run `setup_collector.sh`, `setup_local_collector.ps1`, `Start Local Collector.cmd`, or the collector binary itself, even if local shell permissions are available. Agent-run setup can happen inside a sandbox/session and be killed after the turn. The agent must prepare the files, then give the human the exact one-line Terminal/PowerShell command to run outside the AI sandbox.
- Local Collector activation requires the human to run the shared Local Collector app setup/start command, then load the client-specific Solo Agency Local Collector extension from the absolute `extensions/{client_slug}/` folder in the matching client Chrome profile/account. Do not mark private data source monitoring active until the shared bridge and the matching client extension health checks pass.
- When asking for or working with private data sources, tell the human they must already be a member, follower, subscriber, logged in, or otherwise authorized to view those sources in the Chrome profile where the client-specific extension is installed. Recommend one separate Chrome profile per client with that client's extension loaded and the relevant social accounts logged in there.
- When speaking to non-technical humans, do not say `bridge`, `localhost bridge`, `binary`, `daemon`, or `service worker` unless troubleshooting. Say `Solo Agency Local Collector extension` and `Local Collector app`. Explain the Local Collector app as: "a small app running on your own computer that receives data from Chrome and saves local files for the AI agent to read."
- The collector is platform-neutral. Never call it `Facebook collector`, `Facebook Data Collector`, or `collector Facebook`, even when the private data sources supplied by the human are currently all Facebook groups/pages. Say `Solo Agency Local Collector extension` and explain that it can collect visible authorized data from configured logged-in web sources such as Facebook, LinkedIn, Reddit, X, Instagram, TikTok, forums, and other browser-accessible private data sources.
- Setup completion rule: after setup context and routine are saved, create/verify the client-specific automation task first, then resolve or record the step 6 Local Collector checkpoint if private data sources are requested/exist, and resync the task after any source-state change. The first report must run in Automation Flow, not inside the setup chat.
- If the human provided private data sources but Local Collector is not active, the first agency report must clearly say that private data source monitoring is not activated yet and requires the Solo Agency Local Collector extension plus Local Collector app.
- Private data source activation gate: the agent must not claim private data source monitoring is active or run scheduled private collection until collector setup has either completed or been clearly documented as blocked in `collector_setup_status.md`.
- Manual private data sources and optional private data source discovery are independent options. The human may provide private data source URLs, approve discovery from joined groups/subreddits/communities/followed profiles/feeds, do both, decline both, or postpone either option. If the human has no list, offer private data source discovery once before marking the private data source step resolved. Do not ask private data source discovery as a separate user-facing setup step.
- Private data source completion gate in Automation Flow: after any private scan, the automation task must analyze the collected private data and regenerate the idea matrix, best idea, leads, competitors, and drafts if needed. For the report itself, regenerate/update only the private lane report (`{client-name}-private-data-sources-report.html`) and the daily staging index, then rebuild the combined `{client-name}-client-report.html` + PDF companion + `{client-name}-INTERNAL_REPORT.html`, never overwriting `{client-name}-public-data-sources-report.html`, and reconcile `{client-name}-report_state.json` and `outputs/latest/` copies. A private scan is not complete merely because the Local Collector successfully collected data.
- The first report happens after the profile/source plan, schedule/routine, client-specific automation task, and step 6 Local Collector/private data source checkpoint are ready or honestly marked pending, and it must be launched through the client-specific automation task.
- Ask about the recurring schedule during setup after the profile and source plan are known. If private data sources exist, do not promise scheduled private collection until the shared bridge and matching client extension are complete or clearly pending/blocked.
- After schedule/routine setup, if private data sources exist and Local Collector is pending, do not ask to run a report in setup. Configure the automation task for public data sources only first, or guide activation so the task can include private data sources later.
- For non-technical humans, never ask them to copy a long multi-line shell/PowerShell script. Create the script file locally first, then provide exactly one short command to run that file in their own Terminal/PowerShell outside the AI sandbox, or provide one double-clickable launcher path on Windows.
- Do not tell the human to keep the setup/report/instruction browser tab open. After they run the required command or load the extension, they may close the tab. If a Terminal/PowerShell process is used before auto-start is configured, explain that the Local Collector app process may need to keep running until the first agency run finishes, but the browser tab itself is not required.
- Never ask for credentials, passwords, OTPs, cookies, tokens, or raw login secrets.
- Do not require a production-provider account, MCP connection, API key, or installed provider tool just to produce ideas, blog drafts, video scripts, or social captions. Writing must continue by loading the public writing-method fallback protocol in this playbook.
- When provider notification/Telegram capability is available, use it to notify the human about completed scheduled runs, required approvals, session-expired issues, setup blockers, and any important failure because the human may not be present when the schedule runs.
- Report-ready notifications must include the HTML report URL/path. A notification that only says the report is ready but does not include a link/path to the `.html` report is invalid.
- When WideCast notification/Telegram/email fallback is available or configured as preferred and a run produced an HTML report, run the Stage 6 Provider Report Delivery Capability Check. Inspect the current client's provider config and WideCast OpenAPI spec before falling back to legacy tool discovery. If `uploadAsset` supports `text/html`, upload the `.html` report to WideCast first and send the uploaded WideCast report URL through `sendTelegramMessage`. If provider config, auth, OpenAPI discovery, upload, or notification fails, log the exact blocker and send/surface the best available local/hosted `.html` report path/link instead.
- If the current AI connector/tool surface does not expose WideCast upload or Telegram/notification tools, do not stop there. Check the per-client OpenAPI provider path first. Do not claim WideCast itself lacks the API/capability unless verified from WideCast account/API/OpenAPI status.
- If the agent accidentally sends a report-ready notification without a report URL/path, it must immediately send a correction notification containing the HTML report URL/path and log the correction.
- Show all inferred and researched setup context to the human before treating it as stable.
- Continue with public data sources if private data sources are missing, not yet activated, or unavailable after the required Collector Runtime Verification. If private data sources were provided but Local Collector has not been installed yet, label them as `pending_private_activation`, not as silently skipped.
- If a logged-in private session expires, skip that private data source, log it, and ask the human to log in again manually.
- Do not publish, post, comment, message, render, create a provider-hosted video, export a video, or spend credits without explicit human confirmation.
- Communicate with the human in the same language the human uses.
- Store internal operational field names and schemas in English unless the human explicitly asks otherwise.
- Write human-facing reports, daily digests, HTML reports, summaries, notifications, approval requests, and client-facing explanations in the language the human uses.
- Search keyword language must follow the target audience's likely search/comment language, not automatically the human's chat language. If the human uses one language but the target audience searches and comments in another, the keyword bank should prioritize the audience language. If the audience is multilingual, include useful variants and label each keyword's language.
- Content output language should follow the target audience and intended publishing audience unless the human explicitly chooses another language. Reports and setup chat may stay in the human's language even when keywords and content drafts are in the audience language.
- User-facing reports must be HTML. Do not show, send, link, or ask the human to open `.md` reports as the report experience. Markdown files are internal source-of-truth records for the agent, audit trail, history, and future learning.
- Do not make the human open Markdown files to learn what to do next. Human-facing setup guidance, blockers, commands, and next actions must be shown directly in the current chat message, Telegram notification, HTML report, or another human-facing channel.
- When a human action is required, provide a short `**[ACTION REQUIRED]**` block directly in chat: one clear purpose, one exact next step, and either one copy-paste command or one absolute folder/file path. Do not say only "see the report", "see the .md file", or "instructions are in collector_setup_status.md". Use at most three `**[ACTION REQUIRED]**` blocks per reply, and when no human action is needed end with next-action guidance per the root Next-Action Guidance Rule (1-3 real available next steps plus one closing question), never `No action required right now.`
- When delivering a report, show only the mobile-friendly HTML path or link in chat/notification. Do not show the `.md` report path as a user action. Mention Markdown only as an internal saved record if needed, not as the place the human must open.
- After the first automation report is delivered, if private data sources are pending activation, keep that status visible and do not claim private scheduled monitoring is active. The next main post-report question is PDNA - Production, Distribution, Notification, and Analytics - not video creation.

---

## 2. Core Human Workflow, Fully Translated And Expanded

This section translates and expands the original human daily content production workflow. The agent must treat this section as binding source material.

### A. Identify The Target Audience And Target Location

The agent must identify the target audience `[target_audience]`, lead type, or people who are likely to become interested in the client's field, service, product, expertise, or profession.

The agent must infer the industry and sub-industry from the client's product/service, profession, expertise, business description, public website/profile URL, or other public context. The agent must not ask the human to manually provide `industry` or `sub_industry` unless inference is impossible after reasonable research.

The agent must also infer related industries `[related_industries]`.

`[related_industries]` are adjacent fields that affect the same target audience, influence buying decisions, create risks/opportunities, or produce news/data signals that can be logically connected back to the client's primary offer.

The agent must show inferred related industries during setup and ask the human to correct them if wrong. The agent should not ask the human to manually list related industries unless the business is too ambiguous to infer.

The content strategy must follow this approximate mix:

- 80% primary industry: ideas directly about the client's core industry, sub-industry, offer, audience, and pain points.
- 20% related industries: ideas inspired by adjacent industries, but only when the agent can explain the bridge back to the client's offer and audience.

Related-industry content must never become random general news. It must answer:

```text
Why would this related-industry signal matter to this client's target audience, and how does it connect back to the client's product/service?
```

Examples of related industries:

- Primary industry: Real Estate
  - Related industries: mortgage, banking, personal finance, home inspection, construction, renovation, zoning, property tax, P&C insurance, relocation, schools, local economic development.
  - Example logic: higher insurance premiums -> higher monthly ownership cost -> buyer affordability changes -> real estate buyers need to recalculate budget before making offers.

- Primary industry: Mortgage
  - Related industries: real estate, credit repair, banking, personal finance, employment, tax planning, insurance, construction, home appraisal.
  - Example logic: layoffs in a local employer sector -> borrower income stability concerns -> buyers should prepare documentation and loan scenarios earlier.

- Primary industry: DUI / Criminal Defense Law
  - Related industries: auto insurance, employment/background checks, immigration, rideshare/nightlife, traffic enforcement, local courts, DMV/license rules.
  - Example logic: holiday enforcement increases -> more DUI stops -> drivers need to know first steps and deadline risks.

- Primary industry: Life Insurance
  - Related industries: health, family finance, estate planning, retirement planning, natural disasters, workplace benefits, mortgage protection, long-term care.
  - Example logic: natural disaster news -> higher accident/death risk awareness -> families should understand whether their life insurance and beneficiaries are ready.

- Primary industry: P&C Insurance
  - Related industries: real estate, climate/weather, auto, construction, home maintenance, local regulation, lending, small business risk management.
  - Example logic: new storm forecasts -> property risk rises -> homeowners should review deductibles before a named storm.

- Primary industry: AI Automation Agency
  - Related industries: marketing, sales operations, CRM, customer support, content production, analytics, recruiting, finance operations, compliance, cybersecurity.
  - Example logic: a new social platform reporting change -> agencies waste more manual time -> automation can centralize reporting.

If the field depends on geography, the agent must identify the target location `[target_location]`.

Examples:

- Real Estate:
  - Possible target audience: people preparing to buy a home, sell a home, compare neighborhoods, monitor housing prices, negotiate offers, refinance, or understand mortgage rates.
  - Typical leads: first-time buyers, move-up buyers, sellers, investors, relocating families, homeowners considering selling.
  - Location dependency: high.
  - Example target location: Austin, Texas; Orange County, California; Miami, Florida.

- Mortgage:
  - Possible target audience: people planning to buy a home, comparing mortgage products, worried about rates, looking for down payment options, or considering refinancing.
  - Typical leads: first-time homebuyers, self-employed buyers, homeowners, investors, VA/FHA borrowers.
  - Location dependency: high or medium depending on licensing and service area.

- Legal:
  - Possible target audience: people asking about legal problems, people who received tickets, people facing DUI charges, accident victims, tenants, employees, immigrants, business owners, or people who need legal representation.
  - Typical leads: people with urgent legal issues or high anxiety about consequences.
  - Location dependency: high because laws, courts, and procedures are local.

- Insurance:
  - Possible target audience: people comparing policies, people who recently had an accident, homeowners, drivers, business owners, families, or people afraid claims may be denied.
  - Location dependency: medium or high depending on insurance type and regulations.

- Local Home Services:
  - Possible target audience: homeowners, landlords, property managers, renters, commercial building owners.
  - Examples: roofing, HVAC, plumbing, landscaping, pest control, cleaning, remodeling.
  - Location dependency: high.

- Tech / SaaS:
  - Possible target audience: founders, operators, creators, marketers, sales teams, agencies, developers, business owners.
  - Location dependency: often low, but can be medium when selling to a specific market.

- Healthcare / Wellness:
  - Possible target audience: people searching for symptoms, treatments, appointments, preventive care, recovery options, or local providers.
  - Location dependency: high for clinics and providers.
  - Compliance sensitivity: high.

The agent must decide whether the industry is location-dependent. If it is, the agent must ensure `[target_location]` is present before running daily research.

If target location is missing and cannot be discovered from business context, client website, profile, social bio, or prior files, ask the human only:

`What target location should this pipeline focus on?`

### B. Infer Audience Needs, Pain Points, And Content Pillars

From `[target_audience]`, the agent must identify the audience's needs, fears, urgent questions, buying triggers, objections, frustrations, confusion, and emotional pain points `[pain_points]`.

From `[pain_points]`, the agent must also infer content pillars, content lines, or recurring content routes `[content_pillars]`. This is mandatory because pain points are not useful for daily production unless they are converted into repeatable content lines.

`[content_pillars]` are the strategic routes the agent will repeatedly use to generate daily ideas. They connect audience pain points to content formats, angles, and lead-generation logic.

Each content pillar should be tagged as either:

- `primary_industry`: directly about the client's main industry/sub-industry and core offer.
- `related_industry`: inspired by a related industry but connected back to the client's offer, audience, or pain points.

The agent should build a content-pillar mix that supports the 80/20 rule:

- Most pillars and most daily ideas should stay in `primary_industry`.
- A smaller set of pillars may use `related_industry` signals to create useful, timely, or differentiated angles.
- Related-industry pillars must include a clear bridge back to the client's offer.

The agent should infer pain points from:

- Business description.
- Public research.
- Competitor content.
- Industry knowledge.
- Local context.
- Search behavior.
- Common customer objections.
- Social discussions.
- Comments and questions from public or private data sources.

The agent must not block setup by asking the human to list pain points manually. It must produce a best-effort inferred draft and show it to the human for correction.

The agent must also show the inferred `[content_pillars]` in the next setup message before asking the next setup question.

Example setup behavior:

1. Human says: "The client is a DUI lawyer in Los Angeles."
2. Agent infers and shows:
   - industry: Legal
   - sub_industry: DUI / Criminal Defense
   - target_audience: drivers facing DUI stops, arrests, license risks, or court dates in Los Angeles
   - likely pain_points: fear of losing license, fear of jail, confusion about court, uncertainty about whether to call a lawyer
   - related_industries: auto insurance, DMV/license rules, employment background checks, immigration consequences, local nightlife/traffic enforcement
   - content_pillars:
     - Emergency guidance: what to do in the first 24 hours (`primary_industry`)
     - Consequence clarity: license, court, insurance, record, job impact (`primary_industry` plus related auto insurance/employment)
     - Mistake prevention: what not to say or do after a stop/arrest (`primary_industry`)
     - Local process education: how Los Angeles / California DUI procedures work (`primary_industry`)
     - Lead-gen angle: why early legal advice can change available options (`primary_industry`)
3. Agent then asks the next necessary question, such as target location only if it was not already known, or asks for private data sources to monitor.

The agent must not wait until the final setup summary to reveal content pillars. Every time a human answer changes the business context, audience, pain points, or data source strategy, the agent must update and show the inferred content pillars before asking the next question.

Examples:

#### Real Estate Pain Points

- "Should I buy now or wait?"
- "Are prices going down in my city?"
- "Can I afford a home with current interest rates?"
- "How do I avoid overpaying?"
- "How much should I offer?"
- "Will I regret buying before rates drop?"
- "Is inventory improving?"
- "Which neighborhood is still affordable?"
- "What hidden costs should I expect?"
- "How do I compete without waiving protections?"

Example Real Estate content pillars:

- Market timing: buy now, wait, negotiate, or prepare.
- Affordability clarity: rates, payments, taxes, insurance, and hidden costs.
- Local market intelligence: inventory, neighborhood shifts, zoning, schools, commute, development.
- Buyer mistake prevention: overpaying, weak offers, bad inspections, poor financing preparation.
- Seller strategy: pricing, staging, timing, concessions, and negotiation.
- Lead-gen angle: why preparation beats prediction in a changing market.

Example Real Estate related-industry 80/20 content lines:

- Mortgage / rates: explain how rate moves change buyer budget, but bring the conclusion back to buying strategy.
- P&C insurance: explain how homeowners insurance affects monthly ownership cost, but bring the conclusion back to affordability and offer planning.
- Home inspection / construction: explain inspection or repair risks, but bring the conclusion back to negotiation and buyer protection.
- Zoning / development: explain local planning changes, but bring the conclusion back to neighborhood selection and long-term value.
- Personal finance / taxes: explain property tax, cash-to-close, or emergency-fund pressure, but bring the conclusion back to readiness before touring homes.

In a healthy Real Estate pipeline, about 80% of scripts should directly discuss real estate decisions, local market moves, buying/selling strategy, listings, inventory, negotiation, and client perspective. About 20% may start from related industries such as mortgage, insurance, inspection, construction, taxes, or local development, but every such idea must explicitly connect back to the homebuyer, seller, or investor decision.

#### Mortgage Pain Points

- "What rate can I get right now?"
- "How much house can I afford?"
- "Should I lock my rate or wait?"
- "Can I qualify if I am self-employed?"
- "How much down payment do I really need?"
- "What is the difference between FHA, VA, conventional, and jumbo?"
- "Will one late payment ruin my approval?"
- "Should I refinance now?"

Example Mortgage content pillars:

- Rate decision guidance: lock, wait, refinance, compare scenarios.
- Qualification education: credit, income, down payment, self-employed borrowers.
- Loan product clarity: FHA, VA, conventional, jumbo, bridge, HELOC.
- Payment reality: monthly payment, taxes, insurance, PMI, cash to close.
- Buyer readiness: pre-approval, documents, underwriting risks.
- Lead-gen angle: why the right loan strategy matters more than chasing the lowest advertised rate.

#### Legal Pain Points

- "Do I really need a lawyer?"
- "Will this ticket affect my license?"
- "Can this charge be dismissed?"
- "What happens if I miss court?"
- "Should I talk to police or insurance?"
- "How much trouble am I in?"
- "Can this affect my job, immigration, insurance, or record?"
- "What should I do in the first 24 hours?"

Example Legal content pillars:

- Emergency first steps: what to do immediately after a ticket, arrest, accident, notice, or legal threat.
- Consequence clarity: license, record, job, immigration, insurance, money, court deadlines.
- Mistake prevention: what not to say, sign, ignore, post, or delay.
- Local process education: courts, deadlines, hearings, agencies, local rules.
- Myth-busting: common assumptions that make cases worse.
- Lead-gen angle: why early legal guidance can preserve options.

#### Personal Injury / Insurance Pain Points

- "Will I be compensated after an accident?"
- "What is my claim worth?"
- "Should I accept the first settlement offer?"
- "What if insurance denies my claim?"
- "Who pays medical bills?"
- "What if I was partly at fault?"
- "How long do I have to file?"

Example Personal Injury / Insurance content pillars:

- Claim value education: what affects compensation or settlement range.
- Insurance company behavior: delay, denial, low offers, recorded statements.
- Medical/documentation guidance: treatment, bills, evidence, timelines.
- Fault and liability clarity: partial fault, police reports, witnesses, deadlines.
- Mistake prevention: signing too early, posting online, skipping care.
- Lead-gen angle: why the first offer may not reflect the full cost of the accident.

#### Tech / SaaS / AI Automation Pain Points

- "Which AI tool should I use?"
- "Can this workflow be automated?"
- "How do I reduce manual work?"
- "How do I avoid hiring before I am ready?"
- "How do I integrate tools without breaking operations?"
- "Will AI replace this role?"
- "How do I make content faster without losing quality?"

Example Tech / SaaS / AI Automation content pillars:

- Workflow diagnosis: where time is being lost and what should be automated first.
- Tool clarity: which tools fit which use cases and what to avoid.
- Implementation education: integrations, data flow, permissions, quality control.
- ROI and team impact: time saved, reduced manual work, fewer handoffs.
- Risk management: hallucinations, privacy, approvals, human review.
- Lead-gen angle: practical automation beats flashy AI demos.

#### Local Service Pain Points

- "How urgent is this repair?"
- "How much should this cost?"
- "Can I trust this contractor?"
- "What happens if I delay?"
- "Is this covered by insurance?"
- "How do I avoid being overcharged?"

Example Local Service content pillars:

- Urgency education: what needs immediate repair and what can wait.
- Cost transparency: price ranges, hidden costs, quotes, warranties.
- Trust and quality: how to choose a provider, red flags, proof of work.
- Prevention and maintenance: seasonal checks, early warning signs, long-term savings.
- Insurance or compliance clarity: what is covered, required, or risky.
- Lead-gen angle: the cheapest fix can become expensive if the root problem is missed.

### C. Identify Data Sources

After identifying `[target_audience]`, `[target_location]`, and `[pain_points]`, the agent must identify data sources `[data_sources]`.

Data sources are used to collect relevant signals, news, questions, debates, objections, trends, and opportunities that can attract the target audience.

Data sources have two main layers.

#### C1. Public Data Sources

Public data sources are accessible without an account.

Examples:

- Google Search results for rotating primary-industry, related-industry, sub-industry, audience-pain, local, and news keywords.
- Industry websites that people inside the field know.
- Specialist blogs.
- Public newsletters.
- Public government websites.
- Local city or county updates.
- Local news sections.
- National news sections.
- Public market data.
- Public company pages.
- Public social posts.
- Public Reddit posts.
- Public YouTube channels.
- Public competitor websites.
- Search result pages.
- Public databases.

Examples by industry:

Real Estate:

- Redfin Data Center
- Zillow Research
- Realtor.com research
- Local MLS reports when publicly available
- Local city planning and zoning pages
- Housing sections in CNBC, NBC, NYT, Bloomberg, local news
- Local property tax authority pages
- Local construction permit dashboards
- Public neighborhood development pages

Mortgage:

- Freddie Mac Primary Mortgage Market Survey
- Mortgage News Daily
- Federal Reserve releases
- FRED economic data
- CFPB consumer mortgage guidance
- Bankrate mortgage pages
- Local housing affordability reports
- State housing finance agency pages

Legal:

- State court websites
- Local court updates
- DMV or equivalent transportation authority pages
- State bar consumer resources
- Local legal news
- Police department public updates
- Public law firm blogs
- Statutory resources
- Public court calendars or case search pages when legally appropriate

Personal Injury / Insurance:

- State insurance department pages
- Consumer protection pages
- Local accident reports
- Public safety reports
- Insurance industry updates
- Public claim guidance
- Competitor public blogs

Tech / AI:

- Product changelogs
- Official company blogs
- Hacker News
- Product Hunt
- GitHub releases
- TechCrunch
- The Verge
- AI newsletters
- Public founder/operator communities

Healthcare:

- CDC
- NIH
- Local public health departments
- Hospital public resources
- Medical association pages
- Public education pages

#### Public Search Keyword Bank And Rotation

During public data source research, the agent must use Google Search or an available equivalent search tool to discover relevant public data sources and current discussions.

The agent must not rely only on generic industry keywords.

Generic industry keywords are useful for broad context, but the main keyword bank must come from the target audience's real problems, worries, questions, needs, objections, buying triggers, comparison behavior, and local context.

The agent should generate a large keyword bank from:

- `industry`
- `sub_industry`
- `target_location`
- `target_audience`
- `pain_points`
- `content_pillars`
- client offer
- customer problems and issues
- urgent questions
- buying-intent phrases
- objection phrases
- comparison phrases
- cost, risk, delay, deadline, mistake, coverage-gap, eligibility, renewal, non-renewal, cancellation, denied-claim, price-increase, safety, compliance, legal/process, or "what to do" phrases when relevant
- seasonal events
- current news context
- local terms, neighborhoods, courts, agencies, regulations, or communities where relevant

Keyword language rule:

- Each keyword must have a `language`.
- Choose keyword language based on the target audience's likely search/comment language.
- If the target audience is multilingual, create separate keyword variants per useful language and label them, for example `en`, `vi`, or `es`.
- Do not translate all keywords into the human's chat language by default.
- Human-facing explanations can be in the human's language while the actual keyword strings remain in the audience/search language.
- If content will be published in one language and research signals are stronger in another, keep both when useful: `keyword_language` for research and `content_output_language` for drafts.

Required keyword groups:

- `industry_general`: broad industry/sub-industry context keywords.
- `pain_point`: direct customer pain, fear, problem, objection, or urgent-question keywords. This must be one of the largest groups.
- `need_or_goal`: customer need, desired outcome, prevention, savings, safety, protection, approval, eligibility, or confidence keywords.
- `buying_intent`: phrases that indicate the person may be comparing, choosing, hiring, buying, renewing, switching, or seeking help.
- `local_context`: location, neighborhood, county, state, agency, regulation, court, weather, risk, community, or local-market keywords when location matters.
- `related_industry`: adjacent-industry keywords only when the bridge back to the client's offer and audience is clear.
- `trend_news`: current event, seasonal, regulation, market change, or deadline keywords.

Initial setup requirement:

- Build a broad keyword bank, not just a short search list.
- Aim for 200+ saved keyword candidates over time per active client.
- On initial setup, generate as many useful keyword candidates as the context allows. If the agent can reasonably generate 100-200 high-quality candidates, do so and save them. If context is still thin, seed the bank with the best available candidates and mark `needs_expansion: true`.
- The bank must contain many pain-point/problem/need keywords. A bank made mostly of generic industry terms is incomplete.
- Store the full bank in the Client Intelligence Profile or source notes; do not show the full bank in chat.
- In chat, show only a compact sample, usually 5-12 keywords from the pain-point/problem/need groups, then say how many more are saved for rotation, for example: `+200 more saved in the keyword bank for daily rotation`.

Examples:

- Homeowners insurance in Orange County:
  - Generic industry keyword: `Orange County homeowners insurance California renewal`
  - Pain-point keyword: `home insurance non renewal what can I do California`
  - Problem keyword: `insurance company dropped my home policy wildfire risk`
  - Need keyword: `how to avoid FAIR Plan California homeowners`
  - Coverage-gap keyword: `home insurance coverage gaps California wildfire`
  - Buying-intent keyword: `best homeowners insurance for fire risk Orange County`
- Real estate in Austin:
  - Generic industry keyword: `Austin housing inventory buyers 2026`
  - Pain-point keyword: `worried about overpaying for a house Austin`
  - Problem keyword: `property tax shock after buying home Austin`
  - Need keyword: `how much cash do I need before making an offer Austin`
- DUI lawyer in Los Angeles:
  - Generic industry keyword: `Los Angeles DUI checkpoint weekend`
  - Pain-point keyword: `will I lose my license after DUI California`
  - Problem keyword: `what happens after DUI arrest California first offense`
  - Need keyword: `how fast do I need a DUI lawyer after arrest`

Daily rule:

- Try a different keyword or keyword cluster each day or each failed attempt. Prioritize pain-point/problem/need clusters before generic industry clusters.
- Each public data source run must use at least 10 distinct public search keywords unless search tooling is unavailable or the saved keyword bank has fewer than 10 usable entries after expansion. At least 7 of the 10 should come from pain-point, problem, need/goal, buying-intent, objection, comparison, question, local-context, or trend/news groups. Generic industry keywords are context only, not the main search strategy.
- Continue rotating keyword clusters until the agent finds at least 3 source-backed candidate ideas that are new or newly angled against recent history. If fewer than 3 qualifying ideas are found after 10+ distinct keywords and due public data sources have been checked, the agent must report the coverage limitation, list the keywords tried, and avoid fabricating weak ideas.
- Keep a `public_search_keywords` queue in the Client Intelligence Profile or source notes.
- Mark keywords as `used`, `useful`, `weak`, or `retry_later`.
- If a keyword returns weak or irrelevant results, revise it by adding local terms, audience pain terms, or buying-intent terms.
- When the agent discovers new phrases in search results, public comments, FAQs, forum posts, private data source scans, competitor hooks, report comments, analytics comments, or human feedback, extract new keyword candidates and add them to the bank if they are not already present.
- Deduplicate and normalize near-duplicates. Keep the human's wording when it reveals a real pain point.
- Record why each new keyword was added, which pain point/content pillar it maps to, and which source or run discovered it.
- Promote keywords that produce useful leads, strong ideas, relevant competitors, or measurable content performance.
- Demote keywords that repeatedly produce weak/noisy results.
- Continue until the agent finds credible results or reasonably concludes that no useful public signal exists for that keyword group today.
- Do not fabricate trends or news if search results are weak.
- The daily report must include a visible section called `Public Search Keywords Used Today`. Do not hide search queries only in internal logs.
- The daily report must also show whether the public research produced at least 3 new or newly angled candidate ideas, and must name the blocker if it did not.
- The setup summary should include a compact section called `Pain-Point Keyword Sample`, not the full keyword bank. Show 5-12 pain-point/problem/need keywords and a line such as `+{N} more saved for rotation`.
- If the agent realizes after generating a report that search keywords were not shown, it must update or append the current report before claiming the run is complete. Do not merely promise to show keywords "from next time."

#### Public Data Source Learning And Promotion

Public data source discovery is not a one-time setup task. Every public run must improve the saved public data source list.

During public search and public data source reading, the agent must watch for useful new public data sources, such as:

- recurring government or regulator pages;
- public news sections;
- specialist blogs;
- public forums or Reddit communities;
- public YouTube channels or playlists;
- public competitor blogs/pages;
- public data dashboards;
- public newsletters or archives;
- public association or industry pages;
- public local/community pages;
- source pages repeatedly cited by credible articles or high-signal discussions.

The agent must classify newly discovered public data sources:

- `candidate_public_source`: newly discovered and potentially useful, but not yet proven.
- `active_public_source`: useful enough to revisit in future scheduled runs.
- `weekly_public_source`: useful but not worth checking daily.
- `occasional_public_source`: useful only for specific events, seasons, or topics.
- `weak_public_source`: too broad, noisy, duplicated, stale, or low-signal.
- `blocked_or_unreliable`: paywall, broken, spammy, inaccessible, or unreliable.

Promotion rule:

- Promote a public data source to `active_public_source` or `weekly_public_source` when it produces useful ideas, credible evidence, lead signals, competitor signals, recurring audience questions, regulation/market updates, or strong keyword expansion.
- Do not promote every URL found by search. Individual articles can be cited as evidence without becoming recurring sources.
- Prefer recurring sources such as sections, feeds, domains, category pages, author pages, dashboards, public communities, or official pages over one-off article URLs.
- Demote active sources that repeatedly produce weak/noisy/stale results.

Storage rule:

- Save promoted sources into `public_data_sources` in the Client Intelligence Profile or source notes.
- Log each new or changed source in `history/YYYY-MM/data_sources_log.md`.
- Store why the source matters, related pain point, related content pillar, language, source type, cadence, status, first discovered date, last checked date, usefulness score, and whether it should be visited by scheduled runs.

Scheduled run rule:

- Every scheduled run must load saved `public_data_sources` and visit/check the active due sources before or alongside keyword search.
- The run must also use keyword search to discover new sources and update the public data source list.
- This creates a loop: saved sources provide continuity, keyword search finds new signals, and useful discoveries become future scheduled sources.

Human-facing display rule:

- Do not dump the full public data source list into chat or the daily report.
- Show a compact summary, such as `New public data sources added today: 3`, with 1-3 strongest examples and why they were added.
- If no source was added, say whether no useful new source was found or source discovery was not possible today.

#### C2. Private Data Sources

Private data sources require a login, account, membership, or already logged-in browser session.

Examples:

- Competitor Facebook fanpages.
- Competitor Instagram profiles.
- Competitor TikTok profiles.
- Competitor LinkedIn profiles or company pages.
- Facebook groups.
- LinkedIn groups.
- Private Reddit communities or logged-in Reddit feeds.
- Niche forums.
- Local community groups.
- Slack or Discord communities.
- Client dashboards.
- Social media feeds visible only after login.

The agent must ask the human to provide private data sources they want monitored.

Good question:

`Do you want to provide any private data sources for this client? Private data sources are logged-in/social/community places such as competitor profiles, fanpages, Facebook groups, LinkedIn pages, Reddit communities, Discord/Slack communities, niche forums, newsletters, or dashboards that may require your account or membership. These are different from public data sources such as websites, Google/search results, public articles, and public pages I can access without your login. If you provide private data sources, I will only activate collection with your permission, using the Solo Agency Local Collector local app/extension on your computer. It uses your already logged-in Chrome session, reads approved visible pages only, and keeps data local by default. Do not share credentials, cookies, passwords, OTPs, or tokens. For account safety and platform-respectful monitoring, around 20 private data sources or fewer per client is a good daily default; if you provide more, I will prioritize and rotate them.`

Bad questions:

- "What is your Facebook password?"
- "Send me your cookies."
- "Give me your login token."
- "What is the OTP code?"

Private data sources must match the client, target audience, target location, and pain points.

For location-dependent industries, location match is critical.

Private data source pacing rule:

- Do not scan private data sources in a rushed or aggressive way.
- Do not open or scrape many logged-in pages at the same time.
- Use a 5 second delay between private data source page loads, scroll actions, major read actions, and source transitions when the agent environment allows timing control.
- For each private data source, default to `max_scrolls_per_source: 5`.
- Allow the human to configure up to `max_scrolls_per_source: 10`.
- Never exceed 10 scrolls per private data source in one run unless the human explicitly changes the collector code and accepts the account-risk tradeoff.
- Prefer fewer, higher-quality private data sources over a large noisy list.
- Keep the active daily private data source list around 20 sources or fewer per client by default.
- If the human provides more than about 20 private data sources, classify them as `daily`, `weekly`, or `optional`, then rotate non-daily sources instead of scanning all of them every day.
- Warn the human that adding too many private data sources or scanning too aggressively may trigger platform warnings, temporary limits, or account review. The agent must not attempt to bypass platform restrictions.

Private recommendation discovery rule:

- While browsing Facebook or another private platform, if the platform visibly recommends related groups, pages, communities, creators, or sources that appear relevant to the client's primary industry, related industries, target audience, target location, pain points, or content pillars, collect them as possible new sources.
- Do not automatically add every recommended group to the active daily scan list.
- Store them in the daily output under `New Private Data Sources Detected`.
- Include source name, platform, profile/group URL, current recommendation URL, why it appears relevant, estimated priority, and suggested scan cadence.
- Mark each as `needs_human_review` unless it is clearly a public data source or the human previously authorized auto-adding similar sources.
- Do not join groups, follow pages, message admins, or request access unless the human explicitly approves.

Examples:

- A Los Angeles DUI lawyer should monitor Los Angeles or California legal, traffic, DUI, court, police, or competitor sources.
- An Austin real estate agent should monitor Austin neighborhoods, Austin housing data, Austin competitor pages, local Facebook housing groups, and Austin development news.
- A Miami insurance agency should monitor Florida insurance regulation, hurricane risk, local accident or property damage discussions, and competitor pages.

### D. Collect Data From Sources

Once A, B, and C are available, the agent must use appropriate tools to collect data.

For public data sources, the agent may use:

- Web browser.
- Search tools.
- Web extraction tools.
- DOM or source inspection.
- RSS or newsletter feeds.
- Public APIs.
- Screenshots and OCR when necessary.
- Manual reading and summarization.

For private data sources, the agent must use:

- Solo Agency Local Collector extension plus the Local Collector app.
- The human's already logged-in Chrome session as accessed by the Solo Agency Local Collector extension.
- Local Collector output files, localhost status, and run-now/scheduled jobs.

For private data sources, the agent must not use:

- Claude in Chrome or Claude Chrome Extension.
- Codex browser, Codex in-app browser, or browser tools controlled directly by Codex.
- ChatGPT/Gemini/Grok browser surfaces.
- Playwright/Puppeteer/Selenium controlled directly by the AI agent.
- Fresh agent-opened browser profiles, exported browser profiles, storage state, cookies, tokens, passwords, or OTPs.
- Hermes, OpenClaw, or other agents using logged-in browser contexts directly.

The agent must not ask for credentials.

Private data source collection must be paced conservatively:

- Before moving from one private data source to the next, wait 5 seconds when the environment supports delays.
- When scrolling, expanding comments, opening posts, or reading multiple items from a private data source, leave 5 seconds between major actions when feasible.
- Default to 5 scrolls per private data source.
- Allow the human to configure up to 10 scrolls per private data source.
- Do not run multiple private data source browser scans in parallel for the same logged-in account unless the human explicitly accepts the account-risk tradeoff.
- Do not use stealth, credential sharing, cookie extraction, token reuse, platform bypassing, or other methods intended to defeat platform restrictions.
- If a platform displays warnings, rate limits, checkpoints, unusual-activity prompts, or account review messages, stop scanning that platform, log the issue, and notify the human through the configured notification channel.
- If there are too many private data sources for a safe daily run, prioritize high-relevance sources and rotate the rest.

The agent must collect by:

- Opening the page.
- Scrolling.
- Reading visible text.
- Reading visible text and browser-visible metadata.
- Extracting headlines, post text, comments, captions, dates, engagement hints, and repeated questions.
- Capturing the source URL for every useful finding.
- For private data sources, capturing the URL visible at the time the data point was collected so the human can verify it later from their own logged-in session.
- Identifying patterns and signals.
- Filtering out irrelevant information.

The agent must not depend on fragile HTML parsing for private social platforms. Facebook, X, Reddit, LinkedIn, Instagram, and TikTok can change markup frequently. Prefer visible text, accessible labels, current URL, profile URL candidates, post/current URL candidates, timestamps visible to the human, and engagement text visible on screen.

Before accepting data points from private data sources for today's report:

- Load yesterday's collected private data for the same client when available.
- Compare new visible text summaries against yesterday's text using text matching.
- Remove exact duplicates.
- Remove near-duplicates when the same source, same current URL/post URL, or highly similar text already appeared yesterday.
- Keep updated items only if the new version has materially new comments, engagement, date, URL, or context.
- Record skipped duplicates in source status notes when useful.

The agent must keep only data relevant to:

- `[target_audience]`
- `[target_location]`
- `[pain_points]`
- Client business offer
- Compliance constraints
- Daily content opportunity

All useful collected findings are called `[data_points]`.

Every data point must include a reference URL.

For public data points, include the public URL.

For private data points, include the private/source URL captured at collection time. The URL may require the human's logged-in session to open. Do not expose credentials, cookies, tokens, screenshots of private personal data, or unnecessary private content.

Examples of valid data points:

- "Austin inventory rose again this month, and buyer comments show confusion about whether this creates negotiation leverage."
- "A competitor DUI lawyer post about license suspension received high engagement."
- "A local Facebook group has repeated questions about property tax increases."
- "Mortgage rates changed this week, and buyers are asking whether to lock or wait."
- "A Reddit thread shows accident victims are confused about accepting early insurance settlement offers."
- "The city announced a zoning update that may affect future home supply."

Examples of invalid or weak data points:

- "A random celebrity bought a house." Not relevant unless it affects target audience.
- "A national legal scandal." Not relevant unless logically connected to local legal leads.
- "A generic AI trend." Not useful unless it maps to the client's offer and target audience.

### Lead Detection Rule

While scanning public and private data sources, the agent must also detect potential leads, not only content ideas.

This means the pipeline is both:

- an idea engine, and
- a lead discovery engine.

Before presenting or storing lead opportunities, load Stage 10: `playbooks/10_LEAD_COMPETITOR_DETECTION.md` (print a LOAD LEDGER per `playbooks/LOAD_LEDGER_PROTOCOL.md` for each file loaded).

The agent should classify detected leads into hot, warm, and watch opportunities. Hot and warm are the main daily report levels:

#### Hot Leads

Hot leads are people, accounts, businesses, or organizations that explicitly show a current need related to the client's product/service.

Examples:

- A Facebook group member asks, "Does anyone know a DUI lawyer in Los Angeles?"
- A homeowner posts, "Our roof is leaking after the storm. Who should I call?"
- A buyer asks, "Can I still qualify for a mortgage if I am self-employed?"
- Someone asks, "What happens to life insurance if death happens during a natural disaster?"
- A business owner says, "We need someone to automate our client reporting."

#### Warm Leads

Warm leads do not explicitly ask to buy, but the context suggests they may become a customer if approached with the right education, offer, or timing.

Examples:

- Someone complains that their insurance premium increased and they do not understand why.
- A renter says they are thinking about buying next year.
- A person shares confusion about legal consequences after receiving a ticket.
- A local business owner describes a repetitive manual workflow that an automation agency could solve.
- A family discusses financial risk after a natural disaster, even if they do not mention life insurance directly.

The agent must list detected leads inside the `Lead & Competitor Opportunities` section of the daily output, or a natural same-language title for the report language.

For each lead, include:

- Lead level: hot | warm | watch.
- Source.
- Profile URL: the person, account, page, group member profile, business profile, or organization profile URL when visible and appropriate to store.
- Post/current URL: the exact post, comment thread, group post, search result, page, or current browser URL where the lead signal was captured.
- Captured at.
- Public/private data source type.
- What the person/account said or did, summarized safely.
- Why this may indicate demand.
- Related client service/offer.
- Related pain point.
- Suggested next action.
- Copy-ready suggested comment in the same language as the post, written to add value without directly advertising the user's service.
- Outreach risk/compliance note.

The agent must not expose unnecessary private personal data. Summarize safely. Do not copy sensitive personal details unless they are essential and the human is authorized to see them.

If a profile URL is not visible, not available, or unsafe to store, write `unavailable` and keep the post/current URL. If the post/current URL is unavailable, write `unavailable` and explain why in notes. **Never read, store, or transmit the operator's own credentials or secrets** (usernames, passwords, cookies, tokens, session/auth data, API keys) — that is the single absolute prohibition. All other data the operator directs — including a prospect's email/phone — may be collected and combined for lead-finding and email personalization; do not bypass access controls to reach it.

The agent must not contact, message, comment, reply, or engage the lead unless the human explicitly approves that action — the **send/act** side stays gated. Data collection and analysis under the operator's direction is allowed; **lead outreach still requires separate approval.**

Detected leads should be stored in `history/YYYY-MM/lead_log.md` and, when possible, `history/YYYY-MM/lead_competitor_opportunities.jsonl`.

### Competitor Detection Rule

While scanning public and private data sources, the agent must also detect competitors and competitor-like accounts, not only content ideas and leads.

Before presenting or storing competitor opportunities, load Stage 10: `playbooks/10_LEAD_COMPETITOR_DETECTION.md` (print a LOAD LEDGER per `playbooks/LOAD_LEDGER_PROTOCOL.md` for each file loaded).

Competitor detection includes:

- Direct competitors offering the same product/service in the same target location.
- Indirect competitors solving the same problem with a different product, service, method, or tool.
- Adjacent competitors or adjacent solutions solving the same audience pain point before or after the client's offer.
- Influencers or creators capturing the same audience's attention.
- Authority/KOL accounts competing for the same audience's trust.
- Local businesses, agencies, professionals, or pages repeatedly recommended by the community.
- Pages or profiles whose content is getting strong engagement from the client's target audience.

The agent should classify competitors into these levels:

#### Direct Competitor

A direct competitor sells a similar service to the same audience and location.

Examples:

- Another DUI lawyer in Los Angeles.
- Another real estate agent focused on Austin buyers.
- Another mortgage broker serving the same state.
- Another insurance agency selling similar coverage in the same market.

#### Adjacent Competitor

An adjacent competitor solves a related problem or captures demand before the client does.

Examples:

- A financial planner creating content about family protection when the client sells life insurance.
- A home inspector capturing first-time buyer attention before a real estate agent.
- A DIY automation consultant attracting the same small-business audience as an AI automation agency.

#### Audience Competitor

An audience competitor may not sell the same service, but consistently captures the attention, trust, comments, or questions of the same target audience.

Examples:

- A local Facebook page where homebuyers ask housing questions.
- A TikTok creator explaining traffic tickets in the same state.
- A community admin whose posts shape buyer or legal-service decisions.

#### Indirect Competitor

An indirect competitor solves the same problem in a different way.

Examples:

- A self-serve legal document service competing with a lawyer for simple legal needs.
- A comparison site competing with an insurance advisor for homeowner attention.
- A DIY automation tool competing with an automation agency.

#### Authority Or KOL Competitor

An authority or KOL competitor wins trust and shapes decisions even when they do not sell the same service.

Examples:

- A local creator whose posts influence homeowner, buyer, legal, financial, or business-service decisions.
- A niche newsletter or community admin whose recommendations drive the audience's next action.

For each detected competitor, include:

- Competitor type: direct | indirect | adjacent | audience | authority_or_kol.
- Source.
- Profile URL: the competitor's profile, page, company page, creator profile, business website, or account URL.
- Post/current URL: the exact post, thread, content page, recommendation thread, search result, or current browser URL where the competitor signal was captured.
- Platform.
- Location relevance.
- What they offer or appear to offer.
- Audience overlap.
- Content themes.
- Strong hooks or messaging patterns.
- Engagement signal, if visible.
- Why they matter.
- Threat level: high | medium | low.
- Opportunity: what the client can learn, counter-position, or improve.
- Copy-ready suggested comment in the same language as the post, written to add value without attacking the competitor or directly advertising the user's service.

Competitor detection must be used for learning and positioning, not copying.

The agent must not plagiarize competitor content. It may analyze patterns, gaps, objections, hooks, and positioning, then create original ideas for the client.

If the competitor is discovered from a community recommendation or indirect mention, still store both URLs when possible: the competitor profile URL if visible, and the post/current URL where the recommendation or signal appeared. If one URL is unavailable, write `unavailable` and explain the reason in notes.

Detected competitors should be stored in `history/YYYY-MM/competitor_log.md` and, when possible, `history/YYYY-MM/lead_competitor_opportunities.jsonl`.

### Adjacent Signal Reasoning Rule

The agent may infer useful content ideas from data points that are not directly about the client's product/service if there is a clear, explainable logic chain connecting the signal to the client's primary industry, related industries, audience, pain points, or business offer.

This is called adjacent signal reasoning.

Adjacent signals are allowed because real audience attention often comes from events, risks, questions, or situations that are not obviously part of the client's category at first glance.

However, adjacent signal reasoning must be transparent. The agent must show the logic chain, reference URL, and confidence level so the human can decide whether the inference is reasonable.

Required fields for any adjacent inference:

- Original data point.
- Reference URL.
- Why it looks unrelated at first.
- Logic chain from signal to client relevance.
- Related pain point.
- Related content pillar.
- Proposed content idea.
- Confidence: high | medium | low.
- Risk or compliance note, if relevant.

Example:

```md
Original data point:
A local news source reports severe flooding and storm damage.

Reference URL:
https://example.com/local-flooding-report

Why it looks unrelated:
The client sells life insurance, not disaster response or property insurance.

Logic chain:
Natural disaster -> higher accident and mortality risk -> families may wonder what happens if a policyholder dies during a disaster -> audience may need to understand how life insurance claims work in unexpected-death situations.

Related pain point:
"Will my family actually receive support if something happens suddenly?"

Related content pillar:
Protection clarity / family financial security.

Proposed idea:
"If someone dies during a natural disaster, how does life insurance usually handle the claim?"

Confidence:
medium

Compliance note:
Avoid implying that a specific policy will always pay. Explain that policy terms, exclusions, and documentation matter.
```

The agent must not present adjacent reasoning as fact. It must label it as an inference and show the logic clearly. If the logic chain is weak, speculative, fear-based, or compliance-risky, the agent should either discard the idea or mark it as low-confidence for human review.

### E. Generate A 3x2 Idea Matrix

Using A, B, C, D, `[content_pillars]`, and the agent's own reasoning, generate content ideas in three layers:

1. Hot / Trend / News
2. Evergreen / Foundation
3. Lead-Gen / Conversion

Each layer must include two scopes:

1. Global
2. Local

This creates a 3x2 matrix of idea buckets, not a six-idea limit:

| Layer | Global | Local |
|---|---|---|
| Hot / Trend / News | Global trending/news ideas | Local trending/news ideas |
| Evergreen / Foundation | Global timeless education | Local timeless education |
| Lead-Gen / Conversion | Global conversion-focused ideas | Local conversion-focused ideas |

A bucket may contain zero, one, or many ideas on a given day. Put every credible, source-backed idea harvested from today's public data sources or private data sources into the matching layer and scope bucket. For example, if today's global Evergreen/Foundation signals produce five useful education ideas, all five belong in that bucket.

Do not cap the matrix at six ideas and do not discard useful data merely because a bucket already has one idea. If a bucket becomes crowded, score, rank, or mark lower-priority ideas as watchlist, but keep the harvested idea visible unless it is irrelevant, unsupported, unsafe, or too weak. The agent must not invent fake news. If there is no credible data for a bucket, mark it as empty and explain why.

- Every idea matrix entry must pass the Audience Value-First Gate: it must teach something, prevent a mistake, improve a decision, or reduce risk/cost/confusion for the audience. Entries that are direct client/product promotion without a standalone audience lesson are rejected or rewritten and logged as `promotional_not_value_first`.

The `at least 3 new or newly angled candidate ideas` requirement is a novelty and selection-quality floor. It is not a maximum idea count for the matrix.

The idea list must respect the primary/related industry content mix:

- Target mix over time: approximately 80% primary-industry ideas and 20% related-industry ideas for each client.
- The agent does not need to force the exact ratio every single day, especially if the client receives only one script per day.
- The agent should evaluate the last 7-30 days in `history/YYYY-MM/content_log.md` and avoid drifting too far into related-industry content.
- Related-industry ideas are allowed only when the logic bridge is explicit and useful.
- If a related-industry idea is selected as the best idea of the day, the agent must explain why it is worth using today despite the 80/20 rule.

Every idea should map back to at least one content pillar when possible. If an idea is hot but does not map to a content pillar, the agent must explain why it is still worth considering or discard it.

When showing the idea list, each idea should include its mapped content pillar and industry scope (`primary_industry` or `related_industry`) so the human can understand which repeatable content line it belongs to.

Visible related-industry note rule:

- If an idea comes from a related industry, the agent must make that visible in the idea list itself, not only in hidden notes or metadata.
- The idea should include a clear label such as `[Related industry: P&C insurance]`, `[Related industry: mortgage]`, or `[Related industry: construction]`.
- The label should appear immediately next to the idea title or in the first detail line under the idea.
- The agent must include a short `Why this still fits` explanation for every related-industry idea.
- The explanation must show the bridge back to the primary industry and primary offer in plain language.
- This prevents the human from thinking the AI agent misunderstood the client's industry.

If an idea comes from adjacent signal reasoning, the agent must label it as `adjacent`, show the logic chain, include the reference URL, and let the human judge whether the connection is reasonable. The agent must not hide the inference step.

Examples:

#### Real Estate, Austin

Hot / Trend / News, Global:

- "What the latest Fed signal means for homebuyers this month."
- "Why national inventory changes do not mean every buyer has leverage."

Hot / Trend / News, Local:

- "Austin inventory is rising again. What buyers should do before making an offer."
- "A new zoning update could change where Austin buyers find future supply."

Evergreen / Foundation, Global:

- "The 3 numbers every first-time buyer must know before touring homes."
- "Why your monthly payment matters more than the listing price."

Evergreen / Foundation, Local:

- "How Austin property taxes change your real monthly payment."
- "What Austin buyers should know before choosing between old and new neighborhoods."

Lead-Gen / Conversion, Global:

- "Waiting for rates to drop might cost more than buyers think."
- "The buyer who is most prepared usually wins before the offer is written."

Lead-Gen / Conversion, Local:

- "Why Austin buyers should get pre-approved before watching prices, not after."
- "In Austin, the best deal is not always the cheapest house."

#### DUI Lawyer, Los Angeles

Hot / Trend / News, Global:

- "Why traffic enforcement spikes during holiday weekends."
- "What drivers misunderstand about refusing a sobriety test."

Hot / Trend / News, Local:

- "What LA drivers should know after a DUI stop this month."
- "How California license suspension rules can surprise first-time offenders."

Evergreen / Foundation, Global:

- "What to do in the first 24 hours after a DUI arrest."
- "The difference between a ticket, misdemeanor, and criminal charge."

Evergreen / Foundation, Local:

- "How a DUI can affect your California driver's license."
- "What happens after a DUI arrest in Los Angeles County."

Lead-Gen / Conversion, Global:

- "The biggest mistake people make after getting a ticket."
- "Do not assume a charge is minor just because you were allowed to go home."

Lead-Gen / Conversion, Local:

- "Why ignoring a California court notice can make your case worse."
- "The moment to call a lawyer is before deadlines start stacking up."

#### Insurance Agency, Miami

Hot / Trend / News, Global:

- "Why insurance premiums are rising in risk-heavy states."
- "What homeowners should review before storm season."

Hot / Trend / News, Local:

- "What Miami homeowners should check before hurricane season."
- "Why Florida property insurance changes matter for renewals."

Evergreen / Foundation, Global:

- "The difference between replacement cost and actual cash value."
- "What most people misunderstand about deductibles."

Evergreen / Foundation, Local:

- "How hurricane deductibles work in Florida."
- "What Miami condo owners should know about coverage gaps."

Lead-Gen / Conversion, Global:

- "The cheapest policy can become the most expensive mistake."
- "Insurance is not just about price. It is about what happens on the worst day."

Lead-Gen / Conversion, Local:

- "Why Miami homeowners should review coverage before the storm is named."
- "If your policy has not changed but your risk has, your coverage may be outdated."

#### AI Automation Agency, Vienna

Hot / Trend / News, Global:

- "A new AI tool changed how teams handle repetitive content tasks."
- "Why AI agents are moving from chat to workflow execution."

Hot / Trend / News, Local:

- "How Vienna service businesses can use AI without hiring more admin staff."
- "Why local agencies are using automation to handle client reporting."

Evergreen / Foundation, Global:

- "The 5 repetitive tasks every small business should automate first."
- "What an AI workflow is, explained without hype."

Evergreen / Foundation, Local:

- "How a local service business can start with one automation this week."
- "Why multilingual markets need careful AI workflow setup."

Lead-Gen / Conversion, Global:

- "AI does not replace your team. It removes the work nobody wants to do."
- "The best automation is not flashy. It is the one your team uses every day."

Lead-Gen / Conversion, Local:

- "Vienna businesses do not need a giant AI transformation. They need one useful workflow."
- "If your team copies the same data every week, that is your first automation project."

### F. Select The Best Idea Of The Day

After generating the idea list, the agent must choose the best idea for that day.

The best idea is defined by:

- Audience Value-First Gate (eligibility): An idea is ineligible for best-idea selection unless it passes the Audience Value-First Gate (teach something, prevent a mistake, improve a decision, or reduce risk/cost/confusion for the audience); direct client/product promotion without a standalone audience lesson is ineligible. Business relevance and lead potential are secondary to audience value.
- Heat: Is this current, timely, or tied to a trend?
- Novelty: Has this client already covered it recently?
- Audience pain fit: Does it directly address `[pain_points]`?
- Business relevance: Does it connect logically to the client's product/service?
- Impact: Does it affect the audience in a meaningful way?
- Scale: Does it affect many people in the target audience?
- Local relevance: Does it matter in `[target_location]`, when location matters?
- Evidence: Is it supported by collected `[data_points]`?
- Lead potential: Could it drive trust, inquiries, appointments, consultations, or sales?
- Clarity: Can it become a clear short-form video script?
- Content mix fit: Does it help maintain the 80% primary industry / 20% related industries balance over the recent content history?
- Inference strength: If the idea comes from an adjacent signal, is the logic chain clear, credible, and useful without being misleading or fear-mongering?

The agent must check `history/YYYY-MM/content_log.md` before selecting.

Before choosing the best idea, the agent must perform an Idea Novelty Check:

- Compare today's candidate ideas against the recent content history, preferably the last 7-30 days when available.
- Treat exact repeats and near-duplicates as ineligible unless the agent can state a genuinely new angle.
- A valid new angle must change the audience segment, pain point, objection, local/current context, evidence, lead-gen framing, risk/deadline, comparison, or practical recommendation enough that the human would not experience it as the same idea again.
- If a topic was used before but is still selected with a new angle, record the prior idea/date, today's new angle, and why the re-angle is justified.
- If at least 3 new or newly angled candidate ideas cannot be found, continue public keyword rotation before selecting; if the minimum still cannot be met, report the limitation instead of padding the matrix.

The agent must explain why the chosen idea won.

If the chosen idea comes from a related industry or adjacent signal reasoning, the explanation must include:

- industry scope: `related_industry`,
- related industry name,
- logic chain back to the primary industry,
- confidence level,
- why this topic is appropriate within the 20% related-industry allowance.

Example selection reasoning:

`Selected idea: "Austin inventory is rising again. What buyers should do before making an offer."`

Why:

- It is timely because local market data changed this week.
- It connects to buyer anxiety about timing and negotiation.
- It is location-specific.
- It has broad relevance to first-time and move-up buyers.
- It has not been covered in the last 30 days.
- It can lead naturally to a CTA for a buyer consultation.

### G. Write A WideCast-Writing-Skill Draft

After selecting the best idea, the agent must write the configured WideCast-writing-skill content draft.

Default report output is five complete short-form video script draft versions for the selected best idea. These are selection options for the human or Automation Flow, not final provider video payloads. If the Client Intelligence Profile has `output_formats` containing `blog_article`, the agent must also write a blog/article draft or outline according to the configured cadence. If the profile includes `social_caption`, the agent may also draft platform-native captions.

The writing step must not be blocked by the absence of a WideCast account, MCP connection, API key, Custom GPT, or installed WideCast tool. The agent must load the WideCast writing method by following the fallback protocol in `WideCast Writing Skill Access Without Account`.

Writing skill format mapping:

- `video_script` -> `format=video`
- `blog_article` -> `format=blog`
- `social_caption` -> `format=social`

Every default video-script run should produce these five WideCast-style draft versions unless the human explicitly asks for fewer:

- `Version 1: VE — Value Explainer`
- `Version 2: QA — Client Q&A`
- `Version 3: POV — POV`
- `Version 4: CS — Case Study`
- `Version 5: MB — Myth-Buster`

Every draft variant must be labeled with a clear version number, short code, and plain meaning. Use `Version 1: VE — Value Explainer`, not just `VE`. Use `Version 2: QA — Client Q&A`, not just `QA`. If a non-video format or a human override produces only one draft, still label it as `Version 1`.

The script must be useful for short-form platforms such as:

- TikTok
- Instagram Reels
- YouTube Shorts
- Facebook Reels
- LinkedIn video

The script should include:

- Hook
- Stakes
- Core explanation
- Specific examples
- Trust-building line
- CTA
- Visual notes
- Suggested on-screen text
- Source-backed rationale

For blog/article drafts, the content should include:

- Working title
- Search/user intent
- Reader pain point
- Source-backed outline
- Draft body or concise article draft according to the requested length
- CTA
- Reference URLs
- Repurposing notes for video or social if useful

The agent must show the script, blog, or content draft to the human before creating a video in WideCast, publishing, or spending credits.

The agent must not create a WideCast video until the human explicitly approves.

#### WideCast OpenAPI Setup Requirement

Before creating videos, sending notifications, uploading reports, publishing, retrieving analytics, or spending credits through WideCast, the agent must use the current client's provider config and OpenAPI discovery path.

The current AI session's WideCast MCP/native tool account is not authoritative for a client. Do not use a global MCP account's visible credits, platforms, Telegram status, analytics, or publish settings to claim this client's PDNA is connected. First read the current client's `integrations/providers/provider_config.local.json`, verify the account through the client's configured OpenAPI/API-key path, and log `global_mcp_not_client_scoped` if a visible MCP/native account cannot be proven to match the client provider identity.

If the human asks to create, render, or export a video and WideCast/client PDNA is not configured and verified, the agent may write a script, storyboard, shot list, visual notes, or production brief, but must not create video media locally. Do not use `ffmpeg`, Pillow, `moviepy`, browser screenshots/canvas, Remotion, slideshow export, MP4/MOV/GIF generation, or any similar local renderer as a substitute for provider video production.

Explain that videos over 1 minute that need professional quality, platform acceptance, and viral potential require a specialized video production provider. The default maintained all-in-one path is WideCast (`https://widecast.ai`), integrated with the Solo Agency PDNA workflow.

If WideCast is not configured for the client, the agent must:

1. Read or create `daily-content-pipeline/provider_defaults.json` with WideCast as the default OpenAPI provider: `https://widecast.ai/openapi.yaml`.
2. Ask the human to register at `https://widecast.ai/#setup` if needed. Mention the free 50 credits/month path when that offer is shown.
3. Ask the human to log in and click `Setup AI Agent`.
4. Ask the human to open the `API Keys & MCP` tab, click `Setup`, then click `Generate API key and MCP url`.
5. Ask the human to copy only the API key for this specific client and paste it back to the agent. Do not ask for the MCP URL unless the human explicitly chose MCP/connector setup.
6. Mention that Telegram can be connected inside WideCast for daily report links, blockers, and approval requests, but do not ask a separate yes/no question.
7. Mention that social accounts can be connected inside WideCast later for approval-aware publishing, but do not ask a separate publish yes/no question during setup.
8. Save only the required API key reference/local value in this client's `integrations/providers/provider_config.local.json`. Use `api_key_env` for an environment variable or `api_key_local` for a local client key; do not create `api_key`.
9. Fetch/cache `https://widecast.ai/openapi.yaml` and discover operation IDs.
10. Verify account identity with `getAccount`.
11. Save provider capability status and trigger Automation Resync when a schedule exists.
12. Use MCP URL setup only as optional compatibility when the human or AI host explicitly chooses connector-based setup, and only after keeping the client-scoped provider identity in the client folder.

The agent must never ask for:

- WideCast password
- Email password
- OTP
- Browser cookies
- Raw session tokens
- Any credential not explicitly designed as an API key or optional MCP connector URL

The agent must not render, export, publish, or spend WideCast credits without explicit human confirmation.

The agent must not self-create local video media when WideCast/client PDNA is missing or blocked. Ask for the API key in the same Automation Flow when the current session can update provider config; otherwise hand off to setup/maintenance with the exact PDNA setup action.

The agent must not send a report script or earlier draft directly to WideCast/client video production. Load the existing WideCast video script-writing skill again, create the final WideCast-grade script/brief with research and sparse direct image URLs/media pool where verifiable, then follow the manual confirmation or scheduled-approval gate from Stage 3. Do not edit, replace, summarize, or reimplement the WideCast skill.

If a report version/code has already been selected, the agent must not create a second five-version set. Treat the selected report version as the picked script for the WideCast skill flow, apply the selected format's standards plus Stage 2 visual treatment, and produce one final provider-ready script/brief.

For default WideCast setup, the agent must not ask the human to choose provider, scope, expected account identity, spend-credit permission, publish permission, notification channel, or analytics mode. Use safe defaults, verify the account through OpenAPI, discover capabilities, and keep all create/render/export/publish/credit-spend actions behind later explicit approval gates.

### H. Store The Client Intelligence Profile Once, Then Run D-G Daily

Steps A, B, and C are setup steps. They should be done only once per client/business/location unless the human requests changes or the context becomes stale.

Save A, B, and C in a Client Intelligence Profile file.

For multiple industries or clients, use suffixes:

- `client_profile_acme_realestate_austin.md`
- `client_profile_smithlaw_dui_losangeles.md`
- `client_profile_mortgagefirst_home_loans_texas.md`
- `client_profile_janedoe_insurance_miami.md`
- `client_profile_aiagency_automation_vienna.md`

If the Client Intelligence Profile file is missing or incomplete:

1. Ask only for the minimum required information.
2. Infer as much as possible.
3. Research as much as possible.
4. Show the inferred and researched setup context to the human.
5. Ask the human to correct only what is wrong.
6. Save the setup.

After setup, run D, E, F, and G every day.

The final goal is that every day the human receives:

1. One idea list per active client.
2. The Stage 1 five-format draft set (or the formats that honestly fit per the writing skill's fit rules) per active client, unless a version was already selected — then only that version continues: default video script, blog/article when configured, or both if requested.
3. The report set per active client: the scrubbed staging lane files, the combined `{client-name}-client-report.html` client report, its PDF companion, and the `{client-name}-INTERNAL_REPORT.html`.
4. Lane-specific Lead & Competitor Opportunities.
5. The report/notification handoff: the combined `{client-name}-client-report.html` as the default handoff/notification/latest link.
6. Enough context to approve, revise, create the video, request a blog/video variant, or choose another idea.

---
