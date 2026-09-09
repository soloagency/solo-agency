// Offline harness for chrome-extension/core/schema.js — the canonical record contract
// (profile/post/comment/group/message) every platform module normalizes into.
//
// Loads the source with vm.runInContext against a fake self/window (same trick as
// tests/test_zillow_extract.js), exactly like a classic MV3 service worker importScripts
// would see it — no require(), no ES modules.
//
// Run:  node solo-agency-collector/tests/test_schema.js
const fs = require("fs");
const path = require("path");
const vm = require("vm");

const SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "core", "schema.js"), "utf8");

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail) : "")); }
}

function loadSchema() {
  const ctx = {};
  ctx.self = ctx;
  ctx.window = ctx;
  ctx.globalThis = ctx;
  ctx.console = console;
  vm.createContext(ctx);
  vm.runInContext(SRC, ctx, { filename: "schema.js" });
  return ctx.SoloSchema;
}

function clone(o) { return JSON.parse(JSON.stringify(o)); }

const S = loadSchema();

// ------------------------------------------------------------------ fixtures
const HAPPY = {
  profile: {
    kind: "profile", platform: "facebook", name: "Jane Realtor", captured_at: "2026-09-01T12:00:00.000Z",
    platform_id: "100012345", url: "https://www.facebook.com/jane.realtor",
    handle: "jane.realtor", bio: "Realtor in LA", category: "Real Estate Agent", industry: "Real Estate",
    location: ["Los Angeles, CA"], work: [{ employer: "Reach Home Loans", position: "Owner", current: true }],
    education: [{ school: "UCLA", degree: "BA" }], emails: ["jane@example.com"], phones: ["+13105551234"],
    websites: ["https://janerealtor.com"], socials: { instagram: "https://instagram.com/jane" },
    photo_url: "https://example.com/p.jpg", verified: true, follower_count: 1200,
    about: { work: "Reach Home Loans" }, extraction_audit: { source: "dom" },
    source_capability: "fb.profile.dossier", refs: {}, ext: {}
  },
  post: {
    kind: "post", platform: "facebook", platform_id: "pfbid02xxxx", captured_at: "2026-09-01T12:00:00.000Z",
    url: "https://www.facebook.com/groups/1/posts/2/", text: "Looking for a realtor in LA",
    actor: { platform_id: "1", name: "Bob", url: "https://www.facebook.com/bob", type: "profile" },
    created_at: "2026-08-30T00:00:00.000Z", platform_time: 1756512000,
    engagement: { likes: 3, comments: 1, shares: 0, views: null, saves: null },
    attachments: [{ type: "photo", url: "https://example.com/a.jpg" }],
    group_ref: { platform_id: "1", name: "Group", url: "https://www.facebook.com/groups/1" },
    media: null, source_capability: "fb.group.posts", refs: {}, ext: {}
  },
  comment: {
    kind: "comment", platform: "facebook", platform_id: "c1",
    post_ref: "ZmVlZGJhY2s6MTIz",
    text: "I can help!", captured_at: "2026-09-01T12:05:00.000Z",
    url: "https://www.facebook.com/groups/1/posts/2/?comment_id=c1",
    actor: { platform_id: "2", name: "Alice", url: "https://www.facebook.com/alice", type: "profile" },
    created_at: "2026-09-01T12:04:00.000Z", platform_time: 1756731840, depth: 0, reply_count: 0,
    replies: [], own_ref: "ZmVlZGJhY2s6NDU2", pagination: {}, source_capability: "fb.post.comments", refs: {}, ext: {}
  },
  group: {
    kind: "group", platform: "facebook", platform_id: "g1", name: "Real Estate LA",
    url: "https://www.facebook.com/groups/g1", captured_at: "2026-09-01T12:00:00.000Z",
    type: "group", member_count: 5000, source_capability: "fb.groups.search", refs: {}, ext: {}
  },
  message: {
    kind: "message", platform: "facebook", direction: "out", channel: "dm", body_text: "Hi there",
    status: "done", captured_at: "2026-09-01T12:10:00.000Z",
    thread_ref: { platform_id: "t1", url: "https://www.messenger.com/e2ee/t/t1" },
    post_ref: null, group_ref: null, verified: true, contact_ref: "c_01",
    source_capability: "fb.message.send", refs: {}, ext: {}
  }
};

// ------------------------------------------------------------------ happy paths + required-field failures
console.log("\n== kinds: happy path + required-field failures ==");
Object.keys(S.ENTITIES).forEach(function (kind) {
  const rec = HAPPY[kind];
  const r = S.validateRecord(rec);
  check(kind + ": happy-path record validates ok (no errors)", r.ok === true && r.errors.length === 0, r.errors);
  check(kind + ": happy-path record has no warnings for a known platform", r.warnings.length === 0, r.warnings);

  S.ENTITIES[kind].required.forEach(function (field) {
    const copy = clone(rec);
    delete copy[field];
    const rr = S.validateRecord(copy);
    check(
      kind + ": missing required '" + field + "' fails validation",
      rr.ok === false && rr.errors.some(function (e) { return e.indexOf(field) !== -1; }),
      rr.errors
    );
  });
});

// ------------------------------------------------------------------ text-or-url / platform_id-or-url rule
console.log("\n== at-least-one rules ==");
{
  const noId = clone(HAPPY.profile);
  delete noId.platform_id;
  delete noId.url;
  const r1 = S.validateRecord(noId);
  check("profile: neither platform_id nor url -> fails", r1.ok === false && r1.errors.some(function (e) { return e.indexOf("platform_id") !== -1 && e.indexOf("url") !== -1; }), r1.errors);

  const urlOnly = clone(HAPPY.profile);
  delete urlOnly.platform_id;
  check("profile: url only -> passes", S.validateRecord(urlOnly).ok === true, S.validateRecord(urlOnly).errors);

  const idOnly = clone(HAPPY.profile);
  delete idOnly.url;
  check("profile: platform_id only -> passes", S.validateRecord(idOnly).ok === true, S.validateRecord(idOnly).errors);
}
{
  const noTextOrUrl = clone(HAPPY.post);
  delete noTextOrUrl.text;
  delete noTextOrUrl.url;
  const r2 = S.validateRecord(noTextOrUrl);
  check("post: neither text nor url -> fails", r2.ok === false && r2.errors.some(function (e) { return e.indexOf("text") !== -1 && e.indexOf("url") !== -1; }), r2.errors);

  const textOnly = clone(HAPPY.post);
  delete textOnly.url;
  check("post: text only -> passes", S.validateRecord(textOnly).ok === true, S.validateRecord(textOnly).errors);

  const urlOnlyPost = clone(HAPPY.post);
  delete urlOnlyPost.text;
  check("post: url only -> passes", S.validateRecord(urlOnlyPost).ok === true, S.validateRecord(urlOnlyPost).errors);

  const blank = clone(HAPPY.post);
  blank.text = "   ";
  blank.url = "";
  const r3 = S.validateRecord(blank);
  check("post: whitespace-only text and empty url -> fails (non-empty rule)", r3.ok === false, r3.errors);
}

// ------------------------------------------------------------------ sensitive keys
console.log("\n== sensitive-key detection ==");
{
  const rec = clone(HAPPY.post);
  rec.ext = { a: { b: { auth_token: "abc123" } } };
  const r = S.validateRecord(rec);
  check(
    "sensitive key 3 levels deep inside ext is caught",
    r.ok === false && r.errors.some(function (e) { return e.indexOf("ext.a.b.auth_token") !== -1; }),
    r.errors
  );

  check(
    "findSensitiveKeys walks arrays too",
    JSON.stringify(S.findSensitiveKeys({ list: [{ ok: 1 }, { session_id: "x" }] })) === JSON.stringify(["list[1].session_id"])
  );

  check("isSensitiveKey matches by substring, case-insensitively", S.isSensitiveKey("Authorization-Header") === true);
  check("isSensitiveKey flags 'author' too (contains 'auth' — the naming trap this schema documents)", S.isSensitiveKey("author") === true);
  check("isSensitiveKey leaves a clean field alone", S.isSensitiveKey("platform_id") === false);

  const clean = clone(HAPPY.post);
  check("a clean record has zero sensitive-key errors", S.validateRecord(clean).errors.length === 0, S.validateRecord(clean).errors);
}

// ------------------------------------------------------------------ toIso
console.log("\n== toIso: seconds vs ms, junk, passthrough ==");
{
  const seconds = 1756731840; // ~2025-09-01, well under the 1e11 threshold
  const ms = 1756731840123;
  check("toIso treats a small number as unix SECONDS", S.toIso(seconds) === new Date(seconds * 1000).toISOString(), S.toIso(seconds));
  check("toIso treats a >=1e11 number as unix MILLISECONDS", S.toIso(ms) === new Date(ms).toISOString(), S.toIso(ms));
  check("toIso sniffs a numeric STRING the same way as the number (seconds)", S.toIso(String(seconds)) === S.toIso(seconds));
  check("toIso sniffs a numeric STRING the same way as the number (ms)", S.toIso(String(ms)) === S.toIso(ms));
  check("toIso passes an already-ISO string through as ISO", S.toIso("2026-09-01T12:00:00.000Z") === new Date("2026-09-01T12:00:00.000Z").toISOString());
  check("toIso returns null for unparseable junk", S.toIso("not-a-date-at-all") === null);
  check("toIso returns null for empty string", S.toIso("") === null);
  check("toIso returns null for null", S.toIso(null) === null);
  check("toIso returns null for undefined", S.toIso(undefined) === null);
  check("toIso returns null for NaN", S.toIso(NaN) === null);
  check("nowIso returns a parseable ISO string", typeof S.nowIso() === "string" && isFinite(Date.parse(S.nowIso())), S.nowIso());
}

// ------------------------------------------------------------------ platform validation
console.log("\n== platform validation ==");
{
  const unknown = clone(HAPPY.profile);
  unknown.platform = "myspace";
  const r = S.validateRecord(unknown);
  check("an unknown-but-well-formed platform WARNS, does not error", r.ok === true && r.warnings.some(function (w) { return w.indexOf("myspace") !== -1; }), r);

  const malformed = clone(HAPPY.profile);
  malformed.platform = "Face Book!";
  const r2 = S.validateRecord(malformed);
  check("a malformed platform id ERRORS", r2.ok === false && r2.errors.some(function (e) { return e.indexOf("platform") !== -1; }), r2.errors);

  const known = clone(HAPPY.profile);
  known.platform = "zillow";
  check("a known platform has no platform warning", S.validateRecord(known).warnings.length === 0, S.validateRecord(known).warnings);
}

// ------------------------------------------------------------------ enum failures
console.log("\n== enum failures ==");
{
  const badGroupType = clone(HAPPY.group);
  badGroupType.type = "not-a-real-type";
  const r1 = S.validateRecord(badGroupType);
  check("group.type outside the enum fails", r1.ok === false && r1.errors.some(function (e) { return e.indexOf("'type'") !== -1; }), r1.errors);

  const okGroupType = clone(HAPPY.group);
  okGroupType.type = "page";
  check("group.type inside the enum passes", S.validateRecord(okGroupType).ok === true, S.validateRecord(okGroupType).errors);

  const badDirection = clone(HAPPY.message);
  badDirection.direction = "sideways";
  const r2 = S.validateRecord(badDirection);
  check("message.direction outside in|out fails", r2.ok === false && r2.errors.some(function (e) { return e.indexOf("'direction'") !== -1; }), r2.errors);

  const badChannel = clone(HAPPY.message);
  badChannel.channel = "carrier-pigeon";
  const r3 = S.validateRecord(badChannel);
  check("message.channel outside the enum fails", r3.ok === false && r3.errors.some(function (e) { return e.indexOf("'channel'") !== -1; }), r3.errors);
}

// ------------------------------------------------------------------ ext/refs must be plain objects
console.log("\n== ext/refs shape ==");
{
  const arrRefs = clone(HAPPY.post);
  arrRefs.refs = ["not", "an", "object"];
  const r1 = S.validateRecord(arrRefs);
  check("refs as an array fails (must be a plain object)", r1.ok === false && r1.errors.some(function (e) { return e.indexOf("'refs'") !== -1; }), r1.errors);

  const strExt = clone(HAPPY.post);
  strExt.ext = "nope";
  const r2 = S.validateRecord(strExt);
  check("ext as a string fails (must be a plain object)", r2.ok === false && r2.errors.some(function (e) { return e.indexOf("'ext'") !== -1; }), r2.errors);

  const nullExtRefs = clone(HAPPY.post);
  nullExtRefs.ext = null;
  nullExtRefs.refs = null;
  check("ext/refs explicitly null are treated as absent, not an error", S.validateRecord(nullExtRefs).ok === true, S.validateRecord(nullExtRefs).errors);
}

// ------------------------------------------------------------------ unknown top-level keys
console.log("\n== unknown fields ==");
{
  const extra = clone(HAPPY.group);
  extra.totally_made_up_field = "x";
  const r = S.validateRecord(extra);
  check("an unrecognised top-level field WARNS, does not error", r.ok === true && r.warnings.some(function (w) { return w.indexOf("totally_made_up_field") !== -1; }), r);
}

// ------------------------------------------------------------------ malformed input / bad kind
console.log("\n== malformed records ==");
{
  check("a non-object record fails cleanly", S.validateRecord("nope").ok === false);
  check("null record fails cleanly", S.validateRecord(null).ok === false);
  const badKind = clone(HAPPY.post);
  badKind.kind = "tweet";
  const r = S.validateRecord(badKind);
  check("an invalid kind fails and names the allowed kinds", r.ok === false && r.errors.some(function (e) { return e.indexOf("tweet") !== -1; }), r.errors);
}

// ------------------------------------------------------------------ captured_at ISO8601
console.log("\n== captured_at ISO8601 ==");
{
  const badTs = clone(HAPPY.post);
  badTs.captured_at = "yesterday";
  const r = S.validateRecord(badTs);
  check("captured_at that doesn't parse as ISO8601 fails", r.ok === false && r.errors.some(function (e) { return e.indexOf("captured_at") !== -1; }), r.errors);

  const noZone = clone(HAPPY.post);
  noZone.captured_at = "2026-09-01T12:00:00"; // no timezone designator
  check("captured_at without a timezone designator fails the strict ISO8601 check", S.validateRecord(noZone).ok === false, S.validateRecord(noZone).errors);
}

// ------------------------------------------------------------------ validateCanonical
console.log("\n== validateCanonical ==");
{
  const good = { schema_version: S.SCHEMA_VERSION, kind: "post", items: [HAPPY.post, HAPPY.post] };
  const r = S.validateCanonical(good);
  check("validateCanonical: two valid items -> ok, item_count 2", r.ok === true && r.item_count === 2, r);

  const bad = { schema_version: S.SCHEMA_VERSION, kind: "post", items: [HAPPY.post, { kind: "post" }] };
  const r2 = S.validateCanonical(bad);
  check("validateCanonical: one bad item -> not ok, error prefixed with its index", r2.ok === false && r2.errors.some(function (e) { return e.indexOf("items[1]:") === 0; }), r2.errors);

  const staleVersion = { schema_version: 0, kind: "post", items: [HAPPY.post] };
  const r3 = S.validateCanonical(staleVersion);
  check("validateCanonical: schema_version mismatch WARNS, does not error", r3.ok === true && r3.warnings.length > 0, r3);

  const notArray = { schema_version: S.SCHEMA_VERSION, kind: "post", items: "nope" };
  const r4 = S.validateCanonical(notArray);
  check("validateCanonical: items not an array -> error, item_count 0", r4.ok === false && r4.item_count === 0, r4);
}

console.log("\n" + pass + " passed, " + fail + " failed");
if (fail > 0) process.exit(1);
console.log("ALL " + pass + " CHECKS PASSED");
