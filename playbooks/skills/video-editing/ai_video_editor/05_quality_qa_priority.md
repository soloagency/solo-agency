# §7 Quality Standard + §9 Priority Order

Load this module:
- **Before declaring `Scene N: PASS`** — to scan against the Quality Standard alongside the 5 DoD gates (`ai_video_editor/03_dod_gates`).
- **Whenever you hit a conflict between gates** — to use the priority order to break the tie.

There is **no §8 whole-video QA pass** — each scene is complete at its own PASS and is not revisited.

---

## §7 Quality Standard for a Passing Scene

To declare **`Scene N: PASS`** confirm every applicable item below AND the applicable DoD gates. Any miss → `FAIL`, fix, re-scan. The standard is short on purpose — the server owns everything mechanical (placement, dead-zone, face, composition); the agent only certifies the blind spots.

A scene passes when:

- `text` is correct in context and according to the scene's language: spelling/orthography, applicable script-specific characters/marks, Unicode, capitalization, punctuation, spacing, STT meaning, proper nouns, domain terms, and numbers/dates/units/%/currency all pass. Do not normalize or convert to another language or locale.
- **Background fit (Gate 3, when it applies):** the background clip serves the sentence being spoken; for location-sensitive industries/scenes the geography/culture/currency cues match the target market (wrong-country footage, foreign currency, wrong signage/language/road context = FAIL even if the object is otherwise relevant). Grid is an intentional exception within the ≤3-scene shared-grid cap. N/A when the scene is grid or a full-frame A-roll narrator.
- **Overlay integrity/provenance (Gate 4):** deterministic overlay output passes from server data when source/current-text hashes match, render succeeded, there is no truncation or missing glyph, and scene/`voice_file` identity matches. Never OCR deterministic output. No overlay → N/A. An unverified non-deterministic/image-baked, user-uploaded, manually overridden, or legacy source uses the narrow exceptional fallback and cannot PASS on assumption.
- **If an edit was made:** saved `text` plus refreshed integrity/provenance data confirms a text edit; an AFTER composite confirms a background edit. Do not pull an overlay poster merely to reconfirm deterministic text.
- **Module coverage:** the modules the applicable gates needed were loaded (`20_background` for Gate 3; `30_overlay_core`/`31`/`32`/`33` only if an overlay defect was fixed). A step done "from memory" without opening its module does not count as PASS.

Do NOT add PASS criteria for placement, dead-zone, face-clearance, title thickness, or composition balance — the server guarantees those and re-checking them is out of scope.

---

## §9 Priority Order When There's a Conflict

When two gates pull in opposite directions, decide by this order (top wins):

1. Don't get the content wrong (`text`, data, meaning).
2. Background must fit the narration + target-market geo/context (or be an intentional grid).
3. Image-model-baked text must be typo-free (regenerate/replace if not).
4. Prefer the least destructive fix; preserve good existing visuals.
5. Don't break previous intentional edits (e.g. `overlay.narrator.touched=true`, `remotion_spec="none"`).
6. Limit image pulls, but never skip an applicable Gate 3 plate or Gate 5 background AFTER composite. An overlay poster is permitted only in Gate 4 exceptional fallback.
