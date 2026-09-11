# Capability matrix — one standard, four platforms

The collector exposes the same *kinds* of capability on every platform (the "pillar" rows below),
but each platform's web client only offers some of them. An agent must never ask a platform for a
row this table marks impossible: the request cannot succeed, and the bridge would only report a
capability error after burning a tab and a human-paced wait. Read this file before planning any
cross-platform job; the per-platform contracts stay in `GRAPHQL_MAINTENANCE.md` (Facebook),
`INSTAGRAM_CAPABILITIES.md`, `ZILLOW_CAPABILITIES.md` and, when they exist, the X and LinkedIn
documents.

Measured 2026-09-10 on the aven-ngo dev client (catalog 0.2.9). "Feasible" means: doable from the
platform's normal desktop web client, logged in as the operator, at human pace, by intercepting the
platform's own traffic, replaying a captured query, reading the DOM, or driving the platform's own
composer. No official API, no paid access.

Legend: **built** = in the catalog and live-verified · **feasible** = the web surface exists,
nothing built yet · **risky** = works on web but the platform rate-limits, gates by relationship,
or needs heavy UI driving (file upload) · **partial** = only part of the Facebook meaning exists ·
**impossible** = the platform has no such concept or its web client does not expose it.

## 1. The matrix

| Pillar | What it means (Facebook id) | Facebook | Instagram | X | LinkedIn |
|---|---|---|---|---|---|
| feed.home | the logged-in home feed (`fb.newsfeed`) | built | feasible — `PolarisFeedTimelineRootV2Query` seen live on `/` | **built** `x.timeline.home` — HomeTimeline (For you) / HomeLatestTimeline (Following) | feasible — `/feed` |
| search.posts | keyword search over all public posts (`fb.search.posts`) | built (top / posts) | **built** `ig.search.posts` — top grid only, web has no "latest" tab | **built** `x.search.posts` — `mode top|latest` (`?f=live`), operators (`from:`, `since:`, `min_faves:`) work in the query; deeper pages by scrolling | feasible — content tab, sort relevance/date, date filter |
| search.people | keyword search returning people (`fb.people.search`) | built | **built** `ig.people.search` — the search box's top matches (about 5–30), no pager | **built** `x.people.search` — `?f=user`, full profile records with counters and category | feasible — people tab with location/company/title filters |
| search.groups | find groups with privacy + member count (`fb.groups.search`) | built | **impossible** — Instagram has no group entity; broadcast channels are not searchable | **sunset-risky** — X Communities were announced retired (2026-05-30) yet still resolve as of Aug 2026; do not build on them, verify live first | feasible — groups tab shows Public/Private badge + member count |
| group.posts | read posts inside one group (`fb.group.posts`) | built | **impossible** — no groups | **sunset-risky** — CommunityTweetsTimeline while it lasts | feasible — `/groups/<id>/` feed |
| group.search_posts | keyword search inside one group (`fb.group.search_posts`) | built | **impossible** — no groups | **impossible** — never existed even while Communities lived | feasible — search box inside the group |
| profile.header | name / handle / basic header (`fb.profile.header`) | built | **built** (inside `ig.profile.enrich`) | **built** (inside `x.profile.enrich`) | feasible — top card |
| profile.hovercard | cheap identity lookup: id + name (`fb.profile.hovercard`) | built | **covered** — no hover card on web; `ig.people.search` on the username returns pk + name + flags in one REST call | feasible — hover card + UserByScreenName | feasible — mini-profile / typeahead |
| profile.enrich | bio, category, location, links, counters (`fb.profile.enrich`) | built | **built** `ig.profile.enrich` — bio, category, bio links, business address, counters, is_business/is_verified | **built** `x.profile.enrich` — bio, location, website (t.co expanded), counters, joined, verified, professional category; fields moved out of legacy{} in 2026 | feasible — headline, about, location, counters |
| profile.dossier | the About lines: work, education, relationship (`fb.profile.dossier`) | built | **impossible** — no About tab; the "About this account" panel (join date, country, former usernames) is a different, threshold-gated thing (§3) | **partial** — professional category + affiliated-organisation badge only; no work/education | feasible — Experience + Education are the richest of all four |
| profile.contacts | emails / phones / websites published on the profile (`fb.profile.contacts`) | built | **partial** — the Polaris profile query carries NO email/phone field (measured live); today: bio text + bio links + business street/city/zip. Unverified lead: `/api/v1/users/web_profile_info/` may carry `business_email` / `business_phone_number` for business accounts | **impossible** — X never renders email/phone; only the website field and bio text | feasible — "Contact info" overlay (email/phone/website), visibility depends on connection degree |
| profile.friends | friend / follower / following list (`fb.profile.friends`) | built | risky — `/api/v1/friendships/<id>/followers/` + `/following/`, max_id pages, rate-limited; the interceptor's REST whitelist must add `friendships/` first | risky — Followers / Following ops, cursor pages; protected accounts hidden | **impossible for strangers** — full connection list only for 1st-degree or self; public follower count only |
| profile.posts | posts of a profile with author + engagement (`fb.profile.posts`) | built | **built** `ig.profile.posts` | **built** `x.profile.posts` — UserOriginalsTimeline (Posts) / UserTweetsAndReplies (`include_replies`) | feasible — `/in/<x>/recent-activity/all/` |
| profile.videos | videos of a profile / page (`fb.profile.videos`) | built | feasible — Reels tab, `PolarisProfileReelsTabContentQuery` seen live | feasible — `/media` tab (UserMedia, filter by media_type) | feasible — activity "Videos" filter |
| reels.feed | short-video feed / player (`fb.reels.feed`) | built | feasible — `/reels/` vertical feed | feasible — dedicated video tab | feasible — immersive video feed on web since Feb 2025 |
| post.comments | comments of one post with author identity (`fb.post.comments`) | built | **built** `ig.post.comments` — owner may hide comments (reason `comments_hidden`) | **built** `x.post.replies` — TweetDetail, depth 0/1, `replies_hidden` when X serves none | feasible — comments panel, "load more" |
| post.react | like / react (`fb.post.react`, write) | built | **built** `ig.post.react` — `POST /api/v1/web/likes/<media_id>/like/`, idempotent | **built** `x.post.like` — clicks the focal post's own Like control (X signs its own request), FavoriteTweet as proof | risky — reactions are the first thing LinkedIn's anti-automation watches |
| post.comment | write a comment (`fb.post.comment`, write) | built | **built** `ig.post.comment` — `POST /api/v1/web/comments/<media_id>/add/`, comment id back | **built** `x.post.reply` — the post page's own reply composer, CreateTweet as proof | risky — same |
| message.send | direct message to a person (`fb.message.send`, write) | built | **built** `ig.message.send` — drives the profile's Message button + thread composer; to a non-follower it lands in Requests | **built, blocked** `x.dm.send` — the profile's Message control + XChat conversation; blocked until the operator finishes XChat onboarding (PIN, X Number) once in the collector's Chrome; `dm_not_allowed` when the account's DMs are closed | risky — 1st-degree only; InMail is paid and capped |
| group.post | publish into a group (`fb.group.post`, write) | built | **impossible** — no groups | **sunset-risky** | risky — group share box |
| post.publish | publish on the operator's own timeline | **built** `fb.profile.post` — own profile root or home composer, optional audience public/friends/only_me set before typing | **not built** — the desktop Create flow REQUIRES an image or video; Instagram has no text-only post | **built** `x.post.publish` — home composer + Post, CreateTweet as proof; every post is public | risky — "Start a post" box |
| hashtag.posts | posts under a hashtag | not built (`/hashtag/<tag>` exists) | **covered** — `/explore/tags/<tag>/` now redirects to keyword search; use `ig.search.posts` with `#tag` | feasible — `search?q=%23tag` (a canned SearchTimeline) | feasible — `/feed/hashtag/<tag>/` |
| location.posts | posts tagged at a place | not built (place pages exist) | feasible — `/explore/locations/<id>/` | **impossible** — only the unreliable `near:` search operator remains | **impossible** — no place tagging |

Counts (of 23 pillar rows): Facebook 21 built + 2 not built; Instagram 5 built + 3 covered by a
built one + 8 feasible/risky/partial + 7 impossible; X 7 built (6 capabilities, header inside
enrich) + 7 feasible, 5 risky/partial, 4 impossible (three of them hinge on Communities, which
is being retired); LinkedIn 0 built, 17 feasible/risky, 3 impossible.

## 2. What this means for lead finding

- **Instagram** finds leads the way a human does: keyword search (top posts only), people search,
  a profile's grid and reels, a post's comments. There are no groups to mine and no way to sort
  search by newest. Authors come with every post and comment (`actor.username`, numeric id). The
  contact yield is lower than Facebook: bio text, bio links and a business address; no About tab.
  Outreach: comment and like are plain web calls; a DM to a stranger sits in Requests; a post
  needs media.
- **X** is the closest to Facebook for *search*: top and latest, operators, people, hashtags,
  plus post/reply/like/DM through one GraphQL surface. Communities are the group analog but X has
  announced their end; plan X without groups. Contacts: website and bio only.
- **LinkedIn** has the richest profiles (experience, education, contact overlay) and real groups,
  but every write action and every bulk read is an account-restriction risk. Treat it read-only,
  human-paced, and never build DM/posting there without the Boss's explicit decision.

## 3. Platform-only extras (no Facebook row; list them so agents know they exist)

| Platform | Capability idea | Lead value |
|---|---|---|
| Instagram | `ig.account.about` — the "About this account" panel: join month/year, country, former usernames, active ads | provenance check before outreach (threshold-gated, not on every account) |
| Instagram | `ig.profile.card` — the opt-in Profile Card | rarely filled; low |
| X | `x.post.quote` (who quoted a post), `x.post.repost` / `x.post.bookmark` (writes), `x.list.timeline` (curated lists = a free prospect list), `x.search.verified` (filter), `x.space.listen` | lists and quote-readers are strong intent signals |
| LinkedIn | `li.search.companies`, `li.company.posts`, `li.creator.follower_count` | company pages are the business-lead surface Facebook pages are |

## 4. How to keep this table honest

- Every row marked built points at a healthcheck probe; when a probe FAILs the row is broken, not
  "partially working" — report it.
- A row marked feasible/risky is a research claim (web sources + one live discovery pass on
  2026-09-10), not a promise: before building it, run `_discover.<platform>` on the page and
  confirm the query or endpoint named here still fires.
- When a platform removes a surface (X Communities is the live example), downgrade the row here
  and in the catalog in the same commit.
- The catalog should carry this mapping machine-readably (a `pillar` per capability and an
  `unsupported` list per platform with the reason) so `/capabilities` can answer "does platform P
  support pillar R" and the playbooks can refuse an impossible job before it is queued. That
  change is proposed, not made.

## 5. Reading a post's comments: what "all comments" actually means (measured 2026-09-11)

None of the three built `post.comments` capabilities drains a thread; each stops at a page or
row cap, and a job has one wall-clock budget. The honest signal is `stopped_because` +
`page_info.resumable` (Facebook: `by_post[].resumable`), never `count` alone.

| | `fb.post.comments` | `ig.post.comments` | `x.post.replies` |
|---|---|---|---|
| Defaults | 5 pages, 60 comments | 1 page, 50 comments | 1 page, 100 replies |
| Hard caps per job | 20 pages, 500 comments (global across posts and reply levels) | 20 pages; no row cap | 20 pages; no row cap |
| Replies under comments | opt-in `depth` ≤ 4, one request per parent per level | never (`reply_count` only; the platform total counts them) | depth label 0/1 only, collapsed "show more" never opened |
| Platform total in the result | no — join `engagement.comments` from the post record | `comment_count` | `reply_count` |
| Resume in a later job | `start_cursor` / `start_cursors` | not yet (`next_min_id` is reported, no input takes it) | not yet, and a stored cursor cannot be replayed (404) — a new job scrolls from the top |
| Time budget | `inputs.time_budget_ms`, default 50 s of the 60 s kill timer; `stopped_because: "time_budget"`, rows and cursor kept | same | same (every X read) |

For lead finding this is enough: the first few hundred top-level commenters of a post are the
people worth a look, and a thread of thousands is a job for several legs, not one. What must
never happen again is the pre-2026-09-11 loss: a walk that ran past the 60 s kill timer on a
slow network answered `count: 0` and the bridge could not tell it from an empty thread.

