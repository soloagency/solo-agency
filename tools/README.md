# OutreachCRM tools

Stdlib-only Python (no `pip install` needed for the @gmail.com App Password path).

| Tool | Purpose |
|---|---|
| `storage/` | pluggable storage adapter (`json_adapter` default; Postgres later). All CRM I/O goes through it. |
| `crm_store.py` | the ONLY sanctioned writer of `crm/` collections: contacts, deals, activities, tasks, pipelines, suppression + the deterministic rules engine. Also renders the operator surfaces (`approval-report`, `today-view`, `kanban`) and the client-facing `weekly-report` + `monthly-report` (scrub-gated), and reports the daily drafting budget (`draft budget --campaign X`). |
| `import_leads.py` | CSV/TXT/XLSX → contacts (mapping, dedupe, suppression, idempotent). |
| `email_verify.py` | cheap syntax + MX check (no SMTP probe). |
| `gmail_client.py` | @gmail.com App Password send/sync (SMTP/IMAP): auth, health, quota, send (with the pre-send gate chain), sync (with the DSN-first classifier). |
| `report_renderer.py` | standalone HTML/PDF report renderer + Client-Blind Scrub Gate. |
| `provider_openapi.py` | WideCast notification provider adapter (Telegram + report-link upload). `notify` composes verify→upload→send→log in one step, with `--dry-run` and a non-fatal `local_path_only` degrade when no provider is configured. |

Tests: `python3 -m unittest discover -s tests` (27 tests, offline).

Advanced OAuth/Workspace sending + the Cloudflare tracker (open/click) are Phase 2+.
