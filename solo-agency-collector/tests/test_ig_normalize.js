// Offline harness for chrome-extension/platforms/instagram/ig_normalize.js — the canonical
// normalizer that reshapes today's ig.* capability output into the shared
// {schema_version, kind, items[]} contract (chrome-extension/core/schema.js's root.SoloSchema).
//
// Style follows tests/test_zillow_normalize.js: the REAL ig_extract.js runs first over fixtures
// shaped like live captures, and its REAL output is what gets normalized — nothing here is
// invented. fs.readFileSync + vm.createContext + "  ok"/"  FAIL" lines, exit 1 on any failure,
// "ALL N CHECKS PASSED" on none.
//
// Run:  node solo-agency-collector/tests/test_ig_normalize.js
const fs = require("fs");
const path = require("path");
const vm = require("vm");

const IG_SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "platforms", "instagram", "ig_extract.js"), "utf8");
const NORM_SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "platforms", "instagram", "ig_normalize.js"), "utf8");
const SCHEMA_PATH = path.join(__dirname, "..", "chrome-extension", "core", "schema.js");

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail) : "")); }
}

const CAPTURED_AT = "2026-09-10T12:00:00.000Z";

// Same needle list schema.js / bridge-go's isSensitiveKey use, verbatim from
// tests/test_zillow_normalize.js.
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

const ALL_ITEMS = [];
function collect(canonical) {
  if (canonical && Array.isArray(canonical.items)) canonical.items.forEach((it) => ALL_ITEMS.push(it));
  return canonical;
}
function isPlainObj(v) { return !!v && typeof v === "object" && !Array.isArray(v); }

// ============================================================================================
// tests/test_ig_extract.js's fake fetch + vm context, copied verbatim so the REAL ig_extract.js
// extractors run over the fixtures below.
// ============================================================================================
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
function makeCtx(opts) {
  opts = opts || {};
  const pathname = opts.pathname || "/";
  const href = "https://www.instagram.com" + pathname;
  const headerNode = opts.headerText !== undefined ? { innerText: opts.headerText } : null;
  const document = { querySelector: (sel) => (sel === "header" ? headerNode : null) };
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
  vm.runInContext(IG_SRC, ctx, { filename: "ig_extract.js" });
  vm.runInContext(NORM_SRC, ctx, { filename: "ig_normalize.js" });
  return ctx;
}

// tests/test_ig_extract.js's fixture factories, copied verbatim so the fixtures here are shaped
// exactly like the ones already proven against the real extractor.
const PROFILE_USER = {
  username: "loanfactoryhq", full_name: "Loan Factory HQ", pk: "7654321", id: "7654321",
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

// ============================================================================================
// Tests
// ============================================================================================

(async function testProfileEnrich() {
  console.log("\n-- ig.profile.enrich (real extractor, graphql path) --");
  const ctx = makeCtx({ pathname: "/loanfactoryhq/", captures: [profileCapture(PROFILE_USER, "999999")] });
  const res = await ctx.window.__soloIgRun("ig.profile.enrich", {});
  check("extractor ran ok", res.available === true && res.found === true && res.items.length === 1, res);

  const before = JSON.stringify(res);
  const out = collect(ctx.__soloInstagramNormalize("ig.profile.enrich", res, { captured_at: CAPTURED_AT }));
  check("input not mutated", JSON.stringify(res) === before);
  check("schema_version 1, kind profile, 1 item", out.schema_version === 1 && out.kind === "profile" && out.items.length === 1, out);

  const it = out.items[0];
  const raw = res.items[0];
  check("platform/platform_id (item.id)/url (profile_url)/name/handle", it.platform === "instagram" && it.platform_id === raw.id && it.platform_id === "7654321" && it.url === raw.profile_url && it.name === "Loan Factory HQ" && it.handle === "loanfactoryhq", it);
  check("bio/category/website/websites", it.bio === raw.bio && it.category === "Loan Service" && it.website === "https://loanfactory.com/apply" && JSON.stringify(it.websites) === JSON.stringify(raw.websites), it);
  check("emails[]/phones[] passthrough", JSON.stringify(it.emails) === JSON.stringify(raw.emails) && JSON.stringify(it.phones) === JSON.stringify(raw.phones), [it.emails, raw.emails]);
  check("follower_count / verified / photo_url", it.follower_count === 18234 && it.verified === true && it.photo_url === raw.profile_pic_url, it);
  check("location[] built from address (street, 'city zip')", JSON.stringify(it.location) === JSON.stringify(["500 Market St", "San Francisco 94105"]), it.location);
  check("ext carries username/is_business/is_professional/account_type/is_private/address/bio_links/media_count/following_count/mutual_followers_count/fbid/source", it.ext.username === "loanfactoryhq" && it.ext.is_business === true && it.ext.is_professional === null && it.ext.account_type === 2 && it.ext.is_private === false && isPlainObj(it.ext.address) && Array.isArray(it.ext.bio_links) && it.ext.media_count === 341 && it.ext.following_count === 512 && it.ext.mutual_followers_count === null && it.ext.fbid === "17841400000000000" && it.ext.source === "graphql", it.ext);

  console.log("\n-- ig.profile.enrich: self-profile refusal normalizes to zero items --");
  const ctxSelf = makeCtx({ pathname: "/loanfactoryhq/", captures: [profileCapture(PROFILE_USER, "7654321")] });
  const resSelf = await ctxSelf.window.__soloIgRun("ig.profile.enrich", {});
  check("extractor refused (available:false)", resSelf.available === false && resSelf.reason === "self_profile", resSelf);
  const outSelf = collect(ctxSelf.__soloInstagramNormalize("ig.profile.enrich", resSelf, { captured_at: CAPTURED_AT }));
  check("normalizes to zero items (capabilityResult.items is [])", outSelf.items.length === 0, outSelf);

  console.log("\n-- ig.profile.enrich (real extractor, dom fallback) --");
  const headerText = "loanfactoryhq\nLoan Factory HQ\n341 posts\n18.2K followers\n512 following";
  const ctxDom = makeCtx({ pathname: "/loanfactoryhq/", captures: [], headerText });
  const resDom = await ctxDom.window.__soloIgRun("ig.profile.enrich", { ensure_tries: 1 });
  check("extractor fell back to dom", resDom.source === "dom" && resDom.items.length === 1, resDom);
  const outDom = collect(ctxDom.__soloInstagramNormalize("ig.profile.enrich", resDom, { captured_at: CAPTURED_AT }));
  const itDom = outDom.items[0];
  check("dom-fallback record still normalizes to one profile item", outDom.items.length === 1, outDom);
  check("dom record: no id -> platform_id empty, url still built from username", itDom.platform_id === "" && itDom.url === "https://www.instagram.com/loanfactoryhq/", itDom);
  check("dom record: fields absent on the raw item come back as null/'' rather than throwing", itDom.photo_url === "" && itDom.ext.is_professional === null && itDom.ext.fbid === "" && itDom.ext.mutual_followers_count === null, itDom);
  check("dom record: location[] empty (address is all-blank on the dom fallback)", Array.isArray(itDom.location) && itDom.location.length === 0, itDom.location);

  return testPeopleSearch();
})().catch((e) => { console.error(e); process.exit(1); });

function testPeopleSearch() {
  console.log("\n-- ig.people.search (real extractor) --");
  function fetchStubUsersSearch() {
    return function (url, init) {
      void init;
      const u = String(url);
      if (u.indexOf("/api/v1/users/search/") !== -1) {
        return Promise.resolve({
          status: 200,
          text: () => Promise.resolve(JSON.stringify({
            users: [
              { pk: "501", username: "jane_loans", full_name: "Jane Loans", is_verified: false, is_private: false, profile_pic_url: "https://scontent.cdninstagram.com/jane.jpg" },
              { pk: "502", username: "loan_bob", full_name: "Bob Loan", is_verified: true, is_private: true, profile_pic_url: "https://scontent.cdninstagram.com/bob.jpg" },
            ],
          })),
        });
      }
      return Promise.resolve({ status: 404, text: () => Promise.resolve("") });
    };
  }
  const ctx = makeCtx({ origFetch: fetchStubUsersSearch() });
  return ctx.window.__soloIgRun("ig.people.search", { query: "loan" }).then((res) => {
    check("extractor ran ok, 2 profiles", res.found === true && res.items.length === 2, res);
    const before = JSON.stringify(res);
    const out = collect(ctx.__soloInstagramNormalize("ig.people.search", res, { captured_at: CAPTURED_AT }));
    check("input not mutated", JSON.stringify(res) === before);
    check("schema_version 1, kind profile, 2 items", out.schema_version === 1 && out.kind === "profile" && out.items.length === 2, out);
    const jane = out.items.find((i) => i.handle === "jane_loans");
    check("platform_id (id)/url/name/handle/verified/photo_url", jane.platform === "instagram" && jane.platform_id === "501" && jane.url === "https://www.instagram.com/jane_loans/" && jane.name === "Jane Loans" && jane.verified === false && jane.photo_url === "https://scontent.cdninstagram.com/jane.jpg", jane);
    const bob = out.items.find((i) => i.handle === "loan_bob");
    check("ext.is_private/subtitle", bob.ext.is_private === true && typeof bob.ext.subtitle === "string", bob.ext);

    return testProfilePosts();
  });
}

function testProfilePosts() {
  console.log("\n-- ig.profile.posts (real extractor: photo, video, reel, carousel) --");
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
  return ctx.window.__soloIgRun("ig.profile.posts", {}).then((res) => {
    check("extractor ran ok, 4 posts", res.found === true && res.items.length === 4, res);
    const before = JSON.stringify(res);
    const out = collect(ctx.__soloInstagramNormalize("ig.profile.posts", res, { captured_at: CAPTURED_AT }));
    check("input not mutated", JSON.stringify(res) === before);
    check("schema_version 1, kind post, 4 items", out.schema_version === 1 && out.kind === "post" && out.items.length === 4, out);

    const photo = out.items.find((i) => i.platform_id === "2001");
    const video = out.items.find((i) => i.platform_id === "2002");
    const reelIt = out.items.find((i) => i.platform_id === "2003");
    const carouselIt = out.items.find((i) => i.platform_id === "2004");

    check("photo: platform/url/text/actor/created_at (non-zero epoch)/platform_time", photo.platform === "instagram" && photo.url.indexOf("/p/CODE2001/") !== -1 && photo.text === "Great rates this week!" && photo.actor.platform_id === "7654321" && photo.actor.name === "Loan Factory HQ" && photo.actor.type === "profile" && typeof photo.created_at === "string" && photo.platform_time === 1749500000, photo);
    check("photo: engagement {likes,comments,shares:0,views}, no media block (not video/reel)", photo.engagement.likes === 40 && photo.engagement.comments === 3 && photo.engagement.shares === 0 && (photo.media === null || photo.media === undefined), photo);
    check("photo: attachments[] carried, refs{media_id,code}", photo.attachments.length === 1 && photo.attachments[0].type === "photo" && photo.refs.media_id === "2001" && photo.refs.code === "CODE2001", photo);
    check("photo: ext{product_type,media_type,carousel_media_count,location}", photo.ext.product_type === "feed" && photo.ext.media_type === 1 && photo.ext.carousel_media_count === null && photo.ext.location === null, photo.ext);

    check("video (feed, media_type 2): media{type:'video',url,views}", video.media && video.media.type === "video" && video.media.views === 5000 && video.media.url === video.attachments[0].url, video);
    check("reel (product_type clips): media{type:'reel',...}, url uses /reel/", reelIt.media && reelIt.media.type === "reel" && reelIt.media.views === 15000 && reelIt.url.indexOf("/reel/REELCODE1/") !== -1, reelIt);
    check("carousel: no media block (only video/reel get one), 3 attachments (cover+2 slides)", (carouselIt.media === null || carouselIt.media === undefined) && carouselIt.attachments.length === 3 && carouselIt.ext.carousel_media_count === 2, carouselIt);

    return testSearchPosts();
  });
}

function testSearchPosts() {
  console.log("\n-- ig.search.posts (real extractor: header unit skipped, dedupe) --");
  const dupe = mediaNode({ pk: "4001" });
  const other = mediaNode({ pk: "4002" });
  const cap = serpCapture([dupe, other, dupe], { end_cursor: "", has_next_page: false }, "mortgage");
  const ctx = makeCtx({ captures: [cap] });
  return ctx.window.__soloIgRun("ig.search.posts", {}).then((res) => {
    check("extractor ran ok, 2 unique posts", res.found === true && res.items.length === 2, res);
    const out = collect(ctx.__soloInstagramNormalize("ig.search.posts", res, { captured_at: CAPTURED_AT }));
    check("normalizes to kind post, 2 items (same mapping as ig.profile.posts)", out.kind === "post" && out.items.length === 2, out);
    check("both items carry the shared post shape", out.items.every((i) => i.platform === "instagram" && i.refs && typeof i.refs.media_id === "string"), out.items);

    return testComments();
  });
}

function testComments() {
  console.log("\n-- ig.post.comments (real extractor) --");
  const comments = [
    commentNode({ pk: "9001", text: "How do I apply?", comment_like_count: 2 }),
    commentNode({ pk: "9002", text: "" }), // empty-text comment still counts
  ];
  const cap = commentsCapture("123456789", comments, "", false);
  // The media id is taken from the post the page opened (post-root capture), never from a
  // comments capture — see tests/test_ig_extract.js "stray comments capture".
  const postRoot = { kind: "graphql", queryName: "PolarisPostRootQuery", docId: "root", variables: {}, url: "https://www.instagram.com/graphql/query", requestBody: "", response: { data: { xdt_shortcode_media: mediaNode({ pk: "123456789", code: "DU9kCDekVk9" }) } } };
  const ctx = makeCtx({ pathname: "/p/DU9kCDekVk9/", captures: [postRoot, cap] });
  return ctx.window.__soloIgRun("ig.post.comments", {}).then((res) => {
    check("extractor ran ok, media_id resolved, 2 comments", res.found === true && res.media_id === "123456789" && res.items.length === 2, res);
    const before = JSON.stringify(res);
    const out = collect(ctx.__soloInstagramNormalize("ig.post.comments", res, { captured_at: CAPTURED_AT }));
    check("input not mutated", JSON.stringify(res) === before);
    check("schema_version 1, kind comment, 2 items", out.schema_version === 1 && out.kind === "comment" && out.items.length === 2, out);

    const c1 = out.items.find((i) => i.platform_id === "9001");
    const c2 = out.items.find((i) => i.platform_id === "9002");
    check("post_ref taken from the envelope's media_id (not from the comment item itself)", c1.post_ref === "123456789" && c2.post_ref === "123456789", [c1.post_ref, c2.post_ref]);
    check("text/actor/created_at/reply_count/depth/replies[]", c1.text === "How do I apply?" && c1.actor.name === "C One" && typeof c1.created_at === "string" && c1.reply_count === 0 && c1.depth === 0 && Array.isArray(c1.replies) && c1.replies.length === 0, c1);
    check("empty-text comment still normalizes (text stays '')", c2.text === "", c2);
    check("ext.like_count carried", c1.ext.like_count === 2 && c2.ext.like_count === 0, [c1.ext, c2.ext]);

    console.log("\n-- ig.post.comments: media-id-unknown envelope normalizes to zero items --");
    const ctxUnknown = makeCtx({ pathname: "/p/UnknownCode/", captures: [] });
    return ctxUnknown.window.__soloIgRun("ig.post.comments", { ensure_tries: 1 }).then((resUnknown) => {
      check("extractor could not resolve a media id", resUnknown.found === false && resUnknown.reason === "media_id_unknown", resUnknown);
      const outUnknown = collect(ctxUnknown.__soloInstagramNormalize("ig.post.comments", resUnknown, { captured_at: CAPTURED_AT }));
      check("normalizes to zero items", outUnknown.items.length === 0, outUnknown);

      return testDiscoverAndErrorsAndDefaults();
    });
  });
}

function testDiscoverAndErrorsAndDefaults() {
  console.log("\n-- _discover.ig / unknown capability id / generic error row --");
  const ctx = makeCtx({ captures: [profileCapture(PROFILE_USER, "999999")] });
  return ctx.window.__soloIgRun("_discover.ig", {}).then((discoverRes) => {
    check("an unrecognised OR diagnostic-only capability normalizes to null", ctx.__soloInstagramNormalize("_discover.ig", discoverRes, {}) === null && ctx.__soloInstagramNormalize("ig.totally.unknown", { items: [{ id: "1" }] }, {}) === null);

    return ctx.window.__soloIgRun("ig.nope", {}).then((errRes) => {
      check("extractor's own error envelope for an unknown capability id", errRes.status === "error" && errRes.items[0].status === "error", errRes);
      const errOut = collect(ctx.__soloInstagramNormalize("ig.nope", errRes, { captured_at: CAPTURED_AT }));
      check("normalizing an unmapped capability id (even with a status-only item) is null, not a crash", errOut === null, errOut);

      console.log("\n-- ig.profile.enrich: a generic status-only error row is skipped, not normalized --");
      const errRaw = { capability: "ig.profile.enrich", available: true, count: 0, status: "error", error: "boom", items: [{ capability: "ig.profile.enrich", status: "error", error: "boom", url: "https://www.instagram.com/x/" }] };
      const errRawOut = collect(ctx.__soloInstagramNormalize("ig.profile.enrich", errRaw, { captured_at: CAPTURED_AT }));
      check("error-only row normalizes to zero items", errRawOut.items.length === 0, errRawOut);

      console.log("\n-- captured_at defaults to now() when opts.captured_at is omitted --");
      const before2 = Date.now();
      const defaulted = ctx.__soloInstagramNormalize("ig.people.search", { items: [{ id: "1", username: "x", name: "X", url: "https://www.instagram.com/x/" }] }, {});
      const parsed = Date.parse(defaulted.items[0].captured_at);
      check("captured_at defaults to now()", isFinite(parsed) && Math.abs(parsed - before2) < 5000, defaulted.items[0].captured_at);

      return finishSuite();
    });
  });
}

function finishSuite() {
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
