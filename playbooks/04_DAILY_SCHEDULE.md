# Daily Schedule

Stage: `04`

## Load Rule

Load during one-time setup after the Client Intelligence Profile, public data source plan, and private data source status are known, so the routine can be configured before the first agency run. Also load during scheduled runs and whenever routine/schedule config is reviewed or repaired.

## Hard Gates For This Stage

- During one-time setup, configure schedule/routine before the first agency run, after the basic source plan is known.
- After configuring the routine, ask whether the human wants to run the first agency run immediately.
- Support manual-only, daily, multiple-times-daily, weekly, and environment-specific schedules.
- Scheduled runs must run research, private scans if active, analysis, production-ready drafts, approved video/blog/social asset creation when provider setup and explicit approvals allow it, HTML report, and notification.
- Scheduled runs must load Stage 10 and produce Lead & Competitor Opportunities, or explicitly mark them as not found, not scanned, pending activation, or unavailable.
- Scheduled runs must run published-URL analytics and measurement-learning only when published URLs/metrics exist. On the first run with no published history, mark measurement as `no published URLs yet` instead of pretending it ran.
- Scheduled runs must load the needed playbooks again at run time; they must not rely on memory from setup.
- Every scheduled-run human-facing reply, notification, or report handoff must include an updated progress block. If the agent sends multiple progress updates during the scheduled run, each update must show the current completed/current/remaining state.
- If private collection is blocked, continue public data sources and notify the human. Do not fall back to Claude in Chrome, Codex/browser tools, Playwright/Puppeteer/Selenium, or another agent-controlled browser for logged-in/private data sources.
- Store schedule config and notification channel.

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

Use this title:

```text
Solo Agency daily run progress
```

The agent may use a compact form in notifications, but it must not send only a report link or summary while steps remain.

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
   7. If private data sources are configured but not yet activated, do not attempt private collection during this run. Do not use Claude in Chrome, Codex/browser tools, Playwright/Puppeteer/Selenium, or another agent-controlled browser as a fallback. Mark them as `pending_private_activation`, include the activation CTA in the report, and continue with public data sources.
   7a. If no private data sources are configured, and discovery was never offered or was postponed, do not block the scheduled run. Continue with public data sources, but include `Private Data Source Discovery Recommended` or `Private Data Source Discovery Declined/Postponed` in the report/notification. Explain that public-only runs can still produce useful ideas but may miss community, lead, and competitor signals from logged-in/member spaces.
   8. If private data sources are activated, connect to the already-running Local Collector app according to `collector_config.run_mode`.
   9. If private data sources are activated, check and update `daily-content-pipeline/collector/collector_setup_status.md` before deciding whether private collection is available.
   10. Check private collector health through `GET http://127.0.0.1:17321/status` when the Local Collector app is expected to be running.
      - If the bridge is offline, do not start it from inside the AI sandbox. Prepare an absolute-path human-run start command, mark private collection as unavailable for this run, and continue with public data sources.
      - If the bridge is online but `extension_health.status` is `stale` or `no_extension_check_yet` after the 75-second extension check grace window, mark private collection as unavailable for this run and notify the human.
      - If `extension_health.status` is `recent`, continue private collection.
   11. Prepare the private data source queue if private data sources are available and collector health is acceptable:
      - keep the active daily queue around 20 sources or fewer per client by default;
      - prioritize sources most relevant to the client, target audience, target location, pain points, and content pillars;
      - classify extra sources as `weekly` or `optional` and rotate them across future runs;
      - do not run aggressive or parallel private data source scans for the same logged-in account.
   12. Check private data sources if available, using the Solo Agency Local Collector extension plus the Local Collector app when available, with `collector_config.scroll_delay_seconds` defaulting to 5 seconds and `collector_config.max_scrolls_per_source` defaulting to 5.
   13. If the collector bridge was started in `agent_on_demand` mode, stop it after collection completes or after timeout.
   14. Log skipped, pending-activation, expired, rate-limited, warning-triggered, collector-unavailable, extension-unavailable, Chrome-not-running, stale-extension, bridge-offline, or unavailable private data sources.
   15. Load yesterday's private data for this client when available and filter duplicate or near-duplicate data points using visible text matching. Do not parse private-platform HTML for duplicate detection.
   16. Extract relevant `[data_points]`, including reference URLs for every data point. Keep data points that are directly about the primary industry or clearly connected through a related industry. Discard related-industry data when the bridge back to the client's offer is weak.
   17. Add newly recommended private groups/pages/profiles/communities to `New Private Data Sources Detected` and `history/YYYY-MM/new_private_sources_log.md`.
   18. Load Stage 10 and detect hot/warm/watch leads plus direct, indirect, adjacent, attention, and authority competitors during the same research/private-scan pass. The first lead/competitor pass for a client/source set should use 10 scrolls per approved private data source when safe; normal daily runs use 5 scrolls per approved private data source by default.
   19. For every useful lead or competitor opportunity, preserve profile URLs and post/current URLs when available, safe context summaries, reasoning, suggested human action, and a copy-ready value-first comment in the same language as the post.
   20. Generate the 3x2 idea matrix, labeling each idea as `primary_industry` or `related_industry`.
   21. Check `history/YYYY-MM/content_log.md`, including the recent primary/related ratio.
   22. Select the best idea of the day.
   23. Write the configured WideCast-writing-skill draft using the writing skill fallback if MCP/account is unavailable.
   24. If a production provider is connected and the human has explicitly approved creation/rendering/publishing for a selected draft, load Stage 3 and create the approved video/blog/social asset according to provider approval gates. If approval or provider setup is missing, keep the asset as `approval_required` or `provider_setup_required`.
   25. Save `outputs/YYYY-MM/YYYY-MM-DD.md` as the canonical source-of-truth report.
   26. Generate `outputs/YYYY-MM/YYYY-MM-DD.html` as a polished standalone human-facing report. It must be factually aligned with the Markdown report, mobile-friendly, and include editable draft review blocks when drafts exist.
   27. Update or copy `outputs/latest.md`.
   28. Update or copy `outputs/latest.html`.
   29. Update `history/YYYY-MM/content_log.md`.
   30. Update `history/YYYY-MM/data_sources_log.md`.
   31. Update `history/YYYY-MM/lead_log.md`.
   32. Update `history/YYYY-MM/competitor_log.md`.
   33. Update `history/YYYY-MM/lead_competitor_opportunities.jsonl` when possible.
4. Create or update `outputs/YYYY-MM/YYYY-MM-DD_master_digest.md`.
5. Generate `outputs/YYYY-MM/YYYY-MM-DD_master_digest.html` as a polished standalone human-facing master report.
6. Update or copy `outputs/latest_master_digest.md`.
7. Update or copy `outputs/latest_master_digest.html`.
8. Present the daily digest to the human.
9. Prepare a report-delivery record containing the local `.html` report path, upload attempt status, uploaded report URL if available, notification channel, and final notification report link.
10. If WideCast MCP notification/Telegram capability is available, inspect whether an HTML-capable WideCast report/file/asset upload API is available. If it is, upload the HTML report to WideCast first, then send a notification to the human that includes the uploaded WideCast report URL, agent identity, run status, clients processed, blockers, lead/competitor counts, and required actions.
11. If WideCast notification is available but HTML upload is unavailable or fails, log the exact upload blocker and still send a WideCast notification that includes the best available local/hosted `.html` report path/link.
12. If another authorized channel can send the HTML file or link more conveniently only because WideCast notification tooling is unavailable or blocked, use it.
13. Log the upload attempt and notification attempt in `notifications/notification_log.md`.

The daily run is complete only when every active client is processed or explicitly logged as skipped.

When presenting the daily idea list to the human, include reference URLs next to data points, top ideas, and the selected best idea so the human can verify the information. For private data, include the captured source URL and note that it may require the human's logged-in session.

Scheduled runs must assume the human may not be present in the AI agent UI. The run is not fully operationally complete until the mobile-friendly HTML result or a result-ready notification with the HTML path/link has been sent through the configured notification channel, preferably WideCast MCP / Telegram.

If WideCast notification/Telegram is connected and WideCast report upload supports HTML, the notification link must be the uploaded WideCast report URL, not only a local file path. If upload fails or the current wrapper does not support HTML upload, log the blocker and send the best available HTML path/link.

A notification that only says the report is ready but contains no HTML report URL/path is invalid. If this happens, immediately send a correction notification with the HTML report URL/path and log the correction.

---

- After the first report, ask the human whether they want daily, multiple-times-daily, weekly, manual-only, or another cadence.
- Then write or update `schedule.md` and the relevant automation/config files.

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

1. Try `GET http://127.0.0.1:17321/status`.
2. If the request succeeds:
   - record `bridge_status: running`,
   - record `status.persistent`,
   - record `status.job_available`,
   - record `status.output_dir`,
   - record `status.counts`,
   - inspect `status.extension_health`.
3. If `extension_health.status` is `recent`, private collection infrastructure is currently healthy.
4. If `extension_health.status` is `no_extension_check_yet` immediately after extension install, bridge restart, or settings save, wait and re-check for up to 75 seconds before declaring private collection unavailable.
5. If `extension_health.status` is `stale` or `no_extension_check_yet` after the 75-second grace window, treat private collection as unavailable for now and identify likely causes:
   - Chrome is closed,
   - extension is not installed,
   - extension is disabled or removed,
   - Solo Agency Local Collector extension and Local Collector app URL/port mismatch,
   - Chrome service worker is asleep and has not woken recently,
   - browser profile is not the one where the extension was installed.
6. If `/status` fails:
   - record `bridge_status: offline`,
   - do not try to start the bridge from inside the AI agent sandbox during setup/repair,
   - provide the human with the absolute-path Local Collector app setup/start command,
   - continue with public data sources and previously collected private data.
6. If the bridge is running but the extension is stale, do not keep retrying aggressively. Continue with public data sources, log the private data source blocker, and notify the human.
7. If the extension is recent but a private data source fails due to login/captcha/checkpoint/session expiry, skip that source, log the platform-specific issue, and notify the human.

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
