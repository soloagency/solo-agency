# Basic Profile And Public Report

Stage: `01`

## Load Rule

Load during first setup, add-client flow, setup repair, and Automation Flow first agency run/report generation. This stage must be loaded together with Stage 0 before the first setup question. In Setup Flow, the report-generation parts of this stage are superseded by the Setup Flow hard stop.

## Hard Gates For This Stage

- First question asks only for product/service, profession, expertise, business description, or a public website/profile URL.
- Do not ask for industry or sub-industry.
- Show inference before asking the next question.
- Explain any marketing/tech term in plain language when asking the human for input.
- Configure schedule/routine and the client-specific automation task once the basic source plan is known.
- Do not ask the private data source question before schedule/automation setup. Ask and resolve private data sources once after the task exists, then resync the task if source state changes.
- If private data sources exist and Local Collector is not installed/running/healthy, resolve or record the step 6 checkpoint after automation exists: guide Local Collector setup, or clearly mark private data sources as pending and resync the automation run contract.
- The first report must be produced by the client-specific automation task, not by the Setup Flow chat.
- Load Stage 10 before reporting leads, competitors, comment opportunities, or lead/competitor logs.
- PDNA setup - Production, Distribution, Notification, and Analytics - is a setup/config provider gate; do not start production or ask "make a video now?" inside Setup Flow.
- If the human asks to run, create, generate, show, refresh, or update a report while this stage is being used for Setup Flow, do not run the report. Finish or resync the client-specific automation task and tell the human the exact task name to run.

## Latest Override: Setup Flow Does Not Run Reports

This stage contains older first-run/report instructions for the previous workflow. The current Solo Agency control-plane model supersedes those instructions:

- In Setup Flow, do not run the first agency run, first report, public scan, private data source scan, draft generation, video creation, publishing, or PDNA actions.
- Setup Flow must finish by creating or updating the client-specific automation task and all persistent config needed for that task to run correctly.
- The first report must run in Automation Flow, using a client-specific task whose name begins with the client name, for example `AvenNgo - Solo Agency First Run` or `AvenNgo - Solo Agency Daily Run`.
- If the human asks to run, create, generate, show, refresh, or update a report while still in Setup Flow, verify/resync the automation task instead and tell the human the exact task name to run. Do not ask whether to run the report now, do not load the scheduled-run entrypoint in the setup chat, and do not perform public research, private data source collection, report generation, idea matrix updates, Lead & Competitor Opportunities, draft generation, analytics scans, or report notifications.
- Any later setup/config change in this session must update the Client Intelligence Profile, source state, collector config, extension registry, schedule, automation manifest, scheduled prompt, native task prompt when editable, and resync log.

Updated setup completion means `ready_for_automation_first_run`, not `first_report_completed`.

## Source Preservation Rule

This file is detailed source material moved from the original monolithic `SOLO_AGENCY_PLAYBOOK.md`.

Do not summarize away requirements, examples, checklists, schemas, protocols, URLs, edge cases, warnings, approval gates, or completion gates. If a downstream agent needs to shorten its response to the human, it may summarize the response, but it must still obey the full requirements in this file.

---

## 3. Minimal Human Input Rule

At setup, the agent must ask only for:

- Client name, if not already known.
- The client's product/service, profession, expertise, business description, or public website/profile URL.
- Target location only if location matters and cannot be inferred.

The agent must not ask for `output_formats` by default. If no output format is specified, default to `video_script`. If the human asks for blog, article, newsletter, SEO content, or long-form content, add `blog_article`. If the human asks for platform captions, add `social_caption`.

The agent must not ask the human to define:

- `industry`
- `sub_industry`
- `related_industries`
- `target_audience`
- `pain_points`
- `content_pillars`
- `public_data_sources`
- `private_data_sources`
- `idea categories`
- `content angles`
- `daily matrix`

The agent must infer these first.

Good first setup question:

`What product/service, profession, expertise, business description, or public website/profile URL should this pipeline focus on? If location matters, include the target location.`

Good add-client question:

`Please provide the new client's name and product/service, profession, expertise, business description, or public website/profile URL. Include target location if known.`

Do not mention private data sources in the first setup or add-client question. Ask about private data sources once, after schedule/routine and the client-specific automation task have been configured. That private data source checkpoint is where the agent asks for actual private data source URLs/lists, offers discovery, gets approval, handles Local Collector activation when needed, and resyncs the automation task afterward.

Bad setup questions:

- "What industry are you in?"
- "What sub-industry should I use?"
- "Please list your target audience."
- "Please list all pain points."
- "Please define your content pillars."
- "Please provide all public data sources."

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

When showing initial public search keywords, do not show only broad industry terms and do not dump the full keyword bank into chat. The agent must generate and save a broad `public_search_keywords` bank, then show only a compact `Pain-Point Keyword Sample` with 5-12 keywords from pain-point/problem/need groups plus a line such as `+200 more saved in the keyword bank for daily rotation`.

Keyword language must follow the target audience's likely search/comment language, not automatically the human's chat language. The human-facing explanation may be in the human's language while the actual keyword strings remain in the audience language. If the audience is multilingual, create and label multilingual keyword variants.

Required setup sequence, aligned to the visible 9-item roadmap. Do not introduce setup steps 10+.

1. Ask for the client's product/service, profession, expertise, business description, or public website/profile URL.
2. Infer and show `industry`, `sub_industry`, `related_industries`, `business_offer`, likely `target_audience`, language assumptions, and whether the business is location-dependent. If the target location is required and cannot be inferred, ask only for `target_location`.
3. Infer and show `pain_points`, `content_pillars`, how each pillar maps to pain points and the business offer, which pillars are `primary_industry` vs `related_industry`, and the planned content mix rule.
4. Select public data sources and build the keyword bank. Show only a compact `Pain-Point Keyword Sample`, not a generic industry keyword dump. Then show the complete setup summary, ask the human to correct only what is wrong, and save the Client Intelligence Profile after that correction chance.
5. Configure the schedule/routine before the first agency run, using the best scheduling mechanism available in the environment, and create or verify the client-specific automation task. Configure the initial task as public data sources only unless active private data sources are already verified healthy.
   - The setup handoff must include the client-specific automation task name; task status; whether the task currently runs public data sources only or public plus activated private data sources; a visible `Solo Agency one-time setup process` progress roadmap; and the exact next action.
6. Ask and resolve private data sources once, after the automation task exists. Explain plainly that private data sources mean logged-in/social/community places the human may want monitored later, such as competitor profiles, fanpages, communities, LinkedIn pages, Reddit communities/subreddits, niche forums, and Facebook groups. Load `playbooks/PRIVATE_SOURCE_GATE.md`, Stage 2, Stage 8, and Stage 9 when private data sources are requested, already provided, or discovery/Local Collector is relevant (print a LOAD LEDGER per `playbooks/LOAD_LEDGER_PROTOCOL.md` for each file loaded).
   - If the human already provided private data sources in an earlier message, record them as `pending_private_review` and process them in this step only.
   - Ask for actual private data source URLs/lists or offer one optional private data source discovery pass from approved joined/followed/member spaces or Facebook keyword group search.
   - Get human approval before adding any discovered source; then guide Local Collector setup or record the exact pending blocker.
   - After the human approves, declines, postpones, or blocks private data sources, update source state and perform Automation Resync so the already-created task has the newest contract.
   - Do not ask a separate private data source discovery checklist question. Discovery stays inside step 6.
   - Do not label the collector by platform. Even if the provided sources are all Facebook, call it the Solo Agency Local Collector extension and Local Collector app.
   - If the human approves Facebook member-groups discovery, scan the joined-groups discovery page through the Solo Agency Local Collector: `https://www.facebook.com/groups/joins/?nav_source=tab&ordering=viewer_added`.
   - If the human approves Facebook keyword group search discovery, infer client-relevant keywords, scan `https://www.facebook.com/search/groups/?q={url_encoded_keyword}` through the Solo Agency Local Collector, scroll 10 times per keyword, filter out UI noise/non-group results, and ask for approval before adding any recommended group.
   - If the human approves other platform discovery categories, use the private data source discovery platform starting URL registry and mark each approved category as `pending_private_activation` until the Local Collector is active and healthy.
   - After source intake or discovery, infer and show which private data sources are likely useful, which should be skipped or treated as optional, how they map to content pillars, and whether each source should be `daily`, `weekly`, or `optional` based on relevance and safe monitoring volume.
7. If the human asks for production/video/blog/social, publishing, notifications, analytics, or "full automatic", load Stage 3 (print a LOAD LEDGER per `playbooks/LOAD_LEDGER_PROTOCOL.md` for each file loaded) and complete the PDNA provider/capability setup gate only. Do not end the setup handoff with `Do you want me to make a video now?`, start scene editing/rendering, or create/publish assets from Setup Flow.
8. Record analytics as an Automation Flow concern. In Setup Flow, only save whether published URL history exists and whether future scheduled runs should load Stage 5.
9. Record that reports, idea matrix updates, Lead & Competitor Opportunities, draft generation, analytics scans, and PDNA production actions belong to Automation Flow. After any private scan or approved source-discovery scan in Automation Flow, analyze the collected private data and update the report there.

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
Did I read the pain points right? Please correct anything that is wrong in the list above, and confirm the target location: should I focus on Los Angeles only, or also cover nearby counties where your clients get stopped or go to court? The answer changes which local sources and keywords I select next.
```

Note: private data sources are asked only at setup step 6, after the schedule/routine and the client-specific automation task exist; do not ask about them in this follow-up.

---

## 4. Inference-First Rule

The agent must think, infer, and research before asking.

The agent must:

- Use existing files first.
- Use the client description or public website/profile URL.
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
- Requested private data source categories
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

1. Smith Law - DUI lawyer - Los Angeles - private data sources: competitor FB pages A, B
2. Austin Home Group - real estate agent - Austin, TX - private data sources: none yet
3. Bright Mortgage - home loans - Texas - private data sources: competitor TikTok X
4. Miami Shield Insurance - home and auto insurance - Miami - private data sources: local FB group Y
5. Vienna AI Ops - AI automation agency - Vienna - private data sources: LinkedIn competitors
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
- Product/service, profession, expertise, business description, or public website/profile URL.
- Target location only if location matters and cannot be inferred.
- If the human volunteers private data sources early, record them as `pending_private_review` and resolve them only at setup step 6 (do not ask for them here).

Do not create fake client pipelines. If the client name or enough business context is missing, ask for the missing information and keep the root pipeline ready. A public website/profile URL counts as business context if it is publicly accessible and gives enough information to infer the setup.

Whenever the human says something like:

- "Add a new client"
- "Add this client to the pipeline"
- "We just got a new client"
- "Start monitoring content ideas for this business"
- "Add client: ..."

The agent must enter Add Client Mode.

In Add Client Mode, ask only for missing critical information:

- Client name.
- Product/service, profession, expertise, business description, or public website/profile URL.
- Target location only if location matters and cannot be inferred.
- If the human volunteers private data sources early, record them as `pending_private_review` and resolve them only at setup step 6 (do not ask for them here).

The agent must infer:

- `industry`
- `sub_industry`
- `target_audience`
- `pain_points`
- `content_pillars`
- `business_offer`
- `public_data_sources`
- `brand_voice`
- `language`, including `human_report_language`, `target_audience_language`, `keyword_language`, and `content_output_language`
- `platforms`
- `compliance_notes`
- `negative_topics`

Then the agent must follow the same 9-item setup model. Do not introduce Add Client setup steps 10+.

1. Show the inferred setup summary to the human and ask them to correct only what is wrong.
2. Create or update the client pipeline folder, Client Intelligence Profile, history folder, outputs folder, and `clients_index.md` row.
3. Save the inferred pain points, content pillars, business offer, language assumptions, and compliance notes.
4. Save the public data source plan and keyword bank.
5. Configure the recurring schedule/routine once the basic source plan is known, prepare or verify the client-specific extension folder under `extensions/{client_slug}/`, and create/resync the client-specific automation task with the task name beginning with the client name, for example `Nguyen Law - Solo Agency First Run` or `Nguyen Law - Solo Agency Daily Run`. Configure the initial task as public data sources only unless active private data sources are already verified.
6. Ask and resolve private data sources once after automation exists: record provided sources as `pending_private_review`, declined/no sources, pending Local Collector activation, private data source discovery approved/pending, or `discovery_declined_or_postponed`. If private data sources exist or discovery is approved/pending, guide Local Collector setup now, or record private data sources as `pending_private_activation` so the first automation run can continue with public data sources only if needed. Update the Client Intelligence Profile, `schedule.md`, collector config, extension registry, automation manifest, scheduled-run prompt/task body, and resync log so the already-created task has the newest source state. Finish the setup handoff by telling the human the exact automation task name to run for the first report, the absolute extension folder path, the matching Chrome profile/account to use when private data sources are active/pending, exact `chrome://extensions` -> Developer mode -> `Load unpacked` steps when needed, and whether private data sources are active, pending, stale, declined, or not requested. Do not merely say "extension created."
7. If the human asks for production/video/blog/social, publishing, notifications, analytics, or "full automatic" during Setup Flow, load `playbooks/03_PRODUCTION_DISTRIBUTION.md` only as a provider/configuration gate. Do not create or publish assets from Setup Flow.
8. Record analytics as an Automation Flow concern only.
9. Do not run the first agency run, first report, public scan, private data source scan, report updates, idea matrix updates, Lead & Competitor Opportunities, draft generation, analytics scans, video creation, publishing, or PDNA production actions inside Setup Flow.

Example:

Human:

```md
Add this client to the daily content pipeline: Nguyen Law, immigration lawyer in San Jose. Private data sources to monitor: [links].
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

The agent must configure the routine, prepare the `Nguyen Law - ...` automation task, then resolve/resync step 6 if private data sources are pending, and tell the human to run that client-specific task for the first report. Setup Flow must not run Nguyen Law's report directly.

---

## 14. Mandatory Automation Readiness Protocol

This protocol applies after the first client setup, after adding a new client, and after repairing an incomplete Client Intelligence Profile.

The setup flow is not a menu of optional next steps. The agent must not ask the human to choose between:

- providing private data sources,
- configuring the schedule,
- running the first agency run/report,
- creating a video.

The one allowed private data source choice is operational, not a new menu: after the automation task exists, if private data sources exist and Local Collector is pending, ask whether to activate Local Collector now so future automation can include data from private data sources, or mark private data sources pending so the automation task can run public data sources only until the blocker is resolved.

The correct order is fixed:

1. Finish setup context and save the Client Intelligence Profile.
2. Configure the schedule/routine once the basic source plan is known.
3. Prepare or verify the client-specific extension folder under `extensions/{client_slug}/`.
4. Create or resync the client-specific automation task. The task name must begin with the client name.
5. Ask and resolve the step 6 private data source intake/discovery/approval and Local Collector activation when applicable.
6. Update every persistent state file that the next automation run reads: Client Intelligence Profile, source state, collector config, extension registry, schedule, automation manifest, scheduled prompt/task body, and resync log.
7. If the human wants PDNA setup - Production, Distribution, Notification, and Analytics - complete only the provider/configuration gate in Setup Flow.
8. Record analytics as an Automation Flow concern only.
9. End Setup Flow with `ready_for_automation_first_run`, not `first_report_completed`; reports, drafts, analytics, production, publishing, and private data source scans run only in Automation Flow.

First automation report rule:

- The first report happens in the client-specific automation task, after routine setup, task creation, the step 6 private data source checkpoint, and human action to run or schedule that task.
- Setup Flow must not create `/jobs/run_now`, must not scan public data sources, and must not scan private data sources merely to produce a report.
- The first automation task should use public data sources, public search, client context, inferred pain points, inferred content pillars, related industries, and any previously collected local data.
- If private data sources were provided but are not active, the automation report must include a section called `Private Data Sources Pending Activation`.
- If private data source discovery was approved but not yet run, the automation report must include `Private Data Source Discovery Pending Activation`.
- That section must list the private data source URLs or discovery categories, explain that they were not scanned yet, and say that activation requires the Solo Agency Local Collector extension plus Local Collector app.
- The automation report must include at least one draft script/blog/caption or a clear report section containing the draft.
- The automation report must ask a clear next-step question after delivering the useful output. Unless PDNA setup - Production, Distribution, Notification, and Analytics - was already completed or explicitly declined, the next-step question should be:

```md
Do you want me to set up PDNA - Production (create real video/blog/social assets), Distribution (publish approved content), Notification (send reports/blockers), and Analytics (measure results) - so approved drafts can become real assets and the system can learn from performance later?
```

The automation agent must ask this question directly in the chat message or notification where it announces the first report result. It must not hide the question or setup steps inside a Markdown file.

The same chat message must show the updated `Solo Agency automation process` progress roadmap or a compact progress summary. It must show that first report generation is complete and that production/provider setup, private data source activation, published-URL analytics status, and report/recommendation update status are still pending, completed, declined, blocked, or not applicable. After any schedule/routine or client-specific automation task exists, every human-facing progress block must include the `Automation freshness check` line required by the root playbook.

If PDNA setup - Production, Distribution, Notification, and Analytics - is already completed, declined, or blocked and private data source activation is pending, the final line must be:

```md
Private data sources (logged-in/social/community places such as groups, profiles, pages, channels, or forums) are not activated yet because they require the Local Collector app and Chrome extension on your computer. Do you want me to prepare the setup files and then give you the two required local steps: run one Terminal/PowerShell command yourself and load the Chrome extension from the folder I show you?
```

If there are no private data sources and discovery was declined or not requested, the final line must ask the next required decision, usually:

```md
Do you want me to set up PDNA - Production (create real video/blog/social assets), Distribution (publish approved content), Notification (send reports/blockers), and Analytics (measure results) - so approved drafts can become real assets and the system can learn from performance later?
```

or, if production was already declined:

```md
Do you want me to keep the saved routine as-is for tomorrow's automatic run?
```

Do not end the report handoff with only a report link, a summary, or "let me know."

Good first automation report chat pattern:

```md
The first automation report is ready.

Best idea today: {best idea}
Report for mobile: {absolute HTML path or URL}
First draft: {script/blog/caption title}

Solo Agency automation process
This is the planned automation process for this client. You only need to reply when I ask one specific question.

✓ 1. You provided the product/service, profession, expertise, business description, or public website/profile URL
✓ 2. I inferred the industry, sub-industry, related industries, audience, and offer
✓ 3. I inferred pain points and content pillars
✓ 4. I selected public data sources and search keywords
✓ 5. I configured the automatic schedule/routine and client-specific automation task
– 6. Private data sources/Local Collector are pending or postponed; this automation run uses public data sources only
→ 7. I help set up PDNA: Production, Distribution, Notification, and Analytics
– 8. From the second run onward, if PDNA is set up, I scan analytics for published URLs from the last 7 days
✓ 9. I created the HTML report, idea matrix, Lead & Competitor Opportunities, competitor signals, and first script/blog/caption draft in Automation Flow

Automation freshness check: ✓ Have the latest approved changes been synced into the automation/scheduled task prompt, contract, playbooks, source approvals, and state? If tomorrow's scheduled run starts, will it load the newest state: current.

The operator-only `INTERNAL_REPORT` includes the PDNA/WideCast status and setup note. The client-facing report stays clean and does not mention Solo Agency, WideCast, provider tooling, Local Collector, automation, API keys, Telegram, or internal system details.

This run used public data sources only. I have {N} private data sources waiting, including:
- {source name or URL}
- {source name or URL}

Do you want me to set up PDNA - Production (create real video/blog/social assets), Distribution (publish approved content), Notification (send reports/blockers), and Analytics (measure results) - so approved drafts can become real assets and the system can learn from performance later?
```

Bad first automation report chat pattern:

```md
Private data sources were not scanned. Instructions are in collector/collector_setup_status.md.
Now choose a schedule.
```

Also bad:

```md
The first automation report is ready.
Report: {html path}
Let me know what you want to do next.
```

This is bad because it does not show progress and does not end with a concrete next-step question.

Private data source activation rule:

- If the human agrees to activate private data sources, collector setup becomes mandatory at that point.
- The agent should proceed automatically as far as file preparation allows, but it must not run the one-time Local Collector setup/start command itself.
- During one-time setup/update/repair, the agent must never execute `setup_collector.sh`, `setup_local_collector.ps1`, `Start Local Collector.cmd`, or the collector binary from inside the AI agent, even if shell permissions are available. Agent-run setup can be trapped in a sandbox/session and killed after the turn.
- The agent must create the script/launcher file first and give the human exactly one short Terminal/PowerShell command or one double-clickable file path to run outside the AI sandbox, not a long multi-line script.
- The same setup handoff must include the Chrome extension install steps and the one absolute client-specific extension folder path under `extensions/{client_slug}/` for every new client, even when private data sources are not active yet.
- The handoff must say which Chrome profile/account to open for that client and must show `chrome://extensions` -> Developer mode -> `Load unpacked` -> select the absolute `extensions/{client_slug}/` folder.
- Saying only "I created the extension" or "extension folder exists" is incomplete.
- The exact human action must be shown directly in chat. The agent may also save it in `collector_setup_status.md`, but the saved file is only the agent's record and must not be the only place where the human receives the instruction.
- The agent must not label the collector by the current platform, such as `Facebook collector`.
- The agent must create or update `daily-content-pipeline/collector/collector_setup_status.md` when private data source activation begins.
- If the AI environment can write local files, the agent should download/update/extract the collector artifacts and create/update the setup script/launcher, but it must not run that setup script or start/restart the Local Collector app from inside the AI sandbox during one-time setup.
- After the human confirms they ran the setup/start command and loaded the Chrome extension, the agent may check `GET http://127.0.0.1:17321/status`.
- If the Solo Agency Local Collector extension is not loaded, the agent must show the absolute extension folder path and the exact Chrome `Load unpacked` steps.
- The extension path shown to the human must be the runtime workspace path under `extensions/{client_slug}/`, not any toolkit/source path under `solo-agency/solo-agency-collector/chrome-extension/`.
- After collector setup succeeds in Setup Flow, the agent should update the automation task and mark private data sources as ready for the next Automation Flow run. Do not run a private data source activation scan inside Setup Flow.
- The agent must not claim private data source monitoring is active until collector health confirms the Local Collector app and Solo Agency Local Collector extension are working.
- The agent must not configure a recurring schedule that promises private data source collection until collector setup is either `installed_and_running` or explicitly documented as pending/blocked with a human action.

Automation run-now rule:

- Setup Flow must not create `/jobs/run_now` or run an equivalent manual report path. It must prepare or resync the client-specific automation task instead.
- In Automation Flow, a human request such as `run now`, `manual run`, `test run`, `trial run`, `collect now`, or `scan now` should bypass recurring schedule windows for the target client only.
- If the Local Collector app is already installed, running, healthy, and matched to the target client's extension identity, the Automation Flow agent may include private data sources by creating a run-now job.
- If the Local Collector app is not installed/running/healthy or the matching extension is stale, the Automation Flow agent should run public data sources and list private data sources as pending activation.
- The automation report output must include a mobile-friendly HTML report, a concise summary, and at least one useful draft script/blog/caption.
- If the client's WideCast/OpenAPI provider config is not connected and verified, the operator-only `INTERNAL_REPORT` and operator handoff must include the PDNA/WideCast setup note so the human sees how the useful report can become video/blog production, 10+ platform distribution, Telegram notifications, performance measurement, and a learning loop after one WideCast setup. Client-facing reports must not include this note.

Manual run / run-now rule:

- Any human request such as `run now`, `manual run`, `test run`, `trial run`, `collect now`, or `scan now` must bypass recurring schedule windows.
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
- If the Local Collector app is not reachable, the agent must not try to start it from inside the AI sandbox during setup/repair. Provide the one-line Local Collector app setup/start command for the human to run outside the sandbox, then retry the run-now job only after the app is reachable.
- Recurring schedule windows are only for unattended scheduled runs. They must not block manual runs.
- Do not simulate a manual run by editing `scheduled_windows` or creating a temporary schedule window. Manual runs must use `/jobs/run_now`.
- If the agent cannot call `http://127.0.0.1:17321` from its own sandbox but can write local files, it must write one unique per-client job file under `daily-content-pipeline/collector/jobs/pending/`. The Local Collector app claims matching pending jobs on `/status`, moves claimed files into `jobs/claimed/`, writes output for that client, then moves completed files into `jobs/completed/`.
- `daily-content-pipeline/collector/run_now_request.json` is a legacy/batch shim only. It may contain one job or `{"jobs":[...]}` and the bridge converts it into `jobs/pending/` queue files. Do not use this single filename when multiple agents or scheduled tasks may write concurrently.
- If the agent cannot call HTTP and cannot write the local queue file, only then create a local run-now helper script or launcher and give the human exactly one short command/path to run it. The helper script must POST `/jobs/run_now` with the correct payload, then optionally poll `/status`.
- Do not ask the human to restart the Local Collector app merely to make a manually edited schedule file take effect. Restarting is only appropriate for updating the Local Collector app itself, recovering a stuck/offline process, or applying an intentional recurring schedule change when both `/config` and file auto-reload are unavailable.
- If a legacy collector without `/jobs/run_now` forces a temporary schedule fallback, the agent must clearly label it as a fallback, back up the original config, create a short unique temporary window, restart or reload only through an already-running service or a human-run setup/start command when required, restore the original config immediately after completion/timeout, and report that fallback to the human. This fallback must not be used when `/jobs/run_now` exists.

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
    "do_not_exfiltrate_secrets": true
  }
}
```

- `run_id` must be unique for every manual run. A recommended pattern is `YYYY-MM-DD_client-slug_manual_HHMMSS`.
- `run_now` must be `true`.
- `force` must be `false` unless the human explicitly asks for a troubleshooting rerun and understands the same `run_id` may run again.
- `run_now_ttl_minutes` should be 30 by default and must not exceed 120.
- `sources` must contain the private data sources for that client if private data sources exist. If there are no private data sources, the agent should still run public research without the Local Collector app.
- `pacing.scroll_steps` defaults to 5 and must not exceed 10.
- If the agent cannot make this POST itself but can write local files, it should write the JSON payload to one unique per-client queue file:

```text
daily-content-pipeline/collector/jobs/pending/{timestamp}_{client_slug}_{run_id}.json
```

The agent should write this file atomically: write a temporary file in the same folder first, then rename it into `jobs/pending/` only after the JSON is complete.

The running Local Collector app should pick up queued files on the next `/status` check from the matching Chrome extension, usually within a few seconds while Chrome is active. The bridge must:

- write `run_now_request_status.json`;
- move claimed files into `jobs/claimed/`;
- move completed files into `jobs/completed/`;
- clear the active run-now job on `/complete`;
- expire the active run-now job after `run_now_ttl_minutes` if `/complete` never arrives, then allow the next queued client job to proceed.

The single file `daily-content-pipeline/collector/run_now_request.json` remains supported only as a legacy/batch shim. It is safe for one agent to write a batch object with `{"jobs":[...]}`; it is not safe for multiple writers to race on the same filename.

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

- After posting `/jobs/run_now` or writing a queue file, do not fake extension headers. Plain `GET /status` is only a bridge health check; `job_available` is scoped to the real client extension identity.
- Track progress through `run_now_request_status.json`, `bridge_health.json`, `collector_status.json`, `jobs/claimed/`, `jobs/completed/`, and new output files under the run output directory until the job completes or TTL expires.

Schedule rule:

- Ask schedule/routine questions after the profile and source plan are known and before the client-specific automation task is marked ready.
- Ask whether the human wants daily, multiple-times-daily, weekly, manual-only, first-run-only, or another cadence.
- Then write or update `schedule.md`, the automation manifest, the scheduled-run prompt/task body, and the relevant automation/config files.
- During Setup Flow, do not ask to run the first agency run immediately and do not run a report. Finish by preparing or resyncing the client-specific automation task whose task name begins with the client name.
- After schedule/routine setup and automation task creation, ask/resolve private data sources once. If private data sources exist and Local Collector is pending, handle step 6: guide Local Collector setup or clearly mark private data sources as `pending_private_activation` in the automation contract so the first automation run can continue with public data sources only if needed, then resync the task.

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
- If the human has not activated private data source monitoring yet, configure the recurring schedule as public data sources only and clearly mark private data sources as `pending_private_activation`.
- Only configure scheduled private data source collection after Local Collector activation is accepted and collector health is confirmed or explicitly documented as pending/blocker.
- The Local Collector app must run in persistent mode for unattended scheduled collection:

```text
solo-agency-local-collector/bin/collector-bridge-darwin-arm64 \
  --host 127.0.0.1 \
  --port 17321 \
  --config-file daily-content-pipeline/collector/collector_config.json \
  --output-dir daily-content-pipeline/collector/inbox \
  --persistent
```

- The Solo Agency Local Collector extension polls `/status`; when the current local time is inside an enabled `scheduled_windows` item and private data sources exist, `/status` should expose a scheduled job with `current_job_type: scheduled` and `job_available: true`.
- Scheduled run IDs are generated by the Local Collector app, usually using `YYYY-MM-DD_schedule-name`.
- The agent must still write a human-readable `schedule.md` explaining the cadence, clients included, private data source limits, and notification behavior.

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
   5. Load saved `public_data_sources` and visit/check active due public data sources before or alongside keyword search.
      - Visit sources where `visit_in_scheduled_runs: true` and cadence is due today.
      - Prioritize `active_public_source` daily sources, then due `weekly_public_source` sources, then relevant `occasional_public_source` sources when the topic/event matches.
      - Record source status, useful URLs, useful signals, weak/noisy results, and whether the source should stay active, be promoted, or be demoted.
   6. Use Google Search or an available equivalent search tool with rotating keywords from `public_search_keywords`.
      - Do not use only generic industry keywords.
      - Prioritize pain-point/problem/need/buying-intent keyword clusters because these are closer to real audience demand.
      - Use keywords in the target audience's likely search/comment language. Do not translate the keyword bank into the human's chat/report language unless the audience uses that language.
      - Use at least 10 distinct public search keywords per public data source run unless search tooling is unavailable or the saved keyword bank has fewer than 10 usable entries after expansion.
      - At least 7 of the 10 keywords should come from pain-point/problem/need/buying-intent/objection/comparison/question/local-context/trend-news groups.
      - Include at least one broad primary-industry keyword for context, at least one pain-point/problem keyword, at least one need/goal or buying-intent keyword, and local/location keywords when location matters.
      - Use a smaller rotation of related-industry keywords only when the bridge back to the client's offer is clear.
      - If results are weak, try a different pain-point/problem/need cluster before giving up.
      - Continue keyword rotation until at least 3 source-backed candidate ideas are new or newly angled against `history/YYYY-MM/content_log.md`. If fewer than 3 qualify after 10+ distinct keywords and due public data sources have been checked, record the coverage limitation instead of fabricating weak ideas.
      - Record every keyword used, keyword group, result quality, useful URLs, and final keyword status.
      - Extract new keyword candidates from useful search results, public discussions, questions, competitor hooks, comments, and emerging phrases. Add useful new candidates to the keyword bank with source/reason, related pain point, and content pillar.
      - Detect useful recurring public data sources from search results and public pages. Promote strong recurring sources into `public_data_sources` with status/cadence so future scheduled runs can visit them automatically.
      - Include this record in the daily report section `Public Search Keywords Used Today`.
      - If no search was possible, explicitly explain the blocker in that same section.
   7. Before deciding whether to skip private data sources, follow Stage 4 and Stage 8 Collector Runtime Verification. Do not trust saved labels such as `pending_private_activation`, `public_data_sources_only`, or `private sources postponed` alone. If `/status` is unreachable from the AI sandbox, read local collector health/status files before claiming the Local Collector is inactive.
   8. If private data sources remain unavailable after Collector Runtime Verification, mark the exact blocker, include the activation/repair CTA in the report, and continue with public data sources and previously collected private data when available.
   9. If private data sources are available, connect to the already-running Local Collector app according to `collector_config.run_mode` or use the Stage 8 file-based run-now path when localhost is isolated but local health files prove a recent current-workspace collector.
   10. Check private collector health according to Stage 4/Stage 8 before collection and before claiming private data sources were skipped.
   11. Prepare the private data source queue if private data sources are available and collector health is acceptable:
      - keep the active daily queue around 20 sources or fewer per client by default;
      - prioritize sources most relevant to the client, target audience, target location, pain points, and content pillars;
      - classify extra sources as `weekly` or `optional` and rotate them across future runs;
      - do not run aggressive or parallel private data source scans for the same logged-in account.
   12. Check private data sources if available, using the Solo Agency Local Collector extension plus the Local Collector app when available, with `collector_config.scroll_delay_seconds` defaulting to 5 seconds and `collector_config.max_scrolls_per_source` defaulting to 5.
      - After private collection reaches a terminal state, reconcile status and counts before report handoff: private scan status, completed timestamp, sources attempted/completed/blocked, data points kept, leads, competitors, recommended private data sources, noisy/skipped discovery candidates, notifications, and blockers must match across the private report, daily index, internal source record, report state JSON, and `outputs/latest/` copies.
      - Do not leave stale `scan in progress`, `partial`, `pending`, or old recommended-source totals in one artifact after another artifact says the private scan is complete.
   13. If the collector bridge was started in `agent_on_demand` mode, stop it after collection completes or after timeout.
   14. Log skipped, pending-activation, expired, rate-limited, warning-triggered, collector-unavailable, extension-unavailable, Chrome-not-running, stale-extension, bridge-offline, or unavailable private data sources.
   15. Load yesterday's private data for this client when available and filter duplicate or near-duplicate data points using visible text matching. Do not parse private-platform HTML for duplicate detection.
   16. Extract relevant `[data_points]`, including reference URLs for every data point. Keep data points that are directly about the primary industry or clearly connected through a related industry. Discard related-industry data when the bridge back to the client's offer is weak.
   17. Add newly recommended private groups/pages/profiles/communities to `New Private Data Sources Detected` and `history/YYYY-MM/new_private_sources_log.md`.
   17b. Load `playbooks/10_LEAD_COMPETITOR_DETECTION.md` (Stage 10, print a LOAD LEDGER per `playbooks/LOAD_LEDGER_PROTOCOL.md`) before detecting, scoring, or reporting any leads or competitors.
   18. Detect hot and warm leads, including profile URLs, post/current URLs, safe summaries, and reasoning.
   19. Detect direct, adjacent, and audience competitors, including profile URLs, post/current URLs, and positioning notes.
   20. Generate the 3x2 idea matrix as six buckets, not six total ideas. Put every credible, source-backed idea from today's data into the matching layer/scope bucket, and label each idea as `primary_industry` or `related_industry`.
      - Every idea must pass the Audience Value-First Gate: it must state the target audience pain point, the viewer value/lesson, the source signal, the non-promotional angle, why it helps the audience, and only then the soft business relevance.
      - Do not make the client's product/service name the premise of the idea. If an idea only says the client/product wins, out-positions, is better, or should be chosen, rewrite it into an educational viewer lesson or reject it as `promotional_not_value_first`.
   21. Check `history/YYYY-MM/content_log.md`, including the recent primary/related ratio and duplicate/near-duplicate idea risk.
   22. Perform the Idea Novelty Check: prefer at least 3 candidate ideas that are new or newly angled. If a prior topic is reused, record the prior idea/date, today's new angle, and why the re-angle is materially different.
   23. Select the best idea of the day only from ideas that pass the Audience Value-First Gate.
   24. Write the configured production-ready draft using Client tools/OpenAPI first, global MCP/native tools only after identity match, or the account-free writing skill fallback when provider/account access is unavailable. Drafts must preserve the same viewer-value lesson and must not become direct ads for the client's product/service. Keep writing-method/provider details in `INTERNAL_REPORT`, not client-facing files.
   25. Save `outputs/YYYY-MM/YYYY-MM-DD/{client-name}-daily-report.md` as the internal source-of-truth report.
   25b. Before writing any client-facing report HTML, load `playbooks/06_AGENCY_REPORT_STANDARD.md` (Stage 6) and then `playbooks/skills/report-design/SKILL.md` (LOAD LEDGER for each). Render/package with `tools/solo_report_renderer.py` (`render` for staging HTML, `package` for the combined client report + PDF) rather than hand-writing one-off HTML/PDF scripts.
   26. Generate the three-file scrubbed staging HTML report set under `outputs/YYYY-MM/YYYY-MM-DD/` with `tools/solo_report_renderer.py render`: `{client-name}-public-data-sources-report.html`, `{client-name}-private-data-sources-report.html`, and `{client-name}-daily-report.html`.
   27. Generate or update the operator-only `{client-name}-INTERNAL_REPORT.html`, clearly labeled `INTERNAL_REPORT - Not for client sharing`, and put Solo Agency/WideCast/provider/Telegram/social-platform/API-key/config/Local Collector/automation/blocker/debug details there.
   28. Run the Client-Blind Scrub Gate on the client-facing HTML files. They must not mention Solo Agency, WideCast, PDNA/provider tooling, OpenAPI, MCP, Local Collector, Chrome extension, automation/scheduled task, API key/config, Telegram, agent/tool/debug details, or `INTERNAL_REPORT`.
   29. Generate or update `{client-name}-client-report.html`, `{client-name}-client-report.pdf`, and `outputs/latest/{client-name}-client-report.pdf` from the scrubbed three HTML files with `tools/solo_report_renderer.py package`, or record the exact PDF blocker/status.
   29b. Write or update `outputs/YYYY-MM/YYYY-MM-DD/{client-name}-report_state.json` and reconcile it with the staging HTML files, combined client report, PDF companion, and INTERNAL_REPORT (counts, per-lane statuses, timestamps).
   30. Update or copy `outputs/latest/{client-name}-daily-report.html`.
   31. Update or copy `outputs/latest/{client-name}-INTERNAL_REPORT.html`.
   32. Update or copy the latest public/private lane HTML files when those lane reports exist.
   33. Update `history/YYYY-MM/content_log.md`.
   34. Update `history/YYYY-MM/data_sources_log.md`.
   35. Update `history/YYYY-MM/lead_log.md`.
   36. Update `history/YYYY-MM/competitor_log.md`.
4. Create or update `outputs/YYYY-MM/YYYY-MM-DD_master_digest.md`.
5. Generate `outputs/YYYY-MM/YYYY-MM-DD_master_digest.html` as a polished standalone human-facing master report.
6. Update or copy `outputs/latest_master_digest.md`.
7. Update or copy `outputs/latest_master_digest.html`.
8. Present the daily digest to the human.
9. If the configured provider notification capability is available, preferably WideCast OpenAPI `sendNotification`, send a notification to the human/operator that includes run status, client-facing HTML report path/link, PDF companion path/status, INTERNAL_REPORT path/status, clients processed, blockers, lead/competitor counts, and required actions.
10. If another authorized channel can send the HTML/PDF files or links more conveniently, use it.
11. Log the notification attempt in `notifications/notification_log.md`.

The daily run is complete only when every active client is processed or explicitly logged as skipped.

When presenting the daily idea list to the human, include reference URLs next to data points, top ideas, and the selected best idea so the human can verify the information. For private data, include the captured source URL and note that it may require the human's logged-in session.

Scheduled runs must assume the human may not be present in the AI agent UI. The run is not fully operationally complete until the scrubbed mobile-friendly client-facing HTML result plus PDF companion path/status plus INTERNAL_REPORT path/status, or a result-ready notification with those paths/statuses, has been sent through the configured notification channel, preferably WideCast OpenAPI Telegram/email fallback when configured for that client.

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
