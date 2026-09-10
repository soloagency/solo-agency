# Solo Agency Setup Flow Entrypoint

Use this file as the entrypoint for setup/configuration sessions.

Setup Flow is the control plane. It configures Solo Agency so automation tasks run correctly later. It does not run operational reports.

## Runtime Requirement

Before setup proceeds, verify or explain that Solo Agency needs Codex, Claude Desktop/Cowork, Hermes, OpenClaw, or a comparable desktop/local AI agent runtime with workspace file access, automation/scheduled tasks, and multiple parallel/sub-agent work streams. Do not present a plain web chat as the primary runtime. Web chat can review results, but it cannot reliably host the file state, Local Collector handoff, scheduled automation, and multi-agent work this setup configures.

## Setup Flow Contract

0. **First words come first.** Before step 1 — before any load, ledger or question — send the Team Leader introduction from `SOLO_AGENCY_PLAYBOOK.md` ("First Words"), which ends by asking the Boss to name this chat "Team Leader" and pin it (so tomorrow's orders come back to the same conversation, never to a fresh agent with no ledger). Then load `playbooks/TEAM_MODEL.md` and `playbooks/NEXT_JOB_CATALOGUE.md` with the other entry files. Create `daily-content-pipeline/automation/boss_orders.md` (the ledger header from `playbooks/TEAM_MODEL.md`) if it does not exist, and record every request or goal the human states during setup as a row before acting on it.
1. Load `SOLO_AGENCY_PLAYBOOK.md` and `playbooks/LOAD_LEDGER_PROTOCOL.md`. **Full-load discipline applies to every file below: each load needs a LOAD LEDGER (read to the last line; compare `playbooks/LOAD_MANIFEST.md` when present; ledger each named dependency). A truncated / "output too large" / partial read = NOT loaded — re-read in chunks before acting. No side-effect step without a PASS ledger for the stage(s) it needs.**
2. Load `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`, `playbooks/01_BASIC_PROFILE_PUBLIC_REPORT.md`, `playbooks/04_DAILY_SCHEDULE.md`, `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`, and `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md`.
3. Load `playbooks/PRIVATE_SOURCE_GATE.md`, `playbooks/02_PRIVATE_SOURCE_SETUP.md`, and `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md` when private data sources, client Chrome profiles, client extensions, or Local Collector setup are involved.
4. Load `playbooks/11_UPDATE_AND_VERSION_WATCH.md` when the human asks for update/upgrade/sync latest, when setup repair suspects stale playbooks/code, or when configuring the `Solo Agency - GitHub Update Watch` maintenance task.
5. Create or update client setup, public data sources, private data sources approval state, extension folders, collector config, schedule files, automation manifests, scheduled prompts, update-watch state, and resync logs.
6. Do not run public scans, private data source scans, reports, first agency runs, production, rendering, publishing, analytics scans, or outreach in Setup Flow. One narrow exception: the step-6 private data source DISCOVERY pass may run inside Setup Flow when the human approved it in this session and the Local Collector plus the matching client extension are verified healthy — Local Collector only, approved categories only, output limited to the approval shortlist and saved source configuration. Even then, Setup Flow must not analyze the collected data or produce any report/idea/draft from it.
7. If the human asks to run, create, generate, show, refresh, or update a report inside Setup Flow, this is a hard stop for operational work. The setup chat stays Setup Flow; the request does not become Automation Flow. Verify or create the relevant automation task, resync its prompt/config if needed, and verify/resync the client-specific automation task and then START IT — the agent dispatches the task through the scheduler's own run-now, tells the human it has been dispatched and where the result will appear, and reports back when it lands; only a runtime that genuinely cannot start its own tasks falls back to naming the task for the human to run.
8. If the human says only `update`, `upgrade`, `cập nhật`, `sync latest`, or `pull latest`, treat that as the Stage 11 Solo Agency update command, not as `update a report`.
9. Every client-specific automation task name must begin with the client name, for example `AvenNgo - Solo Agency Daily Run`.
10. Every per-client Chrome extension display name must begin with the client name, for example `AvenNgo - Solo Agency Collector`.
11. PDNA provider setup must be client-scoped: read/write the current client's `integrations/providers/` files and verify provider identity through the client's OpenAPI/API-key config before claiming production, distribution, notification, analytics, credits, or connected platforms are available. Default setup is WideCast API-key setup: ask only for the client's WideCast API key, then let the agent configure/verify/discover/resync the rest. Do not ask provider/scope/spend/publish/account-identity questions unless the human explicitly requests a non-default provider or specialist stack. Do not use a global MCP/native provider account as proof of this client's PDNA status.
12. After any approved config change or applied Solo Agency update, perform Automation Resync if a schedule/automation already exists.
13. Setup Flow completion means `ready_for_automation_first_run` or `ready_for_next_automation_run`. The completion message restates, in one line, that every future request goes to this chat with the Team Leader, and lists the scheduled tasks just created as team members with their roles (`playbooks/TEAM_MODEL.md`, Roster).
14. Every human question, approval request, one-line Terminal/PowerShell command, Chrome `Load unpacked` instruction, provider/API-key setup request, and native automation task edit must use the `**[ACTION REQUIRED]**` block from `SOLO_AGENCY_PLAYBOOK.md`. If setup continues without needing the human, end the reply with next-action guidance per the root Next-Action Guidance Rule instead of `No action required right now.`

## Fresh Source Acquisition Hard Gate

Before copying playbooks, `solo-agency-collector/` artifacts, extension templates, scripts, or zipped assets into the human's setup, the setup agent must verify the source repo.

Required behavior:

- Use the current setup root if it is already a verified clone of `https://github.com/soloagency/solo-agency`; otherwise clone into a fresh unique `mktemp -d` directory.
- Do not use fixed shared fallback folders such as `/tmp/solo-agency`, `/var/tmp/solo-agency`, or `/dev/shm/solo-agency`.
- Do not trust a folder that lacks `.git`, has the wrong owner, has an old timestamp, or could not be removed/updated.
- Verify `git remote get-url origin`, `git rev-parse HEAD`, and `git ls-remote origin refs/heads/main`; local `HEAD` must match GitHub `main` before the agent reads or copies files from that checkout.
- If GitHub access fails because of network or sandbox restrictions, request the needed permission or give the human one exact clone/download command. Do not proceed with stale local code.
- Record the verified source path and commit hash in `daily-content-pipeline/collector/collector_setup_status.md` or `daily-content-pipeline/automation/resync_log.md` when those files exist.
- For a full update command, record the verified source path, commit hash, change classification, backup path, client resync state, bridge action requirement, and extension reload requirement in `daily-content-pipeline/automation/update_state.json` and `update_log.md`.

## Setup Blocker Recovery

If setup hits any unexpected blocker, old/corrupt artifact, missing file, permission mismatch, stale extension/collector behavior, instruction conflict, or dead end:

1. Assume the setup agent may be using an old Solo Agency version.
2. Re-run the Fresh Source Acquisition Hard Gate against GitHub `main`.
3. Reload `SOLO_AGENCY_PLAYBOOK.md` plus the relevant child playbooks from the verified latest source.
4. If the newest playbook fixes the setup path, continue from the latest rule and resync the client setup/automation state.
5. If the newest GitHub version still leaves the setup blocked, report it through the channel this install actually has, in this order: **WideCast** when the client's `provider_config.local.json` carries a WideCast key (`<bridge> tool provider --config <that file> --defaults daily-content-pipeline/provider_defaults.json call --operation reportError --body '{"module":"solo-agency/<area>","error_message":"<verbatim>","context":{…}}'` — free on every plan, private, no human step); the **Solo Agency Facebook support group** when the install is keyless or that call is unavailable (`playbooks/TEAM_MODEL.md`, Support requests — the Boss approves the exact text first); a **GitHub issue** only when the runtime ALREADY has an authorized identity (`gh auth status` passes, `GITHUB_TOKEN` / `GH_TOKEN` / `SOLO_AGENCY_GITHUB_ISSUE_TOKEN`, or a maintainer bot) — never ask a human to create a GitHub account; and a ready-to-post draft under `daily-content-pipeline/automation/issues/` when none of those exist. Record whichever channel was used in `daily-content-pipeline/automation/github_issues.md` and check it in later runs. During setup the install is usually still keyless, so in practice this means the support group — which is exactly why the group post is allowed on every plan.

Do not include private client data, secrets, cookies, tokens, raw private data source captures, or logged-in screenshots in GitHub issues. If direct issue creation/sending is unavailable, write the draft under `daily-content-pipeline/automation/issues/`, track it in `daily-content-pipeline/automation/github_issues.md`, and tell the human the path.

## Required Setup Output

For each configured client, Setup Flow must leave these current:

- Client Intelligence Profile.
- public data sources and keyword bank.
- private data sources approval state.
- `extensions/{client_slug}/manifest.json`.
- `extensions/{client_slug}/client_binding.json`.
- `daily-content-pipeline/collector/extension_registry.json`.
- `daily-content-pipeline/collector/collector_config.json`.
- `daily-content-pipeline/schedule.md`.
- `daily-content-pipeline/automation/automation_manifest.md`.
- `daily-content-pipeline/automation/scheduled_run_prompt.md`.
- `daily-content-pipeline/automation/update_state.json` and `update_log.md` when update/watch has been checked, configured, or applied.
- `daily-content-pipeline/automation/resync_log.md`.

If the native automation task prompt cannot be updated directly, mark `automation_prompt_update_pending` in the manifest and schedule, then give the human one concrete instruction to update the task prompt.

For every new client, the setup handoff must include the dedicated extension install instructions, not just a status line. Show the absolute `extensions/{client_slug}/` folder path and the exact Chrome `Load unpacked` steps for the matching client Chrome profile/account inside a `**[ACTION REQUIRED]**` block. Before showing that path or the bridge start command, run the Stage 8 Source Safety Pre-Check and precede the install block with one short plain-language line confirming the collector's code was read and only runs locally (safe to install). If the pre-check does not pass, do not show the install steps; raise it to the operator instead.

**Facebook Login Reminder.** Immediately after that install block — once the bridge and this client's extension are confirmed installed and Local Collector health looks reachable — deliver this reminder inside its own `**[ACTION REQUIRED]**` block, in the human's language:

```text
Để tìm lead trên Facebook, hãy đăng nhập Facebook trên Chrome của máy này (tài khoản cá nhân của bạn là đủ). Nếu bạn không cần nguồn lead từ Facebook, trả lời "bỏ qua Facebook" — vòng quét vẫn chạy trên web nhưng số lead tìm được sẽ ít hơn đáng kể.
```

Immediately after that block (not part of the verbatim text audited above, and not itself
`**[ACTION REQUIRED]**`), add one priming sentence in the human's own words and language: a plain
fact that every lead this pass finds lands straight in the client's CRM, and Free keeps a first
batch of contacts fully open.

Record the answer as `facebook_lead_source: enabled|skipped|pending` in the Client Intelligence
Profile (see `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`, Client Intelligence Profile fields). This
reminder fires exactly once per
client setup, here, right after the bridge + extension install step and before the first run is
dispatched. It fires again later only inside a scheduled/automation run whose extension health shows
logged-out/stale (`playbooks/SCHEDULED_RUN_ENTRYPOINT.md` step 12D).

The client's first dispatched run (Report Request Hard Stop below, and the Setup-Complete Closing
Template's offer 1) is also the first run of the Facebook Discovery Pass
(`playbooks/10_LEAD_COMPETITOR_DETECTION.md`, "Facebook Discovery Pass (step 11C of the daily run)")
whenever `facebook_lead_source` came back `enabled`, or is still `pending` at dispatch time on this
client's very first run (Setup already showed the reminder above) — that first pass uses the FIRST
RUN budget (≤ 21 collector calls, `max_pages` ≤ 4, spread over ≥ 4 hours), never the smaller DAILY
budget. If the answer was `skipped`, the dispatched run skips the pass entirely and its report says
plainly that the lead count is lower because Facebook was skipped.

After schedule/automation exists, set up the separate maintenance task `Solo Agency - GitHub Update Watch` - do not silently skip it. Create the native task if the runtime allows; otherwise write the exact prompt to `daily-content-pipeline/automation/update_watch_prompt.md`, record `update_watch_task_prompt_pending`, AND end with an `**[ACTION REQUIRED]**` block naming the task and the exact way to create it. Default posture is notify-first (`auto_apply_approved: false`).

## Report Request Hard Stop

When the human asks for a report/run while this entrypoint is active, the only valid response is:

1. State that Setup Flow does not run reports.
2. Finish or resync the client-specific automation task.
3. Say that the first run has been dispatched, naming the task and whether it covers public data sources only or public plus activated private data sources; if it could not be started, give the exact task name for the human to run and why.
4. If the native automation UI requires human action, provide that one exact action in a `**[ACTION REQUIRED]**` block.
5. End with a `**[ACTION REQUIRED]**` block naming the exact client-specific automation task to run for the report, AND a feature-discovery block introducing 2-3 unused headline capabilities from `playbooks/FEATURE_CATALOG.md` (Feature Discovery Rule) - setup never ends flat. When no action is needed, still end with next-action guidance plus the feature-discovery block.

### Setup-Complete Closing Template (MANDATORY shape)

The FINAL message of a completed setup must end with this three-offer block (translate to the human's language; adapt names, never the shape). Listing trigger phrases "for later" is NOT compliance: offer 1 must be startable NOW, and the message MUST end with exactly one question.

```text
**[ACTION REQUIRED] — bước tiếp theo, chọn một:**
1. **Chạy báo cáo đầu tiên ngay bây giờ** — mở một phiên chat mới và ra lệnh:
   `run task "{Client} - Daily Run"` (không cần chờ lịch {time} {days}). Lead tìm được sẽ vào
   thẳng CRM chung, và gói Free đang mở sẵn một số lượng contact đầu tiên miễn phí.
2. **Tạo campaign cold-email đầu tiên (OutreachCRM)** — chỉ cần 3 thứ: list lead,
   Gmail App Password, và xác nhận goal+URL. Nói: "set up a cold-email campaign".
3. **{one more unused feature from FEATURE_CATALOG.md, value-first, with its trigger phrase}**

Bạn muốn bắt đầu với cái nào? (Trả lời 1, 2, 3 — hoặc hỏi tôi bất kỳ điều gì khác.)
```

Hard gates on this template: offer 1 is ALWAYS the immediate first run of the just-created automation task (a scheduled future run is never a substitute); one offer is ALWAYS from the OTHER product side (content setup offers Outreach, Outreach setup offers content/video — the cross-introduce rule); the closing line is a QUESTION. A setup-complete message whose last line is not a question is a Stage-9 audit failure. The priming clause folded into offer 1 (leads land in the CRM; Free keeps a first batch of contacts open) is spoken in the human's own words and language like the rest of the block, never this fixed sentence verbatim, and never with an estimated contact count — the shape stays three offers and one closing question; it does not become a fourth offer or a second question.

Do not ask whether to run the report now. Do not load `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` inside the setup chat. Do not perform public research, private data source collection (one exception: the step-6 discovery pass per item 6 of the Setup Flow Contract), report generation, idea matrix updates, Lead & Competitor Opportunities, draft generation, analytics scans, or notification delivery (one exception: the single step-7 WideCast confirmation ping that verifies the notification channel right after the human provides the API key) in Setup Flow.
