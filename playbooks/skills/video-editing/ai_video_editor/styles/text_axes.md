<!-- MIRRORED FROM CODE — do not drift. Source of truth:
     typography_canvas.py (_TYPO_AXES_CANVAS, _TYPO_PALETTES_CANVAS)
     gubo-remotion-player/text_styles_lib/manifest.json (curated presets).
     styles/sync_check.py FAILS if the SYNC lines below drift from code. -->

# Text style library — taxonomy → SVG recipe

_Version: `modular-1.1` · module of the AI Video Editor Playbook (`SKILL.md`)._

This is the **diversity vocabulary for TEXT** (titles, labels, values, quotes). The native Canvas typography renderer already mixes these axes across 142 curated presets — but when YOU author an overlay as SVG you must reproduce the look yourself. Below: every axis value with a **concrete SVG recipe**. Do not ship flat-only text.

> **HOW TO USE.** Per video, derive ONE text look = `structure × depth × fill × typeface × casing × palette` from brand/topic/rotation (see §0.5 "vary between videos"). Keep it across the video; change it next video. The TITLE may carry the richest treatment (bevel/metallic/gradient + a controlled outline); secondary text stays simpler and cleaner: solid heavy fill on a chip/card/quiet area, with **no visible text stroke/outline**. **Title body rule:** a title/hero is not a single `<text>`; build its body with **8–18 same-fill face copies** inside one title object, then add one final top copy with 0–2px stroke. Vary the copy count between titles/videos based on font, word length, and screenshot proof. **Authoring cap:** visible text stroke is **0–2px max on the 720 canvas**; use shadows/backplates/chips/shapes for extra separation, not thicker letter outlines.

> **Reusable `<defs>` (declare once per SVG, reference by id):**
> ```svg
> <defs>
>   <linearGradient id="grad" x1="0" y1="0" x2="0" y2="1">
>     <stop offset="0" stop-color="#FFD479"/><stop offset="1" stop-color="#F59E0B"/></linearGradient>
>   <linearGradient id="metal" x1="0" y1="0" x2="0" y2="1">
>     <stop offset="0" stop-color="#FDFDFD"/><stop offset=".28" stop-color="#C9CDD4"/>
>     <stop offset=".5" stop-color="#8A9099"/><stop offset=".62" stop-color="#EDEFF2"/>
>     <stop offset="1" stop-color="#9AA0A8"/></linearGradient>
>   <filter id="soft" x="-30%" y="-30%" width="160%" height="160%">
>     <feDropShadow dx="0" dy="7" stdDeviation="7" flood-color="#0A0D12" flood-opacity="0.55"/></filter>
> </defs>
> ```

## Axis: structure — the letter body/edge construction
<!-- SYNC: structure = solid, solid_outlined, hollow_outline, double_outline -->

| value | look | SVG recipe |
|---|---|---|
| `solid` | filled letters, no outline | Secondary text: `<text fill="#F4C61F">` on a calm/dark area. Title text: stack 8–18 same-fill copies first; one plain copy is too thin. |
| `solid_outlined` | filled + thin dark contour (hero/title over footage) | Title: 8–18 same-fill face copies with tiny offsets, then a final top `<text fill="#FFF" stroke="#0C0F14" stroke-width="2" paint-order="stroke">` (use 1.5–2px). Secondary text uses `solid` + chip/backplate, not stroke. |
| `hollow_outline` | outline-only, transparent interior (rare display treatment) | Avoid for normal short-form titles because it often reads thin. If used as a deliberate display effect, still add an 8–18x filled/gradient body or shadow mass behind it; never use for labels/values. |
| `double_outline` | sticker/retro without thick text stroke | Stack 8–18 same-fill title copies, then one top copy with 1.5–2px face stroke, plus a separate sticker/backplate shape or soft shadow behind the group. Do **not** stack 7–14px text strokes; if it needs that much separation, the placement/backplate is wrong. |

## Axis: depth — z-dimension of the letters
<!-- SYNC: depth = flat, beveled, soft_drop -->

| value | look | SVG recipe |
|---|---|---|
| `flat` | no depth | nothing extra |
| `beveled` | chiseled 3D edge | stack 3 copies same x/y: (1) hi-light copy offset `dx=-2 dy=-2` `fill="#FFFFFF" opacity=".55"`, (2) shadow copy offset `dx=3 dy=3` `fill="#0A0D12" opacity=".6"`, (3) the face copy on top (its real fill). The two offset ghosts read as a beveled edge. |
| `soft_drop` | floating, soft shadow | wrap the face `<text>` (or its group) with `filter="url(#soft)"` (the `#soft` def above). Cheaper alt: one dark offset copy `dx=0 dy=6 fill="#0A0D12" opacity=".5"` behind. |

## Axis: fill — how the letter face is painted
<!-- SYNC: fill = solid, gradient, metallic -->

| value | look | SVG recipe |
|---|---|---|
| `solid` | one flat colour | `fill="#F4C61F"` |
| `gradient` | smooth 2-colour, brighter on top | `fill="url(#grad)"` (the `#grad` def above; keep top brighter) |
| `metallic` | gold/chrome sheen (multi-band) | `fill="url(#metal)"` (the `#metal` def above; swap stops to gold for gold-metal) |

## Axis: typeface — family family family (renderer set, VN-safe)
<!-- SYNC: typefaceKey = bold_sans, condensed, slab_serif, rounded -->

cairosvg ignores numeric `font-weight` → **name the HEAVY family** (see overlay_core §font). Map the key → a VN-covering family:

| key | use family (heavy) |
|---|---|
| `bold_sans` | `Be Vietnam Pro Black` · `Inter Black` · `Montserrat Black` · `Open Sans Black` |
| `condensed` | `Anton` (single-weight, already heavy) · `Roboto Condensed` · `Noto Sans Condensed` |
| `slab_serif` | `DejaVu Serif` (heaviest VN-safe serif in the set) |
| `rounded` | `Be Vietnam Pro Black` (rounded-ish) — no dedicated rounded VN family ships; prefer bold_sans |

## Axis: casing
<!-- SYNC: casing = upper, title, sentence -->
`upper` = ALL CAPS (most punchy for hero titles) · `title` = Title Case · `sentence` = Sentence case. Apply by writing the literal text in that case (do not rely on CSS text-transform in the SVG).

## Palettes (2-colour text palettes — `[face, deep-bg/outline]`)
<!-- SYNC_PALETTES: _TYPO_PALETTES_CANVAS = ivory_charcoal, gold_maroon, coral_plum, mint_forest, sky_navy, amber_espresso, lilac_indigo, snow_slate, rose_wine, lime_olive, cobalt_white, crimson_cream, teal_ink, violet_pop, orange_navy, black_sun, sky_charcoal, emerald_gold, magenta_cream -->
Pick a palette by topic vibe (warning→crimson/orange; finance→emerald/teal; tech→sky/cobalt; luxury→gold/violet; youthful→coral/rose). The FACE colour must stay bright/punchy (hero-safe); the second colour is the outline/contrast. Full hex list lives in `typography_canvas.py:_TYPO_PALETTES_CANVAS` (mirror the names here, fetch hex from there).

## Curated preset library — 142 ready looks with previews
<!-- SYNC_COUNT: TEXT_PRESETS_COUNT = 142 -->
`gubo-remotion-player/text_styles_lib/manifest.json` holds **142 curated presets** (each: `typeface, casing, fillMode, bevel, depth3d, outline, shadow, italic, letterSpacing, body colours`) + a **preview PNG** per preset in `text_styles_lib/imgs/`. When you want a strong look fast: build/save a local contact sheet from a few preview PNGs, show that local image to the user, then pick one that fits the topic and reproduce its axes in SVG via the recipes above. Do not inspect preset previews privately. (This is the same library the editor picker shows.)
