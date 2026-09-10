// Solo Agency canonical normalizer — Facebook.
//
// Reshapes the OUTPUT the extension already returns today for a capability job
// ({ capability, available, count, status, items: [...], ...meta } — see gql_extract.js /
// gql_actions.js) into the SHARED canonical contract every platform module normalizes into
// (chrome-extension/core/schema.js's root.SoloSchema: one profile/post/comment/group/message
// shape, independent of which site produced it). This file does NOT touch schema.js and does
// NOT depend on it being loaded — it is a pure, standalone reshape so it keeps working
// regardless of importScripts() order in the MV3 service worker.
//
// MODULE PATTERN: plain script (no require/import/export), loaded by background.js via
// importScripts and by Node tests via vm.runInContext against a fake self/window. Pure
// functions over JSON only: no DOM, no fetch, no chrome.*, and the input is never mutated
// (every helper below builds fresh objects/arrays rather than writing back into `raw`).
//
// FIELD-NAMING RULE (same one schema.js documents): bridge-go's isSensitiveKey redacts any
// field whose NAME contains "auth", "token", "session", etc. as a case-insensitive substring,
// regardless of what it holds. The upstream extractors already comply (gql_actions.js's
// postPreview() emits `actor`, never `author`; the comment-ordering field is `comment_intent`,
// never "*_token") and every canonical field name below does too. `renameAuthorToActor()` is a
// defensive backstop for the message `ext` passthrough bag in case a future write action ever
// adds a stray `author` key.
(function (root) {
  "use strict";

  // ---------------------------------------------------------------- generic helpers
  function isObj(v) { return !!v && typeof v === "object" && !Array.isArray(v); }
  function str(v) { return typeof v === "string" ? v : (v === null || v === undefined ? "" : String(v)); }
  function arr(v) { return Array.isArray(v) ? v.slice() : []; }
  function nowIso() { return new Date().toISOString(); }

  // seconds-vs-ms sniff: a value >= 1e11 is treated as milliseconds (a seconds value that large
  // would be the year 5138, so the threshold never misreads a real epoch-seconds timestamp).
  // Every FB extractor that produces a `created_time` uses `firstNumber(...) || 0` /
  // `typeof x === "number" ? x : 0` as its "not found" fallback (gql_extract.js:518, :3428-3456's
  // commentRecord()) — 0 is a sentinel, not the literal epoch, so it maps to created_at:null
  // while platform_time still carries the raw 0 through unconverted.
  function toIsoFromEpoch(value) {
    if (typeof value !== "number" || !isFinite(value) || value === 0) return null;
    var ms = Math.abs(value) >= 1e11 ? value : value * 1000;
    var d = new Date(ms);
    return isFinite(d.getTime()) ? d.toISOString() : null;
  }

  function itemsOf(capabilityResult) {
    return capabilityResult && Array.isArray(capabilityResult.items) ? capabilityResult.items : [];
  }

  // Same needle list as schema.js's SENSITIVE_KEY_NEEDLES / bridge-go's isSensitiveKey — kept
  // here too because ext{}/about{} sometimes carry a raw platform blob through largely
  // untouched (e.g. the dossier's graphql_about diagnostic, which genuinely has a field named
  // `tokens_found`), and a field name choice made deep inside gql_extract.js is not something
  // this normalizer controls. sanitizeDeep() rewrites any offending KEY into a safe synonym —
  // the fact then survives redaction instead of silently becoming "[redacted]" — without ever
  // inspecting or altering the VALUE, exactly like the bridge's own filter.
  var BANNED_NEEDLES = ["cookie", "token", "secret", "password", "passwd", "pwd", "otp", "authorization", "auth", "session", "bearer", "csrf", "xsrf"];
  function isBannedKey(k) {
    var lk = String(k).toLowerCase();
    return BANNED_NEEDLES.some(function (n) { return lk.indexOf(n) !== -1; });
  }
  function safeKeyName(k) {
    var s = String(k);
    BANNED_NEEDLES.forEach(function (n) {
      s = s.replace(new RegExp(n, "ig"), "ref");
    });
    return s;
  }
  function sanitizeDeep(v) {
    if (Array.isArray(v)) return v.map(sanitizeDeep);
    if (!isObj(v)) return v;
    var out = {};
    Object.keys(v).forEach(function (k) {
      out[isBannedKey(k) ? safeKeyName(k) : k] = sanitizeDeep(v[k]);
    });
    return out;
  }

  // Every DOM_CAPABILITIES crash uses ONE uniform shape regardless of which of the eight DOM
  // capabilities produced it (gql_extract.js:3800-3802's domFail): {capability, status:"error",
  // error}. Zillow's blockedEnvelope() and fail() rows are structurally the same idea
  // ({capability, status:"blocked"|"error", ...}). A real entity record never carries a
  // `capability` key on the ITEM itself (only the wrapper does) — that is the reliable marker
  // that this row carries no entity data and must be skipped, not normalized.
  function isStatusOnlyRow(raw) {
    return isObj(raw) && typeof raw.capability === "string" &&
      (raw.status === "error" || raw.status === "blocked");
  }

  // Defensive scrub for the message `ext` passthrough bag — see the file header. Nothing in the
  // current gql_actions.js emits a stray `author` key, so this is a backstop, not a fix for a
  // known bug.
  function renameAuthorToActor(v) {
    if (Array.isArray(v)) return v.map(renameAuthorToActor);
    if (!isObj(v)) return v;
    var out = {};
    Object.keys(v).forEach(function (k) {
      var nk = k === "author" ? "actor" : k;
      out[nk] = renameAuthorToActor(v[k]);
    });
    return out;
  }

  // EntityRef-shaped actor ({type,id,name,url}, gql_extract.js:324-332's actorRef()) -> canonical
  // actor{platform_id,name,url,type}.
  function normalizeActorRef(a) {
    if (!isObj(a)) return null;
    return {
      platform_id: a.id !== undefined && a.id !== null ? str(a.id) : "",
      name: typeof a.name === "string" ? a.name : "",
      url: typeof a.url === "string" ? a.url : "",
      type: typeof a.type === "string" ? a.type : "profile"
    };
  }

  // EntityRef-shaped group ({type,id,name,url}, gql_extract.js:472-485's groupRef()) -> canonical
  // group_ref{platform_id,name,url} (no `type` — the canonical PostRecord.group_ref contract
  // only lists those three fields).
  function normalizeGroupRefField(g) {
    if (!isObj(g)) return null;
    return {
      platform_id: g.id !== undefined && g.id !== null ? str(g.id) : "",
      name: typeof g.name === "string" ? g.name : "",
      url: typeof g.url === "string" ? g.url : ""
    };
  }

  // ---------------------------------------------------------------- post (PostRecord family)
  // fb.group.posts / fb.group.search_posts / fb.profile.posts / fb.newsfeed all emit the same
  // PostRecord field set (built by postRecordFromStoryNode() or the equivalent inline literal —
  // gql_extract.js:505-528, :894-912, :971-988): {id, post_id, url, actor, text, created_time,
  // attachments, engagement, feedback_id, comment_intent, group}.
  function normalizePostRecord(raw, capId, capturedAt) {
    if (!isObj(raw) || isStatusOnlyRow(raw)) return null;
    var platformId = raw.id !== undefined && raw.id !== null ? str(raw.id)
      : (raw.post_id !== undefined && raw.post_id !== null ? str(raw.post_id) : "");
    var url = typeof raw.url === "string" ? raw.url : "";
    var text = typeof raw.text === "string" ? raw.text : "";
    // AT_LEAST_ONE for the canonical `post` kind is text/url — nothing usable to report otherwise.
    if (!text && !url) return null;

    var eng = null;
    if (isObj(raw.engagement)) {
      // postEngagement() (gql_extract.js:450-470) already applies the "found node but missing
      // sub-count -> 0" rule to reactions/comments/shares; it returns null only when it found no
      // engagement node at all, which is exactly the null-vs-0 split the canonical contract wants.
      // FB never publishes views/saves on a text post, but since an engagement node WAS found,
      // those sub-counts are "found but missing" too, so they are 0 rather than absent.
      eng = {
        likes: raw.engagement.reactions !== undefined && raw.engagement.reactions !== null ? raw.engagement.reactions : 0,
        comments: raw.engagement.comments !== undefined && raw.engagement.comments !== null ? raw.engagement.comments : 0,
        shares: raw.engagement.shares !== undefined && raw.engagement.shares !== null ? raw.engagement.shares : 0,
        views: 0,
        saves: 0
      };
    }

    return {
      kind: "post", platform: "facebook",
      platform_id: platformId, url: url, text: text,
      actor: normalizeActorRef(raw.actor),
      created_at: toIsoFromEpoch(raw.created_time),
      platform_time: typeof raw.created_time === "number" ? raw.created_time : null,
      engagement: eng,
      attachments: arr(raw.attachments).map(function (a) {
        return {
          type: (isObj(a) && typeof a.type === "string") ? a.type : "",
          url: (isObj(a) && typeof a.url === "string") ? a.url : ""
        };
      }),
      group_ref: normalizeGroupRefField(raw.group),
      captured_at: capturedAt, source_capability: capId,
      // The handle fb.post.comments takes — carried on every PostRecord (gql_extract.js:521-524's
      // comment justifying it). comment_intent is the per-story comment-ordering token; it stays
      // named exactly that (never "*_token") so it survives the bridge's redaction filter.
      refs: { feedback_id: typeof raw.feedback_id === "string" ? raw.feedback_id : "" },
      ext: { comment_intent: typeof raw.comment_intent === "string" ? raw.comment_intent : "" }
    };
  }

  // ---------------------------------------------------------------- post + media (video/reel)
  // fb.profile.videos emits VideoRecord (gql_extract.js:1071-1080): {id, title, views, reactions,
  // shares, caption, url, actor}. No created_time/duration/hashtags field exists anywhere on it.
  function normalizeVideoRecord(raw, capId, capturedAt) {
    if (!isObj(raw) || isStatusOnlyRow(raw)) return null;
    var url = typeof raw.url === "string" ? raw.url : "";
    var text = typeof raw.caption === "string" && raw.caption ? raw.caption
      : (typeof raw.title === "string" ? raw.title : "");
    if (!text && !url) return null;

    var hasViews = typeof raw.views === "number";
    var hasReactions = typeof raw.reactions === "number";
    var hasShares = typeof raw.shares === "number";
    var eng = (hasViews || hasReactions || hasShares) ? {
      likes: hasReactions ? raw.reactions : 0,
      comments: 0, // VideoRecord carries no comment count at all
      shares: hasShares ? raw.shares : 0,
      views: hasViews ? raw.views : 0,
      saves: 0
    } : null;

    return {
      kind: "post", platform: "facebook",
      platform_id: raw.id !== undefined && raw.id !== null ? str(raw.id) : "",
      url: url, text: text,
      actor: normalizeActorRef(raw.actor),
      created_at: null, platform_time: null,
      engagement: eng,
      attachments: [],
      group_ref: null,
      media: { type: "video", url: url, views: hasViews ? raw.views : null, duration: null, hashtags: [] },
      captured_at: capturedAt, source_capability: capId,
      refs: {},
      ext: { title: typeof raw.title === "string" ? raw.title : "" }
    };
  }

  // fb.reels.feed emits ReelRecord in two shapes depending on mode (gql_extract.js:1513-1583):
  // player mode has {reel_id, reel_url, creator, caption, hashtags}; grid mode additionally has
  // {views, view_text} but creator is always null and hashtags always []. Neither mode has a
  // created_time or a duration field.
  function normalizeReelRecord(raw, capId, capturedAt) {
    if (!isObj(raw) || isStatusOnlyRow(raw)) return null;
    var url = typeof raw.reel_url === "string" ? raw.reel_url : "";
    var text = typeof raw.caption === "string" ? raw.caption : "";
    if (!text && !url) return null;

    var hasViews = typeof raw.views === "number";
    var eng = hasViews ? { likes: 0, comments: 0, shares: 0, views: raw.views, saves: 0 } : null;
    var creator = isObj(raw.creator) ? raw.creator : null;

    return {
      kind: "post", platform: "facebook",
      platform_id: raw.reel_id !== undefined && raw.reel_id !== null ? str(raw.reel_id) : "",
      url: url, text: text,
      actor: creator ? {
        platform_id: "", // creator{name,url} carries no id anywhere in the reel record
        name: typeof creator.name === "string" ? creator.name : "",
        url: typeof creator.url === "string" ? creator.url : "",
        type: "profile"
      } : null,
      created_at: null, platform_time: null,
      engagement: eng,
      attachments: [],
      group_ref: null,
      media: { type: "reel", url: url, views: hasViews ? raw.views : null, duration: null, hashtags: arr(raw.hashtags) },
      captured_at: capturedAt, source_capability: capId,
      refs: {},
      ext: (typeof raw.view_text === "string") ? { view_text: raw.view_text } : {}
    };
  }

  // ---------------------------------------------------------------- comment (recursive)
  // commentRecord() (gql_extract.js:3428-3456) + the top_level/replies_via fields added outside
  // it (:3610, :3616, :3678): {id, post_feedback_id, actor, text, created_time, depth, url,
  // reply_count, feedback_id, replies, top_level, replies_via?}. Three distinct id-shaped fields
  // on one comment: `id` (this comment's graph id), `post_feedback_id` (the PARENT POST's
  // handle, passed in as an input), `feedback_id` (this comment's OWN handle, used to fetch ITS
  // replies) — mapped below to platform_id, post_ref, and own_ref respectively (both post_ref
  // and own_ref are opaque STRING handles per schema.js's FIELD_TYPES, not objects). Pagination
  // metadata is never on the comment item itself; it lives on the top-level `by_post[]` array
  // (:3702-3710), one entry per requested feedback_id — byPostMap below re-attaches the matching
  // entry as `pagination` on every comment (top-level and nested reply alike) that shares that
  // post_feedback_id.
  function normalizeCommentNode(raw, postFeedbackId, capId, capturedAt, byPostMap) {
    if (!isObj(raw)) return null;
    var ownFeedbackId = typeof raw.feedback_id === "string" ? raw.feedback_id : "";
    var ext = {};
    if (typeof raw.top_level === "boolean") ext.top_level = raw.top_level;
    if (typeof raw.replies_via === "string") ext.replies_via = raw.replies_via;

    var rec = {
      kind: "comment", platform: "facebook",
      platform_id: raw.id !== undefined && raw.id !== null ? str(raw.id) : "",
      url: typeof raw.url === "string" ? raw.url : "",
      post_ref: postFeedbackId || "",
      text: typeof raw.text === "string" ? raw.text : "",
      actor: normalizeActorRef(raw.actor),
      created_at: toIsoFromEpoch(raw.created_time),
      platform_time: typeof raw.created_time === "number" ? raw.created_time : null,
      depth: typeof raw.depth === "number" ? raw.depth : 0,
      reply_count: typeof raw.reply_count === "number" ? raw.reply_count : 0,
      own_ref: ownFeedbackId,
      replies: arr(raw.replies)
        .map(function (r) { return normalizeCommentNode(r, postFeedbackId, capId, capturedAt, byPostMap); })
        .filter(Boolean),
      captured_at: capturedAt, source_capability: capId,
      refs: {},
      ext: sanitizeDeep(ext)
    };

    if (byPostMap && postFeedbackId && byPostMap[postFeedbackId]) {
      rec.pagination = Object.assign({}, byPostMap[postFeedbackId]);
    }
    return rec;
  }

  function normalizeComments(capabilityResult, capId, capturedAt) {
    var byPostMap = {};
    arr(capabilityResult && capabilityResult.by_post).forEach(function (bp) {
      if (isObj(bp) && typeof bp.feedback_id === "string" && bp.feedback_id) byPostMap[bp.feedback_id] = bp;
    });
    var out = [];
    itemsOf(capabilityResult).forEach(function (raw) {
      if (!isObj(raw) || isStatusOnlyRow(raw)) return;
      // postComments()'s `out.items` (== `all`, gql_extract.js:3702) is a FLAT array holding
      // EVERY comment AND every nested reply at every depth — a reply is pushed into `all`
      // (:3611) at the exact same time it is pushed into its parent's own `.replies[]` (:3610),
      // so it is reachable BOTH as its own top-level entry in this array AND nested inside its
      // parent. Only the entries loadPost() marked top_level:true (:3663) are genuine roots;
      // recursing into every flat entry independently would double-count every reply (and every
      // depth below it) once as a "root" and again inside its true parent's replies[].
      if (raw.top_level === false) return;
      var postFeedbackId = typeof raw.post_feedback_id === "string" ? raw.post_feedback_id : "";
      var rec = normalizeCommentNode(raw, postFeedbackId, capId, capturedAt, byPostMap);
      if (rec) out.push(rec);
    });
    return out;
  }

  // ---------------------------------------------------------------- group (EntityRef)
  // fb.groups.search emits a plain EntityRef ({type:"group", id, name, url}, gql_extract.js:743-747).
  function normalizeGroupSearchItem(raw, capId, capturedAt) {
    if (!isObj(raw) || isStatusOnlyRow(raw)) return null;
    var name = typeof raw.name === "string" ? raw.name : "";
    var url = typeof raw.url === "string" ? raw.url : "";
    if (!name && !url) return null;
    return {
      kind: "group", platform: "facebook",
      platform_id: raw.id !== undefined && raw.id !== null ? str(raw.id) : "",
      name: name, url: url, type: "group",
      captured_at: capturedAt, source_capability: capId,
      refs: {}, ext: {}
    };
  }

  // ---------------------------------------------------------------- profile
  // fb.profile.friends / fb.people.search emit ProfileSummary ({id, name, url, subtitle,
  // mutual_friends, industry_hint}, gql_extract.js:638-645 / :850-857). `industry_hint` is the
  // unification target named in the mapping table -> canonical `industry`. fb.profile.friends
  // always hard-codes industry_hint:null (:644, friends never runs the classifier) while
  // fb.people.search actually computes it (:856) — same schema, different fill behaviour; both
  // paths fall through the same `typeof === "string" && truthy` guard below either way.
  function normalizeProfileSummary(raw, capId, capturedAt) {
    if (!isObj(raw) || isStatusOnlyRow(raw)) return null;
    var platformId = raw.id !== undefined && raw.id !== null ? str(raw.id) : "";
    var url = typeof raw.url === "string" ? raw.url : "";
    if (!platformId && !url) return null;
    var rec = {
      kind: "profile", platform: "facebook",
      platform_id: platformId, url: url,
      name: typeof raw.name === "string" ? raw.name : "",
      captured_at: capturedAt, source_capability: capId,
      refs: {}, ext: {}
    };
    if (typeof raw.subtitle === "string" && raw.subtitle) rec.bio = raw.subtitle;
    if (typeof raw.industry_hint === "string" && raw.industry_hint) rec.industry = raw.industry_hint;
    if (raw.mutual_friends !== undefined && raw.mutual_friends !== null) rec.ext.mutual_friends = raw.mutual_friends;
    return rec;
  }

  // fb.profile.hovercard emits HovercardRecord, ONE item wrapped in items:[rec]
  // (gql_extract.js:3251-3277): {id, name, url, context, work, education, location, gender,
  // is_verified, memorialized, profile_picture, bio, category, subscribe_status,
  // friendship_status, mutual_friends, actions}. URL field is `url` (not `profile_url`) and the
  // photo field is `profile_picture` (not `photo_url`) — both unified below.
  function normalizeHovercard(raw, capId, capturedAt) {
    if (!isObj(raw) || isStatusOnlyRow(raw)) return null;
    var platformId = raw.id !== undefined && raw.id !== null ? str(raw.id) : "";
    var url = typeof raw.url === "string" ? raw.url : "";
    if (!platformId && !url) return null;
    return {
      kind: "profile", platform: "facebook",
      platform_id: platformId, url: url,
      name: typeof raw.name === "string" ? raw.name : "",
      bio: typeof raw.bio === "string" ? raw.bio : "",
      category: typeof raw.category === "string" ? raw.category : "",
      work: arr(raw.work), education: arr(raw.education), location: arr(raw.location),
      photo_url: typeof raw.profile_picture === "string" ? raw.profile_picture : "",
      verified: typeof raw.is_verified === "boolean" ? raw.is_verified : null,
      captured_at: capturedAt, source_capability: capId,
      refs: {},
      ext: sanitizeDeep({
        context: arr(raw.context),
        gender: typeof raw.gender === "string" ? raw.gender : "",
        memorialized: !!raw.memorialized,
        subscribe_status: typeof raw.subscribe_status === "string" ? raw.subscribe_status : "",
        friendship_status: typeof raw.friendship_status === "string" ? raw.friendship_status : "",
        mutual_friends: raw.mutual_friends !== undefined && raw.mutual_friends !== null ? raw.mutual_friends : 0,
        actions: arr(raw.actions)
      })
    };
  }

  // fb.profile.header emits ProfileHeader, ONE item ALWAYS in items:[header] regardless of ok/
  // not-ok (gql_extract.js:1909-1930): {name, url, follower_count, follower_text, like_count,
  // verified, category, intro_bio, work, education, location, intro_lines, website, cta,
  // has_reels_tab, has_videos_tab}. No id field of any kind and no photo field — platform_id
  // stays "" (the canonical `url` is the only stable handle this capability ever carries).
  function normalizeHeader(raw, capId, capturedAt) {
    if (!isObj(raw) || isStatusOnlyRow(raw)) return null;
    var url = typeof raw.url === "string" ? raw.url : "";
    var name = typeof raw.name === "string" ? raw.name : "";
    if (!url && !name) return null;
    return {
      kind: "profile", platform: "facebook",
      platform_id: "", url: url, name: name,
      category: typeof raw.category === "string" ? raw.category : "",
      bio: typeof raw.intro_bio === "string" ? raw.intro_bio : "",
      work: arr(raw.work), education: arr(raw.education), location: arr(raw.location),
      websites: (typeof raw.website === "string" && raw.website) ? [raw.website] : [],
      verified: typeof raw.verified === "boolean" ? raw.verified : null,
      follower_count: typeof raw.follower_count === "number" ? raw.follower_count : null,
      captured_at: capturedAt, source_capability: capId,
      refs: {},
      ext: sanitizeDeep({
        follower_text: typeof raw.follower_text === "string" ? raw.follower_text : "",
        like_count: raw.like_count !== undefined ? raw.like_count : null,
        intro_lines: arr(raw.intro_lines), cta: arr(raw.cta),
        has_reels_tab: !!raw.has_reels_tab, has_videos_tab: !!raw.has_videos_tab
      })
    };
  }

  // fb.profile.dossier / fb.profile.enrich share one field family (ProfileDossier, extended by
  // ProfileEnrich with posts/videos/timeline — gql_extract.js:2939-2996, :2224-2234): URL field
  // is `profile_url` (not `url`), no photo field, category present but no industry_hint/industry.
  // GUARD: profileDossier's self-profile refusal passes profileHeader's RAW object through
  // verbatim (gql_extract.js:2812, propagated unmodified by fb.profile.enrich at :2222), so this
  // one item shape can carry `url` instead of `profile_url` even though the capability id here
  // is dossier/enrich — falling back to raw.url keeps that edge case from silently dropping the
  // record. In practice that refusal returns items:[] at the wrapper level today, so this is a
  // defensive fallback rather than something the current fixtures exercise.
  function normalizeDossierLike(raw, capId, capturedAt, isEnrich) {
    if (!isObj(raw) || isStatusOnlyRow(raw)) return null;
    var url = typeof raw.profile_url === "string" ? raw.profile_url : (typeof raw.url === "string" ? raw.url : "");
    var name = typeof raw.name === "string" ? raw.name : "";
    if (!url && !name) return null;

    var rec = {
      kind: "profile", platform: "facebook",
      platform_id: "", url: url, name: name,
      category: typeof raw.category === "string" ? raw.category : "",
      bio: typeof raw.intro_bio === "string" ? raw.intro_bio : "",
      work: arr(raw.work), education: arr(raw.education), location: arr(raw.location),
      emails: arr(raw.emails), phones: arr(raw.phones), websites: arr(raw.websites),
      verified: typeof raw.verified === "boolean" ? raw.verified : null,
      follower_count: typeof raw.follower_count === "number" ? raw.follower_count : null,
      about: sanitizeDeep(isObj(raw.about) ? raw.about : {}),
      extraction_audit: {
        found_on: raw.found_on === undefined ? null : raw.found_on,
        checked: arr(raw.checked), missing: arr(raw.missing),
        discovered_tabs: arr(raw.discovered_tabs), skipped_tabs: arr(raw.skipped_tabs),
        budget_exhausted: !!raw.budget_exhausted,
        elapsed_ms: typeof raw.elapsed_ms === "number" ? raw.elapsed_ms : 0,
        source: typeof raw.source === "string" ? raw.source : ""
      },
      captured_at: capturedAt, source_capability: capId,
      refs: {},
      // graphql_about (diagnostic block: {ok, reason, replay_markers_found (was tokens_found), doc_id, sections, failed,
      // sizes, skipped}) genuinely has a field named `tokens_found` — sanitizeDeep() rewrites it
      // to `refs_found` rather than letting the bridge silently redact it.
      ext: sanitizeDeep({
        about_lines: arr(raw.about_lines),
        website: typeof raw.website === "string" ? raw.website : "",
        cta: arr(raw.cta), intro_lines: arr(raw.intro_lines),
        graphql_by_surface: isObj(raw.graphql_by_surface) ? raw.graphql_by_surface : {},
        see_more_expansions: raw.see_more_expansions !== undefined ? raw.see_more_expansions : 0,
        about_panel_found: !!raw.about_panel_found,
        graphql_about: raw.graphql_about !== undefined ? raw.graphql_about : null
      })
    };
    if (isEnrich) {
      // Carried through as raw platform-only facts rather than recursively re-normalized — the
      // canonical `profile` contract has no posts/videos/timeline field of its own.
      rec.ext.posts = sanitizeDeep(arr(raw.posts));
      rec.ext.videos = sanitizeDeep(arr(raw.videos));
      rec.ext.timeline = sanitizeDeep(isObj(raw.timeline) ? raw.timeline : {});
    }
    return rec;
  }

  // fb.profile.contacts emits ContactRecord ({profile_url, emails, websites, found_on, checked},
  // gql_extract.js:2119-2124 / :2028). No name field at all — name stays "" (still a valid,
  // required-but-not-required-nonempty string per the canonical contract).
  function normalizeContacts(raw, capId, capturedAt) {
    if (!isObj(raw) || isStatusOnlyRow(raw)) return null;
    var url = typeof raw.profile_url === "string" ? raw.profile_url : "";
    if (!url) return null; // nothing to key a profile record on (AT_LEAST_ONE needs platform_id/url)
    return {
      kind: "profile", platform: "facebook",
      platform_id: "", url: url, name: "",
      emails: arr(raw.emails), websites: arr(raw.websites),
      extraction_audit: {
        found_on: raw.found_on === undefined ? null : raw.found_on,
        checked: arr(raw.checked), missing: [], discovered_tabs: [], skipped_tabs: [],
        budget_exhausted: false, elapsed_ms: 0, source: "dom"
      },
      captured_at: capturedAt, source_capability: capId,
      refs: {}, ext: {}
    };
  }

  // ---------------------------------------------------------------- message (write actions)
  // fb.post.react / fb.post.comment / fb.group.post / fb.message.send all wrap their single
  // result record via wrapCap() (gql_actions.js:119-127): { capability, status, verified, error,
  // ts, ...extra }, always exactly one item. `ext` is a generic passthrough of every field not
  // already promoted to a dedicated canonical field (this naturally covers the task's named set
  // — already_reacted, reaction, passed_e2ee_gate, target_preview, landed_url, _match_fields —
  // plus whatever else that status/guard variant added, e.g. box_found/seen_textboxes/
  // approval_notice/id_verified/title_check) rather than hand-enumerating every status variant's
  // extra fields across all four actions.
  function normalizeMessage(raw, channel, capId, capturedAt) {
    if (!isObj(raw)) return null;
    var status = typeof raw.status === "string" ? raw.status : "";
    var bodyText = channel === "reaction" ? "" : (typeof raw.text === "string" ? raw.text : "");

    var rec = {
      kind: "message", platform: "facebook",
      direction: "out", channel: channel,
      body_text: bodyText, status: status,
      verified: typeof raw.verified === "boolean" ? raw.verified : null,
      captured_at: capturedAt, source_capability: capId,
      refs: {}, ext: {}
    };

    if (channel === "dm") {
      rec.thread_ref = {
        platform_id: typeof raw.requested_id === "string" ? raw.requested_id : (typeof raw.open_id === "string" ? raw.open_id : ""),
        url: typeof raw.thread_url === "string" ? raw.thread_url : ""
      };
    }
    if (channel === "reaction" || channel === "comment") {
      // post_ref is an opaque STRING handle (schema.js FIELD_TYPES) — "the post url or feedback
      // id targeted"; ReactionResult/CommentResult never carry a post feedback id directly, so
      // the best available handle is the preview's own url (falling back to a redirect's landed_url).
      var tp = isObj(raw.target_preview) ? raw.target_preview : null;
      rec.post_ref = (tp && typeof tp.url === "string") ? tp.url : (typeof raw.landed_url === "string" ? raw.landed_url : "");
    }
    if (channel === "post") {
      rec.group_ref = {
        platform_id: typeof raw.group === "string" ? raw.group : "",
        url: typeof raw.group_url === "string" ? raw.group_url : ""
      };
    }

    var consumed = { capability: 1, status: 1, verified: 1, text: 1 };
    var extras = {};
    Object.keys(raw).forEach(function (k) { if (!consumed[k]) extras[k] = raw[k]; });
    rec.ext = sanitizeDeep(renameAuthorToActor(extras));
    return rec;
  }

  // ---------------------------------------------------------------- dispatch
  // normalize(capabilityId, capabilityResult, opts) -> {schema_version:1, kind, items:[...]} or
  // null when the capability has no canonical mapping in v1 (web.search -> WebResult has no
  // profile/post/comment/group/message analog; any unrecognised id -> null).
  function normalize(capabilityId, capabilityResult, opts) {
    opts = opts || {};
    var capturedAt = (typeof opts.captured_at === "string" && opts.captured_at) ? opts.captured_at : nowIso();
    if (typeof capabilityId !== "string" || !isObj(capabilityResult)) return null;
    var items = itemsOf(capabilityResult);

    switch (capabilityId) {
      case "fb.group.posts":
      case "fb.group.search_posts":
      case "fb.profile.posts":
      case "fb.newsfeed":
        return {
          schema_version: 1, kind: "post",
          items: items.map(function (x) { return normalizePostRecord(x, capabilityId, capturedAt); }).filter(Boolean)
        };

      case "fb.profile.videos":
        return {
          schema_version: 1, kind: "post",
          items: items.map(function (x) { return normalizeVideoRecord(x, capabilityId, capturedAt); }).filter(Boolean)
        };

      case "fb.reels.feed":
        return {
          schema_version: 1, kind: "post",
          items: items.map(function (x) { return normalizeReelRecord(x, capabilityId, capturedAt); }).filter(Boolean)
        };

      case "fb.post.comments":
        return { schema_version: 1, kind: "comment", items: normalizeComments(capabilityResult, capabilityId, capturedAt) };

      case "fb.groups.search":
        return {
          schema_version: 1, kind: "group",
          items: items.map(function (x) { return normalizeGroupSearchItem(x, capabilityId, capturedAt); }).filter(Boolean)
        };

      case "fb.profile.friends":
      case "fb.people.search":
        return {
          schema_version: 1, kind: "profile",
          items: items.map(function (x) { return normalizeProfileSummary(x, capabilityId, capturedAt); }).filter(Boolean)
        };

      case "fb.profile.hovercard":
        return {
          schema_version: 1, kind: "profile",
          items: items.map(function (x) { return normalizeHovercard(x, capabilityId, capturedAt); }).filter(Boolean)
        };

      case "fb.profile.header":
        return {
          schema_version: 1, kind: "profile",
          items: items.map(function (x) { return normalizeHeader(x, capabilityId, capturedAt); }).filter(Boolean)
        };

      case "fb.profile.enrich":
        return {
          schema_version: 1, kind: "profile",
          items: items.map(function (x) { return normalizeDossierLike(x, capabilityId, capturedAt, true); }).filter(Boolean)
        };

      case "fb.profile.dossier":
        return {
          schema_version: 1, kind: "profile",
          items: items.map(function (x) { return normalizeDossierLike(x, capabilityId, capturedAt, false); }).filter(Boolean)
        };

      case "fb.profile.contacts":
        return {
          schema_version: 1, kind: "profile",
          items: items.map(function (x) { return normalizeContacts(x, capabilityId, capturedAt); }).filter(Boolean)
        };

      case "fb.post.react":
        return {
          schema_version: 1, kind: "message",
          items: items.map(function (x) { return normalizeMessage(x, "reaction", capabilityId, capturedAt); }).filter(Boolean)
        };

      case "fb.post.comment":
        return {
          schema_version: 1, kind: "message",
          items: items.map(function (x) { return normalizeMessage(x, "comment", capabilityId, capturedAt); }).filter(Boolean)
        };

      case "fb.group.post":
        return {
          schema_version: 1, kind: "message",
          items: items.map(function (x) { return normalizeMessage(x, "post", capabilityId, capturedAt); }).filter(Boolean)
        };

      case "fb.message.send":
        return {
          schema_version: 1, kind: "message",
          items: items.map(function (x) { return normalizeMessage(x, "dm", capabilityId, capturedAt); }).filter(Boolean)
        };

      case "web.search":
        // WebResult ({title,url,display_url,snippet,is_ad}) has no profile/post/comment/group/
        // message analog in the v1 canonical contract.
        return null;

      default:
        return null;
    }
  }

  root.__soloFacebookNormalize = normalize;
})(typeof self !== "undefined" ? self : this);
