# -*- coding: utf-8 -*-
"""Fail if the style mirrors (text_axes.md / chart_axes.md) drift from the code
source of truth (image_prompt_kit.py, typography_canvas.py, text_styles_lib).
Run:  python3 ai_video_editor/styles/sync_check.py
Exit 0 = in sync; exit 1 = drift (prints the diff).

Note: the axis name 'structure' exists in BOTH the chart taxonomy (6 values) and
the text taxonomy (4 values); the two markdown SYNC lines are matched against
whichever code list they equal."""
import os, re, json, sys

SKILL_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))

def find_code_root():
    required = [
        "image_prompt_kit.py",
        "typography_canvas.py",
        os.path.join("gubo-remotion-player", "text_styles_lib", "manifest.json"),
    ]
    cur = SKILL_ROOT
    while True:
        if all(os.path.exists(os.path.join(cur, p)) for p in required):
            return cur
        parent = os.path.dirname(cur)
        if parent == cur:
            return SKILL_ROOT
        cur = parent

CODE_ROOT = find_code_root()

def rd_skill(p): return open(os.path.join(SKILL_ROOT, p), encoding="utf-8").read()
def rd_code(p): return open(os.path.join(CODE_ROOT, p), encoding="utf-8").read()

def names(var, src):
    m = re.search(var + r"\s*=\s*\[(.*?)\n\]", src, re.S)
    return re.findall(r'"name"\s*:\s*"([^"]+)"', m.group(1)) if m else []

ipk = rd_code("image_prompt_kit.py"); tc = rd_code("typography_canvas.py")

# canonical lists from code (keys = the SYNC keys used in the markdown)
code = {
    "surface": names("SURFACE", ipk), "finish": names("FINISH", ipk),
    "outline": names("OUTLINE", ipk), "color_application": names("COLOR_APPLICATION", ipk),
    "typography": names("TYPOGRAPHY", ipk),
    "CHART_PALETTES": names("CHART_PALETTES", ipk),
    "AMBER_CHART_PALETTES": names("AMBER_CHART_PALETTES", ipk),
    "_TYPO_PALETTES_CANVAS": names("_TYPO_PALETTES_CANVAS", tc),
}
mp = re.search(r"CHART_PATTERNS\s*=\s*\{(.*?)\}", ipk, re.S)
code["CHART_PATTERNS"] = sorted(re.findall(r'"([a-z_]+)"', mp.group(1)))
ma = re.search(r"_TYPO_AXES_CANVAS\s*=\s*\{(.*?)\n\}", tc, re.S)
typo = {k: re.findall(r'"([a-z_]+)"', v)
        for k, v in re.findall(r'"(\w+)"\s*:\s*\[([^\]]+)\]', ma.group(1))}
for k in ("depth", "fill", "typefaceKey", "casing"):
    code[k] = typo.get(k, [])
structure_chart = names("STRUCTURE", ipk)          # 6 chart structures
structure_text = typo.get("structure", [])         # 4 text structures
code["TEXT_PRESETS_COUNT"] = len(json.load(open(
    os.path.join(CODE_ROOT, "gubo-remotion-player/text_styles_lib/manifest.json"))))

# claimed lists from the two markdown mirrors
md = rd_skill("ai_video_editor/styles/text_axes.md") + "\n" + rd_skill("ai_video_editor/styles/chart_axes.md")
sync = re.findall(r'SYNC(?:_PALETTES|_SET|_COUNT)?:\s*([A-Za-z_]+)\s*=\s*([^\n>]+?)\s*-->', md)
claimed = {}
struct_claims = []
for key, vals in sync:
    if key == "structure":
        struct_claims.append([x.strip() for x in vals.split(",")]); continue
    claimed[key] = int(vals) if key == "TEXT_PRESETS_COUNT" else [x.strip() for x in vals.split(",")]

fail = []
def cmp(key):
    if key not in claimed: fail.append(f"[MISSING in markdown] {key}"); return
    c = claimed[key]
    if key == "TEXT_PRESETS_COUNT":
        if c != code[key]: fail.append(f"[{key}] md={c} code={code[key]}")
        return
    a, b = set(c), set(code[key])
    if a != b: fail.append(f"[{key}] md-only={sorted(a-b)} code-only={sorted(b-a)}")

for k in ["surface","finish","outline","color_application","typography","depth","fill",
          "typefaceKey","casing","CHART_PALETTES","AMBER_CHART_PALETTES",
          "_TYPO_PALETTES_CANVAS","CHART_PATTERNS","TEXT_PRESETS_COUNT"]:
    cmp(k)

# structure: each claimed list must equal one of the two code lists; both must be covered
hit_text = any(set(s) == set(structure_text) for s in struct_claims)
hit_chart = any(set(s) == set(structure_chart) for s in struct_claims)
for s in struct_claims:
    if set(s) not in (set(structure_text), set(structure_chart)):
        fail.append(f"[structure] md list matches neither code list: {sorted(s)}")
if not hit_text: fail.append(f"[structure/text] no md list == code {structure_text}")
if not hit_chart: fail.append(f"[structure/chart] no md list == code {structure_chart}")

if fail:
    print("STYLE SYNC: DRIFT DETECTED\n" + "\n".join(" - " + f for f in fail)); sys.exit(1)
print(f"STYLE SYNC: OK — mirrors match code (chart structure {len(structure_chart)}, "
      f"surface {len(code['surface'])}, finish {len(code['finish'])}, outline {len(code['outline'])}; "
      f"text structure {len(structure_text)}, depth {len(code['depth'])}, fill {len(code['fill'])}; "
      f"palettes {len(code['CHART_PALETTES'])}+{len(code['AMBER_CHART_PALETTES'])}+{len(code['_TYPO_PALETTES_CANVAS'])}; "
      f"{code['TEXT_PRESETS_COUNT']} presets)")
