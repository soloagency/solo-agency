/*
 * x_extract.js — Solo Agency Local Collector, X (Twitter) module (read capabilities).
 *
 * Injected by background.js into the MAIN world of an x.com tab only for x.* jobs and dispatched
 * through window.__soloXRun (core/platform_registry.js). It reads the captures x_intercept.js
 * recorded while the page loaded and scrolled — X's own GraphQL timelines — and replays a
 * captured query only for deeper pagination, with the same session headers X itself sent.
 *
 * Surface (queryName = the operation in /i/api/graphql/<queryId>/<OperationName>; the data
 * paths below are the ones X's web client has used since 2023 and are read DEFENSIVELY —
 * every timeline is found by locating its `instructions` array, every tweet by unwrapping
 * TweetWithVisibilityResults, every user by reading legacy{} and the newer core{}/avatar{}/
 * location{}/verification{} fields side by side. Live shapes are confirmed with _discover.x
 * and recorded in X_CAPABILITIES.md as they are measured.
 *   profile page      UserByScreenName -> data.user.result {rest_id, legacy{...}, core{...},
 *                     is_blue_verified, professional{}, affiliates_highlighted_label{}}
 *   profile posts     UserTweets (Posts tab) / UserTweetsAndReplies (Replies tab) / UserMedia
 *                     -> data.user.result.timeline.timeline.instructions[]
 *   search            SearchTimeline (x.com/search?q=&f=top|live|user) -> data.search_by_raw_query
 *                     .search_timeline.timeline.instructions[]; variables.product Top|Latest|People
 *   post + replies    TweetDetail (x.com/<handle>/status/<id>) ->
 *                     data.threaded_conversation_with_injections_v2.instructions[]
 *   home feed         HomeTimeline (For you) / HomeLatestTimeline (Following) ->
 *                     data.home.home_timeline_urt.instructions[]
 * Every capability answers the same envelope the Facebook / Instagram / Zillow modules use:
 *   { capability, available, count, status?, items[], source_query?, page_info?, error?, version }
 */
(function () {
  "use strict";
  if (typeof window.__soloXRun === "function") return; // idempotent re-injection

  var VERSION = "0.1.0";
  var CAP_PROFILE = "x.profile.enrich";
  var CAP_POSTS = "x.profile.posts";
  var CAP_SEARCH_POSTS = "x.search.posts";
  var CAP_PEOPLE = "x.people.search";
  var CAP_REPLIES = "x.post.replies";
  var CAP_HOME = "x.timeline.home";
  var CAP_DISCOVER = "_discover.x";

  // ------------------------------------------------------------- helpers
  function isObj(v) { return !!v && typeof v === "object" && !Array.isArray(v); }
  function str(v) { return typeof v === "string" ? v : (v === null || v === undefined ? "" : String(v)); }
  function num(v) { if (typeof v === "number" && isFinite(v)) return v; if (typeof v === "string" && /^\d+$/.test(v)) return Number(v); return null; }
  function wait(ms) { return new Promise(function (r) { setTimeout(r, ms); }); }
  function currentHref() { try { return location.href; } catch (e) { return ""; } }
  function store() { return window.__soloX || { captures: [] }; }
  function captures() { var s = store(); return Array.isArray(s.captures) ? s.captures : []; }
  function findCapture(pred) {
    var caps = captures();
    for (var i = caps.length - 1; i >= 0; i--) { var c = caps[i]; if (!c || !c.response) continue; try { if (pred(c)) return c; } catch (e) { /* keep looking */ } }
    return null;
  }
  function allCaptures(pred) {
    var out = [], caps = captures();
    for (var i = 0; i < caps.length; i++) { var c = caps[i]; if (!c || !c.response) continue; try { if (pred(c)) out.push(c); } catch (e) { /* skip */ } }
    return out;
  }
  function ensureCapture(pred, tries, stepMs) {
    var n = 0;
    function loop() { var c = findCapture(pred); if (c) return Promise.resolve(c); if (n >= tries) return Promise.resolve(null); n += 1; return wait(stepMs).then(loop); }
    return loop();
  }
  function envelope(capId, items, extra) {
    var out = { capability: capId, available: true, count: items.length, items: items, version: VERSION };
    if (extra) for (var k in extra) out[k] = extra[k];
    return out;
  }
  function fail(capId, err) {
    var msg = String(err && err.message || err || "unknown error");
    return { capability: capId, available: true, count: 0, items: [{ capability: capId, status: "error", error: msg, url: currentHref() }], status: "error", error: msg, version: VERSION };
  }
  function responseData(cap) { var r = cap && cap.response; return isObj(r) && isObj(r.data) ? r.data : (isObj(r) ? r : null); }
  function handleFromHref() {
    try {
      var m = location.pathname.match(/^\/([A-Za-z0-9_]{1,15})(?:\/|$)/);
      if (!m) return "";
      if (/^(home|explore|search|notifications|messages|i|settings|compose|login|logout|hashtag|intent|share|tos|privacy)$/i.test(m[1])) return "";
      return m[1];
    } catch (e) { return ""; }
  }
  function statusIdFromHref() { try { var m = location.pathname.match(/\/status\/(\d+)/); return m ? m[1] : ""; } catch (e) { return ""; } }
  function profileUrl(handle) { return handle ? "https://x.com/" + handle : ""; }
  function postUrl(handle, id) { return id ? "https://x.com/" + (handle || "i/web") + "/status/" + id : ""; }
  var EMAIL_RE = /[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}/gi;
  var PHONE_RE = /(?:\+?\(?\d[\d\s().-]{7,}\d)/g;
  function emailsIn(text) { var out = [], m, t = String(text || ""); while ((m = EMAIL_RE.exec(t))) { var e = m[0].toLowerCase(); if (out.indexOf(e) === -1) out.push(e); } EMAIL_RE.lastIndex = 0; return out; }
  function phonesIn(text) {
    var out = [], m, t = String(text || "");
    while ((m = PHONE_RE.exec(t))) { var v = m[0].replace(/\s+/g, " ").trim(); var d = v.replace(/\D/g, ""); if (d.length < 8 || d.length > 15 || !/[\s().+-]/.test(v)) continue; if (out.indexOf(v) === -1) out.push(v); }
    PHONE_RE.lastIndex = 0; return out;
  }
  // The first `instructions` array anywhere under data — the timelines differ only in the
  // path that leads there (user.result.timeline.timeline, search_by_raw_query.search_timeline
  // .timeline, threaded_conversation_with_injections_v2, home.home_timeline_urt).
  function findInstructions(o, depth) {
    depth = depth || 0;
    if (!o || typeof o !== "object" || depth > 8) return null;
    if (Array.isArray(o.instructions)) return o.instructions;
    var keys = Object.keys(o);
    for (var i = 0; i < keys.length; i++) { var r = findInstructions(o[keys[i]], depth + 1); if (r) return r; }
    return null;
  }
  function unwrapTweet(r) {
    if (!isObj(r)) return null;
    if (r.__typename === "TweetWithVisibilityResults" && isObj(r.tweet)) return r.tweet;
    if (r.__typename === "TweetTombstone" || r.__typename === "TweetUnavailable") return null;
    return r;
  }
  function unwrapUser(r) {
    if (!isObj(r)) return null;
    if (r.__typename === "UserUnavailable") return null;
    return r;
  }
  // t.co links carry their real destination in expanded_url.
  function expandUrls(list) {
    var out = [];
    (Array.isArray(list) ? list : []).forEach(function (e) { var u = str(e && (e.expanded_url || e.url)); if (u && out.indexOf(u) === -1) out.push(u); });
    return out;
  }
  function parseTwitterDate(s) { var t = Date.parse(str(s)); return isFinite(t) ? Math.floor(t / 1000) : 0; }

  // ------------------------------------------------------------- user record
  function userRef(u) {
    u = unwrapUser(u);
    if (!u) return null;
    var legacy = isObj(u.legacy) ? u.legacy : {}, core = isObj(u.core) ? u.core : {};
    var handle = str(core.screen_name || legacy.screen_name);
    var verified = (u.is_blue_verified === true) || (isObj(u.verification) && u.verification.verified === true) || legacy.verified === true;
    return {
      id: str(u.rest_id || legacy.id_str),
      username: handle,
      name: str(core.name || legacy.name),
      url: profileUrl(handle),
      type: "profile",
      is_verified: verified,
      is_private: (isObj(u.privacy) && typeof u.privacy.protected === "boolean") ? u.privacy.protected : (typeof legacy.protected === "boolean" ? legacy.protected : null)
    };
  }
  // Measured 2026-09-10 (UserByScreenName on x.com): the profile fields no longer live in
  // legacy{} at all — bio in profile_bio{description, entities}, the site link in website{url}
  // (expanded in profile_bio.entities.url.urls[]), counters in relationship_counts{followers,
  // following}, tweet_counts{tweets, media_tweets}, action_counts{favorites_count}, identity in
  // core{}/avatar{}/location{}/verification{}/privacy{}. Older replies (and other timelines)
  // still carry legacy{}, so both are read, new fields first.
  function userRecord(u) {
    var ref = userRef(u);
    if (!ref) return null;
    u = unwrapUser(u);
    var legacy = isObj(u.legacy) ? u.legacy : {}, core = isObj(u.core) ? u.core : {};
    var pbio = isObj(u.profile_bio) ? u.profile_bio : {};
    var bio = str(pbio.description || legacy.description);
    var ents = isObj(pbio.entities) ? pbio.entities : (isObj(legacy.entities) ? legacy.entities : {});
    var siteUrls = expandUrls(isObj(ents.url) ? ents.url.urls : null);
    var website = siteUrls[0] || str((isObj(u.website) && u.website.url) || legacy.url);
    var bioUrls = expandUrls(isObj(ents.description) ? ents.description.urls : null);
    var rc = isObj(u.relationship_counts) ? u.relationship_counts : {}, tc = isObj(u.tweet_counts) ? u.tweet_counts : {}, ac = isObj(u.action_counts) ? u.action_counts : {};
    var photo = str((isObj(u.avatar) && u.avatar.image_url) || legacy.profile_image_url_https).replace(/_normal(\.[a-z]+)$/i, "_400x400$1");
    var prof = isObj(u.professional) ? u.professional : null;
    var cats = prof && Array.isArray(prof.category) ? prof.category.map(function (c) { return str(c && c.name); }).filter(Boolean) : [];
    var affil = isObj(u.affiliates_highlighted_label) && isObj(u.affiliates_highlighted_label.label) ? str(u.affiliates_highlighted_label.label.description) : "";
    var pick = function () { for (var i = 0; i < arguments.length; i++) { var n = num(arguments[i]); if (n !== null) return n; } return null; };
    return {
      id: ref.id, username: ref.username, name: ref.name, profile_url: ref.url,
      bio: bio,
      location: str((isObj(u.location) && u.location.location) || legacy.location),
      website: website,
      websites: (website ? [website] : []).concat(bioUrls.filter(function (x) { return x !== website; })),
      emails: emailsIn(bio), phones: phonesIn(bio),
      follower_count: pick(rc.followers, legacy.followers_count), following_count: pick(rc.following, legacy.friends_count),
      post_count: pick(tc.tweets, legacy.statuses_count), media_count: pick(tc.media_tweets, legacy.media_count), listed_count: num(legacy.listed_count),
      likes_given_count: pick(ac.favorites_count, legacy.favourites_count),
      created_at: str(core.created_at || legacy.created_at), joined_time: parseTwitterDate(core.created_at || legacy.created_at) || null,
      is_verified: ref.is_verified, verified_type: str(legacy.verified_type), is_private: ref.is_private,
      identity_verified: isObj(u.verification_info) && typeof u.verification_info.is_identity_verified === "boolean" ? u.verification_info.is_identity_verified : null,
      is_business: isObj(u.business_account) && Object.keys(u.business_account).length > 0 ? true : (prof ? str(prof.professional_type) === "Business" : null),
      professional_type: prof ? str(prof.professional_type) : "", category: cats[0] || "", categories: cats,
      affiliation: affil,
      profile_pic_url: photo,
      pinned_post_ids: Array.isArray(legacy.pinned_tweet_ids_str) ? legacy.pinned_tweet_ids_str.map(str) : [],
      source: "graphql"
    };
  }

  // ------------------------------------------------------------- tweet record
  function mediaOf(legacy) {
    var list = isObj(legacy.extended_entities) && Array.isArray(legacy.extended_entities.media) ? legacy.extended_entities.media : (isObj(legacy.entities) && Array.isArray(legacy.entities.media) ? legacy.entities.media : []);
    var out = [];
    for (var i = 0; i < list.length && i < 8; i++) {
      var m = list[i]; if (!isObj(m)) continue;
      var type = str(m.type) === "video" ? "video" : (str(m.type) === "animated_gif" ? "gif" : "photo");
      var url = str(m.media_url_https || m.media_url);
      if (type !== "photo" && isObj(m.video_info) && Array.isArray(m.video_info.variants)) {
        var best = null;
        for (var j = 0; j < m.video_info.variants.length; j++) { var v = m.video_info.variants[j]; if (isObj(v) && /mp4/.test(str(v.content_type)) && (!best || (num(v.bitrate) || 0) > (num(best.bitrate) || 0))) best = v; }
        if (best && best.url) url = str(best.url);
      }
      if (url) out.push({ type: type, url: url, preview_url: str(m.media_url_https || m.media_url) });
    }
    return out;
  }
  function tweetRecord(r) {
    var t = unwrapTweet(r);
    if (!t || !isObj(t.legacy)) return null;
    var legacy = t.legacy;
    var id = str(t.rest_id || legacy.id_str);
    if (!id) return null;
    var actor = userRef(isObj(t.core) && isObj(t.core.user_results) ? t.core.user_results.result : null);
    var note = isObj(t.note_tweet) && isObj(t.note_tweet.note_tweet_results) && isObj(t.note_tweet.note_tweet_results.result) ? str(t.note_tweet.note_tweet_results.result.text) : "";
    var rt = isObj(legacy.retweeted_status_result) ? tweetRecord(legacy.retweeted_status_result.result) : null;
    var quoted = isObj(t.quoted_status_result) ? tweetRecord(t.quoted_status_result.result) : null;
    var views = isObj(t.views) ? num(t.views.count) : null;
    var likes = num(legacy.favorite_count), replies = num(legacy.reply_count), reposts = num(legacy.retweet_count), quotes = num(legacy.quote_count), bookmarks = num(legacy.bookmark_count);
    return {
      id: id,
      url: postUrl(actor ? actor.username : "", id),
      actor: actor,
      text: (note || str(legacy.full_text)).slice(0, 8000),
      created_time: parseTwitterDate(legacy.created_at),
      lang: str(legacy.lang),
      engagement: { likes: likes || 0, comments: replies || 0, shares: (reposts || 0) + (quotes || 0), views: views, reposts: reposts || 0, quotes: quotes || 0, bookmarks: bookmarks },
      attachments: mediaOf(legacy),
      links: expandUrls(isObj(legacy.entities) ? legacy.entities.urls : null),
      hashtags: isObj(legacy.entities) && Array.isArray(legacy.entities.hashtags) ? legacy.entities.hashtags.map(function (h) { return str(h && h.text); }).filter(Boolean) : [],
      mentions: isObj(legacy.entities) && Array.isArray(legacy.entities.user_mentions) ? legacy.entities.user_mentions.map(function (m) { return str(m && m.screen_name); }).filter(Boolean) : [],
      conversation_id: str(legacy.conversation_id_str),
      in_reply_to_post_id: str(legacy.in_reply_to_status_id_str),
      in_reply_to_username: str(legacy.in_reply_to_screen_name),
      is_repost: !!rt, repost_of: rt,
      is_quote: legacy.is_quote_status === true, quote_of: quoted,
      // the handle x.post.replies takes — same idea as Facebook's feedback_id / Instagram's media_id
      post_id: id
    };
  }

  // ------------------------------------------------------------- timelines
  function entriesOf(instructions) {
    var out = [];
    (Array.isArray(instructions) ? instructions : []).forEach(function (ins) {
      if (!isObj(ins)) return;
      if (Array.isArray(ins.entries)) ins.entries.forEach(function (e) { out.push(e); });
      if (isObj(ins.entry)) out.push(ins.entry);
      if (Array.isArray(ins.moduleItems)) ins.moduleItems.forEach(function (mi) { if (isObj(mi) && isObj(mi.item)) out.push({ entryId: str(mi.entryId), content: { itemContent: mi.item.itemContent } }); });
    });
    return out;
  }
  function itemContentsOf(entry) {
    var out = [];
    var c = isObj(entry) ? entry.content : null;
    if (!isObj(c)) return out;
    if (isObj(c.itemContent)) out.push({ entryId: str(entry.entryId), itemContent: c.itemContent });
    if (Array.isArray(c.items)) c.items.forEach(function (it) { if (isObj(it) && isObj(it.item) && isObj(it.item.itemContent)) out.push({ entryId: str(it.entryId || entry.entryId), itemContent: it.item.itemContent }); });
    return out;
  }
  function bottomCursor(instructions) {
    var cur = "";
    entriesOf(instructions).forEach(function (e) {
      var c = isObj(e) ? e.content : null;
      if (!isObj(c)) return;
      if (str(c.entryType) === "TimelineTimelineCursor" && /bottom/i.test(str(c.cursorType)) && c.value) cur = str(c.value);
      if (isObj(c.itemContent) && /bottom/i.test(str(c.itemContent.cursorType)) && c.itemContent.value) cur = str(c.itemContent.value);
    });
    return cur;
  }
  function tweetsFrom(instructions) {
    var out = [], seen = {};
    entriesOf(instructions).forEach(function (e) {
      itemContentsOf(e).forEach(function (ic) {
        var res = isObj(ic.itemContent.tweet_results) ? ic.itemContent.tweet_results.result : null;
        var rec = tweetRecord(res);
        if (rec && !seen[rec.id]) { seen[rec.id] = 1; rec.entry_id = ic.entryId; out.push(rec); }
      });
    });
    return out;
  }
  function usersFrom(instructions) {
    var out = [], seen = {};
    entriesOf(instructions).forEach(function (e) {
      itemContentsOf(e).forEach(function (ic) {
        var res = isObj(ic.itemContent.user_results) ? ic.itemContent.user_results.result : null;
        var rec = userRecord(res);
        if (rec && rec.id && !seen[rec.id]) { seen[rec.id] = 1; out.push(rec); }
      });
    });
    return out;
  }
  function instructionsOf(cap) { return findInstructions(responseData(cap)); }

  // Replay a captured GET query with a new cursor: same url, `variables` rewritten, the same
  // session headers X's own client sent. X may refuse a replay without its per-request
  // transaction id; the caller then stops with replay_failed_<status> and keeps what it has.
  function replay(cap, patchVariables) {
    var s = store();
    var f = typeof s.origFetch === "function" ? s.origFetch : window.fetch;
    if (!cap || !cap.url) return Promise.resolve(null);
    var url;
    try { url = new URL(cap.url, location.origin); } catch (e) { return Promise.resolve(null); }
    var vars = isObj(cap.variables) ? JSON.parse(JSON.stringify(cap.variables)) : {};
    vars = patchVariables(vars) || vars;
    var h = typeof s.headersFor === "function" ? s.headersFor() : {};
    if (!h.authorization) return Promise.resolve({ status: 0, json: null, error: "no session bearer captured yet" });
    var init;
    if (String(cap.method || "GET").toUpperCase() === "POST" && cap.requestBody) {
      var body = {}; try { body = JSON.parse(cap.requestBody); } catch (e) { body = {}; }
      body.variables = vars;
      h["content-type"] = "application/json";
      init = { method: "POST", credentials: "include", headers: h, body: JSON.stringify(body) };
    } else {
      url.searchParams.set("variables", JSON.stringify(vars));
      init = { method: "GET", credentials: "include", headers: h };
    }
    return f(url.toString(), init).then(function (r) { return r.text().then(function (t) { return { status: r.status, json: s.parseResponse ? s.parseResponse(t) : null }; }); });
  }
  // Generic cursor pagination over a timeline: seed = what the page already captured (all
  // captures of this operation, merged), then replay from the newest bottom cursor.
  // Measured 2026-09-10: X answers a replayed GET with 404 unless it carries the page's own
  // per-request x-client-transaction-id. So the next page is asked for the way a human gets
  // it — scroll to the bottom and wait for the page's OWN next query to be captured — and the
  // replay is only the fallback when scrolling produces nothing.
  function scrollForMore(pred, knownCount, timeoutMs) {
    try {
      if (typeof window.scrollTo !== "function" || !document.documentElement) return Promise.resolve(false);
      window.scrollTo(0, document.documentElement.scrollHeight || document.body.scrollHeight || 100000);
    } catch (e) { return Promise.resolve(false); }
    var waited = 0;
    function poll() {
      if (allCaptures(pred).length > knownCount) return Promise.resolve(true);
      if (waited >= timeoutMs) return Promise.resolve(false);
      waited += 300;
      return wait(300).then(poll);
    }
    return poll();
  }
  function paginate(pred, maxPages, mapItems, wantFirst) {
    var items = [], seen = {}, pages = 0, cursor = "", last = null, stopped = null, merged = 0;
    function merge() {
      var caps = allCaptures(pred);
      for (var i = merged; i < caps.length; i++) {
        var ins = instructionsOf(caps[i]); if (!ins) continue;
        pages += 1; last = caps[i];
        mapItems(ins).forEach(function (it) { if (!seen[it.id]) { seen[it.id] = 1; items.push(it); } });
        var b = bottomCursor(ins); if (b) cursor = b;
      }
      merged = caps.length;
    }
    merge();
    function step() {
      if (pages >= maxPages) { if (cursor) stopped = "page_cap_hit"; return Promise.resolve(); }
      if (!cursor || !last) return Promise.resolve();
      var cur = cursor, before = items.length;
      return scrollForMore(pred, merged, 6000).then(function (more) {
        if (more) {
          merge();
          if (items.length === before) { stopped = "no_new_items"; return; }
          return wait(700).then(step);
        }
        return replay(last, function (v) { v.cursor = cur; return v; }).then(function (res) {
        var ins = res && res.json ? findInstructions(res.json) : null;
        if (!ins) { stopped = "replay_failed_" + (res ? (res.status || res.error || "no_response") : "no_response"); return; }
        var before = items.length;
        mapItems(ins).forEach(function (it) { if (!seen[it.id]) { seen[it.id] = 1; items.push(it); } });
        pages += 1;
        var next = bottomCursor(ins);
        if (items.length === before || !next || next === cur) { stopped = items.length === before ? "no_new_items" : null; cursor = ""; return; }
        cursor = next;
        return wait(900).then(step);
        });
      }).catch(function (e) { stopped = "fetch_error"; });
    }
    return step().then(function () {
      if (wantFirst) items = wantFirst(items);
      return { items: items, pages: pages, page_info: { end_cursor: cursor, has_next_page: !!cursor, resumable: !!cursor }, stopped_because: stopped };
    });
  }

  // ------------------------------------------------------------- x.profile.enrich
  function findUserResult(data) {
    var u = data && isObj(data.user) ? data.user.result : null;
    return unwrapUser(u);
  }
  function isProfileCapture(c) { return c.kind === "graphql" && /^UserBy(ScreenName|RestId)$/.test(str(c.queryName)) && !!findUserResult(responseData(c)); }
  function profileEnrich(inputs) {
    var wantHandle = str(inputs.username || inputs.handle).replace(/^@/, "") || handleFromHref();
    return ensureCapture(function (c) {
      if (!isProfileCapture(c)) return false;
      if (!wantHandle) return true;
      var u = userRef(findUserResult(responseData(c)));
      return !!u && u.username.toLowerCase() === wantHandle.toLowerCase();
    }, Number(inputs.ensure_tries) > 0 ? Number(inputs.ensure_tries) : 6, 1000).then(function (cap) {
      if (!cap) return envelope(CAP_PROFILE, [], { found: false, reason: "profile_query_not_captured", error: "no UserByScreenName query captured for " + (wantHandle || "this page") + " (suspended/protected account, login wall, or the query renamed)" });
      var rec = userRecord(findUserResult(responseData(cap)));
      if (!rec) return envelope(CAP_PROFILE, [], { found: false, reason: "profile_unavailable", source_query: cap.queryName });
      return envelope(CAP_PROFILE, [rec], { found: true, source: "graphql", source_query: cap.queryName });
    });
  }

  // ------------------------------------------------------------- x.profile.posts
  // Measured 2026-09-10: the Posts tab fires UserOriginalsTimeline (UserTweets is the older
  // name, kept); the Replies tab is matched by either of its known names.
  function isUserTimelineCapture(c, withReplies) { return c.kind === "graphql" && (withReplies ? /^User(TweetsAndReplies|RepliesTimeline)$/ : /^User(Tweets|OriginalsTimeline)$/).test(str(c.queryName)) && !!instructionsOf(c); }
  function profilePosts(inputs) {
    var withReplies = inputs.include_replies === true;
    var maxPages = Number(inputs.max_pages) > 0 ? Math.min(Number(inputs.max_pages), 40) : 1;
    var maxItems = Number(inputs.max_posts) > 0 ? Number(inputs.max_posts) : 0;
    var pred = function (c) { return isUserTimelineCapture(c, withReplies); };
    return ensureCapture(pred, Number(inputs.ensure_tries) > 0 ? Number(inputs.ensure_tries) : 6, 1000).then(function (cap) {
      if (!cap) return envelope(CAP_POSTS, [], { found: false, reason: "posts_query_not_captured", error: "no " + (withReplies ? "UserTweetsAndReplies" : "UserOriginalsTimeline/UserTweets") + " query captured on this page (protected account, no posts, or the tab did not render)" });
      return paginate(pred, maxPages, tweetsFrom).then(function (r) {
        var items = maxItems ? r.items.slice(0, maxItems) : r.items;
        return envelope(CAP_POSTS, items, { found: items.length > 0, source_query: cap.queryName, pages_fetched: r.pages, page_info: r.page_info, stopped_because: r.stopped_because, username: handleFromHref() });
      });
    });
  }

  // ------------------------------------------------------------- x.search.posts / x.people.search
  function searchProduct(c) { return str(isObj(c.variables) ? c.variables.product : ""); }
  function isSearchCapture(c, products) { return c.kind === "graphql" && /^SearchTimeline$/.test(str(c.queryName)) && products.indexOf(searchProduct(c)) !== -1 && !!instructionsOf(c); }
  function searchPosts(inputs) {
    var mode = str(inputs.mode).toLowerCase() === "latest" ? "Latest" : (str(inputs.mode).toLowerCase() === "top" ? "Top" : "");
    var products = mode ? [mode] : ["Top", "Latest"];
    var maxPages = Number(inputs.max_pages) > 0 ? Math.min(Number(inputs.max_pages), 20) : 1;
    var pred = function (c) { return isSearchCapture(c, products); };
    return ensureCapture(pred, Number(inputs.ensure_tries) > 0 ? Number(inputs.ensure_tries) : 8, 1000).then(function (cap) {
      if (!cap) return envelope(CAP_SEARCH_POSTS, [], { found: false, reason: "search_query_not_captured", error: "no SearchTimeline query captured; open https://x.com/search?q=<keyword>&src=typed_query" + (mode === "Latest" ? "&f=live" : "") });
      var product = searchProduct(cap);
      return paginate(function (c) { return isSearchCapture(c, [product]); }, maxPages, tweetsFrom).then(function (r) {
        return envelope(CAP_SEARCH_POSTS, r.items, { found: r.items.length > 0, source_query: cap.queryName, query: str(isObj(cap.variables) ? cap.variables.rawQuery : "") || str(inputs.query), mode: product.toLowerCase(), pages_fetched: r.pages, page_info: r.page_info, stopped_because: r.stopped_because });
      });
    });
  }
  function peopleSearch(inputs) {
    var maxPages = Number(inputs.max_pages) > 0 ? Math.min(Number(inputs.max_pages), 10) : 1;
    var pred = function (c) { return isSearchCapture(c, ["People"]); };
    return ensureCapture(pred, Number(inputs.ensure_tries) > 0 ? Number(inputs.ensure_tries) : 8, 1000).then(function (cap) {
      if (!cap) return envelope(CAP_PEOPLE, [], { found: false, reason: "search_query_not_captured", error: "no SearchTimeline(People) query captured; open https://x.com/search?q=<keyword>&src=typed_query&f=user" });
      return paginate(pred, maxPages, usersFrom).then(function (r) {
        return envelope(CAP_PEOPLE, r.items, { found: r.items.length > 0, source_query: cap.queryName, query: str(isObj(cap.variables) ? cap.variables.rawQuery : "") || str(inputs.query), pages_fetched: r.pages, page_info: r.page_info, stopped_because: r.stopped_because });
      });
    });
  }

  // ------------------------------------------------------------- x.post.replies
  function isDetailCapture(c) { return c.kind === "graphql" && /^TweetDetail$/.test(str(c.queryName)) && !!instructionsOf(c); }
  function postReplies(inputs) {
    var focalId = str(inputs.post_id) || statusIdFromHref();
    var maxPages = Number(inputs.max_pages) > 0 ? Math.min(Number(inputs.max_pages), 20) : 1;
    var maxReplies = Number(inputs.max_replies) > 0 ? Number(inputs.max_replies) : 100;
    var pred = function (c) { return isDetailCapture(c) && (!focalId || str(isObj(c.variables) ? c.variables.focalTweetId : "") === focalId); };
    return ensureCapture(pred, Number(inputs.ensure_tries) > 0 ? Number(inputs.ensure_tries) : 6, 1000).then(function (cap) {
      if (!cap) return envelope(CAP_REPLIES, [], { found: false, reason: "detail_query_not_captured", error: "no TweetDetail query captured for " + (focalId || currentHref()) });
      var id = focalId || str(isObj(cap.variables) ? cap.variables.focalTweetId : "");
      return paginate(pred, maxPages, tweetsFrom).then(function (r) {
        var post = null, replies = [];
        r.items.forEach(function (t) { if (t.id === id) post = t; else replies.push(t); });
        // depth 0 = a direct reply to the post, 1 = a reply inside a thread under it
        replies.forEach(function (t) { t.depth = t.in_reply_to_post_id === id ? 0 : 1; });
        replies = replies.slice(0, maxReplies);
        return envelope(CAP_REPLIES, replies, { found: replies.length > 0, source_query: cap.queryName, post_id: id, post: post,
          reply_count: post && post.engagement ? post.engagement.comments : null,
          reason: (!replies.length && post && post.engagement && post.engagement.comments > 0) ? "replies_hidden" : null,
          pages_fetched: r.pages, page_info: r.page_info, stopped_because: r.stopped_because });
      });
    });
  }

  // ------------------------------------------------------------- x.timeline.home
  function isHomeCapture(c) { return c.kind === "graphql" && /^Home(Latest)?Timeline$/.test(str(c.queryName)) && !!instructionsOf(c); }
  function timelineHome(inputs) {
    var maxPages = Number(inputs.max_pages) > 0 ? Math.min(Number(inputs.max_pages), 10) : 1;
    return ensureCapture(isHomeCapture, Number(inputs.ensure_tries) > 0 ? Number(inputs.ensure_tries) : 6, 1000).then(function (cap) {
      if (!cap) return envelope(CAP_HOME, [], { found: false, reason: "home_query_not_captured", error: "no HomeTimeline query captured; open https://x.com/home" });
      var name = cap.queryName;
      return paginate(function (c) { return isHomeCapture(c) && c.queryName === name; }, maxPages, tweetsFrom).then(function (r) {
        return envelope(CAP_HOME, r.items, { found: r.items.length > 0, source_query: name, feed: name === "HomeLatestTimeline" ? "following" : "for_you", pages_fetched: r.pages, page_info: r.page_info, stopped_because: r.stopped_because });
      });
    });
  }

  // ------------------------------------------------------------- _discover.x
  var SENSITIVE_KEY = /cookie|token|secret|password|passwd|pwd|otp|authorization|auth|session|bearer|csrf|xsrf/i;
  function skeletonize(o, depth, budget, maxDepth) {
    if (budget.n <= 0 || depth > maxDepth) return typeof o;
    if (Array.isArray(o)) { budget.n -= 1; return ["[" + o.length + "]", o.length ? skeletonize(o[0], depth + 1, budget, maxDepth) : null]; }
    if (isObj(o)) { var r = {}; var keys = Object.keys(o); for (var i = 0; i < keys.length && budget.n > 0; i++) { budget.n -= 1; r[keys[i]] = SENSITIVE_KEY.test(keys[i]) ? "<redacted>" : skeletonize(o[keys[i]], depth + 1, budget, maxDepth); } return r; }
    return typeof o === "string" ? "str:" + o.slice(0, 24) : o;
  }
  function discover(inputs) {
    var q = str(inputs.query);
    var caps = captures().map(function (c) {
      var d = responseData(c);
      var row = { kind: c.kind, url: str(c.url).replace(/\?.*$/, ""), queryName: c.queryName, queryId: c.queryId, method: c.method, size: c.response ? JSON.stringify(c.response).length : 0, dataKeys: isObj(d) ? Object.keys(d).slice(0, 8) : [], variables: isObj(c.variables) ? Object.keys(c.variables).slice(0, 12) : [], product: searchProduct(c) || undefined };
      if (q && (String(c.queryName).indexOf(q) > -1 || String(c.url).indexOf(q) > -1)) row.deep_skeleton = skeletonize(c.response, 0, { n: 3000 }, 22);
      return row;
    });
    return Promise.resolve(envelope(CAP_DISCOVER, caps, { queries: caps.map(function (c) { return c.queryName; }), replay_headers_seen: !!(store().__auth && store().__auth.bearer), href: currentHref() }));
  }

  // ------------------------------------------------------------- dispatch
  var CAPS = {};
  CAPS[CAP_PROFILE] = profileEnrich;
  CAPS[CAP_POSTS] = profilePosts;
  CAPS[CAP_SEARCH_POSTS] = searchPosts;
  CAPS[CAP_PEOPLE] = peopleSearch;
  CAPS[CAP_REPLIES] = postReplies;
  CAPS[CAP_HOME] = timelineHome;
  CAPS[CAP_DISCOVER] = discover;

  window.__soloXRun = function (capId, inputs) {
    capId = String(capId || ""); inputs = inputs && typeof inputs === "object" ? inputs : {};
    var fn = CAPS[capId];
    if (!fn) return Promise.resolve(fail(capId, "no x extractor for " + capId));
    try { return Promise.resolve(fn(inputs)).catch(function (e) { return fail(capId, e); }); }
    catch (e) { return Promise.resolve(fail(capId, e)); }
  };
  window.__soloXCapabilities = Object.keys(CAPS);
  window.__soloXVersion = VERSION;
  // Exposed for the offline harness (tests/test_x_extract.js); not used by background.js.
  window.__soloXInternals = { tweetRecord: tweetRecord, userRecord: userRecord, userRef: userRef, tweetsFrom: tweetsFrom, usersFrom: usersFrom, bottomCursor: bottomCursor, findInstructions: findInstructions, emailsIn: emailsIn, phonesIn: phonesIn, postUrl: postUrl };
})();
