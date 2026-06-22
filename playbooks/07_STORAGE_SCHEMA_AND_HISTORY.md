# Storage Schema And History

Stage: `07`

## Load Rule

Load when creating folders, saving profiles, updating logs, reading history, avoiding duplicate ideas, adding clients, or tracking published content and analytics.

## Hard Gates For This Stage

- Use the dedicated root folder `daily-content-pipeline/`.
- Use `client_intelligence_profile.md` as the canonical profile concept; do not use `ABC.md`.
- Store Markdown internally and HTML for humans.
- Track history to avoid duplicate ideas.
- Keep analytics, comments, learning, lead, competitor, source, and published-content logs.

## Source Preservation Rule

This file is detailed source material moved from the original monolithic `SOLO_AGENCY_PLAYBOOK.md`.

Do not summarize away requirements, examples, checklists, schemas, protocols, URLs, edge cases, warnings, approval gates, or completion gates. If a downstream agent needs to shorten its response to the human, it may summarize the response, but it must still obey the full requirements in this file.

---

## Canonical Profile Name Clarification

Use `client_intelligence_profile.md` as the canonical profile concept and schema name.

For multi-client slugged folders, a slugged profile filename may still be used when needed for uniqueness, but it must represent the Client Intelligence Profile schema. Do not use vague names such as `ABC.md`.

---

## 7. Folder Structure

Use one agency root folder:

```text
{agency_root}/
```

Use one folder per client/business/location:

```text
{agency_root}/
  solo-agency/                         # downloaded toolkit/source repo, no client data
  solo-agency-local-collector/         # runtime app + Chrome extension only
    downloads/
      collector-bridge-binaries-0.1.0.zip
      chrome-extension-collector-root-0.1.0.zip
      SHA256SUMS
    bin/
      collector-bridge-{os}-{arch}
    LOAD_THIS_EXTENSION_IN_CHROME/
      manifest.json
      background.js
      popup.html
      popup.js
    setup_collector.sh
    collector.pid
    collector.log
  daily-content-pipeline/              # data/config/output only
  clients_index.md
  schedule.md
  notifications/
    notification_log.md
  collector/
    collector_setup_status.md
    collector_config.json
    jobs/
      YYYY-MM/
        YYYY-MM-DD_client_slug.json
    inbox/
      YYYY-MM/
        YYYY-MM-DD_client_slug/
          collector_status.json
          private_data_points.jsonl
          leads.jsonl
          competitors.jsonl
          new_private_sources.jsonl
          source_status.jsonl
          snapshots/
  browser_profiles/
    {source_slug}/
  outputs/
    YYYY-MM/
      YYYY-MM-DD_master_digest.md
      YYYY-MM-DD_master_digest.html
    latest_master_digest.md
    latest_master_digest.html
  clients/
      {client_slug}/
        {business_slug}_{location_slug}/
        client_profile_{client_slug}_{business_slug}_{location_slug}.md
        strategy/
          offer_map.md
          brand_voice.md
          content_pillars.md
          funnel_map.md
        calendar/
          content_calendar.md
        approvals/
          approval_log.md
        assets/
          asset_index.md
        publishing/
          publishing_log.md
        analytics/
          metrics_log.md
        reports/
          YYYY-MM_report.md
        experiments/
          experiment_backlog.md
        history/
          YYYY-MM/
            content_log.md
            data_sources_log.md
            lead_log.md
            competitor_log.md
            new_private_sources_log.md
        outputs/
          YYYY-MM/
            YYYY-MM-DD.md
            YYYY-MM-DD.html
          latest.md
          latest.html
```

Examples:

```text
daily-content-pipeline/
  clients_index.md
  schedule.md
  notifications/
    notification_log.md
  collector/
    downloads/
    bin/
      collector-bridge-darwin-arm64
      collector-bridge-windows-amd64.exe
      collector-bridge-linux-amd64
    chrome-extension/
    jobs/
      YYYY-MM/
    inbox/
      YYYY-MM/
  browser_profiles/
    facebook/
    linkedin/
  outputs/
    2026-06/
      2026-06-19_master_digest.md
      2026-06-19_master_digest.html
    latest_master_digest.md
    latest_master_digest.html
  clients/
    smith-law/
      dui_los-angeles/
        client_profile_smith-law_dui_los-angeles.md
        strategy/
          content_pillars.md
          funnel_map.md
        calendar/
          content_calendar.md
        approvals/
          approval_log.md
        analytics/
          metrics_log.md
        reports/
          2026-06_report.md
        history/
          2026-06/
            content_log.md
            data_sources_log.md
            lead_log.md
            competitor_log.md
        outputs/
          2026-06/
            2026-06-19.md
            2026-06-19.html
          latest.md
          latest.html
    austin-home-group/
      realestate_austin/
        client_profile_austin-home-group_realestate_austin.md
        history/
          2026-06/
            content_log.md
            data_sources_log.md
        outputs/
          2026-06/
            2026-06-19.md
    bright-mortgage/
      mortgage_texas/
        client_profile_bright-mortgage_mortgage_texas.md
        history/
          2026-06/
            content_log.md
            data_sources_log.md
        outputs/
          2026-06/
            2026-06-19.md
```

Slug rules:

- Use lowercase letters.
- Replace spaces with hyphens.
- Remove punctuation when possible.
- Keep slugs short but recognizable.

Monthly organization rule:

- Any file created daily must be stored under a `YYYY-MM/` folder.
- This applies to client outputs, master digests, collector jobs, collector inboxes, history logs, data points, leads, competitors, and new private-source logs.
- Keep `latest.md`, `latest.html`, `latest_master_digest.md`, and `latest_master_digest.html` as convenience pointers at their existing locations.
- Do not allow long-running pipelines to accumulate hundreds or thousands of daily files directly in one folder.

---

## 8. Root Files

### `clients_index.md`

The root index of all client pipelines.

Format:

```md
# Clients Index

| Client | Client Slug | Pipeline Folder | Client Profile File | Status | Added Date | Schedule | Notes |
|---|---|---|---|---|---|---|---|
| Smith Law | smith-law | clients/smith-law/dui_los-angeles | client_profile_smith-law_dui_los-angeles.md | active | 2026-06-19 | daily | DUI lawyer in Los Angeles |
```

Allowed status:

- `active`
- `paused`
- `archived`
- `needs_setup`
- `needs_login`

Daily runs must process every client with `active` status.

### `schedule.md`

Records how daily runs happen in the current AI environment.

The schedule may use:

- Native AI automations.
- Reminders.
- Cron.
- Task Scheduler.
- n8n.
- Make.
- GitHub Actions.
- Local desktop routine.
- Manual run instructions.

If true automation is unavailable, create manual instructions.

### `notifications/notification_log.md`

Tracks notifications sent to the human through WideCast MCP / Telegram or any available notification channel.

Format:

```md
# Notification Log

| Date | Agent | Event | Channel | Status | Message Summary | Related Output | Action Needed |
|---|---|---|---|---|---|---|---|
| 2026-06-20 | Claude Schedule | daily_run_completed | WideCast Telegram | sent | 10 client outputs ready | outputs/2026-06/2026-06-20_master_digest.html | Review approvals |
```

Use this log so scheduled runs do not silently complete or fail while the human is away.

### `collector/collector_setup_status.md`

Tracks whether the Solo Agency Local Collector extension and Local Collector app are installed, reachable, blocked, pending activation, or waiting for human action.

This file is mandatory after the human agrees to activate private-source monitoring, when configuring a schedule that includes private sources, or when the agent needs to report a private-source collector blocker.

It is not required when no private sources are active. Before activation, the first agency report should simply list private sources under `Private Sources Pending Activation`.

Format:

```md
# Collector Setup Status

| Date | Agent | Status | Setup Command Given | Human Ran Setup Command | Chrome Extension Folder | Human Loaded Extension | Local Collector App | Health Endpoint | Last Health Check | Blocker | Required Human Action |
|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-06-20 | Claude | needs_user_action | bash "/ABSOLUTE/PATH/solo-agency-local-collector/setup_collector.sh" | no | /ABSOLUTE/PATH/solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME | no | /ABSOLUTE/PATH/solo-agency-local-collector/bin/collector-bridge-darwin-arm64 | http://127.0.0.1:17321/status | unavailable | Extension not loaded in Chrome yet | Run the setup command in Terminal/PowerShell outside the AI sandbox, then Chrome -> chrome://extensions -> Load unpacked -> select only the absolute runtime extension folder |
```

Allowed status:

- `not_needed_no_private_sources`
- `pending_private_activation`
- `setup_files_prepared_waiting_for_human_command`
- `setup_command_given_waiting_for_human_run`
- `setup_command_ran_waiting_for_extension`
- `activation_declined_for_now`
- `installed_and_running`
- `installed_not_running`
- `needs_user_action`
- `blocked_by_sandbox`
- `blocked_by_os_permission`
- `extension_not_loaded`
- `extension_stale`
- `bridge_offline`
- `session_expired`
- `failed`

The agent must update this file before:

- claiming private-source monitoring is active,
- running a manual private-source scan,
- configuring recurring private-source collection,
- reporting that private collection is unavailable.

### `outputs/YYYY-MM/YYYY-MM-DD_master_digest.md`

Daily summary across all active clients.

It should include:

- Date.
- Clients processed.
- Clients skipped.
- For each client:
  - Top ideas.
  - Best idea.
  - Script file path.
  - Reference URLs for top ideas and the selected best idea.
  - Sources skipped.
  - Required human action.

---

## 9. Client Intelligence Profile Schema

Each client pipeline must have one Client Intelligence Profile file.

Filename:

```text
client_profile_{client_slug}_{business_slug}_{location_slug}.md
```

Template:

```md
# Client Intelligence Profile: {client_name}

## Metadata

- client_name:
- client_slug:
- business_slug:
- location_slug:
- created_date:
- last_reviewed_date:
- status: active

## business_description

value:
status:
rationale:

## output_formats

status:
default: video_script
items:
- format: video_script | blog_article | social_caption
  cadence: daily | weekly | on_request
  widecast_skill_format: video | blog | social
  notes:

## industry

value:
status:
rationale:

## sub_industry

value:
status:
rationale:

## related_industries

status:
rationale:
content_mix_rule: approximately 80% primary industry / 20% related industries
items:
- name:
  relationship_to_primary_industry:
  why_it_matters_to_target_audience:
  example_content_bridges:
  allowed_use: signal_source | content_angle | data_source | lead_signal | competitor_context
  priority: high | medium | low

## target_audience

value:
status:
rationale:

## target_location

value:
status:
rationale:

## location_dependency

value: high | medium | low
status:
rationale:

## business_offer

value:
status:
rationale:

## pain_points

status:
rationale:
items:
- 

## content_pillars

status:
rationale:
content_mix_rule: approximately 80% primary industry / 20% related industries
items:
- name:
  industry_scope: primary_industry | related_industry
  related_industry:
  mapped_pain_points:
  strategic_purpose:
  example_angles:
  bridge_back_to_primary_offer:
  lead_gen_connection:

## public_data_sources

items:
- name:
  url:
  type: public
  platform:
  source_status: candidate_public_source | active_public_source | weekly_public_source | occasional_public_source | weak_public_source | blocked_or_unreliable
  source_kind: official | regulator | government | news | specialist_blog | public_forum | public_social | public_video_channel | competitor_public | data_dashboard | newsletter_archive | association | local_community | search_result | other
  language:
  scan_cadence: daily | weekly | occasional | event_based | paused
  visit_in_scheduled_runs: true | false
  location_relevance:
  related_pain_points:
  related_content_pillars:
  related_keywords:
  why_this_source_matters:
  source_or_reason_added:
  discovered_from:
  first_discovered_date:
  last_checked_date:
  useful_count:
  weak_count:
  usefulness_score:
  promoted_date:
  demoted_date:
  access_method:
  collection_notes:

## public_search_keywords

summary:
  total_keywords:
  hidden_keywords_saved:
  primary_keyword_language:
  secondary_keyword_languages:
  needs_expansion: true | false
  last_expanded_date:
  expansion_sources:
    - setup_inference
    - public_search_results
    - private_source_scan
    - competitor_hooks
    - report_comments
    - analytics_learning
    - human_feedback

items:
- keyword:
  language:
  status: unused | used | useful | weak | retry_later
  keyword_group: industry_general | pain_point | need_or_goal | buying_intent | local_context | related_industry | trend_news | objection | comparison | question | problem_issue
  scope: global | local
  industry_scope: primary_industry | related_industry
  related_industry:
  related_content_pillar:
  related_pain_point:
  related_customer_need:
  source_or_reason_added:
  discovered_from:
  first_added_date:
  last_used_date:
  use_count:
  useful_count:
  weak_count:
  result_quality:
  promoted: true | false
  demoted: true | false
  notes:

## private_monitoring_activation

status: not_provided | pending_private_activation | activation_declined_for_now | activation_requested | setup_files_prepared_waiting_for_human_command | setup_command_given_waiting_for_human_run | setup_command_ran_waiting_for_extension | installed_and_running | blocked
first_trial_policy: public_first_small_win
last_prompted_date:
human_decision:
collector_setup_status_file:
notes:

## private_interest_discovery

status: not_asked | declined | partially_approved | approved | pending_private_activation | active | blocked
reassurance_shown:
  professional_setup_once: true | false
  local_data_only: true | false
  daily_scan_prevents_missed_signals: true | false
categories:
  membership_sources:
    status: not_asked | declined | approved | pending_private_activation | active | blocked
    platforms:
    - platform:
      discovery_urls:
      - url:
        status: not_tried | pending_private_activation | scanned | login_required | platform_url_changed | failed
        last_scanned_at:
  following_sources:
    status: not_asked | declined | approved | pending_private_activation | active | blocked
    platforms:
    - platform:
      discovery_urls:
      - url:
        status: not_tried | pending_private_activation | scanned | login_required | platform_url_changed | failed
        last_scanned_at:
  recommendation_feed_sources:
    status: not_asked | declined | approved | pending_private_activation | active | blocked
    platforms:
    - platform:
      discovery_urls:
      - url:
        status: not_tried | pending_private_activation | scanned | login_required | platform_url_changed | failed
        last_scanned_at:
candidate_source_review_policy:
  require_human_approval_before_activating: true
  max_daily_sources_default: 20
  feed_surfaces_are_discovery_only: true
last_discovery_report:

## private_data_sources

items:
- name:
  url:
  type: private
  platform:
  source_type: manually_provided | joined_group | followed_profile | followed_page | subscribed_channel | followed_company | subreddit | community | discovered_from_feed
  discovery_category: manually_provided | membership_sources | following_sources | recommendation_feed_sources
  discovery_url:
  approval_status: pending_human_approval | approved | rejected
  priority: high | medium | low
  scan_cadence: daily | weekly | optional
  location_relevance:
  why_this_source_matters:
  access_method:
  collection_notes:
  activation_status: pending_private_activation | active | declined_for_now | unavailable
  login_status: unknown | available | expired | unavailable

## collector_config

status:
run_mode: agent_on_demand | persistent_bridge_scheduler | manual
default_runs_per_day: 1
scheduled_windows:
- name: morning
  enabled: true
  local_time_start: "09:00"
  local_time_end: "09:30"
  timezone:
max_sources_per_run: 20
max_scrolls_per_source: 5
max_scrolls_allowed: 10
scroll_delay_seconds: 5
duplicate_filter:
  compare_against_previous_day: true
  method: visible_text_matching
  parse_html: false
collector_panel:
  show_current_source: true
  show_scroll_count: true
  show_data_point_count: true
  show_status: true

## brand_voice

value:
status:
rationale:

## language

human_report_language:
target_audience_language:
keyword_language:
secondary_keyword_languages:
content_output_language:
status:
rationale:

## platforms

value:
status:
rationale:

## compliance_notes

value:
status:
rationale:

## negative_topics

value:
status:
rationale:

## assumptions

- 

## human_corrections

- 
```

---

## 10. History Files

### `history/YYYY-MM/content_log.md`

Purpose:

- Avoid repeating the same idea too often.
- Track selected ideas, scripts, approvals, videos, and outcomes.
- Track whether each selected idea was `primary_industry` or `related_industry` so the agent can maintain the 80/20 content mix over time.

Format:

```md
# Content Log

| Date | Idea | Category | Scope | Industry Scope | Related Industry | Content Pillar | Script Path | Status | Notes |
|---|---|---|---|---|---|---|---|---|---|
| 2026-06-19 | Austin inventory is rising again | Hot / Trend / News | Local | primary_industry |  | Local market intelligence | outputs/2026-06/2026-06-19.md | drafted | Not yet approved |
| 2026-06-20 | Why rising insurance premiums change your homebuying budget | Hot / Trend / News | Local | related_industry | P&C insurance | Affordability clarity | outputs/2026-06/2026-06-20.md | drafted | Related-industry idea connected back to buyer affordability |
```

Allowed status:

- `drafted`
- `approved`
- `video_created`
- `published`
- `rejected`
- `revised`
- `skipped`

### `history/YYYY-MM/data_sources_log.md`

Purpose:

- Track source checks.
- Track unavailable sources.
- Track private login/session failures.
- Track platform warnings, rate limits, checkpoints, and conservative pacing decisions.
- Avoid silently losing coverage.

Format:

```md
# Data Sources Log

| Date | Source | Type | Source URL | Status | Data Collected | Issue | Next Action |
|---|---|---|---|---|---|---|---|
| 2026-06-19 | Competitor FB Page A | private | https://www.facebook.com/... | skipped | no | session expired | Human must log in manually |
```

Allowed status:

- `checked`
- `collected`
- `skipped`
- `blocked`
- `session_expired`
- `rate_limited`
- `platform_warning`
- `collector_unavailable`
- `extension_unavailable`
- `extension_stale`
- `bridge_offline`
- `captcha_or_checkpoint`
- `chrome_not_running`
- `not_relevant_today`
- `unavailable`

### `history/YYYY-MM/lead_log.md`

Purpose:

- Track potential hot and warm leads discovered during public/private source scanning.
- Preserve source URLs and reasoning for why the lead may be relevant.
- Avoid losing sales opportunities discovered during content research.

Format:

```md
# Lead Log

| Date | Lead Level | Source | Source Type | Profile URL | Post/Current URL | Safe Lead Summary | Related Offer | Related Pain Point | Suggested Next Action | Status | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-06-20 | hot | Facebook Group | private | https://www.facebook.com/profile.php?id=... | https://www.facebook.com/groups/.../posts/... | Person asked for a DUI lawyer in Los Angeles | DUI legal consultation | Fear of license/court consequences | Human should review and decide whether to respond | needs_review | Do not contact automatically |
```

Allowed status:

- `needs_review`
- `approved_for_outreach`
- `contacted`
- `not_relevant`
- `do_not_contact`
- `converted`
- `skipped`

### `history/YYYY-MM/competitor_log.md`

Purpose:

- Track direct, adjacent, and audience competitors discovered during source scanning.
- Preserve competitor URLs, positioning notes, content patterns, and engagement signals.
- Help the agent improve positioning, content pillars, and idea selection over time.

Format:

```md
# Competitor Log

| Date | Competitor Type | Name/Page | Platform | Profile URL | Post/Current URL | Location Relevance | Audience Overlap | Offer/Positioning | Content Themes | Engagement Signal | Threat Level | Opportunity | Status |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-06-20 | direct | Example DUI Law Firm | Facebook | https://www.facebook.com/exampleduilaw | https://www.facebook.com/exampleduilaw/posts/... | Los Angeles | Drivers facing DUI/legal issues | Free consultation for DUI cases | License suspension, court mistakes | Repeated comments asking for help | medium | Create clearer local process education | monitoring |
```

Allowed status:

- `monitoring`
- `high_priority`
- `not_relevant`
- `archived`

### `history/YYYY-MM/new_private_sources_log.md`

Purpose:

- Track new private-source candidates discovered while scanning private platforms.
- Preserve Facebook-recommended groups, pages, communities, profiles, or similar source suggestions.
- Let the human review new sources before they become part of the active daily private-source queue.

Format:

```md
# New Private Sources Log

| Date | Platform | Source Type | Source Name | Profile/Group URL | Current Recommendation URL | Detected While Scanning | Why Relevant | Related Content Pillar | Estimated Priority | Suggested Cadence | Status | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-06-20 | Facebook | group | Los Angeles DUI Support Questions | https://www.facebook.com/groups/... | https://www.facebook.com/groups/... | Competitor group scan | Repeated questions about DUI court and license issues | Local process education | medium | weekly | needs_human_review | Do not join automatically |
```

Allowed status:

- `needs_human_review`
- `added`
- `skipped`
- `not_relevant`
- `blocked`

---

### 23.1 Client Strategy And Positioning

For each client, the agent should maintain strategy files:

- `strategy/offer_map.md`
- `strategy/brand_voice.md`
- `strategy/content_pillars.md`
- `strategy/funnel_map.md`

The agent must infer and maintain:

- Core offer.
- Secondary offers.
- Ideal customer segments.
- Lead magnets or conversion actions.
- Trust signals.
- Differentiators.
- Proof points.
- Objections.
- Compliance boundaries.
- Brand voice.
- Content pillars.
- Funnel stage mapping.

Example funnel mapping:

| Funnel Stage | Goal | Example Content |
|---|---|---|
| Awareness | Make the audience recognize the problem | "Why buyers are confused by rising inventory" |
| Education | Explain options and consequences | "How property taxes change your real payment" |
| Trust | Show expertise and perspective | "Why preparation beats prediction in this market" |
| Lead-Gen | Prompt action | "Get pre-approved before you start touring" |

### 23.2 Content Calendar And Cadence

The agent should maintain:

- `calendar/content_calendar.md`

The calendar should include:

- Planned publish date.
- Platform.
- Client.
- Content pillar.
- Funnel stage.
- Topic.
- Script/output file.
- Approval status.
- Publishing status.
- Reference URLs.

The agent should use daily ideas to populate the calendar, but must avoid overfilling it without approval. The daily best idea becomes a candidate for the calendar, not automatically a published post.

Example calendar row:

```md
| Date | Platform | Pillar | Funnel Stage | Topic | Status | Output |
|---|---|---|---|---|---|---|
| 2026-06-20 | Reels / Shorts | Market timing | Education | Austin inventory is rising again | drafted | outputs/2026-06-20.md |
```

### 23.8 Analytics And Reporting

The agent should maintain:

- `analytics/metrics_log.md`
- `reports/YYYY-MM_report.md`

Track metrics when available:

- Views.
- Watch time.
- Retention.
- Likes.
- Comments.
- Shares.
- Saves.
- Clicks.
- Leads.
- Calls booked.
- Cost or credits spent.
- Published URL.
- Content pillar.
- Funnel stage.

### WideCast MCP Analytics Collection Rule

When running weekly learning, monthly reporting, or any performance review, the agent must use available WideCast MCP capabilities to collect performance data before drawing conclusions.

The agent should inspect the available WideCast MCP tool/API list at runtime and call the relevant tools for:

- Recently published content.
- Published post/video URLs.
- Title.
- Description.
- Caption.
- Hashtags.
- Platform.
- Publish date.
- Topic or video ID.
- General account analytics.
- View counts.
- Follower counts.
- Engagement trends.

If WideCast MCP exposes a list of published posts, recent videos, production history, publishing history, analytics dashboard, or platform statistics, the agent must use those sources first.

For each published content item from the last 7 days, the agent should measure it daily for up to 7 days after publishing:

1. Retrieve the published URL and metadata through WideCast MCP when available.
2. Save URL, title, description, caption, hashtags, platform, publish date, and related script/output file.
3. Use the Solo Agency Local Collector extension plus Local Collector app to capture visible metrics from each published URL when tools, permissions, and login state allow it.
4. Measure or extract available engagement metrics, such as:
   - views
   - likes
   - comments
   - shares
   - saves
   - reposts
   - reactions
   - follower/subscriber count where relevant
   - audience questions
   - objections
   - requests for help
   - lead signals in comments
5. If direct platform metrics are not accessible, record the limitation and use whatever WideCast MCP analytics or visible public metrics are available.
6. Store all results in `analytics/metrics_log.md`.
7. Store audience questions, objections, and useful comment signals in `analytics/comment_signal_log.md`.
8. Store strategic learnings in `analytics/learning_log.md`.
9. Use the results to update reports, content pillar scoring, hook learnings, CTA learnings, source priority, lead-gen angles, and future idea selection.

### Published URL Measurement Via Local Collector

The Local Collector is not only for private-source idea discovery. It should also be reused for published URL measurement when possible.

Reason:

- Some platform metrics are visible only inside the logged-in browser session.
- Some AI agents cannot reliably browse platform pages directly.
- The Solo Agency Local Collector extension can capture visible page text, current URL, engagement hints, and source metadata in the same browser/profile where the human is logged in.

When measuring published URLs:

1. Build a temporary run-now collector job whose sources are the published URLs retrieved from WideCast MCP.
2. Mark these sources clearly, for example:
   - `source_type: published_content_url`
   - `purpose: performance_measurement`
   - `platform: youtube | tiktok | instagram | facebook | x | linkedin | threads | pinterest | reddit | google_business_profile | other`
3. Use conservative pacing and do not hammer platform pages.
4. Capture visible text, current URL, page title, engagement hints, any visible metric labels/counts, and comments/questions when visible.
5. Store raw collector output under the normal collector `inbox/YYYY-MM/{run_id}/` folder.
6. Parse the captured visible text into normalized metrics when possible.
7. Store normalized metrics in `analytics/metrics_log.md`.
8. Store useful comment/question/objection/lead signals in `analytics/comment_signal_log.md`.
9. Store strategic learnings in `analytics/learning_log.md`.
10. If a metric is hidden, unavailable, or not visible in the logged-in session, write `unavailable` and explain why.

The agent must not scrape hidden APIs, extract cookies, bypass login, or defeat platform restrictions to measure metrics. Use only authorized visible data or WideCast MCP analytics.

The agent must also call WideCast MCP analytics or dashboard tools that provide overall account-level statistics, such as total views, follower growth, platform performance, or other aggregate metrics. These aggregate metrics should be stored and used for learning even when per-post data is incomplete.

Do not invent metrics. If a platform hides likes, shares, comments, views, or follower data from the current agent/session, mark the metric as `unavailable` and explain why.

Suggested `analytics/metrics_log.md` format:

```md
| Date Checked | Published Date | Client | Platform | URL | Title | Description | Hashtags | Content Pillar | Funnel Stage | Views | Likes | Comments | Shares | Saves | Followers/Subscribers | Source Of Metric | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-06-20 | 2026-06-18 | Smith Law | TikTok | https://... | What to do after a DUI stop | Short DUI education video | #dui #california | Emergency first steps | Education | 1200 | 44 | 8 | 3 | unavailable | unavailable | WideCast MCP + public URL check | Comments show license-suspension anxiety |
```

Suggested `analytics/comment_signal_log.md` format:

```md
| Date Checked | Client | Platform | URL | Comment/Question Summary | Signal Type | Pain Point | Lead Potential | Suggested Follow-Up Content | Notes |
|---|---|---|---|---|---|---|---|---|---|
| 2026-06-20 | Smith Law | TikTok | https://... | Viewers asked what happens to a driver's license after a DUI arrest | question | license suspension fear | warm | Explain the DMV deadline after a DUI arrest | Use as future QA script |
```

Suggested `analytics/learning_log.md` format:

```md
| Date | Client | Evidence | Learning | Affected Pillar | Hook/CTA Impact | Future Action |
|---|---|---|---|---|---|---|
| 2026-06-20 | Smith Law | DUI deadline video got high comment rate | License-suspension anxiety drives comments | Emergency first steps | Use deadline hooks more often | Prioritize DMV-deadline Q&A ideas next week |
```

The agent should generate weekly or monthly reports when asked or scheduled:

- What worked.
- What did not work.
- Best content pillars.
- Best hooks.
- Best platforms.
- Recommended next experiments.
- Content ideas to repeat or retire.

### 23.9 Experiment Backlog

The agent should maintain:

- `experiments/experiment_backlog.md`

Examples:

- Test fear-based hook vs curiosity hook.
- Test local news angle vs evergreen education.
- Test direct CTA vs soft CTA.
- Test face-on-camera vs faceless B-roll.
- Test short 25-second version vs 60-second version.
- Test competitor-response angle.

Each experiment should include:

- Hypothesis.
- Client.
- Content pillar.
- Platform.
- Success metric.
- Result.
- Next decision.
