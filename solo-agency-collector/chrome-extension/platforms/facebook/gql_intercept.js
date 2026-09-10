/*
 * gql_intercept.js — Solo Agency Local Collector
 *
 * PHASE 1 (hybrid capture). Runs in the MAIN world at document_start on
 * facebook.com only (declared as a content_script in manifest.json). Its sole
 * job is to PASSIVELY observe Facebook's own internal GraphQL traffic so the
 * collector can, later in the same page session, read structured data instead
 * of scraping HTML.
 *
 * Design contract — this file must be boring and safe:
 *   - It ONLY hooks window.fetch + XMLHttpRequest to record request/response
 *     pairs for /api/graphql and /graphql/query into a small in-memory ring
 *     buffer on window.__soloGql.captures.
 *   - It NEVER re-issues requests, NEVER paginates, NEVER sends anything off the
 *     page. (Replay/pagination is a later, per-screen phase.)
 *   - It always delegates to the original fetch/XHR and never throws into the
 *     page: every hook is wrapped so Facebook behaves exactly as if we were not
 *     here. If anything goes wrong we silently fall back to the native call.
 *
 * Nothing reads this buffer except the collector's own gql_extract.js, injected
 * into the same MAIN world only while a collection job is running.
 */
(function () {
  "use strict";

  // Guard against double-install (SPA soft-navigations, re-injection).
  if (window.__soloGql && window.__soloGql.__installed) return;

  var MAX_CAPTURES = 50;          // ring-buffer size (per page/tab)
  var MAX_RESPONSE_BYTES = 4000000; // skip storing pathologically large bodies

  var store = window.__soloGql || {};
  store.__installed = true;
  store.captures = store.captures || [];
  store.version = 1;
  // Keep a handle to the pristine fetch so a later phase could replay without
  // being re-captured. Unused in Phase 1, but cheap and handy to expose.
  store.origFetch = window.fetch ? window.fetch.bind(window) : null;
  window.__soloGql = store;

  function isGQL(u) {
    if (!u) return false;
    u = String(u);
    return u.indexOf("/api/graphql") > -1 || u.indexOf("/graphql/query") > -1;
  }

  // Strip Facebook's anti-JSON-hijack prefix and parse either a single JSON
  // object or a newline-delimited multi-part (streaming) GraphQL response.
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
        if (parts.length > 1) return parts;
        if (parts.length === 1) return parts[0];
      }
      return JSON.parse(t);
    } catch (e) {
      return null;
    }
  }
  store.parseResponse = parseResponse;

  // Pull the identifying bits out of a GraphQL request body. Returns null unless
  // the request carries a doc_id (persisted query) — that filter matches
  // essentially all of Facebook's Comet GraphQL traffic and skips ad-hoc noise.
  function buildCapture(body, url) {
    try {
      var p = new URLSearchParams(body);
      var docId = p.get("doc_id");
      if (!docId) return null;
      var variables = {};
      try { variables = JSON.parse(p.get("variables") || "{}"); } catch (e) { variables = {}; }
      return {
        docId: docId,
        queryName: p.get("fb_api_req_friendly_name") || "",
        variables: variables,
        // fb_dtsg (CSRF) + av (actor id) are needed only to REPLAY a query for
        // deeper pagination (see __soloGqlPaginate). Captured here so the replay
        // carries the same session authority as Facebook's own request.
        fbDtsg: p.get("fb_dtsg") || "",
        av: p.get("av") || "",
        url: String(url || ""),
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
      if (store.captures.length > MAX_CAPTURES) {
        store.captures = store.captures.slice(-MAX_CAPTURES);
      }
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

  // ---- fetch hook ---------------------------------------------------------
  try {
    var nativeFetch = window.fetch;
    if (typeof nativeFetch === "function") {
      window.fetch = function (input, init) {
        var url = "";
        var method = "GET";
        var body = null;
        try {
          url = (typeof input === "string") ? input : (input && input.url) || "";
          method = (init && init.method) || (input && input.method) || "GET";
          body = init && init.body;
        } catch (e) { /* fall through to native */ }

        var promise = nativeFetch.apply(this, arguments);

        try {
          if (isGQL(url) && String(method).toUpperCase() === "POST" && typeof body === "string") {
            var cap = buildCapture(body, url);
            if (cap) {
              promise.then(function (resp) {
                try {
                  resp.clone().text().then(function (text) {
                    attachResponse(cap, text);
                  }).catch(function () { push(cap); });
                } catch (e) { push(cap); }
              }).catch(function () { /* request failed; ignore */ });
            }
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
        try {
          this.__soloGqlUrl = url;
          this.__soloGqlMethod = method;
        } catch (e) { /* ignore */ }
        return nativeOpen.apply(this, arguments);
      };

      XHR.prototype.send = function (body) {
        try {
          var url = this.__soloGqlUrl;
          var method = this.__soloGqlMethod;
          if (isGQL(url) && String(method || "").toUpperCase() === "POST" && typeof body === "string") {
            var cap = buildCapture(body, url);
            if (cap) {
              this.addEventListener("load", function () {
                try { attachResponse(cap, this.responseText); }
                catch (e) { push(cap); }
              });
            }
          }
        } catch (e) { /* observation is best-effort only */ }
        return nativeSend.apply(this, arguments);
      };
    }
  } catch (e) { /* leave native XHR untouched */ }
})();
