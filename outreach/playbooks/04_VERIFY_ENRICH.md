# Stage 4 — Verify & Enrich

## Load Rule

Load before ANY enrichment: the daily run's "load new pipeline → verify → enrich" step, or an
OPPORTUNISTIC follow-up micro-refresh (enrichment is NOT re-run per bump — see the Completion
Gates and `skills/email-writing/followup.md`). Its dependency is the skill `playbooks/skills/email-verify-enrich/`
(SKILL.md + `channel_reality.md` + `etiquette.md`) — each needs its own LOAD LEDGER per
`playbooks/LOAD_LEDGER_PROTOCOL.md` before you act.

## Hard Gates For This Stage

- **Every usable hook carries an `evidence_url`.** Enforced by `tool crm-store enrich write` (it
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
  client's campaigns. Always check `tool crm-store enrich status` first and act on its verdict.
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
  writer weaves. **Recency is enforced, not aspirational (code gate in `enrich write`):** a
  `high` band REQUIRES ≥1 dated hook ≤60 days old that is NOT a `website_update` — a website
  positioning line is background, not proof-of-life, because websites go stale while social does
  not. If a Facebook profile is on file (or you just found one) and you have NOT read it via the
  collector, the store caps the band to `review_carefully` and tells you to read it. Never stamp a
  hook's `observed_date` with today's date because that is when YOU read the page — use the
  content's own publish/update date, or leave it empty (an undated hook never counts as recent). Universal, industry-AGNOSTIC taxonomy (INFER the sources that fit the lead's field
  — do NOT work from a hardcoded per-industry list): (1) recent activity/output, (2) reputation /
  social proof, (3) positioning / identity, (4) scale / momentum.
- **Layer C — The Opening (NOT collected here):** the specific gap the offer resolves; the WRITER
  derives it at write time. Your job is only to make Layers A+B rich.

**The floor gate.** ≥1 Layer-B point → write-ready; the COUNT + freshness + goal-fit of Layer-B
points scales `personalization_confidence`. Springboard exhausted and still 0 Layer-B → NOT
write-ready → `mark_no_hook` (`no_hook_fallback` decides). Layer A fails → assisted or skip. "Use
what you have" applies to the CHANNEL and the degraded path — the personalization floor stays ≥1
Layer-B. **The springboard (social-FIRST, mandatory order):** from ANY seed (name / email / one URL),
resolve the person, then read their SOCIAL presence BEFORE their website — social is where recent
professional activity lives; a website is a background/confirmation source only. Order: **social
(read it, don't just save the URL) → website → industry/directory page → reverse search**, looping
until returns diminish. Concretely: any lead with a Facebook profile MUST be read through the
collector (`fb.profile.header`, then `fb.profile.posts` / `fb.profile.videos`) and its 3–5 most
recent readable posts/videos analyzed for dated signals BEFORE the lead can reach `high`. Saving a
Facebook URL without reading it is NOT enrichment. Stopping at the website is the #1 failure mode —
websites are months out of date; the whole point of Layer B is RECENT activity.

## Source Preservation Rule

The dossier and `contact.enrichment` are written through `tool crm-store` (a `crm/` mutation).
Do not hand-edit them. When any instruction here disagrees with `docs/DESIGN.md`, `docs/DESIGN.md`
wins.

## The run

1. Get the batch: `tool crm-store enrich due --campaign <slug> --limit N` returns queued leads that
   need enrich or refresh (already-fresh ones are skipped — that is cross-campaign inheritance).
2. Load the skill (`email-verify-enrich`) and run its two-tier flow per lead: Tier 1 verify +
   reachability (still active? profile URLs + any real channel = Layer A), Tier 2 proof-of-life
   (gather as MANY evidenced Layer-B points as you can find — `public_business` only, do NOT cap;
   each is a conclusion-basis the writer weaves), distill a `writing_brief`.
3. Write it: `tool crm-store enrich write --contact <lead_id> --campaign <slug> --json '<dossier>'`.
   It stores the full dossier under `campaigns/{slug}/queue/enriched/YYYY-MM-DD/` and a distilled
   copy into `contact.enrichment`, and returns `usable_hooks` / `confidence_band` / `problems`. A
   usable hook that lacks an `observed_date` is **kept but flagged** in `problems` (recency
   unverified) — always set `observed_date`, since recency is what makes proof-of-life real.
4. No-hook leads (the ≥1 Layer-B floor failed — 0 proof-of-life after the springboard is
   exhausted): set `mark_no_hook` and let the campaign's `no_hook_fallback` decide — default
   `skip` (a hookless step-1 draft is rejected), or the explicit opt-in `generic_honest_opener`
   (grounded in license/roster facts). One evidenced Layer-B point already clears the floor — only
   mark no-hook at genuine zero. Inactive leads: `still_active: inactive`, stop — do not draft.
4b. **Start from the clues the list already gave you (`custom_fields`).** Import keeps EVERY
   column of the operator's list, including the ones the canonical mapping did not claim (NPN,
   license number/type, years in practice, agency/brokerage, MLS or DRE id, business address,
   the operator's own notes...). Read `custom_fields` BEFORE searching blind: a licence number
   or NPN resolves to an official roster page in one lookup, an agency name plus a personal name
   finds the profile far faster than a name alone, and a roster page is often where a real
   business email lives. These are research starting points, not evidence: a hook still needs
   its own dated `evidence_url`, and a roster/licence fact only ever grounds a
   `generic_honest_opener`. Personal attributes that may ride along in the source list (age,
   gender, home address/phone) are stored for provenance but are NOT search fuel, NOT
   segmentation criteria, and never appear in copy — treat them exactly like `personal`
   signals: push them to `do_not_mention`.

5. **Anchorless leads — resolution ladder first (DESIGN §9.1b): seed → profile → email → hooks.**
   When `enrich status` says `seed_unresolved`, Tier 1's FIRST move is origin resolution. **Take the
   owner from the URL whenever the URL carries it — do not spend a collector call:**
   `/<vanity>/posts/…` → `facebook.com/<vanity>`; `/<numeric>/posts/…` → `facebook.com/<numeric>`;
   `permalink.php`/`story.php?…&id=<numeric>` (incl. `m.facebook.com`) → `facebook.com/profile.php?id=<numeric>`.
   Measured on 19 real seeds: URL-derived owners 11/11 correct, while the collector's DOM
   `profile_candidates` on those same post pages was 7/7 WRONG (unrelated profiles, even group URLs)
   and 4 pages timed out. Only `/reel/<id>` and group permalinks need the collector (a reel URL has
   no owner in it) — reels are reliable (8/8 verified) when you read the `sk=reels_tab` "See Owner"
   link and reject any record with `url_drifted: true`. For non-Facebook content (YouTube/TikTok/blogs)
   open it in the browser and read the byline/channel/handle. Record the profile in
   `identity.channels_found.profiles` — the store writes it back into `identities.socials` as a
   canonical dedupe-eligible identity and marks the seed `resolved`. A `name_only_fragment` lead
   gets a name search instead — run it through the Local Collector's **`fb.people.search`**
   capability with **the NAME ONLY as the query** (Facebook ANDs every token: 6+ token
   name+company+city queries returned 0/56, name-only returned results 87% of the time). Use
   company/city/state to FILTER `records.items` by `subtitle`, never to narrow the query; ladder
   `First Middle Last` → `First Last` → `First Last <state>` and stop at the first rung with records.
   Take candidates from **`records.items`** ONLY; NEVER the DOM `entity_candidates` (it grabs the
   operator's open Messenger-chat contact — see `channel_reality.md`). Ladder exhausted → genuine
   no-result; never accept a DOM candidate. Then continue as a profile-seeded lead.

   **Batch-resolve BEFORE deep enrichment (reel-heavy lists).** User lists routinely carry many
   content links of the SAME person. So: resolve ALL unresolved seeds to owner profiles FIRST
   (cheap header reads), submitting each via `enrich write` with just `channels_found.profiles`
   — the store consolidates automatically (consolidation-on-discovery, DESIGN §9.1b): fragments
   sharing a profile/email auto-merge into ONE contact with a full union (every reel, every hook
   kept), and the result reports `consolidated: [{survivor, merged, ...}]`. Only THEN deep-enrich
   each surviving unique profile ONCE — never burn collector budget enriching the same person
   twice. Two rules: (a) always continue work against the returned `lead_id` (the survivor, which
   may differ from the id you submitted); (b) if the result carries `duplicate_suspected`, the
   store found a CONFLICTING record sharing that identity (e.g. a brokerage page posting several
   agents' reels) — it did NOT merge and both records are held out of campaign queues; surface it
   to the operator (`contact merge` if same person, `contact unsuspect --id A --other B` if not).
   Report all consolidations and suspects in the run summary — "30 rows became 12 people" is a
   normal, healthy outcome, not an error.
6. **No-email leads (email discovery):** a lead an `email_first` campaign queued with no email is
   here precisely so Tier 1 can DISCOVER one (website, license/roster, Google, other public
   channels). Store any real address found (`source: enrich`). If discovery genuinely fails, set
   `mark_email_not_found` → a 30-day negative cache (so a later campaign does not re-burn the dead
   end) and the contact becomes an **assisted-channel candidate** (manual SMS/Messenger/Zalo).
   Never invent a guessed address.

## TTL, inheritance, negative cache (all in `enrich status`)

- Identity (still-active, company, role, profile URLs) is durable — TTL **360 days**, reused as-is by
  other campaigns. (Operator decision 2026-07-27: the old 90/10 pair dragged every contact back
  through full enrichment ~4x and hook-refresh ~36x a year, against the enrich-once doctrine. The
  accepted trade is that someone who changed jobs is caught at SEND time by the bounce path
  instead of proactively.)
- Hooks are fresh — TTL **360 days**; a stale-hook contact returns `needs: refresh` (revisit known
  URLs), not a full re-enrich. A dossier holding **zero** hooks is never "fresh" regardless of when
  you last looked — those leads are governed by the `no_verifiable_hook` 30-day cache instead, so
  an empty-handed lead is retried in a month, not parked for a year.

**Recency, and the one exemption (code-enforced).** A hook needs a date within the past **360 days**
to carry a `high` band on its own. The exemption: **a hook whose `evidence_url` is a URL the OPERATOR
supplied** (a reel/post they saw, judged relevant and saved — it lives in `identities.seeds[]` with
`source: import`) is exempt from recency entirely, at any age. The human act of saving it IS the
qualification, exactly like the User-Curated List Rule. This is NOT a judgment you make: the store
compares the hook's URL against the stored seeds and decides. Hooks the system dug up on its own get
no exemption — they must still be recent. Do not add a seed yourself to manufacture the exemption.
- `email_not_found` / `no_verifiable_hook` are inherited negative caches so a second campaign does
  not re-burn the same dead end within its retry window. **Both windows are 30 days, and both are
  enforced in `enrich status`**: a lead whose springboard was exhausted returns
  `skip / no_verifiable_hook_recent` instead of coming back as `hooks_stale` every 10 days. The
  mark clears itself the moment a later pass DOES find a hook (or an email), so a cached failure
  never outlives the fact that produced it. Only set `mark_no_hook` when the springboard is
  genuinely exhausted: it costs the lead a month.

## Completion Gates

- Every drafted personalized detail (Stage 6) traces to a stored hook with an `evidence_url`.
- No guessed email exists; found emails are `source: enrich`.
- No `personal` hook is a usable hook (it is in `do_not_mention`).
- Freshness respected: step-1 drafts use in-TTL hooks. Follow-ups draw on RESERVED points + the
  campaign `message_bank` (no per-bump re-enrichment); a micro-refresh ran only opportunistically
  (reserved points exhausted AND collector spare capacity); any time-sensitive hook referenced in
  a bump was re-verified or dropped (stale-hook guard).

## Phase status

The enrich storage/TTL/validation tooling (`tool crm-store enrich`, 2B) is **built**. The web
verify/enrich itself is agent behavior driven by the `email-verify-enrich` skill. Downstream Stage
6 (email writing, 2C) and Stage 10 (follow-up/reply, 2D) are **built** too. Only Stages 12/15
remain `status: planned` (Phase 3); where a referenced row is still planned, follow DESIGN §22 R1.

When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
