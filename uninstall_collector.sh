#!/usr/bin/env bash
# Solo Agency — Local Collector uninstall (macOS + Linux).
#
# Reverses setup_collector.sh: stops the autostart supervisor it registered, kills
# the collector-bridge process, releases the one-key-one-install entitlement seat
# (best effort), and deletes the install's runtime state — leaving the machine as
# if Solo Agency had never been installed, MODULO a short list of things this
# script cannot safely touch on its own (an AI desktop app's own Scheduled panel,
# another agent runtime's automations, the unpacked Chrome extension entry) —
# those are printed, never silently skipped.
#
#   Usage:  bash uninstall_collector.sh [--dry-run] [--yes] [--root PATH ...]
#                                       [--keep-source] [--keep-root-dir]
#                                       [--port N] [--open-browser]
#
# Safe to run with no arguments: it discovers every Solo Agency install on this
# machine (script location, launchd/systemd registrations, the process on the
# collector port) and asks ONE y/N confirmation before touching anything. Pass
# --dry-run to see the plan and change nothing, or --yes to skip the prompt (this
# is what an agent should pass after the human has explicitly confirmed).
#
# It NEVER deletes anything outside a validated Solo Agency install root, and it
# NEVER kills a process whose command does not contain "collector-bridge".

set -euo pipefail
# Every mutating command below runs inside an `if`/`&&`/`||` guard or is
# followed by `|| true` — set -e's documented exemption for commands that are
# part of a test/&&/|| list (everywhere except the final command of such a
# list) is exactly what makes "one step fails, the rest still runs" hold.

DRY_RUN=0
ASSUME_YES=0
KEEP_SOURCE=0
KEEP_ROOT_DIR=0
OPEN_BROWSER=0
PORT="${SOLO_AGENCY_BRIDGE_PORT:-17321}"
declare -a EXPLICIT_ROOTS=()

say()  { printf '\n\033[1m%s\033[0m\n' "$*"; }
info() { printf '  %s\n' "$*"; }
ok()   { printf '  \033[32m✓\033[0m %s\n' "$*"; }
warn() { printf '  \033[33m!\033[0m %s\n' "$*" >&2; }
err()  { printf '  \033[31m✗\033[0m %s\n' "$*" >&2; }

usage() {
  cat <<'USAGE'
Usage: bash uninstall_collector.sh [options]

Options:
  --root PATH       Uninstall this install root (repeatable for multiple installs).
                     Without --root, every install found on this machine is discovered
                     and handled.
  --dry-run          Print the full plan for every root found and change NOTHING. Exit 0.
  --yes              Do not ask for confirmation (the agent's automated path — use only
                     after the human has explicitly confirmed, in their own words, that
                     they want everything removed).
  --keep-source      Do not delete <root>/solo-agency (the source checkout).
  --keep-root-dir    Do not rmdir <root> even if it ends up empty.
  --port N           Collector bridge port to check for a stray listener (default 17321,
                     or $SOLO_AGENCY_BRIDGE_PORT).
  --open-browser     Also open chrome://extensions so the human can remove the extension
                     entries by hand (macOS/Linux; best effort).
  -h, --help         Show this help.

Exit codes: 0 = everything scripted succeeded (or --dry-run). 2 = at least one
scripted step failed, or a --root was refused for safety — see the summary.
USAGE
}

while [ $# -gt 0 ]; do
  case "$1" in
    --root) [ $# -ge 2 ] || { err "--root needs a value"; exit 2; }; EXPLICIT_ROOTS+=("$2"); shift 2 ;;
    --dry-run) DRY_RUN=1; shift ;;
    --yes) ASSUME_YES=1; shift ;;
    --keep-source) KEEP_SOURCE=1; shift ;;
    --keep-root-dir) KEEP_ROOT_DIR=1; shift ;;
    --port) [ $# -ge 2 ] || { err "--port needs a value"; exit 2; }; PORT="$2"; shift 2 ;;
    --open-browser) OPEN_BROWSER=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) err "Unknown argument: $1"; usage; exit 2 ;;
  esac
done

# --- checksum-free helpers ----------------------------------------------------
os="$(uname -s 2>/dev/null || echo unknown)"
case "$os" in Darwin) O=darwin;; Linux) O=linux;; *) O=other;; esac

sha8_of_string() {
  printf '%s' "$1" | { shasum -a 256 2>/dev/null || sha256sum; } | awk '{print substr($1,1,8)}'
}

abs_path() {
  # Resolve to an absolute, symlink- and case-canonicalized path without
  # requiring the target to exist. `pwd -P` (not plain `pwd`) is load-bearing:
  # it resolves symlinks AND, on a case-insensitive-but-preserving filesystem
  # (default APFS/HFS+ on macOS, NTFS on Windows), returns the actual on-disk
  # spelling regardless of the case the caller passed in — verified on this
  # machine (`cd .../realcasedir && pwd -P` -> `.../RealCaseDir`, and a
  # symlink alias resolves to its real target). Plain `pwd` does neither.
  local p="$1"
  if [ -d "$p" ]; then (cd -P "$p" 2>/dev/null && pwd -P) || printf '%s\n' "$p"
  else
    local d b
    d="$(dirname "$p")"; b="$(basename "$p")"
    if [ -d "$d" ]; then
      local dc
      dc="$(cd -P "$d" 2>/dev/null && pwd -P)" || dc="$d"
      printf '%s/%s\n' "$dc" "$b"
    else
      printf '%s\n' "$p"
    fi
  fi
}

# True if $1 and $2 refer to the same directory. Prefers filesystem identity
# (device+inode via `stat`) when both exist, since that is correct even if a
# caller handed in a path that bypassed abs_path's canonicalization (symlink
# alias, different case); falls back to a plain string compare only when
# `stat` is unavailable or one side doesn't exist yet (e.g. planning delete of
# an already-removed root).
same_path() {
  local a="$1" b="$2" sa sb
  if [ -e "$a" ] && [ -e "$b" ] && command -v stat >/dev/null 2>&1; then
    sa="$(stat -f '%d:%i' "$a" 2>/dev/null || stat -c '%d:%i' "$a" 2>/dev/null || true)"
    sb="$(stat -f '%d:%i' "$b" 2>/dev/null || stat -c '%d:%i' "$b" 2>/dev/null || true)"
    if [ -n "$sa" ] && [ -n "$sb" ]; then
      [ "$sa" = "$sb" ]
      return $?
    fi
  fi
  [ "$a" = "$b" ]
}

# --- safety invariant: the only guard standing between this script and a wide
# rm -rf. A root is valid only if it is not "/", not $HOME (including a
# symlinked alias or differently-cased spelling of $HOME — see same_path()),
# and actually looks like a Solo Agency install (carries daily-content-pipeline/
# or solo-agency-local-collector/). Every delete target below is built by
# joining a root that passed THIS check with a fixed, hardcoded subdirectory
# name — never a user-supplied glob or pattern.
HOME_CANON="$(abs_path "$HOME")"
root_is_safe() {
  local r="$1"
  [ -n "$r" ] || return 1
  [ "$r" != "/" ] || return 1
  # same_path compares by device+inode when both paths exist, so a symlinked
  # alias to $HOME or a differently-cased spelling of it (both real risks on
  # a case-insensitive-preserving filesystem) is caught the same as the
  # literal path — a plain string `!=` against $HOME is not enough.
  ! same_path "$r" "$HOME_CANON" || return 1
  ! same_path "$r" "$HOME" || return 1
  # A relative or empty resolution is never trusted.
  case "$r" in /*) : ;; *) return 1 ;; esac
  if [ -d "$r/daily-content-pipeline" ] || [ -d "$r/solo-agency-local-collector" ]; then
    return 0
  fi
  return 1
}

# --- root discovery ------------------------------------------------------------
declare -a FOUND_ROOTS=()

add_root() {
  local r
  r="$(abs_path "$1")"
  local existing
  for existing in "${FOUND_ROOTS[@]:-}"; do [ "$existing" = "$r" ] && return 0; done
  FOUND_ROOTS+=("$r")
}

discover_from_script_location() {
  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd || echo "")"
  [ -n "$script_dir" ] || return 0
  # <root>/solo-agency/solo-agency-collector/uninstall_collector.sh
  if [ -d "$script_dir/../../daily-content-pipeline" ] || [ -d "$script_dir/../../solo-agency-local-collector" ]; then
    add_root "$script_dir/../.."
  fi
  # <root>/solo-agency-local-collector/uninstall_collector.sh (copied here by setup_collector.sh)
  if [ -d "$script_dir/../daily-content-pipeline" ]; then
    add_root "$script_dir/.."
  fi
}

root_from_bin_path() {
  # <root>/solo-agency-local-collector/bin/collector-bridge-<os>-<arch>[.exe] -> <root>
  local bin_path="$1"
  case "$bin_path" in
    */solo-agency-local-collector/bin/collector-bridge-*)
      local d1 d2 d3
      d1="$(dirname "$bin_path")"       # .../solo-agency-local-collector/bin
      d2="$(dirname "$d1")"             # .../solo-agency-local-collector
      d3="$(dirname "$d2")"             # <root>
      printf '%s\n' "$d3"
      ;;
    *) return 1 ;;
  esac
}

discover_from_launchd() {
  [ "$O" = "darwin" ] || return 0
  local dir="$HOME/Library/LaunchAgents"
  [ -d "$dir" ] || return 0
  local f
  for f in "$dir"/com.solo-agency.collector.*.plist; do
    [ -f "$f" ] || continue
    local bin_path
    bin_path="$(awk '/<key>ProgramArguments<\/key>/{found=1; next} found && /<string>/{gsub(/.*<string>|<\/string>.*/,""); print; exit}' "$f" 2>/dev/null)"
    [ -n "$bin_path" ] || continue
    local r
    r="$(root_from_bin_path "$bin_path")" || continue
    add_root "$r"
  done
}

discover_from_systemd() {
  [ "$O" = "linux" ] || return 0
  local dir="$HOME/.config/systemd/user"
  [ -d "$dir" ] || return 0
  local f
  for f in "$dir"/solo-agency-collector-*.service; do
    [ -f "$f" ] || continue
    local bin_path
    # Extract only the binary path, not word-split on whitespace: `.*` is
    # greedy so it consumes up to the LAST "collector-bridge-<token>" run in
    # the line, which correctly keeps any spaces that are part of the path
    # itself (e.g. a root under a synced-folder mount named with spaces) while
    # still stopping before the trailing ` --host ...` flags.
    bin_path="$(sed -n 's/^ExecStart=//p' "$f" 2>/dev/null | head -1 | grep -oE '.*collector-bridge-[^[:space:]]+' || true)"
    [ -n "$bin_path" ] || continue
    local r
    r="$(root_from_bin_path "$bin_path")" || continue
    add_root "$r"
  done
}

discover_from_port_and_processes() {
  command -v lsof >/dev/null 2>&1 || return 0
  local pids p cmd bin_path r
  pids="$(lsof -tiTCP:"$PORT" -sTCP:LISTEN 2>/dev/null || true)"
  pids="$pids $(pgrep -f 'collector-bridge' 2>/dev/null || true)"
  for p in $pids; do
    [ -n "$p" ] || continue
    cmd="$(ps -p "$p" -o command= 2>/dev/null || true)"
    printf '%s' "$cmd" | grep -q "collector-bridge" || continue
    # Same non-word-splitting extraction as discover_from_systemd above.
    bin_path="$(printf '%s' "$cmd" | grep -oE '.*collector-bridge-[^[:space:]]+' || true)"
    r="$(root_from_bin_path "$bin_path")" || continue
    add_root "$r"
  done
}

if [ "${#EXPLICIT_ROOTS[@]}" -gt 0 ]; then
  for r in "${EXPLICIT_ROOTS[@]}"; do add_root "$r"; done
else
  discover_from_script_location
  discover_from_launchd
  discover_from_systemd
  discover_from_port_and_processes
fi

if [ "${#FOUND_ROOTS[@]}" -eq 0 ]; then
  say "No Solo Agency install found on this machine."
  info "(searched: this script's own location, launchd/systemd registrations, and port $PORT)"
  exit 0
fi

# --- MULTI_BRAIN_OPERATIONS.md install-root pointer-file signatures ----------
AGENTS_SIGNATURE="This folder is a LIVE Solo Agency install, not a fresh setup."
CLAUDE_SIGNATURE="Read \`AGENTS.md\` in this same folder now, then \`solo-agency/AGENTS.md\` in full"

# run_with_timeout SECONDS -- CMD...   (portable: uses `timeout`/`gtimeout` if present,
# else a manual background+kill so a hung binary can never stall the uninstall).
run_with_timeout() {
  local secs="$1"; shift
  if command -v timeout >/dev/null 2>&1; then timeout "$secs" "$@"; return $?; fi
  if command -v gtimeout >/dev/null 2>&1; then gtimeout "$secs" "$@"; return $?; fi
  "$@" &
  local pid=$!
  ( sleep "$secs"; kill -0 "$pid" 2>/dev/null && kill "$pid" 2>/dev/null ) &
  local watchdog=$!
  local rc
  wait "$pid"; rc=$?
  kill "$watchdog" 2>/dev/null || true
  return "$rc"
}

# --- per-root plan/execute -----------------------------------------------------
# REMOVED / SKIPPED / MANUAL / FAILED accumulate across every root for the final
# summary; ANY_FAILED / ANY_REFUSED decide the exit code.
declare -a REMOVED=() SKIPPED=() MANUAL=() FAILED=()
ANY_FAILED=0
ANY_REFUSED=0

process_root() {
  local root="$1" mode="$2"   # mode: plan | execute
  local prefix
  if [ "$mode" = "plan" ]; then prefix="[would]"; else prefix="[did] "; fi

  say "Root: $root  ($([ "$mode" = plan ] && echo "PLAN — nothing changes" || echo "EXECUTING"))"

  local insthash launchd_label launchd_plist systemd_unit systemd_file
  insthash="$(sha8_of_string "$root")"
  launchd_label="com.solo-agency.collector.$insthash"
  launchd_plist="$HOME/Library/LaunchAgents/$launchd_label.plist"
  systemd_unit="solo-agency-collector-$insthash.service"
  systemd_file="$HOME/.config/systemd/user/$systemd_unit"

  # setup_collector.sh hashes ROOT with plain `cd && pwd` (not the case/symlink
  # -canonicalizing `pwd -P` this script uses in abs_path), so an insthash
  # computed here from a differently-cased or symlinked --root can miss a job
  # that is really this root's own, even though root_is_safe already proved
  # (via same_path/stat identity) that it IS this root. Before concluding
  # "nothing registered", fall back to scanning every registered job's own
  # recorded binary path and matching it to $root by filesystem identity —
  # this finds a job that setup actually created for this root regardless of
  # which case/spelling was hashed into its name at setup time.
  if [ "$O" = "darwin" ] && [ ! -f "$launchd_plist" ] && [ -d "$HOME/Library/LaunchAgents" ]; then
    local f bp r2
    for f in "$HOME/Library/LaunchAgents"/com.solo-agency.collector.*.plist; do
      [ -f "$f" ] || continue
      bp="$(awk '/<key>ProgramArguments<\/key>/{found=1; next} found && /<string>/{gsub(/.*<string>|<\/string>.*/,""); print; exit}' "$f" 2>/dev/null)"
      [ -n "$bp" ] || continue
      r2="$(root_from_bin_path "$bp")" || continue
      if same_path "$r2" "$root"; then
        launchd_plist="$f"
        launchd_label="$(basename "$f" .plist)"
        break
      fi
    done
  elif [ "$O" = "linux" ] && [ ! -f "$systemd_file" ] && [ -d "$HOME/.config/systemd/user" ]; then
    local f bp r2
    for f in "$HOME/.config/systemd/user"/solo-agency-collector-*.service; do
      [ -f "$f" ] || continue
      bp="$(sed -n 's/^ExecStart=//p' "$f" 2>/dev/null | head -1 | grep -oE '.*collector-bridge-[^[:space:]]+' || true)"
      [ -n "$bp" ] || continue
      r2="$(root_from_bin_path "$bp")" || continue
      if same_path "$r2" "$root"; then
        systemd_file="$f"
        systemd_unit="$(basename "$f")"
        break
      fi
    done
  fi

  # (a) entitlement seat release — best effort, never fails the uninstall.
  local bin_path=""
  for f in "$root/solo-agency-local-collector/bin/collector-bridge-"*; do
    [ -f "$f" ] || continue
    bin_path="$f"
    break
  done
  local has_key=0
  if [ -n "$bin_path" ]; then
    if find "$root/daily-content-pipeline/clients" -maxdepth 6 -name 'provider_config.local.json' -print -quit 2>/dev/null | grep -q .; then
      has_key=1
    fi
    if env | grep -Eqi '^(SOLO|OUTREACHCRM)[A-Z0-9_]*KEY='; then has_key=1; fi
  fi
  if [ -n "$bin_path" ] && [ "$has_key" = "1" ]; then
    if [ "$mode" = "plan" ]; then
      info "$prefix release entitlement seat: $bin_path tool entitlement release --pipeline $root/daily-content-pipeline"
    else
      info "Releasing entitlement seat (best effort, 20s timeout)..."
      if run_with_timeout 20 "$bin_path" tool entitlement release --pipeline "$root/daily-content-pipeline" >/tmp/solo-agency-uninstall-entitlement.$$ 2>&1; then
        ok "entitlement seat released"
      else
        warn "entitlement release did not succeed (non-fatal): $(tail -c 400 /tmp/solo-agency-uninstall-entitlement.$$ 2>/dev/null | tr '\n' ' ')"
      fi
      rm -f /tmp/solo-agency-uninstall-entitlement.$$
    fi
  else
    info "$prefix skip entitlement release (no bridge binary or no provider key found)"
  fi

  # (b) stop the supervisor.
  if [ "$O" = "darwin" ]; then
    if [ -f "$launchd_plist" ]; then
      if [ "$mode" = "plan" ]; then
        info "$prefix launchctl bootout gui/\$(id -u)/$launchd_label ; rm $launchd_plist"
      else
        launchctl bootout "gui/$(id -u)/$launchd_label" >/dev/null 2>&1 || true
        if rm -f "$launchd_plist"; then ok "removed launchd job $launchd_label"; REMOVED+=("$root: launchd $launchd_label")
        else warn "could not remove $launchd_plist"; FAILED+=("$root: rm $launchd_plist"); fi
      fi
    else
      info "$prefix no launchd job registered for this root ($launchd_label) — nothing to stop"
    fi
  elif [ "$O" = "linux" ]; then
    if [ -f "$systemd_file" ]; then
      if [ "$mode" = "plan" ]; then
        info "$prefix systemctl --user stop/disable $systemd_unit ; rm $systemd_file ; daemon-reload"
      else
        systemctl --user stop "$systemd_unit" >/dev/null 2>&1 || true
        systemctl --user disable "$systemd_unit" >/dev/null 2>&1 || true
        if rm -f "$systemd_file"; then
          systemctl --user daemon-reload >/dev/null 2>&1 || true
          ok "removed systemd unit $systemd_unit"
          REMOVED+=("$root: systemd $systemd_unit")
          info "note: if 'loginctl enable-linger' was turned on for this account it is left as-is (it may serve other installs) — disable it yourself with: loginctl disable-linger \$(id -un)"
        else
          warn "could not remove $systemd_file"; FAILED+=("$root: rm $systemd_file")
        fi
      fi
    else
      info "$prefix no systemd unit registered for this root ($systemd_unit) — nothing to stop"
    fi
  fi

  # (c) kill the bridge process — PID file, then anything still on the port —
  # and ONLY when its command line contains "collector-bridge".
  local pid_file="$root/solo-agency-local-collector/collector.pid"
  if [ -f "$pid_file" ]; then
    local pid cmd
    pid="$(cat "$pid_file" 2>/dev/null || true)"
    if [ -n "$pid" ] && kill -0 "$pid" >/dev/null 2>&1; then
      cmd="$(ps -p "$pid" -o command= 2>/dev/null || true)"
      if printf '%s' "$cmd" | grep -q "collector-bridge"; then
        if [ "$mode" = "plan" ]; then
          info "$prefix kill PID $pid ($cmd)"
        else
          kill "$pid" >/dev/null 2>&1 || true; sleep 1
          kill -0 "$pid" >/dev/null 2>&1 && kill -9 "$pid" >/dev/null 2>&1 || true
          ok "stopped collector-bridge PID $pid"
        fi
      else
        info "$prefix collector.pid PID $pid is not a collector-bridge process — leaving it alone"
      fi
    fi
  fi
  if command -v lsof >/dev/null 2>&1; then
    for p in $(lsof -tiTCP:"$PORT" -sTCP:LISTEN 2>/dev/null || true); do
      local cmd cmd_bin cmd_root
      cmd="$(ps -p "$p" -o command= 2>/dev/null || true)"
      if printf '%s' "$cmd" | grep -q "collector-bridge"; then
        # The collector port (17321 by default) is a single shared value
        # across every install on the machine — "the listener's command
        # contains collector-bridge" is NOT proof it belongs to THIS root.
        # Resolve its actual binary path back to a root and require that
        # root be this one (by filesystem identity) before killing it; a
        # collector-bridge for a different, still-installed root is left
        # running, exactly like the non-collector case below.
        cmd_bin="$(printf '%s' "$cmd" | grep -oE '.*collector-bridge-[^[:space:]]+' || true)"
        cmd_root="$(root_from_bin_path "$cmd_bin" 2>/dev/null || true)"
        if [ -n "$cmd_root" ] && ! same_path "$cmd_root" "$root"; then
          info "port $PORT is held by another install's collector-bridge (PID $p, root $cmd_root) — left running"
          [ "$mode" = "execute" ] && MANUAL+=("$root: port $PORT held by a DIFFERENT Solo Agency install's collector-bridge (PID $p, root $cmd_root) — not touched; stop it yourself if you actually mean to remove that install too")
          continue
        fi
        if [ "$mode" = "plan" ]; then
          info "$prefix kill listener on port $PORT: PID $p ($cmd)"
        else
          kill "$p" >/dev/null 2>&1 || true; sleep 1
          kill -0 "$p" >/dev/null 2>&1 && kill -9 "$p" >/dev/null 2>&1 || true
          ok "stopped listener on port $PORT (PID $p)"
        fi
      else
        info "port $PORT is held by a NON-collector process (PID $p): $cmd — left running"
        [ "$mode" = "execute" ] && MANUAL+=("$root: port $PORT held by non-collector PID $p — stop it yourself if needed")
      fi
    done
  fi

  # (d) delete runtime state — every path below is <root> (already safety-checked)
  # joined with a fixed subdirectory name. No globs, no user input in the path.
  delete_dir() {
    local target="$1" desc="$2"
    if [ ! -e "$target" ]; then
      info "$prefix $desc: nothing at $target"
      return 0
    fi
    if [ "$mode" = "plan" ]; then
      info "$prefix rm -rf $target"
    else
      if rm -rf "$target"; then ok "removed $desc"; REMOVED+=("$root: $desc ($target)")
      else warn "failed to remove $target"; FAILED+=("$root: rm -rf $target"); fi
    fi
  }
  delete_dir "$root/solo-agency-local-collector" "local collector runtime"
  delete_dir "$root/extensions" "per-client Chrome extension copies"
  delete_dir "$root/daily-content-pipeline" "pipeline state (CRM, content library, config, secrets)"

  # <root>/outreach: no repo-defined marker distinguishes a live install's outreach
  # STATE directory from an accidental source checkout at that path (the source
  # repo's own outreach/ folder is a maintainer checkout, not install state, and
  # playbooks/07 + outreach/playbooks name the state path as
  # {agency_root}/outreachcrm — not {root}/outreach). Be conservative: never touch it.
  if [ -e "$root/outreach" ]; then
    info "left in place: not a confirmed Solo Agency state dir: $root/outreach (no repo-defined marker — verify and remove by hand if it is agency state)"
    [ "$mode" = "execute" ] && SKIPPED+=("$root: outreach (no marker, left in place)")
  fi
  if [ -e "$root/outreachcrm" ]; then
    delete_dir "$root/outreachcrm" "outreach CRM state"
  fi

  for pf in AGENTS.md CLAUDE.md; do
    local p="$root/$pf" sig=""
    [ "$pf" = "AGENTS.md" ] && sig="$AGENTS_SIGNATURE" || sig="$CLAUDE_SIGNATURE"
    if [ -f "$p" ]; then
      if grep -qF "$sig" "$p" 2>/dev/null; then
        if [ "$mode" = "plan" ]; then
          info "$prefix rm $p (matches MULTI_BRAIN_OPERATIONS.md pointer-file signature)"
        else
          if rm -f "$p"; then ok "removed pointer file $pf"; REMOVED+=("$root: $pf"); else warn "failed to remove $p"; FAILED+=("$root: rm $p"); fi
        fi
      else
        info "left in place: not a Solo Agency pointer file: $p"
        [ "$mode" = "execute" ] && SKIPPED+=("$root: $pf (no signature match, left in place)")
      fi
    fi
  done

  if [ "$KEEP_SOURCE" = "1" ]; then
    info "$prefix keep $root/solo-agency (--keep-source)"
    [ -e "$root/solo-agency" ] && [ "$mode" = "execute" ] && SKIPPED+=("$root: solo-agency (kept: --keep-source)")
  else
    delete_dir "$root/solo-agency" "source checkout"
  fi

  if [ "$KEEP_ROOT_DIR" = "1" ]; then
    info "$prefix keep root directory (--keep-root-dir)"
  else
    if [ "$mode" = "plan" ]; then
      info "$prefix rmdir $root (only if empty afterward)"
    else
      if rmdir "$root" >/dev/null 2>&1; then
        ok "removed empty root directory"
        REMOVED+=("$root: root directory")
      else
        local remaining
        remaining="$(ls -A "$root" 2>/dev/null | tr '\n' ' ')"
        info "root directory left in place — not empty: ${remaining:-<unreadable>}"
      fi
    fi
  fi

  # (e) CLI-era Claude scheduled-task folders. ~/.claude/scheduled-tasks is a
  # SINGLE machine-wide directory shared by every Solo Agency install (and every
  # other Claude project) on this box — it carries no root/insthash in its own
  # name, so a "*solo-agency*" glob here cannot tell "this root's task" apart
  # from a different, still-installed root's task, or even another client's
  # (e.g. clientB's "clientB-solo-agency-daily-run" while only clientA's root
  # is being uninstalled). Never auto-delete it: list matches for the human to
  # review and remove by hand, the same treatment already given to the Claude
  # desktop Scheduled panel and Codex automations right below.
  local claude_dir="${SOLO_AGENCY_CLAUDE_SCHEDULED_TASKS_DIR:-$HOME/.claude/scheduled-tasks}"
  if [ -d "$claude_dir" ]; then
    local hit
    for hit in "$claude_dir"/*solo-agency*; do
      [ -e "$hit" ] || continue
      info "$prefix NOT deleted (machine-wide, not root-scoped — review by hand): $hit"
      [ "$mode" = "execute" ] && MANUAL+=("(machine-wide, review before deleting — may belong to a different install) $hit")
    done
  fi

  if [ "$mode" = "execute" ]; then
    MANUAL+=(
      "$root: Claude desktop app — Scheduled panel entries for this install (e.g. '<client>-solo-agency-daily-run', 'solo-agency-github-update-watch') — remove them there; this script cannot reach that app's own registry."
      "$root: Codex automations for this install (if any) — remove them in Codex's own automations UI."
      "$root: chrome://extensions — the unpacked Solo Agency extension entries loaded from $root/extensions/*/ (one per Chrome profile) — Load unpacked folders are gone from disk now, but Chrome keeps the entry until you remove it there."
    )
  else
    info "$prefix note manual leftovers (Claude desktop Scheduled panel, Codex automations, chrome://extensions entries)"
  fi
  if [ "$mode" = "execute" ] && [ "$OPEN_BROWSER" = "1" ]; then
    if [ "$O" = "darwin" ]; then open -a "Google Chrome" "chrome://extensions/" >/dev/null 2>&1 || true
    elif command -v xdg-open >/dev/null 2>&1; then xdg-open "chrome://extensions/" >/dev/null 2>&1 || true
    elif command -v google-chrome >/dev/null 2>&1; then google-chrome "chrome://extensions/" >/dev/null 2>&1 || true
    fi
  fi
}

# --- validate every root before touching anything ------------------------------
# A root that simply does not exist (a previous uninstall already rmdir'd it, or
# --root was given a typo of a path nothing ever occupied) is a no-op, not a
# safety refusal — that is what makes running this twice in a row idempotent.
# An EXISTING path that fails the marker check (including "/" and "$HOME", which
# always exist) is refused: this script will never guess that unrecognized,
# still-present content is safe to delete.
declare -a VALID_ROOTS=()
for r in "${FOUND_ROOTS[@]}"; do
  if [ ! -e "$r" ]; then
    info "Nothing to uninstall — path does not exist (already removed): $r"
  elif root_is_safe "$r"; then
    VALID_ROOTS+=("$r")
  else
    err "Refusing root (fails safety check — not \"/\", not \$HOME, and must contain daily-content-pipeline/ or solo-agency-local-collector/): $r"
    ANY_REFUSED=1
  fi
done

if [ "${#VALID_ROOTS[@]}" -eq 0 ]; then
  say "Nothing to uninstall — no valid Solo Agency install root."
  exit $([ "$ANY_REFUSED" = "1" ] && echo 2 || echo 0)
fi

say "Solo Agency Local Collector uninstall — plan"
info "Install root(s) found: ${#VALID_ROOTS[@]}"
for r in "${VALID_ROOTS[@]}"; do info "  - $r"; done
[ "$KEEP_SOURCE" = "1" ] && info "--keep-source: source checkout (solo-agency/) will be kept"
[ "$KEEP_ROOT_DIR" = "1" ] && info "--keep-root-dir: root directory will be kept even if empty"

for r in "${VALID_ROOTS[@]}"; do process_root "$r" "plan"; done

if [ "$DRY_RUN" = "1" ]; then
  say "Dry run — nothing was changed."
  exit $([ "$ANY_REFUSED" = "1" ] && echo 2 || echo 0)
fi

if [ "$ASSUME_YES" != "1" ]; then
  say "This will permanently delete the above for ${#VALID_ROOTS[@]} install(s)."
  printf "Proceed? [y/N] "
  read -r ans || ans=""
  case "$ans" in
    y|Y|yes|YES) : ;;
    *) say "Aborted — nothing changed."; exit 0 ;;
  esac
fi

say "Uninstalling..."
for r in "${VALID_ROOTS[@]}"; do process_root "$r" "execute"; done

say "Summary"
info "Removed (${#REMOVED[@]}):"
for x in "${REMOVED[@]:-}"; do [ -n "$x" ] && info "  - $x"; done
info "Skipped/left in place (${#SKIPPED[@]}):"
for x in "${SKIPPED[@]:-}"; do [ -n "$x" ] && info "  - $x"; done
info "Manual leftovers to remove yourself (${#MANUAL[@]}):"
for x in "${MANUAL[@]:-}"; do [ -n "$x" ] && info "  - $x"; done
if [ "${#FAILED[@]}" -gt 0 ]; then
  warn "Failed steps (${#FAILED[@]}):"
  for x in "${FAILED[@]}"; do warn "  - $x"; done
  ANY_FAILED=1
fi
info "Untouched by design: ~/.claude/projects transcripts and this AI runtime's own CLAUDE.md — they are the AI runtime's own data, not Solo Agency's."

if [ "$ANY_FAILED" = "1" ] || [ "$ANY_REFUSED" = "1" ]; then
  say "Uninstall finished WITH ERRORS — see Failed steps above."
  exit 2
fi
say "Uninstall complete."
exit 0
