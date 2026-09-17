// Actor identity guard (bridge actor_guard.go ↔ platform action libs). Each write lib reads the
// platform's identity cookie, reports it as `actor_account_id` on every write record, and refuses
// a write when the job's `_declared_accounts` (per platform, from collector_config.json
// clients[].accounts[]) does not list that login. Loaded into a vm with a fake window whose
// document carries only a cookie: the guard must answer before any DOM is touched.
//
// Run:  node solo-agency-collector/tests/test_actor_guard.js
const fs = require("fs");
const path = require("path");
const vm = require("vm");

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail) : "")); }
}

function load(file, origin, cookie) {
  const src = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", file), "utf8");
  const document = { cookie, activeElement: null, querySelector: () => null, querySelectorAll: () => [], body: { innerText: "" }, createElement: () => ({ style: {} }), addEventListener: () => {}, visibilityState: "visible" };
  const ctx = { document, location: { pathname: "/home", href: origin + "/home", origin, hostname: origin.replace("https://", "") }, console, setTimeout, clearTimeout, URL, Promise, Date, JSON, Math,
    MouseEvent: function (t) { this.type = t; }, KeyboardEvent: function () {}, InputEvent: function () {}, Event: function (t) { this.type = t; },
    MutationObserver: function () { this.observe = () => {}; this.disconnect = () => {}; }, decodeURIComponent, encodeURIComponent, navigator: { userAgent: "test" }, performance: { now: () => Date.now() } };
  ctx.window = ctx; ctx.self = ctx; ctx.globalThis = ctx;
  ctx.window.__soloX = {}; ctx.window.__soloIg = {}; ctx.window.__soloGql = {};
  vm.createContext(ctx);
  vm.runInContext(src, ctx, { filename: file });
  return ctx;
}

async function main() {
  const cases = [
    { file: "platforms/x/x_actions.js", entry: "__soloXAct", cap: "x.post.publish", origin: "https://x.com", cookie: "guest_id=v1; twid=u%3D123; ct0=abc", me: "123" },
    { file: "platforms/instagram/ig_actions.js", entry: "__soloIgAct", cap: "ig.post.comment", origin: "https://www.instagram.com", cookie: "csrftoken=x; ds_user_id=777", me: "777" },
    { file: "platforms/facebook/gql_actions.js", entry: "__soloActRun", cap: "fb.post.comment", origin: "https://www.facebook.com", cookie: "datr=zz; c_user=555; xs=q", me: "555" },
  ];
  for (const c of cases) {
    let ctx;
    try { ctx = load(c.file, c.origin, c.cookie); } catch (e) { check(c.file + " loads in the fake window", false, String(e && e.message || e)); continue; }
    const run = ctx.window[c.entry];
    check(c.file + " exposes " + c.entry, typeof run === "function");
    if (typeof run !== "function") continue;
    const refused = await run(c.cap, { text: "hello", _declared_accounts: ["999"] });
    const item = refused && refused.items && refused.items[0];
    check(c.file + ": undeclared login is refused before any DOM work", item && item.status === "actor_mismatch", refused);
    check(c.file + ": the record names the actual login", item && item.actor_account_id === c.me, item);
    check(c.file + ": nothing was verified", item && item.verified === false);
    const allowed = await run(c.cap, { text: "hello", _declared_accounts: [c.me], dry_run: true });
    const a1 = allowed && allowed.items && allowed.items[0];
    check(c.file + ": a declared login passes the guard", a1 && a1.status !== "actor_mismatch", allowed);
    check(c.file + ": every write record carries actor_account_id", a1 && a1.actor_account_id === c.me, a1);
    const none = await run(c.cap, { text: "hello", _declared_accounts: [], dry_run: true });
    const n1 = none && none.items && none.items[0];
    check(c.file + ": nothing declared → nothing refused", n1 && n1.status !== "actor_mismatch", none);
  }
  console.log("\n" + pass + " passed, " + fail + " failed");
  if (fail > 0) process.exit(1);
  console.log("ALL " + pass + " CHECKS PASSED");
}
main().catch((e) => { console.error(e); process.exit(1); });
