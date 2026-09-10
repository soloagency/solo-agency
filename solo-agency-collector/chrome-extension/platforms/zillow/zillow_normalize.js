// Solo Agency canonical normalizer — Zillow.
//
// Reshapes the OUTPUT zillow_extract.js already returns today for a capability job
// ({ capability, available, count, status, items: [...], ...meta }) into the SHARED canonical
// contract every platform module normalizes into (chrome-extension/core/schema.js's
// root.SoloSchema). This file does NOT touch schema.js and does NOT depend on it being loaded —
// it is a pure, standalone reshape so it keeps working regardless of importScripts() order in
// the MV3 service worker.
//
// MODULE PATTERN: plain script (no require/import/export), loaded by background.js via
// importScripts and by Node tests via vm.runInContext against a fake self/window. Pure
// functions over JSON only: no DOM, no fetch, no chrome.*, and the input is never mutated.
(function (root) {
  "use strict";

  function isObj(v) { return !!v && typeof v === "object" && !Array.isArray(v); }
  function str(v) { return typeof v === "string" ? v : (v === null || v === undefined ? "" : String(v)); }
  function arr(v) { return Array.isArray(v) ? v.slice() : []; }
  function nowIso() { return new Date().toISOString(); }

  function itemsOf(capabilityResult) {
    return capabilityResult && Array.isArray(capabilityResult.items) ? capabilityResult.items : [];
  }

  // blockedEnvelope()'s and fail()'s item rows (zillow_extract.js:117-127, :607-610) both carry
  // {capability, status:"blocked"|"error", ...} directly on the item — a real ZillowAgentCard/
  // ZillowProfileEnrich item never has a `capability` key on itself (only the wrapper does).
  // Skip these; they carry no entity data to normalize.
  function isStatusOnlyRow(raw) {
    return isObj(raw) && typeof raw.capability === "string" &&
      (raw.status === "error" || raw.status === "blocked");
  }

  // zillow.agents.list emits ZillowAgentCard (cardToItem(), zillow_extract.js:226-262 /
  // domCards()'s thinner DOM fallback, :265-289 — same field names either way): {profile_url,
  // screen_name, encoded_zuid, name, brokerage, is_team, is_top_agent, rating, reviews_count,
  // price_range, sales_last_12_months, sales_in_region, sales_in_region_label, photo_url, tags,
  // stats}. `encoded_zuid` is Zillow's durable per-agent id (FB's numeric-id analogue); fall
  // back to screen_name only when it is genuinely absent.
  function normalizeAgentCard(raw, capId, capturedAt) {
    if (!isObj(raw) || isStatusOnlyRow(raw)) return null;
    var url = typeof raw.profile_url === "string" ? raw.profile_url : "";
    var platformId = (typeof raw.encoded_zuid === "string" && raw.encoded_zuid) ? raw.encoded_zuid
      : (typeof raw.screen_name === "string" ? raw.screen_name : "");
    if (!platformId && !url) return null;
    return {
      kind: "profile", platform: "zillow",
      platform_id: platformId, url: url,
      name: typeof raw.name === "string" ? raw.name : "",
      handle: typeof raw.screen_name === "string" ? raw.screen_name : "",
      photo_url: typeof raw.photo_url === "string" ? raw.photo_url : "",
      // Every Zillow agent-directory result is a real-estate professional by construction of the
      // source page — same literal zillow_extract.js uses for zillow.profile.enrich (:35's
      // INDUSTRY_DEFAULT, "Real Estate"), applied here for parity across both capabilities.
      industry: "Real Estate",
      captured_at: capturedAt, source_capability: capId,
      refs: {},
      ext: {
        brokerage: typeof raw.brokerage === "string" ? raw.brokerage : "",
        is_team: !!raw.is_team, is_top_agent: !!raw.is_top_agent,
        rating: raw.rating !== undefined ? raw.rating : null,
        reviews_count: raw.reviews_count !== undefined ? raw.reviews_count : null,
        price_range: raw.price_range !== undefined ? raw.price_range : null,
        sales_last_12_months: raw.sales_last_12_months !== undefined ? raw.sales_last_12_months : null,
        sales_in_region: raw.sales_in_region !== undefined ? raw.sales_in_region : null,
        sales_in_region_label: typeof raw.sales_in_region_label === "string" ? raw.sales_in_region_label : "",
        tags: arr(raw.tags), stats: arr(raw.stats)
      }
    };
  }

  // zillow.profile.enrich emits ONE ZillowProfileEnrich item, deliberately shaped like FB's
  // ProfileDossier/ProfileEnrich (baseRecord(), zillow_extract.js:396-412) plus a typed `zillow{}`
  // block (:586-596) and a constant `industry` field. The top-level `emails`/`phones` arrays
  // ALREADY fold in zillow.email / zillow.phones (profileEnrich() builds them via
  // `uniqEmails([du.email].concat(...))` at :456 and `uniqPhones(phoneList)` at :455, where
  // phoneList itself is seeded from phoneMap = du.phoneNumbers, the same object that becomes
  // zillow.phones) — so canonical emails[]/phones[] are taken from the top-level arrays AS-IS,
  // never re-merged with zillow.email/zillow.phones, to avoid duplicate entries.
  function normalizeProfileEnrich(raw, capId, capturedAt) {
    if (!isObj(raw) || isStatusOnlyRow(raw)) return null;
    var url = typeof raw.profile_url === "string" ? raw.profile_url : "";
    var name = typeof raw.name === "string" ? raw.name : "";
    var z = isObj(raw.zillow) ? raw.zillow : {};
    var platformId = (typeof z.encoded_zuid === "string" && z.encoded_zuid) ? z.encoded_zuid
      : (typeof z.screen_name === "string" ? z.screen_name : "");
    if (!url && !platformId) return null;

    return {
      kind: "profile", platform: "zillow",
      platform_id: platformId, url: url, name: name,
      category: typeof raw.category === "string" ? raw.category : "",
      bio: typeof raw.intro_bio === "string" ? raw.intro_bio : "",
      industry: typeof raw.industry === "string" && raw.industry ? raw.industry : "Real Estate",
      work: arr(raw.work), education: arr(raw.education), location: arr(raw.location),
      emails: arr(raw.emails), phones: arr(raw.phones), websites: arr(raw.websites),
      // photo_url lives ONLY nested at zillow.photo_url in the source (baseRecord/zillow{} never
      // puts it at the item's top level) — promoted here to the canonical top-level field every
      // profile record carries, in addition to staying inside ext.zillow for full fidelity.
      photo_url: typeof z.photo_url === "string" ? z.photo_url : "",
      verified: typeof raw.verified === "boolean" ? raw.verified : null,
      follower_count: typeof raw.follower_count === "number" ? raw.follower_count : null,
      about: isObj(raw.about) ? raw.about : {},
      extraction_audit: {
        found_on: raw.found_on === undefined ? null : raw.found_on,
        checked: arr(raw.checked), missing: arr(raw.missing),
        discovered_tabs: arr(raw.discovered_tabs),
        // zillow's baseRecord() never had a skipped_tabs key at all (unlike FB's ProfileDossier,
        // which does) — added here as [] for extraction_audit shape parity across platforms.
        skipped_tabs: [],
        budget_exhausted: !!raw.budget_exhausted,
        elapsed_ms: typeof raw.elapsed_ms === "number" ? raw.elapsed_ms : 0,
        source: typeof raw.source === "string" ? raw.source : ""
      },
      captured_at: capturedAt, source_capability: capId,
      refs: {},
      ext: {
        website: typeof raw.website === "string" ? raw.website : "",
        about_lines: arr(raw.about_lines),
        // posts[] here is Zillow's review/sale-synthesized PostRecord-*like* shape (kind:"review"|
        // "sale", plus a `date` alongside created_time) — NOT the FB PostRecord shape — carried
        // through as a raw platform-only fact rather than recursively re-normalized, same as
        // FB's fb.profile.enrich treatment of its own posts[]/videos[]/timeline{}.
        posts: arr(raw.posts), videos: arr(raw.videos), timeline: isObj(raw.timeline) ? raw.timeline : {},
        zillow: z
      }
    };
  }

  // normalize(capabilityId, capabilityResult, opts) -> {schema_version:1, kind, items:[...]} or
  // null when the capability has no canonical mapping in v1 (any id other than the two Zillow
  // capabilities below).
  function normalize(capabilityId, capabilityResult, opts) {
    opts = opts || {};
    var capturedAt = (typeof opts.captured_at === "string" && opts.captured_at) ? opts.captured_at : nowIso();
    if (typeof capabilityId !== "string" || !isObj(capabilityResult)) return null;
    var items = itemsOf(capabilityResult);

    switch (capabilityId) {
      case "zillow.agents.list":
        return {
          schema_version: 1, kind: "profile",
          items: items.map(function (x) { return normalizeAgentCard(x, capabilityId, capturedAt); }).filter(Boolean)
        };

      case "zillow.profile.enrich":
        return {
          schema_version: 1, kind: "profile",
          items: items.map(function (x) { return normalizeProfileEnrich(x, capabilityId, capturedAt); }).filter(Boolean)
        };

      default:
        return null;
    }
  }

  root.__soloZillowNormalize = normalize;
})(typeof self !== "undefined" ? self : this);
