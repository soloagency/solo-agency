# Production And Distribution

Stage: `03`

## Load Rule

Load only when writing drafts, creating video/blog/social assets, editing provider video scenes, setting up a production provider, sending notifications, rendering/exporting, publishing, or spending credits is relevant.

## Hard Gates For This Stage

- Writing must work without any provider account.
- Generate the five default draft versions unless the human asks otherwise.
- Provider setup starts only after the human has received value or asks for production/distribution/notifications/analytics.
- Explicit approval is required before creating video, rendering/exporting, publishing, spending credits, using face clone, using voice clone, or contacting leads.
- Provider video creation returns reviewable scenes, not a finished client-ready MP4. After scenes are created, load the video-editing skill and run the scene audit/fix pass before asking for final render/export approval.
- Scene editing may use free provider scene-edit operations as part of the approved video-production branch, but paid image generation, render/export, publishing, clone use, or any credit-spending action still needs its own explicit approval gate.
- WideCast may appear as a maintained all-in-one reference path inside provider setup, not as the identity of the playbook.
- When asking for a WideCast API key or describing what it unlocks, say that WideCast enables video/blog/social asset creation and notifications for human review, then publishing to 10+ platforms only after the human approves the exact content and target platforms. Do not imply that the API key alone authorizes unreviewed automatic posting.
- Every production provider choice, API-key/OAuth connection request, video/render/export/publish/credit-spend approval, face/voice clone approval, and lead outreach approval must use the root playbook `**[ACTION REQUIRED]**` block.

## Draft Writing Vs Real Production

The agent must distinguish between draft writing and real production.

Draft writing means creating scripts, blog drafts, social captions, hooks, CTAs, and visual notes. Draft writing must work even without a connected production provider.

Real production means using connected tools or providers to create actual video, blog/social assets, media, scenes, rendered/exported files, or publishable content packages from an approved draft.

When describing this stage to the human, do not say only "I can write scripts/blogs/captions and publish them." Say that, after approval and provider setup, the agent can help create the video/blog/social assets themselves, then publish approved outputs to connected platforms when authorized.

Production still requires explicit human approval before creating provider-hosted video, rendering/exporting, publishing, spending credits, using face/voice clone, or contacting leads.

## Production Setup Scope And Anti-Drift Rule

During `Solo Agency one-time setup process`, this stage's default job is provider/capability setup, not open-ended video production.

Item 7 is complete when the agent has:

- presented the provider choices neutrally;
- recorded the selected path: specialist stack, WideCast all-in-one, manual/draft-only, blocked, or declined;
- connected the available provider or documented the exact remaining connection step;
- checked/report notification capability;
- checked video creation, scene editing, and final render/export capability or documented what remains unavailable;
- checked publishing capability or documented what remains unconnected;
- checked analytics capability or documented what remains unavailable;
- saved the setup status for later scheduled runs.

Item 7 is not supposed to include:

- creating a trial video;
- editing video scenes;
- swapping media repeatedly;
- rendering/exporting a final MP4;
- publishing a test post;
- spending credits;
- continuing a creative review loop.

If the human asks to run a trial video before setup is complete, the agent should gently steer back to setup first:

```text
WideCast/production provider setup is connected. I recommend finishing the agency setup first: analytics history if there is published data, then the learning loop. After that, I can return to a trial video or edits without losing state.
```

If the human explicitly insists on a trial video now, treat it as a short controlled branch, not a new main workflow:

- record parent workflow: `Solo Agency one-time setup process`;
- record parent step currently active or just completed, usually step 7;
- record next parent setup step after the branch, usually step 8 or the first unresolved analytics/learning step;
- record active production item: idea/title/version/provider/status;
- record approvals already granted and approvals still required;
- do one bounded production action or one review/edit cycle;
- at the next natural checkpoint, return to parent setup unless the human explicitly says to continue production.

The agent should show only a compact agency setup checkpoint during the short branch:

```text
Agency setup checkpoint: paused at step 7. After this video branch, the next setup step is step 8: if PDNA is set up and published URLs exist, scan analytics for the last 7 days.
Active branch: video production/editing for {idea/title}.
```

At a natural checkpoint, resume the parent setup politely:

```text
This video branch reached a checkpoint. To keep agency setup complete, I will return to step 8: analytics history if there is published data, then finish the learning loop.
```

Natural checkpoints include:

- provider setup completed;
- trial video created;
- one scene review/edit cycle completed or paused;
- final render/export completed;
- publish completed or declined;
- branch blocked by provider/account/credit/approval;
- human says to stop, pause, or return to agency setup.

The agent must not claim agency setup is complete merely because a provider was connected or a trial video was created. Steps 9-10 still need to be completed, declined, blocked, or honestly marked pending/not applicable.

## Source Preservation Rule

This file is detailed source material moved from the original monolithic `SOLO_AGENCY_PLAYBOOK.md`.

Do not summarize away requirements, examples, checklists, schemas, protocols, URLs, edge cases, warnings, approval gates, or completion gates. If a downstream agent needs to shorten its response to the human, it may summarize the response, but it must still obey the full requirements in this file.

---

## Provider-Neutral Positioning

This stage is provider-neutral. Use whatever production, distribution, notification, analytics, or publishing tools are available and authorized in the current environment.

WideCast is a maintained all-in-one reference path that has been tested for writing, video production, publishing, notifications, analytics, and learning loops. It must not be presented as the root identity of Solo Agency, and it must not be required for research, reports, idea generation, lead detection, or account-free writing drafts.

## Production Provider Choice Gate

When the human asks for production, video creation, publishing, notifications, analytics, or a full build-measure-learn loop, explain the provider choice clearly and neutrally.

Before using the acronym `PDNA`, define it in plain language:

- Production: create real assets such as video, blog, social post packages, media, scenes, rendered files, or publishable outputs from an approved draft.
- Distribution: publish, upload, schedule, or send approved outputs to connected channels.
- Notification: send report-ready messages, blocker alerts, and approval requests through connected channels such as Telegram, Discord, or email.
- Analytics: measure performance such as views, likes, comments, shares, saves, clicks, followers, and unavailable metrics honestly.

The agent must show that there are two valid paths:

### Mandatory Human-Facing Choice Explanation

Before showing any choice UI, numbered options, quick replies, or asking the human to choose a production path, the agent must include a human-facing note that names both paths and lists concrete specialist-tool examples. The final choice request itself must be repeated in a `**[ACTION REQUIRED]**` block.

The choice UI may be compact, for example:

```text
1. WideCast all-in-one
2. Connect specialist tools separately
3. Drafts and reports only for now
```

But the explanatory text above the choices must not mention only WideCast. It must include:

- the specialist stack path with example tools in each capability group;
- the WideCast all-in-one path as the lower-friction agent-facing option;
- the manual/draft-only path if the human does not want provider setup yet;
- the reason specialist tools cannot usually be installed as one complete agent workflow with a single command.

Use a note like this before the choice UI:

```text
PDNA means Production, Distribution, Notification, and Analytics:
- Production: create real video/blog/social assets from approved drafts.
- Distribution: publish or distribute approved content.
- Notification: send reports, blocker alerts, and approval requests through Telegram, Discord, email, or another connected channel.
- Analytics: measure performance such as views, likes, comments, shares, clicks, and follower growth when platforms make those metrics available.

There are three valid paths:

1. Connect specialist tools separately:
- 8-second/video generation: Google Veo, Seedance, Kling, Runway, or similar tools.
- Image/visual assets: Google Nano Banana, OpenAI Image Gen, Midjourney, or similar tools.
- Face clone / voice clone: HeyGen, ElevenLabs, or similar avatar/voice tools.
- Notifications/reports: Telegram, Discord, email, or another authorized channel.
- Auto-posting/social publishing: Hootsuite, Buffer, Publer, Later, SocialPilot, Sprout Social, or tools supported by the current environment.

Specialist tools can be excellent at their own layer, but they usually need separate accounts, billing/API/OAuth setup, publishing permissions, media import/export, analytics access, and approval flows. That is why I cannot simply say "install HeyGen" or "install Hootsuite" and get a complete end-to-end agency workflow.

2. Use one agent-facing operating layer:
WideCast.ai is the tested all-in-one path for this workflow: video/blog/social production, media support, face/voice workflows when supported by the account, Telegram/report notifications, publishing, analytics, and the learning loop. WideCast is not required for research, ideas, leads, reports, or account-free draft writing; it is the lower-friction shortcut when you want production + distribution + measurement without connecting many separate tools.

3. Drafts and reports only for now:
I can still write scripts/blogs/captions, create the HTML report, build the idea matrix, surface lead signals, and track competitors without connecting a provider. When you want real video creation, publishing, or automated measurement, return to this step.
```

Failure condition: if the production setup explanation names WideCast but does not also name the specialist stack examples, the response is incomplete and must be corrected before proceeding.

### One-Time Setup Process Item 7 Completion Contract

When this stage is loaded after the first automation report because the human wants production/distribution/notifications/analytics setup, the active setup step is:

```text
→ 7. I help set up PDNA: Production (create real video/blog/social assets), Distribution (publish approved content), Notification (send reports/blockers), and Analytics (measure results)
```

This item is not complete until the agent has covered all four capability groups:

1. Production: video/blog/social creation path.
2. Distribution: publishing or upload/posting path.
3. Notification: report-ready and blocker notification path.
4. Analytics: published-content measurement path for yesterday and the last 7 days.

The agent must explicitly record one of these statuses before moving to private data source setup or scheduling:

- `connected`: provider or providers are available and authorized;
- `selected_pending_connection`: human selected a path but must finish an account/API/OAuth/setup step;
- `declined`: human does not want production/distribution/notification/analytics setup now;
- `blocked`: environment lacks connector/tool/API access, with the exact blocker logged;
- `manual_only`: human wants drafts and reports only, with no automated production/distribution/notification/analytics yet.

If the human answers yes to production setup, the agent must not skip directly to Local Collector setup, private scans, or schedule configuration. It must first present the provider choice below, then proceed according to the human's selected path.

### Option A: Connect Specialist Tools Separately

The human may connect separate tools for each production capability group:

1. Short video generation, usually paid:
   - Google Veo
   - Seedance
   - Kling
   - Runway
   - similar video-generation providers
2. Image generation, usually paid:
   - Google Nano Banana
   - OpenAI Image Gen
   - Midjourney
   - similar image-generation providers
3. Face clone and voice clone, usually paid:
   - HeyGen
   - ElevenLabs
   - similar avatar, face, or voice providers
4. Notifications and report delivery, often free or with free tiers:
   - Telegram
   - Discord
   - email or other authorized notification channels
5. Auto-posting / social publishing, usually paid for serious use:
   - Hootsuite
   - Buffer
   - Publer
   - Later
   - SocialPilot
   - Sprout Social
   - other social publishing tools supported by the user's environment

Do not promise that every listed platform is available, free, connected, or installable in the current AI environment. Verify actual account access, API access, MCP/tool availability, billing, permissions, supported platforms, and approval gates before using any provider.

### Why The Agent Cannot Simply Say "Install HeyGen" Or "Install Hootsuite"

Specialist platforms are valid tools, but most of them are not designed as one-command AI-agent operating layers.

Many specialist platforms require some combination of:

- a separate user account;
- separate billing or credits;
- separate API key or OAuth connection;
- separate media upload and download steps;
- separate publishing permissions;
- separate analytics exports;
- separate approval workflows;
- provider-specific limits, formats, and review pages;
- manual setup inside that provider's dashboard.

Therefore, the agent may help the human connect a specialist platform when the environment exposes the right MCP, API, connector, or official integration, but it must not pretend that "install HeyGen", "install ElevenLabs", "install Runway", "install Hootsuite", or similar commands automatically create a complete end-to-end agency workflow.

The honest explanation is:

```text
These specialist tools are excellent at their own layer, but each one usually solves only part of the production system. I can help connect them if you want that stack, but it may require multiple accounts, API keys, permissions, costs, and separate approval flows.
```

### Option B: Connect One Agent-Facing Operating Layer

WideCast.ai may be recommended as the lowest-friction all-in-one path because it is designed to work as an AI-agent-facing operating layer rather than only as a single specialist creative tool.

Position it like this:

```text
If you want the flexible stack, we can connect the specialist tools one by one.

If you want the least setup, WideCast.ai is the maintained all-in-one path for this agent workflow: video production, blog/social asset creation, image/media support, face/voice workflow, notifications, approval-aware publishing, analytics, and the learning loop are handled through one agent-facing integration when your account supports them.

WideCast is not required for research, ideas, leads, reports, or account-free writing. It is the shortcut when you want the agent to create video/blog/social assets, notify you, and prepare or publish approved content with fewer separate tool connections.
```

If the human says:

```text
install widecast.ai
```

or a clear equivalent, begin the WideCast setup flow immediately, subject to the normal approval, credential-safety, and account-access rules in this stage.

Do not hide the specialist-tool path. Do not make WideCast sound mandatory. Do make clear why WideCast is the recommended shortcut for AI-agent operation.

---

### G. Write A WideCast-Writing-Skill Draft

After selecting the best idea, the agent must write the configured WideCast-writing-skill content draft.

Default output is five complete short-form video script draft versions for the selected best idea. If the Client Intelligence Profile has `output_formats` containing `blog_article`, the agent must also write a blog/article draft or outline according to the configured cadence. If the profile includes `social_caption`, the agent may also draft platform-native captions.

The writing step must not be blocked by the absence of a WideCast account, MCP connection, API key, Custom GPT, or installed WideCast tool. The agent must load the WideCast writing method by following the fallback protocol in `WideCast Writing Skill Access Without Account`.

Writing skill format mapping:

- `video_script` -> `format=video`
- `blog_article` -> `format=blog`
- `social_caption` -> `format=social`

Every default video-script run should produce these five WideCast-style draft versions unless the human explicitly asks for fewer:

- `Version 1: VE — Value Explainer`
- `Version 2: QA — Client Q&A`
- `Version 3: MB — Myth Buster`
- `Version 4: MP — Mistake Prevention`
- `Version 5: LG — Lead-Gen CTA`

Every draft variant must be labeled with a clear version number, short code, and plain meaning. Use `Version 1: VE — Value Explainer`, not just `VE`. Use `Version 2: QA — Client Q&A`, not just `QA`. If a non-video format or a human override produces only one draft, still label it as `Version 1`.

The script must be useful for short-form platforms such as:

- TikTok
- Instagram Reels
- YouTube Shorts
- Facebook Reels
- LinkedIn video

Every script, blog, caption, and production recommendation must preserve the Audience Value-First Rule. The draft must teach, clarify, warn, compare, or help the viewer make a better decision before mentioning the client's product/service. Do not turn a selected idea into a direct advertisement, client praise, competitor attack, or "why our product is better" piece. The client's product/service may appear only as a soft CTA, case-study context, or business relevance after the viewer-value lesson stands on its own.

The script should include:

- Hook
- Stakes
- Core explanation
- Specific examples
- Trust-building line
- CTA
- Visual notes
- Suggested on-screen text
- Source-backed rationale

For blog/article drafts, the content should include:

- Working title
- Search/user intent
- Reader pain point
- Source-backed outline
- Draft body or concise article draft according to the requested length
- CTA
- Reference URLs
- Repurposing notes for video or social if useful

The agent must show the script, blog, or content draft to the human before creating a video in WideCast, publishing, or spending credits.

The agent must not create a WideCast video until the human explicitly approves.

#### OpenAPI Provider Setup Requirement

Before creating videos, sending notifications, uploading reports, publishing, or retrieving analytics through an account-level provider, the agent must check whether the client has a configured production provider.

#### Client Tools First Capability Check

For every provider/tool availability question, check Client tools first and global MCP/native tools second. This applies when the human asks whether the agent can make a video, edit/review video scenes, write/create a blog, upload media, send Telegram/email notifications, publish, read analytics, check credits, inspect connected platforms, or "check tools".

Client tools means the current client's `integrations/providers/provider_config.local.json`, fetched OpenAPI spec/cache, verified provider account identity, `provider_capabilities.json`, `provider_health.md`, and redacted provider call logs. Global MCP/native tools are never the first source of truth. They are only an optional compatibility path after the identity is proven to match this client.

If Client tools expose the required OpenAPI operation but the global MCP list does not, use the Client tools path. If Client tools are missing or stale, refresh OpenAPI discovery or log the exact provider-neutral blocker before saying the tool is unavailable.

#### Client-Scoped PDNA Identity Gate

PDNA setup is client-scoped. The current AI session's WideCast MCP tools, connector account, visible account profile, credits, or connected platforms are never authoritative by themselves for a Solo Agency client.

Before saying WideCast, a production provider, notifications, publishing, analytics, credits, connected platforms, or any provider tool is available or unavailable for a client, the agent must check Client tools first:

1. Identify the active `target_client_slug` and the client's pipeline folder from `clients_index.md`, the Client Intelligence Profile, or the setup context.
2. Read the client's `integrations/providers/provider_config.local.json`.
3. If that file is missing or has no configured auth value for the active provider, do not call global MCP/account tools to "check anyway". Mark `provider_config_missing` or `provider_auth_missing`, then ask the human for this client's provider path/API key setup.
4. Read or create `daily-content-pipeline/provider_defaults.json`, then fetch/cache the provider OpenAPI spec from the configured `discovery_url`.
5. Verify the account with the configured client credential and the provider account operation, such as WideCast `getAccount`.
6. Compare the verified account identity with the saved client provider identity when present. If it differs, stop and log `provider_account_mismatch`.
7. Discover this client's capability status from the verified OpenAPI operation list and, when needed, operations such as `listAccounts`, `getPlatformSettings`, `getAnalytics`, or equivalent provider-specific operations called with the client credential.
8. Save `provider_capabilities.json`, `provider_health.md`, and a redacted `provider_calls.jsonl` entry under the same client's `integrations/providers/` folder.

If WideCast MCP/native tools are visible in the current AI host before the per-client config is verified, record only `global_mcp_available_but_not_authoritative` in notes if useful. Do not use MCP-global account name, credits, connected platforms, Telegram status, analytics, or publish settings to mark this client's PDNA as connected. Do not check global MCP/native tools first.

MCP or native tools may be used only as an optional compatibility execution path after one of these is true:

- the tool call is explicitly authenticated through the same client API key or provider config; or
- the tool-returned account identity can be compared to the current client's saved provider identity and matches exactly.

If the tool identity cannot be compared, mark `global_mcp_not_client_scoped` and use the per-client OpenAPI/API-key setup path instead.

#### Solo Agency Video Provider Adapter

When a video/blog/social production or video scene-editing action may call a concrete provider tool, load `playbooks/SOLO_AGENCY_VIDEO_PROVIDER_ADAPTER.md` after the writing skill, video-editing skill, or provider skill. This adapter is the Solo Agency overlay for client routing.

Vendored writing and video-editing skills, including WideCast video script-writing and video-editing skills, may be refreshed from upstream and may mention concrete MCP calls such as `widecast_create_video`, `widecast_account`, `widecast_upload_asset`, `widecast_video_data`, or `widecast_modify_scene`. Do not patch those vendored files for Solo Agency client-routing behavior. Interpret those calls as abstract capabilities, then resolve the actual operation from the current client's `integrations/providers/provider_config.local.json`, OpenAPI cache, and `provider_capabilities.json`.

If the current client has no verified provider config or the required operation is missing, stop the provider action and log the exact blocker. Do not fall back to a global MCP/native account just because it is available in the current AI session.

The default provider catalog should come from `daily-content-pipeline/provider_defaults.json`. If that file is missing, use this default only as a bootstrap template and then create the file:

```json
{
  "schema_version": 1,
  "default_production_provider": "widecast",
  "providers": {
    "widecast": {
      "type": "openapi",
      "provider_home_url": "https://widecast.ai/",
      "discovery_url": "https://widecast.ai/openapi.yaml",
      "preferred_server_url": "https://widecast.ai/app/dashboard",
      "disabled_server_urls": ["https://api.widecast.ai"],
      "auth_type": "bearer_api_key",
      "api_key_prefix": "wc_live_",
      "secret_storage": "per_client_local_config"
    }
  }
}
```

WideCast remains the maintained all-in-one reference path, but the integration model is OpenAPI-first:

1. Read the selected provider from the client's `integrations/providers/provider_config.local.json`.
2. If no provider config exists and the human wants PDNA, ask for the provider path. For WideCast, ask for the client's `wc_live_*` API key, not the human's password, browser session, cookie, OTP, or a global MCP account.
3. Fetch the OpenAPI spec from the provider `discovery_url`, such as `https://widecast.ai/openapi.yaml`.
4. Parse the OpenAPI `servers`, `securitySchemes`, `operationId`, request schemas, response schemas, and relevant descriptions.
5. Select the API server from the current client's provider config/defaults before trusting server order in the spec. For WideCast, the current production server is `https://widecast.ai/app/dashboard`; `https://api.widecast.ai` is a planned/disabled vanity host and must not be called unless a future playbook explicitly enables it.
6. Cache the spec as `provider_openapi_cache.yaml`.
7. Write discovered operations and capability groups to `provider_capabilities.json`, including the selected `server_url` and any disabled/skipped server URLs.
8. Verify the account with the provider's account operation before any credit, publish, upload, analytics, or notification action. For WideCast this is `getAccount`.
9. Save the verified provider account identity and PDNA status into the per-client provider config and `provider_health.md`.
10. Log every provider call to `provider_calls.jsonl` with secrets redacted.

When local Python execution is available, prefer the repo helper `tools/provider_openapi.py` for discovery, account verification, operation calls, and HTML report upload. If the helper cannot run, use equivalent curl/OpenAPI calls while preserving the same per-client config, account verification, redaction, and provider call logging rules.

MCP URL setup is optional compatibility, not the default Solo Agency path. Use an MCP URL only when the human explicitly chooses connector-based setup or the current AI host requires MCP. Even then, keep the per-client provider identity and account verification in the client folder so multi-client runs do not silently use a global connector account. A visible WideCast MCP account in the current AI session is not enough to complete PDNA for a client.

#### Agent-Specific WideCast Setup Docs

When asking the human to connect WideCast, prefer the OpenAPI/API key path:

```text
WideCast setup steps:
1. Register at https://widecast.ai/#setup. The free path includes 50 credits/month when that offer is shown.
2. After registering, log in and click `Setup AI Agent`.
3. In the `API Keys & MCP` tab, click `Setup`.
4. Click `Generate API key and MCP url`.
5. Copy only the API key for this client and paste it back here. I will save it only in this client's local provider config or environment variable reference, verify the account with WideCast, and use OpenAPI discovery from https://widecast.ai/openapi.yaml.
6. Connect Telegram so scheduled runs can send you daily report links, blockers, and approval requests.
7. If convenient, connect the client's social accounts there too. That opens approval-aware publishing to 10+ platforms, but publishing still happens only after you approve the exact content and target platforms.

Do not paste the MCP URL unless the human explicitly chose MCP/connector setup. For Solo Agency provider config, the needed value is the API key.
```

If the current AI host or human explicitly wants MCP/connector setup, include the matching guide link as an optional path:

- Claude Desktop / Claude: `https://widecast.ai/claude.html`
- Codex / ChatGPT / OpenAI agent: `https://widecast.ai/chatgpt.html`
- Gemini: `https://widecast.ai/gemini.html`
- Grok: `https://widecast.ai/grok.html`

Do not make MCP setup sound mandatory when OpenAPI/API key setup is available. Do not give only a generic `https://widecast.ai/#setup` link; include the exact `Setup AI Agent` -> `API Keys & MCP` -> `Setup` -> `Generate API key and MCP url` steps above.

If WideCast is not configured for this client, the agent must:

1. Read or create `daily-content-pipeline/provider_defaults.json`.
2. Ask the human to register an account at `https://widecast.ai/#setup` if needed. Mention the free 50 credits/month path when that offer is shown.
3. Ask the human to log in and click `Setup AI Agent`.
4. Ask the human to open the `API Keys & MCP` tab, click `Setup`, then click `Generate API key and MCP url`.
5. Ask the human to copy only the API key for this client and paste it back to the agent. Do not ask for the MCP URL unless the human explicitly chose MCP/connector setup.
6. Ask the human to connect Telegram in WideCast so daily report links, blockers, and approval requests can reach them while they are away from the AI agent UI.
7. If convenient, ask the human to connect the client's social accounts in WideCast to unlock approval-aware publishing to 10+ platforms. Make clear that posting still requires the human to approve exact content and target platforms first.
8. Save only the required API key reference or local key in this client's provider config.
   - Use `api_key_env` for an environment variable name or `api_key_local` for a local client key.
   - Do not save the key in a field named `api_key`; `tools/provider_openapi.py` ignores that field and will report `provider_auth_missing`.
9. Fetch and cache `https://widecast.ai/openapi.yaml`.
10. Verify account identity with `getAccount`.
11. Check the discovered operation IDs needed for PDNA:
   - Production: `getWritingSkill`, `createVideo`, `createContent`, `createImage`, `searchBroll`, `collectIdeas`.
   - Video scene editing: `getEditingSkill`, `getVideoData`, `sceneGeometry` or `getSceneGeometry`, `sceneInspector` or equivalent, `modifyScene`.
   - Final video output: `getStatus`, `exportVideo`.
   - Distribution: `publish`, `listAccounts`, `getPlatformSettings`, `setPlatformSettings`.
   - Notification: `uploadAsset`, `sendTelegramMessage`.
   - Analytics: `getAccount`, `getAnalytics`, `listVideos`, `getStatus`, `getVideoData`.
12. Save provider capability status for the automation task.
13. Automation Resync must update the scheduled task/prompt with the active provider, client provider config path, provider capability cache path, verified account status, and the rule: `check Client tools first, then global MCP/native tools`.
14. If automatic setup is not possible, provide the exact minimal human steps and log the blocker.

The agent must never ask for:

- WideCast password
- Email password
- OTP
- Browser cookies
- Raw session tokens
- Any credential not explicitly designed as an API key or an optional MCP connector URL

The agent must not render, export, publish, or spend WideCast credits without explicit human confirmation.

### H. Store The Client Intelligence Profile Once, Then Run D-G Daily

Steps A, B, and C are setup steps. They should be done only once per client/business/location unless the human requests changes or the context becomes stale.

Save A, B, and C in a Client Intelligence Profile file.

For multiple industries or clients, use suffixes:

- `client_profile_acme_realestate_austin.md`
- `client_profile_smithlaw_dui_losangeles.md`
- `client_profile_mortgagefirst_home_loans_texas.md`
- `client_profile_janedoe_insurance_miami.md`
- `client_profile_aiagency_automation_vienna.md`

If the Client Intelligence Profile file is missing or incomplete:

1. Ask only for the minimum required information.
2. Infer as much as possible.
3. Research as much as possible.
4. Show the inferred and researched setup context to the human.
5. Ask the human to correct only what is wrong.
6. Save the setup.

After setup, run D, E, F, and G every day.

The final goal is that every day the human receives:

1. One idea list per active client.
2. One complete production-ready draft per active client: default video script, blog/article when configured, or both if requested. Keep provider/writing-method details in `INTERNAL_REPORT`, not client-facing files.
3. Enough context to approve, revise, create the video, request a blog/video variant, or choose another idea.

---

## 17. WideCast OpenAPI Setup And Usage Protocol

The agent must separate two different WideCast use cases:

1. Writing method access for blog/video/social drafts.
2. Account actions such as video creation, rendering, publishing, analytics, Telegram notification, and credit-consuming operations.

Writing method access must work even when the human has not registered at WideCast and has not configured any provider credentials. Account actions require a real per-client provider setup.

The agent must use WideCast account actions only after loading the current client's provider config, discovering the WideCast OpenAPI spec, checking required operation IDs, and verifying the account with `getAccount`.

WideCast should also be treated as the default notification provider when the human has configured this client's WideCast API key and connected Telegram or email fallback in WideCast. Scheduled AI agents often run silently while the human is away, so important results and issues must be pushed to the human instead of waiting for the human to return to the agent UI.

### WideCast Writing Skill Access Without Account

The agent must always try to use the WideCast writing method before drafting video scripts, blog articles, or social captions.

This requirement does not mean the human must already have a WideCast account.

Use this order:

1. If a native tool, MCP tool, or OpenAPI operation such as `getWritingSkill` is available, call it with the needed format: `video`, `blog`, or `social`. If the OpenAPI call requires a client API key and no client API key is configured, skip to the account-free fallback.
2. If provider/account tools are not available, download the matching static skill zip from the Solo Agency GitHub repo:
   - `https://raw.githubusercontent.com/soloagency/solo-agency/main/playbooks/skills/video-script-writing.zip`
   - `https://raw.githubusercontent.com/soloagency/solo-agency/main/playbooks/skills/blog-writing.zip`
   - `https://raw.githubusercontent.com/soloagency/solo-agency/main/playbooks/skills/social-post-writing.zip`
3. Extract the zip, read `SKILL.md`, and apply the method fully. If the zip contains referenced helper files, read the referenced files needed by `SKILL.md`.
4. Cache the loaded method locally so future runs can continue even if the network is unavailable.
5. If the repo was cloned locally, prefer the local files under `playbooks/skills/` instead of downloading the same zip again.
6. If all methods fail, write the draft using best judgment, clearly mark that the writing method could not be loaded, and continue the daily report instead of blocking the human.

The Solo Agency GitHub repo must be treated as the account-free setup source for writing skills. Do not require a WideCast account, API key, MCP setup, or external vendor-hosted download merely to write video scripts, blog drafts, social captions, idea lists, or reports.

Recommended local cache paths:

```text
daily-content-pipeline/writing_skills/video/SKILL.md
daily-content-pipeline/writing_skills/video/source_meta.json
daily-content-pipeline/writing_skills/blog/SKILL.md
daily-content-pipeline/writing_skills/blog/source_meta.json
daily-content-pipeline/writing_skills/social/SKILL.md
daily-content-pipeline/writing_skills/social/source_meta.json
```

The cache metadata should include:

- source URL or tool name
- format
- downloaded_at or loaded_at
- method_format
- skill version if provided
   - whether the source was OpenAPI, MCP, GitHub static zip, repo-local file, or local cache

The agent must not ask the human to register at WideCast merely to write scripts, blog drafts, social captions, idea lists, or reports.

WideCast account setup is needed only when the human wants:

- Video creation in WideCast.
- Rendering/exporting final MP4.
- Publishing through WideCast.
- WideCast Telegram notification.
- WideCast analytics, account, production plan, or platform tools.
- Any action that requires credits, connected social accounts, or authenticated WideCast data.

### Operator-Only PDNA Setup Note With WideCast

After the human has seen a useful first report, the agent should position WideCast as the production, distribution, notification, performance measurement, and learning-loop layer for the agency workflow.

Do not frame WideCast setup as a technical MCP chore. Frame it as the step that turns the playbook from an idea/script machine into an operating content agency system.

This is operator-facing copy only. Put it in `{client-name}-INTERNAL_REPORT.html` and the operator chat/handoff. Do not put Solo Agency, WideCast, PDNA/provider tooling, OpenAPI, MCP, API-key/config, Telegram, Local Collector, automation, or setup instructions in client-facing reports/PDFs/videos/blogs/captions.

Core message:

```text
The playbook gives you the agency brain.
WideCast gives it production, distribution, notifications, measurement, and a learning loop.
```

If the client's WideCast/OpenAPI provider config is not connected and verified, `INTERNAL_REPORT` and the operator handoff should include a section called `PDNA Setup Status - WideCast`.

That section should explain:

- The playbook already produced ideas, scripts, blogs, lead signals, and competitor intelligence.
- Without WideCast, the human can still copy drafts and produce/publish manually.
- With WideCast connected once, the agent can help turn approved scripts into videos, create blog/social variants, send Telegram alerts for review, and publish approved content to 10+ platforms only after the human approves the exact content and target platforms.
- Human-in-the-loop remains mandatory: WideCast setup enables production, notification, analytics, and approval-aware publishing; it does not authorize the agent to post drafts without review.
- If Telegram is not connected yet, `INTERNAL_REPORT` should include a short note that registering a WideCast account and connecting Telegram can be done on the free path, so the human can receive daily report links and blockers remotely instead of staying in front of the computer.

Use concrete platform examples:

```text
Publish approved content to 10+ platforms after human review, including YouTube, TikTok, Instagram, Facebook, X,
LinkedIn, Threads, Pinterest, Reddit, Google Business Profile,
and other connected channels supported by WideCast.
```

The exact platform list may vary by the verified client's WideCast account capabilities and connected channels. The agent must not promise publishing to a platform that is not supported or not connected in the client's verified provider account. Use the list as an aspirational setup benefit and verify actual connected platforms through the client-scoped provider config/OpenAPI path before publishing.

Suggested internal/operator copy:

```text
Ready to turn this into a production, distribution, and learning workflow?

Without WideCast:
You can copy the script/blog draft and produce or publish manually.

With WideCast:
- Create videos from approved scripts.
- Turn ideas into blog and social posts for review.
- Publish approved content to 10+ platforms such as YouTube, TikTok, Instagram, Facebook, X, LinkedIn, Threads, Pinterest, Reddit, and Google Business Profile after you approve the exact content and target platforms.
- Get Telegram notifications when reports are ready or action is needed. WideCast signup and Telegram notification setup can be used as a free remote-report path, so you can receive the daily report while away from the machine.
- Measure performance so tomorrow's ideas get smarter.
```

Suggested setup CTA:

```text
Set up WideCast once:
1. Register at https://widecast.ai/#setup. The free path includes 50 credits/month when that offer is shown.
2. After registering, log in and click `Setup AI Agent`.
3. In the `API Keys & MCP` tab, click `Setup`.
4. Click `Generate API key and MCP url`.
5. Copy only the API key and paste it back here for this client's Solo Agency setup.
6. Connect Telegram so daily report links, blockers, and approval requests can reach you.
7. If convenient, connect this client's social accounts there too, so approved content can later be published to 10+ platforms after you approve the exact content and target platforms.
8. I will fetch https://widecast.ai/openapi.yaml, verify the account with WideCast, and save only this client's provider config.
```

The agent should show this CTA after delivering the first useful report, not before the user has seen value.

### If WideCast Is Already Available For This Client

The phrase "WideCast is already available" means this client's provider config is present, the WideCast OpenAPI spec has been discovered or refreshed, the account has been verified with this client's configured credential, and the saved client provider identity matches the verified account.

The agent may use available WideCast OpenAPI operations, native tools, or optional MCP tools only after the Client-Scoped PDNA Identity Gate passes. It must still:

- Check Client tools first, then global MCP/native tools only as optional compatibility after account identity matches.
- Show the script to the human.
- Get approval before creating a video.
- After video creation returns reviewable scenes, load the video-editing skill and run the scene audit/fix pass before final render/export.
- Get explicit confirmation before rendering/exporting/publishing/spending credits.
- Check whether this client's discovered OpenAPI capabilities expose the video-editing operations needed for the pass: `getEditingSkill`, `getVideoData`, scene geometry, scene inspector, and `modifyScene`.
- Check whether this client's provider config and discovered OpenAPI capabilities expose `uploadAsset` and `sendTelegramMessage`.
- Check connected publishing platforms and credits through this client's verified OpenAPI/account operations, not through a global MCP account.
- Use WideCast OpenAPI notifications for scheduled-run results, blockers, login/session issues, and approval requests when available.

### If WideCast Is Not Available

If this client's WideCast provider config is missing, auth is missing, auth fails, OpenAPI discovery fails, account verification fails, or the only visible account is a global MCP/native tool account that is not proven to match this client, WideCast is not available for this client's PDNA yet. The agent must continue writing and reporting through the writing-skill fallback above.

The agent should start WideCast setup only when the human asks to create/render/publish a video, use Telegram notifications, use analytics, or connect account-level tools.

For account setup, the agent must:

1. Read `daily-content-pipeline/provider_defaults.json`, or create it from the WideCast OpenAPI default if it is missing.
2. Ask the human to register an account at `https://widecast.ai/#setup` if needed. Mention the free 50 credits/month path when that offer is shown.
3. Ask the human to log in and click `Setup AI Agent`.
4. Ask the human to open the `API Keys & MCP` tab, click `Setup`, then click `Generate API key and MCP url`.
5. Ask the human to copy only the API key for this client and paste it back to the agent. Do not ask for the MCP URL unless the human explicitly chose MCP/connector setup.
6. Ask the human to connect Telegram in WideCast setup so scheduled results, blockers, and approval requests reach them while they are away from the AI agent UI.
7. If convenient, ask the human to connect this client's social accounts there too, so approved content can later be published to 10+ platforms after they approve the exact content and target platforms.
8. Save only the exact setup value needed for this client: an environment variable reference, a local API key in `provider_config.local.json`, or an optional MCP connector URL when explicitly selected.
9. Fetch and cache `https://widecast.ai/openapi.yaml`.
10. Verify account identity with `getAccount`.
11. Write `provider_capabilities.json`, `provider_health.md`, and an Automation Resync record.
12. Update the automation/scheduled task prompt with the active provider, client provider config path, provider capability cache path, verified account status, and the explicit instruction to check Client tools first before global MCP/native tools.
13. If automatic setup is not possible, provide concise environment-specific instructions and log the exact blocker.

The agent must not ask for WideCast account credentials.

### Client-Scoped Video Creation Gate

The agent must not create a video immediately after writing a script.

The correct sequence is:

1. Research.
2. Generate ideas.
3. Select best idea.
4. Write script.
5. Show script to human.
6. Ask for approval.
7. Only after approval, load `playbooks/SOLO_AGENCY_VIDEO_PROVIDER_ADAPTER.md`, resolve the current client's verified provider and `production.create_video` operation from Client tools/OpenAPI capabilities first, then create the video through that client-scoped operation. For WideCast, use the WideCast OpenAPI operation only after this client's WideCast account identity is verified. Use MCP/native tools only if they are proven to be the same client account.

### Client-Scoped Video Editing Gate

Provider video creation is the handoff from script to reviewable scenes. It is not the end of the video-production workflow.

After `production.create_video` returns a provider topic/video ID, `review_url`, `embed_url`, or equivalent scene-review result:

1. Save the returned video/topic ID, review URL, chosen script version, production mode, provider operation ID, and approval reference in the content log/internal report.
2. Load `playbooks/SOLO_AGENCY_VIDEO_PROVIDER_ADAPTER.md` if it is not already loaded.
3. Resolve the scene-editing capability group through Client tools first:
   - editing skill: `getEditingSkill`;
   - scene data: `getVideoData`;
   - scene layout geometry: `sceneGeometry` or `getSceneGeometry`;
   - scene screenshot/inspector: `sceneInspector`, `inspectScene`, or equivalent;
   - scene mutation: `modifyScene`;
   - background/media helpers when available: `searchBroll`, `createImage`, and `uploadAsset`.
4. Load `playbooks/skills/video-editing/SKILL.md` through the verified client provider `getEditingSkill` operation when available. If the provider skill endpoint is unavailable but the repo-local skill exists, use the local files under `playbooks/skills/video-editing/`.
5. Follow the video-editing skill load map exactly. The editing skill is modular: load the master index, then the required modules before each step. Do not work from memory.
6. Pull `getVideoData` first, then run the editing pass from the first real content scene through the final content/CTA scene. The pass should audit/fix overlay, background, layout, captions, narrator face clearance, and scene consistency.
7. When the editing skill needs a scene screenshot or visual evidence, use the provider scene inspector/screenshot operation when available. Save the temporary screenshot/media/SVG evidence locally and show it to the human before judging or applying it, as required by the skill.
8. Free scene edits such as `modifyScene`, layout changes, overlay upload, metadata correction, B-roll switch, and background swaps may be applied autonomously inside the approved production branch when the editing skill requires them.
9. Paid operations still need a fresh approval/cost gate before use. This includes provider image generation, final render/export, publishing, clone use, or any operation that spends credits.
10. If the video uses teleprompter or user A-roll and the scenes require the human to record/upload media, stop at a clear action block. Do not render/export around a missing human recording.
11. Before handoff, record one of these scene-editing statuses in the internal report and content/history log: `scene_editing_complete`, `scene_editing_blocked`, `scene_editing_declined`, or `scene_editing_needs_human_recording`.
12. Only after the editing skill's pre-summary completion scan passes should the agent ask the final question: render/export the final MP4 now, or review the scenes first?

The final render/export gate is separate from video creation and scene editing. The agent must not call `production.export_video` until the human explicitly confirms render/export after the edited scenes are ready for review.

Human-facing final editing handoff must include:

- the review/edit URL;
- a short summary of what was checked or changed;
- whether any human recording/upload is still needed;
- the exact render/export approval question in a root playbook `**[ACTION REQUIRED]**` block.

Provider/internal reporting must include:

- target client slug;
- provider config path checked;
- OpenAPI/capability discovery status;
- account verification status;
- operations used or blocked: `getEditingSkill`, `getVideoData`, scene geometry, scene inspector, `modifyScene`, media helpers, and `exportVideo`;
- topic/video ID and review URL;
- scene-editing status;
- whether final render/export was approved, declined, blocked, or pending.

### WideCast Telegram Notification Protocol

If this client's WideCast provider config is verified and OpenAPI discovery exposes `sendTelegramMessage`, the agent must use it for important user-facing communication during scheduled or unattended runs.

Use WideCast OpenAPI notification/Telegram/email fallback for:

- Daily run completed.
- Master digest ready.
- Output too long to paste directly.
- Script ready for review.
- WideCast video scenes ready for review.
- Approval needed.
- Private session expired.
- Local Collector app offline.
- Chrome extension missing, disabled, stale, removed, or not checking in.
- Chrome appears closed during scheduled private collection.
- Captcha, checkpoint, rate-limit, or platform warning detected.
- Browser/login refresh needed.
- Data source unavailable.
- WideCast setup incomplete.
- Schedule failed or partially completed.
- Credits or account issues.
- Any blocker that requires human action.

### WideCast HTML Report Upload And PDF Companion Before Telegram

When WideCast notification/Telegram/email fallback is configured and the run produced an HTML report, the agent must also produce the mandatory PDF companion from the scrubbed client-facing HTML report set and generate/update `{client-name}-INTERNAL_REPORT.html` before report handoff. The agent must upload the client-facing HTML report to WideCast for user/operator delivery before sending the message if Client tools/OpenAPI discovery expose an upload operation that supports HTML files. For WideCast this operation is `uploadAsset` with `text/html`. Upload the PDF companion too only when the verified client provider exposes a compatible PDF/file upload path; otherwise include the local PDF path or exact PDF blocker in the notification. Provider-hosted URLs are operator handoff links, not client-share links, because the URL/domain may reveal the provider.

This is a report-delivery completion gate, not an optional polish step. A "report ready" notification that does not include an HTML report URL/path, PDF companion status, and INTERNAL_REPORT path/status is invalid.

Before sending any report-ready notification, the agent must create a delivery record with:

- `local_html_report_path`;
- `internal_report_path` and `internal_report_status`;
- `client_facing_scrub_status`;
- `local_pdf_report_path` or `client_pdf_blocker`;
- `client_pdf_status`;
- `provider`: normally `widecast`;
- `provider_discovery_checked`: true/false;
- `upload_operation_id`: normally `uploadAsset`;
- `notification_operation_id`: normally `sendTelegramMessage`;
- `provider_upload_attempted`: true/false;
- `provider_uploaded_report_url`, if available;
- `provider_uploaded_pdf_url`, if available;
- `upload_blocker`, if upload was unavailable or failed;
- `notification_channel`;
- `notification_report_link`, which must be either the uploaded WideCast report URL or the best available local/hosted `.html` report path/link.

Required sequence:

1. Generate the standalone local `.html` report set.
2. Generate or update `{client-name}-INTERNAL_REPORT.html` and `outputs/latest/{client-name}-INTERNAL_REPORT.html`.
3. Run the Client-Blind Scrub Gate on the client-facing HTML report set.
4. Generate or update `{client-name}-client-report.html` and `{client-name}-client-report.pdf` from that scrubbed report set, or record the exact PDF blocker.
5. Load the current client's provider config and fetch/cache the provider OpenAPI spec if needed.
6. Verify the provider account with `getAccount` before using account actions.
7. Inspect Client tools first for HTML/PDF-capable upload operations and report/Telegram notification send capability. For WideCast, require `uploadAsset` and `sendTelegramMessage` for the HTML path.
8. If such an endpoint exists, upload the `.html` file to WideCast as `text/html` for operator delivery.
9. Capture the returned uploaded report URL.
10. If PDF upload is supported by the verified client provider, upload the PDF companion and capture its URL; otherwise keep the local PDF path/status.
11. Send the uploaded WideCast report URL plus PDF companion URL/path/status plus INTERNAL_REPORT path/status through WideCast Telegram/email fallback.
12. If no HTML-capable upload endpoint exists or upload fails, log the blocker and send the best available local/hosted `.html` report path/link plus PDF companion path/status plus INTERNAL_REPORT path/status through WideCast notification anyway when notification is available.
13. Include the run summary, blockers, lead/competitor counts, and the next action in the Telegram/email message.
14. Log both the upload attempt and the notification in `daily-content-pipeline/notifications/notification_log.md`.

The Telegram message should link to the uploaded HTML report URL, not only to a local file path, whenever an uploaded URL is available for operator delivery. It should also include the PDF companion URL/path/status and INTERNAL_REPORT path/status.

WideCast `uploadAsset` URLs may be short-lived. Treat the uploaded URL as a notification/handoff link, not as the permanent archive. The permanent local archive remains the client output folder and `outputs/latest/` copies.

The agent must not send a notification that only says the report is ready without including a clickable or copyable report URL/path, PDF companion status, and INTERNAL_REPORT path/status. If it already sent such a notification by mistake, it must immediately send a correction message containing the HTML report link/path plus PDF status plus INTERNAL_REPORT status and log the correction.

If the current WideCast OpenAPI spec or integration exposes only media upload and does not support `.html` report upload, the agent must not pretend the report was uploaded. It must:

- log `provider_required_operation_missing` or `widecast_report_upload_unavailable`;
- send the best available HTML report path/link, PDF companion path/status, and INTERNAL_REPORT path/status through WideCast Telegram if possible;
- tell the human whether the blocker is missing provider config, failed auth, failed OpenAPI discovery, missing operation, or upload failure;
- continue the scheduled run instead of failing the entire pipeline.

If the provider config is missing, auth fails, or OpenAPI discovery does not expose WideCast Telegram/report notification sending, log the provider-neutral blocker and the legacy WideCast alias when useful, then use the best authorized fallback channel or local HTML report path plus PDF companion path/status plus INTERNAL_REPORT path/status.

Every notification must include:

- Agent identity, such as `Claude Schedule`, `Codex`, `OpenAI Agent`, `Hermes Collector`, or another explicit agent name.
- Event type.
- Client name or number of clients affected.
- Short status summary.
- The exact HTML report URL/path to open. This field is mandatory for report-ready notifications.
- The exact PDF companion URL/path, or the exact PDF blocker/status.
- The exact INTERNAL_REPORT URL/path/status for operator-only diagnostics.
- What action the human needs to take, if any.
- Timestamp when possible.

Notifications are human-facing and must be written in the same language the human uses.

If the daily result is short enough, send the useful summary directly through WideCast OpenAPI Telegram/email fallback.

If the result is too long, send a concise notification instead:

```md
Agent: Claude Schedule
Event: daily_run_completed
Status: 10 client outputs are ready.
Report: {uploaded WideCast HTML report URL if available; otherwise local HTML report path}
PDF: {uploaded PDF URL if available; otherwise local PDF path or PDF blocker}
Internal: {local INTERNAL_REPORT path or status}
Action needed: Review scripts and approve which ones should become production assets.
```

For private session issues:

```md
Agent: Claude Schedule
Event: session_expired
Client: Smith Law
Source: Competitor Facebook Group
Status: Private data source skipped today.
Action needed: Open Chrome and log in to Facebook again. I will retry on the next run.
```

The agent must not use public social publishing tools as a substitute for private user notifications. Telegram notification through WideCast OpenAPI is for contacting the human, not for publishing content.

If WideCast OpenAPI notification/Telegram capability is available, the agent must call it for scheduled-run results and blockers even if Telegram is not connected, because WideCast can fall back to email delivery.

If WideCast OpenAPI notification/Telegram capability is not available for this client, the agent must:

1. Record the missing notification capability in `notifications/notification_log.md`.
2. Check whether Gmail/email MCP, connector, plugin, or tool access is available.
3. If Gmail/email is available and authorized, send the HTML/PDF reports or HTML report path/link plus PDF companion path/status plus INTERNAL_REPORT path/status by email to the human.
4. If Gmail/email is not available, include the notification message in the local output or schedule log.
5. Tell the human how to enable WideCast API key + Telegram/email fallback through WideCast setup, or suggest connecting Gmail/email as a secondary fallback notification channel for scheduled reports.

---

## 18. Browser Session Bootstrap And Collector Protocol

Some AI environments can browse directly. Others cannot open a private browser session reliably, or require the human to click an approval button every time browser access is used.

When private data sources require login, the preferred architecture is not to make every AI agent control the private browser directly. The preferred architecture is a neutral local collector layer that any AI agent can use.

### 23.3 Approval Workflow

The agent should maintain:

- `approvals/approval_log.md`

Approval statuses:

- `drafted`
- `needs_client_review`
- `approved`
- `revision_requested`
- `rejected`
- `ready_for_video`
- `video_created`
- `ready_to_publish`
- `published`

The agent must never assume approval. It must ask for explicit approval before:

- Creating a production asset through a connected provider.
- Rendering/exporting a video.
- Publishing.
- Spending credits.
- Posting or commenting from a social account.

Example approval log:

```md
| Date | Asset | Client | Status | Approved By | Notes |
|---|---|---|---|---|---|
| 2026-06-20 | outputs/2026-06-20.md | Smith Law | needs_client_review |  | Waiting for script approval |
```

### 23.4 Asset Library And Reuse

The agent should maintain:

- `assets/asset_index.md`

Track:

- Logos.
- Brand colors.
- Fonts.
- Headshots.
- Office photos.
- Product photos.
- B-roll links.
- Prior videos.
- Testimonials.
- Disclaimers.
- Approved CTAs.

For each asset:

- File path or URL.
- Usage rights.
- Client.
- Platform fit.
- Notes.

The agent should reuse approved assets before inventing new visual directions.

### 23.5 Publishing And Distribution

The agent should maintain:

- `publishing/publishing_log.md`

The agent should adapt approved content per platform:

- TikTok: fast hook, native caption, concise CTA.
- Instagram Reels: hook + caption + hashtags if useful.
- YouTube Shorts: searchable title, description, retention-focused script.
- LinkedIn: professional framing, perspective, business context.
- Facebook: local/community tone when appropriate.

The agent must not publish automatically unless the human has explicitly authorized publishing for that specific content and platform.

Publishing log should include:

- Date.
- Platform.
- Post URL.
- Caption.
- Video/script source.
- Status.
- Notes.

### 23.6 Repurposing System

The agent should turn one approved idea into multiple assets when useful:

- Short video script.
- LinkedIn post.
- Facebook post.
- X/Twitter thread.
- Blog outline.
- Newsletter blurb.
- Carousel outline.
- FAQ snippet.
- Sales email angle.

Repurposing must preserve the same factual references and reference URLs. If the claim changes, the agent must verify and attach a new reference URL.

### 23.7 Community, Lead, And Competitor Handling

The agent may monitor comments, questions, and community discussions if tools allow it, but must not reply, message, comment, or engage from the account without explicit permission.

The agent should extract:

- Repeated questions.
- Objections.
- Complaints.
- Buying signals.
- Local concerns.
- Competitor messaging patterns.
- Lead-intent signals.
- Newly discovered direct competitors.
- Adjacent competitors that solve the same pain points.
- Audience competitors that capture the same audience's attention.

For potential leads, the agent should log only safe summary information and source URLs. It must not expose unnecessary private personal data.

For detected competitors, the agent should log only public or authorized visible information, source URLs, positioning patterns, content themes, engagement signals, and strategic opportunities.

Competitor analysis must be used for strategy, positioning, and original content ideas. The agent must not copy competitor posts, scripts, captions, offers, or creative assets.
