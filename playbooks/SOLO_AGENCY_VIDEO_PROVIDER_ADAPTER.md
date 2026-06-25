# Solo Agency Video Provider Adapter

Stage: `03A`

## Load Rule

Load this file after any vendored writing or production skill whenever the agent may create a video, estimate production credits, upload production media, poll production status, publish produced content, or use any provider account action for a Solo Agency client.

This adapter is Solo Agency policy. It intentionally lives outside vendored skills such as `playbooks/skills/video-script-writing/SKILL.md`, because those skills may be refreshed from upstream. If a vendored skill tells the agent to call a concrete MCP tool such as `widecast_create_video`, `widecast_account`, `widecast_upload_asset`, or `widecast_publish`, treat that instruction as an abstract capability request and resolve it through this adapter first.

## Hard Gates

- Do not edit vendored provider skills to add Solo Agency client-routing policy. Put overrides in this adapter and load it after the vendored skill.
- Every tool/capability check must check Client tools first and global MCP/native tools second. Client tools are the current client's provider config, OpenAPI cache/spec, verified account identity, `provider_capabilities.json`, provider health, and redacted provider logs.
- Do not use a global MCP/native provider account as the current client's account unless its identity is proven to match the saved client provider identity.
- Do not estimate credits, create video, upload media, publish, notify, or poll account data from an account-level provider until the current client's provider config and OpenAPI capabilities are verified.
- If provider config, auth, discovery, account identity, or a required operation is missing, stop the provider action and log the exact blocker. Continue with draft/report work when possible.

## Tool Availability Check Rule

When the human or another agent asks whether the system has tools for video, blog, social posts, media upload, render/export, publishing, notification, analytics, account credits, connected platforms, or WideCast itself, do not start from the current chat's MCP tool list.

Check in this order:

1. Current client's provider config and auth value.
2. Current client's OpenAPI discovery/cache and verified provider account identity.
3. Current client's `provider_capabilities.json` or freshly discovered operation list.
4. Required capability group and operation schema.
5. Global MCP/native tools only as optional compatibility after identity match is proven.

If Client tools expose the required operation and global MCP does not, report that the client tool exists and use the Client tools path. If Client tools are missing or stale, refresh discovery or log the exact blocker before saying a capability is unavailable.

## Client-Scoped Provider Action Resolver

Before any provider action, resolve the target provider through Client tools first in this order:

1. Identify `target_client_slug` from the task prompt, `clients_index.md`, the Client Intelligence Profile, or setup context.
2. Load only that client's Client Intelligence Profile and folder under `daily-content-pipeline/clients/{client_slug}/...`.
3. Read that client's `integrations/providers/provider_config.local.json`.
4. Read `daily-content-pipeline/provider_defaults.json` for provider bootstrap defaults when needed.
5. Fetch or refresh the configured provider OpenAPI spec into `integrations/providers/provider_openapi_cache.yaml`.
6. Read or create `integrations/providers/provider_capabilities.json` from the discovered OpenAPI operation list.
7. Verify account identity through the configured client credential and the provider account operation, such as WideCast `getAccount`.
8. Compare the verified account identity with the saved client provider identity when present.
9. Select the operation by capability group and required schema from Client tools, not by whatever MCP tools happen to be visible in the current AI host.
10. Log the redacted call intent/result to `integrations/providers/provider_calls.jsonl` and update `integrations/providers/provider_health.md`.

Preferred execution path:

1. Use the repo OpenAPI helper when available, such as `tools/provider_openapi.py`.
2. Otherwise use equivalent direct OpenAPI calls with the configured client credential.
3. Use MCP/native tools only as an optional compatibility path after identity is proven to be the same client account.

## Capability Mapping

When a writing or production skill names a provider-specific tool, map it to the client-scoped provider capability:

| Skill/tool phrase | Solo Agency capability to resolve |
| --- | --- |
| `widecast_account`, account balance, credits | `account.verify` and `account.credits` from the current client's provider config/OpenAPI account operation |
| `widecast_create_video`, `create_video` | `production.create_video` from the current client's verified provider capabilities |
| `widecast_upload_asset`, media upload, file upload | `media.upload_asset` from the current client's verified provider capabilities |
| `widecast_wait_for_video`, `widecast_get_status`, scene/status polling | `production.get_status` from the current client's verified provider capabilities |
| `widecast_export_video`, render/export | `production.export_video` from the current client's verified provider capabilities, with a fresh human approval gate |
| `widecast_publish`, post to platforms | `distribution.publish` from the current client's verified provider capabilities, with exact content and platform approval |
| `sendTelegramMessage`, Telegram/email fallback | `notification.send` from the current client's verified provider capabilities |
| analytics/dashboard/list videos | `analytics.read` from the current client's verified provider capabilities |

For WideCast, the default discovery URL is `https://widecast.ai/openapi.yaml`, but the agent must still read the current client's provider config first and must not infer availability from a global MCP connection.

## Override For Vendored WideCast Skill Text

If a vendored WideCast writing skill says:

- "call `widecast_account`" -> read credits/account status through the current client's verified provider config/OpenAPI path.
- "call `widecast_create_video`" -> call the resolved `production.create_video` operation for the current client.
- "call `widecast_upload_asset`" -> call the resolved `media.upload_asset` operation for the current client.
- "open MCP review URL / use MCP status" -> use the review/status URL returned by the verified client provider operation, or mark the exact blocker.

Do not silently fall back to the current chat's MCP/global account when the client provider config is missing or unverifiable.

## Blockers

Use these blocker names consistently:

- `target_client_unknown`
- `provider_config_missing`
- `provider_auth_missing`
- `provider_discovery_failed`
- `provider_account_verify_failed`
- `provider_account_mismatch`
- `provider_capability_cache_missing`
- `provider_required_operation_missing`
- `global_mcp_available_but_not_authoritative`
- `global_mcp_not_client_scoped`
- `provider_call_failed`
- `human_approval_missing`

When blocked, tell the human what was not done, why, and the exact next action. Do not spend credits, render, publish, or imply the provider action succeeded.

## Human-In-The-Loop Rule

Provider setup does not authorize automatic publishing. The agent may create video/blog/social assets and send notifications for human review only after the relevant approval gate. Publishing to external platforms requires a separate approval of the exact content and target platforms. Rendering/exporting a final video requires a fresh explicit approval for that export.

## Required Report Back

After a video/provider action attempt, report:

- `target_client_slug`;
- confirmation that Client tools were checked before global MCP/native tools;
- provider name and config path checked;
- OpenAPI discovery status and operation ID used, or blocker;
- account verification status, redacted identity, and match/mismatch result;
- human approval status;
- review URL/status URL/output URL if returned;
- local logs updated under `integrations/providers/`.
