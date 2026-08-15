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
  // Re-run when an OLDER copy of the lib is already installed (it has __soloActRun but
  // not the newer __soloActResolve): re-executing only reassigns the window globals, and
  // silently keeping a stale lib is how a resolve call turns into "function not present".
  if (window.__soloActRun && window.__soloActResolve) return;

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

  // ---- content-addressed targeting (match_text) -----------------------------
  // "Comment on the post that says X" used to be the AGENT's job: it listed the feed,
  // eyeballed which item matched, and handed the permalink back. That is judgement, not
  // mechanism — two agents gave two answers, and a wrong pick writes to the wrong post.
  // The match now happens HERE, in code: one deterministic filter over the SAME listing
  // extractor the read path uses, and a refusal — never a guess — when the answer is not
  // exactly one post.
  //
  // This resolver only ever RETURNS a permalink. It must not act inline on the listing
  // page: a feed holds one "Comment as …" composer PER POST, so acting there would land
  // on whichever one the selector happened to reach first. background.js navigates to the
  // resolved permalink and re-injects before anything is written.
  var LIST_CAPS = { "fb.group.posts": 1, "fb.profile.posts": 1, "fb.group.search_posts": 1, "fb.newsfeed": 1 };
  function listCapabilityFor(url, override) {
    if (override && LIST_CAPS[override]) return override;
    return /\/groups\/[^/?#]+/i.test(String(url || "")) ? "fb.group.posts" : "fb.profile.posts";
  }

  // The permalink is lifted from the PAGE's own GraphQL payload — untrusted data that is
  // about to become a tab navigation. Pin the host to an allowlist rather than a
  // /facebook\.com$/ test: that pattern also admits l.facebook.com, the link shim, which
  // forwards anywhere. A write must never be steered off-platform by feed content.
  var FB_HOSTS = { "facebook.com": 1, "www.facebook.com": 1, "m.facebook.com": 1, "web.facebook.com": 1 };
  function safeFbPermalink(raw) {
    var s = String(raw || "").trim();
    if (!/^https?:\/\//i.test(s)) return "";
    try {
      var u = new URL(s);
      if (!FB_HOSTS[lower(u.hostname)]) return "";
      return u.href;
    } catch (e) { return ""; }
  }

  function matchesText(hay, needle, mode, flags) {
    var h = norm(hay);
    if (mode === "regex") { try { return new RegExp(needle, flags || "i").test(h); } catch (e) { return false; } }
    if (mode === "exact") return lower(h) === lower(needle);
    return lower(h).indexOf(lower(needle)) > -1; // contains (default)
  }
  // One post arrives in several captures AND again in every replayed page. Without a
  // dedupe key every single match would come back as "ambiguous_match" and nothing would
  // ever be actionable. Prefer the canonical permalink; fall back to the story id.
  function dedupeKey(it) {
    var u = safeFbPermalink(it && it.url);
    if (u) { try { var p = new URL(u); return lower(p.origin + p.pathname.replace(/\/+$/, "")); } catch (e) { /* fall through */ } }
    return String((it && (it.post_id || it.id)) || "");
  }
  function candidatePreview(it) {
    return { url: String((it && it.url) || ""), text: norm(it && it.text).slice(0, 140), created_time: (it && it.created_time) || 0, from: (it && it._from) || "graphql" };
  }

  // Facebook server-renders the TOP of a feed into the initial document and only fetches
  // OLDER pages over GraphQL. gql_intercept.js hooks fetch/XHR, so the listing extractor
  // is systematically blind to the NEWEST posts — exactly the ones worth acting on.
  // Measured on the test group: fb.group.posts returned "Post 2" and "Post 3" while the
  // newest post ("Post 5 post 5 post 5") was rendered on the page the whole time. So read
  // the rendered feed as a SECOND source and merge; the canonical-url dedupe collapses the
  // overlap, and GraphQL items win because they carry post_id and created_time.
  // This lives here rather than in gql_extract.js on purpose: the read path is proven and
  // in daily use, and a write-targeting fix has no business changing what it returns.
  function domPermalink(rawHref) {
    var abs;
    try { abs = new URL(String(rawHref || ""), location.origin); } catch (e) { return ""; }
    if (!FB_HOSTS[lower(abs.hostname)]) return "";
    // /posts/<id> and /permalink/<id> carry their identity in the PATH, so drop the query
    // (Facebook hangs __cft__/__tn__ tracking blobs off feed links). permalink.php keeps
    // its query — that is where story_fbid lives.
    if (/\/(posts|permalink)\/[^/]+/i.test(abs.pathname)) return abs.origin + abs.pathname;
    return abs.href;
  }
  // The rendered-feed posts, extracted by filtering.js in the ISOLATED world and handed in
  // by background.js as _dom_posts. They are NOT re-scraped here on purpose. A hand-rolled
  // [role="article"] scan was tried first and measured wrong live: Facebook renders each
  // COMMENT as its own article whose permalink is the POST's url plus ?comment_id=…, so the
  // scan produced the right url carrying a commenter's words instead of the post body.
  // filtering.js already solves exactly that — selectContent() stops at the first comment
  // boundary — and it has been doing so in the daily read pipeline for months. The two
  // worlds share only the DOM, so the array has to travel through background.js.
  function providedDomPosts(inputs) {
    var raw = Array.isArray(inputs._dom_posts) ? inputs._dom_posts : [];
    var out = [];
    for (var i = 0; i < raw.length; i++) {
      var p = raw[i];
      if (!p) continue;
      var url = domPermalink(p.url);   // absolutise + re-apply the host allowlist
      var text = norm(p.text);
      if (!url || !text) continue;
      out.push({ id: "", post_id: "", url: url, text: text, created_time: 0, _from: "dom" });
    }
    return out;
  }

  async function resolveByContent(capId, inputs) {
    var needle = norm(inputs.match_text || "");
    var mode = lower(inputs.match_mode || "contains");
    if (["contains", "exact", "regex"].indexOf(mode) < 0) mode = "contains";
    if (!needle) return wrapCap(capId, "error", { error: "match_text is empty" });

    var listCap = listCapabilityFor(inputs._target_url || location.href, lower(inputs.match_source || ""));
    var base = { match_text: needle, match_mode: mode, list_capability: listCap, listing_url: location.href };
    var hasPaginate = typeof window.__soloGqlPaginate === "function";
    if (!hasPaginate && typeof window.__soloGqlExtractCapability !== "function") {
      return wrapCap(capId, "error", Object.assign({}, base, { error: "listing extractor not present — gql_extract.js was not injected into this page" }));
    }

    var listing = null;
    try {
      var lim = Number(inputs.match_max_pages);
      var lin = { max_pages: Number.isFinite(lim) ? lim : 3 };
      listing = hasPaginate ? await window.__soloGqlPaginate(listCap, lin) : window.__soloGqlExtractCapability(listCap, lin);
    } catch (e) {
      // Not fatal: the rendered feed is a second, independent source and often holds the
      // post anyway. Record why the capture failed and carry on with the DOM.
      base.listing_error = String(e && e.message || e);
    }

    // GraphQL first so its richer records win the dedupe; the rendered feed then supplies
    // whatever the intercept never saw (the newest posts).
    var gqlItems = (listing && Array.isArray(listing.items)) ? listing.items : [];
    var domItems = providedDomPosts(inputs);
    var items = gqlItems.concat(domItems);
    base.sources_read = { graphql: gqlItems.length, dom: domItems.length };

    var seen = {}, considered = [], hits = [];
    for (var i = 0; i < items.length; i++) {
      var it = items[i];
      if (!it) continue;
      var key = dedupeKey(it);
      if (!key || seen[key]) continue;
      seen[key] = 1;
      considered.push(it);
      if (matchesText(it.text || "", needle, mode, inputs.match_flags)) hits.push(it);
    }

    // "nothing was readable" is not "nothing matched" — the first is a capture problem to
    // retry, the second is an answer. Never collapse them into one status.
    if (!considered.length) {
      return wrapCap(capId, "listing_unavailable", Object.assign({}, base, {
        candidates_considered: 0, listing_reason: String((listing && listing.reason) || "no_items"),
        error: "no posts readable on this page — neither the " + listCap + " capture nor the rendered feed returned any; cannot resolve match_text"
      }));
    }
    if (!hits.length) {
      return wrapCap(capId, "no_match", Object.assign({}, base, {
        candidates_considered: considered.length,
        candidates: considered.slice(0, 5).map(candidatePreview),
        error: "no post matched match_text (" + mode + ") among " + considered.length + " posts read — nothing was written"
      }));
    }
    if (hits.length > 1) {
      return wrapCap(capId, "ambiguous_match", Object.assign({}, base, {
        candidates_considered: considered.length,
        candidates: hits.slice(0, 5).map(candidatePreview),
        error: "match_text matched " + hits.length + " posts — refusing to guess; nothing was written"
      }));
    }

    var hit = hits[0];
    var permalink = safeFbPermalink(hit.url);
    if (!permalink) {
      return wrapCap(capId, "error", Object.assign({}, base, {
        candidates_considered: considered.length, matched_text: norm(hit.text).slice(0, 220),
        error: "the matched post carries no usable facebook.com permalink (" + String(hit.url || "empty") + ")"
      }));
    }
    return wrapCap(capId, "resolved", Object.assign({}, base, {
      matched_post: { url: permalink, text: norm(hit.text).slice(0, 220), post_id: String(hit.post_id || hit.id || ""), created_time: hit.created_time || 0, from: hit._from || "graphql" },
      candidates_considered: considered.length
    }));
  }

  // Fail-safe for a half-updated install: match_text asks for a post that only the
  // orchestration layer can navigate to, so a write that sees it WITHOUT the resolved
  // marker is running on the listing page — where the first composer belongs to the wrong
  // post. Refuse rather than write somewhere plausible.
  function unresolvedMatch(capId, inputs) {
    if (!norm(inputs.match_text || "") || inputs._resolved_url) return null;
    return wrapCap(capId, "unresolved_match", {
      match_text: norm(inputs.match_text),
      error: "match_text was not resolved to a permalink before this ran (stale background.js?) — refusing to act on the listing page"
    });
  }

  // On the resolved path driftInfo() is not enough: targetIdFrom only recognises NUMERIC
  // ids, so a pfbid permalink pins nothing and any page would pass. Compare the paths
  // directly instead. This is the last thing standing between "the second navigation
  // landed" and writing onto the listing page we just searched.
  function resolvedDrift(capId, inputs) {
    var want = String(inputs._resolved_url || "");
    if (!want) return null;
    try {
      var w = new URL(want);
      var wp = lower(w.pathname.replace(/\/+$/, ""));
      if (lower(new URL(location.href).pathname.replace(/\/+$/, "")) === wp) return null;
      // Facebook may re-shape a permalink (…/posts/<id> ↔ permalink.php?story_fbid=<id>);
      // the post token surviving anywhere in the landed url still proves the same item.
      var tok = (wp.match(/(pfbid[0-9a-z]+|\d{6,})/i) || [])[1] || "";
      if (tok && lower(location.href).indexOf(lower(tok)) > -1) return null;
      return wrapCap(capId, "redirected", {
        resolved_url: want, landed_url: location.href,
        error: "the navigation to the matched post landed on a different page (" + location.href + ") — nothing was written"
      });
    } catch (e) { return null; }
  }

  // ---- P1: fb.post.react --------------------------------------------------
  async function doReact(inputs) {
    var reaction = lower(inputs.reaction || "like");
    var want = REACT[reaction] ? reaction : "like";

    var unresolved = unresolvedMatch("fb.post.react", inputs) || resolvedDrift("fb.post.react", inputs);
    if (unresolved) return unresolved;

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

    var unresolved = unresolvedMatch("fb.post.comment", inputs) || resolvedDrift("fb.post.comment", inputs);
    if (unresolved) return unresolved;

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
  //
  // messenger.com/e2ee/t/<vanity-or-id> is the preferred entry point over
  // facebook.com/messages/t/<numeric_id>, for three measured reasons: it accepts a
  // VANITY (no numeric-id lookup first), it is a dedicated surface with no docked chat
  // heads to confuse the composer count, and its page title becomes "<Name> | Messenger"
  // — a second, independent proof of who the thread is with. Both forms stay accepted.
  var WRITE_TO = /^write to\s+(.+)$/i;
  var THREAD_URL = [
    /^https?:\/\/(www\.)?messenger\.com\/(e2ee\/)?t\/[^/?#]+/i,
    /^https?:\/\/([\w-]+\.)?facebook\.com\/messages\/(e2ee\/)?t\/[^/?#]+/i
  ];
  function isThreadUrl(u) {
    var s = String(u || "");
    for (var i = 0; i < THREAD_URL.length; i++) { if (THREAD_URL[i].test(s)) return true; }
    return false;
  }
  // The identity anchor is the PROFILE ID (or vanity) in the requested url, not the display
  // name: people rename themselves, and a name guard would refuse a perfectly correct send
  // the day the recipient becomes someone else on paper. The id is stable; the name is a
  // label that happens to be on it today.
  //
  // The catch is that passing the E2EE gate rewrites the address to /e2ee/t/<thread_id>,
  // dropping the id — which is exactly why the original guard had nothing but the name to
  // work with. So the id is checked BEFORE the gate is touched: when Messenger cannot
  // resolve the id it falls back to the inbox (or another thread), and the requested id is
  // no longer in the address. That silent fallback is the failure this catches.
  function threadKeyFrom(u) {
    var m = String(u || "").match(/\/t\/([^/?#]+)/i);
    return m ? decodeURIComponent(m[1]) : "";
  }
  // "(3) Bob Nguyen | Messenger" -> "Bob Nguyen". Returns "" when the title carries no
  // name (facebook.com/messages renders a bare "Messenger"), which is NOT a failure —
  // guard 3 simply has nothing to check there and stands down.
  function recipientFromTitle() {
    var t = norm(document.title).replace(/^\(\s*\d+\s*\)\s*/, "");
    var m = t.match(/^(.+?)\s*[|·\-–]\s*(messenger|facebook)\b/i);
    var who = m ? norm(m[1]) : "";
    if (!who || /^(messenger|facebook|chats?|inbox)$/i.test(who)) return "";
    return who;
  }
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

    // Guard 0: the job url must be a THREAD. Previously a profile url was caught only
    // downstream, by the "more than one composer" guard — true but indirect, and it
    // depended on chat heads happening to be docked. Reject the wrong surface up front.
    var jobUrl = String(inputs._target_url || location.href);
    if (!isThreadUrl(jobUrl) && !isThreadUrl(location.href)) {
      return wrapCap("fb.message.send", "error", {
        text: text, recipient_expected: expected, job_url: jobUrl,
        error: "not_a_thread_url: open messenger.com/e2ee/t/<vanity-or-id> (or facebook.com/messages/t/<id>) — a profile url cannot identify its own owner"
      });
    }

    // Guard 1: the requested id must still be the open thread. Read this BEFORE the gate,
    // while the address still carries it.
    var wantKey = threadKeyFrom(inputs.recipient_id || jobUrl);
    var entryUrl = location.href;
    var entryKey = threadKeyFrom(entryUrl);
    var idVerified = !!wantKey && !!entryKey && lower(wantKey) === lower(entryKey);
    // A mismatch here is NOT proof of the wrong thread, and must not refuse on its own:
    // Messenger rewrites /t/<profile_id> to /t/<thread_id> once a thread is encrypted, and
    // on a thread that needs no gate that rewrite can happen before this code ever runs.
    // From the url alone "resolved my id" and "opened someone else" look identical. So a
    // mismatch only means the anchor is UNAVAILABLE — the name below becomes the evidence,
    // exactly as it was before this change. A genuine wrong thread is still caught there,
    // because a different thread is a different person with a different name.

    var passedGate = await passE2eeGate();
    var found = await waitFor(function () { var c = findMessageComposers(); return c.length ? c : null; }, 10000, 400) || [];

    // Guard 1: exactly one thread composer, or we cannot know which one is the target.
    if (found.length !== 1) {
      return wrapCap("fb.message.send", "error", {
        text: text, recipient_expected: expected, composers_found: found.length,
        composer_labels: found.map(function (c) { return c.label; }),
        passed_e2ee_gate: passedGate,
        error: found.length === 0 ? "no message composer on this page — open messenger.com/e2ee/t/<vanity-or-profile-id>. A url that is not a real thread (a photo id, say) lands on a page with no composer at all."
                                  : "ambiguous_composer: more than one open chat — refusing to guess the recipient"
      });
    }
    // Guard 3: the NAME, now a fallback rather than the anchor. A display name is not an
    // identity — the same profile is "Bob Nguyen" today and "Alex Nguyen" next week — so a
    // name that no longer matches must not veto a send whose id already checked out. It
    // still carries real weight when the id could NOT be verified: there it is the only
    // evidence left, and the old refusal stands.
    var box = found[0].el, label = found[0].label;
    var who = (label.match(WRITE_TO) || [])[1] || "";
    var titleWho = recipientFromTitle();
    var nameMatched = !expected ? null : (lower(who) === lower(expected));
    var titleCheck = !titleWho ? "unavailable" : (!expected ? "unchecked" : (lower(titleWho) === lower(expected) ? "match" : "mismatch"));
    var identity = {
      requested_id: wantKey, open_id: entryKey, id_verified: idVerified, entry_url: entryUrl,
      recipient_open: who, recipient_expected: expected || null, name_matched: nameMatched,
      recipient_title: titleWho, title_check: titleCheck
    };
    if (!idVerified && expected && nameMatched === false) {
      return wrapCap("fb.message.send", "error", Object.assign({ text: text, passed_e2ee_gate: passedGate }, identity, {
        error: "recipient_mismatch: the thread id could not be verified AND the open thread is with \"" + who + "\", not \"" + expected + "\" — nothing was typed"
      }));
    }
    if (!idVerified && !expected) {
      return wrapCap("fb.message.send", "error", Object.assign({ text: text, passed_e2ee_gate: passedGate }, identity, {
        error: "unverified_recipient: the thread id could not be read from the url and no recipient_name was given — there is nothing left to prove who this is"
      }));
    }

    if (inputs.dry_run) {
      return wrapCap("fb.message.send", "dry_run", Object.assign({ text: text, recipient: who, passed_e2ee_gate: passedGate }, identity));
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
    return wrapCap("fb.message.send", sent ? "done" : "error", Object.assign({
      text: text, recipient: who, verified: sent, cleared: !!cleared, appeared: !!appeared,
      passed_e2ee_gate: passedGate, thread_url: location.href,
      error: sent ? null : (cleared ? "composer cleared but the message did not appear" : "message not confirmed (composer still holds text)")
    }, identity));
  }

  // ---- dispatcher ---------------------------------------------------------
  // Phase 1 of a match_text write: read the listing and return AT MOST one permalink.
  // Deliberately separate from __soloActRun so this call can never write anything —
  // background.js navigates to the result and calls __soloActRun on the target page.
  window.__soloActResolve = async function (capId, inputs) {
    inputs = inputs && typeof inputs === "object" ? inputs : {};
    try { return await resolveByContent(String(capId || ""), inputs); }
    catch (e) { return wrapCap(String(capId || ""), "error", { error: "resolve failed: " + String(e && e.message || e) }); }
  };

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
