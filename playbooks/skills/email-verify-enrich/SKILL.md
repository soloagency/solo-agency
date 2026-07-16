---
name: outreachcrm-email-verify-enrich
description: >-
  Use during a daily run's enrich step (Stage 4) to VERIFY a contact is still active and
  ENRICH them with fresh, evidenced hooks for a personalized cold email. Two tiers: a cheap
  verify pass (still in the business? gather profile URLs) then a hook pass (a new listing, a
  recent post, a review — each with a source URL). Writes the dossier via crm_store.py. Never
  guesses an email address, never mines personal life. Loaded after Stage 4 / from the daily
  run's "load new pipeline -> enrich" step.
---

# Email Verify & Enrich (Stage 4 skill)

This is an INDEX. Load the module you need when you reach its step:
`channel_reality.md` (what is actually readable, per source) and `etiquette.md` (what is fair
game vs off-limits). The dossier schema and storage are Stage 7 + `crm_store.py`.

## Hard gates

- **Every usable hook needs an `evidence_url`.** A detail with no source is dropped — Stage 6
  may not write it, and Stage 9 greps for it. `crm_store.py enrich write` enforces this.
- **No guessing.** Never fabricate an email address. If you find a real address (on a website,
  a license record, a roster) store it as a found identity (`source: enrich`). If you find
  none, mark `mark_email_not_found` — do not invent a `first.last@domain` guess (MVP decision).
- **No personal-life mining.** `public_business` signals only (listings, work posts, reviews,
  awards, market views). Family/health/vacation/children → `sensitivity: personal`, which goes
  to `do_not_mention`, never to email copy. See `etiquette.md`.
- **Read-only.** Use WebSearch / WebFetch (and a browser tool only where the readability table
  says so). Do not log into anyone's account. Facebook/LinkedIn logged-out = store the URL only
  (Phase 4 Local Collector reads those later).
- **Inherit, don't re-burn.** The dossier belongs to the CONTACT and is reused across that
  client's campaigns. Check `crm_store.py enrich status` first; only enrich/refresh when it says
  so. A `no_verifiable_hook` / `email_not_found` negative cache is inherited — respect its window.

## The two-tier flow

Run over the leads `crm_store.py enrich due --campaign <slug>` returns.

**Tier 1 — Verify (cheap).** Confirm the person is still active and collect profile URLs. Order
of reliable sources (adapt per industry): official license/registry lookup → brokerage/employer
roster (email domain is a hint) → Zillow/realtor.com search snippet → Google Business Profile.
If the person is clearly inactive/left the field, set `still_active: inactive` and STOP — do not
run Tier 2 (don't spend the hook pass on a dead lead).

**Tier 2 — Profile & hooks.** Visit the URLs Tier 1 found (see `channel_reality.md` for what
each actually yields). Extract 1–3 hooks, each with `type`, `summary`, `evidence_url`,
`observed_date`, `confidence`, and `analysis.sensitivity`. Analyze the 3–5 latest readable posts
where possible. Distill a `writing_brief`: a one-liner, ranked angles (freshness × goal-fit ×
confidence), a `do_not_mention` list, and a `personalization_confidence` (drives the band:
≥0.7 high, 0.4–0.7 review_carefully, <0.4 fallback).

## Freshness at write time

Before drafting **step 1**, hooks must be within TTL (`enrich status` handles this). Before a
**follow-up** (Stage 10), micro-refresh the person's 1–2 best sources: a fresh event is a great
bump hook, and — critically — **a stale hook must be invalidated** (a listing that sold must not
be referenced as active). Re-run `enrich write` with the refreshed hooks.

## Writing the dossier

```sh
python3 tools/crm_store.py --client-dir DIR enrich write --contact <lead_id> --campaign <slug> --json '<dossier>'
```

Dossier shape (DESIGN §9.2):
```json
{"identity":{"still_active":"confirmed|inactive|unknown","current_company":"","role":"",
   "profiles":{"zillow":"","website":"","facebook":"","instagram":"","gbp":""},
   "evidence":[{"fact":"","url":"","retrieved_at":""}],
   "channels_found":{"emails":["only REAL found addresses"],"phones":[]}},
 "context":{"market":"","volume_signals":"","specialty":"","content_style":""},
 "hooks":[{"type":"new_listing|social_post|review|award|market_view|website_update",
   "summary":"","analysis":{"topic":"","angle":"","sensitivity":"public_business|personal"},
   "evidence_url":"https://...","observed_date":"YYYY-MM-DD","confidence":0.0}],
 "writing_brief":{"one_liner":"","ranked_angles":[],"do_not_mention":[],"personalization_confidence":0.0},
 "mark_email_not_found": false, "mark_no_hook": false}
```
`enrich write` stores the full dossier under `campaigns/{slug}/queue/enriched/YYYY-MM-DD/` and a
distilled copy into `contact.enrichment` (inherited by other campaigns). It returns
`usable_hooks`, `confidence_band`, and any `problems` (e.g. a hook it dropped for missing evidence).

## No-hook branch

Realistic hit-rate for a deeply-personalized hook is ~30–50% (see `channel_reality.md`). When no
verifiable hook is found, set `mark_no_hook: true` and let the campaign's `no_hook_fallback`
decide: a generic-but-honest opener grounded only in license/roster facts, or skip the contact.
Do not pad a thin dossier with unsourced guesses.

When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
