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
- The AI agent must never launch the bridge binary or its setup/start scripts from inside the AI sandbox, in any flow (setup, update, repair, or normal runs). This includes `setup_collector.sh`, `setup_local_collector.ps1`, `Start Local Collector.cmd`, and the collector binary itself, even if the agent has shell permissions. The human, or an installed OS startup service, must run the setup/start command outside the agent sandbox. The agent's role is to prepare files and hand the human the exact one-line command.
- **Sandbox localhost rule (hard gate).** A scheduled/automation run executes INSIDE the AI sandbox, where `127.0.0.1:17321` is normally NOT the human machine's Local Collector localhost. In sandbox/automation runs the agent must NOT depend on localhost as the control path: submit work through the file-based job queue (`daily-content-pipeline/collector/jobs/pending/`, one unique per-client file) and verify liveness from local health/status files (`bridge_health.json`, `collector_status.json`, recent run-now consumed status). The agent must NEVER tell the human the Local Collector is down, stopped, unresponsive, or needs a restart because a localhost request failed — a failed localhost call in a sandbox is expected network isolation and proves nothing about the collector, which may be running perfectly on the human's machine. A Local Collector error/blocker is valid ONLY when the FILE-QUEUE path itself fails: the local health/status files are missing, stale, or point to another workspace, OR a submitted job file is not claimed/consumed within its TTL. Only then report the exact blocker (`collector_status_unverified`, `collector_offline_or_unreachable`, `wrong_workspace_bridge`, or `job_not_consumed`) and continue with public data sources and previously collected private data.
- Before handing the human any command to run the bridge or any Chrome `Load unpacked` path, the agent must run the Source Safety Pre-Check (see the section below) and only then give the install steps. A verified-fresh checkout is not enough on its own.
- One-time setup must include both human actions: run the Local Collector app setup/start command, then install/load the Chrome extension from the absolute runtime extension folder.
- No credentials, hidden APIs, DMs, inboxes, account pages, or contact scraping.
- Private data source discovery jobs are allowed only after explicit human consent and must produce candidate sources for review, not automatically activated monitoring sources.
- A reachable bridge is not automatically healthy. The agent must verify `/status.config_file`, `/status.output_dir`, and `/status.run_now_request_file` point to the current setup's `daily-content-pipeline/collector/` tree. If they point elsewhere, mark `wrong_workspace_bridge` and require the human-run setup/start command for the current setup.
- A normal machine should have one active shared Solo Agency Local Collector runtime/bridge for the current setup, and one client-specific Solo Agency Local Collector Chrome extension per client Chrome profile/account. When old installs are suspected, ask the human to remove/disable stale entries in `chrome://extensions` and keep only the current per-client extension entries under `extensions/{client_slug}/`.

## Source Safety Pre-Check

Before giving the human any bridge start command or any Chrome `Load unpacked` path, read the code that will actually run on the human's machine and confirm it does not send data anywhere off the local machine. This is a light "where do requests go" read, not a full security audit. It protects against the case where the upstream repo was hijacked and now ships code that exfiltrates the human's logged-in data — something the fresh-checkout verification alone cannot catch.

Scan these three, in the exact copies that will be installed/run (the prepared per-client folder, not only the template):

1. The per-client extension folder `extensions/{client_slug}/` — every `*.js` file.
2. The bridge source `solo-agency-collector/bridge-go/main.go`.
3. `solo-agency-collector/scripts/prepare_client_extension.sh`.

What to look for (outbound requests only):

- Network call sites: `fetch(`, `XMLHttpRequest` / `.open(`, `navigator.sendBeacon`, `new WebSocket(`, `new EventSource(`, image beacons (`new Image()` / `.src =` to a URL), and in the bridge any outbound HTTP client (`http.Get`/`http.Post`/`http.Client`/`net.Dial`) or `curl`/`wget` in the script.
- For each real call site, confirm the destination is the local bridge only: `http://127.0.0.1:<port>` or `http://localhost:<port>` (the `bridgeBaseUrl`). If `client_binding.json` overrides `bridge_base_url`, confirm that override is also `127.0.0.1`/`localhost`.
- Confirm the bridge binds to `127.0.0.1`/`localhost` (not `0.0.0.0`) and has no outbound/telemetry client.

Known false positives — DO NOT flag these (read the whole line for context; never flag on a substring match alone):

- `readability.js` is the vendored Mozilla Readability DOM parser. Its `XMLHttpRequest`/`_ajax` helper is defined but never called, and its many `http://` strings are license, attribution, and code-comment/docstring references. Not egress.
- URLs inside comments or usage examples (e.g. a `https://www.facebook.com/...` line inside a `/* Usage: ... */` block).
- A placeholder base URL such as `https://example.invalid/` passed into `new URL(href, base)` to resolve relative links — it is never fetched.
- `fetch(chrome.runtime.getURL("client_binding.json"))` and any `chrome.runtime.getURL(...)` / `chrome-extension://` target — this reads a file packaged inside the extension, not the network.
- Broad `host_permissions` like `http://*/*` and `https://*/*` in `manifest.json` — a page-reading collector needs read access to whatever site the human approved. Broad read scope is expected and is not exfiltration; judge only by where requests are sent.
- Substring artifacts from a plain text search (e.g. `nc ` inside `func `, or `http` inside a comment). Confirm against the actual code line.

Outcome:

- If every real outbound request goes only to the local bridge and the bridge has no outbound client: the pre-check passes. Record the result (files/commit reviewed, call sites checked, destinations) in `INTERNAL_REPORT` only, and give the human exactly one short, calm confirmation line in plain language before the install steps, for example: `I read through the collector's code and confirmed it only runs on your computer and does not send your data anywhere. It is safe to install.` Do not list findings, severities, or technical terms to the human, and do not add extra warnings that could worry a non-technical user.
- If a real request goes to any non-local destination, or the bridge opens an outbound connection, or code is obfuscated so the destination cannot be read: do NOT say it is safe and do NOT give the install command. Stop, record the exact finding (file, line, destination) in `INTERNAL_REPORT`, and raise it to the operator in an `**[ACTION REQUIRED]**` block, in calm plain language, so it can be checked against the latest verified GitHub source before any install.

## Latest Override: One Shared Bridge, Many Client Extensions

The current multi-client model supersedes older one-extension wording:

- A normal machine should have one active shared Local Collector app/bridge for the current agency root.
- Each client may have its own Chrome profile/account and must have its own unpacked extension folder under `extensions/{client_slug}/`.
- Recommend one separate Chrome profile per client. That profile should have the matching client extension installed and should be logged in to the social accounts/private data sources that the human is already authorized to view for that client.
- The Chrome extension display name must begin with the client name: `{Client Name} - Solo Agency Collector`.
- The human-facing `Load unpacked` folder for a client is the absolute path to `extensions/{client_slug}/`, not `solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME/`.
- The shared bridge routes jobs by `client_slug`, binds each active job to the claiming `extension_instance_id` when present, can run different client identities in parallel, serializes only jobs for the same client/profile, and writes output only under `daily-content-pipeline/collector/inbox/YYYY-MM/{client_slug}/{run_id}/`.
- Agents running in sandboxes should prefer file-based job requests under `daily-content-pipeline/collector/jobs/pending/` and local health/status files over localhost calls.
- Extension popup/settings must not write global `collector_config.json`; global agency/collector config is managed by the agent/playbook and Automation Resync.

Per-client extension setup handoff:

```text
Open the Chrome profile/account for {Client Name}.
Go to chrome://extensions.
Enable Developer mode.
Click Load unpacked.
Select:
{ABSOLUTE_AGENCY_ROOT}/extensions/{client_slug}/
```

Every Add Client or First Client Setup handoff must include this block with the real absolute path. The agent must not merely say that the extension was created. The human needs the path and steps because a new unpacked extension must be loaded into the matching client Chrome profile/account before private data source collection can work for that client.

The agent must run the Source Safety Pre-Check first and precede this handoff with the one short plain-language safety confirmation line (see the Source Safety Pre-Check section). Do not give the `Load unpacked` path or the bridge command until the pre-check has passed.

The agent must prepare `extensions/{client_slug}/manifest.json` with at least:

```json
{
  "name": "{Client Name} - Solo Agency Collector"
}
```

It may also set `"short_name": "{Client Name} Collector"` and a client-specific `"description"` / `"action.default_title"`. The helper `scripts/prepare_client_extension.sh` patches only `name`, `description`, and `action.default_title` (not `short_name`); its output is compliant. `short_name` is optional.

The agent must also create `extensions/{client_slug}/client_binding.json` with `client_slug`, `client_name`, `extension_instance_id`, `extension_display_name`, and `bridge_base_url`.

Bridge/extension health for automation must be checked per client. A global `extension_health.status: recent` is not enough when multiple client extensions exist; the scheduled task must find the matching extension entry for the target `client_slug` and `extension_instance_id`.

## Source Preservation Rule

This file is detailed source material moved from the original monolithic `SOLO_AGENCY_PLAYBOOK.md`.

Do not summarize away requirements, examples, checklists, schemas, protocols, URLs, edge cases, warnings, approval gates, or completion gates. If a downstream agent needs to shorten its response to the human, it may summarize the response, but it must still obey the full requirements in this file.

---

## Latest Delta Override: Discovery Mode Versus Daily Monitoring

Source Discovery Mode normally scrolls until no new source names or URLs appear for 3 consecutive scrolls, with a hard safety cap of 10 scrolls.

Facebook keyword group search discovery is a bounded Source Discovery Mode variant: for `https://www.facebook.com/search/groups/?q={url_encoded_keyword}`, use 10 scrolls per keyword by default, with `purpose: "facebook_group_keyword_search_discovery"`. This search-results pass is for collecting candidate groups, filtering UI noise, and asking the human to approve sources. It must not join groups, request access, or add groups as active sources without approval.

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

When the human has no private data source list, discovery is a first-class Local Collector job type. The job should use approved discovery surfaces such as joined Facebook groups, Facebook keyword group search, joined/subscribed subreddits, followed pages/KOLs, subscribed channels, communities, and feeds. It must return candidate sources with enough context for the agent to filter and ask for human approval before saving anything as active.

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

The AI agent should prepare the collector locally as much as its environment allows, but it must not start the one-time setup script or collector app itself. The setup/start command must be run by the human outside the AI agent sandbox so the Local Collector app survives after the agent command/session ends.

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
    {client_slug}/
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
- The only folder the agent may tell a normal human to load in Chrome for a client is the per-client runtime folder under `extensions/{client_slug}/`.
- If both folders exist, the agent must explicitly warn: `Do not load the extension folder inside solo-agency/solo-agency-collector. Load only the extensions/{client_slug} folder shown below for this client.`
- A normal machine should have only one active shared Solo Agency Local Collector runtime/bridge, but may have multiple client-specific Solo Agency Local Collector extensions, one per client Chrome profile/account.
- The generated setup instructions, setup status file, and chat message must show the absolute per-client extension path: `extensions/{client_slug}/`.
- Do not put app binaries, downloaded zips, the unpacked extension, PID files, or collector logs inside `daily-content-pipeline/`. That folder should remain data/config/output only.

Install flow:

1. Detect the user's OS and CPU architecture.
2. Establish a verified fresh source: use the current setup root only if it passes Fresh Source Verification, otherwise clone GitHub `main` into a fresh unique `mktemp -d` checkout. If fresh source cannot be verified, stop instead of using local cache.
3. Copy `SHA256SUMS`, `collector-bridge-binaries-0.1.0.zip`, extension templates, and helper scripts only from the verified checkout, or download the exact raw GitHub URLs above when cloning is unavailable.
4. Verify checksums when the environment has checksum tools available.
5. Extract bridge binaries into the absolute runtime path for `solo-agency-local-collector/bin/`.
6. Prepare the Chrome extension template into the absolute per-client path `extensions/{client_slug}/`, patch the manifest name to `{Client Name} - Solo Agency Collector`, and create `client_binding.json`. Prefer the repo helper when available from the verified checkout:
   ```bash
   solo-agency-collector/scripts/prepare_client_extension.sh "{Client Name}" "{client_slug}" "{extension_instance_id}" "{ABSOLUTE_AGENCY_ROOT}"
   ```
7. Select the correct bridge binary for the current machine.
8. On macOS/Linux, ensure the selected binary is executable.
9. Create the setup/start script or launcher, but do not execute it from the AI agent.
10. Give the human exactly one Terminal/PowerShell command or one double-clickable launcher path to run outside the AI sandbox.
11. In the same human-facing message, give the Chrome extension `Load unpacked` steps and the one absolute per-client extension folder path.
12. Record the verified source path and commit hash in `daily-content-pipeline/collector/collector_setup_status.md` when the file exists or is being created.
13. After the human confirms both actions, health-check `GET http://127.0.0.1:17321/status` and run the workspace identity check before claiming the collector is healthy.
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

### Required One-Time Human Setup Handoff

During first Local Collector activation, setup repair, or collector update, the AI agent must not run the setup/start command itself. This is mandatory for Claude, Codex, Hermes, OpenClaw, ChatGPT, and other AI agents, even when the shell appears to allow local execution, because the command may run inside an agent sandbox/session whose processes can be killed when the agent turn ends.

The agent must create or update the local setup files, then hand the human exactly these two actions in chat:

1. Run the Local Collector app setup/start command outside the agent sandbox:
   - macOS/Linux: `bash "/ABSOLUTE/PATH/TO/solo-agency-local-collector/setup_collector.sh"`
   - Windows: one prepared PowerShell command or one double-clickable `Start Local Collector.cmd` path.
2. Install the client-specific Solo Agency Local Collector Chrome extension in the matching Chrome profile/account:
   - use a separate Chrome profile for this client when possible;
   - make sure this profile is already logged in to the approved private data sources and the human has member/follower/subscriber/access rights for them;
   - open `chrome://extensions`;
   - enable Developer mode;
   - click `Load unpacked`;
   - **prefer the browser-UI drag-drop path** — hand the human `http://127.0.0.1:17321/ui/{client_slug}/extension` (click "Open the extension folder", then drag it onto `chrome://extensions`); the absolute per-client folder `/ABSOLUTE/PATH/TO/extensions/{client_slug}/` is the manual fallback for the file picker.

The human-facing setup message must show both actions together. Do not say only "I started it", "I ran setup", or "instructions are in collector_setup_status.md".

After the human confirms both actions are done, the agent may check `GET http://127.0.0.1:17321/status` and inspect collector logs/status files. Health checks are allowed; starting the one-time setup script from the AI agent is not.

For later scheduled runs, do not ask the human to repeat these steps. Use the already-running persistent Local Collector app, create scheduled/run-now jobs when available, and notify the human only if the app or extension becomes unavailable.

Chrome extension installation flow:

1. The agent copies/extracts and patches the extension into an absolute per-client path, for example:

```text
/Users/alex/oneman_agency/extensions/avenngo/
```

2. The agent tells the human directly in chat, Telegram, or another human-facing channel:

```md
Please install the Solo Agency Local Collector extension for {Client Name}:

1. Open the Chrome profile/account for {Client Name}. A separate Chrome profile per client is recommended.
2. Go to `chrome://extensions`.
3. Turn on `Developer mode`.
4. Click `Load unpacked`.
5. Select this folder:
   `/ABSOLUTE/PATH/TO/extensions/{client_slug}/`

Important: if you also see a folder named `solo-agency/solo-agency-collector/chrome-extension`, do not select that one. That is the toolkit/source copy. Select only the client folder under `extensions/{client_slug}/`.

After this one-time setup, you may close this instruction tab whenever you want. For private data source collection to work at scheduled times, that Chrome profile should be open, logged in to the social accounts/private data sources approved for this client, and already have member/follower/subscriber/access rights for those sources. The shared Local Collector app should be running or configured to auto-start.
```

3. The agent must not ask for passwords, cookies, OTPs, or credentials.
4. If the extension is not installed or cannot contact the Local Collector app, the agent logs `extension_unavailable`, continues with public data sources, and notifies the human.

The AI agent must create a ready-to-run setup/start script file and give the human exactly one short command to paste into Terminal or PowerShell. The agent must do this even if it can run local commands itself; one-time collector setup must happen outside the AI agent sandbox.

Do not show the human a long multi-line script as the primary instruction. Non-technical humans should not have to copy a large code block.

Do not tell the human that setup instructions are only in a Markdown file. The Markdown file may store the same information for agent memory, but the current chat must contain the exact action the human should take.

The generated collector setup script must be named `setup_collector.sh`. Do not invent alternative names such as `start_local_collector.sh`. The AI agent must not execute this script itself during one-time setup/update/repair; it must provide the command for the human to run in Terminal. Every run must check who owns the collector port before starting a new Local Collector app.

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

echo "Per-client Chrome extension folders are managed separately under $AGENCY_ROOT/extensions/{client_slug}."

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
echo "Client-specific Chrome extension folders are prepared separately under: $AGENCY_ROOT/extensions/{client_slug}/"
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

For the Local Collector app setup action, tell the human only this one-line command, with the real absolute path:

```bash
bash "/ABSOLUTE/PATH/TO/solo-agency-local-collector/setup_collector.sh"
```

Human-facing wording for the required two-step setup:

```md
I created the Local Collector setup file. Please do these two one-time steps:

Step 1 - start the Local Collector app outside the AI agent sandbox.
Open Terminal, paste this one line, and press Enter:

`bash "/ABSOLUTE/PATH/TO/solo-agency-local-collector/setup_collector.sh"`

**Easiest path (recommended) — the browser UI does the folder-finding.** Give the human this one link and three no-typing steps:
`http://127.0.0.1:17321/ui/{client_slug}/extension` → click **Open the extension folder** (Finder/Explorer opens the exact folder) → in the client's Chrome, open `chrome://extensions`, turn on **Developer mode**, and **drag that folder onto the page**. The UI page flips to a green ✓ connected on its own when the extension checks in — no path to remember, no file picker. The absolute-path instructions below are the manual fallback only.

Step 2 (manual fallback) - load the client-specific Chrome extension in the Chrome profile/account for this client.
Open Chrome -> `chrome://extensions` -> turn on Developer mode -> Load unpacked -> select this folder:

`/ABSOLUTE/PATH/TO/extensions/{client_slug}/`

Important: do not select any `solo-agency/solo-agency-collector/chrome-extension` folder. Use only the client folder under `extensions/{client_slug}/`.

After both steps are done, tell me "done". Then I will check the Local Collector status and continue.
```

Windows:

Do not tell Windows users to run bash. On Windows, use PowerShell or create a `.cmd` launcher that the human can double-click.

Important Windows note:

- The human can run an `.exe`, but double-clicking `collector-bridge-windows-amd64.exe` by itself is not enough for the recommended persistent setup because the app needs configuration arguments.
- The AI agent should create a friendly launcher such as `Start Local Collector.cmd` and, if needed, a setup script such as `setup_local_collector.ps1`.
- The human-facing instruction must include two setup actions: first run the prepared PowerShell command or double-click `Start Local Collector.cmd` outside the AI sandbox, then load the Chrome extension from the absolute runtime extension folder.
- If the human wants it to run after restart, use Windows Task Scheduler with "At log on".

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

Write-Host "Per-client Chrome extension folders are managed separately under $AgencyRoot\extensions\{client_slug}."

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
Write-Host "Client-specific Chrome extension folders are prepared separately under: $AgencyRoot\extensions\{client_slug}\"
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

Human-facing Windows wording for the required two-step setup:

```md
I created the Local Collector setup file. Please do these two one-time steps:

Step 1 - start the Local Collector app outside the AI agent sandbox.
Open PowerShell, paste this one line, and press Enter:

`powershell -ExecutionPolicy Bypass -File "C:\ABSOLUTE\PATH\TO\solo-agency-local-collector\setup_local_collector.ps1"`

**Easiest path (recommended) — the browser UI does the folder-finding.** Give the human this one link and three no-typing steps:
`http://127.0.0.1:17321/ui/{client_slug}/extension` -> click **Open the extension folder** (Explorer opens the exact folder) -> in the client's Chrome, open `chrome://extensions`, turn on **Developer mode**, and **drag that folder onto the page**. The UI page flips to a green connected on its own when the extension checks in — no path to remember, no file picker. The absolute-path instructions below are the manual fallback only.

Step 2 (manual fallback) - load the client-specific Chrome extension in the Chrome profile/account for this client.
Open Chrome -> `chrome://extensions` -> turn on Developer mode -> Load unpacked -> select this folder:

`C:\ABSOLUTE\PATH\TO\extensions\{client_slug}\`

Important: do not select any `solo-agency\solo-agency-collector\chrome-extension` folder. Use only the client folder under `extensions\{client_slug}\`.

After both steps are done, tell me "done". Then I will check the Local Collector status and continue.

Later, if you need to start the Local Collector app manually again, double-click:
`C:\ABSOLUTE\PATH\TO\solo-agency-local-collector\Start Local Collector.cmd`
```

Future update rule:

- When the project moves from raw GitHub files to GitHub Releases, replace the raw artifact URLs with GitHub release URLs.
- When the extension is published to Chrome Web Store, replace the developer-mode `Load unpacked` flow with the Chrome Web Store install flow.
- Until then, the AI agent should handle download/extraction/script preparation automatically when possible, but the human must perform both one-time local actions outside the AI sandbox: run the Local Collector app setup/start command and install/load the Chrome extension.

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
   - the human should run the current setup's one-line Local Collector command outside the AI sandbox so the script can stop the old `collector-bridge` process and start the bridge with the current workspace paths;
   - if the human has loaded old Solo Agency Local Collector extensions in Chrome, they should open `chrome://extensions`, remove or disable stale entries from previous setup folders, and keep only the current client-specific extensions loaded from this setup's absolute `extensions/{client_slug}/` folders. Multiple current extensions are expected when multiple clients use different Chrome profiles/accounts.
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
Action: Please run the Local Collector setup/start command for the current setup outside the AI sandbox. If you previously loaded old Solo Agency Local Collector extensions, open chrome://extensions and remove or disable stale entries from old setup folders. Keep the current client-specific extension loaded from /Users/alex/oneman_agency/extensions/{client_slug}/ in the matching client's Chrome profile.
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

If the bridge is not installed as a startup service, the human must start it manually after reboot by running the prepared setup/start command outside the AI sandbox. The agent should not start it from inside the AI sandbox during setup or repair.

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
4. The human-run local process or installed startup service already has the bridge running on `127.0.0.1`. The agent must not start the bridge itself; if it is not reachable, the agent gives the human the one-line start command to run outside the AI sandbox and waits.
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

If the local bridge is not already running, the agent should not assume it can safely start an on-demand localhost bridge from inside the AI sandbox. In that case:

- The extension may queue a limited amount of data in extension storage until a bridge is available.
- The agent must log `collector_unavailable`.
- The agent must notify the human that the Local Collector app is not running and provide the one-line setup/start command for the human to run outside the AI sandbox.
- The agent should continue with public data sources and previously collected private data if available.

Important constraint:

- A Chrome extension cannot magically start a localhost server if no local process is already running. If Native Messaging is not used, then the human-run setup command, an OS startup service, or another local scheduler must start the Local Collector app outside the AI sandbox.

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

The extension should not require the human to click Allow on every scheduled run. The human's one-time actions are running the Local Collector app setup/start command outside the AI sandbox, installing/loading the extension, and granting the extension permissions requested by Chrome.

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
- If `/status` fails to connect, the Local Collector app is not running or is blocked. The AI agent must not start it from inside the AI sandbox during setup/repair; give the human the one-line setup/start command generated during setup.
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
- If the agent sees `current_job_type: run_now` for longer than the configured TTL, it should report a Local Collector app bug or stale process, notify the human through the configured provider notification channel if available, and provide the human-run setup/start command or documented troubleshooting path instead of restarting the Local Collector app from inside the AI sandbox.
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

Every detected lead must include both:

- `profile_url`
- `post_url` or `current_url`

Every detected competitor must include both:

- `profile_url`
- `post_url` or `current_url`

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
- `approval_status`: `pending_human_approval`, `approved`, `rejected`, or `skipped`

If a URL is unavailable, write `unavailable` and include a note explaining why.

### Agent Compatibility Rule

Codex:

- During one-time Local Collector setup/update/repair, Codex must not run `setup_collector.sh`, `setup_local_collector.ps1`, `Start Local Collector.cmd`, or the collector binary itself, even if Codex has shell permission. Codex must prepare the files and give the human the Terminal/PowerShell command to run outside the Codex sandbox.
- After the human-run setup is complete and the Local Collector app is reachable, Codex may create run-now jobs through `/jobs/run_now` or one unique per-client job file under `daily-content-pipeline/collector/jobs/pending/`, read collector output, and continue the daily pipeline. `run_now_request.json` is only a legacy/batch shim.
- If Codex cannot access Chrome's logged-in session directly, it should still use the extension/bridge output files.

Claude:

- Claude must use the Solo Agency Local Collector extension plus the Local Collector app for automated private data source collection.
- Claude must not use Claude Chrome Extension for automated private data source collection because it can require repeated human Allow clicks and can block unattended schedules.
- Claude must provide a user-run command, persistent bridge startup instructions, or OS startup service setup instructions. It must not run the one-time setup/start command from inside Claude.
- After the bridge is running, Claude reads collector output files and performs reasoning, idea generation, script writing, reporting, and WideCast actions.

Hermes, OpenClaw, and other agents:

- During one-time Local Collector setup/update/repair, use the same human-run setup rule: prepare files, then give the human the Terminal/PowerShell command and Chrome extension steps. Do not run the setup/start command from inside the AI agent.
- After human-run setup is complete and the Local Collector app is reachable, use the same run-now/scheduled collector flow.
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
- After the human-run Local Collector setup is complete and the Local Collector app is reachable, Codex may create run-now jobs through `/jobs/run_now` or one unique per-client job file under `daily-content-pipeline/collector/jobs/pending/`, read collector output, and continue the daily pipeline. `run_now_request.json` is only a legacy/batch shim.

Claude:

- Claude must not use Claude in Chrome or Claude Chrome Extension for private data source collection.
- Claude must use the Solo Agency Local Collector extension plus the Local Collector app described above.
- Claude must give the human a one-time command or startup-service instructions to run the Local Collector app outside the sandbox. It must not run the one-time setup/start command from inside Claude.
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
- If the Local Collector app is not reachable, the agent must not try to start it from inside the AI sandbox during one-time setup/repair. Provide the one-line Local Collector app setup/start command for the human to run outside the sandbox, then retry the run-now job only after the app is reachable.
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

- `do_not_exfiltrate_secrets` (`true`) is the collector's single absolute data prohibition: the operator's own credentials and secrets — usernames, passwords, cookies, tokens, session/auth data, API keys — are never read, stored, or transmitted. Everything else is consented by the operator's setup + command: the collector may read, extract, and combine whatever the job directs, including a prospect's contact details (email/phone), surfaced in the optional `emails`/`phones` fields. The `read_only`/`do_not_message`/`do_not_comment`/`do_not_react` flags keep the **send/act** side gated — lead outreach still requires separate explicit human approval.
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
- After schedule/routine setup and automation task creation, if private data sources exist and Local Collector is pending, handle step 6: guide Local Collector setup or clearly mark private data sources as `pending_private_activation` in the automation contract so the first automation run can continue with public data sources only if needed, then resync the task.
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
