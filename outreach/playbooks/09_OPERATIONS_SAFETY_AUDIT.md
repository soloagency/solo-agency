# OutreachCRM Operations Safety Audit

Stage: `09`

## Load Rule

Load this stage IN FULL before claiming completion of any of: client setup, a daily run, list import, verification/enrichment, campaign creation, drafting, send, inbound sync, follow-up, tracking pull, CRM mutation, reporting, notification, schedule/automation change, or update/upgrade work. Any "done", "complete", "ready", "sent", or hand-back-to-human claim requires the relevant checklist in this file to have been run first.

This file is the audit and completion-gate layer. It does not replace the stage that owns a behavior (Stage 8 owns send mechanics, Stage 13 owns CRM lifecycle, and compliance rules live in the send/import code per DESIGN §16 (compliance encoded in send/import code) — there is no "Stage 16"); it confirms those stages were loaded and their rules honored before you claim the work is finished.

## Hard Gates For This Stage

- Run the relevant checklist before every completion claim. Report missing steps honestly.
- Respect the Setup Flow vs Automation Flow split. Setup Flow **never sends an email, never enriches for send, never runs a campaign, never imports for send**. Its only terminal state is `ready_for_automation_first_run` (or `ready_for_next_automation_run` for a returning client). A send/enrich/campaign-run inside a setup session is a critical workflow violation.
- No side-effect action without a PASS LOAD LEDGER above it. If a required stage was not loaded IN FULL (LOAD LEDGER printed, `Verdict: PASS`, matching `LOAD_MANIFEST.md` when present, dependencies ledgered), load it before proceeding. A file read to 200 of 1900 lines is NOT loaded.
- Every send goes through `gmail_client.py send`, which runs the ordered pre-send re-check chain **in code**. A raw `smtplib`/`imaplib`/Gmail-API send from a one-off script or from playbook prose, bypassing that chain, is a critical workflow violation.
- Every CRM mutation goes through `crm_store.py`. A direct write to any file under `crm/` (contacts, accounts, deals, activities, tasks, pipelines, suppression, identities) is a critical workflow violation. Reading raw JSON for debugging is allowed.
- Nothing leaves without an explicit chat `approve`. A send whose `draft_id` has no matching `approve` decision in `approvals/approval_log.md` is a critical workflow violation. Default `approval_mode: manual_all`, even for bumps.
- Suppression is checked at **every** send-capable path — initial send, follow-up, assisted channels (SMS/Messenger/Zalo), and at import against ALL identities. A send to a suppressed address is a critical workflow violation.
- Every `deal.stage` change carries `evidence_activity_id`. A stage moved without a backing activity is a critical workflow violation.
- Every personalization detail in a draft must trace to a dossier fact with an `evidence_url`. A personalized claim with no evidence is a critical workflow violation.
- Step-1 subjects must not begin `Re:` or `Fwd:`. This is linted in this stage's audit AND enforced in `gmail_client.py` pre-send.
- Only the **weekly client report** is client-facing. It is the only output run through the Client-Blind Scrub Gate. Every other output (approval report, Today View, daily-ops, `INTERNAL_REPORT`) is operator-only and is NOT scrubbed and NOT sent to the client.
- Reports and PDFs are produced by `tools/report_renderer.py` (`render`/`package`) or a reusable approved template. A one-off report/PDF script is a workflow violation unless the exact blocker and an approved exception are logged.
- Before claiming any post-schedule change is complete, verify Automation Resync ran when a schedule/automation already exists. Config-only updates are not enough if a native scheduled-task prompt may still hold an old snapshot.
- Before claiming any update/upgrade/sync-latest work is complete, verify Stage 11 was loaded, GitHub `main` was checked from a verified source, backups/logs were written, clients and automations were resynced, and any `tracker_worker_deploy_required` / `storage_schema_migration_required` human actions were given.

## Latest Override: Setup Flow And Client Isolation Audit

Before claiming **Setup Flow** completion:

- Confirm no send, enrichment-for-send, campaign run, import-for-send, or report of send activity was executed inside the setup chat.
- Confirm the client-specific automation task exists or has been proposed for human review, and its name begins with the client name — for example `AvenNgo - OutreachCRM Daily Run`.
- Confirm the scheduled-run prompt pins `target_client_slug` and cannot touch another client.
- Confirm the agency-wide `OutreachCRM - GitHub Update Watch` task exists or is proposed, and that it is barred from client-facing channels.
- Confirm any setup/config change made after schedule creation triggered Automation Resync.
- Confirm at least one sendbox is connected and verified through Stage 2 (or explicitly recorded as pending), and that `sendboxes/{slug}/credentials.json` and `token.json` are gitignored and `chmod 600`.
- Completion wording must be `ready_for_automation_first_run` or `ready_for_next_automation_run`, never `campaign sent`, `report complete`, or `emails delivered`.

Before claiming **Automation Flow (Daily Run)** completion for a client:

- Confirm the task processed only the target client (`target_client_slug`), and touched no other client's contacts, suppression, sendboxes, quotas, or output.
- Confirm inbox sync ran across all of that client's sendboxes, and that bounces and unsubscribes were suppressed immediately (same run).
- Confirm tracking was pulled from the worker and bot-filtered, and that no open/click alone drove a stage change or auto-action — only a reply is conversion evidence.
- Confirm `apply-rules` ran and every resulting stage change carries `evidence_activity_id`.
- Confirm every send came out of `outbox/approved/` through `gmail_client.py` within quota, with the full in-code pre-send chain, and that every sent `draft_id` has a matching `approve` in `approvals/approval_log.md`.
- Confirm the operator-only outputs exist for the day: `{client}-approval-report.html`, `{client}-today-view.html`, `{client}-daily-ops.html`, `{client}-INTERNAL_REPORT.html`, and `{client}-report_state.json`, each labeled operator-only where applicable.
- On Mondays, confirm the client-facing `{client}-weekly-client-report.html` (+ `.pdf`) was generated through the Client-Blind Scrub Gate and that the scrub grep returned zero internal-term hits.
- Confirm the operator was notified (WideCast `sendTelegramMessage` with email fallback) with counts + report path/link, and that the attempt was logged in `daily-content-pipeline/notifications/notification_log.md`.
- Confirm the per-client `run_lock` was taken and released, and that changes discovered during the run were written back to persistent config and resynced.

Treat these as **critical multi-client isolation violations**:

- A sendbox belonging to client A sends client B's campaign, or a draft in client A's `outbox` is sent from a sendbox registered to client B.
- The agent reads another client's contacts, dossiers, or suppression (any file under `clients/{other_slug}/crm/`) while processing the target client.
- A send occurs without an explicit `approve` for that exact `draft_id`, or with an `approve` that names a different draft.
- A send goes to an address present in `global_suppression.jsonl` or the client's `suppression.jsonl` (checked after `resolve()` follows merge chains).
- A CRM record is written by a direct file write instead of through `crm_store.py`.
- A `deal.stage` (or `stage_history` entry) is moved with no `evidence_activity_id` pointing at a real activity.
- A draft contains a personalization detail (a listing, a post, a review, a company fact, a number) that is not present in the contact's dossier with an `evidence_url`.
- A step-1 subject begins with `Re:` or `Fwd:` (fake-thread deception).
- An automation task processes a client other than its pinned `target_client_slug`, or a client-specific task name does not begin with the client name.
- A client's send is counted against another client's box or domain quota, or one sendbox is shared across two clients.
- Agency-tier suppression (`global_suppression.jsonl`) is treated as proof the client-tier suppression (`clients/{slug}/crm/suppression.jsonl`) was checked, without checking both.
- The weekly client report leaks any internal term (scrub-gate failure), or a client-facing artifact is produced by a one-off script instead of `tools/report_renderer.py`.

## Source Preservation Rule

This file is detailed source material for the audit and completion-gate layer. Do not summarize away requirements, examples, checklists, schemas, protocols, gate chains, edge cases, warnings, approval gates, or completion gates. A downstream agent may shorten its human-facing summary, but it must still obey the full requirements in this file.

---

## Automation Resync Safety Check

Run this check before saying any setup repair, sendbox change, provider/notification change, campaign edit, schedule change, or client/profile update is complete **when a schedule/automation already exists**.

Ask:

1. Did this change happen after `daily-content-pipeline/schedule.md` or a native automation/scheduled task was created?
2. Would tomorrow's scheduled run need to know about this change?
3. Could the scheduled task have an old prompt snapshot, old sendbox/quota state, old suppression state, old campaign contract, old approval mode, or old provider/notification config?

If yes to any of those, load the Automation Scheduling / Resync protocol (`playbooks/AUTOMATION_SCHEDULING.md`) and perform Automation Resync before claiming completion.

Minimum resync audit:

- Client profile (`client_profile_{client_slug}_{business_slug}_{location_slug}.md`) updated.
- Sendbox registry (`sendboxes/sendboxes.json`) and any campaign `campaign_config.json` updated when send/quota/sequence config changed.
- Suppression state reconciled when contacts opted out or bounced (client `suppression.jsonl`, and `daily-content-pipeline/suppression/global_suppression.jsonl` for agency-tier entries).
- `daily-content-pipeline/provider_defaults.json` updated when the WideCast notification catalog/discovery defaults changed.
- The client's `integrations/providers/` config, OpenAPI cache, capabilities, health, and provider call log updated when the notification provider config changed.
- `daily-content-pipeline/schedule.md` updated.
- `daily-content-pipeline/automation/automation_manifest.md` updated.
- `daily-content-pipeline/automation/scheduled_run_prompt.md` updated.
- The actual native scheduled-task prompt updated when accessible, or `automation_prompt_update_pending` clearly logged when not accessible.
- `daily-content-pipeline/automation/github_issues.md` updated when a tracked GitHub issue, maintainer/community response, or issue-derived workaround affects future runs.
- `daily-content-pipeline/automation/update_state.json` and `daily-content-pipeline/automation/update_log.md` updated when an update check or applied update affects future runs.
- `daily-content-pipeline/automation/resync_log.md` updated.
- Dry-read verification performed from the scheduled-run entrypoint and the latest local files. The dry read must enumerate and confirm these 8 sources: the scheduled entrypoint (`playbooks/SCHEDULED_RUN_ENTRYPOINT.md`), `daily-content-pipeline/automation/automation_manifest.md`, `daily-content-pipeline/automation/github_issues.md`, `daily-content-pipeline/automation/update_state.json`, `daily-content-pipeline/provider_defaults.json` (when relevant), `daily-content-pipeline/schedule.md`, the client profile, and `sendboxes/sendboxes.json`.
- The human-facing progress block includes an `Automation freshness check` that answers whether the latest changes were synced into the automation/scheduled-task prompt/contract/source state, not only config, and whether tomorrow's run will read the newest state.

Completion wording must distinguish full vs partial sync:

```text
Automation Resync complete: the next scheduled run will read the latest approved config/campaign/suppression state.
```

or:

```text
Automation Resync partially complete: local files are updated, but the native scheduled-task prompt still needs a human update.
```

Do not say:

```text
Config updated, so automation is done.
```

That is a safety-audit failure.

---

## Update And Version Watch Safety Check

Run this check before saying an update command, update-watch run, setup-repair update, or stale-version recovery is complete.

Minimum update audit:

- Stage 11 `playbooks/11_UPDATE_AND_VERSION_WATCH.md` was loaded IN FULL.
- The source was GitHub `main` from the Solo Agency repo — OutreachCRM's `outreach/` module (`https://github.com/soloagency/solo-agency`).
- The agent used the current verified setup root or a fresh unique `mktemp -d` checkout. No fixed shared fallback folder (such as `/tmp/outreachcrm`, `/var/tmp/outreachcrm`, `/dev/shm/outreachcrm`) was used.
- `.git`, `origin`, local `HEAD`, and remote `refs/heads/main` were verified on the parent checkout (the `outreach/` module has no `.git` of its own) before reading or copying source files.
- The update check covered: root instructions, all playbooks, provider/OpenAPI tooling (`tools/provider_openapi.py`), the report renderer (`tools/report_renderer.py`), the storage adapter + `schema_version`, `crm_store.py` / `gmail_client.py` / `import_leads.py` / `email_verify.py`, `tracker/worker.js` (+ its `wrangler deploy` rerun step), sendbox token compatibility, the deploy script, and skills.
- `daily-content-pipeline/automation/update_state.json` and `update_log.md` were created or updated. The state enum uses `tracker_worker_deploy_required` and `storage_schema_migration_required`.
- Runtime files/folders replaced by the update were backed up under `daily-content-pipeline/automation/backups/update_YYYY-MM-DD_HHMMSS/` or an equivalent logged backup path.
- Secrets and local client state were preserved (merge config, never overwrite secrets/history): provider API keys, `provider_config.local.json`, `secrets/`, sendbox `credentials.json`/`token.json`, the tracker key, client profiles, suppression, CRM data (contacts/accounts/deals/activities/tasks), approvals, sent logs, reports, outputs, and analytics.
- Every active/configured client was checked for required `schema_version` / template updates. If any collection needs a migration, `storage_schema_migration_required` was set and the human received the exact `crm_store.py migrate` step (run under the storage freeze flag, verified by per-record content hashes).
- If `tracker/worker.js` changed, `tracker_worker_deploy_required` was set and the human received the exact `wrangler deploy` command to run outside the AI sandbox.
- Automation Resync was performed when a schedule/automation exists. If native scheduled-task prompts could not be edited directly, `automation_prompt_update_pending` was logged with the exact replacement prompt path/action.
- The human-facing completion states `update complete`, `update partially complete`, or `update blocked`, with the exact remaining action.

Treat these as critical workflow violations:

- Saying "updated" after only pulling local code but not resyncing client automation tasks.
- Copying from an unverified stale folder because GitHub access failed.
- Updating playbooks but leaving scheduled-prompt snapshots on the old behavior.
- Updating `tracker/worker.js` without setting `tracker_worker_deploy_required` and telling the human to rerun `wrangler deploy`.
- Applying a storage-schema migration without the storage freeze flag or without per-record content-hash verification.
- Overwriting `provider_config.local.json`, API keys, `secrets/`, sendbox tokens, suppression, CRM data, reports, history, or outputs during an update.
- Running a send, enrichment, campaign, or report merely because the human said `update`.

---

## Completion Gates

Run the gate that matches what you are about to claim. Each gate is a mechanical checklist: every line must be satisfied, or the miss must be reported honestly (see Final Hard Gate).

> **Phase note (read these gates against DESIGN §22).** The Phase-1 runtime tools (`crm_store.py`, `gmail_client.py`, `import_leads.py`, `email_verify.py`, storage adapter) **exist** — the direct-`crm/`-write critical-violation gate and the real send/sync/enrich gates are in FULL force; a "tool_not_built" skip for these is itself a finding, not an excuse. The only still-active degradation for the core loop is DESIGN §22 R1: a `status: planned` Phase-2 stage file (04/05/06/10/12–15) is not a missing-file/recovery trigger — load the covering `docs/DESIGN.md` section with its own ledger and record `stage_file_pending`, never a GitHub re-fetch or Last-Resort Recovery. One migration exception (DESIGN §22 R3): a workspace carried over from an older Phase-0 install may still contain `phase0_direct_write`-logged records; that is compliant only until `python3 tools/crm_store.py --client-dir <DIR> validate --rebuild-index` has run (validates the records and rebuilds the identity index). After that migration, every CRM write must go through `crm_store.py`.

### Setup completion gate

Claim `ready_for_automation_first_run` only when:

1. No email was sent, no contact was enriched for send, no campaign was run, and no list was imported for send inside the setup session.
2. The client profile exists at `clients/{client_slug}/{business_slug}_{location_slug}/client_profile_{client_slug}_{business_slug}_{location_slug}.md` and all slugs obey the slug rules (lowercase, hyphens, no punctuation).
3. The client folder tree exists (`crm/`, `lists/`, `campaigns/`, `sendboxes/`, `approvals/`, `analytics/`, `inbox_sync/`, `reports/`, `outputs/`, `integrations/providers/`), and monthly folders use the `YYYY-MM/` convention.
4. At least one sendbox is connected and verified through Stage 2, or explicitly recorded as pending; `credentials.json` / `token.json` are gitignored and `chmod 600`; `sendboxes.json` records `auth_mode`, `quota_today`, `warmup_stage`, and `status`.
5. The client row is added to `daily-content-pipeline/clients_index.md`.
6. The recurring schedule/cadence is configured or explicitly marked manual-only, and `daily-content-pipeline/schedule.md` records it plus the notification channel.
7. The client-specific automation task exists or is proposed, its name begins with the client name, and its prompt pins `target_client_slug` and cannot touch another client.
8. The agency-wide `OutreachCRM - GitHub Update Watch` task exists or is proposed and is barred from client-facing channels.
9. Any post-schedule config change triggered Automation Resync (see the Automation Resync Safety Check).
10. The setup handoff tells the human the exact automation task name to run for the first run, directly in chat, inside a `**[ACTION REQUIRED]**` block. Markdown-only handoffs are a failure.
11. The completion wording is `ready_for_automation_first_run` (or `ready_for_next_automation_run`), never a send/report claim.

### Daily Run completion gate

Claim a daily run complete only when, for every active client processed (or explicitly logged as skipped):

1. The per-client `run_lock` was taken at the start and released at the end.
2. The contract + LOAD LEDGER were loaded; the automation manifest and `update_state.json` were read (Update Watch runs as a separate task and does not touch clients).
3. Inbox was synced across all sendboxes using the deterministic classifier in its exact order (DSN/bounce → auto-reply/OOO → unsub alias → threadId/`In-Reply-To` → contact-no-thread → personal-count-only); bounces and unsubscribes were suppressed immediately.
4. Tracking was pulled from the worker and bot-filtered; opens are labeled estimated; no open/click alone triggered a stage change or auto-action.
5. Semantic triage + `apply-rules` ran; replies became deals/tasks; the SLA sweep created nudge tasks; every stage change carries `evidence_activity_id`; guard keys made `apply-rules` idempotent.
6. Any inbound reply froze the remaining sequence for that contact until triage completed (enforced at draft-time and send-time).
7. Follow-up advising and new-pipeline drafting produced drafts into `outbox/pending_approval/YYYY-MM-DD/`; nothing was auto-approved.
8. Sends came only from `outbox/approved/` through `gmail_client.py`, within quota, with the full pre-send chain, jittered 30–180s; each sent `draft_id` has a matching `approve`.
9. Assisted-channel drafts (SMS/Messenger/Zalo) were produced only where the campaign allows AND documented consent exists; each Today-View draft shows its legal basis; the human sends and reports back → `assisted_sent` activity.
10. The Today View and kanban were regenerated by `tools/report_renderer.py`.
11. Operator-only reports were produced: `{client}-daily-ops.html`, `{client}-approval-report.html`, `{client}-INTERNAL_REPORT.html`, `{client}-report_state.json`. On Mondays, the client-facing `{client}-weekly-client-report.html` (+ `.pdf`) was produced through the Client-Blind Scrub Gate.
12. The operator was notified via WideCast `sendTelegramMessage` (email fallback) with counts + report path/link; the attempt was logged in `daily-content-pipeline/notifications/notification_log.md`.
13. Every active client was processed or explicitly logged as skipped; this Stage 9 audit + completion gates ran before the `run_lock` was released.

### Drafting completion gate

Claim a drafting pass complete only when, for every draft:

1. The draft is assembled from four inputs: client profile (voice, offer, compliance) + campaign goal (objective, offer, CTA, proof points) + contact dossier (hooks + evidence) + step intent. A bump carries NEW value, never "just following up".
2. Every personalization detail in the body traces to a dossier fact that carries an `evidence_url`. Details with no evidence are removed or the draft drops to the no-hook fallback.
3. The freshness gate passed: for a step-1 draft, hooks are within TTL (else known URLs were refreshed); a bump drew on RESERVED dossier points + the campaign `message_bank` (no per-bump re-enrichment — micro-refresh only opportunistically), and the stale-hook guard held (any time-sensitive hook past TTL was re-verified or dropped; a sold listing is not referenced as active).
3b. House Style held: the draft contains zero em dashes (`—`). If the campaign declares `goal.companion_doc`, the body embeds a real produced URL (or the lead was handled per `on_fail`); if it declares `goal.message_bank`, the draft weaves 1–2 bank messages not used by this lead's earlier touches (menu, not checklist — no message the lead's data contradicts).
4. The step-1 subject does not begin `Re:` or `Fwd:`. Continuation steps use a truthful `Re:` on a real in-thread reply only.
5. No banned claim from `guardrails.banned_claims` (e.g. guarantees) appears; `no_fake_re` is honored.
6. Each draft landed in `outbox/pending_approval/`; nothing was moved to `approved` without a chat `approve`.
7. The Approval Report (`{client}-approval-report.html`, operator-only, NOT scrubbed) was rendered by `tools/report_renderer.py`, splitting drafts into **High confidence** (verified email, ≥0.7 hook) and **Review carefully** (weak hook, guessed email, fallback opener), with one card per lead showing clickable evidence URLs, the editable body, and warning flags.
8. A Telegram note ("N drafts awaiting review" + path) was sent.

### Send completion gate

Claim a send complete only when, for every message sent:

1. It was sent by `gmail_client.py send`, not by a one-off script and not from playbook prose.
2. The ordered pre-send re-check chain ran **in code**, in this exact order, and every gate passed (see The Ordered Pre-Send Gate Chain below).
3. The `draft_id` was in `outbox/approved/` and has a matching `approve` decision in `approvals/approval_log.md` naming that exact draft.
4. The recipient address is not in `global_suppression.jsonl` or the client's `suppression.jsonl` (checked after `resolve()`).
5. The sticky sender was honored: `contact.assigned_sendbox` was used for every bump/reply; rotation (lowest `sent_today/quota_today`, round-robin on ties) applied to step-1 only, and `assigned_sendbox` was then fixed.
6. `List-Unsubscribe` (mailto + https) and the footer opt-out are present, including in `plain_text_mode` (no pixel, no link rewrite, but `/u/` and the footer always stay).
7. `sent/YYYY-MM/sent_log.jsonl` was appended with `{lead_id, campaign, step, sendbox, provider_id, thread_id, rfc_message_id, token, links, sent_at, seq}`, and an `email_sent` activity was appended; jitter 30–180s was applied.
8. Errors were handled: 429/quota → pause box today; `invalid_grant` → `needs_reauth` + `**[ACTION REQUIRED]**`; other → draft returned to `approved` with a blocker (never silently dropped).
9. Guessed/unverified addresses were sent only with the explicit guessed-approval flag on the draft and within the daily guessed-send cap read from `sent_log`; `catch_all` addresses were excluded from the guessed quota or capped ~2%.

---

## The Ordered Pre-Send Gate Chain

`gmail_client.py send` must run these gates in this exact order for every message (this is the
Phase-1 chain actually in code — Stage 8 §3 is the source): the audit confirms the chain ran in
code (not narrated in prose) and that any block halted the send. Order is load-bearing.

1. **resolve(lead)** — follow merge chains so suppression/quota/channel checks hit the survivor record.
2. **Suppression** — client (`crm/suppression.jsonl`) + agency (`global_suppression.jsonl`), across the contact's identities AND the actual recipient address, before any quota is reserved.
3. **channels.email.status** — must be usable (not `opted_out` or `bounced`).
4. **CAN-SPAM sending identity (gate 2b)** — `config/sending_identity.json` must exist with a physical mailing address; the engine appends the compliance footer to every body. Fail closed: `missing_physical_address`.
5. **Guessed-email approval** — a `guessed_only` address sends only with the per-draft `guessed_approved` flag.
6. **Sequence-freeze check** — any inbound reply (and any unsubscribe/hard bounce) freezes the remaining bumps for that contact until triage clears it.
7. **Step-1 subject lint** — reject `^(Re|Fwd):` on step-1 subjects.
8. **Atomic quota reservation (last)** — `store.reserve(sendbox, day, cap)` under the lock; a draft blocked by any earlier gate never reserves. A failed real send releases its reservation.

If any of these is enforced only in playbook prose and not in `gmail_client.py`, that is a critical violation — record it and block the claim. **Phase-2/3 gates (tracker worker) — NOT in Phase-1 code; do not assert them and do not flag their absence as a violation:** live tracker `/events` + `+unsub` pull with block-the-box staleness, warmup-stage cap, two-tier domain cap, send-window (recipient tz), guessed cohort ≤10%/day/box + per-domain kill switch. A failed send is never silent: the blocker is persisted on the draft record (terminal blockers flip it to `status: blocked`; transient ones keep `approved` + `blocker`/`blocked_at` for natural retry).

---

## Compliance Audit

Compliance is encoded in the send and import code, not just described here. The audit confirms the encoding held.

### CAN-SPAM

- **Physical address + working opt-out** in every commercial email. Confirm the footer opt-out and a valid `List-Unsubscribe` header (`<mailto:{box}+unsub-{token}@...>, <https://trk.{domain}/u/{token}>`) plus `List-Unsubscribe-Post: List-Unsubscribe=One-Click` are present, including in `plain_text_mode` — `/u/` is compliance, not tracking.
- **Truthful subjects.** Step-1 subjects must not begin `Re:`/`Fwd:` (linted here and in pre-send). Bumps must be real in-thread replies with a truthful `Re:`.
- **Opt-out honored.** Default legal window is 10 business days; OutreachCRM honors it **same-run**. Confirm every opt-out (footer link, `POST /u/`, `+unsub` mailbox, or a `negative`/`remove_intent` reply even without the word "unsubscribe") produced an immediate suppression entry and closed the contact's open tasks.

### Opt-out reach

- Suppression is checked at **every** send-capable path: initial send, follow-up, assisted channels, and at import against ALL identities (emails, phones, canonical socials).
- On merge, suppression and channel statuses are **unioned** into the survivor; a contact with a pending merge proposal is excluded from every campaign queue until resolved.
- If track-pull has not succeeded within N hours, sending for that box is **blocked**, so worker/mailto unsubscribes cannot sit unhonored past the window.

### Guessed email

- Guessed/unverified addresses go through the third-party verification API (called from local Python) before any send.
- `catch_all` domains are excluded from the guessed quota or capped ~2% (an MX check is near-meaningless there — catch-alls accept any RCPT).
- **Per-domain kill switch:** the first hard bounce on a guessed pattern at domain X suppresses all other guessed addresses at X.
- `guessed_only` status is enforced **in `gmail_client.py send`** (requires the explicit per-draft guessed-approval flag + a daily guessed-send cap read from `sent_log`), never only in prose.
- The guessed cohort's bounce rate is reported separately in the operator report.

### Assisted channels

- Manual send reduces automation/platform-detection risk but does NOT change the legality of the solicitation.
- **US SMS** is gated on documented consent `{optin_source, optin_at, evidence_activity_id}` or an existing relationship; default SMS is inbound-initiated only. Each assisted draft in the Today View shows its legal basis.
- **Zalo** cold-messaging of strangers stays off by default (Vietnam Decree 91/2020 + platform ToS).

### Regulated / sensitive industries

For legal, healthcare, finance, mortgage, insurance, tax, immigration, investment, and employment offers, the copy audit must confirm: no unsupported claims; no guaranteed outcomes; disclaimers where appropriate; no personalized legal/medical/financial/tax advice unless the client is qualified and the copy frames it safely; no fear-based manipulation beyond reasonable urgency; no exploitation of tragedy or private personal information.

Unsafe → safer reframes:

- "We guarantee your case will be dismissed." → "Depending on the facts, there may be options."
- "You will definitely receive compensation." → "Do not assume the first offer is the final answer."
- "This investment will make you money." → "Rules vary by situation; talk to a qualified professional first."

---

## Preview & Chat-Approval Gate Audit

The Approval Report is the gate before any send. Confirm:

- The Approval Report was rendered to `outputs/.../{client}-approval-report.html` (operator-only, NOT scrubbed) by `tools/report_renderer.py`, reusing its contenteditable + Copy-button blocks.
- Header split into High confidence vs Review carefully; one card per lead with `#id`, name/company/email + verify status, hooks with clickable evidence URLs, subject + editable body + warning flags (guessed, generic, bump step).
- Chat is the write path; editing the HTML does not persist. Only the chat grammar approves: `approve all` / `approve 1-20, 35, 41` / `reject 7: reason` / `edit 12: ...` / `hold 5`.
- Approved → `outbox/approved/` → sent in-session within quota with the full in-code re-check chain. Rejected → logged with reason → reason feeds `analytics/learning_log.md`. Edit → the agent patches, re-confirms, then approves.
- Every decision is written to `approvals/approval_log.md`. Nothing left without an explicit `approve`. Default `approval_mode: manual_all`, even for bumps.

---

## CRM Integrity Audit

Confirm:

- Every mutation to `crm/` went through `crm_store.py` (`put`/`update`/`append`/`apply-rules`/merge). No direct file write to any CRM collection. Reading raw JSON for debugging is allowed.
- Every record carries `schema_version`, `id`, `created_at`, `updated_at`; the adapter applied any on-read upgrades and persisted them on next write.
- Activities are append-only with a monotonic `seq`; a contact timeline = filter activities by `contact_id` following merge chains via `resolve()`.
- Deal stages come from `pipelines.json`; a blocked invalid stage transition stayed blocked; each `stage_history` entry has `at`, `by`, and `evidence_activity_id`.
- Rules are deterministic (`crm_store.py apply-rules`), never improvised by the model; guard keys `(rule_id, trigger_activity_id)` kept `apply-rules` idempotent/re-runnable.
- Merge semantics: auto-merge only on exact email / E.164 phone / canonical social URL; fuzzy name+company was proposed for human approval; the losing record became a permanent tombstone `{merge:{status:"merged", merged_into:A}}` (never deleted); identities, channel statuses, and suppression were unioned into the survivor.
- Every `lead_id` lookup path (sync classifier, track-pull, unsub handler, `apply-rules`, drafting) called `resolve(lead_id)`; pending-merge contacts were excluded from every campaign queue.

---

## Enrichment & Evidence Audit

Confirm:

- The dossier belongs to the contact (client-scope); campaigns reference `lead_id`; the enrich queue is deduped by `lead_id`.
- TTLs were respected: durable identity/context ~90d (inherited as-is by other campaigns); fresh hooks 7–14d (other campaigns run a cheap refresh, not full re-discovery); negative cache inherited (`email_not_found` retried after 30d then stopped; `no_verifiable_hook` respected).
- Cross-campaign hook reuse blocked: a second campaign did not open with a hook already `used_in` on that person; a contact in an active sequence of campaign A was not drafted by B (`min_days_between_touches_across_campaigns`).
- Etiquette hard rule held: `public_business` signals (listings, work posts, reviews, awards, market opinions) were fair game; `personal` signals (family, health, vacations, children) were default-banned from copy and went only into `do_not_mention`.
- Channel-reality honesty: logged-out-unreadable sources (Facebook, LinkedIn) were stored as URL only, not fabricated into readable content.
- Every hook carries an `evidence_url`; the write-time freshness gate invalidated stale hooks. The mechanical draft-vs-dossier check (below) confirms no personalization detail lacks evidence.

---

## Reporting & Delivery Audit

Two-lane reporting: operator-only (`INTERNAL_REPORT`, full detail) vs client-facing (through the Client-Blind Scrub Gate). **Only the weekly client report is client-facing.**

- No one-off report/PDF scripts. Client-facing HTML/PDF is produced by `tools/report_renderer.py` (`render` / `package`) or a reusable approved template; the report-design skill (`playbooks/skills/report-design/SKILL.md`) was loaded before report generation/repair. A new one-off Python/browser/PDF script instead of using or fixing the reusable renderer is a workflow violation unless the exact blocker and an approved exception are logged.
- Standalone/portable HTML: no `fetch("./report.md")`, no remote scripts/CSS/fonts/images; embed assets. Wide tables are inside a scroll wrapper or stacked into cards; the document has no horizontal overflow at a 390px viewport.
- Every operator/human-facing report link points to `.html` (or the uploaded URL), never `.md`.
- `{client}-INTERNAL_REPORT.html` holds all internal detail (sendbox status, quota/warmup, suppression counts, guessed-cohort bounce rate, provider/notification status, blockers, debug) and is labeled `INTERNAL_REPORT - Not for client sharing`. `outputs/latest/` copies are refreshed.

### Client-Blind Scrub Gate (weekly client report only)

The weekly client report is the sole scrubbed, client-facing output. Run the scrub gate on `{client}-weekly-client-report.html` (and the `.pdf`) only. The operator-only outputs are never scrubbed and never sent to the client.

Confirm the extracted text of the weekly client report contains **none** of the internal terms in `tools/report_renderer.py` `CLIENT_BLIND_TERMS` (per DESIGN §19 this prose list must enumerate exactly that set): `OutreachCRM`, `WideCast`, `Telegram`, `MCP`, `OpenAPI`, `automation`, `scheduled task`, `API key`, `config file`, `debug`, `agent debug`, `PDNA`, `provider_config`, `Client tools`, `global MCP`, `sendbox`, `gmail_client`, `crm_store`, `storage_config`, `trk.`, `HMAC`, `token.json`, `sent_log`, `suppression`, `warmup`, `quota`, `guessed`, `INTERNAL_REPORT`. The mechanical scrub gate (see Evidence-Based Audit Requirements) does not re-list these — it reads the renderer's own scrub result so there is one authoritative term source.

The weekly client report communicates results in the client's language: pipeline movement, replies and meetings booked, deals created/advanced, and recommended next actions — never sendbox mechanics, suppression counts, quota/warmup, guessed cohorts, provider tooling, or agent internals.

---

## Notification Audit

WideCast is the operator notification provider only.

- Notification is sent via WideCast `sendTelegramMessage` with an email fallback; an optional `uploadAsset` may host the report `.html` so the operator receives a URL. Provider-hosted URLs are operator handoff links, not client-share links.
- Provider discovery is via `tools/provider_openapi.py` against the client's `integrations/providers/provider_config.local.json`. That file references the secret with `api_key_env` (environment variable name) or `api_key_local` (local path) — never a field literally named `api_key`.
- WideCast is used for notification only. There is no WideCast production, publishing, or analytics call.
- Every report-ready notification includes: run status, clients processed, blockers, counts (sends, replies, bounces, unsubscribes, deals created/advanced, tasks due), and the report path/link + `INTERNAL_REPORT` path/status. A bare "report ready" with no path/link is invalid — send a correction if that happens.
- If WideCast notification is unavailable, fall back to Gmail/email if connected; if neither is available, record `notification_channel: local_path_only` and give the human the exact path plus how to connect a channel. Log every attempt in `daily-content-pipeline/notifications/notification_log.md`.

---

## Daily Run Order (per client, pins `target_client_slug`)

For reference during the Daily Run completion gate. The scheduled-run entrypoint owns the canonical procedure; this is the audit's model of it.

1. Load contract + LOAD LEDGER; read the automation manifest + `update_state.json` (Update Watch is a separate task and does not touch clients); take the per-client `run_lock`.
2. **Sync inbox** across all sendboxes: classify in the deterministic order, split personal (count only, no body stored), suppress bounces/unsubs immediately.
3. **Pull tracking** from the worker: record open/click activities, bot-filtered; opens labeled estimated.
4. **Semantic triage + `apply-rules`**: replies → deals/tasks; SLA sweep → nudge tasks; every stage change carries evidence.
5. **Follow-up advising (deal-aware):** replies → reply drafts; due-silent → value-add bumps → `pending_approval`.
6. **Load new pipeline** (cold/trigger campaigns, JIT buffer 3–7 days): priority pick → Tier-1 verify → Tier-2 enrich → step-1 draft → `pending_approval`.
7. **Send** `outbox/approved/` within quota (approval happens in chat, any time).
8. **Assisted channels:** draft SMS/Messenger for no-email contacts where the campaign allows + consent exists → Today View copy buttons; human sends, reports back → activity.
9. **Compile Today View + regenerate kanban** (renderer).
10. **Reports:** daily-ops + Approval Report + `INTERNAL_REPORT`; **Mondays** add the weekly client report through the scrub gate.
11. **Notify** via WideCast `sendTelegramMessage` (email fallback): counts + report link → `notification_log.md`.
12. **Stage 9 audit** → completion gates → release `run_lock`.

---

## Completion Criteria

### Initial setup is complete when

1. The root data tree (`daily-content-pipeline/` with `clients_index.md`, `schedule.md`, `clients/`, `outputs/`, `automation/`, `suppression/`, `notifications/`) exists.
2. Each configured client has a pipeline folder and a client profile file at the correct path with valid slugs.
3. Each configured client has at least one sendbox connected/verified or explicitly pending, with secrets gitignored and `chmod 600`.
4. Inferred/researched setup context was shown to the human step by step; human corrections were applied.
5. The recurring schedule/cadence was configured or explicitly marked manual-only, and recorded in `schedule.md` with the notification channel.
6. The client-specific automation task exists or is proposed; its name begins with the client name; its prompt pins `target_client_slug` and the Setup Flow / Automation Flow contract.
7. The agency-wide `OutreachCRM - GitHub Update Watch` task exists or is proposed and is barred from client-facing channels.
8. Any post-schedule config change triggered Automation Resync.
9. The exact automation task name to run for the first run was shown directly in chat inside a `**[ACTION REQUIRED]**` block. Markdown-only handoffs are a failure.
10. Completion wording is `ready_for_automation_first_run` or `ready_for_next_automation_run` — no send/report claim.

### Recurring schedule setup is complete when

1. `schedule.md` exists and records the cadence (daily, multiple-times-daily, weekly, manual-only, first-run-only, or other), chosen after the profile and sendbox plan are known and before the automation task is marked ready.
2. The schedule/manual-run process is documented.
3. The client-specific automation task name begins with the client name and its prompt pins `target_client_slug`.
4. The configured notification channel is documented.

### A daily run is complete when

1. Every active client was processed or explicitly logged as skipped.
2. Inbox sync ran across all sendboxes; bounces/unsubs were suppressed immediately.
3. Tracking was pulled and bot-filtered; no open/click alone drove a stage change.
4. Semantic triage + `apply-rules` ran; replies → deals/tasks; SLA sweep → nudge tasks; every stage change carries `evidence_activity_id`.
5. Follow-up and new-pipeline drafts were produced into `pending_approval`; nothing was auto-approved.
6. Sends came only from `outbox/approved/` through `gmail_client.py` within quota with the full pre-send chain; each sent `draft_id` has a matching `approve`.
7. Assisted-channel drafts were produced only with consent and a shown legal basis; the human sent and reported back → `assisted_sent` activity.
8. Operator-only reports exist for the day; on Mondays the weekly client report exists and passed the scrub gate.
9. The `{client}-report_state.json` file is created/updated with reconciled counts/statuses that match the report set.
10. `outputs/latest/` copies were refreshed for the operator outputs (and the weekly report on Mondays).
11. The operator was notified with counts + report path/link + `INTERNAL_REPORT` status; the attempt was logged in `notification_log.md`.
12. Reports are written in the correct language: the weekly client report in the client's language, operator notifications/`INTERNAL_REPORT` in the operator's language.
13. This Stage 9 audit + completion gates ran before the `run_lock` was released.

### A drafting pass is complete when

1. Every draft = client profile + campaign goal + contact dossier + step intent; bumps carry NEW value.
2. Every personalization detail traces to a dossier `evidence_url` (mechanical check passed).
3. The freshness gate passed (step-1 hooks in TTL; bumps from reserved points + `message_bank`, no per-bump re-enrichment; stale-hook guard held). House Style: zero em dashes; companion-doc URL and bank rotation honored when declared.
4. No step-1 subject begins `Re:`/`Fwd:`; no banned claim appears; `no_fake_re` honored.
5. All drafts landed in `pending_approval`; the Approval Report was rendered with the confidence split and clickable evidence URLs; a Telegram note was sent.

### A send is complete when

1. Every message went through `gmail_client.py send` with the full ordered pre-send chain in code.
2. Every sent `draft_id` was in `outbox/approved/` with a matching `approve`.
3. No message went to a suppressed address; the sticky sender was honored.
4. `List-Unsubscribe` + footer opt-out are present (including plain-text mode).
5. `sent_log` + `email_sent` activity were written; jitter applied; errors handled (429 pause, `invalid_grant` reauth, other → back to `approved` with a blocker).
6. Guessed sends had the guessed-approval flag and stayed within the daily guessed cap; `catch_all` excluded/capped.

---

## Final Agent Self-Audit Checklist

Use this before replying to the human, before claiming setup complete, and before claiming a daily run / drafting pass / send complete. The playbook is intentionally comprehensive; long instructions are easy to partially miss. Actively check for omissions instead of relying on memory.

### Response Self-Audit Checklist

- [ ] Did I answer in the same language the human used?
- [ ] Did I explain internal terms in plain language when they appear in human-facing text (sendbox, suppression, warmup, quota, guessed, dossier, pipeline stage, approval report)?
- [ ] Did I avoid asking for information I can infer, research, or discover myself?
- [ ] If I asked a question, did I first show what I inferred from the previous answer?
- [ ] If a workflow is not complete, did I show an updated progress block in this reply?
- [ ] If schedule/automation already exists and this reply includes a progress block, did I include an `Automation freshness check` (synced into the automation prompt/contract/source state, not only config; will tomorrow's run read the newest state)?
- [ ] If handing control back while required steps remain, is the final line exactly one concrete next-step question?
- [ ] If human action is needed, did I show the exact action directly in chat inside a `**[ACTION REQUIRED]**` block, not buried in paragraphs or a file link?
- [ ] If no human action is needed, did I say `No action required right now.` instead of ending ambiguously?
- [ ] Did I keep the most important action at the end, with no more than three `**[ACTION REQUIRED]**` blocks, and never an icon as the only signal?
- [ ] Did I avoid telling the human to open a Markdown file for instructions?
- [ ] If I mentioned a report, did I provide the `.html` path/link (never `.md`) and, for the weekly report, confirm the scrub gate?
- [ ] Before reporting a blocker, repeated failure, contradiction, stale artifact, or dead end, did I run Last-Resort Recovery — check GitHub `main` for newer OutreachCRM playbooks/code and reload the latest relevant instructions?
- [ ] If the latest GitHub version still did not resolve the blocker, did I record it in `daily-content-pipeline/automation/github_issues.md` and tell the human how it is tracked, without requiring them to have a GitHub account?
- [ ] Did I avoid asking for credentials, cookies, passwords, OTPs, or tokens?
- [ ] Did I mention blockers clearly, with the next action if any?

### Client Setup Self-Audit Checklist

- [ ] Did I keep Setup Flow free of any send, enrichment-for-send, campaign run, or import-for-send?
- [ ] Did I infer industry/sub-industry/audience/offer and show them before saving?
- [ ] Did I save the client profile at the correct path with valid slugs, only after showing the summary?
- [ ] Did I connect/verify at least one sendbox (or record it pending), with `credentials.json`/`token.json` gitignored and `chmod 600`?
- [ ] For an `app_password` box, did I confirm 2FA + App Password; for an `oauth` box, did I keep scopes to `gmail.send + gmail.readonly` and prefer an Internal OAuth app (else a scheduled day-6 re-auth `**[ACTION REQUIRED]**`)?
- [ ] Did I add the client row to `clients_index.md`?
- [ ] Did I configure the schedule and create/propose the automation task with the client name at the start and `target_client_slug` pinned?
- [ ] Did I confirm the agency-wide `OutreachCRM - GitHub Update Watch` task exists/is proposed and is barred from client-facing channels?
- [ ] Did I perform Automation Resync for any post-schedule config change?
- [ ] Did I end with `ready_for_automation_first_run` and the exact task name to run, in chat, in a `**[ACTION REQUIRED]**` block?

### Sendbox & Send Self-Audit Checklist

- [ ] Did every send go through `gmail_client.py send`, never a one-off script or prose?
- [ ] Did the ten-gate ordered pre-send chain run in code, in order, with every block halting the send?
- [ ] Did I check global + client suppression after `resolve()`, and pull live unsubscribes (tracker `/events` + `+unsub` mailbox) before the batch?
- [ ] If track-pull had not succeeded within N hours, did I block the box instead of sending?
- [ ] Did I honor the sticky sender (`assigned_sendbox`) for bumps/replies, applying rotation to step-1 only?
- [ ] Did I respect warmup cap + two-tier domain cap + the recipient send-window?
- [ ] Did I keep `List-Unsubscribe` + footer opt-out present, including in plain-text mode?
- [ ] Did I write `sent_log` + the `email_sent` activity and apply 30–180s jitter?
- [ ] Did I handle 429 (pause box), `invalid_grant` (reauth + `**[ACTION REQUIRED]**`), and other errors (draft back to `approved` with a blocker)?
- [ ] Did a broken box keep its pending follow-ups waiting (never reassigned) and surface "N follow-ups blocked"?

### Drafting & Approval Self-Audit Checklist

- [ ] Did each draft combine client profile + campaign goal + dossier + step intent, with bumps carrying NEW value?
- [ ] Does every personalization detail trace to a dossier fact with an `evidence_url`?
- [ ] Did the freshness gate pass (step-1 hooks in TTL; bumps from reserved points + `message_bank` with NO per-bump re-enrichment; stale-hook guard applied to every referenced time-sensitive hook)?
- [ ] Is every draft free of em dashes (`—`), and — when the campaign declares them — does the body carry the produced companion-doc URL (or the `on_fail` path) and 1–2 rotated `message_bank` messages this lead has not seen?
- [ ] Is every step-1 subject free of `Re:`/`Fwd:`, and are continuation subjects truthful `Re:` on real in-thread replies?
- [ ] Did I drop any draft with no verifiable hook to the honest no-hook fallback (or skip), never inventing detail?
- [ ] Did all drafts land in `pending_approval`, with nothing moved to `approved` without a chat `approve`?
- [ ] Did I render the Approval Report (operator-only, NOT scrubbed) with the High-confidence / Review-carefully split and clickable evidence URLs?
- [ ] Did I log every approve/reject/edit/hold decision to `approvals/approval_log.md`, and feed reject reasons to `learning_log`?

### Inbound Sync & Reply Self-Audit Checklist

- [ ] Did I sync every sendbox using the correct cursor (`historyId` for OAuth, IMAP UID for app_password) and `last_successful_sync_ts`?
- [ ] For an expired OAuth `historyId`, did I fall back to `q="after:{last_sync_epoch}"` (overlap + dedupe), never `newer_than:2d`?
- [ ] Did the deterministic classifier run in exact order (DSN/bounce → auto-reply/OOO → unsub alias → threadId/`In-Reply-To` → contact-no-thread → personal-count-only)?
- [ ] Did I map DSNs to the original via threadId + `rfc_message_id` + recipient/sent_at window, before the threadId check (Gmail threads DSNs into the original thread)?
- [ ] Did I suppress `negative`/`remove_intent` replies (even without the word "unsubscribe"), or raise an `**[ACTION REQUIRED]**` confirm task that blocks further sends?
- [ ] Did any inbound reply freeze the remaining sequence for that contact until triage completed?
- [ ] Did I keep personal (non-campaign) email as count-only — no body stored, no deep read?

### CRM Integrity Self-Audit Checklist

- [ ] Did every CRM mutation go through `crm_store.py`, with no direct file writes to `crm/`?
- [ ] Does every record carry `schema_version`/`id`/`created_at`/`updated_at`, with on-read upgrades persisted on next write?
- [ ] Are activities append-only with a monotonic `seq`?
- [ ] Did every stage change come from `pipelines.json` and carry `evidence_activity_id`, with invalid transitions blocked?
- [ ] Did `apply-rules` run deterministically with idempotent guard keys, never improvised by me?
- [ ] Did merges follow the deterministic rule (exact email/phone/social auto; fuzzy proposed), tombstone the loser, and union identities/channels/suppression into the survivor?
- [ ] Did every `lead_id` lookup call `resolve()`, and were pending-merge contacts excluded from every queue?

### Compliance Self-Audit Checklist

- [ ] Physical address + working opt-out present in every commercial email, including plain-text mode?
- [ ] Step-1 subjects truthful (no `Re:`/`Fwd:`); bumps real in-thread `Re:`?
- [ ] Opt-outs honored same-run (footer, `POST /u/`, `+unsub` mailbox, and `negative`/`remove_intent` replies) with immediate suppression + task closure?
- [ ] Suppression checked at initial send, follow-up, assisted channels, and at import against ALL identities; unioned on merge; pending-merge excluded?
- [ ] Guessed addresses verified via the API, `catch_all` excluded/capped, per-domain kill switch active, guessed cap + per-draft flag enforced in `gmail_client.py`, guessed bounce rate reported separately?
- [ ] US SMS gated on documented consent `{optin_source, optin_at, evidence_activity_id}` or existing relationship, default inbound-initiated, legal basis shown per assisted draft; Zalo cold off by default?
- [ ] For regulated industries, no unsupported claims / guarantees / fear-manipulation / personalized professional advice; disclaimers where appropriate?

### Reporting & Delivery Self-Audit Checklist

- [ ] Did I produce client-facing HTML/PDF only via `tools/report_renderer.py` (or a reusable approved template), never a one-off script, with the report-design skill loaded?
- [ ] Is the only client-facing output the weekly client report, and did I run the Client-Blind Scrub Gate on it (and its PDF) alone?
- [ ] Did the weekly client report's extracted text return zero hits for the `CLIENT_BLIND_TERMS` set?
- [ ] Did I keep all internal detail (sendbox/quota/warmup/suppression/guessed/provider/blocker/debug) in `INTERNAL_REPORT`, labeled `Not for client sharing`?
- [ ] Is every report standalone/portable (no remote fetch/scripts/CSS/fonts/images), with no horizontal overflow at 390px and wide tables wrapped/scrolled?
- [ ] Did I create/update `{client}-report_state.json` with counts/statuses that match the report set, and refresh `outputs/latest/` copies?
- [ ] Does every report link in chat/notification point to `.html` (or the uploaded URL), never `.md`?

### Notification Self-Audit Checklist

- [ ] Did I notify via WideCast `sendTelegramMessage` with email fallback, using `tools/provider_openapi.py` against the client's `provider_config.local.json` (`api_key_env`/`api_key_local`, never a field named `api_key`)?
- [ ] Did I optionally `uploadAsset` the `.html` for an operator URL, treating provider-hosted URLs as operator handoff links, not client-share links?
- [ ] Did the notification include status, counts, blockers, report path/link, and `INTERNAL_REPORT` status (never a bare "report ready")?
- [ ] Did I avoid any WideCast production/publishing/analytics call — notification only?
- [ ] If WideCast was unavailable, did I fall back to Gmail/email, or record `notification_channel: local_path_only` with the exact human action?
- [ ] Did I log the attempt in `daily-content-pipeline/notifications/notification_log.md`?

### Multi-Client Isolation Self-Audit Checklist

- [ ] Did the automation task process only its pinned `target_client_slug`?
- [ ] Did I read no file under another client's `clients/{other_slug}/` (contacts, dossiers, suppression, sendboxes)?
- [ ] Did every send use a sendbox registered to this client, with quota counted against this client's box/domain only?
- [ ] Did I check both agency-tier (`global_suppression.jsonl`) and client-tier (`crm/suppression.jsonl`) suppression, not one as proof of the other?
- [ ] Did the client-specific task name begin with the client name?

### Measure-Learning Self-Audit Checklist

- [ ] Did I record reply/bounce/unsubscribe as exact and opens as estimated, never inventing metrics?
- [ ] Did I bot-filter tracking (UA class, click-with-no-prior-open, all-links-within-N-seconds, datacenter ASN), storing only a UA classification, never the raw User-Agent?
- [ ] Did I keep open/click from triggering any stage change or auto-action — only a reply is conversion evidence?
- [ ] Did I report the guessed cohort's bounce rate separately?
- [ ] Did I feed reply/reject/bounce learnings into `analytics/learning_log.md`, campaign angle selection, and future drafting?

### Evidence-Based Audit Requirements

Scope the self-audit to the reply being sent:

- Before any completion claim or human handoff, run the full Response Self-Audit plus the gate(s) that match the work (Setup, Daily Run, Drafting, Send) and their stage checklists.
- For an intermediate progress reply, check the core set: human language; the `**[ACTION REQUIRED]**` contract; no secrets/credentials/API keys leaked; and a progress block plus `Automation freshness check` when schedule/automation exists. Add the items relevant to the action just taken.

For these mechanical gates, the audit must paste real command output, not a self-declaration:

- **Client-blind scrub (weekly report only):** the renderer owns the single term source (its `CLIENT_BLIND_TERMS`, defined by DESIGN §19) — do not hand-maintain a separate grep pattern here that can silently drift from it and miss terms. Render the weekly report through the renderer's own scrub gate and paste its result:
  ```bash
  python3 tools/report_renderer.py package --inputs <staging.html ...> \
    --output-html <path>/{client}-weekly-client-report.html --client-facing --fail-on-scrub
  # (or `render --input REPORT.md --output-html <path>.html --client-facing --fail-on-scrub`)
  ```
  Paste the `client_blind_terms_found` and `scrub_status` fields from that output: `client_blind_terms_found` must be empty and `scrub_status` must be `pass` (a scrub hit is a non-zero exit that blocks the render). On a scrub hit, reword the flagged sentence and re-render — never bypass the gate or hand-edit the blocked output. (Per DESIGN §19 the renderer's `CLIENT_BLIND_TERMS`, this playbook's prose list, and this check must all resolve to exactly that one set — the renderer output is the authority, so this gate reads it rather than re-listing the terms.)
- **Draft evidence check:** for each draft, confirm every personalization detail appears in the contact dossier with an `evidence_url`; paste the mismatch count (must be `0`).
- **Pre-send gate chain:** paste the ordered gate result for the batch (each of the ten gates → pass/blocked), proving it ran in code.
- **Approval consistency:** for each send, quote the `approvals/approval_log.md` line whose decision `approve` names that `draft_id`.
- **report_state consistency:** quote the status/count fields from `outputs/YYYY-MM/YYYY-MM-DD/{client}-report_state.json`.
- **LOAD LEDGER line counts:** paste the printed ledgers for the stages loaded this run.
- **Notification record:** paste the `daily-content-pipeline/notifications/notification_log.md` row for this run.

Print the evidence as a compact block, one line per gate:

```text
gate | evidence | pass/miss
```

Honest misses are compliant: a miss with a stated reason is acceptable, but a rubber-stamped pass without pasted evidence is a critical violation.

### Final Hard Gate

If any required checkbox above is not satisfied:

- Do not claim the run is complete.
- Fix the missing step if possible. Do not merely promise to fix a required missing item next run when it can be corrected now.
- If it cannot be fixed, explicitly report: what was missed; why; whether the output is still usable; what should happen next.

---

## Final Reminder For The Agent

The human should not need to manage the workflow manually.

The human provides only:

- Client name and business context (product/service, website/profile URL, location when it matters).
- One or more sendboxes to connect (priority path: @gmail.com App Password).
- Lists to import and campaigns with their goal.
- Corrections to the agent's inferred setup.
- The chat `approve` before anything is sent.
- The WideCast API key once per client, if they want Telegram/email alerts while away from the AI agent UI.

The agent owns:

- Industry/audience inference and client profile maintenance.
- List import, verification, and enrichment (dossier + writing brief).
- Goal-driven drafting with evidence-backed personalization.
- The Approval Report and chat-approval gate.
- The in-code send engine with the full pre-send safety chain, multi-sendbox rotation, and sticky sender.
- Inbound sync, deterministic classification, and semantic triage.
- The CRM pipeline (accounts/contacts/deals/activities/tasks) with deterministic rules and evidence-backed stage changes.
- Follow-up advising and sequence-freeze discipline.
- Tracking, honest metrics, and the learning loop.
- Compliance encoding (CAN-SPAM, suppression at every path, guessed-email policy, US SMS consent).
- Operator-only reporting and the single scrubbed weekly client report.
- Notification delivery via WideCast Telegram/email fallback.
- Schedule/automation setup, Automation Resync, and Stage 11 update discipline.
- Running this Stage 9 audit and its completion gates before any completion claim.

This is the intended operating model. When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
