# 32 · Charts — single_metric / bar / proportion / trend / structural, diversified

_Version: `modular-1.1` · module of the AI Video Editor Playbook (`SKILL.md`)._

> **Module of the AI Video Editor Playbook.** Master index + checklist + critical rules live in `SKILL.md`. **Load this when:** the scene pattern is single_metric, bar_chart, proportion_chart, trend_chart, or structural_diagram.
> Cross-refs: style recipes (3D/glossy/metallic/pattern-fill/…) → `styles/chart_axes.md`; core SVG rules → `30_overlay_core.md`; general drawing rules also in `30_overlay_core.md`.

---

> **⭐ LOAD CHAIN — before drawing the chart you MUST `Read` `styles/chart_axes.md`** (the look recipes: extruded-3D / soft-3D / glossy / metallic-gradient / pattern-fills / sticker / line-art + 24 warm + 25 cool palettes). Shipping flat-solid charts = you skipped it. **Checklist:** ☐ `30_overlay_core.md` ☐ `32_charts.md` (this) ☐ `styles/chart_axes.md`. (Labels/values also follow the text rules — `31_typography.md`.)

What to draw per chart pattern (each piece in its own `data-wc-object` group — granularity, §0.5):

| `pattern` (/`sub_mode`) | What to draw (each piece in its own `data-wc-object` group) |
|---|---|
| `single_metric` | A GIANT number/`%`/date center (~60% of height), bold contrasting font; small caption below. |
| `bar_chart` | N bars, a value label per bar, title top-center; minimal, no axes/gridlines. |
| `proportion_chart` | A donut split into N parts (`%` + label), bold center text. |
| `trend_chart` | A line up/down, data dots, X-axis labels, title; minimal. |
| `structural_diagram` | By substructure: flow (N steps →), cycle (circle + arrows), funnel (tiers), hierarchy (tree), before/after (2 panels), do-vs-dont (✓ green / ✗ red), list, formula (formula + labeled arrows), spectrum (bar + marker). |

**Chart text stroke policy.** Chart labels/values are secondary text, not title text. Keep each value, axis/bar label, legend row, and total badge readable at 280×498 with **no visible text stroke/outline**. If a label/value needs contrast, give it a clean chip/card/backplate or move it to a quiet area; do not stroke small text. Chart titles may use the global title treatment, but their visible text stroke is still capped at **2px on the 720 canvas**. Before upload and again on the final screenshot, check label/value collisions explicitly: total badges must not cover `$` values, bar labels must not sit inside dark/muddy cards, and no value may overlap another text/object.

### Diversify the chart look — do NOT ship flat-solid only

Pick a **look** for this video and reproduce it in SVG via **`styles/chart_axes.md`** (axes: structure incl. `extruded_3d_angular`/`soft_3d_rounded`/`hand_drawn`/`line_art_mono` · surface `solid/gradient/dots/grid/stripes/wavy/confetti` · finish `matte/glossy/textured_grain` · outline · color_application · typography; + 24 warm + 25 cool palettes). One look per video, kept across it, varied between videos. If you use any external/reference/preset chart image to choose the look, save/show it locally before evaluating it. (`hybrid_vertical` reuses this look on its lower data half — top half is a photo, see `33_patterns.md`.)
