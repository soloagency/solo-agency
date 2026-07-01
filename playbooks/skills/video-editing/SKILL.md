# AI Video Editor Playbook

Version: `modular-2.0` · This file is the **MASTER INDEX**. It is intentionally small so every host/MCP runtime can deliver it without hitting per-tool-call output caps. Every detail — rules, jump-prevention triggers, DoD gate templates, principles, workflow, quality bar, priority order — lives in **separate modules** under `ai_video_editor/`. Reach a step → open the matching module → then act.

The goal of the AI video editor is to edit each scene so that the final video has **correct content, a clear face, clear text, the right visuals, good layout, and consistent quality from beginning to end.**

> **Overarching principle — name the field, do not guess.** Stick to the exact field name in the data: `show_narrator=true` (not "the scene has a narrator"), `overlay.<sub>.visible=true` (not "it has an overlay"). All geometry (safe zone, narrator face, overlay position) is precomputed — call `scene_geometry` instead of estimating coordinates.

---

## 🛑 HOW TO USE — open a module before you do its step

This file is an INDEX, not the manual. **Opening a module is a REQUIRED ACTION** — when you reach a step, you MUST load the module named for it BEFORE doing that step, every time, even if you think you remember it.

### Two transports — both ALWAYS LIVE, never cached

- **MCP transport** (Claude/ChatGPT/Codex MCP servers, plain HTTP) — call `widecast_get_editing_skill(module='<id>')` where `<id>` is the path without `.md`. Examples: `widecast_get_editing_skill(module='ai_video_editor/01_critical_rules')`, `widecast_get_editing_skill(module='ai_video_editor/10_mechanics')`. First call (no `module`) returns this SKILL.md + live `available_modules[]` index. Server emits `Cache-Control: no-store`; `/app/*` bypasses Cloudflare. If a call returns 404 `module_not_found`, recall with no args to refresh the index.
- **Anthropic Skill upload transport** (`video-editing.zip` mounted locally) — use the host's local `Read` tool with the path relative to skill root: `Read("ai_video_editor/01_critical_rules.md")`.

**Stable rule across both transports:** reach a step → open the module → act. Memory of a module loaded earlier does NOT replace re-loading it. Both transports are cheap.

> **This is exactly how the background audit gets skipped:** an agent reads the master, treats a scene as "an overlay geometry task", and never opens `20_background` — so the whole background/B-roll branch silently disappears. **The fix is mechanical: reach a step → open its module → then act.** If you did not open the module, you have not done the step.

**Stable TEXT markers** (icons render differently across AI apps — the TEXT is the source of truth): use these literal markers; an emoji (⭐ ✓ → ○ !) MAY decorate but never replaces the text:

- `[ACTION REQUIRED]` — a standalone block whenever the human must do something (record A-roll, approve, run a command, final hand-off).
- `Scene N: PASS` / `Scene N: FAIL — …` — the per-scene verdict.
- `No action required.` — when a hand-off needs nothing from the human.

### Run kickoff — load these 5 core modules FIRST, before scene 1

The 5 modules below carry the rules + workflow that apply across the whole run. Load all five at the START of every run (or whenever you do a Gate Resume Scan after a detour):

1. **`ai_video_editor/01_critical_rules`** — 14 critical rules that hold across every scene + the self-audit checklist run before each reply.
2. **`ai_video_editor/02_jump_prevention`** — "about to do X → STOP, do Y first" interrupt list.
3. **`ai_video_editor/03_dod_gates`** — per-scene Definition of Done (9 gates) + every template block (Gate 4 module-load proof, Gate 4 A-roll layout priority proof, Gate 4 title proof, Gate 4 secondary text proof, Gate 5 background proof, Gate 6 screenshot checks, Gate 9 module coverage).
4. **`ai_video_editor/04_principles_workflow`** — §1 general principles, §2 whole-video workflow (initial context pass + Background Audit Ledger init), §10 reminders.
5. **`ai_video_editor/05_quality_qa_priority`** — §7 Quality Standard, §8 video-level QA, §9 priority order for gate conflicts.

These 5 + `ai_video_editor/00_ENTRYPOINT` are the kickoff set. Per-scene modules (10/20/30/31/32/33/40) load at the step that needs them.

---

## ⬇ LOAD MAP — reach a step → open the matching module

The **Module id** column is what you pass to `widecast_get_editing_skill(module=...)` (MCP) or what to `Read` (upload transport — append `.md`). One row = one required load.

| When you reach this step | Module id to load |
|---|---|
| Run kickoff — every run | **`ai_video_editor/00_ENTRYPOINT`** + the 5 core modules above |
| Anything that uses one of the 14 critical rules | **`ai_video_editor/01_critical_rules`** (already loaded at kickoff; re-load on resume) |
| About to take any action you might jump past | **`ai_video_editor/02_jump_prevention`** |
| Start of every scene — print the DoD plan + gate templates | **`ai_video_editor/03_dod_gates`** |
| Before declaring `Scene N: PASS` — scan §7 against the scene | **`ai_video_editor/05_quality_qa_priority`** |
| Reading scene data · coordinates · the 13 `modify_scene` branches · A-roll layout priority ladder · how to look (screenshot) | **`ai_video_editor/10_mechanics`** |
| Deciding the background (grid vs real) · searching · evaluating · applying footage | **`ai_video_editor/20_background`** |
| About to (re)build or apply ANY overlay (internal vector model, rebuild threshold, `data-wc-*`, reuse-a-photo, verify) | **`ai_video_editor/30_overlay_core`** — FIRST for any overlay |
| Choosing the overlay's design language (style direction, not QA standard) | **`ai_video_editor/styles/design_languages`** |
| Overlay has TEXT (title/label/value/quote) | **`ai_video_editor/31_typography`** + **`ai_video_editor/styles/text_axes`** |
| Pattern is a CHART (`single_metric`/`bar_chart`/`proportion_chart`/`trend_chart`/`structural_diagram`) | **`ai_video_editor/32_charts`** + **`ai_video_editor/styles/chart_axes`** |
| Pattern is OTHER (`map_chart`/`comparison_table`/`timeline_events`/`checklist_tips`/`quote_card`/`illustration`/`hybrid_vertical`/`real_entity`/`typography_only`/`narration_only`) | **`ai_video_editor/33_patterns`** |
| Scene 2 (opening poster) · post-scene-2 thumbnail sync · last content/CTA scene | **`ai_video_editor/40_thumbnail_cta`** |

**Adding modules later — fully automatic, ZERO formatting required.** Drop a new `.md` file anywhere under `widecast/skills/video-editing/` and it appears in the live `available_modules[]` index returned by the entry call. The server auto-generates `title` (first H1 → first H2 → first content line → filename basename) and `summary` (first ~200 chars of meaningful content). No code change, no SKILL.md edit, no required formatting.

If you see an available module whose `title`/`summary` matches a step that this table doesn't cover yet, load it. Treat the live `available_modules[]` as the source of truth; this table is the curated default chain.

---

## ⭐ CRITICAL RULES — 1-line headlines (full text → `ai_video_editor/01_critical_rules`)

Load the module for the full text + nuance. These headlines are reminders, not the rules themselves.

0. **Visual evidence gate.** Every image used as evidence — screenshots (`scene_inspector` → `curl` `result.screenshot.url` → local file), found media, generated images, and cheap local overlay previews when available — must be saved locally and SHOWN visibly to the user BEFORE the agent judges/edits/uploads from it. Overlay previews are opportunistic; the mandatory overlay truth is the post-upload composite screenshot.
1. **Name the field, never guess.** Selector = `voice_file` (not `id`). After every `modify_scene`, re-pull `video_data`/`scene_geometry` to confirm saved.
2. **Runtime = autonomous, end-to-end.** Work scene 2 → last content scene in ONE pass. Never pause to ask. No `A or B?` questions to the user.
2a. **Decision protocol — choose, don't defer.** Priority: content correctness → face preservation → readability → safe-zone/caption → aesthetic → minimal edit.
2b. **WideCast edit trigger = full autonomous run.** "edit this video" + a WideCast URL/`topic_id` = full audit + fix. Never ask scope.
3. **Decide by SIGHT, not by `pattern`.** Visual calls need a local-shown screenshot. `scene_geometry` never substitutes for looking.
4. **Overlay = transparent internal vector, hosted, uploaded, then screenshot-verified.** Safe box x∈[36,684], y∈[128,960]. Show a local overlay preview only when the environment already supports it cheaply; never expose the internal format to normal users.
5. **ONE atom = ONE object** (`<g data-wc-object>`). Atomize, never clump. Co-appear via shared `data-wc-delay`.
6. **Font: HEAVY family** (e.g. `"<Family> Black"`). One font + accent per video; vary between videos.
6a. **Overlay copy correctness is its own gate.** Every visible string proofread; typos/grammar/wrong currency/wrong term = FAIL.
7. **Diversify the LOOK.** Load the style library; reproduce a real look (gradient/glossy/3D/metallic/…); never ship flat-only.
8. **A-roll: face is sacred.** Never edit `narrator_face`; solve narrator + overlay together. A full-canvas narrator trial cannot fail because the current overlay is in the wrong place; move/resize/simplify/rebuild the overlay before shrinking the narrator. Final CTA scene: narrator-primary + typography-led CTA.
9. **Grid ≤ 3 scenes/video, all sharing ONE grid.** Default to real background otherwise.
10. **Realistic photos are REUSED, never "drawn".** Add to a good overlay via `modify_scene` (M) `remotion.add_element`.
11. **Show found media in chat BEFORE looking/evaluating/applying.** No private preview first.
12. **Every scene = overlay review (Gate 4) AND background audit (Gate 5).** Two separate passes. Never skip the background.
12a. **Endpoint scenes (scene 2 + thumbnail + final CTA) are special.** Load `40_thumbnail_cta` for both. Sync thumbnail immediately after scene 2 PASS.
12b. **Gate 5 requires a printed BACKGROUND PROOF.** No proof = scene not done.
12c. **Gate Resume Scan after any detour/fix.** A fix is not a verdict; rescan from the earliest invalidated gate.
12d. **Pre-summary completion scan.** No summary/Telegram/export until every scene PASS + Module Coverage Gate + ledgers complete.
13. **The master is an INDEX — load each module BEFORE its step.** Memory ≠ re-loading.
13a. **Module Coverage Gate — missing playbook = not done.** Gate 9 proves required playbooks loaded.
14. **Announce plan + report progress.** Vertical 9-gate checklist at scene start, gate-by-gate progress, ✓/✗ recap + `Scene N: PASS|FAIL` verdict at scene end.

---

## ⛔ JUMP-PREVENTION TRIGGERS — 1-line interrupts (full list → `ai_video_editor/02_jump_prevention`)

If you're about to do any of these, STOP and do the prerequisite first:

- start a scene → load `10_mechanics`
- handle scene 2 / thumbnail / final CTA → load `40_thumbnail_cta`
- ask "what kind of edit?" after a WideCast URL → STOP. Scope is **Full audit + fix**.
- audit background → load `20_background`
- start Gate 6 without printed Gate 5 BACKGROUND PROOF → STOP, run Gate 5
- finished any detour/fix and feel ready to summarize → run Gate Resume Scan
- (re)build overlay → load the whole chain `30_overlay_core` + `31`/`32`/`33` + `styles/*`
- visual call from `scene_geometry` alone → STOP, pull screenshot first
- act on a screenshot you have not SHOWN locally → show it first
- about to spend time rendering/converting an overlay preview → only do it if the environment already supports it cheaply; otherwise skip pre-upload preview and verify via the post-upload composite screenshot
- ask the user to choose/approve during runtime → STOP, decide yourself
- reject A-roll full-canvas because the current overlay touches the face / caption / dead zone → STOP, solve narrator + overlay together first
- declare `Scene N: PASS` without scanning 9 DoD gates + §7 → run the scan
- declare PASS without naming Gate 5 verdict (`PASS keep` / `PASS grid-by-design` / `FIXED + PASS`) → STOP
- declare PASS with any missing required module in MODULE COVERAGE GATE → STOP
- move to next scene without stated `PASS`/`FAIL` → declare verdict first
- write final summary / hand-off / export question → run Pre-summary completion scan
- final-handoff without complete Background Audit Ledger → STOP
- resuming/continuing a run → re-load modules, never work from memory

---

## ⭐ DEFINITION OF DONE — 9 gates per scene (full DoD + all template blocks → `ai_video_editor/03_dod_gates`)

Print this 9-gate checklist VERTICALLY at the start of every scene; tick ✓/✗ at the end and state `Scene N: PASS|FAIL`.

1. ☐ **Text / STT** checked in whole-video context, fixed if wrong.
2. ☐ **Role** understood — `type` · `pattern`/`sub_mode` · `visual` · `quote` · `talking_point`.
3. ☐ **BEFORE screenshot** pulled, downloaded with `curl`, SHOWN locally, THEN evaluated.
4. ☐ **Overlay reviewed/rebuilt** — load `30_overlay_core` first + endpoint/typography/content modules + style lib; for A-roll print Gate 4 A-ROLL LAYOUT PRIORITY PROOF before overlay decisions, proving narrator + overlay were solved together and the overlay was not treated as fixed; print Gate 4 MODULE LOAD PROOF + TITLE GATE PROOF + SECONDARY TEXT GATE PROOF.
5. ☐ **Background audited** — load `20_background` first; print Gate 5 BACKGROUND PROOF with two local-visible images.
6. ☐ **Final composition tuned** — layout + safe zone + dead-zone + face clearance + caption + balance.
7. ☐ **AFTER screenshot** pulled, downloaded, SHOWN locally, evaluated for all of above.
8. ☐ **Server-saved** — re-pulled `video_data`/`scene_geometry` to confirm persisted.
9. ☐ **MODULE COVERAGE GATE** — print proof; PASS only if every required playbook loaded at the correct step.

All 9 checked + §7 (`05_quality_qa_priority`) scan met → `Scene N: PASS`. Otherwise `Scene N: FAIL — ✗K …`; fix and re-scan.

---

## ✅ SELF-AUDIT — before every reply (full version → tail of `ai_video_editor/01_critical_rules`)

Silently confirm — and fix any "no" before replying:
- Did I open the module(s) for the action I'm taking THIS turn, not work from memory?
- Every image I used as evidence — saved locally AND shown to the user?
- Visual call from a SCREENSHOT, not `scene_geometry` alone?
- Which DoD gate am I on; did I announce progress?
- Ending a scene? Scanned 9 gates + §7? Stated `Scene N: PASS|FAIL`?
- MODULE COVERAGE GATE clean (no missing required playbook)?
- Field discipline: `voice_file` selector + re-pull to confirm saved?
- Finished a detour? Ran Gate Resume Scan?
- About to summarize/hand-off/export? Ran Pre-summary completion scan?
- Human must act? Used a standalone `[ACTION REQUIRED]` block?

---

## next_action

**Right now → load `ai_video_editor/00_ENTRYPOINT` AND the 5 core modules (`01_critical_rules`, `02_jump_prevention`, `03_dod_gates`, `04_principles_workflow`, `05_quality_qa_priority`).** Then pull `video_data`, do the §2 whole-video context pass, and start scene 2. Do not start scene 1; scene 2 is the first real content scene.
