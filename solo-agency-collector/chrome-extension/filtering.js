/*
 * Platform-neutral social DOM text filter.
 *
 * Audit notes for future agents:
 * - Keep this pure code: no LLM calls, no network, no platform SDK assumptions.
 * - Prefer HTML/DOM semantics, URL intent, ARIA/title/alt/text, and metric words;
 *   treat class/data-* names only as weak hints because social platforms churn them.
 * - Test on live DOM, not only documentElement.outerHTML. YouTube-like apps can
 *   hydrate cards through web components/open shadow roots that raw outerHTML misses.
 * - When improving one platform, run cross-platform fixtures. Facebook comments,
 *   YouTube shelves/Shorts, TikTok action counters, and X analytics links all
 *   look post-like but can be navigation, comments, or sibling cards.
 * - Same-domain output should stay path-only with junk params stripped, while
 *   preserving identity params such as fbid/story_fbid/v/id/post_id.
 *
 * Usage in a browser/content script:
 *   const data = SocialHtmlFilter.filterCurrentPage();
 *   // or
 *   const data = SocialHtmlFilter.filterSocialHtml(document.documentElement.outerHTML, {
 *     currentUrl: location.href,
 *   });
 *
 * Usage in Node:
 *   node filtering.js https://www.facebook.com/... < current-dom.html
 */
(function attach(root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  } else {
    root.SocialHtmlFilter = factory();
  }
})(typeof globalThis !== "undefined" ? globalThis : this, function factory() {
  "use strict";

  const VERSION = "1.0.0";
  const MAX_WALK_NODES = 30000;
  const MAX_POSTS = 80;
  const MAX_CONTENT_CHARS = 5000;

  const BLOCKED_TAGS = new Set([
    "script",
    "style",
    "noscript",
    "template",
    "svg",
    "canvas",
    "iframe",
    "head",
    "meta",
    "link",
    "input",
    "textarea",
    "select",
    "option",
  ]);

  const NOISE_ROLES = new Set([
    "banner",
    "navigation",
    "menu",
    "menubar",
    "search",
    "dialog",
    "alertdialog",
    "tooltip",
    "presentation",
  ]);

  const UI_TEXT = [
    "home",
    "watch",
    "reels",
    "friends",
    "groups",
    "group by",
    "public group",
    "members",
    "marketplace",
    "gaming",
    "go to channel",
    "notifications",
    "messages",
    "messenger",
    "grok actions",
    "settings",
    "privacy",
    "search",
    "search within this group",
    "menu",
    "more",
    "more actions",
    "see more",
    "see less",
    "show more",
    "show less",
    "like",
    "likes",
    "comment",
    "comments",
    "share",
    "shares",
    "reply",
    "replies",
    "follow",
    "following",
    "subscribe",
    "subscribed",
    "join",
    "joined",
    "save",
    "saved",
    "copy link",
    "share post",
    "view group",
    "view profile",
    "view page",
    "view channel",
    "view community",
    "embed",
    "report",
    "block",
    "hide",
    "mute",
    "turn on notifications",
    "turn off notifications",
    "open control menu",
    "open control menu for post",
    "open user actions",
    "actions available for this post",
    "send",
    "post",
    "photo",
    "video",
    "embedded video",
    "video player",
    "video progress",
    "watch in full screen",
    "tap to watch live",
    "adjust volume",
    "go to previous page of document",
    "go to next page of document",
    "slide to navigate to a page",
    "add to favorites",
    "unlock premium tools and grow your career",
    "want to view more unlock the full document below",
    "live",
    "story",
    "stories",
    "open",
    "close",
    "back",
    "next",
    "previous",
    "skip",
    "start now",
    "not now",
    "accept all",
    "decline optional cookies",
    "log in",
    "login",
    "sign in",
    "sign up",
    "create account",
    "create a post",
    "create story",
    "cover photo",
    "see everyone",
    "see recommended groups",
    "about featured section",
    "expand/collapse featured section",
    "write something",
    "write something...",
    "anonymous post",
    "live video",
    "stories tray",
    "next items",
    "feed posts",
    "shorts",
    "forgot password",
    "password",
    "email address",
    "phone number",
    "what's on your mind",
    "actions for this post",
    "hide post by",
    "send this to friends or post it on your profile",
    "see translation",
    "view translation",
    "translation preferences",
    "see original",
    "rate this translation",
    "open reel in reels viewer",
    "play video",
    "change position",
    "enter fullscreen",
    "change volume",
    "write a comment",
    "leave a comment",
    "add a comment",
    "comment as",
    "view more comments",
    "view previous comments",
    "see who reacted to this",
    "most relevant",
    "all comments",
    "newest",
    "oldest",
    "sort by",
    "sort group feed by",
    "featured",
    "featured content badge",
    "discussion",
    "people",
    "events",
    "media",
    "files",
    "joined",
    "invite",
    "sponsored",
    "promoted",
    "ad",
    "advertisement",
    "trang chủ",
    "bạn bè",
    "nhóm",
    "thông báo",
    "tin nhắn",
    "tìm kiếm",
    "xem thêm",
    "ẩn bớt",
    "thích",
    "bình luận",
    "chia sẻ",
    "trả lời",
    "theo dõi",
    "đang theo dõi",
    "tham gia",
    "đã tham gia",
    "lưu",
    "đã lưu",
    "viết bình luận",
    "xem thêm bình luận",
    "phù hợp nhất",
    "đăng nhập",
    "đăng ký",
    "được tài trợ",
  ];

  const UI_SET = new Set(UI_TEXT.map(normalizeKey));

  const COMMENT_BOUNDARY = [
    "write a comment",
    "add a comment",
    "comment as",
    "view more comments",
    "view previous comments",
    "all comments",
    "comment by",
    "most relevant",
    "be the first to comment",
    "viết bình luận",
    "xem thêm bình luận",
    "phù hợp nhất",
  ].map(normalizeKey);

  const METRIC_WORDS = {
    views: [
      "view",
      "views",
      "viewed",
      "watching",
      "lượt xem",
      "luot xem",
      "visualización",
      "visualizaciones",
      "vistas",
      "vues",
      "visualizações",
      "조회",
      "再生",
      "播放",
      "просмотров",
    ],
    likes: [
      "like",
      "likes",
      "liked",
      "reaction",
      "reactions",
      "thích",
      "thich",
      "lượt thích",
      "luot thich",
      "me gusta",
      "j’aime",
      "jaime",
      "curtidas",
      "좋아요",
    ],
    shares: [
      "share",
      "shares",
      "shared",
      "chia sẻ",
      "chia se",
      "lượt chia sẻ",
      "luot chia se",
      "partages",
      "compartidos",
      "compartilhamentos",
    ],
    comments: [
      "comment",
      "comments",
      "commented",
      "reply",
      "replies",
      "bình luận",
      "binh luan",
      "lượt bình luận",
      "luot binh luan",
      "comentarios",
      "commentaires",
      "kommentare",
      "댓글",
    ],
    reposts: [
      "repost",
      "reposts",
      "retweet",
      "retweets",
      "quote",
      "quotes",
      "đăng lại",
      "dang lai",
      "reposts",
    ],
    saves: [
      "save",
      "saves",
      "saved",
      "favorite",
      "favorites",
      "added to favorites",
      "bookmark",
      "bookmarks",
      "lưu",
      "luu",
      "đã lưu",
      "da luu",
    ],
  };

  const METRIC_LABEL_TO_KEY = buildMetricLabelMap(METRIC_WORDS);
  const TIME_OR_DATE_RE = new RegExp(
    [
      "^\\d+\\s*(s|sec|second|seconds|min|mins|minute|minutes|h|hr|hrs|hour|hours|d|day|days|w|week|weeks|mo|month|months|y|yr|year|years)$",
      "^\\d+\\s*(s|sec|second|seconds|min|mins|minute|minutes|h|hr|hrs|hour|hours|d|day|days|w|week|weeks|mo|month|months|y|yr|year|years)\\s+ago$",
      "^\\d+\\s*(sec|min|hr|hrs?)\\.\\s+ago$",
      "^\\d+\\s*(giây|phút|gio|giờ|ngày|tuần|tháng|năm)$",
      "^(just now|yesterday|today|vừa xong|hôm qua|hôm nay)$",
      "^[A-Z][a-z]{2}\\s+\\d{1,2}(,\\s*\\d{4})?$",
      "^\\d{1,2}[:.]\\d{2}(\\s*(AM|PM))?$",
      "^\\d{1,2}[/-]\\d{1,2}([/-]\\d{2,4})?$",
    ].join("|"),
    "i",
  );
  const FULL_DATE_TIME_RE = /^(monday|tuesday|wednesday|thursday|friday|saturday|sunday),\s+[a-z]+\s+\d{1,2},\s+\d{4}\s+at\s+\d{1,2}:\d{2}(?::\d{2})?\s*(am|pm)?(?:\s+[a-z]{2,4})?$/i;

  const SOCIAL_HOST_PARTS = [
    "facebook.",
    "fb.watch",
    "twitter.",
    "x.com",
    "linkedin.",
    "youtube.",
    "youtu.be",
    "tiktok.",
    "instagram.",
    "threads.",
    "reddit.",
  ];

  function filterSocialHtml(input, options) {
    const opts = Object.assign(
      {
        currentUrl: "",
        keepRawText: false,
        maxPosts: MAX_POSTS,
        maxItems: 160,
        outputMode: "text",
      },
      options || {},
    );

    const parsed = parseInput(input, opts);
    const root = parsed.root;
    const baseUrl = opts.currentUrl || parsed.baseUrl || "";
    const textMemo = new WeakMap();
    const linkMemo = new WeakMap();

    const allLinks = collectLinks(root, baseUrl, linkMemo);
    const seedElements = collectSeedElements(root, allLinks);
    if (isPostUrl(baseUrl)) seedElements.push(root);
    const listLikePage = isEntityListPage(baseUrl);
    const candidateNodes = choosePostContainers(root, seedElements, textMemo, linkMemo, baseUrl);
    const posts = [];
    const seenPostKeys = new Set();

    for (const node of candidateNodes) {
      const post = extractPostFromContainer(node, {
        baseUrl,
        textMemo,
        linkMemo,
      });

      if (listLikePage && !post.postUrl && !Object.keys(post.stats || {}).length) continue;
      if (!isUsefulPost(post)) continue;

      const key = post.postUrl
        ? `url:${canonicalizeUrlForDedupe(post.postUrl)}`
        : `text:${fingerprint([post.accountName, post.content].filter(Boolean).join(" "))}`;

      if (seenPostKeys.has(key)) {
        continue;
      }
      seenPostKeys.add(key);
      posts.push(post);
      if (posts.length >= opts.maxPosts) break;
    }

    const dedupedPosts = dedupeNearDuplicatePosts(posts).slice(0, opts.maxPosts);
    const entityItems = collectEntityItems(root, allLinks, {
      baseUrl,
      textMemo,
      linkMemo,
      maxItems: opts.maxItems,
    });
    const displayedItems = shouldDisplayEntityItems(baseUrl, entityItems, dedupedPosts)
      ? entityItems.slice(0, opts.maxItems)
      : [];
    const accounts = uniqueAccounts(dedupedPosts);
    const postLinks = uniqueLinks(dedupedPosts.map((post) => post.postUrl).filter(Boolean));
    const displayBaseUrl = opts.outputBaseUrl || baseUrl;
    const meaningfulText = buildMeaningfulText(dedupedPosts, displayedItems, displayBaseUrl);
    const text = formatResultAsText({ posts: dedupedPosts, items: displayedItems }, { baseUrl: displayBaseUrl });

    const result = {
      version: VERSION,
      platformHint: detectPlatformHint(baseUrl || allLinks[0]?.url || ""),
      posts: dedupedPosts,
      items: entityItems,
      accounts,
      links: {
        accounts: accounts.map((account) => account.url).filter(Boolean),
        posts: postLinks,
        items: uniqueLinks(entityItems.map((item) => item.url).filter(Boolean)),
      },
      meaningfulText,
      text,
      compactText: text,
      meta: {
        inputMode: parsed.mode,
        seedCount: seedElements.length,
        candidateCount: candidateNodes.length,
        postCount: dedupedPosts.length,
        itemCount: entityItems.length,
        displayedItemCount: displayedItems.length,
      },
    };

    if (opts.keepRawText) {
      result.rawMeaningfulLines = uniqueBy(
        collectTextRecords(root, textMemo).map((record) => record.text),
        normalizeKey,
      ).filter((line) => isMeaningfulLine(line, null));
    }

    return result;
  }

  function filterCurrentPage(options) {
    if (typeof document === "undefined") {
      throw new Error("filterCurrentPage() only works in a browser-like environment.");
    }
    const opts = Object.assign(
      {
        currentUrl: typeof location !== "undefined" ? location.href : document.baseURI,
      },
      options || {},
    );
    return filterSocialHtml(document, opts);
  }

  function filterSocialHtmlToText(input, options) {
    return filterSocialHtml(input, options).text;
  }

  function filterCurrentPageToText(options) {
    return filterCurrentPage(options).text;
  }

  function serializeCurrentPageHtml(options) {
    if (typeof document === "undefined") {
      throw new Error("serializeCurrentPageHtml() only works in a browser-like environment.");
    }
    return serializeDomForFiltering(document.body || document.documentElement, options);
  }

  function serializeDomForFiltering(root, options) {
    const opts = Object.assign(
      {
        maxNodes: MAX_WALK_NODES,
        keepAttrs: ["href", "aria-label", "title", "alt", "role", "data-testid", "data-test-id", "data-e2e", "id", "class"],
      },
      options || {},
    );
    let count = 0;

    function serialize(node) {
      if (!node || count >= opts.maxNodes) return "";

      if (isTextNode(node)) {
        return escapeHtml(cleanText(textOf(node)));
      }

      if (!isElementNode(node)) {
        return childNodes(node).map(serialize).join("");
      }

      if (shouldSkipElement(node)) return "";
      count += 1;

      const tag = tagName(node);
      const attrs = opts.keepAttrs
        .map((name) => {
          const value = getAttr(node, name);
          return value ? ` ${name}="${escapeHtml(value)}"` : "";
        })
        .join("");
      const children = childNodes(node).map(serialize).join("");
      return `<${tag}${attrs}>${children}</${tag}>`;
    }

    return serialize(root);
  }

  function formatResultAsText(result, options) {
    const opts = options || {};
    const posts = result.posts || [];
    const items = result.items || [];
    const sections = [];
    const postText = posts
      .map((post, index) => {
        const lines = [`Post ${index + 1}`];
        if (post.accountName) lines.push(`Account: ${post.accountName}`);
        if (post.accountUrl) lines.push(`Account URL: ${normalizeOutputUrl(post.accountUrl, opts.baseUrl)}`);
        if (post.postUrl) lines.push(`Post URL: ${normalizeOutputUrl(post.postUrl, opts.baseUrl)}`);
        if (post.content) lines.push(`Content: ${post.content}`);

        for (const key of ["views", "likes", "comments", "shares", "reposts", "saves"]) {
          const metric = post.stats && post.stats[key];
          if (metric && metric.raw) lines.push(`${capitalize(key)}: ${formatMetricForText(metric)}`);
        }

        return lines.join("\n");
      })
      .join("\n\n")
      .trim();

    if (postText) sections.push(postText);

    const itemText = items
      .map((item, index) => {
        const lines = [`Item ${index + 1}`];
        if (item.type) lines.push(`Type: ${item.type}`);
        if (item.name) lines.push(`Name: ${item.name}`);
        if (item.url) lines.push(`URL: ${normalizeOutputUrl(item.url, opts.baseUrl)}`);
        for (const detail of item.details || []) lines.push(`Detail: ${detail}`);
        for (const action of item.actions || []) lines.push(`Action: ${action}`);
        return lines.join("\n");
      })
      .join("\n\n")
      .trim();

    if (itemText) sections.push(itemText);

    return sections.join("\n\n").trim();
  }

  function parseInput(input, opts) {
    if (input && isElementNode(input)) {
      const doc = input.ownerDocument || null;
      return {
        root: input.body || input.documentElement || input,
        baseUrl: opts.currentUrl || doc?.baseURI || "",
        mode: "dom",
      };
    }

    if (input && input.nodeType === 9) {
      return {
        root: input.body || input.documentElement,
        baseUrl: opts.currentUrl || input.baseURI || "",
        mode: "document",
      };
    }

    const html = String(input || "");
    if (typeof DOMParser !== "undefined") {
      const doc = new DOMParser().parseFromString(html, "text/html");
      return {
        root: doc.body || doc.documentElement,
        baseUrl: opts.currentUrl || doc.baseURI || "",
        mode: "domparser",
      };
    }

    return {
      root: parseHtmlLite(html),
      baseUrl: opts.currentUrl || "",
      mode: "lite",
    };
  }

  function parseHtmlLite(html) {
    html = stripRawNonContentBlocks(html);
    const root = makeLiteElement("body", {});
    const stack = [root];
    const tokenRe = /<!--[\s\S]*?-->|<![^>]*>|<\/([a-zA-Z][\w:-]*)\s*>|<([a-zA-Z][\w:-]*)([^>]*)>|([^<]+)/g;
    const voidTags = new Set(["area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "source", "track", "wbr"]);
    let match;

    while ((match = tokenRe.exec(html))) {
      if (match[1]) {
        const closing = match[1].toLowerCase();
        for (let index = stack.length - 1; index > 0; index -= 1) {
          if (stack[index].tagName === closing) {
            stack.length = index;
            break;
          }
        }
        continue;
      }

      if (match[2]) {
        const tagName = match[2].toLowerCase();
        const attrSource = match[3] || "";
        const attrs = parseAttrsLite(attrSource);
        const node = makeLiteElement(tagName, attrs);
        appendLiteNode(stack[stack.length - 1], node);
        if (!voidTags.has(tagName) && !/\/\s*$/.test(attrSource)) {
          stack.push(node);
        }
        continue;
      }

      if (match[4]) {
        const text = decodeEntities(match[4]);
        if (cleanText(text)) {
          appendLiteNode(stack[stack.length - 1], {
            nodeType: 3,
            textContent: text,
            parentNode: null,
          });
        }
      }
    }

    return root;
  }

  function stripRawNonContentBlocks(html) {
    return String(html || "").replace(
      /<(script|style|noscript|template|canvas|iframe|head)\b[^>]*>[\s\S]*?<\/\1\s*>/gi,
      " ",
    );
  }

  function makeLiteElement(tagName, attrs) {
    return {
      nodeType: 1,
      tagName,
      attributesMap: attrs,
      childNodes: [],
      parentNode: null,
    };
  }

  function appendLiteNode(parent, child) {
    child.parentNode = parent;
    parent.childNodes.push(child);
  }

  function parseAttrsLite(source) {
    const attrs = {};
    const attrRe = /([^\s"'<>/=]+)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'=<>`]+)))?/g;
    let match;
    while ((match = attrRe.exec(source))) {
      const name = match[1].toLowerCase();
      const value = match[2] ?? match[3] ?? match[4] ?? "";
      attrs[name] = decodeEntities(value);
    }
    return attrs;
  }

  function collectSeedElements(root, allLinks) {
    const seeds = new Set();

    for (const link of allLinks) {
      if (link.kind === "post") {
        seeds.add(link.element);
      }
    }

    walkElements(root, (node) => {
      if (seeds.size > 2000) return false;
      if (isSuppressedChromeContainer(node)) return false;
      if (hasSuppressedChromeAncestor(node)) return true;
      const tag = tagName(node);
      const role = normalizeKey(getAttr(node, "role"));
      const attrBlob = [
        getAttr(node, "data-testid"),
        getAttr(node, "data-test-id"),
        getAttr(node, "data-e2e"),
        getAttr(node, "aria-label"),
        getAttr(node, "class"),
        getAttr(node, "id"),
      ]
        .join(" ")
        .toLowerCase();

      if (
        tag === "article" ||
        role === "article" ||
        /\b(post|tweet|status|feed-shared-update|update-components|video-card|reel|shorts|recommend-list-item-container|feed-video)\b/.test(attrBlob)
      ) {
        seeds.add(node);
      }
      return true;
    });

    if (seeds.size === 0) {
      for (const link of allLinks) {
        if (link.kind === "account" && isNameLike(link.text)) {
          seeds.add(link.element);
          if (seeds.size >= 200) break;
        }
      }
    }

    return Array.from(seeds);
  }

  function choosePostContainers(root, seeds, textMemo, linkMemo, baseUrl) {
    const candidates = [];
    const seen = new Set();
    let order = 0;

    for (const seed of seeds) {
      let node = isElementNode(seed) ? seed : parentElement(seed);
      let climb = 0;
      let best = null;

      if (node === root) {
        const score = scoreContainer(node, textMemo, linkMemo, baseUrl);
        best = { node, score };
      }

      while (node && node !== root && climb < 10) {
        if (!isElementNode(node) || shouldSkipElement(node)) {
          node = parentElement(node);
          climb += 1;
          continue;
        }

        const score = scoreContainer(node, textMemo, linkMemo, baseUrl);
        if (!best || candidateRankScore(score) > candidateRankScore(best.score)) {
          best = { node, score };
        }

        const role = normalizeKey(getAttr(node, "role"));
        const articleLike = tagName(node) === "article" || role === "article";
        const compactPostCard =
          score.postLinks > 0 &&
          score.textChars >= 20 &&
          score.textChars <= 1200 &&
          score.linkCount <= 16 &&
          score.contentLineCount > 0 &&
          (score.metricHits > 0 || score.accountLinks > 0);
        if (
          (articleLike || compactPostCard) &&
          score.textChars >= 40 &&
          score.textChars <= 5000 &&
          (score.postLinks > 0 || score.metricHits > 0) &&
          !/^comment by\b/i.test(cleanText(getAttr(node, "aria-label")))
        ) {
          break;
        }

        if (score.textChars > 3500 && score.postLinks > 0 && score.accountLinks > 0) {
          break;
        }

        node = parentElement(node);
        climb += 1;
      }

      if (!best) continue;
      const candidate = best.node;
      if (seen.has(candidate)) continue;
      seen.add(candidate);
      candidates.push({
        node: candidate,
        score: best.score.total,
        rank: candidateRankScore(best.score),
        textChars: best.score.textChars,
        linkCount: best.score.linkCount,
        postLinks: best.score.postLinks,
        distinctPostLinks: best.score.distinctPostLinks,
        metricHits: best.score.metricHits,
        accountLinks: best.score.accountLinks,
        order: order++,
      });
    }

    const filtered = candidates
      .filter((candidate) => candidate.score >= 20)
      .sort((a, b) => {
        if (b.rank !== a.rank) return b.rank - a.rank;
        return a.textChars - b.textChars;
      });

    const selected = [];
    for (const candidate of filtered) {
      let duplicate = false;
      let replaceIndex = -1;

      for (let index = 0; index < selected.length; index += 1) {
        const selectedCandidate = selected[index];
        if (candidate.node === selectedCandidate.node) {
          duplicate = true;
          break;
        }
        const aContainsB = nodeContains(candidate.node, selectedCandidate.node);
        const bContainsA = nodeContains(selectedCandidate.node, candidate.node);
        if (!aContainsB && !bContainsA) continue;

        const aText = fingerprint(collectTextRecords(candidate.node, textMemo).map((record) => record.text).join(" "));
        const bText = fingerprint(collectTextRecords(selectedCandidate.node, textMemo).map((record) => record.text).join(" "));
        const sameContent = aText && bText && (aText === bText || aText.includes(bText) || bText.includes(aText));
        if (!sameContent) continue;

        duplicate = true;
        if (bContainsA && isMoreSpecificPostCandidate(candidate, selectedCandidate)) {
          replaceIndex = index;
        } else if (aContainsB && isBetterAncestorPostCandidate(candidate, selectedCandidate)) {
          replaceIndex = index;
        }
        break;
      }

      if (replaceIndex >= 0) {
        selected[replaceIndex] = candidate;
      } else if (!duplicate) {
        selected.push(candidate);
      }
      if (selected.length >= MAX_POSTS * 2) break;
    }

    return selected
      .sort((a, b) => a.order - b.order)
      .map((candidate) => candidate.node);
  }

  function isMoreSpecificPostCandidate(candidate, selectedCandidate) {
    if (!candidate || !selectedCandidate) return false;
    if (candidate.postLinks <= 0 || candidate.postLinks > 3) return false;
    if (selectedCandidate.distinctPostLinks <= candidate.distinctPostLinks) return false;
    if (candidate.textChars > 1400) return false;
    if (candidate.textChars >= selectedCandidate.textChars * 0.75) return false;
    if (selectedCandidate.postLinks <= candidate.postLinks && selectedCandidate.textChars < 1800) return false;
    return candidate.metricHits > 0 || candidate.accountLinks > 0 || candidate.score >= selectedCandidate.score * 0.45;
  }

  function isBetterAncestorPostCandidate(candidate, selectedCandidate) {
    if (!candidate || !selectedCandidate) return false;
    if (candidate.distinctPostLinks !== selectedCandidate.distinctPostLinks) return false;
    if (candidate.distinctPostLinks !== 1) return false;
    if (candidate.order >= selectedCandidate.order) return false;
    if (candidate.textChars > 5000) return false;
    if (candidate.textChars <= selectedCandidate.textChars) return false;
    return candidate.score >= selectedCandidate.score * 0.5 || candidate.metricHits > selectedCandidate.metricHits;
  }

  function scoreContainer(node, textMemo, linkMemo, baseUrl) {
    const records = collectTextRecords(node, textMemo);
    const links = collectLinks(node, baseUrl, linkMemo);
    const text = records.map((record) => record.text).join(" ");
    const textChars = text.length;
    const postLinks = links.filter((link) => link.kind === "post").length;
    const distinctPostLinks = uniqueLinks(links.filter((link) => link.kind === "post").map((link) => link.url)).length;
    const accountLinks = links.filter((link) => link.kind === "account" && isNameLike(link.text)).length;
    const linkCount = links.length;
    const metricHits = extractStatsFromRecords(records).hitCount;
    const contentLines = records.filter((record) => isMeaningfulLine(record.text, null));
    const tag = tagName(node);
    const role = normalizeKey(getAttr(node, "role"));
    const attrBlob = [
      getAttr(node, "data-testid"),
      getAttr(node, "data-test-id"),
      getAttr(node, "aria-label"),
      getAttr(node, "class"),
      getAttr(node, "id"),
    ]
      .join(" ")
      .toLowerCase();

    let total = 0;
    total += Math.min(postLinks, 3) * 26;
    total += Math.min(accountLinks, 3) * 14;
    total += Math.min(metricHits, 6) * 9;
    total += Math.min(contentLines.length, 8) * 5;
    total += Math.min(Math.floor(textChars / 120), 12);

    if (tag === "article" || role === "article") total += 22;
    if (/\b(post|tweet|status|feed-shared-update|video-card|reel|shorts)\b/.test(attrBlob)) total += 16;
    if (/\b(recommend-list-item-container|feed-video)\b/.test(attrBlob)) total += 18;
    if (/^comment by\b/i.test(cleanText(getAttr(node, "aria-label")))) total -= 90;
    if (textChars < 35) total -= 25;
    if (textChars > 6000) total -= 30;
    if (textChars > 12000) total -= 70;
    if (linkCount > 60) total -= 20;
    if (linkCount > 160) total -= 60;
    if (postLinks === 0 && accountLinks > 1 && metricHits > 3) total -= (accountLinks - 1) * 45;
    total -= repeatedTokenPenalty(text);
    if (NOISE_ROLES.has(role)) total -= 80;

    return {
      total,
      textChars,
      linkCount,
      postLinks,
      distinctPostLinks,
      accountLinks,
      metricHits,
      contentLineCount: contentLines.length,
    };
  }

  function candidateRankScore(score) {
    return (
      score.total -
      Math.max(0, score.textChars - 2500) / 35 -
      Math.max(0, score.linkCount - 30) * 4
    );
  }

  function extractPostFromContainer(container, context) {
    let records = collectTextRecords(container, context.textMemo);
    let links = collectLinks(container, context.baseUrl, context.linkMemo);
    let postUrl = selectPostUrl(links, context.baseUrl);
    const distinctPostCount = uniqueLinks(links.filter((link) => link.kind === "post").map((link) => link.url)).length;
    const focusedNode = distinctPostCount > 1 ? findFocusedPostNode(container, postUrl, context) : null;

    if (focusedNode && focusedNode !== container) {
      records = collectTextRecords(focusedNode, context.textMemo);
      links = collectLinks(focusedNode, context.baseUrl, context.linkMemo);
      postUrl = selectPostUrl(links, context.baseUrl) || postUrl;
    }

    const stats = extractStatsFromRecords(records).stats;
    const account = refineAccountFromPostUrl(selectAccount(links, records), postUrl, records);
    const content = selectContent(records, links, account.name, stats);
    const platformHint = detectPlatformHint(postUrl || account.url || context.baseUrl);
    const confidence = computePostConfidence({
      account,
      postUrl,
      content,
      stats,
      links,
      records,
    });

    return pruneEmpty({
      platformHint,
      accountName: account.name || "",
      accountUrl: account.url || "",
      postUrl: postUrl || "",
      content,
      stats,
      confidence,
    });
  }

  function findFocusedPostNode(container, postUrl, context) {
    if (!postUrl) return null;
    const targetKey = canonicalizeUrlForDedupe(postUrl);
    if (!targetKey) return null;

    const links = collectLinks(container, context.baseUrl, context.linkMemo);
    const matchingLinks = links.filter((link) => {
      return link.kind === "post" && canonicalizeUrlForDedupe(link.url) === targetKey;
    });

    let best = null;
    for (const link of matchingLinks) {
      let node = parentElement(link.element);
      let climb = 0;

      while (node && node !== container && climb < 40) {
        if (isElementNode(node) && !shouldSkipElement(node)) {
          if (isCommentLikeNode(node, context.textMemo)) {
            node = parentElement(node);
            climb += 1;
            continue;
          }

          const nodeLinks = collectLinks(node, context.baseUrl, context.linkMemo);
          const distinctPosts = uniqueLinks(nodeLinks.filter((candidate) => candidate.kind === "post").map((candidate) => candidate.url)).length;
          const score = scoreContainer(node, context.textMemo, context.linkMemo, context.baseUrl);
          const compactEnough =
            distinctPosts <= 1 &&
            score.postLinks > 0 &&
            score.textChars >= 20 &&
            score.textChars <= 5000 &&
            score.linkCount <= 80 &&
            score.contentLineCount > 0 &&
            (score.metricHits > 0 || score.accountLinks > 0);

          if (compactEnough) {
            const rank = candidateRankScore(score);
            if (
              !best ||
              score.metricHits > best.metricHits ||
              (score.metricHits === best.metricHits && (rank > best.rank || (rank === best.rank && score.textChars > best.textChars)))
            ) {
              best = {
                node,
                rank,
                textChars: score.textChars,
                metricHits: score.metricHits,
              };
            }
          }
        }

        node = parentElement(node);
        climb += 1;
      }
    }

    return best?.node || null;
  }

  function isCommentLikeNode(node, textMemo) {
    if (/^comment by\b/i.test(cleanText(getAttr(node, "aria-label")))) return true;
    const records = collectTextRecords(node, textMemo);
    return records.some((record) => /^comment by\b/i.test(cleanText(record.text)));
  }

  function selectAccount(links, records) {
    const contentStartOrder = findLikelyContentStartOrder(records);

    const linkCandidates = links
      .filter((link) => link.kind === "account")
      .map((link, index) => {
        const text = cleanAccountName(link.text || link.ariaLabel || link.title || "");
        const key = normalizeKey(text);
        const order = findRecordOrderForText(records, text, index);
        let score = 0;
        if (isNameLike(text)) score += 40;
        if (link.kind === "account") score += 30;
        if (/@[a-z0-9._-]{2,}/i.test(text) || /\/@/.test(link.url)) score += 12;
        if (Number.isFinite(contentStartOrder) && order <= contentStartOrder) {
          const distance = contentStartOrder - order;
          score += Math.max(0, 100 - distance) * 0.25;
          score += Math.min(18, order * 0.4);
        }
        if (isMetricLike(text) || isUiText(text)) score -= 60;
        if (isAccountNoiseText(text)) score -= 80;
        if (text.length > 80) score -= 35;
        if (isPostUrl(link.url)) score -= 40;
        score -= Math.min(index, 10);
        return {
          name: text,
          url: link.url,
          score,
          order,
          link,
        };
      })
      .filter((candidate) => candidate.score > 15);

    const postLinkTextKeys = new Set(
      links
        .filter((link) => link.kind === "post")
        .map((link) => normalizeKey(link.text))
        .filter(Boolean),
    );
    const recordCandidates = collectRecordAccountCandidates(records, contentStartOrder, linkCandidates, postLinkTextKeys);
    const candidates = linkCandidates
      .concat(recordCandidates)
      .sort((a, b) => b.score - a.score);

    if (candidates[0]) {
      return {
        name: candidates[0].name,
        url: candidates[0].url || "",
      };
    }

    const fallbackName = records
      .map((record) => record.text)
      .find((text) => isNameLike(text) && !isMetricLike(text) && !isUiText(text));

    return {
      name: fallbackName || "",
      url: "",
    };
  }

  function refineAccountFromPostUrl(account, postUrl, records) {
    const parsed = safeUrl(postUrl || "");
    if (!parsed) return account;
    const host = stripWww(parsed.hostname).toLowerCase();
    const match = parsed.pathname.match(/^\/([A-Za-z0-9_]{1,20})\/status\/\d+/);
    if (!(host === "x.com" || host.includes("twitter.")) || !match) return account;

    const handle = match[1];
    const currentName = cleanText(account?.name || "");
    const currentLooksBad = !currentName || TIME_OR_DATE_RE.test(currentName) || isUiText(currentName);
    const inferred = inferXAccountName(records, handle);

    if (inferred && inferred !== `@${handle}` && normalizeKey(currentName) === normalizeKey(`@${handle}`)) {
      return {
        name: inferred,
        url: account?.url || `https://${parsed.hostname}/${handle}`,
      };
    }

    if (currentLooksBad || !account?.url) {
      return {
        name: inferred || `@${handle}`,
        url: `https://${parsed.hostname}/${handle}`,
      };
    }

    return account;
  }

  function inferXAccountName(records, handle) {
    const handleKey = normalizeKey(`@${handle}`);
    for (let index = 0; index < records.length; index += 1) {
      const text = cleanText(records[index].text);
      const key = normalizeKey(text);
      if (key !== handleKey && key !== normalizeKey(handle)) continue;

      for (let prev = index - 1; prev >= Math.max(0, index - 4); prev -= 1) {
        const candidate = cleanText(records[prev].text);
        if (!candidate || isUiText(candidate) || TIME_OR_DATE_RE.test(candidate) || isMetricLike(candidate)) continue;
        if (isGeneratedAltText(candidate) || /\b(profile picture|avatar)\b/i.test(candidate)) continue;
        if (candidate.startsWith("@")) continue;
        if (isNameLike(candidate)) return `${candidate} @${handle}`;
      }
      return `@${handle}`;
    }
    return `@${handle}`;
  }

  function collectRecordAccountCandidates(records, contentStartOrder, linkCandidates, postLinkTextKeys) {
    if (!Number.isFinite(contentStartOrder)) return [];
    const candidates = [];
    const start = Math.max(0, contentStartOrder - 140);
    const matchingLinks = linkCandidates || [];

    for (const record of records) {
      if (record.order < start || record.order >= contentStartOrder) continue;
      const raw = cleanText(record.text);
      if (!raw || isUiText(raw) || isMetricLike(raw) || isGeneratedAltText(raw)) continue;
      if (postLinkTextKeys && postLinkTextKeys.has(normalizeKey(raw))) continue;

      const byline = raw.match(/^(?:reel|post|video|photo|short|tweet)\s+by\s+(.+)$/i);
      const name = cleanAccountName(byline ? byline[1] : raw);
      if (!isLikelyTextAuthorName(name) || isAccountNoiseText(raw) || isAccountNoiseText(name)) continue;

      const distance = contentStartOrder - record.order;
      let score = byline ? 58 : 60;
      score += Math.max(0, 90 - distance) * 0.35;
      score += Math.min(20, record.order * 0.2);
      if (distance <= 12) score += 18;
      if (/^(facebook|active|stories?|video player)$/i.test(name)) score -= 80;

      const matchingLink = matchingLinks.find((candidate) => {
        const a = normalizeKey(candidate.name);
        const b = normalizeKey(name);
        return a === b || a.includes(b) || b.includes(a);
      });

      candidates.push({
        name,
        url: matchingLink?.url || "",
        score,
        order: record.order,
      });
    }

    return dedupeRecordAccountCandidates(candidates);
  }

  function dedupeRecordAccountCandidates(candidates) {
    const bestByName = new Map();
    for (const candidate of candidates) {
      const key = normalizeKey(candidate.name);
      const existing = bestByName.get(key);
      if (!existing || candidate.score > existing.score) bestByName.set(key, candidate);
    }
    return Array.from(bestByName.values());
  }

  function findLikelyContentStartOrder(records) {
    for (const record of records) {
      const text = cleanText(record.text);
      if (!isMeaningfulLine(text, null)) continue;
      const score = scoreContentLine(text);
      if (isNameLike(text) && text.length < 80 && score < 18) continue;
      if (score >= 18) return record.order;
    }
    return Number.POSITIVE_INFINITY;
  }

  function findRecordOrderForText(records, text, fallback) {
    const key = normalizeKey(text);
    if (!key) return fallback;
    for (const record of records) {
      const recordKey = normalizeKey(record.text);
      if (!recordKey) continue;
      if (
        recordKey === key ||
        recordKey.startsWith(`${key} `) ||
        key.startsWith(`${recordKey} `)
      ) {
        return record.order;
      }
    }
    return fallback;
  }

  function cleanAccountName(text) {
    return cleanText(text)
      .replace(/^go to channel\s+/i, "")
      .replace(/\s*,?\s*view story$/i, "")
      .replace(/\s*'s\s+(timeline|story)$/i, "")
      .trim();
  }

  function isAccountNoiseText(text) {
    const key = normalizeKey(text);
    return (
      /\b(timeline|story|stories tray)\b/.test(key) ||
      /^(discussion|about|featured|people|events|media|files|more|more actions|start now|shorts|mix|group by|public group|joined|invite)$/.test(key) ||
      /\b(followers|connections)$/.test(key) ||
      /^search within this group\b/.test(key) ||
      /\bprofile$/.test(key) ||
      /^go to channel\b/.test(key) ||
      /^hide post by\b/.test(key) ||
      /^create story\b/.test(key)
    );
  }

  function isLikelyTextAuthorName(text) {
    const cleaned = cleanText(text);
    if (!isNameLike(cleaned)) return false;
    if (/^@?[a-z0-9._-]{2,30}$/i.test(cleaned)) return true;
    if (/[’']/.test(cleaned)) return false;
    if (/[.!?。！？…]/.test(cleaned)) return false;

    const words = cleaned.split(/\s+/).filter(Boolean);
    const lowerStopwords = new Set([
      "a",
      "an",
      "and",
      "or",
      "the",
      "with",
      "without",
      "have",
      "has",
      "had",
      "more",
      "less",
      "officially",
      "changed",
      "career",
      "nails",
      "eyelashes",
    ]);

    if (words.some((word) => lowerStopwords.has(normalizeKey(word)))) return false;
    return true;
  }

  function selectPostUrl(links, baseUrl) {
    const candidates = links
      .filter((link) => link.kind === "post")
      .map((link, index) => {
        let score = 60;
        const url = link.url;
        const path = safeUrl(url)?.pathname.toLowerCase() || "";
        if (/\/status\/\d+|\/posts\/|\/feed\/update|\/watch|\/shorts\/|\/video\/|\/reel\/|\/p\//.test(path)) score += 20;
        if (link.text && TIME_OR_DATE_RE.test(cleanText(link.text))) score += 18;
        if (isUiText(link.text)) score -= 10;
        score -= index;
        return { url, score };
      })
      .sort((a, b) => b.score - a.score);

    if (candidates[0]) return candidates[0].url;
    return isPostUrl(baseUrl) ? baseUrl : "";
  }

  function selectContent(records, links, accountName, stats) {
    const linkTextKeys = new Set(
      links
        .filter((link) => link.kind === "account")
        .map((link) => normalizeKey(link.text))
        .filter(Boolean),
    );
    const nonContentLinkTextKeys = new Set(
      links
        .filter((link) => isNonContentLink(link))
        .map((link) => normalizeKey(link.text || link.ariaLabel || link.title))
        .filter(Boolean),
    );
    const postContentLinkTextKeys = new Set(
      links
        .filter((link) => link.kind === "post")
        .map((link) => normalizeKey(link.text || link.ariaLabel || link.title))
        .filter(Boolean),
    );
    const statKeys = new Set();
    for (const metric of Object.values(stats || {})) {
      if (metric?.raw) statKeys.add(normalizeKey(metric.raw));
    }

    const accountKey = normalizeKey(accountName || "");
    const selected = [];
    const seen = new Set();
    let sawContent = false;

    for (const record of records) {
      const text = cleanText(record.text);
      if (!text) continue;

      const key = normalizeKey(text);
      if (!key || seen.has(key)) continue;

      if (sawContent && COMMENT_BOUNDARY.some((boundary) => key.includes(boundary))) {
        break;
      }

      if (accountKey && key === accountKey) continue;
      if (statKeys.has(key)) continue;
      if (nonContentLinkTextKeys.has(key)) continue;
      if (linkTextKeys.has(key) && text.length < 80) continue;
      if (!isMeaningfulLine(text, accountName)) continue;

      const score = scoreContentLine(text);
      if (
        score < 10 &&
        !postContentLinkTextKeys.has(key) &&
        !((/[#@]/.test(text) || /[’']/.test(text)) && score >= 6)
      ) continue;

      const contentLine = cleanContentLine(text);
      if (!contentLine || isUiText(contentLine)) continue;
      selected.push(contentLine);
      seen.add(key);
      sawContent = true;

      const currentLength = selected.join("\n").length;
      if (currentLength >= MAX_CONTENT_CHARS) break;
    }

    const compact = mergeContentLines(selected);
    return compact.length > MAX_CONTENT_CHARS ? `${compact.slice(0, MAX_CONTENT_CHARS - 3).trim()}...` : compact;
  }

  function cleanContentLine(text) {
    return cleanText(text)
      .replace(/^go to channel\s+/i, "")
      .replace(/^['’]s\s+(?:post|reel|video|photo|story)\s*:\s*/i, "")
      .replace(/^.+?[’']s\s+(?:post|reel|video|photo|story)\s*:\s*/i, "")
      .replace(/^.+?\s+(?:shared|posted|created|added)\s+(?:a\s+|an\s+)?(?:new\s+)?(?:post|reel|video|photo|story|short)\s*:\s*/i, "")
      .replace(/^(?:shared|posted|created|added)\s+(?:a\s+|an\s+)?(?:new\s+)?(?:post|reel|video|photo|story|short)\s*:\s*/i, "")
      .replace(/\s+\d+\s+(?:second|seconds|minute|minutes|hour|hours)(?:,\s*\d+\s+(?:second|seconds|minute|minutes))?$/i, "")
      .replace(/^[“"](.+)[”"]$/s, "$1")
      .trim();
  }

  function isNonContentLink(link) {
    const parsed = safeUrl(link?.url || "");
    if (!parsed) return false;
    const path = parsed.pathname.toLowerCase();
    if (/^\/music\//.test(path)) return true;
    if (/\/sound\//.test(path)) return true;
    if (/^\/reels?\/audio\//.test(path)) return true;
    return false;
  }

  function mergeContentLines(lines) {
    const deduped = [];
    for (const line of lines) {
      const key = fingerprint(line);
      if (!key) continue;
      const alreadyCovered = deduped.some((existing) => {
        const existingKey = fingerprint(existing);
        return existingKey === key || existingKey.includes(key) || key.includes(existingKey);
      });
      if (!alreadyCovered) deduped.push(line);
    }
    return deduped.join("\n").trim();
  }

  function isMeaningfulLine(text, accountName) {
    const cleaned = cleanText(text);
    if (!cleaned) return false;
    const key = normalizeKey(cleaned);
    if (!key) return false;
    if (accountName && key === normalizeKey(accountName)) return false;
    if (cleaned.length < 3) return false;
    if (isUiText(cleaned)) return false;
    if (isAccountNoiseText(cleaned)) return false;
    if (/^comment by\b/i.test(cleaned)) return false;
    if (/^community status:/i.test(cleaned)) return false;
    if (/^img\s+alt=/i.test(cleaned)) return false;
    if (isGeneratedAltText(cleaned)) return false;
    if (repeatedTokenPenalty(cleaned) > 25) return false;
    if (isMetricLike(cleaned)) return false;
    if (TIME_OR_DATE_RE.test(cleaned)) return false;
    if (FULL_DATE_TIME_RE.test(cleaned)) return false;
    if (/^https?:\/\//i.test(cleaned)) return false;
    if (/^[a-z][\w:-]*\s+[^<>]{0,220}=["']/i.test(cleaned)) return false;
    if (/^[\W_]+$/u.test(cleaned)) return false;
    if (/^\d+$/.test(cleaned)) return false;
    if (/^[\d\s.,:/-]+$/.test(cleaned)) return false;
    if (/^(·|•|\||-)+$/.test(cleaned)) return false;

    const words = cleaned.split(/\s+/).filter(Boolean);
    if (words.length === 1 && cleaned.length < 18 && !/^#/.test(cleaned) && !/@/.test(cleaned)) {
      return false;
    }

    return true;
  }

  function scoreContentLine(text) {
    const cleaned = cleanText(text);
    const words = cleaned.split(/\s+/).filter(Boolean);
    let score = 0;
    score += Math.min(cleaned.length / 4, 35);
    score += Math.min(words.length * 2, 30);
    if (/[.!?。！？]/.test(cleaned)) score += 8;
    if (/[#@]/.test(cleaned)) score += 4;
    if (/[’']/.test(cleaned)) score += 8;
    if (/[a-zA-ZÀ-ỹ]{2,}/.test(cleaned) && /\s/.test(cleaned)) score += 8;
    if (looksLikeNameOnly(cleaned) && !/[’']/.test(cleaned)) score -= 18;
    if (cleaned.length < 20 && words.length <= 3) score -= 12;
    if (/^(follow|subscribe|join|like|comment|share)\b/i.test(cleaned)) score -= 15;
    return score;
  }

  function computePostConfidence(post) {
    let confidence = 0;
    if (post.account?.name) confidence += 0.2;
    if (post.account?.url) confidence += 0.15;
    if (post.postUrl) confidence += 0.25;
    if (post.content && post.content.length >= 20) confidence += 0.25;
    if (Object.keys(post.stats || {}).length) confidence += 0.15;
    if ((post.links || []).some((link) => link.kind === "post")) confidence += 0.1;
    if ((post.records || []).length > 80 && !post.postUrl) confidence -= 0.15;
    return Math.max(0, Math.min(1, Number(confidence.toFixed(2))));
  }

  function isUsefulPost(post) {
    if (!post) return false;
    if (post.content && post.content.length >= 20) return true;
    if (post.postUrl && (post.content || Object.keys(post.stats || {}).length) && (post.accountName || Object.keys(post.stats || {}).length)) return true;
    if (post.accountName && Object.keys(post.stats || {}).length && post.confidence >= 0.35) return true;
    return false;
  }

  function collectEntityItems(root, allLinks, context) {
    const items = [];
    const bestByUrl = new Map();
    const maxItems = context.maxItems || 160;

    for (const link of allLinks) {
      if (items.length >= maxItems * 4) break;
      if (!link || link.kind === "post") continue;
      if (hasSuppressedChromeAncestor(link.element)) continue;

      const type = classifyEntityType(link.url);
      if (!type) continue;
      if (isCurrentPageChromeLink(link, context.baseUrl)) continue;

      const name = cleanEntityName(link.text || link.ariaLabel || link.title);
      if (!isLikelyEntityName(name, type)) continue;

      const container = findEntityItemContainer(link.element, root, context);
      const records = collectTextRecords(container || link.element, context.textMemo);
      const links = collectLinks(container || link.element, context.baseUrl, context.linkMemo);
      const details = extractEntityDetails(records, links, name);
      const score = scoreEntityItem({ name, type, link, details, container, records, links });
      if (score < 20) continue;

      const item = pruneEmpty({
        type,
        name,
        url: link.url,
        details: details.details,
        actions: details.actions,
        confidence: Math.max(0.2, Math.min(1, Number((score / 100).toFixed(2)))),
      });
      const key = canonicalizeUrlForDedupe(link.url) || `${type}:${normalizeKey(name)}`;
      const existing = bestByUrl.get(key);
      if (!existing || score > existing.score) {
        bestByUrl.set(key, { item, score, order: items.length });
      }
      items.push(item);
    }

    return Array.from(bestByUrl.values())
      .sort((a, b) => {
        if (b.score !== a.score) return b.score - a.score;
        return a.order - b.order;
      })
      .map((entry) => entry.item)
      .slice(0, maxItems);
  }

  function shouldDisplayEntityItems(baseUrl, items, posts) {
    if (!items.length) return false;
    if (isEntityListPage(baseUrl)) return true;
    if (!posts.length) return true;
    return false;
  }

  function isEntityListPage(baseUrl) {
    const parsed = safeUrl(baseUrl || "");
    if (!parsed) return false;
    const path = parsed.pathname.toLowerCase();
    return /\/(groups\/joins|friends|members|followers|following|connections|people|pages|communities|channels|subscriptions|mynetwork)\b/.test(path);
  }

  function classifyEntityType(url) {
    const parsed = safeUrl(url);
    if (!parsed || isPostUrl(url)) return "";
    const host = stripWww(parsed.hostname).toLowerCase();
    const path = parsed.pathname.replace(/\/+$/, "");
    const lowerPath = path.toLowerCase();
    const segments = lowerPath.split("/").filter(Boolean);
    if (!isSocialHost(host) || segments.length === 0) return "";

    if (host.includes("facebook.")) {
      if (lowerPath === "/profile.php" && parsed.searchParams.get("id")) return "profile";
      if (/^\/groups\/[^/]+/.test(lowerPath) && segments[1] && !["joins", "discover", "feed", "create", "category"].includes(segments[1])) return "group";
      if (/^\/people\/[^/]+/.test(lowerPath)) return "profile";
      if (/^\/pages\/[^/]+/.test(lowerPath)) return "page";
      if (segments.length === 1 && !isTopLevelSocialRoute(segments[0])) return "account";
      return "";
    }

    if (host === "x.com" || host.includes("twitter.")) {
      return /^\/[a-z0-9_]{1,20}$/i.test(path) && !isTopLevelSocialRoute(segments[0]) ? "account" : "";
    }

    if (host.includes("linkedin.")) {
      if (/^\/in\/[^/]+/.test(lowerPath)) return "profile";
      if (/^\/company\/[^/]+/.test(lowerPath)) return "page";
      if (/^\/school\/[^/]+/.test(lowerPath)) return "page";
      if (/^\/showcase\/[^/]+/.test(lowerPath)) return "page";
      return "";
    }

    if (host.includes("youtube.") || host === "youtu.be") {
      if (/^\/(@[^/]+|channel\/[^/]+|c\/[^/]+|user\/[^/]+)/.test(path)) return "channel";
      return "";
    }

    if (host.includes("tiktok.")) return /^\/@[^/]+/.test(path) ? "account" : "";
    if (host.includes("instagram.")) return segments.length === 1 && !isTopLevelSocialRoute(segments[0]) ? "account" : "";
    if (host.includes("threads.")) return /^\/@[^/]+/.test(path) ? "account" : "";
    if (host.includes("reddit.")) {
      if (/^\/r\/[^/]+/.test(lowerPath)) return "community";
      if (/^\/user\/[^/]+/.test(lowerPath)) return "account";
      return "";
    }

    return isAccountUrl(url) ? "account" : "";
  }

  function isTopLevelSocialRoute(segment) {
    return new Set([
      "home",
      "watch",
      "marketplace",
      "gaming",
      "groups",
      "friends",
      "reels",
      "notifications",
      "messages",
      "explore",
      "search",
      "settings",
      "privacy",
      "login",
      "signup",
      "jobs",
      "help",
      "hashtag",
      "topics",
      "i",
      "compose",
      "feed",
      "shorts",
      "trending",
      "subscriptions",
      "playlist",
      "pages",
      "people",
      "members",
      "events",
      "saved",
    ]).has(normalizeKey(segment));
  }

  function isCurrentPageChromeLink(link, baseUrl) {
    const text = cleanEntityName(link.text || link.ariaLabel || link.title);
    if (!text || isUiText(text)) return true;
    const parsed = safeUrl(link.url);
    const base = safeUrl(baseUrl || "");
    if (!parsed || !base) return false;
    const samePath = stripWww(parsed.hostname) === stripWww(base.hostname) && parsed.pathname.replace(/\/+$/, "") === base.pathname.replace(/\/+$/, "");
    return samePath && normalizeKey(text) === normalizeKey(base.hostname);
  }

  function cleanEntityName(text) {
    return cleanText(text)
      .replace(/^view\s+(?:group|profile|page|channel|community)\s*/i, "")
      .replace(/\s*,?\s*(?:profile picture|avatar)$/i, "")
      .replace(/\s+profile$/i, "")
      .replace(/\s+/g, " ")
      .trim();
  }

  function isLikelyEntityName(name, type) {
    const cleaned = cleanEntityName(name);
    if (!cleaned || cleaned.length < 2 || cleaned.length > 120) return false;
    if (isUiText(cleaned) || isMetricLike(cleaned) || TIME_OR_DATE_RE.test(cleaned) || FULL_DATE_TIME_RE.test(cleaned)) return false;
    if (isGeneratedAltText(cleaned) || isAccountNoiseText(cleaned)) return false;
    if (/^https?:\/\//i.test(cleaned)) return false;
    if (/^[\d\s.,:/-]+$/.test(cleaned)) return false;
    if (type === "group" || type === "page" || type === "community" || type === "channel") return /[\p{L}\p{N}]/u.test(cleaned);
    return isNameLike(cleaned) || /^@?[a-z0-9._-]{2,40}$/i.test(cleaned);
  }

  function findEntityItemContainer(linkElement, root, context) {
    let node = parentElement(linkElement);
    let best = node || linkElement;
    let climb = 0;

    while (node && node !== root && climb < 12) {
      if (!isElementNode(node) || shouldSkipElement(node) || isSuppressedChromeContainer(node)) break;

      const role = normalizeKey(getAttr(node, "role"));
      const records = collectTextRecords(node, context.textMemo);
      const links = collectLinks(node, context.baseUrl, context.linkMemo);
      const textChars = records.map((record) => record.text).join(" ").length;
      const entityLinkCount = links.filter((candidate) => classifyEntityType(candidate.url)).length;
      const postLinkCount = links.filter((candidate) => candidate.kind === "post").length;
      const compactEnough = textChars <= 900 && links.length <= 18 && entityLinkCount <= 6 && postLinkCount <= 1;

      if (compactEnough) best = node;
      if (role === "listitem" && textChars <= 1200) return node;
      if (!compactEnough && textChars > 900) break;

      node = parentElement(node);
      climb += 1;
    }

    return best;
  }

  function extractEntityDetails(records, links, name) {
    const nameKey = normalizeKey(name);
    const linkTextKeys = new Set(
      links
        .map((link) => cleanEntityName(link.text || link.ariaLabel || link.title))
        .map(normalizeKey)
        .filter(Boolean),
    );
    const details = [];
    const actions = [];

    for (let index = 0; index < records.length; index += 1) {
      let text = cleanEntityDetailLine(records[index].text);
      const nextText = cleanEntityDetailLine(records[index + 1]?.text || "");
      if (isEntityStatusText(text) && TIME_OR_DATE_RE.test(nextText)) {
        text = `${text} ${nextText}`;
      }
      const key = normalizeKey(text);
      if (!key || key === nameKey) continue;
      if (linkTextKeys.has(key) && !isEntityActionText(text)) continue;
      if (TIME_OR_DATE_RE.test(text) || FULL_DATE_TIME_RE.test(text)) continue;
      if (isGeneratedAltText(text) || /^https?:\/\//i.test(text)) continue;

      if (isEntityActionText(text)) {
        if (!actions.some((action) => normalizeKey(action) === key)) actions.push(text);
        continue;
      }

      if (!isEntityDetailText(text)) continue;
      if (!details.some((detail) => normalizeKey(detail) === key)) details.push(text);
    }

    return {
      details: details.slice(0, 4),
      actions: actions.slice(0, 3),
    };
  }

  function cleanEntityDetailLine(text) {
    return cleanText(text)
      .replace(/\s+/g, " ")
      .trim();
  }

  function isEntityDetailText(text) {
    const key = normalizeKey(text);
    if (!key || key.length < 3 || text.length > 180) return false;
    if (/^[a-z][\w:-]*\s+[^<>]{0,220}=["']/.test(text)) return false;
    if (/^(public|private|closed|visible|hidden)\s+group\b/.test(key)) return true;
    if (isEntityStatusText(text)) return true;
    if (/\b(mutual friends?|members?|followers?|connections?|subscribers?)\b/.test(key)) return true;
    if (/\b(admin|moderator|owner|creator|manager)\b/.test(key)) return true;
    if (/^[0-9][0-9,.\s]*(k|m|b)?\s+(members?|followers?|connections?|subscribers?)\b/.test(key)) return true;
    if (isUiText(text) || isMetricLike(text)) return false;
    if (/^(sort|more|view|open|search|create|edit|manage)\b/.test(key)) return false;
    return text.split(/\s+/).length >= 2 && /[\p{L}]/u.test(text);
  }

  function isEntityActionText(text) {
    const key = normalizeKey(text);
    return /^(answer questions|update responses|add friend|confirm|delete|remove|message|follow|connect|invite|join group|cancel request|request sent)$/.test(key);
  }

  function isEntityStatusText(text) {
    const key = normalizeKey(text);
    return /^(requested to join|you last visited|joined|following|follows you|connected|pending|invited)\b/.test(key);
  }

  function scoreEntityItem({ name, type, link, details, container, records, links }) {
    let score = 20;
    if (type) score += 18;
    if (name && name.length >= 3) score += 20;
    if (isAccountUrl(link.url) || classifyEntityType(link.url)) score += 15;
    if ((details.details || []).length) score += Math.min(details.details.length, 3) * 8;
    if ((details.actions || []).length) score += 4;
    if (normalizeKey(getAttr(container, "role")) === "listitem") score += 14;
    if ((records || []).length > 40) score -= 15;
    if ((links || []).length > 20) score -= 15;
    if (isUiText(name) || isAccountNoiseText(name)) score -= 80;
    return score;
  }

  function collectTextRecords(root, memo) {
    if (memo && memo.has(root)) return memo.get(root);

    const records = [];
    let position = 0;

    function visit(node, blocked) {
      if (!node || position > MAX_WALK_NODES) return;
      if (isElementNode(node)) {
        if (shouldSkipElement(node)) {
          for (const attrName of ["aria-label", "title"]) {
            const attrValue = cleanText(getAttr(node, attrName));
            if (metricKeyForLabel(attrValue)) {
              records.push({
                text: attrValue,
                element: node,
                source: attrName,
                order: position++,
              });
              break;
            }
          }
          return;
        }
        const nextBlocked = blocked || isNoiseContainer(node);

        for (const attrName of ["aria-label", "title", "alt"]) {
          const attrValue = cleanText(getAttr(node, attrName));
          if (attrValue && (isMetricLike(attrValue) || isMeaningfulLine(attrValue, null))) {
            records.push({
              text: attrValue,
              element: node,
              source: attrName,
              order: position++,
            });
          }
        }

        for (const child of childNodes(node)) {
          visit(child, nextBlocked);
        }
        return;
      }

      if (isTextNode(node) && !blocked) {
        for (const part of splitText(cleanText(textOf(node)))) {
          if (!part) continue;
          records.push({
            text: part,
            element: parentElement(node),
            source: "text",
            order: position++,
          });
        }
      }
    }

    visit(root, false);
    const deduped = uniqueBy(records, (record) => `${normalizeKey(record.text)}:${record.source}:${record.order}`);
    if (memo) memo.set(root, deduped);
    return deduped;
  }

  function collectLinks(root, baseUrl, memo) {
    if (memo && memo.has(root)) return memo.get(root);

    const links = [];
    walkElements(root, (node) => {
      if (tagName(node) !== "a" && tagName(node) !== "area") return true;
      if (shouldSkipElement(node)) return true;
      if (hasSuppressedChromeAncestor(node)) return true;

      const rawHref = getAttr(node, "href");
      const url = normalizeUrl(rawHref, baseUrl);
      if (!url) return true;

      const text = cleanText(textContentOf(node) || getAttr(node, "aria-label") || getAttr(node, "title"));
      const link = {
        url,
        text,
        ariaLabel: cleanText(getAttr(node, "aria-label")),
        title: cleanText(getAttr(node, "title")),
        kind: classifySocialUrl(url),
        element: node,
      };
      links.push(link);
      return true;
    });

    if (memo) memo.set(root, links);
    return links;
  }

  function extractStatsFromRecords(records) {
    const stats = {};
    let hitCount = 0;

    for (const record of records) {
      const found = parseMetricsFromText(record.text);
      for (const metric of found) {
        hitCount += 1;
        const existing = stats[metric.key];
        if (!existing || metric.score > existing.score) {
          stats[metric.key] = {
            raw: metric.raw,
            value: metric.value,
          };
          Object.defineProperty(stats[metric.key], "_score", {
            value: metric.score,
            enumerable: false,
            configurable: true,
          });
        }
      }
    }

    for (let index = 0; index < records.length - 1; index += 1) {
      const labelKey = metricKeyForLabel(records[index].text);
      if (!labelKey) continue;

      for (let lookahead = index + 1; lookahead <= Math.min(records.length - 1, index + 3); lookahead += 1) {
        const valueText = cleanText(records[lookahead].text);
        const value = parseMetricValue(valueText);
        if (!Number.isFinite(value)) continue;

        const raw = `${valueText} ${records[index].text}`;
        if (isMetricFalsePositive(labelKey, raw, raw)) break;
        hitCount += 1;

        const score = metricScore(labelKey, raw, value) + 4;
        const existing = stats[labelKey];
        if (!existing || score > existing._score) {
          stats[labelKey] = {
            raw,
            value,
          };
          Object.defineProperty(stats[labelKey], "_score", {
            value: score,
            enumerable: false,
            configurable: true,
          });
        }
        break;
      }
    }

    for (const value of Object.values(stats)) {
      if (value && Object.prototype.hasOwnProperty.call(value, "_score")) {
        delete value._score;
      }
    }

    return { stats, hitCount };
  }

  function parseMetricsFromText(text) {
    const cleaned = cleanText(text);
    if (!cleaned) return [];

    const results = [];
    const labels = Array.from(METRIC_LABEL_TO_KEY.keys())
      .sort((a, b) => b.length - a.length)
      .map(escapeRegExp);
    const labelGroup = labels.join("|");
    const numberGroup = "([0-9][0-9.,\\s]*\\s*(?:K|M|B|T|k|m|b|t)?|[0-9]+(?:[.,][0-9]+)?\\s*(?:nghìn|ngan|triệu|trieu|tỷ|ty))";

    const numberBefore = new RegExp(`${numberGroup}\\s*(${labelGroup})\\b`, "giu");
    const labelBefore = new RegExp(`\\b(${labelGroup})\\s*[:\\-]?\\s*${numberGroup}`, "giu");

    collectMetricMatches(cleaned, numberBefore, false, results);
    collectMetricMatches(cleaned, labelBefore, true, results);

    const compactX = /\b([0-9][0-9.,\s]*\s*[KMBTkmbt]?)\s+(replies|reply|reposts|repost|retweets|retweet|likes|like|views|view|bookmarks|bookmark)\b/giu;
    collectMetricMatches(cleaned, compactX, false, results);

    return dedupeMetrics(results);
  }

  function formatMetricForText(metric) {
    const raw = cleanText(metric.raw);
    if (Number.isFinite(metric.value) && /^[\p{L}\s]+:\s*[0-9,.]+/u.test(raw)) {
      return String(metric.value);
    }
    return raw;
  }

  function collectMetricMatches(text, regex, labelFirst, results) {
    let match;
    while ((match = regex.exec(text))) {
      const label = labelFirst ? match[1] : match[2];
      const number = labelFirst ? match[2] : match[1];
      const key = metricKeyForLabel(label);
      if (!key) continue;

      const value = parseMetricValue(number);
      const raw = cleanText(match[0]);
      if (isMetricFalsePositive(key, raw, text)) continue;
      results.push({
        key,
        raw,
        value,
        score: metricScore(key, raw, value),
      });
    }
  }

  function isMetricFalsePositive(key, raw, fullText) {
    const rawKey = normalizeKey(raw);
    const fullKey = normalizeKey(fullText);
    if (key === "views" && /^view\s+\d+/.test(rawKey)) return true;
    if (key === "views" && /\bview\s+(all\s+)?\d+\s+(reply|replies|comment|comments)\b/.test(fullKey)) return true;
    return false;
  }

  function dedupeMetrics(metrics) {
    const byKey = new Map();
    for (const metric of metrics) {
      const existing = byKey.get(metric.key);
      if (!existing || metric.score > existing.score) byKey.set(metric.key, metric);
    }
    return Array.from(byKey.values());
  }

  function metricScore(key, raw, value) {
    let score = 10;
    if (Number.isFinite(value)) score += Math.min(Math.log10(Math.max(value, 1)) * 3, 20);
    if (raw.length < 30) score += 6;
    if (key === "views") score += 2;
    return score;
  }

  function parseMetricValue(raw) {
    if (raw == null) return null;
    const text = String(raw).trim().toLowerCase();
    const suffixMap = {
      k: 1_000,
      m: 1_000_000,
      b: 1_000_000_000,
      t: 1_000_000_000_000,
      nghìn: 1_000,
      ngan: 1_000,
      triệu: 1_000_000,
      trieu: 1_000_000,
      tỷ: 1_000_000_000,
      ty: 1_000_000_000,
    };
    const match = text.match(/^([0-9][0-9.,\s]*)(?:\s*(k|m|b|t|nghìn|ngan|triệu|trieu|tỷ|ty))?$/i);
    if (!match) return null;

    let numberText = match[1].replace(/\s+/g, "");
    const suffix = match[2] ? match[2].toLowerCase() : "";
    const multiplier = suffixMap[suffix] || 1;

    if (suffix && numberText.includes(",") && !numberText.includes(".")) {
      numberText = numberText.replace(",", ".");
    } else if (!suffix) {
      const commaCount = (numberText.match(/,/g) || []).length;
      const dotCount = (numberText.match(/\./g) || []).length;
      if (commaCount && dotCount) {
        const lastComma = numberText.lastIndexOf(",");
        const lastDot = numberText.lastIndexOf(".");
        const decimalMark = lastComma > lastDot ? "," : ".";
        const thousandsMark = decimalMark === "," ? "." : ",";
        numberText = numberText.split(thousandsMark).join("");
        numberText = numberText.replace(decimalMark, ".");
      } else if (commaCount === 1 && /\d,\d{1,2}$/.test(numberText)) {
        numberText = numberText.replace(",", ".");
      } else {
        numberText = numberText.replace(/[,.]/g, "");
      }
    }

    const parsed = Number.parseFloat(numberText);
    if (!Number.isFinite(parsed)) return null;
    return Math.round(parsed * multiplier);
  }

  function isMetricLike(text) {
    return parseMetricsFromText(text).length > 0;
  }

  function metricKeyForLabel(label) {
    return METRIC_LABEL_TO_KEY.get(normalizeKey(label));
  }

  function buildMetricLabelMap(metricWords) {
    const map = new Map();
    for (const [key, words] of Object.entries(metricWords)) {
      for (const word of words) map.set(normalizeKey(word), key);
    }
    return map;
  }

  function classifySocialUrl(url) {
    if (isPostUrl(url)) return "post";
    if (isAccountUrl(url)) return "account";
    return "other";
  }

  function isPostUrl(url) {
    const parsed = safeUrl(url);
    if (!parsed) return false;
    const host = parsed.hostname.replace(/^www\./, "").toLowerCase();
    const path = parsed.pathname.toLowerCase();
    const query = parsed.search.toLowerCase();

    if (host.includes("facebook.") || host === "fb.watch") {
      return (
        host === "fb.watch" ||
        /\/(posts|permalink|videos|watch|share\/(?:p|r|v)|groups\/[^/]+\/posts)\b/.test(path) ||
        /\/reel\/[a-z0-9_-]+/.test(path) ||
        path.includes("/story.php") ||
        path.includes("/photo.php") ||
        /[?&](story_fbid|fbid|v)=/.test(query)
      );
    }

    if (host === "x.com" || host.includes("twitter.")) return /^\/[^/]+\/status\/\d+/.test(path);
    if (host.includes("linkedin.")) {
      if (/^\/company\/[^/]+\/posts\/?$/.test(path)) return false;
      return /\/(feed\/update|posts|pulse|video)\b/.test(path);
    }
    if (host.includes("youtube.") || host === "youtu.be") return host === "youtu.be" || /\/(watch|shorts|post|live|clip)\b/.test(path) || /[?&]v=/.test(query);
    if (host.includes("tiktok.")) return /\/@[^/]+\/video\/\d+/.test(path) || /\/video\/\d+/.test(path);
    if (host.includes("instagram.")) {
      if (/^\/reels?\/audio\//.test(path)) return false;
      return /\/(p|reel|reels|tv)\/[^/]+/.test(path);
    }
    if (host.includes("threads.")) return /\/@[^/]+\/post\/[^/]+/.test(path);
    if (host.includes("reddit.")) return /\/comments\/[^/]+/.test(path);

    return isSocialHost(host) && /\/(post|posts|status|update|video|videos|reel|shorts|article|watch)\b/.test(path);
  }

  function isAccountUrl(url) {
    const parsed = safeUrl(url);
    if (!parsed) return false;
    if (isPostUrl(url)) return false;

    const host = parsed.hostname.replace(/^www\./, "").toLowerCase();
    const path = parsed.pathname.replace(/\/+$/, "");
    const lowerPath = path.toLowerCase();
    const segments = lowerPath.split("/").filter(Boolean);
    const excluded = new Set([
      "home",
      "watch",
      "marketplace",
      "gaming",
      "groups",
      "reels",
      "notifications",
      "messages",
      "explore",
      "search",
      "settings",
      "privacy",
      "login",
      "signup",
      "jobs",
      "help",
      "hashtag",
      "topics",
      "i",
      "compose",
      "notifications",
      "messages",
      "feed",
      "shorts",
      "watch",
      "trending",
      "subscriptions",
      "playlist",
    ]);

    if (!isSocialHost(host)) return false;

    if (host.includes("facebook.")) {
      if (lowerPath === "" || lowerPath === "/") return false;
      if (lowerPath === "/profile.php" && parsed.searchParams.get("id")) return true;
      if (/^\/(people|pages)\/[^/]+/.test(lowerPath)) return true;
      if (segments.length === 1 && !excluded.has(segments[0])) return true;
      if (segments.length === 2 && segments[0] === "groups") return true;
      return false;
    }

    if (host === "x.com" || host.includes("twitter.")) {
      return /^\/[a-z0-9_]{1,20}$/i.test(path) && !excluded.has(segments[0]);
    }

    if (host.includes("linkedin.")) return /^\/(in|company|school|showcase)\/[^/]+/.test(lowerPath);
    if (host.includes("youtube.")) return /^\/(@[^/]+|channel\/[^/]+|c\/[^/]+|user\/[^/]+)/.test(path);
    if (host.includes("tiktok.")) return /^\/@[^/]+/.test(path);
    if (host.includes("instagram.")) return segments.length === 1 && !excluded.has(segments[0]);
    if (host.includes("threads.")) return /^\/@[^/]+/.test(path);
    if (host.includes("reddit.")) return /^\/(r|user)\/[^/]+/.test(lowerPath);

    return segments.length === 1 && !excluded.has(segments[0]);
  }

  function isSocialHost(host) {
    return SOCIAL_HOST_PARTS.some((part) => host.includes(part));
  }

  function detectPlatformHint(url) {
    const parsed = safeUrl(url);
    if (!parsed) return "unknown";
    const host = parsed.hostname.replace(/^www\./, "").toLowerCase();
    if (host.includes("facebook.") || host === "fb.watch") return "facebook";
    if (host === "x.com" || host.includes("twitter.")) return "x";
    if (host.includes("linkedin.")) return "linkedin";
    if (host.includes("youtube.") || host === "youtu.be") return "youtube";
    if (host.includes("tiktok.")) return "tiktok";
    if (host.includes("instagram.")) return "instagram";
    if (host.includes("threads.")) return "threads";
    if (host.includes("reddit.")) return "reddit";
    return "unknown";
  }

  function normalizeUrl(rawHref, baseUrl) {
    const href = cleanText(rawHref);
    if (!href || href === "#") return "";
    if (/^(javascript|mailto|tel|sms|data):/i.test(href)) return "";

    try {
      const url = new URL(href, baseUrl || "https://example.invalid/");
      if (!/^https?:$/i.test(url.protocol)) return "";
      const unwrapped = unwrapRedirectUrl(url);
      if (unwrapped) return normalizeUrl(unwrapped, baseUrl);
      const junkParams = new Set([
        "fbclid",
        "utm_source",
        "utm_medium",
        "utm_campaign",
        "utm_content",
        "utm_term",
        "ref_src",
        "__tn__",
        "mibextid",
        "s",
        "checkpoint_src",
        "comment_id",
        "reply_comment_id",
      ]);
      for (const key of Array.from(url.searchParams.keys())) {
        if (junkParams.has(key) || key.startsWith("__cft__")) {
          url.searchParams.delete(key);
        }
      }
      url.hash = "";
      return url.href;
    } catch (_) {
      return "";
    }
  }

  function normalizeOutputUrl(url, baseUrl) {
    const parsed = safeUrl(url);
    if (!parsed) return cleanText(url);

    const unwrapped = unwrapRedirectUrl(parsed);
    if (unwrapped) return normalizeOutputUrl(unwrapped, baseUrl);

    const base = safeUrl(baseUrl || "");
    const cleaned = new URL(parsed.href);
    normalizeSocialUrlInPlace(cleaned);
    const importantParams = importantQueryParams(cleaned);
    cleaned.search = "";
    for (const [key, value] of importantParams) {
      cleaned.searchParams.set(key, value);
    }
    cleaned.hash = "";

    const sameDomain =
      base &&
      stripWww(base.hostname).toLowerCase() === stripWww(cleaned.hostname).toLowerCase();

    const pathAndQuery = `${cleaned.pathname || "/"}${cleaned.search}`;
    if (sameDomain) return pathAndQuery;
    return `${cleaned.origin}${pathAndQuery}`;
  }

  function importantQueryParams(url) {
    const keep = new Set([
      "id",
      "story_fbid",
      "fbid",
      "v",
      "set",
      "group_id",
      "profile_id",
      "post_id",
      "video_id",
      "photo_id",
      "share_id",
      "story_id",
      "tweet_id",
      "update_id",
      "urn",
    ]);
    const output = [];
    for (const [key, value] of url.searchParams.entries()) {
      const lower = key.toLowerCase();
      if (keep.has(lower)) {
        output.push([key, value]);
      }
    }
    return output;
  }

  function unwrapRedirectUrl(url) {
    const host = stripWww(url.hostname).toLowerCase();
    const path = url.pathname.toLowerCase();
    const redirectHosts = new Set([
      "l.facebook.com",
      "lm.facebook.com",
      "m.facebook.com",
      "lnkd.in",
      "linkedin.com",
      "youtube.com",
      "t.co",
      "twitter.com",
      "x.com",
    ]);

    const looksLikeRedirect =
      (host === "l.facebook.com" && path.includes("/l.php")) ||
      (host === "lm.facebook.com" && path.includes("/l.php")) ||
      (host.includes("linkedin.com") && path.includes("/safety/go")) ||
      (host.includes("youtube.com") && path.includes("/redirect")) ||
      host === "t.co";

    if (!looksLikeRedirect && !redirectHosts.has(host)) return "";

    for (const key of ["u", "url", "q", "target"]) {
      const value = url.searchParams.get(key);
      if (value && /^https?:\/\//i.test(value)) return value;
    }
    return "";
  }

  function stripWww(hostname) {
    return String(hostname || "").replace(/^www\./i, "");
  }

  function canonicalizeUrlForDedupe(url) {
    const parsed = safeUrl(url);
    if (!parsed) return url || "";
    normalizeSocialUrlInPlace(parsed);
    parsed.hash = "";
    for (const key of ["fbclid", "utm_source", "utm_medium", "utm_campaign", "utm_content", "utm_term"]) {
      parsed.searchParams.delete(key);
    }
    return parsed.href.replace(/\/+$/, "");
  }

  function normalizeSocialUrlInPlace(url) {
    const host = stripWww(url.hostname).toLowerCase();
    if (host === "x.com" || host.includes("twitter.")) {
      const match = url.pathname.match(/^\/([A-Za-z0-9_]{1,20})\/status\/(\d+)(?:\/analytics)?\/?$/);
      if (match) url.pathname = `/${match[1]}/status/${match[2]}`;
    }
    if (host.includes("linkedin.")) {
      const match = url.pathname.match(/^\/(company|school|showcase)\/([^/]+)\/posts\/?$/i);
      if (match) url.pathname = `/${match[1]}/${match[2]}`;
    }
  }

  function safeUrl(url) {
    try {
      return new URL(url);
    } catch (_) {
      return null;
    }
  }

  function uniqueAccounts(posts) {
    const map = new Map();
    for (const post of posts) {
      if (!post.accountName && !post.accountUrl) continue;
      const key = post.accountUrl || normalizeKey(post.accountName);
      if (!map.has(key)) {
        map.set(key, {
          name: post.accountName || "",
          url: post.accountUrl || "",
        });
      }
    }
    return Array.from(map.values());
  }

  function uniqueLinks(links) {
    return uniqueBy(links, canonicalizeUrlForDedupe);
  }

  function buildMeaningfulText(posts, items, baseUrl) {
    const lines = [];
    for (const post of posts) {
      if (post.accountName) lines.push(post.accountName);
      if (post.accountUrl) lines.push(normalizeOutputUrl(post.accountUrl, baseUrl));
      if (post.postUrl) lines.push(normalizeOutputUrl(post.postUrl, baseUrl));
      if (post.content) lines.push(post.content);
      for (const metric of Object.values(post.stats || {})) {
        if (metric?.raw) lines.push(metric.raw);
      }
    }
    for (const item of items || []) {
      if (item.name) lines.push(item.name);
      if (item.url) lines.push(normalizeOutputUrl(item.url, baseUrl));
      for (const detail of item.details || []) lines.push(detail);
      for (const action of item.actions || []) lines.push(action);
    }
    return uniqueBy(lines, normalizeKey);
  }

  function dedupeNearDuplicatePosts(posts) {
    const selected = [];
    for (const post of posts) {
      const textKey = fingerprint([post.accountName, post.content].join(" "));
      const duplicate = selected.some((existing) => {
        if (post.postUrl && existing.postUrl && canonicalizeUrlForDedupe(post.postUrl) === canonicalizeUrlForDedupe(existing.postUrl)) return true;
        const existingKey = fingerprint([existing.accountName, existing.content].join(" "));
        return textKey && existingKey && (textKey === existingKey || textKey.includes(existingKey) || existingKey.includes(textKey));
      });
      if (!duplicate) selected.push(post);
    }
    return selected;
  }

  function shouldSkipElement(node) {
    const tag = tagName(node);
    if (BLOCKED_TAGS.has(tag)) return true;
    if (getAttr(node, "hidden") !== "") return true;
    if (normalizeKey(getAttr(node, "aria-hidden")) === "true") return true;

    const style = getAttr(node, "style").toLowerCase();
    if (/display\s*:\s*none|visibility\s*:\s*hidden/.test(style)) return true;

    return false;
  }

  function hasSuppressedChromeAncestor(node) {
    let current = isElementNode(node) ? node : parentElement(node);
    while (current) {
      if (isSuppressedChromeContainer(current)) return true;
      current = parentElement(current);
    }
    return false;
  }

  function isSuppressedChromeContainer(node) {
    if (!isElementNode(node)) return false;
    const tag = tagName(node);
    const role = normalizeKey(getAttr(node, "role"));
    const label = normalizeKey(getAttr(node, "aria-label") || getAttr(node, "title"));

    if (tag === "nav" || tag === "header" || tag === "footer") return true;
    if (role === "navigation" || role === "banner" || role === "menu" || role === "menubar" || role === "search" || role === "tooltip") return true;
    if (role === "dialog" && /^(notifications|messenger|account controls and settings|facebook menu|menu|search)/.test(label)) return true;
    if (/^(notifications|account controls and settings|facebook menu)$/.test(label)) return true;
    return false;
  }

  function isNoiseContainer(node) {
    const tag = tagName(node);
    const role = normalizeKey(getAttr(node, "role"));
    if (tag === "nav" || tag === "header" || tag === "footer" || tag === "aside") return true;
    if (NOISE_ROLES.has(role)) return true;
    return false;
  }

  function walkElements(root, visitor) {
    let count = 0;

    function walk(node) {
      if (!node || count >= MAX_WALK_NODES) return;
      if (isElementNode(node)) {
        count += 1;
        const shouldContinue = visitor(node);
        if (shouldContinue === false) return;
      }
      for (const child of childNodes(node)) {
        walk(child);
      }
    }

    walk(root);
  }

  function isElementNode(node) {
    return !!node && node.nodeType === 1;
  }

  function isTextNode(node) {
    return !!node && node.nodeType === 3;
  }

  function tagName(node) {
    return String(node?.tagName || "").toLowerCase();
  }

  function childNodes(node) {
    if (!node) return [];
    const children = node.childNodes ? Array.from(node.childNodes) : [];
    if (node.shadowRoot && node.shadowRoot.childNodes) {
      children.push(...Array.from(node.shadowRoot.childNodes));
    }
    return children;
  }

  function parentElement(node) {
    let parent = node?.parentElement || node?.parentNode || null;
    while (parent && !isElementNode(parent)) parent = parent.parentElement || parent.parentNode || null;
    return parent;
  }

  function nodeContains(parent, child) {
    if (!parent || !child || parent === child) return parent === child;
    if (typeof parent.contains === "function") return parent.contains(child);
    let node = child.parentNode;
    while (node) {
      if (node === parent) return true;
      node = node.parentNode;
    }
    return false;
  }

  function getAttr(node, name) {
    if (!node || !name) return "";
    if (typeof node.getAttribute === "function") {
      const value = node.getAttribute(name);
      return value == null ? "" : String(value);
    }
    const lower = name.toLowerCase();
    if (node.attributesMap && Object.prototype.hasOwnProperty.call(node.attributesMap, lower)) {
      return String(node.attributesMap[lower]);
    }
    return "";
  }

  function textOf(node) {
    return node?.textContent || "";
  }

  function textContentOf(node) {
    if (!node) return "";
    if (typeof node.textContent === "string") return node.textContent;
    if (isTextNode(node)) return textOf(node);
    return childNodes(node)
      .map((child) => textContentOf(child))
      .join(" ");
  }

  function splitText(text) {
    return cleanText(text)
      .split(/\n+|(?<=\S)\s{3,}(?=\S)/u)
      .map(cleanText)
      .filter(Boolean);
  }

  function cleanText(value) {
    return decodeEntities(String(value || ""))
      .replace(/\u00a0/g, " ")
      .replace(/[\u200b-\u200f\u202a-\u202e\u2060]/g, "")
      .replace(/[ \t\r\f\v]+/g, " ")
      .replace(/\s*\n\s*/g, "\n")
      .trim();
  }

  function decodeEntities(value) {
    const text = String(value || "");
    if (!text.includes("&")) return text;

    const named = {
      amp: "&",
      lt: "<",
      gt: ">",
      quot: '"',
      apos: "'",
      nbsp: " ",
    };

    return text.replace(/&(#x?[0-9a-f]+|[a-z]+);/gi, (full, entity) => {
      const key = entity.toLowerCase();
      if (key[0] === "#") {
        const radix = key[1] === "x" ? 16 : 10;
        const number = Number.parseInt(key.slice(radix === 16 ? 2 : 1), radix);
        return Number.isFinite(number) ? String.fromCodePoint(number) : full;
      }
      return Object.prototype.hasOwnProperty.call(named, key) ? named[key] : full;
    });
  }

  function escapeHtml(value) {
    return String(value || "")
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function normalizeKey(text) {
    return cleanText(text)
      .toLowerCase()
      .normalize("NFD")
      .replace(/[\u0300-\u036f]/g, "")
      .replace(/[^\p{L}\p{N}#@]+/gu, " ")
      .replace(/\s+/g, " ")
      .trim();
  }

  function fingerprint(text) {
    return normalizeKey(text).replace(/\s+/g, "");
  }

  function isUiText(text) {
    const key = normalizeKey(text);
    if (!key) return true;
    if (UI_SET.has(key)) return true;
    if (key.length < 2) return true;
    if (/^(like|comment|share|reply|follow|subscribe|join|save)(\s+\d+)?$/.test(key)) return true;
    if (/^(thich|binh luan|chia se|tra loi|theo doi|tham gia|luu)(\s+\d+)?$/.test(key)) return true;
    if (/^(what'?s|what s) on your mind\b/.test(key)) return true;
    if (/^go to channel\b/.test(key)) return true;
    if (/^actions for this post\b/.test(key)) return true;
    if (/^(open user actions|actions available for this post)\b/.test(key)) return true;
    if (/^open control menu( for post)?\b/.test(key)) return true;
    if (/^hide post by\b/.test(key)) return true;
    if (/^(go to previous page of document|go to next page of document|slide to navigate to a page)\b/.test(key)) return true;
    if (/^(seek to live|restore all settings|opens subtitles settings dialog)\b/.test(key)) return true;
    if (/^search within this group\b/.test(key)) return true;
    if (/^(see everyone|see recommended groups|about featured section|expand collapse featured section|cover photo)\b/.test(key)) return true;
    if (/\bprofile$/.test(key)) return true;
    if (/^send this to friends\b/.test(key)) return true;
    if (/^[0-9]+(?:\s+[0-9]+)?\s*(k|m|b)?\s+members\b/.test(key)) return true;
    if (/^[0-9][0-9,.\s]*(k|m|b)?\s+(followers|connections)\b/.test(key)) return true;
    if (/^(watch more videos|watch in full screen|add to favorites|video progress|like video|read or add comments|share video)\b/.test(key)) return true;
    if (/^adjust volume\b/.test(key)) return true;
    if (/^tap to watch live\b/.test(key)) return true;
    if (/^(see more|see less|see translation|view translation)\b/.test(key)) return true;
    if (/^(create a post|create story|live video|stories tray|next items|feed posts|leave a comment|see who reacted to this)\b/.test(key)) return true;
    if (/^(video player|translation preferences|see original|rate this translation|open reel in reels viewer|play video|change position|enter fullscreen|change volume)\b/.test(key)) return true;
    if (/^(group by|public group|write something|anonymous post|sort group feed by|featured|featured content badge|discussion|people|events|media|files|joined|invite)\b/.test(key)) return true;
    if (/^(unlock premium tools|want to view more unlock the full document)\b/.test(key)) return true;
    return false;
  }

  function isGeneratedAltText(text) {
    const key = normalizeKey(text);
    return (
      /^may be (an?|a graphic|an image|image|photo|video)\b/.test(key) ||
      /^image may contain\b/.test(key) ||
      /^no photo description available\b/.test(key) ||
      /^flag of\b/.test(key) ||
      /\b(profile picture|avatar)\b/.test(key)
    );
  }

  function repeatedTokenPenalty(text) {
    const tokens = normalizeKey(text)
      .split(/\s+/)
      .filter((token) => token.length > 2);
    if (tokens.length < 12) return 0;

    const counts = new Map();
    for (const token of tokens) counts.set(token, (counts.get(token) || 0) + 1);
    const topCount = Math.max(...counts.values());
    const ratio = topCount / tokens.length;
    if (ratio < 0.28) return 0;
    return Math.min(90, Math.round((ratio - 0.25) * 220));
  }

  function isNameLike(text) {
    const cleaned = cleanText(text);
    if (!cleaned || cleaned.length < 2 || cleaned.length > 80) return false;
    if (isUiText(cleaned) || isMetricLike(cleaned) || TIME_OR_DATE_RE.test(cleaned)) return false;
    if (/https?:\/\//i.test(cleaned)) return false;
    if (/^[\d\s.,:/-]+$/.test(cleaned)) return false;

    const words = cleaned.split(/\s+/).filter(Boolean);
    if (words.length > 8) return false;
    if (/^@?[a-z0-9._-]{2,30}$/i.test(cleaned)) return true;
    if (/[A-ZÀ-Ỹ]/.test(cleaned[0]) && words.length <= 6) return true;
    if (words.length >= 2 && words.length <= 6 && words.every((word) => /[\p{L}\p{N}.&'-]/u.test(word))) return true;
    return false;
  }

  function looksLikeNameOnly(text) {
    const cleaned = cleanText(text);
    if (!isNameLike(cleaned)) return false;
    return !/[.!?。！？]/.test(cleaned) && cleaned.split(/\s+/).length <= 5;
  }

  function pruneEmpty(value) {
    if (Array.isArray(value)) return value.map(pruneEmpty).filter((item) => item !== undefined);
    if (!value || typeof value !== "object") return value;
    const output = {};
    for (const [key, child] of Object.entries(value)) {
      const pruned = pruneEmpty(child);
      if (pruned === "" || pruned == null) continue;
      if (Array.isArray(pruned) && pruned.length === 0) continue;
      if (typeof pruned === "object" && !Array.isArray(pruned) && Object.keys(pruned).length === 0) continue;
      output[key] = pruned;
    }
    return output;
  }

  function uniqueBy(items, keyFn) {
    const seen = new Set();
    const output = [];
    for (const item of items) {
      const key = keyFn(item);
      if (!key || seen.has(key)) continue;
      seen.add(key);
      output.push(item);
    }
    return output;
  }

  function escapeRegExp(value) {
    return String(value).replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  }

  function capitalize(value) {
    return String(value || "").charAt(0).toUpperCase() + String(value || "").slice(1);
  }

  return {
    VERSION,
    filterSocialHtml,
    extractSocialContent: filterSocialHtml,
    filterSocialHtmlToText,
    filterCurrentPage,
    filterCurrentPageToText,
    serializeCurrentPageHtml,
    serializeDomForFiltering,
    formatResultAsText,
    cleanText,
    normalizeUrl,
    normalizeOutputUrl,
    parseMetricValue,
    parseMetricsFromText,
    isPostUrl,
    isAccountUrl,
  };
});

if (typeof module === "object" && module.exports && typeof require === "function" && require.main === module) {
  let html = "";
  process.stdin.setEncoding("utf8");
  process.stdin.on("data", (chunk) => {
    html += chunk;
  });
  process.stdin.on("end", () => {
    const args = process.argv.slice(2);
    const jsonMode = args.includes("--json");
    const currentUrl = args.find((arg) => arg !== "--json") || "";
    const result = module.exports.filterSocialHtml(html, {
      currentUrl,
    });
    process.stdout.write(jsonMode ? `${JSON.stringify(result, null, 2)}\n` : `${result.text}\n`);
  });
}
