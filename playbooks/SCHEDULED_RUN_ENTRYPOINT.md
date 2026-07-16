# OutreachCRM Scheduled Run Entrypoint

Use this file as the scheduler prompt for unattended daily runs. The prompt should be short but explicit enough to force the agent to load the real playbooks instead of improvising from memory.

## Scheduler Prompt

```text
Run the scheduled OutreachCRM daily run now.

1. Load OUTREACHCRM_PLAYBOOK.md from the local workspace or the configured GitHub raw URL.
2. Follow the Daily Run order in playbooks/AUTOMATION_SCHEDULING.md.
3. Do not rely on memory from setup. Load the required child playbooks again at run time.
3F. Obey playbooks/LOAD_LEDGER_PROTOCOL.md full-load discipline: every stage/child playbook must be read to its last line and pass a LOAD LEDGER (compare playbooks/LOAD_MANIFEST.md when present; ledger each named dependency) BEFORE you act on it. A truncated / "output too large" / partial read = NOT loaded — re-read in chunks. Do not run any operational step without a Verdict: PASS loaded-in-full ledger for the stage(s) it needs.
4. Read outreach-pipeline/automation/automation_manifest.md, outreach-pipeline/automation/scheduled_run_prompt.md when present, outreach-pipeline/automation/github_issues.md when present, outreach-pipeline/automation/update_state.json when present, outreach-pipeline/provider_defaults.json when present, and outreach-pipeline/schedule.md before processing the client. If the manifest says automation_prompt_update_pending, report that blocker and still run from the latest local playbooks/profile/config instead of the stale snapshot.
4A. On any blocker, repeated failure, stale artifact, tool/config mismatch, or dead end, run Last-Resort Recovery before declaring the run blocked: load playbooks/11_UPDATE_AND_VERSION_WATCH.md, check GitHub main for a newer version, reload the latest relevant instructions, retry if the newer rule resolves it, and if still blocked create/send/draft a redacted issue tracked in outreach-pipeline/automation/github_issues.md.
4B. If this scheduled task is OutreachCRM - GitHub Update Watch or the request is an update/upgrade/sync-latest request, load playbooks/11_UPDATE_AND_VERSION_WATCH.md and run only the update-watch workflow. Do not process any client, sync any inbox, enrich, draft, or send from this task, and do not use any client-facing or notification channel.
5. This is a client-specific task: process only target_client_slug. Read outreach-pipeline/clients_index.md and verify the target client is active. Do not read or write any other client's data. Take the per-client run_lock.
6. Do not ask setup questions when the saved Client Intelligence Profile is complete.
7. Run the Daily Run in this order (playbooks/AUTOMATION_SCHEDULING.md has the full contract):
   a. Sync the inbox across all of this client's sendboxes (gmail_client.py sync). Classify each inbound message in the mandatory order: DSN/bounce FIRST (mailer-daemon/postmaster, multipart/report), then Auto-Submitted/OOO, then the +unsub alias token (deterministic), then threadId/In-Reply-To match = campaign reply, then contacts-but-no-thread = contact_message, else personal (count only, do not store the body). Suppress bounces and unsubscribes immediately. Cursor is historyId (OAuth) or IMAP UID (app_password) plus last_successful_sync_ts; the OAuth fallback for an expired cursor is q="after:{last_sync_epoch}" with overlap + dedupe, never newer_than:2d.
   b. Pull tracking events from the tracker worker (bot-filtered). Record open/click activities. Opens/clicks NEVER trigger a stage change or any automated action; only a reply is conversion evidence.
   c. Semantically triage replies (positive/question/objection/negative/remove_intent), then run crm_store.py apply-rules (deterministic): reply_positive -> create deal + "reply within 4h" task + freeze sequence; negative/remove_intent -> suppress + freeze; SLA sweep -> nudge tasks. Every stage change carries an evidence activity.
   d. Advise follow-ups (Stage 10 + email-writing skill), deal-aware: replies -> reply drafts; due-silent contacts -> value-add bumps. Before each follow-up, micro-refresh the contact's best sources and invalidate stale hooks. Route drafts to outbox/pending_approval/.
   e. Load new pipeline for cold/trigger campaigns (JIT buffer 3-7 days): priority pick -> Stage 4 Tier-1 verify -> Tier-2 enrich -> step-1 draft -> outbox/pending_approval/. Every personalized detail must map to a dossier hook with an evidence_url; contacts below min_confidence go to the campaign's no_hook_fallback.
   f. Render the operator-only Approval Report (report_renderer.py, NOT scrubbed) grouping High confidence vs Review carefully, one card per lead with clickable evidence URLs and an editable draft. Nothing sends yet.
   g. Send outbox/approved/ within quota (gmail_client.py send). Each send runs the ordered in-code pre-send re-check (resolve -> suppression incl. live unsub pull -> channel status -> atomic quota reservation -> warmup cap -> two-tier domain cap -> send-window -> guessed cap -> sequence-freeze -> step-1 subject lint), uses the sticky assigned sendbox (rotation only for step-1), records sent_log with the on-the-wire rfc_message_id, and appends an email_sent activity. Never send without an approval logged in approvals/approval_log.md.
   h. Draft assisted-channel messages for no-email contacts when the campaign allows and consent exists; surface them in the Today View for the human to send manually (the agent never sends them).
   i. Compile the Today View and regenerate the kanban (report_renderer.py, operator-only).
   j. Produce the daily ops report + refresh the operator-only Approval Report + INTERNAL_REPORT (operator-only). On the weekly schedule, also produce the weekly client report through the Client-Blind Scrub Gate (report_renderer.py package --client-facing --fail-on-scrub).
   k. Notify the operator via WideCast sendTelegramMessage (or chat if no provider) with counts (sent, replies, new deals, tasks due, drafts awaiting approval) and the report link; log to outreach-pipeline/notifications/notification_log.md.
8. Every human-facing reply, notification, or report handoff must include an OutreachCRM daily run progress block (completed/current/remaining steps + blockers) and, after automation exists, an Automation freshness check line. If the human must approve drafts, act on a blocker, reconnect a sendbox, or run a command, put that in a standalone **[ACTION REQUIRED]** block. If no human action is needed, say No action required right now.
9. The weekly client report is the only client-facing file and must pass the Client-Blind Scrub Gate: no OutreachCRM, WideCast, PDNA, OpenAPI, MCP, API key, Telegram, automation/scheduled task, sendbox, crm_store, sent_log, suppression, warmup, quota, guessed, tracker domain, config, or debug details. Operator reports (Approval Report, Today View, daily ops, INTERNAL_REPORT) are NOT scrubbed.
10. If notification provider config is missing, auth fails, discovery fails, or the operation is missing, do not pretend it succeeded. Log the exact blocker and surface the local report path/link in chat.
11. Load playbooks/09_OPERATIONS_SAFETY_AUDIT.md before claiming the scheduled run is complete. After the Stage 9 audit and completion gates pass, release/close the per-client run_lock (outputs/YYYY-MM/YYYY-MM-DD/{client}-run_lock.json) before ending the run.
```

## Client-Specific Automation Prompt

Prefer one automation task per client. The task name must begin with the client name:

```text
{Client Name} - OutreachCRM Daily Run
```

Client-specific automation prompts must pin `target_client_slug` and follow this contract:

```text
Run OutreachCRM daily run for target_client_slug="{client_slug}" only.

Load OUTREACHCRM_PLAYBOOK.md and the required stage playbooks.
Then Load playbooks/SCHEDULED_RUN_ENTRYPOINT.md (with a LOAD LEDGER) and follow every numbered rule of its Scheduler Prompt, plus the Daily Run order in playbooks/AUTOMATION_SCHEDULING.md, restricted to this one client. That includes: LOAD LEDGER full-load discipline per playbooks/LOAD_LEDGER_PROTOCOL.md; inbox sync with the mandatory classifier order; deterministic crm_store.py apply-rules; the evidence-URL personalization rule; the operator-only Approval Report; the in-code pre-send re-check chain; sticky-sender sendbox rotation; suppression at every send path; opens/clicks never triggering an action; the OutreachCRM daily run progress block with an Automation freshness check line; the **[ACTION REQUIRED]** / No action required right now. contract; and loading Stage 9 before claiming completion.
Read outreach-pipeline/clients_index.md and verify the target client is active. Do not read or write any other client's data. Take the per-client run_lock. After the Stage 9 audit and completion gates pass, release/close the per-client run_lock (outputs/YYYY-MM/YYYY-MM-DD/{client}-run_lock.json) before ending the run.
Read only this client's Client Intelligence Profile, CRM data (contacts/deals/activities/tasks/pipelines), sendboxes, suppression, campaigns, outbox, sent_log, inbox_sync, analytics, and provider config.
If any blocker, repeated failure, stale artifact, or instruction/tool mismatch occurs, treat stale OutreachCRM playbooks/code as the first suspect: check GitHub main, reload the latest relevant playbook instructions, and only then declare the run blocked. If still blocked, create/send/draft a redacted issue without requiring the human to have a GitHub account, tracked in outreach-pipeline/automation/github_issues.md.
For notification, prefer this client's configured provider. WideCast defaults to OpenAPI discovery at https://widecast.ai/openapi.yaml, server https://widecast.ai/app/dashboard, operations sendTelegramMessage and (optional) uploadAsset. Check this client's provider config first; do not treat a global MCP account as this client's connection. If a global MCP/native provider account is visible but cannot be proven to match this client, log global_mcp_not_client_scoped and do not use it.
Nothing sends without an approval logged in approvals/approval_log.md. Assisted-channel messages are drafted for the human to send manually; the agent never sends them.
```

If the prompt contains a `target_client_slug`, the scheduled agent must not loop through other clients.

## GitHub Update Watch Task Prompt

Use this prompt for the maintenance task named:

```text
OutreachCRM - GitHub Update Watch
```

The task checks whether OutreachCRM changed upstream and keeps installed playbooks/tools/tracker/adapter aligned with GitHub. It must not run any client workflow, sync any inbox, send anything, or use any notification/client-facing channel, and must not touch anything under `clients/`.

```text
Run OutreachCRM GitHub update watch now.

Load OUTREACHCRM_PLAYBOOK.md, playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md, playbooks/09_OPERATIONS_SAFETY_AUDIT.md, and playbooks/11_UPDATE_AND_VERSION_WATCH.md.
Read outreach-pipeline/automation/update_state.json, automation_manifest.md, scheduled_run_prompt.md, github_issues.md, and outreach-pipeline/schedule.md when present.
Check https://github.com/OWNER/outreachcrm main using the Stage 11 Fresh GitHub Checkout Protocol or a safe remote commit check.
Compare the installed/local commit with GitHub main.
If there is no new commit, update update_state.json and update_log.md with the check result, then stop.
If there is a new commit, compare root instructions, playbooks, tools (crm_store.py/gmail_client.py/import_leads.py/email_verify.py and the storage adapter), tracker/worker.js, storage schema/schema_version, setup scripts, templates, and automation contracts.
Classify the change as no_change, playbook_only, tool_change, storage_adapter_or_schema, tracker_worker, setup_or_schedule_contract, breaking_or_major_behavior, or unknown.
If auto_apply_approved is true, apply the update from a verified fresh checkout, preserve secrets/client data/suppression/history/tokens, resync automation/scheduled task prompts, and update update_state.json, update_log.md, automation_manifest.md, scheduled_run_prompt.md, and resync_log.md. If tracker_worker_deploy_required, include the exact wrangler deploy command. If storage_schema_migration_required, include the exact crm_store.py migrate/migrate-schema step.
If auto_apply_approved is false, do not apply. Write outreach-pipeline/automation/update_notice.md with the classification and the apply question, and surface it in the native task output or maintenance chat. Do not send any Telegram/provider/client notification for update-watch.
Do not process any client, sync any inbox, enrich, draft, send, or use any notification/client-facing channel in this task.
```

## Required Runtime Loads

At the start of every scheduled run, load (each with a LOAD LEDGER per `playbooks/LOAD_LEDGER_PROTOCOL.md`, checked against `playbooks/LOAD_MANIFEST.md`):

- `OUTREACHCRM_PLAYBOOK.md`
- `playbooks/LOAD_LEDGER_PROTOCOL.md`
- `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`
- `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`
- `playbooks/AUTOMATION_SCHEDULING.md`
- `outreach-pipeline/automation/automation_manifest.md` when present
- `outreach-pipeline/provider_defaults.json` when present
- `outreach-pipeline/schedule.md` when present
- `outreach-pipeline/automation/github_issues.md` when present
- `outreach-pipeline/automation/update_state.json` when present
- the target client's `integrations/providers/provider_config.local.json` when notification is needed

Then conditionally load:

- `playbooks/01_CLIENT_SETUP_PROFILE.md` if setup repair is needed or this is the client's first Automation Flow run.
- `playbooks/10_FOLLOWUP_REPLY_MANAGEMENT.md` for inbox sync and follow-up advising (normally every run).
- `playbooks/04_VERIFY_ENRICH.md` + skill `email-verify-enrich` before enrichment.
- `playbooks/06_EMAIL_WRITING_STANDARD.md` + skill `email-writing` before drafting.
- `playbooks/08_SEND_ENGINE_PROTOCOL.md` before any send.
- `playbooks/05_CAMPAIGN_MANAGEMENT.md` when a campaign is created/edited.
- `playbooks/13_CRM_CORE.md` and `playbooks/14_TASKS_TODAY_VIEW.md` for CRM object/rules/task work.
- `playbooks/12_TRACKING_ANALYTICS.md` when reading metrics / running the learning loop.
- `playbooks/15_CRM_REPORTING.md` + `playbooks/skills/report-design/SKILL.md` before rendering the weekly client report or kanban.
- `playbooks/11_UPDATE_AND_VERSION_WATCH.md` for the update-watch task or Last-Resort Recovery.
- `playbooks/09_OPERATIONS_SAFETY_AUDIT.md` before claiming completion.

(If a planned stage file does not exist yet, load `docs/DESIGN.md` for its contract.)

## Scheduled Run Difference From First Setup

First setup asks the minimum setup questions because profile/config/history do not exist yet. Scheduled runs should not ask those questions again; they read saved state, run automatically, and interrupt the human only for approval gates (the Approval Report), blockers, expired sendbox sessions, or missing critical data. Any interruption uses the `**[ACTION REQUIRED]**` block.

## Notification Requirement

If the target client's notification provider is configured:

1. Produce the operator reports (Approval Report, Today View, daily ops, INTERNAL_REPORT) and, on schedule, the scrubbed weekly client report via `tools/report_renderer.py`.
2. Load the client's provider config and `outreach-pipeline/provider_defaults.json`; fetch/cache the OpenAPI spec if needed; verify the account (WideCast `getAccount`).
3. Check Client tools for `sendTelegramMessage` (and optional `uploadAsset` for the report link). Upload the report link when supported; provider-hosted URLs may be short-lived operator handoff links, not the archive.
4. Send the notification with the run summary, counts, drafts-awaiting-approval count, and report link.
5. Log the attempt to `outreach-pipeline/notifications/notification_log.md`.
6. If provider config is missing/auth fails/operation missing, do not pretend it succeeded; log the exact blocker and surface the local report path/link in chat.

Notification is operator-facing status only, never outbound marketing to a contact.
