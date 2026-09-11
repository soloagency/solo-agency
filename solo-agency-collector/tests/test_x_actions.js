/*
 * Offline tests for chrome-extension/platforms/x/x_actions.js (x.post.like, x.post.reply,
 * x.post.publish, x.dm.send). Loaded into a vm with a fake window: hand-built DOM nodes for the
 * focal article + Like control, the composer + submit button, the DM controls, and a
 * window.__soloX capture ring the fake page appends mutations to when a control is clicked —
 * the same proof the live page gives. Run: node solo-agency-collector/tests/test_x_actions.js
 */
"use strict";
const fs = require("fs");
const path = require("path");
const vm = require("vm");
const SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "platforms", "x", "x_actions.js"), "utf8");
let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail).slice(0, 400) : "")); }
}
function fakeEl(opts) {
  opts = opts || {};
  const el = { tagName: opts.tag || "DIV", _attrs: opts.attrs || {}, innerText: opts.innerText || "", disabled: false, _q: opts.q || {},
    getAttribute(k) { return this._attrs[k] === undefined ? null : this._attrs[k]; },
    getBoundingClientRect() { return { width: opts.hidden ? 0 : 100, height: opts.hidden ? 0 : 20 }; },
    focus() { if (el._doc) el._doc._active = el; }, dispatchEvent(e) { if (opts.onEvent) opts.onEvent(e, el); return true; },
    querySelector(sel) { for (const k of Object.keys(el._q)) { if (String(sel).indexOf(k) !== -1) return el._q[k]; } return null; }, querySelectorAll() { return []; } };
  return el;
}
function makeCtx(opts) {
  opts = opts || {};
  const pathname = opts.pathname || "/";
  const nodes = opts.nodes || {};
  const document = { _active: null,
    querySelector: (sel) => { for (const k of Object.keys(nodes)) { if (String(sel).indexOf(k) !== -1 && nodes[k].length) return nodes[k][0]; } return null; },
    querySelectorAll: (sel) => { let out = []; for (const k of Object.keys(nodes)) { if (String(sel).indexOf(k) !== -1) out = out.concat(nodes[k]); } return out; },
    execCommand: (cmd, _u, text) => { if (cmd === "insertText" && document._active) document._active.innerText = text; return true; },
    body: { get innerText() { return opts.bodyText ? opts.bodyText() : ""; } } };
  Object.values(nodes).forEach((list) => list.forEach((el) => { el._doc = document; }));
  const store = { captures: opts.captures || [] };
  const ctx = { document, location: { pathname, href: "https://x.com" + pathname, origin: "https://x.com" }, console, setTimeout, clearTimeout, URL, Promise, Date, JSON, MouseEvent: function (t) { this.type = t; }, KeyboardEvent: function () {}, InputEvent: function () {} };
  ctx.window = ctx; ctx.window.__soloX = store;
  vm.createContext(ctx);
  vm.runInContext(SRC, ctx, { filename: "x_actions.js" });
  return ctx;
}
const createTweetCapture = (id, handle) => ({ kind: "graphql", queryName: "CreateTweet", queryId: "q", method: "POST", variables: {}, url: "https://x.com/i/api/graphql/q/CreateTweet", capturedAt: Date.now(), response: { data: { create_tweet: { tweet_results: { result: { rest_id: id, core: { user_results: { result: { core: { screen_name: handle } } } }, legacy: { id_str: id } } } } } } });
const P = "2098031513552187670";

(async () => {
  console.log("x.post.like — guards, dry_run, click -> flip + FavoriteTweet, already");
  {
    const r0 = await makeCtx({ pathname: "/heyrohitai" }).window.__soloXAct("x.post.like", { _target_url: "https://x.com/heyrohitai" });
    check("a profile url is refused (not_a_post_url)", r0.status === "error" && /not_a_post_url/.test(r0.items[0].error), r0.items[0].error);
    const r1 = await makeCtx({ pathname: "/a/status/111" }).window.__soloXAct("x.post.like", { _target_url: "https://x.com/a/status/222" });
    check("landing on a different post refuses (post_mismatch)", r1.status === "error" && /post_mismatch/.test(r1.items[0].error), r1.items[0].error);
    const like = fakeEl({ attrs: { "data-testid": "like" } });
    const link = fakeEl({ tag: "A", attrs: { href: "/heyrohitai/status/" + P } });
    const article = fakeEl({ attrs: { "data-testid": "tweet" }, q: { ['a[href*="/status/' + P + '"]']: link, '[data-testid="like"]': like } });
    const ctx = makeCtx({ pathname: "/heyrohitai/status/" + P, nodes: { article: [article] } });
    const d = await ctx.window.__soloXAct("x.post.like", { _target_url: "https://x.com/heyrohitai/status/" + P, dry_run: true });
    check("dry_run: focal article + like control found, state not_liked, nothing clicked", d.status === "dry_run" && d.items[0].article_found === true && d.items[0].like_control_found === true && d.items[0].current_state === "not_liked", d.items[0]);
    const unlike = fakeEl({ attrs: { "data-testid": "unlike" } });
    const like2 = fakeEl({ attrs: { "data-testid": "like" }, onEvent: (e) => { if (e.type === "click") { article2._q = { ['a[href*="/status/' + P + '"]']: link, '[data-testid="unlike"]': unlike }; ctx2.window.__soloX.captures.push({ kind: "graphql", queryName: "FavoriteTweet", capturedAt: Date.now(), response: { data: { favorite_tweet: "Done" } } }); } } });
    const article2 = fakeEl({ attrs: { "data-testid": "tweet" }, q: { ['a[href*="/status/' + P + '"]']: link, '[data-testid="like"]': like2 } });
    const ctx2 = makeCtx({ pathname: "/heyrohitai/status/" + P, nodes: { article: [article2] } });
    const r = await ctx2.window.__soloXAct("x.post.like", { _target_url: "https://x.com/heyrohitai/status/" + P });
    check("click: control flips to unlike and FavoriteTweet was seen -> done verified", r.status === "done" && r.items[0].verified === true && r.items[0].mutation_seen === "FavoriteTweet" && r.items[0].state_after === "liked", r.items[0]);
    const article3 = fakeEl({ attrs: { "data-testid": "tweet" }, q: { ['a[href*="/status/' + P + '"]']: link, '[data-testid="unlike"]': unlike } });
    const a = await makeCtx({ pathname: "/heyrohitai/status/" + P, nodes: { article: [article3] } }).window.__soloXAct("x.post.like", { _target_url: "https://x.com/heyrohitai/status/" + P });
    check("already liked -> status already", a.status === "already" && a.items[0].verified === true, a.items[0]);
  }

  console.log("x.post.reply / x.post.publish — composer flow with CreateTweet proof");
  {
    const link = fakeEl({ tag: "A", attrs: { href: "/heyrohitai/status/" + P } });
    const article = fakeEl({ attrs: { "data-testid": "tweet" }, q: { ['a[href*="/status/' + P + '"]']: link } });
    let ctx;
    const box = fakeEl({ attrs: { "data-testid": "tweetTextarea_0", contenteditable: "true" } });
    const btn = fakeEl({ attrs: { "data-testid": "tweetButtonInline", "aria-disabled": "true" }, innerText: "Reply", onEvent: (e) => { if (e.type === "click") { ctx.window.__soloX.captures.push(createTweetCapture("2100000000000000001", "nguyenbinh")); box.innerText = ""; } } });
    ctx = makeCtx({ pathname: "/heyrohitai/status/" + P, nodes: { article: [article], 'tweetTextarea_0': [box], 'tweetButtonInline': [btn] } });
    const d = await ctx.window.__soloXAct("x.post.reply", { _target_url: "https://x.com/heyrohitai/status/" + P, text: "Great thread", dry_run: true });
    check("reply dry_run: composer + submit found (disabled until text), in_reply_to set, nothing typed", d.status === "dry_run" && d.items[0].composer_found === true && d.items[0].submit_button_found === true && d.items[0].submit_button_disabled === true && d.items[0].in_reply_to === P && box.innerText === "", d.items[0]);
    // the button enables once text is in
    box.focus = () => { ctx.document._active = box; btn._attrs["aria-disabled"] = "false"; };
    const r = await ctx.window.__soloXAct("x.post.reply", { _target_url: "https://x.com/heyrohitai/status/" + P, text: "Great thread" });
    check("reply: typed, Reply clicked, CreateTweet seen -> done with the reply url under the operator's handle", r.status === "done" && r.items[0].verified === true && r.items[0].post_id === "2100000000000000001" && r.items[0].post_url === "https://x.com/nguyenbinh/status/2100000000000000001" && r.items[0].post_id !== P, r.items[0]);
    const np = await makeCtx({ pathname: "/heyrohitai/status/" + P }).window.__soloXAct("x.post.publish", { text: "hello" });
    check("publish refuses a post page (its composer is a reply)", np.status === "error" && /not_a_composer_url/.test(np.items[0].error), np.items[0].error);
    const hbox = fakeEl({ attrs: { "data-testid": "tweetTextarea_0" } }); const hbtn = fakeEl({ attrs: { "data-testid": "tweetButtonInline" }, innerText: "Post" });
    const h = await makeCtx({ pathname: "/home", nodes: { 'tweetTextarea_0': [hbox], 'tweetButtonInline': [hbtn] } }).window.__soloXAct("x.post.publish", { text: "hello", dry_run: true });
    check("publish dry_run on /home reports the composer and Post button", h.status === "dry_run" && h.items[0].composer_found === true && h.items[0].submit_button_found === true && h.items[0].composer_page === "/home", h.items[0]);
    const nomut = fakeEl({ attrs: { "data-testid": "tweetTextarea_0" } }); const nobtn = fakeEl({ attrs: { "data-testid": "tweetButtonInline" }, innerText: "Post" });
    const ctxn = makeCtx({ pathname: "/home", nodes: { 'tweetTextarea_0': [nomut], 'tweetButtonInline': [nobtn] } });
    nomut.focus = () => { ctxn.document._active = nomut; };
    const n = await ctxn.window.__soloXAct("x.post.publish", { text: "hello" });
    check("publish without a CreateTweet reply -> error, never claims done", n.status === "error" && /no CreateTweet mutation/.test(n.items[0].error) && n.items[0].verified === false, n.items[0]);
  }

  console.log("x.dm.send — guards and dry_run in the conversation");
  {
    const r0 = await makeCtx({ pathname: "/a/status/1" }).window.__soloXAct("x.dm.send", { _target_url: "https://x.com/a/status/1", text: "hi" });
    check("a post url is refused (not_a_profile_url)", r0.status === "error" && /not_a_profile_url/.test(r0.items[0].error), r0.items[0].error);
    const r1 = await makeCtx({ pathname: "/someone" }).window.__soloXAct("x.dm.send", { _target_url: "https://x.com/MrPromptify", text: "hi" });
    check("landing on another profile refuses (recipient_mismatch)", r1.status === "error" && /recipient_mismatch/.test(r1.items[0].error), r1.items[0].error);
    const r2 = await makeCtx({ pathname: "/MrPromptify", nodes: {} }).window.__soloXAct("x.dm.send", { _target_url: "https://x.com/MrPromptify", text: "hi" });
    check("no Message control -> dm_not_allowed", r2.status === "error" && /dm_not_allowed/.test(r2.items[0].error), r2.items[0].error);
    const pin = await makeCtx({ pathname: "/i/chat/pin/new", nodes: {} }).window.__soloXAct("x.dm.send", { _target_url: "https://x.com/MrPromptify", username: "MrPromptify", text: "hi" });
    check("X's chat PIN setup page -> chat_pin_setup_required, nothing typed", pin.status === "error" && /chat_pin_setup_required/.test(pin.items[0].error), pin.items[0].error);
    const composer = fakeEl({ attrs: { "data-testid": "dmComposerTextInput" } });
    const send = fakeEl({ attrs: { "data-testid": "dmComposerSendButton" } });
    const header = fakeEl({ tag: "A", attrs: { href: "/MrPromptify" } });
    const ctx = makeCtx({ pathname: "/messages/123-456", nodes: { dmComposerTextInput: [composer], dmComposerSendButton: [send], 'a[href^="/"]': [header] } });
    const d = await ctx.window.__soloXAct("x.dm.send", { _target_url: "https://x.com/MrPromptify", username: "MrPromptify", text: "hello", dry_run: true });
    check("dry_run in the conversation: recipient from the header, composer + send found, nothing typed", d.status === "dry_run" && d.items[0].recipient === "MrPromptify" && d.items[0].composer_found === true && d.items[0].send_button_found === true && composer.innerText === "", d.items[0]);
    const wrong = await ctx.window.__soloXAct("x.dm.send", { _target_url: "https://x.com/recap_david", username: "recap_david", text: "hello" });
    check("conversation with somebody else -> recipient_mismatch, nothing typed", wrong.status === "error" && /recipient_mismatch/.test(wrong.items[0].error) && composer.innerText === "", wrong.items[0].error);
  }

  console.log("");
  console.log(fail === 0 ? "ALL " + pass + " CHECKS PASSED" : pass + " passed, " + fail + " FAILED");
  process.exit(fail ? 1 : 0);
})().catch((e) => { console.error(e); process.exit(2); });
