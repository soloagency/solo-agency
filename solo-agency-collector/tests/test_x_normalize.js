/*
 * Offline tests for chrome-extension/platforms/x/x_normalize.js: runs the REAL extractor
 * (x_extract.js) over capture fixtures, normalizes its output with __soloXNormalize, checks the
 * canonical shape, sweeps for sensitive key names, and validates every item with
 * chrome-extension/core/schema.js's SoloSchema.validateRecord — the pattern
 * tests/test_ig_normalize.js set. Run: node solo-agency-collector/tests/test_x_normalize.js
 */
"use strict";
const fs = require("fs");
const path = require("path");
const vm = require("vm");
const EXTRACT_SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "platforms", "x", "x_extract.js"), "utf8");
const NORMALIZE_SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "platforms", "x", "x_normalize.js"), "utf8");
const SCHEMA_PATH = path.join(__dirname, "..", "chrome-extension", "core", "schema.js");

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail).slice(0, 400) : "")); }
}
const SENSITIVE = ["cookie", "token", "secret", "password", "passwd", "pwd", "otp", "authorization", "auth", "session", "bearer", "csrf", "xsrf"];
function sensitiveKeys(o, p, out) {
  out = out || [];
  if (Array.isArray(o)) { o.forEach((v, i) => sensitiveKeys(v, p + "[" + i + "]", out)); return out; }
  if (!o || typeof o !== "object") return out;
  Object.keys(o).forEach((k) => { const lk = k.toLowerCase(); if (SENSITIVE.some((n) => lk.indexOf(n) !== -1)) out.push(p + "." + k); sensitiveKeys(o[k], p + "." + k, out); });
  return out;
}
function makeCtx(opts) {
  opts = opts || {};
  const pathname = opts.pathname || "/";
  const store = { captures: opts.captures || [], origFetch: () => Promise.resolve({ status: 404, text: () => Promise.resolve("") }), headersFor: () => ({ authorization: "Bearer x", "x-csrf-token": "c" }), parseResponse: (t) => { try { return JSON.parse(t); } catch (e) { return null; } }, __auth: { bearer: "Bearer x", extra: {} } };
  const ctx = { document: { cookie: "" }, location: { pathname, href: "https://x.com" + pathname, origin: "https://x.com" }, console, setTimeout, clearTimeout, URL, URLSearchParams, Promise, Date, JSON };
  ctx.window = ctx; ctx.self = ctx; ctx.window.__soloX = store;
  vm.createContext(ctx);
  vm.runInContext(EXTRACT_SRC, ctx, { filename: "x_extract.js" });
  vm.runInContext(NORMALIZE_SRC, ctx, { filename: "x_normalize.js" });
  return ctx;
}
function userResult(o) {
  o = o || {};
  return { __typename: "User", rest_id: o.id || "44196397", is_blue_verified: true,
    core: { screen_name: o.handle || "loanfactoryhq", name: o.name || "Loan Factory", created_at: "Wed Jan 15 10:00:00 +0000 2020" },
    avatar: { image_url: "https://pbs.twimg.com/profile_images/1/lf_normal.jpg" }, location: { location: "San Jose, CA" }, privacy: { protected: false },
    professional: { professional_type: "Business", category: [{ name: "Mortgage Broker" }] },
    legacy: { description: "hello@loanfactory.com (408) 555-0199", entities: { url: { urls: [{ expanded_url: "https://loanfactory.com" }] }, description: { urls: [] } }, followers_count: 18234, friends_count: 512, statuses_count: 3410, media_count: 1, listed_count: 0, url: "https://t.co/x", verified: false, pinned_tweet_ids_str: [] } };
}
function tweetResult(o) {
  o = o || {}; const id = o.id || "1700000000000000001";
  return { __typename: "Tweet", rest_id: id, core: { user_results: { result: userResult({ handle: o.handle || "loanfactoryhq" }) } }, views: { count: "99" },
    legacy: { id_str: id, full_text: o.text || "Rates dipped this week https://t.co/r1", created_at: "Tue Sep 09 12:00:00 +0000 2026", favorite_count: 5, reply_count: 2, retweet_count: 1, quote_count: 0, bookmark_count: 0, lang: "en", conversation_id_str: o.conversation || id, in_reply_to_status_id_str: o.replyTo || null, is_quote_status: false,
      entities: { urls: [{ expanded_url: "https://loanfactory.com/rates" }], hashtags: [], user_mentions: [] }, extended_entities: { media: [{ type: "video", media_url_https: "https://pbs.twimg.com/v.jpg", video_info: { variants: [{ content_type: "video/mp4", bitrate: 832000, url: "https://video.twimg.com/a.mp4" }] } }] } } };
}
function tweetEntry(id, o) { return { entryId: "tweet-" + id, content: { entryType: "TimelineTimelineItem", itemContent: { itemType: "TimelineTweet", tweet_results: { result: tweetResult(Object.assign({ id }, o || {})) } } } }; }
function cap(name, variables, data) { return { kind: "graphql", queryId: "q", queryName: name, method: "GET", variables, features: {}, url: "https://x.com/i/api/graphql/q/" + name, requestBody: "", capturedAt: Date.now(), response: { data } }; }
const CAPTURED_AT = "2026-09-10T12:00:00.000Z";
const ALL = [];

(async () => {
  console.log("-- x.profile.enrich -> profile --");
  {
    const ctx = makeCtx({ pathname: "/loanfactoryhq", captures: [cap("UserByScreenName", { screen_name: "loanfactoryhq" }, { user: { result: userResult() } })] });
    const raw = await ctx.window.__soloXRun("x.profile.enrich", {});
    const before = JSON.stringify(raw);
    const out = ctx.window.__soloXNormalize("x.profile.enrich", raw, { captured_at: CAPTURED_AT });
    check("input not mutated", JSON.stringify(raw) === before);
    check("schema_version 1, kind profile, 1 item", out.schema_version === 1 && out.kind === "profile" && out.items.length === 1, out);
    const p = out.items[0]; ALL.push(p);
    check("platform x, platform_id, url, name, handle, bio, website, emails, phones, follower_count, verified, location[]", p.platform === "x" && p.platform_id === "44196397" && p.url === "https://x.com/loanfactoryhq" && p.name === "Loan Factory" && p.handle === "loanfactoryhq" && p.emails[0] === "hello@loanfactory.com" && p.phones.length === 1 && p.follower_count === 18234 && p.verified === true && p.location[0] === "San Jose, CA" && p.website === "https://loanfactory.com", p);
    check("ext carries counters, professional category, joined_at ISO", p.ext.post_count === 3410 && p.ext.categories[0] === "Mortgage Broker" && /^2020-01-15/.test(p.ext.joined_at), p.ext);
    check("captured_at + source_capability", p.captured_at === CAPTURED_AT && p.source_capability === "x.profile.enrich");
  }
  console.log("-- x.search.posts -> post --");
  {
    const ctx = makeCtx({ pathname: "/search", captures: [cap("SearchTimeline", { rawQuery: "realtor", product: "Top" }, { search_by_raw_query: { search_timeline: { timeline: { instructions: [{ type: "TimelineAddEntries", entries: [tweetEntry("1800000000000000001")] }] } } } })] });
    const raw = await ctx.window.__soloXRun("x.search.posts", { mode: "top" });
    const out = ctx.window.__soloXNormalize("x.search.posts", raw, { captured_at: CAPTURED_AT });
    const p = out.items[0]; ALL.push(p);
    check("kind post, platform x, id/url/text/actor/created_at ISO/engagement", out.kind === "post" && p.platform === "x" && p.platform_id === "1800000000000000001" && /status\/1800000000000000001/.test(p.url) && p.actor.name === "Loan Factory" && /^2026-09-09T12:00:00/.test(p.created_at) && p.engagement.likes === 5 && p.engagement.comments === 2 && p.engagement.shares === 1 && p.engagement.views === 99, p);
    check("video attachment -> media{type video, url mp4}", p.media && p.media.type === "video" && /\.mp4$/.test(p.media.url) && p.attachments[0].type === "video", p.media);
    check("refs.post_id + ext.links/hashtags", p.refs.post_id === p.platform_id && p.ext.links[0] === "https://loanfactory.com/rates", p.refs);
  }
  console.log("-- x.post.replies -> comment --");
  {
    const focal = "1900000000000000001";
    const ins = [{ type: "TimelineAddEntries", entries: [tweetEntry(focal), { entryId: "conversationthread-1", content: { entryType: "TimelineTimelineModule", items: [{ entryId: "conversationthread-1-tweet-2", item: { itemContent: { itemType: "TimelineTweet", tweet_results: { result: tweetResult({ id: "1900000000000000002", handle: "commenter1", text: "Nice", replyTo: focal }) } } } }] } }] }];
    const ctx = makeCtx({ pathname: "/loanfactoryhq/status/" + focal, captures: [cap("TweetDetail", { focalTweetId: focal }, { threaded_conversation_with_injections_v2: { instructions: ins } })] });
    const raw = await ctx.window.__soloXRun("x.post.replies", {});
    const out = ctx.window.__soloXNormalize("x.post.replies", raw, { captured_at: CAPTURED_AT });
    const c = out.items[0]; ALL.push(c);
    check("kind comment, post_ref = focal id, text/actor/depth/reply_count", out.kind === "comment" && c.post_ref === focal && c.text === "Nice" && c.actor.name === "Loan Factory" && c.depth === 0 && c.reply_count === 2, c);
  }
  console.log("-- _discover.x -> null; unknown -> null --");
  {
    const ctx = makeCtx({});
    check("_discover.x has no canonical mapping", ctx.window.__soloXNormalize("_discover.x", { items: [] }) === null);
    check("unknown id -> null", ctx.window.__soloXNormalize("x.nope", { items: [] }) === null);
  }
  console.log("-- cross-cutting --");
  check("no sensitive key name in any normalized item", sensitiveKeys(ALL, "items").length === 0, sensitiveKeys(ALL, "items"));
  if (fs.existsSync(SCHEMA_PATH)) {
    const schemaCtx = { console }; vm.createContext(schemaCtx); vm.runInContext(fs.readFileSync(SCHEMA_PATH, "utf8"), schemaCtx);
    const SoloSchema = schemaCtx.SoloSchema;
    let allOk = true;
    ALL.forEach((item, i) => { const r = SoloSchema.validateRecord(item); if (!r.ok) { allOk = false; console.log("    item[" + i + "] errors: " + JSON.stringify(r.errors)); } });
    check("SoloSchema.validateRecord().ok for all " + ALL.length + " items", allOk);
  }
  console.log("\n" + (fail === 0 ? "ALL " + pass + " CHECKS PASSED" : pass + " passed, " + fail + " FAILED"));
  process.exit(fail ? 1 : 0);
})().catch((e) => { console.error(e); process.exit(2); });
