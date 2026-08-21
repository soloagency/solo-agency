// Offline harness for fb.post.comments and the with_comments composition.
//
// Comments do not travel with a post: a story node carries a total_count and nothing else, so
// every comment in the system arrives through a SEPARATE replayed query keyed by feedback id.
// That makes three failures possible which all LOOK the same from the outside — an empty list:
//   1. the feedback id never made it onto the PostRecord, so nothing was ever asked for;
//   2. the query was rejected (missing provider variable, stale doc_id) and answered 200 with
//      an `errors` array that nobody read;
//   3. the post genuinely has no comments.
// The codebase has been burned by exactly this class before, so these tests assert on WHICH
// requests went out and on the REPORTED reason, not merely on the row count.
const fs = require("fs");
const path = require("path");
const vm = require("vm");

const SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "gql_extract.js"), "utf8");

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail) : "")); }
}

const COMMENT_Q = "CommentsListComponentsPaginationQuery";
const REPLY_Q = "Depth1CommentsListPaginationQuery";
const SEARCH_Q = "SearchCometResultsPaginatedResultsQuery";

// ---------------------------------------------------------------------------
// Fixtures: the shapes Facebook actually answers with.
// ---------------------------------------------------------------------------

function author(n) {
  return { id: "u" + n, name: "Person " + n, url: "https://www.facebook.com/p" + n };
}
// A comment carries THREE distinct handles: its own id, its feedback id (what the reply query
// is addressed to), and the post's feedback id. Conflating any two of them is silent: the reply
// walk returns empty for every comment while reporting success.
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
// Top-level comments on fbP1: 5 across 3 pages. c1 has 2 replies; r1a has 1 reply of its own.
const TOP_PAGES = [
  { edges: [{ node: commentNode("c1", 1, { replies: 2 }) }, { node: commentNode("c2", 2) }], next: "cc:1" },
  { edges: [{ node: commentNode("c3", 3) }, { node: commentNode("c4", 4) }], next: "cc:2" },
  { edges: [{ node: commentNode("c5", 5) }], next: null },
];
const REPLIES = {
  "cfb:c1": [{ node: commentNode("r1a", 11, { depth: 1, replies: 1 }) }, { node: commentNode("r1b", 12, { depth: 1 }) }],
  "cfb:r1a": [{ node: commentNode("r2a", 21, { depth: 2 }) }],
};
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

// A search story node carrying its UFI — the node fb.group.search_posts extracts from.
function storyNode(id, feedbackId, nesting) {
  // The UFI node carries the engagement counts and the selectable intents. Its OWN id is
  // deliberately set to something else: the feedback id the comment query wants lives at
  // comet_sections.content.story.feedback.id, and a test that let the two be the same would
  // pass whichever one the code happened to read.
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
function searchResponse(stories) {
  return { data: { serpResponse: { results: {
    edges: stories.map((s) => ({ rendering_strategy: { view_model: { click_model: { story: s } } } })),
    page_info: { end_cursor: null, has_next_page: false },
  } } } };
}

// ---------------------------------------------------------------------------
// Harness
// ---------------------------------------------------------------------------

function makeCtx(opts) {
  opts = opts || {};
  const sent = [];                       // every outgoing request: { query, vars }
  const registry = opts.registry || { [COMMENT_Q]: "doc_comments", [REPLY_Q]: "doc_replies", [SEARCH_Q]: "doc_search" };
  const providers = opts.providers || {};

  const ctx = {
    window: {
      require: (name) => {
        if (name === "WebPixelRatio") return { get: () => 2 };
        const m = /^(.*)\.graphql$/.exec(name);
        if (!m) throw new Error("module not found: " + name);
        const id = registry[m[1]];
        if (!id) throw new Error("module not found: " + name);      // exactly how Facebook fails
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
  vm.runInContext(SRC, ctx);

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
      else if (query === REPLY_Q) body = replyReply(vars.id);   // keyed by the comment's FEEDBACK id
      else body = opts.seedResponse || searchResponse([]);
      return Promise.resolve({ text: () => Promise.resolve(JSON.stringify(body)) });
    },
  };
  return { ctx, sent, seed };
}
const call = (h, inputs) => h.ctx.window.__soloGqlPaginate("fb.post.comments", inputs);

async function run() {
  console.log("== the feedback id reaches the PostRecord, or nothing can ever be asked for ==");
  {
    const h = makeCtx({});
    const rec = h.ctx.window.__soloGqlExtractCapability;
    // Both nestings Facebook serves, plus a bare feedback:{id} with no *_with_context wrapper.
    for (const [label, nesting] of [["story_ufi_container", "plain"], ["comet_feed_ufi_container", "ufi_container"]]) {
      const h2 = makeCtx({ seedResponse: searchResponse([storyNode("P1", "fb:P1", nesting)]) });
      h2.ctx.window.__soloGql.captures[0].response = searchResponse([storyNode("P1", "fb:P1", nesting)]);
      const res = h2.ctx.window.__soloGqlExtractCapability("fb.group.search_posts", {});
      check("feedback_id survives the " + label + " nesting", (res.items[0] || {}).feedback_id === "fb:P1", (res.items[0] || {}).feedback_id);
      // The UFI node has an id of its own and it is NOT interchangeable with the feedback id.
      check("it is not the UFI node's id (" + label + ")", (res.items[0] || {}).feedback_id !== "ufi:P1", (res.items[0] || {}).feedback_id);
      check("the story's published intent is carried on the record (" + label + ")",
        (res.items[0] || {}).comment_intent === "RANKED_UNFILTERED_INTENT_V1", (res.items[0] || {}).comment_intent);
      check("engagement still reads through the same node (" + label + ")", ((res.items[0] || {}).engagement || {}).comments === 5, (res.items[0] || {}).engagement);
    }
    void rec;
  }
  {
    // A SERP story that carries only `feedback: { id }` — no *_with_context wrapper at all.
    const bare = { id: "P9", post_id: "P9", permalink_url: "https://x/9", actors: [{ id: "a9", name: "A", url: "u" }],
                   comet_sections: { feedback: { id: "fb:P9" }, content: {} } };
    const h = makeCtx({});
    h.ctx.window.__soloGql.captures[0].response = searchResponse([bare]);
    const res = h.ctx.window.__soloGqlExtractCapability("fb.group.search_posts", {});
    check("a bare feedback:{id} is still found", (res.items[0] || {}).feedback_id === "fb:P9", (res.items[0] || {}).feedback_id);
  }

  console.log("\n== a missing feedback id is refused loudly, not returned as zero comments ==");
  {
    const h = makeCtx({});
    const res = await call(h, {});
    check("it refuses with a reason", typeof res.error === "string" && /feedback_id/.test(res.error), res.error);
    check("no request was made", h.sent.length === 0, h.sent.length);
  }

  console.log("\n== every page of top-level comments is walked ==");
  {
    const h = makeCtx({});
    const res = await call(h, { feedback_id: "fb:P1" });
    check("all 5 comments came back", res.count === 5, res.count);
    check("found is true", res.found === true, res.found);
    const cursors = h.sent.filter((s) => s.query === COMMENT_Q).map((s) => s.vars.commentsAfterCursor);
    check("page 1 asked with a null cursor", cursors[0] === null, cursors);
    check("it followed end_cursor forward", JSON.stringify(cursors) === JSON.stringify([null, "cc:1", "cc:2"]), cursors);
    check("it stopped on has_next_page:false", (res.by_post[0] || {}).stopped_because === "end_of_connection", res.by_post);
    check("ids are in feed order", res.items.map((c) => c.id).join(",") === "c1,c2,c3,c4,c5", res.items.map((c) => c.id));
  }

  console.log("\n== author id, name and url stay together on one record ==");
  {
    const h = makeCtx({});
    const res = await call(h, { feedback_id: "fb:P1" });
    const c1 = res.items[0];
    check("the profile id is present", c1.actor.id === "u1", c1.actor);
    check("the name belongs to that id", c1.actor.name === "Person 1", c1.actor);
    check("the url belongs to that id", c1.actor.url === "https://www.facebook.com/p1", c1.actor);
    // The bridge redacts any field whose name contains "auth". Shipping the commenter under
    // `author` would silently strip exactly the identity this capability exists to collect.
    check("it is NOT named author — the bridge would redact it", c1.author === undefined, Object.keys(c1));
    check("the text is the comment body", c1.text === "comment c1", c1.text);
    check("created_time is the comment's own, not a nested reply's", c1.created_time === 1700000001, c1.created_time);
    check("the post it belongs to is recorded", c1.post_feedback_id === "fb:P1", c1.post_feedback_id);
  }

  console.log("\n== depth defaults to 1: direct comments only, no reply query at all ==");
  {
    const h = makeCtx({});
    const res = await call(h, { feedback_id: "fb:P1" });
    check("depth is reported as 1", res.depth === 1, res.depth);
    check("the reply query was never fired", h.sent.every((s) => s.query !== REPLY_Q), h.sent.map((s) => s.query));
    check("c1 still reports it HAS replies", res.items[0].reply_count === 2, res.items[0].reply_count);
    check("but none were fetched", res.items[0].replies.length === 0, res.items[0].replies);
    check("the count is the 5 direct comments", res.count === 5, res.count);
  }

  console.log("\n== depth 2 fetches replies, nested under their parent and counted once ==");
  {
    const h = makeCtx({});
    const res = await call(h, { feedback_id: "fb:P1", depth: 2 });
    const c1 = res.items.find((c) => c.id === "c1");
    check("c1 carries its 2 replies", c1.replies.map((r) => r.id).join(",") === "r1a,r1b", c1.replies.map((r) => r.id));
    // Addressed by the comment's FEEDBACK id, not its comment id. The reply query resolves a
    // feedback node; handed a comment id it finds one with no replies_connection and answers
    // empty for every comment while still reporting success.
    check("the reply query is addressed by the comment's feedback id",
      h.sent.filter((s) => s.query === REPLY_Q).map((s) => s.vars.id).join(",") === "cfb:c1",
      h.sent.filter((s) => s.query === REPLY_Q).map((s) => s.vars.id));
    check("the reply query carries no intent token — its shape has no such key",
      h.sent.filter((s) => s.query === REPLY_Q).every((s) => !("intentToken" in s.vars) && !("commentsIntentToken" in s.vars)),
      (h.sent.find((s) => s.query === REPLY_Q) || {}).vars);
    check("the reply query sends clientKey", h.sent.filter((s) => s.query === REPLY_Q).every((s) => s.vars.clientKey === null),
      (h.sent.find((s) => s.query === REPLY_Q) || {}).vars);
    check("the expansion token came off the parent", (h.sent.find((s) => s.query === REPLY_Q) || {}).vars.expansionToken === "xt:c1", (h.sent.find((s) => s.query === REPLY_Q) || {}).vars);
    check("replies are marked as not top-level", c1.replies.every((r) => r.top_level === false), c1.replies.map((r) => r.top_level));
    check("depth 2 does NOT recurse into r1a", h.sent.filter((s) => s.query === REPLY_Q && s.vars.id === "cfb:r1a").length === 0, h.sent.map((s) => s.vars.id));
    check("total is 5 direct + 2 replies", res.count === 7, res.count);
  }

  console.log("\n== depth 3 recurses one level further ==");
  {
    const h = makeCtx({});
    const res = await call(h, { feedback_id: "fb:P1", depth: 3 });
    const r1a = res.items.find((c) => c.id === "c1").replies.find((r) => r.id === "r1a");
    check("r1a carries its own reply", (r1a.replies[0] || {}).id === "r2a", r1a.replies);
    check("total is 5 + 2 + 1", res.count === 8, res.count);
  }
  {
    const h = makeCtx({});
    const res = await call(h, { feedback_id: "fb:P1", depth: 99 });
    check("depth is clamped to Facebook's own ceiling", res.depth === 4, res.depth);
  }

  console.log("\n== a REJECTED query is reported as rejected, never as 'no comments' ==");
  {
    const h = makeCtx({ gqlError: "Variable $__relay_internal__pv__X of required type Boolean! was not provided" });
    const res = await call(h, { feedback_id: "fb:P1" });
    check("it returns zero rows", res.count === 0, res.count);
    check("available stays true — the capability ran", res.available === true, res.available);
    check("the reason says the query was rejected", res.reason === "query_rejected", res.reason);
    check("the actual error text is carried out", /required type Boolean/.test((res.notes || []).join(" ")), res.notes);
    check("it did not keep paging into the error", h.sent.filter((s) => s.query === COMMENT_Q).length === 1, h.sent.length);
    check("stopped_because names it", (res.by_post[0] || {}).stopped_because === "graphql_error", res.by_post);
  }

  console.log("\n== an empty post is distinguishable from a rejected one ==");
  {
    const h = makeCtx({});
    h.ctx.window.__soloGql.origFetch = () => Promise.resolve({ text: () => Promise.resolve(JSON.stringify(
      { data: { node: { comment_rendering_instance_for_feed_location: { comments: { edges: [], page_info: { end_cursor: null, has_next_page: false } } } } } })) });
    const res = await call(h, { feedback_id: "fb:P1" });
    check("reason is no_comments, not query_rejected", res.reason === "no_comments", res.reason);
    check("no notes were raised", res.notes === undefined, res.notes);
  }

  console.log("\n== a query name missing from the registry names the alternatives ==");
  {
    const h = makeCtx({ registry: { [SEARCH_Q]: "doc_search" } });   // comment modules never loaded
    const res = await call(h, { feedback_id: "fb:P1" });
    check("it refuses with an explanation", /module registry/.test(res.error || ""), res.error);
    check("it reports what the registry DOES hold", Array.isArray(res.registry_probe) && res.registry_probe.length >= 5, res.registry_probe);
    check("the probe covers the sibling names", res.registry_probe.some((r) => r.query === REPLY_Q), res.registry_probe);
    check("no request went out", h.sent.length === 0, h.sent.length);
  }
  {
    const h = makeCtx({ registry: { [SEARCH_Q]: "doc_search" } });
    const res = await call(h, { feedback_id: "fb:P1", comment_doc_id: "9999" });
    check("an explicit doc_id overrides the missing module", res.count === 5, res.count);
    check("that doc_id was the one sent", h.sent[0].doc_id === "9999", h.sent[0]);
  }

  console.log("\n== budgets are enforced, including across the reply walk ==");
  {
    const h = makeCtx({});
    const res = await call(h, { feedback_id: "fb:P1", max_comment_pages: 2 });
    check("it stopped after 2 pages", res.count === 4, res.count);
    check("the cap is named, not disguised as the end", (res.by_post[0] || {}).stopped_because === "page_cap", res.by_post);
    check("it still hands back a cursor to resume from", (res.by_post[0] || {}).end_cursor === "cc:2", res.by_post);
  }
  {
    const h = makeCtx({});
    const res = await call(h, { feedback_id: "fb:P1", max_comments: 3, depth: 3 });
    check("max_comments is honoured", res.count === 3, res.count);
    check("the reply walk did not overrun it", res.items.length === 3, res.items.length);
  }

  console.log("\n== a transport failure is not reported as an empty thread ==");
  {
    // fetch does not reject on 4xx/5xx, and parseResponse answers null for anything that is not
    // GraphQL JSON. A login redirect, a checkpoint interstitial and a rate-limit page all land
    // here — and all three used to read downstream as "this post has no comments", which is the
    // one conclusion that stops a caller from ever retrying.
    const h = makeCtx({});
    h.ctx.window.__soloGql.parseResponse = () => null;
    const res = await call(h, { feedback_id: "fb:P1" });
    check("it does not claim the thread ended", (res.by_post[0] || {}).stopped_because === "unreadable_response", res.by_post);
    check("reason distinguishes it from an empty post", res.reason === "query_rejected", res.reason);
    check("the note says what actually happened", /GraphQL JSON/.test((res.notes || []).join(" ")), res.notes);
    check("available stays true — the capability itself ran", res.available === true, res.available);
  }

  console.log("\n== errors ALONGSIDE data keep the rows ==");
  {
    // @defer/@stream means one failed fragment populates `errors` while the rest of the payload
    // is good. Treating that as fatal throws away real comments for a fragment nobody asked for.
    const h = makeCtx({});
    const good = commentReply(null);
    h.ctx.window.__soloGql.origFetch = (url, o) => {
      const p = new URLSearchParams(o.body);
      if (p.get("fb_api_req_friendly_name") !== COMMENT_Q) return Promise.resolve({ text: () => Promise.resolve("{}") });
      const withErr = JSON.parse(JSON.stringify(good));
      withErr.errors = [{ message: "one deferred fragment failed" }];
      withErr.data.node.comment_rendering_instance_for_feed_location.comments.page_info = { end_cursor: null, has_next_page: false };
      return Promise.resolve({ text: () => Promise.resolve(JSON.stringify(withErr)) });
    };
    const res = await call(h, { feedback_id: "fb:P1" });
    check("the rows that DID arrive are kept", res.count === 2, res.count);
    check("the partial failure is still reported", /partial/.test((res.notes || []).join(" ")), res.notes);
    check("it is not reported as a rejected query", (res.by_post[0] || {}).stopped_because !== "graphql_error", res.by_post);
  }

  console.log("\n== a walk cut short by the budget says so, even mid-connection ==");
  {
    // The bug this pins: `stopped` initialises to end_of_connection and the continuation only
    // rewrote it when has_next_page was false. A walk stopped by the comment budget while more
    // pages remained therefore reported the thread as FINISHED — so a caller had no reason to
    // come back, and no way to learn there was a rest.
    const h = makeCtx({});
    const res = await call(h, { feedback_id: "fb:P1", max_comments: 2 });
    check("it does not claim the connection ended", (res.by_post[0] || {}).stopped_because === "comment_cap", res.by_post);
    check("it reports there IS more", (res.by_post[0] || {}).has_next_page === true, res.by_post);
    check("and hands back where to resume", (res.by_post[0] || {}).end_cursor === "cc:1", res.by_post);
  }

  console.log("\n== a nested reply thread's cursor never hijacks the top-level walk ==");
  {
    // A comment page carries a replies_connection under every threaded comment, each with its
    // own page_info. A payload-wide search for "the page_info" can return one of those and send
    // the top-level walk down a different connection — silently, and with plausible rows.
    const h = makeCtx({});
    const asked = [];
    h.ctx.window.__soloGql.origFetch = (url, o) => {
      const p = new URLSearchParams(o.body);
      if (p.get("fb_api_req_friendly_name") !== COMMENT_Q) return Promise.resolve({ text: () => Promise.resolve("{}") });
      const cur = JSON.parse(p.get("variables")).commentsAfterCursor;
      asked.push(cur);
      const body = commentReply(cur);
      const edges = body.data.node.comment_rendering_instance_for_feed_location.comments.edges;
      edges.forEach((e) => {
        e.node.replies_connection = { edges: [], page_info: { end_cursor: "WRONG", has_next_page: true } };
      });
      return Promise.resolve({ text: () => Promise.resolve(JSON.stringify(body)) });
    };
    const res = await call(h, { feedback_id: "fb:P1" });
    const cursors = asked;
    check("it followed the top-level cursors only", JSON.stringify(cursors) === JSON.stringify([null, "cc:1", "cc:2"]), cursors);
    check("the nested cursor was never requested", cursors.indexOf("WRONG") === -1, cursors);
    check("all 5 comments still came back", res.count === 5, res.count);
  }

  console.log("\n== a comment walk can be RESUMED where it stopped ==");
  {
    // Facebook pages comments about ten at a time, so page_cap is the normal case, not a corner.
    // Without a way to feed the cursor back, every later job re-walks from the head and returns
    // the same first rows — four legs that each look like a success and collectively collect one
    // page. These tests assert on WHICH cursors went out, not just on the row count.
    const leg1 = makeCtx({});
    const r1 = await call(leg1, { feedback_id: "fb:P1", max_comment_pages: 1 });
    check("leg 1 is cut by the budget and says so", (r1.by_post[0] || {}).stopped_because === "page_cap", r1.by_post);
    check("leg 1 reports it is resumable", (r1.by_post[0] || {}).resumable === true, r1.by_post);
    check("leg 1 started at the head", leg1.sent[0].vars.commentsAfterCursor === null, leg1.sent[0].vars);
    const cur = (r1.by_post[0] || {}).end_cursor;
    check("leg 1 hands back a cursor", cur === "cc:1", cur);

    const leg2 = makeCtx({});
    const r2 = await call(leg2, { feedback_id: "fb:P1", start_cursor: cur });
    // The whole point: a resuming leg that still asks for the head returns the same first rows
    // every time while reporting success, and the run silently collects nothing new.
    const asked = leg2.sent.filter((x) => x.query === COMMENT_Q).map((x) => x.vars.commentsAfterCursor);
    check("leg 2 never asks for the head", asked.indexOf(null) === -1, asked);
    check("leg 2 starts at the given cursor", asked[0] === "cc:1", asked);
    check("leg 2 echoes the start_cursor back", (r2.by_post[0] || {}).start_cursor === "cc:1", r2.by_post);

    const ids1 = r1.items.map((c) => c.id), ids2 = r2.items.map((c) => c.id);
    check("leg 2 returns only its own slice", ids2.join(",") === "c3,c4,c5", ids2);
    check("no row is collected twice", ids1.filter((i) => ids2.indexOf(i) > -1).length === 0, { ids1, ids2 });
    check("the two legs cover the whole thread", ids1.concat(ids2).join(",") === "c1,c2,c3,c4,c5", ids1.concat(ids2));
    check("the last leg says the thread ended", (r2.by_post[0] || {}).stopped_because === "end_of_connection", r2.by_post);
    check("and is not resumable", (r2.by_post[0] || {}).resumable === false, r2.by_post);
  }
  {
    // A cursor is per POST, so a multi-post job resumes through a map keyed by feedback id —
    // one shared cursor would send the second post into the first post's thread.
    const h = makeCtx({});
    await call(h, { feedback_ids: ["fb:P1", "fb:P2"], max_comment_pages: 1,
                    start_cursors: { "fb:P2": "cc:2" } });
    const byId = {};
    h.sent.filter((x) => x.query === COMMENT_Q).forEach((x) => { (byId[x.vars.id] = byId[x.vars.id] || []).push(x.vars.commentsAfterCursor); });
    check("the post with no cursor starts at the head",
      byId["fb:P1"].length === 1 && byId["fb:P1"][0] === null, byId);
    check("the post with a cursor resumes from it",
      byId["fb:P2"].length === 1 && byId["fb:P2"][0] === "cc:2", byId);
    check("one post's cursor never leaks into the other",
      byId["fb:P1"].indexOf("cc:2") === -1, byId);
  }

  console.log("\n== several posts in one call stay separated ==");
  {
    const h = makeCtx({});
    const res = await call(h, { feedback_ids: ["fb:P1", "fb:P2"] });
    check("both posts were walked", res.by_post.length === 2, res.by_post.map((p) => p.feedback_id));
    check("requests are serial, not a burst", h.sent.filter((s) => s.query === COMMENT_Q).length === 6, h.sent.length);
    check("every row says which post it came from", new Set(res.items.map((c) => c.post_feedback_id)).size === 2, res.items.map((c) => c.post_feedback_id));
    check("the ids requested are the ids given", h.sent.filter((s) => s.query === COMMENT_Q).map((s) => s.vars.id).join(",") === "fb:P1,fb:P1,fb:P1,fb:P2,fb:P2,fb:P2", h.sent.map((s) => s.vars.id));
  }

  console.log("\n== the request Facebook is sent matches the shape it expects ==");
  {
    const h = makeCtx({});
    const res0 = await call(h, { feedback_id: "fb:P1" });
    const v = h.sent[0].vars;
    check("commentsAfterCount is -1", v.commentsAfterCount === -1, v);
    check("feedLocation is the commenting surface", v.feedLocation === "DEDICATED_COMMENTING_SURFACE", v.feedLocation);
    check("the feedback id is the id variable", v.id === "fb:P1", v.id);
    check("scale came from Facebook's own WebPixelRatio", v.scale === 2, v.scale);
    check("useDefaultActor is false", v.useDefaultActor === false, v.useDefaultActor);
    // NOT an invented literal. The ordering token is per-story and published by Facebook in the
    // story's selectable_intents; with no story to read, null is sent — which is exactly what
    // the reference implementation does. A plausible-looking constant would either be rejected
    // or silently swapped for the server's ranked/FILTERED default, returning a subset of the
    // thread that looks identical to the whole of it.
    check("with no story to read, the intent is null, not a guessed constant", v.commentsIntentToken === null, v.commentsIntentToken);
    check("the UFI provider Relay always merges is present", v.__relay_internal__pv__CometUFIReactionEnableShortNamerelayprovider === true, v);
    // The bridge redacts any key containing token/auth/secret/session/csrf — on job INPUTS as
    // well as on collected records — so an input named intent_token could never be set by a job.
    const SENSITIVE = /cookie|token|secret|password|passwd|pwd|otp|auth|session|bearer|csrf|xsrf/;
    const badInputs = ["feedback_id", "feedback_ids", "max_comment_pages", "max_comments", "depth",
      "comment_intent", "comment_doc_id", "reply_doc_id", "with_comments"].filter((k) => SENSITIVE.test(k));
    check("no input name trips the bridge's sanitizer", badInputs.length === 0, badInputs);
    const badOutputs = Object.keys(res0).concat(Object.keys(res0.items[0] || {})).filter((k) => SENSITIVE.test(k));
    check("no output field name trips it either", badOutputs.length === 0, badOutputs);
  }
  {
    const h = makeCtx({});
    await call(h, { feedback_id: "fb:P1", comment_intent: "CHRONOLOGICAL_UNFILTERED_INTENT_V1" });
    check("the intent token is overridable per job", h.sent[0].vars.commentsIntentToken === "CHRONOLOGICAL_UNFILTERED_INTENT_V1", h.sent[0].vars.commentsIntentToken);
  }

  console.log("\n== provider variables are read off the artifact, not hardcoded ==");
  {
    const h = makeCtx({ providers: { [COMMENT_Q]: { __relay_internal__pv__Xrelayprovider: { get: () => true } } } });
    await call(h, { feedback_id: "fb:P1" });
    check("the declared provider was sent", h.sent[0].vars.__relay_internal__pv__Xrelayprovider === true, h.sent[0].vars);
  }
  {
    // A provider that throws must not take the request down with it.
    const h = makeCtx({ providers: { [COMMENT_Q]: {
      good: { get: () => false },
      bad: { get: () => { throw new Error("not ready"); } },
    } } });
    const res = await call(h, { feedback_id: "fb:P1" });
    check("one unresolvable provider does not lose the rest", h.sent[0].vars.good === false, h.sent[0].vars);
    check("the call still succeeded", res.count === 5, res.count);
  }
  {
    // Providers seed the variables; anything explicit must still win.
    const h = makeCtx({ providers: { [COMMENT_Q]: { id: { get: () => "WRONG" }, scale: { get: () => 99 } } } });
    await call(h, { feedback_id: "fb:P1" });
    check("an explicit variable is never overwritten by a provider", h.sent[0].vars.id === "fb:P1", h.sent[0].vars.id);
    check("scale likewise", h.sent[0].vars.scale === 2, h.sent[0].vars.scale);
  }

  console.log("\n== with_comments composes search and comments in ONE job ==");
  {
    const stories = [storyNode("P1", "fb:P1", "plain"), storyNode("P2", "fb:P2", "plain")];
    const h = makeCtx({ seedResponse: searchResponse(stories) });
    h.ctx.window.__soloGql.captures[0].response = searchResponse(stories);
    const res = await h.ctx.window.__soloGqlPaginate("fb.group.search_posts", { max_pages: 1, with_comments: 2 });
    check("the posts still came back", res.count === 2, res.count);
    check("post 1 carries its comments", (res.items[0].comments || []).length === 5, (res.items[0].comments || []).length);
    check("post 2 carries its own, not post 1's", (res.items[1].comments || []).every((c) => c.post_feedback_id === "fb:P2"), (res.items[1].comments || []).map((c) => c.post_feedback_id));
    check("a per-post summary is reported", (res.comments_by_post || []).length === 2, res.comments_by_post);
    check("the total is reported", res.comments_count === 10, res.comments_count);
    // The story published two intents; the UNFILTERED one is the one that must be used, or the
    // server answers with its ranked/filtered subset and the thread silently looks complete.
    check("the story's own unfiltered intent reached the wire",
      h.sent.filter((s) => s.query === COMMENT_Q).every((s) => s.vars.commentsIntentToken === "RANKED_UNFILTERED_INTENT_V1"),
      (h.sent.find((s) => s.query === COMMENT_Q) || {}).vars.commentsIntentToken);
    check("the filtered intent was rejected in favour of it",
      (h.sent.find((s) => s.query === COMMENT_Q) || {}).vars.commentsIntentToken !== "RANKED_FILTERED_INTENT_V1",
      (h.sent.find((s) => s.query === COMMENT_Q) || {}).vars.commentsIntentToken);
    check("no comment page was fetched for a third post", h.sent.filter((s) => s.query === COMMENT_Q).length === 6, h.sent.filter((s) => s.query === COMMENT_Q).length);
  }
  {
    const stories = [storyNode("P1", "fb:P1", "plain"), storyNode("P2", "fb:P2", "plain")];
    const h = makeCtx({ seedResponse: searchResponse(stories) });
    h.ctx.window.__soloGql.captures[0].response = searchResponse(stories);
    const res = await h.ctx.window.__soloGqlPaginate("fb.group.search_posts", { max_pages: 1, with_comments: 1 });
    check("with_comments:1 touches only the first post", h.sent.filter((s) => s.query === COMMENT_Q).length === 3, h.sent.filter((s) => s.query === COMMENT_Q).length);
    check("the second post is left without a comments field", res.items[1].comments === undefined, res.items[1].comments);
  }
  {
    // Replies must not be re-attached at the top level — that would double every threaded comment.
    const stories = [storyNode("P1", "fb:P1", "plain")];
    const h = makeCtx({ seedResponse: searchResponse(stories) });
    h.ctx.window.__soloGql.captures[0].response = searchResponse(stories);
    const res = await h.ctx.window.__soloGqlPaginate("fb.group.search_posts", { max_pages: 1, with_comments: 1, depth: 2 });
    check("only direct comments sit at the top level", (res.items[0].comments || []).length === 5, (res.items[0].comments || []).length);
    check("the replies are nested, and counted once overall", res.comments_count === 7, res.comments_count);
  }
  {
    // A post with no feedback id must SAY so rather than silently returning no comments.
    const bare = { id: "P8", post_id: "P8", permalink_url: "https://x/8", actors: [{ id: "a8", name: "A", url: "u" }], comet_sections: {} };
    const h = makeCtx({ seedResponse: searchResponse([bare]) });
    h.ctx.window.__soloGql.captures[0].response = searchResponse([bare]);
    const res = await h.ctx.window.__soloGqlPaginate("fb.group.search_posts", { max_pages: 1, with_comments: 1 });
    check("it names why comments were skipped", res.comments_skipped === "no_feedback_id_on_items", res.comments_skipped);
    check("the posts themselves are unaffected", res.count === 1, res.count);
  }
  {
    // A comment failure must never destroy the posts that were already collected.
    const stories = [storyNode("P1", "fb:P1", "plain")];
    const h = makeCtx({ seedResponse: searchResponse(stories), gqlError: "rejected" });
    h.ctx.window.__soloGql.captures[0].response = searchResponse(stories);
    const res = await h.ctx.window.__soloGqlPaginate("fb.group.search_posts", { max_pages: 1, with_comments: 1 });
    check("the posts survive a comment failure", res.count === 1, res.count);
    check("the failure is reported alongside them", /rejected/.test((res.comments_notes || []).join(" ")), res.comments_notes);
  }

  console.log("\n== existing capabilities are untouched by the provider seeding ==");
  {
    const h = makeCtx({
      seedQuery: "FriendsListPaginationQuery",
      registry: { FriendsListPaginationQuery: "doc_friends", [COMMENT_Q]: "doc_comments", [REPLY_Q]: "doc_replies" },
      providers: { FriendsListPaginationQuery: { pv: { get: () => true } } },
    });
    h.ctx.window.__soloGql.captures[0].docId = "doc_friends";
    h.ctx.window.__soloGql.captures[0].response = { data: { node: { pageItems: { edges: [], page_info: { end_cursor: "f:1", has_next_page: true } } } } };
    let seenVars = null;
    h.ctx.window.__soloGql.origFetch = (url, o) => {
      seenVars = JSON.parse(new URLSearchParams(o.body).get("variables"));
      return Promise.resolve({ text: () => Promise.resolve(JSON.stringify({ data: { node: { pageItems: { edges: [], page_info: { end_cursor: null, has_next_page: false } } } } })) });
    };
    await h.ctx.window.__soloGqlPaginate("fb.profile.friends", { max_pages: 1 });
    check("the captured variables are still carried forward", seenVars && seenVars.count === 5, seenVars);
    check("the declared provider is added underneath them", seenVars && seenVars.pv === true, seenVars);
  }
  {
    // A query whose module is NOT loaded must still replay on the captured doc_id — a missing
    // artifact means no providers to add, not a dead request.
    const h = makeCtx({ seedQuery: "FriendsListPaginationQuery" });   // registry has no friends entry
    h.ctx.window.__soloGql.captures[0].docId = "doc_friends";
    h.ctx.window.__soloGql.captures[0].response = { data: { node: { pageItems: { edges: [], page_info: { end_cursor: "f:1", has_next_page: true } } } } };
    let sentDoc = null;
    h.ctx.window.__soloGql.origFetch = (url, o) => {
      sentDoc = new URLSearchParams(o.body).get("doc_id");
      return Promise.resolve({ text: () => Promise.resolve(JSON.stringify({ data: { node: { pageItems: { edges: [], page_info: { end_cursor: null, has_next_page: false } } } } })) });
    };
    await h.ctx.window.__soloGqlPaginate("fb.profile.friends", { max_pages: 1 });
    check("an unloaded artifact falls back to the captured doc_id", sentDoc === "doc_friends", sentDoc);
  }

  console.log("\n" + (fail === 0 ? "ALL " + pass + " CHECKS PASSED" : pass + " passed, " + fail + " FAILED"));
  process.exit(fail === 0 ? 0 : 1);
}
run().catch((e) => { console.error(e); process.exit(1); });
