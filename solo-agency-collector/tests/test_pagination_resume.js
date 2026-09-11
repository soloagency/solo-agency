// Offline harness for RESUMABLE pagination.
//
// A connection bigger than one job's budget cannot be drained in one pass: 5,000 friends is ~623
// pages at 8 per page, while the page cap is 40 and the capability is killed at 45s. Legs are how
// it gets collected — each says where to start, and hands back where it stopped.
//
// The failure this file exists to catch is silent by construction. If a resuming leg still fetches
// the head page, every leg returns the same first rows and reports success; the run looks like
// sixteen collections of 336 friends instead of one collection of 5,000, and nothing in the record
// says so. So these tests assert on WHICH cursors were requested, not just on the row count.
const fs = require("fs");
const path = require("path");
const vm = require("vm");

const SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "platforms", "facebook", "gql_extract.js"), "utf8");

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail) : "")); }
}

// Eight rows per page, which is what a live run measured: 96 friends over a head page of 16 plus
// ten replayed pages of 8.
const PER_PAGE = 8;

// A synthetic friends connection of `total` people, paged the way Facebook pages one: each reply
// carries its own end_cursor, and has_next_page is the only end-of-connection signal.
function makeCtx(total, opts) {
  opts = opts || {};
  const asked = [];               // every cursor the walk requested, in order
  const inits = [];               // the fetch init of each request, in order
  function pageAt(offset) {
    const items = [];
    for (let i = offset; i < Math.min(offset + PER_PAGE, total); i++) {
      // The REAL shape, read off extractProfileFriends: edges[].node is the list-ITEM, and its
      // own .node is the friend entity carrying the id. A fixture shaped by guesswork produced
      // rows with empty ids, which the dedupe then discarded — the test failed for a reason that
      // had nothing to do with resuming.
      items.push({
        node: {
          title: { text: "Friend " + i },
          subtitle_text: { text: (i % 3) + " mutual friends" },
          url: "https://www.facebook.com/u" + i,
          node: { id: "u" + i, url: "https://www.facebook.com/u" + i },
        },
      });
    }
    const next = offset + PER_PAGE;
    return {
      data: { node: { pageItems: {
        edges: items,
        page_info: { end_cursor: next < total ? "cur:" + next : null, has_next_page: next < total },
      } } },
    };
  }
  const seed = {
    queryName: "FriendsListPaginationQuery", docId: "doc_friends", fbDtsg: "TOKEN", av: "1",
    url: "/api/graphql/",
    variables: { id: "P1", cursor: "cur:16", scale: 1 },
    response: pageAt(0),          // the captured page: rows 0-7, end_cursor cur:8
  };

  const ctx = {
    window: {},
    location: { href: "https://www.facebook.com/someone/friends", origin: "https://www.facebook.com", pathname: "/someone/friends", search: "" },
    document: { title: "", body: { innerText: "", innerHTML: "" }, querySelector: () => null, querySelectorAll: () => [] },
    setTimeout, clearTimeout, URL, URLSearchParams, console, AbortController,
    Date: fakeDate(opts.now),   // opts.now: a fake clock for the time-budget tests
    MutationObserver: function () { this.observe = () => {}; this.disconnect = () => {}; },
  };
  ctx.globalThis = ctx;
  vm.createContext(ctx);
  vm.runInContext(SRC, ctx);
  ctx.window.__soloGql = {
    captures: [seed],
    parseResponse: (t) => JSON.parse(t),
    origFetch: (url, o) => {
      const vars = JSON.parse(new URLSearchParams(o.body).get("variables"));
      // A null/absent cursor is the HEAD request — the one a resuming leg must never make.
      const cur = Object.prototype.hasOwnProperty.call(vars, "cursor") ? vars.cursor : null;
      asked.push(cur === null || cur === undefined ? "HEAD" : String(cur));
      inits.push(o);
      if (opts.onFetch) { const custom = opts.onFetch(cur, o); if (custom) return custom; }
      const off = cur == null ? 0 : parseInt(String(cur).split(":")[1], 10);
      return Promise.resolve({ text: () => Promise.resolve(JSON.stringify(pageAt(off))) });
    },
  };
  return { ctx, asked: () => asked, inits: () => inits };
}
// A Date whose now() is the test's fake clock; parse/UTC/construction still work.
function fakeDate(now) {
  if (!now) return Date;
  const F = function () { return new (Function.prototype.bind.apply(Date, [null].concat(Array.prototype.slice.call(arguments))))(); };
  F.now = now; F.parse = Date.parse; F.UTC = Date.UTC;
  return F;
}

async function run() {
  console.log("== leg 1 starts at the head and reports where it stopped ==");
  {
    const h = makeCtx(5000);
    const res = await h.ctx.window.__soloGqlPaginate("fb.profile.friends", { max_pages: 3 });
    check("the head page was fetched", h.asked()[0] === "HEAD", h.asked().slice(0, 2));
    check("it reports there is more", res.has_next_page === true && res.resumable === true, { n: res.has_next_page, r: res.resumable });
    check("it hands back an end_cursor", typeof res.end_cursor === "string" && res.end_cursor.length > 0, res.end_cursor);
    check("the budget cap is reported, not disguised as the end", res.page_cap_hit === true, res.page_cap_hit);
    check("start_cursor is null on the first leg", res.start_cursor === null, res.start_cursor);
  }

  console.log("\n== leg 2 resumes and NEVER refetches the head ==");
  {
    const h = makeCtx(5000);
    const res = await h.ctx.window.__soloGqlPaginate("fb.profile.friends", { max_pages: 3, start_cursor: "cur:800" });
    // This is the whole point. A resuming leg that still calls HEAD returns the same first rows
    // every time while reporting success, and the run silently collects nothing new.
    check("no HEAD request was made", h.asked().indexOf("HEAD") === -1, h.asked());
    check("the first request used the given cursor", h.asked()[0] === "cur:800", h.asked().slice(0, 2));
    check("the rows are from that offset, not from the start",
      (res.items || []).every((it) => parseInt(String(it.id).slice(1), 10) >= 800), (res.items || []).slice(0, 2));
    check("head_page_via says why it was skipped", res.head_page_via === "skipped_resuming", res.head_page_via);
    check("start_cursor is echoed back", res.start_cursor === "cur:800", res.start_cursor);
  }

  console.log("\n== consecutive legs cover the connection without gap or overlap ==");
  {
    const seenIds = new Set();
    let cursor = null, legs = 0, firstLeg = true;
    while (legs < 8) {
      const h = makeCtx(200);
      const inputs = { max_pages: 3 };
      if (cursor) inputs.start_cursor = cursor;
      const res = await h.ctx.window.__soloGqlPaginate("fb.profile.friends", inputs);
      (res.items || []).forEach((it) => seenIds.add(it.id));
      legs += 1;
      firstLeg = false;
      if (!res.has_next_page || !res.end_cursor) break;
      cursor = res.end_cursor;
    }
    check("the connection was drained in a few legs", legs <= 8 && legs > 1, legs);
    check("every row was collected exactly once", seenIds.size === 200, seenIds.size);
    // A leg boundary IS a page boundary, so there is nothing to dedupe — but if that ever stops
    // being true, this is where it shows.
    let missing = [];
    for (let i = 0; i < 200; i++) if (!seenIds.has("u" + i)) missing.push("u" + i);
    check("no gap between legs", missing.length === 0, missing.slice(0, 5));
    void firstLeg;
  }

  console.log("\n== the last leg says the connection ended ==");
  {
    const h = makeCtx(40);
    const res = await h.ctx.window.__soloGqlPaginate("fb.profile.friends", { max_pages: 40 });
    // has_next_page:false is the only honest end signal. A short page is not one, and neither is
    // page_cap_hit — that means the budget ran out with more waiting.
    check("has_next_page is false", res.has_next_page === false, res.has_next_page);
    check("resumable is false", res.resumable === false, res.resumable);
    check("the cap was NOT hit — it really ended", res.page_cap_hit === false, res.page_cap_hit);
    check("everyone was collected", (res.items || []).length === 40, (res.items || []).length);
  }


  console.log("\n== time budget: the generic pagination engine returns the pages in hand ==");
  {
    // Fake clock: the head page costs 2s of a 4s budget; page 2 is not started.
    let now = 1000000;
    const h = makeCtx(40, { now: () => now, onFetch: () => { now += 2000; return null; } });
    const res = await h.ctx.window.__soloGqlPaginate("fb.profile.friends", { max_pages: 5, time_budget_ms: 4000 });
    check("head page rows kept (8), no cursor page started", res.count === 8 && h.asked().length === 1 && h.asked()[0] === "HEAD", [res.count, h.asked()]);
    check("stopped_because time_budget, time_budget_hit, has_next_page + cursor kept, resumable", res.stopped_because === "time_budget" && res.time_budget_hit === true && res.has_next_page === true && res.end_cursor === "cur:8" && res.resumable === true, [res.stopped_because, res.time_budget_hit, res.has_next_page, res.end_cursor, res.resumable]);
    check("elapsed_ms / time_budget_ms reported; page_cap_hit stays false", res.elapsed_ms >= 2000 && res.time_budget_ms === 4000 && res.page_cap_hit === false, [res.elapsed_ms, res.time_budget_ms, res.page_cap_hit]);
  }
  {
    // A cursor page that hangs is aborted at the budget line: rows so far come back, the
    // aborted page stays owed (cursor kept, pages_fetched not counting it). Real clock, 3s.
    const h = makeCtx(24, { onFetch: (cur, o) => {
      if (cur === null) return null;   // the head page answers normally
      return new Promise((resolve, reject) => { if (o.signal) o.signal.addEventListener("abort", () => { const e = new Error("The operation was aborted"); e.name = "AbortError"; reject(e); }); });
    } });
    const t0 = Date.now();
    const res = await h.ctx.window.__soloGqlPaginate("fb.profile.friends", { max_pages: 5, time_budget_ms: 3000 });
    const took = Date.now() - t0;
    check("returned within the budget (" + took + "ms) with the head rows", took < 3700 && res.count === 8, [took, res.count]);
    check("aborted page not counted, cursor kept for the next leg, stopped_because time_budget", res.pages_fetched === 0 && res.end_cursor === "cur:8" && res.resumable === true && res.stopped_because === "time_budget", [res.pages_fetched, res.end_cursor, res.resumable, res.stopped_because]);
    check("both requests carried an abort signal", h.inits().length === 2 && h.inits().every((i) => !!i.signal), h.inits().map((i) => Object.keys(i)));
  }
  {
    // A genuine network failure in the last second of the budget is NOT relabelled time_budget.
    let now = 2000000;
    const h = makeCtx(24, { now: () => now, onFetch: (cur) => { if (cur === null) { now += 3200; return null; } return Promise.reject(new Error("net::ERR_CONNECTION_RESET")); } });
    const res = await h.ctx.window.__soloGqlPaginate("fb.profile.friends", { max_pages: 5, time_budget_ms: 6000 });
    check("net error keeps its own label: has_next_page false, no time_budget", res.time_budget_hit === undefined && res.stopped_because === "end_of_connection" && res.has_next_page === false, [res.stopped_because, res.time_budget_hit, res.has_next_page]);
  }
  {
    // No budget: unchanged behaviour and no signal on requests.
    const h = makeCtx(24, {});
    const res = await h.ctx.window.__soloGqlPaginate("fb.profile.friends", { max_pages: 5 });
    check("no budget => all 24 rows, no time_budget_ms, no abort signal", res.count === 24 && res.time_budget_ms === undefined && h.inits().every((i) => !i.signal) && res.stopped_because === "end_of_connection", [res.count, res.time_budget_ms, res.stopped_because]);
  }

  console.log("\n" + (fail === 0 ? "ALL " + pass + " CHECKS PASSED" : pass + " passed, " + fail + " FAILED"));
  process.exit(fail === 0 ? 0 : 1);
}

run().catch((e) => { console.error("harness crashed:", e); process.exit(1); });
