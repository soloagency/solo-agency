# 10 · Data model, coordinates, modify_scene branches, layout & screenshots

_Version: `modular-1.1` · module of the AI Video Editor Playbook (`SKILL.md`)._

> **Module of the AI Video Editor Playbook.** Master index + checklist + critical rules live in `SKILL.md`. **Load this when:** you start a scene, read its data, audit/adjust layout (narrator+overlay positions), or decide how to look.
> Cross-refs: background→`20_background.md`; overlay authoring→`30_overlay_core.md`.

---

### 0.1. The data-first tools (cheap, no render, prefer these)

1. **`video_data`** (`/v1/video_data`): pull the entire video + every segment. Always call this first.
2. **`scene_geometry`** (`/v1/scene_geometry`): **this is the tool to audit layout using data** — synchronous, free, read-only, no render. It is a measuring tool, **not visual proof**: it can tell you where boxes are, but it cannot tell you whether the scene looks good. Pass the scene's `voice_file`, get back:
   - `coordinate_space` = `{width: 280, height: 498, unit: editor_preview_px}` (see §0.3).
   - `safe_zones`: `dead_top` (top 10%), `dead_bottom` (bottom 25%), `safe_rect` (the safe middle region) — precomputed.
   - `boxes.narrator` (`rect`, `visible`, `face`, `face_center`), `boxes.caption` (`container_rect`, `text_rect_estimate`), `boxes.remotion` (objects + `object_layer` with `layout_id`, `rect` preview and `rect_canvas` 720×1280).
   - The **action-mapping** in `object_layer`: each object exposes its `layout_id` + the `update_field` to pass back to `modify_scene` — i.e. it tells you EXACTLY how to move/resize that object. This is the part that lets a no-vision agent act on the layout.
   - Note: the pre-scored `violations`/`warnings` arrays and `remotion_poster_state` were **removed** (they biased agents toward trusting blind mechanical verdicts). Reason over the raw boxes + safe_zones yourself; for any **visual/aesthetic** judgment use a **screenshot**, not the geometry (see §6).
3. **`modify_scene`** (`/v1/modify_scene`): the write tool (see §0.4).

> The canonical selector for both `scene_geometry` and `modify_scene` is **`voice_file`** (the scene's stable UID, also the root of `{voice_file}_spec.json`). DO NOT use `segment.id` as the selector — `id` is only the display index, and it changes on reorder/add/delete.

### 0.2. Field dictionary for a segment

**Role & media (A-roll / B-roll):**
- `faceless` (bool, **VIDEO level**, not segment) — `true` = video has NO narrator (every scene is B-roll, **no face to avoid**); `false` = video has a narrator. This is the global switch that decides how text is placed (see §3 and §5.2). ⚠ **`video_data` (slim) does NOT return the `faceless` field** — **infer it**: if NO scene has `show_narrator=true` (≡ `active_roll="A"`) then the video is **faceless**; if at least one A-roll exists, it is a video with a narrator.
- `show_narrator` (bool) — `true` = A-roll scene (narrator face shown); `false` = B-roll (visual replaces the narrator). When in doubt, the segmenter defaults to `true`. (In a `faceless` video this is always `false`.)
- `active_roll` (`"A"` | `"B"`) — runtime flag equivalent to `show_narrator`.
- `arollUrl`, `arollThumbnailUrl` — the narrator (A-roll) video and its thumbnail. **If it still points to a library placeholder image (e.g. `statics/aroll_male.png`, `mediaType=image`) → the A-roll scene has NOT BEEN RECORDED YET** (the "YOUR FACE & VOICE HERE" placeholder); remind the user to complete it — see §3.
- `brollUrl`, `brollThumbnailUrl` — the background (B-roll) video and its thumbnail.
- `mediaUrl`, `mediaType` (`video`|`image`) — the scene's currently active media.
- `thumbnailUrl` — the thumbnail of the active media.
- `originalMediaUrl`, `originalThumbnailUrl` — **the built-in baseline** (original clip/thumbnail) — this is the emergency restore point; you don't need to save it by hand.

**Content & semantics:**
- `text` — the scene's spoken line (original caption).
- `talking_point` — a condensed version of the main ideas joined by ` -> `.
- `quote` — the figure/punch phrase (always contains a number if the scene has a number).
- `keyword` — 3 English keywords for finding B-roll, ordered by priority `primary, fallback1, fallback2`.
- `visual` — **English image prompt** describing the overlay (empty if `pattern` is `typography_only`/`narration_only`; it is a **search query** if `pattern=real_entity`).
- `pattern` — 1 of the **15 canonical values** (see table §4.1).
- `sub_mode` — mandatory when `pattern=illustration`: `photo_with_people` | `photo_no_people` | `document` | `digital_ui` (empty for other patterns). **This is the field that decides grid-vs-footage for illustration.**
- `type` — `HOOK` | `STAT` | `KEY POINT` | `DATA` | `FACT` | `CALL TO ACTION` | `thumbnail`.
- `language`, `sfx_type`, `sfx_trigger`.

**Geometry (all in the 280×498 preview space — see §0.3):**
- `narrator_face` `{x,y,w,h}` — the face box in the **source** space `narrator_media_280x498`. **You must NOT edit `narrator_face` to adjust layout** (it is source data; `scene_geometry.boxes.narrator.face` is the displayed face box, derived from `overlay.narrator.rect`).
- `overlay` — a dict of sub-objects: `narrator`, `caption`, `text`, `image`, `zoom`, `avatar`. Each sub-object has `{x,y,w,h,visible,config}`. "The scene has overlay X" = `overlay.X.visible == true`.

**Spec & render:**
- `remotion_spec` — the overlay spec file currently in use (e.g. `JMXmX_spec.json?v=...`). The value `"none"` = the user INTENTIONALLY disabled the overlay; in that case every layout edit returns `remotion_spec_disabled`, and you must NOT re-enable it on your own.
- `chroma_bg` (`cobalt` | `amber`), `chroma_bg_locked` (bool) — the scene's key color for the **AUTO** pipeline only; you author overlays as **transparent SVG** (§5), so you never set or depend on it.
- `stock_fallback_reason` — the reason the editor forced grid (e.g. `illustration_document_force_grid`, `illustration_digital_ui_force_grid`). If it has a value = the editor intentionally kept grid, don't force footage.
- `use_stock_video` (bool) — the flag that permits stock B-roll.
- `thumbnailCustomized` (bool) — the auto-vs-manual gate for the thumbnail; only the thumbnail scene cares about this.

### 0.3. Coordinate system — two spaces, the SAME scale (×2.571)

There are two spaces, but they are **not independent** — they are just the same frame at two resolutions, converted by a single uniform scale:

- **PREVIEW space = `editor_preview_px` 280 (wide) × 498 (tall)** — used for the design/preview screen and for layout edits (`overlay.narrator.rect`, `overlay.caption.y`, `remotion.object.rect`). Every `rect` from `scene_geometry` (except fields with the `_canvas` suffix) is in this space.
- **OUTPUT/CANVAS space = `720 (wide) × 1280 (tall)`** — this is the **real video when rendered**, and also the **size of the overlay image the agent creates to generate the spec (§5)**.
- **Conversion: multiply by ~2.571** (`sx = 720/280 ≈ 2.571`, `sy = 1280/498 ≈ 2.571`, uniform). The server applies this exact scale automatically; in `object_layer` each object has both `rect` (preview) and `rect_canvas` (720×1280) — these two values always differ by exactly ×2.571.

The safe zone is **the same percentage** in both spaces — only the absolute numbers differ:

| Region | Preview space (280×498) | Output space (720×1280) | Meaning |
|---|---|---|---|
| `dead_top` (top 10%) | `y 0 → 49.8` | `y 0 → 128` | status bar/notch — the main object is NOT here |
| `safe_rect` (middle) | `y 49.8 → 373.5` | `y 128 → 960` | region for placing main content |
| `dead_bottom` (bottom 25%) | `y 373.5 → 498` | `y 960 → 1280` | caption + gesture area — the main object is NOT here |

> The key consequence for image creation (§5): when the agent authors a 720×1280 overlay (SVG), the server reads it at 720 wide and then **keeps each object's position** and scales it ÷2.571 into the preview space. That means **wherever you draw an object in the image, the overlay lands at the same HORIZONTAL position in the layout** — placing main content right at the top/bottom edge of the image makes the overlay fall into the dead zone. So the image itself must respect the 720×1280 safe band (leave **128px at the top** and **320px at the bottom**). ⚠ Caveat: this holds exactly for the **horizontal** axis; **vertically** the server auto-fits a decomposed `upload_overlay` group into the safe zone (it places/centers the group inside canvas y 128→960) — see the position note in §5.1.


### 0.4. `modify_scene` — 13 branches, each call picks EXACTLY ONE field family

| Branch | Task | Main field_name | Notes |
|---|---|---|---|
| (A) Background swap | Swap the background media | `mediaUrl` (+`mediaType`) | Roll-aware: B-roll writes to `mediaUrl`/`brollUrl`; A-roll writes `brollUrl`/user_asset without touching the narrator. **This is how you apply a background clip/image.** |
| (B) **Upload Overlay** | Load the **SVG** the agent authored internally → server converts to the spec | `remotion.upload_overlay` | **FREE.** Save the authored source locally. Show a pre-upload **overlay preview** only if the runtime can display it directly or convert it through an already-available lightweight path; otherwise skip preview. After upload, verify composition with the composite screenshot and verify rendered overlay text by calling `widecast_scene_inspector` with `action="overlay_poster"` when visible text exists. Do not install tools, launch headless browsers, or probe conversion paths just for preview. User-facing language is "overlay", not SVG. Canvas 720×1280, **transparent** — no background. The server (`svg2spec`) maps each `<g data-wc-object>` group → one spec object, renders text with the project's fonts, and bakes the poster. You do **not** upload/rasterize JPG for the overlay. Author per §5; stick to the scene context. |
| (C) Object-layer rect | Move/resize an overlay object (preferred for a few visible objects) | `remotion.object.rect` | Read `boxes.remotion.object_layer.objects` from `scene_geometry` first; pass a `layout_id` of the form `*.one_by_one` to adjust the whole timed group. **You can only MOVE/arrange; you CANNOT enlarge one_by_one text** (the image is pre-rendered, contain-fit, already near-full-width → upscale capped at ~+2%). To change text size/color/font → re-author the overlay SVG (§5.2). `layout.batch` accepts up to **12** `remotion.object.rect` edits at once. |
| (D) Group rect | Move/resize the whole Storyboard group | `remotion.group.rect` | Use move-only `x/y` when you need to translate the entire overlay together (for example: title in `dead_top`, whole overlay needs +28px down, or the overlay decomposed into >12 objects). If translating fixes one edge but pushes the opposite edge into `dead_top`/`dead_bottom`, use whole-group resize (`w/h`, `resize_mode:"scale_children"`) to fit the group into `safe_rect`. After resize, pull/show a screenshot and re-run typography gates; rebuild the overlay only if the resize makes title/body/secondary text too small, muddy, cramped, or unreadable. Prefer (C) for targeted per-object adjustments; prefer (D) for whole-group translation/resize. |
| (E) Narrator rect | Move/resize the narrator box | `overlay.narrator.rect` / `overlay.narrator.x|y|w|h` | 280×498 space. Keep `narrator_face` unchanged; set `overlay.narrator.touched=true` so auto-fit doesn't override. |
| (F) Caption Y | Move the caption vertically only | `overlay.caption.y` | Y ONLY — don't change x/w/h/visible/config/text. |
| (G) Layout batch | Combine multiple layout edits (1 persist + 1 MQTT) | `layout.batch` | Allowed children: `overlay.narrator.*`, `overlay.caption.y`, `remotion.object.rect`, `remotion.group.rect`. Keep `remotion.object.rect` children to **12 or fewer** per call; for >12-object whole-overlay translation/resize, use one `remotion.group.rect`. |
| (H) Upload Voice | User audio (ASYNC, queued) | — | Free; wait for MQTT `scene_voice_upload_applied` then pull again. |
| (I) Upload Narrator Video | User A-roll (ASYNC, queued) | `narrator.upload_video` | Free. |
| (J) A/B-roll switch | Turn the narrator A-roll on/off | — | Sync; keeps both lanes, swaps the active field. |
| (K) Segment text correction | Fix the text while keeping audio timing | metadata `text` | Rebuilds word-timing, doesn't change duration. |
| (L) Scene metadata | Edit `pattern` + `quote` | metadata | `pattern` is validated against the 15 canonical values; `type ∈ {HOOK,STAT,KEY POINT,DATA,FACT,CALL TO ACTION}`; setting `pattern=typography_only` will auto-clear `visual`. |
| (M) **Add element** (add-only) | **Append** ONE text/stat/label/callout to the EXISTING spec **without overwriting it** | `remotion.add_element` | **FREE, sync, additive — does NOT touch the existing objects.** `value={kind, value?, label?, url?, position?, rect?, style_token?, emphasis?}` (or a bare string for `kind="text"`). Response returns the new `element_id`/`object_id`/`layout_id`/`rect`. Use ONLY in the **narrow preserve case** (§5.4): an existing realistic photo/overlay is already good and you ONLY need to ADD a missing label/stat/callout — this keeps the photo instead of re-authoring + overwriting. **NOT the default** — most fixes (wrong/ugly/off-message) = re-author the whole SVG via (B). |

After every `modify_scene`, **pull `video_data` again (and `scene_geometry` if it was a layout edit)** to confirm the value actually changed correctly on the server.

### Layout is server-managed — the agent does NOT audit placement

**WideCast now guarantees the mechanical layout for you.** After any overlay upload/edit, the server auto-fits every overlay object so that: no object enters `dead_top`/`dead_bottom`, no object covers `boxes.narrator.face`, and everything stays inside the safe zone. **The agent must NOT re-verify any of this** — no dead-zone proof, no face-clearance check, no A-roll layout-priority ladder. Those checks are gone. Trust the server for placement.

**The narrator is fixed input.** Never edit `segment.narrator_face`, and never resize/reposition the narrator (`overlay.narrator.rect`) to make room for an overlay — the server already keeps overlays clear of the face. If a scene genuinely needs a different narrator crop that only the user can decide, that is a user action, not an agent layout edit.

`scene_geometry` is therefore only a helper for the rare case where you must make a *targeted content fix* to an overlay object (e.g. read a `layout_id` to move one object via branch (C), or confirm an edit saved in Gate 5). It is not a per-scene audit step. You do not pull it to "check the layout is OK" — the layout is OK by construction.

**What the agent still does with overlays is narrow:** fix a genuine defect the server can't judge — an image-model typo (Gate 4), or a background that doesn't fit (Gate 3, which touches `mediaUrl` only, never the overlay). Everything else about placement/composition is the server's job.

## 6. When to Use Screenshots

> **⭐ To SEE/evaluate the CURRENT state of a scene's overlay there is exactly ONE correct method — pick by capability:**
> - **Agent CAN see images** → pull a **screenshot**: `scene_inspector` / `widecast_scene_inspector` → `screenshot_scene_280x498`. It returns the **real server-composited view** (background footage + overlay poster + caption) at 280×498 — exactly what the viewer sees. There is exactly ONE valid transport for WideCast scene screenshots: read `result.screenshot.url`, download it with `curl -L -s -o scene.jpg "<url>"`, show `scene.jpg`, then evaluate. Sidecar JSON, base64, binary `ImageContent`, browser screenshots, REST-auth calls, and remote URL embeds do not count. **This screenshot path is THE way to look for composition/layout.** Narrow exception: for overlay typo/grammar/diacritic/glyph proof, call `widecast_scene_inspector` with `action="overlay_poster"`, the topic `id`, the scene `voice_file`, and `activate:true`; read the returned poster URL, download it with `curl -L -s -o overlay_poster.png "<url>"`, show the local PNG before judging, then read text there. Do not construct the poster URL manually from `voice_file`. The poster (Gate 4) is the ONLY look on a preserve scene with image-gen text; the composite (Gate 5) is only for confirming an edit.
> - **Agent CANNOT see images** (pure-LLM) → pull the **layout**: `scene_geometry`, and reason over the JSON boxes (object rects, narrator/caption boxes, safe zones).
> - **DO NOT fetch the spec yourself and render/composite it locally** (PIL / canvas / headless browser). It is wasteful, non-canonical, and can diverge from the server's real render. The screenshot endpoint already composes the truthful view server-side — **including the `caption` layer, which a hand-rendered poster misses**. For text proof, use only the server URL returned by `widecast_scene_inspector action="overlay_poster"` as a clean overlay-text image; do not hand-composite it over the thumbnail/background, do not construct the poster URL manually, and do not launch a browser/local converter to recreate it.
>
> **Overlay poster transport — same pattern as screenshots, different action.** Use this only for overlay text/glyph proof:
> ```json
> {
>   "name": "widecast_scene_inspector",
>   "arguments": {
>     "id": "<topic_id>",
>     "action": "overlay_poster",
>     "voice_file": "<scene_voice_file>",
>     "activate": true
>   }
> }
> ```
> Then read the returned poster URL, run `curl -L -s -o overlay_poster.png "<url>"`, show the local file, and only then proofread the poster text. Do not derive the URL from `voice_file`.
>
> **⭐ EVERY screenshot you pull → DOWNLOAD LOCALLY FROM `result.screenshot.url` + SHOW it to the user BEFORE you reason or act on it.** The `scene_inspector` result must carry `result.screenshot.url`. Run `curl -L -s -o scene.jpg "<url>"`, then present the local file via the environment's local-file display (`SendUserFile` / `present_files` / local image attachment). Do **not** judge from the remote URL itself, and never embed the remote URL as evidence. Sidecar JSON (`returned_as`, `bytes`, `request_id`, `status`), base64 dumps, binary `ImageContent`, browser screenshots, REST-auth calls, and online `<img>` galleries do NOT count. Looking at the image yourself is NOT enough — the user must see what you saw. This applies to the plate look, the poster look, and the AFTER look. **Show ≠ pause:** present it, then keep working (don't wait for a reply — only the end of the video is a stop, §2). Skipping the show (silently consuming the screenshot) is a process error.

> **The bytes are NOT hard to get — stop over-thinking it. The whole mechanic:** call `scene_inspector` `screenshot_scene_280x498` → read `result.screenshot.url` → `curl -L -s -o scene.jpg "<url>"` → show `scene.jpg`. Done. The URL may be public/signed/unguessable and short-lived; it is a transport only. Do not judge from the URL and do not embed the remote URL as the evidence.
>
> **Do NOT** substitute a browser screenshot, base64, binary ImageContent, local rendering, or REST-auth download. For WideCast scene screenshots, the temp URL → local file path is the only supported route.

**The three looks — at most 3 per scene, often 0–1. There is NO BEFORE composite look.**

1. **Background plate (Gate 3)** — the active background thumbnail (`thumbnailUrl`/plate), shown locally, to judge semantic/logic/geo/context fit. Only when Gate 3 applies (non-grid, narrator not covering the frame).
2. **Overlay poster (Gate 4)** — the isolated overlay on black, shown locally, to proofread image-model-baked text for typos. Only when Gate 4 applies (illustration/chart/diagram/object with image-generated text). Never for SVG typography.
3. **AFTER composite (Gate 5)** — the full server render, shown locally, ONLY if you actually made an edit — to confirm the fix reads as intended. No edit → no AFTER look.

You do NOT pull a composite screenshot to "check the layout/dead-zone/face/composition" — the server guarantees all of that. Pull an image only to judge the two blind spots (background fit, image-gen typo) or to confirm a fix. Never pull reflexively after each small tweak; batch edits, then one AFTER look. Never consume a pulled image privately — every look is one user-visible render (Critical Rule 0 / §1.9).
