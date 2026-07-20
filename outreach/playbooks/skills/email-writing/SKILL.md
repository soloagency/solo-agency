---
name: outreachcrm-email-writing
description: >-
  Use to WRITE a cold-email draft (Stage 6) from a campaign goal + a contact's enriched dossier +
  the step intent. The campaign's goal_type drives the structure (book a meeting / get a reply /
  direct sale / reactivation). Every personalized detail must trace to a dossier hook with an
  evidence_url. Writes the draft to pending_approval via tool crm-store — it never sends. Loaded
  after Stage 6 / from the daily run's draft step and Stage 10 follow-up.
---

# Email Writing (Stage 6 skill)

This is an INDEX. The engine is **`weave.md`** — the rhetorical method (the four pillars, the
fact→conclusion move, tension→release, the arc, the 3 data-richness modes). **Load it whenever you
draft.** Then load `structures.md` (goal_type → emotional landing + release move) for a step-1,
`channels.md` (email vs messenger packaging — same weave, different wrapper) for the channel you're
sending on, and `followup.md` (bumps + reply drafts) for Stage 10 follow-ups.

## Hard gates

- **Every personalized detail traces to a dossier hook with an `evidence_url`.** You may only
  reference facts present in `contact.enrichment.hooks`; pass those exact hooks as `hooks_used`.
  `tool crm-store draft write` rejects a hook that isn't an evidenced dossier hook.
- **Never mention anything in `writing_brief.do_not_mention`** (personal-life details live there).
- **Step-1 subject must not begin `Re:`/`Fwd:`** (deceptive, CAN-SPAM). Enforced by the tool.
- **The draft never sends.** It goes to `pending_approval`; the operator approves in chat, then
  the send engine (Stage 8) runs. Do not call `tool gmail send` from this stage.
- **No guessing / no invented facts / no fabricated proof.** Proof points come from the campaign
  goal (each with its own evidence). If a claim has no source, don't make it.
- **Facts vs conclusions.** A *fact* must trace to a dossier hook with an `evidence_url` (what you
  observed). A *conclusion* is your honest INFERENCE from that fact ("people are already passing
  your content along" from an evidenced share count) — stated as inference, never invented as a new
  fact. Reason freely from evidenced facts; never manufacture one. (See `weave.md`.)
- **No em dash (`—`).** Never use `—` in a draft, on any channel, in any language; it reads as
  machine-written and costs trust on the first line. Use a comma, colon, period, or parentheses
  instead; ranges use "to". Hyphens in compounds (30-day, first-time) are fine. (`weave.md` → House Style.)

## Compose from four inputs

1. **Client profile** — voice, offer, compliance (physical address + opt-out are added by the
   send engine's MIME + footer, not by you).
2. **Campaign goal** — `objective`, `offer`, `value_proposition`, `proof_points` (evidenced),
   `cta`. `goal_type` picks the structure (`structures.md`).
3. **Contact dossier** — `writing_brief.ranked_angles` (already ranked by freshness × goal-fit ×
   confidence). **Weave** the angles — each fact earning a conclusion (`weave.md`), not one hook read
   out flat; lead with the load-bearing signal and demote vanity metrics. A step-1 draft needs a
   recent evidenced hook: with
   no usable hooks (`confidence_band` is `fallback`), `draft write` rejects it
   (`no_evidenced_hook`) unless the campaign opts into `no_hook_fallback: "generic_honest_opener"`
   (the default is `skip`). When it opts in, the fallback is a generic-but-honest opener grounded
   only in license/roster facts — don't fake a hook.
3. **Step intent** — step 1 = hook + offer + one CTA; bumps carry NEW value, never "just
   following up" (`followup.md`).

## The load-bearing-detail test

Delete the personalized sentence. If the email still stands, the detail was decoration — the
reader will smell it as scraped. A good hook is the *reason the email exists*: not "congrats on
your new listing, btw I make videos", but "your Main St listing has been up 40 days with
photo-only — video tours are closing comparable Alabaster homes in 18 days."

This is the weave's **cut rule**: every fact must earn a conclusion that advances the goal, or it
gets dropped (`weave.md`, Pillar 1). It is *not* "use fewer facts" — you may weave several — it is
"every fact does rhetorical work." That is also the **anti-creepy** line: a peer who did their
homework references public professional signals that each do work; a surveillant reads out a dated,
itemized list. Reference only public, professional signals; never personal-life details
(`do_not_mention`). The full stance is stated verbatim in `weave.md`.

## Form — the adaptive weave (not a fixed length)

Plain text (Phase-1 `plain_text_mode`). The shape is the `weave.md` arc — observation → conclusion →
reframe into the latent gap → defuse the likely objection → ROI anchor to their real numbers →
release into the offer → near-zero-friction CTA — and its **length scales with the dossier's Layer-B
richness**, not a fixed sentence count:

- **RICH** (≥3 solid Layer-B points) → full weave, multiple fact→conclusion moves.
- **MEDIUM** (1–2 points) → tight arc: one strong observation → one reframe → offer.
- **THIN** (1 weak point / fallback) → the short honest opener. **This is the only place the old
  "3–5 sentences" guidance applies** — it is a THIN-data mode, not a universal law. (The prior "3–5
  sentences / longer reads as machine-made" rule contradicted the operator's proven method and is
  retired as a universal.)

Multi-point weaving is the norm. What still reads as machine-made is a *flat list of facts* (a
scrape), not length — so every fact must earn a conclusion (the cut rule above). The CTA stays
near-zero-friction (reply a word, not "book a 30-min call"). The wrapper — subject, length ceiling,
footer — adapts to the channel (`channels.md`).

## Write the draft

```sh
<bridge> tool crm-store --client-dir DIR draft write --contact <lead_id> --campaign <slug> --json \
  '{"step":1,"subject":"Quick idea for your Main St listing",
    "body_text":"Hi Susan, I saw 123 Main St went up last week...","tracking":"plain_text",
    "hooks_used":[{"type":"new_listing","evidence_url":"https://zillow.com/..."}]}'
```

`draft write` picks the sendbox (sticky sender for a bump; lowest-load rotation among the
campaign's healthy boxes for step 1), sets `confidence_band` from the dossier, flags warnings
(`generic_opener`, `bump_step`), records `used_in` on the hooks so another campaign won't reuse
them, and stores the draft in `pending_approval`. It returns the `draft_id` + `sendbox` + band.

The drafts you produce become cards in the Approval Report (Stage 14/15); nothing sends until the
operator approves.

When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
