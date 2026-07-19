#!/usr/bin/env bash
# Solo Agency — Local Collector setup + start (canonical, tested).
#
# ONE command does everything and is safe to re-run:
#   1) download the prebuilt bridge bundle, 2) VERIFY its checksum (matched by
#   basename, so any SHA256SUMS format works), 3) extract the binary for THIS machine,
#   4) STOP any bridge already on the port (graceful /shutdown → PID → port; it will
#      never kill a non-collector process), 5) START the newest bridge in the
#      BACKGROUND (persistent) so you can close the Terminal.
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

# --- resolve agency root + runtime folders (robust to where it's run from) ---
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd || echo "$PWD")"
if   [ -d "$PWD/daily-content-pipeline" ];            then ROOT="$PWD"
elif [ -d "$SCRIPT_DIR/../daily-content-pipeline" ];  then ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
elif [ -d "$SCRIPT_DIR/daily-content-pipeline" ];     then ROOT="$SCRIPT_DIR"
else ROOT="$PWD"; fi
RUNTIME="$ROOT/solo-agency-local-collector"
DL="$RUNTIME/downloads"
BIN="$RUNTIME/bin"
CONFIG_FILE="$ROOT/daily-content-pipeline/collector/collector_config.json"
OUTPUT_DIR="$ROOT/daily-content-pipeline/collector/inbox"
PID_FILE="$RUNTIME/collector.pid"
LOG_FILE="$RUNTIME/collector.log"
mkdir -p "$DL" "$BIN" "$OUTPUT_DIR"

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

say "6/6  Starting the newest bridge (background, persistent)"
[ -f "$CONFIG_FILE" ] || warn "config not found at $CONFIG_FILE — starting anyway; if the bridge exits, create the config and re-run."
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

say "Done — the collector is running in the background. You can close this Terminal."
info "Port   : 127.0.0.1:$PORT"
info "Status : curl -s http://127.0.0.1:$PORT/status"
info "Logs   : $LOG_FILE"
info "Stop   : kill \$(cat \"$PID_FILE\")"
info "One-time: install the Chrome extension via Developer Mode (see AGENT_RUNBOOK.md)."
