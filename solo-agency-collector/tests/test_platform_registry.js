// Offline harness for chrome-extension/core/platform_registry.js — the data-only
// description of what background.js hard-codes today as six capability-id tables.
//
// Two things make this test different from a normal unit test:
//   1. It is a DRIFT GUARD. background.js is NOT changed by this step (that is later), so
//      this file pins that background.js takes its tables from the registry (it used to regex-extract six literal tables out of background.js's own
//      source text (the same trick tests/test_gql_actions.js uses to pin
//      background.js's PIN_TARGET/POLICY_FLAG guards) and asserts platform_registry.js
//      reproduces them exactly. If a future edit to background.js changes one of the six
//      tables without updating platform_registry.js, this test fails immediately instead of
//      the two silently diverging.
//   2. It cross-checks against bridge-go/collector_capabilities.json (every capability id
//      must resolve to a module) and bridge-go/main.go (schema.js's sensitive-key needle
//      list must agree with isSensitiveKey's).
//
// Run:  node solo-agency-collector/tests/test_platform_registry.js
const fs = require("fs");
const path = require("path");
const vm = require("vm");

const ROOT = path.join(__dirname, "..");
const REGISTRY_SRC = fs.readFileSync(path.join(ROOT, "chrome-extension", "core", "platform_registry.js"), "utf8");
const SCHEMA_SRC = fs.readFileSync(path.join(ROOT, "chrome-extension", "core", "schema.js"), "utf8");
const BACKGROUND_SRC = fs.readFileSync(path.join(ROOT, "chrome-extension", "background.js"), "utf8");
const MAIN_GO_SRC = fs.readFileSync(path.join(ROOT, "bridge-go", "main.go"), "utf8");
const CATALOG = JSON.parse(fs.readFileSync(path.join(ROOT, "bridge-go", "collector_capabilities.json"), "utf8"));

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail) : "")); }
}

function loadCtx(src, filename) {
  const ctx = {};
  ctx.self = ctx;
  ctx.window = ctx;
  ctx.globalThis = ctx;
  ctx.console = console;
  vm.createContext(ctx);
  vm.runInContext(src, ctx, { filename: filename });
  return ctx;
}

const registryCtx = loadCtx(REGISTRY_SRC, "platform_registry.js");
const P = registryCtx.SoloPlatforms;
const schemaCtx = loadCtx(SCHEMA_SRC, "schema.js");
const Schema = schemaCtx.SoloSchema;

function sortedEqual(a, b) {
  const sa = a.slice().sort();
  const sb = b.slice().sort();
  return JSON.stringify(sa) === JSON.stringify(sb);
}

function setToSortedArray(set) {
  return Array.from(set).sort();
}

// ------------------------------------------------------------------ regex pins on background.js
// Strips // line comments before scanning for quoted strings, so an inline comment
// containing punctuation never gets mistaken for a table entry.
function stripLineComments(s) {
  return s.replace(/\/\/[^\n]*/g, "");
}

function extractSetLiteral(varName) {
  const re = new RegExp("const\\s+" + varName + "\\s*=\\s*new Set\\(\\[([\\s\\S]*?)\\]\\)");
  const m = BACKGROUND_SRC.match(re);
  if (!m) throw new Error("could not find `const " + varName + " = new Set([...])` in background.js");
  const body = stripLineComments(m[1]);
  const out = [];
  const strRe = /"([^"]*)"/g;
  let sm;
  while ((sm = strRe.exec(body))) out.push(sm[1]);
  return out;
}

function extractObjectLiteral(varName) {
  const re = new RegExp("const\\s+" + varName + "\\s*=\\s*\\{([\\s\\S]*?)\\};");
  const m = BACKGROUND_SRC.match(re);
  if (!m) throw new Error("could not find `const " + varName + " = {...};` in background.js");
  const body = stripLineComments(m[1]);
  const out = {};
  const pairRe = /"([^"]+)"\s*:\s*"([^"]+)"/g;
  let pm;
  while ((pm = pairRe.exec(body))) out[pm[1]] = pm[2];
  return out;
}

console.log("\n== drift guard: background.js takes its capability tables from the registry ==");
// Until 2026-09-09 background.js carried six literal tables and this test compared them with
// the registry. The tables ARE the registry's now: pin that each name is assigned from
// SoloPlatforms, that dispatch resolves files and entry points through it, and that no
// literal fb./zillow. capability id or capability-prefix regex is left in the service
// worker's code (comments excluded). Any of these coming back is the old scatter returning.
const CODE = stripLineComments(BACKGROUND_SRC);
[["WRITE_ACTIONS", "writeActions"], ["MATCH_RESOLVABLE", "matchResolvable"], ["PIN_TARGET", "pinTarget"],
 ["POLICY_FLAG", "policyFlags"], ["HIDEABLE_CAPABILITIES", "hideable"]].forEach(function (pair) {
  const re = new RegExp("const\\s+" + pair[0] + "\\s*=\\s*SoloPlatforms\\." + pair[1] + "\\(\\)");
  check("background.js " + pair[0] + " = SoloPlatforms." + pair[1] + "()", re.test(CODE), pair[0]);
});
check("background.js infoOnly uses SoloPlatforms.isInfoOnly", /SoloPlatforms\.isInfoOnly\(/.test(CODE));
check("background.js dispatch resolves run/runFallback entries through the registry",
  /dispatchEntryFor\(capabilityId, "run"\)/.test(CODE) && /dispatchEntryFor\(capabilityId, "runFallback"\)/.test(CODE));
check("background.js resolves act/resolve entries through the registry",
  /dispatchEntryFor\(capabilityId, "act"\)/.test(CODE) && /dispatchEntryFor\(capabilityId, "resolve"\)/.test(CODE));
check("background.js injects module files via dispatchFilesFor", /dispatchFilesFor\(capabilityId/.test(CODE));
check("background.js FB_PERMALINK_HOSTS comes from the registry's host table", /const FB_PERMALINK_HOSTS = new Set\(SoloPlatforms\.moduleForHost\(/.test(CODE));
check("background.js loads core/platform_registry.js via importScripts", /importScripts\([^)]*"core\/platform_registry\.js"/.test(BACKGROUND_SRC));
{
  const leftovers = CODE.match(/"(fb|zillow)\.[a-z_.]+"/g) || [];
  check("no literal fb./zillow. capability id left in background.js code", leftovers.length === 0, leftovers.slice(0, 5));
  check("no capability-prefix regex left in background.js code", !/\/\^(zillow|fb)\\\./.test(CODE));
}
// The registry's own tables, pinned to what shipped (a later change must be deliberate).
check("registry writeActions() has the 4 write ids", P.writeActions().size === 4, setToSortedArray(P.writeActions()));
check("registry hideable() carries the 7 Facebook/Zillow/web hideable ids", ["fb.profile.dossier","fb.profile.header","fb.profile.contacts","fb.profile.hovercard","zillow.agents.list","zillow.profile.enrich","web.search"].every(function (id) { return P.hideable().has(id); }), setToSortedArray(P.hideable()));
check("registry hideable() includes the instagram profile/people capabilities", P.hideable().has("ig.profile.enrich") && P.hideable().has("ig.people.search"), setToSortedArray(P.hideable()));
check("registry policyFlags() maps the 4 write ids", Object.keys(P.policyFlags()).length === 4, P.policyFlags());

// needs_active_tab is the negation of HIDEABLE_CAPABILITIES (background.js
// capabilityNeedsActiveTab, ~3314) — check it holds for every capability the registry knows.
console.log("\n== needs_active_tab === !hideable, for every registered capability ==");
{
  let allOk = true;
  const mismatches = [];
  P.PLATFORM_MODULES.forEach(function (mod) {
    Object.keys(mod.capabilities || {}).forEach(function (capId) {
      const meta = P.capabilityMeta(capId);
      if (meta.needs_active_tab === meta.hideable) { allOk = false; mismatches.push(capId); }
    });
  });
  check("needs_active_tab is the exact negation of hideable for every capability", allOk, mismatches);
}

// ------------------------------------------------------------------ catalog coverage
console.log("\n== every capability id in bridge-go/collector_capabilities.json resolves to a module ==");
{
  const ids = CATALOG.capabilities.map(function (c) { return c.id; });
  check("catalog has capability entries to check (sanity)", ids.length > 0, ids.length);

  let unresolved = [];
  ids.forEach(function (id) {
    if (!P.moduleForCapability(id)) unresolved.push(id);
  });
  check("every catalog capability id resolves to a platform module", unresolved.length === 0, unresolved);

  // The task spec's own count: 20 fb.* ids + web.search + 2 zillow.* ids = 23.
  check("catalog capability count matches the expected 29", ids.length === 29, ids.length);

  ids.forEach(function (id) {
    const mod = P.moduleForCapability(id);
    check("  " + id + " -> module", !!mod && typeof mod.name === "string", mod);
  });
}

// ------------------------------------------------------------------ isSensitiveKey parity with bridge-go/main.go
console.log("\n== schema.js isSensitiveKey agrees with bridge-go/main.go's needle list ==");
{
  const m = MAIN_GO_SRC.match(/needles\s*:=\s*\[\]string\{([^}]*)\}/);
  check("found the Go `needles := []string{...}` literal in main.go", !!m, m);
  const goNeedles = [];
  if (m) {
    const strRe = /"([^"]*)"/g;
    let sm;
    while ((sm = strRe.exec(m[1]))) goNeedles.push(sm[1]);
  }
  check("Go needle list has entries (sanity)", goNeedles.length > 0, goNeedles);
  check(
    "SENSITIVE_KEY_NEEDLES matches the Go needle list exactly (same set)",
    sortedEqual(Schema.SENSITIVE_KEY_NEEDLES.slice(), goNeedles),
    { schema: Schema.SENSITIVE_KEY_NEEDLES.slice().sort(), go: goNeedles.slice().sort() }
  );

  // Behavioural parity on a few names, mirroring Go's strings.Contains(strings.ToLower(key), needle).
  ["auth", "Authorization", "cookie_jar", "SECRET_key", "csrf_token", "plain_field", "author", "session_id"].forEach(function (name) {
    const wantSensitive = goNeedles.some(function (n) { return name.toLowerCase().indexOf(n) !== -1; });
    check(
      "isSensitiveKey(" + JSON.stringify(name) + ") agrees with Go semantics (" + wantSensitive + ")",
      Schema.isSensitiveKey(name) === wantSensitive
    );
  });
}

// ------------------------------------------------------------------ basic lookup sanity
console.log("\n== lookup helpers ==");
{
  check("moduleForCapability('fb.post.react').name === 'facebook'", (P.moduleForCapability("fb.post.react") || {}).name === "facebook");
  check("moduleForCapability('zillow.agents.list').name === 'zillow'", (P.moduleForCapability("zillow.agents.list") || {}).name === "zillow");
  check("moduleForCapability('web.search').name === 'facebook'", (P.moduleForCapability("web.search") || {}).name === "facebook");
  check("moduleForCapability('') === null", P.moduleForCapability("") === null);
  check("moduleForCapability('nonexistent.thing') === null", P.moduleForCapability("nonexistent.thing") === null);

  check("moduleForHost('www.facebook.com').name === 'facebook'", (P.moduleForHost("www.facebook.com") || {}).name === "facebook");
  check("moduleForHost('WWW.ZILLOW.COM') is case-insensitive -> 'zillow'", (P.moduleForHost("WWW.ZILLOW.COM") || {}).name === "zillow");
  check("moduleForHost('example.com') === null", P.moduleForHost("example.com") === null);

  check("isWrite('fb.post.react') === true", P.isWrite("fb.post.react") === true);
  check("isWrite('fb.group.posts') === false", P.isWrite("fb.group.posts") === false);
  check("isMatchResolvable('fb.post.comment') === true", P.isMatchResolvable("fb.post.comment") === true);
  check("isMatchResolvable('fb.message.send') === false (write but not match-resolvable)", P.isMatchResolvable("fb.message.send") === false);
  check("isPinTarget('fb.group.post') === true", P.isPinTarget("fb.group.post") === true);
  check("isPinTarget('fb.message.send') === false (write but not pin-target)", P.isPinTarget("fb.message.send") === false);
  check("policyFlagFor('fb.post.react') === 'do_not_react'", P.policyFlagFor("fb.post.react") === "do_not_react");
  check("policyFlagFor('fb.group.posts') === null", P.policyFlagFor("fb.group.posts") === null);
  check("isHideable('fb.profile.header') === true", P.isHideable("fb.profile.header") === true);
  check("needsActiveTab('fb.profile.header') === false", P.needsActiveTab("fb.profile.header") === false);
  check("needsActiveTab('fb.group.posts') === true", P.needsActiveTab("fb.group.posts") === true);

  // zillow.* is info_only (isZillowCapability catch-all) even though it is NOT part of the
  // literal INFO_ONLY_CAPABILITIES table — see the comment in platform_registry.js.
  check("isInfoOnly('zillow.agents.list') === true (isZillowCapability catch-all)", P.isInfoOnly("zillow.agents.list") === true);
  check("infoOnly() is derived from every module: zillow and instagram info-only ids are in it", P.infoOnly().has("zillow.agents.list") && P.infoOnly().has("ig.profile.enrich") && P.infoOnly().has("fb.profile.header"), setToSortedArray(P.infoOnly()));
  check("isInfoOnly('fb.profile.contacts') === true", P.isInfoOnly("fb.profile.contacts") === true);
  check("isInfoOnly('fb.group.posts') === false", P.isInfoOnly("fb.group.posts") === false);

  check("filesFor('fb.group.posts', {}) reads gql_extract.js", JSON.stringify(P.filesFor("fb.group.posts", {})) === JSON.stringify(["platforms/facebook/gql_extract.js"]));
  check("filesFor('fb.post.react', {write:true}) reads gql_actions.js", JSON.stringify(P.filesFor("fb.post.react", { write: true })) === JSON.stringify(["platforms/facebook/gql_actions.js"]));
  check("filesFor('zillow.agents.list', {}) reads zillow_extract.js", JSON.stringify(P.filesFor("zillow.agents.list", {})) === JSON.stringify(["platforms/zillow/zillow_extract.js"]));
  check("filesFor('zillow.agents.list', {write:true}) is empty (no zillow write file)", JSON.stringify(P.filesFor("zillow.agents.list", { write: true })) === JSON.stringify([]));
  check("filesFor('nonexistent.thing', {}) is empty", JSON.stringify(P.filesFor("nonexistent.thing", {})) === JSON.stringify([]));

  check("entryFor('fb.post.react', 'act') === '__soloActRun'", P.entryFor("fb.post.react", "act") === "__soloActRun");
  check("entryFor('fb.post.comment', 'resolve') === '__soloActResolve'", P.entryFor("fb.post.comment", "resolve") === "__soloActResolve");
  check("entryFor('zillow.agents.list', 'run') === '__soloZillowRun'", P.entryFor("zillow.agents.list", "run") === "__soloZillowRun");
  check("entryFor('fb.group.posts', 'nonexistent_entry') === null", P.entryFor("fb.group.posts", "nonexistent_entry") === null);
  check("entryFor('nonexistent.thing', 'run') === null", P.entryFor("nonexistent.thing", "run") === null);

  const meta = P.capabilityMeta("fb.post.comment");
  check("capabilityMeta('fb.post.comment') carries entity 'message'", meta.entity === "message", meta);
  check("capabilityMeta of an unknown id returns the defaults (entity generic, all false, null flag)",
    JSON.stringify(P.capabilityMeta("nonexistent.thing")) === JSON.stringify({
      entity: "generic", write: false, match_resolvable: false, info_only: false,
      pin_target: false, policy_flag: null, hideable: false, needs_active_tab: false
    }),
    P.capabilityMeta("nonexistent.thing")
  );
}

// ------------------------------------------------------------------ facebook module hosts
console.log("\n== facebook module hosts ==");
{
  const fbMod = P.PLATFORM_MODULES.filter(function (m) { return m.name === "facebook"; })[0];
  check("facebook module declares the four permalink hosts",
    sortedEqual(fbMod.hosts.slice(), ["facebook.com", "www.facebook.com", "m.facebook.com", "web.facebook.com"]), fbMod.hosts);
  check("moduleForHost('www.facebook.com') is the facebook module", P.moduleForHost("www.facebook.com") === fbMod);
}

console.log("\n" + pass + " passed, " + fail + " failed");
if (fail > 0) process.exit(1);
console.log("ALL " + pass + " CHECKS PASSED");
