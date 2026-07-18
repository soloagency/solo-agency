# 40 · Thumbnail + CTA scenes — opening poster, static cover, closing action

_Version: `modular-1.1` · module of the AI Video Editor Playbook (`SKILL.md`)._

> **⚠ RETIRED — DORMANT ON PURPOSE (2026 lean model). Do NOT load. Do NOT delete.** Endpoint/poster authoring — opening-poster, thumbnail-sync, closing-CTA design — is no longer an agent gate; WideCast produces the overlays and the final CTA / thumbnail are routed like any other scene. The one surviving opening-frame concern (the hook poster) lives INLINE as the **Gate 4 OPENING POSTER CHECK** in `03_dod_gates` — not here. This file is kept intentionally, unpromoted and unused, ONLY as a reference in case endpoint authoring is ever re-enabled. It is not in the LOAD MAP; the fail-open "load unknown modules" rule does NOT apply to a RETIRED module. Do not load it, do not cite it, do not run an ENDPOINT DESIGN VARIANT PROOF or thumbnail-sync gate — and do not delete it either.

> **Module of the AI Video Editor Playbook.** Master index + checklist + critical rules live in `SKILL.md`. **Load this when:** you are handling the first real scene after the thumbnail (usually scene 2 / the opening poster scene), its immediate static thumbnail sync, OR the last content/CTA scene.
> Cross-refs: screenshot truth + geometry → `10_mechanics.md`; background plate evidence → `20_background.md`; SVG overlay rules → `30_overlay_core.md`; endpoint visual grammar → `styles/design_languages.md`; poster typography → `31_typography.md` + `styles/text_axes.md`.

---

## 40.0. Mental model

The video's high-leverage endpoints are special:

1. **Opening poster scene:** the first real scene after the thumbnail, usually scene 2. Platforms often auto-extract this frame as the default video cover, so this scene must look thumbnail-grade even if its `type` is `HOOK`, `KEY POINT`, `DATA`, `FACT`, etc.
2. **Static thumbnail scene:** the `type="thumbnail"` scene. It is synced immediately after the opening poster scene passes. It should clone the opening poster's visual identity: same selected frame/background plate, same short poster title, same SVG overlay unless the immediate sync screenshot proves a small adjustment is needed.
3. **Closing CTA scene:** the last non-thumbnail/content scene, especially `type="CALL TO ACTION"`. It is the viewer's final memory and the moment to ask for interaction. Treat it like a closing action poster, not a generic explanatory scene.

The opening poster pair has one job: make a viewer understand the promise/consequence in ~1 second and want to click. The closing CTA has a different job: make the viewer know exactly what to do next in ~1 second.

Do **not** judge these endpoint scenes by the normal scene-overlay standard alone. A normal scene explains one beat; the opening poster + thumbnail package the whole video, and the final CTA converts attention into action.

### Default winning formula

1. **Background:** first make scene 2's frame strong enough to be the cover. Use its own `thumbnailUrl`/A-roll frame when it has a clear face/emotion and matches the video topic. If scene 2's frame is weak, fix/choose the opening frame first; the static thumbnail should normally inherit that frame.
2. **Poster title:** 2-5 words, max 2 primary lines. It should be the click promise or consequence, not the full quote.
3. **Text placement:** lower-third / chest / negative-space placement that preserves eyes, nose, mouth, and the recognisable subject. For non-human thumbnails, preserve the product/vehicle/document/prop.
4. **Readability is only the floor:** huge heavy title, bright fill, controlled thin outline/shadow, and poster-level first-second punch. A thumbnail title may be larger than normal scene titles, and its **letter body must look genuinely thick in the final 280x498 screenshot**.
5. **Contrast device:** a local scrim/gradient behind title is allowed and often useful, but keep it tasteful. It should support open typography, not become a boxed title card or giant ad banner.
6. **Controlled poster decoration:** if the poster still feels flat after the title is thick enough, add a small set of SVG decoration accents (for example a vertical/horizontal bar, title bracket, diagonal slash near the title, tiny alert mark, or thin underline). These accents should create TV-poster energy and guide the eye toward the title/subject, not add new information.
7. **Endpoint style mandate:** scene 2 / thumbnail / final CTA must look like an endpoint poster, not like an inside-scene card. A plain horizontal text bar, centered rounded card, or normal title-over-panel layout is too generic for endpoints even when readable. Pick one clear endpoint style from the ladder below and make it visible in the final screenshot.

Example for an insurance deductible video:

- Better thumbnail hook: `SAI DEDUCTIBLE?` / `MẤT NGHÌN ĐÔ`
- Optional LARGE subline (≥48px, same size as any other support text): `MỖI NĂM`
- Worse: `CHỌN SAI DEDUCTIBLE MẤT HÀNG NGHÌN ĐÔ/NĂM` as four small muddy lines.

### Endpoint visual style ladder

The ladder below defines the **endpoint archetype** (the role/composition of the endpoint). It is not the full visual style. For every opening poster, synced thumbnail, and final CTA, the visual grammar must come from one chosen `styles/design_languages` language. Print `Gate 4 ENDPOINT DESIGN VARIANT PROOF` before authoring, uploading, applying, or approving an endpoint overlay.

Good endpoint design = `design language` + `endpoint archetype` + scene-specific background/subject integration. Bad endpoint design = repeating one safe recipe across unrelated videos.

The known bad default template is: **giant outlined all-caps title + red vertical side bar + red/double underline**. That motif is BLOCKED as a default. It can be used only when the chosen design language and topic genuinely justify it, and the proof shows distinct variant tokens beyond those bars/underlines.

Pick **one** of these for scene 2, the synced thumbnail, and the final CTA. The agent chooses; do not ask the user.

**Hard layout rule (applies to every archetype below): the poster is MAX 3 TEXT LINES TOTAL — hero line(s) and support/contact lines all counted; default is 1–2 lines. Each line is ONE full-width horizontal band spanning 70–100% of the safe width. Never a narrow column that wraps a sentence into 2-words-per-line stacks, never scattered word blocks — those collide with the hero and cannot be auto-fit. Everything beyond the 3 lines must be vector decoration, not text.**

1. **Dynamic poster typography:** oversized FULL-WIDTH title lines (max 3) with staggered baselines and strong scale contrast, and a few motion slashes/brackets anchored to the title — no word blocks or columns.
2. **Magazine-cover thumbnail:** editorial masthead-like full-width headline plus ONE full-width cover-line below it, a sticker/seal or side rule, and cover-style framing around the face/subject.
3. **Kinetic stacked type:** 2-3 FULL-WIDTH lines stacked like trailer beats (each line spans the safe width — never single floating words), with stepped baselines, directional underline bars, or staggered reveal-ready groups.
4. **Typographic collage:** one giant full-width keyword line plus 1-2 full-width support lines with contrasting alignment, still readable in one glance.
5. **Object-integrated title:** full-width title lines placed to frame the narrator face, product, vehicle, document, or key prop without covering the important part.
6. **Premium CTA poster:** giant action verb (`SAVE`, `COMMENT`, `DM`, `BOOK`) as the full-width hero line plus ONE full-width support line, with a vertical side bar, editorial rule, seal, or bracket. Use this often for the final scene.
7. **Minimal premium cover:** one huge word/number line plus ONE LARGE (≥48px) secondary label line — a real second reading level, never a tiny caption — lots of negative space, one elegant accent. Minimal is allowed; generic is not.

Do **not** use the normal inside-scene recipe here: a neat rounded card with horizontal title + subtitle, a row of chips, a balanced info panel, or text centered in a box. Those are acceptable for interior explainer scenes, but endpoint scenes need poster energy — delivered as at most 3 full-width lines, not as a card of small text.

Endpoint style still obeys all hard gates: face/subject clear, caption clear, safe zone, no dead-zone intrusion, copy correctness, title thickness, secondary text readability, and server-saved proof. Style is not an excuse for clutter.

---

## 40.1. Opening poster pair workflow

Run this when you reach the first real scene after the thumbnail. **Before moving to scene 3, sync the static thumbnail with the same poster overlay and verify it.** There is no final thumbnail pass at the end of the video unless the user explicitly asks for thumbnail-only debugging.

### A. First real scene / opening poster scene

1. **Read data:** identify the first non-thumbnail segment by order. Do not depend on `type=HOOK`; whatever comes first after the thumbnail is the opening poster scene.
2. **Show current scene:** pull `screenshot_scene_280x498`, download `result.screenshot.url` locally with `curl`, show it, then evaluate.
3. **Show the active frame/plate:** download the scene's active `thumbnailUrl`/media plate locally and show it separately. Decide by sight whether this frame can sell the whole video.
4. **If the frame is weak:** fix the opening scene's plate/frame first. Do not let the static thumbnail become stronger than scene 2; the pair should match.
5. **Author the poster overlay:** load `30_overlay_core.md`, `31_typography.md`, `styles/design_languages.md`, and `styles/text_axes.md`. Choose one design language and one endpoint archetype from the ladder (`dynamic poster typography`, `magazine-cover thumbnail`, `kinetic stacked type`, `typographic collage`, `object-integrated title`, `premium CTA poster`, or `minimal premium cover`). Print `Gate 4 ENDPOINT DESIGN VARIANT PROOF` before drawing/uploading. Use this module's short-title, thick-title, and decoration standards. Show a cheap local overlay preview only if already available; otherwise skip pre-upload preview and verify from the post-upload composite screenshot.
6. **Upload/apply to scene 2 first:** use `remotion.upload_overlay` by scene 2's `voice_file`.
7. **Verify the video-scene version:** pull/show the final scene 2 composite screenshot. It must pass poster readability AND video-scene coexistence: face/subject clear, caption still readable, and the caption does not visually compete with or crowd the poster title.
8. **If caption conflicts:** revise the poster layout, reserve a caption lane, simplify the poster text, or adjust caption placement when the tool supports it. Do not declare scene 2 PASS while the poster title and caption fight each other.
9. **Save the poster identity:** record the selected frame/plate URL and the uploaded overlay URL; these become the source of truth for the static thumbnail scene.
10. **Immediate thumbnail sync before scene 3:** identify the `type="thumbnail"` scene by stable `voice_file`, then apply the same uploaded poster overlay URL with `remotion.upload_overlay`. If the thumbnail background plate is not already scene 2's chosen frame/`thumbnailUrl`, set/check that too when the edit branch is available. Pull `screenshot_scene_280x498`, download `result.screenshot.url` locally with `curl`, show it, and evaluate the synced thumbnail. Re-pull `video_data`/`scene_geometry` to confirm the thumbnail scene now points to the synced spec/overlay. **Do not start scene 3 until this sync is server-saved and visually verified.** If your scene-2 change was a TEXT-ONLY fix applied via `overlay.text_edit` (no upload), sync the thumbnail the same way: apply the identical `overlay.text_edit` to the thumbnail scene's `voice_file` — its spec and overlay SVG are clones of the opening poster's.
11. **No final thumbnail pass:** after this immediate sync gate passes, thumbnail is done for the run. At the end of the video, do not re-open or re-check the thumbnail unless the user explicitly asks.

---

## 40.2. Final CTA scene workflow

Run this when you reach the last non-thumbnail/content scene, and always when `type="CALL TO ACTION"`.

1. **Read data:** identify whether this is the last content scene and whether `type="CALL TO ACTION"`, `talking_point`, `quote`, or `text` asks for an action.
2. **Show current scene:** pull `screenshot_scene_280x498`, download `result.screenshot.url` locally with `curl`, show it, then evaluate the closing frame.
3. **Decide the CTA role:** the scene must end with one clear action. Examples: `COMMENT`, `FOLLOW`, `SAVE`, `BOOK A CALL`, `DM ME`, `GET A QUOTE`, `REVIEW YOUR POLICY`, `TALK TO AN EXPERT`. Do not use a vague title like `KẾT LUẬN` when the viewer should act.
4. **Prefer typography over objects:** final CTA overlays should usually be text-led, not object-heavy. Use 1 short hero CTA line plus, at most, one small support line. A clear typographic call beats a chart/checklist/icon collage at the end.
5. **A-roll final scene:** if `show_narrator=true` / `active_roll="A"`, the human close wins. Keep the narrator full canvas or large by default. Place the CTA as bold lower-third/chest/side typography that clears eyes, nose, mouth, and caption. Do not shrink the narrator into a fallback picture-in-picture layout for a CTA unless a detail-dense visual is truly indispensable and the higher-priority full-canvas/shifted-full-canvas layouts have been rejected.
6. **B-roll/faceless final scene:** use a clean centered or lower-third CTA group inside the safe zone. If the background is busy, add a tasteful local scrim/backplate behind the CTA text.
7. **Author/adjust the CTA overlay:** load `30_overlay_core.md`, `31_typography.md`, `styles/design_languages.md`, and `styles/text_axes.md`. Choose one design language and one endpoint archetype, usually `premium CTA poster`, `dynamic poster typography`, `magazine-cover thumbnail`, or `minimal premium cover`. Print `Gate 4 ENDPOINT DESIGN VARIANT PROOF` before drawing/uploading. The CTA may share a brand accent with scene 2, but it must not blindly clone scene 2's poster motif; it needs action-led hierarchy. Use poster-grade title/body thickness, bright fill, thin controlled outline/shadow for the hero CTA, and no visible stroke on small support text.
8. **Avoid small text clutter:** remove social icons, QR-like marks, tiny handles, tiny labels, or decorative objects unless they are large enough to read at 280x498 and directly support the action. If a handle/URL is needed, it must be large, simple, and not fight the caption.
9. **Verify:** pull/show the AFTER screenshot. CTA PASS requires: one dominant action, title-grade readability, face/caption clear, no dead-zone object, no cramped support text, and server-saved confirmation.

CTA title patterns:

- `COMMENT "DEDUCTIBLE"`
- `SAVE THIS CHECKLIST`
- `FOLLOW FOR MORE`
- `BOOK A QUICK REVIEW`
- `DM ME "QUOTE"`
- `CHECK YOUR POLICY`

CTA FAIL triggers:

- The closing scene looks like another explainer beat with no clear next action.
- The CTA is a tiny subtitle while decorative objects dominate.
- A large chart/checklist/object collage competes with the narrator's final human close.
- The CTA duplicates the caption or states a vague category instead of an action.
- The CTA text is dark, thin, cramped, or hidden in the caption/dead-bottom area.
- No `Gate 4 ENDPOINT DESIGN VARIANT PROOF`, no chosen design language, or no CTA-specific variant check.
- The CTA blindly clones scene 2's poster motif instead of creating an action-led closing composition.

---

## 40.3. Title formula

The poster title should be shorter and more forceful than the scene quote or narration. It is shared by scene 2 and the static thumbnail.

Good patterns:

- `MISTAKE?` / `BIG CONSEQUENCE`
- `WRONG CHOICE?` / `MONEY LOSS`
- `ONE RULE` / `BIG NUMBER`
- `AVOID THIS` / `SAVE/PROTECT X`

For Vietnamese:

- Keep diacritics.
- Use mixed Vietnamese + domain term when the term is what viewers search for: `SAI DEDUCTIBLE?` is acceptable.
- Do not shrink to fit a long sentence. Shorten the hook instead.

Hard limits:

- Primary title: max 2 lines.
- Optional subline/badge: max 1 short line.
- No title line should visually touch the canvas edge; keep at least ~5% side breathing room.
- If a title needs 3-5 lines, the copy is wrong for thumbnail.

---

## 40.4. Typography and layout standard

Opening-poster / thumbnail / CTA typography may be more aggressive than normal scene text:

- Title can exceed the normal scene title cap when needed; target roughly **70-110px** on the 720 canvas for worded titles, larger only if the line is very short.
- Use a 900-equivalent VN-safe face such as `Be Vietnam Pro Black` / `Inter Black`, and set `font-weight="900"` where possible.
- Use bright white/yellow/red accents with a **thin controlled outline** and shadow. Do not make the title look strong by making the outline huge.
- **Poster title thickness is stricter than normal scene title thickness.** Endpoint titles should usually use **12–15 same-fill face copies** (within the global 8–15 range) with tiny offsets around center (`x±1.5-3.5px`, `y±1.5-3.5px`, plus diagonal offsets), then one final top copy with only a **thin** dark stroke (`~1.5-2px`). This thickens the letter body without turning the outline into a black blob. Do this for the main poster words before changing layout or adding more decorations. But 12–15 is still only a starting range: if the poster title becomes blobby, deformed, closes counters/negative space, crushes tracking, swallows Vietnamese marks, or feels less premium, reduce count/offset or switch font. More than 15 face copies = FAIL.
- The title must read at 280x498 without zoom AND feel punchy in the first second; "readable but still thin" is endpoint FAIL.
- Tiny sublines must have no visible stroke/outline. Use solid fill on a scrim/card or remove the subline.

Layout:

- Preserve face/subject: eyes, nose, mouth should stay clear unless the title intentionally frames around them.
- Lower-third title over torso/negative space is often best for human thumbnails.
- A local scrim/gradient behind the title is allowed; avoid full-canvas opaque panels that kill the photo and never put the poster title inside a generic card/chip/pill/rounded rectangle.
- A badge/pill is optional. If it looks like a clickable button or clutters the poster, remove it or turn it into simple open text.
- Accent strips/underlines must be thin and directional. Do not add multiple warning marks, stickers, badges, arrows, or boxes just to create energy.
- Do not let accent strips become the whole identity. The repeated motif `red side bar + red/white underline + giant outlined title` is a FAIL as a default endpoint recipe; if used, the endpoint proof must justify it through a named design language and distinct composition.
- Avoid normal explainer-card composition at endpoints: no generic rounded card with centered horizontal title, no plain lower-third rectangle with one line of text, no row of small chips as the main design, and no balanced info-panel look. If a backplate/card is needed for secondary/support text contrast, it must be integrated into a poster/magazine composition, not become the design; the main title remains open typography.

### Poster/CTA decoration standard

Decorations are allowed on the opening poster pair and final CTA because these are endpoint scenes, not normal explanatory scenes. Use SVG-native shapes only unless the user explicitly asks for an image:

- Good decoration types: side bars, top/bottom bars, title brackets, thin title-adjacent diagonal slashes, small alert icons, glow wedges behind/near the title, underline strokes, simple spark marks near the title.
- Use **1-3 decoration groups** total. A group may contain several related thin shapes, such as two bracket lines around one title block.
- Decoration must support the hierarchy: face/subject first, title second, decoration third.
- Decoration must be visually **anchored** to the title block or subject: title-adjacent, behind/around the title, or an edge/frame accent that clearly points the eye into the title/subject. Do not place a standalone corner mark just because that corner is empty; "negative space" is not enough. If the eye cannot trace the decoration back to the title/subject in the 280x498 screenshot, remove it or relocate it.
- Keep the palette tight: usually one accent family plus white/black contrast. Do not turn the thumbnail into a multicolor sticker sheet.
- Bars, brackets, underlines, and slashes should point attention toward the title or consequence. Standalone corner rays / corner laser marks are banned because they often float away from the message. Random or floating decorative marks are fail.
- If using an icon, it must be instantly legible at 280x498 and topic-relevant. Prefer a simple warning triangle, dollar mark, shield, car, document, or check/cross mark.
- Decoration can be bright, but it must not make the title less readable. If an accent touches or crowds the title, move it outside or lower opacity.
- Decoration is still a real overlay object: before upload, preflight every decoration group's bbox against the overlay safe box (`x=36..684`, `y=128..960` on the 720x1280 canvas). A bar/slash/spark that sits in the top notch or bottom caption reserve is a FAIL even if it is "only decorative"; move it inward before uploading.
- Decoration must visibly register in the final 280x498 screenshot, not only in any pre-upload overlay preview. If bars/slashes/underlines disappear after compositing/compression, thicken or brighten them; if that makes the poster crowded, remove them.
- Title-adjacent bars/slashes are often safer than decoration over the face. Use edge bars, corner brackets, or underlines around the title block before placing marks near a human subject.
- Decoration is a post-pass after copy, title thickness, plate choice, and face preservation are already solved.

### Iteration standard

After the first passing opening-poster/thumbnail pair, run one controlled improvement pass:

- Compare the current screenshot against the previous screenshot by sight.
- Improve only one visual variable at a time: title body thickness, scrim opacity, poster decoration, or placement.
- Keep the stronger version only if it improves click clarity without adding clutter.
- If the improvement makes the thumbnail feel crowded, revert to the simpler passing version.

---

## 40.5. Pass/fail gate

Opening poster pair PASS requires all of:

- The current/final screenshot for scene 2 was downloaded locally and visibly shown before judgment.
- The synced thumbnail screenshot was downloaded locally and visibly shown immediately after scene 2, before judgment.
- The background plate was shown separately.
- The chosen opening frame/plate has a clear subject and matches the video topic.
- Scene 2 and the static thumbnail use the same poster identity: same selected frame family, same short hook/consequence, same SVG unless the immediate thumbnail sync screenshot justified a small clone-specific adjustment.
- The thumbnail sync happened immediately after scene 2 PASS and before scene 3 started; there is no final thumbnail pass.
- The title communicates the whole-video promise/consequence in ~1 second.
- The design uses a named endpoint style from the ladder and looks poster-like, not like a normal inside-scene card/text overlay.
- The design also uses a named `styles/design_languages` language and passed `Gate 4 ENDPOINT DESIGN VARIANT PROOF`.
- The title is huge, bright, **extra thick-bodied from a 12–15x face stack**, and readable at feed size. "Readable but still thin" is not enough for a thumbnail; muddy/blobby/deformed duplicate fill fails even inside 12–15, and >15 copies is over-thick and fails.
- The title does not cover the subject's eyes/nose/mouth or the key product/prop.
- In scene 2, caption and poster title coexist: no caption/title overlap, no visual crowding, and the caption does not become a second competing headline.
- No cramped text, no muddy/dark text, no edge-touching title, no subtitle that needs zoom.
- The final `video_data`/`scene_geometry` confirms the new overlay/spec persisted.

Opening poster pair FAIL triggers:

- Full quote pasted as many small lines.
- Endpoint design is just a normal card, centered text box, horizontal lower-third bar, or generic title/subtitle layout.
- Endpoint design repeats the same giant outlined title + red vertical bar + red/double underline motif by habit, especially across unrelated videos.
- No `Gate 4 ENDPOINT DESIGN VARIANT PROOF`, no chosen design language, or a vague style claim without variant tokens.
- Weak/dim text over a bright or busy photo.
- Title is technically readable but not poster-thick / lacks first-second punch.
- Title covers the face or key object.
- Badge/subline makes the design feel crowded or ad-like.
- Decorations feel random, cover the subject, crowd the title, or become louder than the message.
- Scene 2 and the thumbnail look like unrelated designs.
- Scene 2 looks good as a paused frame but fails once caption is visible.
- Background is a random stock/abstract plate while a strong opening-scene frame exists.

Final CTA PASS requires all of:

- The current/final screenshot for the CTA scene was downloaded locally and visibly shown before judgment.
- The scene communicates one clear action in ~1 second.
- The design uses a named endpoint style from the ladder and looks like a CTA poster, not a normal interior explainer card.
- The design also uses a named `styles/design_languages` language and passed `Gate 4 ENDPOINT DESIGN VARIANT PROOF`.
- The CTA hero text is title-grade: bright, thick-bodied from the face stack, high contrast, and punchy/readable at 280x498.
- If A-roll, narrator face remains the primary human close; CTA typography supports the face instead of replacing it with object clutter.
- Support text, if any, has no visible stroke/outline and remains readable without zoom.
- Caption, CTA, and overlay do not fight each other.
- No CTA text/object sits in `dead_top` or `dead_bottom`.
- The final `video_data`/`scene_geometry` confirms the new overlay/spec/layout persisted.

Final CTA FAIL triggers:

- No clear next action, or the action is less prominent than decoration.
- No `Gate 4 ENDPOINT DESIGN VARIANT PROOF`, no chosen design language, or no CTA-specific variant check.
- The CTA blindly clones scene 2's poster motif instead of creating an action-led closing composition.
- The CTA uses the same red side-bar/double-underline/giant-outline recipe as a default rather than a justified design-language choice.
