# Per-Scene Definition of Done — 5 gates + template blocks

Load this module at the **START of every scene**. Every scene must pass 5 gates before you may state `Scene N: PASS` and move on.

**What changed and why (read once):** WideCast mechanically guarantees deterministic overlay text rendering plus placement, dead-zone avoidance, narrator-face clearance, safe-zone fit, and whether a scene needs an overlay. **The agent must NOT re-verify any of those.** The agent's job is limited to:

1. **Background semantic/logic/geo/context fit** — does the background clip actually suit what is being said, and match the target country/market. (Look: the background plate.)
2. **Authoritative source-text correctness** — validate the scene's `text` field in whole-video context and according to that scene's language. Overlay output is not a second copy source to OCR.

Plus the always-cheap overlay provenance/integrity guard and confirming an edit saved.

**Normal image looks are now at most 2 per scene, often 0–1:** the background plate (Gate 3, only when it applies) and an AFTER composite after a background edit (Gate 5). There is **no BEFORE screenshot, no routine overlay-poster look, and no per-scene final-composition look**. Gate 4 is data-only unless an exceptional source cannot be proven deterministic.

---

## LOAD LEDGER — proof-of-read before any write

`LOAD_MANIFEST.md` publishes the expected line count per module. After loading a module, compare your actual `lines` to its manifest row — a shortfall means truncated = NOT loaded; re-read to EOF before writing. That one number is the required check.

**KICKOFF LOAD LEDGER — print BEFORE the first `modify_scene` of the run:**

```text
KICKOFF LOAD LEDGER:
☑ 00_ENTRYPOINT          lines=<N>  manifest=<M | absent>
☑ 01_critical_rules      lines=<N>  manifest=<M | absent>
☑ 02_jump_prevention     lines=<N>  manifest=<M | absent>
☑ 03_dod_gates           lines=<N>  manifest=<M | absent>
☑ 04_principles_workflow lines=<N>  manifest=<M | absent>
☑ 05_quality_qa_priority lines=<N>  manifest=<M | absent>
☑ 10_mechanics           lines=<N>  manifest=<M | absent>
Verdict: <PASS — every lines==manifest | BLOCKED — re-read <module> to EOF first>
```

**SCENE LOAD LEDGER — print at the START of each scene** for the modules that scene needs: `20_background` when Gate 3 applies (non-grid, narrator not covering the frame); `30_overlay_core`/`31_typography`/`32_charts`/`33_patterns` only when you must fix an overlay defect. Same `lines=` + `manifest=` per line.

**Rule:** a module short of its manifest row = truncated = NOT loaded → BLOCKED from writing. **Context compaction voids ledgers:** if the conversation was compacted since a ledger was printed, re-load and reprint (Critical Rule 13c). **Delegation mode:** the coordinator prints the slimmer COORDINATOR LOAD LEDGER from `06_subagent_protocol`.

---

## Batch / gallery / script outputs are triage only

Batch contact sheets, galleries, tables, bulk API results, or any "all scenes at once" artifact are **triage only** — never DoD proof, never authorize `Scene N: PASS`. If the run used only batch triage or fixed a selected subset without closing every content scene, the status is `partial_triage_only` (or `partial_fix_only`). Do not ask for render/export, call `export_video`, or send completion notification while any content scene lacks its own `Scene N: PASS`.

---

## DoD — the 5 gates

Do NOT advance while any applicable gate is unchecked.

1. ☐ **Text / STT** — validate `text` in whole-video context and according to the scene's language: spelling/orthography, applicable script-specific characters/marks, Unicode, capitalization, punctuation, spacing, STT meaning, proper nouns, domain terms, and numbers/dates/units/%/currency. Do not normalize or convert to another language or locale. Fix errors with `modify_scene` branch (K) (Step 1 below).
2. ☐ **Role / route** — read `type` · `pattern`/`sub_mode` · `visual` · `keyword` · `quote` · `talking_point` · `show_narrator`/`active_roll` · `mediaType` plus overlay provenance. This decides Gate 3 applicability and Gate 4's deterministic/fallback/N-A route. No image look here — data only.
3. ☐ **Background audit** — *applies ONLY when the scene is NOT a grid background AND the narrator does not cover most of the frame* (both read from data: `mediaType`/grid flag + `show_narrator`/narrator rect). When it applies: load `20_background`, pull the active background **plate** (`thumbnailUrl` first; fallback per `active_roll`/`mediaType`), show it locally, and print the **Gate 3 BACKGROUND PROOF**. When it does not apply (grid, or A-roll narrator fills the frame), mark `N/A — <grid | narrator fills frame>` and take no look.
4. ☐ **Overlay integrity/provenance** — *data-only by default*. For an overlay, require server evidence of deterministic source, `overlay_source_text_hash == current_text_hash`, successful render, no truncation, no missing glyph, and matching scene/`voice_file`. Never OCR deterministic output. No overlay → N/A. Missing/mismatched evidence or a non-deterministic, image-baked, user-uploaded, manually overridden, or legacy source → `UNVERIFIED` and the narrow fallback in the Gate 4 template.
   - **No opening-scene exception.** The `opening` row follows the same rule. Never infer a text, overlap, timing, or readability defect from a static poster that flattens animated states such as `one_by_one`.
5. ☐ **Confirm & save** — if you made an edit, use proof appropriate to the changed field: the durable response plus refreshed `text`/integrity data for a text edit, or the **AFTER composite** for a background edit. Do not pull an overlay poster to confirm deterministic text. If you made **no** edit, this gate is `N/A — no edit`. Then print the **MODULE COVERAGE GATE**.

All applicable gates checked → `Scene N: PASS`. **Show ≠ pause:** present each image, then keep working.

---

## Announce the plan + report progress (mandatory)

- **At the START of each scene**, post the 5-gate checklist VERTICALLY (one gate per line) — the plan the user audits up front. Do NOT compress it into one inline sentence.
- **As you work**, announce each gate (`→ Gate 3…` then `✓ Gate 3`).
- **At the END**, repeat the checklist with ✓/✗/N-A and a one-line note per gate, then the verdict.
- **Announce ≠ pause** — report and keep working; do not wait for a reply.

---

## Exact template blocks — use these VERBATIM shapes

### Scene start (plan) — vertical only

```text
Scene N plan:
☐ Gate 1 — Text / STT
☐ Gate 2 — Role / route
☐ Gate 3 — Background audit (or N/A)
☐ Gate 4 — Overlay integrity/provenance (or N/A)
☐ Gate 5 — Confirm & save (or N/A no edit)
```

### Gate 3 BACKGROUND PROOF — only when Gate 3 applies

```text
Gate 3 BACKGROUND PROOF:
☑ 20_background.md — opened for this scene
Applies check: <PASS applies | N/A grid | N/A A-roll narrator fills frame>
Active plate evidence: <local file path shown to user>
Active media: <mediaUrl or active thumbnail/media URL>
Current background read: <what is actually visible in the plate>
Scene context: <1-line text/talking_point/visual summary>
Fit check: <PASS|FAIL> — background relevant to narration/keyword/visual
Geo/currency check: <PASS|FAIL|N/A> — if location-sensitive (insurance, tax, legal, real estate, healthcare, local services, finance…), country/region/currency/signage/language/road/form cues match the target market
Technical check: <PASS|FAIL|N/A> — visible real footage is portrait, not too bright/cluttered, no watermark/burned-in text, no duplicate real clip
Action: <keep current media | replace via mediaUrl>
Verdict: <PASS keep | FIXED + PASS | FAIL — continue background search>
```

Background changes touch ONLY `mediaUrl`/`mediaType` (branch A). A wrong background never authorizes touching the overlay.

**If Gate 3 applies and this proof is missing, the scene is not done.**

### Gate 4 OVERLAY INTEGRITY / PROVENANCE — data-only by default

Do not pull, OCR, or proofread an overlay poster when deterministic provenance is healthy. Field names may vary by transport, but the returned data must prove the following invariants.

```text
Gate 4 OVERLAY INTEGRITY / PROVENANCE:
Route: <DETERMINISTIC | EXCEPTIONAL_FALLBACK | N/A no overlay>
Deterministic source: <PASS true | UNVERIFIED>
Source/current text: <PASS overlay_source_text_hash == current_text_hash | UNVERIFIED missing | FAIL mismatch>
Render status: <PASS success | UNVERIFIED missing | FAIL>
Truncation: <PASS none | UNVERIFIED missing | FAIL>
Missing glyph: <PASS none | UNVERIFIED missing | FAIL>
Identity: <PASS scene + voice_file match | UNVERIFIED missing | FAIL mismatch>
Poster OCR used: <no | yes — exceptional fallback reason>
Verdict: <PASS deterministic integrity | N/A no overlay | UNVERIFIED — inspect/fix exceptional source | FAIL — fix integrity mismatch>
```

`EXCEPTIONAL_FALLBACK` is allowed only for a non-deterministic/image-baked, user-uploaded, manually overridden, or legacy overlay, or when a required provenance invariant is unavailable. Inspect only the affected source; show any fallback image once before judging it. A static poster may flatten multiple timed animation states, so stacked/unreadable poster text alone is **never** a typo, overlap, timing, or readability failure. Fix/verify the source or provenance, then rerun this data block.

### MODULE COVERAGE GATE — before declaring Scene PASS

```text
MODULE COVERAGE GATE:
☑ 00_ENTRYPOINT / 01_critical_rules / 02_jump_prevention / 03_dod_gates / 04_principles_workflow / 05_quality_qa_priority / 10_mechanics — loaded at kickoff
☑ 20_background.md — loaded because Gate 3 applied OR N/A: <reason>
☑ 30_overlay_core / 31_typography / 32_charts / 33_patterns — loaded because an overlay defect had to be fixed OR N/A: <reason>
Verdict: <PASS module coverage | FAIL — load missing module and resume>
```

### Progress + verdict markers

- Progress: `→ Gate K: <doing…>` then `✓ Gate K: <result>` (or `Gate K: N/A — <reason>`).
- Scene end: `Scene N: PASS — ✓1…✓5` **or** `Scene N: FAIL — ✗K <what's missing>; fixing.`
- Human must act (record A-roll / final hand-off): a standalone `**[ACTION REQUIRED]**` block.

---

## PASS / FAIL verdict — declare it before EVERY scene hand-off

- To say **`Scene N: PASS`** you must scan all 5 gates and the §7 Quality Standard (`ai_video_editor/05_quality_qa_priority`) and confirm each applicable one is met (or justified N/A). PASS is earned by the scan, never from memory.
- If Gate 3 applied, the PASS scan names its verdict (`PASS keep` / `FIXED + PASS`). If Gate 4 applied, its integrity/provenance block is present. If an edit was made, the edit-appropriate save proof is present.
- Batch/gallery/table/script/bulk-API outputs are not PASS evidence.
- Any unmet gate → `Scene N: FAIL — [gates]`, fix, re-scan, re-declare. Never advance on a FAIL or with no verdict.

**Each scene is complete on its own** — there is no separate whole-video QA pass at the end. When a scene reaches PASS, it is done and not revisited.

---

## Final video hand-off — after the last content scene passes

- Run the **Pre-summary completion scan** (Critical Rule 12d): `Read` the run_ledger file and count `Scene N: PASS` rows against the SCENE ROSTER total. Every roster row must have a PASS verdict (a blank/FAIL row = not done). No unhandled `[ACTION REQUIRED]` item hidden.
- In delegation mode, `widecast_edit_session action='commit'` first (staged edits are not live until commit).
- Pull/keep the `review_url`.
- Send the user a Telegram/self-notification (WideCast self-notify tool, email fallback) with the `review_url`.
- Short summary of what changed/fixed. Keep it concise; do not replay every gate.
- Ask exactly one export question after every scene has individual PASS: `Render/export the final MP4 now, or do you want to review the scenes first?`
- Do **not** call `export_video` until the user explicitly confirms in the current turn.

If any content scene is not individually PASS:

```text
Run status: <partial_triage_only | partial_fix_only>
Scenes with individual PASS: <scene ids or none>
Scenes not individually PASS: <scene ids + missing gate>
Export/render: blocked until every content scene has Scene N: PASS
```

---

## Pre-checklist setup — one thing to determine

### Has the A-roll scene been recorded yet?

If `arollUrl`/`mediaUrl` still points to a library placeholder (`statics/aroll_*.png`, `mediaType=image`) → **not recorded yet**. Leave the layout as produced, but **REMIND the user to complete the narrator** in a standalone `[ACTION REQUIRED]` block — each scene ≤20 seconds:

1. **WideCast's built-in teleprompter** — RECOMMENDED (authentic).
2. **Upload a file** with the narrator's face + voice (`modify_scene` (I) `narrator.upload_video`).
3. **AI generate from a single photo** — a WideCast feature.

The narrator is fixed input: **never edit `narrator_face`, never resize/reposition the narrator.** The server keeps every overlay clear of the face and the safe zone automatically.

---

## Step 1 — Check Text and STT Errors

Read the scene's `text` field in the context of the whole video and the scene's declared or detected language. Check spelling and orthography, applicable script-specific characters/marks, Unicode, capitalization, punctuation, spacing, STT mishearing, domain terminology, proper names, numbers/dates/units/%/currency, missing/extra words, grammatically valid but contextually wrong words, and mismatch with the audio. Do not strip marks, transliterate, translate, or normalize into another locale.

If wrong, fix `text` with `modify_scene` branch (K) Segment text correction (keeps audio timing) before anything else.

If the correction is a domain term, proper noun, number, symbol, or entity (`Living Church` → `Living Trust`, `95` → `95%`, a company/person/product name), run a **semantic field sweep**: update the authoritative `text` and check dependent metadata when relevant; then require Gate 4's current-text hash and identity invariants so stale overlay state cannot pass.

A term fixed in `text` but still wrong in `quote`/`visual`/overlay is a Gate 1 failure, not a partial pass. Do not edit from personal feeling — rely on the full script, topic, `visual`, `keyword`, and neighboring scenes.

---

## Step 2 — Understand the Scene's Role (routing)

Read `type` (HOOK / STAT / KEY POINT / DATA / FACT / CALL TO ACTION / thumbnail), `pattern` + `sub_mode`, `visual`, `keyword`, `quote`, `talking_point`, `text`, `show_narrator`/`active_roll`, `mediaUrl`/`mediaType`.

From these DATA fields plus overlay provenance, route the conditional gates:

- **Gate 3 (background) applies** if the scene is NOT a grid background AND the narrator does not fill/cover most of the frame. Grid or full-frame A-roll narrator → Gate 3 is N/A.
- **Gate 4 (overlay integrity/provenance)** is data-only for deterministic overlays and N/A when no overlay exists. Missing/mismatched evidence or a non-deterministic/image-baked, user-uploaded, manually overridden, or legacy source routes to `EXCEPTIONAL_FALLBACK`. There is no opening-scene exception.

No screenshot here — routing is a pure data read. Then run the applicable gates in order. Layout mechanics + `modify_scene` branches → `ai_video_editor/10_mechanics`; background work → `20_background`; overlay-defect fixes → `30_overlay_core` (+ matching content/style module).
