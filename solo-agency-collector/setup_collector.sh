#!/usr/bin/env bash
# Solo Agency — Local Collector setup (canonical, tested).
#
# Downloads the prebuilt bridge binary bundle, VERIFIES its checksum, extracts the
# binary for THIS machine, and prints the one command to start it. It never starts the
# bridge itself, and it is safe to re-run (idempotent).
#
#   Usage:  bash setup_collector.sh
#
# Run it from your agency root (the folder that contains daily-content-pipeline/). If
# run elsewhere it creates ./solo-agency-local-collector/ next to where you run it.
#
# Why this script exists: agents used to hand-write the download+checksum step and some
# assumed the wrong SHA256SUMS format, dead-ending after a 10 MB download. This script
# is the ONE supported path — verification matches the checksum by BASENAME, so it works
# whether SHA256SUMS lists a bare name, "*name", or a full path.

set -euo pipefail

VERSION="${SOLO_AGENCY_COLLECTOR_VERSION:-0.1.0}"
BASE_URL="${SOLO_AGENCY_DIST_BASE:-https://raw.githubusercontent.com/soloagency/solo-agency/dist}"
BUNDLE="collector-bridge-binaries-${VERSION}.zip"
SUMS="SHA256SUMS"

say()  { printf '\n\033[1m%s\033[0m\n' "$*"; }
info() { printf '  %s\n' "$*"; }
ok()   { printf '  \033[32m✓\033[0m %s\n' "$*"; }
warn() { printf '  \033[33m!\033[0m %s\n' "$*" >&2; }
fail() { printf '\n\033[31m✗ %s\033[0m\n' "$1" >&2; [ -n "${2:-}" ] && printf '  → %s\n' "$2" >&2; exit 1; }

# --- runtime folders ---------------------------------------------------------
ROOT="$(pwd)"
RUNTIME="$ROOT/solo-agency-local-collector"
DL="$RUNTIME/downloads"
BIN="$RUNTIME/bin"
mkdir -p "$DL" "$BIN"

# --- checksum tool -----------------------------------------------------------
if command -v shasum >/dev/null 2>&1; then SHACMD() { shasum -a 256 "$1"; }
elif command -v sha256sum >/dev/null 2>&1; then SHACMD() { sha256sum "$1"; }
else fail "No checksum tool found (need 'shasum' or 'sha256sum')." "Install coreutils and re-run. Nothing was changed."; fi
sha_of() { SHACMD "$1" | awk '{print $1}'; }

# --- platform detection ------------------------------------------------------
os="$(uname -s 2>/dev/null || echo unknown)"
arch="$(uname -m 2>/dev/null || echo unknown)"
case "$os" in Darwin) O=darwin;; Linux) O=linux;; MINGW*|MSYS*|CYGWIN*) O=windows;; *) O=unknown;; esac
case "$arch" in arm64|aarch64) A=arm64;; x86_64|amd64) A=amd64;; *) A=unknown;; esac
EXT=""; [ "$O" = "windows" ] && EXT=".exe"
TARGET_BIN="collector-bridge-${O}-${A}${EXT}"
case "${O}-${A}" in
  darwin-arm64|darwin-amd64|linux-amd64|windows-amd64) : ;;
  *) fail "No prebuilt bridge for your system ($os / $arch)." "Supported: macOS arm64/amd64, Linux amd64, Windows amd64. Ask a maintainer to build one for your platform.";;
esac

# --- helpers -----------------------------------------------------------------
download() {
  # download <remote-name> <dest-path>
  local name="$1" dest="$2" url="$BASE_URL/$1" tries=3 i=1
  while [ "$i" -le "$tries" ]; do
    if curl -fsSL --retry 2 --connect-timeout 15 -o "$dest.part" "$url"; then
      mv -f "$dest.part" "$dest"; return 0
    fi
    warn "Download failed for $name (attempt $i/$tries) — retrying in 2s..."
    rm -f "$dest.part"; i=$((i + 1)); sleep 2
  done
  fail "Could not download $name" "Check your internet connection, then run this script again. Nothing was changed."
}

expected_sum_for() {
  # Extract the published hash for a file by BASENAME — tolerant of bare-name,
  # "*name", and full-path formats in SHA256SUMS.
  grep -F "$1" "$DL/$SUMS" 2>/dev/null | awk '{print $1}' | head -1
}

# --- run ---------------------------------------------------------------------
say "Solo Agency Local Collector setup (v$VERSION)"
info "Machine : $os / $arch  →  bridge binary: $TARGET_BIN"
info "Install : $RUNTIME"

say "1/4  Fetching checksums"
download "$SUMS" "$DL/$SUMS"
ok "got $SUMS"

say "2/4  Fetching the bridge bundle"
want="$(expected_sum_for "$BUNDLE")"
[ -n "$want" ] || fail "Checksum for $BUNDLE not found in $SUMS." "The published checksum file looks out of date. Re-run in a minute; if it persists, tell your setup agent (SHA256SUMS may have changed format)."
if [ -f "$DL/$BUNDLE" ] && [ "$(sha_of "$DL/$BUNDLE")" = "$want" ]; then
  ok "already downloaded and up to date (skipped the 11 MB download)"
else
  download "$BUNDLE" "$DL/$BUNDLE"
  ok "downloaded $BUNDLE"
fi

say "3/4  Verifying checksum"
got="$(sha_of "$DL/$BUNDLE")"
if [ "$want" != "$got" ]; then
  rm -f "$DL/$BUNDLE"
  fail "Checksum MISMATCH for $BUNDLE (download corrupted or tampered)." "Deleted the bad file — run this script again to re-download. Do NOT use a file that fails this check."
fi
ok "checksum verified"

say "4/4  Extracting your binary"
command -v unzip >/dev/null 2>&1 || fail "'unzip' is not installed." "Install unzip and re-run."
unzip -o -q "$DL/$BUNDLE" -d "$BIN"
[ -f "$BIN/$TARGET_BIN" ] || fail "The bundle did not contain $TARGET_BIN." "It may be built for a different version. Tell your setup agent."
chmod +x "$BIN/$TARGET_BIN" 2>/dev/null || true
ok "installed: $BIN/$TARGET_BIN"

say "Setup complete. Start the bridge yourself with (this script does NOT start it):"
cat <<EOF

  "$BIN/$TARGET_BIN" \\
    --host 127.0.0.1 --port 17321 \\
    --config-file daily-content-pipeline/collector/collector_config.json \\
    --output-dir daily-content-pipeline/collector/inbox \\
    --persistent

EOF
info "One-time: install the Chrome extension via Developer Mode (see AGENT_RUNBOOK.md)."
