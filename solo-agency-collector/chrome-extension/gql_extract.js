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
  // The UFI node — Facebook's "feedback target" — carries a story's reaction/comment counts
  // AND its feedback id. That id is the ONLY input every comment query takes, so locating this
  // node once serves both purposes. Two nestings are in the wild (with and without
  // comet_feed_ufi_container) and the SERP nests it differently again, so try the measured
  // paths first and fall back to a bounded search for the key by name.
  var UFI_PATHS = [
    "feedback.story.story_ufi_container.story.feedback_context.feedback_target_with_context",
    "feedback.story.comet_feed_ufi_container.story.story_ufi_container.story.feedback_context.feedback_target_with_context"
  ];
  function storyFeedbackTarget(node) {
    if (!isObj(node)) return null;
    var cs = node.comet_sections;
    if (isObj(cs)) {
      for (var i = 0; i < UFI_PATHS.length; i++) {
        var hit = getPath(cs, UFI_PATHS[i]);
        if (isObj(hit)) return hit;
      }
      var inCs = deepFind(cs, { feedback_target_with_context: 1 }, isObj, 0, 16);
      if (isObj(inCs)) return inCs;
    }
    // Deliberately NOT a whole-node fallback. A reshared post nests the original story, whose
    // own feedback_target_with_context would then be bound to the WRAPPER — reporting the inner
    // post's reaction and comment counts as the outer post's. The old code searched only
    // comet_sections and returned null here; keep that.
    return null;
  }

  // A story's feedback id — the ONLY input the comment queries take.
  //
  // These two paths, in this order, are what a working third-party extension reads, measured
  // out of its source. The obvious-looking candidate — feedback_target_with_context.id, the UFI
  // node this file already locates for engagement counts — is a DIFFERENT node and its id is not
  // interchangeable, so it is tried last and only as a rescue. Search results run through the
  // same normalizer as feed posts there, so one lookup serves both screens.
  var FEEDBACK_ID_PATHS = ["comet_sections.content.story.feedback.id", "feedback.id"];
  function storyFeedbackId(node) {
    if (!isObj(node)) return "";
    for (var i = 0; i < FEEDBACK_ID_PATHS.length; i++) {
      var v = getPath(node, FEEDBACK_ID_PATHS[i]);
      if (typeof v === "string" && v) return v;
    }
    var fb = deepFind(node, { feedback: 1 }, function (x) {
      return isObj(x) && typeof x.id === "string" && x.id;
    }, 0, 14);
    if (isObj(fb) && fb.id) return String(fb.id);
    var ufi = storyFeedbackTarget(node);
    return isObj(ufi) && ufi.id ? String(ufi.id) : "";
  }

  // The comment-ordering token is NOT a constant to be invented. Facebook publishes the tokens a
  // given story ACCEPTS in selectable_intents, and the reference implementation picks the first
  // whose token contains "unfiltered" — i.e. show every comment, not the ranked/filtered subset.
  // Guessing a plausible-looking literal here would be replaced by the server's own idea of a
  // default, silently returning a filtered subset that looks like the whole thread.
  var INTENT_PATHS = [
    "comet_sections.feedback.story.comet_feed_ufi_container.story.story_ufi_container.story.feedback_context.feedback_target_with_context.comment_list_renderer.feedback.comment_rendering_instance_for_feed_location.selectable_intents",
    "comet_sections.feedback.story.story_ufi_container.story.feedback_context.feedback_target_with_context.comment_list_renderer.feedback.comment_rendering_instance_for_feed_location.selectable_intents"
  ];
  function storyCommentIntent(node) {
    if (!isObj(node)) return "";
    var list = null;
    for (var i = 0; i < INTENT_PATHS.length && !list; i++) {
      var v = getPath(node, INTENT_PATHS[i]);
      if (Array.isArray(v) && v.length) list = v;
    }
    if (!list) {
      var found = deepFind(node, { selectable_intents: 1 }, function (x) {
        return Array.isArray(x) && x.length;
      }, 0, 18);
      if (Array.isArray(found)) list = found;
    }
    if (!list) return "";
    for (var j = 0; j < list.length; j++) {
      var t = list[j] && list[j].intent_token;
      if (typeof t === "string" && t.toLowerCase().indexOf("unfiltered") > -1) return t;
    }
    return "";
  }

  function postEngagement(node) {
    var ufi = storyFeedbackTarget(node);
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
  // An ANONYMOUS group post has its top-level `story.actors` scrubbed — that scrubbing is
  // what makes the UI read "Anonymous member". Facebook still ships the REAL actor, numeric
  // id and all, further down inside the avatar renderer's own story, and does not redact it
  // there. Reading only the obvious path produced a record with no author at all for every
  // anonymous post; reading this one too recovers it. A post is never rejected for lacking an
  // actor either way — an author is a field, not a precondition.
  var ACTOR_FALLBACK_PATH = "comet_sections.context_layout.story.comet_sections.actor_photo.story.actors";
  function storyActor(node) {
    if (Array.isArray(node.actors) && isObj(node.actors[0])) return node.actors[0];
    var deep = getPath(node, ACTOR_FALLBACK_PATH);
    if (Array.isArray(deep) && isObj(deep[0])) return deep[0];
    return null;
  }

  function postRecordFromStoryNode(node) {
    // Accept a story identified by EITHER id: the pagination reply keys some stories only by
    // post_id, and gating on the story id alone silently dropped them.
    if (!isObj(node) || (!node.id && !node.post_id)) return null;
    var actor = storyActor(node);
    return {
      id: String(node.id || node.post_id),
      post_id: node.post_id ? String(node.post_id) : "",
      url: (typeof node.permalink_url === "string" && node.permalink_url)
        ? node.permalink_url
        : (firstString(node, { wwwURL: 1, url: 1, permalink_url: 1, permalink: 1 }) || ""),
      actor: actorRef(actor),
      text: (deepText(node.comet_sections, 0) || deepText(node, 0) || "").slice(0, 4000),
      created_time: firstNumber(node, { creation_time: 1, created_time: 1, publish_time: 1 }) || 0,
      attachments: postAttachments(node),
      engagement: postEngagement(node),
      // The handle fb.post.comments takes. Carried on every PostRecord so a caller that already
      // has a search result never has to re-open the post to ask for its comments — together
      // with the ordering token THIS story published, which is per-story and cannot be guessed.
      feedback_id: storyFeedbackId(node),
      comment_intent: storyCommentIntent(node),
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
          // Key on whichever id the story carries. Gating on the story id alone dropped
          // stories that arrive keyed only by post_id, and the record builder already
          // refuses a node that has neither.
          var rec = isObj(node) ? postRecordFromStoryNode(node) : null;
          if (!rec || seen[rec.id]) continue;
          seen[rec.id] = 1;
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
          var actor = storyActor(node);   // top-level actors, then the un-redacted avatar path
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
            // Same two handles postRecordFromStoryNode emits. The catalog promises every
            // PostRecord carries them; these two extractors build their records inline, so
            // without this the promise is false exactly here and fb.post.comments would refuse
            // for a timeline post while working for the identical post found via group search.
            feedback_id: storyFeedbackId(node),
            comment_intent: storyCommentIntent(node),
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
          var actor = storyActor(node);   // top-level actors, then the un-redacted avatar path
          if (!actor && isObj(contentStory)) actor = storyActor(contentStory);

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
            // See the note on the same two fields in extractProfileTimeline.
            feedback_id: storyFeedbackId(node),
            comment_intent: storyCommentIntent(node),
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

  // fb.profile.videos — a page's Videos tab (PagesCometChannelTabAllVideosCard…
  // PaginationQuery). Per video: title, VIEW count (play_count), reactions,
  // caption, url. This is the "your video X with 77K views / 52 shares" data the
  // outreach emails open with. Paginated (data.node.all_videos.page_info).
  function extractProfileVideos(caps, opts) {
    opts = opts || {};
    var items = [], seen = {}, sourceQuery = "", firstNode = null;
    for (var c = 0; c < caps.length; c++) {
      var cap = caps[c];
      if (!cap) continue;
      if (String(cap.queryName || "").indexOf("AllVideos") === -1) continue;
      var chunks = Array.isArray(cap.response) ? cap.response : (cap.response ? [cap.response] : []);
      for (var k = 0; k < chunks.length; k++) {
        var edges = getPath(chunks[k], "data.node.all_videos.edges");
        if (!Array.isArray(edges)) continue;
        sourceQuery = cap.queryName || sourceQuery;
        for (var i = 0; i < edges.length; i++) {
          var v = getPath(edges[i], "node.channel_tab_thumbnail_renderer.video");
          if (!isObj(v)) continue;
          var id = v.id ? String(v.id) : "";
          if (!id) continue;
          if (seen[id]) continue;
          seen[id] = 1;
          if (!firstNode) firstNode = v;
          var vcaption = (deepText(v.creation_story, 0) || "").slice(0, 1000);
          var title = firstString(v, { savable_title: 1 }) || "";
          if (!title) {
            var vt = deepFind(v, { video_title: 1 }, function (x) { return (typeof x === "string" && x.trim()) || (isObj(x) && typeof x.text === "string"); }, 0, 12);
            if (typeof vt === "string") title = vt; else if (isObj(vt)) title = vt.text;
          }
          if (!title) title = vcaption.slice(0, 120);
          var views = coerceCount(v.play_count);
          if (views === null) views = coerceCount(v.post_play_count);
          var fb = isObj(v.feedback) ? v.feedback : null;
          var reactions = fb ? coerceCount(fb.reaction_count) : null;
          if (reactions === null && fb) reactions = coerceCount(fb.i18n_reaction_count);
          var shares = coerceCount(deepFind(v, { share_count: 1, i18n_share_count: 1, reshare_count: 1 }, function (x) { return coerceCount(x) !== null; }, 0, 12));
          var acts = getPath(v, "creation_story.actors");
          items.push({
            id: id,
            title: title ? String(title).slice(0, 300) : "",
            views: views,
            reactions: reactions,
            shares: shares,
            caption: vcaption,
            url: (typeof v.permalink_url === "string" && v.permalink_url) ? v.permalink_url : ("https://www.facebook.com/watch/?v=" + id),
            actor: actorRef(Array.isArray(acts) ? acts[0] : (isObj(v.owner) ? v.owner : null))
          });
        }
      }
    }
    var result = { capability: "fb.profile.videos", schema: "VideoRecord[]", source_query: sourceQuery, count: items.length, items: items };
    if (opts.debug && firstNode) { try { result._debug_node_skeleton = skeletonize(firstNode, 0, { n: 1200 }, 16); } catch (e) { /* ignore */ } }
    return result;
  }

  var CAPABILITY_EXTRACTORS = {
    "fb.group.posts": extractGroupPosts,
    "fb.group.search_posts": extractGroupSearchPosts,
    "fb.profile.friends": extractProfileFriends,
    "fb.groups.search": extractGroupsSearch,
    "fb.people.search": extractPeopleSearch,
    "fb.profile.posts": extractProfileTimeline,
    "fb.profile.videos": extractProfileVideos,
    "fb.newsfeed": extractNewsfeed,
    "_discover.deep": discoverDeep
  };

  // Dispatch a capability id to its precise extractor. Returns { available,
  // capability, schema, count, items, ... }; available=false (with a reason)
  // when there is no capture, no extractor, or nothing matched — the caller
  // then falls back to the generic/HTML layers. `inputs` are the job source's
  // inputs (e.g. { debug: true }).
  // --- time window ---------------------------------------------------------
  // Restrict a post listing to a period: `within_days: 90`, or explicit `since`/`until`
  // (ISO date or unix seconds). No window given → everything is kept, unchanged.
  //
  // The filter runs AFTER extraction and never stops pagination early. That is deliberate:
  // group SEARCH results come back ranked by relevance, and even a group FEED sorted by
  // RECENT_ACTIVITY floats an old post the moment someone comments on it — so "I have seen
  // something older than the window, therefore the rest is older" is false on both surfaces.
  // The bound on work is the page cap (`max_pages`), which the caller sets.
  function toEpochSeconds(v) {
    if (v === null || v === undefined || v === "") return 0;
    if (typeof v === "number") return v > 1e11 ? Math.floor(v / 1000) : Math.floor(v);
    var s = String(v).trim();
    if (/^\d+$/.test(s)) { var n = parseInt(s, 10); return n > 1e11 ? Math.floor(n / 1000) : n; }
    var t = Date.parse(s);
    return isNaN(t) ? 0 : Math.floor(t / 1000);
  }
  function timeWindowFrom(inputs) {
    inputs = inputs || {};
    var since = toEpochSeconds(inputs.since);
    var until = toEpochSeconds(inputs.until);
    var days = Number(inputs.within_days);
    if (!since && Number.isFinite(days) && days > 0) since = Math.floor(Date.now() / 1000) - Math.floor(days * 86400);
    if (!since && !until) return null;
    return { since: since || 0, until: until || 0, within_days: (Number.isFinite(days) && days > 0) ? days : null };
  }
  function applyTimeWindow(res, inputs) {
    if (!res || !Array.isArray(res.items)) return res;
    var win = timeWindowFrom(inputs);
    if (!win) return res;
    // A post with no readable timestamp cannot be judged against the window. Dropping it
    // silently would hide real posts; keeping it silently would let a five-year-old post into
    // a "last 90 days" run and get commented on. So: excluded by default, and COUNTED, with
    // include_undated:true to override. Never invisible either way.
    var keepUndated = inputs && inputs.include_undated === true;
    var kept = [], older = 0, newer = 0, undated = 0;
    for (var i = 0; i < res.items.length; i++) {
      var it = res.items[i];
      var t = Number(it && it.created_time) || 0;
      if (!t) { undated += 1; if (keepUndated) kept.push(it); continue; }
      if (win.since && t < win.since) { older += 1; continue; }
      if (win.until && t > win.until) { newer += 1; continue; }
      kept.push(it);
    }
    res.items = kept;
    res.count = kept.length;
    res.available = kept.length > 0;
    res.time_window = {
      since: win.since || null, until: win.until || null, within_days: win.within_days,
      excluded_older: older, excluded_newer: newer,
      undated: undated, undated_kept: !!keepUndated
    };
    return res;
  }

  window.__soloGqlExtractCapability = function (capabilityId, inputs) {
    // available means THE CAPABILITY RAN, never "it found something". background.js nulls a record
    // whose capability reports unavailable, so tying the flag to the row count made "the feed never
    // rendered", "there is no capture for this query", "the extractor is missing" and "the code
    // threw" produce the identical output: records:null, and nothing to tell them apart.
    //
    // Measured: a run of every read capability in hidden tabs returned records:null for seven of
    // them — group.posts, group.search_posts, groups.search, newsfeed, people.search,
    // profile.friends, profile.videos — all of which had simply never seen a capture, because a
    // hidden tab does not render a feed. That is a diagnosable fact and it was thrown away.
    //
    // Emptiness travels in `found`, `count`, `items` and `reason`. This is the same contract the
    // DOM capabilities were fixed to three times over; the GraphQL path still had the old one.
    var out = { available: true, found: false, capability: capabilityId || "", count: 0, items: [] };
    try {
      var CAP = window.__soloGql;
      if (!CAP || !Array.isArray(CAP.captures) || !CAP.captures.length) { out.reason = "no_capture"; return out; }
      var fn = CAPABILITY_EXTRACTORS[capabilityId];
      if (!fn) { out.reason = "no_extractor"; return out; }
      // Facebook streams these replies: a capture's later chunks are @defer payloads that
      // belong INSIDE chunk 0. Merge before extracting, or the story content is invisible and
      // only the skeleton is read.
      var streamMerged = 0;
      var mergedCaps = CAP.captures.map(function (c) {
        if (!c || !c.response) return c;
        var mm = mergeStreamed(c.response);
        streamMerged += mm.merged;
        if (!mm.merged) return c;
        var copy = {};
        for (var k in c) copy[k] = c[k];
        copy.response = mm.chunks;
        return copy;
      });
      var res = fn(mergedCaps, inputs || {});
      res.stream_chunks_merged = streamMerged;
      res.available = true;
      res.found = res.count > 0;
      res = applyTimeWindow(res, inputs);
      if (!res.found) res.reason = res.reason || "no_match";
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
    "fb.profile.videos":     { scope: "AllVideos",           pageInfoPath: "data.node.all_videos.page_info" },
    "fb.profile.friends":    { scope: "Friends",            pageInfoPath: "data.node.pageItems.page_info" },
    "fb.newsfeed":           { scope: "CometNewsFeed",       pageInfoPath: "data.viewer.news_feed.page_info" }
  };

  function firstChunkOf(cap) {
    if (!cap || !cap.response) return null;
    var chunk = Array.isArray(cap.response) ? cap.response[0] : cap.response;
    return isObj(chunk) ? chunk : null;
  }

  function wait(ms) { return new Promise(function (r) { setTimeout(r, ms); }); }

  // --- cursor discovery ----------------------------------------------------
  // CAPABILITY_PAGINATION hard-codes one page_info path per capability, taken from the
  // INITIAL feed query. The PAGINATION query for the same feed answers in a different shape:
  // measured on a live group, the config expected data.node.group_feed.page_info while the
  // cursor actually sat at data.page_info. The path was never found, so no feed ever
  // paginated — and the only symptom was a quietly short list. Treat the configured path as a
  // hint: try it, then search the payload. Hard-coding a second path just queues up the same
  // failure for the next time Facebook moves it.
  function chunksOf(response) {
    return Array.isArray(response) ? response : (response ? [response] : []);
  }
  function findInChunks(response, path) {
    var chunks = chunksOf(response);
    for (var i = 0; i < chunks.length; i++) {
      var hit = isObj(chunks[i]) ? getPath(chunks[i], path) : null;
      if (isObj(hit)) return hit;
    }
    return null;
  }
  function deepFindPageInfo(response) {
    var chunks = chunksOf(response), found = null;
    function walk(node, depth) {
      if (found || depth > 9 || !isObj(node)) return;
      for (var k in node) {
        if (found) return;
        var v = node[k];
        if (k === "page_info" && isObj(v) && ("end_cursor" in v || "has_next_page" in v)) { found = v; return; }
        if (isObj(v)) walk(v, depth + 1);
        else if (Array.isArray(v) && v.length && isObj(v[0])) walk(v[0], depth + 1);
      }
    }
    for (var i = 0; i < chunks.length && !found; i++) walk(chunks[i], 0);
    return found;
  }
  function resolvePageInfo(response, path) {
    return findInChunks(response, path) || deepFindPageInfo(response);
  }

  // Facebook STREAMS this response. With stream_initial_count / COMET_STREAM, chunk 0 is a
  // skeleton — its first edge is a "GroupsSectionHeaderUnit" titled "Recent activity", not a
  // post — and the real story content arrives in later chunks as @defer payloads, each
  // carrying a `path` telling you where it belongs in that skeleton. Reading the chunks
  // SEPARATELY, which is what this file did, sees the skeleton and never the stories: the
  // head fetch looked like it "returned nothing new" when in fact its content was sitting
  // unmerged in chunks 1..4.
  //
  // Merging is additive and non-destructive: the merged root is prepended and the original
  // chunks are kept, so anything an extractor used to find it still finds.
  function mergeStreamed(response) {
    var chunks = Array.isArray(response) ? response : (response ? [response] : []);
    if (chunks.length < 2 || !isObj(chunks[0])) return { chunks: chunks, merged: 0, paths: [] };
    var root = chunks[0], merged = 0, paths = [];
    for (var i = 1; i < chunks.length; i++) {
      var c = chunks[i];
      if (!isObj(c) || !Array.isArray(c.path) || !c.path.length || c.data === undefined) continue;
      // Walk to the PARENT of the target, never to the target itself. A @stream payload
      // addresses a slot that does not exist yet — "node.group_feed.edges.1" when edges holds
      // a single element — so resolving the full path lands on undefined and the payload gets
      // silently discarded. That is exactly what happened: the anonymous post and Post 4 both
      // arrived in stream chunks and were dropped on the floor for being "unmergeable".
      var parent = root.data, ok = true;
      for (var k = 0; k < c.path.length - 1; k++) {
        parent = parent ? parent[c.path[k]] : null;
        if (parent === null || parent === undefined || typeof parent !== "object") { ok = false; break; }
      }
      if (!ok) continue;
      var last = c.path[c.path.length - 1];
      var existing = parent[last];
      // @defer refines a node that is already there (merge its keys); @stream delivers a new
      // element (assign it into the slot).
      if (isObj(existing) && isObj(c.data)) { for (var key in c.data) existing[key] = c.data[key]; }
      else { parent[last] = c.data; }
      merged += 1;
      if (paths.length < 8) paths.push(c.path.join("."));
    }
    return { chunks: [root].concat(chunks.slice(1)), merged: merged, paths: paths };
  }

  // A reply can carry MANY `edges` arrays — the first one encountered was the comment-sort
  // dropdown, which produced a phantom "Most relevant" record with no url. Never pick by name:
  // collect the candidates and let the capability's OWN extractor decide which is real.
  function collectEdgeArrays(response, limit) {
    var out = [], chunks = chunksOf(response);
    function walk(node, depth) {
      if (out.length >= limit || depth > 9 || !isObj(node)) return;
      for (var k in node) {
        if (out.length >= limit) return;
        var v = node[k];
        if (k === "edges" && Array.isArray(v) && v.length) out.push(v);
        else if (isObj(v)) walk(v, depth + 1);
        else if (Array.isArray(v) && v.length && isObj(v[0])) walk(v[0], depth + 1);
      }
    }
    for (var i = 0; i < chunks.length; i++) walk(chunks[i], 0);
    return out;
  }
  function shapeAt(path, edges) {
    var shell = {}, cur = shell, parts = String(path).split(".");
    for (var p = 0; p < parts.length - 1; p++) { cur[parts[p]] = {}; cur = cur[parts[p]]; }
    cur[parts[parts.length - 1]] = edges;
    return [shell];
  }
  function extractReplayItems(resp, capabilityId, seed, pageInfoPath) {
    var extractor = CAPABILITY_EXTRACTORS[capabilityId];
    if (!extractor) return { items: [], via: "no_extractor" };
    var m = mergeStreamed(resp);
    resp = m.chunks;
    function run(response) {
      var r = extractor([{ queryName: seed.queryName, docId: seed.docId, variables: seed.variables, response: response }], {});
      return (r && Array.isArray(r.items)) ? r.items : [];
    }
    var direct = run(resp);
    if (direct.length) return { items: direct, via: "native_shape+merged" + m.merged };
    var edgesPath = String(pageInfoPath || "").replace(/\.page_info$/, ".edges");
    if (!edgesPath) return { items: [], via: "no_edges_path" };
    var cands = collectEdgeArrays(resp, 12);
    for (var i = 0; i < cands.length; i++) {
      var items = run(shapeAt(edgesPath, cands[i]));
      if (items.length) return { items: items, via: "reshaped_" + i + "_of_" + cands.length };
    }
    return { items: [], via: "no_candidate_matched_of_" + cands.length };
  }

  // Replay one persisted query with a new cursor via the pristine fetch.
  // Ask Facebook's own module registry for the query artifact, which is how a working
  // third-party extension does it: the doc_id is then always the CURRENT one the page itself
  // would use, and can never go stale the way a captured value can.
  function docIdFromRegistry(queryName) {
    try {
      if (typeof window.require !== "function" || !queryName) return "";
      var mod = window.require(String(queryName) + ".graphql");
      var id = mod && mod.params && mod.params.id;
      return id ? String(id) : "";
    } catch (e) { return ""; }
  }

  // Relay declares its own `__relay_internal__pv__*` provider variables on the compiled
  // artifact and resolves them itself at request time. A hand-built fetch does not get that
  // for free: a persisted query whose providers are missing is REJECTED outright ("Variable
  // ... of required type Boolean! was not provided"), which reads downstream as an empty
  // result rather than a malformed request. Read them from the same artifact the doc_id comes
  // from, so the set is always the one this build of Facebook actually declares — the
  // hardcoded list in FEED_VARS goes stale the moment they add a flag.
  function providedVariables(queryName) {
    var out = {};
    try {
      if (typeof window.require !== "function" || !queryName) return out;
      var mod = window.require(String(queryName) + ".graphql");
      var pv = mod && mod.params && mod.params.providedVariables;
      if (!isObj(pv)) return out;
      for (var k in pv) {
        try {
          var val = (pv[k] && typeof pv[k].get === "function") ? pv[k].get() : undefined;
          // undefined is not a value a persisted query accepts, and a provider entry with no
          // .get() is not one we know how to resolve — in both cases leave the key out and let
          // an explicit variable or the server's own default stand.
          if (val !== undefined) out[k] = val;
        } catch (e) { /* one unresolvable provider must not lose the rest */ }
      }
    } catch (e) { /* module not loaded on this screen */ }
    return out;
  }

  // The variable set a working third-party extension sends for this query, reproduced
  // verbatim. Inheriting Facebook's own captured variables was not enough: those describe the
  // slice its UI happened to want, and replaying them — even with the cursor removed — walked
  // the same middle of the feed and returned 3 of 6 posts. Driving the query with a known-good
  // set is what makes it answer from the top.
  //
  // count is deliberately SMALL: the reference implementation asks for 3 at a time and
  // recurses, which is also gentler on the account than one large sweep.
  function webPixelRatio() {
    try { return window.require("WebPixelRatio").get(); } catch (e) { return 1; }
  }
  var FEED_VARS = {
    "fb.group.posts": function (id, cursor) {
      var v = {
        count: 3,
        feedLocation: "GROUP",
        feedType: "DISCUSSION",
        feedbackSource: 0,
        focusCommentID: null,
        privacySelectorRenderLocation: "COMET_STREAM",
        renderLocation: "group",
        scale: webPixelRatio(),
        sortingSetting: "RECENT_ACTIVITY",
        stream_initial_count: 1,
        useDefaultActor: false,
        useGroupFeedWithEntQL_EXPERIMENTAL: false,
        id: id,
        __relay_internal__pv__GHLShouldChangeAdIdFieldNamerelayprovider: true,
        __relay_internal__pv__CometImmersivePhotoCanUserDisable3DMotionrelayprovider: false,
        __relay_internal__pv__IsWorkUserrelayprovider: false,
        __relay_internal__pv__IsMergQAPollsrelayprovider: false,
        __relay_internal__pv__FBReelsMediaFooter_comet_enable_reels_ads_gkrelayprovider: false,
        __relay_internal__pv__CometUFIReactionsEnableShortNamerelayprovider: false,
        __relay_internal__pv__CometUFIShareActionMigrationrelayprovider: true,
        __relay_internal__pv__IncludeCommentWithAttachmentrelayprovider: true,
        __relay_internal__pv__StoriesArmadilloReplyEnabledrelayprovider: true,
        __relay_internal__pv__EventCometCardImage_prefetchEventImagerelayprovider: false,
        __relay_internal__pv__CometUFIReactionEnableShortNamerelayprovider: true
      };
      if (cursor) v.cursor = cursor;
      return v;
    }
  };
  // The group id: prefer what Facebook itself sent, fall back to the address bar.
  function feedTargetId(cap) {
    var id = cap && cap.variables && cap.variables.id;
    if (id) return String(id);
    var m = String(location.href).match(/\/groups\/(\d+)/);
    return m ? m[1] : "";
  }

  // cursor === null means "the head of the connection". The key is DELETED rather than sent
  // as null, and that distinction is the whole difference between working and not: sending
  // "cursor": null returned a degenerate slice with a single non-story edge, while omitting
  // the key returns the newest posts. A working third-party extension passes `undefined`
  // here, which JSON.stringify drops — same thing, arrived at by accident on their side.
  function replayPage(store, cap, cursor, capabilityId) {
    // Providers go in FIRST so anything explicit or captured still wins — this only fills the
    // flags the query declares and nobody supplied.
    var vars = providedVariables(cap.queryName);
    var build = FEED_VARS[capabilityId];
    if (build) {
      // Known-good set, built from scratch. Anything Facebook sent that we did not enumerate
      // is carried over underneath it, so an unrecognised provider flag is never lost.
      var explicit = build(feedTargetId(cap), cursor);
      for (var c0 in cap.variables) vars[c0] = cap.variables[c0];
      delete vars.cursor;
      for (var e0 in explicit) vars[e0] = explicit[e0];
    } else {
      for (var k in cap.variables) vars[k] = cap.variables[k];
      // A null cursor means the HEAD of the connection, and the key must be DELETED rather
      // than sent as null — Facebook answers "cursor": null with a degenerate slice.
      if (cursor === null || cursor === undefined) delete vars.cursor; else vars.cursor = cursor;
    }
    var p = new URLSearchParams();
    p.set("av", cap.av || "");
    p.set("__a", "1");
    p.set("fb_dtsg", cap.fbDtsg || "");
    p.set("fb_api_caller_class", "RelayModern");
    p.set("fb_api_req_friendly_name", cap.queryName || "");
    p.set("variables", JSON.stringify(vars));
    p.set("doc_id", docIdFromRegistry(cap.queryName) || cap.docId);
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
  // A profile REELS TAB (/…/reels/) renders a thumbnail GRID, not the immersive
  // one-reel player — each card is an <a href*="/reel/<id>"> with a view-count
  // overlay. currentReel() only works inside /reel/<id>, so on the grid it found
  // nothing (id from the URL was empty). Grid mode scrapes the cards directly.
  function reelIdFromHref(href) { var m = String(href || "").match(/\/reel\/(\d+)/); return m ? m[1] : ""; }
  function gridReels() {
    var out = [], seen = {};
    var links = document.querySelectorAll('a[href*="/reel/"]');
    for (var i = 0; i < links.length; i++) {
      var a = links[i];
      var id = reelIdFromHref(a.getAttribute("href") || a.href || "");
      if (!id || seen[id]) continue;
      seen[id] = 1;
      var txt = (a.innerText || "").replace(/\s+/g, " ").trim();
      // view overlay: a short count token like "1.2M" / "45K" / "1,234" on the card
      var vm = txt.match(/(\d[\d.,]*\s*[KMB])\b/i) || txt.match(/(\d[\d.,]{2,})/);
      // the grid card usually shows ONLY the view count — strip it so caption is
      // the real text if any, else empty (captions live in the player/GraphQL).
      var capText = vm ? txt.replace(vm[1], "").replace(/\s+/g, " ").trim() : txt;
      out.push({
        reel_id: id, reel_url: absUrl("/reel/" + id),
        views: vm ? parseCount(vm[1]) : null, view_text: vm ? vm[1].replace(/\s+/g, "") : "",
        caption: capText.slice(0, 200), creator: null, hashtags: []
      });
    }
    return out;
  }
  function reelsGridDebug() {
    var links = document.querySelectorAll('a[href*="/reel/"]'), samples = [];
    for (var i = 0; i < links.length && samples.length < 6; i++) {
      samples.push({ href: (links[i].getAttribute("href") || "").slice(0, 70), text: (links[i].innerText || "").replace(/\s+/g, " ").trim().slice(0, 90) });
    }
    var gqlNames = [];
    try {
      var caps = (window.__soloGql && window.__soloGql.captures) || [];
      for (var c = Math.max(0, caps.length - 25); c < caps.length; c++) { if (caps[c] && caps[c].queryName) gqlNames.push(caps[c].queryName); }
    } catch (e) { /* ignore */ }
    return { mode: "grid", url: location.href, reel_links: links.length, samples: samples, gql_recent: gqlNames };
  }
  function reelsCollect(inputs) {
    inputs = inputs || {};
    var maxSteps = Math.max(1, Math.min(80, inputs.max_reels || inputs.max_pages || 20));

    // GRID MODE — a reels tab / any page that is not a single /reel/<id> player.
    if (!/\/reel\/\d+/.test(location.pathname)) {
      var grec = [], gseen = {}, gstep = 0, dry = 0;
      function harvest() { var g = gridReels(); for (var i = 0; i < g.length; i++) { if (!gseen[g[i].reel_id]) { gseen[g[i].reel_id] = 1; grec.push(g[i]); } } }
      harvest();
      function gloop() {
        if (gstep >= maxSteps || dry >= 2) return Promise.resolve();
        gstep++;
        try { window.scrollBy(0, Math.round((window.innerHeight || 800) * 0.9)); } catch (e) { /* ignore */ }
        return wait(850).then(function () { var before = grec.length; harvest(); dry = (grec.length === before) ? dry + 1 : 0; return gloop(); });
      }
      return gloop().then(function () {
        var out = { capability: "fb.reels.feed", schema: "ReelRecord[]", available: true, count: grec.length, items: grec, steps: gstep, mode: "grid" };
        if (inputs.debug) out._debug = reelsGridDebug();
        return out;
      });
    }

    // PLAYER MODE — the immersive /reel/<id> viewer; advance and scrape each reel.
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
      var out = { capability: "fb.reels.feed", schema: "ReelRecord[]", available: true, count: records.length, items: records, steps: step, mode: "player" };
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
    function hostOf(u) { var m = String(u).match(/^https?:\/\/([^\/]+)/); return m ? m[1] : ""; }

    // DuckDuckGo HTML (html.duckduckgo.com/html/) — clean, low bot-detection.
    function scrapeDDG() {
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
        if (seen[key]) continue; seen[key] = 1;
        items.push({ title: title.slice(0, 300), url: (isAd ? "" : String(real)).slice(0, 600), display_url: clean(r.querySelector(".result__url")).slice(0, 300), snippet: clean(r.querySelector(".result__snippet")).slice(0, 500), is_ad: isAd });
      }
      return items;
    }

    // Google — an organic result is an external <a href> wrapping an <h3>. Keys
    // off that semantic structure (not Google's churning class names) and skips
    // google/gstatic/ad/redirect links. Richer results, but higher CAPTCHA risk
    // at volume — best for targeted enrichment, not bulk.
    function scrapeGoogle() {
      var anchors = document.querySelectorAll('a[href^="http"]');
      var items = [], seen = {};
      for (var i = 0; i < anchors.length; i++) {
        var a = anchors[i], href = a.getAttribute("href") || "";
        if (/(^|\.)(google|gstatic|googleadservices|googlesyndication)\.|\/aclk|\/url\?|youtube\.com\/results|webcache\./.test(href)) continue;
        var h3 = a.querySelector("h3");
        if (!h3) continue;
        if (seen[href]) continue; seen[href] = 1;
        var title = clean(h3);
        if (!title) continue;
        var c = a;
        for (var up = 0; up < 4 && c.parentElement; up++) c = c.parentElement;
        var snip = clean(c).replace(title, "").replace(/https?:\/\/\S+/g, "").replace(/\s+/g, " ").trim().slice(0, 400);
        items.push({ title: title.slice(0, 300), url: href.slice(0, 600), display_url: hostOf(href), snippet: snip, is_ad: false });
      }
      return items;
    }

    // Bing — organic results in li.b_algo.
    function scrapeBing() {
      var results = document.querySelectorAll("li.b_algo");
      var items = [], seen = {};
      for (var i = 0; i < results.length; i++) {
        var r = results[i], a = r.querySelector("h2 a") || r.querySelector("a[href^='http']");
        if (!a) continue;
        var href = a.getAttribute("href") || "";
        if (!/^https?:/.test(href) || /bing\.com\/aclick/.test(href)) continue;
        if (seen[href]) continue; seen[href] = 1;
        items.push({ title: clean(r.querySelector("h2")).slice(0, 300), url: href.slice(0, 600), display_url: hostOf(href), snippet: clean(r.querySelector(".b_caption p") || r.querySelector("p")).slice(0, 500), is_ad: false });
      }
      return items;
    }

    // Route by the SERP host so the agent just submits the engine's search URL.
    function scrape() {
      var host = location.hostname || "";
      if (/duckduckgo\./.test(host)) return { engine: "duckduckgo", items: scrapeDDG() };
      if (/(^|\.)google\./.test(host)) return { engine: "google", items: scrapeGoogle() };
      if (/bing\./.test(host)) return { engine: "bing", items: scrapeBing() };
      var d = scrapeDDG(); if (d.length) return { engine: "duckduckgo", items: d };
      return { engine: "google", items: scrapeGoogle() };
    }
    function pack(res) {
      var q = "";
      try { q = new URLSearchParams(location.search).get("q") || ""; } catch (e) { /* ignore */ }
      return { capability: "web.search", schema: "WebResult[]", provider: res.engine, query: q, available: res.items.length > 0, count: res.items.length, items: res.items };
    }
    var first = scrape();
    if (first.items.length) return Promise.resolve(pack(first));
    // SERP may render a touch late — one short retry.
    return wait(1200).then(function () { return pack(scrape()); });
  }

  // "10K" / "1.2M" / "710" -> integer.
  function parseCount(s) {
    if (!s) return null;
    var m = String(s).replace(/,/g, "").match(/([\d.]+)\s*([KMB])?/i);
    if (!m) return null;
    var n = parseFloat(m[1]);
    var u = (m[2] || "").toUpperCase();
    if (u === "K") n *= 1e3; else if (u === "M") n *= 1e6; else if (u === "B") n *= 1e9;
    return Math.round(n);
  }
  // fb.profile.header — DOM/SSR scrape of a profile/page header: follower count,
  // verified badge, category, external website, CTA buttons, and whether the
  // profile has Reels/Videos tabs. These are server-rendered (no clean GraphQL,
  // like About), so we read text + links — robust to Facebook's class churn.
  // Returns ONE record: the profile header. (This is the top opener of ~every
  // hand-written outreach email: "your page has N followers, a verified badge…")
  function profileHeader(inputs) {
    inputs = inputs || {};
    // Defense-in-depth: if the tab landed on the operator's OWN profile (bare
    // profile.php with no id, or /me), do NOT return self as if it were the lead.
    try {
      var _p = location.pathname.replace(/\/+$/, "").toLowerCase();
      if ((_p === "/profile.php" && !new URLSearchParams(location.search).get("id")) || _p === "/me") {
        // `reason` is a STABLE marker, not decoration: fb.profile.dossier has to tell this
        // refusal apart from the ordinary available:false this function returns when it merely
        // failed to read a name. Bailing on both would throw away a whole About walk over a
        // missing heading. Never match on the error prose — it is not a contract.
        return { capability: "fb.profile.header", available: false, reason: "self_profile", count: 0, items: [], error: "resolved to the logged-in operator (ambiguous URL); need profile.php?id=<id> or a vanity URL", _debug: { href: location.href } };
      }
    } catch (e) { /* ignore */ }
    var body = (document.body ? document.body.innerText : "").replace(/ /g, " ");
    var html = document.body ? document.body.innerHTML : "";
    var flat = body.replace(/\s+/g, " ");
    function m1(re) { var m = flat.match(re); return m ? m[1] : null; }

    var followersRaw = m1(/([\d.,]+\s*[KMB]?)\s*followers/i);
    var likesRaw = m1(/([\d.,]+\s*[KMB]?)\s*likes/i);
    // name: a level-1 heading in the profile that is not Facebook chrome (the
    // logged-in tab title is just "(N) Facebook"; og:title is often generic).
    // Facebook renders its own chrome as level-1 headings too — the chat rail's "Chats" won
    // this race on all 50 profiles of a live sweep, so every record came back named "Chats".
    // Anything on this list is furniture, never a person.
    var GENERIC = /^(Notifications?|Facebook|Menu|Search|Marketplace|Home|Watch|Groups?|Gaming|Messenger|Chats?|Contacts?|Reels|Videos|Photos|About|Profile|Friends|Create|Feed|Pages?|Stories|Shortcuts|Explore|Saved|Events|Memories|Your shortcuts|Sponsored|Suggested for you)$/i;
    var name = "";
    // Prefer headings inside the main column: the chat rail and left nav also emit h1s.
    var main = document.querySelector('[role="main"]') || document;
    var heads = main.querySelectorAll('h1, [role="heading"][aria-level="1"]');
    if (!heads.length) heads = document.querySelectorAll('h1, [role="heading"][aria-level="1"]');
    for (var hi = 0; hi < heads.length; hi++) {
      var ht = (heads[hi].innerText || "").replace(/\s+/g, " ").trim();
      if (ht.length >= 2 && ht.length <= 70 && !GENERIC.test(ht)) { name = ht; break; }
    }
    if (!name) { var og = document.querySelector('meta[property="og:title"]'); if (og) { var ogt = String(og.getAttribute("content") || "").trim(); if (ogt && !GENERIC.test(ogt)) name = ogt; } }
    // the display name is often only in the self-link (an <a> back to the profile slug).
    if (!name) {
      var slug = (location.pathname.split("/").filter(Boolean)[0] || "");
      if (slug && slug !== "profile.php") {
        var pls = document.querySelectorAll('a[href^="/' + slug + '"], a[href*="facebook.com/' + slug + '"]');
        for (var pi = 0; pi < pls.length; pi++) {
          var pt = (pls[pi].innerText || "").replace(/\s+/g, " ").trim();
          if (pt.length >= 2 && pt.length <= 70 && !GENERIC.test(pt) && !/^\d/.test(pt) && !/^https?:/i.test(pt)) { name = pt; break; }
        }
      }
    }
    var titleName = (document.title || "").replace(/^\(\d+\)\s*/, "").replace(/\s*[|·].*$/, "").trim();
    if (!name && titleName && !GENERIC.test(titleName)) name = titleName;
    if (GENERIC.test(name)) name = ""; // never ship "Facebook"/"Notifications" as a name
    var verified = /verified account/i.test(html) || /aria-label="[^"]*[Vv]erified/.test(html);

    // website: Facebook wraps a profile's declared site in l.php?u=<enc>. Decode
    // those, skip maps/embeds, and prefer the link whose VISIBLE TEXT is a bare
    // domain (that is the profile's own website, not an embedded map/share link).
    var website = "", fallbackExt = "";
    var links = document.querySelectorAll('a[href^="http"]');
    for (var i = 0; i < links.length; i++) {
      var a = links[i], h = a.getAttribute("href") || "";
      var atext = (a.innerText || "").replace(/\s+/g, " ").trim();
      if (/\/l\.php\?|l\.facebook\.com\/l\.php/.test(h)) {
        var um = h.match(/[?&]u=([^&]+)/);
        if (um) {
          try {
            var real = decodeURIComponent(um[1]).split("?")[0];
            if (real && !/facebook\.com|instagram\.com|threads\.net|messenger\.com|\bmaps\.|\/maps|bing\.com|google\.[a-z.]+\/maps/.test(real)) {
              if (/^[\w-]+(\.[\w-]+)+\/?$/.test(atext.replace(/^https?:\/\//, ""))) { website = real; break; } // anchor text is a domain
              if (!fallbackExt) fallbackExt = real;
            }
          } catch (e) { /* skip */ }
        }
        continue;
      }
      if (/facebook\.com|fbcdn|fb\.com|messenger\.com|instagram\.com|threads\.net|bing\.com|\/maps/.test(h)) continue;
      if (!fallbackExt) fallbackExt = h.split("?")[0];
    }
    if (!website) website = fallbackExt;

    var hasReels = /\/reels(\/|\?|")/i.test(html) || />Reels</.test(html);
    var hasVideos = /\/videos(\/|\?|")/i.test(html) || />Videos</.test(html);

    var cta = [], seenC = {};
    var btns = document.querySelectorAll('[role="button"], a[role="button"]');
    for (var b = 0; b < btns.length && cta.length < 5; b++) {
      var t = (btns[b].innerText || btns[b].getAttribute("aria-label") || "").replace(/\s+/g, " ").trim();
      if (/^(Book Now|Send Message|Message|Call Now|Contact|Contact Us|Learn More|Sign Up|Shop Now|Get Quote|View Shop|WhatsApp)$/i.test(t) && !seenC[t.toLowerCase()]) { seenC[t.toLowerCase()] = 1; cta.push(t); }
    }

    // NARROW ON PURPOSE, and not the answer to "what does this person do". This whitelist only
    // recognises real-estate and insurance wording, matched anywhere in the page text — so any
    // other trade reads as empty here, and a post that merely mentions "real estate" reads as a
    // realtor. Kept for backward compatibility; `intro`/`work` below are what an agent should
    // actually read.
    var category = m1(/\b(Real Estate Agent|Realtor|Real Estate Company|Real Estate Service|Estate Agent|Mortgage Broker|Loan Officer|Insurance Agent|Insurance Broker|Real Estate)\b/i) || "";

    // THE INTRO CARD — what the profile says about itself, on the page already loaded.
    // People who work for a living say so on their own profile, so this is both cheaper and
    // more reliable than inferring a trade from their posts. No extra navigation: the About
    // sub-tab ladder is what made a previous attempt at this blow the capability timeout.
    // Returned as TEXT for the agent to read, not squeezed into a fixed taxonomy — a fixed
    // list is exactly what makes every trade outside it invisible.
    var INTRO_PAT = {
      work: [/^works?\s+at\s+(.+)$/i, /^working\s+at\s+(.+)$/i, /^worked\s+at\s+(.+)$/i,
             /^former\s+.*\bat\s+(.+)$/i, /^(?:từng\s+)?làm\s+việc\s+tại\s+(.+)$/i],
      education: [/^stud(?:ied|ies)\s+at\s+(.+)$/i, /^went\s+to\s+(.+)$/i, /^(?:từng\s+)?học\s+(?:tại|ở)\s+(.+)$/i],
      location: [/^lives?\s+in\s+(.+)$/i, /^from\s+(.+)$/i, /^sống\s+(?:tại|ở)\s+(.+)$/i, /^đến\s+từ\s+(.+)$/i]
    };
    function tidy(x) { return String(x == null ? "" : x).replace(/\s+/g, " ").trim(); }
    var introWork = [], introEdu = [], introLoc = [], introLines = [];
    (function () {
      var CHROME = /^(like|comment|share|follow|message|add friend|see all|more|photos|videos|reels|friends|about|posts|intro|edit profile|create|log in|sign up|thích|bình luận|chia sẻ|theo dõi|nhắn tin|xem thêm|giới thiệu)$/i;
      var raw = String(body).split("\n");
      var seen = {};
      for (var i = 0; i < raw.length && introLines.length < 40; i++) {
        var line = tidy(raw[i]);
        if (!line || line.length < 3 || line.length > 220 || seen[line] || CHROME.test(line)) continue;
        seen[line] = 1;
        var pushed = false;
        for (var g in INTRO_PAT) {
          for (var k = 0; k < INTRO_PAT[g].length; k++) {
            var mm = line.match(INTRO_PAT[g][k]);
            if (!mm) continue;
            var val = tidy(mm[1]).slice(0, 160);
            var bucket = g === "work" ? introWork : (g === "education" ? introEdu : introLoc);
            if (val && bucket.indexOf(val) === -1) bucket.push(val);
            pushed = true; break;
          }
          if (pushed) break;
        }
        introLines.push(line);
      }
    })();
    // The self-declared blurb: the longest line that is not one of the structured facts and not
    // page furniture. Facebook does not label it, so it cannot be keyed off a prefix.
    // "Longest line that is not a structured fact" picked a REVIEW on all three live pages —
    // "Cảm ơn Ann Vương…", "Family and Friends; If you are looking for a realestate agent…" —
    // because a recommendation is always longer than a bio. Position settles it: the intro card
    // is above the reviews, always. Stop reading at the first review marker instead of trying to
    // recognise review prose, which is unbounded and multilingual.
    var REVIEW_MARK = /recommends|%\s*recommend|\(\d+\s*Reviews?\)|^Reviews?$|^Rating and reviews$|^Đánh giá$/i;
    var introBio = "";
    for (var li = 0; li < introLines.length; li++) {
      var L = introLines[li];
      if (REVIEW_MARK.test(L)) break;   // everything from here down belongs to other people
      if (L.length < 20 || L.length > 300) continue;
      if (/^(works?|worked|stud|went to|lives?|from|làm việc|từng|học|sống|đến từ)/i.test(L)) continue;
      if (/\d+\s*(followers|following|friends|likes|người theo dõi)/i.test(L)) continue;
      // Facebook's own chrome reads like prose and is long enough to win a longest-line contest:
      // two live profiles came back with "Number of unread notifications" as their bio, because
      // their real intro lines were short ("Mortgage Brokers", "📍Expert IL/GA Realtor").
      if (/^(number of unread|new notification|unread |active status|notifications?$|search$|menu$)/i.test(L)) continue;
      // The display name is not a bio. A profile whose every intro line is short — "Mortgage
      // Brokers", an emoji tagline — leaves the name as the longest survivor, and returning it
      // says nothing a caller does not already have in `name`.
      if (name && L === name) continue;
      if (L.length > introBio.length) introBio = L;
    }

    var header = {
      name: name,
      url: (location.href || "").split("?")[0],
      follower_count: parseCount(followersRaw),
      follower_text: followersRaw ? followersRaw.replace(/\s+/g, "") : "",
      like_count: parseCount(likesRaw),
      verified: !!verified,
      category: category,
      // What the profile SAYS about itself. Read these to work out a trade — `category` above
      // only ever recognises real-estate/insurance wording and is blind to every other job.
      intro_bio: introBio,
      work: introWork,
      education: introEdu,
      location: introLoc,
      intro_lines: introLines.slice(0, 20),
      website: website,
      cta: cta,
      has_reels_tab: !!hasReels,
      has_videos_tab: !!hasVideos
    };
    var ok = !!(header.name || header.follower_count);
    var out = { capability: "fb.profile.header", schema: "ProfileHeader", available: ok, count: ok ? 1 : 0, items: [header] };
    if (inputs.debug) out._debug = { body_len: body.length, followersRaw: followersRaw, likesRaw: likesRaw, external_link_count: links.length };
    return Promise.resolve(out);
  }

  // fb.profile.contacts — dig the whole profile for a published address in ONE job.
  // A business almost always prints its email somewhere on its own profile, but it can
  // be in any of several places: the main page bio (often behind "See more"), or one of
  // the About sub-tabs (contact_info → intro → basic_info → links). Measured: bgvinvest
  // showed it on all five, Khanhngo.us only on MAIN (once "See more" was expanded) and
  // contact_info — so a pass that checks only one surface misses real addresses.
  // The sub-tabs are CLICKED, never fetched — see clickTabAndScan below for why a
  // same-origin fetch returns the SPA shell without the contact block. One job still
  // covers the whole ladder; it just walks it the way a person would.
  var CONTACT_TABS = ["directory_contact_info", "directory_intro", "directory_basic_info", "directory_links"];
  var C_EMAIL_RE = /[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}/g;
  var C_JUNK_DOMAIN = /(^|\.)(example|test|domain|yourdomain|sentry|facebook|fbcdn|whatsapp|instagram)\.[a-z]{2,}$/i;
  var C_ASSET = /\.(png|jpe?g|gif|svg|webp|bmp|ico|css|js|json|woff2?|ttf|mp4)$/i;
  function contactEmailsFrom(text) {
    var out = [], seen = {};
    var m = String(text || "").match(C_EMAIL_RE) || [];
    for (var i = 0; i < m.length; i++) {
      var e = m[i].toLowerCase().replace(/[.,;:)]+$/, "");
      var at = e.lastIndexOf("@"); if (at <= 0) continue;
      var dom = e.slice(at + 1);
      if (C_ASSET.test(e) || C_JUNK_DOMAIN.test(dom) || /@\d+x/.test(e)) continue;
      if (seen[e]) continue; seen[e] = 1; out.push(e);
    }
    return out;
  }
  // The profile also publishes a WEBSITE (usually under directory_links). When no
  // address is on Facebook that link is the next step, so return it instead of making
  // the caller re-scan — my own ad-hoc scan of the record missed sites that were plainly
  // there, which mislabelled those leads "nothing anywhere".
  var C_SOCIAL_HOST = /(facebook|fbcdn|fbsbx|messenger|whatsapp)\./i;
  function contactWebsitesFrom(text) {
    var out = [], seen = {};
    var m = String(text || "").match(/https?:\/\/[^\s"'<>\\)]+/g) || [];
    for (var i = 0; i < m.length; i++) {
      var u = m[i].replace(/[",.);]+$/, "");
      if (C_SOCIAL_HOST.test(u) || C_ASSET.test(u.split("?")[0])) continue;
      try {
        var h = new URL(u).hostname.replace(/^www\./, "");
        if (!h || seen[h]) continue; seen[h] = 1; out.push(u.split("?")[0]);
      } catch (e) { /* skip */ }
      if (out.length >= 6) break;
    }
    return out;
  }

  function stripMarkup(html) {
    return String(html || "")
      .replace(/<script[\s\S]*?<\/script>/gi, " ")
      .replace(/<style[\s\S]*?<\/style>/gi, " ")
      .replace(/\\u0040/gi, "@").replace(/&#64;/g, "@").replace(/&amp;/g, "&");
  }
  function profileBaseFrom(href) {
    try {
      var u = new URL(href, location.origin);
      var id = u.searchParams.get("id");
      if (/\/profile\.php$/i.test(u.pathname) && id) return { numeric: true, id: id, base: u.origin + "/profile.php?id=" + id };
      var seg = u.pathname.split("/").filter(Boolean)[0] || "";
      if (!seg || /^(profile\.php|me|reel|watch|groups|search)$/i.test(seg)) return null;
      return { numeric: false, id: seg, base: u.origin + "/" + seg };
    } catch (e) { return null; }
  }
  // Wait for the tab's content to actually render before reading it. A fixed pause
  // is not enough: the SAME profile on the SAME build reached only the main page in
  // one run and the full ladder in the next, purely from render timing — that race,
  // not a selector bug, is what made contact discovery flaky.
  function settleThenScan(maxMs) {
    var last = -1, stable = 0, waited = 0;
    function tick() {
      var len = document.body && document.body.innerText ? document.body.innerText.length : 0;
      if (len === last && len > 0) { stable += 1; if (stable >= 2) return Promise.resolve(len); }
      else { stable = 0; }
      last = len; waited += 350;
      if (waited >= (maxMs || 5000)) return Promise.resolve(len);
      return wait(350).then(tick);
    }
    return tick();
  }

  function profileContacts(inputs) {
    inputs = inputs || {};
    var target = String(inputs.profile_url || location.href);
    var info = profileBaseFrom(target);
    var checked = [], emails = [], websites = [], foundOn = "";
    function addFrom(text, label) {
      var got = contactEmailsFrom(text);
      for (var i = 0; i < got.length; i++) if (emails.indexOf(got[i]) === -1) { emails.push(got[i]); if (!foundOn) foundOn = label; }
      var sites = contactWebsitesFrom(text);
      for (var w = 0; w < sites.length; w++) if (websites.indexOf(sites[w]) === -1 && websites.length < 6) websites.push(sites[w]);
    }
    // 1) whatever is already rendered here (background.js has expanded "See more").
    addFrom(document.body ? document.body.innerText : "", "current_page");
    checked.push("current_page");
    if (!info) {
      return Promise.resolve({ capability: "fb.profile.contacts", schema: "ContactRecord", available: true, found: emails.length > 0, count: emails.length ? 1 : 0, items: [{ profile_url: target, emails: emails, websites: websites, found_on: foundOn || null, checked: checked }], error: emails.length ? null : "could not resolve a profile base from the url" });
    }
    // 2) walk the About sub-tabs by CLICKING them, never by fetching them.
    // A same-origin fetch of /directory_contact_info returns the SPA shell, not the
    // rendered contact block: across 68 profiles every single address came from the
    // rendered page and ZERO came from a fetched tab — including one whose email was
    // sitting in Contact info the whole time. Worse, counting those fetches as
    // "checked" produced an audit trail that claimed the ladder had run when it had
    // not. Clicking makes Facebook render the tab for real, in the same job.
    var stopEarly = inputs.stop_at_first !== false;
    var TAB_LABELS = {
      directory_contact_info: /^(contact info|contact and basic info|thông tin liên hệ)/i,
      directory_intro: /^(intro|giới thiệu)/i,
      directory_basic_info: /^(basic info|overview|thông tin cơ bản|tổng quan)/i,
      directory_links: /^(links|websites and social links|liên kết)/i
    };
    function clickTabAndScan(tab) {
      var re = TAB_LABELS[tab];
      // Prefer the tab's own HREF (href*= slug first): the visible label differs between
      // Pages and personal profiles and across locales, and matching text alone missed
      // "Contact info" on 13 of 29 profiles (they clicked Intro and stopped). The slug
      // in the link is the same everywhere.
      var byHref = document.querySelectorAll('a[href*="' + tab + '"]');
      for (var h = 0; h < byHref.length; h++) {
        var hb = byHref[h].getBoundingClientRect();
        if (hb.width <= 0 || hb.height <= 0) continue;
        try { byHref[h].click(); } catch (e) { break; }
        return wait(600).then(function () { return settleThenScan(5000); }).then(function () {
          addFrom(document.body ? document.body.innerText : "", tab);
          checked.push(tab);
          return true;
        });
      }
      var nodes = document.querySelectorAll('a[role="link"], [role="tab"], [role="button"], [role="listitem"] a');
      for (var i = 0; i < nodes.length; i++) {
        var n = nodes[i];
        var label = (n.innerText || n.getAttribute("aria-label") || "").replace(/\s+/g, " ").trim();
        if (!re.test(label)) continue;
        var box = n.getBoundingClientRect();
        if (box.width <= 0 || box.height <= 0) continue;
        try { n.click(); } catch (e) { return Promise.resolve(false); }
        return wait(600).then(function () { return settleThenScan(5000); }).then(function () {
          addFrom(document.body ? document.body.innerText : "", tab);
          checked.push(tab);
          return true;
        });
      }
      return Promise.resolve(false); // tab not offered by this profile
    }
    // The sub-tab strip only exists INSIDE the About section — on the main profile
    // there is just an Intro card — so enter About first, otherwise every sub-tab click
    // finds nothing and the ladder silently does nothing.
    function enterAboutSection() {
      if (/\/about|sk=about|directory_/i.test(location.href)) return Promise.resolve(true);
      var re = /^(about|giới thiệu)$/i;
      var byHref = document.querySelectorAll('a[href*="sk=about"], a[href$="/about"], a[href*="/about?"]');
      for (var h = 0; h < byHref.length; h++) {
        var hb = byHref[h].getBoundingClientRect();
        if (hb.width <= 0 || hb.height <= 0) continue;
        try { byHref[h].click(); } catch (e) { break; }
        return wait(700).then(function () { return settleThenScan(5000); }).then(function () { addFrom(document.body ? document.body.innerText : "", "about"); checked.push("about"); return true; });
      }
      var nodes = document.querySelectorAll('a[role="link"], [role="tab"], [role="button"]');
      for (var i = 0; i < nodes.length; i++) {
        var n = nodes[i];
        var label = (n.innerText || n.getAttribute("aria-label") || "").replace(/\s+/g, " ").trim();
        if (!re.test(label)) continue;
        var box = n.getBoundingClientRect();
        if (box.width <= 0 || box.height <= 0) continue;
        try { n.click(); } catch (e) { return Promise.resolve(false); }
        return wait(700).then(function () { return settleThenScan(5000); }).then(function () { addFrom(document.body ? document.body.innerText : "", "about"); checked.push("about"); return true; });
      }
      return Promise.resolve(false);
    }
    var idx = 0;
    function step() {
      if (idx >= CONTACT_TABS.length || (stopEarly && emails.length)) return Promise.resolve();
      var tab = CONTACT_TABS[idx++];
      return clickTabAndScan(tab).then(function () { return wait(200).then(step); });
    }
    function runLadder() {
      if (stopEarly && emails.length) return Promise.resolve();
      return enterAboutSection().then(step);
    }
    return runLadder().then(function () {
      var ok = emails.length > 0;
      // available:true whenever the LADDER RAN — background.js nulls out a record whose
      // capability reports unavailable, which threw away the `checked` audit trail for
      // exactly the leads that need it: a playbook could no longer tell "dug the whole
      // profile, genuinely no address" from "never looked". Emptiness is carried by
      // `emails: []`, not by hiding the record.
      return {
        capability: "fb.profile.contacts", schema: "ContactRecord", available: true, found: ok, count: ok ? 1 : 0,
        items: [{ profile_url: info.base, emails: emails, websites: websites, found_on: foundOn || null, checked: checked }],
        checked: checked,
        error: ok ? null : "no published address found on the profile or its About sub-tabs"
      };
    });
  }

  // fb.profile.dossier — ONE visit, the WHOLE profile. Header + the full About ladder in a
  // single capability, because enrich previously needed two capabilities (fb.profile.header,
  // then fb.profile.contacts) on the same person, which means two tab loads, twice the
  // wall-clock and twice the rate-limit exposure for one lead. Halving the requests doubles
  // how many leads a run can process, so this is a throughput change, not a tidy-up.
  //
  // Three things it does that the two older capabilities do not:
  //   1) it walks "Work and education", which the contacts ladder never visited — that tab is
  //      where a profile prints a JOB TITLE ("Loan Officer at Wells Fargo"). The title, not the
  //      employer, is what separates Loan & Mortgage from Banking & Financial. `work[]` on
  //      fb.profile.header only ever captured the employer, because its patterns are anchored
  //      to "works at <X>".
  //   2) it never stops at the first email. Stopping early is right when the goal is an address
  //      and wrong when the goal is a dossier: the tabs after the hit are the ones with the trade.
  //   3) it KEEPS the text of every tab it opens. The contacts ladder rendered those tabs and then
  //      ran two regexes over them, discarding the profession it had just paid to load.
  //
  // Deliberately NOT a classifier. It returns what the profile says about itself as text and lets
  // the reading agent choose an industry. A keyword map here is what makes every trade outside the
  // map invisible — the failure already recorded on `category` (line ~1703) and `industryHint()`.
  //
  // TIMEOUT — read before raising any budget. background.js:752 kills a capability at 45s, and an
  // earlier attempt at exactly this ladder (fb.profile.about) blew that limit twice and had to be
  // deleted. Five tabs at the contacts ladder's 5s settle is ~35s worst case, which leaves no room
  // for a slow profile. So: a 2.8s per-tab settle (an About sub-tab is small; settleThenScan
  // returns as soon as the text stops growing, typically ~1s), and a hard internal deadline. When
  // the budget runs out the walk STOPS and returns what it has with `checked[]` and
  // `budget_exhausted: true` — a partial dossier that says which tabs it read beats a record the
  // 45s timeout turned into nothing.
  // FIVE sections, and only five. The About sub-nav has about fourteen — Hobbies, Interests,
  // Travel, Communities, Offers, Names, Privacy and legal info — and each one costs a click, a
  // settle and a scan while answering neither question this capability exists for. These five
  // do: contact_info is the only place a published address lives, and work / education / intro
  // / category are what say what someone does for a living.
  //
  // The slugs are a fixed vocabulary (/<profile>/directory_<section>), not something to infer.
  // An earlier version of this list was invented — work_and_education, basic_info, links — and
  // only `intro` was real; Work and Education are two separate sections.
  var DOSSIER_TABS = [
    { key: "contact_info", hrefs: ["directory_contact_info"], label: /^(contact info|thông tin liên hệ)/i },
    { key: "work", hrefs: ["directory_work"], label: /^(work|công việc)/i },
    { key: "education", hrefs: ["directory_education"], label: /^(education|học vấn|giáo dục)/i },
    { key: "intro", hrefs: ["directory_intro"], label: /^(intro|giới thiệu)/i },
    { key: "category", hrefs: ["directory_category"], label: /^(category|hạng mục|danh mục)/i }
  ];
  // Discovery drives the walk, so trimming the list above is not enough on its own: a profile
  // that publishes all fourteen sections would still be walked end to end. Discovery is filtered
  // to these keys, in this order — Facebook renders Hobbies before Contact info, and following
  // its menu order would spend the budget on hobbies.
  var DOSSIER_RANK = {};
  for (var _d = 0; _d < DOSSIER_TABS.length; _d++) DOSSIER_RANK[DOSSIER_TABS[_d].key] = _d;
  // Headings Facebook writes in the About panel. They label the value below them and are never
  // the value themselves.
  var SECTION_LABEL = /^(all|followers|category|details|links|services|work|education|contact info|personal details|privacy and legal info|names|reviews|social media|bio|address|phone|mobile|email|messenger|hours|website|rating and reviews|intro|giới thiệu|địa chỉ|điện thoại)$/i;
  var DOSSIER_CHROME = /^(like|comment|share|follow|following|message|add friend|see all|see more|more|photos|videos|reels|friends|about|posts|intro|edit profile|create|log in|sign up|suggested for you|sponsored|thích|bình luận|chia sẻ|theo dõi|nhắn tin|xem thêm|xem tất cả|giới thiệu|bạn bè|ảnh|video)$/i;

  // fb.profile.enrich — ONE tab, the whole lead: recent posts AND the About section.
  //
  // Stage 4 requires both and they came from different capabilities, so enriching one person cost
  // two or three page loads: fb.profile.posts for the dated proof-of-life a `high` band demands,
  // then fb.profile.dossier for the address and the trade. Same person, same tab, twice the
  // wall-clock and twice the rate-limit exposure.
  //
  // The order is forced and is the whole trick. The timeline query only fires when the tab LANDS
  // on the profile root, and gql_intercept captures it there; the About walk then happens in the
  // same tab, after. Submit /about and the timeline never renders, so there is nothing to read —
  // which is why this capability takes the ROOT url while fb.profile.dossier takes /about. Reading
  // the captures first also means no second navigation: the posts are already in the store.
  function profileEnrich(inputs) {
    inputs = inputs || {};
    var maxPosts = Number(inputs.max_posts) > 0 ? Math.min(Number(inputs.max_posts), 25) : 5;
    var startedAt = Date.now();

    function takeCaptured(capId, n) {
      // The synchronous, capture-reading extractor — no navigation, no scroll. It reports
      // available:false when nothing matched, which here means "the timeline had not rendered
      // yet", not "this person has no posts"; the difference is carried in `reason`.
      var res;
      try { res = window.__soloGqlExtractCapability(capId, inputs); }
      catch (e) { return { available: false, reason: String(e && e.message || e), items: [] }; }
      if (!res || !Array.isArray(res.items)) return { available: false, reason: (res && res.reason) || "no_items", items: [] };
      return { available: res.available !== false, reason: res.reason || null, items: res.items.slice(0, n), total_seen: res.items.length };
    }

    // Give the timeline a moment to arrive. background.js has already waited for load, but the
    // feed query lands after first paint and this capability is worthless without it.
    return settleThenScan(Number(inputs.settle_ms) > 0 ? Number(inputs.settle_ms) : 2500).then(function () {
      var posts = takeCaptured("fb.profile.posts", maxPosts);
      // Videos live behind their own tab and normally have not fired here. Take them when the
      // capture happens to exist and never navigate for them — a second page load is the cost
      // this capability exists to remove.
      var videos = takeCaptured("fb.profile.videos", maxPosts);
      var landedOn = (location.href || "").split("?")[0];
      return profileDossier(inputs).then(function (dos) {
        if (dos && dos.reason === "self_profile") return dos;
        var about = (dos && dos.items && dos.items[0]) || {};
        var item = {};
        for (var k in about) item[k] = about[k];
        item.posts = posts.items;
        item.videos = videos.items;
        item.timeline = {
          landed_on: landedOn,
          posts_available: posts.available, posts_reason: posts.reason,
          posts_seen: posts.total_seen || 0, posts_kept: posts.items.length,
          videos_available: videos.available, videos_seen: videos.total_seen || 0
        };
        item.elapsed_ms = Date.now() - startedAt;
        var ok = !!(item.name || (item.emails || []).length || (item.about_lines || []).length || posts.items.length);
        return {
          capability: "fb.profile.enrich", schema: "ProfileEnrich",
          // available:true whenever the pass RAN. A profile with no public posts and a private
          // About is a real answer; hiding the record makes it indistinguishable from a crash,
          // which this file has been bitten by three times.
          available: true, found: ok, count: ok ? 1 : 0, items: [item],
          checked: item.checked || [],
          error: ok ? null : "profile opened but yielded no name, address, About text or posts"
        };
      });
    });
  }

  // ---- About sections by GraphQL replay, no clicking ---------------------------------------
  //
  // Measured on three live pages: landing on /about surfaces 6, 10 and 10 section tokens with
  // ZERO clicks, all served by one persisted query (ProfileCometAboutAppSectionQuery,
  // doc_id 27470497829312569). So the sections can be fetched directly.
  //
  // This is not an optimisation. Scoping the DOM to the About card failed three times — first by
  // not clicking posts' "See more", then by filtering role=article/feed, then by walking up to a
  // container — and each fix was outflanked by a section type the last one did not anticipate
  // (it was a Reviews card, which is neither article nor feed). Facebook renders About, Reviews,
  // Posts and Photos into ONE tree, so any boundary drawn with selectors is a guess that holds
  // until the next layout. A GraphQL response for an About section cannot contain a review: the
  // separation is structural, not positional, and that is why this replaces the walk instead of
  // patching it.
  var ABOUT_TOKEN_RE = /^YXBwX2NvbGxl/;   // base64 of "app_colle…" — an app_collection token

  // Every string leaf, which is all a section needs to be: the caller wants plain text, and
  // parsing a fixed shape out of these responses would be one more thing to guess wrong when
  // Facebook moves a field.
  function textLeaves(node, out, depth) {
    if (!node || depth > 12 || out.length >= 200) return out;
    if (typeof node === "string") {
      var s = node.replace(/\s+/g, " ").trim();
      if (s.length >= 2 && s.length <= 400 && !/^[A-Za-z0-9+/=_-]{40,}$/.test(s) && !/^https?:\/\/(scontent|static)\./.test(s)) out.push(s);
      return out;
    }
    if (typeof node !== "object") return out;
    if (Array.isArray(node)) { for (var i = 0; i < node.length; i++) textLeaves(node[i], out, depth + 1); return out; }
    for (var k in node) {
      if (k === "__typename" || k === "id" || /token|cursor|key$/i.test(k)) continue;
      textLeaves(node[k], out, depth + 1);
    }
    return out;
  }

  // Collect every app_collection token the page already received, with a label when one sits
  // beside it — the nav entry carries both, so the section can be named rather than numbered.
  function collectSectionTokens(caps) {
    var found = [], seen = {};
    function walk(node, label, depth) {
      if (!node || typeof node !== "object" || depth > 14) return;
      if (Array.isArray(node)) { for (var i = 0; i < node.length; i++) walk(node[i], label, depth + 1); return; }
      var here = label;
      for (var t in node) {
        var v = node[t];
        if (typeof v === "string" && /^(title|name|label|text)$/i.test(t) && v.length < 60) here = v;
      }
      for (var k in node) {
        var val = node[k];
        if (typeof val === "string" && ABOUT_TOKEN_RE.test(val)) {
          if (!seen[val]) { seen[val] = 1; found.push({ token: val, key: k, label: here || ("section_" + found.length) }); }
        } else walk(val, here, depth + 1);
      }
    }
    for (var c = 0; c < caps.length; c++) { try { walk(caps[c].response, "", 0); } catch (e) { /* skip */ } }
    return found;
  }

  function replaySection(store, seed, token) {
    var vars = {};
    for (var k in seed.variables) vars[k] = seed.variables[k];
    // ONLY collectionToken changes. sectionToken and rawSectionToken are separate variables in
    // the captured call, and overwriting them with the collection token returned nine sections
    // that were structurally fine and completely empty — the hardest failure to notice, because
    // ok:true, failed:0 and sections:9 all say it worked.
    vars.collectionToken = token;
    var p = new URLSearchParams();
    p.set("av", seed.av || "");
    p.set("__a", "1");
    p.set("fb_dtsg", seed.fbDtsg || "");
    p.set("fb_api_caller_class", "RelayModern");
    p.set("fb_api_req_friendly_name", seed.queryName || "");
    p.set("variables", JSON.stringify(vars));
    p.set("doc_id", docIdFromRegistry(seed.queryName) || seed.docId);
    p.set("server_timestamps", "true");
    return store.origFetch(seed.url || "/api/graphql/", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded", "X-FB-Friendly-Name": seed.queryName || "" },
      body: p.toString()
    }).then(function (r) { return r.text(); }).then(function (text) {
      return store.parseResponse ? store.parseResponse(text) : JSON.parse(String(text).replace(/^for\s*\(;;\);/, ""));
    });
  }

  function aboutViaGraphQL(inputs) {
    var store = window.__soloGql;
    if (!store || !Array.isArray(store.captures) || typeof store.origFetch !== "function") {
      return Promise.resolve({ ok: false, reason: "no_capture_store" });
    }
    var caps = store.captures, seed = null;
    for (var i = caps.length - 1; i >= 0; i--) {
      var c = caps[i];
      if (!c || !/AboutAppSection/i.test(String(c.queryName || ""))) continue;
      if (!c.docId || !c.fbDtsg || !c.variables) continue;
      seed = c; break;
    }
    if (!seed) return Promise.resolve({ ok: false, reason: "no_about_query_captured" });
    var tokens = collectSectionTokens(caps);
    if (!tokens.length) return Promise.resolve({ ok: false, reason: "no_section_tokens_in_responses" });

    // Reviews and Reels are sections of the About sub-nav too, so replaying every token fetched
    // them as well — recommendations and view counts filed as profile facts. Excluding them by
    // NAME is exact and stays exact; this is the same exclusion that was impossible to express
    // against the DOM, where a Reviews card is not distinguishable by role or structure.
    var SECTION_NOISE = /^(reviews?|reels?|photos?|videos?|posts?|community|đánh_giá)$/i;
    // LOCAL. This function lives outside profileDossier, so reaching for that closure's `skipped`
    // was a ReferenceError at runtime — and it threw mid-loop, which aborted the remaining
    // sections and dropped the whole record. The offline test missed it only because the noise
    // section happened to be last in the fixture, so the throw landed after the useful work.
    var skippedSections = [], sections = {}, order = [], failed = [], sizes = [];
    var idx = 0;
    function step() {
      if (idx >= tokens.length) return Promise.resolve();
      var t = tokens[idx++];
      if (SECTION_NOISE.test(String(t.label || "").replace(/\s+/g, "_"))) { skippedSections.push(String(t.label)); return step(); }
      return replaySection(store, seed, t.token).then(function (res) {
        var lines = textLeaves(res && res.data, [], 0);
        // Bytes, not just line count: an empty section and a section whose text was filtered out
        // by textLeaves look identical from the outside, and they need opposite fixes.
        var bytes = 0;
        try { bytes = JSON.stringify(res && res.data || {}).length; } catch (e) { bytes = -1; }
        var err = null;
        try {
          var es = (res && (res.errors || res.error)) || null;
          if (es) err = String(JSON.stringify(es)).slice(0, 220);
          else if (bytes <= 2 && res) err = "empty data; top keys=" + Object.keys(res).join(",");
        } catch (e) { /* ignore */ }
        sizes.push({ label: String(t.label || ""), key: t.key || null, bytes: bytes, lines: lines.length, error: err });
        var key = String(t.label || ("section_" + idx)).replace(/\s+/g, "_").toLowerCase();
        if (sections[key]) key = key + "_" + idx;
        sections[key] = lines;
        order.push(key);
      }).catch(function (e) {
        failed.push({ label: t.label, error: String(e && e.message || e) });
      }).then(step);
    }
    return step().then(function () {
      return { ok: order.length > 0, sections: sections, order: order, failed: failed,
               skipped: skippedSections, sizes: sizes,
               doc_id: String(seed.docId || ""), query: String(seed.queryName || ""),
               tokens_found: tokens.length, reason: order.length ? null : "every_replay_failed" };
    });
  }

  function profileDossier(inputs) {
    inputs = inputs || {};
    var startedAt = Date.now();
    // Budget for the LADDER only, and the number that actually stops the walk — the capability
    // timeout is the wall behind it, not the limit in practice. Raised with that ceiling from
    // 45s to 60s: a background tab is throttled 2-3x, and at 30s the longer walks exhausted the
    // budget and dropped their last section. 45s keeps ~15s for the landing-page settle, the
    // second header pass and posting the record.
    var budgetMs = Number(inputs.budget_ms) > 0 ? Number(inputs.budget_ms) : 45000;
    var settleMs = Number(inputs.settle_ms) > 0 ? Number(inputs.settle_ms) : 2000;
    function budgetLeft() { return budgetMs - (Date.now() - startedAt); }

    var target = String(inputs.profile_url || location.href);
    var info = profileBaseFrom(target);
    var emails = [], websites = [], foundOn = "", checked = [], missing = [];
    var about = {}, aboutLines = [], seenLine = {}, budgetExhausted = false, discovered = [], skipped = [], seenAll = [], seeMoreClicks = 0, panelFound = false, gqlAbout = null, feedLinesCache = null;

    // ---- GraphQL probe -------------------------------------------------------------------
    // Which query does Facebook fire when a sub-tab is clicked? If a tab's content arrives in
    // one named query, that query can be replayed by doc_id the way fb.profile.hovercard already
    // replays CometHovercardQueryRendererQuery — and the clicking goes away entirely. This is
    // how that hovercard call was found: watch the capture store across a real interaction
    // rather than guess a query name. Recording the names is cheap and always on; the full
    // variables only come along under probe_graphql, because they are large and mostly noise.
    var gqlBySurface = {};
    function captureMark() {
      var s = window.__soloGql;
      return s && Array.isArray(s.captures) ? s.captures.length : -1;
    }
    function recordGql(label, mark) {
      var s = window.__soloGql;
      if (mark < 0 || !s || !Array.isArray(s.captures)) return;
      var out = [];
      for (var i = mark; i < s.captures.length && out.length < 12; i++) {
        var c = s.captures[i] || {};
        var row = { query: String(c.queryName || ""), doc_id: String(c.docId || "") };
        if (inputs.probe_graphql) {
          try { row.variables = JSON.stringify(c.variables).slice(0, 900); } catch (e) { row.variables = "<unserialisable>"; }
          row.has_fb_dtsg = !!c.fbDtsg;
        }
        out.push(row);
      }
      if (out.length) gqlBySurface[label] = (gqlBySurface[label] || []).concat(out);
    }

    // "See more" truncates a bio mid-sentence, and the collector only expands it ONCE, on the
    // entry page, before this capability is injected — so text truncated inside a sub-tab stayed
    // truncated. That was recorded as a known limit; expanding per surface removes it, and the
    // probe then shows whether the expansion is a local re-render or its own query.
    function expandSeeMore() {
      var hits = [], nodes = [];
      // Scoped to the About panel, never the document. A Page's About url also renders Reviews,
      // Posts and Photos below the panel, and every one of those cards carries its own
      // "See more" — a live run clicked them 5, 12 and 14 times. No panel, no clicking: an
      // un-expanded bio is a smaller loss than a review thread filed as a profile fact.
      var panel = aboutPanel();
      if (!panel) return Promise.resolve(0);
      try { nodes = panel.querySelectorAll('[role="button"], span[role="button"], div[role="button"]') || []; } catch (e) { return Promise.resolve(0); }
      for (var i = 0; i < nodes.length && hits.length < 3; i++) {
        var t = (nodes[i].innerText || "").replace(/\s+/g, " ").trim();
        // EXACT labels only. "See all friends" and "See all photos" navigate away, and a walk
        // that wanders off the profile reports whatever page it landed on as the lead.
        if (!/^(see more|xem thêm)$/i.test(t)) continue;
        // EVERY POST HAS ONE. Matching "See more" across the whole document expanded posts and
        // reels, not the bio: a live run clicked it 5, 12 and 14 times on three profiles and
        // pulled "<name>'s Post", a comment thread and a row of reel view counts into the
        // dossier — post text dressed as profile facts, which is exactly the DOM pollution this
        // capability exists to avoid. Refuse anything inside a story/feed container.
        if (inFeed(nodes[i])) continue;
        var b = nodes[i].getBoundingClientRect ? nodes[i].getBoundingClientRect() : { width: 1, height: 1 };
        if (b.width <= 0 || b.height <= 0) continue;
        hits.push(nodes[i]);
      }
      if (!hits.length) return Promise.resolve(0);
      for (var h = 0; h < hits.length; h++) { try { hits[h].click(); } catch (e) { /* ignore */ } }
      return wait(350).then(function () { return settleThenScan(Math.min(settleMs, 1500)); }).then(function () { return hits.length; });
    }
    function inFeed(el) {
      for (var n = el, hops = 0; n && hops < 12; n = n.parentElement, hops++) {
        var role = n.getAttribute ? (n.getAttribute("role") || "") : "";
        if (role === "article" || role === "feed") return true;
        var lbl = n.getAttribute ? (n.getAttribute("aria-label") || "") : "";
        if (/^(post|posts|reels?|bài viết|thước phim)\b/i.test(lbl)) return true;
      }
      return false;
    }

    function addSite(u) {
      if (!u || websites.length >= 8) return;
      var clean = String(u).split("?")[0].replace(/[",.);]+$/, "");
      if (!clean || C_SOCIAL_HOST.test(clean) || C_ASSET.test(clean)) return;
      var host = clean.replace(/^https?:\/\//, "").replace(/^www\./, "").split("/")[0].toLowerCase();
      if (!host) return;
      for (var i = 0; i < websites.length; i++) {
        var h = websites[i].replace(/^https?:\/\//, "").replace(/^www\./, "").split("/")[0].toLowerCase();
        if (h === host) return;
      }
      websites.push(clean);
    }
    // Facebook prints a profile's site as a BARE DOMAIN under "Website" — no scheme — so a
    // scheme-anchored regex over the page text returns nothing on the very tab that publishes
    // it. contactWebsitesFrom() only matches https?://, which is why fb.profile.contacts can
    // report an empty `websites` for a profile that plainly shows one. Read the anchors too
    // (that is how fb.profile.header gets it right), and accept a bare domain when the line is
    // nothing but a domain — narrow on purpose, so prose mentioning a company never counts.
    // The TLD must be two or more LETTERS. Without that the pattern reads Facebook's own
    // follower counters as hostnames — a live run came back with "4.2K", "6.8K" and "1.1K"
    // sitting in `websites` next to real sites, which is worse than missing them: a lead whose
    // website is "1.5K" looks enriched.
    var BARE_DOMAIN = /^[a-z0-9][a-z0-9-]*(\.[a-z0-9-]+)*\.[a-z]{2,}(\/[^\s]*)?$/i;
    function harvestAnchors() {
      var links = [];
      try { links = document.querySelectorAll('a[href]') || []; } catch (e) { return; }
      for (var i = 0; i < links.length; i++) {
        var h = links[i].getAttribute ? (links[i].getAttribute("href") || "") : "";
        if (!h) continue;
        if (/\/l\.php\?|l\.facebook\.com\/l\.php/.test(h)) {
          var um = h.match(/[?&]u=([^&]+)/);
          if (um) { try { addSite(decodeURIComponent(um[1])); } catch (e) { /* skip */ } }
          continue;
        }
        if (/^https?:\/\//.test(h)) addSite(h);
      }
    }
    function addFrom(text, label) {
      var got = contactEmailsFrom(text);
      for (var i = 0; i < got.length; i++) if (emails.indexOf(got[i]) === -1) { emails.push(got[i]); if (!foundOn) foundOn = label; }
      var sites = contactWebsitesFrom(text);
      for (var w = 0; w < sites.length; w++) addSite(sites[w]);
      harvestAnchors();
      var raw = String(text || "").split("\n");
      for (var r = 0; r < raw.length; r++) {
        var line = raw[r].replace(/\s+/g, " ").trim();
        if (line.length < 4 || line.length > 100 || !BARE_DOMAIN.test(line)) continue;
        if (/@/.test(line)) continue;
        addSite(line);
      }
    }
    // Harvest the DELTA, not the page. Every tab re-renders the same header, nav and footer, so
    // storing each tab's full innerText would ship the same ~200 furniture lines five times and
    // bury the handful of lines that are actually this tab's content. New-lines-only keeps the
    // record small enough that an agent can read all of it, which is the whole point.
    // THE ABOUT PANEL ONLY — never the feed underneath it.
    //
    // Facebook keeps rendering the post feed while the About section is open, so
    // document.body.innerText carries the whole timeline. fb.profile.contacts read the same
    // polluted text and got away with it because it ran two regexes over it and threw the rest
    // away; a dossier KEEPS the text, so the same read drags "<name>'s Post", a comment thread
    // and a row of reel view counts into the record as if they were profile facts. Suppressing
    // the "See more" clicks inside posts was not enough and could never have been: the post text
    // is in innerText whether or not anything is clicked.
    //
    // Subtraction by LINE rather than by container, because the About panel has no stable
    // selector to anchor on, while a post always sits in a role=article/feed subtree. Take the
    // main column's text, drop every line that also appears inside one of those subtrees.
    // The About CARD, found by the one thing that is always inside it: the sub-nav. Walk up from
    // a directory_* link until the ancestor holds the whole nav, then one level more to take in
    // the content pane beside it. That element is the About panel; everything below it —
    // Reviews, Posts, Photos, Related Pages — is somebody else's content.
    //
    // Blacklisting containers was the wrong shape and kept losing. role=article/feed does not
    // match a Reviews card, so the reviews under a Page's About kept being read and their
    // "See more" kept being clicked. A whitelist cannot be outflanked by a section type nobody
    // anticipated: if the panel is not found, nothing is expanded at all.
    function aboutPanel() {
      var links = [];
      try { links = document.querySelectorAll('a[href*="directory_"]') || []; } catch (e) { return null; }
      if (!links.length) return null;
      // Climb to the smallest ancestor holding the whole sub-nav. That container is the NAV, and
      // stopping one level above it was wrong: a live run came back with four lines — "Category,
      // Details, Links, Contact info" — the menu and nothing else, no bio, no email.
      var nav = links[0], hops = 0;
      while (nav.parentElement && hops < 12) {
        nav = nav.parentElement; hops += 1;
        var inside = 0;
        try { inside = (nav.querySelectorAll('a[href*="directory_"]') || []).length; } catch (e) { break; }
        if (inside >= links.length) break;
      }
      // Then keep climbing while the ancestor adds nothing but the menu back. The card that also
      // holds the content pane is the first one whose text is materially longer than the nav's.
      // Measured by TEXT GROWTH rather than by a fixed number of hops, because the depth differs
      // between Pages and personal profiles and a magic number would only be the next guess.
      var navLen = String(nav.innerText || "").length;
      var panel = nav, up = 0;
      while (panel.parentElement && up < 8) {
        panel = panel.parentElement; up += 1;
        var len = String(panel.innerText || "").length;
        // The FIRST ancestor that adds real content is the card wrapping the menu and the pane
        // beside it. Anything holding Reviews and Posts as well is further up, so stopping at
        // the first one is what keeps them out. An upper size cap was tried here and was itself
        // a guess — it rejected the correct card on a rich profile.
        if (len > navLen + 120) return panel;
      }
      return null;   // no ancestor carried content — say so rather than return the menu
    }

    // Text length at each ancestor level above the sub-nav. Three attempts at scoping this panel
    // have now failed on a different guess each time; this reports the actual shape of the tree
    // so the next fix is chosen from measurements instead of a fourth guess.
    function panelLadder() {
      var out = [];
      try {
        var links = document.querySelectorAll('a[href*="directory_"]') || [];
        if (!links.length) return out;
        var n = links[0];
        for (var i = 0; i < 14 && n.parentElement; i++) {
          n = n.parentElement;
          var t = String(n.innerText || "");
          out.push({ level: i + 1, chars: t.length,
                     navs: (n.querySelectorAll('a[href*="directory_"]') || []).length,
                     head: t.replace(/\s+/g, " ").slice(0, 70) });
        }
      } catch (e) { /* ignore */ }
      return out;
    }

    // Best-effort text for the FALLBACK path only. It does not try to find the About card any
    // more: three attempts did, each drew the boundary somewhere different, and each was walked
    // through by a section the last one had not met — the last one returned four lines of menu
    // and lost every email. Facebook renders About, Reviews, Posts and Photos into one tree and
    // the boundary is not reliably findable with selectors. So this subtracts what it can prove
    // is foreign (article/feed subtrees) and the record says source:"dom", which is the reader's
    // signal to distrust it. The GraphQL path above has no such problem and is the real answer.
    function profileText() {
      var root = null;
      try { root = document.querySelector('[role="main"]'); } catch (e) { /* ignore */ }
      root = root || document.body;
      if (!root) return "";
      // ONCE, not once per tab. Reading .innerText forces a layout pass, and a Page's About
      // renders dozens of post and review cards, so recomputing this on every harvest turned a
      // 12-second walk into a capability killed at the 45-second ceiling. The feed does not
      // change while the About sub-nav is being walked, so one pass is the whole answer.
      if (!feedLinesCache) {
        feedLinesCache = {};
        var feeds = [];
        try { feeds = root.querySelectorAll('[role="article"], [role="feed"]') || []; } catch (e) { feeds = []; }
        // Bounded as well: an infinite feed can keep growing while the walk runs, and the cost
        // of scanning it is unbounded in a way the budget check between tabs cannot interrupt.
        var lim = Math.min(feeds.length, 40);
        for (var f = 0; f < lim; f++) {
          var ft = String(feeds[f].innerText || "").split("\n");
          for (var i = 0; i < ft.length; i++) {
            var fl = ft[i].replace(/\s+/g, " ").trim();
            if (fl) feedLinesCache[fl] = 1;
          }
        }
      }
      var feedLines = feedLinesCache;
      var out = [], raw = String(root.innerText || "").split("\n");
      for (var r = 0; r < raw.length; r++) {
        var line = raw[r].replace(/\s+/g, " ").trim();
        if (line && feedLines[line]) continue;
        out.push(raw[r]);
      }
      return out.join("\n");
    }

    function harvestDelta(label) {
      var text = profileText();
      addFrom(text, label);
      var out = [], raw = String(text).split("\n");
      for (var i = 0; i < raw.length && out.length < 40; i++) {
        var line = raw[i].replace(/\s+/g, " ").trim();
        if (!line || line.length < 3 || line.length > 220) continue;
        if (seenLine[line]) continue;
        seenLine[line] = 1;
        if (DOSSIER_CHROME.test(line)) continue;
        if (/^\d+[\d.,]*\s*(followers?|following|friends?|likes?|người theo dõi|bạn bè)$/i.test(line)) continue;
        out.push(line);
      }
      for (var o = 0; o < out.length && aboutLines.length < 120; o++) aboutLines.push(out[o]);
      return out;
    }

    function clickInto(tab) {
      for (var s = 0; s < tab.hrefs.length; s++) {
        var byHref = document.querySelectorAll('a[href*="' + tab.hrefs[s] + '"]');
        for (var h = 0; h < byHref.length; h++) {
          var hb = byHref[h].getBoundingClientRect();
          if (hb.width <= 0 || hb.height <= 0) continue;
          try { byHref[h].click(); } catch (e) { return Promise.resolve(false); }
          return wait(500).then(function () { return settleThenScan(settleMs); }).then(function () { return true; });
        }
      }
      var nodes = document.querySelectorAll('a[role="link"], [role="tab"], [role="button"], [role="listitem"] a');
      for (var i = 0; i < nodes.length; i++) {
        var n = nodes[i];
        var lab = (n.innerText || n.getAttribute("aria-label") || "").replace(/\s+/g, " ").trim();
        if (!tab.label.test(lab)) continue;
        var box = n.getBoundingClientRect();
        if (box.width <= 0 || box.height <= 0) continue;
        try { n.click(); } catch (e) { return Promise.resolve(false); }
        return wait(500).then(function () { return settleThenScan(settleMs); }).then(function () { return true; });
      }
      return Promise.resolve(false); // this profile does not offer the tab
    }

    function enterAbout() {
      var aboutMark = captureMark();
      if (/\/about|sk=about|directory_/i.test(location.href)) { checked.push("about(already)"); return Promise.resolve(true); }
      var re = /^(about|giới thiệu)$/i;
      var byHref = document.querySelectorAll('a[href*="sk=about"], a[href$="/about"], a[href*="/about?"]');
      for (var h = 0; h < byHref.length; h++) {
        var hb = byHref[h].getBoundingClientRect();
        if (hb.width <= 0 || hb.height <= 0) continue;
        try { byHref[h].click(); } catch (e) { break; }
        return wait(600).then(function () { return settleThenScan(settleMs); })
          .then(function () {
            recordGql("about", aboutMark);
            about.about = harvestDelta("about"); checked.push("about"); return true;
          });
      }
      var nodes = document.querySelectorAll('a[role="link"], [role="tab"], [role="button"]');
      for (var i = 0; i < nodes.length; i++) {
        var n = nodes[i];
        var lab = (n.innerText || n.getAttribute("aria-label") || "").replace(/\s+/g, " ").trim();
        if (!re.test(lab)) continue;
        var box = n.getBoundingClientRect();
        if (box.width <= 0 || box.height <= 0) continue;
        try { n.click(); } catch (e) { return Promise.resolve(false); }
        return wait(600).then(function () { return settleThenScan(settleMs); })
          .then(function () {
            recordGql("about", aboutMark);
            about.about = harvestDelta("about"); checked.push("about"); return true;
          });
      }
      return Promise.resolve(false);
    }

    // DISCOVER the sub-nav instead of guessing its slugs. The first live run opened 0 of 5
    // hard-coded tabs: the fixed list bet on `directory_*` (Pages) and `about_*` (profiles) and
    // lost, because a slug this code does not know about is indistinguishable from a tab the
    // profile does not offer — both come back as `missing`. Reading whatever About links are
    // actually on the page removes the guess entirely and survives Facebook renaming them.
    function discoverTabs() {
      var out = [], seen = {}, as = [];
      seenAll = []; skipped = [];
      try { as = document.querySelectorAll('a[href]') || []; } catch (e) { return out; }
      for (var i = 0; i < as.length; i++) {
        var h = as[i].getAttribute ? (as[i].getAttribute("href") || "") : "";
        var m = h.match(/(?:\/|sk=)((?:about|directory)_[a-z_]+)(?:[\/?&#]|$)/i);
        if (!m) continue;
        var slug = m[1].toLowerCase();
        if (seen[slug]) continue;
        var b = as[i].getBoundingClientRect ? as[i].getBoundingClientRect() : { width: 1, height: 1 };
        if (b.width <= 0 || b.height <= 0) continue;
        seen[slug] = 1;
        var key = slug.replace(/^(?:about|directory)_/, "");
        seenAll.push(slug);
        out.push({ key: key, slug: slug });
      }
      // Every section, none skipped — measured at about a second each, so there is no tail worth
      // dropping. Order is still priority, because the BUDGET can still cut: if a profile
      // publishes fourteen sections and the walk runs short, losing Hobbies beats losing the
      // address. Sections outside the known set sort last rather than being dropped.
      out.sort(function (a, b) {
        var ra = DOSSIER_RANK[a.key], rb = DOSSIER_RANK[b.key];
        return (ra === undefined ? 99 : ra) - (rb === undefined ? 99 : rb);
      });
      return out;
    }
    function clickSlug(slug) {
      var as = [];
      try { as = document.querySelectorAll('a[href*="' + slug + '"]') || []; } catch (e) { return Promise.resolve(false); }
      for (var i = 0; i < as.length; i++) {
        var b = as[i].getBoundingClientRect ? as[i].getBoundingClientRect() : { width: 1, height: 1 };
        if (b.width <= 0 || b.height <= 0) continue;
        try { as[i].click(); } catch (e) { return Promise.resolve(false); }
        return wait(300).then(function () { return settleThenScan(settleMs); }).then(function () { return true; });
      }
      return Promise.resolve(false);
    }

    var plan = [], idx = 0;
    function buildPlan() {
      var found = discoverTabs();
      // Fall back to the label/slug list only when the page exposes no About links at all — a
      // profile with its About section closed, or a layout this discovery does not understand.
      plan = found.length ? found : DOSSIER_TABS.map(function (t) { return { key: t.key, tab: t }; });
      discovered = seenAll.slice();
    }
    function step() {
      if (idx >= plan.length) return Promise.resolve();
      // Stop on the BUDGET, never on a hit. One more tab needs a click, a settle and a scan;
      // if that will not fit, stop honestly rather than get killed mid-tab at 45s.
      if (budgetLeft() < settleMs + 1500) {
        budgetExhausted = true;
        for (var r = idx; r < plan.length; r++) missing.push(plan[r].key);
        return Promise.resolve();
      }
      var t = plan[idx++];
      var mark = captureMark();
      var open = t.slug ? clickSlug(t.slug) : clickInto(t.tab);
      return open.then(function (opened) {
        if (!opened) { missing.push(t.key); return wait(150).then(step); }
        // Expand BEFORE harvesting, or the delta records the truncated line and the expanded
        // one never gets read — the harvest only ever sees each line once.
        return Promise.resolve().then(function () {
          recordGql(t.key, mark);
          about[t.key] = harvestDelta(t.key);
          checked.push(t.key);
          return wait(150).then(step);
        });
      });
    }

    // The landing page FIRST: name, category, followers, website, CTA and the Intro card are all
    // on the main profile, and reading them before navigating means the dossier still carries the
    // header even if the About ladder finds nothing (or the budget runs out on the first tab).
    // Promise.resolve, not a bare .then: profileHeader returns a Promise on its normal path but
    // a plain object from its self-profile guard. Calling .then on that throws, the dispatcher's
    // catch turns the throw into a generic error record, and a refusal to scrape the operator's
    // own profile comes back looking like available:true with no reason. Normalise both call
    // sites instead of trusting the shape.
    return Promise.resolve(profileHeader(inputs)).then(function (hdr) {
      // Bail ONLY on the operator guard. profileHeader also reports available:false when it
      // simply could not read a name, and a profile with an unreadable heading is exactly the
      // one whose About tabs are worth walking — treating both the same would discard the walk
      // before it started.
      if (hdr && hdr.reason === "self_profile") return hdr;
      var header = (hdr && hdr.items && hdr.items[0]) || {};
      var mainMark = captureMark();
      // Expand here too even though background.js already ran one pass before injection: that
      // pass happens before this capability exists, so its result is invisible to the probe and
      // a bio that grew a second "See more" after the first click stays cut.
      return Promise.resolve().then(function () {
      recordGql("main", mainMark);
      // NOT harvested yet. Reading the landing page here swept in the review carousel and the
      // reel view counts, and because the harvest records each line once, the GraphQL sections
      // behind it deduped to empty — the clean data arrived and was discarded as duplicate.
      // Only the fallback path needs this text, so only the fallback path takes it.
      if (!info) {
        return { capability: "fb.profile.dossier", schema: "ProfileDossier", available: true, found: false, count: 0,
          items: [], checked: checked, error: "could not resolve a profile base from the url: " + target };
      }
      // Landing straight on /about renders the About OVERVIEW, and the overview already contains
      // the bio, the category and the published contact block. Measured on three pages entered
      // that way: the first harvest returned 40 lines including the email, and the subsequent
      // clicks into contact_info, intro and category each returned ZERO new lines — three
      // clicks, three settles, three scans, nothing gained. Only work and education hold
      // anything the overview does not ("Owner/President at Reach Home Loans", "NMLS: 2266637").
      //
      // So when the job already points at About, do not re-enter it and do not re-read what is
      // on screen. Clicking "About" from an About page was also what dragged the post feed in.
      // DOM by default. The GraphQL path is real but not finished: it names every section
      // correctly, skips Reviews and touches no DOM, yet every replayed response came back
      // data:{} — the section token is being sent in the wrong variable, and collectionToken /
      // sectionToken / rawSectionToken are three separate variables in the captured call. It is
      // kept behind use_graphql because the framework is sound and only that mapping is missing;
      // shipping it as the default would trade working-but-noisy for clean-and-empty.
      //
      // Nine empty sections reported ok:true, failed:0, sections:9. That is why `sizes` and the
      // GraphQL `errors` are now recorded: without them the failure is indistinguishable from
      // success at every level the caller can see.
      function gatherAbout() {
      if (!inputs.use_graphql) { gqlAbout = { ok: false, reason: "graphql_disabled_by_default" }; return walkTheDom(); }
      return aboutViaGraphQL(inputs).then(function (gql) {
        gqlAbout = gql;
        if (gql && gql.ok) {
          for (var sk = 0; sk < gql.order.length; sk++) {
            var key = gql.order[sk], lines = gql.sections[key] || [];
            about[key] = lines;
            checked.push(key);
            for (var li = 0; li < lines.length && aboutLines.length < 200; li++) {
              var ln = lines[li];
              if (seenLine[ln]) continue;
              seenLine[ln] = 1;
              if (DOSSIER_CHROME.test(ln)) continue;
              aboutLines.push(ln);
            }
            addFrom(lines.join("\n"), key);
          }
          panelFound = true;   // the data came from the About query itself, not from a guess
          return null;
        }
        return walkTheDom();
      });
    }

    function walkTheDom() {
      about.main = harvestDelta("main");
      checked.push("main");
      var atAbout = /\/about|sk=about|directory_/i.test(location.href);
      return (atAbout ? Promise.resolve(true) : enterAbout()).then(function (entered) {
        if (atAbout) checked.push("about(already)");
        else if (!entered) missing.push("about");
        // The sub-nav can paint after the text has stopped growing, so one look is not enough:
        // settleThenScan returns on stable LENGTH, and a nav rendering into already-counted
        // space does not move it. Measured cost of the whole walk was 0.9-4.3s against a 30s
        // budget, so a few hundred ms of retry is free — and cheaper than a run that reports
        // every tab missing. Re-look up to three times before giving up on discovery.
        var tries = 0;
        function look() {
          buildPlan();
          if (discovered.length || tries >= 2 || budgetLeft() < 4000) return Promise.resolve();
          tries += 1;
          return wait(500).then(look);
        }
        return look();
      }).then(step);
    }

    // Reached by BOTH paths. When it lived at the tail of the DOM walk, the GraphQL path
    // returned without ever building a record.
    function buildRecord() {
        // A second header read, now that the About sections have arrived. Same DOM parser, more
        // text to parse: "Works at"/"Studied at"/"Lives in" lines the main page truncated are
        // complete here. Union the two, never overwrite — the main page is not always poorer.
        return Promise.resolve(profileHeader(inputs)).then(function (h2) {
          var later = (h2 && h2.items && h2.items[0]) || {};
          function union(a, b) {
            var out = (a || []).slice();
            for (var i = 0; i < (b || []).length; i++) if (out.indexOf(b[i]) === -1) out.push(b[i]);
            return out;
          }
          // work / education / location came back EMPTY on every live profile, and dedup was not
          // why. The header parses them from sentence patterns — "Works at <X>", "Lives in <X>" —
          // and Facebook does not write those: a Page prints "Owner/President at Reach Home
          // Loans" under a Work heading, and an address under an "Address" label. The text was
          // always in about_lines; it just never reached a typed field.
          //
          // So read them by LABEL. That is structural, not interpretive: taking the value under
          // a heading the page itself wrote is nothing like guessing a trade from keywords, and
          // it is why industry stays the agent's call while these do not.
          function sectionValues(key) {
            var raw = (about && about[key]) || [], out = [];
            for (var i = 0; i < raw.length; i++) {
              var L = String(raw[i]).replace(/\s+/g, " ").trim();
              if (!L || L.length > 160) continue;
              if (SECTION_LABEL.test(L)) continue;                    // the heading itself
              if (/^\d+[\d.,]*\s*(followers?|following|likes?)/i.test(L)) continue;
              if (out.indexOf(L) === -1 && out.length < 8) out.push(L);
            }
            return out;
          }
          // A labelled value sits on the NEXT line: "Address" then the address, "Phone" then the
          // number. Searching the flat list keeps this working whichever section rendered it.
          function afterLabel(re) {
            var out = [];
            for (var i = 0; i < aboutLines.length - 1; i++) {
              if (!re.test(String(aboutLines[i]).trim())) continue;
              var v = String(aboutLines[i + 1]).replace(/\s+/g, " ").trim();
              if (v && v.length <= 160 && !SECTION_LABEL.test(v) && out.indexOf(v) === -1) out.push(v);
            }
            return out;
          }
          var item = {
            profile_url: info.base,
            name: header.name || later.name || "",
            category: header.category || later.category || "",
            follower_count: header.follower_count != null ? header.follower_count : later.follower_count,
            verified: !!header.verified,
            // ---- what the profile says it does. TEXT, for an agent to read. ----
            intro_bio: header.intro_bio || later.intro_bio || "",
            work: union(union(header.work, later.work), sectionValues("work")),
            education: union(union(header.education, later.education), sectionValues("education")),
            // "Lives in <X>" is a personal-profile phrasing; a Page states an Address instead.
            location: union(union(header.location, later.location), afterLabel(/^(address|địa chỉ)$/i)),
            phones: afterLabel(/^(phone|mobile|điện thoại|số điện thoại)$/i),
            intro_lines: union(header.intro_lines, later.intro_lines).slice(0, 20),
            // The job TITLE lives here and nowhere else — see the note above DOSSIER_TABS.
            about: about,
            about_lines: aboutLines,
            // ---- how to reach them ----
            emails: emails,
            websites: websites,
            website: header.website || later.website || (websites.length ? websites[0] : ""),
            cta: header.cta || [],
            found_on: foundOn || null,
            // ---- audit trail: what was actually opened, and what was not ----
            checked: checked,
            missing: missing,
            // The About slugs actually found on the page. Empty means discovery saw no About
            // links and the walk fell back to label matching — the difference between "this
            // profile publishes nothing" and "this code no longer recognises the sub-nav",
            // which is the exact confusion that made the first live run unreadable.
            discovered_tabs: discovered,
            // Sections the profile publishes that the walk deliberately did not open. Five
            // sections answer the questions here; the other nine cost a second each and answer
            // none. Listed so the record never reads as "this profile has nothing else".
            skipped_tabs: skipped,
            // Which GraphQL query each surface fired, keyed the same way as `about`. A surface
            // whose content arrives in ONE named query can be replayed by doc_id instead of
            // clicked — that is how fb.profile.hovercard stopped needing a real hover. Pass
            // probe_graphql:true to get the variables as well.
            graphql_by_surface: gqlBySurface,
            see_more_expansions: seeMoreClicks,
            // true when every line in this record came from the About card itself. false means
            // the panel was not located and the text is whatever the main column held — posts,
            // reviews and all. Treat a false here as a reason to distrust about_lines.
            about_panel_found: panelFound,
            // How the sections were obtained. "graphql" means the About query was replayed by
            // doc_id and nothing on the page was clicked or scraped — the only mode in which a
            // review or a post cannot leak in. "dom" means the query never fired and the walk
            // fell back to clicking, so about_lines is worth a second look.
            source: gqlAbout && gqlAbout.ok ? "graphql" : "dom",
            graphql_about: gqlAbout ? { ok: !!gqlAbout.ok, reason: gqlAbout.reason || null,
              tokens_found: gqlAbout.tokens_found || 0, doc_id: gqlAbout.doc_id || null,
              sections: (gqlAbout.order || []).length, failed: (gqlAbout.failed || []).length,
              sizes: gqlAbout.sizes || [], skipped: gqlAbout.skipped || [] } : null,
            _panel_ladder: inputs.debug_panel ? panelLadder() : undefined,
            budget_exhausted: budgetExhausted,
            elapsed_ms: Date.now() - startedAt
          };
          var ok = !!(item.name || emails.length || aboutLines.length);
          return {
            capability: "fb.profile.dossier", schema: "ProfileDossier",
            // available:true whenever the pass RAN. background.js nulls a record whose capability
            // reports unavailable, which is how "walked the whole profile, it really says nothing"
            // became indistinguishable from "never looked" — the exact bug fixed three times in
            // this file already. Emptiness travels in the fields, not by hiding the record.
            available: true, found: ok, count: ok ? 1 : 0,
            items: [item], checked: checked,
            error: ok ? null : "profile opened but yielded no name, address or About text"
          };
        });
    }

    return gatherAbout().then(buildRecord);
      });
    });
  }

  // --- diagnostic: what does HOVERING a profile fire? ----------------------
  // Hovering a friend opens Facebook's preview card, which fetches more about that person.
  // If that fetch is GraphQL, the query can be replayed directly and the whole hover step
  // disappears. gql_intercept already records every GraphQL call, so the only missing piece
  // is triggering the hover and reporting what arrived because of it.
  //
  // The pointer chain here is the same one the reaction flyout uses, which Facebook accepts —
  // a bare element.hover() does not open these cards.
  // _diag.about_sections — can the About sections be REPLAYED instead of clicked?
  //
  // Three attempts at scoping the DOM to the About card have failed, each on a different guess,
  // because Facebook renders About, Reviews, Posts and Photos into one tree and the boundary
  // between them is not something a selector can be relied on to find. A GraphQL response for an
  // About section cannot contain a review: the separation is structural rather than positional.
  //
  // The probe already established that ONE query serves every section
  // (ProfileCometAboutAppSectionQuery, doc_id 27470497829312569) and that its selector is a
  // per-profile, per-section `collectionToken`. What decides whether replay is viable is a
  // question only a live page can answer: does landing on /about surface EVERY section's token,
  // or does each token appear only when its own tab is clicked? If it is the former, the DOM walk
  // can be deleted outright. If the latter, replay saves nothing and the clicking has to stay.
  function diagAboutSections(inputs) {
    inputs = inputs || {};
    var store = window.__soloGql;
    var res = { capability: "_diag.about_sections", schema: "Diagnostic", available: true, count: 0, items: [], diagnostic: {} };
    if (!store || !Array.isArray(store.captures)) { res.diagnostic.error = "no capture store"; return Promise.resolve(res); }

    return settleThenScan(Number(inputs.settle_ms) || 5000).then(function () {
      var caps = store.captures || [], calls = [], tokens = {}, scanned = 0;
      for (var i = 0; i < caps.length; i++) {
        var c = caps[i] || {};
        var v = c.variables || {};
        if (/AboutAppSection/i.test(String(c.queryName || ""))) {
          calls.push({
            query: String(c.queryName || ""), doc_id: String(c.docId || ""), has_dtsg: !!c.fbDtsg,
            variable_keys: Object.keys(v).sort(),
            pageID: v.pageID || null, userID: v.userID || null,
            appSectionFeedKey: v.appSectionFeedKey ? String(v.appSectionFeedKey).slice(0, 40) + "…" : null,
            collectionToken: v.collectionToken ? decodeToken(v.collectionToken) : null,
            sectionToken: v.sectionToken ? decodeToken(v.sectionToken) : null
          });
        }
        // The tokens are base64 in the variables and decode to "app_collection:pfbid…". If the
        // page's own responses already carry every section's token, one load is all it takes.
        var body = "";
        try { body = JSON.stringify(c.response || ""); } catch (e) { body = ""; }
        if (!body) continue;
        scanned += 1;
        var hits = body.match(/YXBwX2NvbGxl[A-Za-z0-9+/=_-]{20,}/g) || [];
        for (var h = 0; h < hits.length; h++) {
          var dec = decodeToken(hits[h]);
          if (!dec) continue;
          if (!tokens[dec]) tokens[dec] = { token: dec, seen_in: [] };
          var qn = String(c.queryName || "?");
          if (tokens[dec].seen_in.indexOf(qn) === -1) tokens[dec].seen_in.push(qn);
        }
      }
      var list = [];
      for (var t in tokens) list.push(tokens[t]);
      res.diagnostic = {
        url: location.href,
        about_query_calls: calls,
        distinct_section_tokens_in_responses: list.length,
        section_tokens: list.slice(0, 25),
        captures_scanned: scanned,
        // The verdict this probe exists for, stated rather than left to be inferred.
        verdict: list.length >= 3
          ? "REPLAYABLE — one page load surfaced " + list.length + " section tokens, so the sections can be fetched by doc_id and the DOM walk deleted"
          : "NOT REPLAYABLE FROM ONE LOAD — only " + list.length + " token(s) appeared without clicking; each section's token seems to arrive with its own click, so replay would save nothing"
      };
      res.count = 1;
      res.items = [res.diagnostic];
      return res;
    });
  }
  function decodeToken(b64) {
    try { return decodeURIComponent(escape(atob(String(b64).replace(/-/g, "+").replace(/_/g, "/")))); }
    catch (e) { try { return atob(String(b64)); } catch (e2) { return null; } }
  }

  function diagHover(inputs) {
    inputs = inputs || {};
    var store = window.__soloGql;
    var res = { capability: "_diag.hover_probe", schema: "Diagnostic", available: true, count: 0, items: [], diagnostic: {} };
    if (!store || !Array.isArray(store.captures)) { res.diagnostic.error = "no capture store"; return Promise.resolve(res); }

    function pev(el, type, Ctor) {
      try { el.dispatchEvent(new Ctor(type, { bubbles: true, cancelable: true, view: window })); }
      catch (e) { try { el.dispatchEvent(new MouseEvent(type.replace("pointer", "mouse"), { bubbles: true, cancelable: true, view: window })); } catch (e2) { /* ignore */ } }
    }
    function hoverEl(el) {
      var P = window.PointerEvent || MouseEvent;
      pev(el, "pointerover", P); pev(el, "mouseover", MouseEvent);
      pev(el, "pointerenter", P); pev(el, "mouseenter", MouseEvent);
      pev(el, "pointermove", P); pev(el, "mousemove", MouseEvent);
    }
    function unhover(el) {
      var P = window.PointerEvent || MouseEvent;
      pev(el, "pointerout", P); pev(el, "mouseout", MouseEvent);
      pev(el, "pointerleave", P); pev(el, "mouseleave", MouseEvent);
    }

    // Friend rows link to a profile. Take the first N distinct profile links that are visible.
    var targets = [], seenHref = {};
    var links = document.querySelectorAll('a[href*="facebook.com/"], a[role="link"][href]');
    for (var i = 0; i < links.length && targets.length < (Number(inputs.count) || 3); i++) {
      var a = links[i], href = a.getAttribute("href") || "";
      if (!href || /\/(groups|photo|watch|events|marketplace)\//i.test(href)) continue;
      if (!/facebook\.com\/[^/?#]+/i.test(href) && href.charAt(0) !== "/") continue;
      var r = a.getBoundingClientRect();
      if (r.width <= 0 || r.height <= 0 || r.top < 0) continue;
      var key = href.split("?")[0];
      if (seenHref[key]) continue;
      seenHref[key] = 1;
      targets.push(a);
    }
    var before = store.captures.length;
    var beforeNames = {};
    for (var b = 0; b < store.captures.length; b++) beforeNames[String(store.captures[b].queryName || "")] = 1;

    var idx = 0;
    function step() {
      if (idx >= targets.length) return Promise.resolve();
      var el = targets[idx++];
      try { el.scrollIntoView({ block: "center" }); } catch (e) { /* ignore */ }
      return wait(300).then(function () {
        hoverEl(el);
        // The card is lazy: it waits out a short intent delay before fetching.
        return wait(Number(inputs.hover_ms) || 2200);
      }).then(function () {
        unhover(el);
        return wait(400).then(step);
      });
    }

    return step().then(function () {
      var added = store.captures.slice(before);
      res.diagnostic = {
        hovered: targets.length,
        hover_targets: targets.map(function (t) { return String(t.getAttribute("href") || "").split("?")[0].slice(0, 80); }),
        captures_before: before,
        captures_after: store.captures.length,
        new_captures: added.length
      };
      res.items = added.map(function (c) {
        var sk = null;
        try { sk = skeletonize(firstChunkOf(c), 0, { n: 700 }, 14); } catch (e) { /* ignore */ }
        return {
          queryName: c.queryName || "",
          docId: c.docId || "",
          is_new_query_name: !beforeNames[String(c.queryName || "")],
          variable_keys: c.variables ? Object.keys(c.variables) : [],
          variables: c.variables || {},
          response_skeleton: sk
        };
      });
      res.count = res.items.length;
      return res;
    });
  }

  // --- fb.profile.hovercard -------------------------------------------------
  // Facebook's hover preview is a plain GraphQL call, CometHovercardQueryRendererQuery, whose
  // only varying input is entityID — measured by hovering three profiles and diffing the
  // capture buffer. So the hover itself is unnecessary: give it an id and it answers.
  // fb.profile.friends already returns each friend's id, which makes the two compose directly.
  //
  // Auth (fb_dtsg, av) is borrowed from any capture the page already made, and the doc_id is
  // taken from Facebook's own module registry so it can never go stale.
  // Any capture the page already made will do for auth: fb_dtsg and av are per-SESSION, not
  // per-query. Prefer one of the same family when there is one, because its `url` is then
  // certainly the endpoint that query is served from.
  function authSeed(store, preferName) {
    var caps = (store && store.captures) || [];
    var preferred = null, anyAuth = null;
    for (var i = caps.length - 1; i >= 0; i--) {
      var c = caps[i];
      if (!c || !c.fbDtsg) continue;
      if (!anyAuth) anyAuth = c;
      if (preferName && String(c.queryName || "").indexOf(preferName) > -1) { preferred = c; break; }
    }
    return preferred || anyAuth;
  }
  function hovercardSeed(store) { return authSeed(store, "CometHovercard"); }
  function profileHovercard(inputs) {
    inputs = inputs || {};
    var store = window.__soloGql;
    var out = { capability: "fb.profile.hovercard", schema: "HovercardRecord", available: true, count: 0, items: [] };
    var entityId = String(inputs.entity_id || inputs.profile_id || "").trim();
    if (!entityId) { out.error = "entity_id is required (fb.profile.friends returns it for every friend)"; return Promise.resolve(out); }
    var seed = hovercardSeed(store);
    if (!seed || typeof store.origFetch !== "function") {
      out.error = "no captured request to borrow auth from — open a Facebook page first";
      return Promise.resolve(out);
    }
    var docId = docIdFromRegistry("CometHovercardQueryRendererQuery")
      || (String(seed.queryName || "").indexOf("CometHovercard") > -1 ? seed.docId : "")
      || "27713673081633221"; // measured 2026-08-16; registry lookup is preferred and tried first
    var vars = {
      actionBarRenderLocation: "WWW_COMET_HOVERCARD",
      context: "DEFAULT",
      entityID: entityId,
      scale: webPixelRatio(),
      __relay_internal__pv__WorkCometIsEmployeeGKProviderrelayprovider: false
    };
    var cap = { queryName: "CometHovercardQueryRendererQuery", docId: docId, fbDtsg: seed.fbDtsg, av: seed.av, url: seed.url, variables: vars };
    return replayPage(store, cap, undefined, null).then(function (resp) {
      var m = mergeStreamed(resp);
      var user = findInChunks(m.chunks, "data.node.comet_hovercard_renderer.user")
        || deepFindKeyObj(m.chunks, "user");
      if (!isObj(user)) {
        out.error = "hovercard returned no user node for entity " + entityId;
        out.response_skeleton = (function () { try { return skeletonize(firstChunkOf({ response: m.chunks }), 0, { n: 600 }, 12); } catch (e) { return null; } })();
        return out;
      }
      // timeline_context_items is what the card actually SHOWS under the name — "Works at X",
      // "Studied at Y", "Lives in Z", "Followed by N people". It is the only part of this
      // payload that says what someone DOES, which is the whole point of asking: an agent
      // cannot tell a realtor from an insurance agent from a name and an avatar.
      // Collected tolerantly (any *.text under each node) because the row shapes differ per
      // item type, and a row missed is a signal lost.
      function contextRows(u) {
        var rows = [], nodes = getPath(u, "timeline_context_items.nodes");
        if (!Array.isArray(nodes)) return rows;
        for (var i = 0; i < nodes.length && rows.length < 12; i++) {
          var n = nodes[i];
          if (!isObj(n)) continue;
          var t = getPath(n, "title.text") || getPath(n, "renderer.context_item.title.text") || "";
          if (!t) { try { t = deepText(n, 0) || ""; } catch (e) { t = ""; } }
          t = String(t).replace(/\s+/g, " ").trim().slice(0, 160);
          if (t && rows.indexOf(t) === -1) rows.push(t);
        }
        return rows;
      }
      var ctx = contextRows(user);
      var rec = {
        id: user.id ? String(user.id) : entityId,
        name: typeof user.name === "string" ? user.name : "",
        url: (typeof user.url === "string" && user.url) ? user.url : (typeof user.profile_url === "string" ? user.profile_url : ""),
        context: ctx,
        work: ctx.filter(function (t) { return /^(works?|worked|làm việc|từng làm)/i.test(t); }),
        education: ctx.filter(function (t) { return /^(stud|went to|học|từng học)/i.test(t); }),
        location: ctx.filter(function (t) { return /^(lives?|from|sống|đến từ)/i.test(t); }),
        gender: getPath(user, "primaryActions.0.client_handler.profile_action.profile_owner.gender")
          || (function () { var g = deepFindKeyStr(user, "gender"); return g || ""; })(),
        is_verified: !!user.is_verified,
        memorialized: !!user.is_visibly_memorialized,
        profile_picture: getPath(user, "profile_picture.uri") || getPath(user, "profile_picture_depth_0.uri") || "",
        bio: getPath(user, "bio_text.text") || getPath(user, "profile_intro_card.bio.text") || "",
        category: getPath(user, "profile_plus_transition_page_category") || getPath(user, "category_name") || "",
        subscribe_status: user.subscribe_status || "",
        friendship_status: user.friendship_status || "",
        mutual_friends: firstNumber(user, { mutual_friends_count: 1, count: 1 }) || 0,
        actions: (function () {
          var acts = [], list = Array.isArray(user.primaryActions) ? user.primaryActions : [];
          for (var i = 0; i < list.length && i < 6; i++) {
            var t = getPath(list[i], "title.text") || getPath(list[i], "label.text") || "";
            if (t) acts.push(String(t));
          }
          return acts;
        })()
      };
      out.items = [rec];
      out.count = 1;
      out.found = !!(rec.name || rec.url);
      // Ship the shape too: this query returns different fields for a person, a page and a
      // stranger, and guessing which ones exist is how a field silently becomes empty.
      if (inputs.debug) {
        try { out.user_skeleton = skeletonize(user, 0, { n: 1400 }, 14); } catch (e) { /* ignore */ }
        try {
          var cn = getPath(user, "timeline_context_items.nodes");
          if (Array.isArray(cn) && cn.length) out.context_node_skeleton = skeletonize(cn[0], 0, { n: 900 }, 14);
        } catch (e) { /* ignore */ }
      }
      return out;
    }).catch(function (e) {
      out.error = "hovercard call failed: " + String(e && e.message || e);
      return out;
    });
  }
  function deepFindKeyStr(obj, key) {
    var found = "";
    (function walk(n, d) {
      if (found || d > 8 || !isObj(n)) return;
      for (var k in n) {
        if (found) return;
        var v = n[k];
        if (k === key && typeof v === "string" && v) { found = v; return; }
        if (isObj(v)) walk(v, d + 1);
        else if (Array.isArray(v)) for (var j = 0; j < v.length && j < 6; j++) if (isObj(v[j])) walk(v[j], d + 1);
      }
    })(obj, 0);
    return found;
  }

  function deepFindKeyObj(response, key) {
    var chunks = chunksOf(response), found = null;
    function walk(n, d) {
      if (found || d > 10 || !isObj(n)) return;
      for (var k in n) {
        if (found) return;
        var v = n[k];
        if (k === key && isObj(v) && (v.id || v.name)) { found = v; return; }
        if (isObj(v)) walk(v, d + 1);
      }
    }
    for (var i = 0; i < chunks.length && !found; i++) walk(chunks[i], 0);
    return found;
  }


  // --- fb.post.comments -----------------------------------------------------
  // Comments do NOT travel with a post. Not in a group feed, not in a search result: the
  // story node carries a total_count and nothing else. Facebook fetches them with a SEPARATE
  // persisted query per post, which is what this capability replays —
  //   CommentsListComponentsPaginationQuery   top-level comments
  //   Depth1CommentsListPaginationQuery       replies beneath one comment
  // walking has_next_page/end_cursor until the connection ends or a budget stops it.
  //
  // No DOM, no scrolling, and the post is never opened. The only input is the story's
  // feedback id, which every PostRecord now carries — so a search result composes straight
  // into this without a second page load.
  var COMMENT_QUERY = "CommentsListComponentsPaginationQuery";
  var REPLY_QUERY = "Depth1CommentsListPaginationQuery";
  // There is deliberately NO default token literal here. The value is per-story, published by
  // Facebook in the story's own selectable_intents (see storyCommentIntent) — an invented
  // constant would either be rejected or silently swapped for the server's ranked/filtered
  // default, which returns a subset that looks exactly like a complete thread. When the story
  // published none, null is sent, which is what the reference implementation does.
  var COMMENT_INTENT = null;
  // Relay merges this provider into every request it makes. Read from the artifact when the
  // module is loaded; kept here as a floor so a hand-built request is never short one variable.
  var UFI_PROVIDER = "__relay_internal__pv__CometUFIReactionEnableShortNamerelayprovider";
  var COMMENT_EDGES_PATH = "data.node.comment_rendering_instance_for_feed_location.comments.edges";
  var COMMENT_PAGEINFO_PATH = "data.node.comment_rendering_instance_for_feed_location.comments.page_info";
  var REPLY_EDGES_PATH = "data.node.replies_connection.edges";
  var REPLY_PAGEINFO_PATH = "data.node.replies_connection.page_info";
  // Sibling names Facebook has served this connection under. Probed and REPORTED when the
  // primary name is absent from the registry, so a rename shows up as a named alternative
  // instead of an empty result nobody can explain.
  var COMMENT_QUERY_CANDIDATES = [
    "CommentsListComponentsPaginationQuery", "Depth1CommentsListPaginationQuery",
    "CometUFICommentsProviderPaginationQuery", "CometUFICommentRefetchQuery",
    "UFI2CommentsProviderPaginationQuery", "CommentListComponentsRootQuery"
  ];
  function registryProbe(names) {
    var rows = [];
    for (var i = 0; i < names.length; i++) {
      rows.push({ query: names[i], doc_id: docIdFromRegistry(names[i]) || "" });
    }
    return rows;
  }

  // A persisted query that is REJECTED answers 200 with an `errors` array, so a caller that
  // only looks at the data path sees "no comments" for what is really a malformed request.
  //
  // But `errors` and `data` also arrive TOGETHER: with @defer/@stream, one failed fragment
  // populates errors while the rest of the payload is perfectly good. So callers must treat this
  // as fatal only when nothing usable came back, and otherwise record it and keep the rows.
  function gqlErrorOf(chunks) {
    for (var i = 0; i < chunks.length; i++) {
      var e = isObj(chunks[i]) && chunks[i].errors;
      if (Array.isArray(e) && e.length) {
        return String((e[0] && (e[0].message || e[0].summary)) || "graphql error").slice(0, 300);
      }
    }
    return "";
  }

  // Take the CONNECTION at the measured path; if Facebook moved it, accept the first connection
  // whose edges hold comment-shaped nodes (an author or a body). Same principle as the feed
  // extractors: never pick an edge array by NAME, pick it by shape.
  //
  // It returns the connection's OWN page_info, never one searched for separately. A comment page
  // carries a nested replies_connection under every threaded comment, each with its own
  // page_info — so a payload-wide search for "the page_info" can hand back a REPLY thread's
  // cursor and send the top-level walk down a different connection, silently.
  function pickCommentConnection(resp, edgesPath) {
    var m = mergeStreamed(resp);
    var connPath = String(edgesPath).replace(/\.edges$/, "");
    for (var i = 0; i < m.chunks.length; i++) {
      var conn = isObj(m.chunks[i]) ? getPath(m.chunks[i], connPath) : null;
      if (isObj(conn) && Array.isArray(conn.edges) && conn.edges.length) {
        return { edges: conn.edges, page_info: isObj(conn.page_info) ? conn.page_info : null,
                 via: "direct+merged" + m.merged, chunks: m.chunks };
      }
    }
    var found = null, scanned = 0;
    function walk(n, d) {
      if (found || d > 10 || !isObj(n)) return;
      // Check BEFORE descending, so the outer connection wins over the reply connections nested
      // inside its own edges.
      if (Array.isArray(n.edges) && n.edges.length) {
        scanned += 1;
        var n0 = n.edges[0] && n.edges[0].node;
        if (isObj(n0) && (isObj(n0.author) || isObj(n0.body))) { found = n; return; }
      }
      for (var k in n) {
        if (found) return;
        var v = n[k];
        if (isObj(v) && !Array.isArray(v)) walk(v, d + 1);
        else if (Array.isArray(v)) { for (var j = 0; j < v.length && j < 30 && !found; j++) walk(v[j], d + 1); }
      }
    }
    for (var c = 0; c < m.chunks.length && !found; c++) walk(m.chunks[c], 0);
    if (found) {
      return { edges: found.edges, page_info: isObj(found.page_info) ? found.page_info : null,
               via: "scanned_of_" + scanned, chunks: m.chunks };
    }
    return { edges: [], page_info: null, via: scanned ? "no_comment_shaped_edges_of_" + scanned : "no_edges", chunks: m.chunks };
  }

  function commentRecord(node, feedbackId) {
    if (!isObj(node)) return null;
    var text = getPath(node, "body.text");
    if (typeof text !== "string" || !text) text = getPath(node, "preferred_body.text");
    if (typeof text !== "string" || !text) text = deepText(node.body, 0) || deepText(node.preferred_body, 0) || "";
    return {
      id: String(node.id || ""),
      post_feedback_id: String(feedbackId || ""),
      // The whole point of asking: who said it. actorRef keeps id/name/url together so a name
      // can never end up paired with the wrong profile id.
      //
      // Facebook's field is `author`, but this one MUST be called `actor`: the bridge redacts
      // any field whose name contains "auth", so shipping it as `author` would strip the
      // commenter's identity somewhere downstream and leave a body of text with nobody
      // attached — the exact information we opened this query to get.
      actor: actorRef(isObj(node.author) ? node.author : null),
      text: String(text).slice(0, 4000),
      // Read DIRECTLY, not by deep search — a deep search for created_time inside a comment
      // finds a nested reply's timestamp and stamps it on the parent.
      created_time: typeof node.created_time === "number" ? node.created_time : 0,
      depth: typeof node.depth === "number" ? node.depth : 0,
      url: typeof node.url === "string" ? node.url : "",
      reply_count: coerceCount(getPath(node, "feedback.replies_fields.total_count")) || 0,
      // The comment's OWN feedback id — what the reply query is addressed to. Distinct from
      // post_feedback_id above, and from the comment id: all three are different handles.
      feedback_id: (function () { var v = getPath(node, "feedback.id"); return typeof v === "string" ? v : ""; })(),
      replies: []
    };
  }

  function postComments(inputs) {
    inputs = inputs || {};
    var store = window.__soloGql;
    var out = { capability: "fb.post.comments", schema: "CommentRecord[]", available: true, found: false, count: 0, items: [] };

    // The ordering token is per-STORY, so a target is a pair, not a bare id. Accept both forms:
    // a plain id (no token known) and { feedback_id, comment_intent } as PostRecord carries them.
    var ids = [];
    // RESUMING. A thread bigger than one job's budget cannot be drained in one pass, and the
    // report already hands back has_next_page and an end_cursor saying where it stopped — but
    // without a way to feed that back, every later job re-walks from the head and returns the
    // same first rows again. That is the shape of "we collected it four times" rather than "we
    // collected it", and page_cap is not a rare corner: Facebook pages comments about ten at a
    // time, so a 40-comment thread already needs four legs at the default budget.
    //
    // A cursor is per-POST, so a multi-post job resumes through start_cursors keyed by feedback
    // id. Unlike the feed capabilities there is no head-page problem to work around here: page 1
    // is simply the query with a null cursor, so resuming is just starting from a non-null one.
    var startCursors = isObj(inputs.start_cursors) ? inputs.start_cursors : {};
    function pushTarget(v) {
      if (typeof v === "string" && v) v = { feedback_id: v };
      if (!isObj(v) || !v.feedback_id) return;
      var id = String(v.feedback_id);
      var cur = v.start_cursor || startCursors[id] || null;
      ids.push({
        id: id,
        intent: v.comment_intent ? String(v.comment_intent) : null,
        start: typeof cur === "string" && cur ? cur : null
      });
    }
    if (Array.isArray(inputs.feedback_ids)) inputs.feedback_ids.forEach(pushTarget);
    else if (inputs.feedback_id) {
      pushTarget({ feedback_id: inputs.feedback_id, comment_intent: inputs.comment_intent,
                   start_cursor: inputs.start_cursor });
    }
    if (!ids.length) {
      out.error = "feedback_id (or feedback_ids[]) is required — every PostRecord from fb.group.search_posts / fb.group.posts carries one";
      return Promise.resolve(out);
    }
    var seed = authSeed(store, "");
    if (!seed || !store || typeof store.origFetch !== "function") {
      out.error = "no captured request to borrow auth from — load a Facebook page first";
      return Promise.resolve(out);
    }
    var docId = docIdFromRegistry(COMMENT_QUERY) || String(inputs.comment_doc_id || "");
    if (!docId) {
      // The comment modules are loaded on demand: a screen that never renders a UFI may not
      // have them. Say so, and hand back what the registry DOES hold, so the next run knows
      // which name to ask for instead of guessing again.
      out.error = COMMENT_QUERY + " is not in this page's module registry (no screen here has rendered comments). Pass comment_doc_id, or run this on a page that shows a post.";
      out.registry_probe = registryProbe(COMMENT_QUERY_CANDIDATES);
      return Promise.resolve(out);
    }

    // NOT called intent_token. The bridge's sanitizer redacts any key containing "token" (as
    // well as auth/secret/session/csrf), recursively, on BOTH job inputs and collected records —
    // so a job passing intent_token would reach the extension holding the literal string
    // "[redacted]" and every request would be rejected, with nothing anywhere saying why.
    // A job-level token overrides every story's own; otherwise each post uses what IT published,
    // and null when it published nothing. null is a MEANINGFUL value here — see COMMENT_INTENT.
    var intentOverride = inputs.comment_intent ? String(inputs.comment_intent) : null;
    var maxPages = Math.max(1, Math.min(20, inputs.max_comment_pages != null ? inputs.max_comment_pages : 5));
    var maxComments = Math.max(1, Math.min(500, inputs.max_comments != null ? inputs.max_comments : 60));
    // depth counts LEVELS OF COMMENT, not levels of reply. depth:1 — the default — is the
    // direct comments on the post and nothing beneath them; depth:2 adds their replies; 3 adds
    // replies of replies. The ceiling is 4 because Facebook's own client stops recursing there.
    //
    // The default is 1 because each extra level costs one request PER COMMENT: a post with 40
    // comments becomes 40 extra requests the moment replies are asked for. Opt in per job.
    var depth = inputs.depth != null ? inputs.depth
      : (inputs.reply_depth != null ? Number(inputs.reply_depth) + 1 : 1);
    depth = Math.max(1, Math.min(4, Number(depth) || 1));
    var replyDepth = depth - 1;
    var replyDocId = replyDepth > 0 ? (docIdFromRegistry(REPLY_QUERY) || String(inputs.reply_doc_id || "")) : "";

    var all = [], perPost = [], notes = [];

    function capFor(queryName, id, vars) {
      return { queryName: queryName, docId: queryName === REPLY_QUERY ? replyDocId : docId,
               fbDtsg: seed.fbDtsg, av: seed.av, url: seed.url, variables: vars };
    }

    function commentPage(feedbackId, cursor, intent) {
      var vars = {
        commentsAfterCount: -1,
        commentsAfterCursor: cursor || null,
        commentsBeforeCount: null,
        commentsBeforeCursor: null,
        commentsIntentToken: intent,
        feedLocation: "DEDICATED_COMMENTING_SURFACE",
        focusCommentID: null,
        scale: webPixelRatio(),
        useDefaultActor: false,
        id: feedbackId
      };
      vars[UFI_PROVIDER] = true;
      return replayPage(store, capFor(COMMENT_QUERY, feedbackId, vars), undefined, null);
    }

    // The reply query's variable set is NOT the top-level one with different names. It takes
    // exactly these eleven keys — no intent token at all — and its `id` is the parent comment's
    // FEEDBACK id (node.feedback.id), not the comment id. Passing the comment id addresses a
    // node that has no replies_connection, so the reply walk answers empty for every comment
    // while reporting success.
    function replyPage(commentFeedbackId, expansionToken, cursor) {
      var vars = {
        clientKey: null,
        expansionToken: expansionToken || null,
        feedLocation: "DEDICATED_COMMENTING_SURFACE",
        focusCommentID: null,
        repliesAfterCount: null,
        repliesAfterCursor: cursor || null,
        repliesBeforeCount: null,
        repliesBeforeCursor: null,
        scale: webPixelRatio(),
        useDefaultActor: false,
        id: commentFeedbackId
      };
      vars[UFI_PROVIDER] = true;
      return replayPage(store, capFor(REPLY_QUERY, commentFeedbackId, vars), undefined, null);
    }

    // Replies for ONE comment, then recursively for its own replies while the budget allows.
    // Best-effort by design: a failed reply fetch is recorded and the top-level comment is
    // still returned, because losing the parent to a missing child is the worse trade.
    function loadReplies(rec, node, level, feedbackId) {
      if (level > replyDepth || !replyDocId || !rec.reply_count) return Promise.resolve();
      // Addressed by the comment's feedback id. Without one there is nothing to ask, and saying
      // so beats returning a comment that claims N replies and carries none.
      if (!rec.feedback_id) {
        if (notes.length < 8) notes.push("replies " + rec.id + ": comment carries no feedback.id");
        return Promise.resolve();
      }
      var token = getPath(node, "feedback.expansion_info.expansion_token")
        || getPath(node, "expansion_info.expansion_token") || null;
      var cursor = null, pages = 0;
      function stepReply() {
        if (pages >= maxPages || all.length >= maxComments) return Promise.resolve();
        pages += 1;
        return replyPage(rec.feedback_id, token, cursor).then(function (resp) {
          // fetch does not reject on 4xx/5xx, and parseResponse answers null for anything that
          // is not GraphQL JSON — a login redirect, a checkpoint interstitial, a rate-limit page.
          // Left unchecked, all of those read downstream as "this comment has no replies".
          if (resp === null || resp === undefined) {
            if (notes.length < 8) notes.push("replies " + rec.id + ": the server did not answer with GraphQL JSON (login, checkpoint, rate limit or 5xx)");
            return;
          }
          var chunks = chunksOf(resp);
          var err = gqlErrorOf(chunks);
          var got = pickCommentConnection(resp, REPLY_EDGES_PATH);
          if (err && !got.edges.length) { if (notes.length < 8) notes.push("replies " + rec.id + ": " + err); return; }
          if (err && notes.length < 8) notes.push("replies " + rec.id + " (partial): " + err);
          if (!rec.replies_via) rec.replies_via = got.via;
          var kids = [];
          for (var i = 0; i < got.edges.length; i++) {
            var kn = got.edges[i] && got.edges[i].node;
            var kr = commentRecord(kn, feedbackId);
            if (!kr || !kr.id) continue;
            kr.top_level = false;
            rec.replies.push(kr);
            all.push(kr);
            kids.push({ rec: kr, node: kn });
            if (all.length >= maxComments) break;
          }
          // The page_info of the connection the rows actually came from — never one searched for
          // separately, which on a threaded payload can be a nested reply thread's.
          var pi = got.page_info || {};
          cursor = pi.end_cursor || null;
          var more = !!(pi.has_next_page && cursor && all.length < maxComments);
          // Depth first for this page's children, then the next page of siblings.
          var deeper = kids.map(function (k) { return function () { return loadReplies(k.rec, k.node, level + 1, feedbackId); }; });
          return deeper.reduce(function (chain, fn) { return chain.then(fn); }, Promise.resolve())
            .then(function () { return more ? wait(300).then(stepReply) : undefined; });
        }).catch(function (e) {
          if (notes.length < 8) notes.push("replies " + rec.id + ": " + String(e && e.message || e));
        });
      }
      return stepReply();
    }

    function loadPost(target) {
      var feedbackId = target.id;
      var intent = intentOverride || target.intent || COMMENT_INTENT;
      // A resuming leg starts from the caller's cursor and returns exactly its own slice. A leg
      // boundary IS a page boundary — it is Facebook's own opaque cursor replayed into the same
      // query — so there is no overlap to dedupe and no gap to discover later.
      var cursor = target.start || null, pages = 0, mine = [], via = "", stopped = "end_of_connection", hasNext = false;
      function step() {
        if (pages >= maxPages || all.length >= maxComments) {
          stopped = pages >= maxPages ? "page_cap" : "comment_cap";
          return Promise.resolve();
        }
        pages += 1;
        return commentPage(feedbackId, cursor, intent).then(function (resp) {
          // A 200 carrying a login page, a checkpoint, or a rate-limit body parses to null. That
          // is a transport failure and it must NOT be reported as "this post has no comments" —
          // the two are indistinguishable downstream and only one of them is worth retrying.
          if (resp === null || resp === undefined) {
            stopped = "unreadable_response";
            if (notes.length < 8) notes.push(feedbackId + ": the server did not answer with GraphQL JSON (login, checkpoint, rate limit or 5xx)");
            return;
          }
          var chunks = chunksOf(resp);
          var err = gqlErrorOf(chunks);
          var got = pickCommentConnection(resp, COMMENT_EDGES_PATH);
          // Fatal only when nothing usable arrived. A partial @defer failure carries real rows
          // alongside its errors, and throwing those away loses comments for a fragment nobody
          // asked about.
          if (err && !got.edges.length) {
            stopped = "graphql_error";
            if (notes.length < 8) notes.push(feedbackId + ": " + err);
            return;
          }
          if (err && notes.length < 8) notes.push(feedbackId + " (partial): " + err);
          via = via || got.via;
          var fresh = [];
          for (var i = 0; i < got.edges.length; i++) {
            var n = got.edges[i] && got.edges[i].node;
            var rec = commentRecord(n, feedbackId);
            if (!rec || !rec.id) continue;
            rec.top_level = true;
            mine.push(rec); all.push(rec);
            fresh.push({ rec: rec, node: n });
            if (all.length >= maxComments) break;
          }
          var pi = got.page_info || {};
          cursor = pi.end_cursor || null;
          hasNext = !!pi.has_next_page;
          var kids = fresh.map(function (f) { return function () { return loadReplies(f.rec, f.node, 1, feedbackId); }; });
          return kids.reduce(function (chain, fn) { return chain.then(fn); }, Promise.resolve()).then(function () {
            if (hasNext && cursor && all.length < maxComments) return wait(350).then(step);
            // Say which of the three it was. `stopped` initialises to end_of_connection, so a
            // walk cut short by the budget while has_next_page was still true would otherwise
            // report the thread as finished — the caller then has no reason to come back for the
            // rest, and no way to know there is a rest.
            if (!hasNext) stopped = "end_of_connection";
            else if (all.length >= maxComments) stopped = "comment_cap";
            else if (!cursor) stopped = "no_cursor_despite_has_next_page";
          });
        }).catch(function (e) {
          stopped = "fetch_failed";
          if (notes.length < 8) notes.push(feedbackId + ": " + String(e && e.message || e));
        });
      }
      return step().then(function () {
        perPost.push({ feedback_id: feedbackId, count: mine.length, pages_fetched: pages,
                       via: via || "no_response", stopped_because: stopped,
                       comment_intent: intent, start_cursor: target.start || null,
                       has_next_page: hasNext, end_cursor: cursor || null,
                       // The handle for the next leg. has_next_page:false is the only honest
                       // end-of-thread signal — a short page is not one, and neither is page_cap.
                       resumable: !!(hasNext && cursor) });
      });
    }

    // Serial, not parallel: these are authenticated writes to the same session and firing a
    // dozen at once is exactly the shape that trips a rate fuse.
    return ids.reduce(function (chain, t) {
      return chain.then(function () { return loadPost(t); });
    }, Promise.resolve()).then(function () {
      out.items = all;
      out.count = all.length;
      out.found = all.length > 0;
      out.by_post = perPost;
      out.doc_id = docId;
      out.reply_doc_id = replyDocId || "";
      out.depth = depth;
      if (!out.found) out.reason = notes.length ? "query_rejected" : "no_comments";
      if (notes.length) out.notes = notes;
      return out;
    });
  }

  var DOM_CAPABILITIES = {
    "_diag.hover_probe": diagHover, "_diag.about_sections": diagAboutSections,
    "fb.profile.hovercard": profileHovercard, "fb.post.comments": postComments,
    "fb.reels.feed": reelsCollect, "web.search": webSearch, "fb.profile.header": profileHeader,
    "fb.profile.contacts": profileContacts, "fb.profile.dossier": profileDossier,
    "fb.profile.enrich": profileEnrich };

  // The typed extraction is PASSIVE — it reads whatever GraphQL response is already in
  // the ring buffer. On a search/list page the results query (e.g.
  // SearchCometResultsPaginatedResultsQuery) normally fires on load, but it can be
  // MISSED when the tab is busy (verified: the operator actively using Messenger in the
  // SAME tab left 32 people-searches with no SearchComet capture → records null → the
  // flow fell back to the DOM scan and leaked the chat contact). So before extracting,
  // actively ensure the scoped query fired: scroll the results feed + wait, retrying
  // until the capture appears (or a small cap). No-op when the capture is already there.
  function hasCaptureForScope(scope) {
    var caps = (window.__soloGql && window.__soloGql.captures) || [];
    for (var i = caps.length - 1; i >= 0; i--) {
      var c = caps[i];
      if (c && c.response && (!scope || String(c.queryName || "").indexOf(scope) !== -1)) return true;
    }
    return false;
  }
  function scrollResultsFeed() {
    var el = document.querySelector('[role="feed"]');
    if (!el) {
      var best = null, bestH = 0, nodes = document.querySelectorAll("div");
      for (var i = 0; i < nodes.length && i < 4000; i++) {
        var n = nodes[i], sh = n.scrollHeight, ch = n.clientHeight;
        if (sh > ch + 200 && ch > 300 && sh > bestH) { bestH = sh; best = n; }
      }
      el = best;
    }
    try { if (el) el.scrollTop = el.scrollHeight; } catch (e) { /* ignore */ }
    try { window.scrollBy(0, Math.round((window.innerHeight || 800) * 0.9)); } catch (e) { /* ignore */ }
  }
  // A search that genuinely matches NOBODY never fires the paginated results query
  // (there is nothing to page through), so retrying is pure waste — measured at ~8s
  // per lead across a 131-lead batch. Detect Facebook's empty state and bail early.
  function looksLikeNoResults() {
    var t = (document.body ? document.body.innerText : "") || "";
    return /we (?:did ?n[o']?t|could ?n[o']?t) find any results|no results found|check the spelling or try different keywords|không tìm thấy kết quả|hãy thử từ khóa khác/i.test(t);
  }
  function ensureCapture(scope, maxTries, stepMs) {
    if (!scope || hasCaptureForScope(scope)) return Promise.resolve(true);
    var tries = 0;
    function loop() {
      if (hasCaptureForScope(scope)) return Promise.resolve(true);
      if (tries >= maxTries) return Promise.resolve(false);
      // Only after a couple of tries, so a slow render is not mistaken for empty.
      if (tries >= 2 && looksLikeNoResults()) return Promise.resolve(false);
      tries += 1;
      scrollResultsFeed();
      return wait(stepMs).then(loop);
    }
    return loop();
  }

  // Extract page-1 from natural captures, then replay forward until has_next_page
  // is false or max_pages is reached. inputs.max_pages (default 8, cap 40).
  var __soloGqlPaginateImpl = function (capabilityId, inputs) {
    inputs = inputs || {};
    if (DOM_CAPABILITIES[capabilityId]) {
      // available:true even on failure. Reporting a broken DOM capability as unavailable makes
      // background.js null the whole record, so "this capability crashed" and "this page has
      // nothing" become the same output — which is how a capability that never completed once
      // looked exactly like an empty profile. Also catch the PROMISE rejection: the old try
      // only guarded the synchronous call, so anything that failed after the first await
      // escaped and left a null record with no explanation at all.
      function domFail(e) {
        return { capability: capabilityId, available: true, count: 0, items: [{ capability: capabilityId, status: "error", error: String(e && e.message || e) }], error: String(e && e.message || e) };
      }
      try { return Promise.resolve(DOM_CAPABILITIES[capabilityId](inputs)).catch(domFail); }
      catch (e) { return Promise.resolve(domFail(e)); }
    }
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
    var seed = null, seedInfo = null, scoped = 0, noIdentity = 0;
    for (var i = caps.length - 1; i >= 0; i--) {
      var c = caps[i];
      if (!c || !c.response) continue;
      if (cfg.scope && String(c.queryName || "").indexOf(cfg.scope) === -1) continue;
      scoped += 1;
      if (!c.docId || !c.fbDtsg) { noIdentity += 1; continue; }
      var info = resolvePageInfo(c.response, cfg.pageInfoPath);
      if (info) { seed = c; seedInfo = info; break; }
    }
    if (!seed) {
      // Say WHY instead of returning a quietly short list — a silent bail here is exactly
      // what hid the broken group-feed pagination.
      base.pagination_skipped = !scoped ? "no_capture_in_scope"
        : noIdentity === scoped ? "capture_missing_doc_id_or_fb_dtsg"
        : "no_page_info_in_response";
      base.pagination_scope = cfg.scope;
      base.pagination_candidates = scoped;
      base.pagination_expected_path = cfg.pageInfoPath;
      return Promise.resolve(base);
    }

    // A RESUMING leg starts empty. base.items comes from the capture sitting on the page, and on
    // a list screen that capture IS the head of the connection — so carrying it forward would
    // staple the first rows onto every leg, and leg 16 of a 5,000-friend walk would still ship
    // rows 1-8. The caller already has them from leg 1. Each leg returning exactly its own slice
    // is what makes "from here to there" true rather than approximately true.
    var resuming = typeof inputs.start_cursor === "string" && !!inputs.start_cursor;
    var items = resuming ? [] : base.items.slice();
    var seen = {};
    items.forEach(function (it) { if (it && it.id) seen[it.id] = 1; });
    var pi = seedInfo || {};
    // RESUMING. A connection larger than one job's page budget cannot be drained in one pass —
    // 5,000 friends is ~623 pages at 8 per page, and both the 40-page cap and the 45-second
    // capability timeout stop long before that. Without a way to say where to continue, a second
    // job re-walks from the head and returns the same first 336 rows again, which is the shape of
    // "we collected it 16 times" rather than "we collected it".
    //
    // So a caller may hand back the end_cursor of the previous leg. It is Facebook's own opaque
    // cursor, replayed into the same persisted query, so a leg boundary is exactly a page
    // boundary — no overlap to dedupe, no gap to notice later.
    var startCursor = typeof inputs.start_cursor === "string" && inputs.start_cursor ? inputs.start_cursor : null;
    var state = { cursor: startCursor || pi.end_cursor, hasNext: startCursor ? true : !!pi.has_next_page,
                  pages: 0, added: 0, head: 0, headVia: startCursor ? "skipped_resuming" : "not_run" };

    // PAGE 1 NEVER REACHES THIS LAYER. Facebook server-renders the top of a feed into the
    // initial document, so nothing is fetched for it and gql_intercept — which hooks
    // fetch/XHR — never sees it. The only feed query captured on a group page is
    // GroupsCometFeedRegularStoriesPaginationQuery, which fires on SCROLL and walks BACKWARDS
    // in time. Measured: the extractor returned "Post 2" and "Post 3" while the newest post
    // sat on screen unseen. Replaying the same persisted query with a NULL cursor asks for the
    // head of the connection, bringing the newest posts back through GraphQL — where a story's
    // id, url and text come from ONE node and cannot be mispaired, unlike anything rebuilt
    // from the DOM, where a comment's text can end up attached to the post it sits under.
    function headPage() {
      return replayPage(store, seed, null, capabilityId).then(function (resp) {
        if (!resp) return;
        var got = extractReplayItems(resp, capabilityId, seed, cfg.pageInfoPath);
        state.headVia = got.via;
        got.items.forEach(function (it) {
          // A feed connection also carries non-story edges — the sort control came back as a
          // record titled "Most relevant" with no permalink. Anything without a url is not a
          // post, and a post with no url is useless downstream anyway: nothing can act on it.
          if (it && it.id && it.url && !seen[it.id]) { seen[it.id] = 1; items.push(it); state.head += 1; }
        });
        // The head reply carries its own page_info. Continue forward from THERE — the
        // captured seed's cursor described a slice further down the feed, and trusting its
        // has_next_page could stop the walk before the newest pages are drained.
        var hinfo = resolvePageInfo(resp, cfg.pageInfoPath);
        if (hinfo && hinfo.end_cursor) { state.cursor = hinfo.end_cursor; state.hasNext = !!hinfo.has_next_page; }
      }).catch(function () { state.headVia = "replay_failed"; });
    }

    function step() {
      if (!state.hasNext || !state.cursor || state.pages >= maxPages) return Promise.resolve();
      state.pages += 1;
      return replayPage(store, seed, state.cursor, capabilityId).then(function (resp) {
        if (!resp) { state.hasNext = false; return; }
        var got = extractReplayItems(resp, capabilityId, seed, cfg.pageInfoPath);
        got.items.forEach(function (it) {
          if (it && it.id && !seen[it.id]) { seen[it.id] = 1; items.push(it); state.added += 1; }
        });
        // A replayed page nests and streams just like the captured one.
        var pinfo = resolvePageInfo(resp, cfg.pageInfoPath) || {};
        state.cursor = pinfo.end_cursor;
        state.hasNext = !!pinfo.has_next_page;
        return wait(400 + Math.floor((state.pages % 3) * 150)).then(step); // gentle pacing
      }).catch(function () { state.hasNext = false; });
    }

    // The head page is the FIRST leg's job only. Re-fetching it while resuming would re-add rows
    // the caller already has and, worse, overwrite state.cursor with the head's page_info —
    // sending the walk back to the start of the connection every time.
    return (startCursor ? Promise.resolve() : headPage()).then(step).then(function () {
      base.items = items;
      base.count = items.length;
      base.paginated = true;
      base.pages_fetched = state.pages;
      base.added_by_pagination = state.added;
      base.added_by_head_page = state.head;
      base.head_page_via = state.headVia;
      base.available = base.count > 0;
      // Whether the walk stopped because the feed ended or because it ran out of budget. With
      // a time window this is the difference between "nothing else in range" and "I gave up
      // early" — a caller cannot tell them apart from a short list alone.
      base.page_cap_hit = !!(state.hasNext && state.cursor && state.pages >= maxPages);
      base.max_pages = maxPages;
      // The handle for the next leg. `has_next_page:false` is the ONLY honest end-of-connection
      // signal — a short page is not one, and neither is page_cap_hit, which means the budget ran
      // out with more waiting. A caller that stops on anything else stops early and cannot tell.
      base.start_cursor = startCursor;
      base.end_cursor = state.cursor || null;
      base.has_next_page = !!state.hasNext;
      base.resumable = !!(state.hasNext && state.cursor);
      base = applyTimeWindow(base, inputs);
      // The base extraction runs BEFORE the head fetch and the pagination walk, so on a screen
      // whose natural capture held nothing it stamps reason:"no_match" — which then survived
      // onto a result carrying five posts. Clear it once anything arrived, or every consumer
      // reading `reason` is told the opposite of what happened.
      if (base.available && base.reason === "no_match") delete base.reason;

      // with_comments: N — attach comments to the first N posts within the SAME job. Comments
      // are a separate query per post, so this costs N extra requests, not N page loads. The
      // alternative is shipping the feedback ids back to the caller and starting a second job,
      // which is a round trip per post for data we can already reach from here.
      if (inputs.with_comments && Array.isArray(base.items) && base.items.length) {
        var take = inputs.with_comments === true ? 5
          : Math.max(0, Math.min(25, Number(inputs.with_comments) || 0));
        var targets = base.items.slice(0, take).filter(function (it) { return it && it.feedback_id; });
        if (!targets.length) { base.comments_skipped = "no_feedback_id_on_items"; return base; }
        return postComments({
          // Pairs, not bare ids: the ordering token is per-story and the record already has it.
          feedback_ids: targets.map(function (it) {
            return { feedback_id: it.feedback_id, comment_intent: it.comment_intent };
          }),
          max_comment_pages: inputs.max_comment_pages, max_comments: inputs.max_comments,
          depth: inputs.depth, reply_depth: inputs.reply_depth, comment_intent: inputs.comment_intent,
          comment_doc_id: inputs.comment_doc_id, reply_doc_id: inputs.reply_doc_id,
          start_cursors: inputs.start_cursors
        }).then(function (cres) {
          var byPost = {};
          (cres.items || []).forEach(function (c) {
            // A reply already sits inside its parent's `replies`; re-attaching it at the top
            // would double every threaded comment.
            if (!c.top_level) return;
            (byPost[c.post_feedback_id] = byPost[c.post_feedback_id] || []).push(c);
          });
          targets.forEach(function (it) { it.comments = byPost[it.feedback_id] || []; });
          base.comments_count = cres.count;
          base.comments_by_post = cres.by_post;
          if (cres.error) base.comments_error = cres.error;
          if (cres.registry_probe) base.comments_registry_probe = cres.registry_probe;
          if (cres.notes) base.comments_notes = cres.notes;
          return base;
        }).catch(function (e) { base.comments_error = String(e && e.message || e); return base; });
      }
      return base;
    });
  };

  // Public entry: for a paginated GraphQL capability, make sure its results query was
  // captured (actively trigger it if the tab was busy) BEFORE the passive extraction.
  window.__soloGqlPaginate = function (capabilityId, inputs) {
    inputs = inputs || {};
    var cfg = CAPABILITY_PAGINATION[capabilityId];
    if (DOM_CAPABILITIES[capabilityId] || !cfg || !cfg.scope || !window.__soloGql) {
      return __soloGqlPaginateImpl(capabilityId, inputs);
    }
    var tries = inputs.ensure_tries != null ? inputs.ensure_tries : 8;
    return ensureCapture(cfg.scope, tries, 1000).then(function () { return __soloGqlPaginateImpl(capabilityId, inputs); });
  };
})();
