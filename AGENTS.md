# Agent Instructions

When the user asks to set up this repo, always read `SOLO_AGENCY_PLAYBOOK.md` first and follow its checklist in order.

During Setup Flow, never run, create, generate, show, refresh, or update a report in the setup chat, even if the human explicitly asks. Treat the request as a handoff request: verify/resync the client-specific automation task, tell the human the exact task name to run, and do not load the scheduled-run entrypoint or perform public research/private collection/report generation inside Setup Flow.

Do not install, start, or configure `solo-agency-collector/` before the playbook explicitly reaches the Local Collector/private data source stage and the human approves it.

Use the canonical terms `public data sources` and `private data sources` in human-facing text. Do not shorten them, omit `data`, or use slash labels.

If the human asks to scan, monitor, collect, or review private data sources (logged-in groups, feeds, profiles, communities, or social sources) after any amount of conversation drift, reload `playbooks/PRIVATE_SOURCE_GATE.md`, `playbooks/02_PRIVATE_SOURCE_SETUP.md`, `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md`, and `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md` before taking action.

Never use Claude in Chrome, Claude Chrome Extension, Codex built-in/in-app browser, Playwright/Puppeteer/Selenium, a fresh agent-opened browser profile, or any agent-controlled browser to read private data sources. Use only the Solo Agency Local Collector extension plus the Local Collector app for private data source collection.

During Local Collector activation, do not run `setup_collector.sh`, PowerShell setup scripts, `.cmd` launchers, or collector binaries from inside the AI agent, even if shell permissions are available. Prepare the files, then give the human the one-line Terminal/PowerShell command to run outside the AI sandbox and the Chrome extension `Load unpacked` folder path.

When checking an already-running Local Collector app, do not trust `ready` alone. Verify `/status.config_file`, `/status.output_dir`, and `/status.run_now_request_file` point to the current setup's `daily-content-pipeline/collector/` tree. If they point to another setup, treat it as `wrong_workspace_bridge`, ask the human to run the current setup's Local Collector command, and remind them to remove/disable old Solo Agency Local Collector extensions in `chrome://extensions`.

During scheduled/manual runs, do not skip private data sources only because saved config says `public_data_sources_only`, `private sources postponed`, or `pending_private_activation`. If private data sources exist in any state or collector status files exist, perform Collector Runtime Verification first: try `/status`, verify current-workspace identity, and if localhost is unreachable from the AI sandbox, read local collector health/status files before deciding.

For multi-client manual/scheduled private collection, do not have multiple agents write the same `daily-content-pipeline/collector/run_now_request.json` file. Prefer `POST /jobs/run_now` when localhost is reachable, or write one unique per-client job file under `daily-content-pipeline/collector/jobs/pending/` when using file-based requests. The shared bridge supports parallel private collector jobs across different `client_slug` identities, binds each active job to the claiming extension instance when present, and serializes jobs only within the same client/profile.

After a schedule/automation has been configured, any later human-approved change to sources, approvals, Local Collector status, PDNA, provider/OpenAPI config, notification, analytics, profile fields, cadence, or playbook behavior must trigger an Automation Resync. Do not update only one config file. Update the Client Intelligence Profile, provider config/capability cache when relevant, schedule.md, collector config when relevant, automation manifest, scheduled-run prompt/task body, and resync log; then verify the next scheduled run will read the newest state.

Every human-facing progress block after schedule/automation exists must include an Automation freshness check: whether the latest changes were synced into the automation/scheduled task prompt/contract/playbook/source state, not only config, and whether tomorrow's scheduled run will load the newest state.

Every scheduled/manual report handoff must include a Provider Report Delivery Capability Check outcome: whether the configured provider/OpenAPI spec was discovered, whether the provider account was verified, whether HTML upload/Telegram or email-fallback notification was attempted, the uploaded URL or exact blocker, and the final HTML report path/link. For WideCast, check the per-client OpenAPI path first (`https://widecast.ai/openapi.yaml`, `uploadAsset`, `sendTelegramMessage`) and do not claim WideCast itself lacks capability merely because the current AI/MCP tool surface does not expose it.

The repo entrypoint is `SOLO_AGENCY_PLAYBOOK.md`, not `solo-agency-collector/`.

## Imported Claude Cowork project instructions
