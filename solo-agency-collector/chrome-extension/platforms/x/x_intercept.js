/*
 * x_intercept.js — Solo Agency Local Collector, X (Twitter) module
 *
 * Runs in the MAIN world at document_start on x.com / twitter.com (declared as a content_script
 * in manifest.json; merged into each live client's manifest by sync-dev-extension.sh). Its sole
 * job is to PASSIVELY observe X's own traffic so x_extract.js can, later in the same page
 * session, read structured data instead of scraping HTML.
 *
 * What it records, into a ring buffer on window.__soloX.captures:
 *   - the GraphQL calls X's web client makes: GET/POST /i/api/graphql/<queryId>/<OperationName>
 *     (variables + features ride in the query string for GETs, in a JSON body for POSTs) — the
 *     profile query (UserByScreenName), the timelines (UserTweets, SearchTimeline, TweetDetail,
 *     HomeTimeline, Followers/Following) and the mutations the operator's own actions fire;
 *   - a few REST calls under /i/api/1.1 and /i/api/2 that a capability may read (typeahead,
 *     adaptive search) — everything else under /i/api is chatter and stays out of the ring.
 * The session headers X's own requests carry (the public web bearer, the ct0 csrf token, the
 * x-twitter-* flags) are kept on the store ONLY so x_extract.js can replay a captured query
 * with a new cursor; they are never written into a record.
 *
 * Design contract, identical to the Facebook and Instagram interceptors: never re-issue, never
 * paginate, never send anything off the page, always delegate to the native call, never throw
 * into the page.
 */
(function () {
  "use strict";

  if (window.__soloX && window.__soloX.__installed) return;

  var MAX_CAPTURES = 80;
  var MAX_RESPONSE_BYTES = 4000000;
  var MAX_BODY_BYTES = 200000;

  var store = window.__soloX || {};
  store.__installed = true;
  store.captures = store.captures || [];
  store.version = 1;
  store.origFetch = window.fetch ? window.fetch.bind(window) : null;
  // Replay-only session material (see header). Never copied into a capture or a record.
  store.__auth = store.__auth || { bearer: "", extra: {} };
  window.__soloX = store;

  var GRAPHQL = /\/i\/api\/graphql\/([^/?#]+)\/([^/?#]+)/;
  // Only the REST endpoints a capability reads.
  var REST_KEEP = /\/i\/api\/(1\.1\/search\/typeahead\.json|2\/search\/adaptive\.json|1\.1\/users\/(show|lookup)\.json)/;

  function isGraphQL(u) { return GRAPHQL.test(String(u || "")); }
  function isRest(u) { return REST_KEEP.test(String(u || "")); }

  function parseResponse(text) {
    if (!text) return null;
    try { return JSON.parse(String(text).trim()); } catch (e) { return null; }
  }
  store.parseResponse = parseResponse;

  store.csrf = function () {
    try {
      var m = document.cookie.match(/(?:^|;\s*)ct0=([^;]+)/);
      return m ? decodeURIComponent(m[1]) : "";
    } catch (e) { return ""; }
  };
  // The headers a replayed query needs — the same ones X's own client sent last.
  store.headersFor = function () {
    var h = {
      "authorization": store.__auth.bearer || "",
      "x-csrf-token": store.csrf(),
      "x-twitter-active-user": "yes",
      "x-twitter-auth-type": "OAuth2Session",
      "x-twitter-client-language": store.__auth.extra["x-twitter-client-language"] || "en"
    };
    return h;
  };
  // queryId for a named operation, from the latest capture that used it.
  store.queryIdFor = function (name) {
    try {
      for (var i = store.captures.length - 1; i >= 0; i--) {
        var c = store.captures[i];
        if (c && c.kind === "graphql" && c.queryName === String(name)) return c.queryId;
      }
    } catch (e) { /* fall through */ }
    return "";
  };

  function rememberHeaders(headers) {
    try {
      if (!headers) return;
      var get = function (name) {
        if (typeof headers.get === "function") return headers.get(name);
        var keys = Object.keys(headers);
        for (var i = 0; i < keys.length; i++) { if (keys[i].toLowerCase() === name) return headers[keys[i]]; }
        return null;
      };
      var bearer = get("authorization");
      if (bearer && /^Bearer\s/i.test(String(bearer))) store.__auth.bearer = String(bearer);
      ["x-twitter-client-language", "x-twitter-auth-type", "x-twitter-active-user"].forEach(function (k) {
        var v = get(k); if (v) store.__auth.extra[k] = String(v);
      });
    } catch (e) { /* observation only */ }
  }

  function bodyToString(body) {
    try {
      if (!body) return "";
      if (typeof body === "string") return body;
      if (typeof URLSearchParams !== "undefined" && body instanceof URLSearchParams) return body.toString();
    } catch (e) { /* ignore */ }
    return "";
  }
  function parseJson(s, fallback) { try { return s ? JSON.parse(s) : fallback; } catch (e) { return fallback; } }

  function buildGraphQLCapture(url, method, body) {
    try {
      var u = String(url || "");
      var m = u.match(GRAPHQL);
      if (!m) return null;
      var variables = {}, features = {};
      var q = "";
      try { q = new URL(u, location.origin).search; } catch (e) { q = ""; }
      var p = new URLSearchParams(q);
      if (p.get("variables")) variables = parseJson(p.get("variables"), {});
      if (p.get("features")) features = parseJson(p.get("features"), {});
      var bodyStr = bodyToString(body);
      if (bodyStr && /^\s*\{/.test(bodyStr)) {
        var j = parseJson(bodyStr, {});
        if (j && typeof j === "object") {
          if (j.variables && typeof j.variables === "object") variables = j.variables;
          if (j.features && typeof j.features === "object") features = j.features;
        }
      }
      return {
        kind: "graphql",
        queryId: m[1],
        queryName: m[2],
        method: String(method || "GET").toUpperCase(),
        variables: variables,
        features: features,
        url: u,
        requestBody: bodyStr.length <= MAX_BODY_BYTES ? bodyStr : "",
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
        kind: "rest", queryId: "", queryName: path.split("?")[0], method: String(method || "GET").toUpperCase(),
        variables: {}, features: {}, url: u, requestBody: "", capturedAt: Date.now(), response: null
      };
    } catch (e) { return null; }
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
      if (text && text.length > MAX_RESPONSE_BYTES) { cap.response = null; cap.responseSkipped = "too_large"; }
      else cap.response = parseResponse(text);
    } catch (e) { cap.response = null; }
    push(cap);
  }

  function captureFor(url, method, body) {
    if (isGraphQL(url)) return buildGraphQLCapture(url, method, body);
    if (isRest(url)) return buildRestCapture(url, method);
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
          if (init && init.headers) rememberHeaders(init.headers);
          else if (input && input.headers) rememberHeaders(input.headers);
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
      var nativeSet = XHR.prototype.setRequestHeader;
      XHR.prototype.open = function (method, url) {
        try { this.__soloXUrl = url; this.__soloXMethod = method; this.__soloXHeaders = {}; } catch (e) { /* ignore */ }
        return nativeOpen.apply(this, arguments);
      };
      XHR.prototype.setRequestHeader = function (name, value) {
        try { if (this.__soloXHeaders) this.__soloXHeaders[String(name).toLowerCase()] = String(value); } catch (e) { /* ignore */ }
        return nativeSet.apply(this, arguments);
      };
      XHR.prototype.send = function (body) {
        try {
          rememberHeaders(this.__soloXHeaders);
          var cap = captureFor(this.__soloXUrl, this.__soloXMethod, body);
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
