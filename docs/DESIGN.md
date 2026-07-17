# OutreachCRM — Authoritative Design (source of truth)

> This document is the single source of truth for OutreachCRM. Every playbook, tool,
> and skill must conform to it. It was produced by transforming the Solo Agency
> playbook architecture into a multi-client cold-email + CRM system, then hardening
> the design against a 6-reviewer adversarial pass (42 findings integrated).
> When any file disagrees with this document, this document wins.

Status: Phase 0 (scaffold). Snapshot date: 2026-07-15.

---

## 1. What OutreachCRM is

A **local-first, multi-client cold-email + CRM system** operated by an AI agent
(Claude Desktop / Codex) via markdown playbooks. It inherits Solo Agency's operating
system — thin router + Stage Map, lazy-load with LOAD_LEDGER full-load discipline,
Setup Flow vs Automation Flow split, one automation task per client, Automation
Resync, Stage 11 GitHub Update Watch, provider adapter (WideCast for Telegram
notification), and the standalone HTML report renderer — and replaces the
content/video/collector business layer with: list import → verify → enrich →
goal-driven email drafting → **preview & chat-approval** → send (multi-sendbox
rotation) → tracking (reply/bounce always; open/click optional) → follow-up →
CRM pipeline (accounts/contacts/deals/activities/tasks) → weekly client report.

**Model:** 1 agency → N clients; each client = one isolated CRM workspace with its
own pipelines, sendboxes, suppression, and data; each client → N campaigns, each
campaign declaring its own **goal** that drives what the agent writes.

**Positioning:** open source (MIT), English playbooks, @gmail.com sendboxes are the
priority path, agency operates on behalf of clients, clients receive only a weekly
scrubbed report.

---

## 2. Non-negotiable inherited mechanisms (keep verbatim or near-verbatim)

1. **Thin router + Stage Map.** `OUTREACHCRM_PLAYBOOK.md` is a dispatch table only;
   business logic lives in numbered stage playbooks loaded on demand.
2. **Full-load discipline.** `playbooks/LOAD_LEDGER_PROTOCOL.md` (kept ~verbatim) +
   auto-generated `playbooks/LOAD_MANIFEST.md` (path | lines | sha256 | last_line).
   A short read = NOT loaded. No side-effect action without a PASS ledger. This is
   the machinery the whole architecture exists to enforce — do not weaken it.
3. **Setup Flow vs Automation Flow split.** Setup Flow is the control plane: it
   creates config + the automation task and **never sends an email, never runs a
   campaign, never enriches for send**. Terminal state: `ready_for_automation_first_run`.
4. **One automation task per client**, name begins with the client name, prompt pins
   `target_client_slug`, cannot touch another client. Plus one agency-wide
   `OutreachCRM - GitHub Update Watch` task.
5. **Automation Resync.** Any post-setup config change re-syncs the profile, schedule,
   automation manifest, scheduled-run prompt, and native task body, with a dry-read
   verification, before it is called complete.
6. **Stage 11 Update & Version Watch.** Fresh GitHub checkout protocol, `update_state.json`,
   backup-and-safe-apply (merge config, never overwrite secrets/history), change
   classification, update-watch task barred from client-facing channels.
7. **Provider adapter + PDNA (Telegram only).** Per-client `provider_config.local.json`
   (`api_key_env`/`api_key_local`, never a field literally named `api_key`), OpenAPI
   discovery via `tools/provider_openapi.py`, notification via WideCast
   `sendTelegramMessage` with email fallback. Kept for operator notification only.
8. **Two-lane reporting.** Operator-only (`INTERNAL_REPORT`, full detail) vs
   client-facing (through the Client-Blind Scrub Gate). Rendered by
   `tools/report_renderer.py` (stdlib only). The client-facing reports are the **weekly**
   report and the **monthly** report; every other output is operator-only.
9. **`[ACTION REQUIRED]` contract.** One purpose, one exact next step, one command or
   path. Say `No action required right now.` when nothing is needed.
10. **Slug rules** (lowercase, hyphens, no punctuation) and monthly `YYYY-MM/` folders.
11. **Deploy script discipline:** auto-generate LOAD_MANIFEST, secret-scan staged diff
    before commit, refuse to commit into the wrong git root.

---

## 3. Naming / path map (Solo Agency → OutreachCRM)

| Solo Agency | OutreachCRM |
|---|---|
| `SOLO_AGENCY_PLAYBOOK.md` | `OUTREACHCRM_PLAYBOOK.md` |
| `deploy-soloagency.sh` | `deploy-outreachcrm.sh` |
| data root `daily-content-pipeline/` | `outreach-pipeline/` |
| `tools/solo_report_renderer.py` | `tools/report_renderer.py` |
| task `{Client} - Solo Agency Daily Run` | `{Client} - OutreachCRM Daily Run` |
| `Solo Agency - GitHub Update Watch` | `OutreachCRM - GitHub Update Watch` |
| GitHub `github.com/soloagency/solo-agency` | `github.com/soloagency/outreach` (repo name is `outreach`; product name stays OutreachCRM) |

**Deleted components** (must have zero surviving references anywhere except this row):
`solo-agency-collector/`, Local Collector / bridge / Chrome extension / `client_binding.json`
/ `127.0.0.1:17321`, `PRIVATE_SOURCE_GATE.md`, `02_PRIVATE_SOURCE_SETUP.md`,
`08_LOCAL_COLLECTOR_TECHNICAL_PROTOCOL.md`, `SOLO_AGENCY_VIDEO_PROVIDER_ADAPTER.md`,
video/blog/social skills, `10_LEAD_COMPETITOR_DETECTION.md` (folded into CRM),
public/private "data sources" concepts, PDNA production/video/render/distribution
(only PDNA **notification** survives).

---

## 4. Stage Map (new)

| Stage | File | Load when |
|---|---|---|
| 0 | `playbooks/00_CORE_CONTEXT_REQUIREMENTS.md` | always first |
| — | `playbooks/LOAD_LEDGER_PROTOCOL.md` | referenced by every load |
| 1 | `playbooks/01_CLIENT_SETUP_PROFILE.md` | new client setup / first run |
| 2 | `playbooks/02_SENDBOX_SETUP.md` | connect/check a sendbox |
| 3 | `playbooks/03_IMPORT_LIST.md` | import a CSV/TXT/XLSX list |
| 4 | `playbooks/04_VERIFY_ENRICH.md` (+ skill `email-verify-enrich`) | before any enrichment |
| 5 | `playbooks/05_CAMPAIGN_MANAGEMENT.md` | create/edit a campaign, define goal |
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
| 6A | `playbooks/skills/report-design/SKILL.md` | report rendering (kept) |
| Auto | `playbooks/AUTOMATION_SCHEDULING.md` | configuring the schedule/automation task in Setup Flow; the start of every scheduled run; any Automation Resync. Defines the Daily Run order, run_lock, and resync machinery. |
| Setup | `playbooks/SETUP_FLOW_ENTRYPOINT.md` | setup sessions |
| Sched | `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` | unattended daily runs |

Built so far: 0, 1, 7, 9, 11, AUTOMATION_SCHEDULING, root playbook, both entrypoints,
LOAD_LEDGER, AGENTS.md, deploy script, README, renderer, LOAD_MANIFEST (Phase 0); **2, 3, 8
plus tools `storage/`, `crm_store.py`, `import_leads.py`, `email_verify.py`, `gmail_client.py`
and `tests/` (Phase 1)**; **4, 5, 6, 10, 13, 14 plus the `email-verify-enrich` /
`email-writing` skills, the Approval Report + chat-approve handler, follow-up/reply, and the
Today View / kanban (Phase 2, milestones 2A–2D)**. Still `status: planned`: 12 and 15 (metrics /
polished CRM reporting) and `tracker/worker.js` — Phase 3; the minimal weekly report + full
Scheduled-Run wiring land in 2E. Per §22 R1, a `status: planned` row is never a load failure —
load this DESIGN section for its contract.

---

## 5. On-disk layout (`outreach-pipeline/`)

```text
{agency_root}/
  outreachcrm/                         # this repo (toolkit/source), no client data
  outreach-pipeline/                   # data/config/output only
    clients_index.md
    schedule.md
    storage_config.json                # {"backend":"json"}  (or postgres)
    provider_defaults.json             # WideCast notification catalog, no secrets
    secrets/                           # gitignored; agency-wide secrets (OAuth client, tracker key)
    suppression/
      global_suppression.jsonl         # agency-tier suppression (checked before every send)
    automation/
      automation_manifest.md  scheduled_run_prompt.md  resync_log.md  github_issues.md
      update_state.json  update_log.md  update_notice.md  update_watch_prompt.md
      backups/update_YYYY-MM-DD_HHMMSS/
      issues/YYYY-MM-DD_{blocker_slug}.md
    notifications/notification_log.md
    clients/{client_slug}/{business_slug}_{location_slug}/
      client_profile_{client_slug}_{business_slug}_{location_slug}.md
      sendboxes/
        sendboxes.json
        {sendbox_slug}/credentials.json  {sendbox_slug}/token.json   # gitignored, chmod 600
      lists/{list_slug}/list_manifest.json  leads.jsonl  import_log.md
      crm/
        accounts/{account_id}.json
        contacts/{lead_id}.json
        contact_identities.jsonl        # reverse index: (kind,value)->lead_id, unique
        deals/{deal_id}.json
        activities/YYYY-MM/activities.jsonl   # append-only, each row has monotonic seq
        tasks/tasks.jsonl
        pipelines.json
        segments.json
        suppression.jsonl               # client-tier suppression
      campaigns/{campaign_slug}/
        campaign_config.json
        queue/enrich_queue.jsonl
        queue/enriched/YYYY-MM-DD/{lead_id}.json
        outbox/pending_approval/YYYY-MM-DD/{draft_id}.json
        outbox/approved/{draft_id}.json
        sent/YYYY-MM/sent_log.jsonl
        history/YYYY-MM/campaign_log.md  reply_log.md
      assets/
        asset_index.md
        proposals/{slug}/v001/...   flyers/{slug}/v001/...
      approvals/approval_log.md
      analytics/metrics_log.md  learning_log.md
      inbox_sync/YYYY-MM/sync_log.jsonl
      reports/YYYY-MM_report.md
      outputs/YYYY-MM/YYYY-MM-DD/
        {client}-approval-report.html          # operator-only, NOT scrubbed
        {client}-today-view.html               # operator-only
        {client}-daily-ops.html  {client}-INTERNAL_REPORT.html
        {client}-weekly-client-report.html     # CLIENT-FACING, scrubbed, Mondays
        {client}-weekly-client-report.pdf
        {client}-monthly-client-report.html    # CLIENT-FACING, scrubbed, first run each month
        {client}-report_state.json
      outputs/latest/...
      integrations/providers/
        provider_config.local.json  provider_capabilities.json
        provider_openapi_cache.yaml  provider_calls.jsonl  provider_health.md
```

**Client-scope is structural, not disciplinary.** The storage adapter is instantiated
per client rooted at `clients/{slug}/crm/`. Agency-tier collections (global
suppression, `secrets/`, `provider_defaults.json`, tracker key) are the only things
allowed to be global and are enumerated explicitly.

---

## 6. Storage adapter (pluggable JSON → Postgres)

`tools/storage/adapter.py` defines the interface; `json_adapter.py` is default;
`postgres_adapter.py` comes later and must pass the same parametrized contract tests.

Interface:
```
get(collection, id) -> dict | None
put(collection, id, record) -> None                 # atomic (temp+rename), bumps updated_at
update(collection, id, mutate_fn) -> dict            # read-modify-write under the collection lock
delete(collection, id) -> None                       # rarely used; prefer tombstones
query(collection, where: [Cond], sort=None, limit=None, offset=None) -> [dict]
append(log, record) -> None                          # append-only, stamps ts + monotonic seq
read_log(log, since_seq=None, where=None) -> [dict]  # ordered by seq (backend-independent)
find_by_identity(kind, normalized_value) -> id | None  # backed by unique reverse index
reserve(sendbox_slug, day) -> token | None           # atomic quota reservation (see §10)
```
- `Cond = (field, op, value)`, `op ∈ {=, !=, <, >, contains, in}`. This DSL covers
  flat fields only; **identity lookups do NOT use it** — they use `find_by_identity`
  over a maintained unique reverse index (`contact_identities`). Do not claim Cond
  translates arbitrary nested-array matches to SQL.
- Every record: `schema_version`, `id`, `created_at`, `updated_at`. Adapter holds a
  per-collection `{from_version: fn}` upgrade registry applied on read, persisted on
  next write.
- JSON adapter: one file per record; logs are monthly JSONL; atomic writes via
  temp+rename; per-collection `fcntl` lockfile; a per-log counter file (under the log
  lock) supplies the monotonic `seq`.
- Postgres adapter: table per collection `(client_id, id, payload jsonb, created_at,
  updated_at, <generated index cols>)`; `client_id` mandatory in every table and every
  generated WHERE; `contact_identities(client_id, kind, value UNIQUE, contact_id)`;
  logs get a `seq bigserial`; index columns are GENERATED from payload.
- `crm_store.py migrate --to postgres` runs under a storage freeze flag, verifies with
  **per-record content hashes** (not counts), upgrades all records to current
  schema_version first.

**All CRM mutations go through `crm_store.py`.** Direct file writes are a critical
violation (inherited "no one-off scripts" rule). Reading raw JSON is allowed only for
debugging.

---

## 7. CRM data model

### 7.1 Contact (`crm/contacts/{lead_id}.json`) — email NOT required
`lead_id` = ULID minted at import (not a hash of email, because email may be absent).
```json
{
  "id": "c_01J...", "schema_version": 2, "created_at": "", "updated_at": "",
  "name": {"full": "", "first": "", "last": ""},
  "account_id": "a_...",
  "identities": {
    "emails": [{"address": "", "source": "import|enrich|guess", "status": "unverified|mx_ok|delivered|bounced|guessed_only|catch_all|email_not_found", "is_primary": true}],
    "phones": [{"number": "+1...", "type": "cell|office", "source": ""}],
    "socials": {"facebook": null, "instagram": null, "linkedin": null, "zalo": null, "x": null},
    "website": null
  },
  "channels": {
    "email":     {"status": "usable|needs_data|opted_out|bounced"},
    "sms":       {"status": "needs_optin|usable|opted_out", "mode": "assisted", "optin": {"source":"", "at":"", "evidence_activity_id":""}},
    "messenger": {"status": "usable|needs_data", "mode": "assisted"},
    "zalo":      {"status": "needs_data", "mode": "assisted"}
  },
  "lifecycle_stage": "lead|engaged|opportunity|customer|evangelist|lost|do_not_contact",
  "tz": "America/Chicago",              // for send-window gate; inferred from state/area code
  "tags": [], "custom_fields": {},
  "owner": "agency",
  "enrichment": { /* distilled copy of the dossier — see §9 */ },
  "assigned_sendbox": null,             // sticky sender, set on first send (see §10)
  "merge": {"status": "active|merged", "merged_into": null},
  "next_action": {"task_id": null}
}
```

### 7.2 Account (`crm/accounts/{account_id}.json`)
Company/office (e.g. a brokerage). `{id, name, domain, type, location, contact_ids[], custom_fields}`.

### 7.3 Deal (`crm/deals/{deal_id}.json`)
```json
{"id":"d_...","schema_version":1,"name":"","contact_ids":[],"account_id":"",
 "pipeline":"default_sales","stage":"new_reply","value":0,"currency":"USD","probability":0.1,
 "expected_close":"","source_campaign":"",
 "stage_history":[{"stage":"new_reply","at":"","by":"rule:r1","evidence_activity_id":""}],
 "status":"open|won|lost","lost_reason":null,"next_action":{"task_id":null}}
```

### 7.4 Activity (`crm/activities/YYYY-MM/activities.jsonl`) — append-only, the event backbone
```json
{"seq":123,"ts":"","id":"act_...","contact_id":"","deal_id":null,
 "type":"email_sent|email_reply|email_open|email_click|email_bounce|unsubscribe|call|meeting|note|stage_change|task_done|enriched|imported|merged|assisted_sent",
 "summary":"","ref":{"message_id":"","url":"","path":""},"by":"agent|human|rule"}
```
A contact timeline = filter this by `contact_id` (following merge chains via `resolve()`).

### 7.5 Task (`crm/tasks/tasks.jsonl`)
`{id, contact_id?, deal_id?, title, due_at, status: open|done|cancelled, created_by: rule|human|agent, guard_key}`.

### 7.6 Pipelines + rules (`crm/pipelines.json`)
Stages carry `probability` + `sla_days`. Rules are **deterministic**, executed by
`crm_store.py apply-rules`, never improvised by the LLM.
```json
{"pipelines":[{"id":"default_sales","stages":[
   {"id":"new_reply","probability":0.10,"sla_days":1},
   {"id":"engaged","probability":0.25,"sla_days":7},
   {"id":"meeting_booked","probability":0.50,"sla_days":7},
   {"id":"proposal_sent","probability":0.70,"sla_days":10},
   {"id":"won"},{"id":"lost"}]}],
 "rules":[
   {"id":"r1","on":"reply_positive","do":["create_deal_if_none(stage=new_reply)","create_task(title=Reply within 4h,due=+4h)","freeze_sequence"]},
   {"id":"r2","on":"reply_question","do":["create_deal_if_none(stage=engaged)","freeze_sequence","draft_reply_for_approval"]},
   {"id":"r3","on":"reply_negative|remove_intent","do":["suppress(contact)","freeze_sequence","close_open_tasks"]},
   {"id":"r4","on":"stage_age_exceeds_sla","do":["create_task(nudge)","flag_in_report"]},
   {"id":"r5","on":"deal_won","do":["set_lifecycle(customer)","enroll_segment(customers)","create_task(onboarding)"]},
   {"id":"r6","on":"hard_bounce|unsubscribe","do":["suppress(contact)","close_open_tasks"]}
 ]}
```
Guard keys `(rule_id, trigger_activity_id)` make `apply-rules` idempotent/re-runnable.

### 7.7 Merge semantics (deterministic)
Auto-merge on exact email / E.164 phone / canonical social URL match. Fuzzy
name+company → propose, human approves. Losing record becomes a permanent tombstone
`{merge:{status:"merged", merged_into:A}}` (never deleted); identities, channel
statuses, and suppression are **unioned** into the survivor. Every `lead_id` lookup
path (sync classifier, track-pull, unsub handler, apply-rules, drafting) calls
`resolve(lead_id)` to follow merge chains. Contacts with a pending merge proposal are
excluded from every campaign queue until resolved.

---

## 8. Sendboxes & multi-sendbox rotation

`sendboxes/sendboxes.json`:
```json
{"sendboxes":[
  {"slug":"sb-a","auth_mode":"app_password|oauth","email":"...","domain":"gmail.com",
   "quota_today":40,"warmup_stage":"week_1|week_2|mature","status":"healthy|needs_reauth|paused",
   "historyId":null,"imap_uid_cursor":null,"last_successful_sync_ts":""}]}
```
- **Two auth modes, one interface.** `app_password` (priority for @gmail.com): SMTP
  send + IMAP read via Python stdlib (`smtplib`/`imaplib`), no OAuth, no 7-day expiry,
  preserves our Message-ID. `oauth` (Workspace/custom domain): Gmail API, scopes
  `gmail.send + gmail.readonly` only (drop `gmail.modify`), OAuth app must be
  **Internal** to avoid the 7-day refresh-token expiry; if forced External/testing,
  weekly re-auth becomes a scheduled day-6 `[ACTION REQUIRED]`, not an error path.
- **Rotation is step-1 only; sticky sender thereafter.** First outreach picks the
  healthy referenced sendbox with the lowest `sent_today/quota_today` ratio
  (round-robin on ties); `contact.assigned_sendbox` is then fixed. Every bump/reply
  goes from the assigned box (threading + reply routing + anti-spam require it).
- **Two-tier cap.** `min(remaining_box_quota, remaining_domain_cap)` — several boxes on
  one domain share domain reputation; domain volume ramps too. Real scale = 2–3
  variant domains, 1–2 boxes each, each warmed independently.
- **Broken box:** dropped from step-1 rotation; its assigned pending follow-ups **wait**
  (never reassigned) + `[ACTION REQUIRED]` re-auth; report shows "N follow-ups blocked".
- **Consumer @gmail.com limits (documented, accepted):** From is gmail.com → tracking
  links live on an unrelated domain → default `plain_text_mode` (no pixel, no link
  rewrite), measure by reply; no custom Message-ID domain; ~20–50 cold/day/box; never
  the operator's primary Gmail; cold bulk risks account suspension (accepted at low
  volume with tight personalization). App Password requires 2FA and is a
  Google-tightened surface — keep the OAuth mode available as fallback.

---

## 9. Enrichment (Stage 4 + skill `email-verify-enrich`)

### 9.1 Cross-campaign inheritance
The dossier belongs to the **contact** (client-scope), campaigns reference `lead_id`.
Enrich queue is client-level, deduped by `lead_id` (one job even if two campaigns want
the same person). Two TTL tiers:
- **Durable (identity + context), TTL ~90d:** still-active, license, current company,
  profile URLs, found emails/phones, market, content style. Inherited as-is by other
  campaigns.
- **Fresh (hooks), TTL 7–14d:** new listing, new post, new review, recent event. Other
  campaigns run a cheap **refresh** (revisit known URLs), not full re-discovery.
- **Negative cache inherited too:** `email_not_found` (retry after 30d then stop),
  `no_verifiable_hook` (with last-tried date) — don't re-burn the same dead end.
- **No-email leads are queued for email DISCOVERY, not skipped.** Under an `email_first`
  campaign, a contact with no (or invalid) email is still QUEUED so enrichment can find one
  (visit the profile, the website, license/roster records, Google, other channels) — a missing
  email is the reason to search, not a reason to skip. It is skipped at queue time ONLY when a
  recent negative cache says discovery already failed: `enrichment.email_not_found_at` within
  `NEG_RETRY_DAYS` (30d). The email requirement still hard-gates at draft/send time (`_valid_email`
  in `draft write` + the pre-send chain). After a failed discovery the agent marks
  `mark_email_not_found` (→ 30d negative cache) and the lead becomes an **assisted-channel
  candidate** (manual SMS/Messenger/Zalo), never emailed at a guessed address.
- Each hook carries `used_in: ["campaign/step"]`; a second campaign may not open with a
  hook already used on that person. A contact in an active sequence of campaign A is
  not drafted by B (`min_days_between_touches_across_campaigns`).

### 9.2 Dossier (`queue/enriched/YYYY-MM-DD/{lead_id}.json`; distilled copy into `contact.enrichment`)
```json
{"lead_id":"","identity":{"still_active":"confirmed|inactive|unknown",
   "evidence":[{"fact":"","url":"","retrieved_at":""}],"current_company":"","role":"",
   "profiles":{"zillow":"","website":"","facebook":"","instagram":"","gbp":""},
   "channels_found":{"emails":[],"phones":[]}},
 "context":{"market":"","volume_signals":"","specialty":"","content_style":""},
 "hooks":[{"type":"new_listing|social_post|review|award|market_view|website_update",
   "summary":"","analysis":{"topic":"","angle":"","sensitivity":"public_business|personal"},
   "evidence_url":"","observed_date":"","confidence":0.0,"used_in":[]}],
 "writing_brief":{"one_liner":"","ranked_angles":[],"do_not_mention":[],"personalization_confidence":0.0}}
```
The email-writing skill consumes `writing_brief` (ranked by freshness × goal-fit ×
confidence), not raw data.

### 9.3 Channel reality (be honest about what is readable)
Readable now (MVP, WebSearch/WebFetch/browser tool): personal website/blog (best),
YouTube title/desc, Instagram/X public (best-effort), Zillow/GBP reviews (browser tool
or snippet). **Not readable logged-out: Facebook, LinkedIn** — store URL only.
Reading Facebook posts (`fb.profile.posts`) is exactly what a future **Phase 4 Local
Collector** (inherited from Solo Agency, using the operator's own logged-in Chrome)
would solve. Do not promise reading logged-in-only content in the MVP.

### 9.4 Social post analysis + etiquette
Where readable, analyze the 3–5 latest posts: `{date, topic, summary, what it reveals,
angle, sensitivity}`. Etiquette hard rule: `public_business` signals (listings, work
posts, reviews, awards, market opinions) are fair game; `personal` signals (family,
health, vacations, children) are **default-banned from email copy** and go only into
`do_not_mention`.

### 9.5 Two-tier flow + freshness gate
- **Tier 1 Verify (cheap subagent):** check existing dossier first; if identity in TTL,
  skip to hooks. Else: name+company+location search → license → roster → Zillow snippet
  → collect profile URLs + emails/phones. `inactive/unknown` → mark, stop (no Tier 2).
- **Tier 2 Profile & hooks (main model):** visit known URLs per readability table →
  extract hooks with evidence → analyze social content → distill `writing_brief` → score
  `personalization_confidence` (≥0.7 High; 0.4–0.7 Review carefully; <0.4 →
  `no_hook_fallback`).
- **Freshness gate at write time:** before step-1 draft, hooks must be within TTL (else
  refresh known URLs); before every follow-up, micro-refresh the person's 1–2 best
  sources to (a) find a fresh bump hook and (b) **invalidate stale hooks** (a sold
  listing must not be referenced as active). Hard rule: a draft may contain only
  details present in the dossier with an `evidence_url`; Stage 9 audit checks this
  mechanically.

### 9.6 Guessed email
MX check is near-meaningless (catch-all domains accept any RCPT). Guessed/unverified
addresses go through a **third-party verification API** (MillionVerifier/NeverBounce,
cheap, called from local Python). `catch_all` → excluded from guessed quota or capped
~2%. Per-domain kill switch: first hard bounce on a guessed pattern at domain X →
suppress all other guessed addresses at X. `guessed_only` status enforced **in
`gmail_client.py send`** (requires explicit guessed-approval flag on the draft + a
daily guessed-send cap read from sent_log), never only in prose. Guessed cohort bounce
rate is reported separately.

---

## 10. Send engine (Stage 8)

`gmail_client.py send` per draft, in code (do not trust playbook prose):
1. **Pre-send re-check, ordered:** resolve(lead) → global+client suppression (live: also
   pull new unsubscribes from the tracker `/events` + the `+unsub` mailbox before any
   batch; if track-pull failed > N hours, **block** the box) → `channels.email.status`
   → **atomic quota reservation** (`reserve(sendbox, day)`; append `send_reserved` under
   the sent_log lock before releasing — no count-then-send race) → warmup cap →
   two-tier domain cap → send-window (recipient tz) → guessed cap (10%/day/box, and
   guessed requires its approval flag) → sequence-freeze check (any inbound reply
   freezes remaining bumps) → step-1 subject lint (reject `^(Re|Fwd):`).
2. **Build MIME multipart/alternative:** text/plain primary + minimal text/html. For
   continuation sends (bump step>1, replies): set `threadId` (OAuth) / thread via
   `In-Reply-To`+`References` from the prior `rfc_message_id` in sent_log, with a
   consistent `Re:` subject.
3. **Tracking (only when the box's mode allows and the campaign enables it):** open pixel
   `https://trk.{domain}/o/{token}.gif` (Cache-Control: no-store); links rewritten to
   `https://trk.{domain}/c/{token}/{sig}/{b64url}` with `sig = HMAC(secret, token||url)`;
   `token = base32(hmac(secret, lead_id|message_uid))[:12]`. In `plain_text_mode`:
   **no pixel, no rewrite, but always keep** `List-Unsubscribe` (mailto + https) and the
   footer opt-out — `/u/` is compliance, not tracking.
4. **Headers:** `List-Unsubscribe: <mailto:{box}+unsub-{token}@...>, <https://trk.{domain}/u/{token}>`
   + `List-Unsubscribe-Post: List-Unsubscribe=One-Click`. No `X-Campaign` fingerprint.
   Message-ID: our own only when we control the domain (OAuth/Workspace) — after send,
   fetch the on-the-wire Message-ID (`messages.get` metadata for OAuth; SMTP preserves
   ours) and store it as `rfc_message_id`.
5. **Send**, record `sent/YYYY-MM/sent_log.jsonl`
   `{lead_id, campaign, step, sendbox, provider_id, thread_id, rfc_message_id, token,
   links:{}, sent_at, seq}`, append activity `email_sent`, sleep jitter 30–180s.
6. **Errors:** 429/quota → pause box today; invalid_grant → `needs_reauth` +
   `[ACTION REQUIRED]`; other → draft returns to `approved` with a blocker.

---

## 11. Tracker worker (`tracker/worker.js`, Cloudflare)

One Worker on `trk.{domain}`, **D1** (SQLite, strongly consistent — not KV, whose
eventual consistency can lose unsubscribe events → CAN-SPAM risk). Endpoints:
- `GET /o/{token}.gif` → 1×1 gif, `Cache-Control: no-store`, log open.
- `GET /c/{token}/{sig}/{b64url}` → verify `HMAC(secret, token||url)`, 302 to the URL, log click.
- `GET /u/{token}` → renders a confirm page, **does not change state** (scanners fetch
  GET links). `POST /u/{token}` body `List-Unsubscribe=One-Click` → unsubscribe now,
  idempotent, 200 no redirect (RFC 8058). POST from the page button → unsubscribe.
- `GET /events?since={seq}` (Bearer `TRACKER_API_KEY`) → events for agent pull; also a
  reconcilable `unsub:{token}` state key so unsubscribe reconciliation doesn't depend on
  the cursor.
- **Injection defense:** never store raw User-Agent (attacker-controlled, later read by
  the agent); store only a classification (`gmail-proxy|safelinks|browser|unknown`).
  Track-pull accepts only click URLs matching the token's stored `links{}` in sent_log.
- **Bot filter (unified):** UA class (GoogleImageProxy = reliable open signal; scanner
  list), click-with-no-prior-open, all-links-within-N-seconds regardless of timing,
  datacenter ASN from `request.cf`. **Open/click never alone trigger a stage change or
  auto-action — only a reply is conversion evidence.**

**Metric honesty:** reply/bounce/unsubscribe are exact (IMAP/Gmail + DSN + worker);
open is an estimate (Gmail image proxy, Apple MPP prefetch, image-blocking); click is
fairly reliable after bot filtering. Reports label opens "estimated."

---

## 12. Inbound sync + classification (Stage 10)

Per sendbox, cursor = `historyId` (OAuth) or IMAP UID (app_password) + `last_successful_sync_ts`.
OAuth fallback on expired historyId is `q="after:{last_sync_epoch}"` (overlap + dedupe
by message id, handle `nextPageToken`) — never `newer_than:2d`. **Deterministic
classifier, in this exact order** (order is load-bearing — Gmail threads DSN bounces
into the original thread, so DSN must be checked before threadId):
1. **DSN/bounce:** From mailer-daemon/postmaster, `multipart/report; report-type=delivery-status`,
   or a `message/delivery-status` part → hard(5.x.x)/soft(4.x.x); map to the original via
   threadId + `rfc_message_id` + recipient/sent_at window.
2. `Auto-Submitted: auto-replied` / OOO.
3. **Unsub alias (deterministic):** any To/Delivered-To matches `{box}+unsub-{token}@` →
   extract token → unsubscribe for the exact lead (mailto unsubs often have empty bodies).
4. threadId / In-Reply-To match sent_log → campaign reply → mark `reply_untriaged`.
5. From ∈ contacts but no thread match → `contact_message`.
6. Else → personal email: **count only, do not store body, do not deep-read.**

Then semantic triage of `reply_untriaged` → `positive|question|objection|negative|remove_intent`;
`negative`/`remove_intent` (even without the word "unsubscribe") → suppression (or an
`[ACTION REQUIRED]` confirm task that blocks further sends). Invariant enforced in code
at both draft-time and send-time: **any inbound reply freezes the remaining sequence**
for that contact until triage completes.

---

## 13. Campaigns & goal-driven writing (Stage 5 + 6)

`campaign_config.json` (goal is the writing blueprint, not a label):
```json
{"campaign_slug":"","goal":{"goal_type":"book_meeting|get_reply|direct_sale|reactivation|nurture_upsell|event_invite",
   "objective":"","offer":"","value_proposition":"","proof_points":[{"claim":"","evidence_url":""}],
   "cta":{"type":"reply_yes|link|calendar","text":""},
   "success_event":{"on":"reply_positive","create_deal_stage":"new_reply"}},
 "audience":{"segment":"","personalization":{"required_hook_types":[],"min_confidence":0.7,"no_hook_fallback":"skip|generic_honest_opener"}},
 "sequence":[{"step":1,"intent":"hook + offer, one CTA","tracking":"plain_text"},
   {"step":2,"gap_days":4,"intent":"deliver new value"},
   {"step":3,"gap_days":5,"intent":"social proof"},
   {"step":4,"gap_days":7,"intent":"breakup"}],
 "sendboxes":[],"daily_quota":40,"approval_mode":"manual_all",
 "guardrails":{"banned_claims":["guarantees"],"no_fake_re":true},
 "channel_strategy":"email_first|any_channel"}
```
`no_hook_fallback` defaults to **`skip`** (proof-of-life). Recent evidenced activity is the reason
an email exists, so `draft write` REJECTS a step-1 draft that has no evidenced hook
(`no_evidenced_hook`) unless the campaign explicitly opts into `generic_honest_opener` (which then
still flags a `generic_opener` warning). Step>1 drafts (bumps/replies) are exempt — an existing
conversation is its own justification, governed by the no-"just following up" rule.
`daily_quota` doubles as the **daily draft budget**: the daily run drafts while
`crm_store.py draft budget --campaign <slug>` reports `remaining > 0`, then stops.

`goal_type → email structure` table (skill `email-writing`, modeled on the video-script
skill's format table): `book_meeting`→short, one time-bound CTA; `get_reply`→ends with a
question, no link; `direct_sale`→value + one offer link (the only place click tracking
is on by default); `reactivation`→evidence of prior relationship + "still doing X?";
every final step→breakup. A draft = client profile (voice, offer, compliance) + campaign
goal (objective, CTA, proof) + contact dossier (hooks + evidence) + step intent (bumps
carry NEW value, never "just following up"). `success_event` wires straight into the
rules engine.

---

## 14. Preview & chat-approval (the gate before any send)

At the end of the drafting pass, the agent renders an **Approval Report**
(`outputs/.../{client}-approval-report.html`, operator-only, NOT scrubbed) via the
inherited renderer (reusing its contenteditable + Copy-button blocks):
- **Three sections.** New step-1 drafts split into **High confidence** (verified email, ≥0.7
  hook) and **Review carefully** (weak hook, guessed email, fallback opener) — both computed
  over step-1 drafts only. Step>1 drafts (bumps + reply drafts) go to a dedicated
  **Follow-ups due** section (*bumps and reply drafts — threaded onto an existing conversation*).
  Numbering is stable and unique across all three sections.
- Header: total drafts by campaign/step.
- One card per lead: `#id` · name/company/email + verify status · hooks with **clickable
  evidence URLs** · subject + editable body + warning flags (guessed, generic, bump step).
Telegram: "N drafts awaiting review" + path.

**Chat approval grammar** (chat is the write path; editing the HTML does not persist):
```
approve all
approve 1-20, 35, 41
reject 7: hook is stale, that listing sold
edit 12: change CTA to "Worth a quick look?"
hold 5
```
Approved → `outbox/approved/` → **sent immediately in-session** (within quota, jitter,
full in-code re-check chain); rejected → logged with reason → reason feeds
`learning_log` for the next batch; edit → agent patches, re-confirms, then approves.
Every decision → `approvals/approval_log.md`. Nothing leaves without an explicit
"approve". Default `approval_mode: manual_all` even for bumps.

---

## 15. Daily Run order (per client, pins `target_client_slug`)
1. Load contract + LOAD LEDGER; read automation manifest + `update_state.json` (Update
   Watch is a separate task, doesn't touch clients); take per-client `run_lock`.
2. **Sync inbox** across all sendboxes (§12): classify, split personal, suppress
   bounces/unsubs immediately.
3. **Pull tracking** from the worker (§11): record open/click activities (bot-filtered).
4. **Semantic triage + `apply-rules`** (§7.6): replies → deals/tasks; SLA sweep → nudge
   tasks; every stage change carries evidence.
5. **Follow-up advising (deal-aware, Stage 10):** replies → reply drafts; due-silent →
   value-add bumps → `pending_approval`.
6. **Load new pipeline** (cold/trigger campaigns, JIT buffer 3–7 days): priority pick →
   Tier-1 verify → Tier-2 enrich → step-1 draft → `pending_approval`. The drafting pass is
   **bounded by each campaign's `daily_quota`** (its daily draft budget): query
   `crm_store.py draft budget --campaign <slug>` and draft while `remaining > 0`. **Stop
   contract** (the loop is agent-driven): if the operator says "stop"/"ngưng" mid-loop, finish
   the current lead and halt — nothing already in `pending_approval` is lost; an unattended run
   drafts up to budget and stops.
7. **Send** `outbox/approved/` within quota (§10). (Approval happens in chat, any time.)
8. **Assisted channels:** draft SMS/Messenger for no-email contacts if the campaign
   allows + consent exists (§9/§16) → Today View copy buttons; human sends, reports back
   → activity.
9. **Compile Today View + regenerate kanban** (renderer).
10. **Reports:** daily ops + Approval Report + INTERNAL_REPORT; **Mondays** add the
    Weekly CRM Report; the **first daily run of a new month** additionally builds the prior
    month's **Monthly Client Report** (`crm_store.py monthly-report --month <prior YYYY-MM>`).
    Both the weekly and monthly reports are client-facing, through the scrub gate; their
    pipeline snapshot is point-in-time "as of report date".
11. **Notify Telegram** via WideCast `sendTelegramMessage`: counts + report link →
    `notification_log.md`.
12. **Stage 9 audit** → completion gates → release `run_lock`.

---

## 16. Compliance (encoded, not just prose)
- **CAN-SPAM:** physical address + working opt-out in every commercial email; honor
  opt-out (default 10 business days, we do it same-run); truthful subjects — step-1
  subjects must not begin `Re:`/`Fwd:` (linted in Stage 9 audit + pre-send); bumps must
  be real in-thread replies (truthful `Re:`).
- **Opt-out reach:** suppression checked at every send-capable path — initial, follow-up,
  assisted channels, and at import against ALL identities; unioned on merge; pending-merge
  contacts excluded from queues. If track-pull hasn't succeeded within N hours, sending
  for that box is blocked (so worker/mailto unsubs can't sit unhonored > window).
- **Assisted channels:** manual send reduces automation/platform-detection risk but does
  NOT change the legality of the solicitation. US SMS gated on documented consent
  `{optin_source, optin_at, evidence_activity_id}` or existing relationship; default SMS
  = inbound-initiated only; each assisted draft in Today View shows its legal basis. Zalo
  cold-messaging strangers stays off by default (Vietnam Decree 91/2020 + ToS).
- **Guessed email** policy per §9.6.

---

## 17. Testing (client "Max Output")
Fixture list `tests/fixtures/max_output_list.csv` (5 rows):
1. `huubinhnguyen81@gmail.com` — happy path: receive→(open)→click→reply positive→deal+task+follow-up draft.
2. `leadup@livechatwith.us` — own domain: MX check; no open/reply → bump due at day 4 (via injectable clock).
3. `tainguyenvdcc@gmail.com` — reply "unsubscribe" → suppression + close tasks + never re-drafted.
4. synthetic "Nguyen No-Email" (name + facebook URL only) — no-email flow: enrich fails
   to find email (controlled) → assisted messenger draft → human "sent" → reply pasted → activities.
5. synthetic `bounce-test@nonexistent-...invalid` — MX fail at verify; send re-check must
   **block** it (inverted assertion: send refused, draft stays pending). No live forced bounce.

**Injectable clock**, not backdating: the single `now_iso()` in `tools/storage/adapter.py`
(from which `today_str`/`month_str` derive) honors `OUTREACHCRM_FAKE_NOW`
(`YYYY-MM-DD` or `YYYY-MM-DDTHH:MM:SSZ`) **only when `OUTREACHCRM_TEST_MODE` is truthy** — so a
real scheduled run can never have its send timestamps or quota-day shifted by a stray env var
(`OUTREACHCRM_TEST_MODE` must never be set in a live automation task). A malformed value raises
loudly *while in test mode* and is inert otherwise. E2E advances time forward with it. **Built (2E).**

**Unit tests (pytest, offline, Gmail/SMTP/IMAP mocked):** import mapping+dedupe+ULID+idempotency;
email_verify MX ok/fail; **adapter contract suite parametrized over [json, postgres]** covering
all methods, every Cond op, atomicity, lock mutual-exclusion; crm_store CRUD + blocked invalid
stage transition + merge + resolve(); each rule (fixture events); suppression blocks send;
quota/warmup math + **ordered pre-send gate chain**; token HMAC sign/verify + link rewrite;
sync classifier over real MIME fixtures (reply, **DSN inside a sent thread**, DNS-fail DSN,
550 5.1.1, OOO, empty-body plus-alias unsub, personal); scrub gate; timezone inference;
multi-client isolation (2nd synthetic client: A invisible to B, client vs global suppression,
no cross-client quota); resync-from-backups + "update-watch writes nothing under clients/".

**Worker tests** (miniflare/vitest-pool-workers, D1 in-memory): valid token→302; tampered
sig→refused, no redirect; `/events` wrong Bearer→401; cursor pagination; `POST /u`→unsub;
`GET /u` does not mutate.

**E2E runbook** (`tests/E2E_RUNBOOK.md`, one real sendbox): mandatory **step 0 =
`crm_store.py reset-client max-output --confirm`** (wipes test client data +
`test_fixture`-tagged suppression + sendbox cursor + worker events by run prefix);
subjects/campaign slugs salted per run-id (avoid Gmail thread collisions); ~25 assertions;
open = soft assertion (design admits it's estimated), hard-assert click/reply/unsub.

---

## 18. Deploy script surgery (`deploy-outreachcrm.sh`)
- Hardcoded fallback API key **removed** (already stripped pre-commit; fail-closed).
- Remove unconditional `die` checks requiring collector/bridge/extension dirs, and their
  blocks in `run_static_checks`/`sync_and_zip_skills`.
- New concrete `GIT_REMOTE_URL`/`GIT_REMOTE_NAME`/`GIT_AUTHOR_*` for the outreachcrm repo
  (drop the `github.com-soloagency` SSH alias + soloagency author identity).
- Extend `scan_staged_secrets` regex: add `"refresh_token"\s*:`, `"client_secret"\s*:`,
  `TRACKER_API_KEY`, and block staging of `token.json`/`client_secret*.json`.
- `.gitignore`: add `sendboxes/*/token.json`, `secrets/`, `.dev.vars`, keep
  `provider_config.local.json`.
- Keep: `generate_load_manifest`, git-root safety, secret scan framework, skills zip.

## 19. Renderer scrub term list (`tools/report_renderer.py`, `CLIENT_BLIND_TERMS`)
This is the single source for the term list; `tools/report_renderer.py`, playbook 09's
prose list, and 09's mechanical scrub grep must all enumerate exactly this set.
**Keep** (must never reach a client): "OutreachCRM", "WideCast" (the notification provider's
name must never appear in a client report), "INTERNAL_REPORT", "MCP", "OpenAPI", "API key",
"PDNA", "Telegram", "automation", "scheduled task", "config file", "debug", "agent debug",
"provider_config", "Client tools", "global MCP". **Add** (OutreachCRM internal vocabulary):
"sendbox", "gmail_client", "crm_store", "storage_config", "trk.", "HMAC", "token.json",
"sent_log", "suppression", "warmup", "quota", "guessed". Natural-English terms (`quota`,
`automation`, `debug`, `guessed`, `warmup`, `suppression`) are matched with word boundaries so
innocent copy ("quotations", "home automation") does not false-positive; technical tokens keep
substring matching. On a scrub hit the recovery is to reword the flagged sentence and
re-render — never bypass the gate or hand-edit the blocked output. (The weekly and monthly
client reports are the only scrubbed, client-facing outputs; both pass this same gate.)

## 20. Update Watch (Stage 11) — rewritten scope
Diff scope + change-classification enum rebuilt around OutreachCRM components: storage
adapter/schema_version, `crm_store.py`/`gmail_client.py`/`import_leads.py`/`email_verify.py`,
`tracker/worker.js` + its `wrangler deploy` rerun step, sendbox token compat. Replace
`bridge_update_required`/`extension_reload_required` in `update_state.json` with
`tracker_worker_deploy_required`/`storage_schema_migration_required`. Repo-wide there must
be **no surviving `soloagency`/`solo-agency`/`github.com/soloagency` literal**, especially
in AGENTS.md's blocker-recovery clause (it fires on any blocker, not just explicit updates).

---

## 21. Build phases
- **Phase 0 (DONE):** clone+prune+rename; DESIGN.md; rewrite root playbook, AGENTS,
  both entrypoints, 00, 07, 09, 11, LOAD_LEDGER scrub; deploy surgery; renderer term list;
  README; LOAD_MANIFEST; scrub verification. 6-critic audit (48 findings) fixed.
- **Phase 1 (DONE):** storage adapter (`tools/storage/`, json backend) + `crm_store.py`
  (contacts/accounts/deals/activities/tasks/pipelines/suppression + idempotent rules engine +
  merge/resolve) + `import_leads.py` + `email_verify.py` + `gmail_client.py` (App Password
  SMTP/IMAP: auth/health/quota/send-with-gate-chain/sync-with-DSN-first-classifier) + playbooks
  02/03/08 + `tests/test_phase1.py` (27 stdlib unittest cases, all green). Manual core loop:
  a real @gmail.com send + reply-sync now works end to end. Open/click tracking and the
  OAuth/Workspace mode remain Phase 2+.
- **Phase 2 (in progress):** the intelligence + automation loop, ending at the Approval Report.
  Locked decisions: (1) tracker worker deferred to Phase 3 — MVP is @gmail.com `plain_text_mode`,
  measured by reply/bounce/unsub; (2) **no guessed-email path** — enrichment never guesses an
  address; drafts only ever target a found email that has a source (the `guessed_only` send gate
  stays as a safety net but nothing produces guessed addresses); (3) the daily run **auto-enriches
  the whole JIT batch**; (4) the daily run stops at the **Approval Report and waits** — the
  operator approves in chat, and only then does the send step run (approve-and-send is a separate
  operator-triggered pass, `approval_mode: manual_all`).
  Milestones: **2A ✅** campaigns + segments + enrich queue (05); **2B ✅** verify & enrich (04 +
  skill `email-verify-enrich`, dossier TTL + cross-campaign inheritance, no guessing); **2C ✅**
  goal-driven email writing (06 + skill `email-writing`) → drafts in `pending_approval`; **2D ✅**
  Approval Report render + chat-approve handler + follow-up/reply (10) + Today View/kanban (13/14);
  **2E ✅** Scheduled-Run wiring + injectable clock (`OUTREACHCRM_FAKE_NOW`/`TEST_MODE`) +
  composed WideCast `notify` (`provider_openapi.py notify`, dry-run + `local_path_only` degrade) +
  minimal client-facing `weekly-report` (scrub-gated) + expanded E2E acceptance runbook (~25
  assertions). **Phase 2 complete** — E2E runbook is the acceptance gate; Stages 12/15 + the
  tracker worker are Phase 3.
- **Phase 3:** playbook 12/15, kanban/timeline/**polished** weekly report/forecast/segments,
  the Cloudflare tracker worker (open/click) + OAuth/Workspace sending, Postgres adapter.
- **Phase 4:** Local Collector + lead-engine (Facebook enrichment) re-imported **+
  assisted-channel distribution (DM/comment) via the collector**. No voice channel is planned —
  non-email channels stay assisted (the agent drafts, the human sends by hand): SMS / Messenger / Zalo.
- **Phase 5 (optional):** local web UI over crm_store.

---

## 22. Interim Operating Rules (Phase-1 tools have shipped)

**Status update:** the Phase-1 runtime tools — `crm_store.py`, `gmail_client.py`,
`import_leads.py`, `email_verify.py`, and the storage adapter — **now exist and MUST be
used**. The Phase-0 degradations that let setup proceed before they existed (direct `crm/`
writes, the sendbox "pending_connectivity_check" path, "tool_not_built" skips for these
tools) **no longer apply to them**. Setup writes CRM records through `crm_store.py`, connects
sendboxes with `gmail_client.py auth`, and imports with `import_leads.py` for real. The
critical-violation rule for a direct `crm/` write is back in force.

What is still not built (Phase 2+): the stage FILES 04/05/06/10/12–15 and the
`email-verify-enrich` / `email-writing` skills, and the tracker worker. Those are covered by
R1 (they are stage files, not tools).

**R1 — Planned stage files & skills are not "missing files".** A Stage Map row marked
`status: planned` (or a skill not yet on disk) must NOT be treated as a truncated/404 read,
must NOT trigger GitHub re-fetch or Last-Resort Recovery. Load `docs/DESIGN.md` (the section
covering that stage) with its own LOAD LEDGER instead, and record `stage_file_pending` in
the run/setup progress block. This is the ONLY degradation still active for the core loop.

**R2 — A genuinely missing tool is an honest blocker, not a recovery trigger.** If a required
tool is absent (should not happen post-Phase-1 in a full checkout, but can if a partial
install lacks it), do NOT improvise a replacement script and do NOT enter Last-Resort
Recovery. Report it via one `**[ACTION REQUIRED]**` block, record `skipped: tool_not_built`,
and continue with steps that need no missing tool. This is a fallback, not the expected path.

**R3 — Migrating a Phase-0 install.** An install whose CRM records were created by the
Phase-0 direct-write path (logged `phase0_direct_write`) predates the identity reverse index,
so `find_by_identity` — and therefore dedupe — would miss them. After updating to a checkout
with the tools, run once per client:
`python3 tools/crm_store.py --client-dir <DIR> validate --rebuild-index`
which validates every `crm/` record against the Stage 7 schema and rebuilds
`contact_identities.jsonl` from the existing contacts. Stage 9 treats a logged
`phase0_direct_write` as compliant only until this migration runs; after it, all CRM writes
go through `crm_store.py`.

**R4 — When GitHub is unreachable, the local checkout is the source of truth.** The repo is
`github.com/soloagency/outreach`. If GitHub cannot be verified — the repo is not yet
published, `git ls-remote` fails, network/sandbox blocks it, or `OUTREACHCRM_GIT_REMOTE_URL`
is unset — the Fresh GitHub Source gate and Last-Resort Recovery must treat THIS local working
copy as the verified source: record `fresh_source_check: skipped_local_unreachable` in
`resync_log.md` (or `update_state.json` for updates), skip the clone/remote-verify/GitHub-fetch
steps, and continue. Do not block setup or a run on a GitHub check that cannot pass. Once the
repo is published and reachable, normal Fresh-Source verification against
`github.com/soloagency/outreach` resumes automatically.

**R5 — What a full setup reaches now.** With the Phase-1 tools present, a Setup Flow session
completes to `ready_for_automation_first_run` for real: profile + pipeline + campaign written
via `crm_store.py`, at least one sendbox authenticated with `gmail_client.py auth`, the first
list imported with `import_leads.py`, notification configured or declined, automation task
created. The first Daily Run performs real send/sync/enrich (no `tool_not_built` skips for the
core loop). Only the Phase-2 stage files (04/05/06/10) are still `status: planned` and follow R1.
```
