# Solo Agency Update And Version Watch

Stage: `11`

## Load Rule

Load whenever the human asks for `update`, `upgrade`, `cap nhat`, `cập nhật`, `sync latest`, `pull latest`, `refresh Solo Agency`, `update Solo Agency`, or any equivalent request to bring Solo Agency to the newest version.

Also load during blocker recovery, setup repair, Local Collector repair, extension repair, provider/tooling mismatch repair, and the scheduled GitHub update-watch task.

## Hard Gates For This Stage

- GitHub `main` for `https://github.com/soloagency/solo-agency` is the source of truth.
- Do not use fixed shared fallback folders such as `/tmp/solo-agency`, `/var/tmp/solo-agency`, `/dev/shm/solo-agency`, or any old checkout left by another session.
- Use a fresh unique `mktemp -d` checkout unless the current setup root is already proven to be a verified clone of the Solo Agency GitHub repo.
- Verify `.git`, `origin`, local `HEAD`, and remote `refs/heads/main` before reading or copying from the source checkout.
- Treat a failed delete/update, wrong owner, missing `.git`, mismatched remote, or old timestamp as stale cache.
- Never overwrite human/client data, secrets, cookies, tokens, provider API keys, local private data source captures, reports, history, approvals, or outputs while applying an update.
- Back up changed runtime files before replacing them.
- An update is incomplete until playbooks, contracts, templates, collector code, extension folders, provider defaults, client setup state, and automation/scheduled task prompts have all been checked and resynced.
- If bridge Go/runtime files or extension templates changed, tell the human exactly which command to run outside the AI sandbox and exactly which client Chrome extensions/profiles to reload.

## Update Trigger Command

When the human sends only a short command such as:

```text
update
```

the agent must interpret it as:

```text
Check GitHub main for the latest Solo Agency version, compare it with this setup, apply the update safely, resync every client and automation task, and tell me any bridge/extension actions I must perform.
```

This is not a request to update a daily report. In Setup Flow, it stays control-plane work. Do not run reports, public research, private data source scans, video/blog/social production, analytics scans, or publishing because of an update command.

## Fresh GitHub Checkout Protocol

Use this sequence before reading, diffing, or copying source files:

1. Create a fresh unique temporary folder with `mktemp -d`.
2. Clone `https://github.com/soloagency/solo-agency` into that folder.
3. Verify the source checkout:
   - `.git` exists.
   - `git remote get-url origin` resolves to the Solo Agency GitHub repo.
   - `git rev-parse HEAD` succeeds.
   - `git ls-remote origin refs/heads/main` succeeds.
   - local `HEAD` matches GitHub `main`.
4. Record `source_checkout_path`, `source_commit`, `remote_main_commit`, and `verified_at` in the update log.
5. If network or sandbox access blocks GitHub, request permission or give the human one exact clone/download command. Do not fall back to unverified local code.

Valid sources:

- A freshly cloned and verified `mktemp -d` checkout.
- The current setup root only when it is a verified clone matching GitHub `main`.
- Raw GitHub files from `https://raw.githubusercontent.com/soloagency/solo-agency/main/` only for instruction reload when full clone is blocked.

Invalid sources:

- `/tmp/solo-agency`, `/var/tmp/solo-agency`, `/dev/shm/solo-agency`, or any fixed temporary path.
- Any folder without `.git`.
- Any folder whose delete/update failed.
- Any folder whose owner or timestamp suggests it belongs to another session.
- Any source where `HEAD` cannot be matched to GitHub `main`.

## Required Diff Scope

An update check must compare at least these areas:

- Root instructions and router:
  - `AGENTS.md`
  - `SOLO_AGENCY_PLAYBOOK.md`
  - setup/readme/deploy scripts when present
- All playbooks:
  - `playbooks/*.md`
  - setup and scheduled entrypoints
  - provider adapter playbooks
  - any vendored or generated skill packaging rules owned by Solo Agency
- Provider and Client tools support:
  - provider defaults/templates
  - OpenAPI helper code such as `tools/provider_openapi.py`
  - provider capability schema files
  - default WideCast/OpenAPI catalog files
- Local Collector:
  - bridge Go source
  - bridge binaries or downloadable artifact metadata
  - checksums
  - setup/start scripts and launchers
  - collector protocol/config schema
- Chrome extension:
  - extension template files
  - manifest template
  - background/popup/content scripts
  - `client_binding.json` schema/defaults
- Installed runtime copies:
  - `{agency_root}/solo-agency-local-collector/`
  - `{agency_root}/extensions/{client_slug}/`
  - `{agency_root}/daily-content-pipeline/collector/`
  - `{agency_root}/daily-content-pipeline/provider_defaults.json`
  - each client's `integrations/providers/` capability/cache schema files
  - automation manifests and scheduled prompts

Do not decide "no update needed" after checking only one file or only the root playbook.

## Backup And Safe Apply Protocol

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
- cookies, tokens, passwords, OTPs, browser sessions
- raw private data source captures
- client-confidential reports, customer data, approvals, or outputs

Safe apply rules:

- Replace toolkit/playbook/template/code files from the verified source.
- Merge local config schemas without deleting local values.
- Preserve `api_key_env`, `api_key_local`, provider identities, local account notes, and call logs.
- Preserve Client Intelligence Profiles, source approvals, private data source lists, history, reports, outputs, analytics, and publishing ledgers.
- Preserve each extension's `client_binding.json` identity while updating extension code/templates.
- Preserve per-client Chrome profile guidance and extension registry entries while updating schema fields.
- If a file has both new template content and user-local values, merge by schema instead of blind overwrite.
- If a merge is ambiguous, write an `update_conflict` record and ask the human before overwriting.

## Client And Automation Resync After Update

Any applied update is a playbook/behavior change and therefore triggers Automation Resync when schedule/automation exists.

For every active or configured client, check and update:

- Client Intelligence Profile schema fields when the latest playbook requires them.
- public data sources and keyword-bank schema fields when changed.
- private data source approval state when schema changed.
- `extensions/{client_slug}/` from the latest extension template while preserving binding.
- `daily-content-pipeline/collector/extension_registry.json`.
- `daily-content-pipeline/collector/collector_config.json` when collector schema changed.
- `daily-content-pipeline/provider_defaults.json`.
- the client's `integrations/providers/provider_capabilities.json`, `provider_openapi_cache.yaml`, and `provider_health.md` schema expectations when provider tooling changed.
- `daily-content-pipeline/schedule.md`.
- `daily-content-pipeline/automation/automation_manifest.md`.
- `daily-content-pipeline/automation/scheduled_run_prompt.md`.
- the native AI automation/scheduled task body when the environment exposes it.
- `daily-content-pipeline/automation/update_state.json` and `update_log.md`.
- `daily-content-pipeline/automation/resync_log.md`.

If the agent cannot edit the native scheduled task body directly, it must:

1. write the exact replacement prompt to `daily-content-pipeline/automation/scheduled_run_prompt.md`;
2. mark `automation_prompt_update_pending` in `automation_manifest.md` and `schedule.md`;
3. give the human one exact instruction to replace the native task prompt;
4. say `Automation Resync partially complete`, not `complete`.

## Bridge And Extension Change Protocol

Classify changes that touch these paths as bridge/runtime changes:

- Local Collector bridge Go source
- bridge binaries
- `solo-agency-local-collector/`
- setup/start/launcher scripts
- collector protocol or job routing code
- checksums or bridge artifact metadata

When bridge/runtime changes are applied:

- Prepare the updated files.
- Do not run `setup_collector.sh`, PowerShell setup scripts, `.cmd` launchers, or collector binaries from inside the AI agent during setup/update/repair.
- Tell the human the one-line command to run outside the AI sandbox.
- The command must use the current setup root, not the source checkout.
- After the human runs it, verify bridge status and workspace identity.

Example macOS/Linux command shape:

```bash
cd "{agency_root}" && bash "solo-agency-local-collector/setup_collector.sh"
```

Use the exact path created by the current setup. If this setup stores the launcher elsewhere, show that actual absolute path instead.

Classify changes that touch these paths as extension changes:

- Chrome extension template
- extension manifest/popup/background/content scripts
- `client_binding.json` schema
- extension permissions or host permissions
- extension bridge protocol fields

When extension changes are applied:

- Regenerate or patch every configured `extensions/{client_slug}/` folder.
- Preserve each client's `client_binding.json`.
- Show the human one absolute extension folder path per client.
- Tell the human to open the matching Chrome profile/account for each client.
- Tell the human: `chrome://extensions` -> Developer mode -> find `{Client Name} - Solo Agency Collector` -> Reload.
- If the extension cannot reload cleanly, tell the human to remove the old unpacked extension and `Load unpacked` from the shown `extensions/{client_slug}/` folder.
- Record `extension_reload_required: true` until the human confirms reload and the bridge sees fresh extension health.

## Daily GitHub Update Watch Task

After setup/routine exists, the agent should recommend a lightweight update-watch automation because Solo Agency changes frequently.

Canonical task name:

```text
Solo Agency - GitHub Update Watch
```

Recommended cadence:

```text
Daily, before client daily runs when possible.
```

The agent should explain in plain language:

```text
Solo Agency is updated often. A small daily update-watch task can check GitHub, notify you when a new version changes behavior, and, if you approve auto-apply, resync playbooks, collector, extensions, provider contracts, and scheduled tasks before the daily client runs.
```

The task must load this Stage 11 playbook and run only update/version-check work. It must not run client reports, private data source scans, production, publishing, or analytics.

Update-watch task algorithm:

1. Load `SOLO_AGENCY_PLAYBOOK.md`, Stage 7, Stage 9, and Stage 11.
2. Read `daily-content-pipeline/automation/update_state.json` when present.
3. Check GitHub `main` using the Fresh GitHub Checkout Protocol or a remote commit check.
4. Compare `installed_commit` and `latest_checked_commit`.
5. If no change, update `last_checked_at` and stop.
6. If a new commit exists, perform the Required Diff Scope.
7. Classify the change.
8. Notify the human/operator through the configured notification path when available, or write a local update notice.
9. If `auto_apply_approved: true`, apply the update, resync clients/tasks, and still require human bridge/extension actions when those files changed.
10. If `auto_apply_approved: false`, ask the human whether to apply the update.

Change classification values:

- `no_change`
- `playbook_only`
- `provider_tooling`
- `collector_bridge`
- `chrome_extension`
- `collector_bridge_and_extension`
- `setup_or_schedule_contract`
- `breaking_or_major_behavior`
- `unknown`

Notification content must include:

- installed commit
- latest GitHub commit
- change classification
- files/categories changed
- whether auto-apply is enabled
- whether bridge rerun is required
- whether extension reload is required
- whether automation/scheduled task prompt update is required
- exact next human action

## Update Logs And State

Maintain:

```text
daily-content-pipeline/automation/update_log.md
daily-content-pipeline/automation/update_state.json
daily-content-pipeline/automation/backups/
```

Minimum `update_state.json`:

```json
{
  "schema_version": 1,
  "installed_commit": "",
  "latest_checked_commit": "",
  "last_checked_at": "",
  "last_applied_commit": "",
  "last_applied_at": "",
  "auto_apply_approved": false,
  "update_watch_task_name": "Solo Agency - GitHub Update Watch",
  "last_change_classification": "",
  "bridge_update_required": false,
  "extension_reload_required": false,
  "automation_prompt_update_pending": false,
  "clients_resynced": [],
  "automations_resynced": [],
  "human_actions_required": []
}
```

Minimum `update_log.md` table:

```md
# Solo Agency Update Log

| Date | Agent | Local Commit Before | GitHub Main Commit | Change Classification | Applied | Backup Path | Clients Resynced | Automations Resynced | Bridge Action Required | Extension Reload Required | Blocker / Next Action |
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
- Toolkit/playbooks/templates/code were updated from the verified source when needed.
- Client configs were merged safely without losing local values or secrets.
- Every configured client extension folder was checked and updated when extension code changed.
- Every configured client and scheduled/automation task was resynced or a precise `automation_prompt_update_pending` blocker was logged.
- `update_state.json` and `update_log.md` were updated.
- If bridge/runtime changed, the human received the exact current-setup command to rerun the bridge outside the AI sandbox.
- If extension changed, the human received exact reload or `Load unpacked` steps for each client Chrome profile.
- The final human-facing message says whether update state is `complete`, `partial`, or `blocked`.

## Safety Audit Notes

- Treat update work as operator/internal work. Do not put update details into client-facing reports.
- Do not expose provider names, API-key status, Local Collector status, extension paths, GitHub commits, or update internals to the client's client/customer.
- If an update is triggered during a scheduled daily run, finish the update/resync only when safe. If it would delay a time-sensitive report, record the pending update and notify the operator, but do not mix partially updated playbooks with an in-progress report.
- If the latest GitHub version still does not resolve the blocker that triggered the update, follow the Last-Resort Recovery And GitHub Issue Escalation Rule.
