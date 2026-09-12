# Lead Engine — Recipes

Ready-made capability sequences for common lead intents. Each recipe is a
starting plan for the gather loop in `SKILL.md` — adapt inputs to the real
persona/keywords/location, and always run under `safety.md` limits and Stage 10
qualification. Capability ids and inputs come from `GET /capabilities`; if a
capability's `status` is not `stable`/`beta`, skip or substitute it.

Keyword banks below are examples — build the real bank from the client's `buyer_profile.types`
(who the audience is) and `why_they_need` (the frictions the offer removes), in the audience's own
language (Vietnamese / English / etc.), per channel — the channel table lives once in
`playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`'s keyword-bank section: role/product/stage terms are
PRIMARY for in-group search, intent terms only when anchored to a role or the offer; occupation+place
for people search; profession jargon+hashtags for IG/X search. A term with no role or offer anchor —
a bare "advice", "looking for", "need help", "any recommendations", "content help" — is rejected by
the quality gate, not banked.

---

## Recipe A — Facebook leg of the Social Discovery Pass (canonical; feed → people → groups → in-group)

> The canonical implementation of the Facebook leg of `playbooks/10_LEAD_COMPETITOR_DETECTION.md`'s
> Social Discovery Pass (step 11C of the daily run, and Setup Flow's first-run trigger). The same
> pass also runs Instagram (search → people → profile depth → comments) and X (search Latest →
> people → profile depth → replies) in the same shallow-to-deep shape, interleaved round-robin with
> Facebook — one Facebook job, then one Instagram job, then one X job, repeat, each platform
> advancing one step per round; a platform not connected/logged-in/at-budget/tripped just loses its
> turn. See Stage 10's "Social Discovery Pass" for the full platform table, round-robin rule, and
> Instagram/X call details. This Facebook shape is also the right one for an open-ended "find people
> who need X" ask — e.g. "find people who need insurance", "who is looking to buy a house in OC" —
> the fixed order below IS the general answer to that request, not just the automated daily version
> of it.

Fixed order — feed first, then people, then groups, then in-group. Do not reorder or parallelize;
narrow the funnel at each step instead of skipping one.

```text
1. FEED FIRST
     fb.search.posts { search_url: "https://www.facebook.com/search/posts/?q=<discovery term>" }
        Facebook's own global Posts-tab search. No group membership needed; fastest first read on
        who is using in-market language right now.
2. THEN PEOPLE
     fb.people.search { query: "<discovery term>" }
        (the url Facebook itself renders: https://www.facebook.com/search/people/?q=<discovery term>)
        Returns ProfileSummary[] rows (name, url, subtitle/work line, mutual_friends, industry_hint).
        No post text exists, so classification runs `playbooks/LEAD_QUALIFICATION_RULE.md` Step 1
        against subtitle + industry_hint + name/url only: a subtitle/bio that matches a line of the
        client's `buyer_profile.types` is `fit = high`, and with no stated need that is
        `intent = none` → `warm` per the matrix — never dropped for lacking a stated need. No match
        stays lower per the same rule. Qualified rows → `tool crm-store ... lead capture` tagged
        source:people_search plus kw:{term}, exactly like every other lead.
3. THEN GROUPS
     fb.groups.search { query: "<discovery term>", max_pages: <=4 }
        Keep a group when privacy == "public" OR viewer_join_state == "MEMBER" — the account can
        read either one, so both are groups_readable. A private group the account has not joined
        (viewer_join_state CAN_REQUEST / REQUEST_TO_JOIN, or any other non-member join state under
        privacy == "private") is groups_no_access: registered state: no_access, listed for the Boss
        to see, never scanned, never joined, never requested. Unknown privacy is NOT treated as
        no-access by default: spend one fb.group.posts { max_pages: 1 } probe — posts come back →
        readable; an access wall, or empty with stopped_because naming access or login → no_access.
        Score every readable group with the Group Potential Rule
        (playbooks/10_LEAD_COMPETITOR_DETECTION.md) and register it: `tool source-registry add
        --client <slug> --platform facebook --source-type group --origin discovered --url <u> --name
        <n> --member-count <m> --privacy <public|private> --state <active|not_selected|no_access>
        --potential <high|medium|low> --reason "<one line>"` — no shortlist file, no approval. Skip a
        group already in private_data_sources, and skip one this pass already scanned in the last 7
        days (the source registry is the memory now; `tool source-registry plan` ranks by recency and
        leads automatically).
4. THEN IN-GROUP
     For the groups from `tool source-registry plan --client <slug> --platform facebook --max 20`
     (up to 20 monitored groups: most leads across their last 3 scans first, then never-scanned
     newest first, then longest-unscanned, ties by member count; whatever does not fit the 20 rolls
     to the next run automatically):
     fb.group.search_posts { group_search_url: ".../groups/<id>/search/?q=<term>", max_pages: <=4 }
        Terms come from `tool source-keywords ... plan --kind <kind>` (seed the group's bank first
        with `seed --industry --market --lang` when it is empty, plus the client's setup seed file
        when one exists) — NOT the discovery term from step 1/2/3. Draw per the in-group row of the
        keyword-bank channel table (`playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`, kinds
        intent|role|product|stage|place): role and product/stage terms are PRIMARY — what the
        client's `buyer_profile.types` calls themselves, what they say about their work; intent
        terms only when anchored to the role or the offer, never a bare need phrase. Per-kind
        quota, drawn with `--kind` and rotated within a kind least-recently-run first:
        DAILY — 2 intent terms per group.
        FIRST RUN — 3 intent terms per group.
        After each group's scan, record it — `tool source-registry record --client <slug> --run
        <run_id> --url <group_url> --leads <n>` (n = hot+warm+watch captured this scan) — this is
        what re-ranks the plan for next time.
        Example for a client selling to real-estate agents:
        good — "listing agent" (role), "just listed" (product/stage), "realtor video" / "cần video
        bất động sản" (intent, anchored to the offer).
        rejected by the quality gate (generic, no role/offer anchor) — "advice", "looking for",
        "need help", "any recommendations", "content help" alone.
5. Every post and person row from all four steps is classified immediately against
   `playbooks/LEAD_QUALIFICATION_RULE.md` (Step 1 first — who they are, before intent) → keep
   decision hot|warm|watch; dedupe by post/profile URL. Then `tool crm-store ... lead capture` —
   even for a group not yet in private_data_sources.
6. Deepen only within the budget below; do not raise max_pages past it without human approval.
7. WRITE BACK what this pass learned, same mechanic as every hunt. A pass discovers, in an hour,
   which phrasings a particular group answers to — and without this step that knowledge dies when
   the run ends and tomorrow's monitoring keeps searching the same guesses. For every group that is
   (or becomes) a watched private source, put the terms that produced qualified leads into its bank:
     `<bridge> tool source-keywords --pipeline daily-content-pipeline --client {slug} --url {group url} add --term "{term}" --kind intent --origin mined --note "{what this pass saw: N qualified leads on {date}}"`
   then record the outcome so the term carries its evidence rather than an opinion:
     `... record --json '{"{term}":{"hits":N,"leads":M}}'`
   Terms that produced nothing are worth recording too — a zero is how the bank learns to stop
   spending a slot on them. Discovery terms themselves (steps 1, 2 and 3) write back through a
   different bank/kind instead:
     `<bridge> tool public-keywords --pipeline daily-content-pipeline --client {slug} plan --kind discovery` to draw them (default 3 terms; add one by hand with `add --group community_discovery --term "..."`),
     `... record --json '{"{term}":{"verdict":"useful|used|weak|retry_later","urls":N,"ideas":M}}'` to record outcomes — the `record` verb has no `--kind` flag and requires a verdict: `useful` on ≥1 lead, `used` on results but no lead, `weak` on nothing, `retry_later` after a safety trip.
```

Budget (owner-approved; `playbooks/10_LEAD_COMPETITOR_DETECTION.md` and `safety.md` are authoritative
— restated here only so the recipe is self-contained). This is Facebook's own budget; Instagram runs
≤ 12 calls FIRST RUN / ≤ 4 calls DAILY (3/3/3/3 and 1/1/1/1 across search/people/profile-depth/
comments) and X the same ≤ 12 / ≤ 4 shape (search-Latest/people/profile-depth/replies) — see Stage
10's platform table for the exact per-platform breakdown:

| | discovery terms | feed searches | people searches | group searches | monitored groups (registry plan) | intent terms/group | total calls | spread |
|---|---|---|---|---|---|---|---|---|
| FIRST RUN | 3 | 3 | 3 | 3 | up to 20 | 3 | 9 discovery + up to 20 × 3 | Pacing Rule: random 5–10 s per request, no added gaps |
| DAILY | 1 | 1 | 1 | 1 | up to 20 | 2 | 3 discovery + up to 20 × 2 | Pacing Rule: random 5–10 s per request, no added gaps |

Lead target: FIRST RUN is a floor of 10 across all three platforms combined, not a stop — keep
working the ranked candidates until each platform's own budget is spent. Safety trip is per platform
and unforgiving: the first checkpoint/rate-limit/logged-out signal on a platform stops THAT platform
for the day; read every job's result for that signal before submitting the next job on the same
platform (`safety.md`). The three platforms interleave round-robin — one Facebook job, then one
Instagram job, then one X job, repeat — so a trip on one never stops the other two.

Note: this fixed order scans every readable group — public groups, and private groups the account is
already a member of (groups_readable) — with no separate join/approval needed for either, and no
approval needed to monitor it either: the Group Potential Rule registers `high`/`medium` potential
`state: active` automatically (see `safety.md`'s join boundary and
`playbooks/PRIVATE_SOURCE_GATE.md`'s reconciliation paragraph). A private group the account has not
joined is groups_no_access: never scanned, joined, or requested here — list it for the Boss, who can
join it in their own session if they want it monitored. Use Recipe D for the recurring shallow
monitoring shape once a group is registered `state: active`.

## Recipe B — Persona by occupation ("find realtors / loan officers"), Facebook + Instagram + X

> When the target IS the profession (e.g. you sell TO realtors), not the buyer. Same shape on all
> three platforms — occupation + place terms, per the "People search FB/IG/X" row of the keyword-
> bank channel table.

```text
1. Facebook: fb.people.search { query: "<occupation> <location>", max_pages: 4..8 }
      e.g. "realtor Westminster", "loan officer Orange County", "bao hiem"
   Instagram: ig.people.search { query: "<occupation> <location>" }
      e.g. "realtor Orange County"
   X:         x.people.search { query: "<occupation> <location>" }
      e.g. "loan officer Westminster", "môi giới nhà đất Cali"
      Occupation + place only here — profession jargon/hashtags belong to IG/X SEARCH (the search-
      posts capability), not people search.
2. Read the returned ProfileSummary[] rows (name/handle, url, subtitle/bio line, industry_hint
   where the platform provides one).
3. Run every row through `playbooks/LEAD_QUALIFICATION_RULE.md` Step 1 (WHO is this person) against
   the client's `buyer_profile.types`. A bio-only row has no post text for Steps 2/3, so
   `intent = none` by the rule's own bio-only clause; a `fit = high` row is still `warm` — never
   dropped for lacking a stated need (SKILL.md's "Classification (extractor tier)").
4. (optional) fb.groups.search for that profession's communities → fb.group.posts to see who is active.
5. `tool crm-store ... lead capture` records each row (person_type, sells_to_match, fit, fit_reason,
   intent, intent_reason, decision) with the profile URL; no contact scraping.
```

## Recipe C — Friend-of-friend by industry ("mine my network")

> Warm network: people connected to a seed profile, filtered by industry.

```text
1. fb.profile.friends { profile_url: "<seed profile>/friends", max_pages: 4..8 }  → ProfileSummary[]
2. For each friend, infer industry:
     - fast/free: the friend's name + vanity url + subtitle (e.g. "edsocalrealtor", "Loan Officer").
     - confirm: fb.people.search { query: "<friend name>" } → industry_hint.
     (fb.profile.about is NOT reliable via GraphQL — see the catalog note.)
3. Keep friends in the target industries (immigration / real estate / insurance / ...).
4. Stage 10 records the shortlist. Friend-of-friend one more level = repeat step 1 per kept friend
   (heavy — cap by safety.md; this can explode into thousands, so obey the volume budget).
```

## Recipe D — Watch a known group's fresh posts (recurring monitoring)

> The daily/recurring lead pass over already-monitored private groups.

```text
1. fb.group.posts { group_url: "<group>", max_pages: 2..3 }  (recurring = shallow; Stage 10: 5 scrolls/day)
2. Classify every post by the AUTHOR'S TYPE FIRST — `playbooks/LEAD_QUALIFICATION_RULE.md` Step 1
   (WHO is this person) against the client's `buyer_profile.types` — before reading the post for
   need/intent language. Then run Steps 2-3 (competitor / why-now) on the same post. A `fit = high`
   author with `intent = none` is still `warm`, never dropped for a routine post.
3. Store to the Stage 10 ledger; only NEW opportunities vs prior days (dedupe against history).
```

## Recipe E — Harvest a discovered thread (on the Boss's order only)

> Reopen ONE already-recorded `likely` (Step 5) thread and triage its comment authors into the CRM.
> Never run inside the daily pass or any recipe above — only as its own job, on an explicit order
> naming one `source_id`. Full contract: `playbooks/10_LEAD_COMPETITOR_DETECTION.md`, "Harvest a
> discovered thread (on the Boss's order only)".

```text
0. `tool harvest-thread prepare --pipeline DIR --client SLUG --id SOURCE_ID` does steps 1-3 below
   in one call — use it instead of hand-rolling them. It refuses to run unless the source's
   `status` is `approved` (or `harvested` with `--force`, for an explicit Boss-ordered re-harvest).
1. fb.post.comments / ig.post.comments / x.post.replies { post_url, paginate }
   — or reuse comments already captured in the inbox for that post when nothing needs re-fetching.
2. Dedupe rows by author (all of one author's comments become one row); drop rows that are empty,
   emoji-only, or "following"/"bump"/tag-only.
3. Write survivors to batch files of exactly 40 rows each
   (`history/YYYY-MM/harvest/{source_id}/batch_01.json …`).
4. For EACH batch file, spawn one classifier sub-agent on the LOWEST model available (Haiku on
   Claude, the smallest Codex model) applying `playbooks/COMMENT_TRIAGE_RULE.md` verbatim — one
   sub-agent per batch, never one sub-agent walking every batch serially. Write each sub-agent's
   keep/drop verdicts to a results directory as JSON, rows keyed by row `id`.
5. `tool harvest-thread ingest --pipeline DIR --client SLUG --id SOURCE_ID --results DIR` does
   steps 5-6 below in one call — use it instead of a manual `crm-store` loop: it deliberately does
   NOT pass the thread's `post_url` as every kept row's `crm-store` identity seed, because
   `contactFields()`/`addContact` would then merge multiple distinct commenters sharing one
   `post_url` into a single CRM contact (known bug); it attaches the post as an evidence hook
   separately instead. A manual loop calling `tool crm-store ... lead capture` per author would
   reproduce that merge.
5a. Kept authors → CRM lead with fit/intent tags and `source:thread:{id}`
   (`outreach/playbooks/13_CRM_CORE.md`) — warm by default, `intent` only when the comment itself
   stated one. Competitor/noise rows: drop. `medium` rows: drop from capture, list in the run
   report for a human override.
6. Source `status: harvested`, with `harvested_at`, `harvest_job_id` (= `source_id`),
   `leads_added`, `authors_seen` — written automatically by `ingest`. (Manual fallback only, never
   needed in the normal flow: `tool source-registry discovered mark-harvested`.) Never re-harvest a
   `harvested` source without a fresh, explicit Boss order for that same source — `ingest` itself
   refuses a second run without `--force`.
```

## Composing your own

If none of the above fits, compose from the catalog:

- **Where do they gather?** → `fb.groups.search` / `fb.people.search`.
- **What did they say?** → `fb.group.search_posts` (keyword) / `fb.group.posts` / `fb.profile.posts` / `fb.newsfeed`.
- **Who are they connected to?** → `fb.profile.friends`.
- **Need more results?** → same capability with a higher `inputs.max_pages` (cursor replay), not more scrolling.

Reels note: `fb.reels.feed` (beta) streams reel creators whose NAME often states
the trade (e.g. "Meres Mortgage", "Bao Hiem Kim Anh", "Nhà Đất Texas") — usable
to discover industry creators, but its caption/hashtag text is weak; treat the
creator name/url as the reliable signal.
