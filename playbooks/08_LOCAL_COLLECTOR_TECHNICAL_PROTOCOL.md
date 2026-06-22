# Local Collector Technical Protocol

Stage: `08`

## Load Rule

Load when installing, starting, stopping, checking, scheduling, updating, or troubleshooting the Solo Agency Local Collector extension or Local Collector app.

## Hard Gates For This Stage

- Default localhost is `127.0.0.1:17321`.
- Use `GET /status` for health checks.
- Use `POST /jobs/run_now` or `run_now_request.json` for manual/private run-now jobs.
- Do not fake extension health by sending extension-only headers from the AI agent.
- Setup scripts must preserve data/config and stop only old collector processes occupying port 17321.
- No credentials, hidden APIs, DMs, inboxes, account pages, or contact scraping.

## Source Preservation Rule

This file is detailed source material moved from the original monolithic `SOLO_AGENCY_PLAYBOOK.md`.

Do not summarize away requirements, examples, checklists, schemas, protocols, URLs, edge cases, warnings, approval gates, or completion gates. If a downstream agent needs to shorten its response to the human, it may summarize the response, but it must still obey the full requirements in this file.

---

## Latest Delta Override: Discovery Mode Versus Daily Monitoring

Source Discovery Mode must scroll deeply until no new source names or URLs appear for 3 consecutive scrolls, with a hard safety cap such as 80 scrolls.

Daily Content Monitoring Mode keeps the conservative default: 5 scrolls, max 10, 5 seconds between scrolls, and about 20 private sources or fewer per client.

Do not apply the daily 5-scroll default to source discovery.

## Scan Depth Disclosure Rule

Whenever the agent announces that it will scan groups, communities, fanpages, social profiles, or other private/logged-in sources, it must disclose the scan depth in plain language.

For daily content monitoring, say:

```text
I will go through each approved group/source one by one and scroll {N} times per source. I will read {N} from the Local Collector configuration when available; otherwise I will use the safe default of 5 scrolls, max 10, with about 5 seconds between scrolls.
```

To resolve `{N}`, use this order:

1. Read `daily-content-pipeline/collector/collector_config.json`.
2. If the Local Collector app is running, call `GET http://127.0.0.1:17321/status` and/or `GET /config` when available.
3. Fall back to `5` only when config cannot be read.

For source discovery, disclose the different rule:

```text
This is source discovery, not daily monitoring. I will scroll until no new source names/URLs appear for 3 consecutive scrolls, with a hard safety cap such as 80 scrolls.
```

Do not let the human think "scan groups" is unbounded or vague.

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
- Claude Chrome Extension must not be used for automated private-source collection because it may require the human to click Allow during runs and can stop an unattended schedule.
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
https://raw.githubusercontent.com/soloagency/solo-agency/main/solo-agency-collector/dist/collector-bridge-binaries-0.1.0.zip
https://raw.githubusercontent.com/soloagency/solo-agency/main/solo-agency-collector/dist/chrome-extension-collector-root-0.1.0.zip
https://raw.githubusercontent.com/soloagency/solo-agency/main/solo-agency-collector/dist/SHA256SUMS
```

Current writing-skill artifacts:

```text
https://raw.githubusercontent.com/soloagency/solo-agency/main/solo-agency-collector/skills/video-script-writing.zip
https://raw.githubusercontent.com/soloagency/solo-agency/main/solo-agency-collector/skills/blog-writing.zip
https://raw.githubusercontent.com/soloagency/solo-agency/main/solo-agency-collector/skills/social-post-writing.zip
```

If the agent is already running inside a cloned copy of `https://github.com/soloagency/solo-agency`, it must prefer local repo files under `solo-agency-collector/dist/` and `solo-agency-collector/skills/` before downloading the same files from raw GitHub URLs.

The AI agent should install the collector locally as much as its environment allows.

Canonical local layout:

```text
{agency_root}/
  solo-agency/                         # downloaded toolkit/source repo
  solo-agency-local-collector/         # runtime app + Chrome extension only
    downloads/
    bin/
    LOAD_THIS_EXTENSION_IN_CHROME/
    setup_collector.sh
    collector.pid
    collector.log
  daily-content-pipeline/              # data/config/output only
    collector/
      collector_setup_status.md
      collector_config.json
      jobs/
      inbox/
```

Chrome extension folder disambiguation:

- There may be another `chrome-extension/` folder inside the downloaded toolkit/repo, such as `solo-agency/solo-agency-collector/chrome-extension/`.
- That toolkit folder is source/developer material. It is not the human-facing Chrome `Load unpacked` folder during agency setup.
- The only folder the agent may tell a normal human to load in Chrome is the runtime folder under `solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME/`.
- If both folders exist, the agent must explicitly warn: `Do not load the extension folder inside solo-agency/solo-agency-collector. Load only the solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME folder shown below.`
- The generated setup instructions, setup status file, and chat message must show only one Chrome extension path: the absolute `solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME/` path.
- Do not put app binaries, downloaded zips, the unpacked extension, PID files, or collector logs inside `daily-content-pipeline/`. That folder should remain data/config/output only.

Install flow:

1. Detect the user's OS and CPU architecture.
2. Check whether `solo-agency-collector/dist/` already exists locally from a cloned repo.
3. If local artifacts exist, copy `SHA256SUMS`, `collector-bridge-binaries-0.1.0.zip`, and `chrome-extension-collector-root-0.1.0.zip` from the local repo.
4. If local artifacts do not exist, download them from the raw GitHub URLs above.
5. Verify checksums when the environment has checksum tools available.
6. Extract bridge binaries into the absolute runtime path for `solo-agency-local-collector/bin/`.
7. Extract the Chrome extension zip into the absolute runtime path for `solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME/`.
8. Select the correct bridge binary for the current machine.
9. On macOS/Linux, ensure the selected binary is executable.
10. Ask for one-time human approval if the AI environment requires permission before running a downloaded executable.
11. Prefer persistent scheduler mode for unattended collection, or run the selected bridge binary only when a collection job starts if using on-demand mode.
12. In on-demand mode, stop the bridge after the job completes or let it auto-shutdown by TTL. In persistent mode, keep the bridge running and let `/complete` mark only the current window done.

Absolute path rule:

- The AI agent must never tell the human to load the Chrome extension from a relative path.
- The AI agent must resolve and show the absolute folder path.
- The AI agent must never show `daily-content-pipeline/collector/chrome-extension/` or `solo-agency/solo-agency-collector/chrome-extension/` as the folder for a normal human to load in Chrome. The first path belongs to the old mixed data/runtime layout; the second path is for source/development only.
- Correct examples:
  - macOS/Linux: `/Users/alex/oneman_agency/solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME/`
  - Windows: `C:\Users\Alex\oneman_agency\solo-agency-local-collector\LOAD_THIS_EXTENSION_IN_CHROME\`
- Incorrect examples:
  - `daily-content-pipeline/collector/chrome-extension/`
  - `solo-agency/solo-agency-collector/chrome-extension/`

Binary selection:

| OS | CPU | Binary |
|---|---|---|
| macOS | arm64 / Apple Silicon | `collector-bridge-darwin-arm64` |
| macOS | amd64 / Intel | `collector-bridge-darwin-amd64` |
| Windows | amd64 / x64 | `collector-bridge-windows-amd64.exe` |
| Linux | amd64 / x64 | `collector-bridge-linux-amd64` |

If the current OS/CPU is not listed, the agent must log `collector_unavailable`, continue with public sources, and notify the human that a compatible collector binary is not available yet.

Chrome extension installation flow:

1. The agent downloads and extracts the extension into an absolute path, for example:

```text
/Users/alex/oneman_agency/solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME/
```

2. The agent tells the human directly in chat, Telegram, or another human-facing channel:

```md
Please install the Solo Agency Local Collector extension once:

1. Open Chrome.
2. Go to `chrome://extensions`.
3. Turn on `Developer mode`.
4. Click `Load unpacked`.
5. Select this folder:
   `/ABSOLUTE/PATH/TO/solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME/`

Important: if you also see a folder named `solo-agency/solo-agency-collector/chrome-extension`, do not select that one. That is the toolkit/source copy. Select only the `solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME` folder above.

After this one-time setup, you may close this instruction tab whenever you want. For private-source collection to work at scheduled times, Chrome should be open and logged in to the private sources, and the Local Collector app should be running or configured to auto-start.
```

3. The agent must not ask for passwords, cookies, OTPs, or credentials.
4. If the extension is not installed or cannot contact the Local Collector app, the agent logs `extension_unavailable`, continues with public sources, and notifies the human.

If the AI agent cannot run the downloaded Local Collector app itself, it must create a ready-to-run script file and give the human exactly one short command to paste into Terminal or PowerShell.

Do not show the human a long multi-line script as the primary instruction. Non-technical humans should not have to copy a large code block.

Do not tell the human that setup instructions are only in a Markdown file. The Markdown file may store the same information for agent memory, but the current chat must contain the exact action the human should take.

The generated collector setup script must be named `setup_collector.sh`. Do not invent alternative names such as `start_local_collector.sh`. Every run must check who owns the collector port before starting a new Local Collector app.

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
- The restart order must be: call `POST /shutdown` when possible, stop the PID in `collector.pid` if alive, inspect the process holding port `17321`, kill only collector processes such as `collector-bridge`, then start the newest executable and write a fresh PID/log. If a non-collector process owns the port, stop and show the human the blocking command instead of killing unrelated software.
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
EXTENSION_DIR="$COLLECTOR_RUNTIME_ROOT/LOAD_THIS_EXTENSION_IN_CHROME"
BASE_URL="https://raw.githubusercontent.com/soloagency/solo-agency/main/solo-agency-collector"
BRIDGE_ZIP_URL="$BASE_URL/dist/collector-bridge-binaries-0.1.0.zip"
EXTENSION_ZIP_URL="$BASE_URL/dist/chrome-extension-collector-root-0.1.0.zip"
PORT="17321"
CONFIG_FILE="$COLLECTOR_DATA_ROOT/collector_config.json"
PID_FILE="$COLLECTOR_RUNTIME_ROOT/collector.pid"
LOG_FILE="$COLLECTOR_RUNTIME_ROOT/collector.log"

mkdir -p "$COLLECTOR_RUNTIME_ROOT/downloads" "$COLLECTOR_RUNTIME_ROOT/bin" "$EXTENSION_DIR" "$COLLECTOR_DATA_ROOT/inbox" "$COLLECTOR_DATA_ROOT/jobs"

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

if [ ! -f "$EXTENSION_DIR/manifest.json" ]; then
  echo "Installing Solo Agency Local Collector extension files..."
  curl -L -o "$COLLECTOR_RUNTIME_ROOT/downloads/chrome-extension-collector-root-0.1.0.zip.tmp" "$EXTENSION_ZIP_URL"
  mv "$COLLECTOR_RUNTIME_ROOT/downloads/chrome-extension-collector-root-0.1.0.zip.tmp" "$COLLECTOR_RUNTIME_ROOT/downloads/chrome-extension-collector-root-0.1.0.zip"
  unzip -o "$COLLECTOR_RUNTIME_ROOT/downloads/chrome-extension-collector-root-0.1.0.zip" -d "$EXTENSION_DIR"
else
  echo "Keeping existing Solo Agency Local Collector extension folder unchanged."
fi

if [ ! -f "$CONFIG_FILE" ]; then
  echo "Creating default collector_config.json..."
  cat > "$CONFIG_FILE.tmp" <<'JSON'
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
  if command -v curl >/dev/null 2>&1; then
    curl -fsS -m 2 -X POST "http://127.0.0.1:$PORT/shutdown" >/dev/null 2>&1 || true
  fi

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

echo "Install the Chrome extension from this ONE absolute folder:"
echo "$EXTENSION_DIR"
echo "Do NOT load any chrome-extension folder under solo-agency/solo-agency-collector; that is the toolkit/source copy."
echo "Starting the Local Collector app in the background with the newest executable."
nohup "$BRIDGE" --host 127.0.0.1 --port "$PORT" --config-file "$CONFIG_FILE" --output-dir "$COLLECTOR_DATA_ROOT/inbox" --persistent >> "$LOG_FILE" 2>&1 &
BRIDGE_PID="$!"
echo "$BRIDGE_PID" > "$PID_FILE"
echo "Local Collector app started. PID: $BRIDGE_PID"
echo "Log file: $LOG_FILE"
echo "You can close this Terminal window now."
```

Then tell the human only this one-line command, with the real absolute path:

```bash
bash "/ABSOLUTE/PATH/TO/solo-agency-local-collector/setup_collector.sh"
```

Human-facing wording:

```md
I created a setup file for you. Please open Terminal, paste this one line, and press Enter:

`bash "/ABSOLUTE/PATH/TO/solo-agency-local-collector/setup_collector.sh"`

After it starts, you can close this instruction tab and Terminal window. The Local Collector app runs in the background. If you need troubleshooting later, I will check the local status endpoint and the collector log file.
```

Windows:

Do not tell Windows users to run bash. On Windows, use PowerShell or create a `.cmd` launcher that the human can double-click.

Important Windows note:

- The human can run an `.exe`, but double-clicking `collector-bridge-windows-amd64.exe` by itself is not enough for the recommended persistent setup because the app needs configuration arguments.
- The AI agent should create a friendly launcher such as `Start Local Collector.cmd` and, if needed, a setup script such as `setup_local_collector.ps1`.
- The human-facing instruction should be one action: either double-click `Start Local Collector.cmd` or paste one short PowerShell command that runs the prepared script.
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
$ExtensionDir = Join-Path $CollectorRuntimeRoot "LOAD_THIS_EXTENSION_IN_CHROME"
$BaseUrl = "https://raw.githubusercontent.com/soloagency/solo-agency/main/solo-agency-collector"
$BridgeZipUrl = "$BaseUrl/dist/collector-bridge-binaries-0.1.0.zip"
$ExtensionZipUrl = "$BaseUrl/dist/chrome-extension-collector-root-0.1.0.zip"
$Port = 17321
$ConfigPath = Join-Path $CollectorDataRoot "collector_config.json"
$PidPath = Join-Path $CollectorRuntimeRoot "collector.pid"
$LogPath = Join-Path $CollectorRuntimeRoot "collector.out.log"
$ErrLogPath = Join-Path $CollectorRuntimeRoot "collector.err.log"

New-Item -ItemType Directory -Force -Path `
  (Join-Path $CollectorRuntimeRoot "downloads"), `
  (Join-Path $CollectorRuntimeRoot "bin"), `
  $ExtensionDir, `
  (Join-Path $CollectorDataRoot "inbox"), `
  (Join-Path $CollectorDataRoot "jobs") | Out-Null

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

$ExtensionManifest = Join-Path $ExtensionDir "manifest.json"
if (-not (Test-Path $ExtensionManifest)) {
  Write-Host "Installing Solo Agency Local Collector extension files..."
  $ExtensionZipTmp = Join-Path $CollectorRuntimeRoot "downloads\chrome-extension-collector-root-0.1.0.zip.tmp"
  $ExtensionZip = Join-Path $CollectorRuntimeRoot "downloads\chrome-extension-collector-root-0.1.0.zip"
  Invoke-WebRequest -Uri $ExtensionZipUrl -OutFile $ExtensionZipTmp
  Move-Item -Force $ExtensionZipTmp $ExtensionZip
  Expand-Archive -Force $ExtensionZip $ExtensionDir
} else {
  Write-Host "Keeping existing Solo Agency Local Collector extension folder unchanged."
}

if (-not (Test-Path $ConfigPath)) {
  Write-Host "Creating default collector_config.json..."
  @'
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
'@ | Set-Content -Encoding UTF8 $ConfigPath
} else {
  Write-Host "Keeping existing collector_config.json unchanged."
}

$Bridge = Join-Path $CollectorRuntimeRoot "bin\collector-bridge-windows-amd64.exe"

try {
  Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:$Port/shutdown" -TimeoutSec 2 | Out-Null
} catch {
  # Existing bridge may not be running yet. Continue.
}

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

Write-Host "Install the Solo Agency Local Collector extension from this folder:"
Write-Host $ExtensionDir
Write-Host "Do NOT load any chrome-extension folder under solo-agency\solo-agency-collector; that is the toolkit/source copy."
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

Then tell the human one short PowerShell command:

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
powershell -NoProfile -ExecutionPolicy Bypass -Command "try { Invoke-RestMethod -Method Post -Uri 'http://127.0.0.1:17321/shutdown' -TimeoutSec 2 | Out-Null } catch {}; if (Test-Path '%PID_FILE%') { $p = Get-Content '%PID_FILE%' -ErrorAction SilentlyContinue | Select-Object -First 1; if ($p) { Stop-Process -Id $p -Force -ErrorAction SilentlyContinue }; Remove-Item '%PID_FILE%' -Force -ErrorAction SilentlyContinue }; try { Get-NetTCPConnection -LocalPort 17321 -State Listen -ErrorAction Stop | Select-Object -ExpandProperty OwningProcess -Unique | ForEach-Object { $proc = Get-Process -Id $_ -ErrorAction SilentlyContinue; $path = ''; try { $path = $proc.Path } catch {}; if ($proc -and (($proc.ProcessName -like '*collector-bridge*') -or ($path -like '*collector-bridge*'))) { Stop-Process -Id $_ -Force -ErrorAction SilentlyContinue } else { Write-Host ('Port 17321 is used by a non-collector process: ' + $proc.ProcessName + ' ' + $path); exit 1 } } } catch {}"
if errorlevel 1 exit /b 1
powershell -NoProfile -ExecutionPolicy Bypass -Command "$p = Start-Process -FilePath '%COLLECTOR_RUNTIME_ROOT%\bin\collector-bridge-windows-amd64.exe' -ArgumentList @('--host','127.0.0.1','--port','17321','--config-file','%COLLECTOR_DATA_ROOT%\collector_config.json','--output-dir','%COLLECTOR_DATA_ROOT%\inbox','--persistent') -RedirectStandardOutput '%LOG_FILE%' -RedirectStandardError '%ERR_LOG_FILE%' -WindowStyle Hidden -PassThru; Set-Content -Encoding ASCII -Path '%PID_FILE%' -Value $p.Id; Write-Host ('Local Collector app started. PID: ' + $p.Id); Write-Host 'You can close this window now.'"
```

Human-facing Windows wording:

```md
I created a setup file for you. Please open PowerShell, paste this one line, and press Enter:

`powershell -ExecutionPolicy Bypass -File "C:\ABSOLUTE\PATH\TO\solo-agency-local-collector\setup_local_collector.ps1"`

After setup, you can start the Local Collector app later by double-clicking:
`C:\ABSOLUTE\PATH\TO\solo-agency-local-collector\Start Local Collector.cmd`
```

Future update rule:

- When the project moves from raw GitHub files to GitHub Releases, replace the raw artifact URLs with GitHub release URLs.
- When the extension is published to Chrome Web Store, replace the developer-mode `Load unpacked` flow with the Chrome Web Store install flow.
- Until then, the AI agent should handle download, extraction, binary selection, on-demand bridge start/stop, or persistent startup-service setup automatically, while the human performs only the one-time Chrome extension installation approval.

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
- 5 scrolls per private source.
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
  - new private sources detected,
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
   - record `status.counts`,
   - inspect `status.extension_health`.
3. If `extension_health.status` is `recent`, private collection infrastructure is currently healthy.
4. If `extension_health.status` is `no_extension_check_yet` immediately after extension install, bridge restart, or settings save, wait and re-check for up to 75 seconds before declaring private collection unavailable.
5. If `extension_health.status` is `stale` or `no_extension_check_yet` after the 75-second grace window, treat private collection as unavailable for now and identify likely causes:
   - Chrome is closed,
   - extension is not installed,
   - extension is disabled or removed,
   - Solo Agency Local Collector extension and Local Collector app URL/port mismatch,
   - Chrome service worker is asleep and has not woken recently,
   - browser profile is not the one where the extension was installed.
6. If `/status` fails:
   - record `bridge_status: offline`,
   - try to start the bridge if the AI environment has permission,
   - otherwise provide the human with the absolute-path Local Collector app start command,
   - continue with public sources and previously collected private data.
6. If the bridge is running but the extension is stale, do not keep retrying aggressively. Continue with public sources, log the private-source blocker, and notify the human.
7. If the extension is recent but a private source fails due to login/captcha/checkpoint/session expiry, skip that source, log the platform-specific issue, and notify the human.

The AI agent must surface this health information transparently in the daily report and in Telegram notifications when private sources are unavailable.

Example notification:

```md
Agent: Claude Schedule
Collector status: bridge_running, extension_stale
Last extension check: 2026-06-20 08:52 local time
Likely cause: Chrome is closed or the extension is disabled.
Impact: Private Facebook/LinkedIn sources were skipped today. Public sources still ran.
Action: Open Chrome with the Solo Agency Local Collector extension enabled, stay logged in, or run the Local Collector app start command again if needed.
```

### OS Startup For Persistent Bridge

If the AI agent can run local commands, it should install or document an OS startup service for the bridge when the human wants unattended collection after reboot.

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

If the bridge is not installed as a startup service, the human must start it manually after reboot or the AI agent must start it when the environment allows local command execution.

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

When `run_mode` is `agent_on_demand`, the AI agent should start the localhost bridge immediately before private data collection and stop it immediately after collection completes.

When `run_mode` is `persistent_bridge_scheduler`, the bridge should start at user login or machine startup and remain idle until a configured collection window is active.

Typical run:

1. Agent detects the operating system and CPU architecture.
2. Agent selects the matching bridge binary from `solo-agency-local-collector/bin/`.
3. Agent creates a collection job file.
4. Agent starts the bridge on `127.0.0.1` with a short TTL.
5. Solo Agency Local Collector extension detects the bridge by polling localhost.
6. Extension fetches the job, collects visible authorized data from configured private sources, and posts results back to the bridge.
7. Bridge writes JSONL/status/snapshot files.
8. Agent reads the files.
9. Agent stops the bridge or lets it auto-shutdown.

Example bridge command shape:

```text
solo-agency-local-collector/bin/collector-bridge-darwin-arm64 \
  --host 127.0.0.1 \
  --port 17321 \
  --run-id YYYY-MM-DD_client_slug \
  --job-file daily-content-pipeline/collector/jobs/YYYY-MM-DD_client_slug.json \
  --output-dir daily-content-pipeline/collector/inbox/YYYY-MM/YYYY-MM-DD_client_slug \
  --ttl-minutes 30
```

The exact command may differ by implementation, but the behavior must remain the same.

If the agent cannot execute local commands, it cannot start an on-demand localhost bridge by itself. In that case:

- The extension may queue a limited amount of data in extension storage until a bridge is available.
- The agent must log `collector_unavailable`.
- The agent must notify the human that the current AI environment cannot start the local bridge.
- The agent should continue with public sources and previously collected private data if available.

Important constraint:

- A Chrome extension cannot magically start a localhost server if no local process is already running. If Native Messaging is not used, then either the AI agent, another local scheduler, or the human must start the bridge.

### Solo Agency Local Collector Extension Behavior

The human installs the Solo Agency Local Collector extension once in the Chrome browser/profile where they are already logged in to the relevant social platforms.

Important browser reality:

- Chrome Manifest V3 background service workers are not guaranteed to stay awake continuously.
- Do not rely on `alert()` or fake UI prompts to prevent browser sleep; background service workers do not have a reliable visible alert context and this is not a dependable automation strategy.
- Use `chrome.alarms` as the durable wake-up mechanism while Chrome is running.
- Use a short in-memory poll loop only while the service worker is awake.
- If Chrome is closed, the computer is asleep, the extension is disabled/removed, or the browser profile is not running, the extension cannot collect private data.
- In those cases, the bridge/agent must mark private collection as temporarily unavailable, continue with public sources and previously collected private data, and notify the human through WideCast MCP / Telegram when available.

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
- Default to 5 scrolls per private source and wait 5 seconds between scrolls.
- Allow the human to configure up to 10 scrolls per private source.
- Collect visible text, URLs, timestamps, engagement hints, profile URLs, post/current URLs, and source metadata.
- Collect relevant recommended groups/pages/communities as `new_private_sources` when visible.
- Post structured results back to the local bridge.
- Avoid posting, commenting, reacting, messaging, following, scraping contact details, or changing account state.

The extension should not require the human to click Allow on every scheduled run. The human's one-time action should be installing the extension and granting the extension permissions requested by Chrome.

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

Health API:

- `GET http://127.0.0.1:17321/status` is the Local Collector app health API.
- The AI agent may call `/status` at any time, before setup, before a manual run, during a run, after a run, before generating a report, before sending a Telegram notification, or while troubleshooting.
- `/status` is read-only. Calling it must not create a job, start a collection run, advance a schedule window, or mark a run complete.
- The AI agent should call `/status` without special headers.
- The Solo Agency Local Collector extension may call `/status` from its extension context and may include `X-Collector-Extension: media-agency-local-collector`; that is how the Local Collector app records `extension_health.last_extension_check_at`.
- The AI agent must not use the extension header during normal health checks, because it would make the bridge think the browser extension checked in when only the AI agent did.
- If `/status` fails to connect, the Local Collector app is not running or is blocked. The AI agent should start it if allowed, otherwise give the human the one-line start command generated during setup.
- If `/status` succeeds but `extension_health.status` is `stale` or `no_extension_check_yet` after the 75-second extension check grace window, the Local Collector app is running but the Solo Agency Local Collector extension is not currently checking in. The AI agent should treat private-source collection as unavailable until fixed, continue public-source work, and notify the human through WideCast Telegram if available.

`POST /jobs/run_now` is required for manual runs and first-trial runs. It lets the AI agent tell the Local Collector app:

```text
Run this private-source job immediately. Do not wait for the recurring schedule window.
```

Run-now behavior:

- The Local Collector app stores the run-now job as the active job.
- `/status` returns `job_available: true` and `current_job_type: run_now`.
- The Solo Agency Local Collector extension sees the job on its next poll and starts collecting.
- The run-now job should default to `force: false`.
- Each manual run should use a fresh unique `run_id`.
- The run-now job must include a TTL, default 30 minutes and maximum 120 minutes.
- If `/complete` is never received, the Local Collector app must stop exposing the run-now job after its TTL expires.
- After `/complete`, the run-now job is cleared so it does not repeat.

Run-now stuck-status guard:

- A manual or first-trial job must never be exposed forever.
- The Local Collector app must treat `run_now_expires_at` as a hard stop. After that time, `/status` must return `job_available: false` for that run-now job even if the Solo Agency Local Collector extension crashed, Chrome was closed, the machine slept, or `/complete` was never called.
- The agent must not set `force: true` for routine manual runs. `force: true` is reserved only for explicit troubleshooting when the human understands that it can intentionally re-run a previously completed `run_id`.
- The agent must not reuse yesterday's or a previous manual `run_id` to “run again”. It must create a new unique `run_id`.
- If the agent sees `current_job_type: run_now` for longer than the configured TTL, it should report a Local Collector app bug or stale process, restart the Local Collector app if allowed, and notify the human through WideCast Telegram if available.
- If the Solo Agency Local Collector extension reports `already_completed`, the agent should not force the same job. It should create a new run-now job with a new `run_id`.

The `/status` response should include:

- bridge status,
- active run/window id,
- current job type: `run_now`, `scheduled`, `on_demand`, or `none`,
- output directory,
- job availability,
- completed status,
- counts,
- `extension_health.last_extension_check_at`,
- `extension_health.seconds_since_last_check`,
- `extension_health.extension_check_count`,
- `extension_health.status` such as `recent`, `stale`, or `no_extension_check_yet`.

The bridge should also write a local health file:

```text
daily-content-pipeline/collector/inbox/bridge_health.json
```

Every time the extension checks `/status`, the bridge should update the last extension check timestamp. This lets the AI agent distinguish between:

- bridge not running,
- bridge running but extension not installed,
- bridge running but Chrome closed,
- extension installed but stale/sleeping,
- extension recent and healthy,
- private source session expired,
- platform checkpoint/captcha/rate limit.

The bridge may run smoothly without admin permission on many machines because it binds only to loopback, but the agent must not promise zero operating-system prompts in every environment. Some corporate devices, antivirus tools, endpoint security tools, firewalls, Gatekeeper, or SmartScreen policies may still warn about new executables. Signed binaries are recommended for public distribution.

### Collector Output Files

For each run, the bridge should write:

```text
daily-content-pipeline/collector/
  jobs/
    YYYY-MM/
      YYYY-MM-DD_client_slug.json
  inbox/
    YYYY-MM/
      YYYY-MM-DD_client_slug/
        collector_status.json
        private_data_points.jsonl
        leads.jsonl
        competitors.jsonl
        new_private_sources.jsonl
        source_status.jsonl
        snapshots/
          source_slug_post_or_thread.html
```

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

Every new private source candidate must include:

- `source_name`
- `platform`
- `source_type`
- `profile_or_group_url`
- `current_recommendation_url`
- `detected_while_scanning`
- `why_relevant`
- `related_content_pillar`
- `estimated_priority`
- `suggested_scan_cadence`
- `status`

If a URL is unavailable, write `unavailable` and include a note explaining why.

### Agent Compatibility Rule

Codex:

- If Codex can run local commands, Codex should start the bridge on demand, wait for collector output, stop the bridge, then continue the daily pipeline.
- If Codex cannot access Chrome's logged-in session directly, it should still use the extension/bridge output files.

Claude:

- Claude must use the Solo Agency Local Collector extension plus the Local Collector app for automated private-source collection.
- Claude must not use Claude Chrome Extension for automated private-source collection because it can require repeated human Allow clicks and can block unattended schedules.
- If Claude cannot start local commands, Claude must provide a user-run command, persistent bridge startup instructions, or OS startup service setup instructions.
- After the bridge is running, Claude reads collector output files and performs reasoning, idea generation, script writing, reporting, and WideCast actions.

Hermes, OpenClaw, and other agents:

- If the agent can run local commands, use the same on-demand bridge flow.
- If the agent cannot run local commands, read the latest collector files or use an MCP wrapper that exposes the collector folder.

### Native Messaging Decision

Do not require Native Messaging for the default version of this playbook.

Native Messaging is a valid production architecture, but it requires OS-specific host registration and may create more installation friction:

- macOS requires native host manifest placement and may trigger Gatekeeper warnings if unsigned.
- Windows requires registry registration and may trigger SmartScreen warnings if unsigned.
- Linux requires Chrome/Chromium-specific manifest paths.

The default collector should use localhost because it is easier for AI agents to start and stop on demand.

Native Messaging may be added later as an advanced or enterprise option.

### Fallback Browser Session Flow

If the extension plus on-demand localhost bridge is unavailable, use this two-phase browser session flow whenever the environment allows it.

### Phase 1: Manual Login Bootstrap

If the agent can show a headed browser UI, the agent must:

1. Open a headed browser window with a dedicated persistent profile folder.
2. Use a source-specific profile path such as:
   - `daily-content-pipeline/browser_profiles/facebook/`
   - `daily-content-pipeline/browser_profiles/linkedin/`
   - `daily-content-pipeline/browser_profiles/reddit/`
3. Ask the human to log in manually inside that browser window.
4. Never ask the human to share credentials.
5. Keep cookies, local storage, and browser session data inside the dedicated profile folder.
6. Treat the profile folder as sensitive because it may contain authenticated session data.

The agent should say:

`I will open a dedicated browser profile for this source. Please log in manually in the browser window. Do not share your password, OTP, cookies, or credentials. After login, close the browser window and I will reuse that browser profile for future collection until the session expires.`

### If The Agent Cannot Show Browser UI

Some environments, including some Claude Desktop or Claude sandbox setups, may allow file or command execution but cannot display a Playwright headed browser window. In that case, the agent must not claim it can complete headed login bootstrap by itself.

Use one of these alternatives:

1. External local bootstrap script:
   - The agent creates or provides a small local script that the human runs outside the sandbox.
   - The script opens a visible browser with a dedicated persistent profile folder.
   - The human logs in manually.
   - Future collection reuses that profile.

2. Local CDP bridge:
   - The human opens Chrome outside the sandbox with a dedicated `--user-data-dir` and `--remote-debugging-port`.
   - The agent or collector connects to that browser through Chrome DevTools Protocol if the environment can access the local endpoint.
   - The human logs in in the visible browser, while the collector later reuses the same profile or CDP session.

3. External scheduled collector:
   - A local cron job, LaunchAgent, n8n workflow, Make scenario, Browserbase, Browserless, Apify, or another browser automation service performs private-source collection.
   - The collector writes daily data points into the pipeline files.
   - The AI agent reads those data points and performs reasoning, idea generation, and script writing.

4. Manual fallback:
   - If no browser automation path is available, the human provides exported text, screenshots, copied posts, or a group/source list.
   - The agent treats this as manually supplied data and continues the pipeline.

The agent should say:

`This environment cannot display a browser login window. I will not ask for credentials. Please run the external browser bootstrap or open the provided Chrome profile outside the sandbox, log in manually, and then I will reuse the resulting profile or data files for future collection.`

### Phase 2: Scheduled Headless Collection

After the human has logged in once, the agent or collector should:

1. Reuse the same persistent browser profile.
2. Run future collection jobs headlessly when possible.
3. Visit only the configured private data sources.
4. Extract only relevant visible text and metadata.
5. Filter collected data against primary industry, sub-industry, related industries, target audience, target location, pain points, and business offer.
6. Save collected findings as data points in the client pipeline.
7. Log skipped or expired sessions.

If the session expires:

- Skip the private source for that run.
- Log `session_expired` in `history/YYYY-MM/data_sources_log.md`.
- Ask the human to refresh login manually through the headed browser profile.
- Never ask for credentials.

### AI-Service-Specific Guidance

Codex:

- If Codex has a native browser or in-app browser tool available, Codex may use that browser directly for private-source review.
- If persistent login is needed for scheduled collection, Codex may still use the browser session bootstrap and collector flow.

Claude:

- If the human is using Claude, the private-source path is the Solo Agency Local Collector extension plus the Local Collector app described above.
- Claude must not use Claude Chrome Extension for this automated private-source workflow.
- If Claude cannot run the bridge binary in its sandbox, Claude must give the human a one-time command or startup-service instructions to run the bridge outside the sandbox.
- The recommended Claude-safe mode is `persistent_bridge_scheduler`, because once the bridge is running at OS startup, Claude only needs to read local collector files.
- If the bridge is unavailable, Claude should continue with public sources and previously collected private data, then notify the human.

Other agents:

- If the agent has reliable native browser automation, it may use that.
- If native browsing is unreliable, approval-gated, or unavailable, use the persistent browser profile collector flow.

Security note:

- Browser profile folders and storage state files may contain sensitive authenticated session data.
- Do not commit them to git.
- Do not upload them.
- Do not share them across users.
- Store them locally and restrict access where possible.

---


Manual run / run-now rule:

- Any human request such as `run now`, `manual run`, `test run`, `trial run`, `collect now`, `scan now`, or `chạy thử` must bypass recurring schedule windows.
- The agent must not wait for `scheduled_windows` when the human requested a manual run.
- If the Local Collector app is reachable, the agent must create a run-now job and call `POST http://127.0.0.1:17321/jobs/run_now`.
- The run-now job must include:
  - unique `run_id`,
  - `run_now: true`,
  - `force: false` by default,
  - `run_now_ttl_minutes`, default 30 and maximum 120,
  - private `sources`,
  - pacing rules,
  - client/business/location metadata when available.
- To run again, the agent should create a new unique `run_id` instead of forcing the same run id repeatedly.
- The run-now job must expire automatically if it is not completed, so the extension cannot keep seeing the same manual job all day.
- The Solo Agency Local Collector extension should see `job_available: true` on the next `/status` poll and run immediately.
- If the Local Collector app is not reachable, the agent should start it if possible. If the agent cannot start it, provide the one-line Local Collector app start command, then retry the run-now job after the app is reachable.
- Recurring schedule windows are only for unattended scheduled runs. They must not block manual runs.
- Do not simulate a manual run by editing `scheduled_windows` or creating a temporary schedule window. Manual runs must use `/jobs/run_now`.
- If the agent cannot call `http://127.0.0.1:17321` from its own sandbox but can write local files, it must write the same run-now payload to `daily-content-pipeline/collector/run_now_request.json`. The Local Collector app must check this file on `/status`, load it as a run-now job, write `run_now_request_status.json`, and move the request aside as consumed. This avoids asking the human to run another command.
- If the agent cannot call HTTP and cannot write the local request file, only then create a local run-now helper script or launcher and give the human exactly one short command/path to run it. The helper script must POST `/jobs/run_now` with the correct payload, then optionally poll `/status`.
- Do not ask the human to restart the Local Collector app merely to make a manually edited schedule file take effect. Restarting is only appropriate for updating the Local Collector app itself, recovering a stuck/offline process, or applying an intentional recurring schedule change when both `/config` and file auto-reload are unavailable.
- If a legacy collector without `/jobs/run_now` forces a temporary schedule fallback, the agent must clearly label it as a fallback, back up the original config, create a short unique temporary window, restart or reload only if required, restore the original config immediately after completion/timeout, and report that fallback to the human. This fallback must not be used when `/jobs/run_now` exists.

Exact manual run-now contract:

- Health-check the Local Collector app first with plain `GET http://127.0.0.1:17321/status`.
- Do not send `X-Collector-Extension` when the AI agent checks health. That header is for the Solo Agency Local Collector extension only. If the AI agent fakes it, `extension_health` can become misleading.
- If `/status` is reachable, call `POST http://127.0.0.1:17321/jobs/run_now`.
- The minimum payload should look like this:

```json
{
  "run_id": "2026-06-20_client-slug_manual_150405",
  "client_slug": "client-slug",
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
    "do_not_scrape_contact_details": true
  }
}
```

- `run_id` must be unique for every manual run. A recommended pattern is `YYYY-MM-DD_client-slug_manual_HHMMSS`.
- `run_now` must be `true`.
- `force` must be `false` unless the human explicitly asks for a troubleshooting rerun and understands the same `run_id` may run again.
- `run_now_ttl_minutes` should be 30 by default and must not exceed 120.
- `sources` must contain the private sources for that client if private sources exist. If there are no private sources, the agent should still run public research without the Local Collector app.
- `pacing.scroll_steps` defaults to 5 and must not exceed 10.
- If the agent cannot make this POST itself but can write local files, it should write the JSON payload to:

```text
daily-content-pipeline/collector/run_now_request.json
```

The agent should write this file atomically: write a temporary file in the same folder first, then rename it to `run_now_request.json` only after the JSON is complete.

The running Local Collector app should pick up this file on the next `/status` check from the Chrome extension or AI agent, usually within a few seconds while Chrome is active. After loading the request, the Local Collector app must immediately consume the request so it cannot loop forever:

- move it to `run_now_request.{run_id}.{timestamp}.consumed.json`;
- write `run_now_request_status.json`;
- remember the processed file signature in memory as a replay guard if moving/removing fails;
- clear the active run-now job on `/complete`;
- expire the active run-now job after `run_now_ttl_minutes` if `/complete` never arrives.

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

- After posting `/jobs/run_now`, poll plain `GET /status` until either:
  - `current_job_type` becomes `run_now` and `job_available` is `true`,
  - the extension completes and `/status` returns `job_available: false`, or
  - the TTL expires and private collection is marked unavailable for this run.

Schedule rule:

- Ask schedule/routine questions after the profile and source plan are known and before the first agency run.
- Ask whether the human wants daily, multiple-times-daily, weekly, manual-only, or another cadence.
- Then write or update `schedule.md` and the relevant automation/config files.
- After schedule/routine setup, ask whether to run the first agency run immediately.

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
- If the human has not activated private-source monitoring yet, configure the recurring schedule as public-only and clearly mark private sources as `pending_private_activation`.
- Only configure scheduled private-source collection after Local Collector activation is accepted and collector health is confirmed or explicitly documented as pending/blocker.
- The Local Collector app must run in persistent mode for unattended scheduled collection:

```text
solo-agency-local-collector/bin/collector-bridge-darwin-arm64 \
  --host 127.0.0.1 \
  --port 17321 \
  --config-file daily-content-pipeline/collector/collector_config.json \
  --output-dir daily-content-pipeline/collector/inbox \
  --persistent
```

- The Solo Agency Local Collector extension polls `/status`; when the current local time is inside an enabled `scheduled_windows` item and private sources exist, `/status` should expose a scheduled job with `current_job_type: scheduled` and `job_available: true`.
- Scheduled run IDs are generated by the Local Collector app, usually using `YYYY-MM-DD_schedule-name`.
- The agent must still write a human-readable `schedule.md` explaining the cadence, clients included, private-source limits, and notification behavior.

---
