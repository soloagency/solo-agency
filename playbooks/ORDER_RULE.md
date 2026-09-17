# ORDER RULE — any job the Boss describes runs from its description

## Load Rule

Load this file when a Boss request is not a row of `playbooks/FEATURE_CATALOG.md` /
`playbooks/NEXT_JOB_CATALOGUE.md`, or the Boss wants a catalogued row shaped their own way; at
takeover step 2 of `AGENTS.md`; and in the Daily Run's orders pass (`playbooks/04_DAILY_SCHEDULE.md`
step 28d, `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` step 16B).

## Scope and vocabulary

Same pattern as a campaign's `goal.description` and `companion_doc.instructions`: the Boss's words are
stored verbatim, the agent derives everything else, and the stored brief is the permission to execute
at every later run. Nothing in this rule names an industry, a platform quirk or a data field: what the
business is lives in the client profile and in the brief the agent writes for that client. The
vocabulary is the house one: a request is an **order** (`boss_orders.md` is its ledger), its brief is
the **order brief**, and the Daily Run's pass over briefs is the **orders pass**.

## 1. Intake is one question, often none

- Store the Boss's words verbatim. Never ask for a task type, a capability name, a cadence vocabulary,
  a field list or a template.
- **Which client.** One client on the install → use it and say so in half a sentence. Several → use the
  one the words identify (a name, a product, a place, an audience that matches exactly one client) and
  say which one you took; ask only when the words fit more than one, once, before anything else.
- **Once or recurring** is decided by the nature of the outcome, not by the verb: work that
  completes (a named batch, a list, one send, one report) is once; work that is a stream (more leads,
  staying present, keeping in touch, watching anything) recurs, whatever word the Boss used. A
  recurring order with no stated frequency runs on the client's Daily Run cadence — say so in the
  reply ("I'll do it every day with the Daily Run; say the word if you want another time or rhythm").
  Ask only when the words truly cannot tell once from recurring.
- **The goal.** Take it from the words when it is there ("so more people know me", "so they come
  back", "to close them"). When the order will write to people and the words carry no goal, ask for it
  in one sentence together with anything else missing; no answer → the default goal is presence and
  trust through value (knowledge, advice, help), never selling — write the default into the brief and
  say it in the reply. The goal becomes `goal.description` of every campaign the order creates.
- **Missing concrete things** (a URL, a list, a file the Boss mentioned but did not attach, an audience,
  a target, the account to publish from when the install has more than one, a prerequisite the Boss
  must set up) → one question or one `**[ACTION REQUIRED]**` block naming all of them, with the steps
  when the Boss has to do something. Parts of the order that need nothing start now; the parts
  waiting for an answer are marked `waiting_boss` in the brief and in `boss_orders.md`. Never block a
  whole order on one missing thing.
- Thresholds the Boss left open (how long is "long", how many is "a few") are not questions: pick a
  sensible default, write it in the brief, and say it in the reply so one word from the Boss changes it.
- **Targets the Boss did not list** (a place, a topic, "the groups", "people asking about X") are
  chosen the way the Daily Run already chooses them: the monitored sources and the Group Potential
  Rule, plus searches on the words the Boss used. Name what you will scan in the reply.

## 2. Derive the plan from the hands you actually have

Read the capability catalog (`GET /capabilities`, mirrored in
`solo-agency-collector/bridge-go/collector_capabilities.json`) and
`solo-agency-collector/CAPABILITY_MATRIX.md` (built / feasible / impossible per platform). A
capability id that is not in the catalog does not exist, whatever the request sounds like; the
catalog's `feature` tag says what the plan gates, its `tier` label is legacy. Map the Boss's words
onto:

- **targets** — groups, pages, profiles, searches, URLs, a CRM segment, a file;
- **reads** — capability ids and bridge tools with their inputs, in order;
- **judgment** — the rule that fits the artifact: leads → `playbooks/LEAD_QUALIFICATION_RULE.md`;
  comments and replies → `playbooks/COMMENT_TRIAGE_RULE.md`; anything else → the Boss's own criteria
  quoted from their words; layering two is fine. Bulk (> 5 records) goes to an extractor or worker
  sub-agent per `playbooks/TEAM_MODEL.md`, file in, file out; the leader keeps the judgment;
- **writes** — every outward artifact, routed by WHERE it goes, never executed by the order itself:

  | outward artifact | how it is drafted | where the Boss approves | who executes after approval |
  |---|---|---|---|
  | email to a person | `draft write` in a campaign of that order | Approval page | the campaign's Send step |
  | comment or reaction on someone's post; post into a group; post on the profile timeline | `draft comment` / `draft post` in a campaign of that order (`channel_strategy` comment or post, the target listed in `audience.groups`) | Approval page | the bridge publishes on approve, paced by the daily caps |
  | post on the business's own Page or business account, video, blog | the content pipeline (WideCast connected account) | the content Approval Workflow | WideCast publish |
  | direct message to a person (Facebook Messenger, Instagram) | `draft dm` in a campaign of that order (`channel_strategy` messenger) — only to a CRM contact the contact cap left unlocked, once per person per campaign | Approval page | the bridge sends on approve, capped by `dm_per_account_per_day` |
  | reply on X (`x.post.reply`), comment on Instagram (`ig.post.comment`) | `draft comment` with `platform` x or instagram in a campaign whose `audience.groups` lists the accounts to answer (or `"*"` for search-driven replies; Facebook never wildcards) | Approval page | the bridge publishes on approve, same caps |
  | a write the bridge has no dispatch path for yet (X direct messages; posting to an Instagram account) | drafted and shown in chat, or routed to the content pipeline | in chat, per message | the agent submits the approved piece once; say why |
  | anything with no row here (a channel, a platform or a site the matrix does not build) | — | — | say it cannot be done and offer the nearest built alternative |

  Nothing leaves under the client's name without passing one of those approval points; a request to
  skip approval is declined in one sentence and answered with the batch-approve offer.
  **The Boss's words win over the house defaults of a channel** (for a first Facebook comment the
  default is value in text, no link, no pitch, the profile as the call to action). When the words ask
  for what a default advises against, do what the Boss asked — and say the concrete risk once, with
  the safer variant ready ("Facebook watches accounts whose every comment carries a link; I'll add
  the link only where it truly fits and answer in text otherwise — say the word if you want it in
  every one"); record the choice in the brief. What never yields to anyone's words: the approval
  points, the account identity, the daily caps, and the law.
- **campaigns** — each write lane an order needs is a campaign the agent creates (`campaign create`)
  with `goal.description` = the Boss's words, the lane's `channel_strategy`, and the targets in
  `audience.groups` or `audience.segment`; the brief lists its campaigns; when the reads discover new
  targets the agent adds them with `campaign update`, never by bypassing the audience;
- **CRM work the order owns** — a date-to-act-on the order discovers becomes a CRM task on the contact
  (`outreach/playbooks/13_CRM_CORE.md`, "What the agent stores about a person") with `guard_key` = the
  order slug, so the orders pass finds its own due work and no other pass mistakes it for theirs;
- **report** — one line per run: what ran, the counts, the links, what waits for approval;
- **self-check** — what evidence you will READ before saying it worked, and what "nothing found" is;
- **prerequisites** — before promising a lane, read what it needs on this install and say what you
  found: the plan's daily write allowance (`contact lock-status` → `writes_today`; every plan carries every
  feature; a counted plan — Free: <!--plan:free.writes_line-->1 post, 3 comments or replies, 3 direct messages per day<!--/plan--> — publishes that many a day per
  install and holds the rest for the next day, so say how many of the order's items go out per day and
  that the rest wait or need an upgrade), the account or login it publishes from (the Login
  Reminder state, `{platform}_lead_source`), the sendboxes an email lane needs (`sendboxes.json`:
  none → the setup step is part of the one question; N → "I'll use the N sendboxes already set
  up"), the connected accounts a Page or content lane needs, the notification channel the report
  rides on. A lane whose prerequisite is missing is written into the brief with the gate named and
  no campaign is created for it until the gate opens;
- **what already exists** — say which existing mechanisms the order rides instead of promising to
  build them: the client's Daily Run and its time, the monitored sources, lead capture into the CRM
  at detection, the content library, the Approval page, the daily caps, the notification channel.

What the matrix marks impossible or not built: say so plainly with the nearest built alternative.
Offering a substitute is not a promise and not a question: its read side runs now and the reply says
so; only publishing on a platform or channel the Boss did not name waits for the Boss's word. What
the plan gates: say so once per the Upsell rule; the order still runs its read side.

## 3. Answer, then write the brief

Reply in the Boss's language the way a good employee takes an assignment — five parts, in this
order, short:
1. **What I understood** — the order in your own words, one or two sentences, so a wrong reading is
   corrected now (this replaces most questions: state the understanding, ask only the missing thing).
2. **How I will do it** — when it runs, what I scan, how I judge, what I draft, what I attach, what
   waits for their approval, what I read before I claim anything; in plain words the Boss can picture.
3. **When something is missing or goes wrong** — what I do when no matching content exists, when a
   platform is not connected or the login is stale, when nothing is found, when a daily cap is
   reached, when a plan gate blocks a lane, when something cannot be done at all (say so, name the
   nearest built alternative). Anything the Boss must provide or set up for the order to run (a file,
   a list, a URL, an app password for a sendbox, connecting an account, the notification channel, a
   plan) is asked for or walked through right here in one `**[ACTION REQUIRED]**` block with the
   exact steps the playbooks already carry — never a bare "missing"; the lane waits as
   `waiting_boss`, everything else starts.
4. **What you get** — the output per run, countable: drafts waiting on the Approval page, leads landed
   in the CRM, emails created, where to see each.
5. **How you will hear from me** — after every run the output line goes to the notification channel
   (Telegram/email through the Daily Run's channel) and into the daily report's Orders section; when
   drafts are waiting for approval the run's message says so and the next reply in chat reminds the
   Boss, without being asked; if no notification channel is connected, say so now and offer to
   connect it.

Then write the brief at
`daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/orders/{order_slug}.md`,
prose only, exactly these headings:

```md
# {title in the Boss's language}
status: active | waiting_boss | paused | done · cadence: {Boss's words or "Daily Run"; the most frequent part when parts differ} · client: {slug} · created: {YYYY-MM-DD} · campaigns: {slugs the order owns, or none}

## Boss's words
{verbatim}

## Done today means
{what one good run produces — countable}

## How I do it
{targets; reads as capability ids with inputs, in order; the judgment rule; defaults you chose; the sub-agent brief if bulk; the campaigns this order owns. An order whose parts run on different cadences names each part with its own cadence here; the orders pass runs the parts that are due}

## What needs approval
{each outward artifact → its row of the routing table; "none" if read-only}

## When something is missing or goes wrong
{no matching content → …; platform not connected / login stale → …; nothing found → …; cap reached → …; plan gate → …; approval waiting → remind}

## Output and report
{the countable output per run; the one-line report template with placeholders; where it goes: the notification channel and the daily report's Orders section}

## Self-check
{the evidence to read before claiming; what counts as nothing found; when to stop and ask}

## Run log
| date | outcome | evidence | next |
```

Record the order in `boss_orders.md` with the brief's path as evidence. The brief is the permission:
at every later run execute it, do not re-ask; only new words from the Boss change it.

## 4. Nothing about an order lives only in a chat

Solo Agency runs several brains, in turn or at once. A brain that arrives later must rebuild the
whole picture from disk, never from the previous brain's memory: `boss_orders.md` lists every order
with the brief's path; the brief holds the words, the plan, the campaigns it owns, its run log;
`campaign_config.json` carries `goal.description`; the order's CRM tasks carry its `guard_key`;
`automation_manifest.md` names which brain owns the Daily Run that carries the orders pass. The
takeover and multi-brain checklists read all of these before touching client state, and an order
executed outside the Daily Run is done only under an `order` lease
(`playbooks/MULTI_BRAIN_OPERATIONS.md`), so two brains never run the same brief.

## 5. Running

- Once: execute now, append the run log, set `status: done`.
- Recurring: the client's Daily Run carries every active brief — no scheduled task per order. Its
  orders pass, after the lead stages and before the report: for each brief with `status: active`
  whose cadence is due → LOAD LEDGER the brief → execute "How I do it" → route every write to its
  approval point → append the run log → add the report line under **Orders** in the daily report and
  to the run's notification line. The Approval reminder rule then tells the Boss what is waiting.
- A brief that could not run gets a run-log row saying why; a due brief is never skipped silently.
- Two consecutive runs with nothing found: tell the Boss, propose the change to the brief you would
  make, keep the brief active until they answer.
- `<bridge> tool orders status --pipeline daily-content-pipeline` lists every brief with its status,
  cadence, whether it is due today, its last run, the drafts waiting on its campaigns and the CRM
  tasks carrying its `guard_key` — read it before the pass and before any sentence about an order.
- Read-Before-Claim applies to every number and state word; the run log never carries a result that
  was not read this turn.
- Never invent a capability, never execute a write, never bypass an approval point, never create a
  scheduled task per order, never index fields to fake a second entity on a contact (see the CRM
  rule in `outreach/playbooks/13_CRM_CORE.md`).
