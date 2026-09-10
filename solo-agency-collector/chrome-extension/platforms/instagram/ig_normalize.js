// Solo Agency canonical normalizer — Instagram.
//
// Reshapes the OUTPUT ig_extract.js already returns today for a capability job
// ({ capability, available, count, status?, items[], ...meta } — see ig_extract.js) into the
// SHARED canonical contract every platform module normalizes into (chrome-extension/core/
// schema.js's root.SoloSchema: one profile/post/comment/group/message shape, independent of
// which site produced it). This file does NOT touch schema.js and does NOT depend on it being
// loaded — it is a pure, standalone reshape so it keeps working regardless of importScripts()
// order in the MV3 service worker.
//
// MODULE PATTERN: plain script (no require/import/export), loaded by background.js via
// importScripts and by Node tests via vm.runInContext against a fake self/window. Pure
// functions over JSON only: no DOM, no fetch, no chrome.*, and the input is never mutated
// (every helper below builds fresh objects/arrays rather than writing back into `raw`).
//
// FIELD-NAMING RULE (same one schema.js / fb_normalize.js / zillow_normalize.js document):
// bridge-go's isSensitiveKey redacts any field whose NAME contains "auth", "token", "session",
// etc. as a case-insensitive substring. ig_extract.js's own item shapes already comply (the
// posting/commenting entity is `actor`, never `author`), but `ext`/`about`-style passthrough
// bags can still carry a platform field with an unlucky name (Instagram's own JSON has no such
// field today, but the bar is "never trust it", not "never observed it yet") — sanitizeDeep()
// below is the same defensive backstop fb_normalize.js and zillow_normalize.js use.
(function (root) {
  "use strict";

  // ---------------------------------------------------------------- generic helpers
  function isObj(v) { return !!v && typeof v === "object" && !Array.isArray(v); }
  function str(v) { return typeof v === "string" ? v : (v === null || v === undefined ? "" : String(v)); }
  function arr(v) { return Array.isArray(v) ? v.slice() : []; }
  function nowIso() { return new Date().toISOString(); }

  // seconds-vs-ms sniff, same rule as fb_normalize.js's toIsoFromEpoch. ig_extract.js's
  // postRecord()/commentRecord() (ig_extract.js:147, :163) both use `num(...) || 0` as their
  // "not found" fallback, i.e. 0 is a sentinel meaning "no timestamp", not the literal Unix
  // epoch — it maps to created_at:null while platform_time still carries the raw 0 through.
  function toIsoFromEpoch(value) {
    if (typeof value !== "number" || !isFinite(value) || value === 0) return null;
    var ms = Math.abs(value) >= 1e11 ? value : value * 1000;
    var d = new Date(ms);
    return isFinite(d.getTime()) ? d.toISOString() : null;
  }

  function itemsOf(capabilityResult) {
    return capabilityResult && Array.isArray(capabilityResult.items) ? capabilityResult.items : [];
  }

  // Same needle list as schema.js's SENSITIVE_KEY_NEEDLES / bridge-go's isSensitiveKey.
  var BANNED_NEEDLES = ["cookie", "token", "secret", "password", "passwd", "pwd", "otp", "authorization", "auth", "session", "bearer", "csrf", "xsrf"];
  function isBannedKey(k) {
    var lk = String(k).toLowerCase();
    return BANNED_NEEDLES.some(function (n) { return lk.indexOf(n) !== -1; });
  }
  function safeKeyName(k) {
    var s = String(k);
    BANNED_NEEDLES.forEach(function (n) { s = s.replace(new RegExp(n, "ig"), "ref"); });
    return s;
  }
  function sanitizeDeep(v) {
    if (Array.isArray(v)) return v.map(sanitizeDeep);
    if (!isObj(v)) return v;
    var out = {};
    Object.keys(v).forEach(function (k) { out[isBannedKey(k) ? safeKeyName(k) : k] = sanitizeDeep(v[k]); });
    return out;
  }

  // fail()'s item row (ig_extract.js:225-229) carries {capability, status:"error", error, url}
  // directly on the item — same structural marker fb_normalize.js/zillow_normalize.js use to
  // recognise a status-only row that carries no entity data. ig_extract.js has no "blocked"
  // status today (unlike Zillow's PerimeterX gate), but the check covers it too for parity.
  function isStatusOnlyRow(raw) {
    return isObj(raw) && typeof raw.capability === "string" &&
      (raw.status === "error" || raw.status === "blocked");
  }

  // userRef()-shaped actor (ig_extract.js:108-120: {id, username, name, url, type, is_verified,
  // is_private}) -> canonical actor{platform_id, name, url, type}. `type` is hard-coded to
  // "profile" (userRef() itself always sets it that way — every actor this module ever produces
  // IS a profile), matching the literal mapping the task spec calls for rather than passing
  // through a.type indirectly.
  function igActorRef(a) {
    if (!isObj(a)) return null;
    return {
      platform_id: a.id !== undefined && a.id !== null ? str(a.id) : "",
      name: typeof a.name === "string" ? a.name : "",
      url: typeof a.url === "string" ? a.url : "",
      type: "profile"
    };
  }

  // ---------------------------------------------------------------- profile (ig.profile.enrich)
  // profileEnrich()'s graphql item (ig_extract.js:284-310) and its DOM fallback (profileFromDom(),
  // ig_extract.js:247-255) share the same field family: {username, id, name, profile_url, bio,
  // category, external_url, bio_links, website, websites, follower_count, following_count,
  // media_count, is_business, is_professional, account_type, is_private, is_verified, address,
  // emails, phones, profile_pic_url, fbid, mutual_followers_count, source}. The DOM variant omits
  // is_professional/profile_pic_url/fbid/mutual_followers_count/websites entirely (object simply
  // has no such key) — every read below treats "missing" and "wrong type" the same way (falls
  // back to null/""/[]).
  function normalizeProfileEnrich(raw, capId, capturedAt) {
    if (!isObj(raw) || isStatusOnlyRow(raw)) return null;
    var url = typeof raw.profile_url === "string" ? raw.profile_url : "";
    var platformId = raw.id !== undefined && raw.id !== null ? str(raw.id) : "";
    if (!url && !platformId) return null; // AT_LEAST_ONE for the canonical `profile` kind

    var addr = isObj(raw.address) ? raw.address : {};
    var street = typeof addr.street === "string" ? addr.street.trim() : "";
    var cityZip = [addr.city, addr.zip]
      .filter(function (v) { return typeof v === "string" && v.trim() !== ""; })
      .map(function (v) { return v.trim(); })
      .join(" ");
    // location[] holds only the non-empty lines — a DOM-fallback record (address always
    // {street:"",city:"",zip:""}, ig_extract.js:253) contributes an empty array, never ["",""].
    var location = [street, cityZip].filter(function (v) { return v !== ""; });

    return {
      kind: "profile", platform: "instagram",
      platform_id: platformId, url: url,
      name: typeof raw.name === "string" ? raw.name : "",
      handle: typeof raw.username === "string" ? raw.username : "",
      bio: typeof raw.bio === "string" ? raw.bio : "",
      category: typeof raw.category === "string" ? raw.category : "",
      // website is already "external_url or first bio link" (ig_extract.js:283) — reused as-is
      // rather than re-derived, so this normalizer stays a pure reshape of what was computed.
      website: typeof raw.website === "string" ? raw.website : "",
      websites: arr(raw.websites),
      emails: arr(raw.emails),
      phones: arr(raw.phones),
      follower_count: typeof raw.follower_count === "number" ? raw.follower_count : null,
      verified: typeof raw.is_verified === "boolean" ? raw.is_verified : null,
      photo_url: typeof raw.profile_pic_url === "string" ? raw.profile_pic_url : "",
      location: location,
      captured_at: capturedAt, source_capability: capId,
      refs: {},
      ext: sanitizeDeep({
        username: typeof raw.username === "string" ? raw.username : "",
        is_business: typeof raw.is_business === "boolean" ? raw.is_business : null,
        is_professional: typeof raw.is_professional === "boolean" ? raw.is_professional : null,
        account_type: raw.account_type !== undefined ? raw.account_type : null,
        is_private: typeof raw.is_private === "boolean" ? raw.is_private : null,
        address: addr,
        bio_links: arr(raw.bio_links),
        media_count: raw.media_count !== undefined ? raw.media_count : null,
        following_count: raw.following_count !== undefined ? raw.following_count : null,
        mutual_followers_count: raw.mutual_followers_count !== undefined ? raw.mutual_followers_count : null,
        fbid: typeof raw.fbid === "string" ? raw.fbid : "",
        source: typeof raw.source === "string" ? raw.source : ""
      })
    };
  }

  // ---------------------------------------------------------------- profile (ig.people.search)
  // peopleSearch()'s fromUsers() (ig_extract.js:416-427) emits userRef() extended with
  // profile_pic_url and subtitle: {id, username, name, url, type, is_verified, is_private,
  // profile_pic_url, subtitle}.
  function normalizePeopleSearchItem(raw, capId, capturedAt) {
    if (!isObj(raw) || isStatusOnlyRow(raw)) return null;
    var platformId = raw.id !== undefined && raw.id !== null ? str(raw.id) : "";
    var url = typeof raw.url === "string" ? raw.url : "";
    if (!platformId && !url) return null;
    return {
      kind: "profile", platform: "instagram",
      platform_id: platformId, url: url,
      name: typeof raw.name === "string" ? raw.name : "",
      handle: typeof raw.username === "string" ? raw.username : "",
      verified: typeof raw.is_verified === "boolean" ? raw.is_verified : null,
      photo_url: typeof raw.profile_pic_url === "string" ? raw.profile_pic_url : "",
      captured_at: capturedAt, source_capability: capId,
      refs: {},
      ext: sanitizeDeep({
        is_private: typeof raw.is_private === "boolean" ? raw.is_private : null,
        subtitle: typeof raw.subtitle === "string" ? raw.subtitle : ""
      })
    };
  }

  // ---------------------------------------------------------------- post (ig.profile.posts / ig.search.posts)
  // postRecord() (ig_extract.js:127-157): {id, code, url, actor, text, created_time, engagement,
  // attachments, media_type, product_type, carousel_media_count, location, media_id}.
  // media_type: 1 photo, 2 video, 8 carousel (ig_extract.js header comment, :11-22); product_type
  // "feed"|"clips"|"carousel_container" — a reel is product_type "clips" (checked the same way
  // postUrl() does, ig_extract.js:84-87: productType.indexOf("clips") === 0).
  function mediaKindOf(mediaType, productType) {
    if (str(productType).indexOf("clips") === 0) return "reel";
    if (mediaType === 2) return "video";
    if (mediaType === 8) return "carousel";
    return "photo";
  }

  function normalizePostRecord(raw, capId, capturedAt) {
    if (!isObj(raw) || isStatusOnlyRow(raw)) return null;
    var platformId = raw.id !== undefined && raw.id !== null ? str(raw.id) : "";
    var url = typeof raw.url === "string" ? raw.url : "";
    var text = typeof raw.text === "string" ? raw.text : "";
    if (!text && !url) return null; // AT_LEAST_ONE for the canonical `post` kind

    var attachments = arr(raw.attachments).map(function (a) {
      return {
        type: (isObj(a) && typeof a.type === "string") ? a.type : "",
        url: (isObj(a) && typeof a.url === "string") ? a.url : ""
      };
    });

    var eng = isObj(raw.engagement) ? {
      likes: typeof raw.engagement.likes === "number" ? raw.engagement.likes : 0,
      comments: typeof raw.engagement.comments === "number" ? raw.engagement.comments : 0,
      shares: 0,
      views: raw.engagement.views !== undefined ? raw.engagement.views : null
    } : null;

    // media{} is built only for an actual video/reel post (the same restriction
    // fb_normalize.js's normalizeVideoRecord()/normalizeReelRecord() apply to Facebook posts) —
    // a photo or carousel post carries its full picture set in attachments[] instead.
    var kind = mediaKindOf(typeof raw.media_type === "number" ? raw.media_type : null, raw.product_type);
    var media = null;
    if (kind === "video" || kind === "reel") {
      media = {
        type: kind,
        url: attachments[0] ? attachments[0].url : "",
        views: (eng && eng.views !== null && eng.views !== undefined) ? eng.views : null
      };
    }

    return {
      kind: "post", platform: "instagram",
      platform_id: platformId, url: url, text: text,
      actor: igActorRef(raw.actor),
      created_at: toIsoFromEpoch(raw.created_time),
      platform_time: typeof raw.created_time === "number" ? raw.created_time : null,
      engagement: eng,
      attachments: attachments,
      media: media,
      captured_at: capturedAt, source_capability: capId,
      // media_id is the handle ig.post.comments takes; code is the post's own shortcode.
      refs: {
        media_id: typeof raw.media_id === "string" ? raw.media_id : "",
        code: typeof raw.code === "string" ? raw.code : ""
      },
      ext: sanitizeDeep({
        product_type: typeof raw.product_type === "string" ? raw.product_type : "",
        media_type: typeof raw.media_type === "number" ? raw.media_type : null,
        carousel_media_count: raw.carousel_media_count !== undefined ? raw.carousel_media_count : null,
        location: isObj(raw.location) ? raw.location : null
      })
    };
  }

  // ---------------------------------------------------------------- comment (ig.post.comments)
  // commentRecord() (ig_extract.js:158-170): {id, text, created_time, actor, reply_count,
  // like_count, depth, replies:[]}. Unlike Facebook's flat/recursive comment tree,
  // ig.post.comments never nests replies (postComments() only ever calls commentRecord(c, 0),
  // ig_extract.js:479) — replies[] stays [] on every canonical comment too. The parent post's
  // handle is not on the comment item itself; it is the envelope's own `media_id`
  // (envelope(CAP_COMMENTS, ..., {media_id: mediaId, ...}), ig_extract.js:502), so it is passed
  // in here rather than read off `raw`.
  function normalizeCommentRecord(raw, postRef, capId, capturedAt) {
    if (!isObj(raw) || isStatusOnlyRow(raw)) return null;
    var platformId = raw.id !== undefined && raw.id !== null ? str(raw.id) : "";
    if (!platformId) return null; // platform_id is a required field for the `comment` kind
    return {
      kind: "comment", platform: "instagram",
      platform_id: platformId,
      post_ref: postRef || "",
      // an empty-string comment (an emoji-only or media-only reply Instagram renders with no
      // text) still counts — text is only ever "absent" here if `raw.text` was never a string.
      text: typeof raw.text === "string" ? raw.text : "",
      actor: igActorRef(raw.actor),
      created_at: toIsoFromEpoch(raw.created_time),
      platform_time: typeof raw.created_time === "number" ? raw.created_time : null,
      depth: typeof raw.depth === "number" ? raw.depth : 0,
      reply_count: typeof raw.reply_count === "number" ? raw.reply_count : 0,
      replies: [],
      captured_at: capturedAt, source_capability: capId,
      refs: {},
      ext: sanitizeDeep({ like_count: raw.like_count !== undefined ? raw.like_count : null })
    };
  }

  function normalizeComments(capabilityResult, capId, capturedAt) {
    var postRef = typeof capabilityResult.media_id === "string" ? capabilityResult.media_id : "";
    return itemsOf(capabilityResult)
      .map(function (raw) { return normalizeCommentRecord(raw, postRef, capId, capturedAt); })
      .filter(Boolean);
  }

  // ---------------------------------------------------------------- dispatch
  // normalize(capabilityId, capabilityResult, opts) -> {schema_version:1, kind, items:[...]} or
  // null when the capability has no canonical mapping in v1 (_discover.ig is a diagnostic dump
  // with no profile/post/comment/group/message analog; any unrecognised id falls through the
  // same way).
  function normalize(capabilityId, capabilityResult, opts) {
    opts = opts || {};
    var capturedAt = (typeof opts.captured_at === "string" && opts.captured_at) ? opts.captured_at : nowIso();
    if (typeof capabilityId !== "string" || !isObj(capabilityResult)) return null;
    var items = itemsOf(capabilityResult);

    switch (capabilityId) {
      case "ig.profile.enrich":
        return {
          schema_version: 1, kind: "profile",
          items: items.map(function (x) { return normalizeProfileEnrich(x, capabilityId, capturedAt); }).filter(Boolean)
        };

      case "ig.people.search":
        return {
          schema_version: 1, kind: "profile",
          items: items.map(function (x) { return normalizePeopleSearchItem(x, capabilityId, capturedAt); }).filter(Boolean)
        };

      case "ig.profile.posts":
      case "ig.search.posts":
        return {
          schema_version: 1, kind: "post",
          items: items.map(function (x) { return normalizePostRecord(x, capabilityId, capturedAt); }).filter(Boolean)
        };

      case "ig.post.comments":
        return { schema_version: 1, kind: "comment", items: normalizeComments(capabilityResult, capabilityId, capturedAt) };

      default:
        // covers _discover.ig and any capability id this normalizer does not yet know.
        return null;
    }
  }

  root.__soloInstagramNormalize = normalize;
})(typeof self !== "undefined" ? self : this);
