# Handoff — Facebook WRITE actions in the Local Collector

Written 2026-08-01 (react / comment / DM), updated 2026-08-15 (Task A + Task B landed,
extension `0.1.71-id-anchored-dm`). Task C stays blocked — see §4.

## What changed on 2026-08-15 (read this before §4)

Both open tasks are DONE and verified live on `aven-ngo`.

**Task A — target a post by CONTENT (`match_text`), option A.** `fb.post.comment` and
`fb.post.react` now accept `match_text` + `match_mode` (`contains` | `exact` | `regex`)
with `url` pointing at a GROUP or TIMELINE. Two phases in one job:

1. `window.__soloActResolve` (gql_actions.js) — a separate entry point from `__soloActRun`
   so it is structurally unable to write — reads the listing and filters IN CODE. Exactly
   one hit is required: 0 → `no_match`, >1 → `ambiguous_match`, and both write NOTHING and
   return `candidates` so the refusal is auditable.
2. `background.js` then navigates the SAME tab to the resolved permalink, re-injects, and
   writes there. It must never act on the listing — a feed carries one "Comment as …"
   composer PER POST.

Live: `"Post 5 post 5 post 5"` → `/posts/676092881161049/`, commented, `done`/`verified`.
`"Post"` → `ambiguous_match` over 3 candidates, nothing written.

**Three things measured that no amount of code-reading would have shown:**

- **The GraphQL listing is blind to the NEWEST posts.** Facebook server-renders the top of
  a feed into the first document and only FETCHES older pages, and `gql_intercept.js`
  hooks fetch/XHR. On the test group `fb.group.posts` returned Post 2 and Post 3 while
  "Post 5" — the target — was on screen the whole time. Several runs had
  `graphql_capture_count: 0` and the DOM source did all the work. So the resolver merges
  TWO sources and reports `sources_read: {graphql, dom}`. **Do not "simplify" it back to
  GraphQL only.**
- **Do not scrape the feed yourself.** A hand-rolled `[role="article"]` scan looked right
  and was wrong: Facebook renders each COMMENT as its own article, and a comment's
  permalink is the POST's url plus `?comment_id=…`, which survives the query strip. The
  scan produced the RIGHT url carrying a commenter's words. `filtering.js`
  (`SocialHtmlFilter.filterCurrentPage().posts`, ISOLATED world) already pairs each
  permalink with its BODY and stops at the first comment boundary — reuse it. The two
  worlds share only the DOM, so `background.js` relays the array in as `_dom_posts`.
- **The feed is often not rendered when the capture ends.** Back-to-back runs on the same
  group: one read three posts, the next read only the "Featured" card and refused. The
  resolve step now polls up to 6× with a nudge-scroll between tries. A slow render must
  cost time, not correctness.

**Task B — DM via `messenger.com/e2ee/t/<vanity_or_profile_id>`, identity anchored to the
ID.** The operator's correction drove this: *a display name is not an identity* — the same
profile is "Bob Nguyen" today and "Alex Nguyen" next week, and a name gate would refuse a
correct send. So:

- `recipient_name` is **no longer required**. The anchor is the id in the url, read BEFORE
  the E2EE gate is touched — that is the only moment it still exists, because passing the
  gate rewrites the address to `/e2ee/t/<thread_id>`.
- A name mismatch **no longer aborts** a send whose `id_verified` is true; it is recorded
  as `name_matched: false`.
- `id_verified: false` deliberately does NOT abort on its own. Once a thread is encrypted
  Messenger rewrites `/t/<profile_id>` → `/t/<thread_id>`, and from the url a legitimate
  rewrite and a wrong thread look identical. It only means the anchor is unavailable, and
  the NAME becomes the deciding guard again — a different thread is a different person.
- With neither a verified id nor a name → `unverified_recipient`, refuse.

Live: vanity sent (`done`/`verified`, `id_verified`, title agreed); profile id
`100026030446486` accepted with **no** `recipient_name`; a wrong name on an unverifiable id
refused with nothing typed.

**Gotcha the operator hit:** `1307949203416082` is a PHOTO id, not a profile. A non-thread
id opens a page with **no composer at all** (`composers_found: 0`) — which is the guard
working, not a bug. The real test profile is `100026030446486`.

Offline harnesses now live in the repo at `solo-agency-collector/tests/` (`test_gql_actions.js`,
`test_gql_extract.js` — 57 checks, plain `node`, no browser). Run them before spending an
extension reload; they caught the regex/dedupe/allowlist and streamed-chunk cases long before a
browser round-trip could.

**If a group scan ever comes back short, read `GRAPHQL_MAINTENANCE.md` §9.0 FIRST.** A scan
returning 2 of 6 posts cost four wrong diagnoses and a dozen reloads in one session. The causes
were: Facebook streams the reply in `@stream`/`@defer` chunks that must be ASSEMBLED (an
`@stream` path addresses a slot that does not exist yet, so walking the full path silently
discards the post); the pagination reply nests `page_info` somewhere else than the initial
query, so pagination had never once run; `"cursor": null` is not the same as omitting `cursor`;
and an anonymous post keeps its real author at an un-redacted nested path. §9.0 also records
the method lesson — print the shape of every chunk before theorising about why data is absent.

---

## Original 2026-08-01 handoff follows

Everything described as DONE is live on `origin/main` and in the published `dist` branch.

---

## 1. Where things are

| What | Path |
|---|---|
| **Canonical source** (edit HERE) | `/Users/binhnguyen/Downloads/soloagency/solo-agency-collector` |
| Write-action code | `chrome-extension/gql_actions.js` (MAIN world, injected only for write jobs) |
| Read capabilities | `chrome-extension/gql_extract.js` |
| Job orchestration, capture, guards | `chrome-extension/background.js` |
| Capability catalog (MCP-style) | `bridge-go/collector_capabilities.json` |
| Bridge (Go) | `bridge-go/` |
| Deploy script | `/Users/binhnguyen/Downloads/soloagency/deploy-soloagency.sh` |
| Test client build (aven-ngo) | `/Users/binhnguyen/Downloads/oneman_agency/extensions/aven-ngo` |
| Live production clone (codex) | `/Users/binhnguyen/Downloads/microagency/codex/solo-agency` |
| Collector output (inbox) | `/Users/binhnguyen/Downloads/microagency/codex/daily-content-pipeline/collector/inbox` |

`aven-ngo` is only an EXTENSION NAME — the Facebook account behind it is the operator's
own (`nguyenhuubinh`). That is why it can act on the operator's own private test targets.

Read `GRAPHQL_MAINTENANCE.md` (same folder) for how the GraphQL read layer works, and
`EMAIL_DISCOVERY.md` for the email-dig capability.

---

## 2. The build / test / ship loop (follow exactly)

```bash
# 1. edit canonical source, then syntax-check
node --check chrome-extension/gql_actions.js

# 2. bump the build tag + version (they are how you know a reload landed)
#    background.js:  const EXTENSION_BUILD = "0.1.NN-short-name";
#    manifest.json:  "version": "0.1.NN"

# 3. copy canonical -> oneman template, then rebuild the aven-ngo client
cd /Users/binhnguyen/Downloads/soloagency/solo-agency-collector/chrome-extension
for f in *.js *.json *.html *.md; do
  case "$f" in *_2026-*|*.bak|*.go.bak) continue;; esac
  cp "$f" /Users/binhnguyen/Downloads/oneman_agency/solo-agency-collector/chrome-extension/$f
done
cd /Users/binhnguyen/Downloads/oneman_agency/solo-agency-collector
bash scripts/prepare_client_extension.sh "Aven Ngo" aven-ngo ext_aven-ngo_default

# 4. ASK THE USER TO RELOAD the aven-ngo extension in Chrome, then confirm:
curl -s http://127.0.0.1:17321/status | grep -o '"extension_version":"[^"]*"'
#    do NOT test until the version matches what you just built

# 5. after live verification, push canonical (this also republishes the dist branch)
cd /Users/binhnguyen/Downloads/soloagency
bash deploy-soloagency.sh --collector-only
```

**Never** hand-edit files under `oneman_agency/extensions/<client>/` — they are generated.
Pass the existing `extension_instance_id` (`ext_aven-ngo_default`) as the 3rd argument or
routing identity changes.

### Running a job

```bash
curl -s -X POST http://127.0.0.1:17321/jobs/run_now \
  -H "Content-Type: application/json" -d '{
  "run_id":"test_'$(date +%H%M%S)'","client_slug":"aven-ngo","run_now_ttl_minutes":30,
  "pacing":{"scroll_steps":0,"min_delay_seconds":2,"max_delay_seconds":3,"max_sources":4},
  "sources":[{"name":"…","url":"…","platform":"facebook",
              "capability":"fb.post.comment","inputs":{"text":"…"}}]}'
```

Results land in `…/collector/inbox/YYYY-MM/<client>/<run_id>/private_data_points.jsonl`.
Poll `collector_status.json` → `.completed === true`, then read `records.items[0]`.

**Job gotchas that WILL bite you:**
- The bridge hard-caps `max_sources` at **20** (`bridge-go/main.go`, `clampInt(…,1,20)`),
  whatever the job asks for. Batch anything larger.
- Two sources with the **same url** in one job: the second is silently **deduped away**.
  Split them into separate jobs (this cost a guard test once).
- Jobs are serialized per client — a queued job waits for the running one.

---

## 3. What is DONE (verified live, both by hand and through real bridge jobs)

Three write capabilities, all `status: beta` in the catalog, all in `gql_actions.js`
behind `window.__soloActRun(capId, inputs)`:

### `fb.post.react`
Like/love a post. Idempotent (already reacted → no-op; it never un-reacts).

### `fb.post.comment`
```json
{ "url": "<POST PERMALINK>", "capability": "fb.post.comment", "inputs": { "text": "…" } }
```
Verified on all three surfaces: profile post permalink, group post permalink, reel.

How it works (do not "simplify" these — each line is a fixed bug):
- Composer selector is `div[contenteditable="true"][role="textbox"]` whose **aria-label
  starts with "Comment as "** (`COMMENT_AS`). Facebook uses that same label on posts,
  group posts and reels. The earlier loose `/comment/i` match also hit the SEARCH box and
  the status composer — it could comment in the wrong place. **No fallback guessing.**
- A **reel** has no composer until the `[role=button]` labelled exactly `"Comment"` is
  clicked (not "Comment with a GIF"), then you must WAIT for the composer to render.
- Type with `document.execCommand("insertText", false, text)` — the composer is Lexical
  and accepts it. `.value` does not exist on it.
- Submit with Enter dispatched as **keydown + keypress + keyup** (keypress was missing
  once and submission was flaky).
- Verify BOTH: the composer emptied AND the text is on the page. Either alone can lie.

### `fb.message.send`
```json
{ "url": "https://www.facebook.com/messages/t/<numeric_profile_id>",
  "capability": "fb.message.send",
  "inputs": { "recipient_name": "Bob Nguyen", "text": "…" } }
```
- The job url MUST be a THREAD. A **profile url is rejected** — and that is not paranoia:
  clicking "Message" on a profile leaves SEVERAL `"Write to <someone>"` composers in the
  DOM at once (the target plus every docked chat head), and the profile page does not
  expose its owner's name (its `h1` is "Notifications"). A live test aimed at a profile
  found the composer of **a third person**.
- `recipient_name` is REQUIRED. After the E2EE "Continue" gate the url becomes
  `/messages/e2ee/t/<thread_id>` — the profile id is gone, so the name in the composer's
  aria-label is the only remaining proof of who you are talking to.
- Guards, in order: exactly **one** composer (`ambiguous_composer` otherwise) → its
  `"Write to <name>"` must equal `recipient_name` (`recipient_mismatch` otherwise).
  **Nothing is typed when a guard fires.**

### Cross-cutting guard fields on every data point
| Field | Meaning |
|---|---|
| `url_drifted` | the page read is not the item requested (covers reel / watch / posts / story_fbid / permalink / fbid; profile urls are deliberately NOT pinned because `profile.php?id=` legitimately redirects to a vanity) |
| `landed_on_self` | Facebook fell back to the OPERATOR'S own profile — its contact details are the operator's, reject the record |
| `owner_resolution` | reels only: `resolved` \| `retryable_not_rendered` \| `n/a` — retry on the middle one, do not conclude "no owner" |

`wrapCap()` in `gql_actions.js` now always returns `available: true`. Do not "fix" that
back: `background.js` discards a record whose capability reports unavailable, which threw
away exactly WHY a write refused. Success/failure lives in `status` / `verified`.

---

## 4. The task list as it stood on 2026-08-01 (A and B are now done — see the top)

### Task A — DONE 2026-08-15 (option A was chosen). Kept below for the reasoning; see the top of this file for what shipped.

**The problem.** `fb.post.comment` only accepts a permalink that the CALLER already has.
Turning "the post that says X" into a permalink is currently done by the AGENT: it runs
`fb.group.posts` / `fb.profile.posts`, reads `records.items` (each item has `text` and
`url`), eyeballs the match, and passes the url on. That is agent judgement — it can match
the wrong post, pick the first one to save effort, or decline when unsure, and two agents
give two answers. The user explicitly wants this in code.

**Option A (recommended, one job):** teach `fb.post.comment` to resolve the target itself.
```json
{ "url": "https://www.facebook.com/groups/<gid>",
  "capability": "fb.post.comment",
  "inputs": { "match_text": "Post 5 post 5 post 5",
              "match_mode": "contains",      // contains | exact | regex
              "text": "the comment body" } }
```
Behaviour: run the existing GraphQL listing extractor → filter `items` by `text` **in
code** → if exactly ONE match, navigate to its `url` and comment there → if 0 or >1,
return `no_match` / `ambiguous_match` and comment on NOTHING. Echo back
`matched_post: {url, text}` for the audit trail.

Note it must NOT try to comment inline in the feed: a listing page holds one
`"Comment as …"` composer PER POST, so the first-match selector would comment on the
wrong one. Resolve to the permalink, navigate, then comment.

**Option B:** a separate `fb.post.find` capability that only returns the permalink. Fewer
moving parts per capability, but the agent still carries the url between two jobs.

The user picked **A**, and asked for it on `fb.post.react` too. Both shipped.

### Task B — DONE 2026-08-15, and then hardened further (identity moved off the NAME onto the ID). Kept below for the reasoning; see the top of this file.

The user found a better entry point than `facebook.com/messages/t/<numeric_id>`:

```
https://www.messenger.com/e2ee/t/<vanity OR numeric id>
```
Verified live with a vanity (`thaian.nguyen.731572`): the "Continue" gate appears, and
after clicking it the url becomes `/e2ee/t/<thread_id>` with **exactly one** composer
`"Write to Bob Nguyen"`. Three advantages over the current url:
1. accepts a **vanity** — no need to resolve a numeric profile id first;
2. messenger.com is a dedicated surface — no docked chat heads, no feed noise;
3. the page **title becomes `"<Name> | Messenger"`** — a second verification anchor to
   check alongside the composer's aria-label.

So: accept `messenger.com/e2ee/t/…` (and keep accepting the facebook.com form), and add
the title check as guard #3.

### Task C — video attachment: BLOCKED, do not retry blindly

Attaching a video FILE to a comment or DM does not work from a content script, and the
reason is a browser security boundary, not a missing trick:
- comment composer: `DataTransfer` **can** set `input.files`, but Facebook ignores the
  synthetic `change` event (no preview, no error);
- DM composer: `input.files` **cannot even be set** (`filesSet: 0`);
- synthetic `paste` is refused; synthetic `drop` IS received (preventDefault fires) but
  the file is ignored.
Synthetic events carry `isTrusted: false` and Chrome deliberately refuses file attachment
without a real user gesture.

Also learned: **facebook.com cannot fetch `http://127.0.0.1`** (CSP + Private Network
Access; adding `Access-Control-Allow-Private-Network` did not help). So any future design
must fetch the file in the **background service worker** and pass bytes into the page —
never fetch from the page.

Realistic paths if it is ever required: `chrome.debugger` + CDP `Input.dispatchDragEvent`
(trusted events, but Chrome then shows a permanent "being debugged" banner), or OS-level
automation of the native file picker. Both are big changes and need the user's explicit
approval. The cheap alternative that works TODAY: send a **link** to the video — that is
just text, and text already works.

---

## 5. Working rules the user cares about (learned the hard way this session)

1. **Inherit before you rewrite.** Before replacing something that works, find the
   existing version (including backups), diff it, and keep its proven behaviour. A
   rewrite of `setup_collector.sh` once dropped its kill-and-restart logic and broke the
   bridge start.
2. **Announce behaviour changes BEFORE making them.** Do not let the user discover a
   regression by running into it.
3. **Verify with data, not reasoning.** Every claim in this document was measured. When
   the user pushed back ("scroll doesn't change the contact UI"), the data proved them
   right and me wrong — re-measure instead of defending a hypothesis.
4. **Never `git add -A` untracked WIP** and never commit build artifacts: `dist/` is
   gitignored and the binaries ride the `dist` branch. History was once rewritten to
   remove 700 MB of committed binaries.
5. **Client extension builds must come from the FULL template** — never strip
   capabilities per client (see the rule in `AGENT_RUNBOOK.md`).
6. **Never touch the production clients** (`leadup`, `angela-do`, `binh-nguyen`) — not their
   extensions, not their client data. Clarified by the operator 2026-08-15: those are live
   field installs running as real users under a DIFFERENT agent (Codex), which owns
   rebuilding and reloading them. Develop in the canonical repo and verify on **`aven-ngo`**,
   the test client, and stop there. When a live client needs the new build, say so and let
   the operator route it — do not offer to do it. (Shared infrastructure — the bridge binary,
   `daily-content-pipeline/collector/collector_capabilities.json` — is genuinely shared;
   changes there need the operator's explicit go-ahead, which they give as a command to run.)

## 6. First moves in the new session

1. Read this file, then `git -C /Users/binhnguyen/Downloads/soloagency log --oneline -5`
   to confirm you are at/after `b821d11`.
2. Check the bridge and which build is loaded:
   `curl -s http://127.0.0.1:17321/status`
3. Ask the user to confirm **Task A option A or B**, then implement in
   `gql_actions.js` (comment path) and `gql_extract.js` (listing extractors already
   return `{text, url}` — reuse them, do not write a new scraper).
4. Ship Task B at the same time (small).
5. Test targets the user has authorised for write actions (private, theirs — do NOT use
   any other target):
   - post permalink: `facebook.com/nguyenhuubinh/posts/pfbid06qGgZmA56oPHk5Wem7U1WDm6aT6ZmpPPAyJVQHGgUn2gP8zgfuNpUMx834x9Sne1l`
   - reel: `facebook.com/reel/695865683585752/`
   - group: `facebook.com/groups/668676178569386` (contains posts "Post 5 post 5 post 5"
     → `…/posts/676092881161049/`, "Post 3: …" → `…/posts/669851668451837/`, "Post 2")
   - DM thread: `messenger.com/e2ee/t/thaian.nguyen.731572` (recipient name "Bob Nguyen")
