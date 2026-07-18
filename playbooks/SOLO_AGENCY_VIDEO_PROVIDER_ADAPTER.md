# Solo Agency Video Provider Adapter

Stage: `03A`

## Load Rule

Load this file after any vendored writing, video-editing, or production skill whenever the agent may create a video, edit/review provider video scenes, estimate production credits, upload production media, poll production status, render/export, publish produced content, or use any provider account action for a Solo Agency client.

This adapter is Solo Agency policy. It intentionally lives outside vendored skills such as `playbooks/skills/video-script-writing/SKILL.md` and `playbooks/skills/video-editing/SKILL.md`, because those skills may be refreshed from upstream. If a vendored skill tells the agent to call a concrete MCP tool such as `widecast_create_video`, `widecast_account`, `widecast_upload_asset`, `widecast_video_data`, `widecast_modify_scene`, or `widecast_publish`, treat that instruction as an abstract capability request and resolve it through this adapter first.

## Hard Gates

- Do not edit vendored provider skills to add Solo Agency client-routing policy. Put overrides in this adapter and load it after the vendored skill.
- Every tool/capability check must check Client tools first and global MCP/native tools second. Client tools are the current client's provider config, OpenAPI cache/spec, verified account identity, `provider_capabilities.json`, provider health, and redacted provider logs.
- Do not use a global MCP/native provider account as the current client's account unless its identity is proven to match the saved client provider identity.
- Do not estimate credits, create video, edit scenes, upload media, render/export, publish, notify, or poll account data from an account-level provider until the current client's provider config and OpenAPI capabilities are verified.
- Do not call `production.create_video`, `widecast_create_video`, or any equivalent provider video operation with a report script, Markdown source record, prior draft, or content-history script pasted through unchanged. A final script/brief produced by loading and applying the existing WideCast video script-writing skill is required first.
- If provider config, auth, discovery, account identity, or a required operation is missing, stop the provider action and log the exact blocker. Continue with draft/report work when possible, but do not create video media locally.
- Never replace missing client-scoped video capability with a local/system video renderer. `ffmpeg`, Pillow, `moviepy`, browser screenshots/canvas, Remotion, slideshow export, MP4/MOV/GIF generation, or similar tools are forbidden as fallback video production paths.

## Tool Availability Check Rule

When the human or another agent asks whether the system has tools for video, video scene editing, blog, social posts, media upload, render/export, publishing, notification, analytics, account credits, connected platforms, or WideCast itself, do not start from the current chat's MCP tool list.

Check in this order:

1. Current client's provider config and auth value.
2. Current client's OpenAPI discovery/cache and verified provider account identity.
3. Current client's `provider_capabilities.json` or freshly discovered operation list.
4. Required capability group and operation schema.
5. Global MCP/native tools only as optional compatibility after identity match is proven.

If Client tools expose the required operation and global MCP does not, report that the client tool exists and use the Client tools path. If Client tools are missing or stale, refresh discovery or log the exact blocker before saying a capability is unavailable.

## No Local Video Substitute

This adapter only resolves provider-backed video capabilities. If `production.create_video`, `video_editing.*`, or `production.export_video` cannot be resolved from the verified current client's provider config/OpenAPI capability cache, the correct outcome is a provider blocker plus a PDNA setup request.

The agent may create script text, storyboards, shot lists, visual notes, or production briefs. It must not create a local video file, rough cut, animated slideshow, preview MP4, or final export with non-provider tools. Missing WideCast/client provider setup is a video-production blocker, not permission to improvise with local rendering.

If the current Automation Flow can safely update the client's provider config, ask for the client's WideCast API key by default in that same session, save it as `api_key_env` or `api_key_local`, fetch/cache OpenAPI, verify account identity, refresh `provider_capabilities.json`, update provider health, and resync automation before any later video action. Do not ask provider/scope/spend/publish/account-identity questions for the default path. If the human explicitly chose a non-WideCast provider, use that provider's API key instead. If the session cannot update provider config, hand off to setup/maintenance with the exact PDNA setup action.

## Final WideCast Script Skill Requirement

This adapter may resolve a video creation operation only after the current run has a final WideCast-grade script or production brief created by the WideCast video script-writing skill.

Required evidence before create:

1. The agent loaded the existing `playbooks/skills/video-script-writing/SKILL.md` through the verified client provider `getWritingSkill(format=video)` operation or through the repo-local/static fallback.
2. The final script/brief was produced by applying that skill to the selected idea/report draft, not copied from the client-facing report and not hand-rolled from agent memory.
3. If a report version/code, pasted edited version, or automation recommended/approved version existed, the final script/brief follows only that selected version/code; the agent did not generate a second five-version set during production.
4. If no selected/recommended version existed, the run records that the WideCast skill's Stage 1 options were produced and a pick/recommendation was obtained before provider creation.
5. The final script/brief records research bullets and the selected format/code when available.
6. For visual-dependent videos, the final script includes vetted direct image/video URLs in markdown image syntax, or a `media_pool`/visual blocker entry when direct URLs could not be verified. URLs must not be fabricated.
7. The run state records approval status: manual/interactive confirmation, saved scheduled-run video-creation approval, or `approval_required`.

If this evidence is missing, do not resolve or call the provider video operation. Load Stage 3's Final WideCast Script Skill Gate and produce the final script first by applying the existing WideCast skill. If PDNA is missing, the skill may still run through the account-free fallback, but the adapter must stop before provider video creation.

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
| local renderer phrases such as `ffmpeg`, Pillow, `moviepy`, Remotion, canvas video, slideshow video, or local MP4 | No capability mapping. These are forbidden fallbacks when provider video capability is missing. |
| `widecast_get_editing_skill`, video-editing skill | `production.video_editing_skill` / `video_editing.get_editing_skill` from the current client's verified provider capabilities |
| `widecast_upload_asset`, media upload, file upload | `media.upload_asset` from the current client's verified provider capabilities |
| `widecast_wait_for_video`, `widecast_get_status`, scene/status polling | `production.get_status` from the current client's verified provider capabilities |
| `widecast_video_data`, scene data | `production.get_video_data` / `video_editing.get_video_data` from the current client's verified provider capabilities |
| `widecast_scene_geometry`, scene geometry | `video_editing.scene_geometry` from the current client's verified provider capabilities |
| `widecast_scene_inspector`, scene screenshots/inspection | `video_editing.scene_inspector` from the current client's verified provider capabilities |
| `widecast_modify_scene`, scene edits | `video_editing.modify_scene` from the current client's verified provider capabilities |
| `widecast_search_broll`, B-roll/media search | `media.search_broll` from the current client's verified provider capabilities |
| `widecast_create_image`, generated scene image | `media.create_image` from the current client's verified provider capabilities, with explicit cost/credit approval before paid generation |
| `widecast_export_video`, render/export | `production.export_video` from the current client's verified provider capabilities, with a fresh human approval gate |
| `widecast_publish`, post to platforms | `distribution.publish` from the current client's verified provider capabilities, with exact content and platform approval |
| `sendNotification`, Telegram/email fallback | `notification.send` from the current client's verified provider capabilities |
| analytics/dashboard/list videos | `analytics.read` from the current client's verified provider capabilities |

For WideCast, the default discovery URL is `https://widecast.ai/openapi.yaml`, but the agent must still read the current client's provider config first and must not infer availability from a global MCP connection.

WideCast server selection rule:

- Current production API server: `https://widecast.ai/app/dashboard`.
- Disabled/planned vanity host: `https://api.widecast.ai`.
- If the OpenAPI `servers` list includes both, select `https://widecast.ai/app/dashboard` and skip `https://api.widecast.ai`.
- Do not try `https://api.widecast.ai` first and do not fall back to it after a dashboard-path failure unless a future Solo Agency playbook explicitly removes it from `disabled_server_urls`.
- If an older `provider_config.local.json`, `provider_capabilities.json`, or `provider_openapi_cache.yaml` selected `https://api.widecast.ai`, refresh discovery/capabilities, update the selected server, and log `provider_server_selection_corrected`.

## Override For Vendored WideCast Skill Text

If a vendored WideCast writing skill says:

- "call `widecast_account`" -> read credits/account status through the current client's verified provider config/OpenAPI path.
- "call `widecast_create_video`" -> first verify the Final WideCast Script Skill Requirement, then call the resolved `production.create_video` operation for the current client.
- "call `widecast_upload_asset`" -> call the resolved `media.upload_asset` operation for the current client.
- "call `widecast_video_data`, `widecast_scene_geometry`, `widecast_scene_inspector`, or `widecast_modify_scene`" -> call the resolved `video_editing` operation for the current client.
- "call `widecast_get_editing_skill`" -> load the resolved client-scoped editing skill when available, otherwise use the repo-local `playbooks/skills/video-editing/` fallback and log the provider skill blocker.
- "open MCP review URL / use MCP status" -> use the review/status URL returned by the verified client provider operation, or mark the exact blocker.

Do not silently fall back to the current chat's MCP/global account when the client provider config is missing or unverifiable.
Do not silently fall back to local/system video generation when the client provider config or required video operation is missing.

## Video Creation To Scene Editing Chain

When a client-scoped `production.create_video` call succeeds, treat the returned provider topic/video ID and `review_url` as the start of the scene-editing stage, not as the finished video.

Required post-create sequence:

1. Record the provider result in the client's internal report/history: topic/video ID, review URL, operation ID, production mode, approval source, final WideCast script artifact/path, selected format/code, inline image URLs or media pool, and script/version used.
2. Resolve `video_editing` operations from Client tools before using any global MCP/native edit tool:
   - `getEditingSkill`;
   - `getVideoData`;
   - `sceneGeometry` or `getSceneGeometry`;
   - `sceneInspector`, `inspectScene`, or equivalent screenshot/inspection operation;
   - `modifyScene`;
   - media helpers such as `searchBroll`, `createImage`, and `uploadAsset` when needed.
3. Load the video-editing skill from the verified client provider when `getEditingSkill` is available. If it is not available but the repo-local skill exists, load `playbooks/skills/video-editing/SKILL.md` and its required modules from disk.
4. Follow the editing skill's module load map and visual evidence rules. Pull `getVideoData` first, work by stable scene UID/`voice_file` when available, use scene geometry for coordinates, use scene inspector screenshots for visual judgment, and confirm modifications by re-pulling scene data.
5. Do not ask the human to choose scene-by-scene options during the autonomous editing pass. The editing skill decides and fixes, unless a required human asset/action is missing.
6. Paid provider actions inside editing, such as generated images, require explicit cost/credit approval. Free scene mutations such as `modifyScene` may run as part of the approved video-production branch.
7. After the editing skill's pre-summary completion scan passes, ask the human whether to render/export the final MP4 or review scenes first. Rendering/export is a separate approval gate and must call `production.export_video` only after a fresh explicit yes.

If any required edit operation is missing, log the exact blocker from this adapter's blocker list, keep the review URL available, and ask the human whether to review manually or connect/repair the provider capability. Do not claim the final MP4 is ready.

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
- `provider_setup_required_for_video`
- `final_widecast_script_missing`
- `inline_media_vetting_blocked`
- `global_mcp_available_but_not_authoritative`
- `global_mcp_not_client_scoped`
- `local_diy_video_fallback_forbidden`
- `provider_call_failed`
- `video_editing_skill_missing`
- `scene_editing_operation_missing`
- `human_approval_missing`

When blocked, tell the human what was not done, why, and the exact next action. Do not spend credits, render, publish, create local video media, or imply the provider action succeeded.

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
- final WideCast script skill status, artifact/path, selected format/code, and inline-media/media-pool status;
- review URL/status URL/output URL if returned;
- for created videos, topic/video ID, scene-editing status, edit operations used or blocked, and final render/export approval status;
- local logs updated under `integrations/providers/`.
