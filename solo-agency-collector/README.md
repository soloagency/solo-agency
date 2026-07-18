# Solo Agency Local Collector

This folder is not the Solo Agency setup entrypoint. AI agents must read `../SOLO_AGENCY_PLAYBOOK.md` first and install/start the Local Collector only when the main playbook reaches the private data source stage and the human approves collector activation.

This package contains the local private data source collector prototype for `SOLO_AGENCY_PLAYBOOK.md`.

It has two parts:

1. `bridge-go/`
   - Source code for the localhost bridge.
   - Build-time only.
   - End users do not need to install Go when prebuilt binaries are provided.

2. `chrome-extension/`
   - Chrome Extension MV3 collector.
   - Uses the user's existing logged-in Chrome session.
   - Polls the local bridge on `127.0.0.1`.
   - Sends visible, authorized page data to local files through the bridge.

Writing skills are intentionally outside this collector package. AI agents should load account-free writing skills from `../playbooks/skills/` before requiring any external writing-skill API or account setup.

## End-User Model

The normal user should not need Go, Node.js, Python, Playwright, npm, pip, or a package manager.

Agents should use the Solo Agency GitHub repo as the default setup source. Until GitHub Releases and Chrome Web Store are available, use these raw GitHub artifact URLs:

```text
https://raw.githubusercontent.com/soloagency/solo-agency/dist/collector-bridge-binaries-0.1.0.zip
https://raw.githubusercontent.com/soloagency/solo-agency/dist/chrome-extension-collector-root-0.1.0.zip
https://raw.githubusercontent.com/soloagency/solo-agency/dist/SHA256SUMS
```

The package ships prebuilt bridge binaries through the artifact bundle:

```text
solo-agency-local-collector/bin/
  collector-bridge-darwin-arm64
  collector-bridge-darwin-amd64
  collector-bridge-windows-amd64.exe
  collector-bridge-linux-amd64
```

The AI agent chooses the right binary for the user's OS and CPU.

## Maintainer Build / Pre-Upload

From the repo root, run:

```sh
./deploy-soloagency.sh
```

This rebuilds the Go bridge binaries for macOS arm64, macOS amd64, Linux amd64, and Windows amd64; rebuilds the Chrome extension zip artifacts; refreshes `dist/SHA256SUMS`; runs Go tests; and runs upload preflight checks. Use `./deploy-soloagency.sh --collector-only` when only collector artifacts need to be refreshed.

Recommended unattended mode:

- Start the bridge in persistent scheduler mode.
- Let the Chrome extension check it immediately after install/startup/settings-save, then poll every few seconds while Chrome is active.
- If Chrome suspends the extension service worker, expect the Chrome alarm fallback to check roughly within about 1 minute.
- Let the bridge return a job only inside configured collection windows.
- Let `/complete` mark that window done without shutting down the bridge.

Manual run mode:

- Human-requested runs must not wait for a configured schedule window.
- AI agents should POST a run-now job to `http://127.0.0.1:17321/jobs/run_now`.
- If an AI sandbox cannot call the local HTTP endpoint but can write local files, it should write one unique per-client job payload under `daily-content-pipeline/collector/jobs/pending/`.
- `daily-content-pipeline/collector/run_now_request.json` is a legacy/batch shim only. The bridge converts one job or `{"jobs":[...]}` into per-client queue files, writes `run_now_request_status.json`, moves the shim aside as `run_now_request.{run_id}.{timestamp}.consumed.json`, and keeps an in-memory signature guard if moving/removing fails.
- The matching client extension will pick up its queued run-now job on the next `/status` poll. The shared bridge can expose one active private collector job per client identity, bound to the claiming extension instance, so different client Chrome profiles can collect in parallel. Jobs for the same client/profile remain sequential and move to the next queued job after `/complete` or TTL.
- Every run-now job should have a unique `run_id`, `force: false` by default, and a TTL so it cannot remain active all day if `/complete` is not received.
- `/complete` clears the run-now job so it does not repeat.
- Do not simulate manual collection by editing `scheduled_windows`.

Config reload:

- The bridge checks `collector_config.json` on `/status` and reloads it when the file timestamp or size changes.
- AI agents may still prefer `POST /config`, but direct file edits are allowed for intentional recurring schedule updates when HTTP is unavailable.
- Manual run-now collection should use `/jobs/run_now` or per-client `jobs/pending/` queue files, not temporary schedule windows.

Browser limits:

- Chrome must be open for private collection to run.
- Manifest V3 service workers can sleep; the extension uses immediate checks, Chrome alarms, and short polling while awake.
- After starting the bridge, wait up to 75 seconds before concluding the extension has not checked in.
- Collection uses inactive tabs and closes collector-created tabs after scanning when configured, but a real tab/page context is still needed to read logged-in private web pages.

For Claude or other agents that cannot run local binaries from their sandbox, run the bridge in persistent scheduler mode outside the AI sandbox. Use:

- macOS LaunchAgent
- Windows Task Scheduler
- Linux `systemd --user`

The shared config file is:

```text
daily-content-pipeline/collector/collector_config.json
```

Default collection behavior:

- one run per day
- bridge poll interval: 5 seconds while Chrome is active
- 5 scrolls per private data source
- 5 seconds between scrolls
- maximum user-configurable scrolls: 10

The Chrome extension is installed manually once for now:

1. Open Chrome.
2. Go to `chrome://extensions`.
3. Turn on Developer Mode.
4. Click `Load unpacked`.
5. Select the extracted absolute folder path, for example `/Users/alex/oneman_agency/solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME/`.

Do not load the source folder from a cloned toolkit, such as `solo-agency/solo-agency-collector/chrome-extension/`, for a normal agency setup. The toolkit folder is for development. The running agency should load only the `solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME/` runtime copy.

Use one active Solo Agency Local Collector bridge runtime per machine for the current setup. Use one client-specific Solo Agency Local Collector extension per client Chrome profile/account, loaded from the current setup's generated `extensions/{client_slug}/` folder. If you previously loaded another Solo Agency Local Collector extension from an older setup folder, remove or disable that old entry in `chrome://extensions`.

## Developer Model

Maintainers build the bridge from source:

```sh
cd solo-agency-collector/bridge-go
go build -o ../../solo-agency-local-collector/bin/collector-bridge ./...
```

Cross-compile examples:

```sh
GOOS=darwin GOARCH=arm64 go build -o ../../solo-agency-local-collector/bin/collector-bridge-darwin-arm64 ./...
GOOS=darwin GOARCH=amd64 go build -o ../../solo-agency-local-collector/bin/collector-bridge-darwin-amd64 ./...
GOOS=windows GOARCH=amd64 go build -o ../../solo-agency-local-collector/bin/collector-bridge-windows-amd64.exe ./...
GOOS=linux GOARCH=amd64 go build -o ../../solo-agency-local-collector/bin/collector-bridge-linux-amd64 ./...
```

## Local-Only Design

- The bridge binds only to `127.0.0.1`.
- The extension stays idle when the bridge is not running.
- The extension does not ask for passwords, OTPs, cookies, or tokens.
- The bridge writes JSONL and HTML snapshots locally.
- No private data is uploaded to a server by default.

## Persistent Scheduler Example

When using the generated `setup_collector.sh` / PowerShell setup file, running it again should restart the Local Collector app instead of launching a second copy. The script should call `/shutdown` when available, stop the PID in `collector.pid`, then inspect port `17321` and stop only old `collector-bridge` processes before starting the newest executable. This prevents Chrome from talking to a stale bridge instance that has old config and returns "no job".

```sh
solo-agency-local-collector/bin/collector-bridge-darwin-arm64 \
  --host 127.0.0.1 \
  --port 17321 \
  --config-file daily-content-pipeline/collector/collector_config.json \
  --output-dir daily-content-pipeline/collector/inbox \
  --persistent
```

## On-Demand Example Run

```sh
solo-agency-local-collector/bin/collector-bridge-darwin-arm64 \
  --host 127.0.0.1 \
  --port 17321 \
  --run-id 2026-06-20_demo-client \
  --job-file solo-agency-collector/examples/job.sample.json \
  --output-dir daily-content-pipeline/collector/inbox/2026-06/2026-06-20_demo-client \
  --ttl-minutes 30
```

After the bridge starts, the Chrome extension detects it, fetches the job, collects the configured sources, and writes results through the bridge.

## Output Files

```text
daily-content-pipeline/collector/inbox/
  bridge_health.json

daily-content-pipeline/collector/inbox/YYYY-MM/{run_id}/
  collector_status.json
  private_data_points.jsonl
  leads.jsonl
  competitors.jsonl
  new_private_sources.jsonl
  source_status.jsonl
  snapshots/
```

AI agents read these files and continue with filtering, lead detection, competitor detection, idea generation, WideCast-writing-skill drafts, HTML reports, and notifications.

AI agents can also call `GET http://127.0.0.1:17321/status` to check bridge and extension health. The status includes `extension_health.last_extension_check_at`, `extension_health.seconds_since_last_check`, and `extension_health.status`.

Agents must also verify that `/status.config_file`, `/status.output_dir`, and `/status.run_now_request_file` point to the current setup's `daily-content-pipeline/collector/` tree. A reachable bridge may belong to an older setup. If those paths point elsewhere, treat it as `wrong_workspace_bridge`, ask the human to run the current setup's Local Collector setup/start command, and do not collect private data until the bridge writes to the current workspace.
