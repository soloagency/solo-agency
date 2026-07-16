# Research first — and harvest real visuals (Stage 2, the picked script)

Load this module during Stage 2 Step 5 (`method.md`) when you start vetting inline images. It carries the R-ladder (where to find candidate images) plus the inline-media format rules.

The facts research happens once up front (`method.md` Step 2). The **visual** harvest happens in Stage 2 for the picked script, under the step-5 image budget: **1–3 inline images, no two on consecutive scenes, the rest auto-B-roll.** A **specific, vetted inline URL beats auto-B-roll when a beat names a specific thing** (a named product, place, person, dish, event, artwork, chart) — so spend your ≤3 images on the beats that most need a real photo. This ladder is **where to find** the candidate; `method.md` step 5 is **how to vet it** before inlining (download + look, retry 5, generate → upload, `widecast_create_image` last). Do NOT use `widecast_search_broll` here — it is for editing an existing video, not authoring.

---

## R1 · Facts

Done in `method.md` step 2; reuse them. Same Credibility bar as any stat (see `SKILL.md`).

## R2 · Hunt real images — only for the ≤3 beats that most need a real photo

Rank candidates on **Authority** (does the source have standing to depict this subject? — apple.com for an iPhone, a museum for an artwork, AP/Reuters for a news event) and **Stability** (will the URL resolve when WideCast fetches it? — direct CDN file links beat page links). **Descend the ladder ONLY after exhausting each rung:**

- **Rung 1 · Open-license / encyclopedic** — Wikimedia Commons, Wikipedia. The image page's **"Original file"** → stable `upload.wikimedia.org/.../<name>.jpg`, rights-clear, rarely hotlink-protected.
- **Rung 2 · The subject's OWN authoritative source** — product → manufacturer page (apple.com / sony.com); place → tourism board / NPS; business → its own site; company → press kit / newsroom; artwork → the holding museum; film/TV/book → studio / publisher press kit; public figure → official site; science / medical → NASA / NIH / CDC / ESA; recipe → a reputable cooking publication.
- **Rung 3 · Reputable secondary sources — where MOST event/news + review images live.** `web_fetch` a top article (news → AP / Reuters / BBC / NYT; product → The Verge / Wirecutter / Rtings / DPReview; recipe → NYT Cooking / Serious Eats; travel → tourism boards). In the fetched HTML, scan `<meta property="og:image">`, `<meta name="twitter:image">`, and inline `<img src>` in the article body. **A URL you pulled from a fetched page's HTML is verified-real — you literally saw it in the document; this is NOT fabrication.** (Search *snippets* don't return full HTML — `web_fetch` the page.)
- **Rung 4 · Specialty / permissive sources** — Unsplash, Pexels, Pixabay, Flickr CC, NASA Image Library, Library of Congress, USDA, GitHub (software screenshots), OpenStreetMap.
- **Rung 5 · Can't source a real one for a beat you chose to illustrate?** Escalate per `method.md` step 5: generate the image yourself → `widecast_upload_asset`, or (last, 1 credit) `widecast_create_image`. For beats you did NOT pick for an image, auto-B-roll is the normal, fine outcome — don't force a shaky link.

**Skip a URL, not a rung.** Drop a *specific* candidate if it's a tiny thumbnail, a tracking pixel, paywalled, a logo/ad, or the wrong subject — but try another candidate from the SAME rung before descending.

## R3 · Verify it's a DIRECT FILE link

The URL must end in an allowed extension — images **`.jpg`/`.jpeg`/`.png` only** (no `.gif`/`.webp`/`.svg`/`.bmp`/`.avif`/`.ico`/`.tiff` — they don't render reliably), videos `.mp4`/`.webm`/`.mov`/`.m4v`/`.avi`. A `?query` suffix is fine. An article page or "view image" page **won't work**.

- **Verified ✓** — the URL came from a `web_fetch` body, a search result, the user's message, or a known-stable pattern (an `upload.wikimedia.org` file you saw listed).
- **Fabrication ✗** — guessing a URL from a naming convention without ever seeing it return content. NEVER do this — a dead link loses the scene, worse than B-roll.
- **Authority-match test** — *"if a viewer asked 'where's this image from?', would the answer sound authoritative for the subject?"* (recipe → NYT Cooking ✓ / random Pinterest ✗; iPhone → apple.com ✓ / AliExpress ✗).

## R4 · Place each verified URL right after the sentence it illustrates

Then hand the picked script to WideCast **verbatim** (URLs included). Beats without a real URL still get good auto-B-roll — coverage > forcing a shaky link.

## R5 · Unsure where an image belongs? Use `media_pool`

When you have a real, relevant image but aren't confident which beat it fits, **don't force it inline** — pass it in the `create_video` `media_pool` array (a list of direct image/video URLs). WideCast downloads each, makes a thumbnail, and adds them to the scene editor's library so the user can drop any into any scene. **Inline the URLs you're confident about; put the maybes/extras in `media_pool`.** Same rules — direct file links only, never fabricate.

---

## Adding images & video (inline media URLs) — format rules

- **Prefer markdown image syntax** `![brief scene description](https://…)` — chat hosts render the image inline so the user can SEE whether the visual fits before approving; the alt text is also the strongest anchor for the scene matcher. WideCast strips the entire `![…](…)` construct from the spoken narration.
- Raw URLs on their own line still work for backward compat. Mix forms freely.
- **Direct file links only**: images `.jpg`/`.jpeg`/`.png` ONLY, video `.mp4`/`.webm`/`.mov`/`.m4v`/`.avi` (a `?query` suffix is fine).
- A YouTube/TikTok **page** link won't work as inline media. If the user wants a whole clip turned into a video, that's `source="video_url"`, not an inline URL.

Example (markdown form, recommended):

```
Our cold brew steeps for 16 hours. ![Slow extraction makes it smooth](https://cdn.acme.com/coldbrew.jpg) That mellow finish is what people come back for.
```
