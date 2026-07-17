# Stage 13 — CRM Core (objects, lifecycle, stage rules, dedupe/merge)

Stage: `13`

## Load Rule

Load before you reason about **CRM objects** — a contact/account/deal/activity/task, a lifecycle or
pipeline-stage transition, or a dedupe/merge decision. This is the schema-and-lifecycle reference for
everything `crm_store.py` writes. Its authoritative schemas live in `docs/DESIGN.md` §7 (and Stage 7
for the on-disk write discipline); this stage is the operating gloss, not a second source of truth.
Every load needs a LOAD LEDGER per `playbooks/LOAD_LEDGER_PROTOCOL.md`.

## Hard Gates For This Stage

- **`tools/crm_store.py` is the ONLY sanctioned writer of `crm/` collections.** Never hand-edit a
  contact/deal/activity/task file. Reads may be direct; every write goes through the tool (atomic
  temp+rename, fcntl lock, monotonic `seq`).
- **The rules engine is deterministic and idempotent.** Guard keys `(rule_id, trigger_activity_id)`
  make `apply-rules` safely re-runnable — the same reply never creates two deals. Do not simulate a
  rule by hand-writing its effects.
- **`resolve(lead_id)` before you touch a contact.** A merged-away contact is a tombstone; every
  gate, draft, and send must run against the surviving id. Merge is a union of identities (deduped)
  plus a tombstone that `resolve()` follows.
- **A contact does NOT require an email.** Name + phone/social is a valid contact. `email_first`
  campaigns require an email; other channels do not (DESIGN §7.1).

## The objects (DESIGN §7)

| Object | Key facts |
|---|---|
| **Contact** `crm/contacts/{lead_id}.json` | email not required; per-channel status (email/sms/social); `lifecycle_stage: lead\|engaged\|opportunity\|customer\|evangelist\|lost\|do_not_contact`; `next_action.task_id`. |
| **Account** | company/office (brokerage). `{id,name,domain,type,location,contact_ids[]}`. |
| **Deal** `crm/deals/{deal_id}.json` | `pipeline`, `stage`, `value`, `probability`, `stage_history[]`, `status: open\|won\|lost`. |
| **Activity** `crm/activities/YYYY-MM/activities.jsonl` | append-only event backbone; a contact timeline = filter by `contact_id` through `resolve()`. |
| **Task** `crm/tasks/tasks.jsonl` | `{title, due_at, status, created_by, guard_key}` (Stage 14). |
| **Pipeline** `crm/pipelines.json` | stages carry `probability` + `sla_days`; deterministic rules r1–r6. |

## Lifecycle & stage rules (deterministic — DESIGN §7.6)

Default `default_sales` stages: `new_reply` (p .10, SLA 1d) → `engaged` (.25, 7d) →
`meeting_booked` (.50, 7d) → `proposal_sent` (.70, 10d) → `won` / `lost`. Rules fire on activities:

- **r1** `reply_positive` → create deal `new_reply` + "reply within 4h" task + freeze sequence.
- **r2** `reply_question` → deal `engaged` + freeze + draft reply for approval.
- **r3** `reply_negative | remove_intent` → suppress + freeze + close open tasks.
- **r4** `stage_age_exceeds_sla` → nudge task + flag in report.
- **r5** `deal_won` → lifecycle `customer` + enroll `customers` segment + onboarding task.
- **r6** `hard_bounce | unsubscribe` → suppress + close open tasks.

Drive them through `apply-rules` (never by hand):

```sh
python3 tools/crm_store.py --client-dir <CLIENT_DIR> apply-rules --event reply_positive \
  --contact <lead_id> --activity <activity_id>
python3 tools/crm_store.py --client-dir <CLIENT_DIR> deal move --id <deal_id> --stage proposal_sent
python3 tools/crm_store.py --client-dir <CLIENT_DIR> contact merge --json '{"keep":"<id>","drop":"<id>"}'
```

## Dedupe / merge

`add_contact` dedupes on normalized identity (email / E.164 phone / social) via the cached
reverse-index under a lock, so a re-import never splits a lead. `merge` writes a tombstone on the
dropped id and unions identities into the survivor; `resolve()` follows the chain. `validate
--rebuild-index` repairs the identity cache after a bulk change or migration.

## Completion Gates

- No `crm/` file was hand-written; every mutation went through `crm_store.py`.
- Every rule effect came from `apply-rules` (idempotent guard keys), not a simulated write.
- Every contact reference resolved through `resolve()` — no action taken against a tombstone.

## Phase status

The CRM core (`crm_store.py`: objects, `apply-rules` r1–r6, dedupe/merge/`resolve`, pipelines) is
**built** (Phase 1 + 2). The polished kanban/timeline UI and segment analytics are Phase 3.

When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
