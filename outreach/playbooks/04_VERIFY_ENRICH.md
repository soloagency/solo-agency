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
collector and its 3–5 most recent readable posts/videos analyzed for dated signals BEFORE the lead
can reach `high`. Use **`fb.profile.enrich`** with the profile's ROOT url: one page load returns the
recent posts AND the About section (trade, address, contact) in a single record. It replaces the old
`fb.profile.header` + `fb.profile.posts` + `fb.profile.contacts` sequence, which read the same
profile two or three times. Submit the root url, never `/about` — the timeline query only fires on
the root, so `/about` returns the About half with `posts: []`. Saving a
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
2b. **Decide the lead's `industry`, from the closed dictionary.** Read
   `solo-agency-collector/bridge-go/lead_industries.json` — 43 entries, each with `aliases`,
   `positive_signals` and `exclusions` — and store exactly one `industry` value **verbatim**: same
   case, same `&`, same commas. That file is the single source of truth; it is deliberately NOT
   reproduced here, because a second copy is a copy that drifts. YOU classify, by reading signals.
   Never write a value the collector guessed: `fb.people.search` returns an `industry_hint` from a
   14-value keyword map and `fb.profile.*` returns a `category` that only recognises real-estate and
   insurance wording — both are blind to every trade outside their list, and neither is this
   vocabulary.
   - **What to read**, in order of weight: a licence or credential number (`NMLS #2509012`,
     `DRE 01407449`, CPA, bar number) names a regulated trade outright and outranks self-description;
     then the email and website DOMAIN (`reachhomeloans.com` says it plainly); then the JOB TITLE
     under a Work heading — the employer alone is often ambiguous, since "at Wells Fargo" fits both
     `Banking & Financial` and `Loan & Mortgage`; then the bio and intro lines. Facebook's own
     `category` corroborates but never decides: it is blank on roughly a third of profiles.
     `fb.profile.dossier` returns all of this from ONE page load — read `about_lines`.
   - **Do NOT read their posts for this.** A person's trade is published on their profile; reading
     the feed costs many times the tokens for a weaker signal.
   - **Check `exclusions`, not just `positive_signals`.** Seventeen pairs in that list overlap on
     purpose-built boundaries — P&C Insurance vs L&H Insurance, Real Estate vs Loan & Mortgage,
     Immigration vs Work Abroad & Immigration — and the exclusions are what make two runs on the
     same lead agree. When two entries are genuinely co-equal, prefer the REGULATED trade the person
     is licensed for: a licence is a fact about them, a self-description is a claim.
   - **Record the decision, not just the answer:** `industry`, `industry_confidence` (0–1),
     `industry_reason` (one sentence), `industry_signals` (the lines you decided from). A stored
     label nobody can audit is a label nobody can correct.
   - **Thin signals → leave it EMPTY** and say why in `industry_reason`. A lead with no industry
     still gets worked; a wrong one silently mis-targets every message after it, and mis-targeting
     is invisible in a way an empty field is not.
   - This is a different thing from the industry-AGNOSTIC rule in Layer B above. That rule governs
     which EVIDENCE SOURCES you search for hooks and stays agnostic on purpose. This is a label ON
     the lead. It is also not the CLIENT's own `industry` in the Client Intelligence Profile, and not
     LeadUp's 10-value proposal taxonomy, which is remote, finer-grained in finance and law, and does
     not map onto these 42.
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

4c. **Record the profile's NAME while you are reading it.** Set `identity.name` (the name the
   profile actually shows), `identity.entity_type` (`person|company|page`), and for a person
   `identity.name_given` — the ONE word a greeting uses (Vietnamese: the last word of "Nguyễn Văn
   An" is "An"; English: the first word of "Charlie Bui"; code cannot know the order, you can).
   `enrich write` promotes it to `contact.name` when the contact has none — this is what stops the
   CRM listing leads as `c_...` — and refuses values that are not names (page chrome, phone/Zalo
   numbers, emoji taglines, bios), because a bad value becomes the reader's NAME in a real
   greeting. A page name or a "Person - Tagline" composite is stored as displayable but NOT
   greetable. `current_company` is the EMPLOYER, never a copy of the name.

4d. **Classify every owner-lookup failure — an infrastructure hiccup is not an exhausted search.**
   Stamp `identities.seeds[].resolution = {state, attempts, last_error, last_attempt_at}`:
   `resolved`, `retryable` (the COLLECTOR failed in a fixable way), or `exhausted` (the owner
   genuinely cannot be resolved). `retryable` covers a Messenger/chat overlay rendered instead of
   the page, an empty result, a timeout, or a `url_drifted` record. On `retryable`: close the
   overlay, open the URL in a clean tab, retry — up to 3 attempts total — and only then mark
   `exhausted`. **Why this is a hard rule:** on one live batch, 134 of 136 unresolved reel seeds
   were Messenger-overlay contamination. The owner filter refused that data correctly (it had
   already prevented a wrong merge), but every one was then counted as an ordinary unresolved lead,
   the run drafted whatever was ready, and reported — infrastructure noise was indistinguishable
   from finished work. `enrich status` now answers `seed_retryable_failure` for these and the
   approval report counts them in their own bucket: **a run with any left is not complete**, however
   many drafts it produced. And never merge or delete an unresolved seed record to make the number
   go down — with no verified profile, email or phone there is no identity to dedupe on, so the
   merge would be a guess.

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

   **Resolving the OWNER is only half the job: the reel page has no contact info.** The collector
   reads the page you POINT IT AT; it never wanders to another tab on its own. A reel/post page
   carries the content and the owner link and nothing else, so a pass that stops there returns
   `contacts.emails: []` for a person whose address is sitting one click away. Two real profiles
   from the LeadUp list show where it actually lives: `facebook.com/Khanhngo.us` reveals
   `khanh8601@gmail.com` only after "See more" in About, and `facebook.com/bgvinvest` carries
   `info@bgv.com.vn` in the contact-info block of its bio. Both came back "no email" because nobody
   opened those pages.

   So the ladder does NOT end at the owner URL. And "open the About tab" is not one step: on a
   business profile About renders a CHOOSER, and the address lives in a sub-page that is its own
   URL. Walk these in order and stop at the first hit (`bgvinvest` shown as the example):

   | # | Open | Why |
   |---|---|---|
   | 1 | `facebook.com/bgvinvest` — the profile itself, and **expand "See more"** on the bio | very often printed right there; `Khanhngo.us` only reveals `khanh8601@gmail.com` after See more |
   | 2 | `…/directory_contact_info` | the dedicated Contact-info page: the single highest-yield surface |
   | 3 | `…/directory_intro` | some profiles put it in the intro instead |
   | 4 | `…/directory_basic_info` | fallback field for the same data |
   | 5 | `…/directory_links` | no address, but the WEBSITE is here |
   | 6 | the website from #5, its Contact/Team/About page and footer | only worth the hop after 1-5 are empty |
   | 7 | licence roster or directory, web search, reverse search | the off-platform ladder, last |

   **Rows 1-5 are ONE collector job, not five.** Submit capability `fb.profile.contacts` against
   the PROFILE url (a reel/post url is not a profile — resolve the owner first). The capability
   enters About and CLICKS each sub-tab itself, waiting for the render, and expands "See more"
   on the bio. Do NOT hand-write a five-job tab crawl: fetching those sub-tabs returns the app
   shell with no contact block, which is how a run can claim it "checked" a page it never saw.

   **Copy its record into the dossier as `email_discovery`** — `{profile_url, emails, websites,
   found_on, checked}`, verbatim — and put any address it found into
   `identity.channels_found.emails` as well: that is the field the store reads to create the
   identity. (`email_discovery.emails` is now also accepted, but `channels_found` stays canonical.) `checked` lists the surfaces that actually rendered and is the
   audit trail `mark_email_not_found` is gated on; a dossier without it gets the conclusion
   refused. Rows 6-7 stay separate steps, and their input is that record's `websites` — do not
   re-scan the profile to find the site. Full contract: `solo-agency-collector/EMAIL_DISCOVERY.md`.

   **"No Contact info tab" is a legitimate finding.** Most personal profiles simply do not offer
   one (12 of 13 verified by hand), so `checked: [current_page, about]` with no sub-tab means the
   tab is absent, not that a click was missed.

   **Getting the address and getting the hooks are two DIFFERENT jobs — do not merge them.**
   `fb.profile.contacts` walks rows 1-5 and must NOT scroll: the address sits in the bio and the
   four `directory_*` sub-pages, never further down a feed, so scrolling only fires GraphQL queries
   for posts this task discards. Hooks come from `fb.profile.posts` / `fb.profile.videos`, which
   DO paginate. The catalog states this per capability (`scroll: "max_scroll"` vs `scroll: "none"`).

   **The seed the operator saved IS the hook — read it, do not go looking for a new one.**
   A lead added from a reel/post link came with its own reason to be contacted: the human picked
   that content. So the job is `curated content → resolve owner → open THAT content → confirm the
   owner matches and `url_drifted` is false → record the professional detail as a hook whose
   `evidence_url` IS the seed url`. Hunt a substitute only when the content is unreadable, drifted,
   or carries nothing professional. Writing `hooks: []` while that link sits unread is not a
   finding, and `enrich write` now says so: the lead comes back as `seed_hook_unharvested`, not as
   a routine `hooks_stale` refresh, and reporting it "skipped" claims a hunt that never happened.
   (Measured on the last live batch: 100 curated seeds in, 19 hooks citing them.)

   **Run contacts FIRST, and spend the post job only where it pays.** No address means the lead
   cannot be emailed, so its hooks are worthless: do not scroll a feed for someone you cannot
   reach. And a lead from an operator-saved reel/post ALREADY has its hook (that saved content) —
   it needs the contacts job only. On the last live batch this ordering would have been ~128 page
   loads instead of ~200, with nothing lost.

   **Rows 1-5 are where the answer usually is.** A person selling a service publishes a way to be
   contacted; that is the whole point of their page. On the last measured 71-lead run, 42% had the
   address published on Facebook — and **82% of those were only readable after "See more" was
   expanded**, which is why "we opened the profile" was never the same as having looked. Treat the
   website hop (6) and the off-platform ladder (7) as the EXCEPTION, not the routine next step.

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

   **Reading the content is NOT the email hunt (code-enforced).** A Facebook/social pass finds
   HOOKS; it almost never finds an address, because addresses live on the website's About/Contact
   page, in a licence roster or directory listing, or in a search result. Two separate ladders run
   per lead, and finishing one says nothing about the other:

   | Ladder | Done when |
   |---|---|
   | Hooks | ≥1 evidenced Layer-B point, or genuinely zero after the springboard |
   | **Email** | an address is found, **or** the hunt has actually left the social pass |

   **A search result is EVIDENCE, never an IDENTITY (code-enforced).** When the hunt goes through
   Google/Bing/DuckDuckGo, what you report back as a found profile must be the **profile page
   itself**, never the search URL that led you there. `channels_found.profiles` is validated: the
   URL must classify as a PROFILE and match the platform it is filed under, and search/aggregator
   hosts are refused outright. Why this matters more than it looks: identities are **dedupe keys**,
   and the key drops the query string, so every search on one engine collapses to a single value.
   A live run filed Bing search URLs as Facebook profiles; dozens of unrelated leads then "shared
   an identity" and consolidation chain-merged **54 reels from different people into one contact**,
   destroying the cluster. Cite the search URL in `identity.evidence[]` if it is useful; keep it
   out of `profiles`.

   **What "having an email" MEANS here (do not raise this bar yourself).** An address is usable for
   drafting when it is a REAL found address (never guessed), it parses, its domain accepts mail
   (`tool verify-email` → `mx_ok`), and the identity is not suppressed. That is the whole bar.
   There is **no mailbox-verification step in this system** and none is planned: probing a mailbox
   (SMTP `RCPT TO`) is unreliable against Gmail and risks the sending reputation of every sendbox,
   so it is deliberately not done. Consequences to internalize:

   - `identities.emails[].status: "unverified"` is the NORMAL state for an enrichment-found address.
     It means "not mailbox-probed", which is expected. It is **not** a blocker.
   - `channels.email.status` flips to `usable` on its own once a valid address is on file. If you
     ever see `needs_data` next to a real address, that is stale data, not a gate.
   - `draft write` enforces the real bar in code (a parseable address, suppression, hooks). If it
     accepts the draft, the lead was draftable. Do **not** invent an extra verification step and
     skip the lead for failing it: a run that reports "0 drafts, emails not verified" while holding
     real addresses has failed the task, not protected it.

   `enrich write` **REFUSES `mark_email_not_found`** when every evidence URL in the dossier is on
   facebook.com: a run that never left Facebook has not earned the right to call an address
   unfindable. The refusal is non-destructive — the hooks you gathered are still written, only the
   negative cache is withheld, and `enrich status` keeps returning **`needs: enrich`, reason
   `email_discovery`** so the lead stays visible as UNFINISHED WORK. Report those leads as
   **research pending**, never as "skipped": "skipped" claims you looked everywhere.

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
