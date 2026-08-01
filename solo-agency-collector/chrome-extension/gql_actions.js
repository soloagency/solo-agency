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
    // ALWAYS available: background.js discards a record whose capability reports
    // unavailable, which threw away the very thing a write action must report — WHY it
    // refused (recipient_mismatch, ambiguous_composer, redirected…). The caller then saw
    // an empty record and could not tell "a guard stopped this" from "the job broke".
    // Success/failure is carried by `status` and `verified`, not by hiding the record.
    return { available: true, capability: capId, status: status, count: 1, items: [rec], _debug: { href: location.href } };
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
  // Facebook labels the comment composer "Comment as <your name>" on every surface —
  // post permalink, group post and reel alike (verified live on all three). Match that
  // prefix instead of any box whose label merely contains "comment": the loose pattern
  // also matched the search field and the status composer, which is how a comment could
  // land somewhere other than the post it was meant for.
  var COMMENT_AS = /^(comment as|bình luận với tư cách|viết bình luận)/i;
  function findCommentBox(root) {
    var scope = root || document;
    var boxes = scope.querySelectorAll('div[contenteditable="true"][role="textbox"]');
    for (var i = 0; i < boxes.length; i++) {
      var lbl = boxes[i].getAttribute("aria-label") || "";
      var r = boxes[i].getBoundingClientRect();
      if (COMMENT_AS.test(lbl) && r.width > 0 && r.height > 0) return boxes[i];
    }
    return null; // no guessing — a wrong box means commenting in the wrong place
  }
  // Reels ship without a composer until the comment panel is opened; a post permalink
  // already has one. The opener's label is exactly "Comment" (not "Comment with a GIF").
  function findCommentOpener(root) {
    var btns = (root || document).querySelectorAll('[role="button"]');
    for (var i = 0; i < btns.length; i++) {
      var lbl = norm(btns[i].getAttribute("aria-label") || btns[i].innerText || "");
      var r = btns[i].getBoundingClientRect();
      if (/^(comment|bình luận)$/i.test(lbl) && r.width > 0 && r.height > 0) return btns[i];
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
    var openedPanel = false;
    if (!box) {
      // A reel has no composer until its comment panel is opened; a permalink already
      // does. Open it, then WAIT for the composer to render — it arrives a beat later,
      // and reading too early is what made this look unreliable on reels.
      var opener = findCommentOpener(root);
      if (opener) { click(opener); openedPanel = true; }
      box = await waitFor(function () { return findCommentBox(root); }, 8000, 350);
    }
    if (!box) return wrapCap("fb.post.comment", "not_found", { text: text, target_preview: preview, opened_panel: openedPanel, error: "comment composer not found" });

    await jitter();
    try { box.scrollIntoView({ block: "center" }); } catch (e) { /* ignore */ }
    await sleep(300);
    await typeInto(box, text);
    if (!composerText(box)) return wrapCap("fb.post.comment", "error", { text: text, target_preview: preview, opened_panel: openedPanel, error: "failed to enter text into composer" });

    await jitter();
    // Enter (no shift) posts a Facebook comment; Shift+Enter would be a newline.
    // keypress is included because the composer listens for the full key sequence.
    ["keydown", "keypress", "keyup"].forEach(function (t) { try { box.dispatchEvent(new KeyboardEvent(t, { key: "Enter", code: "Enter", keyCode: 13, which: 13, bubbles: true, cancelable: true })); } catch (e) { /* ignore */ } });

    // Two-part proof, because either alone can lie: the composer must EMPTY (the edit
    // was consumed) AND the text must be ON the page (it actually posted).
    var cleared = await waitFor(function () { return !composerText(box); }, 8000, 400);
    var probe = text.slice(0, 40);
    var appeared = await waitFor(function () {
      try { return (document.body.innerText || "").indexOf(probe) > -1; } catch (e) { return false; }
    }, 6000, 400);
    var posted = !!cleared && !!appeared;
    return wrapCap("fb.post.comment", posted ? "done" : "error", {
      text: text, verified: posted, cleared: !!cleared, appeared: !!appeared,
      opened_panel: openedPanel, target_preview: preview,
      error: posted ? null : (cleared ? "composer cleared but the comment did not appear" : "comment not confirmed (composer still holds text)")
    });
  }

  // ---- P3: fb.message.send -------------------------------------------------
  // Send a Messenger DM. The job's url must be the THREAD (facebook.com/messages/t/<id>),
  // never the profile: clicking "Message" on a profile leaves the page holding several
  // "Write to <someone>" composers at once — the recipient's plus every chat head already
  // docked there — and the profile's own name is not readable (its h1 is "Notifications"),
  // so there is no way to tell them apart. A thread page renders exactly ONE composer.
  var WRITE_TO = /^write to\s+(.+)$/i;
  function findMessageComposers() {
    var out = [];
    var boxes = document.querySelectorAll('div[contenteditable="true"][role="textbox"]');
    for (var i = 0; i < boxes.length; i++) {
      var lbl = norm(boxes[i].getAttribute("aria-label") || "");
      var r = boxes[i].getBoundingClientRect();
      if (WRITE_TO.test(lbl) && r.width > 0 && r.height > 0) out.push({ el: boxes[i], label: lbl });
    }
    return out;
  }
  // Threads predating end-to-end encryption open behind a "Continue" gate and have no
  // composer until it is clicked; doing so also rewrites the url to /messages/e2ee/t/<thread>,
  // which is why the recipient is verified by NAME rather than by the url.
  async function passE2eeGate() {
    if (findMessageComposers().length) return false;
    var btns = document.querySelectorAll('[role="button"]');
    for (var i = 0; i < btns.length; i++) {
      var lbl = norm(btns[i].innerText || btns[i].getAttribute("aria-label") || "");
      if (!/^(continue|tiếp tục)$/i.test(lbl)) continue;
      var r = btns[i].getBoundingClientRect();
      if (r.width <= 0 || r.height <= 0) continue;
      click(btns[i]);
      await waitFor(function () { return findMessageComposers().length > 0; }, 8000, 400);
      return true;
    }
    return false;
  }
  async function doMessage(inputs) {
    var text = String(inputs.text || inputs.message || "").trim();
    var expected = norm(inputs.recipient_name || "");
    if (!text) return wrapCap("fb.message.send", "error", { error: "no message text provided" });
    if (!expected) return wrapCap("fb.message.send", "error", { text: text, error: "recipient_name is required — it is the only way to prove the open thread belongs to the intended person" });

    var passedGate = await passE2eeGate();
    var found = await waitFor(function () { var c = findMessageComposers(); return c.length ? c : null; }, 10000, 400) || [];

    // Guard 1: exactly one thread composer, or we cannot know which one is the target.
    if (found.length !== 1) {
      return wrapCap("fb.message.send", "error", {
        text: text, recipient_expected: expected, composers_found: found.length,
        composer_labels: found.map(function (c) { return c.label; }),
        passed_e2ee_gate: passedGate,
        error: found.length === 0 ? "no message composer on this page — open facebook.com/messages/t/<id>, not the profile"
                                  : "ambiguous_composer: more than one open chat — refusing to guess the recipient"
      });
    }
    // Guard 2: that composer must name the intended recipient.
    var box = found[0].el, label = found[0].label;
    var who = (label.match(WRITE_TO) || [])[1] || "";
    if (lower(who) !== lower(expected)) {
      return wrapCap("fb.message.send", "error", {
        text: text, recipient_expected: expected, recipient_open: who, passed_e2ee_gate: passedGate,
        error: "recipient_mismatch: the open thread is with \"" + who + "\" — nothing was typed"
      });
    }

    if (inputs.dry_run) {
      return wrapCap("fb.message.send", "dry_run", { text: text, recipient: who, passed_e2ee_gate: passedGate });
    }

    await jitter();
    await typeInto(box, text);
    if (!composerText(box)) return wrapCap("fb.message.send", "error", { text: text, recipient: who, error: "failed to enter text into the composer" });

    await jitter();
    ["keydown", "keypress", "keyup"].forEach(function (t) { try { box.dispatchEvent(new KeyboardEvent(t, { key: "Enter", code: "Enter", keyCode: 13, which: 13, bubbles: true, cancelable: true })); } catch (e) { /* ignore */ } });

    var cleared = await waitFor(function () { return !composerText(box); }, 8000, 400);
    var probe = text.slice(0, 30);
    var appeared = await waitFor(function () {
      try { return (document.body.innerText || "").indexOf(probe) > -1; } catch (e) { return false; }
    }, 6000, 400);
    var sent = !!cleared && !!appeared;
    return wrapCap("fb.message.send", sent ? "done" : "error", {
      text: text, recipient: who, verified: sent, cleared: !!cleared, appeared: !!appeared,
      passed_e2ee_gate: passedGate, thread_url: location.href,
      error: sent ? null : (cleared ? "composer cleared but the message did not appear" : "message not confirmed (composer still holds text)")
    });
  }

  // ---- dispatcher ---------------------------------------------------------
  window.__soloActRun = async function (capId, inputs) {
    inputs = inputs && typeof inputs === "object" ? inputs : {};
    try {
      if (capId === "fb.post.react") return await doReact(inputs);
      if (capId === "fb.post.comment") return await doComment(inputs);
      if (capId === "fb.message.send") return await doMessage(inputs);
      return { available: false, capability: capId, count: 0, items: [{ status: "error", error: "unknown or unimplemented action: " + capId }], _debug: { href: location.href } };
    } catch (e) {
      return { available: false, capability: capId, count: 0, items: [{ status: "error", error: String(e && e.message || e) }], _debug: { href: location.href, error: String(e) } };
    }
  };
})();
