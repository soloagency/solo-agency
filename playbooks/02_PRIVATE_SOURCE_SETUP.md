# Private Data Source Setup

Stage: `02`

## Load Rule

Load when private data sources, manual private URLs, joined groups, Facebook keyword group search, followed profiles/pages/KOLs, subscribed channels, recommendation feeds, private data source discovery, or Local Collector activation are requested, approved, pending, or blocked.

Also load this stage BEFORE asking the step-6 private data source checkpoint question of the one-time setup: the required checkpoint content and its two-part delivery rule live in §6 of this file, so the question cannot be asked correctly without this stage loaded.

If this stage was triggered by a human request to scan, monitor, collect, review, or open a private data source after any amount of conversation drift, first reload `playbooks/PRIVATE_SOURCE_GATE.md`, then reload Stage 8 and Stage 9 before taking action.

## Hard Gates For This Stage

- Manual private data sources and optional private data source discovery are independent.
- If the human has no private data source list, says they do not know what to add, skips the question, or leaves it blank, offer one optional private data source discovery pass before marking the private data source step resolved.
- Ask explicitly whether the human wants to discover candidate private data sources from joined/member communities and followed/subscribed sources, such as Facebook groups, subreddits, Discord/Slack communities, LinkedIn groups/pages, YouTube channels, X lists/communities, and followed KOLs/pages.
- Explain private data sources in plain language before asking for them.
- Explain Local Collector in plain language before asking the human to install or activate it.
- The step-6 checkpoint question uses the two-part delivery in §6: the plain-language explanation FIRST (as normal prose/bullets), then one compact `**[ACTION REQUIRED]**` question with the three reply options. Translated versions must pass the §6 content-completeness checklist; shortening away any checklist item is a Source Preservation violation even if the shorter question reads better.
- Use the Facebook joined-groups URL only with explicit consent.
- Do not use automated approval-gated browser extension flows for unattended collection.
- Never use Claude in Chrome, Claude Chrome Extension, Codex built-in/in-app browser, ChatGPT/Gemini/Grok browser, Playwright/Puppeteer/Selenium, a fresh agent-opened browser profile, or any agent-controlled browser for private data source collection.
- Private data source collection must go through the Solo Agency Local Collector extension plus the Local Collector app only.
- Before any private data source scan, show or internally verify the `Private Data Source Gate planned preflight` roadmap from `playbooks/PRIVATE_SOURCE_GATE.md`.
- Collector success alone is not completion; analyze data and regenerate the report.
- Load Stage 10 before analyzing or reporting lead/competitor opportunities from private data sources.
- If schedule/automation was already configured, any private data source approval, rejection, activation, discovery result, Local Collector repair, or source cadence change must trigger Automation Resync from Stage 4 before claiming the future scheduled run is updated.
- Every human-facing private data source question, discovery approval, recommended-source approval, Local Collector command, Chrome profile/login reminder, and extension `Load unpacked` handoff must use the root playbook `**[ACTION REQUIRED]**` block. Do not hide the actual question inside explanatory text.

## Source Preservation Rule

This file is detailed source material moved from the original monolithic `SOLO_AGENCY_PLAYBOOK.md`.

Do not summarize away requirements, examples, checklists, schemas, protocols, URLs, edge cases, warnings, approval gates, or completion gates. If a downstream agent needs to shorten its response to the human, it may summarize the response, but it must still obey the full requirements in this file.

---

## Latest Delta Override: Discovery Scroll Depth

Source Discovery Mode and Daily Content Monitoring Mode are different.

For Source Discovery Mode, including joined groups, followed sources, subscribed channels, KOLs, communities, and recommendation surfaces, scroll until no new source names or URLs appear for 3 consecutive scrolls, with a hard safety cap of 10 scrolls.

For Daily Content Monitoring Mode, keep the conservative default: 5 scrolls per source, maximum 10, 5 seconds between scrolls, and around 20 daily private data sources or fewer per client.

For the first lead/competitor pass for a client/source set, Stage 10 overrides the daily default: use 10 scrolls per approved private data source when Local Collector is active and account-safety settings allow it. Recurring daily runs return to the 5-scroll default unless configuration says otherwise.

If any older copied section appears to use the daily 5-scroll default for source discovery, this latest delta override wins.

## Group Scan Communication Rule

Every time the agent tells the human it will scan or monitor groups, communities, fanpages, or logged-in social sources, it must state the actual scan depth and where that value comes from.

The same message must include this reminder:

```text
Private collection method: Solo Agency Local Collector only.
I will not use Claude in Chrome, Codex/browser tools, Playwright, or any agent-controlled browser for logged-in sources.
```

For daily content monitoring, the agent must say the human-facing equivalent of:

```text
I will go through each approved group/source one by one and scroll {N} times per source, using the value from the Local Collector configuration. The default is 5 scrolls per source, the maximum is 10, and I will wait about 5 seconds between scrolls.
```

For the first lead/competitor pass, the agent must say the human-facing equivalent of:

```text
I will go through each approved group/source one by one and scroll 10 times per source for the first lead/competitor pass, if the Local Collector configuration and account-safety limits allow it. Future daily runs usually use 5 scrolls per source.
```

The agent must determine `{N}` from the best available source:

1. Read `daily-content-pipeline/collector/collector_config.json` if it exists.
2. If the Local Collector app is running, check `GET http://127.0.0.1:17321/status` and/or `GET /config` when available.
3. If neither source is available, use the documented default: `5` scrolls per source, max `10`, 5-second delay.

For source discovery, do not say "5 scrolls" unless the configured discovery mode explicitly says that. Source Discovery Mode uses the discovery rule: continue until no new source names/URLs appear for 3 consecutive scrolls, with a hard safety cap of 10 scrolls.

If the agent cannot read the actual config, it must be honest:

```text
I cannot read the Local Collector config right now, so I will use the safe default: 5 scrolls per approved source, max 10, with about 5 seconds between scrolls.
```

---

## 6. Private Data Source Rule

The agent must ask the human to provide private data sources they want monitored.

Plain-language definition for humans:

- Private data sources are logged-in/social/community places the human may want monitored later, such as groups, profiles, pages, channels, forums, communities, newsletters, or dashboards that may require the human's account or membership.
- They are different from public data sources. Public data sources are websites, search engines, public articles, public forums, public docs, or public pages the agent can access without logging into the human's account.
- Private data source collection requires explicit permission and the Solo Agency Local Collector. The Local Collector is a local app plus Chrome extension running on the human's computer; it uses the human's already logged-in Chrome session, reads only approved visible pages, and keeps data local by default.

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

Classification tie-breaker (private vs public):

- Any source already on the client's `private_data_sources` list is collector-only, regardless of whether it happens to load when logged out.
- Any social-platform page, profile, group, or channel of the client, or of a monitored competitor, is collector-only, regardless of whether it happens to load when logged out.
- "Public" for agent-browser research means the non-social web: websites, articles, docs, search results, and public news or forums that are not social-platform pages/profiles/groups/channels.
- Reclassifying a source from collector-only to public (or vice versa) requires explicit human approval; the agent must not reclassify on its own.

The checkpoint is delivered in TWO parts, in this order, in the human's language. Translation is required when the human is not chatting in English; dropping content is not allowed — the delivery must stay content-complete per the checklist below.

**Part 1 — plain-language explanation, BEFORE the question, as normal prose or short bullets (not inside the `[ACTION REQUIRED]` block):** the agent must convey ALL of the following, briefly:

- Private data sources are logged-in/social/community places such as competitor profiles, fanpages, Facebook groups, LinkedIn pages, Reddit communities, Discord/Slack communities, niche forums, newsletters, or dashboards that may require the human's account or membership.
- They are different from public data sources such as websites, Google/search results, public articles, and public pages the agent can access without the human's login (already configured and ready to run).
- The human does NOT need to compile this list by hand — that is usually tiring, hard, and incomplete. With permission, the agent can DISCOVER candidate private data sources automatically from the places the human already joined or follows on their own machine — Facebook joined groups, subreddits, followed pages/profiles/KOLs, subscribed channels, community feeds — through the Local Collector, filter the candidates, and present a shortlist for approval before anything is monitored.
- Collection uses the Solo Agency Local Collector: a local app plus Chrome extension on the human's computer. It uses the already logged-in Chrome session, reads approved visible pages only, and keeps data local by default. It never asks for credentials, cookies, passwords, OTPs, or tokens.
- The human must already be a member, follower, subscriber, logged in, or otherwise authorized to view any source they provide, in the Chrome profile where this client's Solo Agency Local Collector extension is installed; one separate Chrome profile per client is recommended, with that client's extension loaded and the relevant social accounts logged in there.
- Collection activates only with the human's permission. For account safety and platform-respectful monitoring, around 20 private data sources or fewer per client is a good daily default; if the human provides more, the agent prioritizes and rotates them.

**Part 2 — the question, immediately after the explanation, as ONE compact `**[ACTION REQUIRED]**` block** (root playbook format), asking which of the three options the human wants:

- provide private data source URLs/lists now, or
- allow one optional discovery pass from places they already joined/follow (the agent filters candidates and asks approval before anything is monitored), or
- postpone and run public data sources only for now.

Content-completeness checklist for this checkpoint (audited in Stage 9 — a checkpoint question missing any item is non-compliant, even if the shorter version reads better):

1. Definition of private data sources with a couple of concrete examples.
2. Contrast with public data sources (what the system can already read without login).
3. What the Local Collector is + data stays local + never asks for passwords/cookies/OTPs/tokens.
4. The already-a-member/logged-in requirement and the per-client Chrome profile recommendation.
5. The hands-free discovery capability: the agent can find candidate sources from places the human already joined/follows, so no hand-compiled list is needed; approval comes before anything is monitored.
6. The three reply options: provide sources / allow discovery / postpone.

### Private Data Source Discovery When The Human Has No List

Most humans do not remember which groups, subreddits, communities, pages, profiles, or channels are worth monitoring at setup time. The agent must treat this as normal.

If the human provides no private data sources, says "I do not know", skips the question, or only gives a vague answer, the agent must not simply mark private data sources as `not_provided` and move on. It must first offer a concise discovery option:

```text
No problem if you do not know which private data sources to add yet. A lot of the best idea, lead, and competitor signals usually live in groups, subreddits, communities, pages, profiles, channels, and feeds you already follow or joined. Do you want me to discover candidate private data sources from the approved places you already belong to or follow, then filter the list and ask you before monitoring anything?
```

The agent must explain that discovery is optional, consent-based, and local:

- It uses only approved logged-in surfaces.
- The human must already have permission to view those surfaces in the client Chrome profile where the extension is installed.
- One separate Chrome profile per client is recommended so each client's extension and social logins stay cleanly separated.
- It requires the Solo Agency Local Collector extension and Local Collector app.
- It does not ask for passwords, cookies, OTPs, tokens, or credentials.
- It does not add every discovered source automatically.
- It filters candidates first, then asks the human to approve, remove, or add sources.
- If the human declines or postpones discovery, the agent can still run public data source research, but the report must note that lead/competitor/community coverage is limited.

Discovery surfaces to offer when relevant:

- Facebook joined groups and groups feed.
- Reddit joined/subscribed subreddits and home feed when approved.
- LinkedIn groups, followed pages, followed people, company pages, and feed when approved.
- YouTube subscriptions and subscribed channels.
- X lists, communities, following list, and home feed when approved.
- Instagram/TikTok followed creators and recommendation feeds when approved.
- Discord/Slack/community forums only when the human explicitly provides/approves the community surface and Local Collector support exists.

The agent should ask one compact approval question, not a long questionnaire, and put that question in a `**[ACTION REQUIRED]**` block:

```text
Do you want me to run private data source discovery from places you already joined or follow, such as Facebook groups, subreddits, followed pages/KOLs, subscribed channels, and community feeds? I will use the Local Collector only, filter candidates, and ask you to approve the shortlist before anything becomes a daily monitored source.
```

If the human says yes:

Timing: this sequence is the step-6 checkpoint's own interactive flow, and its output (the approved source list) is configuration. When the Local Collector and the matching client extension are verified healthy in the CURRENT session — including a setup session — run it NOW, while the human is present to approve the shortlist; the Setup Flow prohibition on scans does not cover this one configuration-gathering pass. Only when the collector is not yet healthy, the human is not present to approve, or the human postpones, record `approved_pending_first_scan` and hand execution to the first Automation Flow run (which then MUST run it or report the exact collector blocker). In a setup session, stop after saving approved sources and resyncing: do not analyze the collected data, generate reports/ideas/drafts from it, or start daily monitoring there.

1. Load `playbooks/PRIVATE_SOURCE_GATE.md`, Stage 8, and Stage 9 before any scan.
2. Activate/setup the Local Collector if it is not installed and healthy. The agent must prepare files and give the human the one-line Terminal/PowerShell command and Chrome extension `Load unpacked` folder path; it must not run setup/start scripts itself. Before giving that command or folder path, run the Stage 8 Source Safety Pre-Check and precede the handoff with one short plain-language safety confirmation line (for example: `I read through the collector's code and confirmed it only runs on your computer and does not send your data anywhere. It is safe to install.`). If the pre-check does not pass, do not give the install command; stop and raise it to the operator.
3. Ask which broad discovery surfaces are approved if not already clear. Keep the question short and default to the most likely safe set for the client, for example Facebook joined groups and Reddit joined/subscribed communities for community-heavy businesses.
4. Run only approved discovery URLs/surfaces.
5. Use Source Discovery Mode: scroll until no new source names/URLs appear for 3 consecutive scrolls, with a hard safety cap of 10 scrolls.
6. Extract candidate source names, URLs, platform, visible description/context, activity hints, topic hints, audience fit, location fit, and risk/noise signals.
7. Filter and classify candidates. A candidate may be classified `recommended_daily`, `recommended_weekly`, `optional`, or `watch_once` ONLY after its relevance to the client's industry/sub-industry, target audience, target location (per the location-weighting rule in Stage 0), and pain points has been scored and recorded in the Discovery Data Model (`target_audience_fit`, `location_fit`, `matched_pain_points`, `industry_scope`). An unscored candidate defaults to `skip_not_relevant` — it must never reach the approval shortlist unscored. Buckets:
   - `recommended_daily`
   - `recommended_weekly`
   - `optional`
   - `watch_once`
   - `skip_not_relevant`
   - `skip_too_broad`
   - `skip_too_noisy`
   - `skip_sensitive_or_risky`
   - `skip_platform_unavailable`
8. Show a compact approval list DIRECTLY IN CHAT as a numbered list (see the In-Chat Numbered Shortlist Rule below), grouped by proposed cadence:
   - `Recommended daily`
   - `Recommended weekly`
   - `Optional`
   - `Skip` (name or count only)
9. For each recommended source, include:
   - source name;
   - source URL when visible;
   - platform;
   - why it matters;
   - matched pain point or content pillar;
   - lead potential;
   - competitor intelligence value;
   - proposed cadence;
   - risk/noise note.
10. Ask the human to approve, remove, or add sources BY NUMBER in a `**[ACTION REQUIRED]**` block before anything is saved as active (per the In-Chat Numbered Shortlist Rule). Do not ask the human to open a `.md` file or report to read or approve the shortlist.
11. Save approved sources to `private_data_sources`.
12. Save unapproved candidates to the discovery log as `pending_human_approval`, `rejected`, or `skipped`.
13. If `daily-content-pipeline/schedule.md`, `daily-content-pipeline/automation/automation_manifest.md`, or any native automation/scheduled task already exists, load Stage 4 and perform Automation Resync. This must update the Client Intelligence Profile, source logs, `schedule.md`, collector config if relevant, automation manifest, scheduled-run prompt/task body, and resync log. Do not tell the human that tomorrow's scheduled run will scan the approved sources until this resync or a clearly logged `automation_prompt_update_pending` state is complete.

### In-Chat Numbered Shortlist Rule

The discovery shortlist is presented for approval DIRECTLY IN THE CHAT, never as a file the human must open. The saved discovery log (`.md`) is a record only, not the approval surface.

- Show the shortlist in chat as a NUMBERED list (`1.`, `2.`, `3.`, ...), one short phone-scannable line per candidate: `{n}. {source name} - {platform} - {proposed cadence} - {one-line why it fits: matched pain point / audience / location fit}`. Group by cadence (Recommended daily, Recommended weekly, Optional) and list Skip candidates by name only or as a count, so the human sees what was filtered out and why.
- End with an `**[ACTION REQUIRED]**` block asking the human to reply BY NUMBER, for example `approve all` / `approve 1-5, 8` / `skip 3, 7` / `add: {url}`. The numbering is what the human replies against, so it must be stable within that message.
- Never tell the human to open a `.md` file, a report, or a saved log to read or approve the shortlist. This is the root "do not bury the question in a Markdown file" rule applied to discovery: the full per-candidate detail is written to the discovery log as a record, but the human-facing approval always happens in chat.
- If the list is long, show Recommended daily and weekly in full and summarize Optional/Skip as counts, but keep every source the human is asked to approve individually numbered.

### Pending-Approval Shortlist Rule (`discovery_completed_pending_approval`)

After a discovery scan produced a shortlist that the human has not yet approved, trimmed, or rejected:

1. Do NOT re-run discovery on later runs while the shortlist is pending. Re-scan only when the human asks, or offer a refresh when the shortlist is older than 14 days — never silently re-scan.
2. EVERY later run (scheduled or manual) must re-surface the pending shortlist as a numbered in-chat list (per the In-Chat Numbered Shortlist Rule) inside an `**[ACTION REQUIRED]**` approval block, until the human resolves it. A pending shortlist buried in an old report or in a `.md` file the human must open is a workflow failure.
3. When the human approves, save the approved sources to `private_data_sources`, log the remaining candidates per the discovery log states, and perform Automation Resync so the next run monitors the approved sources.

If the human says no or not now:

- Mark discovery as `discovery_declined_or_postponed`.
- Continue with public data sources and any manually provided private data sources.
- Include a report note that private community/lead/competitor coverage is limited until private data source discovery or manually provided private data sources are approved.
- If a schedule/automation already exists, perform Automation Resync (Stage 4) recording the decision status (`discovery_declined_or_postponed` / `not_provided`) and the public-only coverage warning, so the scheduled task snapshot reflects the newest state.

### Facebook Member Groups Review

Facebook member groups are one high-value discovery surface inside the broader private data source discovery workflow.

This is a separate discovery path from asking the human to paste private data source URLs manually. Many non-technical humans do not remember all useful groups they have already joined. The agent must offer to discover candidate groups from the human's joined-groups page when private data source discovery is useful and relevant, with explicit permission, then filter the list for relevance.

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

`Do you want me to review the Facebook groups you are already a member of and select only the groups that look useful for this client? These are private data sources because they require your logged-in Facebook account or group membership. If you say yes, I will use your logged-in Chrome session through the Solo Agency Local Collector local app/extension on your computer, open your joined-groups page, and filter groups based on the client's main industry, related industries, audience, location, and pain points/customer problems. I will not ask for credentials, cookies, passwords, OTPs, or tokens. For account safety, I will keep the active daily private data source list conservative, around 20 sources or fewer per client by default, and rotate lower-priority groups when needed.`

If the human agrees:

1. Treat the following URL as the Facebook joined-groups discovery source:

```text
https://www.facebook.com/groups/joins/?nav_source=tab&ordering=viewer_added
```

2. If the Solo Agency Local Collector extension and Local Collector app are not installed and healthy, activate/setup them before attempting this scan. One-time activation requires a human-run setup handoff: the agent prepares files, then the human runs the Local Collector app setup/start command in Terminal/PowerShell outside the AI sandbox and loads the Chrome extension from the absolute runtime folder.
3. Do not use Claude Chrome Extension for this discovery scan.
4. Do not ask the human to paste Facebook cookies, passwords, tokens, or credentials.
5. Use the human's already logged-in Chrome session. If Facebook is logged out, mark `facebook_session_expired` and ask the human to log in manually.
6. Create a manual `run_now` job for the Local Collector to scan the joined-groups discovery URL.
7. Use Source Discovery Mode, not Daily Content Monitoring Mode:
   - set `job_type: "private_data_source_discovery"` or source `purpose: "source_discovery"`;
   - scroll until no new group names/URLs appear for 3 consecutive scrolls;
   - use a hard safety cap of 10 scrolls;
   - use `scroll_delay_seconds`: 5;
   - read visible text and current URLs only.
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
12. Ask the human to approve the recommended groups in a `**[ACTION REQUIRED]**` block before adding them as active `private_data_sources`.
13. After approval, add selected groups to `private_data_sources`.
14. Log skipped groups as not relevant when appropriate.
15. Save the discovery output under:

```text
daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/history/YYYY-MM/facebook_member_groups_review_YYYY-MM-DD.md
```

16. Also add selected or newly discovered group candidates to `New Private Data Sources Detected` in the next report.
17. If schedule/automation already exists, immediately run Automation Resync from Stage 4 so the next scheduled run reads the approved Facebook group list instead of the old pending/private-data-source-skipped snapshot.

If the human declines:

- Do not inspect Facebook groups.
- Continue with other public and private data sources.
- If a schedule/automation already exists, perform Automation Resync (Stage 4) recording the decision status (`discovery_declined_or_postponed` / `not_provided`) and the public-only coverage warning, so the scheduled task snapshot reflects the newest state.

If the human provides no private data sources:

- Offer the broader private data source discovery workflow once before continuing public-only.
- If the human declines or postpones discovery, continue with public data sources only.
- Mark private monitoring as `not_provided` or `discovery_declined_or_postponed`.
- Do not block the daily pipeline, but note that community, lead, and competitor coverage is limited.
- If a schedule/automation already exists, perform Automation Resync (Stage 4) recording the decision status (`discovery_declined_or_postponed` / `not_provided`) and the public-only coverage warning, so the scheduled task snapshot reflects the newest state.

If a private session expires:

- Skip that source.
- Log the issue in `history/YYYY-MM/data_sources_log.md`.
- Tell the human which source needs manual login.
- Never ask for credentials.

### Facebook Keyword Group Search Discovery

Facebook keyword group search is a second Facebook group discovery path. It is for finding new candidate groups by search keyword, not only reviewing groups the human already joined.

Use it when:

- the human explicitly asks to find new Facebook groups;
- the human has no useful group list and approves keyword-based discovery;
- current private data source scans are too noisy and the agent needs cleaner source candidates;
- the client's topic has obvious group-search keywords.

The agent must ask for explicit consent before running it:

```text
Do you want me to find new Facebook groups with keyword search for this client? I will use the Solo Agency Local Collector in the client's Chrome profile, search Facebook groups with keywords that match the client's audience and pain points, scroll 10 times on each search result page, filter out Facebook UI noise and irrelevant results, then show you a shortlist to approve. I will not join groups, request access, or add any group to private data sources unless you approve it and the client Chrome profile can access it.
```

Keyword selection:

- Infer search keywords from the client's industry, sub-industry, target audience, target location, pain points, content pillars, business offer, buying-intent phrases, and public keyword bank.
- Use short, specific phrases. Good examples: `ai tools`, `small business automation`, `real estate investors Austin`, `DUI help Los Angeles`, `California homeowners insurance`.
- Use 1-5 search keywords per discovery pass unless the human explicitly approves a broader pass.
- URL-encode each keyword and use this URL pattern:

```text
https://www.facebook.com/search/groups/?q={url_encoded_keyword}
```

Run rules:

1. Load `playbooks/PRIVATE_SOURCE_GATE.md`, Stage 8, and Stage 9 before any scan.
2. Use only the Solo Agency Local Collector extension plus Local Collector app in the matching client Chrome profile.
3. Do not use Claude in Chrome, Codex/browser tools, Playwright/Puppeteer/Selenium, or any agent-controlled browser.
4. Do not ask for Facebook cookies, passwords, tokens, OTPs, or credentials.
5. Use `job_type: "private_data_source_discovery"` and `purpose: "facebook_group_keyword_search_discovery"`.
6. For each keyword URL, set `scroll_steps: 10` and `scroll_delay_seconds: 5`.
7. Capture only visible search-result information: group name, group URL, result rank/order when visible, visible description/snippet, member/activity hints, category hints, current URL, search keyword, and discovery URL.
8. If Facebook is logged out, checkpointed, blocked, or the page is unavailable, mark the exact blocker such as `facebook_session_expired`, `facebook_checkpoint`, `platform_url_changed`, or `search_results_unavailable`.

Noise filtering:

- Accept a candidate only if it looks like a real Facebook group result, preferably with a group name and a URL such as `/groups/...`.
- Ignore Facebook navigation, tabs, filters, buttons, ads/sponsored blocks, people/pages/posts/events results, generic UI labels, repeated headers, and any text that is not a group candidate.
- Reject or down-rank groups that are too broad, spammy, low-signal, sensitive/risky, unrelated to the client's audience, unrelated to pain points, or not accessible from the client Chrome profile.
- Do not treat a keyword search result as an active private data source until the human approves it.
- Do not join a group, request access, message admins, or follow pages as part of this workflow.

Classify candidates as:

- `recommended_daily`
- `recommended_weekly`
- `optional`
- `watch_once`
- `skip_not_relevant`
- `skip_too_broad`
- `skip_too_noisy`
- `skip_sensitive_or_risky`
- `skip_platform_unavailable`

For every candidate, save:

- `source_type: facebook_group_search_result`
- `discovery_category: keyword_search_sources`
- `search_keyword`
- `search_url`
- `result_rank`
- `profile_or_group_url`
- `membership_status: unknown | joined | not_joined | public_visible | requires_join | unavailable`
- `why_relevant`
- `matched_pain_points`
- `related_content_pillar`
- `target_audience_fit`
- `location_fit`
- `noise_level`
- `risk_level`
- `classification`
- `approval_status`

Show the human a short `Facebook Keyword Group Search Review` before saving anything as active:

- keywords searched;
- 10-scroll status for each keyword;
- candidate groups found;
- recommended groups and why they fit;
- skipped/noisy result examples;
- access/membership notes;
- which groups need human approval or joining before scheduled monitoring can use them.

Show the recommended groups DIRECTLY IN CHAT as a numbered list and ask the human to approve them by number (per the In-Chat Numbered Shortlist Rule) before adding them as active `private_data_sources`; the saved review `.md` is a record, not the approval surface - do not ask the human to open it to approve. If the group requires membership or access and the client Chrome profile cannot view it yet, save it as `pending_human_approval` or `pending_private_activation`, not active.

Save the discovery output under:

```text
daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/history/YYYY-MM/facebook_group_keyword_search_review_YYYY-MM-DD.md
```

If schedule/automation already exists and the human approves new groups, run Automation Resync from Stage 4 so the next scheduled run reads the approved group list instead of the old source state.

### Optional Private Data Source Discovery

Private data sources are not limited to URLs pasted by the human. Many useful sources are already inside accounts, communities, and feeds the human already uses:

- Groups, communities, or forums the human has already joined.
- Pages, profiles, KOLs, creators, experts, competitors, or industry voices the human follows.
- YouTube/TikTok/Instagram/X/LinkedIn accounts the human subscribes to or frequently sees.
- Home/news/for-you feeds where the platform recommends topics based on the human's existing interests.

This is often the best private data source discovery layer because busy humans may follow good sources but not remember all of them, and platform feeds often surface what the human's market is already discussing.

The agent must treat this as an optional, consent-based private data source discovery workflow, not as automatic surveillance. It must not appear as a separate top-level setup step.

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
Do you want me to discover useful private data sources from accounts and communities you already follow or joined?

Private data sources are logged-in/social/community places such as groups, profiles, pages, channels, forums, or feeds that may require your account or membership. I can review groups you are already a member of, people/pages/KOLs you follow, channels you subscribe to, and feed recommendations that platforms are already showing you. I will only keep sources related to this client's industry, related industries, target audience, location, pain points/customer problems, and content pillars/main content themes. I will not ask for passwords, cookies, OTPs, or tokens, and the data stays local on your computer by default.
```

The human may approve all categories, approve only some, decline, or postpone.

#### Platform Starting URL Registry

These URLs are starting points for the Solo Agency Local Collector to open inside the human's already logged-in browser profile. They are not permanent APIs. Platforms may change paths, redirect, hide surfaces, require login, or A/B test layouts.

If a URL does not work, the agent must mark `platform_url_changed` or `login_required`, continue with other sources, and avoid asking for credentials.

| Platform | Discovery Type | Starting URL | Notes |
|---|---|---|---|
| Facebook | Joined groups | `https://www.facebook.com/groups/joins/?nav_source=tab&ordering=viewer_added` | Use for groups the human has joined. |
| Facebook | Keyword group search | `https://www.facebook.com/search/groups/?q={url_encoded_keyword}` | Use only with explicit keyword-search discovery consent. Scroll 10 times per keyword, filter group results, and ask approval before adding any group. |
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

The agent must use the Solo Agency Local Collector extension plus the Local Collector app for private data source discovery.

The agent must not use Claude Chrome Extension for this workflow.

The agent must not ask for credentials, passwords, OTPs, cookies, raw tokens, or exported browser sessions.

The agent must never open or collect:

- DMs.
- Messenger/chat inboxes.
- Email inboxes.
- Notification centers.
- Payment, account, billing, medical, legal-client, or private dashboard pages unless the human explicitly provides that source for a specific business purpose.
- Anything that requires bypassing access controls.

For private data source discovery, the collector should capture only:

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
- For Facebook keyword group search discovery at `https://www.facebook.com/search/groups/?q={url_encoded_keyword}`, use exactly 10 scrolls per keyword unless the human explicitly approves a different value. This is a bounded search-results pass, not a full joined-source inventory.
- Use a hard safety cap of 10 scrolls to avoid infinite scrolling and account-safety risk.
- Discovery scrolls must be real page-sized scrolls through the active list/page, not tiny nudges. If the output finds only the first few dozen sources despite a high scroll cap, inspect `scroll_debug`, `scroll_count`, and `scroll_stopped_reason`, then retry after updating/reloading the extension.
- When creating a `run_now` job for discovery, mark the job/source with a discovery indicator, such as `job_type: "private_data_source_discovery"`, `purpose: "source_discovery"`, or a discovery URL like the Facebook joined-groups URL, while still keeping the 10-scroll hard cap.
- Use `scroll_delay_seconds`: 5.
- Avoid parallel private data source scans.
- Do not stop source discovery at the daily default of 5 scrolls, because that can miss many groups or followed sources lower in the list.

Daily content monitoring pacing after sources are approved:

- Default `max_scrolls_per_source`: 5.
- Absolute maximum `max_scrolls_per_source`: 10.
- `scroll_delay_seconds`: 5.
- Avoid parallel private data source scans.

The agent must treat feeds as discovery surfaces, not permanent sources by themselves.

For example:

- Do not save `https://www.facebook.com/` as a daily private data source.
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
target_audience_fit: strong | partial | weak | none
location_fit: match | national_or_location_independent | wrong_location | not_applicable
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

The next HTML report must include a `Private Data Source Discovery` section when this workflow runs or is pending.

---

## 19. Private Data Source Access And Failure Protocol

For private data sources:

- Use only the human's already logged-in Chrome session as accessed through the Solo Agency Local Collector extension plus Local Collector app.
- Do not use Claude in Chrome, Codex/browser tools, Playwright/Puppeteer/Selenium, or any agent-controlled browser.
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
- Notify the human through the configured provider notification channel if available, preferably WideCast OpenAPI Telegram/email fallback for the current client.
- Tell the human in the agent UI and notification channel:

`I could not access [source name] because the session appears expired or unavailable. I skipped it for today's run. Please log in manually through the browser/session if you want it included in future runs.`

Continue the pipeline with other sources.

---
