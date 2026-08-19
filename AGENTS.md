# Agent Instructions

When the user asks to set up this repo, always read `SOLO_AGENCY_PLAYBOOK.md` first and follow its checklist in order.

Human-facing required actions must use the Solo Agency `**[ACTION REQUIRED]**` block from `SOLO_AGENCY_PLAYBOOK.md`. Do not bury questions, approvals, commands, Chrome extension paths, provider/API-key setup, or automation task edits in paragraphs or reports. If no human action is needed, end with next-action guidance per the root playbook's Next-Action Guidance Rule (1-3 real available next steps plus one closing question); never end with `No action required right now.`

During Setup Flow, never run, create, generate, show, refresh, or update a report in the setup chat, even if the human explicitly asks. Treat the request as a handoff request: verify/resync the client-specific automation task, tell the human the exact task name to run, and do not load the scheduled-run entrypoint or perform public research/private collection/report generation inside Setup Flow.

Do not install, start, or configure `solo-agency-collector/` before the playbook explicitly reaches the Local Collector/private data source stage and the human approves it. This is a SETUP-FLOW gate only: on a live install, running the already-installed bridge binary and its tools (`tool crm-store`, `tool gmail`, `tool migrate`...) IS normal operation for the operating agent — this line never forbids it, and citing it to avoid running a tool on a live system is a self-invented gate.

Solo Agency must run in an AI agent runtime that supports local workspace files, automation/scheduled tasks, and multiple parallel/sub-agent work streams, such as Codex, Claude Desktop/Cowork, Hermes, OpenClaw, or a comparable desktop/local agent environment. Do not present a plain web chat as the primary runtime for Solo Agency; web chat can review outputs, but it cannot reliably host the automation, file state, Local Collector handoff, and parallel agent work the playbook requires.

For setup, repair, update, or Local Collector preparation, treat the human as a fresh user unless the current setup root is proven otherwise. Do not reuse fixed shared fallback folders such as `/tmp/solo-agency`, `/var/tmp/solo-agency`, or `/dev/shm/solo-agency`. Download or clone from `https://github.com/soloagency/solo-agency` into the current setup root or into a fresh unique `mktemp -d` directory, then verify the source before reading or copying: `.git` must exist, `origin` must be the Solo Agency GitHub repo, and `git rev-parse HEAD` must match `git ls-remote origin refs/heads/main` after fetch/clone. A folder without `.git`, with the wrong owner, or with a failed delete/update is stale cache, not a valid source. If network or sandbox access blocks a fresh GitHub fetch, request permission or hand the human one exact command; do not fall back to unverified local code.

If the human says `update`, `upgrade`, `cập nhật`, `sync latest`, `pull latest`, or any equivalent short update command, load `playbooks/11_UPDATE_AND_VERSION_WATCH.md` and treat the request as a Solo Agency update command: check the latest GitHub `main`, compare playbooks plus collector/bridge/extension/provider/template code, apply safe updates from a verified fresh checkout, resync every client and automation/scheduled task, and update `daily-content-pipeline/automation/update_state.json` plus `update_log.md`. This is not a request to update a report. Do not run reports, scans, production, publishing, analytics, or outreach because of an update command.

Whenever the agent hits a blocker, unexpected behavior, repeated failure, unclear instruction conflict, or dead end in any Solo Agency activity, treat stale local playbooks/code as the first suspect. Before declaring the task blocked, fetch/clone the latest `https://github.com/soloagency/solo-agency` `main` into a fresh verified checkout or read the relevant raw GitHub playbooks, compare the latest commit to the local one, reload any changed instructions, and retry using the newest rule. If the latest GitHub version still does not resolve the issue, escalate without requiring the human to register or use a GitHub account: create a redacted GitHub issue in `soloagency/solo-agency` only when the agent/runtime has an authorized GitHub identity such as authenticated `gh`, `GITHUB_TOKEN`, `GH_TOKEN`, `SOLO_AGENCY_GITHUB_ISSUE_TOKEN`, or a GitHub App/maintainer bot. If no authorized GitHub identity is available, send/queue the redacted issue through a configured support/intake channel when one exists; otherwise write a ready-to-post issue draft under `daily-content-pipeline/automation/issues/`. Record the issue URL, intake channel, or draft path in `daily-content-pipeline/automation/github_issues.md`, mention it in the human-facing blocker, and check the tracked issue in later runs until a fix/answer is available. Never include secrets, private data source content, cookies, tokens, client-confidential data, or raw logged-in screenshots in an issue.

After setup/routine exists, offer or maintain the daily `Solo Agency - GitHub Update Watch` automation task described in `playbooks/11_UPDATE_AND_VERSION_WATCH.md`. It checks GitHub for new Solo Agency versions, classifies changes, writes an internal/local update notice, and applies/resyncs updates only when the human has approved auto-apply. Update-watch must not send Telegram, WideCast/email-fallback, provider notifications, social posts, or client notifications because version maintenance is internal user/agency work. If bridge Go/runtime or extension files changed, the update handoff must include the current setup's exact bridge rerun command and Chrome extension reload/Load unpacked steps for every client profile.


## Brain swap and multi-brain operation — a live install on the same machine

The filesystem is the system; the agent is only the brain. The data root
(`daily-content-pipeline/`), the bridge, the Chrome extension, the playbooks and every credential
are brain-agnostic: when the human switches agent (Codex ran out of quota and opens Claude, or the
reverse), NOTHING is exported, imported, re-set-up, re-cloned or re-authenticated. Recognize this
case by what is on disk: a `daily-content-pipeline/` with clients and a collector config — usually
with a bridge already answering on `127.0.0.1:17321` — means you are the SECOND brain on a LIVE
system, even if this chat says "set up" or "get started".

Two brains may also drive this install AT THE SAME TIME — the human keeps Codex and Claude open
together and hands each of them different work. Taking over and working alongside are the same
procedure; only step 4 differs. Take over in this order, and do not skip step 2:

0. **Probe what THIS surface can do, before promising anything.** Run the installed bridge
   binary once (`<bridge> tool crm-store --help`). If the runtime cannot execute local binaries
   or reach the machine's localhost (a cloud/Cowork session attached to the folder through a
   file bridge cannot), then per the runtime rule above this surface is REVIEW-ONLY: say exactly
   that — "this surface can read and edit files but cannot operate the system; open the agent's
   local CLI/desktop runtime on this machine to run it" — and stop there. Do not blend real
   runtime limits with playbook citations; the playbooks never forbid the operating agent from
   running the installed tools (see the setup-gate note above). Operating includes sending:
   Stage 8 is `tool crm-store ingest-ui` (apply the operator's Approvals-page decisions), then
   `tool gmail send` per the send playbook — the operating brain runs these itself; the human
   never pastes commands into a terminal.

1. **Update first.** Run the standard update flow (`playbooks/11_UPDATE_AND_VERSION_WATCH.md`):
   verified fresh checkout against GitHub `main`, refresh playbooks and the collector per
   `solo-agency-collector/setup_collector.sh`. A stale binary silently corrupts config it touches
   (an outdated whitelist drops fields it does not know — this has happened), so no operation
   precedes the update.
2. **Report before acting.** Read the state and tell the human what you see — active/paused
   campaigns, drafts pending approval, research-pending counts, sendbox health, the automation
   tasks declared in `daily-content-pipeline/automation/` — and reconcile against
   `automation_manifest.md`. A takeover begins by proving you can read the system, not by
   changing it.
3. **Re-register the automations in THIS runtime's scheduler**, using the carried prompt files
   (`daily-content-pipeline/automation/*_scheduled_run_prompt.md`) and the cadence/timezone
   recorded in `schedule.md` / the automation manifest, then resync the manifest per the
   Automation Resync contract with this runtime's native task ids.
4. **One brain owns each scheduled task.** Confirm with the human that the OTHER agent's
   scheduled tasks are disabled (only the human can do that inside the other runtime) before your
   first scheduled run executes: two brains firing the same task against one campaign double-sends.
   Ownership is per TASK, not per install — one brain may own a client's Daily Run while another
   owns Update Watch; record each owner in `automation_manifest.md`. This rule governs SCHEDULED
   runs only. It does not stop the human from working with both agents interactively at the same
   time; see step 6.
5. Then operate normally — same playbooks, same tools, same gates. Nothing about the system knows
   or cares which brain is driving.
6. **Both brains may operate at once.** A swap is not a precondition for using a second agent: the
   human may keep Codex and Claude open together and give each of them different work against this
   same install — run this campaign here, enrich that lead there. That is a supported operating
   mode. Do not refuse it, and do not ask the human to close the other agent for ordinary
   interactive work. Load `playbooks/MULTI_BRAIN_OPERATIONS.md` and honour its claim discipline
   (per-client `run_lock` with `held_by_brain`, scoped work leases, the never-concurrent list —
   update flow, Setup Flow, one client's Daily Run, one campaign's approve-then-send — and brain
   attribution on every write) before touching client state. The other agent also stays installed
   as a cold standby; the swap back is this same procedure in reverse.

Every live install root must carry its own `AGENTS.md` and `CLAUDE.md` pointer files: absolute
paths to the workspace root, the verified source checkout, the runtime state root and the bridge,
plus an instruction to read the source `AGENTS.md` in full. Without them a fresh session opened at
the install root cannot tell it is standing on a live system, and the human has to re-explain the
whole setup by hand every time. Create or verify them during setup, on takeover, and on every
update; the exact templates are in `playbooks/MULTI_BRAIN_OPERATIONS.md`. They are install-local
operator state and are never committed to the product repo.

Use the canonical terms `public data sources` and `private data sources` in human-facing text. Do not shorten them, omit `data`, or use slash labels.

Client-facing deliverables are client-blind by default and must stay that way. Do not mention `Solo Agency`, `WideCast`, PDNA/provider tooling, `OpenAPI`, `MCP`, `Local Collector`, Chrome extensions, automation/scheduled tasks, API keys, Telegram, config files, agent/tool/debug details, or `INTERNAL_REPORT` in reports, PDFs, videos, blogs, captions, comments, or other assets intended for the client's client/customer. Client-facing output should read like a professional agency deliverable: insight, evidence, recommendation, draft, next action.

Every scheduled/manual report run must also create an operator-only file with `INTERNAL_REPORT` in the filename for each client/day/run. Put all Solo Agency, WideCast, provider, Telegram/social-platform, API-key/config, Local Collector, private data source inventory, automation freshness, delivery-capability, blocker, and debug details in that internal report, not in client-facing files. The internal report is for the user/operator only and must be clearly labeled `INTERNAL_REPORT - Not for client sharing`.

Before generating, reviewing, fixing, or packaging any client-facing report HTML/PDF, load `playbooks/06_AGENCY_REPORT_STANDARD.md` and then `playbooks/skills/report-design/SKILL.md`. Use `tools/solo_tool render-report` by default for report HTML rendering and PDF companion packaging. Do not write fresh one-off Python/browser/PDF scripts for ordinary report runs; fix the reusable renderer or log the exact blocker instead.

If the human asks to scan, monitor, collect, or review private data sources (logged-in groups, feeds, profiles, communities, or social sources) after any amount of conversation drift, reload `playbooks/PRIVATE_SOURCE_GATE.md`, `playbooks/02_PRIVATE_SOURCE_SETUP.md`, `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md`, and `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md` before taking action.

Never use Claude in Chrome, Claude Chrome Extension, Codex built-in/in-app browser, Playwright/Puppeteer/Selenium, a fresh agent-opened browser profile, or any agent-controlled browser to read private data sources. Use only the Solo Agency Local Collector extension plus the Local Collector app for private data source collection.

For Facebook keyword group search discovery, use only the Local Collector with explicit human consent. Build search URLs like `https://www.facebook.com/search/groups/?q={url_encoded_keyword}`, scroll 10 times per keyword, filter UI noise/non-group results, and ask the human to approve recommended groups before adding them as private data sources. Never join groups or request access for the human.

When the human provides or approves private data sources, tell them they must already be a member, follower, subscriber, logged in, or otherwise authorized to view those sources in the Chrome profile where the client-specific Solo Agency Local Collector extension is installed. Recommend one separate Chrome profile per client, with that client's extension loaded and the relevant social accounts logged in there.

During Local Collector activation, do not run `setup_collector.sh`, PowerShell setup scripts, `.cmd` launchers, or collector binaries from inside the AI agent, even if shell permissions are available. Prepare the files, then give the human the one-line Terminal/PowerShell command to run outside the AI sandbox and the Chrome extension `Load unpacked` folder path. Before giving that command or folder path, run the Stage 8 Source Safety Pre-Check (read the extension JS, `bridge-go/main.go`, and `prepare_client_extension.sh` for outbound requests and confirm everything only talks to the local `127.0.0.1` bridge). A verified-fresh checkout is not enough. When it passes, precede the install steps with one short plain-language line confirming you read the code and it is safe to install; do not dump findings on a non-technical user. If a real request goes off the local machine, do not give the install command — stop and raise it to the operator.

Whenever adding a new client, create or verify that client's dedicated extension folder under `extensions/{client_slug}/`, patch the extension name to `{Client Name} - Solo Agency Collector`, and show the human the absolute extension folder path plus Chrome `Load unpacked` steps. Do not merely say "I created the extension"; the add-client handoff must tell the human exactly which Chrome profile to open and which folder to select.

When checking an already-running Local Collector app, do not trust `ready` alone. Verify `/status.config_file`, `/status.output_dir`, and `/status.run_now_request_file` point to the current setup's `daily-content-pipeline/collector/` tree. If they point to another setup, treat it as `wrong_workspace_bridge`, ask the human to run the current setup's Local Collector command, and remind them to remove/disable old Solo Agency Local Collector extensions in `chrome://extensions`.

During scheduled/manual runs, do not skip private data sources only because saved config says `public_data_sources_only`, `private sources postponed`, or `pending_private_activation`. If private data sources exist in any state or collector status files exist, perform Collector Runtime Verification first: try `/status`, verify current-workspace identity, and if localhost is unreachable from the AI sandbox, read local collector health/status files before deciding.

For multi-client manual/scheduled private collection, do not have multiple agents write the same `daily-content-pipeline/collector/run_now_request.json` file. Prefer `POST /jobs/run_now` when localhost is reachable, or write one unique per-client job file under `daily-content-pipeline/collector/jobs/pending/` when using file-based requests. The shared bridge supports parallel private collector jobs across different `client_slug` identities, binds each active job to the claiming extension instance when present, and serializes jobs only within the same client/profile.

After a schedule/automation has been configured, any later human-approved change to sources, approvals, Local Collector status, PDNA, provider/OpenAPI config, notification, analytics, profile fields, cadence, or playbook behavior must trigger an Automation Resync. Do not update only one config file. Update the Client Intelligence Profile, provider config/capability cache when relevant, schedule.md, collector config when relevant, automation manifest, scheduled-run prompt/task body, and resync log; then verify the next scheduled run will read the newest state.

Every human-facing progress block after schedule/automation exists must include an Automation freshness check: whether the latest changes were synced into the automation/scheduled task prompt/contract/playbook/source state, not only config, and whether tomorrow's scheduled run will load the newest state.

Every scheduled/manual report handoff must include a Provider Report Delivery Capability Check outcome in the operator-facing handoff and `INTERNAL_REPORT`: whether the configured provider/OpenAPI spec was discovered, whether the provider account was verified, whether HTML upload/Telegram or email-fallback notification was attempted, the uploaded URL or exact blocker, the final client-facing HTML report path/link, the mandatory PDF companion path/status, and the `INTERNAL_REPORT` path/status. For WideCast, check Client tools first: the per-client provider config, OpenAPI path (`https://widecast.ai/openapi.yaml`), verified account identity, and operations such as `uploadAsset` and `sendNotification`. Do not claim WideCast itself lacks capability merely because the current AI/MCP tool surface does not expose it.

Whenever checking tools, capabilities, credits, connected platforms, notifications, analytics, production, publishing, upload, or video/blog/social availability, check Client tools first and global MCP/native tools second. "Client tools" means the current client's `integrations/providers/provider_config.local.json`, fetched OpenAPI spec, verified account identity, `provider_capabilities.json`, and provider health/call logs. Global MCP/native tools are only optional compatibility after the client identity is proven to match. Do not tell the human "there is no video/blog/publish/notification tool" until the client-scoped provider config/OpenAPI/capability cache has been checked or refreshed and the exact blocker has been logged.

During PDNA setup or any WideCast/account-level provider check, do not treat a global MCP connector or current chat tool account as proof that the current client is connected. First identify `target_client_slug`, read that client's `integrations/providers/provider_config.local.json`, discover/cache the provider OpenAPI spec, verify the account with the client's configured API key, and compare the verified account identity to the saved client provider identity. If the per-client config is missing or identity cannot be verified, ask only for the client's WideCast API key by default and log `provider_config_missing`, `provider_auth_missing`, `provider_account_mismatch`, or `global_mcp_not_client_scoped`; do not ask provider/scope/spend/publish/account-identity questions unless the human explicitly requests a non-default provider or specialist stack, and do not list MCP-global credits/platforms as this client's PDNA status.

For WideCast OpenAPI server selection, use `https://widecast.ai/app/dashboard` as the current production server. Treat `https://api.widecast.ai` as a disabled/planned vanity host unless a future Solo Agency playbook explicitly enables it. If the OpenAPI `servers` list includes both, skip `api.widecast.ai`; do not try it first, do not fall back to it, and log `provider_server_disabled` or `provider_server_selection_corrected` if an older config/cache selected it.

For video/blog/social production actions, load `playbooks/SOLO_AGENCY_VIDEO_PROVIDER_ADAPTER.md` after any vendored writing or provider skill. Vendored WideCast skills are upstream-managed and may be replaced during build; do not patch them for Solo Agency client-routing policy. If a vendored skill says to call a concrete MCP tool such as `widecast_create_video`, resolve that as a client-scoped provider capability from the current client's provider config/OpenAPI cache first, and use MCP only when the tool identity is proven to match the current client.

For every video creation path, treat video scripts inside reports, Markdown records, prior drafts, or history as reference context only. Before any `production.create_video`, `widecast_create_video`, or equivalent provider request, load and apply the existing WideCast video script-writing skill through the verified client provider when available or from `playbooks/skills/video-script-writing/SKILL.md` / static fallback when PDNA is not connected. Do not edit, replace, summarize, or reimplement that WideCast skill. The provider payload must be the final script/brief produced by that skill, with vetted direct image URLs/media pool when relevant, not a report draft pasted through unchanged. If PDNA is missing, still produce/save the WideCast-grade script and production brief from the skill, then stop at the PDNA setup blocker without creating local video media. In manual/interactive runs, stop after the final script/visual handoff and wait for explicit human confirmation before creating the video; in an authorized scheduled Automation Flow, do not add a second script-confirmation gate when valid video-creation approval already exists.

The five video script versions inside a report are only a selection surface. If the human has picked a version/code, pasted an edited version, or the Automation Flow has a saved recommended/approved version, do not generate five new versions again during video production. Load the WideCast video script-writing skill and continue only with the selected version/code into the skill's research, factual-core, Stage 2 visual treatment, inline image/video URL, media-pool, and production handoff standards. Generate the five-format Stage 1 set only when no version has been selected or recommended yet.

When saving a client provider API key, use `api_key_env` or `api_key_local` exactly. Do not create `api_key`; the OpenAPI helper ignores that field and treats auth as missing.

When explaining WideCast/API-key setup, give the exact human steps: register at `https://widecast.ai/#setup` (free 50 credits/month when that offer is shown), log in, click `Setup AI Agent`, open the `API Keys & MCP` tab, click `Setup`, click `Generate API key and MCP url`, then copy only the API key back to the agent for this client's config. Mention Telegram and social account connection as optional WideCast-side setup, not as extra questions the human must answer in chat. Do not say or imply that adding an API key authorizes automatic posting of unreviewed drafts.

The repo entrypoint is `SOLO_AGENCY_PLAYBOOK.md`, not `solo-agency-collector/`.

The Bridge + local UI contract (URL map, ui_inbox file bus, Python-to-Go absorption plan) is `docs/UI_DESIGN.md`; when building or changing the bridge/UI, that file wins — amend it first.

Cold-email / CRM work is the separate OutreachCRM module at `outreach/` — see the OutreachCRM row in the `SOLO_AGENCY_PLAYBOOK.md` Stage Map; it has its own Stage Map, gates, and approve-then-send flow, and is not the Stage-10 lead-`outreach` action gated in Solo Agency's own pipeline. Its Stage-1 setup may bootstrap (read-only, one-way) from a client's existing Solo Agency Client Intelligence Profile; Solo Agency never reads the client's `outreach/` subtree. OutreachCRM setup runs INSIDE the same setup session (never spawn a dedicated outreach setup session); automation is one task per campaign (`{Client} - {Campaign} Daily Run`) plus the client's content Daily Run.

## Imported Claude Cowork project instructions
