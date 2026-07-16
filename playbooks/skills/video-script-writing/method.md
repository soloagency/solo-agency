# The method — Stage 1 + Stage 2 (full step-by-step)

Load this module when you reach "the method" in `SKILL.md`. It carries the full Stage 1 (research → write five → pick) and Stage 2 (vet visuals → cost → produce) workflow, which was extracted from the master so the index could stay small.

---

## Stage 1 — research once, write five, hand off for a pick

### 1 — Lock the narrator + the audience + the angle (once, shared by every script)

**First the NARRATOR — who is speaking, by name and trade?** Pull their name, profession/business, services, and market from this conversation, brand context, or their WideCast profile, and write as that person throughout (full rules in *WHO IS SPEAKING*, `SKILL.md`). The narrator IS the professional — their stance, their composite cases, their CTA.

Then write one sentence: *"This is for **[who, by situation not job title]** who **[pain/desire]**, and the angle is **[the one surprising thing]**."* Situation-based targeting ("anyone who's ever rewritten a cold email five times") beats labels ("marketers"). If you can't name the ONE idea, stop and narrow. This narrator + audience + angle is shared across every script.

**Infer this — don't interrogate the user.** The topic almost always implies the audience, the goal, and the angle, so decide them yourself and move on. **Do NOT ask obvious questions** like "who's the target audience?" — the topic already answers them. **Ask back ONLY when something essential is genuinely missing AND it would change the scripts** — e.g. the topic is a bare acronym with two meanings, or "my product" with no hint what it is. When you must ask, ask **one short, batched question (1–2 items max)** — a missing-narrator item counts as one of the two — never a questionnaire.

### 2 — Research the FACTS (once, shared by every script)

Use your web search tool (ChatGPT `web.run` with `search_query`; Claude the `web_search` tool; Gemini Google Search; Grok X + web search). For real or current events / products / places / people, verify who/what/where/when + concrete numbers across at least 2 authoritative sources.

**Go deeper than "general data."** Because QA/POV/CS/MB will wrap invented scenarios around real figures, harvest **representative numbers, ranges and rules** now: typical amounts, limits, statutory deadlines, average costs, common edge-cases. Apply the Credibility bar (in `SKILL.md`) to each. Write a 2–4 bullet research summary you'll show under `### Research`. Do not skip this even for a topic you think you know.

Research anchors the **public tier** of the CORE RULE (laws, rates, stats, real events). **Practitioner-experience figures** (typical client costs/outcomes in the narrator's book of business) may instead be composite and true-to-context — see the two-tier CORE RULE in `SKILL.md`.

### 3 — Write every fitting format (default all five; min 1) — compact narration, ~240–280 words each

Write the same idea in **every format that genuinely fits** — default all five, but only the ones that work honestly for this topic (see *Which formats to write* in `SKILL.md`); minimum one, maximize. For any format you skip, you'll show its coded heading + a one-line fit reason in step 4 (not a script). Keep each script at **~240–280 words (≈75–95s at 3.14 words/second; hard cap 300)** so the user can compare formats quickly. **Do NOT inline images yet** — Stage 1 is bare narration so the formats are easy to scan; the image work happens in Stage 2 only for the picked script(s). Follow each format's opening rule and spine (full per-format playbooks in `formats.md`):

- **VE — Value Explainer:** open with a 3-Layer Hook (10–22 words), then LIST or NORMAL body, then one CTA.
- **QA — Client Q&A:** open with the customer's question verbatim (composite, signalled as common), then a crisp direct answer, then evidence/numbers, then the caveat/exception, then a soft CTA. No separate hook.
- **POV — POV:** open with `POV:` + a second-person, real-time scenario, then walk the viewer through it as if it's happening to them, landing the lesson.
- **CS — Case Study:** open in the middle of a story (composite client), build the tension, resolve it, end on the transferable lesson.
- **MB — Myth-Buster:** open by negating a costly false belief, explain why it's wrong, give the real answer with proof, then the CTA.

Every number in all five obeys *Fictional frame, factual core* (see `SKILL.md` two-tier CORE RULE). Honour the shared craft: the narrator persona (their first person, their stance, their CTA), specific over clever, written for the ear (spoken-voice spec in `SKILL.md`), one idea per sentence, each format's emotional arc (`formats.md`), the INSIDER BAR (≥1 practitioner-only detail per script), honest stats (Credibility ≥ 0.7 or soften/cut), anti-fluff.

### 4 — Hand off the scripts in ONE message, ask for a pick by code

Assemble the hand-off using this structure (sections required; exact wording flexible — the chat host renders markdown inline):

- `### Research` — the 2–4 bullets from step 2 (shared facts behind every script).
- Then one block per format, each under a coded heading so the user can pick by code. Use exactly these headings:
  - `### VE — Value Explainer`
  - `### QA — Client Q&A`
  - `### POV — POV`
  - `### CS — Case Study`
  - `### MB — Myth-Buster`
  Under each format you **wrote**, the clean spoken narration. Under each format you **skipped**, instead of a script put a one-line fit reason (e.g. `### MB — Myth-Buster` → *"skipped: no genuine misconception to bust here"*) so the user sees the menu and why it's shorter than five.
- Close with: *"Reply with the code(s) you want me to produce — `VE`, `QA`, `POV`, `CS`, or `MB` (pick one or several). I'll then research the visuals, drop in real images, and walk you through producing it."*

Before sending, run the **7-test self-check** from `SKILL.md` on each script — proud · spoken · safe · emotion · owner · budget · peer — and revise once wherever a test fails.

Do NOT call `widecast_account`, estimate credits, ask the production question, or call `widecast_create_video` yet — all of that is Stage 2, after the pick. Writing **at least one** script is mandatory; dropping a format is fine **only with a fit-based reason** — skipping for effort, or returning zero scripts, is an invalid Stage-1 hand-off.

### 4b — Presenting the scripts: interactive HTML artifact (MANDATORY when your host supports it)

**STEP 4b.0 — declare your presentation mode FIRST, out loud, before you write the hand-off.** One literal line in the chat:

> `Presentation: HTML-ARTIFACT` — my host renders interactive HTML artifacts, so I will present the versions as an editable page with Copy buttons.

…or, only if your host has no interactive-artifact surface:

> `Presentation: MARKDOWN` — my host (CLI / terminal / plain chat) can't render an interactive artifact, so I'll use coded markdown headings.

**If your host CAN render an interactive HTML artifact (Claude on claude.ai / Desktop, or any surface that runs an HTML/JS canvas), you MUST choose HTML-ARTIFACT.** Dumping the versions as plain markdown when you could have built the artifact is the failure to avoid — the editable cards + Copy buttons are the whole point (the user edits inline and copies the final text straight back). It is NOT optional, NOT "nice to have", and "markdown is faster" is NOT a reason to skip it. MARKDOWN mode is only correct for hosts that genuinely cannot render an interactive artifact (most CLI/terminal agents, plain ChatGPT/Gemini/Grok chat).

In **HTML-ARTIFACT mode**, build the page from the canonical template below:

- Each version in its own **`contenteditable` div** with a **Copy button**, so the user edits inline and copies the final text back — no need to describe edits.
- Label each version **`Version N · CODE — Name`** (e.g. `Version 1 · VE — Value Explainer`, `Version 2 · QA — Client Q&A`), numbered over the formats you wrote.
- Keep `### Research` and the closing pick-invite as **normal chat markdown** (outside the artifact); the artifact holds only the editable version cards.
- Skipped formats get NO card — mention them + their one-line fit reason in chat.

**Accepting the pick (both modes):** treat any of these as a valid choice — a bare CODE (`VE`), a `Version N` / `VN`, or the **pasted final text** the user copied out of a card. If they paste edited text, that pasted text IS the approved script for Stage 2 — use it verbatim.

❌ **Anti-example (the exact mistake to avoid):** your host supports artifacts, but you print the 5 versions as a plain markdown list with no Copy buttons "to save a step." That is a skip. ✓ **Correct:** declare `Presentation: HTML-ARTIFACT`, build the editable + Copy-button page, keep Research/pick-invite as chat text.

Canonical template (inline-styled so it survives the artifact sandbox; one `<section>` per version you wrote — adjust the count, never force five):

```html
<div style="font-family:system-ui,-apple-system,Segoe UI,Roboto,sans-serif;max-width:760px;margin:0 auto;color:#1a1a2e;">
  <p style="font-size:14px;color:#555;margin:0 0 16px;">Edit any version inline, then hit <b>Copy</b> and paste the final text back into the chat — or just reply with the version code (VE / QA / POV / CS / MB).</p>

  <section style="border:1px solid #e3e3ec;border-radius:12px;margin:0 0 16px;overflow:hidden;">
    <div style="display:flex;align-items:center;justify-content:space-between;gap:12px;padding:12px 16px;background:#f5f5fb;border-bottom:1px solid #e3e3ec;">
      <strong style="font-size:15px;">Version 1 · VE — Value Explainer</strong>
      <button onclick="wcCopy(this)" style="border:0;border-radius:8px;padding:7px 16px;font-size:13px;font-weight:600;background:#6c4cff;color:#fff;cursor:pointer;">Copy</button>
    </div>
    <div contenteditable="true" spellcheck="false" style="padding:16px;font-size:15px;line-height:1.6;white-space:pre-wrap;outline:none;min-height:60px;">…Version 1 narration here…</div>
  </section>

  <!-- repeat one <section> per version you actually wrote -->
</div>
<script>
function wcCopy(btn){
  var body=btn.closest('section').querySelector('[contenteditable]');
  var txt=(body.innerText||'').trim();
  function done(){var o=btn.textContent;btn.textContent='Copied!';setTimeout(function(){btn.textContent=o;},1200);}
  function fallback(){var r=document.createRange();r.selectNodeContents(body);var s=getSelection();s.removeAllRanges();s.addRange(r);try{document.execCommand('copy');}catch(e){}s.removeAllRanges();done();}
  if(navigator.clipboard&&navigator.clipboard.writeText){navigator.clipboard.writeText(txt).then(done,fallback);}else{fallback();}
}
</script>
```

Put the clean spoken narration inside each `contenteditable` div (no inline image URLs yet — Stage 1 stays bare). Skipped formats are NOT given a card; mention them with their one-line fit reason in the chat markdown instead.

---

## Stage 2 — after the user picks: full visual treatment, then produce

When the user replies with one or more codes (`VE` / `QA` / `POV` / `CS` / `MB`), run the full treatment **for each picked script only**:

### 5 — Source + VET each inline image (the strict media discipline)

Bad, mismatched, or wall-to-wall inline images ruin the video. So Stage 2 is not "harvest a pile of URLs and inline them" — it is **vet every image before it goes in**, and keep images **sparse**. Two hard rules govern the whole script:

**Image budget (per script):**

- **At most 3 inline images.** Never more, no matter how many candidates you find.
- **At least 1 scene with an image.** Never zero.
- **No two consecutive scenes both carry an image** — spread them out, alternate. If you have more than 3 strong candidates, choose the 3 that are spaced apart.
- **Every other scene gets WideCast's automatic B-roll** — do NOT hunt video for them, and do NOT call `widecast_search_broll` (that tool is for *editing an existing* video, not for authoring a new script). Just leave those beats bare; the engine sources B-roll for them.

**STEP 5.0 — declare your vetting mode FIRST, out loud, before sourcing anything.** Write one literal line in the chat so you (and the user) commit to it:

> `Vetting mode: VISION` — I can download a file and view it, so I will look at every image before inlining.

…or, only if you genuinely cannot open a downloaded image:

> `Vetting mode: URL-AUTHORITY` — I cannot view images, so I will judge candidates by source authority.

**If you have ANY way to download a file and view it (a bash/python sandbox, a file-read or image-view tool, an attach-image capability) you MUST choose VISION.** Reading the source page's caption, alt text, or "it's from Reuters so it's probably fine" is **NOT** vetting and is **NOT** a reason to drop to URL-AUTHORITY. URL-AUTHORITY is only for hosts that truly cannot open an image. Do not let the convenience of a descriptive stock page talk you out of looking.

**VISION mode — each inline image is a 4-state machine. It is "vetted" ONLY after all four states complete, in order. Skip a state → NOT vetted → you may not inline it:**

1. **SOURCED** — find a candidate real-image URL via the **R2 ladder** in `research_visuals.md` (Wikimedia → the subject's own source → reputable secondary via `web_fetch` → stock). Do NOT use `widecast_search_broll` (edit-only).
2. **DOWNLOADED** — save the image to a local file (curl/wget/your download tool).
3. **VIEWED** — actually open and LOOK at the local file. Judge it: right subject? clear? not a logo/watermark/ad/collage/wrong thing? does it match this beat?
4. **SHOWN-LOCAL** — display the **saved LOCAL file** to the user (NOT the online URL — sandboxes can't render an `https://` URL, so pasting it shows nothing and proves nothing). Use your host's local-file display:
   - **Claude / Claude Code:** attach the saved image file in your reply.
   - **Codex / CLI agents:** `view_image` (or equivalent) on the saved path.
   - **Gemini / Grok / others:** your inline local-file / image-attachment view.
   - If — and only if — your host has no way at all to display a local file, say in one line exactly what you saw ("looked at the saved file: a wide AP photo of the flooded highway, clear, on-subject"). Never substitute the online URL.

   Only after SHOWN-LOCAL: **fits** → inline it. **Doesn't fit** → change the keyword / try another source and repeat from state 1. **Retry up to 5 times per beat.**

5. **Still nothing after 5 tries** → if you can generate images yourself, **generate** a fitting one, **upload via `widecast_upload_asset`**, use the returned `url` (view + show it the same way first). **Can't generate** → call **`widecast_create_image`** as the LAST resort — **1 credit per image**, so **tell the user** you're spending it and why. Last because real photos beat AI art for credibility — critical for news / real events / real products.

❌ **Anti-example (this is the exact mistake to avoid):** finding a Pexels/stock page, reading its caption, deciding "looks relevant", and inlining the URL — **without ever downloading or viewing the actual pixels.** That is NOT vetted. ✓ **Correct:** download → view the local file → show the local file to the user → *then* inline. If you only read text about the image, you did not vet it.

**URL-AUTHORITY mode (truly cannot view — rare):** do NOT inline a guessed URL. Judge candidates by **URL + source authority** (the authority-match test in *R3* in `research_visuals.md` — apple.com for an iPhone, AP/Reuters for a news event, a museum for an artwork) and inline only a URL from a real source (`web_fetch` body / search result / the user) that passes it. Do NOT fall back to `widecast_create_image` just because you can't see — news needs real photos; generation would fabricate reality. No credible real URL for a beat → leave it to auto-B-roll.

The image budget (≤3, ≥1, no adjacency) applies in BOTH modes.

### 6 — Offer the backup pool

After placing your vetted inline images, also offer a small **backup pool** of 2–3 extra candidates for the 2–3 most important beats, capped ~6–10 URLs, under a `### Backup image pool — unverified, your call` heading with the note: *"Pulled from open web search, NOT verified for rights. Pick any you like, or tell me which to add to the scene-editor library. You're responsible for usage rights."* The backup pool is **always offered** (WideCast's scene editor has a slot for it, so the user may want options) — but its candidates do NOT count against the 3-image budget and are NOT inlined automatically. Don't fabricate URLs.

### 7 — Estimate cost, show balance, ask the ONE production question

For the picked script: (i) **estimate credits** — 1 credit per scene (1 HOOK/opening + 1 per MAIN_POINT + 1 per KEY_TAKEAWAY + 1 CTA); express as a range (e.g. "~6–8 scenes / ~6–8 credits"). If you spent any credits on `widecast_create_image` during the step-5 vetting (1 credit per generated image), add them to the summary and say so explicitly. (ii) Call `widecast_account` ONCE to read `credits_remaining`. Then present, for each picked script, in ONE message:

- `### Visual assets` — each verified URL: which beat + the `![alt](url)` block + one-line rationale (or, for abstract topics, the 3+ visual directions + why no direct URL fits).
- `### Script with inline media` — the picked script with the inline URLs placed.
- `### Backup image pool — unverified, your call` — from step 6 (or the one-line "No image search available" notice).
- `### Production` —
  - `- Estimated cost: ~N–M credits (1 credit per scene; final count set by WideCast when scenes are generated)`
  - `- Your balance: K credits remaining`
  - `- Cost is the same whether you pick normal or faceless`
  - If `credits_remaining` < the LOW end of the range, add: `- ⚠ Your balance (K credits) is below the estimated cost — top up at widecast.ai before confirming, or I won't be able to produce this video.`
  - Then ask the user to pick ONE of THREE production options (in their language): **(1) Faceless** — B-roll only, no narrator on screen; **(2) Face clone** — their pre-trained Face + Voice clone speaks the script (must set up at https://widecast.ai/#setup first); **(3) Teleprompter** — they record themselves via WideCast's built-in teleprompter after scenes prepare. Close with: *"Want to tweak anything before I produce it?"*

### 8 — Handle the reply, then call `create_video`

- **Edits** (to the picked script's content, wording, hook, facts, URLs) → iterate that script, re-hand it (Step 7) with the same backup pool unless they ask to refresh. Don't call `create_video` yet.
- **A production answer** (`faceless` / `face_clone` / `teleprompter`) = implicit approval → call `widecast_create_video` (see `handoff.md` for the full call shape) with `source="text"`, `script_text` = the picked script with inline URLs verbatim, `media_pool=[urls they picked from the backup pool]` if any, `production_mode` = their literal choice, `script_approved=true`.
