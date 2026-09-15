#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/soloagency-backup-hygiene.XXXXXX")
trap 'rm -rf "$work"' EXIT

cp "$repo_root/.gitignore" "$work/.gitignore"
cp "$repo_root/deploy-soloagency.sh" "$work/deploy-soloagency.sh"
chmod +x "$work/deploy-soloagency.sh"
cd "$work"
cd -P .

git init -q
git config user.name 'Backup hygiene acceptance'
git config user.email 'backup-hygiene@example.invalid'
git add .gitignore
git add -f deploy-soloagency.sh
git commit -qm 'fixture'

assert_ignored() {
  local path=$1
  mkdir -p "$(dirname "$path")"
  : > "$path"
  git check-ignore -q -- "$path" || {
    echo "expected ignored backup path: $path" >&2
    exit 1
  }
}

# Directories, ordinary suffixes, and both timestamp forms must all be ignored.
assert_ignored 'backups/session/AGENTS.md'
assert_ignored 'backup/AGENTS.md'
assert_ignored '_backup/AGENTS.md'
assert_ignored 'playbooks/backup/AGENTS.md'
assert_ignored 'playbooks/backups/AGENTS.md'
assert_ignored 'playbooks/_backup/AGENTS.md'
assert_ignored 'notes/plan.bak'
assert_ignored 'notes/plan.bak.gz'
assert_ignored 'notes/plan.backup'
assert_ignored 'notes/plan.backup.gz'
assert_ignored 'notes/AGENTS_20260912_1743.md'
assert_ignored 'notes/LICENSE_20260912_1743'
assert_ignored 'notes/AGENTS_2026-09-12_2057.md'
assert_ignored 'notes/LICENSE_2026-09-12_20-57-49'

# The guard catches a deliberately force-added ignored file.
printf 'backup' > accidental.bak
git add -f accidental.bak
if ./deploy-soloagency.sh --backup-hygiene-check >/dev/null 2>&1; then
  echo 'backup guard accepted a force-added backup file' >&2
  exit 1
fi
git reset -q -- accidental.bak
rm accidental.bak

# A currently tracked legacy backup fails, but its staged deletion is allowed so
# the one cleanup commit can remove every historical stray file in one pass.
printf 'legacy' > AGENTS_20260912_1743.md
git add -f AGENTS_20260912_1743.md
git commit -qm 'legacy backup fixture'
if ./deploy-soloagency.sh --backup-hygiene-check >/dev/null 2>&1; then
  echo 'backup guard accepted a tracked legacy backup' >&2
  exit 1
fi
git rm -q --cached AGENTS_20260912_1743.md
./deploy-soloagency.sh --backup-hygiene-check >/dev/null

echo 'backup hygiene acceptance passed'
