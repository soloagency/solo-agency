// Offline harness for the three scope guards added to chrome-extension/filtering.js on
// 2026-09-18, after a live run wrote the operator's private Messenger conversation into a
// client's harvest file and made an x.com link shared in that chat the record's profile_url.
//
// The page fixtures below are the shape of the real failure: a Facebook group page whose
// [role=main] holds the group's posts while the Messenger drawer — chat transcript, read
// receipts and an x.com share card — sits OUTSIDE main, exactly where it sits on facebook.com.
//
// Each guard is tested ALONE as well as together, because they exist to cover each other:
// a Facebook redesign that moves [role=main] must not restore the leak, and a market whose
// aria-labels are translated must not weaken any of them.
//
// Run:  node solo-agency-collector/tests/test_filtering_scope.js
const path = require("path");
const F = require(path.join(__dirname, "..", "chrome-extension", "filtering.js"));

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? " — " + JSON.stringify(detail).slice(0, 400) : "")); }
}

const X_POST = "https://x.com/PrismML/status/2100692248480596348";
const X_ACCOUNT = "https://x.com/PrismML";

// The Messenger drawer as facebook.com renders it: a thread list of /messages/t/ permalinks,
// a conversation transcript, and a shared x.com link with its preview card.
const CHAT_HTML = `
  <div role="dialog" aria-label="Messenger">
    <a href="/messages/t/">Chats</a>
    <a href="/messages/t/8869945636392752/">Vo</a>
    <a href="/messages/e2ee/t/7668669486530168/">Le</a>
    <a href="/messages/t/1025773680625528/">Lan</a>
    <div>
      <span>At Monday 4:12pm, Vo: Check out our latest work - evaluating s2s audio models</span>
      <span>Seen by Cong Trinh at Monday 7:22pm</span>
      <span>Seen by Thanh Binh at Monday 8:25pm</span>
      <a href="${X_POST}">PrismML (@PrismML) on X</a>
      <a href="${X_ACCOUNT}">@PrismML</a>
    </div>
  </div>`;

const GROUP_POST_HTML = `
  <div role="feed">
    <div role="article">
      <a href="/keith.turley">Keith Turley</a>
      <a href="/groups/928140707312134/posts/27984725174560321/">2 h</a>
      <div>Anyone have some worker comp leads? I am looking for referrals in Orange County.</div>
    </div>
  </div>`;

const FB_URL = "https://www.facebook.com/groups/928140707312134";

function pageWithMain() {
  return `<body><nav><a href="/">Home</a></nav>${CHAT_HTML}<div role="main">${GROUP_POST_HTML}</div></body>`;
}
// Same page with neither landmark — what a Facebook restructure would look like.
function pageWithoutMain() {
  return `<body>${CHAT_HTML}${GROUP_POST_HTML.replace(' role="feed"', "")}</body>`;
}
function textOf(result) { return String(result.text || "") + "\n" + String(result.meaningfulText || ""); }
function allUrls(result) {
  return []
    .concat(result.links?.accounts || [], result.links?.posts || [], result.links?.items || [])
    .concat((result.posts || []).flatMap((p) => [p.postUrl, p.accountUrl].filter(Boolean)));
}

console.log("filtering.js scope guards");

// --- guard 1 + 2 + 3 together: the real page shape ------------------------------------------
{
  const r = F.filterSocialHtml(pageWithMain(), { currentUrl: FB_URL });
  check("reads the content landmark, not the whole body", r.meta.contentRoot === "role_main", r.meta);
  check("page is treated as facebook", r.meta.platformGroup === "facebook", r.meta);
  check("no x.com url reaches any link list", !allUrls(r).some((u) => String(u).includes("x.com")), allUrls(r));
  check("no x.com url reaches the text", !textOf(r).includes("x.com"), textOf(r).slice(0, 200));
  check("no private message text is captured", !/Seen by |At Monday 4:12pm/.test(textOf(r)), textOf(r).slice(0, 300));
  check("no recipient name from the chat is captured", !/Cong Trinh|Thanh Binh/.test(textOf(r)));
  check("the real group post IS still captured", /worker comp leads/.test(textOf(r)), textOf(r).slice(0, 300));
  check("the real post url IS still captured", allUrls(r).some((u) => String(u).includes("/groups/928140707312134/posts/")), allUrls(r));
}

// --- guard 2 + 3 alone: Facebook moved [role=main] away -------------------------------------
{
  const r = F.filterSocialHtml(pageWithoutMain(), { currentUrl: FB_URL });
  check("without a landmark the scan degrades to the body, not to nothing", r.meta.contentRoot === "body_fallback", r.meta);
  check("body fallback still captures the real post", /worker comp leads/.test(textOf(r)), textOf(r).slice(0, 300));
  check("body fallback: x.com still never becomes a post/account", !allUrls(r).some((u) => String(u).includes("x.com")), allUrls(r));
  check("body fallback: the chat surface is still skipped", r.meta.chatSurfacesSkipped >= 1, r.meta);
  check("body fallback: no private message text", !/Seen by |At Monday 4:12pm/.test(textOf(r)), textOf(r).slice(0, 300));
}

// --- guard 2 alone: an off-platform link inside a genuine post -------------------------------
{
  const html = `<body><div role="main"><div role="article">
      <a href="/karen.boyer">Karen Boyer</a>
      <a href="/groups/2501016683509797/posts/123456789/">3 h</a>
      <div>Great thread on this, worth a read</div>
      <a href="${X_POST}">x.com</a>
    </div></div></body>`;
  const r = F.filterSocialHtml(html, { currentUrl: FB_URL });
  check("an x.com link shared INSIDE a real post is not treated as the post", !allUrls(r).some((u) => String(u).includes("x.com")), allUrls(r));
  check("the post it was shared in is still captured", allUrls(r).some((u) => String(u).includes("/groups/2501016683509797/posts/")), allUrls(r));
}

// --- the gate must stay OFF on a page that belongs to no platform ----------------------------
// web.search and the discovery pass read a search-results page precisely to find links INTO
// the platforms, so a result page must keep collecting both. One link per page: two bare links
// in one container are merged into a single post by choosePostContainers, before this change
// as well as after, and that merge would hide which of the two the gate let through.
{
  const searchUrl = "https://www.google.com/search?q=realtor";
  // The shape of a real result — heading link, visible url, snippet. A bare anchor with no text
  // around it is below the container scorer's bar and yields nothing, before this change as well
  // as after, so it would prove nothing about the gate.
  const page = (href, label) => `<body><div role="main"><div><h3><a href="${href}">${label}</a></h3>` +
    `<cite>${href}</cite><div>Vietnamese realtors in Orange County discuss referrals, ` +
    `commission splits and open house tips every week.</div></div></div></body>`;
  const rf = F.filterSocialHtml(page("https://www.facebook.com/groups/x/posts/999/", "A facebook post"), { currentUrl: searchUrl });
  const rx = F.filterSocialHtml(page(X_POST, "An x post"), { currentUrl: searchUrl });
  check("a search page belongs to no platform", rf.meta.platformGroup === "", rf.meta);
  check("web.search / discovery still collects facebook links", JSON.stringify(rf.links).includes("facebook.com"), rf.links);
  check("web.search / discovery still collects x links", JSON.stringify(rx.links).includes("x.com"), rx.links);
}

// --- reading x.com itself must keep working --------------------------------------------------
{
  const html = `<body><div role="main"><div role="article">
      <a href="https://x.com/recap_david">Recap David</a>
      <a href="https://x.com/recap_david/status/2067031199063617677">2h</a>
      <div>Mortgage rates moved again this week, here is what it means</div>
    </div></div></body>`;
  const r = F.filterSocialHtml(html, { currentUrl: "https://x.com/recap_david" });
  check("on x.com the page is treated as x", r.meta.platformGroup === "x", r.meta);
  check("on x.com its own posts are still captured", allUrls(r).some((u) => String(u).includes("/status/2067031199063617677")), allUrls(r));
}

// --- an empty landmark must not cost the capture ---------------------------------------------
{
  const html = `<body><div role="main"></div>${GROUP_POST_HTML}</body>`;
  const r = F.filterSocialHtml(html, { currentUrl: FB_URL });
  check("an empty landmark falls back to the body", r.meta.contentRoot === "landmark_empty_fallback", r.meta);
  check("and the post is still captured", /worker comp leads/.test(textOf(r)), textOf(r).slice(0, 200));
}

// --- platformGroupOf ---------------------------------------------------------------------------
{
  const g = F.platformGroupOf;
  check("platformGroupOf: www.facebook.com", g("www.facebook.com") === "facebook");
  check("platformGroupOf: m.facebook.com", g("m.facebook.com") === "facebook");
  check("platformGroupOf: fb.watch", g("fb.watch") === "facebook");
  check("platformGroupOf: x.com", g("x.com") === "x");
  check("platformGroupOf: twitter.com", g("twitter.com") === "x");
  check("platformGroupOf: full url", g("https://www.instagram.com/p/abc/") === "instagram");
  check("platformGroupOf: unknown host is no platform", g("www.zillow.com") === "");
  check("platformGroupOf: empty", g("") === "");
  // the lookalike domain a link shim or a phishing page would use
  check("platformGroupOf: facebook.com.evil.example is not facebook", g("facebook.com.evil.example") !== "facebook", g("facebook.com.evil.example"));
}

// --- guards must not leak between calls ---------------------------------------------------------
{
  F.filterSocialHtml(pageWithMain(), { currentUrl: FB_URL });
  check("isPostUrl is unguarded again after a call", F.isPostUrl(X_POST) === true);
  check("isAccountUrl is unguarded again after a call", F.isAccountUrl(X_ACCOUNT) === true);
}

console.log("\n" + pass + " passed, " + fail + " failed");
process.exit(fail ? 1 : 0);
