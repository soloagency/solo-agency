/*
 * ig_actions.js — Solo Agency Local Collector, Instagram module (WRITE actions).
 *
 * Injected together with ig_extract.js into the MAIN world of an instagram.com tab only for
 * ig.* write jobs and dispatched through window.__soloIgAct (core/platform_registry.js). Same
 * contract as the Facebook write module: guard BEFORE acting, report dry_run readiness, prove
 * the write two ways, never throw into the page, and always return a record — a refusal is a
 * result (status + error), never a missing item.
 *
 *   ig.post.react     like / unlike the post the page shows. Instagram's own web client calls
 *                     POST /api/v1/web/likes/<media_id>/like/ (or /unlike/) with the csrf +
 *                     app-id headers; the media id comes from the page's embedded post root
 *                     (mediaIdFromCaptures, shared with ig.post.comments). Proof: {"status":"ok"}
 *                     AND the post's own Like control reads "Unlike" afterwards.
 *   ig.post.comment   POST /api/v1/web/comments/<media_id>/add/ with comment_text (and
 *                     replied_to_comment_id for a reply). Proof: the reply carries the new
 *                     comment's id; the text is then looked for on the page.
 *   ig.message.send   drives the real UI: the "Message" button on the recipient's profile
 *                     opens the thread, the composer gets the text, Enter sends. Guards: the
 *                     thread header must name the recipient asked for; exactly one composer.
 *                     Proof: the text appears in the thread. A DM to a non-follower lands in
 *                     the recipient's Requests folder — reported, not hidden.
 *   ig.post.publish   NOT built: Instagram has no text-only post; the create flow needs an
 *                     image or video upload (see INSTAGRAM_CAPABILITIES.md).
 */
(function () {
  "use strict";
  if (typeof window.__soloIgAct === "function") return;

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
  function composerText(box) { return norm(box.innerText || box.value || box.textContent || ""); }
  async function typeInto(box, text) {
    box.focus(); click(box); box.focus();
    await sleep(150);
    if (box.tagName === "TEXTAREA" || box.tagName === "INPUT") {
      try {
        var proto = box.tagName === "TEXTAREA" ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
        var setter = Object.getOwnPropertyDescriptor(proto, "value").set;
        setter.call(box, text);
        box.dispatchEvent(new Event("input", { bubbles: true }));
      } catch (e) { box.value = text; }
    } else {
      try { document.execCommand("selectAll", false, null); } catch (e) { /* ignore */ }
      var ok = false;
      try { ok = document.execCommand("insertText", false, text); } catch (e) { ok = false; }
      if (!composerText(box)) {
        try {
          box.dispatchEvent(new InputEvent("beforeinput", { bubbles: true, cancelable: true, inputType: "insertText", data: text }));
          box.dispatchEvent(new InputEvent("input", { bubbles: true, cancelable: true, inputType: "insertText", data: text }));
        } catch (e) { /* ignore */ }
      }
    }
    await sleep(200);
  }
  function wrapCap(capId, status, extra) {
    var rec = Object.assign({ capability: capId, status: status, verified: false, error: null, ts: nowISO() }, extra || {});
    return { available: true, capability: capId, status: status, count: 1, items: [rec], version: VERSION, _debug: { href: location.href } };
  }

  // ---- shared with ig_extract.js (injected first) ------------------------
  function store() { return window.__soloIg || { captures: [] }; }
  function internals() { return window.__soloIgInternals || {}; }
  function shortcodeFromHref() { try { var m = location.pathname.match(/\/(?:p|reel|reels)\/([A-Za-z0-9_-]+)/); return m ? m[1] : ""; } catch (e) { return ""; } }
  function shortcodeFrom(u) { try { var m = String(u || "").match(/\/(?:p|reel|reels)\/([A-Za-z0-9_-]+)/); return m ? m[1] : ""; } catch (e) { return ""; } }
  function usernameFrom(u) {
    try {
      var m = new URL(String(u || ""), location.origin).pathname.match(/^\/([A-Za-z0-9._]+)\/?/);
      if (!m || /^(explore|p|reel|reels|stories|accounts|direct|api|graphql)$/i.test(m[1])) return "";
      return m[1];
    } catch (e) { return ""; }
  }
  function headers() {
    var s = store();
    return {
      "x-csrftoken": typeof s.csrfToken === "function" ? s.csrfToken() : "",
      "x-ig-app-id": typeof s.appId === "function" ? s.appId() : "936619743392459",
      "x-requested-with": "XMLHttpRequest",
      "x-instagram-ajax": "1",
      "x-asbd-id": "129477",
      "content-type": "application/x-www-form-urlencoded"
    };
  }
  function restPost(path, form) {
    var s = store();
    var f = typeof s.origFetch === "function" ? s.origFetch : window.fetch;
    var body = "";
    if (form) { var p = new URLSearchParams(); Object.keys(form).forEach(function (k) { p.set(k, String(form[k])); }); body = p.toString(); }
    return f(path, { method: "POST", credentials: "include", headers: headers(), body: body }).then(function (r) {
      return r.text().then(function (t) { var j = null; try { j = JSON.parse(String(t).replace(/^for\s*\(;;\);/, "")); } catch (e) { j = null; } return { status: r.status, json: j }; });
    });
  }
  function resolveMediaId(inputs, shortcode) {
    var id = str(inputs.media_id);
    if (id) return Promise.resolve(id);
    var find = internals().mediaIdFromCaptures;
    if (typeof find !== "function" || !shortcode) return Promise.resolve("");
    return waitFor(function () { return find(shortcode) || null; }, 6000, 500).then(function (v) { return v || ""; });
  }
  // The post's own Like control: an svg with aria-label Like / Unlike inside the article.
  function likeControl() {
    var svgs = document.querySelectorAll('article svg[aria-label], section svg[aria-label], main svg[aria-label]');
    for (var i = 0; i < svgs.length; i++) {
      var lbl = lower(svgs[i].getAttribute("aria-label"));
      if (lbl === "like" || lbl === "unlike" || lbl === "thích" || lbl === "bỏ thích") {
        var btn = svgs[i].closest('[role="button"], button');
        if (btn && visible(btn)) return { el: btn, liked: lbl === "unlike" || lbl === "bỏ thích", label: lbl };
      }
    }
    return null;
  }
  function commentBox() {
    var els = document.querySelectorAll('textarea[aria-label], form textarea, div[contenteditable="true"][role="textbox"]');
    for (var i = 0; i < els.length; i++) {
      var lbl = lower(els[i].getAttribute("aria-label") || els[i].getAttribute("placeholder") || "");
      if (visible(els[i]) && (/comment|bình luận/.test(lbl) || els[i].tagName === "TEXTAREA")) return els[i];
    }
    return null;
  }

  // ------------------------------------------------------------- ig.post.react
  async function doReact(inputs) {
    var want = lower(inputs.action || inputs.reaction || "like") === "unlike" ? "unlike" : "like";
    var jobUrl = str(inputs._target_url || inputs.post_url || location.href);
    var wantCode = shortcodeFrom(jobUrl), hereCode = shortcodeFromHref();
    if (!hereCode) return wrapCap("ig.post.react", "error", { job_url: jobUrl, landed_url: location.href, error: "not_a_post_url: open https://www.instagram.com/p/<code>/ or /reel/<code>/" });
    if (wantCode && wantCode !== hereCode) return wrapCap("ig.post.react", "error", { job_url: jobUrl, landed_url: location.href, error: "post_mismatch: asked for " + wantCode + " but landed on " + hereCode + " — nothing was done" });
    var mediaId = await resolveMediaId(inputs, hereCode);
    if (!mediaId) return wrapCap("ig.post.react", "error", { shortcode: hereCode, error: "media_id_unknown: the post root did not embed a media matching " + hereCode });
    var ctl = await waitFor(likeControl, 6000, 300);
    var state = ctl ? (ctl.liked ? "liked" : "not_liked") : "unknown";
    if (inputs.dry_run) return wrapCap("ig.post.react", "dry_run", { action: want, shortcode: hereCode, media_id: mediaId, like_control_found: !!ctl, current_state: state });
    if ((want === "like" && state === "liked") || (want === "unlike" && state === "not_liked")) {
      return wrapCap("ig.post.react", "already", { action: want, shortcode: hereCode, media_id: mediaId, verified: true, current_state: state, like_control_found: !!ctl });
    }
    await jitter();
    var res = await restPost("/api/v1/web/likes/" + mediaId + "/" + want + "/");
    var ok = !!(res.json && res.json.status === "ok");
    var after = await waitFor(function () { var c = likeControl(); return c && ((want === "like") === c.liked) ? c : null; }, 5000, 300);
    var status = ok ? "done" : "error";
    return wrapCap("ig.post.react", status, {
      action: want, shortcode: hereCode, media_id: mediaId, verified: ok && !!after, http_status: res.status,
      current_state: state, like_control_found: !!ctl, state_before: state, state_after: after ? (after.liked ? "liked" : "not_liked") : state,
      error: ok ? null : ("instagram answered HTTP " + res.status + (res.json && res.json.message ? ": " + res.json.message : ""))
    });
  }

  // ------------------------------------------------------------- ig.post.comment
  async function doComment(inputs) {
    var text = str(inputs.text || inputs.message).trim();
    if (!text) return wrapCap("ig.post.comment", "error", { error: "no comment text provided" });
    var jobUrl = str(inputs._target_url || inputs.post_url || location.href);
    var wantCode = shortcodeFrom(jobUrl), hereCode = shortcodeFromHref();
    if (!hereCode) return wrapCap("ig.post.comment", "error", { text: text, job_url: jobUrl, landed_url: location.href, error: "not_a_post_url: open https://www.instagram.com/p/<code>/ or /reel/<code>/" });
    if (wantCode && wantCode !== hereCode) return wrapCap("ig.post.comment", "error", { text: text, job_url: jobUrl, landed_url: location.href, error: "post_mismatch: asked for " + wantCode + " but landed on " + hereCode + " — nothing was typed" });
    var mediaId = await resolveMediaId(inputs, hereCode);
    if (!mediaId) return wrapCap("ig.post.comment", "error", { text: text, shortcode: hereCode, error: "media_id_unknown: the post root did not embed a media matching " + hereCode });
    var box = await waitFor(commentBox, 5000, 300);
    if (inputs.dry_run) return wrapCap("ig.post.comment", "dry_run", { text: text, shortcode: hereCode, media_id: mediaId, comment_box_found: !!box, reply_to: str(inputs.reply_to_comment_id) || null });
    if (!box) return wrapCap("ig.post.comment", "error", { text: text, shortcode: hereCode, media_id: mediaId, error: "no comment box on the page — comments may be turned off for this post; nothing was sent" });
    await jitter();
    var form = { comment_text: text };
    if (inputs.reply_to_comment_id) form.replied_to_comment_id = str(inputs.reply_to_comment_id);
    var res = await restPost("/api/v1/web/comments/" + mediaId + "/add/", form);
    var ok = !!(res.json && res.json.status === "ok" && (res.json.id || (res.json.comment && res.json.comment.pk)));
    var commentId = ok ? str(res.json.id || (res.json.comment && res.json.comment.pk)) : "";
    var probe = text.slice(0, 40);
    var appeared = ok ? await waitFor(function () { try { return (document.body.innerText || "").indexOf(probe) > -1; } catch (e) { return false; } }, 6000, 500) : null;
    return wrapCap("ig.post.comment", ok ? "done" : "error", {
      text: text, shortcode: hereCode, media_id: mediaId, comment_id: commentId || null, verified: ok, appeared: !!appeared, http_status: res.status, comment_box_found: true,
      comment_url: commentId ? "https://www.instagram.com/p/" + hereCode + "/c/" + commentId + "/" : null,
      error: ok ? null : ("instagram answered HTTP " + res.status + (res.json && (res.json.message || res.json.feedback_message) ? ": " + (res.json.message || res.json.feedback_message) : ""))
    });
  }

  // ------------------------------------------------------------- ig.message.send
  var MESSAGE_BUTTON = /^(message|nhắn tin)$/i;
  function findMessageButton() {
    var els = document.querySelectorAll('[role="button"], button, a[role="link"]');
    for (var i = 0; i < els.length; i++) { if (visible(els[i]) && MESSAGE_BUTTON.test(norm(els[i].innerText || els[i].getAttribute("aria-label") || ""))) return els[i]; }
    return null;
  }
  function dmComposers() {
    var out = [], els = document.querySelectorAll('div[contenteditable="true"][role="textbox"], textarea[placeholder]');
    for (var i = 0; i < els.length; i++) {
      var lbl = lower(els[i].getAttribute("aria-label") || els[i].getAttribute("placeholder") || els[i].getAttribute("aria-placeholder") || "");
      if (visible(els[i]) && /message|tin nhắn|nhắn/.test(lbl)) out.push(els[i]);
    }
    return out;
  }
  // The thread header links to the recipient's profile: the identity check.
  function threadRecipient() {
    var links = document.querySelectorAll('header a[href^="/"], [role="main"] a[href^="/"]');
    for (var i = 0; i < links.length; i++) {
      var href = str(links[i].getAttribute("href"));
      var m = href.match(/^\/([A-Za-z0-9._]+)\/?$/);
      if (m && !/^(direct|explore|reels|accounts|p|stories)$/i.test(m[1]) && visible(links[i])) return m[1];
    }
    return "";
  }
  async function doMessage(inputs) {
    var text = str(inputs.text || inputs.message).trim();
    if (!text) return wrapCap("ig.message.send", "error", { error: "no message text provided" });
    var jobUrl = str(inputs._target_url || inputs.profile_url || location.href);
    var wantUser = str(inputs.username).replace(/^@/, "") || usernameFrom(jobUrl);
    var onThread = /\/direct\/t\//.test(location.pathname);
    var opened = false;
    if (!onThread) {
      var hereUser = usernameFrom(location.href);
      if (!hereUser) return wrapCap("ig.message.send", "error", { text: text, landed_url: location.href, error: "not_a_profile_url: open https://www.instagram.com/<username>/ (the Message button opens the thread)" });
      if (wantUser && lower(wantUser) !== lower(hereUser)) return wrapCap("ig.message.send", "error", { text: text, requested: wantUser, landed: hereUser, error: "recipient_mismatch: asked for " + wantUser + " but landed on " + hereUser + " — nothing was typed" });
      var btn = await waitFor(findMessageButton, 6000, 300);
      if (!btn) return wrapCap("ig.message.send", "error", { text: text, recipient: hereUser, error: "no Message button on this profile — the account does not accept messages from this session, or the page did not render" });
      click(btn); opened = true;
      await waitFor(function () { return /\/direct\/t\//.test(location.pathname) || dmComposers().length > 0; }, 10000, 300);
    }
    var boxes = dmComposers();
    if (boxes.length !== 1) return wrapCap("ig.message.send", "error", { text: text, recipient: wantUser || null, composers_found: boxes.length, opened_thread: opened, error: boxes.length === 0 ? "the thread composer did not open" : "ambiguous_composer: " + boxes.length + " composers on the page — nothing was typed" });
    var recipient = threadRecipient();
    if (wantUser && recipient && lower(recipient) !== lower(wantUser)) return wrapCap("ig.message.send", "error", { text: text, requested: wantUser, thread_recipient: recipient, error: "recipient_mismatch: the thread belongs to " + recipient + " — nothing was typed" });
    if (inputs.dry_run) return wrapCap("ig.message.send", "dry_run", { text: text, recipient: recipient || wantUser || null, thread_url: location.href, opened_thread: opened, composer_found: true });
    await jitter();
    await typeInto(boxes[0], text);
    if (!composerText(boxes[0])) return wrapCap("ig.message.send", "error", { text: text, recipient: recipient || wantUser, error: "failed to enter text into the composer" });
    await jitter();
    var sent = false;
    var sendBtn = null, cands = document.querySelectorAll('[role="button"], button');
    for (var i = 0; i < cands.length; i++) { if (visible(cands[i]) && /^(send|gửi)$/i.test(norm(cands[i].innerText || cands[i].getAttribute("aria-label") || ""))) { sendBtn = cands[i]; break; } }
    if (sendBtn) { click(sendBtn); sent = true; }
    else { try { boxes[0].dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", code: "Enter", keyCode: 13, which: 13, bubbles: true, cancelable: true })); sent = true; } catch (e) { sent = false; } }
    var probe = text.slice(0, 40);
    var cleared = await waitFor(function () { return composerText(boxes[0]) === "" ? true : null; }, 6000, 300);
    var appeared = await waitFor(function () { try { return (document.body.innerText || "").indexOf(probe) > -1; } catch (e) { return false; } }, 8000, 500);
    var status = sent && cleared && appeared ? "done" : "error";
    return wrapCap("ig.message.send", status, {
      text: text, recipient: recipient || wantUser || null, thread_url: location.href, verified: status === "done", composer_found: true, sent_via: sendBtn ? "button" : "enter",
      composer_cleared: !!cleared, appeared: !!appeared, opened_thread: opened,
      error: status === "error" ? (sent ? "the composer did not clear or the text did not appear in the thread — the message may not have been sent" : "no way to submit the message") : null
    });
  }

  window.__soloIgAct = async function (capId, inputs) {
    inputs = inputs && typeof inputs === "object" ? inputs : {};
    try {
      if (capId === "ig.post.react") return await doReact(inputs);
      if (capId === "ig.post.comment") return await doComment(inputs);
      if (capId === "ig.message.send") return await doMessage(inputs);
      return { available: false, capability: capId, count: 0, items: [{ status: "error", error: "unknown or unimplemented action: " + capId }], _debug: { href: location.href } };
    } catch (e) {
      return { available: false, capability: capId, count: 0, items: [{ status: "error", error: String(e && e.message || e) }], _debug: { href: location.href, error: String(e) } };
    }
  };
  window.__soloIgActInternals = { likeControl: likeControl, commentBox: commentBox, threadRecipient: threadRecipient, usernameFrom: usernameFrom, shortcodeFrom: shortcodeFrom };
})();
