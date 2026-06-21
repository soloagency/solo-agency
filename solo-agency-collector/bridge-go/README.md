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
- Load `run_now_request.json` during `/status` as a file-based fallback for sandboxed agents that cannot call localhost HTTP directly.
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
- loads `daily-content-pipeline/collector/run_now_request.json` as a run-now job if present;
- writes `run_now_request_status.json`;
- moves consumed run-now request files aside as `run_now_request.{run_id}.{timestamp}.consumed.json`;
- remembers the processed file signature in memory as a replay guard if moving/removing fails.

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
go build -o ../../daily-content-pipeline/collector/bin/collector-bridge ./...
```

For releases, cross-compile and publish OS-specific binaries. The AI agent should run those binaries directly instead of asking the user to install Go.
