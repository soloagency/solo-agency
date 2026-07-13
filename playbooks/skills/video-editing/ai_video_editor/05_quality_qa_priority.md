# §7 Quality Standard + §9 Priority Order

Load this module:
- **Before declaring `Scene N: PASS`** — to scan against the Quality Standard alongside the 5 DoD gates (`ai_video_editor/03_dod_gates`).
- **Whenever you hit a conflict between gates** — to use the priority order to break the tie.

There is **no §8 whole-video QA pass** — each scene is complete at its own PASS and is not revisited.

---

## §7 Quality Standard for a Passing Scene

To declare **`Scene N: PASS`** confirm every applicable item below AND the applicable DoD gates. Any miss → `FAIL`, fix, re-scan. The standard is short on purpose — the server owns everything mechanical (placement, dead-zone, face, composition); the agent only certifies the blind spots.

A scene passes when:

- `text` is correct in context, with no significant STT errors.
- **Background fit (Gate 3, when it applies):** the background clip serves the sentence being spoken; for location-sensitive industries/scenes the geography/culture/currency cues match the target market (wrong-country footage, foreign currency, wrong signage/language/road context = FAIL even if the object is otherwise relevant). Grid is an intentional exception within the ≤3-scene shared-grid cap. N/A when the scene is grid or a full-frame A-roll narrator.
- **Image-gen text correctness (Gate 4, when it applies):** for illustration/chart/diagram/object overlays with image-model-baked text, the Gate 4 **per-string transcription table** was produced from the overlay poster (typed letter-by-letter from the image, transcribe FIRST) and every string is typo-free, grammar-correct, right language/diacritics/casing, and preserves numbers/currency/%/proper nouns/domain terms. For Vietnamese/diacritic languages, every tone mark, accent, horn/breve/circumflex, and `Đ/đ` is visibly correct. N/A for SVG/typography text (deterministic, never misspells) and for scenes with no image-baked text.
- **If an edit was made:** the ONE AFTER look (poster for a text fix / composite for a background fix) was shown and confirms the fix reads as intended — that look is the save-confirmation (no separate re-pull needed).
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
6. Limit expensive image pulls, but never skip an applicable plate/poster/AFTER look to save cost.
