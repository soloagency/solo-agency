# Collector Healthcheck (active capability probes)

Facebook and Zillow change their GraphQL and DOM without notice; a collector capability can go
silently blind while `/status` still says `recent`. The healthcheck re-runs every catalog
capability against fixtures the operator owns, scores the records the collector returns, and
names the capability that broke, why, and which maintenance procedure to open. This playbook is
how ANY brain (Codex, Claude Code, Claude Cowork, another runtime) drives it: the human decides,
the agent asks only for what is missing and relays the verdicts.

It is NOT the passive `Private Collector Health Check Protocol` in
`playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md` (bridge reachable, extension recent, right
workspace). That protocol is the precondition; this one is the layer above it. Mechanics (CLI,
HTTP routes, fixture keys, verdict rules, assertion DSL, alerts) live in
`solo-agency-collector/HEALTHCHECK.md`; this file is the operating procedure.

Plan gating never touches the healthcheck: probes the bridge enqueues itself (`source: healthcheck`) bypass every plan gate, and the dev executor workspace (aven-ngo) is not a CRM client, so a Free install can always run its diagnostics. If a probe record carries `solo_entitlement_required`, that is the extension's own plan check (enforced since launch: the capability's feature is not in this install's plan) — record it in the report, do not count the probe as a collector failure.

## Load Rule

Load when the human says any of: healthcheck, "kiểm tra collector", "capability còn sống
không", "test the extension", "Facebook changed something", "why did the scan come back empty";
after a bridge or extension update; before registering a scheduled healthcheck; at the start of
every scheduled healthcheck run. Also load `playbooks/MULTI_BRAIN_OPERATIONS.md` (live install)
and keep Stage 8 loaded for the passive liveness check.

## Hard Gates

- The human chooses the executor. List the connected extensions and ask; never pick one, never
  default to the last one used. Production client extensions run read-only suites only.
- Real writes (`--allow-writes`) only on the operator's own test client, only into assets the
  operator owns (their test group, their own post, a consenting recipient), and only after the
  human approved that exact command. You compose the probe text (`--text`); never reuse a
  client's content. No delete/unsend capability exists: a real comment, post or DM stays until
  the operator removes it by hand — say so before asking for approval.
- Ask the human ONLY for fixture keys the tool reports missing. Never ask for passwords, cookies,
  tokens, or anything the Hard Gates in `SOLO_AGENCY_PLAYBOOK.md` forbid. Never invent, guess or
  borrow a fixture from client data; a fixture must be the operator's own target.
- Use `<bridge>` — the installed runtime binary defined in `00_CORE_CONTEXT_REQUIREMENTS.md` —
  never a binary from the source tree. If `<bridge> tool healthcheck` answers `unknown
  subcommand`, or the bridge answers `404` on `/healthcheck/plan`, the installed bridge predates
  this feature: hand the human the setup rerun command from Stage 11 ("Bridge And Extension
  Change Protocol") and stop.
- One healthcheck run at a time per install (the bridge refuses a second with `409`). Take a
  `client` lease for the executor's client before `run`; never start while another brain holds
  that client's `run_lock` or lease.
- Never "fix" a FAIL by editing the catalog, the fixture, or the report to make it pass. A FAIL
  is the finding; the repair is code work (`GRAPHQL_MAINTENANCE.md §7`,
  `HANDOFF_WRITE_ACTIONS.md`, `ZILLOW_CAPABILITIES.md §7`) that a developer session does in the
  source repo and ships through the normal update flow.
- Verdict vocabulary is fixed: PASS, WARN, FAIL, BLOCKED, SKIPPED, EXCLUDED. BLOCKED and
  SKIPPED are never reported as failures.

## Files And Commands

Runtime state lives on the install, never in the repo:

```text
{install_root}/daily-content-pipeline/collector/healthcheck/fixtures.json   # operator-owned targets, flat key: "value"
{install_root}/daily-content-pipeline/collector/healthcheck/latest.json     # last report (also /ui/status → "Healthcheck probes")
{install_root}/daily-content-pipeline/collector/healthcheck/reports/        # one JSON per run
{install_root}/daily-content-pipeline/collector/healthcheck/history.jsonl   # one line per run
{install_root}/daily-content-pipeline/collector/healthcheck/baseline.json   # last PASS shape per probe (the drift sensor)
```

```text
<bridge> tool healthcheck executors                       # connected extensions: client, instance, version, freshness
<bridge> tool healthcheck fixtures                        # every fixture key, present or MISSING, with what it means
<bridge> tool healthcheck plan --client {slug}            # ordered probes, effective mode, missing fixtures per probe
<bridge> tool healthcheck run --client {slug}             # daily suite: reads, dry-run composers, one idempotent Like
<bridge> tool healthcheck run --client {slug} --only fb.group.posts,fb.post.comments
<bridge> tool healthcheck run --client {slug} --suite weekly --allow-writes --text "{your probe text}"
<bridge> tool healthcheck report [--run {run_id}]         # last (or one) report
<bridge> tool healthcheck history --days 14
```

Exit codes: `0` no FAIL, `1` a FAIL exists, `2` usage or no executor chosen. `--json` prints the
raw report. `run` blocks until the run finishes: typically 5–12 minutes for the daily suite; a
Zillow probe waiting on Press & Hold adds up to 5 minutes. Without `--client` the command prints
the executor list and exits `2` — that is the cue to ask the human, not an error.

## Procedure

### Step 0 — Preconditions

1. Run the passive `Private Collector Health Check Protocol` (Stage 8):
   `GET http://127.0.0.1:17321/status`, the workspace identity check, and at least one extension
   `recent`. If the bridge is offline or `wrong_workspace_bridge`, stop with that protocol's
   handoff.
2. Probe the feature: `<bridge> tool healthcheck` must print usage and exit `2`. Anything else →
   the Hard Gate above (installed bridge too old).
3. If another brain may be operating this install, identify yourself once in the reply
   ("Claude Code, running the collector healthcheck on aven-ngo").

### Step 1 — The human chooses the executor

1. `<bridge> tool healthcheck executors` (or `GET /status` → `extension_health.extensions[]`).
2. Show the list with client, version and freshness, then ask with the approval block. The
   options are the client slugs; show stale extensions but flag them. Recommend the operator's
   own test client for the full suite and say that production clients get read-only suites only.

```text
**[ACTION REQUIRED]**

**Client:** Solo Agency (agency maintenance)
**Approve one option:** `aven-ngo` / `angela-do (read-only)` / `leadup (read-only)`
**What I will do after approval:** run the daily healthcheck suite on that client's extension and report which capabilities pass, warn, fail or are blocked
**Why:** the probes run inside that client's Chrome session, so only you decide which account is used
```

3. Do nothing until the human replies. Keep the chosen slug for the rest of the session.

### Step 2 — Ask only for the missing fixtures

1. `<bridge> tool healthcheck fixtures`. Keys marked `MISSING` are the only things to ask for. If
   the file does not exist yet every key is missing: ask for the read-only set first, and for the
   write-related keys only when the human wants write probes.
2. Explain each missing key in the human's language, one line each, from the tool's own
   description: what it is and what qualifies (the operator's own group, the test account's own
   post, a consenting recipient). Group them:
   - read-only Facebook: `fb_test_group_url`, `fb_canary_profile_url`,
     `fb_canary_profile_friends_url`, `fb_canary_profile_entity_id`, `fb_canary_profile_name`,
     `fb_canary_profile_email`, `fb_video_page_url`, `fb_reel_url`, `fb_search_keyword`
   - Zillow / web: `zillow_directory_url`, `zillow_profile_url`, `zillow_profile_screen_name`,
     `web_query`, `article_url`
   - writes (dry-run by default): `fb_canary_post_url`, `fb_own_post_url`, `fb_dm_thread_url`,
     `fb_dm_recipient_name`
3. Ask with the generic block. Accept `skip {key}` for anything the human does not want probed:
   that probe is SKIPPED, never failed.

```text
**[ACTION REQUIRED]**

**Client:** Solo Agency (agency maintenance)
**I need you to:** give me the collector healthcheck fixtures that are still missing — {n} keys: {key — what it is, one line each}
**Reply with:** `key = value` per line, or `skip {key}` for any you do not want probed
**Why:** the probes must read targets you own, so an empty result can only mean a broken capability, never an empty page
```

4. Write the answers to
   `{install_root}/daily-content-pipeline/collector/healthcheck/fixtures.json` as a flat JSON
   object (`solo-agency-collector/examples/healthcheck_fixtures.example.json` lists every key).
   Keep existing keys; keys starting with `_` are ignored by the tool. Re-run
   `tool healthcheck fixtures` and show the human what is now present. Values are the
   operator's own targets: treat them like client data (never paste them into notifications).

### Step 3 — Plan and confirm the scope

1. `<bridge> tool healthcheck plan --client {slug}`. Summarise: how many probes will run, which
   are SKIPPED for missing fixtures, which are EXCLUDED and why, and that some probes raise the
   Chrome tab to the front briefly (the extension needs a foreground tab for videos, reels and
   profile enrich).
2. The default scope is `--suite daily` without `--allow-writes`: reads, dry-run composers for
   comment / DM / group post, and one real idempotent Like on the operator's own post. If the
   human asked for real writes, restate exactly what will be written where, that the system
   cannot delete it, and ask for approval of that exact command with the approval block. Compose
   the probe text yourself: short, dated, clearly a test, in the operator's language.

### Step 4 — Lease, run, wait

1. Take a lease: `{install_root}/daily-content-pipeline/automation/leases/{slug}.json` with
   `scope: client`, `scope_key: {slug}`, `intent: "collector healthcheck"`, `held_by_brain`,
   `held_by_session`, `acquired_at`, `ttl_minutes: 30`, `heartbeat_at`. If a fresh lease or
   `run_lock` for that client is held by another brain, report who holds it and offer the free
   executors instead.
2. Run `<bridge> tool healthcheck run --client {slug} [flags]` and wait for it. Do not start a
   second run, do not write `run_now_request.json`, and do not enqueue other jobs for that client
   while it runs.
3. While it runs: if the progress shows `BLOCKED … human gate` on a Zillow probe, tell the human
   immediately: open the executor's Chrome window and pass "Press & Hold" on zillow.com within
   five minutes (a chime plays). If the ceiling passes the probe is scored BLOCKED, not FAIL.
4. Delete the lease when the command returns, including on error.

### Step 5 — Read the verdicts and relay

The report (stdout, `latest.json`, `/ui/status`) carries one verdict per probe:

| Verdict | Meaning | What you do |
|---|---|---|
| PASS | the capability works on the fixture | nothing; the run updated the baseline |
| WARN | works, but through a fallback or with drift (`head_page_via` reshaped, a field that used to be filled is now empty, the query name changed but the shape still parsed) | mention it; WARN twice in a row is a FAIL in the making — recommend opening the repair procedure early |
| FAIL | an assertion or the query sensor broke, or the job timed out after being claimed | relay the report's `**[ACTION REQUIRED]**` block verbatim; name the `cause_group` and the `repair_hint`; several FAILs in one `cause_group` are ONE incident |
| BLOCKED | human gate pending, bot check not passed, account not a member, checkpoint | tell the human what to unblock; never call it a failure |
| SKIPPED | fixture missing, executor stale, dependency probe not PASS, job never claimed within 90 s | list the missing keys or the stale executor; offer a rerun after the fix |
| EXCLUDED | the catalog says not runnable (`not_built`, `planned`) | nothing |

Rules for the reply:

- The CLI already prints a `**[ACTION REQUIRED]**` block when there is something to act on:
  relay it verbatim as the last block of your reply (it satisfies the marker contract). Do not
  reformat it into the four-field template; do add the absolute report path.
- Lead with the one-line summary (`overall`, counts, client, extension build), then the FAIL and
  BLOCKED lines, then WARN. Summarise PASS as a count.
- A FAIL is not fixed by you in this session. State the repair path the report names and offer
  the developer handoff: which file, which procedure, and the evidence already captured
  (`graphql_queries` seen, `pagination_*`, `registry_probe`, `head_page_via`, `seen_textboxes`).
- The runner emails `operator_email` by itself on a new FAIL, a recovery, or three BLOCKED runs
  in a row; do not send a second notification and never use client notification channels.
- End with next-action guidance: rerun after the fix, `--only` for the failed probe, add the
  missing fixtures, or register the scheduled task.

### Adding a capability

A capability is not finished until its catalog entry carries a `healthcheck` block: the
collector's Go tests (`TestCatalogHealthcheckContract`) fail without it, and `plan` lists the new
probe automatically. Developer procedure: `solo-agency-collector/HEALTHCHECK.md` §6 and
`solo-agency-collector/GRAPHQL_MAINTENANCE.md` §8.x. After the build ships, run
`--only {new id}` once on the test client to record its baseline.

## Scheduled Healthcheck Task (only when the human asks)

Manual runs are the default. When the human asks for a recurring healthcheck, register it the
way Stage 11 registers the update watch: create the native task, or write its pending prompt and
hand the creation to the human in an `**[ACTION REQUIRED]**` block naming the task; never
silently skip.

Canonical task name:

```text
Solo Agency - Collector Healthcheck
```

Recommended cadence:

```text
Daily, before client daily runs; the weekly suite on Sunday.
```

Plain-language explanation for the human:

```text
Facebook and Zillow change their internals without notice. A daily healthcheck re-runs every collector capability against targets you own and tells you the same day which one broke and what to fix, instead of a silently empty scan a week later.
```

Scheduled-run prompt (write it to `daily-content-pipeline/automation/scheduled_run_prompt.md`
or the task's own prompt; record `scheduler_runtime`, the task name and the executor slug in
`daily-content-pipeline/automation/automation_manifest.md` per Stage 7):

```text
Load SOLO_AGENCY_PLAYBOOK.md, playbooks/MULTI_BRAIN_OPERATIONS.md, playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md and playbooks/HEALTHCHECK.md. Run only the collector healthcheck: Step 0 preconditions, then `<bridge> tool healthcheck run --client {executor_slug} --suite {daily|weekly}` under the client lease, without --allow-writes. Executor: {executor_slug}, chosen by the human on {date}; do not choose another. Do not ask for fixtures during a scheduled run: a probe SKIPPED for a missing key is reported, not asked about. Write the outcome to the task output and stop. Notify only through the healthcheck's own operator email (automatic on state change); never Telegram, WideCast or client channels. If the report has a FAIL, end with its ACTION REQUIRED block verbatim; otherwise end with the one-line summary and next-action guidance.
```

Boundaries, the same as the update watch: internal agency maintenance, no client notification
channels, no client reports or scans in the same task, exactly one runtime owns the task
(`MULTI_BRAIN_OPERATIONS.md` Tier 1).

## Reference

- `solo-agency-collector/HEALTHCHECK.md` — mechanics: gate tiers, verdict rules, assertion
  DSL, HTTP routes, fixture keys, alerts, capability-id validation, event-log retention.
- `solo-agency-collector/AGENT_RUNBOOK.md` → "Healthcheck Probes (active, per capability)".
- `solo-agency-collector/examples/healthcheck_fixtures.example.json` — every fixture key.
- `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md` → "Private Collector Health Check
  Protocol" — the passive precondition.
- `playbooks/11_UPDATE_AND_VERSION_WATCH.md` — how an install receives the bridge build that
  carries the healthcheck.
