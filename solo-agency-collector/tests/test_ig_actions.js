/*
 * Offline tests for chrome-extension/platforms/instagram/ig_actions.js (ig.post.react,
 * ig.post.comment, ig.message.send). ig_extract.js is loaded first (the actions read the
 * media id through its internals), then ig_actions.js, in a vm with a fake window: captures
 * shaped like the embedded post root, a fetch stub for the web endpoints, and hand-built DOM
 * nodes for the like control, the comment box, the Message button and the thread composer.
 * Run: node solo-agency-collector/tests/test_ig_actions.js
 */
"use strict";
const fs = require("fs");
const path = require("path");
const vm = require("vm");
const EXTRACT = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "platforms", "instagram", "ig_extract.js"), "utf8");
const ACTIONS = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "platforms", "instagram", "ig_actions.js"), "utf8");
let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail).slice(0, 400) : "")); }
}
function fakeEl(opts) {
  opts = opts || {};
  const el = {
    tagName: opts.tag || "DIV", _attrs: opts.attrs || {}, innerText: opts.innerText || "", value: "", _children: opts.children || [],
    getAttribute(k) { return this._attrs[k] === undefined ? null : this._attrs[k]; },
    getBoundingClientRect() { return { width: opts.hidden ? 0 : 100, height: opts.hidden ? 0 : 20 }; },
    focus() { el._doc && (el._doc._active = el); }, dispatchEvent(e) { if (opts.onEvent) opts.onEvent(e, el); return true; },
    closest(sel) { return opts.closestEl || (sel && String(sel).indexOf("button") !== -1 ? el : el); }, querySelector() { return null; }, querySelectorAll() { return []; },
  };
  return el;
}
function fetchStub(rules) {
  const calls = [];
  const fn = function (url, init) {
    calls.push({ url: String(url), init: init || {} });
    for (const r of rules) { if (r.match(String(url), init || {})) return Promise.resolve({ status: r.status || 200, text: () => Promise.resolve(JSON.stringify(r.json)) }); }
    return Promise.resolve({ status: 404, text: () => Promise.resolve("{}") });
  };
  fn.calls = calls; return fn;
}
function postRootCapture(code, pk) {
  return { kind: "graphql", queryName: "PolarisPostRootQuery", docId: "r", variables: {}, url: "https://www.instagram.com/graphql/query", requestBody: "", capturedAt: Date.now(), response: { data: { xdt_api__v1__media__shortcode__web_info: { items: [{ pk, id: pk + "_1", code, caption: { text: "hi" }, user: { pk: "9", username: "nhuwhite" }, like_count: 8, comment_count: 8, taken_at: 1 }] } } } };
}
function makeCtx(opts) {
  opts = opts || {};
  const pathname = opts.pathname || "/";
  const nodes = opts.nodes || {}; // selector substring -> [els]
  const document = {
    _active: null, cookie: "csrftoken=tok",
    querySelector: (sel) => { for (const k of Object.keys(nodes)) { if (String(sel).indexOf(k) !== -1 && nodes[k].length) return nodes[k][0]; } return null; },
    querySelectorAll: (sel) => { let out = []; for (const k of Object.keys(nodes)) { if (String(sel).indexOf(k) !== -1) out = out.concat(nodes[k]); } return out; },
    execCommand: (cmd, _u, text) => { if (cmd === "insertText" && document._active) document._active.innerText = text; return true; },
    body: { get innerText() { return opts.bodyText ? opts.bodyText() : ""; } },
  };
  Object.values(nodes).forEach((list) => list.forEach((el) => { el._doc = document; }));
  const store = { captures: opts.captures || [], origFetch: opts.origFetch || fetchStub([]), csrfToken: () => "tok", appId: () => "936619743392459", parseResponse: (t) => { try { return JSON.parse(t); } catch (e) { return null; } }, docIdFor: () => "" };
  const ctx = { document, location: { pathname, href: "https://www.instagram.com" + pathname, origin: "https://www.instagram.com" }, console, setTimeout, clearTimeout, URL, URLSearchParams, Promise, Date, JSON, MouseEvent: function (t) { this.type = t; }, KeyboardEvent: function (t) { this.type = t; }, InputEvent: function (t) { this.type = t; }, Event: function (t) { this.type = t; }, HTMLTextAreaElement: { prototype: {} }, HTMLInputElement: { prototype: {} } };
  ctx.window = ctx; ctx.window.__soloIg = store;
  vm.createContext(ctx);
  vm.runInContext(EXTRACT, ctx, { filename: "ig_extract.js" });
  vm.runInContext(ACTIONS, ctx, { filename: "ig_actions.js" });
  return ctx;
}
const likeSvg = (label, btn) => { const svg = fakeEl({ tag: "SVG", attrs: { "aria-label": label }, closestEl: btn }); return svg; };

(async () => {
  console.log("ig.post.react — guards, dry_run, like, already");
  {
    const r0 = await makeCtx({ pathname: "/nhuwhite/" }).window.__soloIgAct("ig.post.react", { _target_url: "https://www.instagram.com/nhuwhite/" });
    check("a profile url is refused (not_a_post_url)", r0.status === "error" && /not_a_post_url/.test(r0.items[0].error), r0.items[0].error);
    const r1 = await makeCtx({ pathname: "/p/OTHER/" }).window.__soloIgAct("ig.post.react", { _target_url: "https://www.instagram.com/p/ABC/" });
    check("landing on a different post refuses (post_mismatch)", r1.status === "error" && /post_mismatch/.test(r1.items[0].error), r1.items[0].error);
    const r2 = await makeCtx({ pathname: "/p/ABC/" }).window.__soloIgAct("ig.post.react", { _target_url: "https://www.instagram.com/p/ABC/" });
    check("no embedded post root -> media_id_unknown", r2.status === "error" && /media_id_unknown/.test(r2.items[0].error), r2.items[0].error);
    const btn = fakeEl({ attrs: { role: "button" } });
    const nodes = { "svg[aria-label]": [likeSvg("Like", btn)] };
    const ctx = makeCtx({ pathname: "/p/ABC/", captures: [postRootCapture("ABC", "123")], nodes });
    const d = await ctx.window.__soloIgAct("ig.post.react", { _target_url: "https://www.instagram.com/p/ABC/", dry_run: true });
    check("dry_run resolves the media id and reads the state without calling the endpoint", d.status === "dry_run" && d.items[0].media_id === "123" && d.items[0].like_control_found === true && d.items[0].current_state === "not_liked" && ctx.window.__soloIg.origFetch.calls.length === 0, d.items[0]);
    const nodes2 = { "svg[aria-label]": [] };
    const btn2 = fakeEl({ attrs: { role: "button" }, onEvent: (e) => { if (e.type === "click") { nodes2["svg[aria-label]"] = [likeSvg("Unlike", btn2)]; ctx2.window.__soloIg.captures.push({ kind: "graphql", queryName: "usePolarisLikeMediaLikeMutation", capturedAt: Date.now(), response: { data: {} } }); } } });
    nodes2["svg[aria-label]"] = [likeSvg("Like", btn2)];
    const ctx2 = makeCtx({ pathname: "/p/ABC/", captures: [postRootCapture("ABC", "123")], nodes: nodes2 });
    const r = await ctx2.window.__soloIgAct("ig.post.react", { _target_url: "https://www.instagram.com/p/ABC/" });
    check("like: the post's own control is clicked, flips to Unlike, mutation seen -> done verified, no endpoint called", r.status === "done" && r.items[0].verified === true && r.items[0].state_after === "liked" && /Like/.test(r.items[0].mutation_seen) && ctx2.window.__soloIg.origFetch.calls.length === 0, r.items[0]);
    const ctx3 = makeCtx({ pathname: "/p/ABC/", captures: [postRootCapture("ABC", "123")], nodes: { "svg[aria-label]": [likeSvg("Unlike", btn)] } });
    const a = await ctx3.window.__soloIgAct("ig.post.react", { _target_url: "https://www.instagram.com/p/ABC/" });
    check("already liked -> status already, nothing sent", a.status === "already" && ctx3.window.__soloIg.origFetch.calls.length === 0, a.items[0]);
  }

  console.log("ig.post.comment — dry_run, send, comments off");
  {
    const box = fakeEl({ tag: "TEXTAREA", attrs: { "aria-label": "Add a comment…" } });
    const stub = fetchStub([{ match: (u, i) => u.indexOf("/api/v1/web/comments/123/add/") !== -1 && /comment_text=Nice\+work/.test(i.body), json: { id: "555", text: "Nice work", status: "ok" } }]);
    const ctx = makeCtx({ pathname: "/p/ABC/", captures: [postRootCapture("ABC", "123")], nodes: { "textarea": [box] }, origFetch: stub, bodyText: () => "Nice work" });
    const d = await ctx.window.__soloIgAct("ig.post.comment", { _target_url: "https://www.instagram.com/p/ABC/", text: "Nice work", dry_run: true });
    check("dry_run: media id + comment box found, nothing sent", d.status === "dry_run" && d.items[0].media_id === "123" && d.items[0].comment_box_found === true && stub.calls.length === 0, d.items[0]);
    const r = await ctx.window.__soloIgAct("ig.post.comment", { _target_url: "https://www.instagram.com/p/ABC/", text: "Nice work" });
    check("send: POST comment_text, comment id + url back, verified", r.status === "done" && r.items[0].comment_id === "555" && /\/p\/ABC\/c\/555\//.test(r.items[0].comment_url) && r.items[0].verified === true && r.items[0].appeared === true, r.items[0]);
    const off = await makeCtx({ pathname: "/p/ABC/", captures: [postRootCapture("ABC", "123")], nodes: {} }).window.__soloIgAct("ig.post.comment", { _target_url: "https://www.instagram.com/p/ABC/", text: "x" });
    check("no comment box (comments off) -> error, nothing sent", off.status === "error" && /no comment box/.test(off.items[0].error), off.items[0].error);
    const empty = await makeCtx({ pathname: "/p/ABC/" }).window.__soloIgAct("ig.post.comment", { _target_url: "https://www.instagram.com/p/ABC/", text: "  " });
    check("empty text refused", empty.status === "error" && /no comment text/.test(empty.items[0].error));
  }

  console.log("ig.message.send — guards and dry_run on the thread");
  {
    const r0 = await makeCtx({ pathname: "/p/ABC/" }).window.__soloIgAct("ig.message.send", { _target_url: "https://www.instagram.com/p/ABC/", text: "hi" });
    check("a post url is refused (not_a_profile_url)", r0.status === "error" && /not_a_profile_url/.test(r0.items[0].error), r0.items[0].error);
    const r1 = await makeCtx({ pathname: "/someone_else/" }).window.__soloIgAct("ig.message.send", { _target_url: "https://www.instagram.com/binhnguyen_ai/", text: "hi" });
    check("landing on another profile refuses (recipient_mismatch)", r1.status === "error" && /recipient_mismatch/.test(r1.items[0].error), r1.items[0].error);
    const r2 = await makeCtx({ pathname: "/binhnguyen_ai/", nodes: {} }).window.__soloIgAct("ig.message.send", { _target_url: "https://www.instagram.com/binhnguyen_ai/", text: "hi" });
    check("no Message button -> error, nothing typed", r2.status === "error" && /no Message button/.test(r2.items[0].error), r2.items[0].error);
    const composer = fakeEl({ attrs: { contenteditable: "true", role: "textbox", "aria-label": "Message" } });
    const headerLink = fakeEl({ tag: "A", attrs: { href: "/binhnguyen_ai/" } });
    const ctx = makeCtx({ pathname: "/direct/t/123/", nodes: { 'div[contenteditable="true"][role="textbox"]': [composer], 'a[href^="/"]': [headerLink] } });
    const d = await ctx.window.__soloIgAct("ig.message.send", { _target_url: "https://www.instagram.com/binhnguyen_ai/", username: "binhnguyen_ai", text: "hello", dry_run: true });
    check("dry_run on the thread: recipient from the header, composer found, nothing typed", d.status === "dry_run" && d.items[0].recipient === "binhnguyen_ai" && d.items[0].composer_found === true && composer.innerText === "", d.items[0]);
    const wrong = await ctx.window.__soloIgAct("ig.message.send", { _target_url: "https://www.instagram.com/nhuwhite/", username: "nhuwhite", text: "hello" });
    check("thread header names somebody else -> recipient_mismatch, nothing typed", wrong.status === "error" && /recipient_mismatch/.test(wrong.items[0].error) && composer.innerText === "", wrong.items[0].error);
    const bad = await ctx.window.__soloIgAct("ig.nope", {});
    check("unknown action -> visible error", bad.available === false && /unknown or unimplemented/.test(bad.items[0].error));
  }

  console.log("");
  console.log(fail === 0 ? "ALL " + pass + " CHECKS PASSED" : pass + " passed, " + fail + " FAILED");
  process.exit(fail ? 1 : 0);
})().catch((e) => { console.error(e); process.exit(2); });
