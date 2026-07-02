# 20 · Audit & choose background (grid vs real, search, evaluate, apply)

_Version: `modular-1.1` · module of the AI Video Editor Playbook (`SKILL.md`)._

> **Module of the AI Video Editor Playbook.** Master index + checklist + critical rules live in `SKILL.md`. **Load this when:** you reach the background decision for a scene (grid or a real image/video) and the find/evaluate/apply loop.
> Cross-refs: decide-by-sight + pattern table also summarized in master; overlay→`30_overlay_core.md`.

---

## 4. Audit & Choose Background

Whether A-roll or B-roll, the agent must evaluate whether the background is appropriate. **But before going to find footage, the agent must decide whether this scene actually needs a video background at all.** Not every scene needs stock video; many scenes left as grid (or a neutral dark/blurred background) will be cleaner and easier to read.

> **Show ≠ ask.** Contact sheets, candidate thumbnails, and downloaded plates are shown so the run is auditable; they are not a request for the user to pick. At runtime the agent evaluates the visible evidence, chooses the best background itself, applies it, and reports the decision. Do not stop to ask "which clip/image should I use?"

### 4.0. Layer isolation — Gate 5 is background-only

Gate 5 owns only the background/media plate. If the current background is wrong, no fitting background is found, the geo/context check fails, or the correct fallback is grid, the allowed edit is branch (A) `mediaUrl`/`mediaType` only.

- Do **not** rebuild, replace, disable, restyle, or upload an overlay because the background failed.
- Do **not** redraw an existing realistic overlay photo/map/entity image/complex visual just because the background changed or became grid.
- If the new background makes the overlay harder to read, first finish Gate 5 as a background action, then return to Gate 4/Gate 6 with an independent overlay FAIL proof and the normal least-destructive repair ladder.
- The reverse is also true: an overlay defect does not justify background replacement unless Gate 5 independently proves the background fails.

### 4.1. The first decision: GRID or a REAL background (image/video)

The first audit question for every scene: **grid, or a real background?**

> **⭐ DECIDE BY SIGHT, NOT BY `pattern`.** The `pattern`/`sub_mode` fields exist for the **blind automated pipeline** (WideCast can't see, so it leans on labels). **You CAN see — so play a different game: decide from the topic's vibe and what the scene actually needs, not from the pattern label.** The table below is only what the blind pipeline *would* default to — treat it as a hint you are free to override, not a rule.

**Lean toward a real background by default** — a fitting image/video is more alive, specific, and engaging, and gives the video variety. **Grid is the exception and is capped (≤3 scenes/video, §4.1a).**

- **Choose a REAL background when** there's a concrete subject/action/entity to show (person, brand, product, place, event, the very thing being discussed), or the narration describes a tangible scene footage reinforces, or it adds context/emotion/credibility.
- **Choose GRID only when** the overlay fully carries the message AND any footage would be generic/distracting (an abstract list / set of criteria / formula / pure-typography statement), OR you want a deliberate calm, neutral beat, OR no fitting background can be made or found (last resort).

> Quick test: remove the overlay — if the scene still says something, it wants a real background; if it's meaningless without the overlay, the overlay IS the content and grid is a candidate.

### 4.1c. Geo / location / currency context gate

This gate applies **only when location materially changes trust or meaning**. Turn it on when the full script, `text`, `visual`, `keyword`, `talking_point`, offer, disclaimer, audience, currency, law/regulation, or industry points to a specific country/state/city/region.

Location-sensitive examples include: real estate, mortgage, insurance, tax, legal/estate planning, healthcare, immigration, education, local services, jobs, government/regulation, finance, benefits, and any scene with a named city/state/country or currency.

When this gate is on, the background must match the target geography and cultural cues, not just the object/action:

- US real estate / mortgage / insurance / finance should not show non-US currency, non-US streets/signage, clearly foreign houses, foreign license plates, left-side driving, non-English/local-language signage, or documents that obviously belong to another country.
- If the scene says dollars / USD / US rules, visible money should look like US dollars or be neutral; foreign cash is an immediate FAIL.
- If the scene is about a named place, prefer footage/images from that place or visually neutral close-ups. Do not use a different country's landmarks, architecture, street signs, police/medical/official uniforms, forms, or road environment.
- If a candidate is otherwise beautiful but geographically/culturally wrong, reject it. It damages credibility more than a simpler neutral background.
- If no correct geo-specific footage can be found, choose a neutral/abstract close-up with no wrong-location cues (desk, keys, generic paperwork, hands, house interior, calculator, car detail) or grid, rather than a vivid but wrong country.

Search/evaluation implication: include the needed location/currency in search keywords when relevant (`US real estate`, `California home`, `US dollars`, `American car insurance`, etc.), and explicitly state geo fit in the background verdict.

**Blind-pipeline default table — a HINT you may override by sight:**

| `pattern` (`sub_mode`) | Pipeline default | Seeing-agent note |
|---|---|---|
| `typography_only` · `single_metric` · `bar_chart` · `proportion_chart` · `trend_chart` · `structural_diagram` · `map_chart` · `comparison_table` · `timeline_events` · `checklist_tips` · `quote_card` · `hybrid_vertical` · `illustration`(`document`/`digital_ui`) | grid (overlay is the content) | The overlay carries the message, **but a fitting real background behind a readable (dark-card / outlined) overlay is usually better than a bare grid** — verified this session (scenes 4/6/7/8). Use grid only within the ≤3 cap. |
| `illustration`(`photo_with_people` / `photo_no_people`) | **real background** | The photo/scene IS the message → real background (§4.1b). |
| `narration_only` | grid or light footage | `visual=""`, `quote=""`; if A-roll, the narrator face is the visual. If the narrator fills/occludes the canvas, do not content-check the hidden background/fallback plate. |
| `real_entity` | real image (pipeline pre-filled it BLIND) | The pre-filled image is often wrong — you SEE, so fix it (§5.0). |

- **⭐ FIRST CHECK — BYPASS CASES:**
  - **Force-grid / active grid:** if the scene is force-grid or the active media is already a grid clip, SKIP the real-background search/evaluation below — grid is already applied on purpose, leave it. Signal: **`segment.force_grid === true`**, `stock_fallback_reason` ending in `_force_grid` (e.g. `hybrid_vertical_force_grid`, `illustration_document_force_grid`, `illustration_digital_ui_force_grid`), or an active `mediaUrl`/`brollUrl` that is clearly a grid background. Do **not** evaluate subject matter, geo cues, or content fit for a grid. Only confirm the local-shown composite/plate, the grid cap/shared-grid rule, and that overlay/caption readability is not harmed. Edit the **overlay** only if Gate 4 independently proves an overlay defect; grid-by-design is not an overlay rebuild trigger.
  - **Full-canvas A-roll:** if `active_roll="A"` / `show_narrator=true` and the narrator fills or occludes the canvas, the narrator is the visual. Do **not** evaluate the hidden fallback/background plate for content fit. Mark the background action as no-op and continue with narrator/overlay/layout checks.
  - Also respect an explicit `use_stock_video=false` the user deliberately set.

### 4.1a. Grid background — the rules

- Find it with `search_broll` using **exactly one keyword `"grid"`**. It returns DOZENS of grid clips — you cannot view them all: pull **a few at random**, download their thumbnails locally, build/show a local contact sheet, and **only then** look/pick by the **current topic's vibe**. Never choose a grid from an online thumbnail or private visual inspection.
- **One grid for the WHOLE video**: every grid scene shares the SAME chosen grid (consistency). Pick a **different grid from other videos** (cross-video variety).
- **Hard cap: ≤3 scenes per video may use grid**; in general, minimize grid. If a video genuinely needs more, state the reason in the final hand-off.
- Apply: `modify_scene` (A) `mediaUrl=<grid_url>`, `mediaType="video"`.
- **Grid is the ONE exception to the no-duplicate rule (§4.6)** — grids are deliberately shared across the video's grid scenes; no-duplicate applies to real footage only.
- **Grid has no content audit**: do not judge whether the grid's "subject" matches the narration, geo/currency context, or real-world setting. It is an intentional neutral stage for the overlay; audit only the grid count/shared-grid consistency and whether the composite remains readable.

### 4.1b. Real background — image or video, and how to get it

Two kinds; pick by whether motion matters.

**Static IMAGE** — WideCast **auto-animates** it (pan / tilt / zoom, Ken-Burns), so a still still feels alive. Use it for a specific entity / portrait / logo, or a background you can craft perfectly as a still. Sources:
- **Search client-side YOURSELF** (your own web/image search), download candidate images locally, show them locally to the user, and **only then** evaluate by eye / pick the winner. ⚠ **Do NOT use any WideCast image-search API** — image search is the agent's own job (the WideCast image-search endpoint is being deprecated and is intentionally not part of this playbook).
- Apply: `upload_asset` (presign) → PUT the file → `modify_scene` (A) `mediaUrl=<asset_url>`, `mediaType="image"`. (A plain public web URL can go straight into `mediaUrl`.)

**VIDEO** — use when motion / action carries it (people working, a process, a dynamic scene). Sources:
- **Self-gen** the video (only if you have video-gen, or an MCP connected to a video engine).
- Else **`search_broll kind=video`** (stock — the current default) → evaluate (§4.4–4.6).
- Apply: a result URL → `modify_scene` (A) `mediaUrl=<url>`, `mediaType="video"`. A self-made / gen file → `upload_asset` → PUT → `mediaUrl`.

⚠️ **Background ALWAYS uses `modify_scene` branch (A) `mediaUrl` (+`mediaType`). NEVER use branch (I) `narrator.upload_video` for a background** — that field is the A-roll NARRATOR's face video; putting a background there switches the scene to an A-roll narrator. Two different fields — never confuse them.

### 4.2. Evaluate the CURRENT scene — BEFORE composite + active background/media plate

To judge the current background (and the whole scene), use the already shown **BEFORE composite screenshot** when available, or pull one now (`scene_inspector` / `widecast_scene_inspector` → `screenshot_scene_280x498`, §6), read `result.screenshot.url`, download it to a local file with `curl -L -s -o <local>.jpg "<url>"`, show it visibly in chat, and only then evaluate it. It composites the scene **as it actually renders** — background + narrator + overlay + caption — so it tells you whether the background is actually visible in the final scene.

Then pull the **active background/media plate** separately, download it locally, show it visibly, and only then analyze it. This is required because the composite screenshot can make agents misattribute a background object (e.g. a wallet/card graphic) as an overlay, or wrongly think an overlay is hiding the background.

**Which URL to pull for the active plate:**
1. Prefer `thumbnailUrl` — it tracks the active runtime media (`mediaUrl`) and avoids accidentally judging an inactive lane.
2. If `thumbnailUrl` is missing and `mediaType="image"`, use `mediaUrl`.
3. If still missing, fallback by active roll: `active_roll="B"`/`show_narrator=false` → `brollThumbnailUrl`; `active_roll="A"`/`show_narrator=true` → `arollThumbnailUrl`.
4. Do **not** judge the current rendered background from an inactive lane. Example: an A-roll scene can have `brollThumbnailUrl` in data, but the viewer does not see it while `active_roll="A"`.

From the two local-visible images decide:
1. What belongs to the active background/media plate vs what belongs to overlay/caption/narrator.
2. Is the background appropriate for `text` / `talking_point` / `visual` / `keyword`, including geo/location/currency fit when §4.1c applies, and is it visible in the composite? **Skip this content-fit question for grid and for hidden plates behind a full-canvas A-roll narrator.**
3. If it already fits → keep it.
4. If it does NOT fit → go to §4.4–4.7 to find a better one. (There you look at the new **candidate** thumbnails returned by `search_broll`, but only after downloading them locally and showing a local contact sheet.)

### 4.4. Real-video find & evaluation workflow (for a video background)

1. Download the candidate thumbnails locally.
2. Build a local contact sheet.
3. Show the local contact sheet to the user.
4. **Only after the user-visible local contact sheet is shown**, look at the thumbnails (not just read the URLs).
5. Compare against `text`, `visual`, `keyword`, `talking_point`, `pattern`, and geo/location/currency context when §4.1c applies.
6. Evaluate whether the background matches the topic, emotion, industry, geography/culture, and message.

A background is deemed not good enough if: it's unrelated; too generic; wrong industry; wrong geography/currency/culture/context when location matters; too dark or too cluttered; has unwanted text/logos; distracts from the narrator/overlay; doesn't support the scene's emotion.

### 4.5. Evaluate for real and pick the best candidate

**Mandatory rule:** the clip applied to a scene must be **the result of evaluating context fit** (`text`, `talking_point`, `visual`, `keyword`, `pattern`), and **you must not default to re-setting `originalMediaUrl`/the old clip**.

- Every scene getting a real video background must pass through this evaluation step before applying. There is no "keep the old clip" shortcut that skips the comparison.
- Read the search results carefully and **immediately drop off-topic clips** ("server" returns a cake knife, "rack" returns a gym, "surge" returns ocean waves).
- For location-sensitive scenes (§4.1c), immediately drop candidates with wrong-country cues, wrong currency, wrong signage/language, wrong road/vehicle context, or obviously wrong local architecture/forms. A pretty but wrong-geo clip cannot win.
- When `keyword` is weak, derive a better 1–3 word phrase yourself from `text`/`talking_point`/`visual`. If it's still poor, **search again with a different keyword**.
- State clearly why you picked a given clip and why you rejected the others.
- When placing them on the contact sheet, **don't always put the chosen clip at position #1**.
- **`originalMediaUrl`/the curated clip should only be kept if it genuinely wins the comparison**, or when the user explicitly asks to only restore.

### 4.6. Mandatory technical filter before applying a clip

- **Aspect ratio**: prefer a **natively portrait** clip (height > width) for vertical video; avoid a landscape clip that gets cropped/zoomed.
- **Resolution**: sharp enough for a 720×1280 frame or larger.
- **Motion & brightness**: a scene with lots of text/caption → pick a clip with **little motion, not too bright**, to avoid the background overwhelming the caption.
- **Geo/currency/culture fit**: when §4.1c applies, visible location cues must match the target market. Wrong currency, wrong country signage, wrong road side, wrong official forms, or visibly foreign setting = reject.
- **No duplicates**: don't reuse the same clip for multiple scenes; track the list of `mediaUrl` already used across the whole video. **Exception: the shared grid (§4.1a) is deliberately reused across the video's grid scenes** — this rule applies to real footage only.
- **Clean**: no burned-in text/logo/watermark, no misleading content.

### 4.7. Apply and verify

- Apply the clip with `modify_scene` branch (A): `field_name=mediaUrl` (+ `mediaType="video"`; for an image, `mediaType="image"`), selector `by="voice_file"`. A search/web result is a URL → put it straight in `mediaUrl`; a **self-made or self-gen file** → `upload_asset` (presign) → PUT → then `mediaUrl=<asset_url>`.
- ⚠ **Background = branch (A) `mediaUrl` only. NEVER branch (I) `narrator.upload_video`** (that is the A-roll narrator's face video — see §4.1b).
- **Layer isolation check:** a background apply/fallback must leave `remotion_spec`, overlay objects, overlay layout, and realistic overlay images unchanged unless Gate 4 already printed an independent overlay FAIL proof for this scene.
- **After applying, pull `video_data` again** to confirm `mediaUrl` changed correctly.
- For a scene with `stock_fallback_reason`/`hybrid_vertical`, check whether the background reverts to grid by design; if it does, that is correct behavior — don't override it.
- If needed, take 1 screenshot to check the readability of text on the new background — download `result.screenshot.url` to a local file with `curl`, show the local image first, then evaluate.

## 11. Background Video Selection & Evaluation Workflow (Detailed)

This section describes the operational loop for reviewing and replacing the background per scene. It works for both automatic mode and a demo/recording with a user watching.

### 11.1. Preparation (once for the whole video)

1. Pull the entire video data first.
2. For each target scene, save a baseline: `voice_file`, `id`, `text`, `talking_point`, `keyword`, `visual`, `type`, `pattern`, `sub_mode`, the current `mediaUrl`, `mediaType`, the current thumbnail. Note that `originalMediaUrl`/`originalThumbnailUrl` are already available as a restore point.
3. Identify the target scenes: usually from `id=2` (after the thumbnail scene) to the last scene. Skip a scene only when the user explicitly asks.
4. Create a **Background Audit Ledger** with a blank row for every target scene. Required columns: `scene`, `voice_file`, `composite_local_path`, `active_plate_local_path`, `geo_context_required`, `geo_verdict`, `decision`, `action`, `verdict`. Do not fill visual judgments during preparation; fill each row only inside that scene's Gate 5 after local-visible evidence has been shown.
5. Audit each scene per §4.1 — decide grid vs a real background **by sight + topic vibe, not by the pattern label**; grid is the capped exception (≤3/video, §4.1a). A scene is not background-audited until its ledger row has a PASS/FIXED verdict.

### 11.2. Grid placeholder

Use a grid video as the activating/temporary placeholder media:

`https://widecast.ai/downloads/brolls/grid_backgrounds/grid2026-05-17_at_10.48.55_AM_7.mp4?v=1779040364`

Sending `modify_scene` (A) with the grid for one scene makes the editor auto-scroll/activate that exact scene. No manual browser scrolling.

### 11.3. The per-scene loop (in order)

0. **BYPASS CHECKS — check FIRST.**
   - **Force-grid / active grid:** if `segment.force_grid === true`, `stock_fallback_reason` ends in `_force_grid`, or the active media is already a grid background, the background is **grid by design** → **skip B–H entirely** (no `search_broll`, no real-footage content evaluation, no replace). This does **not** skip Gate 5 proof: still use the local-shown BEFORE composite + active plate, print `Gate 5 BACKGROUND PROOF`, update the Background Audit Ledger, and mark `Verdict: PASS grid-by-design` when the grid cap/shared-grid rule is satisfied. Then go straight to Gate 6 final composition.
   - **Full-canvas A-roll:** if the narrator fills/occludes the frame, the narrator is the active visual → **do not evaluate the hidden fallback/background plate for content**. Still show the composite + active plate for auditability, print `Gate 5 BACKGROUND PROOF`, set `Decision: A-roll narrator is the visual`, `Action: no background action because A-roll narrator fills frame`, and continue.

A. **Activate the scene with a grid request.** `modify_scene` branch (A), `by="voice_file"`, `value=scene.voice_file`, field `mediaUrl=GRID`, `mediaType="video"`. The editor auto-scrolls/activates the scene.

B. **Decide grid or a real background — by sight + topic vibe, not by `pattern`** (§4.1). Lean to a real background; grid is the capped exception (≤3/video, §4.1a).
   - Overlay-led scene (text/diagram/chart/document/digital_ui/hybrid): a fitting real background behind a readable overlay is usually best; use grid only within the ≤3 cap, then state the reason and move on.
   - Special (`narration_only`/`real_entity`): handle per §4.1; `real_entity` image search is the agent's own (§5.0), no WideCast image API.
   - Photo-led scene (`photo_with_people`/`photo_no_people`): continue to B2–I.

B2. **(real-background scene) Check the current background first** — from the **local-shown BEFORE screenshot + local-shown active background/media plate** (§4.2), evaluate what the asset is and how it actually renders under the overlay/caption. Then print the master `Gate 5 BACKGROUND PROOF` template and update the Background Audit Ledger row for this scene. If it already fits, keep it, mark the row `Verdict: PASS keep`, and move to Gate 6 final composition. Only proceed to C if it doesn't fit.

B3. **Hard stop before final composition.** Gate 6 is blocked until the current scene has a filled ledger row and a printed proof verdict: `PASS keep`, `PASS grid-by-design`, or `FIXED + PASS`. If the proof is missing, the correct status is `Scene N: FAIL — Gate 5 incomplete; fixing`, not PASS.

C. **State the keyword/context.** One short sentence explaining what keyword you'll search and why, based on `text`/`talking_point`. If §4.1c is on, include the target country/state/city/currency in the search phrase or explain why a neutral no-geo-cue visual is safer. (For a demo/recording, write this sentence in English.)

D. **Search B-roll.** Use the scene's `keyword` if it's good; if weak, derive a 1–3 word phrase yourself from `text`/`talking_point`/`visual`. For location-sensitive scenes, add the geography/currency term to the keyword when it helps (`US`, `California`, `US dollars`, etc.). Get 4–6 candidates, portrait if the video is vertical. Keep the metadata: number, title, video URL, thumbnail URL, source.

E. **Metadata pre-filter only.** Before seeing thumbnails, you may drop candidates using metadata only: wrong orientation, obvious duplicate URL, unusable source/title. Do **not** make any visual choice yet.

F. **Download the thumbnails locally & build a contact sheet.** Download the thumbnails into `work/broll_review/scene_<voice_file>/` (`option_1.jpg`, ...). Build a horizontal contact sheet (style §11.4). Display via the local file, don't paste an external URL/raw HTML.

G. **Show the contact sheet BEFORE evaluating visually or applying the clip.** Display the local contact sheet first, then state clearly which clip number is chosen and why. Don't look privately, don't apply first, and don't show afterward.

G1. **Contact sheet = candidate triage only.** A contact sheet can choose a candidate, but it does not prove Gate 5 PASS and it never proves `Scene N: PASS`. Gate 5 still needs the active background/media plate + composite screenshot evidence for this exact scene, and the scene still needs the full 9-gate DoD before PASS.

G2. **Evaluate for real & technical filter.** Now that the local contact sheet has been shown, drop off-topic clips; apply the §4.6 filter (natively portrait, not too bright for a text-heavy scene, no duplicates, no watermark). If poor, search again and repeat F→G→G2.

H. **(Demo) Wait ~1 second** so the user can record. Keep going — don't stop to wait for a response; the work must not be interrupted.

I. **Apply the best-fit clip — not the old clip.** `modify_scene` (A) on the same scene with `mediaUrl` = the URL of the clip **that won the evaluation step (E + §4.5)**, `mediaType="video"`. The applied clip must come from the context comparison, not be re-set to `originalMediaUrl` just because it's available. After applying, pull the data again to verify (§4.7), pull/show the updated composite if needed for readability, then update the ledger row to `Verdict: FIXED + PASS`.

J. **Move to the next scene**, repeat.

### 11.4. Suggested contact sheet style

- One horizontal row, 4–6 panels; each thumbnail ~150–220 px, dark background.
- Show the candidate's index; the chosen candidate has a bright border + a `SELECTED` label.
- Save `.jpg`/`.png`; **render it as a viewable image in chat** (Cowork uses `present_files`), don't paste an external path/URL.
- Don't always put the chosen clip at position #1.

### 11.5. Common mistakes to avoid

- Using `segment.id` as the API selector instead of `voice_file` (the selector must be `voice_file` — the stable UID).
- Conversely, referring to a scene by `voice_file` when **talking to the user**: a human thinks in scene numbers, so address it as "scene 4" (the `id`/scene index), not the internal UID. (`voice_file` = API selector; scene number = human communication.)
- Editing `segment.narrator_face` to adjust layout (wrong — use `overlay.narrator.rect`).
- Scrolling the browser manually.
- Looking at candidate thumbnails privately before showing the local contact sheet.
- Treating a contact sheet as scene proof. It is triage/selection evidence only, not Gate 5 PASS and not `Scene N: PASS`.
- Applying the clip before showing the contact sheet.
- Pasting raw HTML or just pasting an external thumbnail URL.
- Always putting the chosen clip at #1.
- Defaulting whole pattern-categories to grid (or, the opposite, forcing footage where the overlay fully carries it) instead of deciding by sight; exceeding the ≤3-scene grid cap.
- Judging the background from only one image. The BEFORE screenshot is render truth, but the active background/media plate is required to separate background content from overlay/caption/narrator. Conversely, never judge from the thumbnail/plate alone because it may be hidden or altered in the composite.
- Re-setting the old clip/`originalMediaUrl` without evaluating context.
- Painting a **full-canvas background inside the overlay SVG** (it hides the scene video) instead of leaving it transparent.
- Low-contrast overlay text with **no outline/chip** (it sinks into the footage), or not wrapping each object in a `data-wc-object` group (wrong or missing animation).
- Forgetting the verification step (pull `video_data`/`scene_geometry` again) after applying.


### 11.6. Map of choosing visuals by context (examples, not exhaustive)

- Network/infrastructure hardware: server rack, router/switch, blinking signal lights.
- Power outage/lightning: lightning storm, power surge.
- Downtime/closure: a dark empty office, a shop pulling down its shutters.
- Hardware solutions: a UPS on a rack, a power strip/surge protector, a 4G device.
- A frustrated/waiting user: a person sitting in front of a laptop with a broken connection.
- Figures/charts/large text/diagrams: the overlay leads — **a fitting real background behind a readable (dark-card/outlined) overlay is usually best**; use a bare grid only within the ≤3-scene cap.
- A "before it's too late" CTA: a storm rolling in, a countdown clock.
