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
# Copies CODE only (and merges the CAPABILITY parts of manifest.json). manifest name/description/title, popup.html and client_binding.json carry
# the client's own branding ("Aven Ngo - Solo Agency Collector") and are never overwritten.
#
# Usage:  ./sync-dev-extension.sh            # bump patch, copy, verify
#         ./sync-dev-extension.sh --check    # verify only, change nothing
#         ./sync-dev-extension.sh --prune    # also remove dev files the repo no longer has
set -euo pipefail

SRC="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/chrome-extension"
DEST="${SOLO_DEV_EXTENSION:-$(cat "$HOME/.config/solo-agency/dev_extension_path" 2>/dev/null || echo /path/to/dev-install/extensions/aven-ngo)}"

[ -d "$SRC" ]  || { echo "no source extension at $SRC" >&2; exit 1; }
[ -d "$DEST" ] || { echo "no dev extension at $DEST (set SOLO_DEV_EXTENSION)" >&2; exit 1; }

# Never these: they are the client's identity, not code.
KEEP=(manifest.json popup.html client_binding.json client_binding.example.json)

check_only=0
for arg in "$@"; do [ "$arg" = "--check" ] && check_only=1; done

# Subdirectories are copied too (platforms/<name>/, core/). The old loop globbed only the top
# level and silently skipped every directory — icons/ proved it — which would have dropped a
# whole platform module on the floor and produced the same `records: null` silence described
# above. Two directories are deliberately NOT code and stay out: icons/ (client branding) and
# .claude/ (local tooling); backup/ in the dev folder is the operator's own. Dated backups and
# *.bak are never source.
SKIP_DIRS=(icons .claude backup)
prune=0
for arg in "$@"; do [ "$arg" = "--prune" ] && prune=1; done

drift=0
while IFS= read -r -d '' f; do
  rel="${f#"$SRC"/}"
  b="$(basename "$f")"
  top="${rel%%/*}"
  case " ${KEEP[*]} " in *" $rel "*) continue ;; esac
  if [ "$top" != "$rel" ]; then case " ${SKIP_DIRS[*]} " in *" $top "*) continue ;; esac; fi
  case "$b" in *_20[0-9][0-9]-*|*.bak) continue ;; esac   # dated backups, not source
  if [ ! -f "$DEST/$rel" ] || ! cmp -s "$f" "$DEST/$rel"; then
    drift=1
    if [ "$check_only" = 1 ]; then echo "  differs: $rel"; else
      mkdir -p "$DEST/$(dirname "$rel")"
      cp "$f" "$DEST/$rel"; echo "  copied : $rel"
    fi
  fi
done < <(find "$SRC" -type f -print0 | sort -z)

# manifest.json is never copied (it carries the client's branding), but its CAPABILITY parts —
# permissions / host_permissions / content_scripts — are code, and a drift there fails silently
# (2026-08-16: the "offscreen" permission for the operator chime; without it the chime never
# plays and the only trace is human_gate.alert.ok:false). Until 2026-09-09 this only warned and
# left the patch to the operator; the platform-module move (content_scripts now names
# platforms/facebook/gql_intercept.js) turned that into a trap — an unpatched client keeps
# loading the old path, and --prune would then delete exactly that file. So the capability parts
# are MERGED into the dev manifest: branding (name, description, icons) and version untouched,
# the previous dev manifest kept as a dated copy next to it. --check only reports the drift.
python3 - "$SRC/manifest.json" "$DEST/manifest.json" "$check_only" <<'PY' || true
import json, sys, io, datetime, shutil
src, dest, check_only = sys.argv[1], sys.argv[2], sys.argv[3] == "1"
a, b = (json.load(open(p)) for p in (src, dest))
drift = [k for k in ("permissions", "host_permissions", "content_scripts") if a.get(k) != b.get(k)]
for key in drift:
    print(("  manifest.%s differs (repo vs dev) — merged on sync:" if check_only else "  merged : manifest.%s") % key)
    print("      repo:", json.dumps(a.get(key), ensure_ascii=False))
    print("      dev :", json.dumps(b.get(key), ensure_ascii=False))
if drift and not check_only:
    stamp = datetime.datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
    shutil.copy(dest, dest[:-5] + "_" + stamp + ".json")
    for key in drift:
        b[key] = a.get(key)
    io.open(dest, "w", encoding="utf-8").write(json.dumps(b, ensure_ascii=False, indent=2) + "\n")

# The dev manifest MUST NOT be the repo's. Its name is the client's, and that is how the operator
# tells two loaded extensions apart in chrome://extensions — where they are otherwise identical
# rows. It was silently lost once already: a full rebuild from the repo overwrote the aven-ngo
# manifest and both extensions then read "Solo Agency Local Collector", visible only by noticing
# the branding was gone. This check makes that loud, since the merge above never touches the name.
brand = [k for k in ("name", "description") if a.get(k) == b.get(k)]
if a.get("name") == b.get("name"):
    print("  WARNING the dev manifest carries the REPO name %r — the client branding was overwritten," % b.get("name"))
    print("      probably by a full rebuild. Restore it from a manifest_<date>.json backup in the dev folder;")
    print("      chrome://extensions shows two identical rows until you do.")
elif "description" in brand:
    print("  note: dev manifest description matches the repo's — client branding may be partly overwritten")
PY

# Files the dev manifest declares (content scripts, the service worker) must never be pruned:
# Chrome would keep looking for them and the Facebook interceptor would simply stop.
manifest_references() {
  python3 - "$DEST/manifest.json" "$1" <<'PY'
import json, sys
try:
    m = json.load(open(sys.argv[1]))
except Exception:
    sys.exit(1)
refs = set()
for cs in m.get("content_scripts", []) or []:
    for js in (cs.get("js", []) or []) + (cs.get("css", []) or []): refs.add(js)
sw = (m.get("background") or {}).get("service_worker")
if sw: refs.add(sw)
sys.exit(0 if sys.argv[2] in refs else 1)
PY
}

# A file that left the repo (moved into platforms/<name>/, or deleted) but still sits in the
# dev folder is stale code Chrome would happily keep loading. Report it always; remove it only
# when asked (--prune), because deleting from the operator's extension folder is not a sync.
while IFS= read -r -d '' f; do
  rel="${f#"$DEST"/}"
  b="$(basename "$f")"
  top="${rel%%/*}"
  case " ${KEEP[*]} " in *" $rel "*) continue ;; esac
  if [ "$top" != "$rel" ]; then case " ${SKIP_DIRS[*]} " in *" $top "*) continue ;; esac; fi
  case "$b" in *_20[0-9][0-9]-*|*.bak|client_binding.json) continue ;; esac
  if [ ! -f "$SRC/$rel" ]; then
    if manifest_references "$rel"; then
      echo "  kept   : $rel (not in repo, but $DEST/manifest.json still declares it)"
    elif [ "$prune" = 1 ] && [ "$check_only" = 0 ]; then rm -f "$f"; echo "  pruned : $rel"; else
      echo "  stale  : $rel (not in repo; re-run with --prune to remove)"
    fi
  fi
done < <(find "$DEST" -type f -print0 | sort -z)

# popup.html is the same story as the manifest: it is kept out of the copy list because its
# <title> and <h1> carry the client's name, but everything else in it is CODE. That was harmless
# while the popup was only settings inputs; the moment it grew a control — the Stop run button —
# a file "kept for branding" started silently withholding a feature from every client. So sync the
# markup and put the client's two branded strings back, exactly as the manifest merge does.
python3 - "$SRC/popup.html" "$DEST/popup.html" <<'PY_POPUP' || true
import re, sys, io
src, dest = sys.argv[1], sys.argv[2]
repo = io.open(src, encoding="utf-8").read()
try:
    dev = io.open(dest, encoding="utf-8").read()
except FileNotFoundError:
    sys.exit(0)
def grab(html, pattern):
    m = re.search(pattern, html, re.S)
    return m.group(1) if m else None
title = grab(dev, r"<title>(.*?)</title>")
heading = grab(dev, r"<h1>(.*?)</h1>")
merged = repo
if title:
    merged = re.sub(r"<title>.*?</title>", lambda _: "<title>%s</title>" % title, merged, count=1, flags=re.S)
if heading:
    merged = re.sub(r"<h1>.*?</h1>", lambda _: "<h1>%s</h1>" % heading, merged, count=1, flags=re.S)
if merged != dev:
    io.open(dest, "w", encoding="utf-8").write(merged)
    print("  merged : popup.html (markup from the repo, client branding kept)")
PY_POPUP


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

# Syntax-check every JS file that is source (not backups), wherever it now lives.
while IFS= read -r -d '' js; do
  rel="${js#"$DEST"/}"; top="${rel%%/*}"
  if [ "$top" != "$rel" ]; then case " ${SKIP_DIRS[*]} " in *" $top "*) continue ;; esac; fi
  case "$(basename "$js")" in *_20[0-9][0-9]-*|*.bak) continue ;; esac
  node --check "$js"
done < <(find "$DEST" -name '*.js' -print0 | sort -z)
echo
echo "synced -> $DEST"
echo "version  v$repo_ver -> v$new_ver   (reload the extension; Chrome must show v$new_ver)"
