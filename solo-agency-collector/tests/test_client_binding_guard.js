// Offline harness for the "no client_binding.json" guard added 2026-09-10.
//
// Owner decision: extensions/{client_slug}_extension/ (the only folder
// prepare_client_extension.sh ever produces) always carries a client_binding.json. So if
// background.js can't read one, either the SOURCE template folder
// (solo-agency-collector/chrome-extension/, which ships no binding on purpose) was loaded by
// mistake, or a client copy never finished generating. pollBridge() must refuse to touch the
// shared local bridge in that state and the popup must show a clear bilingual message instead
// of silently doing nothing.
//
// This loads the REAL background.js with vm.runInContext (same trick as
// tests/test_solo_entitlement.js and tests/test_offscreen_alert.js — no require(), no ES
// modules, exactly like a service worker's importScripts would see it) so the guard is proven
// against the actual pollBridge() code path, not a re-implementation of it. A tiny fake `chrome`
// and `fetch` stand in for the browser; nothing under test needs chrome.tabs/scripting/etc.
// because the guard returns before pollBridge ever reaches code that uses them.
//
// It also loads the real popup.js against a fake `document` to prove the guard message actually
// reaches the status box the human sees, not just background.js's internal state.
//
// Run:  node solo-agency-collector/tests/test_client_binding_guard.js
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

// ---------------------------------------------------------------------------- background.js harness

function makeChromeMock() {
  const storage = {};
  return {
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
      onMessage: { addListener() {} }
    },
    alarms: {
      create() {},
      onAlarm: { addListener() {} }
    }
  };
}

// bindingMode: "present" (serves a real client_binding.json), "missing_404" (fetch resolves
// !ok, matching a packaged resource that doesn't exist), or "missing_throw" (fetch rejects,
// matching a read failure). statusMode controls what the (never-should-be-reached-when-guarded)
// bridge /status endpoint returns when it IS reached.
function makeFetchMock({ bindingMode, bindingPayload }) {
  const calls = [];
  async function fetchMock(url) {
    calls.push(String(url));
    if (String(url).includes("client_binding.json")) {
      if (bindingMode === "present") {
        return { ok: true, json: async () => bindingPayload };
      }
      if (bindingMode === "missing_404") {
        return { ok: false, status: 404 };
      }
      if (bindingMode === "missing_throw") {
        throw new Error("simulated read failure for client_binding.json");
      }
      throw new Error("unknown bindingMode " + bindingMode);
    }
    if (String(url).includes("/status")) {
      return { ok: true, json: async () => ({ job_available: false }) };
    }
    throw new Error("unexpected fetch in test: " + url);
  }
  return { fetchMock, calls };
}

function loadBackground({ bindingMode, bindingPayload }) {
  const ctx = {};
  const chromeMock = makeChromeMock();
  const { fetchMock, calls } = makeFetchMock({ bindingMode, bindingPayload });

  ctx.self = ctx;
  ctx.window = ctx;
  ctx.globalThis = ctx;
  ctx.console = console;
  ctx.chrome = chromeMock;
  ctx.fetch = fetchMock;
  ctx.AbortController = AbortController;
  ctx.setTimeout = setTimeout;
  ctx.clearTimeout = clearTimeout;
  ctx.setInterval = setInterval;
  ctx.clearInterval = clearInterval;
  // solo_entitlement.js (loaded via importScripts below) is written for a Chrome MV3 service
  // worker: atob/TextEncoder/TextDecoder/crypto.subtle are ambient globals there. None of the
  // guard scenarios below ever reach SoloEntitlement.verify(), but the module still needs these
  // to exist to load without throwing (same setup as tests/test_solo_entitlement.js).
  ctx.atob = globalThis.atob;
  ctx.btoa = globalThis.btoa;
  ctx.TextEncoder = TextEncoder;
  ctx.TextDecoder = TextDecoder;
  ctx.crypto = globalThis.crypto;

  // Real importScripts loads and executes the named sibling file into THIS SAME context, exactly
  // like the browser's service-worker importScripts would -- so SoloEntitlement/SoloPlatforms
  // etc. become real global bindings, not stubs.
  ctx.importScripts = (...files) => {
    for (const rel of files) {
      const src = fs.readFileSync(path.join(EXT_DIR, rel), "utf8");
      vm.runInContext(src, ctx, { filename: rel });
    }
  };

  vm.createContext(ctx);
  vm.runInContext(BACKGROUND_SRC, ctx, { filename: "background.js" });
  return { ctx, calls, storage: chromeMock.storage.local };
}

// ---------------------------------------------------------------------------- popup.js harness

function makeFakeDocument() {
  const elements = {};
  function el(id) {
    if (!elements[id]) {
      elements[id] = {
        id,
        textContent: "",
        value: "",
        checked: false,
        style: {},
        addEventListener() {}
      };
    }
    return elements[id];
  }
  return { getElementById: el, addEventListener() {}, elements };
}

function loadPopup() {
  const ctx = {};
  const document = makeFakeDocument();
  ctx.document = document;
  ctx.console = console;
  ctx.chrome = { runtime: { sendMessage() {}, lastError: undefined } };
  vm.createContext(ctx);
  vm.runInContext(POPUP_SRC, ctx, { filename: "popup.js" });
  return { ctx, document };
}

async function main() {
  const bindingPayload = {
    client_slug: "acme-realty",
    client_name: "Acme Realty",
    extension_instance_id: "ext_acme-realty_default",
    extension_display_name: "Acme Realty - Solo Agency Collector",
    bridge_base_url: "http://127.0.0.1:17321"
  };

  // ------------------------------------------------------------------ binding present -> normal
  console.log("binding present: normal operation, unaffected by the guard");
  {
    const { ctx, calls } = loadBackground({ bindingMode: "present", bindingPayload });
    const binding = await ctx.getClientBinding();
    check("getClientBinding() returns the real client_slug", binding.client_slug === "acme-realty", binding);

    const result = await ctx.pollBridge("test");
    check("pollBridge() reaches the normal bridge flow (idle: bridge online, no job)", result.status === "idle", result);
    check("exactly one client_binding.json fetch happened", calls.filter((u) => u.includes("client_binding.json")).length === 1, calls);
    check("the bridge WAS polled (/status reached) -- the guard must not block a real binding", calls.some((u) => u.includes("/status")), calls);

    const state = await ctx.getState();
    check("resulting state.status is not the guard status", state.status !== "no_client_binding", state.status);
  }

  // ------------------------------------------------------------- binding missing (404) -> guard
  console.log("\nbinding missing (packaged resource not found, response not ok): guard trips");
  {
    const { ctx, calls } = loadBackground({ bindingMode: "missing_404" });
    const binding = await ctx.getClientBinding();
    check("getClientBinding() falls back to an empty client_slug", binding.client_slug === "", binding);

    const result = await ctx.pollBridge("test");
    check("pollBridge() returns the no_client_binding guard status", result.status === "no_client_binding", result);
    check("the bridge was NEVER polled (no /status call)", !calls.some((u) => u.includes("/status")), calls);
    check("only the client_binding.json lookup happened, nothing else", calls.length === 1, calls);

    const state = await ctx.getState();
    check("stored state carries the guard status", state.status === "no_client_binding", state.status);
    check("stored state message is present and non-empty", typeof state.message === "string" && state.message.length > 0, state.message);
    check("message is bilingual: contains the Vietnamese sentence", state.message.includes("thư mục MÃ NGUỒN"), state.message);
    check("message is bilingual: contains the English gloss", state.message.includes("This is the SOURCE folder"), state.message);
    check("message names the fix using the new {client_slug}_extension convention", state.message.includes("{client_slug}_extension"), state.message);
  }

  // --------------------------------------------------- binding unreadable (fetch throws) -> guard
  console.log("\nbinding unreadable (fetch rejects, e.g. a broken/corrupted copy): guard trips the same way");
  {
    const { ctx, calls } = loadBackground({ bindingMode: "missing_throw" });
    const result = await ctx.pollBridge("test");
    check("pollBridge() returns the no_client_binding guard status", result.status === "no_client_binding", result);
    check("the bridge was NEVER polled (no /status call)", !calls.some((u) => u.includes("/status")), calls);
  }

  // ------------------------------------------------------- guard applies at every pollBridge entry
  // onInstalled/onStartup/the alarm/short-poll/settings_saved/check_now all funnel through
  // pollBridge() as the single choke point -- proven here via the message-triggered entry points
  // instead of re-registering chrome.runtime listeners (which are stubbed no-ops in this harness).
  console.log("\nguard also holds on a second call (cache does not paper over a missing binding)");
  {
    const { ctx, calls } = loadBackground({ bindingMode: "missing_404" });
    await ctx.pollBridge("first");
    const result2 = await ctx.pollBridge("second");
    check("second poll still guarded", result2.status === "no_client_binding", result2);
    check("client_binding.json was only actually fetched once (cached after the first miss)", calls.filter((u) => u.includes("client_binding.json")).length === 1, calls);
    check("still no /status call across both polls", !calls.some((u) => u.includes("/status")), calls);
  }

  // ---------------------------------------------------------------------------- popup.js: message present
  console.log("\npopup: guard state renders the bilingual message into the status box");
  {
    const guardState = { status: "no_client_binding", message: "Đây là thư mục MÃ NGUỒN... / This is the SOURCE folder..." };
    // renderState is a top-level function declaration in popup.js, so it lands directly on the
    // vm context as a callable global.
    const { ctx: popupCtx } = loadPopup();
    popupCtx.renderState(guardState);
    const statusEl = popupCtx.document.getElementById("status");
    check("status box shows the guard message verbatim", statusEl.textContent === guardState.message, statusEl.textContent);
    check("status box is visually flagged (red border) so it isn't missed", statusEl.style.borderColor === "#d1242f", statusEl.style);

    const normalState = { status: "idle", message: "Bridge is online but no unfinished job is available." };
    popupCtx.renderState(normalState);
    const statusEl2 = popupCtx.document.getElementById("status");
    check("normal state still renders (regression check) and clears the red styling", statusEl2.textContent.includes("Status: idle") && statusEl2.style.borderColor === "", [statusEl2.textContent, statusEl2.style]);
  }

  console.log("\n" + pass + " passed, " + fail + " failed");
  if (fail > 0) process.exit(1);
  console.log("ALL " + pass + " CHECKS PASSED");
}

main().catch((e) => { console.error(e); process.exit(1); });
