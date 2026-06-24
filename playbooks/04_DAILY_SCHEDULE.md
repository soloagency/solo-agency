# Daily Schedule

Stage: `04`

## Load Rule

Load during one-time setup after the Client Intelligence Profile, public data source plan, and private data source status are known, so the routine and client-specific automation task can be configured before the first report. Also load during scheduled runs and whenever routine/schedule config is reviewed or repaired.

## Hard Gates For This Stage

- During one-time setup, configure schedule/routine and the client-specific automation task after the basic source plan is known.
- After configuring the routine, do not run the first report in Setup Flow; verify the client-specific automation task and tell the human the exact task name to run for the first report.
- Support manual-only, daily, multiple-times-daily, weekly, and environment-specific schedules.
- Scheduled runs must run research, private scans if active, analysis, production-ready drafts, approved video/blog/social asset creation when provider setup and explicit approvals allow it, HTML report, and notification.
- Scheduled runs must load Stage 10 and produce Lead & Competitor Opportunities, or explicitly mark them as not found, not scanned, pending activation, or unavailable.
- Scheduled runs must run published-URL analytics and measurement-learning only when published URLs/metrics exist. On the first run with no published history, mark measurement as `no published URLs yet` instead of pretending it ran.
- Scheduled runs must load the needed playbooks again at run time; they must not rely on memory from setup.
- Every scheduled-run human-facing reply, notification, or report handoff must include an updated progress block. If the agent sends multiple progress updates during the scheduled run, each update must show the current completed/current/remaining state.
- If private collection is blocked, continue public data sources and notify the human. Do not fall back to Claude in Chrome, Codex/browser tools, Playwright/Puppeteer/Selenium, or another agent-controlled browser for private data sources.
- Store schedule config and notification channel.
- After any human-approved change made after the schedule/automation was created, perform Automation Resync before claiming the next scheduled run is updated.
- Never say "the automation is updated" if only `collector_config.json`, only `schedule.md`, or only the Client Intelligence Profile was changed. The whole automation package must be synced or the remaining snapshot/update blocker must be stated.

## Latest Override: Client-Specific Automation Tasks

The current Solo Agency model uses separate Setup Flow and Automation Flow.

Setup Flow must create/update schedule and automation tasks, but must not run the first report directly. The first real report must be executed by a client-specific automation task.

Rules:

- Create one client-specific automation task per active client by default.
- Every client-specific task name must begin with the client name, for example `AvenNgo - Solo Agency Daily Run`.
- The task prompt must pin `target_client_slug` and must not process other clients.
- The task may use the shared Local Collector app/bridge, but private data source jobs must be routed by `client_slug + extension_instance_id`.
- If the AI automation environment cannot call `127.0.0.1`, it must use file-based job requests under `daily-content-pipeline/collector/jobs/pending/` and read bridge/extension health from local files.
- A setup/config session may instruct the human to run `AvenNgo - Solo Agency First Run`, but it must not generate the report inside the setup chat.
- Automation Flow may accept config changes during a real run, but must immediately perform Automation Resync before claiming future runs are current.

For multi-client daily operations, prefer separate client tasks plus an optional master digest task. The master digest task must not scan private data sources; it only reads existing client reports/outputs and summarizes them.

## Source Preservation Rule

This file is detailed source material moved from the original monolithic `SOLO_AGENCY_PLAYBOOK.md`.

Do not summarize away requirements, examples, checklists, schemas, protocols, URLs, edge cases, warnings, approval gates, or completion gates. If a downstream agent needs to shorten its response to the human, it may summarize the response, but it must still obey the full requirements in this file.

---

## Scheduled Run Playbook Loading Contract

A scheduled run is not a shortcut around the playbook. It is the same agency workflow executed with saved context instead of asking the human setup questions again.

At the start of every scheduled run, the agent must load or re-load the relevant stage files for the work it is about to do:

1. Always load Stage 0: `00_CORE_CONTEXT_REQUIREMENTS.md`.
2. Always load Stage 7: `07_STORAGE_SCHEMA_AND_HISTORY.md` to read profiles, logs, ledgers, and history.
3. Always load Stage 4: `04_DAILY_SCHEDULE.md` for the scheduled daily-run contract.
4. Load Stage 1 only if a profile is missing, incomplete, stale, or needs setup repair. Do not ask setup questions when the saved profile is complete.
5. Load `playbooks/PRIVATE_SOURCE_GATE.md`, Stage 2, Stage 8, and Stage 9 when private data sources are active, pending, blocked, or being scanned.
6. Load Stage 3 when drafts, production, publishing, provider setup, or notification provider actions are needed.
7. Load Stage 5 when any published content exists or when yesterday/last-7-day measurement is due.
8. Load Stage 6 whenever generating the human-facing HTML report.
9. Load Stage 10 whenever lead/competitor opportunities, comments, opportunity logs, or competitor monitoring are part of the run. This is normally every first run and every scheduled daily run.
10. Load Stage 9 before claiming the scheduled run is complete.

The difference between first setup and scheduled runs:

- First setup asks only the minimum setup questions because the profile does not exist yet.
- Scheduled runs read the saved Client Intelligence Profile, source lists, collector config, content history, publishing ledger, and analytics logs, then continue automatically.
- Scheduled runs must not re-ask industry, sub-industry, audience, pain points, content pillars, or private data source setup questions if those fields are already present.
- Scheduled runs may ask the human only when an approval gate, blocker, missing critical field, expired private session, production/render/publish/credit decision, or lead outreach decision requires human input.

Scheduled run completion requires the same end-to-end path as a manual daily run: public research, private scans if active, published-URL analytics when published content exists, data analysis, Lead & Competitor Opportunities, idea matrix, best idea, production-ready drafts, approved video/blog/social production when authorized, HTML report, notification, and measurement/learning when measurement data exists.

## Scheduled Run Progress Display Contract

Scheduled runs are meant to be automatic, but the human still needs visible state whenever the agent speaks.

Every scheduled-run reply, notification, or report handoff must include:

- completed steps;
- current active step;
- remaining steps;
- blockers or human decisions required;
- whether published-URL analytics was run or skipped because no published URLs/metrics exist yet.
- an `Automation freshness check` stating whether the latest changes are synced into the configured automation/scheduled task and whether tomorrow's run will read the current contracts/prompts/playbooks/source approvals/state, not only the latest config file.

Use this title:

```text
Solo Agency daily run progress
```

The agent may use a compact form in notifications, but it must not send only a report link or summary while steps remain.

Use this compact automation freshness line in scheduled-run updates and setup/repair progress blocks after a schedule exists:

```text
Automation freshness check: {✓ current | → resync in progress | ! action needed | – not applicable yet} - latest approved changes synced into the automation/scheduled task, including prompt/contract/playbook/source state, not only config: {yes | in progress | needs human task prompt update | no schedule yet}.
```

---

## Automation Resync Contract

An automation/scheduled task can contain a stale prompt snapshot from the moment it was created. A later config edit is not enough if the scheduled task still points to old instructions, old source state, or old setup assumptions.

Trigger Automation Resync whenever a human-approved change happens after schedule/automation setup, including:

- private data source discovery was run, approved, rejected, postponed, or changed;
- Local Collector was activated, repaired, moved to another folder, or found to be writing to the wrong workspace;
- public data sources, public search keywords, client profile fields, pain points, content pillars, audience, location, or offer changed;
- PDNA, WideCast, Telegram, publishing, analytics, published URL history, or notification delivery changed;
- schedule cadence, timezone, active clients, manual-only mode, or report delivery channel changed;
- the playbook behavior changed in a way scheduled runs must follow.

Automation Resync requires updating every relevant layer:

1. Client Intelligence Profile: current source status, approvals, profile fields, private monitoring activation, PDNA, analytics, and notification status.
2. Source/history logs: discovery results, approved/rejected/pending private data sources, new public data sources, keyword bank changes, and approval timestamps.
3. `daily-content-pipeline/schedule.md`: cadence, included clients, notification channel, private data source status, PDNA status, and last resync timestamp.
4. `daily-content-pipeline/collector/collector_config.json` or `POST http://127.0.0.1:17321/config`: only when private data source collection schedule/sources/scan depth changed.
5. `daily-content-pipeline/automation/automation_manifest.md`: current run contract, paths, active clients, prompt source, config source, and last known state hash/summary.
6. `daily-content-pipeline/automation/scheduled_run_prompt.md`: the exact prompt that the native AI automation/scheduled task should run.
7. Native AI automation or scheduled task body: update it when the environment stores a separate prompt snapshot.
8. `daily-content-pipeline/automation/resync_log.md`: what changed, what files/tasks were updated, what could not be updated, and what the next scheduled run should see.

If the native AI automation task cannot be edited by the agent, the agent must:

- write the exact replacement prompt to `daily-content-pipeline/automation/scheduled_run_prompt.md`;
- mark `automation_prompt_update_pending` in `automation_manifest.md` and `schedule.md`;
- give the human one concrete instruction to paste/replace the scheduled task prompt;
- avoid claiming the scheduled run is fully updated until the human confirms the native task body was updated.

Automation Resync verification:

Before saying a post-schedule change is complete, do a dry-read as if tomorrow's scheduled run were starting:

1. Read `playbooks/SCHEDULED_RUN_ENTRYPOINT.md`.
2. Read `daily-content-pipeline/automation/automation_manifest.md`.
3. Read `daily-content-pipeline/schedule.md`.
4. Read each active Client Intelligence Profile.
5. Read `collector_config.json` when private data sources are active or pending.
6. Confirm the latest user-approved changes are visible from those files and from the scheduled prompt/task body.

The agent's human-facing completion message must say one of:

```text
Automation Resync complete: the next scheduled run will read the latest approved state.
```

or:

```text
Automation Resync partially complete: config/profile are updated, but the native scheduled task prompt still needs the human to replace it with daily-content-pipeline/automation/scheduled_run_prompt.md.
```

Bad completion wording:

```text
I updated the config, so tomorrow's automation is fixed.
```

This is invalid because it hides the possibility that the scheduled prompt/task still has an old snapshot.

---

## 20. Scheduling Rule

The agent must use the best scheduling mechanism available in the current environment.

Possible scheduling methods:

- Native AI scheduled task.
- Native AI automation.
- Local cron.
- Windows Task Scheduler.
- macOS launchd.
- n8n.
- Make.
- Zapier.
- GitHub Actions.
- Server job.
- Desktop reminder.
- Manual daily run instructions.

The playbook does not require one specific scheduler because different AI services have different capabilities.

The agent must record the chosen method in `schedule.md`.

The agent must also record the notification channel in `schedule.md`. If WideCast MCP notification/Telegram tooling is available, record it as the preferred notification channel for scheduled runs, even if Telegram is not connected yet, because WideCast can fall back to email. If WideCast notification tooling is unavailable but Gmail/email is connected, record Gmail/email as the secondary fallback notification channel. If neither is available, record `notification_channel: local_path_only` and tell the human how to connect WideCast notification/Telegram or Gmail/email.

Scheduled runs should be designed as unattended runs. The human may not be watching the AI agent UI, so the agent must proactively notify the human when the run finishes or when human action is required.

If no automation is available:

1. Explain the limitation.
2. Create manual run instructions.
3. Provide the exact command or prompt the human should use each day.

Example manual run prompt:

```md
Run the daily content pipeline for every active client in clients_index.md. Produce today's outputs and master digest.
```

---

## 15. Daily Run Algorithm

For each daily run:

1. Load `clients_index.md`.
2. Identify all clients with `active` status.
3. For each active client:
   1. Load the client's Client Intelligence Profile file.
   2. Validate required fields.
   3. If the Client Intelligence Profile is incomplete, enter setup repair mode.
   4. Prepare the current month folder key `YYYY-MM`.
   5. Load saved `public_data_sources` and visit/check active due public data sources before or alongside keyword search.
      - Visit sources where `visit_in_scheduled_runs: true` and cadence is due today.
      - Prioritize `active_public_source` daily sources, then due `weekly_public_source` sources, then relevant `occasional_public_source` sources when the topic/event matches.
      - Record source status, useful URLs, useful signals, weak/noisy results, and whether the source should stay active, be promoted, or be demoted.
   6. Use Google Search or an available equivalent search tool with rotating keywords from `public_search_keywords`.
      - Do not use only generic industry keywords.
      - Prioritize pain-point/problem/need/buying-intent keyword clusters because these are closer to real audience demand.
      - Use keywords in the target audience's likely search/comment language. Do not translate the keyword bank into the human's chat/report language unless the audience uses that language.
      - Include at least one broad primary-industry keyword for context, at least one pain-point/problem keyword, at least one need/goal or buying-intent keyword, and local/location keywords when location matters.
      - Use a smaller rotation of related-industry keywords only when the bridge back to the client's offer is clear.
      - If results are weak, try a different pain-point/problem/need cluster before giving up.
      - Record every keyword used, keyword group, result quality, useful URLs, and final keyword status.
      - Extract new keyword candidates from useful search results, public discussions, questions, competitor hooks, comments, and emerging phrases. Add useful new candidates to the keyword bank with source/reason, related pain point, and content pillar.
      - Detect useful recurring public data sources from search results and public pages. Promote strong recurring sources into `public_data_sources` with status/cadence so future scheduled runs can visit them automatically.
      - Include this record in the daily report section `Public Search Keywords Used Today`.
      - If no search was possible, explicitly explain the blocker in that same section.
   7. Before deciding whether to skip private data sources, perform Collector Runtime Verification whenever any of these are true:
      - private data sources are active, pending, requested, approved, present in the Client Intelligence Profile, or listed in any source approval/history file;
      - schedule/config says `public_data_sources_only`, `private sources postponed`, or `pending_private_activation`, but the workspace contains Local Collector files;
      - `daily-content-pipeline/collector/inbox/bridge_health.json`, `daily-content-pipeline/collector/inbox/collector_status.json`, `daily-content-pipeline/collector/collector_setup_status.md`, or recent `daily-content-pipeline/collector/inbox/YYYY-MM/*/collector_status.json` exists.
      Do not treat saved labels such as `pending_private_activation` or `public_data_sources_only` as final without this runtime check; those labels may be stale after a human later installed, repaired, or reconnected the Local Collector.
   8. Load `playbooks/PRIVATE_SOURCE_GATE.md`, Stage 2, Stage 8, and Stage 9 before any Collector Runtime Verification involving private data sources. Do not use Claude in Chrome, Codex/browser tools, Playwright/Puppeteer/Selenium, or another agent-controlled browser as a fallback.
   9. Try to check private collector health through `GET http://127.0.0.1:17321/status`.
      - If the request succeeds, record `bridge_status: running`, check `status.persistent`, `status.job_available`, `status.output_dir`, `status.counts`, and `status.extension_health`.
      - If the bridge is online but `/status.config_file`, `/status.output_dir`, or `/status.run_now_request_file` points outside the current setup's `daily-content-pipeline/collector/` tree, mark `wrong_workspace_bridge`, do not run private collection, ask the human to run the current setup's Local Collector command, and remind them to remove/disable old Solo Agency Local Collector extensions in `chrome://extensions`.
      - If the bridge is online but `extension_health.status` is `stale` or `no_extension_check_yet` after the 75-second extension check grace window, mark private collection as unavailable for this run and notify the human.
      - If the workspace identity check passes and `extension_health.status` is `recent`, continue private collection.
   10. If `GET http://127.0.0.1:17321/status` fails, do not immediately conclude the Local Collector is inactive. In some scheduled-task environments, the AI sandbox's `127.0.0.1` is not the human computer's Local Collector localhost.
      - Read `daily-content-pipeline/collector/inbox/bridge_health.json` when present.
      - Read `daily-content-pipeline/collector/inbox/collector_status.json` when present.
      - Read `daily-content-pipeline/collector/collector_setup_status.md` when present.
      - Inspect recent `daily-content-pipeline/collector/inbox/YYYY-MM/*/collector_status.json` files.
      - Inspect recent consumed run-now status files such as `run_now_request_status.json`, `run_now_request.consumed.json`, or timestamped `run_now_request*.consumed.json` files when present.
      - If those local status files show a recent current-workspace bridge and recent extension check, use the Stage 8 file-based run-now path by writing `daily-content-pipeline/collector/run_now_request.json` and waiting for collector output. Do not ask the human to restart the Local Collector just because the API was unreachable from the AI sandbox.
      - If the files are missing, stale, point to another workspace, or do not prove a recent extension check, mark the precise blocker: `collector_status_unverified`, `collector_offline_or_unreachable`, `wrong_workspace_bridge`, or `extension_status_unknown`.
   11. If no private data sources are configured, and discovery was never offered or was postponed, do not block the scheduled run. Continue with public data sources, but include `Private Data Source Discovery Recommended` or `Private Data Source Discovery Declined/Postponed` in the report/notification. Explain that public-only runs can still produce useful ideas but may miss community, lead, and competitor signals from logged-in/member spaces.
   12. If private data sources remain unavailable after Collector Runtime Verification, continue with public data sources and previously collected private data when available. Log the exact verification outcome in the report and notification; do not merely say the config was public-only.
   13. Prepare the private data source queue if private data sources are available and collector health is acceptable:
      - keep the active daily queue around 20 sources or fewer per client by default;
      - prioritize sources most relevant to the client, target audience, target location, pain points, and content pillars;
      - classify extra sources as `weekly` or `optional` and rotate them across future runs;
      - do not run aggressive or parallel private data source scans for the same logged-in account.
   14. Check private data sources if available, using the Solo Agency Local Collector extension plus the Local Collector app when available, with `collector_config.scroll_delay_seconds` defaulting to 5 seconds and `collector_config.max_scrolls_per_source` defaulting to 5.
   15. If the collector bridge was started in `agent_on_demand` mode, stop it after collection completes or after timeout.
   16. Log skipped, pending-activation, expired, rate-limited, warning-triggered, collector-unavailable, extension-unavailable, Chrome-not-running, stale-extension, bridge-offline, collector-status-unverified, wrong-workspace, or unavailable private data sources.
   17. Load yesterday's private data for this client when available and filter duplicate or near-duplicate data points using visible text matching. Do not parse private-platform HTML for duplicate detection.
   18. Extract relevant `[data_points]`, including reference URLs for every data point. Keep data points that are directly about the primary industry or clearly connected through a related industry. Discard related-industry data when the bridge back to the client's offer is weak.
   19. Add newly recommended private groups/pages/profiles/communities to `New Private Data Sources Detected` and `history/YYYY-MM/new_private_sources_log.md`.
   20. Load Stage 10 and detect hot/warm/watch leads plus direct, indirect, adjacent, attention, and authority competitors during the same research/private-scan pass. The first lead/competitor pass for a client/source set should use 10 scrolls per approved private data source when safe; normal daily runs use 5 scrolls per approved private data source by default.
   21. For every useful lead or competitor opportunity, preserve profile URLs and post/current URLs when available, safe context summaries, reasoning, suggested human action, and a copy-ready value-first comment in the same language as the post.
   22. Generate the 3x2 idea matrix, labeling each idea as `primary_industry` or `related_industry`.
   23. Check `history/YYYY-MM/content_log.md`, including the recent primary/related ratio.
   24. Select the best idea of the day.
   25. Write the configured WideCast-writing-skill draft using the writing skill fallback if MCP/account is unavailable.
   26. If a production provider is connected and the human has explicitly approved creation/rendering/publishing for a selected draft, load Stage 3 and create the approved video/blog/social asset according to provider approval gates. If approval or provider setup is missing, keep the asset as `approval_required` or `provider_setup_required`.
   27. Save `outputs/YYYY-MM/YYYY-MM-DD.md` as the canonical source-of-truth report.
   28. Generate `outputs/YYYY-MM/YYYY-MM-DD.html` as a polished standalone human-facing report. It must be factually aligned with the Markdown report, mobile-friendly, and include editable draft review blocks when drafts exist.
   29. Update or copy `outputs/latest.md`.
   30. Update or copy `outputs/latest.html`.
   31. Update `history/YYYY-MM/content_log.md`.
   32. Update `history/YYYY-MM/data_sources_log.md`.
   33. Update `history/YYYY-MM/lead_log.md`.
   34. Update `history/YYYY-MM/competitor_log.md`.
   35. Update `history/YYYY-MM/lead_competitor_opportunities.jsonl` when possible.
4. Create or update `outputs/YYYY-MM/YYYY-MM-DD_master_digest.md`.
5. Generate `outputs/YYYY-MM/YYYY-MM-DD_master_digest.html` as a polished standalone human-facing master report.
6. Update or copy `outputs/latest_master_digest.md`.
7. Update or copy `outputs/latest_master_digest.html`.
8. Present the daily digest to the human.
9. Load Stage 6 and run the Report Delivery Capability Check before claiming the run is complete.
10. Prepare a report-delivery record containing the local `.html` report path, WideCast capability check status, tool discovery method, upload attempt status, uploaded report URL if available, notification channel, final notification report link, and blockers.
11. If WideCast MCP notification/Telegram capability is available, inspect whether an HTML-capable WideCast report/file/asset upload API is available. Use the current environment's tool discovery/lazy-load mechanism before declaring unavailable. If upload exists, upload the HTML report to WideCast first, then send a notification to the human that includes the uploaded WideCast report URL, agent identity, run status, clients processed, blockers, lead/competitor counts, and required actions.
12. If WideCast notification is available but HTML upload is unavailable or fails, log the exact upload blocker and still send a WideCast notification that includes the best available local/hosted `.html` report path/link.
13. If the current AI connector/tool surface does not expose WideCast upload or notification tools, log `widecast_report_upload_unavailable` and/or `widecast_notification_tool_unavailable`, say this is a current tool-surface blocker, and provide the best available HTML path/link in chat or an authorized fallback channel.
14. If another authorized channel can send the HTML file or link more conveniently only because WideCast notification tooling is unavailable or blocked, use it.
15. Log the upload attempt and notification attempt in `notifications/notification_log.md`.

The daily run is complete only when every active client is processed or explicitly logged as skipped.

When presenting the daily idea list to the human, include reference URLs next to data points, top ideas, and the selected best idea so the human can verify the information. For private data, include the captured source URL and note that it may require the human's logged-in session.

Scheduled runs must assume the human may not be present in the AI agent UI. The run is not fully operationally complete until the mobile-friendly HTML result or a result-ready notification with the HTML path/link has been sent through the configured notification channel, preferably WideCast MCP / Telegram.

If WideCast notification/Telegram is connected and WideCast report upload supports HTML, the notification link must be the uploaded WideCast report URL, not only a local file path. If upload fails or the current wrapper does not support HTML upload, log the blocker and send the best available HTML path/link.

If a WideCast upload/Telegram step is skipped because the agent did not inspect available tools/connectors, the scheduled run is incomplete. The agent must correct the omission by running the Report Delivery Capability Check, updating `notification_log.md`, and sending a correction message with the HTML report URL/path and blocker.

A notification that only says the report is ready but contains no HTML report URL/path is invalid. If this happens, immediately send a correction notification with the HTML report URL/path and log the correction.

---

- During one-time setup, after the profile and source plan are known and before the client-specific automation task is marked ready, ask the human whether they want daily, multiple-times-daily, weekly, manual-only, first-run-only, or another cadence.
- Then write or update `schedule.md`, `daily-content-pipeline/automation/automation_manifest.md`, `daily-content-pipeline/automation/scheduled_run_prompt.md`, `daily-content-pipeline/automation/resync_log.md`, and the relevant collector/native automation config files.

Exact schedule contract:

- Scheduled runs are configured in `daily-content-pipeline/collector/collector_config.json`, or through `POST http://127.0.0.1:17321/config` when the Local Collector app is running.
- Scheduled runs use `scheduled_windows`. They do not use `/jobs/run_now`.
- A daily default schedule should look like this:

```json
{
  "version": "0.1.0",
  "timezone": "local",
  "run_mode": "persistent_bridge_scheduler",
  "default_runs_per_day": 1,
  "poll_interval_seconds": 5,
  "max_sources_per_run": 20,
  "max_scrolls_per_source": 5,
  "max_scrolls_allowed": 10,
  "scroll_delay_seconds": 5,
  "duplicate_filter": {
    "compare_against_previous_day": true,
    "method": "visible_text_matching",
    "parse_html": false
  },
  "scheduled_windows": [
    {
      "name": "daily_morning",
      "enabled": true,
      "local_time_start": "09:00",
      "local_time_end": "09:30",
      "days": ["mon", "tue", "wed", "thu", "fri", "sat", "sun"]
    }
  ],
  "clients": [
    {
      "client_slug": "client-slug",
      "enabled": true,
      "sources": [
        {
          "name": "Competitor page or private group name",
          "url": "https://www.facebook.com/groups/example",
          "platform": "facebook",
          "source_type": "private_group",
          "priority": "high"
        }
      ]
    }
  ]
}
```

- For multiple scheduled runs per day, add multiple enabled items to `scheduled_windows`, for example `morning`, `midday`, and `afternoon`.
- For manual-only mode, set all `scheduled_windows[].enabled` values to `false` and rely only on `/jobs/run_now`.
- If the human has not activated private data source monitoring yet, configure the recurring schedule as public data sources only and clearly mark private data sources as `pending_private_activation`.
- Only configure scheduled private data source collection after Local Collector activation is accepted and collector health is confirmed or explicitly documented as pending/blocker.
- The Local Collector app must run in persistent mode for unattended scheduled collection:

```text
solo-agency-local-collector/bin/collector-bridge-darwin-arm64 \
  --host 127.0.0.1 \
  --port 17321 \
  --config-file daily-content-pipeline/collector/collector_config.json \
  --output-dir daily-content-pipeline/collector/inbox \
  --persistent
```

- The Solo Agency Local Collector extension polls `/status`; when the current local time is inside an enabled `scheduled_windows` item and private data sources exist, `/status` should expose a scheduled job with `current_job_type: scheduled` and `job_available: true`.
- Scheduled run IDs are generated by the Local Collector app, usually using `YYYY-MM-DD_schedule-name`.
- The agent must still write a human-readable `schedule.md` explaining the cadence, clients included, private data source limits, and notification behavior.
- The agent must also write `daily-content-pipeline/automation/automation_manifest.md` and `daily-content-pipeline/automation/scheduled_run_prompt.md` so future agents can repair or resync the actual scheduled task prompt instead of relying on memory.

---

### Collector Schedule Configuration

The collector must use one shared local configuration format so AI agents, the bridge, and the Chrome extension control panel do not conflict.

Required config file:

```text
daily-content-pipeline/collector/collector_config.json
```

Default config:

```json
{
  "version": "0.1.0",
  "timezone": "local",
  "run_mode": "persistent_bridge_scheduler",
  "default_runs_per_day": 1,
  "poll_interval_seconds": 5,
  "max_sources_per_run": 20,
  "max_scrolls_per_source": 5,
  "max_scrolls_allowed": 10,
  "scroll_delay_seconds": 5,
  "duplicate_filter": {
    "compare_against_previous_day": true,
    "method": "visible_text_matching",
    "parse_html": false
  },
  "scheduled_windows": [
    {
      "name": "daily_default",
      "enabled": true,
      "local_time_start": "09:00",
      "local_time_end": "09:30",
      "days": ["mon", "tue", "wed", "thu", "fri", "sat", "sun"]
    }
  ],
  "clients": []
}
```

The AI agent must create this file during first setup if it does not exist.

If the human wants multiple collection runs per day, the same file must be updated instead of creating another schedule format. Example:

```json
{
  "scheduled_windows": [
    { "name": "morning", "enabled": true, "local_time_start": "08:30", "local_time_end": "09:00", "days": ["mon", "tue", "wed", "thu", "fri"] },
    { "name": "midday", "enabled": true, "local_time_start": "12:00", "local_time_end": "12:30", "days": ["mon", "tue", "wed", "thu", "fri"] },
    { "name": "afternoon", "enabled": true, "local_time_start": "16:00", "local_time_end": "16:30", "days": ["mon", "tue", "wed", "thu", "fri"] }
  ]
}
```

The extension control panel may update this file by calling the bridge config endpoint. The AI agent may also update this file during setup when the human asks for a schedule. Both must preserve the same schema.

When the Local Collector app is already running, it should check whether `collector_config.json` changed on each `/status` request and reload the file when its timestamp or size changes. To apply an intentional schedule change, prefer `POST http://127.0.0.1:17321/config` when available. If the agent cannot call the endpoint but can edit the config file, direct file edits are acceptable because the Local Collector app should auto-reload them through `/status`. Do not use schedule edits for manual run-now collection.

### Persistent Bridge Scheduler Mode

For fully unattended operation, especially with Claude or other sandboxed agents that cannot start a binary directly, use `run_mode: persistent_bridge_scheduler`.

In this mode:

- The bridge runs as a lightweight local background process.
- The extension checks the bridge every `poll_interval_seconds` while Chrome is active and the extension service worker is awake.
- The extension should also check immediately after install, browser startup, and settings save.
- If Chrome suspends the extension service worker, Chrome alarms are the fallback and the practical check interval may be about 1 minute until the worker wakes again.
- The bridge returns the current collection window and today's run status.
- If the current local time is inside an enabled collection window and the run has not been completed for that window, the extension starts collecting automatically.
- After collection, the extension posts results to the bridge.
- The bridge marks that window as completed so the extension does not repeat it until the next scheduled window.
- The human does not need to open the extension panel or click anything during normal daily runs.

Default behavior:

- One run per day.
- One daily collection window.
- 5 second extension bridge check interval when Chrome is active and the bridge is running.
- About 60-75 second practical fallback window when Chrome has suspended the extension service worker.
- 5 scrolls per private data source.
- 5 seconds between scrolls.
- Maximum configurable scrolls: 10.

Panel visibility rule:

- The extension panel must show the current collector status.
- During a run, the panel should show:
  - current client,
  - current source/platform,
  - current scroll number,
  - maximum scroll count,
  - data points collected,
  - leads detected,
  - competitors detected,
  - new private data sources detected,
  - last bridge contact time,
  - last error or blocker.

The panel is for visibility and configuration, not for required daily operation.

### Private Collector Health Check Protocol

Before every scheduled run, after every scheduled run, and whenever private data is missing, the AI agent must check the private collector health.

Health check sequence:

1. Do not decide from saved config alone. If private data sources exist in any state, or if collector health/output files exist in the workspace, perform this runtime verification before saying private data sources were skipped.
2. Try `GET http://127.0.0.1:17321/status`.
3. If the request succeeds:
   - record `bridge_status: running`,
   - record `status.persistent`,
   - record `status.job_available`,
   - record `status.output_dir`,
   - record `status.counts`,
   - inspect `status.extension_health`.
4. If `extension_health.status` is `recent`, private collection infrastructure is currently healthy.
5. If `extension_health.status` is `no_extension_check_yet` immediately after extension install, bridge restart, or settings save, wait and re-check for up to 75 seconds before declaring private collection unavailable.
6. If `extension_health.status` is `stale` or `no_extension_check_yet` after the 75-second grace window, treat private collection as unavailable for now and identify likely causes:
   - Chrome is closed,
   - extension is not installed,
   - extension is disabled or removed,
   - Solo Agency Local Collector extension and Local Collector app URL/port mismatch,
   - Chrome service worker is asleep and has not woken recently,
   - browser profile is not the one where the extension was installed.
7. If `/status` fails:
   - do not immediately record `bridge_status: offline` as the final truth;
   - first consider AI sandbox localhost isolation, where the agent's `127.0.0.1` is not the human machine's Local Collector localhost;
   - read local collector status files before deciding:
     - `daily-content-pipeline/collector/inbox/bridge_health.json`,
     - `daily-content-pipeline/collector/inbox/collector_status.json`,
     - `daily-content-pipeline/collector/collector_setup_status.md`,
     - recent `daily-content-pipeline/collector/inbox/YYYY-MM/*/collector_status.json`,
     - recent `run_now_request_status.json` or `run_now_request*.consumed.json` files.
   - if those files show a recent current-workspace bridge and recent extension check, use the file-based run-now path from Stage 8 rather than asking the human to restart the collector;
   - if the files are missing, stale, or point to another workspace, record the exact blocker such as `collector_status_unverified`, `collector_offline_or_unreachable`, or `wrong_workspace_bridge`;
   - do not try to start the bridge from inside the AI agent sandbox during setup/repair;
   - continue with public data sources and previously collected private data if live private collection remains unavailable.
8. If the bridge is running but the extension is stale, do not keep retrying aggressively. Continue with public data sources, log the private data source blocker, and notify the human.
9. If the extension is recent but a private data source fails due to login/captcha/checkpoint/session expiry, skip that source, log the platform-specific issue, and notify the human.

The AI agent must surface this health information transparently in the daily report and in Telegram notifications when private data sources are unavailable.

Example notification:

```md
Agent: Claude Schedule
Collector status: bridge_running, extension_stale
Last extension check: 2026-06-20 08:52 local time
Likely cause: Chrome is closed or the extension is disabled.
Impact: Private Facebook/LinkedIn sources were skipped today. Public data sources still ran.
Action: Open Chrome with the Solo Agency Local Collector extension enabled, stay logged in, or run the Local Collector app start command again if needed.
```

### OS Startup For Persistent Bridge

If the human wants unattended collection after reboot, the AI agent should prepare or document an OS startup service for the bridge, but the human must approve/run the setup outside the AI sandbox. Do not install or start the service from the AI agent during one-time setup unless the user has explicitly moved beyond setup and asked for OS service automation with full awareness of the local action.

Claude-specific rule:

- Claude often cannot run downloaded binaries from inside its sandbox.
- Claude must not try Claude Chrome Extension as a workaround for automated private collection.
- Claude should provide the human with a one-time shell command or OS-specific setup instructions to start or install the bridge.
- After the bridge is installed as a startup service, Claude can read collector output files and continue reasoning without controlling Chrome directly.

Recommended startup methods:

- macOS: LaunchAgent in `~/Library/LaunchAgents/`.
- Windows: Task Scheduler with "At log on" trigger.
- Linux: `systemd --user` service.

The startup service should run the selected bridge binary with a persistent scheduler config, for example:

```text
solo-agency-local-collector/bin/collector-bridge-darwin-arm64 \
  --host 127.0.0.1 \
  --port 17321 \
  --config-file daily-content-pipeline/collector/collector_config.json \
  --output-dir daily-content-pipeline/collector/inbox \
  --persistent
```

If the bridge is not installed as a startup service, the human must start it manually after reboot by running the prepared setup/start command outside the AI sandbox. The AI agent should not start it from inside the AI sandbox during setup or repair.
