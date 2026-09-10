# Feature Catalog — the tour-guide menu

This is the single canonical menu of user-facing capabilities across BOTH products (Solo Agency content pipeline + the OutreachCRM cold-email/CRM module). The Feature Discovery Rule draws from this catalog to introduce unused capabilities to the human. It is the honesty guardrail: only surface features listed here, phrased with their real value, prerequisites, and exact trigger phrase — never invent a capability.

Load this file when composing a feature-discovery suggestion (setup completion, report handoffs in the run reply, the weekly re-surface, or a lead-detected moment — never the client notification). Keep each surfaced item to one scannable line.

## How to use this catalog

- Pick 1-3 features the human has NOT used yet (derive "used" from what exists on disk — see the Feature Discovery Rule). Never surface more than 3 in one message; never repeat the same feature twice in a row.
- Lead with the value, then the exact phrase the human says to start it. Plain language, first-time-user framing, no pressure, no implied Solo Agency-provider affiliation.
- If a feature has an unmet prerequisite, say what setup it needs first (do not present it as one-step when it is not).
- When a run detects leads or competitor moves, the Outreach features move to the top of the suggestion list (that is the highest-intent moment to cross-sell); when `contact lock-status` shows `locked > 0`, the persistent meter line (`{L} leads locked under {tier} — {unlocked}/{max} open`, `AGENTS.md` "Upsell rule") takes the actual top slot ahead of any feature pick — it is a state fact, not a tour item, so it carries no rotation and no cooldown (see the Feature Discovery Rule's exemption).

## How this differs from `NEXT_JOB_CATALOGUE.md`

This catalog is the feature-tour menu: it introduces capabilities the human has not tried yet, on a paced cadence (setup completion, report handoffs, the weekly re-surface, a lead-detected moment) so the same feature is never repeated two messages running. `playbooks/NEXT_JOB_CATALOGUE.md` is a different instrument: it is the state-driven offer list the Next-Action Guidance Rule (`SOLO_AGENCY_PLAYBOOK.md`) reads from on every reply that does not close with an `**[ACTION REQUIRED]**` block, ranked by the state poll's priority tiers (backlog, lead generation, lead exploitation, expansion, plan) rather than by what is unused. A reply can close with a next-jobs block and still owe a feature-discovery block later; the two run on independent clocks.

## Plans (two upsell triggers — the contact cap, and write actions on Free)

Read the install's plan from `GET /status` → `entitlement.tier` before surfacing a feature; the ladder and the full PRIMING / METER / SELLING contract live in `AGENTS.md` ("Upsell rule"). Almost nothing is plan-gated — every data feature is on every plan, Free included; the two things a plan actually changes are the CRM contact cap and whether `write_actions` (group post, comment, react) works. Plan is not purely reactive, but it is not a sales pitch either: a one-sentence PRIMING fact (no link, no ask, spoken the way a real person would offer it) may surface proactively at the six funnel moments A-F defined in `AGENTS.md`; a persistent METER line reports the real `contact lock-status` numbers whenever any contact is locked, or the approaching-cap line once the open-contact ratio is high; a SELLING `**[ACTION REQUIRED]**` block runs only on the two real triggers there (first locked contact in a session, or a refused write action on Free). None of PRIMING/METER/SELLING is a feature-tour item — they never rotate, never wait out a cooldown, and never count against the one-feature-discovery-block-per-message limit below.

- **Every plan, Free included:** Daily content ideas, Blog + social posts, Private source monitoring for the sources you connect, Lead & Competitor detection on them, Daily/weekly reports, Analytics loop, Notifications, Auto update-watch (notify-first and auto-apply), Collector healthcheck, Import a contact list, CRM pipeline (Free: up to 30 CRM contacts unlocked — every lead is still captured), Approval report, Cold-email outreach (per-sendbox Gmail quotas apply), Follow-up engine inside those campaigns, Lead enrichment (dossiers, contact ladders), DM to unlocked contacts, lead harvest (friends lists, people search), Zillow directory + enrich, priority adapter fixes.
- **Starter and up: write actions** — post into groups, comment, react (`write_actions`). Free cannot do these three (the one exception is the support-group post below, allowed on every plan for that single destination).
- **CRM contact cap by plan:** Free 30 · Starter $49 → 500 · Pro $99 → 2000 · Business $199 → 10000 · Enterprise (contact us) → unlimited. A locked contact (above the cap) has no detail view, cannot be emailed, cannot be DM'd, and cannot be enrolled in a campaign — contacts above the cap are still captured, just locked until you upgrade.
- **Every plan:** the client's content library (`tool content`, `/ui/{client}/content`) — every video, post, blog and link this client has, searchable on your own machine, with no server call and no plan gate.
- **Every paid plan runs on one machine per key** (`AGENTS.md`, "One key, one install"): a second install of the same key is Free with `entitlement.reason: seat_limit`; moving to a new computer is `tool entitlement release` on the new one (once per 7 days).
- **Add-on:** Video creation and publishing spend WideCast credits on the same account.

## Content pipeline (Solo Agency)

| Feature | Value (say this first) | Prerequisite | Trigger phrase | Delivered by |
|---|---|---|---|---|
| Daily content ideas | Fresh, audience-first video/blog/social ideas every day, tuned to your client's pains and pillars | client profile (done at setup) | "run today's content" / run the client's Daily Run task | Automation Flow, Stage 4/10 |
| Video creation (WideCast) | Turn an approved idea into a real short video — you record ~5 min or use an AI avatar, the system does the rest | PDNA/WideCast key connected | "make a video from today's best idea" | Stage 3 + WideCast |
| Blog + social posts | Turn one idea into a blog and platform-ready social posts | (optional) WideCast for publishing | "write the blog and social posts" | Stage 3 |
| Lead & Competitor detection | Find people who need your client's offer, and watch what competitors are doing, from monitored sources | private/public sources active | "show me leads and competitors" | Stage 10 |
| Private source monitoring | Watch the groups/communities your audience actually gathers in, for pains, questions, and leads | approved sources (the Local Collector itself connects automatically during setup's "Kết nối Facebook" step, before this is ever offered) | "add private sources" / "run discovery" | Stage 2 + Local Collector |
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
