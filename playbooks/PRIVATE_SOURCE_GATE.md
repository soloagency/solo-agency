# Private Data Source Gate

Load this short gate immediately whenever a human request, scheduled task, automation prompt, or run step involves any private data source topic, including scanning, monitoring, reviewing, collecting, scraping, opening, or reading a private data source.

This gate exists to prevent conversation drift. Even if the conversation moved through many unrelated topics, the moment private data source intent returns, the agent must reload this gate before opening any browser, extension, automation tool, or private URL.

## Trigger Phrases

Treat any of these as private data source triggers:

- `scan private group`, `scan group`
- Facebook groups, joined groups, fanpages, private pages, competitor profiles
- Facebook group search or keyword-based group discovery, such as `https://www.facebook.com/search/groups/?q=...`
- X/Twitter, LinkedIn, Instagram, TikTok, YouTube, Reddit/subreddits, GitHub areas that require login or account context
- Discord, Slack, private forums, newsletters, dashboards, member communities
- `private data source`, logged-in source, social/community source
- private data source discovery, joined/member group discovery, subreddit/community discovery
- home feed, for-you feed, recommendation feed, following list, subscriptions, joined communities
- any request to use the human's logged-in account, membership, social graph, private feed, or browser session

If unsure whether a source is public or private, treat it as private until proven public.

Classification tie-breaker: any source already on the `private_data_sources` list, and any social-platform page/profile/group/channel of the client or of a monitored competitor, is collector-only regardless of whether it loads logged-out. "Public" for agent-browser research means the non-social web (websites, articles, docs, search results, public news/forums). Reclassifying a private/social source as public requires explicit human approval.

Runtime verification before honoring saved flags: at every scheduled or manual run, perform Collector Runtime Verification (try `/status`; if localhost is unreachable from the AI sandbox, read the local collector health/status files) BEFORE honoring any saved `public_data_sources_only`, `private sources postponed`, or `pending_private_activation` flag. The saved flag may be a stale snapshot; the human may have activated the collector since it was written.

## Required Reload

Before acting on a private data source trigger, load:

1. `playbooks/PRIVATE_SOURCE_GATE.md`
2. `playbooks/02_PRIVATE_SOURCE_SETUP.md`
3. `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md`
4. `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md`

Do this even if those files were loaded earlier in the conversation. Do not rely on memory.

## Forbidden Browser Rule

For private data source collection, the agent must not use:

- Claude in Chrome or Claude Chrome Extension;
- Codex built-in browser, Codex in-app browser, or browser tools controlled directly by Codex;
- ChatGPT browser, Gemini browser, Grok browser, or any agent-native browsing surface;
- Playwright/Puppeteer/Selenium controlled directly by the AI agent;
- a fresh browser profile opened by the AI agent;
- remote-debugging browser sessions opened by the AI agent;
- browser cookies, exported storage state, credentials, OTPs, tokens, or passwords.

These tools may be used for public web research, local UI testing, opening setup instructions, or checking public pages. They must not be used to read logged-in private groups, feeds, dashboards, profiles, member pages, or community data.

## Collector-Only Rule

Private data source collection must use only:

```text
Human's logged-in Chrome
  -> Solo Agency Local Collector extension
  -> Local Collector app running outside the AI sandbox
  -> local files / localhost status
  -> AI agent reads local output and analyzes it
```

If the Local Collector app or extension is unavailable after Collector Runtime Verification, do not fall back to Claude in Chrome, Codex browser, Playwright, or another agent-controlled browser. Continue work with public data sources only and mark private data sources as `pending_private_activation`, `collector_status_unverified`, or `collector_offline_or_unreachable` with the exact blocker.

## Human-Facing Preflight Roadmap

Before starting any private data source scan, show or internally verify this gate. If handing control back to the human, include it in the reply as a progress roadmap, not as a form for the human to answer line by line.

Use font/text status icons:

- `✓` done
- `→` current check
- `○` pending
- `!` blocked or needs human action
- `–` skipped or not applicable with a short reason

The human-facing version must explain that the agent is doing these checks and that the human only needs to act when the agent asks one concrete next-step question. That concrete next-step question or command must be in a standalone `**[ACTION REQUIRED]**` block from the root playbook.

```text
Private Data Source Gate planned preflight
These are the checks I run before scanning private data sources. You only need to act when I ask one specific next-step question.

✓ Stage 2, Stage 8, and Stage 9 reloaded for this private data source request
→ Local Collector app reachable
○ Bridge identity verified: /status.config_file, /status.output_dir, and /status.run_now_request_file point to the current setup's daily-content-pipeline/collector tree
○ Solo Agency Local Collector extension recent
○ Approved private data sources loaded
○ Scan depth read from collector config, or safe default stated
○ Automation freshness check: if a schedule/automation already exists, latest private data source changes must be synced into the automation/scheduled task prompt/contract/source state, not only collector config
Collection method: Solo Agency Local Collector only
Forbidden for logged-in sources: Claude in Chrome, Codex/browser tools, Playwright, or any agent-controlled browser
```

If any required item is missing, do not scan private data sources yet. Ask the next concrete setup/repair question in a `**[ACTION REQUIRED]**` block or continue public data sources only.

## Human-Facing Reminder

When replying about a private data source scan, include a compact reminder. If the human needs to approve sources, load/reload an extension, start/restart the bridge, log into a Chrome profile, or choose scan/discovery scope, put that request in a `**[ACTION REQUIRED]**` block.

Include this compact reminder:

```text
Private collection method: Solo Agency Local Collector only.
I will not use Claude in Chrome, Codex/browser tools, Playwright, or any agent-controlled browser for logged-in sources.
```

This reminder should appear in private data source progress updates, blocker messages, setup repair messages, and report handoffs while private data source work is pending or active.

## Completion Gate

A private data source scan is not complete until:

- the Local Collector app was reachable;
- the Solo Agency Local Collector extension was recent;
- data was collected through approved sources only;
- Stage 10 was loaded (LOAD LEDGER) before presenting any leads/competitors;
- collected data was analyzed for data points, leads, competitors, newly discovered sources, and evidence;
- the idea matrix, best idea, and drafts were updated;
- the private lane report (`{client-name}-private-data-sources-report.html`) and the daily staging index were updated WITHOUT overwriting `{client-name}-public-data-sources-report.html`;
- the combined `{client-name}-client-report.html` + PDF companion + `{client-name}-INTERNAL_REPORT.html` were rebuilt, and `{client-name}-report_state.json` plus `outputs/latest/` copies were reconciled;
- or the blocker was honestly reported.
