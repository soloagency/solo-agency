# 33 · Other patterns — map / table / timeline / checklist / quote / illustration / hybrid / real_entity / typography_only / narration_only

_Version: `modular-1.1` · module of the AI Video Editor Playbook (`SKILL.md`)._

> **Module of the AI Video Editor Playbook.** Master index + checklist + critical rules live in `SKILL.md`. **Load this when:** the scene pattern is NOT a chart and NOT pure typography — i.e. one of the patterns listed above.
> Cross-refs: photo reuse (illustration/hybrid/real_entity) → §5.4 in `30_overlay_core.md`; chart half of hybrid → `32_charts.md`; text rules → `31_typography.md`.

---

> **⭐ LOAD CHAIN — before drawing you MUST `Read` the lower tiers this pattern needs:**
> - **any photo pattern** (`illustration` `photo_*` · `hybrid_vertical` top half · `real_entity`) → §5.4 in `30_overlay_core.md` (reuse a REAL image, never draw it photoreal);
> - **`hybrid_vertical`** → ALSO `styles/chart_axes.md` (its lower data half is a chart);
> - **`map_chart`** → `styles/chart_axes.md` note (SVG can't draw an accurate map → reuse a real map image);
> - **`quote_card`/`timeline_events`/`checklist_tips`/`comparison_table`** (text-heavy) → the text look in `styles/text_axes.md` (`31_typography.md`).
>
> **Checklist:** ☐ `30_overlay_core.md` ☐ `33_patterns.md` (this) ☐ §5.4 if a photo pattern ☐ the relevant `styles/*.md` for the look.

What to draw per pattern (each piece in its own `data-wc-object` group — granularity, §0.5):

| `pattern` (/`sub_mode`) | What to draw (each piece in its own `data-wc-object` group) |
|---|---|
| `map_chart` | N pins (`name + value`) over a map of the region; bold title. **⚠ SVG's weak spot — you CANNOT hand-draw an accurate geographic map** (a freehand `<path>` is only a crude/stylized blob, verified). For real geography **reuse a real map image** (§5.4): download/save candidate map images locally, show them to the user, then choose/place the pin/label groups over the selected map; hand-draw an abstract shape ONLY when a rough/stylized map is acceptable for the beat. |
| `comparison_table` | 2–3 columns × N rows; bold header; winning value highlighted green; bold title. |
| `timeline_events` | A thick axis, N markers; date above + event (≤4 words) below; bold title. |
| `checklist_tips` | A vertical list of N items: numbered circle + icon + short label (≤5 words); even spacing; bold title. |
| `quote_card` | A large `"` top-left, the quote centered (≤12 words) bold, a closing `"` bottom-right, `— Name` italic. |
| `illustration` / `hybrid_vertical` / `photo_*` | Needs a **realistic image** → **reuse** one (§5.4), don't draw it photoreal. Download/save the candidate image locally and show it before judging or composing. Compose: the image in one `<g data-wc-object="photo"><image .../></g>` (e.g. top ~55%) + the data/labels as other groups below. For extracted/current-spec photos, embed the image as a `data:image/...;base64,...` href inside the SVG; do not rely on a remote `<image href="https://...">` unless the post-upload screenshot proves it rendered. (For `hybrid_vertical` the top half is the PHOTO; you only draw the chart/labels in the lower half.) |
| `real_entity` | A named real entity → **NOT drawn**: reuse/search a real photo (`visual` is the search query — §4 / §5.4), download/save/show locally before choosing, embed via `<g data-wc-object="photo"><image .../></g>`, then add name/label/callout groups over it. |
| `typography_only` | Text only — §5.2. |
| `narration_only` | No overlay (`remotion_spec="none"`) — leave it. |

**Panel/card/checklist text is secondary text.** It must pass the secondary text gate separately from the title: no visible stroke/outline on small text, no dark text on dark panels, no low-contrast label tucked inside an icon/card, and no overlaps between panel labels, badges, icons, and card copy. If a panel label is not readable at 280×498, simplify the words, enlarge it, use a clean chip/backplate, or rebuild the panel; do not declare PASS because the title looks good.

> Patterns needing a REAL photo (illustration `photo_*`, hybrid_vertical top half, real_entity) are NOT drawn — reuse a real image via `<image>` per §5.4 in `30_overlay_core.md`. **Every reused photo/map candidate is visual evidence: local file + shown to user before the agent evaluates it.** `map_chart` is SVG's weak spot (see its row + `styles/chart_axes.md` note).
