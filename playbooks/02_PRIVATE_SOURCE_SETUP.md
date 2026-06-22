# Private Source Setup

Stage: `02`

## Load Rule

Load when private sources, manual private URLs, joined groups, followed profiles/pages/KOLs, subscribed channels, recommendation feeds, source discovery, or Local Collector activation are requested, approved, pending, or blocked.

## Hard Gates For This Stage

- Manual private sources and optional source discovery are independent.
- Ask explicitly whether the human wants to scan Facebook groups they are already a member of.
- Use the Facebook joined-groups URL only with explicit consent.
- Do not use automated approval-gated browser extension flows for unattended collection.
- Collector success alone is not completion; analyze data and regenerate the report.

## Source Preservation Rule

This file is detailed source material moved from the original monolithic `SOLO_AGENCY_PLAYBOOK.md`.

Do not summarize away requirements, examples, checklists, schemas, protocols, URLs, edge cases, warnings, approval gates, or completion gates. If a downstream agent needs to shorten its response to the human, it may summarize the response, but it must still obey the full requirements in this file.

---

## Latest Delta Override: Discovery Scroll Depth

Source Discovery Mode and Daily Content Monitoring Mode are different.

For Source Discovery Mode, including joined groups, followed sources, subscribed channels, KOLs, communities, and recommendation surfaces, scroll deeply until no new source names or URLs appear for 3 consecutive scrolls, with a hard safety cap such as 80 scrolls.

For Daily Content Monitoring Mode, keep the conservative default: 5 scrolls per source, maximum 10, 5 seconds between scrolls, and around 20 daily private sources or fewer per client.

If any older copied section appears to use the daily 5-scroll default for source discovery, this latest delta override wins.

## Group Scan Communication Rule

Every time the agent tells the human it will scan or monitor groups, communities, fanpages, or logged-in social sources, it must state the actual scan depth and where that value comes from.

For daily content monitoring, the agent must say the human-facing equivalent of:

```text
I will go through each approved group/source one by one and scroll {N} times per source, using the value from the Local Collector configuration. The default is 5 scrolls per source, the maximum is 10, and I will wait about 5 seconds between scrolls.
```

The agent must determine `{N}` from the best available source:

1. Read `daily-content-pipeline/collector/collector_config.json` if it exists.
2. If the Local Collector app is running, check `GET http://127.0.0.1:17321/status` and/or `GET /config` when available.
3. If neither source is available, use the documented default: `5` scrolls per source, max `10`, 5-second delay.

For source discovery, do not say "5 scrolls" unless the configured discovery mode explicitly says that. Source Discovery Mode uses the deep-scroll rule: continue until no new source names/URLs appear for 3 consecutive scrolls, with a hard safety cap such as 80 scrolls.

If the agent cannot read the actual config, it must be honest:

```text
I cannot read the Local Collector config right now, so I will use the safe default: 5 scrolls per approved source, max 10, with about 5 seconds between scrolls.
```

---

## 6. Private Data Source Rule

The agent must ask the human to provide private data sources they want monitored.

Examples of private data sources:

- Competitor Facebook fanpages.
- Competitor Instagram profiles.
- Competitor TikTok profiles.
- Competitor LinkedIn profiles.
- Competitor YouTube channels where logged-in access is useful.
- KOL, creator, expert, or influencer profiles the human follows.
- Subscribed YouTube channels.
- Followed TikTok, Instagram, X, LinkedIn, or Facebook profiles/pages.
- Facebook groups.
- LinkedIn groups.
- Reddit communities.
- Niche forums.
- Local community groups.
- Platform recommendation feeds, used only as discovery surfaces with explicit consent.
- Private newsletters.
- Slack or Discord communities.
- Client-owned dashboards.

The agent must say:

`Please provide any private or logged-in sources you want monitored, such as competitor profiles, fanpages, groups, communities, or forums. I will prioritize sources related to the client's primary industry, target audience, location, pain points, and carefully selected related industries. If login is required, please log in manually through the available browser session. Do not share credentials. For account safety and platform-respectful monitoring, please avoid adding too many private sources for one client; around 20 sources or fewer is a good daily default. If you provide more, I will prioritize the most relevant sources and rotate the rest.`

### Facebook Member Groups Review

The agent must specifically ask whether the human wants to include Facebook groups where the human is already a member.

This is a separate discovery path from asking the human to paste private-source URLs manually. Many non-technical humans do not remember all useful groups they have already joined. The agent must offer to discover candidate groups from the human's joined-groups page, with explicit permission, then filter the list for relevance.

The agent must explain that it will not treat every group as useful by default. It will review and filter the available groups based on whether they contain discussions relevant to:

- The client's industry.
- The client's sub-industry.
- The client's related industries, only when the bridge back to the primary offer is clear.
- The target audience.
- The target location.
- The inferred pain points.
- The client's business offer.
- Recurring questions, objections, complaints, or buying signals.

The agent should say:

`Do you want me to review the Facebook groups you are already a member of and select only the groups that look useful for this client? If you say yes, I will use your logged-in Chrome session through the Solo Agency Local Collector, open your joined-groups page, and filter groups based on the client's primary industry, related industries, audience, location, and pain points. I will not ask for credentials, cookies, passwords, or tokens. For account safety, I will keep the active daily private-source list conservative, around 20 sources or fewer per client by default, and rotate lower-priority groups when needed.`

If the human agrees:

1. Treat the following URL as the Facebook joined-groups discovery source:

```text
https://www.facebook.com/groups/joins/?nav_source=tab&ordering=viewer_added
```

2. If the Solo Agency Local Collector extension and Local Collector app are not installed and healthy, activate/setup them before attempting this scan.
3. Do not use Claude Chrome Extension for this discovery scan.
4. Do not ask the human to paste Facebook cookies, passwords, tokens, or credentials.
5. Use the human's already logged-in Chrome session. If Facebook is logged out, mark `facebook_session_expired` and ask the human to log in manually.
6. Create a manual `run_now` job for the Local Collector to scan the joined-groups discovery URL.
7. Use conservative scan settings:
   - `max_scrolls_per_source`: default 5, max 10.
   - `scroll_delay_seconds`: 5.
   - Read visible text and current URLs only.
8. Review visible group names, group URLs, descriptions, category hints, membership/context hints, and any visible preview text.
9. Select only groups that are relevant to the client pipeline.
10. Classify candidate groups as:
    - `recommended_daily`
    - `recommended_weekly`
    - `optional`
    - `skip_not_relevant`
    - `skip_too_broad`
    - `skip_sensitive_or_risky`
11. Show the human a short `Facebook Member Groups Review` result before saving:
    - recommended groups
    - why each group is relevant
    - proposed scan cadence
    - skipped groups count and examples
    - account-safety note
12. Ask the human to approve the recommended groups before adding them as active `private_data_sources`.
13. After approval, add selected groups to `private_data_sources`.
14. Log skipped groups as not relevant when appropriate.
15. Save the discovery output under:

```text
daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/history/YYYY-MM/facebook_member_groups_review_YYYY-MM-DD.md
```

16. Also add selected or newly discovered group candidates to `New Private Sources Detected` in the next report.

If the human declines:

- Do not inspect Facebook groups.
- Continue with other public and private sources.

If the human provides no private sources:

- Continue with public sources only.
- Mark private monitoring as unavailable or not provided.
- Do not block the daily pipeline.

If a private session expires:

- Skip that source.
- Log the issue in `history/YYYY-MM/data_sources_log.md`.
- Tell the human which source needs manual login.
- Never ask for credentials.

### Optional Private Source Discovery

Private sources are not limited to URLs pasted by the human. Many useful sources are already inside accounts, communities, and feeds the human already uses:

- Groups, communities, or forums the human has already joined.
- Pages, profiles, KOLs, creators, experts, competitors, or industry voices the human follows.
- YouTube/TikTok/Instagram/X/LinkedIn accounts the human subscribes to or frequently sees.
- Home/news/for-you feeds where the platform recommends topics based on the human's existing interests.

This is often the best source discovery layer because busy humans may follow good sources but not remember all of them, and platform feeds often surface what the human's market is already discussing.

The agent must treat this as an optional, consent-based source discovery workflow, not as automatic surveillance. It must not appear as a separate top-level setup step.

Before using this workflow, the agent must reassure the human:

```text
You are setting up a professional agency-scale system. The first setup takes a little patience, but it is normally a one-time setup.

The data stays on your own computer. I will not ask for passwords, cookies, or tokens, and private data must not be sent outside your computer unless you explicitly approve an export.

Once this is activated, the system can scan every day so you do not miss useful market signals, leads, competitor moves, and content ideas.
```

The agent must ask for explicit consent for each discovery category:

1. `membership_sources`
   - Groups, communities, subreddits, forums, and memberships the human has already joined.
2. `following_sources`
   - Pages, profiles, KOLs, creators, competitors, experts, channels, and companies the human follows or subscribes to.
3. `recommendation_feed_sources`
   - Home feed, news feed, for-you feed, subscriptions feed, groups feed, or other platform-recommended surfaces.

Recommended human-facing question:

```text
Do you want me to discover useful private sources from accounts and communities you already follow or joined?

I can review groups you are already a member of, people/pages/KOLs you follow, channels you subscribe to, and feed recommendations that platforms are already showing you. I will only keep sources related to this client's industry, related industries, target audience, location, pain points, and content pillars. I will not ask for passwords or cookies, and the data stays local on your computer.
```

The human may approve all categories, approve only some, decline, or postpone.

#### Platform Starting URL Registry

These URLs are starting points for the Solo Agency Local Collector to open inside the human's already logged-in browser profile. They are not permanent APIs. Platforms may change paths, redirect, hide surfaces, require login, or A/B test layouts.

If a URL does not work, the agent must mark `platform_url_changed` or `login_required`, continue with other sources, and avoid asking for credentials.

| Platform | Discovery Type | Starting URL | Notes |
|---|---|---|---|
| Facebook | Joined groups | `https://www.facebook.com/groups/joins/?nav_source=tab&ordering=viewer_added` | Use for groups the human has joined. |
| Facebook | Groups feed | `https://www.facebook.com/groups/feed/` | Use for posts from joined groups and group recommendations. |
| Facebook | Home/news feed | `https://www.facebook.com/` | Use only with explicit feed discovery consent. |
| Facebook | Liked/followed pages candidate | `https://www.facebook.com/pages/?category=liked` | Treat as candidate URL; verify in logged-in browser. |
| YouTube | Subscriptions feed | `https://www.youtube.com/feed/subscriptions` | Good for channels the human follows. |
| YouTube | Subscribed channels candidate | `https://www.youtube.com/feed/channels` | Treat as candidate URL; verify in logged-in browser. |
| YouTube | Home recommendations | `https://www.youtube.com/` | Use only with explicit feed discovery consent. |
| X | Home feed | `https://x.com/home` | Use only with explicit feed discovery consent. |
| X | Following list | `https://x.com/{username}/following` | Replace `{username}` only if known or visible; otherwise use profile navigation if available. |
| X | Lists | `https://x.com/i/lists` | Lists may reveal curated sources. |
| X | Communities | `https://x.com/i/communities` | Treat as candidate URL; verify in logged-in browser. |
| LinkedIn | Feed | `https://www.linkedin.com/feed/` | Use only with explicit feed discovery consent. |
| LinkedIn | Following feed candidate | `https://www.linkedin.com/feed/following/` | Treat as candidate URL; verify in logged-in browser. |
| LinkedIn | My Network | `https://www.linkedin.com/mynetwork/` | May surface followed people/pages and recommendations. |
| LinkedIn | Groups candidate | `https://www.linkedin.com/groups/` | Treat as candidate URL; verify in logged-in browser. |
| Instagram | Home feed | `https://www.instagram.com/` | Use only with explicit feed discovery consent. |
| Instagram | Explore | `https://www.instagram.com/explore/` | Use as trend/source discovery, not an active source by itself. |
| Instagram | Following candidate | `https://www.instagram.com/{username}/following/` | Replace `{username}` only if known; may require manual profile navigation. |
| TikTok | Following feed candidate | `https://www.tiktok.com/following` | Treat as candidate URL; verify in logged-in browser. |
| TikTok | For You/home | `https://www.tiktok.com/` | Use only with explicit feed discovery consent. |
| Reddit | Joined subreddits | `https://www.reddit.com/subreddits/mine/` | Use for communities the human joined when visible. |
| Reddit | Home feed | `https://www.reddit.com/` | Use only with explicit feed discovery consent. |

#### Discovery Behavior Rules

The agent must use the Solo Agency Local Collector extension plus the Local Collector app for private source discovery.

The agent must not use Claude Chrome Extension for this workflow.

The agent must not ask for credentials, passwords, OTPs, cookies, raw tokens, or exported browser sessions.

The agent must never open or collect:

- DMs.
- Messenger/chat inboxes.
- Email inboxes.
- Notification centers.
- Payment, account, billing, medical, legal-client, or private dashboard pages unless the human explicitly provides that source for a specific business purpose.
- Anything that requires bypassing access controls.

For private source discovery, the collector should capture only:

- Visible text.
- Source name.
- Source URL.
- Current URL.
- Platform.
- Discovery URL used.
- Visible engagement hints when available.
- Visible author/page/profile/channel candidates.
- Timestamp.

Source discovery pacing:

- Source discovery is not the same as daily content monitoring.
- For joined groups, followed profiles/pages/KOLs, subscribed channels, communities, and similar source lists, the collector must scroll deeply until no new source names/URLs appear for 3 consecutive scrolls.
- Use a hard safety cap, for example 80 scrolls, to avoid infinite scrolling.
- Use `scroll_delay_seconds`: 5.
- Avoid parallel private-source scans.
- Do not stop source discovery at the daily default of 5 scrolls, because that can miss many groups or followed sources lower in the list.

Daily content monitoring pacing after sources are approved:

- Default `max_scrolls_per_source`: 5.
- Absolute maximum `max_scrolls_per_source`: 10.
- `scroll_delay_seconds`: 5.
- Avoid parallel private-source scans.

The agent must treat feeds as discovery surfaces, not permanent sources by themselves.

For example:

- Do not save `https://www.facebook.com/` as a daily private source.
- Instead, use the feed to discover relevant groups, pages, profiles, competitors, KOLs, recurring topics, and lead signals.
- Then propose specific sources for human approval.

#### Source Filtering Rules

The agent must not add everything it finds.

For every candidate source, evaluate:

- Relevance to primary industry.
- Relevance to related industries, only if the bridge back to the offer is logical.
- Relevance to target audience.
- Relevance to target location.
- Pain point match.
- Content pillar match.
- Lead signal potential.
- Competitor intelligence value.
- KOL/trend authority value.
- Posting/activity frequency.
- Noise level.
- Sensitivity/risk level.
- Whether scanning it daily would be platform-respectful.

Classify each candidate source:

- `recommended_daily`
- `recommended_weekly`
- `optional`
- `watch_once`
- `skip_not_relevant`
- `skip_too_broad`
- `skip_too_noisy`
- `skip_sensitive_or_risky`
- `skip_platform_unavailable`

The agent must show the filtered result to the human before adding newly discovered sources to active `private_data_sources`.

#### Discovery Data Model

Use this shape in logs and reports:

```yaml
source_type: joined_group | followed_profile | followed_page | subscribed_channel | followed_company | subreddit | community | recommendation_feed_author | recommendation_feed_topic
platform:
source_name:
source_url:
discovery_category: membership_sources | following_sources | recommendation_feed_sources
discovery_url:
current_url:
captured_at:
why_relevant:
matched_pain_points:
matched_content_pillars:
industry_scope: primary_industry | related_industry
related_industry:
bridge_back_to_primary_offer:
recommended_cadence: daily | weekly | optional | watch_once
risk_level: low | medium | high
approval_status: pending_human_approval | approved | rejected
```

Save discovery outputs under:

```text
daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/history/YYYY-MM/private_source_discovery_YYYY-MM-DD.md
```

When useful, also update:

```text
daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/history/YYYY-MM/new_private_sources_log.md
```

The next HTML report must include a `Private Source Discovery` section when this workflow runs or is pending.

---

## 19. Private Source Access And Failure Protocol

For private sources:

- Use already logged-in browser sessions only.
- Do not request credentials.
- Do not request cookies.
- Do not request OTP.
- Do not attempt to bypass access controls.
- Do not interact socially unless explicitly allowed.

If access works:

- Collect relevant visible data.
- Log the source as checked or collected.

If access fails:

- Skip the source.
- Log `session_expired` or `unavailable`.
- Notify the human through WideCast MCP / Telegram if available.
- Tell the human in the agent UI and notification channel:

`I could not access [source name] because the session appears expired or unavailable. I skipped it for today's run. Please log in manually through the browser/session if you want it included in future runs.`

Continue the pipeline with other sources.

---
