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
      base.listing_error = String(e && e.message || e);
    }

    // GRAPHQL ONLY. A rendered-feed source was added here for a day and then removed: it was
    // introduced to recover a post the GraphQL listing "could not see", and that premise was
    // wrong. The missing post was an ANONYMOUS group post, which Facebook does not serve
    // through the group feed query at all — the feed reported end-of-feed without it, and a
    // dump of every edge showed both real posts present and accepted. Nothing was blind.
    //
    // What the DOM source did add was damage: Facebook renders each COMMENT as its own
    // [role="article"] whose permalink is the POST's url plus ?comment_id=…, so a scan
    // returned the right url carrying a commenter's words, and the group's "Featured" card —
    // not a post — arrived wearing a real post's permalink. In GraphQL a story's id, url and
    // text come from ONE node and cannot be mispaired.
    //
    // An anonymous post is also the one kind this campaign has no use for: no author to add as
    // a lead, no profile for a reader to click through to. Losing it costs nothing.
    var items = (listing && Array.isArray(listing.items)) ? listing.items : [];
    base.sources_read = { graphql: items.length };

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
  // The token that identifies the POST inside a permalink. Taking "the first long
  // number in the path" is wrong for a group permalink: /groups/<gid>/posts/<pid>
  // yields the GROUP id, which of course survives on the group feed — so a deleted
  // post that redirects to its feed looked like "same item" and the comment would
  // land under whatever post sits on top. Read the post segment explicitly, and only
  // fall back to a bare number when there is no post segment at all.
  function postTokenIn(path) {
    var p = lower(String(path || ""));
    var m = p.match(/\/posts\/(pfbid[0-9a-z]+|\d{6,})/)
      || p.match(/\/permalink\/(pfbid[0-9a-z]+|\d{6,})/)
      || p.match(/\/videos\/(\d{6,})/) || p.match(/\/reel\/(\d{6,})/)
      || p.match(/(pfbid[0-9a-z]+)/);
    if (m) return m[1];
    // No post segment (a group root, a profile): the first long run is the best we have.
    return (p.match(/(\d{6,})/) || [])[1] || "";
  }

  function resolvedDrift(capId, inputs) {
    var want = String(inputs._resolved_url || "");
    if (!want) return null;
    try {
      var w = new URL(want);
      var wp = lower(w.pathname.replace(/\/+$/, ""));
      if (lower(new URL(location.href).pathname.replace(/\/+$/, "")) === wp) return null;
      // Facebook may re-shape a permalink (…/posts/<id> ↔ permalink.php?story_fbid=<id>);
      // the post token surviving anywhere in the landed url still proves the same item.
      var tok = postTokenIn(wp);
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
  // Facebook's composer label is NOT a stable key. It has now moved three times: the original
  // "Comment as <name>", two label guesses that both missed, and — measured on one post, two
  // accounts, at the same moment — "Answer as Binh" for one and "Write an answer…" for the other.
  // A question-style group renames Comment to Answer, and the wording also differs per viewer.
  // Widening the pattern again would be the fourth guess.
  //
  // So the label is now a HINT and the structure is the key: inside the post's own scope, a
  // visible contenteditable textbox is the composer. What is enumerated instead is the set of
  // impostors, because those are stable — the search field, a Messenger thread, and the
  // create-a-post composer are the same three boxes on every layout.
  var COMPOSER_HINT = /(comment|answer|reply|bình luận|trả lời|phản hồi)/i;
  var COMPOSER_IMPOSTOR = /(^| )(search|tìm kiếm|message|nhắn tin|aa|write something|create a post|what's on your mind|bạn đang nghĩ gì|viết gì đó|tạo bài viết)/i;
  // Every editable box the page is showing, with the attributes a matcher could key on. Shared by
  // dry_run and by the not_found path: both exist to answer "why did it not type", and answering
  // it twice in two shapes is how the two answers drift apart.
  function seenTextboxes() {
    var out = [];
    try {
      var tb = document.querySelectorAll('div[contenteditable="true"], [role="textbox"], textarea');
      for (var i = 0; i < tb.length && out.length < 12; i++) {
        var r = tb[i].getBoundingClientRect();
        out.push({
          label: (tb[i].getAttribute("aria-label") || "").slice(0, 80),
          placeholder: (tb[i].getAttribute("data-placeholder") || tb[i].getAttribute("placeholder") || "").slice(0, 80),
          role: tb[i].getAttribute("role") || "", editable: tb[i].getAttribute("contenteditable") || "",
          visible: r.width > 0 && r.height > 0
        });
      }
    } catch (e) { /* diagnostics must never be the thing that fails */ }
    return out;
  }
  function composerLabelOf(el) {
    return norm((el.getAttribute("aria-label") || "") + " " + (el.getAttribute("data-placeholder") || el.getAttribute("placeholder") || ""));
  }
  function findCommentBox(root) {
    var scope = root || document;
    var boxes = scope.querySelectorAll('div[contenteditable="true"][role="textbox"], div[contenteditable="true"]');
    var candidates = [];
    for (var i = 0; i < boxes.length; i++) {
      var lbl = composerLabelOf(boxes[i]);
      var r = boxes[i].getBoundingClientRect();
      if (r.width <= 0 || r.height <= 0) continue;
      if (COMPOSER_IMPOSTOR.test(lbl)) continue;             // search / chat / status composer
      if (COMPOSER_HINT.test(lbl)) return boxes[i];          // said so itself, in either attribute
      candidates.push(boxes[i]);
    }
    // No label matched. One surviving box inside the post's scope is unambiguous, so take it —
    // that is what a reader would do. Zero or several is still a refusal: a wrong box means
    // commenting in the wrong place, and that is worse than not commenting.
    return candidates.length === 1 ? candidates[0] : null;
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
  // A non-member sees the post and NO composer; what they do see is an invitation to
  // join. That difference is the whole diagnosis — reporting it as "composer not found"
  // sends the operator hunting for a bug in the extension when the fix is one click in
  // Facebook. Returns the label found, or "".
  var JOIN_LABEL = /^(join group|join this group|join|tham gia nhóm|tham gia)$/i;
  function findJoinAffordance() {
    var els = document.querySelectorAll('[role="button"], a[role="link"]');
    for (var i = 0; i < els.length; i++) {
      var lbl = norm(els[i].getAttribute("aria-label") || els[i].innerText || "");
      var r = els[i].getBoundingClientRect();
      if (JOIN_LABEL.test(lbl) && r.width > 0 && r.height > 0) return lbl;
    }
    return "";
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
      var dbox = findCommentBox(scope === document ? null : scope);
      var out = { text: text, target_preview: preview, box_found: !!dbox };
      if (dbox) out.box_label = composerLabelOf(dbox).slice(0, 80);
      // dry_run is the mode you reach for BECAUSE something is wrong, so it carries the same
      // evidence the failure path does. Reporting box_found:false alone sends the operator back
      // to guessing, which is how the label was mis-guessed twice already.
      else out.seen_textboxes = seenTextboxes();
      return wrapCap("fb.post.comment", "dry_run", out);
    }

    var root = scope === document ? null : scope;
    var box = findCommentBox(root);
    var openedPanel = false;
    if (!box) {
      // The composer can be missing for two different reasons and the old order could
      // only survive one: a reel ships without a composer until its panel is opened,
      // while a permalink that has not finished rendering has neither the composer NOR
      // the opener yet. Looking for the opener exactly once, BEFORE the wait, meant a
      // slow page then waited 8s for a box that could only appear after a click that
      // never happened — measured live on a group post permalink: opened_panel:false,
      // "comment composer not found", on a page that had the composer moments later.
      // Re-look for the opener on every poll instead.
      box = await waitFor(function () {
        var b = findCommentBox(root);
        if (b) return b;
        if (!openedPanel) {
          var opener = findCommentOpener(root);
          if (opener) { click(opener); openedPanel = true; }
        }
        return null;
      }, 12000, 400);
    }
    if (!box) {
      var join = findJoinAffordance();
      // Two label-based selectors both missed, twice, on a post the collector could
      // read perfectly and in a group the account belongs to. Guessing which label
      // changed is how the last two attempts were spent — so report what is actually
      // on the page instead. This is the evidence the next fix is written from.
      var sawBoxes = seenTextboxes(), sawButtons = [];
      try {
        var bt = document.querySelectorAll('[role="button"]');
        for (var ci = 0; ci < bt.length && sawButtons.length < 25; ci++) {
          var lb = norm(bt[ci].getAttribute("aria-label") || bt[ci].innerText || "");
          if (!lb || lb.length > 40) continue;
          var cr = bt[ci].getBoundingClientRect();
          if (cr.width > 0 && cr.height > 0) sawButtons.push(lb.slice(0, 40));
        }
      } catch (e) { /* diagnostics must never be the thing that fails */ }
      return wrapCap("fb.post.comment", join ? "not_a_member" : "not_found", {
        text: text, target_preview: preview, opened_panel: openedPanel, join_prompt: join,
        seen_textboxes: sawBoxes, seen_buttons: sawButtons, page_lang: (document.documentElement && document.documentElement.lang) || "",
        error: join
          ? "this account is not a member of the group — Facebook offers \"" + join + "\" instead of a composer. Join the group in that Chrome profile, then approve again."
          : "comment composer not found"
      });
    }

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

  // ---- P4: fb.group.post ---------------------------------------------------
  // Publish a NEW post into a group. This is the most exposed write of the set: a comment
  // sits under someone else's post, a DM is private, but a post stands alone in front of the
  // whole group and is the thing members report as spam. So it guards harder, not softer.
  //
  // Shape of the surface (why this is not the comment flow):
  //   - the group page shows a TRIGGER ("Write something...", "Viết gì đó..."), not a composer;
  //   - clicking it opens a MODAL dialog which holds the real contenteditable;
  //   - submission is a "Post" BUTTON, not the Enter key — Enter inserts a newline here.
  var GROUP_URL = /^https?:\/\/([\w-]+\.)?facebook\.com\/groups\/[^/?#]+/i;
  var COMPOSER_TRIGGER = /(write something|create (a )?public post|viết gì đó|bạn viết gì đi)/i;
  var POST_BUTTON = /^(post|đăng)$/i;

  function isGroupUrl(u) { return GROUP_URL.test(String(u || "")); }
  function groupIdFrom(u) {
    var m = String(u || "").match(/\/groups\/([^/?#]+)/i);
    return m ? m[1] : "";
  }
  // The composer dialog is the only [role=dialog] holding a textbox. Requiring exactly one
  // is the same rule the DM path uses, for the same reason: several open composers mean the
  // page is not in the state we think it is, and typing into "the first one" is a guess.
  function findComposerDialogs() {
    var out = [], dialogs = document.querySelectorAll('[role="dialog"]');
    for (var i = 0; i < dialogs.length; i++) {
      var r = dialogs[i].getBoundingClientRect();
      if (r.width <= 0 || r.height <= 0) continue;
      if (dialogs[i].querySelector('div[contenteditable="true"][role="textbox"]')) out.push(dialogs[i]);
    }
    return out;
  }
  function findComposerTrigger() {
    var els = document.querySelectorAll('[role="button"], [role="textbox"], div[tabindex]');
    for (var i = 0; i < els.length; i++) {
      var lbl = norm(els[i].getAttribute("aria-label") || els[i].innerText || "");
      var r = els[i].getBoundingClientRect();
      if (COMPOSER_TRIGGER.test(lbl) && r.width > 0 && r.height > 0) return els[i];
    }
    return null;
  }
  function findPostButton(dialog) {
    var btns = dialog.querySelectorAll('[role="button"], button');
    for (var i = 0; i < btns.length; i++) {
      var lbl = norm(btns[i].getAttribute("aria-label") || btns[i].innerText || "");
      var r = btns[i].getBoundingClientRect();
      if (POST_BUTTON.test(lbl) && r.width > 0 && r.height > 0) return btns[i];
    }
    return null;
  }
  // Many groups hold member posts for an admin to approve. "Submitted" is NOT "published",
  // and reporting the first as the second would corrupt the ledger and the operator's read of
  // what is actually live. Facebook says so in a toast/notice — look for it before claiming.
  var PENDING_APPROVAL = /(pending (admin )?approval|waiting for approval|admin will review|chờ (quản trị viên )?phê duyệt|đang chờ duyệt)/i;
  function approvalNotice() {
    try {
      var t = document.body.innerText || "";
      var m = t.match(PENDING_APPROVAL);
      return m ? norm(m[0]) : "";
    } catch (e) { return ""; }
  }

  async function doGroupPost(inputs) {
    var text = String(inputs.text || inputs.message || "").trim();
    if (!text) return wrapCap("fb.group.post", "error", { error: "no post text provided" });

    // Guard 0: the job url must be a GROUP. A permalink or profile would open a different
    // composer entirely — on a profile it would post to the operator's own timeline.
    var jobUrl = String(inputs._target_url || location.href);
    if (!isGroupUrl(jobUrl) || !isGroupUrl(location.href)) {
      return wrapCap("fb.group.post", "error", {
        text: text, job_url: jobUrl, landed_url: location.href,
        error: "not_a_group_url: open facebook.com/groups/<id> — a permalink or profile opens a different composer, and on a profile this would post to the operator's own timeline"
      });
    }
    // Guard 1: the group we landed on must be the group we were sent to.
    var wantGroup = groupIdFrom(jobUrl), hereGroup = groupIdFrom(location.href);
    if (wantGroup && hereGroup && lower(wantGroup) !== lower(hereGroup)) {
      return wrapCap("fb.group.post", "error", {
        text: text, requested_group: wantGroup, landed_group: hereGroup,
        error: "group_mismatch: asked for \"" + wantGroup + "\" but landed on \"" + hereGroup + "\" — nothing was typed"
      });
    }

    // Open the composer. On a group page the box is a trigger, not an editor.
    var dialogs = findComposerDialogs();
    var openedDialog = false;
    if (!dialogs.length) {
      var trigger = findComposerTrigger();
      if (!trigger) {
        return wrapCap("fb.group.post", "error", {
          text: text, group: hereGroup,
          error: "no composer trigger on this page — the account may not be a member, or posting is restricted to admins"
        });
      }
      click(trigger);
      openedDialog = true;
      await waitFor(function () { return findComposerDialogs().length > 0; }, 8000, 300);
      dialogs = findComposerDialogs();
    }
    // Guard 2: exactly ONE composer dialog.
    if (dialogs.length !== 1) {
      return wrapCap("fb.group.post", "error", {
        text: text, group: hereGroup, dialogs_found: dialogs.length, opened_dialog: openedDialog,
        error: dialogs.length === 0
          ? "the composer dialog did not open"
          : "ambiguous_composer: " + dialogs.length + " composer dialogs are open — nothing was typed"
      });
    }
    var dialog = dialogs[0];
    var box = dialog.querySelector('div[contenteditable="true"][role="textbox"]');
    if (!box) {
      return wrapCap("fb.group.post", "error", { text: text, group: hereGroup, error: "composer dialog has no text box" });
    }

    if (inputs.dry_run) {
      return wrapCap("fb.group.post", "dry_run", {
        text: text, group: hereGroup, opened_dialog: openedDialog,
        post_button_found: !!findPostButton(dialog)
      });
    }

    await jitter();
    await typeInto(box, text);
    if (!composerText(box)) {
      return wrapCap("fb.group.post", "error", { text: text, group: hereGroup, error: "failed to enter text into the composer" });
    }

    // Submit with the BUTTON. Enter inserts a newline in this composer — the comment flow's
    // Enter trick would silently do nothing here except break the post into lines.
    var btn = findPostButton(dialog);
    if (!btn) {
      return wrapCap("fb.group.post", "error", {
        text: text, group: hereGroup, typed: true,
        error: "no enabled Post button in the composer — text was typed but NOT submitted"
      });
    }
    await jitter();
    click(btn);

    // Two-part proof, same rule as the comment path: the dialog must close (the submit was
    // consumed) AND the text must be on the page (it actually went somewhere).
    var closed = await waitFor(function () { return findComposerDialogs().length === 0; }, 12000, 500);
    var probe = text.slice(0, 40);
    var appeared = await waitFor(function () {
      try { return (document.body.innerText || "").indexOf(probe) > -1; } catch (e) { return false; }
    }, 8000, 500);
    var pending = approvalNotice();

    // published | pending_admin_approval | error — never collapse the middle one into "done".
    var status = pending ? "pending_admin_approval" : ((closed && appeared) ? "done" : "error");
    return wrapCap("fb.group.post", status, {
      text: text, group: hereGroup, group_url: location.href,
      verified: status === "done", closed: !!closed, appeared: !!appeared,
      opened_dialog: openedDialog, approval_notice: pending || null,
      error: status === "error"
        ? (closed ? "the composer closed but the post did not appear" : "the composer did not close — the post may not have been submitted")
        : null
    });
  }

  // ---- P5: fb.profile.post --------------------------------------------------
  // Publish a NEW post on the operator's OWN timeline. Same composer shape as a group post
  // (trigger → modal dialog → "Post" button), with the guards inverted: the page must be the
  // operator's own profile root (facebook.com/me resolves there) or the home feed — never a
  // group, a permalink, a media page, or somebody ELSE's profile. On a friend's profile the
  // box says "Write something to <Name>…" and would publish on THEIR timeline; the own-timeline
  // trigger is the only one that says "What's on your mind" / "Bạn đang nghĩ gì", so that
  // wording is the identity check, on top of the url shape.
  var OWN_TRIGGER = /(what'?s on your mind|bạn đang nghĩ gì)/i;
  var OTHER_TIMELINE_TRIGGER = /(write something to|write (something )?on [^?]*timeline|viết gì đó cho)/i;
  var NOT_A_TIMELINE = /facebook\.com\/(groups|events|marketplace|watch|reel|reels|videos|photo|photos|posts|permalink|story|stories|messages|search|pages|gaming|share|hashtag)([/?#]|$)|\/posts\/|permalink\.php|story\.php|photo\.php|\/videos\//i;
  var AUDIENCE = {
    public: ["public", "công khai"],
    friends: ["friends", "bạn bè"],
    only_me: ["only me", "chỉ mình tôi"]
  };
  var AUDIENCE_CONTROL = /(edit privacy|sharing with|select audience|chỉnh sửa quyền riêng tư|chia sẻ với|đối tượng)/i;
  // "Done with privacy audience selection and …" is the real label (measured 2026-09-10).
  var DONE_BUTTON = /^(done|save|xong|lưu)\b/i;
  // Measured 2026-09-10 on the operator's own profile: the composer is a TWO-step flow — the
  // first screen's submit is "Next" (disabled until text is typed), the "Post" button only
  // exists on the second screen. The home feed and groups submit with "Post" directly.
  var NEXT_BUTTON = /^(next|tiếp( theo)?)$/i;
  var COMPOSER_LABEL = /(create post|tạo bài viết)/i;
  function findNextButton(dialog) {
    var btns = dialog.querySelectorAll('[role="button"], button');
    for (var i = 0; i < btns.length; i++) {
      var lbl = norm(btns[i].getAttribute("aria-label") || btns[i].innerText || "");
      var r = btns[i].getBoundingClientRect();
      if (NEXT_BUTTON.test(lbl) && r.width > 0 && r.height > 0) return btns[i];
    }
    return null;
  }
  function isDisabled(el) { return !!el && String(el.getAttribute("aria-disabled")) === "true"; }
  // Every visible dialog that still looks like the composer (its text box, a Post/Next
  // button, or its title) — the second step of the two-step flow has no text box, so the
  // "did it close" proof cannot rely on findComposerDialogs alone.
  function composerLikeDialogs() {
    var out = [], dialogs = document.querySelectorAll('[role="dialog"]');
    for (var i = 0; i < dialogs.length; i++) {
      var d = dialogs[i], r = d.getBoundingClientRect();
      if (r.width <= 0 || r.height <= 0) continue;
      if (d.querySelector('div[contenteditable="true"][role="textbox"]') || findPostButton(d) || findNextButton(d) || COMPOSER_LABEL.test(norm(d.getAttribute("aria-label") || ""))) out.push(d);
    }
    return out;
  }
  function buttonLabels(root, max) {
    var out = [], btns = root.querySelectorAll('[role="button"], button');
    for (var i = 0; i < btns.length && out.length < (max || 30); i++) {
      var r = btns[i].getBoundingClientRect();
      if (r.width <= 0 || r.height <= 0) continue;
      var lbl = norm(btns[i].getAttribute("aria-label") || btns[i].innerText || "").slice(0, 40);
      if (lbl) out.push(isDisabled(btns[i]) ? lbl + " [disabled]" : lbl);
    }
    return out;
  }

  var PROFILE_SUBTAB = /^\/[^/?#]+\/(about|about_[a-z_]+|photos|photos_by|photos_albums|videos|friends|friends_[a-z_]+|reels|map|likes|music|sports|events|groups|mentions|reviews|check-ins|tv|followers|following|saved|notes|posts)(\/|$)/i;
  function isTimelineUrl(u) {
    u = String(u || "");
    if (!/^https?:\/\/(www\.|m\.|web\.)?facebook\.com(\/|$)/i.test(u)) return false; // business.facebook.com is a Page surface
    if (NOT_A_TIMELINE.test(u)) return false;
    try { if (PROFILE_SUBTAB.test(new URL(u).pathname)) return false; } catch (e) { return false; }
    return true;
  }
  // A stable account key when the url names one: "id:<numeric>" for profile.php?id=, "vanity:<name>"
  // for facebook.com/<name>; "" for /me and the home feed, which resolve to whoever is logged in.
  function accountKeyFrom(u) {
    try {
      var url = new URL(String(u || ""));
      var path = url.pathname.replace(/\/+$/, "");
      if (/^\/profile\.php$/i.test(path)) { var id = url.searchParams.get("id"); return id ? "id:" + id : ""; }
      var m = path.match(/^\/([A-Za-z0-9.\-_]+)$/);
      if (!m || /^(me|home\.php|profile\.php)$/i.test(m[1])) return "";
      return "vanity:" + m[1].toLowerCase();
    } catch (e) { return ""; }
  }
  // { el, label } for the own-timeline trigger; { other: label } when the page's composer
  // belongs to somebody else's timeline; {} when there is no composer at all.
  function findOwnComposerTrigger() {
    var els = document.querySelectorAll('[role="button"], [role="textbox"], div[tabindex]');
    var other = "";
    for (var i = 0; i < els.length; i++) {
      var lbl = norm(els[i].getAttribute("aria-label") || els[i].innerText || "");
      var r = els[i].getBoundingClientRect();
      if (r.width <= 0 || r.height <= 0 || !lbl) continue;
      if (OWN_TRIGGER.test(lbl)) return { el: els[i], label: lbl.slice(0, 80) };
      if (!other && OTHER_TIMELINE_TRIGGER.test(lbl)) other = lbl.slice(0, 80);
    }
    return other ? { other: other } : {};
  }
  // Exact match on the label (trailing punctuation dropped), or the "<…> with <name>" tail an
  // aria-label such as "Edit privacy. Sharing with Public." carries. A qualified row —
  // "Friends except…", "Only show to…", "Custom" — is a different sharing rule, never the base.
  var AUDIENCE_QUALIFIER = /(except|only show|specific|custom|trừ|cụ thể|tùy chỉnh)/i;
  function audienceKey(lbl) {
    lbl = lower(lbl).replace(/[.:;,!…\s]+$/g, "");
    if (!lbl || AUDIENCE_QUALIFIER.test(lbl)) return "";
    var tail = lbl.match(/(?:^|\s)(?:with|với)\s+(.+)$/);
    var cands = [lbl];
    if (tail) cands.push(tail[1].replace(/[.:;,!…\s]+$/g, ""));
    for (var c = 0; c < cands.length; c++) {
      for (var k in AUDIENCE) {
        for (var i = 0; i < AUDIENCE[k].length; i++) { if (cands[c] === AUDIENCE[k][i]) return k; }
      }
    }
    return "";
  }
  // The composer's audience selector: a button whose aria-label describes the sharing
  // setting, or whose own text is exactly an audience name.
  function findAudienceButton(dialog) {
    var btns = dialog.querySelectorAll('[role="button"], button');
    for (var i = 0; i < btns.length; i++) {
      var aria = norm(btns[i].getAttribute("aria-label") || "");
      var txt = norm(btns[i].innerText || "");
      var r = btns[i].getBoundingClientRect();
      if (r.width <= 0 || r.height <= 0) continue;
      if (AUDIENCE_CONTROL.test(aria) || audienceKey(txt) || audienceKey(aria)) return btns[i];
    }
    return null;
  }
  function currentAudience(btn) {
    if (!btn) return { key: "", raw: "" };
    var aria = norm(btn.getAttribute("aria-label") || ""), txt = norm(btn.innerText || "");
    var raw = txt || aria;
    return { key: audienceKey(txt) || audienceKey(aria), raw: raw.slice(0, 80) };
  }
  // The audience picker: whichever visible dialog holds radio rows — a second dialog, or the
  // composer itself when Facebook swaps its content for the "Post audience" panel in place.
  function findAudiencePicker() {
    var dialogs = document.querySelectorAll('[role="dialog"]');
    for (var i = 0; i < dialogs.length; i++) {
      var r = dialogs[i].getBoundingClientRect();
      if (r.width <= 0 || r.height <= 0) continue;
      var rows = dialogs[i].querySelectorAll('[role="radio"], [role="menuitemradio"]');
      for (var j = 0; j < rows.length; j++) { var rr = rows[j].getBoundingClientRect(); if (rr.width > 0 && rr.height > 0) return dialogs[i]; }
    }
    return null;
  }
  // The picker's main rows (Public / Friends / Only me) are not role=radio on the measured
  // page — only the sub-options ("Don't show to…", "Only show to…", "Custom") are — so a row
  // is matched by its FIRST LINE of text across every clickable shape Facebook uses.
  var OPTION_SELECTOR = '[role="radio"], [role="menuitemradio"], [role="option"], [role="menuitem"], [aria-checked], input[type="radio"], label, [role="button"], [tabindex]';
  function optionLabel(el) {
    var lbl = norm(el.getAttribute("aria-label") || "");
    if (!lbl) lbl = norm((el.innerText || "").split("\n")[0]);
    if (!lbl && el.labels && el.labels[0]) lbl = norm((el.labels[0].innerText || "").split("\n")[0]);
    if (!lbl && el.closest && el.closest("label")) lbl = norm((el.closest("label").innerText || "").split("\n")[0]);
    return lbl;
  }
  function findAudienceOption(picker, want) {
    var rows = picker.querySelectorAll(OPTION_SELECTOR);
    for (var i = 0; i < rows.length; i++) {
      var r = rows[i].getBoundingClientRect();
      if (r.width <= 0 || r.height <= 0) continue;
      if (audienceKey(optionLabel(rows[i])) !== want) continue;
      // click the labelled wrapper of a native radio, the element itself otherwise
      return (rows[i].tagName === "INPUT" && rows[i].closest && rows[i].closest("label")) || rows[i];
    }
    return null;
  }
  function optionRows(picker, max) {
    var out = [], rows = picker.querySelectorAll(OPTION_SELECTOR);
    for (var i = 0; i < rows.length && out.length < (max || 16); i++) {
      var r = rows[i].getBoundingClientRect();
      if (r.width <= 0 || r.height <= 0) continue;
      var lbl = optionLabel(rows[i]);
      if (lbl) out.push((rows[i].getAttribute("role") || rows[i].tagName.toLowerCase()) + ":" + lbl.slice(0, 40));
    }
    return out;
  }
  function findDoneButton(picker) {
    var btns = picker.querySelectorAll('[role="button"], button');
    for (var i = 0; i < btns.length; i++) {
      var lbl = norm(btns[i].getAttribute("aria-label") || btns[i].innerText || "");
      var r = btns[i].getBoundingClientRect();
      if (DONE_BUTTON.test(lbl) && r.width > 0 && r.height > 0) return btns[i];
    }
    return null;
  }
  async function setAudience(dialog, want) {
    var btn = findAudienceButton(dialog);
    if (!btn) return { applied: false, reason: "audience_control_not_found", before: null, after: null };
    var before = currentAudience(btn);
    if (before.key === want) return { applied: true, changed: false, before: before.raw, after: before.raw };
    click(btn);
    await waitFor(function () { return !!findAudiencePicker(); }, 6000, 300);
    var picker = findAudiencePicker();
    var seen = { dialogs: [] };
    try {
      var ds = document.querySelectorAll('[role="dialog"]');
      for (var i = 0; i < ds.length && seen.dialogs.length < 4; i++) {
        var rr = ds[i].getBoundingClientRect(); if (rr.width <= 0 || rr.height <= 0) continue;
        var rows = ds[i].querySelectorAll('[role="radio"], [role="menuitemradio"]'), rl = [];
        for (var j = 0; j < rows.length && rl.length < 10; j++) rl.push(norm(rows[j].getAttribute("aria-label") || rows[j].innerText || "").slice(0, 40));
        seen.dialogs.push({ label: norm(ds[i].getAttribute("aria-label") || "").slice(0, 40), radios: rl, rows: optionRows(ds[i], 16), buttons: buttonLabels(ds[i], 14) });
      }
    } catch (e) { /* diagnostics only */ }
    if (!picker) return { applied: false, reason: "audience_picker_did_not_open", before: before.raw, after: before.raw, seen: seen };
    var option = findAudienceOption(picker, want);
    if (!option) return { applied: false, reason: "audience_option_not_found", before: before.raw, after: before.raw, seen: seen };
    click(option);
    await waitFor(function () { var d = findDoneButton(picker); return d && !isDisabled(d); }, 4000, 250);
    var done = findDoneButton(picker);
    if (done && !isDisabled(done)) click(done);
    await waitFor(function () { return !findAudiencePicker(); }, 6000, 300);
    // the composer may have been re-rendered: read the audience from the CURRENT dialog
    var cur = findComposerDialogs();
    var after = currentAudience(cur.length === 1 ? findAudienceButton(cur[0]) : findAudienceButton(dialog));
    return { applied: after.key === want, changed: true, before: before.raw, after: after.raw, seen: seen, reason: after.key === want ? "" : "composer still says \"" + after.raw + "\"" };
  }
  // The composer's own mutation answers with the new story: its permalink is the proof a
  // text search of the page cannot give (the feed may render the post below the fold).
  // The mutation's reply nests the story differently across composer variants, so the id and
  // permalink are found by walking the whole document rather than one fixed path.
  function findStoryFields(root) {
    var found = { id: "", url: "" };
    var walk = function (o, depth) {
      if (!o || typeof o !== "object" || depth > 12 || (found.id && found.url)) return;
      if (Array.isArray(o)) { for (var i = 0; i < o.length; i++) walk(o[i], depth + 1); return; }
      if (!found.id && /^\d{6,}$/.test(String(o.post_id || ""))) found.id = String(o.post_id);
      if (!found.id && /^\d{6,}$/.test(String(o.legacy_story_hideable_id || ""))) found.id = String(o.legacy_story_hideable_id);
      if (!found.url && typeof o.url === "string" && /facebook\.com\/.+\/(posts|videos|reel)\//i.test(o.url)) found.url = o.url;
      if (!found.url && typeof o.permalink_url === "string" && /facebook\.com\//i.test(o.permalink_url)) found.url = o.permalink_url;
      for (var k in o) { if (o[k] && typeof o[k] === "object") walk(o[k], depth + 1); }
    };
    walk(root, 0);
    return found;
  }
  // Key shape of a document, depth-limited — repair evidence when the walk finds nothing.
  function shapeOf(o, depth) {
    depth = depth || 0;
    if (!o || typeof o !== "object") return typeof o;
    if (Array.isArray(o)) return ["[" + o.length + "]", o.length ? shapeOf(o[0], depth + 1) : null];
    if (depth >= 5) return "{…}";
    var r = {}, keys = Object.keys(o);
    for (var i = 0; i < keys.length && i < 25; i++) r[keys[i]] = shapeOf(o[keys[i]], depth + 1);
    return r;
  }
  function storyCreatedSince(t0) {
    try {
      var caps = (window.__soloGql && window.__soloGql.captures) || [];
      for (var i = caps.length - 1; i >= 0; i--) {
        var c = caps[i];
        if (!c || (c.capturedAt || 0) < t0 - 2000 || !/StoryCreate/i.test(String(c.queryName || ""))) continue;
        var f = findStoryFields(c.response);
        if (f.id || f.url) return { id: f.id, url: f.url || (f.id ? "https://www.facebook.com/" + f.id : ""), query: String(c.queryName || "") };
        return { id: "", url: "", query: String(c.queryName || ""), shape: shapeOf(c.response) };
      }
    } catch (e) { /* proof is optional */ }
    return null;
  }

  async function doProfilePost(inputs) {
    var text = String(inputs.text || inputs.message || "").trim();
    if (!text) return wrapCap("fb.profile.post", "error", { error: "no post text provided" });
    var want = lower(inputs.audience || "").replace(/[\s-]+/g, "_");
    if (want && !AUDIENCE[want]) return wrapCap("fb.profile.post", "error", { text: text, error: "unknown audience \"" + inputs.audience + "\" — use public, friends or only_me" });

    // Guard 0: the url shape. A group url is fb.group.post's job; a permalink or media page
    // opens a comment box, not the timeline composer.
    var jobUrl = String(inputs._target_url || location.href);
    if (!isTimelineUrl(jobUrl) || !isTimelineUrl(location.href)) {
      return wrapCap("fb.profile.post", "error", {
        text: text, job_url: jobUrl, landed_url: location.href,
        error: "not_a_timeline_url: open the operator's own profile root (https://www.facebook.com/me) or the home feed — a group url is fb.group.post, a permalink or media url opens a different composer"
      });
    }

    // Guard 0b: when both urls name an account, they must name the SAME one (a friend's vanity
    // is as timeline-shaped as /me). /me and the home feed carry no name and skip this.
    var wantKey = accountKeyFrom(jobUrl), hereKey = accountKeyFrom(location.href);
    if (wantKey && hereKey && wantKey !== hereKey) {
      return wrapCap("fb.profile.post", "error", {
        text: text, job_url: jobUrl, landed_url: location.href,
        error: "profile_mismatch: asked for " + wantKey + " but landed on " + hereKey + " — nothing was typed"
      });
    }

    var dialogs = findComposerDialogs();
    var openedDialog = false, triggerLabel = "";
    if (!dialogs.length) {
      var t = findOwnComposerTrigger();
      // Guard 1: the composer must be the OWN-timeline one. Somebody else's profile offers a
      // "Write something to <Name>…" box that would publish on their timeline.
      if (t.other) {
        return wrapCap("fb.profile.post", "error", {
          text: text, landed_url: location.href, composer_label: t.other,
          error: "other_timeline: this page's composer says \"" + t.other + "\" — that is somebody else's timeline; nothing was typed"
        });
      }
      if (!t.el) {
        return wrapCap("fb.profile.post", "error", {
          text: text, landed_url: location.href,
          error: "no own-timeline composer trigger on this page (\"What's on your mind\") — open https://www.facebook.com/me or the home feed"
        });
      }
      triggerLabel = t.label;
      click(t.el);
      openedDialog = true;
      await waitFor(function () { return findComposerDialogs().length > 0; }, 8000, 300);
      dialogs = findComposerDialogs();
    }
    // Guard 2: exactly ONE composer dialog, same rule as the group and DM paths.
    if (dialogs.length !== 1) {
      return wrapCap("fb.profile.post", "error", {
        text: text, dialogs_found: dialogs.length, opened_dialog: openedDialog,
        error: dialogs.length === 0 ? "the composer dialog did not open" : "ambiguous_composer: " + dialogs.length + " composer dialogs are open — nothing was typed"
      });
    }
    var dialog = dialogs[0];
    var box = dialog.querySelector('div[contenteditable="true"][role="textbox"]');
    if (!box) return wrapCap("fb.profile.post", "error", { text: text, error: "composer dialog has no text box" });
    // The text box renders before the buttons: let the dialog settle before reading it.
    await waitFor(function () { return !!(findPostButton(dialog) || findNextButton(dialog)); }, 4000, 250);

    var audienceBtn = findAudienceButton(dialog);
    var current = audienceBtn ? currentAudience(audienceBtn) : { key: "", raw: "" };
    var submit = findPostButton(dialog) ? "Post" : (findNextButton(dialog) ? "Next" : null);
    if (inputs.dry_run) {
      // Measured 2026-09-10: pressing Done in the audience picker PERSISTS the choice as the
      // account's default audience even when nothing is posted. So a dry run leaves the picker
      // alone unless probe_audience is set explicitly — and then says what it did.
      var probe = (want && inputs.probe_audience === true) ? await setAudience(dialog, want) : null;
      return wrapCap("fb.profile.post", "dry_run", {
        text: text, profile_url: location.href, opened_dialog: openedDialog, trigger: triggerLabel || null,
        post_button_found: !!submit, submit_button: submit, seen_buttons: buttonLabels(dialog, 30),
        audience_control_found: !!audienceBtn, audience_current: current.raw || null, audience_requested: want || null,
        audience_applied: probe ? !!probe.applied : null, audience_after: probe ? probe.after || null : null,
        audience_reason: probe ? probe.reason || null : null, audience_seen: probe ? probe.seen || null : null,
        audience_probe_changed_default: probe ? !!probe.changed : false
      });
    }

    // The audience is set BEFORE any text goes in: when it cannot be applied nothing is
    // typed and nothing is published — an "only me" test post must never go out to friends.
    var aud = null;
    if (want) {
      aud = await setAudience(dialog, want);
      if (!aud.applied) {
        return wrapCap("fb.profile.post", "error", {
          text: text, profile_url: location.href, audience_requested: want, audience_before: aud.before, audience_after: aud.after, audience_seen: aud.seen || null,
          error: "audience_not_applied: " + aud.reason + " — nothing was typed"
        });
      }
      // the picker may have re-rendered the composer: re-resolve the dialog and its text box
      dialogs = findComposerDialogs();
      if (dialogs.length !== 1) return wrapCap("fb.profile.post", "error", { text: text, profile_url: location.href, dialogs_found: dialogs.length, error: "the composer was not the single open dialog after the audience switch — nothing was typed" });
      dialog = dialogs[0];
      box = dialog.querySelector('div[contenteditable="true"][role="textbox"]');
      if (!box) return wrapCap("fb.profile.post", "error", { text: text, profile_url: location.href, error: "composer lost its text box after the audience switch — nothing was typed" });
    }

    await jitter();
    await typeInto(box, text);
    if (!composerText(box)) return wrapCap("fb.profile.post", "error", { text: text, profile_url: location.href, error: "failed to enter text into the composer" });

    // Two-step flow: "Next" first (enabled once text is in), then the "Post" screen.
    var steps = [];
    var btn = findPostButton(dialog);
    if (!btn) {
      var next = findNextButton(dialog);
      if (next && !isDisabled(next)) {
        await jitter();
        click(next);
        steps.push("Next");
        await waitFor(function () { var ds = composerLikeDialogs(); for (var i = 0; i < ds.length; i++) { if (findPostButton(ds[i])) return true; } return false; }, 8000, 300);
        var ds2 = composerLikeDialogs();
        for (var k = 0; k < ds2.length && !btn; k++) btn = findPostButton(ds2[k]);
      }
    }
    if (!btn) {
      return wrapCap("fb.profile.post", "error", {
        text: text, profile_url: location.href, typed: true, steps: steps, seen_buttons: buttonLabels(dialog, 30),
        error: steps.length ? "no Post button on the screen after Next — text was typed but NOT submitted" : "no enabled Post button in the composer — text was typed but NOT submitted"
      });
    }
    steps.push("Post");
    var t0 = Date.now();
    await jitter();
    click(btn);

    // Proof: every composer-like dialog closed (the submit was consumed) AND either the text
    // is on the page or the composer's own mutation answered with the new story.
    var closed = await waitFor(function () { return composerLikeDialogs().length === 0; }, 12000, 500);
    var probe = text.slice(0, 40);
    var appeared = await waitFor(function () {
      try { return (document.body.innerText || "").indexOf(probe) > -1; } catch (e) { return false; }
    }, 8000, 500);
    var created = storyCreatedSince(t0);
    var createdProof = !!(created && (created.id || created.url));
    var status = (closed && (appeared || createdProof)) ? "done" : "error";
    // The composer's own queries after the submit — repair evidence when the story proof is
    // missing (which mutation fired, under which name).
    var afterSubmit = [];
    try {
      var caps = (window.__soloGql && window.__soloGql.captures) || [];
      for (var ci = caps.length - 1; ci >= 0 && afterSubmit.length < 10; ci--) { if (caps[ci] && (caps[ci].capturedAt || 0) >= t0 - 2000) afterSubmit.push(String(caps[ci].queryName || caps[ci].docId || "")); }
    } catch (e) { /* diagnostics only */ }
    // Our audience switch is now the account's DEFAULT for the operator's own next post.
    // Put it back unless told not to: reopen the composer, pick the original, close it.
    // Done persisted our switch the moment it was confirmed, whatever became of the post — so the
    // restore runs on every outcome, and says whether the composer it reopened closed again.
    var restore = { attempted: false, restored: null, to: null, reason: null, closed: null };
    if (aud && aud.changed && inputs.restore_audience !== false) {
      var originalKey = audienceKey(aud.before || "");
      restore.attempted = true; restore.to = aud.before || null;
      if (!originalKey) {
        restore.restored = false; restore.reason = "original audience \"" + (aud.before || "") + "\" is not one of public/friends/only_me";
      } else {
        try {
          var t2 = findOwnComposerTrigger();
          if (t2.el) {
            click(t2.el);
            await waitFor(function () { return findComposerDialogs().length > 0; }, 8000, 300);
            var d2 = findComposerDialogs();
            if (d2.length === 1) {
              await waitFor(function () { return !!findAudienceButton(d2[0]); }, 4000, 250);
              var back = await setAudience(d2[0], originalKey);
              restore.restored = !!back.applied; restore.reason = back.reason || null;
              var closeBtn = null, cbs = d2[0].querySelectorAll('[role="button"], button');
              for (var bi = 0; bi < cbs.length; bi++) { if (/^(close composer|close|đóng)/i.test(norm(cbs[bi].getAttribute("aria-label") || ""))) { closeBtn = cbs[bi]; break; } }
              if (closeBtn) click(closeBtn);
              restore.closed = !!(await waitFor(function () { return composerLikeDialogs().length === 0; }, 6000, 300));
              if (!closeBtn) restore.reason = (restore.reason ? restore.reason + "; " : "") + "no close button on the reopened composer";
            } else { restore.restored = false; restore.reason = "composer did not reopen as a single dialog"; }
          } else { restore.restored = false; restore.reason = "own-timeline trigger not found after posting"; }
        } catch (e) { restore.restored = false; restore.reason = String(e && e.message || e); }
      }
    }
    return wrapCap("fb.profile.post", status, {
      text: text, profile_url: location.href,
      verified: status === "done", closed: !!closed, appeared: !!appeared, opened_dialog: openedDialog, steps: steps,
      audience_requested: want || null,
      audience_before: aud ? aud.before : (current.raw || null),
      audience_after: aud ? aud.after : (current.raw || null),
      post_url: created ? created.url || null : null, post_id: created ? created.id || null : null,
      story_query: created ? created.query || null : null, story_create_shape: created && !createdProof ? created.shape || null : null,
      audience_control_found: !!audienceBtn,
      queries_after_submit: afterSubmit,
      audience_restored: restore.attempted ? restore.restored : null, audience_restore_to: restore.to, audience_restore_reason: restore.reason, audience_restore_closed: restore.closed,
      error: status === "error"
        ? (closed ? "the composer closed but the post did not appear" : "the composer did not close — the post may not have been submitted")
        : null
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
  // Exposed for the offline harness. The composer matcher is the piece that has been wrong three
  // times, and driving it through a full fake post DOM tests the plumbing rather than the rule —
  // this makes the rule itself assertable against the exact labels a live page produced.
  window.__soloActFindCommentBox = findCommentBox;
  // Exposed for tests/test_gql_actions.js only.
  window.__soloActInternals = { audienceKey: audienceKey, isTimelineUrl: isTimelineUrl, accountKeyFrom: accountKeyFrom, findStoryFields: findStoryFields };

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
      if (capId === "fb.group.post") return await doGroupPost(inputs);
      if (capId === "fb.profile.post") return await doProfilePost(inputs);
      return { available: false, capability: capId, count: 0, items: [{ status: "error", error: "unknown or unimplemented action: " + capId }], _debug: { href: location.href } };
    } catch (e) {
      return { available: false, capability: capId, count: 0, items: [{ status: "error", error: String(e && e.message || e) }], _debug: { href: location.href, error: String(e) } };
    }
  };
})();
