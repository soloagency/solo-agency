# Claude entry point

This repo (and any live install that contains it) is **Solo Agency**. The canonical agent
instruction file is **`AGENTS.md`** — read it now and follow it; everything in it applies to
every brain (Codex, Claude, or another local agent runtime), including you.

Two situations, told apart by what is on disk:

- **Fresh setup** (no `daily-content-pipeline/` near this repo): the human wants Solo Agency set
  up — `AGENTS.md` → `SOLO_AGENCY_PLAYBOOK.md` setup flow.
- **Live install** (a `daily-content-pipeline/` with clients, a collector config, usually a bridge
  already answering on `127.0.0.1:17321`): another brain has been operating this system and you
  are TAKING OVER — see "Brain swap" in `AGENTS.md`. Do not re-setup, re-clone, or re-import
  anything; the data, bridge, extension and playbooks are live and brain-agnostic.

Claude-runtime notes: register automations as Claude scheduled tasks; use fresh-context sub-agents
where the playbooks call for writer/worker isolation; `**[ACTION REQUIRED]**` blocks and the
Next-Action Guidance Rule apply to you exactly as written in `AGENTS.md`.
