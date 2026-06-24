# Solo Agency Setup Flow Entrypoint

Use this file as the entrypoint for setup/configuration sessions.

Setup Flow is the control plane. It configures Solo Agency so automation tasks run correctly later. It does not run operational reports.

## Setup Flow Contract

1. Load `SOLO_AGENCY_PLAYBOOK.md`.
2. Load `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`, `playbooks/01_BASIC_PROFILE_PUBLIC_REPORT.md`, `playbooks/04_DAILY_SCHEDULE.md`, `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`, and `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md`.
3. Load `playbooks/PRIVATE_SOURCE_GATE.md`, `playbooks/02_PRIVATE_SOURCE_SETUP.md`, and `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md` when private data sources, client Chrome profiles, client extensions, or Local Collector setup are involved.
4. Create or update client setup, public data sources, private data sources approval state, extension folders, collector config, schedule files, automation manifests, scheduled prompts, and resync logs.
5. Do not run public scans, private data source scans, reports, first agency runs, production, rendering, publishing, analytics scans, or outreach in Setup Flow.
6. If the human asks for a report or run inside Setup Flow, verify the relevant automation task and tell the human to run that task instead.
7. Every client-specific automation task name must begin with the client name, for example `AvenNgo - Solo Agency Daily Run`.
8. Every per-client Chrome extension display name must begin with the client name, for example `AvenNgo - Solo Agency Collector`.
9. After any approved config change, perform Automation Resync if a schedule/automation already exists.
10. Setup Flow completion means `ready_for_automation_first_run` or `ready_for_next_automation_run`.

## Required Setup Output

For each configured client, Setup Flow must leave these current:

- Client Intelligence Profile.
- public data sources and keyword bank.
- private data sources approval state.
- `extensions/{client_slug}/manifest.json`.
- `extensions/{client_slug}/client_binding.json`.
- `daily-content-pipeline/collector/extension_registry.json`.
- `daily-content-pipeline/collector/collector_config.json`.
- `daily-content-pipeline/schedule.md`.
- `daily-content-pipeline/automation/automation_manifest.md`.
- `daily-content-pipeline/automation/scheduled_run_prompt.md`.
- `daily-content-pipeline/automation/resync_log.md`.

If the native automation task prompt cannot be updated directly, mark `automation_prompt_update_pending` in the manifest and schedule, then give the human one concrete instruction to update the task prompt.
