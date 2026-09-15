#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$repo_root"

for asset in assets/group_cover.png assets/agency-structure_light.png assets/theloop_light.png; do
  [[ -f "$asset" ]] || { echo "missing required setup-chat asset: $asset" >&2; exit 1; }
done

git check-ignore -q assets/group_cover.png && { echo "group_cover.png is ignored" >&2; exit 1; } || true
git check-ignore -q assets/theloop_light.png && { echo "theloop_light.png is ignored" >&2; exit 1; } || true

require() {
  local pattern=$1
  local file=$2
  rg -q --fixed-strings "$pattern" "$file" || {
    echo "missing contract text in $file: $pattern" >&2
    exit 1
  }
}

require 'assets/group_cover.png' playbooks/TEAM_MODEL.md
require 'before the rename-and-pin instruction' playbooks/TEAM_MODEL.md
require 'assets/agency-structure_light.png' playbooks/01_BASIC_PROFILE_PUBLIC_REPORT.md
require 'before the existing Login Reminder' playbooks/SETUP_FLOW_ENTRYPOINT.md
require 'assets/theloop_light.png' playbooks/SETUP_FLOW_ENTRYPOINT.md
require 'setup_chat_asset_missing' playbooks/SETUP_FLOW_ENTRYPOINT.md
require 'real absolute filesystem path' playbooks/SETUP_FLOW_ENTRYPOINT.md
require 'never change the visible 1-10 roadmap' playbooks/SETUP_FLOW_ENTRYPOINT.md

echo 'setup chat image acceptance passed'
