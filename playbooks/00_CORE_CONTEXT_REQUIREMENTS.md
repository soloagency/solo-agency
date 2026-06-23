# Core Context Requirements

Stage: `00`

## Load Rule

Load first for every setup or run. This stage contains the core reasoning model, non-negotiables, A-H workflow, source logic, lead/competitor logic, adjacent reasoning, idea matrix, best-idea selection, and language/report rules.

## Hard Gates For This Stage

- The agent must infer before asking.
- The agent must preserve related-industry 80/20 logic.
- The agent must detect leads and competitors while researching.
- The agent must load Stage 10 before presenting lead/competitor opportunities, report comments, or lead/competitor logs.
- The agent must keep user-facing language aligned with the human's language.
- The agent must never treat public-only research as private-source coverage.
- The agent must explain marketing, analytics, and technical terms in plain language when speaking to a non-technical/non-marketing human.

## Source Preservation Rule

This file is detailed source material moved from the original monolithic `SOLO_AGENCY_PLAYBOOK.md`.

Do not summarize away requirements, examples, checklists, schemas, protocols, URLs, edge cases, warnings, approval gates, or completion gates. If a downstream agent needs to shorten its response to the human, it may summarize the response, but it must still obey the full requirements in this file.

---

## 0. Latest Delta Requirements And Modularization Plan

This section records the latest requirements that must be treated as part of the source of truth before any future split into child playbooks.

The original monolithic playbook remains the base manuscript, but the agent must also preserve and implement the requirements in this section. If the playbook is later split into modular files, do not summarize away any requirement from this section.

### Thin Root And Child Playbooks

Future modular versions must use a thin root `SOLO_AGENCY_PLAYBOOK.md` as an index/router, with detailed instructions moved into child playbooks under `playbooks/`.

The root file should contain only:

- mission;
- first agent instruction;
- stage map;
- missing-playbook download rule;
- mandatory setup flow;
- visible checklist;
- non-negotiable summary;
- completion gates;
- self-audit summary;
- routing instructions.

Detailed protocols must live in child playbooks.

Recommended child playbooks:

- `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`: full A-H workflow, examples, reasoning rules, lead/competitor rules, related-industry rule, language/report rules.
- `playbooks/01_BASIC_PROFILE_PUBLIC_REPORT.md`: setup interview, inference-first profile, public research, keyword rotation, first agency run and HTML report.
- `playbooks/PRIVATE_SOURCE_GATE.md`: short anti-drift gate for any private/logged-in group/feed/profile/community scan request.
- `playbooks/02_PRIVATE_SOURCE_SETUP.md`: manual private sources, optional private source discovery, Local Collector activation, source discovery, first private scan, private-enhanced report update.
- `playbooks/03_PRODUCTION_DISTRIBUTION.md`: writing drafts, production provider setup, video/blog/social creation, publishing, notifications. Do not name this file after any specific vendor.
- `playbooks/04_DAILY_SCHEDULE.md`: manual/daily/weekly routine and schedule.
- `playbooks/05_MEASURE_LEARN_IMPROVE.md`: published content analytics, comments, 7-day measurement, learning loop.
- `playbooks/06_AGENCY_REPORT_STANDARD.md`: agency-grade mobile HTML report standard.
- `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md`: folder structure, client profile schema, logs, ledgers, history.
- `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md`: Solo Agency Local Collector app, Chrome extension, localhost API, health, run-now, source discovery, schedule config.
- `playbooks/09_AGENCY_OPERATIONS_SAFETY_AUDIT.md`: multi-client ops, approvals, safety, regulated industries, self-audit, completion gates.
- `playbooks/10_LEAD_COMPETITOR_DETECTION.md`: deep lead and competitor opportunity detection, scan depth rules, post-link preservation, value-first copy-ready comments, and report/storage requirements.

If the `playbooks/` folder is not available, the agent must download the needed child playbook from:

```text
https://raw.githubusercontent.com/soloagency/solo-agency/main/playbooks/
```

### Canonical User-Facing Description Rule

When explaining what Solo Agency does, the agent must not describe it as only researching, finding ideas, writing drafts, and publishing.

The explanation must include production explicitly:

- researches the market every day;
- finds source-backed content ideas, hot/warm leads, and competitors;
- writes approval-ready scripts/blogs/captions;
- after human approval and provider setup, creates video/blog/social assets through connected production tools;
- can publish approved content to 10+ connected platforms when authorized;
- measures results and feeds the learning into the next run.

In Vietnamese, a good concise explanation is:

```text
Mỗi ngày hệ thống tự nghiên cứu thị trường, tìm ý tưởng nội dung có dẫn nguồn, phát hiện lead nóng/ấm và đối thủ, viết sẵn kịch bản/blog/caption để bạn duyệt; sau khi bạn duyệt và đã kết nối provider, nó có thể tự sản xuất video/blog/social assets, đăng nội dung đã duyệt lên 10+ nền tảng, đo lường kết quả, rồi dùng dữ liệu đó để cải thiện vòng sau.
```

Do not imply that production is only a manual copy/paste step. Also do not imply that rendering, publishing, spending credits, face clone, voice clone, or outreach happens without explicit human approval.

### Required Visible Setup Checklist

The agent must show and update this checklist during setup so the human can catch missed steps.

This is a human-facing progress checklist, not an internal agent instruction list. Use the human's language. Use `You`/`Bạn` for the actions the human must provide or approve, and `I`/`Tôi` for the actions the agent performs. Do not display internal verbs such as "Ask", "Infer", "Select", or "Run" as if the human were reading agent instructions.

The checklist must not assume the human understands marketing or technical terms. Explain terms directly in the checklist or immediately below it. Required meanings:

- `nguồn công khai`: website, Google/tìm kiếm, báo, diễn đàn/trang công khai the agent can access without the human's login.
- `nguồn riêng tư`: logged-in or membership-based sources such as Facebook groups/pages, X, LinkedIn, Instagram, TikTok, YouTube, Reddit, GitHub areas that require access, Discord/Slack communities, competitor profiles, newsletters, or private forums.
- `Local Collector`: local app plus Chrome extension on the human's computer; it uses the already logged-in Chrome session, reads approved visible pages only, and keeps private data local by default.
- `offer`: business promise/package/value proposition.
- `pain points`: customer problems, worries, objections, or urgent questions.
- `content pillars`: repeatable main content themes.
- `lead`: potential-customer or buying-signal.
- `Lead & Competitor Opportunities`: report section where lead/competitor signals become reviewable opportunities with source links, context, and a suggested value-first comment the human can copy.
- `PDNA`: Production creates real assets, Distribution posts/sends approved outputs, Notification sends reports/blockers, Analytics measures performance.
- `learning loop`: using results to improve the next run.

For Vietnamese humans, use this wording:

```text
Solo Agency onetime setup
[ ] 1. Bạn cung cấp sản phẩm/dịch vụ, nghề, chuyên môn hoặc mô tả doanh nghiệp
[ ] 2. Tôi tự suy luận ngành, ngành phụ, ngành liên quan, đối tượng, offer (gói giá trị/lý do khách hàng nên mua)
[ ] 3. Tôi tự suy luận pain points (vấn đề/nỗi đau khách hàng) và content pillars (chủ đề nội dung chính)
[ ] 4. Tôi tự tìm/chọn nguồn công khai (website, Google/tìm kiếm, báo, diễn đàn/trang công khai không cần tài khoản của bạn) và từ khóa tìm kiếm
[ ] 5. Bạn cung cấp nguồn riêng tư nếu muốn (nhóm/profile/trang/kênh social hoặc cộng đồng cần đăng nhập như Facebook, X, LinkedIn, GitHub riêng, Discord...); tôi chỉ kích hoạt Local Collector (app/extension chạy trên máy bạn, giữ dữ liệu local) nếu bạn cho phép
[ ] 6. Tôi cấu hình lịch/routine tự động (giờ và tần suất chạy)
[ ] 7A. Nếu bạn đã cung cấp nguồn riêng tư, tôi hướng dẫn bạn cài/kích hoạt Local Collector (app/extension chạy trên máy bạn, dùng Chrome đã đăng nhập và giữ dữ liệu local) để lần chạy đầu có thể lấy dữ liệu từ các nguồn đó; nếu bạn muốn chạy nhanh trước, tôi giữ nguồn riêng tư ở trạng thái pending
[ ] 7B. Tôi chạy lần đầu: quét nguồn công khai và nguồn riêng tư đã kích hoạt (hoặc public-only nếu 7A chưa xong/được hoãn), tạo HTML report (báo cáo mở bằng trình duyệt/điện thoại), bảng ý tưởng, cơ hội lead/khách hàng tiềm năng và đối thủ kèm link bài viết + comment gợi ý để bạn copy khi phù hợp, và bản nháp kịch bản/blog/caption đầu tiên
[ ] 8. Tôi trợ giúp bạn thiết lập PDNA: Production (tạo tài sản thật như video/blog/social), Distribution (đăng/phân phối), Notification (gửi report/cảnh báo), Analytics (đo hiệu quả) nếu bạn muốn biến bản nháp thành tài sản thật và tự động phân phối/đo lường
[ ] 9. Từ lần chạy thứ hai, nếu đã setup PDNA, tôi quét analytics/số liệu hiệu quả các URL đã đăng trong 7 ngày gần nhất
[ ] 10. Tôi cập nhật report, bảng ý tưởng, ý tưởng tốt nhất, cơ hội lead/khách hàng tiềm năng và đối thủ, bản nháp, analytics/statistics, và learning loop (dùng dữ liệu để cải thiện lần chạy sau)
```

Checklist integrity rule:

- Every setup progress block must show all 10 numbered items in order, including both substeps 7A and 7B.
- Never hide steps 5-10 because they are pending, declined, blocked, or not applicable yet.
- Use `[ ]` for pending items, `[x]` for completed items, and `[-]` only after the human has explicitly declined or the item has been logged as blocked/not applicable.
- Do not ask source discovery as a separate checklist item or gate. If extra private-source discovery is useful, describe it plainly as optional source discovery and ask for approval only when needed.
- Step 6 is the one-time schedule/routine setup. It should happen before the first full agency run so future automation is already defined.
- Step 7A is the private-source activation checkpoint. If private sources were provided/approved and Local Collector is not installed, running, and healthy, 7A becomes the next required question after step 6. The agent must either guide Local Collector setup or ask whether the human wants to run public-only first while keeping private sources pending.
- Step 7A may be marked `[-]` only when no private sources exist, the human declines/postpones Local Collector, or the human explicitly chooses a public-only first run. The reason must be shown in plain language.
- Step 7B is the small win: report plus useful drafts. It must state whether the run will use public-only data or public plus activated private sources. After step 7B, the agent must not ask `làm video luôn không?` or start video editing. The next setup question is step 8.
- Step 8 is provider/capability setup only: choose the provider path, connect or document the production/distribution/notification/analytics provider, check notification/publishing/analytics availability, and save the setup status. Notification setup must stay inside this step. It must not expand into open-ended trial video creation, scene editing, rendering, or publishing while onetime setup is still incomplete unless the human explicitly overrides after being told that setup will resume immediately after a short checkpoint.
- Step 9 applies only after PDNA - Production, Distribution, Notification, and Analytics - has been set up and published URL history exists. It must not be marked complete on the first setup run unless PDNA is set up, published URLs exist, and measurable signals already exist. If PDNA is not set up yet or there is no published URL history yet, mark step 9 as `[-]` with the honest reason such as `PDNA not set up yet` or `no published URLs yet`.
- Step 10 is the final onetime setup item and the daily learning-loop outcome. On the first run it uses report/draft/private-source data; from the second run onward it also includes analytics/statistics from step 9.

### Progress And Next-Step Question Rule

While setup, daily run, private-source activation, production setup, publishing, scheduling, or measurement is still incomplete, every human-facing reply that hands control back to the human must include a compact progress block.

During scheduled runs, every human-facing progress update, notification, or report handoff must include `Solo Agency daily run progress`. If the scheduled run sends multiple updates, each update must refresh completed/current/remaining steps.

### Production Branch Anti-Drift Rule

Production/video work can become a tempting branch inside the larger Solo Agency setup. The agent must not let trial video creation, scene editing, rendering, or publishing cause the setup flow to be forgotten.

Default behavior during onetime setup:

- complete provider/capability setup first;
- do not start open-ended trial video creation or editing while steps 9-10 are still pending;
- after provider setup, gently return to the next setup step;
- defer trial video creation/editing until after onetime setup unless the human explicitly insists.

Good Vietnamese transition after provider setup:

```text
Production provider đã nối xong. Để hệ thống agency không thiếu các bước quan trọng, tôi sẽ hoàn tất setup chính: analytics history nếu đã có dữ liệu và learning loop. Sau khi setup xong, tôi có thể quay lại tạo/chỉnh video thử ngay.
```

If the human explicitly asks to create or edit a video before setup is complete, treat it as a short controlled branch:

- save the parent setup checkpoint before entering the branch;
- state that this is a temporary branch and the agent will resume setup at the next checkpoint;
- show a compact parent checkpoint, not the full 16-item setup list, while the branch is active;
- after one natural checkpoint, gently resume the parent setup unless the human explicitly asks to continue the production branch.

For Vietnamese humans, use this compact parent checkpoint format during an active production branch:

```text
Ghi nhớ setup agency: đang tạm dừng ở bước {N}; sau nhánh video này, bước setup tiếp theo là {M}: {nhãn ngắn}.
Nhánh đang xử lý: sản xuất/chỉnh video/blog/social cho {idea/title}.
```

After a natural checkpoint such as provider connected, draft approved, video created, scenes reviewed, final render/export/publish completed, branch blocked, or the human says they are done with the asset, the final question should usually return to the parent setup flow.

Good Vietnamese final question after a branch checkpoint:

```text
Video branch đã tới checkpoint. Tôi quay lại setup agency để hoàn tất nguồn riêng tư và lịch chạy tự động nhé?
```

The progress block must show:

- completed steps;
- the current active step;
- remaining required steps;
- any blocker or human decision needed.

For setup, use the exact title:

```text
Solo Agency onetime setup
```

For other flows, use a specific progress title such as:

```text
Solo Agency daily run progress
Solo Agency production progress
Solo Agency private-source progress
Solo Agency measurement progress
```

If any required step remains and the agent is waiting for the human, the final line of the message must be exactly one clear next-step question.

Do not end with a passive summary, a report link, or a vague statement such as "let me know what you think."

Good final lines:

```text
Bạn đã cung cấp nguồn riêng tư nhưng Local Collector chưa bật. Bạn muốn tôi hướng dẫn bật Local Collector ngay để lần chạy đầu có dữ liệu nguồn riêng tư, hay chạy public-only trước và giữ nguồn riêng tư ở trạng thái pending?
```

```text
Do you want to create the video from Version 1 now?
```

```text
Do you want daily, multiple-times-daily, weekly, or manual-only runs?
```

Bad final lines:

```text
Here is the report.
```

```text
Let me know if you need anything else.
```

```text
Next steps are in the report.
```

The agent may omit the next-step question only when the entire requested workflow is complete and no human decision is required.

### Manual Private Sources And Discovery Are Independent

Private-source setup must support both paths independently:

1. The human manually provides private sources:
   - competitor profiles;
   - fanpages;
   - Facebook groups;
   - social profiles;
   - KOL accounts;
   - YouTube channels;
   - TikTok accounts;
   - X accounts, lists, or communities;
   - LinkedIn profiles, pages, or groups;
   - Reddit communities;
   - any private or community source they want monitored.

2. The human allows the agent to discover private sources from sources and feeds the user already follows or belongs to:
   - groups the user has joined;
   - profiles/pages/KOLs the user follows;
   - channels the user subscribes to;
   - recommendation feeds;
   - news feed / home feed signals.

The human can choose only manual sources, only discovery, both, neither, or postpone either option.

The agent must not assume discovery replaces manual source input.

### Source Discovery Deep Scroll Rule

There are two separate private collection modes:

1. Source Discovery Mode:
   - Used to discover groups, profiles, pages, channels, KOLs, and communities.
   - Must scroll deeply until no new source names/URLs appear for 3 consecutive scrolls.
   - Use a hard safety cap, for example 80 scrolls.
   - Do not stop at the daily default of 5 scrolls because this can miss many joined groups or followed sources.
   - Filter candidate sources by relevance before asking the human to approve them.

2. Daily Content Monitoring Mode:
   - Used after sources are approved.
   - Default: 5 scrolls per source.
   - Maximum: 10 scrolls per source.
   - Delay: 5 seconds between scrolls.
   - Recommend about 20 daily private sources or fewer per client.

### Private Scan Completion Rule

After private collection runs, the agent must not stop at `collector succeeded`.

Private-source setup or enrichment is complete only when the agent:

1. reads the collected data;
2. extracts relevant data points;
3. detects hot/warm leads;
4. detects competitors;
5. detects new private sources;
6. updates the idea matrix;
7. re-scores or updates the best idea;
8. updates drafts/scripts/blogs if private data changes the recommendation;
9. regenerates the HTML report;
10. shows the updated report to the human.

Collector success alone is not completion.

### Stage Flow After First Agency Run

Stage 1 must take the client from basic profile to first agency run and small-win report/draft.

After the first agency run small win, ask:

- Do you want me to set up PDNA - Production (create real video/blog/social assets), Distribution (publish approved content), Notification (send reports/blockers), and Analytics (measure results) - so approved drafts can become real assets and the system can learn from performance later?

After the first run small win exists, if the human wants production, video/blog/social, publishing, notifications, analytics, or fully automatic operation, load the production/provider setup playbook and complete checklist step 8. Do not ask this before the first useful report and draft unless the human explicitly requests production first.

If the human wants private sources before the first run, load the private-source playbook, complete Local Collector setup if needed, run approved discovery/scan, then include that data in the first agency run or mark it pending honestly.

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
   - future scripts/blogs.

Do not invent metrics. Mark unavailable metrics clearly.

### README Feature Requirement

The human README must include a polished feature line explaining that Solo Agency can, with consent, help discover and collect useful private data sources from groups they joined, profiles/pages/KOLs they follow, subscriptions, and feeds, so social signals are not lost.

Phrase it as a marketing benefit, not a technical implementation detail.

## 1. Non-Negotiable Operating Principles

The agent must follow these principles at all times:

- Preserve every requirement in this playbook.
- Think and infer as much as possible before asking the human anything.
- Ask only for information that is truly required and cannot be inferred, researched, or discovered.
- During setup, ask questions step by step; after every human answer, immediately infer what can be inferred from that answer and show the inference before asking the next question.
- Do not ask the human to define `industry` or `sub_industry`.
- Ask the human first only for the product/service, profession, expertise, or business description.
- Infer `related_industries` after inferring the primary industry and sub-industry. Show those related industries to the human during setup and use them to broaden research and content angles.
- Keep the content strategy anchored around the primary industry: approximately 80% of ideas/scripts should revolve around the primary industry and primary offer, and approximately 20% may use related industries when there is a clear logical bridge back to the client's offer, audience, pain points, or lead-generation goals.
- Ask for `target_location` only if the business is location-dependent and the location cannot be inferred.
- Ask the human to provide private data sources they want monitored, such as competitor profiles, fanpages, groups, communities, or social accounts.
- Ask the human whether they want to include Facebook groups where they are already a member as monitored private data sources; explain that the agent will filter those groups based on whether they contain discussions relevant to the client's primary industry, related industries, audience, location, and pain points.
- If the human wants help finding more private sources, offer optional source discovery in plain language: groups/communities they joined, pages/profiles/KOLs they follow, channels they subscribe to, and platform feeds that recommend relevant content. Explain that this discovery is optional, requires consent, uses the Solo Agency Local Collector, and must be filtered before anything becomes an active private source.
- During private-source setup, repeatedly reassure the human in simple language:
  - They are setting up a professional agency-scale system, so the first setup takes patience but normally happens only once.
  - Private data is saved locally on their own computer and must not be sent outside their computer unless they explicitly approve an export.
  - Once activated, the system can scan daily so important market signals, leads, competitor moves, and content ideas are less likely to be missed.
- When researching public sources, use Google Search or an available equivalent search tool to try primary-industry, related-industry, sub-industry, audience-pain, local, and news-related keywords. Rotate keywords daily or per attempt until the results produce useful data points.
- When scanning private or logged-in sources, use conservative pacing: do not scan aggressively, do not run many private-source browser checks in parallel, and leave a 5 second delay between private-source scroll/read actions so platform feeds have time to load.
- Warn the human not to add too many private sources for one client. As a practical default, keep the daily private-source monitoring list around 20 sources or fewer per client. If the human provides more, prioritize the most relevant sources and rotate lower-priority sources across different days.
- Do not use Claude Chrome Extension for automated private-source collection. It can require repeated human permission clicks and can trap the human in an approval-gated flow. For Claude, use the Solo Agency Local Collector extension plus the Local Collector app, a user-started Local Collector command, or an OS startup service.
- If the conversation drifts and later returns to private-source work, the agent must treat that as a fresh private-source turn. Before scanning, opening, monitoring, or collecting any logged-in/private group, feed, profile, page, community, or source, reload `playbooks/PRIVATE_SOURCE_GATE.md`, Stage 2, Stage 8, and Stage 9.
- Never use Claude in Chrome, Claude Chrome Extension, Codex built-in/in-app browser, ChatGPT/Gemini/Grok browser, Playwright/Puppeteer/Selenium, a fresh agent-opened browser profile, remote-debugging browser, or any agent-controlled browser for logged-in/private-source collection. Use only the Solo Agency Local Collector extension plus Local Collector app.
- If an AI environment cannot browse private sources reliably, cannot show a headed browser UI, cannot run downloaded executables, or requires per-run browser approvals, use the Solo Agency Local Collector extension plus the Local Collector app as the preferred private data collection layer instead of trying to bypass permission prompts.
- During one-time Local Collector setup/update/repair, the AI agent must not run `setup_collector.sh`, `setup_local_collector.ps1`, `Start Local Collector.cmd`, or the collector binary itself, even if local shell permissions are available. Agent-run setup can happen inside a sandbox/session and be killed after the turn. The agent must prepare the files, then give the human the exact one-line Terminal/PowerShell command to run outside the AI sandbox.
- Local Collector activation requires two human actions in the same setup handoff: run the Local Collector app setup/start command, then load the Solo Agency Local Collector Chrome extension from the absolute `solo-agency-local-collector/LOAD_THIS_EXTENSION_IN_CHROME/` runtime folder. Do not mark private-source monitoring active until both are done and health checks pass.
- When speaking to non-technical humans, do not say `bridge`, `localhost bridge`, `binary`, `daemon`, or `service worker` unless troubleshooting. Say `Solo Agency Local Collector extension` and `Local Collector app`. Explain the Local Collector app as: "a small app running on your own computer that receives data from Chrome and saves local files for the AI agent to read."
- The collector is platform-neutral. Never call it `Facebook collector`, `Facebook Data Collector`, or `collector Facebook`, even when the private sources supplied by the human are currently all Facebook groups/pages. Say `Solo Agency Local Collector extension` and explain that it can collect visible authorized data from configured logged-in web sources such as Facebook, LinkedIn, Reddit, X, Instagram, TikTok, forums, and other browser-accessible private sources.
- First agency run small-win rule: after setup context and routine are saved, the agent should resolve the 7A Local Collector checkpoint if private sources exist. Then it may ask whether to run the first agency run immediately. This gives the human a useful report and draft before asking them to connect production/distribution/notification/analytics providers.
- If the human provided private sources but Local Collector is not active, the first agency report must clearly say that private-source monitoring is not activated yet and requires the Solo Agency Local Collector extension plus Local Collector app.
- Private-source activation gate: the agent must not claim private-source monitoring is active or run scheduled private collection until collector setup has either completed or been clearly documented as blocked in `collector_setup_status.md`.
- Manual private sources and optional source discovery are independent options. The human may provide private-source URLs, approve discovery from joined groups/followed profiles/feeds, do both, decline both, or postpone either option. Do not ask interest-graph discovery as a separate user-facing step.
- Private-source completion gate: after any private scan, the agent must analyze the collected private data and regenerate the idea matrix, best idea, leads, competitors, drafts if needed, and human-facing HTML report. A private scan is not complete merely because the Local Collector successfully collected data.
- The first agency run happens after the profile/source plan, schedule/routine, and 7A Local Collector checkpoint are ready or honestly marked pending/public-only. Ask whether to run it immediately, then use run-now/manual-run behavior rather than waiting for the next scheduled window.
- Ask about the recurring schedule before the first agency run, after the profile and source plan are known. If private sources exist, do not promise scheduled private collection until Local Collector activation is complete or clearly pending/blocked.
- After schedule/routine setup, if private sources exist and Local Collector is pending, do not ask only `Do you want me to run the first agency run now?` Ask whether to activate Local Collector first so private-source data can be included, or run public-only first while keeping private sources pending.
- For non-technical humans, never ask them to copy a long multi-line shell/PowerShell script. Create the script file locally first, then provide exactly one short command to run that file in their own Terminal/PowerShell outside the AI sandbox, or provide one double-clickable launcher path on Windows.
- Do not tell the human to keep the setup/report/instruction browser tab open. After they run the required command or load the extension, they may close the tab. If a Terminal/PowerShell process is used before auto-start is configured, explain that the Local Collector app process may need to keep running until the first agency run finishes, but the browser tab itself is not required.
- Never ask for credentials, passwords, OTPs, cookies, tokens, or raw login secrets.
- Do not require a production-provider account, MCP connection, API key, or installed provider tool just to produce ideas, blog drafts, video scripts, or social captions. Writing must continue by loading the public writing-method fallback protocol in this playbook.
- When provider notification/Telegram capability is available, use it to notify the human about completed scheduled runs, required approvals, session-expired issues, setup blockers, and any important failure because the human may not be present when the schedule runs.
- Report-ready notifications must include the HTML report URL/path. A notification that only says the report is ready but does not include a link/path to the `.html` report is invalid.
- When WideCast notification/Telegram is available and a run produced an HTML report, inspect whether WideCast exposes an HTML-capable report/file/asset upload API. If it does, upload the `.html` report to WideCast first and send the uploaded WideCast report URL. If upload is unavailable or fails, log the exact blocker and send the best available local/hosted `.html` report path/link instead.
- If the agent accidentally sends a report-ready notification without a report URL/path, it must immediately send a correction notification containing the HTML report URL/path and log the correction.
- Show all inferred and researched setup context to the human before treating it as stable.
- Continue with public sources if private sources are missing, not yet activated, or unavailable. If private sources were provided but Local Collector has not been installed yet, label them as `pending_private_activation`, not as silently skipped.
- If a logged-in private session expires, skip that private source, log it, and ask the human to log in again manually.
- Do not publish, post, comment, message, render, create a provider-hosted video, export a video, or spend credits without explicit human confirmation.
- Communicate with the human in the same language the human uses.
- Store internal operational field names and schemas in English unless the human explicitly asks otherwise.
- Write human-facing reports, daily digests, HTML reports, summaries, notifications, approval requests, and client-facing explanations in the language the human uses.
- Search keyword language must follow the target audience's likely search/comment language, not automatically the human's chat language. If the human uses Vietnamese but the target audience is English-speaking homeowners in Orange County, the keyword bank should be primarily English. If the target audience is Vietnamese-speaking homeowners in Orange County, include Vietnamese and English keyword variants and label each keyword's language.
- Content output language should follow the target audience and intended publishing audience unless the human explicitly chooses another language. Reports and setup chat may stay in the human's language even when keywords and content drafts are in the audience language.
- User-facing reports must be HTML. Do not show, send, link, or ask the human to open `.md` reports as the report experience. Markdown files are internal source-of-truth records for the agent, audit trail, history, and future learning.
- Do not make the human open Markdown files to learn what to do next. Human-facing setup guidance, blockers, commands, and next actions must be shown directly in the current chat message, Telegram notification, HTML report, or another human-facing channel.
- When a human action is required, provide a short `Action needed` block directly in chat: one clear purpose, one exact next step, and either one copy-paste command or one absolute folder/file path. Do not say only "see the report", "see the .md file", or "instructions are in collector_setup_status.md".
- When delivering a report, show only the mobile-friendly HTML path or link in chat/notification. Do not show the `.md` report path as a user action. Mention Markdown only as an internal saved record if needed, not as the place the human must open.
- After the first agency report, if private sources are pending activation, keep that status visible and do not claim private scheduled monitoring is active. The next main setup question after the small win is PDNA - Production, Distribution, Notification, and Analytics - not video creation.

---

## 2. Core Human Workflow, Fully Translated And Expanded

This section translates and expands the original human daily content production workflow. The agent must treat this section as binding source material.

### A. Identify The Target Audience And Target Location

The agent must identify the target audience `[target_audience]`, lead type, or people who are likely to become interested in the client's field, service, product, expertise, or profession.

The agent must infer the industry and sub-industry from the client's product/service, profession, expertise, or business description. The agent must not ask the human to manually provide `industry` or `sub_industry` unless inference is impossible after reasonable research.

The agent must also infer related industries `[related_industries]`.

`[related_industries]` are adjacent fields that affect the same target audience, influence buying decisions, create risks/opportunities, or produce news/data signals that can be logically connected back to the client's primary offer.

The agent must show inferred related industries during setup and ask the human to correct them if wrong. The agent should not ask the human to manually list related industries unless the business is too ambiguous to infer.

The content strategy must follow this approximate mix:

- 80% primary industry: ideas directly about the client's core industry, sub-industry, offer, audience, and pain points.
- 20% related industries: ideas inspired by adjacent industries, but only when the agent can explain the bridge back to the client's offer and audience.

Related-industry content must never become random general news. It must answer:

```text
Why would this related-industry signal matter to this client's target audience, and how does it connect back to the client's product/service?
```

Examples of related industries:

- Primary industry: Real Estate
  - Related industries: mortgage, banking, personal finance, home inspection, construction, renovation, zoning, property tax, P&C insurance, relocation, schools, local economic development.
  - Example logic: higher insurance premiums -> higher monthly ownership cost -> buyer affordability changes -> real estate buyers need to recalculate budget before making offers.

- Primary industry: Mortgage
  - Related industries: real estate, credit repair, banking, personal finance, employment, tax planning, insurance, construction, home appraisal.
  - Example logic: layoffs in a local employer sector -> borrower income stability concerns -> buyers should prepare documentation and loan scenarios earlier.

- Primary industry: DUI / Criminal Defense Law
  - Related industries: auto insurance, employment/background checks, immigration, rideshare/nightlife, traffic enforcement, local courts, DMV/license rules.
  - Example logic: holiday enforcement increases -> more DUI stops -> drivers need to know first steps and deadline risks.

- Primary industry: Life Insurance
  - Related industries: health, family finance, estate planning, retirement planning, natural disasters, workplace benefits, mortgage protection, long-term care.
  - Example logic: natural disaster news -> higher accident/death risk awareness -> families should understand whether their life insurance and beneficiaries are ready.

- Primary industry: P&C Insurance
  - Related industries: real estate, climate/weather, auto, construction, home maintenance, local regulation, lending, small business risk management.
  - Example logic: new storm forecasts -> property risk rises -> homeowners should review deductibles before a named storm.

- Primary industry: AI Automation Agency
  - Related industries: marketing, sales operations, CRM, customer support, content production, analytics, recruiting, finance operations, compliance, cybersecurity.
  - Example logic: a new social platform reporting change -> agencies waste more manual time -> automation can centralize reporting.

If the field depends on geography, the agent must identify the target location `[target_location]`.

Examples:

- Real Estate:
  - Possible target audience: people preparing to buy a home, sell a home, compare neighborhoods, monitor housing prices, negotiate offers, refinance, or understand mortgage rates.
  - Typical leads: first-time buyers, move-up buyers, sellers, investors, relocating families, homeowners considering selling.
  - Location dependency: high.
  - Example target location: Austin, Texas; Orange County, California; Miami, Florida.

- Mortgage:
  - Possible target audience: people planning to buy a home, comparing mortgage products, worried about rates, looking for down payment options, or considering refinancing.
  - Typical leads: first-time homebuyers, self-employed buyers, homeowners, investors, VA/FHA borrowers.
  - Location dependency: high or medium depending on licensing and service area.

- Legal:
  - Possible target audience: people asking about legal problems, people who received tickets, people facing DUI charges, accident victims, tenants, employees, immigrants, business owners, or people who need legal representation.
  - Typical leads: people with urgent legal issues or high anxiety about consequences.
  - Location dependency: high because laws, courts, and procedures are local.

- Insurance:
  - Possible target audience: people comparing policies, people who recently had an accident, homeowners, drivers, business owners, families, or people afraid claims may be denied.
  - Location dependency: medium or high depending on insurance type and regulations.

- Local Home Services:
  - Possible target audience: homeowners, landlords, property managers, renters, commercial building owners.
  - Examples: roofing, HVAC, plumbing, landscaping, pest control, cleaning, remodeling.
  - Location dependency: high.

- Tech / SaaS:
  - Possible target audience: founders, operators, creators, marketers, sales teams, agencies, developers, business owners.
  - Location dependency: often low, but can be medium when selling to a specific market.

- Healthcare / Wellness:
  - Possible target audience: people searching for symptoms, treatments, appointments, preventive care, recovery options, or local providers.
  - Location dependency: high for clinics and providers.
  - Compliance sensitivity: high.

The agent must decide whether the industry is location-dependent. If it is, the agent must ensure `[target_location]` is present before running daily research.

If target location is missing and cannot be discovered from business context, client website, profile, social bio, or prior files, ask the human only:

`What target location should this pipeline focus on?`

### B. Infer Audience Needs, Pain Points, And Content Pillars

From `[target_audience]`, the agent must identify the audience's needs, fears, urgent questions, buying triggers, objections, frustrations, confusion, and emotional pain points `[pain_points]`.

From `[pain_points]`, the agent must also infer content pillars, content lines, or recurring content routes `[content_pillars]`. This is mandatory because pain points are not useful for daily production unless they are converted into repeatable content lines.

`[content_pillars]` are the strategic routes the agent will repeatedly use to generate daily ideas. They connect audience pain points to content formats, angles, and lead-generation logic.

Each content pillar should be tagged as either:

- `primary_industry`: directly about the client's main industry/sub-industry and core offer.
- `related_industry`: inspired by a related industry but connected back to the client's offer, audience, or pain points.

The agent should build a content-pillar mix that supports the 80/20 rule:

- Most pillars and most daily ideas should stay in `primary_industry`.
- A smaller set of pillars may use `related_industry` signals to create useful, timely, or differentiated angles.
- Related-industry pillars must include a clear bridge back to the client's offer.

The agent should infer pain points from:

- Business description.
- Public research.
- Competitor content.
- Industry knowledge.
- Local context.
- Search behavior.
- Common customer objections.
- Social discussions.
- Comments and questions from public or private data sources.

The agent must not block setup by asking the human to list pain points manually. It must produce a best-effort inferred draft and show it to the human for correction.

The agent must also show the inferred `[content_pillars]` in the next setup message before asking the next setup question.

Example setup behavior:

1. Human says: "The client is a DUI lawyer in Los Angeles."
2. Agent infers and shows:
   - industry: Legal
   - sub_industry: DUI / Criminal Defense
   - target_audience: drivers facing DUI stops, arrests, license risks, or court dates in Los Angeles
   - likely pain_points: fear of losing license, fear of jail, confusion about court, uncertainty about whether to call a lawyer
   - related_industries: auto insurance, DMV/license rules, employment background checks, immigration consequences, local nightlife/traffic enforcement
   - content_pillars:
     - Emergency guidance: what to do in the first 24 hours (`primary_industry`)
     - Consequence clarity: license, court, insurance, record, job impact (`primary_industry` plus related auto insurance/employment)
     - Mistake prevention: what not to say or do after a stop/arrest (`primary_industry`)
     - Local process education: how Los Angeles / California DUI procedures work (`primary_industry`)
     - Lead-gen angle: why early legal advice can change available options (`primary_industry`)
3. Agent then asks the next necessary question, such as target location only if it was not already known, or asks for private sources to monitor.

The agent must not wait until the final setup summary to reveal content pillars. Every time a human answer changes the business context, audience, pain points, or data source strategy, the agent must update and show the inferred content pillars before asking the next question.

Examples:

#### Real Estate Pain Points

- "Should I buy now or wait?"
- "Are prices going down in my city?"
- "Can I afford a home with current interest rates?"
- "How do I avoid overpaying?"
- "How much should I offer?"
- "Will I regret buying before rates drop?"
- "Is inventory improving?"
- "Which neighborhood is still affordable?"
- "What hidden costs should I expect?"
- "How do I compete without waiving protections?"

Example Real Estate content pillars:

- Market timing: buy now, wait, negotiate, or prepare.
- Affordability clarity: rates, payments, taxes, insurance, and hidden costs.
- Local market intelligence: inventory, neighborhood shifts, zoning, schools, commute, development.
- Buyer mistake prevention: overpaying, weak offers, bad inspections, poor financing preparation.
- Seller strategy: pricing, staging, timing, concessions, and negotiation.
- Lead-gen angle: why preparation beats prediction in a changing market.

Example Real Estate related-industry 80/20 content lines:

- Mortgage / rates: explain how rate moves change buyer budget, but bring the conclusion back to buying strategy.
- P&C insurance: explain how homeowners insurance affects monthly ownership cost, but bring the conclusion back to affordability and offer planning.
- Home inspection / construction: explain inspection or repair risks, but bring the conclusion back to negotiation and buyer protection.
- Zoning / development: explain local planning changes, but bring the conclusion back to neighborhood selection and long-term value.
- Personal finance / taxes: explain property tax, cash-to-close, or emergency-fund pressure, but bring the conclusion back to readiness before touring homes.

In a healthy Real Estate pipeline, about 80% of scripts should directly discuss real estate decisions, local market moves, buying/selling strategy, listings, inventory, negotiation, and client perspective. About 20% may start from related industries such as mortgage, insurance, inspection, construction, taxes, or local development, but every such idea must explicitly connect back to the homebuyer, seller, or investor decision.

#### Mortgage Pain Points

- "What rate can I get right now?"
- "How much house can I afford?"
- "Should I lock my rate or wait?"
- "Can I qualify if I am self-employed?"
- "How much down payment do I really need?"
- "What is the difference between FHA, VA, conventional, and jumbo?"
- "Will one late payment ruin my approval?"
- "Should I refinance now?"

Example Mortgage content pillars:

- Rate decision guidance: lock, wait, refinance, compare scenarios.
- Qualification education: credit, income, down payment, self-employed borrowers.
- Loan product clarity: FHA, VA, conventional, jumbo, bridge, HELOC.
- Payment reality: monthly payment, taxes, insurance, PMI, cash to close.
- Buyer readiness: pre-approval, documents, underwriting risks.
- Lead-gen angle: why the right loan strategy matters more than chasing the lowest advertised rate.

#### Legal Pain Points

- "Do I really need a lawyer?"
- "Will this ticket affect my license?"
- "Can this charge be dismissed?"
- "What happens if I miss court?"
- "Should I talk to police or insurance?"
- "How much trouble am I in?"
- "Can this affect my job, immigration, insurance, or record?"
- "What should I do in the first 24 hours?"

Example Legal content pillars:

- Emergency first steps: what to do immediately after a ticket, arrest, accident, notice, or legal threat.
- Consequence clarity: license, record, job, immigration, insurance, money, court deadlines.
- Mistake prevention: what not to say, sign, ignore, post, or delay.
- Local process education: courts, deadlines, hearings, agencies, local rules.
- Myth-busting: common assumptions that make cases worse.
- Lead-gen angle: why early legal guidance can preserve options.

#### Personal Injury / Insurance Pain Points

- "Will I be compensated after an accident?"
- "What is my claim worth?"
- "Should I accept the first settlement offer?"
- "What if insurance denies my claim?"
- "Who pays medical bills?"
- "What if I was partly at fault?"
- "How long do I have to file?"

Example Personal Injury / Insurance content pillars:

- Claim value education: what affects compensation or settlement range.
- Insurance company behavior: delay, denial, low offers, recorded statements.
- Medical/documentation guidance: treatment, bills, evidence, timelines.
- Fault and liability clarity: partial fault, police reports, witnesses, deadlines.
- Mistake prevention: signing too early, posting online, skipping care.
- Lead-gen angle: why the first offer may not reflect the full cost of the accident.

#### Tech / SaaS / AI Automation Pain Points

- "Which AI tool should I use?"
- "Can this workflow be automated?"
- "How do I reduce manual work?"
- "How do I avoid hiring before I am ready?"
- "How do I integrate tools without breaking operations?"
- "Will AI replace this role?"
- "How do I make content faster without losing quality?"

Example Tech / SaaS / AI Automation content pillars:

- Workflow diagnosis: where time is being lost and what should be automated first.
- Tool clarity: which tools fit which use cases and what to avoid.
- Implementation education: integrations, data flow, permissions, quality control.
- ROI and team impact: time saved, reduced manual work, fewer handoffs.
- Risk management: hallucinations, privacy, approvals, human review.
- Lead-gen angle: practical automation beats flashy AI demos.

#### Local Service Pain Points

- "How urgent is this repair?"
- "How much should this cost?"
- "Can I trust this contractor?"
- "What happens if I delay?"
- "Is this covered by insurance?"
- "How do I avoid being overcharged?"

Example Local Service content pillars:

- Urgency education: what needs immediate repair and what can wait.
- Cost transparency: price ranges, hidden costs, quotes, warranties.
- Trust and quality: how to choose a provider, red flags, proof of work.
- Prevention and maintenance: seasonal checks, early warning signs, long-term savings.
- Insurance or compliance clarity: what is covered, required, or risky.
- Lead-gen angle: the cheapest fix can become expensive if the root problem is missed.

### C. Identify Data Sources

After identifying `[target_audience]`, `[target_location]`, and `[pain_points]`, the agent must identify data sources `[data_sources]`.

Data sources are used to collect relevant signals, news, questions, debates, objections, trends, and opportunities that can attract the target audience.

Data sources have two main layers.

#### C1. Public Data Sources

Public data sources are accessible without an account.

Examples:

- Google Search results for rotating primary-industry, related-industry, sub-industry, audience-pain, local, and news keywords.
- Industry websites that people inside the field know.
- Specialist blogs.
- Public newsletters.
- Public government websites.
- Local city or county updates.
- Local news sections.
- National news sections.
- Public market data.
- Public company pages.
- Public social posts.
- Public Reddit posts.
- Public YouTube channels.
- Public competitor websites.
- Search result pages.
- Public databases.

Examples by industry:

Real Estate:

- Redfin Data Center
- Zillow Research
- Realtor.com research
- Local MLS reports when publicly available
- Local city planning and zoning pages
- Housing sections in CNBC, NBC, NYT, Bloomberg, local news
- Local property tax authority pages
- Local construction permit dashboards
- Public neighborhood development pages

Mortgage:

- Freddie Mac Primary Mortgage Market Survey
- Mortgage News Daily
- Federal Reserve releases
- FRED economic data
- CFPB consumer mortgage guidance
- Bankrate mortgage pages
- Local housing affordability reports
- State housing finance agency pages

Legal:

- State court websites
- Local court updates
- DMV or equivalent transportation authority pages
- State bar consumer resources
- Local legal news
- Police department public updates
- Public law firm blogs
- Statutory resources
- Public court calendars or case search pages when legally appropriate

Personal Injury / Insurance:

- State insurance department pages
- Consumer protection pages
- Local accident reports
- Public safety reports
- Insurance industry updates
- Public claim guidance
- Competitor public blogs

Tech / AI:

- Product changelogs
- Official company blogs
- Hacker News
- Product Hunt
- GitHub releases
- TechCrunch
- The Verge
- AI newsletters
- Public founder/operator communities

Healthcare:

- CDC
- NIH
- Local public health departments
- Hospital public resources
- Medical association pages
- Public education pages

#### Public Search Keyword Bank And Rotation

During public-source research, the agent must use Google Search or an available equivalent search tool to discover relevant public data sources and current discussions.

The agent must not rely only on generic industry keywords.

Generic industry keywords are useful for broad context, but the main keyword bank must come from the target audience's real problems, worries, questions, needs, objections, buying triggers, comparison behavior, and local context.

The agent should generate a large keyword bank from:

- `industry`
- `sub_industry`
- `target_location`
- `target_audience`
- `pain_points`
- `content_pillars`
- client offer
- customer problems and issues
- urgent questions
- buying-intent phrases
- objection phrases
- comparison phrases
- cost, risk, delay, deadline, mistake, coverage-gap, eligibility, renewal, non-renewal, cancellation, denied-claim, price-increase, safety, compliance, legal/process, or "what to do" phrases when relevant
- seasonal events
- current news context
- local terms, neighborhoods, courts, agencies, regulations, or communities where relevant

Keyword language rule:

- Each keyword must have a `language`.
- Choose keyword language based on the target audience's likely search/comment language.
- If the target audience is multilingual, create separate keyword variants per useful language and label them, for example `en`, `vi`, or `es`.
- Do not translate all keywords into the human's chat language by default.
- Human-facing explanations can be in the human's language while the actual keyword strings remain in the audience/search language.
- If content will be published in one language and research signals are stronger in another, keep both when useful: `keyword_language` for research and `content_output_language` for drafts.

Required keyword groups:

- `industry_general`: broad industry/sub-industry context keywords.
- `pain_point`: direct customer pain, fear, problem, objection, or urgent-question keywords. This must be one of the largest groups.
- `need_or_goal`: customer need, desired outcome, prevention, savings, safety, protection, approval, eligibility, or confidence keywords.
- `buying_intent`: phrases that indicate the person may be comparing, choosing, hiring, buying, renewing, switching, or seeking help.
- `local_context`: location, neighborhood, county, state, agency, regulation, court, weather, risk, community, or local-market keywords when location matters.
- `related_industry`: adjacent-industry keywords only when the bridge back to the client's offer and audience is clear.
- `trend_news`: current event, seasonal, regulation, market change, or deadline keywords.

Initial setup requirement:

- Build a broad keyword bank, not just a short search list.
- Aim for 200+ saved keyword candidates over time per active client.
- On initial setup, generate as many useful keyword candidates as the context allows. If the agent can reasonably generate 100-200 high-quality candidates, do so and save them. If context is still thin, seed the bank with the best available candidates and mark `needs_expansion: true`.
- The bank must contain many pain-point/problem/need keywords. A bank made mostly of generic industry terms is incomplete.
- Store the full bank in the Client Intelligence Profile or source notes; do not show the full bank in chat.
- In chat, show only a compact sample, usually 5-12 keywords from the pain-point/problem/need groups, then say how many more are saved for rotation, for example: `+200 more saved in the keyword bank for daily rotation`.

Examples:

- Homeowners insurance in Orange County:
  - Generic industry keyword: `Orange County homeowners insurance California renewal`
  - Pain-point keyword: `home insurance non renewal what can I do California`
  - Problem keyword: `insurance company dropped my home policy wildfire risk`
  - Need keyword: `how to avoid FAIR Plan California homeowners`
  - Coverage-gap keyword: `home insurance coverage gaps California wildfire`
  - Buying-intent keyword: `best homeowners insurance for fire risk Orange County`
- Real estate in Austin:
  - Generic industry keyword: `Austin housing inventory buyers 2026`
  - Pain-point keyword: `worried about overpaying for a house Austin`
  - Problem keyword: `property tax shock after buying home Austin`
  - Need keyword: `how much cash do I need before making an offer Austin`
- DUI lawyer in Los Angeles:
  - Generic industry keyword: `Los Angeles DUI checkpoint weekend`
  - Pain-point keyword: `will I lose my license after DUI California`
  - Problem keyword: `what happens after DUI arrest California first offense`
  - Need keyword: `how fast do I need a DUI lawyer after arrest`

Daily rule:

- Try a different keyword or keyword cluster each day or each failed attempt. Prioritize pain-point/problem/need clusters before generic industry clusters.
- Keep a `public_search_keywords` queue in the Client Intelligence Profile or source notes.
- Mark keywords as `used`, `useful`, `weak`, or `retry_later`.
- If a keyword returns weak or irrelevant results, revise it by adding local terms, audience pain terms, or buying-intent terms.
- When the agent discovers new phrases in search results, public comments, FAQs, forum posts, private-source scans, competitor hooks, report comments, analytics comments, or human feedback, extract new keyword candidates and add them to the bank if they are not already present.
- Deduplicate and normalize near-duplicates. Keep the human's wording when it reveals a real pain point.
- Record why each new keyword was added, which pain point/content pillar it maps to, and which source or run discovered it.
- Promote keywords that produce useful leads, strong ideas, relevant competitors, or measurable content performance.
- Demote keywords that repeatedly produce weak/noisy results.
- Continue until the agent finds credible results or reasonably concludes that no useful public signal exists for that slot today.
- Do not fabricate trends or news if search results are weak.
- The daily report must include a visible section called `Public Search Keywords Used Today`. Do not hide search queries only in internal logs.
- The setup summary should include a compact section called `Pain-Point Keyword Sample`, not the full keyword bank. Show 5-12 pain-point/problem/need keywords and a line such as `+{N} more saved for rotation`.
- If the agent realizes after generating a report that search keywords were not shown, it must update or append the current report before claiming the run is complete. Do not merely promise to show keywords "from next time."

#### Public Source Learning And Promotion

Public source discovery is not a one-time setup task. Every public run must improve the saved public source list.

During public search and public source reading, the agent must watch for useful new public sources, such as:

- recurring government or regulator pages;
- public news sections;
- specialist blogs;
- public forums or Reddit communities;
- public YouTube channels or playlists;
- public competitor blogs/pages;
- public data dashboards;
- public newsletters or archives;
- public association or industry pages;
- public local/community pages;
- source pages repeatedly cited by credible articles or high-signal discussions.

The agent must classify newly discovered public sources:

- `candidate_public_source`: newly discovered and potentially useful, but not yet proven.
- `active_public_source`: useful enough to revisit in future scheduled runs.
- `weekly_public_source`: useful but not worth checking daily.
- `occasional_public_source`: useful only for specific events, seasons, or topics.
- `weak_public_source`: too broad, noisy, duplicated, stale, or low-signal.
- `blocked_or_unreliable`: paywall, broken, spammy, inaccessible, or unreliable.

Promotion rule:

- Promote a public source to `active_public_source` or `weekly_public_source` when it produces useful ideas, credible evidence, lead signals, competitor signals, recurring audience questions, regulation/market updates, or strong keyword expansion.
- Do not promote every URL found by search. Individual articles can be cited as evidence without becoming recurring sources.
- Prefer recurring sources such as sections, feeds, domains, category pages, author pages, dashboards, public communities, or official pages over one-off article URLs.
- Demote active sources that repeatedly produce weak/noisy/stale results.

Storage rule:

- Save promoted sources into `public_data_sources` in the Client Intelligence Profile or source notes.
- Log each new or changed source in `history/YYYY-MM/data_sources_log.md`.
- Store why the source matters, related pain point, related content pillar, language, source type, cadence, status, first discovered date, last checked date, usefulness score, and whether it should be visited by scheduled runs.

Scheduled run rule:

- Every scheduled run must load saved `public_data_sources` and visit/check the active due sources before or alongside keyword search.
- The run must also use keyword search to discover new sources and update the public source list.
- This creates a loop: saved sources provide continuity, keyword search finds new signals, and useful discoveries become future scheduled sources.

Human-facing display rule:

- Do not dump the full public source list into chat or the daily report.
- Show a compact summary, such as `New public sources added today: 3`, with 1-3 strongest examples and why they were added.
- If no source was added, say whether no useful new source was found or source discovery was not possible today.

#### C2. Private Data Sources

Private data sources require a login, account, membership, or already logged-in browser session.

Examples:

- Competitor Facebook fanpages.
- Competitor Instagram profiles.
- Competitor TikTok profiles.
- Competitor LinkedIn profiles or company pages.
- Facebook groups.
- LinkedIn groups.
- Private Reddit communities or logged-in Reddit feeds.
- Niche forums.
- Local community groups.
- Slack or Discord communities.
- Client dashboards.
- Social media feeds visible only after login.

The agent must ask the human to provide private data sources they want monitored.

Good question:

`Do you want to provide any private sources for this client? Private sources means logged-in/social/community places such as competitor profiles, fanpages, Facebook groups, LinkedIn pages, Reddit communities, Discord/Slack communities, niche forums, newsletters, or dashboards that may require your account or membership. These are different from public sources such as websites, Google/search results, public articles, and public pages I can access without your login. If you provide private sources, I will only activate collection with your permission, using the Solo Agency Local Collector local app/extension on your computer. It uses your already logged-in Chrome session, reads approved visible pages only, and keeps data local by default. Do not share credentials, cookies, passwords, OTPs, or tokens. For account safety and platform-respectful monitoring, around 20 private sources or fewer per client is a good daily default; if you provide more, I will prioritize and rotate them.`

Bad questions:

- "What is your Facebook password?"
- "Send me your cookies."
- "Give me your login token."
- "What is the OTP code?"

Private data sources must match the client, target audience, target location, and pain points.

For location-dependent industries, location match is critical.

Private data source pacing rule:

- Do not scan private sources in a rushed or aggressive way.
- Do not open or scrape many logged-in pages at the same time.
- Use a 5 second delay between private-source page loads, scroll actions, major read actions, and source transitions when the agent environment allows timing control.
- For each private source, default to `max_scrolls_per_source: 5`.
- Allow the human to configure up to `max_scrolls_per_source: 10`.
- Never exceed 10 scrolls per private source in one run unless the human explicitly changes the collector code and accepts the account-risk tradeoff.
- Prefer fewer, higher-quality private sources over a large noisy list.
- Keep the active daily private-source list around 20 sources or fewer per client by default.
- If the human provides more than about 20 private sources, classify them as `daily`, `weekly`, or `optional`, then rotate non-daily sources instead of scanning all of them every day.
- Warn the human that adding too many private sources or scanning too aggressively may trigger platform warnings, temporary limits, or account review. The agent must not attempt to bypass platform restrictions.

Private recommendation discovery rule:

- While browsing Facebook or another private platform, if the platform visibly recommends related groups, pages, communities, creators, or sources that appear relevant to the client's primary industry, related industries, target audience, target location, pain points, or content pillars, collect them as possible new sources.
- Do not automatically add every recommended group to the active daily scan list.
- Store them in the daily output under `New Private Sources Detected`.
- Include source name, platform, profile/group URL, current recommendation URL, why it appears relevant, estimated priority, and suggested scan cadence.
- Mark each as `needs_human_review` unless it is clearly a public source or the human previously authorized auto-adding similar sources.
- Do not join groups, follow pages, message admins, or request access unless the human explicitly approves.

Examples:

- A Los Angeles DUI lawyer should monitor Los Angeles or California legal, traffic, DUI, court, police, or competitor sources.
- An Austin real estate agent should monitor Austin neighborhoods, Austin housing data, Austin competitor pages, local Facebook housing groups, and Austin development news.
- A Miami insurance agency should monitor Florida insurance regulation, hurricane risk, local accident or property damage discussions, and competitor pages.

### D. Collect Data From Sources

Once A, B, and C are available, the agent must use appropriate tools to collect data.

For public sources, the agent may use:

- Web browser.
- Search tools.
- Web extraction tools.
- DOM or source inspection.
- RSS or newsletter feeds.
- Public APIs.
- Screenshots and OCR when necessary.
- Manual reading and summarization.

For private sources, the agent must use:

- Solo Agency Local Collector extension plus the Local Collector app.
- The human's already logged-in Chrome session as accessed by the Solo Agency Local Collector extension.
- Local Collector output files, localhost status, and run-now/scheduled jobs.

For private sources, the agent must not use:

- Claude in Chrome or Claude Chrome Extension.
- Codex browser, Codex in-app browser, or browser tools controlled directly by Codex.
- ChatGPT/Gemini/Grok browser surfaces.
- Playwright/Puppeteer/Selenium controlled directly by the AI agent.
- Fresh agent-opened browser profiles, exported browser profiles, storage state, cookies, tokens, passwords, or OTPs.
- Hermes, OpenClaw, or other agents using logged-in browser contexts directly.

The agent must not ask for credentials.

Private-source collection must be paced conservatively:

- Before moving from one private source to the next, wait 5 seconds when the environment supports delays.
- When scrolling, expanding comments, opening posts, or reading multiple items from a private source, leave 5 seconds between major actions when feasible.
- Default to 5 scrolls per private source.
- Allow the human to configure up to 10 scrolls per private source.
- Do not run multiple private-source browser scans in parallel for the same logged-in account unless the human explicitly accepts the account-risk tradeoff.
- Do not use stealth, credential sharing, cookie extraction, token reuse, platform bypassing, or other methods intended to defeat platform restrictions.
- If a platform displays warnings, rate limits, checkpoints, unusual-activity prompts, or account review messages, stop scanning that platform, log the issue, and notify the human through the configured notification channel.
- If there are too many private sources for a safe daily run, prioritize high-relevance sources and rotate the rest.

The agent must collect by:

- Opening the page.
- Scrolling.
- Reading visible text.
- Reading visible text and browser-visible metadata.
- Extracting headlines, post text, comments, captions, dates, engagement hints, and repeated questions.
- Capturing the source URL for every useful finding.
- For private sources, capturing the URL visible at the time the data point was collected so the human can verify it later from their own logged-in session.
- Identifying patterns and signals.
- Filtering out irrelevant information.

The agent must not depend on fragile HTML parsing for private social platforms. Facebook, X, Reddit, LinkedIn, Instagram, and TikTok can change markup frequently. Prefer visible text, accessible labels, current URL, profile URL candidates, post/current URL candidates, timestamps visible to the human, and engagement text visible on screen.

Before accepting private-source data points for today's report:

- Load yesterday's collected private data for the same client when available.
- Compare new visible text summaries against yesterday's text using text matching.
- Remove exact duplicates.
- Remove near-duplicates when the same source, same current URL/post URL, or highly similar text already appeared yesterday.
- Keep updated items only if the new version has materially new comments, engagement, date, URL, or context.
- Record skipped duplicates in source status notes when useful.

The agent must keep only data relevant to:

- `[target_audience]`
- `[target_location]`
- `[pain_points]`
- Client business offer
- Compliance constraints
- Daily content opportunity

All useful collected findings are called `[data_points]`.

Every data point must include a reference URL.

For public data points, include the public URL.

For private data points, include the private/source URL captured at collection time. The URL may require the human's logged-in session to open. Do not expose credentials, cookies, tokens, screenshots of private personal data, or unnecessary private content.

Examples of valid data points:

- "Austin inventory rose again this month, and buyer comments show confusion about whether this creates negotiation leverage."
- "A competitor DUI lawyer post about license suspension received high engagement."
- "A local Facebook group has repeated questions about property tax increases."
- "Mortgage rates changed this week, and buyers are asking whether to lock or wait."
- "A Reddit thread shows accident victims are confused about accepting early insurance settlement offers."
- "The city announced a zoning update that may affect future home supply."

Examples of invalid or weak data points:

- "A random celebrity bought a house." Not relevant unless it affects target audience.
- "A national legal scandal." Not relevant unless logically connected to local legal leads.
- "A generic AI trend." Not useful unless it maps to the client's offer and target audience.

### Lead Detection Rule

While scanning public and private sources, the agent must also detect potential leads, not only content ideas.

This means the pipeline is both:

- an idea engine, and
- a lead discovery engine.

Before presenting or storing lead opportunities, load Stage 10: `playbooks/10_LEAD_COMPETITOR_DETECTION.md`.

The agent should classify detected leads into hot, warm, and watch opportunities. Hot and warm are the main daily report levels:

#### Hot Leads

Hot leads are people, accounts, businesses, or organizations that explicitly show a current need related to the client's product/service.

Examples:

- A Facebook group member asks, "Does anyone know a DUI lawyer in Los Angeles?"
- A homeowner posts, "Our roof is leaking after the storm. Who should I call?"
- A buyer asks, "Can I still qualify for a mortgage if I am self-employed?"
- Someone asks, "What happens to life insurance if death happens during a natural disaster?"
- A business owner says, "We need someone to automate our client reporting."

#### Warm Leads

Warm leads do not explicitly ask to buy, but the context suggests they may become a customer if approached with the right education, offer, or timing.

Examples:

- Someone complains that their insurance premium increased and they do not understand why.
- A renter says they are thinking about buying next year.
- A person shares confusion about legal consequences after receiving a ticket.
- A local business owner describes a repetitive manual workflow that an automation agency could solve.
- A family discusses financial risk after a natural disaster, even if they do not mention life insurance directly.

The agent must list detected leads inside the `Lead & Competitor Opportunities` section of the daily output, or a natural same-language title for the report language.

For each lead, include:

- Lead level: hot | warm | watch.
- Source.
- Profile URL: the person, account, page, group member profile, business profile, or organization profile URL when visible and appropriate to store.
- Post/current URL: the exact post, comment thread, group post, search result, page, or current browser URL where the lead signal was captured.
- Captured at.
- Public/private source type.
- What the person/account said or did, summarized safely.
- Why this may indicate demand.
- Related client service/offer.
- Related pain point.
- Suggested next action.
- Copy-ready suggested comment in the same language as the post, written to add value without directly advertising the user's service.
- Outreach risk/compliance note.

The agent must not expose unnecessary private personal data. Summarize safely. Do not copy sensitive personal details unless they are essential and the human is authorized to see them.

If a profile URL is not visible, not available, or unsafe to store, write `unavailable` and keep the post/current URL. If the post/current URL is unavailable, write `unavailable` and explain why in notes. Do not try to extract hidden profile IDs, private contact details, emails, phone numbers, cookies, tokens, or tracking parameters.

The agent must not contact, message, comment, reply, scrape contact info, or engage the lead unless the human explicitly approves that action. Lead detection is allowed; lead outreach requires separate approval.

Detected leads should be stored in `history/YYYY-MM/lead_log.md` and, when possible, `history/YYYY-MM/lead_competitor_opportunities.jsonl`.

### Competitor Detection Rule

While scanning public and private sources, the agent must also detect competitors and competitor-like accounts, not only content ideas and leads.

Before presenting or storing competitor opportunities, load Stage 10: `playbooks/10_LEAD_COMPETITOR_DETECTION.md`.

Competitor detection includes:

- Direct competitors offering the same product/service in the same target location.
- Indirect competitors solving the same problem with a different product, service, method, or tool.
- Adjacent competitors or adjacent solutions solving the same audience pain point before or after the client's offer.
- Influencers or creators capturing the same audience's attention.
- Authority/KOL accounts competing for the same audience's trust.
- Local businesses, agencies, professionals, or pages repeatedly recommended by the community.
- Pages or profiles whose content is getting strong engagement from the client's target audience.

The agent should classify competitors into these levels:

#### Direct Competitor

A direct competitor sells a similar service to the same audience and location.

Examples:

- Another DUI lawyer in Los Angeles.
- Another real estate agent focused on Austin buyers.
- Another mortgage broker serving the same state.
- Another insurance agency selling similar coverage in the same market.

#### Adjacent Competitor

An adjacent competitor solves a related problem or captures demand before the client does.

Examples:

- A financial planner creating content about family protection when the client sells life insurance.
- A home inspector capturing first-time buyer attention before a real estate agent.
- A DIY automation consultant attracting the same small-business audience as an AI automation agency.

#### Audience Competitor

An audience competitor may not sell the same service, but consistently captures the attention, trust, comments, or questions of the same target audience.

Examples:

- A local Facebook page where homebuyers ask housing questions.
- A TikTok creator explaining traffic tickets in the same state.
- A community admin whose posts shape buyer or legal-service decisions.

#### Indirect Competitor

An indirect competitor solves the same problem in a different way.

Examples:

- A self-serve legal document service competing with a lawyer for simple legal needs.
- A comparison site competing with an insurance advisor for homeowner attention.
- A DIY automation tool competing with an automation agency.

#### Authority Or KOL Competitor

An authority or KOL competitor wins trust and shapes decisions even when they do not sell the same service.

Examples:

- A local creator whose posts influence homeowner, buyer, legal, financial, or business-service decisions.
- A niche newsletter or community admin whose recommendations drive the audience's next action.

For each detected competitor, include:

- Competitor type: direct | indirect | adjacent | audience | authority_or_kol.
- Source.
- Profile URL: the competitor's profile, page, company page, creator profile, business website, or account URL.
- Post/current URL: the exact post, thread, content page, recommendation thread, search result, or current browser URL where the competitor signal was captured.
- Platform.
- Location relevance.
- What they offer or appear to offer.
- Audience overlap.
- Content themes.
- Strong hooks or messaging patterns.
- Engagement signal, if visible.
- Why they matter.
- Threat level: high | medium | low.
- Opportunity: what the client can learn, counter-position, or improve.
- Copy-ready suggested comment in the same language as the post, written to add value without attacking the competitor or directly advertising the user's service.

Competitor detection must be used for learning and positioning, not copying.

The agent must not plagiarize competitor content. It may analyze patterns, gaps, objections, hooks, and positioning, then create original ideas for the client.

If the competitor is discovered from a community recommendation or indirect mention, still store both URLs when possible: the competitor profile URL if visible, and the post/current URL where the recommendation or signal appeared. If one URL is unavailable, write `unavailable` and explain the reason in notes.

Detected competitors should be stored in `history/YYYY-MM/competitor_log.md` and, when possible, `history/YYYY-MM/lead_competitor_opportunities.jsonl`.

### Adjacent Signal Reasoning Rule

The agent may infer useful content ideas from data points that are not directly about the client's product/service if there is a clear, explainable logic chain connecting the signal to the client's primary industry, related industries, audience, pain points, or business offer.

This is called adjacent signal reasoning.

Adjacent signals are allowed because real audience attention often comes from events, risks, questions, or situations that are not obviously part of the client's category at first glance.

However, adjacent signal reasoning must be transparent. The agent must show the logic chain, reference URL, and confidence level so the human can decide whether the inference is reasonable.

Required fields for any adjacent inference:

- Original data point.
- Reference URL.
- Why it looks unrelated at first.
- Logic chain from signal to client relevance.
- Related pain point.
- Related content pillar.
- Proposed content idea.
- Confidence: high | medium | low.
- Risk or compliance note, if relevant.

Example:

```md
Original data point:
A local news source reports severe flooding and storm damage.

Reference URL:
https://example.com/local-flooding-report

Why it looks unrelated:
The client sells life insurance, not disaster response or property insurance.

Logic chain:
Natural disaster -> higher accident and mortality risk -> families may wonder what happens if a policyholder dies during a disaster -> audience may need to understand how life insurance claims work in unexpected-death situations.

Related pain point:
"Will my family actually receive support if something happens suddenly?"

Related content pillar:
Protection clarity / family financial security.

Proposed idea:
"If someone dies during a natural disaster, how does life insurance usually handle the claim?"

Confidence:
medium

Compliance note:
Avoid implying that a specific policy will always pay. Explain that policy terms, exclusions, and documentation matter.
```

The agent must not present adjacent reasoning as fact. It must label it as an inference and show the logic clearly. If the logic chain is weak, speculative, fear-based, or compliance-risky, the agent should either discard the idea or mark it as low-confidence for human review.

### E. Generate A 3x2 Idea Matrix

Using A, B, C, D, `[content_pillars]`, and the agent's own reasoning, generate content ideas in three layers:

1. Hot / Trend / News
2. Evergreen / Foundation
3. Lead-Gen / Conversion

Each layer must include two scopes:

1. Global
2. Local

This creates a 3x2 matrix:

| Layer | Global | Local |
|---|---|---|
| Hot / Trend / News | Global trending/news ideas | Local trending/news ideas |
| Evergreen / Foundation | Global timeless education | Local timeless education |
| Lead-Gen / Conversion | Global conversion-focused ideas | Local conversion-focused ideas |

A slot may be empty on a given day if there is no credible idea.

The agent must not invent fake news. If there is no credible data for a slot, mark it as empty and explain why.

The idea list must respect the primary/related industry content mix:

- Target mix over time: approximately 80% primary-industry ideas and 20% related-industry ideas for each client.
- The agent does not need to force the exact ratio every single day, especially if the client receives only one script per day.
- The agent should evaluate the last 7-30 days in `history/YYYY-MM/content_log.md` and avoid drifting too far into related-industry content.
- Related-industry ideas are allowed only when the logic bridge is explicit and useful.
- If a related-industry idea is selected as the best idea of the day, the agent must explain why it is worth using today despite the 80/20 rule.

Every idea should map back to at least one content pillar when possible. If an idea is hot but does not map to a content pillar, the agent must explain why it is still worth considering or discard it.

When showing the idea list, each idea should include its mapped content pillar and industry scope (`primary_industry` or `related_industry`) so the human can understand which repeatable content line it belongs to.

Visible related-industry note rule:

- If an idea comes from a related industry, the agent must make that visible in the idea list itself, not only in hidden notes or metadata.
- The idea should include a clear label such as `[Related industry: P&C insurance]`, `[Related industry: mortgage]`, or `[Related industry: construction]`.
- The label should appear immediately next to the idea title or in the first detail line under the idea.
- The agent must include a short `Why this still fits` explanation for every related-industry idea.
- The explanation must show the bridge back to the primary industry and primary offer in plain language.
- This prevents the human from thinking the AI agent misunderstood the client's industry.

If an idea comes from adjacent signal reasoning, the agent must label it as `adjacent`, show the logic chain, include the reference URL, and let the human judge whether the connection is reasonable. The agent must not hide the inference step.

Examples:

#### Real Estate, Austin

Hot / Trend / News, Global:

- "What the latest Fed signal means for homebuyers this month."
- "Why national inventory changes do not mean every buyer has leverage."

Hot / Trend / News, Local:

- "Austin inventory is rising again. What buyers should do before making an offer."
- "A new zoning update could change where Austin buyers find future supply."

Evergreen / Foundation, Global:

- "The 3 numbers every first-time buyer must know before touring homes."
- "Why your monthly payment matters more than the listing price."

Evergreen / Foundation, Local:

- "How Austin property taxes change your real monthly payment."
- "What Austin buyers should know before choosing between old and new neighborhoods."

Lead-Gen / Conversion, Global:

- "Waiting for rates to drop might cost more than buyers think."
- "The buyer who is most prepared usually wins before the offer is written."

Lead-Gen / Conversion, Local:

- "Why Austin buyers should get pre-approved before watching prices, not after."
- "In Austin, the best deal is not always the cheapest house."

#### DUI Lawyer, Los Angeles

Hot / Trend / News, Global:

- "Why traffic enforcement spikes during holiday weekends."
- "What drivers misunderstand about refusing a sobriety test."

Hot / Trend / News, Local:

- "What LA drivers should know after a DUI stop this month."
- "How California license suspension rules can surprise first-time offenders."

Evergreen / Foundation, Global:

- "What to do in the first 24 hours after a DUI arrest."
- "The difference between a ticket, misdemeanor, and criminal charge."

Evergreen / Foundation, Local:

- "How a DUI can affect your California driver's license."
- "What happens after a DUI arrest in Los Angeles County."

Lead-Gen / Conversion, Global:

- "The biggest mistake people make after getting a ticket."
- "Do not assume a charge is minor just because you were allowed to go home."

Lead-Gen / Conversion, Local:

- "Why ignoring a California court notice can make your case worse."
- "The moment to call a lawyer is before deadlines start stacking up."

#### Insurance Agency, Miami

Hot / Trend / News, Global:

- "Why insurance premiums are rising in risk-heavy states."
- "What homeowners should review before storm season."

Hot / Trend / News, Local:

- "What Miami homeowners should check before hurricane season."
- "Why Florida property insurance changes matter for renewals."

Evergreen / Foundation, Global:

- "The difference between replacement cost and actual cash value."
- "What most people misunderstand about deductibles."

Evergreen / Foundation, Local:

- "How hurricane deductibles work in Florida."
- "What Miami condo owners should know about coverage gaps."

Lead-Gen / Conversion, Global:

- "The cheapest policy can become the most expensive mistake."
- "Insurance is not just about price. It is about what happens on the worst day."

Lead-Gen / Conversion, Local:

- "Why Miami homeowners should review coverage before the storm is named."
- "If your policy has not changed but your risk has, your coverage may be outdated."

#### AI Automation Agency, Vienna

Hot / Trend / News, Global:

- "A new AI tool changed how teams handle repetitive content tasks."
- "Why AI agents are moving from chat to workflow execution."

Hot / Trend / News, Local:

- "How Vienna service businesses can use AI without hiring more admin staff."
- "Why local agencies are using automation to handle client reporting."

Evergreen / Foundation, Global:

- "The 5 repetitive tasks every small business should automate first."
- "What an AI workflow is, explained without hype."

Evergreen / Foundation, Local:

- "How a local service business can start with one automation this week."
- "Why multilingual markets need careful AI workflow setup."

Lead-Gen / Conversion, Global:

- "AI does not replace your team. It removes the work nobody wants to do."
- "The best automation is not flashy. It is the one your team uses every day."

Lead-Gen / Conversion, Local:

- "Vienna businesses do not need a giant AI transformation. They need one useful workflow."
- "If your team copies the same data every week, that is your first automation project."

### F. Select The Best Idea Of The Day

After generating the idea list, the agent must choose the best idea for that day.

The best idea is defined by:

- Heat: Is this current, timely, or tied to a trend?
- Novelty: Has this client already covered it recently?
- Audience pain fit: Does it directly address `[pain_points]`?
- Business relevance: Does it connect logically to the client's product/service?
- Impact: Does it affect the audience in a meaningful way?
- Scale: Does it affect many people in the target audience?
- Local relevance: Does it matter in `[target_location]`, when location matters?
- Evidence: Is it supported by collected `[data_points]`?
- Lead potential: Could it drive trust, inquiries, appointments, consultations, or sales?
- Clarity: Can it become a clear short-form video script?
- Content mix fit: Does it help maintain the 80% primary industry / 20% related industries balance over the recent content history?
- Inference strength: If the idea comes from an adjacent signal, is the logic chain clear, credible, and useful without being misleading or fear-mongering?

The agent must check `history/YYYY-MM/content_log.md` before selecting.

The agent must explain why the chosen idea won.

If the chosen idea comes from a related industry or adjacent signal reasoning, the explanation must include:

- industry scope: `related_industry`,
- related industry name,
- logic chain back to the primary industry,
- confidence level,
- why this topic is appropriate within the 20% related-industry allowance.

Example selection reasoning:

`Selected idea: "Austin inventory is rising again. What buyers should do before making an offer."`

Why:

- It is timely because local market data changed this week.
- It connects to buyer anxiety about timing and negotiation.
- It is location-specific.
- It has broad relevance to first-time and move-up buyers.
- It has not been covered in the last 30 days.
- It can lead naturally to a CTA for a buyer consultation.

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

#### WideCast Setup Requirement

Before creating videos, the agent must check whether WideCast MCP, OpenAPI, API key, native WideCast tools, or WideCast integration is available in the current environment.

If WideCast is not installed or not connected, the agent must:

1. Visit `https://widecast.ai` by itself.
2. Learn the current WideCast setup and installation process.
3. Ask the human to register or log in only if required.
4. Ask the human to open `https://widecast.ai/#setup`.
5. Direct the human to the `API Keys & MCP` section.
6. Ask only for the required MCP URL, API key, or setup value needed by the current AI environment.
7. Complete MCP/OpenAPI/tool setup if the current AI environment allows it.
8. If the environment does not allow automatic setup, provide the exact minimal steps the human must do.

The agent must never ask for:

- WideCast password
- Email password
- OTP
- Browser cookies
- Raw session tokens
- Any credential not explicitly designed as an API key or MCP URL

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
2. One complete WideCast-writing-skill draft per active client: default video script, blog/article when configured, or both if requested.
3. Enough context to approve, revise, create the video, request a blog/video variant, or choose another idea.

---
