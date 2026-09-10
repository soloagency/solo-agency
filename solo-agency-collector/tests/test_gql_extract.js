// Offline harness for gql_extract.js. Runs the real file in a vm with a fake
// window.__soloGql, so the capability extractors can be exercised against synthetic
// GraphQL payloads without a browser.
//
// The shapes here are not invented: they mirror what a working third-party extension
// (2.13.15) reads out of the same GroupsCometFeedRegularStoriesPaginationQuery response.
const fs = require("fs");
const path = require("path");
const vm = require("vm");

const SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "platforms", "facebook", "gql_extract.js"), "utf8");

function makeCtx(captures) {
  const win = {};
  const ctx = {
    window: win,
    location: { href: "https://www.facebook.com/groups/000000000000000", origin: "https://www.facebook.com" },
    document: { querySelectorAll: () => [], querySelector: () => null, title: "", body: { innerText: "" } },
    setTimeout, clearTimeout, URL, console, MutationObserver: function () { this.observe = () => {}; this.disconnect = () => {}; },
  };
  ctx.globalThis = ctx;
  vm.createContext(ctx);
  vm.runInContext(SRC, ctx);
  win.__soloGql = { captures: captures || [], origFetch: null };
  return ctx;
}

// A NORMAL group story: the author sits in the top-level actors array.
function namedStory(postId, text, name) {
  return {
    node: {
      __typename: "Story",
      id: "S:" + postId,
      post_id: postId,
      permalink_url: "https://www.facebook.com/groups/000000000000000/permalink/" + postId + "/",
      actors: [{ id: "724699549", name: name, url: "https://www.facebook.com/binh" }],
      comet_sections: { content: { story: { message: { text: text } } } },
    },
  };
}

// An ANONYMOUS group story. Facebook scrubs the top-level actors — that scrubbing is what
// makes the UI say "Anonymous member" — but still ships the real actor, numeric id and all,
// inside the avatar renderer's own story, where it is NOT redacted.
function anonymousStory(postId, text) {
  return {
    node: {
      __typename: "Story",
      id: "S:" + postId,
      post_id: postId,
      permalink_url: "https://www.facebook.com/groups/000000000000000/permalink/" + postId + "/",
      comet_sections: {
        content: { story: { message: { text: text } } },
        context_layout: {
          story: {
            comet_sections: {
              actor_photo: {
                story: {
                  actors: [{
                    id: "378233344858797",
                    name: "Anonymous member",
                    profile_picture: { uri: "https://scontent.example/a.jpg" },
                  }],
                },
              },
            },
          },
        },
      },
    },
  };
}

function capture(edges) {
  return {
    queryName: "GroupsCometFeedRegularStoriesPaginationQuery",
    docId: "1234567890",
    fbDtsg: "NAcM-fake",
    variables: { id: "000000000000000", count: 3 },
    response: [{ data: { node: { group_feed: { edges: edges, page_info: { end_cursor: null, has_next_page: false } } } } }],
  };
}

let pass = 0, fail = 0;
function check(name, cond, detail) {
  if (cond) { pass++; console.log("  ok   " + name); }
  else { fail++; console.log("  FAIL " + name + (detail !== undefined ? "  -> " + JSON.stringify(detail) : "")); }
}

console.log("\n== fb.group.posts: anonymous posts must survive ==");
{
  const ctx = makeCtx([capture([
    namedStory("668676911902646", "Post 2: Post 2 Post 2", "Binh Nguyen"),
    anonymousStory("000000000000001", "Post 5 post 5 post 5"),
    namedStory("000000000000002", "Post 3: Post 3 Post 3", "Binh Nguyen"),
  ])]);
  const res = ctx.window.__soloGqlExtractCapability("fb.group.posts", {});
  check("all three stories are returned", res.count === 3, res.count);

  const anon = (res.items || []).find((i) => /Post 5/.test(i.text || ""));
  // This is the whole point: an anonymous post used to vanish, and a third-party extension
  // reading the same response got it — because it reads a path Facebook does not redact.
  check("the anonymous post is present at all", !!anon, (res.items || []).map((i) => i.id));
  check("its permalink survives", !!anon && /000000000000001/.test(anon.url), anon && anon.url);
  check("its real actor id is recovered from the avatar path", !!anon && anon.actor && anon.actor.id === "378233344858797", anon && anon.actor);
  check("its displayed name is kept as Facebook gives it", !!anon && anon.actor && anon.actor.name === "Anonymous member", anon && anon.actor);

  const named = (res.items || []).find((i) => /Post 2/.test(i.text || ""));
  check("a normal post still reads its top-level actor", !!named && named.actor && named.actor.id === "724699549", named && named.actor);
}

console.log("\n== a story keyed only by post_id ==");
{
  const edge = anonymousStory("999888777", "keyed by post_id only");
  delete edge.node.id; // some pagination replies carry no separate story id
  const ctx = makeCtx([capture([edge])]);
  const res = ctx.window.__soloGqlExtractCapability("fb.group.posts", {});
  check("it is not dropped for lacking a story id", res.count === 1, res.count);
  check("post_id becomes the record id", res.count === 1 && res.items[0].id === "999888777", res.items && res.items[0] && res.items[0].id);
}

console.log("\n== a node with neither id is still refused ==");
{
  const ctx = makeCtx([capture([{ node: { __typename: "Story", comet_sections: { content: { story: { message: { text: "junk" } } } } } }])]);
  const res = ctx.window.__soloGqlExtractCapability("fb.group.posts", {});
  check("an unidentifiable node produces no record", res.count === 0, res.count);
}

console.log("\n== streamed @stream / @defer chunks ==");
{
  // Measured live: Facebook answers this query with 5 chunks. Chunk 0 is a skeleton whose
  // only edge is a section header; each later chunk is an @stream payload addressing a slot
  // that does not exist yet ("node.group_feed.edges.1"), and the last is an @defer carrying
  // page_info. Walking the FULL path lands on undefined and drops the payload — which is how
  // the anonymous post and one more post went missing for a whole session.
  const header = { node: { __typename: "GroupsSectionHeaderUnit", title: { text: "Recent activity" } } };
  const streamed = [
    { data: { node: { group_feed: { edges: [header] } } } },
    { label: "…$stream$…", path: ["node", "group_feed", "edges", 1], data: anonymousStory("000000000000001", "Post 5 post 5 post 5") },
    { label: "…$stream$…", path: ["node", "group_feed", "edges", 2], data: namedStory("668676911902646", "Post 2: Post 2", "Binh Nguyen") },
    { label: "…$stream$…", path: ["node", "group_feed", "edges", 3], data: namedStory("674440161326321", "Post 4 Post 4", "Binh Nguyen") },
    { label: "…$defer$…", path: ["node", "group_feed"], data: { page_info: { end_cursor: "CUR", has_next_page: true } } },
  ];
  const ctx = makeCtx([{
    queryName: "GroupsCometFeedRegularStoriesPaginationQuery",
    docId: "1", fbDtsg: "x", variables: { id: "000000000000000" }, response: streamed,
  }]);
  const res = ctx.window.__soloGqlExtractCapability("fb.group.posts", {});
  check("stream chunks are merged, not discarded", res.stream_chunks_merged === 4, res.stream_chunks_merged);
  check("all three streamed posts are recovered", res.count === 3, res.count);
  const ids = (res.items || []).map((i) => i.id).sort();
  check("the anonymous post survives the merge", ids.indexOf("S:000000000000001") > -1, ids);
  check("Post 4 survives the merge", ids.indexOf("S:674440161326321") > -1, ids);
  check("the section header is not mistaken for a post", !(res.items || []).some((i) => /Recent activity/.test(i.text || "")), res.items && res.items.map((i) => i.text.slice(0, 20)));
}

console.log("\n== time window ==");
{
  const now = Math.floor(Date.now() / 1000), DAY = 86400;
  const dated = (id, text, ago) => {
    const e = namedStory(id, text, "Binh Nguyen");
    e.node.creation_time = now - ago * DAY;
    return e;
  };
  const undated = namedStory("444", "no timestamp", "Binh Nguyen");

  // No window -> nothing is filtered.
  let ctx = makeCtx([capture([dated("1", "recent", 10), dated("2", "old", 400), undated])]);
  let res = ctx.window.__soloGqlExtractCapability("fb.group.posts", {});
  check("no window keeps everything", res.count === 3 && !res.time_window, res.count);

  // within_days keeps only what falls inside it.
  ctx = makeCtx([capture([dated("1", "recent", 10), dated("2", "old", 400), dated("3", "edge", 80)])]);
  res = ctx.window.__soloGqlExtractCapability("fb.group.posts", { within_days: 90 });
  check("within_days keeps only the in-range posts", res.count === 2, (res.items || []).map((i) => i.text));
  check("it reports what it dropped", res.time_window.excluded_older === 1, res.time_window);

  // An undated post cannot be judged. Excluding it silently would hide real posts; keeping it
  // silently would let a five-year-old post into a "last 90 days" run. So: excluded, COUNTED.
  ctx = makeCtx([capture([dated("1", "recent", 10), undated])]);
  res = ctx.window.__soloGqlExtractCapability("fb.group.posts", { within_days: 90 });
  check("an undated post is excluded by default", res.count === 1, res.count);
  check("but it is counted, never silent", res.time_window.undated === 1, res.time_window);

  ctx = makeCtx([capture([dated("1", "recent", 10), undated])]);
  res = ctx.window.__soloGqlExtractCapability("fb.group.posts", { within_days: 90, include_undated: true });
  check("include_undated keeps it on request", res.count === 2 && res.time_window.undated_kept === true, res.time_window);

  // Explicit since/until, ISO or unix.
  ctx = makeCtx([capture([dated("1", "recent", 5), dated("2", "older", 200)])]);
  res = ctx.window.__soloGqlExtractCapability("fb.group.posts", { since: new Date((now - 30 * DAY) * 1000).toISOString() });
  check("an ISO `since` works", res.count === 1, res.count);

  ctx = makeCtx([capture([dated("1", "recent", 5), dated("2", "older", 200)])]);
  res = ctx.window.__soloGqlExtractCapability("fb.group.posts", { until: now - 100 * DAY });
  check("`until` drops what is too new", res.count === 1 && res.time_window.excluded_newer === 1, res.time_window);
}

console.log("\n== duplicates collapse ==");
{
  const ctx = makeCtx([capture([
    anonymousStory("000000000000001", "Post 5 post 5 post 5"),
    anonymousStory("000000000000001", "Post 5 post 5 post 5"),
  ])]);
  const res = ctx.window.__soloGqlExtractCapability("fb.group.posts", {});
  check("the same story twice yields one record", res.count === 1, res.count);
}

console.log("\n== a capability that found nothing still returns a RECORD ==");
{
  // A live run of every read capability in hidden tabs produced records:null on seven of them,
  // all because a hidden tab never renders a feed and so no capture existed. "Never looked",
  // "nothing there" and "the code threw" were the same output. They need opposite fixes.
  const ctx = makeCtx([]);                       // no captures at all
  const res = ctx.window.__soloGqlExtractCapability("fb.group.posts", {});
  check("the record exists", !!res, res);
  check("available says the capability RAN", res.available === true, res.available);
  check("found says it got nothing", res.found === false, res.found);
  check("and the reason is nameable", res.reason === "no_capture", res.reason);

  // With captures present, an unknown id reaches the extractor lookup and must name THAT — the
  // no-capture check runs first, so this needs a populated store to be a real test.
  const ctx2 = makeCtx([capture([namedStory("1", "x", "N")])]);
  const unknown = ctx2.window.__soloGqlExtractCapability("fb.not.a.capability", {});
  check("an unknown id is reported, not nulled", unknown.available === true && unknown.reason === "no_extractor", unknown.reason);
}

// ============================================================================================
// fb.groups.search — privacy / member_count / snippet / viewer_join_state (2026-09-09).
// extractGroupsSearch() reads the SearchComet SERP envelope
// (data.serpResponse.results.edges[]), keeps only edges edgeIsGroup() (gql_extract.js:683-692)
// recognizes as a Group, and parses privacy/member_count out of the descriptor line via
// parseGroupSnippet(text, joinState) (gql_extract.js:735-761). NOTE on the real algorithm
// (verified by running the actual source, not assumed): parseGroupSnippet takes viewer_join_state
// as a SECOND argument and consults it FIRST — CAN_JOIN/CAN_REQUEST resolve privacy before the
// snippet text is even looked at, and only when join_state gives no signal does it fall back to
// the multi-language word list. It also returns two more fields than a plain {privacy,
// member_count} — privacy_source ("join_state"|"snippet"|"") and member_count_text (the raw
// winning segment) — which fb_normalize.js's normalizeGroupSearchItem carries into `ext`. Every
// edge below sets viewer_join_state to "" so the snippet text is what actually drives the parse
// (matching what is being asserted), with one dedicated case proving the join_state shortcut.
function groupEdge(id, name, url, snippetPrimary, snippetFacepile, joinState) {
  var vm = {
    __typename: "SearchRenderable_group",
    profile: { __typename: "Group", id: id, name: name, url: url, viewer_join_state: joinState },
  };
  if (snippetPrimary !== undefined) vm.primary_snippet_text_with_entities = { text: snippetPrimary };
  if (snippetFacepile !== undefined) vm.snippet_with_facepile = { simple_text_with_entities: { text: snippetFacepile } };
  // node.role: "ENTITY_GROUPS" is one of the independent signals edgeIsGroup() accepts
  // (gql_extract.js:687); profile.__typename:"Group" above is another. Both present here,
  // matching a real SERP group card.
  return { node: { role: "ENTITY_GROUPS" }, rendering_strategy: { view_model: vm } };
}
function groupsSearchCapture(edges) {
  return {
    queryName: "SearchCometResultsPaginatedResultsQuery",
    docId: "doc_search_groups",
    fbDtsg: "NAcM-fake",
    variables: { query: "nail salon", count: 10 },
    response: [{ data: { serpResponse: { results: { edges: edges, page_info: { end_cursor: "c1", has_next_page: false } } } } }],
  };
}

console.log("\n== fb.groups.search: privacy/member_count/snippet/viewer_join_state ==");
{
  const edges = [
    groupEdge("g1", "Group One", "https://www.facebook.com/groups/g1/", "Public · 1,234 members", undefined, ""),
    groupEdge("g2", "Group Two", "https://www.facebook.com/groups/g2/", "Riêng tư · 1,2K thành viên", undefined, ""),
    groupEdge("g3", "Group Three", "https://www.facebook.com/groups/g3/", "Nhóm Công khai · 12 N thành viên", undefined, ""),
    groupEdge("g4", "Group Four", "https://www.facebook.com/groups/g4/", "Public · Very active community", undefined, ""),
    groupEdge("g5", "Group Five", "https://www.facebook.com/groups/g5/", "12,000 members", undefined, ""),
    groupEdge("g6", "Group Six", "https://www.facebook.com/groups/g6/", undefined, "Private · 500 members", ""),
    groupEdge("g7", "Group Seven", "https://www.facebook.com/groups/g7/", undefined, undefined, "CAN_JOIN"),
  ];
  const ctx = makeCtx([groupsSearchCapture(edges)]);
  const res = ctx.window.__soloGqlExtractCapability("fb.groups.search", {});

  check("capability id", res.capability === "fb.groups.search", res.capability);
  check("schema is GroupSummary[]", res.schema === "GroupSummary[]", res.schema);
  check("all 7 group edges extracted", res.count === 7, res.count);

  const byId = {};
  (res.items || []).forEach((i) => { byId[i.id] = i; });

  check("'Public · 1,234 members' -> public, 1234", byId.g1 && byId.g1.privacy === "public" && byId.g1.member_count === 1234, byId.g1);
  check("g1 snippet passthrough (verbatim, just whitespace-normalized)", byId.g1 && byId.g1.snippet === "Public · 1,234 members", byId.g1 && byId.g1.snippet);

  check("VN 'Riêng tư · 1,2K thành viên' -> private, 1200", byId.g2 && byId.g2.privacy === "private" && byId.g2.member_count === 1200, byId.g2);

  check("VN 'Nhóm Công khai · 12 N thành viên' -> public, 12000 (N is the vi thousands suffix)", byId.g3 && byId.g3.privacy === "public" && byId.g3.member_count === 12000, byId.g3);

  check("no numeric segment in the descriptor -> member_count null", byId.g4 && byId.g4.member_count === null, byId.g4);
  check("privacy is still read off the snippet even with no member count", byId.g4 && byId.g4.privacy === "public", byId.g4);

  check("a members phrase with no public/private word -> privacy unknown ('')", byId.g5 && byId.g5.privacy === "", byId.g5 && byId.g5.privacy);
  check("member_count still parses from that same unknown-privacy line", byId.g5 && byId.g5.member_count === 12000, byId.g5);

  check("primary snippet absent -> falls back to the facepile snippet", byId.g6 && byId.g6.snippet === "Private · 500 members", byId.g6 && byId.g6.snippet);
  check("privacy/member_count are parsed FROM the fallback facepile text", byId.g6 && byId.g6.privacy === "private" && byId.g6.member_count === 500, byId.g6);

  check("viewer_join_state passthrough on every item", byId.g1.viewer_join_state === "" && byId.g7.viewer_join_state === "CAN_JOIN", [byId.g1.viewer_join_state, byId.g7.viewer_join_state]);
  check("no snippet text at all -> privacy can still resolve from viewer_join_state alone", byId.g7 && byId.g7.privacy === "public" && byId.g7.member_count === null, byId.g7);
}

// ============================================================================================
// fb.search.posts — new capability id, registered as an alias of extractGroupSearchPosts that
// rewrites result.capability (gql_extract.js:768-772). Verified against the real source: it
// reuses postRecordFromStoryNode(), so the minimal story fixture is copied from the SAME shape
// namedStory()/anonymousStory() above already use for fb.group.posts — id, post_id, actors,
// message.text, creation_time, permalink_url, feedback.id — just addressed via
// rendering_strategy.view_model.click_model.story instead of edges[].node (extractGroupSearchPosts,
// gql_extract.js:571-599). A profile/page edge (view_model.profile, no click_model.story) proves
// the story filter skips non-post results, exactly as it does for fb.group.search_posts.
function postStoryEdge(id, text) {
  return {
    rendering_strategy: {
      view_model: {
        click_model: {
          story: {
            id: "S:" + id,
            post_id: id,
            permalink_url: "https://www.facebook.com/groups/000000000000000/posts/" + id + "/",
            actors: [{ id: "500", name: "Search Result Author", url: "https://www.facebook.com/searchauthor" }],
            message: { text: text },
            creation_time: 1700000000,
            feedback: { id: "fb:" + id },
          },
        },
      },
    },
  };
}
function nonPostEdge(id, name) {
  // A person/page SERP result: entity under view_model.profile, no click_model.story at all.
  return { rendering_strategy: { view_model: { profile: { __typename: "User", id: id, name: name, url: "https://www.facebook.com/" + name } } } };
}
function searchPostsCapture(edges) {
  return {
    queryName: "SearchCometResultsPaginatedResultsQuery",
    docId: "doc_search_posts",
    fbDtsg: "NAcM-fake",
    variables: { query: "nail salon", count: 10 },
    response: [{ data: { serpResponse: { results: { edges: edges, page_info: { end_cursor: "c1", has_next_page: false } } } } }],
  };
}

console.log("\n== fb.search.posts: alias of fb.group.search_posts on the global SearchComet SERP ==");
{
  const caps = [searchPostsCapture([
    postStoryEdge("777", "Looking for a great nail salon near downtown"),
    nonPostEdge("900", "Some Person"),
  ])];

  const ctxA = makeCtx(caps);
  const resSearchPosts = ctxA.window.__soloGqlExtractCapability("fb.search.posts", {});
  const ctxB = makeCtx(caps);
  const resGroupSearchPosts = ctxB.window.__soloGqlExtractCapability("fb.group.search_posts", {});

  check("fb.search.posts reports its own capability id", resSearchPosts.capability === "fb.search.posts", resSearchPosts.capability);
  check("schema is PostRecord[]", resSearchPosts.schema === "PostRecord[]", resSearchPosts.schema);
  check("only the post edge is extracted; the profile/page edge is skipped", resSearchPosts.count === 1, resSearchPosts.count);

  check("fb.group.search_posts still reports its own capability id (alias didn't leak)", resGroupSearchPosts.capability === "fb.group.search_posts", resGroupSearchPosts.capability);
  check(
    "fb.search.posts yields the SAME items as fb.group.search_posts on the same captures",
    JSON.stringify(resSearchPosts.items) === JSON.stringify(resGroupSearchPosts.items),
    { a: resSearchPosts.items, b: resGroupSearchPosts.items }
  );
  check("the one item carries the story's feedback_id and text", resSearchPosts.items[0] && resSearchPosts.items[0].feedback_id === "fb:777" && /nail salon/.test(resSearchPosts.items[0].text), resSearchPosts.items[0]);
}

console.log("\n== CAPABILITY_PAGINATION registers fb.search.posts (not exported; pinned via a source regex) ==");
{
  // CAPABILITY_PAGINATION is a module-private `var`, never attached to window — the same
  // constraint test_platform_registry.js works around for background.js's tables. Pin the literal
  // table entry in the source text instead of trying to reach the variable at runtime.
  const m = SRC.match(/var\s+CAPABILITY_PAGINATION\s*=\s*\{([\s\S]*?)\};/);
  check("CAPABILITY_PAGINATION table found in source", !!m);
  const body = m ? m[1] : "";
  check(
    "fb.search.posts is registered with scope SearchComet and the SearchComet page_info path",
    /"fb\.search\.posts"\s*:\s*\{\s*scope:\s*"SearchComet"\s*,\s*pageInfoPath:\s*"data\.serpResponse\.results\.page_info"\s*\}/.test(body),
    body
  );
}

console.log("\n" + (fail === 0 ? "ALL " + pass + " CHECKS PASSED" : pass + " passed, " + fail + " FAILED"));
process.exit(fail === 0 ? 0 : 1);
