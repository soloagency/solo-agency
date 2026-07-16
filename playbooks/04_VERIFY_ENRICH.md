# Stage 4 — Verify & Enrich

## Load Rule

Load before ANY enrichment: the daily run's "load new pipeline → verify → enrich" step, or a
follow-up's micro-refresh. Its dependency is the skill `playbooks/skills/email-verify-enrich/`
(SKILL.md + `channel_reality.md` + `etiquette.md`) — each needs its own LOAD LEDGER per
`playbooks/LOAD_LEDGER_PROTOCOL.md` before you act.

## Hard Gates For This Stage

- **Every usable hook carries an `evidence_url`.** Enforced by `crm_store.py enrich write` (it
  drops a hook with no source) and re-checked by Stage 9. A detail with no source is never written
  and never sent.
- **No guessing.** Enrichment never fabricates an email address (MVP decision). Store only real
  found addresses (`source: enrich`); if none is found, set `mark_email_not_found`.
- **No personal-life mining.** `public_business` signals are hooks; `personal` signals go to
  `do_not_mention` only (see the skill's `etiquette.md`).
- **Inherit before you enrich.** The dossier belongs to the contact and is reused across that
  client's campaigns. Always check `crm_store.py enrich status` first and act on its verdict.
- **Read-only, logged-out.** WebSearch/WebFetch (+ browser tool only where `channel_reality.md`
  says). Never log into an account; Facebook/LinkedIn store URL only (Phase-4 Local Collector).

## Source Preservation Rule

The dossier and `contact.enrichment` are written through `crm_store.py` (a `crm/` mutation).
Do not hand-edit them. When any instruction here disagrees with `docs/DESIGN.md`, `docs/DESIGN.md`
wins.

## The run

1. Get the batch: `crm_store.py enrich due --campaign <slug> --limit N` returns queued leads that
   need enrich or refresh (already-fresh ones are skipped — that is cross-campaign inheritance).
2. Load the skill (`email-verify-enrich`) and run its two-tier flow per lead: Tier 1 verify
   (still active? profile URLs), Tier 2 hooks (evidenced, `public_business` only), distill a
   `writing_brief`.
3. Write it: `crm_store.py enrich write --contact <lead_id> --campaign <slug> --json '<dossier>'`.
   It stores the full dossier under `campaigns/{slug}/queue/enriched/YYYY-MM-DD/` and a distilled
   copy into `contact.enrichment`, and returns `usable_hooks` / `confidence_band` / `problems`.
4. No-hook leads: set `mark_no_hook` and let the campaign's `no_hook_fallback` decide (generic
   honest opener grounded in license/roster facts, or skip). Inactive leads: `still_active:
   inactive`, stop — do not draft.

## TTL, inheritance, negative cache (all in `enrich status`)

- Identity (still-active, company, role, profile URLs) is durable — TTL ~90 days, reused as-is by
  other campaigns.
- Hooks are fresh — TTL ~10 days; a stale-hook contact returns `needs: refresh` (revisit known
  URLs), not a full re-enrich.
- `email_not_found` / `no_verifiable_hook` are inherited negative caches so a second campaign does
  not re-burn the same dead end within its retry window.

## Completion Gates

- Every drafted personalized detail (Stage 6) traces to a stored hook with an `evidence_url`.
- No guessed email exists; found emails are `source: enrich`.
- No `personal` hook is a usable hook (it is in `do_not_mention`).
- Freshness respected: step-1 drafts use in-TTL hooks; follow-ups micro-refreshed and stale hooks
  invalidated.

## Phase status

The enrich storage/TTL/validation tooling (`crm_store.py enrich`, 2B) is **built**. The web
verify/enrich itself is agent behavior driven by the `email-verify-enrich` skill. Downstream Stage
6 (email writing) ships in 2C and Stage 10 (follow-up) in 2D; where still `status: planned`, follow
DESIGN §22 R1.

When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
