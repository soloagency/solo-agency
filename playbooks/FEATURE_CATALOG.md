# Feature Catalog — the tour-guide menu

This is the single canonical menu of user-facing capabilities across BOTH products (Solo Agency content pipeline + the OutreachCRM cold-email/CRM module). The Feature Discovery Rule draws from this catalog to introduce unused capabilities to the human. It is the honesty guardrail: only surface features listed here, phrased with their real value, prerequisites, and exact trigger phrase — never invent a capability.

Load this file when composing a feature-discovery suggestion (setup completion, report/notification handoffs, the weekly re-surface, or a lead-detected moment). Keep each surfaced item to one scannable line.

## How to use this catalog

- Pick 1-3 features the human has NOT used yet (derive "used" from what exists on disk — see the Feature Discovery Rule). Never surface more than 3 in one message; never repeat the same feature twice in a row.
- Lead with the value, then the exact phrase the human says to start it. Plain language, first-time-user framing, no pressure, no implied Solo Agency-provider affiliation.
- If a feature has an unmet prerequisite, say what setup it needs first (do not present it as one-step when it is not).
- When a run detects leads or competitor moves, the Outreach features move to the top of the suggestion list (that is the highest-intent moment to cross-sell).

## Content pipeline (Solo Agency)

| Feature | Value (say this first) | Prerequisite | Trigger phrase | Delivered by |
|---|---|---|---|---|
| Daily content ideas | Fresh, audience-first video/blog/social ideas every day, tuned to your client's pains and pillars | client profile (done at setup) | "run today's content" / run the client's Daily Run task | Automation Flow, Stage 4/10 |
| Video creation (WideCast) | Turn an approved idea into a real short video — you record ~5 min or use an AI avatar, the system does the rest | PDNA/WideCast key connected | "make a video from today's best idea" | Stage 3 + WideCast |
| Blog + social posts | Turn one idea into a blog and platform-ready social posts | (optional) WideCast for publishing | "write the blog and social posts" | Stage 3 |
| Lead & Competitor detection | Find people who need your client's offer, and watch what competitors are doing, from monitored sources | private/public sources active | "show me leads and competitors" | Stage 10 |
| Private source monitoring | Watch the groups/communities your audience actually gathers in, for pains, questions, and leads | Local Collector + approved sources | "add private sources" / "run discovery" | Stage 2 + Local Collector |
| Daily/weekly reports | A clean HTML report of ideas, drafts, leads, and opportunities — daily for you, a scrubbed weekly one for the client | client configured | "show me the latest report" | Stage 6 |
| Analytics & learning loop | Once content is published, measure what worked and feed it back into better ideas | published URLs exist | "review analytics" | Stage 5 |
| Notifications (Telegram + email) | Get the report and hot-lead alerts pushed to you the moment they happen | WideCast key connected | "turn on notifications" | PDNA notification |
| Auto update-watch | A daily task that checks GitHub and keeps your Solo Agency install current; tells you when a new version changes behavior (or auto-applies if you opt in) | schedule/automation exists | "set up the update watch task" | Stage 11 + `Solo Agency - GitHub Update Watch` |

## Outreach + CRM (OutreachCRM module)

| Feature | Value (say this first) | Prerequisite | Trigger phrase | Delivered by |
|---|---|---|---|---|
| Cold-email outreach | Reach the leads directly with personalized, evidence-backed cold emails — nothing sends until you approve | a sendbox + a list | "set up a cold-email campaign" | OutreachCRM Stage 1-6 |
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
