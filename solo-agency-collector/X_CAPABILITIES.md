# X (Twitter) capabilities — caller contract

Status 2026-09-10: **phase 0 — built against X's documented web GraphQL shapes, NOT yet
measured live.** The module ships with `_discover.x` so the first live pass on the operator's
own X session confirms every query name and data path below; this file is rewritten with the
measured facts the same way `INSTAGRAM_CAPABILITIES.md` was. Until then every capability here
is `beta` at best and not in the catalog.

Module files: `chrome-extension/platforms/x/x_intercept.js` (MAIN world at document_start on
x.com / twitter.com — records X's own GraphQL calls `/i/api/graphql/<queryId>/<Operation>` and
the session headers needed to replay one), `x_extract.js` (capabilities, dispatched through
`window.__soloXRun`), `x_normalize.js` (canonical profile / post / comment). Registry entry:
`core/platform_registry.js` module `x` (hosts x.com, twitter.com and their www/mobile forms).

## 1. Capabilities (read, phase 0)

| id | url the job opens | reads | notes |
|---|---|---|---|
| `x.profile.enrich` | `https://x.com/<handle>` | `UserByScreenName` → `data.user.result` | bio, location, website (t.co expanded), bio links, counters, joined, verified (blue / legacy), professional category, affiliation; emails/phones parsed from the bio text — X renders no contact fields |
| `x.profile.posts` | `https://x.com/<handle>` | `UserTweets` (Posts tab) or `UserTweetsAndReplies` with `include_replies:true` | every page captured while the tab scrolled is merged; deeper pages by cursor replay (`max_pages`, `max_posts`) |
| `x.search.posts` | `https://x.com/search?q=<kw>&src=typed_query` (Top) or `…&f=live` (Latest) | `SearchTimeline` with `variables.product` Top / Latest | `mode: top|latest` picks the capture; both are real X products, Latest is the lead-finding one |
| `x.people.search` | `https://x.com/search?q=<kw>&src=typed_query&f=user` | `SearchTimeline` product People | full user records per row |
| `x.post.replies` | `https://x.com/<handle>/status/<id>` | `TweetDetail` (`variables.focalTweetId`) | focal post carried as `post`; items are the replies with `depth` 0 (direct) / 1 (in-thread); `replies_hidden` when the post counts replies but none are served |
| `x.timeline.home` | `https://x.com/home` | `HomeTimeline` (For you) / `HomeLatestTimeline` (Following) | `feed: for_you|following` |
| `_discover.x` | any x.com page | the capture ring | `inputs.query` = substring of an operation name → `deep_skeleton` (sensitive key names redacted) |

Envelope: `{ capability, available, count, items, version, found, source_query, pages_fetched,
page_info{end_cursor, has_next_page, resumable}, stopped_because }` — same family as the
Facebook, Instagram and Zillow modules. `stopped_because` values: `page_cap_hit`,
`no_new_items`, `replay_failed_<status>` (X refused the replayed query — the page-scroll
captures are still returned), `fetch_error`.

## 2. Records

- Post (`tweetRecord`): `id, url, actor{id, username, name, url, is_verified, is_private},
  text` (long posts use the note text), `created_time, lang, engagement{likes, comments, shares,
  views, reposts, quotes, bookmarks}, attachments[{type photo|video|gif, url, preview_url}],
  links[] (expanded), hashtags[], mentions[], conversation_id, in_reply_to_post_id,
  in_reply_to_username, is_repost, repost_of, is_quote, quote_of, post_id`.
- Profile (`userRecord`): `id, username, name, profile_url, bio, location, website, websites,
  emails, phones, follower_count, following_count, post_count, media_count, listed_count,
  created_at, joined_time, is_verified, verified_type, is_private, professional_type, category,
  categories, affiliation, profile_pic_url, pinned_post_ids`.
- Canonical mapping (`x_normalize.js`): profile ← enrich / people.search; post ← profile.posts /
  search.posts / timeline.home; comment ← post.replies (`post_ref` = the envelope's `post_id`).

## 3. Replay and session

A replayed query is the captured GET with only `variables.cursor` rewritten, sent with the
headers X's own client sent last (public web bearer, `x-csrf-token` from the `ct0` cookie,
`x-twitter-*` flags). X also stamps requests with a per-request `x-client-transaction-id`; if
it refuses a replay without one the capability stops with `replay_failed_<status>` and keeps the
pages the tab's own scrolling produced — so give the job `scroll_steps` when deeper pages
matter. The session material lives only on `window.__soloX.__auth` and is never written into a
record (`_discover.x` reports `session_headers_seen: true|false`, never the value).

## 4. Not built (see CAPABILITY_MATRIX.md)

Writes (`x.post.like`, `x.post.reply`, `x.post.publish`, `x.dm.send`) wait for the read baseline
and the Boss's go; followers / following lists are feasible but rate-limited; Communities are
being retired by X and are not planned.
