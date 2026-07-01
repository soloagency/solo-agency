# Hand-off to WideCast — A · B · C

Load this module when you reach hand-off. It carries the full 3-step hand-off (script pick → visuals + production → `create_video` call) plus the post-call observability rules.

**Order is fixed: five scripts first, then a pick, then visuals + production, then create.** Don't ask about faceless or call `create_video` before the user has picked a code and seen the visual hand-off.

---

## A · Stage 1 — hand off the scripts, ask for a pick

Show `### Research` + one block per format you wrote (each under its coded heading; skipped formats get a one-line fit reason instead), then invite a pick by code (`VE` / `QA` / `POV` / `CS` / `MB`, one or several). When the host supports interactive HTML artifacts, present those blocks as the editable artifact from `method.md` step 4b (each version `contenteditable` + Copy, labeled `Version N · CODE — Name`) and keep Research + the pick-invite as chat markdown. No production question yet.

## B · Stage 2 — for each picked script, hand off visuals + the production question

Show the picked script with inline image URLs + `### Visual assets` + `### Backup image pool` + `### Production` (credit estimate + balance + the three-option production question, in the user's language):

> "Three ways I can produce this:
>
> 1. **Faceless** — B-roll only, no narrator on screen. Nothing else for you to do.
> 2. **Face clone** — your trained Face clone + Voice clone speaks the script (set up at https://widecast.ai/#setup if you haven't yet).
> 3. **Teleprompter** — you record yourself reading the script via the built-in teleprompter, once the scenes are ready.
>
> Which one?"

Close with *"Want to tweak the script first, or shall I produce it?"* If they pick **face clone** and likely haven't set up the clone, add: *"You'll want your Face + Voice clones trained at https://widecast.ai/#setup before the scenes finish — takes ~3 min."*

## C · Call `create_video`

> **⚠ MCP / ChatGPT-Action callers — required confirmation flags**
>
> `widecast_create_video` requires TWO flags (the REST API stays free of them — SDK / curl callers are unaffected):
>
> - `script_approved: true` — set ONLY after Stage 2 Step B (the user saw the picked script with inline URLs + the production sections) AND picked a production option. A generic "make a video about X" is NOT approval.
> - `production_mode: "faceless" | "face_clone" | "teleprompter"` — the user's EXPLICIT pick. Do NOT infer from a prior video. Ask each time.
>
> The tool rejects with a clear error if either is missing or false. Don't bypass with placeholders — go fix the dialog flow.

- **Picked script →** `source="text"`, `script_text=<the picked script, inline URLs VERBATIM>`, `script_approved=true`, `production_mode=<their pick>`, `media_pool=[urls they picked]` if any. (Legacy SDK / HTTP: keep `faceless=true|false` — the MCP wrapper maps `face_clone`/`teleprompter` → `faceless=false`, `faceless` → `faceless=true`.)
- **Only a topic, no script yet →** `source="idea"`, `idea_text=<tight 1–3 sentence brief>`, plus `language`, `video_length` ("short"/"normal").
- **User attached / linked audio →** if public URL, use it; if bytes in chat, FIRST call `widecast_upload_asset` (24-hour S3 TTL), then `create_video(source="audio_url", audio_url=<that URL>, …)`. The audio IS the script (WideCast transcribes it) so you SKIP the five-format writing flow — but you still owe the production question (`production_mode` required; same three options).

`output_type` is left at the default — `create_video` always produces reviewable scenes; the user renders the final MP4 from the WideCast UI.

Then poll `wait_for_video` (or `get_status` no faster than **every 5 seconds**) until `completed`. The status response carries `progress_hint.label` — a human-readable English sub-stage with ETA (e.g. `"Generating scene visuals · ~7 min left"`). **Relay this every poll**, translated to the user's language. It's pseudo-progress (time-based) — display only, don't gate logic on it.

**Show the result INLINE** when done: embed `embed_url` in an HTML artifact `<iframe>` so the user can watch in chat, and offer `review_url` as the "open / edit in WideCast" link. If the host won't render the iframe, show `review_url` as a clickable button.

**Built-in web viewer / browser tool** — if your runtime exposes one (Codex view-url, ChatGPT browse tool, any host that can iframe an external URL, future inline browsers), **open `review_url` in it as soon as you surface the link**, not as an afterthought. The review page is a full scene editor; loading it inside the chat session is dramatically better UX than asking the user to copy-paste a link into another tab. This applies after every call that returns `review_url`: `create_video`, `wait_for_video`, `get_status`, `modify_scene` (re-open to show the swap), and `export_video` (open after the final MP4 lands).

**If the create call returns HTTP 402** (`error.code` is `credit_exhausted` or `account_expired`), `error.details` carries a structured upgrade/wait block. Surface BOTH options:

1. **Wait** until `details.reset_at` (next monthly quota refresh).
2. **Upgrade now** to `details.next_plan` (`details.next_plan_quota` credits/month) at `details.upgrade_url` — `https://widecast.ai/#pricing_plans`.

For `account_expired`, use `details.expired_at` + `details.renew_url`.

---

## Deep references

- `hooks.md` — hook playbook + 12 templates (Stage 1 step 3 for VE format)
- `ctas.md` — CTA banks (Stage 1 step 3 for VE/CS/MB formats)
- `method.md` — full Stage 1 + Stage 2 methodology
- `formats.md` — VE/QA/POV/CS/MB per-format playbooks
- `research_visuals.md` — R-ladder for image sourcing + inline media format rules
