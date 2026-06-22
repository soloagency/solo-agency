# Solo Agency Scheduled Run Entrypoint

Use this file as the scheduler prompt for unattended daily runs.

The scheduled prompt should be short, but it must be explicit enough to force the agent to load the real playbooks instead of improvising from memory.

## Scheduler Prompt

```text
Run the scheduled Solo Agency daily run now.

1. Load SOLO_AGENCY_PLAYBOOK.md from the local workspace or from the configured GitHub raw URL.
2. Follow the Scheduled Run Playbook Loading Contract in playbooks/04_DAILY_SCHEDULE.md.
3. Do not rely on memory from setup. Load the required child playbooks again at run time.
4. Process every active client in daily-content-pipeline/clients_index.md.
5. Do not ask setup questions when the saved Client Intelligence Profile is complete.
6. Run public research, private scans if active, published-URL analytics only when published URLs/metrics exist, analysis, idea matrix, best idea selection, drafts, HTML report generation, learning updates, and notification.
7. If private sources are active, read the Local Collector config before announcing scan depth. If config is unavailable, use the safe default: 5 scrolls per approved source, max 10, about 5 seconds between scrolls.
8. If published content exists, retrieve yesterday's and last-7-day published URLs, inspect each URL when authorized, record metrics/comment signals/learnings, and mark unavailable metrics honestly.
9. If no published URLs/metrics exist yet, mark measurement as `no published URLs yet`; do not pretend the measurement-learning loop ran.
10. Every human-facing reply, notification, or report handoff must include `Solo Agency daily run progress` with completed/current/remaining steps and blockers. If sending multiple updates, show updated progress each time.
11. Human-facing reports must be HTML only. Markdown is internal.
12. If WideCast notification/Telegram is connected and WideCast report/file upload is available, upload each HTML report to WideCast first, then send the uploaded WideCast report URL through WideCast Telegram. Do not send only a local file path when an uploaded URL is available.
13. If report upload fails, log the blocker, notify the human with the best available HTML path/link, and say that WideCast report upload failed.
14. Load playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md before claiming the scheduled run is complete.
```

## Required Runtime Loads

At the start of every scheduled run, load:

- `SOLO_AGENCY_PLAYBOOK.md`
- `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`
- `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`
- `playbooks/04_DAILY_SCHEDULE.md`

Then conditionally load:

- `playbooks/01_BASIC_PROFILE_PUBLIC_REPORT.md` only if setup repair is needed.
- `playbooks/02_PRIVATE_SOURCE_SETUP.md` and `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md` when private sources are active, pending, blocked, or being scanned.
- `playbooks/03_PRODUCTION_DISTRIBUTION.md` when drafts, production, publishing, provider setup, notification, or WideCast upload/Telegram delivery are needed.
- `playbooks/05_MEASURE_LEARN_IMPROVE.md` when any published content exists or yesterday/last-7-day measurement is due.
- `playbooks/06_AGENCY_REPORT_STANDARD.md` whenever creating or delivering HTML reports.
- `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md` before claiming completion.

## Scheduled Run Difference From First Setup

First setup asks the minimum setup questions because profile/config/history do not exist yet.

Scheduled runs should not ask those questions again. They should read saved state, run automatically, and interrupt the human only for approval gates, blockers, expired sessions, missing critical data, production/render/publish/credit decisions, or lead outreach decisions.

## Notification Requirement

If WideCast Telegram is connected:

1. Generate the local `.html` report.
2. Upload the `.html` report to WideCast using the available report/file/asset upload API that supports HTML.
3. Capture the returned public or reviewable report URL.
4. Send that uploaded URL through WideCast Telegram with the run summary, blockers, lead/competitor counts, and next action.
5. Log the upload and notification in `daily-content-pipeline/notifications/notification_log.md`.

If the current WideCast wrapper does not expose an HTML-capable upload API, do not pretend upload succeeded. Log `widecast_report_upload_unavailable`, send the best available HTML path/link, and tell the human how to enable report upload.
