# Solo Agency Bridge + Local UI — Design Contract (U-spec v1)

Status: U0 approved by the operator (2026-07-19). **U1 shipped** (read-only UI + G1 tool CLI). **U2 shipped 2026-07-19** (interactive approvals + shortlist via `ui_inbox`, consumed by `crm_store.py ingest-ui`; playbook wiring in AUTOMATION_SCHEDULING/06/09/DESIGN §14–15 and content playbook 02). **G2+G3 shipped 2026-07-19**: ALL six Python tools now live inside the bridge binary (`tool crm-store|import-leads|render-report|provider|gmail|verify-email`) with golden cross-validation against the Python originals (crm-store: 32-step scenario, stdout+tree parity incl. `ingest-ui`; import-leads: byte-identical idempotency keys; render-report: byte-identical HTML + scrub-gate parity; provider: stub-server parity incl. the composed notify; gmail: classifier/presend/dry-run/quota parity; **live SMTP/IMAP VALIDATED 2026-07-19 on a real @gmail.com sendbox** — Go auth (SMTP 465 + hand-rolled IMAP 993), Python health on Go-written credentials, Go draft → Python approve → Go real send → Python quota parity → Go sync fetched + classified the arriving mail as campaign_reply/from_address and froze the sequence; Python re-sync on the advanced cursor saw 0 new. The retirement precondition (parity + one real run) was met and **the retirement sweep executed 2026-07-19 on the operator's order: every `.py` tool and the Python test suite are deleted; the bridge binary is the only implementation; the golden xval tests became pure-Go behavioral scenarios asserting the Python-verified outcomes; the deploy preflight runs `go test`; Stage-0 Binary-First is binary-only (no fallback)**. Same slice added the lead-data taxonomy (DESIGN §9.1b): `identities.seeds[]` content clues (reel/video/post/blog) as dedupe-eligible records, URL-shape classification on import incl. per-line .txt fragments and multi-column clue mapping, the seed → profile → email resolution ladder with `seed_unresolved`/`name_only_fragment` enrich reasons, and canonical write-back of found profiles via `channels_found.profiles`). Windows is unlocked (flock/LockFileEx replaces fcntl). Playbooks now carry the Binary-First Tool Invocation rule (Stage 0 both sides). **U3a shipped 2026-07-19** (CRM list columns + full contact profile page with the hooks timeline). **U3-campaigns + autostart shipped 2026-07-20**: campaigns list + detail/edit pages with the operator whitelist (`campaignUpdate` in the store, `tool crm-store campaign update` on the CLI, `POST /api/ui/{c}/campaign-update` on the UI — the second sanctioned direct write, see §6), code-enforced paused semantics (queue no-ops with `skipped.campaign_paused`, draft write refuses, send parks a transient `campaign_paused` blocker that keeps the draft approved), and boot autostart in the setup scripts (macOS LaunchAgent `com.solo-agency.collector.{insthash}` with KeepAlive-on-failure, Linux systemd user unit + best-effort lingering, Windows logon Scheduled Task wrapped in `cmd /c` for logging; `SOLO_AGENCY_NO_AUTOSTART=1` opts out, clean `/shutdown` exits 0 so supervisors do not fight upgrades). **UI redesign shipped 2026-07-20 (v1.6, presentation-only):** dark-only design system (`app.css` vendored in-binary at `/ui/assets/app.css`, served no-cache so upgrades restyle instantly; Pico stays as the form/reset layer, forced `data-theme=dark`) with a blue-black canvas, layered surfaces and ONE lime accent; sidebar shell (global nav + per-client section with a live pending-approvals badge) + topbar breadcrumb + SSE "live" dot; home = hero tagline + cross-client stat tiles (clients / active campaigns / pending drafts / sent today) + a rich grouped feature catalog (Content pipeline, Outreach + CRM, Runs itself) with vendored Tabler icons and web-UI vs agent-chat chips; client overview = clickable stat tiles + icon action cards; campaigns get quota progress bars, approvals get band-colored draft cards under the sticky batch toolbar, extension gets a numbered stepper, CRM/contact/status/sendboxes/reports/jobs/locked restyled to the same tokens. Zero behavior change: no route/API/form-field/id was touched, every test-asserted string preserved, SSE/footform semantics intact; em-dash-free UI copy. This document is the authoritative contract for the localhost UI and for absorbing the Python tools into the single bridge binary. Implementation phases below reference it; when any implementation disagrees with this file, this file wins (amend it first, then the code). (v1.6)

## 1. Purpose and principles

Solo Agency is a new kind of software: an AI agent (any vendor — Claude, Codex, Hermes, OpenClaw, ...) is the orchestrator and tour guide, chat is the conversational spine, and a localhost web UI covers everything chat can never do well (tables, buttons, live status, batch review). Principles, in priority order:

1. **The filesystem is the bus.** Every agent's I/O is files under `daily-content-pipeline/`. No agent ever depends on HTTP to the bridge: agents in sandboxes often cannot reach localhost (see the Sandbox Localhost Rule). The UI and the agents meet only at files.
2. **Agents produce links, never fetch them.** An agent constructs UI URLs as static strings from the URL map below and hands them to the human. The human's browser (on the human's own machine) opens them. An agent must never HTTP-GET a UI URL, and a UI URL being unreachable from a sandbox proves nothing.
3. **One process.** The existing collector bridge (`solo-agency-collector/bridge-go`, `127.0.0.1:17321`, Go stdlib only, single binary) grows four roles; nothing else runs. No second daemon, no node server, no build chain.
4. **UI complements chat; chat stays sufficient.** Every UI surface has a chat fallback (today's flows). If the bridge is down, nothing is blocked — the UI is an enhancement, never a dependency. Natural-language, unstructured input stays in chat; structured review/action moves to UI.
5. **No new trust surface.** Nothing auto-sends. A button click is the same human approval as a chat `approve` — one more way for the human to decide, never a way for the system to decide.

## 2. Where the UI fits the session model

The three session types are unchanged (see the OutreachCRM router "Session Model"): one shared setup session; one content automation session per client; one automation session per campaign. The UI is not a session — it is a window over the shared filesystem state that all sessions read/write. The chat agent guides the human into UI rooms via links; UI decisions flow back to sessions via files.

## 3. Process architecture — four roles in one binary

| Role | Description | Status |
|---|---|---|
| Extension hub | Existing 13 endpoints (`/status`, `/jobs/*`, `/collect/*`, `/complete`, ...), extension token auth. UNCHANGED; UI work must not alter these routes or their semantics. | shipped |
| Static/report server | Serve report HTML/PDF/assets from the data root read-only (`/files/...`), so handoffs become clickable URLs instead of file paths. | U1 |
| UI app | Embedded server-rendered pages + vanilla JS (Go templates compiled into the binary), served under `/ui/...`. No node, no build chain, no external CDN (self-contained like the reports). SSE-driven auto-refresh. (v1.1: server-rendered chosen over an SPA — same no-build guarantee, less client state. v1.3: styling is vendored Pico CSS v2 (MIT), embedded in-binary and served at `/ui/assets/pico.min.css` — a real design system with automatic dark mode, still zero CDN/build.) | U1–U3 |
| File-bus API + tools | Read APIs over existing files, SSE change feed, write endpoints restricted to `ui_inbox/`, and CLI subcommands replacing the Python tools (G1–G3). | U1–G3 |

## 4. Security

- Bind `127.0.0.1` only (already true). Never expose beyond localhost.
- **UI token**: on start the bridge writes `daily-content-pipeline/bridge/ui_token` (0600) containing a random token, and prints/serves the tokenized entry URL. First browser open at `/ui/enter/{token}` sets a session cookie; every mutating request requires it (CSRF defense against other local pages). Read-only GETs of client-blind-safe surfaces may be tokenless; anything operator-only requires the cookie.
- The extension token mechanism is separate and unchanged.
- Agents never need the UI token (they never fetch UI URLs). The entry URL with token is given to the human by the agent reading the token file — or the human bookmarks it once.
- Everything the UI shows is operator-only by definition (it runs on the operator's machine). The client-blind rule still applies to anything exported OUT of the UI (the scrubbed weekly/monthly reports remain the only client-facing artifacts).

## 5. URL map (stable contract — agents build links from this table)

Base: `http://127.0.0.1:17321`

| Path | Surface | Phase |
|---|---|---|
| `/ui` | Home = Feature Catalog rendered clickable + client list + global status (v1.4: capability overview on home; per-client Action cards — `ui`-kind opens the page, `agent`-kind shows the exact FEATURE_CATALOG trigger phrase + copy button + which session to paste it into) | U1 |
| `/ui/jobs` | Job queue & run monitor (collector jobs, automation runs, live states) | U1 |
| `/ui/status` | Bridge / extensions / sendboxes / provider health | U1 |
| `/ui/{client_slug}` | Client home: latest reports, pending actions, quick links | U1 |
| `/ui/{client_slug}/reports` | Reports hub (dated + `latest/`, HTML/PDF) | U1 |
| `/ui/{client_slug}/leads` | Lead & Competitor Opportunities table (links + copy-ready comments) | U1 |
| `/ui/{client_slug}/crm` | CRM kanban + contacts table (rows clickable; short id for nameless leads; phone/social columns + enriched / seed-unresolved badges) | U1 (read) / U3 (interact) |
| `/ui/{client_slug}/contact/{id}` | Full contact profile: all identities + seeds (with resolve state), the enrichment HOOKS timeline (the proof-of-life 'latest activities' that personalize email, each with evidence link + observed date + used-in), writing brief, and the activity timeline. A consolidated fragment's old id 302-redirects to its survivor (activities of merged fragments included); a `duplicate_suspects[]` record shows a resolve banner (merge vs unsuspect) | U3 |
| `/ui/{client_slug}/campaigns` | Campaign list: status pill (active/⏸ paused), goal type + objective, today's draft budget, pending-approval count, sent total + last-sent date; cards click through to the detail page | U3 |
| `/ui/{client_slug}/campaign/{campaign_slug}` | Campaign detail + operator edit form (`POST /api/ui/{c}/campaign-update`): prominent Pause/Resume, Goal section (goal_type select, objective, offer, value proposition, proof points one-per-line, CTA text), Companion-link section (plain-language instructions "how to get the link for each lead", on_fail select, default link), Sending section (daily draft budget). Same whitelist as `tool crm-store campaign update`; paused campaigns neither queue, draft, nor send (send blocker `campaign_paused` is transient — the draft stays approved and goes out on resume) | U3 |
| `/ui/{client_slug}/approvals` | Interactive Approval Report (all campaigns; per-campaign at `/approvals/{campaign_slug}`). Per-draft approve/edit/hold/reject, PLUS a batch lane: every draft has a pre-checked checkbox, a sticky check-all/uncheck-all header, a "select high-confidence only" shortcut, and "Approve checked (N)" that approves exactly the ticked drafts (keeping any inline edits) — each decision still lands in `ui_inbox/approval_decisions.jsonl` | U2 |
| `/ui/{client_slug}/shortlist` | Discovery shortlist review (checkbox table) | U2 |
| `/ui/{client_slug}/sendboxes` | Sendbox list + App Password connect form (`POST /api/ui/{c}/sendbox-auth`) | U2.5 |
| `/ui/{client_slug}/extension` | Drag-drop Chrome-extension install: reveal the folder (`POST /api/ui/{c}/reveal-extension` opens it in the OS file manager) + live check-in state | U2.7 |
| `/files/...` | Raw file serving from the data root (reports, assets), read-only | U1 |
| `/events?scope=...` | SSE change feed (file-watch driven) | U1 |
| `/api/ui/...` | JSON read APIs + `ui_inbox` write endpoints (cookie-gated) | U1–U2 |

Rules: paths are stable once shipped (append, never repurpose). Agents hand the human absolute URLs. When the bridge is not running, agents fall back to today's chat/file-path flows and may offer the one-line command to start the bridge.

## 6. File-bus contract

### 6.1 Read model (bridge → UI)

The bridge reads existing files as-is — no schema changes: `clients_index.md`, `outputs/**/report_state.json` + report HTML/PDF, `collector/jobs/*` + `collector/inbox/*`, `outreach/**/crm/*` (contacts/deals/tasks/pipelines via the same JSON the tools write), `sendboxes/sendboxes.json` (never `credentials.json`), `campaigns/*/campaign_config.json`, `outbox/*` (the Approvals page scans `campaigns/*/outbox/pending_approval/**/*.json` and shows only `status: pending_approval`), `approvals/approval_log.md`, `notifications/notification_log.md`, `automation/*`, and `history/discovery_shortlist.json` — the machine-readable shortlist mirror the CONTENT agent writes alongside the in-chat numbered list (playbook 02): `{"generated_at": "<ISO-8601 UTC>", "candidates": [{"n": <chat number>, "source_name", "source_url", "platform", "cadence_suggested": "daily|weekly|optional", "why", "classification"}]}`. The bridge never reads: `credentials.json`, `token.json`, `secrets/`, provider keys.

### 6.2 Write model (UI → agents): `ui_inbox/` only

The bridge writes ONLY append-only JSONL under dedicated `ui_inbox/` directories (single-line `O_APPEND` + fsync — safe because the ownership matrix makes the bridge the file's sole writer; v1.2 supersedes the earlier temp+rename wording, which cannot append):

- `clients/{c}/{bl}/outreach/ui_inbox/approval_decisions.jsonl` — `{ts, draft_id, decision: approve|reject|hold|edit, campaign?, edited_subject?, edited_body?, note?, ui_session}`. `edited_*` may accompany ANY decision (applied before it); `decision: edit` alone patches the draft and leaves it pending. POST `/api/ui/{client}/approval`.
- `clients/{c}/{bl}/ui_inbox/shortlist_decisions.jsonl` — `{ts, source_url, source_name?, decision: approve|skip, cadence?: daily|weekly|optional, ui_session}`. POST `/api/ui/{client}/shortlist` (body `{decisions: [...]}`, invalid entries skipped, response reports the queued count).
- (U3) `clients/{c}/{bl}/outreach/ui_inbox/crm_actions.jsonl` — task done / deal stage moves, each with evidence of the human click.

**The one sanctioned exception (v1.3, U2.5): sendbox credential entry.** `POST /api/ui/{client}/sendbox-auth` (`{slug, email, app_password}`) makes the bridge run the SAME auth code as `tool gmail auth` in-process: live SMTP+IMAP verification, then it writes `outreach/sendboxes/{slug}/credentials.json` (0600) and the `sendboxes.json` entry. This is deliberately NOT routed through `ui_inbox/` — agents read ui_inbox, and the App Password must never reach any agent-readable surface (chat, queue, log). Error responses are sanitized to a class name; the secret is never echoed. This is the only canonical write the UI can trigger outside `ui_inbox/`.

**The second sanctioned exception (v1.5, U3): operator campaign edits.** `POST /api/ui/{client}/campaign-update` (`{slug, patch}`) applies the SAME strict whitelist as `tool crm-store campaign update` (status active|paused, daily_quota 1..500, goal.{goal_type, objective, offer, value_proposition, proof_points, cta.text, companion_doc{instructions, on_fail, default_link}}; every other key is rejected loudly) and writes `campaign_config.json` atomically in-process. Rationale: campaign goal/budget/pause are OPERATOR-owned configuration — the human is the authority, so the edit takes effect instantly instead of waiting for an agent ingest cycle (pause must stop the very next send). The UI additionally appends an informational event to `outreach/ui_inbox/campaign_edits.jsonl` (`{ts, campaign, changed[], ui_session}`) so agents see that config shifted under them; it is a notification, not the write path. Everything else the UI does still goes through `ui_inbox/` only.

Consumption invariant: tools/agents merge `ui_inbox` into the canonical ledgers at the next run — `crm_store.py ingest-ui` (run at the start of every campaign daily run and again immediately before the send step) applies approval decisions with `by: ui` in `approvals/approval_log.md` (single-ledger invariant preserved; reject reasons feed `learning_log`), idempotent via the processed-line cursor `outreach/ui_inbox/.approval_cursor`; approved drafts are then honored by `gmail_client send` exactly like chat approvals. Shortlist decisions are consumed by the content agent (playbook 02, cursor `history/.shortlist_cursor`) and feed the same save path as chat-numbered approval. The bridge never writes canonical ledgers, CRM collections, configs, or profiles.

### 6.3 Ownership matrix

| Writer | May write |
|---|---|
| Agents + Python/Go tools | everything they write today (CRM via crm_store, drafts, reports, jobs, ledgers, configs) |
| Bridge | `ui_inbox/**`, `bridge/ui_token`, its existing collector inbox/status files, SSE (no disk) |
| Human (browser) | nothing directly — only via bridge `ui_inbox` endpoints |

One writer per file, append-only JSONL for decisions, atomic renames everywhere — the same discipline the storage adapter already enforces.

## 7. UI vs chat — division per surface

UI owns: approval review (cards, inline edit, approve/reject/hold, batch), shortlist review (checkboxes + evidence links), job queue/run monitor, CRM kanban/contacts, reports hub, dashboards (sendbox health, quota/warmup, metrics), Lead & Competitor tables, the clickable Feature Catalog home.

Chat owns: all natural-language declaration and correction (business context, goals, message-bank discussion), every conversational decision, tour-guide navigation ("mở [Approvals](.../ui/leadup/approvals), duyệt 3 draft đầu — hoặc trả lời `approve 1-3` ngay tại đây"), feature discovery, next-action guidance. The one-block Campaign Quick Start confirmation stays in chat (semi-structured, conversational).

Playbook rules to ship with U1/U2 (not before the surfaces exist): link-first handoffs with file-path fallback; an `[ACTION REQUIRED]` UI variant (link + what to click + chat fallback); the approval-gate amendment (chat OR ui_inbox, same trust); the agents-never-fetch rule; Stage-9 checkboxes.

## 8. Absorbing the Python tools into the bridge (G-phases)

Audit result (2026-07-19, full import/mechanism scan): 11 files, 7,739 lines, stdlib-only by design (the only non-stdlib imports are optional weasyprint/reportlab PDF fallbacks — dropped in the port). No eval/fork/signal/dynamic imports anywhere. Today the CRM layer refuses to run on Windows (`fcntl` hard requirement) — the Go port removes that blocker.

| Tier | Files | Port notes |
|---|---|---|
| G1 (with U1) | `email_verify` (Go `net.LookupMX` replaces dig/nslookup shell-outs); `storage/` read+write core (atomic write, ULID, cross-platform file lock — Flock/LockFileEx wrapper); unified report renderer (hand-rolled MD→HTML parser ports 1:1; PDF stays Chrome-headless via `os/exec`, wkhtmltopdf fallback; adopt the stricter scrub logic; merge the two ~51-line-divergent forks); unified `provider_openapi` (hand-rolled YAML line-scanner ports 1:1, no YAML lib; merge the 218-line-divergent copies, keep the outreach copy's guards + `notify` with `sendNotification` subject/message) | ~2,300 py-lines |
| G2 (with U2) | `crm_store` full port behind a CLI-compatible surface; its 3 `subprocess python3 report_renderer` call sites disappear (renderer is in-binary — also fixes the hardcoded `python3` name that breaks Windows); `import_leads` (zip+XML XLSX reader ports to `archive/zip`+`encoding/xml`; reimplement the small CSV delimiter sniffer) | ~2,750 |
| G3 | `gmail_client`: SMTP = implicit-TLS 465 via `tls.Dial` + `net/smtp` client (login + send only); IMAP = minimal client for the exact surface used — LOGIN, SELECT INBOX, UID SEARCH (ALL/range), UID FETCH RFC822, LOGOUT — no IDLE/APPEND/STORE; port the MIME build (CAN-SPAM footer, List-Unsubscribe) and the deterministic classifier with table tests; preserve the NFKC subject-gate normalization | ~800 |

CLI-compat contract: the binary exposes `solo-agency-bridge tool <name> <subcommand> [flags]` mirroring each Python CLI's flags and JSON stdout exactly. Validation gate per tool: a cross-validation harness runs Python and Go against the same fixture workspace and diffs JSON output (golden tests); a tool's Python original is retired only after parity is green plus one real daily run. During transition, playbooks say binary-first with `python3` fallback; after full parity, a sweep removes the Python paths and the user machine needs only the binary (existing dist pipeline: per-OS zips + SHA256SUMS + setup scripts).

## 9. Combined roadmap

| Phase | Ships | Risk |
|---|---|---|
| U0 | This contract | — |
| U1 + G1 | Read-only UI (home, jobs, status, reports, leads, CRM-read) + static serving + SSE + G1 tool ports + link-first playbook rules | near-zero (read-only) |
| U2 + G2 | Interactive approvals + shortlist via `ui_inbox`; crm_store/import_leads in Go; approval-gate amendment | medium — gated by cross-validation + ledger merge invariant. **U2 half shipped 2026-07-19** (UI pages + POST endpoints + Python `ingest-ui` + playbook wiring); G2 Go port still open and must carry `ingest-ui` over |
| U3 + G3 | CRM interactions, campaign controls (pause/quota), gmail port, Python retirement sweep | medium — IMAP/SMTP needs live validation on a real sendbox |

## 10. Risks and guardrails

- **Two writers**: prevented by the ownership matrix + append-only `ui_inbox` + atomic renames.
- **Collector protocol breakage**: extension routes/token untouched; UI code lives in new files; `main.go` changes limited to route mounting and subcommand dispatch. Coordinate with any concurrent collector work before editing shared files.
- **Scope creep**: vanilla JS embedded, no dependencies, each surface must keep a chat fallback; a surface that cannot state its chat fallback does not ship.
- **Approval integrity**: nothing in the UI creates an auto-send path; `manual_all` semantics unchanged; every ui_inbox decision carries `ui_session` provenance and lands in the canonical ledger before any send.
- **Windows**: G1's lock wrapper + in-binary renderer are the unlock; add Windows Chrome path candidates (`Program Files` locations) to the PDF exec candidates during the port.

## 11. Non-goals

No cloud/hosted UI, no auth beyond the localhost token, no mobile app (responsive pages suffice), no UI-side LLM calls, no bypassing of approval gates, and no second process.
