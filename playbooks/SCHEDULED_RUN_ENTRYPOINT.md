# Solo Agency Scheduled Run Entrypoint

Use this file as the scheduler prompt for unattended daily runs.

The scheduled prompt should be short, but it must be explicit enough to force the agent to load the real playbooks instead of improvising from memory.

## Scheduler Prompt

```text
Run the scheduled Solo Agency daily run now.

1. Load SOLO_AGENCY_PLAYBOOK.md from the local workspace or from the configured GitHub raw URL.
2. Follow the Scheduled Run Playbook Loading Contract in playbooks/04_DAILY_SCHEDULE.md.
3. Do not rely on memory from setup. Load the required child playbooks again at run time.
4. Read `daily-content-pipeline/automation/automation_manifest.md`, `daily-content-pipeline/automation/scheduled_run_prompt.md` when present, and `daily-content-pipeline/schedule.md` before processing clients. If the manifest says `automation_prompt_update_pending`, report that blocker and still run from the latest local playbooks/profile/config instead of the stale snapshot.
5. Process every active client in daily-content-pipeline/clients_index.md.
6. Do not ask setup questions when the saved Client Intelligence Profile is complete.
7. Run public research, private scans if active, published-URL analytics only when published URLs/metrics exist, analysis, Lead & Competitor Opportunities, idea matrix, best idea selection, production-ready drafts, approved video/blog/social asset creation when provider setup and explicit approval allow it, HTML report generation, learning updates, and notification.
8. If private data sources are active, pending, requested, approved, present in a profile, or if any current-workspace collector status files exist, load `playbooks/PRIVATE_SOURCE_GATE.md`, `playbooks/02_PRIVATE_SOURCE_SETUP.md`, `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md`, and `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md`; use only the Solo Agency Local Collector extension plus Local Collector app. Do not use Claude in Chrome, Codex/browser tools, Playwright, or any agent-controlled browser for private data sources.
9. Do not trust saved config/status labels alone, including `public_data_sources_only`, `pending_private_activation`, or `private sources postponed`. Before skipping private data sources, perform Collector Runtime Verification: try the bridge status API, verify current-workspace identity, and read local collector health/output status files when the API is unreachable from the AI sandbox.
10. If `GET http://127.0.0.1:17321/status` works, verify `/status.config_file`, `/status.output_dir`, and `/status.run_now_request_file` point to the current setup's `daily-content-pipeline/collector/` tree before trusting the bridge. If they point elsewhere, mark `wrong_workspace_bridge`, skip private collection, and tell the human to run the current setup's Local Collector command and remove/disable old Solo Agency Local Collector extensions in `chrome://extensions`.
11. If `GET http://127.0.0.1:17321/status` fails, do not immediately conclude the Local Collector is inactive. Some AI scheduled tasks run in a sandbox where `127.0.0.1` is not the human machine's localhost. Read `daily-content-pipeline/collector/inbox/bridge_health.json`, `daily-content-pipeline/collector/inbox/collector_status.json`, `daily-content-pipeline/collector/collector_setup_status.md`, recent `daily-content-pipeline/collector/inbox/YYYY-MM/*/collector_status.json`, and any recent consumed run-now status files. If those files show a recent healthy current-workspace bridge/extension, use the file-based run-now path described in Stage 8; if they are stale/missing, mark `collector_status_unverified` or the exact blocker and continue public data sources.
12. If private data sources may be collectable, read the Local Collector config before announcing scan depth. For the first lead/competitor pass for a client/source set, use 10 scrolls per approved source when safe. For normal daily scheduled runs, use the safe default: 5 scrolls per approved source, max 10, about 5 seconds between scrolls.
13. If no private data sources are active after Collector Runtime Verification, do not block the scheduled run. Continue public data source research, but include the exact verification outcome in the HTML report/notification: `private data sources not configured`, `private data sources pending activation`, `collector_status_unverified`, `collector_offline`, `wrong_workspace_bridge`, `extension_stale`, or `Private Data Source Discovery Recommended/Declined/Postponed`.
14. If published content exists, retrieve yesterday's and last-7-day published URLs, inspect each URL when authorized, record metrics/comment signals/learnings, and mark unavailable metrics honestly.
15. If no published URLs/metrics exist yet, mark measurement as `no published URLs yet`; do not pretend the measurement-learning loop ran.
16. Every human-facing reply, notification, or report handoff must include `Solo Agency daily run progress` with completed/current/remaining steps and blockers. If sending multiple updates, show updated progress each time. Also include an `Automation freshness check` that states whether the latest changes are synced into the automation/scheduled task prompt/contract/playbook/source state, not only config, and whether tomorrow's run will load the newest state.
17. Human-facing reports must be HTML only. Markdown is internal.
18. Load Stage 6 and run the Report Delivery Capability Check for every HTML report. Use tool discovery/lazy-load before declaring WideCast upload or Telegram notification unavailable.
19. Prepare a report-delivery record for every HTML report: local `.html` path, WideCast capability checked true/false, tool discovery method, upload tool available true/false/unknown, notification tool available true/false/unknown, upload attempted true/false, uploaded URL if available, upload blocker if any, notification channel, notification attempted true/false, and final notification report link.
20. If WideCast notification/Telegram is connected and WideCast report/file upload is available, upload each HTML report to WideCast first, then send the uploaded WideCast report URL through WideCast Telegram/email fallback. Do not send only a local file path when an uploaded URL is available.
21. If the current AI connector/tool surface does not expose WideCast HTML upload or Telegram/notification tools, log `widecast_report_upload_unavailable` and/or `widecast_notification_tool_unavailable`, say this is a current tool-surface blocker, and notify/surface the best available HTML path/link through chat or another authorized fallback.
22. Never send a report-ready notification without an HTML report URL/path. If that happens by mistake, immediately send a correction notification with the HTML report URL/path and log the correction.
23. Load playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md before claiming the scheduled run is complete.
```

## Required Runtime Loads

At the start of every scheduled run, load:

- `SOLO_AGENCY_PLAYBOOK.md`
- `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`
- `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`
- `playbooks/04_DAILY_SCHEDULE.md`
- `daily-content-pipeline/automation/automation_manifest.md` when present
- `daily-content-pipeline/schedule.md` when present

Then conditionally load:

- `playbooks/01_BASIC_PROFILE_PUBLIC_REPORT.md` only if setup repair is needed.
- `playbooks/PRIVATE_SOURCE_GATE.md`, `playbooks/02_PRIVATE_SOURCE_SETUP.md`, `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md`, and `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md` when private data sources are active, pending, blocked, missing-but-discovery-recommended, or being scanned.
- `playbooks/03_PRODUCTION_DISTRIBUTION.md` when drafts, production, publishing, provider setup, notification, or WideCast upload/Telegram delivery are needed.
- `playbooks/05_MEASURE_LEARN_IMPROVE.md` when any published content exists or yesterday/last-7-day measurement is due.
- `playbooks/06_AGENCY_REPORT_STANDARD.md` whenever creating or delivering HTML reports.
- `playbooks/10_LEAD_COMPETITOR_DETECTION.md` whenever detecting, scoring, reporting, storing, or improving lead and competitor opportunities. This is normally required for every daily run.
- `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md` before claiming completion.

## Scheduled Run Difference From First Setup

First setup asks the minimum setup questions because profile/config/history do not exist yet.

Scheduled runs should not ask those questions again. They should read saved state, run automatically, and interrupt the human only for approval gates, blockers, expired sessions, missing critical data, production/render/publish/credit decisions, or lead outreach decisions.

## Notification Requirement

If WideCast Telegram is connected:

1. Generate the local `.html` report.
2. Inspect available WideCast tools/capabilities for a report/file/asset upload API that supports HTML and a Telegram/report notification send tool. Use tool discovery/lazy-load when available.
3. Upload the `.html` report to WideCast when an HTML-capable upload API exists.
4. Capture the returned public or reviewable report URL.
5. Send that uploaded URL through WideCast Telegram with the run summary, blockers, lead/competitor counts, and next action.
6. If upload is unavailable or fails but notification is available, send the best available local/hosted `.html` report path/link through WideCast notification and clearly state the upload blocker.
7. Log the upload attempt and notification in `daily-content-pipeline/notifications/notification_log.md`.

If the current AI connector/tool surface or WideCast wrapper does not expose an HTML-capable upload API, do not pretend upload succeeded. Log `widecast_report_upload_unavailable`, send/surface the best available HTML path/link, and state the tool-surface blocker clearly.

If the current AI connector/tool surface does not expose a WideCast Telegram/report notification send tool, log `widecast_notification_tool_unavailable`, surface the best available HTML path/link in chat or another authorized fallback, and do not claim WideCast itself lacks notification capability.
