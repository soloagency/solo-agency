# Solo Agency Scheduled Run Entrypoint

Use this file as the scheduler prompt for unattended daily runs.

The scheduled prompt should be short, but it must be explicit enough to force the agent to load the real playbooks instead of improvising from memory.

## Scheduler Prompt

```text
Run the scheduled Solo Agency daily run now.

1. Load SOLO_AGENCY_PLAYBOOK.md from the local workspace or from the configured GitHub raw URL.
2. Follow the Scheduled Run Playbook Loading Contract in playbooks/04_DAILY_SCHEDULE.md.
3. Do not rely on memory from setup. Load the required child playbooks again at run time.
3F. Obey `playbooks/LOAD_LEDGER_PROTOCOL.md` full-load discipline: every stage/child playbook must be read to its last line and pass a LOAD LEDGER (compare `playbooks/LOAD_MANIFEST.md` when present; ledger each named dependency) BEFORE you act on it. A truncated / "output too large" / partial read = NOT loaded — re-read in chunks. Do not run any client/report/scan/production step without a `Verdict: PASS loaded-in-full` ledger for the stage(s) it needs.
4. Read `daily-content-pipeline/automation/automation_manifest.md`, `daily-content-pipeline/automation/scheduled_run_prompt.md` when present, `daily-content-pipeline/automation/github_issues.md` when present, `daily-content-pipeline/automation/update_state.json` when present, `daily-content-pipeline/provider_defaults.json` when present, and `daily-content-pipeline/schedule.md` before processing clients. If the manifest says `automation_prompt_update_pending`, report that blocker and still run from the latest local playbooks/profile/config instead of the stale snapshot. If tracked GitHub issues have new maintainer/community responses that fix a blocker, apply the fix and resync automation before claiming the run is current.
4A. If any blocker, repeated failure, unexpected behavior, stale artifact, tool/config mismatch, or dead end occurs, perform Last-Resort Recovery before declaring the run blocked: load `playbooks/11_UPDATE_AND_VERSION_WATCH.md`, check GitHub `main` for a newer Solo Agency playbook/code version, reload the latest relevant instructions, retry if the newer rule resolves it, and if still blocked create, send, or draft a redacted issue tracked in `daily-content-pipeline/automation/github_issues.md` without requiring the human to have a GitHub account.
4B. If this scheduled task is `Solo Agency - GitHub Update Watch` or the run request is an update/upgrade/sync-latest request, load `playbooks/11_UPDATE_AND_VERSION_WATCH.md` and run only the update-watch workflow. Do not process clients, run reports, scan public data sources, scan private data sources, create production assets, publish, or scan analytics from this update-watch task.
5. Process every active client in daily-content-pipeline/clients_index.md.
6. Do not ask setup questions when the saved Client Intelligence Profile is complete.
7. Run public research first, then private scans if active, then published-URL analytics only when published URLs/metrics exist and the configured provider/URL inspection path allows it, analysis, Lead & Competitor Opportunities, audience-value-first idea matrix, best idea selection, production-ready draft options, final WideCast video-script skill pass before any video provider request, approved video/blog/social asset creation when provider setup and explicit approval allow it, video scene editing after approved provider video creation returns reviewable scenes, client-facing HTML report generation, operator-only INTERNAL_REPORT generation, mandatory PDF companion generation, learning updates, and notification. Every idea and draft must teach the target audience something useful before it benefits the client's brand; reject or rewrite direct product-praise ideas as `promotional_not_value_first`. Before client-facing HTML report generation or PDF companion packaging, load `playbooks/skills/report-design/SKILL.md` and use `tools/solo_report_renderer.py` by default instead of writing ad hoc report/PDF scripts. Before any tool or capability check for video/video scene editing/blog/social/provider/upload/notification/publishing/analytics, check Client tools first and global MCP/native tools second, and put those details in `INTERNAL_REPORT`, not client-facing files. Before any video provider creation request, load and apply `playbooks/skills/video-script-writing/SKILL.md` through the verified client provider writing-skill operation when available or from repo-local/static fallback. If a report version/code, pasted edited version, or automation recommended/approved version already exists, produce only that one selected final WideCast-grade script or production brief; do not generate five new versions again. If no selected/recommended version exists, generate the skill's Stage 1 options first and obtain a pick/recommendation before provider creation. Run the skill's research and Stage 2 inline-media/direct-image/video-URL workflow as far as the runtime can verify, record any research/visual blocker, never fabricate facts or image/video URLs, do not edit/reimplement the skill, and use only that skill-produced final script/brief as the provider payload. In authorized scheduled Automation Flow, do not ask for a second confirmation after this skill pass when valid video-creation approval already exists; if approval is missing, stop at `approval_required`. Before any video/blog/social provider or video scene-editing action, load `playbooks/SOLO_AGENCY_VIDEO_PROVIDER_ADAPTER.md` after any vendored writing/provider/video-editing skill and resolve the current client's provider config/OpenAPI capability instead of using global MCP state. If the verified client-scoped provider or required video operation is missing, still produce the final WideCast-grade script/storyboard/production brief, then stop at a PDNA setup `**[ACTION REQUIRED]**` block asking only for the WideCast API key by default; do not ask provider/scope/spend/publish/account-identity questions and do not create local video media with `ffmpeg`, Pillow, `moviepy`, browser/canvas screenshots, Remotion, slideshow export, or any other local renderer. If video scenes are created, load `playbooks/skills/video-editing/SKILL.md` through the verified client provider `getEditingSkill` operation when available or from local repo files as fallback, run the editing skill pass, and stop at a fresh render/export approval gate before final MP4 export. Public data source intelligence and private data source intelligence must be separate full staging lane reports, then packaged into one canonical `{client-name}-client-report.html` handoff file.
8. If private data sources are active, pending, requested, approved, present in a profile, or if any current-workspace collector status files exist, load `playbooks/PRIVATE_SOURCE_GATE.md`, `playbooks/02_PRIVATE_SOURCE_SETUP.md`, `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md`, and `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md`; use only the Solo Agency Local Collector extension plus Local Collector app. Do not use Claude in Chrome, Codex/browser tools, Playwright, or any agent-controlled browser for private data sources.
9. Do not trust saved config/status labels alone, including `public_data_sources_only`, `pending_private_activation`, or `private sources postponed`. Before skipping private data sources, perform Collector Runtime Verification: try the bridge status API, verify current-workspace identity, and read local collector health/output status files when the API is unreachable from the AI sandbox.
10. If `GET http://127.0.0.1:17321/status` works, verify `/status.config_file`, `/status.output_dir`, and `/status.run_now_request_file` point to the current setup's `daily-content-pipeline/collector/` tree before trusting the bridge. If they point elsewhere, mark `wrong_workspace_bridge`, skip private collection, and tell the human to run the current setup's Local Collector command and remove/disable old Solo Agency Local Collector extensions in `chrome://extensions`.
11. If `GET http://127.0.0.1:17321/status` fails, do not immediately conclude the Local Collector is inactive. Some AI scheduled tasks run in a sandbox where `127.0.0.1` is not the human machine's localhost. Read `daily-content-pipeline/collector/inbox/bridge_health.json`, `daily-content-pipeline/collector/inbox/collector_status.json`, `daily-content-pipeline/collector/collector_setup_status.md`, recent `daily-content-pipeline/collector/inbox/YYYY-MM/*/collector_status.json`, and any recent consumed run-now status files. If those files show a recent healthy current-workspace bridge/extension, use the file-based run-now queue described in Stage 8 by writing one unique job file per client under `daily-content-pipeline/collector/jobs/pending/`; if they are stale/missing, mark `collector_status_unverified` or the exact blocker and continue public data sources.
12. If private data sources may be collectable, read the Local Collector config before announcing scan depth. For the first lead/competitor pass for a client/source set, use 10 scrolls per approved source when safe. For normal daily scheduled runs, use the safe default: 5 scrolls per approved source, max 10, about 5 seconds between scrolls.
13. If no private data sources are active after Collector Runtime Verification, do not block the scheduled run. Continue public data source research, but include the exact verification outcome in the HTML report/notification: `private data sources not configured`, `private data sources pending activation`, `collector_status_unverified`, `collector_offline`, `wrong_workspace_bridge`, `extension_stale`, or `Private Data Source Discovery Recommended/Declined/Postponed`.
14. If published content exists, retrieve yesterday's and last-7-day published URLs, inspect each URL when authorized, record metrics/comment signals/learnings, and mark unavailable metrics honestly.
15. If no published URLs/metrics exist yet, mark measurement as `no published URLs yet`; do not pretend the measurement-learning loop ran.
16. Every human-facing reply, notification, or report handoff must include `Solo Agency daily run progress` with completed/current/remaining steps and blockers. If sending multiple updates, show updated progress each time. Also include an `Automation freshness check` that states whether the latest changes are synced into the automation/scheduled task prompt/contract/playbook/source state, not only config, and whether tomorrow's run will load the newest state. If the human must answer, approve, fix a blocker, reload an extension, reconnect a provider, edit an automation task, approve publishing/credit spend, or run a command, put that request in a standalone `**[ACTION REQUIRED]**` block. If no human action is needed, say `No action required right now.`
17. Canonical client-facing reports must be one combined HTML file and client-blind. Markdown is internal. A PDF companion is mandatory after the combined HTML is created or updated. Load Stage 6 plus `playbooks/skills/report-design/SKILL.md`, then use `tools/solo_report_renderer.py package` to generate `{client-name}-client-report.html`, `{client-name}-client-report.pdf`, `outputs/latest/{client-name}-client-report.html`, and `outputs/latest/{client-name}-client-report.pdf` from the three scrubbed staging HTML files, or record the exact PDF blocker if export/redaction is not safe.
18. Load Stage 6 and obey the Single Client Report Contract With Lane Staging: maintain `{client-name}-public-data-sources-report.html`, `{client-name}-private-data-sources-report.html`, and `{client-name}-daily-report.html` as staging inputs, then rebuild `{client-name}-client-report.html` as the only default client/human handoff file. Use `tools/solo_report_renderer.py render` by default for generated/updated staging HTML reports.
19. If public data sources finish before private data sources, generate/update only `{client-name}-public-data-sources-report.html` and `{client-name}-daily-report.html` with private status pending/blocked, then rebuild `{client-name}-client-report.html` and PDF. If private data sources later finish, generate/update only `{client-name}-private-data-sources-report.html` and `{client-name}-daily-report.html`; do not rewrite the public report; then rebuild `{client-name}-client-report.html` and PDF. After the private pass reaches a terminal state, reconcile private status and counts across the private report, daily index, combined client report, internal source record, report state JSON, notification log, and `outputs/latest/` copies; remove stale `scan in progress`, `partial`, `pending`, or old count text before handoff.
20. Generate/update `{client-name}-INTERNAL_REPORT.html` and `outputs/latest/{client-name}-INTERNAL_REPORT.html`, labeled `INTERNAL_REPORT - Not for client sharing`, with provider/PDNA, WideCast, Telegram/social platform, Local Collector, automation, API-key/config, delivery, blocker, recovery, and count-reconciliation details.
21. Run the Client-Blind Scrub Gate on client-facing HTML/PDF before calling them client-ready. They must not mention Solo Agency, WideCast, PDNA/provider tooling, OpenAPI, MCP, Local Collector, Chrome extension, automation/scheduled task, API key/config, Telegram, agent/tool/debug details, or `INTERNAL_REPORT`.
22. Two notifications are acceptable: one for `public_report_ready` and one for `private_report_ready` or `private_report_blocked`. Notifications to the user/operator should normally point to `{client-name}-client-report.html` or its uploaded operator-delivery URL, include the PDF companion path/status, include `{client-name}-INTERNAL_REPORT.html` as an operator-only secondary link/path, and state the lane status.
23. Run the Provider Report Delivery Capability Check for every report notification or report handoff and record the details in `INTERNAL_REPORT`. Check Client tools first: read this client's provider config, fetch/cache the provider OpenAPI spec, verify account identity, and inspect operation IDs before declaring upload or notification unavailable. WideCast default operations are `uploadAsset` for HTML/PDF report upload when supported and `sendTelegramMessage` for Telegram/email fallback notification. Check global MCP/native tools only after the client tool identity is proven to match this client.
24. Prepare a report-delivery record for every report notification/handoff: lane status, local `.html` path, local `.pdf` path or PDF blocker, local `INTERNAL_REPORT` path/status, client-facing scrub status, provider, provider discovery checked true/false, account verification status, upload operation ID, notification operation ID, upload attempted true/false, uploaded HTML/PDF URLs if available, upload blocker if any, notification channel, notification attempted true/false, and final notification report link.
25. If WideCast OpenAPI notification/Telegram/email fallback is configured and `uploadAsset` supports `text/html`, upload the current scrubbed `{client-name}-client-report.html` to WideCast before each report notification for user/operator delivery, then send that uploaded URL through `sendTelegramMessage`. If the verified client provider also supports PDF upload, upload the PDF companion and include its URL; otherwise include the local PDF companion path/status. Do not send only a local file path when an uploaded HTML URL is available for the operator. Treat uploaded URLs as short-lived operator handoff links when the provider says they have a TTL; the local report remains the permanent archive. Provider-hosted URLs are not client-share links because the URL/domain may reveal the provider.
26. If provider config is missing, auth fails, OpenAPI discovery fails, account identity mismatches, the only visible provider account is a global MCP/native account that is not client-scoped, or the required upload/notification operation is missing, log the provider-neutral blocker plus any useful legacy WideCast alias in `INTERNAL_REPORT`, and notify/surface the best available HTML path/link plus PDF companion path/status plus INTERNAL_REPORT path/status through chat or another authorized fallback.
27. Never send a report-ready notification without an HTML report URL/path and PDF companion status. If that happens by mistake, immediately send a correction notification with the HTML report URL/path plus PDF status, then log the correction.
28. Load playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md before claiming the scheduled run is complete.
```

## Latest Override: Client-Specific Automation Prompt

For the current architecture, prefer one automation task per client. The task name must begin with the client name:

```text
{Client Name} - Solo Agency Daily Run
```

Client-specific automation prompts must include `target_client_slug` and should follow this contract:

```text
Run Solo Agency daily run for target_client_slug="{client_slug}" only.

Load SOLO_AGENCY_PLAYBOOK.md and the required stage playbooks.
Read daily-content-pipeline/clients_index.md and verify the target client is active.
Do not process any other client.
Read only this client's Client Intelligence Profile, public data sources, private data sources approval state, history, collector inbox, outputs, analytics, and publishing state.
If any blocker, repeated failure, stale artifact, or instruction/tool mismatch occurs, treat stale Solo Agency playbooks/code as the first suspect: check GitHub main, reload the latest relevant playbook instructions, and only then declare the run blocked. If still blocked, create, send, or draft a redacted issue without requiring the human to have a GitHub account, and track it in daily-content-pipeline/automation/github_issues.md.
Read only this client's provider config under integrations/providers/. For every tool or capability check, check Client tools first and global MCP/native tools second: read this client's provider config, OpenAPI cache, verified identity, and provider_capabilities.json before inspecting any global MCP/native tool list. If PDNA actions, analytics, report upload, notification, video creation, scene editing, render/export, or publishing are needed, load playbooks/SOLO_AGENCY_VIDEO_PROVIDER_ADAPTER.md when production/video actions are involved, fetch/cache the configured provider OpenAPI spec, verify the provider account belongs to this client, and stop provider actions on account mismatch. Before any video creation request, load and apply the existing WideCast video script-writing skill to produce the final provider payload from report/draft context with research and inline-media/direct-image/video-URL workflow; if a report version/code or automation recommendation already exists, process only that selected version and do not generate five new versions again. Do not edit or reimplement the skill. In scheduled automation this skill pass does not require a second human confirmation when valid video-creation approval already exists. If the verified client provider or required video operation is missing, do not create local video media; still save the final WideCast-grade script/production brief, then ask only for the WideCast API key by default in this automation session when config can be updated, otherwise hand off to setup/maintenance. Do not ask provider/scope/spend/publish/account-identity questions for the default path. If a global MCP/native provider account is visible but cannot be proven to match this client, log `global_mcp_not_client_scoped` and do not use its credits, platforms, notifications, analytics, or publish settings as this client's state.
For private data sources, use only the extension_instance_id configured for this client in daily-content-pipeline/collector/extension_registry.json.
If localhost is unavailable from the automation sandbox, read local bridge/extension health files and write a per-client job file under daily-content-pipeline/collector/jobs/pending/.
With one shared bridge, private data source collection is parallel per client identity, bound to the claiming extension instance. Different client Chrome profiles can collect at the same scheduled time, while jobs for the same client/profile stay queued and sequential until `/complete` or TTL.
The collector output path must be daily-content-pipeline/collector/inbox/YYYY-MM/{client_slug}/{run_id}/.
If the matching client extension is missing, stale, or wrong, continue with public data sources only and report the exact blocker.
Generate this client's canonical three-file client-facing HTML report set using Stage 6 plus `playbooks/skills/report-design/SKILL.md` and `tools/solo_report_renderer.py` by default, generate/update `{client-name}-INTERNAL_REPORT.html`, generate/update the mandatory PDF companion from the scrubbed client-facing HTML set with `tools/solo_report_renderer.py package`, update this client's history/learning, and resync any configuration changes made during the run.
For report delivery, prefer this client's configured provider. WideCast defaults to OpenAPI discovery at https://widecast.ai/openapi.yaml and operations uploadAsset + sendTelegramMessage.
If private data sources complete after public data sources, create/update only {client-name}-private-data-sources-report.html and {client-name}-daily-report.html; preserve {client-name}-public-data-sources-report.html exactly except for explicit link repair. Reconcile status/counts across the private report, daily index, internal source record, INTERNAL_REPORT, report state JSON, notification log, and outputs/latest copies before handoff.
```

If the prompt contains a `target_client_slug`, the scheduled agent must not loop through every active client. The older all-clients loop applies only to an explicitly named all-clients/master task.

Optional master digest task:

```text
Solo Agency Master Digest - All Clients
```

The master digest task must not scan private data sources. It only reads existing client reports/outputs and creates a summary.

## Optional GitHub Update Watch Task Prompt

Use this prompt for the maintenance task named:

```text
Solo Agency - GitHub Update Watch
```

The task checks whether Solo Agency has changed upstream and keeps installed playbooks/code/templates aligned with GitHub. It must not run any client report or collection workflow.

```text
Run Solo Agency GitHub update watch now.

Load SOLO_AGENCY_PLAYBOOK.md, playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md, playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md, and playbooks/11_UPDATE_AND_VERSION_WATCH.md.
Read daily-content-pipeline/automation/update_state.json when present, daily-content-pipeline/automation/automation_manifest.md when present, daily-content-pipeline/automation/scheduled_run_prompt.md when present, daily-content-pipeline/automation/github_issues.md when present, and daily-content-pipeline/schedule.md when present.
Check https://github.com/soloagency/solo-agency main using the Stage 11 Fresh GitHub Checkout Protocol or a safe remote commit check.
Compare the installed/local commit with GitHub main.
If there is no new commit, update update_state.json and update_log.md with the check result, then stop.
If there is a new commit, compare root instructions, playbooks, provider/OpenAPI tooling, Local Collector bridge/runtime, Chrome extension templates, setup scripts, templates, and automation contracts.
Classify the change as no_change, playbook_only, provider_tooling, collector_bridge, chrome_extension, collector_bridge_and_extension, setup_or_schedule_contract, breaking_or_major_behavior, or unknown.
If auto_apply_approved is true in update_state.json, apply the update from a verified fresh checkout, preserve secrets/client data/private captures/history/outputs, update every configured client and extension folder as needed, resync automation/scheduled task prompts, and update update_state.json, update_log.md, automation_manifest.md, scheduled_run_prompt.md, and resync_log.md.
If auto_apply_approved is false, do not apply the update. Write daily-content-pipeline/automation/update_notice.md with the classification and the question of whether to apply. Surface the same notice in the native task output or setup/maintenance chat when available. Do not send Telegram, WideCast/email-fallback, provider notification, social post, or client notification for update-watch.
If bridge/runtime files changed, include the exact current-setup command the human must run outside the AI sandbox.
If extension files changed, include the exact extension folder path and Chrome reload/Load unpacked steps for each client profile.
Do not process client reports, public data sources, private data sources, video/blog/social production, publishing, notifications, Telegram, or analytics in this update-watch task.
```

## Required Runtime Loads

At the start of every scheduled run, load:

- `SOLO_AGENCY_PLAYBOOK.md`
- `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`
- `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`
- `playbooks/04_DAILY_SCHEDULE.md`
- `daily-content-pipeline/automation/automation_manifest.md` when present
- `daily-content-pipeline/provider_defaults.json` when present
- `daily-content-pipeline/schedule.md` when present
- `daily-content-pipeline/automation/github_issues.md` when present
- `daily-content-pipeline/automation/update_state.json` when present
- the target client's `integrations/providers/provider_config.local.json` when PDNA, analytics, report delivery, notification, production, or publishing is needed

Then conditionally load:

- `playbooks/01_BASIC_PROFILE_PUBLIC_REPORT.md` only if setup repair is needed.
- `playbooks/PRIVATE_SOURCE_GATE.md`, `playbooks/02_PRIVATE_SOURCE_SETUP.md`, `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md`, and `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md` when private data sources are active, pending, blocked, missing-but-discovery-recommended, or being scanned.
- `playbooks/03_PRODUCTION_DISTRIBUTION.md` when drafts, production, publishing, provider setup, notification, or provider report delivery are needed.
- `playbooks/skills/video-script-writing/SKILL.md` and the modules it requires before any video provider creation request, even when PDNA is missing and the run can only produce a final script/storyboard/production brief.
- `playbooks/SOLO_AGENCY_VIDEO_PROVIDER_ADAPTER.md` after any vendored writing/provider/video-editing skill when video creation, scene editing, credit checks, production uploads, render/export, publishing, notification, analytics, or provider account actions are needed.
- `playbooks/skills/video-editing/SKILL.md` after approved video creation returns reviewable scenes, when a provider video needs scene audit/fix, or before asking for final render/export approval.
- `playbooks/05_MEASURE_LEARN_IMPROVE.md` when any published content exists or yesterday/last-7-day measurement is due.
- `playbooks/06_AGENCY_REPORT_STANDARD.md` whenever creating or delivering HTML reports.
- `playbooks/skills/report-design/SKILL.md` immediately after Stage 6 and before writing, repairing, or packaging report HTML/PDF.
- `playbooks/10_LEAD_COMPETITOR_DETECTION.md` whenever detecting, scoring, reporting, storing, or improving lead and competitor opportunities. This is normally required for every daily run.
- `playbooks/11_UPDATE_AND_VERSION_WATCH.md` whenever the run is the `Solo Agency - GitHub Update Watch` task, the human asked for update/upgrade/sync latest, or Last-Resort Recovery checks the newest GitHub version.
- `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md` before claiming completion.

## Scheduled Run Difference From First Setup

First setup asks the minimum setup questions because profile/config/history do not exist yet.

Scheduled runs should not ask those questions again. They should read saved state, run automatically, and interrupt the human only for approval gates, blockers, expired sessions, missing critical data, production/render/publish/credit decisions, or lead outreach decisions. Any such interruption must use the `**[ACTION REQUIRED]**` block from the root playbook.

## Notification Requirement

If the target client's provider notification is configured:

1. Generate or update the local `{client-name}-daily-report.html` staging cover/index plus any affected lane report after loading Stage 6 and `playbooks/skills/report-design/SKILL.md`; use `tools/solo_report_renderer.py render` by default.
2. Generate or update `{client-name}-INTERNAL_REPORT.html` and `outputs/latest/{client-name}-INTERNAL_REPORT.html`.
3. Run the Client-Blind Scrub Gate on the staging HTML files and final combined package.
4. Generate or update the mandatory `{client-name}-client-report.html` and `{client-name}-client-report.pdf` companion from the current scrubbed three-file staging HTML set using `tools/solo_report_renderer.py package`, or record the exact PDF blocker. The combined HTML is the only default report handoff/upload file.
5. Load the target client's provider config and `daily-content-pipeline/provider_defaults.json`.
6. Fetch/cache the provider OpenAPI spec if needed.
7. Verify the provider account identity before using account actions. For WideCast, call `getAccount`.
8. Inspect Client tools first for a report/file/asset upload API that supports HTML/PDF and a notification send operation. For WideCast, use `uploadAsset` with `text/html` and `sendTelegramMessage`; upload PDF too only when the verified client provider supports it.
9. Upload the combined `{client-name}-client-report.html` to the provider for user/operator delivery when an HTML-capable upload operation exists. Provider-hosted URLs are not client-share links.
10. Capture the returned public or reviewable report URL and its TTL if the provider returns one.
11. Send that uploaded URL through the provider notification channel with the run summary, lane status, PDF companion path/status, INTERNAL_REPORT path/status, blockers, lead/competitor counts, and next action.
12. If upload is unavailable or fails but notification is available, send the best available local/hosted `.html` report path/link plus PDF companion path/status plus INTERNAL_REPORT path/status through provider notification and clearly state the upload blocker.
13. Log the upload attempt and notification in `daily-content-pipeline/notifications/notification_log.md`, including lane status, provider, operation IDs, upload URL, PDF path/status, INTERNAL_REPORT path/status, client-facing scrub status, TTL if known, and blocker if any.
14. If the notification or handoff needs the human to act, end with one `**[ACTION REQUIRED]**` block. If it is purely informational, include `No action required right now.`

If both public and private notifications are sent, both notifications should link to the same combined `{client-name}-client-report.html` path or uploaded URL. Lane-specific public/private staging links should be omitted unless the human explicitly requests diagnostics.

If provider config is missing, auth fails, OpenAPI discovery fails, account verification mismatches, or the required provider operation is missing, do not pretend upload or notification succeeded. Log the exact provider-neutral blocker, send/surface the best available HTML path/link plus PDF companion path/status plus INTERNAL_REPORT path/status, and state the blocker clearly.

Do not claim WideCast itself lacks upload or notification capability merely because a legacy MCP/tool surface is unavailable. Check Client tools first: the configured OpenAPI provider path, account identity, and provider_capabilities.json. Only then check global MCP/native tools as optional compatibility. Conversely, do not claim this client's WideCast PDNA is connected merely because a legacy MCP/tool surface is available.
