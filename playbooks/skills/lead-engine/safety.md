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
   trip reason to the human, and suggest resuming later.
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
- **Pacing:** rely on the collector's built-in pacing + the paginator's
  inter-page delay. Do not remove delays or hammer replay. If the human asks for
  "faster", explain the ban tradeoff rather than disabling safety.
- Prefer **one deeper pass on a productive source** over many shallow passes
  across many sources in a short window.

## The join boundary (human-in-loop, never automatic)

- Joining a group, following, or any membership/subscription change is a WRITE
  action and is OUT OF SCOPE for this read-only collector.
- When a recipe needs a group the human is not in: PRESENT the group(s) + a
  one-line reason each and let the human join in their own session. Only scan
  groups the human is already a member of, plus publicly-viewable groups.
- Never automate join, never chain "search groups → auto-join → scan".

## Outreach boundary (Stage 10)

No auto-DM, auto-comment, follow, or any outreach. Leads are review signals for
the human. Suggested comments are copy-to-clipboard only (Stage 10 report
contract). This loop's job ends at "here are qualified leads for you to review".

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
