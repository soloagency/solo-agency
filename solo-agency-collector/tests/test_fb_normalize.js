// Offline harness for chrome-extension/platforms/facebook/fb_normalize.js — the canonical
// normalizer that reshapes today's fb.* capability output into the shared
// {schema_version, kind, items[]} contract (chrome-extension/core/schema.js's root.SoloSchema).
//
// Style follows tests/test_zillow_extract.js / tests/test_post_comments.js: fs.readFileSync +
// vm.createContext + "  ok"/"  FAIL" lines, exit 1 on any failure, "ALL N CHECKS PASSED" on none.
//
// Fixtures: wherever a real fixture-factory already exists in this repo's test suite
// (tests/test_gql_extract.js, tests/test_post_comments.js, tests/test_profile_dossier.js,
// tests/test_gql_actions.js), it is copied verbatim below and the REAL extractor/action is run
// over it, then normalized. For capabilities with no existing fixture (fb.profile.posts,
// fb.newsfeed, fb.profile.videos, fb.reels.feed, fb.groups.search*, fb.profile.friends,
// fb.people.search, fb.profile.hovercard, fb.profile.header, fb.profile.contacts,
// fb.post.react), a minimal capabilityResult is hand-built matching the EXACT field names/shape
// read directly out of chrome-extension/gql_extract.js / gql_actions.js (file:line cited at each
// fixture) — never invented. fb.groups.search DOES get a real run: it shares extractGroupsSearch's
// SearchComet envelope with fb.group.search_posts, so a minimal matching capture drives the real
// extractor.
const fs = require("fs");
const path = require("path");
const vm = require("vm");

const GQL_EXTRACT_SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "platforms", "facebook", "gql_extract.js"), "utf8");
const GQL_ACTIONS_SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "platforms", "facebook", "gql_actions.js"), "utf8");
const NORM_SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "platforms", "facebook", "fb_normalize.js"), "utf8");
const SCHEMA_PATH = path.join(__dirname, "..", "chrome-extension", "core", "schema.js");

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail) : "")); }
}

const CAPTURED_AT = "2026-09-09T12:00:00.000Z";

// Verbatim from tests/test_profile_dossier.js:269-282 — recursive scanner for any key whose name
// matches the bridge's redaction needle list (mirrors chrome-extension/core/schema.js's
// SENSITIVE_KEY_NEEDLES exactly).
const SENSITIVE = ["cookie", "token", "secret", "password", "passwd", "pwd", "otp", "authorization", "auth", "session", "bearer", "csrf", "xsrf"];
function sensitiveKeys(o, pathStr, out) {
  out = out || [];
  if (Array.isArray(o)) { o.forEach((v, i) => sensitiveKeys(v, pathStr + "[" + i + "]", out)); return out; }
  if (!o || typeof o !== "object") return out;
  Object.keys(o).forEach((k) => {
    const lk = k.toLowerCase();
    if (SENSITIVE.some((n) => lk.indexOf(n) !== -1)) out.push(pathStr + "." + k);
    sensitiveKeys(o[k], pathStr + "." + k, out);
  });
  return out;
}

// Every canonical item produced anywhere in this file is collected here for one final
// cross-cutting pass: no sensitive key anywhere, and (if schema.js exists yet) validateRecord().ok.
const ALL_ITEMS = [];
function collect(canonical) {
  if (canonical && Array.isArray(canonical.items)) canonical.items.forEach((it) => ALL_ITEMS.push(it));
  return canonical;
}

function loadNormalizer(ctx) {
  vm.runInContext(NORM_SRC, ctx);
  return ctx.__soloFacebookNormalize;
}

// ============================================================================================
// Harness 1 — tests/test_gql_extract.js style: a static `captures` array, __soloGqlExtractCapability.
// ============================================================================================
function makeGqlExtractCtx(captures) {
  const win = {};
  const ctx = {
    window: win,
    location: { href: "https://www.facebook.com/groups/000000000000000", origin: "https://www.facebook.com" },
    document: { querySelectorAll: () => [], querySelector: () => null, title: "", body: { innerText: "" } },
    setTimeout, clearTimeout, URL, console, MutationObserver: function () { this.observe = () => {}; this.disconnect = () => {}; },
  };
  ctx.globalThis = ctx;
  vm.createContext(ctx);
  vm.runInContext(GQL_EXTRACT_SRC, ctx);
  win.__soloGql = { captures: captures || [], origFetch: null };
  const normalize = loadNormalizer(ctx);
  return { ctx, normalize };
}

// tests/test_gql_extract.js:29-40 — a normal fb.group.posts story (author in top-level `actors`).
function namedStory(postId, text, name) {
  return {
    node: {
      __typename: "Story",
      id: "S:" + postId,
      post_id: postId,
      permalink_url: "https://www.facebook.com/groups/000000000000000/permalink/" + postId + "/",
      actors: [{ id: "724699549", name: name, url: "https://www.facebook.com/binh" }],
      comet_sections: { content: { story: { message: { text: text } } } },
    },
  };
}

// tests/test_gql_extract.js:45-72 — anonymous-post fallback (top-level `actors` scrubbed, real
// actor recoverable via the avatar-renderer path).
function anonymousStory(postId, text) {
  return {
    node: {
      __typename: "Story",
      id: "S:" + postId,
      post_id: postId,
      permalink_url: "https://www.facebook.com/groups/000000000000000/permalink/" + postId + "/",
      comet_sections: {
        content: { story: { message: { text: text } } },
        context_layout: {
          story: {
            comet_sections: {
              actor_photo: {
                story: {
                  actors: [{
                    id: "378233344858797",
                    name: "Anonymous member",
                    profile_picture: { uri: "https://scontent.example/a.jpg" },
                  }],
                },
              },
            },
          },
        },
      },
    },
  };
}

// tests/test_gql_extract.js:74-82 — wraps edges into the GroupsCometFeedRegularStoriesPaginationQuery envelope.
function capture(edges) {
  return {
    queryName: "GroupsCometFeedRegularStoriesPaginationQuery",
    docId: "1234567890",
    fbDtsg: "NAcM-fake",
    variables: { id: "000000000000000", count: 3 },
    response: [{ data: { node: { group_feed: { edges: edges, page_info: { end_cursor: null, has_next_page: false } } } } }],
  };
}

(function testGroupPosts() {
  console.log("\n-- fb.group.posts (real extractor) --");
  const named = namedStory("111", "Just listed! 3bd/2ba in Venice", "Jane Realtor");
  named.node.creation_time = 1757000000; // real unix-seconds epoch
  const anon = anonymousStory("222", "Anonymous listing tip");
  const { ctx, normalize } = makeGqlExtractCtx([capture([named, anon])]);

  const res = ctx.window.__soloGqlExtractCapability("fb.group.posts", {});
  check("extractor produced 2 posts", res.count === 2, res.count);

  const before = JSON.stringify(res);
  const out = collect(normalize("fb.group.posts", res, { captured_at: CAPTURED_AT }));
  check("input not mutated", JSON.stringify(res) === before);

  check("schema_version 1, kind post", out.schema_version === 1 && out.kind === "post", out);
  check("2 canonical posts", out.items.length === 2, out.items.length);

  const p1 = out.items.find((i) => i.platform_id === "S:111");
  check("platform/platform_id/url/source_capability", p1 && p1.platform === "facebook" && p1.url === named.node.permalink_url && p1.source_capability === "fb.group.posts", p1);
  check("actor mapped (id/name/url -> platform_id/name/url)", p1.actor && p1.actor.platform_id === "724699549" && p1.actor.name === "Jane Realtor", p1.actor);
  check("created_at is ISO from creation_time seconds", p1.created_at === new Date(1757000000 * 1000).toISOString(), p1.created_at);
  check("platform_time carries the raw seconds value unconverted", p1.platform_time === 1757000000, p1.platform_time);
  check("engagement is null (no UFI node in this fixture)", p1.engagement === null, p1.engagement);
  check("group_ref is null (no node.to on a plain group post fixture)", p1.group_ref === null, p1.group_ref);
  check("feedback_id lands in refs, not at top level", p1.refs.feedback_id === "" && !("feedback_id" in p1), p1.refs);
  check("comment_intent lands in ext", p1.ext.comment_intent === "", p1.ext);

  const p2 = out.items.find((i) => i.platform_id === "S:222");
  check("the anonymous post's real actor id is recovered via the avatar path", p2.actor && p2.actor.platform_id === "378233344858797", p2.actor);
  // postRecordFromStoryNode()'s created_time ALWAYS carries a number — `firstNumber(...) || 0`
  // (gql_extract.js:518) — so "not found" is the sentinel 0, never an absent field. created_at
  // treats that sentinel as null (no real timestamp), but platform_time is the RAW number
  // unconverted, so it stays 0, not null.
  check("no real created_time on this fixture -> created_at null, platform_time is the raw 0 sentinel", p2.created_at === null && p2.platform_time === 0, p2);
})();

// ============================================================================================
// Harness 2 — tests/test_post_comments.js style: live origFetch stub + require() registry, so
// pagination and reply-fetching actually walk.
// ============================================================================================
const COMMENT_Q = "CommentsListComponentsPaginationQuery";
const REPLY_Q = "Depth1CommentsListPaginationQuery";
const SEARCH_Q = "SearchCometResultsPaginatedResultsQuery";

function makePostCommentsCtx(opts) {
  opts = opts || {};
  const sent = [];
  const registry = opts.registry || { [COMMENT_Q]: "doc_comments", [REPLY_Q]: "doc_replies", [SEARCH_Q]: "doc_search" };
  const providers = opts.providers || {};

  const ctx = {
    window: {
      require: (name) => {
        if (name === "WebPixelRatio") return { get: () => 2 };
        const m = /^(.*)\.graphql$/.exec(name);
        if (!m) throw new Error("module not found: " + name);
        const id = registry[m[1]];
        if (!id) throw new Error("module not found: " + name);
        const out = { params: { id: id } };
        if (providers[m[1]]) out.params.providedVariables = providers[m[1]];
        return out;
      },
    },
    location: { href: "https://www.facebook.com/groups/1/search/?q=agent", origin: "https://www.facebook.com", pathname: "/groups/1/search/", search: "?q=agent" },
    document: { title: "", body: { innerText: "", innerHTML: "" }, querySelector: () => null, querySelectorAll: () => [] },
    setTimeout, clearTimeout, URL, URLSearchParams, console, Date,
    MutationObserver: function () { this.observe = () => {}; this.disconnect = () => {}; },
  };
  ctx.globalThis = ctx;
  vm.createContext(ctx);
  vm.runInContext(GQL_EXTRACT_SRC, ctx);

  const seed = {
    queryName: opts.seedQuery || SEARCH_Q, docId: "doc_search", fbDtsg: "DTSG", av: "100", url: "/api/graphql/",
    variables: { count: 5, cursor: "s:1" },
    response: opts.seedResponse || null,
  };
  ctx.window.__soloGql = {
    captures: [seed],
    parseResponse: (t) => JSON.parse(t),
    origFetch: (url, o) => {
      const p = new URLSearchParams(o.body);
      const query = p.get("fb_api_req_friendly_name");
      const vars = JSON.parse(p.get("variables"));
      sent.push({ query, vars, doc_id: p.get("doc_id") });
      if (opts.gqlError && query === COMMENT_Q) {
        return Promise.resolve({ text: () => Promise.resolve(JSON.stringify({ data: { node: null }, errors: [{ message: opts.gqlError }] })) });
      }
      let body;
      if (query === COMMENT_Q) body = commentReply(vars.commentsAfterCursor);
      else if (query === REPLY_Q) body = replyReply(vars.id);
      else body = opts.seedResponse || searchResponse([]);
      return Promise.resolve({ text: () => Promise.resolve(JSON.stringify(body)) });
    },
  };
  const normalize = loadNormalizer(ctx);
  return { ctx, sent, seed, normalize };
}
const call = (h, inputs) => h.ctx.window.__soloGqlPaginate("fb.post.comments", inputs);

// tests/test_post_comments.js:32-34
function author(n) { return { id: "u" + n, name: "Person " + n, url: "https://www.facebook.com/p" + n }; }

// tests/test_post_comments.js:38-51 — the core comment fixture, including replies/pagination
// metadata (feedback.replies_fields.total_count, feedback.expansion_info.expansion_token).
function commentNode(id, n, opts) {
  opts = opts || {};
  const node = {
    id: id,
    author: author(n),
    body: { text: "comment " + id },
    created_time: 1700000000 + n,
    depth: opts.depth || 0,
    url: "https://www.facebook.com/c/" + id,
    feedback: { id: "cfb:" + id, replies_fields: { total_count: opts.replies || 0 } },
  };
  if (opts.replies) node.feedback.expansion_info = { expansion_token: "xt:" + id };
  return node;
}

// tests/test_post_comments.js:53-61
const TOP_PAGES = [
  { edges: [{ node: commentNode("c1", 1, { replies: 2 }) }, { node: commentNode("c2", 2) }], next: "cc:1" },
  { edges: [{ node: commentNode("c3", 3) }, { node: commentNode("c4", 4) }], next: "cc:2" },
  { edges: [{ node: commentNode("c5", 5) }], next: null },
];
const REPLIES = {
  "cfb:c1": [{ node: commentNode("r1a", 11, { depth: 1, replies: 1 }) }, { node: commentNode("r1b", 12, { depth: 1 }) }],
  "cfb:r1a": [{ node: commentNode("r2a", 21, { depth: 2 }) }],
};

// tests/test_post_comments.js:62-74
function commentReply(cursor) {
  const i = cursor == null ? 0 : parseInt(String(cursor).split(":")[1], 10);
  const p = TOP_PAGES[i] || { edges: [], next: null };
  return { data: { node: { comment_rendering_instance_for_feed_location: { comments: {
    edges: p.edges, page_info: { end_cursor: p.next, has_next_page: !!p.next },
  } } } } };
}
function replyReply(commentId) {
  return { data: { node: { replies_connection: {
    edges: REPLIES[commentId] || [],
    page_info: { end_cursor: null, has_next_page: false },
  } } } };
}

// tests/test_post_comments.js:77-107 — a search-result post/story node carrying UFI (engagement
// + comment intent + feedback id), used to feed fb.group.search_posts (and, here, fb.groups.search).
function storyNode(id, feedbackId, nesting) {
  const ufi = {
    id: "ufi:" + id,
    comment_rendering_instance: { comments: { total_count: 5 } },
    reaction_count: { count: 9 },
    comment_list_renderer: { feedback: { comment_rendering_instance_for_feed_location: {
      selectable_intents: [
        { intent_token: "RANKED_FILTERED_INTENT_V1" },
        { intent_token: "RANKED_UNFILTERED_INTENT_V1" },
      ],
    } } },
  };
  const ctxLayer = { feedback_context: { feedback_target_with_context: ufi } };
  const feedback = nesting === "ufi_container"
    ? { story: { comet_feed_ufi_container: { story: { story_ufi_container: { story: ctxLayer } } } } }
    : { story: { story_ufi_container: { story: ctxLayer } } };
  return {
    id: id,
    post_id: id,
    permalink_url: "https://www.facebook.com/groups/g/posts/" + id + "/",
    actors: [{ id: "a" + id, name: "Author " + id, url: "https://www.facebook.com/a" + id }],
    comet_sections: {
      feedback: feedback,
      content: { story: { message: { text: "post " + id }, feedback: { id: feedbackId } } },
    },
  };
}
// tests/test_post_comments.js:108-113
function searchResponse(stories) {
  return { data: { serpResponse: { results: {
    edges: stories.map((s) => ({ rendering_strategy: { view_model: { click_model: { story: s } } } })),
    page_info: { end_cursor: null, has_next_page: false },
  } } } };
}

(function testGroupSearchPosts() {
  console.log("\n-- fb.group.search_posts (real extractor) --");
  const h = makePostCommentsCtx({ seedQuery: SEARCH_Q, seedResponse: searchResponse([storyNode("P1", "fb:P1")]) });
  const res = h.ctx.window.__soloGqlExtractCapability("fb.group.search_posts", {});
  check("extractor found the post", res.count === 1 && res.items[0].feedback_id === "fb:P1", res);

  const before = JSON.stringify(res);
  const out = collect(h.normalize("fb.group.search_posts", res, { captured_at: CAPTURED_AT }));
  check("input not mutated", JSON.stringify(res) === before);

  const it = out.items[0];
  check("refs.feedback_id carries the story's feedback id", it.refs.feedback_id === "fb:P1", it.refs);
  check("engagement.likes from reaction_count.count, .comments from comments.total_count", it.engagement.likes === 9 && it.engagement.comments === 5, it.engagement);
  check("engagement.shares/views/saves default to 0 when the UFI node was found", it.engagement.shares === 0 && it.engagement.views === 0 && it.engagement.saves === 0, it.engagement);
  check("comment_intent picks the token containing 'unfiltered'", it.ext.comment_intent === "RANKED_UNFILTERED_INTENT_V1", it.ext);
})();

(function testGroupsSearch() {
  console.log("\n-- fb.groups.search (real extractor) --");
  // extractGroupsSearch() (gql_extract.js:694-757) reads the SAME SearchComet envelope as
  // fb.group.search_posts, filtered to edges whose rendering_strategy.view_model.profile (or
  // .loggedProfile) looks like a Group (edgeIsGroup(), :683-692) — here __typename:"Group" is
  // the qualifying signal.
  function groupEdge(id, name, url) {
    return { node: {}, rendering_strategy: { view_model: { profile: { __typename: "Group", id: id, url: url, name: name } } } };
  }
  const resp = { data: { serpResponse: { results: {
    edges: [groupEdge("1234567890", "LA Realtors Network", "https://www.facebook.com/groups/1234567890/")],
    page_info: { end_cursor: null, has_next_page: false },
  } } } };
  const h = makePostCommentsCtx({ seedQuery: SEARCH_Q, seedResponse: resp });
  const res = h.ctx.window.__soloGqlExtractCapability("fb.groups.search", {});
  check("extractor found the group", res.count === 1, res);

  const out = collect(h.normalize("fb.groups.search", res, { captured_at: CAPTURED_AT }));
  check("schema_version 1, kind group", out.schema_version === 1 && out.kind === "group", out);
  const g = out.items[0];
  check("group platform_id/name/url/type", g.platform_id === "1234567890" && g.name === "LA Realtors Network" && g.url === "https://www.facebook.com/groups/1234567890/" && g.type === "group", g);
})();

function testPostCommentsRecursiveAndPagination() {
  console.log("\n-- fb.post.comments (real extractor, nested replies + pagination) --");
  const h = makePostCommentsCtx({});
  return call(h, { feedback_id: "fb:P1", depth: 3 }).then((res) => {
    // res.count is postComments()'s FLAT `all.length` — 5 top-level comments (c1-c5, walked
    // across the 3 TOP_PAGES) PLUS 3 replies (r1a, r1b, r2a) it also pushes into that same flat
    // array (gql_extract.js:3610-3611), so 8, not 5. The canonical normalizer must NOT treat
    // every flat entry as an independent root — see the top_level:false filter it applies.
    check("extractor's flat count is 5 top-level + 3 replies = 8", res.count === 8, res.count);
    const before = JSON.stringify(res);
    const out = collect(h.normalize("fb.post.comments", res, { captured_at: CAPTURED_AT }));
    check("input not mutated", JSON.stringify(res) === before);
    check("schema_version 1, kind comment, 5 top-level canonical items (no flattening)", out.schema_version === 1 && out.kind === "comment" && out.items.length === 5, out.items.length);

    const c1 = out.items.find((c) => c.platform_id === "c1");
    check("post_ref (opaque string handle) is the PARENT POST's feedback id", c1.post_ref === "fb:P1", c1.post_ref);
    check("own_ref (opaque string handle) is the COMMENT's OWN feedback id (used to fetch its replies)", c1.own_ref === "cfb:c1", c1.own_ref);
    check("actor mapped (author() -> actor{platform_id,name,url})", c1.actor.platform_id === "u1" && c1.actor.name === "Person 1", c1.actor);
    check("text carried through", c1.text === "comment c1", c1.text);
    check("created_at is ISO from created_time seconds (1700000001)", c1.created_at === new Date(1700000001 * 1000).toISOString(), c1.created_at);
    check("platform_time carries the raw seconds unconverted", c1.platform_time === 1700000001, c1.platform_time);
    check("replies recursively normalized: c1 has r1a, r1b", c1.replies.map((r) => r.platform_id).sort().join(",") === "r1a,r1b", c1.replies.map((r) => r.platform_id));
    check("reply depth carried through", c1.replies.every((r) => r.depth === 1), c1.replies.map((r) => r.depth));

    const r1a = c1.replies.find((r) => r.platform_id === "r1a");
    check("nested reply-of-a-reply recursion: r1a carries its own reply r2a", r1a.replies.length === 1 && r1a.replies[0].platform_id === "r2a", r1a.replies);
    check("r2a depth is 2", r1a.replies[0].depth === 2, r1a.replies[0].depth);

    check("pagination metadata (by_post) attached to the top-level comment", isPlainObj(c1.pagination) && c1.pagination.feedback_id === "fb:P1", c1.pagination);
    check("the same pagination entry is attached to nested replies sharing the post", isPlainObj(r1a.pagination) && r1a.pagination.feedback_id === "fb:P1", r1a.pagination);

    check("no field anywhere is literally named `author` (bridge would redact it)", sensitiveKeys(out, "").length === 0, sensitiveKeys(out, ""));
  });
}

function isPlainObj(v) { return !!v && typeof v === "object" && !Array.isArray(v); }

// ============================================================================================
// Hand-built fixtures for capabilities with no existing test-suite fixture. Each shape is
// copied from a direct read of chrome-extension/gql_extract.js at the cited lines — not invented.
// ============================================================================================

(function testProfilePosts() {
  console.log("\n-- fb.profile.posts (hand-built, gql_extract.js:894-912) --");
  const raw = {
    capability: "fb.profile.posts", schema: "PostRecord[]", source_query: "ProfileCometTimelineFeedQuery", count: 2,
    available: true, found: true,
    items: [
      {
        id: "9001", post_id: "9001", url: "https://www.facebook.com/jane.realtor/posts/9001/",
        actor: { type: "profile", id: "700111222333", name: "Jane Realtor", url: "https://www.facebook.com/jane.realtor" },
        text: "Just closed on Main St!", created_time: 1757000000,
        attachments: [{ type: "Photo", url: "https://scontent.xx.fbcdn.net/photo1.jpg" }],
        engagement: { reactions: 12, comments: 3, shares: 0 },
        feedback_id: "ZmVlZGJhY2s6OTAwMQ==", comment_intent: "RANKED_UNFILTERED_COMMENTS_INTENT_V2_TOKEN",
        group: null, // fb.profile.posts always sets group:null (gql_extract.js:911)
      },
      {
        id: "9002", post_id: "9002", url: "", actor: null, text: "no timestamp on this one",
        created_time: 0, attachments: [], engagement: null, feedback_id: "", comment_intent: "", group: null,
      },
    ],
  };
  const { ctx, normalize } = makeGqlExtractCtx([]);
  const out = collect(normalize("fb.profile.posts", raw, { captured_at: CAPTURED_AT }));
  check("2 canonical posts", out.items.length === 2, out.items.length);
  check("group_ref is null (profile timeline posts are never group-owned)", out.items[0].group_ref === null, out.items[0].group_ref);
  check("engagement null-vs-0: found node keeps 0 sub-counts, no node -> null", out.items[0].engagement.shares === 0 && out.items[1].engagement === null, [out.items[0].engagement, out.items[1].engagement]);
  check("created_time:0 sentinel -> created_at null (not epoch)", out.items[1].created_at === null, out.items[1].created_at);
})();

(function testNewsfeed() {
  console.log("\n-- fb.newsfeed (hand-built, gql_extract.js:971-988) --");
  const raw = {
    capability: "fb.newsfeed", schema: "PostRecord[]", available: true, found: true, count: 1,
    items: [{
      id: "8001", post_id: "8001", url: "https://www.facebook.com/groups/1/posts/8001/",
      actor: { type: "profile", id: "700333444555", name: "John Buyer", url: "https://www.facebook.com/john.buyer" },
      text: "Reshared: check this listing", created_time: 1757000500,
      attachments: [], engagement: { reactions: 4, comments: 0, shares: 1 },
      feedback_id: "fb:8001", comment_intent: "",
      // fb.newsfeed DOES populate group (a reshared/quoted post's original poster) — unlike fb.profile.posts.
      group: { type: "group", id: "1234567890", name: "LA Realtors Network", url: "https://www.facebook.com/groups/1234567890/" },
    }],
  };
  const { normalize } = makeGqlExtractCtx([]);
  const out = collect(normalize("fb.newsfeed", raw, { captured_at: CAPTURED_AT }));
  const it = out.items[0];
  check("group_ref populated for fb.newsfeed", it.group_ref && it.group_ref.platform_id === "1234567890" && it.group_ref.name === "LA Realtors Network", it.group_ref);
})();

(function testProfileVideos() {
  console.log("\n-- fb.profile.videos (hand-built, gql_extract.js:1071-1080) --");
  const raw = {
    capability: "fb.profile.videos", schema: "VideoRecord[]", available: true, count: 2,
    items: [
      {
        id: "998877665544332", title: "Open House Walkthrough", views: 77000, reactions: 210, shares: 52,
        caption: "Come see this beautiful 4bd home this Saturday! #openhouse",
        url: "https://www.facebook.com/watch/?v=998877665544332",
        actor: { type: "profile", id: "700111222333", name: "Jane Realtor", url: "https://www.facebook.com/jane.realtor" },
      },
      {
        id: "998877665544333", title: "Market Update", views: null, reactions: null, shares: null,
        caption: "", url: "https://www.facebook.com/watch/?v=998877665544333", actor: null,
      },
    ],
  };
  const { normalize } = makeGqlExtractCtx([]);
  const out = collect(normalize("fb.profile.videos", raw, { captured_at: CAPTURED_AT }));
  check("kind post, media.type video", out.kind === "post" && out.items[0].media.type === "video", out.items[0].media);
  check("media.url mirrors the post url", out.items[0].media.url === raw.items[0].url, out.items[0].media.url);
  check("engagement.likes from reactions, .views from views", out.items[0].engagement.likes === 210 && out.items[0].engagement.views === 77000, out.items[0].engagement);
  check("all-null views/reactions/shares -> engagement null", out.items[1].engagement === null, out.items[1].engagement);
  check("no duration/hashtags field on VideoRecord -> duration null, hashtags []", out.items[0].media.duration === null && Array.isArray(out.items[0].media.hashtags) && out.items[0].media.hashtags.length === 0, out.items[0].media);
})();

(function testReelsFeed() {
  console.log("\n-- fb.reels.feed (hand-built, gql_extract.js:1513-1583, player + grid mode) --");
  const player = {
    capability: "fb.reels.feed", schema: "ReelRecord[]", available: true, count: 1, mode: "player",
    items: [{
      reel_id: "1234567890123456", reel_url: "https://www.facebook.com/reel/1234567890123456",
      creator: { name: "Jane Realtor", url: "https://www.facebook.com/jane.realtor" },
      caption: "Day in the life of a realtor #realestate #realtor", hashtags: ["#realestate", "#realtor"],
    }],
  };
  const grid = {
    capability: "fb.reels.feed", schema: "ReelRecord[]", available: true, count: 1, mode: "grid",
    items: [{
      reel_id: "1234567890123457", reel_url: "https://www.facebook.com/reel/1234567890123457",
      views: 45000, view_text: "45K", caption: "", creator: null, hashtags: [],
    }],
  };
  const { normalize } = makeGqlExtractCtx([]);

  const outPlayer = collect(normalize("fb.reels.feed", player, { captured_at: CAPTURED_AT }));
  const p = outPlayer.items[0];
  check("player mode: media.type reel, hashtags carried through", p.media.type === "reel" && p.media.hashtags.join(",") === "#realestate,#realtor", p.media);
  check("player mode: actor from creator{name,url}, no id available -> platform_id ''", p.actor.platform_id === "" && p.actor.name === "Jane Realtor", p.actor);
  check("player mode: no views field at all -> engagement null, media.views null", p.engagement === null && p.media.views === null, [p.engagement, p.media.views]);

  const outGrid = collect(normalize("fb.reels.feed", grid, { captured_at: CAPTURED_AT }));
  const g = outGrid.items[0];
  check("grid mode: creator null -> actor null", g.actor === null, g.actor);
  check("grid mode: views present -> engagement.views + media.views populated", g.engagement && g.engagement.views === 45000 && g.media.views === 45000, [g.engagement, g.media]);
  check("grid mode: view_text (undocumented in the JSON schema, present in code) carried into ext", g.ext.view_text === "45K", g.ext);
})();

(function testProfileFriendsAndPeopleSearch() {
  console.log("\n-- fb.profile.friends / fb.people.search (hand-built, gql_extract.js:638-645 / :850-857) --");
  const friendsRaw = {
    capability: "fb.profile.friends", schema: "ProfileSummary[]", available: true, count: 1,
    items: [{ id: "700333444555", name: "John Buyer", url: "https://www.facebook.com/john.buyer", subtitle: "12 mutual friends", mutual_friends: 12, industry_hint: null }],
  };
  const peopleRaw = {
    capability: "fb.people.search", schema: "ProfileSummary[]", available: true, count: 1,
    items: [{ id: "700333444556", name: "Sam Prospect", url: "https://www.facebook.com/sam.prospect", subtitle: "Realtor at Keller Williams · 8 mutual friends", mutual_friends: 8, industry_hint: "real estate" }],
  };
  const { normalize } = makeGqlExtractCtx([]);

  const fOut = collect(normalize("fb.profile.friends", friendsRaw, { captured_at: CAPTURED_AT }));
  check("friends: kind profile, subtitle -> bio", fOut.kind === "profile" && fOut.items[0].bio === "12 mutual friends", fOut.items[0]);
  check("friends: industry_hint:null never set as canonical industry", !("industry" in fOut.items[0]), fOut.items[0]);
  check("friends: mutual_friends carried into ext", fOut.items[0].ext.mutual_friends === 12, fOut.items[0].ext);

  const pOut = collect(normalize("fb.people.search", peopleRaw, { captured_at: CAPTURED_AT }));
  check("people.search: industry_hint unified to canonical `industry`", pOut.items[0].industry === "real estate", pOut.items[0]);
})();

(function testHovercard() {
  console.log("\n-- fb.profile.hovercard (hand-built, gql_extract.js:3251-3277) --");
  const raw = {
    capability: "fb.profile.hovercard", schema: "HovercardRecord", available: true, count: 1,
    items: [{
      id: "700333444555", name: "John Buyer", url: "https://www.facebook.com/john.buyer",
      context: ["Works at Acme Realty", "Lives in Los Angeles, California"],
      work: ["Works at Acme Realty"], education: [], location: ["Lives in Los Angeles, California"],
      gender: "male", is_verified: false, memorialized: false,
      profile_picture: "https://scontent.xx.fbcdn.net/v/t1/s200x200/x.jpg",
      bio: "", category: "", subscribe_status: "", friendship_status: "not_friends",
      mutual_friends: 3, actions: ["Add friend", "Message"],
    }],
  };
  const { normalize } = makeGqlExtractCtx([]);
  const out = collect(normalize("fb.profile.hovercard", raw, { captured_at: CAPTURED_AT }));
  const it = out.items[0];
  check("hovercard `url` field maps straight to canonical url", it.url === "https://www.facebook.com/john.buyer", it.url);
  check("hovercard `profile_picture` unifies into canonical photo_url", it.photo_url === raw.items[0].profile_picture, it.photo_url);
  check("verified from is_verified", it.verified === false, it.verified);
  check("gender/friendship_status/mutual_friends (no canonical field) land in ext", it.ext.gender === "male" && it.ext.friendship_status === "not_friends" && it.ext.mutual_friends === 3, it.ext);
})();

(function testHeader() {
  console.log("\n-- fb.profile.header (hand-built, gql_extract.js:1909-1930) --");
  const raw = {
    capability: "fb.profile.header", schema: "ProfileHeader", available: true, count: 1,
    items: [{
      name: "Jane Realtor", url: "https://www.facebook.com/jane.realtor",
      follower_count: 10000, follower_text: "10K", like_count: 9800, verified: true,
      category: "Real Estate Agent", intro_bio: "Helping LA families find home since 2015.",
      work: ["Acme Realty"], education: [], location: ["Los Angeles, California"],
      intro_lines: ["Real Estate Agent", "Works at Acme Realty"],
      website: "https://janerealtor.com", cta: ["Send Message", "Call Now"],
      has_reels_tab: true, has_videos_tab: true,
    }],
  };
  const { normalize } = makeGqlExtractCtx([]);
  const out = collect(normalize("fb.profile.header", raw, { captured_at: CAPTURED_AT }));
  const it = out.items[0];
  check("header `url` field maps straight to canonical url; no id at all -> platform_id ''", it.url === "https://www.facebook.com/jane.realtor" && it.platform_id === "", it);
  check("no photo field on ProfileHeader -> photo_url absent/unset", it.photo_url === undefined, it.photo_url);
  check("singular website -> canonical websites[] array", Array.isArray(it.websites) && it.websites[0] === "https://janerealtor.com", it.websites);
  check("follower_count/verified mapped", it.follower_count === 10000 && it.verified === true, it);

  // Self-profile refusal (gql_extract.js:1761): items:[] at the wrapper level -> nothing to normalize.
  const refusal = { capability: "fb.profile.header", available: false, reason: "self_profile", count: 0, items: [], error: "resolved to the logged-in operator" };
  const refusedOut = collect(normalize("fb.profile.header", refusal, { captured_at: CAPTURED_AT }));
  check("self-profile refusal normalizes to zero items, not an error", refusedOut.items.length === 0, refusedOut);
})();

(function testDomFailSkipped() {
  console.log("\n-- DOM_CAPABILITIES generic crash shape is skipped, not normalized (gql_extract.js:3800-3802) --");
  const crash = { capability: "fb.profile.hovercard", available: true, count: 0, items: [{ capability: "fb.profile.hovercard", status: "error", error: "timed out" }], error: "timed out" };
  const { normalize } = makeGqlExtractCtx([]);
  const out = collect(normalize("fb.profile.hovercard", crash, { captured_at: CAPTURED_AT }));
  check("status-only crash row produced zero canonical items", out.items.length === 0, out);
})();

(function testProfileContacts() {
  console.log("\n-- fb.profile.contacts (hand-built, gql_extract.js:2119-2124) --");
  const raw = {
    capability: "fb.profile.contacts", schema: "ContactRecord", available: true, found: true, count: 1,
    items: [{ profile_url: "https://www.facebook.com/jane.realtor", emails: ["jane@example.com"], websites: ["https://janerealtor.com"], found_on: "directory_contact_info", checked: ["current_page", "about", "directory_contact_info"] }],
  };
  const { normalize } = makeGqlExtractCtx([]);
  const out = collect(normalize("fb.profile.contacts", raw, { captured_at: CAPTURED_AT }));
  const it = out.items[0];
  check("contacts `profile_url` maps to canonical url; no name field -> name ''", it.url === raw.items[0].profile_url && it.name === "", it);
  check("emails/websites carried through", it.emails[0] === "jane@example.com" && it.websites[0] === "https://janerealtor.com", it);
  check("extraction_audit built from found_on/checked with defaults for the rest", it.extraction_audit.found_on === "directory_contact_info" && Array.isArray(it.extraction_audit.skipped_tabs) && it.extraction_audit.skipped_tabs.length === 0, it.extraction_audit);
})();

// ============================================================================================
// Harness 3 — tests/test_profile_dossier.js style: a fake navigable DOM (clicking a tab anchor
// swaps the page text). Copied in full since fb.profile.dossier/enrich cannot be exercised any
// other way; this is the heaviest but most faithful fixture in the suite.
// ============================================================================================
const CHROME = ["Facebook", "Search", "Home", "Marketplace", "Notifications", "Menu"];
function page(lines) { return CHROME.concat(lines).join("\n"); }
const PAGES = {
  main: page(["Dana Hanh Lam", "1.2K followers", "Works at ZenWealth Solutions", "Lives in Houston, Texas"]),
  about: page(["About"]),
  contact_info: page(["Email", "advisor@northstar-wealth.invalid", "Website", "northstar-wealth.invalid"]),
  work: page(["Work", "Loan Officer at Wells Fargo", "Mortgage Advisor at ZenWealth Solutions"]),
  education: page(["College", "Studied at University of Houston"]),
  intro: page(["Helping families finance their first home with clarity"]),
  category: page(["Digital creator"]),
};
const TAB_SLUG = {
  contact_info: "directory_contact_info",
  work: "directory_work",
  education: "directory_education",
  intro: "directory_intro",
  category: "directory_category",
};
function makeDossierCtx(opts) {
  opts = opts || {};
  const SLUGS = opts.slugs || TAB_SLUG;
  const offers = opts.offers || Object.keys(SLUGS);
  const pages = opts.pages || PAGES;
  const state = { current: "main", clicks: [] };
  const captures = [];
  let feedScans = 0;
  let seeMoreLeft = opts.seeMore === undefined ? 1 : opts.seeMore;

  function anchor(href, text, onClick) {
    return { innerText: text, getAttribute: (a) => (a === "href" ? href : null), getBoundingClientRect: () => ({ width: 100, height: 20 }), click: onClick };
  }
  function seeMoreBtn() {
    return {
      innerText: "See more", getAttribute: () => null, getBoundingClientRect: () => ({ width: 60, height: 16 }),
      parentElement: opts.inFeed ? { getAttribute: (a) => (a === "role" ? "article" : null), parentElement: null } : null,
      click: () => { seeMoreLeft -= 1; captures.push({ queryName: "CometTextWithEntitiesSeeMoreQuery", docId: "doc_seemore", variables: { id: "x" }, fbDtsg: "TOKEN" }); },
    };
  }
  function navTo(key) {
    return () => { state.current = key; state.clicks.push(key); captures.push({ queryName: "ProfileCometAbout" + key + "Query", docId: "doc_" + key, variables: { id: "1369773994", scale: 1 }, fbDtsg: "TOKEN" }); };
  }

  const document = {
    title: "Dana Hanh Lam | Facebook",
    get body() { return { innerText: pages[state.current] || "", innerHTML: "<div>" + (pages[state.current] || "") + "</div>" }; },
    querySelector: (sel) => {
      if (/role="main"/.test(sel)) {
        return {
          get innerText() { return (pages[state.current] || "") + (opts.feed ? "\n" + opts.feed : ""); },
          querySelectorAll: (q) => {
            if (/article|feed/.test(q)) { feedScans += 1; return opts.feed ? [{ get innerText() { return opts.feed; } }] : []; }
            return [];
          },
        };
      }
      if (/href\*="directory_"/.test(sel)) return null;
      return null;
    },
    querySelectorAll: (sel) => {
      if (sel === 'a[href*="directory_"]') {
        const card = {
          get innerText() { return (pages[state.current] || ""); },
          querySelectorAll: (q) => (/role="button"/.test(q) && seeMoreLeft > 0 ? [seeMoreBtn()] : []),
          parentElement: { get innerText() { return (pages[state.current] || "") + "\n" + (opts.feed || ""); }, querySelectorAll: () => [], parentElement: null },
        };
        const nav = { innerText: Object.keys(SLUGS).join("\n"), querySelectorAll: (q) => (/directory_/.test(q) ? navLinks() : []), parentElement: card };
        function navLinks() {
          return offers.map((k) => ({ getAttribute: () => "/claire/" + SLUGS[k], parentElement: nav, getBoundingClientRect: () => ({ width: 80, height: 18 }) }));
        }
        card.querySelectorAll = (q) => (/directory_/.test(q) ? navLinks() : (/role="button"/.test(q) && seeMoreLeft > 0 ? [seeMoreBtn()] : []));
        return navLinks();
      }
      if (/sk=about|\/about/.test(sel) && state.current === "main") return [anchor("/claire/about", "About", navTo("about"))];
      const m = sel.match(/href\*="([^"]+)"/);
      if (m) {
        const slug = m[1];
        const key = Object.keys(SLUGS).find((k) => SLUGS[k] === slug);
        if (key && offers.indexOf(key) !== -1) return [anchor("/claire/" + slug, key, navTo(key))];
        return [];
      }
      if (/^a\[href\^="http"\]$/.test(sel)) return [];
      if (sel === "a[href]") {
        if (state.current === "main") return [anchor("/claire/about", "About", navTo("about"))];
        return offers.map((k) => anchor("/claire/" + SLUGS[k], k, navTo(k)));
      }
      if (/h1/.test(sel)) {
        const first = (pages[state.current] || "").split("\n").find((l) => /Dana/.test(l));
        return first ? [{ innerText: first }] : [];
      }
      if (/role="button"/.test(sel) && seeMoreLeft > 0) return [seeMoreBtn()];
      return [];
    },
  };

  const ctx = {
    window: {},
    location: { href: "https://www.facebook.com/claire", origin: "https://www.facebook.com", pathname: "/claire", search: "" },
    document, setTimeout, clearTimeout, URL, console, Date, URLSearchParams,
    MutationObserver: function () { this.observe = () => {}; this.disconnect = () => {}; },
  };
  ctx.globalThis = ctx;
  vm.createContext(ctx);
  vm.runInContext(GQL_EXTRACT_SRC, ctx);
  ctx.window.__soloGql = { captures, origFetch: null };
  const normalize = loadNormalizer(ctx);
  return { ctx, state, normalize };
}
const FAST = { settle_ms: 1 };

function runDossierAndEnrichTests() {
  console.log("\n-- fb.profile.dossier (real extractor, heavy DOM harness) --");
  const { ctx, normalize } = makeDossierCtx();
  return ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST).then((res) => {
    check("dossier ran and found a record", res.found === true && res.items.length === 1, res.found);
    const before = JSON.stringify(res);
    const out = collect(normalize("fb.profile.dossier", res, { captured_at: CAPTURED_AT }));
    check("input not mutated", JSON.stringify(res) === before);
    const it = out.items[0];
    check("dossier `profile_url` maps to canonical url (no `url` field on this record)", it.url === "https://www.facebook.com/claire", it.url);
    check("name parsed off the main page", it.name === "Dana Hanh Lam", it.name);
    // This fixture's PAGES.main has no category-shaped text for profileHeader's own regex scan
    // to catch, so top-level `category` stays "" here (it is filled by header/later text
    // elsewhere, e.g. testHeader()'s hand-built fixture) — but the About "category" tab's own
    // text IS captured, inside the sectioned about{} map. Either way, no industry/industry_hint
    // sibling ever appears on a dossier/enrich item — that unification only applies to
    // fb.profile.friends/fb.people.search's ProfileSummary.
    check("category has no industry/industry_hint sibling; About-tab text lands in about.category", typeof it.category === "string" && !("industry_hint" in it) && !("industry" in it) && it.about.category.indexOf("Digital creator") !== -1, [it.category, it.about.category]);
    check("emails discovered on the contact_info tab", it.emails.indexOf("advisor@northstar-wealth.invalid") !== -1, it.emails);
    check("work discovered on the work tab", it.work.some((w) => /Loan Officer at Wells Fargo/.test(w)), it.work);
    check("about is the sectioned open-map object, carried through as-is", isPlainObj(it.about), it.about);
    check("extraction_audit.checked/discovered_tabs populated", Array.isArray(it.extraction_audit.checked) && it.extraction_audit.checked.length > 0, it.extraction_audit);
    check("platform_id stays '' (dossier carries no id field)", it.platform_id === "", it.platform_id);

    console.log("\n-- fb.profile.enrich (real extractor: dossier + stubbed fb.profile.posts) --");
    const { ctx: ctx2, normalize: normalize2 } = makeDossierCtx({ seeMore: 0 });
    ctx2.location.href = "https://www.facebook.com/jebsmith";
    ctx2.window.__soloGqlExtractCapability = (id) => id === "fb.profile.posts"
      ? { available: true, items: [{ id: "p1", text: "Just closed on Main St" }, { id: "p2", text: "Market update" }, { id: "p3", text: "Open house" }, { id: "p4", text: "old" }, { id: "p5", text: "older" }, { id: "p6", text: "oldest" }] }
      : { available: false, reason: "no_capture_in_scope", items: [] };
    return ctx2.window.__soloGqlPaginate("fb.profile.enrich", { settle_ms: 1, max_posts: 3 }).then((res2) => {
      check("enrich returned exactly one merged record", res2.capability === "fb.profile.enrich" && res2.items.length === 1, res2.capability);
      const out2 = collect(normalize2("fb.profile.enrich", res2, { captured_at: CAPTURED_AT }));
      const it2 = out2.items[0];
      check("enrich still uses profile_url -> canonical url", it2.url === "https://www.facebook.com/jebsmith", it2.url);
      check("max_posts honoured, posts carried into ext (not re-normalized as a nested post kind)", Array.isArray(it2.ext.posts) && it2.ext.posts.length === 3, it2.ext.posts);
      check("timeline carried into ext as a raw platform-only fact", isPlainObj(it2.ext.timeline), it2.ext.timeline);
      return runActionsTests();
    });
  });
}

// ============================================================================================
// Harness 4 — tests/test_gql_actions.js style: fakeEl + querySelectorAll stub.
// ============================================================================================
function fakeEl(attrs, opts) {
  opts = opts || {};
  return {
    _attrs: attrs || {}, innerText: opts.innerText || "",
    getAttribute(k) { return this._attrs[k] === undefined ? null : this._attrs[k]; },
    getBoundingClientRect() { return { width: opts.hidden ? 0 : 100, height: opts.hidden ? 0 : 20 }; },
    focus() {}, scrollIntoView() {}, dispatchEvent() { return true; },
    closest() { return this; }, matches() { return false; }, querySelector() { return null; },
  };
}
function makeActionsCtx(opts) {
  opts = opts || {};
  const win = {};
  const href = opts.href || "https://www.facebook.com/groups/000000000000000";
  const ctx = {
    window: win,
    location: { href, origin: new URL(href).origin },
    document: {
      title: opts.title || "", body: { innerText: opts.bodyText || "" },
      querySelectorAll: opts.querySelectorAll || (() => []), querySelector: () => null, execCommand: () => true,
    },
    setTimeout, clearTimeout, URL, console,
    MouseEvent: function () {}, KeyboardEvent: function () {}, InputEvent: function () {},
  };
  ctx.window = win;
  ctx.globalThis = ctx;
  vm.createContext(ctx);
  vm.runInContext(GQL_ACTIONS_SRC, ctx);
  const normalize = loadNormalizer(ctx);
  return { ctx, normalize };
}

function runActionsTests() {
  console.log("\n-- fb.group.post (real action, dry_run, gql_actions.js:661-758) --");
  const box = fakeEl({ "aria-label": "Create a public post" });
  const btn = fakeEl({ "aria-label": "Post" }, { innerText: "Post" });
  const dlg = fakeEl({ role: "dialog" });
  dlg.querySelector = (sel) => (String(sel).indexOf("contenteditable") > -1 ? box : null);
  dlg.querySelectorAll = (sel) => (String(sel).indexOf("button") > -1 ? [btn] : []);
  const { ctx: gCtx, normalize: gNorm } = makeActionsCtx({
    querySelectorAll: (sel) => (String(sel).indexOf('[role="dialog"]') > -1 ? [dlg] : []),
  });
  return gCtx.window.__soloActRun("fb.group.post", { text: "hello group", dry_run: true, _target_url: "https://www.facebook.com/groups/000000000000000" }).then((r) => {
    check("dry_run readiness reported", r.status === "dry_run" && r.items[0].post_button_found === true, r.items[0]);
    const before = JSON.stringify(r);
    const out = collect(gNorm("fb.group.post", r, { captured_at: CAPTURED_AT }));
    check("input not mutated", JSON.stringify(r) === before);
    check("kind message, channel post, direction out", out.kind === "message" && out.items[0].channel === "post" && out.items[0].direction === "out", out.items[0]);
    check("body_text carries the posted text", out.items[0].body_text === "hello group", out.items[0].body_text);
    check("status passthrough", out.items[0].status === "dry_run", out.items[0].status);
    check("group_ref.platform_id from the dry_run's reported group", out.items[0].group_ref.platform_id === "000000000000000", out.items[0].group_ref);
    check("post_button_found (extra field) carried into ext", out.items[0].ext.post_button_found === true, out.items[0].ext);

    console.log("\n-- fb.message.send (real action, dry_run, gql_actions.js:833-924) --");
    const composer = (name) => fakeEl({ "aria-label": "Write to " + name });
    // idVerified (gql_actions.js:851-854) compares the requested thread key against the key
    // parsed out of the CURRENT location.href — so the fixture's href must already be the
    // landed thread url, not merely _target_url, for id_verified to come back true.
    const THREAD_URL = "https://www.facebook.com/messages/t/700333444555";
    const { ctx: mCtx, normalize: mNorm } = makeActionsCtx({
      href: THREAD_URL,
      querySelectorAll: (sel) => (String(sel).indexOf("contenteditable") > -1 ? [composer("Bob Nguyen")] : []),
    });
    return mCtx.window.__soloActRun("fb.message.send", { text: "hi", recipient_name: "Bob Nguyen", dry_run: true, _target_url: THREAD_URL }).then((r2) => {
      check("dry_run status, id_verified true (numeric profile id pre-gate)", r2.status === "dry_run" && r2.items[0].id_verified === true, r2.items[0]);
      const out2 = collect(mNorm("fb.message.send", r2, { captured_at: CAPTURED_AT }));
      check("kind message, channel dm", out2.kind === "message" && out2.items[0].channel === "dm", out2.items[0]);
      check("body_text carries the sent text", out2.items[0].body_text === "hi", out2.items[0].body_text);
      check("thread_ref.platform_id from requested_id/open_id", typeof out2.items[0].thread_ref.platform_id === "string" && out2.items[0].thread_ref.platform_id.length > 0, out2.items[0].thread_ref);

      console.log("\n-- fb.post.comment (real action resolve + write, gql_actions.js:503-599) --");
      function post(text, url, id) { return { id: id || url, post_id: id || "", url, text, created_time: 1 }; }
      const G = "https://www.facebook.com/groups/000000000000000/posts/";
      const { ctx: cCtx } = makeActionsCtx();
      cCtx.window.__soloGqlPaginate = async () => ({
        available: true,
        items: [post("Post 5 post 5 post 5", G + "000000000000001/"), post("Post 3: something else", G + "000000000000002/")],
        reason: "",
      });
      return cCtx.window.__soloActResolve("fb.post.comment", { match_text: "Post 5 post 5 post 5", _target_url: "https://www.facebook.com/groups/000000000000000" }).then((envelope) => {
        check("resolution matched the one post", envelope.status === "resolved", envelope.status);

        console.log("\n-- fb.post.react (hand-built from verified source, gql_actions.js:359-404 + :118-127 wrapCap — no existing fixture in test_gql_actions.js) --");
        const reactRaw = {
          available: true, capability: "fb.post.react", status: "done", count: 1,
          items: [{
            capability: "fb.post.react", status: "done", verified: true, error: null, ts: "2026-09-09T12:00:00.000Z",
            reaction: "love",
            target_preview: { actor: "Jane Realtor", snippet: "Just listed! 3bd/2ba in Venice", url: "https://www.facebook.com/groups/1/posts/2/" },
          }],
          _debug: { href: "https://www.facebook.com/groups/1/posts/2/" },
        };
        const { normalize: rNorm } = makeActionsCtx();
        const outR = collect(rNorm("fb.post.react", reactRaw, { captured_at: CAPTURED_AT }));
        check("kind message, channel reaction, direction out", outR.kind === "message" && outR.items[0].channel === "reaction" && outR.items[0].direction === "out", outR.items[0]);
        check("body_text is '' for a reaction (nothing was typed)", outR.items[0].body_text === "", outR.items[0].body_text);
        check("post_ref (opaque string handle) taken from target_preview.url", outR.items[0].post_ref === reactRaw.items[0].target_preview.url, outR.items[0].post_ref);
        check("target_preview.actor is NOT named author anywhere downstream (already correct at the source)", outR.items[0].ext.target_preview.actor === "Jane Realtor", outR.items[0].ext.target_preview);
        check("reaction/verified carried into ext / top-level", outR.items[0].ext.reaction === "love" && outR.items[0].verified === true, outR.items[0]);

        console.log("\n-- web.search -> null (no canonical kind in v1) --");
        const { normalize: wNorm } = makeActionsCtx();
        check("web.search normalizes to null", wNorm("web.search", { capability: "web.search", items: [{ title: "x", url: "https://x" }] }, {}) === null);

        console.log("\n-- unknown capability -> null --");
        check("an unrecognised capability id normalizes to null", wNorm("fb.totally.unknown", { items: [{ id: "1" }] }, {}) === null);

        console.log("\n-- default captured_at (opts omitted) --");
        const before2 = Date.now();
        const defaulted = wNorm("fb.groups.search", { items: [{ type: "group", id: "1", name: "N", url: "https://x" }] }, {});
        const parsed = Date.parse(defaulted.items[0].captured_at);
        check("captured_at defaults to now() when opts.captured_at is omitted", isFinite(parsed) && Math.abs(parsed - before2) < 5000, defaulted.items[0].captured_at);

        return finishFbSuite();
      });
    });
  });
}

function finishFbSuite() {
  console.log("\n-- cross-cutting: no sensitive key anywhere across every normalized item --");
  const hits = sensitiveKeys(ALL_ITEMS, "");
  check("zero sensitive-key hits across " + ALL_ITEMS.length + " normalized items", hits.length === 0, hits);

  console.log("\n-- cross-cutting: chrome-extension/core/schema.js validateRecord().ok --");
  if (fs.existsSync(SCHEMA_PATH)) {
    const SCHEMA_SRC = fs.readFileSync(SCHEMA_PATH, "utf8");
    const schemaCtx = { console };
    vm.createContext(schemaCtx);
    vm.runInContext(SCHEMA_SRC, schemaCtx);
    const SoloSchema = schemaCtx.SoloSchema;
    check("schema.js loaded and exposes SoloSchema.validateRecord", typeof SoloSchema.validateRecord === "function");
    let allOk = true;
    ALL_ITEMS.forEach((item, i) => {
      const r = SoloSchema.validateRecord(item);
      if (!r.ok) { allOk = false; console.log("    item[" + i + "] (" + item.kind + "/" + item.source_capability + ") errors: " + JSON.stringify(r.errors)); }
    });
    check("validateRecord().ok for all " + ALL_ITEMS.length + " normalized items", allOk);
  } else {
    console.log("  SKIP  chrome-extension/core/schema.js not found — schema validation skipped");
  }

  console.log("\n" + (fail === 0 ? "ALL " + pass + " CHECKS PASSED" : pass + " passed, " + fail + " FAILED"));
  process.exit(fail === 0 ? 0 : 1);
}

testPostCommentsRecursiveAndPagination()
  .then(() => runDossierAndEnrichTests())
  .catch((e) => { console.error(e); process.exit(1); });
