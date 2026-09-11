# Daily Schedule

Stage: `04`

## Load Rule

Load during one-time setup after the Client Intelligence Profile, public data source plan, and private data source status are known, so the routine and client-specific automation task can be configured before the first report. Also load during scheduled runs and whenever routine/schedule config is reviewed or repaired.

## Hard Gates For This Stage

- During one-time setup, configure schedule/routine and the client-specific automation task after the basic source plan is known.
- After configuring the routine, do not run the first report in Setup Flow; verify/resync the client-specific automation task and then START IT — the agent dispatches the task through the scheduler's own run-now, tells the human it has been dispatched — the first run reads all of this client's sources, default and custom (and, only when Facebook, Instagram, or X are not yet connected, that lead counts will be lower until they are) — and where the result will appear, arms one background wait where the runtime supports it, and reports back in the same chat when it lands; only a runtime that genuinely cannot start its own tasks falls back to naming the task for the human to run.
- If the human asks to run, create, generate, show, refresh, or update a report during Setup Flow, do not run it in the setup chat and do not ask whether to run it now. Treat the request as a handoff request: verify/resync the client-specific automation task and then START IT — the agent dispatches the task through the scheduler's own run-now, tells the human it has been dispatched — the first run reads all of this client's sources, default and custom (and, only when Facebook, Instagram, or X are not yet connected, that lead counts will be lower until they are) — and where the result will appear, arms one background wait where the runtime supports it, and reports back in the same chat when it lands; only a runtime that genuinely cannot start its own tasks falls back to naming the task for the human to run.
- Support manual-only, daily, multiple-times-daily, weekly, and environment-specific schedules.
- Scheduled runs must run research, private scans if active, analysis, production-ready draft options, final WideCast video-script skill pass before any video provider request, approved video/blog/social asset creation when provider setup and explicit approvals allow it, HTML report, and notification. Before report HTML/PDF work, scheduled runs must load `playbooks/skills/report-design/SKILL.md` and use `tools/solo_tool render-report` by default instead of writing ad hoc report/PDF scripts. Before any video provider request, scheduled runs must load and apply the existing WideCast video script-writing skill, treat report scripts as reference only, and if a report version/code is already selected or recommended, produce only that one final production script/brief with research and inline-media/direct-image/video-URL workflow. Generate a new five-version set only when no report version has been selected or recommended yet. Use only the skill-produced final artifact as the provider payload. If video provider setup is missing or blocked, scheduled runs must still save the final WideCast-grade script/storyboard/production-brief output and ask for PDNA setup; they must not create local video media with `ffmpeg`, Pillow, `moviepy`, Remotion, browser/canvas screenshots, slideshow export, or similar tools.
- Scheduled runs must load Stage 10 and produce Lead & Competitor Opportunities, or explicitly mark them as not found, not scanned, pending activation, or unavailable.
- Scheduled runs must run published-URL analytics and measurement-learning only when published URLs/metrics exist. On the first run with no published history, record `measurement_status: no_published_urls_yet` instead of pretending it ran.
- Scheduled runs must load the needed playbooks again at run time; they must not rely on memory from setup.
- Every scheduled-run human-facing reply or report handoff must include an updated progress block. If the agent sends multiple progress updates during the scheduled run, each update must show the current completed/current/remaining state. The provider notification is read by the CLIENT and never carries progress blocks (Client Notification Contract, playbook 03).
- If private collection is blocked, continue public data sources and notify the human. Do not fall back to Claude in Chrome, Codex/browser tools, Playwright/Puppeteer/Selenium, or another agent-controlled browser for private data sources.
- Store schedule config and notification channel.
- After any human-approved change made after the schedule/automation was created, perform Automation Resync before claiming the next scheduled run is updated.
- Never say "the automation is updated" if only `collector_config.json`, only `schedule.md`, or only the Client Intelligence Profile was changed. The whole automation package must be synced or the remaining snapshot/update blocker must be stated.
- Every schedule/routine question, native automation task creation/update instruction, scheduled prompt paste/replace instruction, report-request hard-stop handoff, and automation freshness blocker that needs the human must use the root playbook `**[ACTION REQUIRED]**` block.

## Latest Override: Client-Specific Automation Tasks

The current Solo Agency model uses separate Setup Flow and Automation Flow.

Setup Flow must create/update schedule and automation tasks, but must not run the first report directly. The first real report must be executed by a client-specific automation task.

Each client-specific automation task runs once for all of this client's sources — default and custom together — and produces one report; there is no separate run or report per source.

Rules:

- Create one client-specific automation task per active client by default.
- That task is a team member: the client's Scout + Creator + Analyst agent (`playbooks/TEAM_MODEL.md`). Record `Role` and `Brain` for it in the Team Roster of `automation_manifest.md`; the task name itself stays `{Client} - Solo Agency Daily Run`.
- Every client-specific task name must begin with the client name, for example `AvenNgo - Solo Agency Daily Run`.
- The task prompt must pin `target_client_slug` and must not process other clients.
- The task may use the shared Local Collector app/bridge, but private data source jobs must be routed by `client_slug` and bound to the claiming `extension_instance_id` when present.
- If the AI automation environment cannot call `127.0.0.1`, it must use file-based job requests under `daily-content-pipeline/collector/jobs/pending/` and read bridge/extension health from local files.
- With one shared Local Collector app/bridge, private data source collection is parallel across different client Chrome profiles/extensions. Multiple scheduled agents may enqueue jobs at the same time; the bridge should expose a separate active collector job per `client_slug`, bind it to the claiming extension instance, route each run to its own output folder, and serialize only jobs for the same client/profile after `/complete` or TTL expiry.
- The task prompt must require one canonical combined client-facing report per client/day/run: `{client-name}-client-report.html`, built from the three scrubbed staging lane files:
  - `{client-name}-public-data-sources-report.html`
  - `{client-name}-private-data-sources-report.html`
  - `{client-name}-daily-report.html`
  The combined `{client-name}-client-report.html` is the output built from those three staging files and is the default handoff link; the staging files are not the handoff.
- If private data sources run after public data sources, the task must create/update only the private report and daily report, then rebuild the combined client report. It must not overwrite, regenerate, or summarize away the public report.
- Time-slot collision rule: before creating or re-timing ANY client automation task, run `tools/solo_tool schedule-slots --pipeline {setup-root}/daily-content-pipeline suggest --task "{task name}" --client {client_slug} --time "HH:MM" --cadence-hours {H}` (or `--cadence monthly`) with the human's preferred start time. With mixed cadences (daily/48h/72h/96h/weekly/monthly) a clock-time compare is meaningless — the tool projects every registered task's future occurrences and pushes the start time in steps (default 15 min) until no occurrence lands in a slot already holding the max (default 10 running tasks). Use the returned `suggested_time`/`suggested_anchor_date` for the native task; when it differs from the requested time, tell the human in ONE sentence why ("09:00 already has 10 tasks running on the days this task fires; scheduled 09:30 instead") — never ask permission for the shift, the preferred time was already given and the shift is a technical adjustment. After the native task is created or updated, `... schedule-slots register --task ... --client ... --time ... --cadence-hours ...`; after deleting a task run `remove`, after pausing re-`register --status paused`. The per-slot limit, push step, projection horizon, and the operator's contact email live in `{setup-root}/daily-content-pipeline/system_settings.json`, editable by the operator in the web UI at `/ui/settings` — agents read these settings, never overwrite them.
- **The one case where the check cannot run: the first task, before the bridge exists.** `tool schedule-slots` is a subcommand of the collector bridge, and Setup Flow forbids installing `solo-agency-collector/` until the private-data-source stage (`AGENTS.md`, collector gate) — which the roadmap places AFTER the first automation task exists. On a fresh install that is a genuine deadlock, and the way out is not to install the bridge early, nor to hand-edit `task_slots.json`, nor to stall setup. Create THAT ONE task at the human's requested time, say so in one sentence ("I could not run the slot check yet, the tool arrives with the collector; I will register this time as soon as it does"), and record the debt in `daily-content-pipeline/automation/automation_manifest.md` as `slot_check_pending: true` with the task name, client and time. The FIRST action after the bridge is installed — before any other `suggest` — is `schedule-slots register` for that task, then clear the flag. Nothing is lost by waiting: on a fresh install `task_slots.json` is empty, so there is nothing for a first task to collide with, and the rule below already requires backfilling a registry that predates it. This exception covers exactly one task on one install; a second client, a re-timing, or any task created once the bridge exists is bound by the rule above with no exception.
- Slot-registry backfill: an install whose automation tasks predate `automation/task_slots.json` must register every existing task (name, client, cadence, time) at the next Automation Resync, before any new `suggest` — an empty registry makes every suggestion blind.
- A setup/config session starts `AvenNgo - Solo Agency First Run` itself and says so; it must not generate the report inside the setup chat. Only a runtime that cannot start its own tasks instructs the human to run it.
- A setup/config session must not load `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` as a workaround for a human report request. The scheduled entrypoint belongs in the native automation task or a separate Automation Flow run, not inside Setup Flow.
- Automation Flow may accept config changes during a real run, but must immediately perform Automation Resync before claiming future runs are current.

For multi-client daily operations, prefer separate client tasks plus an optional master digest task. The master digest task must not scan private data sources; it only reads existing client reports/outputs and summarizes them.

### Unattended runs on Claude Code desktop (permissions)

Verified 2026-09-10: a scheduled run on Claude Code desktop starts in permission mode `default` and stops at its first Bash/Edit call until a human clicks Approve. `allowed-tools` in the task's SKILL.md and `permissions.defaultMode` in the project's `.claude/settings.local.json` do nothing for a headless run; project-level `.claude/settings.json` allow rules are not applied either (workspace-trust gate). Only a USER-level allow rule in `~/.claude/settings.json` that fully matches the command lets the run pass with no click (verified, ~3s).

So, at setup step 6 — right after the client-specific automation task is created, on a LOCAL Claude Code runtime only — Sam asks ONE consent, in the Boss's language, for example:

```text
Để các lượt chạy tự động không dừng lại hỏi quyền từng lệnh, tôi sẽ ghi vào cài đặt Claude của bạn quyền chạy bridge Solo Agency, gọi bridge trên máy bạn và ghi file trong thư mục cài đặt này (chỉ trong thư mục này). Đồng ý?
```

On yes, edit `~/.claude/settings.json` (create it if missing; back it up first as `settings_YYYY-MM-DD_HH-MM-SS.json`; merge into `permissions.allow` without removing any existing entry) with EXACTLY this rule set, substituting the install root `R`, bridge port `P`, the resolved bridge binary file name `B` (the actual os-arch binary present under `R/solo-agency-local-collector/bin/` on this machine), and the source checkout `S = R/solo-agency`:

```json
{
  "permissions": {
    "allow": [
      "Bash(R/solo-agency-local-collector/bin/B *)",
      "Bash(S/tools/solo_tool *)",
      "Bash(tools/solo_tool *)",
      "Bash(bash S/tools/wait_for_run *)",
      "Bash(curl -s --max-time * http://127.0.0.1:P/*)",
      "Read(R/**)",
      "Edit(R/daily-content-pipeline/**)",
      "Edit(R/extensions/**)"
    ]
  }
}
```

Both `solo_tool` rules are needed: every call site in `playbooks/SCHEDULED_RUN_ENTRYPOINT.md`, this file's own Daily Run Algorithm, and `playbooks/02_PRIVATE_SOURCE_SETUP.md` types the bare relative form `tools/solo_tool ...` (the scheduled run's working directory is the source checkout `S`), not the `S/`-prefixed absolute form — Claude Code's Bash allow-glob matches the literal command text typed, not a resolved path, so only the bare-form rule actually matches those calls. Keep the `S/`-prefixed rule too, for any call site that does type the absolute form.

For the GitHub Update Watch task only, also add these to the same `permissions.allow` array:

```json
"Bash(mktemp -d)",
"Bash(git clone https://github.com/soloagency/solo-agency *)",
"Bash(git -C * remote get-url origin)",
"Bash(git -C * rev-parse HEAD)",
"Bash(git -C * ls-remote origin refs/heads/main)",
"Bash(git -C * status --porcelain)"
```

Write rules are not consulted by Claude Code — `Edit` alone covers both create and modify. Record the outcome on `automation_manifest.md` (schema in `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`): `unattended_permissions: granted | declined | not_applicable`, `unattended_permissions_scope: user_settings`, `unattended_permissions_written_at`.

If the Boss declines, Sam says in one sentence that the first run will pause once in its own session for an "Always allow" click, and continues — do not ask again this session.

On Codex or any remote runtime, record `unattended_permissions: not_applicable` — Codex approvals follow Codex's own model; its automations have run unattended on the live install since 2026-07.

### Run-now per runtime

**Claude Code desktop.** The agent has no run-now tool for a scheduled task. Verified 2026-09-10: a ONE-TIME scheduled task (`fireAt` = now + 2 minutes) fires about 40 seconds after `fireAt`, runs in its own fresh session, and auto-disables. So the first run on Claude Code desktop is: create a one-time task named `{Client} - Solo Agency First Run` (`taskId`: `{client_slug}-solo-agency-first-run`) through the runtime's scheduled-task tool (`create_scheduled_task` with that `fireAt`, `notifyOnCompletion: false` — completion notifications never reached the chat in 4 tests), using the SAME prompt body as the daily task (read `daily-content-pipeline/automation/scheduled_run_prompt.md` in full; `target_client_slug` pinned). Record `first_run_task_id` (the `taskId`) and `first_run_dispatched_at` (the dispatch timestamp, ISO 8601) on `automation_manifest.md` at the moment of dispatch. Delete the one-time task after the result is reported.

**Codex desktop.** The agent triggers the automation's own run-now itself (owner-confirmed 2026-09-10: the Codex agent can start an automation immediately), then arms the wait exactly as on Claude. The `**[ACTION REQUIRED]**` block naming the task is only the fallback for a genuine failure to start.

**Other scheduler types** keep the existing run-now table below.

### Wait and report

Right after dispatch, arm ONE background wait — never a foreground sleep, never a poll loop in the chat:

- On Claude Code (desktop or CLI), run `bash R/solo-agency/tools/wait_for_run <client_slug> <dispatched_at_iso> --timeout 2700` with the runtime's run-in-background option; the session is re-invoked when it exits (verified 5/5). The helper exits 0 when a new `standup.jsonl` line for that client with `ts > dispatched_at` appears (it prints that line), exit 3 on timeout. The moment the wait is armed, record `first_run_wait: armed` on `automation_manifest.md`.
- On wake, Sam speaks the First-Run Report (below) in the SAME chat, then records `first_run_wait: reported` and `first_run_reported_at` (ISO 8601, now) on `automation_manifest.md`.
- On a runtime with no background execution: `first_run_wait` stays `not_available` (no helper was armed); Sam states the expected duration (10-15 minutes for a first run) and the exact phrase to send ("xong chưa?" / "is it done?"), plus the Telegram/email channel if configured, and the Reply Frame delivers the report on the next turn — recording `first_run_wait: reported` and `first_run_reported_at` once that report is actually spoken.
- On timeout (exit 3): record `first_run_wait: timed_out`. Sam says the run has not finished, names the two usual causes (a permission prompt waiting in the run's own session in the Scheduled panel → "Always allow" once; the extension not connected), and offers to check again — a later successful report still records `first_run_wait: reported` and `first_run_reported_at`.

**First-Run Report** (spoken by Sam, in the Boss's language, ≤ 8 lines): when it finished; leads found (hot/warm/watch counts from the standup line / report_state) and the locked-contacts meter line; what needs the Boss (`needs_boss` items, ≤ 3); the report opened beside the chat per the Answer-and-Show Rule (`/ui/{client}/reports?open=latest`) with the link printed; then the normal next-job offers and one question. If Facebook/Instagram/X were not connected, the awareness line ("... are off, so lead counts are lower") stays.

### Command shapes (so unattended runs never pause)

For the allow rules above to match, every shell command in a scheduled run must be one of: `<bridge binary> tool <family> ...`, `tools/solo_tool <family> ...` (run from `S`, the form the playbooks actually type) or `S/tools/solo_tool <family> ...`, `bash S/tools/wait_for_run ...`, or `curl -s --max-time <n> http://127.0.0.1:P/<path> [-X POST -H ... -d ...]` with the URL immediately after the fixed flags. Never `python3`, `node`, `bash -c`, `sh -c`, pipes, `&&` chains, `xargs`, `find -exec`, or `sed -i`. File writes only under `R/daily-content-pipeline` and `R/extensions`. Anything else means the run pauses on a permission prompt.

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
3b. Always load `playbooks/TEAM_MODEL.md` and `playbooks/NEXT_JOB_CATALOGUE.md`, before the first
    Boss-facing reply of the run — the STATE POLL, CATALOGUE, and meter they define govern every
    reply's next-jobs/`**[ACTION REQUIRED]**` close.
4. Load Stage 1 only if a profile is missing, incomplete, stale, or needs setup repair, or this is the first Automation Flow agency run/report for the client (Stage 1 holds the first-report contract). Do not ask setup questions when the saved profile is complete.
5. Load `playbooks/PRIVATE_SOURCE_GATE.md`, Stage 2, Stage 8, and Stage 9 when private data sources are active, pending, blocked, or being scanned.
6. Load Stage 3 when drafts, production, publishing, provider setup, or notification provider actions are needed.
7. Before any video provider creation request, load `playbooks/skills/video-script-writing/SKILL.md` and its required modules through the verified client provider when available or repo-local/static fallback when PDNA is missing. If a report version/code is already selected or recommended, apply that existing skill only to that selected version/code and continue into Stage 2 visual treatment/final handoff. Generate five options only when no version is selected or recommended. Do not edit/reimplement the skill and do not send report drafts unchanged.
8. Load Stage 3A: `playbooks/SOLO_AGENCY_VIDEO_PROVIDER_ADAPTER.md` after any vendored writing/provider skill when provider or video actions are relevant to the run.
9. Load Stage 3B: `playbooks/skills/video-editing/SKILL.md` after provider video creation returns reviewable scenes or when editing a provider video.
10. Load Stage 5 when any published content exists or when run-window/last-7-day measurement is due. Load `playbooks/ACCOUNTABILITY_POSTING.md` alongside it (posting-gap check + fleet snapshot).
11. Load Stage 6 and then `playbooks/skills/report-design/SKILL.md` whenever generating, reviewing, fixing, or packaging the human-facing HTML/PDF report.
12. Load Stage 10 whenever lead/competitor opportunities, comments, opportunity logs, or competitor monitoring are part of the run. This is normally every first run and every scheduled daily run.
13. Load Stage 11 when the task is `Solo Agency - GitHub Update Watch`, when an update/upgrade/sync-latest request is being handled, or when blocker recovery checks GitHub for a newer Solo Agency version.
14. Load Stage 9 before claiming the scheduled run is complete.

Every load in this contract requires a LOAD LEDGER entry per `playbooks/LOAD_LEDGER_PROTOCOL.md`, checked against `playbooks/LOAD_MANIFEST.md`.

The difference between first setup and scheduled runs:

- First setup asks only the minimum setup questions because the profile does not exist yet.
- Scheduled runs read the saved Client Intelligence Profile, source lists, collector config, content history, publishing ledger, and analytics logs, then continue automatically.
- Scheduled runs must not re-ask industry, sub-industry, audience, pain points, content pillars, or private data source setup questions if those fields are already present.
- Scheduled runs may ask the human only when an approval gate, blocker, missing critical field, expired private session, production/render/publish/credit decision, or lead outreach decision requires human input.

Scheduled run completion requires the same end-to-end path as a manual daily run: public research, private scans if active, published-URL analytics when published content exists, data analysis, Lead & Competitor Opportunities, idea matrix, best idea, production-ready drafts, approved video/blog/social production when authorized, Stage 6 plus report-design-rendered HTML/PDF report, the Top 3 ideas queued into the provider production plan when the verified client provider exposes that operation (three, each deduped, never a blocker when the operation is missing), the content-plan `magic_url` minted when the client-link operation is available, notification, and measurement/learning when measurement data exists.

## Scheduled Run Progress Display Contract

Scheduled runs are meant to be automatic, but the human still needs visible state whenever the agent speaks.

Every scheduled-run reply or report handoff must include:

- completed steps;
- current active step;
- remaining steps;
- blockers or human decisions required;
- whether published-URL analytics was run or skipped because no published URLs/metrics exist yet.
- an `Automation freshness check` stating whether the latest changes are synced into the configured automation/scheduled task and whether tomorrow's run will read the current contracts/prompts/playbooks/source approvals/state, not only the latest config file.
- when no human decision is required: a "next jobs" block of 2-3 offers chosen, by the STATE POLL, from `playbooks/NEXT_JOB_CATALOGUE.md` (`playbooks/SCHEDULED_RUN_ENTRYPOINT.md` step 16 carries the poll's full field list), plus exactly one closing question — the reply never ends flat.

If this run touched the CRM (any capture, discovery pass, or reconciliation that could change contact counts), read `tool crm-store ... contact lock-status` once at the end of the run, before composing this reply, and write its `locked` count into this run's report state as `leads_locked`. Carry the resulting meter line (`playbooks/10_LEAD_COMPETITOR_DETECTION.md`, "Capture never stops at the plan's contact cap") in this reply and in `INTERNAL_REPORT` whenever `locked > 0` or the approaching-cap condition is met.

Use this title:

```text
Solo Agency daily run progress
```

This progress state is operator material: it appears in the run reply/report handoff, never in the client notification (Client Notification Contract, playbook 03). The run must not end with only a report link or summary while steps remain.

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
- PDNA, production provider, WideCast API key/OpenAPI config, Telegram/email fallback, publishing, analytics, published URL history, or notification delivery changed;
- schedule cadence, timezone, active clients, manual-only mode, or report delivery channel changed;
- the playbook behavior changed in a way scheduled runs must follow;
- Solo Agency update/version-watch state changed, an upstream update was applied, the update-watch task was created/changed, bridge rerun is required, or extension reload is required.

Automation Resync requires updating every relevant layer:

1. Client Intelligence Profile: current source status, approvals, profile fields, private monitoring activation, PDNA, analytics, and notification status.
2. Source/history logs: discovery results, approved/rejected/pending private data sources, new public data sources, keyword bank changes, and approval timestamps.
3. `daily-content-pipeline/provider_defaults.json`: provider catalog/discovery URL defaults, no secrets.
4. The client provider files under `integrations/providers/`: provider config, OpenAPI cache/capability snapshot, provider health, and provider call log when relevant.
5. `daily-content-pipeline/schedule.md`: cadence, included clients, notification channel, private data source status, PDNA/provider status, and last resync timestamp.
6. `daily-content-pipeline/collector/collector_config.json` or `POST http://127.0.0.1:17321/config`: only when private data source collection schedule/sources/scan depth changed.
7. `daily-content-pipeline/automation/automation_manifest.md`: current run contract, paths, active clients, prompt source, config source, provider config source, and last known state hash/summary.
8. `daily-content-pipeline/automation/scheduled_run_prompt.md`: the exact prompt that the native AI automation/scheduled task should run.
9. Native AI automation or scheduled task body: update it when the environment stores a separate prompt snapshot.
10. `daily-content-pipeline/automation/update_state.json` and `update_log.md`: update-watch state, checked/applied commits, change classification, and human actions required.
11. `daily-content-pipeline/automation/resync_log.md`: what changed, what files/tasks were updated, what could not be updated, and what the next scheduled run should see.

If the native AI automation task cannot be edited by the agent, the agent must:

- write the exact replacement prompt to `daily-content-pipeline/automation/scheduled_run_prompt.md`;
- mark `automation_prompt_update_pending` in `automation_manifest.md` and `schedule.md`;
- give the human one concrete instruction to paste/replace the scheduled task prompt in a `**[ACTION REQUIRED]**` block;
- avoid claiming the scheduled run is fully updated until the human confirms the native task body was updated.

Automation Resync verification:

Before saying a post-schedule change is complete, do a dry-read as if tomorrow's scheduled run were starting:

1. Read `playbooks/SCHEDULED_RUN_ENTRYPOINT.md`.
2. Read `daily-content-pipeline/automation/automation_manifest.md`.
3. Read `daily-content-pipeline/provider_defaults.json` when present.
4. Read `daily-content-pipeline/schedule.md`.
5. Read each active Client Intelligence Profile.
6. Read each relevant client's provider config/capability files when PDNA, notification, analytics, publishing, report delivery, or production was changed.
7. Read `collector_config.json` when private data sources are active or pending.
8. Read `daily-content-pipeline/automation/update_state.json` when update/version-watch or a GitHub-applied change affects future runs.
9. Confirm the latest user-approved changes are visible from those files and from the scheduled prompt/task body.

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

## Scheduling Rule

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

The agent must also record the notification channel in `schedule.md`. If the client has verified WideCast OpenAPI config and the discovered spec exposes `sendNotification`, record WideCast email+Telegram as the preferred notification channel for scheduled runs, even if Telegram is not connected yet, because WideCast can fall back to email when the account supports it. If WideCast OpenAPI notification is unavailable but Gmail/email is connected, record Gmail/email as the secondary fallback notification channel. If neither is available, record `notification_channel: local_path_only` and tell the human how to connect WideCast API key + Telegram/email fallback or Gmail/email. The notification channel is governed by the Provider Consent & Mandatory Notify Rule (outreach AUTOMATION_SCHEDULING carries the full text, and it applies to content runs identically): configuring WideCast IS the operator's consent to upload reports and send notifications to their own account; the only privacy gates are the operator-secrets red line, the client-facing scrub, and the notification copy rules; "the report contains private-source research" is never a valid blocker. If an upload is refused for any reason (including a runtime/sandbox safety layer), STILL send `sendNotification` when a client-openable hosted link exists (an earlier upload or the minted magic link), composed per the Client Notification Contract (playbook 03) — never with local machine paths; when no hosted link exists at all, record the delivery blocker with the local paths in the run output and INTERNAL_REPORT instead of notifying the client. Only a refused `sendNotification` attempt itself may end as `notification_blocked_by_environment` + one `**[ACTION REQUIRED]**` with the exact outside-sandbox command. A run whose completion was never surfaced (no client notification when a hosted link existed, and no run-output record or environment-block recorded) is incomplete.

Scheduled runs should be designed as unattended runs. The human may not be watching the AI agent UI, so the agent must proactively notify the human when the run finishes or when human action is required.

## GitHub Update Watch Scheduling Rule

After the first schedule/automation has been configured, set up a separate maintenance automation - create the native task or write its pending prompt AND hand it to the human in an `**[ACTION REQUIRED]**` block (never silently skip):

```text
Solo Agency - GitHub Update Watch
```

This task exists because Solo Agency is updated frequently and older playbooks/code may be the cause of tomorrow's blocker.

Rules:

- The task should run daily, preferably before client daily runs.
- It must load `playbooks/11_UPDATE_AND_VERSION_WATCH.md`.
- It must check GitHub `main`, compare the installed version, classify the change, and update `daily-content-pipeline/automation/update_state.json` plus `update_log.md`.
- It must not run client reports, public data source scans, private data source scans, production, publishing, or analytics.
- It must not send Telegram, WideCast/email-fallback, provider notifications, social posts, or client notifications. GitHub update checks are internal user/agency maintenance; write `daily-content-pipeline/automation/update_notice.md` and surface the result in the setup/maintenance chat or native task output instead.
- It may auto-apply updates only when the human has approved auto-apply in `update_state.json` or an equivalent operator setting.
- Even when auto-apply is approved, bridge/runtime changes still require a human-run command outside the AI sandbox and extension changes still require Chrome reload/Load unpacked steps per client profile.
- If the automation environment cannot create the native task directly, write the exact prompt from `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` to `daily-content-pipeline/automation/update_watch_prompt.md`, log `update_watch_task_prompt_pending`, and give the human the exact task name and prompt path.

If no automation is available:

1. Explain the limitation.
2. Create manual run instructions.
3. Provide the exact command or prompt the human should use each day.

Example manual run prompt:

```md
Run the daily content pipeline for every active client in clients_index.md. Produce today's outputs and master digest.
```

---

## Run Locking And Notification Dedup

Scheduled runs can overlap (a previous run still finishing, a manual re-run, or a master/all-clients task and a client-specific task both touching one client). Protect against duplicate work and duplicate notifications:

- Before starting a client's daily run, create or check `outputs/YYYY-MM/YYYY-MM-DD/{client-name}-run_lock.json` (`started_at`, task name, session hint, `held_by_brain`, `held_by_session`). If a fresh lock exists (younger than about 3 hours), do not start a duplicate run for that client — log it and stop. A stale lock (older than the window, or from a run that clearly died) may be taken over, with a note in the run record. Remove or close the lock on completion.
- `held_by_brain` names the runtime that holds it (`codex`, `claude-code`, `claude-cowork`, ...), never the generic string `agent`. Several brains may operate one install at the same time, so a lock that does not say who holds it is unreadable. When you find a fresh lock held by ANOTHER brain, report it to the human by name and by elapsed time ("Codex started this client's run 12 minutes ago") and stop; do not take it over and do not silently wait.
- The `run_lock` covers a FULL client run. Smaller operator-directed work — enrich one lead, draft one email, render one existing report — takes a scoped work lease instead, so two brains on disjoint scopes do not block each other. See `playbooks/MULTI_BRAIN_OPERATIONS.md`.
- Master/all-clients digest tasks only READ client reports; they never rebuild `{client-name}-client-report.html` or the `outputs/latest/` client files. Only the client's own run rebuilds them.
- Before sending any `public_report_ready` or `private_report_ready` notification, read `notifications/notification_log.md` and `{client-name}-report_state.json` for the same client/day. If an equivalent notification was already sent, do not re-send. A resumed run records `resumed_from` in the report state so retries stay idempotent.

---

## Run Window — Cadence-Aware Time Anchor

Clients run on different cadences (daily, every 48/72/96 hours, weekly, monthly). Every "new
since last time" window in a run is anchored to the PREVIOUS COMPLETED RUN of this client's
task — never to the calendar:

- `previous_run_date` = the date of the most recent completed run for this client, read from
  real run history on disk: the newest dated `outputs/YYYY-MM/YYYY-MM-DD/` folder containing
  this client's `{client-name}-report_state.json`, or the newest canonical
  `outputs/YYYY-MM/YYYY-MM-DD.md` report. Never compute it as `today − 1 day`, and never
  derive it from the cadence configured in `schedule.md` — configured cadence says when runs
  SHOULD happen; history says when one actually DID.
- `run_window` = everything after that previous completed run, up to now. On a daily cadence
  the run_window happens to equal one calendar day; on any other cadence it does not.
- First run ever (no run history on disk): the run_window is unbounded — treat all collected
  data as new and say so in the report.
- Late, missed, or manually re-triggered runs need no special handling: anchoring on real
  history stretches or shrinks the window automatically.

Wherever a playbook says "yesterday", "yesterday's data", or "today's data" about collected
data, published content, or measurement windows, read it as the run_window. On a
multi-day/weekly/monthly cadence, comparing against a literal `today − 1 day` folder finds
nothing and silently treats every old data point as new — that is the failure this anchor
exists to prevent.

## Daily Run Algorithm

For each daily run:

1. Load `clients_index.md`.
2. Identify all clients with `active` status. If the run has a pinned `target_client_slug`, restrict the loop to that client only; a client-named task must not process other clients.
3. For each active client, processed in `clients_index.md` order unless `schedule.md` defines a different priority:
   1. Load the client's Client Intelligence Profile file.
   2. Validate required fields.
   3. If the Client Intelligence Profile is incomplete, enter setup repair mode.
   4. Prepare the current month folder key `YYYY-MM`.
   5. Load saved `public_data_sources` and visit/check active due public data sources before or alongside keyword search.
      - Shared public sources are scanned once and shared: before visiting, run `tools/solo_tool source-registry --pipeline {setup-root}/daily-content-pipeline due --client {client_slug} --kind public` with the source URLs. `reuse` entries: read the client-neutral notes from the returned `data_dir` instead of re-visiting. `scan` entries: visit, write the raw findings (facts, URLs, quotes, dates — no client analysis, no client names) to `daily-content-pipeline/collector/public_pool/{uid_hash}/YYYY-MM-DD.md` using the `uid_hash` from the `due` output, then record it: `tools/solo_tool source-registry --pipeline {setup-root}/daily-content-pipeline record --client {client_slug} --run {run_id} --kind public --data-dir daily-content-pipeline/collector/public_pool/{uid_hash} --url {source_url}`. `wait` entries: use `stale_data_dir` when present, else skip this run. Client-specific filtering always happens per client, on top of the shared notes.
      - Visit sources where `visit_in_scheduled_runs: true` and cadence is due at this run.
      - Prioritize `active_public_source` daily sources, then due `weekly_public_source` sources, then relevant `occasional_public_source` sources when the topic/event matches.
      - Record source status, useful URLs, useful signals, weak/noisy results, and whether the source should stay active, be promoted, or be demoted.
   6. Search the open web with today's plan from the public keyword bank.
      - Ask the bank what to search: `<bridge> tool public-keywords --pipeline {setup-root}/daily-content-pipeline --client {client_slug} plan --recency "{Month YYYY}"`. It returns at least 10 terms — earning terms pinned (at most two), the rest rotated, at least two of them SHORT (1-3 words) — each with a `query` that carries the month and year only for time-bound groups. Search the `query`; everything you record later is keyed by the `term`.
      - If the bridge answers `tool public-keywords` with a usage dump or "unknown tool", the bridge binary predates this playbook: report it as a blocker in the keywords section, tell the human to update the bridge (Stage 11), and do not search this run without a plan.
      - If `plan` answers that the client has no bank: a new client has `public_keywords_seed.jsonl` in its workspace from Setup step 4 — load it with `add --file {that file}` and delete the file; an older install kept the bank inside the profile — run `migrate --profile {client_profile path}` once (terms are imported dateless, old run verdicts folded in). Then plan again. If `plan` WARNS that the bank has no short terms, add a few 1-3 word keywords before searching; a run of ten long questions only confirms yesterday's thinking.
      - `plan --kind web` (the default, no `--kind` flag needed) never returns `community_discovery` terms — those belong to step 11C's Social Discovery Pass via `plan --kind discovery` only. On the first run, `public_keywords_seed.jsonl` may carry `"group":"community_discovery"` lines, which the same `add --file` load above files into the same bank alongside the web terms. `collector/source_keywords_seed/{client_slug}.jsonl`, when Setup wrote it, is merged automatically by `tool source-keywords ... seed` the first time each group's intent-term bank is created — nothing extra to do here.
      - Before each search, run `tools/solo_tool search-pool --pipeline {setup-root}/daily-content-pipeline check --industry {industry} --keyword {query}`. A fresh hit means a same-industry client already ran this search inside the TTL — reuse its results (then apply THIS client's relevance filtering) and count the term as used. On a miss, search normally and `record` the client-neutral results (`{url, title, note}` array) back to the pool.
      - Use keywords in the target audience's likely search/comment language. Do not translate the bank into the human's chat/report language unless the audience uses that language.
      - At least 7 of the run's terms should come from demand groups (pain-point/problem/need/buying-intent/objection/comparison/question/local-context/trend-news). The plan ranks those groups first and `stats` shows `by_group`; if the bank itself is mostly industry_general, fix the bank, not the run. This is the Google / open-web channel of `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`, "Buyer Profile Channel Keyword Table" — the 12 groups stay, but every `buying_intent`/`need_or_goal` term must be anchored to a role or the offer (`marketing help realtor`, not `marketing help`); an unanchored term is a bank defect, not a weak-result problem to fix by adding words.
      - When a query returns nothing or near-nothing, retry it shortened once — drop its rarest constraint — before calling it weak. If the short form works, `add` it as its own term. Most weak verdicts on a long phrase are a narrowness problem, not a topic problem.
      - Continue until at least 3 source-backed candidate ideas are new or newly angled against `history/YYYY-MM/content_log.md`, or until the plan is exhausted; then report the coverage limitation and the terms tried rather than fabricating weak ideas.
      - **Before the run ends, record every term searched, in one command:** `<bridge> tool public-keywords --pipeline {setup-root}/daily-content-pipeline --client {client_slug} record --json '{"{term}":{"verdict":"useful|used|weak|retry_later","urls":N,"ideas":N}}'` — keyed by the saved `term`, not the dated `query`; `urls` is how many useful URLs you kept from it, `ideas` how many ideas it produced. Status, probation, retirement and pins all move from this record; nothing else moves them, and nothing is written into the profile.
      - Extract new keyword candidates from useful search results, public discussions, questions, competitor hooks, comments, and emerging phrases, and add them with `add` (a `mined` term carries `--note` saying what you saw; dateless; the shorter phrasing when two compete). Detect useful recurring public data sources from search results and public pages. Promote strong recurring sources into `public_data_sources` with cadence and status — normalizing each URL first (`tool source-registry normalize`; store only the `clean_url`).
      - Put the plan, the `record` output and `tool public-keywords stats` in the daily report section `Public Search Keywords Used Today`.
      - If no search was possible, explicitly explain the blocker in that same section.
   7. Before deciding whether to skip private data sources, perform Collector Runtime Verification whenever any of these are true:
      - private data sources are active, pending, requested, approved, present in the Client Intelligence Profile, or listed in any source approval/history file;
      - schedule/config says `public_data_sources_only`, `private sources postponed`, or `pending_private_activation`, but the workspace contains Local Collector files;
      - `daily-content-pipeline/collector/inbox/bridge_health.json`, `daily-content-pipeline/collector/inbox/collector_status.json`, `daily-content-pipeline/collector/collector_setup_status.md`, or recent `daily-content-pipeline/collector/inbox/YYYY-MM/*/*/collector_status.json` exists.
      - When this client's `extension_health.status` comes back `stale` or `no_extension_check_yet` past the 75-second grace window, deliver the Login Reminder block (Facebook, Instagram and X) inside its own `**[ACTION REQUIRED]**` block (quoted verbatim in `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` step 12D and `playbooks/SETUP_FLOW_ENTRYPOINT.md`), and record the answer per platform as `facebook_lead_source` / `instagram_lead_source` / `x_lead_source`: `enabled|web_only|pending` (`web_only` only on a clear confirming phrase such as "không dùng Facebook" — never "để sau" or silence, which leave it `pending`). A platform left `web_only` is skipped by step 11C's Social Discovery Pass this run for that platform, and every report this run produces must carry that platform in the persistent web-only awareness line (`playbooks/06_AGENCY_REPORT_STANDARD.md`) instead of a bare lead-count note. A platform still `pending` is also skipped this run for that platform (it now requires `enabled`), but nothing else about this run is blocked — and step 11C still runs for any other platform that is `enabled`.
      Do not treat saved labels such as `pending_private_activation` or `public_data_sources_only` as final without this runtime check; those labels may be stale after a human later installed, repaired, or reconnected the Local Collector.
   8. Load `playbooks/PRIVATE_SOURCE_GATE.md`, Stage 2, Stage 8, and Stage 9 before any Collector Runtime Verification involving private data sources. Do not use Claude in Chrome, Codex/browser tools, Playwright/Puppeteer/Selenium, or another agent-controlled browser as a fallback.
   9. Try to check private collector health through `GET http://127.0.0.1:17321/status`.
      - If the request succeeds, record `bridge_status: running`, check `status.persistent`, `status.job_available`, `status.output_dir`, `status.counts`, and `status.extension_health`.
      - If the bridge is online but `/status.config_file`, `/status.output_dir`, or `/status.run_now_request_file` points outside the current setup's `daily-content-pipeline/collector/` tree, mark `wrong_workspace_bridge`, do not run private collection, ask the human to run the current setup's Local Collector command, and remind them to remove/disable old Solo Agency Local Collector extensions in `chrome://extensions`.
      - If the bridge is online but `extension_health.status` is `stale` or `no_extension_check_yet` after the 75-second extension check grace window, mark private collection as unavailable for this run and notify the human.
      - If the workspace identity check passes and `extension_health.status` is `recent`, continue private collection.
   10. A scheduled/automation run executes inside the AI sandbox, where `127.0.0.1` is normally NOT the human machine's Local Collector localhost. Do not depend on localhost as the control path, and NEVER conclude the collector is inactive/down/unresponsive/needing-restart from a localhost failure — that failure is expected sandbox network isolation, not a collector fault. Use the file-based job queue and local health/status files as the primary path (a working localhost is only a bonus workspace-identity check, never the basis for a down/restart claim):
      - Read `daily-content-pipeline/collector/inbox/bridge_health.json` when present.
      - Read `daily-content-pipeline/collector/inbox/collector_status.json` when present.
      - Read `daily-content-pipeline/collector/collector_setup_status.md` when present.
      - Inspect recent `daily-content-pipeline/collector/inbox/YYYY-MM/*/*/collector_status.json` files.
      - Inspect recent consumed run-now status files such as `run_now_request_status.json`, `run_now_request.consumed.json`, or timestamped `run_now_request*.consumed.json` files when present.
      - If those local status files show a recent current-workspace bridge and recent extension check, use the Stage 8 file-based run-now queue by writing one unique per-client job file under `daily-content-pipeline/collector/jobs/pending/` and waiting for collector output. Do not ask the human to restart the Local Collector just because the API was unreachable from the AI sandbox.
      - A collector error is valid ONLY when this file-queue path fails: if the files are missing, stale, point to another workspace, do not prove a recent extension check, or a submitted job is not claimed/consumed within its TTL, mark the precise blocker: `collector_status_unverified`, `collector_offline_or_unreachable`, `wrong_workspace_bridge`, `job_not_consumed`, or `extension_status_unknown`.
   11. If no custom sources are configured, and discovery was never offered or was postponed, do not block the scheduled run. Continue with default sources, but include `Source Discovery Recommended` or `Source Discovery Declined/Postponed` in the report/notification. Explain that default-sources-only runs can still produce useful ideas but may miss community, lead, and competitor signals from sources that need a login.
   11A. If discovery (or a provided source list) was human-approved but no discovery scan has run yet (`approved_pending_first_scan`), this run must resolve it: when Collector Runtime Verification shows a healthy current-workspace bridge and a recent matching extension, create the first discovery run-now job (`job_type: "private_data_source_discovery"`, approved categories only, Source Discovery Mode pacing) via `POST /jobs/run_now` or a per-client job file under `daily-content-pipeline/collector/jobs/pending/`, wait for collector output, filter/classify candidates per Stage 2, and present the approval shortlist in the report/notification inside an `**[ACTION REQUIRED]**` block. Only the exact collector blocker from Collector Runtime Verification justifies reporting `Source Discovery Pending Activation` instead; a healthy collector with approved discovery must not defer it to a later run.
   11B. If a discovery shortlist is pending human approval (`discovery_completed_pending_approval`), do not re-run discovery. Continue public data sources plus any already-approved sources, and re-surface the pending shortlist in the report/notification inside an `**[ACTION REQUIRED]**` approval block until the human resolves it; offer a refresh scan only when the human asks or the shortlist is older than 14 days. After approval, save the active sources and perform Automation Resync so the next run monitors them.
   11C. **Social Discovery Pass.** One pass covering Facebook, Instagram and X. Precondition: at
      least one of `facebook_lead_source` / `instagram_lead_source` / `x_lead_source` is `enabled`
      (Client Intelligence Profile) — that is what makes the step run at all; each platform then
      participates independently. `pending` on a platform (Setup Flow's step 4 not yet resolved for
      it) skips that platform's steps quietly, with no awareness line, and does not block the rest
      of the run. `web_only` on a platform (the human's explicit, acknowledged choice — never a bare
      "để sau"/"bỏ qua") also skips that platform's steps, but carries it in the persistent web-only
      awareness line (`playbooks/06_AGENCY_REPORT_STANDARD.md`) in every report this run produces,
      with the lead-count consequence and the one-line way to turn it back on. Exception: a
      `web_only` platform whose reason starts with "not logged in" is re-probed with its step-1 call
      in its normal round-robin slot every run, not skipped outright (see
      `playbooks/10_LEAD_COMPETITOR_DETECTION.md`, "Re-probe on every run").
      - Platform table (ordered steps and budgets): `playbooks/10_LEAD_COMPETITOR_DETECTION.md`,
        "Social Discovery Pass" → "Platform table". Facebook: feed (`fb.search.posts`) → people
        (`fb.people.search`) → groups (`fb.groups.search`, `privacy == "public"` only, ranked by
        `member_count`) → in-group (`fb.group.search_posts`). Instagram: search (`ig.search.posts`)
        → people (`ig.people.search`) → profile depth (`ig.profile.posts`) → comments
        (`ig.post.comments`). X: search Latest (`x.search.posts`) → people (`x.people.search`) →
        profile depth (`x.profile.posts`) → replies (`x.post.replies`). Within a platform this order
        is fixed — never reordered, parallelized, or skipped to save budget.
      - Rotation: interleave platforms round-robin — one Facebook job, then one Instagram job, then
        one X job, repeat — per platform advancing one step per round; a platform that is not
        connected, not logged in, at its budget, or tripped simply loses its turn (its slot is
        skipped, never handed to another platform). Full rule text:
        `playbooks/10_LEAD_COMPETITOR_DETECTION.md`, "Social Discovery Pass" → "Round-robin rule".
      - Discovery terms (Facebook steps 1-3, Instagram steps 1-2, X steps 1-2) come from
        `tool public-keywords ... plan --kind discovery` (the `community_discovery` kind), shared
        across all three platforms; Facebook's in-group intent terms (step 4) come from
        `tool source-keywords ... plan` as usual.
      - Budget is decided per platform: use that platform's FIRST RUN tier when its own
        `{platform}_discovery_first_pass_done` is not yet `true` in the Client Intelligence Profile
        — i.e. this is this client's first-ever pass on that platform, regardless of which
        automation run number it is — and set that field `true` immediately after this pass
        completes for it; use the DAILY tier on every pass after that. Facebook FIRST RUN: ≤ 21
        collector calls (3 discovery terms, 3 feed, 3 people, 3 group searches, up to 4 public
        groups × 3 intent terms), spread ≥ 4 hours, `max_pages` ≤ 4; DAILY: ≤ 7 calls (1/1/1/1, up
        to 2 groups × 2 terms). Instagram and X FIRST RUN: ≤ 12 calls each (3/3/3/3); DAILY: ≤ 4
        calls each (1/1/1/1).
      - FIRST RUN: minimum 10 leads across all platforms combined is a FLOOR, not a stop — reaching
        it does not end the run; keep working every enabled platform's shortlist until its own
        budget is spent.
      - Read every job's result for a checkpoint/rate-limit/logged-out signal before submitting the
        next job on that same platform; the first trip on a platform stops THAT platform for the day
        — the round-robin rule keeps the other two running.
      - Persist the ranked Facebook group shortlist at `history/YYYY-MM/facebook_discovery_shortlist.jsonl`
        (Facebook only — Instagram and X have no groups).
      - Every post/person row from every platform goes through Stage 10 and straight to
        `lead capture` immediately, `platform` set to `facebook` | `instagram` | `x`, even from a
        Facebook group not yet in `private_data_sources`.
      - Scanning a public Facebook group needs no per-group approval; recommend the top groups by
        `leads_found`/`member_count` afterward and let the human decide which (if any) get promoted
        into `private_data_sources`. Instagram and X have no group concept.
      - Write actions on Instagram/X (react, comment, message, like, reply, publish, DM) are out of
        scope for this pass; it only reads and scores.
      - Full algorithm and report section: `playbooks/10_LEAD_COMPETITOR_DETECTION.md`, "Social
        Discovery Pass".
   11D. If no notification channel is configured (`notification_channel_missing`), include ONE short value-first re-offer in the run's report and progress block (instant alerts for hot leads, report-ready, drafts awaiting review) with the standard WideCast API-key instructions - once per run, never more.
   12. If private data sources remain unavailable after Collector Runtime Verification, continue with public data sources and previously collected private data when available. Log the exact verification outcome in the report and notification; do not merely say the config was public-only.
   13. Prepare the private data source queue if private data sources are available and collector health is acceptable:
      - Shared-Scan Gate first (playbook 08 carries the full contract; discovery jobs bypass it entirely): run `tools/solo_tool source-registry --pipeline {setup-root}/daily-content-pipeline due --client {client_slug} --kind private` with the queue's source URLs. Only `scan` entries go into the collector job; each `reuse` entry's `data_dir` is this run's data for that source (another subscriber scanned it inside the freshness TTL) — keep only records whose `source_uid` matches the reused `uid`, never `snapshots/`. `wait` entries: another client is scanning right now — use `stale_data_dir` when present, else skip this run (`waiting_shared_scan`). All sources fresh → no job at all; record `all_sources_reused`. `scope: exclusive` sources (the client's own page/group/site) are never served as reuse and always scan under this client.
      - keep the active daily queue around 20 sources or fewer per client by default;
      - prioritize sources most relevant to the client, target audience, target location, pain points, and content pillars;
      - classify extra sources as `weekly` or `optional` and rotate them across future runs;
      - do not run aggressive or parallel private data source scans for the same logged-in account.
   14. Check private data sources if available, using the Solo Agency Local Collector extension plus the Local Collector app when available, with `collector_config.scroll_delay_seconds` defaulting to 5 seconds and `collector_config.max_scrolls_per_source` defaulting to 5. **Each source gets two passes, and they answer different questions.**

    **14a. The search pass — who is asking.** Reading a group's newest posts finds whatever is newest, which in a community of any size is mostly not about this client. Searching the same group for the phrases members use when they need someone finds the reason to make contact. Ask the bank what to search today and submit exactly what it returns:

    ```sh
    <bridge> tool source-keywords --pipeline daily-content-pipeline --client {slug} --url {source url} urls
    ```

    It answers with ready run-now sources (capability `fb.group.search_posts`), each named `kw:<term>`, plus the depth to use. On a source's FIRST scan it plans up to 8 terms and digs; every day after, 3 terms with one scroll and a recency window, pinning the terms that have earned their place and rotating the rest (`playbooks/skills/lead-engine/safety.md`, monitoring search pass). This is the in-group channel of `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`, "Buyer Profile Channel Keyword Table": the per-kind quota is daily 1 role + 1 product/stage + 1 intent (the 3 terms above) and first-run 3 role + 2 product/stage + 3 intent (the 8 terms above), rotating within a kind least-recently-run first; `... source-keywords plan --kind {intent|role|product|stage|place}` requests one kind's quota alone, omitted returns the standard mixed set. A source whose bank is empty is seeded first: `... source-keywords ... seed --industry {industry} --market {market} --lang {lang}` — the seed set is chosen by industry AND market, because a trade's words are not the same in two countries, and role/product seeds sit alongside the existing intent seeds in `search_seeds.json` as fallback only, never overriding terms already generated from this client's own `buyer_profile.types`.

    When the pass finishes, count the NEW posts each `kw:` source returned and give those counts straight back, so the bank learns instead of guessing:

    ```sh
    <bridge> tool source-keywords ... record --json '{"kw:<term>":{"hits":N,"leads":M}}'
    ```

    `hits` is new posts (post_id not seen for this source before); `leads` is how many Stage 10 then qualified. A term that comes back empty enough times goes on probation and then retires itself, so the bank stays the words this group actually uses. If you notice a phrasing in the feed that members use and the bank lacks, add it WITH the reason you saw it: `... add --term "..." --kind intent --origin mined --note "three posts this week used this"`. A term nobody can justify is not a term.

    **14b. The feed pass — what the community is talking about.** Then scroll the feed as before, but shallow: its job is no longer to find leads, it is to catch what the terms missed, to see the shift in what the group discusses (which Stage 3 turns into content), and to supply the phrasings that grow the bank. Two or three scrolls is enough for that; the depth budget moved to the search pass. Classify every post this pass surfaces by the AUTHOR'S TYPE first (`playbooks/LEAD_QUALIFICATION_RULE.md`, Step 1) before deciding whether it is content signal, a lead, or noise — the feed pass has no search terms to anchor on, so the author's own words are the only fit signal it has.

    Both passes skip a post whose `post_id` was already collected for this source, so a term that returns the same twenty posts every day costs one comparison, not twenty judgements.
      - After private collection reaches a terminal state, reconcile status and counts before report handoff: private scan status, completed timestamp, sources attempted/completed/blocked, data points kept, leads, competitors, recommended private data sources, noisy/skipped discovery candidates, notifications, and blockers must match across the private report, daily index, internal source record, report state JSON, and `outputs/latest/` copies.
      - Do not leave stale `scan in progress`, `partial`, `pending`, or old recommended-source totals in one artifact after another artifact says the private scan is complete.
   14b. After private collection reaches a terminal state, record the scan outcomes in the source registry: `tools/solo_tool source-registry --pipeline {setup-root}/daily-content-pipeline record --client {client_slug} --run {run_id} --kind private` with the URLs actually scanned (`--status failed` for sources the collector could not read — a failure never overwrites the last good scan and the next subscriber's run retries with its own login; recording also releases this client's scan claim).
   15. If the collector bridge was started in `agent_on_demand` mode, stop it after collection completes or after timeout.
   16. Log skipped, pending-activation, expired, rate-limited, warning-triggered, collector-unavailable, extension-unavailable, Chrome-not-running, stale-extension, bridge-offline, collector-status-unverified, wrong-workspace, or unavailable private data sources.
   17. Load the private data stored by earlier completed runs for this client when available — at minimum the previous completed run (per the Run Window anchor: located from run history on disk, never `today − 1 day`), extended to all stored data from the last 7 days or the last 3 completed runs, whichever covers more — and filter duplicates: records carrying `point_uid` dedup by key equality first (exact, cross-client-safe), then visible text matching for near-duplicates. Do not parse private-platform HTML for duplicate detection. When this run consumed shared-scan data (`reuse` pointers), the reused records dedup against this client's history the same way — reuse changes where data comes FROM, not what this client has already SEEN.
   18. Extract relevant `[data_points]`, including reference URLs for every data point. Keep data points that are directly about the primary industry or clearly connected through a related industry. Discard related-industry data when the bridge back to the client's offer is weak.
   19. Add newly recommended private groups/pages/profiles/communities to `New Private Data Sources Detected` and `history/YYYY-MM/new_private_sources_log.md`.
   20. Load Stage 10 and qualify what the two passes returned: hot/warm/watch leads plus direct, indirect, adjacent, attention, and authority competitors. Before qualifying any row from either pass, load `playbooks/LEAD_QUALIFICATION_RULE.md` and apply Step 1 (WHO is this person) first, for every row, against this client's own `buyer_profile.types` — a person of the right type with no stated need is `warm`, never dropped; only after Step 1 do Step 2 (competitor/noise) and Step 3 (why now) run, and the matrix in Step 4 gives the decision. Every lead row carries `person_type`, `sells_to_match`, `fit`, `fit_reason`, `intent`, `intent_reason` alongside the decision. The leads are not a by-product of this run — step 14a went looking for them on purpose — so they lead the report (`playbooks/06_AGENCY_REPORT_STANDARD.md`, `People To Contact Today`), and the competitor and idea findings follow. The first lead/competitor pass for a client/source set should use 10 scrolls per approved private data source when safe; normal daily runs use 5 scrolls per approved private data source by default.
   21. For every useful lead or competitor opportunity, preserve profile URLs and post/current URLs when available, safe context summaries, reasoning, suggested human action, and a copy-ready value-first comment in the same language as the post.
   22. Generate the 3x2 idea matrix as six buckets, not six total ideas. Put every credible, source-backed idea from this run's collected data (the run_window) into the matching layer/scope bucket, and label each idea as `primary_industry` or `related_industry`. The matrix lives in the REPORT (the report is the idea archive); it is never bulk-queued into the provider production plan — only the TOP 3 (the three role cards — hottest, new development, foundation — the same three the report leads with) are queued (see the notification step), because ten queued look-alikes a day would bury the plan. The operator promotes any other matrix idea by asking.
   23. Check `history/YYYY-MM/content_log.md`, including the recent primary/related ratio and duplicate/near-duplicate idea risk.
   24. Perform the Idea Novelty Check: prefer at least 3 candidate ideas that are new or newly angled. If a prior topic is reused, record the prior idea/date, this run's new angle, and why the re-angle is materially different.
   25. Select the TOP 3 of the day by ROLE, not by one leaderboard (the Top 3 Role Rule):
       - Slot 1 — hottest: the strongest idea overall in the matrix (any source lane, any layer), per the Decision Scorecard. This is "the best idea of the day" wherever other playbooks use that phrase.
       - Slot 2 — new development: the best idea with genuine run_window novelty from a DIFFERENT source lane or matrix layer than Slot 1. A genuinely new public development (policy, regulation, market news that affects the client's customers) takes this slot ahead of a hotter lookalike trend; when no other lane/layer has genuine novelty, the slot falls to the next-best remaining idea.
       - Slot 3 — foundation: the highest-value `available` entry in the profile's `foundation_bank` that is not already queued/produced in the production plan or content history. Mark it `queued` with the run date. If the profile has no `foundation_bank` yet, generate it now in setup-repair mode per Stage 1 (from saved pain points × audience segments, no questions to the human) before selecting.
       - Foundation backfill: ANY slot that cannot be filled with a genuinely new, source-backed, value-first idea is filled from the foundation bank instead (two or even three foundation cards are valid on a signal-poor run). Never fill a slot with a weak or repeated idea just to have three.
       - Every slot must pass the Audience Value-First Gate and the Idea Novelty Check. When the foundation bank has no `available` entries left, re-angle a `produced` topic (seasonal, local, law/regulation update), record it per the novelty check, and mark the entry `re_angled` with the new angle noted.
   26. Write the configured production-ready draft using OpenAPI/native/MCP access when available, or the account-free writing skill fallback when provider/account access is unavailable. Keep writing-method/provider details in `INTERNAL_REPORT`, not client-facing files.
   27. Before any video provider creation request, load Stage 3 and the existing WideCast video script-writing skill again, treat the report/draft script as reference only, and create the final WideCast-grade script/brief with research plus inline-media/direct-image/video-URL workflow where verifiable. If the report already has a selected/recommended version/code, process only that one selected version; do not create a second five-version set. Do not edit or reimplement the skill.
   28. If a production provider is connected and the human has explicitly approved creation/rendering/publishing for a selected draft, create the approved video/blog/social asset according to provider approval gates. For video, use only the skill-produced final script/brief from the prior step as the provider payload. If approval or provider setup is missing, keep the asset as `approval_required` or `provider_setup_required`; for video, do not create local video media as a fallback.
   28b. Load Stage 5 (`playbooks/05_MEASURE_LEARN_IMPROVE.md`) and run published-URL measurement when published URLs exist for this client; otherwise record `measurement_status: no_published_urls_yet`.
   28c. Load `playbooks/ACCOUNTABILITY_POSTING.md` and run the posting-accountability check with the publish dates measurement already fetched: gap over the threshold (`accountability_max_gap_hours` in `system_settings.json`, default 72h, per-client override honored) → ONE reminder this run per the ladder (max 3 per episode, dedup via `analytics/accountability_log.md`); after the 3rd unheeded reminder → escalate ONCE to the operator via `tools/solo_tool gmail send-operator`. The client reminder is a SEPARATE notification under the Client Notification Contract, never merged into the report notification.
   29. Save `outputs/YYYY-MM/YYYY-MM-DD.md` as the canonical source-of-truth report.
   30. Generate the three scrubbed staging HTML report files under `outputs/YYYY-MM/YYYY-MM-DD/`: `{client-name}-public-data-sources-report.html`, `{client-name}-private-data-sources-report.html`, and `{client-name}-daily-report.html`.
   31. Generate or update the operator-only `{client-name}-INTERNAL_REPORT.html`, clearly labeled `INTERNAL_REPORT - Not for client sharing`, and put Solo Agency/WideCast/provider/Telegram/social-platform/API-key/config/Local Collector/automation/blocker/debug details there.
   32. Run the Client-Blind Scrub Gate on the staging HTML files and the final package. They must not mention Solo Agency, WideCast, PDNA/provider tooling, OpenAPI, MCP, Local Collector, Chrome extension, automation/scheduled task, API key/config, Telegram, agent/tool/debug details, or `INTERNAL_REPORT`.
   33. Generate or update `{client-name}-client-report.html`, `{client-name}-client-report.pdf`, `outputs/latest/{client-name}-client-report.html`, and `outputs/latest/{client-name}-client-report.pdf` from the scrubbed three staging HTML files, or record the exact PDF blocker/status. The combined HTML is the only default report link to send/upload; it must not require the reader to open separate daily/public/private files.
   33b. Write or update `outputs/YYYY-MM/YYYY-MM-DD/{client-name}-report_state.json` and reconcile it with the three staging HTML files, the combined client report, the PDF companion, and the INTERNAL_REPORT (counts, per-lane statuses, timestamps). No artifact may say `scan in progress`/`partial` while another says `complete`.
   33c. Write the operator-dashboard fleet snapshot `daily-content-pipeline/fleet/{client_slug}.json` (schema in Stage 7) with everything this run learned — posting gap + accountability state, engagement totals, production/lead counts, report paths, blockers. The dashboard only ever READS these files; the run is the only writer. Unavailable values stay null — never invented.
   34. Update or copy `outputs/latest/{client-name}-daily-report.html` only as a staging/diagnostic convenience, not as the primary report handoff.
   35. Update or copy `outputs/latest/{client-name}-INTERNAL_REPORT.html`.
   36. Update or copy the latest public/private lane HTML files when those lane reports exist.
   37. Update `history/YYYY-MM/content_log.md`.
   38. Update `history/YYYY-MM/data_sources_log.md`.
   39. Update `history/YYYY-MM/lead_log.md`.
   40. Update `history/YYYY-MM/competitor_log.md`.
   41. Update `history/YYYY-MM/lead_competitor_opportunities.jsonl` when possible. Carry the author's NAME into each row's `name` field, verbatim from the collected record — a post scan always saw it, and a row without it produces a CRM contact labelled with a database id instead of a person. Carry `emails`/`phones` too when the collector found them. Never write Facebook interface text (`Top contributor`, `Verified account`, a group name, a url) into `name`: the capture refuses it and the contact stays honestly nameless (`playbooks/10_LEAD_COMPETITOR_DETECTION.md`, the ledger contract).
   42. Capture the run's leads into the client's CRM — `<bridge> tool crm-store --pipeline daily-content-pipeline --client {slug} lead capture --file history/YYYY-MM/lead_competitor_opportunities.jsonl` (Stage 10, "Every lead also becomes a CRM contact"). This reads the file written in step 41, so it runs after it and never before. It sends nothing; it only means the person is still findable tomorrow. Report what came back — created versus matched — in the run summary, because "matched" is the system recognising somebody it already knows. If it answers `no outreach workspace ... run init-client first`, this client has never had a CRM: run `init-client` once (Stage 10, "First run for a client that has never had a CRM") and repeat this step. When this capture (or the Social Discovery Pass) produced ≥ 1 lead, append the CRM link line to the operator-facing reply and `INTERNAL_REPORT` — the exact bilingual lines and the zero-lead variant live in `playbooks/10_LEAD_COMPETITOR_DETECTION.md`, "Show the CRM link after any scan that produced ≥ 1 lead". First-run priming (funnel moment F): if this capture is the run that moves this client's CRM from 0 to > 0 contacts for the first time, read `contact lock-status` right after the capture and add one plain-fact sentence to the run summary and `INTERNAL_REPORT` naming the real unlocked-contact count against the Free ceiling from that response — never an estimate, and this fires once, on that first 0→>0 run only (later runs use the meter defined in `playbooks/10_LEAD_COMPETITOR_DETECTION.md`, "Capture never stops at the plan's contact cap").
4. Create or update `outputs/YYYY-MM/YYYY-MM-DD_master_digest.md`.
5. Generate `outputs/YYYY-MM/YYYY-MM-DD_master_digest.html` as a polished standalone human-facing master report.
6. Update or copy `outputs/latest_master_digest.md`.
7. Update or copy `outputs/latest_master_digest.html`.
8. Present the daily digest to the human.
9. Load Stage 6 and run the Provider Report Delivery Capability Check before claiming the run is complete. This check must use each target client's provider config/OpenAPI identity first and must be recorded in `INTERNAL_REPORT`; a global MCP/native provider account in the current AI session is not proof that the client has notification/upload/analytics/publishing configured.
10. Prepare a report-delivery record containing the local scrubbed `.html` report path, local PDF path/status, INTERNAL_REPORT path/status, client-facing scrub status, provider, provider discovery/account verification status, upload operation ID, upload attempt status, uploaded report URL if available, notification channel, final notification report link, and blockers.
10b. Before the notification, queue the run's TOP 3 ideas (the three role cards per the Top 3 Role Rule — the same three the report leads with) into the provider production plan when the verified client provider exposes the add operation (WideCast: `addToProductionPlan`, `POST /v1/production_plan/add`, synchronous, free): dedup EACH against `getProductionPlan` first (skip with `skipped_duplicate` when that topic is already queued/produced), send `idea_text` + one-line `description` + `source: "idea"` + `language` (the client's `content_output_language`) per idea — and for the RECOMMENDED idea (slot 1, the best-idea card) also `recommended: true` + `scripts` (the report's five client-safe version texts as `{"format": "VE|QA|POV|CS|MB", "text": ...}`, verbatim from the report's ## Version N sections) + `recommended_format` when the report names one. The new fields ride ONLY when the discovered spec exposes them (older server → legacy payload, note `provider_spec_predates_scripts` internally). Record each outcome (including `scripts_attached`) for the notification. Three per run and never the whole matrix; the magic link opens Saved Ideas, so the screen must show exactly what the report showed. A missing operation is noted internally and never blocks the notification.

10c. Then MINT the no-login content-plan link when the provider exposes a client-link operation (WideCast: `sendClientLink`, mint-only — `{"link_type": "content_plan"}`, `channels` omitted): keep the returned `magic_url` for the notification, so the recipient opens the Saved Ideas / production plan screen with no login. Mint-only: the provider sends nothing itself; the run's ONE notification carries every link. Missing operation → note internally, never a blocker.

11. If WideCast OpenAPI notification/Telegram/email fallback is configured, inspect whether the discovered spec exposes an HTML-capable upload operation. For WideCast, use `uploadAsset` with `text/html` and `sendNotification`. If upload exists, upload the combined `{client-name}-client-report.html` to WideCast for operator delivery first, then send a notification to the human/operator that includes the uploaded report URL, the minted content-plan `magic_url` (10c), the add-to-plan outcome (10b), PDF companion path/status, INTERNAL_REPORT path/status, run status, clients processed, blockers, lead/competitor counts, and required actions — one message, every link. A minted `magic_url` is a client-ready no-login link by design; ad-hoc provider-hosted URLs (raw upload URLs, other provider screens) remain operator handoff links, and report CONTENT stays client-blind as before.
12. If WideCast notification is available but HTML upload is unavailable or fails, log the exact upload blocker and still send a WideCast notification that includes the best available local/hosted combined `{client-name}-client-report.html` path/link plus PDF companion path/status plus INTERNAL_REPORT path/status.
13. If provider config is missing, auth fails, OpenAPI discovery fails, account identity mismatches, the only visible provider account is a global MCP/native account that is not proven to match the client, or required operations are missing, log the exact provider-neutral blocker and provide the best available HTML path/link plus PDF companion path/status plus INTERNAL_REPORT path/status in chat or an authorized fallback channel.
14. If another authorized channel can send the HTML file or link more conveniently only because provider notification is unavailable or blocked, use it.
15. Log the upload attempt and notification attempt in `notifications/notification_log.md`.

The daily run is complete only when every active client is processed or explicitly logged as skipped.

When presenting the daily idea list to the human, include reference URLs next to data points, top ideas, and the selected best idea so the human can verify the information. For private data, include the captured source URL and note that it may require the human's logged-in session.

Scheduled runs must assume the human may not be present in the AI agent UI. The run is not fully operationally complete until the scrubbed mobile-friendly client-facing HTML result plus PDF companion path/status plus INTERNAL_REPORT path/status, or a result-ready notification with those paths/statuses, has been sent through the configured notification channel, preferably WideCast OpenAPI Telegram/email fallback when configured for that client.

If this client's WideCast notification/Telegram/email fallback is connected and WideCast report upload supports HTML, the operator notification link should use the uploaded report URL, not only a local file path. Ad-hoc provider-hosted URLs are not client-share links; the minted content-plan `magic_url` is, by design. If upload fails, discovery fails, auth fails, account identity cannot be proven client-scoped, or the current provider spec does not support HTML upload, log the blocker and send the best available HTML path/link plus PDF companion path/status plus INTERNAL_REPORT path/status.

If a WideCast upload/Telegram step is skipped because the agent did not inspect the configured client provider/OpenAPI capabilities, the scheduled run is incomplete. The agent must correct the omission by running the Provider Report Delivery Capability Check, updating `notification_log.md` and `INTERNAL_REPORT`, and sending a correction message with the HTML report URL/path, PDF companion path/status, INTERNAL_REPORT path/status, and blocker.

A notification that only says the report is ready but contains no HTML report URL/path is invalid. If this happens, immediately send a correction notification with the HTML report URL/path and log the correction.

---

- During one-time setup, after the profile and source plan are known and before the client-specific automation task is marked ready, ask the human whether they want daily, multiple-times-daily, weekly, manual-only, first-run-only, or another cadence — and their preferred start time in the same question.
- Resolve the actual start time through the Time-slot collision rule above (`tool schedule-slots suggest` → create the native task at the suggested time → `register` it). Never create a native task at an unchecked time, with the single exception named in that rule: the first task of a fresh install, created before the bridge that carries the tool exists, which is created at the requested time and registered the moment the bridge arrives (`slot_check_pending`).
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

- Timezone definition: `"timezone": "local"` means the human machine's local timezone as recorded in `daily-content-pipeline/schedule.md`. All dates, `YYYY-MM-DD` folder keys, run_window boundaries, and 7-day measurement windows use that timezone. The AI scheduled-task environment may be a cloud/sandbox running at UTC; before computing any date key or window, the scheduled run must read the recorded timezone from `schedule.md` so a run does not split one logical day across two date folders or mis-window measurement.
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
     - recent `daily-content-pipeline/collector/inbox/YYYY-MM/*/*/collector_status.json`,
     - recent `run_now_request_status.json` or `run_now_request*.consumed.json` files.
   - if those files show a recent current-workspace bridge and recent extension check, use the file-based run-now queue from Stage 8 rather than asking the human to restart the collector;
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
Impact: Facebook, Instagram and X were not read today (not connected), so lead counts may be lower than reality; everything else ran.
Action: Open Chrome with the Solo Agency Local Collector extension enabled, stay logged in, or run the Local Collector app start command again if needed.
```

### OS Startup For Persistent Bridge

The canonical setup scripts (`setup_collector.sh` / `setup_collector.ps1`, 2026-07-20+) register OS autostart BY DEFAULT when the human runs them: macOS per-user LaunchAgent `com.solo-agency.collector.{insthash}` (RunAtLoad; restarts on crash, stays stopped after a clean stop), Linux systemd user unit `solo-agency-collector-{insthash}.service` (`Restart=on-failure`, best-effort `loginctl enable-linger` for start-before-login), Windows logon Scheduled Task `SoloAgencyCollector-{insthash}` (crash restarts, output wrapped into `collector.log`). `{insthash}` = first 8 hex chars of SHA-256 of the agency root path, so two installs on one machine never collide. The human opts out with `SOLO_AGENCY_NO_AUTOSTART=1` (plain background start). Re-running the script refreshes the registration and remains the ONLY start/upgrade command the human ever needs. After a reboot the bridge should be up without anyone running anything.

Every setup run records the outcome in `solo-agency-local-collector/autostart.json`: `{"mode": "launchd"|"systemd"|"scheduled_task"|"none", "label", "port", "root", "registered_at", "reason"}`. This file is the canonical evidence for agents — it lives inside the workspace, so even a sandboxed agent can read it when OS commands and `~/Library`/`~/.config` are out of reach.

**Agent duty — verify autostart after any collector setup/upgrade, and FIRST when diagnosing a bridge that is down after a reboot. Pick the deepest rung you can actually perform; never assume and never silently skip:**

1. Agent can run OS commands (local, unsandboxed): verify directly — macOS `launchctl print gui/$(id -u)/com.solo-agency.collector.{insthash}`; Linux `systemctl --user is-enabled solo-agency-collector-{insthash}.service`; Windows `Get-ScheduledTask -TaskName SoloAgencyCollector-{insthash}`.
2. Sandboxed agent: read `solo-agency-local-collector/autostart.json`. `mode` other than `"none"` = registered (report which). `mode: "none"` = autostart is OFF — tell the human why (`reason`: `opt_out_env`, `*_registration_failed`, `no_supervisor_available`) and that the fix is re-running the setup script. File missing = the install predates the autostart scripts — ask the human to re-run the current setup script once.
3. Agent can neither run commands nor read the file: give the human the ONE copy-paste command for their OS from rung 1 and ask them to paste the output back.

Guardrails (unchanged in spirit): do not hand-craft a custom LaunchAgent/systemd unit/Scheduled Task beyond what the script registers. For a missing or broken registration, branch on runtime: on a **local runtime** (rung 1, the agent can run OS commands directly and see the install root), the agent re-runs the setup script itself — one consent ask, safe to re-run — to repair/refresh the autostart registration, then re-verifies; on a **remote runtime** (rung 2/3, sandboxed with no direct OS access), the fix is that the HUMAN re-runs the setup script, since the agent cannot register the autostart service from inside the sandbox. Claude-specific: Claude often cannot run downloaded binaries from inside its sandbox; Claude must not use Claude Chrome Extension as a workaround for automated private collection; after the bridge runs as a startup service, Claude reads collector output files and continues without controlling Chrome directly.
