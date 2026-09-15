#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

require_text() {
  local file="$1"
  local text="$2"
  if ! grep -Fq "$text" "$repo_root/$file"; then
    echo "missing required contract in $file: $text" >&2
    exit 1
  fi
}

reject_text() {
  local file="$1"
  local text="$2"
  if grep -Fq "$text" "$repo_root/$file"; then
    echo "obsolete contract remains in $file: $text" >&2
    exit 1
  fi
}

require_text "playbooks/CONTENT_SIGNAL_RESEARCH.md" "Hard-cap the entire task at 120 meaningful comments"
require_text "playbooks/CONTENT_SIGNAL_RESEARCH.md" "Luna"
require_text "playbooks/CONTENT_SIGNAL_RESEARCH.md" "write a script, blog, caption, CTA, CRM contact, lead record or outreach message."
require_text "playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md" "Content Evidence Bank"
require_text "playbooks/FEATURE_CATALOG.md" 'research_content_signals'
require_text "playbooks/NEXT_JOB_CATALOGUE.md" 'research_content_signals'
require_text "playbooks/04_DAILY_SCHEDULE.md" "never harvest comment threads"
require_text "playbooks/04_DAILY_SCHEDULE.md" "never write full video/blog/social drafts"
require_text "playbooks/06_AGENCY_REPORT_STANDARD.md" "Content Decision Packet"
require_text "playbooks/03_PRODUCTION_DISTRIBUTION.md" "one complete, strongest draft"
require_text "playbooks/03_PRODUCTION_DISTRIBUTION.md" "Client Removal Test"

reject_text "playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md" "The automation report must include at least one draft script/blog/caption"
reject_text "playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md" "First draft: {script/blog/caption title}"
reject_text "playbooks/03_PRODUCTION_DISTRIBUTION.md" "Default report output is five complete short-form video script draft versions"
reject_text "playbooks/06_AGENCY_REPORT_STANDARD.md" "then the five script versions"

echo "content signal research acceptance: PASS"
