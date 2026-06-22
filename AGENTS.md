# Agent Instructions

When the user asks to set up this repo, always read `SOLO_AGENCY_PLAYBOOK.md` first and follow its checklist in order.

Do not install, start, or configure `solo-agency-collector/` before the playbook explicitly reaches the Local Collector/private-source stage and the human approves it.

During Local Collector activation, do not run `setup_collector.sh`, PowerShell setup scripts, `.cmd` launchers, or collector binaries from inside the AI agent, even if shell permissions are available. Prepare the files, then give the human the one-line Terminal/PowerShell command to run outside the AI sandbox and the Chrome extension `Load unpacked` folder path.

The repo entrypoint is `SOLO_AGENCY_PLAYBOOK.md`, not `solo-agency-collector/`.

## Imported Claude Cowork project instructions
