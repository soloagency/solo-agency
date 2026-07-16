# OutreachCRM

Tell your AI agent: **"Setup https://github.com/OWNER/outreachcrm now"** to turn it into a local-first, multi-client cold-email + CRM operator that works every day: it enriches your leads, drafts personalized emails, shows you every draft for approval, sends only what you approve, tracks replies, and moves opportunities through your pipeline.

Open source (MIT). Runs on your machine through Codex or Claude Desktop/Cowork — your contact data and email accounts stay local.

## The idea

Cold email changed. Sending 10,000 shallow emails a day is a losing game against modern spam filters and a saturated inbox. The winning game is fewer, better emails: verify the person is real and active, personalize with a fresh, *evidenced* detail, send from a warmed sendbox inside normal limits, and put your energy into follow-up. An AI agent is very good at exactly that work — verifying, finding the one honest hook, drafting, and tracking — while you stay in control of what actually sends.

OutreachCRM is that agent's operating playbook, plus the local tools it drives.

## What it does

- **One agency → many clients.** Each client is an isolated workspace: its own pipeline, sendboxes, suppression list, campaigns, and reports. You operate on their behalf; they receive a weekly report.
- **Import any list.** CSV / TXT / XLSX. The agent infers the column mapping, de-duplicates, and checks suppression. Email is optional — a contact can be just a name + phone/social, and enrichment (or you) can fill in a channel later.
- **Verify + enrich before writing.** A cheap pass confirms the person is still active and gathers profile URLs; a deeper pass finds a fresh, evidenced hook (a new listing, a recent post, a review) and drafts the email. Every personalized detail must carry a source URL — no invented facts.
- **Goal-driven campaigns.** Each campaign declares a goal (book a meeting, get a reply, direct sale, reactivation, …) that drives how the email is written and how a reply converts into a pipeline deal.
- **Approve, then send.** Every daily run produces an Approval Report: one card per lead with the dossier, clickable evidence, and the drafted email, grouped by confidence. You approve in chat (`approve all`, `approve 1-20, 35`, `reject 7: reason`, `edit 12: ...`). Only approved emails send.
- **Multi-sendbox rotation.** Add several sendboxes per client; step-1 emails rotate across healthy boxes, and each contact then sticks to its sendbox for every follow-up and reply.
- **Tracking that's honest about limits.** Replies, bounces, and unsubscribes are detected reliably from the mailbox. Opens/clicks are optional and clearly labeled as estimates; they never trigger an automated action — only a reply does.
- **A real CRM underneath.** Contacts, accounts, deals, activities, and tasks; a per-client pipeline with deterministic rules; a Today View and kanban; a weekly client report.
- **Notifications + follow-up.** Optional Telegram notification (via WideCast) tells you what happened and what's waiting. The system knows how many emails a person has received, what they replied, and what to say next — and drafts it for you.

## Priority path: @gmail.com sendboxes

You can start with a dedicated Gmail account and an App Password — no Google Cloud project, no OAuth app verification. Send via SMTP, read replies via IMAP, all local. For custom-domain sending with full open/click tracking, connect a Google Workspace mailbox (OAuth, Internal app) — the schema supports both side by side. Either way, use a dedicated account, never your primary inbox, and keep volume in the 20–50/day per box range for cold outreach.

## What stays in your control

Nothing sends without your explicit approval in chat. The agent drafts and waits. Suppression (bounces, unsubscribes, complaints) is checked before every send. Opt-outs are honored immediately. Assisted-channel messages (SMS / Messenger) are drafted for *you* to send manually. Compliance guardrails (CAN-SPAM address + opt-out, truthful subjects, no fake `Re:`) are built into the send path.

## Best first prompt

```text
Setup https://github.com/OWNER/outreachcrm now.
```

The root playbook (`OUTREACHCRM_PLAYBOOK.md`) tells the agent which detailed stage playbook to load next. If the `playbooks/` folder is not local, the agent fetches the needed stage from this repo.

## Best new-client prompt

```text
Add a new client: [client name].
They sell [product/service] to [ideal customer].
Their website/profile: [URL].
Target market: [location if it matters].
Set up the client-specific daily automation task. Do not send anything in this setup chat.
```

## Agent entry point

If you are an AI agent setting up this repo:

1. Read `OUTREACHCRM_PLAYBOOK.md` first, then `docs/DESIGN.md` (the authoritative design).
2. Follow the Stage Map and the LOAD LEDGER full-load discipline (`playbooks/LOAD_LEDGER_PROTOCOL.md`).
3. Setup Flow only configures — it never sends. Sending happens in the scheduled daily run, and only after the operator approves drafts in chat.
4. All CRM data lives under `outreach-pipeline/` and every mutation goes through `tools/crm_store.py`.
5. Keep each client's workspace isolated; a `target_client_slug` run touches only that client.

## Status

Under active construction. Phase 0 (this commit) delivers the inherited operating system — router, load discipline, setup/automation split, storage/schema contract, safety audit, update watch, deploy tooling — transformed for OutreachCRM. The email/CRM tools (`crm_store.py`, `gmail_client.py`, `import_leads.py`, `email_verify.py`), the tracker worker, and the Phase 1–2 stage playbooks are built next. See `docs/DESIGN.md` §21 for the phase plan.

## Agent compatibility

Designed for Codex and Claude Desktop/Cowork. No vision model required.
