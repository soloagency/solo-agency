/*
 * ig_extract.js — Solo Agency Local Collector, Instagram module (read capabilities).
 *
 * Injected by background.js into the MAIN world of an instagram.com tab only for ig.* jobs and
 * dispatched through window.__soloIgRun (core/platform_registry.js). It reads the captures
 * ig_intercept.js recorded during the page load — Instagram's own Polaris GraphQL and /api/v1/
 * REST traffic — and replays a captured query only for deeper pagination, carrying the same
 * session authority (csrf token, app id, the request body Instagram itself sent).
 *
 * Measured 2026-09-10 (logged-in web session, en locale):
 *   profile page      PolarisProfilePageContentQuery  -> data.user {username, full_name, pk, biography,
 *                     category, external_url, bio_links[], follower_count, following_count, media_count,
 *                     is_business, account_type, is_private, is_verified, address_street, city_name, zip}
 *                     — no public email / phone on the web query; they come from the bio text.
 *   profile posts     PolarisProfilePostsQuery -> data.xdt_api__v1__feed__user_timeline_graphql_connection
 *                     {edges[].node: XDTMediaDict, page_info{end_cursor, has_next_page}}, 12 per page.
 *   keyword search    PolarisKeywordSearchExplorePageRelayQuery (/explore/search/keyword/?q=) ->
 *                     data.xdt_fbsearch__top_serp_graphql.edges[].node (XDTTopSerpMediaGridUnit.items[]).
 *   people search     GET /api/v1/users/search/?q=&count= first (more rows when it answers), else
 *                     GET /api/v1/web/search/topsearch/?context=blended&query= -> users[].user (the
 *                     search box's own call, ~5 top matches; live 2026-09-10 only this one answered).
 *                     Both are called directly with the csrf + app-id headers.
 *   posts (tab)       the grid also fires PolarisProfilePostsTabContentQuery_connection — same
 *                     connection path, matched by path not by name; cursor replay gave page 2 live.
 *   post page         NO post query over the network: the media arrives EMBEDDED in the HTML
 *                     (data-sjs Relay entry PolarisPostRootQuery -> data.xdt_api__v1__media__
 *                     shortcode__web_info.items[0]: pk, code, caption, user, like_count,
 *                     comment_count, taken_at) with an empty PolarisPostCommentsContainerQuery.
 *   post comments     GET /api/v1/media/<pk>/comments/?can_support_threading=true -> comments[],
 *                     next_min_id, has_more_comments — fired by the page when the panel opens,
 *                     called directly here with the pk read from the embedded media.
 * Every capability answers the same envelope the Facebook and Zillow modules use:
 *   { capability, available, count, status?, items[], source_query?, page_info?, error?, version }
 */
(function () {
  "use strict";
  if (typeof window.__soloIgRun === "function") return; // idempotent re-injection

  var VERSION = "0.1.0";
  var CAP_PROFILE = "ig.profile.enrich";
  var CAP_POSTS = "ig.profile.posts";
  var CAP_SEARCH_POSTS = "ig.search.posts";
  var CAP_PEOPLE = "ig.people.search";
  var CAP_COMMENTS = "ig.post.comments";
  var CAP_DISCOVER = "_discover.ig";

  // ------------------------------------------------------------- helpers
  function isObj(v) { return !!v && typeof v === "object" && !Array.isArray(v); }
  function getPath(o, path) {
    var cur = o;
    var parts = String(path).split(".");
    for (var i = 0; i < parts.length; i++) {
      if (cur === null || cur === undefined) return undefined;
      cur = cur[parts[i]];
    }
    return cur;
  }
  function wait(ms) { return new Promise(function (r) { setTimeout(r, ms); }); }
  function currentHref() { try { return location.href; } catch (e) { return ""; } }
  function str(v) { return typeof v === "string" ? v : (v === null || v === undefined ? "" : String(v)); }
  function num(v) { return typeof v === "number" && isFinite(v) ? v : null; }
  function store() { return window.__soloIg || { captures: [] }; }
  // Data Instagram embeds in the page itself: every data-sjs script carries Relay entries
  // ["adp_<QueryName>RelayPreloader_<hash>", {__bbox: {result: {data}}}]. Measured 2026-09-10:
  // a post page ships PolarisPostRootQuery (the media) and PolarisPostCommentsContainerQuery
  // (an EMPTY comments connection — comments load lazily) this way and fires NO post query
  // over the network, while a profile page fires PolarisProfilePageContentQuery by XHR. So
  // both sources are read: live captures first, then these prefetched ones, as pseudo-captures
  // with kind "prefetch". Parsed once per page (48 scripts, ~1 MB) and cached on window.
  function prefetched() {
    try {
      var scripts = document.querySelectorAll('script[type="application/json"][data-sjs]');
      var cache = window.__soloIgPrefetch;
      if (cache && cache.count === scripts.length) return cache.entries;
      var entries = [];
      var walk = function (o, depth) {
        if (depth > 24 || !o || typeof o !== "object") return;
        if (Array.isArray(o)) {
          if (typeof o[0] === "string" && /^adp_/.test(o[0]) && isObj(o[1]) && isObj(o[1].__bbox) && isObj(o[1].__bbox.result) && isObj(o[1].__bbox.result.data)) {
            entries.push({ kind: "prefetch", queryName: o[0].replace(/^adp_/, "").replace(/RelayPreloader.*$/, ""), docId: "", variables: {}, url: currentHref(), requestBody: "", capturedAt: Date.now(), response: { data: o[1].__bbox.result.data } });
          }
          for (var i = 0; i < o.length; i++) walk(o[i], depth + 1);
          return;
        }
        for (var k in o) walk(o[k], depth + 1);
      };
      for (var i = 0; i < scripts.length; i++) {
        var t = scripts[i].textContent || "";
        if (t.indexOf("adp_") === -1) continue;
        var j = null; try { j = JSON.parse(t); } catch (e) { continue; }
        walk(j, 0);
      }
      window.__soloIgPrefetch = { count: scripts.length, entries: entries };
      return entries;
    } catch (e) { return []; }
  }
  function captures() {
    var s = store();
    var live = Array.isArray(s.captures) ? s.captures : [];
    return prefetched().concat(live);   // live captures come last, so newest-first walks see them first
  }
  // Newest capture first — a profile page navigated within the SPA has several, and the latest
  // is the one for the profile on screen; prefetched page data sits behind the live captures.
  function findCapture(pred) {
    var caps = captures();
    for (var i = caps.length - 1; i >= 0; i--) {
      var c = caps[i];
      if (!c || !c.response) continue;
      try { if (pred(c)) return c; } catch (e) { /* keep looking */ }
    }
    return null;
  }
  function responseData(cap) {
    var r = cap && cap.response;
    if (Array.isArray(r)) r = r[0];
    return isObj(r) && isObj(r.data) ? r.data : (isObj(r) ? r : null);
  }
  function usernameFromHref() {
    try {
      var m = location.pathname.match(/^\/([A-Za-z0-9._]+)\/?/);
      if (!m) return "";
      var seg = m[1];
      if (/^(explore|p|reel|reels|stories|accounts|direct|api|graphql)$/i.test(seg)) return "";
      return seg;
    } catch (e) { return ""; }
  }
  function shortcodeFromHref() {
    try { var m = location.pathname.match(/\/(?:p|reel|reels)\/([A-Za-z0-9_-]+)/); return m ? m[1] : ""; } catch (e) { return ""; }
  }
  function profileUrl(username) { return username ? "https://www.instagram.com/" + username + "/" : ""; }
  function postUrl(code, productType) {
    if (!code) return "";
    return "https://www.instagram.com/" + (String(productType || "").indexOf("clips") === 0 ? "reel" : "p") + "/" + code + "/";
  }
  var EMAIL_RE = /[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}/gi;
  var PHONE_RE = /(?:\+?\d[\d\s().-]{7,}\d)/g;
  function emailsIn(text) {
    var out = [], m, t = String(text || "");
    while ((m = EMAIL_RE.exec(t))) { var e = m[0].toLowerCase(); if (out.indexOf(e) === -1) out.push(e); }
    EMAIL_RE.lastIndex = 0;
    return out;
  }
  function phonesIn(text) {
    var out = [], m, t = String(text || "");
    while ((m = PHONE_RE.exec(t))) {
      var v = m[0].replace(/\s+/g, " ").trim();
      var digits = v.replace(/\D/g, "");
      if (digits.length < 8 || digits.length > 15) continue;
      if (!/[\s().+-]/.test(v)) continue;
      if (out.indexOf(v) === -1) out.push(v);
    }
    PHONE_RE.lastIndex = 0;
    return out;
  }
  function userRef(u) {
    if (!isObj(u)) return null;
    var username = str(u.username);
    return {
      id: str(u.pk || u.id),
      username: username,
      name: str(u.full_name),
      url: profileUrl(username),
      type: "profile",
      is_verified: typeof u.is_verified === "boolean" ? u.is_verified : null,
      is_private: typeof u.is_private === "boolean" ? u.is_private : null
    };
  }
  function firstImage(node) {
    var cands = getPath(node, "image_versions2.candidates");
    return Array.isArray(cands) && cands[0] && typeof cands[0].url === "string" ? cands[0].url : "";
  }
  // XDTMediaDict -> PostRecord (same field names as the Facebook PostRecord where the concept
  // is the same: id, url, text, actor, created_time, engagement, attachments).
  function postRecord(node) {
    if (!isObj(node) || (!node.pk && !node.id)) return null;
    var code = str(node.code);
    var owner = userRef(node.user || node.owner);
    var likes = num(node.like_count), comments = num(node.comment_count), views = num(node.view_count != null ? node.view_count : node.play_count);
    var carousel = Array.isArray(node.carousel_media) ? node.carousel_media : [];
    var attachments = [];
    var img = firstImage(node);
    if (img) attachments.push({ type: node.media_type === 2 ? "video" : "photo", url: img });
    for (var i = 0; i < carousel.length && i < 10; i++) {
      var cimg = firstImage(carousel[i]);
      if (cimg) attachments.push({ type: carousel[i].media_type === 2 ? "video" : "photo", url: cimg });
    }
    var loc = isObj(node.location) ? { id: str(node.location.pk), name: str(node.location.name), lat: num(node.location.lat), lng: num(node.location.lng) } : null;
    return {
      id: str(node.pk || String(node.id).split("_")[0]),
      code: code,
      url: postUrl(code, node.product_type),
      actor: owner,
      text: str(getPath(node, "caption.text")).slice(0, 4000),
      created_time: num(node.taken_at) || 0,
      engagement: (likes === null && comments === null && views === null) ? null : { likes: likes || 0, comments: comments || 0, shares: 0, views: views },
      attachments: attachments,
      media_type: num(node.media_type),
      product_type: str(node.product_type),
      carousel_media_count: num(node.carousel_media_count),
      location: loc,
      // the handle ig.post.comments takes (media pk) — same idea as Facebook's feedback_id
      media_id: str(node.pk || String(node.id).split("_")[0])
    };
  }
  function commentRecord(c, depth) {
    if (!isObj(c) || !c.pk) return null;
    return {
      id: str(c.pk),
      text: str(c.text),
      created_time: num(c.created_at_utc != null ? c.created_at_utc : c.created_at) || 0,
      actor: userRef(c.user),
      reply_count: num(c.child_comment_count) || 0,
      like_count: num(c.comment_like_count) || 0,
      depth: depth || 0,
      replies: []
    };
  }
  function headers(friendlyName) {
    var s = store();
    var h = {
      "x-csrftoken": typeof s.csrfToken === "function" ? s.csrfToken() : "",
      "x-ig-app-id": typeof s.appId === "function" ? s.appId() : "936619743392459",
      "x-requested-with": "XMLHttpRequest",
      "x-asbd-id": "129477"
    };
    if (friendlyName) h["x-fb-friendly-name"] = friendlyName;
    return h;
  }
  function restGet(path) {
    var s = store();
    var f = typeof s.origFetch === "function" ? s.origFetch : window.fetch;
    return f(path, { method: "GET", credentials: "include", headers: headers("") }).then(function (r) {
      return r.text().then(function (t) {
        var j = null; try { j = JSON.parse(t.replace(/^for\s*\(;;\);/, "")); } catch (e) { j = null; }
        return { status: r.status, json: j };
      });
    });
  }
  // Replay a captured Polaris query with a new `after` cursor: the same body Instagram sent,
  // with only `variables` rewritten — the same trick the Facebook module uses.
  function replay(cap, patchVariables) {
    var s = store();
    var f = typeof s.origFetch === "function" ? s.origFetch : window.fetch;
    if (!cap || !cap.requestBody) return Promise.resolve(null);
    var p = new URLSearchParams(cap.requestBody);
    var vars = {};
    try { vars = JSON.parse(p.get("variables") || "{}"); } catch (e) { vars = {}; }
    vars = patchVariables(vars) || vars;
    p.set("variables", JSON.stringify(vars));
    var h = headers(cap.queryName);
    h["content-type"] = "application/x-www-form-urlencoded";
    return f(cap.url, { method: "POST", credentials: "include", headers: h, body: p.toString() }).then(function (r) {
      return r.text().then(function (t) { return { status: r.status, json: s.parseResponse ? s.parseResponse(t) : null }; });
    });
  }
  function ensureCapture(pred, tries, stepMs) {
    var n = 0;
    function loop() {
      var c = findCapture(pred);
      if (c) return Promise.resolve(c);
      if (n >= tries) return Promise.resolve(null);
      n += 1;
      return wait(stepMs).then(loop);
    }
    return loop();
  }
  function envelope(capId, items, extra) {
    var out = { capability: capId, available: true, count: items.length, items: items, version: VERSION };
    if (extra) for (var k in extra) out[k] = extra[k];
    return out;
  }
  function fail(capId, e) {
    var msg = String(e && e.message || e);
    return { capability: capId, available: true, count: 0, status: "error", error: msg, version: VERSION,
      items: [{ capability: capId, status: "error", error: msg, url: currentHref() }] };
  }

  // ------------------------------------------------------------- ig.profile.enrich
  function isProfileCapture(c) {
    return (c.kind === "graphql" || c.kind === "prefetch") && /PolarisProfilePageContentQuery/.test(c.queryName || "") && isObj(getPath(responseData(c), "user"));
  }
  function profileFromDom() {
    // Fallback when the profile query was not captured: the header's own text. Kept minimal —
    // name, handle, the three counters and the bio — so a record exists rather than nothing.
    var header = document.querySelector("header");
    var text = header ? String(header.innerText || "") : "";
    var lines = text.split("\n").map(function (l) { return l.trim(); }).filter(Boolean);
    var username = usernameFromHref();
    var counts = {};
    lines.forEach(function (l) {
      var m = l.match(/^([\d.,]+\s*[KMB]?)\s*(posts|followers|following)$/i);
      if (m) counts[m[2].toLowerCase()] = parseCount(m[1]);
    });
    return {
      username: username, id: "", name: lines[1] && lines[1] !== username ? lines[1] : "", profile_url: profileUrl(username),
      bio: "", category: "", external_url: "", bio_links: [], website: "",
      follower_count: counts.followers != null ? counts.followers : null, following_count: counts.following != null ? counts.following : null,
      media_count: counts.posts != null ? counts.posts : null,
      is_business: null, account_type: null, is_private: null, is_verified: null,
      address: { street: "", city: "", zip: "" }, emails: emailsIn(text), phones: phonesIn(text), header_lines: lines.slice(0, 20),
      source: "dom"
    };
  }
  function parseCount(s) {
    if (!s) return null;
    var m = String(s).replace(/,/g, "").match(/([\d.]+)\s*([KMB])?/i);
    if (!m) return null;
    var n = parseFloat(m[1]); var u = (m[2] || "").toUpperCase();
    if (u === "K") n *= 1e3; else if (u === "M") n *= 1e6; else if (u === "B") n *= 1e9;
    return Math.round(n);
  }
  function profileEnrich(inputs) {
    return ensureCapture(isProfileCapture, Number(inputs.ensure_tries) > 0 ? Number(inputs.ensure_tries) : 6, 1000).then(function (cap) {
      var username = usernameFromHref();
      if (!cap) {
        var dom = profileFromDom();
        var found = !!(dom.name || dom.username);
        return envelope(CAP_PROFILE, found ? [dom] : [], { found: found, source: "dom", reason: "profile_query_not_captured",
          error: found ? null : "profile query not captured and no header rendered" });
      }
      var data = responseData(cap);
      var u = data.user;
      var viewerId = str(getPath(data, "viewer.user.id") || getPath(data, "viewer.user.pk"));
      if (viewerId && str(u.id || u.pk) === viewerId) {
        return { capability: CAP_PROFILE, available: false, reason: "self_profile", count: 0, items: [], version: VERSION,
          error: "resolved to the logged-in operator's own profile" };
      }
      var links = Array.isArray(u.bio_links) ? u.bio_links.map(function (l) { return { url: str(l.url), title: str(l.title), type: str(l.link_type) }; }).filter(function (l) { return l.url; }) : [];
      var bio = str(u.biography);
      var website = str(u.external_url) || (links[0] ? links[0].url : "");
      var item = {
        username: str(u.username) || username,
        id: str(u.pk || u.id),
        name: str(u.full_name),
        profile_url: profileUrl(str(u.username) || username),
        bio: bio,
        category: str(u.category),
        external_url: str(u.external_url),
        bio_links: links,
        website: website,
        websites: links.map(function (l) { return l.url; }),
        follower_count: num(u.follower_count),
        following_count: num(u.following_count),
        media_count: num(u.media_count),
        is_business: typeof u.is_business === "boolean" ? u.is_business : null,
        is_professional: typeof u.is_professional_account === "boolean" ? u.is_professional_account : null,
        account_type: num(u.account_type),
        is_private: typeof u.is_private === "boolean" ? u.is_private : null,
        is_verified: typeof u.is_verified === "boolean" ? u.is_verified : null,
        address: { street: str(u.address_street), city: str(u.city_name), zip: str(u.zip) },
        emails: emailsIn(bio + " " + links.map(function (l) { return l.url; }).join(" ")),
        phones: phonesIn(bio),
        profile_pic_url: str(getPath(u, "hd_profile_pic_url_info.url") || u.profile_pic_url),
        fbid: str(u.fbid_v2),
        mutual_followers_count: num(u.mutual_followers_count),
        source: "graphql"
      };
      return envelope(CAP_PROFILE, [item], { found: true, source: "graphql", source_query: cap.queryName });
    });
  }

  // ------------------------------------------------------------- ig.profile.posts
  var POSTS_CONN = "xdt_api__v1__feed__user_timeline_graphql_connection";
  function isPostsCapture(c) { return (c.kind === "graphql" || c.kind === "prefetch") && isObj(getPath(responseData(c), POSTS_CONN)); }
  function connectionItems(conn) {
    var edges = isObj(conn) && Array.isArray(conn.edges) ? conn.edges : [];
    var out = [];
    for (var i = 0; i < edges.length; i++) { var rec = postRecord(edges[i] && edges[i].node); if (rec) out.push(rec); }
    return out;
  }
  function paginate(capId, cap, connPath, maxPages, seedItems, seedPageInfo) {
    var items = seedItems.slice(), seen = {};
    items.forEach(function (it) { seen[it.id] = 1; });
    var pageInfo = seedPageInfo || {};
    var pages = 1, stopped = null;
    function step() {
      if (pages >= maxPages) { stopped = pages >= maxPages && pageInfo.has_next_page ? "page_cap_hit" : null; return Promise.resolve(); }
      if (!pageInfo.has_next_page || !pageInfo.end_cursor) return Promise.resolve();
      var cursor = pageInfo.end_cursor;
      return replay(cap, function (vars) {
        vars.after = cursor;
        if (isObj(vars.data) && vars.data.count == null) vars.data.count = 12;
        return vars;
      }).then(function (res) {
        var conn = res && res.json ? getPath(Array.isArray(res.json) ? res.json[0] : res.json, "data." + connPath) : null;
        if (!isObj(conn)) { stopped = "replay_failed_" + (res ? res.status : "no_response"); return; }
        var fresh = connectionItems(conn).filter(function (it) { if (seen[it.id]) return false; seen[it.id] = 1; return true; });
        items = items.concat(fresh);
        pageInfo = isObj(conn.page_info) ? conn.page_info : {};
        pages += 1;
        if (!fresh.length) { stopped = "no_new_items"; return; }
        return wait(800).then(step);
      }).catch(function (e) { stopped = "fetch_error"; });
    }
    return step().then(function () {
      return { items: items, pages: pages, page_info: { end_cursor: str(pageInfo.end_cursor), has_next_page: !!pageInfo.has_next_page, resumable: !!(pageInfo.has_next_page && pageInfo.end_cursor) }, stopped_because: stopped };
    });
  }
  function profilePosts(inputs) {
    var maxPages = Number(inputs.max_pages) > 0 ? Math.min(Number(inputs.max_pages), 40) : 1;
    return ensureCapture(isPostsCapture, Number(inputs.ensure_tries) > 0 ? Number(inputs.ensure_tries) : 6, 1000).then(function (cap) {
      if (!cap) return envelope(CAP_POSTS, [], { found: false, reason: "posts_query_not_captured", error: "no timeline query captured on this page (private profile, no posts, or the page did not render the grid)" });
      var conn = getPath(responseData(cap), POSTS_CONN);
      var seed = connectionItems(conn);
      var maxItems = Number(inputs.max_posts) > 0 ? Number(inputs.max_posts) : 0;
      return paginate(CAP_POSTS, cap, POSTS_CONN, maxPages, seed, conn.page_info).then(function (r) {
        var items = maxItems ? r.items.slice(0, maxItems) : r.items;
        return envelope(CAP_POSTS, items, { found: items.length > 0, source_query: cap.queryName, pages_fetched: r.pages, page_info: r.page_info, stopped_because: r.stopped_because, username: str(getPath(cap, "variables.username")) || usernameFromHref() });
      });
    });
  }

  // ------------------------------------------------------------- ig.search.posts
  var SERP = "xdt_fbsearch__top_serp_graphql";
  function isSerpCapture(c) { return (c.kind === "graphql" || c.kind === "prefetch") && isObj(getPath(responseData(c), SERP)); }
  function serpItems(conn) {
    var edges = isObj(conn) && Array.isArray(conn.edges) ? conn.edges : [];
    var out = [], seen = {};
    for (var i = 0; i < edges.length; i++) {
      var node = edges[i] && edges[i].node;
      if (!isObj(node)) continue;
      var medias = Array.isArray(node.items) ? node.items : (isObj(node.media) ? [node.media] : []);
      for (var j = 0; j < medias.length; j++) {
        var rec = postRecord(medias[j]);
        if (rec && !seen[rec.id]) { seen[rec.id] = 1; out.push(rec); }
      }
    }
    return out;
  }
  function searchPosts(inputs) {
    var maxPages = Number(inputs.max_pages) > 0 ? Math.min(Number(inputs.max_pages), 20) : 1;
    return ensureCapture(isSerpCapture, Number(inputs.ensure_tries) > 0 ? Number(inputs.ensure_tries) : 8, 1000).then(function (cap) {
      if (!cap) return envelope(CAP_SEARCH_POSTS, [], { found: false, reason: "search_query_not_captured", error: "no keyword-search query captured; open https://www.instagram.com/explore/search/keyword/?q=<keyword>" });
      var conn = getPath(responseData(cap), SERP);
      var seed = serpItems(conn);
      var items = seed.slice(), seen = {}; items.forEach(function (it) { seen[it.id] = 1; });
      var pageInfo = isObj(conn.page_info) ? conn.page_info : {};
      var pages = 1, stopped = null;
      function step() {
        if (pages >= maxPages || !pageInfo.has_next_page || !pageInfo.end_cursor) { if (pages >= maxPages && pageInfo.has_next_page) stopped = "page_cap_hit"; return Promise.resolve(); }
        var cursor = pageInfo.end_cursor;
        return replay(cap, function (vars) { vars.after = cursor; return vars; }).then(function (res) {
          var c2 = res && res.json ? getPath(Array.isArray(res.json) ? res.json[0] : res.json, "data." + SERP) : null;
          if (!isObj(c2)) { stopped = "replay_failed_" + (res ? res.status : "no_response"); return; }
          var fresh = serpItems(c2).filter(function (it) { if (seen[it.id]) return false; seen[it.id] = 1; return true; });
          items = items.concat(fresh); pageInfo = isObj(c2.page_info) ? c2.page_info : {}; pages += 1;
          if (!fresh.length) { stopped = "no_new_items"; return; }
          return wait(800).then(step);
        }).catch(function (e) { stopped = "fetch_error"; });
      }
      return step().then(function () {
        return envelope(CAP_SEARCH_POSTS, items, { found: items.length > 0, source_query: cap.queryName, query: str(getPath(cap, "variables.query")) || str(inputs.query), pages_fetched: pages,
          page_info: { end_cursor: str(pageInfo.end_cursor), has_next_page: !!pageInfo.has_next_page, resumable: !!(pageInfo.has_next_page && pageInfo.end_cursor) }, stopped_because: stopped });
      });
    });
  }

  // ------------------------------------------------------------- ig.people.search
  function peopleSearch(inputs) {
    var q = str(inputs.query).trim();
    if (!q) return Promise.resolve(fail(CAP_PEOPLE, "inputs.query is required"));
    var count = Number(inputs.count) > 0 ? Math.min(Number(inputs.count), 50) : 30;
    function fromUsers(list, via) {
      var items = [], seen = {};
      (Array.isArray(list) ? list : []).forEach(function (row) {
        var u = isObj(row) && isObj(row.user) ? row.user : row;
        var ref = userRef(u);
        if (!ref || !ref.username || seen[ref.username] || items.length >= count) return;
        seen[ref.username] = 1;
        ref.profile_pic_url = str(u.profile_pic_url);
        ref.subtitle = str(row.search_social_context || u.search_social_context || u.social_context);
        items.push(ref);
      });
      return envelope(CAP_PEOPLE, items, { found: items.length > 0, query: q, source_query: via });
    }
    // users/search returns more rows than the search box's blended top search; try it first.
    return restGet("/api/v1/users/search/?q=" + encodeURIComponent(q) + "&count=" + count).then(function (r) {
      var users = r.json && Array.isArray(r.json.users) ? r.json.users : null;
      if (r.status === 200 && users && users.length) return fromUsers(users, "users/search");
      return restGet("/api/v1/web/search/topsearch/?context=blended&query=" + encodeURIComponent(q) + "&include_reel=false&search_surface=web_top_search").then(function (r2) {
        if (r2.status !== 200 || !r2.json) return fail(CAP_PEOPLE, "topsearch answered HTTP " + r2.status);
        return fromUsers(r2.json.users, "web/search/topsearch");
      });
    });
  }

  // ------------------------------------------------------------- ig.post.comments
  var COMMENTS_URL = /\/api\/v1\/media\/(\d+)\/comments\//;
  function isCommentsCapture(c) { return c.kind === "rest" && COMMENTS_URL.test(c.url) && isObj(c.response) && Array.isArray(c.response.comments); }
  function mediaIdFromCaptures(shortcode) {
    // The post root query carries the media it opened; any XDTMediaDict whose code matches wins.
    var found = "";
    var walk = function (o, depth) {
      if (found || depth > 10 || !o || typeof o !== "object") return;
      if (Array.isArray(o)) { for (var i = 0; i < o.length && !found; i++) walk(o[i], depth + 1); return; }
      if (o.code === shortcode && (o.pk || o.id)) { found = str(o.pk || String(o.id).split("_")[0]); return; }
      for (var k in o) { if (found) return; walk(o[k], depth + 1); }
    };
    var caps = captures();
    for (var i = caps.length - 1; i >= 0 && !found; i--) { if (caps[i] && caps[i].response) walk(caps[i].response, 0); }
    return found;
  }
  function postHeaderFromCaptures(shortcode) {
    var found = null;
    var walk = function (o, depth) {
      if (found || depth > 10 || !o || typeof o !== "object") return;
      if (Array.isArray(o)) { for (var i = 0; i < o.length && !found; i++) walk(o[i], depth + 1); return; }
      if (o.code === shortcode && (o.pk || o.id)) { found = postRecord(o); return; }
      for (var k in o) { if (found) return; walk(o[k], depth + 1); }
    };
    var caps = captures();
    for (var i = caps.length - 1; i >= 0 && !found; i--) { if (caps[i] && caps[i].response) walk(caps[i].response, 0); }
    return found;
  }
  function postComments(inputs) {
    var shortcode = str(inputs.shortcode) || shortcodeFromHref();
    var mediaId = str(inputs.media_id);
    var maxPages = Number(inputs.max_comment_pages) > 0 ? Math.min(Number(inputs.max_comment_pages), 20) : 1;
    var maxComments = Number(inputs.max_comments) > 0 ? Number(inputs.max_comments) : 50;
    // The media id comes from inputs or from the post this page opened (post-root capture or the
    // embedded Relay entry whose code matches the shortcode) — never from whichever comments
    // capture happens to be newest in the ring, which may belong to a post viewed earlier.
    function resolveMediaId() { return mediaId || (shortcode ? mediaIdFromCaptures(shortcode) : ""); }
    return ensureCapture(function (c) { var id = resolveMediaId(); return !!id && isCommentsCapture(c) && c.url.indexOf("/media/" + id + "/") > -1; }, Number(inputs.ensure_tries) > 0 ? Number(inputs.ensure_tries) : 4, 1000).then(function (cap) {
      if (!mediaId) mediaId = resolveMediaId();
      if (!mediaId) return envelope(CAP_COMMENTS, [], { found: false, reason: "media_id_unknown", error: "could not resolve the media id for " + (shortcode || currentHref()) });
      var items = [], seen = {}, pages = 0, nextMin = "", hasMore = false, stopped = null, header = postHeaderFromCaptures(shortcode);
      function take(json) {
        (Array.isArray(json.comments) ? json.comments : []).forEach(function (c) {
          var rec = commentRecord(c, 0);
          if (rec && !seen[rec.id]) { seen[rec.id] = 1; items.push(rec); }
        });
        nextMin = str(json.next_min_id);
        hasMore = !!json.has_more_comments;
        pages += 1;
      }
      var first = (cap ? Promise.resolve({ status: 200, json: cap.response }) : restGet("/api/v1/media/" + mediaId + "/comments/?can_support_threading=true&permalink_enabled=false"))
        .catch(function (e) { return { status: 0, json: null, error: String(e && e.message || e) }; });
      return first.then(function (r) {
        if (!r.json || !Array.isArray(r.json.comments)) return envelope(CAP_COMMENTS, [], { found: false, media_id: mediaId, error: r.error ? "comments fetch failed: " + r.error : "comments answered HTTP " + r.status });
        take(r.json);
        function step() {
          if (items.length >= maxComments) { stopped = "max_comments"; return Promise.resolve(); }
          if (pages >= maxPages) { if (hasMore) stopped = "page_cap_hit"; return Promise.resolve(); }
          if (!hasMore || !nextMin) return Promise.resolve();
          return restGet("/api/v1/media/" + mediaId + "/comments/?can_support_threading=true&permalink_enabled=false&min_id=" + encodeURIComponent(nextMin)).then(function (r2) {
            if (!r2.json || !Array.isArray(r2.json.comments)) { stopped = "replay_failed_" + r2.status; return; }
            var before = items.length; take(r2.json);
            if (items.length === before) { stopped = "no_new_items"; return; }
            return wait(800).then(step);
          }).catch(function (e) { stopped = "fetch_error"; });
        }
        return step().then(function () {
          var declared = header && header.engagement ? header.engagement.comments : (num(r.json.comment_count));
          // Measured 2026-09-10: two posts with comment_count 7 and 8 answered comments:[] to
          // this account while Instagram's own page showed none either — the owner limits who
          // sees them. That is a platform restriction, not a broken extractor, and the record
          // says so instead of looking like "no comments".
          var hidden = items.length === 0 && !hasMore && typeof declared === "number" && declared > 0;
          return envelope(CAP_COMMENTS, items.slice(0, maxComments), { found: items.length > 0, media_id: mediaId, shortcode: shortcode, post: header, comment_count: declared,
            reason: hidden ? "comments_hidden" : null,
            pages_fetched: pages, page_info: { next_min_id: nextMin, has_more: hasMore, resumable: !!(hasMore && nextMin) }, stopped_because: stopped });
        });
      });
    });
  }

  // ------------------------------------------------------------- _discover.ig
  // Same substring rule as core/schema.js findSensitiveKeys: the skeleton is the one output that
  // never passes through ig_normalize's sanitizeDeep, so it redacts such keys itself.
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
      var row = { kind: c.kind, url: str(c.url).replace(/\?.*$/, ""), queryName: c.queryName, docId: c.docId, size: c.response ? JSON.stringify(c.response).length : 0, dataKeys: isObj(responseData(c)) ? Object.keys(responseData(c)).slice(0, 8) : [] };
      if (q && (String(c.queryName).indexOf(q) > -1 || String(c.url).indexOf(q) > -1)) row.deep_skeleton = skeletonize(c.response, 0, { n: 2500 }, 18);
      return row;
    });
    return Promise.resolve(envelope(CAP_DISCOVER, caps, { queries: caps.map(function (c) { return c.queryName; }), doc_ids: { profile: store().docIdFor ? store().docIdFor("PolarisProfilePageContentQuery") : "", posts: store().docIdFor ? store().docIdFor("PolarisProfilePostsQuery") : "" } }));
  }

  // ------------------------------------------------------------- dispatch
  var CAPS = {};
  CAPS[CAP_PROFILE] = profileEnrich;
  CAPS[CAP_POSTS] = profilePosts;
  CAPS[CAP_SEARCH_POSTS] = searchPosts;
  CAPS[CAP_PEOPLE] = peopleSearch;
  CAPS[CAP_COMMENTS] = postComments;
  CAPS[CAP_DISCOVER] = discover;

  window.__soloIgRun = function (capId, inputs) {
    capId = String(capId || ""); inputs = inputs && typeof inputs === "object" ? inputs : {};
    var fn = CAPS[capId];
    if (!fn) return Promise.resolve(fail(capId, "no instagram extractor for " + capId));
    try { return Promise.resolve(fn(inputs)).catch(function (e) { return fail(capId, e); }); }
    catch (e) { return Promise.resolve(fail(capId, e)); }
  };
  window.__soloIgCapabilities = Object.keys(CAPS);
  // The data point's graphql_manifest for an instagram.com page (live + prefetched captures).
  window.__soloIgManifest = function () {
    var caps = captures(), byName = {}, order = [];
    for (var i = 0; i < caps.length; i++) {
      var c = caps[i]; if (!c) continue;
      var qn = str(c.queryName) || ("doc_" + str(c.docId));
      if (!byName[qn]) { byName[qn] = { queryName: str(c.queryName), docId: str(c.docId), kind: str(c.kind), variableKeys: isObj(c.variables) ? Object.keys(c.variables).slice(0, 40) : [], count: 0 }; order.push(qn); }
      byName[qn].count += 1;
    }
    return { available: caps.length > 0, captureCount: caps.length, manifest: order.map(function (k) { return byName[k]; }) };
  };
  window.__soloIgVersion = VERSION;
  // Exposed for the offline harness (tests/test_ig_extract.js); not used by background.js.
  window.__soloIgInternals = { prefetched: prefetched, mediaIdFromCaptures: mediaIdFromCaptures, postHeaderFromCaptures: postHeaderFromCaptures, postRecord: postRecord, commentRecord: commentRecord, userRef: userRef, emailsIn: emailsIn, phonesIn: phonesIn, serpItems: serpItems, connectionItems: connectionItems, parseCount: parseCount, postUrl: postUrl };
})();
