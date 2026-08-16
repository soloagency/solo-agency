// Offline harness for fb.profile.dossier — the one-visit profile walk.
//
// The point of this capability is that ONE tab load replaces two (fb.profile.header +
// fb.profile.contacts) and additionally opens "Work and education", which neither of them
// ever visited. That tab is where a profile prints a JOB TITLE, and the title — not the
// employer — is what tells apart "Loan Officer at Wells Fargo" (Loan & Mortgage) from
// "Financial Advisor at Wells Fargo" (Banking & Financial).
//
// So this file drives a FAKE NAVIGABLE DOM: clicking a tab anchor swaps the page text, the
// way Facebook's SPA does. Without that, a test can only prove the function returns an
// object — not that the ladder actually walks, actually stops on its budget, and actually
// keeps the text it paid to load.
const fs = require("fs");
const path = require("path");
const vm = require("vm");

const SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "gql_extract.js"), "utf8");

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail) : "")); }
}

// Furniture Facebook repeats on every single tab. The delta harvest must not restate it:
// five copies of the nav would bury the handful of lines that are the actual answer.
const CHROME = ["Facebook", "Search", "Home", "Marketplace", "Notifications", "Menu"];

function page(lines) { return CHROME.concat(lines).join("\n"); }

// A profile whose trade is stated ONLY as a job title, on the Work and education tab —
// exactly the case fb.profile.header cannot reach and fb.profile.contacts threw away.
const PAGES = {
  main: page(["Claire Hanh Lam", "1.2K followers", "Works at ZenWealth Solutions", "Lives in Houston, Texas"]),
  about: page(["About"]),
  contact_info: page(["Email", "claire@zenwealthsolutions.com", "Website", "zenwealthsolutions.com"]),
  work: page(["Work", "Loan Officer at Wells Fargo", "Mortgage Advisor at ZenWealth Solutions"]),
  education: page(["College", "Studied at University of Houston"]),
  intro: page(["Helping families finance their first home with clarity"]),
  personal_details: page(["Speaks English, Vietnamese"]),
};

// The slugs a live run actually observed on five personal profiles. The set this replaces was
// invented and only `intro` was real — a fake DOM built on invented slugs happily proved a walk
// that could not work.
// The slugs a live run actually observed on five personal profiles. The set this replaces was
// invented — contact_info / work_and_education / basic_info / links — and only `intro` was real.
// A fake DOM built on invented slugs happily proved a walk that could not work: Work and
// Education are TWO tabs on Facebook, not one. Keys here match what discovery derives from the
// slug, so the fake and the code cannot drift apart on naming.
const TAB_SLUG = {
  contact_info: "directory_contact_info",
  work: "directory_work",
  education: "directory_education",
  intro: "directory_intro",
  personal_details: "directory_personal_details",
};

// `offers` decides which tabs this synthetic profile actually exposes — a profile that does
// not publish Work and education must land in `missing`, never silently in `checked`.
function makeCtx(opts) {
  opts = opts || {};
  const offers = opts.offers || Object.keys(TAB_SLUG);
  const pages = opts.pages || PAGES;
  const state = { current: "main", clicks: [] };
  // Facebook fires a named GraphQL query when a sub-tab renders. The probe exists to learn
  // WHICH one, so the fake has to produce them or the test proves nothing.
  const captures = [];
  let seeMoreLeft = opts.seeMore === undefined ? 1 : opts.seeMore;

  function anchor(href, text, onClick) {
    return {
      innerText: text,
      getAttribute: (a) => (a === "href" ? href : null),
      getBoundingClientRect: () => ({ width: 100, height: 20 }),
      click: onClick,
    };
  }
  function navTo(key) {
    return () => {
      state.current = key; state.clicks.push(key);
      captures.push({ queryName: "ProfileCometAbout" + key + "Query", docId: "doc_" + key,
                      variables: { id: "1369773994", scale: 1 }, fbDtsg: "TOKEN" });
    };
  }

  const document = {
    title: "Claire Hanh Lam | Facebook",
    get body() { return { innerText: pages[state.current] || "", innerHTML: "<div>" + (pages[state.current] || "") + "</div>" }; },
    querySelector: (sel) => {
      if (/role="main"/.test(sel)) return null;   // fall back to document-wide search
      return null;                                 // no og:title meta
    },
    querySelectorAll: (sel) => {
      // the About entry link
      if (/sk=about|\/about/.test(sel) && state.current === "main") return [anchor("/claire/about", "About", navTo("about"))];
      // a tab addressed by its href slug
      const m = sel.match(/href\*="([^"]+)"/);
      if (m) {
        const slug = m[1];
        const key = Object.keys(TAB_SLUG).find((k) => TAB_SLUG[k] === slug);
        if (key && offers.indexOf(key) !== -1) return [anchor("/claire/" + slug, key, navTo(key))];
        return [];
      }
      // profileHeader's external-link scan
      if (/^a\[href\^="http"\]$/.test(sel)) return [];
      // discovery: every About link on the page, which is how the walk now builds its plan
      if (sel === "a[href]") {
        if (state.current === "main") return [anchor("/claire/about", "About", navTo("about"))];
        return offers.map((k) => anchor("/claire/" + TAB_SLUG[k], k, navTo(k)));
      }
      // the name heading
      if (/h1/.test(sel)) {
        const first = (pages[state.current] || "").split("\n").find((l) => /Claire/.test(l));
        return first ? [{ innerText: first }] : [];
      }
      // the "See more" control that truncates a bio
      if (/role="button"/.test(sel) && seeMoreLeft > 0) {
        return [{
          innerText: "See more",
          getAttribute: () => null,
          getBoundingClientRect: () => ({ width: 60, height: 16 }),
          click: () => {
            seeMoreLeft -= 1;
            captures.push({ queryName: "CometTextWithEntitiesSeeMoreQuery", docId: "doc_seemore",
                            variables: { id: "x" }, fbDtsg: "TOKEN" });
          },
        }];
      }
      // label-fallback / CTA sweep — no extra nodes, the href path already covers the tabs
      return [];
    },
  };

  const ctx = {
    window: {},
    location: { href: "https://www.facebook.com/claire", origin: "https://www.facebook.com", pathname: "/claire", search: "" },
    document,
    setTimeout, clearTimeout, URL, console, Date,
    URLSearchParams,
    MutationObserver: function () { this.observe = () => {}; this.disconnect = () => {}; },
  };
  ctx.globalThis = ctx;
  vm.createContext(ctx);
  vm.runInContext(SRC, ctx);
  ctx.window.__soloGql = { captures, origFetch: null };
  return { ctx, state };
}

// settle_ms: 1 makes settleThenScan return after a single 350ms tick, so the whole ladder
// runs in seconds instead of half a minute. It does not change which tabs get walked.
const FAST = { settle_ms: 1 };

async function run() {
  console.log("== the ladder walks every tab, and does not stop at the first email ==");
  {
    const { ctx, state } = makeCtx();
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST);
    const it = (res.items || [])[0] || {};
    check("the record is available even before judging content", res.available === true, res.available);
    check("all five About tabs were opened", ["contact_info", "work", "education", "intro", "personal_details"].every((k) => (it.checked || []).indexOf(k) !== -1), it.checked);
    // contact_info comes first and carries an address; the old ladder stopped right here and
    // never saw the job title two tabs later.
    check("it did NOT stop after the tab that had the email", (it.checked || []).indexOf("work") !== -1, it.checked);
    check("nothing was reported missing", (it.missing || []).length === 0, it.missing);
    check("the sub-nav was DISCOVERED, not guessed", (it.discovered_tabs || []).length === 5, it.discovered_tabs);
    check("the budget was not exhausted", it.budget_exhausted === false, it.budget_exhausted);
  }

  console.log("\n== the job title survives — the whole reason this capability exists ==");
  {
    const { ctx } = makeCtx();
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST);
    const it = (res.items || [])[0] || {};
    const we = (it.about && it.about.work) || [];
    check("the Work and education tab kept its text", we.some((l) => /Loan Officer at Wells Fargo/.test(l)), we);
    check("the title is in the flat about_lines an agent reads", (it.about_lines || []).some((l) => /Loan Officer/.test(l)), (it.about_lines || []).slice(0, 8));
    // This is the distinction the 42-value dictionary turns on: the employer alone cannot
    // separate Loan & Mortgage from Banking & Financial, so `work` is not enough on its own.
    check("`work` still only holds the employer, not the title", (it.work || []).some((w) => /ZenWealth/.test(w)) && !(it.work || []).some((w) => /Loan Officer/.test(w)), it.work);
  }

  console.log("\n== contact data is collected across tabs, not just the first hit ==");
  {
    const { ctx } = makeCtx();
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST);
    const it = (res.items || [])[0] || {};
    check("the published address was found", (it.emails || []).indexOf("claire@zenwealthsolutions.com") !== -1, it.emails);
    check("found_on names the surface it came from", it.found_on === "contact_info", it.found_on);
    check("outbound links were gathered", (it.websites || []).length > 0, it.websites);
  }

  console.log("\n== page furniture is not restated once per tab ==");
  {
    const { ctx } = makeCtx();
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST);
    const it = (res.items || [])[0] || {};
    const marketplace = (it.about_lines || []).filter((l) => l === "Marketplace").length;
    check("a nav line repeated on all 6 pages appears at most once", marketplace <= 1, marketplace);
    check("about_lines stays small enough for an agent to read", (it.about_lines || []).length <= 120, (it.about_lines || []).length);
  }

  console.log("\n== a tab the profile does not offer is reported, not silently skipped ==");
  {
    const { ctx } = makeCtx({ offers: ["contact_info", "intro"] });
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST);
    const it = (res.items || [])[0] || {};
    check("the tabs that exist were opened", (it.checked || []).indexOf("contact_info") !== -1, it.checked);
    // Since the plan is DISCOVERED, three states stay distinct instead of collapsing into one
    // "missing" bucket: discovered_tabs says what the profile publishes, checked says what
    // opened, and missing is reserved for a tab that WAS found and then failed or was cut off
    // by the budget. A reader asking "did we look at their work history?" reads discovered_tabs
    // — answering that from `missing` alone is what made the first live run unreadable.
    check("a tab the profile does not publish is absent from discovered_tabs",
      (it.discovered_tabs || []).every((s) => !/directory_work\b/.test(s)), it.discovered_tabs);
    check("discovered_tabs lists exactly what this profile does publish",
      (it.discovered_tabs || []).length === 2, it.discovered_tabs);
    check("missing stays empty — nothing was found and then lost", (it.missing || []).length === 0, it.missing);
    check("the record is still available", res.available === true, res.available);
  }

  console.log("\n== the internal deadline returns a partial record instead of dying at 45s ==");
  {
    const { ctx } = makeCtx();
    // Too small for even one tab: the walk must stop before the first click, not mid-tab.
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", { settle_ms: 1, budget_ms: 1 });
    const it = (res.items || [])[0] || {};
    check("the record still exists", res.available === true && (res.items || []).length === 1, res.available);
    check("it says the budget ran out", it.budget_exhausted === true, it.budget_exhausted);
    check("every unvisited tab is listed", ["contact_info", "work", "education", "intro", "personal_details"].every((k) => (it.missing || []).indexOf(k) !== -1), it.missing);
    check("the header read on the landing page is still there", !!it.name, it.name);
    check("elapsed_ms is reported", typeof it.elapsed_ms === "number", it.elapsed_ms);
  }

  console.log("\n== an empty profile returns a record, never a null ==");
  {
    const blank = { main: page([]), about: page([]) };
    const { ctx } = makeCtx({ offers: [], pages: blank });
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST);
    // available:false makes background.js null the whole record, which is how "the capability
    // crashed" and "this profile says nothing" became the same output.
    check("available stays true so the empty result is distinguishable from a crash", res.available === true, res.available);
    check("the audit trail shows the pass actually ran", (res.checked || []).indexOf("main") !== -1, res.checked);
  }

  console.log("\n== an unreadable name does not cancel the About walk ==");
  {
    // profileHeader reports available:false when it cannot read a name. That is NOT a reason to
    // abandon the profile — it is the case where the About tabs matter most. Only the
    // self-profile guard may stop the walk.
    const { ctx } = makeCtx({ pages: Object.assign({}, PAGES, { main: page(["Works at ZenWealth Solutions"]) }) });
    ctx.document.title = "";                       // no title to fall back on
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST);
    const it = (res.items || [])[0] || {};
    check("the walk still ran", (it.checked || []).indexOf("work") !== -1, it.checked);
    check("the job title was still collected", (it.about_lines || []).some((l) => /Loan Officer/.test(l)), (it.about_lines || []).slice(0, 6));
    check("the record was not replaced by the header's failure envelope", res.capability === "fb.profile.dossier", res.capability);
  }

  console.log("\n== the walk reports which GraphQL query each surface fired ==");
  {
    const { ctx } = makeCtx({ seeMore: 0 });
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST);
    const it = (res.items || [])[0] || {};
    const g = it.graphql_by_surface || {};
    // The point of the probe: a surface whose content arrives in ONE named query can be
    // replayed by doc_id instead of clicked, the way fb.profile.hovercard already is.
    check("a query name is recorded for a tab that opened",
      (g.work || []).some((r) => /ProfileCometAbout/.test(r.query)), g.work);
    check("its doc_id comes along, since that is what replay needs",
      (g.work || []).every((r) => !!r.doc_id), g.work);
    check("variables are withheld unless asked for",
      (g.work || []).every((r) => r.variables === undefined), g.work);
  }

  console.log("\n== probe_graphql adds what a replay would need ==");
  {
    const { ctx } = makeCtx({ seeMore: 0 });
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", { settle_ms: 1, probe_graphql: true });
    const it = (res.items || [])[0] || {};
    const rows = (it.graphql_by_surface || {}).work || [];
    check("variables are captured", rows.some((r) => /1369773994/.test(r.variables || "")), rows);
    check("the auth token's presence is reported, never the token", rows.every((r) => r.has_fb_dtsg === true && !/TOKEN/.test(JSON.stringify(r))), rows);
  }

  console.log("\n== a truncated bio gets expanded on every surface, not just the entry page ==");
  {
    const { ctx } = makeCtx({ seeMore: 3 });
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST);
    const it = (res.items || [])[0] || {};
    check("See more was clicked more than once across the walk", (it.see_more_expansions || 0) >= 2, it.see_more_expansions);
    check("the expansion's own query is recorded",
      Object.keys(it.graphql_by_surface || {}).some((k) => (it.graphql_by_surface[k] || []).some((r) => /SeeMore/.test(r.query))), it.graphql_by_surface);
  }

  console.log("\n== the operator's own profile IS refused ==");
  {
    const { ctx } = makeCtx();
    ctx.location.pathname = "/me";                 // the guard's trigger
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST);
    check("it refuses instead of returning the operator as a lead", res.available === false && res.reason === "self_profile", { a: res.available, r: res.reason });
  }

  console.log("\n" + (fail === 0 ? "ALL " + pass + " CHECKS PASSED" : pass + " passed, " + fail + " FAILED"));
  process.exit(fail === 0 ? 0 : 1);
}

run().catch((e) => { console.error("harness crashed:", e); process.exit(1); });
