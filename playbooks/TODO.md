# OutreachCRM TODO

This TODO sits beside the detailed child playbooks as an optimization backlog. Do not treat these notes as mandatory daily questions for the human. The goal is a fully automatic daily outreach + CRM operation where the operator spends only a few minutes reviewing the Approval Report and handling replies that need judgment.

## Daily UX principle

- New layers must become automatic memory, scoring, and decision logic, not a daily questionnaire.
- The Approval Report should be approval-ready; the operator usually decides in a few minutes: approve, edit, reject, or hold.
- Nothing sends without explicit chat approval.

## Phase backlog (see docs/DESIGN.md §21 for the authoritative plan)

### Phase 1 — manual core loop
- `tools/storage/` adapter (json_adapter first) + `tools/crm_store.py` (contacts, activities).
- `tools/gmail_client.py` — app_password (SMTP/IMAP) mode first: `auth`, `send`, `sync`, `quota`, `health`.
- `tools/import_leads.py` — `inspect` + `import` (mapping, dedupe, ULID, idempotency key).
- `tools/email_verify.py` — syntax + MX check (no SMTP probe).
- Playbooks 02 (sendbox), 03 (import), 08 (send engine). Global + per-client suppression.

### Phase 2 — intelligence + automation
- Playbooks 04 (verify/enrich), 05 (campaign), 06 (email writing), 10 (follow-up/reply), 13 (CRM core), 14 (tasks/Today View).
- Skills `email-verify-enrich` and `email-writing` (SKILL.md + modules + LOAD_MANIFEST).
- Deals/tasks/rules engine (`crm_store.py apply-rules`), Approval Report, Today View.
- `tracker/worker.js` (Cloudflare, D1) for open/click/unsubscribe; `gmail_client.py track-pull`.
- Full Scheduled Run; WideCast notification. E2E runbook (client "Max Output") = acceptance gate.

### Phase 3 — CRM depth
- Playbooks 12 (tracking/analytics), 15 (reporting): kanban, contact timeline, weekly client report, forecast, segments, merge/dedupe, reactivation campaigns.
- Open/click fully wired; Postgres adapter (must pass the adapter contract test suite).

### Phase 4 — enrichment reach
- Re-import a Local Collector (from the parent architecture) to read the operator's own logged-in Facebook/LinkedIn for hooks that are unreadable logged-out.

### Phase 5 (optional)
- Local web UI reading/writing through `crm_store.py`.

## Ideas parking lot
- Owner/founder self-serve profile (infer first, ask only for corrections).
- Per-trigger and per-hook reply-rate learning to prioritize segments.
- Second sending domain support for scale (domain-level warmup already in the sendbox schema).
