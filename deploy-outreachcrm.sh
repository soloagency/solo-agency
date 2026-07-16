#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_SKILLS_DIR="${OUTREACHCRM_SOURCE_SKILLS_DIR:-}"
TARGET_SKILLS_DIR="$SCRIPT_DIR/playbooks/skills"
MODE="all"
DEFAULT_GIT_REMOTE_URL="https://github.com/OWNER/outreachcrm.git"
AUTO_GIT_DEPLOY="${OUTREACHCRM_AUTO_GIT_DEPLOY:-1}"
GIT_PUSH="${OUTREACHCRM_GIT_PUSH:-1}"
GIT_REMOTE_NAME="${OUTREACHCRM_GIT_REMOTE_NAME:-origin}"
GIT_REMOTE_URL="${OUTREACHCRM_GIT_REMOTE_URL:-$DEFAULT_GIT_REMOTE_URL}"
GIT_DEFAULT_BRANCH="${OUTREACHCRM_GIT_BRANCH:-main}"
GIT_AUTHOR_NAME="${OUTREACHCRM_GIT_USER_NAME:-OutreachCRM}"
GIT_AUTHOR_EMAIL="${OUTREACHCRM_GIT_USER_EMAIL:-outreachcrm-deploy@users.noreply.github.com}"
DEEPSEEK_MODEL="${OUTREACHCRM_DEEPSEEK_MODEL:-deepseek-chat}"
DEEPSEEK_API_URL="${OUTREACHCRM_DEEPSEEK_API_URL:-https://api.deepseek.com/chat/completions}"
DEEPSEEK_API_KEY_FALLBACK=""  # removed: no embedded fallback secret; set DEEPSEEK_API_KEY env var or the AI commit-message step is skipped
GIT_COMMIT_PREFIX="${OUTREACHCRM_COMMIT_PREFIX:-deploy}"
GIT_DRY_RUN="${OUTREACHCRM_GIT_DRY_RUN:-0}"

usage() {
  cat >&2 <<'USAGE'
Usage:
  ./deploy-outreachcrm.sh [all|--skills-only|--check-only|--git-only] [--no-git] [--no-push] [--dry-run-git]

Default/all:
  - rebuild playbook skill zips
  - generate playbooks/LOAD_MANIFEST.md (full-load ledger reference)
  - clean generated junk files
  - run upload preflight checks
  - secret-scan the staged diff
  - commit and push OutreachCRM repo updates

Environment:
  OUTREACHCRM_SOURCE_SKILLS_DIR=/absolute/path/to/skills
  OUTREACHCRM_AUTO_GIT_DEPLOY=1
  OUTREACHCRM_GIT_PUSH=1
  OUTREACHCRM_GIT_REMOTE_NAME=origin
  OUTREACHCRM_GIT_REMOTE_URL=https://github.com/OWNER/outreachcrm.git
  OUTREACHCRM_GIT_BRANCH=main
  OUTREACHCRM_GIT_USER_NAME="OutreachCRM"
  OUTREACHCRM_GIT_USER_EMAIL="outreachcrm-deploy@users.noreply.github.com"
  OUTREACHCRM_DEEPSEEK_MODEL=deepseek-chat
  OUTREACHCRM_DEEPSEEK_API_URL=https://api.deepseek.com/chat/completions
  DEEPSEEK_API_KEY=sk-...
  deepseek_api_key=sk-...
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    all|--skills-only|--check-only|--git-only)
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

outreachcrm_git_root() {
  git_cmd rev-parse --show-toplevel 2>/dev/null || true
}

is_outreachcrm_git_repo() {
  local root
  root="$(outreachcrm_git_root)"
  [ "$root" = "$SCRIPT_DIR" ] && [ -d "$SCRIPT_DIR/.git" ]
}

ensure_outreachcrm_git_repo() {
  require_cmd git

  if is_outreachcrm_git_repo; then
    return
  fi

  local detected_root
  detected_root="$(outreachcrm_git_root)"

  if [ -z "$GIT_REMOTE_URL" ]; then
    cat >&2 <<EOF
ERROR: $SCRIPT_DIR is not an independent git repo yet.

Detected git root: ${detected_root:-none}

OutreachCRM must be committed into its own repo, not any parent workspace it may
be nested inside. Run once with the OutreachCRM GitHub remote URL, for example:

  OUTREACHCRM_GIT_REMOTE_URL=https://github.com/OWNER/outreachcrm.git bash deploy-outreachcrm.sh --git-only

By default deploy uses:

  $DEFAULT_GIT_REMOTE_URL

Use OUTREACHCRM_GIT_REMOTE_URL only when you need a different remote that belongs
to the OutreachCRM GitHub account. After .git is initialized inside outreachcrm/,
future deploys can just run:

  bash deploy-outreachcrm.sh
EOF
    exit 1
  fi

  log "Initializing independent OutreachCRM git repo"
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
    [ -n "$GIT_REMOTE_URL" ] || die "Missing git remote '$GIT_REMOTE_NAME'. Set OUTREACHCRM_GIT_REMOTE_URL once."
    git_cmd remote add "$GIT_REMOTE_NAME" "$GIT_REMOTE_URL"
    return
  fi
  if [ -n "$GIT_REMOTE_URL" ] && [ "$existing_url" != "$GIT_REMOTE_URL" ]; then
    die "Remote '$GIT_REMOTE_NAME' already points to '$existing_url', not OUTREACHCRM_GIT_REMOTE_URL='$GIT_REMOTE_URL'. Fix manually or choose another OUTREACHCRM_GIT_REMOTE_NAME."
  fi
}

current_git_branch() {
  git_cmd rev-parse --abbrev-ref HEAD
}

scan_staged_secrets() {
  log "Checking staged diff for obvious secrets"
  local secret_hits
  secret_hits="$(git_cmd diff --cached --text --unified=0 | grep -E 'wc_live_[A-Za-z0-9_-]{20,}|sk-ant-[A-Za-z0-9_-]{20,}|sk-[A-Za-z0-9_-]{20,}|api_key_local\"[[:space:]]*:[[:space:]]*\"[^\"]+|\"refresh_token\"[[:space:]]*:|\"client_secret\"[[:space:]]*:|TRACKER_API_KEY' || true)"
  if [ -n "$secret_hits" ]; then
    echo "$secret_hits" >&2
    die "Refusing to commit because staged diff appears to contain a secret. Move secrets into env/local ignored config."
  fi

  local secret_paths
  secret_paths="$(git_cmd diff --cached --name-only | grep -E '(^|/)(token\.json|client_secret[^/]*\.json|google_oauth_client\.json)$' || true)"
  if [ -n "$secret_paths" ]; then
    echo "$secret_paths" >&2
    die "Refusing to commit because staged files include an OAuth/token secret file (token.json / client_secret*.json / google_oauth_client.json). These must stay gitignored."
  fi
}

fallback_commit_message() {
  local msg_file="$1"
  local changed_files="$2"
  {
    printf '%s: update OutreachCRM deployment assets\n\n' "$GIT_COMMIT_PREFIX"
    printf 'Generated by OutreachCRM maintainer deploy.\n\n'
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
  request_file="$(mktemp "${TMPDIR:-/tmp}/outreachcrm-deepseek-request.XXXXXX")"

  SUMMARY_FILE="$summary_file" DEEPSEEK_MODEL="$DEEPSEEK_MODEL" python3 - "$request_file" <<'PY'
import json
import os
import pathlib
import sys

summary = pathlib.Path(os.environ["SUMMARY_FILE"]).read_text(encoding="utf-8")
prompt = f"""Create a concise Git commit message for OutreachCRM.

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
    echo "Skipping git deploy because OUTREACHCRM_AUTO_GIT_DEPLOY=$AUTO_GIT_DEPLOY."
    return
  }

  if [ "$MODE" = "--check-only" ]; then
    echo "Skipping git deploy for --check-only."
    return
  fi

  log "Preparing OutreachCRM git commit"
  ensure_outreachcrm_git_repo
  ensure_git_identity
  ensure_git_remote

  if [ "$(outreachcrm_git_root)" != "$SCRIPT_DIR" ]; then
    die "Refusing git deploy because git root is not the OutreachCRM folder."
  fi

  git_cmd add -A -- .

  if git_cmd diff --cached --quiet; then
    echo "No OutreachCRM changes to commit."
    return
  fi

  scan_staged_secrets

  local summary_file msg_file response_file
  summary_file="$(mktemp "${TMPDIR:-/tmp}/outreachcrm-commit-summary.XXXXXX")"
  msg_file="$(mktemp "${TMPDIR:-/tmp}/outreachcrm-commit-message.XXXXXX")"
  response_file="$(mktemp "${TMPDIR:-/tmp}/outreachcrm-deepseek-response.XXXXXX")"

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
    log "Pushing OutreachCRM repo to $GIT_REMOTE_NAME/$branch"
    git_cmd push -u "$GIT_REMOTE_NAME" "$branch"
  else
    echo "Skipping push because OUTREACHCRM_GIT_PUSH=$GIT_PUSH."
  fi
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

sync_and_zip_skills() {
  if [ -n "$SOURCE_SKILLS_DIR" ]; then
    [ -d "$SOURCE_SKILLS_DIR" ] || die "Source skills folder does not exist: $SOURCE_SKILLS_DIR"
    log "Copying/updating source skills into playbooks without deleting local-only OutreachCRM skills"
    mkdir -p "$TARGET_SKILLS_DIR"
    if have rsync; then
      rsync -a --exclude ".DS_Store" "$SOURCE_SKILLS_DIR"/ "$TARGET_SKILLS_DIR"/
    else
      cp -R "$SOURCE_SKILLS_DIR"/. "$TARGET_SKILLS_DIR"/
    fi
  else
    [ -d "$TARGET_SKILLS_DIR" ] || die "Missing playbook skills. Expected bundled skills at $TARGET_SKILLS_DIR or set OUTREACHCRM_SOURCE_SKILLS_DIR."
    echo "No explicit external source skills folder provided. Keeping bundled playbook skills."
    echo "To sync maintainer skills, run with OUTREACHCRM_SOURCE_SKILLS_DIR=/absolute/path/to/skills."
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
  log "Running deploy preflight checks"

  # Self syntax check: the deploy script must parse cleanly before we commit.
  bash -n "$SCRIPT_DIR/deploy-outreachcrm.sh"

  if have git && is_outreachcrm_git_repo; then
    (
      cd "$SCRIPT_DIR"
      git diff --check
    )
  elif have git; then
    echo "Skipping git diff --check because OutreachCRM is not initialized as an independent git repo yet."
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
    echo "# LOAD_MANIFEST — full-load reference for OutreachCRM playbooks"
    echo
    echo "Auto-generated by deploy-outreachcrm.sh. Do not edit by hand."
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
  done < <( { [ -f "$SCRIPT_DIR/OUTREACHCRM_PLAYBOOK.md" ] && echo "$SCRIPT_DIR/OUTREACHCRM_PLAYBOOK.md"; find "$pb_dir" -type f -name '*.md'; } | sort )
  log "LOAD_MANIFEST.md written: $manifest"
}

echo "Deploying OutreachCRM toolkit from:"
echo "$SCRIPT_DIR"

case "$MODE" in
  all)
    sync_and_zip_skills
    generate_load_manifest
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
echo "Playbook skills:   $TARGET_SKILLS_DIR"
echo "Load manifest:     $SCRIPT_DIR/playbooks/LOAD_MANIFEST.md"
