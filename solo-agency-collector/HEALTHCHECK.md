# Healthcheck — probing every capability on purpose

Facebook and Zillow change their GraphQL and DOM without notice. Until now the
first sign of a broken capability was an empty harvest weeks later. The
healthcheck re-runs every catalog capability against fixtures the operator
owns, scores the records the collector already returns, and says which
capability broke, why, and what to open first.

It is **not** the passive `GET /status` check in `AGENT_RUNBOOK.md` ("Health
Check" there means "is the bridge/extension alive"). This document is about
active probes.

Plan gating never touches the healthcheck: probes the bridge enqueues itself (`source: healthcheck`) bypass every plan gate, and the dev executor workspace (aven-ngo) is not a CRM client, so a Free install can always run its diagnostics. If a probe record carries `solo_entitlement_required`, that is the extension's own plan check (enforced since launch: the capability's feature is not in this install's plan) — record it in the report, do not count the probe as a collector failure.

## 1. What runs where

| Piece | Where | Notes |
|---|---|---|
| Probe definitions | `bridge-go/collector_capabilities.json` → each capability's `healthcheck` block, plus `healthcheck_extra_probes` for the no-capability path | **Mandatory.** `TestCatalogHealthcheckContract` fails the build when a capability has no valid block. Adding a capability = adding its probe. |
| Runner | `bridge-go/healthcheck.go` (plan, gate, enqueue, wait, score, report, alert), `healthcheck_assert.go` (assertion DSL), `healthcheck_http.go` (routes), `healthcheck_cli.go` (`tool healthcheck`) | Runs **inside the bridge**: it enqueues probe jobs the way the harvest daemon does and reads the run output from disk. |
| Runtime state | `<install>/daily-content-pipeline/collector/healthcheck/` | `fixtures.json` (operator-owned targets), `baseline.json`, `state.json`, `latest.json`, `reports/<run>.json`, `history.jsonl`, `alerts.jsonl`. Never in the repo. |
| Probe run output | `<inbox>/<month>/healthcheck/harvest/<run_id>/` | Same files as any run (`collector_status.json`, `private_data_points.jsonl`, `source_status.jsonl`). Probe jobs carry `harvest_owner: healthcheck`, so they never land in a client tree and never touch `run_now_request_status.json`. |
| Panel | `/ui/status` → "Healthcheck probes" | Last run, verdict per probe, action list. |

## 2. Running it

The bridge must be running (persistent mode, as installed). The CLI only talks
HTTP to it.

```bash
# who could execute? (every extension that has checked in, with version and freshness)
collector-bridge tool healthcheck executors

# which fixture keys are still missing? (the "ask the operator" list)
collector-bridge tool healthcheck fixtures

# dry plan: order, effective mode, missing fixtures per probe
collector-bridge tool healthcheck plan --client aven-ngo

# run the daily suite; without --client the CLI lists executors and asks (TTY) or exits 2 with the list (agent)
collector-bridge tool healthcheck run --client aven-ngo

# a few probes only (a chained probe pulls its parent automatically)
collector-bridge tool healthcheck run --client aven-ngo --only fb.group.posts,fb.post.comments

# weekly suite, real writes into the operator's own assets, agent-composed text
collector-bridge tool healthcheck run --client aven-ngo --suite weekly --allow-writes --text "healthcheck 2026-09-02 xin chào"

# last report / one report / two weeks of history
collector-bridge tool healthcheck report
collector-bridge tool healthcheck report --run hc_20260902_083000
collector-bridge tool healthcheck history --days 14
```

Exit codes: `0` no FAIL, `1` overall FAIL, `2` usage or no executor chosen.
`--json` prints the report as JSON; the human form ends with an
`**[ACTION REQUIRED]**` block an agent can relay verbatim.

HTTP, for agents and scheduled tasks (local only, no token, like `/jobs/run_now`):

```text
GET  /healthcheck/plan?client=aven-ngo&suite=daily&only=a,b
POST /healthcheck/run      {"client_slug":"aven-ngo","suite":"daily","only":[],"allow_writes":false,"text":"...","no_alert":false}
GET  /healthcheck/status?run_id=hc_...      live progress, then the finished report
GET  /healthcheck/latest | /healthcheck/report?run_id=... | /healthcheck/history?days=14 | /healthcheck/fixtures
```

One run at a time; a second `POST /healthcheck/run` answers 409.

### Choosing the executor

Nothing is ever the default. The operator picks the client whose extension
runs the probes (`--client`, optionally `--instance` when a client has several).
Production clients should only ever run read-only suites; write probes belong
on the operator's own test client (`aven-ngo` today).

### Modes and writes

| Mode in catalog | Default run | With `--allow-writes` | With `--dry-run-only` |
|---|---|---|---|
| `read` | the read | same | same |
| `dry_run` (comment, DM, group post) | `inputs` with `dry_run:true`, asserts the composer/button chain | `write_inputs` + `write_assert`: a real write into the operator's asset | dry run |
| `write` (react) | the real action — idempotent: `done` first, `already` after | same | dry run |
| `negative` (`fb.profile.about`) | asserts the id still answers "not built" | same | same |
| `excluded` | never runs; `reason` is shown | | |

Probe text for writes comes from `--text` (the agent composes it); without it
the runner writes "Solo Agency healthcheck probe <run id> (<time>)". No delete,
unsend or unreact capability exists, so real comments/posts/DMs stay until the
operator removes them by hand.

## 3. Fixtures

`<collector dir>/healthcheck/fixtures.json` is a flat object `key: "value"`.
`examples/healthcheck_fixtures.example.json` lists every key; `tool healthcheck
fixtures` shows which are missing. A probe whose keys are missing is SKIPPED
and the plan/report names the keys, so the operator is asked for exactly those.

Fixtures must be the operator's own targets (test group, test account, a post
the test account published, a consenting DM recipient). With operator-owned
fixtures "the page was genuinely empty" cannot happen, which is what turns
`count: 0` into a trustworthy failure.

Values chained from an earlier probe are not fixtures: `fb.post.comments` takes
`feedback_id`/`post_url` from the most commented post `fb.group.posts` just
read (selected by `engagement.comments` — the field was `engagement.comment_count` until
2026-09-09, which no record actually carried, so the probe was SKIPPED on every run);
`fb.group.search_posts` searches the first words of the newest post.

`fb_search_group_url` / `fb_search_group_keyword` (added for `fb.group.search_posts`) must be a
public read-only group — never the write-allowed test group used by the comment/react/DM probes.

Instagram (second platform module, `INSTAGRAM_CAPABILITIES.md`) adds three fixture keys:
`ig_canary_profile_url` (ROOT url of an Instagram business/creator profile the operator may
read — read only), `ig_canary_profile_username` (the `<username>` part of that url), and
`ig_search_keyword` (evergreen keyword for `ig.search.posts` and `ig.people.search`, e.g.
"realtor"). `ig.post.comments` needs no fixture of its own — it chains off `ig.profile.posts`'s
most-commented item, same pattern as `fb.post.comments` above.

## 4. How a probe is judged

Four tiers, lower tiers gating the higher ones:

- **L0 gate** — the executor's extension is `recent`; it runs the newest build
  seen; the served catalog equals the embedded one (a stale copy next to
  `collector_config.json` is exactly the drift found on 2026-09-02); no
  orphaned job in `jobs/pending/`; `extension_health.jsonl` is not oversized.
  A failed executor gate makes every probe SKIPPED, not FAIL.
- **L1 platform** — `expect_query`: the persisted query the capability reads
  must appear in `graphql_manifest[].queryName` of the fixture page, or in the
  record's own `records.source_query` (`background.js` re-reads the generic GraphQL
  manifest after capability dispatch too, so a query the capability itself triggered
  counts). Facebook renaming a query trips this before the extractor returns zero.
  Guards: `landed_on_self` (FAIL), `url_drifted` and `maybe_logged_out` (WARN).
- **L2 capability** — the `assert` list: `{path, op, value?, severity?, note?}`
  over `records.…` and `dp.…`. Ops: exists, absent, eq, neq, gte, gt, lte, lt,
  nonempty, empty, in, not_in, starts_with, contains, matches, not_matches,
  is_true, is_false, len_gte, len_eq, eq_fixture, contains_fixture,
  each_has_any, any_has, each_eq. `severity: warn` turns a miss into WARN.
- **L3 baseline** — fill rate of `baseline_fields` across `records.items`
  versus the last PASS: a field that was ≥80% filled and is now 0% while
  `count` holds is a field-path drift (WARN; the incident count makes it a
  FAIL by repetition). Duration more than double the baseline is noted.

Verdicts: **PASS**, **WARN**, **FAIL**, **BLOCKED** (Zillow Press & Hold within
its 5-minute probe ceiling, human gate not solved, `not_a_member`,
`waiting_for_human_captcha`), **SKIPPED** (fixture missing, executor stale,
dependency unmet, job never claimed within 90 s), **EXCLUDED**. BLOCKED and
SKIPPED never count as FAIL; three BLOCKED runs in a row raise an alert.

`cause_group` folds several FAILs into one incident (all three search
capabilities read the same `SearchComet` query, for instance). Each FAIL
carries a `repair_hint` pointing at the maintenance procedure to follow
(`GRAPHQL_MAINTENANCE.md §7`, `HANDOFF_WRITE_ACTIONS.md`,
`ZILLOW_CAPABILITIES.md §7`) and the evidence needed for it: GraphQL query
names seen, `pagination_*`, `registry_probe`, `head_page_via`,
`seen_textboxes[]`, Zillow `status/source`.

## 5. Alerts

The report and `/ui/status` always show the latest run. Email goes to
`operator_email` (system settings) through the first healthy outreach sendbox
**only when state changes**: a probe newly FAILs, recovers from FAIL, or has
been BLOCKED three runs running. `--no-alert` silences it; a missing mail setup
is recorded in the report, never fatal. Scheduling is deliberately manual
today: when the operator wants a daily run, register a scheduled task that
calls `tool healthcheck run --client <slug>` and records ownership in
`automation_manifest.md`.

## 6. Adding a capability

1. Implement it in the extension as before (for a brand-new platform: a `platforms/<name>/`
   extractor + normalizer plus one entry in `core/platform_registry.js` — see
   `GRAPHQL_MAINTENANCE.md` §8).
2. Add its catalog entry **with a `healthcheck` block**: mode, cadence, url
   template, inputs, fixtures it needs (add new keys to `hcFixtureDocs` in
   `healthcheck.go` and to `examples/healthcheck_fixtures.example.json`),
   `expect_query` for GraphQL capabilities, assertions that only a working
   extractor can satisfy, `baseline_fields`, `cause_group`.
3. `go test ./...` — `TestCatalogHealthcheckContract` tells you what is missing.
4. Run `tool healthcheck run --client <test client> --only <id>` and keep the
   first PASS as the baseline.

## 7. Two other things shipped with the healthcheck

- **Capability id validation on job intake.** `enqueueRunNowJob` checks every
  source's `capability` against the catalog (embedded ∪ served). Default
  `--capability-validation=warn`: the job still queues, a `job_routing`
  event `unknown_capability` is written and the bridge log says which id.
  `reject` refuses the job; `off` disables. Internal `_discover.*` /
  `_diag.*` ids are always allowed. Plan: warn for two weeks from 2026-09-02,
  then start the bridge with `reject`.
- **Event-log retention.** `extension_health.jsonl` and `job_routing.jsonl`
  are heartbeats and grew without bound (391 MB on the live install). The
  bridge now keeps 24 hours and compacts each file at most once an hour on
  write (temp file + rename). `COLLECTOR_EVENT_LOG_RETENTION_HOURS` overrides
  the window; `0` disables pruning.
