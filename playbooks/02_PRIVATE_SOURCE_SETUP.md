# Private Data Source Setup

Stage: `02`

## Load Rule

Load when private data sources, manual private URLs, joined groups, Facebook keyword group search, followed profiles/pages/KOLs, subscribed channels, recommendation feeds, private data source discovery, or Local Collector activation are requested, approved, pending, or blocked.

Also load this stage before the first run executes joined-places discovery — step 7 of the one-time setup asks only the first-run question.

If this stage was triggered by a human request to scan, monitor, collect, review, or open a private data source after any amount of conversation drift, first reload `playbooks/PRIVATE_SOURCE_GATE.md`, then reload Stage 8 and Stage 9 before taking action.

## Hard Gates For This Stage

- Manual private data sources and optional private data source discovery are independent.
- If the human has no private data source list, says they do not know what to add, skips the question, or leaves it blank, offer one optional private data source discovery pass before marking the private data source step resolved.
- Ask explicitly whether the human wants to discover candidate private data sources from joined/member communities and followed/subscribed sources, such as Facebook groups, subreddits, Discord/Slack communities, LinkedIn groups/pages, YouTube channels, X lists/communities, and followed KOLs/pages.
- Explain private data sources in plain language before asking for them.
- Explain Local Collector in plain language before asking the human to install or activate it.
- Step 7 of the one-time setup asks only the first-run question (`playbooks/SETUP_FLOW_ENTRYPOINT.md`, "Step 7 — Chạy lượt đầu"); there is no source checkpoint, no reply options and no checklist to preserve — sources are monitored automatically (Group Potential Rule) and paused on the Sources page.
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

## Three Distinct Caps: Discovery Capture vs Shortlist vs Active Monitoring

The number "about 20 sources per client" is the ACTIVE DAILY MONITORING cap only (an account-safety limit on how many sources are scanned every day). It must NEVER be used as the discovery-capture limit. Three separate caps apply to three separate stages:

1. **Discovery capture (broad).** When discovering candidate sources, capture as MANY joined groups / followed communities / subscribed sources as the scroll safety allows — the whole reachable list, not 20. The discovery job's `pacing.max_sources` must be set high enough to cover the client's full joined/followed list (for a heavy-membership account, hundreds), so filtering can see everything before narrowing. Setting the discovery job's `max_sources` to ~20 is a defect: it truncates capture to the first 20 and hides the rest.
2. **Shortlist (filtered).** The relevance filter narrows the full captured set down to a proposed shortlist for approval. Filtering happens over ALL captured candidates, never a truncated sample.
3. **Active daily monitoring (~20).** Only the APPROVED, ACTIVE sources that get scanned every day are held to ~20 per client (account safety); extras are classified weekly/optional and rotated. This is the only place the ~20 belongs.

Capture completeness and honesty:

- Record `capture_status` (`complete` | `capped_incomplete`) and the real `captured_count` for each discovery surface. A scan is `complete` only when it reached the stop condition (no new sources for 3 consecutive scrolls) or the true end of the list; if it stopped because it hit the scroll cap or `max_sources` before the end, it is `capped_incomplete`.
- When `capped_incomplete`, tell the human plainly that the capture is partial ("captured N so far; more joined sources exist beyond this"), and never present the shortlist as if it covered all joined groups. Offer to continue discovery in a later pass and/or run a targeted search.
- Reconcile before reporting: the number of candidates you classified and the counts you tell the human must match the actual captured records on disk (`new_private_sources.jsonl` and the surface snapshots). Never state a capture count you cannot point to in the captured data — do not invent a total.

Targeted named-source search:

- When the human names specific sources (for example a particular group by name) that are not in the capture, or whenever capture is `capped_incomplete`, run a targeted Local Collector search for those exact names plus profile-derived keywords (industry / sub-industry / target location / pain points) instead of speculating about why they are missing. Facebook keyword group search (`https://www.facebook.com/search/groups/?q={url_encoded_keyword}`) is the right tool for locating a named group that a joined-list scroll did not reach.


## Group Scan Communication Rule

Every time the agent tells the human it will scan or monitor groups, communities, fanpages, or logged-in social sources, it must state the actual scan depth and where that value comes from.

The same message must include this reminder:

```text
Collection method for sources that need a login: Solo Agency Local Collector only.
I will not use Claude in Chrome, Codex/browser tools, Playwright, or any agent-controlled browser for those sources.
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

Step 7 is the first-run question (see `playbooks/SETUP_FLOW_ENTRYPOINT.md`): one compact yes/no ask — may the agent run the first report now — in the human's language. There is no source list to review at this checkpoint and no approval step: sources are never approved. The agent picks what to watch itself (Group Potential Rule, `playbooks/10_LEAD_COMPETITOR_DETECTION.md`) as part of that first run, and the human's only lever afterward is pausing or resuming a monitored source on the Sources page.

### Private Data Source Discovery When The Human Has No List

Most humans do not remember which groups, subreddits, communities, pages, profiles, or channels are worth monitoring at setup time. The agent must treat this as normal.

If the human provides no private data sources, says "I do not know", skips the question, or only gives a vague answer, the agent must not simply mark private data sources as `not_provided` and move on. It must first offer a concise discovery option:

```text
No problem if you do not know which custom sources to add yet. A lot of the best idea, lead, and competitor signals usually live in groups, subreddits, communities, pages, profiles, channels, and feeds you already follow or joined. I already discover candidate sources from the places you belong to or follow, filter the list, and start monitoring the ones worth watching myself — you can pause any of them afterward on the Sources page.
```

The agent must explain that discovery is optional, consent-based, and local:

- It uses only approved logged-in surfaces.
- The human must already have permission to view those surfaces in the client Chrome profile where the extension is installed.
- One separate Chrome profile per client is recommended so each client's extension and social logins stay cleanly separated.
- It requires the Solo Agency Local Collector extension and Local Collector app.
- It does not ask for passwords, cookies, OTPs, tokens, or credentials.
- It does not add every discovered source automatically.
- It filters candidates first, then registers the ones worth watching as monitored sources automatically — no approval step.
- If the human declines or postpones discovery, the agent can still run public data source research, but the report must note that lead/competitor/community coverage is limited.

Discovery surfaces to offer when relevant:

- Facebook joined groups and groups feed.
- Reddit joined/subscribed subreddits and home feed.
- LinkedIn groups, followed pages, followed people, company pages, and feed.
- YouTube subscriptions and subscribed channels.
- X lists, communities, following list, and home feed.
- Instagram/TikTok followed creators and recommendation feeds.
- Discord/Slack/community forums only when the human explicitly names the community surface and Local Collector support exists.

No separate question is asked here. Step 7's single yes (`playbooks/SETUP_FLOW_ENTRYPOINT.md`) already covers every discovery category — Facebook groups, subreddits, followed pages/KOLs, subscribed channels, and community feeds alike: the agent records `approved_pending_first_scan` for all of them at that one yes, nothing more to ask.

Timing: this is `private_data_source_discovery` (Stage 2), and it never runs inside the setup chat. Recording `approved_pending_first_scan` at the step-7 yes is always correct in a setup session; the first Automation Flow run then executes it (`playbooks/SCHEDULED_RUN_ENTRYPOINT.md`, step 12A) and judges every result with the Group Potential Rule (`playbooks/10_LEAD_COMPETITOR_DETECTION.md`), registering it straight into monitored sources — no shortlist, no approval. In a setup session, stop after recording the status: do not scan, analyze collected data, generate reports/ideas/drafts from it, or start daily monitoring there.

1. Load `playbooks/PRIVATE_SOURCE_GATE.md`, Stage 8, and Stage 9 before any scan.
2. The Local Collector bridge and this client's extension were already installed at setup step 4 (Kết nối Facebook, Instagram and X — `SOLO_AGENCY_PLAYBOOK.md`, Mandatory Setup Flow; `playbooks/SETUP_FLOW_ENTRYPOINT.md`, "Kết nối Facebook, Instagram and X (step 4)"). This checkpoint does not install anything; check current health with `GET http://127.0.0.1:17321/status` and `extension_health` instead. If the bridge or extension is unhealthy here (install never completed, or it went stale since step 4), repeat step 4's own install flow — the local/remote rule stays the same one used there — rather than re-deriving a new install path here; run the Stage 8 Source Safety Pre-Check first if giving/re-running any install command.
3. Use every discovery surface relevant to the client automatically — no question needed, all categories were recorded `approved_pending_first_scan` at the step-7 yes — defaulting to the most likely set for the client, for example Facebook joined groups and Reddit joined/subscribed communities for community-heavy businesses.
4. Run all relevant discovery URLs/surfaces.
5. Use Source Discovery Mode: scroll until no new source names/URLs appear for 3 consecutive scrolls, with a hard safety cap of 10 scrolls.
6. Extract candidate source names, URLs, platform, visible description/context, activity hints, topic hints, audience fit, location fit, and risk/noise signals.
7. Filter and classify candidates. A candidate may be classified `recommended_daily`, `recommended_weekly`, `optional`, or `watch_once` ONLY after its relevance to the client's industry/sub-industry, target audience, target location (per the location-weighting rule in Stage 0), and pain points has been scored and recorded in the Discovery Data Model (`target_audience_fit`, `location_fit`, `matched_pain_points`, `industry_scope`). An unscored candidate defaults to `skip_not_relevant` — it must never be registered unscored. Buckets:
   - `recommended_daily`
   - `recommended_weekly`
   - `optional`
   - `watch_once`
   - `skip_not_relevant`
   - `skip_too_broad`
   - `skip_too_noisy`
   - `skip_sensitive_or_risky`
   - `skip_individual_profile` (an individual professional is a LEAD, not a monitoring source; see the Source-Type Preference rule)
   - `skip_platform_unavailable`
8. Record what was found, grouped by proposed cadence (no approval list — sources are never approved):
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
10. Register `recommended_daily`/`recommended_weekly`/`optional` candidates as monitored sources automatically — nothing waits on a human decision. List newly monitored sources in the run reply with their potential, reason, and the Sources page link; the human's only lever is pausing one there.
11. NORMALIZE every source URL before saving: `tools/solo_tool source-registry normalize --url U1 --url U2 ...` (no `--pipeline` needed). Operators paste whatever the address bar held — trailing slash, `?ref=...` junk, `m.facebook.com`, an About sub-tab — and ONLY the returned `clean_url` may be written anywhere (`private_data_sources`, `collector_config.json`, the registry): a Facebook group is always stored as exactly `https://www.facebook.com/groups/<name>`. A `no_derivable_identity` result (an opaque share/redirector link) is never stored — ask the human for the real page URL. Then save the sources to `private_data_sources`, and register each one in the cross-client source registry: `tools/solo_tool source-registry --pipeline {setup-root}/daily-content-pipeline register --client {client_slug} --url U --platform P --source-type T --kind private --cadence C --priority P2`. Scope rule — sharing is the AGENCY NORM, not a choice: every third-party source (industry groups, competitor pages, communities) is `shared`, MANDATORY — one scan serves every subscribed client; that is the operating model (operator ruling 2026-08-14). `exclusive` has exactly ONE trigger: the source IS the client's own asset (their own page, group, channel, or website). Nothing else qualifies. Worries about two clients contacting the same person NEVER justify exclusive — that concern is handled at the LEAD level by playbook 10's shared-source collision flag, not by refusing to share scan data. An agent may never invent a scope policy: a `scope_policy`/`scope_rationale` note in a profile that does not quote the operator VERBATIM with a date does not bind anyone — treat it as drift, restore the class rule (`set-scope --scope shared`), delete the invented note, and record the correction in the run reply. Register each Facebook group under ONE canonical URL form (prefer the group page's vanity URL) — a numeric-ID URL and a vanity URL of the same group do not merge automatically.
12. Save `skip_*` candidates to the discovery log as `not_selected`, with the reason.
13. If `daily-content-pipeline/schedule.md`, `daily-content-pipeline/automation/automation_manifest.md`, or any native automation/scheduled task already exists, load Stage 4 and perform Automation Resync. This must update the Client Intelligence Profile, source logs, `schedule.md`, collector config if relevant, automation manifest, scheduled-run prompt/task body, and resync log. Do not tell the human that tomorrow's scheduled run will scan the monitored sources until this resync or a clearly logged `automation_prompt_update_pending` state is complete.

### Monitored sources are listed, never shortlisted

Newly monitored sources are listed in the run reply with their potential, the reason, and the Sources page link — no shortlist file, no `**[ACTION REQUIRED]**` approval block, and nothing re-surfaced on later runs waiting for a decision. The human's only lever is pausing one of them on the Sources page.

If the human says no or not now:

- Mark discovery as `declined`.
- Continue with public data sources and any manually provided private data sources.
- Include a report note that private community/lead/competitor coverage is limited until private data source discovery or manually provided private data sources are approved.
- If a schedule/automation already exists, perform Automation Resync (Stage 4) recording the decision status (`declined` / `not_provided`) and the public-only coverage warning, so the scheduled task snapshot reflects the newest state.

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

No question is asked: the step-7 yes already covers this list, and reading it is part of the first run
(and of the next run after Facebook is connected later). Mechanics:

1. Treat the following URL as the Facebook joined-groups discovery source:

```text
https://www.facebook.com/groups/joins/?nav_source=tab&ordering=viewer_added
```

2. The Solo Agency Local Collector extension and Local Collector app were already installed at setup step 4
   (Kết nối Facebook, Instagram and X). If either is not healthy, the run skips this list and the awareness
   line says so; nothing is asked.
3. Do not use Claude Chrome Extension for this discovery scan.
4. Do not ask the human to paste Facebook cookies, passwords, tokens, or credentials.
5. Use the human's already logged-in Chrome session. If Facebook is logged out, mark `facebook_session_expired`
   and let the Login Reminder handle it.
6. Create a `run_now` job for the Local Collector to scan the joined-groups discovery URL.
7. Use Source Discovery Mode, not Daily Content Monitoring Mode: `job_type: "private_data_source_discovery"`
   or source `purpose: "source_discovery"`; scroll until no new group names/URLs appear for 3 consecutive
   scrolls; hard safety cap of 10 scrolls; pacing per the Pacing Rule (random 5–10 s); read visible text and
   current URLs only.
8. Review visible group names, group URLs, descriptions, category hints, membership/context hints, and any
   visible preview text.
9. Judge every group with the Group Potential Rule (`playbooks/10_LEAD_COMPETITOR_DETECTION.md`): `high` and
   `medium` → `tool source-registry add --client <slug> --url <u> --platform facebook --source-type group
   --origin discovered --state active --potential <high|medium> --reason "<one line>"`; `low` → the same
   command with `--state not_selected`. The account is a member of every group on this list, so none is
   `no_access`.
10. The run reply lists the newly monitored groups (name, potential, reason) with the Sources page link
    (`/ui/{client_slug}/sources?tab=discovered`); the human pauses any there. No approval block, no cadence
    question — the registry plan decides what is scanned each run (up to 20 groups, leads-ranked).
11. Save the review under:

```text
daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/history/YYYY-MM/facebook_member_groups_review_YYYY-MM-DD.md
```

    with the monitored and not-selected groups and their reasons.
12. Also list newly monitored groups under `New Private Data Sources Detected` in the next report.
13. If schedule/automation already exists, run Automation Resync from Stage 4 so the next scheduled run reads
    the registry plan.

### Facebook Keyword Group Search Discovery

Facebook keyword group search is a second Facebook group discovery path. It is for finding new candidate groups by search keyword, not only reviewing groups the human already joined.

Use it when:

- the human explicitly asks to find new Facebook groups;
- the human has no useful group list and keyword-based discovery would help;
- current private data source scans are too noisy and the agent needs cleaner source candidates;
- the client's topic has obvious group-search keywords.

No separate consent question: this is the same keyword search the Social Discovery Pass runs automatically (`playbooks/10_LEAD_COMPETITOR_DETECTION.md`), so results here go through the same Group Potential Rule and monitored-source registration — no shortlist, no approval. Private groups the client Chrome profile already belongs to are read the same as public groups, since the account already has access; groups it has not joined are only listed as `no_access` (worth joining) and are never joined, requested, or added as a source by the agent.

Keyword selection:

- The primary source of group-search keywords is now `tool public-keywords --pipeline {setup-root}/daily-content-pipeline --client {client_slug} plan --kind discovery` — the `community_discovery` terms (niche + location, 2-6 words, dateless) staged for the Facebook discovery plan. Fall back to inferring from the client's industry, sub-industry, target audience, target location, pain points, content pillars, business offer, buying-intent phrases, and the public keyword bank's SHORT rungs (`tool public-keywords list`, terms of 1-3 words) only when the discovery kind is empty — because a Facebook search box matches close to literally and a nine-word question returns nothing there.
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
- Do not treat a keyword search result as an active private data source until the Group Potential Rule scores it `active`.
- Do not join a group, request access, message admins, or follow pages as part of this workflow.

Classify candidates as (prefer groups/communities and reputable organization pages; default-skip individual personal profiles per the Source-Type Preference rule):

- `recommended_daily`
- `recommended_weekly`
- `optional`
- `watch_once`
- `skip_not_relevant`
- `skip_too_broad`
- `skip_too_noisy`
- `skip_sensitive_or_risky`
- `skip_individual_profile`
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
- `potential: high | medium | low` (Group Potential Rule) with one-line `potential_reason`
- `state: active | not_selected | no_access`

Show the human a short `Facebook Keyword Group Search Review` before saving anything as active:

- keywords searched;
- 10-scroll status for each keyword;
- candidate groups found;
- monitored groups and why they fit;
- skipped/noisy result examples;
- access/membership notes;
- which groups the client Chrome profile cannot yet access (worth joining, never joined by the agent).

List the groups registered `active` (high or medium potential) in the run reply with their potential and reason — no numbered shortlist, no approval; the saved review `.md` is a record only. If the group requires membership or access the client Chrome profile does not have yet, register it as `no_access` (worth joining, never joined by the agent), not active.

Save the discovery output under:

```text
daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/history/YYYY-MM/facebook_group_keyword_search_review_YYYY-MM-DD.md
```

If schedule/automation already exists and new groups were registered active, run Automation Resync from Stage 4 so the next scheduled run reads the monitored group list instead of the old source state.

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
Do you want me to discover useful custom sources from accounts and communities you already follow or joined?

Some of these are logged-in/social/community places such as groups, profiles, pages, channels, forums, or feeds that may require your account or membership. I can review groups you are already a member of, people/pages/KOLs you follow, channels you subscribe to, and feed recommendations that platforms are already showing you. I will only keep sources related to this client's industry, related industries, target audience, location, pain points/customer problems, and content pillars/main content themes. I will not ask for passwords, cookies, OTPs, or tokens, and the data stays local on your computer by default.
```

The human may approve all categories, approve only some, decline, or postpone.

#### Platform Starting URL Registry

These URLs are starting points for the Solo Agency Local Collector to open inside the human's already logged-in browser profile. They are not permanent APIs. Platforms may change paths, redirect, hide surfaces, require login, or A/B test layouts.

If a URL does not work, the agent must mark `platform_url_changed` or `login_required`, continue with other sources, and avoid asking for credentials.

| Platform | Discovery Type | Starting URL | Notes |
|---|---|---|---|
| Facebook | Joined groups | `https://www.facebook.com/groups/joins/?nav_source=tab&ordering=viewer_added` | Use for groups the human has joined. |
| Facebook | Keyword group search | `https://www.facebook.com/search/groups/?q={url_encoded_keyword}` | Scroll 10 times per keyword, filter group results, and register each one through the Group Potential Rule — no approval before adding a group. |
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

Groups found by the Facebook leg of the Social Discovery Pass (`playbooks/10_LEAD_COMPETITOR_DETECTION.md`,
"Social Discovery Pass (step 11C of the daily run)") are registered the same way as the keyword-search
row above: readable groups (public, or private where the account is already a member) are scanned
read-only, scored by the Group Potential Rule, and `high`/`medium` potential is promoted into standing
daily monitoring automatically (`state: active`) — no human decision. `low` potential is recorded
`state: not_selected` with the reason; groups the account cannot read are recorded `state: no_access`
and are never joined by the agent.

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
- Competitor intelligence value (a competitor is fair game as an ORGANIZATION page, not as an individual's personal profile).
- Source type: is this a group/community or a reputable organization page (preferred), or an individual personal profile (default-skip; see the Source-Type Preference rule)?
- KOL/trend authority value (only high-reach authority accounts clear the individual-profile skip).
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
- `skip_individual_profile`
- `skip_platform_unavailable`

The agent must show the filtered result to the human before adding newly discovered sources to active `private_data_sources`.

#### Source-Type Preference: groups/communities and reputable organizations, not individual profiles

A private data source is a place that produces steady, multi-person signal — not one person's feed. When selecting and classifying candidates, prefer, in this order:

1. **Groups and communities** (Facebook groups, subreddits, LinkedIn groups, Discord/Slack communities, niche forums) where many of the target audience gather. Highest value: fresh pains, questions, content ideas, and many leads per single scan.
2. **Reputable organization pages/websites**: industry associations, competitor companies/agencies, trade publications, and official brand/company pages that publish steady on-vertical content.

**Default-skip individual personal profiles/accounts** — classify them `skip_individual_profile`. An individual professional who fits the ICP is a LEAD (Stage 10 surfaces them FROM the group/community scans), not a monitoring source; a single person's feed carries too little daily signal to monitor. This includes personal Facebook pages, personal LinkedIn profiles, and single-person accounts, even when the person is on-vertical.

**Narrow exception:** keep an individual account ONLY when it is a high-reach authority/KOL or an official brand account with large, steady, on-vertical output that a group cannot replace — and only when the agent records the reach/authority evidence (follower scale, posting cadence, on-vertical) that justifies it. A modest personal profile never qualifies; when in doubt, skip it as a source and let it surface as a lead instead.

Weight the discovery surfaces accordingly: prioritize joined-groups and keyword group/community search over following-individual-profile surfaces. A shortlist that is mostly individual profiles is a filtering failure — re-filter toward groups, communities, and reputable organization pages.


#### Discovery Data Model

Use this shape in logs and reports:

```yaml
source_type: joined_group | followed_page | subscribed_channel | followed_company | subreddit | community | organization_page | recommendation_feed_topic   # prefer group/community/organization types; an individual profile is default-skip_individual_profile (a lead, not a source) unless it clears the high-reach-authority exception
platform:
source_name:
source_url:
discovery_category: membership_sources | following_sources | recommendation_feed_sources
discovery_url:
current_url:
captured_at:
capture_status: complete | capped_incomplete   # per surface: complete only if it reached the stop condition or true end of list
captured_count:                                   # real number captured for this surface (must match the records on disk)
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
state: active | not_selected | no_access   # automatic: recommended_daily/weekly/optional -> active, skip_* -> not_selected, unreadable/not-joined -> no_access; no human approval
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
