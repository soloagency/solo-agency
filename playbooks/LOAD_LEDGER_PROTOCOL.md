# Load Ledger Protocol — read every stage/module IN FULL before acting

This module is the source of truth for **full-load discipline**. Master (`OUTREACHCRM_PLAYBOOK.md`) and both entrypoints (`SETUP_FLOW_ENTRYPOINT.md`, `SCHEDULED_RUN_ENTRYPOINT.md`) reference it. Load it at the very start of any run, and obey it every time you load a stage/module/dependency.

Reason it exists: OutreachCRM playbooks are large (several are 600-1400 lines) and are loaded on demand. A large file can be returned **truncated** ("output too large", "persisted output", preview-only), or a GitHub-raw download can be **partial/stale**. Acting on a half-read stage silently drops rules. This protocol makes acting on a partially-read file impossible.

---

## Rule 1 — A failed/truncated/partial read = NOT loaded

If any read of a stage/module/dependency **errors, truncates, returns only a preview/part, 404s, times out, or returns fewer lines than expected**, treat that file as **NOT loaded**. It is FORBIDDEN to: act on the preview/partial text; infer the rest from memory; skip it; or mark it "loaded". You MUST re-read the whole file to its end **before** doing the step that needs it. No exception for time, cost, file size, or "I'm running from a schedule".

## Rule 2 — Read large files in chunks until EOF

When a file is too large for one read, page through it: `Read` with `offset`/`limit`, `sed -n 'A,Bp'`, `head`/`tail`, or the transport's pagination. Continue until you have read the **last line**. Do not stop at the first chunk. This is exactly the step that gets skipped when a big stage errors with "output too large".

## Rule 3 — Print a LOAD LEDGER at the moment you load, before acting

Every time you load a stage/module/dependency, print this **minimal** block **before** you act on it. One number is the check — do not turn the ledger into a second task:

```text
LOAD LEDGER:
File: <relative path, e.g. playbooks/09_OPERATIONS_SAFETY_AUDIT.md>
lines_read=<N>   manifest_lines=<M from LOAD_MANIFEST.md | manifest=absent>
Full-read check: <PASS lines_read>=manifest_lines (equal, or +1 from a trailing-newline reader)
                | PASS(no manifest) manifest=absent — quote last line to prove EOF: last="<...>"
                | FAIL lines_read<manifest_lines — truncated; re-read to EOF / re-fetch, then re-check>
Dependencies this file names: <each MUST have its own LOAD LEDGER above>
Verdict: <PASS loaded-in-full | BLOCKED>
```

- **The required check is line count only.** Manifest present (the normal case — deploy regenerates it): `lines_read` MUST be at least `manifest_lines`; a shortfall means the read was truncated. That single comparison catches the core "output too large" failure — nothing else is required per load.
- **Counting convention:** `manifest_lines` is the newline count (`wc -l` / `awk 'END{print NR}'`). Some file readers report one extra line for the final newline or a trailing blank line, so `lines_read` may legitimately be `manifest_lines` or `manifest_lines + 1` — both PASS as long as you reached the real last line (the manifest's `last_line`). Treat only `lines_read < manifest_lines` as truncated. A larger gap above the manifest means the file was edited/grew since the manifest was generated — re-run the deploy manifest or fall back to quoting the last line.
- **No manifest (rare):** nothing to compare against, so quote the exact `last` line instead — you can only quote it after reading to EOF.
- **`last_line` + `sha256` stay in the manifest as an OPTIONAL deeper check**, not a per-load requirement. Use them only when the line count matches but you suspect a stale/edited file, or for an occasional integrity spot-check. Do not quote them every load — extra ceremony makes agents drop real work.

## Rule 4 — Dependency-complete

When a stage's "Load When" (Stage Map) or its own text names dependencies (e.g. Stage 4 → skill `email-verify-enrich/SKILL.md` → its modules, or Stage 6 → skill `email-writing/SKILL.md` → its modules), each named dependency needs its **own** LOAD LEDGER. The parent is not "loaded" until every named child is ledgered in full. Loading the parent and skipping a child is the multi-tier version of the same miss.

## Rule 5 — Verify GitHub-raw downloads against the manifest

If a stage is missing locally and fetched from the OutreachCRM GitHub raw URL (`https://raw.githubusercontent.com/soloagency/outreachcrm/main/playbooks/…`), the download itself can be partial/stale. After fetching, run the LOAD LEDGER against `LOAD_MANIFEST.md`. Mismatch = bad download → re-fetch; never act on a partial download.

## Rule 6 — No side-effect action without a PASS ledger above it

Before any side-effect step (ask the first setup question; run a report/scan; render/export; publish; notify; write client/automation state; call a provider write; claim completion), the stage(s) that step depends on must already have a `Verdict: PASS loaded-in-full` LOAD LEDGER earlier in the same transcript, with dependencies ledgered. Missing → print/complete the ledger first. (Announce ≠ pause: print, then keep working.)

## Rule 7 — No excuse suppresses full-load or the ledger

Forbidden excuses for reading a stage partially, skipping a dependency, or dropping the ledger: "the file is too large / output too large", "to save time/tokens/credits", "the human wants it short", "I already loaded it earlier / I remember it", "I'm running from a schedule", "I only need the top of the file", "the download looked fine". Brevity applies only to the human-facing summary, never to loading or to the ledger.

---

## Completion phrasing

Everywhere a gate or checklist says "Stage X was loaded", read it as **"Stage X was loaded IN FULL — LOAD LEDGER printed with Verdict PASS, matching `LOAD_MANIFEST.md` when present, dependencies ledgered"**. A file read to 200 of 1900 lines is NOT loaded.

## LOAD_MANIFEST.md (Tier B, auto-generated)

`playbooks/LOAD_MANIFEST.md` lists `OUTREACHCRM_PLAYBOOK.md` plus every `playbooks/**/*.md` with `path | lines | sha256 | last_line`. It is regenerated automatically by `deploy-outreachcrm.sh` on every deploy and published to GitHub raw, so adding a new playbook needs no manual step. If the manifest is absent or a file is not yet listed, fall back to Tier A (quote lines + last line). If present, use it for deterministic truncation/staleness detection.
