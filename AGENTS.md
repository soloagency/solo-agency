# Agent Instructions

When the user asks to set up this repo, always read `SOLO_AGENCY_PLAYBOOK.md` first and follow its checklist in order.

Do not install, start, or configure `solo-agency-collector/` before the playbook explicitly reaches the Local Collector/private data source stage and the human approves it.

If the human asks to scan, monitor, collect, or review private/logged-in groups, feeds, profiles, communities, or social sources after any amount of conversation drift, reload `playbooks/PRIVATE_SOURCE_GATE.md`, `playbooks/02_PRIVATE_SOURCE_SETUP.md`, `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md`, and `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md` before taking action.

Never use Claude in Chrome, Claude Chrome Extension, Codex built-in/in-app browser, Playwright/Puppeteer/Selenium, a fresh agent-opened browser profile, or any agent-controlled browser to read logged-in/private data sources. Use only the Solo Agency Local Collector extension plus the Local Collector app for private data source collection.

During Local Collector activation, do not run `setup_collector.sh`, PowerShell setup scripts, `.cmd` launchers, or collector binaries from inside the AI agent, even if shell permissions are available. Prepare the files, then give the human the one-line Terminal/PowerShell command to run outside the AI sandbox and the Chrome extension `Load unpacked` folder path.

When checking an already-running Local Collector app, do not trust `ready` alone. Verify `/status.config_file`, `/status.output_dir`, and `/status.run_now_request_file` point to the current setup's `daily-content-pipeline/collector/` tree. If they point to another setup, treat it as `wrong_workspace_bridge`, ask the human to run the current setup's Local Collector command, and remind them to remove/disable old Solo Agency Local Collector extensions in `chrome://extensions`.

The repo entrypoint is `SOLO_AGENCY_PLAYBOOK.md`, not `solo-agency-collector/`.

## Imported Claude Cowork project instructions
