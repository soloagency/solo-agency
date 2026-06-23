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

That file must say whether the Solo Agency Local Collector extension and Local Collector app are `pending_private_activation`, `activation_declined_for_now`, `installed_and_running`, waiting for a specific human action, blocked by sandbox/OS permission, stale, or offline.

## Required Inputs

- A client pipeline folder.
- A collector job JSON file.
- A prebuilt bridge binary for the current OS/CPU, downloaded from the Solo Agency GitHub distribution if it is not already present.
- The Solo Agency Local Collector extension installed in the user's logged-in Chrome profile.

## Solo Agency GitHub Distribution

Use the Solo Agency GitHub repo as the default setup source. Until GitHub Releases and Chrome Web Store are available, use these raw GitHub artifact URLs:

```text
https://raw.githubusercontent.com/soloagency/solo-agency/main/solo-agency-collector/dist/collector-bridge-binaries-0.1.0.zip
https://raw.githubusercontent.com/soloagency/solo-agency/main/solo-agency-collector/dist/chrome-extension-collector-root-0.1.0.zip
https://raw.githubusercontent.com/soloagency/solo-agency/main/solo-agency-collector/dist/SHA256SUMS
```

The agent should:

1. Resolve the absolute agency root that contains both `daily-content-pipeline/` and `solo-agency-local-collector/`.
2. Download the runtime files into the absolute `solo-agency-local-collector/downloads/` folder.
3. Verify checksums when tools are available.
4. Extract bridge binaries into the absolute `solo-agency-local-collector/bin/` folder.
5. Extract the Chrome extension into the absolute `solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME/` folder.
6. Select the correct bridge binary for the user's OS/CPU.
7. Ask for one-time human approval if the current AI environment requires permission before running a downloaded executable.
8. Prefer persistent scheduler mode for unattended private data source collection.
9. Use on-demand mode only when the AI agent can safely start the bridge for a single run.

The user still needs to install the Chrome extension once using Chrome Developer Mode until the extension is available in Chrome Web Store.

## Runtime Folder

```text
solo-agency-local-collector/
  downloads/
  bin/
  LOAD_THIS_EXTENSION_IN_CHROME/
  setup_collector.sh
  collector.pid
  collector.log
daily-content-pipeline/collector/
  collector_setup_status.md
  collector_config.json
  jobs/
  inbox/
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
2. If network/download is available and authorized, download the prebuilt binary bundle from `https://raw.githubusercontent.com/soloagency/solo-agency/main/solo-agency-collector/dist/collector-bridge-binaries-0.1.0.zip`.
3. If download is unavailable, log `collector_unavailable`.
4. Continue with public data sources and any previously collected private data.
5. Notify the user that the local collector binary is missing.

## Chrome Extension One-Time Install

If `solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME/manifest.json` does not exist, download and extract:

```text
https://raw.githubusercontent.com/soloagency/solo-agency/main/solo-agency-collector/dist/chrome-extension-collector-root-0.1.0.zip
```

Then tell the user with the resolved absolute path, not a relative path:

```md
Please install the local collector extension once:

1. Open Chrome.
2. Go to `chrome://extensions`.
3. Turn on `Developer mode`.
4. Click `Load unpacked`.
5. Select this folder:
   `/ABSOLUTE/PATH/TO/solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME/`

Important: if you also see `/ABSOLUTE/PATH/TO/solo-agency/solo-agency-collector/chrome-extension/`, do not select it. That is the toolkit/source copy. The only folder to load for the running agency is the `solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME/` folder above.

After this one-time setup, keep Chrome open and logged in to the private data sources you want monitored. The local bridge will run in persistent scheduler mode or I will start it during collection runs when this AI environment allows.
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
- The restart order must be:
  1. `POST http://127.0.0.1:17321/shutdown` when `curl` or an HTTP client is available.
  2. Kill the PID stored in `collector.pid` if it is still alive.
  3. Use `lsof -tiTCP:17321 -sTCP:LISTEN` on macOS/Linux to find any remaining process holding the port.
  4. Kill only processes whose command line contains `collector-bridge`; if a non-collector process owns the port, stop and tell the human exactly what is blocking it.
  5. Start the newest Local Collector app executable in background/detached mode.
  6. Write the new PID to `collector.pid` and logs to `collector.log`.
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

For first trials, test runs, "run now", "collect now", or any human-requested manual collection, POST a job to:

```text
POST http://127.0.0.1:17321/jobs/run_now
```

The job should include a unique `run_id`, `run_now: true`, `force: false` by default, `run_now_ttl_minutes` (default 30, max 120), `sources`, `pacing`, and client metadata. The bridge will expose it immediately through `/status` and `/jobs/current`; the Chrome extension should pick it up on its next poll.

If an AI sandbox cannot call the human machine's localhost endpoint directly but can write local files, write the same payload to:

```text
daily-content-pipeline/collector/run_now_request.json
```

Write the file atomically: write a temporary file in the same folder first, then rename it to `run_now_request.json` after the JSON is complete.

The bridge checks this file during `/status`, loads it as a run-now job, writes `run_now_request_status.json`, moves it aside as `run_now_request.{run_id}.{timestamp}.consumed.json`, and keeps an in-memory signature guard if moving/removing fails. This is the preferred fallback because it does not require the human to run another command and it prevents the same file from being replayed forever.

Do not simulate a manual run by editing `scheduled_windows` or creating a temporary schedule window. If an AI sandbox cannot call HTTP and cannot write the request file, create a local helper script/launcher as the last fallback. That helper must POST `/jobs/run_now`; it must not restart the bridge just to make a schedule edit take effect.

If a legacy collector without `/jobs/run_now` requires a temporary schedule fallback, clearly label it as a fallback, back up the config, use a short unique temporary window, restore the original config after completion/timeout, and report the fallback. Do not use this fallback when `/jobs/run_now` exists.

Use a new unique `run_id` for every manual run. Do not reuse the same run id with `force: true` as the default behavior. If `/complete` is never received, the bridge must stop exposing the run-now job after its TTL expires.

After the extension posts `/complete`, the run-now job is cleared.

Config reload behavior:

- The bridge checks `collector_config.json` during `/status`.
- If the file timestamp or size changed, the bridge reloads config without restart.
- Use `POST /config` when available, but direct config file edits are acceptable for intentional recurring schedule updates when HTTP is unavailable.
- Manual runs must still use `/jobs/run_now` or `run_now_request.json`, not schedule edits.

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
daily-content-pipeline/collector/inbox/YYYY-MM/{run_id}/collector_status.json
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
- `extension_health.status: recent`: Chrome extension checked in recently.
- `extension_health.status: stale`: bridge is running but extension has not checked in recently.
- `extension_health.status: no_extension_check_yet`: extension has not contacted this bridge instance yet.
- `bridge_health.json`: local health file written under `daily-content-pipeline/collector/inbox/`.

After starting or restarting the bridge, wait and re-check `/status` for up to 75 seconds before reporting `no_extension_check_yet`, because Chrome's Manifest V3 service worker may be asleep until the next alarm.

If the bridge is offline and the agent can run commands, start the bridge. If the agent is sandboxed, provide the human with the absolute-path command. If the extension is stale or missing, continue with public data sources, skip private data sources for this run, and notify the human through WideCast MCP / Telegram when available.

## Expected Extension Behavior

The extension will:

1. Detect the localhost bridge.
2. Fetch the current job.
3. Open configured sources in inactive Chrome tabs using the user's current login session.
4. Wait 5 seconds between major actions by default.
5. Send records back to the bridge.
6. Mark the run complete.

Chrome Manifest V3 cannot guarantee the background service worker stays awake forever. The extension uses immediate checks on install/startup/settings-save, Chrome alarms, and short polling while awake. If Chrome is closed, the computer is asleep, or the extension is disabled/removed, private collection cannot run.

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
- Do not read cookies directly.
- Do not scrape contact details.
- Do not message, comment, react, follow, or post.
- Do not bypass access controls.
- Do not upload private data to cloud services unless the user explicitly configures that.
