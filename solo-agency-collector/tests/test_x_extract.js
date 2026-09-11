/*
 * Offline tests for chrome-extension/platforms/x/x_extract.js. The extractor is loaded into a
 * vm with a fake window (location, document, a window.__soloX store holding captures shaped like
 * X's GraphQL replies, an origFetch stub for cursor replays) and driven through
 * window.__soloXRun exactly as background.js does. The fixture shapes follow X's web client as
 * documented in x_extract.js's header; live shapes are re-confirmed with _discover.x.
 * Run: node solo-agency-collector/tests/test_x_extract.js
 */
"use strict";
const fs = require("fs");
const path = require("path");
const vm = require("vm");
const SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "platforms", "x", "x_extract.js"), "utf8");

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail).slice(0, 400) : "")); }
}
function fetchStub(rules) {
  const calls = [];
  const fn = function (url, init) {
    calls.push({ url: String(url), init: init || {} });
    for (const r of rules) {
      if (r.match(String(url), init || {})) {
        const text = r.json !== undefined ? JSON.stringify(r.json) : "";
        return Promise.resolve({ status: r.status !== undefined ? r.status : 200, text: () => Promise.resolve(text) });
      }
    }
    return Promise.resolve({ status: 404, text: () => Promise.resolve("") });
  };
  fn.calls = calls;
  return fn;
}
// A Date whose now() is the test's fake clock; parse/UTC/construction still work.
function fakeDate(now) {
  if (!now) return Date;
  const F = function () { return new (Function.prototype.bind.apply(Date, [null].concat(Array.prototype.slice.call(arguments))))(); };
  F.now = now; F.parse = Date.parse; F.UTC = Date.UTC;
  return F;
}
function makeCtx(opts) {
  opts = opts || {};
  const pathname = opts.pathname || "/";
  const store = {
    captures: opts.captures || [],
    origFetch: opts.origFetch || fetchStub([]),
    headersFor: () => ({ authorization: opts.noBearer ? "" : "Bearer AAAA-public-web-token", "x-csrf-token": "ct0value" }),
    parseResponse: (t) => { try { return JSON.parse(t); } catch (e) { return null; } },
    __auth: { bearer: opts.noBearer ? "" : "Bearer AAAA-public-web-token", extra: {} },
    queryIdFor: () => "",
  };
  // opts.now: a fake clock for the time-budget tests (the module only calls Date.now()).
  const ctx = { document: { cookie: "ct0=ct0value" }, location: { pathname, href: "https://x.com" + pathname, origin: "https://x.com" }, console, setTimeout, clearTimeout, URL, URLSearchParams, Promise, Date: fakeDate(opts.now), JSON, AbortController };
  ctx.window = ctx;
  ctx.window.__soloX = store;
  vm.createContext(ctx);
  vm.runInContext(SRC, ctx, { filename: "x_extract.js" });
  return ctx;
}

// ------------------------------------------------------------------ fixtures
function userResult(o) {
  o = o || {};
  return {
    __typename: "User", rest_id: o.id || "44196397",
    is_blue_verified: o.blue !== undefined ? o.blue : true,
    core: { screen_name: o.handle || "loanfactoryhq", name: o.name || "Loan Factory", created_at: "Wed Jan 15 10:00:00 +0000 2020" },
    avatar: { image_url: "https://pbs.twimg.com/profile_images/1/lf_normal.jpg" },
    location: { location: o.location !== undefined ? o.location : "San Jose, CA" },
    privacy: { protected: false },
    verification: { verified: false },
    professional: { professional_type: "Business", category: [{ id: 1, name: "Mortgage Broker" }] },
    legacy: {
      description: o.bio !== undefined ? o.bio : "FinTech mortgage platform. hello@loanfactory.com | (408) 555-0199 https://t.co/abc",
      entities: { url: { urls: [{ url: "https://t.co/xyz", expanded_url: "https://loanfactory.com" }] }, description: { urls: [{ url: "https://t.co/abc", expanded_url: "https://loanfactory.com/apply" }] } },
      followers_count: 18234, friends_count: 512, statuses_count: 3410, media_count: 120, listed_count: 9,
      url: "https://t.co/xyz", verified: false, verified_type: "", pinned_tweet_ids_str: ["1700000000000000001"]
    }
  };
}
function tweetResult(o) {
  o = o || {};
  const id = o.id || "1700000000000000001";
  const t = {
    __typename: "Tweet", rest_id: id,
    core: { user_results: { result: userResult({ id: o.userId || "44196397", handle: o.handle || "loanfactoryhq", name: o.userName || "Loan Factory" }) } },
    views: { count: o.views !== undefined ? String(o.views) : "1234" },
    legacy: {
      id_str: id, full_text: o.text !== undefined ? o.text : "Rates dipped to 6.375% this week. https://t.co/r1 #mortgage", created_at: o.created_at || "Tue Sep 09 12:00:00 +0000 2026",
      favorite_count: o.likes !== undefined ? o.likes : 51, reply_count: o.replies !== undefined ? o.replies : 6, retweet_count: 3, quote_count: 1, bookmark_count: 2, lang: "en",
      conversation_id_str: o.conversation || id, in_reply_to_status_id_str: o.replyTo || null, in_reply_to_screen_name: o.replyToHandle || null, is_quote_status: false,
      entities: { urls: [{ url: "https://t.co/r1", expanded_url: "https://loanfactory.com/rates" }], hashtags: [{ text: "mortgage" }], user_mentions: [], media: [] },
      extended_entities: o.media ? { media: o.media } : undefined
    }
  };
  if (o.note) t.note_tweet = { note_tweet_results: { result: { text: o.note } } };
  return t;
}
function tweetEntry(id, o) { return { entryId: "tweet-" + id, content: { entryType: "TimelineTimelineItem", itemContent: { itemType: "TimelineTweet", tweet_results: { result: tweetResult(Object.assign({ id }, o || {})) } } } }; }
function userEntry(o) { return { entryId: "user-" + (o.id || "1"), content: { entryType: "TimelineTimelineItem", itemContent: { itemType: "TimelineUser", user_results: { result: userResult(o) } } } }; }
function cursorEntry(value, type) { return { entryId: "cursor-" + (type || "bottom") + "-1", content: { entryType: "TimelineTimelineCursor", cursorType: type || "Bottom", value } }; }
function instructions(entries) { return [{ type: "TimelineClearCache" }, { type: "TimelineAddEntries", entries }]; }
function gqlCapture(name, variables, data, method) {
  return { kind: "graphql", queryId: "q" + name, queryName: name, method: method || "GET", variables, features: {}, url: "https://x.com/i/api/graphql/q" + name + "/" + name + "?variables=" + encodeURIComponent(JSON.stringify(variables)), requestBody: "", capturedAt: Date.now(), response: { data } };
}
const SENSITIVE = ["cookie", "token", "secret", "password", "passwd", "pwd", "otp", "authorization", "auth", "session", "bearer", "csrf", "xsrf"];
function sensitiveKeys(o, p, out) {
  out = out || [];
  if (Array.isArray(o)) { o.forEach((v, i) => sensitiveKeys(v, p + "[" + i + "]", out)); return out; }
  if (!o || typeof o !== "object") return out;
  Object.keys(o).forEach((k) => { const lk = k.toLowerCase(); if (SENSITIVE.some((n) => lk.indexOf(n) !== -1)) out.push(p + "." + k); sensitiveKeys(o[k], p + "." + k, out); });
  return out;
}

(async () => {
  console.log("x.profile.enrich — UserByScreenName capture");
  {
    const cap = gqlCapture("UserByScreenName", { screen_name: "loanfactoryhq" }, { user: { result: userResult() } });
    const ctx = makeCtx({ pathname: "/loanfactoryhq", captures: [cap] });
    const res = await ctx.window.__soloXRun("x.profile.enrich", {});
    const r = res.items[0] || {};
    check("envelope: available, found, count 1, source graphql", res.available === true && res.found === true && res.count === 1 && res.source_query === "UserByScreenName", res);
    check("id/handle/name/url from core + rest_id", r.id === "44196397" && r.username === "loanfactoryhq" && r.name === "Loan Factory" && r.profile_url === "https://x.com/loanfactoryhq", r);
    check("bio, location, website (expanded), websites incl. bio links", /FinTech/.test(r.bio) && r.location === "San Jose, CA" && r.website === "https://loanfactory.com" && r.websites.length === 2, [r.website, r.websites]);
    check("emails/phones parsed from the bio", r.emails[0] === "hello@loanfactory.com" && r.phones[0] === "(408) 555-0199", [r.emails, r.phones]);
    check("counters, verified (blue), professional category, joined", r.follower_count === 18234 && r.following_count === 512 && r.post_count === 3410 && r.is_verified === true && r.category === "Mortgage Broker" && r.professional_type === "Business" && typeof r.joined_time === "number", r);
    check("avatar url upgraded from _normal to _400x400", /_400x400\.jpg$/.test(r.profile_pic_url), r.profile_pic_url);
    check("no sensitive key anywhere in the envelope", sensitiveKeys(res, "res").length === 0, sensitiveKeys(res, "res"));
    const other = await makeCtx({ pathname: "/someoneelse", captures: [cap] }).window.__soloXRun("x.profile.enrich", { ensure_tries: 1 });
    check("a capture for a different handle is not used for this page", other.found === false && other.reason === "profile_query_not_captured", other);
  }

  console.log("x.profile.enrich — the 2026 shape: no legacy{}, fields in profile_bio / website / relationship_counts / tweet_counts");
  {
    const modern = { __typename: "User", rest_id: "1927204100308877312", is_blue_verified: false,
      core: { created_at: "Tue May 27 03:24:14 +0000 2025", name: "Loan Factory", screen_name: "loanfactoryus" },
      avatar: { image_url: "https://pbs.twimg.com/profile_images/9/lf_normal.jpg" }, location: { location: "2195 Tully Rd, San Jose, CA" },
      privacy: { protected: false }, verification: { verified: false }, verification_info: { is_identity_verified: false },
      profile_bio: { description: "#2 Mortgage Broker in America. apply@loanfactory.com", entities: { description: {}, url: { urls: [{ display_url: "loanfactory.com", expanded_url: "http://www.loanfactory.com/", url: "https://t.co/eLu3JA964l" }] } } },
      website: { url: "https://t.co/eLu3JA964l" }, relationship_counts: { followers: 72, following: 39 }, tweet_counts: { media_tweets: 871, tweets: 1179 }, action_counts: { favorites_count: 67 },
      professional: { category: [{ icon_name: "IconBriefcaseStroke", id: 192, name: "Financial Services" }], professional_type: "Business", rest_id: "1" }, business_account: {}, pinned_items: {} };
    const cap = gqlCapture("UserByScreenName", { screen_name: "loanfactoryus" }, { user: { result: modern } });
    const res = await makeCtx({ pathname: "/loanfactoryus", captures: [cap] }).window.__soloXRun("x.profile.enrich", {});
    const r = res.items[0] || {};
    check("bio from profile_bio, website expanded from its entities, counters from relationship_counts/tweet_counts", /Mortgage Broker/.test(r.bio) && r.website === "http://www.loanfactory.com/" && r.follower_count === 72 && r.following_count === 39 && r.post_count === 1179 && r.media_count === 871 && r.likes_given_count === 67, r);
    check("email from the bio, category Financial Services, business via professional_type, identity_verified false", r.emails[0] === "apply@loanfactory.com" && r.category === "Financial Services" && r.is_business === true && r.identity_verified === false && r.location === "2195 Tully Rd, San Jose, CA", r);
  }

  console.log("x.profile.posts — UserOriginalsTimeline (the Posts tab's name since 2026) is accepted");
  {
    const cap = gqlCapture("UserOriginalsTimeline", { userId: "1" }, { user: { result: { timeline: { timeline: { instructions: instructions([tweetEntry("1600000000000000001")]) } } } } });
    const res = await makeCtx({ pathname: "/recap_david", captures: [cap] }).window.__soloXRun("x.profile.posts", {});
    check("1 post from UserOriginalsTimeline, source_query says so", res.items.length === 1 && res.source_query === "UserOriginalsTimeline", res);
  }

  console.log("x.profile.posts — UserTweets page 1 captured, page 2 via cursor replay, retweet + long post unwrapped");
  {
    const page1 = instructions([
      tweetEntry("1700000000000000001", { likes: 10 }),
      { entryId: "tweet-1700000000000000002", content: { entryType: "TimelineTimelineItem", itemContent: { itemType: "TimelineTweet", tweet_results: { result: { __typename: "TweetWithVisibilityResults", tweet: tweetResult({ id: "1700000000000000002", note: "This is the long-form text of a note tweet that exceeds the classic limit." }) } } } } },
      cursorEntry("CURSOR-PAGE-2", "Bottom"), cursorEntry("CURSOR-TOP", "Top"),
    ]);
    const cap = gqlCapture("UserTweets", { userId: "44196397", count: 20 }, { user: { result: { timeline: { timeline: { instructions: page1 } } } } });
    const page2 = instructions([tweetEntry("1700000000000000003", { text: "RT @x: something" })]);
    const stub = fetchStub([{ match: (u) => u.indexOf("/UserTweets") !== -1 && u.indexOf("CURSOR-PAGE-2") !== -1, json: { data: { user: { result: { timeline: { timeline: { instructions: page2 } } } } } } }]);
    const ctx = makeCtx({ pathname: "/loanfactoryhq", captures: [cap], origFetch: stub });
    const res = await ctx.window.__soloXRun("x.profile.posts", { max_pages: 2 });
    check("3 posts across 2 pages, pages_fetched 2, no more cursor", res.items.length === 3 && res.pages_fetched === 2 && res.page_info.has_next_page === false, [res.items.length, res.pages_fetched, res.page_info]);
    check("replay sent the bearer + csrf headers X itself uses, GET with the cursor in variables", stub.calls.length === 1 && stub.calls[0].init.headers.authorization.indexOf("Bearer") === 0 && stub.calls[0].init.headers["x-csrf-token"] === "ct0value" && decodeURIComponent(stub.calls[0].url).indexOf('"cursor":"CURSOR-PAGE-2"') !== -1, stub.calls[0]);
    const p1 = res.items[0], p2 = res.items[1];
    check("post: id/url/actor/text/engagement/links/hashtags/post_id", p1.id === "1700000000000000001" && p1.url === "https://x.com/loanfactoryhq/status/1700000000000000001" && p1.actor.username === "loanfactoryhq" && /6\.375/.test(p1.text) && p1.engagement.likes === 10 && p1.engagement.views === 1234 && p1.links[0] === "https://loanfactory.com/rates" && p1.hashtags[0] === "mortgage" && p1.post_id === p1.id, p1);
    check("TweetWithVisibilityResults unwrapped; note tweet text preferred", p2.id === "1700000000000000002" && /long-form/.test(p2.text), p2);
    check("created_time parsed from X's date format", p1.created_time > 1700000000, p1.created_time);
    const none = await makeCtx({ pathname: "/loanfactoryhq", captures: [] }).window.__soloXRun("x.profile.posts", { ensure_tries: 1 });
    check("not captured -> reason posts_query_not_captured", none.found === false && none.reason === "posts_query_not_captured", none);
    const noBearer = await makeCtx({ pathname: "/loanfactoryhq", captures: [cap], noBearer: true }).window.__soloXRun("x.profile.posts", { max_pages: 3 });
    check("without a captured bearer the replay is skipped, page 1 kept, stopped_because says why", noBearer.items.length === 2 && /replay_failed/.test(noBearer.stopped_because), noBearer.stopped_because);
  }

  console.log("x.search.posts / x.people.search — SearchTimeline by product");
  {
    const top = gqlCapture("SearchTimeline", { rawQuery: "realtor", product: "Top" }, { search_by_raw_query: { search_timeline: { timeline: { instructions: instructions([tweetEntry("1800000000000000001"), tweetEntry("1800000000000000002")]) } } } });
    const latest = gqlCapture("SearchTimeline", { rawQuery: "realtor", product: "Latest" }, { search_by_raw_query: { search_timeline: { timeline: { instructions: instructions([tweetEntry("1800000000000000009")]) } } } });
    const people = gqlCapture("SearchTimeline", { rawQuery: "realtor", product: "People" }, { search_by_raw_query: { search_timeline: { timeline: { instructions: instructions([userEntry({ id: "1", handle: "jane_realtor", name: "Jane" }), userEntry({ id: "2", handle: "bob_homes", name: "Bob" })]) } } } });
    const ctx = makeCtx({ pathname: "/search", captures: [top, latest, people] });
    const t = await ctx.window.__soloXRun("x.search.posts", { mode: "top" });
    check("mode top: 2 posts, query echoed, mode reported", t.items.length === 2 && t.query === "realtor" && t.mode === "top", t);
    const l = await ctx.window.__soloXRun("x.search.posts", { mode: "latest" });
    check("mode latest: the Latest capture only", l.items.length === 1 && l.items[0].id === "1800000000000000009" && l.mode === "latest", l);
    const p = await ctx.window.__soloXRun("x.people.search", {});
    check("people: 2 profiles with handle/name/counters", p.items.length === 2 && p.items[0].username === "jane_realtor" && p.items[1].name === "Bob" && p.items[0].follower_count === 18234, p.items);
    const none = await makeCtx({ pathname: "/search", captures: [top] }).window.__soloXRun("x.people.search", { ensure_tries: 1 });
    check("people not captured -> search_query_not_captured", none.found === false && none.reason === "search_query_not_captured", none);
  }

  console.log("x.post.replies — TweetDetail: focal post + replies with depth");
  {
    const focal = "1900000000000000001";
    const ins = [{ type: "TimelineAddEntries", entries: [
      tweetEntry(focal, { replies: 2 }),
      { entryId: "conversationthread-1", content: { entryType: "TimelineTimelineModule", items: [
        { entryId: "conversationthread-1-tweet-1900000000000000002", item: { itemContent: { itemType: "TimelineTweet", tweet_results: { result: tweetResult({ id: "1900000000000000002", handle: "commenter1", userId: "7", text: "Great post", replyTo: focal, replyToHandle: "loanfactoryhq" }) } } } },
        { entryId: "conversationthread-1-tweet-1900000000000000003", item: { itemContent: { itemType: "TimelineTweet", tweet_results: { result: tweetResult({ id: "1900000000000000003", handle: "loanfactoryhq", text: "Thanks!", replyTo: "1900000000000000002", replyToHandle: "commenter1" }) } } } },
      ] } },
      cursorEntry("MORE-REPLIES", "Bottom"),
    ] }];
    const cap = gqlCapture("TweetDetail", { focalTweetId: focal }, { threaded_conversation_with_injections_v2: { instructions: ins } });
    const ctx = makeCtx({ pathname: "/loanfactoryhq/status/" + focal, captures: [cap] });
    const res = await ctx.window.__soloXRun("x.post.replies", {});
    check("post_id from the url, focal post carried, 2 replies", res.post_id === focal && res.post && res.post.id === focal && res.items.length === 2, res);
    check("depth 0 for a direct reply, 1 for a reply in the thread", res.items[0].depth === 0 && res.items[0].actor.username === "commenter1" && res.items[1].depth === 1, res.items.map((i) => [i.id, i.depth]));
    check("resumable with the bottom cursor when max_pages is 1", res.page_info.resumable === true && res.stopped_because === "page_cap_hit", res.page_info);
    const hidden = gqlCapture("TweetDetail", { focalTweetId: focal }, { threaded_conversation_with_injections_v2: { instructions: instructions([tweetEntry(focal, { replies: 5 })]) } });
    const h = await makeCtx({ pathname: "/loanfactoryhq/status/" + focal, captures: [hidden] }).window.__soloXRun("x.post.replies", {});
    check("reply_count > 0 but none served -> reason replies_hidden", h.count === 0 && h.reason === "replies_hidden" && h.reply_count === 5, h);
  }

  console.log("x.timeline.home — HomeLatestTimeline");
  {
    const cap = gqlCapture("HomeLatestTimeline", { count: 20 }, { home: { home_timeline_urt: { instructions: instructions([tweetEntry("1950000000000000001"), tweetEntry("1950000000000000002")]) } } });
    const res = await makeCtx({ pathname: "/home", captures: [cap] }).window.__soloXRun("x.timeline.home", {});
    check("2 posts, feed following", res.items.length === 2 && res.feed === "following" && res.source_query === "HomeLatestTimeline", res);
  }

  console.log("_discover.x — capture skeleton with redaction; dispatch guards");
  {
    const cap = gqlCapture("UserByScreenName", { screen_name: "x" }, { user: { result: userResult() }, session_token: "leak", nested: { csrf_value: "zzz", fine: "ok" } });
    const ctx = makeCtx({ pathname: "/x", captures: [cap] });
    const d = await ctx.window.__soloXRun("_discover.x", { query: "UserBy" });
    const sk = d.items[0].deep_skeleton;
    check("discover lists the capture with queryName/queryId and a skeleton on match", d.items[0].queryName === "UserByScreenName" && d.items[0].queryId === "qUserByScreenName" && !!sk, d.items[0]);
    check("skeleton redacts sensitive key names", sk.data.session_token === "<redacted>" && sk.data.nested.csrf_value === "<redacted>" && sk.data.nested.fine === "str:ok", sk.data);
    check("replay headers presence reported, never their value", d.replay_headers_seen === true && JSON.stringify(d).indexOf("public-web-token") === -1, d.session_headers_seen);
    const bad = await ctx.window.__soloXRun("x.nope", {});
    check("unknown capability -> visible error", bad.status === "error" && /no x extractor/.test(bad.error), bad);
    check("__soloXCapabilities lists 7 ids", ctx.window.__soloXCapabilities.length === 7, ctx.window.__soloXCapabilities);
    const man = ctx.window.__soloXManifest();
    check("__soloXManifest lists the operations with counts and variable keys, never a value", man.available === true && man.captureCount === 1 && man.manifest[0].queryName === "UserByScreenName" && man.manifest[0].variableKeys[0] === "screen_name" && JSON.stringify(man).indexOf("public-web-token") === -1, man);
    ctx.document.querySelector = (sel) => (String(sel).indexOf("SideNav_AccountSwitcher_Button") !== -1 ? {} : null);
    check("__soloXLoggedIn reads X's own chrome: account switcher present -> true", ctx.window.__soloXLoggedIn() === true);
    ctx.document.querySelector = (sel) => (String(sel).indexOf('a[href="/login"]') !== -1 ? {} : null);
    check("__soloXLoggedIn: login link and no article -> false", ctx.window.__soloXLoggedIn() === false);
  }

  console.log("x.post.replies — time budget: the pages in hand come back with the cursor, not a kill-timer error");
  {
    let now = 1000000;
    const focal = "1900000000000000001";
    const page1 = instructions([tweetEntry(focal, { replies: 40 }), tweetEntry("1900000000000000002", { replyTo: focal, handle: "c1", userId: "71" }), cursorEntry("REPLIES-2", "Bottom")]);
    const page2 = instructions([tweetEntry("1900000000000000003", { replyTo: focal, handle: "c2", userId: "72" }), cursorEntry("REPLIES-3", "Bottom")]);
    const cap = gqlCapture("TweetDetail", { focalTweetId: focal }, { threaded_conversation_with_injections_v2: { instructions: page1 } });
    const stub = fetchStub([{ match: (u) => u.indexOf("/TweetDetail") !== -1 && u.indexOf("REPLIES-2") !== -1, json: { data: { threaded_conversation_with_injections_v2: { instructions: page2 } } } }]);
    const slow = function (url, init) { now += 2500; return stub(url, init); };   // one replayed page costs 2.5s of a 5s budget
    const ctx = makeCtx({ pathname: "/loanfactoryhq/status/" + focal, captures: [cap], origFetch: slow, now: () => now });
    const res = await ctx.window.__soloXRun("x.post.replies", { max_pages: 5, time_budget_ms: 5000 });
    check("2 replies across 2 pages, then stopped_because time_budget with the next cursor kept", res.items.length === 2 && res.pages_fetched === 2 && res.stopped_because === "time_budget" && res.page_info.end_cursor === "REPLIES-3" && res.page_info.resumable === true, [res.items.length, res.pages_fetched, res.stopped_because, res.page_info]);
    check("one replay only; the request carried an abort signal; budget fields reported", stub.calls.length === 1 && !!stub.calls[0].init.signal && res.time_budget_ms === 5000 && res.elapsed_ms >= 2500, [stub.calls.length, res.time_budget_ms, res.elapsed_ms]);
    const plain = makeCtx({ pathname: "/loanfactoryhq/status/" + focal, captures: [cap], origFetch: stub });
    const all = await plain.window.__soloXRun("x.post.replies", { max_pages: 5 });
    check("no budget => walks until the replay stops answering (unchanged), time_budget_ms null", all.items.length === 2 && all.time_budget_ms === null && !stub.calls[stub.calls.length - 1].init.signal, [all.items.length, all.stopped_because, all.time_budget_ms]);
  }
  console.log("x.post.replies — a budget stop with no reply read is time_budget, never replies_hidden");
  {
    const focal = "1900000000000000007";
    const cap = gqlCapture("TweetDetail", { focalTweetId: focal }, { threaded_conversation_with_injections_v2: { instructions: instructions([tweetEntry(focal, { replies: 5 }), cursorEntry("MORE-1", "Bottom")]) } });
    const res = await makeCtx({ pathname: "/loanfactoryhq/status/" + focal, captures: [cap] }).window.__soloXRun("x.post.replies", { max_pages: 3, time_budget_ms: 2000 });
    check("0 replies, cursor present, budget short before page 2 => reason time_budget (not replies_hidden), reply_count kept", res.count === 0 && res.stopped_because === "time_budget" && res.reason === "time_budget" && res.reply_count === 5 && res.page_info.resumable === true, [res.count, res.stopped_because, res.reason, res.reply_count]);
  }
  console.log("ensureCapture — x.post.replies' two poll passes stop when the budget cannot cover another try");
  {
    const t0 = Date.now();
    const res = await makeCtx({ pathname: "/loanfactoryhq/status/1900000000000000008", captures: [] }).window.__soloXRun("x.post.replies", { ensure_tries: 100, time_budget_ms: 2200 });
    const took = Date.now() - t0;
    check("no capture + 2.2s budget: gave up in ~2s (" + took + "ms), not 12s/100s, honest reason", took < 3000 && res.found === false && res.reason === "detail_query_not_captured", [took, res.reason]);
  }
  console.log("x.post.replies — time budget: a hung replay is aborted at the budget line");
  {
    const focal = "1900000000000000005";
    const page1 = instructions([tweetEntry(focal, { replies: 9 }), tweetEntry("1900000000000000006", { replyTo: focal, handle: "c1", userId: "71" }), cursorEntry("HANG-2", "Bottom")]);
    const cap = gqlCapture("TweetDetail", { focalTweetId: focal }, { threaded_conversation_with_injections_v2: { instructions: page1 } });
    const hang = (url, init) => new Promise((resolve, reject) => { if (init && init.signal) init.signal.addEventListener("abort", () => { const e = new Error("aborted"); e.name = "AbortError"; reject(e); }); });
    const ctx = makeCtx({ pathname: "/loanfactoryhq/status/" + focal, captures: [cap], origFetch: hang });
    const t0 = Date.now();
    const res = await ctx.window.__soloXRun("x.post.replies", { max_pages: 3, time_budget_ms: 4000 });
    const took = Date.now() - t0;
    check("returned within the budget (" + took + "ms): page 1 reply kept, stopped_because time_budget, cursor kept", took < 4700 && res.items.length === 1 && res.stopped_because === "time_budget" && res.page_info.end_cursor === "HANG-2", [took, res.items.length, res.stopped_because, res.page_info]);
  }

  console.log("");
  console.log(fail === 0 ? "ALL " + pass + " CHECKS PASSED" : pass + " passed, " + fail + " FAILED");
  process.exit(fail ? 1 : 0);
})().catch((e) => { console.error(e); process.exit(2); });
