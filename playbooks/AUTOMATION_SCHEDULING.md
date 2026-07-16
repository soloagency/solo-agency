# Automation & Daily Run Contract

Role: `AUTOMATION_SCHEDULING.md` — the scheduling and daily-run contract for OutreachCRM.

Loaded by `playbooks/SETUP_FLOW_ENTRYPOINT.md` to configure the schedule and the
client-specific automation task, and by `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` to
execute an unattended daily run. This file defines the Daily Run order (DESIGN §15),
the scheduling-mechanism-agnostic rule, the one-automation-task-per-client naming and
pinning rule, the per-client `run_lock`, the Automation Resync machinery with its
dry-read verification, and the inbound classifier ordering (DESIGN §12). It is not a
numbered stage — Stage 4 is now `04_VERIFY_ENRICH.md`.

## Load Rule

Load during one-time setup after the client profile, sendboxes, and at least one campaign
goal are known, so the schedule and the client-specific automation task can be configured
**before** the first send. Also load at the start of every scheduled run, and whenever the
schedule/automation config is reviewed, resynced, or repaired.

This file is loaded, not summarized. A short read is NOT a load: register a LOAD LEDGER
entry per `playbooks/LOAD_LEDGER_PROTOCOL.md`, checked against `playbooks/LOAD_MANIFEST.md`,
before taking any side-effect action (sending, enriching, writing config, creating a task).

## Hard Gates For This Contract

- **Setup Flow is the control plane only.** During one-time setup, configure the schedule
  and the client-specific automation task after the profile, sendboxes, and first campaign
  goal are known. Setup Flow **never sends an email, never enriches for send, and never runs
  a campaign**. Its terminal state is `ready_for_automation_first_run`.
- After configuring the schedule, do **not** run the first send/enrich pass inside Setup
  Flow. Verify the client-specific automation task and tell the human the exact task name to
  run for the first automation run.
- If the human asks to run, send, enrich, generate, draft, refresh, or report during Setup
  Flow, do **not** do it and do **not** ask whether to do it now. Treat the request as a
  handoff: verify/resync the task, then give the human the exact client-specific automation
  task name to run.
- Support manual-only, daily, multiple-times-daily, weekly, and environment-specific
  schedules. The playbook is scheduling-mechanism-agnostic (see Scheduling Rule).
- Scheduled runs must execute the full Daily Run order (DESIGN §15): load contract → sync
  inbox → pull tracking → semantic triage + apply-rules → follow-up advising → load new
  pipeline (verify + enrich + draft) → send approved → assisted channels → Today View +
  kanban → reports (incl. Monday weekly) → Telegram notify → Stage 9 audit → release
  `run_lock`. No step may be silently skipped; a skipped step is recorded with its reason.
- Before any report HTML/PDF work, scheduled runs must load
  `playbooks/skills/report-design/SKILL.md` and use `tools/report_renderer.py` by default
  instead of writing ad hoc report/PDF scripts.
- Before any send, the send happens through `gmail_client.py send` with the full ordered
  pre-send re-check chain in code (DESIGN §10). Playbook prose never replaces the in-code
  gate. If the tracker pull has not succeeded within the configured window for a box,
  **sending for that box is blocked** (opt-out compliance, DESIGN §16) — this is a gate,
  not a warning.
- The Phase-1 tools (`gmail_client.py`, `crm_store.py`, `import_leads.py`, `email_verify.py`)
  **exist** — the sync, apply-rules, and send steps run for real. But the enrich/draft/
  follow-up steps depend on Phase-2 `status: planned` stage files+skills (Stage 4 verify/enrich,
  Stage 6 email-writing, Stage 10 follow-up); until those ship, record those steps as
  `stage_file_pending` per DESIGN §22 R1 (load the covering `docs/DESIGN.md` section, do not
  improvise, do not enter Last-Resort Recovery), and continue with the steps that are built.
  A genuinely missing Phase-1 tool (partial install) is the DESIGN §22 R2 fallback: record
  `skipped: tool_not_built`, raise one `**[ACTION REQUIRED]**`, continue.
- Scheduled runs must load the needed playbooks again at run time; they must not rely on
  memory from setup.
- Every scheduled-run human-facing reply, notification, or report handoff must include an
  updated progress block (see Scheduled Run Progress Display Contract). Multiple updates in
  one run each show the current completed/current/remaining state.
- Store the schedule config, chosen scheduling mechanism, timezone, and notification channel
  in `outreach-pipeline/schedule.md`.
- After any human-approved change made after the schedule/automation was created, perform
  Automation Resync before claiming the next scheduled run is updated.
- Never say "the automation is updated" if only the client profile, only `schedule.md`, or
  only a single campaign config was changed. The whole automation package must be synced, or
  the remaining resync blocker must be stated.
- Take a per-client `run_lock` before starting a client's run; release it on completion.
- Every schedule/routine question, native-automation-task creation/update instruction,
  scheduled-prompt paste/replace instruction, report/send-request hard-stop handoff, and
  automation-freshness blocker that needs the human must use the root playbook
  `**[ACTION REQUIRED]**` block: one purpose, one exact next step, one command or path. When
  nothing is needed, say `No action required right now.`

## Client-Specific Automation Tasks (Setup Flow vs Automation Flow)

OutreachCRM keeps two distinct flows. **Setup Flow** creates config and the automation task
but never sends, enriches, or runs a campaign. **Automation Flow** is the real, unattended
daily run executed by the client-specific automation task.

Rules:

- Create **one client-specific automation task per active client** by default.
- Every client-specific task name must begin with the client name, for example
  `AvenNgo - OutreachCRM Daily Run`.
- The task prompt must pin `target_client_slug` and must not process any other client. A
  client-named task that touches a second client's data is a critical violation.
- The task prompt runs the full Daily Run order (DESIGN §15) for that one client, loading the
  needed stage files fresh at run time.
- A setup/config session may instruct the human to run `AvenNgo - OutreachCRM Daily Run`, but
  it must not send, enrich, or run the campaign inside the setup chat.
- A setup/config session must **not** load `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` as a
  workaround for a human "run it now" request. The scheduled entrypoint belongs in the native
  automation task or a separate Automation Flow run, never inside Setup Flow.
- Automation Flow may accept config changes during a real run, but must immediately perform
  Automation Resync before claiming that future runs are current.
- Plus exactly one agency-wide maintenance task, `OutreachCRM - GitHub Update Watch`, which
  is barred from every client-facing and send-capable channel (see GitHub Update Watch
  Scheduling Rule).

For multi-client daily operations, prefer separate client tasks. Only a client's own run
mutates that client's data. OutreachCRM defines exactly two automation task kinds: one
client-specific Daily Run task per active client and the single agency-wide
`OutreachCRM - GitHub Update Watch` task — there is no agency-wide client-processing task.

## Source Preservation Rule

This file is a detailed operating contract. Do not summarize away requirements, examples,
checklists, schemas, ordered gates, protocols, edge cases, warnings, approval gates, or
completion gates. A downstream agent may summarize its human-facing reply, but it must still
obey the full requirements in this file.

---

## Scheduled Run Playbook Loading Contract

A scheduled run is not a shortcut around the playbook. It is the same OutreachCRM workflow
executed with saved context instead of re-asking the human setup questions.

At the start of every scheduled run, load or re-load the stage files for the work about to
be done. Every load requires a LOAD LEDGER entry per `playbooks/LOAD_LEDGER_PROTOCOL.md`,
checked against `playbooks/LOAD_MANIFEST.md`:

If a stage file or skill listed below is still status: planned and does not exist, load
docs/DESIGN.md (the section covering that stage) with its own LOAD LEDGER and record
stage_file_pending in the run record — do not treat the missing file as a truncated/failed
read and do not fetch it from the GitHub raw URL (DESIGN §22 R1).

1. Always load Stage 0: `00_CORE_CONTEXT_REQUIREMENTS.md`.
2. Always load Stage 7: `07_STORAGE_SCHEMA_AND_HISTORY.md` to read profiles, CRM records,
   logs, sent_log, suppression, and history through `crm_store.py` (direct file writes to CRM
   collections are a critical violation).
3. Always load this contract (`AUTOMATION_SCHEDULING.md`) for the daily-run order.
4. Load Stage 1 (`01_CLIENT_SETUP_PROFILE.md`) only if the profile is missing, incomplete,
   stale, or needs setup repair, or this is the first Automation Flow run for the client.
   Do not re-ask setup questions when the saved profile is complete.
5. Load Stage 10 (`10_FOLLOWUP_REPLY_MANAGEMENT.md`) *(planned)* for inbox sync, the
   deterministic inbound classifier, semantic triage, and follow-up advising. This is loaded
   on every run.
6. Load Stage 12 (`12_TRACKING_ANALYTICS.md`) *(planned)* to pull worker events (open/click,
   bot-filtered) and update metrics/learning logs.
7. Load Stage 13 (`13_CRM_CORE.md`) *(planned)* for `apply-rules`, deals/tasks, stage
   transitions, dedupe/merge, and `resolve()` on every `lead_id` lookup path.
8. Load Stage 14 (`14_TASKS_TODAY_VIEW.md`) *(planned)* for the task engine, SLA sweep, and
   Today View.
9. Load Stage 4 (`04_VERIFY_ENRICH.md`) *(planned)* and skill `email-verify-enrich`
   *(planned)* before any enrichment when loading new pipeline.
10. Load Stage 5 (`05_CAMPAIGN_MANAGEMENT.md`) *(planned)* for campaign goal, audience, and
    sequence when selecting/advancing campaigns.
11. Load Stage 6 (`06_EMAIL_WRITING_STANDARD.md`) *(planned)* and skill `email-writing`
    *(planned)* before drafting any email (step-1 or bump). A draft may contain only details
    present in the dossier with an `evidence_url`.
12. Load Stage 8 (`08_SEND_ENGINE_PROTOCOL.md`) *(planned)* before any send.
13. Load Stage 15 (`15_CRM_REPORTING.md`) *(planned)* and then
    `playbooks/skills/report-design/SKILL.md` whenever generating, reviewing, fixing, or
    packaging any HTML/PDF report — including the Monday weekly client report.
14. Load Stage 9 (`09_OPERATIONS_SAFETY_AUDIT.md`) before claiming the run is complete.
15. Load Stage 11 (`11_UPDATE_AND_VERSION_WATCH.md`) only when the task is
    `OutreachCRM - GitHub Update Watch`, an update/upgrade/sync-latest request is being
    handled, or blocker recovery must check GitHub for a newer OutreachCRM version.

The difference between first setup and scheduled runs:

- First setup asks only the minimum questions because config does not exist yet.
- Scheduled runs read the saved profile, sendboxes, lists, campaign configs, CRM records,
  suppression, sent_log, tracking, and analytics logs, then continue automatically.
- Scheduled runs must not re-ask profile fields (voice, offer, compliance address, market,
  timezone) that are already present.
- Scheduled runs may ask the human only when an approval gate, a blocker, a missing critical
  field, a sendbox re-auth, a guessed-send decision, a suppression-confirm task, or a merge
  proposal requires human input — always through `**[ACTION REQUIRED]**`.

Scheduled-run completion requires the same end-to-end path as a manual daily run: inbox sync,
tracking pull, triage + apply-rules, follow-up advising, new-pipeline drafting, sends within
quota, assisted-channel drafts where allowed, Today View + kanban, operator reports (plus the
Monday client-facing weekly report), Telegram notification, and the Stage 9 audit.

## Scheduled Run Progress Display Contract

Scheduled runs are meant to be automatic, but the human still needs visible state whenever
the agent speaks. Every scheduled-run reply, notification, or report handoff must include:

- completed steps;
- current active step;
- remaining steps;
- blockers or human decisions required (each as an `**[ACTION REQUIRED]**` block);
- send/quota state (sends made, quota remaining per box, any box blocked or paused);
- whether open/click tracking ran or was skipped (e.g. `plain_text_mode`, or track-pull
  failed and sends were blocked).
- an `Automation freshness check` stating whether the latest approved changes are synced into
  the configured automation/scheduled task and whether tomorrow's run will read the current
  contracts/prompts/playbooks/config/state, not only the latest config file.

Use this title:

```text
OutreachCRM daily run progress
```

The agent may use a compact form in notifications, but it must not send only a report link or
a bare summary while steps remain.

Use this compact automation freshness line in scheduled-run updates and setup/repair progress
blocks after a schedule exists:

```text
Automation freshness check: {✓ current | → resync in progress | ! action needed | – not applicable yet} - latest approved changes synced into the automation/scheduled task, including prompt/contract/playbook/config/state, not only a single config file: {yes | in progress | needs human task prompt update | no schedule yet}.
```

---

## Automation Resync Contract

An automation/scheduled task can hold a stale prompt snapshot from the moment it was created.
A later config edit is not enough if the scheduled task still points to old instructions, old
config state, or old setup assumptions.

Trigger Automation Resync whenever a human-approved change happens after schedule/automation
setup, including:

- profile fields, voice, offer, compliance/physical address, market, or timezone changed;
- a campaign was created/edited: goal, audience/segment, sequence steps or `gap_days`,
  daily quota, guardrails, approval mode, or channel strategy changed;
- sendboxes changed: added/removed a box, changed auth mode (app_password/oauth), quota,
  warmup stage, or a box moved to `needs_reauth`/`paused`;
- suppression, segments, pipelines, or CRM rules changed;
- the tracker worker changed: `trk.{domain}`, HMAC secret rotation, `TRACKER_API_KEY`, or a
  `wrangler deploy` was required;
- WideCast provider notification config changed (`api_key_env`/`api_key_local`, OpenAPI
  discovery, Telegram/email fallback), or the notification channel changed;
- schedule cadence, timezone, active clients, manual-only mode, or notification channel
  changed;
- the playbook behavior changed in a way scheduled runs must follow;
- OutreachCRM update/version-watch state changed, an upstream update was applied, the
  update-watch task was created/changed, `tracker_worker_deploy_required` was set, or
  `storage_schema_migration_required` was set.

Automation Resync requires updating every relevant layer:

1. **Client profile** (`clients/{slug}/.../client_profile_*.md`): current voice/offer/
   compliance, active campaigns, sendbox status, notification status.
2. **CRM + campaign config** (via `crm_store.py`): campaign configs, sendboxes.json,
   suppression, segments, pipelines — the current state the run will read.
3. **`outreach-pipeline/provider_defaults.json`**: WideCast notification catalog/discovery
   defaults, no secrets.
4. **Client provider files** (`integrations/providers/`): `provider_config.local.json`
   (`api_key_env`/`api_key_local`, never a field named `api_key`), `provider_capabilities.json`,
   `provider_openapi_cache.yaml`, `provider_health.md`, and `provider_calls.jsonl` when relevant.
5. **`outreach-pipeline/schedule.md`**: cadence, scheduling mechanism, included clients,
   timezone, notification channel, and last resync timestamp.
6. **`outreach-pipeline/automation/automation_manifest.md`**: current run contract, data
   paths, active clients, prompt source, config source, provider-config source, and last
   known state hash/summary.
7. **`outreach-pipeline/automation/scheduled_run_prompt.md`**: the exact prompt the native
   AI automation/scheduled task should run (pins `target_client_slug`).
8. **Native AI automation or scheduled task body**: update it when the environment stores a
   separate prompt snapshot.
9. **`outreach-pipeline/automation/update_state.json`** and `update_log.md`: update-watch
   state, checked/applied commits, change classification (including
   `tracker_worker_deploy_required`/`storage_schema_migration_required`), and human actions
   required.
10. **`outreach-pipeline/automation/resync_log.md`**: what changed, which files/tasks were
    updated, what could not be updated, and what the next scheduled run should see.

If the native AI automation task cannot be edited by the agent, the agent must:

- write the exact replacement prompt to `outreach-pipeline/automation/scheduled_run_prompt.md`;
- mark `automation_prompt_update_pending` in `automation_manifest.md` and `schedule.md`;
- give the human one concrete instruction to paste/replace the scheduled task prompt in a
  `**[ACTION REQUIRED]**` block;
- not claim the scheduled run is fully updated until the human confirms the native task body
  was updated.

### Automation Resync verification (dry-read)

Before saying a post-schedule change is complete, do a dry-read **as if tomorrow's scheduled
run were starting**:

1. Read `playbooks/SCHEDULED_RUN_ENTRYPOINT.md`.
2. Read `outreach-pipeline/automation/automation_manifest.md`.
3. Read `outreach-pipeline/provider_defaults.json` when present.
4. Read `outreach-pipeline/schedule.md`.
5. Read each active client profile.
6. Read each relevant client's provider config/capability files when WideCast notification,
   report delivery, or provider config changed.
7. Read the changed campaign configs, `sendboxes/sendboxes.json`, and suppression state when
   campaigns, sendboxes, or suppression changed.
8. Read `outreach-pipeline/automation/update_state.json` when update/version-watch or a
   GitHub-applied change affects future runs.
9. Confirm the latest user-approved changes are visible from those files **and** from the
   scheduled prompt/task body.

The agent's human-facing completion message must say one of:

```text
Automation Resync complete: the next scheduled run will read the latest approved state.
```

or:

```text
Automation Resync partially complete: config/profile are updated, but the native scheduled task prompt still needs the human to replace it with outreach-pipeline/automation/scheduled_run_prompt.md.
```

Bad completion wording:

```text
I updated the config, so tomorrow's automation is fixed.
```

This is invalid because it hides the possibility that the scheduled prompt/task still holds
an old snapshot.

---

## Scheduling Rule

The agent must use the best scheduling mechanism available in the current environment. The
playbook does not mandate one scheduler because different environments have different
capabilities.

Possible scheduling methods:

- Native AI scheduled task.
- Native AI automation.
- Local cron.
- Windows Task Scheduler.
- macOS launchd (LaunchAgent).
- n8n.
- Make.
- Zapier.
- GitHub Actions.
- Server job.
- Desktop reminder.
- Manual daily run instructions.

The agent must record the chosen method in `outreach-pipeline/schedule.md`.

The agent must also record the **notification channel** in `schedule.md`:

- If the client has verified WideCast OpenAPI config and the discovered spec exposes
  `sendTelegramMessage`, record WideCast Telegram (with email fallback) as the preferred
  operator notification channel for scheduled runs — even if Telegram is not connected yet,
  because WideCast can fall back to email when the account supports it. WideCast is the
  operator-notification provider only (PDNA notification); it never sends anything to a client.
- If WideCast notification is unavailable but Gmail/email is connected, record Gmail/email as
  the secondary fallback notification channel.
- If neither is available, record `notification_channel: local_path_only` and tell the human,
  via `**[ACTION REQUIRED]**`, how to connect a WideCast API key with Telegram/email fallback
  or Gmail/email.

Scheduled runs are unattended runs. The human may not be watching the agent UI, so the agent
must proactively notify the operator when the run finishes or when human action is required.

**Timezone.** `schedule.md` records the human machine's local timezone. All date keys
(`YYYY-MM-DD` folders), "yesterday", the 7-day metric windows, and "Monday" (weekly-report
day) are computed in that recorded timezone. The AI scheduled-task environment may run at UTC;
before computing any date key or window, read the recorded timezone from `schedule.md` so a
run does not split one logical day across two date folders or mis-window a report. Note this
is distinct from the per-recipient send-window gate in Stage 8, which uses each contact's own
`tz`.

## GitHub Update Watch Scheduling Rule

After the first schedule/automation is configured, offer a separate maintenance automation:

```text
OutreachCRM - GitHub Update Watch
```

This task exists because OutreachCRM is updated frequently and an older playbook or tool may
be the cause of tomorrow's blocker.

Rules:

- The task should run daily, preferably before the client daily runs.
- It must load `playbooks/11_UPDATE_AND_VERSION_WATCH.md`.
- It must check the OutreachCRM repo `main` (placeholder `https://github.com/soloagency/outreach`),
  compare the installed version, classify the change, and update
  `outreach-pipeline/automation/update_state.json` plus `update_log.md`. Classification
  includes `tracker_worker_deploy_required` (a `tracker/worker.js` change needing a
  `wrangler deploy` rerun) and `storage_schema_migration_required` (a storage adapter /
  `schema_version` change needing `crm_store.py migrate`).
- It must **not** run client sends, enrichment, campaigns, tracking pulls, reports, or CRM
  mutations under `clients/`.
- It must **not** send Telegram, WideCast/email-fallback, or any client/operator campaign
  notification. GitHub update checks are internal agency maintenance: write
  `outreach-pipeline/automation/update_notice.md` and surface the result in the
  setup/maintenance chat or native task output instead.
- It may auto-apply updates only when the human has approved auto-apply in `update_state.json`
  or an equivalent operator setting.
- Even when auto-apply is approved, a tracker-worker change still requires a human-run
  `wrangler deploy` outside the AI sandbox, and a storage-schema change still requires a
  human-run `crm_store.py migrate` under the storage freeze flag. Both are surfaced as
  `**[ACTION REQUIRED]**`, not silently applied.
- If the automation environment cannot create the native task directly, write the exact
  prompt from `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` to
  `outreach-pipeline/automation/update_watch_prompt.md`, log `update_watch_task_prompt_pending`,
  and give the human the exact task name and prompt path via `**[ACTION REQUIRED]**`.

If no automation is available for the update watch:

1. Explain the limitation.
2. Create manual update-watch instructions.
3. Provide the human the exact GitHub Update Watch prompt to run on demand — the
   "GitHub Update Watch Task Prompt" in `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` (which loads
   `11_UPDATE_AND_VERSION_WATCH.md`, processes no client, syncs no inbox, and never sends) —
   via a `**[ACTION REQUIRED]**` block naming the task `OutreachCRM - GitHub Update Watch` and
   the prompt path. This maintenance prompt is not a client daily-run prompt; the manual
   daily-run prompt lives in the Schedule Contract's manual-only cadence.

---

## Run Locking And Notification Dedup

Scheduled runs can overlap (yesterday's run still finishing, or a manual re-run of a
client-specific task touching a client already being processed). Protect against duplicate
work, duplicate sends, and duplicate notifications:

- Before starting a client's daily run, create or check
  `outputs/YYYY-MM/YYYY-MM-DD/{client}-run_lock.json` (`started_at`, task name, session hint,
  `target_client_slug`). If a fresh lock exists (younger than about 3 hours), do **not** start
  a duplicate run for that client — log it and stop. A stale lock (older than the window, or
  from a run that clearly died) may be taken over, with a note in the run record. Remove or
  close the lock on completion.
- The `run_lock` is per client. It gates the whole client run, and it protects sends in
  particular — combined with the in-code atomic quota reservation (`reserve(sendbox, day)`),
  it prevents two concurrent runs from double-spending a box's daily quota.
- Only a client's own run rebuilds that client's Today View, kanban, reports, or
  `outputs/latest/` files; no other task may rebuild, send, or enrich on its behalf.
- Before sending any run-complete or weekly-report-ready notification, read
  `notifications/notification_log.md` and `outputs/.../{client}-report_state.json` for the
  same client/day. If an equivalent notification was already sent, do not re-send. A resumed
  run records `resumed_from` in the report state so retries stay idempotent.

---

## Daily Run Order (DESIGN §15)

For each client, pinning `target_client_slug`, in this exact order:

1. **Load contract + LOAD LEDGER.** Load Stage 0, Stage 7, and this file; read the automation
   manifest and `outreach-pipeline/automation/update_state.json` (Update Watch is a separate
   task and does not touch clients). Compute the date key in the recorded timezone. Take the
   per-client `run_lock`.
2. **Sync inbox** across all sendboxes (DESIGN §12, Stage 10): run the deterministic classifier
   (see below), split personal mail off, and suppress bounces/unsubs immediately.
3. **Pull tracking** from the worker (DESIGN §11, Stage 12): record open/click activities,
   bot-filtered. Open/click never alone trigger a stage change or auto-action — only a reply
   is conversion evidence.
4. **Semantic triage + `apply-rules`** (DESIGN §7.6, Stage 13): triaged replies create/advance
   deals and tasks; the SLA sweep creates nudge tasks; every stage change carries an
   `evidence_activity_id`. Rules are deterministic and idempotent via guard keys — never
   improvised by the model.
5. **Follow-up advising** (deal-aware, Stage 10): triaged replies become reply drafts;
   due-silent sequences get value-add bumps (never "just following up") → `pending_approval`.
   Every bump micro-refreshes the person's best 1–2 sources to find a fresh hook and to
   invalidate stale hooks before drafting.
6. **Load new pipeline** (cold/trigger campaigns, JIT buffer 3–7 days, Stages 4/5/6): priority
   pick → Tier-1 verify → Tier-2 enrich → step-1 draft → `pending_approval`. At the **END of
   this drafting pass** — after all new-pipeline drafting and **before** any send — render the
   **Approval Report** (`{client}-approval-report.html`, operator-only, NOT scrubbed) per
   DESIGN §14 so the operator can approve in chat; it is **refreshed** in the reports phase
   (step 10) per DESIGN §15.
7. **Send** `outbox/approved/` within quota through `gmail_client.py send` (DESIGN §10). The
   ordered pre-send re-check chain runs in code. Approval happens in chat, at any time — the
   run sends only what is already approved.
8. **Assisted channels:** draft SMS/Messenger/Zalo for no-email contacts only if the campaign
   allows and consent/legal basis exists (DESIGN §9/§16) → Today View copy buttons. The human
   sends and reports back → `assisted_sent` activity.
9. **Compile Today View + regenerate kanban** via `tools/report_renderer.py`.
10. **Reports:** daily ops HTML + **refreshed** Approval Report HTML (first rendered at the end
    of the drafting pass in step 6, per DESIGN §14) + INTERNAL_REPORT (operator-only, not
    scrubbed). On **Mondays**, additionally build the Weekly CRM Report, which is the only
    client-facing output and must pass the Client-Blind Scrub Gate.
11. **Notify the operator** via WideCast `sendTelegramMessage` (email fallback): counts +
    report link → `notifications/notification_log.md`.
12. **Stage 9 audit** → completion gates → release `run_lock`.

---

## Daily Run Algorithm (detailed)

Build-state rule (applies to every step below): the Phase-1 tools (`gmail_client.py`,
`crm_store.py`, `import_leads.py`, `email_verify.py`) exist — sync, apply-rules, and send run
for real. Steps needing a Phase-2 `status: planned` stage/skill (Stage 4 enrich, Stage 6
draft, Stage 10 follow-up) are recorded as `stage_file_pending` (DESIGN §22 R1): load the
covering `docs/DESIGN.md` section, do not improvise, do not enter Last-Resort Recovery, and
continue with the built steps. A genuinely absent Phase-1 tool (partial install only) falls
back to `skipped: tool_not_built` with one `**[ACTION REQUIRED]**` (DESIGN §22 R2).

Pre-loop:

1. Load `clients_index.md`.
2. Identify all clients with `active` status. If the run has a pinned `target_client_slug`,
   restrict the loop to that client only; a client-named task must never process another
   client.
3. Process active clients in `clients_index.md` order unless `schedule.md` defines a different
   priority.

For each active client:

### A. Client load, lock, and validation

1. Load the client's profile file. Validate required fields (voice, offer, compliance/physical
   address, timezone). If incomplete, enter setup-repair mode (Stage 1) instead of sending.
2. Compute the current month folder key `YYYY-MM` and the day key `YYYY-MM-DD` in the recorded
   timezone from `schedule.md`.
3. Take/verify the per-client `run_lock` (Run Locking rule). If a fresh lock exists, log and
   stop for this client.

### B. Sync inbox (Stage 10, DESIGN §12)

4. For each sendbox, advance the cursor: `historyId` (oauth) or IMAP UID (app_password) plus
   `last_successful_sync_ts`. OAuth fallback on an expired `historyId` is
   `q="after:{last_sync_epoch}"` with overlap + dedupe by message id and `nextPageToken`
   handling — never `newer_than:2d`.
5. Run the **deterministic inbound classifier in this exact order** (order is load-bearing):
   1. **DSN/bounce first.** From mailer-daemon/postmaster, or a
      `multipart/report; report-type=delivery-status`, or a `message/delivery-status` part →
      hard (5.x.x) / soft (4.x.x); map to the original via threadId + `rfc_message_id` +
      recipient/sent_at window. DSN is checked **before** threadId because Gmail threads a DSN
      bounce back into the original sent thread — a threadId-first classifier would misread a
      bounce as a reply.
   2. `Auto-Submitted: auto-replied` / out-of-office.
   3. **Unsub alias, before any keyword check.** Any `To`/`Delivered-To` matching
      `{box}+unsub-{token}@` → extract the token → unsubscribe the exact lead. This
      deterministic plus-alias token match runs **before** keyword/semantic classification
      because mailto one-click unsubs often arrive with empty bodies and no keyword to match.
   4. threadId / `In-Reply-To` / `References` match against sent_log → campaign reply → mark
      `reply_untriaged`.
   5. From ∈ contacts but no thread match → `contact_message`.
   6. Else → personal email: **count only, do not store the body, do not deep-read.**
6. Immediately suppress on hard bounce and on any unsubscribe (worker `/u`, mailto plus-alias,
   or explicit remove intent). Suppression is unioned across all identities and follows merge
   chains via `resolve(lead_id)`.
7. **Reply-freezes-sequence invariant:** any inbound reply for a contact freezes the remaining
   bumps in that contact's sequence until triage completes. This invariant is enforced in code
   at both draft-time and send-time, not only here.

### C. Pull tracking (Stage 12, DESIGN §11)

8. Pull worker events from `GET /events?since={seq}` (Bearer `TRACKER_API_KEY`) and reconcile
   `unsub:{token}` state. Record open/click activities, bot-filtered (UA class, click-with-no-
   prior-open, all-links-within-N-seconds, datacenter ASN). Opens are labeled "estimated."
9. **Send-safety gate:** if the tracker pull has not succeeded within the configured window
   for a box, block sending for that box this run (so worker/mailto unsubs cannot sit
   unhonored beyond the window) and record the blocker.

### D. Semantic triage + apply-rules (Stage 13, DESIGN §7.6)

10. Semantically triage each `reply_untriaged` → `positive | question | objection | negative |
    remove_intent`. `negative`/`remove_intent` (even without the literal word "unsubscribe")
    → suppression, or an `**[ACTION REQUIRED]**` confirm task that blocks further sends to
    that contact until resolved.
11. Run `crm_store.py apply-rules` (idempotent via `(rule_id, trigger_activity_id)` guard
    keys): positive reply → create deal + reply-within-4h task + freeze sequence; question →
    deal at `engaged` + draft reply for approval; negative/remove → suppress + close open
    tasks; SLA sweep → nudge tasks + report flag; won → lifecycle=customer + onboarding task.
    Every stage change carries an `evidence_activity_id`.
12. Contacts with a pending merge proposal are excluded from every campaign queue until
    resolved.

### E. Follow-up advising (Stage 10)

13. For triaged replies needing a human-approved answer, draft the reply and place it in
    `outbox/pending_approval/`.
14. For due-silent sequences within a healthy assigned sendbox, micro-refresh the person's 1–2
    best sources (freshness gate), draft a value-add bump carrying NEW value, and place it in
    `pending_approval`. If the assigned box is broken, the bump **waits** (never reassigned) and
    a `**[ACTION REQUIRED]**` re-auth is raised; the report shows "N follow-ups blocked."

### F. Load new pipeline (Stages 4/5/6)

15. Maintain a JIT buffer of 3–7 days of drafts. Priority-pick from cold/trigger campaigns,
    honoring cross-campaign rules: a contact in an active sequence of campaign A is not drafted
    by B; a hook already used on a person may not open a second campaign.
16. Tier-1 verify (check dossier TTL first; cheap subagent) → Tier-2 enrich (main model, visit
    known URLs per the readability table, extract hooks with `evidence_url`, distill
    `writing_brief`, score `personalization_confidence`).
17. Draft step-1 through skill `email-writing` (goal_type → structure). A draft may contain
    only details present in the dossier with an `evidence_url`; step-1 subjects must not begin
    `Re:`/`Fwd:`. Place in `pending_approval`. At the **END of this drafting pass** — after all
    new-pipeline drafting and **before** the Send step (section G) — render the **Approval
    Report** (`{client}-approval-report.html`, operator-only, NOT scrubbed) per DESIGN §14 so
    the operator can approve in chat; it is **refreshed** in the reports phase (step 21) per
    DESIGN §15.

### G. Send approved (Stage 8, DESIGN §10)

18. Send `outbox/approved/` through `gmail_client.py send`, which runs the full ordered
    pre-send re-check in code: `resolve(lead)` → global + client suppression (with fresh
    track-pull) → `channels.email.status` → atomic quota reservation → warmup cap → two-tier
    domain cap → send-window (recipient tz) → guessed cap + guessed-approval flag →
    sequence-freeze check → step-1 subject lint. Sticky sender: rotation picks the box only on
    step 1, then `assigned_sendbox` is fixed. Record `sent_log.jsonl`, append `email_sent`,
    sleep jitter 30–180s. Approval itself happens in chat, any time; the run only sends what is
    already approved.

### H. Assisted channels (DESIGN §9/§16)

19. For no-email contacts, draft SMS/Messenger/Zalo only when the campaign allows and a
    documented legal basis exists (US SMS gated on `{optin_source, optin_at,
    evidence_activity_id}` or existing relationship; Zalo cold-messaging strangers stays off by
    default). Surface each draft with its legal basis in Today View copy buttons. The human
    sends and reports back → `assisted_sent` activity.

### I. Today View + kanban (Stage 14/15)

20. Compile the Today View and regenerate the deal kanban with `tools/report_renderer.py`
    (reusing its contenteditable + Copy-button blocks).

### J. Reports (Stage 15 + report-design skill)

21. Generate the operator-only outputs (NOT scrubbed): `{client}-daily-ops.html`, the
    **refreshed** `{client}-approval-report.html` (first rendered at the end of the drafting
    pass in step 17, per DESIGN §14; refreshed here per DESIGN §15), `{client}-today-view.html`,
    and `{client}-INTERNAL_REPORT.html` (clearly labeled `INTERNAL_REPORT - Not for client
    sharing`; keep OutreachCRM/WideCast/provider/Telegram/API-key/config/sendbox/tracker/
    debug details here).
22. **On Mondays only**, build the client-facing `{client}-weekly-client-report.html` (and its
    `.pdf` companion) and run the Client-Blind Scrub Gate on it via
    `tools/report_renderer.py` (`CLIENT_BLIND_TERMS`). It must not mention OutreachCRM,
    WideCast, provider tooling, OpenAPI, MCP, sendbox, gmail_client, crm_store, tracker/`trk.`,
    HMAC, token.json, sent_log, suppression, warmup, quota, guessed, automation/scheduled task,
    API key/config, Telegram, agent/tool/debug details, or `INTERNAL_REPORT`. The weekly report
    is the only client-facing output.
23. Write/update `outputs/.../{client}-report_state.json` and reconcile it with every rendered
    artifact (counts, per-section status, timestamps). No artifact may say `in progress` while
    another says `complete`. Update `outputs/latest/...` copies.

### K. Notify the operator

24. Run the Provider Report Delivery Capability Check using **this client's** provider
    config/OpenAPI identity first — a global MCP/native provider account in the current session
    is not proof the client has notification configured. Record it in `INTERNAL_REPORT`.
25. If WideCast OpenAPI notification is configured and the discovered spec exposes an
    HTML-capable `uploadAsset`, upload the operator report (`{client}-daily-ops.html`, or the
    Monday weekly report link) with `text/html`, then send `sendTelegramMessage` (email
    fallback) including: run status, counts (replies triaged, deals moved, drafts pending,
    emails sent, bounces/unsubs suppressed), the report link/path, and any
    `**[ACTION REQUIRED]**`. Provider-hosted URLs are operator handoff links, not client-share
    links.
26. If HTML upload is unavailable or fails, log the exact blocker and still notify with the
    best available local/hosted report path plus the counts and required actions.
27. If provider config is missing, auth fails, discovery fails, identity mismatches, or the
    only visible account is an unproven global account, log the provider-neutral blocker and
    hand off the report path/link in chat or an authorized fallback channel.
28. A notification that only says "the run is complete" with no report link/path and no counts
    is invalid; send a correction immediately and log it. Record every attempt in
    `notifications/notification_log.md` (deduped per client/day).

### L. Stage 9 audit + release

29. Load Stage 9 (`09_OPERATIONS_SAFETY_AUDIT.md`) and pass the completion gates before
    claiming the run is complete: suppression honored at every send-capable path; no draft
    contains a detail without an `evidence_url`; step-1 subjects not `Re:`/`Fwd:`; quota/warmup
    respected; sticky-sender preserved; sent_log/activities/report_state reconciled; scrub gate
    passed on the weekly report; no direct CRM file writes outside `crm_store.py`.
30. Release/close the per-client `run_lock`.

The daily run is complete only when every active client is processed or explicitly logged as
skipped, and the operator has been notified through the configured channel with counts and the
report link/path.

---

## Schedule Contract (`schedule.md` and automation files)

During one-time setup, after the profile, sendboxes, and first campaign goal are known and
before the client-specific automation task is marked `ready_for_automation_first_run`, ask the
human — via `**[ACTION REQUIRED]**` — whether they want daily, multiple-times-daily, weekly,
manual-only, first-run-only, or another cadence.

Then write or update, at minimum:

- `outreach-pipeline/schedule.md`
- `outreach-pipeline/automation/automation_manifest.md`
- `outreach-pipeline/automation/scheduled_run_prompt.md`
- `outreach-pipeline/automation/resync_log.md`
- the native automation/scheduled task body (or the pending-prompt handoff)

OutreachCRM has no separate external scheduler config file of its own. The cadence is
expressed in three places that the daily run reads at run time:

- the **scheduling mechanism** (native task / cron / launchd / etc.), recorded in
  `schedule.md` and instantiated in the environment;
- each campaign's **`sequence[].gap_days`** in `campaign_config.json`, which determines when a
  contact's next bump is due;
- the JIT buffer target (3–7 days) that governs how far ahead the run drafts new pipeline.

### `schedule.md` — human-readable, required fields

`schedule.md` must record, in prose the next agent can repair from:

- cadence (daily / multiple-times-daily / weekly / manual-only / first-run-only);
- the chosen scheduling mechanism and how it is instantiated;
- included clients and their per-client task names;
- the recorded local timezone (see the timezone rule above);
- the notification channel (WideCast Telegram/email fallback → Gmail/email → `local_path_only`);
- the weekly-report day (default Monday, in the recorded timezone);
- the last resync timestamp.

Example schedule mechanisms recorded in `schedule.md`:

```text
scheduling_mechanism: native_ai_scheduled_task
run_window: daily 09:00 local
timezone: America/Chicago
weekly_client_report_day: monday
clients:
  - slug: avenngo-realty-austin
    task_name: "AvenNgo - OutreachCRM Daily Run"
    target_client_slug: avenngo-realty-austin
notification_channel: widecast_telegram_email_fallback
last_resync: 2026-07-15T09:00:00-05:00
```

For a cron-based environment, record the intended cadence in `schedule.md`, but be honest
about what cron can and cannot do:

```text
# OutreachCRM daily run — AvenNgo (09:00 America/Chicago)
# cron cannot launch the agent runtime by itself; a human-created wrapper must invoke the
# agent (e.g. codex CLI) with the scheduled_run_prompt.md contents. OutreachCRM does not ship
# a runner script. Prefer a native AI scheduled task or macOS launchd (below), which can drive
# the agent directly.
0 9 * * *  # -> human wrapper that invokes the agent with automation/scheduled_run_prompt.md
```

For macOS launchd, record the LaunchAgent label and plist path in `schedule.md` and hand the
human the install command via `**[ACTION REQUIRED]**`; do not install a LaunchAgent from
inside the AI sandbox during setup.

- For **multiple runs per day**, record several run windows (e.g. morning / midday /
  afternoon) in `schedule.md` and the mechanism; each window re-checks the `run_lock` so a
  slow earlier run is not duplicated.
- For **manual-only** mode, record `cadence: manual_only`, provide the exact prompt the human
  runs on demand, and rely on no unattended trigger. Pin the prompt to one
  `target_client_slug` (or clearly label it the multi-client manual variant that loops every
  active client in `clients_index.md`, one per `run_lock`). The prompt must run the full Daily
  Run order, render the Approval Report and hold at the chat approval gate before any send
  (nothing leaves without an explicit "approve"), pass the Stage 9 audit, and release the
  per-client `run_lock` on completion.

  Example manual daily-run prompt (single client, pinned slug):

  ```md
  Run the OutreachCRM daily run for target_client_slug: avenngo-realty-austin only. Load the stage files fresh with LOAD LEDGER entries. Sync inboxes, pull tracking, triage replies, run apply-rules, advise follow-ups, load new pipeline, render the Approval Report and hold at the chat approval gate — send only outbox/approved drafts within quota after I approve — draft assisted channels where allowed, refresh Today View + kanban and the reports (plus the Monday weekly client report through the scrub gate), notify the operator with counts and the report link, pass the Stage 9 audit, and release the run_lock.
  ```

  For the multi-client manual variant, replace the pinned slug with "every active client in
  clients_index.md, one per run_lock," keeping the same approval-gate, Stage 9, and
  run_lock-release requirements.
- For **weekly** cadence, still run inbox sync, tracking pull, triage, and suppression on the
  configured days; the client-facing weekly report is produced on the weekly-report day.

### `automation_manifest.md` and `scheduled_run_prompt.md`

Write `outreach-pipeline/automation/automation_manifest.md` and
`outreach-pipeline/automation/scheduled_run_prompt.md` so a future agent can repair or resync
the actual scheduled task prompt instead of relying on memory. The `scheduled_run_prompt.md`
must pin `target_client_slug`, instruct the run to load the playbooks fresh at run time, and
execute the full Daily Run order for that one client.

Example `scheduled_run_prompt.md` body:

```md
Load playbooks/SCHEDULED_RUN_ENTRYPOINT.md, then run the OutreachCRM daily run for
target_client_slug: avenngo-realty-austin only. Do not process any other client. Load the
stage files fresh (0, 7, this automation contract, 10, 12, 13, 14, and 4/5/6/8/9/15 as needed)
with LOAD LEDGER entries. Sync all sendbox inboxes, pull tracking, triage replies and run
apply-rules, advise follow-ups, load new pipeline (verify + enrich + draft to pending_approval),
send only outbox/approved within quota, draft assisted channels where allowed, refresh Today
View + kanban, render operator reports (plus the Monday weekly client report through the scrub
gate), notify the operator via WideCast Telegram/email fallback with counts and the report link,
pass the Stage 9 audit, and release the run_lock.
```

If the client has no notification channel yet, still write these files, set
`notification_channel: local_path_only`, and give the human one `**[ACTION REQUIRED]**`
instruction to connect WideCast (API key + Telegram/email fallback) or Gmail/email.
