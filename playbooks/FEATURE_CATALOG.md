# Feature Catalog — the Revenue Engine jobs the Boss can ask Sam to do

The first four sections of this file are the single canonical, Boss-facing **Revenue Engine job
menu**. The client Overview / Revenue Engine section and Sam's complete menu use the same 26 rows,
in the same group and row order. The existing Overview Actions and the detailed capability
reference remain available below; this menu adds a simple way to discover work Sam can do and does
not replace those operational controls. A menu item is named as a job the Boss already understands,
never as a collector capability, stage, module, provider operation, or campaign schema.

The Revenue Engine section is a menu, not a setup wizard. Selecting any item sends (or makes ready to send) its exact
**Prompt to Sam** in the pinned **SAM** chat. Sam owns the rest: ask only for a missing client, URL,
file, audience, or target; inspect current state; choose the right tools and campaign shape; apply
plan and safety gates; and show anything that needs approval. The Boss never has to translate a
job into a technical workflow.

`{client}` is replaced with the current client before the prompt is shown. Text inside angle
brackets is the one piece of job-specific input the Boss may replace. If it is still missing, Sam
asks one short follow-up instead of sending the Boss to another configuration screen.

## Find leads now

| ID | Title | Value | Prompt to Sam |
|---|---|---|---|
| `extract_post_leads` | Extract leads from a post | Paste any Facebook post to find qualified buyers among the author and commenters. | `Extract qualified leads from this post for {client}: <paste post URL>` |
| `find_group_leads` | Find leads in a group | Search a Facebook group for people whose posts or comments match your buyer profile. | `Find qualified leads in this Facebook group for {client}: <paste group URL>` |
| `find_people` | Find people by role or location | Find prospects by job, business type, market, location, or real-world situation. | `Find people for {client} matching: <role, business type, location, or situation>` |
| `friend_network` | Find leads through a friend network | Turn a visible Facebook friend list into qualified, deduplicated CRM leads. | `Find leads for {client} in this Facebook friend list: <paste profile URL>` |
| `problem_signals` | Find people talking about a problem | Spot conversations where people describe the pain, goal, or buying signal you solve. | `Find people for {client} who are talking about: <problem or desired outcome>` |
| `watch_source` | Watch a group, page, profile, or competitor | Monitor a source you choose for new prospects and useful market signals. | `Watch this source for {client}: <paste URL>` |
| `import_list` | Import a lead list | Bring in a CSV, spreadsheet, or text list, then deduplicate and qualify each lead. | `Import this lead list for {client}: <paste file path>` |
| `directory_leads` | Find leads from Zillow and directories | Collect matching prospects from Zillow or another supported business directory. | `Find directory leads for {client} matching: <target profile and location>` |

## Attract inbound leads

| ID | Title | Value | Prompt to Sam |
|---|---|---|---|
| `research_content_signals` | Research content ideas from post comments | Find useful questions, pains, and opinions in recent noteworthy posts and comments, then update the evidence bank and Idea Matrix without writing content. | `Research recent high-signal posts and comments for {client}, update the Content Evidence Bank and Idea Matrix, and do not write content yet.` |
| `daily_ideas` | Get today's best content ideas | Turn current customer questions, trends, and lead signals into the best ideas to publish today. | `Run today's Solo Agency report for {client}` |
| `write_content` | Write blog and social posts | Turn one idea into a useful blog post and channel-ready social copy. | `Write the blog and social posts for {client} from: <idea or topic>` |
| `make_video` | Make a video | Turn an approved idea or report item into a finished video. | `Make a video for {client} from this approved idea: <idea or report item>` |
| `comment_authority` | Build authority with helpful comments | Join relevant conversations with useful comments that demonstrate expertise. | `Create a helpful comment campaign for {client}` |
| `group_post` | Post where the audience gathers | Create valuable posts for the groups where your ideal customers already spend time. | `Create a group-post campaign for {client}` |
| `publish_content` | Publish approved content | Publish content you have approved through the right connected channel. | `Publish this approved content for {client}: <content or file>` |
| `content_analytics` | See what content is working | Review performance signals and learn which topics and formats deserve more attention. | `Review content analytics for {client} and tell me what to do more of` |

## Nurture and convert

| ID | Title | Value | Prompt to Sam |
|---|---|---|---|
| `enrich_leads` | Check and enrich qualified leads | Verify fit and add useful profile, business, contact, and recent-activity context. | `Enrich the qualified leads for {client} that are ready for outreach` |
| `email_campaign` | Start personalized email outreach | Build an evidence-based email campaign for a qualified CRM audience. | `Set up a personalized email campaign for {client}` |
| `messenger_dm` | Draft Messenger messages | Prepare personalized Facebook messages for selected CRM leads, ready for your approval. | `Draft Messenger messages for these {client} CRM leads and show them for approval: <names or segment>` |
| `follow_up` | Follow up without sounding repetitive | Find due follow-ups and add a fresh reason for each lead to reply. | `Show follow-ups due for {client} and draft the next value-add message` |
| `review_approvals` | Review everything waiting for approval | See every email, message, comment, and post that needs your decision. | `Show everything waiting for approval for {client}` |
| `crm_pipeline` | See who needs attention next | Review pipeline stages, replies, hot leads, and overdue next actions in one place. | `Show the CRM pipeline and leads needing attention for {client}` |

## Run automatically

| ID | Title | Value | Prompt to Sam |
|---|---|---|---|
| `run_revenue_engine` | Run the whole revenue engine today | Find leads, create content, update learning, and produce today's report in one run. | `Run the Solo Agency Daily Run for {client} now` |
| `schedule_revenue_engine` | Run it automatically | Choose a cadence so the revenue engine keeps working without a daily reminder. | `Set up the Solo Agency Daily Run schedule for {client}` |
| `latest_report` | Show the latest report | Open the newest report with leads, opportunities, content ideas, drafts, and results. | `Show me the latest Solo Agency report for {client}` |
| `notifications` | Get reports and lead alerts | Receive report updates and timely alerts when a hot lead needs attention. | `Turn on report and hot-lead notifications for {client}` |

## Sam's menu behavior

### Complete-menu requests

When the Boss asks any equivalent of “what can Solo Agency do?”, “how can it find leads?”, “show
me the features”, or “show/open Revenue Engine”, Sam:

1. shows all four groups and all 26 rows above in chat, preserving their order;
2. opens the current client's Overview / Revenue Engine page at `/ui/{client}` under the
   Answer-and-Show Rule; and
3. asks which job the Boss wants Sam to run.

This is an explicit full-menu request, so it is not limited by the normal 1–3-item feature-tour
pace. If the install has multiple clients and the conversation does not identify one, Sam still
shows the complete menu, then asks which client before opening or running client-specific work.

### Always-visible awareness anchor

On every post-setup interactive Sam reply with no required human action, show the compact Revenue
Engine anchor defined in `SOLO_AGENCY_PLAYBOOK.md` immediately before the next-jobs block. It names
the three human outcomes — Find leads now, Attract inbound leads, Nurture and convert — and links to
the current client's `/ui/{client}` page (or says `show me the features` when the dashboard is not
available). It is a navigation reminder, not one of the 26 jobs and not a second CTA.

Never show the anchor in First Words, before Setup Flow completes, in a complete-menu reply, in any
reply containing `**[ACTION REQUIRED]**`, or in a scheduled/provider notification, INTERNAL_REPORT,
or client-facing artifact. A required-action reply also suppresses the optional feature-tour block;
both return in the next eligible reply instead of competing with the required action.

### Everyday language maps to the same jobs

The exact prompts above are stable card copy, not magic commands. Sam recognizes ordinary
paraphrases and maps them to the same feature ID. At minimum:

| What the Boss may say | Feature ID |
|---|---|
| “extract leads from this post”, “who commented here?”, “get the buyers under this post”, “lấy lead từ bài/post này” | `extract_post_leads` |
| “find leads in this group”, “search this group for prospects”, “tìm lead trong group này” | `find_group_leads` |
| “find realtors in California”, “find people with this job”, “tìm người đúng chân dung này” | `find_people` |
| “find people asking for help with X”, “who is talking about this problem?” | `problem_signals` |
| “keep an eye on this group/page/profile/competitor” | `watch_source` |
| “email these leads” | `email_campaign` |
| “DM these CRM leads” | `messenger_dm` |
| “run everything today” | `run_revenue_engine` |

Contextual feature discovery follows the Boss's current job rather than rotating randomly:

| Current context | Prefer these feature IDs |
|---|---|
| A post URL or “who commented here?” | `extract_post_leads` |
| A Facebook group or community | `find_group_leads`, then `watch_source` when recurring monitoring helps |
| A profession, business type, location, or situation | `find_people` |
| A named pain, problem, or desired result | `problem_signals` |
| Leads just landed in CRM | `enrich_leads`, then the eligible `email_campaign` or `messenger_dm` path |
| An idea or draft was selected | `write_content`, `make_video`, or `publish_content` according to what is missing |
| The Boss wants stronger evidence from post comments before choosing an idea | `research_content_signals`, then `daily_ideas` |
| Content has already been published | `content_analytics` |
| Nothing is pending and lead flow is quiet | `run_revenue_engine`, `find_group_leads`, or `find_people` |

At most 1–2 contextual items appear in an ordinary reply, and they must not repeat a job already in
that reply's next-jobs block. When human action is required, show none of them and defer the moment.

If a post URL is inside a group, `extract_post_leads` remains the job: the source is the named
post and its author/commenters. `find_group_leads` means search or scan across the group. Sam asks
for the URL when the request does not contain it; it never makes the Boss choose an internal
Facebook operation.

### Honest result and approval rules

- Lead extraction qualifies, records source evidence, removes competitors/noise, deduplicates,
  and writes eligible results to CRM. Some sources expose only a name and profile URL, so a valid
  result may initially be a **profile-only lead** with no visible email or phone; enrichment is a
  separate next step, and Sam never invents contact details.
- Sam reads only what the connected account is allowed to view. Finding or watching a source never
  joins a group, sends a friend request, or contacts anyone.
- Nothing is emailed, messaged, commented, posted, published, or otherwise sent without the Boss's
  approval. Live comment and group-post actions require the plan's `write_actions`; unavailable
  actions can still be prepared for review. Locked CRM contacts cannot be enrolled, emailed, or
  messaged until the plan opens them.
- Video/publishing uses the client's verified connected provider and may spend its credits; Sam
  presents the required approval before production or publishing.
- `research_content_signals` is on demand by default. Sam may set up a separate research automation
  only when the Boss asks. Setup never creates it automatically. The job loads
  `CONTENT_SIGNAL_RESEARCH.md` only for this work, updates the Content Evidence Bank and Idea Matrix,
  and never publishes, sends, writes CRM records, or drafts content.

## Feature Discovery Rule — ordinary replies stay paced

For an ordinary reply that is not a complete-menu request and contains no
`**[ACTION REQUIRED]**`, select only 1–3 relevant features the
Boss has not used yet. Derive use from state on disk (for example a campaign exists, a video was
produced, a source is active), rotate through unused IDs, never repeat the same feature in two
consecutive messages, and re-surface a declined feature only after roughly a month. Keep each item
to its one-line value plus its exact **Prompt to Sam** above. A lead-detected moment prioritizes
the most relevant `Nurture and convert` job.

If an action-required block is needed, omit both this feature-tour block and the always-visible
Revenue Engine anchor from that reply. Do not place either above or below the required block; surface
the deferred feature moment in the next no-required-action reply.

This paced tour is independent from `playbooks/NEXT_JOB_CATALOGUE.md`. That catalogue selects the
next state-driven work on every reply; this file helps the Boss discover unused jobs. Do not merge
their clocks, rows, or selection logic, and do not repeat the same job in both blocks of one reply.
Plan meters and upgrade messages are state facts governed by the Next Job Catalogue, not feature
tour items.

Maintenance and support procedures (health checks, update watch, reconnecting accounts, asking the
support group) remain available under their own operational playbooks. They are intentionally not
extra Revenue Engine cards and must not be inserted as a second, competing feature menu.

## Detailed capability and plan reference

The Revenue Engine menu above is the complete menu Sam shows and the UI renders. The older
module-based tables below remain as an additive implementation reference for plan behavior,
prerequisites, accepted legacy trigger phrases, and the stages that deliver each capability. They
are not a second Boss-facing menu. When wording differs, the 25-row menu above supplies the card
title, value, order, and exact **Prompt to Sam**; a trigger phrase below remains a valid alias.

## Plans (two upsell triggers — the contact cap, and write actions on Free)

Read the install's plan from `GET /status` → `entitlement.tier` before surfacing a feature; the ladder and the full PRIMING / METER / SELLING contract live in `AGENTS.md` ("Upsell rule"). Almost nothing is plan-gated — every data feature is on every plan, Free included; the two things a plan actually changes are the CRM contact cap and whether `write_actions` (group post, comment, react) works. Plan is not purely reactive, but it is not a sales pitch either: a one-sentence PRIMING fact (no link, no ask, spoken the way a real person would offer it) may surface proactively at the six funnel moments A-F defined in `AGENTS.md`; a persistent METER line reports the real `contact lock-status` numbers whenever any contact is locked, or the approaching-cap line once the open-contact ratio is high; a SELLING `**[ACTION REQUIRED]**` block runs only on the two real triggers there (first locked contact in a session, or a refused write action on Free). None of PRIMING/METER/SELLING is a feature-tour item — they never rotate, never wait out a cooldown, and never count against the one-feature-discovery-block-per-message limit below.

- **Every plan, Free included:** Daily content ideas, Blog + social posts, Custom source monitoring for the sources you connect, Lead & Competitor detection on them, Daily/weekly reports, Analytics loop, Notifications, Auto update-watch (notify-first and auto-apply), Collector healthcheck, Import a contact list, CRM pipeline (Free: up to <!--plan:free.max_contacts-->100<!--/plan--> CRM contacts unlocked — every lead is still captured), Approval report, Cold-email outreach (per-sendbox Gmail quotas apply), Follow-up engine inside those campaigns, Lead enrichment (dossiers, contact ladders), DM to unlocked contacts, lead harvest (friends lists, people search), Zillow directory + enrich, priority adapter fixes.
- **Starter and up: write actions** — post into groups, comment, react (`write_actions`). Free cannot do these three (the one exception is the support-group post below, allowed on every plan for that single destination).
- **CRM contact cap by plan:** <!--plan:ladder_line-->Free 100 · Starter $49 → 500 · Pro $99 → 2000 · Business $199 → 10000 · Enterprise (contact us) → unlimited<!--/plan--> (rendered from `plans.json`; speak these numbers only via `tool plans show`). A locked contact (above the cap) has no detail view, cannot be emailed, cannot be DM'd, and cannot be enrolled in a campaign — contacts above the cap are still captured, just locked until you upgrade.
- **Every plan:** the client's content library (`tool content`, `/ui/{client}/content`) — every video, post, blog and link this client has, searchable on your own machine, with no server call and no plan gate.
- **Every paid plan runs on one machine per key** (`AGENTS.md`, "One key, one install"): a second install of the same key is Free with `entitlement.reason: seat_limit`; moving to a new computer is `tool entitlement release` on the new one (once per 7 days).
- **Add-on:** Video creation and publishing spend WideCast credits on the same account.

## Content pipeline (Solo Agency)

| Feature | Value (say this first) | Prerequisite | Trigger phrase | Delivered by |
|---|---|---|---|---|
| Daily content ideas | Fresh, audience-first video/blog/social ideas every day, tuned to your client's pains and pillars | client profile (done at setup) | "run today's content" / run the client's Daily Run task | Automation Flow, Stage 4/10 |
| Video creation (WideCast) | Turn an approved idea into a real short video — you record ~5 min or use an AI avatar, the system does the rest | PDNA/WideCast key connected | "make a video from today's best idea" | Stage 3 + WideCast |
| Blog + social posts | Turn one idea into a blog and platform-ready social posts | (optional) WideCast for publishing | "write the blog and social posts" | Stage 3 |
| Lead & Competitor detection | Find people who need your client's offer, and watch what competitors are doing, from monitored sources | default or custom sources active | "show me leads and competitors" | Stage 10 |
| Custom source monitoring | Watch the groups/communities your audience actually gathers in, for pains, questions, and leads | sources configured (the Local Collector itself connects automatically during setup's "Kết nối Facebook, Instagram and X" step, before this is ever offered) | "add sources" / "watch this URL" | Stage 2 + Local Collector |
| Instagram discovery (beta, read-only) | Search Instagram posts and people, then read the best hits' profile activity and post comments — feeds the same Lead & Competitor detection as Facebook and X, one leg of the Social Discovery Pass | Instagram connected (part of setup step 4, "Kết nối Facebook, Instagram and X") | "show me leads and competitors" (Instagram is one source among the ones that ran) | Stage 10, Social Discovery Pass |
| X discovery (beta, read-only) | Search X posts (Latest) and people, then read the best hits' profile activity and post replies — feeds the same Lead & Competitor detection as Facebook and Instagram, one leg of the Social Discovery Pass | X connected (part of setup step 4, "Kết nối Facebook, Instagram and X") | "show me leads and competitors" (X is one source among the ones that ran) | Stage 10, Social Discovery Pass |
| Daily/weekly reports | A clean HTML report of ideas, drafts, leads, and opportunities — daily for you, a scrubbed weekly one for the client | client configured | "show me the latest report" | Stage 6 |
| Analytics & learning loop | Once content is published, measure what worked and feed it back into better ideas | published URLs exist | "review analytics" | Stage 5 |
| Notifications (Telegram + email) | Get the report and hot-lead alerts pushed to you the moment they happen | WideCast key connected | "turn on notifications" | PDNA notification |
| Auto update-watch | A daily task that checks GitHub and keeps your Solo Agency install current; tells you when a new version changes behavior (or auto-applies if you opt in) | schedule/automation exists | "set up the update watch task" | Stage 11 + `Solo Agency - GitHub Update Watch` |
| Collector healthcheck | Know the same day when Facebook or Zillow changed and a collector capability stopped reading, with the exact thing to fix — instead of a silently empty scan a week later | Local Collector running + a few targets you own (test group, your own post, a Zillow profile) | "run the collector healthcheck" | `playbooks/HEALTHCHECK.md` + `<bridge> tool healthcheck` |
| Ask the founder and community for help | When something is stuck or you want a feature, your Team Leader drafts a redacted post, shows it to you, and with your OK posts it to the Solo Agency Facebook support group with your account, then watches the thread for answers — every plan | member of the support group in the collector's Chrome profile | "post this to the support group" | `playbooks/TEAM_MODEL.md` (Support requests) + `fb.group.post` |

## Outreach + CRM (OutreachCRM module)

| Feature | Value (say this first) | Prerequisite | Trigger phrase | Delivered by |
|---|---|---|---|---|
| Cold-email outreach | Reach the leads directly with personalized, evidence-backed cold emails — nothing sends until you approve. After Solo Agency setup it takes just 3 questions: your lead list, a Gmail App Password, and one goal+URL confirmation | a sendbox + a list | "set up a cold-email campaign" | OutreachCRM Stage 1-6 |
| Import a contact list | Bring a CSV/list of prospects in, deduped and suppression-checked | none | "import a list" | OutreachCRM Stage 3 |
| Lead enrichment | Gather verified, evidenced hooks on each contact so every email is genuinely personal | a list imported | "enrich my leads" | OutreachCRM Stage 4 |
| Follow-up / bump engine | Automatic, distinct follow-ups that add new value each time, never "just checking in" | a campaign running | (runs in the daily task) | OutreachCRM Stage 10 |
| CRM pipeline | Replies become deals moving through stages, with tasks and reminders | a campaign running | "show my pipeline" | OutreachCRM Stage 13 |
| Approval report | Review every drafted email on your phone and approve/edit/reject by chat | drafts produced | "show the approval report" | OutreachCRM Stage 6 |

## The one funnel (why cross-introduce)

Content is passive reach (people find the client); outreach is active reach (the client reaches people). They share the same client workspace and the same leads. So:

- In a content-pipeline session, once leads are detected, introduce Outreach: "I found N leads — I can also reach them directly with personalized cold emails. Say `set up a cold-email campaign`."
- In an OutreachCRM session, introduce Content/video: "I can also build a steady daily content engine so leads come to you too, and turn approved ideas into real videos. Say `set up content` / `make a video`."

Keep the data boundary intact (each product reads/writes its own subtree; the only cross-read is the one-way Stage-1 profile bootstrap). Introducing a feature is not a data read — it is always allowed.
