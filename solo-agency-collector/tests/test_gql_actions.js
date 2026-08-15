// Offline harness for gql_actions.js: exercises the pure decision logic (content
// matching, dedupe, host allowlist, thread-url + title guards) in a fake DOM, so the
// browser round-trip only has to prove the DOM parts.
const fs = require("fs");
const vm = require("vm");

const path = require("path");
const SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "gql_actions.js"), "utf8");

function fakeEl(attrs, opts) {
  opts = opts || {};
  return {
    _attrs: attrs || {},
    innerText: opts.innerText || "",
    getAttribute(k) { return this._attrs[k] === undefined ? null : this._attrs[k]; },
    getBoundingClientRect() { return { width: opts.hidden ? 0 : 100, height: opts.hidden ? 0 : 20 }; },
    focus() {}, scrollIntoView() {}, dispatchEvent() { return true; },
    closest() { return this; }, matches() { return false; }, querySelector() { return null; }
  };
}

function makeCtx(opts) {
  opts = opts || {};
  const win = {};
  const href = opts.href || "https://www.facebook.com/groups/668676178569386";
  const ctx = {
    window: win,
    location: { href, origin: new URL(href).origin },
    document: {
      title: opts.title || "",
      body: { innerText: opts.bodyText || "" },
      querySelectorAll: opts.querySelectorAll || (() => []),
      querySelector: () => null,
      execCommand: () => true
    },
    setTimeout, clearTimeout, URL, console,
    MouseEvent: function () {}, KeyboardEvent: function () {}, InputEvent: function () {}
  };
  ctx.window = win;
  ctx.globalThis = ctx;
  vm.createContext(ctx);
  vm.runInContext(SRC, ctx);
  return ctx;
}

function post(text, url, id) { return { id: id || url, post_id: id || "", url, text, created_time: 1 }; }

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass++; console.log("  ok   " + name); }
  else { fail++; console.log("  FAIL " + name + (detail ? "  -> " + JSON.stringify(detail) : "")); }
}

async function resolveWith(items, inputs, opts) {
  const ctx = makeCtx(opts);
  ctx.window.__soloGqlPaginate = async () => ({ available: items.length > 0, items, reason: items.length ? "" : "no_capture" });
  const r = await ctx.window.__soloActResolve("fb.post.comment", Object.assign({ _target_url: (opts && opts.href) || "https://www.facebook.com/groups/668676178569386" }, inputs));
  return { env: r, item: r.items[0] };
}

(async () => {
  console.log("\n== Task A: content matching ==");

  const G = "https://www.facebook.com/groups/668676178569386/posts/";
  {
    const { env, item } = await resolveWith(
      [post("Post 5 post 5 post 5", G + "676092881161049/"), post("Post 3: something else", G + "669851668451837/"), post("Post 2", G + "111/")],
      { match_text: "Post 5 post 5 post 5" });
    check("exact one match -> resolved", env.status === "resolved", env.status);
    check("resolved url is the right post", item.matched_post && /676092881161049/.test(item.matched_post.url), item.matched_post);
    check("candidates_considered counts all posts read", item.candidates_considered === 3, item.candidates_considered);
    check("record survives background discard (available)", env.available === true);
  }
  {
    // The SAME post arrives in several captures and again in every replayed page.
    const dup = [post("Post 5 post 5 post 5", G + "676092881161049/", "a"),
                 post("Post 5 post 5 post 5", G + "676092881161049", "b"),
                 post("Post 5 post 5 post 5", G + "676092881161049/?comment_id=9", "c")];
    const { env, item } = await resolveWith(dup, { match_text: "post 5" });
    check("duplicates of one post do NOT read as ambiguous", env.status === "resolved", env.status);
    check("dedupe collapses to 1 candidate", item.candidates_considered === 1, item.candidates_considered);
  }
  {
    const { env, item } = await resolveWith(
      [post("Post 5 post 5 post 5", G + "676092881161049/"), post("Post 5 is also here", G + "999/")],
      { match_text: "Post 5" });
    check("two different posts -> ambiguous_match", env.status === "ambiguous_match", env.status);
    check("ambiguous lists the colliding posts", Array.isArray(item.candidates) && item.candidates.length === 2, item.candidates);
    check("ambiguous carries no matched_post (nothing to write to)", !item.matched_post);
  }
  {
    const { env, item } = await resolveWith([post("Post 3: something", G + "1/"), post("Post 2", G + "2/")], { match_text: "Post 5" });
    check("nothing matches -> no_match", env.status === "no_match", env.status);
    check("no_match shows what WAS read", item.candidates.length === 2, item.candidates);
  }
  {
    const { env } = await resolveWith([], { match_text: "Post 5" });
    check("empty listing -> listing_unavailable, not no_match", env.status === "listing_unavailable", env.status);
  }
  {
    const { env, item } = await resolveWith([post("Post 5", "https://l.facebook.com/l.php?u=https://evil.example/")], { match_text: "Post 5" });
    check("l.facebook.com link shim is rejected", env.status === "error", env.status);
    check("rejection names the offending url", /allowlist|permalink/i.test(item.error || ""), item.error);
  }
  {
    const { env } = await resolveWith([post("Post 5", "https://evil.example/x")], { match_text: "Post 5" });
    check("off-platform url is rejected", env.status === "error", env.status);
  }
  {
    const items = [post("Post 5 post 5 post 5", G + "1/"), post("Post 5", G + "2/")];
    const { env, item } = await resolveWith(items, { match_text: "Post 5", match_mode: "exact" });
    check("match_mode exact picks only the identical text", env.status === "resolved" && /\/2\//.test(item.matched_post.url), item.matched_post);
  }
  {
    const { env, item } = await resolveWith([post("Post 5 post 5", G + "1/"), post("Post 3", G + "2/")], { match_text: "^post\\s+5\\b", match_mode: "regex" });
    check("match_mode regex works", env.status === "resolved" && /\/1\//.test(item.matched_post.url), item.matched_post);
  }
  {
    const { env, item } = await resolveWith([post("x", G + "1/")], { match_text: "[unclosed", match_mode: "regex" });
    check("invalid regex refuses instead of throwing", env.status === "no_match", env.status);
  }
  {
    const { item } = await resolveWith([post("Post 5", G + "1/")], { match_text: "Post 5" });
    check("group url -> fb.group.posts listing", item.list_capability === "fb.group.posts", item.list_capability);
  }
  {
    const { item } = await resolveWith([post("Post 5", "https://www.facebook.com/nguyenhuubinh/posts/pfbid06qGgZ")],
      { match_text: "Post 5" }, { href: "https://www.facebook.com/nguyenhuubinh" });
    check("profile url -> fb.profile.posts listing", item.list_capability === "fb.profile.posts", item.list_capability);
    check("pfbid permalink is accepted", /pfbid06qGgZ/.test(item.matched_post.url), item.matched_post);
  }
  {
    const ctx = makeCtx({});
    const r = await ctx.window.__soloActResolve("fb.post.comment", { match_text: "x", _target_url: "https://www.facebook.com/groups/1" });
    check("no gql_extract injected -> explicit error, not a crash", r.status === "error" && /not present/i.test(r.items[0].error), r.items[0]);
  }

  console.log("\n== Task A: DOM merge (posts GraphQL never saw) ==");
  // Facebook server-renders the newest posts; only older pages come over GraphQL.
  // filtering.js hands back {postUrl, content} pairs; background.js relays them as _dom_posts.
  const domPost = (url, text) => ({ url, text });
  {
    const G = "https://www.facebook.com/groups/668676178569386/posts/";
    const ctx = makeCtx({});
    ctx.window.__soloGqlPaginate = async () => ({ available: true, items: [post("Post 2: Post 2", G + "668676911902646/"), post("Post 3: Post 3", G + "669851668451837/")] });
    const r = await ctx.window.__soloActResolve("fb.post.comment", { match_text: "Post 5 post 5 post 5", _target_url: "https://www.facebook.com/groups/668676178569386",
      _dom_posts: [domPost("/groups/668676178569386/posts/676092881161049/?__cft__[0]=AZX", "Post 5 post 5 post 5")] });
    const it = r.items[0];
    check("the newest post, invisible to GraphQL, is now found", r.status === "resolved", r.status);
    check("resolved to the right permalink", it.matched_post && /676092881161049/.test(it.matched_post.url), it.matched_post);
    check("tracking params (__cft__) stripped from the url", it.matched_post && !/__cft__/.test(it.matched_post.url), it.matched_post);
    check("audit shows both sources", it.sources_read && it.sources_read.graphql === 2 && it.sources_read.dom === 1, it.sources_read);
    check("matched_post records it came from the DOM", !!it.matched_post && it.matched_post.from === "dom", it.matched_post);
    check("GraphQL posts still counted", it.candidates_considered === 3, it.candidates_considered);
  }
  {
    // Same post in BOTH sources must not read as two candidates.
    const url = "https://www.facebook.com/groups/1/posts/999/";
    const ctx = makeCtx({});
    ctx.window.__soloGqlPaginate = async () => ({ available: true, items: [post("Post 5 post 5 post 5", url)] });
    const r = await ctx.window.__soloActResolve("fb.post.comment", { match_text: "Post 5", _target_url: "https://www.facebook.com/groups/1",
      _dom_posts: [domPost("/groups/1/posts/999/", "Post 5 post 5 post 5")] });
    check("same post from both sources dedupes to one", r.status === "resolved" && r.items[0].candidates_considered === 1, r.items[0].candidates_considered);
    check("GraphQL record wins the dedupe (richer)", r.items[0].matched_post.from === "graphql", r.items[0].matched_post.from);
  }
  {
    // A needle that only appears in a COMMENT must not retarget the write onto that post.
    const ctx = makeCtx({});
    ctx.window.__soloGqlPaginate = async () => ({ available: true, items: [] });
    const r = await ctx.window.__soloActResolve("fb.post.comment", { match_text: "findmeplease", _target_url: "https://www.facebook.com/groups/1",
      _dom_posts: [domPost("/groups/1/posts/999/", "Totally unrelated body")] });
    check("comment text never reaches the matcher (filtering.js strips it)", r.status === "no_match", r.status);
  }

  {
    const ctx = makeCtx({});
    ctx.window.__soloGqlPaginate = async () => ({ available: true, items: [] });
    const r = await ctx.window.__soloActResolve("fb.post.comment", { match_text: "Post 5", _target_url: "https://www.facebook.com/groups/1",
      _dom_posts: [domPost("https://l.facebook.com/l.php?u=https://evil.example/", "Post 5")] });
    check("host allowlist is re-applied to DOM-supplied urls", r.status === "listing_unavailable", r.status);
  }

  console.log("\n== Task A: fail-safe when orchestration is stale ==");
  {
    const ctx = makeCtx({});
    const r = await ctx.window.__soloActRun("fb.post.comment", { match_text: "Post 5", text: "hi", _target_url: "https://www.facebook.com/groups/1" });
    check("match_text without _resolved_url refuses to write", r.status === "unresolved_match", r.status);
  }
  {
    // Second navigation did not land: still on the listing page.
    const ctx = makeCtx({ href: "https://www.facebook.com/groups/668676178569386" });
    const r = await ctx.window.__soloActRun("fb.post.comment", {
      text: "hi", match_text: "Post 5",
      _resolved_url: "https://www.facebook.com/groups/668676178569386/posts/676092881161049/",
      _target_url: "https://www.facebook.com/groups/668676178569386/posts/676092881161049/"
    });
    check("landed on the listing instead of the post -> redirected", r.status === "redirected", r.status);
  }
  {
    const ctx = makeCtx({ href: "https://www.facebook.com/groups/668676178569386/posts/676092881161049/?comment_id=1" });
    ctx.document.querySelectorAll = () => [];
    const r = await ctx.window.__soloActRun("fb.post.comment", {
      text: "hi", match_text: "Post 5", dry_run: true,
      _resolved_url: "https://www.facebook.com/groups/668676178569386/posts/676092881161049/",
      _target_url: "https://www.facebook.com/groups/668676178569386/posts/676092881161049/"
    });
    check("landed on the right post (query string ignored) -> proceeds", r.status === "dry_run", r.status);
  }

  console.log("\n== Task B: identity anchored to the thread ID, not the name ==");
  const composer = (name) => fakeEl({ "aria-label": "Write to " + name });
  const oneComposer = (name) => (sel) => (String(sel).indexOf("contenteditable") > -1 ? [composer(name)] : []);
  const dm = (ctx, inputs) => ctx.window.__soloActRun("fb.message.send", Object.assign({ text: "hi" }, inputs));

  {
    const u = "https://www.messenger.com/e2ee/t/100026030446486";
    const ctx = makeCtx({ href: u, title: "Bob Nguyen | Messenger", querySelectorAll: oneComposer("Bob Nguyen") });
    const r = await dm(ctx, { recipient_name: "Bob Nguyen", dry_run: true, _target_url: u });
    check("numeric profile id thread accepted", r.status === "dry_run", r.items[0]);
    check("id is verified from the pre-gate url", r.items[0].id_verified === true, r.items[0]);
  }
  {
    // THE case the operator raised: the profile renamed itself since the job was written.
    const u = "https://www.messenger.com/e2ee/t/100026030446486";
    const ctx = makeCtx({ href: u, title: "Alex Nguyen | Messenger", querySelectorAll: oneComposer("Alex Nguyen") });
    const r = await dm(ctx, { recipient_name: "Bob Nguyen", dry_run: true, _target_url: u });
    check("a RENAMED recipient no longer blocks the send", r.status === "dry_run", r.items[0]);
    check("the rename is still recorded, not hidden", r.items[0].name_matched === false && r.items[0].recipient_open === "Alex Nguyen", r.items[0]);
  }
  {
    // Messenger could not resolve the id and fell back to another thread. The url alone
    // cannot prove that (a legitimate rewrite looks the same), so the NAME catches it.
    const ctx = makeCtx({ href: "https://www.messenger.com/e2ee/t/9999999999", title: "Someone Else | Messenger", querySelectorAll: oneComposer("Someone Else") });
    const r = await dm(ctx, { recipient_name: "Bob Nguyen", _target_url: "https://www.messenger.com/e2ee/t/100026030446486" });
    check("silent fallback to another thread is refused", r.status === "error" && /recipient_mismatch/.test(r.items[0].error), r.items[0].error);
    check("the refusal records both ids", r.items[0].requested_id === "100026030446486" && r.items[0].open_id === "9999999999", r.items[0]);
  }
  {
    // Facebook rewrote /messages/t/<profile_id> to /messages/e2ee/t/<thread_id> before this
    // ran, so the id anchor is gone: the NAME must still guard, exactly as it used to.
    const ctx = makeCtx({ href: "https://www.facebook.com/messages/e2ee/t/778899", title: "Messenger", querySelectorAll: oneComposer("Someone Else") });
    const r = await dm(ctx, { recipient_name: "Bob Nguyen", _target_url: "https://www.facebook.com/messages/t/1234" });
    check("with no id anchor, a name mismatch still refuses", r.status === "error" && /recipient_mismatch/.test(r.items[0].error), r.items[0].error);
  }
  {
    const ctx = makeCtx({ href: "https://www.facebook.com/messages/e2ee/t/778899", title: "Messenger", querySelectorAll: oneComposer("Someone Else") });
    const r = await dm(ctx, { _target_url: "https://www.facebook.com/messages/t/1234" });
    check("no id AND no name -> refuse, nothing proves who this is", r.status === "error" && /unverified_recipient/.test(r.items[0].error), r.items[0].error);
  }
  {
    const ctx = makeCtx({ href: "https://www.facebook.com/messages/e2ee/t/778899", title: "Messenger", querySelectorAll: oneComposer("Bob Nguyen") });
    const r = await dm(ctx, { recipient_name: "Bob Nguyen", dry_run: true, _target_url: "https://www.facebook.com/messages/t/1234" });
    check("rewritten url + matching name still sends (old path preserved)", r.status === "dry_run", r.items[0]);
    check("and it is honest that the id was not verified", r.items[0].id_verified === false, r.items[0]);
  }
  {
    const u = "https://www.messenger.com/e2ee/t/thaian.nguyen.731572";
    const ctx = makeCtx({ href: u, title: "Bob Nguyen | Messenger", querySelectorAll: oneComposer("Bob Nguyen") });
    const r = await dm(ctx, { dry_run: true, _target_url: u });
    check("vanity id alone is enough — recipient_name no longer required", r.status === "dry_run", r.items[0]);
  }
  {
    const ctx = makeCtx({ href: "https://www.facebook.com/nguyenhuubinh", title: "Bob Nguyen | Facebook", querySelectorAll: oneComposer("Bob Nguyen") });
    const r = await dm(ctx, { recipient_name: "Bob Nguyen", _target_url: "https://www.facebook.com/nguyenhuubinh" });
    check("profile url rejected up front", r.status === "error" && /not_a_thread_url/.test(r.items[0].error), r.items[0].error);
  }
  {
    const u = "https://www.messenger.com/e2ee/t/100026030446486";
    const two = (sel) => (String(sel).indexOf("contenteditable") > -1 ? [composer("Bob Nguyen"), composer("Someone Else")] : []);
    const ctx = makeCtx({ href: u, title: "Bob Nguyen | Messenger", querySelectorAll: two });
    const r = await dm(ctx, { recipient_name: "Bob Nguyen", _target_url: u });
    check("two composers -> ambiguous_composer (unchanged)", /ambiguous_composer/.test(r.items[0].error || ""), r.items[0].error);
  }

  console.log("\n" + (fail === 0 ? "ALL " + pass + " CHECKS PASSED" : pass + " passed, " + fail + " FAILED"));
  process.exit(fail === 0 ? 0 : 1);
})();
