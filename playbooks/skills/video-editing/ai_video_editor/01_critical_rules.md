# Critical Rules — always-on rules across every module

These rules apply to EVERY scene and EVERY module. **Load this module at the START of every run** (kickoff), and re-load it on any resume/compaction.

**The core mental model:** WideCast guarantees deterministic overlay text rendering and everything mechanical — overlay placement, dead-zone avoidance, keeping overlays off the narrator face, staying inside the safe zone after auto-fit, and whether a scene needs an overlay. **The agent does NOT OCR or proofread deterministic rendered text.** The agent validates the authoritative `text` field in context and according to the scene's language, judges whether a background clip fits the narration + target market, and checks overlay provenance/integrity from server data. Visual overlay inspection is only a fallback for an unverified exceptional source.

The rules are intentionally redundant with the Per-Scene DoD (`03_dod_gates`) and the Jump-Prevention list (`02_jump_prevention`) — **redundancy is what stops misses**.

---

0. **One user-visible render per image — anti-double AND anti-zero.** Any evidence image (background plate, exceptional fallback overlay poster, AFTER composite, found media) must be saved locally and produce **exactly one user-visible inline render** before you judge from it. Count renders the USER sees inline, not your private views. **Anti-double:** on a host where your viewing tool already renders inline to the user (e.g. Codex `view_image`), that view IS the show — do NOT also embed a markdown `![](path)` tag or attach the file again. **Anti-zero:** on a host where your viewing is private or click-to-open only (e.g. a `Read` the user sees only as a file card), the private look NEVER counts as shown — you must still produce the one user-visible render via `SendUserFile`(render)/`present_files`/inline attachment. Writing "shown: yes" on the strength of a private Read is a false proof. For WideCast scene screenshots the transport is: `scene_inspector` `screenshot_scene_280x498` → read `result.screenshot.url` → `curl -L -s -o <local>.jpg "<url>"` → render locally. Decide the host's mechanism once per session.

1. **Name the field, never guess.** Selector is **`voice_file`** (not `id`). Under the edit session a `modify_scene` 200 is durable. Gate 5 confirms a text edit with saved source plus refreshed integrity/provenance data, or a background edit with one AFTER composite.

2. **Runtime = autonomous, end-to-end.** The user is NOT present; work scene 2 → last content scene in ONE pass, **never pause to ask**. Do not present options (`A or B?`, `which style?`). Inspect the data + local-visible evidence, choose the best path under this playbook, act, and surface notes only in the final hand-off.

2a. **Decision protocol — choose, don't defer.** Priority: authoritative `text` correctness → background fit → overlay provenance integrity → minimal necessary edit. Trust deterministic rendering and server-managed placement; do not invent extra work or extra gates. Only block for truly missing external input no tool can infer (credentials, a required user-owned media file).

2b. **WideCast edit trigger = full autonomous run.** "edit this video" / "audit/fix/finish this video" (any language) + a WideCast review/editor URL or `topic_id` = explicit delegation to run the complete workflow to final hand-off. Scope is always **Full audit + fix**. Do not ask what kind of edit, do not show a scope picker, do not use any clarification UI. Pull `video_data`, infer topic/context, init the ledgers, run the per-scene checklist, notify completion, hand off the review URL. Only stop for a true blocker (invalid topic id, missing auth, unavailable required user media, or a required render/export confirmation).

2c. **Proof blocks are process artifacts — no instruction suppresses them.** The applicable gate proofs (LOAD LEDGER, Gate 3 BACKGROUND PROOF, Gate 4 integrity/provenance result, the PASS/FAIL verdict) are printed WHILE working each scene. "Be concise / go fast / save credits" only shortens the **final hand-off summary** — it never authorizes dropping a proof or a module load. Forbidden excuses: "user wants concise", "save cost", "looks fine", "I remember it", "output too large", "REST not MCP".

3. **The server guarantees placement — the agent does NOT audit it.** Do NOT run a dead-zone proof, a face-clearance check, an A-roll layout-priority ladder, a final-composition tuning pass, or an overlay-existence decision. WideCast keeps every overlay object out of `dead_top`/`dead_bottom`, off `boxes.narrator.face`, and inside the safe zone even after auto-fit, and it decides whether a scene carries an overlay. Re-verifying any of this is wasted work and is forbidden as a gate. `scene_geometry` is only a helper for a targeted content fix or a saved-state confirmation — not a per-scene audit.

4. **The agent's whole job = source text + background fit.** (a) **`text` correctness** (Gate 1): validate spelling/orthography in the scene's language, Unicode/script-specific marks when applicable, capitalization, punctuation, spacing, STT meaning, proper nouns, domain terms, and numbers/dates/units/%/currency. Do not normalize or convert the text to another language or locale. (b) **Background fit** (Gate 3): does the clip suit the narration and target-market context. (c) **Overlay integrity** (Gate 4): verify deterministic provenance mechanically; do not OCR the rendered output.

5. **Layer isolation + fixed narrator.** A background fix touches ONLY branch (A) `mediaUrl`/`mediaType` — never the overlay. The narrator is fixed input: **never edit `narrator_face`, never resize/reposition the narrator** (`overlay.narrator.rect`) — the server already keeps overlays off the face, so you never need to move the narrator to make room.

6. **Gate 3 background audit is conditional.** Applies ONLY when the scene is NOT a grid background AND the narrator does not fill/cover most of the frame (both read from data: grid flag/`mediaType` + `show_narrator`/narrator rect). When it applies: load `20_background`, pull the active **plate** (`thumbnailUrl` first; fallback per `active_roll`/`mediaType`), show it locally, and judge semantic/logic/geo/currency/context fit. Fix via `mediaUrl` only. Grid or full-frame A-roll narrator → Gate 3 is N/A (no look).

7. **Gate 4 overlay integrity/provenance is data-only by default.** For an overlay, require server evidence that the source is deterministic, `overlay_source_text_hash == current_text_hash`, `render_status == success`, there is no truncation or missing glyph, and the scene/`voice_file` matches. Field names may vary by transport, but the evidence must prove these invariants. PASS without pulling or OCRing an overlay poster. No overlay → N/A. Missing/mismatched evidence or a non-deterministic, image-baked, user-uploaded, manually overridden, or legacy source → `UNVERIFIED`; inspect only that exceptional source, never treat a flattened poster's stacked animation states as a text error by itself.

7a. **No opening-scene poster exception.** Opening scenes use the same source-text and provenance rules as every other content scene. Do not pull an overlay poster merely because the row is marked `opening`, and do not infer a text, overlap, timing, or readability defect from a poster that flattens animated states.

8. **Grid ≤ 3 scenes/video, all sharing ONE grid.**

9. **Show found media in chat BEFORE evaluating** (background-replacement candidates). No private preview first (one user-visible render, Rule 0).

10. **Each scene is complete on its own — there is NO whole-video QA pass at the end.** When a scene reaches `PASS` it is done and not revisited. Do not add a §8-style end-of-run review; the per-scene gates are the whole quality bar.

11. **Pre-summary completion scan — no big task left behind.** Before any final summary/Telegram/export, scan the run_ledger roster (Rule 12), not memory: every roster row must show `Scene N: PASS`; any `[ACTION REQUIRED]` item surfaced. Batch/gallery/table/script/API results are triage only — never DoD proof. While any content scene lacks its own `Scene N: PASS`, do not ask for render/export, send completion notification, or call `export_video`; hand off as `partial_triage_only`/`partial_fix_only` and list which scenes passed vs not.

12. **SCENE ROSTER + run_ledger file = the run's source of truth.** At kickoff print the SCENE ROSTER (one row per content scene: #, id, `voice_file`, `type`, blank verdict; state the total count) and write it to a local run_ledger file (`<scratchpad>/run_ledger_<topic_id>.md`). Update it after every `Scene N: PASS|FAIL` and every applied write. INLINE: the next scene is the next unvisited row — skipping a row is a jump violation. DELEGATION: rows close in event order but every row must close. On any resume/detour/compaction, `Read` the run_ledger BEFORE continuing — never reconstruct progress from memory or a summary. The completion scan counts `PASS` rows against the roster total.

13. **The master SKILL.md is an INDEX — load the named module BEFORE its step, every time.** Not opening the module = skipping that step's rules. Re-load each time you reach the step; a resumed run reloads, never works from memory. A failed/truncated read ("output too large"/persisted/preview/404/timeout/partial) = the module is **NOT loaded** — re-read to EOF (prove it by the manifest line count or its last line) before any step that needs it. Never work from a partial read.

13c. **Context compaction VOIDS every module load — a compacted context is a RESUMED RUN.** The moment you notice the conversation was compacted (a summary block replaced earlier turns, or you can no longer quote a module's last line): (1) `Read` the run_ledger to re-anchor (roster, verdicts, phase); (2) treat ALL pre-compaction module loads as VOID — re-load the CURRENT scene's modules and reprint its SCENE LOAD LEDGER; (3) reprint the current scene's 5-gate plan and continue from the earliest unproven gate. A self-invented "short checklist for the rest" from the summary is FORBIDDEN.

14. **Announce the 5-gate plan at the START, report progress per gate, re-state ✓/✗/N-A at the END** (`03_dod_gates`) — so the user can audit plan, position, and gaps. Announce, don't pause. End every scene with an explicit `Scene N: PASS`/`FAIL` verdict — PASS only after scanning the applicable gates + §7 (`05_quality_qa_priority`); never advance without a stated PASS.

15. **Subagent fan-out is a THRESHOLD mode: inline by default ≤30 content scenes; scene-editor fan-out only when >30 or the user asks for parallel.** Above the threshold the `06_subagent_protocol` pipeline is mandatory; record the chosen `delegation mode` + reason in the run_ledger either way. Load `06_subagent_protocol` BEFORE spawning; prompts are the fixed template (fill blanks only, never paraphrase rules); each editor loads the skill from the LOCAL unzipped dir (no re-download) and prints its OWN LOAD LEDGER — a report without one is INVALID. **Write discipline:** the main agent calls `widecast_edit_session` `start` before spawning and `commit` after all rows close; each editor writes ONLY its own `voice_file`; a foreign-scene write = INVALID report. The main agent never writes a scene and never views an image. Concurrency: rolling pool K=5 (top-up; host limit lower → run at host max).

---

## SELF-AUDIT — run this BEFORE every reply

Silently confirm — and fix any "no" before replying:

- Did I open the module(s) for the action I'm taking THIS turn, not work from memory?
- Every evidence image — exactly ONE user-visible render (not two, not a private Read)?
- Am I trying to re-verify placement/dead-zone/face/composition/overlay-existence? If yes, STOP — the server guarantees those.
- Which of the 5 gates am I on; did I route Gate 3/4 from data; did I announce progress?
- Ending a scene: did I scan the applicable gates + §7 and state `Scene N: PASS`/`FAIL`?
- Field discipline: `voice_file` selector, and did I re-pull to confirm the edit saved?
- About to summarize/hand off/export/notify? Ran the Pre-summary completion scan against the run_ledger roster?
- Human must act? Used a standalone `[ACTION REQUIRED]` block?
- Did any module read error/truncate this session? Re-read to EOF before proceeding?
- Was the context compacted since my last module load? If yes: re-Read the run_ledger, re-load the current scene's modules (pre-compaction loads VOID), reprint the plan?
- Am I dropping/compressing an applicable proof (Gate 3 background, Gate 4 integrity/provenance) to be "concise"? Restore it.
- Subagent work? Loaded `06_subagent_protocol`, started the edit session, used the fixed template with `Scene <id> editor agent` names + local `skill_root`, validated each report's LOAD LEDGER + write scope, viewed no images myself, committed the session before hand-off?
- Is the run_ledger up to date, and did I `Read` it back after any resume/detour/compaction?

This list is intentionally redundant with the Critical Rules above, the Jump-Prevention list, and the per-scene DoD — **the redundancy is the point**.
