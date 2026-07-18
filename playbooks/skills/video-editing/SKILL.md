# AI Video Editor Playbook

Version: `modular-2.0` · This file is the **MASTER INDEX**. It is intentionally small so every host/MCP runtime can deliver it without hitting per-tool-call output caps. Every detail — rules, jump-prevention triggers, DoD gate templates, principles, workflow, quality bar, priority order — lives in **separate modules** under `ai_video_editor/`. Reach a step → open the matching module → then act.

The goal of the AI video editor is to audit each scene for the two things WideCast (a blind server) CANNOT judge, and fix them: **(1) whether the background clip actually fits what is being said and the target market's geography/context, and (2) whether image-model-generated text (illustrations/charts) has typos.** Everything mechanical — overlay placement, dead-zone avoidance, keeping overlays off the narrator face, whether a scene needs an overlay at all — is already guaranteed by the server; the agent does NOT re-verify it. Plus the always-cheap data checks: `text`/STT context correctness.

> **Overarching principle — name the field, do not guess.** Stick to the exact field name in the data: `show_narrator=true` (not "the scene has a narrator"), `pattern="illustration"` + `sub_mode` (not "it's an image"). Route the conditional gates from data, not from a screenshot.

---

## 🛑 HOW TO USE — open a module before you do its step

This file is an INDEX, not the manual. **Opening a module is a REQUIRED ACTION** — when you reach a step, you MUST load the module named for it BEFORE doing that step, every time, even if you think you remember it.

### Two transports — both ALWAYS LIVE, never cached

- **MCP transport** (Claude/ChatGPT/Codex MCP servers, plain HTTP) — call `widecast_get_editing_skill(module='<id>')` where `<id>` is the path without `.md`. Examples: `widecast_get_editing_skill(module='ai_video_editor/01_critical_rules')`, `widecast_get_editing_skill(module='ai_video_editor/10_mechanics')`. First call (no `module`) returns this SKILL.md + live `available_modules[]` index. Server emits `Cache-Control: no-store`; `/app/*` bypasses Cloudflare. If a call returns 404 `module_not_found`, recall with no args to refresh the index.
- **Anthropic Skill upload transport** (`video-editing.zip` mounted locally) — use the host's local `Read` tool with the path relative to skill root: `Read("ai_video_editor/01_critical_rules.md")`.

**Stable rule across both transports:** reach a step → open the module → act. Memory of a module loaded earlier does NOT replace re-loading it. Both transports are cheap.

> **Transport does not relax anything.** MCP tool, REST endpoint (`/v1/scene_inspector`, `/v1/scene_geometry`, `/v1/modify_scene`, `/v1/upload_asset`…), or local `Read`/`cat` — every gate, module load, proof, and the LOAD LEDGER is identical and equally mandatory. Mapping tool names grants NO latitude to skip or soften a proof. Reading with `cat`/`sed` instead of `Read` is fine; skipping because of transport is not.

> **This is exactly how the background audit gets skipped:** an agent reads the master, treats a scene as "an overlay geometry task", and never opens `20_background` — so the whole background/B-roll branch silently disappears. **The fix is mechanical: reach a step → open its module → then act.** If you did not open the module, you have not done the step.

**Stable TEXT markers** (icons render differently across AI apps — the TEXT is the source of truth): use these literal markers; an emoji (⭐ ✓ → ○ !) MAY decorate but never replaces the text:

- `[ACTION REQUIRED]` — a standalone block whenever the human must do something (record A-roll, approve, run a command, final hand-off).
- `Scene N: PASS` / `Scene N: FAIL — …` — the per-scene verdict.
- When a hand-off needs nothing from the human, end with next-action guidance per the root Next-Action Guidance Rule (for example suggest the render/export approval or the next pending step) instead of `No action required.`

### Run kickoff — load these 5 core modules FIRST, before scene 1

The 5 modules below carry the rules + workflow that apply across the whole run. Load all five at the START of every run (or whenever you do a Gate Resume Scan after a detour):

1. **`ai_video_editor/01_critical_rules`** — critical rules that hold across every scene + the self-audit checklist run before each reply.
2. **`ai_video_editor/02_jump_prevention`** — "about to do X → STOP, do Y first" interrupt list.
3. **`ai_video_editor/03_dod_gates`** — per-scene Definition of Done (5 gates) + template blocks (Gate 3 background proof, Gate 4 overlay-text typo table, module coverage).
4. **`ai_video_editor/04_principles_workflow`** — §1 general principles, §2 whole-video workflow (initial context pass + roster/ledger init), §10 reminders.
5. **`ai_video_editor/05_quality_qa_priority`** — §7 Quality Standard, §9 priority order for gate conflicts.

These 5 + `ai_video_editor/00_ENTRYPOINT` are the kickoff set; the KICKOFF LOAD LEDGER also lists `10_mechanics` because scene work starts immediately after kickoff. Per-scene modules (20/30/31/32/33/40) load only when a scene actually needs them; `06_subagent_protocol` loads before spawning any subagent.

**Before the first `modify_scene`/`upload_asset` of the run, print a KICKOFF LOAD LEDGER** (template in `ai_video_editor/03_dod_gates`): for each kickoff module report its line count and match it to `LOAD_MANIFEST.md`; a shortfall = truncated = NOT loaded → you are BLOCKED from writing. (Quote the last line only when no manifest is present.) This is the mechanical defense against skipping a module that errored with "output too large".

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
| Reading scene data · coordinates · the 13 `modify_scene` branches · how to look (screenshot) | **`ai_video_editor/10_mechanics`** |
| Gate 3 background audit (semantic/geo fit) · searching · evaluating · applying footage | **`ai_video_editor/20_background`** |
| About to (re)build or apply ANY overlay (internal vector model, rebuild threshold, `data-wc-*`, reuse-a-photo, verify) | **`ai_video_editor/30_overlay_core`** — FIRST for any overlay |
| Choosing the overlay's design language (style direction, not QA standard) | **`ai_video_editor/styles/design_languages`** |
| Overlay has TEXT (title/label/value/quote) | **`ai_video_editor/31_typography`** + **`ai_video_editor/styles/text_axes`** |
| Pattern is a CHART (`single_metric`/`bar_chart`/`proportion_chart`/`trend_chart`/`structural_diagram`) | **`ai_video_editor/32_charts`** + **`ai_video_editor/styles/chart_axes`** |
| Pattern is OTHER (`map_chart`/`comparison_table`/`timeline_events`/`checklist_tips`/`quote_card`/`illustration`/`hybrid_vertical`/`real_entity`/`typography_only`/`narration_only`) | **`ai_video_editor/33_patterns`** |
| About to spawn ANY subagent for scene work (scene editors, fix agents) · edit-session start/commit | **`ai_video_editor/06_subagent_protocol`** |

*(`30`/`31`/`32`/`33` + style libs load ONLY when you must fix an overlay defect — the server authors overlays and guarantees placement; you don't build/audit them routinely. `40_thumbnail_cta` is retired: endpoint/poster authoring is no longer an agent gate.)*

**Adding modules later — fully automatic, ZERO formatting required.** Drop a new `.md` file anywhere under `widecast/skills/video-editing/` and it appears in the live `available_modules[]` index returned by the entry call. The server auto-generates `title` (first H1 → first H2 → first content line → filename basename) and `summary` (first ~200 chars of meaningful content). No code change, no SKILL.md edit, no required formatting.

If you see an available module whose `title`/`summary` matches a step that this table doesn't cover yet, load it. Treat the live `available_modules[]` as the source of truth; this table is the curated default chain.

---

## ⭐ CRITICAL RULES — 1-line headlines (full text → `ai_video_editor/01_critical_rules`)

Load the module for the full text + nuance. These headlines are reminders, not the rules themselves.

0. **One user-visible render per image (anti-double AND anti-zero).** Any evidence image (background plate, overlay poster, AFTER composite) must be saved locally and produce EXACTLY ONE user-visible inline render before you judge from it — never two (don't view + also embed a markdown tag), never zero (a private `Read` the user only sees as a file card does NOT count as shown). Decide the host's render mechanism once per session.
1. **Name the field, never guess.** Selector = `voice_file` (not `id`). A `modify_scene` 200 under the edit session is durable — the ONE Gate 5 AFTER look (poster/composite) is the save-confirmation; no separate re-pull.
2. **Runtime = autonomous, end-to-end.** Work scene 2 → last content scene in ONE pass. Never pause to ask. No `A or B?` questions to the user.
2a. **Decision protocol — choose, don't defer.** Priority: content correctness → background fit → image-gen text correctness → minimal necessary edit. Trust the server for placement; don't invent extra work.
2b. **WideCast edit trigger = full autonomous run.** "edit this video" + a WideCast URL/`topic_id` = full audit + fix. Never ask scope.
2c. **Proof is a process artifact — no request suppresses it.** "Be concise / save credits / go fast" only shortens the final summary; it never cancels the applicable gate proofs (Gate 3 background, Gate 4 typo table) or module loads.
3. **Server guarantees placement — the agent does NOT audit it.** WideCast keeps every overlay object out of dead zones, off the narrator face, and inside the safe zone even after auto-fit, and decides whether a scene needs an overlay. **Do NOT re-verify any of this** — no dead-zone proof, no face-clearance, no A-roll layout ladder, no final-composition tuning, no overlay-existence decision. These are gone.
4. **The agent's whole job = the two WideCast blind spots + text.** (Gate 3) does the background clip fit the narration + target-market geography/context; (Gate 4) does image-model-baked text have typos. SVG typography text is deterministic and NEVER misspells — skip its typo check entirely. Plus (Gate 1) `text`/STT context correctness.
5. **Layer isolation + fixed narrator.** A background fix touches ONLY `mediaUrl`/`mediaType` — never the overlay. Never edit `narrator_face`, never resize/reposition the narrator (the server keeps overlays off the face for you).
6. **Gate 3 — background audit (conditional).** Applies only when the scene is NOT grid AND the narrator does not fill the frame (both from data). Look = the active plate; judge semantic/logic/geo/currency/context fit; fix via `mediaUrl` only.
7. **Gate 4 — overlay text typo (conditional).** Applies only when overlay text was image-model-generated (`illustration` sub_mode≠`photo_with_people`, or chart/diagram/object with baked text). Look = the overlay poster; print the per-string transcription table (transcribe letter-by-letter FIRST, judge after). SVG/typography → N/A.
8. **Grid ≤ 3 scenes/video, all sharing ONE grid.**
9. **Show found media in chat BEFORE evaluating** (background candidates when replacing a clip). No private preview first.
10. **Each scene is complete on its own — there is NO whole-video QA pass at the end.** When a scene reaches PASS it is done and not revisited.
11. **Pre-summary completion scan.** No summary/Telegram/export until every roster row is PASS. Batch/gallery/table/script/API results are triage only, never DoD proof.
12. **SCENE ROSTER + run_ledger file = the run's source of truth.** Print the roster at kickoff, persist to a local run_ledger file, update after every verdict/write; inline works rows in order, delegation closes rows in event order — either way EVERY row must close; re-`Read` on any resume/detour/compaction — never trust memory.
13. **The master is an INDEX — load each module BEFORE its step.** Memory ≠ re-loading. A truncated read ("output too large"/404/timeout/partial) = NOT loaded; re-read to EOF first.
13c. **Compaction VOIDS all loads.** Context compacted = resumed run: re-`Read` run_ledger, re-load the current scene's modules, reprint its ledger + plan; the summary's "short checklist" is forbidden.
14. **Announce plan + report progress.** Vertical 5-gate checklist at scene start, gate-by-gate progress, ✓/✗/N-A recap + `Scene N: PASS|FAIL` verdict at scene end.
15. **Mode threshold: inline by default ≤30 content scenes; scene-editor fan-out when >30 or the user asks for parallel.** Above the threshold the `06_subagent_protocol` pipeline is mandatory. Main agent: `edit_session start` → spawn editors (rolling K=5, local skill dir, fixed template, own LOAD LEDGER) → validate reports (no images) → `edit_session commit`. Each editor writes ONLY its own `voice_file`; the server lock + session cache make parallel scene-scoped writes safe.

---

## ⛔ JUMP-PREVENTION TRIGGERS — 1-line interrupts (full list → `ai_video_editor/02_jump_prevention`)

If you're about to do any of these, STOP and do the prerequisite first:

- start a scene → load `10_mechanics`
- ask "what kind of edit?" after a WideCast URL → STOP. Scope is **Full audit + fix**.
- Gate 3 applies (non-grid, narrator not covering frame) → load `20_background`, pull the plate, print Gate 3 BACKGROUND PROOF
- about to audit dead-zone / face-clearance / overlay-existence / composition → STOP, the server guarantees these; do not re-verify
- about to run an A-roll layout ladder / resize the narrator → STOP, the narrator is fixed input and the server keeps overlays off the face
- Gate 4 applies (image-gen text) → pull the overlay poster, print the per-string typo table (transcribe first)
- about to typo-check an SVG/typography overlay → STOP, deterministic text never misspells; mark Gate 4 N/A
- act on an image you have not SHOWN locally (one user-visible render) → show it first
- ask the user to choose/approve during runtime → STOP, decide yourself
- declare `Scene N: PASS` without scanning the 5 gates + §7 → run the scan
- move to next scene without stated `PASS`/`FAIL` → declare verdict first
- about to run a whole-video QA pass at the end → STOP, there is none; each scene is done at its own PASS
- write final summary / hand-off / export question → run Pre-summary completion scan
- spawn a subagent / process scenes in parallel → load `06_subagent_protocol` first; fixed template; editors write only their own `voice_file`
- a subagent writes OUTSIDE its own scene (foreign `voice_file`/export/publish) → STOP, report INVALID
- hand off while the edit session is still open → STOP, run pre-summary scan then `edit_session commit`
- context just got compacted → STOP: compaction VOIDS all module loads; `Read` run_ledger, re-load current scene's modules, reprint its plan
- resuming/continuing a run → `Read` the run_ledger file + re-load modules, never work from memory

---

## ⭐ DEFINITION OF DONE — 5 gates per scene (full DoD + template blocks → `ai_video_editor/03_dod_gates`)

Print this 5-gate checklist VERTICALLY at the start of every scene; tick ✓/✗/N-A at the end and state `Scene N: PASS|FAIL`. Gates 3 and 4 are CONDITIONAL — routed from data at Gate 2; many scenes take 0–1 image looks.

1. ☐ **Text / STT** — read `text` in whole-video context, fix STT/context/domain errors (branch K). No look.
2. ☐ **Role / route** — read `type`/`pattern`/`sub_mode`/`show_narrator`/`mediaType` to decide whether Gate 3 and Gate 4 apply. No look.
3. ☐ **Background audit** — *only if non-grid AND narrator doesn't fill the frame.* Load `20_background`, pull the plate, print Gate 3 BACKGROUND PROOF (semantic/geo/context). Else `N/A`.
4. ☐ **Overlay text typo** — *only if overlay text was image-model-generated.* Pull the overlay poster, print the per-string transcription table. SVG/typography or no overlay text → `N/A`. **Opening-scene exception:** the first content row (`opening`) ALWAYS pulls the poster + runs the OPENING POSTER CHECK (aesthetic hook pass) even for typography — rebuild if it clearly falls short (cap 1), author one if none exists.
5. ☐ **Confirm & save** — if you edited: one AFTER look + re-pull to confirm saved. If no edit → `N/A`. Then print MODULE COVERAGE GATE.

All applicable gates met + §7 (`05_quality_qa_priority`) scan → `Scene N: PASS`. Otherwise `Scene N: FAIL — ✗K …`; fix and re-scan. The server guarantees placement/dead-zone/face/composition — do NOT add gates to re-verify them.

---

## ✅ SELF-AUDIT — before every reply (full version → tail of `ai_video_editor/01_critical_rules`)

Silently confirm — and fix any "no" before replying:
- Did I open the module(s) for the action I'm taking THIS turn, not work from memory?
- Every image I used as evidence — exactly ONE user-visible render (not two, not a private Read)?
- Am I trying to re-verify placement/dead-zone/face/composition/overlay-existence? If yes, STOP — the server guarantees those.
- Which of the 5 gates am I on; did I route Gate 3/4 from data; did I announce progress?
- Ending a scene? Scanned the 5 gates + §7? Stated `Scene N: PASS|FAIL`?
- Field discipline: `voice_file` selector + re-pull to confirm saved?
- About to summarize/hand-off/export? Ran Pre-summary completion scan against the run_ledger roster?
- Human must act? Used a standalone `[ACTION REQUIRED]` block?
- Did any module read error/truncate/compact this session? If yes, re-read to EOF / re-anchor on run_ledger before proceeding?
- Am I dropping/compressing an applicable proof (Gate 3 background, Gate 4 typo table) to be "concise"? If yes, restore it.
- Spawning/accepting subagent work? Loaded `06_subagent_protocol`, started the edit session, used the fixed template with `Scene <id> editor agent` names + local `skill_root`, validated each report's LOAD LEDGER + write scope, viewed no images myself, and committed the session before hand-off?

---

## next_action

**Right now → load `ai_video_editor/00_ENTRYPOINT` AND the 5 core modules (`01_critical_rules`, `02_jump_prevention`, `03_dod_gates`, `04_principles_workflow`, `05_quality_qa_priority`).** Then pull `video_data`, do the §2 whole-video context pass, and start scene 2. Do not start scene 1; scene 2 is the first real content scene.
