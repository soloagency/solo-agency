# Agency Operations Safety Audit

Stage: `09`

## Load Rule

Load before claiming setup, daily run, private data source setup, schedule, production, publishing, notification, measurement, or agency operating cycle completion.

## Hard Gates For This Stage

- Run the relevant checklist before every completion claim.
- Report missing steps honestly.
- Respect approval gates and regulated-industry safety.
- For each test log, identify skipped stages, unnecessary questions, jump-ahead behavior, report format failures, and missed gates.
- If any required stage was not loaded, load it before proceeding.
- Before any private data source scan, confirm `playbooks/PRIVATE_SOURCE_GATE.md`, Stage 2, Stage 8, and Stage 9 were loaded in the current private data source turn, even if the conversation drifted through unrelated topics.
- Treat any use of Claude in Chrome, Claude Chrome Extension, Codex/browser tools, Playwright/Puppeteer/Selenium, a fresh agent-opened browser profile, or another agent-controlled browser for private data source collection as a critical workflow violation.
- Before claiming any post-schedule change is complete, verify Automation Resync was performed when schedule/automation already exists. Config-only updates are not enough if a native scheduled task prompt may still contain an old snapshot.
- Before claiming any Solo Agency update/upgrade/sync-latest work is complete, verify Stage 11 was loaded, GitHub `main` was checked from a verified source, backups/logs were written, clients and automations were resynced, and bridge/extension human actions were given when required.
- Treat local DIY video production as a critical workflow violation when client-scoped PDNA/video provider setup is missing, unverified, mismatched, or missing the required operation. Do not create MP4/MOV/GIF/slideshow/rough video files with `ffmpeg`, Pillow, `moviepy`, browser/canvas screenshots, Remotion, or similar fallback renderers.
- Treat sending a report script, Markdown source record, previous draft, or content-history script directly to a video provider without first loading and applying the existing WideCast video script-writing skill as a critical workflow violation. The skill pass must run research and Stage 2 inline-media/direct-image-URL workflow when relevant, even if PDNA is missing and the run can only stop at a script/production-brief blocker. Editing, replacing, summarizing, or reimplementing the WideCast skill is also a workflow violation.
- Treat generating a second five-version script set during video production as a workflow violation when a report version/code, pasted edited version, or automation recommended/approved version already exists. In that case the agent must process only the selected version/code through the WideCast skill's research, factual-core, Stage 2 visual treatment, inline media, media pool, and production handoff standards.
- Before claiming any report run is complete, verify Stage 6 loaded `playbooks/skills/report-design/SKILL.md` and that client-facing HTML/PDF was generated with `tools/solo_report_renderer.py` or a reusable approved template. One-off report/PDF scripts are a workflow violation unless the exact blocker and approved exception are logged.

## Latest Override: Setup Flow And Client Isolation Audit

Before claiming Setup Flow completion:

- Confirm no report, first agency run, public scan, private data source scan, production action, render, publish, or outreach was executed inside the setup chat.
- Confirm the client-specific automation task exists or has been proposed for human review, and its name begins with the client name, for example `AvenNgo - Solo Agency Daily Run`.
- Confirm the scheduled prompt pins the target client and does not process other clients.
- Confirm any setup/config change after schedule creation triggered Automation Resync.
- Completion wording must be `ready_for_automation_first_run` or `ready_for_next_automation_run`, not `report complete`.

Before claiming Automation Flow completion for a client:

- Confirm the task processed only the target client.
- Confirm private data source collection, if attempted, used only the shared Local Collector app plus the matching client extension.
- Confirm `client_slug`, `extension_instance_id`, and output path all match.
- Confirm collector output was read only from `daily-content-pipeline/collector/inbox/YYYY-MM/{client_slug}/`.
- Confirm the client has one canonical combined client-facing report for the day/run: `{client-name}-client-report.html`, built from scrubbed staging files `{client-name}-public-data-sources-report.html`, `{client-name}-private-data-sources-report.html`, and `{client-name}-daily-report.html`.
- Confirm the report-design skill was loaded before report generation/repair and `tools/solo_report_renderer.py render` or a reusable approved template produced the client-facing HTML.
- Confirm `tools/solo_report_renderer.py package` or a reusable approved package template produced the PDF companion package, or the exact PDF blocker is logged.
- Confirm the public report and private report have separate source coverage, evidence, Lead & Competitor Opportunities, idea matrix, best idea, and draft/recommendation.
- Confirm the private pass did not overwrite, delete, reorder, summarize away, or regenerate the public report file.
- Confirm `outputs/YYYY-MM/YYYY-MM-DD/{client-name}-report_state.json` exists or the exact blocker is logged, and that its public/private statuses match the report set.
- Confirm any public and private notifications point to `{client-name}-client-report.html` or its uploaded URL as the canonical handoff. Daily/public/private staging links should be omitted unless requested for diagnostics.
- Confirm changes discovered during the run were written back to persistent setup/config and resynced.

Treat these as critical workflow violations:

- Setup Flow runs a report directly instead of directing the human to the automation task.
- A client-specific task name does not begin with the client name.
- An extension for client A receives, writes, or completes a job for client B.
- An agent reads private data source output from another client's collector inbox.
- A global `extension_health.status: recent` is used as proof that the target client's extension is healthy without checking the matching `client_slug + extension_instance_id`.
- A private data source pass overwrites, regenerates, or summarizes away `{client-name}-public-data-sources-report.html`.
- The report set is missing `{client-name}-daily-report.html` as the staging index file.
- A report run writes a new one-off Python/browser/PDF script instead of using or fixing the reusable report renderer.

## Source Preservation Rule

This file is detailed source material moved from the original monolithic `SOLO_AGENCY_PLAYBOOK.md`.

Do not summarize away requirements, examples, checklists, schemas, protocols, URLs, edge cases, warnings, approval gates, or completion gates. If a downstream agent needs to shorten its response to the human, it may summarize the response, but it must still obey the full requirements in this file.

---

## Automation Resync Safety Check

Run this check before saying any setup repair, private data source approval, Local Collector repair, PDNA connection, notification change, analytics change, schedule change, or client/profile update is complete when a schedule/automation already exists.

Ask:

1. Did this change happen after `daily-content-pipeline/schedule.md` or a native automation/scheduled task was created?
2. Would tomorrow's scheduled run need to know about this change?
3. Could the scheduled task have an old prompt snapshot, old source status, old approval state, or old collector path?

If yes to any of those, the agent must load Stage 4 and perform Automation Resync before claiming completion.

Minimum resync audit:

- Client Intelligence Profile updated.
- Source/discovery/history logs updated when source state changed.
- `daily-content-pipeline/provider_defaults.json` updated when provider catalog/discovery defaults changed.
- The relevant client's `integrations/providers/` config, OpenAPI cache, capabilities, health, and provider call log updated when PDNA/provider/notification/analytics changed.
- `daily-content-pipeline/schedule.md` updated.
- `daily-content-pipeline/collector/collector_config.json` or `POST /config` updated when private data source collection changed.
- `daily-content-pipeline/automation/automation_manifest.md` updated.
- `daily-content-pipeline/automation/scheduled_run_prompt.md` updated.
- Actual native AI automation/scheduled task prompt updated when accessible, or `automation_prompt_update_pending` clearly logged when not accessible.
- `daily-content-pipeline/automation/github_issues.md` updated when a tracked GitHub issue, maintainer/community response, or issue-derived workaround affects future runs.
- `daily-content-pipeline/automation/update_state.json` and `daily-content-pipeline/automation/update_log.md` updated when an update check or applied update affects future runs.
- `daily-content-pipeline/automation/resync_log.md` updated.
- Dry-read verification performed from the scheduled-run entrypoint, issue tracker when present, and latest local files. The dry read must enumerate and confirm the following 8 files: the scheduled entrypoint, `daily-content-pipeline/automation/automation_manifest.md`, `daily-content-pipeline/automation/github_issues.md` issue tracker, `daily-content-pipeline/automation/update_state.json` update state, `daily-content-pipeline/provider_defaults.json` provider defaults/config when relevant, `daily-content-pipeline/schedule.md` schedule, the Client Intelligence Profile, and `daily-content-pipeline/collector/collector_config.json` collector config.
- The human-facing progress block includes an `Automation freshness check` that answers whether the latest changes were synced into automation/scheduled task prompt/contract/playbook/source state, not only config, and whether tomorrow's run will load the newest state.

Completion wording must distinguish full vs partial sync:

```text
Automation Resync complete: the next scheduled run will read the latest approved sources/config.
```

or:

```text
Automation Resync partially complete: local files are updated, but the native scheduled task prompt still needs human update.
```

Do not say:

```text
Config updated, so automation is done.
```

That is a safety-audit failure.

---

## Update And Version Watch Safety Check

Run this check before saying an update command, update-watch run, setup repair update, or stale-version recovery is complete.

Minimum update audit:

- Stage 11 `playbooks/11_UPDATE_AND_VERSION_WATCH.md` was loaded.
- The source was GitHub `main` from `https://github.com/soloagency/solo-agency`.
- The agent used the current verified setup root or a fresh unique `mktemp -d` checkout.
- No fixed shared fallback folder such as `/tmp/solo-agency`, `/var/tmp/solo-agency`, or `/dev/shm/solo-agency` was used.
- `.git`, `origin`, local `HEAD`, and remote `refs/heads/main` were verified before reading or copying source files.
- The update check covered root instructions, all playbooks, provider/OpenAPI tooling, Local Collector bridge/runtime, Chrome extension templates, setup scripts, templates, client runtime copies, and automation contracts.
- `daily-content-pipeline/automation/update_state.json` and `update_log.md` were created or updated.
- Runtime files/folders replaced by the update were backed up under `daily-content-pipeline/automation/backups/` or an equivalent logged backup path.
- Secrets and local user/client state were preserved, including provider API keys, `provider_config.local.json`, private data source captures, Client Intelligence Profiles, approvals, history, reports, outputs, analytics, publishing logs, and extension `client_binding.json`.
- Every active/configured client was checked for required schema/template updates.
- Every affected `extensions/{client_slug}/` folder was regenerated or patched when extension code/templates changed.
- Automation Resync was performed when schedule/automation exists.
- If native scheduled task prompts could not be edited directly, `automation_prompt_update_pending` was logged and the human received the exact replacement prompt path/action.
- If bridge/runtime files changed, the human received the exact current-setup command to run outside the AI sandbox.
- If extension files changed, the human received the exact Chrome profile plus `chrome://extensions` reload or `Load unpacked` steps for each client extension.
- The human-facing completion states `update complete`, `update partially complete`, or `update blocked` with the exact remaining action.

Treat these as critical workflow violations:

- Saying "updated" after only pulling local code but not resyncing client automation tasks.
- Copying from an unverified stale folder because GitHub access failed.
- Updating playbooks but leaving scheduled prompt snapshots on the old behavior.
- Updating the extension template but not updating per-client `extensions/{client_slug}/` folders.
- Updating bridge/extension code without telling the human to rerun the bridge or reload the Chrome extension.
- Overwriting `provider_config.local.json`, API keys, private data source captures, reports, history, or outputs during an update.
- Running a report, scan, production action, publish action, or analytics scan merely because the human said `update`.

---

## 12. Multi-Client Batch Mode

The pipeline must support many clients across many industries.

If the human already has multiple clients, the agent should accept a compact list.

Example human input:

```md
I manage 10 clients. Set up one daily content pipeline for each:

1. Smith Law - DUI lawyer - Los Angeles - private data sources: competitor FB pages A, B
2. Austin Home Group - real estate agent - Austin, TX - private data sources: none yet
3. Bright Mortgage - home loans - Texas - private data sources: competitor TikTok X
4. Miami Shield Insurance - home and auto insurance - Miami - private data sources: local FB group Y
5. Vienna AI Ops - AI automation agency - Vienna - private data sources: LinkedIn competitors
```

The agent must:

1. Create one pipeline folder per client.
2. Infer all setup fields for each client.
3. Show a setup summary for each client.
4. Ask the human to correct only what is wrong.
5. Save one Client Intelligence Profile file per client.
6. Add all clients to `clients_index.md`.
7. Configure or document the daily schedule/routine.
8. Ensure daily runs process every active client.

If the human provides incomplete entries, infer what is possible and ask only for missing critical information.

---

## 13. Incremental Client Onboarding Rule

The pipeline must support starting with zero clients and adding clients over time.

The human is not required to provide all clients at once.

If there are no clients yet, create only:

```text
daily-content-pipeline/
  clients_index.md
  schedule.md
  clients/
  outputs/
```

Then immediately enter First Client Setup Mode.

First Client Setup Mode is the same as Add Client Mode, but it is triggered automatically during the first run when `clients_index.md` has no real client rows. The agent must proceed as far as possible toward setting up the first client instead of stopping after root folder creation.

In First Client Setup Mode, ask only for the minimum information required to create the first client pipeline:

- Client name, if not already known.
- Product/service, profession, expertise, business description, or public website/profile URL.
- Target location only if location matters and cannot be inferred.
- Optional private data sources to monitor.
- Optional permission for private data source discovery from joined groups/subreddits/communities, followed profiles/pages/KOLs/channels, subscriptions, and platform recommendation feeds.

Do not create fake client pipelines. If the client name or enough business context is missing, ask for the missing information and keep the root pipeline ready. A public website/profile URL counts as business context if it is publicly accessible and gives enough information to infer the setup.

Whenever the human says something like:

- "Add a new client"
- "Add this client to the pipeline"
- "We just got a new client"
- "Start monitoring content ideas for this business"
- "Add client: ..."

The agent must enter Add Client Mode.

In Add Client Mode, ask only for missing critical information:

- Client name.
- Product/service, profession, expertise, business description, or public website/profile URL.
- Target location only if location matters and cannot be inferred.
- Optional private data sources to monitor.
- Optional permission for private data source discovery from joined groups/subreddits/communities, followed profiles/pages/KOLs/channels, subscriptions, and platform recommendation feeds.

The agent must infer:

- `industry`
- `sub_industry`
- `target_audience`
- `pain_points`
- `content_pillars`
- `business_offer`
- `public_data_sources`
- `brand_voice`
- `language`
- `platforms`
- `compliance_notes`
- `negative_topics`

Then the agent must follow the same 9-item setup model. Do not introduce Add Client setup steps 10+.

1. Show the inferred setup summary to the human and ask them to correct only what is wrong.
2. Create or update the client pipeline folder, Client Intelligence Profile, history folder, outputs folder, and `clients_index.md` row.
3. Save the inferred pain points, content pillars, business offer, language assumptions, and compliance notes.
4. Save the public data source plan and keyword bank.
5. Configure the recurring schedule/routine once the basic source plan is known, prepare or verify the client-specific extension folder under `extensions/{client_slug}/`, and create/resync the client-specific automation task with the task name beginning with the client name.
6. Ask and resolve private data sources once after automation exists: record provided sources as `pending_private_review`, declined/no sources, pending Local Collector activation/private data source discovery, or `discovery_declined_or_postponed`. If private data sources exist or discovery is approved/pending, guide Local Collector setup now, or record private data sources as `pending_private_activation` so the automation task can continue with public data sources only if needed. Discovery may include approved joined/followed/member spaces or Facebook keyword group search; any discovered source still needs human approval before activation. Resync the already-created automation task after the decision.
7. If the human asks for production/video/blog/social, publishing, notifications, analytics, or "full automatic" during Setup Flow, load `playbooks/03_PRODUCTION_DISTRIBUTION.md` only as a provider/configuration gate. Do not create or publish assets from Setup Flow.
8. Record analytics as an Automation Flow concern only.
9. Do not run the first agency run, first report, public scan, private data source scan, report updates, idea matrix updates, Lead & Competitor Opportunities, draft generation, analytics scans, video creation, publishing, or PDNA production actions inside Setup Flow.

If the human asks to run, create, generate, show, refresh, or update a report during Setup Flow, treat it as a failed-safety condition unless the agent stops operational work. The agent must verify/resync the client-specific automation task, provide the exact task name to run, and avoid loading the scheduled-run entrypoint or running any report work inside the setup chat.

Example:

Human:

```md
Add this client to the daily content pipeline: Nguyen Law, immigration lawyer in San Jose. Private data sources to monitor: [links].
```

Agent must create:

```text
daily-content-pipeline/
  clients/
    nguyen-law/
      immigration-law_san-jose/
        client_profile_nguyen-law_immigration-law_san-jose.md
        history/
          content_log.md
          data_sources_log.md
        outputs/
```

The agent must configure the routine, prepare the `Nguyen Law - ...` automation task, then resolve/resync step 6 if private data sources are pending, and tell the human to run that client-specific task for the first report. Setup Flow must not run Nguyen Law's report directly.

---

## 14. Mandatory Automation Readiness Protocol

This protocol applies after the first client setup, after adding a new client, and after repairing an incomplete Client Intelligence Profile.

The setup flow is not a menu of optional next steps. The agent must not ask the human to choose between:

- providing private data sources,
- configuring the schedule,
- running the first agency run/report,
- creating a video.

The one allowed private data source choice is operational, not a new menu: after the automation task exists, if private data sources exist and Local Collector is pending, ask whether to activate Local Collector now so future automation can include data from private data sources, or mark private data sources pending so the automation task can run public data sources only until the blocker is resolved.

The correct order is fixed:

1. Finish setup and save the Client Intelligence Profile.
2. Configure the schedule/routine once the basic source plan is known.
3. Prepare or verify the client-specific extension folder under `extensions/{client_slug}/`.
4. Create or resync the client-specific automation task. The task name must begin with the client name.
5. Ask and resolve the step 6 private data source intake/discovery/approval and Local Collector activation when applicable.
6. If private data sources exist and Local Collector is not installed/running/healthy, guide Local Collector setup now, or record `pending_private_activation` so automation can continue with public data sources only if needed.
7. Update every persistent state file that the next automation run reads: Client Intelligence Profile, source state, collector config, extension registry, schedule, automation manifest, scheduled prompt/task body, and resync log.
8. End Setup Flow with `ready_for_automation_first_run`, not `first_report_completed`.
9. If the human wants PDNA setup - Production, Distribution, Notification, and Analytics - complete only the provider/configuration gate in Setup Flow.
10. Reports, drafts, analytics, production, publishing, and private data source scans run only in Automation Flow.

First automation report rule:

- The first report happens in the client-specific automation task, after routine setup, task creation, the step 6 private data source checkpoint, and human action to run or schedule that task.
- Setup Flow must not create `/jobs/run_now`, must not scan public data sources, and must not scan private data sources merely to produce a report. One sanctioned exception: the step-6 discovery pass (configuration gathering) — when the human approved discovery in-session and the collector plus matching extension are verified healthy, Setup Flow may create the discovery run-now job, present the shortlist, and save approved sources, but must not analyze the collected data or produce a report from it.
- If private data sources were provided/approved and Local Collector is not installed/running/healthy, the setup agent must handle the step 6 checkpoint after automation exists: guide Local Collector setup now, or record private data sources as pending, then resync the task.
- The first automation task should use public data sources, public search, client context, inferred pain points, inferred content pillars, related industries, and any previously collected local data.
- If private data sources were provided, the automation report must include a section called `Private Data Sources Pending Activation`.
- If private data source discovery was approved but not yet run, the automation report must include `Private Data Source Discovery Pending Activation`.
- That section must list the private data source URLs or discovery categories, explain that they were not scanned yet, and say that activation requires the Solo Agency Local Collector extension plus Local Collector app.
- `Private Data Source Discovery Pending Activation` is valid ONLY together with the exact collector blocker that prevented the scan (collector not installed/unreachable/unhealthy, stale or mismatched extension, `collector_status_unverified`). When discovery is approved AND Collector Runtime Verification shows a healthy current-workspace bridge with a recent matching extension, the run must execute the discovery job instead of writing this section — see the Automation run-now rule below.
- The automation report must include at least one draft script/blog/caption or a clear report section containing the draft.
- The automation report must ask a clear next-step question after delivering the useful output. Unless PDNA setup - Production, Distribution, Notification, and Analytics - was already completed or explicitly declined, the next-step question should be:

```md
Do you want me to set up PDNA - Production (create real video/blog/social assets), Distribution (publish approved content), Notification (send reports/blockers), and Analytics (measure results) - so approved drafts can become real assets and the system can learn from performance later?
```

The automation agent must ask this question directly in the chat message or notification where it announces the first report result. It must not hide the question or setup steps inside a Markdown file.

The same chat message must show the `Solo Agency automation process` progress roadmap or compact status. The automation agent must not hide pending private data source activation, report delivery blockers, production/provider setup status, or analytics state.

Good first automation report chat pattern:

```md
The first automation report is ready.

Best idea today: {best idea}
Report for mobile: {absolute HTML path or URL}
First draft: {script/blog/caption title}

Solo Agency automation process
This is the planned automation process for this client. You only need to reply when I ask one specific question.

✓ 1. You provided the product/service, profession, expertise, business description, or public website/profile URL
✓ 2. I inferred the industry, sub-industry, related industries, audience, and offer
✓ 3. I inferred pain points and content pillars
✓ 4. I selected public data sources and search keywords
✓ 5. I configured the automatic schedule/routine and client-specific automation task
– 6. Private data sources/Local Collector are pending or postponed; this automation run uses public data sources only
→ 7. I help set up PDNA: Production, Distribution, Notification, and Analytics
– 8. From the second run onward, if PDNA is set up, I scan analytics for published URLs from the last 7 days
✓ 9. I created the HTML report, idea matrix, Lead & Competitor Opportunities, competitor signals, and first script/blog/caption draft in Automation Flow

The operator-only `INTERNAL_REPORT` includes the PDNA/WideCast status and setup note. Client-facing reports do not mention Solo Agency, WideCast, provider tooling, Local Collector, automation, API keys, Telegram, or internal system details.

This run used public data sources only. I have {N} private data sources waiting, including:
- {source name or URL}
- {source name or URL}

Do you want me to set up PDNA - Production (create real video/blog/social assets), Distribution (publish approved content), Notification (send reports/blockers), and Analytics (measure results) - so approved drafts can become real assets and the system can learn from performance later?
```

Bad first automation report chat pattern:

```md
Private data sources were not scanned. Instructions are in collector/collector_setup_status.md.
Now choose a schedule.
```

Private data source activation rule:

- If the human agrees to activate private data sources, collector setup becomes mandatory at that point.
- The agent should proceed automatically as far as file preparation allows, but it must not run the one-time Local Collector setup/start command itself.
- During one-time setup/update/repair, the agent must never execute `setup_collector.sh`, `setup_local_collector.ps1`, `Start Local Collector.cmd`, or the collector binary from inside the AI agent, even if shell permissions are available. Agent-run setup can be trapped in a sandbox/session and killed after the turn.
- The agent must create the script/launcher file first and give the human exactly one short Terminal/PowerShell command or one double-clickable file path to run outside the AI sandbox, not a long multi-line script.
- The Stage 8 Source Safety Pre-Check was run before the bridge command or `Load unpacked` path was given: the prepared extension JS, `bridge-go/main.go`, and `prepare_client_extension.sh` were read, every outbound request was confirmed to go only to the local `127.0.0.1` bridge, and the result was recorded in `INTERNAL_REPORT`. The install handoff was preceded by the one short plain-language safety confirmation line. If the pre-check did not pass, the install command was NOT given and the finding was raised to the operator instead.
- The same setup handoff must include the Chrome extension install steps and the one absolute client-specific extension folder path under `extensions/{client_slug}/` for every new client, even when private data sources are not active yet.
- The handoff must say which Chrome profile/account to open for that client and must show `chrome://extensions` -> Developer mode -> `Load unpacked` -> select the absolute `extensions/{client_slug}/` folder.
- Saying only "I created the extension" or "extension folder exists" is incomplete.
- The exact human action must be shown directly in chat. The agent may also save it in `collector_setup_status.md`, but the saved file is only the agent's record and must not be the only place where the human receives the instruction.
- The agent must not label the collector by the current platform, such as `Facebook collector`.
- The agent must create or update `daily-content-pipeline/collector/collector_setup_status.md` when private data source activation begins.
- If the AI environment can write local files, the agent should download/update/extract the collector artifacts and create/update the setup script/launcher, but it must not run that setup script or start/restart the Local Collector app from inside the AI sandbox during one-time setup.
- After the human confirms they ran the setup/start command and loaded the Chrome extension, the agent may check `GET http://127.0.0.1:17321/status`.
- If the Solo Agency Local Collector extension is not loaded, the agent must show the absolute extension folder path and the exact Chrome `Load unpacked` steps.
- The extension path shown to the human must be the runtime workspace path under `extensions/{client_slug}/`, not any toolkit/source path under `solo-agency/solo-agency-collector/chrome-extension/`.
- After collector setup succeeds in Setup Flow, prefer to run the step-6 discovery pass immediately while the human is present (the sanctioned configuration-gathering exception above): scan approved categories, present the shortlist, save approved sources, then resync. If discovery is not run in-session, update the automation task and record `approved_pending_first_scan`. Daily monitoring/activation scans still never run inside Setup Flow.
- The agent must not claim private data source monitoring is active until collector health confirms the Local Collector app and Solo Agency Local Collector extension are working.
- The agent must not configure a recurring schedule that promises private data source collection until collector setup is either `installed_and_running` or explicitly documented as pending/blocked with a human action.

Automation run-now rule:

- Setup Flow must not create `/jobs/run_now` or run an equivalent manual report path. It must prepare or resync the client-specific automation task instead.
- In Automation Flow, a human request such as `run now`, `manual run`, `test run`, `trial run`, `collect now`, or `scan now` should bypass recurring schedule windows for the target client only.
- If the Local Collector app is already installed, running, healthy, and matched to the target client's extension identity, the Automation Flow agent MUST include private data sources by creating a run-now job. In particular, when discovery (or a provided source list) was human-approved but no discovery scan has run yet (`approved_pending_first_scan`), the first Automation Flow run with a healthy collector MUST create the discovery run-now job (`job_type: "private_data_source_discovery"`, approved categories only) and present the filtered candidate shortlist for human approval in the report/notification. Deferring an approved, collector-healthy discovery to a later run without an exact blocker is a safety-audit failure.
- If a discovery shortlist is pending human approval (`discovery_completed_pending_approval`), do not re-run discovery; re-surface the pending shortlist with an `**[ACTION REQUIRED]**` approval block in every run's report/notification until the human resolves it (offer a refresh only when the human asks or the shortlist is older than 14 days).
- If the Local Collector app is not installed/running/healthy or the matching extension is stale, the Automation Flow agent should run public data sources and list private data sources as pending activation with the exact blocker (`collector_offline_or_unreachable`, `extension_stale`, `wrong_workspace_bridge`, or `collector_status_unverified`). That blocked state is the ONLY one that may leave approved discovery unrun.
- The automation report output must include a mobile-friendly HTML report, a concise summary, and at least one useful draft script/blog/caption.
- If the client's WideCast/OpenAPI provider config is not connected and verified, the operator-only `INTERNAL_REPORT` and chat handoff must include the PDNA setup note so the human sees how the useful report can become video/blog production, 10+ platform distribution, Telegram notifications, performance measurement, and a learning loop after one WideCast setup. Client-facing reports must not include this note.

Manual run / run-now rule:

- In Automation Flow, any human request such as `run now`, `manual run`, `test run`, `trial run`, `collect now`, or `scan now` must bypass recurring schedule windows for the target client.
- The agent must not wait for `scheduled_windows` when the human requested a manual run.
- If the Local Collector app is reachable, the agent must create a run-now job and call `POST http://127.0.0.1:17321/jobs/run_now`.
- The run-now job must include:
  - unique `run_id`,
  - `run_now: true`,
  - `force: false` by default,
  - `run_now_ttl_minutes`, default 30 and maximum 120,
  - private `sources`,
  - pacing rules,
  - client/business/location metadata when available.
- To run again, the agent should create a new unique `run_id` instead of forcing the same run id repeatedly.
- The run-now job must expire automatically if it is not completed, so the extension cannot keep seeing the same manual job all day.
- The Solo Agency Local Collector extension should see `job_available: true` on the next `/status` poll and run immediately.
- If the Local Collector app is not reachable, the agent must not try to start it from inside the AI sandbox during setup/repair. Provide the one-line Local Collector app setup/start command for the human to run outside the sandbox, then retry the run-now job only after the app is reachable.
- Recurring schedule windows are only for unattended scheduled runs. They must not block manual runs.
- Do not simulate a manual run by editing `scheduled_windows` or creating a temporary schedule window. Manual runs must use `/jobs/run_now`.
- If the agent cannot call `http://127.0.0.1:17321` from its own sandbox but can write local files, it must write one unique per-client job file under `daily-content-pipeline/collector/jobs/pending/`. The Local Collector app claims matching pending jobs on `/status`, moves claimed files into `jobs/claimed/`, writes output for that client, then moves completed files into `jobs/completed/`.
- `daily-content-pipeline/collector/run_now_request.json` is a legacy/batch shim only. It may contain one job or `{"jobs":[...]}` and the bridge converts it into `jobs/pending/` queue files. Do not use this single filename when multiple agents or scheduled tasks may write concurrently.
- If the agent cannot call HTTP and cannot write the local queue file, only then create a local run-now helper script or launcher and give the human exactly one short command/path to run it. The helper script must POST `/jobs/run_now` with the correct payload, then optionally poll `/status`.
- Do not ask the human to restart the Local Collector app merely to make a manually edited schedule file take effect. Restarting is only appropriate for updating the Local Collector app itself, recovering a stuck/offline process, or applying an intentional recurring schedule change when both `/config` and file auto-reload are unavailable.
- If a legacy collector without `/jobs/run_now` forces a temporary schedule fallback, the agent must clearly label it as a fallback, back up the original config, create a short unique temporary window, restart or reload only through an already-running service or a human-run setup/start command when required, restore the original config immediately after completion/timeout, and report that fallback to the human. This fallback must not be used when `/jobs/run_now` exists.

Exact manual run-now contract:

- Health-check the Local Collector app first with plain `GET http://127.0.0.1:17321/status`.
- Do not send `X-Collector-Extension` when the AI agent checks health. That header is for the Solo Agency Local Collector extension only. If the AI agent fakes it, `extension_health` can become misleading.
- If `/status` is reachable, call `POST http://127.0.0.1:17321/jobs/run_now`.
- The minimum payload should look like this:

```json
{
  "run_id": "2026-06-20_client-slug_manual_150405",
  "client_slug": "client-slug",
  "business_slug": "business-or-brand-slug",
  "industry": "life insurance",
  "sub_industry": "family protection and retirement planning",
  "target_location": "California, United States",
  "run_now": true,
  "force": false,
  "run_now_ttl_minutes": 30,
  "sources": [
    {
      "name": "Competitor page or private group name",
      "url": "https://www.facebook.com/groups/example",
      "platform": "facebook",
      "source_type": "private_group",
      "purpose": "monitor audience questions, competitor positioning, leads, and content ideas",
      "priority": "high"
    }
  ],
  "pacing": {
    "min_delay_seconds": 5,
    "max_delay_seconds": 5,
    "max_sources": 20,
    "scroll_steps": 5,
    "max_text_chars": 12000
  },
  "collector_policy": {
    "read_only": true,
    "do_not_comment": true,
    "do_not_message": true,
    "do_not_react": true,
    "do_not_exfiltrate_secrets": true
  }
}
```

- `run_id` must be unique for every manual run. A recommended pattern is `YYYY-MM-DD_client-slug_manual_HHMMSS`.
- `run_now` must be `true`.
- `force` must be `false` unless the human explicitly asks for a troubleshooting rerun and understands the same `run_id` may run again.
- `run_now_ttl_minutes` should be 30 by default and must not exceed 120.
- `sources` must contain the private data sources for that client if private data sources exist. If there are no private data sources, the agent should still run public research without the Local Collector app.
- `pacing.scroll_steps` defaults to 5 and must not exceed 10.
- If the agent cannot make this POST itself but can write local files, it should write the JSON payload to one unique per-client queue file:

```text
daily-content-pipeline/collector/jobs/pending/{timestamp}_{client_slug}_{run_id}.json
```

The agent should write this file atomically: write a temporary file in the same folder first, then rename it into `jobs/pending/` only after the JSON is complete.

The running Local Collector app should pick up queued files on the next `/status` check from the matching Chrome extension, usually within a few seconds while Chrome is active. The bridge must:

- write `run_now_request_status.json`;
- move claimed files into `jobs/claimed/`;
- move completed files into `jobs/completed/`;
- clear the active run-now job on `/complete`;
- expire the active run-now job after `run_now_ttl_minutes` if `/complete` never arrives, then allow the next queued client job to proceed.

The single file `daily-content-pipeline/collector/run_now_request.json` remains supported only as a legacy/batch shim. It is safe for one agent to write a batch object with `{"jobs":[...]}`; it is not safe for multiple writers to race on the same filename.

After loading the request, the Local Collector app should write:

```text
daily-content-pipeline/collector/run_now_request_status.json
```

Only if the agent cannot write the request file should it create one of these helper files:
  - macOS/Linux: `daily-content-pipeline/collector/run_private_now.sh`
  - Windows: `daily-content-pipeline/collector/Run Private Collector Now.cmd`
- The human-facing instruction should be one line, for example:

```bash
bash "/ABSOLUTE/PATH/TO/daily-content-pipeline/collector/run_private_now.sh"
```

- After posting `/jobs/run_now` or writing a queue file, do not fake extension headers. Plain `GET /status` is only a bridge health check; `job_available` is scoped to the real client extension identity.
- Track progress through `run_now_request_status.json`, `bridge_health.json`, `collector_status.json`, `jobs/claimed/`, `jobs/completed/`, and new output files under the run output directory until the job completes or TTL expires.

Schedule rule:

- Ask schedule/routine questions after the profile and source plan are known and before the client-specific automation task is marked ready.
- Ask whether the human wants daily, multiple-times-daily, weekly, manual-only, first-run-only, or another cadence.
- Then write or update `schedule.md`, the automation manifest, the scheduled-run prompt/task body, and the relevant automation/config files.
- During Setup Flow, do not ask to run the first agency run immediately and do not run a report. Finish by preparing or resyncing the client-specific automation task whose task name begins with the client name.
- After schedule/routine setup and automation task creation, ask/resolve private data sources once. If private data sources exist and Local Collector is pending, handle step 6: guide Local Collector setup or clearly mark private data sources as `pending_private_activation` in the automation contract so the first automation run can continue with public data sources only if needed, then resync the task.

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
      - Use at least 10 distinct public search keywords per public data source run unless search tooling is unavailable or the saved keyword bank has fewer than 10 usable entries after expansion.
      - At least 7 of the 10 keywords should come from pain-point/problem/need/buying-intent/objection/comparison/question/local-context/trend-news groups.
      - Include at least one broad primary-industry keyword for context, at least one pain-point/problem keyword, at least one need/goal or buying-intent keyword, and local/location keywords when location matters.
      - Use a smaller rotation of related-industry keywords only when the bridge back to the client's offer is clear.
      - If results are weak, try a different pain-point/problem/need cluster before giving up.
      - Continue keyword rotation until at least 3 source-backed candidate ideas are new or newly angled against `history/YYYY-MM/content_log.md`. If fewer than 3 qualify after 10+ distinct keywords and due public data sources have been checked, record the coverage limitation instead of fabricating weak ideas.
      - Record every keyword used, keyword group, result quality, useful URLs, and final keyword status.
      - Extract new keyword candidates from useful search results, public discussions, questions, competitor hooks, comments, and emerging phrases. Add useful new candidates to the keyword bank with source/reason, related pain point, and content pillar.
      - Detect useful recurring public data sources from search results and public pages. Promote strong recurring sources into `public_data_sources` with status/cadence so future scheduled runs can visit them automatically.
      - Include this record in the daily report section `Public Search Keywords Used Today`.
      - If no search was possible, explicitly explain the blocker in that same section.
   7. Before deciding whether to skip private data sources, perform Collector Runtime Verification whenever private data sources exist in any state or collector status files exist. Do not treat saved labels such as `pending_private_activation`, `public_data_sources_only`, or `private sources postponed` as final; those labels can be stale after the human later installs, repairs, or reconnects the Local Collector.
   8. Load `playbooks/PRIVATE_SOURCE_GATE.md`, Stage 2, Stage 8, and Stage 9 before any Collector Runtime Verification involving private data sources.
   9. Try `GET http://127.0.0.1:17321/status`, but if it fails, check local collector health/status files for AI sandbox localhost isolation before claiming the Local Collector is inactive.
      - If the bridge is offline or unverified after checking both `/status` and local health files, do not start it from inside the AI sandbox. Prepare an absolute-path human-run start command, mark the exact blocker, and continue with public data sources.
      - If the bridge is online but `/status.config_file`, `/status.output_dir`, or `/status.run_now_request_file` points outside the current setup's `daily-content-pipeline/collector/` tree, mark `wrong_workspace_bridge`, do not run private collection, ask the human to run the current setup's Local Collector command, and remind them to remove/disable old Solo Agency Local Collector extensions in `chrome://extensions`.
      - If the bridge is online but `extension_health.status` is `stale` or `no_extension_check_yet` after the 75-second extension check grace window, mark private collection as unavailable for this run and notify the human.
      - If the workspace identity check passes and `extension_health.status` is `recent`, continue private collection.
   10. Prepare the private data source queue if private data sources are available and collector health is acceptable:
      - keep the active daily queue around 20 sources or fewer per client by default;
      - prioritize sources most relevant to the client, target audience, target location, pain points, and content pillars;
      - classify extra sources as `weekly` or `optional` and rotate them across future runs;
      - do not run aggressive or parallel private data source scans for the same logged-in account.
   11. Check private data sources if available, using the Solo Agency Local Collector extension plus the Local Collector app when available, with `collector_config.scroll_delay_seconds` defaulting to 5 seconds and `collector_config.max_scrolls_per_source` defaulting to 5.
   12. If the collector bridge was started in `agent_on_demand` mode, stop it after collection completes or after timeout.
   13. Log skipped, pending-activation, expired, rate-limited, warning-triggered, collector-unavailable, extension-unavailable, Chrome-not-running, stale-extension, bridge-offline, or unavailable private data sources.
   14. Load yesterday's private data for this client when available and filter duplicate or near-duplicate data points using visible text matching. Do not parse private-platform HTML for duplicate detection.
   15. Extract relevant `[data_points]`, including reference URLs for every data point. Keep data points that are directly about the primary industry or clearly connected through a related industry. Discard related-industry data when the bridge back to the client's offer is weak.
   16. Add newly recommended private groups/pages/profiles/communities to `New Private Data Sources Detected` and `history/YYYY-MM/new_private_sources_log.md`.
   17. Detect hot and warm leads, including profile URLs, post/current URLs, safe summaries, and reasoning.
   18. Detect direct, adjacent, and audience competitors, including profile URLs, post/current URLs, and positioning notes.
   19. Generate the 3x2 idea matrix as six buckets, not six total ideas. Put every credible, source-backed idea from today's data into the matching layer/scope bucket, and label each idea as `primary_industry` or `related_industry`.
      - Every idea must state the target audience pain point, viewer value/lesson, source signal, non-promotional angle, why it helps the audience, and soft business relevance.
      - Rewrite or reject ideas that mainly praise, position, or advertise the client's product/service as `promotional_not_value_first`.
   20. Check `history/YYYY-MM/content_log.md`, including the recent primary/related ratio and duplicate/near-duplicate idea risk.
   21. Perform the Idea Novelty Check: prefer at least 3 candidate ideas that are new or newly angled. If a prior topic is reused, record the prior idea/date, today's new angle, and why the re-angle is materially different.
   22. Select the best idea of the day only from ideas that pass the Audience Value-First Gate.
   23. Write the configured production-ready draft using Client tools/OpenAPI first, global MCP/native tools only after identity match, or the writing skill fallback if provider/account access is unavailable. Drafts must preserve the viewer-value lesson and must not become direct ads for the client's product/service. Keep writing-method/provider details in `INTERNAL_REPORT`, not client-facing files.
   24. Save `outputs/YYYY-MM/YYYY-MM-DD/{client-name}-daily-report.md` as the internal source-of-truth report.
   25. Generate the three-file scrubbed staging HTML report set under `outputs/YYYY-MM/YYYY-MM-DD/`: `{client-name}-public-data-sources-report.html`, `{client-name}-private-data-sources-report.html`, and `{client-name}-daily-report.html` (staging index).
   26. Create or update the operator-only `{client-name}-INTERNAL_REPORT.html` and copy it to `outputs/latest/{client-name}-INTERNAL_REPORT.html`, labeled `INTERNAL_REPORT - Not for client sharing`.
   27. Run the Client-Blind Scrub Gate on all client-facing files, confirming no Solo Agency, WideCast, PDNA/provider tooling, OpenAPI, MCP, Local Collector, Chrome extension, automation/scheduled task, API-key/config, Telegram, agent/tool/debug details, or `INTERNAL_REPORT` mention remains.
   28. Create or update `outputs/YYYY-MM/YYYY-MM-DD/{client-name}-report_state.json` with reconciled public/private counts and statuses that match the report set.
   29. Generate or update `{client-name}-client-report.html`, `{client-name}-client-report.pdf`, and `outputs/latest/{client-name}-client-report.pdf` from the three scrubbed staging HTML files, or record the exact PDF blocker/status.
   30. Refresh `outputs/latest/{client-name}-client-report.html` as the default handoff/notification/latest link, pointing to the combined client-facing report, not a lane-specific or staging report.
   31. Update or copy `outputs/latest/{client-name}-daily-report.html` as the latest staging index.
   32. Update or copy the latest public/private lane HTML files when those lane reports exist.
   33. Update `history/YYYY-MM/content_log.md`.
   34. Update `history/YYYY-MM/data_sources_log.md`.
   35. Update `history/YYYY-MM/lead_log.md`.
   36. Update `history/YYYY-MM/competitor_log.md`.
   37. Append `history/YYYY-MM/lead_competitor_opportunities.jsonl` with the run's lead/competitor opportunity records.
4. Create or update `outputs/YYYY-MM/YYYY-MM-DD_master_digest.md`.
5. Generate `outputs/YYYY-MM/YYYY-MM-DD_master_digest.html` as a polished standalone human-facing master report.
6. Update or copy `outputs/latest_master_digest.md`.
7. Update or copy `outputs/latest_master_digest.html`.
8. Present the daily digest to the human.
9. If the configured provider notification capability is available, preferably WideCast OpenAPI `sendNotification`, upload the scrubbed client-facing HTML report first when an HTML-capable upload operation such as `uploadAsset` is available, upload the PDF companion too when the verified client provider supports PDF upload, then send a notification to the human/operator that includes the uploaded report URL, PDF companion path/status, INTERNAL_REPORT path/status, run status, clients processed, blockers, lead/competitor counts, and required actions. Treat provider-hosted URLs as operator handoff links, not client-share links.
10. If another authorized channel can send the HTML/PDF files or links more conveniently, use it.
11. Log the notification attempt in `notifications/notification_log.md`.

The daily run is complete only when every active client is processed or explicitly logged as skipped.

When presenting the daily idea list to the human, include reference URLs next to data points, top ideas, and the selected best idea so the human can verify the information. For private data, include the captured source URL and note that it may require the human's logged-in session.

Scheduled runs must assume the human may not be present in the AI agent UI. The run is not fully operationally complete until the scrubbed mobile-friendly client-facing HTML result plus PDF companion path/status plus INTERNAL_REPORT path/status, or a result-ready notification with those paths/statuses, has been sent through the configured notification channel, preferably WideCast OpenAPI Telegram/email fallback when configured for that client.

---

## 16. Setup Repair Mode

If a Client Intelligence Profile file exists but is incomplete, stale, or inconsistent:

1. Infer missing values where possible.
2. Research missing values where possible.
3. Show proposed repairs to the human.
4. Ask the human to correct only what is wrong.
5. Update the Client Intelligence Profile file.
6. Continue the daily run.

Do not discard existing user-provided values unless the human confirms.

---

## 19. Private Data Source Access And Failure Protocol

For private data sources:

- Use only the human's already logged-in Chrome session as accessed through the Solo Agency Local Collector extension plus Local Collector app.
- Do not use Claude in Chrome, Codex/browser tools, Playwright/Puppeteer/Selenium, or any agent-controlled browser.
- Do not request credentials.
- Do not request cookies.
- Do not request OTP.
- Do not attempt to bypass access controls.
- Do not interact socially unless explicitly allowed.

If access works:

- Collect relevant visible data.
- Log the source as checked or collected.

If access fails:

- Skip the source.
- Log `session_expired` or `unavailable`.
- Notify the human through the configured provider notification channel if available, preferably WideCast OpenAPI Telegram/email fallback.
- Tell the human in the agent UI and notification channel:

`I could not access [source name] because the session appears expired or unavailable. I skipped it for today's run. Please log in manually through the browser/session if you want it included in future runs.`

Continue the pipeline with other sources.

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

The agent must also record the notification channel in `schedule.md`. If the client has verified WideCast OpenAPI config and the discovered spec exposes `sendNotification`, record WideCast email+Telegram as the preferred notification channel for scheduled runs, even if Telegram is not connected yet, because WideCast can fall back to email when the account supports it. If WideCast OpenAPI notification is unavailable but Gmail/email is connected, record Gmail/email as the secondary fallback notification channel. If neither is available, record `notification_channel: local_path_only` and tell the human how to connect WideCast API key + Telegram/email fallback or Gmail/email.

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

## 21. Master Digest Format

Root output:

```text
daily-content-pipeline/outputs/YYYY-MM/YYYY-MM-DD_master_digest.md
```

Template:

```md
# Master Daily Digest: YYYY-MM-DD

## Summary

- Active clients:
- Processed:
- Skipped:
- Private data sources needing login:
- Notification channel:
- Notification status:

## Client Outputs

### {Client Name}

- Pipeline folder:
- Output file:
- Best idea:
- Mapped content pillar:
- Reference URLs:
- Hot leads detected:
- Warm leads detected:
- Competitors detected:
- Category:
- Scope:
- Why it matters:
- Approval options:

Top ideas:
- Idea:
  - Reference URLs:

Private data sources skipped:
- Source:
  - Captured URL:
  - Reason:

Leads detected:
- Lead level:
  - Safe summary:
  - Profile URL:
  - Post/current URL:
  - Suggested next action:

Competitors detected:
- Competitor type:
  - Name/Page:
  - Profile URL:
  - Post/current URL:
  - Threat level:
  - Opportunity:

### {Next Client}

...

## Human Actions Needed

-
```

---

## 22. Compliance And Safety

For regulated or sensitive industries, the agent must be careful.

Examples:

- Legal
- Healthcare
- Finance
- Mortgage
- Insurance
- Tax
- Immigration
- Investment
- Employment

The agent must:

- Avoid unsupported claims.
- Avoid guaranteeing outcomes.
- Include disclaimers when appropriate.
- Encourage consultation with a qualified professional when needed.
- Avoid giving personalized legal, medical, financial, or tax advice unless the client is qualified and the script frames it safely.
- Avoid using fear-based manipulation beyond reasonable urgency.
- Avoid exploiting tragedy or private personal information.

Examples of unsafe claims:

- "We guarantee your DUI will be dismissed."
- "You will definitely receive compensation."
- "This investment will make you money."
- "This treatment will cure you."

Safer framing:

- "Depending on the facts, there may be options."
- "Do not assume the first offer is the final answer."
- "Rules vary by state and situation."
- "Talk to a qualified professional before making a decision."

---

## 23. Prompt Examples For Humans

### Start With Zero Clients

```md
I have no clients yet. Set up the root daily content pipeline, then immediately help me set up the first client. Ask only for the minimum required information, infer everything else, and show me the setup summary before saving the client as active.
```

### Add One New Client

```md
Add this client to the daily content pipeline: Nguyen Law, immigration lawyer in San Jose. Private data sources to monitor: [links]. Infer everything else and show me the setup summary before saving.
```

### Add Multiple Clients

```md
I manage these clients. Set up one pipeline for each. Ask only for missing critical information and infer everything else:

1. Smith Law - DUI lawyer - Los Angeles - private data sources: [links]
2. Austin Home Group - real estate agent - Austin, TX
3. Bright Mortgage - home loans - Texas - private data sources: [links]
```

### Run Daily Pipeline

```md
Run the daily content pipeline for every active client in clients_index.md. Produce today's idea lists, selected best ideas, configured production-ready drafts, and the master digest.
```

### Add Private Data Sources Later

```md
Add these private data sources to Smith Law's pipeline: [links]. Do not ask for credentials. If login is required, tell me to log in manually through the browser session.
```

### Add Facebook Member Groups

```md
Ask me whether I want to include Facebook groups where I am already a member as private data sources. If I agree, review the available groups through my logged-in browser session and keep only groups with discussions relevant to the client's primary industry, related industries, audience, location, and pain points. Do not ask for credentials.
```

### Pause A Client

```md
Pause Austin Home Group in the daily content pipeline until I reactivate it.
```

### Reactivate A Client

```md
Reactivate Austin Home Group and include it in future daily runs.
```

---

## 24. Media Agency Operating Layer

The daily idea/script workflow plus approved asset creation is the core production engine, but a media agency needs more than daily ideas. The agent must support an agency operating layer around strategy, planning, video/blog/social asset production, approval, publishing, performance, and client communication.

This layer should be added gradually. Do not block the first daily run just because every agency file is not perfect yet. Infer first, show the human, then save and improve over time.

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

### 23.3 Approval Workflow

The agent should maintain:

- `approvals/approval_log.md`

Approval statuses:

- `drafted`
- `needs_client_review`
- `approved`
- `revision_requested`
- `rejected`
- `ready_for_video`
- `video_created`
- `ready_to_publish`
- `published`

The agent must never assume approval. It must ask for explicit approval before:

- Creating a production asset through a connected provider.
- Rendering/exporting a video.
- Publishing.
- Spending credits.
- Posting or commenting from a social account.

Example approval log:

```md
| Date | Asset | Client | Status | Approved By | Notes |
|---|---|---|---|---|---|
| 2026-06-20 | outputs/2026-06-20.md | Smith Law | needs_client_review |  | Waiting for script approval |
```

### 23.4 Asset Library And Reuse

The agent should maintain:

- `assets/asset_index.md`

Track:

- Logos.
- Brand colors.
- Fonts.
- Headshots.
- Office photos.
- Product photos.
- B-roll links.
- Prior videos.
- Testimonials.
- Disclaimers.
- Approved CTAs.

For each asset:

- File path or URL.
- Usage rights.
- Client.
- Platform fit.
- Notes.

The agent should reuse approved assets before inventing new visual directions.

### 23.5 Publishing And Distribution

The agent should maintain:

- `publishing/publishing_log.md`

The agent should adapt approved content per platform:

- TikTok: fast hook, native caption, concise CTA.
- Instagram Reels: hook + caption + hashtags if useful.
- YouTube Shorts: searchable title, description, retention-focused script.
- LinkedIn: professional framing, perspective, business context.
- Facebook: local/community tone when appropriate.

The agent must not publish automatically unless the human has explicitly authorized publishing for that specific content and platform.

Publishing log should include:

- Date.
- Platform.
- Post URL.
- Caption.
- Video/script source.
- Status.
- Notes.

### 23.6 Repurposing System

The agent should turn one approved idea into multiple assets when useful:

- Short video script.
- LinkedIn post.
- Facebook post.
- X/Twitter thread.
- Blog outline.
- Newsletter blurb.
- Carousel outline.
- FAQ snippet.
- Sales email angle.

Repurposing must preserve the same factual references and reference URLs. If the claim changes, the agent must verify and attach a new reference URL.

### 23.7 Community, Lead, And Competitor Handling

The agent may monitor comments, questions, and community discussions if tools allow it, but must not reply, message, comment, or engage from the account without explicit permission.

The agent should extract:

- Repeated questions.
- Objections.
- Complaints.
- Buying signals.
- Local concerns.
- Competitor messaging patterns.
- Lead-intent signals.
- Newly discovered direct competitors.
- Adjacent competitors that solve the same pain points.
- Audience competitors that capture the same audience's attention.

For potential leads, the agent should log only safe summary information and source URLs. It must not expose unnecessary private personal data.

For detected competitors, the agent should log only public or authorized visible information, source URLs, positioning patterns, content themes, engagement signals, and strategic opportunities.

Competitor analysis must be used for strategy, positioning, and original content ideas. The agent must not copy competitor posts, scripts, captions, offers, or creative assets.

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

### 23.10 Client Communication

The agent should produce client-facing summaries when useful:

- Daily digest.
- Weekly content plan.
- Monthly performance report.
- Approval request.
- Revision summary.
- Source/evidence appendix.

Client-facing communication should be concise and decision-oriented:

- What was found.
- What is recommended.
- Why it matters.
- What needs approval.
- What happens next.

### 23.11 Account Growth And Retention

For agency operations, the agent should periodically identify:

- Clients with missing setup data.
- Clients with weak or stale content pillars.
- Clients with low publishing cadence.
- Clients whose private data sources need login refresh.
- Clients with no performance data.
- Clients with strong-performing pillars worth doubling down on.

The agent should not upsell automatically, but it may prepare recommendations such as:

- "This client needs more local data sources."
- "This pillar is producing the strongest engagement."
- "This account needs a new approval workflow."
- "This client is ready for a monthly report."

### 23.12 Agency Operating Principle

The agent must treat content production as a loop:

```text
Research -> Insights -> Content pillars -> Ideas -> Script -> Approval -> Production -> Publishing -> Analytics -> Learning -> Better research
```

The daily pipeline is not just for generating ideas. It is how the agency learns what each client's audience cares about and improves the next day's content.

---

## 25. Expected Agent Behavior In A New Environment

When a new AI agent receives this playbook, the human may say:

```md
Read and follow SOLO_AGENCY_PLAYBOOK.md exactly. Start by asking me only for the minimum setup information.
```

The correct first response from the agent should be similar to:

```md
What product/service, profession, expertise, business description, or public website/profile URL should this pipeline focus on? If location matters, include the target location. I can use a public website/profile URL to understand the business if you prefer. I will infer industry, audience, pain points/customer problems, content pillars/main content themes, and public data sources/web-search sources I can access without your login, then show you the setup summary before saving anything as stable context.
```

If space allows, the first response should mention that the agent will also infer related industries and keep content focused around an 80% primary / 20% related-industry mix.

If the human says they have no clients yet, or if the first run discovers that `clients_index.md` has no real client rows, the agent should create or verify the root structure and immediately enter First Client Setup Mode. It should ask only for the first client's name and product/service, profession, expertise, business description, or public website/profile URL, plus target location only if location matters and cannot be inferred. It must not mention private data sources in the first client setup question. Private data sources are asked once after schedule/routine and the client-specific automation task have been configured; that step is where the agent may ask for actual private data source URLs/lists, offer optional discovery from approved joined groups, subreddits, communities, followed profiles/pages/KOLs, subscribed channels, and feeds, get approval, handle Local Collector activation if needed, and resync the task afterward.

If the human gives a new client, the agent should enter Add Client Mode.

After Add Client Mode or First Client Setup Mode, the agent must follow the fixed order: setup context, configure schedule/routine, prepare or resync the client-specific automation task, resolve the step 6 private data source/Local Collector checkpoint if private data sources are pending or requested, resync the task after any source-state change, then hand off the exact task name the human should run for the first report. Setup Flow must not jump into report generation, video creation, publishing, or production actions.

The agent must summarize the first report and any required next action directly in chat. It must provide the client-facing HTML report path/link as the primary review link plus the PDF companion path/status plus the operator-only `INTERNAL_REPORT` path/status. It must not make the human open a Markdown file to review the report, activate private data sources, run setup, fix a blocker, or choose the next step.

If the human asks for daily output, the agent should process all active clients in `clients_index.md`.

---

## 26. Completion Criteria

Initial setup is complete when:

1. The root folder exists.
2. `clients_index.md` exists.
3. Each configured client has a pipeline folder.
4. Each configured client has a Client Intelligence Profile file.
5. Each configured client has initial strategy files or planned placeholders for offer map, brand voice, content pillars, and funnel map.
6. Inferred/researched setup context has been shown to the human step by step.
7. Inferred related industries, content pillars, and the 80% primary / 20% related-industry content mix rule have been shown to the human.
8. Human corrections have been applied.
9. The recurring schedule/routine has been configured or explicitly marked manual-only/pending.
10. The client-specific automation task exists or is proposed for the native automation system, and its task name begins with the client name.
11. The scheduled-run prompt/task body pins `target_client_slug`, the expected client extension identity, source state, output path, and the Setup Flow / Automation Flow contract.
12. The matching client extension folder exists under `extensions/{client_slug}/`, with the client name at the start of the Chrome extension name, and the setup handoff shows the absolute folder path plus the exact Chrome `Load unpacked` steps for the matching client Chrome profile/account.
13. If private data sources exist but Local Collector is not active yet, the automation contract includes `Private Data Sources Pending Activation` and lists the pending sources.
14. If no private data sources were provided, the agent offered optional private data source discovery or recorded that discovery was declined/postponed.
15. If no private data sources are active, the automation contract includes `Private Data Source Discovery Recommended` or `Private Data Source Discovery Declined/Postponed`, with a plain note that public-only reports can miss community, lead, and competitor signals.
16. If the client's WideCast/OpenAPI provider config is not connected and verified, the automation report contract requires the PDNA/WideCast setup note in `INTERNAL_REPORT` and the operator handoff, not in client-facing reports.
17. If the human agrees to activate private data sources, `daily-content-pipeline/collector/collector_setup_status.md` exists and shows either `installed_and_running` or a precise blocked status with the required human action.
18. The setup handoff tells the human the exact client-specific automation task name to run for the first report.
19. Any required human action is also shown directly in the current chat message with one clear command, one double-clickable launcher path, or one absolute extension folder path. Markdown-only setup instructions are a failure.
20. Only after the first agency report and draft are shown does the agent ask whether to set up PDNA - Production, Distribution, Notification, and Analytics.

Recurring schedule setup is complete when:

1. `schedule.md` exists.
2. The human has chosen a recurring cadence, first-run-only mode, or manual-only mode before the client-specific automation task is marked ready, after the profile and source plan are known.
3. If any active client has private data sources, the schedule explains whether private collection is activated, declined for now, or waiting on Local Collector setup.
4. The schedule or manual run process is documented.
5. The client-specific automation task name begins with the client name and its prompt pins `target_client_slug`.
6. The configured notification channel is documented.

A daily run is complete when:

1. Every active client has been processed or explicitly skipped.
2. Source checks are logged.
3. Data points are collected.
4. Hot and warm leads are detected, listed, or explicitly marked as none found.
5. Direct, adjacent, and audience competitors are detected, listed, or explicitly marked as none found.
6. A separate 3x2 public idea matrix and private idea matrix are created for each processed client when the corresponding lane has data, or the lane states why it is pending/skipped/blocked. Each matrix is six buckets, not six total ideas; every credible, source-backed idea harvested for that lane today should appear in the matching bucket.
7. One best public idea and one best private idea are selected for each processed client when data exists, plus any overall recommendation if useful.
8. Each idea maps to a content pillar when possible.
9. Each idea is labeled as `primary_industry` or `related_industry`, with a visible related-industry note and bridge-back logic shown for related-industry ideas.
10. One production-ready draft is written for each processed client, defaulting to video script and adding blog/article or social caption when configured.
11. One per-client canonical three-file client-facing HTML report set is created for each processed client: `{client-name}-public-data-sources-report.html`, `{client-name}-private-data-sources-report.html`, and `{client-name}-daily-report.html`.
12. The operator-only `{client-name}-INTERNAL_REPORT.html` is created for each processed client and clearly labeled `INTERNAL_REPORT - Not for client sharing`.
13. The client-facing HTML report set passes the Client-Blind Scrub Gate and does not mention Solo Agency, WideCast, PDNA/provider tooling, OpenAPI, MCP, Local Collector, Chrome extension, automation/scheduled task, API-key/config, Telegram, agent/tool/debug details, or `INTERNAL_REPORT`.
14. The mandatory PDF companion `{client-name}-client-report.pdf` is created from the combined `{client-name}-client-report.html`, which itself is assembled from the scrubbed three staging HTML files, or the exact PDF blocker/status is recorded.
15. The report state file `outputs/YYYY-MM/YYYY-MM-DD/{client-name}-report_state.json` is created/updated for each processed client.
16. `outputs/latest/{client-name}-client-report.html`, `outputs/latest/{client-name}-INTERNAL_REPORT.html`, and `outputs/latest/{client-name}-client-report.pdf` are updated for each processed client when available and point to the combined report/internal report/PDF companion, not a lane-specific report.
17. Client history is updated, including industry scope for selected ideas so the 80/20 mix can be tracked over time.
18. Lead and competitor logs are updated.
19. Approval status is tracked.
20. Markdown and mobile-friendly HTML master digests are created when a master digest task is configured.
21. `latest_master_digest.md` and `latest_master_digest.html` are updated when a master digest task is configured.
22. Client-facing reports are written in the client's report language; operator notifications/internal reports are written in the human/operator language.
23. The human/operator is notified through the configured notification channel, preferably WideCast OpenAPI Telegram/email fallback, with the `{client-name}-client-report.html` path/link plus PDF companion path/status plus `INTERNAL_REPORT` path/status. Public and private notifications are both allowed, but they should point to the same combined report path or uploaded operator-delivery URL. Daily/public/private staging links should be omitted unless requested for diagnostics. The Markdown report path must not be presented as a user-facing report link.
24. Human approval options are shown.
25. Stage 10 (`playbooks/10_LEAD_COMPETITOR_DETECTION.md`) was loaded IN FULL (LOAD LEDGER printed) before lead/competitor detection, or an explicit skip reason was recorded.

An agency operating cycle is complete when:

1. Approved content is tracked in the calendar.
2. Assets and references are organized.
3. Publishing status is logged.
4. Verified provider analytics, preferably WideCast OpenAPI, are checked for recently published content URLs, metadata, and account/platform analytics when available.
5. Performance metrics are captured when available, reusing the Solo Agency Local Collector extension plus Local Collector app for published URL measurement when possible.
6. Reports or client-facing summaries are produced on the chosen cadence in the human's language.
7. Important results, blockers, and required actions are pushed to the human through the configured notification channel.
8. Mobile-friendly HTML reports are generated for review when useful.
9. Learnings are fed back into content pillars, source strategy, and future ideas.

---

## 27. Final Agent Self-Audit Checklist

The agent must use this checklist before replying to the human, before claiming setup is complete, and before claiming a daily run is complete.

This checklist exists because the playbook is intentionally comprehensive. Long instructions are easy to partially miss. The agent must actively check for omissions instead of relying on memory.

### Response Self-Audit Checklist

Before replying to the human, verify:

- [ ] Did I answer in the same language the human used?
- [ ] Did I explain marketing/tech terms in plain language when they appear in human-facing text, especially public data sources, private data sources, Local Collector, offer, pain points, content pillars, lead, competitor, idea matrix, HTML report, draft, PDNA, analytics, and learning loop?
- [ ] If this reply asked the step-6 private data source checkpoint question, did I load Stage 2 first and deliver it in two parts (plain-language explanation BEFORE the `**[ACTION REQUIRED]**` question), content-complete per the Stage-2 §6 checklist: private definition + examples, public contrast, Local Collector + local-only + never asks passwords/cookies/OTPs/tokens, already-a-member requirement, the hands-free discovery option (no hand-compiled list needed), and the three reply options?
- [ ] Did I separate human/report language from target-audience keyword/content language when they differ?
- [ ] Did I avoid asking for information I can infer, research, or discover myself?
- [ ] If I asked a question, did I first show what I inferred from the previous answer?
- [ ] Did I show setup or research assumptions clearly instead of hiding them in files?
- [ ] If the setup, daily run, private data source setup, production setup, scheduling, publishing, or measurement workflow is not complete, did I show an updated progress block in this reply?
- [ ] If schedule/automation already exists and this reply includes a progress block, did I include an `Automation freshness check` stating whether the latest changes are synced into the automation/scheduled task prompt/contract/playbook/source state, not only config, and whether tomorrow's run will load the newest state?
- [ ] If I am handing control back to the human while required steps remain, is the final line exactly one concrete next-step question?
- [ ] If human action is needed, did I show the exact action directly in chat or notification?
- [ ] If human action is needed, did I use the root playbook `**[ACTION REQUIRED]**` block instead of burying the question/action in paragraphs, reports, file links, or progress text?
- [ ] If no human action was required, did I end with next-action guidance per the Next-Action Guidance Rule - 1-3 real, currently-available next steps (the first resuming the current or interrupted flow) plus exactly one closing question - instead of `No action required right now.` or a passive ending?
- [ ] Did I keep the most important required action at the end of the reply, with no more than three `**[ACTION REQUIRED]**` blocks?
- [ ] Did I avoid telling the human to open a Markdown file for instructions?
- [ ] If I am about to report a blocker, repeated failure, unclear contradiction, stale artifact, missing capability, or dead end, did I first run Last-Resort Recovery by checking GitHub `main` for newer Solo Agency playbooks/code and reloading the latest relevant instructions?
- [ ] If the latest GitHub version still did not resolve the blocker, did I create, send, or draft a redacted issue without requiring the human to have a GitHub account, record the issue URL/number, intake channel, or draft path in `daily-content-pipeline/automation/github_issues.md`, and tell the human how it will be tracked?
- [ ] If I mentioned a report, did I provide the HTML path/link as the primary human review link, include the PDF companion path/status, and avoid showing the Markdown report path?
- [ ] If I mentioned a report and any workflow step remains, did I include both the progress block and the required next-step question in chat instead of relying on the report's `Next Action` section?
- [ ] If I checked tools/capabilities or claimed a tool was available/unavailable, did I check Client tools first (`provider_config.local.json`, OpenAPI cache/spec, verified identity, `provider_capabilities.json`) and global MCP/native tools only second?
- [ ] If video creation/render/export was requested and client-scoped PDNA provider setup was missing or blocked, did I stop at script/storyboard/production-brief work, explain the provider requirement, and use a `**[ACTION REQUIRED]**` block instead of creating local video media?
- [ ] Before any video provider creation request, did I treat report scripts as reference only and create a final WideCast-grade script/brief by loading and applying the existing video script-writing skill, including research and inline-media/direct-image-URL workflow where verifiable?
- [ ] If a report version/code or automation recommendation already existed, did I process only that selected version and avoid generating a second five-version set?
- [ ] In Setup Flow, did I avoid running the first agency run/report directly and instead prepare or resync the client-specific automation task?
- [ ] In Automation Flow, did I avoid jumping to the first report before private data source status, the step 6 Local Collector checkpoint, schedule/routine, and client-specific automation task were resolved or honestly marked pending?
- [ ] If I generated or announced an HTML report, did I generate/update the mandatory PDF companion or record the exact PDF blocker/status?
- [ ] If I generated or announced an HTML report, did I run the Stage 6 Provider Report Delivery Capability Check: inspect Client tools first, verify the configured provider/OpenAPI spec and account identity, attempt upload/notification when available, log exact blockers when unavailable, and provide the HTML report path/link plus PDF companion path/status?
- [ ] If WideCast upload/Telegram was skipped, did I check Client tools first before treating legacy/global MCP/native tool absence as a blocker?
- [ ] Did I avoid asking for credentials, cookies, passwords, OTPs, or tokens?
- [ ] Did I avoid calling the collector a Facebook collector?
- [ ] If the human asked for any private data source scan after conversation drift, including logged-in/account-required groups, feeds, profiles, pages, communities, or sources, did I reload `playbooks/PRIVATE_SOURCE_GATE.md`, Stage 2, Stage 8, and Stage 9 before acting?
- [ ] If I skipped private data sources during a scheduled/manual run, did I perform Collector Runtime Verification instead of trusting saved config labels such as `public_data_sources_only`, `private sources postponed`, or `pending_private_activation`?
- [ ] If a discovery shortlist was pending human approval, did this reply/run re-surface it with an `**[ACTION REQUIRED]**` approval block instead of leaving it buried in an older report?
- [ ] Did every candidate on a discovery shortlist carry a recorded relevance score (target_audience_fit, location_fit per the Stage-0 location-weighting rule, matched_pain_points, industry_scope) against the Client Intelligence Profile, so no unscored or off-vertical/wrong-location source reached the shortlist?
- [ ] Did I present the discovery shortlist as a numbered list directly in chat with a by-number `**[ACTION REQUIRED]**` approval block, instead of telling the human to open a `.md`/report/log to read or approve it?
- [ ] Did the shortlist prefer groups/communities and reputable organization pages and default-skip individual personal profiles (`skip_individual_profile`), keeping an individual account only with recorded high-reach-authority evidence — not filling the list with low-signal personal profiles that are really leads?
- [ ] Did the discovery job capture broadly (discovery `max_sources` sized to the full joined/followed list, not the ~20 active-monitoring cap), and did I classify over ALL captured candidates rather than a truncated sample?
- [ ] Did I record `capture_status`/`captured_count` per surface, tell the human honestly when capture was `capped_incomplete`, and make every count I reported match the captured records on disk (no invented totals like a group count that appears in no file)?
- [ ] If the human named a specific source missing from the capture, did I run a targeted Local Collector search for it instead of speculating about why it was missing?
- [ ] If `GET http://127.0.0.1:17321/status` failed, did I check local collector health/status files for sandbox-localhost isolation before claiming the Local Collector was inactive?
- [ ] In a scheduled/automation run, did I use the file-based job queue as the primary path and AVOID telling the human the collector was down/stopped/unresponsive/needs-restart from a localhost failure — reporting a collector blocker only when the file-queue path itself failed (files missing/stale/wrong-workspace or a job not consumed within TTL)?
- [ ] If this involved private data sources, did I avoid Claude in Chrome, Claude Chrome Extension, Codex/browser tools, Playwright/Puppeteer/Selenium, fresh agent-opened browser profiles, and all other agent-controlled browsers?
- [ ] If this was one-time Local Collector setup/update/repair, did I avoid running `setup_collector.sh`, `setup_local_collector.ps1`, `Start Local Collector.cmd`, or the collector binary from inside the AI sandbox?
- [ ] If this was one-time Local Collector setup/update/repair, did I give the human both required local actions in chat: run the one-line Terminal/PowerShell setup/start command outside the AI sandbox and load the Chrome extension from the absolute runtime folder?
- [ ] If I discussed private data source setup, Local Collector activation, or private data source discovery, did I reassure the human about one-time professional setup patience, local-only data safety, and daily scanning coverage?
- [ ] Did I mention blockers clearly, with the next action if any?

### Client Setup Self-Audit Checklist

Before saving a Client Intelligence Profile as stable, verify:

- [ ] Did I ask first only for product/service, profession, expertise, business description, or public website/profile URL?
- [ ] Did I avoid mentioning private data sources in the first setup or first add-client question?
- [ ] Did I infer industry and sub-industry myself?
- [ ] Did I infer target audience?
- [ ] Did I infer target location, or ask only if location matters and is missing?
- [ ] Did I infer pain points?
- [ ] Did I infer content pillars and content angles?
- [ ] Did I infer related industries?
- [ ] Did I show the 80% primary industry / 20% related industries rule?
- [ ] Did I explain that public data sources are websites/search/public pages I can access without the human's login?
- [ ] Did I use canonical source terms in human-facing text: `public data sources` and `private data sources`?
- [ ] Did I treat step 5 as schedule/routine plus client-specific automation task setup, not as a private data source preference gate?
- [ ] Did I avoid asking for private data source URLs/lists, discovery details, or Local Collector setup until step 6 after automation exists?
- [ ] If the human wanted private data sources or was unsure, did I handle actual source intake/discovery/approval in step 6, including the optional discovery pass from approved joined groups, subreddits, communities, followed profiles/pages/KOLs, subscribed channels, and feeds, then resync the automation task?
- [ ] Before asking the step-6 checkpoint question, did I load `playbooks/PRIVATE_SOURCE_GATE.md` and Stage 2 in full, and follow the Stage-2 §6 two-part delivery rule instead of compressing the explanation away?
- [ ] If the human approved discovery and the collector plus matching extension were verified healthy in-session, did I run the step-6 discovery pass right there and get the shortlist approved (or record exactly why it was deferred as `approved_pending_first_scan`)?
- [ ] Did I ask the step-7 PDNA notification question during setup with the value-first framing (hot-lead alerts, report-ready, drafts awaiting review), without pressure language and without implying any Solo Agency-provider affiliation - or record the decline as `notification_channel_missing` for the once-per-run re-offer?
- [ ] If this was Add Client Mode or First Client Setup Mode, did I create or verify a dedicated `extensions/{client_slug}/` folder, patch the Chrome extension name to `{Client Name} - Solo Agency Collector`, and show the absolute folder path plus exact `chrome://extensions` -> Developer mode -> `Load unpacked` steps for the matching client Chrome profile/account?
- [ ] Did I avoid merely saying "I created the extension" or "extension folder exists" without the path and install steps?
- [ ] Did I build a public keyword bank from pain points, problems, needs, objections, buying triggers, and local context, not only generic industry terms?
- [ ] Did I choose keyword language based on the target audience's likely search/comment language, not automatically the human's chat language?
- [ ] If the audience is multilingual, did I label keyword languages and include useful variants?
- [ ] Did I show only a compact pain-point keyword sample to the human and save the full keyword bank in the client profile/source notes?
- [ ] Did I save useful recurring public data sources to `public_data_sources` with status, cadence, language, related pain point, and `visit_in_scheduled_runs`?
- [ ] Did I avoid asking a separate private data source discovery checklist question and instead keep optional private data source discovery inside the private data source step?
- [ ] Did I reassure the human that this is a professional agency-scale setup that normally takes patience only once?
- [ ] Did I reassure the human that private data stays local on their computer and must not be sent outside without explicit approval?
- [ ] Did I reassure the human that daily scanning helps avoid missing market signals, leads, competitor moves, and content ideas?
- [ ] Did I mention common discovery surfaces in plain language, including Facebook groups where the human is already a member, subreddits/communities they joined, and profiles/pages/KOLs/channels they follow?
- [ ] If the human agreed to Facebook member-groups discovery, did I use `https://www.facebook.com/groups/joins/?nav_source=tab&ordering=viewer_added` as the discovery source through the Solo Agency Local Collector?
- [ ] If the human agreed to Facebook keyword group search discovery, did I use `https://www.facebook.com/search/groups/?q={url_encoded_keyword}` with client-relevant keywords, Local Collector only, and 10 scrolls per keyword?
- [ ] Did I filter Facebook keyword search results to real group candidates and remove UI noise, people/pages/posts/events results, ads/sponsored blocks, and irrelevant broad groups?
- [ ] Did I avoid joining groups, requesting access, messaging admins, or adding keyword-search groups to active `private_data_sources` before human approval?
- [ ] If Facebook member-groups discovery was approved but Local Collector was not active yet, did I mark it as `pending_private_activation` instead of silently skipping it?
- [ ] Did I avoid adding all joined Facebook groups automatically and instead filter by client relevance before asking the human to approve recommended groups?
- [ ] Did I show which private data sources are daily, weekly, or optional?
- [ ] Did I show public data sources and public search keyword ideas?
- [ ] Did I let the human correct only what is wrong?
- [ ] Did I save the profile only after showing the setup summary?

### Public Research And Keyword Rotation Checklist

Before completing public research, verify:

- [ ] Did I load `public_search_keywords` from the client profile?
- [ ] Did I load saved `public_data_sources` and visit/check active due sources?
- [ ] Did I use Google Search or an available equivalent search tool?
- [ ] Did I use keywords in the target audience's likely search/comment language?
- [ ] Did I use at least one primary-industry keyword?
- [ ] Did I use at least one local/location keyword if location matters?
- [ ] Did I use at least one pain-point/problem keyword?
- [ ] Did I use at least one need/goal or buying-intent keyword?
- [ ] Did I optionally use one related-industry keyword if useful?
- [ ] Did I use at least 10 distinct public search keywords, or document why search/tooling/keyword-bank limits made that impossible?
- [ ] Did at least 7 of those keywords come from pain-point/problem/need/buying-intent/objection/comparison/question/local-context/trend-news groups?
- [ ] Did I keep rotating keyword clusters until I found at least 3 source-backed candidate ideas that are new or newly angled, or document why that minimum could not be met?
- [ ] Did I rotate keywords instead of reusing only old queries?
- [ ] Did I record each keyword as `used`, `useful`, `weak`, or `retry_later`?
- [ ] Did I extract new keyword candidates from useful search results, public discussions, private scans, competitor hooks, comments, analytics, or human feedback?
- [ ] Did I add non-duplicate useful new keywords to the saved keyword bank with source/reason, related pain point, and content pillar?
- [ ] Did I detect useful recurring public data sources and promote/demote them in `public_data_sources` for future scheduled visits?
- [ ] Did I save useful URLs as references?
- [ ] Did I show search keywords used in the report?
- [ ] If I forgot to show search keywords, did I update the current report instead of only promising to show them next time?
- [ ] If public search was skipped, did I explicitly explain why?

### Private Collector Checklist

Before claiming private data sources were collected, verify:

- [ ] Did I reload `playbooks/PRIVATE_SOURCE_GATE.md`, Stage 2, Stage 8, and Stage 9 for this private data source turn?
- [ ] Did I use only the Solo Agency Local Collector extension plus Local Collector app, not Claude in Chrome, Codex/browser tools, Playwright/Puppeteer/Selenium, or another agent-controlled browser?
- [ ] Is the Local Collector app running?
- [ ] Did I tell the human private data sources require the matching client Chrome profile to already be logged in and authorized as a member/follower/subscriber or otherwise allowed viewer?
- [ ] Did I recommend one separate Chrome profile per client with that client's extension loaded and relevant social accounts logged in?
- [ ] Did I verify `/status.config_file`, `/status.output_dir`, and `/status.run_now_request_file` point to the current setup's `daily-content-pipeline/collector/` tree, not an older Solo Agency setup folder?
- [ ] If `/status` showed a running bridge from another setup folder, did I mark `wrong_workspace_bridge`, avoid private collection, and give the human the current setup's one-line Local Collector setup/start command?
- [ ] If an old bridge or old extension install was detected, did I remind the human that one machine should have one active Solo Agency Local Collector runtime and ask them to remove/disable old Solo Agency Local Collector entries in `chrome://extensions`?
- [ ] Is the Solo Agency Local Collector extension recent, not stale?
- [ ] Did I avoid Claude Chrome Extension for automated private collection?
- [ ] If this is one-time setup/update/repair, did I avoid starting/restarting the Local Collector app from inside the AI sandbox and instead provide the human-run setup/start command?
- [ ] If the bridge failed with `address already in use` or `/status` showed stale/wrong config, did the setup/start script or an explicit human-approved troubleshooting path handle the restart by stopping only old `collector-bridge` processes on port `17321` before starting the newest executable?
- [ ] For manual run, did I use `/jobs/run_now` or one unique per-client job under `daily-content-pipeline/collector/jobs/pending/`, leaving `run_now_request.json` only as a legacy/batch shim?
- [ ] For Facebook joined-groups discovery, did I use a manual `run_now` job for `https://www.facebook.com/groups/joins/?nav_source=tab&ordering=viewer_added` instead of pretending the joined groups were manually provided?
- [ ] For Facebook keyword group search discovery, did I create a discovery job with `purpose: "facebook_group_keyword_search_discovery"`, `discovery_category: "keyword_search_sources"`, and `pacing.scroll_steps: 10` per keyword URL?
- [ ] After Facebook joined-groups discovery, did I filter groups by client relevance and ask the human to approve recommended groups before adding them to active `private_data_sources`?
- [ ] After Facebook keyword group search discovery, did I show keywords searched, candidate counts, recommended groups, noisy/skipped examples, membership/access notes, and ask for approval before activation?
- [ ] For optional private data source discovery, did I use only approved discovery categories and platform starting URLs?
- [ ] Did I treat feeds such as Facebook Home, YouTube Home, X Home, LinkedIn Feed, Instagram Explore, TikTok For You, and Reddit Home as discovery surfaces rather than permanent private data sources?
- [ ] Did I avoid collecting DMs, inboxes, notifications, payment/account pages, or unrelated personal data?
- [ ] Did I ask the human to approve discovered sources before adding them to active `private_data_sources`?
- [ ] Did I avoid faking manual run by editing schedule windows?
- [ ] Did I respect max scrolls: default 5, maximum 10?
- [ ] Did I wait 5 seconds between scrolls?
- [ ] Did I avoid scanning too many private data sources at once?
- [ ] Did I capture source URL and current URL?
- [ ] Did I save snapshot or visible capture for audit?
- [ ] Did I mark expired sessions, captcha, warnings, or blocked sources clearly?
- [ ] Did I notify the human via WideCast/Telegram if private collection is blocked and that channel is available?

### Data Quality Checklist

Before using collected data, verify:

- [ ] Did I remove obvious duplicate data from yesterday?
- [ ] Did I avoid parsing private-platform HTML as the main source of truth?
- [ ] Did I keep reference URLs for every important data point?
- [ ] Did I separate public data from private data?
- [ ] Did I identify weak or noisy data honestly?
- [ ] Did I avoid treating UI junk as real source/content?
- [ ] Did I keep low-confidence items out of main recommendations?

### Report Merge Checklist

Before generating, updating, or notifying a report, verify:

- [ ] Is there exactly one canonical combined client-facing report for this client/day/run?
- [ ] Does the report set include staging files `{client-name}-public-data-sources-report.html`, `{client-name}-private-data-sources-report.html`, and `{client-name}-daily-report.html`, plus the delivered `{client-name}-client-report.html`?
- [ ] Did I read the existing source/state file and `outputs/YYYY-MM/YYYY-MM-DD/{client-name}-report_state.json` before updating a lane report?
- [ ] If this is a public pass, did I update only the public report and daily index while preserving any existing private report, then rebuild the combined client report/PDF?
- [ ] If this is a private pass, did I update only the private report and daily index while preserving the public report, then rebuild the combined client report/PDF?
- [ ] After a private pass reached a terminal state, did I reconcile private status and counts across the private report, daily index, combined client report, internal Markdown/source record, report state JSON, notification log, and `outputs/latest/` copies?
- [ ] Did I remove stale `scan in progress`, `partial`, `pending`, or old count text from all artifacts after the private scan completed or failed?
- [ ] Do private source attempted/completed/blocked counts, data point counts, lead counts, competitor counts, and recommended-source counts match everywhere they are shown?
- [ ] Did `outputs/latest/{client-name}-client-report.html` point to the combined client-facing report, not to a lane-specific or daily staging report?
- [ ] If I sent two notifications, did both point to the same combined client report path or uploaded URL, without lane-specific staging links unless requested for diagnostics?
- [ ] Did I log lane status as `public_report_ready`, `private_report_ready`, `private_report_blocked`, or `daily_report_ready`?

### Idea Generation Checklist

Before selecting the best idea, verify:

- [ ] Did I create the 3 sections: Hot/Trend/News, Evergreen/Foundation, Lead-Gen / Conversion?
- [ ] Did I consider both global and local scale?
- [ ] Did I treat the 3x2 matrix as six buckets, not six total ideas?
- [ ] Did I include every credible, source-backed idea harvested from today's public data sources or private data sources in the matching bucket, even when one bucket has 3-5+ ideas?
- [ ] Did I allow empty buckets if no good data exists?
- [ ] Did I label each idea as `primary_industry` or `related_industry`?
- [ ] If related industry, did I explain the bridge back to the client offer?
- [ ] Did every idea map to a pain point or content pillar?
- [ ] Did every idea state a viewer value/lesson that helps the target audience even if they never buy from the client?
- [ ] Did I keep the client's product/service name out of the idea premise unless the idea also contains a standalone educational lesson?
- [ ] Did I reject or rewrite client-praise/direct-promo ideas as `promotional_not_value_first` instead of letting them enter the matrix?
- [ ] Did important ideas include a value-first status such as `pass`, `rewritten`, or `promotional_not_value_first`?
- [ ] Did every important idea include reference URLs?
- [ ] Did I check history to avoid repeating old ideas?
- [ ] Did I label each important idea's novelty status as `new`, `new_angle`, `near_duplicate_rejected`, or `repeat_rejected`?
- [ ] If I reused a prior topic, did I record the prior idea/date and explain the materially new angle?

### Best Idea Selection Checklist

Before choosing the best idea, verify:

- [ ] Did I compare heat/trend strength?
- [ ] Did I check whether this idea was already used?
- [ ] Did I reject exact repeats and near-duplicates unless there was a genuinely new angle?
- [ ] Did I evaluate impact on target audience?
- [ ] Did I evaluate audience size and scope?
- [ ] Did I evaluate lead potential?
- [ ] Did I ensure it logically matches target audience and pain points?
- [ ] Did I choose only from ideas that pass the Audience Value-First Gate, not from ideas that mainly promote, praise, or position the client's product/service?
- [ ] Did I explain why this idea won?
- [ ] Did I include source URLs for verification?

### Lead And Competitor Checklist

Before final report, verify:

- [ ] Did I load Stage 10: `playbooks/10_LEAD_COMPETITOR_DETECTION.md`?
- [ ] Did I treat lead/competitor detection as a core opportunity module, not a small appendix?
- [ ] Did I detect leads and competitors during the same data collection pass, unless the human explicitly approved a deeper pass?
- [ ] For the first lead/competitor private data source pass, did I use 10 scrolls per approved source when safe, or document why I could not?
- [ ] For recurring daily runs, did I use 5 scrolls per approved source by default, or document the configured value?
- [ ] Did I detect hot leads?
- [ ] Did I detect warm leads?
- [ ] Did I detect indirect need signals, pain signals, buying triggers, objections, comparisons, complaints, or adjacent needs when relevant?
- [ ] Did each lead include profile URL and post/current URL?
- [ ] Did I explain why each lead is hot or warm?
- [ ] Did I detect direct competitors?
- [ ] Did I detect indirect competitors?
- [ ] Did I detect adjacent competitors?
- [ ] Did I detect audience competitors?
- [ ] Did I detect authority/KOL competitors when they compete for the same audience's trust?
- [ ] Did each competitor include profile URL and post/current URL?
- [ ] Did the HTML report include `Public Lead & Competitor Opportunities` and `Private Lead & Competitor Opportunities`, or same-language equivalents, when those lanes have data?
- [ ] Did every displayed lead/competitor opportunity include a post/current URL when available?
- [ ] Did every displayed lead/competitor opportunity include a context-aware copy-ready comment?
- [ ] Did each copy button copy only the suggested comment and avoid implying auto-posting?
- [ ] Did each suggested comment use the same language as the post?
- [ ] Did each suggested comment provide value without directly advertising, saying `DM me`, `message me`, `inbox me`, `book a call`, `reach out to start`, or attacking a competitor?
- [ ] If I used one or two tiny natural imperfections or typos, did they make the comment sound human without making the user look careless or unclear?
- [ ] Did I avoid suggesting spammy outreach or unsafe actions?
- [ ] Did I update `lead_log.md`, `competitor_log.md`, and `lead_competitor_opportunities.jsonl` when possible?

### WideCast Writing Draft Checklist

Before presenting the content draft, verify:

- [ ] Did I load the WideCast writing method through MCP, public API, static zip, or local cache?
- [ ] If MCP/account was unavailable, did I continue through the public writing-skill fallback instead of blocking?
- [ ] Did the draft match the selected best idea?
- [ ] Did every draft variant use a clear label like `Version 1: VE — Value Explainer`, not an unexplained abbreviation like `VE` or `QA` alone?
- [ ] Did the hook, headline, or opening speak to the target audience pain point?
- [ ] Does the draft teach, clarify, warn, compare, or help the viewer make a better decision before mentioning the client's product/service?
- [ ] Did I avoid turning the draft into a direct ad, client praise piece, competitor attack, or "why our product is better" pitch?
- [ ] Did the draft include source-backed rationale?
- [ ] If this is a video script, did I include visual notes?
- [ ] Did I include CTA?
- [ ] Before any provider video creation request, did I load and apply the existing WideCast video script-writing skill and save the final script/brief artifact instead of sending the report script unchanged or writing from agent memory?
- [ ] If a report version/code was already selected, did I use that version as the picked script and continue into Stage 2 visual treatment only, instead of repeating Stage 1's five-format comparison?
- [ ] For visual-dependent videos, did the skill-produced final script include vetted direct image URLs, markdown image syntax, or a media-pool/visual blocker entry, with no fabricated URLs?
- [ ] In manual/interactive work, did I stop after the final script/visual handoff and wait for explicit confirmation before provider video creation?
- [ ] In scheduled Automation Flow, if I continued directly after the final skill pass, did I verify the run already had valid video-creation approval?
- [ ] Did I ask for approval before creating/rendering/publishing video?
- [ ] Before any provider-backed video creation, scene editing, credit check, media upload, render/export, publish, notification, or analytics action, did I load `playbooks/SOLO_AGENCY_VIDEO_PROVIDER_ADAPTER.md` after the vendored writing/video-editing/provider skill?
- [ ] If a vendored skill named a concrete MCP call such as `widecast_create_video`, `widecast_video_data`, `widecast_modify_scene`, or `widecast_account`, did I resolve it as a client-scoped provider capability from this client's config/OpenAPI cache before using any tool?
- [ ] If the verified client provider or required video operation was missing, did I avoid all local video fallbacks (`ffmpeg`, Pillow, `moviepy`, Remotion, browser/canvas screenshots, slideshow export, MP4/MOV/GIF) and ask for PDNA setup/API key instead?
- [ ] If explaining the system's capabilities, did I explicitly mention that approved drafts can become produced video/blog/social assets through connected providers, not only scripts/blogs/captions waiting for manual production?
- [ ] Did I avoid spending credits without explicit confirmation?

### Video Scene Editing Checklist

Before saying a provider-created video is ready for final render/export, verify:

- [ ] Did I load Stage 3 and `playbooks/SOLO_AGENCY_VIDEO_PROVIDER_ADAPTER.md` before any provider scene/edit action?
- [ ] Did I resolve the video-editing operations from this client's Client tools first: `getEditingSkill`, `getVideoData`, scene geometry, scene inspector, `modifyScene`, and media helpers when needed?
- [ ] Did I load `playbooks/skills/video-editing/SKILL.md` through the verified client provider when available, or from the local repo fallback when the provider skill endpoint was unavailable?
- [ ] Did I follow the editing skill module load map instead of working from memory?
- [ ] Did I pull `getVideoData` before editing and use stable scene UID/`voice_file` when available?
- [ ] Did I use scene geometry for coordinates and scene inspector screenshots/local visual evidence for visual judgment?
- [ ] Did I show local screenshot/media/SVG evidence before judging, applying, or uploading it when the editing skill required visual proof?
- [ ] Did I confirm `modifyScene` changes by re-pulling scene data/geometry?
- [ ] Did I log `scene_editing_complete`, `scene_editing_blocked`, `scene_editing_declined`, or `scene_editing_needs_human_recording` in the internal report/history?
- [ ] Did I avoid paid generated images, render/export, publishing, clone use, or any credit spend without a fresh explicit approval gate?
- [ ] Did I ask the human to render/export the final MP4 only after the editing skill's pre-summary completion scan passed?

### Production Provider Choice Checklist

Before claiming the PDNA setup gate is complete, verify that default setup stayed low-friction:

- [ ] Did I explain PDNA as Production, Distribution, Notification, and Analytics in plain language?
- [ ] If the human wanted default setup, did I ask only for the WideCast API key and avoid provider/scope/spend/publish/account-identity questions?
- [ ] After the human provided the WideCast key, did I send the one-time confirmation ping (`sendNotification` "Hello", no report content), report per-channel delivery (email + Telegram-or-not) to the human, log it as `setup_notification_confirmation`, and mark notification `connected` only on success (or the exact blocker on failure)?
- [ ] Did I avoid asking "Provider=...", "Scope=...", "Spend credits yes/no", "Publish yes/no", "What account identity?", or similar setup-form questions before starting the default WideCast path?
- [ ] Did I treat WideCast as the default maintained all-in-one agent-facing path, not as the identity of Solo Agency and not as mandatory for research, ideas, leads, reports, or free draft writing?
- [ ] Did I discuss specialist-stack examples only if the human explicitly asked for alternatives, rejected WideCast, or requested a non-default provider?
- [ ] If asking the human to connect WideCast, did I give the exact OpenAPI/API key path: register at `https://widecast.ai/#setup` (free 50 credits/month when that offer is shown), log in, click `Setup AI Agent`, open `API Keys & MCP`, click `Setup`, click `Generate API key and MCP url`, then copy only the API key for this specific client?
- [ ] Did I mention Telegram and social-account connection as optional WideCast-side setup, without turning them into separate yes/no questions during default PDNA setup?
- [ ] Before checking WideCast account status, credits, connected platforms, publish settings, Telegram, analytics, or capabilities, did I identify the active `target_client_slug` and read that client's `integrations/providers/provider_config.local.json`?
- [ ] Before checking any provider tool availability, did I check Client tools first (provider config, OpenAPI cache/spec, verified identity, `provider_capabilities.json`) and global MCP/native tools only second?
- [ ] For WideCast OpenAPI, did I select `https://widecast.ai/app/dashboard` as the current production server and skip `https://api.widecast.ai` as a disabled/planned vanity host unless a future playbook explicitly enables it?
- [ ] Did I avoid treating a global WideCast MCP/native tool account in the current AI session as proof that this client's PDNA is connected?
- [ ] If only a global MCP/native provider account was visible, did I mark `global_mcp_not_client_scoped` or `global_mcp_available_but_not_authoritative` instead of listing those global credits/platforms as this client's status?
- [ ] Did I verify the account through this client's configured OpenAPI/API-key path and compare the verified identity to the saved client provider identity before claiming PDNA is connected?
- [ ] Did I load `playbooks/SOLO_AGENCY_VIDEO_PROVIDER_ADAPTER.md` before treating any video/blog/social provider action as available for this client?
- [ ] Did I select production, scene editing, media upload, render/export, publish, notification, and analytics operations from this client's `provider_capabilities.json` or freshly discovered OpenAPI operation list, not from the current AI host's global MCP tool list?
- [ ] Did I check connected publishing platforms and provider settings through the verified client account, not through MCP-global `accounts` or `platform_settings` output?
- [ ] Did I avoid asking for passwords, cookies, OTPs, social credentials, or browser session tokens?
- [ ] If the human or AI host explicitly chooses MCP/connector setup, did I include the agent-specific setup guide link: Claude `https://widecast.ai/claude.html`, Codex/ChatGPT/OpenAI `https://widecast.ai/chatgpt.html`, Gemini `https://widecast.ai/gemini.html`, or Grok `https://widecast.ai/grok.html`?
- [ ] If asking for an MCP URL, did I explain that it is optional compatibility, may include a `wc_mcp_...` token, and should be pasted exactly with no separate password/OAuth unless the official guide says otherwise?
- [ ] Did I verify the provider account with the account operation, such as WideCast `getAccount`, before claiming PDNA setup is connected?
- [ ] Did I save provider discovery/capabilities and account identity only under the correct client's `integrations/providers/` folder?

### Production Setup Anti-Drift Checklist

When production/video/blog/social work happens inside the one-time agency setup process, verify:

- [ ] Did I treat step 7 as provider/capability setup after the setup handoff or an Automation Flow report, not open-ended trial video creation?
- [ ] Did I avoid starting scene editing, repeated media swaps, render/export, publishing, or credit-spending while steps 8-9 were still pending, unless the human explicitly overrode after a warning and the client-scoped provider plus required operation were verified?
- [ ] After provider setup completed, did I gently return to the next setup step instead of asking to keep playing with the video?
- [ ] If the human explicitly insisted on a trial video before setup completed, did I verify the client-scoped provider and required operation first, avoid local video fallback, and record the parent setup checkpoint before entering the branch?
- [ ] Did I remember the next parent setup step after the branch, instead of losing the agency setup thread?
- [ ] Did I show a compact parent setup checkpoint during the short production branch?
- [ ] After one natural branch checkpoint, did I return to the full `Solo Agency one-time setup process` roadmap unless the human explicitly asked to continue the branch?
- [ ] Did I avoid claiming agency setup was complete merely because a provider was connected or a video trial was created?
- [ ] Did I avoid forgetting steps 8-9 after production/video testing ended?

### Output And Delivery Checklist

Before saying the run is complete, verify:

- [ ] Did I save Markdown as the canonical internal record?
- [ ] Did I generate a polished mobile-friendly client-facing HTML report set as the canonical client-ready report?
- [ ] Did I generate `{client-name}-INTERNAL_REPORT.html` and label it `INTERNAL_REPORT - Not for client sharing`?
- [ ] Did the client-facing HTML report follow the Agency-Grade HTML Report Standard, not merely list raw ideas?
- [ ] Did the client-facing HTML/PDF pass the Client-Blind Scrub Gate: no Solo Agency, WideCast, PDNA/provider tooling, OpenAPI, MCP, Local Collector, Chrome extension, automation/scheduled task, API-key/config, Telegram, agent/tool/debug details, or `INTERNAL_REPORT`?
- [ ] Did I put all WideCast/provider/Telegram/social-platform/API-key/config/Local Collector/private source inventory/automation/blocker/debug details in `INTERNAL_REPORT`, not client-facing files?
- [ ] Did the top of the report include an Executive Snapshot with source coverage status, best idea, lead/competitor counts, content readiness, blockers, and one recommended next action?
- [ ] If optional private data source discovery was asked, approved, pending, blocked, or completed, did the HTML report include a clear `Private Data Source Discovery` section?
- [ ] Did the client-facing discovery section summarize only safe coverage/signal categories, while exact discovery categories, platforms/URLs, candidate sources, skipped/noisy sources, feed signals, approval needs, and internal mechanics are kept in `INTERNAL_REPORT`?
- [ ] Did I include a claim-level Evidence Ledger for important facts, numbers, dates, laws, prices, platform policy claims, and market signals?
- [ ] Did I remove or down-rank unsupported numeric/date/regulatory claims instead of using them in the main hook?
- [ ] Did every important idea include its own reference URL(s), not only a generic source list elsewhere?
- [ ] Did I include Source Coverage And Data Quality, including public keywords used, private data source status, blind spots, and confidence?
- [ ] Did I include a Decision Scorecard comparing top candidate ideas before selecting the winner?
- [ ] Did I clearly distinguish `not detected` from `not scanned`, `pending activation`, or `session expired` for leads and competitors?
- [ ] If competitor data was only a hypothesis without profile/post URLs, did I label it as a hypothesis rather than detected competitor evidence?
- [ ] Did the report include a Production Readiness status for each draft, such as `production-ready`, `script-ready, media-pending`, or `needs human detail`?
- [ ] Did the report end with exactly one primary next action, with secondary actions clearly de-emphasized?
- [ ] Did the operator chat or notification that announces the report show an updated progress block when required steps remain?
- [ ] If schedule/automation already exists, did that operator chat/notification and `INTERNAL_REPORT` include an `Automation freshness check` instead of only saying the config/report is updated?
- [ ] Did the operator chat/notification and `INTERNAL_REPORT` include the Provider Report Delivery Capability Check outcome: Client tools checked first, provider/OpenAPI discovery checked, account verified or blocker, upload attempted or blocker, notification attempted or blocker, final HTML report path/link, PDF companion path/status, and INTERNAL_REPORT path/status?
- [ ] Did that chat or notification end with exactly one concrete next-step question in a `**[ACTION REQUIRED]**` block when the human needs to choose the next step?
- [ ] Is the HTML factually aligned with the internal Markdown report?
- [ ] Is the HTML standalone and portable?
- [ ] Did I avoid making the HTML depend on `fetch("./report.md")`, remote scripts, remote CSS, or a neighboring Markdown file?
- [ ] Did I check the HTML at a 390px mobile viewport, or reason through equivalent CSS, so the document itself has no horizontal overflow?
- [ ] Are wide tables/evidence ledgers/scorecards inside a dedicated scroll wrapper or converted to stacked cards, with long URLs/source names wrapping inside the container?
- [ ] If the client's WideCast/OpenAPI provider config is not connected and verified, did `INTERNAL_REPORT` and the operator handoff include the PDNA/WideCast setup note, while the client-facing report stayed clean?
- [ ] If WideCast Telegram is not connected yet, did `INTERNAL_REPORT` include a concise operator note about registering/logging in to WideCast and connecting Telegram for daily report links/blockers, while the client-facing report stayed clean?
- [ ] If the report includes script/blog/social drafts, did I present each version in an editable HTML block with a working local `Copy this version` button?
- [ ] Did the HTML draft section avoid saying `AI chat`, `agent`, Solo Agency, WideCast, providers, or internal workflow mechanics?
- [ ] Did every editable version clearly say the reviewer can copy the edited final text for review or production?
- [ ] Did I update `outputs/latest/{client-name}-daily-report.html`, `outputs/latest/{client-name}-INTERNAL_REPORT.html`, and the latest lane HTML files when those lane reports exist?
- [ ] Did I generate `{client-name}-client-report.html` from the three scrubbed canonical HTML files before exporting the mandatory `{client-name}-client-report.pdf`, or record the exact PDF blocker/status?
- [ ] If the PDF includes private data source findings, did I redact raw private posts, group member details, login/session details, collector internals, private source inventory, and unapproved private source URLs/excerpts, or mark `client_pdf_redaction_status: needs_human_review` instead of exporting?
- [ ] Did I preserve the canonical `.html` report path/link even when also providing a `.pdf` export?
- [ ] Did I generate/update master digest if multiple clients exist?
- [ ] Did I write the report in the human's language?
- [ ] Did every user/operator-facing report link/path in chat, Telegram, or notification point to `.html`, not `.md`?
- [ ] Did I avoid fake interactive buttons in static HTML, except real local copy buttons for editable draft review?
- [ ] Did I include references/URLs in the report?
- [ ] Did I notify the human through the configured provider notification channel if available, preferably WideCast OpenAPI `sendNotification`, relying on WideCast's email fallback if Telegram is not connected and fallback is available?
- [ ] Did every report-ready notification include an HTML report URL/path, PDF companion path/status, and INTERNAL_REPORT path/status? A plain "report ready" notification with no report URL/path and PDF/internal status is invalid.
- [ ] If WideCast OpenAPI notification/Telegram was available and an HTML-capable `uploadAsset` operation was available, did I upload the `.html` report to WideCast for operator delivery first and send the uploaded report URL instead of only a local path, while treating provider-hosted URLs as non-client-share links?
- [ ] Did I record a report-delivery object with local HTML path, local PDF path/status, INTERNAL_REPORT path/status, client-facing scrub status, provider, OpenAPI discovery status, account verification status, upload attempted status, uploaded HTML/PDF URL if any, upload blocker if any, notification channel, and final notification report link?
- [ ] If WideCast report upload was unavailable or failed, did I log the provider-neutral blocker, such as `provider_config_missing`, `provider_auth_failed`, `provider_discovery_failed`, `provider_required_operation_missing`, `provider_account_mismatch`, `global_mcp_not_client_scoped`, or `provider_upload_failed`, and send the best available HTML path/link plus PDF companion path/status plus INTERNAL_REPORT path/status?
- [ ] If I accidentally sent a notification without a report URL/path or PDF companion status, did I immediately send a correction notification with the HTML report URL/path plus PDF status and log the correction?
- [ ] If provider notification was unavailable, did I try Gmail/email MCP or connector if available?
- [ ] If neither WideCast OpenAPI notification nor Gmail/email was connected, did I suggest connecting WideCast API key + Telegram/email fallback first, or Gmail/email as a secondary fallback?
- [ ] Did the notification include status, HTML report path/link, PDF companion path/status, INTERNAL_REPORT path/status, blockers, and next action?

### Measure-Learning Checklist

Before claiming a weekly/monthly performance review or learning loop is complete, verify:

- [ ] Did I call available verified provider analytics operations, such as WideCast OpenAPI `getAnalytics`, `listVideos`, `getStatus`, and `getVideoData`, for published URLs, metadata, and account/platform analytics?
- [ ] Did I reuse the Solo Agency Local Collector extension plus Local Collector app to capture visible metrics from published URLs when possible?
- [ ] Did I store normalized metrics in `analytics/metrics_log.md`?
- [ ] Did I mark hidden or unavailable metrics as `unavailable` instead of inventing numbers?
- [ ] Did I use the measurements to update content pillar scoring, hook learnings, CTA learnings, source priority, and future idea selection?

### Evidence-Based Audit Requirements

Scope the self-audit to the reply being sent:

- Before any completion claim or human handoff, run the full Response Self-Audit Checklist (§27) plus the relevant Output And Delivery, Measure-Learning, and stage checklists.
- For an intermediate progress reply, check the core set: human language; the `**[ACTION REQUIRED]**` contract; no secrets/credentials/API keys leaked; and a progress block plus `Automation freshness check` when schedule/automation exists. Add the items relevant to the action just taken.

For these five mechanical gates, the audit must paste real command output, not a self-declaration:

- Client-blind scrub: run `grep -iE 'Solo Agency|WideCast|Telegram|INTERNAL_REPORT|api_key|OpenAPI|MCP|Local Collector'` over the extracted text of every client-facing HTML/PDF; the printed hit count must be `0`.
- Report-set existence: `ls` the report-set files for the client/day and paste the listing.
- report_state consistency: quote the status/count fields from `outputs/YYYY-MM/YYYY-MM-DD/{client-name}-report_state.json`.
- LOAD LEDGER line counts: paste the printed ledgers for the stages loaded this run.
- Notification record: paste the `notifications/notification_log.md` row for this run.

Print the evidence as a compact block, one line per gate, in the form:

```text
gate | evidence | pass/miss
```

Honest misses are compliant: a miss with a stated reason is acceptable, but a rubber-stamped pass without pasted evidence is a critical violation.

### Final Hard Gate

If any required checkbox above is not satisfied:

- Do not claim the run is complete.
- Fix the missing step if possible.
- Do not merely promise to fix a required missing item in the next run when it can be corrected in the current report.
- If it cannot be fixed, explicitly report:
  - what was missed;
  - why it was missed;
  - whether the output is still usable;
  - what should happen next.

---

## 28. Final Reminder For The Agent

The human should not need to manage the workflow manually.

The human provides only:

- Client name.
- Product/service, profession, expertise, business description, or public website/profile URL.
- Target location only when needed and not inferable.
- Private data sources they want monitored.
- Corrections to the agent's inferred setup.
- Approval before video creation, rendering, publishing, or spending credits.
- Telegram/WideCast API key notification setup once per client, if they want scheduled alerts while away from the AI agent UI.

The agent owns:

- Industry inference.
- Sub-industry inference.
- Related-industry inference.
- Target audience inference.
- Pain point inference.
- Public data source discovery.
- Data collection.
- Hot/warm lead detection.
- Direct/adjacent/audience competitor detection.
- Idea generation.
- Best idea selection.
- Script writing.
- Content pillar management.
- Content calendar management.
- Approval tracking.
- Asset indexing.
- Publishing status tracking.
- Repurposing suggestions.
- Analytics and reporting.
- Experiment backlog management.
- Client-facing summaries.
- Mobile-friendly HTML report generation.
- Delivery of report files/links through the most convenient authorized channel.
- History tracking.
- Schedule/routine setup according to environment capability.
- WideCast OpenAPI setup discovery and integration guidance.
- WideCast OpenAPI Telegram/email fallback notification delivery for scheduled results, blockers, and human-action alerts.

This is the intended operating model.
