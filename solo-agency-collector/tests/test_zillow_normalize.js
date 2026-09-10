// Offline harness for chrome-extension/platforms/zillow/zillow_normalize.js — the canonical
// normalizer that reshapes today's zillow.* capability output into the shared
// {schema_version, kind, items[]} contract (chrome-extension/core/schema.js's root.SoloSchema).
//
// Style follows tests/test_zillow_extract.js: fs.readFileSync + vm.createContext + "  ok"/"  FAIL"
// lines, exit 1 on any failure, "ALL N CHECKS PASSED" on none.
//
// Fixtures: the fake-DOM element model, makeCtx(), and the card()/LIST_CARDS/listNextData()/
// profileNextData() factories below are copied verbatim from tests/test_zillow_extract.js so the
// REAL zillow_extract.js extractors run over them, and their real output is what gets normalized
// — nothing here is invented.
const fs = require("fs");
const path = require("path");
const vm = require("vm");

const ZILLOW_SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "platforms", "zillow", "zillow_extract.js"), "utf8");
const NORM_SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "platforms", "zillow", "zillow_normalize.js"), "utf8");
const SCHEMA_PATH = path.join(__dirname, "..", "chrome-extension", "core", "schema.js");

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass += 1; console.log("  ok   " + name); }
  else { fail += 1; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail) : "")); }
}

const CAPTURED_AT = "2026-09-09T12:00:00.000Z";

// Same needle list schema.js / bridge-go's isSensitiveKey use, verbatim from
// tests/test_profile_dossier.js:269-282.
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

const ALL_ITEMS = [];
function collect(canonical) {
  if (canonical && Array.isArray(canonical.items)) canonical.items.forEach((it) => ALL_ITEMS.push(it));
  return canonical;
}

function isPlainObj(v) { return !!v && typeof v === "object" && !Array.isArray(v); }

// ============================================================================================
// tests/test_zillow_extract.js:29-104 — fake DOM element model, CSS-selector matcher, makeCtx().
// ============================================================================================
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
    const parts = alt.trim().split(/\s+(?![^\[]*\])/);
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
  vm.runInContext(ZILLOW_SRC, ctx, { filename: "zillow_extract.js" });
  vm.runInContext(NORM_SRC, ctx, { filename: "zillow_normalize.js" });
  return ctx;
}

// tests/test_zillow_extract.js:109-122 — one Zillow agent-directory result card.
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
// tests/test_zillow_extract.js:123-141
const LIST_CARDS = [
  card({ link: "https://www.zillow.com/profile/agentjordanlee", name: "Jordan Lee", zuid: "X1-ZUx1ng2k9w3cax_1acra", rating: 4.9, reviews: "(28)", brokerage: "The Agency",
    stats: [
      { __typename: "AgentDirectoryFinderProfileCardData", formattedData: null, label: "No recent price range" },
      { __typename: "AgentDirectoryFinderProfileCardData", formattedData: null, label: "No sales last 12 months" },
      { __typename: "AgentDirectoryFinderProfileCardData", formattedData: "65", label: "sales in Los Angeles" },
    ] }),
  card({ link: "https://www.zillow.com/profile/Sample%20Realty", name: "Alex Rivera", zuid: "X1-ZUz0nmomozy2o9_9bpwk", rating: 5, reviews: "(1,664)", brokerage: "Sample Realty", top: true,
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
// tests/test_zillow_extract.js:142-160
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

// tests/test_zillow_extract.js:173-267 — the zillow.profile.enrich fixture (lead/member variant).
function profileNextData(kind) {
  const lead = kind === "lead";
  const du = lead ? {
    businessAddress: { address1: "1524 Abbot Kinney Blvd.", address2: null, city: "Venice", postalCode: "90291", state: "CA" },
    businessName: "Sample Realty", email: "zillow@example.com", encodedZuid: "X1-ZUz0nmomozy2o9_9bpwk", flag: 588249, inCanada: false,
    isTopAgent: true, name: "Alex Rivera", phoneNumbers: { brokerage: "(555) 555-0142", cell: "(555) 555-0142" }, profileImageId: null,
    profilePhotoSrc: "https://photos.zillowstatic.com/fp/d949d0899cb2304bdd9d4ffed699e310-h_l.jpg",
    profileTypeIds: [1, 2, 5, 8, 12, 15], profileTypes: ["consumer", "agent", "advertiser", "broker", "showcaseBuyer", "peeps"],
    ratings: { __typename: "ReviewRatings", average: 5, count: 1664 }, screenName: "Sample Realty",
  } : {
    businessAddress: { address1: "1524 Abbot Kinney Blvd", address2: null, city: "Venice", postalCode: "90291", state: "CA" },
    businessName: "Sample Realty", email: "sam@example.com", encodedZuid: "X1-ZU11dmckyyeehah_7q4l7", flag: 71739056, inCanada: false,
    isTopAgent: true, name: "Sam Delgado", phoneNumbers: { business: "(555) 555-0143", cell: "(555) 555-0143" }, profileImageId: null,
    profilePhotoSrc: "https://photos.zillowstatic.com/fp/6c767055fa794a92b1e4ad1cb8037e0c-h_l.jpg",
    profileTypeIds: [1, 2], profileTypes: ["consumer", "agent"],
    ratings: { __typename: "ReviewRatings", average: 5, count: 144 }, screenName: "sampleagent",
  };
  const gk = lead ? {
    title: "Broker", description: "<p>Alex Rivera is the founder &amp; CEO of Sample Realty.</p><p>Since 2004 the team has closed over $8B.</p>",
    yearsInIndustry: 22, videoUrl: "https://www.youtube.com/watch?v=j0BjxSkOosw",
    specialties: ["Buyer's Agent", "Listing Agent", "Relocation", "Luxury Homes"], languages: [],
    websiteUrl: "https://example.com/?utm_source=zillow", facebookUrl: "https://www.facebook.com/samplerealty",
    linkedInUrl: "http://www.linkedin.com/in/samplealex", xUrl: "https://twitter.com/sample_realty", instagramUrl: "https://www.instagram.com/samplerealty",
    tiktokUrl: "https://www.tiktok.com/@samplerealty", youtubeUrl: "https://www.youtube.com/c/samplerealty",
  } : {
    title: null, description: "<p>Sam was born in Manhattan Beach before moving to North Carolina.</p>", yearsInIndustry: 13, videoUrl: null,
    specialties: ["Buyer's Agent", "Listing Agent", "Relocation", "First Time Homebuyers"], languages: [],
    websiteUrl: "https://example.com/team/sample-agent/", instagramUrl: "https://www.instagram.com/sampleagent/",
  };
  const pi = lead ? [
    { lines: ["Sample Realty", "1524 Abbot Kinney Blvd.", "Venice, CA 90291"], term: "Broker address" },
    { description: "(555) 555-0142", term: "Cell phone" },
    { description: "(555) 555-0142", term: "Broker phone" },
    { links: [{ text: "Website", url: "https://example.com/?utm_source=zillow" }, { text: "Facebook", url: "https://www.facebook.com/samplerealty" }, { text: "Instagram", url: "https://www.instagram.com/samplerealty" }, { text: "TikTok", url: "https://www.tiktok.com/@samplerealty" }, { text: "X", url: "https://twitter.com/sample_realty" }, { text: "YouTube", url: "https://www.youtube.com/c/samplerealty" }, { text: "LinkedIn", url: "http://www.linkedin.com/in/samplealex" }], term: "Websites" },
    { description: "Sample Realty", term: "Screenname" },
    { description: "12/08/2006", term: "Member since" },
    { lines: ["01421451 (CA)"], term: "Real Estate Licenses" },
  ] : [
    { lines: ["Sample Realty", "1524 Abbot Kinney Blvd", "Venice, CA 90291"], term: "Broker address" },
    { description: "(555) 555-0143", term: "Cell phone" },
    { links: [{ text: "Website", url: "https://example.com/team/sample-agent/" }, { text: "Instagram", url: "https://www.instagram.com/sampleagent/" }], term: "Websites" },
    { description: "sampleagent", term: "Screenname" },
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
          reviewee: { firstName: "Alex", lastName: "Sample Realty", screenName: "Sample Realty" }, reviewer: { firstName: "J", lastName: "D", screenName: "jd" }, subRatings: [], workDescription: "Sold a Condo home in 2026 in Venice, CA." },
      ], filters: {} },
      seoFooters: [], serviceAreas: [{ regionId: 10389, text: "Beverly Hills, CA", url: "/beverly-hills-ca/" }, { regionId: 12447, text: "Los Angeles, CA", url: "/los-angeles-ca/" }, { regionId: 12520, text: "Malibu, CA", url: "/malibu-ca/" }],
      teamDisplayInformation: lead
        ? { teamLeadInfo: { children: [
              { encodedZuid: "X1-ZU11dmckyyeehah_7q4l7", isTopAgent: true, name: "Sam Delgado", profilePhotoUrl: "https://photos.zillowstatic.com/fp/6c76-h_l.jpg", ratings: { average: 5, count: 144 }, screenName: "sampleagent" },
              { encodedZuid: "X1-ZUwkltbxgz2qdl_22g4q", isTopAgent: true, name: "Kerry Ann Sullivan", profilePhotoUrl: "https://photos.zillowstatic.com/fp/d852-h_l.jpg", ratings: { average: 5, count: 65 }, screenName: "Kerry Ann Sullivan" },
            ], teamName: "Sample Realty" }, teamMemberInfo: null }
        : { teamLeadInfo: null, teamMemberInfo: { hasContactRedirectOptInFunction: false, teamLead: { encodedZuid: "X1-ZUz0nmomozy2o9_9bpwk", name: "Alex Rivera", ratings: { average: 5, count: 1664 }, screenName: "Sample Realty" }, teamName: "Sample Realty" } },
      zGuid: "8a7d11c8-ab7c-4a10-a9d6-0ceb42776527",
    } },
    page: "/profile/[screenName]", query: { screenName: du.screenName },
  };
}

// ============================================================================================
// Tests
// ============================================================================================

(function testAgentsList() {
  console.log("\n-- zillow.agents.list (real extractor) --");
  const ctx = makeCtx({ href: "https://www.zillow.com/los-angeles-ca/real-estate-agent-reviews/", nextData: listNextData({}) });
  return ctx.window.__soloZillowRun("zillow.agents.list", {}).then((res) => {
    check("extractor kept 3 of the 4 cards (upsell card skipped)", res.count === 3, res.count);

    const before = JSON.stringify(res);
    const out = collect(ctx.__soloZillowNormalize("zillow.agents.list", res, { captured_at: CAPTURED_AT }));
    check("input not mutated", JSON.stringify(res) === before);
    check("schema_version 1, kind profile", out.schema_version === 1 && out.kind === "profile", out);
    check("3 canonical profiles", out.items.length === 3, out.items.length);

    const jordan = out.items.find((i) => i.name === "Jordan Lee");
    check("platform/platform_id (encoded_zuid)/url/handle", jordan.platform === "zillow" && jordan.platform_id === "X1-ZUx1ng2k9w3cax_1acra" && jordan.url === "https://www.zillow.com/profile/agentjordanlee" && jordan.handle, jordan);
    check("industry constant 'Real Estate' on every agent card", out.items.every((i) => i.industry === "Real Estate"), out.items.map((i) => i.industry));
    check("photo_url mapped from imageUrl", jordan.photo_url === "https://photos.zillowstatic.com/fp/x-h_l.jpg", jordan.photo_url);
    check("'No recent price range'/'No sales...' -> null, not a string, carried into ext", jordan.ext.price_range === null && jordan.ext.sales_last_12_months === null, jordan.ext);
    check("sales_in_region found -> numeric, carried into ext", jordan.ext.sales_in_region === 65, jordan.ext.sales_in_region);

    const alex = out.items.find((i) => i.name === "Alex Rivera");
    check("TEAM tag -> ext.is_team true", alex.ext.is_team === true, alex.ext);
    check("is_top_agent carried", alex.ext.is_top_agent === true, alex.ext.is_top_agent);

    console.log("\n-- zillow.agents.list: blocked page is skipped, not normalized (zillow_extract.js:117-127) --");
    const blockedCtx = makeCtx({ href: "https://www.zillow.com/los-angeles-ca/real-estate-agent-reviews/", nodes: [el("div", { id: "px-captcha-wrapper" }, "", [el("div", { id: "px-captcha" }, "Press & Hold to confirm you are a human (and not a bot).")])] });
    return blockedCtx.window.__soloZillowRun("zillow.agents.list", {}).then((blockedRes) => {
      check("extractor reports blocked", blockedRes.status === "blocked", blockedRes.status);
      const blockedOut = collect(blockedCtx.__soloZillowNormalize("zillow.agents.list", blockedRes, { captured_at: CAPTURED_AT }));
      check("blocked page normalizes to zero canonical items", blockedOut.items.length === 0, blockedOut);
      return testProfileEnrich();
    });
  });
})().catch((e) => { console.error(e); process.exit(1); });

function testProfileEnrich() {
  console.log("\n-- zillow.profile.enrich (real extractor, lead variant) --");
  const ctx = makeCtx({ href: "https://www.zillow.com/profile/Sample%20Realty", nextData: profileNextData("lead") });
  return ctx.window.__soloZillowRun("zillow.profile.enrich", {}).then((res) => {
    check("extractor ran ok", res.status === "ok" && res.items.length === 1, res.status);

    const before = JSON.stringify(res);
    const out = collect(ctx.__soloZillowNormalize("zillow.profile.enrich", res, { captured_at: CAPTURED_AT }));
    check("input not mutated", JSON.stringify(res) === before);
    check("schema_version 1, kind profile, 1 item", out.schema_version === 1 && out.kind === "profile" && out.items.length === 1, out);

    const it = out.items[0];
    const raw = res.items[0];
    check("platform_id from zillow.encoded_zuid, url from profile_url, name", it.platform === "zillow" && it.platform_id === raw.zillow.encoded_zuid && it.url === raw.profile_url && it.name === "Alex Rivera", it);
    check("industry passthrough of the item's own `industry` field ('Real Estate')", it.industry === "Real Estate", it.industry);
    check("photo_url promoted from zillow.photo_url to the top-level canonical field", it.photo_url === raw.zillow.photo_url && it.photo_url.length > 0, it.photo_url);
    check("emails[]/phones[] taken from the top-level arrays AS-IS (already include zillow.email/zillow.phones)", JSON.stringify(it.emails) === JSON.stringify(raw.emails) && JSON.stringify(it.phones) === JSON.stringify(raw.phones), [it.emails, raw.emails]);
    check("zillow.email is NOT duplicated a second time into canonical emails[]", it.emails.filter((e) => e === raw.zillow.email).length <= 1, it.emails);
    check("full zillow{} block carried into ext.zillow", isPlainObj(it.ext.zillow) && it.ext.zillow.brokerage === "Sample Realty", it.ext.zillow);
    check("extraction_audit.skipped_tabs:[] added for parity (zillow's baseRecord never had this key)", Array.isArray(it.extraction_audit.skipped_tabs) && it.extraction_audit.skipped_tabs.length === 0, it.extraction_audit);
    check("about{} (dossier-style open map) carried through", isPlainObj(it.about) && it.about.brokerage === "Sample Realty", it.about);
    check("posts[]/videos[]/timeline carried into ext as raw platform-only facts (Zillow's posts[] has an extra `kind`+`date` PostRecord-*like* shape, not re-normalized)", Array.isArray(it.ext.posts) && it.ext.posts.length > 0 && it.ext.posts[0].kind, it.ext.posts[0]);

    console.log("\n-- zillow.profile.enrich (real extractor, member variant) --");
    const ctx2 = makeCtx({ href: "https://www.zillow.com/profile/sampleagent", nextData: profileNextData("member") });
    return ctx2.window.__soloZillowRun("zillow.profile.enrich", {}).then((res2) => {
      const out2 = collect(ctx2.__soloZillowNormalize("zillow.profile.enrich", res2, { captured_at: CAPTURED_AT }));
      const it2 = out2.items[0];
      check("member variant: team.role 'member' carried through inside ext.zillow.team", it2.ext.zillow.team && it2.ext.zillow.team.role === "member", it2.ext.zillow.team);
      check("member variant: name/platform_id distinct from the lead", it2.name === "Sam Delgado" && it2.platform_id !== it.platform_id, [it2.name, it2.platform_id]);

      console.log("\n-- zillow.profile.enrich: no __NEXT_DATA__ -> dom fallback, still normalizes --");
      const ctx3 = makeCtx({ href: "https://www.zillow.com/profile/thin-agent", bodyText: "Thin Agent" });
      return ctx3.window.__soloZillowRun("zillow.profile.enrich", {}).then((res3) => {
        check("extractor fell back to dom", res3.status === "no_next_data" && res3.source === "dom", res3.status);
        const out3 = collect(ctx3.__soloZillowNormalize("zillow.profile.enrich", res3, { captured_at: CAPTURED_AT }));
        check("dom-fallback record still normalizes to one profile item", out3.items.length === 1, out3);

        console.log("\n-- zillow.profile.enrich: generic fail() error row is skipped, not normalized (zillow_extract.js:607-610) --");
        const errRaw = { capability: "zillow.profile.enrich", available: true, count: 0, status: "error", error: "boom", items: [{ capability: "zillow.profile.enrich", status: "error", error: "boom", url: "https://www.zillow.com/profile/x" }] };
        const errOut = collect(ctx3.__soloZillowNormalize("zillow.profile.enrich", errRaw, { captured_at: CAPTURED_AT }));
        check("error-only row normalizes to zero items", errOut.items.length === 0, errOut);

        console.log("\n-- zillow.agents.list / zillow.profile.enrich: unknown capability and default captured_at --");
        check("an unrecognised capability id normalizes to null", ctx3.__soloZillowNormalize("zillow.totally.unknown", { items: [{ id: "1" }] }, {}) === null);
        const before2 = Date.now();
        const defaulted = ctx3.__soloZillowNormalize("zillow.agents.list", { items: [{ profile_url: "https://www.zillow.com/profile/x", encoded_zuid: "X1-ZUx", name: "X" }] }, {});
        const parsed = Date.parse(defaulted.items[0].captured_at);
        check("captured_at defaults to now() when opts.captured_at is omitted", isFinite(parsed) && Math.abs(parsed - before2) < 5000, defaulted.items[0].captured_at);

        return finishZillowSuite();
      });
    });
  });
}

function finishZillowSuite() {
  console.log("\n-- cross-cutting: no sensitive key anywhere across every normalized item --");
  const hits = sensitiveKeys(ALL_ITEMS, "");
  check("zero sensitive-key hits across " + ALL_ITEMS.length + " normalized items", hits.length === 0, hits);

  console.log("\n-- cross-cutting: chrome-extension/core/schema.js validateRecord().ok --");
  if (fs.existsSync(SCHEMA_PATH)) {
    const SCHEMA_SRC = fs.readFileSync(SCHEMA_PATH, "utf8");
    const schemaCtx = { console };
    vm.createContext(schemaCtx);
    vm.runInContext(SCHEMA_SRC, schemaCtx);
    const SoloSchema = schemaCtx.SoloSchema;
    check("schema.js loaded and exposes SoloSchema.validateRecord", typeof SoloSchema.validateRecord === "function");
    let allOk = true;
    ALL_ITEMS.forEach((item, i) => {
      const r = SoloSchema.validateRecord(item);
      if (!r.ok) { allOk = false; console.log("    item[" + i + "] (" + item.kind + "/" + item.source_capability + ") errors: " + JSON.stringify(r.errors)); }
    });
    check("validateRecord().ok for all " + ALL_ITEMS.length + " normalized items", allOk);
  } else {
    console.log("  SKIP  chrome-extension/core/schema.js not found — schema validation skipped");
  }

  console.log("\n" + (fail === 0 ? "ALL " + pass + " CHECKS PASSED" : pass + " passed, " + fail + " FAILED"));
  process.exit(fail === 0 ? 0 : 1);
}
