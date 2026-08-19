# Claude entry point

This repo (and any live install that contains it) is **Solo Agency**. The canonical agent
instruction file is **`AGENTS.md`** — read it now and follow it; everything in it applies to
every brain (Codex, Claude, or another local agent runtime), including you.

Two situations, told apart by what is on disk:

- **Fresh setup** (no `daily-content-pipeline/` near this repo): the human wants Solo Agency set
  up — `AGENTS.md` → `SOLO_AGENCY_PLAYBOOK.md` setup flow.
- **Live install** (a `daily-content-pipeline/` with clients, a collector config, usually a bridge
  already answering on `127.0.0.1:17321`): another brain has been operating this system and you
  are either TAKING OVER or WORKING ALONGSIDE it — see "Brain swap and multi-brain operation" in
  `AGENTS.md`. Do not re-setup, re-clone, or re-import anything; the data, bridge, extension and
  playbooks are live and brain-agnostic.

Another agent may be driving this same install right now, and the human may deliberately be using
both at once. That is supported: read `playbooks/MULTI_BRAIN_OPERATIONS.md` and take the
`run_lock` or scoped work lease your task requires before touching client state. Never tell the
human to close Codex so you can work — say which scope is held, by whom, and what is free.

Claude-runtime notes: register automations as Claude scheduled tasks only for the tasks this
runtime OWNS (ownership is per task, recorded in `automation_manifest.md`); use fresh-context
sub-agents where the playbooks call for writer/worker isolation; `**[ACTION REQUIRED]**` blocks
and the Next-Action Guidance Rule apply to you exactly as written in `AGENTS.md`.

Running the installed bridge and its tools on a live install (`<bridge> tool crm-store ...`,
`tool gmail ...`) is normal OPERATION, not a code change and not a setup step — no "ask before
editing code" convention applies to it, and the setup-flow collector gate in `AGENTS.md` never
forbids it. Editing files in this source repo IS a code change and follows the normal repo rules.
