/*
 * infinity_loops.js
 * Per-scroll capture for the Solo Agency Local Collector.
 *
 * Loaded AFTER collector_helpers.js and readability.js into the same injected
 * (isolated-world) context, so it can call:
 *   - from collector_helpers.js : getHumanReadableText(), cleanText()
 *   - from readability.js       : Readability, convertRelativeUrlsToAbsolute(),
 *                                 getBaseLocation(), mergeAndRemoveDuplicateElementsOrdered()
 *
 * Two source families, one shared output format ("text(url)" inline links):
 *   - platform / feed (facebook, linkedin, instagram, tiktok, x, reddit, youtube)
 *       -> capture the visible feed verbatim, keep stats + links, only strip
 *          page-level chrome (nav/banner/search). Posts are NOT cleaned.
 *   - website (forum / news / blog, platform === "web")
 *       -> run Readability to drop menu/topbar/recommendations, then serialize.
 *
 * Across scrolls the caller passes the previously-merged text back in; we merge
 * with mergeAndRemoveDuplicateElementsOrdered() (dedupes repeated sentences but
 * always keeps URLs and preserves order), so re-capturing overlapping posts on
 * the next scroll does not duplicate content.
 *
 * Exposes window.__collectorCapture(prevText, opts) -> result object.
 */
(function () {
  "use strict";

  function inferPlatformInPage(url) {
    var u = String(url || "").toLowerCase();
    if (u.indexOf("facebook.com") !== -1) return "facebook";
    if (u.indexOf("linkedin.com") !== -1) return "linkedin";
    if (u.indexOf("instagram.com") !== -1) return "instagram";
    if (u.indexOf("tiktok.com") !== -1) return "tiktok";
    if (u.indexOf("reddit.com") !== -1) return "reddit";
    if (u.indexOf("youtube.com") !== -1 || u.indexOf("youtu.be") !== -1) return "youtube";
    if (u.indexOf("x.com") !== -1 || u.indexOf("twitter.com") !== -1) return "x";
    return "web";
  }

  function countUrls(text) {
    var m = String(text || "").match(/https?:\/\/[^\s)]+/g);
    return m ? m.length : 0;
  }

  function baseLocation(url) {
    if (typeof getBaseLocation === "function") {
      try { return getBaseLocation(url); } catch (e) { /* fall through */ }
    }
    return location.protocol + "//" + location.host;
  }

  function toAbsolute(html, base) {
    if (typeof convertRelativeUrlsToAbsolute === "function") {
      try { return convertRelativeUrlsToAbsolute(html, base); } catch (e) { /* keep */ }
    }
    return html;
  }

  // ---- Structural extraction (platform/HTML-agnostic) ---------------------
  // Works off DOM SEMANTICS, not site-specific class names: strip non-content +
  // ARIA/landmark chrome, harvest universal engagement aria-labels, serialize
  // remaining text in DOM order with text(url) link annotation.

  var SKIP_TAGS = ['script', 'style', 'noscript', 'svg', 'img', 'link', 'meta', 'iframe', 'input', 'textarea', 'select', 'path', 'video', 'audio'];
  var BLOCK_TAGS = { DIV:1, P:1, LI:1, UL:1, OL:1, SECTION:1, ARTICLE:1, HEADER:1, FOOTER:1, H1:1, H2:1, H3:1, H4:1, H5:1, H6:1, BR:1, TR:1, BLOCKQUOTE:1 };
  var CHROME_ROLES = ['navigation', 'banner', 'contentinfo', 'complementary', 'dialog', 'toolbar', 'menu', 'menubar', 'menuitem', 'combobox', 'search', 'progressbar', 'tablist', 'tab', 'status', 'alert', 'listbox', 'slider', 'separator', 'presentation'];
  // number + a UNIVERSAL social-engagement noun (works on any platform, not FB-specific).
  var ENGAGEMENT_RE = /\d[\d.,]*\s*[KMkm]?\b[^.]{0,12}?(reaction|comment|share|view|repl|answer|reacted)|(?:reaction|comment|share|view)s?\b[^.]{0,3}\d/i;

  // Remove non-content + chrome from a parsed clone (never the live page).
  function stripForExtraction(doc) {
    try {
      doc.querySelectorAll(SKIP_TAGS.join(",")).forEach(function (n) { n.remove(); });
      doc.querySelectorAll('[aria-hidden="true"]').forEach(function (n) { n.remove(); });
      CHROME_ROLES.forEach(function (r) {
        doc.querySelectorAll('[role="' + r + '"]').forEach(function (n) { n.remove(); });
      });
      doc.querySelectorAll('nav,header,footer,aside').forEach(function (n) { n.remove(); });
    } catch (e) { /* best effort */ }
  }

  // Walk DOM in order. <a> -> text(href); block tags -> newline; engagement
  // aria-labels ("194 reactions", "16 comments") harvested so counts survive
  // even when the platform renders them only in aria-label.
  function serializeStructured(node, out) {
    if (!node) return;
    if (node.nodeType === 3) {
      var t = String(node.nodeValue || "").replace(/\s+/g, " ");
      if (t.trim()) out.push(t);
      return;
    }
    if (node.nodeType !== 1) return;
    var tag = (node.tagName || "").toUpperCase();
    if (SKIP_TAGS.indexOf(tag.toLowerCase()) !== -1) return;

    var al = node.getAttribute && node.getAttribute("aria-label");
    if (al && ENGAGEMENT_RE.test(al)) {
      out.push(String(al).replace(/;.*$/, "").replace(/\s+/g, " ").trim());
      return; // engagement control is a leaf; avoid double-counting its number child
    }

    if (tag === "A") {
      var href = node.getAttribute("href") || "";
      var txt = String(node.textContent || "").replace(/\s+/g, " ").trim();
      if (href && !/^(javascript:|#|void)/i.test(href)) out.push((txt || "") + "(" + href + ")");
      else if (txt) out.push(txt);
      return;
    }

    var kids = node.childNodes;
    for (var i = 0; i < kids.length; i++) serializeStructured(kids[i], out);
    if (BLOCK_TAGS[tag]) out.push("\n");
  }

  // HTML chunk -> annotated text(url). Parses a CLONE so the live page is untouched.
  function extractStructured(html) {
    try {
      var parser = new DOMParser();
      var doc = parser.parseFromString(html, "text/html");
      stripForExtraction(doc);
      var out = [];
      serializeStructured(doc.body || doc.documentElement, out);
      return out.join(" ")
        .replace(/[ \t]*\n[ \t]*/g, "\n")
        .replace(/\n{2,}/g, "\n")
        .replace(/ {2,}/g, " ")
        .trim();
    } catch (e) {
      return "";
    }
  }

  function pickFeedRoot() {
    return document.querySelector('[role="feed"]')
        || document.querySelector('[role="main"]')
        || document.querySelector('main')
        || document.body;
  }

  // One scroll snapshot of a feed/timeline -> annotated text(url).
  function captureFeedText(url, base) {
    var root = pickFeedRoot();
    var html = root ? root.outerHTML : (document.body ? document.body.outerHTML : "");
    html = toAbsolute(html, base);
    return cleanText(extractStructured(html));
  }

  // Article page -> Readability strips boilerplate, then annotated text(url).
  function captureArticleText(url, base) {
    var html = document.documentElement
      ? document.documentElement.outerHTML
      : (document.body ? document.body.outerHTML : "");
    html = toAbsolute(html, base);

    var cleaned = "";
    try {
      var parser = new DOMParser();
      var doc = parser.parseFromString(html, "text/html"); // clone: Readability mutates it
      var loc = new URL(url);
      var uri = {
        spec: loc.href,
        host: loc.host,
        prePath: loc.protocol + "//" + loc.host,
        scheme: loc.protocol.replace(":", ""),
        pathBase: loc.origin + loc.pathname.substring(0, loc.pathname.lastIndexOf("/") + 1)
      };
      if (typeof Readability === "function") {
        var res = new Readability(uri, doc).parse();
        if (res && res.content) cleaned = res.content;
      }
    } catch (e) { /* fall back below */ }

    // No whole-page fallback: dumping an index/homepage here floods the output
    // with nav/menu links. Index pages are handled by SocialHtmlFilter upstream;
    // if Readability found no article, return empty so the caller can decide.
    if (!cleaned) return "";
    return cleanText(extractStructured(cleaned));
  }

  // Block-level merge for the SocialHtmlFilter (primary) path: its output is
  // already clean, per-post blocks separated by blank lines. Across scrolls we
  // just dedup whole blocks (ignoring the volatile "Post N" counter).
  function mergeBlocks(prevText, captureText) {
    var combined = (prevText ? prevText + "\n\n" : "") + (captureText || "");
    var blocks = combined.split(/\n\s*\n/);
    var seen = new Set();
    var out = [];
    blocks.forEach(function (b) {
      var norm = b.replace(/^(?:Post|Item)\s+\d+\s*/i, "").replace(/\s+/g, " ").trim().toLowerCase();
      if (!norm || seen.has(norm)) return;
      seen.add(norm);
      out.push(b.trim());
    });
    return out.join("\n\n");
  }

  // Prose-oriented merge (article/website branch).
  function mergeText(prevText, captureText) {
    if (!prevText || !String(prevText).trim()) return captureText;
    if (typeof mergeAndRemoveDuplicateElementsOrdered === "function") {
      try {
        return cleanText(mergeAndRemoveDuplicateElementsOrdered(prevText, captureText));
      } catch (e) { /* fall back */ }
    }
    return cleanText(prevText + "\n" + captureText);
  }

  // ----- Phase 2: final filter for FEED text -------------------------------
  // The serializer's text(url) output, accumulated across scrolls, is full of
  // navigation junk (bare links with tiny labels), repeated brand tokens, and
  // FB decoy strings. This phase cuts that without parsing platform HTML.

  // All rules below are platform-agnostic: no site names, no URL-path lists.
  var DEFAULT_FILTER = {
    repeatRun: 5,                 // remove a token repeated >= this many times in a row
    gibberishLen: 25,             // drop letter+digit runs at least this long (no spaces)
    shortenSameDomainUrls: true,  // same-domain URLs -> path only; external URLs kept full
    trimUrlBloat: true,           // drop bloated query params (tracking blobs)
    maxParamValueChars: 40        // a query param value longer than this AND non-numeric is dropped
  };

  // Registrable-ish root domain (last two labels): facebook.com, lilly.com.
  function rootOf(host) {
    var parts = String(host || "").split(".");
    return parts.slice(-2).join(".");
  }

  // Final OUTPUT transform for URLs (accumulator keeps absolute URLs so
  // cross-scroll dedup still works). Two passes, both platform-agnostic:
  //   1. trimUrlBloat: drop query params whose value is long AND non-numeric
  //      (opaque tracking blobs like __cft__/__tn__); keep short params and
  //      numeric ids (comment_id, fbid, v=...) so the link's identity survives.
  //   2. shortenSameDomainUrls: drop scheme+host for URLs on the same site as
  //      the page (keep path); external URLs are left full so the LLM still sees
  //      which other site a link points to.
  function cleanUrlsInText(text, pageHost, opts) {
    opts = Object.assign({}, DEFAULT_FILTER, opts || {});
    var pageRoot = opts.shortenSameDomainUrls ? rootOf(pageHost) : "";
    var maxV = opts.maxParamValueChars || 40;

    return String(text).replace(/https?:\/\/[^\s)]+/g, function (u) {
      var hashIdx = u.indexOf("#");
      var hash = hashIdx >= 0 ? u.slice(hashIdx) : "";
      if (hash === "#") hash = "";
      var noHash = hashIdx >= 0 ? u.slice(0, hashIdx) : u;
      var qIdx = noHash.indexOf("?");
      var base = qIdx >= 0 ? noHash.slice(0, qIdx) : noHash;
      var query = qIdx >= 0 ? noHash.slice(qIdx + 1) : "";

      if (opts.trimUrlBloat !== false && query) {
        query = query.split("&").filter(function (pair) {
          var eq = pair.indexOf("=");
          var v = eq >= 0 ? pair.slice(eq + 1) : "";
          return v.length <= maxV || /^[0-9]+$/.test(v);
        }).join("&");
      }

      var out = base + (query ? "?" + query : "") + hash;

      if (pageRoot) {
        var m = out.match(/^https?:\/\/([^\/]+)(.*)$/);
        if (m && rootOf(m[1]) === pageRoot) {
          out = m[2] || "/";
        }
      }
      return out;
    });
  }

  // Remove a meaningless run of the SAME token repeated >= repeatRun times in a
  // row (e.g. "Facebook Facebook Facebook Facebook Facebook ..."). The whole run
  // is deleted, not collapsed to one.
  function collapseRepeatedTokens(text, repeatRun) {
    var more = Math.max(1, (repeatRun || 5) - 1); // first match + this many more
    try {
      return String(text).replace(new RegExp("(\\b[\\p{L}\\p{N}][\\p{L}\\p{N}'’-]{0,40}\\b)(?:\\s+\\1\\b){" + more + ",}", "giu"), " ");
    } catch (e) {
      return String(text).replace(new RegExp("(\\b[\\w'’-]{1,40}\\b)(?:\\s+\\1\\b){" + more + ",}", "gi"), " ");
    }
  }

  // Remove standalone alphanumeric runs that mix letters AND digits and are
  // long with no spaces -- FB injects scrambled decoy tokens like that. Run
  // this on label text only (URLs already stripped) so real URLs are safe.
  function dropGibberish(text, minLen) {
    try {
      var re = new RegExp("\\b(?=\\w*[A-Za-z])(?=\\w*\\d)[A-Za-z0-9]{" + minLen + ",}\\b", "g");
      return String(text).replace(re, " ");
    } catch (e) {
      return text;
    }
  }

  // Detect a scrambled decoy token (FB anti-scrape: obfuscated account names made
  // of per-character spans) WITHOUT hitting real tokens. Two signatures:
  //   (a) mixes letters+digits with a lowercase letter and the two interleave a
  //       lot (>=3 transitions) -- catches "DCsQ3O2.comNguyen", "8c7M";
  //   (b) length >= 10 with an INTERNAL capital (lowercase then an uppercase
  //       later) -- catches pure-letter scrambles like "osnptdSoerait".
  // Real words, acronyms (EB2NIW/USDA), money ($5000), ranges (10-50), timestamps
  // (3h), brand/model names (gpt-4o-mini, iPhone15, McDonald, LinkedIn) survive.
  function looksLikeDecoyToken(tok) {
    try {
      var t = tok.replace(/^[^\p{L}\p{N}]+|[^\p{L}\p{N}]+$/gu, "");
      if (t.length < 4) return false;
      if (/[0-9]/.test(t) && /\p{L}/u.test(t) && /\p{Ll}/u.test(t)) {
        var trans = 0;
        for (var i = 1; i < t.length; i++) {
          var pD = /[0-9]/.test(t[i - 1]), cD = /[0-9]/.test(t[i]);
          var pL = /\p{L}/u.test(t[i - 1]), cL = /\p{L}/u.test(t[i]);
          if ((pD && cL) || (pL && cD)) trans++;
        }
        if (trans >= 3) return true;
      }
      if (t.length >= 10 && /\p{Ll}/u.test(t) && /\p{Ll}.*\p{Lu}/u.test(t)) return true;
      return false;
    } catch (e) {
      return false;
    }
  }

  function dropDecoyTokens(text) {
    return String(text).split(/\s+/).filter(function (tok) {
      return !looksLikeDecoyToken(tok);
    }).join(" ");
  }

  // A "real word" token = letters only, length >= 3 (any script), not a decoy.
  function hasRealWord(label) {
    return label.split(/\s+/).some(function (w) {
      if (looksLikeDecoyToken(w)) return false;
      try { return /^[\p{L}][\p{L}'’\-]{2,}$/u.test(w); }
      catch (e) { return /^[A-Za-z][A-Za-z'’\-]{2,}$/.test(w); }
    });
  }

  // A URL has a meaningful path if it points beyond the site root (e.g.
  // /posts/<id>, /user/<id>, /<username>) -- i.e. it carries identity.
  function hasMeaningfulPath(urls) {
    return urls.some(function (x) {
      try { return new URL(x).pathname.replace(/\/+$/, "").length > 0; }
      catch (e) { return false; }
    });
  }

  // Split text into URL-anchored units: each piece is "leading text + one URL",
  // plus any trailing free text. Newlines are split first.
  function tokenizeUnits(text) {
    var units = [];
    String(text).split(/\n+/).forEach(function (line) {
      var re = /https?:\/\/[^\s)]+\)?/g;
      var last = 0, m;
      while ((m = re.exec(line)) !== null) {
        units.push(line.slice(last, re.lastIndex));
        last = re.lastIndex;
      }
      if (last < line.length) units.push(line.slice(last));
    });
    return units.map(function (s) { return s.trim(); }).filter(Boolean);
  }

  function finalFilterFeed(rawText, opts) {
    opts = Object.assign({}, DEFAULT_FILTER, opts || {});
    var text = collapseRepeatedTokens(rawText, opts.repeatRun);
    var rawUnits = tokenizeUnits(text);

    var seen = new Set();
    var kept = [];
    var dropped = 0;

    rawUnits.forEach(function (u) {
      var urls = u.match(/https?:\/\/[^\s)]+/g) || [];

      // Build a clean label = unit text with URLs and serializer artifacts removed.
      var label = u.replace(/https?:\/\/[^\s)]+/g, " ");
      label = label.replace(/\(\s*\.?\s*\)/g, " ") // empty ( ) or (.)
                   .replace(/\.{2,}/g, " ")          // runs of dots
                   .replace(/[()]/g, " ");           // leftover parens
      label = dropGibberish(label, opts.gibberishLen);
      label = dropDecoyTokens(label);
      label = label.replace(/\s+/g, " ").trim();

      // Drop contentless / pure-decoy units: keep a unit only if it has a real
      // word OR a URL with a meaningful path (post/profile/comment link). This
      // needs BOTH to be absent before dropping, so real info is never lost.
      if (!hasRealWord(label) && !hasMeaningfulPath(urls)) { dropped++; return; }

      // Dedup by content: normalized label + URL path WITHOUT volatile query
      // (__cft__/__tn__ differ per capture), so re-captured posts collapse.
      var keyUrls = urls.map(function (x) { return x.split("?")[0].split("#")[0]; }).join(" ");
      var key = label.toLowerCase() + "|" + keyUrls;
      if (seen.has(key)) { dropped++; return; }
      seen.add(key);

      kept.push(urls.length ? (label ? label + " " : "") + urls.join(" ") : label);
    });

    return { text: kept.join("\n"), kept: kept.length, dropped: dropped };
  }

  /*
   * prevText : merged text accumulated from earlier scroll captures ("" to start)
   * opts     : { platform? }  -- platform override; otherwise inferred from URL
   */
  window.__collectorCapture = function (prevText, opts) {
    opts = opts || {};
    var url = location.href;
    var base = baseLocation(url);
    var platform = opts.platform || inferPlatformInPage(url);
    var isFeed = platform !== "web";
    var branch = isFeed ? "feed" : "website";

    var captureText = "";
    var engine = "";
    try {
      // SocialHtmlFilter = generic LIST/FEED extractor (social feeds + news
      // index/homepages). Computed once so the web branch can use it as fallback.
      var filterResult = null;
      if (typeof SocialHtmlFilter !== "undefined" && SocialHtmlFilter && SocialHtmlFilter.filterCurrentPage) {
        try { filterResult = SocialHtmlFilter.filterCurrentPage({ currentUrl: url, outputBaseUrl: url }); }
        catch (e) { filterResult = null; }
      }
      var fText = (filterResult && filterResult.text) ? String(filterResult.text).trim() : "";

      if (isFeed) {
        // Social feed -> filtering primary, structural extractor as fallback.
        if (fText) { captureText = fText; engine = "filtering"; }
        else { captureText = captureFeedText(url, base); engine = "infinity_loops"; }
      } else {
        // Web -> a SINGLE ARTICLE is the norm, so Readability goes FIRST (it
        // returns "" when there is no article body). Only if there is no
        // article (homepage/index/category) do we use the list extractor.
        // This stops related-article cards on an article page from hijacking it.
        captureText = captureArticleText(url, base);
        if (captureText) { engine = "readability"; }
        else if (fText) { captureText = fText; engine = "filtering"; }
        else { engine = "readability"; }
      }
    } catch (e) {
      return {
        error: String(e && e.message ? e.message : e),
        url: url, title: document.title || "", platform: platform, branch: branch
      };
    }

    // merged = accumulator fed back as prevText next scroll. mergedDisplay = LLM-facing.
    var merged, mergedDisplay, dropped = 0, keptUnits = 0;
    var filterOpts = opts.filter || {};
    if (engine === "filtering") {
      // SocialHtmlFilter already cleans URLs + dedups within a snapshot; across
      // scrolls just dedup blocks. No extra URL pass needed.
      merged = mergeBlocks(prevText, captureText);
      mergedDisplay = merged;
    } else if (engine === "infinity_loops") {
      var f = finalFilterFeed((prevText ? prevText + "\n" : "") + captureText, filterOpts);
      merged = f.text;
      dropped = f.dropped;
      keptUnits = f.kept;
      mergedDisplay = cleanUrlsInText(merged, location.host, filterOpts);
    } else {
      merged = mergeText(prevText, captureText);
      mergedDisplay = merged;
    }

    return {
      url: url,
      title: document.title || "",
      platform: platform,
      branch: branch,
      engine: engine,
      scrollY: Math.round(window.scrollY || 0),
      captureText: captureText,
      merged: merged,               // accumulator
      mergedDisplay: mergedDisplay, // LLM-facing
      captureChars: captureText.length,
      mergedChars: mergedDisplay.length,
      captureUrls: countUrls(captureText),
      mergedUrls: countUrls(mergedDisplay),
      droppedUnits: dropped,
      keptUnits: keptUnits,
      // Structured aux (only when the filtering engine ran) so the automated
      // collector can fill the same data_point fields it always did.
      accountUrls: (filterResult && filterResult.links && filterResult.links.accounts) ? filterResult.links.accounts.filter(Boolean) : [],
      postUrls: (filterResult && filterResult.links && filterResult.links.posts) ? filterResult.links.posts.filter(Boolean) : [],
      entityItems: (filterResult && filterResult.items) ? filterResult.items : []
    };
  };
})();
