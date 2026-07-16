# OutreachCRM — Phase 1 E2E runbook (real send, one sendbox)

This is the manual acceptance test for the Phase-1 core loop. Unlike `test_phase1.py`
(offline, 27 cases), this exercises a **real Gmail send + real reply sync** with the three
Max Output test inboxes. Run it once after any change to `gmail_client.py` or the send path.

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

## Step 0 — Reset (idempotent; safe to re-run the whole runbook)
```bash
rm -rf "$PIPE"
python3 tools/crm_store.py --pipeline "$PIPE" --client max-output --business ai-automation --location hcmc init-client
export CDIR=$(python3 -c "import glob;print(glob.glob('$PIPE/clients/max-output/*')[0])")
echo "CDIR=$CDIR"
```
(For a client that already exists, the sanctioned reset is
`python3 tools/crm_store.py --client-dir "$CDIR" reset-client --confirm`, which also wipes
`test_fixture`-tagged suppression so `tainguyenvdcc@` is not permanently blocked between runs.)

## Step 1 — Connect the sendbox
```bash
python3 tools/gmail_client.py --client-dir "$CDIR" auth --sendbox sb-a --email "$SENDER"
python3 tools/gmail_client.py --client-dir "$CDIR" health --sendbox sb-a   # expect smtp/imap: ok
```
**Assert:** `sendboxes/sb-a/credentials.json` exists and is `-rw-------` (chmod 600);
`sendboxes/sendboxes.json` lists `sb-a` status `healthy`.

## Step 2 — Import the fixture list
```bash
python3 tools/import_leads.py inspect --file tests/fixtures/max_output_list.csv
python3 tools/import_leads.py import --client-dir "$CDIR" --file tests/fixtures/max_output_list.csv --list-slug max-output
```
**Assert:** created 3 (the three real inboxes), skipped/normalized as expected; `tainguyenvdcc@`
is imported now (it becomes suppressed only in Step 5).

## Step 3 — Approve + send a step-1 email to inbox #1
Build a draft (normally Stage 6 writes this; here we author it by hand), approve, send:
```bash
LEAD=$(python3 tools/crm_store.py --client-dir "$CDIR" contact list --where identities.emails,contains,x 2>/dev/null; \
  python3 -c "import sys;sys.path.insert(0,'tools');from crm_store import CrmStore;s=CrmStore('$CDIR');\
import json;print([c['id'] for c in s.a.query('contacts') if any(e['address']=='huubinhnguyen81@gmail.com' for e in c['identities']['emails'])][0])")
mkdir -p "$CDIR/campaigns/demo/outbox/approved"
cat > "$CDIR/campaigns/demo/outbox/approved/d1.json" <<JSON
{"id":"draft_e2e_1","schema_version":1,"lead_id":"$LEAD","campaign_slug":"demo","step":1,
 "sendbox":"sb-a","to":"huubinhnguyen81@gmail.com","subject":"Quick question about your listings",
 "body_text":"Hi — testing OutreachCRM Phase 1. Please reply 'yes' to this email.\n\n— E2E",
 "confidence_band":"high","tracking":"plain_text","status":"approved","guessed_approved":false}
JSON
python3 tools/gmail_client.py --client-dir "$CDIR" send --draft "$CDIR/campaigns/demo/outbox/approved/d1.json" --dry-run   # inspect gates first
python3 tools/gmail_client.py --client-dir "$CDIR" send --draft "$CDIR/campaigns/demo/outbox/approved/d1.json"            # real send
```
**Assert:** the email arrives at `huubinhnguyen81@gmail.com`; `sent_log.jsonl` has one row with a
real `rfc_message_id`; an `email_sent` activity exists on the contact; the draft file is now
`status:"sent"`. The message has a `List-Unsubscribe: <mailto:...+unsub-...>` header.

## Step 4 — Reply positive, then sync
From `huubinhnguyen81@gmail.com`, **reply** to that email (keep the `Re:` subject). Then:
```bash
python3 tools/gmail_client.py --client-dir "$CDIR" sync --sendbox sb-a
```
**Assert:** the sync output shows `campaign_reply: 1` and a `replies_untriaged` entry for the
lead; the contact's `sequence_state` is now `frozen` (any reply freezes the sequence).
Run the rule:
```bash
python3 tools/crm_store.py --client-dir "$CDIR" apply-rules --event reply_positive --contact "$LEAD" --activity <activity_seq_from_sync>
```
**Assert:** a deal at `new_reply` exists; a "Reply within 4h" task is open. Re-running the same
`apply-rules` is a no-op (idempotent).

## Step 5 — Unsubscribe path (inbox #3)
Send a step-1 email to `tainguyenvdcc@gmail.com` (repeat Step 3 with that address), then from that
inbox **reply with the word "unsubscribe"** (or click the mailto List-Unsubscribe). Sync:
```bash
python3 tools/gmail_client.py --client-dir "$CDIR" sync --sendbox sb-a
python3 tools/crm_store.py --client-dir "$CDIR" suppress check --email tainguyenvdcc@gmail.com
```
**Assert:** `suppressed: true`. Now attempt to send again to that address:
```bash
# author a draft to tainguyenvdcc@ and try to send
python3 tools/gmail_client.py --client-dir "$CDIR" send --draft <that draft> --dry-run
```
**Assert:** blocked with `"blocker": "suppressed"` — the send is refused.

## Step 6 — Bounce path (no live forced bounce)
Do **not** force a live bounce. The DSN classifier is covered by
`test_phase1.py::TestGmailPresendAndClassifier::test_classifier_dsn_before_thread` (a real DSN
threaded into a sent message is classified as a hard bounce, not a reply). If you want a live
bounce, send to a non-existent mailbox **on a domain you control** and confirm the returned DSN,
after `sync`, suppresses the lead and logs an `email_bounce` activity.

## Assertion checklist (~15)
- [ ] auth: smtp ok, imap ok; credentials.json is chmod 600
- [ ] import: 3 real inboxes as contacts; phones normalized to E.164; dedupe on repeat rows
- [ ] dry-run send shows the gate result + List-Unsubscribe + reserved token
- [ ] real send: email received; sent_log row with rfc_message_id; email_sent activity; draft→sent
- [ ] quota: `gmail_client.py quota --sendbox sb-a` shows sent/remaining correctly
- [ ] reply sync: campaign_reply counted; contact sequence_state=frozen
- [ ] apply-rules reply_positive: deal@new_reply + task; idempotent on re-run
- [ ] unsubscribe reply → suppressed:true; subsequent send blocked with `suppressed`
- [ ] step-1 subject starting `Re:` is blocked (`step1_subject_looks_like_reply`)
- [ ] a draft not `status:"approved"` is blocked (`draft_not_approved`)
- [ ] classifier DSN-before-thread unit test passes
- [ ] no CRM file was written by hand — every mutation went through `crm_store.py`

Open/click tracking is Phase 2 (needs the Cloudflare worker); opens are not asserted here.
