# OutreachCRM Setup Flow Entrypoint

Use this file as the entrypoint for setup/configuration sessions.

Setup Flow is the control plane. It configures OutreachCRM so the daily automation runs correctly later. **It never sends an email, runs a campaign, enriches a lead for send, or drafts-and-sends.**

## Runtime Requirement

Before setup proceeds, verify or explain that OutreachCRM needs Codex, Claude Desktop/Cowork, or a comparable desktop/local AI agent runtime with workspace file access, scheduled/automation tasks, local Python execution, and parallel/sub-agent work. Do not present a plain web chat as the primary runtime.

## Setup Flow Contract

1. Load `OUTREACHCRM_PLAYBOOK.md` and `playbooks/LOAD_LEDGER_PROTOCOL.md`. **Full-load discipline applies to every file below: each load needs a LOAD LEDGER (read to the last line; compare `playbooks/LOAD_MANIFEST.md` when present; ledger each named dependency). A truncated / "output too large" / partial read = NOT loaded — re-read in chunks before acting. No side-effect step without a PASS ledger for the stage(s) it needs.**
2. Load `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`, `playbooks/01_CLIENT_SETUP_PROFILE.md`, `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`, and `playbooks/09_OPERATIONS_SAFETY_AUDIT.md`.
3. Load `playbooks/02_SENDBOX_SETUP.md` when connecting a sendbox, `playbooks/03_IMPORT_LIST.md` when importing a list, `playbooks/05_CAMPAIGN_MANAGEMENT.md` when creating the first campaign, and `playbooks/AUTOMATION_SCHEDULING.md` when configuring the schedule or creating/resyncing the automation task. (If a planned stage file does not exist yet, load `docs/DESIGN.md` for its contract.)
4. Load `playbooks/11_UPDATE_AND_VERSION_WATCH.md` when the human asks for update/upgrade/sync-latest, when setup repair suspects stale playbooks/code, or when configuring the `OutreachCRM - GitHub Update Watch` maintenance task.
5. Create or update client setup, the Client Intelligence Profile, pipelines and custom fields, sending identity, sendbox connections, imported lists, campaigns, schedule files, automation manifests, scheduled prompts, update-watch state, and resync logs — all CRM state through `tools/crm_store.py`. Write CRM records (pipelines, segments, custom-field definitions, contacts) through `tools/crm_store.py` — it exists (Phase 1). The Client Intelligence Profile is a `.md` file written directly. A workspace updated from an older Phase-0 install must run `python3 tools/crm_store.py --client-dir <DIR> validate --rebuild-index` once per client before use (DESIGN §22 R3).
6. Do not send any email, run any campaign, enrich a lead for send, generate a live-send Approval Report, or start outreach in Setup Flow.
7. If the human asks to send, run a campaign, or draft-and-send inside Setup Flow, this is a hard stop for operational work. The setup chat stays Setup Flow. Verify or create the relevant automation task, resync its prompt/config if needed, and tell the human the exact task name to run instead.
8. If the human says only `update`, `upgrade`, `cập nhật`, `sync latest`, or `pull latest`, treat that as the Stage 11 OutreachCRM update command, not as `send now`.
9. Every client-specific automation task name must begin with the client name, e.g. `Max Output - OutreachCRM Daily Run`.
10. Notification setup (optional) is client-scoped: read/write the current client's `integrations/providers/` files and verify provider identity through the client's OpenAPI/API-key config before claiming notification is available. Default is WideCast: ask only for the client's WideCast API key. Do not treat a global MCP/native provider account as this client's connection. Notification is optional; mark it `–` if declined.
11. After any approved config change or applied OutreachCRM update, perform Automation Resync if a schedule/automation already exists.
12. Setup Flow completion means `ready_for_automation_first_run`.
13. Every human question, approval request, one-line command, sendbox-connection step, provider/API-key setup request, and native automation task edit must use the `**[ACTION REQUIRED]**` block from `OUTREACHCRM_PLAYBOOK.md`. If setup continues without needing the human, end the reply with next-action guidance per the Next-Action Guidance Rule instead of `No action required right now.`

## The 9-step Setup Sequence

Follow `playbooks/01_CLIENT_SETUP_PROFILE.md` for detail. Summary:

1. One opening question (product/service or business + ideal customer + optional URL). Agent infers ICP, value proposition, and email voice, then shows them for correction.
2. Propose a pipeline (default 6 stages) and custom fields; the human adjusts.
3. Confirm the sending identity: from-name, signature, physical mailing address (CAN-SPAM), unsubscribe method.
4. Connect the first sendbox (Stage 2). `@gmail.com` App Password is the quick path; Workspace OAuth is advanced.
5. Import the first list (Stage 3): inspect + confirm column mapping, then normalize, dedupe, and check suppression.
6. Create the first campaign and its structured goal (Stage 5).
7. Notification (optional): WideCast API key only.
8. Record a baseline (nothing sent).
9. Create the `{Client} - OutreachCRM Daily Run` automation task (pinning `target_client_slug`) and, once a schedule exists, offer `OutreachCRM - GitHub Update Watch`. Explain what Automation Flow will do; it does not run in Setup Flow.

## Fresh Source Acquisition Hard Gate

Before copying playbooks, tools, or templates into the human's setup, verify the source repo:

- Use the current setup root if it is already a verified clone of `https://github.com/soloagency/solo-agency` (the OutreachCRM module is its `outreach/` subpath); otherwise clone into a fresh unique `mktemp -d`.
- Do not use fixed shared fallback folders such as `/tmp/outreachcrm`, `/var/tmp/outreachcrm`, or `/dev/shm/outreachcrm`.
- Do not trust a folder that lacks `.git`, has the wrong owner, has an old timestamp, or could not be removed/updated.
- Verify `git remote get-url origin`, `git rev-parse HEAD`, and `git ls-remote origin refs/heads/main` on the parent checkout (the `outreach/` module has no `.git` of its own); local `HEAD` must match GitHub `main` before reading/copying.
- If GitHub access fails, request permission or give the human one exact clone/download command. Do not proceed with stale local code.
- If `OUTREACHCRM_GIT_REMOTE_URL` is unset or still contains the `OWNER` placeholder, treat this local working copy as the verified source, record `fresh_source_check: skipped_local_phase0` in `resync_log.md`, skip the GitHub clone/verify/fetch steps, and continue (DESIGN §22 R4).
- Record the verified source path and commit hash in `daily-content-pipeline/automation/resync_log.md` (or `update_state.json`/`update_log.md` for a full update).

## Setup Blocker Recovery

If setup hits any unexpected blocker, old/corrupt artifact, missing file, permission mismatch, instruction conflict, or dead end:

1. Assume the setup agent may be using an old OutreachCRM version.
2. Re-run the Fresh Source Acquisition Hard Gate against GitHub `main`. If `OUTREACHCRM_GIT_REMOTE_URL` is unset or still contains the `OWNER` placeholder, treat this local working copy as the verified source, record `fresh_source_check: skipped_local_phase0` in `resync_log.md`, skip the GitHub clone/verify/fetch steps, and continue (DESIGN §22 R4).
3. Reload `OUTREACHCRM_PLAYBOOK.md` plus the relevant child playbooks from the verified latest source.
4. If the newest playbook fixes the setup path, continue from the latest rule and resync the client setup/automation state.
5. If the newest version still leaves setup blocked, escalate without requiring the human to have a GitHub account: create a redacted issue when an authorized identity exists, send/queue via a configured intake channel, or write a ready-to-post draft.

Do not include secrets, API keys, tokens, sendbox credentials, or contact PII in issues. Write drafts under `daily-content-pipeline/automation/issues/`, track them in `daily-content-pipeline/automation/github_issues.md`, and tell the human the path.

## Required Setup Output

For each configured client, Setup Flow must leave these current (all CRM writes via `crm_store.py`). Write CRM records (pipelines, segments, custom-field definitions, contacts) through `tools/crm_store.py` — it exists (Phase 1). The Client Intelligence Profile is a `.md` file written directly. A workspace updated from an older Phase-0 install must run `python3 tools/crm_store.py --client-dir <DIR> validate --rebuild-index` once per client before use (DESIGN §22 R3).

- Client Intelligence Profile.
- `crm/pipelines.json` and custom field definitions.
- Sending identity (from-name, signature, physical address, unsubscribe method) in the profile.
- `sendboxes/sendboxes.json` with at least one connected sendbox (or a recorded pending action).
- At least one imported list (`lists/{list_slug}/`) or a recorded pending action.
- At least one `campaigns/{campaign_slug}/campaign_config.json` with a structured goal.
- `integrations/providers/provider_config.local.json` when notification was configured.
- `daily-content-pipeline/schedule.md`.
- `daily-content-pipeline/automation/automation_manifest.md`.
- `daily-content-pipeline/automation/scheduled_run_prompt.md`.
- `daily-content-pipeline/automation/update_state.json` and `update_log.md` when update/watch has been checked/configured.
- `daily-content-pipeline/automation/resync_log.md`.

If the native automation task prompt cannot be updated directly, mark `automation_prompt_update_pending` in the manifest and schedule, then give the human one concrete instruction to update the task prompt.

For sendbox connection, the setup handoff must present the connection steps in an `**[ACTION REQUIRED]**` block, not just a status line. `tools/gmail_client.py` exists (Phase 1): the handoff includes the App Password creation steps (Google Account → Security → 2-Step Verification → App passwords) plus the exact command — set `OUTREACHCRM_APP_PASSWORD` in the human's shell, then `python3 tools/gmail_client.py --client-dir <DIR> auth --sendbox <slug> --email <you@gmail.com>`, which verifies SMTP+IMAP and writes the box as `status: healthy`.

After schedule/automation exists, offer the maintenance task `OutreachCRM - GitHub Update Watch`. If the runtime cannot create it directly, write the exact prompt to `daily-content-pipeline/automation/update_watch_prompt.md`, tell the human the task name to create, and record `update_watch_task_prompt_pending` in the automation/update state.

## Send Request Hard Stop

When the human asks to send / run a campaign / draft-and-send while this entrypoint is active, the only valid response is:

1. State that Setup Flow does not send.
2. Finish or resync the client-specific automation task.
3. Provide the exact task name to run for the first daily run.
4. If the native automation UI requires human action, provide that one exact action in a `**[ACTION REQUIRED]**` block.
5. End with a `**[ACTION REQUIRED]**` block naming the exact automation task to run, AND a feature-discovery block of 2-3 unused headline capabilities (Feature Discovery Rule; also introduce the content/video side of the funnel) - setup never ends flat. When no action is needed, still end with next-action guidance plus the feature-discovery block.

Do not ask whether to send now. Do not load `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` inside the setup chat. Do not enrich, draft-to-send, or notify in Setup Flow.
