# Sendbox Setup

Stage: `02`

## Load Rule

Load this stage whenever you connect, re-authenticate, health-check, or inspect the quota of a sendbox: Setup Flow Step 4 (connect the first sendbox), the add-sendbox flow, a `needs_reauth` recovery, and the start of any run that will send or sync (Stage 8 send / Stage 10 sync both assume a healthy, registered box). It must be loaded together with Stage 0 (`playbooks/00_CORE_CONTEXT_REQUIREMENTS.md`) before you write any sendbox config or run `gmail_client.py`.

This stage owns the **sending-and-reading identity** for a client: how a `@gmail.com` mailbox is authenticated (App Password priority; OAuth/Workspace advanced), what `sendboxes/sendboxes.json` records, how warmup ramps, how multiple boxes rotate, and how a broken box degrades. The `sendboxes.json` and credentials layout is defined canonically in Stage 7 (`playbooks/07_STORAGE_SCHEMA_AND_HISTORY.md` §6) and DESIGN §8; this stage owns the connect/verify workflow, not the schema itself.

This file is loaded, not summarized. A short read is NOT a load: register a LOAD LEDGER entry per `playbooks/LOAD_LEDGER_PROTOCOL.md`, checked against `playbooks/LOAD_MANIFEST.md` when present, before taking any side-effect action (authenticating a box, writing `sendboxes.json`, writing credentials).

## Hard Gates For This Stage

- **Never ask for the App Password in chat, and never pass it as a CLI argument.** The App Password reaches `gmail_client.py` **only** through the environment variable `OUTREACHCRM_APP_PASSWORD`, which the human sets locally. It must never appear in a chat message you request, a command line, a log, a report, the profile, or any committed file.
- **Never ask for the Google account password, cookies, or OTP.** An App Password is a scoped app credential, not the account password. That is the only sending secret Setup Flow collects for the App Password path.
- **All sending secrets stay local and private.** `sendboxes/{sendbox_slug}/credentials.json` (and `token.json` for OAuth) are gitignored and written `chmod 600`. They are never committed, never printed, never copied into a report or the Client Intelligence Profile. The deploy script blocks staging of `token.json` / `client_secret*.json` and secret-scans the staged diff.
- **`gmail_client.py` is the only sanctioned way to authenticate a box.** Do NOT improvise a one-off `smtplib`/`imaplib` connectivity script (the no-one-off-scripts rule holds). The tool verifies SMTP+IMAP without sending outbound mail and writes the registry entry atomically.
- **Setup Flow never sends.** Connecting and health-checking a sendbox is a connectivity check, not a cold send. Setup Flow does not send a test email, cold or otherwise; its terminal state stays `ready_for_automation_first_run`.
- **The `@gmail.com` App Password path is the priority path**; OAuth/Workspace is the advanced fallback. Do not push a client onto OAuth unless they run Google Workspace on a custom domain and explicitly want it.
- **Data root is `daily-content-pipeline/`.** Every sendbox artifact lives under `daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outreach/sendboxes/`. The toolkit/source repo holds no sendbox data or secrets.
- Every human step in this stage — the App Password creation request, the re-auth request, any blocker — uses the `**[ACTION REQUIRED]**` contract from `OUTREACHCRM_PLAYBOOK.md`: one purpose, one exact next step, one command or path. When nothing is needed, end with next-action guidance per the Next-Action Guidance Rule.
- This stage must be loaded IN FULL (LOAD LEDGER printed with `Verdict: PASS`, matching `LOAD_MANIFEST.md` when present) before any side-effect write.

## Source Preservation Rule

This file is detailed source material for connecting and operating sendboxes. Do not summarize away the auth steps, the schema, the warmup ramp, the rotation rules, the security requirements, the exact commands, or the warnings. A downstream agent may summarize its human-facing *response*, but it must still obey the full requirements here. If you cannot fit the schema, load the schema — do not reconstruct it from memory.

---

## 1. Two Auth Modes, One Interface

A **sendbox** is one mailbox OutreachCRM sends from and reads replies into. Every sendbox is authenticated in one of two modes, and both are driven through the same tool, `tools/gmail_client.py`:

- **`app_password` — the priority path for `@gmail.com`.** SMTP send (`smtp.gmail.com:465`, SSL) + IMAP read (`imap.gmail.com:993`, SSL) via Python stdlib (`smtplib`/`imaplib`). No OAuth, no 7-day refresh-token expiry, and it preserves our own `Message-ID`. This is what almost every client uses. Requires 2-Step Verification on the Google account and a 16-character App Password.
- **`oauth` — advanced (Google Workspace / custom domain).** Gmail API with scopes `gmail.send + gmail.readonly` only (`gmail.modify` is deliberately dropped). Covered in §9; use it only for a Workspace/custom-domain box. It is a later addition behind the same CLI.

The pre-send re-check chain, atomic quota reservation, and inbound classifier are implemented **in code** inside `gmail_client.py` (Stages 8 and 10), not in playbook prose. This stage gets you a healthy, registered box; Stage 8 uses it to send, Stage 10 uses it to sync.

`--client-dir DIR` below always means the client pipeline directory:

```text
daily-content-pipeline/clients/{client_slug}/{business_slug}_{location_slug}/outreach/
```

**Command form (verified):** `--client-dir DIR` is a global flag that comes **before** the subcommand. `gmail_client.py auth --client-dir DIR …` (subcommand first) is rejected by the parser.

---

## 2. The App Password Path (priority for `@gmail.com`)

### 2.1 Create the App Password (human step)

The App Password is a Google-tightened surface and requires 2-Step Verification. Ask the human with a single `**[ACTION REQUIRED]**` block. Never ask them to paste the password into chat.

```text
**[ACTION REQUIRED]**

**Client:** {Client Name}
**I need you to (all local, nothing pasted to me):**
  1. Open Google Account → Security for the sending Gmail account.
  2. Turn ON 2-Step Verification if it is not already on (App passwords require it).
  3. Go to Security → App passwords → generate a new 16-character App Password.
  4. In your local terminal, set it as an environment variable (do NOT send it to me):
       export OUTREACHCRM_APP_PASSWORD="the 16-char app password"
  5. Reply here only with the sending address, e.g. `sendbox: outreach@gmail.com`.
**Why:** OutreachCRM sends and reads that mailbox over SMTP/IMAP using the App Password.
It is scoped to this app, never your main Google password, and I never see it — the tool
reads it from your environment variable.
```

Notes:
- The App Password is often shown with spaces (e.g. `abcd efgh ijkl mnop`); the tool strips spaces, so either form of the env value works.
- If the human reports there is no "App passwords" option, 2-Step Verification is not fully enabled — that is the blocker to surface, not an error to retry around.

### 2.2 Authenticate and register the box

Once the human has set `OUTREACHCRM_APP_PASSWORD` locally and given you the sending address, run the auth subcommand. Pick a short `sendbox_slug` per the slug rules (lowercase, hyphens, no punctuation) — `sb-a`, `sb-b`, … are the conventional slugs.

```bash
python3 tools/gmail_client.py --client-dir DIR auth --sendbox sb-a --email outreach@gmail.com
```

What `auth` does, in code:
1. Reads the App Password from `OUTREACHCRM_APP_PASSWORD` (exits with an instruction if unset — it is never a CLI arg).
2. Verifies **both** channels: an SMTP SSL login to `smtp.gmail.com:465` and an IMAP SSL login to `imap.gmail.com:993`. No outbound mail is sent.
3. Writes `sendboxes/sb-a/credentials.json` (`{email, app_password, smtp_host, imap_host}`) and `chmod 600`s it.
4. Registers/updates the box in `sendboxes/sendboxes.json` with `auth_mode: app_password`, the derived `domain`, `status: healthy`, and — for a new box — `quota_today: 20`, `warmup_stage: week_1` (existing values are preserved on re-auth).

A successful run prints JSON like `{"ok": true, "sendbox": "sb-a", "email": "…", "smtp": "ok", "imap": "ok", "quota_today": 20, "warmup_stage": "week_1"}`. If SMTP or IMAP login fails, the login raises and the box is **not** registered — surface it as an `**[ACTION REQUIRED]**` (usually: 2-Step Verification off, wrong/rotated App Password, or the env var not exported in the same shell).

### 2.3 Phase note (these tools now exist)

`gmail_client.py`, `import_leads.py`, `email_verify.py`, and `crm_store.py` are **built** (Phase 1). The DESIGN §22 R2 `tool_not_built` degradation path — writing the sendbox as `status: pending_connectivity_check` and recording `gmail_client_auth_pending` — **no longer applies to sendbox auth**: run the real connectivity check now. (Stage-1 Setup Flow prose that describes the pending path was written for Phase 0; with `gmail_client.py` present, authenticate for real.) The Phase-2 stage files (`04_VERIFY_ENRICH.md`, `05_CAMPAIGN_MANAGEMENT.md`, `06_EMAIL_WRITING_STANDARD.md`, `10_FOLLOWUP_REPLY_MANAGEMENT.md`, `13_CRM_CORE.md`, `14_TASKS_TODAY_VIEW.md`) are now **built** (2A–2D); only Stages 12/15 remain `status: planned` (Phase 3), and a load of one of those follows DESIGN §22 R1 (load the DESIGN section, record `stage_file_pending`), not a tool-missing blocker.

---

## 3. The `sendboxes.json` Schema

`sendboxes/sendboxes.json` is the registry of every box for a client (DESIGN §8, Stage 7 §6.1). One object per box:

```json
{"sendboxes":[
  {"slug":"sb-a","auth_mode":"app_password|oauth","email":"outreach@gmail.com","domain":"gmail.com",
   "quota_today":40,"warmup_stage":"week_1|week_2|mature","status":"healthy|needs_reauth|paused",
   "historyId":null,"imap_uid_cursor":null,"last_successful_sync_ts":""}]}
```

Field enums and meaning:
- **`slug`** — the sendbox slug; the credentials folder is `sendboxes/{slug}/`.
- **`auth_mode`** ∈ `app_password | oauth`.
- **`email`** / **`domain`** — the sending address and its host; `domain` groups boxes for the domain-tier cap (§5) and the guessed-email per-domain kill switch.
- **`quota_today`** — the operator-set daily cap for this box (the warmup number; see §4).
- **`warmup_stage`** ∈ `week_1 | week_2 | mature`.
- **`status`** ∈ `healthy | needs_reauth | paused`. Only a `healthy` box sends; `health` flips this automatically, and a send error can set it (Stage 8).
- **Cursors:** `historyId` (OAuth) or `imap_uid_cursor` (app_password), plus `last_successful_sync_ts`. These are advanced by inbound sync (Stage 10) — do not hand-edit them.

`sendboxes.json` is plain config (not a `crm/` collection), so it is a normal config write, not a `crm_store.py` mutation. It carries **no secrets** — the App Password lives only in `credentials.json`. The Client Intelligence Profile references boxes by slug in its `sending_identity.sendboxes[]` block; it never stores the credential.

---

## 4. Warmup Ramp (documented, operator-set)

New mailboxes must ramp volume slowly or they get flagged. The ramp is documented policy that the operator sets on the box's `quota_today`; the tool does not auto-increase it.

- **Start at 20/day** (`warmup_stage: week_1`; this is the default `quota_today` a freshly authed box gets).
- **Increase by roughly +5 per week** as the box behaves (few bounces, some replies, no spam complaints): ~25 in week 2, then continue upward.
- **Mature at roughly ~50/day** (`warmup_stage: mature`). Consumer `@gmail.com` boxes should not be pushed above ~50 cold/day (§8).
- Advance `warmup_stage` (`week_1 → week_2 → mature`) and raise `quota_today` deliberately, in step with the ramp — never jump a cold box straight to 50. Re-auth preserves the existing `quota_today`/`warmup_stage`, so raising the ramp is an explicit `sendboxes.json` edit, not a side effect of re-authenticating.

Real scale is not one big box: it is **2–3 variant domains, 1–2 boxes each**, each warmed independently, so domain reputation ramps alongside box volume.

---

## 5. Multi-Sendbox Rotation, Sticky Sender & Two-Tier Caps

When a client has more than one healthy box, the rules below govern which box a contact is emailed from. Selection happens at draft/queue time (Stage 5/8) and is recorded on the draft's `sendbox` field and on `contact.assigned_sendbox`; `gmail_client.py send` then honors that box and performs the atomic per-box quota reservation.

- **Rotation is step-1 only.** The very first outreach to a contact picks the **healthy** referenced box with the lowest `sent_today / quota_today` ratio (round-robin on ties). This spreads first-touch volume across boxes and domains.
- **Sticky sender thereafter.** Once a contact's first email goes out, `contact.assigned_sendbox` is fixed. Every bump and every reply for that contact goes from the **same** box — threading, reply routing, and anti-spam all require it. A contact is never re-rotated to a different box mid-sequence.
- **Two-tier cap.** Effective daily headroom for a first touch is `min(remaining_box_quota, remaining_domain_cap)` — several boxes on one domain share that domain's reputation, so the domain's combined volume is also capped and ramps. The per-box side is enforced in code by the atomic reservation (`reserve(sendbox, day)`); the domain-tier cap is an operator-set ramp applied during selection.

---

## 6. Broken Box Handling

A box whose SMTP/IMAP login fails (`status: needs_reauth`) or that the operator has `paused` is **broken** for sending:

- It is **dropped from step-1 rotation** immediately — no new first-touch contacts are assigned to it.
- Its already-assigned pending follow-ups **wait**; they are **never reassigned** to another box (sticky sender is non-negotiable — reassigning would break threading and reply routing). The daily-ops/report shows `N follow-ups blocked` for that box.
- Surface it with an `**[ACTION REQUIRED]**` re-auth request. Re-authentication uses the same auth command; the human re-sets `OUTREACHCRM_APP_PASSWORD` (rotating the App Password if it was revoked) and you re-run auth, which flips the box back to `healthy` while preserving its `quota_today`/`warmup_stage`/cursor:

```text
**[ACTION REQUIRED]**

**Client:** {Client Name}
**I need you to:** re-authorize sendbox `sb-a` ({address}@gmail.com). In your local
terminal, `export OUTREACHCRM_APP_PASSWORD="…"` (generate a fresh 16-char App Password if
the old one was revoked), then reply `ready`.
**Why:** `sb-a` failed its SMTP/IMAP login, so it is dropped from new sends and {N}
assigned follow-ups are waiting on it. I do not move those to another box — that would
break the reply threading.
```

After the human replies `ready`, run:

```bash
python3 tools/gmail_client.py --client-dir DIR auth --sendbox sb-a --email outreach@gmail.com
```

---

## 7. Health & Quota Commands

Both are read/verify commands — neither sends mail.

**Health** re-runs the SMTP and IMAP logins and writes back `status`:

```bash
python3 tools/gmail_client.py --client-dir DIR health --sendbox sb-a
```

Prints `{"sendbox":"sb-a","email":"…","smtp":"ok|fail:…","imap":"ok|fail:…","status":"healthy|needs_reauth"}`. If either channel fails, the box is set `needs_reauth` in `sendboxes.json`. Run health before a sending run and as the first diagnostic when a send blocks.

**Quota** reports today's headroom for a box (optionally for a specific day):

```bash
python3 tools/gmail_client.py --client-dir DIR quota --sendbox sb-a
python3 tools/gmail_client.py --client-dir DIR quota --sendbox sb-a --day 2026-07-16
```

Prints `{"sendbox","day","cap","sent","reserved","remaining"}` where `cap` is `quota_today`, `sent` counts real sends from the sent log, `reserved` counts atomic quota reservations, and `remaining = max(0, cap - max(sent, reserved))`. Use it to see how much of the warmup cap is left before drafting more first touches.

---

## 8. Consumer `@gmail.com` Limits & `plain_text_mode`

These limits are **documented and accepted**, not bugs to work around:

- **`plain_text_mode` is the default for `@gmail.com` boxes.** Because the From is `gmail.com` and any tracking domain would be unrelated, there is **no open pixel and no link rewrite** — measure by reply. Phase 1 is plain-text throughout; the `List-Unsubscribe` mailto alias (`{box}+unsub-{token}@…`) is **always** present because it is compliance, not tracking.
- **No custom `Message-ID` domain** — SMTP preserves our own `Message-ID`, but it is on `gmail.com`.
- **~20–50 cold emails/day/box**, per the warmup ramp (§4).
- **Never use the operator's primary Gmail** as a sendbox. Cold bulk from a personal primary risks the account. Use a dedicated address.
- **Cold bulk risks account suspension** at volume; this is accepted only at low volume with tight personalization.
- **App Password requires 2-Step Verification** and is a Google-tightened surface. Keep the OAuth mode (§9) available as a fallback if Google restricts App Passwords for an account.

---

## 9. The `oauth` Mode (advanced, Workspace / custom domain — later)

Use OAuth only for a Google Workspace box on a domain the client controls:

- Gmail API with scopes **`gmail.send + gmail.readonly` only** (`gmail.modify` intentionally dropped).
- The OAuth app should be **Internal** to the Workspace to avoid the 7-day refresh-token expiry. If it is forced External/testing, weekly re-auth becomes a scheduled **day-6 `**[ACTION REQUIRED]**`**, treated as routine maintenance, not an error path.
- OAuth boxes store `sendboxes/{slug}/token.json` (gitignored, `chmod 600`) in addition to the credentials reference, and use the `historyId` cursor for sync. `token.json` and `client_secret*.json` are blocked from staging by the deploy script.
- A custom-domain box unlocks the tracker worker on `trk.{domain}` (open pixel, link rewrite, one-click HTTPS `/u/` unsubscribe) — that is Phase-2/3 wiring (Stage 8/11), not part of connecting the box here.

Full OAuth connect flow lands with the OAuth path; for Phase 1, prefer the App Password path above.

---

## 10. Credentials Security (non-negotiable)

- `sendboxes/{slug}/credentials.json` and `sendboxes/{slug}/token.json` are **gitignored and `chmod 600`**. The tool writes them with those permissions; do not loosen them.
- The App Password lives **only** in the human's local `OUTREACHCRM_APP_PASSWORD` env var (transiently) and in `credentials.json` (at rest, mode 600). It is never in a chat message, a command line, a log, a report, or the Client Intelligence Profile.
- The deploy script secret-scans the staged diff (`refresh_token`, `client_secret`, `TRACKER_API_KEY`) and refuses to stage `token.json` / `client_secret*.json`. Do not defeat these checks.
- Reading `sendboxes.json` (no secrets) for debugging is fine; printing or copying `credentials.json`/`token.json` is not.

---

## 11. Completion Gates For This Stage

A sendbox is "connected" when all of the following hold:

- `gmail_client.py auth` returned `ok: true` with `smtp: ok` and `imap: ok` — the box authenticated for real (not a Phase-0 pending placeholder).
- `sendboxes/sendboxes.json` has the box with `auth_mode`, `email`, `domain`, `status: healthy`, a warmup-appropriate `quota_today`, and `warmup_stage`.
- `sendboxes/{slug}/credentials.json` exists, is `chmod 600`, and is gitignored — and no secret was ever exposed in chat, a command, a log, or the profile.
- The box is referenced by slug in the Client Intelligence Profile `sending_identity.sendboxes[]`, and any change here was carried through Automation Resync so the next scheduled run reads it.
- No cold email was sent during setup — connectivity was verified without sending.

Surface any unmet gate with the `[ACTION REQUIRED]` contract: one purpose, one exact next step, one command or path. When nothing is needed, end with next-action guidance per the Next-Action Guidance Rule.

---

This is the intended operating model for sendbox setup. When any file disagrees with `docs/DESIGN.md`, `docs/DESIGN.md` wins.
