# Setup entry acceptance checklist (manual, one runtime at a time)

Why this file exists: on 2026-09-12 the Boss found that the same install request, worded differently, made
Codex behave differently — one phrasing followed the playbook, another web-searched and improvised, a third
introduced Sam and then handed the terminal back after a single failed process start. Nothing here can be
automated: it needs a real session in a real runtime, started from a clean folder. Run it after any change to
`README.md`, `AGENTS.md`, `playbooks/SETUP_FLOW_ENTRYPOINT.md` or `playbooks/TEAM_MODEL.md`.

## How to run one case

1. New empty folder, and a path whose parent exists but whose last segment does NOT (e.g. `~/Documents/client`
   when `client` is missing) — that missing directory is the point of the test.
2. Fresh chat in the runtime under test (Codex app, Claude Code desktop, or whatever is being certified).
3. Send ONE line from the phrasing list, nothing else. Answer nothing until the five checks are scored.

## Phrasings (each is one case; add local-language variants freely, they must all pass)

| # | Message |
|---|---|
| 1 | `Setup https://github.com/soloagency/solo-agency cho tôi.` |
| 2 | `Hãy tìm hiểu Solo Agency trên github và setup cho tôi` |
| 3 | `Tìm hiểu Solo Agency trên github và cài đặt cho tôi` |
| 4 | `Setup Solo Agency cho tôi. tìm trên github ý` |
| 5 | `Cài Solo Agency` |
| 6 | `Thiết lập Solo Agency giúp tôi` |
| 7 | `Install Solo Agency from GitHub` |

## The five checks, scored per case

- [ ] **A. First words.** The very first human-visible message is the Team Leader opening line
      (`playbooks/TEAM_MODEL.md`). No plan, no "I will now…", no web-search recap, no tool narration before it.
- [ ] **B. No improvising from the web page.** Whatever it looked up, it clones the repo and reads `AGENTS.md`
      and `SOLO_AGENCY_PLAYBOOK.md` before acting on anything it read on a rendered GitHub page.
- [ ] **C. One flow.** All seven phrasings converge on `playbooks/SETUP_FLOW_ENTRYPOINT.md` and the same
      10-step roadmap. No generic package-installer or "research the repository" routine.
- [ ] **D. A failed process start is recovered, not handed over.** With the missing directory from step 1, the
      run creates it from the nearest existing parent and retries; it does not conclude "this session cannot
      run commands" and does not hand the human a clone command on that evidence.
- [ ] **E. Honest state.** It never says an install, clone or scan happened when it did not.

## Recording a run

One line per case in the session's notes: `case {n} · {runtime} · A/B/C/D/E = pass|fail · what broke`.
A single fail is a regression in the entry path, not a one-off: fix the wording that allowed it before
shipping, and note which file carried the fix.

## Step 5 — automatic sources acceptance

- [ ] The human-facing roadmap still contains exactly 10 numbered setup steps; Step 5 remains Sources.
- [ ] Step 5 automatically configures default sources and the per-channel keyword banks. It asks no custom-URL question and adds no `**[ACTION REQUIRED]**` block for sources.
- [ ] A missing custom-source response never delays Step 6 scheduling/automation or Step 7 first-run dispatch.
- [ ] If the Boss volunteers a URL during setup, it is preserved and processed when the collector is ready without becoming a separate question or gate.
- [ ] After setup, the Boss can add a custom source using Revenue Engine `watch_source` (Watch a group, page, profile, or competitor) or a direct request to Sam; the change triggers Automation Resync.

## Step 6/7 — automatic default schedule, retained run permission

- [ ] With no explicit or saved schedule, Step 6 automatically creates the client Daily Run daily at 09:00 local; it calls `schedule-slots suggest`, creates at the returned actual time, registers it, reports the actual start time and finish window, and says the Boss can change it later.
- [ ] Step 6 asks no cadence/start-time question and has no `**[ACTION REQUIRED]**` block for the reversible default. An explicit schedule wins, and repair/update preserves an existing schedule.
- [ ] If schedule-slots is genuinely unavailable, the first-task exception uses 09:00 local (or the explicit override), records `slot_check_pending`, and registers once the tool returns.
- [ ] Step 7 still asks the existing run-now yes/not-now question. The first run is never dispatched before the Boss answers yes.
- [ ] After yes, Codex and Claude create the exact separate `{Client} - Solo Agency First Run`, never reuse `{Client} - Solo Agency Daily Run`, start it autonomously, and never ask the Boss to click Run now/open Automations.
- [ ] Claude uses one-time `fireAt` and Codex starts that one-time task natively; both carry the latest client-pinned contract, record lifecycle/idempotency, wait/report, then delete/mark the First Run task deleted while the Daily Run persists.
- [ ] `first_run_task_id` differs from `daily_run_task_id`; the recurring Daily Run remains registered after First Run cleanup.
- [ ] Cleanup records `first_run_task_deleted_at` after the First-Run Report; this evidence remains after first-run status becomes `deleted`.
