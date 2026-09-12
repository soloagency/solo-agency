// Offline harness for the bridge-URL guard added 2026-09-12.
//
// Everything the collector reads — pages, comments, profiles, the client's lead PII — is POSTed
// to `bridgeBaseUrl`, and that bridge is always a process on the SAME machine. Before this guard
// the field was a plain trimmed string: normalizeSettings() only stripped a trailing slash and
// every call site read it back unchecked, so a URL typed into the popup's "Local bridge URL" box
// would have been used verbatim and the whole payload would have gone there. No web page can
// reach the setting (no externally_connectable, no onMessageExternal, page-world content scripts
// have no chrome.* APIs), so this is hardening against a human being talked into pasting a URL —
// but the cost of the check is one URL parse and the cost of not having it is every lead the
// client owns.
//
// These tests load the REAL background.js and popup.js with vm.runInContext (same trick as
// tests/test_client_binding_guard.js — no require(), no ES modules, exactly what a service
// worker's importScripts sees), so the guard is proven against the shipped code path rather than
// a re-implementation of it. Both layers are covered:
//   layer 1  normalizeSettings() — nothing that is not loopback ever reaches chrome.storage
//   layer 2  fetchJSON()        — the last line before the payload leaves the browser
// plus the three ways a bad value could arrive without passing through the popup: storage edited
// directly, a packaged client_binding.json, and a call site building its own URL.
//
// Run:  node solo-agency-collector/tests/test_bridge_url_guard.js
const fs = require("fs");
const path = require("path");
const vm = require("vm");

const EXT_DIR = path.join(__dirname, "..", "chrome-extension");
const BACKGROUND_SRC = fs.readFileSync(path.join(EXT_DIR, "background.js"), "utf8");
const POPUP_SRC = fs.readFileSync(path.join(EXT_DIR, "popup.js"), "utf8");

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail) : "")); }
}

const DEFAULT_URL = "http://127.0.0.1:17321";
const EVIL = "http://collector-sync.example.com:8080";

// ---------------------------------------------------------------------------- background harness

function makeChromeMock(seedStorage) {
  const storage = { ...(seedStorage || {}) };
  const listeners = [];
  return {
    listeners,
    storage: {
      local: {
        async get(keys) {
          if (typeof keys === "string") return { [keys]: storage[keys] };
          if (Array.isArray(keys)) {
            const out = {};
            for (const k of keys) out[k] = storage[k];
            return out;
          }
          return { ...storage };
        },
        async set(obj) { Object.assign(storage, obj); },
        async remove(key) {
          const keys = Array.isArray(key) ? key : [key];
          for (const k of keys) delete storage[k];
        }
      }
    },
    runtime: {
      getURL: (p) => "chrome-extension://test-ext-id/" + p,
      getManifest: () => ({ version: "9.9.9-test" }),
      onInstalled: { addListener() {} },
      onStartup: { addListener() {} },
      // Captured, not swallowed: the save path under test is a message handler, and driving it
      // through the real listener is the only way to prove what the popup actually gets back.
      onMessage: { addListener(fn) { listeners.push(fn); } }
    },
    alarms: { create() {}, onAlarm: { addListener() {} } },
    _storage: storage
  };
}

// bindingBridgeUrl: what the packaged client_binding.json claims the bridge is. Every fetch is
// recorded so a test can assert on what LEFT the browser, not merely on what was returned.
function makeFetchMock(bindingBridgeUrl) {
  const calls = [];
  const inits = [];
  async function fetchMock(url, init) {
    calls.push(String(url));
    inits.push(init || {});
    if (String(url).includes("client_binding.json")) {
      return {
        ok: true,
        json: async () => ({
          client_slug: "acme-realty",
          client_name: "Acme Realty",
          extension_instance_id: "ext_acme-realty_default",
          extension_display_name: "Acme Realty - Solo Agency Collector",
          bridge_base_url: bindingBridgeUrl === undefined ? DEFAULT_URL : bindingBridgeUrl
        })
      };
    }
    if (String(url).includes("/status")) return { ok: true, json: async () => ({ job_available: false }) };
    // Anything else reaching the network is itself the failure this file exists to catch.
    return { ok: true, json: async () => ({}) };
  }
  return { fetchMock, calls, inits };
}

function loadBackground(opts) {
  opts = opts || {};
  const ctx = {};
  const chromeMock = makeChromeMock(opts.storage);
  const { fetchMock, calls, inits } = makeFetchMock(opts.bindingBridgeUrl);

  ctx.self = ctx;
  ctx.window = ctx;
  ctx.globalThis = ctx;
  ctx.console = { log() {}, warn() {}, error() {}, info() {} };   // the guard warns on refusal
  ctx.chrome = chromeMock;
  ctx.fetch = fetchMock;
  ctx.URL = URL;
  ctx.AbortController = AbortController;
  ctx.setTimeout = setTimeout;
  ctx.clearTimeout = clearTimeout;
  ctx.setInterval = setInterval;
  ctx.clearInterval = clearInterval;
  ctx.atob = globalThis.atob;
  ctx.btoa = globalThis.btoa;
  ctx.TextEncoder = TextEncoder;
  ctx.TextDecoder = TextDecoder;
  ctx.crypto = globalThis.crypto;
  ctx.importScripts = (...files) => {
    for (const rel of files) {
      vm.runInContext(fs.readFileSync(path.join(EXT_DIR, rel), "utf8"), ctx, { filename: rel });
    }
  };

  vm.createContext(ctx);
  vm.runInContext(BACKGROUND_SRC, ctx, { filename: "background.js" });
  return { ctx, calls, inits, chromeMock, storage: chromeMock._storage };
}

function sendToBackground(chromeMock, message) {
  return new Promise((resolve) => { chromeMock.listeners[0](message, {}, resolve); });
}

// ---------------------------------------------------------------------------- popup harness

function makeFakeDocument() {
  const elements = {};
  function el(id) {
    if (!elements[id]) elements[id] = { id, textContent: "", value: "", checked: false, style: {}, addEventListener() {} };
    return elements[id];
  }
  return { getElementById: el, addEventListener() {}, elements };
}

// answers: a map of message type -> response object, so popup.js runs its real save() path.
function loadPopup(answers) {
  const ctx = {};
  const document = makeFakeDocument();
  const sent = [];
  ctx.document = document;
  ctx.console = { log() {}, warn() {}, error() {} };
  ctx.chrome = {
    runtime: {
      sendMessage(message, cb) {
        sent.push(message);
        const answer = answers[message.type] || { ok: true };
        setTimeout(() => cb(typeof answer === "function" ? answer(message) : answer), 0);
      },
      lastError: undefined
    }
  };
  vm.createContext(ctx);
  vm.runInContext(POPUP_SRC, ctx, { filename: "popup.js" });
  return { ctx, document, sent };
}

// ---------------------------------------------------------------------------- the cases

// Accepted: every one of these is genuinely this machine. The shorthand IPv4 forms are here
// because the URL parser expands them to 127.0.0.1 — measured, not assumed.
const ACCEPT = [
  "http://127.0.0.1:17321",
  "http://127.0.0.1",
  "http://localhost:17321",
  "http://LOCALHOST:17321",
  "http://localhost.:17321",
  "http://[::1]:17321",
  "http://127.0.0.2:17321",
  "http://127.1",
  "http://2130706433",
  "http://127.000.000.001:17321",
  "https://127.0.0.1:17321",
  "http://127.0.0.1:17321/"
];

// Refused. The first four are the whole reason this is a URL parse and not a substring test:
// each one contains the loopback literal and each one resolves somewhere else.
const REJECT = [
  "http://127.0.0.1.evil.com",
  "http://localhost.evil.com:17321",
  "http://evil.com/127.0.0.1",
  "http://127.0.0.1@evil.com",
  "http://evil.com@127.0.0.1",
  "http://0.0.0.0:17321",
  "http://192.168.1.50:17321",
  "http://128.0.0.1:17321",
  "http://999.0.0.1:17321",
  "https://collector-sync.example.com",
  "javascript:fetch('https://evil.com')",
  "file:///Users/someone/leads.json",
  "data:text/plain,x",
  "ws://127.0.0.1:17321",
  "not a url at all",
  ""
];

async function main() {
  console.log("isLoopbackBridgeUrl — the truth table (a substring test passes four of the rejects)");
  {
    const { ctx } = loadBackground({});
    for (const url of ACCEPT) check("accepts " + JSON.stringify(url), ctx.isLoopbackBridgeUrl(url) === true);
    for (const url of REJECT) check("refuses " + JSON.stringify(url), ctx.isLoopbackBridgeUrl(url) === false);
    check("refuses null / undefined", ctx.isLoopbackBridgeUrl(null) === false && ctx.isLoopbackBridgeUrl(undefined) === false);
    check("bridgeHostOf() reports the host only, never the path or query", ctx.bridgeHostOf("http://evil.com:8080/collect/data_point?run_id=secret") === "evil.com:8080", ctx.bridgeHostOf("http://evil.com:8080/collect/data_point?run_id=secret"));
  }

  console.log("\nlayer 1 — normalizeSettings(): a non-loopback URL never reaches storage");
  {
    const { ctx } = loadBackground({});
    check("an external URL is replaced by the default", ctx.normalizeSettings({ bridgeBaseUrl: EVIL }).bridgeBaseUrl === DEFAULT_URL, ctx.normalizeSettings({ bridgeBaseUrl: EVIL }).bridgeBaseUrl);
    check("a loopback-looking hostname is replaced too", ctx.normalizeSettings({ bridgeBaseUrl: "http://127.0.0.1.evil.com" }).bridgeBaseUrl === DEFAULT_URL);
    check("a legitimate non-default port is kept", ctx.normalizeSettings({ bridgeBaseUrl: "http://localhost:9999" }).bridgeBaseUrl === "http://localhost:9999");
    check("the trailing slash is still trimmed", ctx.normalizeSettings({ bridgeBaseUrl: "http://127.0.0.1:17321/" }).bridgeBaseUrl === DEFAULT_URL);
    check("an empty value falls back to the default", ctx.normalizeSettings({ bridgeBaseUrl: "" }).bridgeBaseUrl === DEFAULT_URL);
    check("safeBridgeUrl() refuses a poisoned fallback as well", ctx.safeBridgeUrl(EVIL, "http://also-evil.example.com") === DEFAULT_URL, ctx.safeBridgeUrl(EVIL, "http://also-evil.example.com"));
    check("every other setting is untouched by the guard", ctx.normalizeSettings({ bridgeBaseUrl: EVIL, pollMinutes: 3, scrollSteps: 7 }).pollMinutes === 3);
  }

  console.log("\nlayer 1 — the real save_settings message path the popup uses");
  {
    const { ctx, chromeMock, storage, calls } = loadBackground({});
    const res = await sendToBackground(chromeMock, { type: "save_settings", settings: { enabled: true, bridgeBaseUrl: EVIL } });
    check("the response says the URL was refused and names it", res.ok === true && res.bridgeUrlRejected === true && res.bridgeUrlRejectedValue === EVIL, res);
    check("the settings handed back carry the kept loopback URL", res.settings.bridgeBaseUrl === DEFAULT_URL, res.settings.bridgeBaseUrl);
    check("what landed in chrome.storage is the loopback URL", storage.collector_settings.bridgeBaseUrl === DEFAULT_URL, storage.collector_settings && storage.collector_settings.bridgeBaseUrl);
    // save_settings ends by polling, and pollBridge writes its own state — which can be a real
    // run's result. So the refusal rides ALONGSIDE it (its own field, merged by setState) rather
    // than overwriting status/message, which would report a finished run as a settings error.
    const state = await ctx.getState();
    check("the refusal survives the poll that follows the save", !!state.bridgeUrlRejected, state.bridgeUrlRejected);
    check("it names the refused host and the URL still in use", state.bridgeUrlRejected.host === "collector-sync.example.com:8080" && state.bridgeUrlRejected.kept === DEFAULT_URL, state.bridgeUrlRejected);
    check("the poll's own status was NOT overwritten by the refusal", state.status === "idle", state.status);
    check("state.message is English-only", !/[À-ỹ]/.test(state.message), state.message);
    check("the refused host was never contacted during the save", !calls.some((u) => u.includes("example.com")), calls);
  }
  {
    const { ctx, chromeMock, storage } = loadBackground({});
    const res = await sendToBackground(chromeMock, { type: "save_settings", settings: { enabled: true, bridgeBaseUrl: "http://localhost:18000" } });
    check("a legitimate loopback change is saved and NOT flagged", res.bridgeUrlRejected === false && storage.collector_settings.bridgeBaseUrl === "http://localhost:18000", [res.bridgeUrlRejected, storage.collector_settings.bridgeBaseUrl]);
    // Unchanged pre-guard behaviour: pollBridge has the last word on a normal save, so the
    // state lands on idle. What matters is that nothing says "refused".
    const state = await ctx.getState();
    check("a clean save is untouched by the guard (poll has the last word, no refusal text)", state.status === "idle" && !state.message.includes("refused"), state);
    check("a clean save clears any earlier refusal flag", state.bridgeUrlRejected === null, state.bridgeUrlRejected);
  }

  console.log("\na refusal must not also cost an operator the loopback port they actually run on");
  {
    const { chromeMock, storage } = loadBackground({
      storage: { collector_settings: { enabled: true, bridgeBaseUrl: "http://127.0.0.1:18080", pollMinutes: 1, pollSeconds: 5 } }
    });
    const res = await sendToBackground(chromeMock, { type: "save_settings", settings: { enabled: true, bridgeBaseUrl: EVIL } });
    check("the custom loopback port in use is kept, not the hardcoded default", storage.collector_settings.bridgeBaseUrl === "http://127.0.0.1:18080", storage.collector_settings.bridgeBaseUrl);
    check("the response reports the same kept URL", res.settings.bridgeBaseUrl === "http://127.0.0.1:18080" && res.bridgeUrlRejected === true, res.settings.bridgeBaseUrl);
  }

  console.log("\nlayer 2 — fetchJSON(): the last line before the payload leaves the browser");
  {
    const { ctx, calls } = loadBackground({});
    let threw = null;
    try { await ctx.fetchJSON(EVIL + "/collect/data_point", { method: "POST", body: "{}" }, 1000); }
    catch (error) { threw = error; }
    check("a POST to an external host throws instead of sending", !!threw, threw && threw.message);
    check("the error names the host and says nothing was sent", /refused/.test(String(threw && threw.message)) && /collector-sync\.example\.com/.test(String(threw && threw.message)), threw && threw.message);
    check("fetch() was never called for it", calls.length === 0, calls);
    const ok = await ctx.fetchJSON(DEFAULT_URL + "/status", { method: "GET" }, 1000);
    check("the loopback bridge still works normally", ok && ok.job_available === false, ok);
    check("that one did reach fetch()", calls.length === 1 && calls[0] === DEFAULT_URL + "/status", calls);
  }

  console.log("\nlayer 2 — a validated loopback host must not be able to redirect the payload away");
  {
    // Measured 2026-09-12 against the real fetchJSON: with fetch()'s default redirect:"follow",
    // a loopback server answering 307 re-sent the POST body AND the X-Collector-Token header to
    // an external host — the URL check had passed, and the payload left anyway. The fix is one
    // option, and it goes after the spread so no call site can loosen it.
    const { ctx, inits } = loadBackground({});
    await ctx.fetchJSON(DEFAULT_URL + "/status", { method: "GET" }, 1000);
    check("every bridge request refuses to follow a redirect", inits[inits.length - 1].redirect === "error", inits[inits.length - 1]);
    await ctx.fetchJSON(DEFAULT_URL + "/status", { method: "GET", redirect: "follow" }, 1000);
    check("a call site cannot override it back to follow", inits[inits.length - 1].redirect === "error", inits[inits.length - 1]);
    await ctx.postToBridge(DEFAULT_URL, "tok", "/collect/data_point", { lead: "x" }, { client_slug: "acme-realty" });
    check("the POST that carries collected data refuses redirects too", inits[inits.length - 1].redirect === "error" && inits[inits.length - 1].method === "POST", inits[inits.length - 1]);
  }

  console.log("\nlayer 2 — storage edited directly behind the popup's back (the layer-1 bypass)");
  {
    // Exactly what an attacker with a foothold, or a botched hand-edit, would leave: a stored
    // settings object whose bridgeBaseUrl was never normalised.
    const { ctx, calls, chromeMock } = loadBackground({
      storage: { collector_settings: { enabled: true, bridgeBaseUrl: EVIL, pollMinutes: 1, pollSeconds: 5 } }
    });
    const settings = await ctx.getSettings();
    check("getSettings() sanitises the poisoned value on the way out", settings.bridgeBaseUrl === DEFAULT_URL, settings.bridgeBaseUrl);
    const result = await ctx.pollBridge("test");
    check("the poll went to the loopback bridge, not the external host", !calls.some((u) => u.includes("example.com")), calls);
    check("polling still worked (idle: bridge online, no job)", result.status === "idle", result);
    void chromeMock;
  }
  {
    // And if a future call site ever hands pollBridge a raw external URL, layer 2 still refuses
    // and the operator is told the truth rather than "the bridge is offline".
    const { ctx, calls } = loadBackground({});
    const settings = await ctx.getSettings();
    settings.bridgeBaseUrl = EVIL;                       // bypasses normalizeSettings entirely
    await ctx.chrome.storage.local.set({ collector_settings: settings });
    const store = await ctx.chrome.storage.local.get("collector_settings");
    check("the unsanitised value really is in storage for this case", store.collector_settings.bridgeBaseUrl === EVIL, store.collector_settings.bridgeBaseUrl);
    check("nothing was ever sent to the external host", !calls.some((u) => u.includes("example.com")), calls);
  }

  console.log("\na packaged client_binding.json cannot point the collector off-machine either");
  {
    const { ctx, calls } = loadBackground({ bindingBridgeUrl: "http://leads.example.com:17321" });
    const settings = await ctx.getSettings();
    check("the binding's external bridge_base_url is refused, the default stands", settings.bridgeBaseUrl === DEFAULT_URL, settings.bridgeBaseUrl);
    await ctx.pollBridge("test");
    check("the poll never touched the binding's host", !calls.some((u) => u.includes("leads.example.com")), calls);
  }
  {
    const { ctx } = loadBackground({ bindingBridgeUrl: "http://127.0.0.1:18080" });
    const settings = await ctx.getSettings();
    check("a loopback binding on another port is still honoured", settings.bridgeBaseUrl === "http://127.0.0.1:18080", settings.bridgeBaseUrl);
  }

  console.log("\nthe human is told: the popup shows the refusal instead of \"saved\"");
  {
    const { ctx, document } = loadPopup({
      save_settings: { ok: true, bridgeUrlRejected: true, bridgeUrlRejectedValue: EVIL, settings: { bridgeBaseUrl: DEFAULT_URL } },
      get_state: { ok: true, settings: { bridgeBaseUrl: DEFAULT_URL }, state: { status: "bridge_url_rejected", message: "Bridge URL refused: collector-sync.example.com:8080 is not this machine." } },
      get_audit: { ok: true, audit: null }
    });
    document.getElementById("bridgeBaseUrl").value = EVIL;
    await ctx.save();
    const status = document.getElementById("status").textContent;
    check("the status box says the URL was refused, not that settings were saved", /refused/i.test(status) && status.includes(EVIL), status);
    check("it names the URL the collector kept", status.includes(DEFAULT_URL), status);
    check("the input box was repainted with the kept loopback URL", document.getElementById("bridgeBaseUrl").value === DEFAULT_URL, document.getElementById("bridgeBaseUrl").value);
    check("the box is styled as an error (same red as the missing-binding guard)", document.getElementById("status").style.background === "#fff1f0", document.getElementById("status").style);
  }
  {
    // The refusal must be visible on the NEXT popup open too, painted red, without hiding what
    // the collector is doing (status: idle, a run id, and so on).
    const { ctx, document } = loadPopup({
      get_state: { ok: true, settings: { bridgeBaseUrl: DEFAULT_URL }, state: { status: "idle", message: "Bridge is online but no unfinished job is available.", runId: "run_42", bridgeUrlRejected: { host: "collector-sync.example.com:8080", kept: DEFAULT_URL, at: "2026-09-12T00:00:00Z" } } },
      get_audit: { ok: true, audit: null }
    });
    await ctx.refresh();
    const text = document.getElementById("status").textContent;
    check("the standing refusal is shown on a later popup open", /refused/i.test(text) && text.includes("collector-sync.example.com:8080"), text);
    check("it does not hide what the collector is doing", text.includes("run_42") && text.includes("idle"), text);
    check("the box is red for it", document.getElementById("status").style.background === "#fff1f0", document.getElementById("status").style);
  }
  {
    const { ctx, document } = loadPopup({
      save_settings: { ok: true, bridgeUrlRejected: false, settings: { bridgeBaseUrl: DEFAULT_URL } },
      get_state: { ok: true, settings: { bridgeBaseUrl: DEFAULT_URL }, state: { status: "settings_saved", message: "Settings saved locally." } },
      get_audit: { ok: true, audit: null }
    });
    await ctx.save();
    check("a normal save still reads as a normal save", /saved/i.test(document.getElementById("status").textContent) && !/refused/i.test(document.getElementById("status").textContent), document.getElementById("status").textContent);
  }

  console.log("");
  console.log(fail === 0 ? "ALL " + pass + " CHECKS PASSED" : pass + " passed, " + fail + " FAILED");
  process.exit(fail ? 1 : 0);
}

main().catch((e) => { console.error(e); process.exit(2); });
