# Collector Bridge

This is the localhost bridge for the Solo Agency Local Collector.

**The bridge ships as a binary. Its Go source is not in this repository.**

As of 2026-09-06 the source lives in the private repository `soloagency/solo-agency-bridge`,
and this repository's history was rewritten so it is not in any commit here. What remains in
this directory is what a machine needs to RUN the bridge and what an agent needs to USE it:

- `collector_capabilities.json` — the capability catalog every agent reads before planning work
- `lead_industries.json` — the closed industry vocabulary an enrich must choose from

Install a bridge with `../setup_collector.sh`, which downloads the binary for your platform
from the `dist` branch and verifies its SHA256 before running it. Nothing here is built from
source, and no Go toolchain is required.

This document describes the bridge's INTERFACE — its routes, its job contract, its outputs — so
that agents and the Chrome extension can be written against it. It is not a build guide.

## Responsibilities

- Bind to `127.0.0.1` only.
- Serve the current collector job to the Chrome extension.
- Accept structured records from the extension.
- Require a per-run write token for POST endpoints.
- Write JSONL/status/snapshot files locally.
- Support short on-demand runs and persistent scheduler runs.
- In persistent scheduler mode, expose `/config`, return jobs only inside configured collection windows, and stay online after `/complete`.
- Reload `collector_config.json` during `/status` when the file timestamp or size changes.
- Queue run-now jobs from `POST /jobs/run_now` and from per-client files under `daily-content-pipeline/collector/jobs/pending/`.
- Treat `run_now_request.json` as a legacy/batch shim for sandboxed agents that cannot call localhost HTTP directly; convert it to queued jobs during `/status`.
- Maintain separate active job state, counters, output dirs, and completion state per `client_slug`, bound to the claiming extension instance when present, so different client Chrome profiles can collect in parallel through one shared bridge.
- Record extension check-ins from Chrome extension `/status` calls.
- Expose bridge and extension health through `/status` and `bridge_health.json`.

## Persistent Scheduler Mode

Use this for unattended schedules:

Generated setup/start scripts should be idempotent restarters. Before starting a new bridge, they should call `POST /shutdown` when available, stop the PID in `collector.pid`, then inspect port `17321` and stop only old `collector-bridge` processes. If a non-collector process owns the port, report the blocker instead of killing unrelated software.

```sh
collector-bridge \
  --host 127.0.0.1 \
  --port 17321 \
  --config-file daily-content-pipeline/collector/collector_config.json \
  --output-dir daily-content-pipeline/collector/inbox \
  --persistent
```

## Solo Agency plan (entitlement.go)

The client's WideCast API key doubles as the Solo Agency license. The bridge exchanges it for an Ed25519-signed entitlement token (`GET /v1/solo/entitlement` on `widecast.ai`, at start and about every 24 hours; only the key, a random install id and the bridge version are sent), caches the RAW token at `daily-content-pipeline/collector/inbox/entitlement.json` (0600) and re-verifies its signature on every read — editing the file, faking the server or patching the wire cannot grant a plan. The tier follows the account's WideCast plan — Free · Starter $49 · Pro $99 · Business $199 · Enterprise — as `features` + `limits` in the token (clients 1/5/10/20/∞, watched sources per client 1/5/15/30/∞, campaigns per client 1/3/10/20/∞, sends/day 20/150/400/1000/∞; enrich + write actions from Starter, harvest + Zillow from Pro). No key, or a server unreachable for more than 14 days past the token's expiry, means Free.

- `GET /status` now carries `version`, `install_id` and an `entitlement` block (`tier`, `features`, `limits`, `source`, `stale`, `mode`, `upgrade_url`); `bridge_health.json` mirrors it. `GET /jobs/current` adds `entitlement_token`, `entitlement_tier`, `entitlement_mode` and `bridge_version` to `collector_bridge`, and `job.entitlement` lists any `cut_sources`.
- `--entitlement-mode enforce|log` (default `enforce`; `log` records every decision in `collector/logs/entitlement.jsonl` and refuses nothing — the escape hatch). CLI tools (`tool crm-store`, `tool gmail`) read the same cache and enforce too; `SOLO_ENTITLEMENT_MODE=log` switches them.
- `--version` prints the build version stamped by the deploy script.
- `<bridge> tool entitlement status|refresh --pipeline daily-content-pipeline` shows the verified plan, the cache file, `last_error`, and how many keys were found; `refresh` forces a fetch.
- Paid capabilities are marked `tier: pro` + `feature` (enrich / write_actions / harvest / zillow) in `collector_capabilities.json`; a capability runs when the token's `features` lists its feature. The bridge reads the catalog compiled into the binary only, and the extension re-verifies the same token (`solo_entitlement.js`) before running them.

## On-Demand Run

Ask a RUNNING bridge for work with `POST /jobs/run_now` rather than starting one with flags:

```sh
curl -s -X POST http://127.0.0.1:17321/jobs/run_now \
  -H 'Content-Type: application/json' \
  -d @../examples/job.sample.json
```

Stop a run that is already collecting with `POST /jobs/cancel` (`{"run_id":"..."}`,
`{"client_slug":"..."}` or `{"all":true}`); queued jobs are withdrawn at once and a run in
flight stops at its next source boundary.


## Health

Agents can call:

```text
GET http://127.0.0.1:17321/status
```

Each `/status` call also synchronizes local control files:

- reloads `collector_config.json` if it changed;
- converts `daily-content-pipeline/collector/run_now_request.json` into queued run-now jobs if present;
- writes `run_now_request_status.json`;
- moves consumed run-now request files aside as `run_now_request.{run_id}.{timestamp}.consumed.json`;
- remembers the processed file signature in memory as a replay guard if moving/removing fails.
- checks `jobs/pending/` and exposes the next matching queued job only to the extension identity for that client; different client identities can be active at the same time, while jobs for the same client/profile remain sequential.

Agents can override parallel source tabs per run by setting
`pacing.source_concurrency` (or top-level `source_concurrency`) in the
run-now payload. The bridge clamps it to `1..3`; omission keeps the default `1`.

```json
{
  "run_id": "manual_deep_scan",
  "sources": [{ "name": "Example group", "url": "https://www.facebook.com/groups/example" }],
  "pacing": {
    "source_concurrency": 3,
    "scroll_steps": 5
  }
}
```

The response includes `extension_health`:

- `status`: `recent`, `stale`, or `no_extension_check_yet`
- `last_extension_check_at`
- `seconds_since_last_check`
- `extension_check_count`
- `possible_missing_reasons`

The bridge also writes:

```text
daily-content-pipeline/collector/inbox/bridge_health.json
```

## Healthcheck probes

`/status` is liveness. `POST /healthcheck/run` (and `tool healthcheck run --client <slug>`) runs
every catalog capability against the operator's fixtures and scores the records; results are
under `<collector dir>/healthcheck/` and on `/ui/status`. Every catalog entry must carry a
`healthcheck` block (`TestCatalogHealthcheckContract`). Details: `../HEALTHCHECK.md`.

Also in this build: `--capability-validation=warn|reject|off` (unknown capability ids on job
intake; default warn) and 24h retention for `logs/extension_health.jsonl` and
`logs/job_routing.jsonl` (`COLLECTOR_EVENT_LOG_RETENTION_HOURS`).

## Build

Not from here. The source is in `soloagency/solo-agency-bridge` (private); binaries are
published to the `dist` branch of this repository as `collector-bridge-binaries-<version>.zip`
alongside `SHA256SUMS`, and `../setup_collector.sh` is what consumes them.

