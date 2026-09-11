/*
 * x_actions.js — Solo Agency Local Collector, X (Twitter) module (WRITE actions).
 *
 * Injected together with x_extract.js into the MAIN world of an x.com tab only for x.* write
 * jobs and dispatched through window.__soloXAct (core/platform_registry.js). Every write drives
 * X's REAL UI: X stamps each request with a per-request x-client-transaction-id, so a mutation
 * sent by the collector itself would be refused (404, measured on reads) — clicking the page's
 * own controls lets the page sign its own requests. The interceptor then sees the mutation
 * (FavoriteTweet, CreateTweet, the DM call) and that reply is the proof.
 *
 *   x.post.like      the focal post's Like control ([data-testid=like] → unlike). Proof: the
 *                    control flips AND/OR a FavoriteTweet mutation was captured.
 *   x.post.reply     the inline reply composer on the post page ([data-testid=tweetTextarea_0])
 *                    + the Reply button ([data-testid=tweetButtonInline]). Proof: a CreateTweet
 *                    mutation whose reply names the new post id → reply url.
 *   x.post.publish   the home / compose composer + the Post button. Proof: CreateTweet → the
 *                    new post's id and url under the operator's own handle.
 *   x.dm.send        the profile's Message control ([data-testid=sendDMFromProfile]) opens the
 *                    conversation; [data-testid=dmComposerTextInput] gets the text,
 *                    [data-testid=dmComposerSendButton] sends. Guards: the conversation must be
 *                    with the handle asked for. Proof: the text appears as a sent entry.
 */
(function () {
  "use strict";
  if (typeof window.__soloXAct === "function") return;

  var VERSION = "0.1.0";
  function sleep(ms) { return new Promise(function (r) { setTimeout(r, ms); }); }
  function rnd(a, b) { return a + Math.floor(Math.random() * (b - a + 1)); }
  function jitter() { return sleep(rnd(350, 950)); }
  function norm(s) { return String(s == null ? "" : s).replace(/\s+/g, " ").trim(); }
  function lower(s) { return norm(s).toLowerCase(); }
  function str(v) { return typeof v === "string" ? v : (v === null || v === undefined ? "" : String(v)); }
  function isObj(v) { return !!v && typeof v === "object" && !Array.isArray(v); }
  function nowISO() { return new Date().toISOString(); }
  async function waitFor(fn, timeoutMs, stepMs) {
    stepMs = stepMs || 300; timeoutMs = timeoutMs || 10000;
    var t = 0;
    while (t < timeoutMs) { var v = fn(); if (v) return v; await sleep(stepMs); t += stepMs; }
    return null;
  }
  function ev(el, type, Ctor) {
    try { el.dispatchEvent(new Ctor(type, { bubbles: true, cancelable: true, view: window })); }
    catch (e) { try { el.dispatchEvent(new MouseEvent(type.replace("pointer", "mouse"), { bubbles: true, cancelable: true, view: window })); } catch (e2) { /* ignore */ } }
  }
  function click(el) {
    ev(el, "pointerover", window.PointerEvent || MouseEvent); ev(el, "mouseover", MouseEvent); ev(el, "mousemove", MouseEvent);
    ev(el, "pointerdown", window.PointerEvent || MouseEvent); ev(el, "mousedown", MouseEvent);
    ev(el, "pointerup", window.PointerEvent || MouseEvent); ev(el, "mouseup", MouseEvent); ev(el, "click", MouseEvent);
  }
  function visible(el) { try { var r = el.getBoundingClientRect(); return r.width > 0 && r.height > 0; } catch (e) { return false; } }
  function isDisabled(el) { return !!el && (String(el.getAttribute("aria-disabled")) === "true" || el.disabled === true); }
  function composerText(box) { return norm(box.innerText || box.value || box.textContent || ""); }
  async function typeInto(box, text) {
    box.focus(); click(box); box.focus();
    await sleep(150);
    try { document.execCommand("selectAll", false, null); } catch (e) { /* ignore */ }
    var ok = false;
    try { ok = document.execCommand("insertText", false, text); } catch (e) { ok = false; }
    if (!composerText(box)) {
      try {
        box.dispatchEvent(new InputEvent("beforeinput", { bubbles: true, cancelable: true, inputType: "insertText", data: text }));
        box.dispatchEvent(new InputEvent("input", { bubbles: true, cancelable: true, inputType: "insertText", data: text }));
      } catch (e) { /* ignore */ }
    }
    await sleep(250);
  }
  function wrapCap(capId, status, extra) {
    var rec = Object.assign({ capability: capId, status: status, verified: false, error: null, ts: nowISO() }, extra || {});
    return { available: true, capability: capId, status: status, count: 1, items: [rec], version: VERSION, _debug: { href: location.href } };
  }
  function store() { return window.__soloX || { captures: [] }; }
  function captures() { var s = store(); return Array.isArray(s.captures) ? s.captures : []; }
  function statusIdFrom(u) { try { var m = String(u || "").match(/\/status\/(\d+)/); return m ? m[1] : ""; } catch (e) { return ""; } }
  function handleFrom(u) {
    try {
      var m = new URL(String(u || ""), location.origin).pathname.match(/^\/([A-Za-z0-9_]{1,15})(?:\/|$)/);
      if (!m || /^(home|explore|search|notifications|messages|i|settings|compose|login|logout|hashtag|intent|share)$/i.test(m[1])) return "";
      return m[1];
    } catch (e) { return ""; }
  }
  // A mutation the page fired after t0, by operation name.
  function mutationSince(t0, re) {
    var caps = captures();
    for (var i = caps.length - 1; i >= 0; i--) { var c = caps[i]; if (c && (c.capturedAt || 0) >= t0 - 1500 && re.test(str(c.queryName))) return c; }
    return null;
  }
  function createdTweet(cap) {
    try {
      var r = cap && cap.response && cap.response.data && cap.response.data.create_tweet && cap.response.data.create_tweet.tweet_results && cap.response.data.create_tweet.tweet_results.result;
      if (!r) return null;
      var t = r.__typename === "TweetWithVisibilityResults" && r.tweet ? r.tweet : r;
      var id = str(t.rest_id || (t.legacy && t.legacy.id_str));
      var u = t.core && t.core.user_results && t.core.user_results.result;
      var handle = str(u && ((u.core && u.core.screen_name) || (u.legacy && u.legacy.screen_name)));
      return id ? { id: id, handle: handle, url: "https://x.com/" + (handle || "i/web") + "/status/" + id } : null;
    } catch (e) { return null; }
  }
  // The article that IS the post the page opened (the focal one), not a reply below it.
  function focalArticle(id) {
    var arts = document.querySelectorAll('article[data-testid="tweet"], article');
    for (var i = 0; i < arts.length; i++) {
      if (!visible(arts[i])) continue;
      if (!id) return arts[i];
      if (arts[i].querySelector('a[href*="/status/' + id + '"]')) return arts[i];
    }
    return null;
  }
  function likeControl(article) {
    var root = article || document;
    var un = root.querySelector('[data-testid="unlike"]'), li = root.querySelector('[data-testid="like"]');
    if (un && visible(un)) return { el: un, liked: true };
    if (li && visible(li)) return { el: li, liked: false };
    return null;
  }
  function composerBox() {
    var b = document.querySelector('[data-testid="tweetTextarea_0"]');
    return b && visible(b) ? b : null;
  }
  function submitButton() {
    var cands = ['[data-testid="tweetButtonInline"]', '[data-testid="tweetButton"]'];
    for (var i = 0; i < cands.length; i++) { var b = document.querySelector(cands[i]); if (b && visible(b)) return b; }
    return null;
  }

  // ------------------------------------------------------------- x.post.like
  async function doLike(inputs) {
    var want = lower(inputs.action || "like") === "unlike" ? "unlike" : "like";
    var jobUrl = str(inputs._target_url || inputs.post_url || location.href);
    var wantId = statusIdFrom(jobUrl), hereId = statusIdFrom(location.href);
    if (!hereId) return wrapCap("x.post.like", "error", { job_url: jobUrl, landed_url: location.href, error: "not_a_post_url: open https://x.com/<handle>/status/<id>" });
    if (wantId && wantId !== hereId) return wrapCap("x.post.like", "error", { job_url: jobUrl, landed_url: location.href, error: "post_mismatch: asked for " + wantId + " but landed on " + hereId + " — nothing was done" });
    var art = await waitFor(function () { return focalArticle(hereId); }, 8000, 300);
    var ctl = art ? likeControl(art) : null;
    var state = ctl ? (ctl.liked ? "liked" : "not_liked") : "unknown";
    if (inputs.dry_run) return wrapCap("x.post.like", "dry_run", { action: want, post_id: hereId, article_found: !!art, like_control_found: !!ctl, current_state: state });
    if (!ctl) return wrapCap("x.post.like", "error", { action: want, post_id: hereId, error: "no Like control on the focal post (the post may be unavailable or the page did not render)" });
    if ((want === "like") === ctl.liked) return wrapCap("x.post.like", "already", { action: want, post_id: hereId, verified: true, current_state: state });
    var t0 = Date.now();
    await jitter();
    click(ctl.el);
    var flipped = await waitFor(function () { var c = likeControl(focalArticle(hereId)); return c && ((want === "like") === c.liked) ? c : null; }, 6000, 300);
    var mut = mutationSince(t0, want === "like" ? /^FavoriteTweet$/ : /^UnfavoriteTweet$/);
    var status = (flipped || mut) ? "done" : "error";
    return wrapCap("x.post.like", status, { action: want, post_id: hereId, verified: !!(flipped && mut) || !!flipped, control_flipped: !!flipped, mutation_seen: mut ? mut.queryName : null, state_before: state, state_after: flipped ? (want === "like" ? "liked" : "not_liked") : state, error: status === "error" ? "the Like control did not flip and no FavoriteTweet mutation was seen" : null });
  }

  // ------------------------------------------------------------- x.post.reply / x.post.publish
  async function compose(capId, text, expect, inputs) {
    var box = await waitFor(composerBox, 8000, 300);
    var btn = box ? submitButton() : null;
    if (inputs.dry_run) return wrapCap(capId, "dry_run", Object.assign({ text: text, composer_found: !!box, submit_button_found: !!btn, submit_button_disabled: btn ? isDisabled(btn) : null }, expect.dry || {}));
    if (!box) return wrapCap(capId, "error", Object.assign({ text: text, error: "no composer on this page — nothing was typed" }, expect.dry || {}));
    await jitter();
    await typeInto(box, text);
    if (!composerText(box)) return wrapCap(capId, "error", Object.assign({ text: text, error: "failed to enter text into the composer" }, expect.dry || {}));
    btn = await waitFor(function () { var b = submitButton(); return b && !isDisabled(b) ? b : null; }, 5000, 250);
    if (!btn) return wrapCap(capId, "error", Object.assign({ text: text, typed: true, error: "the submit button never enabled — text was typed but NOT submitted" }, expect.dry || {}));
    var t0 = Date.now();
    await jitter();
    click(btn);
    var mut = await waitFor(function () { var c = mutationSince(t0, /^CreateTweet$/); return c && c.response ? c : null; }, 12000, 400);
    var created = createdTweet(mut);
    var cleared = await waitFor(function () { var b = composerBox(); return !b || composerText(b) === "" ? true : null; }, 6000, 300);
    var status = created ? "done" : (mut ? "error" : (cleared ? "error" : "error"));
    return wrapCap(capId, status, Object.assign({}, expect.dry || {}, { text: text, verified: !!created, composer_cleared: !!cleared, mutation_seen: mut ? mut.queryName : null, post_id: created ? created.id : null, post_url: created ? created.url : null, handle: created ? created.handle : null,
      error: created ? null : (mut ? "CreateTweet answered without a post (X may have rejected the text: duplicate, limit, or flagged)" : "no CreateTweet mutation was seen after the click — the post was not submitted") }));
  }
  async function doReply(inputs) {
    var text = str(inputs.text || inputs.message).trim();
    if (!text) return wrapCap("x.post.reply", "error", { error: "no reply text provided" });
    var jobUrl = str(inputs._target_url || inputs.post_url || location.href);
    var wantId = statusIdFrom(jobUrl), hereId = statusIdFrom(location.href);
    if (!hereId) return wrapCap("x.post.reply", "error", { text: text, job_url: jobUrl, landed_url: location.href, error: "not_a_post_url: open https://x.com/<handle>/status/<id>" });
    if (wantId && wantId !== hereId) return wrapCap("x.post.reply", "error", { text: text, job_url: jobUrl, landed_url: location.href, error: "post_mismatch: asked for " + wantId + " but landed on " + hereId + " — nothing was typed" });
    var art = await waitFor(function () { return focalArticle(hereId); }, 8000, 300);
    if (!art) return wrapCap("x.post.reply", "error", { text: text, post_id: hereId, error: "the focal post did not render — nothing was typed" });
    return compose("x.post.reply", text, { dry: { in_reply_to: hereId } }, inputs);
  }
  async function doPublish(inputs) {
    var text = str(inputs.text || inputs.message).trim();
    if (!text) return wrapCap("x.post.publish", "error", { error: "no post text provided" });
    var path = location.pathname.replace(/\/+$/, "");
    if (!/^(\/home|\/compose\/post|)$/.test(path)) return wrapCap("x.post.publish", "error", { text: text, landed_url: location.href, error: "not_a_composer_url: open https://x.com/home or https://x.com/compose/post — a post page's composer is a reply" });
    return compose("x.post.publish", text, { dry: { composer_page: path || "/" } }, inputs);
  }

  // ------------------------------------------------------------- x.dm.send
  function dmComposer() { var b = document.querySelector('[data-testid="dmComposerTextInput"]'); return b && visible(b) ? b : null; }
  function dmSendButton() { var b = document.querySelector('[data-testid="dmComposerSendButton"]'); return b && visible(b) ? b : null; }
  function conversationHandle() {
    var links = document.querySelectorAll('[data-testid="DMConversationHeader"] a[href^="/"], [data-testid="conversation"] a[href^="/"], main a[href^="/"]');
    for (var i = 0; i < links.length; i++) {
      var h = handleFrom("https://x.com" + str(links[i].getAttribute("href")));
      if (h && visible(links[i]) && !/messages/.test(str(links[i].getAttribute("href")))) return h;
    }
    return "";
  }
  async function doDm(inputs) {
    var text = str(inputs.text || inputs.message).trim();
    if (!text) return wrapCap("x.dm.send", "error", { error: "no message text provided" });
    var jobUrl = str(inputs._target_url || inputs.profile_url || location.href);
    var wantHandle = str(inputs.username || inputs.handle).replace(/^@/, "") || handleFrom(jobUrl);
    var onMessages = /^\/messages\//.test(location.pathname);
    var opened = false;
    if (!onMessages) {
      var hereHandle = statusIdFrom(location.href) ? "" : handleFrom(location.href);
      if (!hereHandle) return wrapCap("x.dm.send", "error", { text: text, landed_url: location.href, error: "not_a_profile_url: open https://x.com/<handle> (its Message control opens the conversation)" });
      if (wantHandle && lower(wantHandle) !== lower(hereHandle)) return wrapCap("x.dm.send", "error", { text: text, requested: wantHandle, landed: hereHandle, error: "recipient_mismatch: asked for " + wantHandle + " but landed on " + hereHandle + " — nothing was typed" });
      var btn = await waitFor(function () { var b = document.querySelector('[data-testid="sendDMFromProfile"]'); return b && visible(b) ? b : null; }, 8000, 300);
      if (!btn) return wrapCap("x.dm.send", "error", { text: text, recipient: hereHandle, error: "dm_not_allowed: this profile shows no Message control — the account does not accept DMs from this session" });
      click(btn); opened = true;
      await waitFor(function () { return /^\/messages\//.test(location.pathname) && dmComposer(); }, 10000, 300);
    }
    var box = dmComposer();
    if (!box) return wrapCap("x.dm.send", "error", { text: text, recipient: wantHandle || null, opened_conversation: opened, error: "the conversation composer did not open" });
    var who = conversationHandle();
    if (wantHandle && who && lower(who) !== lower(wantHandle)) return wrapCap("x.dm.send", "error", { text: text, requested: wantHandle, conversation_with: who, error: "recipient_mismatch: the conversation is with " + who + " — nothing was typed" });
    if (inputs.dry_run) return wrapCap("x.dm.send", "dry_run", { text: text, recipient: who || wantHandle || null, conversation_url: location.href, opened_conversation: opened, composer_found: true, send_button_found: !!dmSendButton() });
    await jitter();
    await typeInto(box, text);
    if (!composerText(box)) return wrapCap("x.dm.send", "error", { text: text, recipient: who || wantHandle, error: "failed to enter text into the composer" });
    var send = await waitFor(function () { var b = dmSendButton(); return b && !isDisabled(b) ? b : null; }, 5000, 250);
    if (!send) return wrapCap("x.dm.send", "error", { text: text, recipient: who || wantHandle, typed: true, error: "the Send control never enabled — text was typed but NOT sent" });
    var t0 = Date.now();
    await jitter();
    click(send);
    var probe = text.slice(0, 40);
    var cleared = await waitFor(function () { var b = dmComposer(); return !b || composerText(b) === "" ? true : null; }, 6000, 300);
    var appeared = await waitFor(function () { try { var entries = document.querySelectorAll('[data-testid="messageEntry"], [data-testid="tweetText"]'); for (var i = entries.length - 1; i >= 0; i--) { if ((entries[i].innerText || "").indexOf(probe) > -1) return true; } return (document.body.innerText || "").indexOf(probe) > -1; } catch (e) { return false; } }, 8000, 500);
    var call = mutationSince(t0, /dm\/new|useSendMessageMutation|DMSend/i);
    var status = (cleared && appeared) ? "done" : "error";
    return wrapCap("x.dm.send", status, { text: text, recipient: who || wantHandle || null, conversation_url: location.href, verified: status === "done", composer_cleared: !!cleared, appeared: !!appeared, call_seen: call ? call.queryName : null, opened_conversation: opened,
      error: status === "error" ? "the composer did not clear or the text did not appear in the conversation — the message may not have been sent" : null });
  }

  window.__soloXAct = async function (capId, inputs) {
    inputs = inputs && typeof inputs === "object" ? inputs : {};
    try {
      if (capId === "x.post.like") return await doLike(inputs);
      if (capId === "x.post.reply") return await doReply(inputs);
      if (capId === "x.post.publish") return await doPublish(inputs);
      if (capId === "x.dm.send") return await doDm(inputs);
      return { available: false, capability: capId, count: 0, items: [{ status: "error", error: "unknown or unimplemented action: " + capId }], _debug: { href: location.href } };
    } catch (e) {
      return { available: false, capability: capId, count: 0, items: [{ status: "error", error: String(e && e.message || e) }], _debug: { href: location.href, error: String(e) } };
    }
  };
  window.__soloXActInternals = { createdTweet: createdTweet, focalArticle: focalArticle, likeControl: likeControl, handleFrom: handleFrom, statusIdFrom: statusIdFrom };
})();
