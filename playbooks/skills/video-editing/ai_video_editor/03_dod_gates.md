# Per-Scene Definition of Done — 9 gates + all template blocks

Load this module at the **START of every scene**. Every scene must pass through 9 gates before you may state `Scene N: PASS` and move to the next scene.

The 9 gates are pass/fail; the actual work-recipes live in the topic modules (`10_mechanics`, `20_background`, `30_overlay_core`, `31_typography`, `32_charts`, `33_patterns`, `40_thumbnail_cta`). This module is the **checklist** + the **exact template blocks** that prove a gate ran.

---

## Batch / gallery / script outputs are triage only

Batch contact sheets, galleries, tables, scripts, bulk API results, multi-scene scans, or any "all scenes at once" artifact are useful only to **triage** likely problems. They are never DoD proof and never authorize `Scene N: PASS`.

If the run used only batch triage, or fixed only a selected subset without closing every content scene, the required status is `partial_triage_only` (or `partial_fix_only` when edits were made but not all scenes have individual PASS). Do not ask for render/export, do not call `export_video`, and do not send completion notification while any content scene lacks its own `Scene N: PASS`.

`Scene N: PASS` requires that exact scene's own 9-gate evidence: BEFORE shown, Gate 4/5/6 completed as applicable, AFTER/final shown, server-saved confirmation, required module coverage, and §7 quality scan. A batch table saying a scene "looks OK" is not PASS.

## DoD — finish EVERY gate below before moving to the next scene

Do NOT advance while any gate is unchecked. Overlay review, background audit, screenshot-show, and final composition/dead-zone are the most-often-skipped checks — they are explicit gates here on purpose.

1. ☐ **Text / STT** checked in whole-video context, fixed if wrong (Step 1, `modify_scene` branch K).
2. ☐ **Role** understood — `type` · `pattern`/`sub_mode` · `visual` · `quote` · `talking_point` (Step 2).
3. ☐ **BEFORE screenshot** pulled with MCP `screenshot_scene_280x498` → `result.screenshot.url` **downloaded locally with `curl` + SHOWN visibly to the user** → **only then evaluated** (`ai_video_editor/10_mechanics`). *Every screenshot you pull is shown, never just consumed silently. Remote URLs, base64, binary `ImageContent`, sidecar JSON, and request ids do not count.*
4. ☐ **Load `ai_video_editor/30_overlay_core` FIRST, then handle the overlay.** Gate 4 starts with **overlay existence/preservation**, not drawing. Read `pattern`/`sub_mode`/`visual`/`quote`/`talking_point`/`remotion_spec` and the local-shown BEFORE screenshot, then print the Gate 4 OVERLAY EXISTENCE + PRESERVE PROOF below. If `pattern="narration_only"`, `visual` is empty, `remotion_spec=="none"`, no overlay is visible, or the scene is intentionally carried by narrator/background/caption, mark overlay audit **N/A** and do not add an overlay/title. If an existing map, realistic image, chart, diagram, or illustration is good enough and on-topic, preserve it; repair only serious defects. Full replacement is forbidden unless the proof shows layout-only/additive/text-preserve/extract-reuse repairs are insufficient and the new overlay is strictly better than BEFORE. If this is an endpoint scene (scene 2/opening poster, thumbnail sync, or final content/CTA), also load `ai_video_editor/40_thumbnail_cta`. If this is A-roll (`show_narrator=true` / `active_roll="A"`), also load `ai_video_editor/10_mechanics` and run the A-roll layout priority ladder **inside Gate 4**, before deciding to leave, rebuild, upload, apply, or layout-fix the overlay. Gate 4 is where the overlay/narrator tradeoff is designed; Gate 6 only verifies/tunes the result. Print the Gate 4 A-ROLL LAYOUT PRIORITY PROOF below; if a normal A-roll scene is currently in a shrunken narrator layout, still test the full-canvas narrator option first. A-roll priority testing is a joint narrator + overlay + caption solve: the current overlay is movable/resizable/rebuildable, not fixed evidence, so full-canvas cannot be rejected until overlay move/resize/simplify/rebuild options have failed. Load `ai_video_editor/styles/design_languages` and choose one compact visual language **only if authoring/rebuilding is actually needed**; style never overrides the Universal Overlay Design Standard or the preserve gate. If the overlay contains ANY text/title/label/value, or a needed repair will create/alter text, also load `ai_video_editor/31_typography`; then load the matching content module `31`/`32`/`33` and its `styles/*`. **Before authoring or rebuilding any overlay, print the Gate 4 MODULE LOAD PROOF template below; if any required module line is missing, STOP and load it before drawing.** If the overlay has or legitimately needs a title/hero line, also print **Gate 4 TITLE GATE PROOF** after drafting and before upload; missing title is not a defect by itself. If the overlay has ANY non-title text/value/label/card copy, also print **Gate 4 SECONDARY TEXT GATE PROOF** after drafting and before upload. Loading typography without passing the relevant title, secondary-text, and copy-correctness gates is not enough. Repair decision follows the ladder in `30_overlay_core`: `LEAVE` → `layout-only fix` → `minor additive fix` → `text/style fix while preserving current visual` → `extract/reuse current visual then rebuild around it` → `full replace`. If rebuilt → authored as a **DIVERSE** overlay from the style lib (internally SVG, never flat-only), saved locally, optionally shown as a cheap **overlay preview** when the runtime already supports it, then applied, granular objects, A-roll face cleared. Do not homogenize all scenes into the same "big title + one object" template. Do not install tools, launch headless browsers, or spend time converting just to preview. **Preflight constraint:** before judging/rebuilding, pull `scene_geometry` to read face/caption/safe zones/object rects, but do NOT declare final composition PASS in this gate. **Copy correctness gate:** every visible overlay string must be spelled, punctuated, grammatical, in the correct language, and correct for numbers/currency/symbols/proper nouns/domain terms; any typo/grammar/diacritic error = overlay FAIL. **Readability is the floor, not approval:** dark/muddy/thin-bodied title text, readable-but-timid title, title below 900-equivalent weight, title/hero/primary CTA/hero metric with fewer than 8 same-fill face copies, title over-thick/muddy/blobby/deformed from duplicate fill even inside 8–15, swallowed Vietnamese marks, closed counters/negative space, title without first-second punch, title boxed inside a card/chip/pill/panel/banner/black translucent rectangle, title stroke above 2px, tiny labels, **secondary text/labels with visible stroke/outline**, low-contrast callouts, card text that sinks into a dark panel, labels/values overlapped by another badge/object, text that only becomes legible when zoomed, or any text that touches/grazes/clips/laps a chip/card/bar border = overlay FAIL, even if the geometry is clean. A title PASS never compensates for unreadable, cramped, or misspelled labels.
5. ☐ **Load `ai_video_editor/20_background` FIRST, then audit the BACKGROUND as its OWN pass immediately after overlay** — NOT folded into the overlay check. Use two local-visible images: the BEFORE composite screenshot (what actually renders) AND the current active background/media plate (`thumbnailUrl` first; fallback per `active_roll`/`mediaType`) downloaded locally + SHOWN before analysis. Decide grid-vs-real BY SIGHT; if real and visible, confirm: fits narration/`keyword`, **geo/location/currency context matches when the industry/scene is location-sensitive** (real estate, insurance, tax, legal, healthcare, local services, finance, etc.), natively portrait, not too bright/cluttered behind the text, no watermark/burned-in text, clean start frame, and actually VISIBLE in the composite. Also check whether the chosen overlay now makes the background too busy or causes text to sink. **Bypass content checks in two cases:** if the active media is grid/force-grid, do not evaluate subject matter or geo cues — only check the grid cap/shared-grid rule and readability; if an A-roll narrator fills/occludes the canvas, do not evaluate the hidden fallback/background content — the narrator is the visual. Apply background changes via branch (A) `mediaUrl`.
6. ☐ **Final composition audited/tuned AFTER overlay/background decisions** (`scene_geometry`; Steps 3–4 in `ai_video_editor/10_mechanics`, plus verify step in `ai_video_editor/30_overlay_core`) — this single gate includes layout, safe zone, dead-zone, face clearance, caption coexistence, background fit, and visual balance. A-roll face clear, hero/title text bright, prominent, 900-weight/open typography, **not boxed in a card/panel**, thick-bodied from the documented 8–15x face stack without muddy/blobby over-thick duplicate fill, punchy in the first second, **all secondary text/labels/values readable on mobile, not overlapped, and not cramped against their container borders**, caption clear, background not fighting the overlay, and final overlay/media arrangement balanced. For A-roll scenes, Gate 6 verifies the Gate 4 A-ROLL LAYOUT PRIORITY PROOF was already printed and obeyed. If it is missing, return to Gate 4; do not invent the proof after the fact. A fallback/shrunken narrator layout cannot PASS unless the higher-priority full-canvas and shifted-full-canvas options were rejected with concrete narrator + overlay + caption reasons during Gate 4, including tried overlay move/resize/simplify/rebuild when relevant. If Gate 4 or Gate 5 changed anything, this is the first place composition can PASS. **B-roll safe-zone composition gate:** for `typography_only` / text-only overlays, prefer placing the text block near the vertical center of `safe_rect`, not top-safe by habit. For mixed text+object overlays, prefer a top-safe title/text band with objects/charts/cards below it, all inside `safe_rect` and visually connected. If the first placement covers a clearly important background face/product/prop, slide the group within `safe_rect` until the composition breathes. **Dead-zone is a hard subcheck here:** no overlay object/text may intrude into `dead_top`/`dead_bottom`, and caption must not overflow `dead_bottom`. If you need to move only 1–12 individual overlay objects, use `layout.batch` + `remotion.object.rect`. If the whole overlay group is misplaced or decomposed into more than 12 objects, use `remotion.group.rect` on the Storyboard group: first try **move-only** `x/y`; if moving the group fixes one dead zone but pushes the opposite edge into another dead zone, try a **whole-group resize** (`w/h` with `resize_mode:"scale_children"`). Only rebuild/regenerate the SVG if the resized group makes title/body/secondary text too small, muddy, cramped, or otherwise fails the typography/readability/aesthetic-padding gates.
7. ☐ **AFTER screenshot** pulled with MCP `screenshot_scene_280x498` → `result.screenshot.url` **downloaded locally with `curl` + SHOWN visibly to the user** → **only then evaluated** → confirms: face clear, hero/title text is bright/prominent, 900-weight/open typography, not boxed in a card/panel, thick from the 8–15x face stack but still crisp (not blobby/deformed/diacritic-swallowing), vivid/punchy in the first second, **every secondary label/value/card line is readable without zoom and has visible inner padding from its chip/card/bar border**, all visible overlay text is typo-free/grammar-correct/domain-correct, no dark/muddy/blurred text, no text-on-text or badge-over-label collision, no text touching/grazing/clipping/lapping a border, nothing in a dead zone, caption fits, background fits. If no edit was made, the already-shown BEFORE screenshot may serve as the final look only when the agent explicitly says "no edit, BEFORE screenshot is the AFTER evidence"; otherwise pull and show a fresh AFTER screenshot.
8. ☐ **Server-saved** — re-pulled `video_data`/`scene_geometry` to confirm every edit persisted.
9. ☐ **MODULE COVERAGE GATE** — print the module coverage proof below and PASS only if every required playbook for this scene/run was loaded at the correct step, or explicitly marked N/A with a reason.

All nine checked → next scene. **Show ≠ pause:** present each screenshot, then keep working — the only stop is the very end of the video. **Scene transition gate:** if the user has not visibly seen the final local screenshot evidence for this scene, the scene is NOT done and the agent must not start the next scene.

---

## Announce the plan + report progress (mandatory)

The user must be able to see, at any moment, **which steps you will do, which you are on, and which remain.** So:

- **At the START of each scene**, post the **task list for THIS scene as a VERTICAL checklist** (one DoD gate per line). This is the plan the user audits up front. **Do NOT compress the 9 gates into one inline sentence** — inline checklists are easy for agents to skim past and hard for humans to audit.
- **As you work**, announce each gate as you enter/finish it ("→ Gate 4: overlay review/rebuild…" then "✓ Gate 4 done"). Don't silently jump between gates.
- **At the END of the scene**, repeat the checklist with ✓/✗ and a one-line note per gate, so the user sees exactly what was done and whether anything was skipped, BEFORE you move on.
- **Announce ≠ pause** — report and keep working; do not wait for a reply. Skipping the announcement (working silently) is a process error: the user cannot audit what they cannot see.

---

## Exact template blocks — use these VERBATIM shapes

Text markers are the source of truth (icons render differently across AI hosts).

### Scene start (plan) — vertical only, never inline

```text
Scene N plan:
☐ Gate 1 — Text / STT
☐ Gate 2 — Role
☐ Gate 3 — BEFORE screenshot shown
☐ Gate 4 — Overlay review/rebuild
☐ Gate 5 — Background audit
☐ Gate 6 — Final composition audit/tune
☐ Gate 7 — AFTER/final screenshot shown
☐ Gate 8 — Server-saved confirmation
☐ Gate 9 — Module coverage
```

### Gate 5 BACKGROUND PROOF — during Gate 5, after Gate 4, before Gate 6

This is the mechanical guard against skipping the background pass after the long overlay workstream.

```text
Gate 5 BACKGROUND PROOF:
☑ 20_background.md — opened for this scene before judging background
BEFORE composite evidence: <local file path shown to user>
Active plate evidence: <local file path shown to user>
Active media: <mediaUrl or active thumbnail/media URL>
Current background read: <what is actually visible in the plate/composite>
Scene context: <1-line text/talking_point/visual summary>
Decision: <real background | grid | A-roll narrator is the visual | force-grid by design>
Fit check: <PASS|FAIL> — relevant to narration/keyword/visual and visible in composite
Geo/currency check: <PASS|FAIL|N/A> — if location-sensitive, country/region/currency/signage/road/form cues match the target market
Technical check: <PASS|FAIL|N/A> — for visible real footage: portrait, not too bright/cluttered, no watermark/burned-in text, no duplicate real clip; N/A for grid or full-canvas A-roll
Grid cap check: <PASS|FAIL|N/A> — grid scenes count <= 3 and share one grid
Action: <keep current media | replace via mediaUrl | leave force-grid | no background action because A-roll narrator fills frame>
Verdict: <PASS keep | PASS grid-by-design | FIXED + PASS | FAIL — continue background search>
```

**If this proof is missing, Gate 5 is BLOCKED.** Do not start Gate 6/final composition, do not declare `Scene N: PASS`, and do not final-handoff the video.

### Gate 4 MODULE LOAD PROOF — before authoring/rebuilding overlay

Include only the modules required for this scene, but do not omit typography when the overlay has text.

```text
Gate 4 MODULE LOAD PROOF:
☑ 30_overlay_core.md — overlay object/upload rules
☑ 10_mechanics.md — A-roll layout priority ladder (required when `show_narrator=true`)
☑ 40_thumbnail_cta.md — endpoint scene rules (required only for scene 2/opening poster, thumbnail sync, or final CTA)
☑ styles/design_languages.md — chosen language: <language_id>
☑ 31_typography.md — title/label/readability rules (required because overlay has text)
☑ <31_typography.md | 32_charts.md | 33_patterns.md> — content module for pattern=<pattern>
☑ `styles/text_axes.md` or `styles/chart_axes.md` — style recipe library
Decision: <overlay N/A | leave existing overlay | layout-only fix | minor additive fix | text/style preserve fix | extract/reuse current visual then rebuild | full replace>
```

**If a required module is not listed with ☑, Gate 4 is BLOCKED.** Do not draw, upload, or declare PASS.

### Gate 4 OVERLAY EXISTENCE + PRESERVE PROOF — before overlay edit/upload

This gate prevents forced overlays, forced titles, and unnecessary replacement of good visuals.

```text
Gate 4 OVERLAY EXISTENCE + PRESERVE PROOF:
Pattern/sub_mode: <pattern/sub_mode>
Visual field: <empty | quoted visual summary>
Existing overlay/spec: <none | visible | hidden | broken | unknown>
BEFORE screenshot shown: <yes|no> — local file path: <path>
Overlay needed check: <N/A no overlay | PASS overlay needed> — if pattern="narration_only", visual is empty, remotion_spec=="none", or narrator/background/caption already carries the scene, overlay is N/A and no overlay/title may be added
Existing visual type: <map | realistic image/photo | chart | diagram | illustration | typography-only | mixed | none>
Preserve current visual check: <PASS preserve | FAIL serious defect> — preserve good maps/photos/charts/diagrams/illustrations; missing title alone is not a defect
Serious defects found: <none | typo/wrong data | unreadable | dead-zone | face/caption collision | off-topic | broken render | severe clutter | other>
Repair ladder decision: <LEAVE | layout-only fix | minor additive fix | text/style preserve fix | extract/reuse current visual then rebuild | full replace>
Title creation check: <N/A no title needed | existing title only | title explicitly required by visual/pattern> — do not invent a title just to satisfy style
No-homogenization check: <PASS|FAIL> — decision preserves the scene's pattern/visual variety; not "big title + one object" by habit
Verdict: <N/A overlay audit | PASS preserve/repair | BLOCKED — replacement not justified>
```

If `Repair ladder decision` is `full replace`, print this additional proof before authoring/uploading:

```text
Gate 4 FULL REPLACE PROOF:
Current overlay value worth preserving: <map/photo/chart/diagram/illustration/text/none>
Why layout-only is insufficient: <specific reason>
Why minor additive fix is insufficient: <specific reason>
Why text/style preserve fix is insufficient: <specific reason>
Why extract/reuse current visual is insufficient: <specific reason>
Replacement risk check: <PASS|FAIL> — replacement will not simplify a good map/photo/chart/diagram into a worse homemade version
Strictly-better claim: <PASS|FAIL> — new overlay will be both more correct and higher quality than the local-shown BEFORE screenshot
Verdict: <PASS full replace allowed | BLOCKED — preserve/repair instead>
```

**If this proof is missing, Gate 4 is BLOCKED for any overlay upload that overwrites an existing overlay/spec.** Do not add an overlay to an N/A scene, do not invent a title for a scene that does not need one, and do not replace a good existing visual just to make the style consistent.

### Gate 4 A-ROLL LAYOUT PRIORITY PROOF — before overlay decision/authoring/upload on A-roll

This gate exists because the overlay/narrator tradeoff is decided in Gate 4, not after the overlay has already been accepted.

```text
Gate 4 A-ROLL LAYOUT PRIORITY PROOF:
☑ 10_mechanics.md — opened for the A-roll priority ladder before overlay decision
Scene class: <normal | special_endpoint_or_trust>
Narrator role: <primary | secondary>
Overlay role: <support | main_subject>
Current layout read: <full-canvas | shifted-full-canvas | shrunken/fallback | other>
Starting layout bias check: <PASS|FAIL|N/A> — if current narrator is shrunken/fallback, reset design baseline to Priority 1 full-canvas before judging overlay; the small narrator is not the baseline
Reset to full-canvas candidate before overlay judgment: <yes|no|N/A>
Overlay adjustments tested: <move | resize | simplify | rebuild | N/A> — current overlay is not fixed evidence
Full-canvas gate: <PASS|FAIL> — tested narrator full canvas first while moving/resizing/simplifying/rebuilding the overlay as needed; face clear, no dead zones, readable, caption clear
Priority 2 check: <PASS|FAIL|N/A> — pull narrator down + overlay above head; reason if rejected
Priority 3 check: <PASS|FAIL|N/A> — push narrator up + overlay over chest; reason if rejected
Priority 4 fallback check: <PASS|FAIL|N/A> — allowed only after 1–3 fail; narrator as large as possible, face large/clear/in safe_rect, X inside canvas, Y overflow allowed if it improves face size
Chosen priority: <1 | 2 | 3 | 4>
Overlay plan: <overlay N/A | leave existing | layout-only fix | minor additive fix | text/style preserve fix | extract/reuse current visual then rebuild | full replace | disable overlay> with placement rationale
Verdict: <PASS to continue Gate 4 | BLOCKED — revise overlay/narrator plan>
```

**If this is A-roll and this proof is missing, Gate 4 is BLOCKED.** Do not accept an existing shrunken narrator layout, draw/upload an overlay, or mark overlay review PASS. If the current narrator is shrunken/fallback and `Reset to full-canvas candidate before overlay judgment` is not `yes`, Gate 4 is BLOCKED. If choosing priority 4, the proof must show concrete reasons priorities 1–3 failed as joint narrator + overlay + caption compositions. Do not reject a full-canvas priority just because the old overlay placement touches the face/caption/dead zone; move/resize/simplify/rebuild the overlay first.

### Gate 4 TITLE GATE PROOF — after drafting a title/hero line and before upload

This gate exists because simply loading `31_typography.md` is not enough.

```text
Gate 4 TITLE GATE PROOF:
Title copy: "<exact title text>"
Takeaway source: <quote/talking_point words it encodes>
Copy correctness check: <PASS|FAIL> — no typo/grammar/diacritic/casing/number/symbol/proper-noun/domain-term error; phrasing is natural for the scene language
Semantic check: <PASS|FAIL> — states the scene takeaway, not just a panel label
Hierarchy check: <PASS|FAIL> — title is stronger than badges/panel labels
Weight/card check: <PASS|FAIL> — title uses 900-equivalent / Black typography and is open typography, NOT inside a card/chip/pill/panel/banner/black translucent box
Face-stack count: <8–15 copies> — same-fill copies inside the same title object; state why this count fits the font/word length; >15 copies = FAIL
Over-thick visual check: <PASS|FAIL> — rendered title keeps crisp letterforms, open counters/negative space, clean tracking, and visible Vietnamese diacritics; if duplicate fill makes it muddy/blobby/deformed, reduce count/offset or change font even inside 8–15
First-second punch check: <PASS|FAIL> — title is not merely readable; it is thick, vivid, high-energy, and graspable immediately at 280×498
Thickness/stroke check: <PASS|FAIL> — body is thick via 900-equivalent font + 8–15x face stack; visible text stroke is <=2px, thin/controlled, not over-stroked, and the face stack does not over-thicken the glyphs
Overlay preview check: <shown cheaply | skipped — no cheap renderer | N/A> — if shown, preview appears title-led; if skipped, post-upload composite screenshot is the mandatory visual proof
Verdict: <PASS to upload | REVISE title before upload>
```

**Any FAIL = title is not approved for upload.** Revise the overlay/title and repeat the gate. A skipped pre-upload preview is not a FAIL. After upload, the final screenshot gate must still PASS from the saved/shown 280×498 composite:

```text
Gate 6 TITLE SCREENSHOT CHECK:
Composite screenshot shown: <yes|no>
Screenshot check: <PASS|FAIL> — title matches the approved copy, has no visible typo/grammar/diacritic error, and is readable as the floor PLUS punchy in the first second: prominent, vivid, 900-weight/thick-bodied from the 8–15x face stack, open typography (not boxed in a card/panel), thin-outlined <=2px/not over-stroked, not muddy/blobby/over-thick from duplicate fill, and stronger than labels
Verdict: <PASS title | REVISE title/rebuild overlay>
```

### Gate 4 SECONDARY TEXT GATE PROOF — after drafting every non-title text/value/label/card line and before upload

This gate is separate from title: a title can PASS while labels FAIL.

```text
Gate 4 SECONDARY TEXT GATE PROOF:
Text inventory: <exact non-title strings: values, labels, card text, badge text, callouts>
Copy correctness check: <PASS|FAIL> — every string is typo-free, grammar-correct, domain-correct, and preserves required numbers/currency/%/proper nouns/diacritics
Size floor check: <PASS|FAIL> — each non-title text is >= ~30px on the 720 canvas OR deliberately simplified/removed; nothing depends on zoom
Contrast check: <PASS|FAIL> — each line uses solid high-contrast fill on a clean chip/card/quiet area; **secondary text has NO visible stroke/outline**; no dark-on-dark card text
Container/padding check: <PASS|FAIL> — every line fits inside its card/chip/bar area with generous interior padding; no clipped/overflowing words; no text touches, grazes, or visually crosses a border
Collision/Z-order check: <PASS|FAIL> — labels/values/badges do not overlap or cover each other; badges never sit on top of value labels unless both remain readable
Overlay preview check: <shown cheaply | skipped — no cheap renderer | N/A> — if shown, preview makes all secondary text appear readable; if skipped, post-upload composite screenshot is the mandatory visual proof
Verdict: <PASS to upload | REVISE labels before upload>
```

**Any FAIL = the overlay is not approved for upload**, even if the title gate passed. After upload/layout tuning, the final screenshot must pass:

```text
Gate 6 SECONDARY TEXT SCREENSHOT CHECK:
Composite screenshot shown: <yes|no>
Screenshot check: <PASS|FAIL> — every secondary label/value/card line matches the approved copy, has no visible typo/grammar/diacritic/domain error, is readable at 280×498 without zoom, has no visible outline/stroke, is not dark/muddy, is not overlapped/covered by another text/object, and has visible breathing room from its chip/card/bar border
Verdict: <PASS secondary text | REVISE labels/layout/rebuild overlay>
```

### Gate 9 MODULE COVERAGE GATE — before declaring Scene PASS and again before final video hand-off

```text
MODULE COVERAGE GATE:
☑ 00_ENTRYPOINT.md — loaded for run kickoff
☑ 01_critical_rules.md — loaded at run kickoff
☑ 02_jump_prevention.md — loaded at run kickoff
☑ 03_dod_gates.md — loaded at the start of this scene
☑ 04_principles_workflow.md — loaded at run kickoff for the whole-video pass
☑ 05_quality_qa_priority.md — loaded for §7 quality scan before declaring PASS
☑ 10_mechanics.md — loaded for this scene's data/layout/screenshot mechanics
☑ 30_overlay_core.md — loaded before Gate 4 overlay review/rebuild for this scene
☑ 20_background.md — loaded before Gate 5 background audit for this scene
☑ 31_typography.md — loaded because overlay has text OR N/A: <reason>
☑ 32_charts.md — loaded because chart/diagram pattern=<pattern> OR N/A: <reason>
☑ 33_patterns.md — loaded because non-chart pattern=<pattern> OR N/A: <reason>
☑ 40_thumbnail_cta.md — loaded because scene 2/thumbnail/final CTA endpoint OR N/A: <reason>
Verdict: <PASS module coverage | FAIL — load missing module and resume earliest invalidated gate>
```

**Missing required module = Gate 9 FAIL.** Loading a module once at the beginning does **not** mean the gate/work is complete; it only satisfies the coverage line. The actual gate still requires its proof, evidence, edit decision, and PASS verdict.

### Progress + verdict markers

- Progress (each gate): `→ Gate K: <doing…>` then `✓ Gate K: <result>`
- Scene end (verdict): `Scene N: PASS — ✓1…✓9` **or** `Scene N: FAIL — ✗K <what's missing>; fixing.`
- Human must act (record A-roll / final hand-off): a standalone `**[ACTION REQUIRED]**` block.

---

## PASS / FAIL verdict — declare it before EVERY scene hand-off

You may advance to the next scene ONLY after you state an explicit verdict:

- To say **"Scene N: PASS"** you must FIRST **scan all 9 DoD gates above AND the §7 Quality Standard (`ai_video_editor/05_quality_qa_priority`)**, and confirm **every** one is met. PASS is *earned by the scan* — never declared from memory or assumption.
- The PASS scan must include the local-visible screenshot evidence: "BEFORE shown: yes" and "AFTER/final shown: yes". If either is missing, PASS is forbidden.
- Batch/contact-sheet/gallery/table/script/bulk API outputs are not PASS evidence. If that is the only evidence, the scene status is `triaged_only`, not `PASS`.
- If any gate / §7 item is unmet → **"Scene N: FAIL — [list the failing gates]"**, fix them, then **re-scan and re-declare**. Loop until PASS.
- **Never advance on a FAIL, and never advance with no verdict at all** (an un-verdicted scene = not done). The verdict line + its gate-by-gate ✓ is the last thing you post for a scene before starting the next.

---

## Final video hand-off — after the last content scene passes

- First run the **Pre-summary completion scan** (Critical Rule 12d in `ai_video_editor/01_critical_rules`): every content scene has `Scene N: PASS`, Gate 1–9 checked, Module Coverage Gate PASS, Background Audit Ledger complete, scene 2 thumbnail sync complete, final CTA endpoint handling complete for the last content scene, and no unhandled `[ACTION REQUIRED]` item hidden.
- If the scan finds missing major work, do that work now. Do not write a "done" summary as a substitute for completing it. If the user explicitly asked for only triage or only selected-scene fixes, hand off as `partial_triage_only` / `partial_fix_only`, not complete.
- Do **not** revisit the thumbnail; it was completed by the immediate post-scene-2 sync gate.
- Pull/keep the `review_url` for the video.
- Send the user a Telegram/self-notification (WideCast self-notify tool, with email fallback if Telegram is not connected) saying the edit is complete and including the `review_url`.
- In chat, give a short summary of what was changed/fixed. Keep it concise; do not replay every gate.
- Ask exactly one export question only after every content scene has individual PASS: `Render/export the final MP4 now, or do you want to review the scenes first?`
- Do **not** call `export_video` until the user explicitly confirms render/export in the current conversation turn.

If any content scene is not individually PASS, replace the completion hand-off with:

```text
Run status: <partial_triage_only | partial_fix_only>
Scenes with individual PASS: <scene ids or none>
Scenes not individually PASS: <scene ids + missing proof/gates>
Export/render: blocked until every content scene has Scene N: PASS
```

Do not ask the render/export question in this partial state.

---

## Pre-checklist setup — determine 2 things

### (a) Video type — `faceless` or with a narrator?

- `faceless=true`: there is NO narrator → **drop the face-clearance condition**, but **still keep the safe-zone condition**. Place text based on **context + the current background visual**, and check that the text does not **sink into the same color tone as the background** (`ai_video_editor/31_typography`).
- `faceless=false` (has a narrator): keep the `narrator_face` clearance condition as usual (Steps 3–4 in `ai_video_editor/10_mechanics`).

### (b) Has the A-roll scene been recorded yet?

If `arollUrl`/`mediaUrl` still points to a library placeholder image (`statics/aroll_*.png`, `mediaType=image`) → **not recorded yet**. Still edit layout/overlay normally (the placeholder is full-canvas, face in the upper part), but **REMIND the user to complete the narrator** via one of 3 ways — **each scene at most 20 seconds** (all 3 ways are capped at 20s, since each scene should only be ≤20s):

1. **WideCast's built-in teleprompter** — available, convenient, high quality; **RECOMMENDED** to preserve authenticity.
2. **Upload a file** with the narrator's face + voice (`modify_scene` (I) `narrator.upload_video`).
3. **AI generate from a single photo** — a feature available in WideCast.

---

## Step 1 — Check Text and STT Errors

Read the scene's `text` field in the context of the whole video.

Look for:

- spelling errors
- typos
- STT mishearing errors
- wrong industry terminology
- wrong proper names
- wrong figures
- sentences missing or having extra words
- sentences that are grammatically correct but wrong in meaning for the topic
- a caption that doesn't match the audio/context content

If wrong, fix `text` with `modify_scene` branch (K) Segment text correction (keeping audio timing) before doing layout or visuals.

If the correction is a domain term, proper noun, number, symbol, or entity (for example `Living Church` → `Living Trust`, `95` → `95%`, a company/person/product name), run a **semantic field sweep** before leaving Gate 1:

- check/update `text`, `quote`, `talking_point`, `visual`, `keyword`, `pattern`/`sub_mode` when relevant;
- note any overlay text that may still be baked into `remotion_spec` and must be verified in Gate 4/7 screenshots;
- re-pull `video_data` after metadata/text edits so later gates use the corrected context.

Gate 1 does **not** clear overlay copy by itself. A caption may be fixed while a title/label/value baked into `remotion_spec` or a newly authored overlay source still has a typo, grammar issue, missing diacritic, wrong currency, wrong number, or wrong term. Gate 4 must inventory and proofread the exact visible overlay strings again, and Gate 7 must confirm the final screenshot still shows correct copy.

Do not let the caption fix become the whole task. A term fixed in `text` but still wrong in `quote`/`visual`/overlay is a Gate 1 failure, not a partial pass.

Do not edit based on personal feeling if you're not sure of the context. Rely on the full script, the topic, `visual`, `keyword`, and the neighboring scenes.

---

## Step 2 — Understand the Scene's Role

Before adjusting visual/layout, the agent must understand what job this scene is doing; read:

- `type` (HOOK / STAT / KEY POINT / DATA / FACT / CALL TO ACTION / thumbnail)
- `pattern` (15 values) **and** `sub_mode` (when `pattern=illustration`)
- `visual`, `keyword`, `quote`, `talking_point`, `text`
- `show_narrator` / `active_roll`
- `mediaUrl` / `mediaType`, the thumbnails
- `overlay.*` and `narrator_face`

The goal is to understand what idea the scene is trying to convey, whether the current visual serves that idea, and whether the scene wants a grid or a real background — **decided by sight, not by the pattern label** (`ai_video_editor/20_background`).

> **Step 3 (audit layout with `scene_geometry`) and Step 4 (the A-roll layout priority ladder) → `ai_video_editor/10_mechanics`.** Then the per-scene visual work: background → `20_background`; overlay → `30_overlay_core` (+ the matching style module).
