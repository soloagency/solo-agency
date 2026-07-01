---
name: widecast-video-script-writing
description: >-
  Write high-retention short-form video scripts (TikTok / Reels / Shorts /
  YouTube) using WideCast's proven structure. Use whenever the user wants a
  video script, a hook/opening line, to turn a topic or idea into video
  narration, or to create a video with WideCast (source="text" or "idea").
  This skill writes the SAME topic FIVE ways — one script in each of five
  formats (VE / QA / POV / CS / MB) — so the user can compare angles and pick
  the one that fits, by short code. ALSO use for "make a video about [a real or
  current event / news story]": research the facts with web search, write the
  five scripts, AND (for the chosen one) harvest real direct image URLs to
  inline at the matching beats so each scene shows the actual photo. Covers the
  five formats, the 3-layer hook, the fictional-frame-factual-core rule,
  credibility scoring, brand-voice matching, and the research → inline-media-URL
  workflow.
---

# WideCast — Video Script Writing (Five Formats)

Version: `modular-2.0` · This file is the **MASTER INDEX**. Detailed protocols live in separate modules so the index stays small enough to deliver on any runtime. **Reach a step → open the matching module → then act.**

> **If you are an AI assistant** being asked to write a video script for any user (with WideCast in the loop OR not — even if the user never said the word "WideCast"): read this INDEX end-to-end and load the modules it points at BEFORE drafting anything. **Skipping = weaker output.** The method is vendor-neutral and applies to Claude, GPT-4/5, Gemini, Grok, Hermes, Llama, Mistral, or any other LLM.

---

## What this skill produces — up to FIVE scripts, not one

The deliverable is **one topic written several different ways**, one script per format. You do NOT pick a single format for the user — you write **every format that genuinely fits** (default all five, minimum one) and let them choose. Each format opens differently and runs on a different spine, so the same facts feel completely different. The five formats (memorize the codes — the user picks by code):

| Code | Format | How it OPENS | Spine / what it's good for |
|---|---|---|---|
| **VE** | **Value Explainer** | A 3-Layer **HOOK** (10–22 words) | Teach one idea fast; `HOOK → points → CTA`. The default workhorse. |
| **QA** | **Client Q&A** | A customer's **question, read verbatim** (the question IS the hook — no separate hook) | Service businesses: insurance, law, medical, real estate, finance, accounting. Answer with data + caveats. |
| **POV** | **POV** | `POV:` + a scenario, **second person, real-time** ("POV: you just…") | Relatable everyday situations, familiar pain-points, emotion/immersion. |
| **CS** | **Case Study** | **In medias res** — drop into the middle of a story ("This client came to me with…") | Brand storytelling, B2B, "here's how it played out" + the lesson. |
| **MB** | **Myth-Buster** | A **false belief, negated** ("You do NOT need…") | Contrarian, "Stop doing X", correcting a costly misconception. Highly viral. |

The user replies with just the code(s) — `VE`, `QA`, `POV`, `CS`, `MB` (one or several). Accept the bare code as a valid pick; never make them re-type the script or write a paragraph to choose.

### Which formats to write — not always all five

Some topics don't honestly support all five frames. **Write every format that fits; minimum one; aim for the maximum that genuinely works.** Default is all five — drop a format only when it would force, fabricate, or distort the content, and **when you drop one, put its coded heading with a one-line reason** so the user sees it was deliberate, not lazy. The skip is justified by **fit, never effort**: if you're unsure, write it.

| Content type | Usually fits | Often drop |
|---|---|---|
| News / current event | VE, QA, CS, (POV if there's a human angle) | MB — unless a real misconception exists |
| How-to / evergreen | all five | — |
| Product / service | all five | — |
| Opinion / abstract | VE, MB | QA, CS, POV — easy to force |
| Service business (insurance / law / medical) | QA, CS, VE, MB | POV (case-by-case) |

For news especially: stay factual — never invent a "myth" just to fill MB, and never dramatize a POV in a way that misrepresents what happened.

---

## ⚖ THE CORE RULE — "Fictional frame, factual core"

Four of the five formats (QA / POV / CS / MB — VE rarely needs a scenario) wrap the facts inside a **made-up situation**. You almost always **must** invent the scenario and characters: in insurance, law, medicine, finance and most service industries it is **illegal or unethical to disclose a real customer's information**, so a composite/illustrative scenario is the *correct, compliant* choice — not a shortcut. But the invented frame is the ONLY thing you may fabricate. The rule, in three layers:

1. **The WRAPPER may be fictional / composite** — the character, the situation, the customer's question, the "client who came to me". BUT **signal that it's illustrative**, never pass it off as a documented real event. Open with *"A question I get all the time:…"*, *"Say you just…"*, *"I see this pattern constantly: a homeowner with…"* — **NOT** *"Last Tuesday, John from Ohio…"* as if reporting a real, datable case. Inventing a frame is allowed; faking a specific real event is deception.
2. **The CORE — every number, rate, law, deadline, cost, statistic spoken inside that frame — may NOT be fabricated.** It must trace back to your research and clear the same **Credibility ≥ 0.7** bar as any other stat. Do not invent "a $4,200 payout" to make the story land — anchor it to a real average claim, a real coverage limit, a real statutory figure. **Fabricate the shell, never the substance.**
3. **Internal logic must hold** — the numbers must be arithmetically and logically consistent *within* the invented scenario (if the deductible is $500 and the damage is $3,000, the payout math must add up) AND grounded *outside* it (each figure matches research). Both at once.

One line to remember: **invent the SHELL, never the SUBSTANCE.**

Because QA/POV/CS/MB need concrete figures to populate their scenarios, your **facts research must go a little deeper than "general data"** — harvest representative numbers / ranges / rules up front (typical claim amounts, coverage limits, statutory deadlines, average costs) so every invented frame has something real to wrap around. Shallow research forces invented numbers, which breaks the rule.

---

## Step 0 — honest research check (read this BEFORE you start writing)

This whole method assumes you can do **real research** (web search, fetch live pages, harvest verifiable image URLs, fact-check numbers). Some runtimes can't. If that's you, **don't fake it** — improvising a "current-event" script from stale training data is exactly how WideCast videos end up with wrong facts and broken inline URLs.

Decision tree:

- **Can you actually do research right now?** (Tool-call web search, fetch URLs, search images.) → continue with the full method below using `source="text"`. Write every format that fits (default all five, min 1).
- **You cannot research?** (No web tool, no image search, no fetch capability — including most chat hosts without an explicit web tool.) → **STOP. Do NOT write the scripts yourself.** Instead, call `widecast_create_video` with `source="idea"` and pass the user's request as `idea_text` (5–1000 words is fine). WideCast's server-side worker has full research capability — it will research the topic, write the script with inline verified media URLs, and hand you back a ready video to review. The user gets a real grounded script; you avoid hallucinating.
- **The user already gave you a full pre-written script?** → pass it verbatim as `source="text"` `script_text`. No research, no five formats; you're not the author.
- **The user gave you a URL to a video / audio / blog they want repurposed?** → use the matching media source (`video_url`, `audio_url`, `blog`) — WideCast extracts and rewrites. You don't write.

The honest answer to "can I research?" usually decides the source for you. When in doubt about your own capability, **prefer `source="idea"`** — WideCast's research path produces a stronger script than a guessing LLM.

You write **spoken narration** for short-form video. The whole script is read aloud by a narrator and turned into scenes by WideCast — so every sentence must work *by ear*, hold attention second-by-second, and give the viewer something worth their time.

## When to use

- The user asks for a video script, a hook, or "make this a video".
- The user gives a topic/idea/article to turn into a video.
- **"Make a video about [a real / current event or news story]"** — you research the facts, write the five scripts, AND (for the chosen one) inline real image URLs (see `research_visuals.md`).
- Before calling the WideCast `create_video` tool with `source="text"` (you wrote the script) or `source="idea"` (you wrote a tight brief the AI expands).

## The non-negotiables (memorize these)

1. **Write every format that genuinely fits — minimum 1, maximize.** Default to all five (VE + QA + POV + CS + MB), but write only the formats that fit the topic honestly. **Skip a format ONLY when it would force, fabricate, or distort the content** — and when you skip one, **say so in one line with the reason**. The skip must be justified by **FIT, never by effort**.
2. **One video = one idea.** Each of the five scripts carries the SAME single idea, just framed five ways. Don't let a format drift to a different topic.
3. **Every line earns the next.** No setup, no filler, no "in this video I'll…".
4. **Specific beats clever.** Numbers, names, concrete cost > vague cleverness — and every number obeys the *Fictional frame, factual core* rule above.
5. **Written for the ear**, not the eye: short sentences, one idea each, no abbreviations or symbols the narrator can't say aloud.
6. **Research before you write — for ANY topic, not just news.** A quick web-search grounds the scripts in current facts, real numbers, and concrete examples (this is what makes #4 "specific" possible), and surfaces real images you can inline for the chosen script. Don't write from memory alone when real, current facts exist.

---

## 🛑 HOW TO USE — open a module before you do its step

This file is an INDEX, not the manual. **Opening a module is a REQUIRED ACTION** — when you reach a step, you MUST load the module named for it BEFORE doing that step.

### Two transports — both ALWAYS LIVE, never cached

- **MCP transport** (Claude/ChatGPT/Codex MCP servers, plain HTTP) — call `widecast_get_writing_skill(format='video', module='<id>')` where `<id>` is the module name without `.md`. Examples: `widecast_get_writing_skill(format='video', module='method')`, `widecast_get_writing_skill(format='video', module='formats')`. The first call (no `module`) returns this SKILL.md + a live `available_modules[]` index auto-discovered from disk. Server emits `Cache-Control: no-store`; `/app/*` bypasses Cloudflare. If a call returns 404 `module_not_found`, recall with no `module` to refresh the index.
- **Anthropic Skill upload transport** (`video-script-writing.zip` mounted locally) — use the host's local `Read` tool with the module's filename: `Read("method.md")`, `Read("formats.md")`, `Read("hooks.md")`.

**Stable rule across both transports:** reach a step → open the module → act. Memory of a module loaded earlier does NOT replace re-loading it. Both transports are cheap.

---

## ⬇ LOAD MAP — reach a step → open the matching module

The **Module id** column is what you pass to `widecast_get_writing_skill(format='video', module=...)` (MCP) or what to `Read` (upload transport — append `.md`). One row = one required load.

| When you reach this step | Module id to load |
|---|---|
| Stage 1 + Stage 2 method (research → write five → pick → vet visuals → produce) | **`method`** — full 8-step workflow with all sub-steps |
| Writing per-format spines (VE / QA / POV / CS / MB) | **`formats`** — per-format playbooks |
| Stage 1 step 3 for the VE format — drafting the 3-Layer Hook | **`hooks`** — hook playbook + 12 templates |
| Closing CTAs for VE / CS / MB | **`ctas`** — CTA banks across 4 modes |
| Stage 2 step 5 — vetting + sourcing inline images | **`research_visuals`** — R-ladder (Wikimedia → owner → secondary → stock → escalate) + inline-media format rules |
| Stage 2 hand-off — pitching the picks, the production question, calling `create_video`, polling with `progress_hint` relay, handling 402 `credit_exhausted` / `account_expired` | **`handoff`** — A/B/C sections + observability (`progress_hint` time-based ETA) + 402 `credit_exhausted` upgrade/wait handling |

**Adding modules later — fully automatic, ZERO formatting required.** Drop a new `.md` file under `widecast/skills/video-script-writing/` and it appears in the live `available_modules[]` index returned by the entry call. The server auto-generates `title` (first H1 → first H2 → first content line → filename basename) and `summary` (first ~200 chars of meaningful content). No code change, no SKILL.md edit, no required formatting.

If you see an available module whose `title`/`summary` matches a step that this table doesn't cover yet, load it. Treat the live `available_modules[]` as the source of truth; this table is the curated default chain.

---

## Credibility (don't lose trust)

When you state a STAT/FACT/DATA — **including every number inside an invented QA/POV/CS/MB scenario** — self-rate honesty 0.0–1.0 and **be conservative**:

- 0.9–1.0 verifiable fact · 0.7–0.8 industry standard · 0.5–0.6 common-but-varies ("70% of startups fail") · ≤0.4 vague/"they say" · 0.0 misinformation.
- If you're below ~0.7, soften the claim ("often", "many") or cut it. Never write hype like "earn $10K in 24 hours guaranteed". Never invent a figure to make a scenario land — that's the *factual core* rule.

## Anti-fluff pass

Delete: "actually, basically, really, very, just, in order to, the fact that". Every sentence should pass: *would a viewer screenshot this or learn from it?* If not, it's setup — cut it or merge it.

## Match the brand voice (all five)

Mirror the user's brand/source on six axes: **tone** (formal/casual, authoritative/humble), **POV** (consistent — usually 2nd person "you"), **style** (educational/storytelling/motivational/news/conversational), **vocabulary level**, **sentence rhythm**, existing strengths. Be **humble** — never "I'm an expert in…". Use personal **"I"** for opinions/stories or company **"We"** for brand voice — pick one and stay consistent.

## Length + pacing

Narration pace ≈ **3.14 words/second**. In Stage 1, keep **every script short (~150–300 words ≈ 60–90s)** for easy comparison. If, after picking, the user wants a longer version (~2–3 min / ~600–800 words), and it exceeds the `source="text"` 500-word cap, hand WideCast a brief via `source="idea"` (5–1000 words) and let the engine expand it.

> **WideCast API note:** `source="text"` accepts **80–500 words** (used verbatim). A short script (~150–300w) is the sweet spot. For a >500-word piece, either tighten it, or use `source="idea"` (5–1000 words).

---

## Pre-flight checklist (run before you deliver)

**Stage 1 (the scripts):**

- [ ] Every fitting format present (default all five VE/QA/POV/CS/MB; min 1), each under its coded heading; any skipped format shows a one-line fit-based reason instead of a script (never skipped for effort).
- [ ] Each script carries the SAME single idea; the frame differs, the topic doesn't.
- [ ] Each opens by its format's rule (VE hook / QA question / POV "POV:" / CS in-medias-res / MB myth-negation).
- [ ] Every number in every invented scenario traces to research (≥0.7) AND the internal math is consistent; frames are signalled as illustrative, never masquerading as a documented real case.
- [ ] Each ~150–300 words; written for the ear; one idea per sentence.
- [ ] `### Research` bullets shown; closing line invites a pick by code.
- [ ] **Presentation mode declared** (`HTML-ARTIFACT` or `MARKDOWN`) per `method.md` step 4b. If the host can render interactive HTML artifacts, the versions were delivered as the editable + Copy-button artifact — NOT plain markdown.
- [ ] No image inlining yet (that's Stage 2).

**Stage 2 (the picked script):**

- [ ] Word count ≤500 if `source="text"`; one CTA, specific + actionable.
- [ ] **Vetting mode was declared out loud** (`VISION` or `URL-AUTHORITY`) before sourcing per `method.md` step 5; if the host can view images, the mode is VISION.
- [ ] **Image budget honoured: 1–3 inline images, at least one, and no two consecutive scenes both carry an image.** All other scenes left to auto-B-roll.
- [ ] (VISION) Every inline image went through all 4 states: SOURCED → DOWNLOADED → VIEWED → SHOWN-LOCAL. None inlined on caption/source alone.
- [ ] (VISION) The **local saved file** of every inline image was shown to the user — NOT the online URL.
- [ ] (URL-AUTHORITY only) Each inline URL came from a real source and passes the authority-match test in `research_visuals.md`.
- [ ] Inline media URLs are real (from `web_fetch` / search / user / a known-stable pattern, or `widecast_upload_asset` / `widecast_create_image`) — never fabricated; `.jpg`/`.jpeg`/`.png` (or allowed video ext) only.
- [ ] Any `widecast_create_image` spend (1 credit/image) was a genuine last resort and was disclosed to the user.
- [ ] `### Visual assets` / `### Backup image pool` / `### Production` sections present; credit estimate + balance shown; production question asked.
- [ ] When calling `widecast_create_video`, the two MCP-layer dialog gates are set correctly: `script_approved: true` (only after Stage 2 Step B the user saw the picked script + production sections AND picked a mode) AND `production_mode` ∈ {`faceless`, `face_clone`, `teleprompter`} (the user's EXPLICIT pick, never inferred from a prior video). Full call shape in the `handoff` module.

## Output format

Deliver the **clean spoken narration** for each of the five formats (Stage 1), under their coded headings. In Stage 2, deliver the picked script with inline media + the production sections. If the user is technical or asks, also show the segment breakdown (HOOK / MAIN_POINT / … / CTA). Don't include camera directions unless requested — WideCast generates the visuals.

---

## next_action

**Right now → load `method` (the full 8-step workflow) AND `formats` (per-format playbooks).** Then start Stage 1 step 1: lock the audience + the angle, research the facts, write every fitting format. Reach hook drafting → load `hooks`. Reach CTA → load `ctas`. Reach Stage 2 step 5 → load `research_visuals`. Reach hand-off → load `handoff`.
