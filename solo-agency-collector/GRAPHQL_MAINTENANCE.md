# Facebook GraphQL Capture — Maintenance & Extension Guide

> **Audience:** an engineer or AI agent maintaining or extending the collector's
> Facebook data capture. **If Facebook changed something and a capability broke,
> jump to [§7 "It broke — audit & fix"](#7-it-broke--audit--fix).**

This document is the single starting point for the GraphQL-based Facebook
collection layer. It explains the architecture, the exact (validated) field
paths each capability depends on, how to add a new screen, and — most
importantly — how to diagnose and repair a capability after Facebook rotates its
internal GraphQL schema (which it does periodically).

Last validated end-to-end: **2026-07-15** (extension build `0.1.23-discover`).

---

## 1. What this is (one paragraph)

The collector reads Facebook data by **passively intercepting Facebook's own
internal GraphQL traffic** in the page (the private `/api/graphql/` calls the
`facebook.com` web app makes for itself) — NOT the public Graph API, and no
access tokens. A MAIN-world content script hooks `fetch`/`XHR`, buffers the
responses, and per-screen **extractors** turn those responses into typed records.
It is a **hybrid**: GraphQL is the structured source where it works; the existing
HTML pipeline (`filtering.js`) remains the fallback for everything else. The
whole layer is **additive** — it only writes new `data_point` fields and never
changes existing ones.

A machine-readable **capability catalog** (MCP-style) lists what can be
collected, so playbook agents can discover which capability to call. Served at
`GET http://127.0.0.1:17321/capabilities`.

---

## 2. File map

Canonical source of truth is the **`soloagency`** tree (this repo). The live
clients run from a separate `oneman_agency` tree — see [§6 Deploy](#6-deploy).

| File | Role |
|---|---|
| `chrome-extension/gql_intercept.js` | **Interceptor.** MAIN world, `document_start`, on `*.facebook.com`. Hooks `fetch` + `XHR`, buffers the last 50 GraphQL request/response pairs into `window.__soloGql.captures`. Passive only — never replays, never sends anything. |
| `chrome-extension/gql_extract.js` | **Extractors + dispatcher.** Runs in MAIN world during a job. `window.__soloGqlExtract()` = generic best-effort + manifest. `window.__soloGqlExtractCapability(id, inputs)` = per-screen precise extractor via `CAPABILITY_EXTRACTORS`. **This is the file you edit to add/fix a screen.** |
| `chrome-extension/background.js` | Service worker. Injects the capture files, drives scrolling (`collectCleanPage`), then reads GraphQL: generic (`__soloGqlExtract`) + capability (`__soloGqlExtractCapability`) and writes new `data_point` fields. Key constant `EXTENSION_BUILD` (bump on each deploy so `/status` shows which build a client runs). |
| `chrome-extension/manifest.json` | Declares the MAIN-world `content_scripts` entry for `gql_intercept.js` (`run_at: document_start`, `world: MAIN`). |
| `bridge-go/collector_capabilities.json` | **The capability catalog** (English). `//go:embed`-ed into the bridge as the default; also copied next to the running config so it can be edited live. |
| `bridge-go/main.go` | The Go bridge. Serves the catalog at `GET /capabilities` (`handleCapabilities`, `resolveCapabilitiesPath`, `capabilitiesJSON`). |

### New `data_point` fields this layer adds (all additive, null when absent)
- `graphql_available` (bool), `graphql_capture_count` (int)
- `graphql_records` `{ posts, entities }` — generic best-effort
- `graphql_manifest` `[{ queryName, docId, variableKeys, count, skeleton }]` — the
  **shape map** of every GraphQL query seen on the screen (values stripped). This
  is the primary audit artifact.
- `capability` (string), `records` `{ capability, schema, source_query, count, items }` — typed per-screen output

---

## 3. End-to-end flow

```
job source { url, capability, inputs }
        │
        ▼
background.js opens a tab → gql_intercept.js (MAIN, document_start) hooks fetch/XHR
        │  user-like scrolling triggers Facebook's own GraphQL calls
        ▼
window.__soloGql.captures = [ { queryName, docId, variables, response }, … ]   (last 50)
        │
        ▼
background.js injects gql_extract.js (MAIN) → __soloGqlExtractCapability(capability, inputs)
        │  dispatcher → CAPABILITY_EXTRACTORS[capability](caps, inputs)
        ▼
records { schema, source_query, count, items:[typed] }  → data_point.records
        │
        ▼
Go bridge → inbox/YYYY-MM/<client>/<run_id>/private_data_points.jsonl
```

If no `capability` is set, or the extractor finds nothing, the HTML pipeline's
output (unchanged) is what you get.

---

## 4. The capability catalog

- **Served at** `GET /capabilities` (no token, GET only). Agents read `id`,
  `title`, `when_to_use`, `inputs`, `output_schema`, `status`. `_impl` is
  internal detail (query name, edge path, node field paths).
- **Where it lives / how to update:** edit `bridge-go/collector_capabilities.json`
  (canonical). The bridge serves `--capabilities-file` if set, else
  `collector_capabilities.json` next to `--config-file`, else the embedded copy.
  It is **read fresh per request**, so editing the deployed file updates
  `/capabilities` **without restarting the bridge**. (The embedded default only
  changes on a Go rebuild.)
  - Deployed copy path: `oneman_agency/daily-content-pipeline/collector/collector_capabilities.json`
- **`status` values:** `stable` = live-verified · `beta` = unit-verified, live
  pending · `planned` = not built · `not_graphql` = not served by GraphQL
  interception (server-rendered / interaction-gated), needs a DOM/HTML approach.

---

## 5. Capability reference (the validated shape map)

These are the exact GraphQL paths each stable extractor depends on. **When a
capability breaks, this table tells you what the response *used to* look like —
compare against a fresh capture (§7).** Extractor functions are in
`chrome-extension/gql_extract.js`.

### fb.group.posts → `extractGroupPosts` → PostRecord[]
- Query: `GroupsCometFeedRegularStoriesPaginationQuery`
- Edges: `data.node.group_feed.edges[].node`
- actor `node.actors[0].{id,name,url}` · text `node.comet_sections.content.story.message.text` · url `node.permalink_url` · post_id `node.post_id` · group `node.to` · attachments `node.attachments[]`
- reactions `node.comet_sections.feedback.story.story_ufi_container.story.feedback_context.feedback_target_with_context …reaction_count.count` (the `{count}` object = grand total; bare numbers under `top_reactions.edges` are per-emoji)
- comments `…feedback_target_with_context.comment_rendering_instance.comments.total_count`

### fb.group.search_posts → `extractGroupSearchPosts` → PostRecord[]
- Keyword search INSIDE one group. URL form: `facebook.com/groups/<id>/search/?q=<keyword>` (**not** `/search/posts/`).
- Query: `SearchCometResultsPaginatedResultsQuery` (the SAME shared SERP query as people/groups search). Scoped to `SearchComet`.
- Edges: `data.serpResponse.results.edges[]` → the post story is at `rendering_strategy.view_model.click_model.story`. That story node is **shape-identical to the group_feed node**, so it is mapped by the shared `postRecordFromStoryNode()` helper. Entity results (groups/people) have `view_model.profile` instead of a `click_model.story`, so requiring a story cleanly excludes them.

### fb.profile.friends → `extractProfileFriends` → ProfileSummary[]
- Query: `ProfileCometAppCollectionNonSelfFriendsListRendererPaginationQuery` (+ Self variant). Scoped to captures whose `queryName` contains **`Friends`** (because `pageItems` is a generic profile-collection container).
- Edges: `data.node.pageItems.edges[]`
- name `edge.node.title.text` · url `edge.node.url` · id `edge.node.node.id` · subtitle `edge.node.subtitle_text.text` · mutual parsed from subtitle

### fb.people.search → `extractPeopleSearch` → ProfileSummary[]
- Query: `SearchCometResultsPaginatedResultsQuery` (shared SERP). Scoped to `SearchComet`.
- Edges: `data.serpResponse.results.edges[]` → entity at `rendering_strategy.view_model.profile`, filter `__typename === "User"`
- subtitle `view_model.primary_snippet_text_with_entities.text` · mutual from `view_model.snippet_with_facepile.simple_text_with_entities.text` · `industry_hint` = keyword classifier over subtitle

### fb.groups.search → `extractGroupsSearch` → EntityRef[]
- Query: `SearchCometResultsPaginatedResultsQuery` (same SERP). Scoped to `SearchCometResultsPaginatedResults`.
- Edges: `data.serpResponse.results.edges[]` → `rendering_strategy.view_model.profile`, keep only Group (typename `Group`, or `loggedProfile.type === "group"`, or `node.role` ~ /group/, or a `facebook.com/groups/<id>` url)

### fb.profile.posts → `extractProfileTimeline` → PostRecord[]
- Query: `ProfileCometTimelineFeedRefetchQuery` (+ initial variant). Scoped by the **edge path**, not the name.
- Edges: `data.node.timeline_list_feed_units.edges[].node` — then reuses the exact same field helpers as `fb.group.posts` (`actorRef`, `deepText`, `postAttachments`, `postEngagement`).

### fb.newsfeed → `extractNewsfeed` → PostRecord[]
- Query: `CometNewsFeedPaginationQuery`
- Edges: `data.viewer.news_feed.edges[].node` — skips ads/suggestions (requires `comet_sections` + a story `message`); handles reshares via inner `comet_sections.content.story`.

### not_graphql (do NOT try to fix these as GraphQL)
- **fb.profile.about** — structured fields (work/education/city) are **server-rendered**, absent from all interceptable responses (confirmed via `_discover.deep`; `ProfileCometAppSectionFeedPaginationQuery` carries only nav + pageItems urls). Needs a dedicated About-tab **DOM parser**. *Workaround for inferring industry:* call `fb.people.search` with the person's name → occupation subtitle + `industry_hint`.
- **fb.post.comments** — first page of comments is server-rendered (RSC); only "load more" pagination uses GraphQL (`CometUFICommentsProviderQuery`), which needs enough comments to trigger. Needs HTML/RSC parsing or a pagination trigger.

---

## 6. Deploy

Two trees: **canonical = this `soloagency` repo**; **live = `oneman_agency`**.
Each live client is `oneman_agency/extensions/<client>/`, built from the template
`oneman_agency/solo-agency-collector/chrome-extension/` by
`oneman_agency/solo-agency-collector/scripts/prepare_client_extension.sh`.

### Extension change (extractors, interceptor, manifest)
1. Edit in `soloagency/chrome-extension/`. Bump `EXTENSION_BUILD` in `background.js` + `version` in `manifest.json`.
2. Sync changed files → `oneman_agency/solo-agency-collector/chrome-extension/` (the template). **Never leave backups inside that folder** — the build script copies everything and Chrome rejects `_`-prefixed names.
3. Rebuild the client, **preserving its identity** (pass the existing `extension_instance_id`):
   ```
   cd oneman_agency/solo-agency-collector
   bash scripts/prepare_client_extension.sh "Aven Ngo" aven-ngo ext_aven-ngo_default
   ```
4. User reloads that client at `chrome://extensions` (⟳). Confirm via
   `curl -s 127.0.0.1:17321/status | grep extension_version` → the new build tag.

### Bridge change (main.go)
Needs a **rebuild + restart** (heavier; briefly interrupts all clients):
`GOOS=darwin GOARCH=amd64 go build -o <bin>/collector-bridge-darwin-amd64 .` (all
3 clients are localhost profiles → only darwin-amd64 matters), back up the old
binary, swap, user re-runs `oneman_agency/solo-agency-local-collector/setup_collector.sh`.

### Catalog-only change
Just edit the deployed `collector_capabilities.json` (next to the config) — live,
no restart. Keep the canonical `bridge-go/collector_capabilities.json` in sync.

---

## 7. It broke — audit & fix

**Symptom:** a capability's `records` is `null`, `count: 0`, or fields are empty/wrong,
even though the screen has data. Almost always this means **Facebook renamed a
query or moved a field** — the extractor's hardcoded path no longer resolves.

**The doc_id / friendly-name are NOT hardcoded** anywhere (captured at runtime),
so those rotating never breaks us. What breaks us is a changed **response shape**
(edge path or node field path). Fix it data-first:

1. **Get a fresh manifest.** Run a plain discovery job to the screen's URL (no
   capability) and read `graphql_manifest` from the resulting
   `private_data_points.jsonl`:
   ```
   curl -s -X POST 127.0.0.1:17321/jobs/run_now -H 'Content-Type: application/json' -d '{
     "run_id":"audit_'$(date +%s)'","client_slug":"aven-ngo",
     "sources":[{"url":"<SCREEN URL>","platform":"facebook","purpose":"discovery"}],
     "pacing":{"scroll_steps":6}}'
   # then read inbox/YYYY-MM/aven-ngo/<run_id>/private_data_points.jsonl → .graphql_manifest[].queryName
   ```
   Requires: a client on the current build, logged in to Facebook. `manifest[].skeleton`
   is the response shape (values stripped) at moderate depth.

2. **Compare to §5.** Did the `queryName` change? Did the edge path move
   (`.edges` now under a different key)? Cross-check the `skeleton` against the
   documented edge path + field paths.

3. **For deep fields** (engagement, About-like nesting), use the built-in deep
   probe — capability `_discover.deep`, `inputs.query` = a friendly-name substring:
   ```
   "sources":[{"url":"<SCREEN URL>","platform":"facebook",
     "capability":"_discover.deep","inputs":{"query":"<QueryNameSubstring>"}}]
   ```
   `records.deep_skeleton` is a deep (budget 2500 / depth 18) shape dump; `records.queries`
   lists every captured query name. Walk it to find the new field path.

4. **Patch the extractor** in `gql_extract.js`: update `getPath(chunk, "<edge_path>")`
   and the per-field paths to match the fresh skeleton. Keep using the shared
   helpers (`getPath`, `deepFind(…, maxDepth)`, `coerceCount`, `actorRef`, …).

5. **Unit-test before deploy** (no browser needed) — build a mock from the
   skeleton shape and run it under node:
   ```js
   global.window = { __soloGql: { captures: [ { queryName:"<name>", response: <mock> } ] } };
   eval(require('fs').readFileSync('gql_extract.js','utf8'));
   console.log(window.__soloGqlExtractCapability('<capability-id>', {}));
   ```
   `node --check gql_extract.js` for syntax.

6. **Deploy** (§6), reload, **live-verify** with a real capability job, then flip
   the catalog `status` back to `stable` and update the paths in §5 + `_impl`.

---

## 8. Add a NEW screen/capability

Same loop, proven on 6 screens:
1. **Discovery run** to the screen URL (no capability) → read `graphql_manifest`
   to find the query + edge path. Use `_discover.deep` for deep field paths.
2. **Write** `function extractXxx(caps, opts)` in `gql_extract.js` mirroring
   `extractGroupPosts` / `extractProfileFriends` (ES5, `var`, defensive, dedupe by
   id, support `opts.debug` → `_debug_node_skeleton`). Return
   `{ capability, schema, source_query, count, items }`.
3. **Register** it: add `"fb.xxx": extractXxx` to `CAPABILITY_EXTRACTORS`.
4. **Add a catalog entry** in `collector_capabilities.json` (id, title,
   when_to_use, inputs, output_schema, `_impl.graphql_query`, `_impl.edge_path`),
   `status: "beta"`.
5. Unit-test (§7.5) → deploy (§6) → live-verify → `status: "stable"`.

> **Tip:** batch discovery of several screens in ONE collector run (multiple
> `sources`), then draft extractors in parallel. That is how the current 6 were
> built.

---

## 9. Two hard-won rules (do not relearn these)

1. **Never name an output field with a substring the bridge redacts.** The Go
   bridge's `sanitizeMap` redacts any key containing `auth`, `token`, `session`,
   `secret`, `password`, `otp`, `csrf`, `bearer`. The person who posts is called
   **`actor`**, never `author` (`author` contains `auth` → the value becomes
   `"[redacted]"`). This cost a debugging cycle.
2. **Deep data needs `maxDepth`.** Engagement/UFI counts sit ~12–16 levels deep,
   past the default `WALK_DEPTH` (8). Pass `maxDepth` (~16) to `deepFind` for
   those. `coerceCount` handles `number | "1,234" | {count: N}`.

---

## 10. Key helpers in `gql_extract.js`
`isObj` · `getPath(obj,'a.b.c')` · `firstString/firstNumber(obj,names)` ·
`deepFind(obj,names,ok,depth,maxDepth)` · `deepText(obj,depth)` ·
`coerceCount(v)` / `objCount(v)` · `shallowName/shallowUrl(node)` ·
`actorRef(actor)` · `postAttachments(node)` · `postEngagement(node)` ·
`groupRef(node)` · `parseMutual(subtitle)` · `skeletonize(v,depth,budget,maxDepth)` ·
`postRecordFromStoryNode(node)` (the shared story-node → PostRecord mapper used by
`fb.group.posts`, `fb.group.search_posts`; profile.posts/newsfeed have their own
variants — if the story node shape changes, fix it in one place here).

## 12. Deep pagination (cursor replay)

Facebook's search/list screens do **not** reliably load more results on passive
scroll (a keyword search often fires its paginated query exactly **once**). So we
don't depend on scroll: `window.__soloGqlPaginate(capabilityId, inputs)` (in
`gql_extract.js`) takes the `end_cursor` from a captured response and **replays**
the same persisted query — reusing its `doc_id` + `variables` + `fb_dtsg` + `av`
(captured by `gql_intercept.js`) via the pristine `window.__soloGql.origFetch` —
to pull page after page. Each replayed page is run through the **same capability
extractor**, so filtering/mapping is reused for free. `background.js` calls the
paginator (not the plain extractor) for capability jobs.

- **Control:** `inputs.max_pages` (default 8, max 40). Result carries
  `paginated`, `pages_fetched`, `added_by_pagination`.
- **Proven:** `fb.group.search_posts` "cho thue" → **66 posts across 10 pages**
  (vs 6 from the single natural capture).
- **To enable it for a new capability:** add an entry to `CAPABILITY_PAGINATION`
  with `scope` (query-name substring to find the replayable seed) and
  `pageInfoPath` (dotted path to the connection's `page_info` = `{has_next_page,
  end_cursor}`). Nothing else — the extractor is reused per page.
- **If pagination breaks** (0 added): check that `gql_intercept.js` still captures
  `fb_dtsg`/`av`; that `pageInfoPath` still resolves in a fresh skeleton (§7); and
  that the query still paginates on the top-level `cursor` variable. FB replay
  auth (`fb_dtsg`) is session-bound — it works because the replay runs in the
  logged-in collector tab.

## 11. Collector API cheatsheet
- `GET /status` — health + which build each client runs (no token)
- `GET /capabilities` — the catalog (no token)
- `POST /jobs/run_now` — enqueue `{ run_id, client_slug, sources:[{url,capability?,inputs?}], pacing }` (persistent mode; a matching client claims it)
- Results: `inbox/YYYY-MM/<client_slug>/<run_id>/private_data_points.jsonl` (no read endpoint — read the file). Write endpoints need `X-Collector-Token` (from `GET /jobs/current` → `collector_bridge.write_token`).
