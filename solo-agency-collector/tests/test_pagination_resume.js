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

const SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "gql_extract.js"), "utf8");

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
    setTimeout, clearTimeout, URL, URLSearchParams, console, Date,
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
      const off = cur == null ? 0 : parseInt(String(cur).split(":")[1], 10);
      return Promise.resolve({ text: () => Promise.resolve(JSON.stringify(pageAt(off))) });
    },
  };
  return { ctx, asked: () => asked };
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

  console.log("\n" + (fail === 0 ? "ALL " + pass + " CHECKS PASSED" : pass + " passed, " + fail + " FAILED"));
  process.exit(fail === 0 ? 0 : 1);
}

run().catch((e) => { console.error("harness crashed:", e); process.exit(1); });
