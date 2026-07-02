# §7 Quality Standard + §8 Video-Level QA + §9 Priority Order

Load this module:
- **Before declaring `Scene N: PASS`** — to scan against the Quality Standard alongside the 9 DoD gates (`ai_video_editor/03_dod_gates`).
- **After editing every scene** — for the video-level QA pass.
- **Whenever you hit a conflict between gates** — to use the priority order to break the tie.

---

## §7 Quality Standard for a Passing Scene

This is the bar the per-scene **PASS verdict** is checked against: to declare **`Scene N: PASS`** you must confirm **every** item below **AND** all 9 DoD gates (`ai_video_editor/03_dod_gates`) are met. Any miss → `FAIL`, fix, re-scan.

A scene is considered passing when:

- `text` is correct in context, with no significant STT errors
- the visual fits the content (correct `pattern`/`sub_mode`)
- for location-sensitive industries/scenes, the background's geography/culture/currency cues match the target market; wrong-country footage, foreign currency, wrong signage/language, wrong road context, or obviously foreign forms/architecture fails the scene even if the object/action is otherwise relevant
- the background is relevant (or an intentional grid within the ≤3-scene cap)
- if `show_narrator=true`, `boxes.narrator.face` is clear and not covered
- the overlay is correct in content and large enough to read on mobile
- every visible overlay string is typo-free, grammar-correct, uses the right language/diacritics/casing, and preserves required numbers, currency, %, proper nouns, and domain terms
- **rendered image typo/grammar check passed from the screenshot itself:** the agent read the exact visible words in the local-shown 280×498 image, including generated/image-baked text, chart/map/document/UI labels, and old overlay assets; no pseudo-text, malformed glyph, stale spec text, missing/wrong diacritic, typo, grammar error, wrong language, wrong number/currency/%/symbol, or wrong domain term remains; for Vietnamese/diacritic languages, every tone mark, accent, horn/breve/circumflex, vowel mark, and `Đ/đ` is visibly correct
- **title readability is only the minimum floor:** hero/title text must be bright, high-contrast, vivid, **900-weight and thick-bodied via an 8–15x same-fill face stack**, with a thin controlled outline <=2px, immediately prominent, graspable in the first second, and states the scene takeaway rather than merely repeating a panel label. Readable-but-thin, hollow-looking, grey, or timid title text fails even if every letter can technically be read. Also inspect for **over-thick duplicate fill**: if letters look blobby, counters/negative space close up, Vietnamese diacritics are swallowed, or the form loses premium sharpness, reduce stack count/offset or change font; it fails even within 8–15. More than 15 face copies is over-thick and fails.
- **secondary text/label readability is a separate PASS gate:** every non-title label/value/card line/callout is readable in the 280×498 screenshot without zoom, has enough contrast against its own card/background, fits its container with padding, and is not overlapped/covered by another badge/object/text. **Small/non-title text must NOT use visible stroke/outline**; if contrast is weak, use a clean chip/card/backplate, increase size/weight, or simplify the copy instead of adding a border. No secondary text is dark-on-dark, muddy, over-stroked, thin, blurred, cramped into an icon/card, hidden inside a dark card, or decorative at the expense of comprehension. A strong title does not forgive unreadable labels.
- **every overlay object/text is inside `safe_rect` — NONE intrudes into `dead_top` or `dead_bottom`** and the printed **Gate 6 DEAD-ZONE PROOF** passed for the latest overlay/layout state (re-check on the rendered screenshot vs `scene_geometry` rects, because server auto-fit / A-roll auto-center can push content out after upload — see verify step in `ai_video_editor/30_overlay_core`). If the whole overlay is too tall for the safe band, the remedy order is: move whole group → resize whole group → rebuild only if the resize breaks title/secondary readability.
- the caption is readable, doesn't overflow `dead_bottom`
- the layout is balanced — nothing covers the face, overflows the safe zone, overlaps the caption, or makes important text visually fight the background/objects
- the scene plays its correct role in the whole-video flow
- **the modules required for the work actually done were loaded at this step** (background pass → `20_background`; endpoint scene → `40_thumbnail_cta`; A-roll overlay/narrator tradeoff → `10_mechanics`; overlay → `30_overlay_core` + `31_typography` whenever the overlay contains any text + the matching content module/style lib) — a step done "from memory" without opening its module does not count as PASS
- **if the scene is A-roll and Gate 4 handled/approved/rebuilt/applied an overlay, the Gate 4 A-ROLL LAYOUT PRIORITY PROOF was printed before the overlay decision** — a shrunken narrator fallback or accepted existing layout without full-canvas-first proof fails even if the final geometry looks clean
- **if an SVG was authored/rebuilt, the Gate 4 MODULE LOAD PROOF was printed before drawing and included every required module** — a beautiful SVG created without the proof still fails the process gate, because future agents need auditable evidence that they loaded the right playbook pieces
- **Gate 9 MODULE COVERAGE GATE passed** — `00_ENTRYPOINT`, `01_critical_rules`, `02_jump_prevention`, `03_dod_gates`, `04_principles_workflow`, `05_quality_qa_priority`, `10_mechanics`, `20_background`, `30_overlay_core`, `31_typography`, `32_charts`, `33_patterns`, and `40_thumbnail_cta` are either loaded where required or explicitly N/A with reasons

---

## §8 Video-Level QA After Editing All Scenes

After editing each scene, the agent must re-check the whole video:

1. Re-read the full script after the `text` edits.
2. Check continuity between scenes.
3. Check which scenes have repeated visuals (including reusing the same `mediaUrl`).
4. Check whether the caption style is consistent.
5. Check whether the A-roll/B-roll (`show_narrator`) alternation is reasonable.
6. Check whether the hook scene (`type=HOOK`) is strong enough.
7. Check whether the CTA (`type=CALL TO ACTION`) and the last content scene are clear CTA endpoints: one action understood in ~1 second, typography stronger than decorative objects, and narrator-primary if A-roll.
8. Check whether the claim/disclaimer is correct for the industry.
9. Check whether the visual tone is consistent.
10. Check whether any scene looks markedly lower quality.
11. Check the grid balance: **≤3 scenes use grid, all sharing ONE grid** (see `ai_video_editor/20_background`); the rest use a fitting real background.

---

## §9 Priority Order When There's a Conflict

When two gates pull in opposite directions, decide by this order (top wins):

1. Don't get the content wrong.
2. Don't cover the narrator's face (`boxes.narrator.face`).
3. The caption/text must be readable; for title/hero text, readability is only the floor and first-second punch/thickness must also pass.
4. The main overlay must be large enough.
5. The background must be relevant (or a clean grid when the overlay is the content).
6. The layout must be beautiful and intentional.
7. Limit expensive screenshot/API use only after the required local-visible visual evidence gates are satisfied; never use cost-saving as a reason to skip BEFORE/AFTER screenshots or local-show.
8. Don't break previous intentional edits (e.g. `overlay.narrator.touched=true`, `remotion_spec="none"`).
