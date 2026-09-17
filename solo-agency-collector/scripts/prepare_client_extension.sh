#!/usr/bin/env bash
# prepare_client_extension.sh — thin wrapper around the bridge's `tool extension prepare`.
#
# The work (copy the template, patch manifest.json, write client_binding.json, rename the popup)
# is done by the collector bridge binary, in Go, so a normal user's machine needs nothing else:
# no Python, no Node. This script only keeps the old command shape working:
#
#   prepare_client_extension.sh "Client Name" client_slug [extension_instance_id] [workspace_root]
#
# On Windows call the binary directly:
#   collector-bridge-windows-amd64.exe tool extension prepare --client-name "Client Name" --client-slug SLUG --root AGENCY_ROOT
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
Usage:
  prepare_client_extension.sh "Client Name" client_slug [extension_instance_id] [workspace_root]

Creates or refreshes the client's own copy of the extension:
  {workspace_root}/extensions/{client_slug}_extension/

Backward compatibility: if an older {workspace_root}/extensions/{client_slug}/ folder already
exists and the newer {client_slug}_extension/ name does not, that older folder is reused in
place (refreshing its code and branding) instead of creating a second copy. Never both.

The extension display name always starts with the client name, for example:
  AvenNgo - Solo Agency Collector

This is a wrapper: the bridge binary does the work (`<bridge> tool extension prepare ...`).
The bridge is found via $SOLO_AGENCY_BRIDGE, else under {workspace_root}/solo-agency-local-collector/bin/,
else via tools/solo_tool's search from the current directory upwards.
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -lt 2 || $# -gt 4 ]]; then
  usage
  exit 2
fi

client_name="$1"
client_slug="$2"
extension_instance_id="${3:-${client_slug}-local-collector}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
workspace_root="${4:-$repo_root}"
template_dir="${SOLO_AGENCY_EXTENSION_TEMPLATE_DIR:-$repo_root/solo-agency-collector/chrome-extension}"

if [[ ! -d "$template_dir" ]]; then
  echo "Template extension folder not found: $template_dir" >&2
  exit 1
fi

# Find the bridge binary: explicit path, then the install next to the workspace root, then the
# generic search tools/solo_tool does (from the current directory upwards, then PATH).
bridge="${SOLO_AGENCY_BRIDGE:-}"
if [[ -z "$bridge" ]]; then
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"; arch="$(uname -m)"
  case "$arch" in x86_64|amd64) arch=amd64 ;; arm64|aarch64) arch=arm64 ;; esac
  for cand in "$workspace_root/solo-agency-local-collector/bin/collector-bridge-$os-$arch" \
              "$workspace_root/solo-agency-local-collector/bin/collector-bridge-$os-amd64" \
              "$repo_root/../solo-agency-local-collector/bin/collector-bridge-$os-$arch" \
              "$repo_root/../solo-agency-local-collector/bin/collector-bridge-$os-amd64"; do
    if [[ -x "$cand" ]]; then bridge="$cand"; break; fi
  done
fi

if [[ -n "$bridge" && -x "$bridge" ]]; then
  exec "$bridge" tool extension prepare \
    --client-name "$client_name" --client-slug "$client_slug" \
    --instance-id "$extension_instance_id" --root "$workspace_root" --template "$template_dir"
fi

if [[ -x "$repo_root/tools/solo_tool" ]]; then
  exec "$repo_root/tools/solo_tool" extension prepare \
    --client-name "$client_name" --client-slug "$client_slug" \
    --instance-id "$extension_instance_id" --root "$workspace_root" --template "$template_dir"
fi

echo "prepare_client_extension.sh: no bridge binary found. Run solo-agency-collector/setup_collector.sh first (it installs the bridge), or set SOLO_AGENCY_BRIDGE=/path/to/collector-bridge-<os>-<arch>." >&2
exit 2
