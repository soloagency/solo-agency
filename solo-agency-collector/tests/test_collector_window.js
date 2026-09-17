// Offline harness for chrome-extension/collector_window_pool.js — the collector's own window.
//
// Fakes chrome.windows / chrome.tabs / storage in memory and pins the contract background.js
// relies on: a window is created focused:false with the home tab; a source tab is created
// ACTIVE inside it; a window whose tab is still collecting is never reused; the last window
// only goes after the linger, and never while a tab is busy; a window the operator closed is
// simply not there next time (no cached state); a home-less window is not ours.
//
// Run:  node solo-agency-collector/tests/test_collector_window.js
const path = require("path");
const { createCollectorWindowPool, DEFAULTS, IDLE_KEY } = require(path.join(__dirname, "..", "chrome-extension", "collector_window_pool.js"));

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? " — " + JSON.stringify(detail) : "")); }
}

const HOME = "chrome-extension://abc/collector_window.html";

function fakeChrome() {
  let nextWin = 100, nextTab = 1000;
  const wins = new Map(); // id -> { id, focused, state, width, height, tabs: [] }
  const created = [];
  const windows = {
    async create(props) {
      const id = nextWin++;
      const w = { id, focused: !!props.focused, state: "normal", width: props.width, height: props.height, type: props.type, tabs: [] };
      wins.set(id, w);
      created.push(props);
      if (props.url) w.tabs.push({ id: nextTab++, windowId: id, url: props.url, active: true });
      return w;
    },
    async getAll() { return Array.from(wins.values()).map((w) => ({ ...w, tabs: w.tabs.slice() })); },
    async get(id) { if (!wins.has(id)) throw new Error("No window with id " + id); return { ...wins.get(id) }; },
    async remove(id) { if (!wins.has(id)) throw new Error("No window with id " + id); wins.delete(id); }
  };
  const tabs = {
    async create(props) {
      const w = wins.get(props.windowId);
      if (!w) throw new Error("No window with id " + props.windowId);
      const t = { id: nextTab++, windowId: props.windowId, url: props.url, active: !!props.active };
      if (t.active) w.tabs.forEach((x) => { x.active = false; });
      w.tabs.push(t);
      return t;
    },
    async remove(id) {
      for (const w of wins.values()) {
        const i = w.tabs.findIndex((t) => t.id === id);
        if (i !== -1) { w.tabs.splice(i, 1); if (!w.tabs.length) wins.delete(w.id); return; }
      }
    }
  };
  const store = new Map();
  const storage = { async get(k) { return store.get(k); }, async set(k, v) { store.set(k, v); } };
  return { windows, tabs, storage, wins, created, store };
}

(async () => {
  console.log("collector window pool");
  {
    const c = fakeChrome();
    const busy = new Set();
    let clock = 1000;
    const pool = createCollectorWindowPool({
      windows: c.windows, tabs: c.tabs, storage: c.storage, homeUrl: HOME,
      isBusyTab: (id) => busy.has(id), now: () => clock
    });

    check("nothing discovered before any window", (await pool.discover()).length === 0);

    const a = await pool.openTab({ url: "https://www.facebook.com/groups/x", width: 1280, height: 900 });
    check("first source creates a window", a.created === true && typeof a.windowId === "number");
    check("window is created focused:false", c.created[0].focused === false, c.created[0]);
    check("window opens on the home page", c.created[0].url === HOME);
    check("window is a normal window with the asked size", c.created[0].type === "normal" && c.created[0].width === 1280 && c.created[0].height === 900);
    check("source tab is ACTIVE inside the collector window", a.tab.active === true && a.tab.windowId === a.windowId);
    check("home tab is no longer the active one", c.wins.get(a.windowId).tabs.find((t) => t.url === HOME).active === false);
    busy.add(a.tab.id);

    const b = await pool.openTab({ url: "https://www.instagram.com/y" });
    check("a busy window is not reused: second concurrent source gets a second window", b.created === true && b.windowId !== a.windowId);
    busy.add(b.tab.id);
    check("both windows are ours", (await pool.discover()).length === 2);

    // source A finishes: tab closed, home keeps the window alive
    await c.tabs.remove(a.tab.id); busy.delete(a.tab.id);
    check("window survives its source tab (home tab keeps it open)", c.wins.has(a.windowId));
    const d = await pool.discover();
    check("finished window is free, busy window is busy", d.find((w) => w.windowId === a.windowId).busy === false && d.find((w) => w.windowId === b.windowId).busy === true);

    const e = await pool.openTab({ url: "https://x.com/z" });
    check("free window is reused before creating a new one", e.created === false && e.windowId === a.windowId);
    busy.add(e.tab.id);

    // run end with sources still busy: idle marker set but sweep refuses while busy
    await pool.markIdle();
    check("markIdle writes the marker to storage", c.store.get(IDLE_KEY) === 1000);
    clock += DEFAULTS.lingerMs + 1;
    let r = await pool.sweepIdle();
    check("sweep never closes while a tab is busy", r.closed === 0 && r.reason === "busy", r);

    await c.tabs.remove(e.tab.id); busy.delete(e.tab.id);
    await c.tabs.remove(b.tab.id); busy.delete(b.tab.id);
    clock = 1000 + 5000;
    r = await pool.sweepIdle();
    check("sweep waits out the linger", r.closed === 0 && r.reason === "lingering", r);
    clock = 1000 + DEFAULTS.lingerMs;
    r = await pool.sweepIdle();
    check("sweep closes every collector window once the linger has passed", r.closed === 2 && c.wins.size === 0, r);
    check("sweep clears the marker", !c.store.get(IDLE_KEY));
    r = await pool.sweepIdle();
    check("nothing to sweep afterwards", r.closed === 0 && r.reason === "not_idle", r);

    // a new run reuses: acquire clears any idle marker
    await pool.markIdle();
    const f = await pool.openTab({ url: "https://www.facebook.com/p" });
    check("acquire clears the idle marker (a new run keeps the window)", !c.store.get(IDLE_KEY) && f.created === true);

    // the operator closes the window: no cached state, the next source creates a fresh one
    await c.windows.remove(f.windowId);
    const g = await pool.openTab({ url: "https://www.facebook.com/q" });
    check("a window the operator closed is not remembered", g.created === true && g.windowId !== f.windowId);

    // a user window without our home tab is never ours
    await c.windows.create({ url: "https://www.facebook.com/", focused: true });
    const mine = await pool.discover();
    check("a window without the home tab is not ours", mine.length === 1 && mine[0].windowId === g.windowId);
    const closed = await pool.closeAll();
    check("closeAll closes only ours", closed === 1 && c.wins.size === 1);

    const desc = await pool.describe(999999);
    check("describe of a gone window says so", desc.gone === true);
  }
  {
    const c = fakeChrome();
    const pool = createCollectorWindowPool({ windows: c.windows, tabs: c.tabs, homeUrl: HOME });
    const r = await pool.sweepIdle();
    check("no storage: sweep is a no-op", r.reason === "no_storage");
    const w = await pool.openTab({ url: "https://example.com", width: 50, height: 99999 });
    check("size is clamped to sane bounds", c.created[0].width === 600 && c.created[0].height === 3000, c.created[0]);
    check("defaults apply when size missing", (await pool.openTab({ url: "https://example.org" })).created === false);
  }
  {
    let threw = null;
    try { createCollectorWindowPool({ windows: {}, tabs: {} }); } catch (e) { threw = e; }
    check("pool refuses to build without a home url", !!threw);
  }
  console.log("\n" + pass + " passed, " + fail + " failed");
  process.exit(fail ? 1 : 0);
})();
