# Measure Learn Improve

Stage: `05`

## Load Rule

Load once content has been published, during daily published-content checks, and during weekly/monthly performance review.

## Hard Gates For This Stage

- Measure content from yesterday and the last 7 days when available.
- Measure each published URL daily for up to 7 days after publishing.
- Use connected provider analytics first when available.
- Reuse the Local Collector for visible published URL measurement when useful and authorized.
- Do not invent metrics; mark unavailable metrics clearly.
- Feed learnings back into source priority, content pillars, hooks, CTAs, lead-gen angles, future ideas, and the public search keyword bank.

## Source Preservation Rule

This file is detailed source material moved from the original monolithic `SOLO_AGENCY_PLAYBOOK.md`.

Do not summarize away requirements, examples, checklists, schemas, protocols, URLs, edge cases, warnings, approval gates, or completion gates. If a downstream agent needs to shorten its response to the human, it may summarize the response, but it must still obey the full requirements in this file.

---

### Published Content Measurement Requirement

The measure-learn-improve phase is mandatory once content has been published.

For each published content item, the agent must:

1. Use connected provider tools when available.
2. If provider MCP/tools are connected, call the relevant tools to retrieve:
   - videos/posts published yesterday;
   - videos/posts published in the last 7 days;
   - published URLs;
   - title;
   - description;
   - caption;
   - hashtags;
   - platform;
   - publish date;
   - topic/video/content IDs;
   - account/platform analytics when available.
3. Measure each published URL daily for up to 7 days after publishing.
4. Reuse the Solo Agency Local Collector to open each published URL when useful and authorized, because some metrics/comments require a logged-in browser.
5. Capture visible:
   - views;
   - likes/reactions;
   - comments;
   - shares;
   - saves;
   - reposts;
   - follower/subscriber count when relevant;
   - audience questions;
   - objections;
   - lead signals in comments.
6. Store metrics in `analytics/metrics_log.md`.
7. Store comments/questions in `analytics/comment_signal_log.md`.
8. Store learnings in `analytics/learning_log.md`.
9. Use learnings to improve:
   - source priority;
   - content pillars;
   - hook selection;
   - CTA selection;
   - idea scoring;
   - lead-gen angles;
   - future scripts/blogs;
   - public search keywords, especially phrases copied from real audience questions, objections, needs, and comments.

Do not invent metrics. Mark unavailable metrics clearly.

### Scheduled Published URL Measurement Contract

During every scheduled run, after the normal research and draft workflow, the agent must check whether there is published content that still needs measurement.

If this is the first setup/run and no content has been published yet, the agent must not run or simulate measurement. It must record:

```text
measurement_status: no_published_urls_yet
```

Then continue with report generation, recommendations, and schedule setup if applicable.

The agent must:

1. Load the published content ledger, publishing logs, provider history, or connected provider account data.
2. Retrieve the list of videos/posts published yesterday.
3. Retrieve the list of videos/posts published in the last 7 days.
4. Extract every available published URL, platform, title, caption/description, hashtags, publish date, topic/video/content ID, and source output/script path.
5. For every published URL still inside its 7-day measurement window, open or inspect that URL using the best authorized source:
   - connected provider analytics first;
   - platform/account analytics if available;
   - Solo Agency Local Collector for visible URL/page/comment metrics when useful and authorized;
   - public visible page data if logged-in access is unavailable.
6. Visit or inspect each URL one by one. Do not summarize account-level analytics as a substitute for URL-level measurement when URL-level data is available.
7. Capture visible views, likes/reactions, comments, shares, saves, reposts, follower/subscriber count when relevant, audience questions, objections, and lead signals in comments.
8. Store unavailable metrics explicitly as `unavailable` with the reason.
9. Write normalized metrics to `analytics/metrics_log.md`.
10. Write useful questions, objections, and lead/comment signals to `analytics/comment_signal_log.md`.
11. Write strategic learnings to `analytics/learning_log.md`.
12. Feed the learning back into future source priority, content pillars, hooks, CTAs, idea scoring, lead-gen angles, scripts/blogs, and the saved public search keyword bank.
13. Extract new keyword candidates from audience questions, objections, comments, lead signals, high-performing hooks, captions, and hashtags. Add non-duplicate useful candidates to `public_search_keywords` with keyword group, related pain point, related content pillar, source/reason, and first-added date.

If the agent cannot access provider tools or published URLs, it must log the blocker and continue the rest of the scheduled run. It must not claim the measurement loop is complete.

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
9. Use the results to update reports, content pillar scoring, hook learnings, CTA learnings, source priority, lead-gen angles, future idea selection, and the public search keyword bank.
10. Extract new keyword candidates from high-signal comments, audience questions, objections, captions, hashtags, and winning hooks. Add non-duplicate candidates to `public_search_keywords` so future public research searches the way the audience actually talks.

### Published URL Measurement Via Local Collector

The Local Collector is not only for private data source idea discovery. It should also be reused for published URL measurement when possible.

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
| Date | Client | Evidence | Learning | Affected Pillar | Hook/CTA Impact | New Keyword Candidates | Future Action |
|---|---|---|---|---|---|---|---|
| 2026-06-20 | Smith Law | DUI deadline video got high comment rate | License-suspension anxiety drives comments | Emergency first steps | Use deadline hooks more often | `will I lose my license after DUI`, `DMV deadline after DUI arrest` | Prioritize DMV-deadline Q&A ideas next week |
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


### Measure-Learning Checklist

Before claiming a weekly/monthly performance review or learning loop is complete, verify:

- [ ] Did I call available WideCast MCP tools for published URLs, metadata, and account/platform analytics?
- [ ] Did I reuse the Solo Agency Local Collector extension plus Local Collector app to capture visible metrics from published URLs when possible?
- [ ] Did I store normalized metrics in `analytics/metrics_log.md`?
- [ ] Did I mark hidden or unavailable metrics as `unavailable` instead of inventing numbers?
- [ ] Did I use the measurements to update content pillar scoring, hook learnings, CTA learnings, source priority, future idea selection, and the public search keyword bank?
- [ ] Did I extract new keyword candidates from comments, objections, questions, high-performing hooks, captions, hashtags, or lead signals?

### Final Hard Gate
