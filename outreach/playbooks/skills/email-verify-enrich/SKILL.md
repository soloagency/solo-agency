---
name: outreachcrm-email-verify-enrich
description: >-
  Use during a daily run's enrich step (Stage 4) to VERIFY a contact is still active and
  ENRICH them into a write-ready lead for a deeply personalized cold email. Two tiers: a cheap
  verify pass (still in the business? gather profile URLs) then a proof-of-life pass — gather as
  MANY evidenced public-professional signals as you can find (recent activity, reputation,
  positioning, scale — each with a source URL), never capping at one or three. Writes the dossier
  via crm_store.py. Never guesses an email address, never mines personal life. Loaded after
  Stage 4 / from the daily run's "load new pipeline -> enrich" step.
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
- **No personal-life mining (anti-creepy stance).** Collect ONLY public, professional signals —
  the dossier feeds a writer who is a dedicated peer that did their homework before a professional
  approach, NOT a surveillant. `public_business` signals only (listings, work posts, reviews,
  awards, market views). Family/health/vacation/children → `sensitivity: personal`, which goes to
  `do_not_mention`, never to email copy. Gather MANY public-professional points, but never a
  dated, itemized log of someone's life — that is exactly what reads as scraped. See `etiquette.md`.
- **Read-only.** Use WebSearch / WebFetch (and a browser tool only where the readability table
  says so). Do not log into anyone's account. Facebook/LinkedIn logged-out = store the URL only
  (Phase 4 Local Collector reads those later).
- **Inherit, don't re-burn.** The dossier belongs to the CONTACT and is reused across that
  client's campaigns. Check `crm_store.py enrich status` first; only enrich/refresh when it says
  so. A `no_verifiable_hook` / `email_not_found` negative cache is inherited — respect its window.

## The Write-Ready gate (3 layers + floor)

Everything below serves ONE target: make each lead **write-ready**. A lead is write-ready for a
deeply personalized message only when it has all three layers (DESIGN §9):

- **Layer A — Reachability (≥1 required):** at least one deliverable channel — a real found email
  (→ email), or a DM-capable social profile / phone (→ messenger/assisted). This is
  `identity.channels_found` + `identity.profiles`.
- **Layer B — Proof-of-Life (≥1 required; MORE IS BETTER — do NOT cap):** evidenced, recent public
  PROFESSIONAL signals — the personalization fuel. Each point is one basis for a conclusion the
  writer weaves. The taxonomy is universal and industry-AGNOSTIC; INFER which sources fit the
  lead's field — never work from a hardcoded per-industry list:
  1. **Recent activity/output** (strongest — proves alive AND gives a hook): a new post / video /
     article / release / listing / project / menu, etc.
  2. **Reputation / social proof:** review count + substance, ratings, testimonials, awards, press,
     verified badge, follower / engagement figures.
  3. **Positioning / identity:** website tagline, bio, "since 20xx", stated specialty / mission.
  4. **Scale / momentum:** volume figures, team / locations, growth story, price range.

  (Those four are the taxonomy, not a source list. Prefer #1/#2 with a recent `observed_date`;
  #3/#4 support the reframe.)
- **Layer C — The Opening (NOT collected here):** the specific gap the campaign's offer resolves.
  The WRITER derives it at write time from Layer B + the offer. Do not try to produce it in the
  dossier; your job is to make Layers A+B rich enough that the writer can.

**The floor.** ≥1 Layer-B point → **write-ready** (deep personalization); the COUNT + quality of
Layer-B points scales `personalization_confidence` (one thin point → review-carefully; ≥3 solid,
fresh, on-goal → high). Springboard exhausted and still 0 Layer-B → **NOT write-ready** →
`mark_no_hook` and let `no_hook_fallback` decide. Layer A fails (no channel) → assisted or skip.
"Use what you have" applies to the CHANNEL and the degraded path — the personalization floor stays
≥1 Layer-B.

**Seed normalization — resolve a REAL identity before the first search.** A search query is built
from a real person/business, NEVER from a URL. If the only seed is a Facebook/social URL (or the CRM
`name` is blank), FIRST read the profile with the Local Collector's `fb.profile.header` — the Phase-4
collector is now LIVE (operator's own logged-in Chrome; bridge `127.0.0.1:17321`, enqueue capability
`fb.profile.header` with the profile URL; see `solo-agency-collector`). Take its `name` + `category`
+ a location signal (city from the header/intro) and build the first query as `"<name>" <category>
<city>`. **Do NOT build a query from a URL path/slug** (e.g. `absellsaz`, or "videos/reels insurance
agent" stitched from URL words) — slug queries return directory junk, not the person. If the header
yields no usable name, pull the person's own words + place-names from `fb.profile.videos` /
`fb.profile.posts` captions before ever falling back to a slug. Pass a `profile.php` URL only WITH
its `?id=<numeric_id>` — a bare `profile.php` resolves to the operator, not the lead.

**The springboard (industry-agnostic, iterative).** From ANY seed — a name, an email, a single URL
— pivot to find the rest and loop until returns diminish: social → website (email + tagline) →
industry / directory page (volume, reviews) → reverse search (email / name → other profiles).
REASON about which sources fit the lead's trade; there is no fixed per-industry list. Keep digging
until Layers A+B are satisfied, then stop.

## The two-tier flow

Run over the leads `crm_store.py enrich due --campaign <slug>` returns.

**Tier 1 — Verify + reachability (cheap).** Confirm the person is still active and collect profile
URLs + any real channel (Layer A). Run the springboard from the seed — these sources are
ILLUSTRATIVE of a real-estate lead, so GENERALIZE them to the lead's actual field rather than
treating them as a fixed list: official license/registry lookup → brokerage/employer roster (email
domain is a hint) → Zillow/realtor.com search snippet → Google Business Profile → the person's own
website (email + tagline). For a different trade you would infer the equivalents (a chef → the
restaurant site + reservation/review platforms; a consultant → their site + LinkedIn URL +
directory listings). If the person is clearly inactive/left the field, set `still_active: inactive`
and STOP — do not run Tier 2 (don't spend the proof-of-life pass on a dead lead).

**Tier 2 — Profile & proof-of-life (hooks).** Visit the URLs Tier 1 found (see `channel_reality.md`
for what each actually yields) and **gather as MANY evidenced Layer-B proof-of-life points as you
can find — do NOT cap at one or three.** Each point is a conclusion-basis the writer will weave, so
more solid, fresh, on-goal points = a richer message and a higher `personalization_confidence`;
there is no upside to stopping early. Record each with `type`, `summary`, `evidence_url`,
`observed_date`, `confidence`, and `analysis` where `analysis.sensitivity` gates copy eligibility
and **`analysis.angle` is the conclusion that hook supports** (the implication the writer can draw
from it, e.g. a share count → "people are already passing your version along"). **Always set
`observed_date`** — a usable hook missing it is kept but `enrich write` flags a `problems` note
(recency unverified), and recency is what makes proof-of-life real. Analyze the 3–5 latest readable
posts where possible. Distill a `writing_brief`: a one-liner, ranked angles (freshness × goal-fit ×
confidence), a `do_not_mention` list, and a `personalization_confidence` set by the **COUNT +
freshness + goal-fit** of the Layer-B points (drives the band: ≥0.7 high, 0.4–0.7 review_carefully,
<0.4 fallback). Selecting which points to actually use is the WRITER's job, not yours — your job is
to find and evidence as many as exist.

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
`hooks[]` is **not capped** — record every evidenced Layer-B point you found, not a curated top few.
`enrich write` stores the full dossier under `campaigns/{slug}/queue/enriched/YYYY-MM-DD/` and a
distilled copy into `contact.enrichment` (inherited by other campaigns). It returns
`usable_hooks`, `confidence_band`, and any `problems` (e.g. a hook it dropped for missing evidence).

## No-hook branch

This is the **floor failing**: the springboard is exhausted and you have **0 Layer-B proof-of-life
points**. (Realistic hit-rate for a deeply-personalized hook is ~30–50%; see `channel_reality.md`.)
Then set `mark_no_hook: true` and let the campaign's `no_hook_fallback` decide: the default `skip`
(a hookless step-1 draft is rejected by `draft write` — evidenced proof-of-life is the reason an
email exists), or the explicit opt-in `generic_honest_opener` (a generic-but-honest opener grounded
only in license/roster facts). Do not pad a thin dossier with unsourced guesses, and do not lower
the floor: **one** evidenced Layer-B point already clears it, so only mark no-hook when there is
genuinely zero.

## No-email branch (email discovery)

A lead an `email_first` campaign queued with no email is here so you can FIND one — check the
website, a license/registry record, a brokerage roster, Google, other public channels. Store any
real address as a found identity (`source: enrich`). If discovery genuinely fails, set
`mark_email_not_found: true` — this writes a 30-day negative cache so a later campaign does not
re-burn the same dead end, and the contact becomes an **assisted-channel candidate** (manual
SMS/Messenger/Zalo). Never fabricate a `first.last@domain` guess.

When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
