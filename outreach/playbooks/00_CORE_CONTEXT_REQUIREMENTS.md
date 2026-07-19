# Core Context Requirements

Stage: `00`

## Load Rule

Load this file first, IN FULL, for every setup session and every scheduled run, before anything else. It defines the mission, the operating model (1 agency → N clients → N campaigns), the non-negotiable inherited mechanisms, client-scope isolation, the core operating workflow (import → verify → enrich → goal-driven drafting → preview & chat-approval → send → tracking → follow-up → CRM → weekly report), the compliance backbone, and the language/reporting rules. Downstream stages assume the contracts stated here are already in force.

Before acting on this or any other stage, obey `playbooks/LOAD_LEDGER_PROTOCOL.md`: read the file to its last line, print a LOAD LEDGER (path, `lines_read` vs `LOAD_MANIFEST.md`, dependency list, verdict) before you act on it, and ledger every named dependency. A short/truncated/previewed read = the file is NOT loaded. No side-effect action without a `Verdict: PASS loaded-in-full` ledger earlier in the same transcript.

## Hard Gates For This Stage

These are the non-negotiables. They are restated in detail in the relevant sections below, but they bind from the moment this file is loaded. Every one of them is enforced in code where DESIGN says so — never only in prose.

- **Infer before asking.** Think, research, and infer as much as possible before asking the human anything. Ask only for what is truly required and cannot be inferred, researched, or read from a public URL the human gave you.
- **All CRM mutations go through `crm_store.py`.** Writing directly to any file under `clients/{slug}/crm/` (contacts, accounts, deals, activities, tasks, pipelines, suppression) is a critical violation of the inherited "no one-off scripts" rule. Reading raw JSON is allowed only for debugging.
- **Evidence-URL rule for personalization.** A draft may contain only details that are present in the contact's dossier with an `evidence_url`. No claim, fact, hook, or personalization line without a backing `evidence_url`. Stage 9 audit checks this mechanically before any send.
- **A reply freezes the sequence.** Any inbound reply from a contact freezes the remaining follow-up sequence for that contact until triage completes. This invariant is enforced in code at both draft-time and send-time.
- **Open/click never alone trigger an action.** An open or a click is never, on its own, a reason to change a deal stage, advance lifecycle, or take an auto-action. **Only a reply is conversion evidence.** Opens are labeled "estimated" everywhere.
- **Suppression is checked at every send-capable path.** Global (agency-tier) and client-tier suppression are checked before every initial send, every follow-up, and every assisted-channel draft, and at import against ALL identities of the contact. If a box's track-pull has not succeeded within the configured window, sending from that box is blocked so unsubscribes cannot sit unhonored.
- **Setup Flow never operates.** Setup Flow is the control plane. It never sends an email, never enriches for send, never runs a campaign, never pulls tracking, never notifies a client. Its terminal state is `ready_for_automation_first_run`.
- **One automation task per client.** Each client-specific task name begins with the client name, pins `target_client_slug`, and cannot touch another client. Plus exactly one agency-wide `OutreachCRM - GitHub Update Watch` task, which is barred from every client-facing channel.
- **Client scope is structural.** Every client is an isolated CRM workspace rooted at `clients/{slug}/crm/`. Only the explicitly enumerated agency-tier collections may be global.
- **Full-load discipline.** Every stage/module/dependency load requires a LOAD LEDGER; a short read = NOT loaded; no side-effect without a PASS ledger.
- **Only the weekly report is client-facing.** It is produced through the Client-Blind Scrub Gate. Every other output (approval report, Today View, daily ops, `INTERNAL_REPORT`) is operator-only and must never be shared with the client.
- **Never a field literally named `api_key`.** Use `api_key_env` (environment variable name) or `api_key_local` (local client value). Never store credentials, OAuth secrets, refresh tokens, or the tracker key in client config, reports, issue drafts, or committed files.
- **WideCast is notification-only.** WideCast is the operator's Telegram-notification provider: `sendNotification` (email always, plus Telegram when connected), and optional `uploadAsset` to host the report link. It is not a content, publishing, or production tool.
- **`[ACTION REQUIRED]` contract.** When the human must act, use a standalone `**[ACTION REQUIRED]**` block: one purpose, one exact next step, one command or one absolute path. When nothing is needed, end with next-action guidance per the root playbook's Next-Action Guidance Rule, never `No action required right now.`
- **Guessed email is code-enforced.** Guessed/unverified addresses require an explicit guessed-approval flag on the draft plus a daily guessed-send cap, both enforced in `gmail_client.py send`. First guessed-pattern hard bounce at a domain suppresses all other guessed addresses at that domain.

## Source Preservation Rule

This file is detailed source material for the OutreachCRM operating system. It replaces the content-marketing core-context of the original monolith with the cold-email + CRM core context defined in `docs/DESIGN.md`.

Do not summarize away requirements, schemas, ordered checklists, protocols, gates, thresholds, or edge cases. A downstream agent may summarize its human-facing reply, but it must still obey the full requirements in this file. When this file and `docs/DESIGN.md` disagree, `docs/DESIGN.md` wins; report the gap.

---

## Latest Override: Control Plane Versus Operations Plane

OutreachCRM strictly separates setup/configuration from operational runs. This override wins over any older wording anywhere in the repo that would have a setup session send an email, enrich for send, run a campaign, or deliver a report.

- **Setup Flow is the control plane.** It creates and updates client config, sendbox connections, imported lists, campaign definitions, schedules, the client-specific automation task, provider (notification) config, and resync logs. It must not send email, verify/enrich for send, draft-for-send a campaign, pull tracking, run `apply-rules`, or notify a client.
- **Automation Flow is the operations plane.** It runs the configured pipeline from saved state (sync inbox → pull tracking → triage + rules → follow-up advising → load new pipeline → send approved → assisted drafts → compile views → reports → notify → audit). It may also record practical config changes discovered during a run (a new bounce pattern, a segment tweak), but it must resync those changes for future runs.
- **The first run is not a setup deliverable.** It must execute through a client-specific automation task whose name begins with the client name, e.g. `Acme Realty - OutreachCRM Daily Run`.
- **Client-specific tasks process only their pinned `target_client_slug`.** A task cannot read, draft into, send from, or suppress within any other client's workspace.
- **Terminal setup state is `ready_for_automation_first_run`.** Setup ends when configuration is current and the human has the exact automation-task name to run.

Send/report request hard stop in Setup Flow:

- If the human asks to send, run a campaign, enrich, generate/refresh/show a report, or "just do the first run" while the current session is Setup Flow, do not comply by operating.
- Such a request does not switch the setup chat into Automation Flow.
- Finish or resync the client-specific automation task and tell the human the exact task name to run.
- Do not ask "Do you want me to run it now?" in Setup Flow.
- Do not load the scheduled-run entrypoint as a workaround inside the same setup chat.
- Do not start inbox sync, tracking pulls, enrichment for send, drafting for send, sending, or client notification inside Setup Flow.
- If the native automation task cannot be updated by the agent, mark `automation_prompt_update_pending`, write the exact prompt/update instructions to `daily-content-pipeline/automation/scheduled_run_prompt.md`, and ask the human to update/run the native task.

Required response pattern in Setup Flow when asked to operate:

```text
I will not send or run a campaign in this setup chat because Setup Flow is only for configuration. I will finish or resync the client-specific automation task instead. After setup is ready, run `{Client Name} - OutreachCRM Daily Run` for the first pipeline pass.
```

---

## Required Runtime

OutreachCRM must run in an AI agent runtime that supports:

- local workspace file reads/writes (the entire data root `daily-content-pipeline/` lives on the operator's machine);
- scheduled automation or native tasks (one per client, plus the update-watch task);
- multiple parallel/sub-agent work streams for verify/enrich, drafting, inbox sync, tracking, reporting, and resync verification;
- outbound email and inbox read for at least one sendbox (SMTP+IMAP for `app_password`, or the Gmail API for `oauth`);
- running local Python tools (`crm_store.py`, `gmail_client.py`, `import_leads.py`, `email_verify.py`, `report_renderer.py`, `provider_openapi.py`) and, when tracking is enabled, a Cloudflare Worker under `trk.{domain}`.

Good runtime examples include Codex, Claude Desktop/Cowork, or a comparable desktop/local agent environment. A plain web chat is not enough: it cannot reliably host the sendbox credentials, the local file state, the scheduled runs, or the multi-agent work OutreachCRM requires. When asked how to install or run OutreachCRM, say this plainly before setup proceeds; do not imply that pasting the playbook into a browser-only chat creates the system.

---

## 0. Non-Negotiable Inherited Mechanisms

These are the machinery the whole architecture exists to enforce. Keep them near-verbatim in behavior; do not weaken them.

### 0.1 Thin router + Stage Map

`OUTREACHCRM_PLAYBOOK.md` is a dispatch table only. Business logic lives in the numbered stage playbooks, loaded on demand. Do not paste stage protocols back into the root. Stage Map (load-when):

| Stage | File | Load when |
|---|---|---|
| 0 | `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md` | always first |
| — | `playbooks/LOAD_LEDGER_PROTOCOL.md` | referenced by every load |
| 1 | `playbooks/01_CLIENT_SETUP_PROFILE.md` | new client setup / first run |
| 2 | `playbooks/02_SENDBOX_SETUP.md` | connect/check a sendbox |
| 3 | `playbooks/03_IMPORT_LIST.md` | import a CSV/TXT/XLSX list |
| 4 | `playbooks/04_VERIFY_ENRICH.md` (+ skill `email-verify-enrich`) | before any enrichment |
| 5 | `playbooks/05_CAMPAIGN_MANAGEMENT.md` | create/edit a campaign, define its goal |
| 6 | `playbooks/06_EMAIL_WRITING_STANDARD.md` (+ skill `email-writing`) | before drafting any email |
| 7 | `playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md` | any file create / history write |
| 8 | `playbooks/08_SEND_ENGINE_PROTOCOL.md` | before any send |
| 9 | `playbooks/09_OPERATIONS_SAFETY_AUDIT.md` | before claiming completion |
| 10 | `playbooks/10_FOLLOWUP_REPLY_MANAGEMENT.md` | inbox sync, follow-up advising |
| 11 | `playbooks/11_UPDATE_AND_VERSION_WATCH.md` | update/upgrade/sync-latest |
| 12 | `playbooks/12_TRACKING_ANALYTICS.md` | read metrics, learning loop |
| 13 | `playbooks/13_CRM_CORE.md` | objects, lifecycle, stage rules, dedupe/merge |
| 14 | `playbooks/14_TASKS_TODAY_VIEW.md` | task engine, SLA, Today View |
| 15 | `playbooks/15_CRM_REPORTING.md` | pipeline report, forecast, weekly client report |
| 6A | `playbooks/skills/report-design/SKILL.md` | report rendering |
| Auto | `playbooks/AUTOMATION_SCHEDULING.md` | configuring the schedule/automation task in Setup Flow; the start of every scheduled run; any Automation Resync. Defines the Daily Run order, run_lock, and resync machinery. |
| Setup | `playbooks/SETUP_FLOW_ENTRYPOINT.md` | setup sessions |
| Sched | `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` | unattended daily runs |

Some stages/tools/skills may be marked `status: planned` until built; their Stage Map rows still exist. Load only the stage needed for the current action plus any dependency that stage names, each with its own LOAD LEDGER. Until a planned file exists, load `docs/DESIGN.md` for its contract (DESIGN §22 R1).

### 0.2 Full-load discipline (LOAD_LEDGER)

`playbooks/LOAD_LEDGER_PROTOCOL.md` plus the auto-generated `playbooks/LOAD_MANIFEST.md` (`path | lines | last_line | sha256`) enforce that every stage is read to EOF before acting. A read that errors, truncates, previews, 404s, times out, or returns fewer lines than the manifest = NOT loaded; re-read to EOF (chunk large files with `offset`/`limit` or `sed -n 'A,Bp'`). No side-effect action (ask the first setup question; import; verify; enrich; draft; send; notify; write client/automation state; claim completion) without a `Verdict: PASS loaded-in-full` ledger earlier in the transcript, with dependencies ledgered. No excuse ("too large", "save tokens", "running from schedule", "I remember it", "human wants it short") suppresses full-load or the ledger. Brevity applies only to the human-facing summary.

### 0.3 Setup Flow vs Automation Flow split

Stated in full in the override above. The control-plane / operations-plane split is a hard boundary, not a suggestion.

### 0.4 One automation task per client

Client-specific task name begins with the client name (task lists truncate long names), pins `target_client_slug`, and cannot touch another client. Example name (there is exactly one client-specific task):

```text
Acme Realty - OutreachCRM Daily Run
```

The first run is just that `{Client} - OutreachCRM Daily Run` task's first execution — there is no separate `First Run` task. Plus one agency-wide `OutreachCRM - GitHub Update Watch` task (§0.6), barred from every client-facing channel.

### 0.5 Automation Resync

Any post-setup change that affects what a future scheduled run should do or read triggers an Automation Resync before the change is called complete. This includes: client profile fields (offer, sending identity, voice, physical address, compliance notes), sendbox add/remove/warmup/auth-mode, imported lists and segments, campaign definitions and goals, suppression policy, schedule cadence/timezone/active-clients, provider (notification) config, tracker configuration, and update/version-watch state.

Resync updates the full package, not one file: the client profile + relevant state, `daily-content-pipeline/schedule.md`, `daily-content-pipeline/automation/automation_manifest.md`, `daily-content-pipeline/automation/scheduled_run_prompt.md` (and the native task prompt when editable), `daily-content-pipeline/automation/resync_log.md`, plus provider/tracker/update files when relevant. Finish with a **dry-read verification**: read the scheduled entrypoint, manifest, schedule, profile, and provider/tracker config as tomorrow's scheduled agent would, and confirm the newest approved state is visible. If the native task body cannot be edited by the agent, write the exact replacement prompt to `scheduled_run_prompt.md`, mark `automation_prompt_update_pending`, and ask the human to update the native task.

### 0.6 Stage 11 Update & Version Watch

`update`/`upgrade`/`sync latest`/`pull latest` is an update command, not a report request: load `playbooks/11_UPDATE_AND_VERSION_WATCH.md`, do a fresh verified GitHub checkout, compare local `HEAD` against `main`, classify changes (storage adapter / `schema_version`, `crm_store.py` / `gmail_client.py` / `import_leads.py` / `email_verify.py`, `tracker/worker.js` + its `wrangler deploy` rerun, sendbox token compat), back up and safe-apply (merge config, never overwrite secrets/history/suppression), and write `daily-content-pipeline/automation/update_state.json` + `update_log.md`. The daily `OutreachCRM - GitHub Update Watch` task performs this maintenance and must not send Telegram/email, provider notifications, or any client-facing message — version maintenance is internal agency work and writes nothing under `clients/`.

### 0.7 Fresh GitHub source + missing-playbook download

For setup, repair, update, or copying playbooks into a human's install, GitHub `main` is the source of truth unless the current root is a verified fresh clone of the same repo. Clone/download from `https://github.com/soloagency/solo-agency` (the OutreachCRM module is its `outreach/` subpath) into the current root or a fresh unique `mktemp -d`; never reuse fixed shared caches; verify `.git`, `origin`, and that local `HEAD` matches `git ls-remote origin refs/heads/main` on the parent checkout (the `outreach/` module has no `.git` of its own) before reading/copying; treat non-`.git`, wrong-owner, or stale-timestamp folders as bad cache; never fall back to unverified local code. If the local `playbooks/` folder is missing a stage, download it from:

```text
https://raw.githubusercontent.com/soloagency/solo-agency/main/outreach/playbooks/
```

After any GitHub-raw download, verify it against `playbooks/LOAD_MANIFEST.md` via a LOAD LEDGER; a short/partial download must be re-fetched before use.

Carve-out (DESIGN §22 R1): Stages marked `status: planned` are not missing files — do not GitHub-fetch them and do not enter Last-Resort Recovery for them; load `docs/DESIGN.md` for their contract instead.

### 0.8 Last-resort recovery + GitHub issue escalation

Because the repo changes frequently, treat any blocker, repeated failure, contradiction, tool/config mismatch, stale asset, or dead end as a possible old-playbook/code problem first: do a Fresh GitHub Source Check, record local vs `main` commits, reload the relevant latest stage, and follow it if it fixes the issue (resync afterward). If still blocked, open or draft a GitHub issue for `soloagency/solo-agency` (prefix the title `outreach:` for triage):

- The human does not need a GitHub account; do not make registration the required next step.
- Direct creation requires an authorized identity: `gh issue create` only when `gh auth status` passes and `GITHUB_TOKEN`, `GH_TOKEN`, or `OUTREACHCRM_GITHUB_ISSUE_TOKEN` is configured, or a maintainer bot/App is available. Never store this token in client config, reports, or committed files.
- If no authorized identity but an intake channel is configured, send/queue the redacted draft there.
- Otherwise write a ready-to-post draft under `daily-content-pipeline/automation/issues/YYYY-MM-DD_{blocker_slug}.md` and track it in `daily-content-pipeline/automation/github_issues.md`.
- Include a redacted blocker fingerprint, safe repro steps, expected/actual, local commit, `main` commit checked, runtime, and redacted logs. Never include API keys, tokens, OAuth secrets, the tracker key, credentials, raw prospect PII, or client-confidential data. Reuse an existing issue when the fingerprint matches; do not duplicate.

### 0.9 Provider adapter + PDNA notification (Telegram only)

Per-client `integrations/providers/provider_config.local.json` holds `api_key_env` or `api_key_local` (never a field literally named `api_key`). OpenAPI discovery runs via `tools/provider_openapi.py`; capabilities cache in `provider_capabilities.json`. The only surviving provider use is **operator notification (PDNA notification)**: WideCast `sendNotification` (email always, plus Telegram when connected), and optional `uploadAsset` to host the report link. A report-ready notification is invalid unless it includes the HTML report URL/path; if one is ever sent without the link, send an immediate correction. Before asking the human for a WideCast key, read existing config first: reuse a client-scoped key that already exists for THIS client (OutreachCRM's own config, or the same client's sibling Solo Agency content-pipeline provider config one level above `outreach/`) instead of re-asking. Do not treat a global MCP/native account as this client's provider; verify against the client's own config path and log `global_mcp_not_client_scoped` if it cannot be proven to match. Reusing the same client's own/sibling client-scoped key is allowed; adopting a global MCP account is not.

### 0.10 Two-lane reporting

Operator-only outputs (`INTERNAL_REPORT`, approval report, Today View, daily ops) carry full technical detail. The **weekly** client report is the only client-facing output and passes through the Client-Blind Scrub Gate. All reports are rendered by `tools/report_renderer.py` (stdlib only). The scrub term list drops any operator-only vocabulary; client-facing files must never mention OutreachCRM internals, sendbox/gmail_client/crm_store, `trk.`/HMAC/token, sent_log, suppression, warmup, quota, guessed, or `INTERNAL_REPORT`.

### 0.11 `[ACTION REQUIRED]` contract

Any human-facing reply, setup handoff, blocker, notification, or next-step question that needs the human to answer, approve, paste, run, connect, or confirm must put that request in a standalone block: one purpose, one exact next step, and one copy-paste command or one absolute path. Do not bury required actions in prose, progress blocks, or report links. At most three `**[ACTION REQUIRED]**` blocks per reply. When no action is needed, end with next-action guidance per the root playbook's Next-Action Guidance Rule. Never ask for passwords, cookies, OTPs, session tokens, or credentials; provider setup blocks may ask only for the specific API key the playbook allows.

### 0.12 Slug rules + monthly folders + deploy discipline

Slugs are lowercase, hyphenated, no punctuation. History and time-partitioned data live in monthly `YYYY-MM/` folders. Internal field names and schemas are English. Deploy is the root `deploy-soloagency.sh` `generate_outreach_artifacts` step (mode `--outreach-only`): it regenerates `outreach/playbooks/LOAD_MANIFEST.md`, rezips the skills, runs the module's 103-test suite as a preflight, secret-scans the staged diff before commit (`refresh_token`, `client_secret`, `TRACKER_API_KEY`, `token.json`, `client_secret*.json`), and refuses to commit into the wrong git root.

### 0.13 Canonical user-facing description

When explaining what OutreachCRM does, do not undersell it as "a mail merge." A good concise description:

```text
OutreachCRM runs cold-email outreach and a full CRM for one agency across many clients. For each client it imports prospect lists, verifies and enriches contacts with source-backed evidence, writes goal-driven personalized emails, shows every draft for chat approval, sends from rotating warmed sendboxes, tracks replies and bounces (opens/clicks optional and estimated), advises follow-ups, and manages the pipeline of accounts, contacts, deals, tasks, and activities — then sends the operator a run summary and the client a scrubbed weekly report. Nothing is sent without explicit approval, and every commercial email carries a working opt-out.
```

Do not imply anything is sent or auto-actioned without explicit approval, and do not imply opens/clicks drive conversions.

### 0.14 Setup progress roadmap (human-facing)

During Setup Flow, show and keep updating a human-facing roadmap so the human can catch missed steps. It is the agent's planned process, not a form. Use `You` for the human's actions, `I` for the agent's; use status icons: `✓` done, `→` current, `○` pending, `!` blocked/needs-human, `–` skipped/declined/not-applicable with a short reason. Explain specialist terms in plain language (`sendbox`, `suppression`, `campaign goal`, `opt-out`, `enrichment`, `follow-up sequence`).

```text
OutreachCRM one-time setup process
This is the planned setup process I am working through. You only need to reply when I ask one specific question.

→ 1. You provide the product/service or business, ideal customer, and (optional) website/URL
○ 2. I infer the ideal-customer profile, value proposition, and email voice, then show them for correction; I propose a pipeline (stages) and custom fields
○ 3. You confirm the sending identity: from-name, signature, physical mailing address, and unsubscribe method (the compliance basics for the email footer)
○ 4. You connect the first sendbox (the mailbox we send from; a dedicated Gmail via App Password is the quickest/priority path, Workspace/custom-domain OAuth is the fallback)
○ 5. You give me a contact list (CSV/TXT/XLSX); I map columns, de-duplicate, and check suppression (the do-not-contact list)
○ 6. We create the first campaign and its goal (e.g. book a meeting — the goal is the writing blueprint), sequence, and daily quota
○ 7. I set up operator notifications (optional): Telegram via a WideCast API key, with email fallback, so you get run summaries and the report link
○ 8. I record a baseline (nothing has been sent yet)
○ 9. I create the client-specific daily automation task ({Client} - OutreachCRM Daily Run); from then on the daily run enriches, drafts, and shows you an Approval Report — nothing sends until you approve
```

Roadmap integrity: always show all items in order (all 9); never hide later items because they are pending; mark `–` only after an explicit human decline or a logged not-applicable reason; the automation task is created LAST (item 9), and the verify/enrich/draft/send/track/report work it runs (item 9) never runs in Setup Flow. After any approved config change once a schedule exists, run Automation Resync and show an `Automation freshness` line (`current` / `resync in progress` / `action needed` / `not applicable yet`).

### 0.15 Progress + next-step question rule

While setup or a run is still incomplete, every human-facing reply that hands control back must include a compact progress block. If any required step remains and the agent is waiting, the final line must be exactly one clear next-step question — never a passive summary, a bare report link, or "let me know what you think." Even when the entire requested workflow is complete and no human decision is required, still close with next-action guidance AND a feature-discovery block per the OutreachCRM Feature Discovery Rule (unused headline features from `playbooks/FEATURE_CATALOG.md` in the Solo Agency root, both products). Human-facing reports are HTML; Markdown files are internal source-of-truth records and must not be handed to the human as the report experience.

---

## 1. Non-Negotiable Operating Principles

The agent must follow these at all times.

- Preserve every requirement in this file. Think and infer before asking. Ask only for what cannot be inferred, researched, or read from a public URL the human provided.
- During setup, ask step by step; after every human answer, immediately infer what can be inferred and show the inference before asking the next question.
- Do not ask the human to define `industry` or `sub_industry`. Ask first only for the client's business/offer/target-prospect profile, or a public website/profile URL the agent can read for setup context. Reading that public page for setup context is allowed in Setup Flow; it is not an operational run.
- Ask for `target_location` only when the offer is location-dependent and it cannot be inferred.
- **All CRM mutations go through `crm_store.py`.** Never hand-edit or one-off-script a file under `clients/{slug}/crm/`. Every record carries `schema_version`, `id`, `created_at`, `updated_at`; the adapter applies its upgrade registry on read. Raw reads for debugging only.
- **Client scope is structural.** The storage adapter is instantiated per client rooted at `clients/{slug}/crm/`. The only collections allowed to be global are the enumerated agency-tier ones (§2.3). Never let one client's data, quota, suppression, or sendbox leak into another.
- **Evidence-URL rule.** Personalization must be grounded: a draft may reference only dossier facts that carry an `evidence_url`. If a hook has no evidence URL, it does not go in the email. Stale hooks (a sold listing, a past event) must be invalidated at write time.
- **A reply freezes the sequence.** The moment a contact replies, the remaining bumps for that contact are frozen until semantic triage completes. Enforced in code at draft-time and send-time; never rely on prose alone.
- **Open/click never alone trigger an action.** Only a reply is conversion evidence. Never advance a deal, change lifecycle, or auto-send because of an open or a click. Opens are reported as "estimated."
- **Suppression at every send path.** Check global + client suppression before every initial send, follow-up, and assisted-channel draft, and at import against ALL identities. Suppression is unioned on merge; pending-merge contacts are excluded from every queue. If a box's track-pull has not succeeded within the configured window, block sending from that box.
- **Nothing leaves without an explicit approve.** Default `approval_mode: manual_all`, even for bumps. Chat is the write path for approvals; editing the rendered HTML does not persist. See §3E.
- **Guessed email is code-enforced.** A guessed/unverified address sends only with an explicit guessed-approval flag on the draft and within a daily guessed-send cap, enforced in `gmail_client.py send`. Route guessed/unverified through a third-party verification API; exclude/limit `catch_all`; kill all guessed sends at a domain after its first guessed hard bounce; report the guessed cohort's bounce rate separately.
- Never send from the operator's primary personal Gmail; never scale cold @gmail.com volume beyond the documented ~20–50/day/box at tight personalization.
- Do not require any provider account, API key, MCP connection, or installed tool merely to import, verify, enrich, or draft. Only operator notification uses a provider, and even that has an email fallback.
- When provider notification is available, use it to notify the human about completed scheduled runs, drafts awaiting approval, sendbox re-auth needs, blockers, and important failures, because the human may be away when the schedule runs. Every report-ready notification must include the HTML report URL/path.
- Use the `**[ACTION REQUIRED]**` contract for every required human action; otherwise end with next-action guidance per the root playbook's Next-Action Guidance Rule.
- Communicate with the human in the human's language. Store internal field names and schemas in English. Human-facing reports, notifications, and approval requests are in the human's language. Email copy is written in the recipient's language (default English for the priority path unless the campaign/segment says otherwise).
- User-facing reports must be HTML. Do not hand the human `.md` report files as the report experience or make them open Markdown to learn the next step; put next actions directly in chat, the notification, or the HTML report.
- Never fabricate facts, evidence URLs, metrics, replies, or send results. Mark unavailable metrics clearly. If a required signal is missing, say so and stop at the honest blocker.

---

## 2. The OutreachCRM Model: 1 Agency → N Clients → N Campaigns

### 2.1 The model

One agency operates OutreachCRM on behalf of many clients. Each **client** is a fully isolated CRM workspace with its own pipelines, sendboxes, suppression, lists, and data. Each client runs **N campaigns**; each campaign declares its own **goal**, and the goal is the blueprint that drives what the agent writes (§3D). The agency positions as open source (MIT), English playbooks, @gmail.com sendboxes as the priority sending path, operating on behalf of clients, and delivering only a scrubbed **weekly** report to the client.

### 2.2 Isolation is structural, not disciplinary

The storage adapter is instantiated per client, rooted at `clients/{client_slug}/{business_slug}_{location_slug}/crm/`. A client's contacts, accounts, deals, activities, tasks, pipelines, segments, and client-tier suppression are readable and writable only within that client's root. There is no code path by which client A's automation task sees client B's data, consumes client B's sendbox quota, or writes into client B's suppression. The one-task-per-client rule (§0.4) reinforces this at the scheduler level.

### 2.3 The only things allowed to be global (agency-tier)

Enumerated explicitly — nothing else may be global:

- `daily-content-pipeline/suppression/global_suppression.jsonl` — agency-tier suppression, checked before every send in every client.
- `daily-content-pipeline/secrets/` — gitignored agency-wide secrets (OAuth client, tracker key).
- `daily-content-pipeline/provider_defaults.json` — WideCast notification catalog, no secrets.
- The tracker key used to sign/verify tracking tokens.

Everything else is per-client. Global suppression is checked in addition to (never instead of) the client's own `crm/suppression.jsonl`.

### 2.4 On-disk shape (reference)

The data root is `daily-content-pipeline/` (the repo itself holds no client data). Per client: `client_profile_*.md`, `sendboxes/`, `lists/`, `crm/` (accounts, contacts, `contact_identities.jsonl`, deals, monthly `activities/`, tasks, `pipelines.json`, `segments.json`, `suppression.jsonl`), `campaigns/{slug}/` (config, enrich queue, `outbox/pending_approval` → `outbox/approved`, monthly `sent/`, history), `assets/`, `approvals/`, `analytics/`, `inbox_sync/`, `reports/`, `outputs/YYYY-MM/YYYY-MM-DD/` (approval report, Today View, daily ops, `INTERNAL_REPORT`, and the Monday weekly client report + PDF), and `integrations/providers/`. Full schema lives in Stage 7; do not re-derive it here.

---

## 3. Core Operating Workflow (A–H)

This is the OutreachCRM operating model, end to end. Stage 0 states the reasoning and the invariants; the deep mechanics live in the referenced stages and are loaded (with a LOAD LEDGER) before the matching action. The pipeline is: **import → verify → enrich → goal-driven drafting → preview & chat-approval → send → tracking → follow-up → CRM pipeline → weekly report.**

### A. Client Scope & the Isolation Boundary

Every operation names its client and pins `target_client_slug`. Before any read/write, confirm the storage adapter is rooted at that client's `crm/`. The audience of OutreachCRM is not a content viewer — it is the client's set of **prospects** (accounts and contacts) plus the client's existing relationships. The job is to move prospects along a lifecycle (`lead → engaged → opportunity → customer → evangelist`, or `lost` / `do_not_contact`) using compliant, personalized, goal-driven email, and to record every touch as an activity.

The 80/20-style discipline inherited from the source becomes, here, **relevance discipline**: the campaign's goal and the contact's dossier decide what is written. Do not stray into generic pitches. If the dossier gives no honest, evidenced reason to contact this person for this goal, that contact is a `no_hook_fallback` (generic honest opener) or is skipped — never padded with invented relevance.

### B. Import & Identity

Load Stage 3 (`03_IMPORT_LIST.md`) before importing.

- Lists arrive as CSV/TXT/XLSX and are imported through `import_leads.py` into `crm/contacts/{lead_id}.json`. `lead_id` is a **ULID minted at import** — not a hash of email, because **email is NOT required**. A contact may enter with only a name and a social URL.
- Every contact carries `identities` (emails with per-address `source` and `status`, phones, socials, website), `channels` (email/sms/messenger/zalo with status and — for assisted channels — a documented opt-in), `lifecycle_stage`, `tz` (for the send-window gate, inferred from state/area code), tags, custom fields, and (after enrichment) a distilled `enrichment` copy of the dossier.
- **Dedupe/merge is deterministic.** Auto-merge on exact email / E.164 phone / canonical social-URL match; fuzzy name+company is proposed for human approval. The losing record becomes a permanent tombstone (`merge.status = merged`, `merged_into`), never deleted; identities, channel statuses, and suppression are **unioned** into the survivor. Every `lead_id` lookup path calls `resolve(lead_id)` to follow merge chains. A contact with a pending merge proposal is excluded from every campaign queue until resolved.
- At import, check every identity against global + client suppression; a suppressed identity marks the contact accordingly and keeps it out of queues.
- Accounts (`crm/accounts/{account_id}.json`) group contacts by company/office (e.g. a brokerage): name, domain, type, location, `contact_ids[]`.

### C. Verify & Enrich

Load Stage 4 (`04_VERIFY_ENRICH.md`) and the `email-verify-enrich` skill before any enrichment. Enrichment belongs to the **contact** (client-scope), not the campaign; campaigns reference `lead_id`, and the enrich queue is client-level, deduped by `lead_id` (one job even if two campaigns want the same person).

- **Two-tier flow.** Tier 1 Verify (cheap subagent): check the existing dossier first; if identity is within TTL, skip to hooks; else confirm still-active / license / roster / profile URLs and collect emails/phones. `inactive`/`unknown` → mark and stop (no Tier 2). Tier 2 Profile & hooks (main model): visit known URLs per the readability table, extract hooks **each with an `evidence_url`**, analyze social content, distill a `writing_brief`, and score `personalization_confidence` (≥0.7 High; 0.4–0.7 Review carefully; <0.4 → `no_hook_fallback`).
- **The dossier** carries identity (with evidence), context (market, specialty, style), and `hooks` (`type`, `summary`, analysis with sensitivity, `evidence_url`, `observed_date`, `confidence`, `used_in`). The `email-writing` skill consumes the `writing_brief` (ranked by freshness × goal-fit × confidence), never raw data.
- **TTL tiers + negative cache.** Durable identity/context ≈90d (inherited as-is by other campaigns); fresh hooks 7–14d (other campaigns run a cheap refresh, not full re-discovery). Negative cache is inherited too: `email_not_found` (retry after 30d, then stop), `no_verifiable_hook` (with last-tried date). A second campaign may not open with a hook already `used_in` on that person; a contact in an active sequence of campaign A is not drafted by B (`min_days_between_touches_across_campaigns`).
- **Freshness gate at write time.** Before a step-1 draft, hooks must be within TTL (else refresh the known URLs). **Follow-ups do NOT re-run enrichment per bump**: enrichment runs once, richly, at entry; secondary Layer-B points are reserved across the sequence and the campaign's `message_bank` carries the rotation. A micro-refresh before a bump is opportunistic (reserved points used up AND collector has spare capacity), never a per-bump requirement. What stays mandatory before every bump is the **stale-hook guard**: a time-sensitive hook past TTL is re-verified (that one URL) or not referenced. **Hard rule: a draft may contain only details present in the dossier with an `evidence_url`;** Stage 9 audit checks this mechanically.
- **Channel reality (be honest).** Readable now (WebSearch/WebFetch/browser tool): personal website/blog (best), YouTube title/description, public Instagram/X (best-effort), Zillow/GBP reviews. **Not readable when logged out: Facebook, LinkedIn — store the URL only.** Do not promise reading logged-in-only content in the MVP.
- **Etiquette (hard rule).** `public_business` signals (listings, work posts, reviews, awards, market opinions) are fair game. `personal` signals (family, health, vacations, children) are **default-banned from email copy** and go only into `do_not_mention`.
- **Guessed email.** MX alone is near-meaningless on catch-all domains. Route guessed/unverified addresses through a third-party verification API (called from local Python); `catch_all` → excluded from the guessed quota or capped ~2%; per-domain kill switch on the first guessed hard bounce. `guessed_only` sends are enforced in `gmail_client.py send` (guessed-approval flag + daily cap), never only in prose; the guessed cohort's bounce rate is reported separately.

### D. Campaigns & Goal-Driven Writing

Load Stage 5 (`05_CAMPAIGN_MANAGEMENT.md`) to define a campaign, and Stage 6 (`06_EMAIL_WRITING_STANDARD.md`) + the `email-writing` skill before drafting.

The **goal is the writing blueprint, not a label.** `campaign_config.json` declares `goal` (`goal_type`, `objective`, `offer`, `value_proposition`, `proof_points[]` each with `evidence_url`, `cta`, and a `success_event` that wires straight into the rules engine), `audience` (segment + personalization requirements: `required_hook_types`, `min_confidence`, `no_hook_fallback`), a `sequence` of steps (each with intent, gap days, and tracking mode), sendboxes, `daily_quota`, `approval_mode`, guardrails (`banned_claims`, `no_fake_re`), and `channel_strategy`.

`goal_type → email structure` (the writing skill's format table):

| goal_type | Structure |
|---|---|
| `book_meeting` | short; one time-bound CTA |
| `get_reply` | ends with a question; **no link** |
| `direct_sale` | value + exactly one offer link (the only place click tracking is on by default) |
| `reactivation` | evidence of the prior relationship + "still doing X?" |
| `nurture_upsell` | new value tied to the existing relationship |
| `event_invite` | the invite + one clear RSVP CTA |
| every final step | breakup |

A draft = **client profile** (voice, offer, compliance) + **campaign goal** (objective, CTA, proof) + **contact dossier** (hooks + evidence) + **step intent**. Bumps (step > 1) carry NEW value; never "just following up." Step-1 subjects must not begin `Re:`/`Fwd:` (truthful subjects; linted). Continuation steps and replies are real in-thread replies with a truthful `Re:`.

### E. Preview & Chat-Approval (the gate before any send)

This is the hard gate. Nothing is sent without an explicit human `approve`.

At the end of a drafting pass, render an **Approval Report** (`outputs/.../{client}-approval-report.html`, operator-only, NOT scrubbed) via `tools/report_renderer.py` (reusing its contenteditable + Copy-button blocks): a header splitting drafts into **High confidence** (verified email, ≥0.7 hook) and **Review carefully** (weak hook, guessed email, fallback opener), then one card per lead — `#id` · name/company/email + verify status · hooks with **clickable evidence URLs** · subject + editable body + warning flags. Notify Telegram: "N drafts awaiting review" + the path.

Chat is the write path (editing the rendered HTML does not persist). Approval grammar:

```text
approve all
approve 1-20, 35, 41
reject 7: hook is stale, that listing sold
edit 12: change CTA to "Worth a quick look?"
hold 5
```

Approved drafts move to `outbox/approved/` and are **sent immediately in-session** (within quota, jitter, and the full in-code re-check chain). Rejected drafts are logged with the reason, which feeds `learning_log` for the next batch. Edits are patched, re-confirmed, then approved. Every decision is recorded in `approvals/approval_log.md`. Default `approval_mode: manual_all`, even for bumps.

### F. Send Engine (multi-sendbox rotation)

Load Stage 8 (`08_SEND_ENGINE_PROTOCOL.md`) before any send. Sending is `gmail_client.py send` per draft, in code — do not trust playbook prose for the checks.

- **Two auth modes, one interface.** `app_password` (priority for @gmail.com): SMTP send + IMAP read via Python stdlib, no OAuth, no 7-day expiry, preserves our Message-ID. `oauth` (Workspace/custom domain): Gmail API, scopes `gmail.send + gmail.readonly` only; the OAuth app should be Internal to avoid the 7-day refresh-token expiry, else weekly re-auth is a scheduled day-6 `[ACTION REQUIRED]`, not an error path.
- **Rotation is step-1 only; sticky sender thereafter.** First outreach picks the healthy sendbox with the lowest `sent_today/quota_today` ratio (round-robin on ties); `contact.assigned_sendbox` is then fixed. Every bump/reply goes from the assigned box (threading + reply routing + anti-spam require it). A broken box is dropped from step-1 rotation; its pending follow-ups **wait** (never reassigned) plus an `[ACTION REQUIRED]` re-auth, and the report shows "N follow-ups blocked."
- **Ordered pre-send re-check chain** (in this order): `resolve(lead)` → global + client suppression (live: also pull new unsubscribes from the tracker `/events` and the `+unsub` mailbox before any batch; if track-pull failed beyond the window, **block** the box) → `channels.email.status` → **atomic quota reservation** (`reserve(sendbox, day)`, appended under the sent_log lock — no count-then-send race) → warmup cap → two-tier domain cap (`min(remaining_box_quota, remaining_domain_cap)`) → send-window in the recipient's tz → guessed cap + guessed-approval flag → sequence-freeze check (any inbound reply freezes remaining bumps) → step-1 subject lint (reject `^(Re|Fwd):`).
- **Tracking is honest and optional.** Only when the box's mode allows and the campaign enables it: open pixel + rewritten links via the Cloudflare Worker on `trk.{domain}`, with HMAC-signed tokens. On the consumer @gmail.com path, default `plain_text_mode`: **no pixel, no link rewrite, but always keep `List-Unsubscribe` (mailto + https) and the footer opt-out** — the `/u/` path is compliance, not tracking. After send, record `sent/YYYY-MM/sent_log.jsonl`, append activity `email_sent`, and sleep jitter 30–180s. Errors: 429/quota → pause box today; `invalid_grant` → `needs_reauth` + `[ACTION REQUIRED]`; other → draft returns to `approved` with a blocker.

### G. Inbound Sync, Tracking & the CRM Pipeline

Load Stage 10 (`10_FOLLOWUP_REPLY_MANAGEMENT.md`) for inbox sync/follow-up, Stage 12 (`12_TRACKING_ANALYTICS.md`) for metrics, and Stage 13 (`13_CRM_CORE.md`) for objects/rules.

- **Inbound sync (deterministic classifier, in this exact order — the order is load-bearing).** Per sendbox, cursor = `historyId` (OAuth) or IMAP UID (app_password) + `last_successful_sync_ts`. Classify: (1) **DSN/bounce** first (Gmail threads DSNs into the original thread, so DSN must be checked before threadId) → hard/soft, mapped to the original send; (2) `Auto-Submitted: auto-replied`/OOO; (3) **unsub alias** `{box}+unsub-{token}@` → unsubscribe the exact lead; (4) threadId/`In-Reply-To` match sent_log → campaign reply → `reply_untriaged`; (5) from ∈ contacts but no thread → `contact_message`; (6) else personal email → **count only, do not store body, do not deep-read.**
- **Reply freezes the sequence.** The invariant is enforced in code at both draft-time and send-time: any inbound reply freezes the remaining sequence for that contact until triage completes. Then semantic triage of `reply_untriaged` → `positive | question | objection | negative | remove_intent`; `negative`/`remove_intent` (even without the word "unsubscribe") → suppression (or an `[ACTION REQUIRED]` confirm task that blocks further sends).
- **Tracking honesty.** reply/bounce/unsubscribe are exact (IMAP/Gmail + DSN + worker); **open is an estimate** (Gmail image proxy, Apple MPP prefetch, image-blocking) and is labeled "estimated"; click is fairly reliable after bot filtering. **Open/click never alone trigger a stage change or auto-action — only a reply is conversion evidence.** The worker stores only a UA classification (never raw User-Agent), and track-pull accepts only click URLs that match the token's stored `links{}` in sent_log.
- **Deterministic rules engine (`crm_store.py apply-rules`, never improvised by the LLM).** Replies and SLA breaches fire rules whose guard keys `(rule_id, trigger_activity_id)` make them idempotent. E.g. `reply_positive` → create deal (`new_reply`) + "Reply within 4h" task + `freeze_sequence`; `reply_question` → deal (`engaged`) + `freeze_sequence` + `draft_reply_for_approval`; `reply_negative | remove_intent` → `suppress` + `freeze_sequence` + close open tasks; `stage_age_exceeds_sla` → nudge task + flag in report; `deal_won` → lifecycle `customer` + enroll `customers` segment + onboarding task; `hard_bounce | unsubscribe` → `suppress` + close tasks. Every stage change carries an `evidence_activity_id`.
- **CRM objects.** Deals (`crm/deals/{deal_id}.json`) move through a pipeline whose stages carry `probability` + `sla_days`; activities (`crm/activities/YYYY-MM/activities.jsonl`) are the append-only event backbone (each row a monotonic `seq`), and a contact timeline is that log filtered by `contact_id` following merge chains via `resolve()`; tasks (`crm/tasks/tasks.jsonl`) carry a `guard_key` for idempotency. Load Stage 14 for the task engine, SLA sweep, and Today View.

### H. Report & Store

Load Stage 15 (`15_CRM_REPORTING.md`) for reporting and Stage 7 (`07_STORAGE_SCHEMA_AND_HISTORY.md`) for any file create/history write.

- **Two-lane, weekly-only client output.** Operator outputs (approval report, Today View, daily ops, `INTERNAL_REPORT`) are full-detail and never shared. The **weekly** CRM report (Mondays) is the only client-facing output and passes through the Client-Blind Scrub Gate before it is rendered (HTML + PDF) by `tools/report_renderer.py`. Reports label opens "estimated" and report the guessed cohort's bounce rate separately.
- **Storage discipline.** All CRM mutations go through `crm_store.py` (§1). Records are one-file-per-record JSON (logs are monthly JSONL with a monotonic `seq`), atomic via temp+rename, per-collection lock. The adapter is pluggable (JSON default → Postgres later, same contract tests); identity lookups use `find_by_identity` over the unique reverse index (`contact_identities`), not the flat-field query DSL.
- **Notify.** After the run, notify the operator via WideCast `sendNotification` (email + Telegram when connected): counts + the report link → `daily-content-pipeline/notifications/notification_log.md`. A report-ready notification without the link is invalid.

### Daily Run order (reference; per client, pins `target_client_slug`)

1. Load contract + LOAD LEDGER; read the automation manifest + `update_state.json`; take the per-client `run_lock`.
2. Sync inbox across all sendboxes: classify, split personal, suppress bounces/unsubs immediately.
3. Pull tracking from the worker: record open/click activities (bot-filtered).
4. Semantic triage + `apply-rules`: replies → deals/tasks; SLA sweep → nudge tasks; every stage change carries evidence.
5. Follow-up advising (deal-aware): replies → reply drafts; due-silent → value-add bumps → `pending_approval`.
6. Load new pipeline (cold/trigger campaigns, JIT buffer): priority pick → Tier-1 verify → Tier-2 enrich → step-1 draft → `pending_approval`.
7. Send `outbox/approved/` within quota. (Approval happens in chat, any time.)
8. Assisted channels: draft SMS/Messenger for no-email contacts if the campaign allows and consent exists → Today View copy buttons; human sends and reports back → activity.
9. Compile Today View + regenerate kanban.
10. Reports: daily ops + Approval Report + `INTERNAL_REPORT`; Mondays add the scrubbed Weekly CRM Report.
11. Notify Telegram: counts + report link → `notification_log.md`.
12. Stage 9 audit → completion gates → release `run_lock`.

---

## 4. Compliance Backbone (encoded, not just prose)

Compliance is a code-enforced backbone, not a disclaimer. It binds every send-capable path.

- **CAN-SPAM.** Every commercial email carries a real physical mailing address and a working opt-out. Honor opt-out fast (the legal default is 10 business days; OutreachCRM does it same-run). Subjects must be truthful: step-1 subjects must not begin `Re:`/`Fwd:` (linted in Stage 9 audit and at pre-send), and bumps must be real in-thread replies (a truthful `Re:`). `List-Unsubscribe` (mailto + https) plus `List-Unsubscribe-Post: List-Unsubscribe=One-Click` and the footer opt-out ride on every send, including in `plain_text_mode`.
- **Opt-out reach.** Suppression is checked at every send-capable path — initial, follow-up, assisted channels — and at import against ALL identities; it is unioned on merge; pending-merge contacts are excluded from queues. The `GET /u/{token}` confirm page never changes state (scanners fetch GET links); `POST` (one-click, RFC 8058) unsubscribes idempotently. Because the tracker uses a strongly consistent store, unsubscribe events cannot be lost. If a box's track-pull has not succeeded within the window, sending from that box is blocked so worker/mailto unsubs cannot sit unhonored.
- **Guessed email policy.** As in §3C: third-party verification, `catch_all` handling, per-domain kill switch, code-enforced guessed-approval flag + daily cap in `gmail_client.py send`, separate bounce-rate reporting. A guessed address never sends silently.
- **Assisted-channel legality.** Manual send through the Today View reduces automation/platform-detection risk but does NOT change the legality of the solicitation. US SMS is gated on documented consent `{optin_source, optin_at, evidence_activity_id}` or an existing relationship; default SMS is inbound-initiated only. Each assisted draft in the Today View shows its legal basis. Zalo cold-messaging of strangers stays off by default (Vietnam Decree 91/2020 + platform ToS). Assisted sends are recorded as `assisted_sent` activities after the human confirms.
- **No dark patterns.** Never obscure the sender, never fake a prior relationship, never strip the opt-out, never fabricate a `Re:`. The `no_fake_re` guardrail and truthful-subject lint enforce this.

---

## 5. Storage & CRM Mutation Discipline (`crm_store.py`)

- **All CRM mutations go through `crm_store.py`.** Direct file writes under `clients/{slug}/crm/` are a critical violation (the inherited "no one-off scripts" rule). Reading raw JSON is allowed only for debugging. There is no "quick fix by editing the JSON" path; there is only the store. Scope note: the store owns only files under `clients/{slug}/crm/` (contacts, accounts, deals, activities, tasks, `pipelines.json`, `segments.json`, `suppression.jsonl`). Files outside `crm/` — `sendboxes/sendboxes.json`, `campaigns/{slug}/campaign_config.json`, `lists/`, `analytics/`, and the Client Intelligence Profile `.md` — are plain config/profile writes, never crm_store-only (DESIGN §6; §22 R3).
- The store is backed by a pluggable adapter (`tools/storage/adapter.py`): `json_adapter.py` is default, `postgres_adapter.py` comes later and must pass the same parametrized contract tests. Interface includes `get/put/update/delete/query/append/read_log/find_by_identity/reserve`. `query` uses a flat-field `Cond` DSL (`= != < > contains in`); identity lookups do NOT use it — they use `find_by_identity` over the maintained unique reverse index `contact_identities`.
- Every record carries `schema_version`, `id`, `created_at`, `updated_at`; a per-collection `{from_version: fn}` upgrade registry is applied on read and persisted on next write. JSON adapter: one file per record, monthly JSONL logs, atomic temp+rename, per-collection `fcntl` lock, a per-log counter (under the log lock) supplying the monotonic `seq`. Migration to Postgres runs under a storage-freeze flag, verified with per-record content hashes (not counts), upgrading all records to current `schema_version` first.
- Client scope is enforced by rooting the adapter per client; the Postgres adapter carries a mandatory `client_id` in every table and every generated WHERE. Never bypass this to "peek across clients."

---

## 6. Language, Reporting & Human-Communication Rules

- **Human language.** Talk to the human in the human's language. Internal field names, schemas, slugs, and logs are English. Reports, notifications, approval requests, and setup guidance are in the human's language; email copy is in the recipient's language (default English on the priority path).
- **Client-facing = weekly + scrubbed only.** The weekly CRM report is the sole client-facing artifact. It goes through the Client-Blind Scrub Gate and must never mention OutreachCRM internals: sendbox / `gmail_client` / `crm_store`, `trk.` / HMAC / token, sent_log, suppression, warmup, quota, guessed, provider/Telegram/API-key/config details, or `INTERNAL_REPORT`. Operator reports keep all of that.
- **HTML for humans, Markdown for the agent.** Hand the human the mobile-friendly HTML path/link; keep Markdown as the internal source-of-truth/audit record. Do not make the human open Markdown to find the next step.
- **`[ACTION REQUIRED]` everywhere a human must act** (§0.11). One purpose, one exact step, one command or path; at most three per reply; next-action guidance per the Next-Action Guidance Rule when nothing is needed.
- **Progress + next-step question** on every incomplete-workflow reply that hands control back (§0.15), with an `Automation freshness` line once a schedule exists.
- **Honesty gates.** Never fabricate facts, evidence URLs, replies, metrics, or send outcomes. Opens are "estimated." Missing signals are stated, not guessed. When a hard gate blocks an action, surface the blocker in an `[ACTION REQUIRED]` block rather than working around it.

---

## 7. Worked Examples (goal-driven, evidence-grounded)

These illustrate the model across industries. They reframe the inherited industry set from "content ideas" into "compliant, goal-driven cold email + CRM." Every hook shown assumes a real `evidence_url` in the dossier; a hook with no evidence URL does not appear in a draft.

### Real Estate (agent, Austin) — goal `book_meeting`

- **Prospect profile:** active local agents, brokerages, or investor-facing agents in the Austin metro.
- **Evidenced hooks (each needs an `evidence_url`):** a new listing on the agent's site/Zillow; a recent market-view post; a review or award.
- **Draft shape:** short, one time-bound CTA ("Open to a 15-min call Thursday?"), one evidenced personalization line ("Saw your new Mueller listing went up Tuesday"), truthful subject (not `Re:`), footer address + opt-out.
- **CRM wiring:** positive reply → deal at `new_reply` + a 4h reply task; question → `engaged` + drafted reply for approval; "not interested" / "remove me" → suppress + freeze + close tasks.

### Mortgage (loan officer, Texas) — goal `get_reply`

- **Draft shape:** ends with a genuine question, **no link** (the `get_reply` structure). Personalization from a public work post or a rate-commentary the LO published.
- **Follow-up:** each bump carries NEW value (a fresh rate-context line with evidence), never "just following up"; the final step is a breakup.

### DUI / Criminal Defense (attorney, Los Angeles) — goal `book_meeting`

- **Etiquette:** only `public_business` signals (a firm blog post, a bar-association note, a case result the firm published). Anything `personal` is banned from copy and lives only in `do_not_mention`.
- **Compliance:** truthful subject, working opt-out, physical address; assisted SMS only with documented consent.

### Life / P&C Insurance (agency, Miami) — goal `reactivation`

- **Draft shape:** evidence of the prior relationship + "still doing X?"; `reactivation` targets known contacts, so suppression and lifecycle (`customer`, `lost`) are checked first.
- **CRM wiring:** `deal_won` → lifecycle `customer` + `customers` segment + onboarding task.

### SaaS / AI Automation (agency, remote) — goal `direct_sale`

- **Draft shape:** value + exactly one offer link (the only place click tracking is on by default, and only if the box/campaign enables tracking). Everywhere else, `plain_text_mode` with no pixel/rewrite but always the opt-out.
- **Measurement:** report reply/bounce as exact, clicks as fairly-reliable-after-bot-filter, opens as estimated; never advance a deal on an open or click alone.

### Local Services (roofing/HVAC, metro) — no-email contact, assisted channel

- **Flow:** a contact imported with only a name + Facebook URL, enrichment finds no email → assisted Messenger draft appears in the Today View **only if** the campaign allows the channel and consent/legal basis exists; the human sends and reports back → `assisted_sent` activity. Manual send does not change the legality of the solicitation.

In every example: the **goal** decides the structure, the **dossier + evidence URLs** decide the personalization, **suppression + opt-out** gate every send, a **reply** (never an open/click) is the conversion signal that moves the CRM, and **nothing sends without an explicit approve**.

---

## 8. Where the deep mechanics live (do not re-implement here)

Stage 0 is the contract layer. Load the matching stage (with a LOAD LEDGER, dependencies ledgered) before the action:

- Client setup / profile / first-run readiness → Stage 1.
- Sendbox connect/check, auth modes, warmup → Stage 2.
- List import, mapping, dedupe, ULID → Stage 3.
- Verify/enrich, dossier, freshness gate, guessed email → Stage 4 (+ `email-verify-enrich`).
- Campaign definition + goal → Stage 5; email writing + `goal_type` table → Stage 6 (+ `email-writing`).
- Storage schema + history → Stage 7.
- Send engine + pre-send chain + tracking → Stage 8.
- Operations safety audit + completion gates → Stage 9.
- Inbox sync + reply/follow-up management → Stage 10.
- Update & version watch → Stage 11.
- Tracking analytics + learning loop → Stage 12.
- CRM objects/lifecycle/rules/merge → Stage 13; tasks/SLA/Today View → Stage 14; pipeline/forecast/weekly client report → Stage 15.

If any instruction here conflicts with `docs/DESIGN.md`, `docs/DESIGN.md` wins — follow it and report the gap.
