# Jump-Prevention Rules — catch yourself BEFORE the action

Each line is an **interrupt** on what you are *about to* do. If the trigger matches, **STOP**, do the `→` part first, then resume.

Deliberately redundant with the Critical Rules (`01_critical_rules`) and the per-scene DoD (`03_dod_gates`) — **redundancy is what stops misses**. Load at the START of every run, and re-load after any detour/compaction.

---

### The big one — do NOT re-verify what the server guarantees

- About to **audit dead-zone / face-clearance / overlay placement / final composition / whether a scene needs an overlay** → STOP. The server guarantees all of it (overlays never enter dead zones, never cover the face, always fit the safe zone after auto-fit; the pipeline decides overlay existence). There is no dead-zone proof, no face check, no A-roll layout ladder, no composition-tuning gate, no overlay-existence gate. Move on.

- About to **run an A-roll layout priority ladder, or resize/reposition the narrator** (`overlay.narrator.rect`) to make room for an overlay → STOP. The narrator is fixed input; never edit `narrator_face`; the server already keeps overlays off the face.

- About to **pull a composite screenshot to "check the layout looks OK"** → STOP. Composite looks are only for confirming a fix you made (Gate 5). To judge the two blind spots, use the plate (Gate 3) and the poster (Gate 4). There is NO BEFORE composite look.

### Routing the two conditional gates

- About to **audit a background** → first confirm Gate 3 applies (NOT grid AND narrator not filling the frame — from data). If it applies, load `20_background`, pull the active plate, print the Gate 3 BACKGROUND PROOF (semantic/geo/context). Grid or full-frame A-roll narrator → mark Gate 3 N/A, no look.

- About to **typo-check overlay text** → first confirm Gate 4 applies (text was image-model-generated: `illustration` sub_mode≠`photo_with_people`, or chart/diagram/object with baked text). If it applies, pull the overlay poster and print the per-string transcription table. **`typography_only`/SVG text never misspells → mark Gate 4 N/A, no look.**

- About to **write "typo PASS" / "diacritics correct" for a scene where Gate 4 applies WITHOUT its per-string transcription table printed in THIS scene's block** → STOP. One table per applicable scene, every time — "the last few were clean" is proof-compression drift. A prose assurance is not the table; scene 2's table does not cover scene 9.

- About to **change the background because the overlay looks weak**, or **change the overlay because the background is wrong** → STOP. Layer isolation: a background fix touches only `mediaUrl`; it never authorizes an overlay change, and vice-versa.

### Evidence / looking

- About to **act on an image you have not SHOWN locally** → produce the one user-visible render first (Rule 0). Never a private `Read` counted as "shown"; never a double (view + markdown tag).

- About to **make a call from `scene_geometry` alone** for a fix that needs sight → pull the relevant look (plate or poster) and judge from the image.

### Autonomy / scope

- About to **ask what kind of edit / edit scope** after a WideCast URL or `topic_id` (`What kind of edit?`, `Full audit + fix / Specific scenes / …`, `Before I touch anything…`, `request_user_input`) → STOP. Scope is already **Full audit + fix**. Continue from `video_data` → context pass → scene 2.

- About to **ask the user to choose/approve an option during runtime** → STOP. Use the decision protocol (Rule 2a), choose, act, report the decision.

### Verdict / roster / handoff

- About to **start a scene** → load `10_mechanics`. INLINE: if it is not the next unvisited SCENE ROSTER row → STOP, work the roster in order. DELEGATION: rows close in event order, but no row may stay open.

- About to **move to the next scene** with no stated `PASS`/`FAIL` → declare the verdict first.

- About to **declare `Scene N: PASS`** without scanning the 5 gates + §7 → run the scan; PASS is earned by the scan. If Gate 3 applied, name its verdict; if Gate 4 applied, the per-string table is present; if you edited, "AFTER shown: yes".

- About to **declare `Scene N: PASS` from a batch/gallery/table/script/bulk-API result** → STOP. That is triage only. The scene needs its own applicable-gate evidence.

- About to **run a whole-video QA pass at the end** → STOP. There is none; each scene is done at its own PASS.

- About to **write a final summary / completion message / export question** → run the Pre-summary completion scan against the run_ledger roster first. Any unmet row → do that work first.

- About to **ask for render/export or call `export_video`** while any content scene lacks its own `Scene N: PASS` → STOP. Status is `partial_triage_only`/`partial_fix_only`; do not export.

### Modules / resume

- A **module read errored / truncated / returned only a preview** → STOP. NOT loaded; re-read to EOF before the step that needs it.

- **The context was just compacted** → STOP. RESUME, not continuation: `Read` the run_ledger, treat pre-compaction loads as VOID, re-load the CURRENT scene's modules, reprint its SCENE LOAD LEDGER + 5-gate plan, continue from the earliest unproven gate. No self-invented "short checklist".

- **Resuming / continuing a run** → `Read` the run_ledger FIRST (roster, verdicts, phase), then re-load the modules for the step you are on. A summary is not the ledger.

### Subagents (delegation mode, >30 scenes or user asked)

- About to **spawn a subagent** → load `06_subagent_protocol`, ensure the edit session is started and the skill is unzipped locally. Fixed template (fill blanks only, pass `skill_root` for local reads); editor prints its OWN LOAD LEDGER; name = `Scene <id> editor agent`; rolling K=5.

- About to **accept a subagent report** → validate: LOAD LEDGER PASS, write scope respected (only its own `voice_file`), listed evidence files exist (`ls`, not eyes), report block complete with explicit `Scene <id>: PASS|FAIL`. Any miss → re-spawn (2 structural failures → inline takeover).

- Main agent **about to view a scene image / call `scene_inspector`** in delegation mode → STOP (NO-RELOOK). Editors look; the main agent only forwards saved evidence files. Exception: a scene escalated to inline takeover.

- A **subagent report shows a write OUTSIDE its scene scope** (foreign `voice_file`/export/publish/voice upload) → STOP: INVALID. Re-pull `video_data`, mark it dirty, re-verify. Own-scene `modify_scene` writes on the edit session are the NORMAL path.

- About to **hand off while the edit session is still open** → STOP. Run the completion scan, then `widecast_edit_session action='commit'`. Never commit while an editor is still running.

- About to call **`modify_scene` / `upload_asset` / `export_video`** → confirm immediately above the write: the KICKOFF/COORDINATOR LOAD LEDGER is printed, and this scene's applicable gate proofs are printed. Missing → print first (announce ≠ pause).
