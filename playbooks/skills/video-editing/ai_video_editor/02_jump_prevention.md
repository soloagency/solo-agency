# Jump-Prevention Rules — catch yourself BEFORE the action

Each line below is an **interrupt** on what you are *about to* do. If the trigger matches what you're about to do next, **STOP**, do the `→` part first, then resume your task.

This list is deliberately redundant with the Critical Rules (`ai_video_editor/01_critical_rules`) and the per-scene Definition of Done (`ai_video_editor/03_dod_gates`) — **redundancy is what stops misses**. Load this module at the START of every run, and re-load it whenever you do a Gate Resume Scan after a detour.

---

- A **module read errored / truncated / returned only a preview** ("output too large", "persisted output", 404, timeout, partial) → STOP. That module is NOT loaded. Re-read it in chunks to the end (quote its last line in the LOAD LEDGER) before the step that needs it. Never work from the preview, never skip it, never mark it loaded. (This is exactly how `03_dod_gates` gets skipped when it is large.)

- About to **start a scene** → first load `ai_video_editor/10_mechanics`.

- About to **handle the first real scene after the thumbnail** → first load `ai_video_editor/40_thumbnail_cta`; it is the opening poster frame even if its `type`/`pattern` is not `thumbnail`, and it needs a named endpoint poster style rather than a normal card/text-bar overlay.

- About to **handle `type="thumbnail"`** → only do this as the immediate sync gate after scene 2 PASS (unless the user explicitly asks for a thumbnail-only debug/edit). First load `ai_video_editor/40_thumbnail_cta`; clone/verify the opening poster identity, then continue to scene 3.

- About to **handle the last non-thumbnail/content scene or `type="CALL TO ACTION"`** → first load `ai_video_editor/40_thumbnail_cta`; treat it as the closing CTA endpoint, with one clear action, typography stronger than decorative objects, and poster-grade composition rather than normal inside-scene card styling.

- About to **author, upload, apply, or approve an endpoint overlay** for scene 2/opening poster, thumbnail sync, or final CTA without a printed `Gate 4 ENDPOINT DESIGN VARIANT PROOF` → STOP. Load `styles/design_languages`, choose a language + endpoint archetype, state variant tokens, and pass the anti-template check before touching the overlay. Reusing the same red side-bar/double-underline/giant-outline motif across unrelated videos is not a proof of style; it is a process failure.

- About to **ask what kind of edit / edit scope** after the user gave a WideCast URL or `topic_id` (`What kind of edit do you want?`, `Full audit + fix / Specific scenes / Backgrounds / Text`, `Before I touch anything...`, `edit this video can mean different things`, `Asking Edit scope`, `request_user_input`) → STOP. The scope is already **Full audit + fix**. Do not ask. Continue the autonomous run from `video_data` → whole-video context → scene 2.

- About to **audit/choose the background** → first load `ai_video_editor/20_background`. (Background is its OWN pass immediately after overlay, never folded into the overlay.)

- About to **start Gate 6 / final composition work** and this scene does not yet have a printed `Gate 5 BACKGROUND PROOF` with a PASS/FIXED verdict → STOP. Run Gate 5 first. Overlay urgency, obvious text errors, or "the screenshot looks fine" do not waive background proof.

- About to **rebuild, replace, upload, disable, or restyle an overlay** because background search failed, the background is wrong, geo failed, or grid fallback was chosen → STOP. Gate 5 owns background only; keep overlay/remotion unchanged unless Gate 4 already printed an independent overlay FAIL proof for this exact scene.

- About to **replace or re-search the background** because the overlay is ugly, unreadable, in a dead zone, or stylistically weak → STOP. Overlay/layout defects are fixed in Gate 4/Gate 6; change background only when Gate 5 independently fails.

- About to **declare Gate 6 PASS / Scene PASS** while a visible overlay/remotion object exists and there is no printed `Gate 6 DEAD-ZONE PROOF` for the latest overlay/layout state → STOP. Pull fresh `scene_geometry`, list checked object ids/layout_ids, verify `dead_top`, `dead_bottom`, and caption reserve, fix if needed, then repeat the proof.

- About to **declare Gate 7 / Scene PASS** while visible overlay, chart, label, title, generated image, or image-baked message text exists and there is no printed `Gate 7 RENDERED IMAGE TYPO/GRAMMAR CHECK` using a local PNG from MCP `widecast_scene_inspector action="overlay_poster"` for the current `remotion_spec` → STOP. Call the MCP action with topic `id`, scene `voice_file`, and `activate:true`, download/show the returned URL once, read the exact words from the poster itself, compare against intended copy, and fix any typo/grammar/diacritic/glyph/pseudo-text error before PASS. If the poster result is unavailable, state the composite fallback; do not construct the poster URL manually and do not launch a browser/local converter just to make one.

- Just finished any **detour/fix** (wrong term, typo, missing number/symbol, bad overlay word, covered face, wrong thumbnail, background swap, geo mismatch, layout tweak, tool/debug issue, or cross-scene small fix) and feel ready to summarize/handoff/move on → STOP. Run the Gate Resume Scan from Critical Rule 12c (`ai_video_editor/01_critical_rules`) and continue from the earliest unchecked or invalidated gate; a fix is not a scene/run verdict.

- About to **(re)build, apply, or approve an overlay on A-roll** (`show_narrator=true`) without a printed Gate 4 A-ROLL LAYOUT PRIORITY PROOF → STOP. Load `ai_video_editor/10_mechanics`, run the full-canvas-first ladder inside Gate 4, and decide the overlay/narrator tradeoff before drawing/uploading/accepting the overlay.

- About to **reject an A-roll full-canvas or shifted-full-canvas priority** because the current overlay touches the face, caption, or dead zone → STOP. The overlay is movable/rebuildable. First solve narrator + overlay + caption together: move objects/group, resize the group, simplify/rebuild the overlay, then judge the priority from a local-shown screenshot.

- About to **(re)build or apply an overlay** → first load the whole LOAD CHAIN: `ai_video_editor/30_overlay_core` + `ai_video_editor/10_mechanics` when A-roll + the matching `31`/`32`/`33` + its `styles/*`. Stopping at `30_overlay_core` = flat / off-pattern.

- About to **make ANY visual call** (grid-vs-real, regenerate-or-leave, readable?) from `scene_geometry` alone → STOP: pull the screenshot, save + show it locally, judge from the IMAGE. Geometry never substitutes for looking.

- About to **act on a screenshot / found media you have not SHOWN** locally → save + show it first (Critical Rules 0/11).

- About to spend time **rendering/converting an overlay preview** before upload → STOP. Only show a pre-upload overlay preview if the runtime already has a cheap direct display/conversion path; otherwise upload/apply the overlay, verify composition via the post-upload composite screenshot, and verify rendered text via MCP `widecast_scene_inspector action="overlay_poster"` when visible overlay/message text exists (Critical Rule 4).

- About to **ask the user to choose/approve an option during runtime** → STOP. Use the decision protocol in Critical Rule 2a, choose the best option yourself, act, and only report the decision/proof. Do not turn uncertainty into a user question.

- About to **declare `Scene N: PASS`** without scanning all 9 DoD gates + §7 Quality Standard (incl. module coverage + final composition + the explicit Gate 6 DEAD-ZONE PROOF) → run the scan first; PASS is earned by the scan. The DoD lives in `ai_video_editor/03_dod_gates`; the Quality Standard lives in `ai_video_editor/05_quality_qa_priority`.

- About to **declare `Scene N: PASS` from a batch/contact-sheet/gallery/table/script/bulk API result** → STOP. That result is triage only. Pull/show the scene's own BEFORE and AFTER/final screenshots, complete all 9 gates, include Gate 5 background proof and Module Coverage Gate, confirm server-saved, then declare PASS only for that exact scene.

- About to **declare `Scene N: PASS`** but the end checklist cannot name the Gate 5 verdict (`PASS keep`, `PASS grid-by-design`, or `FIXED + PASS`) → STOP. The background audit is missing even if overlay/composition passed.

- About to **declare `Scene N: PASS` / final handoff** but the `MODULE COVERAGE GATE` has any missing required module → STOP. Load the missing module at the correct step, resume the earliest invalidated gate, then re-scan.

- About to **move to the next scene** with no stated `PASS`/`FAIL` verdict → declare the verdict first. No verdict = scene not done.

- About to **write a final summary / completion report / Telegram completion message / export question** → STOP and run the Pre-summary completion scan (Critical Rule 12d). If any scene/gate/ledger row is incomplete, do that work first instead of summarizing.

- About to **ask for render/export or call `export_video`** while any content scene lacks its own `Scene N: PASS` → STOP. The only allowed status is `partial_triage_only` / `partial_fix_only`; do not ask for render/export and do not call export.

- About to **handoff after triage or a partial set of fixes** → STOP. Say the status is `partial_triage_only` or `partial_fix_only`, then list scenes with individual PASS separately from scenes not individually PASS. Do not imply full video completion.

- About to **final-handoff a video** without a complete per-scene background-audit ledger for every content scene → STOP. Say "background audit not complete" and run the missing Gate 5 rows before hand-off/export.

- User asks **"did you audit backgrounds?" / "are backgrounds suitable?"** → answer from the Gate 5 ledger only. If any content scene lacks a Gate 5 proof row, answer "not yet" and continue the background audit; do not infer from memory or from overlay screenshots.

- About to call **`modify_scene` / `upload_asset` / `export_video`** → STOP and confirm, immediately above the write call: (1) run-level KICKOFF LOAD LEDGER is printed with a valid last-line for each kickoff module; (2) this scene has printed its plan + Gate 3 BEFORE + Gate 4 SCENE LOAD LEDGER for the modules this scene type needs (background→`20_background`; overlay-with-text→`30_overlay_core`+`31_typography`; endpoint→`40_thumbnail_cta`). Any missing → not allowed to write; print it first. (Announce ≠ pause: print, then keep working.)

- **Resuming / continuing a run** → do NOT work from memory: re-load the modules for the step you are on.
