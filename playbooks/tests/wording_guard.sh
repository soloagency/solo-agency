#!/usr/bin/env bash
# playbooks/tests/wording_guard.sh — regression guard (S10) against the retired STANDALONE
# duration promise, retired hour-scale spreading language, and the retired "public only" /
# public-only-filter group wording, coming back into the tracked playbooks / root docs.
#
# Scans: AGENTS.md, SOLO_AGENCY_PLAYBOOK.md, README.md, playbooks/**/*.md, docs/*.md,
# solo-agency-collector/AGENT_RUNBOOK.md, solo-agency-collector/README.md.
# Excludes: timestamped backup copies (*_20??-??-??_??-??-??.*) and playbooks/tests/ (this
# script's own directory, so the guard never matches its own file).
#
# Exit 1 and print every offending line if any of these fixed strings appears:
#   khoảng 10–15 phút | khoảng 10-15 phút | about 10–15 minutes | about 10-15 minutes |
#   10-15 minutes for a first run | 10–15 minutes for a first run |
#   ≥ 4 hours | spread over ≥ | spread over hours | spread over 4 | window of hours |
#   across the run window | trải trong ≥ 4 giờ |
#   public only) | privacy == "public"` only | Keep only results whose `privacy == "public"`
# Exit 0 (silent) when none are found.
set -u

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root" || exit 1

is_backup() {
  case "$(basename "$1")" in
    *_20[2-9][0-9]-[0-9][0-9]-[0-9][0-9]_[0-9][0-9]-[0-9][0-9]-[0-9][0-9].*) return 0 ;;
    *) return 1 ;;
  esac
}

files=()
for f in AGENTS.md SOLO_AGENCY_PLAYBOOK.md README.md \
         solo-agency-collector/AGENT_RUNBOOK.md solo-agency-collector/README.md; do
  [ -f "$f" ] && files+=("$f")
done

while IFS= read -r -d '' f; do
  files+=("$f")
done < <(find playbooks -name '*.md' -not -path 'playbooks/tests/*' -print0 2>/dev/null)

while IFS= read -r -d '' f; do
  files+=("$f")
done < <(find docs -maxdepth 1 -name '*.md' -print0 2>/dev/null)

scanned=()
for f in "${files[@]}"; do
  is_backup "$f" || scanned+=("$f")
done

patterns=(
  'khoảng 10–15 phút'
  'khoảng 10-15 phút'
  'about 10–15 minutes'
  'about 10-15 minutes'
  '10-15 minutes for a first run'
  '10–15 minutes for a first run'
  '≥ 4 hours'
  'spread over ≥'
  'spread over hours'
  'spread over 4'
  'window of hours'
  'across the run window'
  'trải trong ≥ 4 giờ'
  'public only)'
  'privacy == "public"` only'
  'Keep only results whose `privacy == "public"`'
)

hits=0
if [ "${#scanned[@]}" -gt 0 ]; then
  for pat in "${patterns[@]}"; do
    out="$(grep -n -H -F -- "$pat" "${scanned[@]}" 2>/dev/null)"
    if [ -n "$out" ]; then
      echo "$out"
      hits=$((hits + 1))
    fi
  done
fi

if [ "$hits" -gt 0 ]; then
  exit 1
fi
exit 0
