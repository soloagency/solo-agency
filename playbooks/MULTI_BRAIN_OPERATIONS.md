# Multi-Brain Operations — several agents driving one install at the same time

`AGENTS.md` already establishes that the filesystem is the system and the agent is only the
brain. "Brain swap" covers the SEQUENTIAL case: one brain stops, another takes over. This file
covers the CONCURRENT case: the operator has Codex open in one window and Claude in another and
gives each of them different work against the same install, in the same hour.

That is a supported operating mode, not a violation. The operator does not have to close one
agent to use the other, and no agent may invent a gate that forbids it. What IS forbidden is
narrow and listed below.

---

## The two tiers

Every unit of work on a Solo Agency / OutreachCRM install falls into exactly one tier:

- **Tier 1 — Scheduled work.** A recurring task fired by a runtime's own scheduler
  (`{Client} - Solo Agency Daily Run`, `{Client} - {Campaign} Daily Run`,
  `Solo Agency - GitHub Update Watch`). **Exactly one runtime owns each scheduled task.**
  This is unchanged and non-negotiable — two schedulers firing the same task against one
  campaign double-sends.
- **Tier 2 — Operator-directed work.** Anything a human asks for in a live chat right now:
  run this campaign, enrich this lead, draft this email, render this report, check this status.
  **Tier 2 is concurrent by default**, bounded only by the claim discipline below.

The operator's example is the shape to support: Claude enriches a lead in campaign A while
Codex writes an email for a different lead in campaign B. Both are Tier 2. Neither waits for
the other.

---

## What is actually enforced, and what is only agreed

Be honest about this distinction when reporting to the operator; do not present an agreement
as a guarantee.

**Enforced in the binary — safe even if a brain misbehaves:**

- Per-collection exclusive `flock` on every `crm/` read-modify-write (`tool crm-store`).
- Atomic per-sendbox/day quota reservation, committed under the lock BEFORE the SMTP call, so
  two concurrent sends cannot both slip past the cap.
- Collector job claiming by rename out of `jobs/pending/`, bound to `client_slug` +
  `extension_instance_id`.
- `scan_claim` in `collector/source_registry.json`, so concurrent runs wait instead of
  duplicating a shared-source scan.
- Automation time-slot collision projection (`tool schedule-slots suggest`).
- Atomic temp-file + `rename` writes, and append-only `seq`-stamped JSONL ledgers.

**Agreed between brains, not enforced — honour it or the operator pays:**

- The per-client `run_lock`.
- The work lease described below.
- Brain attribution on writes.

These three are playbook conventions. A brain that skips them will not corrupt a CRM record —
the storage layer prevents that — but it WILL cause duplicated LLM work, contradictory drafts,
two half-finished runs for one client/day, and a report state nobody can reconcile. Treat them
as binding. Moving them into the binary (`tool crm-store lock ...`) is the known next hardening
step and follows the repo's own rule: what must be identical across brains belongs in code, not
in prose.

---

## Tier 1 — scheduler ownership stays exclusive

- One runtime owns each scheduled task. Record the owning runtime in
  `daily-content-pipeline/automation/automation_manifest.md` (`scheduler_runtime`) and in the
  `## Runtime Handover` section of `daily-content-pipeline/schedule.md`.
- A second brain must NOT register the same task in its own scheduler. It may read the task
  contract, report on it, and run the same work MANUALLY as Tier 2 when the operator asks —
  taking the `run_lock` exactly as a scheduled run would.
- Ownership can differ per task. Codex may own `LeadUp - Solo Agency Daily Run` while Claude
  owns `Solo Agency - GitHub Update Watch`. Record each owner separately; do not assume one
  brain owns the whole schedule.
- Only the human can disable a task inside another runtime. When ownership moves, follow the
  Brain swap step-4 confirmation in `AGENTS.md`.

---

## Tier 2 — claim discipline

### Full client run → `run_lock`

Unchanged contract (`playbooks/04_DAILY_SCHEDULE.md`, `outreach/playbooks/AUTOMATION_SCHEDULING.md`),
with two added fields so a second brain can tell WHOSE lock it is:

- `held_by_brain` — the runtime driving it: `codex`, `claude-code`, `claude-cowork`, or the
  runtime's own name. Never the string `agent`.
- `held_by_session` — whatever session/task identifier this runtime can produce.

A brain that finds a fresh lock held by ANOTHER brain reports that to the operator by name
("Codex is mid-run on LeadUp, started 12 minutes ago") and stops — it does not take over, and
it does not silently wait. Stale-lock takeover rules are unchanged.

### Scoped interactive work → work lease

For Tier 2 work smaller than a full run, take a lease instead of the run lock, so two brains
working on disjoint scopes do not block each other:

`daily-content-pipeline/automation/leases/{scope_key}.json`

- `scope` — one of `client`, `campaign`, `contact`, `report`, `install`.
- `scope_key` — `{client_slug}`, `{client_slug}__{campaign_slug}`, `{client_slug}__{contact_id}`, etc.
- `intent` — one short human line: what this brain is doing.
- `held_by_brain`, `held_by_session`, `acquired_at`, `ttl_minutes` (default 30), `heartbeat_at`.

Rules:

- Create the file only if it does not already exist. If it exists and is fresh, the scope is
  taken — report who holds it and what they said they are doing, then offer the operator the
  scopes that ARE free.
- A lease past its TTL with no heartbeat is dead and may be taken over, with a note.
- Delete the lease when the work ends, including when it ends in a blocker.
- Leases nest downward, never upward: holding `client` covers every campaign and contact under
  it; holding one `campaign` does not entitle you to client-wide steps.
- Read-only work — reading state, rendering an existing report, answering a status question —
  takes no lease at all.

---

## Never concurrent

These stay single-holder across the whole install regardless of tier. A second brain asked to
do one of these while another holds it must say so and stop.

- **The update flow.** `update` / `upgrade` / `cập nhật` / `sync latest` rewrites playbooks,
  binaries and extension folders under everyone's feet. Take the `install` scope lease for the
  entire update, including the resync. A brain that finds it held reports and stops.
- **Setup Flow.** The session model already allows exactly one setup session; it also gets
  exactly one brain.
- **One client's Daily Run**, per the `run_lock`.
- **The approve → send sequence for one campaign.** The quota reservation prevents overspend,
  but two brains draining one approval queue produces contradictory sends.
- **Writing `daily-content-pipeline/collector/run_now_request.json`.** Unchanged rule: prefer
  `POST /jobs/run_now`, or write one unique per-client job file under `collector/jobs/pending/`.

---

## Attribution — every write says which brain

An install driven by several brains needs to be readable afterwards. Wherever a record already
carries an actor, write the runtime name rather than the generic `agent`:

- `run_lock.json` and lease files — `held_by_brain`.
- `automation/automation_manifest.md` — `scheduler_runtime`, `verified_by_agent`.
- `automation/resync_log.md`, `notifications/notification_log.md`, `outputs/**/report_state.json`
  (`last_update_agent`) — name the runtime that did it.
- `schedule.md` `## Runtime Handover` — unchanged, still the custody record.

Every human-facing progress block from a brain that is not the sole operator of the install
must name itself once ("Claude Code, operating LeadUp campaign A") so the operator reading two
windows can tell the reports apart.

---

## Install root entry files

**A live install root MUST carry its own `AGENTS.md` and `CLAUDE.md`.** Without them, a fresh
session opened at the install root has no idea it is standing on a live system, and the
operator has to re-explain the whole setup by hand every time — which is exactly the failure
this file exists to remove.

These live at the INSTALL root (the folder holding `daily-content-pipeline/`), not inside the
source checkout. They are thin pointers: absolute paths plus "go read the real contract". They
carry no rules of their own, so they never drift from the playbooks.

Create or verify them during setup step 5, during any Brain swap takeover, and during any
update flow. Both files are install-local operator state — never commit them to the product
repo.

### Install-root `AGENTS.md` template

```md
# Solo Agency — live install

This folder is a LIVE Solo Agency install, not a fresh setup. Do not re-setup, re-clone,
re-import or re-authenticate anything.

- Workspace root:     {ABSOLUTE_INSTALL_ROOT}
- Verified source:    {ABSOLUTE_INSTALL_ROOT}/solo-agency
- Runtime state root: {ABSOLUTE_INSTALL_ROOT}/daily-content-pipeline
- Local collector:    {ABSOLUTE_INSTALL_ROOT}/solo-agency-local-collector  (bridge on 127.0.0.1:{PORT})

Read `solo-agency/AGENTS.md` in full before acting — it is the canonical contract for every
brain. Then read `solo-agency/playbooks/MULTI_BRAIN_OPERATIONS.md`: another agent may be
operating this same install right now.

Current state lives in `daily-content-pipeline/` — `clients_index.md`, `schedule.md`, and
`automation/automation_manifest.md` tell you what is running and who owns each scheduled task.
Report what you see before you change anything.
```

### Install-root `CLAUDE.md` template

```md
# Claude entry point — live Solo Agency install

Read `AGENTS.md` in this same folder now, then `solo-agency/AGENTS.md` in full. Everything in
them applies to you.

This is a LIVE install. Another brain (Codex or another Claude session) may be operating it at
this moment — read `solo-agency/playbooks/MULTI_BRAIN_OPERATIONS.md` and take the lease or
`run_lock` your work requires before you touch client state.

Running the installed bridge and its tools (`<bridge> tool crm-store ...`, `tool gmail ...`) IS
normal operation on a live install, not a code change and not a setup step — no "ask before
editing code" rule applies to it. Editing files under `solo-agency/` (the source checkout) IS a
code change and follows the normal repo rules.

Claude-runtime notes: register automations as Claude scheduled tasks only for tasks this
runtime owns; use fresh-context sub-agents where the playbooks call for writer/worker isolation;
`**[ACTION REQUIRED]**` blocks and the Next-Action Guidance Rule apply to you as written.
```

---

## What the operator should be able to do

After this contract is in force, all of the following work with no setup explanation from the
human, in any runtime, in any order, at the same time:

- "run campaign {campaign_slug}" — the brain resolves the campaign, takes the campaign lease,
  loads the entrypoint, runs it.
- "enrich lead {name or id}" — contact-scope lease, Stage 4, no interference with a campaign
  run happening elsewhere.
- "write the email for {lead}" — contact-scope lease, email-writing skill, drafts to
  `outbox/pending_approval/`.
- "what is running right now?" — no lease; read the locks, leases and manifest, and answer with
  who holds what.

If a brain cannot do one of these because another brain holds the scope, the answer is the
holder's name and intent plus the free alternatives — never a refusal to participate.
