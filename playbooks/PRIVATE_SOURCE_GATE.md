# Private Data Source Gate

Load this short gate immediately whenever the human asks to scan, monitor, review, collect, scrape, open, or read any private data source.

This gate exists to prevent conversation drift. Even if the conversation moved through many unrelated topics, the moment private data source intent returns, the agent must reload this gate before opening any browser, extension, automation tool, or private URL.

## Trigger Phrases

Treat any of these as private data source triggers:

- `scan private group`, `scan group`
- Facebook groups, joined groups, fanpages, private pages, competitor profiles
- X/Twitter, LinkedIn, Instagram, TikTok, YouTube, Reddit/subreddits, GitHub areas that require login or account context
- Discord, Slack, private forums, newsletters, dashboards, member communities
- `private data source`, logged-in source, social/community source
- private data source discovery, joined/member group discovery, subreddit/community discovery
- home feed, for-you feed, recommendation feed, following list, subscriptions, joined communities
- any request to use the human's logged-in account, membership, social graph, private feed, or browser session

If unsure whether a source is public or private, treat it as private until proven public.

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

If the Local Collector app or extension is unavailable, do not fall back to Claude in Chrome, Codex browser, Playwright, or another agent-controlled browser. Continue work with public data sources only and mark private data sources as `pending_private_activation` or `collector_unavailable`.

## Human-Facing Preflight Roadmap

Before starting any private data source scan, show or internally verify this gate. If handing control back to the human, include it in the reply as a progress roadmap, not as a form for the human to answer line by line.

Use font/text status icons:

- `✓` done
- `→` current check
- `○` pending
- `!` blocked or needs human action
- `–` skipped or not applicable with a short reason

The human-facing version must explain that the agent is doing these checks and that the human only needs to act when the agent asks one concrete next-step question.

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

If any required item is missing, do not scan private data sources yet. Ask the next concrete setup/repair question or continue public data sources only.

## Human-Facing Reminder

When replying about a private data source scan, include a compact reminder:

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
- collected data was analyzed for ideas, leads, competitors, new sources, and evidence;
- the report/idea matrix/drafts were updated or the blocker was honestly reported.
