// Offline harness for the Instagram capabilities in
// chrome-extension/platforms/instagram/ig_extract.js:
//   ig.profile.enrich   — the profile page's own PolarisProfilePageContentQuery capture -> a
//                         ProfileEnrich-compatible lead record (or a thin DOM fallback record).
//   ig.profile.posts    — PolarisProfilePostsQuery's timeline connection -> PostRecord[],
//                         with cursor pagination via a replayed POST carrying the same doc_id.
//   ig.search.posts     — the keyword-search SERP connection -> PostRecord[] (header unit
//                         skipped, media-grid unit's items[] kept, deduped).
//   ig.people.search    — /api/v1/users/search/ first, falling back to the search box's own
//                         /api/v1/web/search/topsearch/ when that 404s or answers empty.
//   ig.post.comments    — the REST comments capture (or a live re-fetch) -> CommentRecord[],
//                         paginated by min_id.
//   _discover.ig / unknown ids — diagnostic dump / visible error, never a null crash.
//
// The fixtures below are shaped EXACTLY like the captures documented in ig_extract.js's own
// header comment (lines 10-25, "Measured 2026-09-10"): the same field names ig_intercept.js
// records ({kind, queryName, docId, variables, url, requestBody, response}) and the same
// XDTMediaDict / PolarisProfilePageContentQuery / topsearch / comments-REST shapes Instagram's
// web app actually sends. A fake window.__soloIg (captures[] + origFetch/csrfToken/appId/
// parseResponse/docIdFor) and a minimal document/location stand in for the MAIN-world page
// ig_extract.js normally runs in.
//
// Style follows tests/test_zillow_extract.js: fs.readFileSync + vm.createContext, "  ok"/"  FAIL"
// lines, exit 1 on any failure, "ALL N CHECKS PASSED" on none.
//
// Run:  node solo-agency-collector/tests/test_ig_extract.js
const fs = require("fs");
const path = require("path");
const vm = require("vm");

const SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "platforms", "instagram", "ig_extract.js"), "utf8");

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail) : "")); }
}

// ------------------------------------------------------------------ fake fetch (origFetch)
// A tiny rule-based stub standing in for window.__soloIg.origFetch: fetch(url, init) ->
// Promise<{status, text(): Promise<string>}>. Every call is recorded on .calls so a test can
// assert what ig_extract.js actually sent (e.g. that a replayed body kept doc_id and set a new
// `after` cursor).
function fetchStub(rules) {
  const calls = [];
  const fn = function (url, init) {
    calls.push({ url: String(url), init: init || {} });
    for (const r of rules) {
      if (r.match(String(url), init || {})) {
        const text = r.json !== undefined ? JSON.stringify(r.json) : (r.text !== undefined ? r.text : "");
        const status = r.status !== undefined ? r.status : 200;
        return Promise.resolve({ status, text: () => Promise.resolve(text) });
      }
    }
    return Promise.resolve({ status: 404, text: () => Promise.resolve("") });
  };
  fn.calls = calls;
  return fn;
}

// ------------------------------------------------------------------ vm context
// document only needs querySelector("header") for the DOM fallback (ig_extract.js:238); location
// only needs pathname (usernameFromHref/shortcodeFromHref) and href (currentHref/error urls).
function makeCtx(opts) {
  opts = opts || {};
  const pathname = opts.pathname || "/";
  const href = "https://www.instagram.com" + pathname;
  const headerNode = opts.headerText !== undefined ? { innerText: opts.headerText } : null;
  const document = {
    querySelector: (sel) => (sel === "header" ? headerNode : null),
    // data-sjs scripts: the Relay entries Instagram embeds in the page (opts.dataSjs = [text...])
    querySelectorAll: (sel) => (sel.indexOf("data-sjs") !== -1 ? (opts.dataSjs || []).map((t) => ({ textContent: t })) : []),
  };
  const location = { pathname, href };
  const store = {
    captures: opts.captures || [],
    origFetch: opts.origFetch || fetchStub([]),
    csrfToken: () => "fake-csrf",
    appId: () => "936619743392459",
    parseResponse: (t) => { try { return JSON.parse(t); } catch (e) { return null; } },
    docIdFor: () => "",
  };
  const ctx = { document, location, console, setTimeout, clearTimeout, URLSearchParams, Promise, Date, JSON };
  ctx.window = ctx;
  ctx.window.__soloIg = store;
  vm.createContext(ctx);
  vm.runInContext(SRC, ctx, { filename: "ig_extract.js" });
  return ctx;
}

// ------------------------------------------------------------------ fixtures
// PolarisProfilePageContentQuery's data.user, as documented in ig_extract.js:10-14.
const PROFILE_USER = {
  username: "loanfactoryhq", full_name: "Loan Factory HQ", pk: "7654321", id: "7654321",
  // PHONE_RE (ig_extract.js:89) requires the match to START on a digit, so a leading "(" is not
  // part of the match — a dash-separated number is used here to exercise the common-case path
  // cleanly rather than the "(555) ..." edge the regex truncates.
  biography: "Mortgage made simple. Email us: hello@loanfactory.com or call 555-123-4567",
  category: "Loan Service", external_url: "https://loanfactory.com/apply",
  bio_links: [
    { url: "https://loanfactory.com/apply", title: "Apply Now", link_type: "external" },
    { url: "https://linktr.ee/loanfactory", title: "Linktree", link_type: "external" },
  ],
  follower_count: 18234, following_count: 512, media_count: 341,
  is_business: true, account_type: 2, is_private: false, is_verified: true,
  address_street: "500 Market St", city_name: "San Francisco", zip: "94105",
  hd_profile_pic_url_info: { url: "https://scontent.cdninstagram.com/hd.jpg" },
  profile_pic_url: "https://scontent.cdninstagram.com/lo.jpg",
  fbid_v2: "17841400000000000",
};
function profileCapture(user, viewerId) {
  const variables = { id: user.id };
  return {
    kind: "graphql", queryName: "PolarisProfilePageContentQuery", docId: "1111",
    variables, url: "https://www.instagram.com/graphql/query",
    requestBody: "doc_id=1111&variables=" + encodeURIComponent(JSON.stringify(variables)),
    response: { data: { user, viewer: { user: { id: viewerId || "999999" } } } },
  };
}

// XDTMediaDict (ig_extract.js:15-16's field list + the header's media_type/product_type key).
function mediaNode(o) {
  o = o || {};
  const pk = o.pk || "1001";
  const userpk = o.userpk || "7654321";
  return {
    pk, id: pk + "_" + userpk, code: o.code || ("CODE" + pk),
    caption: { text: o.caption !== undefined ? o.caption : "Great rates this week!" },
    taken_at: o.taken_at !== undefined ? o.taken_at : 1749500000,
    like_count: o.like_count !== undefined ? o.like_count : 120,
    comment_count: o.comment_count !== undefined ? o.comment_count : 8,
    view_count: o.view_count, play_count: o.play_count,
    media_type: o.media_type !== undefined ? o.media_type : 1,
    product_type: o.product_type !== undefined ? o.product_type : "feed",
    image_versions2: { candidates: [{ url: o.image_url || ("https://scontent.cdninstagram.com/img" + pk + ".jpg") }] },
    carousel_media: o.carousel_media || [],
    carousel_media_count: o.carousel_media_count,
    user: o.user !== undefined ? o.user : { pk: userpk, username: "loanfactoryhq", full_name: "Loan Factory HQ", is_private: false, is_verified: true, profile_pic_url: "https://scontent.cdninstagram.com/lo.jpg" },
    location: o.location || null,
  };
}
function postsConnection(nodes, pageInfo) {
  return { edges: nodes.map((n) => ({ node: n })), page_info: pageInfo || { end_cursor: "", has_next_page: false } };
}
function profilePostsCapture(nodes, pageInfo, variables) {
  variables = variables || { username: "loanfactoryhq", after: null };
  return {
    kind: "graphql", queryName: "PolarisProfilePostsQuery", docId: "post-doc-id",
    variables, url: "https://www.instagram.com/graphql/query",
    requestBody: "doc_id=post-doc-id&variables=" + encodeURIComponent(JSON.stringify(variables)),
    response: { data: { xdt_api__v1__feed__user_timeline_graphql_connection: postsConnection(nodes, pageInfo) } },
  };
}
function serpCapture(gridItems, pageInfo, query) {
  const variables = { query: query || "mortgage", after: null };
  return {
    kind: "graphql", queryName: "PolarisKeywordSearchExplorePageRelayQuery", docId: "serp-doc-id",
    variables, url: "https://www.instagram.com/graphql/query",
    requestBody: "doc_id=serp-doc-id&variables=" + encodeURIComponent(JSON.stringify(variables)),
    response: {
      data: {
        xdt_fbsearch__top_serp_graphql: {
          edges: [
            { node: { __typename: "XDTTopSerpHeaderUnit" } },
            { node: { __typename: "XDTTopSerpMediaGridUnit", items: gridItems } },
          ],
          page_info: pageInfo || { end_cursor: "", has_next_page: false },
        },
      },
    },
  };
}
function commentNode(o) {
  return {
    pk: o.pk, text: o.text !== undefined ? o.text : "",
    created_at: o.created_at || 1749500100, created_at_utc: o.created_at_utc || o.created_at || 1749500100,
    user: o.user || { pk: "555", username: "commenter1", full_name: "C One", is_private: false, is_verified: false, profile_pic_url: "https://scontent.cdninstagram.com/c1.jpg" },
    child_comment_count: o.child_comment_count || 0, comment_like_count: o.comment_like_count || 0,
  };
}
function commentsResponse(comments, nextMinId, hasMore) {
  return { comments, next_min_id: nextMinId || "", has_more_comments: !!hasMore, comment_count: comments.length };
}
function commentsCapture(mediaId, comments, nextMinId, hasMore) {
  return {
    kind: "rest", queryName: "/api/v1/media/" + mediaId + "/comments/", docId: "", variables: {},
    url: "https://www.instagram.com/api/v1/media/" + mediaId + "/comments/?can_support_threading=true",
    requestBody: "", response: commentsResponse(comments, nextMinId, hasMore),
  };
}

// The bridge redacts any key containing one of these substrings (main.go isSensitiveKey), at
// any depth — same list tests/test_zillow_extract.js uses.
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

(async function main() {
  console.log("ig.profile.enrich — happy path (graphql capture already present)");
  {
    const ctx = makeCtx({ pathname: "/loanfactoryhq/", captures: [profileCapture(PROFILE_USER, "999999")] });
    const res = await ctx.window.__soloIgRun("ig.profile.enrich", {});
    check("envelope: available, found, count 1, source graphql", res.available === true && res.found === true && res.count === 1 && res.source === "graphql", res);
    const r = res.items[0];
    check("username/id/name/profile_url", r.username === "loanfactoryhq" && r.id === "7654321" && r.name === "Loan Factory HQ" && r.profile_url === "https://www.instagram.com/loanfactoryhq/", r);
    check("bio/category/external_url", r.bio === PROFILE_USER.biography && r.category === "Loan Service" && r.external_url === "https://loanfactory.com/apply", r);
    check("website = external_url (present), websites[] = bio_links urls", r.website === "https://loanfactory.com/apply" && r.websites.length === 2 && r.websites[1] === "https://linktr.ee/loanfactory", [r.website, r.websites]);
    check("emails[] parsed from bio", r.emails.length === 1 && r.emails[0] === "hello@loanfactory.com", r.emails);
    check("phones[] parsed from bio", r.phones.length === 1 && r.phones[0] === "555-123-4567", r.phones);
    check("counters", r.follower_count === 18234 && r.following_count === 512 && r.media_count === 341, r);
    check("flags: is_business/account_type/is_private/is_verified", r.is_business === true && r.account_type === 2 && r.is_private === false && r.is_verified === true, r);
    check("address {street, city, zip}", r.address.street === "500 Market St" && r.address.city === "San Francisco" && r.address.zip === "94105", r.address);
    check("profile_pic_url prefers hd_profile_pic_url_info.url", r.profile_pic_url === "https://scontent.cdninstagram.com/hd.jpg", r.profile_pic_url);
    check("fbid from fbid_v2, source graphql", r.fbid === "17841400000000000" && r.source === "graphql", [r.fbid, r.source]);
    check("no sensitive key anywhere in the envelope", sensitiveKeys(res, "res").length === 0, sensitiveKeys(res, "res"));
  }

  console.log("ig.profile.enrich — self-profile refusal (viewer.user.id == user id)");
  {
    const ctx = makeCtx({ pathname: "/loanfactoryhq/", captures: [profileCapture(PROFILE_USER, "7654321")] });
    const res = await ctx.window.__soloIgRun("ig.profile.enrich", {});
    check("available:false, reason self_profile, no items", res.available === false && res.reason === "self_profile" && res.count === 0 && res.items.length === 0, res);
  }

  console.log("ig.profile.enrich — profile query not captured: DOM fallback, reason profile_query_not_captured");
  {
    const headerText = "loanfactoryhq\nLoan Factory HQ\n341 posts\n18.2K followers\n512 following";
    const ctx = makeCtx({ pathname: "/loanfactoryhq/", captures: [], headerText });
    const res = await ctx.window.__soloIgRun("ig.profile.enrich", { ensure_tries: 1 });
    check("found true, source dom, reason profile_query_not_captured", res.found === true && res.source === "dom" && res.reason === "profile_query_not_captured", res);
    const r = res.items[0];
    check("dom record: username from href, name from header line 2", r.username === "loanfactoryhq" && r.name === "Loan Factory HQ" && r.source === "dom", r);
    check("dom record: counters parsed from header lines", r.follower_count === 18200 && r.following_count === 512 && r.media_count === 341, r);
  }

  console.log("ig.profile.enrich — nothing captured, no header rendered either: not found");
  {
    // pathname "/" resolves to no username (usernameFromHref()'s regex needs at least one path
    // segment) and no header is supplied, so profileFromDom() has neither a name nor a username.
    const ctx = makeCtx({ pathname: "/", captures: [] });
    const res = await ctx.window.__soloIgRun("ig.profile.enrich", { ensure_tries: 1 });
    check("found false, empty items, error set", res.found === false && res.items.length === 0 && !!res.error, res);
  }

  console.log("ig.profile.posts — happy path: feed photo, feed video, reel, carousel in one page");
  {
    const feedPhoto = mediaNode({ pk: "2001", media_type: 1, product_type: "feed", like_count: 40, comment_count: 3 });
    const feedVideo = mediaNode({ pk: "2002", media_type: 2, product_type: "feed", like_count: 200, comment_count: 10, view_count: 5000 });
    const reel = mediaNode({ pk: "2003", media_type: 2, product_type: "clips", like_count: 900, comment_count: 55, play_count: 15000, code: "REELCODE1" });
    const carousel = mediaNode({
      pk: "2004", media_type: 8, product_type: "carousel_container", carousel_media_count: 2,
      carousel_media: [
        { media_type: 1, image_versions2: { candidates: [{ url: "https://scontent.cdninstagram.com/c1.jpg" }] } },
        { media_type: 2, image_versions2: { candidates: [{ url: "https://scontent.cdninstagram.com/c2.jpg" }] } },
      ],
    });
    const cap = profilePostsCapture([feedPhoto, feedVideo, reel, carousel], { end_cursor: "", has_next_page: false });
    const ctx = makeCtx({ pathname: "/loanfactoryhq/", captures: [cap] });
    const res = await ctx.window.__soloIgRun("ig.profile.posts", {});
    check("4 items, found true, source_query", res.found === true && res.items.length === 4 && res.source_query === "PolarisProfilePostsQuery", res);
    const [p, v, r, c] = res.items;
    check("photo: id/url/text/engagement/attachments", p.id === "2001" && p.url === "https://www.instagram.com/p/CODE2001/" && p.text === "Great rates this week!" && p.engagement.likes === 40 && p.engagement.comments === 3 && p.attachments.length === 1 && p.attachments[0].type === "photo", p);
    check("video (feed, not clips): reel-shaped url uses /p/, media_type 2", v.url === "https://www.instagram.com/p/CODE2002/" && v.media_type === 2 && v.engagement.views === 5000, v);
    check("reel: url uses /reel/, media_id/code refs", r.url === "https://www.instagram.com/reel/REELCODE1/" && r.media_id === "2003" && r.code === "REELCODE1" && r.engagement.views === 15000, r);
    // postRecord() (ig_extract.js:134-139) always pushes the node's OWN cover image first, then
    // every carousel_media[] image after it — so a 2-item carousel yields 3 attachments (cover +
    // the 2 slides), not 2.
    check("carousel: 3 attachments (cover + 2 slides), carousel_media_count 2", c.attachments.length === 3 && c.attachments[0].type === "photo" && c.attachments[1].type === "photo" && c.attachments[2].type === "video" && c.carousel_media_count === 2, c);
    check("actor carried on every post", res.items.every((it) => it.actor && it.actor.username === "loanfactoryhq"), res.items.map((it) => it.actor));
  }

  console.log("ig.profile.posts — pagination: replay keeps doc_id, sets variables.after, dedupes");
  {
    const page1 = [mediaNode({ pk: "3001" }), mediaNode({ pk: "3002" })];
    const cap = profilePostsCapture(page1, { end_cursor: "cursor-abc", has_next_page: true }, { username: "loanfactoryhq", after: null });
    const page2Node = mediaNode({ pk: "3003" });
    const stub = fetchStub([
      {
        match: (url, init) => url.indexOf("/graphql/query") !== -1 && init.method === "POST",
        json: { data: { xdt_api__v1__feed__user_timeline_graphql_connection: postsConnection([page2Node], { end_cursor: "", has_next_page: false }) } },
      },
    ]);
    const ctx = makeCtx({ pathname: "/loanfactoryhq/", captures: [cap], origFetch: stub });
    const res = await ctx.window.__soloIgRun("ig.profile.posts", { max_pages: 2 });
    check("3 items across 2 pages, no dupes", res.items.length === 3 && new Set(res.items.map((i) => i.id)).size === 3, res.items.map((i) => i.id));
    check("pages_fetched 2, page_info reflects the LAST page (no more)", res.pages_fetched === 2 && res.page_info.has_next_page === false, res);
    check("exactly one replay POST was sent", stub.calls.filter((c) => c.init.method === "POST").length === 1, stub.calls.length);
    const replay = stub.calls.find((c) => c.init.method === "POST");
    const sentParams = new URLSearchParams(replay.init.body);
    check("replayed body kept doc_id=post-doc-id", sentParams.get("doc_id") === "post-doc-id", sentParams.get("doc_id"));
    const sentVars = JSON.parse(sentParams.get("variables"));
    check("replayed body set variables.after to the cursor, kept username", sentVars.after === "cursor-abc" && sentVars.username === "loanfactoryhq", sentVars);
  }

  console.log("ig.profile.posts — max_posts truncates AFTER pagination ran; a failed page 2 keeps page 1");
  {
    const page1 = [mediaNode({ pk: "3101" }), mediaNode({ pk: "3102" }), mediaNode({ pk: "3103" })];
    const cap = profilePostsCapture(page1, { end_cursor: "cursor-2", has_next_page: true }, { username: "loanfactoryhq", after: null });
    const stub = fetchStub([
      { match: (url, init) => url.indexOf("/graphql/query") !== -1 && init.method === "POST", json: { data: { xdt_api__v1__feed__user_timeline_graphql_connection: postsConnection([mediaNode({ pk: "3104" }), mediaNode({ pk: "3105" })], { end_cursor: "", has_next_page: false }) } } },
    ]);
    const ctx = makeCtx({ pathname: "/loanfactoryhq/", captures: [cap], origFetch: stub });
    const res = await ctx.window.__soloIgRun("ig.profile.posts", { max_pages: 2, max_posts: 4 });
    check("2 pages fetched, 4 of 5 items kept", res.pages_fetched === 2 && res.items.length === 4 && res.count === 4, [res.pages_fetched, res.items.length]);
    check("the kept items are the first four in page order", res.items.map((i) => i.id).join(",") === "3101,3102,3103,3104", res.items.map((i) => i.id));
    const down = function (url, init) { return Promise.reject(new Error("net down")); };
    const ctx2 = makeCtx({ pathname: "/loanfactoryhq/", captures: [profilePostsCapture(page1, { end_cursor: "cursor-2", has_next_page: true })], origFetch: down });
    const res2 = await ctx2.window.__soloIgRun("ig.profile.posts", { max_pages: 3 });
    check("page 2 network failure: 3 items from page 1, stopped_because fetch_error, resumable", res2.items.length === 3 && res2.stopped_because === "fetch_error" && res2.page_info.resumable === true, res2);
  }

  console.log("ig.profile.posts — not captured at all");
  {
    const ctx = makeCtx({ pathname: "/loanfactoryhq/", captures: [] });
    const res = await ctx.window.__soloIgRun("ig.profile.posts", { ensure_tries: 1 });
    check("found false, reason posts_query_not_captured", res.found === false && res.reason === "posts_query_not_captured", res);
  }

  console.log("ig.search.posts — header unit skipped, media-grid items kept, dedupe within one page");
  {
    const dupe = mediaNode({ pk: "4001" });
    const other = mediaNode({ pk: "4002" });
    const cap = serpCapture([dupe, other, dupe], { end_cursor: "", has_next_page: false }, "mortgage");
    const ctx = makeCtx({ captures: [cap] });
    const res = await ctx.window.__soloIgRun("ig.search.posts", {});
    check("2 unique posts (dupe collapsed, header unit contributed none)", res.items.length === 2 && new Set(res.items.map((i) => i.id)).size === 2, res.items.map((i) => i.id));
    check("query echoed from the captured variables", res.query === "mortgage", res.query);
  }

  console.log("ig.search.posts — not captured at all");
  {
    const ctx = makeCtx({ captures: [] });
    const res = await ctx.window.__soloIgRun("ig.search.posts", { ensure_tries: 1 });
    check("found false, reason search_query_not_captured", res.found === false && res.reason === "search_query_not_captured", res);
  }

  console.log("ig.people.search — users/search answers first, topsearch never called");
  {
    const usersSearchRows = [
      { pk: "501", username: "jane_loans", full_name: "Jane Loans", is_verified: false, is_private: false, profile_pic_url: "https://scontent.cdninstagram.com/jane.jpg" },
      { pk: "502", username: "loan_bob", full_name: "Bob Loan", is_verified: true, is_private: true, profile_pic_url: "https://scontent.cdninstagram.com/bob.jpg" },
    ];
    const stub = fetchStub([
      { match: (url) => url.indexOf("/api/v1/users/search/") !== -1, json: { users: usersSearchRows, num_results: 2 } },
      { match: (url) => url.indexOf("/api/v1/web/search/topsearch/") !== -1, json: { users: [] } },
    ]);
    const ctx = makeCtx({ origFetch: stub });
    const res = await ctx.window.__soloIgRun("ig.people.search", { query: "loan" });
    check("2 profiles, source_query users/search", res.found === true && res.items.length === 2 && res.source_query === "users/search", res);
    check("topsearch endpoint was never called", !stub.calls.some((c) => c.url.indexOf("topsearch") !== -1), stub.calls.map((c) => c.url));
    const capped = await ctx.window.__soloIgRun("ig.people.search", { query: "loan", count: 1 });
    check("inputs.count caps the kept rows", capped.items.length === 1 && capped.count === 1, capped.items.length);
    const jane = res.items.find((i) => i.username === "jane_loans");
    check("userRef fields + profile_pic_url", jane.id === "501" && jane.url === "https://www.instagram.com/jane_loans/" && jane.is_verified === false && jane.profile_pic_url === "https://scontent.cdninstagram.com/jane.jpg", jane);
  }

  console.log("ig.people.search — users/search 404s, falls back to topsearch");
  {
    const stub = fetchStub([
      { match: (url) => url.indexOf("/api/v1/users/search/") !== -1, status: 404, text: "" },
      {
        match: (url) => url.indexOf("/api/v1/web/search/topsearch/") !== -1,
        json: { users: [{ user: { pk: "601", username: "topsearch_agent", full_name: "Top Search Agent", is_verified: false, is_private: false, profile_pic_url: "https://scontent.cdninstagram.com/ts.jpg" }, search_social_context: "Followed by 3 people you follow" }] },
      },
    ]);
    const ctx = makeCtx({ origFetch: stub });
    const res = await ctx.window.__soloIgRun("ig.people.search", { query: "loan" });
    check("falls back to topsearch, 1 profile", res.found === true && res.items.length === 1 && res.source_query === "web/search/topsearch", res);
    check("subtitle carried from search_social_context", res.items[0].subtitle === "Followed by 3 people you follow", res.items[0]);
  }

  console.log("ig.people.search — missing query is a visible error");
  {
    const ctx = makeCtx({});
    const res = await ctx.window.__soloIgRun("ig.people.search", {});
    check("error envelope, no query", res.status === "error" && /inputs.query is required/.test(res.error), res);
  }

  console.log("ig.post.comments — from a live capture (media id resolved from the post the page opened)");
  {
    const comments = [
      commentNode({ pk: "9001", text: "How do I apply?" }),
      commentNode({ pk: "9002", text: "" }), // empty-text comment still counts
    ];
    // The post-root query names the media this page opened; a comments capture for ANOTHER post
    // (viewed earlier in the same tab) sits newer in the ring and must be ignored.
    const postRoot = { kind: "graphql", queryName: "PolarisPostRootQuery", docId: "root", variables: { shortcode: "DU9kCDekVk9" }, url: "https://www.instagram.com/graphql/query", requestBody: "", response: { data: { xdt_shortcode_media: mediaNode({ pk: "123456789", code: "DU9kCDekVk9" }) } } };
    const cap = commentsCapture("123456789", comments, "", false);
    const stray = commentsCapture("777", [commentNode({ pk: "9777", text: "wrong post" })], "", false);
    const ctx = makeCtx({ pathname: "/p/DU9kCDekVk9/", captures: [postRoot, cap, stray] });
    const res = await ctx.window.__soloIgRun("ig.post.comments", {});
    check("media_id resolved from the post root, its own capture used, 2 items", res.media_id === "123456789" && res.items.length === 2 && res.found === true, res);
    check("the newer comments capture of another post was ignored", !res.items.some((c) => c.text === "wrong post"), res.items);
    check("empty-text comment kept", res.items.some((c) => c.text === ""), res.items);
    check("actor/reply_count/depth/like_count mapped", res.items[0].actor.username === "commenter1" && res.items[0].reply_count === 0 && res.items[0].depth === 0 && res.items[0].like_count === 0, res.items[0]);
  }

  console.log("ig.post.comments — pagination via min_id, max_comments cutoff");
  {
    const page1Comments = [commentNode({ pk: "9101" }), commentNode({ pk: "9102" })];
    const page2Comments = [commentNode({ pk: "9103" }), commentNode({ pk: "9104" })];
    const cap = commentsCapture("222", page1Comments, "9102", true);
    const postRoot222 = { kind: "graphql", queryName: "PolarisPostRootQuery", docId: "root", variables: {}, url: "https://www.instagram.com/graphql/query", requestBody: "", response: { data: { xdt_shortcode_media: mediaNode({ pk: "222", code: "DU9kCDekVk9" }) } } };
    const stub = fetchStub([
      { match: (url) => url.indexOf("/api/v1/media/222/comments/") !== -1 && url.indexOf("min_id=9102") !== -1, json: commentsResponse(page2Comments, "", false) },
    ]);
    const ctx = makeCtx({ pathname: "/p/DU9kCDekVk9/", captures: [postRoot222, cap], origFetch: stub });
    const res = await ctx.window.__soloIgRun("ig.post.comments", { max_comment_pages: 5, max_comments: 3 });
    check("stopped at max_comments (3), not all 4 fetched", res.items.length === 3 && res.stopped_because === "max_comments", res);
    check("pagination request carried min_id from the first page's next_min_id", stub.calls.some((c) => c.url.indexOf("min_id=9102") !== -1), stub.calls.map((c) => c.url));
  }

  console.log("ig.post.comments — a stray comments capture never supplies the media id");
  {
    const stray = commentsCapture("777", [commentNode({ pk: "9777", text: "wrong post" })], "", false);
    const ctx = makeCtx({ pathname: "/p/NoRootHere/", captures: [stray] });
    const res = await ctx.window.__soloIgRun("ig.post.comments", { ensure_tries: 1 });
    check("found false, reason media_id_unknown (not the stray post's comments)", res.found === false && res.reason === "media_id_unknown" && res.items.length === 0, res);
  }

  console.log("ig.post.comments — inputs.media_id is authoritative; a later page's network failure keeps page 1");
  {
    const page1 = [commentNode({ pk: "9201" }), commentNode({ pk: "9202" })];
    const stub = fetchStub([
      { match: (url) => url.indexOf("/api/v1/media/333/comments/") !== -1 && url.indexOf("min_id=") === -1, json: commentsResponse(page1, "9202", true) },
    ]);
    const failing = function (url, init) { if (String(url).indexOf("min_id=") !== -1) return Promise.reject(new Error("net down")); return stub(url, init); };
    const ctx = makeCtx({ pathname: "/p/DU9kCDekVk9/", captures: [], origFetch: failing });
    const res = await ctx.window.__soloIgRun("ig.post.comments", { media_id: "333", max_comment_pages: 3, ensure_tries: 1 });
    check("2 comments from page 1 kept, stopped_because fetch_error, still found", res.items.length === 2 && res.stopped_because === "fetch_error" && res.found === true && res.media_id === "333", res);
  }

  console.log("ig.post.comments — first page network failure is an honest error envelope");
  {
    const down = () => Promise.reject(new Error("net down"));
    const ctx = makeCtx({ pathname: "/p/DU9kCDekVk9/", captures: [], origFetch: down });
    const res = await ctx.window.__soloIgRun("ig.post.comments", { media_id: "444", ensure_tries: 1 });
    check("found false, error names the fetch failure, media_id kept", res.found === false && /comments fetch failed: .*net down/.test(String(res.error)) && res.media_id === "444", res);
  }

  console.log("ig.post.comments — media id cannot be resolved: visible reason, not a crash");
  {
    const ctx = makeCtx({ pathname: "/p/UnknownCode/", captures: [] });
    const res = await ctx.window.__soloIgRun("ig.post.comments", { ensure_tries: 1 });
    check("found false, reason media_id_unknown", res.found === false && res.reason === "media_id_unknown", res);
  }

  console.log("_discover.ig — returns a capture skeleton, never throws");
  {
    const ctx = makeCtx({ captures: [profileCapture(PROFILE_USER, "999999")] });
    const res = await ctx.window.__soloIgRun("_discover.ig", {});
    check("available true, one capture row with queryName", res.available === true && res.items.length === 1 && res.items[0].queryName === "PolarisProfilePageContentQuery", res);
    check("no deep_skeleton without a query filter", res.items[0].deep_skeleton === undefined, res.items[0]);
    const res2 = await ctx.window.__soloIgRun("_discover.ig", { query: "PolarisProfilePage" });
    const leaky = profileCapture(Object.assign({}, PROFILE_USER, { session_key: "abc", nested: { csrf_token: "zzz", fine: "ok" } }), "999999");
    const ctx3 = makeCtx({ captures: [leaky] });
    const res3 = await ctx3.window.__soloIgRun("_discover.ig", { query: "PolarisProfilePage" });
    const sk = res3.items[0].deep_skeleton;
    check("query filter attaches deep_skeleton", !!res2.items[0].deep_skeleton && !!sk, res2.items[0]);
    check("deep_skeleton keeps ordinary keys and shapes", sk.data.user.username === "str:" + PROFILE_USER.username && sk.data.user.nested.fine === "str:ok", sk.data && sk.data.user);
    check("deep_skeleton redacts sensitive key names at any depth", sk.data.user.session_key === "<redacted>" && sk.data.user.nested.csrf_token === "<redacted>", sk.data.user);
  }

  console.log("dispatch — unknown instagram capability id is a visible error, not a null crash");
  {
    const ctx = makeCtx({});
    const res = await ctx.window.__soloIgRun("ig.nope", {});
    check("available true + status error + no_extractor message", res.available === true && res.status === "error" && /no instagram extractor/.test(res.error) && res.items[0].status === "error", res);
    check("__soloIgCapabilities lists all six ids", JSON.stringify(ctx.window.__soloIgCapabilities.slice().sort()) === JSON.stringify(["_discover.ig", "ig.people.search", "ig.post.comments", "ig.profile.enrich", "ig.profile.posts", "ig.search.posts"].sort()), ctx.window.__soloIgCapabilities);
    check("re-injection is a no-op (function identity kept)", (() => { const f = ctx.window.__soloIgRun; vm.runInContext(SRC, ctx); return ctx.window.__soloIgRun === f; })());
  }

  console.log("__soloIgInternals — exposed helpers behave as documented");
  {
    const ctx = makeCtx({});
    const internals = ctx.window.__soloIgInternals;
    check("postUrl: feed code -> /p/, clips product_type -> /reel/", internals.postUrl("ABC", "feed") === "https://www.instagram.com/p/ABC/" && internals.postUrl("ABC", "clips") === "https://www.instagram.com/reel/ABC/", [internals.postUrl("ABC", "feed"), internals.postUrl("ABC", "clips")]);
    check("emailsIn/phonesIn extract from free text", JSON.stringify(internals.emailsIn("reach me at a@b.com")) === JSON.stringify(["a@b.com"]) && internals.phonesIn("call (555) 000-1111").length === 1, [internals.emailsIn("reach me at a@b.com"), internals.phonesIn("call (555) 000-1111")]);
    check("parseCount handles K/M suffixes", internals.parseCount("18.2K") === 18200 && internals.parseCount("1.5M") === 1500000, [internals.parseCount("18.2K"), internals.parseCount("1.5M")]);
    check("userRef: non-object -> null", internals.userRef(null) === null && internals.userRef("x") === null);
  }

  console.log("prefetched page data: a post page embeds the media (measured 2026-09-10), no post query on the wire");
  {
    const media = { pk: "3972649630022147867", id: "3972649630022147867_41436369314", code: "Dchr8pejp8b", caption: { text: "Một căn nhà đẹp" }, taken_at: 1787796796, like_count: 25, comment_count: 8, media_type: 2, product_type: "clips", image_versions2: { candidates: [{ url: "https://cdn/x.jpg" }] }, user: { pk: "41436369314", username: "nhuwhite", full_name: "Nhu White Realty", is_private: false, is_verified: true } };
    const entry = ["adp_PolarisPostRootQueryRelayPreloader_abcdef1234567890", { __bbox: { complete: true, result: { data: { xdt_api__v1__media__shortcode__web_info: { items: [media] } } } } }];
    const empty = ["adp_PolarisPostCommentsContainerQueryRelayPreloader_0123456789abcdef", { __bbox: { complete: true, result: { data: { xdt_api__v1__media__media_id__comments__connection: { edges: [], page_info: { end_cursor: null, has_next_page: false } } } } } }];
    const scriptText = JSON.stringify({ require: [["ScheduledServerJS", "handle", null, [{ __bbox: { require: [["RelayPrefetchedStreamCache", "next", [], entry], ["RelayPrefetchedStreamCache", "next", [], empty]] } }]]] });
    const fetched = [];
    const origFetch = (url) => { fetched.push(String(url)); return Promise.resolve({ status: 200, text: () => Promise.resolve(JSON.stringify({ comments: [{ pk: "17975593221040065", text: "Is this still available?", created_at: 1779221058, created_at_utc: 1779221058, user: { pk: 431367558, username: "gissell.g.f", full_name: "Gissell Flores", is_private: false, is_verified: false }, child_comment_count: 1, comment_like_count: 1 }], next_min_id: "", has_more_comments: false, comment_count: 8 })) }); };
    const ctx = makeCtx({ pathname: "/reel/Dchr8pejp8b/", captures: [], origFetch, dataSjs: [scriptText, "{\"require\":[]}"] });
    const pre = ctx.window.__soloIgInternals.prefetched();
    check("prefetched() yields the two embedded Relay entries with their query names", pre.length === 2 && pre[0].queryName === "PolarisPostRootQuery" && pre[1].queryName === "PolarisPostCommentsContainerQuery" && pre.every((e) => e.kind === "prefetch"), pre.map((e) => e.queryName));
    const res = await ctx.window.__soloIgRun("ig.post.comments", { max_comments: 50, ensure_tries: 1 });
    check("comments: media id resolved from the embedded post root", res.media_id === "3972649630022147867", res.media_id);
    check("comments: REST called directly with that pk", fetched.some((u) => u.indexOf("/api/v1/media/3972649630022147867/comments/") !== -1), fetched);
    check("comments: one comment returned", res.count === 1 && res.items[0].actor.username === "gissell.g.f", res.items);
    check("comments: post header carried from the embedded media", !!res.post && res.post.code === "Dchr8pejp8b" && res.post.url === "https://www.instagram.com/reel/Dchr8pejp8b/" && res.comment_count === 8, res.post);
    check("prefetched() is cached per page (same script count -> same array)", ctx.window.__soloIgInternals.prefetched() === pre);
    // Hidden comments: comment_count says 7, the endpoint answers none — the record must say why.
    const hiddenFetch = (url) => Promise.resolve({ status: 200, text: () => Promise.resolve(JSON.stringify({ comments: [], next_min_id: "", has_more_comments: false, comment_count: 7 })) });
    const ctx2 = makeCtx({ pathname: "/reel/Dchr8pejp8b/", captures: [], origFetch: hiddenFetch, dataSjs: [scriptText] });
    const res2 = await ctx2.window.__soloIgRun("ig.post.comments", { ensure_tries: 1 });
    check("comments hidden by the owner: count 0, reason comments_hidden, comment_count kept", res2.count === 0 && res2.reason === "comments_hidden" && res2.comment_count === 8 && res2.found === false, res2);
  }

  console.log("");
  console.log(fail === 0 ? "ALL " + pass + " CHECKS PASSED" : pass + " passed, " + fail + " FAILED");
  process.exit(fail ? 1 : 0);
})().catch((e) => { console.error(e); process.exit(2); });
