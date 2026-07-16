/*
 * gql_extract.js — Solo Agency Local Collector
 *
 * PHASE 1 (hybrid capture). Injected into the MAIN world ONLY while a collection
 * job is running, AFTER the page has been scrolled by the collector. It reads
 * the passive GraphQL captures left by gql_intercept.js on window.__soloGql and
 * turns them into two things:
 *
 *   1. graphql_records  — a BEST-EFFORT, generic structured extraction (posts +
 *      entities) found by walking the captured responses. This is deliberately
 *      conservative and screen-agnostic: it grabs what it can confidently
 *      recognize and ignores the rest. It is written ONLY into new data_point
 *      fields; it never overwrites the existing HTML-derived fields. Per-screen
 *      precision extractors are added in later phases.
 *
 *   2. graphql_manifest — a compact, privacy-safe description of WHICH queries
 *      Facebook actually issued on this screen: friendly-name, doc_id, the
 *      variable KEYS (not values), a count, and a shape SKELETON of the response
 *      (keys + types only, no values). This is the raw material the team uses to
 *      author a precise per-screen extractor in the next phase.
 *
 * Exposes window.__soloGqlExtract(opts) -> result object. Pure/read-only over
 * the captures; never mutates the page and never throws (returns a safe empty
 * shape on any failure).
 */
(function () {
  "use strict";

  var MAX_POSTS = 80;
  var MAX_ENTITIES = 120;
  var MAX_MANIFEST = 40;
  var MAX_TEXT = 2000;
  var WALK_DEPTH = 8;
  var SKELETON_DEPTH = 9;      // deep enough to reveal node paths for authoring extractors
  var SKELETON_KEYS = 30;      // per-level key cap
  var SKELETON_BUDGET = 300;   // hard node budget per query (bounds payload size)

  var ENTITY_TYPES = {
    User: 1, Page: 1, Group: 1, Profile: 1, GroupMemberProfile: 1,
    Event: 1, ProfilePlusEntity: 1
  };

  // ---- generic helpers ----------------------------------------------------

  function isObj(v) { return v && typeof v === "object"; }

  // Depth-bounded search for the first value at any key whose name is in `names`
  // and whose value passes `ok`. Used to fish scalar fields out of unfamiliar
  // node shapes without hardcoding full JSON paths.
  function deepFind(obj, names, ok, depth, maxDepth) {
    var md = maxDepth || WALK_DEPTH;
    if (!isObj(obj) || depth > md) return undefined;
    if (Array.isArray(obj)) {
      for (var i = 0; i < obj.length && i < 200; i++) {
        var r = deepFind(obj[i], names, ok, depth + 1, md);
        if (r !== undefined) return r;
      }
      return undefined;
    }
    var keys = Object.keys(obj);
    for (var j = 0; j < keys.length; j++) {
      var k = keys[j], v = obj[k];
      if (names[k] && (!ok || ok(v))) return v;
    }
    for (var m = 0; m < keys.length; m++) {
      var vv = obj[keys[m]];
      if (isObj(vv)) {
        var rr = deepFind(vv, names, ok, depth + 1, md);
        if (rr !== undefined) return rr;
      }
    }
    return undefined;
  }

  var TEXT_HOLDER = { message: 1, body: 1, title: 1, preferred_body: 1, text_with_entities: 1 };
  function deepText(obj, depth) {
    if (!isObj(obj) || depth > WALK_DEPTH) return "";
    if (Array.isArray(obj)) {
      for (var i = 0; i < obj.length && i < 100; i++) {
        var r = deepText(obj[i], depth + 1);
        if (r) return r;
      }
      return "";
    }
    // A {text: "..."} under a message/body/title-ish parent is the usual shape.
    if (typeof obj.text === "string" && obj.text.trim()) return obj.text;
    var keys = Object.keys(obj);
    for (var j = 0; j < keys.length; j++) {
      if (TEXT_HOLDER[keys[j]] && isObj(obj[keys[j]])) {
        var t = deepText(obj[keys[j]], depth + 1);
        if (t) return t;
      }
    }
    for (var m = 0; m < keys.length; m++) {
      if (isObj(obj[keys[m]])) {
        var tt = deepText(obj[keys[m]], depth + 1);
        if (tt) return tt;
      }
    }
    return "";
  }

  function firstString(obj, names) {
    return deepFind(obj, names, function (v) { return typeof v === "string" && v.trim(); }, 0);
  }
  function firstNumber(obj, names) {
    return deepFind(obj, names, function (v) { return typeof v === "number" && v > 0; }, 0);
  }

  // Recursively find every array named "edges" (with a few nodes) anywhere in a
  // response. This is the one genuinely generic discovery mechanism; per-screen
  // phases can replace it with exact paths.
  function findEdgeArrays(obj, out, depth) {
    if (!isObj(obj) || depth > WALK_DEPTH) return;
    if (Array.isArray(obj)) {
      for (var i = 0; i < obj.length && i < 200; i++) findEdgeArrays(obj[i], out, depth + 1);
      return;
    }
    if (Array.isArray(obj.edges) && obj.edges.length > 0) out.push(obj.edges);
    var keys = Object.keys(obj);
    for (var j = 0; j < keys.length; j++) {
      if (isObj(obj[keys[j]])) findEdgeArrays(obj[keys[j]], out, depth + 1);
    }
  }

  // ---- record builders ----------------------------------------------------

  function nodeLooksLikePost(node) {
    if (!isObj(node) || !node.id) return false;
    return !!(node.comet_sections || node.message || node.story ||
      node.feedback || node.preferred_body || node.timeline_moments_info);
  }

  function buildPost(node) {
    var text = deepText(node, 0);
    var url = firstString(node, { wwwURL: 1, url: 1, permalink_url: 1, permalink: 1 });
    var actorName = firstString(node, { name: 1 });
    var created = firstNumber(node, { creation_time: 1, created_time: 1, publish_time: 1 });
    var typename = typeof node.__typename === "string" ? node.__typename : "";
    var post = {
      id: String(node.id),
      typename: typename,
      text: text ? String(text).slice(0, MAX_TEXT) : "",
      url: url ? String(url) : "",
      actor: actorName ? String(actorName) : "",
      created_time: created || 0
    };
    // Only keep it if it carries at least some substance.
    if (!post.text && !post.url) return null;
    return post;
  }

  // Entity identity must be read SHALLOWLY (the node's own fields). A deep search
  // would let a container node absorb a descendant's name/url — e.g. a profile
  // wrapper stealing the identity of a post actor nested inside it.
  function shallowName(node) {
    if (typeof node.name === "string" && node.name.trim()) return node.name;
    if (isObj(node.name) && typeof node.name.text === "string" && node.name.text.trim()) return node.name.text;
    if (typeof node.title === "string" && node.title.trim()) return node.title;
    if (isObj(node.title) && typeof node.title.text === "string" && node.title.text.trim()) return node.title.text;
    return "";
  }
  function shallowUrl(node) {
    if (typeof node.url === "string" && node.url) return node.url;
    if (typeof node.profile_url === "string" && node.profile_url) return node.profile_url;
    if (typeof node.wwwURL === "string" && node.wwwURL) return node.wwwURL;
    return "";
  }

  function buildEntity(node) {
    if (!isObj(node) || !node.__typename || !ENTITY_TYPES[node.__typename]) return null;
    var name = shallowName(node);
    var url = shallowUrl(node);
    if (!name && !url) return null;
    return {
      id: node.id ? String(node.id) : "",
      type: String(node.__typename),
      name: name ? String(name).slice(0, 300) : "",
      url: url ? String(url) : ""
    };
  }

  // Walk any object collecting entity-typed nodes (profiles/pages/groups) even
  // when they are not inside an edges list (e.g. actors, owners, members).
  function collectEntities(obj, into, seen, depth) {
    if (!isObj(obj) || depth > WALK_DEPTH) return;
    if (Array.isArray(obj)) {
      for (var i = 0; i < obj.length && i < 300; i++) collectEntities(obj[i], into, seen, depth + 1);
      return;
    }
    var ent = buildEntity(obj);
    if (ent) {
      var key = ent.id || (ent.type + "|" + ent.name + "|" + ent.url);
      if (key && !seen[key] && into.length < MAX_ENTITIES) {
        seen[key] = 1;
        into.push(ent);
      }
    }
    var keys = Object.keys(obj);
    for (var j = 0; j < keys.length; j++) {
      if (isObj(obj[keys[j]])) collectEntities(obj[keys[j]], into, seen, depth + 1);
    }
  }

  // ---- manifest (shape skeleton, values stripped) -------------------------

  // Structure-only description (no values). `budget` caps the total node count so
  // even a huge, deeply nested feed response yields a bounded skeleton. maxDepth
  // defaults to SKELETON_DEPTH; a deeper cap is used for on-demand debug dumps.
  function skeletonize(v, depth, budget, maxDepth) {
    var md = maxDepth || SKELETON_DEPTH;
    if (depth > md || budget.n <= 0) return "…";
    budget.n -= 1;
    if (v === null) return "null";
    if (Array.isArray(v)) {
      return v.length ? ["[" + v.length + "]", skeletonize(v[0], depth + 1, budget, md)] : "[0]";
    }
    if (typeof v === "object") {
      var out = {};
      var keys = Object.keys(v).slice(0, SKELETON_KEYS);
      for (var i = 0; i < keys.length; i++) out[keys[i]] = skeletonize(v[keys[i]], depth + 1, budget, md);
      return out;
    }
    return typeof v; // "string" | "number" | "boolean"
  }

  // ---- main ---------------------------------------------------------------

  window.__soloGqlExtract = function (opts) {
    opts = opts || {};
    var empty = { available: false, posts: [], entities: [], manifest: [], captureCount: 0 };
    try {
      var CAP = window.__soloGql;
      if (!CAP || !Array.isArray(CAP.captures) || !CAP.captures.length) return empty;
      var caps = CAP.captures;

      var posts = [], entities = [];
      var seenPost = {}, seenEnt = {};
      var manifest = {}; // queryName -> entry

      for (var c = 0; c < caps.length; c++) {
        var cap = caps[c];
        if (!cap) continue;

        // --- manifest entry (always, even if response missing) ---
        var qn = cap.queryName || ("doc_" + (cap.docId || "unknown"));
        if (!manifest[qn]) {
          manifest[qn] = {
            queryName: cap.queryName || "",
            docId: cap.docId || "",
            variableKeys: cap.variables ? Object.keys(cap.variables).slice(0, 40) : [],
            count: 0,
            skeleton: null
          };
        }
        manifest[qn].count += 1;

        var chunks = Array.isArray(cap.response) ? cap.response : (cap.response ? [cap.response] : []);
        for (var k = 0; k < chunks.length; k++) {
          var chunk = chunks[k];
          if (!isObj(chunk)) continue;
          if (!manifest[qn].skeleton) {
            try { manifest[qn].skeleton = skeletonize(chunk, 0, { n: SKELETON_BUDGET }); } catch (e) { /* ignore */ }
          }

          // posts from edge arrays
          var edgeArrays = [];
          try { findEdgeArrays(chunk, edgeArrays, 0); } catch (e) { edgeArrays = []; }
          for (var e = 0; e < edgeArrays.length; e++) {
            var edges = edgeArrays[e];
            for (var i = 0; i < edges.length; i++) {
              var node = edges[i] && (edges[i].node || edges[i]);
              if (nodeLooksLikePost(node) && posts.length < MAX_POSTS) {
                var pkey = String(node.id);
                if (!seenPost[pkey]) {
                  var p = buildPost(node);
                  if (p) { seenPost[pkey] = 1; posts.push(p); }
                }
              }
            }
          }

          // entities anywhere (profiles/pages/groups)
          try { collectEntities(chunk, entities, seenEnt, 0); } catch (e2) { /* ignore */ }
        }
      }

      var manifestList = [];
      var mkeys = Object.keys(manifest);
      for (var mi = 0; mi < mkeys.length && manifestList.length < MAX_MANIFEST; mi++) {
        manifestList.push(manifest[mkeys[mi]]);
      }

      return {
        available: true,
        captureCount: caps.length,
        posts: posts,
        entities: entities,
        manifest: manifestList
      };
    } catch (err) {
      empty.error = String(err && err.message ? err.message : err);
      return empty;
    }
  };

  // =========================================================================
  // Capability-specific extractors (PRECISE, per Facebook screen).
  // Each maps ONE screen's exact GraphQL shape to a typed record set from the
  // capability catalog (collector_capabilities.json). Registered in
  // CAPABILITY_EXTRACTORS and dispatched by __soloGqlExtractCapability(id).
  // =========================================================================

  function getPath(obj, path) {
    var parts = path.split(".");
    var cur = obj;
    for (var i = 0; i < parts.length; i++) {
      if (!isObj(cur)) return undefined;
      cur = cur[parts[i]];
    }
    return cur;
  }

  function actorRef(actor) {
    if (!isObj(actor)) return null;
    return {
      type: "profile",
      id: actor.id ? String(actor.id) : "",
      name: typeof actor.name === "string" ? actor.name : "",
      url: typeof actor.url === "string" ? actor.url : ""
    };
  }

  function postAttachments(node) {
    var out = [];
    var atts = node && node.attachments;
    if (!Array.isArray(atts)) return out;
    for (var i = 0; i < atts.length && i < 10; i++) {
      var a = atts[i];
      if (!isObj(a)) continue;
      var media = a.media;
      out.push({
        type: (isObj(media) && typeof media.__typename === "string") ? media.__typename : "",
        url: firstString(a, { url: 1, uri: 1, wwwURL: 1 }) || ""
      });
    }
    return out;
  }

  // Facebook stores engagement counts inconsistently: a plain number, a numeric
  // string ("1,234"), or an object like { count: 42 }. Coerce all three.
  function coerceCount(v) {
    if (typeof v === "number" && isFinite(v)) return v;
    if (typeof v === "string" && /^\d[\d,\.]*$/.test(v.trim())) {
      return parseInt(v.replace(/[,\.]/g, ""), 10);
    }
    if (isObj(v) && typeof v.count === "number") return v.count;
    return null;
  }

  function objCount(v) { return (isObj(v) && typeof v.count === "number") ? v.count : null; }

  // Engagement lives DEEP inside comet_sections' UFI (unified feedback interface)
  // subtree — well beyond the default WALK_DEPTH — so we target the known Comet
  // path with a deep-search fallback. Verified against real group_feed nodes:
  //   ufi = comet_sections.feedback.story.story_ufi_container.story
  //         .feedback_context.feedback_target_with_context
  //   reactions total = <ufi>...reaction_count.count  (the {count} object form;
  //                     per-emoji counts under top_reactions.edges are bare nums)
  //   comments  total = <ufi>.comment_rendering_instance.comments.total_count
  function postEngagement(node) {
    var cs = node.comet_sections;
    if (!isObj(cs)) return null;
    var ufi = getPath(cs, "feedback.story.story_ufi_container.story.feedback_context.feedback_target_with_context");
    if (!isObj(ufi)) ufi = deepFind(cs, { feedback_target_with_context: 1 }, isObj, 0, 16);
    if (!isObj(ufi)) return null;

    // reactions: the {count:N} object form is the grand total (bare-number
    // reaction_count values under top_reactions are per-emoji, so require {count}).
    var reactions = objCount(deepFind(ufi, { reaction_count: 1 }, function (v) { return objCount(v) !== null; }, 0, 16));

    // comments: direct total, with a deep fallback to a {comments:{total_count}}.
    var comments = coerceCount(getPath(ufi, "comment_rendering_instance.comments.total_count"));
    if (comments === null) {
      var cObj = deepFind(ufi, { comments: 1 }, function (v) { return isObj(v) && typeof v.total_count === "number"; }, 0, 16);
      comments = isObj(cObj) ? cObj.total_count : null;
    }

    // shares: best-effort (often absent on group posts).
    var shares = coerceCount(deepFind(ufi, { i18n_share_count: 1, share_count: 1, reshare_count: 1 }, function (v) { return coerceCount(v) !== null; }, 0, 14));

    if (reactions === null && comments === null && shares === null) return null;
    return { reactions: reactions || 0, comments: comments || 0, shares: shares === null ? 0 : shares };
  }

  // Best-effort EntityRef for the group/page a post was posted to (node.to).
  function groupRef(node) {
    var to = node && node.to;
    if (!isObj(to)) return null;
    var name = shallowName(to);
    var url = shallowUrl(to);
    if (!name && !url && !to.id) return null;
    return {
      type: "group",
      id: to.id ? String(to.id) : "",
      name: name || "",
      url: url || ""
    };
  }

  // Map a Facebook "story" feed node -> PostRecord. Shared by every screen whose
  // results ARE posts (group feed, profile timeline, and in-group keyword search),
  // because they all carry the identical story-node shape (actors, comet_sections,
  // permalink_url, post_id, feedback, attachments, to). Returns null for non-posts.
  function postRecordFromStoryNode(node) {
    if (!isObj(node) || !node.id) return null;
    var actor = Array.isArray(node.actors) ? node.actors[0] : null;
    return {
      id: String(node.id),
      post_id: node.post_id ? String(node.post_id) : "",
      url: (typeof node.permalink_url === "string" && node.permalink_url)
        ? node.permalink_url
        : (firstString(node, { wwwURL: 1, url: 1, permalink_url: 1, permalink: 1 }) || ""),
      actor: actorRef(actor),
      text: (deepText(node.comet_sections, 0) || deepText(node, 0) || "").slice(0, 4000),
      created_time: firstNumber(node, { creation_time: 1, created_time: 1, publish_time: 1 }) || 0,
      attachments: postAttachments(node),
      engagement: postEngagement(node),
      group: groupRef(node)
    };
  }

  // fb.group.posts — data.node.group_feed.edges[].node (validated shape).
  function extractGroupPosts(caps, opts) {
    opts = opts || {};
    var items = [], seen = {}, sourceQuery = "", firstNode = null;
    for (var c = 0; c < caps.length; c++) {
      var cap = caps[c];
      if (!cap) continue;
      var chunks = Array.isArray(cap.response) ? cap.response : (cap.response ? [cap.response] : []);
      for (var k = 0; k < chunks.length; k++) {
        var edges = getPath(chunks[k], "data.node.group_feed.edges");
        if (!Array.isArray(edges)) continue;
        sourceQuery = cap.queryName || sourceQuery;
        for (var i = 0; i < edges.length; i++) {
          var node = edges[i] && edges[i].node;
          if (!isObj(node) || !node.id || seen[node.id]) continue;
          var rec = postRecordFromStoryNode(node);
          if (!rec) continue;
          seen[node.id] = 1;
          if (!firstNode) firstNode = node;
          items.push(rec);
        }
      }
    }
    var result = { capability: "fb.group.posts", schema: "PostRecord[]", source_query: sourceQuery, count: items.length, items: items };
    // On-demand debug: a deep skeleton (shape only, no values) of one real node
    // so we can pin down engagement/attachment paths without dumping raw data.
    if (opts.debug && firstNode) {
      try { result._debug_node_skeleton = skeletonize(firstNode, 0, { n: 1200 }, 16); } catch (e) { /* ignore */ }
    }
    return result;
  }

  // fb.group.search_posts — keyword search INSIDE a group. Same shared SERP query
  // (SearchCometResultsPaginatedResultsQuery) as the other searches, but the
  // result entity is a POST: the story sits at
  //   edge.rendering_strategy.view_model.click_model.story
  // (identical story-node shape as group_feed, so we reuse postRecordFromStoryNode).
  // Entity searches (groups/people) put a `.profile` there instead of a story, so
  // requiring a story cleanly excludes them.
  function extractGroupSearchPosts(caps, opts) {
    opts = opts || {};
    var items = [], seen = {}, sourceQuery = "", firstNode = null;
    for (var c = 0; c < caps.length; c++) {
      var cap = caps[c];
      if (!cap) continue;
      if (String(cap.queryName || "").indexOf("SearchComet") === -1) continue;
      var chunks = Array.isArray(cap.response) ? cap.response : (cap.response ? [cap.response] : []);
      for (var k = 0; k < chunks.length; k++) {
        var edges = getPath(chunks[k], "data.serpResponse.results.edges");
        if (!Array.isArray(edges)) continue;
        sourceQuery = cap.queryName || sourceQuery;
        for (var i = 0; i < edges.length; i++) {
          var story = getPath(edges[i], "rendering_strategy.view_model.click_model.story");
          if (!isObj(story) || !story.id || seen[story.id]) continue;
          var rec = postRecordFromStoryNode(story);
          if (!rec) continue;
          seen[story.id] = 1;
          if (!firstNode) firstNode = story;
          items.push(rec);
        }
      }
    }
    var result = { capability: "fb.group.search_posts", schema: "PostRecord[]", source_query: sourceQuery, count: items.length, items: items };
    if (opts.debug && firstNode) {
      try { result._debug_node_skeleton = skeletonize(firstNode, 0, { n: 1200 }, 16); } catch (e) { /* ignore */ }
    }
    return result;
  }

  // "12 mutual friends" / "12 bạn chung" -> 12. Only when the subtitle is a
  // mutual-friends line (it can also be a job/tagline), else null.
  function parseMutual(subtitle) {
    if (!subtitle || !/mutual|chung|共同|친구/i.test(subtitle)) return null;
    var m = String(subtitle).match(/[\d.,]+/);
    return m ? parseInt(m[0].replace(/[.,]/g, ""), 10) : null;
  }

  // fb.profile.friends — data.node.pageItems.edges[] on a *Friends*List* query.
  // `pageItems` is a generic profile-collection container (photos, groups, …),
  // so we scope to captures whose query name contains "Friends".
  function extractProfileFriends(caps, opts) {
    opts = opts || {};
    var items = [], seen = {}, sourceQuery = "", firstNode = null;
    for (var c = 0; c < caps.length; c++) {
      var cap = caps[c];
      if (!cap) continue;
      if (String(cap.queryName || "").indexOf("Friends") === -1) continue;
      var chunks = Array.isArray(cap.response) ? cap.response : (cap.response ? [cap.response] : []);
      for (var k = 0; k < chunks.length; k++) {
        var edges = getPath(chunks[k], "data.node.pageItems.edges");
        if (!Array.isArray(edges)) continue;
        sourceQuery = cap.queryName || sourceQuery;
        for (var i = 0; i < edges.length; i++) {
          var li = edges[i] && edges[i].node; // list-item node
          if (!isObj(li)) continue;
          if (!firstNode) firstNode = li;
          var ent = isObj(li.node) ? li.node : null; // the friend entity
          var name = getPath(li, "title.text");
          var url = (typeof li.url === "string" && li.url) ? li.url : (ent && typeof ent.url === "string" ? ent.url : "");
          var id = ent && ent.id ? String(ent.id) : "";
          var subtitle = getPath(li, "subtitle_text.text");
          subtitle = typeof subtitle === "string" ? subtitle : "";
          if (!name && !url) continue;
          var key = id || url || name;
          if (seen[key]) continue;
          seen[key] = 1;
          items.push({
            id: id,
            name: name ? String(name) : "",
            url: url ? String(url) : "",
            subtitle: subtitle,
            mutual_friends: parseMutual(subtitle),
            industry_hint: null
          });
        }
      }
    }
    var result = { capability: "fb.profile.friends", schema: "ProfileSummary[]", source_query: sourceQuery, count: items.length, items: items };
    if (opts.debug && firstNode) {
      try { result._debug_node_skeleton = skeletonize(firstNode, 0, { n: 1200 }, 16); } catch (e) { /* ignore */ }
    }
    return result;
  }


  // ===== Phase 2 batch: extractors drafted+verified via workflow (2026-07-15) =====

  // fb.groups.search — SearchCometResultsPaginatedResultsQuery (the shared search
  // SERP). Group results live at:
  //   data.serpResponse.results.edges[].rendering_strategy.view_model
  // where the entity itself is under .profile (preferred) and/or .loggedProfile.
  // The SERP is a MIXED list (people/pages/groups share this query), so we keep
  // ONLY group entities, gated by a group typename/role/type OR a canonical
  // facebook.com/groups/<id> url. Output is a deduped EntityRef[] of groups.

  // True iff `u` is a canonical Facebook group url (facebook.com/groups/<id>).
  function isGroupUrl(u) {
  return typeof u === "string" && /facebook\.com\/groups\//i.test(u);
  }

  // Pull the <id> segment out of a facebook.com/groups/<id> url (numeric id or
  // vanity slug). Used only as a fallback when the entity has no own id.
  function groupIdFromUrl(u) {
  if (typeof u !== "string") return "";
  var m = u.match(/\/groups\/([^\/?#]+)/i);
  return m ? m[1] : "";
  }

  // Decide whether a SERP edge is a Group. Any ONE of these signals qualifies it;
  // people/pages carry none of them (no Group typename/role, no /groups/ url), so
  // false positives are effectively impossible while recall stays high.
  function edgeIsGroup(node, vm, profile, logged) {
  if (isObj(profile) && profile.__typename === "Group") return true;
  if (isObj(logged) && logged.__typename === "Group") return true;
  if (isObj(logged) && typeof logged.type === "string" && logged.type.toLowerCase() === "group") return true;
  if (isObj(node) && typeof node.role === "string" && /group/i.test(node.role)) return true;
  if (isObj(vm) && typeof vm.__typename === "string" && /group/i.test(vm.__typename)) return true;
  if (isObj(profile) && (isGroupUrl(profile.url) || isGroupUrl(profile.profile_url))) return true;
  if (isObj(logged) && (isGroupUrl(logged.url) || isGroupUrl(logged.profile_url))) return true;
  return false;
  }

  function extractGroupsSearch(caps, opts) {
  opts = opts || {};
  var items = [], seen = {}, sourceQuery = "", firstNode = null;
  for (var c = 0; c < caps.length; c++) {
    var cap = caps[c];
    if (!cap) continue;
    if (String(cap.queryName || "").indexOf("SearchCometResultsPaginatedResults") === -1) continue;
    var chunks = Array.isArray(cap.response) ? cap.response : (cap.response ? [cap.response] : []);
    for (var k = 0; k < chunks.length; k++) {
      var edges = getPath(chunks[k], "data.serpResponse.results.edges");
      if (!Array.isArray(edges)) continue;
      sourceQuery = cap.queryName || sourceQuery;
      for (var i = 0; i < edges.length; i++) {
        var edge = edges[i];
        if (!isObj(edge)) continue;
        var node = edge.node;
        var vm = getPath(edge, "rendering_strategy.view_model");
        if (!isObj(vm)) continue;
        var profile = isObj(vm.profile) ? vm.profile : null;
        var logged = isObj(vm.loggedProfile) ? vm.loggedProfile : null;
        if (!edgeIsGroup(node, vm, profile, logged)) continue;

        // Prefer the richer `profile` entity; fall back to `loggedProfile`.
        var src = profile || logged;
        if (!src) continue;

        var url = "";
        if (isObj(profile)) {
          url = (typeof profile.url === "string" && profile.url) ? profile.url
            : ((typeof profile.profile_url === "string" && profile.profile_url) ? profile.profile_url : "");
        }
        if (!url && isObj(logged)) {
          url = (typeof logged.url === "string" && logged.url) ? logged.url
            : ((typeof logged.profile_url === "string" && logged.profile_url) ? logged.profile_url : "");
        }

        var id = (src.id !== undefined && src.id !== null && src.id !== "") ? String(src.id) : groupIdFromUrl(url);

        var name = shallowName(src);
        if (!name && isObj(profile)) name = shallowName(profile);
        if (!name && isObj(logged)) name = shallowName(logged);
        if (!name && typeof vm.profile_name_with_possible_nickname === "string") {
          name = vm.profile_name_with_possible_nickname;
        }

        var key = id || url || name;
        if (!key || seen[key]) continue;
        seen[key] = 1;
        if (!firstNode) firstNode = edge;
        items.push({
          type: "group",
          id: id ? String(id) : "",
          name: name ? String(name) : "",
          url: url ? String(url) : ""
        });
      }
    }
  }
  var result = { capability: "fb.groups.search", schema: "EntityRef[]", source_query: sourceQuery, count: items.length, items: items };
  if (opts.debug && firstNode) {
    try { result._debug_node_skeleton = skeletonize(firstNode, 0, { n: 1200 }, 16); } catch (e) { /* ignore */ }
  }
  return result;
  }

  // Best-effort industry/occupation classifier for a SERP result's descriptor
  // line (e.g. "Realtor at Keller Williams" -> "real estate"). Pure keyword
  // scan; returns a normalized label or null when nothing recognizable matches.
  function industryHint(text) {
  if (typeof text !== "string" || !text.trim()) return null;
  var t = text.toLowerCase();
  var map = [
    ["real estate", /real ?estate|realtor|realty|broker/],
    ["mortgage", /mortgage|loan officer|lender|lending|\bnmls\b/],
    ["insurance", /insurance|underwrit/],
    ["finance", /financ|accountant|\bcpa\b|bookkeep|invest/],
    ["marketing", /marketing|social media|content creator|\bads?\b|brand/],
    ["photography", /photograph|videograph/],
    ["fitness", /fitness|personal trainer|\bcoach\b|yoga|pilates/],
    ["beauty", /salon|hair stylist|makeup|\bbeauty\b|esthet|barber|nails?/],
    ["automotive", /car sales|auto sales|dealership|automotive/],
    ["healthcare", /\bnurse\b|doctor|dentist|therapist|medical|\bclinic\b|chiropract/],
    ["legal", /attorney|lawyer|\blegal\b|law firm|paralegal/],
    ["education", /teacher|professor|tutor|educat|\bcoach\b/],
    ["construction", /contractor|construction|builder|remodel|roofing|\bhvac\b/],
    ["food", /\bchef\b|restaurant|caterer|bakery|\bcook\b/]
  ];
  for (var i = 0; i < map.length; i++) {
    if (map[i][1].test(t)) return map[i][0];
  }
  return null;
  }

  // fb.people.search — data.serpResponse.results.edges[] on the shared SERP query
  // SearchCometResultsPaginatedResultsQuery. Each result's entity lives at
  //   edge.rendering_strategy.view_model
  // with the person under view_model.profile (__typename "User"); pages/groups
  // are skipped by requiring a User profile. The occupation/descriptor line is
  //   view_model.primary_snippet_text_with_entities.text
  // and the mutual-friends facepile line is
  //   view_model.snippet_with_facepile.simple_text_with_entities.text
  function extractPeopleSearch(caps, opts) {
  opts = opts || {};
  var items = [], seen = {}, sourceQuery = "", firstNode = null;
  for (var c = 0; c < caps.length; c++) {
    var cap = caps[c];
    if (!cap) continue;
    // Shared SERP query; scope to it so unrelated captures never leak in.
    if (String(cap.queryName || "").indexOf("SearchComet") === -1) continue;
    var chunks = Array.isArray(cap.response) ? cap.response : (cap.response ? [cap.response] : []);
    for (var k = 0; k < chunks.length; k++) {
      var edges = getPath(chunks[k], "data.serpResponse.results.edges");
      if (!Array.isArray(edges)) continue;
      sourceQuery = cap.queryName || sourceQuery;
      for (var i = 0; i < edges.length; i++) {
        var edge = edges[i];
        if (!isObj(edge)) continue;
        var vm = getPath(edge, "rendering_strategy.view_model");
        if (!isObj(vm)) continue;
        var profile = isObj(vm.profile) ? vm.profile : null;
        var logged = isObj(vm.loggedProfile) ? vm.loggedProfile : null;

        // Filter to PEOPLE (User); skip pages/groups and non-profile modules.
        var tn = (profile && typeof profile.__typename === "string") ? profile.__typename : "";
        var ltype = (logged && typeof logged.type === "string") ? logged.type : "";
        var isUser = tn === "User" || /^user$/i.test(ltype);
        if (!isUser) continue;
        if (!firstNode) firstNode = vm;

        var id = (profile && profile.id) ? String(profile.id)
          : (logged && logged.id ? String(logged.id) : "");
        var name = (profile && typeof profile.name === "string" && profile.name) ? profile.name
          : (typeof vm.profile_name_with_possible_nickname === "string" && vm.profile_name_with_possible_nickname) ? vm.profile_name_with_possible_nickname
          : (logged && typeof logged.name === "string" ? logged.name : "");
        var url = (profile && typeof profile.profile_url === "string" && profile.profile_url) ? profile.profile_url
          : (profile && typeof profile.url === "string" && profile.url) ? profile.url
          : (logged && typeof logged.url === "string" ? logged.url : "");

        // Descriptor line: the occupation/location snippet, falling back to the
        // facepile snippet when the primary one is absent.
        var subtitle = getPath(vm, "primary_snippet_text_with_entities.text");
        var facepileText = getPath(vm, "snippet_with_facepile.simple_text_with_entities.text");
        facepileText = typeof facepileText === "string" ? facepileText : "";
        if (typeof subtitle !== "string" || !subtitle.trim()) subtitle = facepileText;
        subtitle = typeof subtitle === "string" ? subtitle : "";

        // Mutual friends come from the facepile line ("12 mutual friends"),
        // with the descriptor as a fallback source.
        var mutual = parseMutual(facepileText);
        if (mutual === null) mutual = parseMutual(subtitle);

        if (!name && !url && !id) continue;
        var key = id || url || name;
        if (seen[key]) continue;
        seen[key] = 1;

        items.push({
          id: id,
          name: name ? String(name) : "",
          url: url ? String(url) : "",
          subtitle: subtitle,
          mutual_friends: mutual,
          industry_hint: industryHint(subtitle)
        });
      }
    }
  }
  var result = { capability: "fb.people.search", schema: "ProfileSummary[]", source_query: sourceQuery, count: items.length, items: items };
  if (opts.debug && firstNode) {
    try { result._debug_node_skeleton = skeletonize(firstNode, 0, { n: 1200 }, 16); } catch (e) { /* ignore */ }
  }
  return result;
  }

  // fb.profile.posts — data.node.timeline_list_feed_units.edges[].node.
  // Emits the SAME PostRecord shape as extractGroupPosts, but sourced from the
  // profile timeline feed (ProfileCometTimelineFeedRefetchQuery, and its initial
  // ProfileCometTimelineFeedQuery variant — both expose this exact path). We scope
  // by the edges PATH rather than the query name so both the first-page and
  // pagination/refetch captures are picked up. A profile timeline post is owned by
  // the profile, not a group, so `group` is always null here (node.to is null).
  function extractProfileTimeline(caps, opts) {
    opts = opts || {};
    var items = [], seen = {}, sourceQuery = "", firstNode = null;
    for (var c = 0; c < caps.length; c++) {
      var cap = caps[c];
      if (!cap) continue;
      var chunks = Array.isArray(cap.response) ? cap.response : (cap.response ? [cap.response] : []);
      for (var k = 0; k < chunks.length; k++) {
        var edges = getPath(chunks[k], "data.node.timeline_list_feed_units.edges");
        if (!Array.isArray(edges)) continue;
        sourceQuery = cap.queryName || sourceQuery;
        for (var i = 0; i < edges.length; i++) {
          var node = edges[i] && edges[i].node;
          if (!isObj(node) || !node.id) continue;
          var id = String(node.id);
          if (seen[id]) continue;
          seen[id] = 1;
          if (!firstNode) firstNode = node;
          var actor = Array.isArray(node.actors) ? node.actors[0] : null;
          items.push({
            id: id,
            post_id: node.post_id ? String(node.post_id) : "",
            url: (typeof node.permalink_url === "string" && node.permalink_url)
              ? node.permalink_url
              : (firstString(node, { wwwURL: 1, url: 1, permalink_url: 1, permalink: 1 }) || ""),
            actor: actorRef(actor),
            text: (deepText(node.comet_sections, 0) || deepText(node, 0) || "").slice(0, 4000),
            created_time: firstNumber(node, { creation_time: 1, created_time: 1, publish_time: 1 }) || 0,
            attachments: postAttachments(node),
            engagement: postEngagement(node),
            group: null
          });
        }
      }
    }
    var result = { capability: "fb.profile.posts", schema: "PostRecord[]", source_query: sourceQuery, count: items.length, items: items };
    // On-demand debug: a deep skeleton (shape only, no values) of one real node
    // so engagement/attachment paths can be pinned down without dumping raw data.
    if (opts.debug && firstNode) {
      try { result._debug_node_skeleton = skeletonize(firstNode, 0, { n: 1200 }, 16); } catch (e) { /* ignore */ }
    }
    return result;
  }

  function extractNewsfeed(caps, opts) {
    opts = opts || {};
    var items = [], seen = {}, sourceQuery = "", firstNode = null;
    for (var c = 0; c < caps.length; c++) {
      var cap = caps[c];
      if (!cap) continue;
      var chunks = Array.isArray(cap.response) ? cap.response : (cap.response ? [cap.response] : []);
      for (var k = 0; k < chunks.length; k++) {
        var edges = getPath(chunks[k], "data.viewer.news_feed.edges");
        if (!Array.isArray(edges)) continue;
        sourceQuery = cap.queryName || sourceQuery;
        for (var i = 0; i < edges.length; i++) {
          var node = edges[i] && edges[i].node;
          if (!isObj(node) || !node.id) continue;
          // Skip ads/suggestions/reels that carry no comet story sections.
          if (!isObj(node.comet_sections)) continue;

          // The real post body lives at comet_sections.content.story.message; a
          // reshare/quote may only expose text on the inner content story.
          var contentStory = getPath(node, "comet_sections.content.story");
          var text = "";
          if (isObj(contentStory)) {
            text = deepText(contentStory.message, 0) || deepText(contentStory, 0) || "";
          }
          if (!text) text = deepText(node.message, 0) || "";
          // No story message => suggestion / reel / bare unit, not a real story.
          if (!text) continue;

          var id = String(node.id);
          if (seen[id]) continue;
          seen[id] = 1;
          if (!firstNode) firstNode = node;

          // Actor: prefer the feed-unit actor, fall back to the content story's.
          var actor = Array.isArray(node.actors) ? node.actors[0] : null;
          if (!actor && isObj(contentStory) && Array.isArray(contentStory.actors)) actor = contentStory.actors[0];

          // Attachments: node-level first, else the inner content story.
          var atts = postAttachments(node);
          if ((!atts || !atts.length) && isObj(contentStory)) atts = postAttachments(contentStory);

          // Engagement UFI hangs off the top-level comet_sections; fall back to
          // the content story's own sections for reshared/nested stories.
          var eng = postEngagement(node);
          if (!eng && isObj(contentStory)) eng = postEngagement(contentStory);

          items.push({
            id: id,
            post_id: node.post_id
              ? String(node.post_id)
              : ((isObj(contentStory) && contentStory.post_id) ? String(contentStory.post_id) : ""),
            url: (typeof node.permalink_url === "string" && node.permalink_url)
              ? node.permalink_url
              : (firstString(node, { wwwURL: 1, url: 1, permalink_url: 1, permalink: 1 }) || ""),
            actor: actorRef(actor),
            text: String(text).slice(0, 4000),
            created_time: firstNumber(node, { creation_time: 1, created_time: 1, publish_time: 1 }) || 0,
            attachments: atts || [],
            engagement: eng,
            group: groupRef(node)
          });
        }
      }
    }
    var result = { capability: "fb.newsfeed", schema: "PostRecord[]", source_query: sourceQuery, count: items.length, items: items };
    if (opts.debug && firstNode) {
      try { result._debug_node_skeleton = skeletonize(firstNode, 0, { n: 1200 }, 16); } catch (e) { /* ignore */ }
    }
    return result;
  }

  // _discover.deep — an authoring aid, not a real capability. Given inputs.query
  // (a substring of a fb_api_req_friendly_name), it returns a DEEP shape skeleton
  // (values stripped) of the first matching captured response, so a new screen's
  // exact field paths can be pinned down without ever dumping raw data. Also lists
  // every captured query name so you can pick the right one.
  function discoverDeep(caps, opts) {
    opts = opts || {};
    var want = String(opts.query || "");
    var seenQueries = [];
    var match = null;
    for (var c = 0; c < caps.length; c++) {
      var cap = caps[c];
      if (!cap) continue;
      if (cap.queryName && seenQueries.indexOf(cap.queryName) === -1) seenQueries.push(cap.queryName);
      if (!match && cap.response && (!want || String(cap.queryName || "").indexOf(want) !== -1)) {
        var chunk = Array.isArray(cap.response) ? cap.response[0] : cap.response;
        if (isObj(chunk)) match = { queryName: cap.queryName, chunk: chunk, full: cap.response };
      }
    }
    var out = { capability: "_discover.deep", available: !!match, count: match ? 1 : 0, items: [], queries: seenQueries };
    if (match) {
      out.matched_query = match.queryName;
      // @defer / streamed responses arrive as multiple NDJSON chunks: chunk 0 is
      // the skeleton with __dr deferred placeholders, later chunks patch in the
      // deferred data (e.g. reel owner + caption). Skeletonize ALL chunks so those
      // deferred fields are visible, not just chunk 0.
      out.chunk_count = Array.isArray(match.full) ? match.full.length : 1;
      try { out.deep_skeleton = skeletonize(match.full, 0, { n: 3500 }, 20); } catch (e) { /* ignore */ }
    } else {
      out.reason = "no_match";
    }
    return out;
  }

  var CAPABILITY_EXTRACTORS = {
    "fb.group.posts": extractGroupPosts,
    "fb.group.search_posts": extractGroupSearchPosts,
    "fb.profile.friends": extractProfileFriends,
    "fb.groups.search": extractGroupsSearch,
    "fb.people.search": extractPeopleSearch,
    "fb.profile.posts": extractProfileTimeline,
    "fb.newsfeed": extractNewsfeed,
    "_discover.deep": discoverDeep
  };

  // Dispatch a capability id to its precise extractor. Returns { available,
  // capability, schema, count, items, ... }; available=false (with a reason)
  // when there is no capture, no extractor, or nothing matched — the caller
  // then falls back to the generic/HTML layers. `inputs` are the job source's
  // inputs (e.g. { debug: true }).
  window.__soloGqlExtractCapability = function (capabilityId, inputs) {
    var out = { available: false, capability: capabilityId || "", count: 0, items: [] };
    try {
      var CAP = window.__soloGql;
      if (!CAP || !Array.isArray(CAP.captures) || !CAP.captures.length) { out.reason = "no_capture"; return out; }
      var fn = CAPABILITY_EXTRACTORS[capabilityId];
      if (!fn) { out.reason = "no_extractor"; return out; }
      var res = fn(CAP.captures, inputs || {});
      res.available = res.count > 0;
      if (!res.available) res.reason = "no_match";
      return res;
    } catch (err) {
      out.error = String(err && err.message ? err.message : err);
      return out;
    }
  };

  // =========================================================================
  // Active pagination (cursor replay). Facebook's search/list screens do NOT
  // reliably load more on passive scroll, so instead of hoping the scroll fires
  // page 2, we take the end_cursor from a captured response and REPLAY the same
  // persisted query (reusing its doc_id + variables + fb_dtsg + av) to pull the
  // next pages directly. Each replayed page is run through the SAME capability
  // extractor, so every screen's filtering/mapping is reused for free.
  // =========================================================================

  // Per-capability: how to find a replayable seed (query-name scope) and where
  // the connection's page_info lives.
  var CAPABILITY_PAGINATION = {
    "fb.group.posts":        { scope: "GroupsCometFeed",    pageInfoPath: "data.node.group_feed.page_info" },
    "fb.group.search_posts": { scope: "SearchComet",        pageInfoPath: "data.serpResponse.results.page_info" },
    "fb.groups.search":      { scope: "SearchComet",        pageInfoPath: "data.serpResponse.results.page_info" },
    "fb.people.search":      { scope: "SearchComet",        pageInfoPath: "data.serpResponse.results.page_info" },
    "fb.profile.posts":      { scope: "ProfileCometTimeline", pageInfoPath: "data.node.timeline_list_feed_units.page_info" },
    "fb.profile.friends":    { scope: "Friends",            pageInfoPath: "data.node.pageItems.page_info" },
    "fb.newsfeed":           { scope: "CometNewsFeed",       pageInfoPath: "data.viewer.news_feed.page_info" }
  };

  function firstChunkOf(cap) {
    if (!cap || !cap.response) return null;
    var chunk = Array.isArray(cap.response) ? cap.response[0] : cap.response;
    return isObj(chunk) ? chunk : null;
  }

  function wait(ms) { return new Promise(function (r) { setTimeout(r, ms); }); }

  // Replay one persisted query with a new cursor via the pristine fetch.
  function replayPage(store, cap, cursor) {
    var vars = {};
    for (var k in cap.variables) vars[k] = cap.variables[k];
    vars.cursor = cursor;
    var p = new URLSearchParams();
    p.set("av", cap.av || "");
    p.set("__a", "1");
    p.set("fb_dtsg", cap.fbDtsg || "");
    p.set("fb_api_caller_class", "RelayModern");
    p.set("fb_api_req_friendly_name", cap.queryName || "");
    p.set("variables", JSON.stringify(vars));
    p.set("doc_id", cap.docId);
    p.set("server_timestamps", "true");
    return store.origFetch(cap.url || "/api/graphql/", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded", "X-FB-Friendly-Name": cap.queryName || "" },
      body: p.toString()
    }).then(function (resp) { return resp.text(); }).then(function (text) {
      return store.parseResponse ? store.parseResponse(text) : JSON.parse(String(text).replace(/^for\s*\(;;\);/, ""));
    });
  }

  // ---- DOM-based capabilities (data not available via GraphQL) ------------
  // Facebook Reels expose video URLs via GraphQL, but the creator + caption +
  // hashtags render from a per-reel module and only exist in the DOM once a reel
  // is displayed. So fb.reels.feed is DOM-driven: advance the player and scrape
  // each visible reel card. Yields ReelRecord[] { reel_id, reel_url, creator,
  // caption, hashtags } for industry filtering by the agent.
  function absUrl(href) {
    try { return new URL(href, location.origin).href; } catch (e) { return String(href || ""); }
  }
  function advanceReel() {
    try {
      var o = { key: "ArrowDown", code: "ArrowDown", keyCode: 40, which: 40, bubbles: true, cancelable: true };
      document.dispatchEvent(new KeyboardEvent("keydown", o));
      document.dispatchEvent(new KeyboardEvent("keyup", o));
    } catch (e) { /* ignore */ }
    try { window.scrollBy(0, Math.round((window.innerHeight || 700) * 0.92)); } catch (e) { /* ignore */ }
  }
  // Read the CURRENTLY displayed reel. The active reel's id is the page URL; its
  // creator is the on-screen a[href*="reels_tab"] link nearest the viewport centre
  // (Facebook tags reel-author profile links with sk=reels_tab); hashtags are
  // /hashtag/ links; caption is the overlay text around the creator link.
  function currentReel() {
    var id = (location.pathname.match(/\/reel\/(\d+)/) || [])[1] || "";
    var creatorLinks = document.querySelectorAll('a[href*="reels_tab"]');
    var cl = null, bestDist = 1e9, cy = (window.innerHeight || 800) / 2;
    for (var i = 0; i < creatorLinks.length; i++) {
      var r = creatorLinks[i].getBoundingClientRect();
      if (r.width > 0 && r.height > 0 && (creatorLinks[i].innerText || "").trim().length > 1) {
        var dist = Math.abs((r.top + r.bottom) / 2 - cy);
        if (dist < bestDist) { bestDist = dist; cl = creatorLinks[i]; }
      }
    }
    var creator = null;
    if (cl) {
      var name = (cl.innerText || "").replace(/\s+/g, " ").trim().replace(/^Follow\s+/i, "");
      creator = { name: name, url: absUrl(cl.getAttribute("href") || "") };
    }
    // Caption = the TIGHTEST text block that still carries the reel's hashtags
    // (that is the caption line). Falls back to the creator-overlay text for a
    // reel with no hashtags.
    var caption = "";
    var capHs = document.querySelectorAll('a[href*="/hashtag/"]');
    for (var ci = 0; ci < capHs.length && ci < 6; ci++) {
      var cnode = capHs[ci];
      for (var cu = 0; cu < 8 && cnode.parentElement; cu++) {
        cnode = cnode.parentElement;
        var ct = (cnode.innerText || "").replace(/\s+/g, " ").trim();
        if (ct.length >= 20 && ct.length <= 500 && ct.indexOf("#") > -1) { if (!caption || ct.length < caption.length) caption = ct; }
      }
    }
    if (!caption && cl) {
      var fnode = cl, fbest = cl;
      for (var fu = 0; fu < 8 && fnode.parentElement; fu++) { fnode = fnode.parentElement; if ((fnode.innerText || "").length > 40) { fbest = fnode; break; } }
      caption = (fbest.innerText || "").replace(/\s+/g, " ").trim();
    }
    caption = caption.slice(0, 1000);
    var tags = [], seenTag = {};
    var htags = document.querySelectorAll('a[href*="/hashtag/"]');
    for (var h = 0; h < htags.length; h++) {
      var tt = (htags[h].innerText || "").trim();
      if (/^#/.test(tt) && !seenTag[tt.toLowerCase()]) { seenTag[tt.toLowerCase()] = 1; tags.push(tt); }
    }
    if (!id && !creator) return null;
    return { reel_id: id, reel_url: id ? absUrl("/reel/" + id) : location.href, creator: creator, caption: caption, hashtags: tags.slice(0, 30) };
  }
  function reelsCollect(inputs) {
    inputs = inputs || {};
    var maxSteps = Math.max(1, Math.min(80, inputs.max_reels || 20));
    var records = [], seen = {}, dbg = [];
    function scan() {
      var reel = currentReel();
      if (dbg.length < 3) dbg.push({ id: reel && reel.reel_id, creator: reel && reel.creator && reel.creator.name, caption: reel && (reel.caption || "").slice(0, 80) });
      if (!reel || !reel.reel_id || seen[reel.reel_id]) return;
      if (!reel.creator && !reel.caption) return;
      seen[reel.reel_id] = 1;
      records.push(reel);
    }
    scan();
    var step = 0;
    function loop() {
      if (step >= maxSteps) return Promise.resolve();
      step++;
      advanceReel();
      return wait(900).then(function () { scan(); return loop(); });
    }
    return loop().then(function () {
      var out = { capability: "fb.reels.feed", schema: "ReelRecord[]", available: true, count: records.length, items: records, steps: step };
      if (inputs.debug) out._debug = { reels_tab_links: document.querySelectorAll('a[href*="reels_tab"]').length, hashtag_links: document.querySelectorAll('a[href*="/hashtag/"]').length, samples: dbg };
      return out;
    });
  }
  // web.search — parse a DuckDuckGo HTML SERP (html.duckduckgo.com/html/?q=...)
  // into WebResult[] { title, url, display_url, snippet, is_ad }. DDG HTML is
  // chosen for clean markup + low bot-detection; the real destination url is in
  // the .result__a href's `uddg` param (url-encoded); sponsored rows (bing
  // aclick / y.js) are dropped unless inputs.include_ads. Used for off-Facebook
  // enrichment (find a person/email's public web presence).
  function webSearch(inputs) {
    inputs = inputs || {};
    var includeAds = !!inputs.include_ads;
    function clean(e) { return e ? (e.innerText || "").replace(/\s+/g, " ").trim() : ""; }
    function scrape() {
      var results = document.querySelectorAll(".result, .web-result");
      var items = [], seen = {};
      for (var i = 0; i < results.length; i++) {
        var r = results[i];
        var a = r.querySelector(".result__a");
        if (!a) continue;
        var href = a.getAttribute("href") || "";
        var real = href, m = href.match(/[?&]uddg=([^&]+)/);
        if (m) { try { real = decodeURIComponent(m[1]); } catch (e) { /* keep raw */ } }
        var isAd = /[?&](ad_provider|ad_domain)=/.test(href) || /\/y\.js(\?|$)/.test(real) || /bing\.com\/aclick/.test(real);
        if (isAd && !includeAds) continue;
        var title = clean(a);
        if (!title && !real) continue;
        var key = (isAd ? "" : real) || title;
        if (seen[key]) continue;
        seen[key] = 1;
        items.push({
          title: title.slice(0, 300),
          url: (isAd ? "" : String(real)).slice(0, 600),
          display_url: clean(r.querySelector(".result__url")).slice(0, 300),
          snippet: clean(r.querySelector(".result__snippet")).slice(0, 500),
          is_ad: isAd
        });
      }
      return items;
    }
    function pack(items) {
      var q = "";
      try { q = new URLSearchParams(location.search).get("q") || ""; } catch (e) { /* ignore */ }
      return { capability: "web.search", schema: "WebResult[]", provider: "duckduckgo-html", query: q, available: items.length > 0, count: items.length, items: items };
    }
    var first = scrape();
    if (first.length) return Promise.resolve(pack(first));
    // SERP may render a touch late — one short retry.
    return wait(1200).then(function () { return pack(scrape()); });
  }

  var DOM_CAPABILITIES = { "fb.reels.feed": reelsCollect, "web.search": webSearch };

  // Extract page-1 from natural captures, then replay forward until has_next_page
  // is false or max_pages is reached. inputs.max_pages (default 8, cap 40).
  window.__soloGqlPaginate = function (capabilityId, inputs) {
    inputs = inputs || {};
    if (DOM_CAPABILITIES[capabilityId]) { try { return DOM_CAPABILITIES[capabilityId](inputs); } catch (e) { return Promise.resolve({ capability: capabilityId, available: false, count: 0, items: [], error: String(e && e.message || e) }); } }
    var base = window.__soloGqlExtractCapability(capabilityId, inputs);
    var cfg = CAPABILITY_PAGINATION[capabilityId];
    var store = window.__soloGql;
    var maxPages = inputs.max_pages != null ? inputs.max_pages : 8;
    maxPages = Math.max(0, Math.min(40, maxPages));
    if (!cfg || maxPages <= 0 || !store || typeof store.origFetch !== "function" || !base || !Array.isArray(base.items)) {
      return Promise.resolve(base);
    }
    // Find the newest capture that (a) matches the query scope, (b) has a
    // replayable identity (docId + fb_dtsg), and (c) exposes this page_info.
    var caps = store.captures || [];
    var seed = null, seedChunk = null;
    for (var i = caps.length - 1; i >= 0; i--) {
      var c = caps[i];
      if (!c || !c.docId || !c.fbDtsg || !c.response) continue;
      if (cfg.scope && String(c.queryName || "").indexOf(cfg.scope) === -1) continue;
      var chunk = firstChunkOf(c);
      if (chunk && getPath(chunk, cfg.pageInfoPath)) { seed = c; seedChunk = chunk; break; }
    }
    if (!seed) return Promise.resolve(base);

    var items = base.items.slice();
    var seen = {};
    items.forEach(function (it) { if (it && it.id) seen[it.id] = 1; });
    var pi = getPath(seedChunk, cfg.pageInfoPath) || {};
    var state = { cursor: pi.end_cursor, hasNext: !!pi.has_next_page, pages: 0, added: 0 };

    function step() {
      if (!state.hasNext || !state.cursor || state.pages >= maxPages) return Promise.resolve();
      state.pages += 1;
      return replayPage(store, seed, state.cursor).then(function (resp) {
        if (!resp) { state.hasNext = false; return; }
        var fakeCap = { queryName: seed.queryName, docId: seed.docId, variables: seed.variables, response: resp };
        var page = CAPABILITY_EXTRACTORS[capabilityId]([fakeCap], {});
        (page && page.items || []).forEach(function (it) {
          if (it && it.id && !seen[it.id]) { seen[it.id] = 1; items.push(it); state.added += 1; }
        });
        var chunk = Array.isArray(resp) ? resp[0] : resp;
        var pinfo = (isObj(chunk) ? getPath(chunk, cfg.pageInfoPath) : null) || {};
        state.cursor = pinfo.end_cursor;
        state.hasNext = !!pinfo.has_next_page;
        return wait(400 + Math.floor((state.pages % 3) * 150)).then(step); // gentle pacing
      }).catch(function () { state.hasNext = false; });
    }

    return step().then(function () {
      base.items = items;
      base.count = items.length;
      base.paginated = true;
      base.pages_fetched = state.pages;
      base.added_by_pagination = state.added;
      base.available = base.count > 0;
      return base;
    });
  };
})();
