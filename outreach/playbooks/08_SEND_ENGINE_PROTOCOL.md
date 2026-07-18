# Send Engine Protocol

Stage: `08`

## Load Rule

Load this stage **before any send** — before running `gmail_client.py send`, before approving a batch that will be sent in-session, before a scheduled run reaches its Send step (Daily Run order step 7, DESIGN §15), and any time you reason about the pre-send gate chain, sticky-sender rotation, threading, or the sent log. The send step is the single most consequential side effect OutreachCRM performs: it is the moment an email actually leaves for a real recipient. Never send from a summary of this file — load it in full.

Every load needs a LOAD LEDGER per `playbooks/LOAD_LEDGER_PROTOCOL.md` (read to the last line; compare `playbooks/LOAD_MANIFEST.md` when present). A short read is **NOT** a load: no `email_sent` activity, no `sent_log.jsonl` append, no SMTP call may happen without a `Verdict: PASS` ledger for this stage.

This stage documents the send engine as it is **actually implemented in code** (`tools/gmail_client.py`, `tools/crm_store.py`). These tools **exist** (Phase 1). The DESIGN §22 R2 "`tool_not_built`" degradation does **not** apply to them — a missing send is a real blocker to fix, not a step to skip. Upstream drafting/approval-report stages (05, 06) and inbound-sync/reply stage (10) are still Phase-2 `status: planned`; where this stage references them, follow DESIGN §22 R1 (load the relevant `docs/DESIGN.md` section with its own ledger, record `stage_file_pending`), never a GitHub re-fetch or Last-Resort Recovery.

## Hard Gates For This Stage

- **The approval invariant is absolute.** `gmail_client.py send` refuses any draft whose record `status` is not exactly `"approved"` (returns `blocker: draft_not_approved`). Nothing leaves the system without an explicit chat `approve` recorded in `approvals/approval_log.md`. Default `approval_mode` is `manual_all` — **bumps and replies are approved too**, never auto-sent.
- **All CRM mutations go through `tools/crm_store.py`.** The send engine's suppression writes and its `email_sent` / `email_bounce` / `unsubscribe` activity writes are made through `crm_store` (`gmail_client.py` imports `CrmStore`). The `sent_log.jsonl` and `sync_log.jsonl` are campaign/client append-only artifacts (not `crm/` collections) and are appended with a monotonic `seq`. Never hand-write a `crm/` collection to fake a send.
- **The ordered pre-send re-check runs IN CODE, not in prose.** Do not reorder, skip, or approximate it in a hand-run. If a gate blocks a draft, the draft does not send — surface the blocker, do not work around it.
- **Data root is `daily-content-pipeline/`.** Drafts, sent logs, approvals, and suppression all live under `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outreach/…`. The toolkit repo holds no client data and no secrets.
- **Phase 1 is `plain_text_mode`.** No open pixel, no link rewrite, no tracker `/events` pull. Open/click tracking and the tracker-based unsubscribe pull are Phase 2/3. The `List-Unsubscribe` mailto is always present for CAN-SPAM. Do not claim opens/clicks are measured in Phase 1.
- **Only a reply is conversion evidence.** Opens and clicks (when they exist in a later phase) never trigger a stage change or auto-action. The send engine records; it does not infer intent from delivery signals.
- Every human step in this stage — a re-auth request, a blocked-send handoff, an approval request — uses the `**[ACTION REQUIRED]**` block from `OUTREACHCRM_PLAYBOOK.md` (one purpose, one exact next step, one command or path). When nothing is needed, end with next-action guidance per the Next-Action Guidance Rule.

## Source Preservation Rule

This file is the detailed source material for the send gate. Do not summarize away the pre-send order, the schemas, the error branches, the command forms, or the compliance rules. A downstream agent may summarize its human-facing reply, but it must still obey every requirement here. If you cannot fit the gate chain, load the gate chain — do not reconstruct it from memory.

---

## 1. The Send Command

Sending is one draft per invocation:

```sh
python3 tools/gmail_client.py --client-dir <CLIENT_DIR> send --draft <path/to/draft.json> [--dry-run]
```

- `--client-dir` is the **global** argument (before the subcommand); it points at the client workspace `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outreach/`.
- `--draft` is the path to a single draft record (see §5 / Stage 7 §7.4), normally under `campaigns/{campaign_slug}/outbox/approved/{draft_id}.json`.
- `--dry-run` runs the **full** pre-send re-check (including the atomic quota reservation) and then stops before SMTP. See the dry-run caution in §3.

Related read-only commands (documented for the send workflow; see Stage 2 for auth):

```sh
python3 tools/gmail_client.py --client-dir <CLIENT_DIR> health --sendbox sb-a
python3 tools/gmail_client.py --client-dir <CLIENT_DIR> quota  --sendbox sb-a [--day YYYY-MM-DD]
```

`quota` reports `{cap, sent, reserved, remaining}` where `remaining = max(0, cap − max(sent, reserved))` — reservations count against remaining, not just completed sends.

There is no batch-send flag. A batch is the caller invoking `send` per approved draft, in order, pacing between them (jitter, §6). Approval happens in chat at any time; the scheduled run's Send step (Daily Run step 7) walks `outbox/approved/` and sends each within quota.

## 2. The Approval Invariant (the gate before the gate)

A draft is inert until a human approves it in chat. The chain is:

1. Drafting (Phase-2 Stages 5/6, or the Phase-1 manual loop) writes a draft to `campaigns/{campaign_slug}/outbox/pending_approval/YYYY-MM-DD/{draft_id}.json` with `status: "pending_approval"`.
2. The operator approves in chat using the approval grammar (`approve all` / `approve 1-20, 35` / `reject 7: reason` / `edit 12: …` / `hold 5`). Chat is the write path; editing an HTML report never persists (DESIGN §14).
3. On approval, the draft record is flipped to `status: "approved"` and moved to `campaigns/{campaign_slug}/outbox/approved/{draft_id}.json`, and the decision is appended to `approvals/approval_log.md` (Stage 7 §9.1).
4. Only then may `gmail_client.py send` accept it. In code, `cmd_send` refuses any `status != "approved"` with `blocker: draft_not_approved` before it even reaches the pre-send re-check.

Default `approval_mode: manual_all` (campaign_config, Stage 7 §7.1) means **every** step — the cold step-1, every bump, and every reply draft — requires its own explicit approval. There is no "auto-approve bumps" mode. The rendered Approval Report (HTML, operator-only, NOT scrubbed) is the Phase-2 convenience surface; the invariant it protects is enforced now, in Phase 1, inside `send`.

## 3. The Ordered Pre-Send Re-Check (exact, in code)

`gmail_client.py send` calls `presend_check(...)` on every draft (including `--dry-run`). The order is **load-bearing and fixed** — this is the exact sequence in `tools/gmail_client.py`:

**0. Resolve the contact.** `store.resolve(draft.lead_id)` follows merge chains to the surviving contact (Stage 7 §4.8). If the contact does not exist → `blocker: contact_not_found`. Every later check runs against the resolved survivor, never a tombstone.

**1. Suppression — client tier + agency tier.** Checks the resolved contact's primary email (and that email's **domain**) and phone against **both** `crm/suppression.jsonl` (client tier) and `daily-content-pipeline/suppression/global_suppression.jsonl` (agency tier). Any match → `blocker: suppressed`. Suppression is checked **before** quota is reserved, so a suppressed contact never burns a quota slot. (DESIGN §16 target is "all identities"; Phase-1 `send` checks the primary email + its domain + phone — broaden secondary emails/socials as that lands. Import-time suppression already checks all identities, Stage 3.)

**2. Email channel status.** If `channels.email.status ∈ {opted_out, bounced}` → `blocker: email_channel_not_usable`. A contact whose email channel is `needs_data` also cannot be emailed (no usable address).

**2b. CAN-SPAM sending identity (fail closed).** `config/sending_identity.json` must exist with a
non-empty `physical_mailing_address`, else → `blocker: missing_physical_address`. The engine appends
the compliance footer (postal address + visible opt-out line) to every body from this file (§5); a
client that has not completed Stage 1 §Step-3 cannot send at all.

**3. Guessed-email approval.** If the primary email `status == "guessed_only"` and the draft's `guessed_approved` is not `true` → `blocker: guessed_email_needs_approval`. A guessed/unverified address may only send with an explicit per-draft `guessed_approved: true` flag, never on prose alone. (The daily guessed-send cap and the per-domain guessed kill switch are the DESIGN §9.6 target that lands with the Phase-2 verify/enrich stage; Phase-1 `send` enforces the approval flag.)

**4. Sequence freeze.** If `contact.sequence_state == "frozen"` → `blocker: sequence_frozen`. **Any inbound reply freezes the remaining bumps** for that contact (the sync classifier sets `sequence_state: "frozen"` on a `campaign_reply`). Frozen stays frozen until triage clears it. This gate exists so a person who replied never receives the next scheduled bump.

**5. Step-1 subject lint.** If `step == 1` and the subject matches `^\s*(re|fwd)\s*:` (case-insensitive) → `blocker: step1_subject_looks_like_reply`. A cold first email must have a truthful subject — a fake `Re:`/`Fwd:` is a CAN-SPAM deception (DESIGN §16). Bumps (step > 1) are genuine in-thread replies and are *supposed* to carry `Re:` (§4).

**6. Atomic quota reservation (last).** Reads the sendbox `quota_today` as the cap. If already-sent-today ≥ cap → `blocker: quota_exhausted`. Otherwise `store.reserve(sendbox, day, cap)` atomically reserves a slot; if the reservation is refused (cap reached under the lock) → `blocker: quota_exhausted`. This is the count-then-send race defense — the reservation is committed under the lock **before** the SMTP call, so two concurrent sends cannot both slip past the cap. Reservation is **last** so a draft blocked by any earlier gate never reserves.

The equivalent manual reservation command (the same `reserve` the send path calls internally) is:

```sh
python3 tools/crm_store.py --client-dir <CLIENT_DIR> reserve --sendbox sb-a --day YYYY-MM-DD --cap 40
```

**Dry-run note:** `--dry-run` runs steps 0–6 but **skips the reservation commit** (`presend_check`
is called with `reserve=False`), returning `{would_send_to, subject, rfc_message_id,
list_unsubscribe}` without sending — a dry-run does NOT consume a quota slot (audit fix; the test
suite asserts this). A real send that fails at SMTP releases its reservation (`store.a.release`).

**Phase-1 scope note:** DESIGN §10's fuller chain also lists warmup cap, two-tier domain cap, send-window (recipient timezone), a daily guessed cap, and a **live** tracker/`+unsub` pull that blocks the box if it has not succeeded within N hours. Phase-1 `presend_check` implements resolve → suppression → channel → **sending-identity (2b)** → guessed-approval → sequence-freeze → subject-lint → atomic reservation as above; the warmup/domain/send-window caps and the live pre-send unsub pull arrive with the tracker worker (Phase 2/3). Do not assert a send passed a warmup or send-window gate that Phase-1 code does not run.

## 4. Sticky-Sender Rotation and Threading

**Rotation is step-1 only.** The sendbox for a cold step-1 draft is chosen upstream (drafting orchestration, DESIGN §8): among healthy sendboxes referenced by the campaign, pick the one with the **lowest `sent_today/quota_today` ratio** (round-robin on ties), then pin `contact.assigned_sendbox`. Once set, that box is **sticky** — every later bump and reply for that contact goes from the same box (threading, reply routing, and anti-spam all require it). A broken box is dropped from step-1 rotation; its assigned pending follow-ups **wait** (never reassigned to another box) behind a re-auth `**[ACTION REQUIRED]**` (§7).

**The send engine honors `draft.sendbox`.** `send` does not re-pick a box — it uses the `sendbox` slug stamped on the draft. For a **continuation** (step > 1), it threads by looking up the prior `rfc_message_id` for this `(lead_id, campaign)` in the campaign's `sent_log.jsonl` and setting both `In-Reply-To` and `References` to it, and prefixing the subject with a truthful `Re:` (added only if not already present). Step-1 has no thread refs. This is how a bump becomes a real in-thread reply rather than a fresh cold email.

## 5. MIME Build

`build_mime(...)` constructs an `EmailMessage`:

- **From:** `formataddr((from_name, sendbox.email))` when a `from_name` is present (`draft.from_name` or the sendbox default), else the bare sendbox address.
- **To:** `draft.to`.
- **Subject:** the draft subject; for step > 1, prefixed with `Re: ` unless it already starts with `Re:`.
- **Message-ID:** our own, minted as `<{uuid}@{sendbox_domain}>`. On the `app_password` path SMTP **preserves** our Message-ID on the wire — this is why the app_password mode is the priority path (no provider rewrite). This value is stored as `rfc_message_id` in the sent log and is what future bumps/replies thread against.
- **Date:** RFC-formatted send time.
- **List-Unsubscribe:** always present, as a mailto to the per-box `+unsub` alias carrying the tracker token: `<mailto:{local}+unsub-{token}@{domain}?subject=unsubscribe>`. The token (`draft.token` or a freshly minted 12-char token) is written back onto the draft so the sent-log row carries it; the sync classifier maps an inbound `{local}+unsub-{token}@…` back to the exact lead (Stage 10 / DESIGN §12).
- **Body:** `text/plain` from `draft.body_text` via `set_content`, **with the CAN-SPAM compliance
  footer appended by the engine on every send** — the `-- ` signature separator, then `from_name`,
  the `physical_mailing_address`, and a visible opt-out line, all read from
  `config/sending_identity.json` (gate 2b fails closed when it is missing). The writer never
  hand-types this footer. A `text/html` alternative is added **only** when
  `draft.tracking == "pixel_and_links"` and `body_html` is present (the footer is appended there
  too, HTML-escaped) — that is the Phase-2 tracked mode.

**Phase-1 vs DESIGN §10 headers:** Phase-1 emits the mailto `List-Unsubscribe` only. The additional `https://trk.{domain}/u/{token}` unsubscribe URL and the `List-Unsubscribe-Post: List-Unsubscribe=One-Click` (RFC 8058) header ship with the tracker worker in Phase 2. The mailto alias alone is a working, compliant one-click opt-out for Phase 1.

## 6. Sent Log and Post-Send Actions

On a successful SMTP send, in order:

1. **Append the sent-log row** to `campaigns/{campaign_slug}/sent/YYYY-MM/sent_log.jsonl` (Stage 7 §7.5), stamped with a monotonic `seq` and append-time `ts`:
   ```json
   {"seq":1,"ts":"","lead_id":"","campaign":"","step":1,"sendbox":"","provider_id":"","thread_id":"",
    "rfc_message_id":"","token":"","links":{},"sent_at":""}
   ```
   `lead_id` is the **resolved** survivor; `rfc_message_id` is the on-the-wire Message-ID; `token` is the unsub/tracker token; `links` is the (Phase-1 empty) rewritten-link map; `sent_at` is the domain send time; `thread_id` carries the prior message reference for bumps.
2. **Append an `email_sent` activity** via `crm_store` (`store.log_activity`), summary `sent step {step} via {sendbox}`, `by: agent`, `ref.message_id = rfc_message_id`. This is the CRM event backbone (Stage 7 §4.4).
3. **Mark the draft `sent`.** The draft record's `status` becomes `"sent"` and `decided_at` is stamped; the file is rewritten in place.
4. **Pace between sends (jitter).** `send` handles one draft per call; the 30–180s jitter between successive sends (DESIGN §10.5) is applied by the caller between invocations. Do not fire a whole batch with no gap — pacing protects box reputation.

`send` returns `{ok: true, sent_to, sendbox, rfc_message_id, activity_seq, sent_at}` on success.

## 7. Error Handling

- **SMTP send failure** → returns `{ok: false, blocker: "smtp_send_failed", error: "…"}`. The draft file is **not** modified, so it stays `status: "approved"` on disk with the reservation already spent (§3) — re-running `send` retries it. Surface the blocker; do not fabricate a sent-log row for a send that failed.
- **Sendbox not configured** (`draft.sendbox` has no entry) → `blocker: sendbox_not_configured`.
- **Sendbox not healthy** → `blocker: sendbox_{status}` (e.g. `sendbox_needs_reauth`, `sendbox_paused`). A box goes `needs_reauth` when SMTP/IMAP login fails (`health` detects and records it). Its pending follow-ups wait — never reassign them to another box.
- **Any pre-send gate** → the specific blocker from §3 (`suppressed`, `email_channel_not_usable`, `guessed_email_needs_approval`, `sequence_frozen`, `step1_subject_looks_like_reply`, `quota_exhausted`, `contact_not_found`, `draft_not_approved`).

When a box needs re-auth, hand it off — never try to send around it:

```text
**[ACTION REQUIRED]**

**Client:** {Client Name or workspace}
**I need you to:** re-authenticate sendbox `sb-a` — its SMTP/IMAP login is failing
**Run this outside the AI sandbox:**
```sh
OUTREACHCRM_APP_PASSWORD='<16-char Gmail App Password>' \
  python3 tools/gmail_client.py --client-dir <CLIENT_DIR> auth --sendbox sb-a --email you@gmail.com
```
**Why:** the box is `needs_reauth`; its pending follow-ups are blocked until it is healthy again
```

The App Password is read only from the `OUTREACHCRM_APP_PASSWORD` environment variable — never a CLI argument, never logged, never written to a report or profile. gmail.com App Passwords require 2FA on the account.

## 8. Compliance (CAN-SPAM, encoded)

- **Physical address + working opt-out in every commercial email — enforced in code.** The CAN-SPAM footer is appended by `gmail_client.py` to EVERY outgoing body, from the machine-readable `config/sending_identity.json` (written at Stage 1 §Step-3 alongside the profile's `sending_identity` block); presend gate 2b **fails closed** (`missing_physical_address`) when the file or address is absent. The `List-Unsubscribe` mailto (§5) plus the footer opt-out line give a working opt-out. A campaign cannot be marked ready with `can_spam_physical_address_present: false`.
- **Opt-out honored immediately.** An inbound unsubscribe (mailto `+unsub` alias, negative reply, or remove-intent) routes through `crm_store` suppression (`store.suppress_contact`), which flips the email channel to `opted_out` and writes the suppression row; the next send is blocked at gate 1. OutreachCRM's default is same-run honoring, well inside the 10-business-day legal window (DESIGN §16).
- **Truthful subjects.** Step-1 subjects must not begin `Re:`/`Fwd:` (gate 5 + the Stage 9 audit lint). Bumps are real in-thread replies, so their `Re:` is truthful.
- **Only a reply is conversion evidence.** Opens and clicks — which do not even exist in Phase-1 `plain_text_mode` — never trigger a stage change, a deal, or a suppression flip. The send engine records events; intent is only ever read from an actual reply (classified in Stage 10).

## 9. Completion Gates For A Send Pass

Before claiming a send pass complete:

- Every draft that was sent has a `status: "sent"` record, a `sent_log.jsonl` row (with `rfc_message_id`), and a matching `email_sent` activity — the three reconcile.
- Every draft that did **not** send has an honest disposition: an explicit blocker recorded, or it remains `pending_approval`/`approved` awaiting the next pass. No draft was silently dropped.
- No send occurred for a draft whose `status` was not `"approved"`; every approval is in `approvals/approval_log.md`.
- No send occurred for a suppressed, frozen, opted-out, or unapproved-guessed contact — the pre-send blockers hold.
- Any blocked box or blocked follow-up was surfaced with an `**[ACTION REQUIRED]**` block; when nothing is needed, next-action guidance per the Next-Action Guidance Rule.
- Counts (emails sent, guessed sent, blocked follow-ups) feed the run's reconciliation ledger (Stage 7 §10) and, on a scheduled run, the Stage 9 audit reads them before any completion claim.

---

When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
