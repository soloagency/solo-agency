#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
Usage:
  prepare_client_extension.sh "Client Name" client_slug [extension_instance_id] [workspace_root]

Creates or refreshes:
  {workspace_root}/extensions/{client_slug}/

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
template_dir="$workspace_root/solo-agency-collector/chrome-extension"
target_dir="$workspace_root/extensions/$client_slug"

if [[ ! "$client_slug" =~ ^[a-z0-9][a-z0-9_-]*$ ]]; then
  echo "client_slug must use lowercase letters, numbers, dashes, or underscores, and start with a letter/number." >&2
  exit 2
fi

if [[ ! -d "$template_dir" ]]; then
  echo "Template extension folder not found: $template_dir" >&2
  exit 1
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

cat <<EOF
Prepared client extension:
  $target_dir

Chrome extension name:
  $client_name - Solo Agency Collector

Load this folder in the client's Chrome profile:
  $target_dir
EOF
