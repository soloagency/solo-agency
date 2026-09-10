// Offline harness for chrome-extension/solo_entitlement.js — the extension-side plan gate.
//
// Owner decision, refined 2026-09-09 evening: features that make the CRM richer (enrich,
// harvest, zillow, ...) ship on every plan, Free/keyless included — the only sold limit is
// exploitation of contacts above the CRM contact cap, enforced by the bridge, not this file.
// Broadcast write actions on the account — fb.group.post, fb.post.comment, fb.post.react —
// stay gated behind `write_actions` (Starter and up); the keyless fallback must NOT carry it.
// fb.message.send (DM) is no longer part of write_actions: it runs on every plan here and is
// gated per-contact by the bridge instead, so it has no entry in SOLO_CAPABILITY_FEATURES at
// all. This test pins:
//   1. The keyless/unverified fallback (free(), reached via verify() with no token or a
//      malformed one) carries SOLO_FREE_FEATURES — every feature in SOLO_ALL_FEATURES except
//      write_actions — so every enrich/harvest/zillow capability is granted on a keyless
//      install, fb.message.send is unconditionally granted (no feature mapping), and the three
//      broadcast write actions are refused (outside the support-group exemption).
//   2. granted() still enforces the gate MECHANISM: a verified token whose `features` doesn't
//      list a capability's feature is still refused, so the server can re-gate a feature later
//      just by not issuing it; a token carrying write_actions grants comment/react/group.post.
//
// Loads the source with vm.runInContext against a fake self/window/atob/crypto (same trick as
// tests/test_schema.js and tests/test_platform_registry.js) — no require(), no ES modules,
// exactly like background.js's importScripts would see it.
//
// Run:  node solo-agency-collector/tests/test_solo_entitlement.js
const fs = require("fs");
const path = require("path");
const vm = require("vm");

const SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "solo_entitlement.js"), "utf8");

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail) : "")); }
}

function loadEntitlement() {
  const ctx = {};
  ctx.self = ctx;
  ctx.window = ctx;
  ctx.globalThis = ctx;
  ctx.console = console;
  // solo_entitlement.js is written for a Chrome MV3 service worker: atob/TextEncoder/
  // TextDecoder/crypto.subtle are ambient globals there. Hand the vm context the host
  // process's own copies so verify()'s no-token/malformed-token paths run unmodified.
  ctx.atob = globalThis.atob;
  ctx.btoa = globalThis.btoa;
  ctx.TextEncoder = TextEncoder;
  ctx.TextDecoder = TextDecoder;
  ctx.crypto = globalThis.crypto;
  vm.createContext(ctx);
  vm.runInContext(SRC, ctx, { filename: "solo_entitlement.js" });
  return ctx.SoloEntitlement;
}

function sortedEqual(a, b) {
  const sa = a.slice().sort();
  const sb = b.slice().sort();
  return JSON.stringify(sa) === JSON.stringify(sb);
}

const Ent = loadEntitlement();

async function main() {
  // ------------------------------------------------------------------ SOLO_ALL_FEATURES itself
  check(
    "ALL_FEATURES matches the server's feature vocabulary",
    sortedEqual(Ent.ALL_FEATURES, ["multi_client", "outreach", "enrich", "write_actions", "harvest", "zillow", "auto_update", "priority_adapter_fixes"]),
    Ent.ALL_FEATURES
  );

  const usedFeatures = Array.from(new Set(Object.values(Ent.CAPABILITY_FEATURES)));
  check(
    "every feature named in CAPABILITY_FEATURES is included in ALL_FEATURES",
    usedFeatures.every((f) => Ent.ALL_FEATURES.includes(f)),
    usedFeatures
  );
  check(
    "FREE_FEATURES is ALL_FEATURES minus write_actions",
    sortedEqual(Ent.FREE_FEATURES, Ent.ALL_FEATURES.filter((f) => f !== "write_actions")) && !Ent.FREE_FEATURES.includes("write_actions"),
    Ent.FREE_FEATURES
  );
  check("fb.message.send has no entry in CAPABILITY_FEATURES (DM is per-contact-gated by the bridge, not this file)", !("fb.message.send" in Ent.CAPABILITY_FEATURES));

  // ------------------------------------------------------------------ keyless: no token at all
  const noToken = await Ent.verify("", Date.now());
  check("no-token fallback: ok=false (mechanism unchanged)", noToken.ok === false, noToken);
  check("no-token fallback: tier stays 'free'", noToken.tier === "free", noToken);
  check("no-token fallback: source is 'none'", noToken.source === "none", noToken);
  check("no-token fallback: features carries FREE_FEATURES (everything except write_actions)", sortedEqual(noToken.features, Ent.FREE_FEATURES), noToken.features);
  check("no-token fallback: features does NOT carry write_actions", !noToken.features.includes("write_actions"), noToken.features);

  // fb.message.send (DM): unconditionally granted keyless — no feature mapping at all, so the
  // bridge (not this file) is what gates it per contact.
  check("keyless granted(): fb.message.send (DM, ungated here)", Ent.granted(noToken, "fb.message.send") === true);

  // enrich/harvest/zillow: every plan carries these features, keyless included.
  const openCapabilities = Object.keys(Ent.CAPABILITY_FEATURES).filter((c) => Ent.CAPABILITY_FEATURES[c] !== "write_actions");
  check("every enrich/harvest/zillow capability is non-empty (sanity on the fixture)", openCapabilities.length > 0);
  for (const capabilityId of openCapabilities) {
    check("keyless granted(): " + capabilityId, Ent.granted(noToken, capabilityId) === true);
  }

  // write_actions (broadcast to the account): fb.group.post / fb.post.comment / fb.post.react
  // are REFUSED keyless outside the support-group exemption.
  const writeActionCapabilities = Object.keys(Ent.CAPABILITY_FEATURES).filter((c) => Ent.CAPABILITY_FEATURES[c] === "write_actions");
  check(
    "write_actions capabilities are exactly group.post/comment/react",
    sortedEqual(writeActionCapabilities, ["fb.group.post", "fb.post.comment", "fb.post.react"]),
    writeActionCapabilities
  );
  for (const capabilityId of writeActionCapabilities) {
    check("keyless REFUSED (non-support target): " + capabilityId, Ent.granted(noToken, capabilityId, { url: "https://www.facebook.com/groups/999" }) === false);
  }
  check("keyless REFUSED fb.post.comment (no target at all)", Ent.granted(noToken, "fb.post.comment") === false);
  check("keyless REFUSED fb.post.react (no target at all)", Ent.granted(noToken, "fb.post.react") === false);

  // ------------------------------------------------------------------ keyless: malformed token
  // ("not.a.jwt" has 3 dot-separated, non-empty parts, so verify() gets past the shape check and
  // fails at base64/JSON decoding — a different fallback path than the empty-token one, still
  // routed through the same free()).
  const badToken = await Ent.verify("not.a.jwt", Date.now());
  check("malformed-token fallback: ok=false", badToken.ok === false, badToken);
  check("malformed-token fallback: features carries FREE_FEATURES", sortedEqual(badToken.features, Ent.FREE_FEATURES), badToken.features);
  check("malformed-token: an open (enrich) capability is still granted", Ent.granted(badToken, "fb.profile.enrich") === true);
  check("malformed-token: a write_actions capability is still refused", Ent.granted(badToken, "fb.post.comment") === false);

  // ------------------------------------------------------------------ unmapped capability: always granted, any ent (even null)
  check("capability with no feature mapping is granted with ent=null", Ent.granted(null, "fb.profile.view_totally_unmapped") === true);

  // ------------------------------------------------------------------ support-group exemption is untouched by this change
  const supportSource = { url: "https://www.facebook.com/groups/1570411591501058" };
  check("support-group post is granted with ent=null", Ent.granted(null, "fb.group.post", supportSource) === true);
  check("a different group is NOT covered by the support exemption", Ent.granted(null, "fb.group.post", { url: "https://www.facebook.com/groups/999" }) === false);

  // ------------------------------------------------------------------ mechanism intact: a verified token is still limited to ITS OWN features
  // Built directly (not through verify()) since signing a real Ed25519 token needs the server's
  // private key, which this repo does not hold — granted() is what's under test here, and it
  // only reads the shape verify() would have produced.
  const partial = { ok: true, tier: "starter", source: "token", reason: "", expiresAt: new Date(Date.now() + 86400000).toISOString(), companyId: "co_1", limits: {}, features: ["enrich", "write_actions"] };
  check("verified partial token: granted for a feature it carries (enrich)", Ent.granted(partial, "fb.profile.enrich") === true);
  check("verified token carrying write_actions: granted fb.post.react", Ent.granted(partial, "fb.post.react") === true);
  check("verified token carrying write_actions: granted fb.post.comment", Ent.granted(partial, "fb.post.comment") === true);
  check("verified token carrying write_actions: granted fb.group.post (non-support target)", Ent.granted(partial, "fb.group.post", { url: "https://www.facebook.com/groups/999" }) === true);
  check("verified partial token: STILL REFUSED for a feature it lacks (zillow)", Ent.granted(partial, "zillow.agents.list") === false);
  check("verified partial token: STILL REFUSED for a feature it lacks (harvest)", Ent.granted(partial, "fb.profile.friends") === false);

  const empty = { ok: true, tier: "pro", source: "token", reason: "", expiresAt: new Date(Date.now() + 86400000).toISOString(), companyId: "co_2", limits: {}, features: [] };
  check("verified token with empty features: refused for every gated capability", Object.keys(Ent.CAPABILITY_FEATURES).every((c) => Ent.granted(empty, c) === false));

  console.log("\n" + pass + " passed, " + fail + " failed");
  if (fail > 0) process.exit(1);
  console.log("ALL " + pass + " CHECKS PASSED");
}

main().catch((e) => { console.error(e); process.exit(1); });
