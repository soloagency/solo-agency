# Lead And Competitor Detection

Stage: `10`

## Load Rule

Load whenever the agent is about to detect, score, report, store, or improve lead and competitor opportunities.

Also load during:

- first agency run;
- scheduled daily run;
- private data source scan analysis;
- public research analysis when lead or competitor signals are possible;
- HTML report generation or repair;
- any human request about leads, competitors, comments, replies, opportunities, outreach, or competitor monitoring.

## Hard Gates For This Stage

- Lead and competitor detection is not a light report appendix. Treat it as a core agency function.
- Detect leads and competitors during the same public/private data collection pass used for ideas and market research. Do not add a separate extra scan unless the human explicitly asks.
- For the first lead and competitor pass for a client/source set, use 10 scrolls per approved private data source when Local Collector is active and safe.
- For recurring daily scheduled runs, use 5 scrolls per approved private data source by default, unless the saved Local Collector configuration is lower or the human explicitly configured a safer lower value.
- Source discovery mode is different: it may scroll deeper under Stage 8 rules. Do not confuse source discovery with daily lead/competitor monitoring.
- For private data sources, use only the Solo Agency Local Collector extension plus Local Collector app. Never use Claude in Chrome, Codex built-in browser, Playwright, Puppeteer, Selenium, or any agent-controlled browser for private data source collection.
- Every detected lead or competitor opportunity shown in the HTML report must include the post/current URL when available, context, classification, why it matters, and a copy-ready suggested comment.
- The HTML report must keep public data source opportunities and private data source opportunities separate. Use `Public Lead & Competitor Opportunities` inside `Public Data Source Intelligence` and `Private Lead & Competitor Opportunities` inside `Private Data Source Intelligence`, or the same meaning in the human/report language.
- Suggested comments are for human review only. Do not auto-comment, auto-DM, or initiate outreach without explicit human approval — the **send/act** side always needs a human (see the approve-then-send gate). **This stage never sends, even after approval**: its job ends at "here are qualified leads and a suggested comment for you to review." Approved agency-level outreach is executed in a different module (OutreachCRM Stages 16/17, the operator's own brand and accounts), never from this scan loop, and never as or on behalf of a client. Data **collection and analysis**, on the other hand, is consented by the operator's own setup + command: the agent may read, extract, and combine whatever the operator directs it to research — the operator's own business data (industry, goal, content) and the prospects/sources they point the collector at, including contact details (email/phone) — for lead-finding and email personalization. **The one absolute prohibition:** never read, store, or transmit the operator's own credentials or secrets (usernames, passwords, cookies, tokens, session/auth data, API keys). Do not bypass access controls or CAPTCHAs — read what the operator's own session already renders.
- Comments must build personal brand and trust by adding value. They must not directly advertise the user's service, attack competitors, or sound like generic AI text.

## Source Preservation Rule

This file is detailed source material. Do not summarize away definitions, scan-depth rules, reporting requirements, comment style requirements, storage fields, or completion gates.

---

## Two passes over a watched source

A scan of a private source answers two different questions and needs two different moves, both
bounded by `playbooks/skills/lead-engine/safety.md`:

- **Search** the source for the phrasings that mean somebody needs help right now (`tool
  source-keywords plan|urls` → `fb.group.search_posts`). This is where leads come from, because it
  asks for people with a need instead of reading whatever was posted most recently.
- **Scroll** the feed shallowly for everything else: the shift in what the community discusses, the
  competitor activity, and the phrasings worth adding to the bank.

The daily procedure is in `playbooks/04_DAILY_SCHEDULE.md` step 14. This stage still qualifies
whatever either pass returns — the passes acquire, this stage judges. When a hunt runs instead
(`playbooks/skills/lead-engine`, an interactive "find me leads" ask), it writes the terms that
worked back into the same bank, so tomorrow's monitoring inherits what the hunt discovered.

## Facebook Discovery Pass (step 11C of the daily run)

This pass is step 11C of `playbooks/04_DAILY_SCHEDULE.md` Daily Run Algorithm, inserted between 11B
and step 12/13 (the former step 11C, `notification_channel_missing`, is now 11D). The full spec
lives here; the scheduled-run call-out lives in `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` steps
12D/12E.

"Vòng khám phá Facebook" — the impressive first Facebook lead scan for a client, and afterwards a
light daily companion. It runs INSIDE the client's existing daily run task; there is no separate
automation task for it. Setup Flow's step 4 ("Kết nối Facebook") installs the bridge and the
client's extension well before this pass ever runs (`playbooks/SETUP_FLOW_ENTRYPOINT.md`), and this
pass is normally the client's first-ever discovery pass — but "first-ever" is tracked by
`facebook_discovery_first_pass_done` on the Client Intelligence Profile, not by which automation run
number it is: a client who postpones Facebook at setup and connects it three runs later still gets
the FIRST RUN budget below on that later run, because that is still the first time this pass has
ever executed for them.

Before this pass may run at all, `facebook_lead_source` must be `enabled` (see
`playbooks/SCHEDULED_RUN_ENTRYPOINT.md` step 12D/12E and `playbooks/SETUP_FLOW_ENTRYPOINT.md`).
`pending` (the human has not yet resolved item 4 either way) skips this whole section quietly — no
awareness line, because nothing has been decided. If the human confirmed `web_only` (the
acknowledgment-based escape, never a bare "bỏ qua"/"để sau"), skip this whole section and carry the
persistent web-only awareness line (`playbooks/06_AGENCY_REPORT_STANDARD.md`) instead, plus the
one-line way to turn it back on.

### Fixed order: feed, then people, then groups, then in-group

The order is fixed because each step narrows and ranks what the next step touches. Do not reorder,
parallelize, or skip a step to save budget — skip whole groups instead once step 4's budget runs out.

1. **FEED FIRST.** `fb.search.posts` with `search_url =
   https://www.facebook.com/search/posts/?q=<discovery term>`. This is Facebook's own global Posts
   search — no group membership needed, and it is the fastest first signal of who is using
   in-market language right now.
2. **THEN PEOPLE.** `fb.people.search` with `query = <discovery term>` (the URL Facebook itself
   renders: `https://www.facebook.com/search/people/?q=<discovery term>`). Returns `ProfileSummary`
   rows (`id`, `name`, `url`, `subtitle`, `mutual_friends`, `industry_hint`) — people, not posts, so
   there is no post text to read. Every row goes through Stage 10 as a PERSON lead candidate: the
   `subtitle`/work line is the qualifying signal (e.g. "Realtor at ..." when the client sells to
   realtors), and classification uses `subtitle` + `industry_hint` + `name`/`url` only. Temperature
   defaults to `watch`; it becomes `warm` only when the subtitle states the target role explicitly.
   Qualified rows go to `tool crm-store ... lead capture` like every other lead, tagged
   `source:people_search` plus `kw:{term}`.
3. **THEN GROUPS.** `fb.groups.search` with `query = <discovery term>`. Keep only results whose
   `privacy == "public"`. If `privacy` is empty/unknown, treat it as unknown, not as private: either
   spend one `fb.group.posts` call with `max_pages: 1` on the group to read its header and decide,
   or skip the group when the call budget is tight — never guess it public. Rank the survivors by
   `member_count` desc, preferring names/snippets that match the client's target location. Skip a
   group already in `private_data_sources`, and skip a group this pass already scanned in the last 7
   days. Persist the ranked shortlist at `history/YYYY-MM/facebook_discovery_shortlist.jsonl` — see
   `playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md` for the exact fields and job shapes.
4. **THEN IN-GROUP.** For the top public groups from step 3, `fb.group.search_posts` with
   `group_search_url = <group_url>/search/?q=<intent term>`. Intent terms — not the discovery term —
   come from `tool source-keywords ... plan`; seed the group's bank first if it is empty
   (`seed --industry --market --lang`, plus the client's setup seed file when one exists).

Discovery terms themselves (steps 1, 2 and 3's `<discovery term>`) come from a new keyword-bank
kind, not from the source-keywords intent bank used in step 4:

```sh
<bridge> tool public-keywords --pipeline daily-content-pipeline --client {slug} plan --kind discovery
```

`--kind discovery` is the new `community_discovery` kind (default 3 terms). Add one by hand with
`<bridge> tool public-keywords --pipeline daily-content-pipeline --client {slug} add --group community_discovery --term "..."`.

The `record` verb has no `--kind` flag and requires a verdict. Record outcomes the same way the
public-keywords bank always does, with `urls` = feed posts + people rows + groups found for that
term and `ideas` = leads captured:

```sh
<bridge> tool public-keywords --pipeline daily-content-pipeline --client {slug} record --json '{"{term}":{"verdict":"useful|used|weak|retry_later","urls":N,"ideas":M}}'
```

Verdict is `useful` when the term produced ≥ 1 lead, `used` when it produced results but no lead,
`weak` when it produced nothing, and `retry_later` when a safety trip cut the term short.

### Every post and person row goes through Stage 10 immediately

Every post record this pass returns — from the feed search, from the group list, or from in-group
search — and every `ProfileSummary` row the people search returns, goes through this stage's
Detection Workflow and then straight to `<bridge> tool crm-store ... lead capture` (see "Every lead
also becomes a CRM contact" below), EVEN THOUGH the group it came from is not (yet) an approved
`private_data_sources` entry. The pass acquires and captures; it does not wait for a group to be
promoted first.

### Lead target: a floor, not a stop

FIRST RUN: **minimum 10 leads.** Reaching 10 does not end the run — keep working down the shortlist
until the day's budget (below) is spent; more is always better than exactly 10. If the budget runs
out below 10, say so plainly in the report and carry the remaining shortlist rows (`status: pending`)
into the next day's pass rather than losing them.

DAILY companion pass: no fixed floor — it is a light top-up, bounded by its own smaller budget below.

### Budget (owner-approved, inside the safety envelope; all calls serial, spread over hours)

Which tier applies is decided by `facebook_discovery_first_pass_done` on the Client Intelligence
Profile, not by the automation run's own number: **FIRST RUN** applies whenever this field is not
yet `true` — i.e. this is the first time this pass has ever executed for this client, whether that
happens to be run 1, run 2, or run 20 because Facebook was connected late — and the agent sets the
field `true` immediately after this pass completes. **DAILY** applies to every pass after that.

| | discovery terms | feed searches | people searches | group searches | new public groups | intent terms/group | total collector calls | `max_pages` | spread |
|---|---|---|---|---|---|---|---|---|---|---|
| **FIRST RUN** | 3 | 3 | 3 | 3 | up to 4 | 3 | ≤ 21 | ≤ 4 | ≥ 4 hours |
| **DAILY** | 1 | 1 | 1 | 1 | up to 2 | 2 | ≤ 7 | ≤ 4 | across the run window |

These numbers are ceilings from `playbooks/skills/lead-engine/safety.md`, which carries the full
envelope (pacing, serial-only, the trip rule). Never raise them without the human's explicit
approval.

### Safety trip is unchanged, and unforgiving

The first checkpoint, rate-limit warning, or logged-out signal stops the WHOLE account for the day —
not just this pass, not just the current group. After every single job in this pass, the agent must
read that job's result for those signals BEFORE submitting the next one. This holds inside the ≤ 21
or ≤ 7-call budget exactly as it holds everywhere else in this playbook.

### Scanning needs no per-group approval; promoting does

Scanning a PUBLIC group in this pass needs no per-group human approval — see the join boundary in
`playbooks/skills/lead-engine/safety.md` and the reconciliation paragraph in
`playbooks/PRIVATE_SOURCE_GATE.md`. PROMOTING a group out of the shortlist into `private_data_sources`
for standing daily monitoring still needs the normal per-group human approval
(`playbooks/02_PRIVATE_SOURCE_SETUP.md`). After the pass, recommend the top groups by `leads_found`
and `member_count`; the human decides which (if any) get promoted.

Groups this pass finds should also be registered in the shared source registry as public groups
(`tool source-registry register`, existing shape) so other clients' passes reuse the notes instead of
rediscovering the same group cold.

### Report section: "Facebook Discovery Pass"

Every run of this pass gets its own named report section (translate the title naturally in a
non-English report; English default below):

- discovery terms used;
- feed posts found;
- people found / people captured;
- groups found / groups kept as public / groups scanned in-group;
- leads found (and how many of those are hot/warm/watch);
- locked leads (see "Every lead also becomes a CRM contact" below);
- budget used (calls spent / calls available, for this run's tier);
- trip status (`clean`, or the exact safety trip that stopped the account).

```text
Facebook Discovery Pass
Discovery terms: 3 · Feed posts: 3 · People found: 9, captured: 5
Groups found: 12, public: 7, scanned: 4
Leads found: 19 (9 hot, 6 warm, 4 watch) · Locked: 0
Budget used: 20/21 calls · Trip status: clean
```

## Definitions

### Lead

A lead is a person, account, post, comment, or thread that shows a direct or indirect need related to the product or service the human provides.

Direct lead signals include:

- asking for a provider, expert, quote, recommendation, estimate, or solution;
- describing an urgent problem the user's offer can solve;
- asking what to do next, who to hire, what to buy, or how much something should cost;
- complaining that a current provider, tool, process, or solution is not working;
- comparing options before making a buying decision.

Indirect lead signals include:

- expressing a pain point, fear, objection, confusion, life event, workflow breakdown, or recurring frustration tied to the user's offer;
- asking adjacent questions that usually happen before the buying moment;
- joining a discussion where the same audience is clearly trying to solve a related problem;
- reacting strongly to a competitor post, case study, offer, or educational explanation.

A lead is a signal for the human to review. It is not permission for the agent to contact the person.

### Competitor

A competitor is any person, account, company, creator, product, community, content asset, or alternative solution that competes for the same target audience, buying intent, attention, or trust.

Competitor types:

- `direct_competitor`: offers the same or very similar product/service.
- `indirect_competitor`: solves the same problem with a different service, product, tool, or method.
- `adjacent_solution`: serves the same pain point before or after the user's offer.
- `attention_competitor`: attracts the same target audience even if it does not sell the same thing.
- `authority_or_kol_competing_for_trust`: owns trust, education, or recommendation power in the same audience.

Competitor posts matter because they often attract the same people the user wants to help. The user should either learn from the signal or appear in the discussion with a useful, natural comment.

## Scan Depth Contract

Lead and competitor extraction should happen inside the normal data collection pass:

```text
research/source scan -> data points -> leads -> competitors -> ideas -> best idea -> draft -> report
```

Do not run a second scan just for lead/competitor detection unless:

- the human explicitly asks for a deeper lead/competitor pass;
- the first scan failed or produced too little usable data;
- a saved schedule or config says a deeper first pass is allowed.

Private data source depth:

- First lead/competitor pass for a client/source set: 10 scrolls per approved source.
- Recurring daily run: 5 scrolls per approved source.
- Use `collector_config.scroll_delay_seconds`, defaulting to about 5 seconds.
- Respect `collector_config.max_scrolls_allowed`, account-safety limits, rate-limit warnings, session-expired states, and platform warnings.
- If Local Collector config exists and has lower safety settings, obey the safer lower value and record why coverage is lower.
- If the human configured a higher number, cap it at Stage 8 safety limits unless the collector explicitly supports a safe higher discovery mode.

Human-facing disclosure when scanning private groups/sources:

```text
I will go through each approved group/source one by one and scroll {N} times per source. For the first lead/competitor pass I use 10 scrolls when safe; for normal daily runs I use 5 scrolls. I read this from the Local Collector configuration when available.
```

## Detection Workflow

For every relevant source item, the agent must ask:

1. Does this show a direct or indirect need connected to the user's offer?
2. Does this reveal a pain point, objection, buying trigger, decision moment, or urgency signal?
3. Does this source/post/account compete for the same audience, trust, intent, or attention?
4. Would a useful comment from the human help build presence without selling?
5. Is there a stable post/current URL or source URL that lets the human inspect the context?
6. Is this safe and appropriate to report without exposing unnecessary personal data?

Required lead fields:

- lead label or safe descriptor;
- source/platform;
- source type: public | private;
- profile URL when visible and safe;
- post/current URL when available;
- captured at;
- emails (optional) — email addresses extracted from the page content the collector captured (visible text + `mailto:` links); present when the captured content contains them; empty/absent otherwise;
- phones (optional) — phone numbers extracted from the same captured page content (visible text + `tel:` links), digits normalized (E.164-style with a leading `+` when a country code is present); present when the captured content contains them; empty/absent otherwise;
- evidence snippet or safe summary;
- lead type: direct_need | indirect_need | pain_signal | buying_trigger | objection | comparison | complaint | adjacent_need;
- lead level: hot | warm | watch;
- related offer;
- related pain point;
- confidence: high | medium | low;
- suggested next action for the human;
- suggested value-first comment;
- outreach/compliance note.

Required competitor fields:

- competitor name/page or safe descriptor;
- competitor type;
- platform/source;
- profile URL when available;
- post/current URL when available;
- captured at;
- audience overlap;
- offer/positioning;
- content theme or hook pattern;
- engagement or comment signal;
- threat/opportunity level: high | medium | low;
- what the user can learn;
- suggested value-first comment;
- monitoring action.

## Opportunity Scoring

Lead score dimensions:

- urgency;
- fit with the user's offer;
- clarity of need;
- ability to help without being spammy;
- source credibility;
- location fit when location matters;
- recency;
- confidence.

Competitor score dimensions:

- audience overlap;
- engagement quality;
- repeated pain points in comments;
- positioning strength;
- freshness;
- content pattern usefulness;
- strategic threat;
- opportunity for the user to add a better, clearer, or more helpful perspective.

## Audience Value-First Opportunity Rule

Lead and competitor intelligence must produce useful audience-facing ideas, not direct praise for the user's product/service.

When a lead or competitor signal becomes an idea, best idea, suggested comment, script, blog, caption, or recommendation, it must state:

- the audience pain point or confusion;
- the viewer value / lesson;
- the source signal;
- the non-promotional angle;
- why it helps the audience;
- the soft business relevance to the user's offer.

Do not convert competitor positioning into `the client should out-position them`, `our product is better`, `{Client Product} wins`, `choose us`, or similar direct promotional framing. If the idea cannot teach the audience something useful without selling, reject it as `promotional_not_value_first`.

Bad competitor-derived idea:

```text
MiniMeo out-positions competitors by selling without selling across multiple platforms.
```

Better competitor-derived idea:

```text
How small brands can test whether their "selling without selling" message is actually clear before pushing it across every platform.
```

Use qualitative labels if numeric scoring would slow the run:

```text
high | medium | low
```

## Comment Drafting Rules

For every lead and competitor opportunity in the HTML report, draft one short comment the human can copy.

The comment must:

- use the same language as the post or thread;
- respond to the actual context, not a generic template;
- provide a useful insight, question, clarification, checklist item, or perspective;
- sound like a real person, not an AI assistant;
- be short enough to paste into a social thread without looking like a mini blog post;
- avoid direct selling, self-promotion, `DM me`, `message me`, `inbox me`, `book a call`, `reach out to start`, `we can help`, or service pitches;
- avoid attacking or undermining a competitor;
- avoid guarantees, regulated claims, legal/medical/financial advice, or unsafe instructions;
- avoid over-polished phrasing, generic motivational language, and obvious AI cadence;
- avoid bullet lists unless the platform/context naturally uses bullets.

The comment may include one or two tiny natural imperfections when appropriate, such as a casual spelling choice, a missing accent, or a small human-sounding typo. This is allowed because a slightly imperfect helpful comment can feel more trustworthy than perfectly polished AI-style text.

Do not force typos. Never make the user look careless, rude, uneducated, unprofessional, or unclear. The comment must remain easy to understand.

Good comment pattern for a lead post:

```text
This usually gets easier if you separate the urgent question from the long-term decision. First check what has to happen this week, then compare options after that. A lot of people accidentally mix those two and overpay or choose too fast.
```

Good comment pattern for a competitor post:

```text
One thing I like about this topic is that the "right" answer depends a lot on timing. The same advice can be great before renewal and pretty bad after a notice already arrives.
```

Bad comments:

```text
We help with this. DM me.
```

```text
As an expert in this industry, I strongly recommend scheduling a consultation today.
```

```text
This competitor forgot to mention that our service is better.
```

## HTML Report Contract

The report must include lane-specific sections named:

```text
Public Lead & Competitor Opportunities
Private Lead & Competitor Opportunities
```

If the report is in another language, translate the titles naturally. English defaults:

```text
Public Lead & Competitor Opportunities
Private Lead & Competitor Opportunities
```

A report-level rollup named `Lead & Competitor Opportunities` is allowed, but it must not replace the lane-specific sections and must not mix public and private evidence without visible `source type` labels.

For each opportunity, include:

- type: lead | competitor | both;
- classification, such as hot lead, warm lead, direct competitor, indirect competitor, attention competitor;
- source/platform;
- post/current URL as a visible link when available;
- profile/account URL when visible and safe;
- safe context summary;
- why this matters;
- suggested human action;
- copy-ready comment;
- a real working copy button for the comment.

Copy buttons are allowed because they perform a real local browser action. They must copy only the suggested comment text. They must not imply that the comment will be posted automatically.

Example HTML behavior:

```html
<button type="button" class="copy-comment" data-copy="Helpful comment text here">Copy comment</button>
<script>
document.querySelectorAll('.copy-comment').forEach(function (button) {
  button.addEventListener('click', async function () {
    var text = button.getAttribute('data-copy') || '';
    try {
      await navigator.clipboard.writeText(text);
      button.textContent = 'Copied';
    } catch (error) {
      var area = document.createElement('textarea');
      area.value = text;
      document.body.appendChild(area);
      area.select();
      document.execCommand('copy');
      document.body.removeChild(area);
      button.textContent = 'Copied';
    }
  });
});
</script>
```

If no opportunities were found, the report must distinguish:

- no leads/competitors found after scanning;
- not scanned because private data sources are pending Local Collector activation;
- not scanned because source/session was unavailable;
- public data sources only run with limited private lead coverage.

## Storage Contract

Store lead and competitor opportunities in both the existing lead/competitor logs and the unified opportunities ledger when possible.

Recommended unified ledger:

```text
history/YYYY-MM/lead_competitor_opportunities.jsonl
```

Recommended JSONL fields:

```json
{
  "date": "YYYY-MM-DD",
  "client_slug": "client",
  "opportunity_type": "lead",
  "classification": "hot_lead",
  "source": "Facebook Group",
  "source_type": "private",
  "platform": "facebook",
  "profile_url": "https://...",
  "post_url": "https://...",
  "emails": ["name@business.com"],
  "phones": ["+14155550100"],
  "captured_at": "ISO-8601",
  "safe_context_summary": "Short summary",
  "evidence_snippet": "Short visible snippet if safe",
  "why_it_matters": "Reason",
  "related_offer": "Offer",
  "related_pain_point": "Pain point",
  "confidence": "medium",
  "suggested_action": "Human reviews and decides whether to comment",
  "suggested_comment": "Value-first comment",
  "comment_language": "en",
  "comment_style_notes": "natural, short, no direct pitch",
  "status": "needs_review"
}
```

The optional `emails` and `phones` arrays are additive: they are populated when the collector's structured extractor finds contact details in the captured page content, and stay empty/absent otherwise. Ignoring them keeps existing behavior unchanged. They do not authorize any auto outreach; the send/act gates (approval required) still apply, and the operator's own credentials/secrets are never read or transmitted.

Do not store unnecessary personal data. Keep safe summaries and source URLs. The human can inspect the original post in their logged-in session when needed.

### Every lead also becomes a CRM contact, the day it is detected

The ledger above is the audit trail. It is not where a lead lives. A person who publicly asked for
what this client sells is a lead at the coldest end of a ladder — stranger, cold, warm, engaged — and
the CRM is the thing built to walk them up it. Leaving them in a file nothing reads means no dossier,
no timeline, no follow-up, and the same person rediscovered as "new" next week.

After writing the ledger, capture the run's leads in one call:

```sh
<bridge> tool crm-store --pipeline daily-content-pipeline --client {slug} \
  lead capture --file history/YYYY-MM/lead_competitor_opportunities.jsonl
```

It reads the same JSONL this stage just wrote, so there is nothing to reformat. Per lead it creates a
contact keyed on the profile URL (or, when there is no profile, on the post URL as a content seed),
files the post as a dated evidence hook, tags it `lead`, `lead:{hot|warm|watch}`, `source:lead_scan`
and `kw:{term}` when a search term found them, and writes one `lead_detected` activity. Competitor
rows are refused by name — a competitor is a business to study, not a person to nurture. Add
`--dry-run` to see the mapping without writing.

**Capture never stops at the plan's contact cap.** Free 30, Starter 500, Pro 2000, Business 10000,
Enterprise unlimited — those are the contact caps `tool crm-store` enforces per plan tier. Hitting the
cap does not stop capture: the newest leads above the cap are stored LOCKED (no detail, no email, no DM,
no campaign) rather than dropped, and a contact that has already progressed past `lead` into any `engaged+` stage
is never locked, no matter how the count moves afterward.

Locked count and the unlocked/max ratio both come from `tool crm-store ... contact lock-status` — read
it after every capture that touched the CRM, and every time a Boss-facing reply is about to be sent for
a run that touched the CRM. Never estimate either number.

**The meter (funnel moments G and H; see `AGENTS.md`, "Upsell rule").** Whenever `contact lock-status`
shows `locked > 0`, every Boss-facing reply, `INTERNAL_REPORT`, and morning brief for this client
carries this persistent one-line meter, translated naturally into the human's language, with no
cooldown — it is a plain fact, not an ask, and it repeats on every reply for as long as the count stays
above zero:

```text
{L} leads locked under {tier} — {unlocked}/{max} open
```

When `locked == 0` but `unlocked / max_contacts ≥ 0.8`, carry the approaching-cap line instead of the
locked meter (moment G):

```text
{unlocked}/{max} open contacts used; new leads may start locking
```

**First lock of the session (moment H).** The FIRST time in this session that `contact lock-status`
shows `locked` move from 0 to > 0, deliver the SELLING moment: a full `**[ACTION REQUIRED]**` upgrade
block, spoken the way a real person would offer the upgrade, in the human's own words and language —
never this template verbatim — naming the actual counts from `contact lock-status`. Tone example
(Vietnamese, tone only, not a script to paste):

```text
Em vừa đưa 214 lead mới vào CRM, 31 lead có tín hiệu tốt. Gói Free đang mở 30 contact, 184 lead còn lại
đang khoá chi tiết, chưa gửi mail hay nhắn tin được. Mở gói Starter thì 500 contact mở ngay, không cần
quét lại.
```

The upgrade path is the WideCast key/plan (https://widecast.ai/#setup); the human runs
`tool entitlement refresh` afterward. This is the session's one Upsell-budget use for the locked-contact
reason — every reply after that first 0→>0 transition, for the rest of the session, drops back to the
plain meter line above; it never repeats the full block for the same transition. A Free install refused
`write_actions` (group post, comment, react) is the other SELLING trigger and spends the same
once-per-session budget on its own reason. `seat_limit` and a stale-token `solo_feature_not_in_tier` on
a non-`write_actions` capability are never a selling moment (Upsell rule carve-outs).

**Show the CRM link after any scan that produced ≥ 1 lead.** In the chat reply, this stage's
lead-capture step ends with the CRM line below (Team Leader frame — this line is mirrored in
`SOLO_AGENCY_PLAYBOOK.md` "Team Leader Reply Frame"; keep both copies in sync). In
`INTERNAL_REPORT` only — NEVER in the three client-facing HTML files, the combined client
report/PDF, or the client notification (Client-Blind Deliverable rule) — the same line appears after
any scan that produced at least one lead:

```text
CRM: http://127.0.0.1:17321/ui/{client_slug}/crm — {N} liên hệ mới hôm nay, {L} đang khoá theo gói; nóng nhất: http://127.0.0.1:17321/ui/{client_slug}/contact/{lead_id}
CRM: http://127.0.0.1:17321/ui/{client_slug}/crm — {N} new contacts today, {L} locked by plan; hottest: .../contact/{lead_id}
```

A zero-lead run prints the bare CRM line (`http://127.0.0.1:17321/ui/{client_slug}/crm`, no counts).
The agent PRODUCES this link as a static string from `docs/UI_DESIGN.md`'s URL map; it never
HTTP-GETs it. Hand it per the SHOW RULE (`docs/UI_DESIGN.md` §1 principle 2, OWNER DECISION
2026-09-10): print it as text every time; on Claude Code desktop also open it in the side Browser
pane; on any other local runtime also run `open`/`start`/`xdg-open` so it lands in a real browser.
It opens directly now (`--ui-auth host` default) — no entry-link/token step, no "Locked" page.

**First run for a client that has never had a CRM.** A client set up for content only has no CRM
workspace yet, and the command will say so: `no outreach workspace under .../clients/{slug}; run
init-client first`. That is not a failure of the scan — create the workspace once and re-run:

```sh
<bridge> tool crm-store --pipeline daily-content-pipeline --client {slug} \
  --business {business_slug} --location {location_slug} init-client
```

Do this silently as part of the run; it creates empty directories and sends nothing. If `init-client`
is refused, that is a bridge/technical blocker (there is no per-plan client limit any more; the
bridge keeps only a technical ceiling far above real use) — report the exact error in
`INTERNAL_REPORT` and carry on; the leads are still in the ledger.

**Running it twice is safe and is the normal case.** The same person seen on three days is ONE
contact with three activity rows and one hook per distinct post; the temperature tag is replaced, not
accumulated, so someone who cools off stops matching a "hot leads" filter; a contact who already
became a customer is never walked back to `lead`.

**Storing is not contacting, and this changes no send gate.** A captured lead has no email, so
`channels.email.status` stays `needs_data` and drafting refuses them outright. Commenting, DMs and
mail all still require the human approval they required yesterday. What changes is only that the
person is still there tomorrow — which is the precondition for warming them at all, and the reason
the evidence hook is recorded now rather than reconstructed from memory months later.

## Shared-Source Lead Collisions

When lead/competitor data comes from a SHARED source scan (records carry `source_uid`/`point_uid` and the source has multiple subscribers in `collector/source_registry.json`), the same person can surface as a lead for several clients at once — and nothing else stops two clients from both reaching out to them.

- Surface the opportunity to every subscriber client normally (each report shows what its client's audience is doing).
- In each client's `INTERNAL_REPORT`, flag any lead whose source is shared: `also surfaced for N other client(s) via shared source {uid}` — the operator decides which client (if any) acts on it.
- Never let two clients' outreach both add the same shared-source person without that flag having appeared; the operator's assignment decision is the gate.
- The flag IS the protection — it is the whole design. Making a shared third-party source `exclusive` to "prevent" collisions is FORBIDDEN: it silently abandons the shared-scan model (duplicate scans, duplicate account footprint) to solve a problem this flag already solves, and it is a scope decision only the operator may make (02's scope rule). An agent that reasons its way to exclusivity from this section has misread it.

## Completion Checklist

Before claiming lead/competitor work is complete, verify:

- Stage 10 was loaded.
- Leads include direct and indirect need signals, not only explicit "I need a provider" posts.
- Competitors include direct, indirect, adjacent, attention, or authority competitors when relevant.
- The first lead/competitor pass used 10 scrolls per approved private data source when safe, or documented why it could not.
- Recurring daily runs used 5 scrolls per approved private data source by default, or documented the configured value.
- Detection happened during the same data collection pass unless a human-approved deeper pass was requested.
- Every report opportunity has a post/current URL when available.
- Every report opportunity has a context-aware copy-ready comment.
- Every copy button copies the comment only and does not imply auto-posting.
- Comments are value-first, same-language, short, natural, and not direct ads.
- One or two small natural imperfections are allowed only when they help the comment sound human and do not reduce trust.
- Logs were updated.
- No auto outreach, auto DM, credential/secret collection, or access-control/CAPTCHA bypass occurred. (Collecting publicly-rendered contact details under the operator's command is permitted per the collection-consent rule above; the send/act side still needs human approval.)
