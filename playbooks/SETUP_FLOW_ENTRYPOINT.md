# Solo Agency Setup Flow Entrypoint

Use this file as the entrypoint for setup/configuration sessions.

Setup Flow is the control plane. It configures Solo Agency so automation tasks run correctly later. It does not run operational reports.

## Runtime Requirement

Before setup proceeds, verify or explain that Solo Agency needs Codex, Claude Desktop/Cowork, Hermes, OpenClaw, or a comparable desktop/local AI agent runtime with workspace file access, automation/scheduled tasks, and multiple parallel/sub-agent work streams. Do not present a plain web chat as the primary runtime. Web chat can review results, but it cannot reliably host the file state, Local Collector handoff, scheduled automation, and multi-agent work this setup configures.

## Process-start failure is not a remote runtime

A command that fails to START — `CreateProcess`, `ENOENT`, `No such file or directory`, `cannot run
program`, `spawn ... failed` — almost always means a working directory that does not exist or a
wrong shell path, not a runtime without a shell. Before drawing any conclusion from it:

1. Check whether the directory the command was told to run in exists.
2. If it does not, walk up to the nearest parent that DOES exist, create the missing directory from
   there, and use it as the working directory.
3. Check the shell path the runtime is using.
4. Retry the command once from that valid directory.

Only when the local filesystem itself is unreachable after those four checks does the runtime count
as remote (`playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md`, "Local vs remote runtime rule").
Never hand the human a clone, install or setup command merely because a process failed to start
once, never call the session remote on that evidence alone, and never report an install, a clone or
a scan that did not actually happen.

## Setup Flow Contract

**Stable setup numbering.** When an onboarding optimization removes a human question or interaction, make that existing numbered step automatic or merge the behavior inside it; keep the visible 1-10 roadmap and all later step numbers stable. An automatic step remains visible and is marked complete when its work finishes. Renumber only with the owner's explicit approval of a roadmap redesign.

## Setup-chat visual contract

The three setup visuals are informational only. They never create an `**[ACTION REQUIRED]**` block,
wait state, approval, or setup step, and never change the visible 1-10 roadmap.

Before every actual visual render, resolve the verified absolute `setup_root` for this installation and
check that the exact asset exists. Render the image using Markdown image syntax whose target is the
real absolute filesystem path, for example `![Alt text](/verified/absolute/setup-root/assets/name.png)`.
Never send a relative path, unresolved `{token}`, `file://` URI, placeholder, or a text link in place
of the image. If an asset is missing, do not block or delay setup and do not emit broken Markdown.
Append a `setup_chat_asset_missing` event with `asset_path` to
`daily-content-pipeline/automation/setup_chat_asset_events.jsonl` as soon as the setup state can be
written, then continue without that visual. A missing first-words asset is logged immediately after
the first message, because First Words must remain the first human-visible message.

Visual order and placement are fixed:

1. In First Words, render `setup_root/assets/group_cover.png` immediately after the plain
   "what setting me up puts on your computer" lines that follow Sam's introduction, and before the
   rename-and-pin instruction.
2. At the first-client initial intake only, render
   `setup_root/assets/agency-structure_light.png` immediately before asking for the first client's
   minimum information. Explain in one short sentence that this first client is the business whose
   pipeline Sam is setting up. Do not add a question or setup step.
3. At step 4, render `setup_root/assets/theloop_light.png` immediately before the existing Login
   Reminder `**[ACTION REQUIRED]**` block. Keep the extension tab/action foreground and unchanged.

0. **First words come first.** Before step 1 — before any load, ledger or question — send the Team Leader First Words from `SOLO_AGENCY_PLAYBOOK.md`: the introduction, the plain lines on what setting up puts on this computer, the inline group-cover visual, the rename-and-pin instruction and the closing "Shall I begin?" — one message — and wait for the human's yes. Then load `playbooks/TEAM_MODEL.md` and `playbooks/NEXT_JOB_CATALOGUE.md` with the other entry files. Create `daily-content-pipeline/automation/boss_orders.md` (the ledger header from `playbooks/TEAM_MODEL.md`) if it does not exist, and record every request or goal the human states during setup as a row before acting on it.
1. Load `SOLO_AGENCY_PLAYBOOK.md` and `playbooks/LOAD_LEDGER_PROTOCOL.md`. **Full-load discipline applies to every file below: each load needs a LOAD LEDGER (read to the last line; compare `playbooks/LOAD_MANIFEST.md` when present; ledger each named dependency). A truncated / "output too large" / partial read = NOT loaded — re-read in chunks before acting. No side-effect step without a PASS ledger for the stage(s) it needs.**
2. Load `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`, `playbooks/01_BASIC_PROFILE_PUBLIC_REPORT.md`, `playbooks/04_DAILY_SCHEDULE.md`, `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`, and `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md`.
3. Load `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md` before step 4 below — every setup now installs the bridge and this client's extension, not only when custom sources that need a login are already known (internal access mode, never spoken to the human). Load `playbooks/PRIVATE_SOURCE_GATE.md` and `playbooks/02_PRIVATE_SOURCE_SETUP.md` separately, later, before step 7 (Chạy lượt đầu).
4. **Kết nối Facebook, Instagram and X**, right after profile inference (the human-facing roadmap's items 1-3: business context, industry/audience inference, pain points/pillars). Install the Local Collector bridge and this client's Chrome extension (the same extension covers all three platforms), then deliver the Login Reminder (naming Facebook, Instagram, and X) and record `facebook_lead_source`, with `instagram_lead_source` and `x_lead_source` following automatically — the full step contract, including the local/remote install rule and the two-gesture extension install, lives below under "Kết nối Facebook, Instagram and X (step 4)".
5. Load `playbooks/11_UPDATE_AND_VERSION_WATCH.md` when the human asks for update/upgrade/sync latest, when setup repair suspects stale playbooks/code, or when configuring the `Solo Agency - GitHub Update Watch` maintenance task.
6. Create or update client setup, default sources, existing or voluntarily supplied custom-source state, source registry state, extension folders, collector config, schedule files, automation manifests, scheduled prompts, update-watch state, and resync logs. Discovered sources are never part of Setup Flow — they appear later, as runs find them (`playbooks/10_LEAD_COMPETITOR_DETECTION.md`, "Step 5"; `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`), added to monitoring automatically (pause any on the Sources page).
7. Do not run source scans of any kind, reports, first agency runs, production, rendering, publishing, analytics scans, or outreach directly inside the Setup Flow chat/session. Step 7 (Chạy lượt đầu) dispatches the first run as its own automation-task session the moment the Boss says yes — that dispatched session is where scanning, joined-places discovery, qualification, and the report actually happen, never this chat (`SOLO_AGENCY_PLAYBOOK.md`, "Dispatch it, do not hand over a button"). Step 5 is automatic and never waits for a custom-source answer.
8. If the human asks to run, create, generate, show, refresh, or update a report inside Setup Flow, this is a hard stop for operational work. The setup chat stays Setup Flow; the request does not become Automation Flow. Jump forward to step 7's one question right now ("Step 7 — Chạy lượt đầu" below) — verify/resync the client-specific automation task first if it does not exist yet — and handle the answer exactly as that section defines; only a runtime that genuinely cannot start its own tasks falls back to naming the task for the human to run.
9. If the human says only `update`, `upgrade`, `cập nhật`, `sync latest`, or `pull latest`, treat that as the Stage 11 Solo Agency update command, not as `update a report`.
10. Every client-specific automation task name must begin with the client name, for example `AvenNgo - Solo Agency Daily Run`.
11. Every per-client Chrome extension display name must begin with the client name, for example `AvenNgo - Solo Agency Collector`.
12. PDNA provider setup must be client-scoped: read/write the current client's `integrations/providers/` files and verify provider identity through the client's OpenAPI/API-key config before claiming production, distribution, notification, analytics, credits, or connected platforms are available. Default setup is WideCast API-key setup: ask only for the client's WideCast API key, then let the agent configure/verify/discover/resync the rest. Do not ask provider/scope/spend/publish/account-identity questions unless the human explicitly requests a non-default provider or specialist stack. Do not use a global MCP/native provider account as proof of this client's PDNA status.
13. After any approved config change or applied Solo Agency update, perform Automation Resync if a schedule/automation already exists.
14. Setup Flow completion means `ready_for_automation_first_run` or `ready_for_next_automation_run`. The completion message restates, in one line, that every future request goes to this chat with the Team Leader, and lists the scheduled tasks just created as team members with their roles (`playbooks/TEAM_MODEL.md`, Roster).
15. Every human question, approval request, one-line Terminal/PowerShell command, Chrome `Load unpacked` instruction, provider/API-key setup request, and native automation task edit must use the `**[ACTION REQUIRED]**` block from `SOLO_AGENCY_PLAYBOOK.md`. If setup continues without needing the human, end the reply with next-action guidance per the root Next-Action Guidance Rule instead of `No action required right now.`
16. **Standing Invitation.** Every Setup Flow message that asks the human for anything — a question, a choice, a consent, an `**[ACTION REQUIRED]**` gesture such as the two-gesture extension install, a paste — closes with one short line, in the human's own language, inviting them to ask about anything unclear before they answer (`SOLO_AGENCY_PLAYBOOK.md`, "Standing Invitation"): fresh wording each time, never dropped, never itself the closing question, and never a substitute for the required next-step question or the `**[ACTION REQUIRED]**` block. Where the message must also end with exactly one question, the invitation line comes before that question, not after it.
17. On a **local Claude Code runtime**, right after step 6 creates the client-specific automation task, the agent asks the one unattended-permissions consent (`SOLO_AGENCY_PLAYBOOK.md`, item 6 of the Mandatory Setup Flow; exact rules in `playbooks/04_DAILY_SCHEDULE.md`) and, on yes, writes the user-level allow rules — step 7's one question is what actually dispatches the first run, not this step — recording `unattended_permissions: granted`; on no, it does NOT write the rules — it records `unattended_permissions: declined` and gives the one-sentence pause explanation from `playbooks/04_DAILY_SCHEDULE.md` ("the first run will pause once in its own session for an Always allow click"), and continues. On any other runtime, the ask is skipped entirely and it records `unattended_permissions: not_applicable`. `declined` and `not_applicable` are never the same outcome. Either way, also record `unattended_permissions_scope` and `unattended_permissions_written_at` (the latter only when `granted`) in the automation manifest.
18. Roadmap step 2 (profile inference) infers and shows `buyer_profile` alongside `industry`/`sub_industry`/`target_audience`, and gets the Boss's one-sentence confirmation before it is treated as stable — full field contract in `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`, "`buyer_profile` (required, inferred at setup step 2)".
19. Roadmap step 5 (Sources) automatically configures default sources and builds the keyword bank per channel FROM `buyer_profile.types` — the channel table, generation procedure, and quality gate in `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`, "Buyer Profile Channel Keyword Table" — not from intent phrases alone. It asks no custom-source question, creates no required action, and cannot delay steps 6-7. Preserve a URL the Boss volunteers and process it when the collector is ready; later additions use Revenue Engine `watch_source` or a direct request to Sam and require Automation Resync.
20. Roadmap step 6 automatically creates the first client Daily Run when no explicit saved or Boss-provided schedule exists: daily at 09:00 local, subject to `schedule-slots suggest`, then create at the returned actual time and `register`. Report that actual start time and the existing finish-window estimate, and say the Boss may ask Sam to change cadence or time later. An explicit schedule wins; repair/update preserves an existing schedule. No schedule question or `**[ACTION REQUIRED]**` block is used for this reversible default. If the bridge/schedule-slots tool is genuinely unavailable, create at 09:00 local (or the explicit override), set `slot_check_pending`, and register once available. Step 7 remains the separate run-now permission gate.

## Kết nối Facebook, Instagram and X (step 4)

This step runs immediately after profile inference (roadmap items 1-3) and before Sources/keyword bank (roadmap item 5). It replaces the old positioning where the bridge/extension install and the Login Reminder sat inside "Required Setup Output" right before dispatch — they now happen here, early, so `facebook_lead_source` (with `instagram_lead_source` and `x_lead_source` alongside it) is known well before the automation task and step 7 (Chạy lượt đầu).

Open with one plain-language value sentence in the human's language, for example:

```text
Bước này để em quét được Facebook, Instagram và X: feed, người, bài viết và bình luận public — đó là nơi phần lớn lead đầu tiên sẽ đến từ.
```

Dashboard text is English; you translate for the Boss in chat — every `/ui/...` page the human sees
in this step (extension, help) renders in English regardless of the human's language.

**Bridge install — agent-run on a local runtime, handed off on a remote runtime.** Run the Stage 8 Source Safety Pre-Check first. Then:

- **Local runtime** (Codex CLI, Claude Code desktop/CLI, Hermes, OpenClaw, or a comparable runtime that can see the install root on its own filesystem): the agent installs and starts the bridge itself. It writes `setup_collector.sh` (`setup_collector.ps1` on Windows) with the absolute install path, says the Source Safety Pre-Check result in a few plain-language lines — the extension's code was read, the bridge was checksum-verified rather than read, and where its outside calls go — for example `Em đã đọc mã của tiện ích mở rộng: mọi thứ nó thu thập chỉ đi tới chương trình nhỏ trên chính máy này. Chương trình đó là bản dựng sẵn tải từ GitHub, em kiểm tra checksum thay vì đọc mã; ngoài máy này nó chỉ gọi tới tài khoản WideCast của anh/chị để báo tin và kiểm tra gói. Em cài và bật nó ngay bây giờ, khoảng một phút.` — asks for consent once, runs the script itself, and waits for `GET http://127.0.0.1:17321/status` to answer (up to 60 seconds). The script hands the process to an OS-level autostart service (macOS LaunchAgent, Linux systemd user unit, or a Windows logon Scheduled Task, per `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md`) — the agent's own shell never owns that process, so this is not the in-sandbox launch the old ban addressed. On Windows, SmartScreen may ask for one click; mention it only if it actually happens.
- **Remote runtime** (a hosted sandbox where `127.0.0.1` is not the human's machine): detected when the agent cannot see the install root on its own filesystem, or `/status` still fails 60 seconds after a bootstrap attempt. Only then does the agent hand the human the one exact one-line command (`bash "/ABSOLUTE/PATH/TO/solo-agency-local-collector/setup_collector.sh"` or the Windows equivalent) instead of running it itself.
- Do not show the install command/path (local or remote) before the pre-check passes; if it fails, stop and raise it to the operator instead.

**Dashboard and extension tab order.** Immediately after `GET /status` succeeds and its config/output/run-now paths prove this is the current workspace, automatically print and show `http://127.0.0.1:17321/ui/{client_slug}#revenue-engine` under the SHOW RULE. Prefer the Codex built-in browser or Claude side Browser; this localhost dashboard use is read-only and never an `**[ACTION REQUIRED]**` block. Then show `/ui/{client_slug}/extension` as the active foreground tab for the extension install. Never cover, navigate away from, or foreground Revenue Engine while waiting for extension install/check-in: with multiple tabs it stays behind the foreground Extension tab; with one tab show Revenue Engine first, then Extension and remain there. Only after `extension_health.status: recent` bring Revenue Engine to the foreground again. Built-in browsers are used here only for localhost dashboard pages, never to browse social or other private sources.

**Extension install — two gestures.** The extension is one install that covers all three platforms — Facebook, Instagram, and X — not a separate install per platform. That client's `/ui/{client}/extension` dashboard page shows, by default, only the video, the two install steps, and the extension folder path — the browser/account pickers and the "Install extension" button live behind `?advanced=1` and are never mentioned to the human. The agent (local runtime) calls `POST /api/ui/{client_slug}/install-extension` itself, with no `browser`/`profile_directory` fields, to reveal the `extensions/{client_slug}_extension/` folder (Finder/Explorer, or the OS equivalent), then polls `/status` until `extension_health` is recent (75-second grace) and celebrates the connection in chat. The page's two steps: turn on Developer mode once, and drag the folder onto the page — "Kéo thư mục vào cửa sổ trình duyệt anh/chị vẫn dùng để lên Facebook" ("Load it into the browser window you normally use for Facebook"). Right under those two steps the page also plays a short recorded video — point the human at it directly on the page rather than only describing the steps in words; see "Show the install video" immediately below for the other ways to hand it over when the page itself is not reachable. Alongside the reveal, the agent says this one fixed sentence, in the human's language, so the drag lands on the right folder: "Chọn đúng thư mục tên `{client}_extension` mà em vừa mở — không chọn thư mục `chrome-extension` nằm trong mã nguồn." (OWNER DECISIONS 2026-09-10 item A).

**Show the install video.** In order of availability:

(a) **Bridge answering `/status`** — open `http://127.0.0.1:17321/ui/{client_slug}/extension` in the side browser (Claude desktop Browser pane / Codex built-in browser) so the video plays right above the two steps, and print that link. The same bridge also serves the recordings directly at `http://127.0.0.1:17321/ui/assets/help/setup_extension_chrome_small.mp4` (Edge: `…_edge_small.mp4`) — they are embedded in the binary, so this URL works no matter how old the human's clone is, and it is the most reliable local source, not a fallback.
(b) **The local file, when it is actually there** — an install that has not pulled recently predates the commit that added the recordings, so check the file exists before naming it (verified 2026-09-12: a 2026-09-07 clone carries no mp4 at all). When it is present: a runtime that can show a file in chat sends the mp4 (Claude Code desktop renders it inline); otherwise open it with the OS default player — macOS `open "{setup-root}/solo-agency/solo-agency-collector/bridge-go/assets/setup_extension_chrome_small.mp4"`, Windows `start "" "…\setup_extension_chrome_small.mp4"`, Linux `xdg-open …` — and/or navigate the side Browser pane to its `file://` URL. Print the absolute local path, never a web URL.
(c) **Remote runtime only** (the agent's shell is not the human's machine, so there is no local file to open): the public links `https://github.com/soloagency/solo-agency/blob/main/solo-agency-collector/bridge-go/assets/setup_extension_chrome_small.mp4` (Chrome) and `https://github.com/soloagency/solo-agency/blob/main/solo-agency-collector/bridge-go/assets/setup_extension_edge_small.mp4` (Edge) — GitHub renders a player on those pages. Never print these on a local runtime — the file is already on disk.

Default to the Chrome recording and name the Edge one in the same line; never ask which one they want.

Order inside step 4: this extension-install guidance starts only after the bridge answers `/status` (local runtime: the agent ran `setup_collector` itself and polled up to 60 seconds). If the poll fails on a local runtime, the agent still shows the video by (b) together with the one-line bridge start command; on a REMOTE runtime where the human runs that command themselves, it shows the video by (c) instead. Nothing about a browser or a profile is ever asked.

**Browser: never asked, never named.** The human installs the extension in whatever browser they already use — Chrome, Edge, Brave, anything Chromium-based — and the video shows the same two gestures on all of them. The agent never names the browsers it detected, never asks a Chrome-vs-Edge question, never asks about profiles or accounts, and records nothing about either. It calls `POST /api/ui/{client_slug}/install-extension` with no `browser`/`profile_directory` fields (the bridge reveals the extension folder and best-effort opens a Chromium extensions page), and the instruction it speaks is browser-agnostic: open your browser's extensions page (`chrome://extensions`, or `edge://extensions` on Edge), turn on Developer mode, drag the folder in. The single exception, and it is not a question: when the machine has NO Chromium-based browser at all (only Safari and/or Firefox), say Chrome is needed and offer to install it — `brew install --cask google-chrome` on a local Mac after one consent, `winget install Google.Chrome` on Windows, otherwise open `https://www.google.com/chrome/`.

**Future, not this task, pending:** once the extension reports `facebook_logged_in: true|false` at check-in, this step and the help loop below will be able to tell "connected and logged in" apart from "connected but this profile is not logged into Facebook" and "not connected" — reference that only as "when the extension reports `facebook_logged_in`."

This step's `**[ACTION REQUIRED]**` block has one goal only — the extension installed and checking in — and asks nothing else:

```text
**[ACTION REQUIRED]** — cài extension cho {Client Name} (khoảng 1 phút)

**Video 30 giây:** {the local mp4 the agent just opened/sent, or the dashboard page link}
**Thư mục extension:** `{absolute path}/extensions/{client_slug}_extension` — em vừa mở sẵn trong Finder;
đúng thư mục tên `{client_slug}_extension`, không phải thư mục `chrome-extension` trong mã nguồn.

1. Mở trang tiện ích của trình duyệt anh vẫn dùng: gõ `chrome://extensions` (Edge: `edge://extensions`).
2. Bật **Developer mode** (góc trên bên phải), rồi kéo thư mục trên thả vào trang đó.

Xong là ô extension hiện xanh ✓ và em thấy ngay — anh nhắn "done" hoặc cứ để em tự nhận.
```

**If check-in stalls — 90-second help loop.** If `extension_health` has not gone recent within 90 seconds of triggering the install action, do not just repeat the reminder. Diagnose in this order and say ONE short line naming the likely cause: (1) Developer mode is still off — the toggle top-right on the extensions page; (2) the folder was dropped into a window that is not the one signed into Facebook; (3) the browser was closed or the page was navigated away. Re-trigger the install action (`POST /api/ui/{client}/install-extension`) so Finder and the extensions page are back in front, then wait again. Run this diagnose-and-retrigger cycle at most 3 rounds. If the human says they cannot find the Developer mode toggle, that is a managed (work) browser — say so plainly, suggest trying from a personal (non-work) computer instead, and only then move to the web-only escape below. Never offer the escape before this help has been given, and never in the first message.

**Login Reminder.** Immediately after the bridge answers `/status` (extension check-in may still be catching up), and immediately before the existing Login Reminder `**[ACTION REQUIRED]**` block, render the verified inline `setup_root/assets/theloop_light.png` visual under the setup-chat visual contract. Do not hide, delay, replace, or navigate away from the extension tab/action. Then deliver this reminder inside its own `**[ACTION REQUIRED]**` block, in the human's language, naming all three platforms:

```text
Để tìm lead trên Facebook, Instagram và X, hãy đăng nhập Facebook, Instagram và X trên Chrome của máy này (tài khoản cá nhân của bạn là đủ, dùng chung cho cả ba). Nếu bỏ qua một nền tảng nào đó, Solo Agency chỉ tìm lead trên web mở cho nền tảng đó — thường ít hơn nhiều, vì ba nền tảng này là nguồn mạnh nhất của hệ thống (feed, người, nhóm public, nhóm riêng trên Facebook; bài viết và người dùng trên Instagram và X). Anh/chị xác nhận có nền tảng nào muốn chạy chế độ không kết nối không?
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
in the Client Intelligence Profile. `instagram_lead_source` and `x_lead_source` are set the same
way automatically, from the extension's own per-platform login detection — no separate question is
asked for Instagram or X; Facebook, Instagram, and X are all default sources, enabled the moment the
extension checks in and confirms that platform's login. This reminder fires exactly once per client setup, here, at step
4, right after profile inference and before Sources, the automation task, and
step 7 (Chạy lượt đầu). It fires again later only inside a scheduled/automation run whose
extension health shows logged-out/stale (`playbooks/SCHEDULED_RUN_ENTRYPOINT.md` step 12D).

**While the human is still deciding, keep moving.** Setup may continue into items 5-6 (Sources and
keyword bank, the automation task, and beyond) with `facebook_lead_source: pending` — do not
stall the rest of setup on this one answer. What `pending` blocks is narrower and specific: the
client's first-ever dispatched run. Per step 7 (Chạy lượt đầu), that first dispatch is gated on `facebook_lead_source` being `enabled` or
`web_only` — NEVER while it is still `pending`. If setup is otherwise complete and item 4 is still
unresolved, hold the dispatch and say plainly which of the two resolutions (log in, or confirm
web-only) unblocks it. Once a client's first run has dispatched, a later run finding this field back
at `pending` (a stale/logged-out extension) does not block that run either — see
`playbooks/SCHEDULED_RUN_ENTRYPOINT.md` step 12D/12E.

The client's first dispatched run is also the first run of the Social Discovery Pass
(`playbooks/10_LEAD_COMPETITOR_DETECTION.md`, "Social Discovery Pass (step 11C of the daily run)")
whenever `facebook_lead_source` is `enabled` at dispatch time — that pass uses
the FIRST RUN budget per platform (Facebook ≤ 21 collector calls / `max_pages` ≤ 4; Instagram ≤ 12;
X ≤ 12), paced by the collector (Pacing Rule, `playbooks/10_LEAD_COMPETITOR_DETECTION.md` — a
random 5–10 second delay before each request, never an hours-long gap), whenever it turns
out to be this client's first-ever Social Discovery Pass (tracked as
`facebook_discovery_first_pass_done` in the Client Intelligence Profile, set `true` immediately after
that pass completes), never the smaller DAILY budget — this is about which pass is the client's
first, not which automation run number it is; a client that starts `web_only` and enables Facebook
three runs later still gets the FIRST RUN budget on that later run. Each platform's step of the pass
runs only while that platform's `{platform}_lead_source` is `enabled`; a platform still `web_only`
or `pending` simply loses its turn (Round-Robin Rule, `playbooks/10_LEAD_COMPETITOR_DETECTION.md`),
and every report/reply this run produces carries the persistent web-only awareness line
(`playbooks/06_AGENCY_REPORT_STANDARD.md`) naming whichever platforms are still off, instead of a
bare lower-lead-count note.

## Step 7 — Chạy lượt đầu (First run)

After step 6 creates the client-specific automation task and its preconditions are met (`facebook_lead_source`, with `instagram_lead_source`/`x_lead_source` alongside it, is `enabled` or `web_only`, never `pending`; on a local Claude Code runtime the step-6 unattended-permissions consent was asked), ask the Boss exactly ONE question — the only source-review-shaped checkpoint left anywhere in setup — translated into the human's language, keeping the shape (Vietnamese sample):

```text
Em chạy lượt quét đầu tiên ngay bây giờ nhé? Em sẽ quét lead trên Facebook/Instagram/X (kênh anh đã kết nối), Google và web chuyên ngành, tự chọn những group có khả năng có lead để theo dõi, rồi báo kết quả ngay tại đây. Chỉ đọc, không đăng, không join gì. (Trả lời "ok" hoặc "để sau".)
```

No source-approval question comes before or after it, and nothing about groups or discovered sources is ever put to a vote.

**On yes.** Record `first_run_consent: yes`. Verify/resync the client contract; inspect `first_run_task_id/status` and wait if already running or safely replace one stale/failed task. Create/start the separate one-time `{Client} - Solo Agency First Run` with the latest pinned scheduled-run prompt and explicit identity — Claude `fireAt` now+2 min; Codex native run-now on THAT task — while recurring Daily Run remains untouched. Record lifecycle, run `tool run-progress eta --calls <calls_planned>`, speak the three dispatch sentences (account safety; progress/finish; PDNA Notification/Telegram), arm `tools/wait_for_run ... --watch progress`, then delete/mark deleted after report. Joined-places discovery rides the same yes: record `approved_pending_first_scan` for all categories, execute it, and apply Group Potential outcomes directly — high/medium monitored, low `not_selected` with reason — with no shortlist/second approval. Never ask Codex/Claude Bosses to click Run now/open a scheduler. Preconditions: Daily task exists; Facebook/Instagram/X enabled or web-only; local Claude unattended-permissions consent asked.

**On "để sau" / not now.** Record `first_run_consent: not_now`. The daily task keeps its existing schedule; mark roadmap step 7 `–` with the reason "runs at {HH:MM} on the schedule"; ask nothing else and continue setup at step 8. If the Boss said "để sau", step 8's Notification ask drops the dispatch-message framing and is asked directly (value-first, the same WideCast-key flow), since there is no run in flight to reference; it is still asked exactly once.

**The step-8 bridge sentence.** The third dispatch sentence above IS step 8's Notification ask — it is never asked twice. Vietnamese shape: "Trong lúc lượt đầu chạy (dự kiến xong {HH:MM}–{HH:MM}), mình cài WideCast luôn để anh nhận thông báo khi xong…" Step 8 (`SOLO_AGENCY_PLAYBOOK.md`, item 8) then follows through on whatever the Boss answers — the WideCast API key flow — while the run itself keeps executing in the background.

**Running-status line.** While `first_run_wait` is `armed` (the run is in flight), every Boss-facing reply — including the rest of setup, steps 8-10 — OPENS with one line, read fresh from `tool run-progress show --client <slug>` in the same turn (Read-Before-Claim Rule):

```text
Lượt đầu đang chạy: giai đoạn {k}/6 — {stage}, đã dùng {calls_done}/{calls_planned} lượt gọi, dự kiến xong {HH:MM}–{HH:MM}.
```

Before the first progress line exists yet: "Lượt đầu đang chạy từ {HH:MM} (chưa có mốc đầu tiên), dự kiến xong {HH:MM}–{HH:MM}." This line disappears the moment the First-Run Report has been spoken.

**Wake interleaving.** A `progress` wake firing while Sam is on steps 8-10 is spoken first, as the stage update (Run Progress Rule shape), and Sam then continues whatever step it was on, in the same message. A `standup` wake is spoken first as the First-Run Report, and Sam then continues the same way.

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

Do not include private client data, secrets, cookies, tokens, raw captures from sources that needed a login, or logged-in screenshots in GitHub issues. If direct issue creation/sending is unavailable, write the draft under `daily-content-pipeline/automation/issues/`, track it in `daily-content-pipeline/automation/github_issues.md`, and tell the human the path.

## Required Setup Output

For each configured client, Setup Flow must leave these current:

- Client Intelligence Profile, including the confirmed `buyer_profile` block.
- automatically configured default sources and the keyword bank per channel (in-group, people search, IG/X search, community discovery, Google/open web) generated from `buyer_profile.types` — `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`, "Buyer Profile Channel Keyword Table"; preserve any custom sources voluntarily supplied during setup without requiring them.
- source registry state (monitored groups, potential, pause/resume).
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

The bridge install, the two-gesture extension install, and the Login Reminder happened earlier, at step 4 ("Kết nối Facebook, Instagram and X" above) — right after profile inference, well before this output list is finalized. Nothing about that install is repeated or re-triggered here; this section only lists the files Setup Flow must leave current.

After schedule/automation exists, set up the separate maintenance task `Solo Agency - GitHub Update Watch` - do not silently skip it. Create the native task if the runtime allows; otherwise write the exact prompt to `daily-content-pipeline/automation/update_watch_prompt.md`, record `update_watch_task_prompt_pending`, AND end with an `**[ACTION REQUIRED]**` block naming the task and the exact way to create it. Default posture is notify-first (`auto_apply_approved: false`).

## Report Request Hard Stop

When the human asks for a report/run while this entrypoint is active, the only valid response is:

1. State that Setup Flow does not run reports directly, then jump forward to step 7's question right now ("Step 7 — Chạy lượt đầu" above), out of the normal step order if needed.
2. If the client-specific automation task does not exist yet, finish or resync it first (step 6).
3. Ask step 7's one question and handle the answer exactly as that section defines: on yes, dispatch and speak the three dispatch sentences (account-safety timing, progress-then-finish messaging, the Telegram/WideCast Notification line) plus arm the background wait, then the Running-status line carries every reply from here on while the run is in flight; on "để sau" / not now, say so plainly and give the exact client-specific automation task name the human can run themselves. If the native automation UI requires human action to actually start, provide that one exact action in a `**[ACTION REQUIRED]**` block instead.
4. Either way, continue setup at the step it was actually on before the report request interrupted it. When no required human action remains, include a feature-discovery block introducing 2-3 unused headline capabilities from `playbooks/FEATURE_CATALOG.md` (Feature Discovery Rule). When an `**[ACTION REQUIRED]**` block is needed, omit and defer feature discovery plus the Revenue Engine anchor so the required block is the exclusive close — this interruption does not end setup.

### Setup-Complete Closing Template (MANDATORY shape)

On the normal no-required-action branch, the FINAL message of a completed setup must open with the first run's CURRENT state — never a second dispatch; the three dispatch sentences were already spoken once, at step 7 — then use this Revenue Engine anchor plus three-offer block (translate to the human's language; adapt names, never the shape). Listing trigger phrases "for later" is NOT compliance: the message MUST end with exactly one question, and NONE of the three offers may be "run the first report" — it was already asked and dispatched at step 7. A genuine dispatch failure does not use this template; it uses the separate required-action branch below and ends at the `**[ACTION REQUIRED]**` block.

```text
[Still running:]
Lượt đầu đang chạy: giai đoạn {k}/6 — {stage}, đã dùng {calls_done}/{calls_planned} lượt gọi, dự
kiến xong {HH:MM}–{HH:MM}. Tôi sẽ nhắn ngay khi xong. Lead tìm được sẽ vào thẳng CRM chung, và gói
Free đang mở sẵn một số lượng contact đầu tiên miễn phí.

[Finished:]
Lượt đầu xong rồi: {the First-Run Report's leads line — hot/warm/watch}. Xem báo cáo đầy đủ tại
{report link}. Lead tìm được đã vào thẳng CRM chung, và gói Free đang mở sẵn một số lượng contact
đầu tiên miễn phí.

Revenue Engine — Tìm lead chủ động · Thu hút lead tự tìm đến · Nuôi dưỡng và chuyển đổi · Xem 25 tính năng: http://127.0.0.1:17321/ui/{client_slug}

**Trong lúc chờ, vài việc khác bạn có thể bắt đầu ngay:**
1. **Tạo campaign cold-email đầu tiên (OutreachCRM)** — chỉ cần 3 thứ: list lead,
   Gmail App Password, và xác nhận goal+URL. Nói: "set up a cold-email campaign".
2. **{one unused feature from FEATURE_CATALOG.md, value-first, with its trigger phrase}**
3. **{another unused feature from FEATURE_CATALOG.md, value-first, with its trigger phrase}**

Chỗ nào chưa rõ, bạn cứ hỏi tôi bất cứ lúc nào nhé — không cần đợi tới lúc trả lời.
Bạn muốn bắt đầu với cái nào? (Trả lời 1, 2, 3 — hoặc hỏi tôi bất kỳ điều gì khác.)
```

The two bracketed variants in the first block are chosen by state, never both spoken and never printed with their brackets: read `tool run-progress show --client <slug>` fresh in this same turn (Read-Before-Claim Rule) — "still running" (the Running-status line, "Step 7 — Chạy lượt đầu" above) while `first_run_wait` is `armed`, "finished" (the First-Run Report's leads line and report link) once the standup wake has fired and the report has actually been spoken. If notification is still not connected by this point, offer 2 may be the Notifications feature ("bật thông báo" / "turn on notifications") instead of a FEATURE_CATALOG pick — that only nudges the Boss to finish an answer already asked once, as the third dispatch sentence at step 7; it is never a repeat of the ask.

Only when dispatch was genuinely impossible does the reply switch to the required-action branch: state the real reason, end with one `**[ACTION REQUIRED]**` block naming the exact task/action, and omit the three offers, Revenue Engine anchor, feature-discovery block, and next-jobs question. After the Boss resolves that action, the next no-required-action reply uses the normal three-offer closing template.

Hard gates on this template: the first line ALWAYS states the first run's CURRENT state — still running (Running-status line + "tôi sẽ nhắn ngay khi xong") or finished (the First-Run Report's leads line + report link). The normal no-required-action branch then keeps the shape one status line, Revenue Engine anchor, three offers, the Standing Invitation, and one closing question; one offer is ALWAYS from the OTHER product side (content setup offers Outreach, Outreach setup offers content/video), no offer tells the Boss to open another chat or run the task themselves, and nothing follows the closing question. On real dispatch failure, use the separate required-action branch from the paragraph above instead — the `**[ACTION REQUIRED]**` block is last and none of those optional offers/anchors/questions appears. The three dispatch sentences (account-safety timing, progress-then-finish messaging, the Telegram/Notification line) are NEVER repeated here. The priming clause folded into the status line (leads land in the CRM; Free keeps a first batch of contacts open) is spoken in the human's own words and language, never this fixed sentence verbatim, and never with an estimated contact count. The Standing Invitation line is freshly worded each time and, on the normal branch, sits immediately before the closing question.

Do not ask whether to run the report now anywhere outside step 7's one question. Do not load `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` inside the setup chat. Do not perform source research or source collection of any kind, report generation, idea matrix updates, Lead & Competitor Opportunities, draft generation, or analytics scans directly inside the Setup Flow chat — the automation task dispatched at step 7 does all of that in its own session. One exception stays for notification delivery: the single step-8 WideCast confirmation ping that verifies the notification channel right after the human provides the API key.
