// Solo Agency canonical normalizer — X (Twitter).
//
// Reshapes the OUTPUT x_extract.js returns for a capability job ({ capability, available,
// count, status?, items[], ...meta }) into the SHARED canonical contract every platform
// module normalizes into (chrome-extension/core/schema.js's SoloSchema: one profile/post/
// comment/group/message shape). Pure reshape over JSON — no DOM, no fetch, no chrome.*, the
// input is never mutated — loaded by background.js via importScripts and by Node tests via
// vm.runInContext. Same field-naming rule as the other normalizers: bridge-go redacts any key
// whose NAME contains auth/token/session/…, so passthrough bags go through sanitizeDeep().
(function (root) {
  "use strict";

  function isObj(v) { return !!v && typeof v === "object" && !Array.isArray(v); }
  function str(v) { return typeof v === "string" ? v : (v === null || v === undefined ? "" : String(v)); }
  function arr(v) { return Array.isArray(v) ? v.slice() : []; }
  function nowIso() { return new Date().toISOString(); }
  function toIsoFromEpoch(value) {
    if (typeof value !== "number" || !isFinite(value) || value === 0) return null;
    var ms = Math.abs(value) >= 1e11 ? value : value * 1000;
    var d = new Date(ms);
    return isFinite(d.getTime()) ? d.toISOString() : null;
  }
  function itemsOf(r) { return r && Array.isArray(r.items) ? r.items : []; }
  var BANNED_NEEDLES = ["cookie", "token", "secret", "password", "passwd", "pwd", "otp", "authorization", "auth", "session", "bearer", "csrf", "xsrf"];
  function isBannedKey(k) { var lk = String(k).toLowerCase(); return BANNED_NEEDLES.some(function (n) { return lk.indexOf(n) !== -1; }); }
  function safeKeyName(k) { var s = String(k); BANNED_NEEDLES.forEach(function (n) { s = s.replace(new RegExp(n, "ig"), "ref"); }); return s; }
  function sanitizeDeep(v) {
    if (Array.isArray(v)) return v.map(sanitizeDeep);
    if (!isObj(v)) return v;
    var out = {};
    Object.keys(v).forEach(function (k) { out[isBannedKey(k) ? safeKeyName(k) : k] = sanitizeDeep(v[k]); });
    return out;
  }
  function isStatusOnlyRow(raw) { return isObj(raw) && typeof raw.capability === "string" && (raw.status === "error" || raw.status === "blocked"); }

  // userRef() (x_extract.js): {id, username, name, url, type, is_verified, is_private}
  function xActorRef(a) {
    if (!isObj(a)) return null;
    return { platform_id: a.id !== undefined && a.id !== null ? str(a.id) : "", name: typeof a.name === "string" ? a.name : "", url: typeof a.url === "string" ? a.url : "", type: "profile" };
  }

  // userRecord() (x_extract.js): {id, username, name, profile_url, bio, location, website,
  // websites, emails, phones, follower_count, following_count, post_count, media_count,
  // listed_count, created_at, joined_time, is_verified, verified_type, is_private,
  // professional_type, category, categories, affiliation, profile_pic_url, pinned_post_ids, source}
  function normalizeProfile(raw, capId, capturedAt) {
    if (!isObj(raw) || isStatusOnlyRow(raw)) return null;
    var url = typeof raw.profile_url === "string" ? raw.profile_url : (typeof raw.url === "string" ? raw.url : "");
    var platformId = raw.id !== undefined && raw.id !== null ? str(raw.id) : "";
    if (!url && !platformId) return null;
    var loc = typeof raw.location === "string" && raw.location.trim() ? [raw.location.trim()] : [];
    return {
      kind: "profile", platform: "x",
      platform_id: platformId, url: url,
      name: typeof raw.name === "string" ? raw.name : "",
      handle: typeof raw.username === "string" ? raw.username : "",
      bio: typeof raw.bio === "string" ? raw.bio : "",
      category: typeof raw.category === "string" ? raw.category : "",
      website: typeof raw.website === "string" ? raw.website : "",
      websites: arr(raw.websites),
      emails: arr(raw.emails),
      phones: arr(raw.phones),
      follower_count: typeof raw.follower_count === "number" ? raw.follower_count : null,
      verified: typeof raw.is_verified === "boolean" ? raw.is_verified : null,
      photo_url: typeof raw.profile_pic_url === "string" ? raw.profile_pic_url : "",
      location: loc,
      captured_at: capturedAt, source_capability: capId,
      refs: {},
      ext: sanitizeDeep({
        username: typeof raw.username === "string" ? raw.username : "",
        following_count: raw.following_count !== undefined ? raw.following_count : null,
        post_count: raw.post_count !== undefined ? raw.post_count : null,
        media_count: raw.media_count !== undefined ? raw.media_count : null,
        listed_count: raw.listed_count !== undefined ? raw.listed_count : null,
        joined_at: toIsoFromEpoch(typeof raw.joined_time === "number" ? raw.joined_time : 0),
        is_private: typeof raw.is_private === "boolean" ? raw.is_private : null,
        verified_type: typeof raw.verified_type === "string" ? raw.verified_type : "",
        professional_type: typeof raw.professional_type === "string" ? raw.professional_type : "",
        categories: arr(raw.categories),
        affiliation: typeof raw.affiliation === "string" ? raw.affiliation : "",
        pinned_post_ids: arr(raw.pinned_post_ids),
        source: typeof raw.source === "string" ? raw.source : ""
      })
    };
  }

  // tweetRecord() (x_extract.js): {id, url, actor, text, created_time, lang, engagement{likes,
  // comments, shares, views, reposts, quotes, bookmarks}, attachments[{type,url,preview_url}],
  // links, hashtags, mentions, conversation_id, in_reply_to_post_id, in_reply_to_username,
  // is_repost, repost_of, is_quote, quote_of, post_id, entry_id, depth?}
  function normalizePost(raw, capId, capturedAt) {
    if (!isObj(raw) || isStatusOnlyRow(raw)) return null;
    var platformId = raw.id !== undefined && raw.id !== null ? str(raw.id) : "";
    var url = typeof raw.url === "string" ? raw.url : "";
    var text = typeof raw.text === "string" ? raw.text : "";
    if (!platformId) return null;
    var attachments = arr(raw.attachments).map(function (a) { return { type: isObj(a) && typeof a.type === "string" ? a.type : "", url: isObj(a) && typeof a.url === "string" ? a.url : "" }; });
    var eng = isObj(raw.engagement) ? {
      likes: typeof raw.engagement.likes === "number" ? raw.engagement.likes : 0,
      comments: typeof raw.engagement.comments === "number" ? raw.engagement.comments : 0,
      shares: typeof raw.engagement.shares === "number" ? raw.engagement.shares : 0,
      views: raw.engagement.views !== undefined ? raw.engagement.views : null
    } : null;
    var video = attachments.filter(function (a) { return a.type === "video" || a.type === "gif"; })[0] || null;
    return {
      kind: "post", platform: "x",
      platform_id: platformId, url: url, text: text,
      actor: xActorRef(raw.actor),
      created_at: toIsoFromEpoch(raw.created_time),
      platform_time: typeof raw.created_time === "number" ? raw.created_time : null,
      engagement: eng,
      attachments: attachments,
      media: video ? { type: "video", url: video.url, views: eng ? eng.views : null } : null,
      captured_at: capturedAt, source_capability: capId,
      refs: {
        post_id: typeof raw.post_id === "string" ? raw.post_id : platformId,
        conversation_id: typeof raw.conversation_id === "string" ? raw.conversation_id : "",
        in_reply_to_post_id: typeof raw.in_reply_to_post_id === "string" ? raw.in_reply_to_post_id : ""
      },
      ext: sanitizeDeep({
        lang: typeof raw.lang === "string" ? raw.lang : "",
        links: arr(raw.links), hashtags: arr(raw.hashtags), mentions: arr(raw.mentions),
        reposts: isObj(raw.engagement) && raw.engagement.reposts !== undefined ? raw.engagement.reposts : null,
        quotes: isObj(raw.engagement) && raw.engagement.quotes !== undefined ? raw.engagement.quotes : null,
        bookmarks: isObj(raw.engagement) && raw.engagement.bookmarks !== undefined ? raw.engagement.bookmarks : null,
        is_repost: raw.is_repost === true, repost_of_id: isObj(raw.repost_of) ? str(raw.repost_of.id) : "",
        is_quote: raw.is_quote === true, quote_of_id: isObj(raw.quote_of) ? str(raw.quote_of.id) : "",
        in_reply_to_username: typeof raw.in_reply_to_username === "string" ? raw.in_reply_to_username : ""
      })
    };
  }

  // x.post.replies: every item is a tweet replying under the envelope's post_id.
  function normalizeReply(raw, postRef, capId, capturedAt) {
    if (!isObj(raw) || isStatusOnlyRow(raw)) return null;
    var platformId = raw.id !== undefined && raw.id !== null ? str(raw.id) : "";
    if (!platformId) return null;
    return {
      kind: "comment", platform: "x",
      platform_id: platformId,
      post_ref: postRef || (typeof raw.conversation_id === "string" ? raw.conversation_id : ""),
      text: typeof raw.text === "string" ? raw.text : "",
      actor: xActorRef(raw.actor),
      created_at: toIsoFromEpoch(raw.created_time),
      platform_time: typeof raw.created_time === "number" ? raw.created_time : null,
      depth: typeof raw.depth === "number" ? raw.depth : 0,
      reply_count: isObj(raw.engagement) && typeof raw.engagement.comments === "number" ? raw.engagement.comments : 0,
      replies: [],
      captured_at: capturedAt, source_capability: capId,
      refs: { url: typeof raw.url === "string" ? raw.url : "", in_reply_to_post_id: typeof raw.in_reply_to_post_id === "string" ? raw.in_reply_to_post_id : "" },
      ext: sanitizeDeep({ like_count: isObj(raw.engagement) ? raw.engagement.likes : null, reposts: isObj(raw.engagement) ? raw.engagement.reposts : null, lang: typeof raw.lang === "string" ? raw.lang : "" })
    };
  }

  function normalize(capabilityId, capabilityResult, opts) {
    opts = opts || {};
    var capturedAt = (typeof opts.captured_at === "string" && opts.captured_at) ? opts.captured_at : nowIso();
    if (typeof capabilityId !== "string" || !isObj(capabilityResult)) return null;
    var items = itemsOf(capabilityResult);
    switch (capabilityId) {
      case "x.profile.enrich":
      case "x.people.search":
        return { schema_version: 1, kind: "profile", items: items.map(function (x) { return normalizeProfile(x, capabilityId, capturedAt); }).filter(Boolean) };
      case "x.profile.posts":
      case "x.search.posts":
      case "x.timeline.home":
        return { schema_version: 1, kind: "post", items: items.map(function (x) { return normalizePost(x, capabilityId, capturedAt); }).filter(Boolean) };
      case "x.post.replies": {
        var postRef = typeof capabilityResult.post_id === "string" ? capabilityResult.post_id : "";
        return { schema_version: 1, kind: "comment", items: items.map(function (x) { return normalizeReply(x, postRef, capabilityId, capturedAt); }).filter(Boolean) };
      }
      default:
        return null; // _discover.x and unknown ids
    }
  }

  root.__soloXNormalize = normalize;
})(typeof self !== "undefined" ? self : this);
