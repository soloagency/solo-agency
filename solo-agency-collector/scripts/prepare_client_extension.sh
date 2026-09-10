#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
Usage:
  prepare_client_extension.sh "Client Name" client_slug [extension_instance_id] [workspace_root]

Creates or refreshes the client's own copy of the extension:
  {workspace_root}/extensions/{client_slug}_extension/

Backward compatibility: if an older {workspace_root}/extensions/{client_slug}/ folder already
exists and the newer {client_slug}_extension/ name does not, this script reuses that older
folder in place (refreshing its code and branding) instead of creating a second copy. It never
creates both.

The extension display name always starts with the client name, for example:
  AvenNgo - Solo Agency Collector
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
workspace_root="${4:-$(cd "$script_dir/../.." && pwd)}"
template_dir="${SOLO_AGENCY_EXTENSION_TEMPLATE_DIR:-$workspace_root/solo-agency-collector/chrome-extension}"

if [[ ! "$client_slug" =~ ^[a-z0-9][a-z0-9_-]*$ ]]; then
  echo "client_slug must use lowercase letters, numbers, dashes, or underscores, and start with a letter/number." >&2
  exit 2
fi

if [[ ! -d "$template_dir" ]]; then
  echo "Template extension folder not found: $template_dir" >&2
  exit 1
fi

# Owner decision 2026-09-10: extensions/{client_slug}/ (no word "extension" in it) sitting next
# to the source folder solo-agency-collector/chrome-extension/ (which looks just as legitimate)
# was the single most confusing thing for a low-tech operator picking "Load unpacked" -- people
# loaded the wrong one. The new client folder name says what it is on its own. An older install
# that already has extensions/{client_slug}/ and has never been renamed keeps working from that
# same folder (refreshed in place); this script never creates both a legacy and a new folder for
# the same client.
new_dir_name="${client_slug}_extension"
new_target_dir="$workspace_root/extensions/$new_dir_name"
legacy_target_dir="$workspace_root/extensions/$client_slug"

using_legacy_dir=0
if [[ -d "$new_target_dir" ]]; then
  target_dir="$new_target_dir"
elif [[ -d "$legacy_target_dir" ]]; then
  target_dir="$legacy_target_dir"
  using_legacy_dir=1
else
  target_dir="$new_target_dir"
fi

if [[ "$using_legacy_dir" == "1" ]]; then
  cat <<EOF
Notice: found an older extension folder for this client and reused it instead of creating a
second copy:
  $target_dir
(The newer naming would have been $new_target_dir -- rename the folder yourself later if you
want to switch to it; this script will never create both.)
EOF
fi

mkdir -p "$target_dir"
cp -R "$template_dir"/. "$target_dir"/
rm -f "$target_dir/client_binding.example.json"

python3 - "$target_dir" "$client_name" "$client_slug" "$extension_instance_id" <<'PY'
import json
import pathlib
import sys

target_dir = pathlib.Path(sys.argv[1])
client_name = sys.argv[2].strip()
client_slug = sys.argv[3].strip()
extension_instance_id = sys.argv[4].strip()
display_name = f"{client_name} - Solo Agency Collector"

manifest_path = target_dir / "manifest.json"
manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
manifest["name"] = display_name
manifest["description"] = (
    f"Collects visible, authorized private data source signals locally for {client_name}."
)
manifest.setdefault("action", {})
manifest["action"]["default_title"] = display_name
manifest_path.write_text(json.dumps(manifest, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")

binding = {
    "client_name": client_name,
    "client_slug": client_slug,
    "extension_instance_id": extension_instance_id,
    "extension_display_name": display_name,
    "bridge_base_url": "http://127.0.0.1:17321",
    "display_name": display_name,
    "routing_mode": "shared_bridge_per_client_extension",
}
(target_dir / "client_binding.json").write_text(
    json.dumps(binding, indent=2, ensure_ascii=False) + "\n",
    encoding="utf-8",
)

popup_path = target_dir / "popup.html"
popup = popup_path.read_text(encoding="utf-8")
popup = popup.replace("<title>Solo Agency Local Collector</title>", f"<title>{display_name}</title>")
popup = popup.replace("<h1>Solo Agency Local Collector</h1>", f"<h1>{display_name}</h1>")
popup_path.write_text(popup, encoding="utf-8")
PY

# Finder shows this the instant the folder is opened -- no need to open manifest.json or guess
# which of several lookalike folders is the right one.
generated_at="$(date -u +"%Y-%m-%d %H:%M:%S UTC")"
cat > "$target_dir/THIS_IS_THE_CLIENT_COPY.txt" <<EOF
This is the client copy for: $client_name ($client_slug) -- generated $generated_at
EOF

cat <<EOF

Prepared client extension:
  $target_dir

Chrome extension name:
  $client_name - Solo Agency Collector

=====================================================================
Load THIS folder in Chrome/Edge (not solo-agency-collector/chrome-extension):
  $target_dir
=====================================================================
EOF
