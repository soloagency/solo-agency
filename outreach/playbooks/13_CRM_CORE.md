# Stage 13 — CRM Core (objects, lifecycle, stage rules, dedupe/merge)

Stage: `13`

## Load Rule

Load before you reason about **CRM objects** — a contact/account/deal/activity/task, a lifecycle or
pipeline-stage transition, or a dedupe/merge decision. This is the schema-and-lifecycle reference for
everything `tool crm-store` writes. Its authoritative schemas live in `docs/DESIGN.md` §7 (and Stage 7
for the on-disk write discipline); this stage is the operating gloss, not a second source of truth.
Every load needs a LOAD LEDGER per `playbooks/LOAD_LEDGER_PROTOCOL.md`.

## Hard Gates For This Stage

- **`tool crm-store` is the ONLY sanctioned writer of `crm/` collections.** Never hand-edit a
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
<bridge> tool crm-store --client-dir <CLIENT_DIR> apply-rules --event reply_positive \
  --contact <lead_id> --activity <activity_id>
<bridge> tool crm-store --client-dir <CLIENT_DIR> deal move --id <deal_id> --stage proposal_sent
<bridge> tool crm-store --client-dir <CLIENT_DIR> contact merge --json '{"keep":"<id>","drop":"<id>"}'
```

## Dedupe / merge

`add_contact` dedupes on normalized identity (email / E.164 phone / social) via the cached
reverse-index under a lock, so a re-import never splits a lead. `merge` writes a tombstone on the
dropped id and unions identities into the survivor; `resolve()` follows the chain. `validate
--rebuild-index` repairs the identity cache after a bulk change or migration.

## Plan cap and locked contacts

CRM contacts are a sold limit (`AGENTS.md`, "Plans"): Free 30, Starter 500, Pro 2000, Business 10000,
Enterprise unlimited. Every detected lead is still written to the CRM regardless of plan — the cap
never blocks capture, only what the agent can DO with a contact once the plan's count is exceeded.

- **Old stays open, new gets locked.** Contacts are ordered by `created_at`; the oldest contacts up
  to `max_contacts` stay unlocked, and contacts past that count are locked. A contact that has moved
  past the `lead` lifecycle stage (`engaged`, `opportunity`, `customer`, `evangelist`) is never locked,
  regardless of its position in that order.
- **What "locked" means.** No detail view, no email/campaign for that contact, no DM, and it is
  redacted in both `contact list` output and the CLI. `fb.message.send` to a locked contact is refused
  by the bridge with `contact_locked`; the agent must not draft a DM to a locked contact either, same as
  it must not draft an email. A locked contact is still counted, still receives new activities/leads on
  re-detection, and unlocks automatically the moment the plan is upgraded or older contacts age out.
- **Write actions are a separate, unrelated gate.** `write_actions` (`fb.group.post`, `fb.post.comment`,
  `fb.post.react`) is gated by plan (Starter and up), not by the contact lock — it has nothing to do
  with whether any particular contact is locked or unlocked, and applies the same way regardless of the
  CRM contact cap.
- **What the agent may show about a locked contact.** Name, lifecycle stage, first-seen date, and the
  source host (e.g. the group/page domain it was detected on) — never an identity: no email, no phone,
  no social handle, no message content, no dossier field.
- **`contact get` on a locked id** returns a redacted record with `"locked": true` and the fields
  above only; treat the absence of identity fields as the lock, not as missing data to re-fetch.
- **Campaigns/queue/send skip locked contacts.** A locked contact is excluded before enrollment with
  blocker `contact_locked`; if already queued, it is skipped with reason `locked`. Do not draft, queue,
  or send to a locked contact under any circumstance — that is the entire point of the cap.
- **Read `contact lock-status` before telling a human how many leads are locked.**
  `<bridge> tool crm-store --client-dir <CLIENT_DIR> contact lock-status` returns JSON:
  `max_contacts`, `lockable`, `unlocked`, `locked`, `upgrade_url`. Use its numbers, never a hand count.
- **The chat line when leads are locked** (fill in `{locked}`, `{tier}`, `{upgrade_url}` from
  `contact lock-status` / `GET /status` → `entitlement`):
  - VI: "{locked} lead(s) đang bị khoá theo gói {tier} — nâng cấp tại {upgrade_url} để mở."
  - EN: "{locked} lead(s) are locked under the {tier} plan — upgrade at {upgrade_url} to unlock them."
- **CRM answers follow the Answer-and-Show Rule** (`SOLO_AGENCY_PLAYBOOK.md`, "Team Leader Reply
  Frame"): who-is-new/lead-count/locked-contact questions navigate the side dashboard to
  `/ui/{client}/crm` with the matching filter (`sort=-created`, `locked=1`, `stage=`, `q=`), and a
  question about one named person navigates to `/ui/{client}/contact/{id}`.

## Completion Gates

- No `crm/` file was hand-written; every mutation went through `tool crm-store`.
- Every rule effect came from `apply-rules` (idempotent guard keys), not a simulated write.
- Every contact reference resolved through `resolve()` — no action taken against a tombstone.

## Phase status

The CRM core (`tool crm-store`: objects, `apply-rules` r1–r6, dedupe/merge/`resolve`, pipelines) is
**built** (Phase 1 + 2). The polished kanban/timeline UI and segment analytics are Phase 3.

When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
