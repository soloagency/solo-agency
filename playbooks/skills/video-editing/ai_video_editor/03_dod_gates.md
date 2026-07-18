# Per-Scene Definition of Done — 5 gates + template blocks

Load this module at the **START of every scene**. Every scene must pass 5 gates before you may state `Scene N: PASS` and move on.

**What changed and why (read once):** WideCast now mechanically guarantees the things a blind server CAN compute — no overlay object enters a dead zone, no object covers the narrator face, every object stays inside the safe zone even after auto-fit, and whether a scene needs an overlay at all. **The agent must NOT re-verify any of those** (dead-zone proof, face-clearance, overlay-existence, A-roll layout ladder, final-composition tuning are all removed — trust the server). The agent's job is ONLY the two things WideCast is blind to:

1. **Background semantic/logic/geo/context fit** — does the background clip actually suit what is being said, and match the target country/market. (Look: the background plate.)
2. **Typos in image-model-generated text** — illustration/chart/diagram/object overlays whose text was baked by an image model (e.g. nano-banana) can misspell. **SVG typography overlays never misspell** (deterministic text render) — skip them entirely. (Look: the overlay poster.)

Plus the always-cheap data checks: `text`/STT context correctness, and confirming an edit saved.

**Image looks are now at most 3 per scene, often 0–1:** the background plate (Gate 3, only when it applies), the overlay poster (Gate 4, only image-gen-text scenes), and an AFTER screenshot (Gate 5, only if you actually edited). There is **no BEFORE screenshot** and **no per-scene final-composition look** — the server guarantees placement; you look only to judge the two blind spots and to confirm a fix.

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

1. ☐ **Text / STT** — read `text` in whole-video context; fix STT/context/domain errors with `modify_scene` branch (K) (Step 1 below). WideCast cannot judge context — this is yours.
2. ☐ **Role / route** — read `type` · `pattern`/`sub_mode` · `visual` · `keyword` · `quote` · `talking_point` · `show_narrator`/`active_roll` · `mediaType`. This decides which of Gate 3 / Gate 4 actually run (Step 2 below). No image look here — data only.
3. ☐ **Background audit** — *applies ONLY when the scene is NOT a grid background AND the narrator does not cover most of the frame* (both read from data: `mediaType`/grid flag + `show_narrator`/narrator rect). When it applies: load `20_background`, pull the active background **plate** (`thumbnailUrl` first; fallback per `active_roll`/`mediaType`), show it locally, and print the **Gate 3 BACKGROUND PROOF**. When it does not apply (grid, or A-roll narrator fills the frame), mark `N/A — <grid | narrator fills frame>` and take no look.
4. ☐ **Overlay text typo** — *applies ONLY when the overlay text was baked by an image model*: `pattern="illustration"` with `sub_mode` ≠ `photo_with_people`, or a chart/diagram/object pattern **containing image-generated text**. When it applies: pull the **overlay poster**, show it locally, and print the **Gate 4 OVERLAY TEXT TYPO CHECK** (per-string transcription table). **Skip entirely** for `typography_only` / SVG-text overlays (deterministic render never misspells), for scenes with no overlay text, and for photo overlays with no baked message text — mark `N/A — <reason>`.
   - **⭐ OPENING-SCENE EXCEPTION** (the ONE roster row marked `opening` — the first content scene after the thumbnail): this frame is the video's hook and platforms may auto-extract it, so it gets an extra aesthetic pass. **ALWAYS pull the overlay poster here — even for `typography_only`/SVG** — and run the **Gate 4 OPENING POSTER CHECK** below (in addition to the typo check when that also applies). If the poster fails the bar, **rebuild it** (load `30_overlay_core` + `31_typography` + styles; cap **1 rebuild**, preserve-biased — a "good enough" opening is kept, not chased). If the opening scene has **no overlay at all**, author a hook poster for it — this is the ONLY scene where the agent may add an overlay the pipeline did not create. This exception is opening-scene-only; every other scene keeps the normal conditional Gate 4.
5. ☐ **Confirm & save** — if you made ANY edit: pull the **ONE AFTER look** that verifies the fix — the overlay **poster** for a text fix (re-run the per-string table), or the **AFTER composite** for a background fix — and show it locally. **That look IS the save-confirmation**: it renders from the saved state under the edit session, so a corrected poster / a swapped composite proves the write persisted. Do NOT also re-pull `video_data` and `scene_geometry` separately — that is redundant (a `modify_scene` 200 under the edit session is already durable). If you made **no** edit, this gate is `N/A — no edit` (no look). Then print the **MODULE COVERAGE GATE**.

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
☐ Gate 4 — Overlay text typo (or N/A)
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

### Gate 4 OVERLAY TEXT TYPO CHECK — only when the overlay text was image-model-generated

Pull the poster: call `widecast_scene_inspector` with `action="overlay_poster"`, the topic `id`, the scene `voice_file`, `activate:true`; download the returned URL with `curl -L -s -o <local>.png "<url>"`; show the local PNG. Do not construct the URL manually.

**Order of operations is fixed: TRANSCRIBE FIRST, JUDGE AFTER.** Type out every visible string letter-by-letter like a proofreader BEFORE any other remark. The transcription IS the table; a one-line "text looks correct" without the table = the gate did not run.

```text
Gate 4 OVERLAY TEXT TYPO CHECK:
Applies check: <PASS applies — image-gen text | N/A typography_only/SVG | N/A no overlay text | N/A photo, no baked message text>
Overlay poster evidence: <MCP overlay_poster downloaded+shown> — local file path: <path>
Per-string transcription table (one row per visible string — title, label, value, badge, card line, callout, baked image text; typed letter-by-letter FROM THE POSTER IMAGE, never pasted from JSON/source):
| # | rendered string (as seen in poster) | intended copy (script/quote/talking_point/source) | char-level diff (for diacritic languages: if accent-stripped forms match but accented forms differ, that difference IS the spelling error) | verdict |
| 1 | <...> | <...> | <none | exact diff> | <PASS|FAIL> |
| … | | | | |
Typo/grammar/diacritic/glyph check: <PASS|FAIL> — no typo, grammar error, missing/wrong diacritic, malformed glyph, pseudo-text, wrong language/casing, wrong number/currency/%/symbol, wrong domain term; for Vietnamese, every tone mark, accent, horn/breve/circumflex, vowel mark, and `Đ/đ` is visibly correct
Action if FAIL: <regenerate/replace the image or its baked text | fix source text | N/A>
Verdict: <PASS rendered text | FAIL — fix and re-pull the poster>
```

If a string FAILS, fix it (regenerate/replace the image or correct the source), then re-pull the poster and re-run the table. Layout-only changes never need a new poster.

### Gate 4 OPENING POSTER CHECK — only for the `opening` roster row

The first content scene is the hook. Judge the poster as a whole (aesthetics, not just spelling) and keep it unless it clearly falls short.

```text
Gate 4 OPENING POSTER CHECK:
Opening poster evidence: <MCP overlay_poster downloaded+shown | no overlay present> — local file path: <path>
Overlay present: <yes | no>
Hook-in-1s: <PASS|FAIL> — a title/hook line reads and lands the video's core hook within ~1 second at 280×498
Not-a-flat-card: <PASS|FAIL> — not a plain centered card / horizontal text-bar; has real opening-poster presence
Copy/diacritics: <PASS|FAIL> — hook copy is correct, on-message for the whole video, Vietnamese marks/`Đ/đ` correct
Verdict: <PASS keep | REBUILD — <which checks failed> | AUTHOR — no overlay present, build a hook poster>
```

- **PASS keep** when the poster clears all three checks, OR is only marginally improvable — do NOT rebuild a decent opening just to chase prettiness.
- **REBUILD** (cap 1) when a check clearly fails: load `30_overlay_core` + `31_typography` + styles, re-author the overlay source, upload, and re-run this check on the AFTER poster. If the rebuild is still weak, keep whichever poster is better and note it — no second rebuild loop.
- **AUTHOR** when the opening scene has no overlay: build a short hook poster (this is the only scene where adding an overlay is allowed). The narrator stays fixed; the server places the overlay off the face.

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
- If Gate 3 applied, the PASS scan names its verdict (`PASS keep` / `FIXED + PASS`). If Gate 4 applied, the per-string table is present. If an edit was made, "AFTER shown: yes" is present.
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

Read the scene's `text` field in the context of the whole video. Look for: spelling/typos, STT mishearing, wrong industry terminology, wrong proper names, wrong figures, missing/extra words, sentences grammatically fine but wrong in meaning for the topic, a caption that doesn't match the audio.

If wrong, fix `text` with `modify_scene` branch (K) Segment text correction (keeps audio timing) before anything else.

If the correction is a domain term, proper noun, number, symbol, or entity (`Living Church` → `Living Trust`, `95` → `95%`, a company/person/product name), run a **semantic field sweep**: check/update `text`, `quote`, `talking_point`, `visual`, `keyword`, `pattern`/`sub_mode` when relevant; note any overlay text baked into an image that must be verified in Gate 4; re-pull `video_data` so later gates use the corrected context.

A term fixed in `text` but still wrong in `quote`/`visual`/overlay is a Gate 1 failure, not a partial pass. Do not edit from personal feeling — rely on the full script, topic, `visual`, `keyword`, and neighboring scenes.

---

## Step 2 — Understand the Scene's Role (routing)

Read `type` (HOOK / STAT / KEY POINT / DATA / FACT / CALL TO ACTION / thumbnail), `pattern` + `sub_mode`, `visual`, `keyword`, `quote`, `talking_point`, `text`, `show_narrator`/`active_roll`, `mediaUrl`/`mediaType`.

From these DATA fields alone, route the two conditional gates:

- **Gate 3 (background) applies** if the scene is NOT a grid background AND the narrator does not fill/cover most of the frame. Grid or full-frame A-roll narrator → Gate 3 is N/A.
- **Gate 4 (overlay typo) applies** if the overlay text was baked by an image model (`pattern="illustration"` sub_mode ≠ `photo_with_people`, or a chart/diagram/object pattern with image-generated text). `typography_only`/SVG text or no overlay text → Gate 4 is N/A. **Exception:** if this is the `opening` roster row (first content scene), Gate 4 ALWAYS runs — pull the poster and run the OPENING POSTER CHECK regardless of pattern.

No screenshot here — routing is a pure data read. Then run the applicable gates in order. Layout mechanics + `modify_scene` branches → `ai_video_editor/10_mechanics`; background work → `20_background`; overlay-defect fixes → `30_overlay_core` (+ matching content/style module).
