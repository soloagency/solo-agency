> **HISTORICAL (Python era).** This runbook drove the retired Python implementation.
> Since 2026-07-19 the acceptance gates are: the bridge-go test suite
> (`solo-agency-collector/bridge-go`, `go test ./...` — behavioral scenario tests that were
> golden-cross-validated against Python before retirement) plus the live sendbox validation
> recorded in `docs/UI_DESIGN.md`'s delivery log. Command snippets below are NOT runnable as-is.

# OutreachCRM — Phase 2 E2E acceptance runbook (real send, one sendbox)

This is the **Phase-2 acceptance gate** (DESIGN §21). Unlike the offline unit suites
(`test_phase1.py` + `test_phase2.py`, 73 cases), it exercises the whole loop end to end with a
**real Gmail send + real reply sync** against the three Max Output test inboxes, then walks the
Phase-2 stages that sit on top: campaign → enrich → draft → **Approval Report + chat-approve** →
send → **follow-up bump** (via the injected clock) → **weekly client report** (scrub-gated) →
**operator notify** (dry-run). Steps 0–6 are the Phase-1 core; Steps 7–12 are Phase-2. Run the
whole thing once before going live and after any change to the send path or the approval/report
tooling. Target: **~25 assertions**.

Prerequisites:
- A **dedicated** Gmail account for sending (NOT your primary). 2-Step Verification ON, and a
  16-char **App Password** generated (Google Account → Security → App passwords).
- Python 3.9+. No pip install needed (App Password path is stdlib-only).
- Access to the three test inboxes to act as the recipients:
  `huubinhnguyen81@gmail.com`, `leadup@livechatwith.us`, `tainguyenvdcc@gmail.com`.

Set once:
```bash
cd /Users/binhnguyen/Downloads/outreachcrm
export PIPE=/tmp/outreach-e2e            # throwaway pipeline root for the test
export SENDER=your-dedicated@gmail.com   # the sending Gmail
export OUTREACHCRM_APP_PASSWORD='xxxxxxxxxxxxxxxx'   # 16-char app password, no spaces
```

> **Injected clock (Steps 10–11).** The follow-up and weekly-report steps advance time with
> `OUTREACHCRM_FAKE_NOW` under `OUTREACHCRM_TEST_MODE=1` (DESIGN §17). That gate exists so a real
> scheduled run can never have its send timestamps shifted: **`OUTREACHCRM_TEST_MODE` must NEVER be
> set in a live automation task** — it is a runbook-only switch. Unset both when you finish.

## Step 0 — Reset (idempotent; safe to re-run the whole runbook)
```bash
rm -rf "$PIPE"
<bridge> tool crm-store --pipeline "$PIPE" --client max-output --business ai-automation --location hcmc init-client
export CDIR=$(python3 -c "import glob;print(glob.glob('$PIPE/clients/max-output/*')[0])")
echo "CDIR=$CDIR"
```
(For a client that already exists, the sanctioned reset is
`<bridge> tool crm-store --client-dir "$CDIR" reset-client --confirm`, which also wipes
`test_fixture`-tagged suppression so `tainguyenvdcc@` is not permanently blocked between runs.)

## Step 1 — Connect the sendbox
```bash
<bridge> tool gmail --client-dir "$CDIR" auth --sendbox sb-a --email "$SENDER"
<bridge> tool gmail --client-dir "$CDIR" health --sendbox sb-a   # expect smtp/imap: ok
```
**Assert:** `sendboxes/sb-a/credentials.json` exists and is `-rw-------` (chmod 600);
`sendboxes/sendboxes.json` lists `sb-a` status `healthy`.

## Step 2 — Import the fixture list
```bash
<bridge> tool import-leads inspect --file tests/fixtures/max_output_list.csv
<bridge> tool import-leads import --client-dir "$CDIR" --file tests/fixtures/max_output_list.csv --list-slug max-output
```
**Assert:** `contacts_created: 5` — the three real inboxes **plus** the no-email realtor
(imported via its phone identity) **plus** `bounce-test@…invalid` (imported with email status
`email_not_found`; MX-fail marks status but does not drop the row). `matched:0 suppressed:0`.
`tainguyenvdcc@` is imported now (it becomes suppressed only in Step 5).

## Step 3 — Approve + send a step-1 email to inbox #1
Build a draft (normally Stage 6 writes this; here we author it by hand), approve, send:
```bash
LEAD=$(python3 -c "import sys; sys.path.insert(0,'tools'); from crm_store import CrmStore; s=CrmStore('$CDIR'); print(next(c['id'] for c in s.a.query('contacts') if any(e['address']=='huubinhnguyen81@gmail.com' for e in c['identities']['emails'])))")
echo "LEAD=$LEAD"
mkdir -p "$CDIR/campaigns/demo/outbox/approved"
cat > "$CDIR/campaigns/demo/outbox/approved/d1.json" <<JSON
{"id":"draft_e2e_1","schema_version":1,"lead_id":"$LEAD","campaign_slug":"demo","step":1,
 "sendbox":"sb-a","to":"huubinhnguyen81@gmail.com","subject":"Quick question about your listings",
 "body_text":"Hi — testing OutreachCRM Phase 1. Please reply 'yes' to this email.\n\n— E2E",
 "confidence_band":"high","tracking":"plain_text","status":"approved","guessed_approved":false}
JSON
<bridge> tool gmail --client-dir "$CDIR" send --draft "$CDIR/campaigns/demo/outbox/approved/d1.json" --dry-run   # inspect gates first
<bridge> tool gmail --client-dir "$CDIR" send --draft "$CDIR/campaigns/demo/outbox/approved/d1.json"            # real send
```
**Assert:** the email arrives at `huubinhnguyen81@gmail.com`; `sent_log.jsonl` has one row with a
real `rfc_message_id`; an `email_sent` activity exists on the contact; the draft file is now
`status:"sent"`. The message has a `List-Unsubscribe: <mailto:...+unsub-...>` header.

## Step 4 — Reply positive, then sync
From `huubinhnguyen81@gmail.com`, **reply** to that email (keep the `Re:` subject). Then:
```bash
<bridge> tool gmail --client-dir "$CDIR" sync --sendbox sb-a
```
**Assert:** the sync output shows `campaign_reply: 1` and a `replies_untriaged` entry for the
lead; the contact's `sequence_state` is now `frozen` (any reply freezes the sequence).
Run the rule:
```bash
<bridge> tool crm-store --client-dir "$CDIR" apply-rules --event reply_positive --contact "$LEAD" --activity <activity_seq_from_sync>
```
**Assert:** a deal at `new_reply` exists; a "Reply within 4h" task is open. Re-running the same
`apply-rules` is a no-op (idempotent).

## Step 5 — Unsubscribe path (inbox #3)
Send a step-1 email to `tainguyenvdcc@gmail.com` (repeat Step 3 with that address). Each sent
email carries a `List-Unsubscribe: <mailto:{sendbox}+unsub-{token}@gmail.com>` header. In Phase 1
the **deterministic** opt-out is that mailto alias: from `tainguyenvdcc@gmail.com`, **click the
List-Unsubscribe link** (or send/reply to the `{sendbox}+unsub-{token}@gmail.com` address — the
mail client's "Unsubscribe" button does exactly this). Then sync:
```bash
<bridge> tool gmail --client-dir "$CDIR" sync --sendbox sb-a
<bridge> tool crm-store --client-dir "$CDIR" suppress check --email tainguyenvdcc@gmail.com
```
**Assert:** `suppressed: true`. (A plain reply whose *body* says "unsubscribe" freezes the
sequence but does not auto-suppress in Phase 1 — semantic remove-intent triage is Stage 10,
Phase 2. The mailto alias is the deterministic Phase-1 path.) Now attempt to send again:
```bash
# author a draft to tainguyenvdcc@ and try to send
<bridge> tool gmail --client-dir "$CDIR" send --draft <that draft> --dry-run
```
**Assert:** blocked with `"blocker": "suppressed"` — the send is refused.

## Step 6 — Bounce path (no live forced bounce)
Do **not** force a live bounce. The DSN classifier is covered by
`test_phase1.py::TestGmailPresendAndClassifier::test_classifier_dsn_before_thread` (a real DSN
threaded into a sent message is classified as a hard bounce, not a reply). If you want a live
bounce, send to a non-existent mailbox **on a domain you control** and confirm the returned DSN,
after `sync`, suppresses the lead and logs an `email_bounce` activity.

---

# Phase 2 — the stages on top of the core loop

## Step 7 — Create a campaign with a goal + segment (Stage 5)
```bash
<bridge> tool crm-store --client-dir "$CDIR" campaign create --json \
  '{"slug":"proposal-q3","goal_type":"book_call","audience":{"segment":"active-listers"},"sendboxes":["sb-a"],
    "sequence":[{"step":1,"gap_days":0},{"step":2,"gap_days":4},{"step":3,"gap_days":6}]}'
<bridge> tool crm-store --client-dir "$CDIR" segment set --json '{"slug":"active-listers","rules":[{"field":"channels.email.status","op":"=","value":"usable"}]}'
<bridge> tool crm-store --client-dir "$CDIR" campaign queue --campaign proposal-q3 --segment active-listers
```
**Assert:** the campaign is created with a **validated** `goal_type` (an unknown goal_type is
rejected). Re-queuing does not double-add a contact already in the campaign, one recently touched
by another campaign (within `min_days_between_touches_across_campaigns`), or one in an active/frozen
sequence; an `email_first` campaign skips a no-email contact.

## Step 8 — Enrich the batch, evidence-only (Stage 4)
```bash
<bridge> tool crm-store --client-dir "$CDIR" enrich due --campaign proposal-q3
<bridge> tool crm-store --client-dir "$CDIR" enrich write --contact "$LEAD" --json \
  '{"identity":{"still_active":"confirmed"},"hooks":[{"type":"new_listing","summary":"New 3BR listing on Oak St",
    "evidence_url":"https://www.zillow.com/homedetails/…","observed_date":"2026-07-14","confidence":0.9,
    "analysis":{"sensitivity":"public_business"}}],"writing_brief":{"personalization_confidence":0.85}}'
```
**Assert:** a hook with **no `evidence_url` is rejected**; a `personal`-sensitivity hook lands in
`do_not_mention`, not in the usable angles; the dossier is stored with a TTL. Cross-campaign
inheritance: enrich the same contact under a second campaign and confirm it **reuses** the dossier
(no re-enrich) while the TTL is fresh.

## Step 9 — Draft → Approval Report → chat-approve (Stage 6 + 2D)
```bash
<bridge> tool crm-store --client-dir "$CDIR" draft write --contact "$LEAD" --campaign proposal-q3 --json \
  '{"step":1,"subject":"Quick idea on your Oak St listing","body_text":"Hi …","hooks_used":[{"type":"new_listing","evidence_url":"https://www.zillow.com/homedetails/…"}]}'
<bridge> tool crm-store --client-dir "$CDIR" approval-report      # renders the operator HTML + approval_index.json
<bridge> tool crm-store --client-dir "$CDIR" approve --json '{"approve":"1","reject":[{"n":2,"reason":"too generic"}]}'
```
**Assert:** `draft write` **rejects a `hooks_used` entry that is not an evidenced dossier hook**, and
a step-1 subject starting `Re:` is rejected; the Approval Report groups **High confidence** vs
**Review carefully** and numbers each draft; `approve 1` flips draft #1 to `status:"approved"` and
**moves it to `outbox/approved/`**, while `reject 2` writes a line to `analytics/learning_log.md`.
Only an approved draft is now sendable (Step 3's `draft_not_approved` gate).

## Step 10 — Send the approved batch, then a follow-up bump via the injected clock (Stage 8 + 10)
Send the approved step-1 draft to inbox #2 (real send, real timestamp — **no test mode**):
```bash
<bridge> tool gmail --client-dir "$CDIR" send --draft "$CDIR/campaigns/proposal-q3/outbox/approved/"*.json
```
Now simulate "several days later" so the silent-lead bump comes due. This only shifts what the tool
*reads as now* — the real send already happened at real time:
```bash
export OUTREACHCRM_TEST_MODE=1
export OUTREACHCRM_FAKE_NOW=$(python3 -c "import datetime;print((datetime.date.today()+datetime.timedelta(days=5)).isoformat())")
<bridge> tool crm-store --client-dir "$CDIR" followups due --campaign proposal-q3
```
**Assert:** with **no reply**, after the step-2 `gap_days` has elapsed the lead appears in
`followups due` at `next_step: 2`. If that lead had replied (Step 4 froze its sequence), it does
**not** appear — a reply freezes the bump. Unset the clock when done: `unset OUTREACHCRM_FAKE_NOW OUTREACHCRM_TEST_MODE` (or keep it for Step 11).

## Step 11 — Weekly client report + scrub verification (2E)
```bash
export OUTREACHCRM_TEST_MODE=1 OUTREACHCRM_FAKE_NOW=$(date +%F)   # a stable "now" for the 7-day window
<bridge> tool crm-store --client-dir "$CDIR" weekly-report --client-name "Max Output"
```
**Assert:** the report renders to `outputs/<date>/max-output-weekly-client-report.html`, `blocked`
is `false`, and the `.md` source shows the delivered/replies/pipeline figures. Open the HTML and
confirm it contains **no internal terms** (no `sendbox`, `crm_store`, `WideCast`, `OutreachCRM`,
campaign slug). Then force the scrub gate: give a deal's contact a name containing a blind term and
re-run — the render returns `blocked:true` with the offending term, writes only a
`*.blocked.html` sidecar, and **does not** produce the real HTML. Unset the clock afterwards.

## Step 12 — Operator notification (2E, dry-run — no live provider needed)
```bash
# no provider config → a valid degraded outcome, never a run failure:
<bridge> tool provider notify --message "Daily run: 2 sent, 1 reply" \
  --log "$CDIR/notifications/notification_log.md"
# with a configured+enabled provider, dry-run composes the plan without touching the network:
<bridge> tool provider --config "$CDIR/integrations/providers/provider_config.local.json" \
  notify --message "Weekly report ready" --event weekly_client_report_ready \
  --report-file "$CDIR/outputs/$(date +%F)/max-output-weekly-client-report.html" \
  --log "$CDIR/notifications/notification_log.md" --dry-run
```
**Assert:** the no-config call exits `0` with `status:"local_path_only"` (the run surfaces report
links in chat instead of failing); the configured `--dry-run` call exits `0` with `status:"dry_run"`,
makes **no** network request, and appends a 16-column row to `notification_log.md`. A real send
(omit `--dry-run`) requires a live per-client WideCast key and is out of scope for this offline gate.

## Scheduled-run note (run_lock)
The full unattended run is the 30-step Daily Run Algorithm in `playbooks/AUTOMATION_SCHEDULING.md`;
the steps above exercise its consequential parts in isolation. A real run first takes the per-client
`run_lock` (`outputs/YYYY-MM/YYYY-MM-DD/{client}-run_lock.json`) — a fresh lock (< ~3h) blocks a
duplicate run, and the lock is released at the Stage-9 completion gate. OutreachCRM ships no runner
binary; a native AI scheduled task (or launchd) invokes the agent with the scheduler prompt.

## Assertion checklist (~25)
Phase 1 core:
- [ ] auth: smtp ok, imap ok; credentials.json is chmod 600
- [ ] import: 3 real inboxes as contacts; phones normalized to E.164; dedupe on repeat rows
- [ ] dry-run send shows the gate result + List-Unsubscribe + reserved token
- [ ] real send: email received; sent_log row with rfc_message_id; email_sent activity; draft→sent
- [ ] quota: `tool gmail quota --sendbox sb-a` shows sent/remaining correctly
- [ ] reply sync: campaign_reply counted; contact sequence_state=frozen
- [ ] apply-rules reply_positive: deal@new_reply + task; idempotent on re-run
- [ ] unsubscribe reply → suppressed:true; subsequent send blocked with `suppressed`
- [ ] step-1 subject starting `Re:` is blocked (`step1_subject_looks_like_reply`)
- [ ] a draft not `status:"approved"` is blocked (`draft_not_approved`)
- [ ] classifier DSN-before-thread unit test passes
- [ ] no CRM file was written by hand — every mutation went through `tool crm-store`

Phase 2 stages:
- [ ] campaign create validates `goal_type`; an unknown goal_type is rejected
- [ ] queue guards: no double-add across campaigns / active sequence / recent-touch; email_first skips no-email
- [ ] enrich rejects a hook with no `evidence_url`; personal hook → `do_not_mention`
- [ ] cross-campaign dossier inheritance: same contact reused, not re-enriched, while TTL fresh
- [ ] draft write rejects a non-evidenced `hooks_used`; step-1 subject `Re:` rejected
- [ ] Approval Report groups High/Review carefully and numbers drafts
- [ ] approve → status:approved + moved to outbox/approved/; reject → learning_log line
- [ ] injected clock: silent lead becomes due at next_step after gap_days; a reply keeps it out
- [ ] weekly report renders client-facing HTML; blocked:false; no internal terms present
- [ ] weekly report scrub gate: blind term → blocked:true + .blocked.html sidecar, no real HTML
- [ ] notify no-config → local_path_only (exit 0, run not failed)
- [ ] notify --dry-run → dry_run, no network, 16-column notification_log.md row appended
- [ ] OUTREACHCRM_TEST_MODE was unset before any live run

Open/click tracking is Phase 3 (needs the Cloudflare worker); opens are not asserted here.
