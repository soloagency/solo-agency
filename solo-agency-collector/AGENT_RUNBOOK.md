# Agent Runbook: Local Collector

Use this runbook when an AI agent needs to collect data from private data sources through the Solo Agency Local Collector extension and localhost bridge.

## Key Rule

Do not ask the user to install Go.

Go is only used by maintainers or CI to build precompiled bridge binaries. In normal operation, the agent runs the correct prebuilt bridge binary for the user's OS/CPU.

If any client has private data sources, the first trial may still run public-first to give the human a quick useful result. The report must list those private data sources as pending activation and ask whether the human wants to activate private data source monitoring now.

Collector setup becomes a hard gate only after the human agrees to activate private data sources, or before any scheduled/manual run that promises private data source collection. The agent must not claim private data source monitoring is active if the collector is not installed and healthy.

When private data source activation begins, or before any schedule/manual run that promises private data source collection, the agent must create or update:

```text
daily-content-pipeline/collector/collector_setup_status.md
```

That file must say whether the Solo Agency Local Collector extension and Local Collector app are `pending_private_activation`, `activation_declined_for_now`, `installed_and_running`, `wrong_workspace_bridge`, waiting for a specific human action, blocked by sandbox/OS permission, stale, or offline.

## Required Inputs

- A client pipeline folder.
- A collector job JSON file.
- A prebuilt bridge binary for the current OS/CPU, downloaded from the Solo Agency GitHub distribution if it is not already present.
- The Solo Agency Local Collector extension installed in the user's logged-in Chrome profile.

## Solo Agency GitHub Distribution

Use the Solo Agency GitHub repo as the default setup source. Until GitHub Releases and Chrome Web Store are available, use these raw GitHub artifact URLs:

```text
https://raw.githubusercontent.com/soloagency/solo-agency/dist/collector-bridge-binaries-0.1.0.zip
https://raw.githubusercontent.com/soloagency/solo-agency/dist/chrome-extension-collector-root-0.1.0.zip
https://raw.githubusercontent.com/soloagency/solo-agency/dist/SHA256SUMS
```

### Recommended: use the canonical `setup_collector.sh` (do NOT hand-write the download/checksum step)

There is ONE supported installer. Download it and have the human run it — it downloads the
bundle, verifies the checksum (matched by BASENAME, so it works regardless of the
`SHA256SUMS` path format), extracts the right binary for the machine, and prints the
launch command. It never starts the bridge and is safe to re-run.

```text
https://raw.githubusercontent.com/soloagency/solo-agency/dist/setup_collector.sh
```

```bash
# from the agency root (the folder that contains daily-content-pipeline/):
curl -fsSL -o setup_collector.sh https://raw.githubusercontent.com/soloagency/solo-agency/dist/setup_collector.sh
bash setup_collector.sh
```

Do NOT reimplement the checksum parsing yourself: a hand-written parser that assumed a
bare filename dead-ended a real setup because `SHA256SUMS` can list a full path. The
canonical script already handles bare-name, `*name`, and full-path formats.

The manual steps below are the fallback only if the script cannot be used:

1. Resolve the absolute agency root that contains both `daily-content-pipeline/` and `solo-agency-local-collector/`.
2. Download the runtime files into the absolute `solo-agency-local-collector/downloads/` folder.
3. Verify checksums by matching the file's BASENAME in `SHA256SUMS` (it may list a bare name, `*name`, or a full path); never require an exact full-path match.
4. Extract bridge binaries into the absolute `solo-agency-local-collector/bin/` folder.
5. Extract/copy the Chrome extension template into a per-client absolute `extensions/{client_slug}/` folder.
6. Select the correct bridge binary for the user's OS/CPU.
7. Never run the downloaded executable itself. Prepare the files and give the human the one-line command to run outside the AI sandbox; the agent must not start the bridge or setup scripts, even with shell permissions.
8. Prefer persistent scheduler mode for unattended private data source collection.
9. Use on-demand mode only when a human-run process or OS startup service already has the bridge reachable for that run; the agent still does not start it.

The user still needs to install the Chrome extension once using Chrome Developer Mode until the extension is available in Chrome Web Store.

## Runtime Folder

```text
solo-agency-local-collector/
  downloads/
  bin/
  setup_collector.sh
  collector.pid
  collector.log
extensions/
  {client_slug}/
    manifest.json
    background.js
    popup.html
    popup.js
    client_binding.json
daily-content-pipeline/collector/
  collector_setup_status.md
  collector_config.json
  extension_registry.json
  jobs/
  inbox/
```

## Latest Multi-Client Runtime Rule

Use one shared Local Collector app/bridge per machine, but one unpacked Chrome extension folder per client Chrome profile/account.

The human-facing extension path is:

```text
/ABSOLUTE/PATH/TO/extensions/{client_slug}/
```

The extension display name must begin with the client name:

```text
{Client Name} - Solo Agency Collector
```

Each client extension folder must include `client_binding.json` with `client_slug`, `client_name`, `extension_instance_id`, `extension_display_name`, and `bridge_base_url`.

Agents running in automation should write per-client private data source jobs under:

```text
daily-content-pipeline/collector/jobs/pending/
```

The bridge claims matching jobs by `client_slug + extension_instance_id` and writes output under:

```text
daily-content-pipeline/collector/inbox/YYYY-MM/{client_slug}/{run_id}/
```

## Binary Selection

Choose one:

| OS | CPU | Binary |
|---|---|---|
| macOS | arm64 | `collector-bridge-darwin-arm64` |
| macOS | amd64 | `collector-bridge-darwin-amd64` |
| Windows | amd64 | `collector-bridge-windows-amd64.exe` |
| Linux | amd64 | `collector-bridge-linux-amd64` |

If the binary is missing:

1. Do not ask the user to install Go.
2. If network/download is available and authorized, download the prebuilt binary bundle from `https://raw.githubusercontent.com/soloagency/solo-agency/dist/collector-bridge-binaries-0.1.0.zip`.
3. If download is unavailable, log `collector_unavailable`.
4. Continue with public data sources and any previously collected private data.
5. Notify the user that the local collector binary is missing.

## Chrome Extension Per-Client Install

If `extensions/{client_slug}/manifest.json` does not exist, prepare it from the local extension template. Prefer the repo helper when available:

```bash
solo-agency-collector/scripts/prepare_client_extension.sh "{Client Name}" "{client_slug}" "{extension_instance_id}" "{ABSOLUTE_AGENCY_ROOT}"
```

If the helper is not available, copy `solo-agency-collector/chrome-extension/` to `extensions/{client_slug}/`, patch that client's `manifest.json` so the extension name starts with the client name, and create `client_binding.json`.

Then tell the user with the resolved absolute path, not a relative path:

```md
Please install the local collector extension for {Client Name}:

1. Open the Chrome profile/account for {Client Name}.
2. Go to `chrome://extensions`.
3. Turn on `Developer mode`.
4. Click `Load unpacked`.
5. Select this folder:
   `/ABSOLUTE/PATH/TO/extensions/{client_slug}/`

Important: if you also see `/ABSOLUTE/PATH/TO/solo-agency/solo-agency-collector/chrome-extension/`, do not select it. That is the toolkit/source copy. Select only the client folder under `extensions/{client_slug}/`.

After this one-time setup, keep that Chrome profile logged in to the private data sources approved for {Client Name}. The shared Local Collector app/bridge handles routing and local file output.
```

If the AI agent cannot run the bridge binary because it is sandboxed, create a ready-to-run setup file such as:

```text
/ABSOLUTE/PATH/TO/solo-agency-local-collector/setup_collector.sh
```

The generated setup file must be named `setup_collector.sh`. Do not invent alternative names. Every run must check and clear the collector port before starting the newest Local Collector app executable.

Then give the user exactly one short command:

```bash
bash "/ABSOLUTE/PATH/TO/solo-agency-local-collector/setup_collector.sh"
```

The setup file must be idempotent:

- Running it again must not overwrite `collector_config.json` if that file already exists.
- Running it again must not delete `inbox/`, client folders, history, outputs, reports, or collected data.
- Running it again may update only the Local Collector app executable files, and should replace/extract them only when the downloaded archive changed or local executable files are missing.
- It should install the Chrome extension files only if the local extension folder is missing or incomplete.
- It must stop any previous Local Collector app process for port `17321` when possible, then start the newest executable.
- Re-running the setup/start file must restart the Local Collector app, not merely try to start a second copy. Otherwise an old collector can keep port `17321`, causing the Chrome extension to keep talking to stale config and report "no job" even after the agent wrote a new client config.
- If the new bridge logs `address already in use`, the setup/start file is incomplete. Fix the script to detect the process holding port `17321` before asking the human to retry.
- The restart order must be (do not rely on `POST /shutdown`: the shipped bridge requires the per-run extension token, held only by the extension, and returns 401 when called tokenless, so a tokenless call is a no-op):
  1. Kill the PID stored in `collector.pid` if it is still alive.
  2. Use `lsof -tiTCP:17321 -sTCP:LISTEN` on macOS/Linux to find any remaining process holding the port.
  3. Kill only processes whose command line contains `collector-bridge`; if a non-collector process owns the port, stop and tell the human exactly what is blocking it.
  4. Start the newest Local Collector app executable in background/detached mode.
  5. Write the new PID to `collector.pid` and logs to `collector.log`.
- Keep PID/log files under `solo-agency-local-collector/`, for example `solo-agency-local-collector/collector.pid` and `solo-agency-local-collector/collector.log`.
- Start the Local Collector app in background/detached mode, write PID/log files, then return control to the user. Do not require the user to keep Terminal or PowerShell open during normal operation.
- Do not show the user a long multi-line script as the main instruction.

## Job File Shape

Create a job file under:

```text
daily-content-pipeline/collector/jobs/{run_id}.json
```

Minimum job:

```json
{
  "run_id": "2026-06-20_demo-client",
  "client_slug": "demo-client",
  "business_slug": "dui-law",
  "location_slug": "los-angeles",
  "pacing": {
    "min_delay_seconds": 5,
    "max_delay_seconds": 5,
    "max_sources": 20,
    "scroll_steps": 5,
    "max_text_chars": 12000
  },
  "sources": [
    {
      "name": "Example private data source",
      "url": "https://www.facebook.com/groups/example",
      "source_type": "private",
      "platform": "facebook",
      "priority": "high",
      "scan_cadence": "daily",
      "purpose": "lead_scan"
    }
  ]
}
```

## Shared Collector Config

Create `daily-content-pipeline/collector/collector_config.json` during setup if missing.

Default behavior:

- `run_mode`: `persistent_bridge_scheduler`
- `poll_interval_seconds`: `5`
- `max_scrolls_per_source`: `5`
- `max_scrolls_allowed`: `10`
- `scroll_delay_seconds`: `5`
- one daily collection window

If the user wants multiple daily runs, update the same config file instead of inventing another schedule format.

If Claude cannot run the binary from its sandbox, provide the user with a one-time command or OS startup service instructions so the bridge starts outside Claude.

## Start Bridge: Persistent Scheduler Mode

Use this mode for Claude, scheduled agents, or any environment where the human will not be present during collection.

```sh
solo-agency-local-collector/bin/collector-bridge-darwin-arm64 \
  --host 127.0.0.1 \
  --port 17321 \
  --config-file daily-content-pipeline/collector/collector_config.json \
  --output-dir daily-content-pipeline/collector/inbox \
  --persistent
```

In this mode:

- The bridge stays idle in the background.
- The extension should call `/status` immediately after install, browser startup, and settings save.
- The extension polls `/status` every few seconds while Chrome is active and the extension service worker is awake.
- If Chrome suspends the extension service worker, Chrome alarms are the fallback and the practical check interval may be about 1 minute.
- The bridge only returns a job inside a configured `scheduled_windows` time range.
- When the extension posts `/complete`, the bridge marks that scheduled run done and stays online for the next window.

## Manual Run / Run Now

Manual runs must not wait for a configured schedule window.

For first trials, test runs, "run now", "collect now", or any human-requested manual collection, enqueue a per-client job. If localhost is reachable, POST the job to:

```text
POST http://127.0.0.1:17321/jobs/run_now
```

The job should include a unique `run_id`, `run_now: true`, `force: false` by default, `run_now_ttl_minutes` (default 30, max 120), `sources`, `pacing`, `client_slug`, and `allowed_extension_instance_ids`. The bridge queues the job and the matching Chrome extension claims it on its next poll. The bridge runs jobs in parallel across different client identities and serializes only within the same client/profile; after `/complete` it moves to that client's next queued job.

If an AI sandbox cannot call the human machine's localhost endpoint directly but can write local files, use the queue directory. Write one atomic JSON file per client/run:

```text
daily-content-pipeline/collector/jobs/pending/{timestamp}_{client_slug}_{run_id}.json
```

Write each file atomically: write a temporary file in the same folder first, then rename it to a unique `.json` filename after the JSON is complete. Never have multiple agents write the same pending filename.

Legacy single-file fallback:

```text
daily-content-pipeline/collector/run_now_request.json
```

The bridge still checks this file during `/status`, but now converts the payload into queue files under `jobs/pending/`, writes `run_now_request_status.json`, and moves the request aside as `run_now_request.{batch_or_run_id}.{timestamp}.consumed.json`. A single file may contain either one job or a batch object with `{"jobs":[...]}`. Do not use the single file when multiple automation agents may write concurrently; two writers can still overwrite the same path before the bridge sees it. Use `jobs/pending/` unique files for multi-client and scheduled automation.

Do not simulate a manual run by editing `scheduled_windows` or creating a temporary schedule window. If an AI sandbox cannot call HTTP and cannot write the request file, create a local helper script/launcher as the last fallback. That helper must POST `/jobs/run_now`; it must not restart the bridge just to make a schedule edit take effect.

If a legacy collector without `/jobs/run_now` requires a temporary schedule fallback, clearly label it as a fallback, back up the config, use a short unique temporary window, restore the original config after completion/timeout, and report the fallback. Do not use this fallback when `/jobs/run_now` exists.

Use a new unique `run_id` for every manual run. Do not reuse the same run id with `force: true` as the default behavior. If `/complete` is never received, the bridge must stop exposing the active run-now job after its TTL expires and then continue with the next queued job.

After the extension posts `/complete`, the run-now job is cleared.

Config reload behavior:

- The bridge checks `collector_config.json` during `/status`.
- If the file timestamp or size changed, the bridge reloads config without restart.
- Use `POST /config` when available, but direct config file edits are acceptable for intentional recurring schedule updates when HTTP is unavailable.
- Manual runs must still use `/jobs/run_now` or per-client files under `jobs/pending/`, not schedule edits. `run_now_request.json` is only a legacy/batch shim that the bridge converts into queue files.

## Start Bridge: On-Demand Mode

Use this mode only when the agent can start a short-lived bridge for one collection run.

```sh
solo-agency-local-collector/bin/collector-bridge-darwin-arm64 \
  --host 127.0.0.1 \
  --port 17321 \
  --run-id 2026-06-20_demo-client \
  --job-file daily-content-pipeline/collector/jobs/2026-06-20_demo-client.json \
  --output-dir daily-content-pipeline/collector/inbox/2026-06/2026-06-20_demo-client \
  --ttl-minutes 30
```

In on-demand mode, the bridge auto-shuts down when the extension posts `/complete` or when TTL expires.

## Wait For Output

Poll:

```text
daily-content-pipeline/collector/inbox/YYYY-MM/{client_slug}/{run_id}/collector_status.json
```

Then read:

```text
private_data_points.jsonl
leads.jsonl
competitors.jsonl
new_private_sources.jsonl
source_status.jsonl
snapshots/
```

## Health Check

Before and after private collection, check:

```text
GET http://127.0.0.1:17321/status
```

Use the response to explain the collector state:

- Bridge reachable: bridge process is running.
- `config_file`, `output_dir`, and `run_now_request_file`: must point to the current setup's `daily-content-pipeline/collector/` tree before the bridge can be considered healthy for this workspace.
- `extension_health.status: recent`: Chrome extension checked in recently.
- `extension_health.status: stale`: bridge is running but extension has not checked in recently.
- `extension_health.status: no_extension_check_yet`: extension has not contacted this bridge instance yet.
- `bridge_health.json`: local health file written under `daily-content-pipeline/collector/inbox/`.

Do not treat `/status` as healthy merely because it is reachable or says `ready`. A user may have installed Solo Agency more than once, leaving an old bridge alive on port `17321`. If `/status.config_file`, `/status.output_dir`, or `/status.run_now_request_file` points outside the current setup folder, mark the state as `wrong_workspace_bridge`.

When `wrong_workspace_bridge` happens:

- do not create run-now jobs;
- do not write `run_now_request.json`;
- do not claim private data source monitoring is active;
- tell the human that the running Local Collector app belongs to a previous Solo Agency setup or another folder;
- give the human the current setup's one-line Local Collector setup/start command to run outside the AI sandbox, so it can stop the old `collector-bridge` process and restart with the current config/output paths;
- remind the human that one machine should have one active shared Solo Agency Local Collector runtime, and one client-specific extension per client Chrome profile/account loaded from `extensions/{client_slug}/`. If stale extension entries exist, the human should remove or disable only the stale entries in `chrome://extensions`.

After starting or restarting the bridge, wait and re-check `/status` for up to 75 seconds before reporting `no_extension_check_yet`, because Chrome's Manifest V3 service worker may be asleep until the next alarm.

If the bridge is offline, give the human the exact absolute-path start command to run outside the AI sandbox; the agent must not start it itself, even when it can run commands. If the extension is stale or missing, continue with public data sources, skip private data sources for this run, and notify the human through WideCast MCP / Telegram when available.

## Expected Extension Behavior

The extension will:

1. Detect the localhost bridge.
2. Fetch the current job.
3. Open configured sources in inactive Chrome tabs using the user's current login session.
4. Wait 5 seconds between major actions by default.
5. Send records back to the bridge.
6. Mark the run complete.

Chrome Manifest V3 cannot guarantee the background service worker stays awake forever. The extension uses immediate checks on install/startup/settings-save, Chrome alarms, and short polling while awake. If Chrome is closed, the computer is asleep, or the extension is disabled/removed, private collection cannot run.

Extension build `0.1.10-filtering-capture` and newer avoids `requestAnimationFrame` in automated collection so background tabs are less likely to pause forever, while still keeping the full `filtering.js`/readability capture pipeline. It gives social captures enough time for 5-10 scroll passes, times out truly stalled capture scripts, and clears stale active-run locks after an extension build update. If bridge and extension heartbeat are healthy but a source stays `started` with zero data points, audit the bridge/extension contract first: client identity headers, run ownership, write token, POST endpoint responses, and output folder routing. Then check `source_status.jsonl` for `capture_timeout_needs_visible_collector_window_or_site_access` or `inject_capture_files_timeout_needs_site_access`. This can also mean a hidden/frozen tab, login/checkpoint page, missing site access, or wrong Chrome profile. The safe operational fallback is a dedicated per-client collector Chrome profile/window, not an agent-controlled browser.

## Failure Handling

If the bridge cannot start:

- Log `collector_unavailable`.
- Continue with public data sources.
- Notify the user.

If the extension is not installed or does not respond:

- Log `extension_unavailable`.
- Continue with public data sources.
- Notify the user to install or enable the extension.

If Chrome is closed:

- Log `chrome_not_running`.
- Continue with public data sources.
- Notify the user to keep Chrome open for private data source collection.

If a platform warning, checkpoint, rate limit, or unusual activity prompt appears:

- Stop scanning that platform.
- Log `platform_warning` or `rate_limited`.
- Notify the user.

## Safety

- Do not ask for credentials.
- **Never read, store, or transmit the operator's own credentials or secrets** (usernames, passwords, cookies, tokens, session/auth data, API keys) — the single absolute red line.
- Collect and analyze whatever the operator has set up and directed — their own business data and the sources/prospects they point the collector at, including contact details (email/phone) — for lead-finding and email personalization. Operator setup + command = consent to read and combine that data.
- Do not message, comment, react, follow, or post (the send/act side needs separate explicit human approval).
- Do not bypass access controls or CAPTCHAs — read only what the operator's own session already renders.
- Do not upload private data to cloud services unless the user explicitly configures that.
