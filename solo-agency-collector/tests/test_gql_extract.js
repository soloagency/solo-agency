// Offline harness for gql_extract.js. Runs the real file in a vm with a fake
// window.__soloGql, so the capability extractors can be exercised against synthetic
// GraphQL payloads without a browser.
//
// The shapes here are not invented: they mirror what a working third-party extension
// (2.13.15) reads out of the same GroupsCometFeedRegularStoriesPaginationQuery response.
const fs = require("fs");
const path = require("path");
const vm = require("vm");

const SRC = fs.readFileSync(path.join(__dirname, "..", "chrome-extension", "gql_extract.js"), "utf8");

function makeCtx(captures) {
  const win = {};
  const ctx = {
    window: win,
    location: { href: "https://www.facebook.com/groups/668676178569386", origin: "https://www.facebook.com" },
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
      permalink_url: "https://www.facebook.com/groups/668676178569386/permalink/" + postId + "/",
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
      permalink_url: "https://www.facebook.com/groups/668676178569386/permalink/" + postId + "/",
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
    variables: { id: "668676178569386", count: 3 },
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
    anonymousStory("676092881161049", "Post 5 post 5 post 5"),
    namedStory("669851668451837", "Post 3: Post 3 Post 3", "Binh Nguyen"),
  ])]);
  const res = ctx.window.__soloGqlExtractCapability("fb.group.posts", {});
  check("all three stories are returned", res.count === 3, res.count);

  const anon = (res.items || []).find((i) => /Post 5/.test(i.text || ""));
  // This is the whole point: an anonymous post used to vanish, and a third-party extension
  // reading the same response got it — because it reads a path Facebook does not redact.
  check("the anonymous post is present at all", !!anon, (res.items || []).map((i) => i.id));
  check("its permalink survives", !!anon && /676092881161049/.test(anon.url), anon && anon.url);
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
    { label: "…$stream$…", path: ["node", "group_feed", "edges", 1], data: anonymousStory("676092881161049", "Post 5 post 5 post 5") },
    { label: "…$stream$…", path: ["node", "group_feed", "edges", 2], data: namedStory("668676911902646", "Post 2: Post 2", "Binh Nguyen") },
    { label: "…$stream$…", path: ["node", "group_feed", "edges", 3], data: namedStory("674440161326321", "Post 4 Post 4", "Binh Nguyen") },
    { label: "…$defer$…", path: ["node", "group_feed"], data: { page_info: { end_cursor: "CUR", has_next_page: true } } },
  ];
  const ctx = makeCtx([{
    queryName: "GroupsCometFeedRegularStoriesPaginationQuery",
    docId: "1", fbDtsg: "x", variables: { id: "668676178569386" }, response: streamed,
  }]);
  const res = ctx.window.__soloGqlExtractCapability("fb.group.posts", {});
  check("stream chunks are merged, not discarded", res.stream_chunks_merged === 4, res.stream_chunks_merged);
  check("all three streamed posts are recovered", res.count === 3, res.count);
  const ids = (res.items || []).map((i) => i.id).sort();
  check("the anonymous post survives the merge", ids.indexOf("S:676092881161049") > -1, ids);
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
    anonymousStory("676092881161049", "Post 5 post 5 post 5"),
    anonymousStory("676092881161049", "Post 5 post 5 post 5"),
  ])]);
  const res = ctx.window.__soloGqlExtractCapability("fb.group.posts", {});
  check("the same story twice yields one record", res.count === 1, res.count);
}

console.log("\n" + (fail === 0 ? "ALL " + pass + " CHECKS PASSED" : pass + " passed, " + fail + " FAILED"));
process.exit(fail === 0 ? 0 : 1);
