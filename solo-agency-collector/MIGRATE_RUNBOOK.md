# Migrate — export / import the working environment

Move a client (or the whole agency) between machines and between AI agents
(Claude ↔ Codex) with the environment restored intact. Agent-agnostic: both
agents run the SAME `tool migrate` commands; the only agent-specific step is
re-registering scheduled tasks in the destination agent's own scheduler.

## What moves, what doesn't (four layers)

| Layer | Handled by |
|---|---|
| **Data** (CRM, campaigns, sent logs, reports, profiles, cursors) | the bundle, verbatim — cursors move as-is so there is no double-send / re-notify after the move |
| **Secrets** (Gmail App Passwords, WideCast key, tracking secret, ui_token) | the bundle, AES-256-GCM encrypted with the operator's passphrase (agent never sees the passphrase or the plaintext) |
| **Tasks** (automation / schedule) | manifest hints + carried prompt files; the **destination agent re-registers** them in its own scheduler (the only step the tool can't do) |
| **Machine-specific** (bridge binary, Chrome extension, autostart, absolute paths) | rebuilt on the destination: `setup_collector` for the binary+autostart, extension re-install by hand, paths auto-rebased on import |

## Source: export

1. The operator sets the passphrase in the environment (the agent never sees it):
   `export SOLO_AGENCY_EXPORT_PASSPHRASE='…a strong passphrase…'`
2. Run the export (whole agency, or one/few clients):
   ```sh
   # whole system
   <bridge> tool migrate export --data-root <DATA_ROOT> --scope agency \
     --out ~/solo-agency-export.zip --agent claude
   # a single client
   <bridge> tool migrate export --data-root <DATA_ROOT> --scope client \
     --clients leadup --out ~/leadup-export.zip --agent claude
   ```
   `<DATA_ROOT>` = the `daily-content-pipeline/` directory. `--agent` records the
   source (claude|codex) for the report; it does not change the data.
   Add `--no-secrets` to leave every credential out (the destination re-auths
   instead); then no passphrase is needed.
3. Move `solo-agency-export.zip` to the destination machine (the operator's own
   file transfer — the bundle is encrypted at rest if it holds secrets).
4. **Do NOT run the source and destination at once.** A migration is a MOVE, not
   a clone: after the destination is verified, pause the source's campaigns,
   disable the source's scheduled tasks, and stop the source bridge, or the same
   campaign will send from two machines.

## Destination: bootstrap the foundation FIRST (from the repo, not the bundle)

The bundle carries data + config + task hints only — never the binary or the
playbooks. So on the destination, first:

1. Install the collector for this OS (binary + autostart): run `setup_collector.sh`
   (macOS/Linux) or `setup_collector.ps1` (Windows) per `AGENT_RUNBOOK.md`. This
   also brings the current playbooks/tools via the dist download.
2. The bridge is now running with an EMPTY data root — that's expected; the
   import fills it.

## Destination: import

1. The operator sets the SAME passphrase used at export:
   `export SOLO_AGENCY_EXPORT_PASSPHRASE='…the same passphrase…'`
2. **Dry-run first** — shows exactly what will land, the parsed tasks, and the
   next steps, writing nothing:
   ```sh
   <bridge> tool migrate import --file ~/solo-agency-export.zip \
     --root <DEST_DATA_ROOT> --dry-run
   ```
3. Real import:
   ```sh
   <bridge> tool migrate import --file ~/solo-agency-export.zip --root <DEST_DATA_ROOT>
   ```
   - It verifies every file checksum, places data, decrypts + restores secrets
     (0600), **rebases** the source machine's absolute paths in the automation
     prompts to the destination, and rebuilds the CRM identity index.
   - If the destination already has one of the clients, it REFUSES unless you add
     `--force` (guards against clobbering a live install — you should have
     deactivated the source instead).
   - `residual_source_paths` in the report lists any automation file where a
     source path could not be rebased automatically (different checkout layout) —
     fix those by hand before running the tasks.

## Destination: re-register the automation tasks (the cross-agent seam)

The tool cannot create another agent's scheduled tasks. Take the `tasks` list
from the import report and re-create each one in THIS agent's scheduler, using
the carried (now path-rebased) prompt file as the run prompt:

- Each task carries: `name`, `cadence` (e.g. "weekdays 07:00"), `timezone`,
  `client_slug`, `campaign_slug`, `notification_channel`, `prompt_file`.
- **On Claude**: create a scheduled task whose prompt is the contents of
  `<DEST_DATA_ROOT>/<prompt_file>`, at the given cadence/timezone. The prompt
  already pins `target_client_slug` / `campaign_slug` in its prose.
- **On Codex**: register the same in Codex's task system, same prompt + cadence.
- After creating each task, update `schedule.md` / `automation/automation_manifest.md`
  with the destination's native task id (Automation Resync contract in
  `outreach/playbooks/AUTOMATION_SCHEDULING.md`).
- Verify the prompt's paths point at the DESTINATION root (the import rebased
  them; confirm no `residual_source_paths` remained).

## Destination: the remaining by-hand steps

1. **Chrome extension**: reinstall the collector extension for each client and
   sign into that client's Chrome profile — browser sessions cannot be exported
   (see the drag-drop flow in `AGENT_RUNBOOK.md`).
2. **Sendboxes**: the App Passwords moved with the bundle; verify each box with a
   sync (`<bridge> tool gmail sync`) — App Passwords are not IP-bound, so they
   should work immediately. If Google flagged a new-location sign-in, re-connect
   that box on the Sendboxes page.
3. **Provider / tracking**: the WideCast key + tracking secret moved; nothing to
   re-enter. (`--no-secrets` exports omit these — reconnect on the Sendboxes /
   provider setup pages.)

## Verify

- `curl -s http://127.0.0.1:<port>/status` answers.
- Open `/ui/{client}/sent` and `/ui/{client}/campaigns` — the data is there.
- Run the first daily task manually; confirm it reads the destination paths and
  notifies through the moved provider.

## Maintenance as the system grows

The export walks the whole data tree and copies everything by default, so most
growth needs NO change here:

- **New playbooks / code / tools** — not in the data tree (they ship from the
  repo/dist; the destination bootstraps them). Export never touches them.
- **New data files or directories** — copied automatically as `data`.
- **New cursor / idempotency files** — copied verbatim like any data.

Only two things couple to this code, and both now **fail loud instead of silent**:

1. **A new SECRET file type** must be added to `portClassify` (bridge-go/
   portability.go) so it travels encrypted, not as plaintext data. If you forget,
   the export growth-guard flags it: secrets here are always written `0600`, so
   any `0600` file the classifier didn't tag as secret is reported as
   `unclassified_sensitive` in the export result (and logged). Register it, or
   add it to `portBenign0600` if it is genuinely not a secret.
2. **A new file that bakes in an absolute path** — auto-rebase only rewrites the
   task prose (`schedule.md`, `automation/**`). Any other file carrying the
   source path is reported by import under `residual_source_paths` (it scans
   every text file), so the agent can fix it. Prefer relative paths / runtime-
   derived paths in new writers to avoid this entirely.

A bundle-format change (rare) bumps `portSchemaVersion`; import refuses a newer
bundle than the destination bridge understands.

## Deactivate the source (finish the move)

Only after the destination is verified: on the SOURCE, pause every campaign
(`campaign update --json '{"status":"paused"}'` or the Campaigns page), disable
its scheduled tasks in its agent's scheduler, and stop its bridge. This is what
makes it a move, not a clone.
