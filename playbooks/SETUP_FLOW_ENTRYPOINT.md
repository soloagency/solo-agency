# Solo Agency Setup Flow Entrypoint

Use this file as the entrypoint for setup/configuration sessions.

Setup Flow is the control plane. It configures Solo Agency so automation tasks run correctly later. It does not run operational reports.

## Runtime Requirement

Before setup proceeds, verify or explain that Solo Agency needs Codex, Claude Desktop/Cowork, Hermes, OpenClaw, or a comparable desktop/local AI agent runtime with workspace file access, automation/scheduled tasks, and multiple parallel/sub-agent work streams. Do not present a plain web chat as the primary runtime. Web chat can review results, but it cannot reliably host the file state, Local Collector handoff, scheduled automation, and multi-agent work this setup configures.

## Setup Flow Contract

0. **First words come first.** Before step 1 — before any load, ledger or question — send the Team Leader introduction from `SOLO_AGENCY_PLAYBOOK.md` ("First Words"), which ends by asking the Boss to name this chat "Team Leader" and pin it (so tomorrow's orders come back to the same conversation, never to a fresh agent with no ledger). Then load `playbooks/TEAM_MODEL.md` and `playbooks/NEXT_JOB_CATALOGUE.md` with the other entry files. Create `daily-content-pipeline/automation/boss_orders.md` (the ledger header from `playbooks/TEAM_MODEL.md`) if it does not exist, and record every request or goal the human states during setup as a row before acting on it.
1. Load `SOLO_AGENCY_PLAYBOOK.md` and `playbooks/LOAD_LEDGER_PROTOCOL.md`. **Full-load discipline applies to every file below: each load needs a LOAD LEDGER (read to the last line; compare `playbooks/LOAD_MANIFEST.md` when present; ledger each named dependency). A truncated / "output too large" / partial read = NOT loaded — re-read in chunks before acting. No side-effect step without a PASS ledger for the stage(s) it needs.**
2. Load `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`, `playbooks/01_BASIC_PROFILE_PUBLIC_REPORT.md`, `playbooks/04_DAILY_SCHEDULE.md`, `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`, and `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md`.
3. Load `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md` before step 4 below — every setup now installs the bridge and this client's extension, not only when private data sources are already known. Load `playbooks/PRIVATE_SOURCE_GATE.md` and `playbooks/02_PRIVATE_SOURCE_SETUP.md` separately, later, before the private data source checkpoint (step 7).
4. **Kết nối Facebook**, right after profile inference (the human-facing roadmap's items 1-3: business context, industry/audience inference, pain points/pillars). Install the Local Collector bridge and this client's Chrome extension, then deliver the Facebook Login Reminder and record `facebook_lead_source` — the full step contract, including the local/remote install rule and the two-gesture extension install, lives below under "Kết nối Facebook (step 4)".
5. Load `playbooks/11_UPDATE_AND_VERSION_WATCH.md` when the human asks for update/upgrade/sync latest, when setup repair suspects stale playbooks/code, or when configuring the `Solo Agency - GitHub Update Watch` maintenance task.
6. Create or update client setup, public data sources, private data sources approval state, extension folders, collector config, schedule files, automation manifests, scheduled prompts, update-watch state, and resync logs.
7. Do not run public scans, private data source scans, reports, first agency runs, production, rendering, publishing, analytics scans, or outreach in Setup Flow. One narrow exception: the step-7 private data source DISCOVERY pass may run inside Setup Flow when the human approved it in this session and the Local Collector plus the matching client extension are verified healthy — Local Collector only, approved categories only, output limited to the approval shortlist and saved source configuration. Even then, Setup Flow must not analyze the collected data or produce any report/idea/draft from it.
8. If the human asks to run, create, generate, show, refresh, or update a report inside Setup Flow, this is a hard stop for operational work. The setup chat stays Setup Flow; the request does not become Automation Flow. Verify or create the relevant automation task, resync its prompt/config if needed, and verify/resync the client-specific automation task and then START IT — the agent dispatches the task through the scheduler's own run-now, tells the human it has been dispatched and where the result will appear, and reports back when it lands; only a runtime that genuinely cannot start its own tasks falls back to naming the task for the human to run.
9. If the human says only `update`, `upgrade`, `cập nhật`, `sync latest`, or `pull latest`, treat that as the Stage 11 Solo Agency update command, not as `update a report`.
10. Every client-specific automation task name must begin with the client name, for example `AvenNgo - Solo Agency Daily Run`.
11. Every per-client Chrome extension display name must begin with the client name, for example `AvenNgo - Solo Agency Collector`.
12. PDNA provider setup must be client-scoped: read/write the current client's `integrations/providers/` files and verify provider identity through the client's OpenAPI/API-key config before claiming production, distribution, notification, analytics, credits, or connected platforms are available. Default setup is WideCast API-key setup: ask only for the client's WideCast API key, then let the agent configure/verify/discover/resync the rest. Do not ask provider/scope/spend/publish/account-identity questions unless the human explicitly requests a non-default provider or specialist stack. Do not use a global MCP/native provider account as proof of this client's PDNA status.
13. After any approved config change or applied Solo Agency update, perform Automation Resync if a schedule/automation already exists.
14. Setup Flow completion means `ready_for_automation_first_run` or `ready_for_next_automation_run`. The completion message restates, in one line, that every future request goes to this chat with the Team Leader, and lists the scheduled tasks just created as team members with their roles (`playbooks/TEAM_MODEL.md`, Roster).
15. Every human question, approval request, one-line Terminal/PowerShell command, Chrome `Load unpacked` instruction, provider/API-key setup request, and native automation task edit must use the `**[ACTION REQUIRED]**` block from `SOLO_AGENCY_PLAYBOOK.md`. If setup continues without needing the human, end the reply with next-action guidance per the root Next-Action Guidance Rule instead of `No action required right now.`
16. **Standing Invitation.** Every Setup Flow message that asks the human for anything — a question, a choice, a consent, an `**[ACTION REQUIRED]**` gesture such as the two-gesture extension install, a paste — closes with one short line, in the human's own language, inviting them to ask about anything unclear before they answer (`SOLO_AGENCY_PLAYBOOK.md`, "Standing Invitation"): fresh wording each time, never dropped, never itself the closing question, and never a substitute for the required next-step question or the `**[ACTION REQUIRED]**` block. Where the message must also end with exactly one question, the invitation line comes before that question, not after it.

## Kết nối Facebook (step 4)

This step runs immediately after profile inference (roadmap items 1-3) and before public data sources/keyword bank (roadmap item 5). It replaces the old positioning where the bridge/extension install and the Facebook Login Reminder sat inside "Required Setup Output" right before dispatch — they now happen here, early, so `facebook_lead_source` is known well before the automation task and the private data source checkpoint.

Open with one plain-language value sentence in the human's language, for example:

```text
Bước này để em quét được Facebook: feed, người và nhóm public — đó là nơi phần lớn lead đầu tiên sẽ đến từ.
```

Dashboard text is English; you translate for the Boss in chat — every `/ui/...` page the human sees
in this step (browsers, extension, help) renders in English regardless of the human's language.

**Bridge install — agent-run on a local runtime, handed off on a remote runtime.** Run the Stage 8 Source Safety Pre-Check first. Then:

- **Local runtime** (Codex CLI, Claude Code desktop/CLI, Hermes, OpenClaw, or a comparable runtime that can see the install root on its own filesystem): the agent installs and starts the bridge itself. It writes `setup_collector.sh` (`setup_collector.ps1` on Windows) with the absolute install path, says ONE plain-language line confirming the code was read and it only runs locally — for example `Em đã đọc mã của collector: nó chỉ chạy trên máy này, không gửi dữ liệu đi đâu. Em cài và bật nó ngay bây giờ, khoảng một phút.` — asks for consent once, runs the script itself, and waits for `GET http://127.0.0.1:17321/status` to answer (up to 60 seconds). The script hands the process to an OS-level autostart service (macOS LaunchAgent, Linux systemd user unit, or a Windows logon Scheduled Task, per `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md`) — the agent's own shell never owns that process, so this is not the in-sandbox launch the old ban addressed. On Windows, SmartScreen may ask for one click; mention it only if it actually happens.
- **Remote runtime** (a hosted sandbox where `127.0.0.1` is not the human's machine): detected when the agent cannot see the install root on its own filesystem, or `/status` still fails 60 seconds after a bootstrap attempt. Only then does the agent hand the human the one exact one-line command (`bash "/ABSOLUTE/PATH/TO/solo-agency-local-collector/setup_collector.sh"` or the Windows equivalent) instead of running it itself.
- Do not show the install command/path (local or remote) before the pre-check passes; if it fails, stop and raise it to the operator instead.

**Extension install — two gestures.** Point the human at that client's `/ui/{client}/extension` dashboard page: its one primary button both reveals the `extensions/{client_slug}_extension/` folder (Finder/Explorer, or the OS equivalent — see the table below) and opens the chosen browser on its `<scheme>://extensions/` page (the bridge runs this on the human's own machine — see "Browser + profile handling" immediately below for exactly which browser, which profile, and which shell command). The page then shows two steps: turn on Developer mode once, and drag the folder onto the page. Right under those two steps the page now also plays a short recorded video for the chosen browser (Chrome or Edge) — point the human at it directly on the page rather than only describing the steps in words. Which browser and which profile it opens is not a fixed "first client's existing profile" assumption — the bridge detects installed browsers and asks at most one short question when there is a real choice; see the table below. The agent (local runtime) can trigger both actions itself via the bridge API, then polls `/status` until `extension_health` is recent (75-second grace) and celebrates the connection in chat. Alongside the reveal, the agent says this one fixed sentence, in the human's language, so the drag lands on the right folder: "Chọn đúng thư mục tên `{client}_extension` mà em vừa mở — không chọn thư mục `chrome-extension` nằm trong mã nguồn." (OWNER DECISIONS 2026-09-10 item A). This step always shows both help-video links — Chrome install, Edge install — even when the agent ran the bridge itself, so the human has more than one way to finish; until the owner records them, reference the future page `http://127.0.0.1:17321/ui/help/facebook` marked "(sắp có)" rather than inventing a video URL (OWNER DECISIONS 2026-09-10 item B).

**Browser + profile handling (OWNER DECISIONS 2026-09-10 afternoon).** Before the button fires, the agent may call `GET /api/ui/{client}/browsers` — every installed Chromium-based browser the bridge detected, each with its profiles (display name, signed-in email if any) read straight from that browser's own `Local State` file. The agent asks at most ONE question about the browser and at most ONE about the profile, ever, per client:

| Situation | What the agent does |
|---|---|
| Chrome installed but not running | Just open it — no question. |
| Only Chrome, or only Edge, installed | Use it silently — no question. |
| Both Chrome and Edge installed | Ask ONE question: "Facebook của anh/chị đang đăng nhập ở Chrome hay Edge?" |
| One supported browser (Chrome or Edge) plus one or more best-effort browsers installed | Pick the supported browser silently — no question. Mention the best-effort ones only if the human says their Facebook lives there. |
| Only Brave, Vivaldi, Opera, or Chromium installed (no Chrome/Edge) | Use it silently; say once that it "works but untested" — best-effort, not the audited Chrome/Edge path. |
| Two or more best-effort browsers installed, no Chrome/Edge | Ask the same one browser question, naming the ones actually installed. |
| Only Safari and/or Firefox installed — no Chromium-based browser at all | Not supported. Tell the human to install Chrome (it's free; using it just for Facebook is fine). Local Mac: offer `brew install --cask google-chrome` after one consent. Windows: offer `winget install Google.Chrome`. Anywhere else: open `https://www.google.com/chrome/` for them. |
| Chosen browser has 2+ profiles | Ask ONE question naming the profile display names / signed-in emails from `/browsers`, then `POST /api/ui/{client}/install-extension` with optional `{browser, profile_directory}` opens the extensions page in exactly that profile. |
| Chosen browser has exactly 1 profile | Pick it silently — no question. |
| Browser already open on the wrong profile | Same `/browsers` listing, same one profile question, then re-open the extensions page in the right profile the same way (`install-extension` with `{browser, profile_directory}`). |
| Windows / Linux host | Same two gestures; the folder reveal uses Explorer (Windows) or the file manager via `xdg-open` (Linux) instead of Finder; Windows may show one SmartScreen click — mention it only if it actually happens; the Developer-mode toggle sits at the same top-right spot on every Chromium browser's extensions page. |

Once resolved, the chosen browser and profile are remembered per client (`extension_registry.json` fields `browser`, `profile_directory` — `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`) so later reopens and the 90-second help loop below reuse them without asking again. **Future, not this task, pending:** once the extension reports `facebook_logged_in: true|false` at check-in, this step and the help loop below will be able to tell "connected and logged in" apart from "connected but this profile is not logged into Facebook" and "not connected" — reference that only as "when the extension reports `facebook_logged_in`."

**If check-in stalls — 90-second help loop.** If `extension_health` has not gone recent within 90 seconds of triggering the install action, do not just repeat the reminder. Diagnose in this order and say ONE short line naming the likely cause: (1) Developer mode is still off — the toggle top-right on the extensions page; (2) the folder was dropped into a different browser profile/window than the one logged into Facebook; (3) the browser was closed or the page was navigated away. Re-trigger the install action (`POST /api/ui/{client}/install-extension`, with the same remembered `{browser, profile_directory}`) so Finder and the extensions page are back in front, then wait again. Run this diagnose-and-retrigger cycle at most 3 rounds. If the human says they cannot find the Developer mode toggle, that is a managed (work) Chrome — say so plainly, ask for a personal computer or a personal Chrome profile, and only then move to the web-only escape below. Never offer the escape before this help has been given, and never in the first message.

**Facebook Login Reminder.** Immediately after the bridge answers `/status` (extension check-in may still be catching up), deliver this reminder inside its own `**[ACTION REQUIRED]**` block, in the human's language:

```text
Để tìm lead trên Facebook, hãy đăng nhập Facebook trên Chrome của máy này (tài khoản cá nhân của bạn là đủ). Nếu không kết nối Facebook, Solo Agency chỉ tìm lead trên web mở — thường ít hơn nhiều, vì Facebook là nguồn mạnh nhất của hệ thống (feed, người, nhóm public, nhóm riêng). Anh/chị xác nhận muốn chạy chế độ không Facebook chứ?
```

Immediately after that block (not part of the verbatim text audited above, and not itself
`**[ACTION REQUIRED]**`), add one priming sentence in the human's own words and language: a plain
fact that every lead this pass finds lands straight in the client's CRM, and Free keeps a first
batch of contacts fully open.

Record the answer as `facebook_lead_source: enabled|web_only|pending` in the Client Intelligence
Profile (see `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`, Client Intelligence Profile fields).
`enabled` means the extension checked in and Facebook login is confirmed. `pending` is the
in-between state — the human is still working on this step and has not decided either way; it is
never itself a decision. `web_only` is a human's explicit, acknowledged choice to run without
Facebook: it is never a default, and never the result of "để sau" or silence — only a clear
confirming phrase (for example "không dùng Facebook") in reply to the block above resolves it.
On that confirmation, record `facebook_lead_source: web_only`, `facebook_lead_source_updated_at`
(ISO-8601), and `facebook_web_only_reason` (the human's own words, verbatim or lightly paraphrased)
in the Client Intelligence Profile. This reminder fires exactly once per client setup, here, at step
4, right after profile inference and before public data sources, the automation task, and the
private data source checkpoint. It fires again later only inside a scheduled/automation run whose
extension health shows logged-out/stale (`playbooks/SCHEDULED_RUN_ENTRYPOINT.md` step 12D).

**While the human is still deciding, keep moving.** Setup may continue into items 5-6 (public data
sources/keyword bank, the automation task, and beyond) with `facebook_lead_source: pending` — do not
stall the rest of setup on this one answer. What `pending` blocks is narrower and specific: the
client's first-ever dispatched run. Per the Report Request Hard Stop below and the Setup-Complete
Closing Template's offer 1, that first dispatch is gated on `facebook_lead_source` being `enabled` or
`web_only` — NEVER while it is still `pending`. If setup is otherwise complete and item 4 is still
unresolved, hold the dispatch and say plainly which of the two resolutions (log in, or confirm
web-only) unblocks it. Once a client's first run has dispatched, a later run finding this field back
at `pending` (a stale/logged-out extension) does not block that run either — see
`playbooks/SCHEDULED_RUN_ENTRYPOINT.md` step 12D/12E.

The client's first dispatched run is also the first run of the Facebook Discovery Pass
(`playbooks/10_LEAD_COMPETITOR_DETECTION.md`, "Facebook Discovery Pass (step 11C of the daily run)")
whenever `facebook_lead_source` is `enabled` at dispatch time — that pass uses
the FIRST RUN budget (≤ 21 collector calls, `max_pages` ≤ 4, spread over ≥ 4 hours) whenever it turns
out to be this client's first-ever Facebook Discovery Pass (tracked as
`facebook_discovery_first_pass_done` in the Client Intelligence Profile, set `true` immediately after
that pass completes), never the smaller DAILY budget — this is about which pass is the client's
first, not which automation run number it is; a client that starts `web_only` and enables Facebook
three runs later still gets the FIRST RUN budget on that later run. If the answer was `web_only`,
the dispatched run skips the pass entirely, and every report/reply this run produces carries the
persistent web-only awareness line (`playbooks/06_AGENCY_REPORT_STANDARD.md`) instead of a bare
lower-lead-count note.

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
- `extensions/{client_slug}_extension/manifest.json`.
- `extensions/{client_slug}_extension/client_binding.json`.
- `daily-content-pipeline/collector/extension_registry.json`.
- `daily-content-pipeline/collector/collector_config.json`.
- `daily-content-pipeline/schedule.md`.
- `daily-content-pipeline/automation/automation_manifest.md`.
- `daily-content-pipeline/automation/scheduled_run_prompt.md`.
- `daily-content-pipeline/automation/update_state.json` and `update_log.md` when update/watch has been checked, configured, or applied.
- `daily-content-pipeline/automation/resync_log.md`.

If the native automation task prompt cannot be updated directly, mark `automation_prompt_update_pending` in the manifest and schedule, then give the human one concrete instruction to update the task prompt.

The bridge install, the two-gesture extension install, and the Facebook Login Reminder happened earlier, at step 4 ("Kết nối Facebook" above) — right after profile inference, well before this output list is finalized. Nothing about that install is repeated or re-triggered here; this section only lists the files Setup Flow must leave current.

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

Chỗ nào chưa rõ, bạn cứ hỏi tôi bất cứ lúc nào nhé — không cần đợi tới lúc trả lời.
Bạn muốn bắt đầu với cái nào? (Trả lời 1, 2, 3 — hoặc hỏi tôi bất kỳ điều gì khác.)
```

Hard gates on this template: offer 1 is ALWAYS the immediate first run of the just-created automation task (a scheduled future run is never a substitute); one offer is ALWAYS from the OTHER product side (content setup offers Outreach, Outreach setup offers content/video — the cross-introduce rule); the closing line is a QUESTION. A setup-complete message whose last line is not a question is a Stage-9 audit failure. The priming clause folded into offer 1 (leads land in the CRM; Free keeps a first batch of contacts open) is spoken in the human's own words and language like the rest of the block, never this fixed sentence verbatim, and never with an estimated contact count — the shape stays three offers and one closing question; it does not become a fourth offer or a second question. The Standing Invitation line shown above the closing question (`SOLO_AGENCY_PLAYBOOK.md`, "Standing Invitation") is its own short line, freshly worded each time, placed immediately BEFORE that closing question — never merged into it, never after it, and never counted as the one required closing question itself.

Do not ask whether to run the report now. Do not load `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` inside the setup chat. Do not perform public research, private data source collection (one exception: the step-7 discovery pass per item 7 of the Setup Flow Contract), report generation, idea matrix updates, Lead & Competitor Opportunities, draft generation, analytics scans, or notification delivery (one exception: the single step-8 WideCast confirmation ping that verifies the notification channel right after the human provides the API key) in Setup Flow.
