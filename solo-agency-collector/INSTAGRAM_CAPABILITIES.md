# Instagram capabilities — the caller contract

Five Local Collector capabilities read **Instagram** the way `fb.profile.enrich` reads Facebook
and `zillow.agents.list` reads Zillow: passively, from the operator's own logged-in web session,
by intercepting Instagram's own internal traffic instead of scraping HTML. This file is for the
agent that writes the CALLER (a playbook step, a harvest walker, a script): what to submit, what
comes back, what the caller owns, and what it must never do. The machine-readable spec is
`bridge-go/collector_capabilities.json` (`ig.profile.enrich`, `ig.profile.posts`,
`ig.search.posts`, `ig.people.search`, `ig.post.comments`, schemas `InstagramProfile`,
`InstagramPostRecord`, `InstagramCommentRecord`, `InstagramProfileSummary`); this document
explains how to use them and does not repeat every field.

| Capability | Reads | Returns | Status |
|---|---|---|---|
| `ig.profile.enrich` | ONE profile `instagram.com/<username>/` | one `InstagramProfile` record — bio, counts, address, emails/phones parsed from the bio | beta |
| `ig.profile.posts` | a profile's posts grid | `items[]` = `InstagramPostRecord`, paginated via cursor replay | beta |
| `ig.search.posts` | the Explore keyword-search Top results | `items[]` = `InstagramPostRecord`, paginated via cursor replay | beta |
| `ig.people.search` | the search box's own REST endpoints | `items[]` = `InstagramProfileSummary` (top matches, not a directory) | beta |
| `ig.post.comments` | one post's comments (REST) | `items[]` = `InstagramCommentRecord` | beta |

Code: `chrome-extension/platforms/instagram/ig_intercept.js` (MAIN world, `document_start`
content script on `instagram.com`, declared in `manifest.json`) +
`chrome-extension/platforms/instagram/ig_extract.js` (MAIN world, injected only for `ig.*` jobs
and dispatched through `window.__soloIgRun`), registered as an `instagram` module in
`chrome-extension/core/platform_registry.js` (capability prefix `ig.`, hosts
`instagram.com`/`www.instagram.com`) — the Facebook library
(`platforms/facebook/gql_extract.js`) and the Zillow library are both untouched.
`chrome-extension/platforms/instagram/ig_normalize.js` is the canonical-record step
(`__soloInstagramNormalize`, `importScripts`-loaded by `background.js` alongside
`fb_normalize.js` / `zillow_normalize.js`): after a capability returns, it maps today's `ig.*`
output into `records.canonical = {schema_version, kind, items[]}` per `core/schema.js`'s
contract, the same additive step §2.1 of `GRAPHQL_MAINTENANCE.md` describes for Facebook/Zillow.
Offline tests: `node solo-agency-collector/tests/test_ig_extract.js` (the extractors, all five
capabilities plus `_discover.ig` and the exposed `__soloIgInternals` helpers) and
`node solo-agency-collector/tests/test_ig_normalize.js` (the normalizer, on the pattern of
`tests/test_zillow_normalize.js`: the real extractor runs first over fixtures shaped like live
captures, and its real output is what gets normalized and schema-validated). Run both before
every sync.

Both the profile query and the posts/search/comments calls are read from Instagram's own private
web traffic — no public Graph API, no access token, nothing the operator's browser was not
already going to ask for.

---

## 1. Submitting a job

Same bridge, same job shape as every other capability.

```bash
curl -s -X POST http://127.0.0.1:17321/jobs/run_now -H "Content-Type: application/json" -d '{
  "run_id": "ig_profile_'$(date +%H%M%S)'", "client_slug": "<client_slug>", "run_now_ttl_minutes": 20,
  "pacing": {"scroll_steps": 0, "min_delay_seconds": 3, "max_delay_seconds": 5, "max_sources": 1},
  "sources": [{
    "name": "ig profile enrich",
    "url": "https://www.instagram.com/<username>/",
    "source_type": "public", "platform": "instagram",
    "capability": "ig.profile.enrich", "inputs": {}
  }]
}'
```

```bash
# posts grid — needs an active (foreground) tab; see §5
"sources": [{"name": "ig profile posts", "url": "https://www.instagram.com/<username>/",
             "source_type": "public", "platform": "instagram",
             "capability": "ig.profile.posts", "inputs": {"max_pages": 2}}]
```

```bash
# keyword search — the job's url IS the search page
"sources": [{"name": "ig search realtor",
             "url": "https://www.instagram.com/explore/search/keyword/?q=realtor",
             "source_type": "public", "platform": "instagram",
             "capability": "ig.search.posts", "inputs": {"query": "realtor", "max_pages": 2}}]
```

```bash
# people search — REST-backed, no page navigation needed
"sources": [{"name": "ig people search", "url": "https://www.instagram.com/explore/",
             "source_type": "public", "platform": "instagram",
             "capability": "ig.people.search", "inputs": {"query": "realtor", "count": 30}}]
```

```bash
# post comments — submit the post/reel url
"sources": [{"name": "ig post comments", "url": "https://www.instagram.com/p/<code>/",
             "source_type": "public", "platform": "instagram",
             "capability": "ig.post.comments", "inputs": {"max_comments": 30}}]
```

Rules the caller must respect:

- **The url is the request**, same convention as Facebook and Zillow. For `ig.profile.enrich` /
  `ig.profile.posts` it is the profile's ROOT url; for `ig.search.posts` it is the
  `/explore/search/keyword/?q=` page (or pass `inputs.query` and let the job open it); for
  `ig.post.comments` it is the post/reel permalink. `ig.people.search` does not navigate at all
  (§5) — any `instagram.com` page the session already has open is enough.
- **`platform: "instagram"`** in the source (otherwise the data point is labelled `web`).
- Do not set `graphql_capture: false` — capability dispatch is gated behind it, same as Facebook.
- `ig.profile.enrich` and `ig.people.search` are hideable (`needs_active_tab: false` in the
  registry) — the tab may stay backgrounded. `ig.profile.posts`, `ig.search.posts` and
  `ig.post.comments` need an ACTIVE tab: their queries only fire while the screen actually
  renders (Instagram, like Facebook, defers feed work on `document.visibilityState`).
- Jobs are serialized per client; the extension for `<client_slug>` must be loaded in that Chrome
  profile and be current (`chrome://extensions`) — a stale extension without
  `platforms/instagram/*.js` returns `records: null` for an `ig.*` capability it does not know.

## 2. Reading the result

`…/collector/inbox/YYYY-MM/<client_slug>/<run_id>/private_data_points.jsonl` — one line per
source. The typed output is `records` (`{capability, available, count, items[], version, …meta}`);
the lead record is `records.items[0]` for `ig.profile.enrich`, `records.items[]` for the rest.

`available` is `false` only for the one case Instagram itself makes unreadable — the operator's
own profile (below). Every other outcome is `available: true` with `count: 0` and a `reason` /
`error` explaining why, so `records: null` (not an empty envelope) is what actually signals a
collector problem: the extension did not run the capability at all (stale extension, injection
failure, or the dispatch timeout).

| Envelope field | Meaning |
|---|---|
| `available: false, reason: "self_profile"` | `ig.profile.enrich` resolved to the logged-in operator's own profile — Instagram's viewer id matched the profile id. Not an error; pick a different target. |
| `found: false, reason: "profile_query_not_captured"` | The profile page never fired `PolarisProfilePageContentQuery` within `ensure_tries`; a DOM-only fallback ran instead if the header rendered at all (`source: "dom"` — fewer fields, see `InstagramProfile` in the catalog). Persisting past one retry means the query renamed (§4) or the profile is gated (login wall, age gate). |
| `found: false, reason: "posts_query_not_captured"` | No `PolarisProfilePostsQuery` fired — private profile, zero posts, the tab was backgrounded, or the grid never rendered. |
| `found: false, reason: "search_query_not_captured"` | No `PolarisKeywordSearchExplorePageRelayQuery` fired — open the `/explore/search/keyword/?q=` url directly rather than relying on `inputs.query` alone. |
| `found: false, reason: "media_id_unknown"` | `ig.post.comments` could not resolve which media the shortcode in the url belongs to (the post-root capture never fired, or the shortcode in `post_url` does not match anything captured). |
| `status: "error"` | The extractor threw; `error` says what. |

## 3. Pagination (the caller owns it, same contract as Facebook)

`ig.profile.posts` and `ig.search.posts` **never scroll** — like every cursor-paginated Facebook
capability, they replay the captured GraphQL query with a new `after` cursor
(`inputs.max_pages`, default 1, max 40 for posts / 20 for search). `ig.post.comments` replays the
REST endpoint with `min_id` instead (`inputs.max_comment_pages`, default 1, max 20, up to
`inputs.max_comments` total rows, default 50). Every paginated envelope carries
`page_info: {end_cursor | next_min_id, has_next_page | has_more, resumable}` and
`stopped_because` (`page_cap_hit`, `no_new_items`, `max_comments`, `replay_failed_<status>`) —
read `resumable` before treating a short page as the end of the list, exactly as
`usage.pagination` in the catalog documents for Facebook's `max_pages` capabilities.

`ig.people.search` and `ig.profile.enrich` do not paginate at all: the former returns whatever
the search box itself returns in one call (§5), the latter is a single record.

## 4. Live-measured queries and endpoints

Measured 2026-09-10, logged-in web session, `en` locale (`ig_extract.js` header comment is the
source of record — re-verify there first if any of this drifts):

| Capability | Query / endpoint | doc_id (GraphQL) |
|---|---|---|
| `ig.profile.enrich` | `PolarisProfilePageContentQuery` → `data.user{…}` | `28036671149327607` |
| `ig.profile.posts` | `PolarisProfilePostsQuery` → `data.xdt_api__v1__feed__user_timeline_graphql_connection{edges[].node, page_info}`, 12/page | `38154989454116081` |
| `ig.post.comments` (post header) | `PolarisPostRootQuery` (opens the post; comments' own capture is REST, see below) | `29326377470285825` |
| `ig.search.posts` | `PolarisKeywordSearchExplorePageRelayQuery` (`/explore/search/keyword/?q=`) → `data.xdt_fbsearch__top_serp_graphql.edges[].node` | not pinned — read live via `docIdFor` (below); do not hardcode |
| `ig.people.search` | `GET /api/v1/web/search/topsearch/?context=blended&query=` (falls back from `/api/v1/users/search/?q=`) | n/a — REST, no doc_id |
| `ig.post.comments` | `GET /api/v1/media/<pk>/comments/?can_support_threading=true` | n/a — REST, no doc_id |

A GraphQL query's `doc_id` is not fixed in this module's source — it is read at **runtime** from
Instagram's own module registry, the same way the page itself resolves it:
`window.___xf("__debug").modulesMap["<QueryName>.graphql"].exports.params.id`. `ig_intercept.js`
wraps this as `store.docIdFor(queryName)` and `ig_extract.js`'s `_discover.ig` capability reports
it back as `doc_ids.profile` / `doc_ids.posts` for whichever queries have already been captured.
This is the fallback path used only when `ig_extract.js` needs to REPLAY a query that has not
been captured on the current screen yet (it never invents a request from scratch) — the normal
path is always "the same body Instagram's own client just sent, with only `variables` patched."

## 5. Limits (by design, measured 2026-09-10)

- **No public email or phone field on the profile query.** `PolarisProfilePageContentQuery`
  carries no contact fields at all; `InstagramProfile.emails` / `.phones` are parsed OUT OF the
  bio text (and, for emails, the bio-links) with plain regexes. A profile that keeps its contact
  info off the bio line returns empty arrays — that is not a parser miss, there is nothing to
  parse.
- **`ig.people.search` returns the search box's own top matches, not a directory.** It calls the
  same two endpoints the search box itself calls (`/api/v1/users/search/`, falling back to
  `/api/v1/web/search/topsearch/`); Instagram's ranking decides what "top" means, and there is no
  pager — `inputs.count` (max 50) only caps how many of THOSE rows are kept.
- **The interceptor must run at `document_start`.** Measured 2026-09-10: the profile being
  VIEWED is not embedded in the page's own `data-sjs` JSON (only the logged-in viewer's profile
  is) — it arrives later, by XHR, as `PolarisProfilePageContentQuery`. A content script injected
  any later than `document_start` risks missing that request entirely, which is why
  `ig_intercept.js` is a static `manifest.json` content script, not something injected per job.
- **`ig.profile.posts` / `ig.search.posts` / `ig.post.comments` need an active tab.** Instagram
  defers feed/search/comment work while the tab is not visible, same behaviour Facebook's feed
  readers have; `ig.profile.enrich` and `ig.people.search` do not need the screen to render at
  all and may run hidden.
- **No write capabilities today.** Read-only in both directions: nothing is liked, followed,
  commented, or messaged. If Instagram writes are ever added they get their own capabilities and
  their own `dry_run` healthcheck contract, exactly like `fb.post.react` / `fb.post.comment`.
- **`ig.post.comments.replies` is always `[]` today.** The REST endpoint this capability reads
  returns a flat comment list; reply threading (Facebook's `depth` > 0 case) is not walked yet.

## 6. Rate and pacing guidance

Nothing here has been live-verified against Instagram's own rate limiting the way the Zillow bot
check has (§5 of `ZILLOW_CAPABILITIES.md`) — treat every one of these as **beta, human-paced,
read-only** until a healthcheck baseline exists (§7):

- Keep Instagram jobs on ONE Chrome profile at **human pace** — seconds between page loads, not
  back-to-back requests. `ig.profile.posts` / `ig.search.posts` / `ig.post.comments` already wait
  800ms between their own internal cursor-replay legs (mirroring the Facebook/Zillow replay
  helpers); the CALLER is responsible for the pacing between separate JOBS on top of that.
- A Chrome profile that has never visited `instagram.com` should be warmed by the operator
  opening it once by hand, same guidance as Zillow.
- There is no known Instagram-side human gate (no PerimeterX-equivalent observed yet) — if one
  ever appears, wire it into `background.js`'s human-gate mechanism the way `zillowHumanGate`
  does, rather than inventing a second one.
- `ig.profile.posts` / `ig.search.posts` cursor replay already caps itself (`max_pages` ≤ 40 / 20)
  and stops on `no_new_items` — do not raise those ceilings casually; a page cap is what protects
  the session from looking like a scraper.

## 7. Fixtures

Declared in `bridge-go/healthcheck.go`'s `hcFixtureDocs` and
`examples/healthcheck_fixtures.example.json`:

| Fixture | Used by | What it is |
|---|---|---|
| `ig_canary_profile_url` | `ig.profile.enrich`, `ig.profile.posts`, `ig.post.comments` (chained) | ROOT url of an Instagram business/creator profile the operator may read, e.g. `https://www.instagram.com/<username>/` — read only |
| `ig_canary_profile_username` | `ig.profile.enrich` | the `<username>` part of `ig_canary_profile_url` |
| `ig_search_keyword` | `ig.search.posts`, `ig.people.search` | evergreen keyword, e.g. `realtor` |
| `ig_canary_post_url` | `ig.post.comments` (OPTIONAL) | permalink of a post the operator may read whose comments are VISIBLE; when set, the probe targets it (`chain.fixture_override`) instead of the chained most-commented post, which may hide its comments |

Same rule as every other platform: fixtures must be the operator's OWN targets (a profile they
are comfortable being read repeatedly), never a private third party's account — `count: 0` is
only a trustworthy failure signal when the fixture is guaranteed to have content.

### Comments hidden by the owner (measured 2026-09-10)

Two posts with `comment_count` 7 and 8 answered `comments: []` to the executor account, and
Instagram's own post page showed no comments either — the owner limits who sees them. The
endpoint itself works (another public post answered 15). `ig.post.comments` reports this as
`count: 0, found: false, reason: "comments_hidden"` with `comment_count` kept, and the
healthcheck probe turns it into WARN, not FAIL. A real breakage answers an `error` or no
`media_id`.

Hiding is per post, not per account: of five posts probed on 2026-09-10 (two profiles, 5–8
comments each) two answered their comments and three answered none, on the same profile. So the
chained "most-commented post" can land on a hidden one every day. Set the OPTIONAL fixture
`ig_canary_post_url` to a post whose comments are known to be visible and the probe targets it
instead (catalog 0.2.8, `chain.fixture_override`); leave it unset and the chain behaves as before.

## 8. Maintenance

- `tool healthcheck run --client <test client> --only ig.profile.enrich` (and the other four ids)
  proves each capability live and records the first PASS as the baseline, same as every other
  platform — see `HEALTHCHECK.md` §6 "Adding a capability".
- Dev loop: `solo-agency-collector/sync-dev-extension.sh` → operator reloads the aven-ngo
  extension (version must change) → submit jobs (§1) → read `records` (§2). The repo's
  `chrome-extension/` is never loaded into Chrome directly.
- If Instagram renames a query or moves a field, the fastest read is `_discover.ig`
  (`window.__soloIgRun("_discover.ig", {})`): it lists every capture's `queryName`/`docId` and,
  for a query matching a filter, a depth-limited skeleton of the actual response — the same
  discovery pattern `GRAPHQL_MAINTENANCE.md` §7 uses for Facebook. Compare against §4 above.
- Catalog: the running bridge serves its embedded catalog until a rebuild; to expose new/edited
  entries live before that, copy `bridge-go/collector_capabilities.json` next to the running
  `collector_config.json` (read fresh, no restart needed) — operator's call, it is shared
  infrastructure.
- Offline: `node solo-agency-collector/tests/test_ig_extract.js` (fixtures shaped exactly like
  the live-observed captures in `ig_extract.js`'s own header comment; asserts on
  `window.__soloIgInternals` — `postRecord`, `commentRecord`, `userRef`, `emailsIn`, `phonesIn`,
  `serpItems`, `connectionItems`, `parseCount`, `postUrl` — exposed for exactly this) and
  `node solo-agency-collector/tests/test_ig_normalize.js` (runs the real extractor over those
  same fixtures, then normalizes and schema-validates its real output, plus a sensitive-key
  sweep — the pattern `tests/test_zillow_normalize.js` set). Run both before every sync, same as
  the Zillow module's `§7 Maintenance`.
