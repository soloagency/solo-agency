// gql_actions.js — WRITE actions (react / comment / DM) driven through the real
// Facebook UI via DOM, MAIN world. Injected by background.js only when a job
// carries a write capability; never runs in the daily read pipeline.
//
// Division of concerns (per operator directive): this file is the MECHANISM —
// make the action land correctly and report a verifiable result. The APPROVAL
// layer lives upstream (the daily report the operator reviews + approves); a
// job that reaches here is meant to execute. We still keep idempotency, a
// verify step, and a target preview because those are correctness, not gating.
//
// P1 implements fb.post.react. P2 (comment) and P3 (message.send) land later.
(function () {
  if (window.__soloActRun) return;

  // ---- tiny utils ---------------------------------------------------------
  function sleep(ms) { return new Promise(function (r) { setTimeout(r, ms); }); }
  function rnd(a, b) { return a + Math.floor(Math.random() * (b - a + 1)); }
  function jitter() { return sleep(rnd(350, 950)); } // human-ish micro delay
  function norm(s) { return String(s == null ? "" : s).replace(/\s+/g, " ").trim(); }
  function lower(s) { return norm(s).toLowerCase(); }
  function nowISO() { return new Date().toISOString(); }

  async function waitFor(fn, timeoutMs, stepMs) {
    stepMs = stepMs || 300; timeoutMs = timeoutMs || 10000;
    var t = 0;
    while (t < timeoutMs) { var v = fn(); if (v) return v; await sleep(stepMs); t += stepMs; }
    return null;
  }

  // Real pointer/mouse event sequences — FB's anti-automation keys off events
  // that a genuine cursor produces, so we emit the full pointer+mouse chain
  // rather than a bare element.click().
  function ev(el, type, Ctor) {
    try { el.dispatchEvent(new Ctor(type, { bubbles: true, cancelable: true, view: window })); }
    catch (e) { try { el.dispatchEvent(new MouseEvent(type.replace("pointer", "mouse"), { bubbles: true, cancelable: true, view: window })); } catch (e2) { /* ignore */ } }
  }
  function hover(el) {
    ev(el, "pointerover", window.PointerEvent || MouseEvent);
    ev(el, "mouseover", MouseEvent);
    ev(el, "pointerenter", window.PointerEvent || MouseEvent);
    ev(el, "mouseenter", MouseEvent);
    ev(el, "mousemove", MouseEvent);
  }
  function click(el) {
    hover(el);
    ev(el, "pointerdown", window.PointerEvent || MouseEvent);
    ev(el, "mousedown", MouseEvent);
    ev(el, "pointerup", window.PointerEvent || MouseEvent);
    ev(el, "mouseup", MouseEvent);
    ev(el, "click", MouseEvent);
  }

  // ---- locale-aware reaction labels (EN + VI; extend as clients need) ------
  var REACT = {
    like: ["like", "thích"],
    love: ["love", "yêu thích"],
    care: ["care", "thương thương"],
    haha: ["haha"],
    wow: ["wow"],
    sad: ["sad", "buồn"],
    angry: ["angry", "phẫn nộ"]
  };
  var REMOVE_PREFIX = ["remove ", "bỏ ", "gỡ "]; // "Remove Like" / "Bỏ thích"
  function isReactName(lbl) {
    for (var k in REACT) { if (REACT[k].indexOf(lbl) >= 0) return true; }
    return false;
  }
  function startsWithRemove(lbl) {
    for (var i = 0; i < REMOVE_PREFIX.length; i++) { if (lbl.indexOf(REMOVE_PREFIX[i]) === 0) return true; }
    return false;
  }

  // The main post's UFI react toggle. On a single-post permalink / reel / watch
  // page the post's UFI renders before the comment list, so the first matching
  // toggle in DOM order is the post's (comment "Like" links appear after).
  function findReactButton(root) {
    var btns = (root || document).querySelectorAll('[role="button"][aria-label]');
    for (var i = 0; i < btns.length; i++) {
      var lbl = lower(btns[i].getAttribute("aria-label"));
      if (isReactName(lbl) || startsWithRemove(lbl)) return btns[i];
    }
    return null;
  }
  // Whether the post already carries a reaction from us.
  function alreadyReacted(btn) {
    if (!btn) return false;
    if (String(btn.getAttribute("aria-pressed")) === "true") return true;
    return startsWithRemove(lower(btn.getAttribute("aria-label")));
  }
  // A specific reaction option inside the hover flyout.
  function findReactionOption(want) {
    var names = REACT[want] || [];
    var opts = document.querySelectorAll('[aria-label][role="button"], [aria-label][role="menuitem"], [role="menu"] [aria-label], [aria-label] img');
    for (var i = 0; i < opts.length; i++) {
      var lbl = lower(opts[i].getAttribute("aria-label"));
      if (names.indexOf(lbl) >= 0) return opts[i].closest('[role="button"],[role="menuitem"]') || opts[i];
    }
    return null;
  }

  // Poster + text snippet so the operator's report can confirm the right post.
  // NOTE: field is `actor`, not `author` — the bridge redacts any key containing
  // "auth" (author → "[redacted]"), so never name an output field with that.
  function postPreview(root) {
    var scope = root || document;
    var actor = "";
    var h = scope.querySelector('[role="article"] h2 a, [role="article"] h3 a, [role="article"] strong a, h2 a[role="link"]');
    if (h) actor = norm(h.innerText).slice(0, 80);
    var msg = "";
    var m = scope.querySelector('[data-ad-preview="message"], [data-ad-comet-preview="message"]');
    if (m) msg = norm(m.innerText);
    if (!msg) { var art = (scope.matches && scope.matches('[role="article"]')) ? scope : scope.querySelector('[role="article"]'); if (art) msg = norm(art.innerText); }
    return { actor: actor, snippet: msg.slice(0, 220), url: location.href };
  }

  function wrapCap(capId, status, extra) {
    var rec = Object.assign({ capability: capId, status: status, verified: false, error: null, ts: nowISO() }, extra || {});
    var ok = status !== "not_found" && status !== "error" && status !== "redirected";
    return { available: ok, capability: capId, count: 1, items: [rec], _debug: { href: location.href } };
  }
  function wrap(status, extra) { return wrapCap("fb.post.react", status, extra); }

  // Extract the post/reel/video id a URL pins to (so a write can confirm the
  // loaded page is still that exact target and never act on a drifted one).
  function targetIdFrom(url) {
    var u = String(url || "");
    var m = u.match(/\/reel\/(\d+)/) || u.match(/[?&]v=(\d+)/) || u.match(/\/videos\/(\d+)/)
      || u.match(/\/posts\/(\d+)/) || u.match(/story_fbid=(\d+)/) || u.match(/\/permalink\/(\d+)/) || u.match(/fbid=(\d+)/);
    return m ? m[1] : "";
  }
  // Returns {want, here} when the page drifted away from the requested id, else null.
  // No id to pin (e.g. a profile URL used with match_caption) → never a drift.
  function driftInfo(inputs) {
    var want = targetIdFrom(inputs && inputs._target_url);
    if (!want) return null;
    if (location.href.indexOf(want) !== -1) return null;
    return { want: want, here: location.href };
  }

  // Optional targeting: when the URL is a profile/timeline (not a single post),
  // `match_caption` scrolls to the article whose text contains that caption and
  // scopes the action to THAT post. No match_caption → act on the whole page
  // (correct for a permalink / reel, where there is one main post).
  async function resolveScope(inputs) {
    var cap = lower(inputs.match_caption || "");
    if (!cap) return document;
    for (var s = 0; s < 14; s++) {
      var arts = document.querySelectorAll('[role="article"]');
      for (var i = 0; i < arts.length; i++) {
        if (lower(arts[i].innerText).indexOf(cap) > -1) {
          try { arts[i].scrollIntoView({ block: "center" }); } catch (e) { /* ignore */ }
          await sleep(500);
          return arts[i];
        }
      }
      try { window.scrollBy(0, Math.round((window.innerHeight || 800) * 0.9)); } catch (e) { /* ignore */ }
      await sleep(700);
    }
    return null; // caption never appeared
  }

  // ---- P1: fb.post.react --------------------------------------------------
  async function doReact(inputs) {
    var reaction = lower(inputs.reaction || "like");
    var want = REACT[reaction] ? reaction : "like";

    var drift = driftInfo(inputs);
    if (drift) return wrap("redirected", { reaction: want, requested_id: drift.want, landed_url: drift.here, error: "page redirected to a different item (" + drift.here + ") — not reacting" });

    var scope = await resolveScope(inputs);
    if (scope === null) return wrap("not_found", { reaction: want, error: "no post matched match_caption on this page" });

    var btn = await waitFor(function () { return findReactButton(scope); }, 12000, 350);
    if (!btn) return wrap("not_found", { reaction: want, error: "react button not found" });

    var preview = postPreview(scope === document ? null : scope);
    if (inputs.dry_run) {
      return wrap("dry_run", { reaction: want, target_preview: preview, already_reacted: alreadyReacted(btn) });
    }
    // Idempotent + non-destructive: if the post is already reacted, do NOT click
    // (clicking Like again would REMOVE the reaction). P1 does not change an
    // existing reaction to a different one — it reports and leaves it.
    if (alreadyReacted(btn)) {
      return wrap("already", { reaction: want, verified: true, target_preview: preview });
    }

    await jitter();
    if (want === "like") {
      click(btn);
    } else {
      hover(btn);
      var opt = await waitFor(function () { return findReactionOption(want); }, 4000, 200);
      if (opt) { await jitter(); click(opt); } else { click(btn); /* fallback: plain like */ }
    }

    var ok = await waitFor(function () {
      var b2 = findReactButton(scope);
      return b2 && alreadyReacted(b2);
    }, 5000, 350);
    return wrap(ok ? "done" : "error", {
      reaction: want, verified: !!ok, target_preview: preview,
      error: ok ? null : "reaction not confirmed after click"
    });
  }

  // ---- P2: fb.post.comment ------------------------------------------------
  var COMMENT_LBL = /comment|bình luận|viết bình luận|write a comment|leave a comment/i;
  function findCommentBox(root) {
    var scope = root || document;
    var boxes = scope.querySelectorAll('div[contenteditable="true"][role="textbox"], textarea');
    for (var i = 0; i < boxes.length; i++) {
      var lbl = (boxes[i].getAttribute("aria-label") || boxes[i].getAttribute("placeholder") || "");
      var r = boxes[i].getBoundingClientRect();
      if (COMMENT_LBL.test(lbl) && r.width > 0 && r.height > 0) return boxes[i];
    }
    // fallback: first visible editable textbox in scope
    for (var j = 0; j < boxes.length; j++) { var rr = boxes[j].getBoundingClientRect(); if (rr.width > 0 && rr.height > 0) return boxes[j]; }
    return null;
  }
  // A "Comment" action that reveals the composer (reels / collapsed posts).
  function findCommentOpener(root) {
    var btns = (root || document).querySelectorAll('[role="button"][aria-label], div[role="button"]');
    for (var i = 0; i < btns.length; i++) {
      var lbl = lower(btns[i].getAttribute("aria-label") || btns[i].innerText || "");
      if (/^comment$|^bình luận$|write a comment|leave a comment/.test(lbl)) return btns[i];
    }
    return null;
  }
  function composerText(box) { return norm(box.innerText || box.value || box.textContent || ""); }
  async function typeInto(box, text) {
    box.focus(); hover(box); click(box); box.focus();
    await sleep(150);
    var ok = false;
    try { document.execCommand("selectAll", false, null); } catch (e) { /* ignore */ }
    try { ok = document.execCommand("insertText", false, text); } catch (e) { ok = false; }
    if (!composerText(box)) {
      // fallback for editors that ignore execCommand: beforeinput/input with data
      try {
        box.dispatchEvent(new InputEvent("beforeinput", { bubbles: true, cancelable: true, inputType: "insertText", data: text }));
        box.dispatchEvent(new InputEvent("input", { bubbles: true, cancelable: true, inputType: "insertText", data: text }));
      } catch (e) { /* ignore */ }
    }
    await sleep(200);
  }
  async function doComment(inputs) {
    var text = String(inputs.text || inputs.comment || "").trim();

    var drift = driftInfo(inputs);
    if (drift) return wrapCap("fb.post.comment", "redirected", { text: text, requested_id: drift.want, landed_url: drift.here, error: "page redirected to a different item (" + drift.here + ") — not commenting" });

    var scope = await resolveScope(inputs);
    if (scope === null) return wrapCap("fb.post.comment", "not_found", { text: text, error: "no post matched match_caption on this page" });
    var preview = postPreview(scope === document ? null : scope);
    if (!text) return wrapCap("fb.post.comment", "error", { text: "", target_preview: preview, error: "no comment text provided" });

    if (inputs.dry_run) {
      return wrapCap("fb.post.comment", "dry_run", { text: text, target_preview: preview, box_found: !!findCommentBox(scope === document ? null : scope) });
    }

    var root = scope === document ? null : scope;
    var box = findCommentBox(root);
    if (!box) { var opener = findCommentOpener(root); if (opener) { click(opener); box = await waitFor(function () { return findCommentBox(root); }, 5000, 250); } }
    if (!box) box = await waitFor(function () { return findCommentBox(root); }, 6000, 300);
    if (!box) return wrapCap("fb.post.comment", "not_found", { text: text, target_preview: preview, error: "comment composer not found" });

    await jitter();
    await typeInto(box, text);
    if (!composerText(box)) return wrapCap("fb.post.comment", "error", { text: text, target_preview: preview, error: "failed to enter text into composer" });

    await jitter();
    // Enter (no shift) posts a Facebook comment; Shift+Enter would be a newline.
    ["keydown", "keyup"].forEach(function (t) { try { box.dispatchEvent(new KeyboardEvent(t, { key: "Enter", code: "Enter", keyCode: 13, which: 13, bubbles: true, cancelable: true })); } catch (e) { /* ignore */ } });

    // Verify: the composer clears after a successful post.
    var cleared = await waitFor(function () { return !composerText(box); }, 6000, 400);
    var appeared = false;
    try { appeared = (document.body.innerText || "").indexOf(text.slice(0, 40)) > -1; } catch (e) { /* ignore */ }
    return wrapCap("fb.post.comment", cleared ? "done" : "error", {
      text: text, verified: !!cleared, appeared: appeared, target_preview: preview,
      error: cleared ? null : "comment not confirmed (composer still holds text)"
    });
  }

  // ---- dispatcher ---------------------------------------------------------
  window.__soloActRun = async function (capId, inputs) {
    inputs = inputs && typeof inputs === "object" ? inputs : {};
    try {
      if (capId === "fb.post.react") return await doReact(inputs);
      if (capId === "fb.post.comment") return await doComment(inputs);
      // P3: fb.message.send — added later.
      return { available: false, capability: capId, count: 0, items: [{ status: "error", error: "unknown or unimplemented action: " + capId }], _debug: { href: location.href } };
    } catch (e) {
      return { available: false, capability: capId, count: 0, items: [{ status: "error", error: String(e && e.message || e) }], _debug: { href: location.href, error: String(e) } };
    }
  };
})();
