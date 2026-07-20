# Agent Instructions

When the user asks to set up this module (the `outreach/` directory of the Solo Agency repo), always read `OUTREACHCRM_PLAYBOOK.md` first and follow its checklist in order. The authoritative design is `docs/DESIGN.md`; when any file disagrees with it, `docs/DESIGN.md` wins.

Human-facing required actions must use the `**[ACTION REQUIRED]**` block from `OUTREACHCRM_PLAYBOOK.md`. Do not bury questions, approvals, commands, sendbox-connection steps, provider/API-key setup, or automation task edits in paragraphs or reports. If no human action is needed, end with next-action guidance per the OutreachCRM Next-Action Guidance Rule (1-3 real available next steps plus one closing question); never end with `No action required right now.`

OutreachCRM is a local-first, multi-client cold-email + CRM system. It must run in an AI agent runtime with local workspace file access, scheduled/automation tasks, local Python execution (`tool crm-store`, `tool gmail`, `tool import-leads`, `tool verify-email`), and parallel/sub-agent work streams, such as Codex or Claude Desktop/Cowork. Do not present a plain web chat as the primary runtime; it can review outputs but cannot host the automation, file state, and mail/tool work the playbook requires.

## Two flows

During Setup Flow, never send an email, run a campaign, enrich a lead for send, or draft-and-send in the setup chat, even if the human explicitly asks. Treat the request as a handoff: create/resync the client-specific automation task and tell the human the exact task name to run. Setup Flow only configures; its terminal state is `ready_for_automation_first_run`.

Automation Flow (the scheduled daily run) is the only place operational work happens: inbox sync, reply/bounce/unsubscribe classification, CRM rule application, follow-up advising, enrichment, drafting, the Approval Report, sending approved drafts within quota, tracking pull, reporting, and operator notification.

## Storage and mutation

All CRM data lives under `daily-content-pipeline/`. Every mutation of a `crm/` collection (contacts, accounts, deals, activities, tasks, pipelines, segments, suppression — the files under `clients/{slug}/crm/`) MUST go through `tool crm-store`; writing those directly is a critical violation. `sendboxes/sendboxes.json`, `campaigns/{slug}/campaign_config.json`, `lists/`, `analytics/`, and the Client Intelligence Profile are plain config/profile files written directly per the Stage 7 schemas. Reading raw JSON is allowed only for debugging. The storage backend is pluggable (JSON now, Postgres later) via `daily-content-pipeline/storage_config.json`; do not hardcode backend assumptions in playbooks. `tool crm-store` exists (Phase 1), so the critical-violation rule for a direct `crm/` write is in force — write CRM records through it. A workspace carried over from an older Phase-0 install (records written before the tool existed) must run `<bridge> tool crm-store --client-dir <DIR> validate --rebuild-index` once to validate the records and rebuild the identity index (DESIGN §22 R3).

## Client isolation

One agency, many clients; each client is an isolated workspace under `daily-content-pipeline/clients/{client_slug}/`. The storage adapter is instantiated per client. The only agency-global collections are the global suppression tier, `secrets/`, `provider_defaults.json`, and the tracker key. A run pinned to `target_client_slug` must never read or write another client's data. Every client-specific automation task name must begin with the client name, e.g. `Max Output - SaaS Founders Intro Daily Run`.

## Sending, approval, and compliance

Nothing sends without explicit human approval given in chat. The agent drafts, renders an operator-only Approval Report, and waits for chat approval (`approve all` / `approve 1-20, 35` / `reject N: reason` / `edit N: ...` / `hold N`). Default `approval_mode` is `manual_all` for every campaign and step, including bumps and assisted channels; the agent never sends assisted-channel messages itself.

Every send goes through `tool gmail send`, which enforces, in code (do not trust prose): resolve(contact) → suppression (including a live unsubscribe pull from the tracker and the `+unsub` mailbox) → channel status → atomic quota reservation → warmup cap → two-tier box+domain cap → recipient send-window → guessed-email cap → sequence-freeze check → step-1 subject lint. Suppression is also checked at import against all of a contact's identities. Opt-out is honored immediately. Step-1 subjects must not begin `Re:`/`Fwd:`; follow-up bumps are real in-thread replies. Only an inbound reply is conversion evidence — opens and clicks never trigger an automated action or a stage change.

## Sendboxes

A client may connect multiple sendboxes. Priority path is `@gmail.com` via App Password + SMTP/IMAP (no OAuth expiry, preserves our Message-ID); the advanced path is Google Workspace via OAuth with the app set to Internal (scopes `gmail.send` + `gmail.readonly` only). Step-1 outreach rotates across healthy sendboxes by lowest `sent_today/quota_today`; once a contact receives its first email, that sendbox is sticky for all later bumps/replies. A broken box is dropped from step-1 rotation; its pending follow-ups wait (never reassigned) with an `[ACTION REQUIRED]` re-auth. Consumer `@gmail.com` boxes default to `plain_text_mode` (no tracking pixel/link rewrite) and are measured by reply.

## Enrichment and personalization

Enrichment gathers evidenced facts and fresh hooks per contact; the dossier belongs to the contact and is inherited across that client's campaigns (durable identity/context ~90 days; fresh hooks 7–14 days; negative caches like `email_not_found` are inherited too). Every personalized detail in a drafted email must map to a dossier hook that carries an `evidence_url`. Before a follow-up, re-check and invalidate stale hooks. Do not reference logged-in-only content (Facebook/LinkedIn posts) as if read; the MVP stores only URLs for those. Guessed email addresses require third-party verification, catch-all handling, a per-domain kill switch, a ≤10%/day/box cap, and never auto-send.

## Notification provider (WideCast, notification only)

WideCast is used ONLY to notify the operator (email + Telegram in one `sendNotification` call (required `subject` + `message`; Telegram added automatically when connected), optional `uploadAsset` for the report link). It is NOT used to produce or publish content. Check the client's `integrations/providers/provider_config.local.json`, verify identity with the client's key, and inspect the client's OpenAPI capabilities before claiming notification is available; do not treat a global MCP account as this client's connection. Save the key only as `api_key_env` or `api_key_local`, never a field named `api_key`. Use `https://widecast.ai/app/dashboard`; treat `https://api.widecast.ai` as a disabled host. Notification is optional; with no provider, surface report links in chat and log the blocker. When explaining WideCast setup, give the exact steps: register at `https://widecast.ai/#setup`, log in, `Setup AI Agent`, open `API Keys & MCP`, `Setup`, `Generate API key and MCP url`, then paste only the API key. Mention Telegram connection as optional WideCast-side setup, not an extra chat question.

## Two-lane reporting

Operator-only reports (Approval Report, Today View, daily ops, `INTERNAL_REPORT`) carry full internal detail and are NOT scrubbed. The weekly client report is the ONLY client-facing deliverable and must pass the Client-Blind Scrub Gate — do not mention OutreachCRM, WideCast, PDNA, OpenAPI, MCP, API keys, Telegram, automation/scheduled tasks, sendboxes, crm_store, sent_log, suppression, warmup, quota, guessed, tracker domains, config files, agent/debug details, or `INTERNAL_REPORT` in it. Before rendering any report, load `playbooks/skills/report-design/SKILL.md` and use `tool render-report` (with `--client-facing --fail-on-scrub` only for the weekly client report). Do not write one-off report/PDF scripts; fix the reusable renderer or log the exact blocker.

## Resync

After a schedule/automation exists, any later approved change to the profile, pipeline/rules, sendboxes, campaigns, suppression policy, notification config, schedule, tracker, or storage backend must trigger an Automation Resync across the whole automation package (profile/config via `tool crm-store`, provider config, `schedule.md`, campaign/sendbox config, automation manifest, scheduled-run prompt/task body, update state, resync log), followed by a dry-read verification. Every human-facing progress block after automation exists must include an `Automation freshness check` line.

## Source of truth and updates

For setup, repair, or update, treat GitHub `main` (`https://github.com/soloagency/solo-agency`, module subpath `outreach/`) as the source of truth unless the current setup root is a verified fresh clone. Do not reuse fixed shared temp folders. Verify `.git` exists, `origin` matches, and local `HEAD` == `git ls-remote origin refs/heads/main` before trusting a checkout. If the human says `update`/`upgrade`/`cập nhật`/`sync latest`/`pull latest`, load `playbooks/11_UPDATE_AND_VERSION_WATCH.md` and run the update workflow (check GitHub main; diff playbooks/tools/tracker worker/storage adapter/schema; apply safe updates preserving secrets, client data, suppression, and tokens; resync every client and task; update `update_state.json`/`update_log.md`). This is not a request to send email or run a campaign.

Whenever the agent hits a blocker, unexpected behavior, repeated failure, or dead end, treat stale local playbooks/code as the first suspect: fetch/verify the latest GitHub `main`, reload changed instructions, and retry using the newest rule before declaring the task blocked. If the latest version still does not resolve it, escalate a redacted issue (never containing secrets, tokens, sendbox credentials, or contact PII) via an authorized GitHub identity, a configured intake channel, or a draft under `daily-content-pipeline/automation/issues/`, tracked in `daily-content-pipeline/automation/github_issues.md`.

After setup/routine exists, offer/maintain the daily `OutreachCRM - GitHub Update Watch` task described in `playbooks/11_UPDATE_AND_VERSION_WATCH.md`. It checks GitHub for new versions, classifies changes, writes a local update notice, and applies/resyncs updates only when the human approved auto-apply. It must NOT send Telegram/provider/client notifications and must NOT touch anything under `clients/` (version maintenance is internal agency work). If the tracker worker or storage schema changed, the handoff must include the exact `wrangler deploy` rerun command or the `tool crm-store migrate` step.

The repo entrypoint is `OUTREACHCRM_PLAYBOOK.md`.
