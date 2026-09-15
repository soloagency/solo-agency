# 00 · Entrypoint — how to start an AI video-editor run

Version: `modular-2.0`. Kickoff prompt for an editing run (manual, continued, or scheduled). Short on purpose, but explicit enough to FORCE loading the real playbook instead of improvising.

## Run prompt

```text
Run an AI video-editor pass on this WideCast video.

1. `Read` SKILL.md (the MASTER INDEX) first — its LOAD MAP, CRITICAL RULES, JUMP-PREVENTION, the 5-gate Definition of Done, and SELF-AUDIT govern everything. Do NOT rely on memory; `Read` each module the LOAD MAP names when you reach its step, every time.
2. Core model: WideCast already guarantees deterministic overlay text rendering, overlay placement, dead-zone avoidance, overlays off the narrator face, safe-zone fit, and whether a scene needs an overlay. Do NOT OCR or proofread deterministic rendered text. Your job is: (Gate 1) validate the authoritative `text` field in context and according to the scene's language, (Gate 3) judge background fit, and (Gate 4) verify overlay provenance/integrity from data, using visual fallback only for an unverified exceptional source.
3. Pull `video_data` once; build the full script for context (§1–§2). Print the SCENE ROSTER and write the run_ledger file (one row per content scene, blank verdict, total count). All content rows use the same gates; there is no special opening-poster text check.
3b. **Delegation check (threshold):** if this runtime can spawn subagents AND the video has MORE THAN 30 content scenes (or the user asked for parallel), `Read` `ai_video_editor/06_subagent_protocol.md` NOW and run the scene-editor pipeline: export steward data files, ensure the skill is unzipped locally, `widecast_edit_session action='start'`, spawn one `Scene <id> editor agent` per content scene (rolling K=5), validate reports without viewing images, then `widecast_edit_session action='commit'`. Record `delegation mode`. Otherwise run inline (record `inline — reason: ≤30 scenes`).
4. Work ONE scene at a time, scene 2 → last content scene, in one continuous pass. Every scene is routed the same way — no special opening-poster / thumbnail-sync / closing-CTA authoring passes.
   - **Autonomous:** the agent chooses; never ask the user to pick a background/style/overlay option. After a WideCast URL/topic_id the scope is already **Full audit + fix** — do not ask "what kind of edit".
   For every content scene:
   a. Announce the plan as a vertical checklist:
      ```text
      Scene N plan:
      ☐ Gate 1 — Text / STT
      ☐ Gate 2 — Role / route
      ☐ Gate 3 — Background audit (or N/A)
      ☐ Gate 4 — Overlay integrity/provenance (or N/A)
      ☐ Gate 5 — Confirm & save (or N/A no edit)
      ```
   b. Gate 1: validate `text` in whole-video context and the scene's language; fix spelling/orthography, STT meaning, proper nouns, domain terms, and numbers/dates/units/%/currency errors with branch K. Gate 2: read `type`/`pattern`/`sub_mode`/`show_narrator`/`mediaType` plus overlay provenance — Gate 3 applies if non-grid AND narrator doesn't fill the frame. Gate 3: load `20_background`, pull the plate, print the Gate 3 BACKGROUND PROOF; fix via `mediaUrl` only. Gate 4: use server data to confirm deterministic source, source/current-text hash match, successful render, no truncation, no missing glyph, and matching scene/`voice_file`; do not OCR the poster. If provenance is absent/mismatched or the source is non-deterministic/baked/user-uploaded/manual/legacy, mark `UNVERIFIED` and run only the narrow fallback inspection. Gate 5: use saved `text`/integrity data for a text edit or the AFTER composite for a background edit; print MODULE COVERAGE.
   c. Every evidence image: exactly ONE user-visible render (Rule 0) before judging — normally the Gate 3 plate and a Gate 5 AFTER composite after a background edit; an overlay poster is allowed only for Gate 4 exceptional fallback. No BEFORE composite. Never re-verify placement/dead-zone/face/composition.
   d. End the scene with a verdict: `Scene N: PASS — …` (after scanning the applicable gates + §7) or `Scene N: FAIL — …; fixing.` Never advance without a stated PASS.
5. There is NO whole-video QA pass and NO final thumbnail pass — each scene is done at its own PASS.
6. Announce ≠ pause: present evidence and keep working; the only stop is the very end. If the human must act, use a standalone `[ACTION REQUIRED]` block (e.g. an unrecorded A-roll narrator).
7. Before every reply, run the master SELF-AUDIT.
8. Before any final summary/export/Telegram, run the **Pre-summary completion scan**: `Read` the run_ledger and confirm every roster row is `Scene N: PASS`, no `[ACTION REQUIRED]` hidden. In delegation mode, `widecast_edit_session action='commit'` first. Then hand off with a short change summary, the review URL, and one question: `Render/export the final MP4 now, or review the scenes first?` Do not call export until the user confirms. Send a Telegram/self-notification with the review URL.
```

## Notes
- This entrypoint does not replace the master — it points at it. All rules live in the master + modules.
- A scheduled/continued run is the SAME workflow with saved context: reload the modules at run time; a compacted context is a RESUMED run (re-`Read` the run_ledger, re-load modules).
