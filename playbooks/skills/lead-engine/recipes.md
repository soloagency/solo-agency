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

## Recipe A — Intent-in-community ("find people who need X")

> e.g. "find people who need insurance", "who is looking to buy a house in OC".

```text
1. fb.groups.search   { query: "<community/industry>", max_pages: 3 }
      → list candidate groups. Rank by relevance + size + locality.
2. HUMAN GATE (join): present the groups the human is NOT yet a member of, with a
   one-line reason each. The human joins the ones they want. Do NOT auto-join.
   (Only member-visible + public groups are scannable — see safety.md.)
3. For each accessible group:
     fb.group.search_posts { group_search_url: ".../groups/<id>/search/?q=<intent kw>", max_pages: 4..8 }
        intent keywords = in-market language, NOT just the industry noun:
        insurance → "cần mua bảo hiểm", "tư vấn bảo hiểm", "health insurance", "life insurance quote"
        real estate → "cần mua nhà", "cho thuê", "looking to buy", "first time buyer"
4. Stage 10 qualifies each post → keep direct_need / buying_trigger / pain_signal; dedupe by post URL.
5. Deepen the productive groups/keywords via higher max_pages until KPI or a safety stop.
```

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
