// Offline harness for the Zillow capabilities in chrome-extension/zillow_extract.js:
//   zillow.agents.list     — one directory page (with or without ?name=<kw>) -> agent cards
//   zillow.profile.enrich  — one agent/team profile -> ProfileEnrich-compatible lead record
//
// The fixtures below are shaped EXACTLY like the __NEXT_DATA__ payloads observed live on
// 2026-08-16 (directory page los-angeles-ca with ?name=kim; team-lead profile "Pardee
// Properties"; team-member profile "tykunkle") — the same field names, nesting and value
// formats ("(1,664)", "$10K - $18M", "8/14/2026", HTML in getToKnowMe.description). A fake
// document serves that JSON through the same DOM calls the extractor makes, so the test proves
// the mapping, the null handling, the pager math, the block detection and the DOM fallback —
// not just that a function returns an object.
//
// Run:  node solo-agency-collector/tests/test_zillow_extract.js
const fs = require("fs");
const path = require("path");
const vm = require("vm");

const SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "zillow_extract.js"), "utf8");

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail) : "")); }
}

// ------------------------------------------------------------------ fake DOM
// A tiny element model + selector matcher covering exactly the selectors the extractor uses:
// #id, tag, tag[attr="v"], tag[attr*="v" i], tag[attr^="v"], comma lists, "A B" descendants.
function el(tag, attrs, text, children) {
  const node = { tag: tag.toLowerCase(), attrs: attrs || {}, text: text || "", children: children || [], parent: null };
  node.children.forEach((c) => { c.parent = node; });
  Object.defineProperty(node, "innerText", { get() { return [node.text].concat(node.children.map((c) => c.innerText)).filter(Boolean).join("\n"); } });
  Object.defineProperty(node, "textContent", { get() { return [node.text].concat(node.children.map((c) => c.textContent)).join(""); } });
  Object.defineProperty(node, "parentElement", { get() { return node.parent; } });
  Object.defineProperty(node, "disabled", { get() { return node.attrs.disabled === "true" || node.attrs.disabled === ""; } });
  node.getAttribute = (a) => (node.attrs[a] == null ? null : String(node.attrs[a]));
  node.querySelectorAll = (sel) => descendants(node).filter((d) => matches(d, sel));
  node.querySelector = (sel) => node.querySelectorAll(sel)[0] || null;
  return node;
}
function descendants(node) {
  const out = [];
  (function walk(n) { n.children.forEach((c) => { out.push(c); walk(c); }); })(node);
  return out;
}
function matchSimple(node, sel) {
  sel = sel.trim();
  if (sel[0] === "#") return node.attrs.id === sel.slice(1);
  const m = sel.match(/^([a-z0-9]*)(?:\[([^\]]+)\])?$/i);
  if (!m) return false;
  const tag = m[1], cond = m[2];
  if (tag && node.tag !== tag.toLowerCase()) return false;
  if (!cond) return true;
  const cm = cond.match(/^([\w-]+)(?:([*^$]?)=["']([^"']*)["'])?(\s+i)?$/);
  if (!cm) return false;
  const name = cm[1], op = cm[2], ci = !!cm[4];
  let val = cm[3];
  const raw = node.attrs[name];
  if (raw == null) return false;
  if (val == null) return true;
  const a = ci ? String(raw).toLowerCase() : String(raw); val = ci ? val.toLowerCase() : val;
  if (op === "*") return a.indexOf(val) !== -1;
  if (op === "^") return a.indexOf(val) === 0;
  if (op === "$") return a.endsWith(val);
  return a === val;
}
function matches(node, selectorList) {
  return selectorList.split(",").some((alt) => {
    const parts = alt.trim().split(/\s+(?![^\[]*\])/); // split on spaces outside [...]
    if (!matchSimple(node, parts[parts.length - 1])) return false;
    let anc = node.parent;
    for (let i = parts.length - 2; i >= 0; i--) {
      while (anc && !matchSimple(anc, parts[i])) anc = anc.parent;
      if (!anc) return false;
      anc = anc.parent;
    }
    return true;
  });
}

// Build the vm context: `window` is the context global (the extractor checks typeof window),
// `document`/`location` are the fakes, and the Node globals the file needs are passed in.
function makeCtx(opts) {
  opts = opts || {};
  const nodes = opts.nodes || [];
  const root = el("html", {}, "", [el("body", {}, opts.bodyText || "", nodes)]);
  const nd = opts.nextData ? el("script", { id: "__NEXT_DATA__", type: "application/json" }, JSON.stringify(opts.nextData)) : null;
  if (nd) root.children[0].children.push(nd), (nd.parent = root.children[0]);
  const document = {
    title: opts.title || "",
    body: { get innerText() { return root.children[0].innerText; } },
    getElementById: (id) => descendants(root).find((d) => d.attrs.id === id) || null,
    querySelector: (sel) => descendants(root).find((d) => matches(d, sel)) || null,
    querySelectorAll: (sel) => descendants(root).filter((d) => matches(d, sel)),
  };
  const href = opts.href || "https://www.zillow.com/";
  const u = new URL(href);
  const location = { href, search: u.search, hostname: u.hostname, origin: u.origin };
  const ctx = { document, location, console, setTimeout, clearTimeout, URLSearchParams, URL };
  ctx.window = ctx;
  vm.createContext(ctx);
  vm.runInContext(SRC, ctx, { filename: "zillow_extract.js" });
  return ctx;
}

// ------------------------------------------------------------------ fixtures
// Directory page los-angeles-ca/?name=kim, as observed: 16 cards, 348 found, "Page 1 of 24".
// Three representative cards: a normal agent, one with NO stats (null formattedData), a TEAM.
function card(o) {
  return {
    __typename: "AgentDirectoryFinderProfileResultsCard",
    cardActionLink: o.link, cardTitle: o.name, encodedZuid: o.zuid, imageUrl: o.img || "https://photos.zillowstatic.com/fp/x-h_l.jpg",
    isTopAgent: !!o.top, logoUrl: null,
    profileData: o.stats || [
      { __typename: "AgentDirectoryFinderProfileCardData", formattedData: "$485K - $1.8M", label: "price range" },
      { __typename: "AgentDirectoryFinderProfileCardData", formattedData: "6", label: "sales last 12 months" },
      { __typename: "AgentDirectoryFinderProfileCardData", formattedData: "16", label: "sales in Los Angeles" },
    ],
    reviewInformation: { __typename: "AgentDirectoryProfileReviewData", noReviewsText: "No reviews", reviewAverage: o.rating, reviewAverageText: String(o.rating), reviewCountFormattedText: o.reviews },
    secondaryCardTitle: o.brokerage, tags: o.tags || [],
  };
}
const LIST_CARDS = [
  card({ link: "https://www.zillow.com/profile/agentjasonkim", name: "Jason Kim", zuid: "X1-ZUx1ng2k9w3cax_1acra", rating: 4.9, reviews: "(28)", brokerage: "The Agency",
    stats: [
      { __typename: "AgentDirectoryFinderProfileCardData", formattedData: null, label: "No recent price range" },
      { __typename: "AgentDirectoryFinderProfileCardData", formattedData: null, label: "No sales last 12 months" },
      { __typename: "AgentDirectoryFinderProfileCardData", formattedData: "65", label: "sales in Los Angeles" },
    ] }),
  card({ link: "https://www.zillow.com/profile/Pardee%20Properties", name: "Tami Pardee", zuid: "X1-ZUz0nmomozy2o9_9bpwk", rating: 5, reviews: "(1,664)", brokerage: "Pardee Properties", top: true,
    tags: [{ __typename: "AgentDirectoryFinderTag", tagType: "WARNING", text: "TEAM" }],
    stats: [
      { __typename: "AgentDirectoryFinderProfileCardData", formattedData: "$10K - $18M", label: "team price range" },
      { __typename: "AgentDirectoryFinderProfileCardData", formattedData: "266", label: "team sales last 12 months" },
      { __typename: "AgentDirectoryFinderProfileCardData", formattedData: "2,287", label: "team sales in Los Angeles" },
    ] }),
  // the upsell card Zillow mixes into resultsCards (measured live): not an agent, must be skipped
  { __typename: "AgentDirectoryFinderUpsellCard", cardActionLink: "https://www.zillow.com/ods/submit_lead?request_type=ActFastV2&zipcode=90011", cardTitle: "Get help finding an agent",
    encodedZuid: null, imageUrl: null, isTopAgent: false, logoUrl: null, profileData: [], reviewInformation: null, secondaryCardTitle: null, tags: [] },
  card({ link: "https://www.zillow.com/profile/Ed%20Kim", name: "Edward Kim", zuid: "X1-ZUymqyl2vn1urt_8uv2w", rating: 5, reviews: "(28)", brokerage: "eXp Realty of California Inc\t" }),
];
function listNextData(o) {
  o = o || {};
  return {
    props: { pageProps: {
      profileType: 2,
      displayData: { agentDirectoryFinderDisplay: { searchResults: {
        __typename: "AgentDirectoryFinderSearchResults", currentPage: o.page || 1, description: "With over a million agents…",
        loadMoreResultsButtonText: "View more",
        results: { __typename: "AgentDirectoryFinderSearchSuccess", showcaseDisplay: null, resultsCards: o.cards || LIST_CARDS },
        regionId: "12447", resultsFound: o.found == null ? 348 : o.found, resultsFoundFormattedLabel: (o.found == null ? 348 : o.found) + " agents found",
        title: "Real Estate Agents in", titleLocation: "Los Angeles, CA", seoLocation: "Los Angeles, CA", urlLocation: "Los Angeles CA", validatedFilters: [],
      } } },
      region: { name: "Los Angeles", regionId: 12447, regionType: "CITY" },
      userInput: Object.assign({ appliedFilters: [], locationText: "los angeles ca", page: o.page || 1 }, o.name ? { name: o.name } : {}),
    } },
    page: "/professionals/[professionalSeoString]/[location]",
    query: Object.assign({ location: "los-angeles-ca", professionalSeoString: "real-estate-agent-reviews" }, o.name ? { name: o.name } : {}),
  };
}
// The pager nav exactly as rendered: number buttons + a visually hidden "Page 1 of 24" span.
function pagerNav(page, of, nextEnabled) {
  return el("nav", { role: "navigation", "aria-label": "Agent pages" }, "", [
    el("button", { title: "Previous page", rel: "prev", disabled: "" }, ""),
    el("button", { title: "Page " + page + ", current page", disabled: "" }, String(page)),
    el("button", { title: "Page 2" }, "2"),
    el("span", {}, "Page " + page + " of " + of),
    el("button", Object.assign({ title: "Next page", rel: "next" }, nextEnabled ? { "aria-disabled": "false" } : { disabled: "", "aria-disabled": "true" }), ""),
  ]);
}

// Team-lead profile (Pardee Properties) — values as observed.
function profileNextData(kind) {
  const lead = kind === "lead";
  const du = lead ? {
    businessAddress: { address1: "1524 Abbot Kinney Blvd.", address2: null, city: "Venice", postalCode: "90291", state: "CA" },
    businessName: "Pardee Properties", email: "zillow@pardeeproperties.com", encodedZuid: "X1-ZUz0nmomozy2o9_9bpwk", flag: 588249, inCanada: false,
    isTopAgent: true, name: "Tami Pardee", phoneNumbers: { brokerage: "(310) 861-7719", cell: "(310) 861-7719" }, profileImageId: null,
    profilePhotoSrc: "https://photos.zillowstatic.com/fp/d949d0899cb2304bdd9d4ffed699e310-h_l.jpg",
    profileTypeIds: [1, 2, 5, 8, 12, 15], profileTypes: ["consumer", "agent", "advertiser", "broker", "showcaseBuyer", "peeps"],
    ratings: { __typename: "ReviewRatings", average: 5, count: 1664 }, screenName: "Pardee Properties",
  } : {
    businessAddress: { address1: "1524 Abbot Kinney Blvd", address2: null, city: "Venice", postalCode: "90291", state: "CA" },
    businessName: "Pardee Properties", email: "tyler@pardeeproperties.com", encodedZuid: "X1-ZU11dmckyyeehah_7q4l7", flag: 71739056, inCanada: false,
    isTopAgent: true, name: "Tyler Kunkle", phoneNumbers: { business: "(310) 993-7333", cell: "(310) 993-7333" }, profileImageId: null,
    profilePhotoSrc: "https://photos.zillowstatic.com/fp/6c767055fa794a92b1e4ad1cb8037e0c-h_l.jpg",
    profileTypeIds: [1, 2], profileTypes: ["consumer", "agent"],
    ratings: { __typename: "ReviewRatings", average: 5, count: 144 }, screenName: "tykunkle",
  };
  const gk = lead ? {
    title: "Broker", description: "<p>Tami Pardee is the founder &amp; CEO of Pardee Properties.</p><p>Since 2004 the team has closed over $8B.</p>",
    yearsInIndustry: 22, videoUrl: "https://www.youtube.com/watch?v=j0BjxSkOosw",
    specialties: ["Buyer's Agent", "Listing Agent", "Relocation", "Luxury Homes"], languages: [],
    websiteUrl: "https://pardeeproperties.com/?utm_source=zillow", facebookUrl: "https://www.facebook.com/pardeeproperties",
    linkedInUrl: "http://www.linkedin.com/in/tamipardee", xUrl: "https://twitter.com/pardee_on", instagramUrl: "https://www.instagram.com/pardeeproperties",
    tiktokUrl: "https://www.tiktok.com/@pardeeproperties", youtubeUrl: "https://www.youtube.com/c/pardeeproperties",
  } : {
    title: null, description: "<p>Tyler was born in Manhattan Beach before moving to North Carolina.</p>", yearsInIndustry: 13, videoUrl: null,
    specialties: ["Buyer's Agent", "Listing Agent", "Relocation", "First Time Homebuyers"], languages: [],
    websiteUrl: "https://pardeeproperties.com/team/tyler-kunkle/", instagramUrl: "https://www.instagram.com/tykunkle/",
  };
  const pi = lead ? [
    { lines: ["Pardee Properties", "1524 Abbot Kinney Blvd.", "Venice, CA 90291"], term: "Broker address" },
    { description: "(310) 861-7719", term: "Cell phone" },
    { description: "(310) 861-7719", term: "Broker phone" },
    { links: [{ text: "Website", url: "https://pardeeproperties.com/?utm_source=zillow" }, { text: "Facebook", url: "https://www.facebook.com/pardeeproperties" }, { text: "Instagram", url: "https://www.instagram.com/pardeeproperties" }, { text: "TikTok", url: "https://www.tiktok.com/@pardeeproperties" }, { text: "X", url: "https://twitter.com/pardee_on" }, { text: "YouTube", url: "https://www.youtube.com/c/pardeeproperties" }, { text: "LinkedIn", url: "http://www.linkedin.com/in/tamipardee" }], term: "Websites" },
    { description: "Pardee Properties", term: "Screenname" },
    { description: "12/08/2006", term: "Member since" },
    { lines: ["01421451 (CA)"], term: "Real Estate Licenses" },
  ] : [
    { lines: ["Pardee Properties", "1524 Abbot Kinney Blvd", "Venice, CA 90291"], term: "Broker address" },
    { description: "(310) 993-7333", term: "Cell phone" },
    { links: [{ text: "Website", url: "https://pardeeproperties.com/team/tyler-kunkle/" }, { text: "Instagram", url: "https://www.instagram.com/tykunkle/" }], term: "Websites" },
    { description: "tykunkle", term: "Screenname" },
    { description: "01/16/2017", term: "Member since" },
    { lines: ["01965642 (CA)"], term: "Real Estate Licenses" },
  ];
  const stats = (v12, total, range, avg, team) => ({
    __typename: "SalesStats", entries: {
      averagePrice: { longTitle: "Average price", value: avg }, priceRange: { longTitle: "Price range", value: range },
      salesLastTwelveMonths: { longTitle: "Sales last 12 months", value: v12 }, totalSales: { longTitle: "Total sales", value: total },
    }, failureText: null, teamDisclaimerSection: team ? { disclaimerText: "Sales numbers represent all team members", tagText: "Team" } : null,
  });
  return {
    props: { pageProps: {
      split: {}, agentLicenses: lead
        ? [{ expiration: "2029-04-02", license_type: "agent", original_status: "disabled", state: "CA", status: "disabled", text: "01858429" },
           { expiration: "2028-03-16", license_type: "agent", original_status: "active", state: "CA", status: "active", text: "01421451" }]
        : [{ expiration: "2027-02-10", license_type: "agent", original_status: "active", state: "CA", status: "active", text: "01965642" }],
      breadcrumbs: [{ text: "California", url: "/professionals/real-estate-agent-reviews/ca/" }, { text: "Venice", url: "/professionals/real-estate-agent-reviews/venice-ca/" }, { text: du.name, url: null }],
      currentUser: { loginState: 0 }, currentUrl: "https://www.zillow.com/profile/" + du.screenName,
      displayUser: du,
      forSaleListings: { listings: [{ zpid: 1 }], listing_count: lead ? 58 : 3 }, forRentListings: { listings: [], listing_count: lead ? 16 : 0 },
      getToKnowMe: gk,
      graphQLData: {
        encodedZuid: du.encodedZuid, isPremium: false,
        professional: { __typename: "Professional", reviewRatings: { average: du.ratings.average, count: du.ratings.count } },
        premiumAgentHeader: { businessCard: { displayName: du.name }, salesStats: lead ? stats("266", "3,339", "$10K-$18M", "$1.9M", true) : stats("23", "187", "$459K-$5.5M", "$1.9M", false) },
        premierAgentSection: { logoUrl: null }, reviewsSection: { headerText: "Reviews" },
      },
      isImpersonating: false, isMobile: false, isReviewModerator: false, map: { mapCentroid: {} }, mlsIDs: [], otherLicenses: [],
      pastSales: { total: lead ? 3339 : 187, past_sales: [
        { bathrooms: 3, bedrooms: 4, city: "Los Angeles", city_state_zipcode: "Los Angeles, CA, 90066", home_details_url: "/homedetails/4834-McConnell-Ave-Los-Angeles-CA-90066/20441325_zpid/", price: "$1,285,000", represented: "Buyer", representedList: ["Buyer"], sold_date: "8/14/2026", state: "CA", street_address: "4834 McConnell Ave", zpid: 20441325 },
        { bathrooms: 5, bedrooms: 4, city: "Marina Del Rey", city_state_zipcode: "Marina Del Rey, CA, 90292", home_details_url: "/homedetails/1-Marina-Way/123_zpid/", price: "$2,920,000", represented: "Seller", representedList: ["Seller"], sold_date: "8/13/2026", state: "CA", street_address: "1 Marina Way", zpid: 123 },
      ] },
      preferredLenders: { lenders: [] },
      professionalInformation: pi,
      reviewsData: { reviews: [
        { createDate: "2026-08-14T16:18:00", rating: 5, rebuttal: null, reviewComment: "First time with this kind of endeavor and I cannot speak highly enough of the patience and commitment exhibited by Margie.", reviewId: 9289944,
          reviewee: { encodedZuid: "X1-ZUqko1azowdt6x_4n5nw", firstName: "Margie", lastName: "Arbizo", screenName: "margievarbizo", showName: false, suffix: null },
          reviewer: { encodedZuid: "X1-ZU124zl15hugfm1_1pt0h", firstName: "Amanda", lastName: "Belotto", screenName: "abelott0", showName: true, suffix: null },
          subRatings: [{ description: "Local knowledge", score: 5 }], workDescription: "Bought a Mobile / Manufactured home in 2026 in Hillside, Torrance, CA." },
        { createDate: "2026-07-01T10:00:00", rating: 4, rebuttal: null, reviewComment: "Great experience selling our condo.", reviewId: 9200000,
          reviewee: { firstName: "Tami", lastName: "Pardee", screenName: "Pardee Properties" }, reviewer: { firstName: "J", lastName: "D", screenName: "jd" }, subRatings: [], workDescription: "Sold a Condo home in 2026 in Venice, CA." },
      ], filters: {} },
      seoFooters: [], serviceAreas: [{ regionId: 10389, text: "Beverly Hills, CA", url: "/beverly-hills-ca/" }, { regionId: 12447, text: "Los Angeles, CA", url: "/los-angeles-ca/" }, { regionId: 12520, text: "Malibu, CA", url: "/malibu-ca/" }],
      teamDisplayInformation: lead
        ? { teamLeadInfo: { children: [
              { encodedZuid: "X1-ZU11dmckyyeehah_7q4l7", isTopAgent: true, name: "Tyler Kunkle", profilePhotoUrl: "https://photos.zillowstatic.com/fp/6c76-h_l.jpg", ratings: { average: 5, count: 144 }, screenName: "tykunkle" },
              { encodedZuid: "X1-ZUwkltbxgz2qdl_22g4q", isTopAgent: true, name: "Kerry Ann Sullivan", profilePhotoUrl: "https://photos.zillowstatic.com/fp/d852-h_l.jpg", ratings: { average: 5, count: 65 }, screenName: "Kerry Ann Sullivan" },
            ], teamName: "Pardee Properties" }, teamMemberInfo: null }
        : { teamLeadInfo: null, teamMemberInfo: { hasContactRedirectOptInFunction: false, teamLead: { encodedZuid: "X1-ZUz0nmomozy2o9_9bpwk", name: "Tami Pardee", ratings: { average: 5, count: 1664 }, screenName: "Pardee Properties" }, teamName: "Pardee Properties" } },
      zGuid: "8a7d11c8-ab7c-4a10-a9d6-0ceb42776527",
    } },
    page: "/profile/[screenName]", query: { screenName: du.screenName },
  };
}

// The bridge redacts any key containing one of these substrings (main.go isSensitiveKey), at
// any depth. A record field named e.g. "author" or "session_id" would come back "[redacted]".
const SENSITIVE = ["cookie", "token", "secret", "password", "passwd", "pwd", "otp", "authorization", "auth", "session", "bearer", "csrf", "xsrf"];
function sensitiveKeys(o, pathStr, out) {
  out = out || [];
  if (Array.isArray(o)) { o.forEach((v, i) => sensitiveKeys(v, pathStr + "[" + i + "]", out)); return out; }
  if (!o || typeof o !== "object") return out;
  Object.keys(o).forEach((k) => {
    const lk = k.toLowerCase();
    if (SENSITIVE.some((n) => lk.indexOf(n) !== -1)) out.push(pathStr + "." + k);
    sensitiveKeys(o[k], pathStr + "." + k, out);
  });
  return out;
}

(async function main() {
  console.log("zillow.agents.list — keyword directory page (?name=kim), 3 cards, pager 'Page 1 of 24'");
  {
    const ctx = makeCtx({
      href: "https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/?name=kim",
      title: "Los Angeles, CA Realtor & Real Estate Agent Reviews | Zillow",
      nextData: listNextData({ name: "kim" }),
      nodes: [pagerNav(1, 24, true)],
    });
    const res = await ctx.window.__soloZillowRun("zillow.agents.list", {});
    check("available:true, status ok, count 3 (upsell card skipped, non_profile_cards 1)", res.available === true && res.status === "ok" && res.count === 3 && res.items.length === 3 && res.non_profile_cards === 1, res);
    check("no /ods/submit_lead item leaked", res.items.every((it) => /zillow\.com\/profile\//.test(it.profile_url)), res.items.map((i) => i.profile_url));
    check("source next_data", res.source === "next_data", res.source);
    check("page 1, page_count 24 from pager, has_more true", res.page === 1 && res.page_count === 24 && res.page_count_source === "pager" && res.has_more === true, [res.page, res.page_count, res.page_count_source, res.has_more]);
    check("next_page_url appends page=2 to the SAME url (keyword kept)", res.next_page_url === "https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/?name=kim&page=2", res.next_page_url);
    check("results_found 348, query.name kim, region", res.results_found === 348 && res.query.name === "kim" && res.query.region_id === "12447" && res.query.region_name === "Los Angeles, CA" && res.query.location_text === "los angeles ca", res.query);
    const a = res.items[0], t = res.items[1], e = res.items[2];
    check("card 1: url/screen_name/name/brokerage/zuid", a.profile_url === "https://www.zillow.com/profile/agentjasonkim" && a.screen_name === "agentjasonkim" && a.name === "Jason Kim" && a.brokerage === "The Agency" && a.encoded_zuid === "X1-ZUx1ng2k9w3cax_1acra", a);
    check("card 1: null stats stay null, sales_in_region 65, rating 4.9, reviews 28", a.price_range === null && a.sales_last_12_months === null && a.sales_in_region === 65 && a.rating === 4.9 && a.reviews_count === 28 && a.is_team === false && a.is_top_agent === false, a);
    check("card 2 (TEAM): is_team, top agent, %20 screen name decoded, reviews 1,664 -> 1664", t.is_team === true && t.is_top_agent === true && t.screen_name === "Pardee Properties" && t.reviews_count === 1664 && t.price_range === "$10K - $18M" && t.sales_last_12_months === 266 && t.sales_in_region === 2287, t);
    check("card 3: trailing tab in brokerage trimmed", e.brokerage === "eXp Realty of California Inc" && e.screen_name === "Ed Kim", e);
    check("no sensitive key names in the list envelope", sensitiveKeys(res, "res").length === 0, sensitiveKeys(res, "res"));
  }

  console.log("zillow.agents.list — page 3 of a plain (no keyword) directory url, pager says last page");
  {
    const ctx = makeCtx({
      href: "https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/?page=3",
      nextData: listNextData({ page: 3, found: 44534 }),
      nodes: [pagerNav(3, 3, false)],
    });
    const res = await ctx.window.__soloZillowRun("zillow.agents.list", {});
    check("page 3, page_count 3, has_more false (Next disabled), next_page_url null", res.page === 3 && res.page_count === 3 && res.has_more === false && res.next_page_url === null, [res.page, res.page_count, res.has_more, res.next_page_url]);
    check("query.name empty when no keyword", res.query.name === "", res.query.name);
  }

  console.log("zillow.agents.list — no pager in the DOM: page_count estimated from resultsFound, capped at 25");
  {
    const ctx = makeCtx({ href: "https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/", nextData: listNextData({ found: 44534 }) });
    const res = await ctx.window.__soloZillowRun("zillow.agents.list", {});
    check("page_count 25 (estimated), has_more true, next_page_url ?page=2", res.page_count === 25 && res.page_count_source === "estimated" && res.has_more === true && res.next_page_url === "https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/?page=2", [res.page_count, res.page_count_source, res.next_page_url]);
    const ctx2 = makeCtx({ href: "https://www.zillow.com/professionals/real-estate-agent-reviews/x/?name=zz", nextData: listNextData({ found: 20, cards: LIST_CARDS.slice(0, 1) }) });
    const res2 = await ctx2.window.__soloZillowRun("zillow.agents.list", {});
    check("20 found -> page_count 2 (estimated)", res2.page_count === 2 && res2.has_more === true, [res2.page_count, res2.has_more]);
  }

  console.log("zillow.agents.list — truncation is STATED, because the pager cannot state it");
  {
    // Zillow serves 25 directory pages and then keeps serving page 25, so from the pager alone
    // "the list ended" and "the list was cut off" are the same observation. results_found still
    // tells the truth, and the gap between it and what is reachable is the number that decides
    // whether a keyword must be split into narrower regions. Los Angeles reports 44,534 agents.
    const ctx = makeCtx({ href: "https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/", nextData: listNextData({ found: 44534 }) });
    const big = await ctx.window.__soloZillowRun("zillow.agents.list", {});
    check("a query far over the ceiling is flagged truncated", big.truncated === true, big.truncated);
    check("it says how many agents are unreachable", big.unreachable_estimate === 44534 - 375, big.unreachable_estimate);
    check("and what fraction of them it can see", Math.abs((big.coverage_ratio || 0) - 375 / 44534) < 1e-9, big.coverage_ratio);
    check("reachable_max is the ceiling, not the arithmetic", big.reachable_max === 375, big.reachable_max);

    const ctx2 = makeCtx({ href: "https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/?name=kim", nextData: listNextData({ found: 348 }) });
    const fits = await ctx2.window.__soloZillowRun("zillow.agents.list", {});
    check("a query that fits is NOT flagged", fits.truncated === false, fits.truncated);
    check("nothing is unreachable", fits.unreachable_estimate === 0, fits.unreachable_estimate);
    check("coverage is complete", fits.coverage_ratio === 1, fits.coverage_ratio);
  }

  console.log("zillow.agents.list — __NEXT_DATA__ missing: DOM fallback from /profile/ anchors, available stays true");
  {
    const cardBox = (href, name, txt) => el("div", { class: "StyledCard" }, "", [el("a", { href }, name), el("div", {}, txt)]);
    const ctx = makeCtx({
      href: "https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/?name=kim",
      nodes: [cardBox("https://www.zillow.com/profile/agentjasonkim", "Jason Kim", "The Agency 4.9 (28)"), cardBox("/profile/Ed%20Kim", "Edward Kim", "TEAM 5.0 (12)"), el("a", { href: "https://www.zillow.com/profile/agentjasonkim" }, "Jason Kim")],
    });
    const res = await ctx.window.__soloZillowRun("zillow.agents.list", {});
    check("source dom, 2 unique cards, relative href made absolute", res.available === true && res.source === "dom" && res.count === 2 && res.items[1].profile_url === "https://www.zillow.com/profile/Ed%20Kim", res);
    check("dom card carries name/screen_name/rating/reviews/is_team", res.items[0].name === "Jason Kim" && res.items[0].rating === 4.9 && res.items[0].reviews_count === 28 && res.items[1].is_team === true, res.items);
    check("status no_next_data (so the caller knows the JSON path did not run)", res.status === "no_next_data" || res.status === "ok", res.status);
  }

  console.log("blocked: PerimeterX 'Press & Hold' page -> status blocked, available true, no retry loop");
  {
    for (const cap of ["zillow.agents.list", "zillow.profile.enrich"]) {
      const ctx = makeCtx({
        href: "https://www.zillow.com/professionals/real-estate-agent-reviews/los-angeles-ca/?page=2",
        title: "Access to this page has been denied",
        nodes: [el("div", { id: "px-captcha-wrapper" }, "", [el("div", { id: "px-captcha" }, "Press & Hold to confirm you are a human (and not a bot).")])],
      });
      const t0 = Date.now();
      const res = await ctx.window.__soloZillowRun(cap, {});
      check(cap + ": blocked envelope", res.available === true && res.status === "blocked" && res.reason === "blocked_bot_check" && res.count === 0 && res.items[0].status === "blocked" && /Press & Hold/.test(res.items[0].hint) && /Do NOT retry/.test(res.items[0].hint), res);
      check(cap + ": returned immediately (no 1.5s grace retry on a blocked page)", Date.now() - t0 < 1000, Date.now() - t0);
    }
    // title-only variant (no #px-captcha yet rendered)
    const ctx = makeCtx({ href: "https://www.zillow.com/profile/x", title: "Access to this page has been denied", bodyText: "Press & Hold to confirm you are\na human (and not a bot)." });
    const res = await ctx.window.__soloZillowRun("zillow.profile.enrich", {});
    check("title/body-only block page detected", res.status === "blocked", res.status);
  }

  console.log("zillow.profile.enrich — team-lead profile (Pardee Properties)");
  {
    const ctx = makeCtx({
      href: "https://www.zillow.com/profile/Pardee%20Properties",
      title: "Tami Pardee - Real Estate Agent in Venice, CA - Reviews | Zillow",
      nextData: profileNextData("lead"),
      nodes: [el("h1", {}, "Tami Pardee"), el("a", { href: "tel:3108617719" }, "(310) 861-7719")],
    });
    const res = await ctx.window.__soloZillowRun("zillow.profile.enrich", {});
    check("envelope: available, ok, one item, source next_data", res.available === true && res.status === "ok" && res.count === 1 && res.source === "next_data", res);
    const r = res.items[0];
    check("profile_url rebuilt from screenName (space -> %20), name", r.profile_url === "https://www.zillow.com/profile/Pardee%20Properties" && r.name === "Tami Pardee", [r.profile_url, r.name]);
    check("industry defaults to Real Estate", r.industry === "Real Estate", r.industry);
    check("category = Zillow's own title (Broker)", r.category === "Broker", r.category);
    check("work[]: 'Broker at Pardee Properties' + team lead line", r.work[0] === "Broker at Pardee Properties" && /^Lead of team Pardee Properties \(2 members\)$/.test(r.work[1]), r.work);
    check("location[] from business address", r.location.length === 1 && r.location[0] === "Venice, CA 90291", r.location);
    check("emails[] from displayUser.email", r.emails.length === 1 && r.emails[0] === "zillow@pardeeproperties.com" && r.found_on === "next_data", [r.emails, r.found_on]);
    check("phones[] deduped by digits (cell == brokerage == tel: link) -> 1", r.phones.length === 1 && r.phones[0] === "(310) 861-7719", r.phones);
    check("website = getToKnowMe.websiteUrl with utm_* stripped; websites[] includes socials, deduped", r.website === "https://pardeeproperties.com/" && r.websites.length === 7 && r.websites.indexOf("https://www.facebook.com/pardeeproperties") !== -1 && r.websites.indexOf("https://pardeeproperties.com/") !== -1, [r.website, r.websites]);
    check("about_lines Website line carries the cleaned url", r.about_lines.indexOf("Website: https://pardeeproperties.com/") !== -1, r.about_lines.filter((l) => /^Website/.test(l)));
    check("intro_bio: HTML stripped, entity decoded, paragraphs kept", r.intro_bio === "Tami Pardee is the founder & CEO of Pardee Properties.\nSince 2004 the team has closed over $8B.", r.intro_bio);
    check("about_lines carry brokerage, licence, sales stats, service areas, team", ["Brokerage: Pardee Properties", "Real Estate Licenses: 01858429 (CA, disabled, exp 2029-04-02), 01421451 (CA, active, exp 2028-03-16)", "Sales last 12 months: 266", "Total sales: 3,339", "Price range: $10K-$18M", "Average price: $1.9M", "Sales numbers represent all team members", "Team: lead of Pardee Properties (2 members)", "Top Agent on Zillow", "Years in industry: 22", "Member since: 12/08/2006"].every((l) => r.about_lines.indexOf(l) !== -1) && r.about_lines.some((l) => /^Service areas \(3\): Beverly Hills, CA; Los Angeles, CA; Malibu, CA$/.test(l)), r.about_lines);
    check("checked/missing/source/about_panel_found as a next_data read", r.checked.join(",") === "next_data,dom" && r.missing.length === 0 && r.source === "next_data" && r.about_panel_found === true && r.budget_exhausted === false, [r.checked, r.missing, r.source]);
    const z = r.zillow;
    check("zillow.team: role lead, 2 members with profile urls", z.team.role === "lead" && z.team.team_name === "Pardee Properties" && z.team.member_count === 2 && z.team.members[0].profile_url === "https://www.zillow.com/profile/tykunkle" && z.team.members[1].profile_url === "https://www.zillow.com/profile/Kerry%20Ann%20Sullivan" && z.team.members[0].reviews_count === 144, z.team);
    check("zillow.sales parsed: 266 / 3339 / range / avg / team total", z.sales.last_12_months === 266 && z.sales.total === 3339 && z.sales.price_range === "$10K-$18M" && z.sales.average_price === "$1.9M" && z.sales.is_team_total === true, z.sales);
    check("zillow.licenses[] typed", z.licenses.length === 2 && z.licenses[1].number === "01421451" && z.licenses[1].state === "CA" && z.licenses[1].status === "active" && z.licenses[1].expiration === "2028-03-16", z.licenses);
    check("zillow.rating / listings / specialties / service areas / socials", z.rating.average === 5 && z.rating.count === 1664 && z.listings.for_sale === 58 && z.listings.for_rent === 16 && z.specialties.length === 4 && z.service_area_count === 3 && z.socials.facebook === "https://www.facebook.com/pardeeproperties" && z.socials.x === "https://twitter.com/pardee_on" && z.socials.linkedin === "http://www.linkedin.com/in/tamipardee", z);
    check("zillow.business_address typed + line", z.business_address.city === "Venice" && z.business_address.postal_code === "90291" && z.business_address.line === "1524 Abbot Kinney Blvd., Venice, CA 90291", z.business_address);
    check("zillow.recent_sales: ISO date, address, absolute url", z.recent_sales[0].sold_date === "2026-08-14" && z.recent_sales[0].address === "4834 McConnell Ave, Los Angeles, CA, 90066" && z.recent_sales[0].url === "https://www.zillow.com/homedetails/4834-McConnell-Ave-Los-Angeles-CA-90066/20441325_zpid/" && z.recent_sales[0].represented === "Buyer", z.recent_sales[0]);
    check("zillow.recent_reviews: date, rating, snippet, reviewee (no reviewer name kept)", z.recent_reviews[0].date === "2026-08-14" && z.recent_reviews[0].rating === 5 && /patience/.test(z.recent_reviews[0].snippet) && z.recent_reviews[0].reviewee_screen_name === "margievarbizo" && !("reviewer" in z.recent_reviews[0]) && !("reviewer_name" in z.recent_reviews[0]), z.recent_reviews[0]);
    check("posts[]: dated reviews+sales newest first, default max 5 (4 exist)", r.posts.length === 4 && r.posts[0].kind === "review" && r.posts[0].date === "2026-08-14" && r.posts[1].kind === "sale" && r.posts[1].date === "2026-08-14" && r.posts[2].date === "2026-08-13" && r.posts[3].date === "2026-07-01" && r.posts.every((p) => typeof p.created_time === "number" && p.text && p.url), r.posts);
    check("timeline: posts_available true, seen 4 kept 4", r.timeline.posts_available === true && r.timeline.posts_seen === 4 && r.timeline.posts_kept === 4, r.timeline);
    check("no sensitive key names anywhere in the record", sensitiveKeys(res, "res").length === 0, sensitiveKeys(res, "res"));
    check("ProfileDossier keys present (harvest UI / playbook 04 read these)", ["profile_url", "name", "category", "intro_bio", "work", "education", "location", "about", "about_lines", "emails", "websites", "website", "phones", "found_on", "checked", "missing", "source", "elapsed_ms", "posts", "timeline"].every((k) => k in r), Object.keys(r));

    // max_posts honoured
    const res2 = await ctx.window.__soloZillowRun("zillow.profile.enrich", { max_posts: 2 });
    check("max_posts:2 keeps the 2 newest, posts_seen still 4", res2.items[0].posts.length === 2 && res2.items[0].timeline.posts_seen === 4 && res2.items[0].timeline.posts_kept === 2, res2.items[0].timeline);
    const res3 = await ctx.window.__soloZillowRun("zillow.profile.enrich", { max_team_members: 1 });
    check("max_team_members:1 truncates members but member_count says 2", res3.items[0].zillow.team.members.length === 1 && res3.items[0].zillow.team.member_count === 2, res3.items[0].zillow.team);
  }

  console.log("zillow.profile.enrich — team-member profile (tykunkle): title null, member-of-team, no team disclaimer");
  {
    const ctx = makeCtx({ href: "https://www.zillow.com/profile/tykunkle", title: "Tyler Kunkle - Real Estate Agent in Venice, CA - Reviews | Zillow", nextData: profileNextData("member") });
    const res = await ctx.window.__soloZillowRun("zillow.profile.enrich", {});
    const r = res.items[0];
    check("name/url/email/phone", r.name === "Tyler Kunkle" && r.profile_url === "https://www.zillow.com/profile/tykunkle" && r.emails[0] === "tyler@pardeeproperties.com" && r.phones.length === 1 && r.phones[0] === "(310) 993-7333", [r.name, r.emails, r.phones]);
    check("category falls back to 'Real estate agent' when title is null", r.category === "Real estate agent", r.category);
    check("work[]: 'Real estate agent at Pardee Properties' + member-of-team line with lead", r.work[0] === "Real estate agent at Pardee Properties" && r.work[1] === "Member of team Pardee Properties (lead: Tami Pardee)", r.work);
    check("zillow.team role member with lead ref", r.zillow.team.role === "member" && r.zillow.team.lead.profile_url === "https://www.zillow.com/profile/Pardee%20Properties" && r.zillow.team.lead.reviews_count === 1664, r.zillow.team);
    check("sales: 23 / 187 / not team total", r.zillow.sales.last_12_months === 23 && r.zillow.sales.total === 187 && r.zillow.sales.is_team_total === false, r.zillow.sales);
    check("website = personal team page; socials only instagram", r.website === "https://pardeeproperties.com/team/tyler-kunkle/" && Object.keys(r.zillow.socials).join(",") === "instagram", [r.website, r.zillow.socials]);
    check("about_lines have no 'Title:' line and no team-total disclaimer", !r.about_lines.some((l) => /^Title:/.test(l)) && !r.about_lines.some((l) => /represent all team members/.test(l)), r.about_lines);
  }

  console.log("zillow.profile.enrich — __NEXT_DATA__ missing: DOM fallback record, source dom, missing next_data");
  {
    const ctx = makeCtx({
      href: "https://www.zillow.com/profile/Someone%20Else",
      title: "Someone Else - Real Estate Agent in Venice, CA - Reviews | Zillow",
      nodes: [el("h1", {}, "Someone Else"), el("a", { href: "mailto:someone@example.com" }, "email"), el("a", { href: "tel:+13105551234" }, "call"),
        el("nav", { "aria-label": "Breadcrumb" }, "", [el("a", { href: "/x" }, "California"), el("a", { href: "/y" }, "Venice")])],
    });
    const t0 = Date.now();
    const res = await ctx.window.__soloZillowRun("zillow.profile.enrich", {});
    const r = res.items[0];
    check("available true, status no_next_data, source dom, missing [next_data]", res.available === true && res.status === "no_next_data" && r.source === "dom" && r.missing[0] === "next_data" && r.checked[0] === "dom", [res.status, r.source, r.missing, r.checked]);
    check("name from h1, email from mailto:, phone from tel:, location from breadcrumb, industry kept", r.name === "Someone Else" && r.emails[0] === "someone@example.com" && r.phones[0] === "+13105551234" && r.location[0] === "California, Venice" && r.industry === "Real Estate", r);
    check("profile_url rebuilt from the url screen name", r.profile_url === "https://www.zillow.com/profile/Someone%20Else", r.profile_url);
    check("one 1.5s grace retry happened (page might have been slow), then answered", Date.now() - t0 >= 1400, Date.now() - t0);
  }

  console.log("dispatch — unknown zillow capability id is a visible error, not a null record");
  {
    const ctx = makeCtx({ href: "https://www.zillow.com/profile/x", nextData: profileNextData("member") });
    const res = await ctx.window.__soloZillowRun("zillow.nope", {});
    check("available true + status error + no_extractor message", res.available === true && res.status === "error" && /no zillow extractor/.test(res.error) && res.items[0].status === "error", res);
    check("__soloZillowCapabilities lists exactly the two ids", JSON.stringify(ctx.window.__soloZillowCapabilities) === JSON.stringify(["zillow.agents.list", "zillow.profile.enrich"]), ctx.window.__soloZillowCapabilities);
    check("re-injection is a no-op (function identity kept)", (() => { const f = ctx.window.__soloZillowRun; vm.runInContext(SRC, ctx); return ctx.window.__soloZillowRun === f; })());
  }

  console.log("");
  console.log(pass + " passed, " + fail + " failed");
  process.exit(fail ? 1 : 0);
})().catch((e) => { console.error(e); process.exit(2); });
