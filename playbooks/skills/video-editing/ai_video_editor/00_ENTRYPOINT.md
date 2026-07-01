# 00 · Entrypoint — how to start an AI video-editor run

Version: `modular-1.1`. Use this as the kickoff prompt for an editing run (manual, continued, or scheduled). It is short on purpose, but explicit enough to FORCE loading the real playbook instead of improvising from memory.

## Run prompt

```text
Run an AI video-editor pass on this WideCast video.

1. `Read` SKILL.md (the MASTER INDEX) first — its LOAD MAP, CRITICAL RULES, JUMP-PREVENTION, the per-scene Definition of Done (§3), and SELF-AUDIT govern everything.
2. Do NOT rely on memory. At each step `Read` the module the LOAD MAP names — every time, even if read earlier this session, and even on a resumed/continued run. Opening a module is a required ACTION.
3. Pull `video_data` once; build the full script for context (§1–§2). Determine `faceless` and find the thumbnail. Initialize a **Background Audit Ledger** with one blank row per content scene; a blank row means the run is not complete.
4. Work ONE scene at a time, scene 2 → last content scene, in one continuous pass. The thumbnail is synced immediately after scene 2 and is NOT revisited at the end.
   - **Autonomous runtime rule:** the agent chooses; the user does not. Do not ask the user to pick a background, style, overlay option, or whether to rebuild. Show evidence for audit, decide using the master decision protocol, apply the best passing option, and report the choice.
   - **WideCast link/topic_id trigger rule:** if the user gave a WideCast editor/review URL or `topic_id` with a command like `edit this video`, `finish this video`, `audit/process/review/fix this video`, or equivalent, the scope is already **Full audit + fix**. Do **not** ask "what kind of edit", do **not** show a scope picker, do **not** use `request_user_input`/multiple-choice options, and do **not** say "`edit this video` can mean different things". Pull `video_data` and start the full autonomous workflow.
   - **Scene 2 / first real scene special rule:** before the normal DoD gates, `Read` `ai_video_editor/40_thumbnail_cta.md` and treat scene 2 as the **opening poster scene** regardless of its `type`/`pattern`. Build a short, thumbnail-like poster hook/consequence overlay for the first frame, with poster-thick title, strong face/subject preservation, and a named endpoint style such as dynamic poster typography, magazine-cover thumbnail, kinetic stacked type, typographic collage, object-integrated title, premium CTA poster, or minimal premium cover. Do not use a normal inside-scene centered card/horizontal text bar. Because scene 2 still plays as video, verify caption coexistence in the AFTER screenshot; if caption competes with the poster title, revise layout/caption placement instead of declaring PASS.
   - **Immediate thumbnail sync gate:** right after scene 2 PASS, before starting scene 3, apply the same uploaded poster overlay URL to the `type="thumbnail"` scene by stable `voice_file` and confirm it saved with `video_data`/`scene_geometry`. Do not continue to scene 3 until this sync is done.
   - **Final content / CTA scene special rule:** when reaching the last non-thumbnail/content scene, and always for `type="CALL TO ACTION"`, `Read` `ai_video_editor/40_thumbnail_cta.md` and treat it as the **closing CTA scene**. The close needs one clear action, typography stronger than objects, narrator-primary layout if it is A-roll, and a poster-grade endpoint style rather than a normal card/text-bar overlay.
   For every content scene:
   a. Announce the plan as a vertical checklist, one DoD gate per line. Do NOT compress it into one inline sentence:
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
   b. Run the 9 DoD gates in order, opening the module each gate requires (overlay → `30_overlay_core.md` + `10_mechanics.md` when A-roll + `31_typography.md` whenever any text/title/label appears + matching `31`/`32`/`33` + its `styles/*.md`; background → `20_background.md`; final composition → `10_mechanics.md`/`30_overlay_core.md` as needed; module coverage → prove every required playbook was loaded or N/A with reason). Report progress per gate (`→ Gate K…` / `✓ Gate K`).
      Attention-drift trap: after any detour/fix (wrong term, typo, missing symbol, bad overlay word, covered face, wrong thumbnail, background swap, geo mismatch, layout tweak, tool/debug issue, or cross-scene small fix), do NOT summarize or jump ahead. Run the master Gate Resume Scan: return to the 9-gate checklist and continue from the earliest unchecked or invalidated gate.
      During Gate 4, print the master `Gate 4 MODULE LOAD PROOF`, `Gate 4 A-ROLL LAYOUT PRIORITY PROOF` when `show_narrator=true`, `Gate 4 TITLE GATE PROOF`, and/or `Gate 4 SECONDARY TEXT GATE PROOF` where applicable before upload/rebuild. A-roll overlay/narrator tradeoff is decided here, not later in Gate 6: do not accept a shrunken narrator fallback unless the full-canvas and shifted-full-canvas priorities were tested first and rejected with concrete reasons. Loading typography is not enough unless all relevant title, secondary-text, and copy-correctness checks pass; typo/grammar/domain errors in overlay text fail Gate 4/7 even when final composition/readability pass; small/non-title labels must not use visible stroke/outline.
      During Gate 5, print the master `Gate 5 BACKGROUND PROOF` template and update the Background Audit Ledger. **Do not start Gate 6/final composition, declare `Scene N: PASS`, or hand off the final video until this scene has a Gate 5 proof verdict.**
      During Gate 6, tune/check final composition as one combined gate: layout, safe zone, dead-zone, face clearance, caption coexistence, background fit, readability, padding, and balance.
      Before declaring Scene PASS, print the master `MODULE COVERAGE GATE`: `00_ENTRYPOINT.md`, `10_mechanics.md`, `20_background.md`, `30_overlay_core.md`, `31_typography.md`, `32_charts.md`, `33_patterns.md`, and `40_thumbnail_cta.md` must be loaded where required or marked N/A with a reason. Missing required module = Gate 9 FAIL. Loading once does not complete a gate; it only satisfies coverage.
   c. Every screenshot / active background/media plate / found media: save locally + SHOW it to the user before judging or applying it. For authored overlays, show a local overlay preview only when the runtime already supports it cheaply; otherwise skip pre-upload preview and rely on the post-upload composite screenshot. Do not mention the internal overlay format to normal users. For WideCast scene screenshots, the only valid route is `result.screenshot.url` → `curl -L -s -o <local>.jpg "<url>"` → show the local file. In Gate 5 also pull the active media plate (`thumbnailUrl` first; fallback by `active_roll`/`mediaType`) as a separate local-visible image before judging the background.
   d. End the scene with a verdict: `Scene N: PASS — …` (only after scanning all 9 gates + §7) or `Scene N: FAIL — …; fixing.` Never advance without a stated PASS.
5. Do NOT run a final thumbnail pass. Thumbnail work is complete once the immediate post-scene-2 sync gate has passed.
6. Announce ≠ pause: present evidence and keep working; the only stop is the very end of the video (the runtime user is not present, §2). If the human must act, use a standalone `[ACTION REQUIRED]` block.
7. Before every reply, run the master SELF-AUDIT checklist.
8. Before any final summary/export question/Telegram completion message, run the master **Pre-summary completion scan**: every content scene has `Scene N: PASS`, Gate 1–9 checked, every Module Coverage Gate is PASS, the Background Audit Ledger is complete, scene 2 thumbnail sync is complete, final CTA endpoint handling is complete for the last content scene, and no major task is still pending. If anything is missing, do that work first. Only then hand off the finished video with a short summary of changes, the completed Background Audit Ledger summary, the review URL, and one explicit question: `Render/export the final MP4 now, or review the scenes first?` Do not call export until the user explicitly confirms. Also send the user a Telegram/self-notification that editing is complete and include the review URL.
```

## Notes
- This entrypoint does not replace the master — it points at it. All rules live in the master + modules.
- A scheduled/continued run is the SAME workflow with saved context, not a memory-only shortcut: reload the modules at run time.
