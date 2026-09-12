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
