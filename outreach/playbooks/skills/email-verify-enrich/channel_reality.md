# Channel reality — what is actually readable (MVP)

Be honest about what a stdlib agent with WebSearch/WebFetch (and a browser tool where noted) can
truly read. Promising a hook from a source you cannot read produces an embarrassing "personalized"
email (a months-old listing, a wrong person). Per hook type, use the reachable source + a fallback.

| Source | Readable now? | How | Notes |
|---|---|---|---|
| Personal website / blog | ✅ full | WebFetch | Best source for a real, current hook. |
| YouTube | ✅ title/description | WebFetch/WebSearch | |
| Instagram / X (public) | ⚠️ best-effort | WebSearch snippet; WebFetch often walled | Store URL if unreadable. |
| Zillow / realtor.com listings | ⚠️ snippet mainly | WebSearch snippet; direct fetch often bot-blocked | A "new listing" hook needs a **date in the snippet** — a dateless snippet may be months old; don't claim "just listed". |
| Google Business Profile reviews | ⚠️ JS-rendered | browser tool (Claude in Chrome), not WebFetch | A recent review is a strong hook when you can read it. |
| State license / registry | ⚠️ varies | WebFetch for GET pages; browser tool for ASP.NET/POST forms | Proves "still active"; some states expose a public email. |
| Brokerage roster (e.g. kw.com) | ⚠️ SPA | often an empty shell on WebFetch → browser tool | Presence on the roster ⇒ still there; email domain is a hint, not a guess. |
| Facebook header | ✅ via Local Collector | `fb.profile.header` (name, category, followers, verified, website) | Collector is LIVE (operator's logged-in Chrome, `127.0.0.1:17321`). Read the header FIRST to get the real name/category, THEN search — never from the URL slug. Bare `profile.php` (no `?id=`) resolves to the operator, not the lead. |
| Facebook posts/videos | ✅ via Local Collector | `fb.profile.posts` / `fb.profile.videos` (captions, view counts) | Logged-out fetch still walled — use the Local Collector, not WebFetch. Captions carry the person's own words + place names. |
| Facebook people-search | ✅ via Local Collector (GraphQL) | `fb.people.search` → read **`records.items`** (`ProfileSummary[]`); source `capability:"fb.people.search"`, `inputs.query` = **THE NAME ONLY** | **Query = name only (2–3 tokens).** Facebook ANDs every token, so appending company/city/state/industry matches NOBODY — measured: 1–3-token queries returned results 87% of the time, 6+ token queries **0/56**. The same lead found 0 as `Sheryl Lynn Fanning Waltersak Insurance Kenai AK` and 36 (incl. herself) as `Sheryl Lynn Fanning`. Use company/city/state to **FILTER `records.items`** (match `subtitle`), never to narrow the query. Take candidates ONLY from `records.items`; **NEVER** `entity_candidates`/`new_private_sources` (DOM — leaks the operator's open Messenger contact; one "Bob Nguyen" polluted 32 searches). `records` null = that query matched nobody → try the next rung of the ladder, then no-result. Never accept a DOM candidate. |
| Facebook post/permalink → OWNER | ✅ from the URL (no fetch) | parse the URL: `/<vanity>/posts/…`, `/<numeric>/posts/…`, `permalink.php`/`story.php?…&id=<numeric>` | **Do NOT call the collector for these.** Measured 19 seeds: URL-derived owner 11/11 correct; the collector's DOM `profile_candidates` on the same post pages 7/7 WRONG (unrelated profiles, even group URLs) and 4 pages timed out at 240s. |
| Facebook reel → OWNER | ✅ via Local Collector | open `/reel/<id>`, take the `sk=reels_tab` "See Owner" link | The reel URL carries no owner, so the collector IS needed here — and is reliable (8/8 verified, `url_drifted:false`). Reject any record with `url_drifted: true`; don't trust `profile_candidates[0]` blindly, prefer the `reels_tab` link. |
| LinkedIn | ❌ logged-out wall | store the URL only | Career-change hooks live here but are unreadable now. |

## Consequences to encode

- Realistically only **personal websites + occasional dated snippets + readable reviews** yield a
  verifiable fresh hook, so plan for a **~30–50% deep-personalization hit-rate**, not 100%. The
  campaign's volume assumptions and the `no_hook_fallback` exist because of this. (That rate is for
  Layer-B **recent activity** (#1); **reputation / positioning / scale** signals (#2–#4) are more
  often readable from the site/snippet/reviews and also clear the ≥1 Layer-B floor — so a lead is
  usually write-ready even when the freshest post is walled.)
- A source marked ❌ here must **never** produce a hook — store the profile URL for later and move
  on. Do not paraphrase a search snippet's guess about a Facebook post as if you read it.
- **Facebook candidates come from GraphQL `records.items` ONLY, never the DOM `entity_candidates`.**
  The DOM entity/candidate scan reads the whole logged-in page, so an open Messenger chat, a "People
  you may know" rail, or a suggested-friends widget leaks the operator's own contacts in as "leads".
  Trust `records` (the typed capability output); if it is null, the search failed — retry or record
  no-result, never fall back to the DOM list.
- When a source needs a browser tool and none is available in the run, record it as unreadable
  this pass (it may become a hook on a later refresh), not a fabricated hook.
