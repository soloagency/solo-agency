# Solo Agency Scheduled Run Entrypoint

Use this file as the scheduler prompt for unattended daily runs.

The scheduled prompt should be short, but it must be explicit enough to force the agent to load the real playbooks instead of improvising from memory.

## Scheduler Prompt

```text
Run the scheduled Solo Agency daily run now.

1. Load SOLO_AGENCY_PLAYBOOK.md from the local workspace or from the configured GitHub raw URL.
2. Follow the Scheduled Run Playbook Loading Contract in playbooks/04_DAILY_SCHEDULE.md.
3. Do not rely on memory from setup. Load the required child playbooks again at run time.
4. Read `daily-content-pipeline/automation/automation_manifest.md`, `daily-content-pipeline/automation/scheduled_run_prompt.md` when present, `daily-content-pipeline/provider_defaults.json` when present, and `daily-content-pipeline/schedule.md` before processing clients. If the manifest says `automation_prompt_update_pending`, report that blocker and still run from the latest local playbooks/profile/config instead of the stale snapshot.
5. Process every active client in daily-content-pipeline/clients_index.md.
6. Do not ask setup questions when the saved Client Intelligence Profile is complete.
7. Run public research first, then private scans if active, then published-URL analytics only when published URLs/metrics exist and the configured provider/URL inspection path allows it, analysis, Lead & Competitor Opportunities, idea matrix, best idea selection, production-ready drafts, approved video/blog/social asset creation when provider setup and explicit approval allow it, HTML report generation, learning updates, and notification. Public data source intelligence and private data source intelligence must be separate full HTML reports inside one canonical report set.
8. If private data sources are active, pending, requested, approved, present in a profile, or if any current-workspace collector status files exist, load `playbooks/PRIVATE_SOURCE_GATE.md`, `playbooks/02_PRIVATE_SOURCE_SETUP.md`, `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md`, and `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md`; use only the Solo Agency Local Collector extension plus Local Collector app. Do not use Claude in Chrome, Codex/browser tools, Playwright, or any agent-controlled browser for private data sources.
9. Do not trust saved config/status labels alone, including `public_data_sources_only`, `pending_private_activation`, or `private sources postponed`. Before skipping private data sources, perform Collector Runtime Verification: try the bridge status API, verify current-workspace identity, and read local collector health/output status files when the API is unreachable from the AI sandbox.
10. If `GET http://127.0.0.1:17321/status` works, verify `/status.config_file`, `/status.output_dir`, and `/status.run_now_request_file` point to the current setup's `daily-content-pipeline/collector/` tree before trusting the bridge. If they point elsewhere, mark `wrong_workspace_bridge`, skip private collection, and tell the human to run the current setup's Local Collector command and remove/disable old Solo Agency Local Collector extensions in `chrome://extensions`.
11. If `GET http://127.0.0.1:17321/status` fails, do not immediately conclude the Local Collector is inactive. Some AI scheduled tasks run in a sandbox where `127.0.0.1` is not the human machine's localhost. Read `daily-content-pipeline/collector/inbox/bridge_health.json`, `daily-content-pipeline/collector/inbox/collector_status.json`, `daily-content-pipeline/collector/collector_setup_status.md`, recent `daily-content-pipeline/collector/inbox/YYYY-MM/*/collector_status.json`, and any recent consumed run-now status files. If those files show a recent healthy current-workspace bridge/extension, use the file-based run-now queue described in Stage 8 by writing one unique job file per client under `daily-content-pipeline/collector/jobs/pending/`; if they are stale/missing, mark `collector_status_unverified` or the exact blocker and continue public data sources.
12. If private data sources may be collectable, read the Local Collector config before announcing scan depth. For the first lead/competitor pass for a client/source set, use 10 scrolls per approved source when safe. For normal daily scheduled runs, use the safe default: 5 scrolls per approved source, max 10, about 5 seconds between scrolls.
13. If no private data sources are active after Collector Runtime Verification, do not block the scheduled run. Continue public data source research, but include the exact verification outcome in the HTML report/notification: `private data sources not configured`, `private data sources pending activation`, `collector_status_unverified`, `collector_offline`, `wrong_workspace_bridge`, `extension_stale`, or `Private Data Source Discovery Recommended/Declined/Postponed`.
14. If published content exists, retrieve yesterday's and last-7-day published URLs, inspect each URL when authorized, record metrics/comment signals/learnings, and mark unavailable metrics honestly.
15. If no published URLs/metrics exist yet, mark measurement as `no published URLs yet`; do not pretend the measurement-learning loop ran.
16. Every human-facing reply, notification, or report handoff must include `Solo Agency daily run progress` with completed/current/remaining steps and blockers. If sending multiple updates, show updated progress each time. Also include an `Automation freshness check` that states whether the latest changes are synced into the automation/scheduled task prompt/contract/playbook/source state, not only config, and whether tomorrow's run will load the newest state.
17. Human-facing reports must be HTML only. Markdown is internal.
18. Load Stage 6 and obey the Three-File Public/Private Report Contract: one canonical report set per client/day/run with `{client-name}-public-data-sources-report.html`, `{client-name}-private-data-sources-report.html`, and `{client-name}-daily-report.html`.
19. If public data sources finish before private data sources, generate/update only `{client-name}-public-data-sources-report.html` and `{client-name}-daily-report.html` with private status pending/blocked. If private data sources later finish, generate/update only `{client-name}-private-data-sources-report.html` and `{client-name}-daily-report.html`; do not rewrite the public report.
20. Two notifications are acceptable: one for `public_report_ready` and one for `private_report_ready` or `private_report_blocked`. Notifications should normally point to `{client-name}-daily-report.html` or its uploaded URL, and each notification must state the lane status.
21. Run the Provider Report Delivery Capability Check for every report notification or report handoff. Read this client's provider config, fetch/cache the provider OpenAPI spec, verify account identity, and inspect operation IDs before declaring upload or notification unavailable. WideCast default operations are `uploadAsset` for HTML report upload and `sendTelegramMessage` for Telegram/email fallback notification.
22. Prepare a report-delivery record for every report notification/handoff: lane status, local `.html` path, provider, provider discovery checked true/false, account verification status, upload operation ID, notification operation ID, upload attempted true/false, uploaded URL if available, upload blocker if any, notification channel, notification attempted true/false, and final notification report link.
23. If WideCast OpenAPI notification/Telegram/email fallback is configured and `uploadAsset` supports `text/html`, upload the current `{client-name}-daily-report.html` to WideCast before each report notification, then send that uploaded WideCast report URL through `sendTelegramMessage`. Do not send only a local file path when an uploaded URL is available. Treat the uploaded URL as a short-lived handoff link when the provider says it has a TTL; the local report remains the permanent archive.
24. If provider config is missing, auth fails, OpenAPI discovery fails, account identity mismatches, or the required upload/notification operation is missing, log the provider-neutral blocker plus any useful legacy WideCast alias, and notify/surface the best available HTML path/link through chat or another authorized fallback.
25. Never send a report-ready notification without an HTML report URL/path. If that happens by mistake, immediately send a correction notification with the HTML report URL/path and log the correction.
26. Load playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md before claiming the scheduled run is complete.
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
Read only this client's provider config under integrations/providers/. If PDNA actions, analytics, report upload, notification, video creation, or publishing are needed, fetch/cache the configured provider OpenAPI spec, verify the provider account belongs to this client, and stop provider actions on account mismatch.
For private data sources, use only the extension_instance_id configured for this client in daily-content-pipeline/collector/extension_registry.json.
If localhost is unavailable from the automation sandbox, read local bridge/extension health files and write a per-client job file under daily-content-pipeline/collector/jobs/pending/.
With one shared bridge, private data source collection is parallel per client identity, bound to the claiming extension instance. Different client Chrome profiles can collect at the same scheduled time, while jobs for the same client/profile stay queued and sequential until `/complete` or TTL.
The collector output path must be daily-content-pipeline/collector/inbox/YYYY-MM/{client_slug}/{run_id}/.
If the matching client extension is missing, stale, or wrong, continue with public data sources only and report the exact blocker.
Generate this client's canonical three-file HTML report set, update this client's history/learning, and resync any configuration changes made during the run.
For report delivery, prefer this client's configured provider. WideCast defaults to OpenAPI discovery at https://widecast.ai/openapi.yaml and operations uploadAsset + sendTelegramMessage.
If private data sources complete after public data sources, create/update only {client-name}-private-data-sources-report.html and {client-name}-daily-report.html; preserve {client-name}-public-data-sources-report.html exactly except for explicit link repair.
```

If the prompt contains a `target_client_slug`, the scheduled agent must not loop through every active client. The older all-clients loop applies only to an explicitly named all-clients/master task.

Optional master digest task:

```text
Solo Agency Master Digest - All Clients
```

The master digest task must not scan private data sources. It only reads existing client reports/outputs and creates a summary.

## Required Runtime Loads

At the start of every scheduled run, load:

- `SOLO_AGENCY_PLAYBOOK.md`
- `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`
- `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`
- `playbooks/04_DAILY_SCHEDULE.md`
- `daily-content-pipeline/automation/automation_manifest.md` when present
- `daily-content-pipeline/provider_defaults.json` when present
- `daily-content-pipeline/schedule.md` when present
- the target client's `integrations/providers/provider_config.local.json` when PDNA, analytics, report delivery, notification, production, or publishing is needed

Then conditionally load:

- `playbooks/01_BASIC_PROFILE_PUBLIC_REPORT.md` only if setup repair is needed.
- `playbooks/PRIVATE_SOURCE_GATE.md`, `playbooks/02_PRIVATE_SOURCE_SETUP.md`, `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md`, and `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md` when private data sources are active, pending, blocked, missing-but-discovery-recommended, or being scanned.
- `playbooks/03_PRODUCTION_DISTRIBUTION.md` when drafts, production, publishing, provider setup, notification, or provider report delivery are needed.
- `playbooks/05_MEASURE_LEARN_IMPROVE.md` when any published content exists or yesterday/last-7-day measurement is due.
- `playbooks/06_AGENCY_REPORT_STANDARD.md` whenever creating or delivering HTML reports.
- `playbooks/10_LEAD_COMPETITOR_DETECTION.md` whenever detecting, scoring, reporting, storing, or improving lead and competitor opportunities. This is normally required for every daily run.
- `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md` before claiming completion.

## Scheduled Run Difference From First Setup

First setup asks the minimum setup questions because profile/config/history do not exist yet.

Scheduled runs should not ask those questions again. They should read saved state, run automatically, and interrupt the human only for approval gates, blockers, expired sessions, missing critical data, production/render/publish/credit decisions, or lead outreach decisions.

## Notification Requirement

If the target client's provider notification is configured:

1. Generate or update the local canonical `{client-name}-daily-report.html` report index plus any affected lane report.
2. Load the target client's provider config and `daily-content-pipeline/provider_defaults.json`.
3. Fetch/cache the provider OpenAPI spec if needed.
4. Verify the provider account identity before using account actions. For WideCast, call `getAccount`.
5. Inspect discovered operations for a report/file/asset upload API that supports HTML and a notification send operation. For WideCast, use `uploadAsset` with `text/html` and `sendTelegramMessage`.
6. Upload the `.html` report to the provider when an HTML-capable upload operation exists.
7. Capture the returned public or reviewable report URL and its TTL if the provider returns one.
8. Send that uploaded URL through the provider notification channel with the run summary, lane status, blockers, lead/competitor counts, and next action.
9. If upload is unavailable or fails but notification is available, send the best available local/hosted `.html` report path/link through provider notification and clearly state the upload blocker.
10. Log the upload attempt and notification in `daily-content-pipeline/notifications/notification_log.md`, including lane status, provider, operation IDs, upload URL, TTL if known, and blocker if any.

If both public and private notifications are sent, both notifications should link to the same `{client-name}-daily-report.html` path or uploaded URL. Lane-specific public/private report links may be included as secondary links.

If provider config is missing, auth fails, OpenAPI discovery fails, account verification mismatches, or the required provider operation is missing, do not pretend upload or notification succeeded. Log the exact provider-neutral blocker, send/surface the best available HTML path/link, and state the blocker clearly.

Do not claim WideCast itself lacks upload or notification capability merely because a legacy MCP/tool surface is unavailable. Check the configured OpenAPI provider path first.
