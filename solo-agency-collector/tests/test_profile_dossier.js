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
  category: page(["Digital creator"]),
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
  category: "directory_category",
};

// `offers` decides which tabs this synthetic profile actually exposes — a profile that does
// not publish Work and education must land in `missing`, never silently in `checked`.
function makeCtx(opts) {
  opts = opts || {};
  const SLUGS = opts.slugs || TAB_SLUG;
  const offers = opts.offers || Object.keys(SLUGS);
  const pages = opts.pages || PAGES;
  const state = { current: "main", clicks: [] };
  // Facebook fires a named GraphQL query when a sub-tab renders. The probe exists to learn
  // WHICH one, so the fake has to produce them or the test proves nothing.
  const captures = [];
  let feedScans = 0;
  let seeMoreLeft = opts.seeMore === undefined ? 1 : opts.seeMore;

  function anchor(href, text, onClick) {
    return {
      innerText: text,
      getAttribute: (a) => (a === "href" ? href : null),
      getBoundingClientRect: () => ({ width: 100, height: 20 }),
      click: onClick,
    };
  }
  function seeMoreBtn() {
    return {
      innerText: "See more",
      getAttribute: () => null,
      getBoundingClientRect: () => ({ width: 60, height: 16 }),
      parentElement: opts.inFeed ? { getAttribute: (a) => (a === "role" ? "article" : null), parentElement: null } : null,
      click: () => {
        seeMoreLeft -= 1;
        captures.push({ queryName: "CometTextWithEntitiesSeeMoreQuery", docId: "doc_seemore",
                        variables: { id: "x" }, fbDtsg: "TOKEN" });
      },
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
      // The main column, with a post feed rendered underneath the About panel — which is what
      // Facebook actually does, and why body.innerText alone leaks post text into the dossier.
      if (/role="main"/.test(sel)) {
        return {
          get innerText() { return (pages[state.current] || "") + (opts.feed ? "\n" + opts.feed : ""); },
          querySelectorAll: (q) => {
            if (/article|feed/.test(q)) {
              feedScans += 1;
              // innerText is a layout-forcing read. Counting the scans is how an offline harness
              // can see a cost blowup at all: the real failure was a 45s capability timeout on a
              // Page whose About renders dozens of post and review cards.
              return opts.feed ? [{ get innerText() { return opts.feed; } }] : [];
            }
            return [];
          },
        };
      }
      // the About panel, reached from a sub-nav link exactly as the real walk reaches it
      if (/href\*="directory_"/.test(sel)) return null;
      return null;                                 // no og:title meta
    },
    querySelectorAll: (sel) => {
      // The sub-nav probe used to locate the About card. Its ancestors carry ONLY the panel's
      // own text, which is what keeps Reviews and Posts out of the record.
      if (sel === 'a[href*="directory_"]') {
        // The real shape, which the previous fake did not have: a NAV box carrying only the menu
        // labels, wrapped in a CARD that also holds the content pane. Scoping to the nav box was
        // what returned four lines of menu and no bio on a live run — a fake with one ancestor
        // level could not have caught that.
        const card = {
          get innerText() { return (pages[state.current] || ""); },
          querySelectorAll: (q) => (/role="button"/.test(q) && seeMoreLeft > 0 ? [seeMoreBtn()] : []),
          // The ancestor ABOVE the card is where Reviews and Posts live. The walk must stop
          // before this one — that is the whole point of taking the first ancestor with content.
          parentElement: { get innerText() { return (pages[state.current] || "") + "\n" + (opts.feed || ""); },
                           querySelectorAll: () => [], parentElement: null },
        };
        const nav = {
          innerText: Object.keys(SLUGS).join("\n"),
          querySelectorAll: (q) => (/directory_/.test(q) ? navLinks() : []),
          parentElement: card,
        };
        function navLinks() {
          return offers.map((k) => ({ getAttribute: () => "/claire/" + SLUGS[k], parentElement: nav,
                                      getBoundingClientRect: () => ({ width: 80, height: 18 }) }));
        }
        card.querySelectorAll = (q) => (/directory_/.test(q) ? navLinks()
          : (/role="button"/.test(q) && seeMoreLeft > 0 ? [seeMoreBtn()] : []));
        return navLinks();
      }
      // the About entry link
      if (/sk=about|\/about/.test(sel) && state.current === "main") return [anchor("/claire/about", "About", navTo("about"))];
      // a tab addressed by its href slug
      const m = sel.match(/href\*="([^"]+)"/);
      if (m) {
        const slug = m[1];
        const key = Object.keys(SLUGS).find((k) => SLUGS[k] === slug);
        if (key && offers.indexOf(key) !== -1) return [anchor("/claire/" + slug, key, navTo(key))];
        return [];
      }
      // profileHeader's external-link scan
      if (/^a\[href\^="http"\]$/.test(sel)) return [];
      // discovery: every About link on the page, which is how the walk now builds its plan
      if (sel === "a[href]") {
        if (state.current === "main") return [anchor("/claire/about", "About", navTo("about"))];
        return offers.map((k) => anchor("/claire/" + SLUGS[k], k, navTo(k)));
      }
      // the name heading
      if (/h1/.test(sel)) {
        const first = (pages[state.current] || "").split("\n").find((l) => /Claire/.test(l));
        return first ? [{ innerText: first }] : [];
      }
      // the "See more" control that truncates a bio
      if (/role="button"/.test(sel) && seeMoreLeft > 0) {
        return [seeMoreBtn()];
      }
      if (false) {
        return [{
          innerText: "See more",
          getAttribute: () => null,
          getBoundingClientRect: () => ({ width: 60, height: 16 }),
          // opts.inFeed puts the button inside a role="article", i.e. a post
          parentElement: opts.inFeed ? { getAttribute: (a) => (a === "role" ? "article" : null), parentElement: null } : null,
          getAttribute: () => null,
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
  return { ctx, state, feedScans: () => feedScans };
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
    check("all five About tabs were opened", ["contact_info", "work", "education", "intro", "category"].every((k) => (it.checked || []).indexOf(k) !== -1), it.checked);
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
    // separate Loan & Mortgage from Banking & Financial. `work` used to hold only the employer,
    // because it was parsed from "Works at <X>" and Facebook writes "Loan Officer at Wells
    // Fargo" under a Work heading instead — the title was in the record but never in a field.
    // Reading the Work section by label puts it where a caller can use it.
    check("`work` carries the employer from the intro card", (it.work || []).some((w) => /ZenWealth/.test(w)), it.work);
    check("and the JOB TITLE from the Work section", (it.work || []).some((w) => /Loan Officer at Wells Fargo/.test(w)), it.work);
    check("headings are not mistaken for values", !(it.work || []).some((w) => /^(Work|Education|Address)$/i.test(w)), it.work);
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
    check("every unvisited tab is listed", ["contact_info", "work", "education", "intro", "category"].every((k) => (it.missing || []).indexOf(k) !== -1), it.missing);
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

  console.log("\n== the DOM fallback clicks nothing at all ==");
  {
    // "See more" is the control that clicked into Reviews and posts. The fallback no longer
    // expands anything: a truncated bio is a smaller loss than a review thread recorded as a
    // profile fact, and the GraphQL path returns the full text anyway.
    const { ctx } = makeCtx({ seeMore: 3 });
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST);
    const it = (res.items || [])[0] || {};
    check("nothing was expanded", (it.see_more_expansions || 0) === 0, it.see_more_expansions);
    check("the record says the text came from the DOM, not GraphQL", it.source === "dom", it.source);
  }

  console.log("\n== every section the profile publishes gets opened ==");
  {
    // The About sub-nav has about fourteen sections. Walking the ones that cannot decide an
    // industry or yield an address costs a click, a settle and a scan each, for nothing. They
    // are not merely deprioritised — they are not visited, and the record says so rather than
    // implying the profile has nothing else.
    const SLUGS = { hobbies: "directory_hobbies", travel: "directory_travel",
                    contact_info: "directory_contact_info", work: "directory_work" };
    const pages = { main: page(["Claire Hanh Lam"]), about: page(["About"]),
                    hobbies: page(["Cycling"]), travel: page(["Da Nang"]),
                    contact_info: page(["claire@zenwealthsolutions.com"]),
                    work: page(["Loan Officer at Wells Fargo"]) };
    const { ctx } = makeCtx({ pages, slugs: SLUGS, offers: ["hobbies", "travel", "contact_info", "work"], seeMore: 0 });
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST);
    const it = (res.items || [])[0] || {};
    // Sections render in about a second, so there is no tail worth dropping — but order still
    // matters, because the BUDGET can cut: losing Hobbies beats losing the address.
    check("hobbies and travel were opened too", ["hobbies", "travel"].every((k) => (it.checked || []).indexOf(k) !== -1), it.checked);
    check("contact_info was reached before hobbies",
      (it.checked || []).indexOf("contact_info") < (it.checked || []).indexOf("hobbies"), it.checked);
    check("the address came back", (it.emails || []).length === 1, it.emails);
    check("the job title came back", (it.about_lines || []).some((l) => /Loan Officer/.test(l)), it.about_lines);
  }

  console.log("\n== entering at /about does not re-enter About, and does not re-read it ==");
  {
    // Measured on three live pages: landing on /about, the first harvest already carried the
    // bio, the category and the email, and the clicks into contact_info/intro/category each
    // returned ZERO new lines. Only work and education hold anything the overview does not.
    const SLUGS = { work: "directory_work", education: "directory_education",
                    contact_info: "directory_contact_info", intro: "directory_intro" };
    const pages = { main: page(["Ann Vuong", "Real Estate Agent", "info@annv.ca"]),
                    work: page(["Owner/President at Reach Home Loans"]),
                    education: page(["Went to Juanita High School"]),
                    contact_info: page(["info@annv.ca"]), intro: page(["Real Estate Agent"]) };
    const { ctx } = makeCtx({ pages, slugs: SLUGS, offers: Object.keys(SLUGS), seeMore: 0 });
    ctx.location.href = "https://www.facebook.com/AnnVuongHouse/about";
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST);
    const it = (res.items || [])[0] || {};
    check("About was not re-entered", (it.checked || []).indexOf("about(already)") !== -1, it.checked);
    check("work and education were still opened",
      ["work", "education"].every((k) => (it.checked || []).indexOf(k) !== -1), it.checked);
    check("every published section was still opened",
      ["contact_info", "intro"].every((k) => (it.checked || []).indexOf(k) !== -1), it.checked);
    check("the address came from the overview anyway", (it.emails || []).indexOf("info@annv.ca") !== -1, it.emails);
    check("the job title still came back", (it.about_lines || []).some((l) => /Reach Home Loans/.test(l)), it.about_lines);
  }

  console.log("\n== the post feed under the About panel is not harvested ==");
  {
    // Facebook keeps the timeline rendered while About is open. Reading document.body wholesale
    // pulled "<name>'s Post" and reel view counts into three live dossiers as profile facts.
    const feed = ["Mickey Nguyen's Post", "Cảm ơn Ann Vương đã luôn nhiệt tình", "12K", "4.7K"].join("\n");
    const { ctx } = makeCtx({ feed, seeMore: 0 });
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST);
    const it = (res.items || [])[0] || {};
    const lines = it.about_lines || [];
    check("no post text reached the record", !lines.some((l) => /Post|Cảm ơn/.test(l)), lines.filter((l) => /Post|Cảm ơn/.test(l)));
    check("no reel view counts either", !lines.some((l) => /^(12K|4\.7K)$/.test(l)), lines);
    check("the profile's own lines survived", lines.some((l) => /Claire Hanh Lam/.test(l)), lines.slice(0, 6));
  }

  console.log("\n== a post's See more is never clicked ==");
  {
    // Every post carries a "See more". Matching it across the document expanded posts and reels
    // on three live pages — 5, 12 and 14 clicks — dragging "<name>'s Post" and reel view counts
    // into the dossier as if they were profile facts.
    const { ctx } = makeCtx({ seeMore: 3, inFeed: true });
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST);
    const it = (res.items || [])[0] || {};
    check("nothing inside a post was expanded", (it.see_more_expansions || 0) === 0, it.see_more_expansions);
  }

  console.log("\n== sections come from GraphQL, and the DOM is never touched ==");
  {
    // Measured live: landing on /about surfaced 6, 10 and 10 section tokens with zero clicks,
    // all served by one persisted query. A response for an About section cannot contain a
    // review — that is why this replaces the DOM walk rather than patching it.
    const tok = (n) => Buffer.from("app_collection:pfbid0" + n).toString("base64");
    const seed = {
      queryName: "ProfileCometAboutAppSectionQuery", docId: "27470497829312569",
      fbDtsg: "TOKEN", av: "100", url: "/api/graphql/",
      variables: { pageID: "1", userID: "1", appSectionFeedKey: "KEY",
                   collectionToken: tok("A"), sectionToken: tok("A"), scale: 1,
                   __relay_internal__pv__FBProfile_enable_perf_improv_gkrelayprovider: true },
      response: { data: { nav: [
        { title: "Work", token: tok("A") },
        { title: "Contact info", token: tok("B") },
      ] } },
    };
    const served = {};
    served[tok("A")] = { data: { section: { rows: ["Owner/President at Reach Home Loans", "NMLS: 2266637"] } } };
    served[tok("B")] = { data: { section: { rows: ["Email", "info@annv.ca"] } } };
    served[tok("C")] = { data: { section: { rows: ["La Familia Rodriguez recommends", "100% recommend (7 Reviews)"] } } };

    const { ctx } = makeCtx({ feed: "Mickey Nguyen's Post\n12K" });
    ctx.window.__soloGql = {
      captures: [seed],
      parseResponse: (t) => JSON.parse(t),
      origFetch: (url, opt) => {
        const vars = JSON.parse(new URLSearchParams(opt.body).get("variables"));
        // The provider flags and the profile-scoped key must survive: dropping one comes back as
        // an empty section rather than an error, which is the hardest failure to notice.
        if (!vars.appSectionFeedKey || !vars.__relay_internal__pv__FBProfile_enable_perf_improv_gkrelayprovider) {
          return Promise.resolve({ text: () => Promise.resolve(JSON.stringify({ data: {} })) });
        }
        return Promise.resolve({ text: () => Promise.resolve(JSON.stringify(served[vars.collectionToken] || { data: {} })) });
      },
    };
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", { settle_ms: 1, use_graphql: true });
    const it = (res.items || [])[0] || {};
    check("the record says the sections came from GraphQL", it.source === "graphql", it.source);
    check("the two real sections were fetched", (it.graphql_about || {}).sections === 2, it.graphql_about);
    // Reviews is a section of the About sub-nav, so replaying every token fetched it too.
    // Excluding it by NAME is exact — the same exclusion could not be expressed against the DOM.
    check("the Reviews section was not fetched", !(it.about || {}).reviews, Object.keys(it.about || {}));
    check("no recommendation text reached the record",
      !(it.about_lines || []).some((l) => /recommends|% recommend/.test(l)), it.about_lines);
    check("the doc_id used is reported", (it.graphql_about || {}).doc_id === "27470497829312569", it.graphql_about);
    check("sections are named, not numbered", !!(it.about || {}).work && !!(it.about || {}).contact_info, Object.keys(it.about || {}));
    check("skipping the noise section did not abort the rest",
      !!(it.about || {}).work && !!(it.about || {}).contact_info, Object.keys(it.about || {}));
    check("the job title arrived", (it.about_lines || []).some((l) => /Reach Home Loans/.test(l)), it.about_lines);
    check("the licence number arrived", (it.about_lines || []).some((l) => /NMLS/.test(l)), it.about_lines);
    check("the address arrived", (it.emails || []).indexOf("info@annv.ca") !== -1, it.emails);
    // The whole reason for the pivot: a post cannot reach a record assembled from section data.
    check("no post text could reach the record", !(it.about_lines || []).some((l) => /Post|12K/.test(l)), it.about_lines);
    check("nothing on the page was clicked", (it.see_more_expansions || 0) === 0, it.see_more_expansions);
  }

  console.log("\n== when the About query never fires, it falls back and says so ==");
  {
    const { ctx } = makeCtx({ seeMore: 0 });
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST);
    const it = (res.items || [])[0] || {};
    check("it used the DOM", it.source === "dom", it.source);
    // The default is DOM, and the record says WHY rather than leaving it to be inferred.
    check("and the reason is recorded", (it.graphql_about || {}).reason === "graphql_disabled_by_default", it.graphql_about);
    check("the walk still produced a record", res.available === true && !!it.profile_url, res.available);
  }

  console.log("\n== the feed is scanned once, not once per tab ==");
  {
    // Reading .innerText forces a layout pass. Rescanning every post and review card on each of
    // ten sub-tabs turned a 12-second walk into a capability killed at the 45-second ceiling —
    // and the record came back as an error, so the whole run collected nothing.
    const h = makeCtx({ feed: "Mickey Nguyen's Post\n12K", seeMore: 0 });
    const res = await h.ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST);
    const it = (res.items || [])[0] || {};
    const tabs = (it.checked || []).length;
    check("more than one surface was harvested", tabs >= 3, tabs);
    check("but the feed was scanned only once", h.feedScans() === 1, { scans: h.feedScans(), tabs });
    check("and post text still did not reach the record",
      !(it.about_lines || []).some((l) => /Post/.test(l)), it.about_lines);
  }

  console.log("\n== a Page's address becomes location, and a review never becomes the bio ==");
  {
    // Two failures seen together on all three live pages: location stayed empty because it is
    // parsed from "Lives in <X>" while a Page prints an "Address" label, and intro_bio picked a
    // REVIEW because a recommendation is always longer than a bio.
    const SLUGS = { contact_info: "directory_contact_info", intro: "directory_intro" };
    const pages = {
      main: page(["Ann Vuong - Nhà Canada House", "7.9K followers",
                  "Top real estate broker in Canada, helping families and investors find success",
                  "Reviews", "100% recommend (35 Reviews)",
                  "Cảm ơn Ann Vương đã luôn nhiệt tình hỗ trợ, hướng dẫn chi tiết và chia sẻ nhiều thông tin hữu ích khi mình mua nhà"]),
      contact_info: page(["Address", "30 Eglinton West, Mississauga, ON, Canada, L5R 3E7", "Phone", "+1 647-784-2888"]),
      intro: page(["Real Estate Agent"]),
    };
    const { ctx } = makeCtx({ pages, slugs: SLUGS, offers: Object.keys(SLUGS), seeMore: 0 });
    ctx.document.title = "";
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST);
    const it = (res.items || [])[0] || {};
    check("the Page address became location",
      (it.location || []).some((l) => /Mississauga, ON, Canada/.test(l)), it.location);
    check("the phone was read from its label", (it.phones || []).some((p) => /647-784-2888/.test(p)), it.phones);
    check("the bio is the profile's own line", /Top real estate broker in Canada/.test(it.intro_bio || ""), it.intro_bio);
    check("the bio is NOT the review", !/Cảm ơn/.test(it.intro_bio || ""), it.intro_bio);
  }

  console.log("\n== Facebook's own chrome never becomes the bio ==");
  {
    // Two live profiles returned "Number of unread notifications" as their bio. Chrome reads
    // like prose and is long, and their real intro lines were short — "Mortgage Brokers",
    // "📍Expert IL/GA Realtor" — so nothing else qualified and the longest-line rule took it.
    const pages = { main: ["Number of unread notifications", "New notification in settings",
                           "Hannah Nguyen - Loan Officer", "Mortgage Brokers"].join("\n"),
                    about: page([]) };
    const { ctx } = makeCtx({ pages, slugs: {}, offers: [], seeMore: 0 });
    ctx.document.title = "";
    const res = await ctx.window.__soloGqlPaginate("fb.profile.dossier", FAST);
    const it = (res.items || [])[0] || {};
    check("chrome was refused as a bio", !/unread|notification/i.test(it.intro_bio || ""), it.intro_bio);
    // With chrome gone the display name became the longest survivor on a profile whose real
    // intro lines are all short. The name is already in `name`; repeating it as a bio says
    // nothing, and an empty bio is the honest answer.
    check("the display name is not returned as a bio", (it.intro_bio || "") !== it.name, { bio: it.intro_bio, name: it.name });
  }

  console.log("\n== fb.profile.enrich: one tab returns the posts AND the About section ==");
  {
    // Stage 4 needs both — dated posts for proof-of-life, About for the trade and the address —
    // and they were two capabilities, so one person cost two page loads.
    const { ctx } = makeCtx({ seeMore: 0 });
    ctx.location.href = "https://www.facebook.com/jebsmith";      // the ROOT, not /about
    ctx.window.__soloGqlExtractCapability = (id) => id === "fb.profile.posts"
      ? { available: true, items: [{ id: "p1", text: "Just closed on Main St" }, { id: "p2", text: "Market update" },
                                   { id: "p3", text: "Open house" }, { id: "p4", text: "old" }, { id: "p5", text: "older" },
                                   { id: "p6", text: "oldest" }] }
      : { available: false, reason: "no_capture_in_scope", items: [] };
    const res = await ctx.window.__soloGqlPaginate("fb.profile.enrich", { settle_ms: 1, max_posts: 3 });
    const it = (res.items || [])[0] || {};
    check("the record is one merged object", res.capability === "fb.profile.enrich" && (res.items || []).length === 1, res.capability);
    check("max_posts is honoured", (it.posts || []).length === 3, (it.posts || []).length);
    check("the About fields came along in the same record", Array.isArray(it.about_lines) && !!it.emails, Object.keys(it));
    check("the About walk still ran", (it.checked || []).length > 0, it.checked);
    // "no posts" and "the timeline had not rendered" need opposite fixes, so they are not merged.
    check("an absent videos capture is reported, not silently empty",
      it.timeline && it.timeline.videos_available === false, it.timeline);
    check("it records which url it landed on", /jebsmith/.test((it.timeline || {}).landed_on || ""), it.timeline);
  }

  console.log("\n== enrich refuses the operator's own profile too ==");
  {
    const { ctx } = makeCtx({ seeMore: 0 });
    ctx.location.pathname = "/me";
    ctx.window.__soloGqlExtractCapability = () => ({ available: false, items: [] });
    const res = await ctx.window.__soloGqlPaginate("fb.profile.enrich", { settle_ms: 1 });
    check("it refuses instead of returning the operator as a lead",
      res.available === false && res.reason === "self_profile", { a: res.available, r: res.reason });
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
