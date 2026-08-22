#!/usr/bin/env bash
# Solo Agency — Local Collector setup + start (canonical, tested).
#
# ONE command does everything and is safe to re-run:
#   1) download the prebuilt bridge bundle, 2) VERIFY its checksum (matched by
#   basename, so any SHA256SUMS format works), 3) extract the binary for THIS machine,
#   4) STOP any bridge already on the port (graceful /shutdown → PID → port; it will
#      never kill a non-collector process), 5) START the newest bridge in the
#      BACKGROUND (persistent) so you can close the Terminal — and REGISTER it to
#      start automatically at login/boot (macOS launchd, Linux systemd; crash =
#      auto-restart, clean stop stays stopped). SOLO_AGENCY_NO_AUTOSTART=1 skips
#      the registration and starts a plain background process instead.
#
#   Usage:  bash setup_collector.sh
#
# It NEVER fails on "address already in use": it stops the old collector first, then
# starts the new one. It does not overwrite collector_config.json or delete any data.

set -euo pipefail

VERSION="${SOLO_AGENCY_COLLECTOR_VERSION:-0.1.0}"
BASE_URL="${SOLO_AGENCY_DIST_BASE:-https://raw.githubusercontent.com/soloagency/solo-agency/dist}"
BUNDLE="collector-bridge-binaries-${VERSION}.zip"
SUMS="SHA256SUMS"
PORT="${SOLO_AGENCY_BRIDGE_PORT:-17321}"

say()  { printf '\n\033[1m%s\033[0m\n' "$*"; }
info() { printf '  %s\n' "$*"; }
ok()   { printf '  \033[32m✓\033[0m %s\n' "$*"; }
warn() { printf '  \033[33m!\033[0m %s\n' "$*" >&2; }
fail() { printf '\n\033[31m✗ %s\033[0m\n' "$1" >&2; [ -n "${2:-}" ] && printf '  → %s\n' "$2" >&2; exit 1; }

# --- resolve agency root + runtime folders ------------------------------------
# Order: SOLO_AGENCY_ROOT env override → the SCRIPT'S OWN location → the current
# directory. The script location must beat $PWD: invoking an install's script by
# absolute path targets THAT install, even when the terminal happens to be
# standing in another workspace that also has a pipeline (two installs on one
# machine — a source repo plus client setups — is normal). $PWD-first here once
# silently restarted the bridge against the wrong workspace.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd || echo "$PWD")"
if   [ -n "${SOLO_AGENCY_ROOT:-}" ];                  then ROOT="$(cd "$SOLO_AGENCY_ROOT" && pwd)"
elif [ -d "$SCRIPT_DIR/../daily-content-pipeline" ];  then ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
elif [ -d "$SCRIPT_DIR/daily-content-pipeline" ];     then ROOT="$SCRIPT_DIR"
elif [ -d "$PWD/daily-content-pipeline" ];            then ROOT="$PWD"
else ROOT="$PWD"; fi
if [ -d "$PWD/daily-content-pipeline" ] && [ "$ROOT" != "$PWD" ]; then
  warn "Terminal is standing in a DIFFERENT workspace: $PWD"
  warn "Using the install this script belongs to: $ROOT"
  warn "(set SOLO_AGENCY_ROOT=/path to override)"
fi
RUNTIME="$ROOT/solo-agency-local-collector"
DL="$RUNTIME/downloads"
BIN="$RUNTIME/bin"
CONFIG_FILE="$ROOT/daily-content-pipeline/collector/collector_config.json"
OUTPUT_DIR="$ROOT/daily-content-pipeline/collector/inbox"
PID_FILE="$RUNTIME/collector.pid"
LOG_FILE="$RUNTIME/collector.log"
mkdir -p "$DL" "$BIN" "$OUTPUT_DIR"

# Per-install autostart identity: two installs on one machine (a source repo plus
# client setups is normal) each get their own launchd label / systemd unit.
INSTHASH="$(printf '%s' "$ROOT" | { shasum -a 256 2>/dev/null || sha256sum; } | awk '{print substr($1,1,8)}')"
LAUNCHD_LABEL="com.solo-agency.collector.$INSTHASH"
LAUNCHD_PLIST="$HOME/Library/LaunchAgents/$LAUNCHD_LABEL.plist"
SYSTEMD_UNIT="solo-agency-collector-$INSTHASH.service"
SYSTEMD_FILE="$HOME/.config/systemd/user/$SYSTEMD_UNIT"
AUTOSTART_STATE="$RUNTIME/autostart.json"

# record_autostart <mode> <label> <reason> — canonical, workspace-readable evidence
# of the autostart outcome. Sandboxed agents cannot run launchctl/systemctl and
# cannot read ~/Library, but they CAN read this file (filesystem is the bus).
record_autostart() {
  printf '{"mode": "%s", "label": "%s", "port": %s, "root": "%s", "registered_at": "%s", "reason": "%s"}\n' \
    "$1" "$2" "$PORT" "$ROOT" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$3" > "$AUTOSTART_STATE" 2>/dev/null || true
}

# --- checksum tool ---
if command -v shasum >/dev/null 2>&1; then SHACMD() { shasum -a 256 "$1"; }
elif command -v sha256sum >/dev/null 2>&1; then SHACMD() { sha256sum "$1"; }
else fail "No checksum tool found (need 'shasum' or 'sha256sum')." "Install coreutils and re-run."; fi
sha_of() { SHACMD "$1" | awk '{print $1}'; }

# --- platform → binary ---
os="$(uname -s 2>/dev/null || echo unknown)"; arch="$(uname -m 2>/dev/null || echo unknown)"
case "$os" in Darwin) O=darwin;; Linux) O=linux;; MINGW*|MSYS*|CYGWIN*) O=windows;; *) O=unknown;; esac
case "$arch" in arm64|aarch64) A=arm64;; x86_64|amd64) A=amd64;; *) A=unknown;; esac
EXT=""; [ "$O" = "windows" ] && EXT=".exe"
TARGET_BIN="collector-bridge-${O}-${A}${EXT}"
case "${O}-${A}" in
  darwin-arm64|darwin-amd64|linux-amd64|windows-amd64) : ;;
  *) fail "No prebuilt bridge for your system ($os / $arch)." "Supported: macOS arm64/amd64, Linux amd64, Windows amd64.";;
esac
BIN_PATH="$BIN/$TARGET_BIN"

download() {
  local name="$1" dest="$2" url="$BASE_URL/$1" tries=3 i=1
  while [ "$i" -le "$tries" ]; do
    if curl -fsSL --retry 2 --connect-timeout 15 -o "$dest.part" "$url"; then mv -f "$dest.part" "$dest"; return 0; fi
    warn "Download failed for $name (attempt $i/$tries) — retrying in 2s..."
    rm -f "$dest.part"; i=$((i + 1)); sleep 2
  done
  fail "Could not download $name" "Check your internet connection, then run this script again. Nothing was changed."
}
expected_sum_for() { grep -F "$1" "$DL/$SUMS" 2>/dev/null | awk '{print $1}' | head -1; }

say "Solo Agency Local Collector setup (v$VERSION)"
info "Machine : $os / $arch  →  bridge binary: $TARGET_BIN"
info "Agency  : $ROOT"

say "1/6  Fetching checksums"
download "$SUMS" "$DL/$SUMS"; ok "got $SUMS"

say "2/6  Fetching the bridge bundle"
want="$(expected_sum_for "$BUNDLE")"
[ -n "$want" ] || fail "Checksum for $BUNDLE not found in $SUMS." "The checksum file looks out of date. Re-run in a minute; if it persists, tell your setup agent."
if [ -f "$DL/$BUNDLE" ] && [ "$(sha_of "$DL/$BUNDLE")" = "$want" ]; then ok "already up to date (skipped download)"
else download "$BUNDLE" "$DL/$BUNDLE"; ok "downloaded $BUNDLE"; fi

say "3/6  Verifying checksum"
[ "$(sha_of "$DL/$BUNDLE")" = "$want" ] || { rm -f "$DL/$BUNDLE"; fail "Checksum MISMATCH for $BUNDLE." "Deleted the bad file — run this script again to re-download."; }
ok "checksum verified"

say "4/6  Extracting your binary"
command -v unzip >/dev/null 2>&1 || fail "'unzip' is not installed." "Install unzip and re-run."
unzip -o -q "$DL/$BUNDLE" -d "$BIN"
[ -f "$BIN_PATH" ] || fail "The bundle did not contain $TARGET_BIN." "It may be built for a different version. Tell your setup agent."
chmod +x "$BIN_PATH" 2>/dev/null || true
ok "installed: $BIN_PATH"

if [ "${SOLO_AGENCY_SETUP_NO_START:-0}" = "1" ]; then
  say "Install complete (SOLO_AGENCY_SETUP_NO_START=1 → not stopping/starting the bridge)."
  info "To run it: bash setup_collector.sh"
  exit 0
fi

say "5/6  Stopping any bridge already on port $PORT"
# a0) detach any autostart supervisor FIRST so it cannot respawn the old binary
#     while we upgrade (bootout/stop also kills the supervised process).
if [ "$O" = "darwin" ] && command -v launchctl >/dev/null 2>&1; then
  launchctl bootout "gui/$(id -u)/$LAUNCHD_LABEL" >/dev/null 2>&1 || true
fi
if [ "$O" = "linux" ] && command -v systemctl >/dev/null 2>&1; then
  systemctl --user stop "$SYSTEMD_UNIT" >/dev/null 2>&1 || true
fi
# a) ask the running bridge to shut down cleanly
command -v curl >/dev/null 2>&1 && curl -s -m 3 -X POST "http://127.0.0.1:$PORT/shutdown" >/dev/null 2>&1 || true
# b) kill the PID we started last time
if [ -f "$PID_FILE" ]; then
  OLD_PID="$(cat "$PID_FILE" 2>/dev/null || true)"
  if [ -n "${OLD_PID:-}" ] && kill -0 "$OLD_PID" >/dev/null 2>&1; then kill "$OLD_PID" >/dev/null 2>&1 || true; fi
fi
# c) anything still holding the port — but ONLY if it is a collector-bridge; never
#    kill an unknown process (that would be dangerous). Tell the user instead.
if command -v lsof >/dev/null 2>&1; then
  for P in $(lsof -tiTCP:"$PORT" -sTCP:LISTEN 2>/dev/null || true); do
    CMD="$(ps -p "$P" -o command= 2>/dev/null || true)"
    if printf '%s' "$CMD" | grep -q "collector-bridge"; then kill "$P" >/dev/null 2>&1 || true
    else fail "Port $PORT is held by a NON-collector process (PID $P): $CMD" "This setup will not kill an unknown process. Stop it yourself, then re-run."; fi
  done
fi
sleep 1
ok "port $PORT is free"

say "6/6  Starting the newest bridge (background, persistent, autostart at login)"
[ -f "$CONFIG_FILE" ] || warn "config not found at $CONFIG_FILE — starting anyway; if the bridge exits, create the config and re-run."

# Wait until /status answers (supervised starts have no PID to poll directly).
wait_healthy() {
  local i=0
  while [ "$i" -lt 20 ]; do
    if curl -s -m 2 "http://127.0.0.1:$PORT/status" >/dev/null 2>&1; then return 0; fi
    i=$((i + 1)); sleep 1
  done
  return 1
}
bridge_pid_on_port() { lsof -tiTCP:"$PORT" -sTCP:LISTEN 2>/dev/null | head -1; }

start_nohup() {
  nohup "$BIN_PATH" \
    --host 127.0.0.1 --port "$PORT" \
    --config-file "$CONFIG_FILE" \
    --output-dir "$OUTPUT_DIR" \
    --persistent >"$LOG_FILE" 2>&1 &
  NEW_PID=$!
  echo "$NEW_PID" > "$PID_FILE"
  sleep 2
  # health check — make a silent background death VISIBLE
  if ! kill -0 "$NEW_PID" >/dev/null 2>&1; then
    fail "The bridge exited right after starting." "Last lines of $LOG_FILE:
$(tail -n 15 "$LOG_FILE" 2>/dev/null | sed 's/^/    /')"
  fi
  ok "bridge running (pid $NEW_PID)"
}

AUTOSTART="none"
STOP_HINT="kill \$(cat \"$PID_FILE\")"
if [ "${SOLO_AGENCY_NO_AUTOSTART:-0}" = "1" ]; then
  info "SOLO_AGENCY_NO_AUTOSTART=1 — plain background start (no boot registration)."
  record_autostart "none" "" "opt_out_env"
  start_nohup

elif [ "$O" = "darwin" ] && command -v launchctl >/dev/null 2>&1; then
  # macOS: a per-user LaunchAgent. RunAtLoad starts it at login; KeepAlive on
  # failure restarts crashes but respects a clean stop (/shutdown exits 0).
  mkdir -p "$HOME/Library/LaunchAgents"
  cat > "$LAUNCHD_PLIST" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>$LAUNCHD_LABEL</string>
  <key>ProgramArguments</key><array>
    <string>$BIN_PATH</string>
    <string>--host</string><string>127.0.0.1</string>
    <string>--port</string><string>$PORT</string>
    <string>--config-file</string><string>$CONFIG_FILE</string>
    <string>--output-dir</string><string>$OUTPUT_DIR</string>
    <string>--persistent</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><dict><key>SuccessfulExit</key><false/></dict>
  <key>StandardOutPath</key><string>$LOG_FILE</string>
  <key>StandardErrorPath</key><string>$LOG_FILE</string>
</dict></plist>
PLIST
  if launchctl bootstrap "gui/$(id -u)" "$LAUNCHD_PLIST" >/dev/null 2>&1 \
     && wait_healthy; then
    AUTOSTART="launchd ($LAUNCHD_LABEL)"
    STOP_HINT="launchctl bootout gui/\$(id -u)/$LAUNCHD_LABEL"
    bridge_pid_on_port > "$PID_FILE" 2>/dev/null || true
    record_autostart "launchd" "$LAUNCHD_LABEL" "registered"
    ok "bridge running under launchd — starts automatically at login, restarts on crash"
  else
    warn "launchd registration failed — falling back to a plain background start."
    rm -f "$LAUNCHD_PLIST"
    record_autostart "none" "" "launchd_registration_failed"
    start_nohup
  fi

elif [ "$O" = "linux" ] && command -v systemctl >/dev/null 2>&1 && systemctl --user show-environment >/dev/null 2>&1; then
  # Linux: a systemd user unit. enable = start at login; Restart=on-failure
  # restarts crashes but respects a clean stop. enable-linger (best effort)
  # makes it start at BOOT, before the user logs in.
  mkdir -p "$HOME/.config/systemd/user"
  cat > "$SYSTEMD_FILE" <<UNIT
[Unit]
Description=Solo Agency local collector bridge ($ROOT)
After=network-online.target

[Service]
ExecStart=$BIN_PATH --host 127.0.0.1 --port $PORT --config-file $CONFIG_FILE --output-dir $OUTPUT_DIR --persistent
Restart=on-failure
RestartSec=3
StandardOutput=append:$LOG_FILE
StandardError=append:$LOG_FILE

[Install]
WantedBy=default.target
UNIT
  systemctl --user daemon-reload
  if systemctl --user enable "$SYSTEMD_UNIT" >/dev/null 2>&1 \
     && systemctl --user restart "$SYSTEMD_UNIT" >/dev/null 2>&1 \
     && wait_healthy; then
    AUTOSTART="systemd ($SYSTEMD_UNIT)"
    STOP_HINT="systemctl --user stop $SYSTEMD_UNIT"
    bridge_pid_on_port > "$PID_FILE" 2>/dev/null || true
    record_autostart "systemd" "$SYSTEMD_UNIT" "registered"
    if loginctl enable-linger "$(id -un)" >/dev/null 2>&1; then
      ok "bridge running under systemd — starts at BOOT (lingering on), restarts on crash"
    else
      ok "bridge running under systemd — starts at login, restarts on crash"
      warn "could not enable lingering; to start at boot before login run: sudo loginctl enable-linger $(id -un)"
    fi
  else
    warn "systemd registration failed — falling back to a plain background start."
    systemctl --user disable "$SYSTEMD_UNIT" >/dev/null 2>&1 || true
    rm -f "$SYSTEMD_FILE"; systemctl --user daemon-reload >/dev/null 2>&1 || true
    record_autostart "none" "" "systemd_registration_failed"
    start_nohup
  fi

else
  # Windows Git-Bash lands here — setup_collector.ps1 registers the Scheduled
  # Task; from bash we can only do a plain background start.
  [ "$O" = "windows" ] && info "For autostart on Windows run setup_collector.ps1 (registers a logon Scheduled Task)."
  record_autostart "none" "" "no_supervisor_available"
  start_nohup
fi

say "Done — the collector is running in the background. You can close this Terminal."
info "Port      : 127.0.0.1:$PORT"
info "Status    : curl -s http://127.0.0.1:$PORT/status"
info "Logs      : $LOG_FILE"
if [ "$AUTOSTART" != "none" ]; then
  info "Autostart : $AUTOSTART — survives reboots; re-run this script after updates."
else
  info "Autostart : OFF — after a reboot run: bash setup_collector.sh"
fi
info "Stop      : $STOP_HINT"
info "One-time: install the Chrome extension via Developer Mode (see AGENT_RUNBOOK.md)."
