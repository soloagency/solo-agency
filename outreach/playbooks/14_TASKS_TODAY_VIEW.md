# Stage 14 — Tasks & Today View (task engine, SLA, operator surface)

Stage: `14`

## Load Rule

Load before you build the **operator's Today View**, reason about the **task engine / SLA sweep**, or
regenerate the **kanban**. This is the daily operating surface for the human running the agency —
what needs doing *today*, in priority order. It reads CRM state (Stage 13); it does not invent it.
Every load needs a LOAD LEDGER per `playbooks/LOAD_LEDGER_PROTOCOL.md`.

## Hard Gates For This Stage

- **The Today View is operator-only and read-derived.** It is rendered from CRM state via
  `crm_store.py today-view` — it never becomes a write path. Editing the HTML changes nothing;
  all decisions (approvals, stage moves) go through the tool/chat. This is the internal surface, NOT
  the scrubbed weekly client report (Stage 15).
- **Tasks are created by rules or by a human/agent, never faked.** A task carries a `guard_key` so a
  rule cannot create the same task twice. Close a task with `task done`, do not delete it.
- **SLA is measured, not asserted.** A deal past its stage `sla_days` is surfaced because the code
  compared `stage_history` age to the stage SLA — not because it "feels" stale. Rule r4 raises the
  nudge task and flags it in the report.
- **Hot replies come first.** A `reply_positive` carries a "reply within 4h" task (r1); those, and
  deals past SLA, sort to the top of the Today View so a same-day reply actually happens.

## The task engine (DESIGN §7.5)

`crm/tasks/tasks.jsonl` — `{id, contact_id?, deal_id?, title, due_at, status: open|done|cancelled,
created_by: rule|human|agent, guard_key}`. Tasks originate from the rules engine (r1 reply-within-4h,
r4 SLA nudge, r5 onboarding) or are added directly:

```sh
python3 tools/crm_store.py --client-dir <CLIENT_DIR> task add  --json '{"title":"Call back","deal_id":"d_...","due_at":"..."}'
python3 tools/crm_store.py --client-dir <CLIENT_DIR> task done --id <task_id>
```

"Due" means `status == open` and `due_at <= now` — a task with a future `due_at` (e.g. the +4h reply
task) is pending, not yet due.

## The Today View

```sh
python3 tools/crm_store.py --client-dir <CLIENT_DIR> today-view
```

`today_view_data` gathers, and `render_today_view` renders (via the report renderer), the operator's
single screen:

- **Tasks due** — open tasks with `due_at <= now`, hot replies first.
- **Deals past SLA** — open deals whose current stage age exceeds its `sla_days` (the r4 sweep input).
- **Hot replies** — `new_reply` deals awaiting a same-day human reply.
- **Drafts pending** — count awaiting approval (link to the Approval Report, Stage 8/10).

## The kanban

```sh
python3 tools/crm_store.py --client-dir <CLIENT_DIR> kanban
```

`render_kanban` lays deals out by pipeline stage with a **weighted forecast** (Σ `value ×
probability` across open deals). This is the internal pipeline view; the polished client-facing
forecast is Stage 15 (Phase 3).

## Completion Gates

- Every surfaced task/deal was read from CRM state through `crm_store.py` — nothing hand-authored.
- SLA breaches were computed from `stage_history` age vs the stage `sla_days`, not guessed.
- Hot replies and past-SLA deals sort to the top; the pending-drafts count links to the Approval Report.
- The Today View was treated as read-only; no approval or stage move happened by editing the HTML.

## Phase status

The task engine (`crm_store.py task`, rule-created tasks), Today View (`today-view`), and kanban
(`kanban`) are **built** (2D). The polished kanban/timeline UI, weighted-forecast client report, and
segment analytics are Phase 3 (Stage 12/15).

When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
