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
      source_registry.json
      search_pool.json
      public_pool/
        {uid_hash}/
          YYYY-MM-DD.md
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

Each `extensions/{client_slug}_extension/client_binding.json` must include:

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
    provider_defaults.json             # default OpenAPI provider catalog, no secrets
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
        update_YYYY-MM-DD_HHMMSS/
      issues/
        YYYY-MM-DD_{blocker_slug}.md
    notifications/
      notification_log.md
    collector/
      collector_setup_status.md
      collector_config.json
      extension_registry.json
      jobs/
        pending/
        claimed/
        completed/
        failed/
      inbox/
        YYYY-MM/
          {client_slug}/
            {run_id}/
              collector_status.json
              private_data_points.jsonl
              leads.jsonl
              competitors.jsonl
              new_private_sources.jsonl
              source_status.jsonl
              snapshots/
      source_registry.json
      search_pool.json
      public_pool/
        {uid_hash}/
          YYYY-MM-DD.md
      logs/
        bridge_events.jsonl
        extension_health.jsonl
        job_routing.jsonl
        agent_handoff.jsonl
    browser_profiles/
      {source_slug}/

Shared-scan files (cross-client, maintained through `tools/solo_tool source-registry` and `tool search-pool` — never hand-edited, always through the tool so concurrent runs cannot corrupt them):

- `collector/source_registry.json` — one entry per canonical source UID across ALL clients: `uid`, `uid_hash`, `sample_url`, `domain`, `platform`, `source_type`, `kind` (private|public — the scan lane; `discovered` rows are comment threads) plus `source_type` (group|page|profile|site), `scope` (`shared` | `exclusive` | `unclassified` — auto-created entries stay `unclassified` and are never served as reuse until an explicit `register`), `subscribers[]` (client_slug, priority, scan_cadence, registered_at), `last_scan` (completed_at, run_id, client_slug, data_dir, status, kind — reuse requires the lane to match), `last_failed`, `scan_claim` (client_slug, claimed_at — an in-progress marker so concurrent runs `wait` instead of duplicating a scan; expires after 2h, released by `record`), and a top-level `freshness_ttl_hours` (default 20). The freshness check is a rolling TTL against `last_scan.completed_at`, never a calendar-day compare. `kind: discovered` entries are a different shape (below, "Discovered sources") — they are never `due`/reused across clients like private/public sources, since a comment thread found for one client is that client's own find.
- Monitored Facebook groups (`source_type: group`, `kind: private` — read through the human's session) carry additional per-client fields, written only by `tool source-registry add|record|pause|resume` — never hand-edited: `state` (`active | paused | not_selected | no_access` — `active` is scanned by the daily plan, `paused` is held out by the Boss on the Sources page, `not_selected` is a Group Potential Rule `low` verdict, `no_access` is a private group the account cannot read), `potential` (`high | medium | low`, the Group Potential Rule's verdict), `potential_reason` (its one-line reason), `scans` (total times this group has been scanned), `leads_total` (lifetime hot+warm+watch leads captured from this group), `leads_recent` (the lead counts from its last 3 scans, newest first — what the plan ranks on), `last_scanned_at` (ISO-8601 timestamp of its most recent scan), `paused_at` (ISO-8601 timestamp of its last pause, blank while active), and `origin` (`discovered | custom | default` — how the source first entered the registry).
- `collector/search_pool.json` — shared public keyword-search results keyed by (industry, normalized keyword): `searched_at`, `client_slug`, `results[]` of client-neutral `{url, title, note}`. Entries older than 7 days are pruned on write.
- `collector/public_pool/{uid_hash}/YYYY-MM-DD.md` — client-neutral raw findings from visiting a shared PUBLIC source (facts, URLs, quotes, dates only — no client analysis, no client names), written by the run that visited it and registered via `source-registry record --data-dir`; other subscriber runs consume it through the registry pointer and do their own client-specific filtering.
- Collector data points/leads/competitors carry bridge-stamped `source_uid` + `point_uid` — key-based dedup for shared-scan consumption.

Scheduling files (maintained through `tool schedule-slots` and the operator web UI, same no-hand-edit rule):

- `system_settings.json` (data-root level) — the operator's GLOBAL system config, editable in the web UI at `/ui/settings` without any agent chat: `operator_email` (where agents send operator notifications when an operator push channel is used — client notifications stay on each client's own channel), `max_concurrent_tasks` (default 10), `slot_step_minutes` (15), `slot_horizon_days` (35), `default_task_duration_min` (30). Agents READ these settings (`tool system-settings get`); they never overwrite the operator's values.
- `automation/task_slots.json` — machine-readable registry of every automation task's run time: `task_name`, `client_slug`, `cadence_hours` or `monthly`, `run_time` "HH:MM" local, `anchor_date`, `duration_min`, `status` (active|paused). For a task that runs the Social Discovery Pass, `duration_min` is the ETA Rule's `run_eta_high_min` for that task, never the `default_task_duration_min` fallback above — re-run `tool schedule-slots suggest` whenever a recomputed ETA exceeds the recorded `duration_min`. `tool schedule-slots suggest` projects all of it onto a timeline to pick collision-free start times; `schedule.md` and the automation manifest remain the human-readable views.
- `system_settings.json` also carries `accountability_max_gap_hours` (default 72) — the posting-gap reminder threshold (`playbooks/ACCOUNTABILITY_POSTING.md`), editable at `/ui/settings`.
- `automation/operator_mail_log.jsonl` — audit line per operator escalation email sent by `tool gmail send-operator` (ts, to, from, sendbox, subject).

Operator dashboard feed (written by runs, only ever READ by the dashboard):

- `fleet/{client_slug}.json` — one snapshot per client, rewritten at the END of every run (04 step 33c). Schema (unavailable values null/absent, never invented): `schema_version`, `client_slug`, `client_name`, `updated_at`, `run_id`, `posting` {`last_posted_at`, `gap_hours`, `source` (provider|ledger|collector), `status` (ok|warning|breach)}, `accountability` {`reminders_sent_this_episode`, `last_reminder_at`, `operator_escalated_at`}, `totals` {`posts_7d`, `posts_30d`}, `engagement` {`views_7d`, `likes_7d`, `comments_7d`, `shares_7d`, `followers`, `top_post` {`url`, `title`, `views`}}, `production` {`ideas_queued`, `drafts_pending_approval`, `videos_produced_7d`}, `leads` {`hot`, `warm`, `watch`}, `report` {`last_report_at`, `html_path`, `pdf_status`}, `blockers` [operator-facing strings]. Per-client `analytics/accountability_log.md` holds the reminder-episode state the snapshot summarizes.
    test_logs/
      YYYY-MM/
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
            learning_log.md
            comment_signal_log.md
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
              {client-name}-client-report.html
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
          learning_log.md
          comment_signal_log.md
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
- Keep `outputs/latest/{client-name}-client-report.html` as the default client-ready report pointer/copy, `outputs/latest/{client-name}-client-report.pdf` when PDF export is available and safe, `outputs/latest/{client-name}-INTERNAL_REPORT.html` for the operator, and daily/public/private latest files only as staging/diagnostic convenience copies. Also keep `latest_master_digest.md` and `latest_master_digest.html` for operator/master reporting.
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

### `content/` — the client's content library

Everything this client has produced or been given, kept locally so it can be found again: the
videos, blogs, social posts, articles and files the outgoing messages draw on. Per client, because
a client's content is that client's asset; sharing between clients is an explicit clone that
records where the copy came from.

```
clients/{slug}/{business}_{location}/content/
  items/ct_<id>.json      one item per file — the record
  index.json              derived search index (safe to delete; `tool content reindex` rebuilds it)
```

One item: `id`, `kind` (video | blog | social_post | article | image | report | note), `source`
(widecast | local | user | imported), `title`, `summary`, `body` (the written text), `script` (a
video's narration), `keywords[]`, `language`, `status` (draft | ready | published | archived),
`urls` (`public`, `player`, `review`, and per-platform published links), `external`
(`topic_id`, `request_id`, `provider`), `origin`, `cloned_from`, `fingerprint`, and the timestamps.

**The one write path is `tool content`** (`import | search | get | list | clone | reindex | stats |
forget`), mirroring the `tool crm-store` rule — never hand-edit these files. The bridge also serves
`GET /content/search`, `GET /content/item`, `GET /content/libraries` and `POST /content/import`
locally, and the operator has a page at `/ui/{client}/content`.

**Nothing is recorded twice.** An item's identity is its WideCast `topic_id`, else its url, else a
hash of its kind + title + opening text, so re-importing the same piece updates one record instead
of growing the library. An update never blanks a field it did not mention: the WideCast capture
knows the ids and urls, the agent knows the text, and they arrive at different times.

**Why it is not optional.** WideCast's content search was withdrawn from the SDK and only ever
covered video topics; `create_content` hands back a viewer url and never the written text again. A
blog whose local record is missing is gone. For written content this library is the only copy.

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
- slot_check_pending: true | false   # true only while a task was created before the bridge existed and still needs `schedule-slots register` (Stage 4, first-task exception); clearing it is the first action after the bridge is installed
- scheduler_type: native_ai_automation | native_ai_scheduled_task | cron | launchd | task_scheduler | n8n | make | zapier | github_actions | server_job | manual
- scheduler_name:
- scheduler_location_or_url:
- timezone:
- schedule_file: daily-content-pipeline/schedule.md
- scheduled_prompt_file: daily-content-pipeline/automation/scheduled_run_prompt.md
- scheduled_entrypoint: playbooks/SCHEDULED_RUN_ENTRYPOINT.md
- root_playbook: SOLO_AGENCY_PLAYBOOK.md
- clients_index: daily-content-pipeline/clients_index.md
- collector_config: daily-content-pipeline/collector/collector_config.json — each watched-source entry may carry `origin: default | custom` (`custom` = a URL the Boss gave at setup step 5 or later; `default` or absent = added by the run, discovery or the industry defaults); the dashboard's Sources page splits its Default / Custom tabs on this field, and Discovered sources live in the source registry (`kind: discovered`), never here A source item may also carry `enabled: false` — paused from the Sources page (Custom tab, `POST /api/ui/{client}/sources/toggle`): skipped by the collector's scheduled windows and by every run-now job the run builds; `enabled` absent or `true` = active.
- provider_defaults: daily-content-pipeline/provider_defaults.json
- notification_channel:
- pdna_status:
- provider_status_summary:
- provider_capability_cache_status:
- private_data_source_status_summary:
- report_merge_contract: single_client_report_with_lane_staging | one_report_two_lanes | legacy_mixed_report | unknown  # single_client_report_with_lane_staging is the current default
- report_notification_policy: same_report_public_private_notifications_allowed | single_final_notification | unknown
- latest_user_change_summary:
- actual_native_task_prompt_updated: true | false | not_applicable | unknown
- automation_prompt_update_pending_reason:
- automation_freshness_status: current | resync_in_progress | action_needed | not_applicable
- automation_freshness_summary: whether latest changes are synced into automation/scheduled task prompt/contract/playbook/source state, not only config, and whether tomorrow's run will load the newest state
- first_run_consent: yes | not_now — the Boss's answer to step 7's one first-run question (`playbooks/SETUP_FLOW_ENTRYPOINT.md`); `not_now` means the daily task keeps its existing schedule and nothing else is asked
- first_run_task_id: the one-time scheduled task id used to dispatch this client's first run (e.g. `{client_slug}-solo-agency-first-run`), when the runtime created one
- first_run_dispatched_at: ISO 8601 timestamp of when the first-run task was dispatched
- first_run_wait: armed | not_available | timed_out | reported — state of the background wait for that first run (`playbooks/04_DAILY_SCHEDULE.md`, "Wait and report")
- first_run_reported_at: ISO 8601 timestamp of when the First-Run Report was spoken back to the Boss in chat
- run_calls_planned: `calls_planned` computed by the ETA Rule (`playbooks/04_DAILY_SCHEDULE.md`) at dispatch for the run in flight, value copied from `tool run-progress eta` output, recorded so a later turn can read it instead of recomputing from memory
- run_eta_low_min: the ETA Rule's `eta_low` for that run, in minutes, value copied from `tool run-progress eta` output
- run_eta_high_min: the ETA Rule's `eta_high` for that run, in minutes, value copied from `tool run-progress eta` output
- run_eta_at: ISO 8601 timestamp of `eta_high_at` — the clock time the run is expected to finish by, value copied from `tool run-progress eta` output
- first_run_last_stage: the last Run Progress Rule stage index (0–6, `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`, "automation/run_progress.jsonl") spoken to the Boss for the first run
- unattended_permissions: granted | declined | not_applicable — outcome of the Stage 4 unattended-permissions consent (Claude Code desktop local runtime only)
- unattended_permissions_scope: user_settings — where the allow rules were written (`~/.claude/settings.json`)
- unattended_permissions_written_at: ISO 8601 timestamp of when the allow rules were written, when `unattended_permissions: granted`

## Active Clients

| Client | Client Slug | Profile Path | Status | Private Data Source Status | PDNA Status | Notes |
|---|---|---|---|---|---|---|

## Team Roster and Team Files

The manifest also lists every scheduled task as a team member (`playbooks/TEAM_MODEL.md`). Task names stay exactly as created — `run_lock`, Automation Resync and the manifest match on them; the role is a label, not part of the name.

| Task | Client | Role(s) | Brain | Owner brain (runtime) | Status |
|---|---|---|---|---|---|

Two team files live next to the manifest under `daily-content-pipeline/automation/`:

- `boss_orders.md` — the Team Leader's ledger of the Boss's requests and goals (header and statuses in `playbooks/TEAM_MODEL.md`). Written by the interactive session only; scheduled runs read it. Never deleted, never closed without a reason.
- `standup.jsonl` — one JSON line per finished scheduled run (`playbooks/SCHEDULED_RUN_ENTRYPOINT.md`, step 16A): `ts`, `task`, `client_slug`, `role`, `outcome`, `reports[]`, `needs_boss[]`, `blockers[]`, `discovered_new` (count of `likely` Step 5 threads recorded this run — see "Discovered sources" above; `0` when none). Append-only; the Team Leader reads the tail at the start of every session. Keep 90 days; older lines may be pruned by the update-watch task.
- `support_requests.md` — posts the Team Leader made to the Solo Agency Facebook support group on the Boss's behalf (`playbooks/TEAM_MODEL.md`, "Support requests"): id, date, type, title, status, group_url, post_id, post_url, feedback_id, last_checked, last_comment_id, replies, notes. Written by the interactive session; scheduled runs fill the post ids after posting, update `last_checked` / `last_comment_id` / `replies`, and surface new replies.

### `automation/run_progress.jsonl`

Purpose: one JSON line appended at each Run Progress Rule stage boundary (`playbooks/04_DAILY_SCHEDULE.md`, "Progress Display Contract") while a Social Discovery Pass is in flight, so any turn — a background wake, the Boss asking mid-run, or a runtime with no background execution — can read the last line for the client and speak stage, counts and ETA without carrying a number from memory (Read-Before-Claim Rule). Written by `tool run-progress --pipeline {setup-root}/daily-content-pipeline append`, read with `tool run-progress --pipeline {setup-root}/daily-content-pipeline show`, never hand-edited; the tool stamps `ts` and `stage_index` itself and refuses a repeated `done` stage for the same client + run_id unless `--force`. Append-only, lines from every client interleaved by `ts`; `tools/wait_for_run <client_slug> <since_iso> --watch progress` matches on `client_slug` and `ts` only.

Line schema (one object per line):

```json
{"ts": "2026-09-12T08:14:03+07:00", "client_slug": "acme", "run_id": "2026-09-12-acme-1",
 "stage": "find_people", "stage_index": 2, "status": "done",
 "calls_done": 6, "calls_planned": 45,
 "counts": {"search_posts": 34, "group_posts": 0, "groups_found": 0, "groups_readable": 0,
            "groups_no_access": 0, "groups_monitored": 0, "groups_not_selected": 0, "groups_paused": 0, "people_found": 57,
            "leads_hot": 0, "leads_warm": 0, "leads_watch": 0},
 "platforms": {"facebook": "running", "instagram": "running", "x": "tripped:rate_limit"},
 "eta_low_at": "2026-09-12T08:40:00+07:00", "eta_high_at": "2026-09-12T09:00:00+07:00",
 "note": ""}
```

`status` is `done` at a boundary, `stalled` when a platform's job produced no result within its TTL, `aborted` when the run stopped early (say why in `note`). `platforms` values: `running | done | skipped:{reason} | tripped:{signal} | web_only`. The six `stage`/`stage_index` values, one line per boundary: `find_posts` (1), `find_people` (2), `find_groups` (3), `scan_in_group` (4), `filter_leads` (5), `build_report` (6) — see the Run Progress Rule.

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

Tracks every defect report made when the latest GitHub playbooks/code still do not resolve a blocker, whichever rung of the reporting ladder it went out on (`AGENTS.md`): a WideCast `reportError` call (record its `request_id`), a Solo Agency support-group post (also in `support_requests.md`), a GitHub issue, or a local draft. The name is historical — this is the defect-report tracker, not a GitHub-only file.

Create or update this file when Last-Resort Recovery reports anything. The human needs neither git nor a GitHub account. It is also the dedupe record: one report per blocker fingerprint, never the same fingerprint twice within 24h, so a failing daily run cannot mail the founder the same thing every morning.

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
  "update_watch_task_prompt_pending": false,
  "clients_resynced": [],
  "automations_resynced": [],
  "human_actions_required": []
}
```

Set `update_watch_task_prompt_pending` to `true` when the `Solo Agency - GitHub Update Watch` task prompt could not be created or updated natively and `daily-content-pipeline/automation/update_watch_prompt.md` holds the pending prompt.

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
| 2026-06-20 | Claude Schedule | daily_run_completed | public_report_ready | WideCast email+Telegram | sent | outputs/2026-06/2026-06-20/angela-do-daily-report.html | outputs/2026-06/2026-06-20/angela-do-client-report.pdf | generated | widecast | yes | uploadAsset | sendNotification | yes | https://... |  | yes | https://... | none | Await private report or review approvals |
| 2026-06-20 | Claude Schedule | daily_run_completed | private_report_ready | WideCast email+Telegram | sent | outputs/2026-06/2026-06-20/angela-do-daily-report.html | outputs/2026-06/2026-06-20/angela-do-client-report.pdf | generated | widecast | yes | uploadAsset | sendNotification | yes | https://... |  | yes | https://... | none | Review daily report index and lane reports |
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
        "video_editing": "not_configured",
        "render_export": "not_configured",
        "distribution": "not_configured",
        "notification": "not_configured",
        "analytics": "not_configured"
      },
      "video_editing": {
        "enabled": false,
        "preferred_skill_operation_id": "getEditingSkill",
        "preferred_video_data_operation_id": "getVideoData",
        "preferred_scene_geometry_operation_id": "sceneGeometry",
        "preferred_scene_inspector_operation_id": "sceneInspector",
        "preferred_modify_scene_operation_id": "modifyScene"
      },
      "render_export": {
        "enabled": false,
        "preferred_operation_id": "exportVideo",
        "requires_fresh_human_approval": true
      },
      "notification": {
        "enabled": false,
        "preferred_operation_id": "sendNotification",
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

When local Python execution is available, agents may create or refresh this file with `tool provider discover --config <client provider_config.local.json> --defaults daily-content-pipeline/provider_defaults.json --out-dir <client integrations/providers folder>`.

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
    "createVideo": {"method": "POST", "path": "/..."},
    "getAccount": {"method": "GET", "path": "/..."}
  },
  "operation_aliases": {
    "account": "getAccount",
    "analytics": "getAnalytics",
    "upload_asset": "uploadAsset",
    "upload_html_report": "uploadAsset",
    "send_notification": "sendNotification",
    "publish": "publish",
    "create_video": "createVideo",
    "export_video": "exportVideo",
    "get_status": "getStatus",
    "get_video_data": "getVideoData",
    "get_writing_skill": "getWritingSkill",
    "get_editing_skill": "getEditingSkill",
    "scene_geometry": "sceneGeometry",
    "scene_inspector": "sceneInspector",
    "modify_scene": "modifyScene",
    "create_content": "createContent",
    "create_image": "createImage",
    "search_broll": "searchBroll",
    "collect_ideas": "collectIdeas",
    "list_videos": "listVideos"
  },
  "capability_status": {
    "production": "available | partial | unavailable",
    "video_editing": "available | partial | unavailable",
    "render_export": "available | partial | unavailable",
    "media": "available | partial | unavailable",
    "distribution": "available | partial | unavailable",
    "notification": "available | partial | unavailable",
    "analytics": "available | partial | unavailable"
  },
  "missing_capability_aliases": {},
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

### `collector/extension_registry.json`

Maps each client to its per-client Chrome extension instance so scheduled runs can route collector jobs to the correct extension. Scheduled runs must read the client→`extension_instance_id` mapping from this file.

Create or update this file when a per-client extension folder is prepared under `extensions/{client_slug}_extension/`, when an extension is loaded or reloaded in Chrome, or when a client's collector status changes.

Minimum fields:

```json
{
  "schema_version": 1,
  "clients": [
    {
      "client_slug": "avenngo",
      "client_name": "AvenNgo",
      "extension_instance_id": "ext_avenngo_default",
      "extension_display_name": "AvenNgo - Solo Agency Collector",
      "extension_folder": "/ABSOLUTE/PATH/extensions/avenngo_extension/",
      "browser": "chrome",
      "profile_directory": "Default",
      "registered_at": "2026-06-20T09:00:00Z",
      "last_health_at": "2026-06-20T09:05:00Z",
      "status": "active"
    }
  ]
}
```

Field notes:

- `client_slug`: client slug used across the pipeline.
- `client_name`: human-readable client name.
- `extension_instance_id`: stable id for this client's extension instance; scheduled runs read the client→`extension_instance_id` mapping from this file.
- `extension_display_name`: Chrome display name, client name first.
- `extension_folder`: per-client pin for the client's unpacked extension folder, read (not written) by the bridge — `uiResolveExtensionFolder` in `solo-agency-collector/bridge-go/ui.go` checks this pin first, before falling back to the current `extensions/{client_slug}_extension/` convention and then the legacy `extensions/{client_slug}/` path. Set this only when the folder lives somewhere other than the current convention; leave it unset otherwise.
- `chrome_profile_hint`: retired 2026-09-11 — no longer written, never read; ignore if present.
- `browser`: the Chromium-based browser this client's extension actually runs in — one of `chrome | edge | brave | vivaldi | opera | chromium`. Recorded only by the install page's advanced mode (`?advanced=1`); setup never asks for it and never writes it from a chat answer. Safari and Firefox are never valid values here; a machine with only those installed is told to install Chrome instead (`playbooks/SETUP_FLOW_ENTRYPOINT.md`, "Kết nối Facebook, Instagram and X (step 4)").
- `profile_directory`: the exact profile folder name from that browser's own `Local State` (`profile.info_cache` key, e.g. `Default`, `Profile 1`). Recorded only by the install page's advanced mode (`?advanced=1`) from its own account list when 2+ profiles exist; setup never asks for it and never writes it from a chat answer.
- `registered_at`: when the extension was registered.
- `last_health_at`: last successful health check timestamp.
- `status`: one of `active | pending_install | disabled`.

Do not store secrets, cookies, tokens, or provider API keys in this file.

### `outputs/YYYY-MM/YYYY-MM-DD/{client-name}-report_state.json`

Tracks the state for the canonical combined client report and the three staging files used to build it. It prevents a later public or private data source pass from overwriting or summarizing away the other lane's full HTML report, and it ensures the combined HTML/PDF are rebuilt after lane updates.

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
  "client_report_html_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-client-report.html",
  "latest_client_html_path": "outputs/latest/{client-name}-client-report.html",
  "latest_daily_html_path": "outputs/latest/{client-name}-daily-report.html",
  "latest_public_html_path": "outputs/latest/{client-name}-public-data-sources-report.html",
  "latest_private_html_path": "outputs/latest/{client-name}-private-data-sources-report.html",
  "internal_report_md_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-INTERNAL_REPORT.md",
  "internal_report_html_path": "outputs/YYYY-MM/YYYY-MM-DD/{client-name}-INTERNAL_REPORT.html",
  "latest_internal_report_html_path": "outputs/latest/{client-name}-INTERNAL_REPORT.html",
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
  "facebook_discovery": {
    "terms_used": 0,
    "feed_posts_found": 0,
    "group_posts_found": 0,
    "people_found": 0,
    "people_captured": 0,
    "groups_found": 0,
    "groups_readable": 0,
    "groups_no_access": 0,
    "groups_scanned": 0,
    "groups_monitored": 0,
    "groups_not_selected": 0,
    "groups_paused": 0,
    "leads_found": 0,
    "leads_locked": 0,
    "budget_used": 0,
    "budget_available": 0,
    "trip_status": "clean"
  },
  "instagram_discovery": {
    "terms_used": 0,
    "posts_found": 0,
    "depth_posts_found": 0,
    "people_found": 0,
    "people_captured": 0,
    "depth_calls": 0,
    "leads_found": 0,
    "leads_locked": 0,
    "budget_used": 0,
    "budget_available": 0,
    "trip_status": "clean"
  },
  "x_discovery": {
    "terms_used": 0,
    "posts_found": 0,
    "depth_posts_found": 0,
    "people_found": 0,
    "people_captured": 0,
    "depth_calls": 0,
    "leads_found": 0,
    "leads_locked": 0,
    "budget_used": 0,
    "budget_available": 0,
    "trip_status": "clean"
  },
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

`facebook_discovery`/`instagram_discovery`/`x_discovery` are the three per-platform legs of the
Social Discovery Pass (`playbooks/10_LEAD_COMPETITOR_DETECTION.md`, "Social Discovery Pass"), one
object per platform, each reconciled independently, and their fields follow the Five result types
(the Five result types, `playbooks/10_LEAD_COMPETITOR_DETECTION.md`): a search-post count, an in-group/depth-post count, a groups count split by access, a group-
monitoring count, and a leads count — never merged into one another. `{platform}_discovery.trip_status`:
`clean`, or the exact safety trip that stopped that platform's account for the day — the trip is per
platform (round-robin rule), so one platform tripping never zeroes another's object. `budget_used`/
`budget_available` are calls, not leads — Instagram and X: FIRST RUN ceiling 12, DAILY ceiling 4;
Facebook's ceiling varies with how many groups the registry plan returns: 9 discovery calls + up to
20 monitored groups × 3 terms FIRST RUN, or 3 discovery calls + up to 20 monitored groups × 2 terms
DAILY. Facebook's `groups_found` splits into `groups_readable` (public, or private where the account
is already a member) and `groups_no_access` (private, account not a member — listed for the Boss to
join, never scanned); `groups_monitored` counts groups registered `state: active` by the Group
Potential Rule (agent-selected, no human decision), `groups_not_selected` counts `state:
not_selected`, and `groups_paused` counts `state: paused` (held out by the Boss on the Sources page).
Instagram and X have no groups, so their objects carry `depth_calls` (profile-depth + comments/replies
calls) instead of Facebook's `groups_found`/`groups_readable`/`groups_no_access`/`groups_scanned`/
`groups_monitored`/`groups_not_selected`/`groups_paused`, `posts_found` instead of
`feed_posts_found`, and `depth_posts_found` (profile depth plus comments/replies) instead of
Facebook's `group_posts_found`. `leads_locked` comes from `tool crm-store ... contact lock-status`,
never a hand count.

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

- Public data source pass may write only `{client-name}-public-data-sources-report.html`, public source records, `{client-name}-daily-report.html` status metadata, and the rebuilt combined `{client-name}-client-report.html`/PDF package. It must preserve any existing private report.
- Private data source pass may write only `{client-name}-private-data-sources-report.html`, private source records, `{client-name}-daily-report.html` status metadata, and the rebuilt combined `{client-name}-client-report.html`/PDF package. It must preserve any existing public report.
- After a public or private data source pass reaches a terminal state, the state file counts must be reconciled with the lane report, daily index, internal source record, notification log, and latest copies. Do not leave stale `partial`, `pending`, `scan in progress`, or old recommended-source totals in one artifact when another artifact reports completion.
- `latest/{client-name}-client-report.html` must point to or copy the combined client-facing HTML report and is the default human/client report link.
- `latest/{client-name}-daily-report.html` may point to or copy the daily staging index, not a lane-specific artifact, but it must not be the primary report handoff link.
- `latest/{client-name}-INTERNAL_REPORT.html` must point to or copy the operator-only internal report and must be clearly labeled `INTERNAL_REPORT - Not for client sharing`.
- Client-facing files and the client PDF must pass the client-blind scrub gate before handoff. If not, keep `client_facing_scrub_status: failed` or `blocked`, record the blocker, and do not present the file as client-ready.
- `latest/{client-name}-client-report.pdf` is required when PDF export is available and safe, and must point to or copy a PDF generated from `{client-name}-client-report.html`, which itself is assembled from the three staging HTML reports. If PDF export is unavailable or unsafe, keep `client_pdf_status: blocked` and record `client_pdf_blocker`.
- If a client-share PDF includes private data source findings, record `client_pdf_redaction_status` as `redacted`, `approved_exact_sources`, or `needs_human_review`.
- If two notifications are sent, both should reference the same combined client report path or uploaded URL, with lane status recorded in `notification_log.md`. Lane-specific staging links should be omitted unless requested for diagnostics.

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

## buyer_profile

Written at Setup step 2 (profile inference) and confirmed by the Boss in one sentence. Feeds the Lead
Qualification Rule (`playbooks/LEAD_QUALIFICATION_RULE.md`) as this client's own buyer profile — every
classification tests fit against `types` here, never another client's.

Format rule: never describe buyers by an activity the product enables ("people who make videos",
"people who buy insurance") — a small model then demands proof of that activity before it will count
someone as a match. Describe who they are instead (trade, role, business, life situation). Each
`types` line names the CATEGORY first, then gives examples after "e.g." — the category is the test;
the examples are illustrative, never a closed list.

status:
rationale:
sells:
sells_to:
types:
-
why_they_need:
location:
competitors:
not_buyers:

Example (one of five reference profiles in `/Users/binhnguyen/Downloads/soloagency_leadtest_2026-09-11/clients_v3.json` — an EXAMPLE, never a list to copy):

```yaml
sells: an app that turns raw phone footage into edited short videos automatically (cuts, captions, music), sold as a monthly subscription
sells_to: people who sell something and could grow by posting more video, whether or not they make any video today. Membership is by who they are, never by whether they already film.
types:
  - independent professional of ANY trade who markets themselves (e.g. real-estate agent, loan officer, insurance agent, escrow officer, lawyer, dentist, advisor, coach, consultant, photographer, videographer)
  - owner of ANY small shop or service business (e.g. salon, restaurant, gym, clinic, auto shop, food truck, tutoring center, property management)
  - founder of a small startup or SaaS
  - creator, course seller, freelancer or one-person agency of ANY kind (a marketing or lead-gen agency, a coach, a graphic designer are customers here; they need video too)
why_they_need: editing is the step that stops them from posting consistently — it takes hours or costs money per video; they lack time, skill or an editor
location: worldwide, any language
competitors: other video-editing apps or AI editing tools, freelance video editors, video production agencies that sell editing to the same people
not_buyers: large media teams with in-house editors, job-seeking editors, students, people who sell nothing
```

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

## foundation_bank

status:
rationale:
items:
- topic:
  pain_point:
  segment:
  status: available | queued | produced | re_angled | excluded
  source: setup_seed | audience_question | operator
  angle_history:
  last_used_date:

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

bank: daily-content-pipeline/collector/public_keywords.json
  # Owned by the bridge. Read and written ONLY through `tool public-keywords`; never edit by hand.
  # Per-term fields as stored: term (dateless), lang, group, origin (setup|mined|operator|migrated),
  # status (active|probation|retired — derived from `recent` by the bridge), words, runs, useful, weak,
  # urls, ideas, recent[] (the last 5 verdicts), last_run_at, last_useful_at, added_at, note.
seed_file: public_keywords_seed.jsonl
  # Written at Setup step 5, through the bridge (`tool public-keywords add --file`); this staging file is only the fallback when the bridge is unreachable in-session: one {"term","group","lang","note"} per line.
  # Loaded by `tool public-keywords add --file` at the first run and then deleted.
  # Lines may carry "group":"community_discovery" (2-6 words, dateless) — these feed the Facebook
  # Discovery Pass's `plan --kind discovery` and are loaded into the same bank as the web terms.
migrated_at:
  # Set once `tool public-keywords migrate --profile` has imported an older profile's items list.
source_keywords_seed_file: collector/source_keywords_seed/{client_slug}.jsonl
  # Written at Setup step 5, one {"term","kind","lang","note"} per line, kind intent|role|product|
  # stage|place, terms <= 3 words. Merged automatically by `tool source-keywords ... seed` the first
  # time each group's intent-term bank is created. NOT deleted after load (unlike the file above).

## private_monitoring_activation

status: not_provided | pending_private_activation | activation_declined_for_now | activation_requested | setup_files_prepared_waiting_for_human_command | setup_command_given_waiting_for_human_run | setup_command_ran_waiting_for_extension | installed_and_running | wrong_workspace_bridge | blocked
first_trial_policy: public_first_small_win
last_prompted_date:
human_decision:
facebook_lead_source: enabled | web_only | pending
  # Set by the Facebook Login Reminder (Setup Flow, or SCHEDULED_RUN_ENTRYPOINT.md step 12D / 04
  # step 7 when an existing client's extension health goes stale/logged-out). `enabled` means the
  # extension checked in and Facebook login is confirmed. `pending` is the in-between state, never
  # itself a decision -- it stays valid across runs until the human resolves it to `enabled` or
  # `web_only` (see SCHEDULED_RUN_ENTRYPOINT.md step 12D, 04_DAILY_SCHEDULE.md step 7/11C). Never
  # assume `pending` only applies on the client's literal first automation run; also, `pending`
  # never blocks a scheduled run -- only the client's very first dispatch is gated on `enabled` or
  # `web_only`, never `pending` (SETUP_FLOW_ENTRYPOINT.md, "Kết nối Facebook, Instagram and X (step
  # 4)"). `web_only` is a human's explicit, acknowledged choice to run without Facebook -- never a
  # default, never the outcome of "để sau" or silence alone; it is set only after the human answers
  # a clear confirming phrase (for example "không dùng Facebook") to the acknowledgment-based
  # escape, OR is set automatically by the logged-out rule below.
facebook_lead_source_updated_at:
  # Timestamp of the last facebook_lead_source change.
facebook_web_only_reason:
  # The human's own words when they confirmed `web_only` (verbatim or lightly paraphrased), or the
  # automatic "not logged in on {date}" reason set by the logged-out rule below. Empty unless
  # facebook_lead_source is (or was) `web_only`.
facebook_discovery_first_pass_done: true | false
  # Set true immediately after this client's first-ever Social Discovery Pass completes Facebook's
  # leg (playbooks/10_LEAD_COMPETITOR_DETECTION.md, "Social Discovery Pass"). Gates the FIRST RUN
  # (9 discovery calls + up to 20 monitored groups x 3 terms) vs DAILY (3 discovery calls + up to 20
  # monitored groups x 2 terms) budget tier in playbooks/04_DAILY_SCHEDULE.md step 11C /
  # playbooks/SCHEDULED_RUN_ENTRYPOINT.md step 12E. Keyed to whether the pass has EVER run for
  # this client, not to the automation run number -- a client who starts `web_only` and later
  # connects Facebook several runs later still gets FIRST RUN on that later run.
  # Browser/profile note: the actual browser and profile chosen for this client's extension is NOT
  # duplicated here -- it lives in collector/extension_registry.json (`browser`, `profile_directory`,
  # above), keyed by client_slug, recorded only by the install page's advanced mode (`?advanced=1`) --
  # setup never asks for them and never writes them from a chat answer.
facebook_last_login_probe_at:
  # Timestamp of the most recent step-1 re-probe issued for this platform while it sat in
  # `web_only` with a `{platform}_web_only_reason` starting "not logged in"
  # (playbooks/10_LEAD_COMPETITOR_DETECTION.md, "Re-probe on every run"). Written on every probe,
  # whether it succeeds (flips the field back to `enabled`) or comes back logged-out again.
facebook_group_discovery: RETIRED 2026-09-12 -- groups live entirely in the source registry
  # (collector/source_registry.json, `source_type: group`; see "Monitored Facebook groups" above).
  # No candidate counter, no review-state field, no shortlist, no close gate: the Group Potential Rule
  # (playbooks/10_LEAD_COMPETITOR_DETECTION.md) registers every group `active`/`not_selected`/
  # `no_access` automatically, and the Boss's only lever is pause/resume on the Sources page.
instagram_lead_source: enabled | web_only | pending
instagram_lead_source_updated_at:
instagram_web_only_reason:
instagram_discovery_first_pass_done: true | false
instagram_last_login_probe_at:
x_lead_source: enabled | web_only | pending
x_lead_source_updated_at:
x_web_only_reason:
x_discovery_first_pass_done: true | false
x_last_login_probe_at:
  # `instagram_*`/`x_*` mirror the five `facebook_*` fields directly above -- same enums, same
  # semantics, same gate wording, only the platform name changes. Both platforms share the one
  # extension the Facebook fields already describe (Setup step 4, "Kết nối Facebook, Instagram and
  # X"), so there is no separate install/connect step for them: `instagram_lead_source` and
  # `x_lead_source` resolve from the same extension check-in that resolves `facebook_lead_source`,
  # and `{platform}_discovery_first_pass_done` gates that platform's FIRST RUN vs DAILY budget tier
  # in the Social Discovery Pass exactly the way `facebook_discovery_first_pass_done` does.
  # `{platform}_last_login_probe_at` is written by the run-time re-probe (see below), on every
  # platform, the same way.
  # Auto web_only on logged-out (all three platforms, OWNER DECISIONS 2026-09-10 night): when a
  # platform's first call in a run comes back logged-out (extension login probe, or a job's
  # `stopped_because: logged_out`), the run sets that platform's `{platform}_lead_source` to
  # `web_only` with `{platform}_web_only_reason` = "not logged in on {date}" (today's date), stamps
  # `{platform}_lead_source_updated_at`, and the awareness line
  # (`playbooks/06_AGENCY_REPORT_STANDARD.md`, "Persistent web-only awareness line") names that
  # platform. This is separate from the human's own explicit `web_only` choice at setup, but reuses
  # the same field/enum -- either path lands the client in `web_only` and either path clears the
  # same way: the Boss logs back in for that platform in the shared extension's Chrome profile: no
  # setup step repeats, and the field resolves back to `enabled` on the next run whose run-time
  # re-probe succeeds (playbooks/10_LEAD_COMPETITOR_DETECTION.md, "Re-probe on every run" -- NOT the
  # extension check-in, which carries no per-platform login state today; see playbooks/TODO.md,
  # "facebook_logged_in" handoff, for that future signal).
collector_setup_status_file:
notes:

## private_data_source_discovery

status: not_asked | approved_pending_first_scan | completed | declined
  # RETIRED 2026-09-12: recommended, postponed, partially_approved, approved, pending_human_approval,
  # pending_private_activation, active, blocked, declined. `approved_pending_first_scan`
  # is recorded for all categories at once, at the step-7 yes (playbooks/SETUP_FLOW_ENTRYPOINT.md);
  # the first run executes discovery and judges every result with the Group Potential Rule
  # (playbooks/10_LEAD_COMPETITOR_DETECTION.md) straight into monitored sources -- no per-category
  # approval state, no shortlist.
reassurance_shown:
  professional_setup_once: true | false
  local_data_only: true | false
  daily_scan_prevents_missed_signals: true | false
why_recommended:
coverage_limitation_if_skipped:
categories:
  membership_sources:
    status: not_asked | approved_pending_first_scan | completed | declined
    platforms:
    - platform:
      discovery_urls:
      - url:
        status: not_tried | pending_private_activation | scanned | login_required | platform_url_changed | failed
        last_scanned_at:
  following_sources:
    status: not_asked | approved_pending_first_scan | completed | declined
    platforms:
    - platform:
      discovery_urls:
      - url:
        status: not_tried | pending_private_activation | scanned | login_required | platform_url_changed | failed
        last_scanned_at:
  recommendation_feed_sources:
    status: not_asked | approved_pending_first_scan | completed | declined
    platforms:
    - platform:
      discovery_urls:
      - url:
        status: not_tried | pending_private_activation | scanned | login_required | platform_url_changed | failed
        last_scanned_at:
  keyword_search_sources:
    status: not_asked | approved_pending_first_scan | completed | declined
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
  state: active | not_selected | no_access   # automatic: recommended_*/optional -> active, skip_* -> not_selected, unreadable/not-joined -> no_access; no human approval
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

## feature_tour   # light state for the Feature Discovery Rule (tour-guide anti-spam/rotation)

introduced:                    # feature keys already surfaced to the human
-
declined:                      # feature keys the human declined (re-surface ~monthly, not never)
-
last_surfaced:                 # feature_key: YYYY-MM-DD, to rotate and avoid repeating twice in a row

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
- `scenes_created`
- `scene_editing_complete`
- `scene_editing_blocked`
- `render_ready`
- `rendered`
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

**This ledger is an audit trail, not the lead's home.** After it is written, the run passes it to
`tool crm-store ... lead capture --file <this file>`, which turns each lead row into a CRM contact
with a dated evidence hook and an activity row (`playbooks/10_LEAD_COMPETITOR_DETECTION.md`). The
`status` field here records what the SCAN concluded; the contact records what happens to the person
afterwards. Do not treat `needs_review` as a queue — nothing consumes it, and the review it implies
was already performed by the agent that classified the post.

**`name` is the author's name as the page showed it**, carried verbatim from the collected record.
A post scan always has it; omit the field only for a lead added from a bare post/reel/video url
before enrichment. Never put Facebook interface text there (`Top contributor`, `Verified account`,
`Anonymous participant`, a group name, a url) — the capture refuses those and leaves the contact
nameless, which is the honest result. `emails`/`phones` follow the same rule: carry what the
collector found, and expect them to be absent until an enrichment pass runs.

**`locked` is computed, never stored.** A CRM contact file carries no `locked` field. Whether a
contact is locked is computed by the bridge at read time from the plan's `max_contacts` and the
contacts' `created_at` order (oldest stay unlocked; a contact past `lead` into any `engaged+` stage
is never locked) — see `outreach/playbooks/13_CRM_CORE.md`, "Plan cap and locked contacts". Always
read `tool crm-store ... contact lock-status` for the current locked count; never derive it from
this ledger or from hand-counting contact files.

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
  "name": "Rick Silva",
  "profile_url": "https://www.facebook.com/profile.php?id=...",
  "post_url": "https://www.facebook.com/groups/.../posts/...",
  "captured_at": "2026-06-20T09:00:00-07:00",
  "safe_context_summary": "Person asked what to do after receiving an insurance non-renewal notice.",
  "evidence_snippet": "Short visible snippet when safe",
  "person_type": "homeowner in California with a lapsing policy",
  "sells_to_match": "person in California with a new asset, a new dependent, a new address, a new job or a switch to self-employment, a rising or expiring policy, or a risk they just became aware of",
  "fit": "high",
  "fit_reason": "Homeowner in the licensed area with a named policy event.",
  "intent": "explicit",
  "intent_reason": "Asks directly what to do about the non-renewal notice.",
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

The `hot_lead`/`warm_lead`/`watch_lead` values here are derived from `fit` × `intent`
(`playbooks/LEAD_QUALIFICATION_RULE.md` Step 4), not chosen independently; the `direct_need`/
`indirect_need`/... values stay the separate evidence label carried over from `lead type`. The CRM
contact this row becomes carries `fit:{high|medium|low}` and `intent:{explicit|implied|none}` tags
next to the existing `lead:{temp}` tag, sourced from this row's `fit` and `intent` fields.

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
- **Never store or transmit the operator's own credentials or secrets** (usernames, passwords, cookies, tokens, session/auth data, API keys) — the single absolute prohibition. All other data the operator's setup + command directs — business data, prospect contact details (email/phone), evidence snippets, source URLs — may be stored and combined for lead-finding and personalization.

### Former per-month group shortlist file (retired 2026-09-12)

RETIRED 2026-09-12: the source registry (`collector/source_registry.json`, `source_type: group`) is
the only store for Facebook group candidates — state, potential, scans, and lead history all live on
the registry entry (see "Monitored Facebook groups" above). No shortlist file, no `decision` field,
no per-group approval; the Group Potential Rule and `tool source-registry add|record|pause|resume`
(`playbooks/10_LEAD_COMPETITOR_DETECTION.md`, `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md`)
are the only writers now.

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

### Discovered sources (`collector/source_registry.json`, `kind: discovered`)

Purpose:

- Persist a Step 5 "likely" verdict (`playbooks/10_LEAD_COMPETITOR_DETECTION.md`, "Step 5") as a
  standing note the Boss can approve and harvest later — never harvested by the run that found it.
- Back the dashboard's `/ui/{client}/sources?tab=discovered` list and the
  `review_discovered_sources` / `harvest_thread` catalogue rows (`playbooks/NEXT_JOB_CATALOGUE.md`).

One entry per recorded thread, written by `tool source-registry discovered add` and updated by
`tool source-registry discovered approve|dismiss|mark-harvested` — never hand-edited. (There is no
`discovered update` subcommand — the status-specific ones above are the only writers.)

```json
{
  "id": "sha256(post_url), short form",
  "kind": "discovered",
  "client_slug": "client",
  "platform": "facebook",
  "post_url": "https://...",
  "community": "Group or page name",
  "author_line": "Original poster's name/handle",
  "excerpt": "first 300 chars of the post",
  "comment_source_reason": "one line, from Step 5",
  "types_match": true,
  "detected_at": "ISO-8601",
  "last_seen_at": "ISO-8601",
  "run_id": "...",
  "comment_count": 0,
  "status": "new",
  "approved_at": null,
  "harvested_at": null,
  "harvest_job_id": null,
  "leads_added": 0,
  "authors_seen": 0
}
```

`id` is derived as a stable hash of `post_url` — dedupe key. `status` allowed values: `new |
approved | harvested | dismissed`; there is no expiry field and no TTL — a discovered source lives
until a human dismisses it or it is harvested, and `detected_at` stays available for filtering
by age. Re-detecting the same `post_url` on a later run updates `last_seen_at` and `comment_count`
only — it never resets `status`, `approved_at`, or `harvested_at`. `status` only ever advances
`new -> approved -> harvested`, or to `dismissed` from `new`/`approved`; a `harvested` source is
never re-harvested except by a fresh, explicit Boss order for that same `id`.

**The harvest job's folder layout** is produced by `tool harvest-thread prepare` / `ingest`
(`solo-agency-collector/bridge-go/harvest_thread.go`) — this is the actual, shipped layout, not a
target to redesign:

```text
history/YYYY-MM/harvest/{id}/
  context.json        (post_url, platform, community, excerpt, author_line, comment_source_reason)
  batch_01.json … batch_NN.json   (40 deduped-by-author rows each, pre-filtered; flat, JSON not JSONL)
  run_report.json     (written by ingest: kept/dropped/medium rows, leads_added, authors_seen)
```

`{id}` is the discovered source's own `id` (the stable hash of `post_url`) — there is no separate
`harvest_job_id`; the registry's `harvest_job_id` field is stamped with this same `id`.

**The dashboard's approve action writes a `ui_inbox` request** — the actual shipped write, per
`docs/UI_DESIGN.md`: `clients/{c}/{bl}/ui_inbox/harvest_thread_requests.jsonl` (no `outreach/`
segment, no per-line `ts`/`ui_session` field) — one line
`{kind: "harvest_thread", client, source_id, requested_at}`, appended by `POST
/api/ui/{client}/sources/discovered/{id}/approve`, and only on the call that actually flips that
source's `status` from `new` to `approved` (which also stamps `approved_at`) — a repeat approve on
a row already `approved` or `harvested` is a no-op and writes no line. `.../dismiss` sets `status:
dismissed` and writes no request. The Team Leader consumes new lines the same way it consumes
`shortlist_decisions.jsonl` (cursor-based, at the start of the next session or run) and treats a
`harvest_thread` line as the Boss's own order to run `tool harvest-thread prepare --id {source_id}`
then `ingest` for that `source_id`. Both `prepare` and `ingest` refuse to run on a source whose
`status` is not `approved` (or `harvested` with `--force`, for an explicit Boss-ordered
re-harvest) — `new` and `dismissed` are refused outright.

**Standup field.** `daily-content-pipeline/automation/standup.jsonl` (below) carries `discovered_new`
— the count of `likely` threads THIS run recorded — on every scheduled-run line, `0` when the Social
Discovery Pass did not run or found none.

---

## Analytics, Funnel, And Lead Classification Rules (Canonical Copy In Stage 9)

The canonical maintained copy of §23.1/23.2/23.8/23.9 (offer/funnel mapping, lead classification, published URL measurement, and analytics log rules) lives in `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md` and must not be duplicated here.

Storage-schema note: the three per-client analytics logs written under each client's `analytics/` directory are `metrics_log.md`, `learning_log.md`, and `comment_signal_log.md`.
