# Follow-ups: the bump engine + reply drafts (Stage 10)

Follow-up is where most replies come from. So a bump is **not a reminder** — it is a fresh,
self-contained attempt from a NEW angle, held to the same bar as step 1. "Just following up" /
"bumping this to the top of your inbox" / "circling back" is banned: each touch has to earn a reply
on its own.

Load `weave.md` (the engine: every bump is a mini-weave) and `structures.md` (the release move per
`goal_type`) alongside this. House Style applies to every bump (no em dash `—`; `weave.md`) because a
bump is a sent message.

---

## The blind rule (read first)

Phase-1 has no open/click signal — you do NOT know whether the person even saw step 1. Assume they
might not have. Therefore every bump must:

- **Stand alone.** A fresh reader must be able to reply to bump 3 without having read step 1. Give
  ONE quick line that re-anchors who you are and the value, then move; never lean on "as I mentioned"
  or "following up on my last email" as if they remember it.
- **Never imply surveillance.** No "I noticed you haven't replied", no "over the past two weeks I've
  been watching…". A bump is a peer's second courtesy, not a stakeout (anti-creepy stance, `weave.md`).

## Each bump is a mini-weave, not a fact-dump

"One new thing" is necessary but not sufficient — a dumped fact does not persuade. Every bump keeps
the weave shape, just shorter than step 1:

> signal → conclusion → one small tension → a lower-friction ask.

A bump that states a new fact and then stops gets the same **cut** as step 1's list rule: if the fact
earns no conclusion that moves toward a reply, it does not belong.

## The 4-door ladder (why a sequence out-pulls a single email)

A sequence is not one pitch sent four times. It is four DIFFERENT doors into the same value. Match the
door to the step; rotate, never repeat the previous door.

| Step | ~gap | The door | The move |
|---|---|---|---|
| **1** | cold | Observation → the gap → offer | The full weave (`weave.md`). |
| **2** | +4d | **A NEW proof or a fresh signal** | Stage 4 micro-refreshes first. If a new signal appeared since step 1 (a fresh post/listing/video), LEAD with it: "you just posted X, that is the pattern I meant." If none, bring a concrete micro-sample or one specific result from a peer in their trade. Show, do not remind. |
| **3** | +5d | **A different emotional door** | Shift the angle away from step 2. Pick ONE, never stack: the quiet cost of inaction; a same-trade peer's outcome they cannot wave off; or defuse the specific objection they are probably sitting on. |
| **4** | +7d | **The breakup (the strongest single touch)** | Honest, warm, zero pressure. Name the specific gap once, leave the door and the link, give the easy out. Breakups reliably out-pull the middle bumps: they drop the pressure and trigger loss-aversion at the same time. |

**Picking the step-3 door:** choose the one FURTHEST from what steps 1–2 already spent. If step 1
already used the cost/bottleneck angle, do not reopen it at step 3 — go to objection-defuse or
peer-outcome instead. Peer-outcome needs a real evidenced peer result; if you have none, default to
defusing the single objection the lead is most likely sitting on.

**Rotate the message bank, not just the data.** The ladder's doors are the emotional FRAME; the
`goal.message_bank` (`05_CAMPAIGN_MANAGEMENT.md` §1c) is the SUBSTANCE. Each touch pulls 1–2 bank
messages the earlier touches did not use, mixed with its data point. Data alone rotated across 4 bumps
still reads as one color (the same underlying pitch); a *different bank message per touch* is what
actually changes the color. 1–2 per touch, never a recital. Treat the bank as a menu, not a checklist:
**skip any message the lead's data contradicts**, and never force one just to fill the rotation. A
positioning or ROI fact (tenure, price band, a "$X listing") may ride along with a touch's primary
data point to anchor the one-client math without counting as a second color.

## The shrinking ask

Lower the friction as the sequence goes: step 1 "worth a look?" → step 2 "want a 2-minute sample?" →
step 3 "want me to map your next two weeks? no commitment" → step 4 "just say the word and I'll stop."
The tiny, near-zero ask is exactly why the breakup earns
replies. Never escalate the ask over a silent sequence (no "let's book 30 minutes" at step 3).

## Enrichment happens ONCE, then you reserve (not per bump)

Do NOT re-run enrichment before every bump. That piles collector work onto every in-flight sequence
while the daily SEND ceiling stays fixed (sendboxes × quota), so the enrichment load outruns the sends
and clogs the single collector. Instead:

- **Enrich richly once, at entry** — the enrich skill already gathers MANY Layer-B points, uncapped.
  Reserve the secondary ones across the sequence (see below). One enrichment feeds all four touches.
- **The message bank carries the rotation**, so a bump does not NEED new data to change color: a
  reserved data point × a fresh bank message is already a new touch.
- **Micro-refresh is opportunistic, never mandatory.** Re-touch a lead's sources only when its
  reserved points are used up AND the lead is worth it AND the collector has spare capacity — gated by
  the send budget, never blocking a bump. If nothing fresh comes back, the bump still works on reserved
  data + the bank. **Retire a stale hook** (a sold listing is not "active", a view count moves on)
  whenever you do refresh. Every referenced fact still carries its `evidence_url`.

**Reserve secondary signals for the bumps.** A step-1 weave that enumerates every number ("241K here,
99K there, 89K on another") leaves the bumps nothing fresh to open with. Step 1 leads with the ONE
load-bearing signal; each bump then gives a *different* secondary signal its own identity and a new
conclusion, so the number feels new even though it was on the page all along.

## The draft is a real in-thread reply

`crm_store.py draft write` with `step > 1` uses the sticky sendbox and threads off the prior
`rfc_message_id`; the subject may keep a truthful `Re:`. Threading is held by the `rfc_message_id`,
not the subject string, so if the original subject carries a banned em dash, **re-render the aside
em-dash-free** ("Glennda (241K views)") and the `Re:` stays truthful. After the breakup step, the
sequence ends for that contact (`followups due` never offers an exhausted sequence again).

---

## Annotated example — Angela's steps 2–4 (companion to `weave.md` Reference A)

Step 1 is the Angela weave in `weave.md`. Here are her next three doors. Bracketed labels are the
door/beat, not part of the email. Every body is em-dash-free (House Style).

**Step 2 (+4d) — the NEW-proof door.** `Re:` thread; self-anchor in one line, then lead with a fresh
signal, and drop the friction to a free sample:
> Quick one, then I'll leave it with you. I put together a 30-day content plan for your Scottsdale
> listings last week.  `[self-anchor in one line — the blind rule]`  Since then your newer Reel has
> kept climbing, right alongside the one that pulled 52 shares.  `[NEW signal, not step 1's]`  That is
> the whole point: the market already stops for your content, it just is not compounding into an
> audience yet.  `[conclusion + the same one-off-vs-system tension, tighter]`  I can send you a
> 2-minute sample built from one of your current listings, so you see the output before deciding
> anything. Want me to?  `[lower-friction ask: a free sample, not "read the plan"]`

**Step 3 (+5d) — a different emotional door (cost of inaction).** Not more proof; a new angle:
> One more angle, then I'll stop.  `[shrinking-ask signal, sets up the easy out]`  The quiet cost of
> content this good sitting as one-off posts is timing: every month, a Scottsdale seller lists with
> the agent whose name they have seen all season, which is usually the most present one, not
> necessarily the best one.  `[cost-of-inaction door, a different tension than step 2]`  A steady
> system is how you become that present name, without adding to your week.  `[release, same offer, new
> framing]`  Worth 5 minutes?  `[low ask]`

**Step 4 (+7d) — the breakup.** Warm, zero pressure, door left open:
> I'll leave this here, Angela. You clearly do not need convincing that video works, your Reels prove
> it.  `[concede honestly, no pressure]`  If turning those one-off hits into a steady system ever
> moves up your list, the 30-day plan is here whenever you want it: [URL].  `[door + link, gap named
> once]`  Either way, good luck with the Scottsdale season, and just say the word if you ever want
> it.  `[loss-aversion is implicit; the easy out is explicit]`

**Why it works:** four distinct doors (viral-proof → free sample → cost of waiting → the graceful
exit), each self-contained, each a short mini-weave, and the ask shrinks every step. Not once does it
say "just following up," and not once does it imply she was being watched.

---

## Reply drafts (they replied)

A reply FROZE the sequence (`sync` sets `sequence_state: frozen`); after triage + rules it may have
created a deal. Draft the human reply that moves the conversation to the goal, and deliver the value
**in the reply itself**, not deferred to a call:

- `reply_positive` → confirm warmly + the ONE tiny next step (send the sample, propose a single
  specific 15-min slot). Do not re-pitch what they already said yes to.
- `reply_question` → answer the actual question plainly and completely, then the next step. A reply
  that answers earns more trust than one that dodges to a meeting.
- `reply_objection` → address the specific objection with evidence, low pressure. Concede what is
  fair; never argue.

Speed matters: a same-day reply beats a next-day one, so hot replies surface in the Today View
(Stage 14). These draft to `pending_approval` too.

## What never changes

- Every referenced detail traces to a dossier hook with an `evidence_url`.
- `do_not_mention` applies; House Style applies (no em dash `—`, `weave.md`) — a bump is sent output.
- Nothing sends without operator approval. A reply freezes the sequence; a `negative` / `remove_intent`
  reply routes to suppression (Stage 10 §2), never another bump.

When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
