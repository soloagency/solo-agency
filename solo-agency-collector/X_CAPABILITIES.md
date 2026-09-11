# X (Twitter) capabilities — caller contract

Measured live 2026-09-10 on the operator's own logged-in x.com session (aven-ngo 0.2.82): six
read capabilities, all answering with full data on the Boss's samples. Catalog 0.2.12 carries
them with daily healthcheck probes. Writes are not built (§5).

Module files: `chrome-extension/platforms/x/x_intercept.js` (MAIN world at document_start on
x.com / twitter.com — records X's own GraphQL calls `/i/api/graphql/<queryId>/<Operation>` and
the session headers a replay would need), `x_extract.js` (capabilities, dispatched through
`window.__soloXRun`), `x_normalize.js` (canonical profile / post / comment). Registry entry:
`core/platform_registry.js` module `x` (hosts x.com, twitter.com and their www/mobile forms).

## 1. Capabilities

| id | url the job opens | query (measured) | live result 2026-09-10 |
|---|---|---|---|
| `x.profile.enrich` | `https://x.com/<handle>` | `UserByScreenName` → `data.user.result` | business + personal profiles complete: bio, location, website, counters, category, verified |
| `x.profile.posts` | `https://x.com/<handle>` | `UserOriginalsTimeline` (Posts tab; `UserTweets` still accepted) / `UserTweetsAndReplies` with `include_replies` | 95 posts over 3 pages by scrolling |
| `x.search.posts` | `https://x.com/search?q=<kw>&src=typed_query` (Top) / `…&f=live` (Latest) | `SearchTimeline`, `variables.product` Top / Latest | Latest "realtor": 50 posts over 3 pages |
| `x.people.search` | `https://x.com/search?q=<kw>&src=typed_query&f=user` | `SearchTimeline` product People | 40 accounts over 2 pages, counters + category filled |
| `x.post.replies` | `https://x.com/<handle>/status/<id>` | `TweetDetail` (`variables.focalTweetId`) | 41 of 48 declared replies over 3 pages, depth 0/1 |
| `x.timeline.home` | `https://x.com/home` | `HomeTimeline` (For you, POST) / `HomeLatestTimeline` (Following) | 35 posts |
| `_discover.x` | any x.com page | the capture ring | `inputs.query` = operation-name substring → `deep_skeleton` (sensitive key names redacted); `replay_headers_seen` |

Envelope: `{ capability, available, count, items, version, found, source_query, pages_fetched,
page_info{end_cursor, has_next_page, resumable}, stopped_because }` — same family as the
Facebook, Instagram and Zillow modules. `stopped_because`: `page_cap_hit`, `no_new_items`,
`replay_failed_<status>`, `fetch_error`.

## 2. Measured shapes (what the extractor reads, defensively)

- **User** (`UserByScreenName` and every `user_results.result`): the profile fields are NOT in
  `legacy{}` any more — `core{screen_name, name, created_at}`, `profile_bio{description,
  entities{url.urls[], description.urls[]}}` (the site link expanded here), `website{url}`
  (t.co), `relationship_counts{followers, following}`, `tweet_counts{tweets, media_tweets}`,
  `action_counts{favorites_count}`, `avatar{image_url}`, `location{location}`,
  `verification{verified}`, `verification_info{is_identity_verified}`, `privacy{protected}`,
  `is_blue_verified`, `professional{professional_type, category[{name}]}`,
  `affiliates_highlighted_label`, `business_account`. `legacy{}` is still read as the fallback.
- **Tweet**: `rest_id`, `legacy{full_text, created_at, favorite_count, reply_count,
  retweet_count, quote_count, bookmark_count, lang, conversation_id_str,
  in_reply_to_status_id_str, entities, extended_entities}`, `core.user_results.result`,
  `views.count`, `note_tweet.note_tweet_results.result.text` (long posts),
  `TweetWithVisibilityResults{tweet}` unwrapped, `TweetTombstone`/`TweetUnavailable` skipped.
- **Timelines**: found by their `instructions[]` array (paths differ per query); entries from
  `TimelineAddEntries.entries[]`, `TimelineAddToModule.moduleItems[]`, `TimelinePinEntry`;
  items in `content.itemContent` or `content.items[].item.itemContent` (threads); the bottom
  cursor from a `TimelineTimelineCursor` entry with `cursorType` Bottom.

## 3. Pagination = scrolling, replay only as fallback

Measured: a replayed GET of a captured query answers **404** — X requires the page's own
per-request `x-client-transaction-id`, which the collector does not forge. So `max_pages > 1`
is served the way a human gets it: the extractor scrolls the tab to the bottom, waits for the
page's OWN next query to be captured (up to 6 s per page), merges it, and repeats; the replay is
attempted only when scrolling produced nothing. Consequences: timeline capabilities need the
foreground tab (`needs_active_tab`), `page_info.resumable` reflects the last bottom cursor, and
`stopped_because: replay_failed_404` on a page-1 result just means "no more pages came from
scrolling" — not a breakage.

## 4. Records

- Post (`tweetRecord`): `id, url, actor{id, username, name, url, is_verified, is_private}, text,
  created_time, lang, engagement{likes, comments, shares, views, reposts, quotes, bookmarks},
  attachments[{type photo|video|gif, url, preview_url}], links[] (expanded), hashtags[],
  mentions[], conversation_id, in_reply_to_post_id, in_reply_to_username, is_repost, repost_of,
  is_quote, quote_of, post_id, entry_id` (+ `depth` on replies).
- Profile (`userRecord`): `id, username, name, profile_url, bio, location, website, websites,
  emails, phones, follower_count, following_count, post_count, media_count, listed_count,
  likes_given_count, created_at, joined_time, is_verified, verified_type, identity_verified,
  is_private, is_business, professional_type, category, categories, affiliation,
  profile_pic_url, pinned_post_ids`. X renders no contact fields: emails/phones are parsed from
  the bio text only.
- Canonical (`x_normalize.js`): profile ← enrich / people.search; post ← profile.posts /
  search.posts / timeline.home; comment ← post.replies (`post_ref` = envelope `post_id`).

## 5. Not built (see CAPABILITY_MATRIX.md)

Writes (`x.post.like`, `x.post.reply`, `x.post.publish`, `x.dm.send`) wait for the Boss's go;
followers / following lists are feasible but rate-limited; Communities are being retired by X
and are not planned. Fixtures: `x_canary_profile_url`, `x_realtor_profile_url`,
`x_canary_post_url`, `x_search_keyword` (HEALTHCHECK.md §3).

## 6. Maintenance

- `tool healthcheck run --client <test client> --only x.profile.enrich` (and the other five ids).
- Offline: `node solo-agency-collector/tests/test_x_extract.js` (32 checks incl. the 2026 user
  shape) and `node solo-agency-collector/tests/test_x_normalize.js` (schema-validated).
- When a query renames or a field moves: `_discover.x` with `inputs.query` on the page, read the
  skeleton, update the regex / path in `x_extract.js` and this file in the same commit.
