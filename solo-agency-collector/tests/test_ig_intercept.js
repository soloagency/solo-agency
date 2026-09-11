/*
 * Offline tests for chrome-extension/platforms/instagram/ig_intercept.js — the MAIN-world
 * interceptor. It is loaded into a vm with a fake window (fetch + XMLHttpRequest stubs, a cookie)
 * and asserted on through the window.__soloIg store it installs:
 *   - only the REST endpoints a capability reads enter the ring (Instagram's /api/v1/ chatter
 *     must not evict them);
 *   - a newline-delimited GraphQL reply (@defer/@stream chunks) is assembled into one document
 *     so no reader ever sees just the first line;
 *   - captures carry docId/queryName/variables/requestBody; the hook never throws into the page.
 * Run: node solo-agency-collector/tests/test_ig_intercept.js
 */
"use strict";
const fs = require("fs");
const path = require("path");
const vm = require("vm");
const SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "platforms", "instagram", "ig_intercept.js"), "utf8");

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail).slice(0, 300) : "")); }
}

function makeCtx(responses) {
  // responses: url substring -> body text the fake fetch answers with
  const nativeFetch = function (input, init) {
    const url = typeof input === "string" ? input : input.url;
    const key = Object.keys(responses).find((k) => url.indexOf(k) !== -1);
    const text = key ? responses[key] : "{}";
    return Promise.resolve({ clone: () => ({ text: () => Promise.resolve(text) }), text: () => Promise.resolve(text) });
  };
  function XHR() { this.listeners = {}; }
  XHR.prototype.open = function (m, u) { this.m = m; this.u = u; };
  XHR.prototype.addEventListener = function (ev, fn) { this.listeners[ev] = fn; };
  XHR.prototype.send = function (body) {
    const key = Object.keys(responses).find((k) => String(this.u).indexOf(k) !== -1);
    this.responseText = key ? responses[key] : "{}";
    const self = this;
    setTimeout(() => { if (self.listeners.load) self.listeners.load.call(self); }, 0);
  };
  const ctx = { console, setTimeout, clearTimeout, Promise, JSON, URLSearchParams, FormData: undefined, document: { cookie: "ig_did=x; csrftoken=tok123; ds_user_id=1" }, fetch: nativeFetch, XMLHttpRequest: XHR };
  ctx.window = ctx;
  vm.createContext(ctx);
  vm.runInContext(SRC, ctx, { filename: "ig_intercept.js" });
  return ctx;
}
const tick = () => new Promise((r) => setTimeout(r, 5));

(async () => {
  console.log("install: store shape, csrf from cookie, idempotent re-install");
  {
    const ctx = makeCtx({});
    const s = ctx.window.__soloIg;
    check("store installed with captures[], version, origFetch, parseResponse, mergeParts", !!s && Array.isArray(s.captures) && s.version === 1 && typeof s.origFetch === "function" && typeof s.parseResponse === "function" && typeof s.mergeParts === "function", Object.keys(s || {}));
    check("csrfToken() reads the csrftoken cookie", s.csrfToken() === "tok123", s.csrfToken());
    check("appId() falls back to the web app id", s.appId() === "936619743392459", s.appId());
    const before = ctx.window.fetch;
    vm.runInContext(SRC, ctx, { filename: "ig_intercept.js" });
    check("re-install is a no-op (fetch hook not double-wrapped)", ctx.window.fetch === before);
  }

  console.log("REST ring: only capability endpoints are recorded");
  {
    const ctx = makeCtx({ "/api/v1/": JSON.stringify({ comments: [], ok: true }) });
    const urls = [
      "https://www.instagram.com/api/v1/media/3972649630022147867/comments/?can_support_threading=true",
      "https://www.instagram.com/api/v1/users/search/?q=realtor&count=30",
      "https://www.instagram.com/api/v1/web/search/topsearch/?context=blended&query=realtor",
      "https://www.instagram.com/api/v1/users/web_profile_info/?username=x",
      "https://www.instagram.com/api/v1/media/1/info/",
      "https://www.instagram.com/api/v1/feed/user/123/username/?count=12",
      // chatter that must NOT enter the ring
      "https://www.instagram.com/api/v1/notifications/badge/",
      "https://www.instagram.com/api/v1/direct_v2/inbox/?persistentBadging=true",
      "https://www.instagram.com/api/v1/web/accounts/login/ajax/",
      "https://www.instagram.com/api/v1/qe/sync/",
      "https://www.instagram.com/api/v1/feed/reels_tray/",
    ];
    for (const u of urls) ctx.window.fetch(u, { method: "GET" });
    await tick(); await tick();
    const got = ctx.window.__soloIg.captures.map((c) => c.queryName);
    check("6 capability endpoints captured, 5 chatter urls skipped", got.length === 6 && got.every((q) => /comments|users\/search|topsearch|web_profile_info|media\/1\/info|feed\/user/.test(q)), got);
    check("REST capture carries kind rest, method GET, parsed JSON", ctx.window.__soloIg.captures[0].kind === "rest" && ctx.window.__soloIg.captures[0].method === "GET" && ctx.window.__soloIg.captures[0].response.ok === true, ctx.window.__soloIg.captures[0]);
  }

  console.log("GraphQL capture: doc_id + friendly name + variables + verbatim body (fetch and XHR)");
  {
    const ctx = makeCtx({ "/graphql/query": JSON.stringify({ data: { user: { id: "1" } } }) });
    const body = "av=0&doc_id=28036671149327607&fb_api_req_friendly_name=PolarisProfilePageContentQuery&variables=" + encodeURIComponent(JSON.stringify({ id: "1", render_surface: "PROFILE" }));
    ctx.window.fetch("https://www.instagram.com/graphql/query", { method: "POST", body });
    const x = new ctx.XMLHttpRequest(); x.open("POST", "https://www.instagram.com/api/graphql"); x.send("doc_id=42&fb_api_req_friendly_name=PolarisPostRootQuery&variables=%7B%7D");
    ctx.window.fetch("https://www.instagram.com/graphql/query", { method: "POST", body: "lsd=abc&variables=%7B%7D" }); // no doc_id: ignored
    await tick(); await tick();
    const caps = ctx.window.__soloIg.captures;
    check("two graphql captures (fetch + xhr), the doc_id-less one ignored", caps.length === 2 && caps.every((c) => c.kind === "graphql"), caps.map((c) => c.docId));
    const f = caps.find((c) => c.docId === "28036671149327607");
    check("fetch capture: queryName, variables parsed, requestBody verbatim, response parsed", f && f.queryName === "PolarisProfilePageContentQuery" && f.variables.render_surface === "PROFILE" && f.requestBody === body && f.response.data.user.id === "1", f);
    check("xhr capture: queryName from the form body", caps.some((c) => c.docId === "42" && c.queryName === "PolarisPostRootQuery"), caps.map((c) => c.queryName));
  }

  console.log("parseResponse: for(;;); prefix, single document, multi-chunk assembly");
  {
    const s = makeCtx({}).window.__soloIg;
    check("strips the anti-hijack prefix", s.parseResponse('for (;;);{"data":{"a":1}}').data.a === 1);
    const base = { data: { xdt_api__v1__feed__user_timeline_graphql_connection: { edges: [{ node: { pk: "1" } }], page_info: { has_next_page: true } } } };
    const chunk1 = { label: "x$defer$0", path: ["xdt_api__v1__feed__user_timeline_graphql_connection", "edges", 1], data: { node: { pk: "2" } } };
    const chunk2 = { label: "x$defer$1", path: ["xdt_api__v1__feed__user_timeline_graphql_connection", "page_info"], data: { end_cursor: "c2" } };
    const chunk3 = { data: { extra: { hello: "world" } } };
    const merged = s.parseResponse([JSON.stringify(base), JSON.stringify(chunk1), JSON.stringify(chunk2), JSON.stringify(chunk3), "", "not json"].join("\n"));
    const conn = merged && merged.data && merged.data.xdt_api__v1__feed__user_timeline_graphql_connection;
    check("returns ONE document, not an array", !Array.isArray(merged) && !!conn, merged && Object.keys(merged));
    check("@stream chunk landed at its array path", conn.edges.length === 2 && conn.edges[1].node.pk === "2", conn.edges);
    check("@defer chunk merged into the object at its path (existing keys kept)", conn.page_info.has_next_page === true && conn.page_info.end_cursor === "c2", conn.page_info);
    check("path-less chunk merged into data root; __soloParts counts the chunks", merged.data.extra.hello === "world" && merged.__soloParts === 4, [merged.data.extra, merged.__soloParts]);
    check("a lone line is returned as-is", s.parseResponse('{"data":{"b":2}}\n').data.b === 2);
    check("garbage never throws", s.parseResponse("{{{") === null && s.parseResponse("") === null);
    const weird = s.mergeParts([[1, 2], { data: { a: 1 } }]);
    check("mergeParts hands back the raw parts when the base is not a document", Array.isArray(weird) && Array.isArray(weird[0]));
  }

  console.log("");
  console.log(fail === 0 ? "ALL " + pass + " CHECKS PASSED" : pass + " passed, " + fail + " FAILED");
  process.exit(fail ? 1 : 0);
})().catch((e) => { console.error(e); process.exit(2); });
