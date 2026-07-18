# 06 · Subagent Protocol v3 — parallel scene editors on a server edit session

Load this module **BEFORE spawning any subagent** for scene work, and re-load it when resuming a run that uses subagents.

## When delegation applies — inline is the default up to 30 content scenes

**Mode selection is a THRESHOLD, not a preference:** videos with **≤30 content scenes run INLINE by default** (measured: on a 7-scene video, inline was both faster and ~2.5× cheaper than fan-out, because the per-editor cold-start cost dominates on short videos). The scene-editor pipeline below is the standard mode only when the video has **more than 30 content scenes** OR the user explicitly asks for parallel/multi-agent processing. When it applies, consent is already granted by the WideCast edit trigger (MCP server instructions rule 5) — do not wait for the word "subagent". Even above the threshold, fall back to inline for: (a) no subagent capability, (d) host HARD-BLOCKS spawning (report it at hand-off). Record `delegation mode: subagent (K=<n>)` or `inline — reason: <≤30 scenes default | a | user asked single-agent | d>` in the run_ledger. The whole mechanism stays maintained and ready — the threshold is one number, not a removal.

## Why this shape is safe — the server does the hard part

- **Per-video write lock (always on):** the server serializes every `modify_scene` on the same `topic_id` — concurrent writes queue, none clobber.
- **Edit session (`widecast_edit_session`):** `action='start'` caches the whole video document in server memory (crash-safe via write-ahead file). All writes stage in the cache; ALL reads — `video_data`, `scene_geometry`, `scene_inspector` screenshots, `overlay_poster` — are served fresh from the cache (read-after-write is instant), and permanent storage is written ONCE at `action='commit'`. Idle sessions auto-commit after 45 minutes.
- Therefore **each scene editor can write its own scene directly** — no payload files, no courier, no single-writer queue in the agent layer.
- If `edit_session` returns `cache_enabled=false` (server kill-switch), proceed WITHOUT a session and note it in the run_ledger — the per-video lock still protects parallel writes.

## Roles

- **MAIN AGENT — pure coordinator, ZERO writes to scenes, ZERO images.** Opens/commits the edit session, prepares the environment (data + skill files), spawns/validates/records, escalates, hands off. It never calls `modify_scene`, never views a screenshot (NO-RELOOK below).
- **SCENE EDITOR** (one per content scene) — the whole per-scene job in one warm context: the 5 gates for ITS scene, writing directly via `modify_scene` **scoped to its own `voice_file` only**. It is both the fixer and the verifier of its own work (route → plate look if Gate 3 applies → poster typo table if Gate 4 applies → edit → AFTER look if edited → verdict).

There is **no video-QA agent** — each scene is complete at its own PASS; nothing is re-reviewed as a batch at the end.

## The flow

- **Phase 0 — kickoff (main, once, SLIM).** The coordinator never edits a scene, so it loads only the COORDINATOR SET: `SKILL.md` + `01_critical_rules` + `02_jump_prevention` + `04_principles_workflow` + `06_subagent_protocol`. It does NOT load `03_dod_gates`, `05`, `10`, `20`, `30`, `31`, `32`, `33` — those are scene-work modules; every editor loads its own and proves it in its report (that is where module coverage lives in delegation mode). Print the COORDINATOR LOAD LEDGER (below). Then: pull `video_data` ONCE; print SCENE ROSTER; write run_ledger; export steward files (`run_script.txt`, per-scene `record.json`, full snapshot); ensure the skill zip is **already unzipped locally** and note its root path; call `widecast_edit_session action='start'`.

```text
COORDINATOR LOAD LEDGER (delegation mode):
☑ SKILL.md               lines=<N> manifest=<M>
☑ 01_critical_rules      lines=<N> manifest=<M>
☑ 02_jump_prevention     lines=<N> manifest=<M>
☑ 04_principles_workflow lines=<N> manifest=<M>
☑ 06_subagent_protocol   lines=<N> manifest=<M>
Scene-work modules (03/05/10/20/30/31/32/33): loaded by each scene editor — proven per-editor in its SCENE LOAD LEDGER.
Verdict: <PASS | BLOCKED — re-read <module> to EOF>
```

- **Phase 1 — spawn scene editors.** One editor per content scene, **rolling pool of K=5**: keep 5 in flight, top up the moment one finishes (never fixed batches). If the host's limit is lower than K, run at the host's max and top up on every freed slot; retry refused spawns as slots open. (There are no "endpoint" scenes anymore — every scene takes the same routing, so ordering is just play order.)
- **Phase 2 — collect (main, event-driven).** As each editor reports: validate mechanically (`ls` checks, report block complete, write scope respected) → record verdict in run_ledger. No waiting on scene order; rows close in arrival order.
- **Phase 3 — close.** When all roster rows are PASS: pre-summary completion scan against the roster → `widecast_edit_session action='commit'` (MANDATORY — staged edits are not live until commit; never commit while any editor is still running) → hand-off + notification.

## Scene-editor contract (the fixed prompt template)

Fill ONLY the `<...>` slots — never paraphrase skill rules into the prompt:

```text
You are "Scene <id> editor agent" for a WideCast video. You own EXACTLY ONE scene.
WRITE SCOPE (hard): you may call widecast_modify_scene ONLY with by=voice_file,
value=<voice_file>. Writing any other scene, export, publish, voice/narrator upload,
or platform tools = your report is INVALID. widecast_upload_asset (S3) is allowed.
An edit session is open on the server: your writes stage safely; reads are always fresh.

FIRST ACTION — load the skill from LOCAL disk (already unzipped; do NOT download
anything): Read <skill_root>/SKILL.md, then: 01_critical_rules, 02_jump_prevention,
03_dod_gates, 05_quality_qa_priority, 10_mechanics. Load 20_background only if Gate 3
applies to your scene (non-grid AND narrator not filling the frame). Load
30_overlay_core (+31/32/33) ONLY if you must fix an image-gen overlay defect.
Print your SCENE LOAD LEDGER against <skill_root>/LOAD_MANIFEST.md. Only if a local
file is missing may you fall back to widecast_get_editing_skill — say so in the report.
SECOND ACTION: Read your record.json and run_script.txt (whole-video context: topic,
terminology, geo, numbers) — you edit one scene but judge it against the whole video.
Do NOT call widecast_video_data unless a file is missing/unreadable.

topic_id: <...>
scene: voice_file=<...>  id=<...>  type=<...>  opening=<yes|no>
scene record: <scratchpad>/scene_<voice_file>/record.json
run script: <scratchpad>/run_script.txt
skill_root: <local path to unzipped skill>
run digest: topic=<...>; tone=<...>; glossary=<...>
run_ledger (READ-ONLY): <path>     your file dir (write ONLY here): <scratchpad>/scene_<voice_file>/

JOB: run the 5-gate playbook for this scene. Gate 1 fix text/STT (branch K). Gate 2
route from data. Gate 3 (if non-grid + narrator not filling frame): pull the plate,
print the Gate 3 BACKGROUND PROOF, fix via mediaUrl only. Gate 4 (if overlay text is
image-model-generated — illustration non-photo / chart / diagram / object): pull the
overlay poster, print the per-string transcription table (transcribe FIRST). If
opening=yes: ALWAYS pull the poster (even typography) and run the OPENING POSTER CHECK
— rebuild the hook poster if it clearly falls short (cap 1 rebuild, preserve-biased),
or author one if there's no overlay. Gate 5:
if you edited, the ONE AFTER look (poster/composite) — it verifies the fix AND confirms
the save (a modify_scene 200 under the edit session is durable; no separate re-pull).
Do NOT audit dead-zone /
face / placement / composition — the server guarantees those. Report: SCENE LOAD
LEDGER, gate verdicts (with N/A + reason where a gate didn't apply), files list, and
final `Scene <id>: PASS|FAIL — reasons`. Decide autonomously; ask no questions.
```

**Naming:** every spawn is labeled exactly `Scene <id> editor agent` / `Scene <id> fix agent (cycle <c>)`, and the same label is used in run_ledger rows and progress messages. Generic names are a process error.

## Main-agent rules

- **NO-RELOOK (hard):** the main agent never downloads/opens/judges any scene image and never calls `scene_inspector`. Editors look; the main agent forwards editors' saved evidence files to the user when recording verdicts (sending ≠ analyzing). Sole exception: a scene escalated to inline (below).
- **Report validity gates:** LOAD LEDGER printed with PASS · write scope respected (no foreign-scene writes) · listed files exist on disk (`ls`, not eyes) · report block complete with explicit verdict. Any miss → re-spawn that scene once with a note; a second structural failure → **inline takeover** of that one scene by the main agent (classic flow, normal look rules apply), rest of the run unaffected.
- **Fix cycles:** an editor that ends `FAIL` on its own scene has already tried; main spawns `Scene <id> fix agent (cycle 1)` with the editor's report paths. Cap 2 cycles → inline takeover. Fix agents inherit the same contract (self-scoped write, local skill, own dir).
- **Steward files are BEFORE-state reference.** Editors work against live server state through their own reads (instant in the session); the main agent does not refresh per-write. Refresh the full snapshot once after commit for the record.
- **run_ledger:** main-agent-only writes; update on every spawn/report/verdict/escalation; re-`Read` after any resume/compaction. Roster = completeness tracker (rows close in event order; every row must close).
- **File namespacing:** every editor writes only inside `<scratchpad>/scene_<voice_file>/`; S3 keys are per-call UUIDs; server per-scene artifacts are keyed by `voice_file` — no collision surfaces.
- **One video at a time.** Commit the session before starting another video's run.
