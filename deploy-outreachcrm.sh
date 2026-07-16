#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_COLLECTOR_DIR="$SCRIPT_DIR/solo-agency-collector"
BRIDGE_DIR="$TARGET_COLLECTOR_DIR/bridge-go"
EXTENSION_DIR="$TARGET_COLLECTOR_DIR/chrome-extension"
DIST_DIR="$TARGET_COLLECTOR_DIR/dist"
SOURCE_SKILLS_DIR="${SOLO_AGENCY_SOURCE_SKILLS_DIR:-}"
TARGET_SKILLS_DIR="$SCRIPT_DIR/playbooks/skills"
GITHUB_RAW_BASE="https://raw.githubusercontent.com/soloagency/solo-agency/main/solo-agency-collector"
COLLECTOR_VERSION="${SOLO_AGENCY_COLLECTOR_VERSION:-0.1.0}"
BRIDGE_ZIP="$DIST_DIR/collector-bridge-binaries-$COLLECTOR_VERSION.zip"
EXTENSION_ROOT_ZIP="$DIST_DIR/chrome-extension-collector-root-$COLLECTOR_VERSION.zip"
EXTENSION_NESTED_ZIP="$DIST_DIR/chrome-extension-collector-$COLLECTOR_VERSION.zip"
SHA_FILE="$DIST_DIR/SHA256SUMS"
GO_CMD=""
GOFMT_CMD=""
MODE="all"
DEFAULT_GIT_REMOTE_URL="git@github.com-soloagency:soloagency/solo-agency.git"
AUTO_GIT_DEPLOY="${SOLO_AGENCY_AUTO_GIT_DEPLOY:-1}"
GIT_PUSH="${SOLO_AGENCY_GIT_PUSH:-1}"
GIT_REMOTE_NAME="${SOLO_AGENCY_GIT_REMOTE_NAME:-origin}"
GIT_REMOTE_URL="${SOLO_AGENCY_GIT_REMOTE_URL:-$DEFAULT_GIT_REMOTE_URL}"
GIT_DEFAULT_BRANCH="${SOLO_AGENCY_GIT_BRANCH:-main}"
GIT_AUTHOR_NAME="${SOLO_AGENCY_GIT_USER_NAME:-Solo Agency}"
GIT_AUTHOR_EMAIL="${SOLO_AGENCY_GIT_USER_EMAIL:-soloagency-deploy@users.noreply.github.com}"
DEEPSEEK_MODEL="${SOLO_AGENCY_DEEPSEEK_MODEL:-deepseek-chat}"
DEEPSEEK_API_URL="${SOLO_AGENCY_DEEPSEEK_API_URL:-https://api.deepseek.com/chat/completions}"
DEEPSEEK_API_KEY_FALLBACK=""  # removed: no embedded fallback secret; set DEEPSEEK_API_KEY env var or the AI commit-message step is skipped
GIT_COMMIT_PREFIX="${SOLO_AGENCY_COMMIT_PREFIX:-deploy}"
GIT_DRY_RUN="${SOLO_AGENCY_GIT_DRY_RUN:-0}"

usage() {
  cat >&2 <<'USAGE'
Usage:
  ./deploy-soloagency.sh [all|--collector-only|--skills-only|--check-only|--git-only] [--no-git] [--no-push] [--dry-run-git]

Default/all:
  - normalize package names/URLs
  - gofmt + go test collector bridge
  - cross-build bridge binaries
  - rebuild collector dist zips
  - refresh SHA256SUMS
  - rebuild playbook skill zips
  - run upload preflight checks
  - commit and push Solo Agency repo updates

Environment:
  SOLO_AGENCY_SOURCE_SKILLS_DIR=/absolute/path/to/skills
  SOLO_AGENCY_COLLECTOR_VERSION=0.1.0
  SOLO_AGENCY_GO_CACHE=/tmp/soloagency-go-cache
  SOLO_AGENCY_AUTO_GIT_DEPLOY=1
  SOLO_AGENCY_GIT_PUSH=1
  SOLO_AGENCY_GIT_REMOTE_NAME=origin
  SOLO_AGENCY_GIT_REMOTE_URL=git@github.com-soloagency:soloagency/solo-agency.git
  SOLO_AGENCY_GIT_BRANCH=main
  SOLO_AGENCY_GIT_USER_NAME="Solo Agency"
  SOLO_AGENCY_GIT_USER_EMAIL="soloagency-deploy@users.noreply.github.com"
  SOLO_AGENCY_DEEPSEEK_MODEL=deepseek-chat
  SOLO_AGENCY_DEEPSEEK_API_URL=https://api.deepseek.com/chat/completions
  DEEPSEEK_API_KEY=sk-...
  deepseek_api_key=sk-...
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    all|--collector-only|--skills-only|--check-only|--git-only)
      MODE="$1"
      ;;
    --no-git)
      AUTO_GIT_DEPLOY=0
      ;;
    --no-push)
      GIT_PUSH=0
      ;;
    --dry-run-git)
      GIT_DRY_RUN=1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage
      exit 1
      ;;
  esac
  shift
done

log() {
  printf '\n==> %s\n' "$*"
}

die() {
  echo "ERROR: $*" >&2
  exit 1
}

have() {
  command -v "$1" >/dev/null 2>&1
}

require_cmd() {
  have "$1" || die "Missing required command: $1"
}

git_cmd() {
  git -C "$SCRIPT_DIR" "$@"
}

solo_git_root() {
  git_cmd rev-parse --show-toplevel 2>/dev/null || true
}

is_solo_git_repo() {
  local root
  root="$(solo_git_root)"
  [ "$root" = "$SCRIPT_DIR" ] && [ -d "$SCRIPT_DIR/.git" ]
}

ensure_solo_git_repo() {
  require_cmd git

  if is_solo_git_repo; then
    return
  fi

  local detected_root
  detected_root="$(solo_git_root)"

  if [ -z "$GIT_REMOTE_URL" ]; then
    cat >&2 <<EOF
ERROR: $SCRIPT_DIR is not an independent git repo yet.

Detected git root: ${detected_root:-none}

Solo Agency is nested inside the WideCast/transcoder workspace, so deploy must
NOT use the parent repo. Run once with the Solo Agency GitHub remote URL, for
example:

  SOLO_AGENCY_GIT_REMOTE_URL=git@github.com-soloagency:soloagency/solo-agency.git bash deploy-soloagency.sh --git-only

By default deploy uses:

  $DEFAULT_GIT_REMOTE_URL

Use SOLO_AGENCY_GIT_REMOTE_URL only when you need an SSH host alias or another
remote that belongs to the Solo Agency GitHub account. After .git is initialized
inside soloagency/, future deploys can just run:

  bash deploy-soloagency.sh
EOF
    exit 1
  fi

  log "Initializing independent Solo Agency git repo"
  (
    cd "$SCRIPT_DIR"
    git init
    git checkout -B "$GIT_DEFAULT_BRANCH"
    git remote add "$GIT_REMOTE_NAME" "$GIT_REMOTE_URL"
  )
}

ensure_git_identity() {
  git_cmd config user.name "$GIT_AUTHOR_NAME"
  git_cmd config user.email "$GIT_AUTHOR_EMAIL"
}

ensure_git_remote() {
  local existing_url
  existing_url="$(git_cmd remote get-url "$GIT_REMOTE_NAME" 2>/dev/null || true)"
  if [ -z "$existing_url" ]; then
    [ -n "$GIT_REMOTE_URL" ] || die "Missing git remote '$GIT_REMOTE_NAME'. Set SOLO_AGENCY_GIT_REMOTE_URL once."
    git_cmd remote add "$GIT_REMOTE_NAME" "$GIT_REMOTE_URL"
    return
  fi
  if [ -n "$GIT_REMOTE_URL" ] && [ "$existing_url" != "$GIT_REMOTE_URL" ]; then
    die "Remote '$GIT_REMOTE_NAME' already points to '$existing_url', not SOLO_AGENCY_GIT_REMOTE_URL='$GIT_REMOTE_URL'. Fix manually or choose another SOLO_AGENCY_GIT_REMOTE_NAME."
  fi
}

current_git_branch() {
  git_cmd rev-parse --abbrev-ref HEAD
}

scan_staged_secrets() {
  log "Checking staged diff for obvious secrets"
  local secret_hits
  secret_hits="$(git_cmd diff --cached --text --unified=0 | grep -E 'wc_live_[A-Za-z0-9_-]{20,}|sk-ant-[A-Za-z0-9_-]{20,}|sk-[A-Za-z0-9_-]{20,}|api_key_local\"[[:space:]]*:[[:space:]]*\"[^\"]+' || true)"
  if [ -n "$secret_hits" ]; then
    echo "$secret_hits" >&2
    die "Refusing to commit because staged diff appears to contain a secret. Move secrets into env/local ignored config."
  fi
}

fallback_commit_message() {
  local msg_file="$1"
  local changed_files="$2"
  {
    printf '%s: update Solo Agency deployment assets\n\n' "$GIT_COMMIT_PREFIX"
    printf 'Generated by Solo Agency maintainer deploy.\n\n'
    printf 'Changed files:\n'
    printf '%s\n' "$changed_files" | sed 's/^/- /'
  } > "$msg_file"
}

deepseek_commit_message() {
  local msg_file="$1"
  local summary_file="$2"
  local response_file="$3"
  local api_key="${DEEPSEEK_API_KEY:-${deepseek_api_key:-$DEEPSEEK_API_KEY_FALLBACK}}"

  [ -n "$api_key" ] || return 1
  have curl || return 1
  have python3 || return 1

  local request_file
  request_file="$(mktemp "${TMPDIR:-/tmp}/soloagency-deepseek-request.XXXXXX")"

  SUMMARY_FILE="$summary_file" DEEPSEEK_MODEL="$DEEPSEEK_MODEL" python3 - "$request_file" <<'PY'
import json
import os
import pathlib
import sys

summary = pathlib.Path(os.environ["SUMMARY_FILE"]).read_text(encoding="utf-8")
prompt = f"""Create a concise Git commit message for Solo Agency.

Return only the commit message:
- first line: imperative subject, 72 chars or fewer
- then a blank line
- then 2-5 short bullet points if useful

Deployment summary:
{summary}
"""

body = {
    "model": os.environ["DEEPSEEK_MODEL"],
    "temperature": 0.2,
    "max_tokens": 400,
    "messages": [
        {"role": "system", "content": "You write concise Git commit messages."},
        {"role": "user", "content": prompt},
    ],
}
pathlib.Path(sys.argv[1]).write_text(json.dumps(body), encoding="utf-8")
PY

  if ! curl -sS "$DEEPSEEK_API_URL" \
    -H "Authorization: Bearer ${api_key}" \
    -H "Content-Type: application/json" \
    --data @"$request_file" \
    -o "$response_file"; then
    rm -f "$request_file"
    return 1
  fi
  rm -f "$request_file"

  python3 - "$response_file" "$msg_file" <<'PY'
import json
import pathlib
import sys

data = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
if "error" in data:
    raise SystemExit(1)
choices = data.get("choices") or []
if not choices:
    raise SystemExit(1)
message = choices[0].get("message") or {}
text = (message.get("content") or "").strip()
if not text:
    raise SystemExit(1)
pathlib.Path(sys.argv[2]).write_text(text + "\n", encoding="utf-8")
PY
}

create_commit_message() {
  local msg_file="$1"
  local summary_file="$2"
  local response_file="$3"

  if deepseek_commit_message "$msg_file" "$summary_file" "$response_file"; then
    echo "Commit message generated with DeepSeek ($DEEPSEEK_MODEL)."
    return
  fi

  echo "DeepSeek commit message generation unavailable; using local fallback."
  fallback_commit_message "$msg_file" "$(git_cmd diff --cached --name-status)"
}

git_deploy() {
  [ "$AUTO_GIT_DEPLOY" = "1" ] || {
    echo "Skipping git deploy because SOLO_AGENCY_AUTO_GIT_DEPLOY=$AUTO_GIT_DEPLOY."
    return
  }

  if [ "$MODE" = "--check-only" ]; then
    echo "Skipping git deploy for --check-only."
    return
  fi

  log "Preparing Solo Agency git commit"
  ensure_solo_git_repo
  ensure_git_identity
  ensure_git_remote

  if [ "$(solo_git_root)" != "$SCRIPT_DIR" ]; then
    die "Refusing git deploy because git root is not the Solo Agency folder."
  fi

  git_cmd add -A -- .

  if git_cmd diff --cached --quiet; then
    echo "No Solo Agency changes to commit."
    return
  fi

  scan_staged_secrets

  local summary_file msg_file response_file
  summary_file="$(mktemp "${TMPDIR:-/tmp}/soloagency-commit-summary.XXXXXX")"
  msg_file="$(mktemp "${TMPDIR:-/tmp}/soloagency-commit-message.XXXXXX")"
  response_file="$(mktemp "${TMPDIR:-/tmp}/soloagency-deepseek-response.XXXXXX")"

  {
    echo "Mode: $MODE"
    echo
    echo "Changed files:"
    git_cmd diff --cached --name-status
    echo
    echo "Diff stat:"
    git_cmd diff --cached --stat
  } > "$summary_file"

  create_commit_message "$msg_file" "$summary_file" "$response_file"

  echo "Commit message:"
  sed 's/^/  /' "$msg_file"

  if [ "$GIT_DRY_RUN" = "1" ]; then
    echo "Dry run: not committing or pushing."
    return
  fi

  git_cmd commit -F "$msg_file"

  if [ "$GIT_PUSH" = "1" ]; then
    local branch
    branch="$(current_git_branch)"
    log "Pushing Solo Agency repo to $GIT_REMOTE_NAME/$branch"
    git_cmd push -u "$GIT_REMOTE_NAME" "$branch"
  else
    echo "Skipping push because SOLO_AGENCY_GIT_PUSH=$GIT_PUSH."
  fi
}

resolve_go() {
  if [ -n "${GO_BIN:-}" ]; then
    [ -x "$GO_BIN" ] || die "GO_BIN is set but not executable: $GO_BIN"
    GO_CMD="$GO_BIN"
    return
  fi
  if have go; then
    GO_CMD="$(command -v go)"
    return
  fi
  for candidate in \
    /opt/homebrew/bin/go \
    /usr/local/go/bin/go \
    /usr/local/bin/go; do
    if [ -x "$candidate" ]; then
      GO_CMD="$candidate"
      return
    fi
  done
  die "Missing Go compiler. Install Go, add it to PATH, or run with GO_BIN=/absolute/path/to/go."
}

resolve_gofmt() {
  resolve_go
  local gofmt_candidate
  gofmt_candidate="$(dirname "$GO_CMD")/gofmt"
  if [ -x "$gofmt_candidate" ]; then
    GOFMT_CMD="$gofmt_candidate"
    return
  fi
  if have gofmt; then
    GOFMT_CMD="$(command -v gofmt)"
    return
  fi
  die "Missing gofmt. It should be installed with Go near $GO_CMD."
}

sha256_for_file() {
  local file="$1"
  if have shasum; then
    shasum -a 256 "$file" | awk '{print $1}'
  elif have sha256sum; then
    sha256sum "$file" | awk '{print $1}'
  else
    die "Missing shasum or sha256sum for checksum generation."
  fi
}

append_checksum() {
  local file="$1"
  local published_path="$2"
  printf '%s  %s\n' "$(sha256_for_file "$file")" "$published_path" >> "$SHA_FILE"
}

normalize_collector_text() {
  log "Normalizing collector package names and download URLs"
  require_cmd perl
  while IFS= read -r -d '' file; do
    perl -0pi -e "s#https://origin\\.widecast\\.ai/media-agency-collector#$GITHUB_RAW_BASE#g; s#media-agency-collector#solo-agency-collector#g; s#media_agency_playbook\\.md#SOLO_AGENCY_PLAYBOOK.md#g; s#temporary WideCast-hosted distribution#Solo Agency GitHub distribution#g; s#Temporary WideCast-Hosted Distribution#Solo Agency GitHub Distribution#g; s#Until GitHub releases and Chrome Web Store are available, agents should use the Solo Agency GitHub distribution:#Agents should use the Solo Agency GitHub repo as the default setup source. Until GitHub Releases and Chrome Web Store are available, use these raw GitHub artifact URLs:#g; s#Until GitHub releases and Chrome Web Store are available, use these temporary artifact URLs:#Use the Solo Agency GitHub repo as the default setup source. Until GitHub Releases and Chrome Web Store are available, use these raw GitHub artifact URLs:#g; s#For public release, ship prebuilt bridge binaries:#The package ships prebuilt bridge binaries through the artifact bundle:#g" "$file"
  done < <(find "$TARGET_COLLECTOR_DIR" -type f \( -name "*.md" -o -name "*.json" -o -name "*.html" -o -name "*.js" -o -name "*.go" -o -name "go.mod" -o -name "*.sh" \) -print0)
}

run_go_checks() {
  log "Formatting and testing collector bridge"
  resolve_gofmt
  echo "Using Go: $GO_CMD"
  local go_cache
  go_cache="${SOLO_AGENCY_GO_CACHE:-${TMPDIR:-/tmp}/soloagency-go-cache}"
  mkdir -p "$go_cache"
  (
    cd "$BRIDGE_DIR"
    "$GOFMT_CMD" -w main.go main_test.go
    GOCACHE="$go_cache" "$GO_CMD" test ./...
  )
}

build_bridge_binaries() {
  log "Building collector bridge binaries"
  resolve_go
  echo "Using Go: $GO_CMD"
  require_cmd zip
  mkdir -p "$DIST_DIR"
  local go_cache
  go_cache="${SOLO_AGENCY_GO_CACHE:-${TMPDIR:-/tmp}/soloagency-go-cache}"
  mkdir -p "$go_cache"

  local build_dir
  build_dir="$(mktemp -d "${TMPDIR:-/tmp}/soloagency-bridge-build.XXXXXX")"
  trap 'rm -rf "$build_dir"' RETURN

  (
    cd "$BRIDGE_DIR"
    GOCACHE="$go_cache" CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 "$GO_CMD" build -trimpath -ldflags="-s -w" -o "$build_dir/collector-bridge-darwin-arm64" ./...
    GOCACHE="$go_cache" CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 "$GO_CMD" build -trimpath -ldflags="-s -w" -o "$build_dir/collector-bridge-darwin-amd64" ./...
    GOCACHE="$go_cache" CGO_ENABLED=0 GOOS=linux GOARCH=amd64 "$GO_CMD" build -trimpath -ldflags="-s -w" -o "$build_dir/collector-bridge-linux-amd64" ./...
    GOCACHE="$go_cache" CGO_ENABLED=0 GOOS=windows GOARCH=amd64 "$GO_CMD" build -trimpath -ldflags="-s -w" -o "$build_dir/collector-bridge-windows-amd64.exe" ./...
  )

  chmod 0755 "$build_dir"/collector-bridge-darwin-* "$build_dir"/collector-bridge-linux-amd64
  rm -f "$BRIDGE_ZIP"
  (
    cd "$build_dir"
    zip -q "$BRIDGE_ZIP" \
      collector-bridge-darwin-amd64 \
      collector-bridge-darwin-arm64 \
      collector-bridge-linux-amd64 \
      collector-bridge-windows-amd64.exe
  )

  rm -f "$SHA_FILE"
  append_checksum "$build_dir/collector-bridge-darwin-amd64" "solo-agency-local-collector/bin/collector-bridge-darwin-amd64"
  append_checksum "$build_dir/collector-bridge-darwin-arm64" "solo-agency-local-collector/bin/collector-bridge-darwin-arm64"
  append_checksum "$build_dir/collector-bridge-linux-amd64" "solo-agency-local-collector/bin/collector-bridge-linux-amd64"
  append_checksum "$build_dir/collector-bridge-windows-amd64.exe" "solo-agency-local-collector/bin/collector-bridge-windows-amd64.exe"
  append_checksum "$BRIDGE_ZIP" "solo-agency-collector/dist/$(basename "$BRIDGE_ZIP")"
}

rebuild_extension_archives() {
  log "Rebuilding Chrome extension archives"
  require_cmd zip
  mkdir -p "$DIST_DIR"
  [ -d "$EXTENSION_DIR" ] || die "Missing extension folder: $EXTENSION_DIR"

  local root_tmp nested_tmp
  root_tmp="$(mktemp -d "${TMPDIR:-/tmp}/soloagency-extension-root.XXXXXX")"
  nested_tmp="$(mktemp -d "${TMPDIR:-/tmp}/soloagency-extension-nested.XXXXXX")"
  trap 'rm -rf "$root_tmp" "$nested_tmp"' RETURN

  copy_extension_files "$root_tmp"
  rm -f "$EXTENSION_ROOT_ZIP"
  (
    cd "$root_tmp"
    zip -qr "$EXTENSION_ROOT_ZIP" .
  )

  mkdir -p "$nested_tmp/solo-agency-collector/chrome-extension"
  copy_extension_files "$nested_tmp/solo-agency-collector/chrome-extension"
  rm -f "$EXTENSION_NESTED_ZIP"
  (
    cd "$nested_tmp"
    zip -qr "$EXTENSION_NESTED_ZIP" solo-agency-collector
  )

  append_checksum "$EXTENSION_ROOT_ZIP" "solo-agency-collector/dist/$(basename "$EXTENSION_ROOT_ZIP")"
  append_checksum "$EXTENSION_NESTED_ZIP" "solo-agency-collector/dist/$(basename "$EXTENSION_NESTED_ZIP")"
}

copy_extension_files() {
  local target="$1"
  mkdir -p "$target"
  if have rsync; then
    rsync -a \
      --exclude ".DS_Store" \
      --exclude ".claude" \
      --exclude "backup" \
      "$EXTENSION_DIR"/ "$target"/
  else
    cp -R "$EXTENSION_DIR"/. "$target"/
    find "$target" -name ".DS_Store" -type f -delete
    rm -rf "$target/.claude" "$target/backup"
  fi
}

sync_and_zip_skills() {
  if [ -z "$SOURCE_SKILLS_DIR" ] && [ -d "$SCRIPT_DIR/../widecast/skills" ]; then
    SOURCE_SKILLS_DIR="$SCRIPT_DIR/../widecast/skills"
    echo "Auto-detected maintainer source skills:"
    echo "$SOURCE_SKILLS_DIR"
  fi

  if [ -n "$SOURCE_SKILLS_DIR" ]; then
    [ -d "$SOURCE_SKILLS_DIR" ] || die "Source skills folder does not exist: $SOURCE_SKILLS_DIR"
    log "Copying/updating source skills into playbooks without deleting local-only Solo Agency skills"
    mkdir -p "$TARGET_SKILLS_DIR"
    if have rsync; then
      rsync -a --exclude ".DS_Store" "$SOURCE_SKILLS_DIR"/ "$TARGET_SKILLS_DIR"/
    else
      cp -R "$SOURCE_SKILLS_DIR"/. "$TARGET_SKILLS_DIR"/
    fi
  else
    [ -d "$TARGET_SKILLS_DIR" ] || die "Missing writing skills. Expected bundled skills at $TARGET_SKILLS_DIR or set SOLO_AGENCY_SOURCE_SKILLS_DIR."
    echo "No explicit external source skills folder provided. Keeping bundled playbook skills."
    echo "To sync maintainer skills, run with SOLO_AGENCY_SOURCE_SKILLS_DIR=/absolute/path/to/skills."
    echo "$TARGET_SKILLS_DIR"
  fi

  log "Rebuilding skill archives from playbook skill folders"
  require_cmd zip
  for skill_dir in "$TARGET_SKILLS_DIR"/*; do
    if [ ! -d "$skill_dir" ] || [ ! -f "$skill_dir/SKILL.md" ]; then
      continue
    fi
    local skill_name
    skill_name="$(basename "$skill_dir")"
    rm -f "$TARGET_SKILLS_DIR/$skill_name.zip"
    (
      cd "$TARGET_SKILLS_DIR"
      zip -qr "$skill_name.zip" "$skill_name"
    )
    echo "Built $TARGET_SKILLS_DIR/$skill_name.zip"
  done
}

run_static_checks() {
  log "Running upload preflight checks"
  require_cmd unzip

  [ -s "$BRIDGE_ZIP" ] || die "Missing or empty bridge zip: $BRIDGE_ZIP"
  [ -s "$EXTENSION_ROOT_ZIP" ] || die "Missing or empty extension root zip: $EXTENSION_ROOT_ZIP"
  [ -s "$EXTENSION_NESTED_ZIP" ] || die "Missing or empty extension nested zip: $EXTENSION_NESTED_ZIP"
  [ -s "$SHA_FILE" ] || die "Missing or empty checksum file: $SHA_FILE"

  unzip -tq "$BRIDGE_ZIP" >/dev/null
  unzip -tq "$EXTENSION_ROOT_ZIP" >/dev/null
  unzip -tq "$EXTENSION_NESTED_ZIP" >/dev/null

  if unzip -l "$EXTENSION_NESTED_ZIP" | grep -q "media-agency-collector"; then
    die "Nested extension zip still contains media-agency-collector paths."
  fi

  if have node; then
    node --check "$EXTENSION_DIR/background.js" >/dev/null
    node --check "$EXTENSION_DIR/popup.js" >/dev/null
  else
    echo "Skipping JS parse check because node is not installed."
  fi

  bash -n "$TARGET_COLLECTOR_DIR/scripts/prepare_client_extension.sh"

  if have rg; then
    if rg -n "origin\\.widecast\\.ai/media-agency-collector|media_agency_playbook\\.md" "$SCRIPT_DIR" >/tmp/soloagency-rg-preflight.txt; then
      cat /tmp/soloagency-rg-preflight.txt >&2
      die "Found stale distribution references."
    fi
  fi

  if have git && is_solo_git_repo; then
    (
      cd "$SCRIPT_DIR"
      git diff --check
    )
  elif have git; then
    echo "Skipping git diff --check because Solo Agency is not initialized as an independent git repo yet."
  fi
}

cleanup_files() {
  log "Cleaning generated junk files"
  find "$SCRIPT_DIR" -name ".DS_Store" -type f -delete
}

generate_load_manifest() {
  # Full-load reference: line count + last non-empty line + sha256 for every playbook .md.
  # Agents compare a freshly-loaded file against this (see playbooks/LOAD_LEDGER_PROTOCOL.md);
  # a mismatch means the file was truncated/stale = NOT loaded. Regenerated every deploy so
  # adding a new playbook needs no manual step.
  local pb_dir="$SCRIPT_DIR/playbooks"
  local manifest="$pb_dir/LOAD_MANIFEST.md"
  [ -d "$pb_dir" ] || { log "No playbooks dir; skipping LOAD_MANIFEST"; return 0; }
  log "Generating playbooks/LOAD_MANIFEST.md (full-load ledger reference)"
  {
    echo "# LOAD_MANIFEST — full-load reference for Solo Agency playbooks"
    echo
    echo "Auto-generated by deploy-soloagency.sh. Do not edit by hand."
    echo "After loading any file below, its actual LINE COUNT must match its row here (see playbooks/LOAD_LEDGER_PROTOCOL.md). A shortfall = truncated = NOT loaded; re-read to EOF or re-fetch from GitHub. last_line + sha256 are OPTIONAL deeper checks, not required every load."
    echo
    echo '| path | lines | sha256 | last_line |'
    echo '|---|---|---|---|'
  } > "$manifest"
  while IFS= read -r f; do
    local rel="${f#$SCRIPT_DIR/}"
    case "$rel" in playbooks/LOAD_MANIFEST.md) continue ;; esac
    local lines sha last
    lines="$(awk 'END{print NR}' "$f")"
    sha="$(sha256_for_file "$f")"
    last="$(awk 'NF{l=$0} END{print l}' "$f")"
    last="${last//|/\\|}"
    printf '| %s | %s | %s | %s |\n' "$rel" "$lines" "$sha" "$last" >> "$manifest"
  done < <( { [ -f "$SCRIPT_DIR/SOLO_AGENCY_PLAYBOOK.md" ] && echo "$SCRIPT_DIR/SOLO_AGENCY_PLAYBOOK.md"; find "$pb_dir" -type f -name '*.md'; } | sort )
  log "LOAD_MANIFEST.md written: $manifest"
}

[ -d "$TARGET_COLLECTOR_DIR" ] || die "Missing canonical collector folder: $TARGET_COLLECTOR_DIR"
[ -d "$BRIDGE_DIR" ] || die "Missing bridge source folder: $BRIDGE_DIR"
[ -d "$EXTENSION_DIR" ] || die "Missing extension source folder: $EXTENSION_DIR"

echo "Using canonical Solo Agency collector folder:"
echo "$TARGET_COLLECTOR_DIR"

case "$MODE" in
  all)
    normalize_collector_text
    run_go_checks
    build_bridge_binaries
    rebuild_extension_archives
    sync_and_zip_skills
    generate_load_manifest
    cleanup_files
    run_static_checks
    git_deploy
    ;;
  --collector-only)
    normalize_collector_text
    run_go_checks
    build_bridge_binaries
    rebuild_extension_archives
    cleanup_files
    run_static_checks
    git_deploy
    ;;
  --skills-only)
    sync_and_zip_skills
    generate_load_manifest
    cleanup_files
    git_deploy
    ;;
  --check-only)
    run_static_checks
    ;;
  --git-only)
    git_deploy
    ;;
esac

echo
echo "Done."
echo "Collector package: $TARGET_COLLECTOR_DIR"
echo "Bridge zip:        $BRIDGE_ZIP"
echo "Extension zips:    $EXTENSION_ROOT_ZIP"
echo "                   $EXTENSION_NESTED_ZIP"
echo "Checksums:         $SHA_FILE"
echo "Playbook skills:   $TARGET_SKILLS_DIR"
echo "Load manifest:     $SCRIPT_DIR/playbooks/LOAD_MANIFEST.md"
