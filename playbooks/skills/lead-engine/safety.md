# Lead Engine — Safety Envelope

Every gather loop runs inside this envelope. If a rule here conflicts with a
desire to gather more, the rule wins. When unsure, STOP and report what was
collected. Nothing here overrides Stage 10 or `08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md`;
it composes with them.

## Stop conditions (the loop ends when ANY is true)

1. **KPI reached.** The human's lead target (N) is met. If N was not given, ask
   once; if the human wants "as many as safe", use the volume budget below as the
   effective cap and report the number found.
2. **Diminishing returns.** Two consecutive deepen/next-source steps each add ~0
   new qualified leads → stop deepening this branch; either widen (new
   keyword/source) once, or stop.
3. **Safety trip (immediate stop).** Any of: a rate-limit / "you're doing this
   too fast" warning, a checkpoint/CAPTCHA, a session-expired / logged-out state,
   an empty or error response where data was expected several times in a row, or
   any platform warning. On a trip: stop the run, do NOT retry harder, report the
   trip reason to the human, and suggest resuming later. Check for these signals
   after EVERY job, before submitting the next one — never batch several jobs and
   check once at the end of a pass.
4. **Budget exhausted.** The per-run volume budget below is hit.

## Volume budget (per run, conservative defaults)

These are ceilings, not targets. Prefer the smallest depth that meets the KPI.

- **Pagination depth:** `inputs.max_pages` per capability call ≤ 8 by default
  (the collector hard-caps at 40). Raise only when a source is clearly
  productive and still under the run budget.
- **Sources per run:** ≤ ~8 groups/searches unless the human asked for a large,
  explicitly-approved sweep.
- **Recurring daily monitoring:** shallow — follow Stage 10 (5 scrolls / source /
  day) and `collector_config` (`max_scrolls_allowed`, `scroll_delay_seconds` ≈ 5s).
  The first pass for a new source may go to 10 scrolls per Stage 10.
- **The monitoring SEARCH pass** (Stage 10's daily two-pass scan, `tool source-keywords plan`):
  ≤ 3 terms per source per day, `max_scroll: 1`, `max_pages: 2`, with a recency window. These
  ceilings are unchanged by the per-kind quota / anchored-term quality gate (`recipes.md` Recipe A,
  `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md` keyword-bank section) — the gate only changes which
  terms fill those same slots, rejecting generic no-anchor terms before they burn a call on a page
  that was never going to return a qualified lead. Each term is one page load, so this is the line
  that decides the day's traffic shape, not its volume:
  keep the collector's random 5–10 s delay on every job (Pacing Rule), keep it strictly serial
  (never parallel tabs), vary the order of sources from day to day, and treat the FIRST platform
  warning as a stop for the whole account, not for one source. A first pass on a NEW source may
  take up to 8 terms at `max_scroll: 6` once —
  `max_pages` still stays at the default 8, because a source nobody has scanned has not yet earned
  the "clearly productive" exemption above.
- **Pacing:** rely on the collector's built-in pacing + the paginator's
  inter-page delay. Do not remove delays or hammer replay. If the human asks for
  "faster", explain the ban tradeoff rather than disabling safety.
- Prefer **one deeper pass on a productive source** over many shallow passes
  across many sources in a short window.
- **The Social Discovery Pass** (`playbooks/10_LEAD_COMPETITOR_DETECTION.md`, step 11C; canonical
  Facebook recipe: `recipes.md` Recipe A): one round-robin pass across Facebook, Instagram and X —
  build the job list in rounds of one Facebook job, then one Instagram job, then one X job, repeat;
  a platform not connected/logged-in/at-budget/tripped just loses its turn, never hands its slot to
  another platform, so the gap between two requests on the same platform never shrinks. Facebook:
  four call types, fixed order — FEED (`fb.search.posts`), PEOPLE (`fb.people.search`), GROUPS
  (`fb.groups.search`), IN-GROUP (`fb.group.search_posts`). FIRST RUN: 9 discovery calls (3 discovery
  terms, 3 feed searches, 3 people searches, 3 group searches) + up to 20 monitored groups per run
  (registry plan, `tool source-registry plan --max 20`) × 3 intent terms, `max_pages` ≤ 4. DAILY
  companion: 3 discovery calls (1 discovery term, 1 feed search, 1 people search, 1 group search) +
  up to 20 monitored groups per run (registry plan) × 2 intent terms, same `max_pages` ceiling. GROUP
  search calls stop early, before spending every discovery term, once this run has registered 20
  groups with `state: active` (Stop the group search at 20) — the group-searches figure above is a
  ceiling, not a target. The registry plan's "up to 20 monitored groups" stays sweep-first —
  never-scanned groups newest-registered first — until every active group has scanned at least once,
  then it switches to the performance order (Group sweep before optimisation, both rules in
  `playbooks/10_LEAD_COMPETITOR_DETECTION.md`).
  Instagram: four call
  types, fixed order — SEARCH (`ig.search.posts`), PEOPLE (`ig.people.search`), PROFILE DEPTH
  (`ig.profile.posts`), COMMENTS (`ig.post.comments`). FIRST RUN ≤ 12 calls total (3/3/3/3); DAILY
  ≤ 4 calls total (1/1/1/1). X: four call types, fixed order — SEARCH LATEST (`x.search.posts`),
  PEOPLE (`x.people.search`), PROFILE DEPTH (`x.profile.posts`), REPLIES (`x.post.replies`). FIRST
  RUN ≤ 12 calls total (3/3/3/3); DAILY ≤ 4 calls total (1/1/1/1). These are ceilings, not targets,
  exactly like every other row here — and per Stop condition 3 above, the agent reads every job's
  result for a trip signal before submitting the next job on that same platform, not just at the end
  of the pass; a trip removes only that platform from the rotation for the day, the other two
  continue. Write actions on Instagram/X (react, comment, message, like, reply, publish, DM) are out
  of scope for this pass.

**Pacing Rule (all platforms, both tiers).** The only spacing between requests is the collector's
own random delay: every discovery job carries `"pacing": {"min_delay_seconds": 5,
"max_delay_seconds": 10}` — a random 5–10 second pause before each page, scroll or request inside
the job — plus the extension's poll interval between jobs. The agent submits the next job as soon as
it has read the previous job's result for a trip signal; it never sleeps, waits, or schedules its own
gaps between jobs, never spreads a pass over hours, and never lowers the delay below 5 s. Round-robin
order (Facebook → Instagram → X) and the per-platform ceilings are unchanged; with this pacing a
FIRST RUN on all three platforms takes about 30–45 minutes of collector time (ETA Rule), not hours.
What protects the account is the trip rule, not slowness: the first checkpoint, rate-limit warning
or logged-out signal stops that platform for the day.

## The join boundary (human-in-loop, never automatic)

- Joining a group, following, or any membership/subscription change is a WRITE
  action and is OUT OF SCOPE for this read-only collector.
- When a recipe needs a group the human is not in: PRESENT the group(s) + a
  one-line reason each and let the human join in their own session. Only scan
  groups the human is already a member of, plus publicly-viewable groups.
- Never automate join, never chain "search groups → auto-join → scan".
- Private groups the account already belongs to (`viewer_join_state == "MEMBER"`) are readable and
  may be scanned; groups it is not in are never joined or requested.

## Outreach boundary (Stage 10)

No auto-DM, auto-comment, follow, or any outreach. Leads are review signals for
the human. Suggested comments are copy-to-clipboard only (Stage 10 report
contract). This loop's job ends at "here are qualified leads for you to review".

This holds even once the human approves: **this loop never becomes a sender.**
Approved agency outreach is executed elsewhere — OutreachCRM Stages 16/17, from
the operator's own brand and accounts, with its own per-message approval and
per-account caps. Keeping the scanner read-only is what stops a monitoring pass
from quietly turning into a mass-messaging pass.

## Privacy / data-minimization

- Data collection + analysis is consented by the operator's own setup and
  command (Stage 10 collection-consent rule): the agent may read, extract, and
  combine the public info the operator directs it to research — including a
  prospect's publicly-rendered contact details (email/phone) for lead-finding and
  email personalization. `contact_extract.js` adds these as a `contacts` field
  from already-captured public visible text + `mailto:`/`tel:` anchors.
- Keep provenance: record which public page each detail came from so the human
  can verify. Prefer safe summaries + source URLs in the report.
- **Absolute bans (never, even under command):**
  - anyone's credentials/secrets — passwords, cookies, OTPs, tokens,
    session/auth data, API keys;
  - bypassing access controls or CAPTCHAs — read only what the operator's own
    logged-in session already renders;
  - opening/expanding hidden or private "contact info" sections;
  - the send/act side — auto-DM, auto-comment, auto-follow, any outreach — which
    always needs explicit human approval.

## Reporting the stop (always)

Whatever ends the run, tell the human plainly:

```text
Collected {K}/{N} leads. Stopped: {KPI met | diminishing returns | SAFETY: <which trip> | budget}.
Coverage: {sources}, keywords {kw}, depth {pages}. To go further: {widen/deepen suggestion, or "resume later" after a safety trip}.
```
