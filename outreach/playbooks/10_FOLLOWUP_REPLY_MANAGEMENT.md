# Stage 10 — Follow-up & Reply Management (inbound)

Stage: `10`

## Load Rule

Load this stage before you **read an inbox**, **triage a reply**, or **advise/draft a
follow-up** — i.e. the Daily Run's Sync + Follow-up steps (DESIGN §15, steps 3 and 8), or any
time an operator asks "what came back?" / "who's due for a bump?". This is the inbound half of
the loop: everything a real recipient sends back enters the system here, and every decision that
touches a person who already got an email is made against this contract.

Every load needs a LOAD LEDGER per `playbooks/LOAD_LEDGER_PROTOCOL.md` (read to the last line;
compare `playbooks/LOAD_MANIFEST.md`). A short read is **not** a load: no reply may be classified,
no sequence frozen, no bump drafted from a summary of this file.

The tools this stage drives — `gmail_client.py sync` (DSN-first classifier) and
`crm_store.py` (`apply-rules`, `followups due`) — **exist** (Phase 1 + 2). The DESIGN §22 R2
`tool_not_built` degradation does not apply to them. The reply/bump **draft** itself is written
through Stage 6 (`crm_store.py draft write`); the operator approval + send is Stage 8. Where this
stage references a still-`planned` row, follow DESIGN §22 R1 (load the DESIGN section with its own
ledger), never a GitHub re-fetch.

## Hard Gates For This Stage

- **Any inbound reply freezes the remaining sequence.** The classifier sets
  `sequence_state: "frozen"` on a `campaign_reply`; the send engine's gate 4 (Stage 8 §3) refuses
  the next bump for a frozen contact. A person who wrote back never receives the next scheduled
  step. Frozen stays frozen until triage clears it.
- **The classifier order is load-bearing and runs IN CODE.** DSN/bounce → auto-reply/OOO →
  unsub alias → thread match (reply) → known-contact message → personal (count only). Gmail
  threads DSN bounces into the original thread, so DSN is checked *before* threadId. Do not
  reorder or approximate it in prose (DESIGN §12).
- **A personal email is counted, never read.** Step 6 of the classifier — From is not a known
  contact and matches no thread — increments a counter and stops. Do not store its body, deep-read
  it, or compile anything from it (DESIGN §12, privacy).
- **Only a reply is conversion evidence.** Opens/clicks (absent in Phase-1 `plain_text_mode`
  anyway) never move a stage, create a deal, or flip suppression. Intent is read only from an
  actual reply, classified here.
- **A follow-up adds NEW value or it is not sent.** No "just bumping this" / "circling back".
  Each bump carries one new evidenced thing (Stage 6 `followup.md`); the final step is the
  breakup, then the sequence ends. A stale hook (a listing that sold) must be retired, not reused.
- **Every reply/bump draft still lands in `pending_approval`.** `approval_mode: manual_all` — the
  operator approves each reply and each bump in chat before Stage 8 sends. This stage never calls
  `gmail_client.py send`.
- **`negative` / `remove_intent` → suppression, even without the word "unsubscribe".** A "take me
  off your list", "not interested, stop", or a bare "no" routes through `crm_store` suppression (or
  an `**[ACTION REQUIRED]**` confirm task that blocks further sends). Opt-out is honored same-run.

## Source Preservation Rule

This file is the detailed source for the inbound gate. Do not summarize away the classifier order,
the triage labels, the freeze invariant, or the command forms. When any instruction here disagrees
with `docs/DESIGN.md`, `docs/DESIGN.md` wins.

---

## 1. Sync the inbox (DSN-first classification)

One sync per sendbox, cursor-based (IMAP UID for the `app_password` path):

```sh
python3 tools/gmail_client.py --client-dir <CLIENT_DIR> sync --sendbox sb-a [--max 100]
```

`sync` walks new messages since the box's `imap_uid_cursor` and runs `classify_message` on each in
the **exact deterministic order** of DESIGN §12:

1. **DSN / bounce** — from mailer-daemon/postmaster, `multipart/report; report-type=delivery-status`,
   or a `message/delivery-status` sub-part → `hard` (5.x.x) / `soft` (4.x.x). The original lead is
   mapped via threadId + `rfc_message_id` + recipient/sent-window. A hard bounce → the email channel
   is flipped `bounced` and the address suppressed (no more sends to it).
2. **Auto-Submitted: auto-replied / OOO** — recorded, no sequence change.
3. **Unsub alias** — any `To`/`Delivered-To` matching `{box}+unsub-{token}@…` → token → unsubscribe
   for the exact lead (mailto unsubs often have empty bodies, so the token is the signal, not the text).
4. **Thread match** — threadId / `In-Reply-To` matches the `sent_log` → **campaign reply** →
   `reply_untriaged`, and `sequence_state` is set **frozen**. A reply with no `In-Reply-To` still
   maps via the From-address resolver fallback (Stage 8 note).
5. **Known contact, no thread** → `contact_message` (they wrote from a different address / new thread).
6. **Else** → personal email: **count only.** No body stored, no deep-read.

The first sync of a box baselines the cursor (does not flood the mailbox with historical mail);
the cursor advances only over a contiguous run so nothing is skipped on an error. `sync` appends to
the campaign `sync_log.jsonl` (append-only, monotonic `seq`) and writes the resulting activities
(`email_bounce`, `unsubscribe`, `email_reply`) through `crm_store`.

## 2. Triage the replies (`reply_untriaged` → intent → rules → deal)

A `reply_untriaged` needs a human/agent semantic read into one label, then the rules engine turns
that label into CRM state (a deal, a task, a suppression) deterministically:

| Reply intent | Rule outcome | Next |
|---|---|---|
| `positive` | `reply_positive` → deal `new_reply`, hot-reply task | Draft the confirm + tiny next step (§3) |
| `question` | `reply_question` → deal + task | Answer plainly in the reply, then the next step |
| `objection` | `reply_objection` → deal + task | Address the specific objection with evidence |
| `negative` | `reply_negative` → **suppression** | Sequence ends; no further sends |
| `remove_intent` | **suppression** (same as unsubscribe) | Honored same-run |

Apply the triage through the rules engine (deterministic, idempotent — the same reply activity
never double-creates a deal):

```sh
python3 tools/crm_store.py --client-dir <CLIENT_DIR> apply-rules \
  --event reply_positive --contact <lead_id> --activity <activity_id>
```

`negative` / `remove_intent` route to suppression (which flips the email channel `opted_out` and
writes the suppression row, blocking the next send at Stage 8 gate 1). When intent is ambiguous or
the recipient's wording is hostile, raise an `**[ACTION REQUIRED]**` confirm task instead of
auto-suppressing — but a bare "stop"/"not interested" is unambiguous and is honored directly.

## 3. Draft the reply (they replied — Stage 6)

A frozen contact who replied gets a **human reply that moves toward the goal**, drafted through
Stage 6 (`crm_store.py draft write`, `pending_approval`). The reply is where value is delivered:

- `reply_positive` → confirm + the tiny next step (send the sample, propose the 15-min slot).
- `reply_question` → answer plainly *in the reply*, then the next step.
- `reply_objection` → address the specific objection with evidence, low pressure.

Speed matters — a same-day reply beats next-day — so hot replies surface in the Today View
(Stage 14). Every referenced detail still needs a dossier hook with an `evidence_url`;
`do_not_mention` still applies; nothing sends without approval.

## 4. Silent-lead bumps (nobody replied — step > 1)

Contacts who did **not** reply and whose `gap_days` has elapsed are due for a bump:

```sh
python3 tools/crm_store.py --client-dir <CLIENT_DIR> followups due --campaign <slug>
```

`followups due` returns, per campaign, each lead whose latest sent step N has passed its
step-(N+1) `gap_days`, who is **not** frozen (no reply — and an unsubscribe or hard bounce also
freezes), whose sequence is **not** exhausted (the breakup step is the last one), and who does
**not already have a bump draft awaiting approval** (dedupe — an unapproved bump is never
re-drafted daily; `draft write` also refuses a duplicate `(lead, step)` with
`duplicate_pending_draft`). Bumps share the campaign's daily draft budget and must leave the
`new_lead_floor` slots for step-1 intake (Stage 5); reply drafts are exempt (`is_reply: true`). For each, the bump draws first on the **reserved** Layer-B from the
initial enrichment plus the campaign's `message_bank` — no new collector call needed. A **micro-refresh
is opportunistic**: run it only when the reserved points are used up and the collector has spare
capacity, gated by the send budget, never as a per-bump requirement (enrichment must not scale with the
in-flight bump count). Retire a stale hook (a sold listing) whenever you do refresh. Then Stage 6 drafts
the bump (`step > 1`): sticky sendbox, threads off the prior
`rfc_message_id`, subject may keep a truthful `Re:`. The final step is the breakup (easy out, door
left open); after it, the sequence ends for that contact. Every bump is a distinct evidenced angle
— never a repeat of step 1 (Stage 6 `followup.md`).

## 5. The approval gate (cross-ref Stage 8)

Reply drafts and bumps are drafts like any other: they collect in `pending_approval` and appear in
the next **Approval Report** (`crm_store.py approval-report`) — specifically in its dedicated
**`## Follow-ups due (n)`** section (*bumps and reply drafts — threaded onto an existing
conversation*). Because they are step>1, they are grouped apart from new step-1 leads, which stay
in **High confidence** / **Review carefully**; numbering is stable and unique across all sections.
They are approved in chat with the approval grammar (`approve all` / `approve 1-20,35` /
`reject 7: reason` / `hold 5` / `edit 12: …`) via `crm_store.py approve`. Approval flips `status: approved` and moves the draft to
`outbox/approved/`; only then may Stage 8 send it. The full gate chain is Stage 8 §2–§3.

## Completion Gates For An Inbound Pass

- Every synced message got a classification; DSN/bounce was checked before thread match; no personal
  email body was stored.
- Every hard bounce flipped its email channel `bounced` and suppressed the address; every unsub alias
  and every `negative`/`remove_intent` reply routed to suppression (or a blocking confirm task).
- Every campaign reply set `sequence_state: frozen`; no frozen contact has a pending bump queued.
- Every reply/bump draft is in `pending_approval` (none sent from this stage); reply drafts and bumps
  each carry an evidenced hook and honor `do_not_mention`.
- Due-silent bumps came from `followups due` (not a hand-scan), each carries a NEW evidenced angle,
  and no exhausted sequence was re-bumped.
- Any ambiguous intent or blocked action was surfaced with an `**[ACTION REQUIRED]**` block; when
  nothing is needed, next-action guidance per the Next-Action Guidance Rule.

## Phase status

2D (this stage's tooling — the DSN-first `gmail_client.py sync` classifier from Phase 1, plus
`crm_store.py apply-rules` / `followups due` and the Approval Report + chat-approve handler) is
**built**. Reply/bump **drafting** is Stage 6 (2C, built); the **send** is Stage 8 (Phase 1, built).
The tracker-based unsub pull and open/click signals remain Phase 3 — do not assert them here.

When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
