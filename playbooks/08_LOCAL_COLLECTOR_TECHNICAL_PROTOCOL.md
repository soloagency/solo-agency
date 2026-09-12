# Local Collector Technical Protocol

Stage: `08`

## Load Rule

Load when installing, starting, stopping, checking, scheduling, updating, or troubleshooting the Solo Agency Local Collector extension or Local Collector app.

## Hard Gates For This Stage

- Default localhost is `127.0.0.1:17321`.
- Use `GET /status` for health checks.
- Use `POST /jobs/run_now` or per-client queue files under `daily-content-pipeline/collector/jobs/pending/` for manual/private run-now jobs. `run_now_request.json` is a legacy/batch shim that the bridge converts into queued jobs.
- Do not fake extension health by sending extension-only headers from the AI agent.
- Setup scripts must preserve data/config and stop only old collector processes occupying port 17321.
- When this stage is loaded for a private data source request, first reload `playbooks/PRIVATE_SOURCE_GATE.md` if it is not already loaded in the current private data source turn.
- Never use Claude in Chrome, Claude Chrome Extension, Codex built-in/in-app browser, ChatGPT/Gemini/Grok browser, Playwright/Puppeteer/Selenium, a fresh agent-opened browser profile, remote-debugging browser, or any agent-controlled browser for private data source collection.
- **Local vs remote runtime rule for install/start (hard gate).** On a LOCAL runtime — the agent's own shell IS the human's machine, e.g. Claude Code desktop/CLI, Codex CLI, or any other local agent runtime — the agent installs and starts the bridge itself in any flow (setup, update, repair, or normal runs): write `setup_collector.sh` (`setup_local_collector.ps1` / `Start Local Collector.cmd` on Windows) with the absolute path filled in as always, give the human the one-line plain-language safety confirmation (see Source Safety Pre-Check), ask for consent once, run the script, then poll `GET http://127.0.0.1:17321/status` for up to 60 seconds waiting for an answer, and report success or the specific failure. This is safe because the script hands the process to an OS-level supervisor — macOS LaunchAgent, a Linux systemd user unit, or a Windows logon Scheduled Task (see OS Startup For Persistent Bridge) — so the running bridge is not a child of the agent's own shell/turn and outlives it; do not fall back to "hand the human the command" just because the agent could run it. On a REMOTE runtime — a hosted/cloud sandbox where `127.0.0.1:17321` is NOT the human's machine — installing/starting stays banned: detect this by (a) the agent cannot see the install root on its own filesystem, or (b) `/status` still fails 60 seconds after a successful-looking bootstrap. Only then prepare the files and hand the human the exact one-line command to run outside the agent's sandbox.
- **Unattended permissions on a local Claude Code runtime.** The consent, the exact allow-rule set written to `~/.claude/settings.json`, and the manifest fields that record the outcome all live in `playbooks/04_DAILY_SCHEDULE.md` ("Unattended runs on Claude Code desktop (permissions)") — this file does not repeat them. Every bridge/`solo_tool`/`curl` command a scheduled run issues against this collector must match the Command Shapes rule (`playbooks/SCHEDULED_RUN_ENTRYPOINT.md`) those allow rules key off of, or the run pauses on a permission prompt instead of completing unattended.
- **Sandbox localhost rule (hard gate).** A scheduled/automation run executes INSIDE the AI sandbox, where `127.0.0.1:17321` is normally NOT the human machine's Local Collector localhost. In sandbox/automation runs the agent must NOT depend on localhost as the control path: submit work through the file-based job queue (`daily-content-pipeline/collector/jobs/pending/`, one unique per-client file) and verify liveness from local health/status files (`bridge_health.json`, `collector_status.json`, recent run-now consumed status). The agent must NEVER tell the human the Local Collector is down, stopped, unresponsive, or needs a restart because a localhost request failed — a failed localhost call in a sandbox is expected network isolation and proves nothing about the collector, which may be running perfectly on the human's machine. A Local Collector error/blocker is valid ONLY when the FILE-QUEUE path itself fails: the local health/status files are missing, stale, or point to another workspace, OR a submitted job file is not claimed/consumed within its TTL. Only then report the exact blocker (`collector_status_unverified`, `collector_offline_or_unreachable`, `wrong_workspace_bridge`, or `job_not_consumed`) and continue with public data sources and previously collected private data.
- Before handing the human any command to run the bridge or any Chrome `Load unpacked` path, the agent must run the Source Safety Pre-Check (see the section below) and only then give the install steps. A verified-fresh checkout is not enough on its own.
- One-time setup must complete both actions — bridge install/start, then Chrome extension install — before collection is claimed healthy. On a local runtime the agent performs both itself (bridge per the rule above; extension via the two-gesture dashboard flow below) and only asks the human to do the two physical clicks Chrome requires (Developer mode, drag the folder). On a remote runtime both stay human-run.
- No credentials, hidden APIs, DMs, inboxes, account pages, or contact scraping.
- Private data source discovery jobs are allowed only after explicit human consent and must produce candidate sources for review, not automatically activated monitoring sources.
- A reachable bridge is not automatically healthy. The agent must verify `/status.config_file`, `/status.output_dir`, and `/status.run_now_request_file` point to the current setup's `daily-content-pipeline/collector/` tree. If they point elsewhere, mark `wrong_workspace_bridge` and apply the current setup's install/start rule (local runtime: run it again itself; remote runtime: hand the human the command).
- A normal machine should have one active shared Solo Agency Local Collector runtime/bridge for the current setup, and one client-specific Solo Agency Local Collector Chrome extension per client Chrome profile/account. When old installs are suspected, ask the human to remove/disable stale entries in `chrome://extensions` and keep only the current per-client extension entries under `extensions/{client_slug}_extension/`.

## Source Safety Pre-Check

Before giving the human any bridge start command or any Chrome `Load unpacked` path, read the code that will actually run on the human's machine and confirm it does not send data anywhere off the local machine. This is a light "where do requests go" read, not a full security audit. It protects against the case where the upstream repo was hijacked and now ships code that exfiltrates the human's logged-in data — something the fresh-checkout verification alone cannot catch.

Scan these three, in the exact copies that will be installed/run (the prepared per-client folder, not only the template):

1. The per-client extension folder `extensions/{client_slug}_extension/` — every `*.js` file.
2. The bridge — NOT its source. The Go source is closed and absent from this repo (only checksum-verified binaries ship from the `dist` branch), so do not look for `main.go`; confirm instead that the installed binary is the one `setup_collector.sh` verified against `SHA256SUMS`, and hold its network behaviour to the disclosed outbound list below.
3. `solo-agency-collector/scripts/prepare_client_extension.sh`.

What to look for (outbound requests only):

- Network call sites: `fetch(`, `XMLHttpRequest` / `.open(`, `navigator.sendBeacon`, `new WebSocket(`, `new EventSource(`, image beacons (`new Image()` / `.src =` to a URL), and in the bridge any outbound HTTP client (`http.Get`/`http.Post`/`http.Client`/`net.Dial`) or `curl`/`wget` in the script.
- For each real call site, confirm the destination is the local bridge only: `http://127.0.0.1:<port>` or `http://localhost:<port>` (the `bridgeBaseUrl`). If `client_binding.json` overrides `bridge_base_url`, confirm that override is also `127.0.0.1`/`localhost`.
- Confirm the bridge binds to `127.0.0.1`/`localhost` (not `0.0.0.0`). Its COLLECTOR routes (`/status`, `/config`, `/capabilities`, `/collect/*`, `/jobs/*`) have no outbound or telemetry client. The bridge's only outbound calls are this disclosed set: (a) `widecast.ai` with the client's own WideCast API key — provider operations, notifications, the tracking secret/events, and the Solo Agency plan check `GET /v1/solo/entitlement` at start and about every 24 hours, which sends only that key, a random install id and the bridge version, never anything the collector gathered; (b) Gmail SMTP/IMAP for OutreachCRM sendboxes the human connected; (c) `raw.githubusercontent.com` from the setup/update script for binaries and playbooks. Anything else is a finding.

Known false positives — DO NOT flag these (read the whole line for context; never flag on a substring match alone):

- `readability.js` is the vendored Mozilla Readability DOM parser. Its `XMLHttpRequest`/`_ajax` helper is defined but never called, and its many `http://` strings are license, attribution, and code-comment/docstring references. Not egress.
- URLs inside comments or usage examples (e.g. a `https://www.facebook.com/...` line inside a `/* Usage: ... */` block).
- A placeholder base URL such as `https://example.invalid/` passed into `new URL(href, base)` to resolve relative links — it is never fetched.
- `fetch(chrome.runtime.getURL("client_binding.json"))` and any `chrome.runtime.getURL(...)` / `chrome-extension://` target — this reads a file packaged inside the extension, not the network.
- Broad `host_permissions` like `http://*/*` and `https://*/*` in `manifest.json` — a page-reading collector needs read access to whatever site the human approved. Broad read scope is expected and is not exfiltration; judge only by where requests are sent.
- Substring artifacts from a plain text search (e.g. `nc ` inside `func `, or `http` inside a comment). Confirm against the actual code line.

Outcome:

- If every real outbound request from the EXTENSION goes only to the local bridge, and the bridge's outbound calls are only the disclosed set above: the pre-check passes. Record the result (files/commit reviewed, call sites checked, destinations) in `INTERNAL_REPORT` only, and give the human exactly one short, calm confirmation line in plain language before the install steps, for example: `I read through the collector's code and confirmed that nothing it collects from Facebook or Zillow ever leaves your computer. The only thing it talks to online is your own WideCast account, the same key you already entered, to send notifications and confirm your plan. It is safe to install.` Do not list findings, severities, or technical terms to the human, and do not add extra warnings that could worry a non-technical user.
- If a real request goes to any non-local destination, or the bridge opens an outbound connection, or code is obfuscated so the destination cannot be read: do NOT say it is safe and do NOT give the install command. Stop, record the exact finding (file, line, destination) in `INTERNAL_REPORT`, and raise it to the operator in an `**[ACTION REQUIRED]**` block, in calm plain language, so it can be checked against the latest verified GitHub source before any install.

## Latest Override: One Shared Bridge, Many Client Extensions

The current multi-client model supersedes older one-extension wording:

- A normal machine should have one active shared Local Collector app/bridge for the current agency root.
- Each client may have its own Chrome profile/account and must have its own unpacked extension folder under `extensions/{client_slug}_extension/`.
- That one unpacked extension folder covers all three social platforms — it is not per-platform. Internally it carries separate `platforms/facebook`, `platforms/instagram`, and `platforms/x` modules, but the human installs it once (setup step 4, "Kết nối Facebook, Instagram and X") and it reads whichever of the three platforms the human is logged into in that Chrome profile. The Login Reminder likewise names all three platforms in one prompt, not three separate reminders.
- Recommend one separate Chrome profile per client. That profile should have the matching client extension installed and should be logged in to the social accounts/private data sources that the human is already authorized to view for that client.
- The Chrome extension display name must begin with the client name: `{Client Name} - Solo Agency Collector`.
- The human-facing `Load unpacked` folder for a client is the absolute path to `extensions/{client_slug}_extension/`, not `solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME/`.
- The shared bridge routes jobs by `client_slug`, binds each active job to the claiming `extension_instance_id` when present, can run different client identities in parallel, serializes only jobs for the same client/profile, and writes output only under `daily-content-pipeline/collector/inbox/YYYY-MM/{client_slug}/{run_id}/`.
- Agents running in sandboxes should prefer file-based job requests under `daily-content-pipeline/collector/jobs/pending/` and local health/status files over localhost calls.
- Extension popup/settings must not write global `collector_config.json`; global agency/collector config is managed by the agent/playbook and Automation Resync.
- Folder resolution order (bridge `uiExtensionInfo`, OWNER DECISIONS 2026-09-10 item A): an `extension_registry.json` pin for that `client_slug` wins unconditionally; otherwise `extensions/{client_slug}_extension/` when it exists; otherwise, only for installs made before the 2026-09-10 rename, the legacy `extensions/{client_slug}/`. Every new client gets the `_extension` name — the legacy path is a read for backward compatibility, never a name the agent creates.

Per-client extension setup handoff — two gestures, folder revealed automatically:

```text
The agent opens the extension folder and best-effort opens your browser's extensions
page for you (local runtime), or the dashboard page shows the folder path and steps for
you to do it yourself (remote runtime) — no button to find either way.
Turn on Developer mode (once).
Drag the highlighted folder onto the extensions page.
```

**Every `/ui/...` link named anywhere in this file — this one and every one below it — follows the
SHOW RULE** (`docs/UI_DESIGN.md` §1 principle 2, OWNER DECISION 2026-09-10): print it as text every
time; on Claude Code desktop also open it in the side Browser pane; on any other local runtime also
run `open`/`start`/`xdg-open` so it lands in a real browser; never HTTP-GET it to "verify" — a
sandboxed agent's own GET proves nothing about the human's browser. The dashboard now opens
directly (`--ui-auth host` default) — no entry-link/token step, no "Locked" page. Which exact
`/ui/...` path to open for a given question is never memorized — it is looked up in the routes
table (`tool ui routes` / `GET /api/ui/routes`), per the Answer-and-Show Rule
(`SOLO_AGENCY_PLAYBOOK.md`, "Team Leader Reply Frame").

The `install-extension` endpoint fires both gestures at once because the bridge runs on the human's own machine and can shell out directly. The folder-reveal half highlights the folder inside its PARENT window instead of opening the folder's own contents — `open -R "<folder>"` on macOS, `explorer /select,"<folder>"` on Windows, an `xdg-open` of the parent directory on Linux (no cross-desktop "select in parent" API there, so the page's own text names the folder). The extensions-page half (no explicit browser chosen) is `open -a "Google Chrome" "chrome://extensions/"` on macOS (works even while Chrome is already running), `cmd /c start "" chrome "chrome://extensions/"` on Windows, `google-chrome`/`xdg-open` fallback on Linux. A local-runtime agent triggers the identical two actions itself by calling `POST /api/ui/{client_slug}/install-extension` on the bridge (documented with the UI worker's endpoints) — this is the default path now, not a fallback to a human click, since the page's own Install button lives behind `?advanced=1` — then polls `GET /status` until `extension_health.status` is recent (75-second grace window) before telling the human it's connected. The absolute path `{ABSOLUTE_AGENCY_ROOT}/extensions/{client_slug}_extension/` is the manual fallback for the file picker only.

**No browser question, ever.** Setup never asks about the browser or the profile the human uses — most humans do not know what a profile is, and the extension is bound to the client by `client_binding.json` inside the folder, never by a profile. The human installs it in whatever Chromium-based browser they already use — Chrome, Edge, Brave, Vivaldi, Opera, Chromium — and the agent never says which ones it detected and never asks a Chrome-vs-Edge question or anything like it. The default install page (video, two steps, folder path) never shows a browser or account picker; those controls exist only behind `?advanced=1` for an operator, and `GET /api/ui/{client_slug}/browsers` is used only by that advanced page, never by the agent in chat. The agent's own `POST /api/ui/{client_slug}/install-extension` call carries no `browser`/`profile_directory` fields — it just reveals the folder and best-effort opens `chrome://extensions` in whichever Chromium browser is already running (`open -a "Google Chrome" "chrome://extensions/"` on macOS; the Windows/Linux equivalents above), and the human is separately told, in the instruction text, to open their own browser's extensions page regardless of which one that automatic open landed on. The single human-facing exception is not a question: when the machine has no Chromium-based browser at all (only Safari and/or Firefox), the agent says Chrome is needed and offers to install it.

Alongside the reveal, the agent says one fixed sentence, in the human's language, naming the exact folder to pick (OWNER DECISIONS 2026-09-10 item A): "Chọn đúng thư mục tên `{client}_extension` mà em vừa mở — không chọn thư mục `chrome-extension` nằm trong mã nguồn." This step always follows the Show the install video rule (OWNER DECISIONS 2026-09-11, updated for the default advanced-mode-gated page): when the bridge answers `/status`, open `http://127.0.0.1:17321/ui/{client_slug}/extension` in the side browser — the video plays right under the two install steps on that page — and print the link; on any local runtime, always also put the local mp4 file itself in front of the human — send it into chat when the runtime can, otherwise open it with the OS default player (macOS `open "{setup-root}/solo-agency/solo-agency-collector/bridge-go/assets/setup_extension_chrome_small.mp4"`, Windows `start "" "{setup-root}\solo-agency\solo-agency-collector\bridge-go\assets\setup_extension_chrome_small.mp4"`, Linux `xdg-open ...`; use the `_edge_small.mp4` file for Edge) — and never print the public GitHub links there, since the file is already on disk; only on a REMOTE runtime does the agent print the two public GitHub links as plain text instead: `https://github.com/soloagency/solo-agency/blob/main/solo-agency-collector/bridge-go/assets/setup_extension_chrome_small.mp4` (Chrome) and `https://github.com/soloagency/solo-agency/blob/main/solo-agency-collector/bridge-go/assets/setup_extension_edge_small.mp4` (Edge). The bridge also serves both recordings directly at `http://127.0.0.1:17321/ui/assets/help/setup_extension_chrome_small.mp4` (Edge: `…_edge_small.mp4`) — embedded in the binary, so that URL works no matter how old the human's clone is and is the preferred local source; the on-disk copy under `solo-agency-collector/bridge-go/assets/` only exists on installs that have pulled since it was added, so check the file exists before naming or opening it, and fall back to the embedded URL when it is missing.

Which browser actually opens is whatever the OS/best-effort command above lands on — the agent never asks and never records it in chat. `extension_registry.json`'s `browser`/`profile_directory` fields exist only for the advanced page's own operator use (`playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`); a second Facebook account only comes up when a SECOND client needs a different one, and even then it is the human's own choice on the advanced page, never a question the agent asks in chat.

Every Add Client or First Client Setup handoff must include this block with the real absolute path for the fallback. The agent must not merely say that the extension was created. The human needs the folder revealed (or the path, on the manual fallback) because a new unpacked extension must be loaded into the matching client Chrome profile/account before private data source collection can work for that client.

The agent must run the Source Safety Pre-Check first and precede this handoff with the one short plain-language safety confirmation line (see the Source Safety Pre-Check section). Do not give the `Load unpacked` button/path or the bridge install/start step until the pre-check has passed.

The agent must prepare `extensions/{client_slug}_extension/manifest.json` with at least:

```json
{
  "name": "{Client Name} - Solo Agency Collector"
}
```

It may also set `"short_name": "{Client Name} Collector"` and a client-specific `"description"` / `"action.default_title"`. The helper `scripts/prepare_client_extension.sh` patches only `name`, `description`, and `action.default_title` (not `short_name`); its output is compliant. `short_name` is optional.

The agent must also create `extensions/{client_slug}_extension/client_binding.json` with `client_slug`, `client_name`, `extension_instance_id`, `extension_display_name`, and `bridge_base_url`.

Bridge/extension health for automation must be checked per client. A global `extension_health.status: recent` is not enough when multiple client extensions exist; the scheduled task must find the matching extension entry for the target `client_slug` and `extension_instance_id`.

## Source Preservation Rule

This file is detailed source material moved from the original monolithic `SOLO_AGENCY_PLAYBOOK.md`.

Do not summarize away requirements, examples, checklists, schemas, protocols, URLs, edge cases, warnings, approval gates, or completion gates. If a downstream agent needs to shorten its response to the human, it may summarize the response, but it must still obey the full requirements in this file.

---

## Latest Delta Override: Discovery Mode Versus Daily Monitoring

Source Discovery Mode normally scrolls until no new source names or URLs appear for 3 consecutive scrolls, with a hard safety cap of 10 scrolls.

Facebook keyword group search discovery is a bounded Source Discovery Mode variant: for `https://www.facebook.com/search/groups/?q={url_encoded_keyword}`, use 10 scrolls per keyword by default, with `purpose: "facebook_group_keyword_search_discovery"`. This search-results pass is for collecting candidate groups, filtering UI noise, and registering each one through the Group Potential Rule — no approval before adding a group. It must not join a group or request access; a group the account cannot read is registered `no_access`, never joined.

Daily Content Monitoring Mode keeps the conservative default: 5 scrolls, max 10, 5 seconds between scrolls, and about 20 private data sources or fewer per client.

Do not apply the daily 5-scroll default to source discovery.

Discovery scrolls must be real page-sized scrolls. The Local Collector extension should scroll the actual active scroll container, not merely nudge `window` by a small amount. If a discovery run reports a high scroll cap but finds only the first few dozen sources, inspect the collector output for `scroll_debug`, `scroll_count`, and `scroll_stopped_reason`; low deltas or repeated `no_scroll_movement` mean the extension did not move through the list deeply enough and the run should be retried after updating/reloading the extension.

Chrome may throttle hidden/background tabs, and some social feeds hydrate only when the page is visible. Current extensions should use `0.1.10-filtering-capture` or newer. This build does not depend on `requestAnimationFrame`, always keeps the full cleaning pipeline (`filtering.js`, `readability.js`, and `infinity_loops.js`) in the automated capture path, gives 5-10 scroll passes enough time to finish, times out truly stalled content-script captures, and clears stale active-run locks after an extension build update. If bridge and extension heartbeat are recent but `source_status.jsonl` remains stuck at `started` with zero data points, audit the bridge/extension contract first: client identity headers, run ownership, write token, POST endpoint responses, and output folder routing. Then check `capture_timeout_needs_visible_collector_window_or_site_access`, `inject_capture_files_timeout_needs_site_access`, login/checkpoint screens, missing site access, wrong Chrome profile, or a minimized/frozen Chrome window. Do not bypass `filtering.js` for private data source reports. The safe fallback is a dedicated per-client collector Chrome profile/window with that client's extension active inside it; do not use Claude in Chrome, Codex/browser tools, Playwright, or any agent-controlled browser for logged-in private data sources.

Lead And Competitor Detection Mode is part of normal data collection, not a separate extra scan:

- First lead/competitor pass for a client/source set: 10 scrolls per approved private data source when Local Collector is active and safety settings allow it.
- Recurring daily scheduled runs: 5 scrolls per approved private data source by default.
- Extract lead and competitor opportunities during the same pass used for ideas, market signals, data points, and source quality.
- Do not run a second lead/competitor scan unless the human explicitly asks for a deeper pass, the first scan failed, or the saved schedule/config allows it.
- If `collector_config.max_scrolls_per_source` is lower than the desired lead/competitor depth, obey the safer lower setting and record the coverage limitation in the report.
- Always load Stage 10 before reporting lead/competitor opportunities.

## Scan Depth Disclosure Rule

Whenever the agent announces that it will scan groups, communities, fanpages, social profiles, or other private data sources, it must disclose the scan depth in plain language.

For daily content monitoring, say:

```text
I will go through each approved group/source one by one and scroll {N} times per source. I will read {N} from the Local Collector configuration when available; otherwise I will use the safe default of 5 scrolls, max 10, with about 5 seconds between scrolls.
```

For the first lead/competitor pass, say:

```text
I will go through each approved group/source one by one and scroll 10 times per source for the first lead/competitor pass, if the Local Collector configuration and account-safety limits allow it. Future daily runs will usually use 5 scrolls per source.
```

To resolve `{N}`, use this order:

1. Read `daily-content-pipeline/collector/collector_config.json`.
2. If the Local Collector app is running, call `GET http://127.0.0.1:17321/status` and/or `GET /config` when available.
3. Fall back to `5` only when config cannot be read.

For source discovery, disclose the different rule:

```text
This is source discovery, not daily monitoring. I will scroll the actual list/page roughly one screen at a time until no new source names/URLs appear for 3 consecutive scrolls, with a hard safety cap of 10 scrolls.
```

Do not let the human think "scan groups" is unbounded or vague.

When the human has no private data source list, discovery is a first-class Local Collector job type, run automatically once the step-7 first-run yes records `approved_pending_first_scan` for all categories — no separate question. The job covers every relevant discovery surface, such as joined Facebook groups, Facebook keyword group search, joined/subscribed subreddits, followed pages/KOLs, subscribed channels, communities, and feeds. It must return candidate sources with enough context for the agent to score and register each one automatically (the Group Potential Rule for Facebook groups, `playbooks/10_LEAD_COMPETITOR_DETECTION.md`) — no shortlist, no human approval before saving a source as active.

---

### Preferred Private Data Collector Architecture

Use this architecture first whenever possible:

```text
User's logged-in Chrome
  -> Solo Agency Local Collector extension
  -> Local Collector app on this computer
  -> Local JSONL / status / HTML snapshot files
  -> Claude, Codex, Hermes, OpenAI agents, or other AI agents read the files
```

Human-facing naming rule:

| Technical/internal term | Human-facing term | Human-facing explanation |
|---|---|---|
| browser extension collector | Solo Agency Local Collector extension | A small Chrome add-on that reads visible pages you are already allowed to see while you stay logged in. |
| localhost bridge | Local Collector app | A small app running only on your computer. It receives data from Chrome and saves local files for the AI agent. |
| bridge binary / executable | Local Collector app file | The app file the AI agent downloaded for your operating system. |
| persistent bridge scheduler | Auto-run Local Collector | The Local Collector app starts with your computer and waits for the scheduled collection time. |

When communicating with the human during setup, the AI agent should say:

```md
I need to set up two local pieces once:

1. Solo Agency Local Collector extension: reads the private pages you are already logged into.
2. Local Collector app: receives that data and saves it on this computer so I can analyze it.

I will not ask for your passwords, cookies, or login codes.
```

Example non-technical setup wording:

```md
I will set up a local data collector on your computer. It has two parts:

1. Solo Agency Local Collector extension: this is added to Chrome so it can read pages you are already logged into.
2. Local Collector app: this is a small app on your computer that receives the data from Chrome and saves it locally for me to analyze.

You do not need to understand the technical details. I will give you the exact button/path/command when your approval is needed.
```

The collector layer exists because many AI agents have unreliable private-browser access:

- Claude may be sandboxed and unable to open a headed Playwright browser.
- Claude Chrome Extension must not be used for automated private data source collection because it may require the human to click Allow during runs and can stop an unattended schedule.
- Some agents can reason and write well but cannot safely operate logged-in social sessions.
- Some scheduled runs happen while the human is away.

The collector layer separates private data collection from reasoning:

- The Chrome extension uses the human's already logged-in Chrome session.
- The Local Collector app, internally called the localhost bridge, receives collected data and writes local files.
- The AI agent reads local files and performs filtering, lead detection, competitor detection, idea generation, script writing, reporting, and WideCast actions.
- No private data should be uploaded to a third-party server unless the human explicitly enables such export.

### Solo Agency GitHub Collector Distribution

For setup, the AI agent must use the Solo Agency GitHub repository as the primary collector distribution source.

Repository:

```text
https://github.com/soloagency/solo-agency
```

Raw download base:

```text
https://raw.githubusercontent.com/soloagency/solo-agency/main/solo-agency-collector/
```

Current collector artifacts:

```text
https://raw.githubusercontent.com/soloagency/solo-agency/dist/collector-bridge-binaries-0.1.0.zip
https://raw.githubusercontent.com/soloagency/solo-agency/dist/chrome-extension-collector-root-0.1.0.zip
https://raw.githubusercontent.com/soloagency/solo-agency/dist/SHA256SUMS
```

Current writing-skill artifacts:

```text
https://raw.githubusercontent.com/soloagency/solo-agency/main/playbooks/skills/video-script-writing.zip
https://raw.githubusercontent.com/soloagency/solo-agency/main/playbooks/skills/blog-writing.zip
https://raw.githubusercontent.com/soloagency/solo-agency/main/playbooks/skills/social-post-writing.zip
```

If the agent is already running inside a cloned copy of `https://github.com/soloagency/solo-agency`, it may prefer local repo files under `solo-agency-collector/dist/` for collector artifacts and `playbooks/skills/` for writing-skill artifacts only after the clone passes the Fresh Source Verification below. Unverified local files are not a distribution source.

Fresh Source Verification:

- Treat GitHub `main` as the source of truth for every one-time setup, repair, update, or Local Collector preparation.
- Do not reuse fixed shared fallback folders such as `/tmp/solo-agency`, `/var/tmp/solo-agency`, `/dev/shm/solo-agency`, or another generic cache path that may contain files from a prior session.
- If a temporary checkout is needed, create a fresh unique directory with `mktemp -d`, clone `https://github.com/soloagency/solo-agency`, and verify it before reading or copying any file.
- Verification requires `.git` to exist, `git remote get-url origin` to resolve to the Solo Agency GitHub repository, and `git rev-parse HEAD` to equal `git ls-remote origin refs/heads/main` after clone/fetch.
- A folder without `.git`, a folder owned by another user, a folder with an old timestamp, or a target that could not be deleted/updated is stale cache. Do not read from it, copy from it, or use it as fallback.
- If `rm -rf`, `git fetch`, `git pull`, `git clone`, `curl`, or archive download fails because of permissions, sandboxing, or network access, stop and request permission or give the human one exact GitHub command. Do not continue with the old local folder.
- Do not let shell chaining hide a failed cleanup. The agent must confirm the clone/download actually happened and must report the verified commit hash in setup status or chat before using the artifacts.

The AI agent should prepare the collector locally as much as its environment allows. On a LOCAL runtime it also runs the one-time setup script itself (one safety line, one consent ask) — the script hands the process to an OS-level autostart supervisor, so the Local Collector app survives after the agent's command/session ends regardless of who started it. Only on a REMOTE runtime must the setup/start command instead be run by the human outside the agent's sandbox.

Canonical local layout:

```text
{agency_root}/
  solo-agency/                         # downloaded toolkit/source repo
  solo-agency-local-collector/         # shared runtime app / bridge only
    downloads/
    bin/
    setup_collector.sh
    collector.pid
    collector.log
  extensions/                          # one Chrome Load unpacked folder per client
    {client_slug}_extension/           # legacy installs may still have {client_slug}/ — see below
      manifest.json
      background.js
      popup.html
      popup.js
      filtering.js
      readability.js
      infinity_loops.js
      collector_helpers.js
      icons/
      client_binding.json
  daily-content-pipeline/              # data/config/output only
    collector/
      collector_setup_status.md
      collector_config.json
      extension_registry.json
      jobs/
      inbox/
```

The listing above is a minimum; the extension folder contains more files than shown. Always copy the FULL template folder (or use the helper script `scripts/prepare_client_extension.sh`) so nothing is missed.

Chrome extension folder disambiguation:

- There may be another `chrome-extension/` folder inside the downloaded toolkit/repo, such as `solo-agency/solo-agency-collector/chrome-extension/`.
- That toolkit folder is source/developer material. It is not the human-facing Chrome `Load unpacked` folder during agency setup.
- The only folder the agent may tell a normal human to load in Chrome for a client is the per-client runtime folder under `extensions/{client_slug}_extension/`.
- If both folders exist, the agent must explicitly warn: `Do not load the extension folder inside solo-agency/solo-agency-collector. Load only the extensions/{client_slug}_extension folder shown below for this client.`
- A normal machine should have only one active shared Solo Agency Local Collector runtime/bridge, but may have multiple client-specific Solo Agency Local Collector extensions, one per client Chrome profile/account.
- The generated setup instructions, setup status file, and chat message must show the absolute per-client extension path: `extensions/{client_slug}_extension/`.
- Do not put app binaries, downloaded zips, the unpacked extension, PID files, or collector logs inside `daily-content-pipeline/`. That folder should remain data/config/output only.

Install flow:

1. Detect the user's OS and CPU architecture.
2. Establish a verified fresh source: use the current setup root only if it passes Fresh Source Verification, otherwise clone GitHub `main` into a fresh unique `mktemp -d` checkout. If fresh source cannot be verified, stop instead of using local cache.
3. Copy `SHA256SUMS`, `collector-bridge-binaries-0.1.0.zip`, extension templates, and helper scripts only from the verified checkout, or download the exact raw GitHub URLs above when cloning is unavailable.
4. Verify checksums when the environment has checksum tools available.
5. Extract bridge binaries into the absolute runtime path for `solo-agency-local-collector/bin/`.
6. Prepare the Chrome extension template into the absolute per-client path `extensions/{client_slug}_extension/`, patch the manifest name to `{Client Name} - Solo Agency Collector`, and create `client_binding.json`. Prefer the repo helper when available from the verified checkout:
   ```bash
   solo-agency-collector/scripts/prepare_client_extension.sh "{Client Name}" "{client_slug}" "{extension_instance_id}" "{ABSOLUTE_AGENCY_ROOT}"
   ```
7. Select the correct bridge binary for the current machine.
8. On macOS/Linux, ensure the selected binary is executable.
9. Create the setup/start script or launcher.
10. On a LOCAL runtime, run it: give the human the one-line plain-language safety confirmation, ask consent once, execute the script, then poll `GET http://127.0.0.1:17321/status` for up to 60 seconds. On a REMOTE runtime (agent cannot see the install root, or `/status` still fails 60 seconds after a bootstrap attempt), instead give the human exactly one Terminal/PowerShell command or one double-clickable launcher path to run outside the agent's sandbox.
11. Trigger the Chrome extension install through the dashboard's two-gesture flow (`/ui/{client_slug}/extension`, or `POST /api/ui/{client_slug}/install-extension` on a local runtime) and poll `extension_health.status` until recent; give the absolute per-client extension folder path only as the manual fallback.
12. Record the verified source path and commit hash in `daily-content-pipeline/collector/collector_setup_status.md` when the file exists or is being created.
13. After both actions are confirmed (by the agent's own polling on a local runtime, or by the human's "done" on a remote one), health-check `GET http://127.0.0.1:17321/status` and run the workspace identity check before claiming the collector is healthy.
14. Prefer persistent scheduler mode for unattended collection. After one-time setup succeeds, scheduled runs should use the already-running Local Collector app and should not ask the human to repeat setup.

Absolute path rule:

- The AI agent must never tell the human to load the Chrome extension from a relative path.
- The AI agent must resolve and show the absolute folder path.
- The AI agent must never show `daily-content-pipeline/collector/chrome-extension/` or `solo-agency/solo-agency-collector/chrome-extension/` as the folder for a normal human to load in Chrome. The first path belongs to the old mixed data/runtime layout; the second path is for source/development only.
- Correct examples:
  - macOS/Linux: `/Users/alex/oneman_agency/extensions/avenngo/`
  - Windows: `C:\Users\Alex\oneman_agency\extensions\avenngo\`
- Incorrect examples:
  - `daily-content-pipeline/collector/chrome-extension/`
  - `solo-agency/solo-agency-collector/chrome-extension/`
  - `solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME/`

Binary selection:

| OS | CPU | Binary |
|---|---|---|
| macOS | arm64 / Apple Silicon | `collector-bridge-darwin-arm64` |
| macOS | amd64 / Intel | `collector-bridge-darwin-amd64` |
| Windows | amd64 / x64 | `collector-bridge-windows-amd64.exe` |
| Linux | amd64 / x64 | `collector-bridge-linux-amd64` |

If the current OS/CPU is not listed, the agent must log `collector_unavailable`, continue with public data sources, and notify the human that a compatible collector binary is not available yet.

### Required One-Time Setup Handoff (Local-Run, With Remote Fallback)

During first Local Collector activation, setup repair, or collector update, a LOCAL runtime (Claude Code desktop/CLI, Codex CLI, or any other agent runtime whose shell IS the human's own machine) runs the setup itself instead of handing it to the human. This applies to Claude, Codex, Hermes, OpenClaw, ChatGPT, and other AI agents alike — the distinction that matters is local vs remote, not which agent it is. A REMOTE runtime (a hosted/cloud sandbox where `127.0.0.1:17321` is not the human's machine) still hands the human the command; detect "remote" by either signal: the agent cannot see the install root on its own filesystem, or `/status` still fails 60 seconds after a bootstrap attempt.

**Local runtime — the agent does both actions:**

1. Bridge install/start: write the setup file (below), say the one-line plain-language safety confirmation (see Source Safety Pre-Check), ask for consent once, run it:
   - macOS/Linux: `bash "/ABSOLUTE/PATH/TO/solo-agency-local-collector/setup_collector.sh"`
   - Windows: `setup_local_collector.ps1`, or the `Start Local Collector.cmd` launcher. SmartScreen may ask for one click on the unsigned script/binary — mention that only if it actually happens.
   Then poll `GET http://127.0.0.1:17321/status` for up to 60 seconds and report success or the specific failure.
2. Extension install, via the two-gesture dashboard flow: call `POST /api/ui/{client_slug}/install-extension` with no `browser`/`profile_directory` fields — it reveals the client folder highlighted inside extensions/ (macOS open -R, Windows explorer /select) and best-effort opens a Chromium extensions page in whichever browser is already running, brought to the front, in the same action. Tell the human the two gestures only a person can do: open their own browser's extensions page if it did not already come to the front, turn on Developer mode, and drag the highlighted folder onto it. Poll `extension_health.status` until it is `recent` (75-second grace window), then celebrate in chat. No browser or profile question is ever asked; a second Facebook account only comes up when a SECOND client needs a different one, resolved on the advanced (`?advanced=1`) install page, never in chat.

**Remote runtime — hand the human exactly these two actions in chat:**

1. Run the Local Collector app setup/start command outside the agent's sandbox:
   - macOS/Linux: `bash "/ABSOLUTE/PATH/TO/solo-agency-local-collector/setup_collector.sh"`
   - Windows: one prepared PowerShell command or one double-clickable `Start Local Collector.cmd` path.
2. Install the client-specific Solo Agency Local Collector extension in whichever browser window the human already uses for Facebook:
   - for the first client, that is the browser window already open and logged in; a separate account only comes up once a second client needs a different Facebook account, resolved on the advanced (`?advanced=1`) install page, never in chat;
   - make sure that window is already logged in to the approved private data sources and the human has member/follower/subscriber/access rights for them;
   - **prefer the dashboard page** — hand the human `http://127.0.0.1:17321/ui/{client_slug}/extension`; by default it shows the install video, the two steps, and the extension folder path (no button, no browser question), then Developer mode + drag; the absolute per-client folder `/ABSOLUTE/PATH/TO/extensions/{client_slug}_extension/` is the manual fallback for the file picker.

The human-facing setup message must show both actions together on a remote runtime. Do not say only "I started it", "I ran setup", or "instructions are in collector_setup_status.md".

After both actions are confirmed — by the agent's own `/status`/`extension_health` polling on a local runtime, or by the human's "done" on a remote one — check `GET http://127.0.0.1:17321/status` and inspect collector logs/status files before claiming the collector is healthy.

For later scheduled runs, do not ask the human to repeat these steps. Use the already-running persistent Local Collector app, create scheduled/run-now jobs when available, and notify the human only if the app or extension becomes unavailable.

Chrome extension installation flow:

1. The agent copies/extracts and patches the extension into an absolute per-client path, for example:

```text
/Users/alex/oneman_agency/extensions/avenngo/
```

2. On a local runtime, the agent triggers the two-gesture dashboard flow itself (`POST /api/ui/{client_slug}/install-extension`, no `browser`/`profile_directory` fields) and polls `extension_health.status`; it tells the human only the two gestures a person still has to do (open their own browser's extensions page if it is not already in front, turn on Developer mode, drag the folder in). Either way, mention the recorded video on the page too ("the page has a 30-second video showing exactly this" — in the human's own language). On a remote runtime, the agent tells the human directly in chat, Telegram, or another human-facing channel:

```md
Please install the Solo Agency Local Collector extension for {Client Name}:

**Easiest — the page shows everything you need:** open `http://127.0.0.1:17321/ui/{client_slug}/extension`. It plays a 30-second install video and shows the extension folder's absolute path (with a Copy button) above the two steps. Open your own browser's extensions page, turn on `Developer mode`, then drag that folder onto the extensions page. The page turns green when it connects. If the video doesn't play: https://github.com/soloagency/solo-agency/blob/main/solo-agency-collector/bridge-go/assets/setup_extension_chrome_small.mp4 (Chrome) or https://github.com/soloagency/solo-agency/blob/main/solo-agency-collector/bridge-go/assets/setup_extension_edge_small.mp4 (Edge).

**Manual fallback:**
1. Load it into whichever browser window you already have open and logged into Facebook for your first client; a separate account only comes up once a second client needs a different Facebook account.
2. Go to `chrome://extensions` (Edge: `edge://extensions`).
3. Turn on `Developer mode`.
4. Click `Load unpacked`.
5. Select this folder:
   `/ABSOLUTE/PATH/TO/extensions/{client_slug}_extension/`

Important: if you also see a folder named `solo-agency/solo-agency-collector/chrome-extension`, do not select that one. That is the toolkit/source copy. Select only the client folder under `extensions/{client_slug}_extension/`.

After this one-time setup, you may close this instruction tab whenever you want. For private data source collection to work at scheduled times, that browser window should be open, logged in to the social accounts/private data sources approved for this client, and already have member/follower/subscriber/access rights for those sources. The shared Local Collector app should be running or configured to auto-start.
```

3. The agent must not ask for passwords, cookies, OTPs, or credentials.
4. If the extension is not installed or cannot contact the Local Collector app, the agent logs `extension_unavailable`, continues with public data sources, and notifies the human.

The AI agent must always create a ready-to-run setup/start script file first. On a LOCAL runtime, it runs that script itself (one-line safety confirmation, one consent ask, then execute) instead of handing it to the human — users are very low-tech and must never be told to open a Terminal in the normal path. Only on a REMOTE runtime does the agent instead give the human exactly one short command to paste into Terminal or PowerShell.

Do not show the human a long multi-line script as the primary instruction. Non-technical humans should not have to copy a large code block — on a local runtime they should not have to open Terminal at all.

Do not tell the human that setup instructions are only in a Markdown file. The Markdown file may store the same information for agent memory, but the current chat must contain the exact action being taken (local runtime) or the exact action the human should take (remote runtime).

The generated collector setup script must be named `setup_collector.sh`. Do not invent alternative names such as `start_local_collector.sh`. A LOCAL runtime executes this script itself during one-time setup/update/repair; a REMOTE runtime provides the command for the human to run in Terminal instead. Every run must check who owns the collector port before starting a new Local Collector app.

Idempotent setup/update rule:

- The setup script must be safe to run again at any time.
- Re-running the setup script must not overwrite or delete current client data, collected data, reports, history, schedules, or an existing `collector_config.json`.
- Re-running the setup script may download and replace the Local Collector app executable files when a newer distribution is available.
- The setup script should download new executable archives to a temporary file first and compare them with the existing downloaded archive. It should replace/extract executable files only when the archive changed or local executable files are missing.
- Re-running the setup script should install the Solo Agency Local Collector extension files only if the local extension folder is missing or incomplete. It should not silently replace an already installed unpacked extension folder during a routine bridge update.
- If `collector_config.json` is missing, the setup script should create a default one.
- If `collector_config.json` already exists, the setup script must keep it unchanged. Schedule changes should be made by editing that config intentionally or by calling `POST /config`, not by re-running setup.
- The setup script must ensure the Local Collector app is restarted so the newest executable is used.
- The setup script should start the Local Collector app in the background/detached mode, write PID/log files, and then return control to the human. It should not require the human to keep Terminal or PowerShell open for normal operation.
- Foreground mode is allowed only for explicit troubleshooting/debugging.
- The setup script should keep PID/log files under `solo-agency-local-collector/`, for example `solo-agency-local-collector/collector.pid` and `solo-agency-local-collector/collector.log`.
- Before starting the Local Collector app, the setup script must detect and restart any previous Local Collector app process for port `17321` when it can do so safely.
- Re-running the setup script must not leave an older Local Collector app holding port `17321`. If an old collector keeps the port, the Chrome extension may keep talking to stale config and report `no job` even after the AI agent wrote new client sources.
- The restart order must be: stop the PID in `collector.pid` if alive, inspect the process holding port `17321`, kill only collector processes such as `collector-bridge`, then start the newest executable and write a fresh PID/log. If a non-collector process owns the port, stop and show the human the blocking command instead of killing unrelated software. Do not rely on `POST /shutdown` to stop the bridge: in the shipped binary `/shutdown` requires the per-run extension token (held only by the extension) and returns 401 when called tokenless, so a tokenless call is a no-op. Use the PID-based/port-based stop instead.
- The setup script must not simply run the bridge and hope the port is free. If the new bridge logs `address already in use`, the setup script is incomplete and must be fixed before asking the human to retry.
- The setup script must not delete `daily-content-pipeline/collector/inbox/`, `daily-content-pipeline/clients/`, `history/`, `outputs/`, or reports.
- The AI agent should generate the setup script from the templates below by replacing only the absolute path placeholders and, when needed, artifact version URLs.

macOS/Linux:

The AI agent must create this file with the real absolute path filled in:

```text
/ABSOLUTE/PATH/TO/solo-agency-local-collector/setup_collector.sh
```

The file content should be the following. This is an internal implementation template for the AI agent; do not show this long file content to the human as the primary setup instruction.

```bash
#!/usr/bin/env bash
set -euo pipefail

AGENCY_ROOT="/ABSOLUTE/PATH/TO"
PIPELINE_ROOT="$AGENCY_ROOT/daily-content-pipeline"
COLLECTOR_RUNTIME_ROOT="$AGENCY_ROOT/solo-agency-local-collector"
COLLECTOR_DATA_ROOT="$PIPELINE_ROOT/collector"
BASE_URL="https://raw.githubusercontent.com/soloagency/solo-agency/main/solo-agency-collector"
BRIDGE_ZIP_URL="$BASE_URL/dist/collector-bridge-binaries-0.1.0.zip"
PORT="17321"
CONFIG_FILE="$COLLECTOR_DATA_ROOT/collector_config.json"
PID_FILE="$COLLECTOR_RUNTIME_ROOT/collector.pid"
LOG_FILE="$COLLECTOR_RUNTIME_ROOT/collector.log"

mkdir -p "$COLLECTOR_RUNTIME_ROOT/downloads" "$COLLECTOR_RUNTIME_ROOT/bin" "$COLLECTOR_DATA_ROOT/inbox" "$COLLECTOR_DATA_ROOT/jobs/pending" "$COLLECTOR_DATA_ROOT/logs"

echo "Downloading or updating the Local Collector app file..."
BRIDGE_ZIP="$COLLECTOR_RUNTIME_ROOT/downloads/collector-bridge-binaries-0.1.0.zip"
BRIDGE_ZIP_TMP="$BRIDGE_ZIP.tmp"
curl -L -o "$BRIDGE_ZIP_TMP" "$BRIDGE_ZIP_URL"
if [ ! -f "$BRIDGE_ZIP" ] || ! cmp -s "$BRIDGE_ZIP_TMP" "$BRIDGE_ZIP" || ! ls "$COLLECTOR_RUNTIME_ROOT/bin"/collector-bridge-* >/dev/null 2>&1; then
  echo "Installing updated Local Collector app executable files..."
  mv "$BRIDGE_ZIP_TMP" "$BRIDGE_ZIP"
  unzip -o "$BRIDGE_ZIP" -d "$COLLECTOR_RUNTIME_ROOT/bin"
else
  echo "Local Collector app executable files are already up to date."
  rm -f "$BRIDGE_ZIP_TMP"
fi

echo "Per-client Chrome extension folders are managed separately under $AGENCY_ROOT/extensions/{client_slug}_extension."

if [ ! -f "$CONFIG_FILE" ]; then
  echo "Creating default collector_config.json..."
  cat > "$CONFIG_FILE.tmp" <<'JSON'
{
  "version": "0.1.0",
  "timezone": "local",
  "run_mode": "persistent_bridge_scheduler",
  "routing_mode": "shared_bridge_parallel_per_client_extension",
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
JSON
  mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"
else
  echo "Keeping existing collector_config.json unchanged."
fi

OS="$(uname -s)"
ARCH="$(uname -m)"
case "$OS/$ARCH" in
  Darwin/arm64) BRIDGE="$COLLECTOR_RUNTIME_ROOT/bin/collector-bridge-darwin-arm64" ;;
  Darwin/x86_64) BRIDGE="$COLLECTOR_RUNTIME_ROOT/bin/collector-bridge-darwin-amd64" ;;
  Linux/x86_64) BRIDGE="$COLLECTOR_RUNTIME_ROOT/bin/collector-bridge-linux-amd64" ;;
  *) echo "Unsupported OS/CPU: $OS/$ARCH"; exit 1 ;;
esac
chmod +x "$BRIDGE"

stop_existing_bridge() {
  # Do not call POST /shutdown: the shipped bridge requires the per-run
  # extension token and returns 401 when called tokenless, so it is a no-op.
  # Stop the bridge by PID first, then by the process holding the port.

  if [ -f "$PID_FILE" ]; then
    OLD_PID="$(cat "$PID_FILE" 2>/dev/null || true)"
    if [ -n "$OLD_PID" ] && kill -0 "$OLD_PID" 2>/dev/null; then
      echo "Stopping previous Local Collector app process: $OLD_PID"
      kill "$OLD_PID" 2>/dev/null || true
      sleep 2
      if kill -0 "$OLD_PID" 2>/dev/null; then
        kill -9 "$OLD_PID" 2>/dev/null || true
      fi
    fi
    rm -f "$PID_FILE"
  fi

  if command -v lsof >/dev/null 2>&1; then
    for PID in $(lsof -tiTCP:"$PORT" -sTCP:LISTEN 2>/dev/null || true); do
      CMD="$(ps -p "$PID" -o command= 2>/dev/null || true)"
      case "$CMD" in
        *collector-bridge*)
          echo "Stopping old Local Collector app process using port $PORT: $PID"
          kill "$PID" 2>/dev/null || true
          sleep 1
          if kill -0 "$PID" 2>/dev/null; then
            kill -9 "$PID" 2>/dev/null || true
          fi
          ;;
        *)
          echo "Port $PORT is used by a non-collector process:"
          echo "$CMD"
          echo "Please stop that process or choose another collector port."
          exit 1
          ;;
      esac
    done
  fi
}

stop_existing_bridge

echo "Local Collector app setup is ready."
echo "Client-specific Chrome extension folders are prepared separately under: $AGENCY_ROOT/extensions/{client_slug}_extension/"
echo "Do NOT load any chrome-extension folder under solo-agency/solo-agency-collector; that is the toolkit/source copy."
echo "One machine should have one active shared Solo Agency Local Collector runtime for the current setup."
echo "Starting the Local Collector app in the background with the newest executable."
nohup "$BRIDGE" --host 127.0.0.1 --port "$PORT" --config-file "$CONFIG_FILE" --output-dir "$COLLECTOR_DATA_ROOT/inbox" --persistent >> "$LOG_FILE" 2>&1 &
BRIDGE_PID="$!"
echo "$BRIDGE_PID" > "$PID_FILE"
echo "Local Collector app started. PID: $BRIDGE_PID"
echo "Log file: $LOG_FILE"
echo "You can close this Terminal window now."
```

**Local runtime:** run this script yourself. Say the one-line safety confirmation, ask consent once, run it, then poll `/status` for up to 60 seconds:

```md
Em đã đọc mã của collector: nó chỉ chạy trên máy này, không gửi dữ liệu đi đâu. Em cài và bật nó ngay bây giờ, khoảng một phút.
```

Then trigger the extension's two-gesture install (`POST /api/ui/{client_slug}/install-extension`, no `browser`/`profile_directory` fields — or point the human at `/ui/{client_slug}/extension` if you prefer they do it themselves), poll `extension_health.status` until recent, and report both as done — no Terminal, no `bash` command, ever shown to this human.

**Remote runtime only:** tell the human only this one-line command, with the real absolute path:

```bash
bash "/ABSOLUTE/PATH/TO/solo-agency-local-collector/setup_collector.sh"
```

Human-facing wording for the required two-step setup (remote runtime — a local runtime performs both steps itself instead, per above):

```md
I created the Local Collector setup file. Please do these two one-time steps:

Step 1 - start the Local Collector app outside the AI agent sandbox.
Open Terminal, paste this one line, and press Enter:

`bash "/ABSOLUTE/PATH/TO/solo-agency-local-collector/setup_collector.sh"`

**Easiest path (recommended) — the page shows everything you need.** Give the human this one link:
`http://127.0.0.1:17321/ui/{client_slug}/extension` — it plays the install video and shows the two steps plus the extension folder's absolute path (with a Copy button). Open your own browser's extensions page, turn on **Developer mode**, then **drag that folder onto the extensions page**. The page flips to a green ✓ connected on its own when the extension checks in — no path to remember, no file picker, no second tab to find yourself. The absolute-path instructions below are the same information, spelled out as plain steps.

Step 2 (manual fallback) - load the client-specific extension into whichever browser window you already use for Facebook.
Open your browser's extensions page (`chrome://extensions`, or `edge://extensions` on Edge) -> turn on Developer mode -> Load unpacked -> select this folder:

`/ABSOLUTE/PATH/TO/extensions/{client_slug}_extension/`

Important: do not select any `solo-agency/solo-agency-collector/chrome-extension` folder. Use only the client folder under `extensions/{client_slug}_extension/`.

After both steps are done, tell me "done". Then I will check the Local Collector status and continue.
```

Windows:

Do not tell Windows users to run bash. On Windows, use PowerShell or create a `.cmd` launcher that the human can double-click.

Important Windows note:

- The human can run an `.exe`, but double-clicking `collector-bridge-windows-amd64.exe` by itself is not enough for the recommended persistent setup because the app needs configuration arguments.
- The AI agent should create a friendly launcher such as `Start Local Collector.cmd` and, if needed, a setup script such as `setup_local_collector.ps1`.
- Same local-vs-remote rule as macOS/Linux: on a LOCAL runtime the agent runs `setup_local_collector.ps1` itself (one safety line, one consent ask, then execute — Microsoft SmartScreen may ask for one click on the unsigned script/binary; mention that only if it happens), then triggers the extension's two-gesture install and polls `extension_health.status`. On a REMOTE runtime, the human-facing instruction must include both setup actions: first run the prepared PowerShell command or double-click `Start Local Collector.cmd` outside the agent's sandbox, then load the Chrome extension from the absolute runtime extension folder (or the one-button dashboard flow).
- If the human wants it to run after restart, use Windows Task Scheduler with "At log on" — though the canonical setup scripts already register this by default (see OS Startup For Persistent Bridge).

PowerShell setup script file path:

```text
C:\ABSOLUTE\PATH\TO\solo-agency-local-collector\setup_local_collector.ps1
```

PowerShell setup script content, with `AgencyRoot` replaced by the real absolute parent path that contains both `daily-content-pipeline` and `solo-agency-local-collector`, for example `C:\Users\Alex\oneman_agency`. This is an internal implementation template for the AI agent; do not show this long file content to the human as the primary setup instruction:

```powershell
$ErrorActionPreference = "Stop"
$AgencyRoot = "C:\ABSOLUTE\PATH\TO"
$PipelineRoot = Join-Path $AgencyRoot "daily-content-pipeline"
$CollectorRuntimeRoot = Join-Path $AgencyRoot "solo-agency-local-collector"
$CollectorDataRoot = Join-Path $PipelineRoot "collector"
$BaseUrl = "https://raw.githubusercontent.com/soloagency/solo-agency/main/solo-agency-collector"
$BridgeZipUrl = "$BaseUrl/dist/collector-bridge-binaries-0.1.0.zip"
$Port = 17321
$ConfigPath = Join-Path $CollectorDataRoot "collector_config.json"
$PidPath = Join-Path $CollectorRuntimeRoot "collector.pid"
$LogPath = Join-Path $CollectorRuntimeRoot "collector.out.log"
$ErrLogPath = Join-Path $CollectorRuntimeRoot "collector.err.log"

New-Item -ItemType Directory -Force -Path `
  (Join-Path $CollectorRuntimeRoot "downloads"), `
  (Join-Path $CollectorRuntimeRoot "bin"), `
  (Join-Path $CollectorDataRoot "inbox"), `
  (Join-Path $CollectorDataRoot "jobs\pending"), `
  (Join-Path $CollectorDataRoot "logs") | Out-Null

Write-Host "Downloading or updating the Local Collector app file..."
$BridgeZipTmp = Join-Path $CollectorRuntimeRoot "downloads\collector-bridge-binaries-0.1.0.zip.tmp"
$BridgeZip = Join-Path $CollectorRuntimeRoot "downloads\collector-bridge-binaries-0.1.0.zip"
Invoke-WebRequest -Uri $BridgeZipUrl -OutFile $BridgeZipTmp
$ExistingBridgeFiles = Get-ChildItem -Path (Join-Path $CollectorRuntimeRoot "bin") -Filter "collector-bridge-*" -ErrorAction SilentlyContinue
$BridgeNeedsInstall = (-not (Test-Path $BridgeZip)) -or (-not $ExistingBridgeFiles)
if (-not $BridgeNeedsInstall) {
  $OldHash = (Get-FileHash $BridgeZip -Algorithm SHA256).Hash
  $NewHash = (Get-FileHash $BridgeZipTmp -Algorithm SHA256).Hash
  $BridgeNeedsInstall = ($OldHash -ne $NewHash)
}
if ($BridgeNeedsInstall) {
  Write-Host "Installing updated Local Collector app executable files..."
  Move-Item -Force $BridgeZipTmp $BridgeZip
  Expand-Archive -Force $BridgeZip (Join-Path $CollectorRuntimeRoot "bin")
} else {
  Write-Host "Local Collector app executable files are already up to date."
  Remove-Item $BridgeZipTmp -Force -ErrorAction SilentlyContinue
}

Write-Host "Per-client Chrome extension folders are managed separately under $AgencyRoot\extensions\{client_slug}_extension."

if (-not (Test-Path $ConfigPath)) {
  Write-Host "Creating default collector_config.json..."
  @'
{
  "version": "0.1.0",
  "timezone": "local",
  "run_mode": "persistent_bridge_scheduler",
  "routing_mode": "shared_bridge_parallel_per_client_extension",
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
'@ | Set-Content -Encoding UTF8 $ConfigPath
} else {
  Write-Host "Keeping existing collector_config.json unchanged."
}

$Bridge = Join-Path $CollectorRuntimeRoot "bin\collector-bridge-windows-amd64.exe"

# Do not call POST /shutdown: the shipped bridge requires the per-run
# extension token and returns 401 when called tokenless, so it is a no-op.
# Stop the bridge by PID first, then by the process holding the port.

if (Test-Path $PidPath) {
  $OldPid = Get-Content $PidPath -ErrorAction SilentlyContinue | Select-Object -First 1
  if ($OldPid) {
    $OldProcess = Get-Process -Id $OldPid -ErrorAction SilentlyContinue
    if ($OldProcess) {
      Write-Host "Stopping previous Local Collector app process: $OldPid"
      Stop-Process -Id $OldPid -Force -ErrorAction SilentlyContinue
    }
  }
  Remove-Item $PidPath -Force -ErrorAction SilentlyContinue
}

try {
  Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction Stop |
    Select-Object -ExpandProperty OwningProcess -Unique |
    ForEach-Object {
      $ProcOnPort = Get-Process -Id $_ -ErrorAction SilentlyContinue
      $ProcPath = ""
      try { $ProcPath = $ProcOnPort.Path } catch {}
      if ($ProcOnPort -and (($ProcOnPort.ProcessName -like "*collector-bridge*") -or ($ProcPath -like "*collector-bridge*"))) {
        Write-Host "Stopping old Local Collector app process using port $Port: $_"
        Stop-Process -Id $_ -Force -ErrorAction SilentlyContinue
      } else {
        Write-Host "Port $Port is used by a non-collector process:"
        if ($ProcOnPort) { Write-Host "$($ProcOnPort.ProcessName) $ProcPath" }
        Write-Host "Please stop that process or choose another collector port."
        exit 1
      }
    }
} catch {
  # Get-NetTCPConnection may not be available in older Windows environments. Continue.
}

Write-Host "Local Collector app setup is ready."
Write-Host "Client-specific Chrome extension folders are prepared separately under: $AgencyRoot\extensions\{client_slug}_extension\"
Write-Host "Do NOT load any chrome-extension folder under solo-agency\solo-agency-collector; that is the toolkit/source copy."
Write-Host "One machine should have one active shared Solo Agency Local Collector runtime for the current setup."
Write-Host "Starting the Local Collector app in the background with the newest executable."
$Args = @(
  "--host", "127.0.0.1",
  "--port", "$Port",
  "--config-file", $ConfigPath,
  "--output-dir", (Join-Path $CollectorDataRoot "inbox"),
  "--persistent"
)
$Proc = Start-Process -FilePath $Bridge -ArgumentList $Args -RedirectStandardOutput $LogPath -RedirectStandardError $ErrLogPath -WindowStyle Hidden -PassThru
Set-Content -Encoding ASCII -Path $PidPath -Value $Proc.Id
Write-Host "Local Collector app started. PID: $($Proc.Id)"
Write-Host "Log files: $LogPath and $ErrLogPath"
Write-Host "You can close this PowerShell window now."
```

For the Local Collector app setup action, tell the human one short PowerShell command:

```powershell
powershell -ExecutionPolicy Bypass -File "C:\ABSOLUTE\PATH\TO\solo-agency-local-collector\setup_local_collector.ps1"
```

Windows `.cmd` launcher file path:

```text
C:\ABSOLUTE\PATH\TO\solo-agency-local-collector\Start Local Collector.cmd
```

Windows `.cmd` launcher content. This is an internal implementation template for the AI agent; do not show this long file content to the human as the primary setup instruction:

```bat
@echo off
set "AGENCY_ROOT=C:\ABSOLUTE\PATH\TO"
set "PIPELINE_ROOT=%AGENCY_ROOT%\daily-content-pipeline"
set "COLLECTOR_RUNTIME_ROOT=%AGENCY_ROOT%\solo-agency-local-collector"
set "COLLECTOR_DATA_ROOT=%PIPELINE_ROOT%\collector"
set "PID_FILE=%COLLECTOR_RUNTIME_ROOT%\collector.pid"
set "LOG_FILE=%COLLECTOR_RUNTIME_ROOT%\collector.out.log"
set "ERR_LOG_FILE=%COLLECTOR_RUNTIME_ROOT%\collector.err.log"
REM Do not call POST /shutdown: the shipped bridge requires the per-run extension token and returns 401 when called tokenless, so it is a no-op. Stop by PID first, then by the process holding the port.
powershell -NoProfile -ExecutionPolicy Bypass -Command "if (Test-Path '%PID_FILE%') { $p = Get-Content '%PID_FILE%' -ErrorAction SilentlyContinue | Select-Object -First 1; if ($p) { Stop-Process -Id $p -Force -ErrorAction SilentlyContinue }; Remove-Item '%PID_FILE%' -Force -ErrorAction SilentlyContinue }; try { Get-NetTCPConnection -LocalPort 17321 -State Listen -ErrorAction Stop | Select-Object -ExpandProperty OwningProcess -Unique | ForEach-Object { $proc = Get-Process -Id $_ -ErrorAction SilentlyContinue; $path = ''; try { $path = $proc.Path } catch {}; if ($proc -and (($proc.ProcessName -like '*collector-bridge*') -or ($path -like '*collector-bridge*'))) { Stop-Process -Id $_ -Force -ErrorAction SilentlyContinue } else { Write-Host ('Port 17321 is used by a non-collector process: ' + $proc.ProcessName + ' ' + $path); exit 1 } } } catch {}"
if errorlevel 1 exit /b 1
powershell -NoProfile -ExecutionPolicy Bypass -Command "$p = Start-Process -FilePath '%COLLECTOR_RUNTIME_ROOT%\bin\collector-bridge-windows-amd64.exe' -ArgumentList @('--host','127.0.0.1','--port','17321','--config-file','%COLLECTOR_DATA_ROOT%\collector_config.json','--output-dir','%COLLECTOR_DATA_ROOT%\inbox','--persistent') -RedirectStandardOutput '%LOG_FILE%' -RedirectStandardError '%ERR_LOG_FILE%' -WindowStyle Hidden -PassThru; Set-Content -Encoding ASCII -Path '%PID_FILE%' -Value $p.Id; Write-Host ('Local Collector app started. PID: ' + $p.Id); Write-Host 'You can close this window now.'"
```

**Local runtime:** run `setup_local_collector.ps1` yourself (safety line, one consent ask, execute — note a SmartScreen click only if it actually appears), then trigger the extension's two-gesture install and poll `extension_health.status`. Report both as done.

Human-facing Windows wording for the required two-step setup (remote runtime only — a local runtime performs both steps itself instead, per above):

```md
I created the Local Collector setup file. Please do these two one-time steps:

Step 1 - start the Local Collector app outside the AI agent sandbox.
Open PowerShell, paste this one line, and press Enter:

`powershell -ExecutionPolicy Bypass -File "C:\ABSOLUTE\PATH\TO\solo-agency-local-collector\setup_local_collector.ps1"`

**Easiest path (recommended) — the page shows everything you need.** Give the human this one link:
`http://127.0.0.1:17321/ui/{client_slug}/extension` — it plays the install video and shows the two steps plus the extension folder's absolute path (with a Copy button). Open your own browser's extensions page, turn on **Developer mode**, then **drag that folder onto the extensions page**. The page turns green connected on its own when the extension checks in — no path to remember, no file picker, no second tab to find yourself. The absolute-path instructions below are the same information, spelled out as plain steps.

Step 2 (manual fallback) - load the client-specific extension into whichever browser window you already use for Facebook.
Open your browser's extensions page (`chrome://extensions`, or `edge://extensions` on Edge) -> turn on Developer mode -> Load unpacked -> select this folder:

`C:\ABSOLUTE\PATH\TO\extensions\{client_slug}_extension\`

Important: do not select any `solo-agency\solo-agency-collector\chrome-extension` folder. Use only the client folder under `extensions\{client_slug}_extension\`.

After both steps are done, tell me "done". Then I will check the Local Collector status and continue.

Later, if you need to start the Local Collector app manually again, double-click:
`C:\ABSOLUTE\PATH\TO\solo-agency-local-collector\Start Local Collector.cmd`
```

Future update rule:

- When the project moves from raw GitHub files to GitHub Releases, replace the raw artifact URLs with GitHub release URLs.
- The extension stays an unpacked, per-client folder for now (no Chrome Web Store submission) — see "One Shared Bridge, Many Client Extensions" for the two-gesture install this implies.
- The AI agent should handle download/extraction/script preparation automatically when possible. On a LOCAL runtime it also performs both one-time local actions itself (bridge setup/start, extension install) with the one consent ask; only on a REMOTE runtime does the human perform them outside the agent's sandbox.

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

### A private source names the capability that reads it

Each entry under `clients[].private_sources[]` in `collector_config.json` carries a `capability`
alongside `name`, `url` and `platform`. The bridge has always read it — the plan gate and the
healthcheck both key off it — and it decides HOW the source is read:

| capability | what it does | when it is the right one |
|---|---|---|
| `fb.group.posts` | scrolls a group's feed | the default for a watched group: what is new, what the community is discussing |
| `fb.group.search_posts` | searches inside one group for a term | the daily search pass (`tool source-keywords urls` emits these, named `kw:<term>`) |
| `fb.profile.posts` | scrolls one profile or page timeline | watching a competitor or a specific person |
| `fb.post.comments` | reads a post's comments by `feedback_id` | the people replying to "looking for an agent" are often the better leads |

An entry with no `capability` is read as a feed. A source registered for the search pass carries the
whole search url in `inputs.group_search_url`; the `url` field stays the plain group url so the same
source is recognisable across passes.

### Social Discovery Pass job shapes — Facebook leg (fb.search.posts, fb.people.search, fb.groups.search, fb.group.search_posts)

`playbooks/10_LEAD_COMPETITOR_DETECTION.md`'s Social Discovery Pass (step 11C) runs four
capabilities from `collector_capabilities.json`, in fixed order. All four are read-only, run
through the same collector chain as every capability in this file, and are bounded by the budget
table in Stage 10 — nothing here raises `max_pages` or scroll depth beyond that table.

1. **`fb.search.posts`** (status `beta`) — Facebook's own global Posts-tab search.
   `inputs.search_url = https://www.facebook.com/search/posts/?q=<url-encoded discovery term>`
   (or `/search/top/?q=...` for the mixed Top tab). Output: `PostRecord[]`, the same shape as
   `fb.group.posts`/`fb.group.search_posts`. No group membership required — this is the human's own
   logged-in Facebook session searching the open Posts tab.
2. **`fb.people.search`** (status `stable`) — Facebook's own People-tab search.
   `inputs.query = <discovery term>` (the url Facebook itself renders:
   `https://www.facebook.com/search/people/?q=<url-encoded discovery term>`); `max_scroll` and
   `max_pages` are both optional, same ceiling as every other capability in this pass. Output:
   `ProfileSummary[]` — `id`, `name`, `url`, `subtitle`, `mutual_friends`, `industry_hint`: row-level
   summaries, not full profiles, and there is no post text to read. Stage 10 classifies each row from
   `subtitle` + `industry_hint` + `name`/`url` alone.
3. **`fb.groups.search`** — `inputs.query = <discovery term>`. Output: `GroupSummary[]`, one row per
   candidate group:
   - `privacy`: `"public"` | `"private"` | `""` (empty = unknown — never treat empty as private OR
     public; see Stage 10's ranking rule);
   - `member_count`: number | `null` (an unrecognized locale magnitude token leaves this `null`);
   - `viewer_join_state`, `snippet`, `privacy_source` (which signal decided `privacy`:
     `viewer_join_state` or the parsed descriptor line), `member_count_text` (the raw matched
     segment), `snippet_lang` (the descriptor line's UI locale). `viewer_join_state: "MEMBER"` means
     the account already belongs and can read the group — it counts as `groups_readable` alongside
     public groups; `viewer_join_state: "CAN_REQUEST"` or `"REQUEST_TO_JOIN"` means the account has no
     access — it counts as `groups_no_access` and is never scanned or joined.
   `privacy`/`member_count` can legitimately come back unknown/null on any given result — that is a
   parsing limitation of Facebook's search card, not a job failure. Stage 10's rule: unknown privacy
   is resolved with one `fb.group.posts` (`max_pages: 1`) header check that decides `readable` (posts
   come back) or `no_access` (an access wall, or empty with `stopped_because` naming access or login)
   — never assumed readable, never assumed no-access without the check, and the group is skipped only
   when the call budget is tight.
4. **`fb.group.search_posts`** — `inputs.group_search_url = <group_url>/search/?q=<url-encoded
   intent term>`. Output: `PostRecord[]`. This is the same capability the daily search pass (§"A
   private source names the capability that reads it" above) already uses for monitored sources; the
   discovery pass points it at groups that are not (yet) in `private_data_sources`.

All four respect `max_pages` (default 8, hard cap 40) and `max_scroll`; the discovery pass caps
`max_pages` at 4 regardless of the capability default (Stage 10 budget table).

**Group registry, not a shortlist file.** The former per-month group shortlist file is
retired. The discovery pass registers its candidate groups directly in the source registry
(`collector/source_registry.json`, `source_type: group`) via `tool source-registry add` — field
definitions (`state`, `potential`, `potential_reason`, `scans`, `leads_total`, `leads_recent`,
`last_scanned_at`, `paused_at`, `origin`) live in `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`,
"Monitored Facebook groups". This is Facebook-only — Instagram and X have no group concept in the
Social Discovery Pass, so their legs never register `source_type: group` entries. `tool
source-registry plan --client <slug> --platform facebook --max 20` decides which groups this run
scans and in what order (most leads across their last 3 scans first, then never-scanned newest
first, then longest-unscanned, ties by member count) — the plan itself is what keeps the DAILY
companion pass from rediscovering the same handful of groups every day; `tool source-registry
record --client <slug> --run <run_id> --url <group_url> --leads <n>` after each scan is what
re-ranks it for next time.

### First action once the bridge exists: settle any deferred slot check

The bridge carries `tool schedule-slots`, and Setup Flow creates the first automation task before it
is installed (Stage 4's first-task exception). So the moment the human has run the setup script and
`/status` answers, check `daily-content-pipeline/automation/automation_manifest.md` for
`slot_check_pending: true`: register that task (`tools/solo_tool schedule-slots register --task ...
--client ... --time ... --cadence-hours ...`), then set the flag to `false` with the resync
timestamp. Do this BEFORE any other `suggest`, or the next task is timed against a registry that is
missing the one task already running. A manifest still carrying the flag after the bridge is up is
an audit finding (`playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md`).

### Private Collector Health Check Protocol

Before every scheduled run, after every scheduled run, and whenever private data is missing, the AI agent must check the private collector health.

Health check sequence:

1. Try `GET http://127.0.0.1:17321/status`.
2. If the request succeeds:
   - record `bridge_status: running`,
   - record `status.persistent`,
   - record `status.job_available`,
   - record `status.output_dir`,
   - record `status.config_file`,
   - record `status.run_now_request_file`,
   - record `status.counts`,
   - inspect `status.extension_health`.
3. Before treating the bridge as healthy, run the current-workspace identity check:
   - expected `config_file`: `{current_setup_root}/daily-content-pipeline/collector/collector_config.json`;
   - expected `output_dir`: `{current_setup_root}/daily-content-pipeline/collector/inbox` or a run folder under that inbox;
   - expected `run_now_request_file`: `{current_setup_root}/daily-content-pipeline/collector/run_now_request.json`.
4. Normalize paths when possible before comparing. Prefer absolute paths from `/status`; if a path is relative or ambiguous, do not assume it matches unless it clearly resolves under the current setup root.
5. If the bridge is running but any of those paths point to another setup folder, mark the collector as `wrong_workspace_bridge`, not healthy. Do not create run-now jobs, do not write `run_now_request.json`, and do not claim private data source monitoring is active.
6. For `wrong_workspace_bridge`, tell the human plainly:
   - a Local Collector app is already running, but it belongs to a previous Solo Agency setup or another folder;
   - one machine should have only one active Solo Agency Local Collector runtime for the current setup;
   - on a LOCAL runtime, the agent re-runs the current setup's script itself (one consent ask) so it can stop the old `collector-bridge` process and start the bridge with the current workspace paths; on a REMOTE runtime, the human runs the current setup's one-line Local Collector command outside the agent's sandbox to do the same;
   - if the human has loaded old Solo Agency Local Collector extensions in Chrome, they should open `chrome://extensions`, remove or disable stale entries from previous setup folders, and keep only the current client-specific extensions loaded from this setup's absolute `extensions/{client_slug}_extension/` folders. Multiple current extensions are expected when multiple clients use different Chrome profiles/accounts.
7. If the workspace identity check passes and `extension_health.status` is `recent`, private collection infrastructure is currently healthy.
8. If the workspace identity check passes and `extension_health.status` is `no_extension_check_yet` immediately after extension install, bridge restart, or settings save, wait and re-check for up to 75 seconds before declaring private collection unavailable.
9. If the workspace identity check passes and `extension_health.status` is `stale` or `no_extension_check_yet` after the 75-second grace window, treat private collection as unavailable for now and identify likely causes:
   - Chrome is closed,
   - extension is not installed,
   - extension is disabled or removed,
   - Solo Agency Local Collector extension and Local Collector app URL/port mismatch,
   - Chrome service worker is asleep and has not woken recently,
   - browser profile is not the one where the extension was installed.
10. If `/status` fails:
   - record `bridge_status: offline`,
   - do not try to start the bridge from inside the AI agent sandbox during setup/repair,
   - provide the human with the absolute-path Local Collector app setup/start command,
   - continue with public data sources and previously collected private data.
11. If the bridge is running but the extension is stale, do not keep retrying aggressively. Continue with public data sources, log the private data source blocker, and notify the human.
12. If the extension is recent but a private data source fails due to login/captcha/checkpoint/session expiry, skip that source, log the platform-specific issue, and notify the human.

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

Wrong workspace example:

```md
Agent: Claude Schedule
Collector status: wrong_workspace_bridge
Running bridge config: /Users/alex/old_setup/daily-content-pipeline/collector/collector_config.json
Current setup config: /Users/alex/oneman_agency/daily-content-pipeline/collector/collector_config.json
Impact: I cannot use this bridge for today's private data source scan because it may write data into the old setup folder.
Action: Please run the Local Collector setup/start command for the current setup outside the AI sandbox. If you previously loaded old Solo Agency Local Collector extensions, open chrome://extensions and remove or disable stale entries from old setup folders. Keep the current client-specific extension loaded from /Users/alex/oneman_agency/extensions/{client_slug}_extension/ in the matching client's Chrome profile.
```

### Beyond Liveness: Active Capability Probes

The protocol above proves the bridge and the extension talk to each other. It does not prove
that any capability still reads Facebook or Zillow correctly — a renamed GraphQL query or a
relabelled composer leaves `/status` saying `recent` while every scan comes back empty. For
that, load `playbooks/HEALTHCHECK.md`: `<bridge> tool healthcheck run --client {slug}` re-runs
every catalog capability against operator-owned fixtures and reports which one broke and what
to open. The liveness check here is its precondition, never its replacement.

### OS Startup For Persistent Bridge

The canonical setup scripts (`setup_collector.sh` / `setup_collector.ps1`, 2026-07-20+) register OS autostart BY DEFAULT when the human runs them: macOS per-user LaunchAgent `com.solo-agency.collector.{insthash}` (RunAtLoad; restarts on crash, stays stopped after a clean stop), Linux systemd user unit `solo-agency-collector-{insthash}.service` (`Restart=on-failure`, best-effort `loginctl enable-linger` for start-before-login), Windows logon Scheduled Task `SoloAgencyCollector-{insthash}` (crash restarts, output wrapped into `collector.log`). `{insthash}` = first 8 hex chars of SHA-256 of the agency root path, so two installs on one machine never collide. The human opts out with `SOLO_AGENCY_NO_AUTOSTART=1` (plain background start). Re-running the script refreshes the registration and remains the ONLY start/upgrade command the human ever needs. After a reboot the bridge should be up without anyone running anything.

Every setup run records the outcome in `solo-agency-local-collector/autostart.json`: `{"mode": "launchd"|"systemd"|"scheduled_task"|"none", "label", "port", "root", "registered_at", "reason"}`. This file is the canonical evidence for agents — it lives inside the workspace, so even a sandboxed agent can read it when OS commands and `~/Library`/`~/.config` are out of reach.

**Agent duty — verify autostart after any collector setup/upgrade, and FIRST when diagnosing a bridge that is down after a reboot. Pick the deepest rung you can actually perform; never assume and never silently skip:**

1. Agent can run OS commands (local, unsandboxed): verify directly — macOS `launchctl print gui/$(id -u)/com.solo-agency.collector.{insthash}`; Linux `systemctl --user is-enabled solo-agency-collector-{insthash}.service`; Windows `Get-ScheduledTask -TaskName SoloAgencyCollector-{insthash}`.
2. Sandboxed agent: read `solo-agency-local-collector/autostart.json`. `mode` other than `"none"` = registered (report which). `mode: "none"` = autostart is OFF — tell the human why (`reason`: `opt_out_env`, `*_registration_failed`, `no_supervisor_available`) and that the fix is re-running the setup script. File missing = the install predates the autostart scripts — ask the human to re-run the current setup script once.
3. Agent can neither run commands nor read the file: give the human the ONE copy-paste command for their OS from rung 1 and ask them to paste the output back.

Guardrails: never hand-craft a custom LaunchAgent/systemd unit/Scheduled Task beyond what the script registers — the fix for a missing or broken registration is always re-running the setup script (one command, safe to re-run), never a bespoke unit. On a LOCAL runtime the agent re-runs it itself (one consent ask); the old blanket "do not install/start services from inside the AI sandbox, the HUMAN re-runs it" applies only on a REMOTE runtime now, where the fix is still handing the human that one command. Claude-specific: on a remote/hosted sandbox Claude often cannot run downloaded binaries at all; Claude must not use Claude Chrome Extension as a workaround for automated private collection; after the bridge runs as a startup service, Claude reads collector output files and continues without controlling Chrome directly.

### Uninstall / Start Over

`solo-agency-collector/uninstall_collector.sh` (`uninstall_collector.ps1` on Windows) reverses everything `setup_collector.sh`/`.ps1` did, leaving the machine as if Solo Agency had never been installed there — this is the owner-decided answer to "xóa hết, như chưa từng cài" (clear it all, as if never installed). It ships next to the setup script on the same `dist` branch and `setup_collector.sh`/`.ps1` stage a copy of it into `solo-agency-local-collector/` (the RUNTIME folder, not the source checkout) on every run, so "start over" still works even if `<root>/solo-agency` is deleted first.

What it removes, per install root: the entitlement seat (`tool entitlement release`, best effort, never blocks the rest), the OS autostart registration (`launchctl bootout` + plist / `systemctl --user stop`+`disable` + unit file / `Stop-ScheduledTask`+`Unregister-ScheduledTask`), the bridge process (PID file, then anything on the collector port — only ever a process whose command line contains `collector-bridge`), `solo-agency-local-collector/`, `extensions/`, `daily-content-pipeline/` (CRM, content library, config, secrets — all of it), the install-root `AGENTS.md`/`CLAUDE.md` pointer files (only when they still carry the `MULTI_BRAIN_OPERATIONS.md` signature line — otherwise left in place with a note), `solo-agency/` (the source checkout, unless `--keep-source`), the empty root directory itself (unless `--keep-root-dir`), and the CLI-era `~/.claude/scheduled-tasks/*solo-agency*` folders. It does not touch `<root>/outreach` — no repo-defined marker distinguishes a live install's outreach state directory from an unrelated folder at that path, so it is always left in place with a note asking a human to verify it (see `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`, `outreach/playbooks/13_CRM_CORE.md`).

Flags: `--dry-run` (print the plan, change nothing), `--yes` (skip the confirmation — this is what an agent passes, and only after the human has explicitly confirmed removal in their own words), `--root PATH` (repeatable — targets specific install roots instead of auto-discovering every one on the machine), `--keep-source`, `--keep-root-dir`, `--port N`, `--open-browser`. Safety invariants enforced in code: it refuses to touch `/`, `$HOME`/`$env:USERPROFILE`, or any root that does not contain `daily-content-pipeline/` or `solo-agency-local-collector/`; every delete target is the validated root joined with a fixed subdirectory name, never a glob or pattern; one failed step never aborts the rest (see the summary — removed / skipped / manual — at the end). Exit code 0 means everything scripted succeeded; 2 means at least one scripted step failed or a `--root` was refused.

Manual leftovers it prints but cannot remove itself: Claude desktop's own Scheduled panel entries for the install (`{client}-solo-agency-daily-run`, `solo-agency-github-update-watch` — a different registry than the CLI-era folders above), Codex automations for the install, and the unpacked Chrome extension entries under `chrome://extensions` per profile (the folders on disk are gone, but Chrome keeps the entry until removed there — `--open-browser` opens the page). `~/.claude/projects` transcripts and this AI runtime's own `CLAUDE.md` are the runtime's own data, not Solo Agency's, and are never touched.

**Local vs remote runtime rule for uninstall (mirrors the install rule above).** On a LOCAL runtime, after the human gives one explicit confirmation in their own words (e.g. "xóa hết, như chưa từng cài" / "uninstall everything") — not implied by an unrelated request — the agent runs `uninstall_collector.sh --yes` itself and reports the summary; it does not fall back to "hand the human the command" just because it could ask instead. On a REMOTE runtime, hand the human the one-line command for their OS instead of running it. Either way, always offer `--dry-run` first when the human seems unsure, and never run it without that one explicit confirmation — this is a destructive, irreversible action on client data.

### Localhost Bridge Choice

The localhost bridge should be implemented as a small cross-platform local executable.

Preferred implementation:

- Use Go for the production bridge because it can compile to small single-file binaries for macOS, Windows, and Linux without requiring Node.js, Python, or a package manager on the user's machine.
- Rust is also acceptable, but Go is the default recommendation because distribution and maintenance are simpler for this use case.
- Go is a build-time choice for maintainers, not an end-user runtime requirement. Normal users should receive prebuilt bridge binaries and should not be asked to install Go.
- Do not require the human to install Python, Node.js, Playwright, or system packages just to run the bridge.
- Ship or download platform-specific binaries, for example:
  - `collector-bridge-darwin-arm64`
  - `collector-bridge-darwin-amd64`
  - `collector-bridge-windows-amd64.exe`
  - `collector-bridge-linux-amd64`
- Store binaries under the collector runtime folder, not the data workspace:

```text
solo-agency-local-collector/bin/
```

The bridge must:

- Bind only to `127.0.0.1`, never `0.0.0.0`.
- Support two modes:
  - `agent_on_demand`: run only during a collection job and shut down after completion or timeout.
  - `persistent_bridge_scheduler`: run as a lightweight local background process and coordinate scheduled collection windows.
- Shut down automatically after the job completes or after a timeout in `agent_on_demand` mode.
- Write output files locally.
- Never ask for credentials.
- Never read browser cookies or tokens.
- Never upload private data to cloud services unless the human explicitly configures that.

### Who Starts And Stops The Localhost Bridge

When `run_mode` is `agent_on_demand`, the agent should use it only if the Local Collector app is already reachable through a human-run local process or an approved local service. The default setup should prefer `persistent_bridge_scheduler` so scheduled runs do not depend on an AI-agent-started process.

When `run_mode` is `persistent_bridge_scheduler`, the bridge should start at user login or machine startup and remain idle until a configured collection window is active.

Typical run:

1. Agent detects the operating system and CPU architecture.
2. Agent confirms the matching bridge binary exists at `solo-agency-local-collector/bin/`.
3. Agent creates a collection job file under `daily-content-pipeline/collector/jobs/pending/`.
4. The human-run local process or installed startup service already has the bridge running on `127.0.0.1`. If it is not reachable: on a LOCAL runtime the agent runs the setup/start script itself (one consent ask) and waits for `/status`; on a REMOTE runtime the agent gives the human the one-line start command to run outside its sandbox and waits.
5. Solo Agency Local Collector extension detects the bridge by polling localhost.
6. Extension fetches the job, collects visible authorized data from configured private data sources, and posts results back to the bridge.
7. Bridge writes JSONL/status/snapshot files.
8. Agent reads the files.
9. In `agent_on_demand` mode the bridge auto-shuts down on completion or timeout; in `persistent_bridge_scheduler` mode it keeps running. The agent does not stop the bridge itself.

Example bridge command shape:

```text
solo-agency-local-collector/bin/collector-bridge-darwin-arm64 \
  --host 127.0.0.1 \
  --port 17321 \
  --run-id {run_id} \
  --config-file daily-content-pipeline/collector/collector_config.json \
  --jobs-dir daily-content-pipeline/collector/jobs \
  --output-dir daily-content-pipeline/collector/inbox \
  --ttl-minutes 30
```

The exact command may differ by implementation, but the behavior must remain the same.

If the local bridge is not already running, a REMOTE-runtime agent should not assume it can safely start an on-demand localhost bridge from inside its sandbox (a LOCAL runtime just runs the setup/start script itself instead — see the Local vs remote runtime rule). On a remote runtime:

- The extension may queue a limited amount of data in extension storage until a bridge is available.
- The agent must log `collector_unavailable`.
- The agent must notify the human that the Local Collector app is not running and provide the one-line setup/start command for the human to run outside its sandbox.
- The agent should continue with public data sources and previously collected private data if available.

Important constraint:

- A Chrome extension cannot magically start a localhost server if no local process is already running. On a local runtime the agent itself is that local process's starter; on a remote runtime, only the human-run setup command, an OS startup service, or another local scheduler can start the Local Collector app.

### Solo Agency Local Collector Extension Behavior

The human installs the Solo Agency Local Collector extension once in the Chrome browser/profile where they are already logged in to the relevant social platforms.

Important browser reality:

- Chrome Manifest V3 background service workers are not guaranteed to stay awake continuously.
- Do not rely on `alert()` or fake UI prompts to prevent browser sleep; background service workers do not have a reliable visible alert context and this is not a dependable automation strategy.
- Use `chrome.alarms` as the durable wake-up mechanism while Chrome is running.
- Use a short in-memory poll loop only while the service worker is awake.
- If Chrome is closed, the computer is asleep, the extension is disabled/removed, or the browser profile is not running, the extension cannot collect private data.
- In those cases, the bridge/agent must mark private collection as temporarily unavailable, continue with public data sources and previously collected private data, and notify the human through the configured provider notification channel when available, preferably WideCast OpenAPI Telegram/email fallback for the current client.

The extension should:

- Use the existing logged-in Chrome session.
- Require no passwords, cookies, tokens, OTPs, or credential sharing.
- Stay idle when the localhost bridge is not running.
- Check a small localhost status endpoint such as `http://127.0.0.1:17321/status`.
- Check the Local Collector app immediately after extension install, browser startup, and settings save.
- In persistent scheduler mode, check the bridge every `poll_interval_seconds`, default 5 seconds, while Chrome is active and the extension service worker is awake. If Chrome/Manifest V3 suspends background work, use Chrome alarms as a fallback and resume short-interval checks when the worker wakes.
- Use a Chrome alarm fallback with a practical minimum of about 1 minute, because Chrome alarms do not reliably support true every-5-second wakeups while the service worker is asleep.
- Start collection automatically when the bridge reports that the current time is inside an enabled collection window and that the window has not already been completed.
- Fetch jobs only from the local bridge.
- Open or inspect only configured sources.
- Prefer inactive background tabs (`active: false`) so collection does not take focus from the human's current tab.
- Close collector-created tabs after collection when configured.
- Do not promise fully invisible collection. A Chrome extension generally needs a real page/tab context to read logged-in private web pages; offscreen/background-only pages cannot reliably read arbitrary logged-in social feeds.
- Apply conservative pacing and delay rules.
- Default to 5 scrolls per private data source and wait 5 seconds between scrolls.
- Allow the human to configure up to 10 scrolls per private data source.
- Collect visible text, URLs, timestamps, engagement hints, profile URLs, post/current URLs, and source metadata.
- Collect relevant recommended groups/pages/communities as `new_private_sources` when visible.
- Post structured results back to the local bridge.
- Avoid posting, commenting, reacting, messaging, following, or changing account state (read-only; the send/act side needs separate human approval). Collect and structure whatever the job directs, including prospect contact details; the only data never read or transmitted is the operator's own credentials/secrets (usernames, passwords, cookies, tokens, session data, API keys) — see the `do_not_exfiltrate_secrets` note above.

The extension should not require the human to click Allow on every scheduled run. The one-time setup — running the Local Collector app setup/start command and installing/loading the extension — is agent-run on a LOCAL runtime and human-run outside the sandbox on a REMOTE runtime (see the Local vs remote runtime rule); either way, the human's own remaining one-time actions are limited to the two physical clicks Chrome requires of a person (Developer mode, drag the folder) and granting the extension permissions Chrome itself prompts for.

Expected extension check timing:

- If the Local Collector app is already running when the Solo Agency Local Collector extension is installed, the extension should ping `/status` immediately after install.
- If the extension is already installed and the Local Collector app starts later, the extension should ping on the next short poll while the service worker is awake, usually about 5 seconds.
- If Chrome has suspended the extension service worker, the next ping may happen on the Chrome alarm fallback, usually within about 60-75 seconds.
- Therefore, after starting the Local Collector app, the AI agent should wait and re-check `GET http://127.0.0.1:17321/status` for up to 75 seconds before declaring `no_extension_check_yet`.
- The AI agent must not report "extension has never pinged" immediately after installation or immediately after starting the Local Collector app.
- If there is still no extension check after about 75 seconds, likely causes include: Chrome is closed, the extension was not loaded, the extension is disabled, the wrong Chrome profile was used, the extension is configured to a different bridge URL/port, or the machine is asleep.
- If the extension popup is available, clicking `Check now` should force an immediate `/status` check.

### Localhost Bridge Security Requirements

The bridge must be simple, local, and conservative.

Required safeguards:

- Bind to `127.0.0.1` only.
- Reject non-local connections.
- Use a per-run random session token.
- Keep the token in memory and expire it when the bridge shuts down.
- Require the token on write endpoints.
- Restrict CORS to the installed extension origin when the extension ID is known.
- Reject unexpected origins.
- Limit request body size.
- Validate schema before writing files.
- Strip or ignore cookies, authorization headers, tokens, tracking parameters, and obvious secrets.
- Write only inside `daily-content-pipeline/collector/`.
- Never execute commands received from the extension.
- Never expose arbitrary filesystem reads.
- In on-demand mode, auto-shutdown on completion or timeout.
- In persistent scheduler mode, stay running after `/complete`; `/complete` marks only the active scheduled window as completed.
- Record bridge and extension health so AI agents can explain whether the bridge is running, whether the extension has checked in recently, and why private collection may be unavailable.

#### Retention And PII Handling

- Raw private captures under `daily-content-pipeline/collector/inbox/` stay local. They are never copied wholesale into reports or exports.
- Client-facing reports use safe summaries only, not raw captured text dumps.
- In any human-facing output, mask private-group member identities: use initials/role, never full names or profile URLs, unless the human explicitly approves showing more.
- The operator may purge old `inbox/` months after the corresponding reports are finalized. Purging inbox data must not delete reports, history, or config.

The bridge should expose only minimal endpoints, such as:

```text
GET  /status
GET  /config
POST /config
GET  /jobs/current
POST /jobs/run_now
POST /collect/data_point
POST /collect/lead
POST /collect/competitor
POST /collect/new_private_source
POST /collect/source_status
POST /collect/snapshot
POST /complete
POST /shutdown
```

`/shutdown` requires the per-run extension token (held only by the extension) and is not callable tokenless by the agent; a tokenless `POST /shutdown` returns 401. To stop the bridge, use the PID-based/port-based stop (read `collector.pid`, ask the human to stop that process outside the AI sandbox), not `/shutdown`.

Health API:

- `GET http://127.0.0.1:17321/status` is the Local Collector app health API.
- The AI agent may call `/status` at any time, before setup, before a manual run, during a run, after a run, before generating a report, before sending a Telegram notification, or while troubleshooting.
- `/status` is read-only. Calling it must not create a job, start a collection run, advance a schedule window, or mark a run complete.
- The AI agent should call `/status` without special headers.
- The Solo Agency Local Collector extension may call `/status` from its extension context and may include `X-Collector-Extension: media-agency-local-collector`; that is how the Local Collector app records `extension_health.last_extension_check_at`.
- The AI agent must not use the extension header during normal health checks, because it would make the bridge think the browser extension checked in when only the AI agent did.
- If `/status` fails to connect, the Local Collector app is not running or is blocked. On a LOCAL runtime, run the setup/start script generated during setup (one consent ask) and re-check; on a REMOTE runtime, give the human the one-line setup/start command generated during setup instead.
- If `/status` succeeds but `extension_health.status` is `stale` or `no_extension_check_yet` after the 75-second extension check grace window, the Local Collector app is running but the Solo Agency Local Collector extension is not currently checking in. The AI agent should treat private data source collection as unavailable until fixed, continue public data source work, and notify the human through the configured provider notification channel if available.

`POST /jobs/run_now` is required for manual runs and first-trial runs when localhost is reachable. It lets the AI agent tell the Local Collector app:

```text
Run this private data source job immediately. Do not wait for the recurring schedule window.
```

Run-now behavior:

- The Local Collector app queues the run-now job.
- The matching Solo Agency Local Collector extension claims the queued job on its next `/status` poll.
- `/status` returns `job_available: true` and `current_job_type: run_now` only for the client extension whose `client_slug` and bound `extension_instance_id` match the active/claimed job.
- With one shared bridge, private collection is parallel per client identity: multiple client Chrome profiles can collect at the same time through the same bridge, each with its own active job, output folder, counters, and completion state. Jobs for the same client/profile remain sequential. After `/complete` or TTL, the bridge continues to that client's next queued job.
- The run-now job should default to `force: false`.
- Each manual run should use a fresh unique `run_id`.
- The run-now job must include a TTL, default 30 minutes and maximum 120 minutes.
- If `/complete` is never received, the Local Collector app must stop exposing the active run-now job after its TTL expires, then allow the next queued job to proceed.
- After `/complete`, the run-now job is cleared so it does not repeat.

Run-now stuck-status guard:

- A manual or first-trial job must never be exposed forever.
- The Local Collector app must treat `run_now_expires_at` as a hard stop. After that time, `/status` must return `job_available: false` for that run-now job even if the Solo Agency Local Collector extension crashed, Chrome was closed, the machine slept, or `/complete` was never called.
- The agent must not set `force: true` for routine manual runs. `force: true` is reserved only for explicit troubleshooting when the human understands that it can intentionally re-run a previously completed `run_id`.
- The agent must not reuse yesterday's or a previous manual `run_id` to “run again”. It must create a new unique `run_id`.
- If the agent sees `current_job_type: run_now` for longer than the configured TTL, it should report a Local Collector app bug or stale process and notify the human through the configured provider notification channel if available. Recovery follows the local/remote rule from the Hard Gate above: on a LOCAL runtime the agent first confirms no other client's job is genuinely in flight (`GET /status` → `active_jobs`, and no fresh `run_lock`/lease held by another brain), tells the human in one line what it is about to do, then re-runs `setup_collector.sh` (idempotent; it stops the stale bridge and starts the newest one) and waits for `/status`; on a REMOTE runtime it provides the human-run setup/start command or the documented troubleshooting path instead. Never restart a bridge that is mid-run for another client.
- If the Solo Agency Local Collector extension reports `already_completed`, the agent should not force the same job. It should create a new run-now job with a new `run_id`.

The `/status` response should include:

- bridge status,
- active run/window id,
- current job type: `run_now`, `scheduled`, `on_demand`, or `none`,
- output directory,
- config file path,
- run-now request file path,
- config file updated timestamp when available,
- job availability,
- completed status,
- counts,
- `extension_health.last_extension_check_at`,
- `extension_health.seconds_since_last_check`,
- `extension_health.extension_check_count`,
- `extension_health.status` such as `recent`, `stale`, or `no_extension_check_yet`.

The AI agent must use `output_dir`, `config_file`, and `run_now_request_file` as a bridge identity check. A bridge is not healthy for the current setup unless those paths point to the current setup's `daily-content-pipeline/collector/` tree. A bridge can be `status: ready` and `extension_health: recent` while still being wrong for the current setup if it was started by a previous install.

The bridge should also write a local health file:

```text
daily-content-pipeline/collector/inbox/bridge_health.json
```

AI agents must read local health/status files when `GET http://127.0.0.1:17321/status` fails from an AI sandbox. A failed localhost request does not always mean the Local Collector app is inactive; in some scheduled-task or hosted-agent environments, the agent's `127.0.0.1` is isolated from the human computer's localhost.

Runtime verification fallback files:

```text
daily-content-pipeline/collector/inbox/bridge_health.json
daily-content-pipeline/collector/inbox/collector_status.json
daily-content-pipeline/collector/collector_setup_status.md
daily-content-pipeline/collector/inbox/YYYY-MM/{client_slug}/{run_id}/collector_status.json
daily-content-pipeline/collector/run_now_request_status.json
daily-content-pipeline/collector/run_now_request*.consumed.json
```

If these files show a recent current-workspace bridge, current-workspace output/config paths, and a recent extension check, the agent should use the file-based queue path (`daily-content-pipeline/collector/jobs/pending/{unique_job}.json`) and wait for collector output instead of asking the human to restart the collector. Use one unique file per client/run. If the files are missing, stale, or point to another setup folder, the agent must log the exact blocker (`collector_status_unverified`, `collector_offline_or_unreachable`, `wrong_workspace_bridge`, or `extension_status_unknown`) and continue with public data sources and previously collected private data when available.

#### Canonical Collector Blocker-Status Enum

These are the only allowed collector blocker statuses in reports and logs:

| Status | Meaning |
| --- | --- |
| `collector_offline_or_unreachable` | Bridge not running or not reachable (localhost failed and no fresh local health/status files). |
| `collector_status_unverified` | Reachability/health could not be confirmed either way. |
| `wrong_workspace_bridge` | A bridge is running but its config/output/run-now paths point to another setup folder. |
| `extension_status_unknown` | Bridge reachable, but the matching client extension check-in cannot be confirmed. |
| `activation_declined_for_now` | Human declined/postponed activating the collector for this run. |
| `collector_unavailable` | No compatible collector binary for this OS/CPU, or the collector is not installed at all (a build/install-availability status, distinct from a running bridge being offline). |
| `installed_and_running` | Collector installed, reachable, and healthy for the current setup. |

Do not invent other blocker-status values; map any situation to the closest value above.

Every time the extension checks `/status`, the bridge should update the last extension check timestamp. This lets the AI agent distinguish between:

- bridge not running,
- bridge running but extension not installed,
- bridge running but Chrome closed,
- extension installed but stale/sleeping,
- extension recent and healthy,
- private data source session expired,
- platform checkpoint/captcha/rate limit.

The bridge may run smoothly without admin permission on many machines because it binds only to loopback, but the agent must not promise zero operating-system prompts in every environment. Some corporate devices, antivirus tools, endpoint security tools, firewalls, Gatekeeper, or SmartScreen policies may still warn about new executables. Signed binaries are recommended for public distribution.

### Collector Output Files

For each run, the bridge should write:

```text
daily-content-pipeline/collector/
  jobs/
    pending/
      {unique_job}.json          # queued jobs waiting to be claimed
    claimed/
      {unique_job}.json          # jobs claimed by a matching extension
    completed/
      {unique_job}.json          # finished jobs
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
            source_slug_post_or_thread.html
```

The job queue moves a job file from `jobs/pending/` to `jobs/claimed/` to `jobs/completed/` as it is claimed and finished (a `jobs/failed/` sibling is used when a run fails). Output for each run is written under `inbox/YYYY-MM/{client_slug}/{run_id}/`, consistent with the Latest Override near the top of this file.

Every private data point must include:

- `client_slug`
- `source_name`
- `source_type`
- `platform`
- `profile_url` when applicable
- `post_url` or `current_url`
- `captured_at`
- `visible_text_summary`
- `raw_visible_text_excerpt` when safe and useful
- `engagement_hint` when visible
- `source_login_status`
- `collector_identity`
- `confidence`
- `source_uid` and `point_uid` (stamped by the bridge when `source_url` is present; `point_uid` only when `post_url` identifies a real item distinct from the scanned page)

Every detected lead must include:

- `profile_url`
- `post_url` or `current_url`
- `source_url` (the source the lead was detected in — required for the bridge's `source_uid` stamp and the shared-source lead-collision flag)

Every detected competitor must include:

- `profile_url`
- `post_url` or `current_url`
- `source_url` (required for the bridge's `source_uid` stamp)

Every new private data source candidate must include:

- `source_name`
- `platform`
- `source_type`: `joined_group`, `facebook_group_search_result`, `subreddit`, `community`, `followed_profile`, `followed_page`, `subscribed_channel`, `followed_company`, `recommendation_feed_author`, `recommendation_feed_topic`, or `other`
- `profile_or_group_url`
- `current_recommendation_url`
- `detected_while_scanning`
- `discovery_category`: `membership_sources`, `following_sources`, `recommendation_feed_sources`, or `keyword_search_sources`
- `discovery_url`
- `search_keyword` when discovered through keyword search
- `search_url` when discovered through keyword search
- `result_rank` when visible or inferable from order
- `membership_status`: `unknown`, `joined`, `not_joined`, `public_visible`, `requires_join`, or `unavailable`
- `why_relevant`
- `matched_pain_points`
- `related_content_pillar`
- `target_audience_fit`
- `location_fit`
- `lead_potential`
- `competitor_intelligence_value`
- `noise_level`
- `risk_level`
- `estimated_priority`
- `suggested_scan_cadence`
- `classification`: `recommended_daily`, `recommended_weekly`, `optional`, `watch_once`, `skip_not_relevant`, `skip_too_broad`, `skip_too_noisy`, `skip_sensitive_or_risky`, or `skip_platform_unavailable`
- `state`: `active`, `not_selected`, or `no_access` — set automatically (`recommended_*`/`optional` -> `active`, `skip_*` -> `not_selected`, unreadable/not-joined -> `no_access`); no human approval

If a URL is unavailable, write `unavailable` and include a note explaining why.

### Agent Compatibility Rule

All agents follow the same Local vs remote runtime rule (see the Hard Gates section) for one-time Local Collector setup/update/repair — Codex CLI running on the human's machine and Claude Code desktop/CLI are LOCAL runtimes; a hosted/cloud sandbox for any of them is REMOTE:

Codex:

- On a LOCAL runtime (Codex CLI on the human's machine), Codex runs `setup_collector.sh` / `setup_local_collector.ps1` / `Start Local Collector.cmd` itself: one plain-language safety line, one consent ask, then execute, then poll `/status` for up to 60 seconds.
- On a REMOTE runtime (Codex's own sandbox cannot see the install root, or `/status` still fails 60 seconds after a bootstrap attempt), Codex prepares the files and gives the human the Terminal/PowerShell command to run outside its sandbox instead.
- After setup is complete and the Local Collector app is reachable, Codex may create run-now jobs through `/jobs/run_now` or one unique per-client job file under `daily-content-pipeline/collector/jobs/pending/`, read collector output, and continue the daily pipeline. `run_now_request.json` is only a legacy/batch shim.
- If Codex cannot access Chrome's logged-in session directly, it should still use the extension/bridge output files.

Claude:

- Claude must use the Solo Agency Local Collector extension plus the Local Collector app for automated private data source collection.
- Claude must not use Claude Chrome Extension for automated private data source collection because it can require repeated human Allow clicks and can block unattended schedules.
- On a LOCAL runtime (Claude Code desktop/CLI running on the human's machine), Claude runs the one-time setup/start command itself, per the Local vs remote runtime rule. On a REMOTE runtime (a hosted Claude sandbox), Claude instead provides a user-run command, persistent bridge startup instructions, or OS startup service setup instructions.
- After the bridge is running, Claude reads collector output files and performs reasoning, idea generation, script writing, reporting, and WideCast actions.

Hermes, OpenClaw, and other agents:

- Same Local vs remote runtime rule: on a LOCAL runtime, run the setup/start command itself (one consent ask); on a REMOTE runtime, prepare files and give the human the Terminal/PowerShell command and Chrome extension steps instead.
- After setup is complete and the Local Collector app is reachable, use the same run-now/scheduled collector flow.
- If the agent cannot run local commands, read the latest collector files or use an MCP wrapper that exposes the collector folder.

### Native Messaging Decision

Do not require Native Messaging for the default version of this playbook.

Native Messaging is a valid production architecture, but it requires OS-specific host registration and may create more installation friction:

- macOS requires native host manifest placement and may trigger Gatekeeper warnings if unsigned.
- Windows requires registry registration and may trigger SmartScreen warnings if unsigned.
- Linux requires Chrome/Chromium-specific manifest paths.

The default collector should use localhost because it is easy for a human-run local app, OS startup service, or local scheduler to expose data safely to AI agents without sharing credentials.

Native Messaging may be added later as an advanced or enterprise option.

### Deprecated Browser Session Fallback

Older drafts allowed AI-agent-controlled headed browser profiles, CDP sessions, and native browser tools as fallback paths for private data source collection. That fallback is no longer allowed for the Solo Agency private data source workflow.

For private data sources, do not use:

- Claude in Chrome or Claude Chrome Extension;
- Codex built-in browser, Codex in-app browser, or Codex-controlled browser tools;
- ChatGPT/Gemini/Grok browser surfaces;
- Playwright/Puppeteer/Selenium controlled directly by the AI agent;
- a fresh browser profile opened by the AI agent;
- remote-debugging/CDP browser sessions opened or controlled by the AI agent;
- exported cookies, browser storage state, credentials, OTPs, or tokens.

The only supported private data source path is:

```text
Human's logged-in Chrome
  -> Solo Agency Local Collector extension
  -> Local Collector app running outside the AI sandbox
  -> local output files / localhost status
  -> AI agent reads local output and analyzes it
```

If the Local Collector is unavailable, the agent must continue work with public data sources only, use previously collected private data if available, or ask the human to complete Local Collector setup/repair. It must not improvise a browser fallback.

### AI-Service-Specific Guidance

Codex:

- Codex must not use native browser, in-app browser, Playwright, remote debugging, or agent-controlled browser tools for private data source review.
- Local Collector setup follows the Local vs remote runtime rule: on a LOCAL runtime Codex runs setup itself; on a REMOTE runtime it hands the human the command. Once the Local Collector app is reachable, Codex may create run-now jobs through `/jobs/run_now` or one unique per-client job file under `daily-content-pipeline/collector/jobs/pending/`, read collector output, and continue the daily pipeline. `run_now_request.json` is only a legacy/batch shim.

Claude:

- Claude must not use Claude in Chrome or Claude Chrome Extension for private data source collection.
- Claude must use the Solo Agency Local Collector extension plus the Local Collector app described above.
- On a LOCAL runtime (Claude Code desktop/CLI on the human's machine), Claude runs the one-time setup/start command itself: one plain-language safety line, one consent ask, then execute, then poll `/status` for up to 60 seconds. On a REMOTE runtime (a hosted Claude sandbox that cannot see the install root, or where `/status` still fails 60 seconds after a bootstrap attempt), Claude instead gives the human a one-time command or startup-service instructions to run the Local Collector app outside the sandbox.
- The recommended Claude-safe mode is `persistent_bridge_scheduler`, because once the Local Collector app is running at OS startup, Claude only needs to read local collector files.
- If the Local Collector app is unavailable, Claude should continue with public data sources and previously collected private data, then notify the human.

Other agents:

- Other AI agents must follow the same collector-only rule for private data sources.
- Native browser automation is allowed only for public pages, setup instructions, or local UI testing, not for private data source collection.

---


Manual run / run-now rule:

- Any human request such as `run now`, `manual run`, `test run`, `trial run`, `collect now`, or `scan now` must bypass recurring schedule windows.
- The agent must not wait for `scheduled_windows` when the human requested a manual run.
- If the Local Collector app is reachable, the agent must create a run-now job and call `POST http://127.0.0.1:17321/jobs/run_now`. The bridge will queue the job, not overwrite the currently active job.
- The run-now job must include:
  - unique `run_id`,
  - `run_now: true`,
  - `force: false` by default,
  - `run_now_ttl_minutes`, default 30 and maximum 120,
  - private `sources`,
  - pacing rules,
  - client/business/location metadata when available,
  - `allowed_extension_instance_ids` for the matching client extension whenever known.
- To run again, the agent should create a new unique `run_id` instead of forcing the same run id repeatedly.
- The run-now job must expire automatically if it is not completed, so the extension cannot keep seeing the same manual job all day.
- The matching Solo Agency Local Collector extension should see `job_available: true` on a `/status` poll when its queued job becomes active. With one shared bridge, jobs for different clients can be active at the same time; only jobs for the same client/profile are queued sequentially.
- If the Local Collector app is not reachable, apply the Local vs remote runtime rule: on a LOCAL runtime, run the setup/start script itself (one consent ask); on a REMOTE runtime, provide the one-line Local Collector app setup/start command for the human to run outside its sandbox. Either way, retry the run-now job only after the app is reachable.
- Recurring schedule windows are only for unattended scheduled runs. They must not block manual runs.
- Do not simulate a manual run by editing `scheduled_windows` or creating a temporary schedule window. Manual runs must use `/jobs/run_now` or file-based queued jobs.
- If the agent cannot call `http://127.0.0.1:17321` from its own sandbox but can write local files, it must write one unique per-client job file under `daily-content-pipeline/collector/jobs/pending/`. The Local Collector app claims matching pending jobs on `/status`, moves claimed files into `jobs/claimed/`, writes output for that client, then moves completed files into `jobs/completed/`. This avoids asking the human to run another command and avoids `run_now_request.json` overwrite races.
- `daily-content-pipeline/collector/run_now_request.json` remains a legacy/batch shim only. It may contain one job or a batch object with `{"jobs":[...]}`, and the bridge converts it into `jobs/pending/` queue files before consuming it. Do not use this single filename when multiple agents or scheduled tasks may write concurrently.
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
  "job_type": "run_now",
  "client_slug": "client-slug",
  "allowed_extension_instance_ids": ["ext-inst-abc123"],
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
    "max_delay_seconds": 10,
    "max_sources": 20,
    "scroll_steps": 5,
    "max_text_chars": 12000
  },
  "collector_policy": {
    "read_only": true,
    "do_not_comment": true,
    "do_not_post": true,
    "do_not_message": true,
    "do_not_react": true,
    "do_not_exfiltrate_secrets": true
  }
}
```

`pacing` is optional on any job. When a job carries none, the bridge seeds it from `collector_config.json`: `min_delay_seconds` = `scroll_delay_seconds` (default 5) and `max_delay_seconds` = `scroll_delay_max_seconds` when that key is set, otherwise `scroll_delay_seconds + 5` — so the default is a random 5–10 s pause before each page, scroll or request (Pacing Rule, `playbooks/10_LEAD_COMPETITOR_DETECTION.md`), never a fixed one. An explicit `pacing` on the job still wins; the bridge clamps both ends into 5–60 s. A source item in `collector_config.json` with `enabled: false` is one the Boss paused on the Sources page (Custom tab): the collector's own scheduled windows already skip it, and the run must leave it out of every run-now job it builds — resume happens only on that page (or `resume` on the Boss's word).

- `do_not_exfiltrate_secrets` (`true`) is the collector's single absolute data prohibition: the operator's own credentials and secrets — usernames, passwords, cookies, tokens, session/auth data, API keys — are never read, stored, or transmitted. Everything else is consented by the operator's setup + command: the collector may read, extract, and combine whatever the job directs, including a prospect's contact details (email/phone), surfaced in the optional `emails`/`phones` fields. The `read_only`/`do_not_message`/`do_not_comment`/`do_not_react`/`do_not_post` flags keep the **send/act** side gated — lead outreach still requires separate explicit human approval. These flags are enforced by the collector itself (`background.js`, before a write action runs) and the permission is minted by the bridge from the job's OWN sources (`collectorPolicyForJob`, `main.go`): a job that carries `fb.post.comment` arrives with `do_not_comment` cleared and every other write still forbidden, and a job that carries no write capability is refused if it tries one. Until 2026-08-17 the flags were carried on every job and read by nothing — do not write a rule that depends on a flag without checking that something enforces it.
- `run_id` must be unique for every manual run. A recommended pattern is `YYYY-MM-DD_client-slug_manual_HHMMSS`.
- `job_type` names the job kind, for example `run_now`, `scheduled`, or `private_data_source_discovery`.
- `allowed_extension_instance_ids` must be included whenever the client's extension instance id is known. It restricts which extension may claim the job and prevents another client's extension from cross-claiming the run.
- `run_now` must be `true`.
- `force` must be `false` unless the human explicitly asks for a troubleshooting rerun and understands the same `run_id` may run again.
- `run_now_ttl_minutes` should be 30 by default and must not exceed 120.
- `sources` must contain the private data sources for that client if private data sources exist. If there are no private data sources, the agent should still run public research without the Local Collector app.
- `pacing.scroll_steps` defaults to 5 and must not exceed 10 for daily monitoring.
- For Source Discovery Mode, `pacing.scroll_steps` must not exceed 10. Mark the job/source with a discovery indicator, such as `job_type: "private_data_source_discovery"`, `purpose: "source_discovery"`, or a discovery URL like `https://www.facebook.com/groups/joins/...`, while still keeping the 10-scroll hard cap.
- For Facebook keyword group search discovery, use `purpose: "facebook_group_keyword_search_discovery"`, `discovery_category: "keyword_search_sources"`, and `pacing.scroll_steps: 10` per keyword URL.
- If the agent cannot make this POST itself but can write local files, it should write the JSON payload as one unique file under:

```text
daily-content-pipeline/collector/jobs/pending/{timestamp}_{client_slug}_{run_id}.json
```

The agent should write this file atomically: write a temporary file in the same folder first, then rename it to a unique `.json` filename only after the JSON is complete. For multiple clients, write one queued file per client/run. Do not reuse a filename.

### Shared-Scan Gate — check the source registry before every job

Many clients monitor the SAME sources. The registry `daily-content-pipeline/collector/source_registry.json` gives every source one canonical identity (UID) across clients and remembers the last completed scan. The per-client extension/account model is unchanged — whichever client's run scans, scans with its own extension and login; sharing happens through the recorded data pointer, never through a shared login.

Discovery jobs are NOT scans: any job with `job_type: private_data_source_discovery` (or a group-keyword-search discovery purpose) bypasses this gate completely — discovery enumerates candidate sources, it does not collect source content, so never pass discovery URLs to `due` or `record`.

Before assembling any content-scan job's `sources[]`:

1. Run `tools/solo_tool source-registry --pipeline {setup-root}/daily-content-pipeline due --client {client_slug} --kind private --url URL1 --url URL2 ...` (or `--urls-file F`, one URL per line) with every source the run wants scanned. Always pass the ABSOLUTE pipeline root — the registry file location and stored pointers must not depend on the caller's cwd. Public recurring sources use `--kind public`; the two lanes never satisfy each other's freshness.
2. Put ONLY the returned `scan` entries into the job's `sources[]`. Each `reuse` entry carries `data_dir`, `scanned_by`, `completed_at`, `uid`, `uid_hash` — that scan's output is this run's data for that source. A reuse `data_dir` is the scanning client's WHOLE run directory: keep only the JSONL records whose `source_uid` equals the reused entry's `uid`; never read `snapshots/` or any other source's records from it. Freshness is a rolling TTL (`freshness_ttl_hours` in the registry file, default 20h), never a calendar-day compare.
3. `wait` entries mean another client claimed this source less than 2 hours ago and is scanning it right now: use the `stale_data_dir` pointer when present (same source_uid filter), otherwise skip the source this run and note `waiting_shared_scan` — never start a duplicate scan of a claimed source.
4. If every source is fresh, do not create a collector job at all — record `all_sources_reused` in the run notes and continue with the pointed-to data.
5. After the run reaches a terminal state, record what was actually scanned: `... source-registry --pipeline {setup-root}/daily-content-pipeline record --client {client_slug} --run {run_id} --kind private --url ...` (use `--status failed` for sources the collector could not read — a failed attempt never overwrites the last good scan, and the next subscriber's run retries with its own login; recording also releases this client's scan claim).
6. `scope: exclusive` sources (a client's OWN page/group/site) are never served as reuse — every run that queues one scans it itself with its own client login. Set with `... source-registry --pipeline {setup-root}/daily-content-pipeline set-scope --url U --scope exclusive`. That is the ONLY trigger for exclusive: sharing third-party sources is the agency's mandatory operating model, and a third-party source found flipped to exclusive without a verbatim, dated operator instruction is agent-invented drift — set it back to shared and report the correction.

First use on an existing install (no `source_registry.json` yet): BACKFILL before the first `due` — for each client, `register` every approved private source (Stage 2 scope rule: `--scope exclusive` for the client's own assets, shared otherwise) and every recurring public source (`--kind public`). The backfill also CLEANS legacy stored URLs: run `tool source-registry normalize` on every existing entry and rewrite the profile/`collector_config.json` value to the returned `clean_url` (pre-normalize installs hold pasted variants — trailing slashes, `?ref=` junk, `m.facebook.com` — that fragment per-client dedup). A `due` on an unregistered URL auto-creates the entry as scope `unclassified`, which scans normally but is NEVER served as reuse until an explicit `register` decides shared vs exclusive — so skipping the backfill costs sharing, never correctness.

Facebook group identity caveat: a numeric-ID URL and a vanity URL of the SAME group do not merge automatically. Register each group under ONE canonical form (prefer the vanity URL shown on the group page) and keep every client registering that same form.

Data points, leads, and competitors written by the bridge carry `source_uid` (canonical source identity, stamped when the record has `source_url`) and `point_uid` (identity of the specific post/item, stamped only when `post_url` identifies a real item distinct from the scanned page — page-level aggregate records carry no `point_uid` and dedup by visible text). Consumers dedup shared data by key equality first, visible-text matching second.

The running Local Collector app should pick up pending jobs on the next `/status` check from the matching Chrome extension, usually within a few seconds while Chrome is active. After claiming a pending job, the Local Collector app must move it through the queue lifecycle so it cannot loop forever:

- move it to `jobs/claimed/`;
- write `run_now_request_status.json`;
- route it only to the matching `client_slug` and bound `extension_instance_id`;
- clear the active run-now job on `/complete`;
- move the claimed file to `jobs/completed/` on `/complete`;
- expire the active run-now job after `run_now_ttl_minutes` if `/complete` never arrives, then allow the next queued job to proceed.

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
- Then write or update `schedule.md`, the automation manifest, the scheduled-run prompt/task body, and the relevant collector/config files.
- During Setup Flow, do not ask to run the first agency run immediately and do not run a report. Finish by preparing or resyncing the client-specific automation task whose task name begins with the client name.
- After schedule/routine setup and automation task creation, if private data sources exist, check the Local Collector installed unconditionally at step 4 (Kết nối Facebook, Instagram and X). If it is healthy, proceed. If it is unhealthy, repeat step 4's own install/repair flow (per `playbooks/SETUP_FLOW_ENTRYPOINT.md`, "Kết nối Facebook, Instagram and X (step 4)") — do not treat this as a new "guide Local Collector setup" gate — and, until it is healthy, clearly mark private data sources as `pending_private_activation` in the automation contract so the first automation run can continue with public data sources only if needed, then resync the task.
- If schedule/routine setup already happened and the human later approves private data sources, repairs Local Collector, changes scan depth, changes source cadence, connects notification/PDNA, or changes any future-run behavior, load Stage 4 and perform Automation Resync. Updating only `collector_config.json` is not enough when the native AI automation prompt/task may still contain an older setup snapshot.
- During Automation Resync, update the Client Intelligence Profile, `schedule.md`, `collector_config.json` or `POST /config` when relevant, `daily-content-pipeline/automation/automation_manifest.md`, `daily-content-pipeline/automation/scheduled_run_prompt.md`, the actual native scheduled task prompt if accessible, and `daily-content-pipeline/automation/resync_log.md`.
- Before claiming the schedule will use the new collector/source state, dry-read the scheduled entrypoint, manifest, schedule, profile, and collector config to confirm the next scheduled run will see the current approved sources/status.

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
- At every scheduled or manual run, perform Collector Runtime Verification BEFORE honoring any saved `public_data_sources_only`, `private sources postponed`, or `pending_private_activation` flag. Verification means: try `GET http://127.0.0.1:17321/status`; if localhost is unreachable from the AI sandbox, read the local collector health/status files (`inbox/bridge_health.json`, `inbox/collector_status.json`, `collector_setup_status.md`). The saved flag may be a stale snapshot from a previous run, so a live check must confirm the current collector state before the run acts on that flag.
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

### Write Capabilities — the collector can also ACT, not only read

Everything above describes reading. The collector also ships three **write** capabilities that
drive the real Facebook UI as the logged-in account. They are catalogued in
`collector_capabilities.json` with `"write": true`, implemented in the extension's
`gql_actions.js` (MAIN world, injected only for a write job — never during a read run), and
documented in full, with the failure modes behind each guard, in
`solo-agency-collector/HANDOFF_WRITE_ACTIONS.md`.

**Who may use them.** Only the agency's own outreach, under OutreachCRM, on the operator's own
brand and accounts, and only to execute a draft the operator has already approved. Never as, or
on behalf of, a paying client — see `playbooks/03` §23.7 and `playbooks/10`. Stage 10's scan
loop never sends, even after approval; execution lives in OutreachCRM Stages 16/17.

| Capability | What it does | Job url |
|---|---|---|
| `fb.post.react` | like/love a post | post permalink, or a listing url with `match_text` |
| `fb.post.comment` | comment on a post / group post / reel | post permalink, or a listing url with `match_text` |
| `fb.message.send` | send one Messenger DM | `messenger.com/e2ee/t/<vanity_or_profile_id>` |

Submitted like any other job — `POST /jobs/run_now` with `capability` + `inputs` — and the
result lands in the run's `private_data_points.jsonl` under `records.items[0]`.

**Targeting a post by CONTENT (`match_text`).** When the caller knows what a post *says* but
not its url, point `url` at the group or timeline and pass `match_text` (+ optional
`match_mode`: `contains` | `exact` | `regex`). The collector resolves the target **in code**:
it reads the listing, filters, and requires **exactly one** match. Zero matches returns
`no_match`; more than one returns `ambiguous_match`; **both write nothing** and return the
candidates they considered, so a refusal is auditable. On a hit it navigates to that permalink
and acts there — it never acts on the listing page, because a feed renders one
"Comment as …" composer *per post* and the first one found would be the wrong one.

Do not do this resolution yourself by listing the feed and eyeballing a match. That is
precisely the judgement call the `match_text` input exists to remove: two agents eyeballing
the same feed produce two different answers, and a wrong pick writes to a stranger's post.

**Identifying a DM recipient.** Identity is anchored to the **profile id** in the url, read
before the E2EE gate is passed (the gate rewrites the address to `/e2ee/t/<thread_id>` and the
id disappears). `recipient_name` is optional — a display name is a label the recipient can
change at will, so gating on it would refuse correct sends. It still decides when the id could
not be verified. A url that is not a real thread — a photo id, say — opens a page with no
composer at all and is refused.

**Reading the result.** A write succeeded only when the record says so: `status: "done"` **and**
`verified: true`, plus `id_verified: true` for a DM. Every other status is a refusal that
explains itself in `error` — `no_match`, `ambiguous_match`, `listing_unavailable`,
`unresolved_match`, `redirected`, `recipient_mismatch`, `ambiguous_composer`,
`not_a_thread_url`. Treat a refusal as information, not as a transient error to retry blindly:
these guards fire because the target could not be identified, and retrying without changing the
input will refuse again for the same good reason.

## Post-Collection Completion Checklist

Collector success alone is never completion. A finished collector job means data was captured, not that the run is done. After any collector job completes, the agent must confirm all of the following before treating the run as complete:

- Stage 10 was loaded (LOAD LEDGER) before presenting any leads or competitors.
- Collected data was analyzed for data points, leads, competitors, and newly discovered sources.
- The idea matrix, best idea, and drafts were updated from the new data.
- The private lane report (`{client-name}-private-data-sources-report.html`) and the daily staging index were updated, WITHOUT touching `{client-name}-public-data-sources-report.html`.
- The combined `{client-name}-client-report.html`, its PDF companion, and the INTERNAL_REPORT were rebuilt.
- `{client-name}-report_state.json` and the `outputs/latest/` copies were reconciled with the newest run.

If any item is not done, the run is not complete regardless of collector status.

---
