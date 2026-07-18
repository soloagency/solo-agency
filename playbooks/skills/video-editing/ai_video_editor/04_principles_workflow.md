# General Principles + Whole-Video Workflow + Agent Reminders

Load this module at **run kickoff**, alongside `01_critical_rules` and `02_jump_prevention`. It carries §1, §2, and §10 from the original master playbook — general principles, the whole-video workflow pass, and the long-form reminders list.

---

## 1. General Principles

1. Always start by pulling the entire video script (`video_data`).
2. Read the whole video context before editing each scene.
3. Concatenate the `text` of the segments into the full script to understand the topic, terminology, proper names, industry, offer, disclaimer, and tone.
4. Go through scenes in the order they appear (start at the first real scene after the thumbnail, usually `id=2`). Every scene is routed the same way — there are no special "opening poster" / "thumbnail sync" / "closing CTA" authoring passes anymore; the pipeline produces overlays and the agent only audits the two blind spots.
5. Read the data first: `text`, `visual`, `pattern`, `sub_mode`, `keyword`, `quote`, `talking_point`, `mediaUrl`, `show_narrator`. **`pattern`/`sub_mode` route the conditional gates** (Gate 3 background applies when non-grid + narrator not covering; Gate 4 typo applies when overlay text is image-model-generated) — this routing is a pure data read, no screenshot.
6. A **local-shown image** is how you SEE the two blind spots: the background **plate** (Gate 3) and the overlay **poster** (Gate 4). There is NO BEFORE composite look — the server guarantees placement, so you never pull a screenshot just to "check the layout". Only pull an AFTER composite to confirm a fix you actually made. Don't pull reflexively after every tiny edit (each pull costs tokens + bandwidth).
7. After each edit, pull `video_data`/`scene_geometry` again to check the result was saved on the server.
8. When doing an interactive review with the user, show the important images/thumbnails/layouts and your reasoning so the user can evaluate alongside you.
9. **Visual evidence before agent judgment (not just before applying).** Any visual artifact the agent is about to use as evidence — screenshot, found media, B-roll candidate thumbnail, contact sheet, downloaded image, generated image, extracted spec image, style/reference preview, or **cheap local overlay preview when available** — must be **downloaded/saved locally and rendered viewable in chat BEFORE the agent evaluates it, chooses from it, edits from it, uploads it, or calls `modify_scene` because of it**. Looking privately first is a process error. A self-authored overlay uses an internal SVG source, but user-facing language is only **overlay** / **overlay preview**; do not mention the internal format to normal users. Pre-upload preview is opportunistic, not a blocker: show it only if the runtime can display/convert it cheaply without installing tools, launching a headless browser, or probing multiple render paths. If no cheap preview path exists, skip it and verify final placement/readability via the **post-upload screenshot** plus rendered text/glyph correctness via MCP `widecast_scene_inspector action="overlay_poster"` when visible overlay/message text exists (`ai_video_editor/30_overlay_core`), both local-shown before judgment when used as evidence.
   - **EXACTLY ONE USER-VISIBLE RENDER PER IMAGE — anti-double AND anti-zero.** Count renders the USER sees inline, not the agent's private views. Two failure modes, both forbidden: (a) **double** — on hosts where the agent's viewing tool already renders inline to the user (Codex `view_image`), that view IS the show; adding a markdown `![](path)` tag or a second attachment duplicates the picture in the user's feed; (b) **zero** — on hosts where the agent's viewing is private or click-to-open only (a `Read` the user sees only as a file card), the private look NEVER satisfies the show-gate; the agent must still produce the one user-visible render via `SendUserFile`(render)/`present_files`/inline attachment. A "shown: yes" proof backed only by a private Read is a false proof and voids the gate. Decide once per session which host type this is; from then on every evidence image gets exactly one visible render.
   - **THE SHOW MECHANISM — LOCAL FILE FROM `result.screenshot.url`, NOT BASE64 / BINARY / BROWSER.** For WideCast scene screenshots, call MCP `scene_inspector` / `widecast_scene_inspector` with `action="screenshot_scene_280x498"` as the source of truth for composition/layout. The tool must return `result.screenshot.url`. The agent must immediately download that URL to a local file with `curl -L -s -o <local>.jpg "<url>"`, then show the local file using the environment's local-file display mechanism (`SendUserFile` / `present_files` / local image attachment). **ABSOLUTELY DO NOT judge from the remote URL itself or show via an online URL** (S3/http) embedded in a widget/HTML. Sidecar JSON, request ids, truncated transcript dumps, base64, binary `ImageContent`, browser screenshots, and HTML galleries with online `<img>` are NOT sufficient for WideCast scene screenshots. Narrow exception: overlay typo/grammar/diacritic/glyph proof uses a local PNG downloaded from MCP `widecast_scene_inspector` with `action="overlay_poster"`, `id`, `voice_file`, and `activate:true`; never construct that URL manually.

---

## 2. Whole-Video Workflow

Before going scene by scene, the agent needs to do one pass of a video-level audit:

1. Pull the entire video data.
2. Build the full script by concatenating each scene's `text`.
3. Identify the topic, field, and primary audience of the video.
4. Note the important terms, product names, people's names, company names, the industry, and the phrases STT is likely to mishear.
5. Identify the video type: educational, sales, explainer, how-to, reaction, case study, news, or CTA.
6. Identify the tone: serious, humorous, warning, expert, friendly, or viral hook.
7. Lightly scan each scene's classification fields **for context only** (`type`, `show_narrator`, `pattern`+`sub_mode`, `overlay.*.visible`, `remotion_spec`) — enough to infer `faceless`, spot the thumbnail, and catch context-level script errors. **This is NOT a per-scene edit plan — make NO visual judgement and take NO screenshot here.**
8. Print a **SCENE ROSTER** — one row per content scene in play order: `#`, `id`, `voice_file`, `type`, and a blank `verdict` column — and state the **total content-scene count** explicitly ("N content scenes"). Mark the **first content row** as `opening` — that scene gets the Gate 4 OPENING POSTER CHECK (an aesthetic pass on the hook frame, since platforms may auto-extract it), unlike every other scene. Write the roster into a local **run_ledger file** (`<scratchpad>/run_ledger_<topic_id>.md`). Update it immediately after every `Scene N: PASS|FAIL` verdict and every applied write. **Inline: the next scene is the next unvisited roster row; delegation (see `06_subagent_protocol`): rows close in event order and the roster is the completeness tracker — either way, every row must end with a verdict.** On ANY resume/detour/compaction, `Read` the run_ledger back before continuing — never reconstruct from memory or a summary. The Pre-summary completion scan counts `PASS` rows against the roster total.
9. **Delegation check (threshold, not preference):** if the runtime can spawn subagents AND the video has MORE THAN 30 content scenes (or the user asked for parallel), the scene-editor pipeline in `ai_video_editor/06_subagent_protocol` is the run mode — load it now, export the steward data files from the kickoff `video_data` pull (`run_script.txt` + per-scene `record.json` + full snapshot — editors read these instead of re-calling `video_data`), ensure the skill is unzipped locally (editors read from `skill_root`, never re-download), call `widecast_edit_session action='start'`, and record `delegation mode: subagent (K=5 rolling)` in the run_ledger. Editors write their own scenes directly (server lock + session cache make that safe); the main agent never writes a scene, never views an image, and must `commit` the session before hand-off. **Inline is the default at ≤30 scenes** — record `inline — reason: ≤30 scenes` (or `no subagent capability` / `user asked single-agent` / `host policy hard-blocks spawning`).

Understanding the whole picture is mandatory because many per-scene errors cannot be detected by reading a single scene in isolation.

> **⭐ AFTER this light context pass, go ONE SCENE AT A TIME — do NOT pre-plan all scenes at once.** For each scene: Gate 1 text → Gate 2 route → (if it applies) Gate 3 background plate look → (if it applies) Gate 4 overlay poster typo table → Gate 5 confirm+save if edited → verdict. Route Gate 3/4 from data, take only the looks those gates need, then verdict and move on. (Producing an all-scenes intention table instead of working scene-by-scene is a process error.)
>
> **⭐ RUN END-TO-END — do NOT pause between scenes.** At runtime the user is NOT present (Critical Rule 2), so the agent works scene 2 → last content scene in one continuous pass, and **only stops at the very end** to hand off. Never stop mid-video to ask. (Per-scene pausing is dev-mode only.)

Example: in a video about insurance or estate planning, STT might write `Living Church` when the correct content should be `Living Trust`. Spelling and grammar aren't wrong, but it's wrong by context.

---

## 10. Reminders for the Agent

- Read the entire video before editing one scene; the full script catches per-scene errors invisible in isolation.
- Name fields precisely; use `voice_file` as the selector. After any edit, re-pull to confirm it saved.
- Don't edit `text` just because a sentence sounds odd — rely on whole-video context (`Living Church` → `Living Trust`).
- **The server owns placement.** Never audit dead-zone/face/composition, never run an A-roll layout ladder, never resize the narrator (`overlay.narrator.rect`), never edit `narrator_face`. Overlays are kept off the face and inside the safe zone for you.
- **Layer isolation.** A background fix touches ONLY `mediaUrl`/`mediaType`; it never authorizes an overlay change, and an overlay fix never authorizes a background change.
- **Gate 3 background (conditional).** Non-grid + narrator-not-covering scenes only: the clip must serve the sentence being spoken and match the target market's geo/currency/context. Grid stays grid (≤3/video, shared). Decide fit from the plate.
- **Gate 4 overlay typo (conditional).** Image-model-baked text only (illustration non-photo/chart/diagram/object). SVG/typography text is deterministic — never typo-check it. Fix a real typo by regenerating/replacing the image or its source, then re-pull the poster.
- **Don't replace good existing visuals.** A good map/photo/chart/diagram/illustration is preserved by default; only a genuine defect (wrong data, a typo, off-topic, broken render) justifies a fix, and the least destructive fix wins.
- An unrecorded A-roll narrator (`arollUrl`=`statics/aroll_*.png`) → **remind the user in a standalone `[ACTION REQUIRED]` block** to complete it: WideCast teleprompter (recommended, authentic) / upload a face+voice file / AI-gen from a photo — **each scene ≤20s**.
- **SHOW every evidence image once (Rule 0)** before judging — never a private `Read`, never a double. Plate look, poster look, AFTER look; nothing else.
- Each scene is complete on its own — there is no whole-video QA pass at the end.
