# Storage Schema And History

Stage: `07`

## Load Rule

Load when creating any folder or file, minting an id, saving a Client Intelligence Profile, importing a list, writing a contact/account/deal/activity/task, drafting or sending, appending to any log, reconciling a report, adding a client, or reading history. Any step that touches disk MUST first load this stage IN FULL.

This stage is the **constitution**: it defines every on-disk structure OutreachCRM is allowed to write, the storage adapter that all mutations must go through, and every record schema with its field enums. When any other playbook describes where or how something is stored, this file is authoritative; when this file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.

## Hard Gates For This Stage

- Use the dedicated data root `daily-content-pipeline/` (the shared Solo Agency per-client root). The toolkit/source is the `outreach/` module of the Solo Agency repo (`soloagency/outreach/`); it holds no client data.
- Use `client_profile_{client_slug}_{business_slug}_{location_slug}.md` (the Client Intelligence Profile) as the canonical profile. Never use a vague name such as `ABC.md`.
- **All CRM mutations go through `crm_store.py`.** Direct file writes to any `crm/` collection are a critical violation (inherited "no one-off scripts" rule). Reading raw JSON is allowed only for debugging.
- Every record carries `schema_version`, `id`, `created_at`, `updated_at`; every append-only log row carries `ts` and a monotonic `seq`.
- Slug rules and monthly `YYYY-MM/` folders apply to every daily/append artifact.
- This stage must be loaded IN FULL (LOAD LEDGER printed with `Verdict: PASS`, matching `LOAD_MANIFEST.md` when present) before any side-effect write. A partial read = NOT loaded. See `playbooks/LOAD_LEDGER_PROTOCOL.md`.

## Source Preservation Rule

This file is the detailed on-disk source material. Do not summarize away requirements, examples, checklists, schemas, field enums, protocols, URLs, edge cases, warnings, approval gates, or completion gates. A downstream agent may summarize its human-facing *response*, but it must still obey the full requirements in this file. If you cannot fit a schema, load the schema — do not reconstruct it from memory.

---

## 1. The On-Disk Layout (`daily-content-pipeline/`)

The toolkit/source is the `outreach/` module of the Solo Agency repo (`soloagency/outreach/`, no client data). Its data lives in the shared Solo Agency data root, `daily-content-pipeline/` (data/config/output only): OutreachCRM data lives in the `outreach/` subtree of the shared per-client workspace (`daily-content-pipeline/clients/{slug}/{business}_{location}/outreach/`) so it sits beside — and never collides with — Solo Agency's content data for the same client. This is the complete, authoritative tree. Nothing outside it may be created without amending this stage.

```text
soloagency/                          # Solo Agency monorepo (.git at this repo root)
  outreach/                          # this module (OutreachCRM toolkit/source), no client data
  ...                                # other Solo Agency modules (content pipeline, collector, …)
daily-content-pipeline/              # shared Solo Agency data root — data/config/output only
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
    outreach/                        # OutreachCRM's per-client subtree (sits beside content data)
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
        {client}-report_state.json
      outputs/latest/...
      integrations/providers/
        provider_config.local.json  provider_capabilities.json
        provider_openapi_cache.yaml  provider_calls.jsonl  provider_health.md
```

**Client-scope is structural, not disciplinary.** The storage adapter is instantiated per client rooted at `clients/{slug}/crm/`. Agency-tier collections — `daily-content-pipeline/suppression/global_suppression.jsonl`, `daily-content-pipeline/secrets/`, `daily-content-pipeline/provider_defaults.json`, and the tracker HMAC key — are the **only** things allowed to be global, and they are enumerated exactly here. A client rooted at `clients/A/` can never read or write `clients/B/`.

### Slug rules

- Use lowercase letters.
- Replace spaces with hyphens.
- Remove punctuation when possible.
- Keep slugs short but recognizable.

`client_slug`, `business_slug`, `location_slug`, `list_slug`, `campaign_slug`, `sendbox_slug`, and asset `{slug}` all follow these rules. Ids (`c_…`, `a_…`, `d_…`, `act_…`, `draft_…`) are minted, not slugged (see §3).

### Monthly organization rule (`YYYY-MM/`)

- Any file created daily or appended continuously must live under a `YYYY-MM/` folder (and, where dated per day, a `YYYY-MM-DD/` folder inside it).
- This applies to: `crm/activities/YYYY-MM/activities.jsonl`, `campaigns/*/queue/enriched/YYYY-MM-DD/`, `campaigns/*/outbox/pending_approval/YYYY-MM-DD/`, `campaigns/*/sent/YYYY-MM/sent_log.jsonl`, `campaigns/*/history/YYYY-MM/`, `inbox_sync/YYYY-MM/sync_log.jsonl`, `outputs/YYYY-MM/YYYY-MM-DD/`, `automation/backups/update_YYYY-MM-DD_HHMMSS/`, and `automation/issues/YYYY-MM-DD_{blocker_slug}.md`.
- Keep `outputs/latest/` as the stable pointer/copy set for the operator and (for the weekly report only) the client-facing handoff link. Keep report state beside the dated set as `outputs/YYYY-MM/YYYY-MM-DD/{client}-report_state.json`.
- Do not let long-running pipelines accumulate hundreds or thousands of daily files directly in one folder. Month partitioning is the mechanism.

---

## 2. Storage Adapter (pluggable JSON → Postgres)

`tools/storage/adapter.py` defines the interface; `tools/storage/json_adapter.py` is the default backend; `tools/storage/postgres_adapter.py` comes later and must pass the **same parametrized contract tests**. The backend in force is read from `daily-content-pipeline/storage_config.json`.

### 2.1 Interface

```text
get(collection, id) -> dict | None
put(collection, id, record) -> None                 # atomic (temp+rename), bumps updated_at
update(collection, id, mutate_fn) -> dict            # read-modify-write under the collection lock
delete(collection, id) -> None                       # rarely used; prefer tombstones
query(collection, where: [Cond], sort=None, limit=None, offset=None) -> [dict]
append(log, record) -> None                          # append-only, stamps ts + monotonic seq
read_log(log, since_seq=None, where=None) -> [dict]  # ordered by seq (backend-independent)
find_by_identity(kind, normalized_value) -> id | None  # backed by unique reverse index
reserve(sendbox_slug, day) -> token | None           # atomic quota reservation (see §6.3, §7.5 / Stage 8)
```

### 2.2 Cond DSL — deliberately small

- `Cond = (field, op, value)`, `op ∈ {=, !=, <, >, contains, in}`.
- **This DSL covers flat fields only.** Identity lookups do **NOT** use it — they use `find_by_identity` over a maintained unique reverse index (`contact_identities`). Do **not** claim Cond translates arbitrary nested-array matches to SQL; it does not.

### 2.3 Record invariants + schema upgrades

- Every record carries `schema_version`, `id`, `created_at`, `updated_at`.
- The adapter holds a per-collection `{from_version: fn}` upgrade registry, applied on read and persisted on the next write. A record is upgraded lazily; a migration (§2.6) forces all records current first.

### 2.4 JSON backend

- One file per record; logs are monthly JSONL.
- Atomic writes via temp+rename.
- Per-collection `fcntl` lockfile guards `update()` read-modify-write and mutual exclusion.
- A per-log counter file, incremented **under the log lock**, supplies the monotonic `seq`. This makes `read_log(since_seq=…)` deterministic and backend-independent.

### 2.5 Postgres backend

- Table per collection: `(client_id, id, payload jsonb, created_at, updated_at, <generated index cols>)`.
- `client_id` is **mandatory in every table and every generated WHERE** — this is how multi-client isolation survives the move off the filesystem.
- `contact_identities(client_id, kind, value UNIQUE, contact_id)` enforces identity uniqueness at the database level.
- Logs get a `seq bigserial`. Index columns are GENERATED from `payload` so the Cond DSL maps to real indexed columns.

### 2.6 Migration

- `crm_store.py migrate --to postgres` runs under a **storage freeze flag**.
- It upgrades all records to the current `schema_version` first.
- It verifies the move with **per-record content hashes** (not row counts) — count parity is not proof of a faithful copy.

### 2.7 The one write path

**All CRM mutations go through `crm_store.py`.** Direct file writes are a critical violation of the inherited "no one-off scripts" rule and are caught by the Stage 9 audit. Reading raw JSON for debugging is fine; writing it by hand is not. The adapter, not prose, is the source of truth for atomicity, locking, and `seq`.

---

## 3. Id Minting

| Record | Id form | Minted when |
|---|---|---|
| Contact (a.k.a. `lead_id`) | `c_` + ULID | at import (NOT a hash of email — email may be absent) |
| Account | `a_` + ULID | on first company/office reference |
| Deal | `d_` + ULID | when a rule creates a deal |
| Activity | `act_` + ULID | on every event append |
| Task | task id (ULID) | when a rule/human/agent creates a task |
| Draft | `draft_` + ULID | when a step is drafted into `pending_approval` |

`lead_id` **is** the contact's `id`; the two names refer to the same value (paths use `crm/contacts/{lead_id}.json`). A ULID is used because it is minted at import, before we know whether the person has a verifiable email — email-as-key would fail the email-optional model (§4.1).

---

## 4. CRM Data Model (`crm/…`)

Every collection below lives under `clients/{client_slug}/{business_slug}_{location_slug}/crm/` and is mutated only through `crm_store.py`.

### 4.1 Contact (`crm/contacts/{lead_id}.json`) — email is NOT required

`lead_id` = ULID minted at import. A contact may have zero verified emails (name + a Facebook URL is a valid contact that flows to assisted channels).

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
  "enrichment": { /* distilled copy of the dossier — see §7 */ },
  "assigned_sendbox": null,             // sticky sender, set on first send (see §6)
  "merge": {"status": "active|merged", "merged_into": null},
  "next_action": {"task_id": null}
}
```

Field notes:

- **`identities.emails[].status`** enum: `unverified | mx_ok | delivered | bounced | guessed_only | catch_all | email_not_found`. `is_primary` marks the address a send routes to. `source` ∈ `import | enrich | guess`.
- **`channels.email.status`** enum: `usable | needs_data | opted_out | bounced`. A contact with no verified email is `needs_data` and cannot be emailed; it may still be `usable` on an assisted channel if consent exists.
- **`channels.sms.status`** ∈ `needs_optin | usable | opted_out`, always `mode: assisted`; the `optin` object records `{source, at, evidence_activity_id}` (compliance basis, see DESIGN §16 — compliance is encoded in the send/import code, Stages 8/3).
- **`lifecycle_stage`** enum: `lead | engaged | opportunity | customer | evangelist | lost | do_not_contact`.
- **`tz`** feeds the send-window gate; inferred from state/area code.
- **`custom_fields`** keys are defined per client in the Client Intelligence Profile `custom_field_definitions` block (§8.1). Do not invent custom-field keys that the profile does not define.
- **`enrichment`** is a distilled copy of the dossier (§7.2); the canonical dossier lives under `campaigns/*/queue/enriched/`.
- **`assigned_sendbox`** is null until the first send, then fixed (sticky sender, §6.2).
- **`merge`** carries the tombstone pointer (§4.7).

### 4.2 Account (`crm/accounts/{account_id}.json`)

A company/office (e.g. a brokerage). Contacts point up at their account.

```json
{
  "id": "a_...", "schema_version": 1, "created_at": "", "updated_at": "",
  "name": "", "domain": "", "type": "brokerage|firm|team|independent|other",
  "location": {"city": "", "state": "", "country": ""},
  "contact_ids": [], "custom_fields": {}, "tags": []
}
```

Minimal required set (DESIGN §7.2): `{id, name, domain, type, location, contact_ids[], custom_fields}`. `domain` is used to group contacts, to detect same-company clustering, and to apply the guessed-email per-domain kill switch.

### 4.3 Deal (`crm/deals/{deal_id}.json`)

```json
{"id":"d_...","schema_version":1,"name":"","contact_ids":[],"account_id":"",
 "pipeline":"default_sales","stage":"new_reply","value":0,"currency":"USD","probability":0.1,
 "expected_close":"","source_campaign":"",
 "stage_history":[{"stage":"new_reply","at":"","by":"rule:r1","evidence_activity_id":""}],
 "status":"open|won|lost","lost_reason":null,"next_action":{"task_id":null}}
```

- `pipeline` names a pipeline id from `pipelines.json`; `stage` is a stage id inside it.
- `probability` is copied from the stage definition; `value` × `probability` drives forecast (Stage 15).
- **`stage_history`** is append-only; every entry carries `at`, `by` (`rule:{id}` / `human` / `agent`), and an `evidence_activity_id` pointing at the activity that justified the change. A stage change with no evidence is a Stage 9 audit failure.
- `status` ∈ `open | won | lost`; `lost_reason` is required when `status = lost`.

### 4.4 Activity (`crm/activities/YYYY-MM/activities.jsonl`) — append-only, the event backbone

The activity log is the spine of the whole CRM. A contact timeline is this log filtered by `contact_id` (following merge chains via `resolve()`). It is append-only and monthly-partitioned; each row has a monotonic `seq`.

```json
{"seq":123,"ts":"","id":"act_...","contact_id":"","deal_id":null,
 "type":"email_sent|email_reply|email_open|email_click|email_bounce|unsubscribe|call|meeting|note|stage_change|task_done|enriched|imported|merged|assisted_sent",
 "summary":"","ref":{"message_id":"","url":"","path":""},"by":"agent|human|rule"}
```

- **`type`** enum (complete): `email_sent | email_reply | email_open | email_click | email_bounce | unsubscribe | call | meeting | note | stage_change | task_done | enriched | imported | merged | assisted_sent`.
- **`by`** ∈ `agent | human | rule`.
- `ref` carries whichever of `{message_id, url, path}` applies (e.g. `rfc_message_id` for a send, the tracker URL for a click, the sent_log path for a send).
- Open/click activities are recorded but **never alone trigger a stage change or auto-action** — only a reply is conversion evidence (DESIGN §11).

### 4.5 Task (`crm/tasks/tasks.jsonl`)

```json
{"id":"","schema_version":1,"contact_id":null,"deal_id":null,"title":"","due_at":"",
 "status":"open|done|cancelled","created_by":"rule|human|agent","guard_key":""}
```

Minimal required set (DESIGN §7.5): `{id, contact_id?, deal_id?, title, due_at, status, created_by, guard_key}`. `status` ∈ `open | done | cancelled`; `created_by` ∈ `rule | human | agent`. `guard_key` makes rule-created tasks idempotent (§4.6). The Task engine, SLA sweep, and Today View live in Stage 14.

### 4.6 Pipelines + rules (`crm/pipelines.json`)

Stages carry `probability` + `sla_days`. Rules are **deterministic**, executed by `crm_store.py apply-rules`, and are **never improvised by the LLM**.

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

**Guard keys** `(rule_id, trigger_activity_id)` make `apply-rules` idempotent and re-runnable — re-processing the same activity does not double-create deals or tasks. Stage transitions are validated against the pipeline's stage list; an invalid transition is blocked in code (Stage 13). The full rule engine and stage rules live in Stage 13.

### 4.7 Contact identities reverse index (`crm/contact_identities.jsonl`)

The unique reverse index that backs `find_by_identity` and deterministic merge. One row per identity; `(kind, value)` is unique within the client.

```json
{"kind":"email|phone|social|website","value":"<normalized>","lead_id":"c_...","added_at":""}
```

- Normalization is fixed per kind: email lowercased/trimmed; phone → E.164; social → canonical profile URL; website → host without scheme/`www.`
- A collision (two contacts claiming the same normalized identity) is exactly what triggers auto-merge (§4.8). In Postgres this is `contact_identities(client_id, kind, value UNIQUE, contact_id)`.

### 4.8 Merge / tombstone / resolve (deterministic)

- **Auto-merge** on exact email / E.164 phone / canonical social URL match.
- **Fuzzy** name+company → *propose* only; a human approves. Contacts with a **pending merge proposal are excluded from every campaign queue** until resolved.
- The **losing record becomes a permanent tombstone** — never deleted:
  ```json
  {"merge": {"status": "merged", "merged_into": "c_SURVIVOR"}}
  ```
- On merge, identities, channel statuses, and suppression are **unioned** into the survivor.
- Every `lead_id` lookup path — the sync classifier, track-pull, unsub handler, `apply-rules`, and drafting — calls `resolve(lead_id)` to follow merge chains to the active survivor. Skipping `resolve()` anywhere is a correctness bug; the audit checks for it.

### 4.9 Segments (`crm/segments.json`)

Named audiences, referenced by campaign `audience.segment` and by rule `r5` (`enroll_segment(customers)`).

```json
{"segments":[
  {"id":"customers","name":"Customers","kind":"dynamic",
   "definition":{"where":[["lifecycle_stage","=","customer"]]},
   "static_member_ids":[],"created_at":"","updated_at":""}]}
```

- `kind: dynamic` segments are computed from `definition.where` (a Cond list, §2.2); `kind: static` segments enumerate `static_member_ids`.
- Segment ids referenced by a Client Intelligence Profile `icp.segments[].segment_id` must exist here.

### 4.10 Suppression (two tiers)

Suppression is checked at **every send-capable path** and at **import against ALL identities**; it is unioned on merge; pending-merge contacts are excluded from queues (DESIGN §16). There are exactly two tiers:

- **Agency tier** — `daily-content-pipeline/suppression/global_suppression.jsonl` (global; applies to all clients).
- **Client tier** — `crm/suppression.jsonl` (this client only).

Both are append-only JSONL with a monotonic `seq`. A send is blocked if the recipient matches **either** tier. Record shape:

```json
{"seq":1,"ts":"","tier":"global|client","match":{"kind":"email|phone|social|domain","value":"<normalized>"},
 "reason":"unsubscribe|hard_bounce|reply_negative|remove_intent|manual|complaint|guessed_domain_kill",
 "scope":"all_clients|{client_slug}","source_activity_id":"","added_by":"agent|human|rule","tags":[]}
```

- **`reason`** enum: `unsubscribe | hard_bounce | reply_negative | remove_intent | manual | complaint | guessed_domain_kill`. `guessed_domain_kill` is the per-domain kill switch — the first hard bounce on a guessed pattern at domain X suppresses all other guessed addresses at X (DESIGN §9.6).
- **`match.kind`** ∈ `email | phone | social | domain`. A `domain` match suppresses every address at that host (used by the guessed kill switch and by existing-customer or do-not-contact-domain carve-outs from the profile suppression policy).
- **`tags`** may include `test_fixture` so the E2E `reset-client` step can wipe only test-created suppression (DESIGN §17).
- The pre-send re-check order (Stage 8) is: `resolve(lead)` → global + client suppression → `channels.email.status` → atomic quota reservation → warmup cap → domain cap → send-window → guessed cap → sequence-freeze → subject lint. Suppression is checked before quota is reserved so we never burn quota on a suppressed contact.

---

## 5. Lists & Import (`lists/{list_slug}/…`)

A list is one imported source file. Import maps columns, dedupes against existing contacts (via `find_by_identity`), mints `lead_id` ULIDs, and is idempotent (re-importing the same file does not double-create). Suppression is checked at import against ALL identities.

### 5.1 `lists/{list_slug}/list_manifest.json`

```json
{
  "schema_version": 1,
  "list_slug": "",
  "source_file": "",
  "source_format": "csv|txt|xlsx",
  "imported_at": "",
  "idempotency_key": "",
  "column_mapping": {"email": "Email", "full_name": "Name", "company": "Company", "phone": "Phone", "website": "Website"},
  "row_count": 0,
  "contacts_created": 0,
  "contacts_matched_existing": 0,
  "suppressed_at_import": 0,
  "rows_skipped": 0,
  "notes": ""
}
```

- `idempotency_key` is derived from the source file content + mapping; a second import with the same key is a no-op.
- `source_format` ∈ `csv | txt | xlsx`.

### 5.2 `lists/{list_slug}/leads.jsonl`

Append-only record of each imported row and what became of it (contact created / matched existing / suppressed / skipped), so an import is auditable and re-runnable.

```json
{"seq":1,"ts":"","raw":{"Email":"","Name":"","Company":""},"normalized":{"email":"","full_name":"","company":""},
 "outcome":"created|matched|suppressed|skipped_invalid","lead_id":"c_...","reason":""}
```

### 5.3 `lists/{list_slug}/import_log.md`

Human-readable import summary (one table row per import run): date, agent, source file, rows in, created, matched, suppressed, skipped, blockers. Surface any blocker with the `[ACTION REQUIRED]` contract (one purpose, one exact next step, one path); say `No action required right now.` when the import is clean.

---

## 6. Sendboxes (`sendboxes/…`) & multi-sendbox rotation

### 6.1 `sendboxes/sendboxes.json`

```json
{"sendboxes":[
  {"slug":"sb-a","auth_mode":"app_password|oauth","email":"...","domain":"gmail.com",
   "quota_today":40,"warmup_stage":"week_1|week_2|mature","status":"healthy|needs_reauth|paused",
   "historyId":null,"imap_uid_cursor":null,"last_successful_sync_ts":""}]}
```

- **`auth_mode`** ∈ `app_password | oauth`; **`warmup_stage`** ∈ `week_1 | week_2 | mature`; **`status`** ∈ `healthy | needs_reauth | paused`.
- Cursors: `historyId` (OAuth) or `imap_uid_cursor` (app_password), plus `last_successful_sync_ts`. Inbound sync (Stage 10) advances these.

### 6.2 Credentials (gitignored, `chmod 600`)

Per-sendbox secrets live beside the box and are **never committed**:

- `sendboxes/{sendbox_slug}/credentials.json` — App Password or OAuth client reference.
- `sendboxes/{sendbox_slug}/token.json` — OAuth token (mode `oauth` only).

Both are gitignored and `chmod 600`. The deploy script blocks staging of `token.json` / `client_secret*.json` and secret-scans the staged diff (`refresh_token`, `client_secret`, `TRACKER_API_KEY`). Do not put these values in any log, report, or profile.

### 6.3 Rotation, sticky sender, caps (the storage-visible rules)

- **Two auth modes, one interface.** `app_password` (priority for `@gmail.com`): SMTP send + IMAP read via Python stdlib (`smtplib`/`imaplib`), no OAuth, no 7-day expiry, preserves our Message-ID. `oauth` (Workspace/custom domain): Gmail API, scopes `gmail.send + gmail.readonly` only (drop `gmail.modify`); the OAuth app should be **Internal** to avoid the 7-day refresh-token expiry — if forced External/testing, weekly re-auth becomes a scheduled day-6 `[ACTION REQUIRED]`, not an error path.
- **Rotation is step-1 only; sticky sender thereafter.** First outreach picks the healthy referenced sendbox with the lowest `sent_today/quota_today` ratio (round-robin on ties); `contact.assigned_sendbox` is then fixed. Every bump/reply goes from the assigned box (threading + reply routing + anti-spam require it).
- **Two-tier cap.** Effective cap = `min(remaining_box_quota, remaining_domain_cap)` — several boxes on one domain share domain reputation; domain volume ramps too.
- **Broken box:** dropped from step-1 rotation; its assigned pending follow-ups **wait** (never reassigned) + `[ACTION REQUIRED]` re-auth; report shows "N follow-ups blocked".
- **Consumer `@gmail.com` limits (documented, accepted):** From is gmail.com → tracking links live on an unrelated domain → default `plain_text_mode` (no pixel, no link rewrite), measure by reply; no custom Message-ID domain; ~20–50 cold/day/box; never the operator's primary Gmail; App Password requires 2FA. Keep OAuth mode available as fallback.

The atomic quota reservation (`reserve(sendbox_slug, day)`) and the ordered pre-send gate chain are implemented in `gmail_client.py send` (Stage 8) — not in playbook prose.

---

## 7. Campaigns, Enrichment & Drafts (`campaigns/{campaign_slug}/…`)

### 7.1 `campaign_config.json` — the goal is the writing blueprint, not a label

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

Field notes:

- **`goal.goal_type`** enum: `book_meeting | get_reply | direct_sale | reactivation | nurture_upsell | event_invite`. This drives the email structure (Stage 6): `book_meeting`→short, one time-bound CTA; `get_reply`→ends with a question, no link; `direct_sale`→value + one offer link (the only place click tracking is on by default); `reactivation`→evidence of prior relationship + "still doing X?"; every final step→breakup.
- **`goal.proof_points[]`** = `{claim, evidence_url}` — evidence-backed; a draft may only cite proof that has an `evidence_url`.
- **`goal.cta.type`** ∈ `reply_yes | link | calendar`.
- **`goal.success_event`** wires straight into the rules engine (§4.6): `{on: reply_positive, create_deal_stage: new_reply}`.
- **`audience.personalization.min_confidence`** gates drafting (≥0.7 High); `no_hook_fallback` ∈ `skip | generic_honest_opener` (default `skip` — a hookless step-1 draft is rejected unless the campaign explicitly opts into the generic opener).
- **`sequence[].tracking`** default `plain_text`; a bump/reply threads on the prior send.
- **`approval_mode`** default `manual_all` even for bumps — nothing leaves without an explicit chat approval (§9).
- **`sendboxes[]`** references `sendbox_slug`s from `sendboxes.json`; `daily_quota` is the campaign's share.
- **`guardrails.no_fake_re`** enforces the truthful-subject rule (step-1 subjects must not begin `Re:`/`Fwd:`).

### 7.2 Dossier (`campaigns/{campaign_slug}/queue/enriched/YYYY-MM-DD/{lead_id}.json`)

The dossier belongs to the **contact** (client-scope); a distilled copy lands in `contact.enrichment`. Campaigns reference `lead_id`; the enrich queue is client-level, deduped by `lead_id`. The email-writing skill consumes `writing_brief`, not raw data.

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

- **`identity.still_active`** ∈ `confirmed | inactive | unknown`; `inactive/unknown` stops enrichment before hooks.
- **`hooks[].type`** enum: `new_listing | social_post | review | award | market_view | website_update`.
- **`hooks[].analysis.sensitivity`** ∈ `public_business | personal`. Etiquette hard rule: `public_business` signals are fair game; `personal` signals (family, health, vacations, children) are **default-banned from email copy** and go only into `do_not_mention`.
- **`hooks[].used_in`** tracks `["campaign/step"]`; a second campaign may not open with a hook already used on that person.
- **`writing_brief.personalization_confidence`**: ≥0.7 High, 0.4–0.7 Review carefully, <0.4 → `no_hook_fallback`.
- A draft may contain **only** details present in the dossier with an `evidence_url`; the Stage 9 audit checks this mechanically. TTL tiers (durable ~90d identity/context vs fresh 7–14d hooks), the negative cache (`email_not_found`, `no_verifiable_hook`), and the freshness gate live in Stage 4.

### 7.3 Enrich queue (`campaigns/{campaign_slug}/queue/enrich_queue.jsonl`)

Client-level intent to enrich, deduped by `lead_id` (one job even if two campaigns want the same person). Append-only with `seq`.

```json
{"seq":1,"ts":"","lead_id":"c_...","tier":"verify|profile|refresh",
 "requested_by":["{campaign_slug}/step1"],"status":"queued|in_progress|done|failed|inactive",
 "result":"dossier_written|no_hook|email_not_found|inactive|refreshed","dossier_path":"","started_at":"","completed_at":""}
```

- `tier` ∈ `verify | profile | refresh`; `status` ∈ `queued | in_progress | done | failed | inactive`.

### 7.4 Drafts (`campaigns/{campaign_slug}/outbox/…`)

A draft is written to `outbox/pending_approval/YYYY-MM-DD/{draft_id}.json` and, once approved in chat (§9), moved to `outbox/approved/{draft_id}.json` and sent in-session. Editing the HTML report never persists — chat is the write path. The `{draft_id}` path token **is** the record's `id` value (a `draft_`+ULID, §3); the record keys it as `id` and carries `created_at`/`updated_at` per the record invariant (§2.3).

```json
{"id":"","schema_version":1,"created_at":"","updated_at":"","lead_id":"c_...","campaign_slug":"","step":1,
 "sendbox":"sb-a","to":"","subject":"","body_text":"","body_html":"",
 "confidence_band":"high|review_carefully",
 "hooks_used":[{"type":"","evidence_url":""}],
 "tracking":"plain_text|pixel_and_links",
 "warnings":["guessed_email","generic_opener","bump_step"],
 "guessed_approved":false,
 "status":"pending_approval|approved|rejected|hold|sent|blocked",
 "decided_at":"","decided_by":"","reject_reason":"","blocker":""}
```

- **`confidence_band`** ∈ `high | review_carefully` — drives which section of the Approval Report the card lands in.
- **`guessed_approved`** must be `true` before a `guessed_only` address may send; the flag is enforced **in `gmail_client.py send`** plus a daily guessed-send cap, never only in prose.
- **`status`** ∈ `pending_approval | approved | rejected | hold | sent | blocked`. A send error returns the draft to `approved` with a `blocker` (Stage 8).

### 7.5 Sent log (`campaigns/{campaign_slug}/sent/YYYY-MM/sent_log.jsonl`)

Append-only, monthly, `seq`-stamped. Written by `gmail_client.py send` after a successful send; also holds the `send_reserved` marker appended under the sent_log lock during atomic quota reservation (so there is no count-then-send race).

```json
{"seq":1,"ts":"","lead_id":"","campaign":"","step":1,"sendbox":"","provider_id":"","thread_id":"",
 "rfc_message_id":"","token":"","links":{},"sent_at":""}
```

- `ts` is the append-time stamp; `append()` stamps `ts` + the monotonic `seq`, while `sent_at` is the domain timestamp of the actual send — both are present per the append-only log invariant (§2.4 / this stage's Hard Gates).
- `rfc_message_id` is the on-the-wire Message-ID (our own only when we control the domain; otherwise fetched after send). Bumps/replies thread via `In-Reply-To`+`References` off the prior `rfc_message_id`.
- `token` is the tracker token; `links` is the map of rewritten click tokens. Track-pull accepts only click URLs matching this stored `links{}` (injection defense, DESIGN §11–12; Stages 10/12).

### 7.6 Campaign history (`campaigns/{campaign_slug}/history/YYYY-MM/`)

- **`campaign_log.md`** — one row per send/step event for human review: date, `lead_id`, step, sendbox, subject, confidence band, tracking mode, outcome, notes.
- **`reply_log.md`** — one row per inbound reply tied to this campaign: date, `lead_id`, triage class (`positive|question|objection|negative|remove_intent`), action taken (deal/task/reply draft/suppress), evidence activity id.

---

## 8. The Client Intelligence Profile

Each client pipeline has exactly one Client Intelligence Profile file:

```text
client_profile_{client_slug}_{business_slug}_{location_slug}.md
```

It is the canonical, human-and-agent-readable brief that every draft is built from: draft = **client profile** (voice, offer, compliance, sending identity) + **campaign goal** (objective, CTA, proof) + **contact dossier** (hooks + evidence) + **step intent**. It keeps the inherited **value / status / rationale** discipline on every asserted field, so confidence and provenance travel with each fact. Slugged filenames provide uniqueness across multi-client folders; the schema name is always "Client Intelligence Profile."

### 8.0 Setup Flow note

The profile is authored in **Setup Flow**, which is the control plane: it creates config + the automation task and **never sends an email, never enriches for send, never runs a campaign**. The profile's terminal setup state is `ready_for_automation_first_run` (Metadata `status`). It flips to `active` on/after the first automation run. Any post-setup change to this profile triggers **Automation Resync** (the `automation_sync` block below plus `resync_log.md` + `automation_manifest.md`), verified by a dry-read before it is called complete.

### 8.1 Template

```md
# Client Intelligence Profile: {client_name}

## Metadata

- client_name:
- client_slug:
- business_slug:
- location_slug:
- created_date:
- last_reviewed_date:
- status: needs_setup | ready_for_automation_first_run | active | paused | archived

## business_description

value:
status:
rationale:

## industry

value:
status:
rationale:

## sub_industry

value:
status:
rationale:

## offer

status:
rationale:
items:
- name:
  description:
  price_point:
  fulfillment:
  primary: true | false

## icp   # ideal customer profile — who campaigns target

status:
rationale:
firmographics:
  company_types:
  company_size:
  industries:
  geography:
roles:
  titles:
  seniority:
  departments:
disqualifiers:
segments:
- name:
  segment_id:            # must exist in crm/segments.json
  definition_summary:
  priority: high | medium | low

## value_prop

value:
status:
rationale:
supporting_points:
-

## proof_points   # claims we may cite; each needs evidence

status:
rationale:
items:
- claim:
  evidence_url:
  strength: strong | medium | weak
  usable_in: cold_open | body | breakup

## sending_identity

status:
rationale:
from_name:
from_title:
reply_to:
signature_block:
physical_mailing_address:      # REQUIRED — appears in the CAN-SPAM footer of every commercial email
sending_domains:
- domain:
  purpose: primary | variant
sendboxes:                     # references sendboxes/sendboxes.json slugs
- slug:
  email:
persona_notes:

## target_triggers   # what qualifies a lead; seeds JIT pipeline + hooks

status:
rationale:
items:
- trigger: new_listing | social_post | review | award | market_view | website_update | job_change | funding | event | list_membership | manual
  why_it_qualifies:
  freshness_ttl_days:
  maps_to_hook_type: new_listing | social_post | review | award | market_view | website_update
  default_goal_type: book_meeting | get_reply | direct_sale | reactivation | nurture_upsell | event_invite

## brand_voice

value:
status:
rationale:
do:
-
dont:
-

## language

human_report_language:
recipient_language:
status:
rationale:

## custom_field_definitions   # defines the keys allowed in contact/account/deal custom_fields

status:
items:
- key:
  label:
  applies_to: contact | account | deal
  type: text | number | enum | date | bool
  allowed_values:              # for type=enum
  required: true | false
  description:

## suppression_policy

status:
client_suppression_file: crm/suppression.jsonl
honor_global_suppression: true
never_contact_domains:         # added to client suppression as domain matches
-
notes:                         # e.g. existing-customer carve-outs, do-not-contact domains

## compliance_notes

value:
status:
rationale:
can_spam_physical_address_present: true | false
opt_out_honor_window: same_run
sms_posture: inbound_initiated_only | documented_consent_required | off
zalo_cold_messaging: off
negative_topics:
-
do_not_mention:
-

## automation_sync

status: current | needs_resync | automation_prompt_update_pending | partial | blocked
last_profile_change_at:
last_profile_change_summary:
last_resynced_at:
last_resynced_by_agent:
automation_manifest_file: daily-content-pipeline/automation/automation_manifest.md
scheduled_prompt_file: daily-content-pipeline/automation/scheduled_run_prompt.md
schedule_file: daily-content-pipeline/schedule.md
native_task_name: {client_name} - OutreachCRM Daily Run
native_task_prompt_updated: true | false | not_applicable | unknown
dry_read_verification:
  verified_at:
  result: pass | fail | partial
  scheduled_run_will_see:
  blockers:

## assumptions

-

## human_corrections

-
```

### 8.2 Field discipline

- Every `value / status / rationale` field records **what we believe, how sure we are, and why**. `status` should read like `confirmed | inferred | assumed | needs_human` so downstream drafting knows what it may lean on. Do not assert a `value` with no `rationale`.
- **`sending_identity.physical_mailing_address` is required** — it is the CAN-SPAM footer address. A commercial campaign cannot be marked ready while `can_spam_physical_address_present` is `false`; surface it as `[ACTION REQUIRED]`.
- **`custom_field_definitions`** is the *only* place custom-field keys are declared. `contact.custom_fields`, `account.custom_fields`, and deal custom fields must use keys defined here (with the declared `type`/`allowed_values`).
- **`target_triggers`** seed the JIT pipeline (which cold/trigger leads to load 3–7 days ahead) and constrain which hook types a campaign may open on.
- **`icp.segments[].segment_id`** must resolve to an id in `crm/segments.json`.
- **`automation_sync`** is the per-client half of Automation Resync; the agency-wide half is `automation/automation_manifest.md` + `resync_log.md`. `native_task_name` pins the one automation task for this client; that task's prompt pins `target_client_slug` and must not touch another client.

---

## 9. Approvals & Analytics (per-client)

### 9.1 `approvals/approval_log.md`

Every approval decision (from the chat approval grammar — `approve`, `reject`, `edit`, `hold`) is logged. Nothing leaves without an explicit `approve`.

```md
# Approval Log

| Date | Agent | Draft ID | Lead ID | Campaign | Step | Decision | Reason / Edit | Sent At | Notes |
|---|---|---|---|---|---|---|---|---|---|
| 2026-07-15 | Claude | draft_01J... | c_01J... | austin-sellers-q3 | 1 | approved | — | 2026-07-15T15:12Z | High confidence |
| 2026-07-15 | Claude | draft_01K... | c_01K... | austin-sellers-q3 | 1 | rejected | hook stale, listing sold | — | Reason fed to learning_log |
```

Decision enum: `approved | rejected | edited | hold`. Rejection reasons feed `analytics/learning_log.md`.

### 9.2 `analytics/metrics_log.md`

Rolling record of campaign performance with **honest metric labels**: reply / bounce / unsubscribe are exact; **open is estimated** (Gmail image proxy, Apple MPP prefetch, image blocking); click is fairly reliable after bot filtering. Reports must label opens "estimated."

```md
# Metrics Log

| Date | Campaign | Step | Sent | Replies (exact) | Positive | Bounces (exact) | Unsub (exact) | Opens (est.) | Clicks | Guessed Sent | Guessed Bounces | Notes |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
```

The guessed cohort bounce rate is reported separately (its own columns) so it never contaminates the verified cohort's numbers.

### 9.3 `analytics/learning_log.md`

Captures what the batch taught us — rejected-draft reasons, hook types that landed vs flopped, subject patterns, cohort bounce signals — so the next drafting pass improves. Free-form dated entries plus a signal table.

### 9.4 `inbox_sync/YYYY-MM/sync_log.jsonl`

Append-only record of every inbound message the sync classifier processed (Stage 10), monthly and `seq`-stamped. The classifier runs in a fixed, load-bearing order (DSN → auto-reply → unsub alias → thread match → contact message → personal).

```json
{"seq":1,"ts":"","sendbox":"sb-a","message_uid":"",
 "classification":"dsn_hard|dsn_soft|auto_reply|unsub_alias|campaign_reply|contact_message|personal",
 "lead_id":"c_...","rfc_message_id_matched":"",
 "triage":"positive|question|objection|negative|remove_intent|untriaged|na",
 "action":"suppressed|deal_created|reply_drafted|counted_only|frozen_sequence|none","by":"agent"}
```

- **`classification`** enum: `dsn_hard | dsn_soft | auto_reply | unsub_alias | campaign_reply | contact_message | personal`.
- **Personal email is `counted_only`** — do not store body, do not deep-read.
- **`triage`** enum: `positive | question | objection | negative | remove_intent | untriaged | na`. `negative`/`remove_intent` (even without the word "unsubscribe") → suppression. **Any inbound reply freezes the remaining sequence** for that contact until triage completes — enforced in code at both draft-time and send-time.

### 9.5 `assets/`

- `asset_index.md` — index of generated proposals/flyers with version + path + status.
- `proposals/{slug}/v001/…`, `flyers/{slug}/v001/…` — versioned asset folders (never overwrite a version; bump `vNNN`).

### 9.6 `reports/YYYY-MM_report.md`

Rolling monthly operator narrative for the client (pipeline movement, notable replies, deals, blockers). This is the operator's month-view, distinct from the dated HTML outputs and the weekly client-facing report.

---

## 10. Outputs & the Report Reconciliation Ledger (`outputs/…`)

Two lanes, one rule: **only the weekly client report is client-facing** (scrubbed through the Client-Blind Scrub Gate, generated Mondays). Every other output is **operator-only and NOT scrubbed**. All HTML is rendered by `tools/report_renderer.py` (stdlib only).

Dated set under `outputs/YYYY-MM/YYYY-MM-DD/`:

| File | Lane | Scrubbed? |
|---|---|---|
| `{client}-approval-report.html` | operator | no |
| `{client}-today-view.html` | operator | no |
| `{client}-daily-ops.html` | operator | no |
| `{client}-INTERNAL_REPORT.html` | operator | no |
| `{client}-weekly-client-report.html` | **client-facing** | **yes (scrub gate)** |
| `{client}-weekly-client-report.pdf` | **client-facing** | **yes** |
| `{client}-monthly-client-report.html` | **client-facing** | **yes (scrub gate)** |
| `{client}-monthly-client-report.pdf` | **client-facing** | **yes** |
| `{client}-report_state.json` | ledger | — |

The monthly report is built on the first run of a new month for the prior calendar month
(`crm_store.py monthly-report --month <prior YYYY-MM>`), same scrub gate and shape as the weekly.
`outputs/latest/` holds the stable pointer/copy of each (the weekly/monthly client reports + PDFs are the client handoff links; the rest are operator convenience copies). Anything in `latest/` that is operator-only must be clearly labeled `INTERNAL_REPORT — Not for client sharing`.

### 10.1 `outputs/YYYY-MM/YYYY-MM-DD/{client}-report_state.json`

The reconciliation ledger. It records every output path + status, the scrub gate result for the weekly report, PDF status, and the run's reconciled counts, so a later pass cannot silently overwrite or contradict an earlier artifact and so counts stay consistent across the sent log, activities, approval log, sync log, deals/tasks, notification log, and the `latest/` copies.

```json
{
  "client_slug": "",
  "run_id": "",
  "report_date": "",
  "report_dir": "outputs/YYYY-MM/YYYY-MM-DD/",
  "is_weekly_report_day": false,

  "approval_report_html_path": "outputs/YYYY-MM/YYYY-MM-DD/{client}-approval-report.html",
  "today_view_html_path": "outputs/YYYY-MM/YYYY-MM-DD/{client}-today-view.html",
  "daily_ops_html_path": "outputs/YYYY-MM/YYYY-MM-DD/{client}-daily-ops.html",
  "internal_report_html_path": "outputs/YYYY-MM/YYYY-MM-DD/{client}-INTERNAL_REPORT.html",
  "weekly_client_report_html_path": "outputs/YYYY-MM/YYYY-MM-DD/{client}-weekly-client-report.html",
  "weekly_client_report_pdf_path": "outputs/YYYY-MM/YYYY-MM-DD/{client}-weekly-client-report.pdf",

  "latest_approval_report_html_path": "outputs/latest/{client}-approval-report.html",
  "latest_today_view_html_path": "outputs/latest/{client}-today-view.html",
  "latest_daily_ops_html_path": "outputs/latest/{client}-daily-ops.html",
  "latest_internal_report_html_path": "outputs/latest/{client}-INTERNAL_REPORT.html",
  "latest_weekly_client_report_html_path": "outputs/latest/{client}-weekly-client-report.html",
  "latest_weekly_client_report_pdf_path": "outputs/latest/{client}-weekly-client-report.pdf",

  "approval_report_status": "pending",
  "today_view_status": "pending",
  "daily_ops_status": "pending",
  "internal_report_status": "pending",
  "weekly_client_report_status": "not_due",
  "client_facing_scrub_status": "not_due",
  "client_facing_scrub_blocker": "",
  "weekly_pdf_status": "not_due",
  "weekly_pdf_blocker": "",

  "counts": {
    "inbox_synced": 0,
    "replies_positive": 0,
    "replies_question": 0,
    "replies_objection": 0,
    "replies_negative": 0,
    "remove_intent": 0,
    "hard_bounces": 0,
    "soft_bounces": 0,
    "unsubscribes": 0,
    "opens_estimated": 0,
    "clicks": 0,
    "drafts_created": 0,
    "drafts_high_confidence": 0,
    "drafts_review_carefully": 0,
    "drafts_approved": 0,
    "drafts_rejected": 0,
    "emails_sent": 0,
    "guessed_sent": 0,
    "guessed_bounces": 0,
    "assisted_drafts": 0,
    "assisted_sent_by_human": 0,
    "deals_created": 0,
    "deals_advanced": 0,
    "deals_won": 0,
    "deals_lost": 0,
    "tasks_created": 0,
    "tasks_overdue": 0,
    "followups_blocked_broken_sendbox": 0,
    "suppressed_total": 0
  },
  "counts_reconciled_at": "",

  "notification_status": "not_sent",
  "last_notification_event": "",
  "last_notification_report_path": "",
  "last_notification_report_url": "",

  "last_update_agent": "",
  "last_update_note": ""
}
```

Allowed per-output status: `pending | complete | skipped | failed | blocked | not_due`.
Allowed `client_facing_scrub_status`: `not_due | pending | passed | failed | blocked`.
Allowed `weekly_pdf_status`: `not_due | pending | generated | blocked`.
Allowed `notification_status`: `not_sent | sent | skipped | failed`.

Rules:

- **Operator outputs are never scrubbed and never client-facing.** The weekly client report + PDF are the only client-facing artifacts and MUST pass the Client-Blind Scrub Gate before handoff. If they do not, keep `client_facing_scrub_status: failed | blocked`, record `client_facing_scrub_blocker`, and do not present the file as client-ready.
- On a **non-Monday** run, `weekly_client_report_status`, `client_facing_scrub_status`, and `weekly_pdf_status` stay `not_due`.
- After a run reaches a terminal state, `counts` must be **reconciled** against the sent log, `activities.jsonl`, `approval_log.md`, `sync_log.jsonl`, deals/tasks, and `notification_log.md`, and `counts_reconciled_at` stamped. Do not leave one artifact reporting "in progress" while another reports completion.
- `latest/` pointers must be updated to the current dated set after each artifact reaches `complete`. Operator-only `latest/` files stay clearly labeled internal.
- The Stage 9 completion gates read this ledger: a run cannot be claimed complete with a required output still `pending`/`failed`, with counts unreconciled, or with a client-facing artifact that failed the scrub gate.

---

## 11. Root Files (`daily-content-pipeline/…`)

### 11.1 `clients_index.md`

The root index of all client pipelines.

```md
# Clients Index

| Client | Client Slug | Pipeline Folder | Client Profile File | Status | Added Date | Schedule | Sendboxes | Sendbox Health | Active Campaigns | Notes |
|---|---|---|---|---|---|---|---|---|---|---|
| Angela Do Realty | angela-do | clients/angela-do/realestate_austin | client_profile_angela-do_realestate_austin.md | active | 2026-07-15 | daily | sb-a, sb-b | all healthy | austin-sellers-q3 | Austin listing agents |
```

Allowed status:

- `needs_setup`
- `ready_for_automation_first_run`  # Setup Flow finished; first automation run not yet done
- `active`
- `paused`
- `archived`

The client-level `status` **never** encodes sendbox health — there is no `needs_reauth` client status, because a broken sendbox must not halt a whole client. Sendbox health lives ONLY in `sendboxes/sendboxes.json` `status` (`healthy | needs_reauth | paused`, §6.1 / DESIGN §8) and the automation manifest's per-client **Sendbox Health** column (§11.7). Per DESIGN §8, a broken box is dropped from step-1 rotation and its assigned pending follow-ups **wait** (never reassigned) with an `[ACTION REQUIRED]` re-auth — the client stays `active` in daily runs; only that box's follow-ups pause. The **Sendbox Health** column above is a non-excluding per-client visibility signal only (e.g. `all healthy` / `sb-b needs_reauth`); it does NOT remove the client from active daily runs.

Daily runs process every client with `active` status. A `ready_for_automation_first_run` client flips to `active` on/after its first successful automation run.

### 11.2 `schedule.md`

Records how daily runs happen in the current AI environment. May use: native AI automations, reminders, cron, launchd, Task Scheduler, n8n, Make, GitHub Actions, or a local desktop routine. If true automation is unavailable, create manual run instructions. Each client has **one** automation task named `{Client} - OutreachCRM Daily Run`; there is also one agency-wide `OutreachCRM - GitHub Update Watch` task (Stage 11). A client's task prompt pins `target_client_slug` and cannot touch another client.

### 11.3 `storage_config.json`

Selects the storage backend the adapter uses (§2). Minimal:

```json
{"schema_version": 1, "backend": "json"}
```

`backend` ∈ `json | postgres`. When `postgres`, add the connection reference (never inline credentials — point at a secret in `secrets/` or the user's secret manager). Changing this file is a migration event (`crm_store.py migrate`, §2.6), not a hand edit.

### 11.4 `provider_defaults.json`

Provider-neutral catalog for the **operator notification** provider only. WideCast is used **exclusively** as the notification provider (Telegram via `sendTelegramMessage` with email fallback, plus optional `uploadAsset` for the report link). It is not used for production, video, or publishing. This file must contain **no** API keys, tokens, cookies, passwords, or client secrets.

```json
{
  "schema_version": 1,
  "default_notification_provider": "widecast",
  "providers": {
    "widecast": {
      "type": "openapi",
      "role": "notification_only",
      "provider_home_url": "https://widecast.ai/",
      "discovery_url": "https://widecast.ai/openapi.yaml",
      "preferred_server_url": "https://widecast.ai/app/dashboard",
      "disabled_server_urls": ["https://api.widecast.ai"],
      "auth_type": "bearer_api_key",
      "api_key_prefix": "wc_live_",
      "secret_storage": "per_client_local_config",
      "notes": "OutreachCRM uses WideCast ONLY as the operator-notification provider (Telegram via sendTelegramMessage with email fallback, plus optional uploadAsset for the report link). Use https://widecast.ai/app/dashboard as the server; do not call https://api.widecast.ai unless a future release enables it. Do not put API keys here. Notification is optional: with no provider configured, the daily run surfaces report links in chat and logs the blocker."
    }
  }
}
```

Rules:

- Agents use `discovery_url` to fetch the OpenAPI spec (via `tools/provider_openapi.py`) rather than hard-coding endpoint paths, and read the `servers` list + operation schemas before calling.
- For WideCast, select `https://widecast.ai/app/dashboard`; skip `https://api.widecast.ai` (disabled/planned host) unless a future playbook enables it.
- Never commit real API keys or account-specific state into this file.

### 11.5 `secrets/` (gitignored)

Agency-wide secrets only: the OAuth client (for `oauth`-mode sendboxes) and the tracker HMAC key (`TRACKER_API_KEY`). Gitignored; never staged (the deploy secret-scan blocks `refresh_token`, `client_secret`, `TRACKER_API_KEY`). Per-sendbox tokens live beside their box (§6.2), not here.

### 11.6 `suppression/global_suppression.jsonl`

Agency-tier suppression, checked before **every** send across all clients. Same record shape as §4.10 with `tier: global`, `scope: all_clients`.

### 11.7 `automation/` package

The automation package that scheduled runs must obey. Native schedulers may snapshot their own prompt at creation time; if the human changes anything afterward, the agent updates this package during **Automation Resync** and verifies with a dry-read.

#### `automation/automation_manifest.md`

```md
# Automation Manifest

- manifest_version: 1
- created_at:
- last_resynced_at:
- resync_status: current | automation_prompt_update_pending | partial | blocked
- scheduler_type: native_ai_automation | native_ai_scheduled_task | cron | launchd | task_scheduler | n8n | make | zapier | github_actions | server_job | manual
- scheduler_name:
- scheduler_location_or_url:
- timezone:
- schedule_file: daily-content-pipeline/schedule.md
- scheduled_prompt_file: daily-content-pipeline/automation/scheduled_run_prompt.md
- scheduled_entrypoint: playbooks/SCHEDULED_RUN_ENTRYPOINT.md
- root_playbook: OUTREACHCRM_PLAYBOOK.md
- clients_index: daily-content-pipeline/clients_index.md
- storage_config: daily-content-pipeline/storage_config.json
- provider_defaults: daily-content-pipeline/provider_defaults.json
- global_suppression: daily-content-pipeline/suppression/global_suppression.jsonl
- notification_channel:
- notification_provider_status:
- provider_capability_cache_status:
- tracker_worker_status:
- report_lane_contract: operator_lane_plus_weekly_client_lane | unknown   # operator_lane_plus_weekly_client_lane is the current default
- report_notification_policy: single_run_completion_notification | weekly_client_report_notification | unknown
- latest_user_change_summary:
- actual_native_task_prompt_updated: true | false | not_applicable | unknown
- automation_prompt_update_pending_reason:
- automation_freshness_status: current | resync_in_progress | action_needed | not_applicable
- automation_freshness_summary: whether latest changes are synced into automation/scheduled task prompt/contract/playbook/state, not only config, and whether tomorrow's run will load the newest state

## Active Clients

| Client | Client Slug | Profile Path | Status | Sendbox Health | Active Campaigns | Notification Status | Notes |
|---|---|---|---|---|---|---|---|

## Current Run Contract

- Scheduled runs must load the latest local playbooks at run time.
- Scheduled runs must read this manifest, schedule.md, storage_config.json, provider_defaults.json, clients_index.md, global_suppression.jsonl, each active Client Intelligence Profile, per-client sendboxes.json, each active campaign_config.json, and per-client provider config.
- Each client's automation task pins its target_client_slug and must not touch another client.
- Scheduled runs must not rely only on the prompt snapshot from the day the automation was created.

## Last Dry-Read Verification

- verified_at:
- verified_by_agent:
- result: pass | fail | partial
- next_scheduled_run_will_see:
- blockers:
```

#### `automation/scheduled_run_prompt.md`

The exact prompt the native automation/scheduler should use; it mirrors `playbooks/SCHEDULED_RUN_ENTRYPOINT.md` while pointing at the local workspace. The agent updates it during Automation Resync whenever a future run needs new behavior or newly approved state. If the runtime stores a separate native prompt the agent cannot edit, write the replacement here and set `resync_status: automation_prompt_update_pending` in the manifest.

#### `automation/resync_log.md`

Tracks every post-schedule change and whether the automation package was fully synced.

```md
# Automation Resync Log

| Date | Agent | Human Change | Files Updated | Native Task Prompt Updated | Dry-Read Result | Remaining Blocker | Next Scheduled Run Expected Behavior |
|---|---|---|---|---|---|---|---|
| 2026-07-15 | Claude | Added campaign austin-sellers-q3, raised daily_quota to 40 | profile, schedule.md, campaign_config.json, automation_manifest.md, scheduled_run_prompt.md | yes | pass | none | Run new campaign at 40/day within sendbox caps |
```

#### `automation/github_issues.md`

Tracks GitHub issues / intake submissions / drafts opened when the latest GitHub playbooks/code still do not resolve a blocker. The human does not need a GitHub account; direct creation uses an authorized agent/runtime identity when available, else a configured intake channel or local draft.

```md
# GitHub Issue Tracker

| Date | Agent | Client Slug | Blocker Fingerprint | Local Commit | GitHub Main Commit Checked | Issue URL / Intake Channel / Draft Path | Status | Next Check | Latest Response / Next Action |
|---|---|---|---|---|---|---|---|---|---|
| 2026-07-15 | Codex | angela-do | sendbox_oauth_invalid_grant_after_fresh_check | abc123 | def456 | https://github.com/soloagency/outreach/issues/123 | opened_by_agent | 2026-07-16 | Waiting for maintainer response |
```

New issues are filed against `soloagency/solo-agency` (prefix the title `outreach:` for triage); the dated row above predates the monorepo merge and is kept only as a format example. Issue drafts live under `daily-content-pipeline/automation/issues/YYYY-MM-DD_{blocker_slug}.md`. Recommended status values: `opened_by_agent`, `sent_to_intake`, `queued_for_intake`, `draft_waiting_for_support_channel`, `draft_waiting_for_human`, `answered`, `fix_applied`, `resolved`, `closed`. Every issue/draft must be redacted — no API keys, tokens, OAuth refresh tokens, `token.json` contents, recipient PII, or raw provider responses; include only safe reproduction steps, expected/actual behavior, local commit, GitHub main commit checked, runtime, blocker names, and redacted logs.

#### `automation/update_state.json`

Tracks the installed OutreachCRM version, latest GitHub check, auto-apply preference, tracker/schema action requirements, and resync state. Created when the first update check runs, when the `OutreachCRM - GitHub Update Watch` task is created, or when Stage 11 applies an update.

```json
{
  "schema_version": 1,
  "installed_commit": "",
  "latest_checked_commit": "",
  "last_checked_at": "",
  "last_applied_commit": "",
  "last_applied_at": "",
  "auto_apply_approved": false,
  "update_watch_task_name": "OutreachCRM - GitHub Update Watch",
  "last_change_classification": "",
  "tracker_worker_deploy_required": false,
  "storage_schema_migration_required": false,
  "sendbox_reauth_required": [],
  "automation_prompt_update_pending": false,
  "update_watch_task_prompt_pending": false,
  "clients_resynced": [],
  "clients_pending_resync": [],
  "automations_resynced": [],
  "human_actions_required": []
}
```

- `tracker_worker_deploy_required` → an update changed `tracker/worker.js` and the Worker must be re-deployed (`wrangler deploy`) before tracking/unsub events are trustworthy.
- `storage_schema_migration_required` → a bumped `schema_version` needs `crm_store.py migrate`/upgrade before runs continue.
- `sendbox_reauth_required` → list of sendbox slugs whose token/auth compatibility changed and must be re-authenticated before they send again; Stage 11 keeps them listed until the human confirms re-auth and a clean sync.
- `clients_pending_resync` → clients the scheduled `OutreachCRM - GitHub Update Watch` task recorded as needing resync **without** writing under `clients/`; a maintenance session or each client's own daily run self-heals them.
- Set `update_watch_task_prompt_pending: true` when the `OutreachCRM - GitHub Update Watch` task prompt could not be created/updated natively and `daily-content-pipeline/automation/update_watch_prompt.md` holds the pending prompt.
- The canonical `update_state.json` schema lives in Stage 11 (`11_UPDATE_AND_VERSION_WATCH.md`, "Minimum `update_state.json`"); keep this block byte-identical to it.
- Do not store secrets, tokens, client-confidential report content, or raw provider responses here.

#### `automation/update_log.md`

```md
# OutreachCRM Update Log

| Date | Agent | Local Commit Before | GitHub Main Commit | Change Classification | Applied | Backup Path | Clients Resynced | Automations Resynced | Tracker Worker Deploy Required | Storage Schema Migration Required | Blocker / Next Action |
|---|---|---|---|---|---|---|---|---|---|---|---|
```

Change classification values (the canonical enum lives in Stage 11 — `11_UPDATE_AND_VERSION_WATCH.md`, "Change classification values"; keep identical):

- `no_change`
- `playbook_only`
- `provider_tooling`
- `crm_core_tooling`
- `storage_schema_migration`
- `tracker_worker`
- `send_or_sendbox_compat`
- `renderer_or_report_format`
- `setup_or_schedule_contract`
- `breaking_or_major_behavior`
- `unknown`

#### `automation/update_notice.md`

Internal/local notice for the latest update-watch outcome.

```md
# OutreachCRM Update Notice

- checked_at:
- installed_commit:
- latest_github_commit:
- change_classification:
- auto_apply_approved:
- update_applied:
- tracker_worker_deploy_required:
- storage_schema_migration_required:
- automation_prompt_update_pending:
- next_human_action:
```

Do **not** send update-watch notices through Telegram/WideCast/email or any client channel, and do not put update-watch rows in `notifications/notification_log.md` — version checks and applied updates are internal maintenance, not report delivery.

#### `automation/update_watch_prompt.md`

The exact prompt for the native maintenance task `OutreachCRM - GitHub Update Watch`, used when the runtime cannot create/edit that task directly. It comes from `playbooks/SCHEDULED_RUN_ENTRYPOINT.md`, loads Stage 11, and must **not** run campaigns, sends, enrichment, or client reports — it explicitly says not to. The update-watch task is barred from client-facing channels.

#### `automation/backups/`

Timestamped update backups: `daily-content-pipeline/automation/backups/update_YYYY-MM-DD_HHMMSS/`. Used for runtime files/folders that Stage 11 replaces (backup-and-safe-apply: merge config, never overwrite secrets/history). Not a long-term archive for reports, secrets, tokens, or provider keys.

### 11.8 `notifications/notification_log.md`

Tracks operator notifications sent through the WideCast Telegram / email-fallback channel (optional `uploadAsset` for the report link). This log is for report/result delivery, not update-watch (§11.7).

```md
# Notification Log

| Date | Agent | Event | Channel | Status | Report Path | Report Link Sent | Provider | Provider Discovery Checked | Upload Operation | Notification Operation | Upload Attempted | Uploaded Report URL | Notification Attempted | Blocker | Action Needed |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-07-15 | Claude Schedule | daily_run_completed | WideCast Telegram/email fallback | sent | outputs/2026-07/2026-07-15/angela-do-daily-ops.html | yes | widecast | yes | uploadAsset | sendTelegramMessage | yes | https://... | yes | none | Review approvals in chat |
| 2026-07-20 | Claude Schedule | weekly_client_report_ready | WideCast Telegram/email fallback | sent | outputs/2026-07/2026-07-20/angela-do-weekly-client-report.html | yes | widecast | yes | uploadAsset | sendTelegramMessage | yes | https://... | yes | none | Client report scrubbed and ready |
```

Event enum: `daily_run_completed | weekly_client_report_ready`. Notification is **optional** — with no provider configured, the run surfaces the report links in chat and logs the blocker; it does not fail the run. Distinguish blockers precisely:

- `provider_config_missing` — no per-client provider config.
- `provider_auth_missing` — config exists but no API key/supported auth value.
- `provider_auth_failed` — provider rejected the credential.
- `provider_discovery_failed` — OpenAPI discovery URL could not be fetched/parsed.
- `provider_required_operation_missing` — spec lacks the needed operation (`sendTelegramMessage`/`uploadAsset`).
- `provider_account_mismatch` — verified account does not match the saved client identity.
- `global_mcp_not_client_scoped` — an MCP/native tool is visible but not proven authenticated as this client's provider account.
- `provider_upload_failed` — `uploadAsset` exists but the call failed.
- `provider_notification_failed` — `sendTelegramMessage` exists but send failed.
- `provider_notification_not_configured` — account valid but Telegram/email destination not configured and no fallback sent.

Preserve `provider_identity_source: per_client_openapi` for client-scoped delivery; if a global MCP/native provider account was visible but not proven to match, record `mcp_compatibility_status: not_client_scoped` and blocker `global_mcp_not_client_scoped`. Do not log `unavailable` generically when the real issue is missing config, failed auth, a missing operation, expired credentials, or an account mismatch.

---

## 12. Per-Client Provider Integration (`integrations/providers/…`)

Each client that uses the notification provider keeps provider state under `clients/{client_slug}/{business_slug}_{location_slug}/integrations/providers/`. WideCast here is **notification-only**.

### 12.1 `provider_config.local.json`

Client-local, sensitive. Never include it in a public repo, zip, screenshot, report, or support bundle unless secrets are removed.

```json
{
  "schema_version": 1,
  "client_slug": "angela-do",
  "active_provider": "widecast",
  "providers": {
    "widecast": {
      "type": "openapi",
      "role": "notification_only",
      "discovery_url": "https://widecast.ai/openapi.yaml",
      "provider_home_url": "https://widecast.ai/",
      "preferred_server_url": "https://widecast.ai/app/dashboard",
      "disabled_server_urls": ["https://api.widecast.ai"],
      "auth_type": "bearer_api_key",
      "api_key_env": "OUTREACHCRM_WIDECAST_API_KEY_ANGELA_DO",
      "api_key_local": "",
      "provider_identity_source": "per_client_openapi",
      "mcp_compatibility_status": "not_used",
      "pdna_setup_blocker": "",
      "account_verified_at": "",
      "account_identity": {
        "company_id": "",
        "email_masked": "",
        "name": "",
        "connected_platforms": []
      },
      "pdna": {
        "notification": "not_configured"
      },
      "notification": {
        "enabled": false,
        "preferred_operation_id": "sendTelegramMessage",
        "delivery": "telegram_or_email_fallback"
      },
      "report_upload": {
        "enabled": false,
        "preferred_operation_id": "uploadAsset",
        "content_type": "text/html",
        "ttl_hours_note": "WideCast uploadAsset URLs may be short-lived. Use for the operator report-link notification, not as a permanent archive; the local report file is the archive."
      }
    }
  }
}
```

Credential rules:

- Prefer `api_key_env` or the user's secret manager. If a local key is saved in `api_key_local`, keep it only in this per-client file and redact it in all logs and reports.
- **Do not create or use a field named `api_key`.** The official helper reads `api_key_env` and `api_key_local`; a stray `api_key` field is ignored and causes `provider_auth_missing`.
- Never store passwords, OTPs, cookies, session tokens, or raw OAuth refresh tokens here.
- Before any provider action, verify the active account with the provider account operation (`getAccount`). If the verified identity changes unexpectedly, stop and log `provider_account_mismatch`.
- Check this per-client config + OpenAPI cache/capability files as **Client tools** before checking any global MCP/native tool. `global_mcp_compat` is allowed only after the MCP/native identity is compared to the saved client identity and matches exactly.
- `mcp_compatibility_status` ∈ `not_used | identity_matched | identity_mismatch | not_client_scoped`; on `identity_mismatch`/`not_client_scoped`, do not use MCP/native account data for this client.
- `pdna_setup_blocker` uses provider-neutral names: `provider_config_missing`, `provider_auth_missing`, `provider_auth_failed`, `provider_discovery_failed`, `provider_account_mismatch`, `global_mcp_not_client_scoped`.
- Keep `preferred_server_url` = `https://widecast.ai/app/dashboard` and `disabled_server_urls` including `https://api.widecast.ai` until a future playbook enables that host.

### 12.2 `provider_capabilities.json`

Snapshot of discovered OpenAPI operations — the main Client-tools inventory, safe without secrets. Refresh with `python3 tools/provider_openapi.py --config <client provider_config.local.json> --defaults daily-content-pipeline/provider_defaults.json discover --out-dir <client integrations/providers folder>` (`--config`/`--defaults` are global flags that must PRECEDE the `discover` subcommand). Because the role is notification-only, only the notification/upload/account operations matter.

```json
{
  "schema_version": 1,
  "provider": "widecast",
  "role": "notification_only",
  "discovered_at": "",
  "discovery_url": "https://widecast.ai/openapi.yaml",
  "server_url": "https://widecast.ai/app/dashboard",
  "server_urls_discovered": [],
  "server_urls_skipped_disabled": ["https://api.widecast.ai"],
  "auth_scheme": "bearerAuth",
  "operation_ids": {
    "getAccount": {"method": "GET", "path": "/..."},
    "sendTelegramMessage": {"method": "POST", "path": "/..."},
    "uploadAsset": {"method": "POST", "path": "/..."}
  },
  "operation_aliases": {
    "account": "getAccount",
    "send_notification": "sendTelegramMessage",
    "upload_asset": "uploadAsset",
    "upload_html_report": "uploadAsset"
  },
  "capability_status": {
    "notification": "available | partial | unavailable"
  },
  "missing_capability_aliases": {},
  "identity": {
    "provider_identity_source": "per_client_openapi | global_mcp_compat | unknown",
    "account_verified": true,
    "mcp_compatibility_status": "not_used | identity_matched | identity_mismatch | not_client_scoped"
  },
  "blockers": []
}
```

Whenever the human or automation asks to check tools, check this Client-tools file first; inspect global MCP/native tools only after this file and the verified provider identity are current.

### 12.3 `provider_openapi_cache.yaml`

Raw OpenAPI spec cache for repeatable automation. Refresh when: the file is missing; the cache is older than the refresh policy; `provider_defaults.json` changes; a provider action fails on a stale operation/schema; or the human changes provider configuration.

### 12.4 `provider_calls.jsonl`

Append-only provider audit log. Each line: timestamp, agent, client_slug, provider, `operationId`, redacted request summary, response status, `request_id` if present, blocker if any. Never log full API keys or raw secrets.

### 12.5 `provider_health.md`

```md
# Provider Health

| Date | Agent | Provider | Role | Identity Source | MCP Compatibility | Account Verified | Notification | Report Upload | Credits | Blocker | Next Action |
|---|---|---|---|---|---|---|---|---|---|---|---|
```

---

## 13. History Model — where "history" lives now

OutreachCRM's history is not one content log; it is a set of append-only, `seq`-stamped ledgers plus per-record `stage_history`, all reconciled by `report_state.json`:

| History concern | Where it lives |
|---|---|
| Every event (the backbone / contact timeline) | `crm/activities/YYYY-MM/activities.jsonl` |
| What was sent | `campaigns/{slug}/sent/YYYY-MM/sent_log.jsonl` |
| What came back | `inbox_sync/YYYY-MM/sync_log.jsonl`, `campaigns/{slug}/history/YYYY-MM/reply_log.md` |
| Per-send/step narrative | `campaigns/{slug}/history/YYYY-MM/campaign_log.md` |
| Approval decisions | `approvals/approval_log.md` |
| Import provenance | `lists/{list_slug}/leads.jsonl`, `import_log.md` |
| Deal movement | `deals/{deal_id}.json` → `stage_history[]` |
| Performance + learning | `analytics/metrics_log.md`, `analytics/learning_log.md` |
| Provider calls | `integrations/providers/provider_calls.jsonl` |
| Run reconciliation | `outputs/YYYY-MM/YYYY-MM-DD/{client}-report_state.json` |

Rules that make history trustworthy:

- Append-only logs are never rewritten in place; corrections are new rows referencing the prior `seq`.
- Every log row carries `ts` + monotonic `seq`; every record carries `created_at`/`updated_at`.
- A timeline is always read through `resolve(lead_id)` so merged contacts show one continuous history.
- Open/click history is recorded but is **never** conversion evidence on its own — only a reply is.
- No side-effect write (send, suppress, stage change, notify, claim completion) happens without this stage loaded IN FULL and the relevant Stage's LOAD LEDGER `PASS` earlier in the transcript.

---

## 14. Completion Gates (cross-reference Stage 9)

A run that touches storage cannot be claimed complete until the Stage 9 audit confirms, against this stage's structures:

- Every write went through `crm_store.py` (no hand-written CRM JSON).
- Every draft that sent had an explicit chat `approve` in `approval_log.md`, and every send passed the ordered pre-send gate chain (suppression both tiers → quota reservation → warmup/domain caps → send-window → guessed cap → sequence-freeze → subject lint).
- Every deal `stage_history` entry has an `evidence_activity_id`; no stage change was improvised by the LLM.
- Every draft cited only dossier facts that carry an `evidence_url`.
- `report_state.json` counts are reconciled; required outputs are `complete`; the weekly client report (Mondays only) passed the Client-Blind Scrub Gate before being presented as client-ready.
- Suppression was checked at import (all identities) and at every send path; merges unioned identities/channels/suppression into the survivor; pending-merge contacts stayed out of queues.
- The client's Metadata `status` and `clients_index.md` row reflect reality; any post-setup change was carried through Automation Resync with a passing dry-read.

Surface any unmet gate with the `[ACTION REQUIRED]` contract: one purpose, one exact next step, one command or path. When nothing is needed, say `No action required right now.`
