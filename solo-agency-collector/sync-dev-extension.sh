#!/usr/bin/env bash
# Ship the repo's extension code to the aven-ngo DEV extension and bump the version.
#
# The repo's chrome-extension/ folder is never loaded into Chrome — the operator loads the
# per-client folder. So a repo edit is invisible to a live run until it is copied across, and
# the failure is silent: the bridge dispatches the capability name, the stale extension has no
# such function, and the data point comes back `records: null`, which is indistinguishable from
# "the capability crashed". A whole five-profile run once collected nothing that way.
#
# The version bump is not bookkeeping. Chrome shows the version on the extensions page, and it
# is the only thing the operator can look at to answer "did my reload pick up the new code?".
# Syncing without bumping already caused exactly that confusion: the code was correct and
# current, the number said 0.2.2 for three different builds, and there was no way to tell.
# So: every sync bumps. No sync without a new number.
#
# Copies CODE only. manifest name/description/title, popup.html and client_binding.json carry
# the client's own branding ("Aven Ngo - Solo Agency Collector") and are never overwritten.
#
# Usage:  ./sync-dev-extension.sh            # bump patch, copy, verify
#         ./sync-dev-extension.sh --check    # verify only, change nothing
set -euo pipefail

SRC="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/chrome-extension"
DEST="${SOLO_DEV_EXTENSION:-/Users/binhnguyen/Downloads/oneman_agency/extensions/aven-ngo}"

[ -d "$SRC" ]  || { echo "no source extension at $SRC" >&2; exit 1; }
[ -d "$DEST" ] || { echo "no dev extension at $DEST (set SOLO_DEV_EXTENSION)" >&2; exit 1; }

# Never these: they are the client's identity, not code.
KEEP=(manifest.json popup.html client_binding.json client_binding.example.json)

check_only=0
[ "${1:-}" = "--check" ] && check_only=1

drift=0
for f in "$SRC"/*; do
  b="$(basename "$f")"
  [ -f "$f" ] || continue
  case " ${KEEP[*]} " in *" $b "*) continue ;; esac
  case "$b" in *_20[0-9][0-9]-*) continue ;; esac   # dated backups, not source
  if [ ! -f "$DEST/$b" ] || ! cmp -s "$f" "$DEST/$b"; then
    drift=1
    if [ "$check_only" = 1 ]; then echo "  differs: $b"; else
      cp "$f" "$DEST/$b"; echo "  copied : $b"
    fi
  fi
done

# manifest.json is never copied (it carries the client's branding), but its CAPABILITY parts —
# permissions / host_permissions / content_scripts — are code, and a drift there fails silently
# (2026-08-16: the "offscreen" permission for the operator chime; without it the chime never
# plays and the only trace is human_gate.alert.ok:false). Warn; patch by hand or full rebuild.
python3 - "$SRC/manifest.json" "$DEST/manifest.json" <<'PY' || true
import json, sys
a, b = (json.load(open(p)) for p in sys.argv[1:3])
for key in ("permissions", "host_permissions", "content_scripts"):
    if a.get(key) != b.get(key):
        print("  WARNING manifest.%s differs (repo vs dev) — not synced by design; patch the dev manifest by hand:" % key)
        print("      repo:", json.dumps(a.get(key), ensure_ascii=False))
        print("      dev :", json.dumps(b.get(key), ensure_ascii=False))
PY

repo_ver="$(python3 -c "import json;print(json.load(open('$SRC/manifest.json'))['version'])")"
dev_ver="$(python3 -c "import json;print(json.load(open('$DEST/manifest.json'))['version'])")"

if [ "$check_only" = 1 ]; then
  echo "repo v$repo_ver | dev v$dev_ver"
  [ "$drift" = 0 ] && echo "in sync" || echo "OUT OF SYNC — run without --check"
  exit 0
fi

# Bump the patch on BOTH so the number the operator reads in Chrome is new every single time,
# even for a one-line change. A version that repeats is worse than no version: it actively
# tells the operator nothing changed when something did.
new_ver="$(python3 - "$repo_ver" <<'PY'
import sys
p = sys.argv[1].split(".")
while len(p) < 3: p.append("0")
p[2] = str(int(p[2]) + 1)
print(".".join(p))
PY
)"
for m in "$SRC/manifest.json" "$DEST/manifest.json"; do
  python3 - "$m" "$new_ver" <<'PY'
import json, sys
p, v = sys.argv[1], sys.argv[2]
d = json.load(open(p))
d["version"] = v
open(p, "w").write(json.dumps(d, ensure_ascii=False, indent=2) + "\n")
PY
done

node --check "$DEST/gql_extract.js"
node --check "$DEST/gql_actions.js" 2>/dev/null || true
echo
echo "synced -> $DEST"
echo "version  v$repo_ver -> v$new_ver   (reload the extension; Chrome must show v$new_ver)"
