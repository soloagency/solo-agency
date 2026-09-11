// Solo Agency platform module registry (platform-module refactor, step 2).
//
// A single, data-only description of what background.js currently hard-codes as six
// separate capability-id tables (WRITE_ACTIONS, MATCH_RESOLVABLE, INFO_ONLY_CAPABILITIES,
// PIN_TARGET, POLICY_FLAG, HIDEABLE_CAPABILITIES — background.js ~lines 869-1160 and
// ~3304-3318) plus the injection file lists and page-context entry-point names. Nothing
// here is wired into background.js yet — that is a later step. Until then,
// tests/test_platform_registry.js regex-extracts the six literal tables straight out of
// background.js's source and asserts the derived sets/objects below are byte-for-byte the
// same data, so this file cannot silently drift from the switch it will eventually replace.
//
// Loaded by background.js via importScripts (classic MV3 service worker, no ES modules) and
// by Node tests via vm.runInContext against a fake self/window — see
// tests/test_platform_registry.js.
(function (root) {
  "use strict";

  // ------------------------------------------------------------------
  // The six literal tables, reproduced verbatim from background.js. This is the single
  // source of truth every per-capability entry below is derived from, and it is also
  // exactly what writeActions()/matchResolvable()/.../hideable() return — so the two views
  // (per-capability metadata, and the flat legacy tables) can never disagree with each other.
  // ------------------------------------------------------------------

  // background.js ~869
  const WRITE_ACTIONS_TABLE = ["fb.post.react", "fb.post.comment", "fb.message.send", "fb.group.post", "fb.profile.post"];
  // background.js ~875
  const MATCH_RESOLVABLE_TABLE = ["fb.post.react", "fb.post.comment"];
  // background.js ~890
  const INFO_ONLY_TABLE = ["fb.profile.contacts", "fb.profile.header"];
  // background.js ~959
  const PIN_TARGET_TABLE = ["fb.post.comment", "fb.post.react", "fb.group.post", "fb.profile.post"];
  // background.js ~828 (collectSource) — write capabilities whose target IS the logged-in
  // operator: facebook.com/me and profile.php without an id are the RIGHT url for them, not
  // the ambiguity that guard exists to catch.
  const SELF_TARGET_TABLE = ["fb.profile.post"];
  // background.js ~970
  const POLICY_FLAG_TABLE = {
    "fb.post.comment": "do_not_comment",
    "fb.post.react": "do_not_react",
    "fb.message.send": "do_not_message",
    "fb.group.post": "do_not_post",
    "fb.profile.post": "do_not_post"
  };
  // background.js ~3304 (HIDEABLE_CAPABILITIES) — capabilities whose data survives a tab
  // that is never shown/activated. capabilityNeedsActiveTab (~3314) is exactly the negation
  // of this set.
  const HIDEABLE_TABLE = [
    "fb.profile.dossier",    // About sections; measured intact hidden, and 2% noise instead of ~30%
    "fb.profile.header",     // header + intro card, rendered at load
    "fb.profile.contacts",   // About sub-tabs, same walk as dossier
    "fb.profile.hovercard",  // calls the hovercard query by entity_id; nothing on screen matters
    "zillow.agents.list",    // __NEXT_DATA__ is in the served HTML
    "zillow.profile.enrich", // same
    "web.search"             // static results HTML
  ];

  function inTable(table, id) { return table.indexOf(id) !== -1; }

  // Per-capability metadata for a Facebook (fb.*, plus web.search — see note below)
  // capability, derived from the six tables above so the two views cannot drift apart.
  function fbCapMeta(id, entity) {
    return {
      entity: entity,
      write: inTable(WRITE_ACTIONS_TABLE, id),
      match_resolvable: inTable(MATCH_RESOLVABLE_TABLE, id),
      info_only: inTable(INFO_ONLY_TABLE, id),
      pin_target: inTable(PIN_TARGET_TABLE, id),
      self_target: inTable(SELF_TARGET_TABLE, id),
      policy_flag: POLICY_FLAG_TABLE[id] || null,
      hideable: inTable(HIDEABLE_TABLE, id),
      needs_active_tab: !inTable(HIDEABLE_TABLE, id)
    };
  }

  // Per-capability metadata for a zillow.* capability. Both are in HIDEABLE_TABLE already,
  // so hideable/needs_active_tab fall out of the shared helper. info_only is true for the
  // whole module: background.js used to express that as a separate isZillowCapability OR
  // next to the Facebook INFO_ONLY table; here it is simply data on the capability.
  function zillowCapMeta(id) {
    return {
      entity: "profile",
      write: inTable(WRITE_ACTIONS_TABLE, id),
      match_resolvable: inTable(MATCH_RESOLVABLE_TABLE, id),
      info_only: true,
      pin_target: inTable(PIN_TARGET_TABLE, id),
      policy_flag: POLICY_FLAG_TABLE[id] || null,
      hideable: inTable(HIDEABLE_TABLE, id),
      needs_active_tab: !inTable(HIDEABLE_TABLE, id)
    };
  }

  const DEFAULT_CAP_META = {
    entity: "generic",
    write: false,
    match_resolvable: false,
    info_only: false,
    pin_target: false,
    policy_flag: null,
    hideable: false,
    needs_active_tab: false
  };

  // ------------------------------------------------------------------
  // PLATFORM_MODULES
  // ------------------------------------------------------------------
  const PLATFORM_MODULES = [
    {
      name: "facebook",
      capPrefix: "fb.",
      // background.js ~778 (FB_PERMALINK_HOSTS) — the allowlist a match_text write's
      // resolved permalink is checked against before the tab is navigated there.
      hosts: ["facebook.com", "www.facebook.com", "m.facebook.com", "web.facebook.com"],
      // Paths are relative to the extension root, exactly as chrome.scripting.executeScript
      // wants them (background.js ~1000-1160).
      files: {
        read: ["platforms/facebook/gql_extract.js"],
        write: ["platforms/facebook/gql_actions.js"]
      },
      // Page-context (MAIN world) entry points background.js calls into (~1000-1160).
      entries: {
        run: "__soloGqlPaginate",
        runFallback: "__soloGqlExtractCapability",
        generic: "__soloGqlExtract",
        act: "__soloActRun",
        resolve: "__soloActResolve",
        // Not implemented yet anywhere in the repo — reserved for the platform-module
        // normalize step (canonical-record production) that comes after this one.
        normalize: "__soloFacebookNormalize"
      },
      // Every fb.* id in bridge-go/collector_capabilities.json, PLUS "web.search": that
      // capability is DOM-driven and lives in gql_extract.js's own DOM_CAPABILITIES table
      // (not Facebook GraphQL at all), but it ships inside this module's read file and is
      // dispatched through the same __soloGqlExtractCapability/__soloGqlPaginate entry
      // points, so it is catalogued here rather than invented a module of its own.
      capabilities: {
        "fb.group.posts": fbCapMeta("fb.group.posts", "post"),
        "fb.group.search_posts": fbCapMeta("fb.group.search_posts", "post"),
        // Global search (/search/posts/, /search/top/): same library, same entry points.
        "fb.search.posts": fbCapMeta("fb.search.posts", "post"),
        "fb.post.comments": fbCapMeta("fb.post.comments", "comment"),
        "fb.profile.friends": fbCapMeta("fb.profile.friends", "profile"),
        // status "not_built" in the catalog (superseded by fb.profile.dossier) — kept here
        // so an id lookup for it still resolves to a module instead of falling through.
        "fb.profile.about": fbCapMeta("fb.profile.about", "profile"),
        "fb.groups.search": fbCapMeta("fb.groups.search", "group"),
        "fb.people.search": fbCapMeta("fb.people.search", "profile"),
        "fb.profile.posts": fbCapMeta("fb.profile.posts", "post"),
        "fb.newsfeed": fbCapMeta("fb.newsfeed", "post"),
        // ReelRecord: creator + caption + hashtags — modelled as a post-shaped entity.
        "fb.reels.feed": fbCapMeta("fb.reels.feed", "post"),
        "fb.profile.header": fbCapMeta("fb.profile.header", "profile"),
        "fb.profile.hovercard": fbCapMeta("fb.profile.hovercard", "profile"),
        // VideoRecord: a page's Videos-tab items — modelled as a post-shaped entity.
        "fb.profile.videos": fbCapMeta("fb.profile.videos", "post"),
        // Write actions normalize to a canonical MESSAGE (channel reaction/comment/post/dm): the
        // entity is what the capability EMITS, not what it acts on.
        "fb.post.react": fbCapMeta("fb.post.react", "message"),
        "fb.post.comment": fbCapMeta("fb.post.comment", "message"),
        "fb.profile.enrich": fbCapMeta("fb.profile.enrich", "profile"),
        "fb.profile.dossier": fbCapMeta("fb.profile.dossier", "profile"),
        "fb.profile.contacts": fbCapMeta("fb.profile.contacts", "profile"),
        "fb.message.send": fbCapMeta("fb.message.send", "message"),
        "fb.group.post": fbCapMeta("fb.group.post", "message"),
        "fb.profile.post": fbCapMeta("fb.profile.post", "message"),
        "web.search": fbCapMeta("web.search", "search")
      }
    },
    {
      name: "instagram",
      capPrefix: "ig.",
      hosts: ["instagram.com", "www.instagram.com"],
      // The interceptor (platforms/instagram/ig_intercept.js) is a static content script on
      // instagram.com (manifest.json); the extractor is injected per job.
      files: {
        read: ["platforms/instagram/ig_extract.js"]
      },
      entries: {
        run: "__soloIgRun",
        normalize: "__soloInstagramNormalize",
        // background.js ~1246: the data point's graphql_manifest for this module's pages
        manifest: "__soloIgManifest"
      },
      capabilities: {
        // Profile data arrives by XHR at load and is complete without scrolling, so the tab may
        // stay hidden; the posts grid, the search grid and comments render on demand.
        "ig.profile.enrich": { entity: "profile", write: false, match_resolvable: false, info_only: true, pin_target: false, policy_flag: null, hideable: true, needs_active_tab: false },
        "ig.profile.posts": { entity: "post", write: false, match_resolvable: false, info_only: false, pin_target: false, policy_flag: null, hideable: false, needs_active_tab: true },
        "ig.search.posts": { entity: "post", write: false, match_resolvable: false, info_only: false, pin_target: false, policy_flag: null, hideable: false, needs_active_tab: true },
        "ig.people.search": { entity: "profile", write: false, match_resolvable: false, info_only: true, pin_target: false, policy_flag: null, hideable: true, needs_active_tab: false },
        "ig.post.comments": { entity: "comment", write: false, match_resolvable: false, info_only: true, pin_target: false, policy_flag: null, hideable: false, needs_active_tab: true },
        // maintenance aid, same idea as Facebook's _discover.deep
        "_discover.ig": { entity: "generic", write: false, match_resolvable: false, info_only: true, pin_target: false, policy_flag: null, hideable: true, needs_active_tab: false }
      }
    },
    {
      name: "x",
      capPrefix: "x.",
      hosts: ["x.com", "www.x.com", "mobile.x.com", "twitter.com", "www.twitter.com", "mobile.twitter.com"],
      // The interceptor (platforms/x/x_intercept.js) is a static content script on x.com /
      // twitter.com (manifest.json); the extractor is injected per job.
      files: {
        read: ["platforms/x/x_extract.js"]
      },
      entries: {
        run: "__soloXRun",
        normalize: "__soloXNormalize",
        // background.js ~1246: graphql_manifest from window.__soloX, and the logged-in state
        // from X's own chrome instead of the page-text heuristic
        manifest: "__soloXManifest",
        login: "__soloXLoggedIn"
      },
      capabilities: {
        // The profile query fires at load; timelines and search results render as the tab
        // scrolls, so those need the foreground.
        "x.profile.enrich": { entity: "profile", write: false, match_resolvable: false, info_only: true, pin_target: false, policy_flag: null, hideable: true, needs_active_tab: false },
        "x.profile.posts": { entity: "post", write: false, match_resolvable: false, info_only: false, pin_target: false, policy_flag: null, hideable: false, needs_active_tab: true },
        "x.search.posts": { entity: "post", write: false, match_resolvable: false, info_only: false, pin_target: false, policy_flag: null, hideable: false, needs_active_tab: true },
        "x.people.search": { entity: "profile", write: false, match_resolvable: false, info_only: false, pin_target: false, policy_flag: null, hideable: false, needs_active_tab: true },
        "x.post.replies": { entity: "comment", write: false, match_resolvable: false, info_only: false, pin_target: false, policy_flag: null, hideable: false, needs_active_tab: true },
        "x.timeline.home": { entity: "post", write: false, match_resolvable: false, info_only: false, pin_target: false, policy_flag: null, hideable: false, needs_active_tab: true },
        // maintenance aid, same idea as _discover.ig
        "_discover.x": { entity: "generic", write: false, match_resolvable: false, info_only: true, pin_target: false, policy_flag: null, hideable: true, needs_active_tab: false }
      }
    },
    {
      name: "zillow",
      capPrefix: "zillow.",
      hosts: ["zillow.com", "www.zillow.com"],
      files: {
        read: ["platforms/zillow/zillow_extract.js"]
        // no write file: Zillow has no write capabilities today.
      },
      entries: {
        run: "__soloZillowRun",
        // Not implemented yet — reserved for the platform-module normalize step.
        normalize: "__soloZillowNormalize"
      },
      // background.js ~808-811, ~848-853: Zillow capabilities never scroll (the page's
      // Next.js state is complete at load) and answer with a PerimeterX "Press & Hold" page
      // instead of content, which needs the operator to solve it before the read proceeds.
      no_scroll: true,
      human_gate: "zillow",
      // Merged into every capability's metadata before its own per-capability entry, so a
      // capability added here later only needs to override what's actually different.
      defaults: { info_only: true },
      capabilities: {
        "zillow.agents.list": zillowCapMeta("zillow.agents.list"),
        "zillow.profile.enrich": zillowCapMeta("zillow.profile.enrich")
      }
    }
  ];

  // ------------------------------------------------------------------
  // Lookups
  // ------------------------------------------------------------------

  // moduleForCapability: an EXACT match against some module's `capabilities` map wins over
  // a capPrefix match, so a module can special-case one id (as facebook does for
  // "web.search", which does not start with "fb.") without it losing to prefix matching.
  function moduleForCapability(capId) {
    const id = String(capId || "");
    if (!id) return null;
    for (let i = 0; i < PLATFORM_MODULES.length; i++) {
      const mod = PLATFORM_MODULES[i];
      if (mod.capabilities && Object.prototype.hasOwnProperty.call(mod.capabilities, id)) return mod;
    }
    for (let i = 0; i < PLATFORM_MODULES.length; i++) {
      const mod = PLATFORM_MODULES[i];
      if (mod.capPrefix && id.indexOf(mod.capPrefix) === 0) return mod;
    }
    return null;
  }

  // dispatchModuleFor: the module whose page libraries serve this capability id. An id no
  // module claims — the internal _discover.* / _diag.* ids, or a typo — is served by the
  // FIRST module (facebook): its read library also carries the generic GraphQL layer every
  // Facebook page needs, exactly as background.js has always injected it. That is a known
  // wart (the generic layer belongs in core), recorded here rather than hidden.
  function dispatchModuleFor(capId) {
    return moduleForCapability(capId) || PLATFORM_MODULES[0];
  }
  function dispatchFilesFor(capId, opts) {
    const mod = dispatchModuleFor(capId);
    const wantWrite = !!(opts && opts.write);
    const list = mod && mod.files ? (wantWrite ? mod.files.write : mod.files.read) : null;
    return Array.isArray(list) ? list.slice() : [];
  }
  function dispatchEntryFor(capId, name) {
    const mod = dispatchModuleFor(capId);
    return mod && mod.entries ? (mod.entries[name] || null) : null;
  }

  function moduleForHost(hostname) {
    const h = String(hostname || "").toLowerCase();
    if (!h) return null;
    for (let i = 0; i < PLATFORM_MODULES.length; i++) {
      const mod = PLATFORM_MODULES[i];
      if (Array.isArray(mod.hosts) && mod.hosts.indexOf(h) !== -1) return mod;
    }
    return null;
  }

  // capabilityMeta: DEFAULT_CAP_META, overridden by the owning module's own `defaults` (if
  // any), overridden by the capability's own explicit entry (if any). A capability id that
  // resolves to no module at all gets plain DEFAULT_CAP_META back.
  function capabilityMeta(capId) {
    const mod = moduleForCapability(capId);
    const perCap = mod && mod.capabilities ? mod.capabilities[String(capId || "")] : null;
    const merged = {};
    Object.keys(DEFAULT_CAP_META).forEach(function (k) { merged[k] = DEFAULT_CAP_META[k]; });
    if (mod && mod.defaults) {
      Object.keys(mod.defaults).forEach(function (k) { merged[k] = mod.defaults[k]; });
    }
    if (perCap) {
      Object.keys(perCap).forEach(function (k) { merged[k] = perCap[k]; });
    }
    return merged;
  }

  function isWrite(capId) { return !!capabilityMeta(capId).write; }
  function isMatchResolvable(capId) { return !!capabilityMeta(capId).match_resolvable; }
  function isInfoOnly(capId) { return !!capabilityMeta(capId).info_only; }
  function isPinTarget(capId) { return !!capabilityMeta(capId).pin_target; }
  function isSelfTarget(capId) { return !!capabilityMeta(capId).self_target; }
  function policyFlagFor(capId) { return capabilityMeta(capId).policy_flag || null; }
  function isHideable(capId) { return !!capabilityMeta(capId).hideable; }
  function needsActiveTab(capId) { return !!capabilityMeta(capId).needs_active_tab; }

  // filesFor: the injection file list for a capability — files.write when opts.write is
  // true (and the module has any), else files.read. Returns [] for an unresolved capability
  // or a module missing that list (e.g. zillow has no write files).
  function filesFor(capId, opts) {
    const mod = moduleForCapability(capId);
    if (!mod || !mod.files) return [];
    const wantWrite = !!(opts && opts.write);
    const list = wantWrite ? mod.files.write : mod.files.read;
    return Array.isArray(list) ? list.slice() : [];
  }

  // entryFor: a page-context (MAIN world) entry-point name for a capability's module, e.g.
  // entryFor("fb.post.react", "act") -> "__soloActRun". null when unresolved.
  function entryFor(capId, name) {
    const mod = moduleForCapability(capId);
    if (!mod || !mod.entries) return null;
    return mod.entries[name] || null;
  }

  // ------------------------------------------------------------------
  // Derived legacy-shaped sets/objects — the same Set / plain-object shapes background.js
  // used to build by hand, now computed from EVERY module's capability metadata (so a module
  // added later, such as instagram, is covered without touching background.js). For the
  // Facebook ids the result is exactly the six literal tables above, since fbCapMeta derives
  // from them. A fresh copy is returned every call — callers may mutate their own copy freely.
  // ------------------------------------------------------------------
  function collect(pred) {
    const out = new Set();
    PLATFORM_MODULES.forEach(function (mod) {
      Object.keys(mod.capabilities || {}).forEach(function (id) { if (pred(capabilityMeta(id), id)) out.add(id); });
    });
    return out;
  }
  function writeActions() { return collect(function (m) { return !!m.write; }); }
  function matchResolvable() { return collect(function (m) { return !!m.match_resolvable; }); }
  function infoOnly() { return collect(function (m) { return !!m.info_only; }); }
  function pinTarget() { return collect(function (m) { return !!m.pin_target; }); }
  function selfTarget() { return collect(function (m) { return !!m.self_target; }); }
  function policyFlags() {
    const out = {};
    PLATFORM_MODULES.forEach(function (mod) {
      Object.keys(mod.capabilities || {}).forEach(function (id) { const f = capabilityMeta(id).policy_flag; if (f) out[id] = f; });
    });
    return out;
  }
  function hideable() { return collect(function (m) { return !!m.hideable; }); }

  root.SoloPlatforms = {
    PLATFORM_MODULES: PLATFORM_MODULES,
    moduleForCapability: moduleForCapability,
    moduleForHost: moduleForHost,
    capabilityMeta: capabilityMeta,
    isWrite: isWrite,
    isMatchResolvable: isMatchResolvable,
    isInfoOnly: isInfoOnly,
    isPinTarget: isPinTarget,
    isSelfTarget: isSelfTarget,
    policyFlagFor: policyFlagFor,
    isHideable: isHideable,
    needsActiveTab: needsActiveTab,
    filesFor: filesFor,
    entryFor: entryFor,
    dispatchModuleFor: dispatchModuleFor,
    dispatchFilesFor: dispatchFilesFor,
    dispatchEntryFor: dispatchEntryFor,
    writeActions: writeActions,
    matchResolvable: matchResolvable,
    infoOnly: infoOnly,
    pinTarget: pinTarget,
    selfTarget: selfTarget,
    policyFlags: policyFlags,
    hideable: hideable
  };
})(typeof self !== "undefined" ? self : globalThis);
