# Storage Schema And History

Stage: `07`

## Load Rule

Load when creating folders, saving profiles, updating logs, reading history, avoiding duplicate ideas, adding clients, or tracking published content and analytics.

## Hard Gates For This Stage

- Use the dedicated root folder `daily-content-pipeline/`.
- Use `client_intelligence_profile.md` as the canonical profile concept; do not use `ABC.md`.
- Store Markdown internally and HTML for humans.
- Track history to avoid duplicate ideas.
- Keep analytics, comments, learning, lead, competitor, source, and published-content logs.

## Latest Override: Shared Bridge, Per-Client Extension Folders

The current canonical runtime/data layout is:

```text
{agency_root}/
  solo-agency/                         # toolkit/source repo, no client data
  solo-agency-local-collector/         # shared Local Collector app/bridge runtime only
    downloads/
    bin/
    setup_collector.sh
    collector.pid
    collector.log
  extensions/                          # per-client Chrome Load unpacked folders
    {client_slug}/
      manifest.json
      background.js
      popup.html
      popup.js
      client_binding.json
  daily-content-pipeline/              # data/config/output only
    clients_index.md
    schedule.md
    provider_defaults.json             # public/provider-neutral catalog, no secrets
    automation/
      automation_manifest.md
      scheduled_run_prompt.md
      resync_log.md
      github_issues.md
      update_state.json
      update_log.md
      update_notice.md
      update_watch_prompt.md
      backups/
        update_YYYY-MM-DD_HHMMSS/
      issues/
        YYYY-MM-DD_{blocker_slug}.md
    collector/
      collector_setup_status.md
      collector_config.json
      extension_registry.json
      agent_registry.json
      jobs/
        pending/
        claimed/
        completed/
        failed/
      inbox/
        YYYY-MM/
          {client_slug}/
            YYYY-MM-DD_{client_slug}_{run_id}/
              collector_status.json
              private_data_points.jsonl
              leads.jsonl
              competitors.jsonl
              new_private_sources.jsonl
              source_status.jsonl
              snapshots/
      logs/
        bridge_events.jsonl
        extension_health.jsonl
        job_routing.jsonl
        agent_handoff.jsonl
    clients/
      {client_slug}/
        {business_slug}_{location_slug}/
          client_profile_{client_slug}_{business_slug}_{location_slug}.md
          integrations/
            providers/
              provider_config.local.json
              provider_capabilities.json
              provider_openapi_cache.yaml
              provider_calls.jsonl
              provider_health.md
          ...
```

Older references to a single `solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME/` folder are legacy. New setup must prepare one extension folder per client under `extensions/{client_slug}/`.

Per-client extension naming:

```text
{Client Name} - Solo Agency Collector
```

The client name must appear first because Chrome and task lists may truncate long names at the end.

Each `extensions/{client_slug}/client_binding.json` must include:

```json
{
  "client_slug": "avenngo",
  "client_name": "AvenNgo",
  "extension_instance_id": "ext_avenngo_default",
  "extension_display_name": "AvenNgo - Solo Agency Collector",
  "bridge_base_url": "http://127.0.0.1:17321"
}
```

## Source Preservation Rule

This file is detailed source material moved from the original monolithic `SOLO_AGENCY_PLAYBOOK.md`.

Do not summarize away requirements, examples, checklists, schemas, protocols, URLs, edge cases, warnings, approval gates, or completion gates. If a downstream agent needs to shorten its response to the human, it may summarize the response, but it must still obey the full requirements in this file.

---

## Canonical Profile Name Clarification

Use `client_intelligence_profile.md` as the canonical profile concept and schema name.

For multi-client slugged folders, a slugged profile filename may still be used when needed for uniqueness, but it must represent the Client Intelligence Profile schema. Do not use vague names such as `ABC.md`.

---

## 7. Folder Structure

Use one agency root folder:

```text
{agency_root}/
```

Use one folder per client/business/location:

```text
{agency_root}/
  solo-agency/                         # downloaded toolkit/source repo, no client data
  solo-agency-local-collector/         # shared bridge runtime app only
    downloads/
      collector-bridge-binaries-0.1.0.zip
      SHA256SUMS
    bin/
      collector-bridge-{os}-{arch}
    setup_collector.sh
    collector.pid
    collector.log
  extensions/                          # one Chrome extension folder per client
    {client_slug}/
      manifest.json                    # name starts with client name
      client_binding.json
      background.js
      popup.html
      popup.js
  daily-content-pipeline/              # data/config/output only
  provider_defaults.json               # default OpenAPI provider catalog, no secrets
  clients_index.md
  schedule.md
  automation/
    automation_manifest.md
    scheduled_run_prompt.md
    resync_log.md
    github_issues.md
    update_state.json
    update_log.md
    update_notice.md
    update_watch_prompt.md
    backups/
    github_issues.md
    update_state.json
    update_log.md
    update_notice.md
    update_watch_prompt.md
    backups/
      update_YYYY-MM-DD_HHMMSS/
    issues/
      YYYY-MM-DD_{blocker_slug}.md
  notifications/
    notification_log.md
  collector/
    collector_setup_status.md
    collector_config.json
    jobs/
      YYYY-MM/
        YYYY-MM-DD_client_slug.json
    inbox/
      YYYY-MM/
        YYYY-MM-DD_client_slug/
          collector_status.json
          private_data_points.jsonl
          leads.jsonl
          competitors.jsonl
          new_private_sources.jsonl
          source_status.jsonl
          snapshots/
  browser_profiles/
    {source_slug}/
  outputs/
    YYYY-MM/
      YYYY-MM-DD_master_digest.md
      YYYY-MM-DD_master_digest.html
    latest_master_digest.md
    latest_master_digest.html
  clients/
      {client_slug}/
        {business_slug}_{location_slug}/
        client_profile_{client_slug}_{business_slug}_{location_slug}.md
        strategy/
          offer_map.md
          brand_voice.md
          content_pillars.md
          funnel_map.md
        calendar/
          content_calendar.md
        approvals/
          approval_log.md
        assets/
          asset_index.md
        publishing/
          publishing_log.md
        analytics/
          metrics_log.md
        integrations/
          providers/
            provider_config.local.json
            provider_capabilities.json
            provider_openapi_cache.yaml
            provider_calls.jsonl
            provider_health.md
        reports/
          YYYY-MM_report.md
        experiments/
          experiment_backlog.md
        history/
          YYYY-MM/
            content_log.md
            data_sources_log.md
            lead_log.md
            competitor_log.md
            lead_competitor_opportunities.jsonl
            new_private_sources_log.md
        outputs/
          YYYY-MM/
            YYYY-MM-DD/
              {client-name}-daily-report.md
              {client-name}-public-data-sources-report.html
              {client-name}-private-data-sources-report.html
              {client-name}-daily-report.html
              {client-name}-INTERNAL_REPORT.md
              {client-name}-INTERNAL_REPORT.html
              {client-name}-client-report.html
              {client-name}-client-report.pdf
              {client-name}-report_state.json
          latest/
            {client-name}-daily-report.html
            {client-name}-public-data-sources-report.html
            {client-name}-private-data-sources-report.html
            {client-name}-INTERNAL_REPORT.html
            {client-name}-client-report.pdf
```

Examples:

```text
daily-content-pipeline/
  clients_index.md
  schedule.md
  provider_defaults.json
  automation/
    automation_manifest.md
    scheduled_run_prompt.md
    resync_log.md
  notifications/
    notification_log.md
  collector/
    downloads/
    bin/
      collector-bridge-darwin-arm64
      collector-bridge-windows-amd64.exe
      collector-bridge-linux-amd64
    chrome-extension/
    jobs/
      YYYY-MM/
    inbox/
      YYYY-MM/
  browser_profiles/
    facebook/
    linkedin/
  outputs/
    2026-06/
      2026-06-19_master_digest.md
      2026-06-19_master_digest.html
    latest_master_digest.md
    latest_master_digest.html
  clients/
    smith-law/
      dui_los-angeles/
        client_profile_smith-law_dui_los-angeles.md
        strategy/
          content_pillars.md
          funnel_map.md
        calendar/
          content_calendar.md
        approvals/
          approval_log.md
        analytics/
          metrics_log.md
        integrations/
          providers/
            provider_config.local.json
            provider_capabilities.json
            provider_openapi_cache.yaml
            provider_calls.jsonl
            provider_health.md
        reports/
          2026-06_report.md
        history/
          2026-06/
            content_log.md
            data_sources_log.md
            lead_log.md
            competitor_log.md
            lead_competitor_opportunities.jsonl
        outputs/
          2026-06/
            2026-06-19/
              smith-law-public-data-sources-report.html
              smith-law-private-data-sources-report.html
              smith-law-daily-report.html
              smith-law-INTERNAL_REPORT.md
              smith-law-INTERNAL_REPORT.html
              smith-law-client-report.html
              smith-law-client-report.pdf
              smith-law-report_state.json
          latest/
            smith-law-daily-report.html
            smith-law-INTERNAL_REPORT.html
            smith-law-client-report.pdf
    austin-home-group/
      realestate_austin/
        client_profile_austin-home-group_realestate_austin.md
        history/
          2026-06/
            content_log.md
            data_sources_log.md
        outputs/
          2026-06/
            2026-06-19/
              austin-home-group-daily-report.html
    bright-mortgage/
      mortgage_texas/
        client_profile_bright-mortgage_mortgage_texas.md
        history/
          2026-06/
            content_log.md
            data_sources_log.md
        outputs/
          2026-06/
            2026-06-19.md
```

Slug rules:

- Use lowercase letters.
- Replace spaces with hyphens.
- Remove punctuation when possible.
- Keep slugs short but recognizable.

Monthly organization rule:

- Any file created daily must be stored under a `YYYY-MM/` folder.
- This applies to client outputs, master digests, collector jobs, collector inboxes, history logs, data points, leads, competitors, and new private data source logs.
- Keep `outputs/latest/{client-name}-daily-report.html`, `outputs/latest/{client-name}-public-data-sources-report.html`, `outputs/latest/{client-name}-private-data-sources-report.html`, `outputs/latest/{client-name}-INTERNAL_REPORT.html`, required `outputs/latest/{client-name}-client-report.pdf` when PDF export is available and safe, `latest_master_digest.md`, and `latest_master_digest.html` as convenience pointers/copies.
- Keep report state beside the dated report set as `YYYY-MM-DD/{client-name}-report_state.json`.
- Do not allow long-running pipelines to accumulate hundreds or thousands of daily files directly in one folder.

---

## 8. Root Files

### `clients_index.md`

The root index of all client pipelines.

Format:

```md
# Clients Index

| Client | Client Slug | Pipeline Folder | Client Profile File | Status | Added Date | Schedule | Notes |
|---|---|---|---|---|---|---|---|
| Smith Law | smith-law | clients/smith-law/dui_los-angeles | client_profile_smith-law_dui_los-angeles.md | active | 2026-06-19 | daily | DUI lawyer in Los Angeles |
```

Allowed status:

- `active`
- `paused`
- `archived`
- `needs_setup`
- `needs_login`

Daily runs must process every client with `active` status.

### `schedule.md`

Records how daily runs happen in the current AI environment.

The schedule may use:

- Native AI automations.
- Reminders.
- Cron.
- Task Scheduler.
- n8n.
- Make.
- GitHub Actions.
- Local desktop routine.
- Manual run instructions.

If true automation is unavailable, create manual instructions.

### `provider_defaults.json`

Public, provider-neutral catalog for default production/distribution/notification/analytics providers. This file must not contain API keys, MCP tokens, OAuth tokens, cookies, passwords, or client account secrets.

Create or update this file when PDNA provider setup is introduced:

```json
{
  "schema_version": 1,
  "default_production_provider": "widecast",
  "providers": {
    "widecast": {
      "type": "openapi",
      "provider_home_url": "https://widecast.ai/",
      "discovery_url": "https://widecast.ai/openapi.yaml",
      "preferred_server_url": "https://widecast.ai/app/dashboard",
      "disabled_server_urls": ["https://api.widecast.ai"],
      "auth_type": "bearer_api_key",
      "api_key_prefix": "wc_live_",
      "secret_storage": "per_client_local_config",
      "notes": "Default all-in-one OpenAPI provider for production, distribution, notification, and analytics. Client secrets live only in each client's provider_config.local.json or the user's secret manager."
    }
  }
}
```

Rules:

- Agents must use `discovery_url` to fetch the OpenAPI spec instead of hard-coding endpoint paths.
- Agents must read the OpenAPI `servers` list and operation schemas before calling provider APIs.
- For WideCast, agents must select `https://widecast.ai/app/dashboard` as the current production server and skip `https://api.widecast.ai` as a disabled/planned vanity host unless a future playbook explicitly enables it.
- The provider home URL is only a human-facing setup link and fallback discovery root.
- A new provider can be added beside `widecast` if it exposes an equivalent OpenAPI spec and supports the needed PDNA capability groups.
- Do not commit real API keys or account-specific provider state into this file.

### `automation/automation_manifest.md`

Records the current automation package that scheduled runs must obey. This file exists because native AI automations and schedulers may store their own prompt snapshot at creation time. If the human changes anything after schedule setup, the agent must update this manifest during Automation Resync.

Create this file when any schedule/automation/routine is configured.

Minimum format:

```md
# Automation Manifest

- manifest_version: 1
- created_at:
- last_resynced_at:
- resync_status: current | automation_prompt_update_pending | partial | blocked
- scheduler_type: native_ai_automation | native_ai_scheduled_task | cron | launchd | task_scheduler | n8n | make | zapier | github_actions | server_job | manual
- scheduler_name:
- scheduler_location_or_url:
- timezone:
- schedule_file: daily-content-pipeline/schedule.md
- scheduled_prompt_file: daily-content-pipeline/automation/scheduled_run_prompt.md
- scheduled_entrypoint: playbooks/SCHEDULED_RUN_ENTRYPOINT.md
- root_playbook: SOLO_AGENCY_PLAYBOOK.md
- clients_index: daily-content-pipeline/clients_index.md
- collector_config: daily-content-pipeline/collector/collector_config.json
- provider_defaults: daily-content-pipeline/provider_defaults.json
- notification_channel:
- pdna_status:
- provider_status_summary:
- provider_capability_cache_status:
- private_data_source_status_summary:
- report_merge_contract: one_report_two_lanes | legacy_mixed_report | unknown
- report_notification_policy: same_report_public_private_notifications_allowed | single_final_notification | unknown
- latest_user_change_summary:
- actual_native_task_prompt_updated: true | false | not_applicable | unknown
- automation_prompt_update_pending_reason:
- automation_freshness_status: current | resync_in_progress | action_needed | not_applicable
- automation_freshness_summary: whether latest changes are synced into automation/scheduled task prompt/contract/playbook/source state, not only config, and whether tomorrow's run will load the newest state

## Active Clients

| Client | Client Slug | Profile Path | Status | Private Data Source Status | PDNA Status | Notes |
|---|---|---|---|---|---|---|

## Current Run Contract

- Scheduled runs must load the latest local playbooks at run time.
- Scheduled runs must read this manifest, schedule.md, provider_defaults.json, clients_index.md, active Client Intelligence Profiles, per-client provider config, and collector_config.json when private data sources are active or pending.
- Scheduled runs must not rely only on the prompt snapshot from the day the automation was created.

## Last Dry-Read Verification

- verified_at:
- verified_by_agent:
- result: pass | fail | partial
- next_scheduled_run_will_see:
- blockers:
```

### `automation/scheduled_run_prompt.md`

Stores the exact prompt that should be used by the native AI automation or scheduler. This should normally mirror `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` while pointing to the local workspace.

The agent must update this file during Automation Resync whenever a future scheduled run needs new behavior or newly approved state.

If the AI environment stores a separate prompt inside a native scheduled task and the agent cannot edit it directly, write the replacement prompt here and set `resync_status: automation_prompt_update_pending` in `automation_manifest.md`.

### `automation/resync_log.md`

Tracks all post-schedule changes and whether the automation package was fully synced.

Format:

```md
# Automation Resync Log

| Date | Agent | Human Change | Files Updated | Native Task Prompt Updated | Dry-Read Result | Remaining Blocker | Next Scheduled Run Expected Behavior |
|---|---|---|---|---|---|---|---|
| 2026-06-23 | Claude | Approved 12 Facebook groups as private data sources | profile, schedule.md, collector_config.json, automation_manifest.md, scheduled_run_prompt.md | yes | pass | none | Scan approved groups via Local Collector |
```

### `automation/github_issues.md`

Tracks GitHub issues, support/intake submissions, or issue drafts created when latest GitHub playbooks/code still do not resolve a blocker.

Create or update this file when Last-Resort Recovery opens, sends, queues, or drafts an issue. The human does not need a GitHub account; direct GitHub creation uses an authorized agent/runtime identity when available, and otherwise falls back to a configured intake channel or local draft.

Format:

```md
# GitHub Issue Tracker

| Date | Agent | Client Slug | Blocker Fingerprint | Local Commit | GitHub Main Commit Checked | Issue URL / Intake Channel / Draft Path | Status | Next Check | Latest Response / Next Action |
|---|---|---|---|---|---|---|---|---|---|
| 2026-06-24 | Codex | smith-law | collector_wrong_workspace_bridge_after_fresh_check | abc123 | def456 | https://github.com/soloagency/solo-agency/issues/123 | opened_by_agent | 2026-06-25 | Waiting for maintainer response |
```

Issue draft files belong under:

```text
daily-content-pipeline/automation/issues/YYYY-MM-DD_{blocker_slug}.md
```

Recommended status values: `opened_by_agent`, `sent_to_intake`, `queued_for_intake`, `draft_waiting_for_support_channel`, `draft_waiting_for_human`, `answered`, `fix_applied`, `resolved`, `closed`.

Each issue, intake submission, or draft must be redacted. Do not include API keys, cookies, tokens, passwords, raw private data source content, client-confidential details, raw logged-in screenshots, or sensitive customer data. Include only safe reproduction steps, expected/actual behavior, local commit, GitHub main commit checked, runtime, relevant blocker names, and redacted logs.

### `automation/update_state.json`

Tracks the installed Solo Agency version, latest GitHub check, auto-apply preference, bridge/extension action requirements, and resync state.

Create this file when the first update check runs, when the `Solo Agency - GitHub Update Watch` task is created, or when Stage 11 applies an update.

Minimum schema:

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

Do not store secrets, private data source content, client-confidential report content, cookies, tokens, or raw provider responses in this file.

### `automation/update_log.md`

Tracks every GitHub update check and every applied update.

Format:

```md
# Solo Agency Update Log

| Date | Agent | Local Commit Before | GitHub Main Commit | Change Classification | Applied | Backup Path | Clients Resynced | Automations Resynced | Bridge Action Required | Extension Reload Required | Blocker / Next Action |
|---|---|---|---|---|---|---|---|---|---|---|---|
```

Recommended change classifications:

- `no_change`
- `playbook_only`
- `provider_tooling`
- `collector_bridge`
- `chrome_extension`
- `collector_bridge_and_extension`
- `setup_or_schedule_contract`
- `breaking_or_major_behavior`
- `unknown`

### `automation/update_notice.md`

Internal/local notice for the latest GitHub update-watch outcome.

Use this file when a new Solo Agency version is available, an update was applied, auto-apply is disabled and the human needs to decide, or bridge/extension human action is required.

Do not send update-watch notices through Telegram, WideCast/email fallback, provider notification channels, social posting, or client notification channels. Version checks and applied updates are internal user/agency maintenance, not client delivery. Do not put update-watch rows in `notifications/notification_log.md`; that log is for report/result delivery and related operational notifications.

Minimum content:

```md
# Solo Agency Update Notice

- checked_at:
- installed_commit:
- latest_github_commit:
- change_classification:
- auto_apply_approved:
- update_applied:
- bridge_action_required:
- extension_reload_required:
- automation_prompt_update_pending:
- next_human_action:
```

### `automation/update_watch_prompt.md`

Stores the exact prompt for the native maintenance automation task named `Solo Agency - GitHub Update Watch` when the current AI runtime cannot create or edit that task directly.

The prompt must come from `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` and must load Stage 11. It must not include client report-generation instructions beyond explicitly saying not to run reports/scans/production.

### `automation/backups/`

Stores timestamped update backups:

```text
daily-content-pipeline/automation/backups/update_YYYY-MM-DD_HHMMSS/
```

Use it for runtime files or folders that Stage 11 replaces. Do not use it as a long-term archive for private data source captures, client reports, secrets, cookies, tokens, or provider API keys.

### `notifications/notification_log.md`

Tracks notifications sent to the human through the configured provider notification channel, WideCast OpenAPI/Telegram/email fallback, or any other authorized notification channel.

Format:

```md
# Notification Log

| Date | Agent | Event | Lane Status | Channel | Status | HTML Report Path | PDF Report Path | PDF Status | Provider | Provider Discovery Checked | Upload Operation | Notification Operation | Upload Attempted | Uploaded HTML URL | Uploaded PDF URL | Notification Attempted | Final Report Link Sent | Blocker | Action Needed |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-06-20 | Claude Schedule | daily_run_completed | public_report_ready | WideCast Telegram/email fallback | sent | outputs/2026-06/2026-06-20/angela-do-daily-report.html | outputs/2026-06/2026-06-20/angela-do-client-report.pdf | generated | widecast | yes | uploadAsset | sendTelegramMessage | yes | https://... |  | yes | https://... | none | Await private report or review approvals |
| 2026-06-20 | Claude Schedule | daily_run_completed | private_report_ready | WideCast Telegram/email fallback | sent | outputs/2026-06/2026-06-20/angela-do-daily-report.html | outputs/2026-06/2026-06-20/angela-do-client-report.pdf | generated | widecast | yes | uploadAsset | sendTelegramMessage | yes | https://... |  | yes | https://... | none | Review daily report index and lane reports |
```

Use this log so scheduled runs do not silently complete or fail while the human is away.

Each provider-backed notification row or adjacent structured record should also preserve `provider_identity_source` and `mcp_compatibility_status` when available. For client-scoped OpenAPI delivery, `provider_identity_source` should be `per_client_openapi`. If a global MCP/native provider account was visible but not proven to match the client, record `mcp_compatibility_status: not_client_scoped` and blocker `global_mcp_not_client_scoped`.

If provider upload or notification cannot be used, the log must distinguish:

- `provider_config_missing`: no per-client provider config exists.
- `provider_auth_missing`: provider config exists but the client has not supplied an API key or supported auth value.
- `provider_auth_failed`: the provider rejected the client credential.
- `provider_discovery_failed`: OpenAPI discovery URL could not be fetched or parsed.
- `provider_required_operation_missing`: the OpenAPI spec lacks the operation needed for the requested action.
- `provider_account_mismatch`: provider account verification does not match the saved client/account identity.
- `global_mcp_not_client_scoped`: an MCP/native provider tool is visible in the AI session, but it is not proven to be authenticated as the current client's configured provider account.
- `provider_upload_failed`: upload operation exists but the upload call failed.
- `provider_notification_failed`: notification operation exists but send failed.
- `provider_notification_not_configured`: provider account is valid but Telegram/email/notification destination is not configured and no fallback was sent.

WideCast-specific aliases may still be logged for backward compatibility:

- `widecast_report_upload_unavailable` means the current provider/OpenAPI capability check or legacy connector path exposed no HTML-capable upload operation.
- `widecast_notification_tool_unavailable` means the current provider/OpenAPI capability check or legacy connector path exposed no WideCast notification send operation.
- `widecast_upload_failed`, `widecast_notification_failed`, and `widecast_telegram_not_connected` keep their legacy meaning but should be accompanied by the provider-neutral blocker when possible.

Do not use `unavailable` generically when the actual issue is missing config, failed auth, missing provider operation, expired credentials, or a provider account mismatch.

### Per-Client Provider Integration Files

Each client that uses PDNA provider actions must keep provider state under:

```text
daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/integrations/providers/
```

#### `provider_config.local.json`

Client-local configuration. Treat this file as sensitive local state. Do not include it in a public repo, zip, screenshot, report, or support bundle unless secrets are removed.

Minimum WideCast OpenAPI example:

```json
{
  "schema_version": 1,
  "client_slug": "angela-do",
  "active_provider": "widecast",
  "providers": {
    "widecast": {
      "type": "openapi",
      "discovery_url": "https://widecast.ai/openapi.yaml",
      "provider_home_url": "https://widecast.ai/",
      "preferred_server_url": "https://widecast.ai/app/dashboard",
      "disabled_server_urls": ["https://api.widecast.ai"],
      "auth_type": "bearer_api_key",
      "api_key_env": "SOLO_AGENCY_WIDECAST_API_KEY_ANGELA_DO",
      "api_key_local": "",
      "provider_identity_source": "per_client_openapi",
      "mcp_compatibility_status": "not_used",
      "pdna_setup_blocker": "",
      "account_verified_at": "",
      "account_identity": {
        "company_id": "",
        "email_masked": "",
        "name": "",
        "connected_platforms": []
      },
      "pdna": {
        "production": "not_configured",
        "distribution": "not_configured",
        "notification": "not_configured",
        "analytics": "not_configured"
      },
      "notification": {
        "enabled": false,
        "preferred_operation_id": "sendTelegramMessage",
        "delivery": "telegram_or_email_fallback"
      },
      "report_upload": {
        "enabled": false,
        "preferred_operation_id": "uploadAsset",
        "content_type": "text/html",
        "ttl_hours_note": "WideCast uploadAsset URLs are currently short-lived. Use for report notifications, not permanent archives."
      }
    }
  }
}
```

Credential rules:

- Prefer `api_key_env` or the user's secret manager when available.
- If a local API key is saved in `api_key_local`, keep it only in this per-client local file and redact it in all logs and reports.
- Do not create or use a field named `api_key` in `provider_config.local.json`. The official helper reads `api_key_env` and `api_key_local`; a stray `api_key` field is ignored and will cause `provider_auth_missing`.
- Never store passwords, OTPs, browser cookies, social session tokens, or raw OAuth refresh tokens here.
- Before any provider action, verify the active provider account with the provider account operation, such as WideCast `getAccount`.
- Before checking global MCP/native tools, check this per-client provider config plus OpenAPI/cache/capability files as Client tools. Global MCP/native tools are only compatibility after identity match.
- If the verified account identity changes unexpectedly, stop provider actions and log `provider_account_mismatch`.
- `provider_identity_source` must be `per_client_openapi` before PDNA is considered connected. `global_mcp_compat` is allowed only when the MCP/native tool identity has been compared to the saved client provider identity and matches exactly.
- `mcp_compatibility_status` may be `not_used`, `identity_matched`, `identity_mismatch`, or `not_client_scoped`. If it is `identity_mismatch` or `not_client_scoped`, do not use MCP/native account data for this client's PDNA status.
- `pdna_setup_blocker` should use provider-neutral blocker names such as `provider_config_missing`, `provider_auth_missing`, `provider_auth_failed`, `provider_discovery_failed`, `provider_account_mismatch`, or `global_mcp_not_client_scoped`.
- For WideCast, `preferred_server_url` must stay `https://widecast.ai/app/dashboard` and `disabled_server_urls` must include `https://api.widecast.ai` until a future playbook explicitly enables that host.

#### `provider_capabilities.json`

Snapshot of the OpenAPI operations discovered for the active provider. This is the main Client tools inventory for provider actions and is safe to keep without secrets.

When local Python execution is available, agents may create or refresh this file with `tools/provider_openapi.py discover --config <client provider_config.local.json> --defaults daily-content-pipeline/provider_defaults.json --out-dir <client integrations/providers folder>`.

Whenever the human or automation asks to check tools, check this Client tools file first. Only inspect global MCP/native tools after this file and the verified provider identity are current.

Minimum shape:

```json
{
  "schema_version": 1,
  "provider": "widecast",
  "discovered_at": "",
  "discovery_url": "https://widecast.ai/openapi.yaml",
  "server_url": "https://widecast.ai/app/dashboard",
  "server_urls_discovered": [],
  "server_urls_skipped_disabled": ["https://api.widecast.ai"],
  "auth_scheme": "bearerAuth",
  "operation_ids": {
    "account": "getAccount",
    "analytics": "getAnalytics",
    "upload_html_report": "uploadAsset",
    "send_notification": "sendTelegramMessage",
    "publish": "publish",
    "create_video": "createVideo",
    "export_video": "exportVideo",
    "get_status": "getStatus",
    "get_video_data": "getVideoData",
    "get_writing_skill": "getWritingSkill",
    "create_content": "createContent",
    "create_image": "createImage",
    "search_broll": "searchBroll",
    "collect_ideas": "collectIdeas"
  },
  "capability_status": {
    "production": "available | partial | unavailable",
    "distribution": "available | partial | unavailable",
    "notification": "available | partial | unavailable",
    "analytics": "available | partial | unavailable"
  },
  "identity": {
    "provider_identity_source": "per_client_openapi | global_mcp_compat | unknown",
    "account_verified": true,
    "mcp_compatibility_status": "not_used | identity_matched | identity_mismatch | not_client_scoped"
  },
  "blockers": []
}
```

#### `provider_openapi_cache.yaml`

Raw OpenAPI spec cache for repeatable automation. Refresh it when:

- the file is missing;
- the cache is older than the configured refresh policy;
- `provider_defaults.json` changes;
- the provider action fails because an operation/schema appears stale;
- the human changes provider configuration.

#### `provider_calls.jsonl`

Append-only provider audit log. Each line should include timestamp, agent, client_slug, provider, operationId, redacted request summary, response status, request_id if present, and blocker if any. Never log full API keys or private data source raw content.

#### `provider_health.md`

Human-readable provider status:

```md
# Provider Health

| Date | Agent | Provider | Identity Source | MCP Compatibility | Account Verified | Production | Distribution | Notification | Analytics | Credits | Connected Platforms | Blocker | Next Action |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
```

### `collector/collector_setup_status.md`

Tracks whether the Solo Agency Local Collector extension and Local Collector app are installed, reachable, blocked, pending activation, writing to the wrong setup folder, or waiting for human action.

This file is mandatory after the human agrees to activate private data source monitoring, when configuring a schedule that includes private data sources, or when the agent needs to report a private data source collector blocker.

It is not required when no private data sources are active. Before activation, the automation contract and automation report should simply list private data sources under `Private Data Sources Pending Activation`.

Format:

```md
# Collector Setup Status

| Date | Agent | Status | Setup Command Given | Human Ran Setup Command | Chrome Extension Folder | Human Loaded Extension | Local Collector App | Health Endpoint | Last Health Check | Blocker | Required Human Action |
|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-06-20 | Claude | needs_user_action | bash "/ABSOLUTE/PATH/solo-agency-local-collector/setup_collector.sh" | no | /ABSOLUTE/PATH/extensions/{client_slug}/ | no | /ABSOLUTE/PATH/solo-agency-local-collector/bin/collector-bridge-darwin-arm64 | http://127.0.0.1:17321/status | unavailable | Client-specific extension not loaded in Chrome yet | Run the setup command in Terminal/PowerShell outside the AI sandbox, then Chrome -> chrome://extensions -> Load unpacked -> select only the absolute client-specific extension folder |
```

Allowed status:

- `not_needed_no_private_sources`
- `pending_private_activation`
- `setup_files_prepared_waiting_for_human_command`
- `setup_command_given_waiting_for_human_run`
- `setup_command_ran_waiting_for_extension`
- `activation_declined_for_now`
- `installed_and_running`
- `installed_not_running`
- `needs_user_action`
- `blocked_by_sandbox`
- `blocked_by_os_permission`
- `extension_not_loaded`
- `extension_stale`
- `bridge_offline`
- `wrong_workspace_bridge`
- `session_expired`
- `failed`

The agent must update this file before:

- claiming private data source monitoring is active,
- running a manual private data source scan,
- configuring recurring private data source collection,
- reporting that private collection is unavailable.

### `outputs/YYYY-MM/YYYY-MM-DD/{client-name}-report_state.json`

Tracks the state for the canonical three-file daily report set. It prevents a later public or private data source pass from overwriting or summarizing away the other lane's full HTML report.

Minimum format:

```json
{
  "client_slug": "",
  "run_id": "",
  "report_date": "",
  "report_dir": "outputs/YYYY-MM/YYYY-MM-DD/",
  "report_md_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-daily-report.md",
  "public_report_html_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-public-data-sources-report.html",
  "private_report_html_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-private-data-sources-report.html",
  "daily_report_html_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-daily-report.html",
  "latest_daily_html_path": "outputs/latest/{client-name}-daily-report.html",
  "latest_public_html_path": "outputs/latest/{client-name}-public-data-sources-report.html",
  "latest_private_html_path": "outputs/latest/{client-name}-private-data-sources-report.html",
  "internal_report_md_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-INTERNAL_REPORT.md",
  "internal_report_html_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-INTERNAL_REPORT.html",
  "latest_internal_report_html_path": "outputs/latest/{client-name}-INTERNAL_REPORT.html",
  "client_report_html_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-client-report.html",
  "client_report_pdf_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-client-report.pdf",
  "latest_client_pdf_path": "outputs/latest/{client-name}-client-report.pdf",
  "internal_report_status": "pending",
  "client_facing_scrub_status": "pending",
  "client_facing_scrub_blocker": "",
  "client_pdf_status": "pending",
  "client_pdf_redaction_status": "not_needed",
  "client_pdf_generated_at": "",
  "client_pdf_blocker": "",
  "public_section_status": "missing",
  "private_section_status": "missing",
  "last_public_update_at": "",
  "last_private_update_at": "",
  "public_data_sources_count": 0,
  "private_data_sources_count": 0,
  "public_sources_attempted": 0,
  "public_sources_completed": 0,
  "public_sources_blocked_or_skipped": 0,
  "private_sources_attempted": 0,
  "private_sources_completed": 0,
  "private_sources_blocked_or_skipped": 0,
  "public_data_points_kept": 0,
  "private_data_points_kept": 0,
  "public_lead_count": 0,
  "private_lead_count": 0,
  "public_watch_lead_count": 0,
  "private_watch_lead_count": 0,
  "public_competitor_count": 0,
  "private_competitor_count": 0,
  "public_new_sources_recommended_count": 0,
  "private_new_sources_recommended_count": 0,
  "private_noisy_or_skipped_discovery_candidates_count": 0,
  "counts_reconciled_at": "",
  "public_notification_status": "not_sent",
  "private_notification_status": "not_sent",
  "last_notification_report_path": "",
  "last_notification_lane": "daily",
  "last_notification_report_url": "",
  "last_update_agent": "",
  "last_update_note": ""
}
```

Allowed section status:

- `missing`
- `pending`
- `complete`
- `complete_live_scan`
- `skipped`
- `failed`
- `blocked`

Allowed notification status:

- `not_sent`
- `sent`
- `skipped`
- `failed`

Allowed `client_pdf_status`:

- `pending`
- `generated`
- `pending_review`
- `blocked`

Rules:

- Public data source pass may write only `{client-name}-public-data-sources-report.html`, public source records, and `{client-name}-daily-report.html` status metadata. It must preserve any existing private report.
- Private data source pass may write only `{client-name}-private-data-sources-report.html`, private source records, and `{client-name}-daily-report.html` status metadata. It must preserve any existing public report.
- After a public or private data source pass reaches a terminal state, the state file counts must be reconciled with the lane report, daily index, internal source record, notification log, and latest copies. Do not leave stale `partial`, `pending`, `scan in progress`, or old recommended-source totals in one artifact when another artifact reports completion.
- `latest/{client-name}-daily-report.html` must point to or copy the daily report index, not a lane-specific artifact.
- `latest/{client-name}-INTERNAL_REPORT.html` must point to or copy the operator-only internal report and must be clearly labeled `INTERNAL_REPORT - Not for client sharing`.
- Client-facing files and the client PDF must pass the client-blind scrub gate before handoff. If not, keep `client_facing_scrub_status: failed` or `blocked`, record the blocker, and do not present the file as client-ready.
- `latest/{client-name}-client-report.pdf` is required when PDF export is available and safe, and must point to or copy a PDF generated from `{client-name}-client-report.html`, which itself is assembled from the three canonical HTML reports. It must not replace the daily report index. If PDF export is unavailable or unsafe, keep `client_pdf_status: blocked` and record `client_pdf_blocker`.
- If a client-share PDF includes private data source findings, record `client_pdf_redaction_status` as `redacted`, `approved_exact_sources`, or `needs_human_review`.
- If two notifications are sent, both should reference the same daily report path or uploaded URL, with lane status recorded in `notification_log.md`. Lane-specific links may be included as secondary links.

### `outputs/YYYY-MM/YYYY-MM-DD_master_digest.md`

Daily summary across all active clients.

It should include:

- Date.
- Clients processed.
- Clients skipped.
- For each client:
  - Top ideas.
  - Best idea.
  - Script file path.
  - Reference URLs for top ideas and the selected best idea.
  - Sources skipped.
  - Required human action.

---

## 9. Client Intelligence Profile Schema

Each client pipeline must have one Client Intelligence Profile file.

Filename:

```text
client_profile_{client_slug}_{business_slug}_{location_slug}.md
```

Template:

```md
# Client Intelligence Profile: {client_name}

## Metadata

- client_name:
- client_slug:
- business_slug:
- location_slug:
- created_date:
- last_reviewed_date:
- status: active

## business_description

value:
status:
rationale:

## output_formats

status:
default: video_script
items:
- format: video_script | blog_article | social_caption
  cadence: daily | weekly | on_request
  widecast_skill_format: video | blog | social
  notes:

## industry

value:
status:
rationale:

## sub_industry

value:
status:
rationale:

## related_industries

status:
rationale:
content_mix_rule: approximately 80% primary industry / 20% related industries
items:
- name:
  relationship_to_primary_industry:
  why_it_matters_to_target_audience:
  example_content_bridges:
  allowed_use: signal_source | content_angle | data_source | lead_signal | competitor_context
  priority: high | medium | low

## target_audience

value:
status:
rationale:

## target_location

value:
status:
rationale:

## location_dependency

value: high | medium | low
status:
rationale:

## business_offer

value:
status:
rationale:

## pain_points

status:
rationale:
items:
-

## content_pillars

status:
rationale:
content_mix_rule: approximately 80% primary industry / 20% related industries
items:
- name:
  industry_scope: primary_industry | related_industry
  related_industry:
  mapped_pain_points:
  strategic_purpose:
  example_angles:
  bridge_back_to_primary_offer:
  lead_gen_connection:

## public_data_sources

items:
- name:
  url:
  type: public
  platform:
  source_status: candidate_public_source | active_public_source | weekly_public_source | occasional_public_source | weak_public_source | blocked_or_unreliable
  source_kind: official | regulator | government | news | specialist_blog | public_forum | public_social | public_video_channel | competitor_public | data_dashboard | newsletter_archive | association | local_community | search_result | other
  language:
  scan_cadence: daily | weekly | occasional | event_based | paused
  visit_in_scheduled_runs: true | false
  location_relevance:
  related_pain_points:
  related_content_pillars:
  related_keywords:
  why_this_source_matters:
  source_or_reason_added:
  discovered_from:
  first_discovered_date:
  last_checked_date:
  useful_count:
  weak_count:
  usefulness_score:
  promoted_date:
  demoted_date:
  access_method:
  collection_notes:

## public_search_keywords

summary:
  total_keywords:
  hidden_keywords_saved:
  primary_keyword_language:
  secondary_keyword_languages:
  needs_expansion: true | false
  last_expanded_date:
  expansion_sources:
    - setup_inference
    - public_search_results
    - private_source_scan
    - competitor_hooks
    - report_comments
    - analytics_learning
    - human_feedback

items:
- keyword:
  language:
  status: unused | used | useful | weak | retry_later
  keyword_group: industry_general | pain_point | need_or_goal | buying_intent | local_context | related_industry | trend_news | objection | comparison | question | problem_issue
  scope: global | local
  industry_scope: primary_industry | related_industry
  related_industry:
  related_content_pillar:
  related_pain_point:
  related_customer_need:
  source_or_reason_added:
  discovered_from:
  first_added_date:
  last_used_date:
  use_count:
  useful_count:
  weak_count:
  result_quality:
  promoted: true | false
  demoted: true | false
  notes:

## private_monitoring_activation

status: not_provided | pending_private_activation | activation_declined_for_now | activation_requested | setup_files_prepared_waiting_for_human_command | setup_command_given_waiting_for_human_run | setup_command_ran_waiting_for_extension | installed_and_running | wrong_workspace_bridge | blocked
first_trial_policy: public_first_small_win
last_prompted_date:
human_decision:
collector_setup_status_file:
notes:

## private_data_source_discovery

status: not_asked | recommended | declined | postponed | partially_approved | approved | pending_human_approval | pending_private_activation | active | blocked | completed | discovery_declined_or_postponed
reassurance_shown:
  professional_setup_once: true | false
  local_data_only: true | false
  daily_scan_prevents_missed_signals: true | false
why_recommended:
coverage_limitation_if_skipped:
categories:
  membership_sources:
    status: not_asked | recommended | declined | postponed | approved | pending_human_approval | pending_private_activation | active | blocked | completed
    platforms:
    - platform:
      discovery_urls:
      - url:
        status: not_tried | pending_private_activation | scanned | login_required | platform_url_changed | failed
        last_scanned_at:
  following_sources:
    status: not_asked | recommended | declined | postponed | approved | pending_human_approval | pending_private_activation | active | blocked | completed
    platforms:
    - platform:
      discovery_urls:
      - url:
        status: not_tried | pending_private_activation | scanned | login_required | platform_url_changed | failed
        last_scanned_at:
  recommendation_feed_sources:
    status: not_asked | recommended | declined | postponed | approved | pending_human_approval | pending_private_activation | active | blocked | completed
    platforms:
    - platform:
      discovery_urls:
      - url:
        status: not_tried | pending_private_activation | scanned | login_required | platform_url_changed | failed
        last_scanned_at:
  keyword_search_sources:
    status: not_asked | recommended | declined | postponed | approved | pending_human_approval | pending_private_activation | active | blocked | completed
    platforms:
    - platform:
      search_keywords:
      - keyword:
        search_url:
        scroll_steps:
        candidate_count:
        recommended_count:
        skipped_noisy_count:
        status: not_tried | pending_private_activation | scanned | login_required | platform_url_changed | failed
        last_scanned_at:
candidate_source_review_policy:
  require_human_approval_before_activating: true
  max_daily_sources_default: 20
  feed_surfaces_are_discovery_only: true
last_discovery_report:

## private_data_sources

items:
- name:
  url:
  type: private
  platform:
  source_type: manually_provided | joined_group | facebook_group_search_result | followed_profile | followed_page | subscribed_channel | followed_company | subreddit | community | discovered_from_feed
  discovery_category: manually_provided | membership_sources | following_sources | recommendation_feed_sources | keyword_search_sources
  discovery_url:
  search_keyword:
  search_url:
  result_rank:
  membership_status: unknown | joined | not_joined | public_visible | requires_join | unavailable
  approval_status: pending_human_approval | approved | rejected
  priority: high | medium | low
  scan_cadence: daily | weekly | optional
  location_relevance:
  why_this_source_matters:
  access_method:
  collection_notes:
  activation_status: pending_private_activation | active | declined_for_now | unavailable
  login_status: unknown | available | expired | unavailable

## collector_config

status:
run_mode: agent_on_demand | persistent_bridge_scheduler | manual
default_runs_per_day: 1
scheduled_windows:
- name: morning
  enabled: true
  local_time_start: "09:00"
  local_time_end: "09:30"
  timezone:
max_sources_per_run: 20
max_scrolls_per_source: 5
max_scrolls_allowed: 10
scroll_delay_seconds: 5
duplicate_filter:
  compare_against_previous_day: true
  method: visible_text_matching
  parse_html: false
collector_panel:
  show_current_source: true
  show_scroll_count: true
  show_data_point_count: true
  show_status: true

## automation_sync

status: current | needs_resync | automation_prompt_update_pending | partial | blocked
last_profile_change_at:
last_profile_change_summary:
last_resynced_at:
last_resynced_by_agent:
automation_manifest_file: daily-content-pipeline/automation/automation_manifest.md
scheduled_prompt_file: daily-content-pipeline/automation/scheduled_run_prompt.md
schedule_file: daily-content-pipeline/schedule.md
collector_config_file: daily-content-pipeline/collector/collector_config.json
native_task_prompt_updated: true | false | not_applicable | unknown
dry_read_verification:
  verified_at:
  result: pass | fail | partial
  scheduled_run_will_see:
  blockers:

## brand_voice

value:
status:
rationale:

## language

human_report_language:
target_audience_language:
keyword_language:
secondary_keyword_languages:
content_output_language:
status:
rationale:

## platforms

value:
status:
rationale:

## compliance_notes

value:
status:
rationale:

## negative_topics

value:
status:
rationale:

## assumptions

-

## human_corrections

-
```

---

## 10. History Files

### `history/YYYY-MM/content_log.md`

Purpose:

- Avoid repeating the same idea too often.
- Track selected ideas, scripts, approvals, videos, and outcomes.
- Track whether each selected idea was `primary_industry` or `related_industry` so the agent can maintain the 80/20 content mix over time.
- Track idea signatures, angles, and novelty decisions so future runs can reuse a topic only when the angle is materially different.

Format:

```md
# Content Log

| Date | Idea | Idea Signature | Angle | Category | Scope | Industry Scope | Related Industry | Content Pillar | Prior Related Idea/Date | Novelty Decision | Script Path | Status | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-06-19 | Austin inventory is rising again | austin-inventory-buyer-strategy | rising inventory changes buyer offer strategy | Hot / Trend / News | Local | primary_industry |  | Local market intelligence |  | new | outputs/2026-06/2026-06-19.md | drafted | Not yet approved |
| 2026-06-20 | Why rising insurance premiums change your homebuying budget | insurance-premiums-homebuying-budget | insurance costs change affordability math | Hot / Trend / News | Local | related_industry | P&C insurance | Affordability clarity | 2026-06-12: monthly payment shock | new_angle | outputs/2026-06/2026-06-20.md | drafted | Related-industry idea connected back to buyer affordability |
```

Allowed status:

- `drafted`
- `approved`
- `video_created`
- `published`
- `rejected`
- `revised`
- `skipped`

### `history/YYYY-MM/data_sources_log.md`

Purpose:

- Track source checks.
- Track unavailable sources.
- Track private login/session failures.
- Track platform warnings, rate limits, checkpoints, and conservative pacing decisions.
- Avoid silently losing coverage.

Format:

```md
# Data Sources Log

| Date | Source | Type | Source URL | Status | Data Collected | Issue | Next Action |
|---|---|---|---|---|---|---|---|
| 2026-06-19 | Competitor FB Page A | private | https://www.facebook.com/... | skipped | no | session expired | Human must log in manually |
```

Allowed status:

- `checked`
- `collected`
- `skipped`
- `blocked`
- `session_expired`
- `rate_limited`
- `platform_warning`
- `collector_unavailable`
- `extension_unavailable`
- `extension_stale`
- `bridge_offline`
- `captcha_or_checkpoint`
- `chrome_not_running`
- `not_relevant_today`
- `unavailable`

### `history/YYYY-MM/lead_log.md`

Purpose:

- Track potential hot and warm leads discovered during public/private data source scanning.
- Preserve source URLs and reasoning for why the lead may be relevant.
- Avoid losing sales opportunities discovered during content research.

Format:

```md
# Lead Log

| Date | Lead Level | Source | Source Type | Profile URL | Post/Current URL | Safe Lead Summary | Related Offer | Related Pain Point | Suggested Next Action | Copy-Ready Suggested Comment | Status | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-06-20 | hot | Facebook Group | private | https://www.facebook.com/profile.php?id=... | https://www.facebook.com/groups/.../posts/... | Person asked for a DUI lawyer in Los Angeles | DUI legal consultation | Fear of license/court consequences | Human should review and decide whether to respond | Short value-first comment in the post language | needs_review | Do not contact automatically |
```

Allowed status:

- `needs_review`
- `approved_for_outreach`
- `contacted`
- `not_relevant`
- `do_not_contact`
- `converted`
- `skipped`

### `history/YYYY-MM/competitor_log.md`

Purpose:

- Track direct, adjacent, and audience competitors discovered during source scanning.
- Preserve competitor URLs, positioning notes, content patterns, and engagement signals.
- Help the agent improve positioning, content pillars, and idea selection over time.

Format:

```md
# Competitor Log

| Date | Competitor Type | Name/Page | Platform | Profile URL | Post/Current URL | Location Relevance | Audience Overlap | Offer/Positioning | Content Themes | Engagement Signal | Threat Level | Opportunity | Copy-Ready Suggested Comment | Status |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-06-20 | direct | Example DUI Law Firm | Facebook | https://www.facebook.com/exampleduilaw | https://www.facebook.com/exampleduilaw/posts/... | Los Angeles | Drivers facing DUI/legal issues | Free consultation for DUI cases | License suspension, court mistakes | Repeated comments asking for help | medium | Create clearer local process education | Short respectful value-first comment in the post language | monitoring |
```

Allowed status:

- `monitoring`
- `high_priority`
- `not_relevant`
- `archived`

### `history/YYYY-MM/lead_competitor_opportunities.jsonl`

Purpose:

- Store every report-ready lead and competitor opportunity in one machine-readable ledger.
- Preserve the post/current URL, safe context, classification, suggested human action, and copy-ready comment.
- Support future learning about which sources, pain points, competitor posts, and comment styles create the best opportunities.
- Keep lead and competitor opportunity analysis separate from raw private collector text.

Use this ledger in addition to `lead_log.md` and `competitor_log.md` when the environment can write JSONL.

Format:

```json
{
  "date": "2026-06-20",
  "client_slug": "example-client",
  "opportunity_type": "lead",
  "classification": "hot_lead",
  "source": "Facebook Group",
  "source_type": "private",
  "platform": "facebook",
  "profile_url": "https://www.facebook.com/profile.php?id=...",
  "post_url": "https://www.facebook.com/groups/.../posts/...",
  "captured_at": "2026-06-20T09:00:00-07:00",
  "safe_context_summary": "Person asked what to do after receiving an insurance non-renewal notice.",
  "evidence_snippet": "Short visible snippet when safe",
  "why_it_matters": "This is a direct need signal tied to the client's offer.",
  "related_offer": "Home insurance review",
  "related_pain_point": "Confusion after non-renewal notice",
  "confidence": "high",
  "suggested_action": "Human reviews the post and decides whether to leave the suggested value-first comment.",
  "suggested_comment": "Short natural comment in the same language as the post",
  "comment_language": "en",
  "comment_style_notes": "natural, short, no direct pitch",
  "status": "needs_review"
}
```

Allowed `opportunity_type`:

- `lead`
- `competitor`
- `both`

Allowed lead classifications:

- `hot_lead`
- `warm_lead`
- `watch_lead`
- `direct_need`
- `indirect_need`
- `pain_signal`
- `buying_trigger`
- `objection`
- `comparison`
- `complaint`
- `adjacent_need`

Allowed competitor classifications:

- `direct_competitor`
- `indirect_competitor`
- `adjacent_solution`
- `attention_competitor`
- `authority_or_kol_competing_for_trust`
- `market_hypothesis`

Allowed status:

- `needs_review`
- `copied_by_human`
- `approved_for_comment`
- `commented_by_human`
- `not_relevant`
- `monitoring`
- `archived`

Privacy rule:

- Do not store unnecessary personal data.
- Prefer safe summaries, source URLs, and short evidence snippets.
- Do not store scraped contact details, DMs, hidden account data, or raw private personal data.

### `history/YYYY-MM/new_private_sources_log.md`

Purpose:

- Track new private data source candidates discovered while scanning private platforms.
- Preserve Facebook-recommended groups, pages, communities, profiles, or similar source suggestions.
- Let the human review new sources before they become part of the active daily private data source queue.

Format:

```md
# New Private Data Sources Log

| Date | Platform | Source Type | Source Name | Profile/Group URL | Current Recommendation URL | Detected While Scanning | Why Relevant | Related Content Pillar | Estimated Priority | Suggested Cadence | Status | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-06-20 | Facebook | group | Los Angeles DUI Support Questions | https://www.facebook.com/groups/... | https://www.facebook.com/groups/... | Competitor group scan | Repeated questions about DUI court and license issues | Local process education | medium | weekly | needs_human_review | Do not join automatically |
```

Allowed status:

- `needs_human_review`
- `added`
- `skipped`
- `not_relevant`
- `blocked`

---

### 23.1 Client Strategy And Positioning

For each client, the agent should maintain strategy files:

- `strategy/offer_map.md`
- `strategy/brand_voice.md`
- `strategy/content_pillars.md`
- `strategy/funnel_map.md`

The agent must infer and maintain:

- Core offer.
- Secondary offers.
- Ideal customer segments.
- Lead magnets or conversion actions.
- Trust signals.
- Differentiators.
- Proof points.
- Objections.
- Compliance boundaries.
- Brand voice.
- Content pillars.
- Funnel stage mapping.

Example funnel mapping:

| Funnel Stage | Goal | Example Content |
|---|---|---|
| Awareness | Make the audience recognize the problem | "Why buyers are confused by rising inventory" |
| Education | Explain options and consequences | "How property taxes change your real payment" |
| Trust | Show expertise and perspective | "Why preparation beats prediction in this market" |
| Lead-Gen | Prompt action | "Get pre-approved before you start touring" |

### 23.2 Content Calendar And Cadence

The agent should maintain:

- `calendar/content_calendar.md`

The calendar should include:

- Planned publish date.
- Platform.
- Client.
- Content pillar.
- Funnel stage.
- Topic.
- Script/output file.
- Approval status.
- Publishing status.
- Reference URLs.

The agent should use daily ideas to populate the calendar, but must avoid overfilling it without approval. The daily best idea becomes a candidate for the calendar, not automatically a published post.

Example calendar row:

```md
| Date | Platform | Pillar | Funnel Stage | Topic | Status | Output |
|---|---|---|---|---|---|---|
| 2026-06-20 | Reels / Shorts | Market timing | Education | Austin inventory is rising again | drafted | outputs/2026-06-20.md |
```

### 23.8 Analytics And Reporting

The agent should maintain:

- `analytics/metrics_log.md`
- `reports/YYYY-MM_report.md`

Track metrics when available:

- Views.
- Watch time.
- Retention.
- Likes.
- Comments.
- Shares.
- Saves.
- Clicks.
- Leads.
- Calls booked.
- Cost or credits spent.
- Published URL.
- Content pillar.
- Funnel stage.

### WideCast OpenAPI Analytics Collection Rule

When running weekly learning, monthly reporting, or any performance review, the agent must use available verified provider analytics capabilities before drawing conclusions. For WideCast, load the current client's provider config, discover or refresh `https://widecast.ai/openapi.yaml`, verify the account with `getAccount`, then call available analytics/library operations such as `getAnalytics`, `listVideos`, `getStatus`, and `getVideoData`.

The agent should inspect the available WideCast OpenAPI operation list at runtime and call the relevant operations for:

- Recently published content.
- Published post/video URLs.
- Title.
- Description.
- Caption.
- Hashtags.
- Platform.
- Publish date.
- Topic or video ID.
- General account analytics.
- View counts.
- Follower counts.
- Engagement trends.

If WideCast OpenAPI exposes a list of published posts, recent videos, production history, publishing history, analytics dashboard, or platform statistics, the agent must use those sources first after verifying that the API key belongs to the current client.

For each published content item from the last 7 days, the agent should measure it daily for up to 7 days after publishing:

1. Retrieve the published URL and metadata through WideCast OpenAPI when available.
2. Save URL, title, description, caption, hashtags, platform, publish date, and related script/output file.
3. Use the Solo Agency Local Collector extension plus Local Collector app to capture visible metrics from each published URL when tools, permissions, and login state allow it.
4. Measure or extract available engagement metrics, such as:
   - views
   - likes
   - comments
   - shares
   - saves
   - reposts
   - reactions
   - follower/subscriber count where relevant
   - audience questions
   - objections
   - requests for help
   - lead signals in comments
5. If direct platform metrics are not accessible, record the limitation and use whatever WideCast OpenAPI analytics or visible public metrics are available.
6. Store all results in `analytics/metrics_log.md`.
7. Store audience questions, objections, and useful comment signals in `analytics/comment_signal_log.md`.
8. Store strategic learnings in `analytics/learning_log.md`.
9. Use the results to update reports, content pillar scoring, hook learnings, CTA learnings, source priority, lead-gen angles, and future idea selection.

### Published URL Measurement Via Local Collector

The Local Collector is not only for private data source idea discovery. It should also be reused for published URL measurement when possible.

Reason:

- Some platform metrics are visible only inside the logged-in browser session.
- Some AI agents cannot reliably browse platform pages directly.
- The Solo Agency Local Collector extension can capture visible page text, current URL, engagement hints, and source metadata in the same browser/profile where the human is logged in.

When measuring published URLs:

1. Build a temporary run-now collector job whose sources are the published URLs retrieved from the configured provider, such as WideCast OpenAPI.
2. Mark these sources clearly, for example:
   - `source_type: published_content_url`
   - `purpose: performance_measurement`
   - `platform: youtube | tiktok | instagram | facebook | x | linkedin | threads | pinterest | reddit | google_business_profile | other`
3. Use conservative pacing and do not hammer platform pages.
4. Capture visible text, current URL, page title, engagement hints, any visible metric labels/counts, and comments/questions when visible.
5. Store raw collector output under the normal collector `inbox/YYYY-MM/{run_id}/` folder.
6. Parse the captured visible text into normalized metrics when possible.
7. Store normalized metrics in `analytics/metrics_log.md`.
8. Store useful comment/question/objection/lead signals in `analytics/comment_signal_log.md`.
9. Store strategic learnings in `analytics/learning_log.md`.
10. If a metric is hidden, unavailable, or not visible in the logged-in session, write `unavailable` and explain why.

The agent must not scrape hidden APIs, extract cookies, bypass login, or defeat platform restrictions to measure metrics. Use only authorized visible data or verified provider analytics.

The agent must also call WideCast OpenAPI analytics or dashboard operations that provide overall account-level statistics, such as total views, follower growth, platform performance, or other aggregate metrics. These aggregate metrics should be stored and used for learning even when per-post data is incomplete.

Do not invent metrics. If a platform hides likes, shares, comments, views, or follower data from the current agent/session, mark the metric as `unavailable` and explain why.

Suggested `analytics/metrics_log.md` format:

```md
| Date Checked | Published Date | Client | Platform | URL | Title | Description | Hashtags | Content Pillar | Funnel Stage | Views | Likes | Comments | Shares | Saves | Followers/Subscribers | Source Of Metric | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-06-20 | 2026-06-18 | Smith Law | TikTok | https://... | What to do after a DUI stop | Short DUI education video | #dui #california | Emergency first steps | Education | 1200 | 44 | 8 | 3 | unavailable | unavailable | WideCast OpenAPI + public URL check | Comments show license-suspension anxiety |
```

Suggested `analytics/comment_signal_log.md` format:

```md
| Date Checked | Client | Platform | URL | Comment/Question Summary | Signal Type | Pain Point | Lead Potential | Suggested Follow-Up Content | Notes |
|---|---|---|---|---|---|---|---|---|---|
| 2026-06-20 | Smith Law | TikTok | https://... | Viewers asked what happens to a driver's license after a DUI arrest | question | license suspension fear | warm | Explain the DMV deadline after a DUI arrest | Use as future QA script |
```

Suggested `analytics/learning_log.md` format:

```md
| Date | Client | Evidence | Learning | Affected Pillar | Hook/CTA Impact | Future Action |
|---|---|---|---|---|---|---|
| 2026-06-20 | Smith Law | DUI deadline video got high comment rate | License-suspension anxiety drives comments | Emergency first steps | Use deadline hooks more often | Prioritize DMV-deadline Q&A ideas next week |
```

The agent should generate weekly or monthly reports when asked or scheduled:

- What worked.
- What did not work.
- Best content pillars.
- Best hooks.
- Best platforms.
- Recommended next experiments.
- Content ideas to repeat or retire.

### 23.9 Experiment Backlog

The agent should maintain:

- `experiments/experiment_backlog.md`

Examples:

- Test fear-based hook vs curiosity hook.
- Test local news angle vs evergreen education.
- Test direct CTA vs soft CTA.
- Test face-on-camera vs faceless B-roll.
- Test short 25-second version vs 60-second version.
- Test competitor-response angle.

Each experiment should include:

- Hypothesis.
- Client.
- Content pillar.
- Platform.
- Success metric.
- Result.
- Next decision.
