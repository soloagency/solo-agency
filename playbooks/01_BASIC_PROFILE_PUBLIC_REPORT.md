# Basic Profile And Public Report

Stage: `01`

## Load Rule

Load during first setup, add-client flow, setup repair, and first public report generation. This stage must be loaded together with Stage 0 before the first setup question.

## Hard Gates For This Stage

- First question asks only for product/service, profession, expertise, or business description.
- Do not ask for industry or sub-industry.
- Show inference before asking the next question.
- Run the first public report immediately after profile setup.
- Do not ask whether the human wants the first trial.
- After the public report, ask production and private-source activation questions.

## Source Preservation Rule

This file is detailed source material moved from the original monolithic `SOLO_AGENCY_PLAYBOOK.md`.

Do not summarize away requirements, examples, checklists, schemas, protocols, URLs, edge cases, warnings, approval gates, or completion gates. If a downstream agent needs to shorten its response to the human, it may summarize the response, but it must still obey the full requirements in this file.

---

## 3. Minimal Human Input Rule

At setup, the agent must ask only for:

- Client name, if not already known.
- The client's product/service, profession, expertise, or business description.
- Target location only if location matters and cannot be inferred.
- Optional private data sources the human wants monitored.

The agent must not ask for `output_formats` by default. If no output format is specified, default to `video_script`. If the human asks for blog, article, newsletter, SEO content, or long-form content, add `blog_article`. If the human asks for platform captions, add `social_caption`.

The agent must not ask the human to define:

- `industry`
- `sub_industry`
- `related_industries`
- `target_audience`
- `pain_points`
- `content_pillars`
- `public_data_sources`
- `idea categories`
- `content angles`
- `daily matrix`

The agent must infer these first.

Good first setup question:

`What product/service, profession, expertise, or business description should this pipeline focus on? If you already know the target location or private sources to monitor, include them too.`

Good add-client question:

`Please provide the new client's name and product/service, profession, expertise, or business description. Include target location if known, and any private sources such as competitor pages or groups you want monitored.`

Bad setup questions:

- "What industry are you in?"
- "What sub-industry should I use?"
- "Please list your target audience."
- "Please list all pain points."
- "Please define your content pillars."
- "Please provide all public sources."

Exception:

If the agent cannot infer a critical field after reasonable research and the field changes the direction materially, it may ask one concise follow-up question.

### Step-By-Step Setup Interview Rule

Setup must be conducted step by step, not as one long questionnaire.

The agent must follow this loop:

1. Ask one minimal setup question.
2. Wait for the human's answer.
3. Immediately infer everything that can be inferred from that answer.
4. Show the inference to the human.
5. Ask the next minimal setup question only after showing the inference.

The agent must not collect all setup answers first and only show reasoning at the end. The human should see the agent's reasoning evolve after every answer.

Required setup sequence:

1. Ask for the client's product/service, profession, expertise, or business description.
2. After the answer, infer and show:
   - `industry`
   - `sub_industry`
   - `related_industries`
   - `business_offer`
   - likely `target_audience`
   - whether the business is location-dependent
3. If the target location is required and cannot be inferred, ask only for `target_location`.
4. After the answer, infer and show:
   - refined `target_audience`
   - local relevance
   - local audience problems
   - local source strategy
5. Infer and show:
   - `pain_points`
   - `content_pillars`
   - how each content pillar maps to pain points and the business offer
   - which content pillars are `primary_industry` vs `related_industry`
   - the planned content mix rule, normally 80% primary industry and 20% related industries
6. If the human has not already provided private data sources, ask whether the human wants to provide private data sources, including competitor profiles, fanpages, communities, LinkedIn pages, Reddit communities, niche forums, and manually known Facebook groups.
   - If the human already provided private sources in an earlier message, do not ask again before setup. Process the provided sources.
   - Do not label the collector by platform. Even if the provided sources are all Facebook, call it the Solo Agency Local Collector extension and Local Collector app.
7. Separately ask whether the human wants Private Interest Graph Discovery:
   - groups/communities they joined
   - pages/profiles/KOLs/creators/companies/channels they follow or subscribe to
   - platform feeds/recommendations that surface topics they care about
   - Explain the three reassurance points: professional one-time agency setup, local-only data safety, and daily scanning after activation.
8. If the human approves Facebook member-groups discovery, scan the joined-groups discovery page through the Solo Agency Local Collector:
   - `https://www.facebook.com/groups/joins/?nav_source=tab&ordering=viewer_added`
   - If the human agrees, this requires Local Collector activation before the joined-groups discovery scan.
   - If the human declines or postpones, do not inspect that page.
9. If the human approves other platform discovery categories, use the `Private Interest Graph Discovery` platform starting URL registry and mark each approved category as `pending_private_activation` until the Local Collector is active and healthy.
10. After the answer, infer and show:
   - which private sources are likely useful
   - which sources should be skipped or treated as optional
   - how the private sources map to content pillars
   - whether the private-source list should be kept as `daily`, `weekly`, or `optional` based on relevance and safe monitoring volume
11. Show the complete setup summary and ask the human to correct only what is wrong.
12. Save the Client Intelligence Profile file only after the human has had a chance to correct the setup summary.
13. Run the first trial immediately after the profile is ready, using public sources and any already available local data.
   - Do not wait for Local Collector installation.
   - If private sources were provided, list them as `pending_private_activation`.
   - If the human agreed to Private Interest Graph Discovery but Local Collector is not active yet, list each approved discovery category as `pending_private_activation`.
   - Explain that private-source monitoring requires a one-time Solo Agency Local Collector extension and Local Collector app setup.
14. Produce the first trial report as a small win.
15. After showing the first report, the chat message must include:
   - the best idea and a short useful summary;
   - the mobile-friendly HTML report path/link;
   - a visible `Solo Agency onetime setup` progress block with completed/current/remaining steps;
   - a clear note that the run used public sources only if private sources are not active;
   - the number and names/URLs of pending private sources, if any;
   - whether Private Interest Graph Discovery categories are pending, approved, declined, or not asked yet;
   - the direct activation question: `Private sources and Private Interest Graph Discovery require the Solo Agency Local Collector extension and Local Collector app. Do you want me to set that up now?`
   - the direct activation question must be the final line of the message when private-source activation is the next required decision.
16. If the human says yes, install or initiate setup for the Solo Agency Local Collector extension and Local Collector app.
17. After collector setup succeeds, run a private-source activation scan or second trial enrichment when possible. If Private Interest Graph Discovery was approved, scan the approved discovery URLs first, filter candidate sources, show the recommendations to the human, and ask for approval before adding them to active `private_data_sources`.
18. After any private scan or approved source-discovery scan, analyze the collected private data and update the report. This means extracting data points, detecting leads, detecting competitors, listing new private sources, updating the idea matrix, re-scoring the best idea, updating drafts if needed, regenerating the HTML report, and showing the updated report to the human.
19. Only after the human has seen the first trial report and has decided whether to activate private sources, ask about the recurring schedule.

Every follow-up question must include a short `What I inferred from your last answer` section before the next question.

Example:

```md
What I inferred from your last answer:
- Industry: Legal
- Sub-industry: DUI / Criminal Defense
- Related industries: auto insurance, DMV/license rules, employment background checks, immigration consequences, local traffic enforcement
- Target audience: drivers in Los Angeles facing DUI stops, arrests, court dates, or license suspension risk
- Pain points: fear of losing license, uncertainty about court, fear of criminal record, not knowing what to say after a stop
- Content mix: roughly 80% DUI/criminal defense, 20% related consequences such as insurance, license, job, or immigration impact when the bridge is clear
- Content pillars:
  - Emergency first steps
  - License and court consequence clarity
  - Mistake prevention after a DUI stop
  - Los Angeles / California process education
  - Lead-gen angle: why early legal guidance preserves options

Next question:
Do you want to provide competitor pages, Facebook groups, or other private sources to monitor for this client? For account safety and platform-respectful monitoring, please avoid adding too many private sources; around 20 or fewer per client is a good daily default. If you provide more, I will prioritize and rotate them.
```

---

## 4. Inference-First Rule

The agent must think, infer, and research before asking.

The agent must:

- Use existing files first.
- Use the client description.
- Use public web research if available.
- Use the client's website or public profile if available.
- Use known industry patterns.
- Use target location context.
- Draft assumptions instead of blocking.

The agent should proceed with reasonable assumptions when optional fields are missing.

Each inferred setup field must include:

- `value`
- `status`
- `rationale`

Allowed status values:

- `provided_by_human`
- `inferred_by_agent`
- `discovered_from_source`
- `human_corrected`

Example:

```md
## target_audience
value: First-time home buyers, mortgage shoppers, and homeowners considering refinancing in Austin.
status: inferred_by_agent
rationale: The client provides mortgage services in Austin. These groups are the most likely to have urgent questions about affordability, rates, pre-approval, and monthly payments.
```

---

## 5. Show Inference And Research Rule

Anything inferred or researched by the agent must be shown to the human before being saved as stable setup context.

The agent must show:

- Inferred `industry`
- Inferred `sub_industry`
- Inferred `target_audience`
- Inferred or discovered `target_location`
- Inferred `pain_points`
- Inferred `content_pillars`
- Inferred `business_offer`
- Discovered public data sources
- Suggested public monitoring sources
- Requested private source categories
- Assumptions and rationale
- Compliance notes
- Negative topics if any are inferred

The agent should ask:

`Please correct anything that is wrong. If this looks right, I will save it and use it for future daily runs.`

The agent must not ask the human to fill every field manually.

---

## 12. Multi-Client Batch Mode

The pipeline must support many clients across many industries.

If the human already has multiple clients, the agent should accept a compact list.

Example human input:

```md
I manage 10 clients. Set up one daily content pipeline for each:

1. Smith Law - DUI lawyer - Los Angeles - private sources: competitor FB pages A, B
2. Austin Home Group - real estate agent - Austin, TX - private sources: none yet
3. Bright Mortgage - home loans - Texas - private sources: competitor TikTok X
4. Miami Shield Insurance - home and auto insurance - Miami - private sources: local FB group Y
5. Vienna AI Ops - AI automation agency - Vienna - private sources: LinkedIn competitors
```

The agent must:

1. Create one pipeline folder per client.
2. Infer all setup fields for each client.
3. Show a setup summary for each client.
4. Ask the human to correct only what is wrong.
5. Save one Client Intelligence Profile file per client.
6. Add all clients to `clients_index.md`.
7. Configure or document the daily schedule/routine.
8. Ensure daily runs process every active client.

If the human provides incomplete entries, infer what is possible and ask only for missing critical information.

---

## 13. Incremental Client Onboarding Rule

The pipeline must support starting with zero clients and adding clients over time.

The human is not required to provide all clients at once.

If there are no clients yet, create only:

```text
daily-content-pipeline/
  clients_index.md
  schedule.md
  clients/
  outputs/
```

Then immediately enter First Client Setup Mode.

First Client Setup Mode is the same as Add Client Mode, but it is triggered automatically during the first run when `clients_index.md` has no real client rows. The agent must proceed as far as possible toward setting up the first client instead of stopping after root folder creation.

In First Client Setup Mode, ask only for the minimum information required to create the first client pipeline:

- Client name, if not already known.
- Product/service, profession, expertise, or business description.
- Target location only if location matters and cannot be inferred.
- Optional private data sources to monitor.
- Optional permission for Private Interest Graph Discovery: joined groups/communities, followed profiles/pages/KOLs/channels, subscriptions, and platform recommendation feeds.

Do not create fake client pipelines. If the client name or business description is missing, ask for that missing information and keep the root pipeline ready.

Whenever the human says something like:

- "Add a new client"
- "Add this client to the pipeline"
- "We just got a new client"
- "Start monitoring content ideas for this business"
- "Add client: ..."

The agent must enter Add Client Mode.

In Add Client Mode, ask only for missing critical information:

- Client name.
- Product/service, profession, expertise, or business description.
- Target location only if location matters and cannot be inferred.
- Optional private data sources to monitor.
- Optional permission for Private Interest Graph Discovery: joined groups/communities, followed profiles/pages/KOLs/channels, subscriptions, and platform recommendation feeds.

The agent must infer:

- `industry`
- `sub_industry`
- `target_audience`
- `pain_points`
- `content_pillars`
- `business_offer`
- `public_data_sources`
- `brand_voice`
- `language`
- `platforms`
- `compliance_notes`
- `negative_topics`

Then the agent must:

1. Show the inferred setup summary to the human.
2. Ask the human to correct only what is wrong.
3. Create a new client pipeline folder.
4. Create the client's Client Intelligence Profile file.
5. Create the client's history folder.
6. Create the client's outputs folder.
7. Add the client to `clients_index.md`.
8. Run the first trial report immediately using public sources and any already available local data.
9. Ask whether the human wants Production & Distribution & Notification & Analytics setup now for video/blog/social, publishing, notifications, analytics, and the build-measure-learn loop.
10. If the human says yes to production/video/blog/social, publishing, notifications, analytics, or "full automatic", load `playbooks/03_PRODUCTION_DISTRIBUTION.md` and complete the provider setup gate before asking schedule.
11. If private sources exist, or if Private Interest Graph Discovery was approved, show them in the report as `pending_private_activation`, then ask whether the human wants to activate private-source monitoring/discovery after the production setup gate is completed, declined, or blocked.
12. If the human agrees, install or initiate the Solo Agency Local Collector extension and Local Collector app setup.
13. If Private Interest Graph Discovery was approved, scan approved discovery URLs from the platform starting URL registry, filter candidate sources, show recommendations, and ask approval before adding sources to active private sources.
14. If any private scan or approved source-discovery scan runs, analyze the collected private data and regenerate the report, idea matrix, best idea, leads, competitors, and drafts if needed before claiming the private-source step is complete.
15. If published URL history exists, load `playbooks/05_MEASURE_LEARN_IMPROVE.md` and scan analytics/signals for the last 7 days before updating the final recommendation. If no published URL history exists, mark this as not available yet instead of pretending measurement ran.
16. Update the report, idea matrix, best idea for today, leads, competitors, and drafts with private data and, from the second run onward, analytics/statistics from published URLs.
17. After the first trial report is shown, the production setup gate is completed/declined/blocked, private-source activation has been accepted/declined/documented as pending, and the published-URL analytics step is completed or honestly marked unavailable, ask the human whether and how to configure the recurring schedule/routine.
18. Only after schedule confirmation, add or update the recurring schedule/routine and confirm whether future scheduled runs include public sources only or both public and activated private sources.

Example:

Human:

```md
Add this client to the daily content pipeline: Nguyen Law, immigration lawyer in San Jose. Private sources to monitor: [links].
```

Agent must create:

```text
daily-content-pipeline/
  clients/
    nguyen-law/
      immigration-law_san-jose/
        client_profile_nguyen-law_immigration-law_san-jose.md
        history/
          content_log.md
          data_sources_log.md
        outputs/
```

The agent must run Nguyen Law's first trial report immediately after setup. Only after the trial report is shown, the production/provider gate is completed/declined/blocked, private-source status is resolved or marked pending, and published-URL analytics is run or marked `no published URLs yet`, should the agent ask how to configure recurring scheduled runs.

---

## 14. Mandatory First Trial Run Protocol

This protocol applies after the first client setup, after adding a new client, and after repairing an incomplete Client Intelligence Profile.

The setup flow is not a menu of optional next steps. The agent must not ask the human to choose between:

- running the first trial,
- installing the collector,
- configuring the schedule.

The correct order is fixed:

1. Finish setup and save the Client Intelligence Profile.
2. Run the first trial report immediately using public sources and any already available local data.
3. Show the first report to the human as a small win.
4. Ask whether the human wants Production & Distribution & Notification & Analytics setup now.
5. If the human says yes to production/video/blog/social, publishing, notifications, analytics, or "full automatic", load `playbooks/03_PRODUCTION_DISTRIBUTION.md` and complete the provider setup gate before moving on.
6. If private sources were provided, or if the human agreed to Private Interest Graph Discovery, explain that private-source monitoring/discovery is still pending and ask whether to activate it after the production setup gate is completed, declined, or blocked.
7. If the human agrees to private sources, install or initiate setup for the Solo Agency Local Collector extension and Local Collector app.
8. If Private Interest Graph Discovery was approved, run approved discovery scans using the platform starting URL registry, filter candidate sources, and ask the human to approve recommended sources before adding them as active private sources.
9. Ask about recurring schedule only after the first report exists, the Production & Distribution & Notification & Analytics setup gate has been completed/declined/blocked, and private-source activation has been accepted, declined, or documented as pending.

Public-first first trial rule:

- The first trial should happen before Local Collector setup unless the Local Collector app is already installed and running.
- The first trial must not be blocked by Chrome extension installation, local binary permissions, sandbox limits, or private-source login state.
- The first trial should use public sources, public search, client context, inferred pain points, inferred content pillars, related industries, and any previously collected local data.
- If private sources were provided, the first trial report must include a section called `Private Sources Pending Activation`.
- If Private Interest Graph Discovery was approved but not yet run, the first trial report must include `Private Interest Graph Discovery Pending Activation`.
- That section must list the private source URLs, explain that they were not scanned yet, and say that activation requires the Solo Agency Local Collector extension plus Local Collector app.
- The first trial report must ask a clear next-step question after delivering the useful output. Unless Production & Distribution & Notification & Analytics setup was already completed or explicitly declined, the first next-step question must be:

```md
Do you want Production & Distribution & Notification & Analytics setup now for video/blog/social, publishing, Telegram/report notifications, analytics, and the learning loop?
```

The agent must ask this question directly in the chat message or notification where it announces the first trial result. It must not hide the question or setup steps inside a Markdown file.

The same chat message must show the updated `Solo Agency onetime setup` progress checklist or a compact progress summary. It must show that public-first research/report generation is complete and that production/provider setup, private-source activation, published-URL analytics status, report/recommendation update status, and final schedule setup are still pending, completed, declined, blocked, or not applicable.

If Production & Distribution & Notification & Analytics setup is already completed, declined, or blocked and private-source activation is pending, the final line must be:

```md
Private sources and Private Interest Graph Discovery are not activated yet because they require the Solo Agency Local Collector extension and Local Collector app. Do you want me to set that up now?
```

If there are no private sources and discovery was declined or not requested, the final line must ask the next required decision, usually:

```md
Do you want Production & Distribution & Notification & Analytics setup now for video/blog/social, publishing, Telegram/report notifications, analytics, and the learning loop?
```

or, if production was already declined:

```md
Do you want daily, multiple-times-daily, weekly, or manual-only runs?
```

Do not end the report handoff with only a report link, a summary, or "let me know."

Good first-trial chat pattern:

```md
The first trial report is ready.

Best idea today: {best idea}
Report for mobile: {absolute HTML path or URL}

Solo Agency onetime setup
[x] 1. Bạn cung cấp thông tin sản phẩm/dịch vụ, nghề, chuyên môn hoặc mô tả doanh nghiệp
[x] 2. Tôi tự suy luận ngành, ngành phụ, ngành liên quan, đối tượng, offer
[x] 3. Tôi tự suy luận pain points và content pillars
[x] 4. Tôi tự tìm và chọn nguồn công khai và từ khóa tìm kiếm
[x] 5. Tôi tự chạy nghiên cứu public-first
[x] 6. Tôi tạo báo cáo HTML public-first
[ ] 7. Tôi trợ giúp bạn thiết lập Production & Distribution & Notification & Analytics nếu bạn muốn
[ ] 8. Tôi tự cấu hình luồng sản xuất/đăng/thông báo/phân tích
[ ] 9. Bạn cung cấp nguồn riêng tư (private) thủ công nếu muốn
[ ] 10. Bạn cho phép chạy Private Interest Graph Discovery nếu muốn
[ ] 11. Tôi kích hoạt Local Collector nếu bạn cho phép quét dữ liệu nguồn riêng
[ ] 12. Tôi chạy source discovery và xin bạn duyệt nguồn đề xuất
[ ] 13. Tôi chạy lần quét riêng đầu tiên
[-] 14. Tôi quét analytics các URL đã đăng trong 7 ngày gần nhất (chưa có URL đã đăng)
[ ] 15. Tôi cập nhật báo cáo, ma trận ý tưởng, ý tưởng tốt nhất hôm nay, lead, đối thủ, bản nháp. Từ lần chạy thứ hai trở đi, tôi thêm analytics và statistics từ bước 14.
[ ] 16. Tôi cấu hình lịch chạy tự động (chỉ setup 1 lần)

The report includes an `Unlock Production & Distribution & Measure-Learning Loop With WideCast` section. You can keep using the playbook manually, or connect WideCast once to create videos, publish to 10+ platforms, receive Telegram alerts, measure performance, and feed that learning back into better ideas.

This run used public sources only. I have {N} private sources waiting, including:
- {source name or URL}
- {source name or URL}

Do you want Production & Distribution & Notification & Analytics setup now for video/blog/social, publishing, Telegram/report notifications, analytics, and the learning loop?
```

Bad first-trial chat pattern:

```md
Private sources were not scanned. Instructions are in collector/collector_setup_status.md.
Now choose a schedule.
```

Also bad:

```md
The first trial report is ready.
Report: {html path}
Let me know what you want to do next.
```

This is bad because it does not show progress and does not end with a concrete next-step question.

Private-source activation rule:

- If the human agrees to activate private sources, collector setup becomes mandatory at that point.
- The agent should proceed automatically as far as its environment allows.
- The agent may ask the human only for required local actions, such as loading the Chrome extension from an absolute path, approving a local command, running a generated macOS/Linux command, or running a generated Windows PowerShell/`.cmd` launcher.
- If a local command is required, the agent must create the script/launcher file first and give the human exactly one short command or one double-clickable file path, not a long multi-line script.
- The exact human action must be shown directly in chat. The agent may also save it in `collector_setup_status.md`, but the saved file is only the agent's record and must not be the only place where the human receives the instruction.
- The agent must not label the collector by the current platform, such as `Facebook collector`.
- The agent must create or update `daily-content-pipeline/collector/collector_setup_status.md` when private-source activation begins.
- If the AI environment can run local commands, the agent must download/update the collector, create/update the setup script, start/restart the Local Collector app, and check `GET http://127.0.0.1:17321/status`.
- If the AI environment cannot run local commands, the agent must still create the setup script/launcher file and give the human exactly one short command or double-clickable file path.
- If the Solo Agency Local Collector extension is not loaded, the agent must show the absolute extension folder path and the exact Chrome `Load unpacked` steps.
- After collector setup succeeds, the agent should run a private-source activation scan or second trial enrichment when possible.
- The agent must not claim private-source monitoring is active until collector health confirms the Local Collector app and Solo Agency Local Collector extension are working.
- The agent must not configure a recurring schedule that promises private-source collection until collector setup is either `installed_and_running` or explicitly documented as pending/blocked with a human action.

First trial rule:

- The agent must not ask `Do you want me to run the first trial?`
- The first trial must not depend on a recurring schedule window.
- If the Local Collector app is already installed, running, and healthy, the agent may include private sources in the first trial by creating a run-now job.
- If the Local Collector app is not already installed/running/healthy, run the public-first trial and list private sources as pending activation.
- The first trial output must include a mobile-friendly HTML report and a concise summary.
- If WideCast account tools are not connected, the first trial HTML report must include `Unlock Production & Distribution & Measure-Learning Loop With WideCast` so the human sees how the useful report can become video/blog production, 10+ platform distribution, Telegram notifications, performance measurement, and a learning loop after one WideCast setup.

Manual run / run-now rule:

- Any human request such as `run now`, `manual run`, `test run`, `trial run`, `collect now`, `scan now`, or `chạy thử` must bypass recurring schedule windows.
- The agent must not wait for `scheduled_windows` when the human requested a manual run.
- If the Local Collector app is reachable, the agent must create a run-now job and call `POST http://127.0.0.1:17321/jobs/run_now`.
- The run-now job must include:
  - unique `run_id`,
  - `run_now: true`,
  - `force: false` by default,
  - `run_now_ttl_minutes`, default 30 and maximum 120,
  - private `sources`,
  - pacing rules,
  - client/business/location metadata when available.
- To run again, the agent should create a new unique `run_id` instead of forcing the same run id repeatedly.
- The run-now job must expire automatically if it is not completed, so the extension cannot keep seeing the same manual job all day.
- The Solo Agency Local Collector extension should see `job_available: true` on the next `/status` poll and run immediately.
- If the Local Collector app is not reachable, the agent should start it if possible. If the agent cannot start it, provide the one-line Local Collector app start command, then retry the run-now job after the app is reachable.
- Recurring schedule windows are only for unattended scheduled runs. They must not block manual runs.
- Do not simulate a manual run by editing `scheduled_windows` or creating a temporary schedule window. Manual runs must use `/jobs/run_now`.
- If the agent cannot call `http://127.0.0.1:17321` from its own sandbox but can write local files, it must write the same run-now payload to `daily-content-pipeline/collector/run_now_request.json`. The Local Collector app must check this file on `/status`, load it as a run-now job, write `run_now_request_status.json`, and move the request aside as consumed. This avoids asking the human to run another command.
- If the agent cannot call HTTP and cannot write the local request file, only then create a local run-now helper script or launcher and give the human exactly one short command/path to run it. The helper script must POST `/jobs/run_now` with the correct payload, then optionally poll `/status`.
- Do not ask the human to restart the Local Collector app merely to make a manually edited schedule file take effect. Restarting is only appropriate for updating the Local Collector app itself, recovering a stuck/offline process, or applying an intentional recurring schedule change when both `/config` and file auto-reload are unavailable.
- If a legacy collector without `/jobs/run_now` forces a temporary schedule fallback, the agent must clearly label it as a fallback, back up the original config, create a short unique temporary window, restart or reload only if required, restore the original config immediately after completion/timeout, and report that fallback to the human. This fallback must not be used when `/jobs/run_now` exists.

Exact manual run-now contract:

- Health-check the Local Collector app first with plain `GET http://127.0.0.1:17321/status`.
- Do not send `X-Collector-Extension` when the AI agent checks health. That header is for the Solo Agency Local Collector extension only. If the AI agent fakes it, `extension_health` can become misleading.
- If `/status` is reachable, call `POST http://127.0.0.1:17321/jobs/run_now`.
- The minimum payload should look like this:

```json
{
  "run_id": "2026-06-20_client-slug_manual_150405",
  "client_slug": "client-slug",
  "business_slug": "business-or-brand-slug",
  "industry": "life insurance",
  "sub_industry": "family protection and retirement planning",
  "target_location": "California, United States",
  "run_now": true,
  "force": false,
  "run_now_ttl_minutes": 30,
  "sources": [
    {
      "name": "Competitor page or private group name",
      "url": "https://www.facebook.com/groups/example",
      "platform": "facebook",
      "source_type": "private_group",
      "purpose": "monitor audience questions, competitor positioning, leads, and content ideas",
      "priority": "high"
    }
  ],
  "pacing": {
    "min_delay_seconds": 5,
    "max_delay_seconds": 5,
    "max_sources": 20,
    "scroll_steps": 5,
    "max_text_chars": 12000
  },
  "collector_policy": {
    "read_only": true,
    "do_not_comment": true,
    "do_not_message": true,
    "do_not_react": true,
    "do_not_scrape_contact_details": true
  }
}
```

- `run_id` must be unique for every manual run. A recommended pattern is `YYYY-MM-DD_client-slug_manual_HHMMSS`.
- `run_now` must be `true`.
- `force` must be `false` unless the human explicitly asks for a troubleshooting rerun and understands the same `run_id` may run again.
- `run_now_ttl_minutes` should be 30 by default and must not exceed 120.
- `sources` must contain the private sources for that client if private sources exist. If there are no private sources, the agent should still run public research without the Local Collector app.
- `pacing.scroll_steps` defaults to 5 and must not exceed 10.
- If the agent cannot make this POST itself but can write local files, it should write the JSON payload to:

```text
daily-content-pipeline/collector/run_now_request.json
```

The agent should write this file atomically: write a temporary file in the same folder first, then rename it to `run_now_request.json` only after the JSON is complete.

The running Local Collector app should pick up this file on the next `/status` check from the Chrome extension or AI agent, usually within a few seconds while Chrome is active. After loading the request, the Local Collector app must immediately consume the request so it cannot loop forever:

- move it to `run_now_request.{run_id}.{timestamp}.consumed.json`;
- write `run_now_request_status.json`;
- remember the processed file signature in memory as a replay guard if moving/removing fails;
- clear the active run-now job on `/complete`;
- expire the active run-now job after `run_now_ttl_minutes` if `/complete` never arrives.

After loading the request, the Local Collector app should write:

```text
daily-content-pipeline/collector/run_now_request_status.json
```

Only if the agent cannot write the request file should it create one of these helper files:
  - macOS/Linux: `daily-content-pipeline/collector/run_private_now.sh`
  - Windows: `daily-content-pipeline/collector/Run Private Collector Now.cmd`
- The human-facing instruction should be one line, for example:

```bash
bash "/ABSOLUTE/PATH/TO/daily-content-pipeline/collector/run_private_now.sh"
```

- After posting `/jobs/run_now`, poll plain `GET /status` until either:
  - `current_job_type` becomes `run_now` and `job_available` is `true`,
  - the extension completes and `/status` returns `job_available: false`, or
  - the TTL expires and private collection is marked unavailable for this run.

Schedule rule:

- Do not ask schedule questions before the first trial report.
- After the first report, ask the human whether they want daily, multiple-times-daily, weekly, manual-only, or another cadence.
- Then write or update `schedule.md` and the relevant automation/config files.

Exact schedule contract:

- Scheduled runs are configured in `daily-content-pipeline/collector/collector_config.json`, or through `POST http://127.0.0.1:17321/config` when the Local Collector app is running.
- Scheduled runs use `scheduled_windows`. They do not use `/jobs/run_now`.
- A daily default schedule should look like this:

```json
{
  "version": "0.1.0",
  "timezone": "local",
  "run_mode": "persistent_bridge_scheduler",
  "default_runs_per_day": 1,
  "poll_interval_seconds": 5,
  "max_sources_per_run": 20,
  "max_scrolls_per_source": 5,
  "max_scrolls_allowed": 10,
  "scroll_delay_seconds": 5,
  "duplicate_filter": {
    "compare_against_previous_day": true,
    "method": "visible_text_matching",
    "parse_html": false
  },
  "scheduled_windows": [
    {
      "name": "daily_morning",
      "enabled": true,
      "local_time_start": "09:00",
      "local_time_end": "09:30",
      "days": ["mon", "tue", "wed", "thu", "fri", "sat", "sun"]
    }
  ],
  "clients": [
    {
      "client_slug": "client-slug",
      "enabled": true,
      "sources": [
        {
          "name": "Competitor page or private group name",
          "url": "https://www.facebook.com/groups/example",
          "platform": "facebook",
          "source_type": "private_group",
          "priority": "high"
        }
      ]
    }
  ]
}
```

- For multiple scheduled runs per day, add multiple enabled items to `scheduled_windows`, for example `morning`, `midday`, and `afternoon`.
- For manual-only mode, set all `scheduled_windows[].enabled` values to `false` and rely only on `/jobs/run_now`.
- If the human has not activated private-source monitoring yet, configure the recurring schedule as public-only and clearly mark private sources as `pending_private_activation`.
- Only configure scheduled private-source collection after Local Collector activation is accepted and collector health is confirmed or explicitly documented as pending/blocker.
- The Local Collector app must run in persistent mode for unattended scheduled collection:

```text
collector-bridge --host 127.0.0.1 --port 17321 --config-file daily-content-pipeline/collector/collector_config.json --output-dir daily-content-pipeline/collector/inbox --persistent
```

- The Solo Agency Local Collector extension polls `/status`; when the current local time is inside an enabled `scheduled_windows` item and private sources exist, `/status` should expose a scheduled job with `current_job_type: scheduled` and `job_available: true`.
- Scheduled run IDs are generated by the Local Collector app, usually using `YYYY-MM-DD_schedule-name`.
- The agent must still write a human-readable `schedule.md` explaining the cadence, clients included, private-source limits, and notification behavior.

---

## 15. Daily Run Algorithm

For each daily run:

1. Load `clients_index.md`.
2. Identify all clients with `active` status.
3. For each active client:
   1. Load the client's Client Intelligence Profile file.
   2. Validate required fields.
   3. If the Client Intelligence Profile is incomplete, enter setup repair mode.
   4. Prepare the current month folder key `YYYY-MM`.
   5. Check public sources.
   6. Use Google Search or an available equivalent search tool with one or more rotating keywords from `public_search_keywords`. Include both primary-industry keywords and a smaller rotation of related-industry keywords. If results are weak, try a different keyword cluster before giving up.
      - Record every keyword used, keyword type, result quality, useful URLs, and final keyword status.
      - Include this record in the daily report section `Public Search Keywords Used Today`.
      - If no search was possible, explicitly explain the blocker in that same section.
   7. If private sources are configured but not yet activated, do not attempt private collection during this run. Mark them as `pending_private_activation`, include the activation CTA in the report, and continue with public sources.
   8. If private sources are activated, start or connect to the localhost collector bridge according to `collector_config.run_mode`.
   9. If private sources are activated, check and update `daily-content-pipeline/collector/collector_setup_status.md` before deciding whether private collection is available.
   10. Check private collector health through `GET http://127.0.0.1:17321/status` when the Local Collector app is expected to be running.
      - If the bridge is offline, try to start it if allowed, otherwise prepare an absolute-path user command and mark private collection as unavailable for this run.
      - If the bridge is online but `extension_health.status` is `stale` or `no_extension_check_yet` after the 75-second extension check grace window, mark private collection as unavailable for this run and notify the human.
      - If `extension_health.status` is `recent`, continue private collection.
   11. Prepare the private-source queue if private sources are available and collector health is acceptable:
      - keep the active daily queue around 20 sources or fewer per client by default;
      - prioritize sources most relevant to the client, target audience, target location, pain points, and content pillars;
      - classify extra sources as `weekly` or `optional` and rotate them across future runs;
      - do not run aggressive or parallel private-source scans for the same logged-in account.
   12. Check private sources if available, using the Solo Agency Local Collector extension plus the Local Collector app when available, with `collector_config.scroll_delay_seconds` defaulting to 5 seconds and `collector_config.max_scrolls_per_source` defaulting to 5.
   13. If the collector bridge was started in `agent_on_demand` mode, stop it after collection completes or after timeout.
   14. Log skipped, pending-activation, expired, rate-limited, warning-triggered, collector-unavailable, extension-unavailable, Chrome-not-running, stale-extension, bridge-offline, or unavailable private sources.
   15. Load yesterday's private data for this client when available and filter duplicate or near-duplicate data points using visible text matching. Do not parse private-platform HTML for duplicate detection.
   16. Extract relevant `[data_points]`, including reference URLs for every data point. Keep data points that are directly about the primary industry or clearly connected through a related industry. Discard related-industry data when the bridge back to the client's offer is weak.
   17. Add newly recommended private groups/pages/profiles/communities to `New Private Sources Detected` and `history/YYYY-MM/new_private_sources_log.md`.
   18. Detect hot and warm leads, including profile URLs, post/current URLs, safe summaries, and reasoning.
   19. Detect direct, adjacent, and audience competitors, including profile URLs, post/current URLs, and positioning notes.
   20. Generate the 3x2 idea matrix, labeling each idea as `primary_industry` or `related_industry`.
   21. Check `history/YYYY-MM/content_log.md`, including the recent primary/related ratio.
   22. Select the best idea of the day.
   23. Write the configured WideCast-writing-skill draft using the writing skill fallback if MCP/account is unavailable.
   24. Save `outputs/YYYY-MM/YYYY-MM-DD.md` as the canonical source-of-truth report.
   25. Generate `outputs/YYYY-MM/YYYY-MM-DD.html` as a polished standalone human-facing report. It must be factually aligned with the Markdown report, mobile-friendly, and include editable draft review blocks when drafts exist.
   26. Update or copy `outputs/latest.md`.
   27. Update or copy `outputs/latest.html`.
   28. Update `history/YYYY-MM/content_log.md`.
   29. Update `history/YYYY-MM/data_sources_log.md`.
   30. Update `history/YYYY-MM/lead_log.md`.
   31. Update `history/YYYY-MM/competitor_log.md`.
4. Create or update `outputs/YYYY-MM/YYYY-MM-DD_master_digest.md`.
5. Generate `outputs/YYYY-MM/YYYY-MM-DD_master_digest.html` as a polished standalone human-facing master report.
6. Update or copy `outputs/latest_master_digest.md`.
7. Update or copy `outputs/latest_master_digest.html`.
8. Present the daily digest to the human.
9. If WideCast MCP notification/Telegram capability is available, send a notification to the human that includes the agent identity, run status, HTML report path/link, clients processed, blockers, lead/competitor counts, and required actions.
9. If another authorized channel can send the HTML file or link more conveniently, use it.
10. Log the notification attempt in `notifications/notification_log.md`.

The daily run is complete only when every active client is processed or explicitly logged as skipped.

When presenting the daily idea list to the human, include reference URLs next to data points, top ideas, and the selected best idea so the human can verify the information. For private data, include the captured source URL and note that it may require the human's logged-in session.

Scheduled runs must assume the human may not be present in the AI agent UI. The run is not fully operationally complete until the mobile-friendly HTML result or a result-ready notification with the HTML path/link has been sent through the configured notification channel, preferably WideCast MCP / Telegram.

---

## 16. Setup Repair Mode

If a Client Intelligence Profile file exists but is incomplete, stale, or inconsistent:

1. Infer missing values where possible.
2. Research missing values where possible.
3. Show proposed repairs to the human.
4. Ask the human to correct only what is wrong.
5. Update the Client Intelligence Profile file.
6. Continue the daily run.

Do not discard existing user-provided values unless the human confirms.

---
