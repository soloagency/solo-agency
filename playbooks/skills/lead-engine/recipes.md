# Lead Engine — Recipes

Ready-made capability sequences for common lead intents. Each recipe is a
starting plan for the gather loop in `SKILL.md` — adapt inputs to the real
persona/keywords/location, and always run under `safety.md` limits and Stage 10
qualification. Capability ids and inputs come from `GET /capabilities`; if a
capability's `status` is not `stable`/`beta`, skip or substitute it.

Keyword banks below are examples — build the real bank from the client's
audience pain points, buying triggers, and language (Stage 10 §Definitions), in
the audience's own language (Vietnamese / English / etc.).

---

## Recipe A — Facebook Discovery Pass (canonical; feed → people → groups → in-group)

> The canonical implementation of `playbooks/10_LEAD_COMPETITOR_DETECTION.md`'s Facebook Discovery
> Pass (step 11C of the daily run, and Setup Flow's first-run trigger). It is also the right shape
> for an open-ended "find people who need X" ask — e.g. "find people who need insurance", "who is
> looking to buy a house in OC" — the fixed order below IS the general answer to that request, not
> just the automated daily version of it.

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
        No post text exists, so classification uses subtitle + industry_hint + name/url only.
        Temperature defaults to `watch`; only becomes `warm` when the subtitle states the target role
        explicitly. Qualified rows → `tool crm-store ... lead capture` tagged source:people_search
        plus kw:{term}, exactly like every other lead.
3. THEN GROUPS
     fb.groups.search { query: "<discovery term>", max_pages: <=4 }
        Keep only privacy == "public". Empty/unknown privacy is NOT treated as private by default:
        spend one fb.group.posts { max_pages: 1 } header check, or skip the group when budget is
        tight. Rank survivors by member_count desc; prefer names/snippets matching the client's
        target location. Skip a group already in private_data_sources, and skip one this pass
        already scanned in the last 7 days (check history/YYYY-MM/facebook_discovery_shortlist.jsonl
        first — playbooks/08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md has the exact fields).
4. THEN IN-GROUP
     For the top public groups from step 3:
     fb.group.search_posts { group_search_url: ".../groups/<id>/search/?q=<intent kw>", max_pages: <=4 }
        Intent keywords come from `tool source-keywords ... plan` (seed the group's bank first with
        `seed --industry --market --lang` when it is empty, plus the client's setup seed file when
        one exists) — NOT the discovery term from step 1/2/3. In-market language, not the industry
        noun:
        insurance → "cần mua bảo hiểm", "tư vấn bảo hiểm", "health insurance", "life insurance quote"
        real estate → "cần mua nhà", "cho thuê", "looking to buy", "first time buyer"
5. Stage 10 qualifies EVERY post and person row from all four steps immediately → keep direct_need /
   buying_trigger / pain_signal (posts) or a qualifying subtitle (people); dedupe by post/profile URL.
   Then `tool crm-store ... lead capture` — even for a group not yet in private_data_sources.
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
— restated here only so the recipe is self-contained):

| | discovery terms | feed searches | people searches | group searches | new public groups | intent terms/group | total calls | spread |
|---|---|---|---|---|---|---|---|---|
| FIRST RUN | 3 | 3 | 3 | 3 | up to 4 | 3 | ≤ 21 | ≥ 4 hours |
| DAILY | 1 | 1 | 1 | 1 | up to 2 | 2 | ≤ 7 | across the run window |

Lead target: FIRST RUN is a floor of 10, not a stop — keep working the shortlist until the budget is
spent. Safety trip is unchanged and unforgiving: the first checkpoint/rate-limit/logged-out signal
stops the whole account for the day; read every job's result for that signal before submitting the
next one (`safety.md`).

Note: this fixed order targets PUBLIC groups — scanning one needs no join/approval (see `safety.md`'s
join boundary and `playbooks/PRIVATE_SOURCE_GATE.md`'s reconciliation paragraph). To hunt inside a
private group the human is already a member of, skip step 3's public-only filter and go straight to
step 4's `fb.group.search_posts` against that group, or use Recipe D for its recurring shallow
monitoring shape.

## Recipe B — Persona by name/occupation ("find realtors / loan officers")

> When the target IS the profession (e.g. you sell TO realtors), not the buyer.

```text
1. fb.people.search { query: "<occupation> <location>", max_pages: 4..8 }
      e.g. "realtor Westminster", "loan officer Orange County", "bao hiem"
2. Read ProfileSummary[] → industry_hint + subtitle often confirm the trade.
      Keep rows whose industry_hint / subtitle matches the target industry.
3. (optional) fb.groups.search for that profession's communities → fb.group.posts to see who is active.
4. Stage 10 records each as a lead/prospect with the profile URL; no contact scraping.
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

> The daily/recurring lead pass over already-approved private groups.

```text
1. fb.group.posts { group_url: "<group>", max_pages: 2..3 }  (recurring = shallow; Stage 10: 5 scrolls/day)
2. Stage 10 qualifies the fresh feed for direct/indirect need + competitor signals.
3. Store to the Stage 10 ledger; only NEW opportunities vs prior days (dedupe against history).
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
