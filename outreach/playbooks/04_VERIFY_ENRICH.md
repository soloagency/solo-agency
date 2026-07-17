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
- **No personal-life mining (anti-creepy stance).** Collect ONLY public, professional signals —
  this dossier feeds a writer who is a dedicated peer that did their homework, not a surveillant.
  `public_business` signals are hooks; `personal` signals go to `do_not_mention` only (see the
  skill's `etiquette.md`). Gather MANY public-professional points, but never a dated, itemized log
  of a person's life.
- **Inherit before you enrich.** The dossier belongs to the contact and is reused across that
  client's campaigns. Always check `crm_store.py enrich status` first and act on its verdict.
- **Read-only, logged-out.** WebSearch/WebFetch (+ browser tool only where `channel_reality.md`
  says). Never log into an account. **Facebook is now readable via the Local Collector**
  (`fb.profile.header` → real name/category, then `fb.profile.posts`/`fb.profile.videos`): read the
  header to get the real name BEFORE searching, and never search from the URL slug. LinkedIn: store URL only.

## The Write-Ready gate (what this stage must produce)

This stage exists to make each lead **write-ready** for a deeply personalized message (DESIGN §9).
Three layers, with a hard floor:

- **Layer A — Reachability (≥1 required):** at least one deliverable channel — a real found email
  (→ email) or a DM-capable social profile / phone (→ messenger/assisted). This is
  `identity.channels_found` + `identity.profiles`.
- **Layer B — Proof-of-Life (≥1 required; MORE IS BETTER — do NOT cap):** evidenced, recent public
  PROFESSIONAL signals — the personalization fuel; each point is one basis for a conclusion the
  writer weaves. Universal, industry-AGNOSTIC taxonomy (INFER the sources that fit the lead's field
  — do NOT work from a hardcoded per-industry list): (1) recent activity/output, (2) reputation /
  social proof, (3) positioning / identity, (4) scale / momentum.
- **Layer C — The Opening (NOT collected here):** the specific gap the offer resolves; the WRITER
  derives it at write time. Your job is only to make Layers A+B rich.

**The floor gate.** ≥1 Layer-B point → write-ready; the COUNT + freshness + goal-fit of Layer-B
points scales `personalization_confidence`. Springboard exhausted and still 0 Layer-B → NOT
write-ready → `mark_no_hook` (`no_hook_fallback` decides). Layer A fails → assisted or skip. "Use
what you have" applies to the CHANNEL and the degraded path — the personalization floor stays ≥1
Layer-B. **The springboard:** from ANY seed (name / email / one URL), pivot and loop until returns
diminish (social → website → industry/directory page → reverse search), reasoning about which
sources fit the trade, until Layers A+B are satisfied.

## Source Preservation Rule

The dossier and `contact.enrichment` are written through `crm_store.py` (a `crm/` mutation).
Do not hand-edit them. When any instruction here disagrees with `docs/DESIGN.md`, `docs/DESIGN.md`
wins.

## The run

1. Get the batch: `crm_store.py enrich due --campaign <slug> --limit N` returns queued leads that
   need enrich or refresh (already-fresh ones are skipped — that is cross-campaign inheritance).
2. Load the skill (`email-verify-enrich`) and run its two-tier flow per lead: Tier 1 verify +
   reachability (still active? profile URLs + any real channel = Layer A), Tier 2 proof-of-life
   (gather as MANY evidenced Layer-B points as you can find — `public_business` only, do NOT cap;
   each is a conclusion-basis the writer weaves), distill a `writing_brief`.
3. Write it: `crm_store.py enrich write --contact <lead_id> --campaign <slug> --json '<dossier>'`.
   It stores the full dossier under `campaigns/{slug}/queue/enriched/YYYY-MM-DD/` and a distilled
   copy into `contact.enrichment`, and returns `usable_hooks` / `confidence_band` / `problems`. A
   usable hook that lacks an `observed_date` is **kept but flagged** in `problems` (recency
   unverified) — always set `observed_date`, since recency is what makes proof-of-life real.
4. No-hook leads (the ≥1 Layer-B floor failed — 0 proof-of-life after the springboard is
   exhausted): set `mark_no_hook` and let the campaign's `no_hook_fallback` decide — default
   `skip` (a hookless step-1 draft is rejected), or the explicit opt-in `generic_honest_opener`
   (grounded in license/roster facts). One evidenced Layer-B point already clears the floor — only
   mark no-hook at genuine zero. Inactive leads: `still_active: inactive`, stop — do not draft.
5. **No-email leads (email discovery):** a lead an `email_first` campaign queued with no email is
   here precisely so Tier 1 can DISCOVER one (website, license/roster, Google, other public
   channels). Store any real address found (`source: enrich`). If discovery genuinely fails, set
   `mark_email_not_found` → a 30-day negative cache (so a later campaign does not re-burn the dead
   end) and the contact becomes an **assisted-channel candidate** (manual SMS/Messenger/Zalo).
   Never invent a guessed address.

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
6 (email writing, 2C) and Stage 10 (follow-up/reply, 2D) are **built** too. Only Stages 12/15
remain `status: planned` (Phase 3); where a referenced row is still planned, follow DESIGN §22 R1.

When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
