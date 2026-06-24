# Collector Bridge

This is the localhost bridge for the Solo Agency Local Collector.

It is written in Go so maintainers can build small single-file binaries for macOS, Windows, and Linux.

End users do not need Go when prebuilt binaries are shipped.

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

## On-Demand Run

```sh
go run . \
  --host 127.0.0.1 \
  --port 17321 \
  --run-id 2026-06-20_demo-client \
  --job-file ../examples/job.sample.json \
  --output-dir ../../daily-content-pipeline/collector/inbox/2026-06/2026-06-20_demo-client \
  --ttl-minutes 30
```

On-demand runs auto-shutdown on completion or TTL.

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

## Build

```sh
go build -o ../../solo-agency-local-collector/bin/collector-bridge ./...
```

For releases, cross-compile and publish OS-specific binaries. The AI agent should run those binaries directly instead of asking the user to install Go.
