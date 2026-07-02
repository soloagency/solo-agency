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

- Exception for A-roll priority 4 fallback: the narrator is allowed to exceed the canvas along the **Y axis**, but the X axis always stays inside the canvas, and the face (`boxes.narrator.face`) must remain inside `safe_rect`.

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

### Step 3: Audit Layout with `scene_geometry` as the Measuring Tool

If the scene has a narrator (`show_narrator=true`, i.e. `active_roll="A"`) or has an overlay (`overlay.<sub>.visible=true`), the agent **calls `scene_geometry` to get the precomputed layout** for measurements. But this does **not** satisfy the visual gate: before choosing an A-roll layout priority, deciding an overlay/background is good, or editing from a visual judgment, the agent must pull the scene screenshot with MCP `scene_inspector` / `widecast_scene_inspector` `action="screenshot_scene_280x498"`, read `result.screenshot.url`, download that URL to a local file with `curl -L -s -o <local>.jpg "<url>"`, show the local file visibly in chat, and only then evaluate it. If `result.screenshot.url` is missing, the visual gate is not satisfied. Do not use base64, binary `ImageContent`, sidecar JSON, browser screenshots, or REST-auth workarounds for WideCast scene screenshots.

Read from the result:

- `boxes.narrator.face` and `face_center` — whether the narrator's face is occluded by an object/text (against `boxes.remotion.objects` and `boxes.caption`).
- `safe_zones` — whether objects/overlays are inside `safe_rect`, and whether they overflow `dead_top`/`dead_bottom`. Use this to fill the mandatory **Gate 6 DEAD-ZONE PROOF** in `03_dod_gates`; do not bury the check inside a general composition verdict.
- `boxes.remotion.object_layer.objects[].rect` — whether the overlay is large enough to read/see detail.
- `boxes.caption.container_rect` — whether the caption covers the face or the main object.
These checks are **yours to compute from the boxes above** (there are no pre-scored verdict arrays). For any **visual/aesthetic** judgment (does it look good, does text sink, is the icon right, is the narrator too small, is this the right A-roll layout priority) use a **local-shown screenshot from MCP `screenshot_scene_280x498`** (§6), not the geometry. Geometry-only approval is forbidden when screenshots are available.

**For B-roll (`show_narrator=false`): IGNORE any overlap between an overlay and the narrator face/body** — even though `overlay.narrator` (PIP) exists in the data, the narrator is NOT the subject being shown, so "face occlusion" here is a false alarm. Only care about: safe zone, not overlapping the caption, a readable overlay. The face-clearance condition **only applies when `show_narrator=true`**.

For an A-roll scene, the narrator's face is a high-priority element. If the face is covered, fix the overlay/narrator first with `modify_scene`:
- move/resize an overlay object: branch (C) `remotion.object.rect` (read `layout_id` from `scene_geometry` first);
- move/resize the whole overlay group: branch (D) `remotion.group.rect`;
- move/resize the narrator: branch (E) `overlay.narrator.rect`;
- move the caption: branch (F) `overlay.caption.y`;
- multiple things at once: branch (G) `layout.batch`.

**Whole-overlay dead-zone decision ladder:** when the overlay group itself is too high/low/tall, do not jump straight to rebuilding. First try move-only `remotion.group.rect` (`x/y`) to translate the whole group. If that solves one dead-zone violation but creates another on the opposite edge, try whole-group `remotion.group.rect` resize with `resize_mode:"scale_children"` so the group fits the safe band. Only if the resized result fails the visible typography/readability gates (title no longer feels like a title; secondary labels drop below mobile-readable size; text becomes muddy/cramped) should the agent rebuild/regenerate the SVG.

**B-roll / faceless / non-face-critical safe-zone placement preference:** first identify whether the overlay is text-only or mixed text+objects. For `typography_only` / text-only overlays, prefer placing the text block near the vertical center of `safe_rect`, not automatically at the top-safe band. For mixed text+object overlays, prefer a top-safe title/text band with the object/chart/cards below it, still fully inside `safe_rect` and visually connected; do not let objects drift near the caption or detach from the title. These are starting preferences, not permission to cover an important background subject: if the local screenshot shows the text/object group sitting on a real person's face, product, or key prop, slide the group within `safe_rect` until the background and overlay both read.

**Absolutely do not edit `segment.narrator_face` to adjust layout** — that is source data; adjust the narrator box via `overlay.narrator.rect`.

The narrator's body may partially overlap an overlay in some layouts, but the face (`boxes.narrator.face`) must not be covered.

### Step 4: Choose the A-Roll Layout Priority

If the scene is A-roll (`show_narrator=true`, i.e. `active_roll="A"`), the agent MUST run the priority ladder below during **Gate 4 overlay review/rebuild**, before deciding to leave, rebuild, upload, apply, or layout-fix the overlay (adjusted via `overlay.narrator.rect` + `remotion.object.rect` / `remotion.group.rect`, 280×498 space). This is not a post-hoc Gate 6 cleanup. For normal scenes, a currently-small narrator / picture-in-picture layout is NOT automatically acceptable. The agent must first test whether the narrator can stay full canvas.

This ladder is for normal A-roll scenes. Special endpoint / trust / CTA scenes keep their stricter rules below.

**Starting-layout bias guard — reset the design baseline.** If the BEFORE screenshot or `scene_geometry` shows a shrunken / picture-in-picture / fallback narrator, treat that as a suspicious current state, not the design baseline. Before judging the overlay, the agent must mentally and operationally reset the candidate to **Priority 1: narrator full canvas**, then solve the overlay around that full-canvas narrator. The small narrator layout is evidence of what is currently on screen only; it is not permission to keep it, and it does not count as testing Priority 1.

If the agent starts from a shrunken narrator and evaluates only whether the existing overlay fits that shrunken layout, Gate 4 has not run. Priority 1 is valid only when the narrator is actually treated as full canvas first, then the overlay is moved/resized/simplified/rebuilt around that full-canvas candidate.

**Joint composition rule — do not freeze the overlay.** Each priority below is a `narrator + overlay + caption` composition attempt, not a narrator-only attempt. The current overlay position/size is only diagnostic; it is not a fixed constraint. Inside every priority, solve in this order:

1. keep the narrator at the priority's target size/position;
2. move overlay objects / the overlay group to a legal zone;
3. resize the overlay group if the group is too tall/wide, then re-check typography;
4. simplify or rebuild the overlay into a compact support shape if the existing shape cannot fit;
5. only then mark that priority failed or move to a smaller narrator priority.

Do **not** reject full-canvas priorities because the existing overlay currently touches `boxes.narrator.face`, the caption, or a dead zone. That is a layout problem to solve first. A full-canvas priority fails only after you prove no moved/resized/simplified/rebuilt overlay can clear the face, avoid caption/dead-zone collisions, preserve readability/padding, and still look balanced in the local-shown screenshot.

> **⭐ REQUIRED GATE 4 A-ROLL LAYOUT DECLARATION — before ANY A-roll overlay decision or layout edit.** After the BEFORE screenshot has been saved locally and shown to the user, explicitly state:
> - `scene_class`: `normal` / `special_endpoint_or_trust`
> - `narrator_role`: `primary` / `secondary`
> - `overlay_role`: `support` / `main_subject`
> - `starting_layout_bias_reset`: `yes` when the current narrator is shrunken/fallback and the design baseline was reset to Priority 1 full canvas before overlay judgment; `N/A` only when the current narrator is already full-canvas/shifted-full-canvas
> - `overlay_adjustments_tested`: `move` / `resize` / `simplify` / `rebuild` as applicable
> - `full_canvas_gate`: `PASS` / `FAIL`
> - chosen priority number (`1`–`4`)
> - why each higher-priority option failed, if choosing priority 2–4
>
> If the current narrator is shrunken/fallback and `starting_layout_bias_reset` is not `yes`, Gate 4 is BLOCKED. If the agent chooses priority 4 without proving priorities 1–3 failed, Gate 4 fails and Gate 6 cannot pass. Do not accept an existing shrunken narrator layout just because it is already on screen.
>
> For **CTA, contact, trust, intro, outro, testimonial, or direct-address scenes**, the narrator is normally `primary`; prefer priorities **1–3** and keep the narrator large. **Priority 4 is forbidden for CTA/contact/trust scenes unless the overlay is detail-dense and truly the main subject** (e.g. a document, UI, chart, comparison table, product screenshot, or technical process diagram). Shrinking the narrator into a small picture-in-picture because it satisfies geometry is a defect, not a pass.
>
> **⭐ FINAL A-ROLL / CTA SCENE — HUMAN CLOSE WINS, CTA TEXT STILL MATTERS.** If this is the **last non-thumbnail/content scene** and `show_narrator=true` / `active_roll="A"`, first `Read` `40_thumbnail_cta.md`, declare `narrator_role: primary` by default, and try to keep the narrator **full canvas**. The default close is: large human face + one short typography-led CTA that clears the face and caption. Do **not** rebuild a large checklist/chart/diagram just because `visual` asks for one if it would turn the closing A-roll into a cramped graphic scene. No-overlay is acceptable only when the visible narrator + caption already communicate the CTA clearly; otherwise use a small strong text support element (short CTA, badge, or one-line reminder). Priority 4 is effectively forbidden for a final A-roll/CTA scene unless the final beat truly depends on a detail-dense visual.

> **⭐ NORMAL A-ROLL PRIORITY ORDER — try in THIS sequence and take the FIRST that works:**
> 1. **Narrator full canvas.** Keep the narrator full canvas and choose the best legal overlay placement/rebuild: above the head, beside the head, over the chest/torso, or as a compact support element. This is the required first test. PASS only if the adjusted overlay clears `boxes.narrator.face`, stays out of `dead_top`/`dead_bottom`, remains readable at 280×498, does not collide with the caption, and the composition looks balanced in the local-shown screenshot.
> 2. **Full canvas but pull narrator down + overlay above head.** Keep the narrator full-size, shift the narrator down to create usable top-safe space, and place the overlay above the head. PASS only if the face stays inside `safe_rect`, the crop still looks natural, the overlay clears the face, avoids dead zones, remains readable, and does not fight the caption.
> 3. **Full canvas but push narrator up + overlay over chest.** Keep the narrator full-size, shift the narrator up to create a clearer chest/torso region, and place the overlay below the face. PASS only if the face stays inside `safe_rect`, the crop does not lose or crowd the face, the overlay clears the face, avoids dead zones, remains readable, and does not fight the caption.
> 4. **Fallback: large top/safe overlay + optimally-small narrator.** Use this only after priorities 1–3 fail. The overlay becomes the main subject in the top/safe area, and the narrator is made as large as possible in the remaining lower space without blocking the overlay.

#### 1. Narrator full canvas

The narrator nearly fills the whole canvas. The overlay must be adapted to this priority, not kept frozen from the previous render. Place or rebuild the overlay in the most viable full-canvas position: above the head, beside the head, over the chest/torso, or as a compact support element. The point is that the narrator stays full canvas while the overlay is arranged around the face and caption.

If the current scene starts with a small narrator, first set/plan the narrator as full canvas for this candidate. Only after that may the overlay be judged. Checking the old small-narrator layout, or keeping the old small narrator while moving the overlay, is not Priority 1.

If the starting overlay is parked at the top safe zone and touches the face, that does **not** fail priority 1. First move it to the chest/side/lower legal band, resize the group if needed, or rebuild it as a shorter typography-led support.

Hard conditions:

- the overlay does not cover `boxes.narrator.face`
- the overlay is large enough to read or see object detail at 280×498
- the overlay stays out of `dead_top` and `dead_bottom`
- the face remains clear
- the caption doesn't break the layout
- the composition looks balanced in the local-shown screenshot

Suited to: a badge, icon, short stat, short quote, warning label, short headline, small/medium object, or light concept visual.

#### 2. Full canvas but pull narrator down + overlay above head

The narrator keeps full-canvas size but shifts down (via `overlay.narrator.y`), possibly cropping the lower body, creating empty space above the head for the overlay.

Hard conditions: the face stays inside `safe_rect`; the overlay doesn't cover the face; the overlay is large enough; the overlay stays out of dead zones; the narrator crop still looks natural; the caption doesn't break the layout.

Suited to when: the overlay needs to sit top/mid-top; the narrator's face is too high; you need to keep the narrator large while still ceding the upper region.

#### 3. Full canvas but push narrator up + overlay over chest

The narrator keeps full-canvas size but shifts up, possibly cropping the head/hair or slightly the upper torso, creating a clearer chest/torso area for the overlay.

Hard conditions: the face stays inside `safe_rect`; the overlay doesn't cover the face; the overlay sits in the chest/torso area and is large enough; the overlay stays out of dead zones; the crop doesn't lose the face or push it too close to the edge; the caption doesn't break the layout.

Suited to when: the overlay should sit below the face; the visual is a badge/stat/object tied to the speech; you need to keep the narrator large.

#### 4. Fallback: large top/safe overlay + optimally-small narrator

This is the last resort for normal A-roll scenes. Use it only when the overlay must be large and central/top-safe enough to communicate the beat, and no full-canvas or shifted-full-canvas arrangement can pass.

The narrator must be as large as possible, not merely "small enough to fit." The narrator may exceed the canvas along the Y axis when that produces a better face size, but the X axis must stay inside the canvas. The goal is to preserve a clear human presence while giving the overlay enough room.

Hard conditions:

- `boxes.narrator.face` stays inside `safe_rect`
- the face is covered by no overlay object, caption, or badge
- the face remains large enough to read as a human face at 280×498
- the narrator is not reduced to a decorative tiny picture-in-picture
- the overlay remains the main subject, large enough to see detail
- the overlay stays out of `dead_top` and `dead_bottom`
- the caption still fits and does not cover the face or main overlay

In this priority, the background should usually be a clean grid or quiet plate so the overlay and narrator do not fight visual clutter.

Suited to: UI, code, a document, a chart, a product screenshot, a comparison table, a process diagram, or another detail-heavy visual that genuinely needs to be seen clearly.

## 6. When to Use Screenshots

> **⭐ To SEE/evaluate the CURRENT state of a scene's overlay there is exactly ONE correct method — pick by capability:**
> - **Agent CAN see images** → pull a **screenshot**: `scene_inspector` / `widecast_scene_inspector` → `screenshot_scene_280x498`. It returns the **real server-composited view** (background footage + overlay poster + caption) at 280×498 — exactly what the viewer sees. There is exactly ONE valid transport for WideCast scene screenshots: read `result.screenshot.url`, download it with `curl -L -s -o scene.jpg "<url>"`, show `scene.jpg`, then evaluate. Sidecar JSON, base64, binary `ImageContent`, browser screenshots, REST-auth calls, and remote URL embeds do not count. **This screenshot path is THE way to look for composition/layout.** Narrow exception: for overlay typo/grammar/diacritic/glyph proof, call `widecast_scene_inspector` with `action="overlay_poster"`, the topic `id`, the scene `voice_file`, and `activate:true`; read the returned poster URL, download it with `curl -L -s -o overlay_poster.png "<url>"`, show the local PNG before judging, then read text there. Do not construct the poster URL manually from `voice_file`. The poster is not a substitute for the composite screenshot's layout/dead-zone/face/caption proof.
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
> **⭐ EVERY screenshot you pull → DOWNLOAD LOCALLY FROM `result.screenshot.url` + SHOW it to the user BEFORE you reason or act on it.** The `scene_inspector` result must carry `result.screenshot.url`. Run `curl -L -s -o scene.jpg "<url>"`, then present the local file via the environment's local-file display (`SendUserFile` / `present_files` / local image attachment). Do **not** judge from the remote URL itself, and never embed the remote URL as evidence. Sidecar JSON (`returned_as`, `bytes`, `request_id`, `status`), base64 dumps, binary `ImageContent`, browser screenshots, REST-auth calls, and online `<img>` galleries do NOT count. Looking at the image yourself is NOT enough — the user must see what you saw so they can catch a wrong call early. This applies to the BEFORE look, the AFTER look, and any look in between. **Show ≠ pause:** present it, then keep working (don't wait for a reply — only the end of the video is a stop, §2). Skipping the show (silently consuming the screenshot) is a process error.

> **⭐ THE SCREENSHOT IS MANDATORY — `scene_geometry` is NOT a substitute for LOOKING.** Geometry gives you rects/boxes (use it for the numeric dead-zone check + `layout_id`s) but says NOTHING about whether the scene *looks* right — text readability/sinking, colour, whether the background fits, whether the overlay is actually beautiful. **A vision-capable agent that evaluates from geometry alone — to avoid pulling a screenshot — has skipped the evaluation. A scene you did not LOOK at cannot be declared `PASS` (§3).** Pull the BEFORE and AFTER screenshots on every scene; geometry only *complements* the look, never replaces it. (Even if the geometry API is unavailable, you still screenshot — the screenshot is the primary tool, geometry is the optional helper.)
>
> **The bytes are NOT hard to get — stop over-thinking it. The whole mechanic:** call `scene_inspector` `screenshot_scene_280x498` → read `result.screenshot.url` → `curl -L -s -o scene.jpg "<url>"` → show `scene.jpg`. Done. The URL may be public/signed/unguessable and short-lived; it is a transport only. Do not judge from the URL and do not embed the remote URL as the evidence. If no URL is returned or `curl` cannot download it, mark the scene `FAIL` and do not edit from that screenshot.
>
> **Do NOT** substitute a browser screenshot, base64, binary ImageContent, local rendering, or REST-auth download. For WideCast scene screenshots, the temp URL → local file path is the only supported route.

**Encouraged shape — roughly BEFORE and AFTER (not a forced count).**

1. **BEFORE screenshot — the composite render truth.** Pull it before editing, download `result.screenshot.url` to a local file, show it in chat, and only then see the scene as it actually renders: is the background visible in the composite (not hidden behind a full-canvas narrator)? is the overlay good? is the face clear? does the caption fit? Pair it with the Gate 5 active background/media plate (§4.2) when judging the background, so you can separate background objects from overlay/caption/narrator. Use `scene_geometry` for structure, but make visual calls from the local-visible images.
2. **AFTER screenshot — confirm.** After your edits, pull it again, download `result.screenshot.url` to a local file, show it in chat, and only then verify the final: the face isn't covered, the overlay is readable, the caption doesn't cover the main content, the background fits, the layout is balanced, the scene feels professional. If it's not OK → fix → pull/show again. If no edit was made, the BEFORE screenshot may serve as AFTER evidence only if it was already shown visibly and you explicitly state that no edit was made.

> **The real rule is NOT a count — it is: don't pull a screenshot after every tiny tweak.** Each pull costs the user tokens (vision) + WideCast bandwidth, so **batch your edits and pull/save/show only when you genuinely need to see the result.** A simple scene is typically ~2 composite screenshots (before + after), plus the Gate 5 active background/media plate; a **complex** edit may legitimately need a few more between major changes — that's fine. Just never pull reflexively after each small adjustment, and never consume a pulled screenshot/media plate privately.

**Crucially, the decision to regenerate/replace an overlay must be based on what the local-visible MCP screenshot SHOWS — never inferred from `scene_geometry` boxes or spec data alone** (geometry is structural only; only the screenshot reveals aesthetic/readability/render truth). Do not downgrade to geometry-only reasoning just because the tool also returned sidecar JSON; instead, save/download/show the screenshot image locally.

If a screenshot reveals a problem the data doesn't show clearly, the agent goes back to fix it with data/layout then QAs again.
