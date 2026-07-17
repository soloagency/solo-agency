---
name: outreachcrm-email-writing
description: >-
  Use to WRITE a cold-email draft (Stage 6) from a campaign goal + a contact's enriched dossier +
  the step intent. The campaign's goal_type drives the structure (book a meeting / get a reply /
  direct sale / reactivation). Every personalized detail must trace to a dossier hook with an
  evidence_url. Writes the draft to pending_approval via crm_store.py — it never sends. Loaded
  after Stage 6 / from the daily run's draft step and Stage 10 follow-up.
---

# Email Writing (Stage 6 skill)

This is an INDEX. Load `structures.md` (goal_type → email shape) when you draft a step-1, and
`followup.md` (bumps + reply drafts) for Stage 10 follow-ups.

## Hard gates

- **Every personalized detail traces to a dossier hook with an `evidence_url`.** You may only
  reference facts present in `contact.enrichment.hooks`; pass those exact hooks as `hooks_used`.
  `crm_store.py draft write` rejects a hook that isn't an evidenced dossier hook.
- **Never mention anything in `writing_brief.do_not_mention`** (personal-life details live there).
- **Step-1 subject must not begin `Re:`/`Fwd:`** (deceptive, CAN-SPAM). Enforced by the tool.
- **The draft never sends.** It goes to `pending_approval`; the operator approves in chat, then
  the send engine (Stage 8) runs. Do not call `gmail_client.py send` from this stage.
- **No guessing / no invented facts / no fabricated proof.** Proof points come from the campaign
  goal (each with its own evidence). If a claim has no source, don't make it.

## Compose from four inputs

1. **Client profile** — voice, offer, compliance (physical address + opt-out are added by the
   send engine's MIME + footer, not by you).
2. **Campaign goal** — `objective`, `offer`, `value_proposition`, `proof_points` (evidenced),
   `cta`. `goal_type` picks the structure (`structures.md`).
3. **Contact dossier** — `writing_brief.ranked_angles` (already ranked by freshness × goal-fit ×
   confidence). Open with the top angle's hook. A step-1 draft needs a recent evidenced hook: with
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

## Form

Short and deliberately plain: 3–5 sentences, plain text (Phase-1 `plain_text_mode`), one
load-bearing observation + one evidenced value line + one near-zero-friction CTA (reply a word,
not "book a 30-min call"). Longer + "hyper-personalized" reads as machine-made.

## Write the draft

```sh
python3 tools/crm_store.py --client-dir DIR draft write --contact <lead_id> --campaign <slug> --json \
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
