/*
 * ig_intercept.js — Solo Agency Local Collector, Instagram module
 *
 * Runs in the MAIN world at document_start on instagram.com (declared as a content_script in
 * manifest.json; merged into each live client's manifest by sync-dev-extension.sh). Its sole
 * job is to PASSIVELY observe Instagram's own traffic so ig_extract.js can, later in the same
 * page session, read structured data instead of scraping HTML.
 *
 * What it records, into a ring buffer on window.__soloIg.captures:
 *   - the Polaris GraphQL calls (POST /api/graphql and /graphql/query, form-encoded bodies
 *     carrying doc_id + fb_api_req_friendly_name + variables) — the profile query, the posts
 *     query, the keyword-search query, the post root query;
 *   - the REST calls under /api/v1/ (the comments list, top search) with their JSON.
 * Measured 2026-09-10 on instagram.com: the viewed profile is NOT embedded in the page's
 * data-sjs JSON (only the logged-in viewer is), it arrives by XHR as
 * PolarisProfilePageContentQuery — which is why this file has to run at document_start.
 *
 * Design contract, identical to the Facebook interceptor: never re-issue, never paginate,
 * never send anything off the page, always delegate to the native call, never throw into the
 * page. The request body of a GraphQL capture is kept verbatim so ig_extract.js can replay the
 * same query with a different cursor carrying the same session authority.
 */
(function () {
  "use strict";

  if (window.__soloIg && window.__soloIg.__installed) return;

  var MAX_CAPTURES = 60;
  var MAX_RESPONSE_BYTES = 4000000;
  var MAX_BODY_BYTES = 200000;

  var store = window.__soloIg || {};
  store.__installed = true;
  store.captures = store.captures || [];
  store.version = 1;
  store.origFetch = window.fetch ? window.fetch.bind(window) : null;
  window.__soloIg = store;

  function isGraphQL(u) {
    u = String(u || "");
    return u.indexOf("/api/graphql") > -1 || u.indexOf("/graphql/query") > -1;
  }
  // Only the REST endpoints a capability reads. Instagram's own /api/v1/ chatter (badge and inbox
  // polling, exposure logging, view beacons) would otherwise share — and evict from — the ring.
  var REST_KEEP = /\/api\/v1\/(media\/\d+\/(comments|info)\/|users\/search\/|web\/search\/topsearch\/|users\/web_profile_info\/|feed\/user\/)/;
  function isRest(u) {
    u = String(u || "");
    return u.indexOf("/api/v1/") > -1 && REST_KEEP.test(u);
  }

  // A newline-delimited GraphQL reply is one base document plus @defer/@stream chunks, each
  // carrying `path` (where it belongs) and `data` (what goes there). Assemble them into the
  // base so a reader sees the whole response; on any surprise, hand back the raw parts.
  function deepMerge(dst, src) {
    for (var k in src) {
      if (src[k] && typeof src[k] === "object" && !Array.isArray(src[k]) && dst[k] && typeof dst[k] === "object" && !Array.isArray(dst[k])) deepMerge(dst[k], src[k]);
      else dst[k] = src[k];
    }
    return dst;
  }
  function setPath(root, path, value) {
    var o = root;
    for (var i = 0; i < path.length - 1; i++) {
      var k = path[i];
      if (o[k] == null || typeof o[k] !== "object") o[k] = (typeof path[i + 1] === "number") ? [] : {};
      o = o[k];
    }
    var last = path[path.length - 1];
    if (o[last] && typeof o[last] === "object" && !Array.isArray(o[last]) && value && typeof value === "object" && !Array.isArray(value)) deepMerge(o[last], value);
    else o[last] = value;
  }
  function mergeParts(parts) {
    try {
      var base = parts[0];
      if (!base || typeof base !== "object" || Array.isArray(base)) return parts;
      for (var i = 1; i < parts.length; i++) {
        var p = parts[i];
        if (!p || typeof p !== "object" || !p.data || typeof p.data !== "object") continue;
        if (!base.data || typeof base.data !== "object") base.data = {};
        if (Array.isArray(p.path) && p.path.length) setPath(base.data, p.path, p.data);
        else deepMerge(base.data, p.data);
      }
      base.__soloParts = parts.length;
      return base;
    } catch (e) {
      return parts;
    }
  }
  store.mergeParts = mergeParts;
  function parseResponse(text) {
    if (!text) return null;
    try {
      var t = String(text).replace(/^for\s*\(;;\);/, "").trim();
      if (t.indexOf("\n") > -1) {
        var parts = t.split("\n").map(function (line) {
          line = line.trim();
          if (!line) return null;
          try { return JSON.parse(line); } catch (e) { return null; }
        }).filter(Boolean);
        if (parts.length > 1) return mergeParts(parts);
        if (parts.length === 1) return parts[0];
      }
      return JSON.parse(t);
    } catch (e) {
      return null;
    }
  }
  store.parseResponse = parseResponse;

  // The CSRF token Instagram's own XHRs carry (readable cookie) and the web app id every
  // Polaris request sends. ig_extract.js uses both only to REPLAY a captured query.
  store.csrfToken = function () {
    try {
      var m = document.cookie.match(/(?:^|;\s*)csrftoken=([^;]+)/);
      return m ? decodeURIComponent(m[1]) : "";
    } catch (e) { return ""; }
  };
  store.appId = function () {
    try {
      if (window.___xf) {
        var cfg = window.___xf("PolarisConfig");
        if (cfg && typeof cfg.getIGAppID === "function") return String(cfg.getIGAppID());
      }
    } catch (e) { /* fall through */ }
    return "936619743392459";
  };
  // doc_id for a named Polaris query, read from Instagram's own live module registry the way
  // the page itself does — the fallback when the query in question has not been captured yet.
  store.docIdFor = function (queryName) {
    try {
      if (!window.___xf) return "";
      var dbg = window.___xf("__debug");
      var mod = dbg && dbg.modulesMap ? dbg.modulesMap[String(queryName) + ".graphql"] : null;
      var params = mod && mod.exports && mod.exports.params;
      return params && params.id ? String(params.id) : "";
    } catch (e) { return ""; }
  };

  function bodyToString(body) {
    try {
      if (!body) return "";
      if (typeof body === "string") return body;
      if (typeof URLSearchParams !== "undefined" && body instanceof URLSearchParams) return body.toString();
      if (typeof FormData !== "undefined" && body instanceof FormData) {
        var p = new URLSearchParams();
        body.forEach(function (v, k) { p.append(k, String(v)); });
        return p.toString();
      }
    } catch (e) { /* ignore */ }
    return "";
  }

  function buildGraphQLCapture(body, url) {
    try {
      var s = bodyToString(body);
      if (!s) return null;
      var p = new URLSearchParams(s);
      var docId = p.get("doc_id");
      if (!docId) return null;
      var variables = {};
      try { variables = JSON.parse(p.get("variables") || "{}"); } catch (e) { variables = {}; }
      return {
        kind: "graphql",
        docId: docId,
        queryName: p.get("fb_api_req_friendly_name") || "",
        variables: variables,
        url: String(url || ""),
        requestBody: s.length <= MAX_BODY_BYTES ? s : "",
        capturedAt: Date.now(),
        response: null
      };
    } catch (e) {
      return null;
    }
  }

  function buildRestCapture(url, method) {
    try {
      var u = String(url || "");
      var path = u.replace(/^https?:\/\/[^/]+/, "");
      return {
        kind: "rest",
        docId: "",
        queryName: path.split("?")[0],
        variables: {},
        url: u,
        method: String(method || "GET").toUpperCase(),
        requestBody: "",
        capturedAt: Date.now(),
        response: null
      };
    } catch (e) {
      return null;
    }
  }

  function push(cap) {
    try {
      store.captures.push(cap);
      if (store.captures.length > MAX_CAPTURES) store.captures = store.captures.slice(-MAX_CAPTURES);
    } catch (e) { /* never throw into the page */ }
  }

  function attachResponse(cap, text) {
    if (!cap) return;
    try {
      if (text && text.length > MAX_RESPONSE_BYTES) {
        cap.response = null;
        cap.responseSkipped = "too_large";
      } else {
        cap.response = parseResponse(text);
      }
    } catch (e) {
      cap.response = null;
    }
    push(cap);
  }

  function captureFor(url, method, body) {
    var m = String(method || "GET").toUpperCase();
    if (isGraphQL(url) && m === "POST") return buildGraphQLCapture(body, url);
    if (isRest(url)) return buildRestCapture(url, m);
    return null;
  }

  // ---- fetch hook ---------------------------------------------------------
  try {
    var nativeFetch = window.fetch;
    if (typeof nativeFetch === "function") {
      window.fetch = function (input, init) {
        var url = "", method = "GET", body = null;
        try {
          url = (typeof input === "string") ? input : (input && input.url) || "";
          method = (init && init.method) || (input && input.method) || "GET";
          body = init && init.body;
        } catch (e) { /* fall through */ }
        var promise = nativeFetch.apply(this, arguments);
        try {
          var cap = captureFor(url, method, body);
          if (cap) {
            promise.then(function (resp) {
              try {
                resp.clone().text().then(function (text) { attachResponse(cap, text); }).catch(function () { push(cap); });
              } catch (e) { push(cap); }
            }).catch(function () { /* request failed; ignore */ });
          }
        } catch (e) { /* observation is best-effort only */ }
        return promise;
      };
    }
  } catch (e) { /* leave native fetch untouched */ }

  // ---- XMLHttpRequest hook ------------------------------------------------
  try {
    var XHR = window.XMLHttpRequest;
    if (XHR && XHR.prototype) {
      var nativeOpen = XHR.prototype.open;
      var nativeSend = XHR.prototype.send;
      XHR.prototype.open = function (method, url) {
        try { this.__soloIgUrl = url; this.__soloIgMethod = method; } catch (e) { /* ignore */ }
        return nativeOpen.apply(this, arguments);
      };
      XHR.prototype.send = function (body) {
        try {
          var cap = captureFor(this.__soloIgUrl, this.__soloIgMethod, body);
          if (cap) {
            this.addEventListener("load", function () {
              try { attachResponse(cap, this.responseText); } catch (e) { push(cap); }
            });
          }
        } catch (e) { /* observation is best-effort only */ }
        return nativeSend.apply(this, arguments);
      };
    }
  } catch (e) { /* leave native XHR untouched */ }
})();
