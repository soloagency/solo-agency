# Stage 6 — Email Writing Standard

## Load Rule

Load before drafting ANY email (the daily run's draft step, or a Stage 10 follow-up). Its
dependency is the skill `playbooks/skills/email-writing/` (SKILL.md + **`weave.md`** (the rhetorical
engine) + `structures.md` + `channels.md` + `followup.md`) — each needs its own LOAD LEDGER per
`playbooks/LOAD_LEDGER_PROTOCOL.md`.

## Hard Gates For This Stage

- **Goal drives the pen.** The campaign's `goal_type` selects the email structure and CTA (see
  the skill's `structures.md`); the draft is written to satisfy the goal's `objective` + `cta`,
  and a positive reply fires the goal's `success_event` (a deal, via the rules engine).
- **Every personalized detail traces to a dossier hook with an `evidence_url`.** Pass those hooks
  as `hooks_used`; `crm_store.py draft write` rejects any that isn't an evidenced dossier hook,
  and Stage 9 greps for it.
- **Step-1 proof-of-life.** A step-1 draft needs a recent evidenced hook — recent activity is the
  reason an email exists. `draft write` REJECTS a hookless step-1 (`no_evidenced_hook`) unless the
  campaign explicitly opts into `no_hook_fallback: "generic_honest_opener"` (the default is
  `skip`). Bumps and reply drafts (step>1) are exempt — an existing conversation is its own
  justification.
- **Weave, don't list; facts vs conclusions.** Weave the Layer-B points so each earns a conclusion
  that advances the goal (`weave.md`) — multi-point weaving is the norm, not one flat hook. A *fact*
  traces to an `evidence_url`; a *conclusion* is your honest inference from it, never a new invented
  fact. Reference only public, professional signals — a peer who did their homework, not a
  surveillant (anti-creepy stance).
- **Never mention `writing_brief.do_not_mention`** (personal-life details).
- **Step-1 subject not `Re:`/`Fwd:`** (deceptive). Bumps thread and may keep `Re:` (truthful).
- **The draft never sends.** It lands in `pending_approval`; the operator approves in chat, then
  Stage 8 sends. This stage must not call `gmail_client.py send`.
- **No guessing, no invented facts, no fabricated proof.**

## Source Preservation Rule

Drafts are written through `crm_store.py draft write`. When any instruction here disagrees with
`docs/DESIGN.md`, `docs/DESIGN.md` wins.

## The run

1. For each enriched, due lead (Stage 4), load the skill and **weave** the email (`weave.md`) from
   four inputs: client profile (voice/offer), campaign goal (objective/CTA/proof), the contact
   dossier's ranked angles + hooks, and the step intent. Match depth to the dossier's Layer-B
   richness (RICH/MEDIUM/THIN), and package for the channel (`channels.md`).
2. Below `min_confidence` / no usable hooks → the campaign's `no_hook_fallback`. Default is
   **`skip`**: `draft write` rejects the hookless step-1 draft (`no_evidenced_hook`). Only a
   campaign that explicitly opts into `generic_honest_opener` gets the generic-but-honest opener
   (grounded only in license/roster facts, flagged `generic_opener`) — never a faked hook. Step>1
   bumps/replies are exempt.
3. Write it:
   ```sh
   python3 tools/crm_store.py --client-dir DIR draft write --contact <lead_id> --campaign <slug> --json \
     '{"step":1,"subject":"...","body_text":"...","hooks_used":[{"type":"new_listing","evidence_url":"https://..."}]}'
   ```
   The tool picks the sendbox (sticky for a bump; lowest-load rotation for step 1), sets the
   `confidence_band`, flags `generic_opener`/`bump_step` warnings, marks the hooks `used_in`, and
   stores the draft in `pending_approval`.

## Form — the adaptive weave

Plain-text. The shape is the skill's weave arc (`weave.md`): observation → conclusion → reframe into
the latent gap → defuse the objection → ROI anchor to their real numbers → release into the offer →
near-zero-friction CTA. **Length scales with the dossier's Layer-B richness** — RICH = full weave /
MEDIUM = tight arc / THIN = the short honest opener; the old "always 3–5 sentences" is retired to the
THIN mode only, not a universal cap. The weave is channel-agnostic — same argument on email vs
messenger, different wrapper (`channels.md`). The "load-bearing-detail test" still governs every
fact: delete the personalized sentence — if the email still stands, the detail was decoration. Every
fact must earn a conclusion (the cut rule); the writer is a peer who did their homework, never a
surveillant.

## Completion Gates

- Every drafted personalized detail is a dossier hook with an `evidence_url` (`hooks_used`).
- Step-1 subjects are not `Re:`/`Fwd:`.
- Drafts are in `pending_approval` only; none was sent.
- High-confidence vs review-carefully bands are set (drive the Approval Report grouping, Stage 14/15).

## Phase status

2C (this stage's tooling — `crm_store.py draft write/list` + the `email-writing` skill) is
**built**, and so is 2D — the Approval Report render + chat-approve handler
(`crm_store.py approval-report` / `approve`) and follow-up/reply (Stage 10). The send itself is
Stage 8 (Phase 1, built). Where a referenced row is still `status: planned`, follow DESIGN §22 R1.

When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
