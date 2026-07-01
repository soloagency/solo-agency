<!-- MIRRORED FROM CODE — do not drift. Source of truth: image_prompt_kit.py
     (STRUCTURE, SURFACE, FINISH, OUTLINE, COLOR_APPLICATION, TYPOGRAPHY,
     CHART_PALETTES, AMBER_CHART_PALETTES, CHART_PATTERNS).
     styles/sync_check.py FAILS if the SYNC lines below drift from code. -->

# Chart/graphic style library — taxonomy → SVG recipe

_Version: `modular-1.1` · module of the AI Video Editor Playbook (`SKILL.md`)._

The diversity vocabulary for **objects/shapes** in charts & diagrams (bars, donut arcs, nodes, pins, cards, icons). The native AI-image pipeline mixes 6 axes (~13,000 combos) — reproduce the chosen look in SVG. **Stop shipping flat-solid only.**

> **HOW TO USE.** Per video derive ONE chart look = `structure × surface × finish × outline × color_application × typography × palette` from brand/topic/rotation; keep it across the video. A `medium` structure (`hand_drawn`, `line_art_mono`) is self-contained and **locks/overrides** surface+finish+outline+typography. `is_3d` structures forbid flat-only and require shading kept INSIDE each silhouette (no cast shadows on the background).
> If you use any visual reference or preset image to choose these axes, save/show it as a local image before evaluating it; do not inspect reference images privately or via online URLs.

> **Reusable `<defs>` (declare once, reference by id):**
> ```svg
> <defs>
>   <linearGradient id="g_top" x1="0" y1="0" x2="0" y2="1">
>     <stop offset="0" stop-color="#7CE0FF"/><stop offset="1" stop-color="#2F6BFF"/></linearGradient>
>   <pattern id="p_dots" width="16" height="16" patternUnits="userSpaceOnUse">
>     <circle cx="4" cy="4" r="2.4" fill="#FFFFFF" opacity=".5"/></pattern>
>   <pattern id="p_grid" width="18" height="18" patternUnits="userSpaceOnUse">
>     <rect width="9" height="9" fill="#FFFFFF" opacity=".22"/><rect x="9" y="9" width="9" height="9" fill="#FFFFFF" opacity=".22"/></pattern>
>   <pattern id="p_diag" width="14" height="14" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
>     <rect width="7" height="14" fill="#FFFFFF" opacity=".22"/></pattern>
>   <pattern id="p_wave" width="40" height="14" patternUnits="userSpaceOnUse">
>     <path d="M0 7 q10 -7 20 0 t20 0" fill="none" stroke="#FFFFFF" stroke-width="2.5" opacity=".4"/></pattern>
>   <linearGradient id="gloss" x1="0" y1="0" x2="0" y2="1">
>     <stop offset="0" stop-color="#FFFFFF" stop-opacity=".55"/><stop offset=".45" stop-color="#FFFFFF" stop-opacity="0"/></linearGradient>
>   <filter id="grain"><feTurbulence type="fractalNoise" baseFrequency="0.9" numOctaves="2" result="n"/>
>     <feColorMatrix in="n" type="saturate" values="0"/><feComponentTransfer><feFuncA type="linear" slope="0.12"/></feComponentTransfer>
>     <feComposite operator="in" in2="SourceGraphic"/></filter>
>   <filter id="rough"><feTurbulence type="fractalNoise" baseFrequency="0.012" numOctaves="2" result="t"/>
>     <feDisplacementMap in="SourceGraphic" in2="t" scale="6"/></filter>
> </defs>
> ```

## Axis: structure — how each object is built (`is_3d`/`medium` noted)
<!-- SYNC: structure = flat_solid, flat_layered, extruded_3d_angular, soft_3d_rounded, hand_drawn, line_art_mono -->

| value | look | SVG recipe |
|---|---|---|
| `flat_solid` | clean filled silhouette, no volume | `<rect/circle/path fill="<color>">` |
| `flat_layered` | flat, overlaps read by lighter tint | where shapes overlap, the upper one = a lighter tint of the SAME hue (no shadow) |
| `extruded_3d_angular` *(3D)* | crisp extruded solid, lit front + dark sides | front face on top; behind it a `<polygon>` side wall offset `+dx,+dy` in a **darker** tint + a top face polygon in a **lighter** tint. For a bar: front `<rect>` + right side parallelogram (darker) + top parallelogram (lighter). All inside the silhouette. |
| `soft_3d_rounded` *(3D)* | pillowy rounded volume | rounded `<rect rx>`/`<circle>` filled with `url(#g_top)` (lit top→dark bottom) + a soft inner highlight ellipse near the top; no outline needed |
| `hand_drawn` *(medium — locks surface/finish/outline/typo)* | wobbly sketchbook | apply `filter="url(#rough)"` to the shape group for wobbly edges; fill with a flat crayon colour; pair with a rounded/casual font |
| `line_art_mono` *(medium)* | bold monoline, interiors empty | `fill="none" stroke="<color>" stroke-width="8" stroke-linejoin="round" stroke-linecap="round"`; one uniform stroke weight for every shape/icon/arrow |

## Axis: surface — fill texture (HEAVILY weight `solid`/`gradient`; patterns are RARE accents)
<!-- SYNC: surface = solid, gradient, dots_round, grid_squares, stripes_diagonal, stripes_vertical, stripes_horizontal, wavy_lines, confetti -->

Patterns auto-clip to the shape (a `<pattern>` fill only paints inside the shape's own geometry). Use a pattern on ≤1 emphasis shape, not all.

| value | SVG recipe |
|---|---|
| `solid` | `fill="<color>"` |
| `gradient` | `fill="url(#g_top)"` (brighter top, inside silhouette) |
| `dots_round` | `fill="url(#p_dots)"` over a base fill |
| `grid_squares` | `fill="url(#p_grid)"` |
| `stripes_diagonal` | `fill="url(#p_diag)"` |
| `stripes_vertical` | `<pattern>` of vertical `<rect>` bars (rotate `p_diag` to 0°) |
| `stripes_horizontal` | `<pattern>` of horizontal bars (rotate to 90°) |
| `wavy_lines` | `fill="url(#p_wave)"` |
| `confetti` | `<pattern>` scattering tiny `<rect>`/`<circle>` specks in mixed palette colours |

## Axis: finish — material sheen
<!-- SYNC: finish = matte, glossy, textured_grain -->

| value | SVG recipe |
|---|---|
| `matte` | nothing (no highlight) |
| `glossy` | overlay the TOP of the shape with a clipped sheen: clip `url(#gloss)` to the shape (`<clipPath>` = the shape) → a white→transparent gradient sits along the top edge inside the silhouette |
| `textured_grain` | apply `filter="url(#grain)"` to the shape group (subtle noise, no shine) |

## Axis: outline
<!-- SYNC: outline = none, thin_ink, sticker_white, bold_color -->

| value | SVG recipe |
|---|---|
| `none` | no stroke (fill defines the shape against the footage) |
| `thin_ink` | `stroke="#0C0F14" stroke-width="2.5"` |
| `sticker_white` | `stroke="#FFFFFF" stroke-width="10" stroke-linejoin="round"` (die-cut sticker) — pairs well with `is_3d` |
| `bold_color` | `stroke="<contrasting palette colour>" stroke-width="8"` |

## Axis: color_application — how palette colours map across objects
<!-- SYNC: color_application = multicolor, mono_tints, accent_on_neutral, duotone -->

A *selection rule*, not a render trick: `multicolor` = a different palette colour per object · `mono_tints` = one hue, separate by tint · `accent_on_neutral` = neutrals everywhere + ONE vivid accent on the key object · `duotone` = two colours alternating.

## Axis: typography (chart labels)
<!-- SYNC: typography = modern_sans, geometric_thin, rounded_friendly, elegant_serif, condensed_display -->
Map → a heavy VN-safe family (see text_axes.md typeface table): `modern_sans`→Be Vietnam Pro/Inter Black · `geometric_thin`→Montserrat (lighter) · `rounded_friendly`→Be Vietnam Pro · `elegant_serif`→DejaVu Serif · `condensed_display`→Anton/Roboto Condensed.

## Palettes (multi-colour, 4–5 swatches each)
<!-- SYNC_PALETTES: CHART_PALETTES = sunset_warm, forest_earth, berry_jewel, autumn_spice, retro_editorial, candy_pop, muted_pastel, emerald_gold, mono_warm, tropical_punch, coral_reef, vineyard, moss_clay, raspberry_cream, pumpkin_spice, jade_sand, plum_gold, terracotta_sage, magenta_lime, charcoal_amber, rose_olive, crimson_cream, forest_berry, mustard_grape -->
<!-- SYNC_PALETTES: AMBER_CHART_PALETTES = ocean_tech, navy_cyan_lime, blueprint_modern, arctic_signal, indigo_mint, teal_violet, blue_unlocked, royal_ice, midnight_aqua, electric_blue, cyan_depth, sapphire_lagoon, periwinkle_glass, ultraviolet_cyan, deep_space_blue, atlantic_greenblue, denim_signal, neon_cyberblue, iris_blue, cool_dashboard, glacier_navy, blue_violet_pop, steel_cyan, ink_azure, teal_blueprint -->
Two named palette banks (warm/editorial `CHART_PALETTES` + cool/tech `AMBER_CHART_PALETTES`), each 4–5 hexes. Pick by topic vibe; mirror the names here, fetch hex from `image_prompt_kit.py`.

## Patterns these styles apply to
<!-- SYNC_SET: CHART_PATTERNS = bar_chart, proportion_chart, single_metric, structural_diagram, trend_chart -->
(`hybrid_vertical` reuses the chart look on its lower data half; the top half is a photo — §5.4.)
