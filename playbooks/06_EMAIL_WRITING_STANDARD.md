# Stage 6 — Email Writing Standard

## Load Rule

Load before drafting ANY email (the daily run's draft step, or a Stage 10 follow-up). Its
dependency is the skill `playbooks/skills/email-writing/` (SKILL.md + `structures.md` +
`followup.md`) — each needs its own LOAD LEDGER per `playbooks/LOAD_LEDGER_PROTOCOL.md`.

## Hard Gates For This Stage

- **Goal drives the pen.** The campaign's `goal_type` selects the email structure and CTA (see
  the skill's `structures.md`); the draft is written to satisfy the goal's `objective` + `cta`,
  and a positive reply fires the goal's `success_event` (a deal, via the rules engine).
- **Every personalized detail traces to a dossier hook with an `evidence_url`.** Pass those hooks
  as `hooks_used`; `crm_store.py draft write` rejects any that isn't an evidenced dossier hook,
  and Stage 9 greps for it.
- **Never mention `writing_brief.do_not_mention`** (personal-life details).
- **Step-1 subject not `Re:`/`Fwd:`** (deceptive). Bumps thread and may keep `Re:` (truthful).
- **The draft never sends.** It lands in `pending_approval`; the operator approves in chat, then
  Stage 8 sends. This stage must not call `gmail_client.py send`.
- **No guessing, no invented facts, no fabricated proof.**

## Source Preservation Rule

Drafts are written through `crm_store.py draft write`. When any instruction here disagrees with
`docs/DESIGN.md`, `docs/DESIGN.md` wins.

## The run

1. For each enriched, due lead (Stage 4), load the skill and write the email from four inputs:
   client profile (voice/offer), campaign goal (objective/CTA/proof), the contact dossier's
   ranked angles + hooks, and the step intent.
2. Below `min_confidence` / no usable hooks → the campaign's `no_hook_fallback` (generic honest
   opener grounded only in license/roster facts, or skip) — never a faked hook.
3. Write it:
   ```sh
   python3 tools/crm_store.py --client-dir DIR draft write --contact <lead_id> --campaign <slug> --json \
     '{"step":1,"subject":"...","body_text":"...","hooks_used":[{"type":"new_listing","evidence_url":"https://..."}]}'
   ```
   The tool picks the sendbox (sticky for a bump; lowest-load rotation for step 1), sets the
   `confidence_band`, flags `generic_opener`/`bump_step` warnings, marks the hooks `used_in`, and
   stores the draft in `pending_approval`.

## Form

Short, plain-text, one load-bearing observation + one evidenced value line + a near-zero-friction
CTA. The "load-bearing-detail test": delete the personalized sentence — if the email still stands,
the detail was decoration.

## Completion Gates

- Every drafted personalized detail is a dossier hook with an `evidence_url` (`hooks_used`).
- Step-1 subjects are not `Re:`/`Fwd:`.
- Drafts are in `pending_approval` only; none was sent.
- High-confidence vs review-carefully bands are set (drive the Approval Report grouping, Stage 14/15).

## Phase status

2C (this stage's tooling — `crm_store.py draft write/list` + the `email-writing` skill) is
**built**. The Approval Report render + chat-approve handler + follow-up (Stage 10) ship in 2D;
where still `status: planned`, follow DESIGN §22 R1.

When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
