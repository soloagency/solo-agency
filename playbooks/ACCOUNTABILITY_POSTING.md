# Posting Accountability — the retention module

## Load Rule

Load on every scheduled run alongside Stage 5, whenever the client has any published
history (or a first video was delivered more than 7 days ago). LOAD LEDGER applies.

## Why this module exists

Clients renew when they see results; results come from posting consistently. A client who
stops posting stops seeing value and churns — silently. This module is the machine's half of
retention: detect the silence early, remind the client with the "consistent AND sufficient"
lesson, and hand the operator the moment a personal call is needed. The reminder is a SERVICE
("your next video is ready"), never a scolding.

## The check — every run, any cadence

1. `last_posted_at` = the newest publish timestamp available, in priority order: provider
   publish history (already fetched by Stage 5 — reuse, never re-query), the local publishing
   ledger, and (fallback, when both are empty and the client's own page is an approved
   exclusive source) the latest post visible in that page's most recent collector scan.
   Record the value AND which source produced it. No source at all →
   `accountability_status: no_data`, note it in INTERNAL_REPORT, stop here — never guess.
2. `gap_hours = now − last_posted_at`, computed in the client's timezone.
3. Threshold: `accountability_max_gap_hours` from `system_settings.json` (`tool
   system-settings get` — operator-editable at `/ui/settings`, default 72h: a weekend gap,
   Saturday→Monday, is normal). A per-client override in the profile
   (`accountability.max_gap_hours`) wins when present; `accountability.enabled: false`
   disables the module for that client.

## The reminder ladder — max one per run, max three per episode

An "episode" starts when `gap_hours` first exceeds the threshold and ends the moment the
client posts again (any new publish timestamp → reset everything).

- Gap over threshold and fewer than 3 reminders sent this episode → send ONE reminder this
  run. Never more than one per run; never re-send when this run already reminded.
- State lives in `analytics/accountability_log.md` per client: episode start, reminders sent
  (ts + gap at send), operator escalation ts. Read it BEFORE reminding — the log is the dedup
  gate, same pattern as the notification log.
- After the 3rd reminder, no further client reminders this episode — escalate to the
  OPERATOR instead (below), once per episode.

## The client reminder — its own notification, its own tone

A SEPARATE `sendNotification` message (never folded into the report notification), fully
under the Client Notification Contract (playbook 03): client's report language, client-blind,
plain words. Content shape:

- Subject: the fact, gently — "Đã N ngày kênh của anh/chị chưa có video mới".
- The "Đều VÀ đủ" lesson in one short paragraph, audience language: platforms reward
  accounts that post consistently and sufficiently; every gap resets momentum with the
  algorithm and with viewers — the compounding is the value.
- The SERVICE hook, not a scolding: what is already waiting — the queued ideas / ready
  video in their production plan, with the no-login Saved Ideas link (mint if no fresh one
  exists). "Video kế tiếp đã sẵn — chỉ cần đăng."
- One ask only: post it (or reply if something is blocking them). Signature per profile.

## Operator escalation — after the 3rd unheeded reminder

Once per episode, email the operator through their own outreach infrastructure:

```
tools/solo_tool gmail send-operator --pipeline {setup-root}/daily-content-pipeline \
  --subject "{Client}: im lặng {D} ngày, đã nhắc 3 lần" \
  --body-file {a short operator note}
```

The tool picks a healthy sendbox automatically (any of the operator's app-password Gmail
accounts) and sends to `operator_email` from `system_settings.json` — nothing to configure
per client. The body is OPERATOR-facing (internal detail allowed): last post date, gap,
the 3 reminder timestamps, engagement trend, and the one-line recommendation ("cần anh gọi
trực tiếp"). Log the escalation in `accountability_log.md` and INTERNAL_REPORT. This email
IS the point of the module: the machine nags, the human saves the account.

## Edge cases

- Never-posted client (no publish history at all): no reminders — accountability starts
  after the first post. Exception: if the first video was DELIVERED more than 7 days ago and
  nothing was ever posted, send one gentle first-post nudge (same service tone), once.
- A client on a posting cadence slower than the threshold by operator agreement: set the
  per-client override, don't let the global default nag them.
- Reminders count toward the run's notification log with their own event code
  (`accountability_reminder`, `accountability_escalation`) — dedup by episode, not by day.

## Fleet snapshot — feed the operator dashboard, never pull

At the END of every run (after the report set is reconciled), write
`daily-content-pipeline/fleet/{client_slug}.json` with everything this run learned (schema
in Stage 7). The dashboard reads these files; nothing ever queries providers on page load.
Unavailable values are null/absent — never invented. The accountability fields
(`posting.gap_hours`, `accountability.reminders_sent_this_episode`, escalation state) come
straight from this module's check.
