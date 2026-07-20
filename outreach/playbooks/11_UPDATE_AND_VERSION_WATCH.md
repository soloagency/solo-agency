# OutreachCRM Update And Version Watch

Stage: `11`

## Load Rule

Load whenever the human asks for `update`, `upgrade`, `cap nhat`, `cập nhật`, `sync latest`, `pull latest`, `refresh OutreachCRM`, `update OutreachCRM`, or any equivalent request to bring OutreachCRM to the newest version.

Also load during blocker recovery, setup repair, storage-adapter or schema-version mismatch repair, tracker-worker mismatch repair, sendbox token/auth-compat repair, provider/tooling mismatch repair, and the scheduled GitHub update-watch task.

## Hard Gates For This Stage

- GitHub `main` for `https://github.com/soloagency/solo-agency` (OutreachCRM is its `outreach/` module) is the source of truth.
- Do not use fixed shared fallback folders such as `/tmp/outreachcrm`, `/var/tmp/outreachcrm`, `/dev/shm/outreachcrm`, or any old checkout left by another session.
- Use a fresh unique `mktemp -d` checkout unless the current setup root is already proven to be a verified clone of the Solo Agency GitHub repo (module subpath `outreach/`).
- Verify `.git`, `origin`, local `HEAD`, and remote `refs/heads/main` on the parent checkout (the `outreach/` module has no `.git` of its own) before reading or copying from the source checkout.
- Treat a failed delete/update, wrong owner, missing `.git`, mismatched remote, or old timestamp as stale cache.
- Never overwrite human/client data or secrets while applying an update: agency and client `secrets/`, sendbox `credentials.json` and `token.json`, provider API keys (`api_key_local`), `TRACKER_API_KEY` / `.dev.vars`, CRM records under `clients/{client_slug}/crm/`, `sent_log.jsonl`, `activities.jsonl`, `approvals/`, global and client `suppression`, `analytics/`, `inbox_sync/`, imported `lists/*/leads.jsonl`, reports, and outputs.
- Back up changed runtime files before replacing them.
- An update is incomplete until playbooks, contracts, templates, tools (`tool crm-store`, `tool gmail`, `tool import-leads`, `tool verify-email`, `tool render-report`, `tool provider`), the storage adapter, `tracker/worker.js`, provider defaults, client setup state, and automation/scheduled task prompts have all been checked and resynced.
- If `tracker/worker.js` (or its D1 schema), the storage adapter `schema_version`, or sendbox token/auth compatibility changed, tell the human the exact command to run outside the AI sandbox (`wrangler deploy`, `tool crm-store migrate`) and exactly which sendboxes must be re-authenticated.
- The scheduled `OutreachCRM - GitHub Update Watch` task must never write under `daily-content-pipeline/clients/` and must never use a client-facing channel. It updates only the agency-tier toolkit and `daily-content-pipeline/automation/` state, and records per-client resync as pending for a maintenance session or each client's own daily automation task.

## Update Trigger Command

When the human sends only a short command such as:

```text
update
```

the agent must interpret it as:

```text
Check GitHub main for the latest OutreachCRM version, compare it with this setup, apply the update safely, resync every client and automation task, and tell me any tracker-worker deploy, storage-schema migration, or sendbox re-auth actions I must perform.
```

This is not a request to update a client report. In Setup Flow, it stays control-plane work. Do not sync inboxes, pull tracking, verify/enrich leads, draft emails, send a campaign, run rules, or generate a client report because of an update command.

## Fresh GitHub Checkout Protocol

Use this sequence before reading, diffing, or copying source files:

1. Create a fresh unique temporary folder with `mktemp -d`.
2. Clone `https://github.com/soloagency/solo-agency` (operate on its `outreach/` subpath) into that folder.
3. Verify the source checkout:
   - `.git` exists.
   - `git remote get-url origin` resolves to the Solo Agency GitHub repo (`soloagency/solo-agency`).
   - `git rev-parse HEAD` succeeds.
   - `git ls-remote origin refs/heads/main` succeeds.
   - local `HEAD` matches GitHub `main`.
4. Record `source_checkout_path`, `source_commit`, `remote_main_commit`, and `verified_at` in the update log.
5. If network or sandbox access blocks GitHub, request permission or give the human one exact clone/download command. Do not fall back to unverified local code.

**Phase-0 escape (DESIGN §22 R4).** If `OUTREACHCRM_GIT_REMOTE_URL` is unset or still contains the OWNER placeholder (repo not yet published), record `fresh_source_check: skipped_local_phase0` in `update_state.json`, treat the local working copy as authoritative, and skip the remote check rather than hard-stopping (DESIGN §22 R4). This escape applies only while the repo is unpublished; once `OUTREACHCRM_GIT_REMOTE_URL` is a real URL, the full protocol above resumes and unverified local code is no longer a valid source.

Valid sources:

- A freshly cloned and verified `mktemp -d` checkout.
- The current setup root only when it is a verified clone matching GitHub `main`.
- Raw GitHub files from `https://raw.githubusercontent.com/soloagency/solo-agency/main/outreach/` only for instruction reload when a full clone is blocked.

Invalid sources:

- `/tmp/outreachcrm`, `/var/tmp/outreachcrm`, `/dev/shm/outreachcrm`, or any fixed temporary path.
- Any folder without `.git`.
- Any folder whose delete/update failed.
- Any folder whose owner or timestamp suggests it belongs to another session.
- Any source where `HEAD` cannot be matched to GitHub `main`.

## Required Diff Scope

An update check must compare at least these areas:

- Root instructions and router:
  - `AGENTS.md`
  - `OUTREACHCRM_PLAYBOOK.md`
  - `README.md`, `.gitignore`, and the root `deploy-soloagency.sh` (`generate_outreach_artifacts`, `--outreach-only`) when present
  - `playbooks/SETUP_FLOW_ENTRYPOINT.md` and `playbooks/SCHEDULED_RUN_ENTRYPOINT.md`
- Playbooks and load discipline:
  - `playbooks/*.md` (Stages 00 through 15)
  - `playbooks/LOAD_LEDGER_PROTOCOL.md`
  - the regenerated `playbooks/LOAD_MANIFEST.md`
  - skills owned by OutreachCRM: `playbooks/skills/report-design/SKILL.md`, `email-verify-enrich`, `email-writing`
- Provider and notification tooling:
  - `tool provider`
  - `provider_defaults.template.json` and `provider_config.local.template.json`
  - the client's `integrations/providers/provider_capabilities.json`, `provider_openapi_cache.yaml`, and `provider_health.md` schema expectations
  - `daily-content-pipeline/provider_defaults.json` (WideCast notification catalog, no secrets)
- CRM and email engine tools:
  - `tool crm-store` (the only sanctioned CRM write path)
  - `tool gmail`
  - `tool import-leads`
  - `tool verify-email`
  - `tool render-report` and its `CLIENT_BLIND_TERMS` list
- Storage adapter and schema:
  - the storage layer inside `solo-agency-collector/bridge-go/` (`store.go`, `flock_*.go`)
  - per-collection `schema_version` definitions and the read-time upgrade registry (`{from_version: fn}`)
  - `daily-content-pipeline/storage_config.json` (`{"backend":"json"}` or `postgres`)
  - the parametrized adapter contract tests as the compatibility signal
- Tracker worker:
  - `tracker/worker.js`
  - `tracker/wrangler.toml` and D1 schema/migrations
  - token/HMAC derivation and the `/o`, `/c`, `/u`, `/events` route contracts
  - `TRACKER_API_KEY` / `.dev.vars` binding expectations (never their values)
- Sendbox and auth compatibility:
  - `sendboxes/sendboxes.json` schema fields (`warmup_stage`, `historyId`, `imap_uid_cursor`, `last_successful_sync_ts`, `status`)
  - `token.json` / `credentials.json` format compatibility and the OAuth scope set (`gmail.send` + `gmail.readonly` only)
- Installed runtime copies and automation:
  - `daily-content-pipeline/provider_defaults.json`
  - `daily-content-pipeline/storage_config.json`
  - each client's `integrations/providers/` capability/cache/health schema files
  - `daily-content-pipeline/schedule.md`
  - `daily-content-pipeline/automation/automation_manifest.md` and `scheduled_run_prompt.md`
  - `daily-content-pipeline/automation/update_state.json` and `update_log.md`

Do not decide "no update needed" after checking only one file or only the root playbook.

## Backup And Safe Apply Protocol

**Dirty-tree guard — run BEFORE any `git reset --hard` / forced checkout, no exceptions.** Check
`git status --porcelain` first. If any TRACKED file is modified, do NOT reset yet: copy those files
into the backup folder below AND `git stash push -m "pre-update {YYYY-MM-DD}"`, record both paths in
the update log, and only then apply the reset. A hard reset on a dirty tree silently destroys
uncommitted local work with no recovery. Untracked files survive a hard reset but must be listed in
the update log and reconciled, never ignored.

Before applying changes, create a timestamped backup of every runtime file or folder that will be replaced.

Recommended backup location:

```text
daily-content-pipeline/automation/backups/update_YYYY-MM-DD_HHMMSS/
```

The backup manifest must include:

- source path
- backup path
- previous local hash or commit when available
- new source commit
- reason for replacement
- whether the file may contain secrets

Never copy these into public support bundles or GitHub issues:

- `provider_config.local.json` when it contains `api_key_local` or other auth values
- sendbox `credentials.json`, `token.json`, OAuth tokens, app passwords, OTPs
- `TRACKER_API_KEY`, `.dev.vars`, or any agency/client `secrets/` content
- client-confidential CRM records, contacts, deals, activities, suppression, reports, approvals, or outputs

Safe apply follows three tiers. Never blur them.

- Replace from the verified source: playbooks, `OUTREACHCRM_PLAYBOOK.md`, `AGENTS.md`, both entrypoints, `LOAD_LEDGER_PROTOCOL.md`, `tracker/worker.js` and its wrangler config, skills, `README.md`, `.gitignore`, and the `*.template.json` provider files. The root `deploy-soloagency.sh` `generate_outreach_artifacts` step (mode `--outreach-only`) regenerates `outreach/playbooks/LOAD_MANIFEST.md`, rezips the skills, and runs the bridge-go test suite as a preflight.
- Merge config schemas without deleting local values: `daily-content-pipeline/provider_defaults.json`, `daily-content-pipeline/storage_config.json`, each client's `integrations/providers/provider_capabilities.json` / `provider_openapi_cache.yaml` / `provider_health.md`, `campaign_config.json`, `pipelines.json`, `sendboxes.json` (schema fields only), and the automation manifest/schedule.
- Never overwrite secrets, history, or client data: `secrets/`, sendbox `credentials.json` / `token.json`, `provider_config.local.json` auth values, `TRACKER_API_KEY` / `.dev.vars`, everything under `clients/{client_slug}/crm/`, `sent/YYYY-MM/sent_log.jsonl`, `activities/YYYY-MM/activities.jsonl`, global and client `suppression`, `approvals/`, `analytics/`, `inbox_sync/`, `lists/*/leads.jsonl`, `reports/`, and `outputs/`.

Additional safe-apply rules:

- Preserve `api_key_env`, `api_key_local`, provider identities, local account notes, and `provider_calls.jsonl` logs.
- Preserve client profiles, campaign goals, pipeline/rule customizations, sendbox warmup progress, cursors (`historyId`, `imap_uid_cursor`), suppression, history, reports, outputs, and analytics.
- All CRM record changes go through `tool crm-store`. Never hand-edit or blind-overwrite a record under `clients/{client_slug}/crm/` during an update; that is a critical "no one-off scripts" violation.
- If a file has both new template content and user-local values, merge by schema instead of blind overwrite.
- If a merge is ambiguous, write an `update_conflict` record and ask the human before overwriting.

## Client And Automation Resync After Update

Any applied update is a playbook/behavior change and therefore triggers Automation Resync when schedule/automation exists.

For every active or configured client, check and update:

- client profile schema fields (`client_profile_{client_slug}_{business_slug}_{location_slug}.md`) when the latest playbook requires them.
- `campaign_config.json` schema fields (goal block, sequence, guardrails) when Stage 5/6 changed.
- `pipelines.json` stages/rules schema when Stage 13 rule definitions changed.
- `sendboxes.json` schema fields when Stage 8/10 changed, without ever touching `credentials.json` or `token.json`.
- the client's `integrations/providers/provider_capabilities.json`, `provider_openapi_cache.yaml`, and `provider_health.md` schema expectations when provider tooling changed.
- `crm/` record `schema_version` through the adapter's read-time upgrade registry or `tool crm-store migrate`, never by hand.
- `daily-content-pipeline/provider_defaults.json`.
- `daily-content-pipeline/storage_config.json`.
- `daily-content-pipeline/schedule.md`.
- `daily-content-pipeline/automation/automation_manifest.md`.
- `daily-content-pipeline/automation/scheduled_run_prompt.md`.
- the native AI automation/scheduled task body (`{Client} - {Campaign} Daily Run`) when the environment exposes it.
- `daily-content-pipeline/automation/update_state.json` and `update_log.md`.
- `daily-content-pipeline/automation/resync_log.md`.

**Update-watch boundary on client writes.** When the update is applied interactively (a human `update` command in a setup/maintenance chat), the agent may perform the per-client resync directly, including a `tool crm-store migrate` under a storage freeze. When the update is applied by the scheduled `OutreachCRM - GitHub Update Watch` task, that task must not write under `daily-content-pipeline/clients/`: it applies only the agency-tier toolkit and `daily-content-pipeline/automation/` state, records each affected client in `clients_pending_resync`, and lets a maintenance session or each client's own daily automation task self-heal its client folder on the next run.

If the agent cannot edit the native scheduled task body directly, it must:

1. write the exact replacement prompt to `daily-content-pipeline/automation/scheduled_run_prompt.md`;
2. mark `automation_prompt_update_pending` in `automation_manifest.md` and `schedule.md`;
3. give the human one exact instruction to replace the native task prompt;
4. say `Automation Resync partially complete`, not `complete`.

## Tracker Worker Deploy And Storage Schema Migration Protocol

Classify changes that touch these paths as tracker-worker changes:

- `tracker/worker.js`
- `tracker/wrangler.toml`
- D1 schema or migration files
- the `/o`, `/c`, `/u`, or `/events` route contracts
- token derivation or `sig = HMAC(secret, token || url)` verification

When tracker-worker changes are applied:

- Prepare the updated `tracker/worker.js` and config in the current setup.
- Do not run `wrangler deploy`, `wrangler d1 migrations apply`, or `wrangler secret put` from inside the AI agent during setup/update/repair. Cloudflare deploy needs the account binding and `TRACKER_API_KEY`, and a sandbox process can be killed after the turn.
- Give the human the one-line deploy command, run from the current setup's `tracker/` directory, not the source checkout:

  ```bash
  cd "{repo_root}/outreach/tracker" && npx wrangler deploy
  ```

- If the D1 schema changed, the human must apply migrations before or with the deploy:

  ```bash
  cd "{repo_root}/outreach/tracker" && npx wrangler d1 migrations apply outreachcrm_tracker --remote
  ```

- After the human deploys, verify: `GET https://trk.{domain}/events?since=0` with the `TRACKER_API_KEY` Bearer returns 200, and a fresh `/o/{token}.gif` responds. Record `tracker_worker_deploy_required: true` until this passes.
- **Token compatibility is a compliance gate.** In-flight emails already carry `token` and `sig` values computed by the installed `tool gmail` against the current `TRACKER_API_KEY`. A worker deploy that changes token derivation or HMAC verification, or that rotates `TRACKER_API_KEY`, will break opens, clicks, and — critically — the `/u/` unsubscribe links already delivered. Never deploy a tracker change that stops honoring tokens minted by the currently installed send engine while any email carrying those tokens is still within its reply/unsubscribe window. Prefer additive, backward-compatible worker changes. A breaking token/HMAC change must ship `tool gmail` and `tracker/worker.js` together, keep old-format acceptance for at least the suppression window, and use a dual-key transition rather than a hard `TRACKER_API_KEY` rotation. If a rotation is unavoidable, block sends for the affected sendboxes until the new worker is live, mirroring the "track-pull stale beyond N hours blocks the box" rule.

Classify changes that touch these paths as storage-schema changes:

- the bridge storage layer (`solo-agency-collector/bridge-go/store.go`)
- a per-collection `schema_version` bump (for example `contacts` moving to a new version)
- the read-time upgrade registry (`{from_version: fn}`)
- `contact_identities` reverse-index structure or `storage_config.json` backend

When storage-schema changes are applied:

- Most `schema_version` bumps are handled lazily: the adapter upgrades a record on read and persists it on the next write, so `storage_schema_migration_required` stays `false` and no explicit migration is run.
- Set `storage_schema_migration_required: true` only for a structural migration the lazy path cannot do safely: a new unique index or backfill (such as rebuilding `contact_identities`), or a backend change (`json` → `postgres`).
- For a structural migration, back up the client CRM folder first (copy, never overwrite), then run the sanctioned tool under a storage freeze flag:

  ```bash
  cd "{agency_root}/outreachcrm" && <bridge> tool crm-store migrate --client {client_slug}
  ```

  A backend migration uses `<bridge> tool crm-store migrate --to postgres`. `tool crm-store migrate` upgrades all records to the current `schema_version` first and verifies with per-record content hashes, not counts.
- The scheduled update-watch task never runs a client migration. It records `storage_schema_migration_required: true` plus the affected clients in `clients_pending_resync` and hands the migration off to a maintenance session or the client's own automation preflight.
- Never hand-edit records to satisfy a schema change. Direct file writes under `clients/{client_slug}/crm/` are a critical violation.

Classify changes that touch these paths as sendbox token/auth-compat changes:

- `token.json` or `credentials.json` format
- the OAuth scope set or `auth_mode` handling in `tool gmail`
- SMTP/IMAP behavior for `app_password` sendboxes

When sendbox token/auth-compat changes are applied:

- Never overwrite `sendboxes/{sendbox_slug}/credentials.json` or `token.json` (gitignored, `chmod 600`).
- If the token format changed and `tool gmail` can migrate it in place, let it; otherwise mark the sendbox `needs_reauth` and raise an `[ACTION REQUIRED]` re-auth for that box.
- List every affected sendbox slug in `sendbox_reauth_required` and keep it there until the human confirms re-auth and a clean sync.

## Daily GitHub Update Watch Task

After setup/routine exists, the agent should recommend a lightweight update-watch automation because OutreachCRM changes frequently.

Canonical task name:

```text
OutreachCRM - GitHub Update Watch
```

Recommended cadence:

```text
Daily, before client daily runs when possible.
```

The agent should explain in plain language:

```text
OutreachCRM is updated often. A small daily update-watch task can check GitHub, write an internal update notice when a new version changes behavior, and, if you approve auto-apply, resync playbooks, tools, the storage adapter, the tracker worker, provider contracts, and scheduled tasks before the daily client runs.
```

The task must load this Stage 11 playbook and run only update/version-check work. It must not sync inboxes, pull tracking, verify/enrich, draft or send email, run rules, or generate client reports. It must never write under `daily-content-pipeline/clients/`.

Update-watch notification boundary:

- Do not send Telegram for update checks or update completion.
- Do not use WideCast `sendNotification`/`sendNotification`, the WideCast email fallback, provider notification channels, or any client notification channel for update checks.
- Update/version-watch is internal user/agency maintenance, not client delivery.
- Record update outcomes in `daily-content-pipeline/automation/update_state.json`, `update_log.md`, and `update_notice.md`.
- Surface the update result in the current setup/maintenance chat or native task output when available.
- Only client daily/weekly report runs may use the configured report notification channel.

Update-watch task algorithm:

1. Load `OUTREACHCRM_PLAYBOOK.md`, Stage 7, Stage 9, and Stage 11.
2. Read `daily-content-pipeline/automation/update_state.json` when present.
3. Check GitHub `main` using the Fresh GitHub Checkout Protocol or a remote commit check.
4. Compare `installed_commit` and `latest_checked_commit`.
5. If no change, update `last_checked_at` and stop.
6. If a new commit exists, perform the Required Diff Scope.
7. Classify the change.
8. Write a local/internal update notice. Do not use Telegram, the WideCast/email fallback, provider notification, or any client notification channel.
9. If `auto_apply_approved: true`, apply the agency-tier toolkit update and resync `daily-content-pipeline/automation/` state, record each affected client in `clients_pending_resync` without writing under `clients/`, and still require human tracker-deploy, storage-migration, or sendbox re-auth actions when those files changed.
10. If `auto_apply_approved: false`, ask the human whether to apply the update.

Change classification values (canonical enum — this Stage 11 list is the authority; Stage 07 aligns to exactly this set, do not diverge):

- `no_change`
- `playbook_only`
- `provider_tooling`
- `crm_core_tooling`
- `storage_schema_migration`
- `tracker_worker`
- `send_or_sendbox_compat`
- `renderer_or_report_format`
- `setup_or_schedule_contract`
- `breaking_or_major_behavior`
- `unknown`

Notification content must include:

- installed commit
- latest GitHub commit
- change classification
- files/categories changed
- whether auto-apply is enabled
- whether a tracker worker deploy is required
- whether a storage schema migration is required
- whether sendbox re-auth is required
- whether an automation/scheduled task prompt update is required
- exact next human action

Write this content to `daily-content-pipeline/automation/update_notice.md` and include it in the setup/maintenance chat or native task output when that surface is available.

## Update Logs And State

Maintain:

```text
daily-content-pipeline/automation/update_log.md
daily-content-pipeline/automation/update_state.json
daily-content-pipeline/automation/update_notice.md
daily-content-pipeline/automation/backups/
```

Minimum `update_state.json` (canonical schema — this Stage 11 shape is the authority, including `sendbox_reauth_required` and `clients_pending_resync`; Stage 07 aligns to this set and must not drop these fields):

```json
{
  "schema_version": 1,
  "installed_commit": "",
  "latest_checked_commit": "",
  "last_checked_at": "",
  "last_applied_commit": "",
  "last_applied_at": "",
  "auto_apply_approved": false,
  "update_watch_task_name": "OutreachCRM - GitHub Update Watch",
  "last_change_classification": "",
  "tracker_worker_deploy_required": false,
  "storage_schema_migration_required": false,
  "sendbox_reauth_required": [],
  "automation_prompt_update_pending": false,
  "update_watch_task_prompt_pending": false,
  "clients_resynced": [],
  "clients_pending_resync": [],
  "automations_resynced": [],
  "human_actions_required": []
}
```

Set `update_watch_task_prompt_pending` to `true` when the `OutreachCRM - GitHub Update Watch` task prompt could not be created or updated natively and `daily-content-pipeline/automation/update_watch_prompt.md` holds the pending prompt.

Minimum `update_log.md` table:

```md
# OutreachCRM Update Log

| Date | Agent | Local Commit Before | GitHub Main Commit | Change Classification | Applied | Backup Path | Clients Resynced | Automations Resynced | Tracker Worker Deploy Required | Storage Schema Migration Required | Blocker / Next Action |
|---|---|---|---|---|---|---|---|---|---|---|---|
```

If an update produces conflicts, create:

```text
daily-content-pipeline/automation/issues/YYYY-MM-DD_update_conflict.md
```

or an equivalent local `update_conflict` record, then track it in `update_log.md`.

## Completion Gate

Do not claim an update is complete until all true items are satisfied:

- Stage 11 was loaded.
- Latest GitHub `main` was checked through a verified source path or remote commit check.
- Local/installed commit and GitHub commit were recorded.
- Required Diff Scope was covered.
- Change classification was recorded.
- Backups were created for every replaced runtime file/folder.
- Toolkit, playbooks, templates, tools, storage adapter, and tracker worker were updated from the verified source when needed, and `playbooks/LOAD_MANIFEST.md` was regenerated.
- Config schemas were merged safely without losing local values or secrets.
- Storage `schema_version` changes were handled through the adapter upgrade registry or `tool crm-store migrate` (never hand-edited), with per-record content-hash verification when a structural migration ran.
- Every configured client and scheduled/automation task was resynced, or, for the update-watch task, recorded in `clients_pending_resync` with a precise handoff and a logged `automation_prompt_update_pending` where the native prompt could not be edited.
- `update_state.json` and `update_log.md` were updated.
- If `tracker/worker.js` or its D1 schema changed, the human received the exact `wrangler deploy` (and `wrangler d1 migrations apply` when needed) command to run against the current setup's `tracker/`, and the token/HMAC backward-compatibility window was preserved.
- If sendbox token/auth compatibility changed, affected sendboxes were flagged for re-auth via `[ACTION REQUIRED]` and no `credentials.json`/`token.json` was overwritten.
- The final human-facing message says whether update state is `complete`, `partial`, or `blocked`.

## Safety Audit Notes

- Treat update work as operator/internal work. Do not put update details into the client-facing weekly report.
- Do not expose provider names, API-key status, tracker/worker internals, sendbox emails or tokens, GitHub commits, storage backend, or update internals to the client's contacts/customers.
- If an update is triggered during a scheduled daily run, finish the update/resync only when safe. If it would delay a time-sensitive send or report, record the pending update in `update_state.json`, `update_log.md`, and `update_notice.md`; do not send WideCast Telegram/provider notification for the update; and do not mix partially updated playbooks or a partially migrated store with an in-progress client run.
- Never apply a storage schema migration or deploy a breaking tracker change mid-run while sends are in flight. A migration would freeze the store under the target client, and a breaking token/HMAC change would invalidate the tokens on already-sent mail. Defer both to a clean maintenance window.
- If the latest GitHub version still does not resolve the blocker that triggered the update, follow the Last-Resort Recovery And GitHub Issue Escalation Rule.
