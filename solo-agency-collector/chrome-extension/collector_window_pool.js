// The collector's own Chrome window(s) — "collector window" (2026-09-16).
//
// Until now every collection tab was created in whatever window the operator was using and,
// for 16 of 25 capabilities, ACTIVATED there (feeds, searches, comments and every write need a
// rendered tab: React virtualisation does not build posts in a hidden tab). So a run stole the
// operator's view for minutes at a time, and the 12 capabilities that CAN run hidden paid for
// it with Chrome's background-tab throttling (measured 2-3x slower, see CAPABILITY_TIMEOUT_MS).
//
// A dedicated window is the way out of both: chrome.windows.create({focused:false}) opens a
// normal window BEHIND the operator's, in the same profile (same logins, nothing to re-auth),
// and every collection tab is created active INSIDE it. Active-in-its-window is what stops the
// throttle; not-focused is what leaves the operator alone. A home tab (collector_window.html)
// keeps the window alive between sources — Chrome closes a window with its last tab.
//
// What this does NOT solve, and measures instead: a window that another window covers
// COMPLETELY is "occluded", and Chrome treats its tabs as hidden (visibilityState, throttling
// included). background.js probes the tab after load and records tab_visibility / timer drift
// on every data point, so a covered collector window shows up in the data instead of as
// silently emptier feeds. Nothing here can un-occlude a window without focusing it.
//
// Dependencies are injected (chrome.windows / chrome.tabs / storage) so tests/test_collector_window.js
// runs this file under node with fakes. Pool state is rediscovered from Chrome on every call
// (a window is ours iff one of its tabs is our home page), never cached — the service worker
// may restart mid-run, and the operator may close the window at any time.
(function (root) {
  const DEFAULTS = Object.freeze({ width: 1280, height: 900, lingerMs: 20000 });
  const IDLE_KEY = "collector_window_idle_since";

  function createCollectorWindowPool(deps) {
    const windowsApi = deps.windows;
    const tabsApi = deps.tabs;
    const homeUrl = String(deps.homeUrl || "");
    const isBusyTab = typeof deps.isBusyTab === "function" ? deps.isBusyTab : function () { return false; };
    const storage = deps.storage || null;
    const now = typeof deps.now === "function" ? deps.now : function () { return Date.now(); };
    if (!windowsApi || !tabsApi || !homeUrl) throw new Error("collector window pool needs windows, tabs and homeUrl");

    function isHomeTab(tab) {
      const url = String((tab && (tab.url || tab.pendingUrl)) || "");
      return url.indexOf(homeUrl) === 0;
    }

    // Every normal window that carries our home tab, with its other tabs and whether any of
    // them belongs to a source still being collected.
    async function discover() {
      let all = [];
      try { all = await windowsApi.getAll({ populate: true, windowTypes: ["normal"] }); }
      catch (error) { return []; }
      const mine = [];
      for (const win of all || []) {
        const tabs = Array.isArray(win.tabs) ? win.tabs : [];
        const home = tabs.find(isHomeTab);
        if (!home) continue;
        const others = tabs.filter(function (t) { return t !== home; });
        mine.push({
          windowId: win.id,
          homeTabId: home.id,
          focused: !!win.focused,
          state: win.state || "",
          tabs: others,
          busy: others.some(function (t) { return typeof t.id === "number" && isBusyTab(t.id); })
        });
      }
      return mine;
    }

    async function create(opts) {
      const o = opts || {};
      const win = await windowsApi.create({
        url: homeUrl,
        type: "normal",
        focused: false,
        width: clampInt(o.width, 600, 4000, DEFAULTS.width),
        height: clampInt(o.height, 400, 3000, DEFAULTS.height)
      });
      if (!win || typeof win.id !== "number") throw new Error("chrome.windows.create returned no window");
      return win.id;
    }

    // A window with no source in flight, or a new one. Reuse comes first: healthcheck runs are
    // many one-source jobs a few seconds apart, and a window per job would flicker.
    async function acquire(opts) {
      const free = (await discover()).find(function (w) { return !w.busy; });
      await clearIdle();
      if (free) return { windowId: free.windowId, created: false };
      return { windowId: await create(opts), created: true };
    }

    // The one call background.js makes per source: a tab, active in a collector window.
    async function openTab(opts) {
      const o = opts || {};
      const slot = await acquire(o);
      const tab = await tabsApi.create({ windowId: slot.windowId, url: o.url, active: true });
      if (!tab || typeof tab.id !== "number") throw new Error("chrome.tabs.create returned no tab");
      return { tab: tab, windowId: slot.windowId, created: slot.created };
    }

    async function describe(windowId) {
      try {
        const win = await windowsApi.get(windowId);
        return { window_id: windowId, focused: !!win.focused, state: win.state || "", width: win.width, height: win.height };
      } catch (error) {
        return { window_id: windowId, gone: true };
      }
    }

    async function closeAll() {
      const mine = await discover();
      let closed = 0;
      for (const w of mine) {
        try { await windowsApi.remove(w.windowId); closed += 1; } catch (error) { /* already gone */ }
      }
      await clearIdle();
      return closed;
    }

    // Run end: not closed at once — the next job (a healthcheck probe, a second client's run)
    // usually arrives within seconds and reuses the window. sweepIdle() from the poll loop
    // closes it once the linger has passed with nothing in flight. The marker lives in storage
    // so a service-worker restart between the two does not leak the window until the next run.
    async function markIdle() {
      if (!storage) return;
      try { await storage.set(IDLE_KEY, now()); } catch (error) { /* ignore */ }
    }
    async function clearIdle() {
      if (!storage) return;
      try { await storage.set(IDLE_KEY, 0); } catch (error) { /* ignore */ }
    }
    async function sweepIdle(lingerMs) {
      if (!storage) return { closed: 0, reason: "no_storage" };
      let since = 0;
      try { since = Number(await storage.get(IDLE_KEY)) || 0; } catch (error) { since = 0; }
      if (!since) return { closed: 0, reason: "not_idle" };
      const linger = typeof lingerMs === "number" ? lingerMs : DEFAULTS.lingerMs;
      if (now() - since < linger) return { closed: 0, reason: "lingering" };
      const mine = await discover();
      if (mine.some(function (w) { return w.busy; })) return { closed: 0, reason: "busy" };
      const closed = await closeAll();
      return { closed: closed, reason: closed ? "closed" : "nothing_open" };
    }

    return { discover, acquire, openTab, describe, closeAll, markIdle, clearIdle, sweepIdle, DEFAULTS, IDLE_KEY };
  }

  function clampInt(value, min, max, fallback) {
    const n = Number(value);
    if (!Number.isFinite(n)) return fallback;
    return Math.max(min, Math.min(max, Math.round(n)));
  }

  const api = { createCollectorWindowPool, DEFAULTS, IDLE_KEY };
  root.SoloCollectorWindow = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof self !== "undefined" ? self : globalThis);
