// Solo Agency Local Collector — Zillow read capabilities (MAIN world).
//
//   zillow.agents.list     one agent-directory page  -> agent cards (ZillowAgentCard[])
//   zillow.profile.enrich  one agent/team profile    -> ProfileEnrich-compatible lead record
//
// WHY A SEPARATE FILE. gql_extract.js is the Facebook GraphQL/DOM library and is injected on
// every capability job; this file is injected by background.js ONLY when the job's capability
// starts with "zillow." and is dispatched through window.__soloZillowRun — so nothing on the
// Facebook path changes shape or timing. Same envelope contract as the DOM capabilities in
// gql_extract.js: { capability, available, count, items[], ...meta }. `available` is TRUE even
// when the page is blocked or empty (background.js nulls the whole record on available:false,
// which would make "blocked" and "collector crashed" indistinguishable from "nothing here").
//
// HOW IT READS. Both Zillow screens are Next.js pages: the complete, typed page state is in
// <script id="__NEXT_DATA__"> at load time, so the extractor parses that JSON and touches no
// CSS selector unless the JSON is missing (then a thin DOM fallback runs and `source` says so).
// Verified 2026-08-16 on the directory (list) page with and without ?name=<kw>, on a team-lead
// profile (Pardee Properties) and on a team-member profile (tykunkle).
//
// WHAT IT NEVER DOES. No pagination — the caller owns ?page=N (each job reads exactly the URL
// it was given). No clicking, no form filling, no captcha handling: Zillow's PerimeterX
// "Press & Hold" page is DETECTED (#px-captcha / title "Access to this page has been denied")
// and reported as status:"blocked". background.js (zillowHumanGate) normally catches that page
// BEFORE this lib runs — it plays a chime, brings the tab to the front and waits for the
// OPERATOR to pass the check, then continues — so a blocked record here means the operator
// never came within the wait ceiling. Never retry a blocked job automatically.
(function () {
  if (typeof window === "undefined") return;
  if (typeof window.__soloZillowRun === "function") return; // idempotent re-injection

  var VERSION = "zillow_extract 0.1.0";
  var CAP_LIST = "zillow.agents.list";
  var CAP_PROFILE = "zillow.profile.enrich";
  var ORIGIN = "https://www.zillow.com";
  var INDUSTRY_DEFAULT = "Real Estate";

  // ---------------------------------------------------------------- helpers
  function wait(ms) { return new Promise(function (r) { setTimeout(r, ms); }); }
  function isObj(v) { return !!v && typeof v === "object" && !Array.isArray(v); }
  function str(v) { return v == null ? "" : String(v); }
  function clean(v) { return str(v).replace(/\s+/g, " ").trim(); }
  function arr(v) { return Array.isArray(v) ? v : []; }
  function num(v) {
    if (v == null || v === "") return null;
    if (typeof v === "number") return isFinite(v) ? v : null;
    var m = str(v).replace(/,/g, "").match(/-?\d+(?:\.\d+)?/);
    return m ? Number(m[0]) : null;
  }
  function uniq(list) {
    var seen = {}, out = [];
    for (var i = 0; i < list.length; i++) {
      var k = clean(list[i]);
      if (!k || seen[k.toLowerCase()]) continue;
      seen[k.toLowerCase()] = 1; out.push(k);
    }
    return out;
  }
  function pushLine(lines, label, value) {
    var v = Array.isArray(value) ? uniq(value.map(clean)).join(", ") : clean(value);
    if (v) lines.push(label + ": " + v);
  }
  // Drop tracking params (utm_*, fbclid, gclid, msclkid) so the same site is one string
  // everywhere in the record — the bridge strips them from bare URL fields, but not from a
  // URL embedded in an about_lines sentence. Everything else in the query is kept.
  function stripTracking(u) {
    var s = str(u);
    var hashAt = s.indexOf("#"), frag = "";
    if (hashAt !== -1) { frag = s.slice(hashAt); s = s.slice(0, hashAt); }
    var qAt = s.indexOf("?");
    if (qAt === -1) return s + frag;
    var base = s.slice(0, qAt), kept = [];
    s.slice(qAt + 1).split("&").forEach(function (p) {
      if (!p) return;
      var k = p.split("=")[0].toLowerCase();
      if (/^utm_/.test(k) || k === "fbclid" || k === "gclid" || k === "msclkid") return;
      kept.push(p);
    });
    return base + (kept.length ? "?" + kept.join("&") : "") + frag;
  }
  function absUrl(u) {
    var s = clean(u);
    if (!s) return "";
    if (/^https?:\/\//i.test(s)) return stripTracking(s);
    if (s.charAt(0) === "/") return stripTracking(ORIGIN + s);
    return s;
  }
  function currentHref() { try { return String(location.href || ""); } catch (e) { return ""; } }
  function pageTitle() { try { return String(document.title || ""); } catch (e) { return ""; } }
  function bodyText() { try { return String((document.body && document.body.innerText) || ""); } catch (e) { return ""; } }
  function qs(sel) { try { return document.querySelector(sel); } catch (e) { return null; } }
  function qsa(sel) { try { return Array.prototype.slice.call(document.querySelectorAll(sel) || []); } catch (e) { return []; } }
  function attr(el, name) { try { return el && typeof el.getAttribute === "function" ? (el.getAttribute(name) || "") : ""; } catch (e) { return ""; } }
  function elText(el) { return el ? clean(el.innerText != null ? el.innerText : el.textContent) : ""; }

  // Zillow's Next.js state. null when the script tag is absent or unparsable.
  function nextData() {
    try {
      var el = document.getElementById("__NEXT_DATA__");
      if (!el) return null;
      var t = el.textContent || "";
      if (!t) return null;
      return JSON.parse(t);
    } catch (e) { return null; }
  }
  function pageProps(nd) { return nd && nd.props && isObj(nd.props.pageProps) ? nd.props.pageProps : null; }

  // PerimeterX bot check. The block page carries no __NEXT_DATA__, so it is checked FIRST —
  // otherwise "blocked" would read exactly like "empty page".
  function blockedReason() {
    if (qs("#px-captcha") || qs("#px-captcha-wrapper")) return "blocked_bot_check";
    var t = pageTitle();
    if (/access to this page has been denied/i.test(t)) return "blocked_bot_check";
    var b = bodyText().slice(0, 4000);
    if (/press\s*&\s*hold to confirm you are\s*a human/i.test(b)) return "blocked_bot_check";
    return "";
  }
  function blockedEnvelope(capId, t0) {
    var reason = blockedReason();
    return {
      capability: capId, available: true, count: 0, status: "blocked", reason: reason,
      url: currentHref(), source: "none", elapsed_ms: Date.now() - t0, version: VERSION,
      items: [{
        capability: capId, status: "blocked", reason: reason, url: currentHref(),
        hint: "Zillow bot check (PerimeterX 'Press & Hold'). background.js normally waits for the operator (chime + tab in front) and re-reads the page, so seeing this record means the wait ceiling passed or the check kept returning. Do NOT retry automatically: pause Zillow jobs on this Chrome profile, have the operator open zillow.com there and pass the check once by hand, then re-run."
      }]
    };
  }

  // getToKnowMe.description is HTML ("<p>Tyler was born …"). Strip to text, keep paragraphs.
  function stripHtml(html) {
    var s = str(html);
    if (!s) return "";
    s = s.replace(/<\s*br\s*\/?>/gi, "\n").replace(/<\/\s*(p|div|li|h[1-6])\s*>/gi, "\n").replace(/<[^>]+>/g, " ");
    s = s.replace(/&nbsp;/gi, " ").replace(/&amp;/gi, "&").replace(/&lt;/gi, "<").replace(/&gt;/gi, ">")
      .replace(/&quot;/gi, "\"").replace(/&#39;|&apos;/gi, "'").replace(/&#(\d+);/g, function (_, d) { return String.fromCharCode(Number(d)); });
    return s.split("\n").map(function (l) { return l.replace(/\s+/g, " ").trim(); }).filter(Boolean).join("\n");
  }
  function screenNameFromUrl(u) {
    var m = str(u).match(/\/profile\/([^\/?#]+)/i);
    if (!m) return "";
    try { return decodeURIComponent(m[1]); } catch (e) { return m[1]; }
  }
  function profileUrlFor(screenName) {
    var sn = clean(screenName);
    return sn ? ORIGIN + "/profile/" + encodeURIComponent(sn) : "";
  }
  function pageFromUrl(u) {
    var m = str(u).match(/[?&]page=(\d+)/i);
    return m ? Number(m[1]) : null;
  }
  function withPage(u, page) {
    var s = str(u) || currentHref();
    if (!s) return "";
    if (/[?&]page=\d+/i.test(s)) return s.replace(/([?&]page=)\d+/i, "$1" + page);
    return s + (s.indexOf("?") === -1 ? "?" : "&") + "page=" + page;
  }
  function digits(s) { return str(s).replace(/\D+/g, ""); }
  function uniqPhones(list) {
    var seen = {}, out = [];
    for (var i = 0; i < list.length; i++) {
      var p = clean(list[i]); if (!p) continue;
      var d = digits(p); if (d.length < 7) continue;
      if (seen[d]) continue; seen[d] = 1; out.push(p);
    }
    return out;
  }
  var EMAIL_RE = /^[^\s@]+@[^\s@]+\.[a-z]{2,}$/i;
  function uniqEmails(list) {
    var seen = {}, out = [];
    for (var i = 0; i < list.length; i++) {
      var e = clean(list[i]).replace(/^mailto:/i, "").split("?")[0].toLowerCase();
      if (!e || !EMAIL_RE.test(e) || seen[e]) continue;
      seen[e] = 1; out.push(e);
    }
    return out;
  }
  function toUnix(v) {
    var s = clean(v); if (!s) return null;
    var t = Date.parse(s);
    if (isNaN(t)) {
      var m = s.match(/^(\d{1,2})\/(\d{1,2})\/(\d{4})$/); // 8/14/2026 (M/D/YYYY)
      if (m) t = Date.UTC(Number(m[3]), Number(m[1]) - 1, Number(m[2]));
    }
    return isNaN(t) ? null : Math.round(t / 1000);
  }
  function toIsoDate(v) {
    var u = toUnix(v);
    if (u == null) return clean(v);
    try { return new Date(u * 1000).toISOString().slice(0, 10); } catch (e) { return clean(v); }
  }
  // "Page 1 of 24" lives in Zillow's pager nav (a visually hidden span). Authoritative when present.
  function pagerInfo() {
    var nav = qs('nav[aria-label="Agent pages"]') || qs('nav[aria-label*="pages" i]');
    if (!nav) return { page_count: null, page: null, next_enabled: null };
    var t = "";
    try { t = String(nav.textContent || ""); } catch (e) { t = ""; }
    var m = t.match(/Page\s+(\d+)\s+of\s+(\d+)/i);
    var next = null;
    try {
      var b = nav.querySelector('button[title="Next page"], button[rel="next"]');
      if (b) next = !(b.disabled || attr(b, "aria-disabled") === "true");
    } catch (e) { next = null; }
    return { page_count: m ? Number(m[2]) : null, page: m ? Number(m[1]) : null, next_enabled: next };
  }
  function socialKey(u) {
    var s = str(u).toLowerCase();
    if (/facebook\.com|fb\.com/.test(s)) return "facebook";
    if (/instagram\.com/.test(s)) return "instagram";
    if (/linkedin\.com/.test(s)) return "linkedin";
    if (/twitter\.com|(^|\/\/)(www\.)?x\.com/.test(s)) return "x";
    if (/tiktok\.com/.test(s)) return "tiktok";
    if (/youtube\.com|youtu\.be/.test(s)) return "youtube";
    return "";
  }

  // ------------------------------------------------------- zillow.agents.list
  // ONE directory page. Cards come from
  //   props.pageProps.displayData.agentDirectoryFinderDisplay.searchResults.results.resultsCards[]
  // and the page-level counters from searchResults + userInput. Verified 2026-08-16:
  // 16 cards/page, resultsFound 44,534 for los-angeles-ca (348 with ?name=kim), pager capped
  // at "Page 1 of 25" — so page_count, not resultsFound, is the walkable ceiling.
  function cardToItem(card) {
    if (!isObj(card)) return null;
    var url = absUrl(card.cardActionLink);
    var name = clean(card.cardTitle);
    if (!url && !name) return null;
    var stats = [], priceRange = null, sales12 = null, salesRegion = null, salesRegionLabel = "";
    arr(card.profileData).forEach(function (pd) {
      if (!isObj(pd)) return;
      var label = clean(pd.label), val = pd.formattedData == null ? null : clean(pd.formattedData);
      stats.push({ label: label, value: val });
      if (/price range/i.test(label)) priceRange = val;                       // "No recent price range" -> null value
      else if (/sales last 12 months/i.test(label)) sales12 = num(val);      // "No sales last 12 months" -> null
      else if (/sales in /i.test(label)) { salesRegion = num(val); salesRegionLabel = label; }
    });
    var ri = isObj(card.reviewInformation) ? card.reviewInformation : {};
    var reviewsCount = ri.reviewCountFormattedText != null ? num(ri.reviewCountFormattedText) : null;
    var tags = arr(card.tags).map(function (t) { return isObj(t) ? clean(t.text) : clean(t); }).filter(Boolean);
    var isTeam = tags.some(function (t) { return /^team$/i.test(t); }) || /team price range|team sales/i.test(stats.map(function (s) { return s.label; }).join(" "));
    return {
      profile_url: url,
      screen_name: screenNameFromUrl(url),
      encoded_zuid: clean(card.encodedZuid),
      name: name,
      brokerage: clean(card.secondaryCardTitle),
      is_team: isTeam,
      is_top_agent: !!card.isTopAgent,
      rating: ri.reviewAverage != null ? num(ri.reviewAverage) : null,
      reviews_count: reviewsCount,
      price_range: priceRange,
      sales_last_12_months: sales12,
      sales_in_region: salesRegion,
      sales_in_region_label: salesRegionLabel,
      photo_url: clean(card.imageUrl),
      tags: tags,
      stats: stats
    };
  }
  // Fallback when __NEXT_DATA__ is missing: every /profile/ anchor is a card. Thin on purpose —
  // it exists so a layout change degrades to "names + urls" instead of nothing.
  function domCards() {
    var out = [], seen = {};
    qsa('a[href*="/profile/"]').forEach(function (a) {
      var url = absUrl(attr(a, "href")).split("#")[0];
      if (!/zillow\.com\/profile\//i.test(url) || seen[url]) return;
      var name = elText(a);
      var box = a; for (var up = 0; up < 3 && box && box.parentElement; up++) box = box.parentElement;
      var txt = elText(box);
      if (!name || name.length > 80) {
        var h = box ? (box.querySelector && (box.querySelector("h2, h3, h4, [class*='Title']"))) : null;
        name = elText(h) || name;
      }
      if (!name) return;
      seen[url] = 1;
      out.push({
        profile_url: url, screen_name: screenNameFromUrl(url), encoded_zuid: "", name: name.slice(0, 120),
        brokerage: "", is_team: /\bTEAM\b/.test(txt), is_top_agent: /top agent/i.test(txt),
        rating: (function () { var m = txt.match(/(\d\.\d)\s*\(/); return m ? Number(m[1]) : null; })(),
        reviews_count: (function () { var m = txt.match(/\(([\d,]+)\)/); return m ? num(m[1]) : null; })(),
        price_range: null, sales_last_12_months: null, sales_in_region: null, sales_in_region_label: "",
        photo_url: "", tags: [], stats: []
      });
    });
    return out;
  }
  function agentsList(inputs) {
    inputs = inputs || {};
    var t0 = Date.now();
    if (blockedReason()) return blockedEnvelope(CAP_LIST, t0);
    var nd = nextData(), pp = pageProps(nd);
    var finder = pp && isObj(pp.displayData) ? pp.displayData.agentDirectoryFinderDisplay : null;
    var sr = isObj(finder) && isObj(finder.searchResults) ? finder.searchResults : null;
    var cards = sr && isObj(sr.results) && Array.isArray(sr.results.resultsCards) ? sr.results.resultsCards : null;
    var source = "next_data", items = [], seen = {}, nonProfileCards = 0;
    if (cards) {
      cards.forEach(function (c) {
        var it = cardToItem(c);
        if (!it) return;
        // Zillow mixes upsell cards into resultsCards ("Get help finding an agent" ->
        // /ods/submit_lead?…, measured live 2026-08-16). Only a /profile/ card is an agent.
        if (!/zillow\.com\/profile\//i.test(it.profile_url)) { nonProfileCards += 1; return; }
        var key = it.profile_url.toLowerCase();
        if (seen[key]) return; seen[key] = 1;
        items.push(it);
      });
    } else {
      source = "dom";
      items = domCards();
    }
    var ui = pp && isObj(pp.userInput) ? pp.userInput : {};
    var href = currentHref();
    var pager = pagerInfo();
    var page = num(ui.page) || (sr && num(sr.currentPage)) || pager.page || pageFromUrl(href) || 1;
    var resultsFound = sr ? num(sr.resultsFound) : null;
    var pageCount = pager.page_count, pageCountSource = pager.page_count ? "pager" : "";
    if (!pageCount && resultsFound != null) {
      // Zillow shows at most 25 directory pages per query, ~15 cards each. An ESTIMATE, marked as such.
      pageCount = Math.max(1, Math.min(25, Math.ceil(resultsFound / 15)));
      pageCountSource = "estimated";
    }
    var hasMore = pager.next_enabled != null ? pager.next_enabled : (pageCount ? page < pageCount : items.length >= 15);
    var nameKw = clean(ui.name);
    if (!nameKw) { try { nameKw = clean(new URLSearchParams(location.search).get("name")); } catch (e) { /* ignore */ } }
    var status = items.length ? "ok" : (nd ? "empty" : "no_next_data");
    return {
      capability: CAP_LIST, schema: "ZillowAgentCard[]", available: true, status: status,
      count: items.length, items: items, source: source, non_profile_cards: nonProfileCards,
      page: page, page_count: pageCount || null, page_count_source: pageCountSource || null,
      results_found: resultsFound, has_more: !!hasMore,
      next_page_url: hasMore ? withPage(href, page + 1) : null,
      query: {
        url: href,
        location_text: clean(ui.locationText),
        name: nameKw,
        applied_filters: arr(ui.appliedFilters),
        region_id: sr ? clean(sr.regionId) : "",
        region_name: sr ? clean(sr.titleLocation || sr.seoLocation) : "",
        results_found_label: sr ? clean(sr.resultsFoundFormattedLabel) : ""
      },
      elapsed_ms: Date.now() - t0, version: VERSION
    };
  }

  // ---------------------------------------------------- zillow.profile.enrich
  // ONE profile page -> a record shaped like fb.profile.enrich / ProfileDossier (same field
  // names, so harvest decide, the campaign UI drill-down and 04_VERIFY_ENRICH read it as-is),
  // plus a typed `zillow{}` block with what Zillow publishes and `industry` defaulted to
  // "Real Estate" — a fact of the source (agent directory), not an inference.
  function piEntry(pi, term) {
    for (var i = 0; i < pi.length; i++) {
      if (isObj(pi[i]) && new RegExp("^" + term + "$", "i").test(clean(pi[i].term))) return pi[i];
    }
    return null;
  }
  function personRef(p) {
    if (!isObj(p)) return null;
    var sn = clean(p.screenName);
    var r = isObj(p.ratings) ? p.ratings : {};
    return {
      name: clean(p.name), screen_name: sn, profile_url: profileUrlFor(sn), encoded_zuid: clean(p.encodedZuid),
      rating: r.average != null ? num(r.average) : null, reviews_count: r.count != null ? num(r.count) : null,
      is_top_agent: !!p.isTopAgent, photo_url: clean(p.profilePhotoUrl || p.profilePhotoSrc)
    };
  }
  function domProfileFallback(t0, nd) {
    var href = currentHref();
    var name = elText(qs("h1")) || clean(pageTitle().split(" - ")[0]);
    var emails = uniqEmails(qsa('a[href^="mailto:"]').map(function (a) { return attr(a, "href"); }));
    var phones = uniqPhones(qsa('a[href^="tel:"]').map(function (a) { return attr(a, "href").replace(/^tel:/i, ""); }));
    var crumbs = qsa('nav[aria-label*="readcrumb" i] a, ol[class*="readcrumb" i] a').map(elText).filter(Boolean);
    var sn = screenNameFromUrl(href);
    var item = baseRecord({
      profile_url: profileUrlFor(sn) || href, name: name, category: "", intro_bio: "", work: [], location: crumbs.length >= 2 ? [crumbs.slice(0, 2).join(", ")] : [],
      about: {}, about_lines: [], emails: emails, phones: phones, websites: [], website: "",
      found_on: emails.length ? "dom" : null, checked: ["dom"], missing: ["next_data"], source: "dom", elapsed_ms: Date.now() - t0,
      posts: [], timeline: { landed_on: "profile", posts_available: false, posts_reason: "no_next_data", posts_seen: 0, posts_kept: 0, videos_available: false, videos_seen: 0 },
      zillow: { screen_name: sn, next_data_present: !!nd }
    });
    return item;
  }
  function baseRecord(o) {
    // ProfileDossier / ProfileEnrich field set, in the same order the FB record uses, so a
    // reader that walks keys sees the familiar shape first and the zillow block last.
    return {
      profile_url: o.profile_url || "", name: o.name || "", category: o.category || "",
      follower_count: null, verified: false, intro_bio: o.intro_bio || "",
      work: o.work || [], education: [], location: o.location || [], intro_lines: [],
      about: o.about || {}, about_lines: o.about_lines || [],
      emails: o.emails || [], websites: o.websites || [], website: o.website || "", cta: [], phones: o.phones || [],
      found_on: o.found_on == null ? null : o.found_on, checked: o.checked || [], missing: o.missing || [],
      budget_exhausted: false, elapsed_ms: o.elapsed_ms || 0, discovered_tabs: [], graphql_by_surface: {},
      see_more_expansions: 0, about_panel_found: o.source === "next_data", source: o.source || "dom", graphql_about: null,
      posts: o.posts || [], videos: [], timeline: o.timeline || {},
      industry: INDUSTRY_DEFAULT,
      zillow: o.zillow || {}
    };
  }
  function profileEnrich(inputs) {
    inputs = inputs || {};
    var t0 = Date.now();
    if (blockedReason()) return blockedEnvelope(CAP_PROFILE, t0);
    var maxPosts = inputs.max_posts != null ? Math.max(0, Math.min(25, Number(inputs.max_posts) || 0)) : 5;
    var maxMembers = inputs.max_team_members != null ? Math.max(0, Math.min(500, Number(inputs.max_team_members) || 0)) : 200;
    var nd = nextData(), pp = pageProps(nd);
    var du = pp && isObj(pp.displayUser) ? pp.displayUser : null;
    if (!du) {
      var fb = domProfileFallback(t0, nd);
      return { capability: CAP_PROFILE, schema: "ZillowProfileEnrich", available: true, status: nd ? "no_display_user" : "no_next_data",
        count: 1, items: [fb], source: "dom", elapsed_ms: Date.now() - t0, version: VERSION };
    }
    var gk = isObj(pp.getToKnowMe) ? pp.getToKnowMe : {};
    var pi = arr(pp.professionalInformation);
    var licenses = arr(pp.agentLicenses);
    var areas = arr(pp.serviceAreas);
    var past = isObj(pp.pastSales) ? pp.pastSales : {};
    var reviews = pp.reviewsData && Array.isArray(pp.reviewsData.reviews) ? pp.reviewsData.reviews : [];
    var teamInfo = isObj(pp.teamDisplayInformation) ? pp.teamDisplayInformation : {};
    var g = isObj(pp.graphQLData) ? pp.graphQLData : {};
    var header = isObj(g.premiumAgentHeader) ? g.premiumAgentHeader : {};
    var statEntries = isObj(header.salesStats) && isObj(header.salesStats.entries) ? header.salesStats.entries : {};
    var statVal = function (k) { return isObj(statEntries[k]) ? clean(statEntries[k].value) : ""; };

    var screenName = clean(du.screenName) || clean(nd && nd.query && nd.query.screenName) || screenNameFromUrl(currentHref());
    var profileUrl = profileUrlFor(screenName) || currentHref();
    var name = clean(du.name);
    var title = clean(gk.title);
    var brokerage = clean(du.businessName);
    var addr = isObj(du.businessAddress) ? du.businessAddress : {};
    var cityState = uniq([clean(addr.city), clean(addr.state)]).join(", ") + (clean(addr.postalCode) ? " " + clean(addr.postalCode) : "");
    cityState = clean(cityState);
    var addressLine = uniq([clean(addr.address1), clean(addr.address2), cityState]).join(", ");

    // contact
    var phoneMap = isObj(du.phoneNumbers) ? du.phoneNumbers : {};
    var phoneList = [];
    ["cell", "business", "brokerage", "office", "phone"].forEach(function (k) { if (phoneMap[k]) phoneList.push(clean(phoneMap[k])); });
    Object.keys(phoneMap).forEach(function (k) { if (phoneMap[k] && phoneList.indexOf(clean(phoneMap[k])) === -1) phoneList.push(clean(phoneMap[k])); });
    pi.forEach(function (e) { if (isObj(e) && /phone/i.test(clean(e.term)) && e.description) phoneList.push(clean(e.description)); });
    qsa('a[href^="tel:"]').forEach(function (a) { phoneList.push(attr(a, "href").replace(/^tel:/i, "")); });
    var phones = uniqPhones(phoneList);
    var emails = uniqEmails([du.email].concat(qsa('a[href^="mailto:"]').map(function (a) { return attr(a, "href"); })));

    // links: the profile's own website + every link Zillow lists under "Websites" + the getToKnowMe socials
    var linkList = [];
    var websiteUrl = absUrl(gk.websiteUrl);
    if (websiteUrl) linkList.push(websiteUrl);
    var pageLinks = piEntry(pi, "Websites");
    arr(pageLinks && pageLinks.links).forEach(function (l) { if (isObj(l) && l.url) linkList.push(absUrl(l.url)); });
    var socials = {};
    [["facebook", gk.facebookUrl], ["instagram", gk.instagramUrl], ["linkedin", gk.linkedInUrl], ["x", gk.xUrl], ["tiktok", gk.tiktokUrl], ["youtube", gk.youtubeUrl]].forEach(function (p) {
      var u = absUrl(p[1]); if (u) { socials[p[0]] = u; linkList.push(u); }
    });
    linkList.forEach(function (u) { var k = socialKey(u); if (k && !socials[k]) socials[k] = u; });
    var websites = uniq(linkList);
    var website = websiteUrl || (function () { for (var i = 0; i < websites.length; i++) { if (!socialKey(websites[i]) && !/zillow\.com/i.test(websites[i])) return websites[i]; } return ""; })();

    // facts
    var memberSince = (function () { var e = piEntry(pi, "Member since"); return e ? clean(e.description) : ""; })();
    var licenseList = licenses.map(function (l) {
      return isObj(l) ? { number: clean(l.text), state: clean(l.state), status: clean(l.status), type: clean(l.license_type), expiration: clean(l.expiration) } : null;
    }).filter(Boolean);
    var licenseLines = licenseList.map(function (l) { return l.number + (l.state ? " (" + l.state + (l.status ? ", " + l.status : "") + (l.expiration ? ", exp " + l.expiration : "") + ")" : ""); });
    var specialties = arr(gk.specialties).map(clean).filter(Boolean);
    var languages = arr(gk.languages).map(clean).filter(Boolean);
    var serviceAreas = areas.map(function (a) { return isObj(a) ? clean(a.text) : clean(a); }).filter(Boolean);
    var ratings = isObj(du.ratings) ? du.ratings : {};
    var rating = { average: ratings.average != null ? num(ratings.average) : null, count: ratings.count != null ? num(ratings.count) : null };
    var sales = {
      last_12_months: num(statVal("salesLastTwelveMonths")),
      total: num(statVal("totalSales")) != null ? num(statVal("totalSales")) : (past.total != null ? num(past.total) : null),
      price_range: statVal("priceRange") || null,
      average_price: statVal("averagePrice") || null,
      raw: { sales_last_12_months: statVal("salesLastTwelveMonths"), total_sales: statVal("totalSales"), price_range: statVal("priceRange"), average_price: statVal("averagePrice") },
      is_team_total: !!(isObj(header.salesStats) && header.salesStats.teamDisclaimerSection)
    };
    var listings = {
      for_sale: pp.forSaleListings && pp.forSaleListings.listing_count != null ? num(pp.forSaleListings.listing_count) : null,
      for_rent: pp.forRentListings && pp.forRentListings.listing_count != null ? num(pp.forRentListings.listing_count) : null
    };
    var recentSales = arr(past.past_sales).slice(0, 10).map(function (s) {
      if (!isObj(s)) return null;
      return {
        price: clean(s.price), sold_date: toIsoDate(s.sold_date), sold_date_raw: clean(s.sold_date), represented: clean(s.represented),
        address: uniq([clean(s.street_address), clean(s.city_state_zipcode)]).join(", "), url: absUrl(s.home_details_url), zpid: s.zpid != null ? String(s.zpid) : ""
      };
    }).filter(Boolean);
    var recentReviews = reviews.slice(0, 10).map(function (r) {
      if (!isObj(r)) return null;
      var ree = isObj(r.reviewee) ? r.reviewee : {};
      return {
        id: r.reviewId != null ? String(r.reviewId) : "", date: toIsoDate(r.createDate), date_raw: clean(r.createDate), rating: r.rating != null ? num(r.rating) : null,
        snippet: clean(r.reviewComment).slice(0, 300), work_description: clean(r.workDescription),
        reviewee_name: uniq([clean(ree.firstName), clean(ree.lastName)]).join(" "), reviewee_screen_name: clean(ree.screenName)
      };
    }).filter(Boolean);

    // team
    var team = { role: null, team_name: "", lead: null, members: [], member_count: 0 };
    if (isObj(teamInfo.teamLeadInfo)) {
      var kids = arr(teamInfo.teamLeadInfo.children).map(personRef).filter(Boolean);
      team = { role: "lead", team_name: clean(teamInfo.teamLeadInfo.teamName), lead: null, members: kids.slice(0, maxMembers), member_count: kids.length };
    } else if (isObj(teamInfo.teamMemberInfo)) {
      team = { role: "member", team_name: clean(teamInfo.teamMemberInfo.teamName), lead: personRef(teamInfo.teamMemberInfo.teamLead), members: [], member_count: 0 };
    }

    // ProfileDossier-compatible text carriers
    var category = title || (team.role === "lead" ? "Real estate team lead" : "Real estate agent");
    var work = [];
    if (brokerage) work.push((title || "Real estate agent") + " at " + brokerage);
    else if (title) work.push(title);
    if (team.role === "lead" && team.team_name) work.push("Lead of team " + team.team_name + " (" + team.member_count + " members)");
    if (team.role === "member" && team.team_name) work.push("Member of team " + team.team_name + (team.lead && team.lead.name ? " (lead: " + team.lead.name + ")" : ""));
    var location = [];
    if (cityState) location.push(cityState);
    var about = {
      title: title, brokerage: brokerage, broker_address: addressLine, phones: phoneMap, email: emails[0] || "", website: website, socials: socials,
      licenses: licenseLines, years_in_industry: gk.yearsInIndustry != null ? num(gk.yearsInIndustry) : null, member_since: memberSince,
      specialties: specialties, languages: languages, service_areas: serviceAreas, sales: sales.raw, listings: listings, rating: rating,
      is_top_agent: !!du.isTopAgent, profile_types: arr(du.profileTypes).map(clean), team: { role: team.role, team_name: team.team_name, member_count: team.member_count }
    };
    var lines = [];
    pushLine(lines, "Name", name);
    pushLine(lines, "Title", title);
    pushLine(lines, "Brokerage", brokerage);
    pushLine(lines, "Broker address", addressLine);
    Object.keys(phoneMap).forEach(function (k) { pushLine(lines, k.charAt(0).toUpperCase() + k.slice(1) + " phone", phoneMap[k]); });
    pushLine(lines, "Email", emails.join(", "));
    pushLine(lines, "Website", website);
    Object.keys(socials).forEach(function (k) { pushLine(lines, k.charAt(0).toUpperCase() + k.slice(1), socials[k]); });
    pushLine(lines, "Real Estate Licenses", licenseLines);
    pushLine(lines, "Years in industry", gk.yearsInIndustry != null ? String(gk.yearsInIndustry) : "");
    pushLine(lines, "Member since", memberSince);
    pushLine(lines, "Specialties", specialties);
    pushLine(lines, "Languages", languages);
    if (serviceAreas.length) lines.push("Service areas (" + serviceAreas.length + "): " + serviceAreas.slice(0, 40).join("; "));
    pushLine(lines, "Sales last 12 months", sales.raw.sales_last_12_months);
    pushLine(lines, "Total sales", sales.raw.total_sales);
    pushLine(lines, "Price range", sales.raw.price_range);
    pushLine(lines, "Average price", sales.raw.average_price);
    if (sales.is_team_total) lines.push("Sales numbers represent all team members");
    if (listings.for_sale != null || listings.for_rent != null) lines.push("Listings: " + (listings.for_sale != null ? listings.for_sale + " for sale" : "") + (listings.for_sale != null && listings.for_rent != null ? ", " : "") + (listings.for_rent != null ? listings.for_rent + " for rent" : ""));
    if (rating.count != null) lines.push("Reviews: " + (rating.average != null ? rating.average : "-") + " (" + rating.count + ")");
    if (du.isTopAgent) lines.push("Top Agent on Zillow");
    if (team.role === "lead") lines.push("Team: lead of " + team.team_name + " (" + team.member_count + " members)");
    if (team.role === "member") lines.push("Team: member of " + team.team_name + (team.lead && team.lead.name ? " (lead: " + team.lead.name + ")" : ""));
    var aboutLines = lines.slice(0, 120);

    // posts[] = the DATED public activity Zillow publishes (reviews received + closed sales),
    // newest first, so Stage 4's "recent dated signal" logic and the UI drill-down work unchanged.
    var posts = [];
    recentReviews.forEach(function (r) {
      var ct = toUnix(r.date_raw);
      posts.push({ kind: "review", id: r.id ? "review:" + r.id : "", text: "[Review" + (r.rating != null ? " " + r.rating + "/5" : "") + "] " + (r.work_description ? r.work_description + " — " : "") + r.snippet,
        date: r.date, created_time: ct, url: profileUrl + "#reviews", actor: { type: "profile", id: clean(du.encodedZuid), name: name, url: profileUrl } });
    });
    recentSales.forEach(function (s) {
      var ct = toUnix(s.sold_date_raw);
      posts.push({ kind: "sale", id: s.zpid ? "sale:" + s.zpid : "", text: "[Sold" + (s.represented ? ", represented " + s.represented : "") + "] " + s.price + (s.address ? " — " + s.address : ""),
        date: s.sold_date, created_time: ct, url: s.url, actor: { type: "profile", id: clean(du.encodedZuid), name: name, url: profileUrl } });
    });
    posts.sort(function (a, b) { return (b.created_time || 0) - (a.created_time || 0); });
    var postsSeen = posts.length;
    posts = posts.slice(0, maxPosts);

    var item = baseRecord({
      profile_url: profileUrl, name: name, category: category, intro_bio: stripHtml(gk.description), work: work, location: location,
      about: about, about_lines: aboutLines, emails: emails, phones: phones, websites: websites, website: website,
      found_on: emails.length ? "next_data" : null, checked: ["next_data", "dom"], missing: [], source: "next_data", elapsed_ms: 0,
      posts: posts,
      timeline: { landed_on: "profile", posts_available: postsSeen > 0, posts_reason: postsSeen > 0 ? "zillow_reviews_and_sales" : "no_reviews_or_sales_published", posts_seen: postsSeen, posts_kept: posts.length, videos_available: false, videos_seen: 0 },
      zillow: {
        screen_name: screenName, encoded_zuid: clean(du.encodedZuid), profile_types: arr(du.profileTypes).map(clean),
        is_top_agent: !!du.isTopAgent, title: title, brokerage: brokerage,
        business_address: { address1: clean(addr.address1), address2: clean(addr.address2), city: clean(addr.city), state: clean(addr.state), postal_code: clean(addr.postalCode), line: addressLine },
        phones: phoneMap, email: emails[0] || "", website: website, socials: socials,
        years_in_industry: gk.yearsInIndustry != null ? num(gk.yearsInIndustry) : null, member_since: memberSince,
        specialties: specialties, languages: languages, service_areas: serviceAreas.slice(0, 100), service_area_count: serviceAreas.length,
        licenses: licenseList, rating: rating, sales: sales, listings: listings,
        recent_sales: recentSales, recent_reviews: recentReviews, team: team, photo_url: clean(du.profilePhotoSrc),
        page_title: pageTitle()
      }
    });
    item.elapsed_ms = Date.now() - t0;
    return { capability: CAP_PROFILE, schema: "ZillowProfileEnrich", available: true, status: "ok", count: 1, items: [item], source: "next_data", elapsed_ms: item.elapsed_ms, version: VERSION };
  }

  // ------------------------------------------------------------- dispatch
  var CAPS = {};
  CAPS[CAP_LIST] = agentsList;
  CAPS[CAP_PROFILE] = profileEnrich;

  function fail(capId, e) {
    return { capability: capId, available: true, count: 0, status: "error", error: String(e && e.message || e), version: VERSION,
      items: [{ capability: capId, status: "error", error: String(e && e.message || e), url: currentHref() }] };
  }
  // The Next.js payload is in the initial HTML, but give a slow document one short grace period
  // before concluding it is absent (never for a blocked page — that answer is final).
  function runOnce(capId, inputs) {
    var fn = CAPS[capId];
    if (!fn) return Promise.resolve(fail(capId, "no zillow extractor for " + capId));
    try { return Promise.resolve(fn(inputs)); } catch (e) { return Promise.resolve(fail(capId, e)); }
  }
  window.__soloZillowRun = function (capId, inputs) {
    capId = String(capId || ""); inputs = inputs && typeof inputs === "object" ? inputs : {};
    return runOnce(capId, inputs).then(function (res) {
      if (res && (res.status === "no_next_data") && !blockedReason()) {
        return wait(1500).then(function () { return runOnce(capId, inputs); });
      }
      return res;
    }).catch(function (e) { return fail(capId, e); });
  };
  window.__soloZillowCapabilities = Object.keys(CAPS);
  window.__soloZillowVersion = VERSION;
  // Exposed for the offline harness (tests/test_zillow_extract.js); not used by background.js.
  window.__soloZillowInternals = { agentsList: agentsList, profileEnrich: profileEnrich, stripHtml: stripHtml, cardToItem: cardToItem, blockedReason: blockedReason, withPage: withPage, toIsoDate: toIsoDate };
})();
