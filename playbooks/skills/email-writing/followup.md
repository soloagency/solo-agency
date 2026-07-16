# Follow-ups: bumps and reply drafts (Stage 10)

Follow-up is where most replies come from — but only if each touch adds NEW value. Never send
"just following up" / "bumping this to the top of your inbox".

## Silent-lead bumps (step > 1)
- The contact did NOT reply; a `gap_days` has elapsed. Before drafting, Stage 4 micro-refreshes
  the person's 1–2 best sources — so you either have a **fresh** hook (a new listing since the
  last touch is a great bump) or you must **retire a stale one** (a listing that sold cannot be
  referenced as active).
- Each bump carries one NEW thing: a different insight, a small sample, a new proof point — not a
  repeat of step 1. Rotate the angle (`structures.md`).
- The draft is a real in-thread reply: `crm_store.py draft write` with `step > 1` uses the sticky
  sendbox and threads off the prior message; the subject may keep `Re:` (truthful now).
- The final step is the breakup (see `structures.md`): easy out, door left open. Then the
  sequence ends for that contact.

## Reply drafts (they replied)
- A reply FROZE the sequence (sync sets `sequence_state: frozen`) and, after triage + rules, may
  have created a deal. Draft the human reply that moves the conversation to the goal:
  - `reply_positive` → confirm + the tiny next step (send the sample, propose the 15-min slot).
  - `reply_question` → answer plainly, then the next step. Deliver value in the reply itself.
  - `reply_objection` → address the specific objection with evidence, low pressure.
- These are drafted to `pending_approval` too — the operator approves before anything is sent.
  Speed matters: a same-day reply beats a next-day one, so surface hot replies in the Today View.

## What never changes
- Every referenced detail still needs a dossier hook with an `evidence_url`.
- `do_not_mention` still applies.
- Nothing sends without operator approval.

When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
